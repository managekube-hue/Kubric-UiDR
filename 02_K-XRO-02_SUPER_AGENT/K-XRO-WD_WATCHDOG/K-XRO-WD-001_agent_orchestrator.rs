//! K-XRO-WD-001 — Watchdog agent orchestrator.
//!
//! Monitors the health of peer Kubric agents (CoreSec, NetGuard, PerfTrace)
//! by subscribing to their NATS heartbeat subjects.  Publishes `AgentStatus`
//! events on a 15-second tick to drive the Grafana health dashboard.
//!
//! # NATS subjects
//! | Direction | Subject                                      |
//! |-----------|----------------------------------------------|
//! | Subscribe | `kubric.{tenant_id}.agent.heartbeat`         |
//! | Publish   | `kubric.{tenant_id}.agent.status.v1`         |
//!
//! # Agent lifecycle
//! ```text
//! Heartbeat received → record Instant::now() in agents map
//! 15-second tick →
//!   elapsed < 30 s  → "healthy"
//!   elapsed < 120 s → "stale"   (warn log)
//!   elapsed ≥ 120 s → "offline" (warn log)
//! ```
//!
//! # Shutdown
//! Graceful shutdown on SIGINT / Ctrl-C.  Publishes a final status sweep.

use crate::config::Config;
use anyhow::{Context, Result};
use futures::StreamExt;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use tokio::time::interval;
use tracing::{error, info, warn};

// ── Thresholds ────────────────────────────────────────────────────────────────

/// Elapsed time after which an agent is considered stale.
const STALE_THRESHOLD: Duration = Duration::from_secs(30);
/// Elapsed time after which an agent is classified as offline.
const OFFLINE_THRESHOLD: Duration = Duration::from_secs(120);
/// How often to sweep all agents and publish status.
const STATUS_INTERVAL: Duration = Duration::from_secs(15);

// ── AgentStatus ───────────────────────────────────────────────────────────────

/// Published status event for a single monitored agent.
#[derive(Debug, Clone, Serialize)]
pub struct AgentStatus {
    /// Kubric tenant ID.
    pub tenant_id: String,
    /// Unique agent identifier (from heartbeat payload).
    pub agent_id: String,
    /// Agent type string: "coresec", "netguard", "perftrace", "watchdog".
    pub agent_type: String,
    /// "healthy", "stale", or "offline".
    pub status: String,
    /// Seconds elapsed since the last heartbeat was received.
    pub last_seen_secs_ago: u64,
    /// Uptime in seconds reported by the agent (from its heartbeat).
    pub uptime_secs: u64,
    /// UTC epoch seconds when this status event was created.
    pub timestamp: u64,
}

// ── Heartbeat payload ─────────────────────────────────────────────────────────

/// Expected JSON structure published by every agent on the heartbeat subject.
#[derive(Debug, Deserialize)]
struct Heartbeat {
    agent_id: String,
    #[serde(default)]
    agent_type: String,
    #[serde(default)]
    uptime_secs: u64,
}

// ── Internal per-agent record ─────────────────────────────────────────────────

struct AgentRecord {
    agent_type: String,
    last_seen: Instant,
    uptime_secs: u64,
}

// ── run ───────────────────────────────────────────────────────────────────────

/// Main entry point for the watchdog orchestrator loop.
///
/// Called from `main.rs` after config is loaded.
pub async fn run(cfg: Config) -> Result<()> {
    let client = async_nats::connect(&cfg.nats_url)
        .await
        .context("watchdog: connect to NATS")?;

    info!(
        tenant_id = %cfg.tenant_id,
        agent_id = %cfg.agent_id,
        nats_url = %cfg.nats_url,
        "Watchdog orchestrator starting"
    );

    let heartbeat_subject = format!("kubric.{}.agent.heartbeat", cfg.tenant_id);
    let status_subject = format!("kubric.{}.agent.status.v1", cfg.tenant_id);

    // Subscribe to all agent heartbeats
    let mut heartbeat_sub = client
        .subscribe(heartbeat_subject.clone())
        .await
        .context("watchdog: subscribe to heartbeat subject")?;

    // Publish our own heartbeat immediately so peers can detect us
    publish_self_heartbeat(&client, &heartbeat_subject, &cfg).await;

    let mut agents: HashMap<String, AgentRecord> = HashMap::new();
    let mut ticker = interval(STATUS_INTERVAL);
    let mut self_hb_ticker = interval(Duration::from_secs(10));
    let shutdown = tokio::signal::ctrl_c();
    tokio::pin!(shutdown);

    let mut last_summary_count = 0usize;

    loop {
        tokio::select! {
            // ── Heartbeat received ────────────────────────────────────────
            msg = heartbeat_sub.next() => {
                let Some(msg) = msg else { break };

                match serde_json::from_slice::<Heartbeat>(&msg.payload) {
                    Ok(hb) if !hb.agent_id.is_empty() => {
                        let now = Instant::now();
                        let is_new = !agents.contains_key(&hb.agent_id);
                        let record = agents
                            .entry(hb.agent_id.clone())
                            .or_insert_with(|| AgentRecord {
                                agent_type: hb.agent_type.clone(),
                                last_seen: now,
                                uptime_secs: hb.uptime_secs,
                            });
                        record.last_seen = now;
                        record.uptime_secs = hb.uptime_secs;
                        if !hb.agent_type.is_empty() {
                            record.agent_type = hb.agent_type.clone();
                        }
                        if is_new {
                            info!(
                                agent_id = %hb.agent_id,
                                agent_type = %hb.agent_type,
                                "new agent registered"
                            );
                        }
                    }
                    Ok(_) => warn!("received heartbeat with empty agent_id — discarded"),
                    Err(e) => warn!(%e, "failed to parse heartbeat JSON"),
                }
            }

            // ── Periodic status sweep ─────────────────────────────────────
            _ = ticker.tick() => {
                let ts = SystemTime::now()
                    .duration_since(UNIX_EPOCH)
                    .unwrap_or_default()
                    .as_secs();

                let now = Instant::now();
                let mut healthy = 0u32;
                let mut stale = 0u32;
                let mut offline = 0u32;

                for (agent_id, record) in &agents {
                    let elapsed = now.duration_since(record.last_seen);
                    let status = classify_agent(elapsed);

                    match status {
                        "healthy" => healthy += 1,
                        "stale" => {
                            stale += 1;
                            warn!(
                                agent_id,
                                agent_type = %record.agent_type,
                                elapsed_secs = elapsed.as_secs(),
                                "agent heartbeat stale"
                            );
                        }
                        _ => {
                            offline += 1;
                            warn!(
                                agent_id,
                                agent_type = %record.agent_type,
                                elapsed_secs = elapsed.as_secs(),
                                "agent appears offline"
                            );
                        }
                    }

                    let event = AgentStatus {
                        tenant_id: cfg.tenant_id.clone(),
                        agent_id: agent_id.clone(),
                        agent_type: record.agent_type.clone(),
                        status: status.to_string(),
                        last_seen_secs_ago: elapsed.as_secs(),
                        uptime_secs: record.uptime_secs,
                        timestamp: ts,
                    };

                    match serde_json::to_vec(&event) {
                        Ok(payload) => {
                            if let Err(e) = client
                                .publish(status_subject.clone(), payload.into())
                                .await
                            {
                                error!(%e, agent_id, "NATS publish agent status failed");
                            }
                        }
                        Err(e) => error!(%e, agent_id, "serialise AgentStatus failed"),
                    }
                }

                if agents.len() != last_summary_count || healthy + stale + offline > 0 {
                    info!(
                        total = agents.len(),
                        healthy,
                        stale,
                        offline,
                        "agent health summary"
                    );
                    last_summary_count = agents.len();
                }

                if agents.is_empty() {
                    info!("watchdog: no agents yet — waiting for heartbeats");
                }
            }

            // ── Self heartbeat ────────────────────────────────────────────
            _ = self_hb_ticker.tick() => {
                publish_self_heartbeat(&client, &heartbeat_subject, &cfg).await;
            }

            // ── Graceful shutdown ─────────────────────────────────────────
            _ = &mut shutdown => {
                info!("Watchdog received shutdown signal");
                break;
            }
        }
    }

    info!("Watchdog orchestrator stopped");
    Ok(())
}

// ── Helpers ───────────────────────────────────────────────────────────────────

fn classify_agent(elapsed: Duration) -> &'static str {
    if elapsed < STALE_THRESHOLD {
        "healthy"
    } else if elapsed < OFFLINE_THRESHOLD {
        "stale"
    } else {
        "offline"
    }
}

async fn publish_self_heartbeat(
    client: &async_nats::Client,
    subject: &str,
    cfg: &Config,
) {
    let payload = match serde_json::to_vec(&serde_json::json!({
        "agent_id": cfg.agent_id,
        "agent_type": "watchdog",
        "uptime_secs": 0u64,
    })) {
        Ok(p) => p,
        Err(e) => {
            error!(%e, "failed to serialise watchdog self-heartbeat");
            return;
        }
    };
    if let Err(e) = client.publish(subject.to_string(), payload.into()).await {
        warn!(%e, "failed to publish watchdog self-heartbeat");
    }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn classify_agent_healthy() {
        assert_eq!(classify_agent(Duration::from_secs(0)), "healthy");
        assert_eq!(classify_agent(Duration::from_secs(29)), "healthy");
    }

    #[test]
    fn classify_agent_stale() {
        assert_eq!(classify_agent(Duration::from_secs(30)), "stale");
        assert_eq!(classify_agent(Duration::from_secs(119)), "stale");
    }

    #[test]
    fn classify_agent_offline() {
        assert_eq!(classify_agent(Duration::from_secs(120)), "offline");
        assert_eq!(classify_agent(Duration::from_secs(600)), "offline");
    }

    #[test]
    fn agent_status_serializes() {
        let s = AgentStatus {
            tenant_id: "t1".into(),
            agent_id: "coresec-1".into(),
            agent_type: "coresec".into(),
            status: "healthy".into(),
            last_seen_secs_ago: 5,
            uptime_secs: 3600,
            timestamp: 1_700_000_000,
        };
        let json = serde_json::to_string(&s).unwrap();
        assert!(json.contains("coresec-1"));
        assert!(json.contains("\"status\":\"healthy\""));
    }

    #[test]
    fn heartbeat_deserializes() {
        let json = r#"{"agent_id":"ng-1","agent_type":"netguard","uptime_secs":120}"#;
        let hb: super::Heartbeat = serde_json::from_str(json).unwrap();
        assert_eq!(hb.agent_id, "ng-1");
        assert_eq!(hb.agent_type, "netguard");
        assert_eq!(hb.uptime_secs, 120);
    }

    #[test]
    fn heartbeat_defaults_for_missing_fields() {
        let json = r#"{"agent_id":"min-1"}"#;
        let hb: super::Heartbeat = serde_json::from_str(json).unwrap();
        assert_eq!(hb.agent_type, "");
        assert_eq!(hb.uptime_secs, 0);
    }
}
