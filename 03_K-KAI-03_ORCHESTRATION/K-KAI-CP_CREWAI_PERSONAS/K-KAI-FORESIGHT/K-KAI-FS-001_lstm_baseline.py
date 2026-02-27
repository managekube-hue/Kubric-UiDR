"""
K-KAI Foresight: LSTM Threat Baseline
Trains and serves an LSTM model that predicts future alert volume
and anomaly probability from historical OCSF event time-series.
"""
from __future__ import annotations
import os, json, logging, math
from datetime import datetime, timedelta, timezone
from typing import List, Dict, Optional, Tuple
import numpy as np
import torch
import torch.nn as nn
from torch.utils.data import Dataset, DataLoader

logger = logging.getLogger(__name__)

# ── hyper-params ─────────────────────────────────────────────────
SEQ_LEN    = 24          # hours of history
HIDDEN     = 128
LAYERS     = 2
DROPOUT    = 0.2
LR         = 1e-3
EPOCHS     = 50
BATCH      = 32
THRESHOLD  = 0.65        # anomaly probability cut-off

# ── dataset ──────────────────────────────────────────────────────
class AlertSeriesDataset(Dataset):
    def __init__(self, series: np.ndarray, seq_len: int = SEQ_LEN):
        self.series  = torch.tensor(series, dtype=torch.float32)
        self.seq_len = seq_len

    def __len__(self) -> int:
        return max(0, len(self.series) - self.seq_len)

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, torch.Tensor]:
        x = self.series[idx : idx + self.seq_len]
        y = self.series[idx + self.seq_len]
        return x, y

# ── model ─────────────────────────────────────────────────────────
class ThreatLSTM(nn.Module):
    def __init__(self, input_size: int = 1, hidden: int = HIDDEN,
                 layers: int = LAYERS, dropout: float = DROPOUT):
        super().__init__()
        self.lstm = nn.LSTM(input_size, hidden, layers,
                            batch_first=True, dropout=dropout)
        self.fc   = nn.Linear(hidden, 1)

    def forward(self, x: torch.Tensor) -> torch.Tensor:        # (B, T, 1)
        out, _ = self.lstm(x)
        return self.fc(out[:, -1, :]).squeeze(-1)              # (B,)

# ── trainer ──────────────────────────────────────────────────────
class LSTMTrainer:
    def __init__(self, model_path: str = "kai_lstm.pt"):
        self.model_path = model_path
        self.model: Optional[ThreatLSTM] = None
        self.scaler_mean: float = 0.0
        self.scaler_std:  float = 1.0

    # ---------- preprocessing -----------
    def _normalise(self, series: np.ndarray) -> np.ndarray:
        self.scaler_mean = float(series.mean())
        self.scaler_std  = float(series.std()) or 1.0
        return (series - self.scaler_mean) / self.scaler_std

    def _denormalise(self, val: float) -> float:
        return val * self.scaler_std + self.scaler_mean

    # ---------- training -----------------
    def train(self, hourly_counts: List[float]) -> Dict:
        series = np.array(hourly_counts, dtype=np.float32)
        norm   = self._normalise(series).reshape(-1, 1)

        dataset    = AlertSeriesDataset(norm)
        loader     = DataLoader(dataset, batch_size=BATCH, shuffle=True)
        self.model = ThreatLSTM()
        optim      = torch.optim.Adam(self.model.parameters(), lr=LR)
        criterion  = nn.MSELoss()

        losses: List[float] = []
        for epoch in range(EPOCHS):
            epoch_loss = 0.0
            for x, y in loader:
                x = x.unsqueeze(-1) if x.dim() == 2 else x
                optim.zero_grad()
                pred = self.model(x)
                loss = criterion(pred, y.squeeze(-1))
                loss.backward()
                optim.step()
                epoch_loss += loss.item()
            avg = epoch_loss / max(len(loader), 1)
            losses.append(avg)
            if epoch % 10 == 0:
                logger.info("epoch %d/%d  loss=%.6f", epoch+1, EPOCHS, avg)

        torch.save({
            "state":        self.model.state_dict(),
            "scaler_mean":  self.scaler_mean,
            "scaler_std":   self.scaler_std,
        }, self.model_path)
        return {"epochs": EPOCHS, "final_loss": losses[-1], "model_path": self.model_path}

    # ---------- inference ----------------
    def load(self) -> None:
        ckpt = torch.load(self.model_path, map_location="cpu")
        self.scaler_mean = ckpt["scaler_mean"]
        self.scaler_std  = ckpt["scaler_std"]
        self.model = ThreatLSTM()
        self.model.load_state_dict(ckpt["state"])
        self.model.eval()

    def predict_next(self, recent_24h: List[float]) -> Dict:
        if self.model is None:
            self.load()
        norm = (np.array(recent_24h, dtype=np.float32) - self.scaler_mean) / self.scaler_std
        x    = torch.tensor(norm, dtype=torch.float32).unsqueeze(0).unsqueeze(-1)  # (1,24,1)
        with torch.no_grad():
            pred_norm = self.model(x).item()
        pred_raw   = self._denormalise(pred_norm)
        anomaly    = pred_raw > (self.scaler_mean + 2 * self.scaler_std)
        return {
            "predicted_count": max(0.0, round(pred_raw, 2)),
            "anomaly":         anomaly,
            "threshold":       round(self.scaler_mean + 2 * self.scaler_std, 2),
            "ts":              datetime.now(timezone.utc).isoformat(),
        }

# ── entrypoint ────────────────────────────────────────────────────
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    # demo: generate synthetic sine-wave data with noise
    rng  = np.random.default_rng(42)
    t    = np.linspace(0, 8*math.pi, 500)
    data = (50 + 20*np.sin(t) + rng.normal(0, 5, 500)).clip(0).tolist()

    trainer = LSTMTrainer()
    result  = trainer.train(data)
    print(json.dumps(result, indent=2))

    forecast = trainer.predict_next(data[-SEQ_LEN:])
    print(json.dumps(forecast, indent=2))
