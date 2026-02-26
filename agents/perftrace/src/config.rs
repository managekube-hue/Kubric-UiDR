use anyhow::{bail, Result};

#[derive(Debug, Clone)]
pub struct Config {
    pub tenant_id: String,
    pub nats_url: String,
    pub agent_id: String,
    /// Metric poll interval in seconds
    pub poll_interval_secs: u64,
}

impl Config {
    pub fn from_env() -> Result<Self> {
        let tenant_id = std::env::var("KUBRIC_TENANT_ID").unwrap_or_default();
        if tenant_id.is_empty() {
            bail!("KUBRIC_TENANT_ID must be set");
        }
        Ok(Self {
            tenant_id,
            nats_url: std::env::var("KUBRIC_NATS_URL")
                .unwrap_or_else(|_| "nats://127.0.0.1:4222".to_string()),
            agent_id: std::env::var("KUBRIC_AGENT_ID")
                .unwrap_or_else(|_| "perftrace-stub".to_string()),
            poll_interval_secs: std::env::var("KUBRIC_POLL_INTERVAL")
                .ok()
                .and_then(|v| v.parse().ok())
                .unwrap_or(15),
        })
    }
}
