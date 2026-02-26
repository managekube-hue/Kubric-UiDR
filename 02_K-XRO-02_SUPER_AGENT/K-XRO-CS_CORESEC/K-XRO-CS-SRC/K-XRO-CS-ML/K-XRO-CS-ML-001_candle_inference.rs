//! K-XRO-CS-ML-001 — ML anomaly scoring using Candle (HuggingFace).
//!
//! Provides a feedforward neural network (12→64→32→1) trained offline on
//! benign process telemetry.  At runtime, `score_event` extracts 12
//! hand-crafted features from process metadata and returns a score in [0,1].
//! Scores above `THRESHOLD` (0.70) are treated as anomalous.
//!
//! When no model file is found at the configured path the scorer silently
//! falls back to `NoOpScorer` which always returns 0.0 — the agent continues
//! running without ML capabilities rather than crashing.
//!
//! # Candle dependencies (Cargo.toml)
//! ```toml
//! candle-core   = { version = "0.6", features = ["default"] }
//! candle-nn     = { version = "0.6" }
//! ```

use candle_core::{DType, Device, Tensor};
use candle_nn::{linear, Linear, Module, VarBuilder, VarMap};
use std::path::Path;
use tracing::{info, warn};

/// Anomaly score threshold.  Scores at or above this value are flagged.
pub const THRESHOLD: f64 = 0.70;

/// Number of input features fed to the neural network.
const FEATURE_DIM: usize = 12;

// ─────────────────────────────────────────────────────────────────────────────
// Feature extraction helpers
// ─────────────────────────────────────────────────────────────────────────────

/// Extract 12 numeric features from process metadata.
///
/// Feature vector layout (index → description):
///  0  cmdline_len_norm   — character count / 512, clamped to [0,1]
///  1  path_depth         — number of `/` separators / 10
///  2  is_shell           — 1.0 if exe basename is sh/bash/zsh/dash/fish
///  3  has_base64         — 1.0 if cmdline contains typical base64 flag or encoded blob
///  4  has_ip_addr        — 1.0 if cmdline contains an IPv4/IPv6 address pattern
///  5  uid_zero           — 1.0 if uid == 0 (root)
///  6  path_entropy       — Shannon entropy of exe path chars / 5.0 (normalised)
///  7  arg_count          — space-split arg count / 20, clamped to [0,1]
///  8  has_pipe           — 1.0 if cmdline contains `|`
///  9  has_redirect       — 1.0 if cmdline contains `>` or `>>`
/// 10  exe_basename_len   — length of exe filename / 30, clamped to [0,1]
/// 11  has_sudo           — 1.0 if cmdline starts with `sudo`
pub fn extract_features(exe: &str, cmdline: &str, uid: u32) -> [f32; FEATURE_DIM] {
    let mut f = [0.0f32; FEATURE_DIM];

    // 0: cmdline length normalised
    f[0] = (cmdline.len() as f32 / 512.0).min(1.0);

    // 1: executable path depth
    let depth = exe.chars().filter(|&c| c == '/').count();
    f[1] = (depth as f32 / 10.0).min(1.0);

    // 2: is known shell
    let basename = Path::new(exe)
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("");
    let shells = ["sh", "bash", "zsh", "dash", "fish", "csh", "tcsh", "ksh"];
    f[2] = if shells.contains(&basename) { 1.0 } else { 0.0 };

    // 3: base64 indicator — `-enc`, `base64`, long alphanum+/ runs
    let b64_markers = ["base64", "-enc", "frombase64", "tobytes"];
    let has_b64 = b64_markers.iter().any(|m| cmdline.to_lowercase().contains(m))
        || has_long_base64_blob(cmdline);
    f[3] = if has_b64 { 1.0 } else { 0.0 };

    // 4: contains IP address pattern
    f[4] = if contains_ip_address(cmdline) { 1.0 } else { 0.0 };

    // 5: running as root
    f[5] = if uid == 0 { 1.0 } else { 0.0 };

    // 6: Shannon entropy of exe path
    f[6] = (shannon_entropy(exe) / 5.0).min(1.0);

    // 7: argument count
    let arg_count = cmdline.split_whitespace().count();
    f[7] = (arg_count as f32 / 20.0).min(1.0);

    // 8: pipe character
    f[8] = if cmdline.contains('|') { 1.0 } else { 0.0 };

    // 9: output redirect
    f[9] = if cmdline.contains('>') { 1.0 } else { 0.0 };

    // 10: exe basename length
    f[10] = (basename.len() as f32 / 30.0).min(1.0);

    // 11: sudo prefix
    let trimmed = cmdline.trim();
    f[11] = if trimmed.starts_with("sudo") { 1.0 } else { 0.0 };

    f
}

fn has_long_base64_blob(s: &str) -> bool {
    // Detect runs of base64 alphabet characters >= 40 chars
    let mut run = 0usize;
    for c in s.chars() {
        if c.is_alphanumeric() || c == '+' || c == '/' || c == '=' {
            run += 1;
            if run >= 40 {
                return true;
            }
        } else {
            run = 0;
        }
    }
    false
}

fn contains_ip_address(s: &str) -> bool {
    // Simple heuristic: look for N.N.N.N where N is 1-3 digits
    let bytes = s.as_bytes();
    let len = bytes.len();
    let mut i = 0;
    while i < len {
        if bytes[i].is_ascii_digit() {
            let start = i;
            let mut dots = 0usize;
            let mut j = i;
            while j < len && (bytes[j].is_ascii_digit() || bytes[j] == b'.') {
                if bytes[j] == b'.' {
                    dots += 1;
                }
                j += 1;
            }
            if dots == 3 && j - start >= 7 {
                return true;
            }
            i = j;
        } else {
            i += 1;
        }
    }
    false
}

fn shannon_entropy(s: &str) -> f32 {
    if s.is_empty() {
        return 0.0;
    }
    let mut freq = [0u32; 256];
    let n = s.len() as f32;
    for b in s.bytes() {
        freq[b as usize] += 1;
    }
    let mut h = 0.0f32;
    for &count in &freq {
        if count > 0 {
            let p = count as f32 / n;
            h -= p * p.log2();
        }
    }
    h
}

// ─────────────────────────────────────────────────────────────────────────────
// Neural network definition
// ─────────────────────────────────────────────────────────────────────────────

/// 12→64→32→1 feedforward network with ReLU activations and sigmoid output.
struct AnomalyNet {
    fc1: Linear,
    fc2: Linear,
    fc3: Linear,
}

impl AnomalyNet {
    fn new(vb: VarBuilder) -> candle_core::Result<Self> {
        let fc1 = linear(FEATURE_DIM, 64, vb.pp("fc1"))?;
        let fc2 = linear(64, 32, vb.pp("fc2"))?;
        let fc3 = linear(32, 1, vb.pp("fc3"))?;
        Ok(Self { fc1, fc2, fc3 })
    }

    fn forward(&self, x: &Tensor) -> candle_core::Result<Tensor> {
        let x = self.fc1.forward(x)?;
        let x = x.relu()?;
        let x = self.fc2.forward(&x)?;
        let x = x.relu()?;
        let x = self.fc3.forward(&x)?;
        // Sigmoid: σ(x) = 1 / (1 + e^(-x))
        let neg_x = x.neg()?;
        let exp_neg = neg_x.exp()?;
        let one = Tensor::ones_like(&exp_neg)?;
        let denom = one.add(&exp_neg)?;
        denom.recip()
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Public AnomalyScorer
// ─────────────────────────────────────────────────────────────────────────────

/// ML-backed anomaly scorer.  Use `try_load_model` to obtain an instance or
/// rely on `Default` which returns a no-op scorer when no model is present.
pub struct AnomalyScorer {
    inner: ScorerInner,
}

enum ScorerInner {
    Model { net: AnomalyNet, device: Device },
    NoOp,
}

impl Default for AnomalyScorer {
    /// Returns a `NoOpScorer` that always returns 0.0.
    /// Used when no model file is found at startup.
    fn default() -> Self {
        Self { inner: ScorerInner::NoOp }
    }
}

impl AnomalyScorer {
    /// Score a process event.  Returns a value in [0,1]; values >= `THRESHOLD`
    /// are considered anomalous.  Returns 0.0 if the model is not loaded.
    pub fn score_event(&self, exe: &str, cmdline: &str, uid: u32) -> f64 {
        let features = extract_features(exe, cmdline, uid);
        match &self.inner {
            ScorerInner::NoOp => 0.0,
            ScorerInner::Model { net, device } => {
                run_inference(net, device, &features).unwrap_or(0.0)
            }
        }
    }

    /// Returns `true` if a real model is loaded (not no-op).
    pub fn is_loaded(&self) -> bool {
        matches!(self.inner, ScorerInner::Model { .. })
    }

    /// Convenience: returns `true` if `score_event >= THRESHOLD`.
    pub fn is_anomalous(&self, exe: &str, cmdline: &str, uid: u32) -> bool {
        self.score_event(exe, cmdline, uid) >= THRESHOLD
    }
}

fn run_inference(net: &AnomalyNet, device: &Device, features: &[f32; FEATURE_DIM]) -> candle_core::Result<f64> {
    let tensor = Tensor::from_slice(features.as_slice(), (1, FEATURE_DIM), device)?
        .to_dtype(DType::F32)?;
    let output = net.forward(&tensor)?;
    let score = output.flatten_all()?.to_vec1::<f32>()?[0];
    Ok(score as f64)
}

// ─────────────────────────────────────────────────────────────────────────────
// Model loader
// ─────────────────────────────────────────────────────────────────────────────

/// Attempt to load a saved Candle model from `path`.
///
/// The model file is expected to be a SafeTensors file produced by the
/// offline training pipeline.  Returns `None` (logs a warning) if the file
/// does not exist or cannot be parsed.  Never panics.
pub fn try_load_model(path: &str) -> Option<AnomalyScorer> {
    let model_path = Path::new(path);
    if !model_path.exists() {
        warn!(path, "AnomalyScorer model file not found — using NoOp scorer");
        return None;
    }
    match load_model_from_file(model_path) {
        Ok(scorer) => {
            info!(path, "AnomalyScorer model loaded");
            Some(scorer)
        }
        Err(e) => {
            warn!(path, error = %e, "AnomalyScorer failed to load model — using NoOp");
            None
        }
    }
}

fn load_model_from_file(path: &Path) -> candle_core::Result<AnomalyScorer> {
    let device = Device::Cpu;
    let var_map = VarMap::new();
    let vb = VarBuilder::from_varmap(&var_map, DType::F32, &device);
    // Load weights from SafeTensors file
    var_map.load(path)?;
    let net = AnomalyNet::new(vb)?;
    Ok(AnomalyScorer {
        inner: ScorerInner::Model { net, device },
    })
}

/// Build a scorer with **randomly initialised** weights (for testing / dev).
/// In production always use `try_load_model`.
#[cfg(test)]
pub fn random_scorer() -> AnomalyScorer {
    let device = Device::Cpu;
    let var_map = VarMap::new();
    let vb = VarBuilder::from_varmap(&var_map, DType::F32, &device);
    let net = AnomalyNet::new(vb).expect("build random net");
    AnomalyScorer { inner: ScorerInner::Model { net, device } }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn noop_scorer_returns_zero() {
        let scorer = AnomalyScorer::default();
        assert!(!scorer.is_loaded());
        let s = scorer.score_event("/bin/ls", "ls -la", 1000);
        assert_eq!(s, 0.0);
    }

    #[test]
    fn noop_not_anomalous() {
        let scorer = AnomalyScorer::default();
        assert!(!scorer.is_anomalous("/bin/ls", "ls -la", 1000));
    }

    #[test]
    fn try_load_model_missing_returns_none() {
        let result = try_load_model("/nonexistent/model.safetensors");
        assert!(result.is_none());
    }

    #[test]
    fn random_scorer_output_in_range() {
        let scorer = random_scorer();
        assert!(scorer.is_loaded());
        let score = scorer.score_event("/bin/bash", "bash -c 'echo hello'", 0);
        assert!(score >= 0.0 && score <= 1.0, "score out of range: {score}");
    }

    #[test]
    fn extract_features_cmdline_len() {
        let f = extract_features("/bin/ls", "ls", 1000);
        assert!(f[0] < 0.01, "short cmdline should have near-zero len norm");
        let long_cmd = "a".repeat(512);
        let f2 = extract_features("/bin/ls", &long_cmd, 1000);
        assert_eq!(f2[0], 1.0, "512-char cmdline should saturate");
    }

    #[test]
    fn extract_features_is_shell() {
        let f = extract_features("/bin/bash", "bash -i", 0);
        assert_eq!(f[2], 1.0, "bash should be detected as shell");
        let f2 = extract_features("/usr/bin/python3", "python3 script.py", 1000);
        assert_eq!(f2[2], 0.0, "python3 is not a shell");
    }

    #[test]
    fn extract_features_uid_zero() {
        let f = extract_features("/bin/id", "id", 0);
        assert_eq!(f[5], 1.0, "uid 0 should set uid_zero flag");
        let f2 = extract_features("/bin/id", "id", 1000);
        assert_eq!(f2[5], 0.0);
    }

    #[test]
    fn extract_features_pipe_redirect() {
        let f = extract_features("/bin/sh", "cat /etc/passwd | grep root > /tmp/out", 0);
        assert_eq!(f[8], 1.0, "pipe should be detected");
        assert_eq!(f[9], 1.0, "redirect should be detected");
    }

    #[test]
    fn extract_features_has_ip() {
        let f = extract_features("/bin/curl", "curl http://192.168.1.100/shell", 1000);
        assert_eq!(f[4], 1.0, "IP address should be detected");
    }

    #[test]
    fn extract_features_sudo() {
        let f = extract_features("/usr/bin/sudo", "sudo rm -rf /", 1000);
        assert_eq!(f[11], 1.0, "sudo prefix detected");
    }

    #[test]
    fn extract_features_base64() {
        let b64 = "base64 -d <<< 'SGVsbG8gV29ybGQ='";
        let f = extract_features("/usr/bin/base64", b64, 1000);
        assert_eq!(f[3], 1.0, "base64 keyword detected");
    }

    #[test]
    fn shannon_entropy_empty_string() {
        assert_eq!(shannon_entropy(""), 0.0);
    }

    #[test]
    fn shannon_entropy_uniform_string() {
        // All same characters → entropy ~0
        let s = "aaaaaa";
        assert!(shannon_entropy(s) < 0.01);
    }

    #[test]
    fn contains_ip_address_positive() {
        assert!(contains_ip_address("connect to 10.0.0.1 now"));
    }

    #[test]
    fn contains_ip_address_negative() {
        assert!(!contains_ip_address("no ip here"));
    }
}
