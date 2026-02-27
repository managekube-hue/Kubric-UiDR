"""
NATS handler entry point for the Triage persona.

Import path: kai.triage.handler
Callable:    handle(payload: dict) -> dict

The NATS consumer calls handle(payload) where payload is the decoded
JetStream message body.  Key fields expected:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
  ...             any additional alert fields forwarded from EDR/NDR/ITDR
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.triage import TriageAgent

log = logging.getLogger(__name__)

_agent: TriageAgent | None = None


def _get_agent() -> TriageAgent:
    global _agent
    if _agent is None:
        _agent = TriageAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """
    Dispatch a triage trigger payload through the TriageAgent.

    Extracts the NATS subject from payload (or constructs one from tenant_id)
    and delegates to TriageAgent.handle(subject=..., event=...).

    Returns a JSON-serialisable dict with status and the triage result.
    """
    try:
        agent = _get_agent()
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.detection.alert",
        )
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "triage", "result": result}
    except Exception as exc:
        log.error("triage handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "triage", "error": str(exc)}
