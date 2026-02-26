use serde::Serialize;

/// OCSF PerformanceMetric (class 4004) — host resource utilization snapshot.
#[derive(Debug, Clone, Serialize)]
pub struct HostMetric {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: String,
    pub class_uid: u32,
    pub severity_id: u8,

    // CPU
    pub cpu_count: usize,
    pub cpu_usage_pct: f32,
    pub load_avg_1: f64,
    pub load_avg_5: f64,
    pub load_avg_15: f64,

    // Memory
    pub mem_total_bytes: u64,
    pub mem_used_bytes: u64,
    pub mem_available_bytes: u64,
    pub mem_usage_pct: f32,
    pub swap_total_bytes: u64,
    pub swap_used_bytes: u64,

    // Disk (aggregate across all mount points)
    pub disk_total_bytes: u64,
    pub disk_used_bytes: u64,
    pub disk_read_bytes: u64,
    pub disk_write_bytes: u64,

    // Network (aggregate across all interfaces)
    pub net_rx_bytes: u64,
    pub net_tx_bytes: u64,
    pub net_rx_packets: u64,
    pub net_tx_packets: u64,

    // Process summary
    pub process_count: usize,

    // Uptime
    pub uptime_secs: u64,

    pub blake3_hash: String,
}

impl HostMetric {
    /// NATS subject for host metrics.
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{}.host.metric.v1", tenant_id)
    }
}
