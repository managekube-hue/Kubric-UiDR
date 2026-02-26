use anyhow::{Context, Result};
use tracing::info;

mod agent;
mod config;
mod event;

#[tokio::main]
async fn main() -> Result<()> {
    // Load .env if present (dev only — production uses Vault-injected env)
    let _ = dotenvy::dotenv();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_env("KUBRIC_LOG")
                .add_directive("coresec=info".parse().unwrap()),
        )
        .json()
        .init();

    let cfg = config::Config::from_env().context("load config")?;
    info!(tenant_id = %cfg.tenant_id, nats_url = %cfg.nats_url, "CoreSec starting");

    agent::run(cfg).await
}
