"""
K-KAI Simulate: Dynamic Pricing Engine
Computes optimal per-tenant pricing adjustments based on usage intensity,
churn risk, and competitive benchmarks using a rule-based + elasticity model.
"""
from __future__ import annotations
import json, logging
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)

# Base list prices (USD/month with standard discount)
BASE_PLATFORM_FEE       = 99.00
BASE_AGENT_PRICE        = 25.00
BASE_EVENT_PRICE_1M     = 10.00
BASE_ML_CALL_PRICE_1K   = 5.00

# Elasticity coefficients (reduction in price sensitivity per unit increase in value metric)
USAGE_ELASTICITY        = -0.02   # price_adj% per % deviation from median usage

@dataclass
class PricingInput:
    tenant_id:          str
    agent_count:        int
    monthly_events:     int
    monthly_ml_calls:   int
    health_score:       float          # 0-100
    churn_risk:         float          # 0-1
    contract_months:    int            # months remaining in contract
    current_mrr:        float          # current spend
    median_peer_mrr:    float          # cohort median MRR

@dataclass
class PricingRecommendation:
    tenant_id:          str
    current_mrr:        float
    recommended_mrr:    float
    adjustment_pct:     float          # positive = increase, negative = discount
    rationale:          str
    platform_fee:       float
    agent_price:        float
    event_price_1m:     float
    ml_price_1k:        float
    valid_from:         str
    approved:           bool = False   # requires human approval before applying

def _volume_multiplier(agent_count: int) -> float:
    if agent_count >= 100: return 0.80
    if agent_count >= 50:  return 0.85
    if agent_count >= 20:  return 0.90
    if agent_count >= 10:  return 0.95
    return 1.00

def _churn_retention_adj(churn_risk: float) -> float:
    """Apply a retention discount to at-risk tenants. 0-10% discount."""
    if churn_risk >= 0.70: return -0.10
    if churn_risk >= 0.50: return -0.07
    if churn_risk >= 0.30: return -0.04
    return 0.00

def _health_premium(health_score: float) -> float:
    """High-health tenants are less price-sensitive; small premium."""
    if health_score >= 90: return 0.03
    if health_score >= 75: return 0.01
    return 0.00

def compute_pricing(inp: PricingInput) -> PricingRecommendation:
    vol_mult  = _volume_multiplier(inp.agent_count)
    churn_adj = _churn_retention_adj(inp.churn_risk)
    health_p  = _health_premium(inp.health_score)

    adj_platform_fee    = round(BASE_PLATFORM_FEE * vol_mult, 2)
    adj_agent_price     = round(BASE_AGENT_PRICE   * vol_mult, 2)
    adj_event_price     = round(BASE_EVENT_PRICE_1M * vol_mult, 2)
    adj_ml_price        = round(BASE_ML_CALL_PRICE_1K * vol_mult, 2)

    # Compute full recommended MRR
    rec_mrr = (
        adj_platform_fee +
        inp.agent_count    * adj_agent_price +
        (inp.monthly_events / 1_000_000) * adj_event_price +
        (inp.monthly_ml_calls / 1_000)  * adj_ml_price
    )

    # Apply churn retention and health premium adjustments
    total_adj = churn_adj + health_p
    rec_mrr   = round(rec_mrr * (1 + total_adj), 2)

    adjustment_pct = round(((rec_mrr - inp.current_mrr) / max(inp.current_mrr, 1)) * 100, 2)

    if churn_adj < 0:
        rationale = f"Retention discount {abs(churn_adj)*100:.0f}% applied (churn risk {inp.churn_risk:.0%})"
    elif health_p > 0:
        rationale = f"Health premium {health_p*100:.0f}% applied (health score {inp.health_score:.0f})"
    elif vol_mult < 1.0:
        rationale = f"Volume discount {(1-vol_mult)*100:.0f}% applied ({inp.agent_count} agents)"
    else:
        rationale = "Standard pricing"

    return PricingRecommendation(
        tenant_id=inp.tenant_id,
        current_mrr=inp.current_mrr,
        recommended_mrr=rec_mrr,
        adjustment_pct=adjustment_pct,
        rationale=rationale,
        platform_fee=adj_platform_fee,
        agent_price=adj_agent_price,
        event_price_1m=adj_event_price,
        ml_price_1k=adj_ml_price,
        valid_from=datetime.now(timezone.utc).strftime("%Y-%m-01"),
        approved=adjustment_pct <= 0,   # auto-approve discounts; premiums need approval
    )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    demo = PricingInput(
        tenant_id="demo-tenant", agent_count=30, monthly_events=8_000_000,
        monthly_ml_calls=15_000, health_score=72.0, churn_risk=0.55,
        contract_months=8, current_mrr=2_500.0, median_peer_mrr=2_200.0,
    )
    rec = compute_pricing(demo)
    print(json.dumps(vars(rec), indent=2))
