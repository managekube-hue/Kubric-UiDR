"""
K-KAI-RISK-001: pyFAIR Quantitative Risk Model
Uses the pyfair library to model primary and secondary loss exposure for an
asset/threat scenario pair. Saves results to PostgreSQL.
"""

import asyncio
import json
import logging
import os
import uuid
from datetime import datetime, timezone
from typing import Any

import asyncpg
import numpy as np
import pyfair
from pyfair import FairModel, FairSimpleReport

logger = logging.getLogger("K-KAI-RISK-001")

DATABASE_URL: str = os.getenv("DATABASE_URL", "postgresql://kubric:kubric@localhost/kubric")
N_SIMULATIONS: int = 10_000

_CREATE_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS risk_assessments (
    id                  UUID PRIMARY KEY,
    asset               TEXT NOT NULL,
    threat_scenario     TEXT NOT NULL,
    primary_loss_mean   DOUBLE PRECISION,
    primary_loss_p10    DOUBLE PRECISION,
    primary_loss_p90    DOUBLE PRECISION,
    secondary_loss_mean DOUBLE PRECISION,
    secondary_loss_p10  DOUBLE PRECISION,
    secondary_loss_p90  DOUBLE PRECISION,
    total_loss_mean     DOUBLE PRECISION,
    total_loss_p10      DOUBLE PRECISION,
    total_loss_p90      DOUBLE PRECISION,
    params              JSONB,
    assessed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
"""


class PyFAIRModel:
    """
    Wraps the pyfair library to run Monte Carlo FAIR risk simulations.

    Default params (all in USD) can be overridden via the params dict:
      tef_low, tef_mode, tef_high  - Threat Event Frequency (annual)
      vuln_low, vuln_mode, vuln_high - Vulnerability (0-1)
      plm_low, plm_mode, plm_high  - Primary Loss Magnitude
      slef_low, slef_mode, slef_high - Secondary Loss Event Frequency (0-1)
      slm_low, slm_mode, slm_high  - Secondary Loss Magnitude
    """

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def model_risk(self, asset: str, threat_scenario: str, params: dict) -> dict:
        """
        Run a FAIR Monte Carlo simulation.

        Args:
            asset:           Asset name / identifier.
            threat_scenario: Free-text description of the threat scenario.
            params:          Dictionary of FAIR parameter overrides (see class docstring).

        Returns:
            dict with primary_loss, secondary_loss, total_loss_exposure
            (each containing mean, p10, p90), plus assessment metadata.
        """
        p = self._defaults(params)

        # Build FAIR model
        model = FairModel(name=f"{asset} | {threat_scenario}", n_simulations=N_SIMULATIONS)

        model.input_data("Threat Event Frequency", low=p["tef_low"], mode=p["tef_mode"], high=p["tef_high"])
        model.input_data("Vulnerability", low=p["vuln_low"], mode=p["vuln_mode"], high=p["vuln_high"])
        model.input_data("Primary Loss Magnitude", low=p["plm_low"], mode=p["plm_mode"], high=p["plm_high"])
        model.input_data("Secondary Loss Event Frequency", low=p["slef_low"], mode=p["slef_mode"], high=p["slef_high"])
        model.input_data("Secondary Loss Magnitude", low=p["slm_low"], mode=p["slm_mode"], high=p["slm_high"])

        model.calculate_all()

        results = model.export_results()
        risk_id = str(uuid.uuid4())

        # Extract simulation arrays
        primary_arr: np.ndarray = np.array(results.get("Primary Loss", [0.0]))
        secondary_arr: np.ndarray = np.array(results.get("Secondary Risk", [0.0]))
        total_arr: np.ndarray = primary_arr + secondary_arr

        assessment: dict = {
            "id": risk_id,
            "asset": asset,
            "threat_scenario": threat_scenario,
            "primary_loss": {
                "mean": float(np.mean(primary_arr)),
                "p10": float(np.percentile(primary_arr, 10)),
                "p90": float(np.percentile(primary_arr, 90)),
            },
            "secondary_loss": {
                "mean": float(np.mean(secondary_arr)),
                "p10": float(np.percentile(secondary_arr, 10)),
                "p90": float(np.percentile(secondary_arr, 90)),
            },
            "total_loss_exposure": {
                "mean": float(np.mean(total_arr)),
                "p10": float(np.percentile(total_arr, 10)),
                "p90": float(np.percentile(total_arr, 90)),
            },
            "params": p,
            "assessed_at": datetime.now(timezone.utc).isoformat(),
        }

        # Persist synchronously (the caller may be async; wrap if needed)
        try:
            asyncio.get_event_loop().run_until_complete(self._save_assessment(assessment))
        except RuntimeError:
            # No running event loop – create a new one
            asyncio.run(self._save_assessment(assessment))

        return assessment

    # ------------------------------------------------------------------
    # PostgreSQL persistence
    # ------------------------------------------------------------------

    async def _save_assessment(self, a: dict) -> None:
        try:
            pool = await asyncpg.create_pool(DATABASE_URL, min_size=1, max_size=3)
            async with pool.acquire() as conn:
                await conn.execute(_CREATE_TABLE_SQL)
                await conn.execute(
                    """
                    INSERT INTO risk_assessments (
                        id, asset, threat_scenario,
                        primary_loss_mean, primary_loss_p10, primary_loss_p90,
                        secondary_loss_mean, secondary_loss_p10, secondary_loss_p90,
                        total_loss_mean, total_loss_p10, total_loss_p90,
                        params
                    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)
                    """,
                    a["id"], a["asset"], a["threat_scenario"],
                    a["primary_loss"]["mean"], a["primary_loss"]["p10"], a["primary_loss"]["p90"],
                    a["secondary_loss"]["mean"], a["secondary_loss"]["p10"], a["secondary_loss"]["p90"],
                    a["total_loss_exposure"]["mean"], a["total_loss_exposure"]["p10"], a["total_loss_exposure"]["p90"],
                    json.dumps(a["params"]),
                )
            await pool.close()
            logger.info("Saved risk assessment %s to database.", a["id"])
        except Exception as exc:  # noqa: BLE001
            logger.error("Failed to save risk assessment: %s", exc)

    # ------------------------------------------------------------------
    # Defaults
    # ------------------------------------------------------------------

    @staticmethod
    def _defaults(params: dict) -> dict:
        d: dict[str, float] = {
            "tef_low":  params.get("tef_low",  0.1),
            "tef_mode": params.get("tef_mode", 1.0),
            "tef_high": params.get("tef_high", 5.0),
            "vuln_low":  params.get("vuln_low",  0.1),
            "vuln_mode": params.get("vuln_mode", 0.4),
            "vuln_high": params.get("vuln_high", 0.9),
            "plm_low":   params.get("plm_low",   10_000),
            "plm_mode":  params.get("plm_mode",  100_000),
            "plm_high":  params.get("plm_high",  1_000_000),
            "slef_low":  params.get("slef_low",  0.05),
            "slef_mode": params.get("slef_mode", 0.3),
            "slef_high": params.get("slef_high", 0.7),
            "slm_low":   params.get("slm_low",   5_000),
            "slm_mode":  params.get("slm_mode",  50_000),
            "slm_high":  params.get("slm_high",  500_000),
        }
        return d
