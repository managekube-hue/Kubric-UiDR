"""
NATS handler entry point for the Analyst persona.

Import path: kai.analyst.handler
Callable:    handle(payload: dict) -> dict

Expected payload keys:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
  severity        str  – alert severity (CRITICAL/HIGH required for analysis)
  triage_id       str  – upstream triage identifier
  ...             full triage enriched event
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.analyst import AnalystAgent

log = logging.getLogger(__name__)

_agent: AnalystAgent | None = None


def _get_agent() -> AnalystAgent:
    global _agent
    if _agent is None:
        _agent = AnalystAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """
    Dispatch an analyst trigger payload through the AnalystAgent.

    AnalystAgent.handle() only proceeds for CRITICAL/HIGH severity events;
    lower-severity events are returned with a 'skipped' indicator.

    Returns a JSON-serialisable dict with status and the investigation report.
    """
    try:
        agent = _get_agent()
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.kai.triage.enriched",
        )
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "analyst", "result": result}
    except Exception as exc:
        log.error("analyst handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "analyst", "error": str(exc)}
