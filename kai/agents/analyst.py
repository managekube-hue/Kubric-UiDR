"""
KAI-ANALYST — Deep-dive investigation and forensic analysis.

Subscribes to:  kubric.kai.triage.enriched (CRITICAL/HIGH only)
Publishes to:   kubric.kai.analyst.report

Agent persona: Senior Threat Analyst
CrewAI integration: make_analyst_crew() → Crew.kickoff() → JSON investigation report
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_analyst_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class AnalystAgent:
    """KAI-ANALYST — performs deep-dive forensic analysis on escalated alerts."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        severity = str(event.get("severity", "")).upper()

        if severity not in ("CRITICAL", "HIGH"):
            return {"skipped": True, "reason": f"severity {severity} below threshold"}

        investigation_id = str(uuid.uuid4())
        event_json = orjson.dumps(event).decode()

        crew = make_analyst_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        report: dict[str, Any] = {
            "investigation_id": investigation_id,
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "source_event_id": event.get("event_id", event.get("triage_id", "")),
            "severity": severity,
            "attack_chain": result.get("attack_chain", []),
            "iocs": result.get("iocs", []),
            "affected_assets": result.get("affected_assets", []),
            "timeline": result.get("timeline", []),
            "containment_actions": result.get("containment_actions", []),
            "confidence": float(result.get("confidence", 0.5)),
        }

        await nats_client.publish(
            f"kubric.{tenant_id}.kai.analyst.report.v1",
            orjson.dumps(report),
        )

        log.info("analyst.report_published", investigation_id=investigation_id, tenant_id=tenant_id)
        return report


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
