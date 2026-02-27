"""
NATS handler entry point for the Deploy persona.

Import path: kai.deploy.handler
Callable:    handle(payload: dict) -> dict

Deploy validates deployment requests for security posture compliance,
runs SAST/container/KAC checks via CrewAI, and either approves or blocks
the deployment. Publishes results to kubric.kai.deploy.validated.

Expected payload keys:
  tenant_id         str   – Kubric tenant identifier
  trigger_subject   str   – original NATS trigger subject (optional)
  deployment_type   str   – k8s | docker | vm | lambda
  target            str   – image, manifest path, or function name
  requester         str   – user or pipeline that triggered the deploy
  change_ticket_id  str   – linked ITSM change ticket (optional)
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.deploy import DeployAgent

log = logging.getLogger(__name__)

_agent: DeployAgent | None = None


def _get_agent() -> DeployAgent:
    global _agent
    if _agent is None:
        _agent = DeployAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Validate a deployment request against Kubric security policy."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.deploy.request",
        )
        agent = _get_agent()
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "deploy", "result": result}
    except Exception as exc:
        log.error("deploy handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "deploy", "error": str(exc)}
