"""
K-KAI Simulate: Churn Simulator
Monte Carlo simulation of tenant churn scenarios to project
MRR impact and retention ROI under different intervention strategies.
"""
from __future__ import annotations
import json, logging
import numpy as np
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Dict, List

logger = logging.getLogger(__name__)

@dataclass
class InterventionStrategy:
    name:               str
    monthly_cost_usd:   float
    churn_reduction_pct: float    # percentage points reduction in monthly churn rate
    time_to_effect_months: int    # months until intervention is fully effective

@dataclass
class SimulationResult:
    strategy:         str
    simulations:      int
    horizon_months:   int
    expected_mrr_end: float
    mrr_p10:          float
    mrr_p50:          float
    mrr_p90:          float
    expected_retention_roi: float
    ts:               str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

BUILT_IN_STRATEGIES: List[InterventionStrategy] = [
    InterventionStrategy("no_action",        0.0,     0.0,  0),
    InterventionStrategy("health_monitoring", 5_000.0, 1.5,  2),
    InterventionStrategy("qbr_program",      15_000.0, 3.0,  3),
    InterventionStrategy("dedicated_csm",    40_000.0, 5.0,  1),
]

def simulate_churn(
    initial_mrr:    float,
    tenant_count:   int,
    base_churn_pct: float,         # monthly churn rate as percent (e.g. 3.0 = 3%)
    strategy:       InterventionStrategy,
    horizon_months: int = 12,
    n_sims:         int = 1000,
    seed:           int = 42,
) -> SimulationResult:
    rng       = np.random.default_rng(seed)
    mrr_arr   = np.full(n_sims, initial_mrr, dtype=np.float64)
    arpu      = initial_mrr / max(tenant_count, 1)

    for month in range(1, horizon_months + 1):
        # Time-lagged churn reduction
        ramp        = min(1.0, month / max(strategy.time_to_effect_months, 1))
        eff_churn   = max(0.0, base_churn_pct - strategy.churn_reduction_pct * ramp)
        churn_rate  = eff_churn / 100.0

        # Random churn from a binomial draw on implicit tenant count
        tenants_now = (mrr_arr / arpu).round().astype(int).clip(0)
        churned     = rng.binomial(tenants_now, churn_rate)
        new_tenants = rng.poisson(tenant_count * 0.02, n_sims)   # 2% monthly new
        net_change  = (new_tenants - churned) * arpu
        mrr_arr    += net_change
        mrr_arr     = mrr_arr.clip(0)

    # Subtract intervention costs
    total_cost  = strategy.monthly_cost_usd * horizon_months
    mrr_p10  = float(np.percentile(mrr_arr, 10))
    mrr_p50  = float(np.percentile(mrr_arr, 50))
    mrr_p90  = float(np.percentile(mrr_arr, 90))
    expected = float(mrr_arr.mean())

    roi = round((expected - initial_mrr - total_cost) / max(total_cost, 1), 4)

    return SimulationResult(
        strategy=strategy.name,
        simulations=n_sims,
        horizon_months=horizon_months,
        expected_mrr_end=round(expected, 2),
        mrr_p10=round(mrr_p10, 2),
        mrr_p50=round(mrr_p50, 2),
        mrr_p90=round(mrr_p90, 2),
        expected_retention_roi=roi,
    )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    results = []
    for strat in BUILT_IN_STRATEGIES:
        res = simulate_churn(
            initial_mrr=500_000,
            tenant_count=200,
            base_churn_pct=3.0,
            strategy=strat,
            horizon_months=12,
        )
        results.append(vars(res))
    print(json.dumps(results, indent=2))
