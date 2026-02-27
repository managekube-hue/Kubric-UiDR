"""
K-KAI-ML-009_unswnb15_random_forest.py
Random Forest network intrusion classifier trained on UNSW-NB15 dataset.
10 classes (Normal + 9 attack categories).
"""

import logging
import os
from typing import Optional

import numpy as np

logger = logging.getLogger(__name__)

try:
    import joblib
    _JOBLIB_AVAILABLE = True
except ImportError:
    _JOBLIB_AVAILABLE = False

try:
    from sklearn.ensemble import RandomForestClassifier
    from sklearn.metrics import classification_report, f1_score, roc_auc_score
    from sklearn.preprocessing import LabelEncoder
    _SKLEARN_AVAILABLE = True
except ImportError:
    _SKLEARN_AVAILABLE = False

# ---- UNSW-NB15 class definitions ----
ATTACK_CLASSES = [
    "Normal", "Fuzzers", "Analysis", "Backdoors",
    "DoS", "Exploits", "Generic", "Reconnaissance",
    "Shellcode", "Worms",
]

# ---- 49 UNSW-NB15 feature names (canonical order) ----
FEATURE_NAMES = [
    "dur", "proto", "service", "state", "spkts", "dpkts",
    "sbytes", "dbytes", "rate", "sttl", "dttl", "sload", "dload",
    "sloss", "dloss", "sinpkt", "dinpkt", "sjit", "djit",
    "swin", "stcpb", "dtcpb", "dwin", "tcprtt", "synack",
    "ackdat", "smean", "dmean", "trans_depth", "response_body_len",
    "ct_srv_src", "ct_state_ttl", "ct_dst_ltm", "ct_src_dport_ltm",
    "ct_dst_sport_ltm", "ct_dst_src_ltm", "is_ftp_login", "ct_ftp_cmd",
    "ct_flw_http_mthd", "ct_src_ltm", "ct_srv_dst", "is_sm_ips_ports",
    "attack_cat", "label", "srcip_num", "sport", "dstip_num", "dsport",
    "attack_num",
]

DEFAULT_PARAMS: dict = {
    "n_estimators": 200,
    "max_depth": 20,
    "min_samples_split": 5,
    "class_weight": "balanced",
    "n_jobs": -1,
    "random_state": 42,
    "verbose": 0,
}

MODEL_FILENAME = "unswnb15_rf.joblib"


class UNSWNB15RandomForest:
    """
    Multi-class network intrusion detector for UNSW-NB15.

    Feature input: 49-column numerical array (encode categoricals before passing).
    Output: one of 10 class labels.
    """

    def __init__(
        self,
        model_dir: Optional[str] = None,
        params: Optional[dict] = None,
    ) -> None:
        if not _SKLEARN_AVAILABLE:
            raise RuntimeError("scikit-learn not installed — pip install scikit-learn")
        self._model_dir = model_dir or os.environ.get("MODEL_DIR", "/tmp/kai/models")
        os.makedirs(self._model_dir, exist_ok=True)

        merged = dict(DEFAULT_PARAMS)
        if params:
            merged.update(params)
        self._params = merged

        self._clf: Optional[RandomForestClassifier] = None
        self._le = LabelEncoder()
        self._le.fit(ATTACK_CLASSES)

    # ------------------------------------------------------------------
    # Training
    # ------------------------------------------------------------------

    def train(self, X: np.ndarray, y: np.ndarray) -> None:
        """
        Fit the Random Forest.

        X: shape (n_samples, 49)  — numerical features
        y: shape (n_samples,)      — string class labels or int indices
        """
        y_enc = self._encode_labels(y)
        self._clf = RandomForestClassifier(**self._params)
        self._clf.fit(X, y_enc)
        logger.info(
            "UNSWNB15RandomForest trained — samples=%d features=%d classes=%d",
            X.shape[0],
            X.shape[1],
            len(self._clf.classes_),
        )

    # ------------------------------------------------------------------
    # Inference
    # ------------------------------------------------------------------

    def predict(self, X: np.ndarray) -> list:
        """Return list of string class labels."""
        self._check_fitted()
        encoded = self._clf.predict(X)
        return self._le.inverse_transform(encoded).tolist()

    def predict_proba(self, X: np.ndarray) -> np.ndarray:
        """Return probability matrix, shape (n_samples, 10)."""
        self._check_fitted()
        return self._clf.predict_proba(X)

    def feature_importances(self) -> dict:
        """Return feature name → importance mapping (sorted descending)."""
        self._check_fitted()
        importances = self._clf.feature_importances_
        pairs = sorted(
            zip(FEATURE_NAMES[: len(importances)], importances),
            key=lambda x: x[1],
            reverse=True,
        )
        return {name: float(imp) for name, imp in pairs}

    # ------------------------------------------------------------------
    # Feature metadata
    # ------------------------------------------------------------------

    def feature_names(self) -> list:
        """Return the list of 49 UNSW-NB15 feature names."""
        return list(FEATURE_NAMES)

    @property
    def classes(self) -> list:
        return list(ATTACK_CLASSES)

    # ------------------------------------------------------------------
    # Persistence
    # ------------------------------------------------------------------

    def save(self, path: Optional[str] = None) -> None:
        if not _JOBLIB_AVAILABLE:
            raise RuntimeError("joblib not installed — pip install joblib")
        self._check_fitted()
        if path is None:
            path = os.path.join(self._model_dir, MODEL_FILENAME)
        joblib.dump({"clf": self._clf, "le": self._le}, path)
        logger.info("UNSWNB15RandomForest saved — path=%s", path)

    def load(self, path: Optional[str] = None) -> None:
        if not _JOBLIB_AVAILABLE:
            raise RuntimeError("joblib not installed — pip install joblib")
        if path is None:
            path = os.path.join(self._model_dir, MODEL_FILENAME)
        bundle = joblib.load(path)
        self._clf = bundle["clf"]
        self._le = bundle["le"]
        logger.info("UNSWNB15RandomForest loaded — path=%s", path)

    # ------------------------------------------------------------------
    # Evaluation
    # ------------------------------------------------------------------

    def evaluate(self, X: np.ndarray, y: np.ndarray) -> dict:
        """Return per-class and macro-average metrics."""
        self._check_fitted()
        y_enc = self._encode_labels(y)
        y_pred_enc = self._clf.predict(X)
        y_proba = self._clf.predict_proba(X)

        report = classification_report(
            y_enc,
            y_pred_enc,
            target_names=self._le.classes_,
            output_dict=True,
            zero_division=0,
        )

        try:
            auc = float(roc_auc_score(y_enc, y_proba, multi_class="ovr", average="macro"))
        except Exception:
            auc = None

        return {
            "classification_report": report,
            "macro_f1": float(f1_score(y_enc, y_pred_enc, average="macro", zero_division=0)),
            "roc_auc_macro": auc,
        }

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _encode_labels(self, y):
        if y.dtype.kind in ("U", "S", "O"):  # string array
            return self._le.transform(y)
        return y  # assume already integer encoded

    def _check_fitted(self) -> None:
        if self._clf is None:
            raise RuntimeError("Model not trained — call train() or load() first")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    rng = np.random.default_rng(0)
    X = rng.random((300, 49)).astype(np.float32)
    y = rng.choice(ATTACK_CLASSES, size=300)
    clf = UNSWNB15RandomForest(params={"n_estimators": 10, "max_depth": 5})
    clf.train(X[:250], y[:250])
    preds = clf.predict(X[250:])
    print("Sample predictions:", preds[:5])
    print("Feature names count:", len(clf.feature_names()))
