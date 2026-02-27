"""
NATS handler entry point for the Sentinel persona.

Import path: kai.sentinel.handler
Callable:    handle(payload: dict) -> dict

Sentinel computes the KiSS (Kubric integrated Security Score) — a 0-100
composite from vuln, compliance, detection, and response sub-scores.

Expected payload keys:
  tenant_id   str  – Kubric tenant identifier (required)
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.sentinel import SentinelAgent

log = logging.getLogger(__name__)

_agent: SentinelAgent | None = None


def _get_agent() -> SentinelAgent:
    global _agent
    if _agent is None:
        _agent = SentinelAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Compute and publish the KiSS health score for the triggering tenant."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        agent = _get_agent()
        result = await agent.compute(tenant_id)
        return {"status": "ok", "persona": "sentinel", "result": result}
    except Exception as exc:
        log.error("sentinel handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "sentinel", "error": str(exc)}
