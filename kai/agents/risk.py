"""
KAI-RISK — FAIR-based quantitative cyber risk assessment.

Subscribes to:  kubric.{tenant}.risk.assess
Publishes to:   kubric.kai.risk.assessment

Agent persona: Cyber Risk Quantification Analyst
CrewAI integration: make_risk_crew() → Crew.kickoff() → JSON risk assessment
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_risk_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class RiskAgent:
    """KAI-RISK — quantifies cyber risk using FAIR methodology."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        assessment_id = str(uuid.uuid4())

        event_json = orjson.dumps(event).decode()

        crew = make_risk_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        assessment: dict[str, Any] = {
            "assessment_id": assessment_id,
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "risk_scenario": result.get("risk_scenario", ""),
            "threat_event_frequency": result.get("threat_event_frequency", 0.0),
            "vulnerability_factor": result.get("vulnerability_factor", 0.0),
            "loss_magnitude_usd": result.get("loss_magnitude_usd", 0.0),
            "annual_loss_expectancy": result.get("annual_loss_expectancy", 0.0),
            "risk_rating": result.get("risk_rating", "MEDIUM"),
            "mitigations": result.get("mitigations", []),
            "residual_risk": result.get("residual_risk", 0.0),
        }

        await nats_client.publish(
            f"kubric.{tenant_id}.kai.risk.assessment.v1",
            orjson.dumps(assessment),
        )

        log.info("risk.assessment_published", assessment_id=assessment_id, tenant_id=tenant_id)
        return assessment


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
