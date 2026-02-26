// K-XRO-PT-005 — Host Metrics via sysinfo
// Collects CPU, memory, disk, network, process metrics using the sysinfo crate.
// Publishes OCSF class 5001 (Performance Activity) events to NATS.

use serde::{Deserialize, Serialize};
use std::cmp::Reverse;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use sysinfo::{CpuRefreshKind, Disks, Networks, ProcessRefreshKind, RefreshKind, System};

// ─── OCSF-aligned data structures ────────────────────────────────────────────

/// Top-level OCSF class 5001 (Performance Activity) snapshot.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HostMetrics {
    pub class_uid: u32,       // always 5001
    pub time: u64,            // Unix epoch milliseconds
    pub hostname: String,
    pub os_name: String,
    pub kernel_version: String,
    pub cpu: CpuMetrics,
    pub memory: MemoryMetrics,
    pub swap: SwapMetrics,
    pub disks: Vec<DiskMetrics>,
    pub networks: Vec<NetworkMetrics>,
    pub process_count: usize,
    pub load_avg: LoadAvg,
    pub uptime_secs: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CpuMetrics {
    pub usage_pct: f32,
    pub core_count: usize,
    pub per_core: Vec<f32>,
    pub brand: String,
    pub frequency_mhz: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryMetrics {
    pub total_bytes: u64,
    pub used_bytes: u64,
    pub available_bytes: u64,
    pub usage_pct: f32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SwapMetrics {
    pub total_bytes: u64,
    pub used_bytes: u64,
    pub usage_pct: f32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiskMetrics {
    pub name: String,
    pub mount_point: String,
    pub fs_type: String,
    pub total_bytes: u64,
    pub available_bytes: u64,
    pub usage_pct: f32,
    pub is_removable: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkMetrics {
    pub interface: String,
    pub bytes_received: u64,
    pub bytes_sent: u64,
    pub packets_received: u64,
    pub packets_sent: u64,
    pub errors_in: u64,
    pub errors_out: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoadAvg {
    pub one: f64,
    pub five: f64,
    pub fifteen: f64,
}

/// Point-in-time snapshot of a single OS process.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProcessSnapshot {
    pub pid: u32,
    pub name: String,
    pub cpu_pct: f32,
    pub mem_bytes: u64,
    pub virtual_mem_bytes: u64,
    pub status: String,
    pub exe: String,
    pub start_time: u64,
}

// ─── Collector ────────────────────────────────────────────────────────────────

/// Owns sysinfo handles and emits `HostMetrics` on demand.
pub struct HostMetricsCollector {
    sys: System,
    disks: Disks,
    networks: Networks,
    hostname: String,
}

impl HostMetricsCollector {
    /// Initialise the collector and perform a warm-up refresh pair so that the
    /// very first `collect()` call returns meaningful CPU usage percentages.
    pub fn new() -> Self {
        let mut sys = System::new_with_specifics(
            RefreshKind::new()
                .with_cpu(CpuRefreshKind::everything())
                .with_memory()
                .with_processes(ProcessRefreshKind::new()),
        );
        // sysinfo requires two samples separated by time to derive CPU usage.
        sys.refresh_all();
        std::thread::sleep(Duration::from_millis(500));
        sys.refresh_all();

        let hostname = System::host_name().unwrap_or_else(|| "unknown".to_string());

        Self {
            sys,
            disks: Disks::new_with_refreshed_list(),
            networks: Networks::new_with_refreshed_list(),
            hostname,
        }
    }

    /// Refresh every subsystem and return a complete `HostMetrics` snapshot.
    pub fn collect(&mut self) -> HostMetrics {
        self.sys.refresh_all();
        self.disks.refresh(true);
        self.networks.refresh(true);

        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_millis() as u64;

        // ── CPU ──────────────────────────────────────────────────────────────
        let cpus = self.sys.cpus();
        let cpu_usage = self.sys.global_cpu_usage();
        let per_core: Vec<f32> = cpus.iter().map(|c| c.cpu_usage()).collect();
        let (brand, freq_mhz) = cpus
            .first()
            .map(|c| (c.brand().to_string(), c.frequency()))
            .unwrap_or_default();

        let cpu = CpuMetrics {
            usage_pct: cpu_usage,
            core_count: cpus.len(),
            per_core,
            brand,
            frequency_mhz: freq_mhz,
        };

        // ── Memory ───────────────────────────────────────────────────────────
        let total_mem = self.sys.total_memory();
        let used_mem = self.sys.used_memory();
        let avail_mem = self.sys.available_memory();
        let mem_pct = if total_mem > 0 {
            (used_mem as f32 / total_mem as f32) * 100.0
        } else {
            0.0
        };
        let memory = MemoryMetrics {
            total_bytes: total_mem,
            used_bytes: used_mem,
            available_bytes: avail_mem,
            usage_pct: mem_pct,
        };

        // ── Swap ─────────────────────────────────────────────────────────────
        let total_swap = self.sys.total_swap();
        let used_swap = self.sys.used_swap();
        let swap_pct = if total_swap > 0 {
            (used_swap as f32 / total_swap as f32) * 100.0
        } else {
            0.0
        };
        let swap = SwapMetrics {
            total_bytes: total_swap,
            used_bytes: used_swap,
            usage_pct: swap_pct,
        };

        // ── Disks ────────────────────────────────────────────────────────────
        let disks: Vec<DiskMetrics> = self
            .disks
            .iter()
            .map(|d| {
                let total = d.total_space();
                let avail = d.available_space();
                let used = total.saturating_sub(avail);
                let pct = if total > 0 {
                    (used as f32 / total as f32) * 100.0
                } else {
                    0.0
                };
                DiskMetrics {
                    name: d.name().to_string_lossy().to_string(),
                    mount_point: d.mount_point().to_string_lossy().to_string(),
                    fs_type: d.file_system().to_string_lossy().to_string(),
                    total_bytes: total,
                    available_bytes: avail,
                    usage_pct: pct,
                    is_removable: d.is_removable(),
                }
            })
            .collect();

        // ── Networks ─────────────────────────────────────────────────────────
        let networks: Vec<NetworkMetrics> = self
            .networks
            .iter()
            .map(|(name, data)| NetworkMetrics {
                interface: name.clone(),
                bytes_received: data.total_received(),
                bytes_sent: data.total_transmitted(),
                packets_received: data.total_packets_received(),
                packets_sent: data.total_packets_transmitted(),
                errors_in: data.total_errors_on_received(),
                errors_out: data.total_errors_on_transmitted(),
            })
            .collect();

        // ── Load average ─────────────────────────────────────────────────────
        let load = System::load_average();
        let load_avg = LoadAvg {
            one: load.one,
            five: load.five,
            fifteen: load.fifteen,
        };

        HostMetrics {
            class_uid: 5001,
            time: now,
            hostname: self.hostname.clone(),
            os_name: System::name().unwrap_or_else(|| "unknown".to_string()),
            kernel_version: System::kernel_version().unwrap_or_else(|| "unknown".to_string()),
            cpu,
            memory,
            swap,
            disks,
            networks,
            process_count: self.sys.processes().len(),
            load_avg,
            uptime_secs: System::uptime(),
        }
    }

    // ── Process helpers ───────────────────────────────────────────────────────

    fn snapshot_process(p: &sysinfo::Process) -> ProcessSnapshot {
        ProcessSnapshot {
            pid: p.pid().as_u32(),
            name: p.name().to_string_lossy().to_string(),
            cpu_pct: p.cpu_usage(),
            mem_bytes: p.memory(),
            virtual_mem_bytes: p.virtual_memory(),
            status: format!("{:?}", p.status()),
            exe: p
                .exe()
                .map(|e| e.to_string_lossy().to_string())
                .unwrap_or_default(),
            start_time: p.start_time(),
        }
    }

    /// Top `n` processes by descending CPU usage.
    pub fn top_processes_by_cpu(&self, n: usize) -> Vec<ProcessSnapshot> {
        let mut procs: Vec<ProcessSnapshot> = self
            .sys
            .processes()
            .values()
            .map(Self::snapshot_process)
            .collect();
        procs.sort_by(|a, b| {
            b.cpu_pct
                .partial_cmp(&a.cpu_pct)
                .unwrap_or(std::cmp::Ordering::Equal)
        });
        procs.truncate(n);
        procs
    }

    /// Top `n` processes by descending RSS memory.
    pub fn top_processes_by_memory(&self, n: usize) -> Vec<ProcessSnapshot> {
        let mut procs: Vec<ProcessSnapshot> = self
            .sys
            .processes()
            .values()
            .map(Self::snapshot_process)
            .collect();
        procs.sort_by_key(|p| Reverse(p.mem_bytes));
        procs.truncate(n);
        procs
    }
}

impl Default for HostMetricsCollector {
    fn default() -> Self {
        Self::new()
    }
}

// ─── Tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn collector_returns_valid_metrics() {
        let mut col = HostMetricsCollector::new();
        let m = col.collect();
        assert_eq!(m.class_uid, 5001);
        assert!(!m.hostname.is_empty());
        assert!(m.cpu.core_count > 0);
        assert!(m.memory.total_bytes > 0);
        assert!(
            (0.0..=100.0).contains(&m.cpu.usage_pct),
            "cpu_usage_pct out of range: {}",
            m.cpu.usage_pct
        );
        assert!(
            (0.0..=100.0).contains(&m.memory.usage_pct),
            "mem_usage_pct out of range: {}",
            m.memory.usage_pct
        );
    }

    #[test]
    fn top_processes_cpu_sorted_descending() {
        let mut col = HostMetricsCollector::new();
        col.collect();
        let procs = col.top_processes_by_cpu(5);
        assert!(procs.len() <= 5);
        for w in procs.windows(2) {
            assert!(
                w[0].cpu_pct >= w[1].cpu_pct,
                "CPU list not descending: {} < {}",
                w[0].cpu_pct,
                w[1].cpu_pct
            );
        }
    }

    #[test]
    fn top_processes_memory_sorted_descending() {
        let mut col = HostMetricsCollector::new();
        col.collect();
        let procs = col.top_processes_by_memory(5);
        assert!(procs.len() <= 5);
        for w in procs.windows(2) {
            assert!(
                w[0].mem_bytes >= w[1].mem_bytes,
                "Memory list not descending"
            );
        }
    }

    #[test]
    fn host_metrics_round_trips_json() {
        let mut col = HostMetricsCollector::new();
        let m = col.collect();
        let json = serde_json::to_string(&m).expect("serialisation failed");
        assert!(json.contains("\"class_uid\":5001"));
        let back: HostMetrics = serde_json::from_str(&json).expect("deserialisation failed");
        assert_eq!(back.class_uid, m.class_uid);
        assert_eq!(back.hostname, m.hostname);
    }

    #[test]
    fn disk_usage_pct_within_range() {
        let mut col = HostMetricsCollector::new();
        let m = col.collect();
        for d in &m.disks {
            assert!(
                (0.0..=100.0).contains(&d.usage_pct),
                "disk {} has out-of-range usage_pct {}",
                d.mount_point,
                d.usage_pct
            );
        }
    }
}
