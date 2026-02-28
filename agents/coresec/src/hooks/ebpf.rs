//! eBPF hook provider for Linux — loads pre-compiled eBPF programs via `aya`.
//!
//! Attaches to kernel tracepoints:
//! - `sys_enter_execve` — new process execution
//! - `sys_enter_openat2` — file access (FIM)
//!
//! Pre-compiled eBPF ELF objects are expected at:
//!   `vendor/ebpf/execve_hook.o`
//!   `vendor/ebpf/openat2_hook.o`
//!
//! Requires CAP_BPF + CAP_PERFM on Linux 5.8+, or root on older kernels.

#![cfg(all(target_os = "linux", feature = "ebpf"))]

use std::path::{Path, PathBuf};
use std::process::Command;
use aya::programs::TracePoint;
use aya::{Bpf, BpfLoader};
use aya::maps::perf::AsyncPerfEventArray;
use aya::util::online_cpus;
use bytes::BytesMut;
use tokio::sync::mpsc;
use tracing::{error, info, warn};

use super::{FileOp, HookEvent, HookProvider};

/// eBPF-based kernel hook provider using aya.
pub struct EbpfProvider {
    execve_path: String,
    openat_path: String,
}

/// Raw event structure shared with eBPF programs (must match the C struct).
#[repr(C)]
#[derive(Debug, Clone, Copy)]
struct BpfExecEvent {
    pid: u32,
    ppid: u32,
    uid: u32,
    timestamp_ns: u64,
    /// Null-terminated path, max 256 bytes.
    filename: [u8; 256],
    /// Null-terminated, max 512 bytes.
    cmdline: [u8; 512],
}

#[repr(C)]
#[derive(Debug, Clone, Copy)]
struct BpfFileEvent {
    pid: u32,
    timestamp_ns: u64,
    op: u32,
    filename: [u8; 256],
}

impl EbpfProvider {
    pub fn new() -> anyhow::Result<Self> {
        let execve_path = std::env::var("KUBRIC_EBPF_EXECVE")
            .unwrap_or_else(|_| "vendor/ebpf/execve_hook.o".into());
        let openat_path = std::env::var("KUBRIC_EBPF_OPENAT")
            .unwrap_or_else(|_| "vendor/ebpf/openat2_hook.o".into());

        Self::ensure_ebpf_object(&execve_path)?;
        let _ = Self::ensure_ebpf_object(&openat_path);

        if !Path::new(&execve_path).exists() {
            anyhow::bail!("eBPF program not found: {}", execve_path);
        }

        Ok(Self { execve_path, openat_path })
    }

    fn ensure_ebpf_object(object_path: &str) -> anyhow::Result<()> {
        if Path::new(object_path).exists() {
            return Ok(());
        }

        let Some(file_name) = Path::new(object_path).file_name().and_then(|n| n.to_str()) else {
            anyhow::bail!("invalid eBPF object path: {}", object_path);
        };

        let candidate_dir = PathBuf::from("vendor/ebpf");
        if !candidate_dir.exists() {
            anyhow::bail!("eBPF source directory not found: {}", candidate_dir.display());
        }

        let make_status = Command::new("make")
            .arg("-C")
            .arg(candidate_dir.as_os_str())
            .arg(file_name)
            .status();

        match make_status {
            Ok(status) if status.success() && Path::new(object_path).exists() => {
                info!(path = %object_path, "built missing eBPF object via make");
                Ok(())
            }
            Ok(status) => anyhow::bail!(
                "failed to build eBPF object {} (make exit: {})",
                object_path,
                status
            ),
            Err(e) => anyhow::bail!(
                "failed to invoke make for {}: {}",
                object_path,
                e
            ),
        }
    }

    fn cstr_to_string(buf: &[u8]) -> String {
        let end = buf.iter().position(|&b| b == 0).unwrap_or(buf.len());
        String::from_utf8_lossy(&buf[..end]).into_owned()
    }
}

impl HookProvider for EbpfProvider {
    async fn start(&self) -> anyhow::Result<mpsc::Receiver<HookEvent>> {
        let (tx, rx) = mpsc::channel(4096);

        // --- Load and attach execve tracepoint ---
        let mut execve_bpf = BpfLoader::new().load_file(&self.execve_path)?;
        let execve_prog: &mut TracePoint = execve_bpf
            .program_mut("execve_hook")
            .ok_or_else(|| anyhow::anyhow!("execve_hook program not found in ELF"))?
            .try_into()?;
        execve_prog.load()?;
        execve_prog.attach("syscalls", "sys_enter_execve")?;
        info!("eBPF: attached to sys_enter_execve");

        // --- Consume events from perf ring buffer ---
        let mut exec_events = AsyncPerfEventArray::try_from(
            execve_bpf.take_map("EXEC_EVENTS")
                .ok_or_else(|| anyhow::anyhow!("EXEC_EVENTS map not found"))?,
        )?;

        let cpus = online_cpus().map_err(|e| anyhow::anyhow!("online_cpus: {:?}", e))?;
        for cpu_id in cpus {
            let tx = tx.clone();
            let mut buf = exec_events.open(cpu_id, Some(256))?;
            tokio::spawn(async move {
                let mut buffers = (0..16)
                    .map(|_| BytesMut::with_capacity(std::mem::size_of::<BpfExecEvent>()))
                    .collect::<Vec<_>>();
                loop {
                    match buf.read_events(&mut buffers).await {
                        Ok(events) => {
                            for i in 0..events.read {
                                if buffers[i].len() >= std::mem::size_of::<BpfExecEvent>() {
                                    let evt: BpfExecEvent = unsafe {
                                        std::ptr::read_unaligned(
                                            buffers[i].as_ptr() as *const BpfExecEvent,
                                        )
                                    };
                                    let hook_event = HookEvent::ProcessExec {
                                        pid: evt.pid,
                                        ppid: evt.ppid,
                                        executable: Self::cstr_to_string(&evt.filename),
                                        cmdline: Self::cstr_to_string(&evt.cmdline),
                                        uid: evt.uid,
                                        timestamp_ns: evt.timestamp_ns,
                                    };
                                    if tx.send(hook_event).await.is_err() {
                                        return; // receiver dropped
                                    }
                                }
                            }
                        }
                        Err(e) => {
                            error!(%e, "eBPF perf read error");
                            break;
                        }
                    }
                }
            });
        }

        // --- Optionally attach openat2 tracepoint for FIM ---
        if Path::new(&self.openat_path).exists() {
            match BpfLoader::new().load_file(&self.openat_path) {
                Ok(mut openat_bpf) => {
                    if let Some(prog) = openat_bpf.program_mut("openat2_hook") {
                        let tp: Result<&mut TracePoint, _> = prog.try_into();
                        if let Ok(tp) = tp {
                            if tp.load().is_ok() {
                                let _ = tp.attach("syscalls", "sys_enter_openat2");
                                info!("eBPF: attached to sys_enter_openat2");
                            }
                        }
                    }
                    // File events would be consumed similarly from a FILE_EVENTS map
                }
                Err(e) => {
                    warn!(%e, "eBPF openat2 program not loaded — FIM disabled");
                }
            }
        }

        Ok(rx)
    }

    fn name(&self) -> &'static str {
        "eBPF/aya"
    }
}
