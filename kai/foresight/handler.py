"""
NATS handler entry point for the Foresight persona.

Import path: kai.foresight.handler
Callable:    handle(payload: dict) -> dict

Foresight runs predictive risk modelling over the last 24 hours of telemetry
and publishes a per-tenant risk forecast to NATS.

Expected payload keys:
  tenant_id   str  – Kubric tenant identifier (required)
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.foresight import ForesightAgent

log = logging.getLogger(__name__)

_agent: ForesightAgent | None = None


def _get_agent() -> ForesightAgent:
    global _agent
    if _agent is None:
        _agent = ForesightAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Run a predictive risk forecast for the triggering tenant."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        agent = _get_agent()
        result = await agent.run_once(tenant_id)
        return {"status": "ok", "persona": "foresight", "result": result}
    except Exception as exc:
        log.error("foresight handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "foresight", "error": str(exc)}
