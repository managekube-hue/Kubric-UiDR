"""
KAI-HUNTER — Proactive threat hunting across telemetry data.

Subscribes to:  kubric.kai.foresight.risk (triggered by elevated risk scores)
Publishes to:   kubric.kai.hunter.findings

Agent persona: Threat Hunter
CrewAI integration: make_hunter_crew() → Crew.kickoff() → JSON hunting findings
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_hunter_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class HunterAgent:
    """KAI-HUNTER — runs hypothesis-driven threat hunts across telemetry."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        hunt_id = str(uuid.uuid4())

        event_json = orjson.dumps(event).decode()

        crew = make_hunter_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        findings: dict[str, Any] = {
            "hunt_id": hunt_id,
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "hypothesis": result.get("hypothesis", ""),
            "data_sources_queried": result.get("data_sources_queried", []),
            "findings": result.get("findings", []),
            "iocs_discovered": result.get("iocs_discovered", []),
            "mitre_techniques": result.get("mitre_techniques", []),
            "severity": result.get("severity", "MEDIUM"),
            "recommendation": result.get("recommendation", ""),
        }

        await nats_client.publish(
            "kubric.kai.hunter.findings",
            orjson.dumps(findings),
        )

        log.info("hunter.findings_published", hunt_id=hunt_id, tenant_id=tenant_id)
        return findings


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
