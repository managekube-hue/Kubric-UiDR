//! K-XRO-CS-EBPF-001 — eBPF execve syscall hook (K-XRO CoreSec reference module)
//!
//! Standalone eBPF provider using the `aya` crate.  Attaches to the Linux kernel
//! `sys_enter_execve` tracepoint and streams process-execution events through a
//! Tokio MPSC channel.
//!
//! Production counterpart: `agents/coresec/src/hooks/ebpf.rs`
//!
//! Kernel requirements: Linux 5.8+, CAP_BPF + CAP_PERFMON (or root).
//! Pre-compiled eBPF ELF objects expected at:
//!   vendor/ebpf/execve_hook.o   (set via env KUBRIC_EBPF_EXECVE)
//!   vendor/ebpf/openat2_hook.o  (set via env KUBRIC_EBPF_OPENAT)
//!
//! Module: K-XRO-CS-EBPF-001

#![cfg(all(target_os = "linux", feature = "ebpf"))]

use std::path::Path;

use aya::maps::perf::AsyncPerfEventArray;
use aya::programs::TracePoint;
use aya::util::online_cpus;
use aya::{Bpf, BpfLoader};
use bytes::BytesMut;
use tokio::sync::mpsc;
use tracing::{error, info, warn};

// ---------------------------------------------------------------------------
// Public event type
// ---------------------------------------------------------------------------

/// A single process-execution event decoded from the eBPF perf ring buffer.
#[derive(Debug, Clone)]
pub struct ProcessEvent {
    /// PID of the newly exec'd process.
    pub pid: u32,
    /// Parent PID at execve time.
    pub ppid: u32,
    /// Effective UID of the calling process.
    pub uid: u32,
    /// Kernel-monotonic timestamp in nanoseconds (CLOCK_MONOTONIC).
    pub timestamp_ns: u64,
    /// Absolute path handed to execve(2).
    pub executable: String,
    /// Reconstructed argv joined with spaces (up to 512 bytes raw).
    pub cmdline: String,
}

// ---------------------------------------------------------------------------
// Kernel-layout struct — must match execve_hook.c exactly (no padding/reorder)
// ---------------------------------------------------------------------------

/// Raw event as written by the eBPF program into the EXEC_EVENTS perf map.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
struct BpfExecEvent {
    pid: u32,
    ppid: u32,
    uid: u32,
    timestamp_ns: u64,
    /// Null-terminated executable path, padded to 256 bytes.
    filename: [u8; 256],
    /// Null-terminated argv string, padded to 512 bytes.
    cmdline: [u8; 512],
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

/// eBPF-backed execve hook provider.
///
/// Loads a pre-compiled eBPF ELF, attaches it to `sys_enter_execve`, and
/// spawns one async reader task per online CPU to drain the perf ring buffer.
pub struct EbpfProvider {
    /// Filesystem path to the execve eBPF ELF object.
    pub execve_path: String,
    /// Filesystem path to the openat2 eBPF ELF object.
    pub openat_path: String,
}

impl EbpfProvider {
    /// Construct an `EbpfProvider` from environment variables.
    ///
    /// Falls back to `vendor/ebpf/execve_hook.o` / `vendor/ebpf/openat2_hook.o`
    /// if the environment variables are not set.
    ///
    /// # Errors
    /// Returns `Err` when the execve ELF object file is not present on disk.
    pub fn new() -> anyhow::Result<Self> {
        let execve_path = std::env::var("KUBRIC_EBPF_EXECVE")
            .unwrap_or_else(|_| "vendor/ebpf/execve_hook.o".into());
        let openat_path = std::env::var("KUBRIC_EBPF_OPENAT")
            .unwrap_or_else(|_| "vendor/ebpf/openat2_hook.o".into());

        if !Path::new(&execve_path).exists() {
            anyhow::bail!(
                "eBPF ELF object not found at '{}'. \
                 Build with 'cargo xtask build-ebpf' or set KUBRIC_EBPF_EXECVE.",
                execve_path
            );
        }

        Ok(Self { execve_path, openat_path })
    }

    /// Load the eBPF program, attach it to `sys_enter_execve`, and begin
    /// streaming events.
    ///
    /// Returns a `Receiver<ProcessEvent>` that yields one item per execve call
    /// observed by the kernel on any CPU.
    pub async fn start(&self) -> anyhow::Result<mpsc::Receiver<ProcessEvent>> {
        let (tx, rx) = mpsc::channel::<ProcessEvent>(4096);

        // ------------------------------------------------------------------
        // Load object file and attach tracepoint
        // ------------------------------------------------------------------
        let mut bpf: Bpf = BpfLoader::new().load_file(&self.execve_path)?;

        {
            let prog: &mut TracePoint = bpf
                .program_mut("execve_hook")
                .ok_or_else(|| {
                    anyhow::anyhow!(
                        "section 'execve_hook' not found in '{}'",
                        self.execve_path
                    )
                })?
                .try_into()?;
            prog.load()?;
            prog.attach("syscalls", "sys_enter_execve")?;
        }
        info!(path = %self.execve_path, "eBPF: attached tracepoint sys_enter_execve");

        // ------------------------------------------------------------------
        // Open the EXEC_EVENTS perf ring-buffer map
        // ------------------------------------------------------------------
        let mut exec_events: AsyncPerfEventArray<_> = AsyncPerfEventArray::try_from(
            bpf.take_map("EXEC_EVENTS")
                .ok_or_else(|| anyhow::anyhow!("EXEC_EVENTS map not found in ELF"))?,
        )?;

        let cpus = online_cpus()
            .map_err(|e| anyhow::anyhow!("failed to enumerate online CPUs: {:?}", e))?;

        info!(cpu_count = cpus.len(), "eBPF: spawning per-CPU ring-buffer readers");

        // ------------------------------------------------------------------
        // Spawn one reader task per CPU
        // ------------------------------------------------------------------
        for cpu_id in cpus {
            // 256-page (1 MiB) ring buffer per CPU
            let mut ring = exec_events.open(cpu_id, Some(256))?;
            let tx_cpu = tx.clone();

            tokio::spawn(async move {
                let event_size = std::mem::size_of::<BpfExecEvent>();
                // 16 receive slots per poll; reused across iterations
                let mut buffers: Vec<BytesMut> = (0..16)
                    .map(|_| BytesMut::with_capacity(event_size))
                    .collect();

                loop {
                    match ring.read_events(&mut buffers).await {
                        Ok(events) => {
                            for i in 0..events.read {
                                if buffers[i].len() < event_size {
                                    warn!(
                                        cpu = cpu_id,
                                        got = buffers[i].len(),
                                        need = event_size,
                                        "eBPF buffer too short — discarding event"
                                    );
                                    continue;
                                }
                                // SAFETY: length verified above; layout matches the
                                // C struct written by the eBPF program.
                                let raw: BpfExecEvent = unsafe {
                                    std::ptr::read_unaligned(
                                        buffers[i].as_ptr() as *const BpfExecEvent,
                                    )
                                };
                                let event = ProcessEvent {
                                    pid:          raw.pid,
                                    ppid:         raw.ppid,
                                    uid:          raw.uid,
                                    timestamp_ns: raw.timestamp_ns,
                                    executable:   cstr_to_string(&raw.filename),
                                    cmdline:      cstr_to_string(&raw.cmdline),
                                };
                                if tx_cpu.send(event).await.is_err() {
                                    // Receiver dropped — shut down this task
                                    return;
                                }
                            }
                        }
                        Err(e) => {
                            error!(cpu = cpu_id, err = %e, "eBPF perf-buffer read error");
                            break;
                        }
                    }
                }
            });
        }

        // NOTE: `bpf` must remain alive for programs and maps to stay loaded.
        // In a real binary, store it in a long-lived struct or Box::leak it.
        // Here we intentionally leak it to keep the tracepoint attached for
        // the lifetime of the process.
        std::mem::forget(bpf);

        Ok(rx)
    }

    /// Human-readable provider identifier used in log lines.
    pub fn name(&self) -> &'static str {
        "K-XRO/eBPF-execve"
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Decode a null-terminated, fixed-size byte slice to a `String`.
///
/// Bytes at and after the first `\0` are discarded.  Invalid UTF-8 sequences
/// are replaced with the Unicode replacement character U+FFFD.
pub fn cstr_to_string(buf: &[u8]) -> String {
    let end = buf.iter().position(|&b| b == 0).unwrap_or(buf.len());
    String::from_utf8_lossy(&buf[..end]).into_owned()
}

// ---------------------------------------------------------------------------
// Unit tests (compile-time only — eBPF hardware not available in CI)
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cstr_nul_terminated() {
        let mut buf = [0u8; 32];
        buf[..5].copy_from_slice(b"hello");
        assert_eq!(cstr_to_string(&buf), "hello");
    }

    #[test]
    fn cstr_full_buffer_no_nul() {
        let buf = b"fullbuffer";
        assert_eq!(cstr_to_string(buf), "fullbuffer");
    }

    #[test]
    fn cstr_empty_buffer() {
        assert_eq!(cstr_to_string(&[0u8; 8]), "");
    }

    #[test]
    fn bpf_exec_event_layout() {
        // Verify that the size matches what the eBPF C struct would produce.
        // pid(4) + ppid(4) + uid(4) [+4 pad] + timestamp_ns(8) + filename(256) + cmdline(512)
        // = 792 bytes with typical C alignment.  Assert at least the minimum.
        assert!(std::mem::size_of::<BpfExecEvent>() >= 256 + 512);
    }
}
