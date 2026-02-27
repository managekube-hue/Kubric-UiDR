"""
K-PSA-CRM-004_ltv_model.py

Customer Lifetime Value (LTV) batch prediction for MSP customer lifecycle analysis.
Reads from PSA CRM tables (sales_opportunities, msp_contracts, billing_usage_summary).
Distinct from K-KAI-SIM-001_ltv_predictor.py which is the real-time KAI version.
"""
from __future__ import annotations

import asyncio
import pickle
import os
from dataclasses import dataclass
from typing import Any

import asyncpg
import structlog

log = structlog.get_logger(__name__)

MODEL_PATH = os.getenv("LTV_MODEL_PATH", "/tmp/ltv_model.pkl")


@dataclass
class TenantLTV:
    tenant_id: str
    predicted_ltv: float
    months_active: int
    avg_monthly_revenue: float
    churn_risk_score: float


class LTVBatchModel:
    """
    Gradient Boosting LTV prediction model for MSP customer lifecycle.
    Trains on billing + CRM data and predicts LTV for all active tenants.
    """

    def __init__(self) -> None:
        self._model: Any = None

    async def build_training_set(self, db_pool: asyncpg.Pool):
        """
        Query billing + CRM tables and return a pandas DataFrame for training.
        Columns: months_active, avg_monthly_revenue, contract_type,
                 industry, hle_count, ticket_volume_monthly, churn_risk_score, ltv_actual
        """
        import pandas as pd  # type: ignore

        rows = await db_pool.fetch("""
            SELECT
                t.id                                                         AS tenant_id,
                EXTRACT(MONTH FROM AGE(NOW(), t.created_at))::int            AS months_active,
                COALESCE(AVG(b.total_usd), 0)                               AS avg_monthly_revenue,
                COALESCE(mc.contract_type, 'standard')                      AS contract_type,
                COALESCE(mc.min_hle, 1.0)                                   AS hle_count,
                COALESCE(tc.tickets_per_month, 0)                           AS ticket_volume_monthly,
                COALESCE(t.churn_risk_score, 0.1)                           AS churn_risk_score,
                -- LTV label: avg_monthly * expected_months_remaining
                COALESCE(AVG(b.total_usd), 0) *
                    GREATEST(1, 24 - EXTRACT(MONTH FROM AGE(NOW(), t.created_at))::int) AS ltv_actual
            FROM tenants t
            LEFT JOIN billing_usage_summary b ON b.tenant_id = t.id
            LEFT JOIN msp_contracts mc ON mc.tenant_id = t.id
                AND mc.effective_from <= CURRENT_DATE
                AND (mc.effective_to IS NULL OR mc.effective_to >= CURRENT_DATE)
            LEFT JOIN (
                SELECT tenant_id,
                       COUNT(*)::float / NULLIF(EXTRACT(MONTH FROM AGE(NOW(), MIN(created_at)))::int, 0)
                           AS tickets_per_month
                FROM service_tickets
                GROUP BY tenant_id
            ) tc ON tc.tenant_id = t.id
            WHERE t.deleted_at IS NULL
            GROUP BY t.id, t.created_at, mc.contract_type, mc.min_hle, tc.tickets_per_month,
                     t.churn_risk_score
        """)

        df = pd.DataFrame([dict(r) for r in rows])
        log.info("ltv_training_set_built", rows=len(df))
        return df

    def train(self, df) -> Any:
        """
        Train a GradientBoostingRegressor, serialize it to MODEL_PATH.
        Returns the fitted estimator.
        """
        import pandas as pd
        from sklearn.ensemble import GradientBoostingRegressor  # type: ignore
        from sklearn.preprocessing import LabelEncoder  # type: ignore

        if df.empty or len(df) < 5:
            log.warning("ltv_insufficient_data", rows=len(df))
            return None

        df = df.copy()

        # Encode categoricals
        for col in ("contract_type",):
            le = LabelEncoder()
            df[col] = le.fit_transform(df[col].fillna("standard").astype(str))

        feature_cols = [
            "months_active", "avg_monthly_revenue", "contract_type",
            "hle_count", "ticket_volume_monthly", "churn_risk_score",
        ]
        X = df[feature_cols].fillna(0)
        y = df["ltv_actual"].fillna(0)

        model = GradientBoostingRegressor(
            n_estimators=200,
            max_depth=4,
            learning_rate=0.05,
            subsample=0.8,
            random_state=42,
        )
        model.fit(X, y)
        log.info("ltv_model_trained", n_estimators=200, rows=len(df))

        with open(MODEL_PATH, "wb") as f:
            pickle.dump({"model": model, "feature_cols": feature_cols}, f)
        log.info("ltv_model_saved", path=MODEL_PATH)
        return model

    def _load_model(self) -> tuple[Any, list[str]]:
        if self._model is not None:
            return self._model, self._feature_cols  # type: ignore
        with open(MODEL_PATH, "rb") as f:
            obj = pickle.load(f)
        self._model = obj["model"]
        self._feature_cols: list[str] = obj["feature_cols"]
        return self._model, self._feature_cols

    async def predict_all_tenants(self, db_pool: asyncpg.Pool) -> None:
        """
        Predict LTV for each active tenant and persist to tenant_ltv table.
        """
        import pandas as pd
        from sklearn.preprocessing import LabelEncoder  # type: ignore

        model, feature_cols = self._load_model()

        rows = await db_pool.fetch("""
            SELECT
                t.id                                                         AS tenant_id,
                EXTRACT(MONTH FROM AGE(NOW(), t.created_at))::int            AS months_active,
                COALESCE(AVG(b.total_usd), 0)                               AS avg_monthly_revenue,
                COALESCE(mc.contract_type, 'standard')                      AS contract_type,
                COALESCE(mc.min_hle, 1.0)                                   AS hle_count,
                COALESCE(tc.tickets_per_month, 0)                           AS ticket_volume_monthly,
                COALESCE(t.churn_risk_score, 0.1)                           AS churn_risk_score
            FROM tenants t
            LEFT JOIN billing_usage_summary b ON b.tenant_id = t.id
            LEFT JOIN msp_contracts mc ON mc.tenant_id = t.id
                AND mc.effective_from <= CURRENT_DATE
                AND (mc.effective_to IS NULL OR mc.effective_to >= CURRENT_DATE)
            LEFT JOIN (
                SELECT tenant_id,
                       COUNT(*)::float / NULLIF(EXTRACT(MONTH FROM AGE(NOW(), MIN(created_at)))::int, 0)
                           AS tickets_per_month
                FROM service_tickets
                GROUP BY tenant_id
            ) tc ON tc.tenant_id = t.id
            WHERE t.deleted_at IS NULL
            GROUP BY t.id, t.created_at, mc.contract_type, mc.min_hle,
                     tc.tickets_per_month, t.churn_risk_score
        """)

        if not rows:
            log.warning("ltv_predict_all_no_tenants")
            return

        df = pd.DataFrame([dict(r) for r in rows])
        tenant_ids = df["tenant_id"].tolist()

        le = LabelEncoder()
        df["contract_type"] = le.fit_transform(df["contract_type"].fillna("standard").astype(str))
        X = df[feature_cols].fillna(0)
        predictions = model.predict(X)

        for tid, pred in zip(tenant_ids, predictions):
            await db_pool.execute("""
                INSERT INTO tenant_ltv (tenant_id, predicted_ltv, updated_at)
                VALUES ($1, $2, NOW())
                ON CONFLICT (tenant_id) DO UPDATE SET predicted_ltv = EXCLUDED.predicted_ltv, updated_at = NOW()
            """, str(tid), float(pred))

        log.info("ltv_predictions_saved", count=len(tenant_ids))

    async def generate_report(self, db_pool: asyncpg.Pool) -> dict:
        """Return top 10 tenants by LTV and at-risk customers (churn_risk > 0.6)."""
        top10 = await db_pool.fetch("""
            SELECT tl.tenant_id, t.name, tl.predicted_ltv
            FROM tenant_ltv tl
            JOIN tenants t ON t.id = tl.tenant_id
            ORDER BY tl.predicted_ltv DESC
            LIMIT 10
        """)
        at_risk = await db_pool.fetch("""
            SELECT tl.tenant_id, t.name, tl.predicted_ltv, t.churn_risk_score
            FROM tenant_ltv tl
            JOIN tenants t ON t.id = tl.tenant_id
            WHERE t.churn_risk_score > 0.6
            ORDER BY tl.predicted_ltv DESC
        """)
        return {
            "top_10_by_ltv": [dict(r) for r in top10],
            "at_risk_customers": [dict(r) for r in at_risk],
        }


async def main() -> None:
    db_url = os.getenv("DATABASE_URL", "postgresql://localhost/kubric")
    db_pool = await asyncpg.create_pool(db_url)

    ltv = LTVBatchModel()
    df = await ltv.build_training_set(db_pool)
    ltv.train(df)
    await ltv.predict_all_tenants(db_pool)
    report = await ltv.generate_report(db_pool)

    print(f"Top 10 by LTV: {len(report['top_10_by_ltv'])} tenants")
    for r in report["top_10_by_ltv"]:
        print(f"  {r.get('name', r['tenant_id'])}: ${r['predicted_ltv']:,.0f}")

    print(f"\nAt-risk: {len(report['at_risk_customers'])} tenants")
    await db_pool.close()


if __name__ == "__main__":
    asyncio.run(main())
