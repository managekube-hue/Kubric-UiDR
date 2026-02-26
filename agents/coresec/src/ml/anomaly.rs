//! Anomaly scoring model — feedforward neural network in candle.
//!
//! The model architecture:
//!   Input (12 features) → Linear(12,64) → ReLU → Linear(64,32) → ReLU → Linear(32,1) → Sigmoid
//!
//! Features extracted from ProcessEvent:
//!   - cmdline_length (normalised)
//!   - path_depth (number of / or \ separators)
//!   - is_shell (1.0 if exe is bash/sh/powershell/cmd)
//!   - is_scripting (1.0 if exe is python/perl/ruby/node)
//!   - has_base64 (1.0 if cmdline contains base64-like patterns)
//!   - has_pipe (1.0 if cmdline contains |)
//!   - has_redirect (1.0 if cmdline contains > or >>)
//!   - has_network_util (1.0 if exe is curl/wget/nc/ncat)
//!   - parent_is_unusual (1.0 if ppid indicates unusual parent)
//!   - entropy (Shannon entropy of cmdline)
//!   - arg_count (number of space-separated arguments)
//!   - exe_in_tmp (1.0 if executable path contains /tmp or %TEMP%)

use candle_core::{DType, Device, Tensor};
use candle_nn::{linear, Linear, Module, VarBuilder};
use serde::{Deserialize, Serialize};
use std::path::Path;
use tracing::{info, warn};

/// Features extracted from a process event for scoring.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EventFeatures {
    pub cmdline_length: u32,
    pub path_depth: u32,
    pub is_shell: f32,
    pub is_scripting: f32,
    pub has_base64: f32,
    pub has_pipe: f32,
    pub has_redirect: f32,
    pub has_network_util: f32,
    pub parent_is_unusual: f32,
    pub entropy: f32,
    pub arg_count: u32,
    pub exe_in_tmp: f32,
}

impl EventFeatures {
    /// Convert to a 12-element f32 vector for model input.
    pub fn to_vec(&self) -> Vec<f32> {
        vec![
            self.cmdline_length as f32 / 1000.0, // normalise to [0,1]-ish
            self.path_depth as f32 / 20.0,
            self.is_shell,
            self.is_scripting,
            self.has_base64,
            self.has_pipe,
            self.has_redirect,
            self.has_network_util,
            self.parent_is_unusual,
            self.entropy / 8.0, // max entropy for ASCII
            self.arg_count as f32 / 50.0,
            self.exe_in_tmp,
        ]
    }
}

/// Extract features from raw process event fields.
pub fn extract_features(
    executable: &str,
    cmdline: &str,
    _ppid: u32,
) -> EventFeatures {
    let exe_lower = executable.to_lowercase();
    let cmd_lower = cmdline.to_lowercase();

    let shells = ["bash", "sh", "zsh", "fish", "powershell", "pwsh", "cmd.exe"];
    let scripting = ["python", "perl", "ruby", "node", "wscript", "cscript"];
    let net_utils = ["curl", "wget", "nc", "ncat", "nmap", "socat"];

    let path_sep = if executable.contains('\\') { '\\' } else { '/' };
    let path_depth = executable.chars().filter(|&c| c == path_sep).count() as u32;

    let is_shell = if shells.iter().any(|s| exe_lower.contains(s)) { 1.0 } else { 0.0 };
    let is_scripting = if scripting.iter().any(|s| exe_lower.contains(s)) { 1.0 } else { 0.0 };
    let has_network_util = if net_utils.iter().any(|s| exe_lower.contains(s)) { 1.0 } else { 0.0 };

    let has_base64 = if cmd_lower.contains("base64")
        || cmd_lower.contains("-enc ")
        || cmd_lower.contains("--encoded")
    { 1.0 } else { 0.0 };

    let has_pipe = if cmdline.contains('|') { 1.0 } else { 0.0 };
    let has_redirect = if cmdline.contains('>') { 1.0 } else { 0.0 };

    let exe_in_tmp = if exe_lower.contains("/tmp")
        || exe_lower.contains("\\temp")
        || exe_lower.contains("%temp%")
    { 1.0 } else { 0.0 };

    let entropy = shannon_entropy(cmdline);
    let arg_count = cmdline.split_whitespace().count() as u32;

    EventFeatures {
        cmdline_length: cmdline.len() as u32,
        path_depth,
        is_shell,
        is_scripting,
        has_base64,
        has_pipe,
        has_redirect,
        has_network_util,
        parent_is_unusual: 0.0, // requires parent process lookup
        entropy,
        arg_count,
        exe_in_tmp,
    }
}

fn shannon_entropy(s: &str) -> f32 {
    if s.is_empty() {
        return 0.0;
    }
    let mut freq = [0u32; 256];
    for &b in s.as_bytes() {
        freq[b as usize] += 1;
    }
    let len = s.len() as f32;
    let mut entropy: f32 = 0.0;
    for &count in &freq {
        if count > 0 {
            let p = count as f32 / len;
            entropy -= p * p.log2();
        }
    }
    entropy
}

/// Anomaly score result.
#[derive(Debug, Clone, Serialize)]
pub struct AnomalyResult {
    pub score: f32,
    pub is_anomalous: bool,
    pub threshold: f32,
}

/// Model for scoring process events.
///
/// If no model file is available, all events score 0.0 (benign).
pub struct AnomalyScorer {
    model: Option<ScorerModel>,
    threshold: f32,
}

struct ScorerModel {
    layer1: Linear,
    layer2: Linear,
    layer3: Linear,
}

impl ScorerModel {
    fn forward(&self, x: &Tensor) -> candle_core::Result<Tensor> {
        let h = self.layer1.forward(x)?.relu()?;
        let h = self.layer2.forward(&h)?.relu()?;
        let logit = self.layer3.forward(&h)?;
        // Sigmoid activation for [0,1] output
        let one = Tensor::ones_like(&logit)?;
        let neg = logit.neg()?;
        let exp_neg = neg.exp()?;
        let denom = (&one + &exp_neg)?;
        one.broadcast_div(&denom)
    }
}

impl AnomalyScorer {
    /// Load model from a safetensors file.  Falls back to no-op scorer if
    /// the file doesn't exist.
    pub fn new(model_path: &str) -> Self {
        let threshold = std::env::var("KUBRIC_ANOMALY_THRESHOLD")
            .ok()
            .and_then(|v| v.parse().ok())
            .unwrap_or(0.7);

        if !Path::new(model_path).exists() {
            info!(path = model_path, "anomaly model not found — scoring disabled");
            return Self { model: None, threshold };
        }

        match Self::load_model(model_path) {
            Ok(model) => {
                info!(path = model_path, "anomaly scoring model loaded");
                Self { model: Some(model), threshold }
            }
            Err(e) => {
                warn!(path = model_path, %e, "anomaly model load failed — scoring disabled");
                Self { model: None, threshold }
            }
        }
    }

    fn load_model(path: &str) -> candle_core::Result<ScorerModel> {
        let device = Device::Cpu;
        let vb = unsafe {
            VarBuilder::from_mmaped_safetensors(&[path], DType::F32, &device)?
        };

        let layer1 = linear(12, 64, vb.pp("layer1"))?;
        let layer2 = linear(64, 32, vb.pp("layer2"))?;
        let layer3 = linear(32, 1, vb.pp("layer3"))?;

        Ok(ScorerModel { layer1, layer2, layer3 })
    }

    /// Create a no-op scorer (always returns 0.0).
    pub fn empty() -> Self {
        Self { model: None, threshold: 0.7 }
    }

    /// Score a set of event features.
    pub fn score(&self, features: &EventFeatures) -> AnomalyResult {
        let model = match &self.model {
            Some(m) => m,
            None => return AnomalyResult { score: 0.0, is_anomalous: false, threshold: self.threshold },
        };

        let input_vec = features.to_vec();
        let result = (|| -> candle_core::Result<f32> {
            let tensor = Tensor::from_vec(input_vec, (1, 12), &Device::Cpu)?;
            let output = model.forward(&tensor)?;
            let score = output.flatten_all()?.to_vec1::<f32>()?;
            Ok(score.first().copied().unwrap_or(0.0))
        })();

        match result {
            Ok(score) => AnomalyResult {
                score,
                is_anomalous: score >= self.threshold,
                threshold: self.threshold,
            },
            Err(e) => {
                warn!(%e, "anomaly inference failed");
                AnomalyResult { score: 0.0, is_anomalous: false, threshold: self.threshold }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_features_basic() {
        let f = extract_features("/usr/bin/bash", "bash -c 'echo hello | base64'", 1);
        assert_eq!(f.is_shell, 1.0);
        assert_eq!(f.has_pipe, 1.0);
        assert_eq!(f.has_base64, 1.0);
        assert!(f.cmdline_length > 0);
        assert!(f.entropy > 0.0);
    }

    #[test]
    fn extract_features_benign() {
        let f = extract_features("/usr/bin/ls", "ls -la /home", 1);
        assert_eq!(f.is_shell, 0.0);
        assert_eq!(f.is_scripting, 0.0);
        assert_eq!(f.has_base64, 0.0);
        assert_eq!(f.has_pipe, 0.0);
    }

    #[test]
    fn extract_features_windows() {
        let f = extract_features(
            "C:\\Windows\\System32\\cmd.exe",
            "cmd.exe /c powershell -enc SGVsbG8=",
            4,
        );
        assert_eq!(f.is_shell, 1.0);
        assert_eq!(f.has_base64, 1.0);
        assert!(f.path_depth >= 3);
    }

    #[test]
    fn shannon_entropy_empty() {
        assert_eq!(shannon_entropy(""), 0.0);
    }

    #[test]
    fn shannon_entropy_uniform() {
        // "aaaa" has zero entropy (only one symbol)
        assert_eq!(shannon_entropy("aaaa"), 0.0);
    }

    #[test]
    fn shannon_entropy_mixed() {
        let e = shannon_entropy("abcd");
        // 4 equally-distributed symbols → entropy = 2.0 bits
        assert!((e - 2.0).abs() < 0.01);
    }

    #[test]
    fn features_to_vec_length() {
        let f = EventFeatures::default();
        assert_eq!(f.to_vec().len(), 12);
    }

    #[test]
    fn scorer_empty_always_benign() {
        let scorer = AnomalyScorer::empty();
        let result = scorer.score(&EventFeatures::default());
        assert_eq!(result.score, 0.0);
        assert!(!result.is_anomalous);
    }
}
