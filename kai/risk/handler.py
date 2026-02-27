"""
NATS handler entry point for the Risk persona.

Import path: kai.risk.handler
Callable:    handle(payload: dict) -> dict

Expected payload keys:
  tenant_id       str   – Kubric tenant identifier
  trigger_subject str   – original NATS trigger subject (optional)
  asset           str   – asset name or identifier being assessed
  threat_scenario str   – description of the threat scenario
  ...             optional FAIR parameter overrides (tef_*, vuln_*, plm_*, slef_*, slm_*)
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.risk import RiskAgent

log = logging.getLogger(__name__)

_agent: RiskAgent | None = None


def _get_agent() -> RiskAgent:
    global _agent
    if _agent is None:
        _agent = RiskAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """
    Dispatch a risk assessment trigger payload through the RiskAgent.

    RiskAgent.handle() runs a FAIR-methodology quantitative risk analysis
    via CrewAI and publishes the assessment to kubric.kai.risk.assessment.

    Returns a JSON-serialisable dict with status and the risk assessment.
    """
    try:
        agent = _get_agent()
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.risk.assess",
        )
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "risk", "result": result}
    except Exception as exc:
        log.error("risk handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "risk", "error": str(exc)}
