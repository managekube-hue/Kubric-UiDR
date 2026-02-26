//! ML inference module — on-device anomaly scoring using HuggingFace candle.
//!
//! Runs a small feedforward neural network (loaded from safetensors) to score
//! process events for anomalousness.  The model is trained offline on
//! normal process behaviour; high scores indicate potential threats.
//!
//! Model files are expected at:
//!   `models/anomaly_scorer.safetensors`  — weights
//!   `models/anomaly_config.json`         — feature config (input dim, thresholds)
//!
//! If no model files are present, scoring returns 0.0 (benign) for all events.
#![allow(dead_code)]

pub mod anomaly;

pub use anomaly::{AnomalyScorer, AnomalyResult};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scorer_without_model_returns_zero() {
        let scorer = AnomalyScorer::new("models/nonexistent.safetensors");
        let result = scorer.score(&anomaly::EventFeatures::default());
        assert_eq!(result.score, 0.0);
        assert!(!result.is_anomalous);
    }

    #[test]
    fn event_features_default_is_benign() {
        let features = anomaly::EventFeatures::default();
        assert_eq!(features.cmdline_length, 0);
        assert_eq!(features.path_depth, 0);
        assert_eq!(features.is_shell, 0.0);
    }
}
