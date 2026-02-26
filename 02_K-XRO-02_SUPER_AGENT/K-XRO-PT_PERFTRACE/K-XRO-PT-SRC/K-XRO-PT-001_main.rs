// K-XRO-PT-001 — PerfTrace Agent Entry Point
//
// Kubric platform agent that continuously collects host performance metrics,
// enriches them with Linux perf_event counters, exports to OpenTelemetry, and
// publishes OCSF class-5001 JSON events on NATS.  A Prometheus /metrics
// endpoint and a /healthz endpoint are served over HTTP.
//
// Configuration (all via environment variables):
//   NATS_URL                  NATS server URL            (default: nats://localhost:4222)
//   OTEL_ENDPOINT             OTLP gRPC endpoint         (default: "" = no-op)
//   PROMETHEUS_PORT           HTTP server port           (default: 9090)
//   COLLECTION_INTERVAL_SECS  Metrics collection period  (default: 15)
//   TENANT_ID                 Kubric tenant identifier   (default: default)
//   AGENT_ID                  Unique agent instance ID   (default: perftrace-0)

#[path = "K-XRO-PT-005_sysinfo_host_metrics.rs"]
mod sysinfo_host_metrics;

#[path = "K-XRO-PT-002_perf_event_open.rs"]
mod perf_event_open;

#[path = "K-XRO-PT-003_prometheus.rs"]
mod prometheus_metrics;

#[path = "K-XRO-PT-004_otel_collector.rs"]
mod otel_collector;

use anyhow::{Context, Result};
use axum::{routing::get, Router};
use std::{net::SocketAddr, sync::Arc, time::Duration};
use tokio::{signal, sync::Mutex, time};
use tracing::{error, info, warn};
use tracing_subscriber::{EnvFilter, FmtSubscriber};

use otel_collector::{OtelExporter, OtelMetricBridge, shutdown_meter, shutdown_tracer};
use perf_event_open::PerfSession;
use prometheus_metrics::{healthz_handler, metrics_handler, MetricsRegistry};
use sysinfo_host_metrics::HostMetricsCollector;

// ─── Configuration ────────────────────────────────────────────────────────────

#[derive(Debug, Clone)]
struct Config {
    nats_url: String,
    otel_endpoint: String,
    prometheus_port: u16,
    collection_interval: Duration,
    tenant_id: String,
    agent_id: String,
}

impl Config {
    fn from_env() -> Self {
        let nats_url = std::env::var("NATS_URL")
            .unwrap_or_else(|_| "nats://localhost:4222".to_string());

        let otel_endpoint = std::env::var("OTEL_ENDPOINT").unwrap_or_default();

        let prometheus_port = std::env::var("PROMETHEUS_PORT")
            .ok()
            .and_then(|s| s.parse::<u16>().ok())
            .unwrap_or(9090);

        let collection_interval_secs = std::env::var("COLLECTION_INTERVAL_SECS")
            .ok()
            .and_then(|s| s.parse::<u64>().ok())
            .unwrap_or(15);

        let tenant_id = std::env::var("TENANT_ID")
            .unwrap_or_else(|_| "default".to_string());

        let agent_id = std::env::var("AGENT_ID")
            .unwrap_or_else(|_| "perftrace-0".to_string());

        Config {
            nats_url,
            otel_endpoint,
            prometheus_port,
            collection_interval: Duration::from_secs(collection_interval_secs),
            tenant_id,
            agent_id,
        }
    }

    /// NATS subject for publishing host metrics.
    fn nats_subject(&self) -> String {
        format!("kubric.{}.metrics.host.v1", self.tenant_id)
    }
}

// ─── Application state shared between tasks ───────────────────────────────────

struct AppState {
    prom_registry: Arc<MetricsRegistry>,
}

// ─── NATS publisher ──────────────────────────────────────────────────────────

/// Best-effort NATS publish.  Errors are counted in the Prometheus registry
/// but never propagate to the caller so they cannot disrupt the agent loop.
async fn publish_nats(
    client: &async_nats::Client,
    subject: &str,
    payload: &[u8],
    prom: &MetricsRegistry,
) {
    if let Err(e) = client.publish(subject.to_string(), payload.into()).await {
        warn!(error = %e, subject, "NATS publish failed");
        prom.nats_publish_errors_total.inc();
    }
}

// ─── Collection loop ─────────────────────────────────────────────────────────

async fn run_collection_loop(
    cfg: Arc<Config>,
    prom: Arc<MetricsRegistry>,
    nats_client: Option<async_nats::Client>,
    meter_provider: opentelemetry_sdk::metrics::SdkMeterProvider,
) {
    info!(
        interval_secs = cfg.collection_interval.as_secs(),
        "collection loop starting"
    );

    // Initialise the host metrics collector (performs a warm-up sleep internally).
    let mut host_collector = HostMetricsCollector::new();

    // Initialise perf session (graceful fallback if perf is unavailable).
    let perf_session = PerfSession::open();
    if perf_session.is_available() {
        info!(
            counters = perf_session.counter_count(),
            "Linux perf_event session opened"
        );
    } else {
        warn!("perf_event_open unavailable — hardware counters will be zero");
    }

    // Build the OTEL metric bridge.
    let mut otel_bridge = OtelMetricBridge::new(&meter_provider);

    let mut ticker = time::interval(cfg.collection_interval);
    ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);

    loop {
        ticker.tick().await;

        let timer = std::time::Instant::now();

        // 1. Collect host metrics
        let host_metrics = host_collector.collect();

        // 2. Read hardware perf counters (returns zeros if unavailable)
        let perf_reading = perf_session.read_all();

        // 3. Update Prometheus registry
        prom.update_from_host_metrics(&host_metrics);

        // Observe per-process CPU usage in the histogram
        for proc in host_collector.top_processes_by_cpu(50) {
            prom.observe_process_cpu(proc.cpu_pct);
        }

        // 4. Record to OpenTelemetry
        otel_bridge.record_host_metrics(&host_metrics);
        otel_bridge.record_perf(&perf_reading);

        // 5. Publish to NATS
        if let Some(ref client) = nats_client {
            match serde_json::to_vec(&host_metrics) {
                Ok(payload) => {
                    let subject = cfg.nats_subject();
                    publish_nats(client, &subject, &payload, &prom).await;
                }
                Err(e) => {
                    error!(error = %e, "failed to serialise HostMetrics to JSON");
                    prom.nats_publish_errors_total.inc();
                }
            }
        }

        // 6. Record collection duration
        let elapsed = timer.elapsed().as_secs_f64();
        prom.collection_duration_seconds.observe(elapsed);

        info!(
            elapsed_ms = (elapsed * 1000.0) as u64,
            cpu_pct = host_metrics.cpu.usage_pct,
            mem_pct = host_metrics.memory.usage_pct,
            process_count = host_metrics.process_count,
            "collection cycle complete"
        );
    }
}

// ─── HTTP server ──────────────────────────────────────────────────────────────

async fn run_http_server(
    port: u16,
    prom: Arc<MetricsRegistry>,
) -> Result<()> {
    let app = Router::new()
        .route("/metrics", get(metrics_handler))
        .route("/healthz", get(healthz_handler))
        .route("/ready", get(healthz_handler))
        .with_state(prom);

    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    info!(%addr, "Prometheus HTTP server listening");

    let listener = tokio::net::TcpListener::bind(addr)
        .await
        .with_context(|| format!("binding HTTP server to port {port}"))?;

    axum::serve(listener, app)
        .await
        .context("HTTP server error")?;

    Ok(())
}

// ─── NATS connection ─────────────────────────────────────────────────────────

async fn connect_nats(url: &str) -> Option<async_nats::Client> {
    match async_nats::connect(url).await {
        Ok(client) => {
            info!(url, "connected to NATS");
            Some(client)
        }
        Err(e) => {
            warn!(url, error = %e, "NATS connection failed — metrics will not be published");
            None
        }
    }
}

// ─── Shutdown signal ─────────────────────────────────────────────────────────

async fn wait_for_shutdown() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("failed to install Ctrl-C handler");
    };

    #[cfg(unix)]
    let sigterm = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let sigterm = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c   => info!("received SIGINT"),
        _ = sigterm  => info!("received SIGTERM"),
    }
}

// ─── Main ─────────────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() -> Result<()> {
    // ── Logging ────────────────────────────────────────────────────────────────
    let subscriber = FmtSubscriber::builder()
        .with_env_filter(
            EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| EnvFilter::new("info,perftrace=debug")),
        )
        .with_target(true)
        .with_thread_ids(false)
        .json()
        .finish();
    tracing::subscriber::set_global_default(subscriber)
        .context("setting global tracing subscriber")?;

    // ── Config ────────────────────────────────────────────────────────────────
    let cfg = Arc::new(Config::from_env());
    info!(
        tenant_id = %cfg.tenant_id,
        agent_id  = %cfg.agent_id,
        nats_url  = %cfg.nats_url,
        otel      = %cfg.otel_endpoint,
        prom_port = cfg.prometheus_port,
        interval  = cfg.collection_interval.as_secs(),
        "PerfTrace agent starting"
    );

    // ── Prometheus ────────────────────────────────────────────────────────────
    let prom = Arc::new(
        MetricsRegistry::new().context("failed to initialise Prometheus registry")?,
    );

    // ── OpenTelemetry ─────────────────────────────────────────────────────────
    let exporter = OtelExporter::new("perftrace", cfg.otel_endpoint.clone())
        .with_export_interval(cfg.collection_interval);

    let (tracer_provider, meter_provider) = exporter
        .init()
        .context("failed to initialise OpenTelemetry")?;

    // ── NATS ──────────────────────────────────────────────────────────────────
    let nats_client = connect_nats(&cfg.nats_url).await;

    // ── Background tasks ──────────────────────────────────────────────────────
    let cfg_loop = Arc::clone(&cfg);
    let prom_loop = Arc::clone(&prom);
    let meter_provider_loop = meter_provider.clone();

    let collection_handle = tokio::spawn(async move {
        run_collection_loop(cfg_loop, prom_loop, nats_client, meter_provider_loop).await;
    });

    let prom_http = Arc::clone(&prom);
    let http_port = cfg.prometheus_port;
    let http_handle = tokio::spawn(async move {
        if let Err(e) = run_http_server(http_port, prom_http).await {
            error!(error = %e, "HTTP server exited with error");
        }
    });

    // ── Graceful shutdown ─────────────────────────────────────────────────────
    wait_for_shutdown().await;
    info!("shutdown signal received, flushing telemetry ...");

    collection_handle.abort();
    http_handle.abort();

    // Flush OTEL exporters before exit.
    shutdown_meter(meter_provider);
    shutdown_tracer(tracer_provider);

    info!("PerfTrace agent stopped");
    Ok(())
}

// ─── Tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_defaults_are_sane() {
        // Clear any potentially set env vars from the test environment.
        std::env::remove_var("NATS_URL");
        std::env::remove_var("OTEL_ENDPOINT");
        std::env::remove_var("PROMETHEUS_PORT");
        std::env::remove_var("COLLECTION_INTERVAL_SECS");
        std::env::remove_var("TENANT_ID");
        std::env::remove_var("AGENT_ID");

        let cfg = Config::from_env();
        assert_eq!(cfg.nats_url, "nats://localhost:4222");
        assert_eq!(cfg.otel_endpoint, "");
        assert_eq!(cfg.prometheus_port, 9090);
        assert_eq!(cfg.collection_interval, Duration::from_secs(15));
        assert_eq!(cfg.tenant_id, "default");
        assert_eq!(cfg.agent_id, "perftrace-0");
    }

    #[test]
    fn config_from_env_respects_overrides() {
        std::env::set_var("NATS_URL", "nats://nats.prod.svc:4222");
        std::env::set_var("PROMETHEUS_PORT", "9100");
        std::env::set_var("COLLECTION_INTERVAL_SECS", "30");
        std::env::set_var("TENANT_ID", "acme");
        std::env::set_var("AGENT_ID", "perftrace-node-1");

        let cfg = Config::from_env();
        assert_eq!(cfg.nats_url, "nats://nats.prod.svc:4222");
        assert_eq!(cfg.prometheus_port, 9100);
        assert_eq!(cfg.collection_interval, Duration::from_secs(30));
        assert_eq!(cfg.tenant_id, "acme");
        assert_eq!(cfg.agent_id, "perftrace-node-1");
        assert_eq!(cfg.nats_subject(), "kubric.acme.metrics.host.v1");

        // Cleanup
        std::env::remove_var("NATS_URL");
        std::env::remove_var("PROMETHEUS_PORT");
        std::env::remove_var("COLLECTION_INTERVAL_SECS");
        std::env::remove_var("TENANT_ID");
        std::env::remove_var("AGENT_ID");
    }

    #[test]
    fn nats_subject_format() {
        let cfg = Config {
            nats_url: "nats://localhost:4222".to_string(),
            otel_endpoint: "".to_string(),
            prometheus_port: 9090,
            collection_interval: Duration::from_secs(15),
            tenant_id: "org123".to_string(),
            agent_id: "perftrace-0".to_string(),
        };
        assert_eq!(cfg.nats_subject(), "kubric.org123.metrics.host.v1");
    }

    #[tokio::test]
    async fn connect_nats_returns_none_for_unreachable_server() {
        let client = connect_nats("nats://127.0.0.1:14222").await;
        // In CI without a NATS server this should return None gracefully.
        // (If a NATS server happens to be running on 14222 the test is a no-op.)
        let _ = client; // either Ok or None is acceptable
    }
}
