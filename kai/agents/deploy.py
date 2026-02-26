"""
KAI-DEPLOY — Secure deployment validation and change management.

Subscribes to:  kubric.{tenant}.deploy.request
Publishes to:   kubric.kai.deploy.validated

Agent persona: Deployment Security Engineer
CrewAI integration: make_deploy_crew() → Crew.kickoff() → JSON deployment validation
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_deploy_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class DeployAgent:
    """KAI-DEPLOY — validates deployments for security posture compliance."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        validation_id = str(uuid.uuid4())

        event_json = orjson.dumps(event).decode()

        crew = make_deploy_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        validation: dict[str, Any] = {
            "validation_id": validation_id,
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "deployment_type": result.get("deployment_type", ""),
            "target": result.get("target", ""),
            "security_checks_passed": result.get("security_checks_passed", []),
            "security_checks_failed": result.get("security_checks_failed", []),
            "approved": result.get("approved", False),
            "risk_level": result.get("risk_level", "MEDIUM"),
            "rollback_plan": result.get("rollback_plan", ""),
            "conditions": result.get("conditions", []),
        }

        await nats_client.publish(
            "kubric.kai.deploy.validated",
            orjson.dumps(validation),
        )

        log.info("deploy.validated", validation_id=validation_id, tenant_id=tenant_id, approved=validation["approved"])
        return validation


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
