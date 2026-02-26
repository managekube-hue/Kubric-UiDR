//! K-XRO-CS-EBPF-002 — eBPF openat2 hook for File Integrity Monitoring
//!
//! Standalone eBPF hook provider that attaches to the `sys_enter_openat2`
//! kernel tracepoint using `aya`.  Every file-open operation is intercepted in
//! the kernel and forwarded to userspace via a perf ring-buffer map named
//! FILE_EVENTS.
//!
//! Production counterpart: `agents/coresec/src/hooks/ebpf.rs` (openat2 section)
//!
//! Kernel requirements: Linux 5.6+ for openat2(2) syscall; 5.8+ for async perf.
//! Pre-compiled eBPF ELF object expected at:
//!   vendor/ebpf/openat2_hook.o   (set via env KUBRIC_EBPF_OPENAT)
//!
//! Module: K-XRO-CS-EBPF-002

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
// Public types
// ---------------------------------------------------------------------------

/// File operation discriminant — mirrors the BPF-side enum.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u32)]
pub enum FileOp {
    Open   = 0,
    Read   = 1,
    Write  = 2,
    Unlink = 3,
}

impl FileOp {
    fn from_u32(v: u32) -> Self {
        match v {
            1 => FileOp::Read,
            2 => FileOp::Write,
            3 => FileOp::Unlink,
            _ => FileOp::Open,
        }
    }
}

impl std::fmt::Display for FileOp {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            FileOp::Open   => write!(f, "open"),
            FileOp::Read   => write!(f, "read"),
            FileOp::Write  => write!(f, "write"),
            FileOp::Unlink => write!(f, "unlink"),
        }
    }
}

/// Decoded file-access event forwarded to userspace consumers.
#[derive(Debug, Clone)]
pub struct FileEvent {
    /// PID that called openat2.
    pub pid: u32,
    /// Absolute path being accessed (as recorded by the kernel).
    pub path: String,
    /// Operation type.
    pub op: FileOp,
    /// Kernel monotonic timestamp in nanoseconds.
    pub timestamp_ns: u64,
}

// ---------------------------------------------------------------------------
// Kernel-layout struct
// ---------------------------------------------------------------------------

/// Raw event layout in FILE_EVENTS perf map.  Must match openat2_hook.c.
#[repr(C)]
#[derive(Debug, Clone, Copy)]
struct BpfFileEvent {
    pid:          u32,
    timestamp_ns: u64,
    op:           u32,
    /// Null-terminated filename, padded to 256 bytes.
    filename:     [u8; 256],
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

/// eBPF-backed openat2 hook provider for FIM.
pub struct Openat2Provider {
    /// Filesystem path to the eBPF ELF object.
    pub openat_path: String,
}

impl Openat2Provider {
    /// Build from the `KUBRIC_EBPF_OPENAT` environment variable.
    ///
    /// # Errors
    /// Returns `Err` if the ELF object is absent from disk.
    pub fn new() -> anyhow::Result<Self> {
        let openat_path = std::env::var("KUBRIC_EBPF_OPENAT")
            .unwrap_or_else(|_| "vendor/ebpf/openat2_hook.o".into());

        if !Path::new(&openat_path).exists() {
            anyhow::bail!(
                "eBPF ELF object not found at '{}'. \
                 Build with 'cargo xtask build-ebpf' or set KUBRIC_EBPF_OPENAT.",
                openat_path
            );
        }

        Ok(Self { openat_path })
    }

    /// Load, attach to `sys_enter_openat2`, and return an event receiver.
    ///
    /// The returned channel emits one `FileEvent` per openat2 syscall
    /// observed by the kernel, across all CPUs.
    pub async fn start(&self) -> anyhow::Result<mpsc::Receiver<FileEvent>> {
        let (tx, rx) = mpsc::channel::<FileEvent>(8192);

        // ------------------------------------------------------------------
        // Load object and attach tracepoint
        // ------------------------------------------------------------------
        let mut bpf: Bpf = BpfLoader::new().load_file(&self.openat_path)?;

        {
            let prog: &mut TracePoint = bpf
                .program_mut("openat2_hook")
                .ok_or_else(|| {
                    anyhow::anyhow!(
                        "section 'openat2_hook' not found in '{}'",
                        self.openat_path
                    )
                })?
                .try_into()?;
            prog.load()?;
            prog.attach("syscalls", "sys_enter_openat2")?;
        }
        info!(path = %self.openat_path, "eBPF: attached tracepoint sys_enter_openat2");

        // ------------------------------------------------------------------
        // Open FILE_EVENTS perf ring-buffer map
        // ------------------------------------------------------------------
        let mut file_events: AsyncPerfEventArray<_> = AsyncPerfEventArray::try_from(
            bpf.take_map("FILE_EVENTS")
                .ok_or_else(|| anyhow::anyhow!("FILE_EVENTS map not found in ELF"))?,
        )?;

        let cpus = online_cpus()
            .map_err(|e| anyhow::anyhow!("online_cpus: {:?}", e))?;

        info!(cpu_count = cpus.len(), "eBPF: spawning FILE_EVENTS per-CPU readers");

        // ------------------------------------------------------------------
        // Per-CPU reader tasks
        // ------------------------------------------------------------------
        for cpu_id in cpus {
            let mut ring = file_events.open(cpu_id, Some(256))?;
            let tx_cpu = tx.clone();

            tokio::spawn(async move {
                let event_size = std::mem::size_of::<BpfFileEvent>();
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
                                        got  = buffers[i].len(),
                                        need = event_size,
                                        "FILE_EVENTS buffer too short — discarding"
                                    );
                                    continue;
                                }
                                // SAFETY: length verified above; layout is #[repr(C)].
                                let raw: BpfFileEvent = unsafe {
                                    std::ptr::read_unaligned(
                                        buffers[i].as_ptr() as *const BpfFileEvent,
                                    )
                                };
                                let event = FileEvent {
                                    pid:          raw.pid,
                                    timestamp_ns: raw.timestamp_ns,
                                    op:           FileOp::from_u32(raw.op),
                                    path:         cstr_to_string(&raw.filename),
                                };
                                if tx_cpu.send(event).await.is_err() {
                                    return; // receiver dropped
                                }
                            }
                        }
                        Err(e) => {
                            error!(cpu = cpu_id, err = %e, "FILE_EVENTS perf-buffer read error");
                            break;
                        }
                    }
                }
            });
        }

        std::mem::forget(bpf);
        Ok(rx)
    }

    pub fn name(&self) -> &'static str {
        "K-XRO/eBPF-openat2"
    }
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

fn cstr_to_string(buf: &[u8]) -> String {
    let end = buf.iter().position(|&b| b == 0).unwrap_or(buf.len());
    String::from_utf8_lossy(&buf[..end]).into_owned()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn file_op_round_trip() {
        assert_eq!(FileOp::from_u32(0), FileOp::Open);
        assert_eq!(FileOp::from_u32(1), FileOp::Read);
        assert_eq!(FileOp::from_u32(2), FileOp::Write);
        assert_eq!(FileOp::from_u32(3), FileOp::Unlink);
        // Unknown values fall back to Open
        assert_eq!(FileOp::from_u32(99), FileOp::Open);
    }

    #[test]
    fn file_op_display() {
        assert_eq!(format!("{}", FileOp::Open),   "open");
        assert_eq!(format!("{}", FileOp::Write),  "write");
        assert_eq!(format!("{}", FileOp::Unlink), "unlink");
    }

    #[test]
    fn bpf_file_event_minimum_size() {
        // filename field alone is 256 bytes; total must be at least that.
        assert!(std::mem::size_of::<BpfFileEvent>() >= 256);
    }
}
