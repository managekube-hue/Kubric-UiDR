"""
K-KAI Sentinel: Churn Risk Model
Trains a logistic regression model on tenant health and usage features
to predict 30-day churn probability. Scores are stored in PostgreSQL
and used by the Comm persona to trigger proactive retention outreach.
"""
from __future__ import annotations
import asyncio, json, logging, os, pickle
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Dict, List, Optional, Tuple
import asyncpg
import numpy as np
from sklearn.linear_model import LogisticRegression
from sklearn.preprocessing import StandardScaler
from sklearn.model_selection import cross_val_score

logger = logging.getLogger(__name__)
PG_DSN     = os.getenv("PG_DSN", "postgresql://kubric:kubric@localhost:5432/kubric")
MODEL_PATH = Path(os.getenv("CHURN_MODEL_PATH", "/tmp/kai_churn_model.pkl"))

FEATURES = [
    "avg_health_score",    # 0-100
    "days_since_login",
    "open_incidents_30d",
    "patches_applied_30d",
    "agents_online_pct",
    "compliance_drift_30d",
    "support_tickets_30d",
]

@dataclass
class ChurnPrediction:
    tenant_id:   str
    probability: float       # 0-1
    risk_tier:   str         # HIGH / MEDIUM / LOW
    features:    Dict[str, float]
    scored_at:   str

def _tier(prob: float) -> str:
    if prob >= 0.70: return "HIGH"
    if prob >= 0.40: return "MEDIUM"
    return "LOW"

class ChurnRiskModel:
    def __init__(self, pg_dsn: str = PG_DSN, model_path: Path = MODEL_PATH):
        self.pg_dsn     = pg_dsn
        self.model_path = model_path
        self._scaler: Optional[StandardScaler]       = None
        self._model:  Optional[LogisticRegression]   = None

    # ── data loading ──────────────────────────────────────────────
    async def _load_training_data(self) -> Tuple[np.ndarray, np.ndarray]:
        pool = await asyncpg.create_pool(self.pg_dsn, min_size=2, max_size=4)
        try:
            sql = """
                SELECT
                    AVG(h.overall_score)          AS avg_health_score,
                    COALESCE(EXTRACT(DAY FROM NOW() - MAX(u.last_login)), 90) AS days_since_login,
                    COUNT(DISTINCT i.id) FILTER (WHERE i.created_at > NOW()-INTERVAL '30 days') AS open_incidents_30d,
                    COUNT(DISTINCT p.id) FILTER (WHERE p.applied_at  > NOW()-INTERVAL '30 days') AS patches_applied_30d,
                    AVG(a.online_pct)             AS agents_online_pct,
                    COUNT(DISTINCT d.id) FILTER (WHERE d.created_at > NOW()-INTERVAL '30 days') AS compliance_drift_30d,
                    COUNT(DISTINCT s.id) FILTER (WHERE s.created_at > NOW()-INTERVAL '30 days') AS support_tickets_30d,
                    CASE WHEN t.churned_at IS NOT NULL THEN 1 ELSE 0 END        AS label
                FROM kai_tenants t
                LEFT JOIN kai_health_history h ON h.tenant_id = t.id
                LEFT JOIN kai_user_sessions u  ON u.tenant_id = t.id
                LEFT JOIN kai_incidents i       ON i.tenant_id = t.id
                LEFT JOIN kai_patches p         ON p.tenant_id = t.id
                LEFT JOIN kai_agent_stats a     ON a.tenant_id = t.id
                LEFT JOIN kai_drift_events d    ON d.tenant_id = t.id
                LEFT JOIN kai_support_tickets s ON s.tenant_id = t.id
                GROUP BY t.id, t.churned_at
                HAVING COUNT(h.id) > 10
            """
            rows = await pool.fetch(sql)
            X = np.array([[r[f] or 0 for f in FEATURES] for r in rows], dtype=np.float32)
            y = np.array([r["label"] for r in rows], dtype=np.int32)
            return X, y
        finally:
            await pool.close()

    # ── training ──────────────────────────────────────────────────
    async def train(self) -> Dict:
        X, y = await self._load_training_data()
        if len(X) < 20:
            logger.warning("Insufficient training data (%d rows)", len(X))
            return {"error": "insufficient_data", "rows": len(X)}

        self._scaler = StandardScaler()
        X_scaled     = self._scaler.fit_transform(X)
        self._model  = LogisticRegression(C=1.0, max_iter=500, class_weight="balanced")
        cv_scores    = cross_val_score(self._model, X_scaled, y, cv=5, scoring="roc_auc")
        self._model.fit(X_scaled, y)

        with open(self.model_path, "wb") as fh:
            pickle.dump({"model": self._model, "scaler": self._scaler}, fh)

        return {"roc_auc_cv": float(cv_scores.mean()), "rows": len(X),
                "model_path": str(self.model_path)}

    # ── inference ─────────────────────────────────────────────────
    def load(self) -> None:
        with open(self.model_path, "rb") as fh:
            obj = pickle.load(fh)
        self._model  = obj["model"]
        self._scaler = obj["scaler"]

    def predict(self, feature_dict: Dict[str, float]) -> ChurnPrediction:
        if self._model is None:
            self.load()
        x = np.array([[feature_dict.get(f, 0) for f in FEATURES]], dtype=np.float32)
        x_scaled = self._scaler.transform(x)
        prob = float(self._model.predict_proba(x_scaled)[0][1])
        return ChurnPrediction(
            tenant_id  = feature_dict.get("tenant_id", "unknown"),
            probability= round(prob, 4),
            risk_tier  = _tier(prob),
            features   = {f: feature_dict.get(f, 0) for f in FEATURES},
            scored_at  = datetime.now(timezone.utc).isoformat(),
        )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    model = ChurnRiskModel()
    demo  = {
        "tenant_id": "demo-tenant-001",
        "avg_health_score": 45.0, "days_since_login": 12,
        "open_incidents_30d": 8, "patches_applied_30d": 2,
        "agents_online_pct": 60.0, "compliance_drift_30d": 15,
        "support_tickets_30d": 3,
    }
    # Can't train without DB; just show structure
    print(json.dumps({"demo_features": demo, "features": FEATURES}))
