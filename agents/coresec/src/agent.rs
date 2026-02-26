use crate::config::Config;
use crate::event::ProcessEvent;
use anyhow::{Context, Result};
use tracing::{error, info};

pub async fn run(cfg: Config) -> Result<()> {
    let client = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;

    info!("NATS connected — beginning process event loop");

    // Phase 1 stub: publishes a single synthetic event then parks.
    // Phase 3 will replace this with real eBPF process tracking via Aya.
    let event = build_stub_event(&cfg);
    let subject = ProcessEvent::nats_subject(&cfg.tenant_id);
    let payload = serde_json::to_vec(&event).context("serialise event")?;

    client
        .publish(subject.clone(), payload.into())
        .await
        .context("publish to NATS")?;

    info!(subject, "stub event published");

    // Park until SIGINT / SIGTERM
    tokio::signal::ctrl_c().await.ok();
    info!("CoreSec shutting down");
    Ok(())
}

fn build_stub_event(cfg: &Config) -> ProcessEvent {
    let raw = format!(
        "stub:{}:{}",
        cfg.tenant_id,
        chrono_now()
    );
    let hash = format!("{:x}", blake3::hash(raw.as_bytes()));

    ProcessEvent {
        tenant_id: cfg.tenant_id.clone(),
        agent_id: cfg.agent_id.clone(),
        event_id: uuid_v4(),
        timestamp: chrono_now(),
        class_uid: 4007,
        severity_id: 1,
        activity_id: 2, // EXEC
        pid: std::process::id(),
        ppid: 0,
        executable: "/usr/bin/kubric-coresec".to_string(),
        cmdline: "coresec --stub".to_string(),
        user: "root".to_string(),
        blake3_hash: hash,
    }
}

fn chrono_now() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    let secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();
    // Minimal RFC3339-ish without chrono dep — Phase 3 will add chrono properly
    format!("{secs}")
}

fn uuid_v4() -> String {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    use std::time::{SystemTime, UNIX_EPOCH};
    let mut h = DefaultHasher::new();
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos()
        .hash(&mut h);
    std::process::id().hash(&mut h);
    format!("{:016x}-stub", h.finish())
}
