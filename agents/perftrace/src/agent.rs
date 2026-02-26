use crate::config::Config;
use crate::metrics::HostMetric;
use anyhow::{Context, Result};
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};
use std::time::{SystemTime, UNIX_EPOCH};
use sysinfo::{Disks, Networks, System};
use tokio::time::{interval, Duration};
use tracing::{error, info, warn};

pub async fn run(cfg: Config) -> Result<()> {
    let client = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;

    info!(tenant_id = %cfg.tenant_id, "PerfTrace metrics collector starting");

    let mut sys = System::new_all();
    let mut disks = Disks::new_with_refreshed_list();
    let mut networks = Networks::new_with_refreshed_list();

    let subject = HostMetric::nats_subject(&cfg.tenant_id);
    let poll_dur = Duration::from_secs(cfg.poll_interval_secs);
    let mut ticker = interval(poll_dur);
    let mut shutdown = Box::pin(tokio::signal::ctrl_c());
    let mut cycle: u64 = 0;

    loop {
        tokio::select! {
            _ = ticker.tick() => {
                sys.refresh_all();
                disks.refresh(true);
                networks.refresh(true);

                let metric = collect_metric(&cfg, &sys, &disks, &networks, cycle);
                cycle += 1;

                if let Ok(payload) = serde_json::to_vec(&metric) {
                    if let Err(e) = client
                        .publish(subject.clone(), payload.into())
                        .await
                    {
                        error!(%e, "NATS publish host metric failed");
                    }
                }

                // Severity thresholds — publish warning events
                if metric.cpu_usage_pct > 90.0 {
                    warn!(cpu = metric.cpu_usage_pct, "CPU usage critical");
                    publish_threshold_alert(&client, &cfg, "cpu_high", metric.cpu_usage_pct as f64).await;
                }
                if metric.mem_usage_pct > 90.0 {
                    warn!(mem = metric.mem_usage_pct, "Memory usage critical");
                    publish_threshold_alert(&client, &cfg, "mem_high", metric.mem_usage_pct as f64).await;
                }
                if metric.disk_total_bytes > 0 {
                    let disk_pct = (metric.disk_used_bytes as f64 / metric.disk_total_bytes as f64) * 100.0;
                    if disk_pct > 90.0 {
                        warn!(disk_pct, "Disk usage critical");
                        publish_threshold_alert(&client, &cfg, "disk_high", disk_pct).await;
                    }
                }

                if cycle % 12 == 0 {
                    info!(
                        cycle,
                        cpu = format!("{:.1}%", metric.cpu_usage_pct),
                        mem = format!("{:.1}%", metric.mem_usage_pct),
                        procs = metric.process_count,
                        "metrics summary"
                    );
                }
            }

            _ = &mut shutdown => {
                info!("PerfTrace shutting down");
                break;
            }
        }
    }

    Ok(())
}

fn collect_metric(
    cfg: &Config,
    sys: &System,
    disks: &Disks,
    networks: &Networks,
    cycle: u64,
) -> HostMetric {
    let ts = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();

    let event_id = {
        let mut h = DefaultHasher::new();
        cfg.tenant_id.hash(&mut h);
        ts.hash(&mut h);
        cycle.hash(&mut h);
        format!("{:016x}", h.finish())
    };

    // CPU
    let cpu_count = sys.cpus().len();
    let cpu_usage_pct = sys.global_cpu_usage();
    let load_avg = System::load_average();

    // Memory
    let mem_total = sys.total_memory();
    let mem_used = sys.used_memory();
    let mem_available = sys.available_memory();
    let mem_pct = if mem_total > 0 {
        (mem_used as f32 / mem_total as f32) * 100.0
    } else {
        0.0
    };

    // Swap
    let swap_total = sys.total_swap();
    let swap_used = sys.used_swap();

    // Disks (aggregate)
    let mut disk_total: u64 = 0;
    let mut disk_used: u64 = 0;
    for disk in disks.iter() {
        disk_total += disk.total_space();
        disk_used += disk.total_space() - disk.available_space();
    }

    // Network (aggregate)
    let mut net_rx: u64 = 0;
    let mut net_tx: u64 = 0;
    let mut net_rx_pkt: u64 = 0;
    let mut net_tx_pkt: u64 = 0;
    for (_name, data) in networks.iter() {
        net_rx += data.received();
        net_tx += data.transmitted();
        net_rx_pkt += data.packets_received();
        net_tx_pkt += data.packets_transmitted();
    }

    // Processes
    let process_count = sys.processes().len();

    // Uptime
    let uptime_secs = System::uptime();

    let raw = format!(
        "{}:{}:{}:{}:{}:{}",
        cfg.tenant_id, ts, cpu_usage_pct, mem_used, disk_used, process_count
    );
    let blake3_hash = blake3::hash(raw.as_bytes()).to_hex().to_string();

    HostMetric {
        tenant_id: cfg.tenant_id.clone(),
        agent_id: cfg.agent_id.clone(),
        event_id,
        timestamp: ts.to_string(),
        class_uid: 4004,
        severity_id: 1,
        cpu_count,
        cpu_usage_pct,
        load_avg_1: load_avg.one,
        load_avg_5: load_avg.five,
        load_avg_15: load_avg.fifteen,
        mem_total_bytes: mem_total,
        mem_used_bytes: mem_used,
        mem_available_bytes: mem_available,
        mem_usage_pct: mem_pct,
        swap_total_bytes: swap_total,
        swap_used_bytes: swap_used,
        disk_total_bytes: disk_total,
        disk_used_bytes: disk_used,
        disk_read_bytes: 0,  // cumulative I/O counters added in next release
        disk_write_bytes: 0,
        net_rx_bytes: net_rx,
        net_tx_bytes: net_tx,
        net_rx_packets: net_rx_pkt,
        net_tx_packets: net_tx_pkt,
        process_count,
        uptime_secs,
        blake3_hash,
    }
}

async fn publish_threshold_alert(
    client: &async_nats::Client,
    cfg: &Config,
    kind: &str,
    value: f64,
) {
    let alert = serde_json::json!({
        "tenant_id": &cfg.tenant_id,
        "agent_id": &cfg.agent_id,
        "type": "threshold_breach",
        "kind": kind,
        "value": value,
        "severity": "HIGH",
        "timestamp": SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs(),
    });
    let subject = format!("kubric.{}.detection.threshold.v1", cfg.tenant_id);
    if let Ok(payload) = serde_json::to_vec(&alert) {
        let _ = client.publish(subject, payload.into()).await;
    }
}
