//! K-XRO-CS-ML-002 — TinyLlama model loader for local LLM inference (offline mode).
//!
//! Loads a TinyLlama (1.1B) language model from a local directory for
//! air-gapped endpoint security analysis.  When model files are missing the
//! struct falls back gracefully by returning `"LLM offline"` for every query.
//!
//! # Candle dependencies (Cargo.toml)
//! ```toml
//! candle-core         = { version = "0.6" }
//! candle-nn           = { version = "0.6" }
//! candle-transformers = { version = "0.6" }
//! tokenizers          = { version = "0.19", default-features = false, features = ["onig"] }
//! ```

use anyhow::{anyhow, Context, Result};
use candle_core::{DType, Device, Tensor};
use candle_nn::VarBuilder;
use candle_transformers::models::llama::{Cache, Config as LlamaConfig, Llama};
use std::path::{Path, PathBuf};
use tokenizers::Tokenizer;
use tracing::{info, warn};

/// Offline fallback string when model files are unavailable.
const OFFLINE_MSG: &str = "LLM offline";

/// System prompt prepended to all security analysis queries.
const SYSTEM_PROMPT: &str = concat!(
    "<|system|>\n",
    "You are a threat detection assistant embedded in a Kubric XDR endpoint agent. ",
    "Analyse the following security event and respond concisely with: ",
    "(1) threat assessment, (2) MITRE ATT&CK technique if applicable, ",
    "(3) recommended action. Keep response under 150 tokens.\n",
    "<|assistant|>\n"
);

// ─────────────────────────────────────────────────────────────────────────────
// ModelConfig
// ─────────────────────────────────────────────────────────────────────────────

/// Configuration for loading TinyLlama from the local filesystem.
#[derive(Debug, Clone)]
pub struct ModelConfig {
    /// Path to the SafeTensors model weights file(s).
    pub model_path: PathBuf,
    /// Path to the HuggingFace tokenizer.json file.
    pub tokenizer_path: PathBuf,
    /// Maximum tokens to generate per inference call.
    pub max_tokens: usize,
    /// Sampling temperature.  Use 0.0 for greedy (deterministic) decoding.
    pub temperature: f64,
}

impl Default for ModelConfig {
    fn default() -> Self {
        Self {
            model_path: PathBuf::from("/opt/kubric/models/tinyllama/model.safetensors"),
            tokenizer_path: PathBuf::from("/opt/kubric/models/tinyllama/tokenizer.json"),
            max_tokens: 150,
            temperature: 0.1,
        }
    }
}

impl ModelConfig {
    /// Create a config pointing to `base_dir/model.safetensors` and
    /// `base_dir/tokenizer.json` with sensible defaults.
    pub fn from_dir(base_dir: &str) -> Self {
        let dir = PathBuf::from(base_dir);
        Self {
            model_path: dir.join("model.safetensors"),
            tokenizer_path: dir.join("tokenizer.json"),
            max_tokens: 150,
            temperature: 0.1,
        }
    }

    /// Check whether all required files are present on disk.
    pub fn files_present(&self) -> bool {
        self.model_path.exists() && self.tokenizer_path.exists()
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// TinyLlamaInference
// ─────────────────────────────────────────────────────────────────────────────

/// Local TinyLlama inference engine.
///
/// # Usage
/// ```rust
/// let cfg = ModelConfig::from_dir("/opt/kubric/models/tinyllama");
/// if TinyLlamaInference::is_available(&cfg) {
///     let llm = TinyLlamaInference::load(&cfg)?;
///     let answer = llm.analyze_event(event_json)?;
/// }
/// ```
pub struct TinyLlamaInference {
    model: Llama,
    tokenizer: Tokenizer,
    device: Device,
    config: ModelConfig,
    cache: Cache,
}

impl TinyLlamaInference {
    /// Check whether model files exist on disk without loading them.
    pub fn is_available(cfg: &ModelConfig) -> bool {
        cfg.files_present()
    }

    /// Load the model from `cfg.model_path` and tokenizer from
    /// `cfg.tokenizer_path`.  Returns an error if any file is missing or
    /// cannot be parsed.
    pub fn load(cfg: &ModelConfig) -> Result<Self> {
        if !cfg.files_present() {
            return Err(anyhow!(
                "TinyLlama model files not found: model={} tokenizer={}",
                cfg.model_path.display(),
                cfg.tokenizer_path.display(),
            ));
        }

        info!(
            model = %cfg.model_path.display(),
            tokenizer = %cfg.tokenizer_path.display(),
            "Loading TinyLlama model"
        );

        let device = Device::Cpu;

        // Load tokenizer
        let tokenizer = Tokenizer::from_file(&cfg.tokenizer_path)
            .map_err(|e| anyhow!("tokenizer load failed: {e}"))?;

        // TinyLlama-1.1B configuration (matches HF hub defaults)
        let llama_cfg = tinyllama_config();

        // Load weights
        let vb = unsafe {
            VarBuilder::from_mmaped_safetensors(
                &[&cfg.model_path],
                DType::F32,
                &device,
            )
            .context("load safetensors weights")?
        };

        let use_kv_cache = true;
        let cache = Cache::new(use_kv_cache, DType::F32, &llama_cfg, &device)
            .context("create KV cache")?;

        let model = Llama::load(vb, &llama_cfg).context("build Llama model")?;

        info!("TinyLlama loaded successfully");
        Ok(Self { model, tokenizer, device, config: cfg.clone(), cache })
    }

    /// Run autoregressive generation on a raw prompt string.
    /// Returns the generated text (excluding the prompt itself).
    pub fn infer(&mut self, prompt: &str) -> Result<String> {
        let encoding = self
            .tokenizer
            .encode(prompt, true)
            .map_err(|e| anyhow!("tokenise failed: {e}"))?;
        let mut token_ids: Vec<u32> = encoding.get_ids().to_vec();

        let eos_token = self
            .tokenizer
            .token_to_id("</s>")
            .unwrap_or(2);

        let prompt_len = token_ids.len();
        let mut generated: Vec<u32> = Vec::new();

        for pos in 0..self.config.max_tokens {
            let current_idx = if pos == 0 {
                // First pass: feed full prompt
                let ids = &token_ids[..];
                Tensor::new(ids, &self.device)?
                    .unsqueeze(0)? // shape [1, seq_len]
            } else {
                // Subsequent passes: only last token (KV cache active)
                let last = *generated.last().unwrap_or(&eos_token);
                Tensor::new(&[last], &self.device)?
                    .unsqueeze(0)?
            };

            let seq_pos = if pos == 0 { 0 } else { prompt_len + pos - 1 };
            let logits = self.model.forward(&current_idx, seq_pos, &mut self.cache)?;

            // Logits shape: [1, seq_len, vocab_size] — take last token
            let last_logits = logits.squeeze(0)?.get(logits.dim(1)? - 1)?;

            let next_token = if self.config.temperature <= 0.0 {
                // Greedy
                let idx = last_logits.argmax(0)?;
                idx.to_scalar::<u32>()?
            } else {
                // Temperature sampling
                let scaled = (last_logits / self.config.temperature)?;
                sample_argmax(&scaled)?
            };

            if next_token == eos_token {
                break;
            }
            generated.push(next_token);
            token_ids.push(next_token);
        }

        let text = self
            .tokenizer
            .decode(&generated, true)
            .map_err(|e| anyhow!("decode failed: {e}"))?;
        Ok(text.trim().to_string())
    }

    /// Wrap `event_json` in the security analysis system prompt and run
    /// inference.  Returns the model's analysis or `"LLM offline"` on error.
    pub fn analyze_event(&mut self, event_json: &str) -> Result<String> {
        let prompt = format!(
            "{SYSTEM_PROMPT}<|user|>\nAnalyse this security event:\n{event_json}\n<|assistant|>\n"
        );
        self.infer(&prompt)
    }
}

/// Greedy argmax sampling — returns the index of the maximum logit.
fn sample_argmax(logits: &Tensor) -> Result<u32> {
    let idx = logits.argmax(0).context("argmax")?;
    Ok(idx.to_scalar::<u32>()?)
}

// ─────────────────────────────────────────────────────────────────────────────
// TinyLlama-1.1B default configuration
// ─────────────────────────────────────────────────────────────────────────────

fn tinyllama_config() -> LlamaConfig {
    // Matches TinyLlama/TinyLlama-1.1B-Chat-v1.0 on HuggingFace Hub
    LlamaConfig {
        hidden_size: 2048,
        intermediate_size: 5632,
        num_hidden_layers: 22,
        num_attention_heads: 32,
        num_key_value_heads: Some(4),
        vocab_size: 32000,
        rms_norm_eps: 1e-5,
        rope_theta: 10_000.0,
        bos_token_id: Some(1),
        eos_token_id: Some(candle_transformers::models::llama::LlamaEosToks::Single(2)),
        use_flash_attn: false,
        ..Default::default()
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Offline wrapper — always safe to call
// ─────────────────────────────────────────────────────────────────────────────

/// A safe, offline-capable wrapper around [`TinyLlamaInference`].
///
/// Callers that do not want to handle the `Option` themselves can use this
/// type: when the model is unavailable every call returns `"LLM offline"`.
pub struct SafeLlm {
    inner: Option<TinyLlamaInference>,
}

impl SafeLlm {
    /// Try to load the model.  On failure (or missing files) returns an
    /// instance that always responds `"LLM offline"`.
    pub fn try_load(cfg: &ModelConfig) -> Self {
        match TinyLlamaInference::load(cfg) {
            Ok(m) => {
                info!("SafeLlm: model loaded");
                Self { inner: Some(m) }
            }
            Err(e) => {
                warn!(error = %e, "SafeLlm: falling back to offline mode");
                Self { inner: None }
            }
        }
    }

    /// Returns `true` if the model is loaded and ready.
    pub fn is_ready(&self) -> bool {
        self.inner.is_some()
    }

    /// Analyse a security event JSON string using the LLM.
    /// Returns `"LLM offline"` if the model is not loaded.
    pub fn analyze_event(&mut self, event_json: &str) -> String {
        match &mut self.inner {
            None => OFFLINE_MSG.to_string(),
            Some(m) => m.analyze_event(event_json).unwrap_or_else(|e| {
                warn!(error = %e, "LLM inference error");
                OFFLINE_MSG.to_string()
            }),
        }
    }

    /// Run raw inference on a prompt.
    /// Returns `"LLM offline"` if model is not loaded.
    pub fn infer(&mut self, prompt: &str) -> String {
        match &mut self.inner {
            None => OFFLINE_MSG.to_string(),
            Some(m) => m.infer(prompt).unwrap_or_else(|_| OFFLINE_MSG.to_string()),
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn model_config_default_paths() {
        let cfg = ModelConfig::default();
        assert!(cfg.model_path.to_str().unwrap().contains("tinyllama"));
        assert!(cfg.tokenizer_path.to_str().unwrap().contains("tokenizer"));
        assert_eq!(cfg.max_tokens, 150);
    }

    #[test]
    fn model_config_from_dir() {
        let cfg = ModelConfig::from_dir("/tmp/mymodel");
        assert_eq!(cfg.model_path, PathBuf::from("/tmp/mymodel/model.safetensors"));
        assert_eq!(cfg.tokenizer_path, PathBuf::from("/tmp/mymodel/tokenizer.json"));
    }

    #[test]
    fn model_config_files_present_false_when_missing() {
        let cfg = ModelConfig::from_dir("/nonexistent/path/that/does/not/exist");
        assert!(!cfg.files_present());
    }

    #[test]
    fn is_available_returns_false_when_missing() {
        let cfg = ModelConfig::from_dir("/nonexistent/model");
        assert!(!TinyLlamaInference::is_available(&cfg));
    }

    #[test]
    fn load_fails_gracefully_on_missing_files() {
        let cfg = ModelConfig::from_dir("/nonexistent/model");
        let result = TinyLlamaInference::load(&cfg);
        assert!(result.is_err(), "should fail when model files missing");
        let msg = result.unwrap_err().to_string();
        assert!(msg.contains("not found"), "error should mention not found: {msg}");
    }

    #[test]
    fn safe_llm_offline_mode() {
        let cfg = ModelConfig::from_dir("/nonexistent/model");
        let mut llm = SafeLlm::try_load(&cfg);
        assert!(!llm.is_ready());
        assert_eq!(llm.analyze_event(r#"{"exe":"/bin/bash"}"#), OFFLINE_MSG);
        assert_eq!(llm.infer("some prompt"), OFFLINE_MSG);
    }

    #[test]
    fn system_prompt_contains_mitre() {
        assert!(SYSTEM_PROMPT.contains("MITRE ATT&CK"));
    }

    #[test]
    fn offline_msg_constant() {
        assert_eq!(OFFLINE_MSG, "LLM offline");
    }
}
