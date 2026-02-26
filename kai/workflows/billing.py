"""
BillingWorkflow — Temporal durable workflow for metered usage invoicing.

Workflow: Aggregate ClickHouse usage events → Create Stripe invoice → Notify tenant

Activities:
  aggregate_usage      — sum billable events per tenant from ClickHouse
  create_stripe_invoice — call Stripe API to create and finalize invoice
  record_invoice       — write invoice record to PostgreSQL via K-SVC

This workflow is triggered:
  - Daily by a Temporal schedule (cron)
  - On-demand when a tenant's usage crosses a threshold

Temporal workflow_id = "billing-{tenant_id}-{billing_period}"
ensures idempotency across retries.
"""

from __future__ import annotations

import os
from datetime import timedelta
from typing import Any

import httpx
import structlog

log = structlog.get_logger(__name__)

_TEMPORAL_URL       = os.getenv("KUBRIC_TEMPORAL_URL", "temporal:7233")
_TASK_QUEUE         = "kubric-billing"
_STRIPE_SECRET_KEY  = os.getenv("KUBRIC_STRIPE_SECRET_KEY", "")
_CLICKHOUSE_HTTP    = os.getenv("KUBRIC_CLICKHOUSE_HTTP_URL", "http://127.0.0.1:8123")


# ─── Activities ──────────────────────────────────────────────────────────────

async def aggregate_usage(tenant_id: str, billing_period: str) -> dict[str, Any]:
    """
    Sum billable events for a tenant in the given billing period (YYYY-MM).
    Returns: {endpoint_hours, alert_count, scan_count, total_usd}
    """
    year, month = billing_period.split("-")
    sql = f"""
SELECT
    countIf(class_uid = 4007) AS process_events,
    countIf(class_uid = 4001) AS network_events,
    countIf(class_uid = 2002) AS vuln_findings
FROM kubric.ocsf_events
WHERE tenant_id = '{tenant_id}'
  AND toYYYYMM(timestamp) = {year}{month}
FORMAT JSONEachRow
""".strip()

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.post(
                _CLICKHOUSE_HTTP,
                content=sql.encode(),
                headers={"X-ClickHouse-Format": "JSONEachRow"},
            )
            if resp.status_code == 200:
                import orjson  # noqa: PLC0415
                lines = resp.text.strip().splitlines()
                rows = [orjson.loads(l) for l in lines if l]
                if rows:
                    row = rows[0]
                    process_events = int(row.get("process_events", 0))
                    network_events = int(row.get("network_events", 0))
                    vuln_findings  = int(row.get("vuln_findings", 0))
                    # Simple metered pricing tiers
                    total_usd = round(
                        (process_events / 10_000) * 5.0
                        + (network_events / 10_000) * 3.0
                        + (vuln_findings / 100) * 10.0,
                        2,
                    )
                    return {
                        "tenant_id":      tenant_id,
                        "billing_period": billing_period,
                        "process_events": process_events,
                        "network_events": network_events,
                        "vuln_findings":  vuln_findings,
                        "total_usd":      total_usd,
                    }
    except Exception as exc:
        log.error("billing.aggregate_failed", tenant_id=tenant_id, error=str(exc))

    return {"tenant_id": tenant_id, "billing_period": billing_period, "total_usd": 0.0}


async def create_stripe_invoice(
    tenant_id: str, stripe_customer_id: str, usage: dict[str, Any]
) -> dict[str, Any]:
    """Create and finalize a Stripe invoice for the usage."""
    if not _STRIPE_SECRET_KEY:
        log.warning("billing.stripe_key_missing")
        return {"status": "skipped", "invoice_id": None}

    headers = {"Authorization": f"Bearer {_STRIPE_SECRET_KEY}"}
    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            # Create invoice item
            await client.post(
                "https://api.stripe.com/v1/invoiceitems",
                headers=headers,
                data={
                    "customer": stripe_customer_id,
                    "amount": int(usage.get("total_usd", 0) * 100),  # cents
                    "currency": "usd",
                    "description": (
                        f"Kubric security telemetry — {usage.get('billing_period')} "
                        f"({usage.get('process_events', 0):,} events)"
                    ),
                },
            )
            # Create and finalize invoice
            inv_resp = await client.post(
                "https://api.stripe.com/v1/invoices",
                headers=headers,
                data={"customer": stripe_customer_id, "auto_advance": "true"},
            )
            inv_resp.raise_for_status()
            invoice = inv_resp.json()
            invoice_id = invoice.get("id", "")

            # Finalize
            await client.post(
                f"https://api.stripe.com/v1/invoices/{invoice_id}/finalize",
                headers=headers,
            )
            log.info("billing.invoice_created", tenant_id=tenant_id, invoice_id=invoice_id)
            return {"status": "created", "invoice_id": invoice_id}
    except Exception as exc:
        log.error("billing.stripe_failed", tenant_id=tenant_id, error=str(exc))
        return {"status": "error", "invoice_id": None}


async def record_invoice(tenant_id: str, invoice: dict[str, Any], usage: dict[str, Any]) -> bool:
    """Write the invoice record to PostgreSQL via K-SVC."""
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.post(
                "http://localhost:8080/v1/invoices",
                json={
                    "tenant_id":      tenant_id,
                    "invoice_id":     invoice.get("invoice_id"),
                    "billing_period": usage.get("billing_period"),
                    "total_usd":      usage.get("total_usd"),
                    "status":         invoice.get("status"),
                },
            )
            return resp.status_code in (200, 201)
    except Exception as exc:
        log.error("billing.record_failed", tenant_id=tenant_id, error=str(exc))
        return False


# ─── Temporal workflow ────────────────────────────────────────────────────────

async def submit_billing(tenant_id: str, stripe_customer_id: str, billing_period: str) -> None:
    """
    Public API — submit a billing run.
    Tries Temporal; falls back to direct async execution.
    """
    try:
        from temporalio import workflow  # noqa: PLC0415
        from temporalio.client import Client  # noqa: PLC0415

        @workflow.defn
        class BillingWorkflow:
            @workflow.run
            async def run(self, tenant_id: str, customer_id: str, period: str) -> str:
                usage = await workflow.execute_activity(
                    aggregate_usage,
                    args=[tenant_id, period],
                    start_to_close_timeout=timedelta(minutes=5),
                )
                invoice = await workflow.execute_activity(
                    create_stripe_invoice,
                    args=[tenant_id, customer_id, usage],
                    start_to_close_timeout=timedelta(minutes=2),
                )
                await workflow.execute_activity(
                    record_invoice,
                    args=[tenant_id, invoice, usage],
                    start_to_close_timeout=timedelta(seconds=30),
                )
                return "complete"

        client = await Client.connect(_TEMPORAL_URL)
        await client.start_workflow(
            BillingWorkflow.run,
            args=[tenant_id, stripe_customer_id, billing_period],
            id=f"billing-{tenant_id}-{billing_period}",
            task_queue=_TASK_QUEUE,
        )
        log.info("billing.temporal_submitted", tenant_id=tenant_id, period=billing_period)
        return

    except ImportError:
        log.warning("billing.temporal_not_installed")
    except Exception as exc:
        log.warning("billing.temporal_fallback", error=str(exc))

    # Fallback — run directly
    usage   = await aggregate_usage(tenant_id, billing_period)
    invoice = await create_stripe_invoice(tenant_id, stripe_customer_id, usage)
    await record_invoice(tenant_id, invoice, usage)
    log.info("billing.direct_complete", tenant_id=tenant_id, period=billing_period)
