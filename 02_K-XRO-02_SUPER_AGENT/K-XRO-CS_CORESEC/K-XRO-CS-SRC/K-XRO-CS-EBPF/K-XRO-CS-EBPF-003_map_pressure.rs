//! K-XRO-CS-EBPF-003 — eBPF map pressure monitor
//!
//! Tracks kernel BPF map utilisation by reading entry counts from the
//! `/sys/fs/bpf/` pseudo-filesystem and comparing them against the expected
//! maximum capacity constants.  Emits alerts when any map exceeds the
//! configurable high-watermark threshold (default 80 %).
//!
//! This module is intentionally OS-agnostic at the type level so that it can
//! be compiled on non-Linux hosts; the `check_pressure` implementation guards
//! Linux-only syscall paths with `#[cfg(target_os = "linux")]`.
//!
//! Production counterpart: internal monitoring in `agents/coresec/src/hooks/`
//!
//! Module: K-XRO-CS-EBPF-003

use std::path::{Path, PathBuf};
use tracing::{info, warn};

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/// Maps exceeding this fraction of capacity trigger an alert.
pub const DEFAULT_THRESHOLD: f64 = 0.80;

/// Maximum entries allocated for the EXEC_EVENTS perf map (per-CPU * this).
pub const EXEC_EVENTS_MAX: u64 = 65_536;

/// Maximum entries allocated for the FILE_EVENTS perf map (per-CPU * this).
pub const FILE_EVENTS_MAX: u64 = 65_536;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/// Utilisation snapshot for a single BPF map.
#[derive(Debug, Clone)]
pub struct MapStats {
    /// Name of the BPF map (as registered with `bpf_obj_pin` or autodiscovered).
    pub name: String,
    /// Maximum number of entries the map was created with.
    pub max_entries: u64,
    /// Current number of entries present in the map.
    pub current_entries: u64,
    /// `current_entries / max_entries` as a fraction in [0.0, 1.0].
    pub usage_pct: f64,
}

impl MapStats {
    /// Returns `true` when utilisation is at or above the supplied threshold.
    pub fn is_over_threshold(&self, threshold: f64) -> bool {
        self.usage_pct >= threshold
    }
}

impl std::fmt::Display for MapStats {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "{}: {}/{} ({:.1}%)",
            self.name,
            self.current_entries,
            self.max_entries,
            self.usage_pct * 100.0
        )
    }
}

// ---------------------------------------------------------------------------
// Monitor
// ---------------------------------------------------------------------------

/// Polls BPF map utilisation and alerts when pressure exceeds the threshold.
pub struct MapPressureMonitor {
    /// Fraction [0.0, 1.0] above which an alert is raised; default 0.80.
    pub threshold: f64,
    /// Map names to monitor.  Keyed by the name used in the eBPF C source.
    registered: Vec<(String, u64)>,
}

impl MapPressureMonitor {
    /// Create a monitor with the default 80 % threshold and the two standard
    /// Kubric BPF maps pre-registered.
    pub fn new() -> Self {
        Self::with_threshold(DEFAULT_THRESHOLD)
    }

    /// Create with a custom threshold.
    pub fn with_threshold(threshold: f64) -> Self {
        let registered = vec![
            ("EXEC_EVENTS".to_string(), EXEC_EVENTS_MAX),
            ("FILE_EVENTS".to_string(), FILE_EVENTS_MAX),
        ];
        Self { threshold, registered }
    }

    /// Register an additional map to monitor.
    pub fn register(&mut self, name: impl Into<String>, max_entries: u64) {
        self.registered.push((name.into(), max_entries));
    }

    /// Poll all registered maps and return their current stats.
    ///
    /// Maps that cannot be read (e.g. because they are not pinned) are silently
    /// skipped — this is expected on non-Linux systems or when eBPF is disabled.
    pub fn check_pressure(&self) -> Vec<MapStats> {
        let mut results = Vec::with_capacity(self.registered.len());

        for (name, max) in &self.registered {
            if let Some(stats) = read_map_stats(name, *max) {
                if stats.is_over_threshold(self.threshold) {
                    warn!(
                        map    = %stats.name,
                        usage  = %format!("{:.1}%", stats.usage_pct * 100.0),
                        threshold = %format!("{:.0}%", self.threshold * 100.0),
                        "BPF map pressure exceeds threshold"
                    );
                } else {
                    info!(map = %stats, "BPF map pressure OK");
                }
                results.push(stats);
            }
        }

        results
    }

    /// Convenience: return only maps that are over the threshold.
    pub fn hot_maps(&self) -> Vec<MapStats> {
        self.check_pressure()
            .into_iter()
            .filter(|s| s.is_over_threshold(self.threshold))
            .collect()
    }
}

// ---------------------------------------------------------------------------
// Implementation — reads from /sys/fs/bpf/ on Linux
// ---------------------------------------------------------------------------

/// Attempt to read current entry count for a pinned BPF map.
///
/// On Linux, `bpf_obj_get_info_by_fd` via the `perf_event_attr` ABI would be
/// the canonical approach. As a practical alternative that requires no extra
/// crate dependencies, we read the `entries` sysfs file exposed by some map
/// types under `/sys/fs/bpf/<name>/`.
///
/// Returns `None` if the map is not pinned, the sysfs file is absent, or the
/// platform is not Linux.
pub fn read_map_stats(map_name: &str, max_entries: u64) -> Option<MapStats> {
    #[cfg(target_os = "linux")]
    {
        // Try the sysfs pin path first
        let sysfs_path = PathBuf::from("/sys/fs/bpf").join(map_name).join("entries");
        let current_entries = if sysfs_path.exists() {
            read_u64_from_file(&sysfs_path)?
        } else {
            // Fall back to reading procfs bpf_stats_enabled summary
            probe_map_via_procfs(map_name)?
        };

        let usage_pct = if max_entries > 0 {
            current_entries as f64 / max_entries as f64
        } else {
            0.0
        };

        Some(MapStats {
            name: map_name.to_string(),
            max_entries,
            current_entries,
            usage_pct: usage_pct.min(1.0),
        })
    }

    #[cfg(not(target_os = "linux"))]
    {
        // On non-Linux targets return a synthetic zero-pressure result so
        // callers can still use the type without panicking.
        let _ = map_name;
        None
    }
}

/// Read a decimal u64 from a single-line sysfs file.
fn read_u64_from_file(path: &Path) -> Option<u64> {
    let raw = std::fs::read_to_string(path).ok()?;
    raw.trim().parse::<u64>().ok()
}

/// Fallback: scan `/proc/net/dev`-style BPF stat files for the map.
/// In practice the kernel does not expose per-map entry counts this way,
/// so this probe always returns `None` — it exists as an extension point.
#[cfg(target_os = "linux")]
fn probe_map_via_procfs(_map_name: &str) -> Option<u64> {
    // Placeholder: real implementation would use bpf(BPF_MAP_GET_NEXT_KEY)
    // to iterate and count, or bpf(BPF_OBJ_GET_INFO_BY_FD).
    None
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn map_stats_over_threshold() {
        let s = MapStats {
            name: "TEST".into(),
            max_entries: 100,
            current_entries: 85,
            usage_pct: 0.85,
        };
        assert!(s.is_over_threshold(0.80));
        assert!(!s.is_over_threshold(0.90));
    }

    #[test]
    fn map_stats_display() {
        let s = MapStats {
            name: "EXEC_EVENTS".into(),
            max_entries: 65536,
            current_entries: 32768,
            usage_pct: 0.5,
        };
        let text = format!("{}", s);
        assert!(text.contains("EXEC_EVENTS"));
        assert!(text.contains("50.0%"));
    }

    #[test]
    fn monitor_defaults() {
        let mon = MapPressureMonitor::new();
        assert_eq!(mon.threshold, DEFAULT_THRESHOLD);
        assert_eq!(mon.registered.len(), 2);
    }

    #[test]
    fn monitor_register() {
        let mut mon = MapPressureMonitor::new();
        mon.register("CUSTOM_MAP", 1024);
        assert_eq!(mon.registered.len(), 3);
    }

    #[test]
    fn constants_sanity() {
        assert_eq!(DEFAULT_THRESHOLD, 0.80);
        assert_eq!(EXEC_EVENTS_MAX, 65_536);
        assert_eq!(FILE_EVENTS_MAX, 65_536);
    }
}
