use anyhow::{Context, Result};
use tracing::info;

mod capture;
mod config;
mod dpi;
mod ids;
mod ipsum_lookup;
mod ndpi_ffi;
mod rita_client;
mod tls;
mod waf;
mod zeek;

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
    info!(tenant_id = %cfg.tenant_id, interface = %cfg.interface, "NetGuard starting");

    capture::run(cfg).await
}
