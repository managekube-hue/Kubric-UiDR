# K-SOC-DET-004 -- YARA-X Integration in CoreSec Rust Agent

**License:** BSD-3-Clause — safe to embed as Cargo dependency.  
**Crate:** `yara-x = "0.12"` (VirusTotal's Rust rewrite of YARA)  
**Role:** File and memory scanning for malware signatures on endpoints.

---

## 1. Cargo Dependencies

```toml
# agents/coresec/Cargo.toml
[dependencies]
yara-x = "0.12"
notify = { version = "6", features = ["macos_fsevent"] }
tokio = { version = "1", features = ["full"] }
nats = "0.35"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tracing = "0.1"
walkdir = "2"
blake3 = "1"
chrono = "0.4"
```

---

## 2. Rule Loading and Compilation

```rust
// agents/coresec/src/detections/yara.rs

use std::path::{Path, PathBuf};
use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn, error};
use walkdir::WalkDir;

const RULES_DIR: &str = "vendor/yara-rules";

/// Compiled YARA-X rule set, protected by RwLock for hot-reload.
pub struct YaraEngine {
    rules: Arc<RwLock<yara_x::Rules>>,
    rules_dir: PathBuf,
}

impl YaraEngine {
    /// Load and compile all .yar/.yara files from the vendor directory.
    pub fn new(rules_dir: &str) -> anyhow::Result<Self> {
        let rules = Self::compile_rules(rules_dir)?;
        Ok(Self {
            rules: Arc::new(RwLock::new(rules)),
            rules_dir: PathBuf::from(rules_dir),
        })
    }

    fn compile_rules(rules_dir: &str) -> anyhow::Result<yara_x::Rules> {
        let mut compiler = yara_x::Compiler::new();
        let mut count = 0u32;

        for entry in WalkDir::new(rules_dir)
            .into_iter()
            .filter_map(|e| e.ok())
            .filter(|e| {
                let p = e.path();
                p.extension()
                    .map(|ext| ext == "yar" || ext == "yara")
                    .unwrap_or(false)
            })
        {
            let source = std::fs::read_to_string(entry.path())?;
            match compiler.add_source(source.as_str()) {
                Ok(_) => count += 1,
                Err(e) => {
                    warn!(
                        path = %entry.path().display(),
                        error = %e,
                        "Failed to compile YARA rule, skipping"
                    );
                }
            }
        }

        let rules = compiler.build();
        info!(
            rules_dir = rules_dir,
            compiled = count,
            "YARA-X rules compiled"
        );
        Ok(rules)
    }

    /// Hot-reload rules (called by file watcher).
    pub async fn reload(&self) -> anyhow::Result<()> {
        let new_rules = Self::compile_rules(
            self.rules_dir.to_str().unwrap_or(RULES_DIR),
        )?;
        let mut guard = self.rules.write().await;
        *guard = new_rules;
        info!("YARA rules hot-reloaded");
        Ok(())
    }

    /// Scan a file on disk.
    pub async fn scan_file(&self, path: &Path) -> anyhow::Result<Vec<YaraMatch>> {
        let data = tokio::fs::read(path).await?;
        self.scan_bytes(&data, path.to_string_lossy().as_ref()).await
    }

    /// Scan a byte buffer (file content or memory region).
    pub async fn scan_bytes(
        &self,
        data: &[u8],
        source: &str,
    ) -> anyhow::Result<Vec<YaraMatch>> {
        let rules = self.rules.read().await;
        let rules_clone = rules.clone();
        let data_owned = data.to_vec();
        let source_owned = source.to_string();

        // Offload CPU-intensive scanning to blocking thread pool
        let matches = tokio::task::spawn_blocking(move || {
            let mut scanner = yara_x::Scanner::new(&rules_clone);
            scanner.set_timeout(30); // 30-second timeout per scan
            let results = scanner.scan(&data_owned);

            match results {
                Ok(scan_results) => {
                    scan_results
                        .matching_rules()
                        .map(|rule| YaraMatch {
                            rule_id: rule.identifier().to_string(),
                            namespace: rule.namespace().to_string(),
                            tags: rule.tags()
                                .map(|t| t.identifier().to_string())
                                .collect(),
                            source: source_owned.clone(),
                        })
                        .collect::<Vec<_>>()
                }
                Err(e) => {
                    tracing::error!(error = %e, "YARA scan error");
                    vec![]
                }
            }
        })
        .await?;

        Ok(matches)
    }
}

#[derive(Debug, Clone, serde::Serialize)]
pub struct YaraMatch {
    pub rule_id: String,
    pub namespace: String,
    pub tags: Vec<String>,
    pub source: String,
}
```

---

## 3. File Watcher for Hot-Reload

```rust
// agents/coresec/src/detections/yara_watcher.rs

use notify::{RecommendedWatcher, RecursiveMode, Watcher, EventKind};
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::info;

use super::yara::YaraEngine;

/// Watch the YARA rules directory and trigger hot-reload on changes.
pub async fn watch_rules(engine: Arc<YaraEngine>, rules_dir: &str) -> anyhow::Result<()> {
    let (tx, mut rx) = mpsc::channel(16);

    let mut watcher = RecommendedWatcher::new(
        move |res: Result<notify::Event, notify::Error>| {
            if let Ok(event) = res {
                match event.kind {
                    EventKind::Create(_) | EventKind::Modify(_) | EventKind::Remove(_) => {
                        let _ = tx.blocking_send(());
                    }
                    _ => {}
                }
            }
        },
        notify::Config::default()
            .with_poll_interval(std::time::Duration::from_secs(2)),
    )?;

    watcher.watch(std::path::Path::new(rules_dir), RecursiveMode::Recursive)?;
    info!(dir = rules_dir, "YARA rule file watcher started");

    // Debounce: wait 2s after last change before reloading
    loop {
        rx.recv().await;
        // Drain any pending notifications
        tokio::time::sleep(std::time::Duration::from_secs(2)).await;
        while rx.try_recv().is_ok() {}

        if let Err(e) = engine.reload().await {
            tracing::error!(error = %e, "Failed to reload YARA rules");
        }
    }
}
```

---

## 4. NATS Event Publishing

```rust
// agents/coresec/src/detections/yara_publisher.rs

use chrono::Utc;
use serde_json::json;

use super::yara::YaraMatch;

/// Publish YARA match as OCSF FileActivity event to NATS.
pub async fn publish_yara_match(
    nc: &async_nats::Client,
    tenant_id: &str,
    file_path: &str,
    file_hash: &str,
    matches: &[YaraMatch],
) -> anyhow::Result<()> {
    for m in matches {
        let severity = if m.tags.contains(&"apt".to_string()) {
            5 // Critical
        } else if m.tags.contains(&"malware".to_string()) {
            4 // High
        } else {
            3 // Medium
        };

        let event = json!({
            "class_uid": 4010,       // OCSF FileActivity
            "activity_id": 6,        // File detected (scan match)
            "category_uid": 4,       // Findings
            "severity_id": severity,
            "time": Utc::now().to_rfc3339(),
            "file": {
                "name": file_path,
                "hashes": [{
                    "algorithm_id": 99,   // Blake3
                    "value": file_hash
                }]
            },
            "finding_info": {
                "title": format!("YARA match: {}", m.rule_id),
                "types": ["Malware"],
                "uid": format!("yara-{}-{}", m.rule_id, file_hash),
                "analytic": {
                    "name": &m.rule_id,
                    "type": "YARA",
                    "category": &m.namespace,
                },
                "data_sources": ["File Content Scan"]
            },
            "metadata": {
                "product": {
                    "name": "CoreSec YARA-X",
                    "vendor_name": "Kubric",
                    "version": env!("CARGO_PKG_VERSION")
                },
                "tenant_uid": tenant_id
            },
            "unmapped": {
                "yara_tags": &m.tags,
                "yara_namespace": &m.namespace
            }
        });

        let subject = format!("kubric.edr.file.{}", tenant_id);
        nc.publish(subject, serde_json::to_vec(&event)?.into()).await?;
    }

    Ok(())
}
```

---

## 5. Integration Entry Point

```rust
// agents/coresec/src/detections/mod.rs (excerpt)

use std::sync::Arc;
use tokio::task::JoinHandle;
use walkdir::WalkDir;

mod yara;
mod yara_watcher;
mod yara_publisher;

pub use yara::YaraEngine;

/// Start YARA scanning subsystem.
pub async fn start_yara_subsystem(
    nc: async_nats::Client,
    tenant_id: String,
    rules_dir: &str,
    scan_dirs: &[&str],
) -> anyhow::Result<(Arc<YaraEngine>, JoinHandle<()>)> {
    let engine = Arc::new(YaraEngine::new(rules_dir)?);

    // Start file watcher for hot-reload
    let watcher_engine = engine.clone();
    let rules_dir_owned = rules_dir.to_string();
    let watcher_handle = tokio::spawn(async move {
        if let Err(e) = yara_watcher::watch_rules(watcher_engine, &rules_dir_owned).await {
            tracing::error!(error = %e, "YARA watcher exited");
        }
    });

    // Initial full scan
    let scan_engine = engine.clone();
    let scan_dirs: Vec<String> = scan_dirs.iter().map(|s| s.to_string()).collect();
    let nc_clone = nc.clone();
    let tenant_clone = tenant_id.clone();

    tokio::spawn(async move {
        for dir in &scan_dirs {
            for entry in WalkDir::new(dir)
                .into_iter()
                .filter_map(|e| e.ok())
                .filter(|e| e.file_type().is_file())
            {
                let path = entry.path();
                match scan_engine.scan_file(path).await {
                    Ok(matches) if !matches.is_empty() => {
                        let data = tokio::fs::read(path).await.unwrap_or_default();
                        let hash = blake3::hash(&data).to_hex().to_string();
                        let _ = yara_publisher::publish_yara_match(
                            &nc_clone,
                            &tenant_clone,
                            &path.to_string_lossy(),
                            &hash,
                            &matches,
                        )
                        .await;
                    }
                    Err(e) => {
                        tracing::warn!(
                            path = %path.display(),
                            error = %e,
                            "YARA scan failed"
                        );
                    }
                    _ => {}
                }
            }
        }
    });

    Ok((engine, watcher_handle))
}
```

---

## 6. Rule Update Pipeline

```bash
#!/usr/bin/env bash
# scripts/update-yara-rules.sh
# Sync YARA rules from upstream and vendor them.

set -euo pipefail

RULES_REPO="https://github.com/Yara-Rules/rules.git"
VENDOR_DIR="vendor/yara-rules"
BRANCH="master"

echo "[+] Syncing YARA rules from $RULES_REPO"

if [ -d "$VENDOR_DIR/.git" ]; then
    cd "$VENDOR_DIR"
    git fetch origin "$BRANCH"
    git reset --hard "origin/$BRANCH"
    cd -
else
    rm -rf "$VENDOR_DIR"
    git clone --depth 1 --branch "$BRANCH" "$RULES_REPO" "$VENDOR_DIR"
fi

# Remove test/example rules that cause compilation issues
rm -rf "$VENDOR_DIR/utils/" "$VENDOR_DIR/deprecated/"

# Count rules
RULE_COUNT=$(find "$VENDOR_DIR" -name '*.yar' -o -name '*.yara' | wc -l)
echo "[+] Vendored $RULE_COUNT YARA rules"

# Verify compilation (dry run)
echo "[+] Verifying rule compilation..."
# Uses the coresec binary to validate
if command -v coresec &>/dev/null; then
    coresec yara-validate --rules-dir "$VENDOR_DIR"
fi

echo "[+] YARA rules update complete"
```

---

## 7. Performance Notes

| Metric | Value |
|--------|-------|
| Compilation time | ~2s for 5,000 rules |
| Scan throughput | ~200 MB/s per core (YARA-X) |
| Memory per rule set | ~50 MB for 5,000 compiled rules |
| Async model | `tokio::spawn_blocking` for scan, non-blocking I/O for everything else |
| Timeout | 30s per file scan (configurable) |
| Hot-reload | 2s debounce, <3s full recompile |
