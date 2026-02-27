"""
KAI-BILL — Usage metering, billing reconciliation, and invoice generation.

Subscribes to:  kubric.{tenant}.billing.*
Publishes to:   kubric.kai.bill.reconciled

Agent persona: Billing Reconciliation Specialist
CrewAI integration: make_bill_crew() → Crew.kickoff() → JSON billing summary
"""

from __future__ import annotations

import time
from typing import Any

import orjson
import structlog

from kai.core.crew import make_bill_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class BillAgent:
    """KAI-BILL — reconciles usage data, validates invoices, flags anomalies."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id = _extract_tenant(subject)
        event_json = orjson.dumps(event).decode()

        crew = make_bill_crew(event_json, tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        billing_result: dict[str, Any] = {
            "tenant_id": tenant_id,
            "timestamp": int(time.time() * 1000),
            "period": result.get("period", ""),
            "total_agents": result.get("total_agents", 0),
            "total_events": result.get("total_events", 0),
            "amount_usd": result.get("amount_usd", 0.0),
            "anomaly_detected": result.get("anomaly_detected", False),
            "merkle_root": result.get("merkle_root", ""),
        }

        await nats_client.publish(
            f"kubric.{tenant_id}.kai.bill.reconciled.v1",
            orjson.dumps(billing_result),
        )

        log.info("bill.reconciled", tenant_id=tenant_id, amount=billing_result["amount_usd"])
        return billing_result


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
