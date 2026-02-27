"""
NATS handler entry point for the Invest persona.

Import path: kai.invest.handler
Callable:    handle(payload: dict) -> dict

Expected payload keys:
  tenant_id          str  – Kubric tenant identifier
  trigger_subject    str  – original NATS trigger subject (optional)
  investigation_id   str  – upstream analyst investigation identifier
  attack_chain       list – MITRE attack chain from the analyst report
  iocs               list – indicators of compromise
  ...                full analyst investigation report
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.invest import InvestAgent

log = logging.getLogger(__name__)

_agent: InvestAgent | None = None


def _get_agent() -> InvestAgent:
    global _agent
    if _agent is None:
        _agent = InvestAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """
    Dispatch a forensic investigation trigger payload through the InvestAgent.

    InvestAgent.handle() manages chain-of-custody for evidence, determines
    root cause, and evaluates regulatory implications via CrewAI.

    Returns a JSON-serialisable dict with status and the investigation case report.
    """
    try:
        agent = _get_agent()
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.kai.analyst.report",
        )
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "invest", "result": result}
    except Exception as exc:
        log.error("invest handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "invest", "error": str(exc)}
