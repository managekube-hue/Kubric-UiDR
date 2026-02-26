use anyhow::{Context, Result};
use tracing::info;

mod config;

#[tokio::main]
async fn main() -> Result<()> {
    let _ = dotenvy::dotenv();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_env("KUBRIC_LOG")
                .add_directive("watchdog=info".parse().unwrap()),
        )
        .json()
        .init();

    let cfg = config::Config::from_env().context("load config")?;
    info!(tenant_id = %cfg.tenant_id, "Watchdog starting");

    // Phase 1 stub — parks until signal.
    // Phase 3 will wire: Vault TUF → signed agent binary fetch → restart other agents
    info!("Watchdog stub running — awaiting Phase 3 TUF/Vault wiring");
    tokio::signal::ctrl_c().await.ok();
    info!("Watchdog shutting down");
    Ok(())
}
