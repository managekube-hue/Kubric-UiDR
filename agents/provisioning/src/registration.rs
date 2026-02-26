//! Agent registration handler — accepts registration requests from new agents
//! via NATS and provisions them with tenant-scoped credentials.
//!
//! Subscribes to: kubric.provisioning.register
//! Publishes to:  kubric.provisioning.registered
//!
//! Registration flow:
//! 1. Agent sends a RegistrationRequest with its fingerprint
//! 2. Provisioning validates the fingerprint against known binaries
//! 3. Provisions a tenant-scoped NATS token + Vault AppRole
//! 4. Returns RegistrationResponse with credentials

use crate::config::Config;
use crate::fingerprint;
use anyhow::{Context, Result};
use futures::StreamExt;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::time::{interval, Duration};
use tracing::{error, info, warn};

/// Registration request from a new agent.
#[derive(Debug, Deserialize, Serialize)]
pub struct RegistrationRequest {
    pub agent_type: String,
    pub hostname: String,
    pub os: String,
    pub arch: String,
    pub binary_hash: String,
    pub requested_tenant_id: String,
}

/// Registration response sent back to the agent.
#[derive(Debug, Serialize, Deserialize)]
pub struct RegistrationResponse {
    pub agent_id: String,
    pub tenant_id: String,
    pub nats_token: String,
    pub vault_role_id: String,
    pub vault_secret_id: String,
    pub approved: bool,
    pub reason: String,
    pub timestamp: u64,
}

/// Agent registration record stored in-memory.
#[derive(Debug, Clone, Serialize)]
pub struct AgentRecord {
    pub agent_id: String,
    pub agent_type: String,
    pub tenant_id: String,
    pub hostname: String,
    pub os: String,
    pub arch: String,
    pub binary_hash: String,
    pub registered_at: u64,
    pub last_seen: u64,
}

pub async fn run(cfg: Config) -> Result<()> {
    let client = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;

    info!("Provisioning agent: listening for registration requests");

    let mut sub = client
        .subscribe("kubric.provisioning.register")
        .await
        .context("subscribe to registration subject")?;

    let mut agents: Vec<AgentRecord> = Vec::new();
    let mut shutdown = Box::pin(tokio::signal::ctrl_c());

    loop {
        tokio::select! {
            msg = sub.next() => {
                let Some(msg) = msg else { break };

                match serde_json::from_slice::<RegistrationRequest>(&msg.payload) {
                    Ok(req) => {
                        let response = process_registration(&cfg, &req, &mut agents).await;

                        if let Ok(payload) = serde_json::to_vec(&response) {
                            // Reply to the requesting agent
                            if let Some(reply) = msg.reply {
                                if let Err(e) = client.publish(reply, payload.clone().into()).await {
                                    error!(%e, "reply to registration failed");
                                }
                            }

                            // Also publish to the registered topic for monitoring
                            let _ = client.publish(
                                "kubric.provisioning.registered",
                                payload.into(),
                            ).await;
                        }
                    }
                    Err(e) => {
                        warn!(%e, "invalid registration request");
                    }
                }
            }

            _ = &mut shutdown => {
                info!("Provisioning agent shutting down");
                break;
            }
        }
    }

    Ok(())
}

async fn process_registration(
    cfg: &Config,
    req: &RegistrationRequest,
    agents: &mut Vec<AgentRecord>,
) -> RegistrationResponse {
    let ts = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();

    let agent_id = format!(
        "{}-{}-{}",
        req.agent_type,
        &req.hostname,
        &uuid::Uuid::new_v4().to_string()[..8]
    );

    // Validate binary hash against known agent fingerprints
    let hash_valid = fingerprint::validate_agent_hash(
        &req.agent_type,
        &req.binary_hash,
        &cfg.binary_dir,
    );

    if !hash_valid {
        warn!(
            agent_type = %req.agent_type,
            hostname = %req.hostname,
            hash = %req.binary_hash,
            "registration rejected — unknown binary hash"
        );
        return RegistrationResponse {
            agent_id,
            tenant_id: req.requested_tenant_id.clone(),
            nats_token: String::new(),
            vault_role_id: String::new(),
            vault_secret_id: String::new(),
            approved: false,
            reason: "Binary hash does not match any known agent build".into(),
            timestamp: ts,
        };
    }

    // Register the agent
    let record = AgentRecord {
        agent_id: agent_id.clone(),
        agent_type: req.agent_type.clone(),
        tenant_id: req.requested_tenant_id.clone(),
        hostname: req.hostname.clone(),
        os: req.os.clone(),
        arch: req.arch.clone(),
        binary_hash: req.binary_hash.clone(),
        registered_at: ts,
        last_seen: ts,
    };

    info!(
        agent_id = %agent_id,
        agent_type = %req.agent_type,
        tenant_id = %req.requested_tenant_id,
        hostname = %req.hostname,
        "agent registered"
    );

    agents.push(record);

    // In production, NATS token and Vault credentials are provisioned
    // from the actual NATS auth server and Vault AppRole backend.
    // For now, generate placeholder tokens.
    let nats_token = format!("nats-{}-{}", req.requested_tenant_id, &agent_id[..8]);
    let vault_role_id = format!("kubric-{}-{}", req.requested_tenant_id, req.agent_type);

    RegistrationResponse {
        agent_id,
        tenant_id: req.requested_tenant_id.clone(),
        nats_token,
        vault_role_id,
        vault_secret_id: "provisioned-at-runtime".into(),
        approved: true,
        reason: "Binary verified, registration approved".into(),
        timestamp: ts,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn registration_request_deserializes() {
        let json = r#"{
            "agent_type": "coresec",
            "hostname": "endpoint-01",
            "os": "linux",
            "arch": "x86_64",
            "binary_hash": "abc123",
            "requested_tenant_id": "tenant-1"
        }"#;
        let req: RegistrationRequest = serde_json::from_str(json).unwrap();
        assert_eq!(req.agent_type, "coresec");
        assert_eq!(req.requested_tenant_id, "tenant-1");
    }

    #[test]
    fn registration_response_serializes() {
        let resp = RegistrationResponse {
            agent_id: "coresec-host-abc12345".into(),
            tenant_id: "tenant-1".into(),
            nats_token: "nats-token".into(),
            vault_role_id: "role-id".into(),
            vault_secret_id: "secret-id".into(),
            approved: true,
            reason: "approved".into(),
            timestamp: 1700000000,
        };
        let json = serde_json::to_string(&resp).unwrap();
        assert!(json.contains("approved"));
        assert!(json.contains("coresec-host"));
    }
}
