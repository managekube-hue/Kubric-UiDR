"""
K-KAI Billing: HLE (Headline Licence Equivalent) Calculator
Converts usage-based billing metrics into seat-equivalent HLE values
for enterprise licence reconciliation and MSP partner invoicing.
"""
from __future__ import annotations
import json, logging
from dataclasses import dataclass
from typing import Dict, List

logger = logging.getLogger(__name__)

# Conversion constants (one HLE = fraction of annual enterprise contract)
EVENTS_PER_HLE_YEAR    = 50_000_000    # 50M events ~ 1 licence year
AGENTS_PER_HLE         = 10            # 10 deployed agents ~ 1 HLE
ML_CALLS_PER_HLE_YEAR  = 100_000       # 100K ML calls ~ 1 HLE year
HLE_MONTHLY_LIST_PRICE = 1_500.00      # USD per HLE per month (list price)

@dataclass
class HleBreakdown:
    tenant_id:          str
    billing_month:      str
    agent_hle:          float
    events_hle:         float
    ml_calls_hle:       float
    total_hle:          float
    list_price_total:   float
    currency:           str = "USD"

def calculate_hle(
    tenant_id:     str,
    billing_month: str,
    agent_count:   int,
    events_count:  int,
    ml_calls:      int,
) -> HleBreakdown:
    agent_hle    = agent_count / AGENTS_PER_HLE
    events_hle   = (events_count / EVENTS_PER_HLE_YEAR) * 12   # annualise -> monthly
    ml_calls_hle = (ml_calls / ML_CALLS_PER_HLE_YEAR) * 12

    total_hle        = round(agent_hle + events_hle + ml_calls_hle, 4)
    list_price_total = round(total_hle * HLE_MONTHLY_LIST_PRICE, 2)

    return HleBreakdown(
        tenant_id=tenant_id,
        billing_month=billing_month,
        agent_hle=round(agent_hle, 4),
        events_hle=round(events_hle, 4),
        ml_calls_hle=round(ml_calls_hle, 4),
        total_hle=total_hle,
        list_price_total=list_price_total,
    )

def batch_hle(records: List[Dict]) -> List[HleBreakdown]:
    """Process multiple tenant usage dicts and return HLE breakdowns."""
    return [
        calculate_hle(
            tenant_id    = r["tenant_id"],
            billing_month= r["billing_month"],
            agent_count  = r.get("agent_count", 0),
            events_count = r.get("events_count", 0),
            ml_calls     = r.get("ml_calls", 0),
        ) for r in records
    ]

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    demo = [
        {"tenant_id": "tenant-A", "billing_month": "2025-01",
         "agent_count": 30, "events_count": 8_000_000, "ml_calls": 15_000},
        {"tenant_id": "tenant-B", "billing_month": "2025-01",
         "agent_count": 5, "events_count": 500_000, "ml_calls": 2_000},
    ]
    for hle in batch_hle(demo):
        print(json.dumps(vars(hle), indent=2))
