"""
NATS handler entry point for the Hunter persona.

Import path: kai.hunter.handler
Callable:    handle(payload: dict) -> dict

Expected payload keys:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
  risk_score      float – foresight risk score that triggered the hunt
  ...             context fields forwarded from the foresight risk event
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.hunter import HunterAgent

log = logging.getLogger(__name__)

_agent: HunterAgent | None = None


def _get_agent() -> HunterAgent:
    global _agent
    if _agent is None:
        _agent = HunterAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """
    Dispatch a threat-hunt trigger payload through the HunterAgent.

    HunterAgent.handle() runs a hypothesis-driven hunt using CrewAI
    and publishes findings to kubric.kai.hunter.findings.

    Returns a JSON-serialisable dict with status and the hunting findings.
    """
    try:
        agent = _get_agent()
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.kai.foresight.risk",
        )
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "hunter", "result": result}
    except Exception as exc:
        log.error("hunter handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "hunter", "error": str(exc)}
