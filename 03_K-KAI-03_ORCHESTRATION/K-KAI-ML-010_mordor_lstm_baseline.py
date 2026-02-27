"""
K-KAI-ML-010_mordor_lstm_baseline.py
LSTM seq2seq model for Mordor dataset behavioral baselining (APT simulation data).
Uses PyTorch. Architecture: 2-layer LSTM → Dense(64) → Dense(32) → Dense(1, sigmoid).
"""

import hashlib
import logging
import os
from typing import Optional

import numpy as np

logger = logging.getLogger(__name__)

try:
    import torch
    import torch.nn as nn
    from torch.utils.data import DataLoader, TensorDataset
    _TORCH_AVAILABLE = True
except ImportError:
    _TORCH_AVAILABLE = False

MAX_CLASS_UID = 10000
MAX_ACTIVITY_ID = 100
MAX_SEVERITY_ID = 10
IP_HASH_BUCKETS = 1000
EMBED_DIM = 8
INPUT_DIM = EMBED_DIM * 4  # 4 fields x 8 dims = 32


def _hash_ip(ip: str, buckets: int = IP_HASH_BUCKETS) -> int:
    h = int(hashlib.md5(ip.encode()).hexdigest(), 16)
    return h % buckets


def _tokenize_event(event: dict) -> list:
    class_uid = int(event.get("class_uid", 0)) % MAX_CLASS_UID
    activity_id = int(event.get("activity_id", 0)) % MAX_ACTIVITY_ID
    severity_id = int(event.get("severity_id", 0)) % MAX_SEVERITY_ID
    ip_idx = _hash_ip(str(event.get("src_ip", "0.0.0.0")))
    return [class_uid, activity_id, severity_id, ip_idx]


class _LSTMModel(nn.Module):
    def __init__(self, input_size: int, hidden_size: int, num_layers: int, dropout: float):
        super().__init__()
        self.lstm = nn.LSTM(
            input_size=input_size,
            hidden_size=hidden_size,
            num_layers=num_layers,
            dropout=dropout,
            batch_first=True,
        )
        self.dense = nn.Sequential(
            nn.Linear(hidden_size, 64),
            nn.ReLU(),
            nn.Linear(64, 32),
            nn.ReLU(),
            nn.Linear(32, 1),
            nn.Sigmoid(),
        )

    def forward(self, x):
        out, _ = self.lstm(x)
        return self.dense(out[:, -1, :]).squeeze(-1)


class MordorLSTMBaseline:
    """
    LSTM anomaly detector for Mordor APT simulation event sequences.
    Predicts anomaly score in [0, 1] for a window of OCSF events.
    """

    def __init__(
        self,
        hidden_size: int = 128,
        num_layers: int = 2,
        dropout: float = 0.2,
        model_dir: Optional[str] = None,
        device: Optional[str] = None,
    ) -> None:
        if not _TORCH_AVAILABLE:
            raise RuntimeError("torch not installed — pip install torch")
        self._hidden_size = hidden_size
        self._num_layers = num_layers
        self._dropout = dropout
        self._model_dir = model_dir or os.environ.get("MODEL_DIR", "/tmp/kai/models")
        os.makedirs(self._model_dir, exist_ok=True)
        self._device = torch.device(device or ("cuda" if torch.cuda.is_available() else "cpu"))
        self._model: Optional[_LSTMModel] = None
        self._emb_class = nn.Embedding(MAX_CLASS_UID, EMBED_DIM)
        self._emb_activity = nn.Embedding(MAX_ACTIVITY_ID, EMBED_DIM)
        self._emb_severity = nn.Embedding(MAX_SEVERITY_ID, EMBED_DIM)
        self._emb_ip = nn.Embedding(IP_HASH_BUCKETS, EMBED_DIM)

    def _build_model(self) -> _LSTMModel:
        return _LSTMModel(
            input_size=INPUT_DIM,
            hidden_size=self._hidden_size,
            num_layers=self._num_layers,
            dropout=self._dropout if self._num_layers > 1 else 0.0,
        ).to(self._device)

    def preprocess(self, raw_events: list) -> "torch.Tensor":
        """Convert list of event dicts to float tensor shape (1, seq_len, 32)."""
        tokens_list = [_tokenize_event(e) for e in raw_events]
        tokens = torch.tensor(tokens_list, dtype=torch.long)
        emb = torch.cat([
            self._emb_class(tokens[:, 0]),
            self._emb_activity(tokens[:, 1]),
            self._emb_severity(tokens[:, 2]),
            self._emb_ip(tokens[:, 3]),
        ], dim=-1)
        return emb.unsqueeze(0)  # (1, seq_len, 32)

    def train(
        self,
        events: list,
        labels: Optional[list] = None,
        epochs: int = 50,
        batch_size: int = 256,
        seq_len: int = 64,
        learning_rate: float = 1e-3,
    ) -> dict:
        """Train on a flat list of OCSF event dicts using sliding windows."""
        self._model = self._build_model()
        optimizer = torch.optim.Adam(self._model.parameters(), lr=learning_rate)
        criterion = nn.BCELoss()
        X, y = self._build_sequences(events, labels, seq_len)
        if len(X) == 0:
            logger.warning("No sequences built — too few events")
            return {"loss_history": []}
        loader = DataLoader(TensorDataset(X, y), batch_size=batch_size, shuffle=True)
        history = []
        self._model.train()
        for epoch in range(1, epochs + 1):
            epoch_loss = 0.0
            for X_b, y_b in loader:
                X_b, y_b = X_b.to(self._device), y_b.to(self._device)
                optimizer.zero_grad()
                loss = criterion(self._model(X_b), y_b)
                loss.backward()
                optimizer.step()
                epoch_loss += loss.item() * len(X_b)
            avg = epoch_loss / len(X)
            history.append(avg)
            if epoch % 10 == 0:
                logger.info("Epoch %d/%d — loss=%.6f", epoch, epochs, avg)
        return {"loss_history": history}

    def predict_anomaly(self, event_window: list) -> float:
        """Return anomaly score [0, 1] for a list of events."""
        if self._model is None:
            raise RuntimeError("Model not initialised — call train() or load()")
        self._model.eval()
        with torch.no_grad():
            t = self.preprocess(event_window).to(self._device)
            return float(self._model(t).item())

    def save(self, path: Optional[str] = None) -> None:
        if self._model is None:
            raise RuntimeError("No model to save")
        if path is None:
            path = os.path.join(self._model_dir, "mordor_lstm.pt")
        torch.save({
            "model_state": self._model.state_dict(),
            "emb_class": self._emb_class.state_dict(),
            "emb_activity": self._emb_activity.state_dict(),
            "emb_severity": self._emb_severity.state_dict(),
            "emb_ip": self._emb_ip.state_dict(),
            "config": {"hidden_size": self._hidden_size, "num_layers": self._num_layers, "dropout": self._dropout},
        }, path)
        logger.info("MordorLSTM saved — %s", path)

    def load(self, path: Optional[str] = None) -> None:
        if path is None:
            path = os.path.join(self._model_dir, "mordor_lstm.pt")
        bundle = torch.load(path, map_location=self._device)
        cfg = bundle["config"]
        self._hidden_size, self._num_layers, self._dropout = cfg["hidden_size"], cfg["num_layers"], cfg["dropout"]
        self._model = self._build_model()
        self._model.load_state_dict(bundle["model_state"])
        self._emb_class.load_state_dict(bundle["emb_class"])
        self._emb_activity.load_state_dict(bundle["emb_activity"])
        self._emb_severity.load_state_dict(bundle["emb_severity"])
        self._emb_ip.load_state_dict(bundle["emb_ip"])
        logger.info("MordorLSTM loaded — %s", path)

    def _build_sequences(self, events, labels, seq_len):
        if len(events) < seq_len + 1:
            return torch.zeros(0), torch.zeros(0)
        seqs, seq_labels = [], []
        use_labels = labels is not None and len(labels) == len(events)
        for i in range(0, len(events) - seq_len, max(1, seq_len // 2)):
            window = events[i: i + seq_len]
            seqs.append(self.preprocess(window).squeeze(0))
            seq_labels.append(float(any(labels[i: i + seq_len])) if use_labels else 0.0)
        return torch.stack(seqs).float(), torch.tensor(seq_labels, dtype=torch.float32)
