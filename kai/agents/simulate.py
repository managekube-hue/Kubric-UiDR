"""
KAI-SIMULATE — Attack simulation and purple team exercise orchestration.

Subscribes to:  kubric.{tenant}.simulate.request
Publishes to:   kubric.kai.simulate.results

Agent persona: Purple Team Operator
CrewAI integration: make_simulate_crew() → Crew.kickoff() → JSON simulation report
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_simulate_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class SimulateAgent:
    """KAI-SIMULATE — plans and executes controlled attack simulations."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        simulation_id = str(uuid.uuid4())

        event_json = orjson.dumps(event).decode()

        crew = make_simulate_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        sim_result: dict[str, Any] = {
            "simulation_id": simulation_id,
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "attack_type": result.get("attack_type", ""),
            "mitre_techniques_tested": result.get("mitre_techniques_tested", []),
            "detections_triggered": result.get("detections_triggered", []),
            "detections_missed": result.get("detections_missed", []),
            "coverage_score": float(result.get("coverage_score", 0.0)),
            "gaps_identified": result.get("gaps_identified", []),
            "recommendations": result.get("recommendations", []),
        }

        await nats_client.publish(
            "kubric.kai.simulate.results",
            orjson.dumps(sim_result),
        )

        log.info("simulate.completed", simulation_id=simulation_id, tenant_id=tenant_id)
        return sim_result


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
