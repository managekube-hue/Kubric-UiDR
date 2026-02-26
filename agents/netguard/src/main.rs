use anyhow::{Context, Result};
use tracing::info;

mod config;

#[tokio::main]
async fn main() -> Result<()> {
    let _ = dotenvy::dotenv();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_env("KUBRIC_LOG")
                .add_directive("netguard=info".parse().unwrap()),
        )
        .json()
        .init();

    let cfg = config::Config::from_env().context("load config")?;
    info!(tenant_id = %cfg.tenant_id, "NetGuard starting");

    // Phase 1 stub — parks until signal.
    // Phase 3 will wire: libpcap/AF_PACKET → OCSF NetworkActivity → NATS
    info!("NetGuard stub running — awaiting Phase 3 pcap/eBPF wiring");
    tokio::signal::ctrl_c().await.ok();
    info!("NetGuard shutting down");
    Ok(())
}
