"""
K-KAI Billing Clerk: Main Billing Agent
Calculates monthly tenant bills based on agent count, events processed,
storage used, and ML inference calls. Applies tiered pricing and HLE adjustments.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field, asdict
from datetime import datetime, date, timezone
from typing import Dict, List, Optional, Tuple
import asyncpg

logger = logging.getLogger(__name__)
PG_DSN = os.getenv("PG_DSN", "postgresql://kubric:kubric@localhost:5432/kubric")

# ── Pricing tiers ─────────────────────────────────────────────────
PRICE_PER_AGENT_MONTH     = 25.00   # USD per agent per month
PRICE_PER_1M_EVENTS       = 10.00   # USD per 1M events processed
PRICE_PER_100GB_STORAGE   = 8.00    # USD per 100GB storage
PRICE_PER_1K_ML_CALLS     = 5.00    # USD per 1K ML inference API calls
PLATFORM_FEE              = 99.00   # USD flat monthly platform fee

@dataclass
class UsageMetrics:
    tenant_id:      str
    billing_month:  str                # YYYY-MM
    agent_count:    int
    events_count:   int                # total events ingested
    storage_bytes:  int                # bytes stored
    ml_calls:       int                # ML inference API calls
    support_hours:  float = 0.0        # billable support hours

@dataclass
class BillLineItem:
    description:  str
    quantity:     float
    unit:         str
    unit_price:   float
    total:        float

@dataclass
class TenantBill:
    tenant_id:       str
    billing_month:   str
    line_items:      List[BillLineItem]
    subtotal:        float
    discount_pct:    float
    discount_amount: float
    tax_rate:        float
    tax_amount:      float
    total:           float
    currency:        str = "USD"
    generated_at:    str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

def _apply_volume_discount(subtotal: float, agent_count: int) -> float:
    if agent_count >= 100: return 0.20    # 20% volume discount
    if agent_count >= 50:  return 0.15
    if agent_count >= 20:  return 0.10
    if agent_count >= 10:  return 0.05
    return 0.00

async def load_usage(
    pool: asyncpg.Pool, tenant_id: str, billing_month: str
) -> UsageMetrics:
    month_start = datetime.strptime(billing_month + "-01", "%Y-%m-%d")
    month_end   = date(month_start.year + (1 if month_start.month == 12 else 0),
                       (month_start.month % 12) + 1, 1)

    async with pool.acquire() as conn:
        agents = await conn.fetchval(
            "SELECT COUNT(*) FROM kai_agents WHERE tenant_id=$1 AND created_at < $2",
            tenant_id, month_end) or 0
        events = await conn.fetchval(
            """SELECT COUNT(*) FROM kai_alerts
               WHERE tenant_id=$1 AND created_at >= $2 AND created_at < $3""",
            tenant_id, month_start, month_end) or 0
        storage = await conn.fetchval(
            "SELECT COALESCE(SUM(size_bytes), 0) FROM kai_storage_ledger WHERE tenant_id=$1",
            tenant_id) or 0
        ml_calls = await conn.fetchval(
            """SELECT COUNT(*) FROM kai_ml_inference_log
               WHERE tenant_id=$1 AND created_at >= $2 AND created_at < $3""",
            tenant_id, month_start, month_end) or 0

    return UsageMetrics(
        tenant_id=tenant_id,
        billing_month=billing_month,
        agent_count=agents,
        events_count=events,
        storage_bytes=storage,
        ml_calls=ml_calls,
    )

def calculate_bill(metrics: UsageMetrics, tax_rate: float = 0.0) -> TenantBill:
    items: List[BillLineItem] = []

    # Platform fee
    items.append(BillLineItem("Platform fee", 1, "month", PLATFORM_FEE, PLATFORM_FEE))

    # Agent seats
    agent_total = metrics.agent_count * PRICE_PER_AGENT_MONTH
    items.append(BillLineItem(
        "Agent seats", metrics.agent_count, "agents/month",
        PRICE_PER_AGENT_MONTH, round(agent_total, 2)))

    # Events
    events_m  = metrics.events_count / 1_000_000
    ev_total  = events_m * PRICE_PER_1M_EVENTS
    items.append(BillLineItem(
        "Events processed", round(events_m, 4), "million events",
        PRICE_PER_1M_EVENTS, round(ev_total, 2)))

    # Storage
    storage_100gb = metrics.storage_bytes / (100 * 1024**3)
    st_total      = storage_100gb * PRICE_PER_100GB_STORAGE
    items.append(BillLineItem(
        "Storage", round(storage_100gb, 4), "x 100 GB",
        PRICE_PER_100GB_STORAGE, round(st_total, 2)))

    # ML calls
    ml_k      = metrics.ml_calls / 1_000
    ml_total  = ml_k * PRICE_PER_1K_ML_CALLS
    items.append(BillLineItem(
        "ML inference calls", round(ml_k, 4), "x 1K calls",
        PRICE_PER_1K_ML_CALLS, round(ml_total, 2)))

    subtotal        = round(sum(i.total for i in items), 2)
    disc_pct        = _apply_volume_discount(subtotal, metrics.agent_count)
    disc_amount     = round(subtotal * disc_pct, 2)
    discounted      = subtotal - disc_amount
    tax_amount      = round(discounted * tax_rate, 2)
    total           = round(discounted + tax_amount, 2)

    return TenantBill(
        tenant_id=metrics.tenant_id,
        billing_month=metrics.billing_month,
        line_items=items,
        subtotal=subtotal,
        discount_pct=disc_pct * 100,
        discount_amount=disc_amount,
        tax_rate=tax_rate * 100,
        tax_amount=tax_amount,
        total=total,
    )

async def generate_bill(tenant_id: str, billing_month: str) -> TenantBill:
    pool = await asyncpg.create_pool(PG_DSN, min_size=2, max_size=4)
    try:
        metrics = await load_usage(pool, tenant_id, billing_month)
        return calculate_bill(metrics)
    finally:
        await pool.close()

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    # Demo without DB
    demo_metrics = UsageMetrics(
        tenant_id="demo-tenant", billing_month="2025-01",
        agent_count=25, events_count=5_000_000,
        storage_bytes=50 * 1024**3, ml_calls=20_000,
    )
    bill = calculate_bill(demo_metrics, tax_rate=0.08)
    d    = asdict(bill)
    d["line_items"] = [asdict(li) for li in bill.line_items]
    print(json.dumps(d, indent=2))
