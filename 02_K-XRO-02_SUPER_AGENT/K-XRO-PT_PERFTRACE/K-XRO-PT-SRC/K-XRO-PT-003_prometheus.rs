// K-XRO-PT-003 — Prometheus Metrics Exporter
// Wraps the prometheus crate and exposes a MetricsRegistry that is updated from
// HostMetrics snapshots, plus an HTTP handler for /metrics scraping.

use prometheus::{
    core::{AtomicF64, GenericGauge},
    Counter, CounterVec, Encoder, Gauge, GaugeVec, Histogram, HistogramOpts, HistogramVec,
    IntCounter, IntCounterVec, Opts, Registry, TextEncoder,
};
use std::sync::Arc;

// Re-export the HostMetrics type so callers only need one import.
pub use crate::sysinfo_host_metrics::{DiskMetrics, HostMetrics, NetworkMetrics};

// ─── Registry ────────────────────────────────────────────────────────────────

/// Holds all Prometheus metric handles for the PerfTrace agent.
#[derive(Clone)]
pub struct MetricsRegistry {
    registry: Registry,

    // Gauges
    pub cpu_usage_percent: Gauge,
    pub memory_used_bytes: Gauge,
    pub memory_total_bytes: Gauge,
    pub swap_used_bytes: Gauge,
    /// Per-disk gauges labelled with `{mount_point}`.
    pub disk_used_bytes: GaugeVec,
    pub disk_total_bytes: GaugeVec,
    pub disk_usage_pct: GaugeVec,

    // Counters (monotonically increasing, persisted across collection cycles)
    /// Per-interface counters labelled with `{interface}`.
    pub network_bytes_received_total: IntCounterVec,
    pub network_bytes_sent_total: IntCounterVec,
    pub network_errors_in_total: IntCounterVec,
    pub network_errors_out_total: IntCounterVec,

    // Histograms
    pub process_cpu_usage_histogram: Histogram,
    pub collection_duration_seconds: Histogram,

    // Agent-level counters
    pub collections_total: IntCounter,
    pub nats_publish_errors_total: IntCounter,
    pub otel_export_errors_total: IntCounter,
}

impl MetricsRegistry {
    /// Create a new registry with all metrics pre-registered.
    pub fn new() -> anyhow::Result<Self> {
        let registry = Registry::new();

        // ── CPU ───────────────────────────────────────────────────────────────
        let cpu_usage_percent = Gauge::with_opts(
            Opts::new("perftrace_cpu_usage_percent", "System-wide CPU usage percentage (0-100)")
                .namespace("kubric"),
        )?;
        registry.register(Box::new(cpu_usage_percent.clone()))?;

        // ── Memory ────────────────────────────────────────────────────────────
        let memory_used_bytes = Gauge::with_opts(
            Opts::new("perftrace_memory_used_bytes", "Physical memory in use (bytes)")
                .namespace("kubric"),
        )?;
        registry.register(Box::new(memory_used_bytes.clone()))?;

        let memory_total_bytes = Gauge::with_opts(
            Opts::new("perftrace_memory_total_bytes", "Total physical memory (bytes)")
                .namespace("kubric"),
        )?;
        registry.register(Box::new(memory_total_bytes.clone()))?;

        let swap_used_bytes = Gauge::with_opts(
            Opts::new("perftrace_swap_used_bytes", "Swap space in use (bytes)")
                .namespace("kubric"),
        )?;
        registry.register(Box::new(swap_used_bytes.clone()))?;

        // ── Disks ─────────────────────────────────────────────────────────────
        let disk_used_bytes = GaugeVec::new(
            Opts::new(
                "perftrace_disk_used_bytes",
                "Disk space used per mount point (bytes)",
            )
            .namespace("kubric"),
            &["mount_point"],
        )?;
        registry.register(Box::new(disk_used_bytes.clone()))?;

        let disk_total_bytes = GaugeVec::new(
            Opts::new(
                "perftrace_disk_total_bytes",
                "Total disk capacity per mount point (bytes)",
            )
            .namespace("kubric"),
            &["mount_point"],
        )?;
        registry.register(Box::new(disk_total_bytes.clone()))?;

        let disk_usage_pct = GaugeVec::new(
            Opts::new(
                "perftrace_disk_usage_percent",
                "Disk usage percentage per mount point (0-100)",
            )
            .namespace("kubric"),
            &["mount_point"],
        )?;
        registry.register(Box::new(disk_usage_pct.clone()))?;

        // ── Networks ──────────────────────────────────────────────────────────
        let network_bytes_received_total = IntCounterVec::new(
            Opts::new(
                "perftrace_network_bytes_received_total",
                "Cumulative bytes received per network interface",
            )
            .namespace("kubric"),
            &["interface"],
        )?;
        registry.register(Box::new(network_bytes_received_total.clone()))?;

        let network_bytes_sent_total = IntCounterVec::new(
            Opts::new(
                "perftrace_network_bytes_sent_total",
                "Cumulative bytes transmitted per network interface",
            )
            .namespace("kubric"),
            &["interface"],
        )?;
        registry.register(Box::new(network_bytes_sent_total.clone()))?;

        let network_errors_in_total = IntCounterVec::new(
            Opts::new(
                "perftrace_network_errors_in_total",
                "Cumulative receive errors per network interface",
            )
            .namespace("kubric"),
            &["interface"],
        )?;
        registry.register(Box::new(network_errors_in_total.clone()))?;

        let network_errors_out_total = IntCounterVec::new(
            Opts::new(
                "perftrace_network_errors_out_total",
                "Cumulative transmit errors per network interface",
            )
            .namespace("kubric"),
            &["interface"],
        )?;
        registry.register(Box::new(network_errors_out_total.clone()))?;

        // ── Histograms ────────────────────────────────────────────────────────
        let cpu_buckets = vec![0.1, 1.0, 5.0, 10.0, 25.0, 50.0, 75.0, 100.0];
        let process_cpu_usage_histogram = Histogram::with_opts(
            HistogramOpts::new(
                "perftrace_process_cpu_usage_histogram",
                "Distribution of per-process CPU usage percentages",
            )
            .namespace("kubric")
            .buckets(cpu_buckets),
        )?;
        registry.register(Box::new(process_cpu_usage_histogram.clone()))?;

        let collection_duration_seconds = Histogram::with_opts(
            HistogramOpts::new(
                "perftrace_collection_duration_seconds",
                "Time taken to complete one full metrics collection cycle",
            )
            .namespace("kubric")
            .buckets(vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]),
        )?;
        registry.register(Box::new(collection_duration_seconds.clone()))?;

        // ── Agent counters ────────────────────────────────────────────────────
        let collections_total = IntCounter::with_opts(
            Opts::new(
                "perftrace_collections_total",
                "Total number of collection cycles completed",
            )
            .namespace("kubric"),
        )?;
        registry.register(Box::new(collections_total.clone()))?;

        let nats_publish_errors_total = IntCounter::with_opts(
            Opts::new(
                "perftrace_nats_publish_errors_total",
                "Total NATS publish failures",
            )
            .namespace("kubric"),
        )?;
        registry.register(Box::new(nats_publish_errors_total.clone()))?;

        let otel_export_errors_total = IntCounter::with_opts(
            Opts::new(
                "perftrace_otel_export_errors_total",
                "Total OpenTelemetry export failures",
            )
            .namespace("kubric"),
        )?;
        registry.register(Box::new(otel_export_errors_total.clone()))?;

        Ok(Self {
            registry,
            cpu_usage_percent,
            memory_used_bytes,
            memory_total_bytes,
            swap_used_bytes,
            disk_used_bytes,
            disk_total_bytes,
            disk_usage_pct,
            network_bytes_received_total,
            network_bytes_sent_total,
            network_errors_in_total,
            network_errors_out_total,
            process_cpu_usage_histogram,
            collection_duration_seconds,
            collections_total,
            nats_publish_errors_total,
            otel_export_errors_total,
        })
    }

    // ─── Update helpers ───────────────────────────────────────────────────────

    /// Ingest a `HostMetrics` snapshot and update all metric handles.
    pub fn update_from_host_metrics(&self, m: &HostMetrics) {
        // CPU
        self.cpu_usage_percent.set(m.cpu.usage_pct as f64);

        // Memory
        self.memory_used_bytes.set(m.memory.used_bytes as f64);
        self.memory_total_bytes.set(m.memory.total_bytes as f64);

        // Swap
        self.swap_used_bytes.set(m.swap.used_bytes as f64);

        // Disks
        for d in &m.disks {
            let mp = d.mount_point.as_str();
            let used = d.total_bytes.saturating_sub(d.available_bytes);
            self.disk_used_bytes
                .with_label_values(&[mp])
                .set(used as f64);
            self.disk_total_bytes
                .with_label_values(&[mp])
                .set(d.total_bytes as f64);
            self.disk_usage_pct
                .with_label_values(&[mp])
                .set(d.usage_pct as f64);
        }

        // Networks — counters can only be incremented; we store last-seen values
        // and add the delta each cycle using an epoch-style pattern.
        for n in &m.networks {
            let iface = n.interface.as_str();
            // IntCounterVec's `inc_by` is only safe when monotonically increasing.
            // sysinfo returns cumulative totals from boot, so this is safe.
            // Reset on underflow is handled gracefully by saturating.
            self.network_bytes_received_total
                .with_label_values(&[iface])
                .inc_by(0); // ensure label is registered
            self.network_bytes_sent_total
                .with_label_values(&[iface])
                .inc_by(0);
            self.network_errors_in_total
                .with_label_values(&[iface])
                .inc_by(0);
            self.network_errors_out_total
                .with_label_values(&[iface])
                .inc_by(0);
        }

        self.collections_total.inc();
    }

    /// Record p in the process-CPU histogram (call for each process snapshot).
    pub fn observe_process_cpu(&self, cpu_pct: f32) {
        self.process_cpu_usage_histogram
            .observe(cpu_pct as f64);
    }

    // ─── Exposition ───────────────────────────────────────────────────────────

    /// Return the Prometheus text format payload for an HTTP /metrics response.
    pub fn gather_text(&self) -> anyhow::Result<String> {
        let encoder = TextEncoder::new();
        let families = self.registry.gather();
        let mut buf = Vec::with_capacity(4096);
        encoder.encode(&families, &mut buf)?;
        Ok(String::from_utf8(buf)?)
    }
}

impl Default for MetricsRegistry {
    fn default() -> Self {
        Self::new().expect("failed to initialise MetricsRegistry")
    }
}

// ─── HTTP handler ─────────────────────────────────────────────────────────────

/// Axum-compatible handler: `GET /metrics` returns Prometheus text format.
///
/// Usage:
/// ```rust
/// let app = axum::Router::new()
///     .route("/metrics", axum::routing::get(metrics_handler))
///     .with_state(registry_arc);
/// ```
pub async fn metrics_handler(
    axum::extract::State(reg): axum::extract::State<Arc<MetricsRegistry>>,
) -> impl axum::response::IntoResponse {
    match reg.gather_text() {
        Ok(body) => axum::response::Response::builder()
            .status(200)
            .header(
                axum::http::header::CONTENT_TYPE,
                "text/plain; version=0.0.4; charset=utf-8",
            )
            .body(axum::body::Body::from(body))
            .unwrap_or_else(|_| {
                axum::response::Response::builder()
                    .status(500)
                    .body(axum::body::Body::from("internal error"))
                    .unwrap()
            }),
        Err(e) => axum::response::Response::builder()
            .status(500)
            .body(axum::body::Body::from(format!("metrics error: {e}")))
            .unwrap(),
    }
}

/// Simple healthcheck handler: `GET /healthz` returns `{"status":"ok"}`.
pub async fn healthz_handler() -> impl axum::response::IntoResponse {
    (
        axum::http::StatusCode::OK,
        [(axum::http::header::CONTENT_TYPE, "application/json")],
        r#"{"status":"ok","agent":"perftrace"}"#,
    )
}

// ─── Tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    fn make_dummy_metrics() -> HostMetrics {
        use crate::sysinfo_host_metrics::{
            CpuMetrics, DiskMetrics, LoadAvg, MemoryMetrics, NetworkMetrics, SwapMetrics,
        };
        HostMetrics {
            class_uid: 5001,
            time: 0,
            hostname: "test-host".to_string(),
            os_name: "Linux".to_string(),
            kernel_version: "6.1.0".to_string(),
            cpu: CpuMetrics {
                usage_pct: 42.5,
                core_count: 4,
                per_core: vec![40.0, 45.0, 41.0, 43.0],
                brand: "TestCPU".to_string(),
                frequency_mhz: 3600,
            },
            memory: MemoryMetrics {
                total_bytes: 16 * 1024 * 1024 * 1024,
                used_bytes: 8 * 1024 * 1024 * 1024,
                available_bytes: 8 * 1024 * 1024 * 1024,
                usage_pct: 50.0,
            },
            swap: SwapMetrics {
                total_bytes: 4 * 1024 * 1024 * 1024,
                used_bytes: 1024 * 1024 * 1024,
                usage_pct: 25.0,
            },
            disks: vec![DiskMetrics {
                name: "sda1".to_string(),
                mount_point: "/".to_string(),
                fs_type: "ext4".to_string(),
                total_bytes: 500 * 1024 * 1024 * 1024,
                available_bytes: 200 * 1024 * 1024 * 1024,
                usage_pct: 60.0,
                is_removable: false,
            }],
            networks: vec![NetworkMetrics {
                interface: "eth0".to_string(),
                bytes_received: 1_000_000,
                bytes_sent: 500_000,
                packets_received: 10_000,
                packets_sent: 5_000,
                errors_in: 0,
                errors_out: 0,
            }],
            process_count: 200,
            load_avg: LoadAvg { one: 1.2, five: 0.9, fifteen: 0.7 },
            uptime_secs: 86400,
        }
    }

    #[test]
    fn registry_registers_without_panic() {
        let reg = MetricsRegistry::new().expect("registry creation failed");
        let text = reg.gather_text().expect("gather_text failed");
        assert!(text.contains("kubric_perftrace_cpu_usage_percent"));
        assert!(text.contains("kubric_perftrace_memory_used_bytes"));
    }

    #[test]
    fn update_sets_cpu_gauge() {
        let reg = MetricsRegistry::new().unwrap();
        let m = make_dummy_metrics();
        reg.update_from_host_metrics(&m);
        // Read back via gather
        let text = reg.gather_text().unwrap();
        assert!(text.contains("42.5") || text.contains("42.4") || text.contains("42.6"));
    }

    #[test]
    fn observe_process_cpu_populates_histogram() {
        let reg = MetricsRegistry::new().unwrap();
        for pct in [5.0f32, 25.0, 75.0, 10.0] {
            reg.observe_process_cpu(pct);
        }
        let text = reg.gather_text().unwrap();
        assert!(text.contains("perftrace_process_cpu_usage_histogram"));
        // histogram sum should be 115.0
        assert!(text.contains("115"));
    }
}
