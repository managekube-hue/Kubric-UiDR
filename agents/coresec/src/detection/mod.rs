//! Detection engine combining Sigma and YARA-X for CoreSec.
//!
//! [`DetectionEngine`] is the single entry point for all rule-based detection
//! in the CoreSec agent.  It loads both rule sets at startup and exposes a
//! single [`DetectionEngine::detect`] method that returns all matches for a
//! given [`ProcessEvent`].

pub mod sigma;
pub mod yara;

pub use sigma::{SigmaEngine, SigmaMatch};
pub use yara::{YaraEngine, YaraMatch};

use crate::event::ProcessEvent;

// ── DetectionEngine ───────────────────────────────────────────────────────────

/// Combined Sigma + YARA detection engine.
///
/// Construct once at agent startup via [`DetectionEngine::new`], then call
/// [`DetectionEngine::detect`] for each event.
pub struct DetectionEngine {
    pub sigma: SigmaEngine,
    pub yara:  YaraEngine,
}

impl DetectionEngine {
    /// Creates an engine with zero rules (safe fallback in tests / CI).
    pub fn empty() -> Self {
        DetectionEngine {
            sigma: SigmaEngine::empty(),
            yara:  YaraEngine::empty(),
        }
    }

    /// Loads Sigma rules from `sigma_dir` and YARA rules from `yara_dir`.
    ///
    /// Both directories are optional — missing paths are treated as empty rule
    /// sets rather than hard errors so the agent can start in restricted
    /// environments.
    pub fn new(sigma_dir: &str, yara_dir: &str) -> anyhow::Result<Self> {
        let sigma = SigmaEngine::load_from_dir(sigma_dir)?;
        let yara  = YaraEngine::load_from_dir(yara_dir)?;
        Ok(DetectionEngine { sigma, yara })
    }

    /// Serialises key [`ProcessEvent`] fields to bytes for YARA scanning.
    ///
    /// Concatenates the executable path and command line with a null separator
    /// so YARA string patterns work across both fields.
    pub fn event_to_bytes(event: &ProcessEvent) -> Vec<u8> {
        let mut buf = Vec::with_capacity(
            event.executable.len() + 1 + event.cmdline.len() + 1 + event.user.len(),
        );
        buf.extend_from_slice(event.executable.as_bytes());
        buf.push(0x00);
        buf.extend_from_slice(event.cmdline.as_bytes());
        buf.push(0x00);
        buf.extend_from_slice(event.user.as_bytes());
        buf
    }

    /// Runs all Sigma rules and YARA rules against `event`.
    ///
    /// Returns `(sigma_matches, yara_matches)`.  Both vecs are empty when no
    /// rules match, which is the expected result for benign events.
    pub fn detect(
        &self,
        event: &ProcessEvent,
    ) -> (Vec<SigmaMatch>, Vec<YaraMatch>) {
        let sigma_hits = self.sigma.evaluate(event);
        let data       = Self::event_to_bytes(event);
        let yara_hits  = self.yara.scan(&data);
        (sigma_hits, yara_hits)
    }
}

#[cfg(test)]
mod tests;
