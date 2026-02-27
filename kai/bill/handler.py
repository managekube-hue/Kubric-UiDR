"""
NATS handler entry point for the Bill persona.

Import path: kai.bill.handler
Callable:    handle(payload: dict) -> dict

Bill reconciles usage metering data, validates Stripe invoices, and flags
billing anomalies. Publishes reconciliation results to kubric.kai.bill.reconciled.

Expected payload keys:
  tenant_id       str  – Kubric tenant identifier
  trigger_subject str  – original NATS trigger subject (optional)
  period          str  – billing period (e.g. "2025-01")
  total_agents    int  – number of active agents in period
  total_events    int  – total events processed in period
"""

from __future__ import annotations

import logging
from typing import Any

from kai.agents.bill import BillAgent

log = logging.getLogger(__name__)

_agent: BillAgent | None = None


def _get_agent() -> BillAgent:
    global _agent
    if _agent is None:
        _agent = BillAgent()
    return _agent


async def handle(payload: dict[str, Any]) -> dict[str, Any]:
    """Reconcile a billing event for the triggering tenant."""
    try:
        tenant_id: str = payload.get("tenant_id", "default")
        subject: str = payload.get(
            "trigger_subject",
            f"kubric.{tenant_id}.billing.usage",
        )
        agent = _get_agent()
        result = await agent.handle(subject=subject, event=payload)
        return {"status": "ok", "persona": "bill", "result": result}
    except Exception as exc:
        log.error("bill handler error: %s", exc, exc_info=True)
        return {"status": "error", "persona": "bill", "error": str(exc)}
