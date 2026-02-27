"""
K-PSA-CRM-003_pyfair_risk_model.py

FAIR (Factor Analysis of Information Risk) Monte Carlo simulation
for cyber risk quantification. Uses the pyfair library.
"""
from __future__ import annotations

import sys
import asyncio
import statistics
from dataclasses import dataclass, field
from typing import Any

import structlog

log = structlog.get_logger(__name__)


@dataclass
class RiskResult:
    scenario_name: str
    ale_mean: float
    ale_90th: float
    ale_95th: float
    risk_level: str  # low / medium / high / critical


class FairRiskModel:
    """FAIR Monte Carlo risk quantification engine."""

    def build_model(
        self,
        name: str,
        tef_min: float,
        tef_max: float,
        v_pct: float,
        plm_min: float,
        plm_max: float,
    ) -> Any:
        """
        Build and return an unexecuted FairModel.
        Falls back to a simple dict-based model if pyfair is unavailable.
        """
        try:
            from pyfair import FairModel  # type: ignore

            model = FairModel(name=name, n_simulations=10_000)
            model.input_data("Threat Event Frequency", mean=tef_min, stdev=(tef_max - tef_min) / 4)
            model.input_data("Vulnerability", mean=v_pct)
            model.input_data("Primary Loss Magnitude", low=plm_min, mode=(plm_min + plm_max) / 2, high=plm_max)
            return model
        except ImportError:
            return {
                "name": name,
                "tef_min": tef_min,
                "tef_max": tef_max,
                "v_pct": v_pct,
                "plm_min": plm_min,
                "plm_max": plm_max,
            }

    def run_simulation(
        self,
        threat_event_freq_min: float,
        threat_event_freq_max: float,
        vuln_pct: float,
        loss_min: float,
        loss_max: float,
        n_sims: int = 10_000,
    ) -> dict[str, float]:
        """
        Run a Monte Carlo FAIR simulation.
        Returns mean_loss, stddev, 90th_pct, 95th_pct, max_loss in USD.
        """
        import random
        import math

        rng = random.Random(42)

        def pert_sample(lo: float, mode: float, hi: float) -> float:
            """4-point PERT distribution approximation via beta."""
            alpha = 1 + 4 * ((mode - lo) / (hi - lo + 1e-12))
            beta_param = 1 + 4 * ((hi - mode) / (hi - lo + 1e-12))
            u = rng.betavariate(alpha, beta_param)
            return lo + u * (hi - lo)

        losses: list[float] = []
        mode_tef = (threat_event_freq_min + threat_event_freq_max) / 2
        mode_loss = (loss_min + loss_max) / 2

        for _ in range(n_sims):
            tef = pert_sample(threat_event_freq_min, mode_tef, threat_event_freq_max)
            events = max(0, round(rng.gauss(tef, tef * 0.2)))
            realized = sum(1 for _ in range(events) if rng.random() < vuln_pct)
            if realized == 0:
                losses.append(0.0)
                continue
            loss = sum(
                pert_sample(loss_min, mode_loss, loss_max) for _ in range(realized)
            )
            losses.append(loss)

        losses.sort()
        mean_loss = statistics.mean(losses)
        stddev = statistics.stdev(losses) if len(losses) > 1 else 0.0
        p90 = losses[int(0.90 * n_sims)]
        p95 = losses[int(0.95 * n_sims)]
        max_loss = losses[-1]

        log.info(
            "fair_simulation_complete",
            mean_loss=round(mean_loss, 2),
            p90=round(p90, 2),
            p95=round(p95, 2),
            max_loss=round(max_loss, 2),
        )
        return {
            "mean_loss": round(mean_loss, 2),
            "stddev": round(stddev, 2),
            "90th_pct": round(p90, 2),
            "95th_pct": round(p95, 2),
            "max_loss": round(max_loss, 2),
        }

    def calculate_annual_risk(
        self,
        asset_value_usd: float,
        threat_capability: float,
        vuln_pct: float,
    ) -> RiskResult:
        """
        Derive ALE using a simplified FAIR calculation:
        ALE = asset_value * threat_capability * vuln_pct
        """
        ale = asset_value_usd * threat_capability * vuln_pct
        result = self.run_simulation(
            threat_event_freq_min=threat_capability * 0.5,
            threat_event_freq_max=threat_capability * 1.5,
            vuln_pct=vuln_pct,
            loss_min=asset_value_usd * 0.01,
            loss_max=asset_value_usd,
        )

        risk_level = "low"
        if ale > 1_000_000:
            risk_level = "critical"
        elif ale > 500_000:
            risk_level = "high"
        elif ale > 100_000:
            risk_level = "medium"

        return RiskResult(
            scenario_name="annual_risk",
            ale_mean=result["mean_loss"],
            ale_90th=result["90th_pct"],
            ale_95th=result["95th_pct"],
            risk_level=risk_level,
        )

    def scenario_comparison(self, scenarios: list[dict]) -> list[RiskResult]:
        """
        Compare multiple FAIR scenarios.

        Each scenario dict: {name, tef_min, tef_max, vuln_pct, loss_min, loss_max}
        """
        results: list[RiskResult] = []
        for s in scenarios:
            sim = self.run_simulation(
                threat_event_freq_min=s["tef_min"],
                threat_event_freq_max=s["tef_max"],
                vuln_pct=s["vuln_pct"],
                loss_min=s["loss_min"],
                loss_max=s["loss_max"],
            )
            ale = sim["mean_loss"]
            if ale > 1_000_000:
                level = "critical"
            elif ale > 500_000:
                level = "high"
            elif ale > 100_000:
                level = "medium"
            else:
                level = "low"

            results.append(
                RiskResult(
                    scenario_name=s.get("name", "unnamed"),
                    ale_mean=sim["mean_loss"],
                    ale_90th=sim["90th_pct"],
                    ale_95th=sim["95th_pct"],
                    risk_level=level,
                )
            )
        return results


async def save_risk_model(
    tenant_id: str,
    asset_id: str,
    result: RiskResult,
    db_pool: Any,
) -> None:
    """Persist a RiskResult to the PostgreSQL fair_risk_results table."""
    await db_pool.execute(
        """
        INSERT INTO fair_risk_results
            (tenant_id, asset_id, scenario_name, ale_mean, ale_90th, ale_95th, risk_level, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
        ON CONFLICT (tenant_id, asset_id, scenario_name)
        DO UPDATE SET
            ale_mean   = EXCLUDED.ale_mean,
            ale_90th   = EXCLUDED.ale_90th,
            ale_95th   = EXCLUDED.ale_95th,
            risk_level = EXCLUDED.risk_level,
            created_at = NOW()
        """,
        tenant_id,
        asset_id,
        result.scenario_name,
        result.ale_mean,
        result.ale_90th,
        result.ale_95th,
        result.risk_level,
    )
    log.info("fair_risk_saved", tenant_id=tenant_id, asset_id=asset_id, risk_level=result.risk_level)


if __name__ == "__main__":
    model = FairRiskModel()

    default_scenario = {
        "name": "ransomware_attack",
        "tef_min": 1.0,
        "tef_max": 5.0,
        "vuln_pct": 0.35,
        "loss_min": 50_000,
        "loss_max": 2_000_000,
    }
    result = model.scenario_comparison([default_scenario])
    for r in result:
        print(
            f"Scenario: {r.scenario_name}\n"
            f"  ALE Mean:  ${r.ale_mean:,.2f}\n"
            f"  ALE 90th:  ${r.ale_90th:,.2f}\n"
            f"  ALE 95th:  ${r.ale_95th:,.2f}\n"
            f"  Risk Level: {r.risk_level.upper()}\n"
        )
