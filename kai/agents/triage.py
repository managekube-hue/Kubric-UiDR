"""
KAI-TRIAGE — Alert enrichment and severity scoring.

Subscribes to:  kubric.{tenant}.edr.*, kubric.{tenant}.ndr.*, kubric.{tenant}.itdr.*
Publishes to:   kubric.{tenant}.kai.triage.enriched.v1

Agent persona: Senior SOC Analyst
CrewAI integration: make_triage_crew() → Crew.kickoff() → JSON output
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_triage_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)

_SEVERITY_ORDER = ["CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"]


class TriageAgent:
    """KAI-TRIAGE agent — stateless, one instance per message."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id  = _extract_tenant(subject)
        triage_id  = str(uuid.uuid4())
        event_json = orjson.dumps(event).decode()

        # ── CrewAI crew kickoff (Ollama LLM, cloud fallback) ──────────────────
        crew   = make_triage_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        triage_result: dict[str, Any] = {
            "triage_id":          triage_id,
            "tenant_id":          tenant_id,
            "source_subject":     subject,
            "source_event_id":    event.get("event_id", ""),
            "timestamp":          int(time.time() * 1000),
            "severity":           _coerce_severity(result.get("severity")),
            "mitre_techniques":   result.get("mitre_techniques", []),
            "summary":            result.get("summary", ""),
            "recommended_action": result.get("recommended_action", ""),
            "confidence":         float(result.get("confidence", 0.5)),
            "model_used":         "crewai/ollama/llama3.2",
        }

        # Publish enriched result back to NATS — subject includes tenant_id for isolation
        await nats_client.publish(
            f"kubric.{tenant_id}.kai.triage.enriched.v1",
            orjson.dumps(triage_result),
        )

        log.info(
            "triage.complete",
            triage_id=triage_id,
            tenant_id=tenant_id,
            severity=triage_result["severity"],
        )
        return triage_result


# ─── helpers ─────────────────────────────────────────────────────────────────

def _extract_tenant(subject: str) -> str:
    """Extract tenant_id from subject like kubric.{tenant_id}.edr.process.v1"""
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"


def _coerce_severity(raw: Any) -> str:
    if isinstance(raw, str) and raw.upper() in _SEVERITY_ORDER:
        return raw.upper()
    return "MEDIUM"
