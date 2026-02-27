"""
NATS handler entry point for the Keeper persona.

Import path: kai.keeper.handler
Callable:    handle(payload: dict) -> dict

Expected payload keys:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
  id / event_id   str  – vulnerability or drift finding identifier
  severity        str  – finding severity
  ...             full VDR vulnerability or KIC drift finding
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.keeper import KeeperAgent

log = logging.getLogger(__name__)

_agent: KeeperAgent | None = None


def _get_agent() -> KeeperAgent:
    global _agent
    if _agent is None:
        _agent = KeeperAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """
    Dispatch a remediation trigger payload through the KeeperAgent.

    KeeperAgent.handle() drafts an AI-powered remediation plan via CrewAI
    and optionally submits it to the Temporal RemediationWorkflow if
    KUBRIC_AUTO_REMEDIATE=true and the plan is flagged as auto-safe.

    Returns a JSON-serialisable dict with status and the remediation plan.
    """
    try:
        agent = _get_agent()
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.vdr.vuln.finding",
        )
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "keeper", "result": result}
    except Exception as exc:
        log.error("keeper handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "keeper", "error": str(exc)}
