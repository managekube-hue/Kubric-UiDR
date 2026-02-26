use anyhow::{bail, Result};

#[derive(Debug, Clone)]
pub struct Config {
    pub nats_url: String,
    pub agent_id: String,
    /// Base URL for the provisioning API (where agents register).
    pub api_base_url: String,
    /// Directory containing agent binaries for fingerprinting.
    pub binary_dir: String,
    /// Vault address for secrets injection.
    pub vault_addr: String,
}

impl Config {
    pub fn from_env() -> Result<Self> {
        Ok(Self {
            nats_url: std::env::var("KUBRIC_NATS_URL")
                .unwrap_or_else(|_| "nats://127.0.0.1:4222".to_string()),
            agent_id: std::env::var("KUBRIC_AGENT_ID")
                .unwrap_or_else(|_| "provisioning-0".to_string()),
            api_base_url: std::env::var("KUBRIC_API_BASE_URL")
                .unwrap_or_else(|_| "http://ksvc:8080".to_string()),
            binary_dir: std::env::var("KUBRIC_BINARY_DIR")
                .unwrap_or_else(|_| "/opt/kubric/bin".to_string()),
            vault_addr: std::env::var("VAULT_ADDR")
                .unwrap_or_else(|_| "http://127.0.0.1:8200".to_string()),
        })
    }
}
