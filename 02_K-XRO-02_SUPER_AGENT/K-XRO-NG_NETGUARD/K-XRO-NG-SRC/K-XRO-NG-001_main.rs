//! NetGuard Agent — Layer 2-4 network capture, flow analysis, IDS, and TI enforcement.
//! Compiled with: cargo build --release --features default
use std::collections::HashMap;
use std::net::{IpAddr, Ipv4Addr};
use std::sync::Arc;
use std::time::{Duration, Instant};

use anyhow::{Context, Result};
use async_nats::Client as NatsClient;
use chrono::Utc;
use serde::{Deserialize, Serialize};
use tokio::signal;
use tokio::sync::{Mutex, RwLock};
use tokio::time::interval;
use tracing::{debug, error, info, warn};

// ── Environment variable names ────────────────────────────────────────────────

const ENV_NATS_URL: &str = "NATS_URL";
const ENV_INTERFACE: &str = "INTERFACE";
const ENV_TENANT_ID: &str = "TENANT_ID";
const ENV_AGENT_ID: &str = "AGENT_ID";
const ENV_PCAP_FILTER: &str = "PCAP_FILTER";
const ENV_FLOW_TIMEOUT_SECS: &str = "FLOW_TIMEOUT_SECS";
const ENV_TI_FEEDS_URL: &str = "TI_FEEDS_URL";
const ENV_PROMETHEUS_PORT: &str = "PROMETHEUS_PORT";
const ENV_LOG_LEVEL: &str = "LOG_LEVEL";

// ── Core data structures ──────────────────────────────────────────────────────

/// Five-tuple identifying a network flow.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct FlowKey {
    pub src_ip: IpAddr,
    pub dst_ip: IpAddr,
    pub src_port: u16,
    pub dst_port: u16,
    pub protocol: u8,
}

/// Accumulated statistics for a single network flow.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FlowRecord {
    pub key: FlowKey,
    pub start_time: i64,      // Unix timestamp seconds
    pub last_seen: i64,
    pub packets: u64,
    pub bytes: u64,
    pub flags: u8,            // OR of TCP flags observed
    pub tls_sni: Option<String>,
    pub completed: bool,
}

/// An alert emitted when a flow matches a TI hit or IDS rule.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Alert {
    pub ts: i64,
    pub tenant_id: String,
    pub agent_id: String,
    pub alert_type: String,   // "ti_match" | "ids_rule" | "tls_anomaly"
    pub severity: u8,         // 1-10
    pub src_ip: String,
    pub dst_ip: String,
    pub src_port: u16,
    pub dst_port: u16,
    pub protocol: u8,
    pub matched_indicator: String,
    pub rule_name: String,
    pub bytes: u64,
    pub packets: u64,
}

/// A compiled IDS rule (Snort/Suricata-compatible subset).
#[derive(Debug, Clone)]
pub struct IdsRule {
    pub name: String,
    pub severity: u8,
    pub src_port: Option<u16>,
    pub dst_port: Option<u16>,
    pub pattern: Option<regex::Regex>,
}

/// Threat Intelligence feed — set of known-bad CIDRs and exact IPs.
#[derive(Debug, Default)]
pub struct TiFeed {
    pub bad_ips: std::collections::HashSet<IpAddr>,
    pub bad_cidrs: Vec<(IpAddr, u32)>, // (network_addr, prefix_len)
    pub last_updated: Option<Instant>,
}

impl TiFeed {
    /// Returns true when ip matches any entry in the feed.
    pub fn contains(&self, ip: &IpAddr) -> bool {
        if self.bad_ips.contains(ip) {
            return true;
        }
        // Simple CIDR matching for IPv4 only in this implementation.
        if let IpAddr::V4(v4) = ip {
            let ip_u32 = u32::from(*v4);
            for (net, prefix) in &self.bad_cidrs {
                if let IpAddr::V4(net_v4) = net {
                    let mask = if *prefix >= 32 {
                        u32::MAX
                    } else {
                        !((1u32 << (32 - prefix)) - 1)
                    };
                    if ip_u32 & mask == u32::from(*net_v4) & mask {
                        return true;
                    }
                }
            }
        }
        false
    }
}

/// Prometheus-compatible metrics counters.
#[derive(Debug, Default)]
pub struct Metrics {
    pub packets_captured: u64,
    pub bytes_captured: u64,
    pub flows_active: u64,
    pub flows_completed: u64,
    pub alerts_emitted: u64,
    pub ti_hits: u64,
    pub ids_hits: u64,
    pub pcap_drops: u64,
}

// ── FlowAnalyzer ─────────────────────────────────────────────────────────────

pub struct FlowAnalyzer {
    flows: Mutex<HashMap<FlowKey, FlowRecord>>,
    timeout_secs: u64,
}

impl FlowAnalyzer {
    pub fn new(timeout_secs: u64) -> Self {
        Self {
            flows: Mutex::new(HashMap::new()),
            timeout_secs,
        }
    }

    /// Process a captured packet, updating or creating the corresponding flow.
    /// Returns Some(FlowRecord) if the flow was just completed (FIN/RST).
    pub async fn process_packet(
        &self,
        key: FlowKey,
        payload_len: u64,
        tcp_flags: u8,
        tls_sni: Option<String>,
    ) -> Option<FlowRecord> {
        let now = Utc::now().timestamp();
        let mut flows = self.flows.lock().await;

        let record = flows.entry(key.clone()).or_insert_with(|| FlowRecord {
            key: key.clone(),
            start_time: now,
            last_seen: now,
            packets: 0,
            bytes: 0,
            flags: 0,
            tls_sni: None,
            completed: false,
        });

        record.packets += 1;
        record.bytes += payload_len;
        record.last_seen = now;
        record.flags |= tcp_flags;
        if record.tls_sni.is_none() && tls_sni.is_some() {
            record.tls_sni = tls_sni;
        }

        // FIN (0x01) or RST (0x04) — mark flow complete.
        if tcp_flags & 0x05 != 0 {
            record.completed = true;
            let completed = record.clone();
            flows.remove(&key);
            return Some(completed);
        }

        None
    }

    /// Expire flows that have been idle longer than timeout_secs.
    /// Returns all expired flow records.
    pub async fn flush_expired(&self) -> Vec<FlowRecord> {
        let now = Utc::now().timestamp();
        let threshold = now - self.timeout_secs as i64;
        let mut flows = self.flows.lock().await;
        let mut expired = Vec::new();
        flows.retain(|_, v| {
            if v.last_seen < threshold {
                expired.push(v.clone());
                false
            } else {
                true
            }
        });
        expired
    }

    pub async fn active_count(&self) -> u64 {
        self.flows.lock().await.len() as u64
    }
}

// ── AlertPublisher ────────────────────────────────────────────────────────────

pub struct AlertPublisher {
    nats: NatsClient,
    tenant_id: String,
    agent_id: String,
}

impl AlertPublisher {
    pub fn new(nats: NatsClient, tenant_id: String, agent_id: String) -> Self {
        Self { nats, tenant_id, agent_id }
    }

    pub async fn publish(&self, mut alert: Alert) -> Result<()> {
        alert.tenant_id = self.tenant_id.clone();
        alert.agent_id = self.agent_id.clone();
        let subject = format!("kubric.alerts.{}.netguard", self.tenant_id);
        let payload = serde_json::to_vec(&alert)?;
        self.nats
            .publish(subject, payload.into())
            .await
            .context("nats publish alert")?;
        Ok(())
    }

    /// Build an alert from a completed flow + match reason.
    pub fn build_alert(
        flow: &FlowRecord,
        alert_type: &str,
        severity: u8,
        matched_indicator: String,
        rule_name: String,
    ) -> Alert {
        Alert {
            ts: Utc::now().timestamp(),
            tenant_id: String::new(),
            agent_id: String::new(),
            alert_type: alert_type.to_string(),
            severity,
            src_ip: flow.key.src_ip.to_string(),
            dst_ip: flow.key.dst_ip.to_string(),
            src_port: flow.key.src_port,
            dst_port: flow.key.dst_port,
            protocol: flow.key.protocol,
            matched_indicator,
            rule_name,
            bytes: flow.bytes,
            packets: flow.packets,
        }
    }
}

// ── TI Feed loader ────────────────────────────────────────────────────────────

async fn load_ti_feed(feeds_url: &str) -> Result<TiFeed> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(30))
        .build()?;

    let body = client
        .get(feeds_url)
        .send()
        .await
        .context("fetch TI feed")?
        .text()
        .await?;

    let mut feed = TiFeed::default();
    for line in body.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        if line.contains('/') {
            // CIDR notation
            let parts: Vec<&str> = line.splitn(2, '/').collect();
            if parts.len() == 2 {
                if let (Ok(ip), Ok(prefix)) = (parts[0].parse::<IpAddr>(), parts[1].parse::<u32>()) {
                    feed.bad_cidrs.push((ip, prefix));
                }
            }
        } else if let Ok(ip) = line.parse::<IpAddr>() {
            feed.bad_ips.insert(ip);
        }
    }

    feed.last_updated = Some(Instant::now());
    info!(
        indicators = feed.bad_ips.len() + feed.bad_cidrs.len(),
        "TI feed loaded"
    );
    Ok(feed)
}

// ── Prometheus metrics HTTP handler ──────────────────────────────────────────

async fn metrics_server(port: u16, metrics: Arc<RwLock<Metrics>>) {
    use std::convert::Infallible;

    let make_svc = {
        let metrics = metrics.clone();
        move || {
            let metrics = metrics.clone();
            async move {
                Ok::<_, Infallible>(hyper_service(metrics).await)
            }
        }
    };
    let _ = make_svc; // avoid unused warning — real impl below

    // Minimal HTTP/1.1 server without a full hyper dependency.
    let addr = format!("0.0.0.0:{port}");
    let listener = match tokio::net::TcpListener::bind(&addr).await {
        Ok(l) => l,
        Err(e) => {
            error!("metrics listener bind failed on {}: {}", addr, e);
            return;
        }
    };
    info!("Prometheus metrics listening on http://{}/metrics", addr);

    loop {
        if let Ok((mut stream, _)) = listener.accept().await {
            let metrics = metrics.clone();
            tokio::spawn(async move {
                use tokio::io::AsyncWriteExt;
                let m = metrics.read().await;
                let body = format!(
                    "# HELP kubric_ng_packets_total Total packets captured\n\
                     kubric_ng_packets_total {}\n\
                     # HELP kubric_ng_bytes_total Total bytes captured\n\
                     kubric_ng_bytes_total {}\n\
                     # HELP kubric_ng_flows_active Currently active flows\n\
                     kubric_ng_flows_active {}\n\
                     # HELP kubric_ng_flows_completed_total Completed flows\n\
                     kubric_ng_flows_completed_total {}\n\
                     # HELP kubric_ng_alerts_total Alerts emitted\n\
                     kubric_ng_alerts_total {}\n\
                     # HELP kubric_ng_ti_hits_total TI indicator matches\n\
                     kubric_ng_ti_hits_total {}\n\
                     # HELP kubric_ng_ids_hits_total IDS rule matches\n\
                     kubric_ng_ids_hits_total {}\n\
                     # HELP kubric_ng_pcap_drops_total Packets dropped by pcap\n\
                     kubric_ng_pcap_drops_total {}\n",
                    m.packets_captured,
                    m.bytes_captured,
                    m.flows_active,
                    m.flows_completed,
                    m.alerts_emitted,
                    m.ti_hits,
                    m.ids_hits,
                    m.pcap_drops,
                );
                let response = format!(
                    "HTTP/1.1 200 OK\r\nContent-Type: text/plain; version=0.0.4\r\nContent-Length: {}\r\n\r\n{}",
                    body.len(),
                    body
                );
                let _ = stream.write_all(response.as_bytes()).await;
            });
        }
    }
}

async fn hyper_service(_metrics: Arc<RwLock<Metrics>>) -> () {}

// ── Packet capture loop ───────────────────────────────────────────────────────

/// Simulate AF_PACKET or pcap capture. In production, replace with pcap::Capture.
async fn capture_loop(
    interface: String,
    filter: String,
    analyzer: Arc<FlowAnalyzer>,
    publisher: Arc<AlertPublisher>,
    ti_feed: Arc<RwLock<TiFeed>>,
    ids_rules: Arc<RwLock<Vec<IdsRule>>>,
    metrics: Arc<RwLock<Metrics>>,
    mut shutdown: tokio::sync::watch::Receiver<bool>,
) {
    info!(interface = %interface, filter = %filter, "Starting packet capture");

    // In production: pcap::Capture::from_device(&interface)
    //   .unwrap().promisc(true).snaplen(65535).open().unwrap()
    // Here we generate synthetic test traffic to keep the loop alive.
    let mut synthetic_ticker = interval(Duration::from_millis(100));
    let mut pkt_counter: u64 = 0;

    loop {
        tokio::select! {
            _ = shutdown.changed() => {
                if *shutdown.borrow() {
                    info!("Capture loop shutting down");
                    return;
                }
            }
            _ = synthetic_ticker.tick() => {
                // Synthetic packet (replace with real pcap data in production)
                pkt_counter += 1;
                let key = FlowKey {
                    src_ip: IpAddr::V4(Ipv4Addr::new(10, 0, 0, (pkt_counter % 254 + 1) as u8)),
                    dst_ip: IpAddr::V4(Ipv4Addr::new(8, 8, 8, 8)),
                    src_port: (30000 + pkt_counter % 35000) as u16,
                    dst_port: 443,
                    protocol: 6,
                };
                let flags: u8 = if pkt_counter % 50 == 0 { 0x01 } else { 0x00 };
                let sni = if pkt_counter % 7 == 0 {
                    Some("example.com".to_string())
                } else {
                    None
                };

                {
                    let mut m = metrics.write().await;
                    m.packets_captured += 1;
                    m.bytes_captured += 1460;
                }

                if let Some(completed_flow) = analyzer.process_packet(key.clone(), 1460, flags, sni).await {
                    handle_completed_flow(
                        completed_flow,
                        publisher.clone(),
                        ti_feed.clone(),
                        ids_rules.clone(),
                        metrics.clone(),
                    ).await;
                }
            }
        }
    }
}

/// Check a completed flow against TI and IDS, emitting alerts on matches.
async fn handle_completed_flow(
    flow: FlowRecord,
    publisher: Arc<AlertPublisher>,
    ti_feed: Arc<RwLock<TiFeed>>,
    ids_rules: Arc<RwLock<Vec<IdsRule>>>,
    metrics: Arc<RwLock<Metrics>>,
) {
    let feed = ti_feed.read().await;

    // TI check on src and dst
    for (ip, role) in [(&flow.key.src_ip, "src"), (&flow.key.dst_ip, "dst")] {
        if feed.contains(ip) {
            let alert = AlertPublisher::build_alert(
                &flow,
                "ti_match",
                8,
                ip.to_string(),
                format!("ti_{}_ip_match", role),
            );
            if let Err(e) = publisher.publish(alert).await {
                warn!("Failed to publish TI alert: {}", e);
            } else {
                let mut m = metrics.write().await;
                m.alerts_emitted += 1;
                m.ti_hits += 1;
            }
        }
    }

    // IDS rules check
    let rules = ids_rules.read().await;
    for rule in rules.iter() {
        let port_match = rule.dst_port.map_or(true, |p| p == flow.key.dst_port)
            || rule.src_port.map_or(true, |p| p == flow.key.src_port);
        if port_match {
            let alert = AlertPublisher::build_alert(
                &flow,
                "ids_rule",
                rule.severity,
                format!("port:{}", flow.key.dst_port),
                rule.name.clone(),
            );
            if let Err(e) = publisher.publish(alert).await {
                warn!("Failed to publish IDS alert: {}", e);
            } else {
                let mut m = metrics.write().await;
                m.alerts_emitted += 1;
                m.ids_hits += 1;
            }
            break;
        }
    }
}

// ── Main entry point ──────────────────────────────────────────────────────────

#[tokio::main]
async fn main() -> Result<()> {
    // ── Configuration from environment ───────────────────────────────────────
    let log_level = std::env::var(ENV_LOG_LEVEL).unwrap_or_else(|_| "info".to_string());
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_new(&log_level)
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .init();

    let nats_url = std::env::var(ENV_NATS_URL).unwrap_or_else(|_| "nats://127.0.0.1:4222".to_string());
    let interface = std::env::var(ENV_INTERFACE).unwrap_or_else(|_| "eth0".to_string());
    let tenant_id = std::env::var(ENV_TENANT_ID).unwrap_or_else(|_| "default".to_string());
    let agent_id = std::env::var(ENV_AGENT_ID).unwrap_or_else(|_| "netguard-0".to_string());
    let pcap_filter = std::env::var(ENV_PCAP_FILTER).unwrap_or_default();
    let flow_timeout: u64 = std::env::var(ENV_FLOW_TIMEOUT_SECS)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(300);
    let ti_feeds_url = std::env::var(ENV_TI_FEEDS_URL).unwrap_or_default();
    let prometheus_port: u16 = std::env::var(ENV_PROMETHEUS_PORT)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(9091);

    info!(
        tenant_id = %tenant_id,
        agent_id  = %agent_id,
        interface = %interface,
        "NetGuard agent starting"
    );

    // ── NATS client ───────────────────────────────────────────────────────────
    let nats = async_nats::connect(&nats_url)
        .await
        .context("connect to NATS")?;
    info!("Connected to NATS at {}", nats_url);

    // ── Shared state ──────────────────────────────────────────────────────────
    let analyzer = Arc::new(FlowAnalyzer::new(flow_timeout));
    let publisher = Arc::new(AlertPublisher::new(nats, tenant_id.clone(), agent_id.clone()));
    let metrics = Arc::new(RwLock::new(Metrics::default()));

    // Initial TI feed load
    let ti_feed: Arc<RwLock<TiFeed>> = if !ti_feeds_url.is_empty() {
        match load_ti_feed(&ti_feeds_url).await {
            Ok(feed) => Arc::new(RwLock::new(feed)),
            Err(e) => {
                warn!("Initial TI feed load failed: {}; starting with empty feed", e);
                Arc::new(RwLock::new(TiFeed::default()))
            }
        }
    } else {
        info!("TI_FEEDS_URL not set; TI matching disabled");
        Arc::new(RwLock::new(TiFeed::default()))
    };

    // IDS rules (empty at startup; populated by reload watcher)
    let ids_rules: Arc<RwLock<Vec<IdsRule>>> = Arc::new(RwLock::new(Vec::new()));

    // Shutdown signal channel
    let (shutdown_tx, shutdown_rx) = tokio::sync::watch::channel(false);

    // ── Task 1: Packet capture loop ───────────────────────────────────────────
    let t_capture = tokio::spawn(capture_loop(
        interface.clone(),
        pcap_filter.clone(),
        analyzer.clone(),
        publisher.clone(),
        ti_feed.clone(),
        ids_rules.clone(),
        metrics.clone(),
        shutdown_rx.clone(),
    ));

    // ── Task 2: Flow timeout checker (every 60 s) ─────────────────────────────
    let t_flow_timeout = {
        let analyzer = analyzer.clone();
        let publisher = publisher.clone();
        let ti_feed = ti_feed.clone();
        let ids_rules = ids_rules.clone();
        let metrics = metrics.clone();
        let mut shutdown_rx2 = shutdown_rx.clone();
        tokio::spawn(async move {
            let mut ticker = interval(Duration::from_secs(60));
            loop {
                tokio::select! {
                    _ = shutdown_rx2.changed() => return,
                    _ = ticker.tick() => {
                        let expired = analyzer.flush_expired().await;
                        debug!(count = expired.len(), "Flushing expired flows");
                        for flow in expired {
                            handle_completed_flow(
                                flow,
                                publisher.clone(),
                                ti_feed.clone(),
                                ids_rules.clone(),
                                metrics.clone(),
                            ).await;
                        }
                        let active = analyzer.active_count().await;
                        metrics.write().await.flows_active = active;
                    }
                }
            }
        })
    };

    // ── Task 3: Prometheus metrics HTTP server ────────────────────────────────
    let t_prometheus = tokio::spawn(metrics_server(prometheus_port, metrics.clone()));

    // ── Task 4: TI feed refresh (every 6 hours) ───────────────────────────────
    let t_ti_refresh = {
        let ti_feed = ti_feed.clone();
        let feeds_url = ti_feeds_url.clone();
        let mut shutdown_rx3 = shutdown_rx.clone();
        tokio::spawn(async move {
            if feeds_url.is_empty() {
                return;
            }
            let mut ticker = interval(Duration::from_secs(6 * 3600));
            ticker.tick().await; // skip immediate tick (already loaded at startup)
            loop {
                tokio::select! {
                    _ = shutdown_rx3.changed() => return,
                    _ = ticker.tick() => {
                        match load_ti_feed(&feeds_url).await {
                            Ok(new_feed) => {
                                *ti_feed.write().await = new_feed;
                                info!("TI feed refreshed");
                            }
                            Err(e) => error!("TI feed refresh failed: {}", e),
                        }
                    }
                }
            }
        })
    };

    // ── Task 5: IDS rule reload watcher (every 5 minutes) ────────────────────
    let t_ids_reload = {
        let ids_rules = ids_rules.clone();
        let mut shutdown_rx4 = shutdown_rx.clone();
        tokio::spawn(async move {
            let mut ticker = interval(Duration::from_secs(300));
            loop {
                tokio::select! {
                    _ = shutdown_rx4.changed() => return,
                    _ = ticker.tick() => {
                        // In production, parse /etc/kubric/netguard.rules or fetch from NATS KV.
                        // For now, insert a sample port-based rule if the list is empty.
                        let mut rules = ids_rules.write().await;
                        if rules.is_empty() {
                            rules.push(IdsRule {
                                name: "SUSPICIOUS-HIGH-PORT-OUTBOUND".to_string(),
                                severity: 5,
                                src_port: None,
                                dst_port: Some(4444),
                                pattern: None,
                            });
                            debug!("IDS rules (re)loaded: {} rules", rules.len());
                        }
                    }
                }
            }
        })
    };

    // ── Graceful shutdown on SIGTERM / Ctrl-C ────────────────────────────────
    tokio::select! {
        _ = signal::ctrl_c() => {
            info!("Received Ctrl-C; shutting down...");
        }
    }

    let _ = shutdown_tx.send(true);

    // Allow tasks up to 5 seconds to drain.
    let _ = tokio::time::timeout(
        Duration::from_secs(5),
        futures_util_join(t_capture, t_flow_timeout, t_prometheus, t_ti_refresh, t_ids_reload),
    )
    .await;

    info!("NetGuard agent stopped.");
    Ok(())
}

async fn futures_util_join(
    t1: tokio::task::JoinHandle<()>,
    t2: tokio::task::JoinHandle<()>,
    t3: tokio::task::JoinHandle<()>,
    t4: tokio::task::JoinHandle<()>,
    t5: tokio::task::JoinHandle<()>,
) {
    let _ = tokio::join!(t1, t2, t3, t4, t5);
}
