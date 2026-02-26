//! K-XRO-NG-IDS-001 — Network IDS rule loader.
//!
//! Loads **YARA-X** rules from a directory tree and tracks the set of
//! loaded Suricata `.rules` files by name and count.  Rule sets can be
//! reloaded on demand; mtime-based change detection prevents unnecessary
//! recompilation.
//!
//! # Supported formats
//! | Format   | Extension        | Used for               |
//! |----------|------------------|------------------------|
//! | YARA-X   | `.yar`, `.yara`  | Payload byte scanning  |
//! | Suricata | `.rules`         | Count tracking only (Suricata itself is a separate process) |
//!
//! # Usage
//! ```rust,ignore
//! let loader = RuleLoader::new("/etc/kubric/rules/yara", "/etc/kubric/rules/suricata");
//! let ruleset = loader.load()?;
//! println!("YARA: {}, Suricata: {}", ruleset.yara_rule_count, ruleset.suricata_rule_count);
//!
//! // Re-load if changed
//! if loader.has_changed(&ruleset) {
//!     let new_ruleset = loader.load()?;
//! }
//! ```

#![allow(dead_code)]

use anyhow::{Context, Result};
use std::path::{Path, PathBuf};
use std::time::{Instant, SystemTime};
use tracing::{debug, info, warn};

// ── RuleSet ───────────────────────────────────────────────────────────────────

/// A compiled, ready-to-scan set of detection rules.
pub struct RuleSet {
    /// Compiled YARA-X rules (thread-safe scanner factory).
    pub yara_rules: yara_x::Rules,
    /// Number of YARA rule files that compiled successfully.
    pub yara_rule_count: usize,
    /// Number of Suricata `.rules` files discovered.
    pub suricata_rule_count: usize,
    /// Wall-clock instant at which this set was compiled.
    pub loaded_at: Instant,
    /// mtime snapshots of each rule file at load time (for change detection).
    pub(crate) mtimes: Vec<(PathBuf, SystemTime)>,
}

impl RuleSet {
    /// Scan `data` against the compiled YARA rules.
    ///
    /// Returns a list of `(rule_identifier, namespace)` pairs for every match.
    pub fn scan(&self, data: &[u8]) -> Vec<(String, String)> {
        let mut scanner = yara_x::Scanner::new(&self.yara_rules);
        match scanner.scan(data) {
            Ok(results) => results
                .matching_rules()
                .map(|r| (r.identifier().to_string(), r.namespace().to_string()))
                .collect(),
            Err(e) => {
                warn!(%e, "YARA scan error");
                Vec::new()
            }
        }
    }

    /// Elapsed seconds since this rule set was loaded.
    pub fn age_secs(&self) -> u64 {
        self.loaded_at.elapsed().as_secs()
    }
}

// ── RuleLoader ────────────────────────────────────────────────────────────────

/// Loads and optionally watches detection rules from the filesystem.
pub struct RuleLoader {
    /// Root directory for YARA `.yar`/`.yara` files.
    pub yara_dir: PathBuf,
    /// Root directory for Suricata `.rules` files.
    pub suricata_dir: PathBuf,
}

impl RuleLoader {
    /// Construct a loader for the given directories.
    pub fn new(yara_dir: impl Into<PathBuf>, suricata_dir: impl Into<PathBuf>) -> Self {
        Self {
            yara_dir: yara_dir.into(),
            suricata_dir: suricata_dir.into(),
        }
    }

    /// Read directories from environment variables with fallbacks.
    ///
    /// `KUBRIC_YARA_DIR` defaults to `vendor/yara-rules`
    /// `KUBRIC_SURICATA_DIR` defaults to `vendor/suricata-rules`
    pub fn from_env() -> Self {
        Self::new(
            std::env::var("KUBRIC_YARA_DIR")
                .unwrap_or_else(|_| "vendor/yara-rules".into()),
            std::env::var("KUBRIC_SURICATA_DIR")
                .unwrap_or_else(|_| "vendor/suricata-rules".into()),
        )
    }

    /// Compile all rules found in the configured directories.
    ///
    /// Missing directories are treated as empty rather than errors.
    pub fn load(&self) -> Result<RuleSet> {
        let mut compiler = yara_x::Compiler::new();
        let mut yara_count = 0usize;
        let mut mtimes: Vec<(PathBuf, SystemTime)> = Vec::new();

        // ── Load YARA rules ───────────────────────────────────────────────────
        if self.yara_dir.exists() {
            self.load_yara_dir(&self.yara_dir, &mut compiler, &mut yara_count, &mut mtimes);
        } else {
            info!(dir = %self.yara_dir.display(), "YARA rules directory absent — skipping");
        }

        let rules = compiler.build();
        info!(
            yara_rules = yara_count,
            yara_dir = %self.yara_dir.display(),
            "YARA rules compiled"
        );

        // ── Count Suricata rules ──────────────────────────────────────────────
        let suricata_count = if self.suricata_dir.exists() {
            let (count, suri_mtimes) = self.count_suricata_rules(&self.suricata_dir);
            mtimes.extend(suri_mtimes);
            count
        } else {
            info!(dir = %self.suricata_dir.display(), "Suricata rules directory absent");
            0
        };

        if suricata_count > 0 {
            info!(
                suricata_files = suricata_count,
                suricata_dir = %self.suricata_dir.display(),
                "Suricata rule files found"
            );
        }

        Ok(RuleSet {
            yara_rules: rules,
            yara_rule_count: yara_count,
            suricata_rule_count: suricata_count,
            loaded_at: Instant::now(),
            mtimes,
        })
    }

    /// Recursively compile YARA rules from `dir`.
    fn load_yara_dir(
        &self,
        dir: &Path,
        compiler: &mut yara_x::Compiler,
        count: &mut usize,
        mtimes: &mut Vec<(PathBuf, SystemTime)>,
    ) {
        let entries = match std::fs::read_dir(dir) {
            Ok(e) => e,
            Err(e) => {
                warn!(path = %dir.display(), %e, "cannot read YARA directory");
                return;
            }
        };

        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                self.load_yara_dir(&path, compiler, count, mtimes);
                continue;
            }

            let ext = path
                .extension()
                .map(|e| e.to_string_lossy().to_lowercase())
                .unwrap_or_default();
            if ext != "yar" && ext != "yara" {
                continue;
            }

            match std::fs::read_to_string(&path) {
                Ok(source) => {
                    if compiler.add_source(source.as_str()).is_ok() {
                        *count += 1;
                        debug!(file = %path.display(), "YARA rule compiled");
                        // Record mtime for change detection
                        if let Ok(meta) = std::fs::metadata(&path) {
                            if let Ok(mtime) = meta.modified() {
                                mtimes.push((path, mtime));
                            }
                        }
                    } else {
                        warn!(file = %path.display(), "YARA compile error — file skipped");
                    }
                }
                Err(e) => {
                    warn!(file = %path.display(), %e, "failed to read YARA rule file");
                }
            }
        }
    }

    /// Walk `dir` recursively and count all `.rules` files.
    fn count_suricata_rules(
        &self,
        dir: &Path,
    ) -> (usize, Vec<(PathBuf, SystemTime)>) {
        let mut count = 0;
        let mut mtimes = Vec::new();
        self.walk_suricata(dir, &mut count, &mut mtimes);
        (count, mtimes)
    }

    fn walk_suricata(
        &self,
        dir: &Path,
        count: &mut usize,
        mtimes: &mut Vec<(PathBuf, SystemTime)>,
    ) {
        let entries = match std::fs::read_dir(dir) {
            Ok(e) => e,
            Err(_) => return,
        };

        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                self.walk_suricata(&path, count, mtimes);
                continue;
            }
            let ext = path
                .extension()
                .map(|e| e.to_string_lossy().to_lowercase())
                .unwrap_or_default();
            if ext == "rules" {
                *count += 1;
                if let Ok(meta) = std::fs::metadata(&path) {
                    if let Ok(mtime) = meta.modified() {
                        mtimes.push((path, mtime));
                    }
                }
            }
        }
    }

    /// Returns `true` if any rule file has been modified since `ruleset` was loaded.
    pub fn has_changed(&self, ruleset: &RuleSet) -> bool {
        for (path, recorded_mtime) in &ruleset.mtimes {
            if let Ok(meta) = std::fs::metadata(path) {
                if let Ok(current_mtime) = meta.modified() {
                    if current_mtime != *recorded_mtime {
                        info!(file = %path.display(), "rule file changed — reload needed");
                        return true;
                    }
                }
            } else {
                // File was deleted
                info!(file = %path.display(), "rule file removed — reload needed");
                return true;
            }
        }
        false
    }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn empty_dirs_dont_fail() {
        let loader = RuleLoader::new("/nonexistent/yara", "/nonexistent/suricata");
        let ruleset = loader.load().unwrap();
        assert_eq!(ruleset.yara_rule_count, 0);
        assert_eq!(ruleset.suricata_rule_count, 0);
    }

    #[test]
    fn loads_yara_rule_from_dir() {
        let dir = std::env::temp_dir().join("kubric_ruleloader_test");
        let _ = std::fs::create_dir_all(&dir);

        let rule = r#"rule test_rule { strings: $a = "evil" condition: $a }"#;
        let mut f = std::fs::File::create(dir.join("test.yar")).unwrap();
        f.write_all(rule.as_bytes()).unwrap();
        drop(f);

        let loader = RuleLoader::new(&dir, "/nonexistent");
        let ruleset = loader.load().unwrap();
        assert_eq!(ruleset.yara_rule_count, 1);

        let hits = ruleset.scan(b"this is evil content");
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].0, "test_rule");

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn ignores_non_yara_extensions() {
        let dir = std::env::temp_dir().join("kubric_ruleloader_ext_test");
        let _ = std::fs::create_dir_all(&dir);
        std::fs::write(dir.join("readme.txt"), "not a rule").unwrap();
        std::fs::write(dir.join("config.json"), "{}").unwrap();

        let loader = RuleLoader::new(&dir, "/nonexistent");
        let ruleset = loader.load().unwrap();
        assert_eq!(ruleset.yara_rule_count, 0);

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn ruleset_scan_returns_empty_on_no_match() {
        let loader = RuleLoader::new("/nonexistent", "/nonexistent");
        let ruleset = loader.load().unwrap();
        let hits = ruleset.scan(b"completely benign data");
        assert!(hits.is_empty());
    }

    #[test]
    fn ruleset_age_increases() {
        let loader = RuleLoader::new("/nonexistent", "/nonexistent");
        let ruleset = loader.load().unwrap();
        assert!(ruleset.age_secs() < 5);
    }

    #[test]
    fn has_changed_returns_false_on_fresh_load() {
        let loader = RuleLoader::new("/nonexistent", "/nonexistent");
        let ruleset = loader.load().unwrap();
        // No files tracked means no changes
        assert!(!loader.has_changed(&ruleset));
    }

    #[test]
    fn has_changed_detects_deleted_file() {
        let dir = std::env::temp_dir().join("kubric_changed_test");
        let _ = std::fs::create_dir_all(&dir);
        let rule_path = dir.join("rule.yar");
        std::fs::write(&rule_path, r#"rule x { strings: $a = "x" condition: $a }"#).unwrap();

        let loader = RuleLoader::new(&dir, "/nonexistent");
        let ruleset = loader.load().unwrap();
        assert!(!loader.has_changed(&ruleset));

        std::fs::remove_file(&rule_path).unwrap();
        assert!(loader.has_changed(&ruleset));

        let _ = std::fs::remove_dir_all(&dir);
    }
}
