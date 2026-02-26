use anyhow::{Context, Result};
use tracing::info;

mod config;

#[tokio::main]
async fn main() -> Result<()> {
    let _ = dotenvy::dotenv();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_env("KUBRIC_LOG")
                .add_directive("perftrace=info".parse().unwrap()),
        )
        .json()
        .init();

    let cfg = config::Config::from_env().context("load config")?;
    info!(tenant_id = %cfg.tenant_id, "PerfTrace starting");

    // Phase 1 stub — parks until signal.
    // Phase 3 will wire: sysinfo → OCSF PerformanceMetric (class 4004) → NATS
    info!("PerfTrace stub running — awaiting Phase 3 sysinfo wiring");
    tokio::signal::ctrl_c().await.ok();
    info!("PerfTrace shutting down");
    Ok(())
}
