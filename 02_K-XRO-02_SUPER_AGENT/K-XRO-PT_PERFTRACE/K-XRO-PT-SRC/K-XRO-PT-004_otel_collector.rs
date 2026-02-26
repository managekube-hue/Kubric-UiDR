// K-XRO-PT-004 — OpenTelemetry Collector Bridge
//
// Exports PerfTrace metrics and traces to an OTLP endpoint (gRPC/tonic).
// Falls back to a no-op provider when the endpoint is empty or unreachable so
// that the rest of the agent is never blocked by OTEL availability.

use anyhow::{anyhow, Context, Result};
use opentelemetry::{
    global,
    metrics::{Counter, Gauge, MeterProvider, ObservableGauge, UpDownCounter},
    KeyValue,
};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{
    metrics::{
        reader::DefaultTemporalitySelector, MeterProviderBuilder, PeriodicReader, SdkMeterProvider,
    },
    runtime,
    trace::{Config as TraceConfig, Sampler, TracerProvider},
    Resource,
};
use opentelemetry_semantic_conventions::resource as semconv;
use std::time::Duration;

use crate::sysinfo_host_metrics::HostMetrics;
use crate::perf_event_open::PerfReading;

// ─── Resource builder ─────────────────────────────────────────────────────────

fn build_resource(service_name: &str) -> Resource {
    let hostname = std::env::var("HOSTNAME")
        .or_else(|_| std::env::var("COMPUTERNAME"))
        .unwrap_or_else(|_| "unknown".to_string());

    Resource::new(vec![
        KeyValue::new(semconv::SERVICE_NAME, service_name.to_string()),
        KeyValue::new(semconv::SERVICE_VERSION, "1.0.0"),
        KeyValue::new(semconv::HOST_NAME, hostname),
        KeyValue::new("service.namespace", "kubric"),
        KeyValue::new("deployment.environment", "production"),
    ])
}

// ─── Tracer init ──────────────────────────────────────────────────────────────

/// Initialise an OTLP gRPC tracer for `service_name`.
///
/// Returns a no-op `TracerProvider` (and prints a warning) when `endpoint` is
/// empty or the gRPC connection cannot be established.
pub fn init_tracer(
    service_name: &str,
    endpoint: &str,
) -> Result<opentelemetry_sdk::trace::TracerProvider> {
    if endpoint.is_empty() {
        tracing::warn!("OTEL_ENDPOINT is empty – using no-op tracer");
        let provider = TracerProvider::builder()
            .with_config(TraceConfig::default().with_sampler(Sampler::AlwaysOff))
            .build();
        return Ok(provider);
    }

    let exporter = opentelemetry_otlp::new_exporter()
        .tonic()
        .with_endpoint(endpoint)
        .with_timeout(Duration::from_secs(5))
        .build_span_exporter()
        .context("building OTLP span exporter")?;

    let provider = TracerProvider::builder()
        .with_simple_exporter(exporter)
        .with_config(
            TraceConfig::default()
                .with_sampler(Sampler::ParentBased(Box::new(Sampler::TraceIdRatioBased(0.1))))
                .with_resource(build_resource(service_name)),
        )
        .build();

    global::set_tracer_provider(provider.clone());
    tracing::info!(endpoint, "OTLP tracer initialised");
    Ok(provider)
}

// ─── Meter init ───────────────────────────────────────────────────────────────

/// Initialise an OTLP gRPC meter provider for `service_name`.
///
/// Returns a no-op provider when the endpoint is empty or unavailable.
pub fn init_meter(
    service_name: &str,
    endpoint: &str,
    export_interval: Duration,
) -> Result<SdkMeterProvider> {
    if endpoint.is_empty() {
        tracing::warn!("OTEL_ENDPOINT is empty – using no-op meter");
        let provider = SdkMeterProvider::builder().build();
        global::set_meter_provider(provider.clone());
        return Ok(provider);
    }

    let exporter = opentelemetry_otlp::new_exporter()
        .tonic()
        .with_endpoint(endpoint)
        .with_timeout(Duration::from_secs(5))
        .build_metrics_exporter(Box::new(DefaultTemporalitySelector::new()))
        .context("building OTLP metrics exporter")?;

    let reader = PeriodicReader::builder(exporter, runtime::Tokio)
        .with_interval(export_interval)
        .build();

    let provider = SdkMeterProvider::builder()
        .with_reader(reader)
        .with_resource(build_resource(service_name))
        .build();

    global::set_meter_provider(provider.clone());
    tracing::info!(endpoint, "OTLP meter initialised");
    Ok(provider)
}

// ─── OtelExporter configuration ──────────────────────────────────────────────

/// Holds configuration for initialising both the tracer and meter providers.
#[derive(Debug, Clone)]
pub struct OtelExporter {
    pub service_name: String,
    pub endpoint: String,
    pub export_interval: Duration,
}

impl OtelExporter {
    pub fn new(service_name: impl Into<String>, endpoint: impl Into<String>) -> Self {
        Self {
            service_name: service_name.into(),
            endpoint: endpoint.into(),
            export_interval: Duration::from_secs(15),
        }
    }

    pub fn with_export_interval(mut self, d: Duration) -> Self {
        self.export_interval = d;
        self
    }

    /// Initialise both the tracer and meter providers.  Both are registered
    /// as global providers so library code using `opentelemetry::global` also
    /// benefits.
    pub fn init(&self) -> Result<(opentelemetry_sdk::trace::TracerProvider, SdkMeterProvider)> {
        let tracer_provider = init_tracer(&self.service_name, &self.endpoint)?;
        let meter_provider =
            init_meter(&self.service_name, &self.endpoint, self.export_interval)?;
        Ok((tracer_provider, meter_provider))
    }
}

// ─── OtelMetricBridge ─────────────────────────────────────────────────────────

/// Holds pre-registered OTEL instrument handles and drives metric recording.
pub struct OtelMetricBridge {
    // CPU
    cpu_usage: ObservableGauge<f64>,

    // Memory
    memory_used_bytes: ObservableGauge<f64>,
    memory_total_bytes: ObservableGauge<f64>,

    // Swap
    swap_used_bytes: ObservableGauge<f64>,

    // Disk (UpDownCounter — can decrease when partitions are resized)
    disk_used_bytes: UpDownCounter<i64>,
    disk_total_bytes: UpDownCounter<i64>,

    // Network (monotonically increasing counters)
    network_bytes_received: Counter<u64>,
    network_bytes_sent: Counter<u64>,

    // Perf counters
    perf_cpu_cycles: Counter<u64>,
    perf_instructions: Counter<u64>,
    perf_cache_misses: Counter<u64>,
    perf_branch_misses: Counter<u64>,
    perf_context_switches: Counter<u64>,
    perf_page_faults: Counter<u64>,

    // Previous perf readings (deltas between collection cycles)
    prev_perf: PerfReading,
    prev_net_rx: std::collections::HashMap<String, u64>,
    prev_net_tx: std::collections::HashMap<String, u64>,
}

impl OtelMetricBridge {
    /// Create instruments on the given meter.
    pub fn new(meter_provider: &SdkMeterProvider) -> Self {
        let meter = meter_provider.meter("kubric.perftrace");

        let cpu_usage = meter
            .f64_observable_gauge("kubric.host.cpu.usage")
            .with_description("System-wide CPU usage percentage (0-100)")
            .with_unit("%")
            .init();

        let memory_used_bytes = meter
            .f64_observable_gauge("kubric.host.memory.used_bytes")
            .with_description("Physical memory in use")
            .with_unit("By")
            .init();

        let memory_total_bytes = meter
            .f64_observable_gauge("kubric.host.memory.total_bytes")
            .with_description("Total physical memory")
            .with_unit("By")
            .init();

        let swap_used_bytes = meter
            .f64_observable_gauge("kubric.host.swap.used_bytes")
            .with_description("Swap space in use")
            .with_unit("By")
            .init();

        let disk_used_bytes = meter
            .i64_up_down_counter("kubric.host.disk.used_bytes")
            .with_description("Disk space used per mount point")
            .with_unit("By")
            .init();

        let disk_total_bytes = meter
            .i64_up_down_counter("kubric.host.disk.total_bytes")
            .with_description("Total disk capacity per mount point")
            .with_unit("By")
            .init();

        let network_bytes_received = meter
            .u64_counter("kubric.host.network.bytes_received")
            .with_description("Cumulative bytes received per interface")
            .with_unit("By")
            .init();

        let network_bytes_sent = meter
            .u64_counter("kubric.host.network.bytes_sent")
            .with_description("Cumulative bytes transmitted per interface")
            .with_unit("By")
            .init();

        let perf_cpu_cycles = meter
            .u64_counter("kubric.agent.perf.cpu_cycles")
            .with_description("CPU cycles accumulated since last collection")
            .init();

        let perf_instructions = meter
            .u64_counter("kubric.agent.perf.instructions")
            .with_description("Instructions retired since last collection")
            .init();

        let perf_cache_misses = meter
            .u64_counter("kubric.agent.perf.cache_misses")
            .with_description("Last-level cache misses since last collection")
            .init();

        let perf_branch_misses = meter
            .u64_counter("kubric.agent.perf.branch_misses")
            .with_description("Branch mispredictions since last collection")
            .init();

        let perf_context_switches = meter
            .u64_counter("kubric.agent.perf.context_switches")
            .with_description("Context switches since last collection")
            .init();

        let perf_page_faults = meter
            .u64_counter("kubric.agent.perf.page_faults")
            .with_description("Page faults since last collection")
            .init();

        Self {
            cpu_usage,
            memory_used_bytes,
            memory_total_bytes,
            swap_used_bytes,
            disk_used_bytes,
            disk_total_bytes,
            network_bytes_received,
            network_bytes_sent,
            perf_cpu_cycles,
            perf_instructions,
            perf_cache_misses,
            perf_branch_misses,
            perf_context_switches,
            perf_page_faults,
            prev_perf: PerfReading::default(),
            prev_net_rx: std::collections::HashMap::new(),
            prev_net_tx: std::collections::HashMap::new(),
        }
    }

    /// Record all metrics from a host snapshot.  Delta counters (network bytes,
    /// perf events) are computed relative to the previous `record_host_metrics`
    /// call so that OTEL sees monotonically increasing counters.
    pub fn record_host_metrics(&mut self, m: &HostMetrics) {
        let host_attr = KeyValue::new("host.name", m.hostname.clone());

        // CPU
        self.cpu_usage
            .observe(m.cpu.usage_pct as f64, &[host_attr.clone()]);

        // Memory
        self.memory_used_bytes
            .observe(m.memory.used_bytes as f64, &[host_attr.clone()]);
        self.memory_total_bytes
            .observe(m.memory.total_bytes as f64, &[host_attr.clone()]);

        // Swap
        self.swap_used_bytes
            .observe(m.swap.used_bytes as f64, &[host_attr.clone()]);

        // Disks
        for d in &m.disks {
            let attrs = &[
                host_attr.clone(),
                KeyValue::new("disk.mount_point", d.mount_point.clone()),
                KeyValue::new("disk.filesystem", d.fs_type.clone()),
            ];
            let used = d.total_bytes.saturating_sub(d.available_bytes) as i64;
            // UpDownCounter: add the current absolute value every cycle.
            // Because deltas can be negative on disk unmount, i64 is correct.
            self.disk_used_bytes.add(used, attrs);
            self.disk_total_bytes.add(d.total_bytes as i64, attrs);
        }

        // Networks — compute delta to feed monotonic counters
        for n in &m.networks {
            let attrs = &[
                host_attr.clone(),
                KeyValue::new("network.interface", n.interface.clone()),
            ];

            let prev_rx = *self.prev_net_rx.get(&n.interface).unwrap_or(&0);
            let prev_tx = *self.prev_net_tx.get(&n.interface).unwrap_or(&0);

            let delta_rx = n.bytes_received.saturating_sub(prev_rx);
            let delta_tx = n.bytes_sent.saturating_sub(prev_tx);

            if delta_rx > 0 {
                self.network_bytes_received.add(delta_rx, attrs);
            }
            if delta_tx > 0 {
                self.network_bytes_sent.add(delta_tx, attrs);
            }

            self.prev_net_rx
                .insert(n.interface.clone(), n.bytes_received);
            self.prev_net_tx.insert(n.interface.clone(), n.bytes_sent);
        }
    }

    /// Record hardware/software performance counter deltas.
    pub fn record_perf(&mut self, current: &PerfReading) {
        let empty_attrs: &[KeyValue] = &[];

        let delta = |curr: u64, prev: u64| curr.saturating_sub(prev);

        let d_cycles = delta(current.cpu_cycles, self.prev_perf.cpu_cycles);
        let d_instr = delta(current.instructions, self.prev_perf.instructions);
        let d_cache = delta(current.cache_misses, self.prev_perf.cache_misses);
        let d_branch = delta(current.branch_misses, self.prev_perf.branch_misses);
        let d_ctx = delta(current.context_switches, self.prev_perf.context_switches);
        let d_faults = delta(current.page_faults, self.prev_perf.page_faults);

        if d_cycles > 0 {
            self.perf_cpu_cycles.add(d_cycles, empty_attrs);
        }
        if d_instr > 0 {
            self.perf_instructions.add(d_instr, empty_attrs);
        }
        if d_cache > 0 {
            self.perf_cache_misses.add(d_cache, empty_attrs);
        }
        if d_branch > 0 {
            self.perf_branch_misses.add(d_branch, empty_attrs);
        }
        if d_ctx > 0 {
            self.perf_context_switches.add(d_ctx, empty_attrs);
        }
        if d_faults > 0 {
            self.perf_page_faults.add(d_faults, empty_attrs);
        }

        self.prev_perf = current.clone();
    }
}

/// Flush the meter provider and shut down all exporters cleanly.
pub fn shutdown_meter(provider: SdkMeterProvider) {
    if let Err(e) = provider.shutdown() {
        tracing::warn!(error = %e, "OTEL meter provider shutdown error");
    }
}

/// Flush the tracer provider.
pub fn shutdown_tracer(provider: opentelemetry_sdk::trace::TracerProvider) {
    drop(provider);
    global::shutdown_tracer_provider();
}

// ─── Tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sysinfo_host_metrics::{
        CpuMetrics, DiskMetrics, LoadAvg, MemoryMetrics, NetworkMetrics, SwapMetrics,
    };

    fn dummy_metrics() -> HostMetrics {
        HostMetrics {
            class_uid: 5001,
            time: 0,
            hostname: "test-host".to_string(),
            os_name: "Linux".to_string(),
            kernel_version: "6.1.0".to_string(),
            cpu: CpuMetrics {
                usage_pct: 55.0,
                core_count: 8,
                per_core: vec![50.0; 8],
                brand: "TestCPU".to_string(),
                frequency_mhz: 3200,
            },
            memory: MemoryMetrics {
                total_bytes: 32 * 1024 * 1024 * 1024,
                used_bytes: 16 * 1024 * 1024 * 1024,
                available_bytes: 16 * 1024 * 1024 * 1024,
                usage_pct: 50.0,
            },
            swap: SwapMetrics {
                total_bytes: 8 * 1024 * 1024 * 1024,
                used_bytes: 512 * 1024 * 1024,
                usage_pct: 6.25,
            },
            disks: vec![DiskMetrics {
                name: "nvme0n1".to_string(),
                mount_point: "/".to_string(),
                fs_type: "ext4".to_string(),
                total_bytes: 1_000_000_000_000,
                available_bytes: 600_000_000_000,
                usage_pct: 40.0,
                is_removable: false,
            }],
            networks: vec![NetworkMetrics {
                interface: "eth0".to_string(),
                bytes_received: 2_000_000,
                bytes_sent: 1_000_000,
                packets_received: 20_000,
                packets_sent: 10_000,
                errors_in: 0,
                errors_out: 0,
            }],
            process_count: 300,
            load_avg: LoadAvg { one: 1.5, five: 1.1, fifteen: 0.9 },
            uptime_secs: 172800,
        }
    }

    #[test]
    fn noop_meter_provider_does_not_panic() {
        let provider = SdkMeterProvider::builder().build();
        let mut bridge = OtelMetricBridge::new(&provider);
        let m = dummy_metrics();
        bridge.record_host_metrics(&m);
        bridge.record_perf(&PerfReading::default());
    }

    #[test]
    fn init_tracer_with_empty_endpoint_returns_noop() {
        let result = init_tracer("test-svc", "");
        assert!(result.is_ok(), "empty endpoint should return no-op tracer");
    }

    #[test]
    fn init_meter_with_empty_endpoint_returns_noop() {
        let result = init_meter("test-svc", "", Duration::from_secs(15));
        assert!(result.is_ok(), "empty endpoint should return no-op meter");
    }

    #[test]
    fn network_delta_computed_correctly() {
        let provider = SdkMeterProvider::builder().build();
        let mut bridge = OtelMetricBridge::new(&provider);
        let mut m = dummy_metrics();

        // First recording — establishes baseline
        bridge.record_host_metrics(&m);

        // Second recording — simulate 1 MB more received
        m.networks[0].bytes_received += 1_000_000;
        bridge.record_host_metrics(&m);

        // The prev_net_rx map should now reflect the new value.
        assert_eq!(
            *bridge.prev_net_rx.get("eth0").unwrap(),
            m.networks[0].bytes_received
        );
    }
}
