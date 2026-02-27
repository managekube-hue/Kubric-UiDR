"""
K-KAI-ML-001_tensorboard_logger.py
TensorBoard metric logger for KAI ML experiments.
"""

import logging
import os
from datetime import datetime
from typing import Union

import numpy as np

try:
    from torch.utils.tensorboard import SummaryWriter
except ImportError:
    from tensorboardX import SummaryWriter  # type: ignore

logger = logging.getLogger(__name__)


class TensorBoardLogger:
    """Wraps TensorBoard SummaryWriter for KAI ML experiment logging."""

    def __init__(self, log_dir: str = "runs/kai") -> None:
        timestamp = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
        self.log_dir = os.path.join(log_dir, timestamp)
        os.makedirs(self.log_dir, exist_ok=True)
        self._writer = SummaryWriter(log_dir=self.log_dir)
        logger.info("TensorBoardLogger initialised — log_dir=%s", self.log_dir)

    def log_scalar(self, tag: str, value: float, step: int) -> None:
        """Log a single scalar value."""
        self._writer.add_scalar(tag, value, global_step=step)

    def log_scalars(self, tag: str, values: dict, step: int) -> None:
        """Log multiple scalars under one tag group (e.g. train/val split)."""
        self._writer.add_scalars(tag, values, global_step=step)

    def log_histogram(self, tag: str, values: Union[list, np.ndarray], step: int) -> None:
        """Log a histogram of values (weights, activations, etc.)."""
        if not isinstance(values, np.ndarray):
            values = np.array(values, dtype=np.float32)
        self._writer.add_histogram(tag, values, global_step=step)

    def log_text(self, tag: str, text: str, step: int) -> None:
        """Log a text string (e.g. prediction explanation)."""
        self._writer.add_text(tag, text, global_step=step)

    def flush(self) -> None:
        """Flush pending writes to disk."""
        self._writer.flush()

    def close(self) -> None:
        """Close the underlying SummaryWriter."""
        self._writer.close()
        logger.info("TensorBoardLogger closed — log_dir=%s", self.log_dir)


class KAIMetricsLogger:
    """
    High-level wrapper that logs the standard KAI training metric suite
    (loss, accuracy, f1_score, false_positive_rate) per epoch.
    """

    METRICS = ("loss", "accuracy", "f1_score", "false_positive_rate")

    def __init__(
        self, log_dir: str = "runs/kai", experiment_name: str = "kai_experiment"
    ) -> None:
        run_log_dir = os.path.join(log_dir, experiment_name)
        self._tb = TensorBoardLogger(log_dir=run_log_dir)
        self.experiment_name = experiment_name
        self._epoch_history: list = []

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def log_epoch(
        self,
        epoch: int,
        loss: float,
        accuracy: float,
        f1_score: float,
        false_positive_rate: float,
        split: str = "train",
    ) -> None:
        """Logs all standard KAI metrics for one epoch."""
        values = {
            "loss": loss,
            "accuracy": accuracy,
            "f1_score": f1_score,
            "false_positive_rate": false_positive_rate,
        }
        for metric_name, value in values.items():
            tag = f"{split}/{metric_name}"
            self._tb.log_scalar(tag, value, step=epoch)

        self._tb.log_scalars("comparison/loss", {split: loss}, step=epoch)
        self._tb.log_scalars("comparison/accuracy", {split: accuracy}, step=epoch)

        record = {"epoch": epoch, "split": split, **values}
        self._epoch_history.append(record)
        logger.debug("Epoch %d [%s] — %s", epoch, split, values)

    def log_hyperparams(self, params: dict, final_metrics: dict) -> None:
        """Log hyperparameter config alongside final metric values."""
        try:
            self._tb._writer.add_hparams(params, final_metrics)
        except Exception as exc:  # noqa: BLE001
            logger.warning("add_hparams failed: %s", exc)

    def log_confusion_matrix_text(self, matrix_text: str, epoch: int) -> None:
        """Log a pre-formatted confusion matrix as TensorBoard text."""
        self._tb.log_text("confusion_matrix", matrix_text, step=epoch)

    def log_weight_histogram(
        self, layer_name: str, weights: np.ndarray, epoch: int
    ) -> None:
        """Log weight distribution histogram for a named layer."""
        self._tb.log_histogram(f"weights/{layer_name}", weights, step=epoch)

    @property
    def history(self) -> list:
        return list(self._epoch_history)

    def flush(self) -> None:
        self._tb.flush()

    def close(self) -> None:
        self._tb.close()

    def __enter__(self) -> "KAIMetricsLogger":
        return self

    def __exit__(self, *_) -> None:
        self.close()


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    with KAIMetricsLogger(log_dir="/tmp/runs/kai", experiment_name="smoke_test") as ml:
        for ep in range(1, 6):
            ml.log_epoch(
                epoch=ep,
                loss=1.0 / ep,
                accuracy=0.5 + ep * 0.08,
                f1_score=0.45 + ep * 0.09,
                false_positive_rate=0.2 - ep * 0.03,
                split="train",
            )
            ml.log_epoch(
                epoch=ep,
                loss=1.2 / ep,
                accuracy=0.48 + ep * 0.07,
                f1_score=0.42 + ep * 0.085,
                false_positive_rate=0.22 - ep * 0.025,
                split="val",
            )
        ml.flush()
    print("Smoke test complete — open TensorBoard at /tmp/runs/kai")
