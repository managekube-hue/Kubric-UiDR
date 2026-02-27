"""
NATS handler entry point for the Simulate persona.

Import path: kai.simulate.handler
Callable:    handle(payload: dict) -> dict

Simulate orchestrates controlled attack simulations (purple team exercises),
maps results to MITRE ATT&CK, scores detection coverage, and publishes
findings to kubric.kai.simulate.results.

Expected payload keys:
  tenant_id         str   – Kubric tenant identifier
  trigger_subject   str   – original NATS trigger subject (optional)
  attack_type       str   – phishing | lateral_movement | c2 | exfil | ransomware
  mitre_techniques  list  – list of ATT&CK technique IDs to simulate (optional)
  scope             str   – asset or subnet scope for simulation (optional)
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.simulate import SimulateAgent

log = logging.getLogger(__name__)

_agent: SimulateAgent | None = None


def _get_agent() -> SimulateAgent:
    global _agent
    if _agent is None:
        _agent = SimulateAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Run a controlled attack simulation for the triggering tenant."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.simulate.request",
        )
        agent = _get_agent()
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "simulate", "result": result}
    except Exception as exc:
        log.error("simulate handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "simulate", "error": str(exc)}
