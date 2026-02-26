use crate::config::Config;
use crate::detection::DetectionEngine;
use crate::event::ProcessEvent;
use anyhow::{Context, Result};
use std::collections::{hash_map::DefaultHasher, HashSet};
use std::hash::{Hash, Hasher};
use std::time::{SystemTime, UNIX_EPOCH};
use sysinfo::{ProcessRefreshKind, ProcessesToUpdate, System};
use tokio::time::{interval, Duration};
use tracing::{error, info, warn};

/// How often to poll the OS process table.
const POLL_INTERVAL: Duration = Duration::from_secs(5);

pub async fn run(cfg: Config) -> Result<()> {
    let client = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;

    info!(tenant_id = %cfg.tenant_id, "CoreSec process monitor starting");

    // Load detection rules.  Missing directories yield an empty engine
    // (safe for CI and restricted environments).
    let sigma_dir = std::env::var("KUBRIC_SIGMA_DIR")
        .unwrap_or_else(|_| "vendor/sigma-rules/rules".to_string());
    let yara_dir = std::env::var("KUBRIC_YARA_DIR")
        .unwrap_or_else(|_| "vendor/yara-rules".to_string());

    let engine = match DetectionEngine::new(&sigma_dir, &yara_dir) {
        Ok(e) => {
            info!(sigma_dir = %sigma_dir, yara_dir = %yara_dir, "DetectionEngine loaded");
            e
        }
        Err(err) => {
            warn!(%err, "DetectionEngine rule load failed — using empty engine");
            DetectionEngine::empty()
        }
    };

    // Initialise sysinfo and capture the baseline process set so we only
    // fire events for processes that appear *after* the agent starts.
    let mut sys = System::new();
    sys.refresh_processes_specifics(ProcessesToUpdate::All, true, ProcessRefreshKind::everything());

    let mut seen_pids: HashSet<u32> = sys
        .processes()
        .keys()
        .map(|p| p.as_u32())
        .collect();

    info!(baseline = seen_pids.len(), "process baseline captured");

    let process_subject = ProcessEvent::nats_subject(&cfg.tenant_id);
    let sigma_subject   = format!("kubric.{}.detection.sigma.v1",  cfg.tenant_id);
    let yara_subject    = format!("kubric.{}.detection.yara.v1",   cfg.tenant_id);

    let mut ticker   = interval(POLL_INTERVAL);
    let mut shutdown = Box::pin(tokio::signal::ctrl_c());

    loop {
        tokio::select! {
            _ = ticker.tick() => {
                sys.refresh_processes_specifics(ProcessesToUpdate::All, true, ProcessRefreshKind::everything());

                // Collect PIDs not yet seen
                let new_pids: Vec<u32> = sys
                    .processes()
                    .keys()
                    .map(|p| p.as_u32())
                    .filter(|p| seen_pids.insert(*p))
                    .collect();

                // Prune exited PIDs to bound memory usage
                seen_pids.retain(|p| {
                    sys.processes()
                        .contains_key(&sysinfo::Pid::from(*p as usize))
                });

                for pid in new_pids {
                    let Some(proc) = sys
                        .processes()
                        .get(&sysinfo::Pid::from(pid as usize))
                    else {
                        continue;
                    };

                    let event = build_process_event(&cfg, proc, pid);

                    // Publish raw process event
                    if let Ok(payload) = serde_json::to_vec(&event) {
                        if let Err(e) = client
                            .publish(process_subject.clone(), payload.into())
                            .await
                        {
                            error!(%e, "NATS publish process event failed");
                        }
                    }

                    // Run Sigma + YARA detection
                    let (sigma_hits, yara_hits) = engine.detect(&event);

                    for hit in &sigma_hits {
                        let alert = serde_json::json!({
                            "tenant_id":  &cfg.tenant_id,
                            "agent_id":   &cfg.agent_id,
                            "event_id":   &event.event_id,
                            "rule_id":    &hit.rule_id,
                            "rule_title": &hit.title,
                            "severity":   &hit.level,
                            "tags":       &hit.tags,
                            "pid":        pid,
                            "executable": &event.executable,
                            "cmdline":    &event.cmdline,
                            "timestamp":  &event.timestamp,
                        });
                        if let Ok(payload) = serde_json::to_vec(&alert) {
                            if let Err(e) = client
                                .publish(sigma_subject.clone(), payload.into())
                                .await
                            {
                                error!(%e, "NATS publish sigma alert failed");
                            }
                        }
                        info!(
                            rule  = %hit.rule_id,
                            level = %hit.level,
                            pid,
                            exe   = %event.executable,
                            "Sigma match"
                        );
                    }

                    for hit in &yara_hits {
                        let alert = serde_json::json!({
                            "tenant_id":  &cfg.tenant_id,
                            "agent_id":   &cfg.agent_id,
                            "event_id":   &event.event_id,
                            "rule_id":    &hit.rule,
                            "rule_title": &hit.rule,
                            "namespace":  &hit.namespace,
                            "severity":   "medium",
                            "pid":        pid,
                            "executable": &event.executable,
                            "cmdline":    &event.cmdline,
                            "timestamp":  &event.timestamp,
                        });
                        if let Ok(payload) = serde_json::to_vec(&alert) {
                            if let Err(e) = client
                                .publish(yara_subject.clone(), payload.into())
                                .await
                            {
                                error!(%e, "NATS publish yara alert failed");
                            }
                        }
                        info!(
                            rule = %hit.rule,
                            pid,
                            exe  = %event.executable,
                            "YARA match"
                        );
                    }
                }
            }

            _ = &mut shutdown => {
                info!("CoreSec shutting down");
                break;
            }
        }
    }

    Ok(())
}

fn now_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

fn build_process_event(cfg: &Config, proc: &sysinfo::Process, pid: u32) -> ProcessEvent {
    let exe = proc
        .exe()
        .map(|p| p.to_string_lossy().into_owned())
        .unwrap_or_else(|| proc.name().to_string_lossy().into_owned());

    let cmdline = proc
        .cmd()
        .iter()
        .map(|s| s.to_string_lossy().into_owned())
        .collect::<Vec<_>>()
        .join(" ");

    let user = proc
        .user_id()
        .map(|uid| uid.to_string())
        .unwrap_or_else(|| "unknown".to_string());

    let ppid = proc.parent().map(|p| p.as_u32()).unwrap_or(0);

    let ts = now_secs();

    let event_id = {
        let mut h = DefaultHasher::new();
        cfg.tenant_id.hash(&mut h);
        pid.hash(&mut h);
        ts.hash(&mut h);
        format!("{:016x}", h.finish())
    };

    let raw = format!("{}:{}:{}:{}", cfg.tenant_id, pid, exe, cmdline);
    let blake3_hash = blake3::hash(raw.as_bytes()).to_hex().to_string();

    ProcessEvent {
        tenant_id:   cfg.tenant_id.clone(),
        agent_id:    cfg.agent_id.clone(),
        event_id,
        timestamp:   ts.to_string(),
        class_uid:   4007,
        severity_id: 1,
        activity_id: 2, // EXEC
        pid,
        ppid,
        executable:  exe,
        cmdline,
        user,
        blake3_hash,
    }
}
