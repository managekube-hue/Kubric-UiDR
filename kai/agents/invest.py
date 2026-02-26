"""
KAI-INVEST — Digital forensics investigation and evidence chain management.

Subscribes to:  kubric.kai.analyst.report (triggered by ANALYST escalation)
Publishes to:   kubric.kai.invest.case

Agent persona: Digital Forensics Investigator
CrewAI integration: make_invest_crew() → Crew.kickoff() → JSON case report
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_invest_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class InvestAgent:
    """KAI-INVEST — manages forensic investigations with chain-of-custody."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        case_id = str(uuid.uuid4())

        event_json = orjson.dumps(event).decode()

        crew = make_invest_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        case_report: dict[str, Any] = {
            "case_id": case_id,
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "source_investigation_id": event.get("investigation_id", ""),
            "evidence_collected": result.get("evidence_collected", []),
            "chain_of_custody": result.get("chain_of_custody", []),
            "root_cause": result.get("root_cause", ""),
            "impact_assessment": result.get("impact_assessment", ""),
            "regulatory_implications": result.get("regulatory_implications", []),
            "remediation_verified": result.get("remediation_verified", False),
        }

        await nats_client.publish(
            "kubric.kai.invest.case",
            orjson.dumps(case_report),
        )

        log.info("invest.case_published", case_id=case_id, tenant_id=tenant_id)
        return case_report


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
