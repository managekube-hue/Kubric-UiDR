"""
NATS handler entry point for the House persona.

Import path: kai.house.handler
Callable:    handle(payload: dict) -> dict

House aggregates agent health, alert volumes, and open tickets into a SOC
operations summary and publishes it to kubric.kai.house.summary.

Expected payload keys:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.house import HouseAgent

log = logging.getLogger(__name__)

_agent: HouseAgent | None = None


def _get_agent() -> HouseAgent:
    global _agent
    if _agent is None:
        _agent = HouseAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Produce a SOC operations summary for the triggering tenant."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.agent.status.update",
        )
        agent = _get_agent()
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "house", "result": result}
    except Exception as exc:
        log.error("house handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "house", "error": str(exc)}
