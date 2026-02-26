use anyhow::{bail, Result};

#[derive(Debug, Clone)]
pub struct Config {
    /// Tenant identifier — propagated into every NATS subject and event payload.
    pub tenant_id: String,
    pub nats_url: String,
    pub agent_id: String,
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
                .unwrap_or_else(|_| hostname()),
        })
    }
}

fn hostname() -> String {
    std::env::var("HOSTNAME").unwrap_or_else(|_| "unknown".to_string())
}
