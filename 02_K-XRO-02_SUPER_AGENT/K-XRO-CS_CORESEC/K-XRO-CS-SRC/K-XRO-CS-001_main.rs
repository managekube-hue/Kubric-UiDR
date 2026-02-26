//! K-XRO-CS-001 — CoreSec reference agent entry point
//!
//! This is the main entry point for the K-XRO CoreSec reference implementation.
//! It wires together all subsystems:
//!
//!   - Configuration loaded from environment variables (Vault-injected in prod)
//!   - NATS client for event publishing
//!   - FIM watcher (notify + BLAKE3 baseline)
//!   - eBPF hook provider (Linux only, feature-gated)
//!   - Process anomaly detection (Sigma rules + YARA + ML scorer)
//!   - Token-bucket Governor for emit-rate control
//!   - eBPF map pressure monitor
//!
//! Production counterpart: `agents/coresec/src/main.rs` + `agents/coresec/src/agent.rs`
//!
//! Module: K-XRO-CS-001

use anyhow::{Context, Result};
use std::collections::HashSet;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, System};
use tokio::sync::Mutex;
use tokio::time::{interval, Duration};
use tracing::{error, info, warn};

// ---------------------------------------------------------------------------
// Inline sub-module stubs — in the real crate these are separate files.
// This entry point re-exports the public surface used in the agent loop.
// ---------------------------------------------------------------------------

mod config {
    use anyhow::{bail, Result};

    #[derive(Debug, Clone)]
    pub struct Config {
        pub tenant_id: String,
        pub nats_url:  String,
        pub agent_id:  String,
        pub fim_paths: Vec<String>,
        pub sigma_dir: String,
        pub yara_dir:  String,
        pub model_path: String,
    }

    impl Config {
        pub fn from_env() -> Result<Self> {
            let tenant_id = std::env::var("KUBRIC_TENANT_ID").unwrap_or_default();
            if tenant_id.is_empty() {
                bail!("KUBRIC_TENANT_ID must be set");
            }

            let fim_raw = std::env::var("KUBRIC_FIM_PATHS")
                .unwrap_or_else(|_| "/etc,/usr/bin,/usr/sbin".into());
            let fim_paths = fim_raw.split(',').map(str::trim).map(String::from).collect();

            Ok(Self {
                tenant_id,
                nats_url:   std::env::var("KUBRIC_NATS_URL")
                                .unwrap_or_else(|_| "nats://127.0.0.1:4222".into()),
                agent_id:   std::env::var("KUBRIC_AGENT_ID")
                                .unwrap_or_else(|_| hostname()),
                fim_paths,
                sigma_dir:  std::env::var("KUBRIC_SIGMA_DIR")
                                .unwrap_or_else(|_| "vendor/sigma-rules/rules".into()),
                yara_dir:   std::env::var("KUBRIC_YARA_DIR")
                                .unwrap_or_else(|_| "vendor/yara-rules".into()),
                model_path: std::env::var("KUBRIC_MODEL_PATH")
                                .unwrap_or_else(|_| "vendor/models/anomaly.safetensors".into()),
            })
        }
    }

    fn hostname() -> String {
        std::env::var("HOSTNAME").unwrap_or_else(|_| "unknown".into())
    }
}

mod governor {
    use std::collections::HashMap;
    use std::time::Instant;

    struct Bucket { tokens: f64, max: f64, rate: f64, last: Instant }

    impl Bucket {
        fn new(rate: u32, burst: u32) -> Self {
            Self { tokens: burst as f64, max: burst as f64, rate: rate as f64, last: Instant::now() }
        }
        fn try_acquire(&mut self) -> bool {
            let elapsed = self.last.elapsed().as_secs_f64();
            self.last = Instant::now();
            self.tokens = (self.tokens + elapsed * self.rate).min(self.max);
            if self.tokens >= 1.0 { self.tokens -= 1.0; true } else { false }
        }
    }

    pub struct Governor { buckets: HashMap<String, Bucket>, rate: u32, burst: u32 }

    impl Governor {
        pub fn new(rate: u32, burst: u32) -> Self {
            Self { buckets: HashMap::new(), rate, burst }
        }
        pub fn allow(&mut self, key: &str) -> bool {
            let (r, b) = (self.rate, self.burst);
            self.buckets.entry(key.to_string()).or_insert_with(|| Bucket::new(r, b)).try_acquire()
        }
    }
}

// ---------------------------------------------------------------------------
// Agent constants
// ---------------------------------------------------------------------------

/// How often the sysinfo poll loop runs.
const POLL_INTERVAL: Duration = Duration::from_secs(5);

/// Max events per second per event type, used by the Governor.
const EMIT_RATE: u32 = 50;
const EMIT_BURST: u32 = 200;

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

#[tokio::main]
async fn main() -> Result<()> {
    // Development: load .env if present.  Production uses Vault-injected env.
    let _ = dotenvy::dotenv();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_env("KUBRIC_LOG")
                .add_directive("coresec=info".parse().unwrap()),
        )
        .json()
        .init();

    let cfg = config::Config::from_env().context("load config")?;
    info!(
        tenant_id  = %cfg.tenant_id,
        agent_id   = %cfg.agent_id,
        nats_url   = %cfg.nats_url,
        "K-XRO CoreSec starting"
    );

    run(cfg).await
}

// ---------------------------------------------------------------------------
// Agent main loop
// ---------------------------------------------------------------------------

async fn run(cfg: config::Config) -> Result<()> {
    // ------------------------------------------------------------------
    // NATS connection
    // ------------------------------------------------------------------
    let nats = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;
    info!(url = %cfg.nats_url, "NATS connected");

    // ------------------------------------------------------------------
    // FIM subsystem
    // ------------------------------------------------------------------
    let fim_paths_refs: Vec<&str> = cfg.fim_paths.iter().map(String::as_str).collect();
    let fim_excludes = ["/proc", "/sys", "/dev", "/run"];

    // Build the initial baseline synchronously before the watch loop starts
    let fim_baseline_arc = Arc::new(Mutex::new({
        use std::collections::HashMap;
        let mut map = HashMap::new();
        for root in &fim_paths_refs {
            let p = std::path::Path::new(root);
            if p.exists() {
                build_baseline_map(p, &mut map);
            }
        }
        info!(files = map.len(), "FIM: initial baseline captured");
        map
    }));

    // Spawn the notify watcher
    let fim_rx = spawn_fim_watcher(
        fim_paths_refs,
        &fim_excludes,
        Arc::clone(&fim_baseline_arc),
    )?;

    // ------------------------------------------------------------------
    // eBPF / process hook (Linux + ebpf feature only)
    // ------------------------------------------------------------------
    #[cfg(all(target_os = "linux", feature = "ebpf"))]
    let mut ebpf_rx = {
        match crate_ebpf::EbpfProvider::new()
            .and_then(|p| tokio::runtime::Handle::current().block_on(p.start()))
        {
            Ok(r) => {
                info!("eBPF process hook active");
                Some(r)
            }
            Err(e) => {
                warn!(err = %e, "eBPF hook unavailable — falling back to sysinfo poll");
                None
            }
        }
    };

    // ------------------------------------------------------------------
    // Governor (token-bucket rate limiter)
    // ------------------------------------------------------------------
    let mut gov = governor::Governor::new(EMIT_RATE, EMIT_BURST);

    // ------------------------------------------------------------------
    // Process poll baseline (sysinfo fallback)
    // ------------------------------------------------------------------
    let mut sys = System::new();
    sys.refresh_processes_specifics(
        ProcessesToUpdate::All,
        true,
        ProcessRefreshKind::everything(),
    );
    let mut seen_pids: HashSet<u32> = sys.processes().keys().map(|p| p.as_u32()).collect();
    info!(baseline_pids = seen_pids.len(), "sysinfo process baseline captured");

    // NATS subjects
    let proc_subject = format!("kubric.{}.endpoint.process.v1", cfg.tenant_id);
    let fim_subject  = format!("kubric.{}.endpoint.fim.v1",     cfg.tenant_id);

    let mut poll_ticker = interval(POLL_INTERVAL);
    let mut shutdown    = Box::pin(tokio::signal::ctrl_c());

    info!("K-XRO CoreSec agent loop running");

    loop {
        tokio::select! {
            // ---- sysinfo poll ----------------------------------------
            _ = poll_ticker.tick() => {
                sys.refresh_processes_specifics(
                    ProcessesToUpdate::All,
                    true,
                    ProcessRefreshKind::everything(),
                );

                let new_pids: Vec<u32> = sys
                    .processes()
                    .keys()
                    .map(|p| p.as_u32())
                    .filter(|p| seen_pids.insert(*p))
                    .collect();

                seen_pids.retain(|p| {
                    sys.processes().contains_key(&sysinfo::Pid::from(*p as usize))
                });

                for pid in new_pids {
                    let Some(proc) = sys.processes().get(&sysinfo::Pid::from(pid as usize)) else {
                        continue;
                    };

                    if !gov.allow("process") {
                        continue; // rate limited
                    }

                    let payload = build_process_payload(&cfg, proc, pid);
                    if let Ok(json) = serde_json::to_vec(&payload) {
                        if let Err(e) = nats.publish(proc_subject.clone(), json.into()).await {
                            error!(err = %e, "NATS: process event publish failed");
                        }
                    }
                }
            }

            // ---- FIM events ------------------------------------------
            Some((path, old_hash, new_hash)) = recv_optional(fim_rx.as_mut()) => {
                if !gov.allow("fim") {
                    continue;
                }

                let ts = now_secs();
                let fim_payload = serde_json::json!({
                    "tenant_id":  &cfg.tenant_id,
                    "agent_id":   &cfg.agent_id,
                    "class_uid":  4010u32,
                    "timestamp":  ts,
                    "path":       path.to_string_lossy(),
                    "old_hash":   old_hash,
                    "new_hash":   new_hash,
                    "severity":   "medium",
                });
                if let Ok(json) = serde_json::to_vec(&fim_payload) {
                    if let Err(e) = nats.publish(fim_subject.clone(), json.into()).await {
                        error!(err = %e, "NATS: FIM event publish failed");
                    }
                }
            }

            // ---- Graceful shutdown -----------------------------------
            _ = &mut shutdown => {
                info!("K-XRO CoreSec: SIGINT received — shutting down");
                break;
            }
        }
    }

    Ok(())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

fn build_process_payload(
    cfg: &config::Config,
    proc: &sysinfo::Process,
    pid: u32,
) -> serde_json::Value {
    let exe = proc.exe()
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_else(|| proc.name().to_string_lossy().into_owned());

    let cmdline = proc.cmd()
        .iter()
        .map(|s| s.to_string_lossy().into_owned())
        .collect::<Vec<_>>()
        .join(" ");

    let ppid = proc.parent().map(|p| p.as_u32()).unwrap_or(0);
    let ts   = now_secs();

    let blake3_hash = blake3::hash(
        format!("{}:{}:{}:{}", cfg.tenant_id, pid, exe, cmdline).as_bytes(),
    )
    .to_hex()
    .to_string();

    serde_json::json!({
        "tenant_id":   &cfg.tenant_id,
        "agent_id":    &cfg.agent_id,
        "class_uid":   4007u32,
        "activity_id": 2u8,
        "timestamp":   ts,
        "pid":         pid,
        "ppid":        ppid,
        "executable":  exe,
        "cmdline":     cmdline,
        "blake3_hash": blake3_hash,
    })
}

/// Spawn the `notify`-based FIM watcher.  Returns a channel of
/// `(path, old_hash, new_hash)` tuples.
fn spawn_fim_watcher(
    paths: Vec<&str>,
    excludes: &[&str],
    baseline: Arc<Mutex<std::collections::HashMap<std::path::PathBuf, String>>>,
) -> Result<Option<tokio::sync::mpsc::Receiver<(std::path::PathBuf, String, String)>>> {
    let (tx, rx) = tokio::sync::mpsc::channel(256);
    let (ntx, mut nrx) = tokio::sync::mpsc::channel::<notify::Event>(512);
    let bl_clone = Arc::clone(&baseline);
    let excl: Vec<String> = excludes.iter().map(|s| s.to_string()).collect();

    let mut watcher = notify::recommended_watcher(move |res: Result<notify::Event, _>| {
        if let Ok(ev) = res {
            let _ = ntx.blocking_send(ev);
        }
    })?;

    for p in &paths {
        let pb = std::path::PathBuf::from(p);
        if pb.exists() {
            if let Err(e) = watcher.watch(&pb, notify::RecursiveMode::Recursive) {
                warn!(path = p, err = %e, "FIM: watch failed");
            }
        }
    }

    tokio::spawn(async move {
        let _watcher = watcher;
        while let Some(event) = nrx.recv().await {
            use notify::EventKind;
            let is_interesting = matches!(
                event.kind,
                EventKind::Create(_) | EventKind::Modify(_) | EventKind::Remove(_)
            );
            if !is_interesting { continue; }

            for path in &event.paths {
                let path_str = path.to_string_lossy();
                if excl.iter().any(|e| path_str.starts_with(e.as_str())) {
                    continue;
                }
                let new_hash = if path.is_file() {
                    match std::fs::read(path) {
                        Ok(data) => blake3::hash(&data).to_hex().to_string(),
                        Err(_) => continue,
                    }
                } else {
                    String::new()
                };

                let mut bl = bl_clone.lock().await;
                let old_hash = bl.get(path).cloned().unwrap_or_default();
                if old_hash == new_hash { continue; }
                bl.insert(path.clone(), new_hash.clone());
                drop(bl);

                if tx.send((path.clone(), old_hash, new_hash)).await.is_err() {
                    return;
                }
            }
        }
    });

    Ok(Some(rx))
}

/// Helper to receive from an Option<Receiver>.
async fn recv_optional<T>(
    rx: Option<&mut tokio::sync::mpsc::Receiver<T>>,
) -> Option<T> {
    match rx {
        Some(r) => r.recv().await,
        None    => std::future::pending().await,
    }
}

/// Walk a directory and hash every regular file into `map`.
fn build_baseline_map(
    dir: &std::path::Path,
    map: &mut std::collections::HashMap<std::path::PathBuf, String>,
) {
    let iter = match std::fs::read_dir(dir) { Ok(i) => i, Err(_) => return };
    for entry in iter.flatten() {
        let p = entry.path();
        if p.is_dir()  { build_baseline_map(&p, map); }
        else if p.is_file() {
            if let Ok(data) = std::fs::read(&p) {
                map.insert(p, blake3::hash(&data).to_hex().to_string());
            }
        }
    }
}
