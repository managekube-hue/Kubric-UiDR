"""
KAI-HOUSE — SOC dashboard insights and tenant health monitoring.

Subscribes to:  kubric.{tenant}.agent.status.*
Publishes to:   kubric.kai.house.summary

Agent persona: SOC Operations Manager
CrewAI integration: make_house_crew() → Crew.kickoff() → JSON health overview
"""

from __future__ import annotations

import time
from typing import Any

import orjson
import structlog

from kai.core.crew import make_house_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class HouseAgent:
    """KAI-HOUSE — aggregates agent health, alert volume, and tenant status."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)

        crew = make_house_crew(tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        summary: dict[str, Any] = {
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "agent_health": result.get("agent_health", {}),
            "alert_24h": result.get("alert_24h", 0),
            "open_tickets": result.get("open_tickets", 0),
            "kiss_score": result.get("kiss_score", 0.0),
            "top_issues": result.get("top_issues", []),
        }

        await nats_client.publish(
            "kubric.kai.house.summary",
            orjson.dumps(summary),
        )

        log.info("house.summary_published", tenant_id=tenant_id)
        return summary


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
