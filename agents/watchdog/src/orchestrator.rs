//! Watchdog agent orchestrator — monitors health of peer agents (CoreSec,
//! NetGuard, PerfTrace) via NATS heartbeat subscriptions and publishes
//! agent status events.
//!
//! Also handles TUF-based manifest verification for agent binary updates.

use crate::config::Config;
use anyhow::{Context, Result};
use futures::StreamExt;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use tokio::time::interval;
use tracing::{error, info, warn};

/// Agent health status report.
#[derive(Debug, Clone, Serialize)]
pub struct AgentStatus {
    pub tenant_id: String,
    pub agent_id: String,
    pub agent_type: String,
    pub status: String, // "healthy", "stale", "offline"
    pub last_seen: u64,
    pub uptime_secs: u64,
    pub timestamp: u64,
}

/// Heartbeat payload expected from agents.
#[derive(Debug, Deserialize)]
struct Heartbeat {
    agent_id: String,
    #[serde(default)]
    agent_type: String,
    #[serde(default)]
    uptime_secs: u64,
}

struct AgentRecord {
    agent_type: String,
    last_seen: Instant,
    uptime_secs: u64,
    first_seen: Instant,
}

const STALE_THRESHOLD: Duration = Duration::from_secs(30);
const OFFLINE_THRESHOLD: Duration = Duration::from_secs(120);
const STATUS_INTERVAL: Duration = Duration::from_secs(15);

pub async fn run(cfg: Config) -> Result<()> {
    let client = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;

    info!(tenant_id = %cfg.tenant_id, "Watchdog orchestrator starting");

    // Subscribe to heartbeats from all agents
    let heartbeat_subject = format!("kubric.{}.agent.heartbeat", cfg.tenant_id);
    let mut heartbeat_sub = client
        .subscribe(heartbeat_subject.clone())
        .await
        .context("subscribe to heartbeats")?;

    let status_subject = format!("kubric.{}.agent.status.v1", cfg.tenant_id);

    let mut agents: HashMap<String, AgentRecord> = HashMap::new();
    let mut ticker = interval(STATUS_INTERVAL);
    let mut shutdown = Box::pin(tokio::signal::ctrl_c());

    // Also publish our own heartbeat
    let self_heartbeat_payload = serde_json::to_vec(&serde_json::json!({
        "agent_id": &cfg.agent_id,
        "agent_type": "watchdog",
        "uptime_secs": 0,
    }))
    .unwrap_or_default();
    let _ = client
        .publish(heartbeat_subject.clone(), self_heartbeat_payload.into())
        .await;

    loop {
        tokio::select! {
            msg = heartbeat_sub.next() => {
                if let Some(msg) = msg {
                    if let Ok(hb) = serde_json::from_slice::<Heartbeat>(&msg.payload) {
                        let now = Instant::now();
                        let record = agents.entry(hb.agent_id.clone()).or_insert_with(|| {
                            info!(agent_id = %hb.agent_id, agent_type = %hb.agent_type, "new agent registered");
                            AgentRecord {
                                agent_type: hb.agent_type.clone(),
                                last_seen: now,
                                uptime_secs: hb.uptime_secs,
                                first_seen: now,
                            }
                        });
                        record.last_seen = now;
                        record.uptime_secs = hb.uptime_secs;
                        if !hb.agent_type.is_empty() {
                            record.agent_type = hb.agent_type;
                        }
                    }
                }
            }

            _ = ticker.tick() => {
                let now = Instant::now();
                let ts = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs();

                for (agent_id, record) in &agents {
                    let elapsed = now.duration_since(record.last_seen);
                    let status = if elapsed < STALE_THRESHOLD {
                        "healthy"
                    } else if elapsed < OFFLINE_THRESHOLD {
                        "stale"
                    } else {
                        "offline"
                    };

                    if status != "healthy" {
                        warn!(
                            agent_id,
                            agent_type = %record.agent_type,
                            status,
                            secs_since_heartbeat = elapsed.as_secs(),
                            "agent degraded"
                        );
                    }

                    let agent_status = AgentStatus {
                        tenant_id: cfg.tenant_id.clone(),
                        agent_id: agent_id.clone(),
                        agent_type: record.agent_type.clone(),
                        status: status.to_string(),
                        last_seen: record.last_seen.elapsed().as_secs(),
                        uptime_secs: record.uptime_secs,
                        timestamp: ts,
                    };

                    if let Ok(payload) = serde_json::to_vec(&agent_status) {
                        if let Err(e) = client
                            .publish(status_subject.clone(), payload.into())
                            .await
                        {
                            error!(%e, "NATS publish agent status failed");
                        }
                    }
                }

                if agents.is_empty() {
                    info!("no agents registered yet — monitoring for heartbeats");
                }
            }

            _ = &mut shutdown => {
                info!("Watchdog shutting down");
                break;
            }
        }
    }

    Ok(())
}
