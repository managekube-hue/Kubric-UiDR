"""
NATS handler entry point for the Comm persona.

Import path: kai.comm.handler
Callable:    handle(payload: dict) -> dict

Comm routes enriched alerts to the correct channel: n8n ITSM webhook,
Vapi voice call (CRITICAL), Twilio SMS (HIGH+), and auto-creates Zammad
PSA tickets for CRITICAL severity and compliance failures.

Expected payload keys:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
  severity        str  – CRITICAL | HIGH | MEDIUM | LOW
  summary         str  – human-readable alert summary
  triage_id       str  – upstream triage identifier
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.comm import CommAgent

log = logging.getLogger(__name__)

_agent: CommAgent | None = None


def _get_agent() -> CommAgent:
    global _agent
    if _agent is None:
        _agent = CommAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Route an enriched alert to the appropriate notification channels."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.comm.alert.enriched",
        )
        agent = _get_agent()
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "comm", "result": result}
    except Exception as exc:
        log.error("comm handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "comm", "error": str(exc)}
