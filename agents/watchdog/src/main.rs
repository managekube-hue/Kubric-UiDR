use anyhow::{Context, Result};
use tracing::info;

mod config;
mod manifest;
mod orchestrator;
mod tuf_updater;
mod zstd_delta;

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

    orchestrator::run(cfg).await
}
