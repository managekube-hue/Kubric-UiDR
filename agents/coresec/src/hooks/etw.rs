//! Windows hook provider for process activity.
//!
//! This implementation emits process-exec style events by tracking process table
//! deltas with `sysinfo` and pushing events to the same hook channel used by
//! Linux eBPF providers.

#![cfg(target_os = "windows")]

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use sysinfo::System;
use tokio::sync::mpsc;
use tracing::{error, info};

use super::HookEvent;

/// ETW-based kernel hook provider for Windows.
pub struct EtwProvider {
    running: Arc<AtomicBool>,
}

impl EtwProvider {
    pub fn new() -> anyhow::Result<Self> {
        info!("Windows process hook provider initialised");
        Ok(Self {
            running: Arc::new(AtomicBool::new(false)),
        })
    }
}

impl EtwProvider {
    pub async fn start(&self) -> anyhow::Result<mpsc::Receiver<HookEvent>> {
        let (tx, rx) = mpsc::channel(4096);
        let running = self.running.clone();
        running.store(true, Ordering::SeqCst);

        // Spawn ETW consumer on a blocking thread (ProcessTrace blocks)
        let tx_clone = tx.clone();
        let running_clone = running.clone();

        tokio::task::spawn_blocking(move || {
            if let Err(e) = run_process_event_loop(tx_clone, running_clone) {
                error!(%e, "Windows process event loop failed");
            }
        });

        info!("Windows process hook provider started");
        Ok(rx)
    }

    fn name(&self) -> &'static str {
        "ETW/Kernel-Process"
    }
}

fn snapshot_processes(sys: &System) -> HashMap<u32, (u32, String, String)> {
    let mut map = HashMap::with_capacity(sys.processes().len());
    for (pid, process) in sys.processes() {
        let pid_u32 = pid.as_u32();
        let ppid = process.parent().map(|v| v.as_u32()).unwrap_or(0);
        let executable = process
            .exe()
            .map(|p| p.display().to_string())
            .unwrap_or_else(|| process.name().to_string_lossy().to_string());
        let cmdline = process
            .cmd()
            .iter()
            .map(|arg| arg.to_string_lossy().to_string())
            .collect::<Vec<_>>()
            .join(" ");

        map.insert(pid_u32, (ppid, executable, cmdline));
    }
    map
}

fn run_process_event_loop(
    tx: mpsc::Sender<HookEvent>,
    running: Arc<AtomicBool>,
) -> anyhow::Result<()> {
    let mut sys = System::new_all();
    sys.refresh_all();
    let mut previous = snapshot_processes(&sys);

    info!("Windows process event loop active");

    while running.load(Ordering::SeqCst) {
        std::thread::sleep(Duration::from_millis(500));
        sys.refresh_all();

        let current = snapshot_processes(&sys);

        let ts = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos() as u64;

        for (pid, (ppid, executable, cmdline)) in &current {
            if previous.contains_key(pid) {
                continue;
            }

            let event = HookEvent::ProcessExec {
                pid: *pid,
                ppid: *ppid,
                executable: executable.clone(),
                cmdline: cmdline.clone(),
                uid: 0,
                timestamp_ns: ts,
            };

            if tx.blocking_send(event).is_err() {
                return Ok(());
            }
        }

        previous = current;
    }

    info!("Windows process event loop stopped");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn etw_provider_creates() {
        let provider = EtwProvider::new();
        assert!(provider.is_ok());
    }

    #[test]
    fn etw_provider_name() {
        let provider = EtwProvider::new().unwrap();
        assert_eq!(provider.name(), "ETW/Kernel-Process");
    }
}
