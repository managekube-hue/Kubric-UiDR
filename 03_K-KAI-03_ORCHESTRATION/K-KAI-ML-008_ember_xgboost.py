"""
K-KAI-ML-008_ember_xgboost.py
XGBoost malware classifier trained on EMBER dataset features.
EMBER feature vector dimension: 2381
"""

import logging
import os
from typing import Optional

import numpy as np

logger = logging.getLogger(__name__)

try:
    import xgboost as xgb
    _XGB_AVAILABLE = True
except ImportError:
    _XGB_AVAILABLE = False

try:
    from sklearn.metrics import (
        accuracy_score,
        f1_score,
        precision_score,
        recall_score,
        roc_auc_score,
    )
    _SKLEARN_AVAILABLE = True
except ImportError:
    _SKLEARN_AVAILABLE = False

EMBER_FEATURE_DIM = 2381
MODEL_FILENAME = "ember_xgboost.json"

DEFAULT_PARAMS: dict = {
    "n_estimators": 500,
    "max_depth": 8,
    "learning_rate": 0.1,
    "subsample": 0.8,
    "colsample_bytree": 0.8,
    "use_label_encoder": False,
    "eval_metric": "logloss",
    "tree_method": "hist",           # fast CPU training
    "n_jobs": -1,
    "random_state": 42,
}


class EMBERXGBoostClassifier:
    """
    Binary malware classifier using XGBoost, tuned for EMBER 2381-dim features.

    Labels: 0 = benign, 1 = malicious
    The model is saved/loaded in XGBoost JSON format to MODEL_DIR/ember_xgboost.json.
    """

    def __init__(
        self,
        model_dir: Optional[str] = None,
        params: Optional[dict] = None,
    ) -> None:
        if not _XGB_AVAILABLE:
            raise RuntimeError("xgboost not installed — pip install xgboost")
        self._model_dir = model_dir or os.environ.get("MODEL_DIR", "/tmp/kai/models")
        os.makedirs(self._model_dir, exist_ok=True)
        merged = dict(DEFAULT_PARAMS)
        if params:
            merged.update(params)
        self._params = merged
        self._clf: Optional[xgb.XGBClassifier] = None

    # ------------------------------------------------------------------
    # Training
    # ------------------------------------------------------------------

    def train(
        self,
        X_train: np.ndarray,
        y_train: np.ndarray,
        X_val: Optional[np.ndarray] = None,
        y_val: Optional[np.ndarray] = None,
        verbose_eval: int = 50,
    ) -> None:
        """
        Fit the XGBoost classifier on EMBER feature vectors.

        X_train — shape (n_samples, 2381)
        y_train — shape (n_samples,), values in {0, 1}
        """
        if X_train.shape[1] != EMBER_FEATURE_DIM:
            logger.warning(
                "Expected %d features; got %d — proceeding anyway",
                EMBER_FEATURE_DIM,
                X_train.shape[1],
            )

        self._clf = xgb.XGBClassifier(**self._params)

        eval_set = None
        if X_val is not None and y_val is not None:
            eval_set = [(X_val, y_val)]

        self._clf.fit(
            X_train,
            y_train,
            eval_set=eval_set,
            verbose=verbose_eval,
        )
        logger.info(
            "EMBERXGBoost trained — samples=%d features=%d",
            X_train.shape[0],
            X_train.shape[1],
        )

    # ------------------------------------------------------------------
    # Inference
    # ------------------------------------------------------------------

    def predict(self, X: np.ndarray) -> np.ndarray:
        """Return binary class labels (0=benign, 1=malicious)."""
        self._check_fitted()
        return self._clf.predict(X)

    def predict_proba(self, X: np.ndarray) -> np.ndarray:
        """Return probability matrix, shape (n_samples, 2)."""
        self._check_fitted()
        return self._clf.predict_proba(X)

    # ------------------------------------------------------------------
    # Persistence
    # ------------------------------------------------------------------

    def save_model(self, path: Optional[str] = None) -> None:
        """Save model to JSON format."""
        self._check_fitted()
        if path is None:
            path = os.path.join(self._model_dir, MODEL_FILENAME)
        self._clf.save_model(path)
        logger.info("EMBER model saved — path=%s", path)

    def load_model(self, path: Optional[str] = None) -> None:
        """Load model from JSON format."""
        if path is None:
            path = os.path.join(self._model_dir, MODEL_FILENAME)
        self._clf = xgb.XGBClassifier()
        self._clf.load_model(path)
        logger.info("EMBER model loaded — path=%s", path)

    # ------------------------------------------------------------------
    # Evaluation
    # ------------------------------------------------------------------

    def evaluate(self, X_test: np.ndarray, y_test: np.ndarray) -> dict:
        """
        Compute classification metrics.

        Returns: accuracy, precision, recall, f1, roc_auc,
                 false_positive_rate, false_negative_rate
        """
        self._check_fitted()
        if not _SKLEARN_AVAILABLE:
            raise RuntimeError("scikit-learn not installed")

        y_pred = self.predict(X_test)
        y_proba = self.predict_proba(X_test)[:, 1]

        tn = int(np.sum((y_pred == 0) & (y_test == 0)))
        fp = int(np.sum((y_pred == 1) & (y_test == 0)))
        fn = int(np.sum((y_pred == 0) & (y_test == 1)))
        tp = int(np.sum((y_pred == 1) & (y_test == 1)))

        fpr = fp / (fp + tn) if (fp + tn) > 0 else 0.0
        fnr = fn / (fn + tp) if (fn + tp) > 0 else 0.0

        metrics = {
            "accuracy": float(accuracy_score(y_test, y_pred)),
            "precision": float(precision_score(y_test, y_pred, zero_division=0)),
            "recall": float(recall_score(y_test, y_pred, zero_division=0)),
            "f1_score": float(f1_score(y_test, y_pred, zero_division=0)),
            "roc_auc": float(roc_auc_score(y_test, y_proba)),
            "false_positive_rate": fpr,
            "false_negative_rate": fnr,
            "true_positives": tp,
            "false_positives": fp,
            "true_negatives": tn,
            "false_negatives": fn,
        }
        logger.info("EMBERXGBoost evaluation — %s", metrics)
        return metrics

    def _check_fitted(self) -> None:
        if self._clf is None:
            raise RuntimeError("Model has not been trained — call train() or load_model() first")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    # Quick smoke test with random data
    rng = np.random.default_rng(0)
    X = rng.random((500, EMBER_FEATURE_DIM)).astype(np.float32)
    y = rng.integers(0, 2, size=500)
    clf = EMBERXGBoostClassifier(params={"n_estimators": 10, "max_depth": 3})
    clf.train(X[:400], y[:400], verbose_eval=5)
    metrics = clf.evaluate(X[400:], y[400:])
    print(metrics)
