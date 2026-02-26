//! K-XRO-CS-FIM-001 — File Integrity Monitoring using the `notify` crate
//!
//! Watches a set of filesystem paths recursively for Create, Modify, and Remove
//! events.  Each change is hashed with BLAKE3 and published as a `FimEvent`.
//!
//! NATS subject pattern:  `kubric.{tenant_id}.endpoint.fim.v1`
//! OCSF class:            4010 (FileActivity)
//!
//! Production counterpart: `agents/coresec/src/fim.rs`
//!
//! Dependencies required in Cargo.toml:
//!   notify     = "6"
//!   blake3     = "1"
//!   serde      = { version = "1", features = ["derive"] }
//!   tokio      = { version = "1", features = ["full"] }
//!   tracing    = "0.1"
//!
//! Module: K-XRO-CS-FIM-001

use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

use blake3::Hasher;
use notify::{Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};
use serde::{Deserialize, Serialize};
use tokio::sync::{mpsc, Mutex};
use tracing::{info, warn};

// ---------------------------------------------------------------------------
// Event type
// ---------------------------------------------------------------------------

/// Change kind emitted by the FIM watcher.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FimKind {
    Create,
    Modify,
    Delete,
}

impl std::fmt::Display for FimKind {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            FimKind::Create => write!(f, "create"),
            FimKind::Modify => write!(f, "modify"),
            FimKind::Delete => write!(f, "delete"),
        }
    }
}

/// OCSF FileActivity (class 4010) event emitted by the FIM subsystem.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FimEvent {
    /// Kubric tenant identifier.
    pub tenant_id: String,
    /// Agent hostname / identifier.
    pub agent_id: String,
    /// OCSF class UID: 4010.
    pub class_uid: u32,
    /// Unique event identifier (hex-encoded u64 hash).
    pub event_id: String,
    /// Unix timestamp in seconds.
    pub timestamp: u64,
    /// Absolute path of the affected file.
    pub path: String,
    /// What happened to the file.
    pub kind: FimKind,
    /// BLAKE3 hash of the file *after* the event (None for Delete events).
    pub blake3_hash: Option<String>,
    /// Human-readable severity.
    pub severity: String,
}

impl FimEvent {
    /// Produce the NATS subject for this event.
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{}.endpoint.fim.v1", tenant_id)
    }
}

// ---------------------------------------------------------------------------
// Watcher struct
// ---------------------------------------------------------------------------

/// Configuration and state for the FIM watcher.
pub struct FimWatcher {
    /// Paths to watch recursively.
    pub paths: Vec<PathBuf>,
    /// Glob-style prefix patterns that should be excluded (e.g. "/proc").
    pub exclude_patterns: Vec<String>,
    /// In-memory baseline of path → BLAKE3 hash for change detection.
    baseline: Arc<Mutex<HashMap<PathBuf, String>>>,
    /// Tenant and agent identifiers stamped on every event.
    tenant_id: String,
    agent_id: String,
    /// Simple token-bucket: max events per second (events are dropped if full).
    rate_limit: usize,
}

impl FimWatcher {
    /// Construct a new `FimWatcher`.
    ///
    /// * `paths`    — directories (or files) to watch recursively.
    /// * `excludes` — path prefix strings to ignore (e.g. `["/proc", "/sys"]`).
    pub fn new(
        tenant_id: impl Into<String>,
        agent_id: impl Into<String>,
        paths: &[&str],
        excludes: &[&str],
    ) -> Self {
        let watch_paths: Vec<PathBuf> = paths
            .iter()
            .map(|p| PathBuf::from(p))
            .filter(|p| p.exists())
            .collect();

        Self {
            paths: watch_paths,
            exclude_patterns: excludes.iter().map(|s| s.to_string()).collect(),
            baseline: Arc::new(Mutex::new(HashMap::new())),
            tenant_id: tenant_id.into(),
            agent_id: agent_id.into(),
            rate_limit: 500, // default: 500 events/s max channel depth
        }
    }

    /// Set the channel depth used as a soft rate-limit.
    pub fn with_rate_limit(mut self, limit: usize) -> Self {
        self.rate_limit = limit;
        self
    }

    /// Build the initial baseline by hashing all watched files.
    pub async fn build_baseline(&self) {
        let mut bl = self.baseline.lock().await;
        for root in &self.paths {
            walk_and_hash(root, &mut bl, &self.exclude_patterns);
        }
        info!(
            files = bl.len(),
            paths = self.paths.len(),
            "FIM baseline captured"
        );
    }

    /// Start watching.  Returns a `Receiver<FimEvent>` that emits one item per
    /// filesystem change that passes the exclude filter.
    pub async fn start(self) -> anyhow::Result<mpsc::Receiver<FimEvent>> {
        // Build baseline before watching so we can compute diffs
        {
            let mut bl = self.baseline.lock().await;
            for root in &self.paths {
                walk_and_hash(root, &mut bl, &self.exclude_patterns);
            }
            info!(files = bl.len(), "FIM baseline built before watch");
        }

        let (tx, rx) = mpsc::channel::<FimEvent>(self.rate_limit);
        let (notify_tx, mut notify_rx) = mpsc::channel::<Event>(1024);

        // Spawn the synchronous notify watcher on a blocking thread
        let mut watcher: RecommendedWatcher =
            notify::recommended_watcher(move |res: Result<Event, notify::Error>| {
                if let Ok(ev) = res {
                    let _ = notify_tx.blocking_send(ev);
                }
            })?;

        for path in &self.paths {
            if let Err(e) = watcher.watch(path, RecursiveMode::Recursive) {
                warn!(path = %path.display(), err = %e, "FIM: failed to watch path");
            } else {
                info!(path = %path.display(), "FIM: watching");
            }
        }

        // Clone shared state for the async processing task
        let baseline   = Arc::clone(&self.baseline);
        let excludes   = self.exclude_patterns.clone();
        let tenant_id  = self.tenant_id.clone();
        let agent_id   = self.agent_id.clone();

        tokio::spawn(async move {
            // Keep watcher alive inside the task
            let _watcher = watcher;

            while let Some(event) = notify_rx.recv().await {
                let kind = match event.kind {
                    EventKind::Create(_) => FimKind::Create,
                    EventKind::Modify(_) => FimKind::Modify,
                    EventKind::Remove(_) => FimKind::Delete,
                    _ => continue,
                };

                for path in &event.paths {
                    // Skip excluded paths
                    if should_exclude(path, &excludes) {
                        continue;
                    }

                    let mut bl = baseline.lock().await;
                    let new_hash = if kind != FimKind::Delete {
                        compute_hash(path)
                    } else {
                        None
                    };

                    // Skip if hash unchanged (e.g. metadata-only notify event)
                    if kind == FimKind::Modify {
                        let old = bl.get(path).cloned().unwrap_or_default();
                        if let Some(ref nh) = new_hash {
                            if old == *nh {
                                continue;
                            }
                        }
                    }

                    // Update baseline
                    match &kind {
                        FimKind::Delete => { bl.remove(path); }
                        _ => {
                            if let Some(ref h) = new_hash {
                                bl.insert(path.clone(), h.clone());
                            }
                        }
                    }
                    drop(bl);

                    let ts = SystemTime::now()
                        .duration_since(UNIX_EPOCH)
                        .unwrap_or_default()
                        .as_secs();

                    let event_id = format!(
                        "{:016x}",
                        blake3::hash(
                            format!("{}:{}:{}", tenant_id, path.display(), ts).as_bytes()
                        )
                        .as_bytes()
                        .iter()
                        .fold(0u64, |acc, &b| acc.wrapping_add(b as u64))
                    );

                    let fim_ev = FimEvent {
                        tenant_id: tenant_id.clone(),
                        agent_id:  agent_id.clone(),
                        class_uid: 4010,
                        event_id,
                        timestamp: ts,
                        path: path.to_string_lossy().into_owned(),
                        kind: kind.clone(),
                        blake3_hash: new_hash,
                        severity: "medium".to_string(),
                    };

                    if tx.send(fim_ev).await.is_err() {
                        return; // receiver dropped
                    }
                }
            }
        });

        Ok(rx)
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Compute the BLAKE3 hash of a file.  Returns `None` if the file cannot be read.
pub fn compute_hash(path: &Path) -> Option<String> {
    let data = fs::read(path).ok()?;
    let mut hasher = Hasher::new();
    hasher.update(&data);
    Some(hasher.finalize().to_hex().to_string())
}

/// Returns `true` if the path starts with any of the exclude prefixes.
pub fn should_exclude(path: &Path, patterns: &[String]) -> bool {
    let path_str = path.to_string_lossy();
    patterns.iter().any(|p| path_str.starts_with(p.as_str()))
}

/// Walk a directory tree, hashing every regular file and inserting into `map`.
fn walk_and_hash(dir: &Path, map: &mut HashMap<PathBuf, String>, excludes: &[String]) {
    if should_exclude(dir, excludes) {
        return;
    }
    let entries = match fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            walk_and_hash(&p, map, excludes);
        } else if p.is_file() {
            if let Some(h) = compute_hash(&p) {
                map.insert(p, h);
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    fn default_excludes() -> Vec<String> {
        vec![
            "/proc".into(),
            "/sys".into(),
            "/dev".into(),
            "/run".into(),
        ]
    }

    #[test]
    fn should_exclude_proc() {
        let excludes = default_excludes();
        assert!(should_exclude(Path::new("/proc/1/maps"), &excludes));
        assert!(should_exclude(Path::new("/sys/fs/bpf"), &excludes));
        assert!(!should_exclude(Path::new("/etc/passwd"), &excludes));
        assert!(!should_exclude(Path::new("/var/log/syslog"), &excludes));
    }

    #[test]
    fn compute_hash_consistent() {
        let dir = std::env::temp_dir().join("kubric_fim_hash_test");
        let _ = fs::create_dir_all(&dir);
        let file = dir.join("sample.txt");
        fs::write(&file, b"stable content").unwrap();

        let h1 = compute_hash(&file).unwrap();
        let h2 = compute_hash(&file).unwrap();
        assert_eq!(h1, h2);
        assert_eq!(h1.len(), 64); // BLAKE3 hex is 64 chars

        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn compute_hash_different_content() {
        let dir = std::env::temp_dir().join("kubric_fim_hash_diff");
        let _ = fs::create_dir_all(&dir);
        let f1 = dir.join("a.txt");
        let f2 = dir.join("b.txt");
        fs::write(&f1, b"alpha").unwrap();
        fs::write(&f2, b"beta").unwrap();

        assert_ne!(compute_hash(&f1), compute_hash(&f2));
        let _ = fs::remove_dir_all(&dir);
    }

    #[test]
    fn fim_event_nats_subject() {
        assert_eq!(
            FimEvent::nats_subject("acme"),
            "kubric.acme.endpoint.fim.v1"
        );
    }

    #[test]
    fn fim_kind_display() {
        assert_eq!(format!("{}", FimKind::Create), "create");
        assert_eq!(format!("{}", FimKind::Modify), "modify");
        assert_eq!(format!("{}", FimKind::Delete), "delete");
    }
}
