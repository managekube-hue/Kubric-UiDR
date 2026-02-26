#![allow(dead_code)]
//! YARA-X engine wrapper for CoreSec.
//!
//! Loads all `*.yar` / `*.yara` files from a directory tree, compiles them
//! with a single [`yara_x::Compiler`] pass, and exposes a [`YaraEngine`] that
//! can scan arbitrary byte slices.
//!
//! # Scale
//! The implementation is designed to handle 25 000+ rules without panicking:
//! - All rules are compiled once at startup into a single [`yara_x::Rules`]
//!   object.
//! - Individual compile errors are logged and skipped instead of aborting.
//! - Missing rule directories are treated as empty (CI-safe).

use std::fs;
use std::path::Path;

use anyhow::Result;
use tracing::warn;
use yara_x::Compiler;

// ── Public types ─────────────────────────────────────────────────────────────

/// A single YARA rule match returned by [`YaraEngine::scan`].
#[derive(Debug, Clone)]
pub struct YaraMatch {
    pub rule:      String,
    pub namespace: String,
}

// ── YaraEngine ────────────────────────────────────────────────────────────────

/// Compiled YARA rule set.  Construct once and reuse across events.
pub struct YaraEngine {
    rules: yara_x::Rules,
    /// Rule count tracked at compile time (yara_x::Rules::num_rules is pub(crate)).
    count: usize,
}

impl YaraEngine {
    /// Creates an engine with zero rules (safe fallback for CI / missing dir).
    pub fn empty() -> Self {
        let compiler = Compiler::new();
        let rules = compiler.build();
        YaraEngine { rules, count: 0 }
    }

    /// Returns the number of compiled rules.
    pub fn rule_count(&self) -> usize {
        self.count
    }

    /// Recursively walks `dir`, compiling every `*.yar` / `*.yara` file.
    ///
    /// Files that fail to compile are skipped with a `WARN` log line so that
    /// a single bad rule file does not block startup.
    pub fn load_from_dir(dir: &str) -> Result<Self> {
        let mut compiler = Compiler::new();
        let mut count = 0usize;
        load_dir(Path::new(dir), &mut compiler, &mut count);
        let rules = compiler.build();
        Ok(YaraEngine { rules, count })
    }

    /// Scans `data` against all compiled rules.
    ///
    /// Returns an empty [`Vec`] when there are no matches or when `data` is
    /// empty.
    pub fn scan(&self, data: &[u8]) -> Vec<YaraMatch> {
        if data.is_empty() {
            return vec![];
        }
        let mut scanner = yara_x::Scanner::new(&self.rules);
        let results = match scanner.scan(data) {
            Ok(r) => r,
            Err(e) => {
                warn!(error = %e, "yara: scan error");
                return vec![];
            }
        };
        results
            .matching_rules()
            .map(|r| YaraMatch {
                rule:      r.identifier().to_string(),
                namespace: r.namespace().to_string(),
            })
            .collect()
    }
}

// ── Internal helpers ──────────────────────────────────────────────────────────

fn load_dir(path: &Path, compiler: &mut Compiler, count: &mut usize) {
    let entries = match fs::read_dir(path) {
        Ok(e) => e,
        Err(_) => return, // directory missing — not an error in CI
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            load_dir(&p, compiler, count);
        } else {
            let ext = p.extension().and_then(|e| e.to_str()).unwrap_or("");
            if ext == "yar" || ext == "yara" {
                match fs::read_to_string(&p) {
                    Ok(src) => {
                        if let Err(e) = compiler.add_source(src.as_bytes()) {
                            warn!(
                                path = %p.display(),
                                error = %e,
                                "yara: skipping rule that failed to compile"
                            );
                        } else {
                            *count += 1;
                        }
                    }
                    Err(e) => {
                        warn!(path = %p.display(), error = %e, "yara: cannot read file");
                    }
                }
            }
        }
    }
}
