"""
K-KAI-ML-002_clearml_experiment.py
ClearML experiment tracker for KAI model training runs.
"""

import logging
import os
from typing import Optional

logger = logging.getLogger(__name__)

# ClearML is optional — import guard keeps the module usable in CI without it.
try:
    from clearml import Task  # type: ignore
    _CLEARML_AVAILABLE = True
except ImportError:
    Task = None  # type: ignore
    _CLEARML_AVAILABLE = False


def _init_clearml() -> None:
    """Configure ClearML SDK from environment variables once."""
    if not _CLEARML_AVAILABLE:
        return
    from clearml.backend_api import Session  # type: ignore

    host = os.environ.get("CLEARML_API_HOST", "")
    access = os.environ.get("CLEARML_API_ACCESS_KEY", "")
    secret = os.environ.get("CLEARML_API_SECRET_KEY", "")
    if host and access and secret:
        Session.add_client(host, access, secret)
        logger.debug("ClearML SDK configured — host=%s", host)


_init_clearml()


class ClearMLExperiment:
    """
    Thin wrapper around a ClearML Task.

    Usage::

        exp = ClearMLExperiment()
        task = exp.init("kubric-kai", "ember-xgboost-v3")
        exp.log_params({"n_estimators": 500, "max_depth": 8})
        exp.log_metric("accuracy", 0.964, iteration=1)
        exp.upload_artifact("model", "/tmp/ember.json")
        exp.close()
    """

    def __init__(self) -> None:
        self._task: Optional[object] = None

    def init(
        self,
        project: str,
        task_name: str,
        task_type: str = "training",
    ) -> object:
        """Create or reuse a ClearML Task."""
        if not _CLEARML_AVAILABLE:
            logger.warning("clearml not installed — experiment tracking disabled")
            return None

        type_map = {
            "training": Task.TaskTypes.training,
            "testing": Task.TaskTypes.testing,
            "inference": Task.TaskTypes.inference,
            "data_processing": Task.TaskTypes.data_processing,
        }
        clearml_type = type_map.get(task_type, Task.TaskTypes.training)

        self._task = Task.init(
            project_name=project,
            task_name=task_name,
            task_type=clearml_type,
            reuse_last_task_id=False,
        )
        logger.info("ClearML task initialised — project=%s name=%s", project, task_name)
        return self._task

    def log_params(self, params: dict) -> None:
        """Connect a dict of hyperparameters to the task."""
        if self._task is None:
            return
        self._task.connect(params)

    def log_metric(self, series: str, value: float, iteration: int) -> None:
        """Report a scalar metric value at a given iteration."""
        if self._task is None:
            return
        logger_obj = self._task.get_logger()
        logger_obj.report_scalar(
            title=series,
            series=series,
            value=value,
            iteration=iteration,
        )

    def upload_artifact(self, name: str, path: str) -> None:
        """Upload a file artifact to ClearML."""
        if self._task is None:
            return
        self._task.upload_artifact(name=name, artifact_object=path)
        logger.info("Artifact uploaded — name=%s path=%s", name, path)

    def close(self) -> None:
        """Mark the task as completed."""
        if self._task is None:
            return
        self._task.close()
        logger.info("ClearML task closed")


# ---------------------------------------------------------------------------
# KAI Experiment Factory
# ---------------------------------------------------------------------------

class KAIExperimentFactory:
    """
    Pre-built experiment configs for the three core KAI models.

    Each config returns a dict ready to pass into ClearMLExperiment.log_params().
    """

    EMBER_XGBOOST = {
        "model": "ember_xgboost",
        "n_estimators": 500,
        "max_depth": 8,
        "learning_rate": 0.1,
        "subsample": 0.8,
        "colsample_bytree": 0.8,
        "eval_metric": "logloss",
        "n_features": 2381,
        "dataset": "EMBER 2018",
    }

    UNSWNB15_RF = {
        "model": "unswnb15_random_forest",
        "n_estimators": 200,
        "max_depth": 20,
        "min_samples_split": 5,
        "class_weight": "balanced",
        "n_classes": 10,
        "dataset": "UNSW-NB15",
    }

    MORDOR_LSTM = {
        "model": "mordor_lstm",
        "input_size": 32,
        "hidden_size": 128,
        "num_layers": 2,
        "dropout": 0.2,
        "dense_units": [64, 32, 1],
        "output_activation": "sigmoid",
        "epochs": 50,
        "batch_size": 256,
        "dataset": "Mordor / OTRF",
    }

    @classmethod
    def create(
        cls,
        model_key: str,
        project: str = "kubric-kai",
        overrides: Optional[dict] = None,
    ) -> dict:
        """
        Returns (experiment_config, params_dict) for the given model key.

        model_key: "ember_xgboost" | "unswnb15_rf" | "mordor_lstm"
        """
        base_configs = {
            "ember_xgboost": cls.EMBER_XGBOOST,
            "unswnb15_rf": cls.UNSWNB15_RF,
            "mordor_lstm": cls.MORDOR_LSTM,
        }
        if model_key not in base_configs:
            raise ValueError(f"Unknown model key: {model_key!r}")

        params = dict(base_configs[model_key])
        if overrides:
            params.update(overrides)

        task_name = f"{model_key}-{params.get('dataset', 'unknown')}"
        return {
            "project": project,
            "task_name": task_name,
            "task_type": "training",
            "params": params,
        }

    @classmethod
    def launch(
        cls,
        model_key: str,
        project: str = "kubric-kai",
        overrides: Optional[dict] = None,
    ) -> "ClearMLExperiment":
        """Create and initialise a ClearMLExperiment for the given model."""
        config = cls.create(model_key, project, overrides)
        exp = ClearMLExperiment()
        exp.init(
            project=config["project"],
            task_name=config["task_name"],
            task_type=config["task_type"],
        )
        exp.log_params(config["params"])
        return exp


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    for key in ("ember_xgboost", "unswnb15_rf", "mordor_lstm"):
        cfg = KAIExperimentFactory.create(key)
        print(f"Config for {key}: {cfg['task_name']}")
