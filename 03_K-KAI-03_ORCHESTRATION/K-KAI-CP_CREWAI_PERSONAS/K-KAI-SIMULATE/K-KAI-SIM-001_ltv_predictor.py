"""
K-KAI Simulate: LTV Predictor
Predicts 12-month customer Lifetime Value using a gradient-boosted tree
trained on usage intensity, health scores, contract value, and support cost.
"""
from __future__ import annotations
import json, logging, os, pickle
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, List, Optional
import numpy as np
from sklearn.ensemble import GradientBoostingRegressor
from sklearn.preprocessing import StandardScaler
from sklearn.model_selection import cross_val_score

logger = logging.getLogger(__name__)
MODEL_PATH = Path(os.getenv("LTV_MODEL_PATH", "/tmp/kai_ltv_model.pkl"))

FEATURES = [
    "contract_value_usd",   # current ARR
    "agent_count",
    "avg_health_score",     # 0-100
    "months_active",
    "support_tickets_12m",
    "events_per_day_avg",
    "ml_calls_per_month",
    "churn_risk_score",     # 0-1 (from Sentinel)
]

@dataclass
class LtvPrediction:
    tenant_id:        str
    predicted_ltv:    float     # USD, 12-month
    confidence_lower: float
    confidence_upper: float
    key_driver:       str
    scored_at:        str

def _confidence_interval(pred: float, std: float) -> tuple:
    return (round(max(0, pred - 1.96 * std), 2), round(pred + 1.96 * std, 2))

class LtvPredictor:
    def __init__(self, model_path: Path = MODEL_PATH):
        self.model_path = model_path
        self._model:  Optional[GradientBoostingRegressor] = None
        self._scaler: Optional[StandardScaler] = None

    def train(self, X: np.ndarray, y: np.ndarray) -> Dict:
        self._scaler = StandardScaler()
        X_scaled     = self._scaler.fit_transform(X)
        self._model  = GradientBoostingRegressor(
            n_estimators=200, learning_rate=0.05, max_depth=4,
            subsample=0.8, random_state=42,
        )
        cv_scores = cross_val_score(self._model, X_scaled, y, cv=5, scoring="r2")
        self._model.fit(X_scaled, y)
        with open(self.model_path, "wb") as fh:
            pickle.dump({"model": self._model, "scaler": self._scaler}, fh)
        return {"r2_cv": float(cv_scores.mean()), "rows": len(X),
                "model_path": str(self.model_path)}

    def load(self) -> None:
        with open(self.model_path, "rb") as fh:
            obj = pickle.load(fh)
        self._model  = obj["model"]
        self._scaler = obj["scaler"]

    def predict(self, tenant_id: str, features: Dict[str, float]) -> LtvPrediction:
        if self._model is None:
            self.load()
        x = np.array([[features.get(f, 0) for f in FEATURES]])
        x_scaled = self._scaler.transform(x)
        pred  = float(self._model.predict(x_scaled)[0])
        std   = float(np.std([
            t.predict(x_scaled)[0] for t in self._model.estimators_[:, 0]
        ]))
        lo, hi = _confidence_interval(pred, std)
        importances = dict(zip(FEATURES, self._model.feature_importances_))
        key_driver  = max(importances, key=lambda k: importances[k])
        return LtvPrediction(
            tenant_id=tenant_id,
            predicted_ltv=round(pred, 2),
            confidence_lower=lo,
            confidence_upper=hi,
            key_driver=key_driver,
            scored_at=datetime.now(timezone.utc).isoformat(),
        )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    rng = np.random.default_rng(42)
    rows = 200
    X = np.column_stack([
        rng.uniform(10_000, 500_000, rows),   # contract
        rng.integers(1, 200, rows),            # agents
        rng.uniform(30, 100, rows),            # health
        rng.integers(1, 60, rows),             # months
        rng.integers(0, 50, rows),             # tickets
        rng.uniform(1000, 1_000_000, rows),    # events/day
        rng.integers(0, 50_000, rows),         # ml_calls
        rng.uniform(0, 0.9, rows),             # churn_risk
    ])
    y = X[:, 0] * rng.uniform(0.8, 2.0, rows)   # synthetic LTV ~= 0.8-2x contract
    predictor = LtvPredictor()
    result    = predictor.train(X, y)
    print("Train:", json.dumps(result))
    demo = {f: v for f, v in zip(FEATURES, [100_000, 20, 75, 18, 5, 50_000, 5_000, 0.15])}
    pred = predictor.predict("demo-tenant", demo)
    print("Predict:", json.dumps(vars(pred)))
