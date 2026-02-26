use anyhow::{Context, Result};
use tracing::info;

mod config;
mod fingerprint;
mod install_script;
mod registration;

#[tokio::main]
async fn main() -> Result<()> {
    let _ = dotenvy::dotenv();

    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_env("KUBRIC_LOG")
                .add_directive("provisioning=info".parse().unwrap()),
        )
        .json()
        .init();

    let cfg = config::Config::from_env().context("load config")?;
    info!(nats_url = %cfg.nats_url, "Provisioning agent starting");

    registration::run(cfg).await
}
