//! K-XRO-CS-YAR-001 — YARA-X rule compiler with hot-reload support.
//!
//! Loads all `.yar` / `.yara` files from a directory tree, compiles them into
//! a single `yara_x::Rules` blob, and tracks when a reload is due.
//!
//! # Hot-reload pattern
//! ```rust
//! let mut compiler = YaraCompiler::new("/opt/kubric/rules/yara")?;
//! loop {
//!     if compiler.needs_reload(Duration::from_secs(60)) {
//!         compiler.reload()?;
//!     }
//!     let rules = compiler.get_rules();
//!     // scan with rules…
//! }
//! ```
//!
//! # Cargo dependency
//! ```toml
//! yara-x  = "0.9"
//! anyhow  = "1"
//! tracing = "0.1"
//! ```

use anyhow::{Context, Result};
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};
use tracing::{info, warn};

// ─────────────────────────────────────────────────────────────────────────────
// YaraCompiler
// ─────────────────────────────────────────────────────────────────────────────

/// YARA-X rule compiler with automatic hot-reload support.
pub struct YaraCompiler {
    /// Root directory scanned recursively for `*.yar` / `*.yara` files.
    rules_dir: PathBuf,
    /// Most recently compiled rule set.
    compiled: yara_x::Rules,
    /// Wall-clock time of the last successful compilation.
    last_compiled: Instant,
    /// Number of rules in the current compiled set.
    rule_count: usize,
}

impl YaraCompiler {
    /// Create a new compiler and perform an initial compilation of all rules
    /// found under `rules_dir`.  Returns an empty-rule compiler (not an error)
    /// if the directory does not exist.
    pub fn new(rules_dir: &str) -> Result<Self> {
        let dir = PathBuf::from(rules_dir);
        let (compiled, rule_count) = compile_dir(&dir);
        info!(
            dir = rules_dir,
            rules = rule_count,
            "YaraCompiler initialised"
        );
        Ok(Self {
            rules_dir: dir,
            compiled,
            last_compiled: Instant::now(),
            rule_count,
        })
    }

    /// Manually trigger a recompile of all rules from disk.
    /// Failed individual files are logged and skipped; the method never
    /// returns an error due to bad rule syntax.
    pub fn reload(&mut self) -> Result<()> {
        let (compiled, count) = compile_dir(&self.rules_dir);
        self.compiled = compiled;
        self.rule_count = count;
        self.last_compiled = Instant::now();
        info!(
            dir = %self.rules_dir.display(),
            rules = count,
            "YaraCompiler reloaded"
        );
        Ok(())
    }

    /// Returns `true` if `interval` has elapsed since the last compilation.
    /// Callers should call `reload()` when this returns `true`.
    pub fn needs_reload(&self, interval: Duration) -> bool {
        self.last_compiled.elapsed() >= interval
    }

    /// Get a reference to the current compiled rules.
    pub fn get_rules(&self) -> &yara_x::Rules {
        &self.compiled
    }

    /// Number of rules in the current compiled set.
    pub fn rule_count(&self) -> usize {
        self.rule_count
    }

    /// Path to the rules directory.
    pub fn rules_dir(&self) -> &Path {
        &self.rules_dir
    }

    /// Seconds since the last compilation.
    pub fn age_secs(&self) -> f64 {
        self.last_compiled.elapsed().as_secs_f64()
    }

    /// Compile rules from an arbitrary directory (one-shot, no hot-reload).
    /// Useful for loading a secondary rule pack.
    pub fn compile_rules(&self) -> Result<yara_x::Rules> {
        let (rules, _) = compile_dir(&self.rules_dir);
        Ok(rules)
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Compilation internals
// ─────────────────────────────────────────────────────────────────────────────

/// Walk `dir` recursively, compile every `*.yar` / `*.yara` file, and return
/// the resulting `(Rules, file_count)`.  Individual compile errors are logged
/// and skipped — the returned rule set may be a subset of all files.
pub fn compile_dir(dir: &Path) -> (yara_x::Rules, usize) {
    let mut compiler = yara_x::Compiler::new();
    let mut count = 0usize;

    if dir.exists() {
        compile_walk(dir, &mut compiler, &mut count);
    } else {
        warn!(dir = %dir.display(), "YARA rules directory not found — using empty rule set");
    }

    let rules = compiler.build();
    (rules, count)
}

fn compile_walk(dir: &Path, compiler: &mut yara_x::Compiler, count: &mut usize) {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(e) => {
            warn!(dir = %dir.display(), error = %e, "Cannot read YARA rules directory");
            return;
        }
    };

    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            compile_walk(&path, compiler, count);
        } else {
            let ext = path
                .extension()
                .and_then(|e| e.to_str())
                .unwrap_or("")
                .to_lowercase();
            if ext != "yar" && ext != "yara" {
                continue;
            }
            match std::fs::read_to_string(&path) {
                Ok(src) => {
                    if compiler.add_source(src.as_str()).is_ok() {
                        *count += 1;
                        // info!(file = %path.display(), "YARA rule compiled");
                    } else {
                        warn!(file = %path.display(), "YARA compile error — skipping rule file");
                    }
                }
                Err(e) => {
                    warn!(file = %path.display(), error = %e, "YARA read error — skipping");
                }
            }
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// RuleStats — helpful diagnostics
// ─────────────────────────────────────────────────────────────────────────────

/// Diagnostic snapshot of the compiler state.
#[derive(Debug)]
pub struct RuleStats {
    pub rule_count: usize,
    pub rules_dir: PathBuf,
    pub age_secs: f64,
}

impl YaraCompiler {
    /// Return a diagnostic snapshot of the current compiler state.
    pub fn stats(&self) -> RuleStats {
        RuleStats {
            rule_count: self.rule_count,
            rules_dir: self.rules_dir.clone(),
            age_secs: self.age_secs(),
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::TempDir;

    fn write_rule(dir: &Path, filename: &str, content: &str) {
        let mut f = std::fs::File::create(dir.join(filename)).unwrap();
        f.write_all(content.as_bytes()).unwrap();
    }

    const VALID_RULE: &str = r#"
rule test_rule {
    meta:
        description = "test"
    strings:
        $a = "malware"
    condition:
        $a
}
"#;

    const INVALID_RULE: &str = "rule broken { this is not valid yara syntax !!!";

    #[test]
    fn new_with_missing_dir_returns_empty() {
        let c = YaraCompiler::new("/nonexistent/path/to/rules").unwrap();
        assert_eq!(c.rule_count(), 0);
    }

    #[test]
    fn compiles_valid_rule() {
        let dir = TempDir::new().unwrap();
        write_rule(dir.path(), "test.yar", VALID_RULE);
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        assert_eq!(c.rule_count(), 1);
    }

    #[test]
    fn skips_invalid_rule_without_panic() {
        let dir = TempDir::new().unwrap();
        write_rule(dir.path(), "good.yar", VALID_RULE);
        write_rule(dir.path(), "bad.yar", INVALID_RULE);
        // Should succeed: bad rule logged and skipped
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        assert_eq!(c.rule_count(), 1, "only the valid rule should be counted");
    }

    #[test]
    fn needs_reload_after_interval() {
        let dir = TempDir::new().unwrap();
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        // A 1ms interval should immediately trigger
        assert!(c.needs_reload(Duration::from_millis(0)));
        // A 1-hour interval should not
        assert!(!c.needs_reload(Duration::from_secs(3600)));
    }

    #[test]
    fn reload_picks_up_new_rules() {
        let dir = TempDir::new().unwrap();
        let mut c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        assert_eq!(c.rule_count(), 0);
        // Add a rule file and reload
        write_rule(dir.path(), "new.yar", VALID_RULE);
        c.reload().unwrap();
        assert_eq!(c.rule_count(), 1);
    }

    #[test]
    fn get_rules_returns_compilable_scanner() {
        let dir = TempDir::new().unwrap();
        write_rule(dir.path(), "t.yar", VALID_RULE);
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        let rules = c.get_rules();
        let mut scanner = yara_x::Scanner::new(rules);
        let result = scanner.scan(b"this contains malware text");
        assert!(result.is_ok());
        let matches: Vec<_> = result.unwrap().matching_rules().collect();
        assert_eq!(matches.len(), 1);
    }

    #[test]
    fn stats_returns_correct_count() {
        let dir = TempDir::new().unwrap();
        write_rule(dir.path(), "s.yar", VALID_RULE);
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        let stats = c.stats();
        assert_eq!(stats.rule_count, 1);
    }

    #[test]
    fn compile_walk_recursive() {
        let dir = TempDir::new().unwrap();
        let sub = dir.path().join("sub");
        std::fs::create_dir_all(&sub).unwrap();
        write_rule(&sub, "sub.yar", VALID_RULE);
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        assert_eq!(c.rule_count(), 1, "recursive walk should find sub-directory rules");
    }

    #[test]
    fn non_yara_files_ignored() {
        let dir = TempDir::new().unwrap();
        write_rule(dir.path(), "readme.txt", "this is not a YARA rule");
        write_rule(dir.path(), "rule.yar", VALID_RULE);
        let c = YaraCompiler::new(dir.path().to_str().unwrap()).unwrap();
        assert_eq!(c.rule_count(), 1, ".txt should be ignored");
    }
}
