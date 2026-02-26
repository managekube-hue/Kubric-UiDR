//! Kernel hook abstraction — platform-dispatched event-driven process/file monitoring.
//!
//! On Linux: eBPF via `aya` (tracepoints on execve, openat2)
//! On Windows: ETW via `windows-sys` (Microsoft-Windows-Kernel-Process provider)
//! Fallback: sysinfo polling (already in `agent.rs`)
//!
//! Uses an enum-based dispatch pattern to avoid dyn-incompatible async traits.

#![allow(dead_code)]

#[cfg(all(target_os = "linux", feature = "ebpf"))]
pub mod ebpf;

#[cfg(target_os = "windows")]
pub mod etw;

use serde::Serialize;
use tokio::sync::mpsc;

/// Event emitted by kernel hooks — either a process creation or file access.
#[derive(Debug, Clone, Serialize)]
pub enum HookEvent {
    ProcessExec {
        pid: u32,
        ppid: u32,
        executable: String,
        cmdline: String,
        uid: u32,
        timestamp_ns: u64,
    },
    FileAccess {
        pid: u32,
        path: String,
        operation: FileOp,
        timestamp_ns: u64,
    },
}

#[derive(Debug, Clone, Copy, Serialize)]
pub enum FileOp {
    Create,
    Write,
    Delete,
    Rename,
    SetPermissions,
}

/// Platform-specific hook provider — enum dispatch avoids dyn trait issues.
pub enum HookProviderKind {
    #[cfg(all(target_os = "linux", feature = "ebpf"))]
    Ebpf(ebpf::EbpfProvider),
    #[cfg(target_os = "windows")]
    Etw(etw::EtwProvider),
}

impl HookProviderKind {
    /// Start receiving kernel events.
    pub async fn start(&self) -> anyhow::Result<mpsc::Receiver<HookEvent>> {
        match self {
            #[cfg(all(target_os = "linux", feature = "ebpf"))]
            Self::Ebpf(p) => p.start().await,
            #[cfg(target_os = "windows")]
            Self::Etw(p) => p.start().await,
        }
    }

    /// Human-readable name.
    pub fn name(&self) -> &'static str {
        match self {
            #[cfg(all(target_os = "linux", feature = "ebpf"))]
            Self::Ebpf(_) => "eBPF/aya",
            #[cfg(target_os = "windows")]
            Self::Etw(_) => "ETW/Kernel-Process",
        }
    }
}

/// Create the best available hook provider for the current platform.
///
/// Returns `None` if no kernel hooks can be initialised (the agent should
/// fall back to sysinfo polling).
pub fn create_provider() -> Option<HookProviderKind> {
    #[cfg(all(target_os = "linux", feature = "ebpf"))]
    {
        match ebpf::EbpfProvider::new() {
            Ok(p) => return Some(HookProviderKind::Ebpf(p)),
            Err(e) => {
                tracing::warn!(%e, "eBPF provider init failed — falling back to polling");
            }
        }
    }

    #[cfg(target_os = "windows")]
    {
        match etw::EtwProvider::new() {
            Ok(p) => return Some(HookProviderKind::Etw(p)),
            Err(e) => {
                tracing::warn!(%e, "ETW provider init failed — falling back to polling");
            }
        }
    }

    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hook_event_serializes() {
        let evt = HookEvent::ProcessExec {
            pid: 1234,
            ppid: 1,
            executable: "/usr/bin/curl".into(),
            cmdline: "curl https://evil.com".into(),
            uid: 1000,
            timestamp_ns: 1700000000_000_000_000,
        };
        let json = serde_json::to_string(&evt).unwrap();
        assert!(json.contains("ProcessExec"));
        assert!(json.contains("1234"));
    }

    #[test]
    fn file_op_variants() {
        let ops = [FileOp::Create, FileOp::Write, FileOp::Delete, FileOp::Rename, FileOp::SetPermissions];
        for op in &ops {
            let _ = serde_json::to_string(op).unwrap();
        }
    }

    #[test]
    fn create_provider_returns_option() {
        let _provider = create_provider();
    }
}
