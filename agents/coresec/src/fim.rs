//! File Integrity Monitoring (FIM) — watches designated paths for changes
//! and publishes OCSF FileActivity (class 4010) events to NATS.
//!
//! Uses the `notify` crate for cross-platform filesystem events and `blake3`
//! for file hashing.  A baseline is captured on startup; subsequent changes
//! are diffed against that baseline.

use anyhow::Result;
use blake3::Hasher;
use notify::{Event, EventKind, RecursiveMode, Watcher};
use serde::Serialize;
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{info, warn};

/// A single FIM event — OCSF FileActivity (class 4010).
#[derive(Debug, Clone, Serialize)]
pub struct FimEvent {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: u64,
    pub class_uid: u32,
    pub activity_id: u8,
    pub path: String,
    pub old_hash: String,
    pub new_hash: String,
    pub severity: String,
}

impl FimEvent {
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{}.endpoint.fim.v1", tenant_id)
    }
}

/// In-memory baseline of file hashes for monitored paths.
pub struct FimEngine {
    baseline: HashMap<PathBuf, String>,
    watch_paths: Vec<PathBuf>,
}

impl FimEngine {
    /// Build a baseline by hashing all regular files under the given paths.
    pub fn new(paths: &[&str]) -> Self {
        let mut baseline = HashMap::new();
        let mut watch_paths = Vec::new();
        for p in paths {
            let path = PathBuf::from(p);
            if path.exists() {
                watch_paths.push(path.clone());
                build_baseline(&path, &mut baseline);
            }
        }
        info!(files = baseline.len(), paths = watch_paths.len(), "FIM baseline captured");
        Self { baseline, watch_paths }
    }

    /// Create an empty engine (no paths to watch).
    pub fn empty() -> Self {
        Self {
            baseline: HashMap::new(),
            watch_paths: Vec::new(),
        }
    }

    /// Check a specific file against the baseline.  Returns the old hash and new hash
    /// if the file changed; None if it hasn't changed or isn't in the baseline.
    pub fn check_file(&mut self, path: &Path) -> Option<(String, String)> {
        let new_hash = hash_file(path)?;
        let old_hash = self.baseline.get(path).cloned().unwrap_or_default();
        if old_hash == new_hash {
            return None;
        }
        self.baseline.insert(path.to_path_buf(), new_hash.clone());
        Some((old_hash, new_hash))
    }

    /// Returns the paths being watched.
    pub fn watch_paths(&self) -> &[PathBuf] {
        &self.watch_paths
    }

    pub fn baseline_count(&self) -> usize {
        self.baseline.len()
    }
}

/// Start the FIM watcher.  Returns a receiver channel that emits (path, old_hash, new_hash)
/// tuples whenever a file is created, modified, or removed.
pub fn start_watcher(
    engine: Arc<tokio::sync::Mutex<FimEngine>>,
) -> Result<mpsc::Receiver<(PathBuf, String, String)>> {
    let (tx, rx) = mpsc::channel(256);

    let engine_clone = engine.clone();
    let (notify_tx, mut notify_rx) = mpsc::channel::<Event>(512);

    // Set up the filesystem watcher
    let mut watcher = notify::recommended_watcher(move |res: Result<Event, notify::Error>| {
        if let Ok(event) = res {
            let _ = notify_tx.blocking_send(event);
        }
    })?;

    // Watch all paths from the engine
    let paths = {
        let eng = engine.blocking_lock();
        eng.watch_paths().to_vec()
    };

    for path in &paths {
        if let Err(e) = watcher.watch(path, RecursiveMode::Recursive) {
            warn!(path = %path.display(), %e, "FIM watch failed for path");
        }
    }

    // Spawn a task to process filesystem events
    tokio::spawn(async move {
        let _watcher = watcher; // keep watcher alive
        while let Some(event) = notify_rx.recv().await {
            match event.kind {
                EventKind::Create(_) | EventKind::Modify(_) | EventKind::Remove(_) => {
                    let mut eng = engine_clone.lock().await;
                    for path in &event.paths {
                        if let Some((old_hash, new_hash)) = eng.check_file(path) {
                            let _ = tx.send((path.clone(), old_hash, new_hash)).await;
                        }
                    }
                }
                _ => {}
            }
        }
    });

    Ok(rx)
}

fn build_baseline(dir: &Path, map: &mut HashMap<PathBuf, String>) {
    let walker = match fs::read_dir(dir) {
        Ok(w) => w,
        Err(_) => return,
    };
    for entry in walker.flatten() {
        let path = entry.path();
        if path.is_dir() {
            build_baseline(&path, map);
        } else if path.is_file() {
            if let Some(h) = hash_file(&path) {
                map.insert(path, h);
            }
        }
    }
}

fn hash_file(path: &Path) -> Option<String> {
    let data = fs::read(path).ok()?;
    let mut hasher = Hasher::new();
    hasher.update(&data);
    Some(hasher.finalize().to_hex().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn fim_engine_empty_baseline() {
        let engine = FimEngine::new(&["/nonexistent/path/that/wont/exist"]);
        assert_eq!(engine.baseline_count(), 0);
    }

    #[test]
    fn fim_engine_detects_change() {
        let dir = std::env::temp_dir().join("kubric_fim_test");
        let _ = fs::create_dir_all(&dir);
        let file = dir.join("test.txt");
        fs::write(&file, b"initial content").unwrap();

        let mut engine = FimEngine::new(&[dir.to_str().unwrap()]);
        assert!(engine.baseline_count() >= 1);

        // No change yet
        assert!(engine.check_file(&file).is_none());

        // Modify file
        let mut f = fs::OpenOptions::new().write(true).truncate(true).open(&file).unwrap();
        f.write_all(b"modified content").unwrap();
        drop(f);

        // Should detect change
        let result = engine.check_file(&file);
        assert!(result.is_some());
        let (old, new) = result.unwrap();
        assert_ne!(old, new);

        // Cleanup
        let _ = fs::remove_dir_all(&dir);
    }
}
