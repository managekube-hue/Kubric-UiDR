//! K-XRO-CS-FIM-002 — BLAKE3 baseline builder for filesystem change detection
//!
//! Captures a point-in-time cryptographic snapshot of a filesystem subtree.
//! Each file is hashed with BLAKE3; the resulting `Baseline` can be serialised
//! to disk and later compared against a freshly-computed `Baseline` to produce
//! a `Vec<BaselineDiff>` describing every added, modified, or removed file.
//!
//! Production counterpart: `agents/coresec/src/fim.rs` (baseline section)
//!
//! Dependencies in Cargo.toml:
//!   blake3  = "1"
//!   serde   = { version = "1", features = ["derive"] }
//!   serde_json = "1"
//!   anyhow  = "1"
//!   tracing = "0.1"
//!
//! Module: K-XRO-CS-FIM-002

use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

use anyhow::Result;
use blake3::Hasher;
use serde::{Deserialize, Serialize};
use tracing::{info, warn};

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

/// Metadata and hash for a single file in the baseline.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct BaselineEntry {
    /// Absolute file path.
    pub path: PathBuf,
    /// BLAKE3 hex-encoded hash of the file contents.
    pub hash: String,
    /// File size in bytes.
    pub size: u64,
    /// Last-modification time as seconds since the Unix epoch.
    pub mtime: u64,
    /// Unix permission bits (lower 12 bits of `st_mode`), e.g. `0o644`.
    pub permissions: u32,
}

/// Describes a difference between an old and a new baseline.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
pub enum BaselineDiff {
    /// A new file appeared that was not in the old baseline.
    Added {
        path: PathBuf,
    },
    /// An existing file changed content (hash mismatch).
    Modified {
        path:     PathBuf,
        old_hash: String,
        new_hash: String,
    },
    /// A file present in the old baseline is gone.
    Removed {
        path: PathBuf,
    },
}

impl std::fmt::Display for BaselineDiff {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BaselineDiff::Added   { path }                        => write!(f, "ADDED    {}", path.display()),
            BaselineDiff::Modified { path, old_hash, new_hash }  => write!(f, "MODIFIED {} ({} → {})", path.display(), &old_hash[..8], &new_hash[..8]),
            BaselineDiff::Removed { path }                        => write!(f, "REMOVED  {}", path.display()),
        }
    }
}

// ---------------------------------------------------------------------------
// Baseline
// ---------------------------------------------------------------------------

/// Immutable filesystem snapshot.
///
/// Build a fresh snapshot with `Baseline::new` + `scan`, then serialise to
/// JSON with `save` and reload with `load`.  Compare two snapshots with
/// `compare`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Baseline {
    /// All hashed files, keyed by absolute path.
    pub entries: HashMap<PathBuf, BaselineEntry>,
    /// Unix timestamp (seconds) when the baseline was created.
    pub created_at: u64,
    /// Tenant identifier.
    pub tenant_id: String,
    /// Agent / host identifier.
    pub agent_id: String,
}

impl Baseline {
    /// Create an empty baseline for the given tenant and agent.
    pub fn new(tenant_id: impl Into<String>, agent_id: impl Into<String>) -> Self {
        Self {
            entries:    HashMap::new(),
            created_at: now_secs(),
            tenant_id:  tenant_id.into(),
            agent_id:   agent_id.into(),
        }
    }

    /// Recursively scan each path in `root_paths` and populate `self.entries`.
    ///
    /// Paths that do not exist are warned about and skipped.  Files that
    /// cannot be read (permission denied, symlink loops) are silently skipped.
    pub fn scan(&mut self, root_paths: &[&str]) -> Result<()> {
        for root in root_paths {
            let p = Path::new(root);
            if !p.exists() {
                warn!(path = root, "baseline scan: path does not exist — skipping");
                continue;
            }
            let before = self.entries.len();
            self.walk(p);
            info!(
                path  = root,
                added = self.entries.len() - before,
                "baseline scan complete"
            );
        }
        Ok(())
    }

    fn walk(&mut self, dir: &Path) {
        let iter = match fs::read_dir(dir) {
            Ok(it) => it,
            Err(e) => {
                warn!(path = %dir.display(), err = %e, "baseline: read_dir failed");
                return;
            }
        };
        for entry in iter.flatten() {
            let path = entry.path();
            if path.is_symlink() {
                // Skip symlinks to avoid loops
                continue;
            }
            if path.is_dir() {
                self.walk(&path);
            } else if path.is_file() {
                if let Some(ent) = make_entry(&path) {
                    self.entries.insert(path, ent);
                }
            }
        }
    }

    /// Compare this baseline against `current`, returning a list of diffs.
    ///
    /// * Files in `current` but not in `self`  → `Added`
    /// * Files in both with different hashes    → `Modified`
    /// * Files in `self` but not in `current`  → `Removed`
    pub fn compare(&self, current: &Baseline) -> Vec<BaselineDiff> {
        let mut diffs = Vec::new();

        // Check for added and modified files
        for (path, cur_ent) in &current.entries {
            match self.entries.get(path) {
                None => {
                    diffs.push(BaselineDiff::Added { path: path.clone() });
                }
                Some(old_ent) if old_ent.hash != cur_ent.hash => {
                    diffs.push(BaselineDiff::Modified {
                        path:     path.clone(),
                        old_hash: old_ent.hash.clone(),
                        new_hash: cur_ent.hash.clone(),
                    });
                }
                _ => {}
            }
        }

        // Check for removed files
        for path in self.entries.keys() {
            if !current.entries.contains_key(path) {
                diffs.push(BaselineDiff::Removed { path: path.clone() });
            }
        }

        diffs
    }

    /// Serialise the baseline to a JSON file at `path`.
    pub fn save(&self, path: &Path) -> Result<()> {
        let json = serde_json::to_string_pretty(self)?;
        fs::write(path, json)?;
        info!(path = %path.display(), entries = self.entries.len(), "baseline saved");
        Ok(())
    }

    /// Deserialise a baseline from a JSON file.
    pub fn load(path: &Path) -> Result<Self> {
        let raw = fs::read_to_string(path)?;
        let bl: Baseline = serde_json::from_str(&raw)?;
        info!(path = %path.display(), entries = bl.entries.len(), "baseline loaded");
        Ok(bl)
    }

    /// Number of files in the baseline.
    pub fn len(&self) -> usize {
        self.entries.len()
    }

    /// Returns `true` if no files have been scanned.
    pub fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn make_entry(path: &Path) -> Option<BaselineEntry> {
    let data = fs::read(path).ok()?;
    let meta = fs::metadata(path).ok()?;

    let mut hasher = Hasher::new();
    hasher.update(&data);
    let hash = hasher.finalize().to_hex().to_string();

    let mtime = meta
        .modified()
        .ok()
        .and_then(|t| t.duration_since(UNIX_EPOCH).ok())
        .map(|d| d.as_secs())
        .unwrap_or(0);

    // Permission bits: on Unix use st_mode; on Windows always 0o644
    #[cfg(unix)]
    let permissions = {
        use std::os::unix::fs::MetadataExt;
        meta.mode() & 0o7777
    };
    #[cfg(not(unix))]
    let permissions: u32 = 0o644;

    Some(BaselineEntry {
        path: path.to_path_buf(),
        hash,
        size: meta.len(),
        mtime,
        permissions,
    })
}

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    fn make_temp_dir(suffix: &str) -> PathBuf {
        let p = std::env::temp_dir().join(format!("kubric_baseline_{}", suffix));
        let _ = fs::create_dir_all(&p);
        p
    }

    #[test]
    fn baseline_empty_on_nonexistent_path() {
        let mut bl = Baseline::new("t1", "a1");
        bl.scan(&["/nonexistent/path/kubric"]).unwrap();
        assert!(bl.is_empty());
    }

    #[test]
    fn baseline_scan_and_len() {
        let dir = make_temp_dir("scan");
        fs::write(dir.join("f1.txt"), b"hello").unwrap();
        fs::write(dir.join("f2.txt"), b"world").unwrap();

        let mut bl = Baseline::new("t1", "a1");
        bl.scan(&[dir.to_str().unwrap()]).unwrap();
        assert_eq!(bl.len(), 2);

        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn baseline_compare_added_modified_removed() {
        let dir = make_temp_dir("compare");
        let f1 = dir.join("unchanged.txt");
        let f2 = dir.join("to_modify.txt");
        let f3 = dir.join("to_remove.txt");

        fs::write(&f1, b"stable").unwrap();
        fs::write(&f2, b"original").unwrap();
        fs::write(&f3, b"will be removed").unwrap();

        let mut old = Baseline::new("t1", "a1");
        old.scan(&[dir.to_str().unwrap()]).unwrap();

        // Mutate the tree
        fs::write(&f2, b"modified").unwrap();   // Modified
        fs::remove_file(&f3).unwrap();           // Removed
        let f4 = dir.join("new_file.txt");
        fs::write(&f4, b"fresh").unwrap();       // Added

        let mut new_bl = Baseline::new("t1", "a1");
        new_bl.scan(&[dir.to_str().unwrap()]).unwrap();

        let diffs = old.compare(&new_bl);

        let modified_count = diffs.iter().filter(|d| matches!(d, BaselineDiff::Modified { .. })).count();
        let removed_count  = diffs.iter().filter(|d| matches!(d, BaselineDiff::Removed  { .. })).count();
        let added_count    = diffs.iter().filter(|d| matches!(d, BaselineDiff::Added    { .. })).count();

        assert_eq!(modified_count, 1, "expected 1 modified file");
        assert_eq!(removed_count,  1, "expected 1 removed file");
        assert_eq!(added_count,    1, "expected 1 added file");

        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn baseline_save_load_round_trip() {
        let dir = make_temp_dir("roundtrip");
        fs::write(dir.join("a.txt"), b"alpha").unwrap();

        let mut bl = Baseline::new("tenant", "agent");
        bl.scan(&[dir.to_str().unwrap()]).unwrap();

        let save_path = dir.join("baseline.json");
        bl.save(&save_path).unwrap();

        let loaded = Baseline::load(&save_path).unwrap();
        assert_eq!(loaded.len(), bl.len());
        assert_eq!(loaded.tenant_id, "tenant");
        assert_eq!(loaded.agent_id, "agent");

        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn baseline_diff_display() {
        let added = BaselineDiff::Added { path: PathBuf::from("/etc/added.conf") };
        assert!(format!("{}", added).contains("ADDED"));

        let removed = BaselineDiff::Removed { path: PathBuf::from("/etc/removed.conf") };
        assert!(format!("{}", removed).contains("REMOVED"));
    }
}
