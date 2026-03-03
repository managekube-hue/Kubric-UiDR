//! NetGuard packet capture loop.
//!
//! Opens a pcap capture on the configured interface, dissects each frame
//! with pnet, classifies via heuristic DPI + nDPI, scans payloads with
//! YARA-X, checks IPs against IPsum blocklist, extracts TLS SNI, and
//! publishes enriched OCSF NetworkActivity (class 4001) events to NATS.
//!
//! An async background task periodically polls the RITA sidecar for
//! beacon / DNS-tunneling / long-connection indicators and publishes
//! those as separate OCSF detection events.
//!
//! # Graceful degradation
//! - If libpcap is unavailable the capture falls back to logging-only mode.
//! - If RITA is unreachable, polling logs a warning and retries every 60 s.
//! - NATS is always required.

use anyhow::{Context, Result};
use serde_json::json;
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tracing::{error, info, warn};

use pnet::packet::{
    ethernet::{EtherTypes, EthernetPacket},
    ip::IpNextHeaderProtocols,
    ipv4::Ipv4Packet,
    ipv6::Ipv6Packet,
    tcp::TcpPacket,
    udp::UdpPacket,
    Packet,
};

use crate::config::Config;
use crate::dpi::DpiEngine;
use crate::ids::IdsEngine;
use crate::ipsum_lookup::IpsumLookup;
use crate::rita_client::RitaClient;
use crate::tls;
use crate::waf::WafEngine;
use crate::zeek::ZeekRunner;

// ---------------------------------------------------------------------------
// NATS subjects — aligned with docs/KUBRIC Orchestration.docx.md §19
// ---------------------------------------------------------------------------

/// NATS subject for network activity events (OCSF 4001).
fn network_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.network.activity.v1")
}

/// NATS subject for IDS detection alerts.
fn ids_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.detection.network_ids.v1")
}

/// NATS subject for C2 detection alerts.
fn c2_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.detection.network_c2.v1")
}

/// NATS subject for RITA behavioural detection events.
fn rita_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.detection.rita.v1")
}

/// NATS subject for TLS fingerprint detections (JA3/JA3S).
fn tls_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.detection.network_tls.v1")
}

/// NATS subject for WAF detections.
fn waf_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.detection.network_waf.v1")
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

/// Entry point called from main.  Runs until SIGINT/SIGTERM.
pub async fn run(cfg: Config) -> Result<()> {
    let nats = async_nats::connect(&cfg.nats_url)
        .await
        .context("connect to NATS")?;

    info!(
        tenant_id = %cfg.tenant_id,
        interface = %cfg.interface,
        "NetGuard capture starting"
    );

    // --- Initialise enrichment engines -----------------------------------

    let dpi_engine = DpiEngine::new();
    info!(ndpi = dpi_engine.has_ndpi(), "DPI engine ready");

    let yara_dir = std::env::var("KUBRIC_YARA_DIR")
        .unwrap_or_else(|_| "vendor/yara-rules".to_string());
    let suricata_dir = std::env::var("KUBRIC_SURICATA_DIR")
        .unwrap_or_else(|_| "vendor/suricata".to_string());
    let strict_suricata = std::env::var("KUBRIC_STRICT_SURICATA")
        .map(|v| v.eq_ignore_ascii_case("true") || v == "1")
        .unwrap_or(false);

    let suricata_validation = IdsEngine::validate_suricata_assets(&suricata_dir);
    if !suricata_validation.missing_files.is_empty() {
        if strict_suricata {
            return Err(anyhow::anyhow!(
                "required Suricata assets missing: {:?}",
                suricata_validation.missing_files
            ));
        }
        warn!(
            missing = ?suricata_validation.missing_files,
            "Suricata rule set incomplete; C2 coverage may be reduced"
        );
    }

    let ids_engine = IdsEngine::load_from_dirs(&yara_dir, &suricata_dir)
        .unwrap_or_else(|e| {
            warn!(%e, "IDS rule load failed — scanning disabled");
            IdsEngine::empty()
        });
    info!(rules = ids_engine.rule_count(), "IDS engine ready");

    let ipsum = IpsumLookup::from_env();
    info!(ips = ipsum.count(), "IPsum blocklist ready");

    let waf_dir = std::env::var("KUBRIC_CRS_DIR")
        .unwrap_or_else(|_| "vendor/coreruleset/rules".to_string());
    let waf_engine = WafEngine::load_from_dir(&waf_dir);

    if let Some(zeek) = ZeekRunner::from_env() {
        info!(scripts = zeek.script_count(), "Zeek integration enabled");
        let tls_events = zeek.run_once();
        let tls_subj = tls_subject(&cfg.tenant_id);
        for evt in tls_events {
            let msg = json!({
                "tenant_id": cfg.tenant_id,
                "agent_id": cfg.agent_id,
                "class_uid": 4001,
                "activity_id": 6,
                "severity_id": 2,
                "category_name": "Network Activity",
                "type_name": "TLS Fingerprint",
                "src_endpoint": { "ip": evt.src_ip },
                "dst_endpoint": { "ip": evt.dst_ip },
                "tls": {
                    "ja3": evt.ja3,
                    "ja3s": evt.ja3s,
                    "sni": evt.server_name,
                },
            });
            publish_json(&nats, &tls_subj, &msg).await;
        }
    }

    // --- Shared shutdown flag --------------------------------------------

    let shutdown = Arc::new(AtomicBool::new(false));
    let shutdown_signal = shutdown.clone();
    tokio::spawn(async move {
        tokio::signal::ctrl_c().await.ok();
        shutdown_signal.store(true, Ordering::Relaxed);
    });

    // --- RITA polling task (async, every 60 s) ----------------------------
    {
        let nats_rita = nats.clone();
        let tenant_id = cfg.tenant_id.clone();
        let shutdown_rita = shutdown.clone();
        tokio::spawn(async move {
            rita_poll_loop(&nats_rita, &tenant_id, &shutdown_rita).await;
        });
    }

    // --- pcap capture ----------------------------------------------------

    let cap_result = open_capture(&cfg.interface);

    match cap_result {
        Err(err) => {
            warn!(%err, interface = %cfg.interface,
                "pcap open failed — running in degraded (no-capture) mode");
            while !shutdown.load(Ordering::Relaxed) {
                tokio::time::sleep(Duration::from_millis(500)).await;
            }
        }
        Ok(cap) => {
            let cfg_clone = cfg.clone();
            let shutdown_clone = shutdown.clone();
            let nats_clone = nats.clone();

            let handle = tokio::task::spawn_blocking(move || {
                run_capture_blocking(
                    cap,
                    &nats_clone,
                    &cfg_clone,
                    &shutdown_clone,
                    &dpi_engine,
                    &ids_engine,
                    &ipsum,
                    &waf_engine,
                );
            });

            while !shutdown.load(Ordering::Relaxed) {
                tokio::time::sleep(Duration::from_millis(200)).await;
            }
            let _ = handle.await;
        }
    }

    info!("NetGuard shutting down");
    Ok(())
}

fn open_capture(interface: &str) -> Result<pcap::Capture<pcap::Active>> {
    pcap::Capture::from_device(interface)
        .context("pcap device lookup")?
        .promisc(true)
        .snaplen(65_535)
        .timeout(500)
        .open()
        .context("pcap open")
}

// ---------------------------------------------------------------------------
// RITA polling loop
// ---------------------------------------------------------------------------

/// Periodically queries the RITA sidecar for beacon, DNS-tunneling, and
/// long-connection indicators and publishes OCSF detection events.
async fn rita_poll_loop(nats: &async_nats::Client, tenant_id: &str, shutdown: &AtomicBool) {
    let rita = RitaClient::from_env();
    let subject = rita_subject(tenant_id);

    // Give RITA time to warm up.
    tokio::time::sleep(Duration::from_secs(30)).await;

    while !shutdown.load(Ordering::Relaxed) {
        if rita.health_check().await {
            let ts = epoch_secs();

            for b in rita.get_beacons(tenant_id).await {
                let event = json!({
                    "class_uid": 4001,
                    "activity_id": 6,
                    "severity_id": if b.score > 0.8 { 4 } else { 2 },
                    "category_name": "Network Activity",
                    "type_name": "Beacon Detected",
                    "timestamp": ts,
                    "src_endpoint": { "ip": b.src_ip },
                    "dst_endpoint": { "ip": b.dst_ip },
                    "analytic": {
                        "name": "RITA Beacon",
                        "type": "Behavioral",
                        "score": b.score,
                        "ts_score": b.ts_score,
                        "ds_score": b.ds_score,
                        "dur_score": b.dur_score,
                        "connections": b.connections,
                    },
                });
                publish_json(nats, &subject, &event).await;
            }

            for t in rita.get_dns_tunneling(tenant_id).await {
                let event = json!({
                    "class_uid": 4001,
                    "activity_id": 6,
                    "severity_id": if t.score > 0.7 { 4 } else { 2 },
                    "category_name": "Network Activity",
                    "type_name": "DNS Tunneling Detected",
                    "timestamp": ts,
                    "src_endpoint": { "ip": t.src_ip },
                    "analytic": {
                        "name": "RITA DNS Tunnel",
                        "fqdn": t.fqdn,
                        "score": t.score,
                        "query_count": t.query_count,
                        "unique_subdomains": t.unique_subdomains,
                    },
                });
                publish_json(nats, &subject, &event).await;
            }

            for lc in rita.get_long_connections(tenant_id).await {
                let event = json!({
                    "class_uid": 4001,
                    "activity_id": 6,
                    "severity_id": 2,
                    "category_name": "Network Activity",
                    "type_name": "Long Connection Detected",
                    "timestamp": ts,
                    "src_endpoint": { "ip": lc.src_ip },
                    "dst_endpoint": { "ip": lc.dst_ip },
                    "analytic": {
                        "name": "RITA Long Connection",
                        "duration_secs": lc.duration_secs,
                        "bytes_sent": lc.bytes_sent,
                        "bytes_received": lc.bytes_received,
                    },
                });
                publish_json(nats, &subject, &event).await;
            }
        } else {
            warn!("RITA sidecar unreachable — retrying in 60 s");
        }

        tokio::time::sleep(Duration::from_secs(60)).await;
    }
}

// ---------------------------------------------------------------------------
// Blocking capture loop — full enrichment pipeline
// ---------------------------------------------------------------------------

/// Blocking capture loop — runs on a dedicated OS thread.
///
/// Pipeline per packet: dissect → DPI classify → TLS SNI → IPsum check
/// → YARA scan → build enriched OCSF event → publish to NATS.
fn run_capture_blocking(
    mut cap: pcap::Capture<pcap::Active>,
    nats: &async_nats::Client,
    cfg: &Config,
    shutdown: &AtomicBool,
    dpi: &DpiEngine,
    ids: &IdsEngine,
    ipsum: &IpsumLookup,
    waf: &WafEngine,
) {
    let subject = network_subject(&cfg.tenant_id);
    let ids_subj = ids_subject(&cfg.tenant_id);
    let c2_subj = c2_subject(&cfg.tenant_id);
    let waf_subj = waf_subject(&cfg.tenant_id);
    let rt = tokio::runtime::Handle::current();
    let mut packet_count: u64 = 0;

    while !shutdown.load(Ordering::Relaxed) {
        match cap.next_packet() {
            Ok(packet) => {
                packet_count += 1;
                if let Some(enriched) = dissect_and_enrich(
                    packet.data, cfg, dpi, ids, ipsum, waf,
                ) {
                    // Publish OCSF network-activity event
                    if let Ok(payload) = serde_json::to_vec(&enriched.ocsf) {
                        let s = subject.clone();
                        let n = nats.clone();
                        rt.spawn(async move {
                            if let Err(e) = n.publish(s, payload.into()).await {
                                error!(%e, "NATS publish network event failed");
                            }
                        });
                    }
                    // Publish IDS alerts (YARA matches)
                    for alert in &enriched.ids_alerts {
                        if let Ok(payload) = serde_json::to_vec(alert) {
                            let s = ids_subj.clone();
                            let n = nats.clone();
                            rt.spawn(async move {
                                if let Err(e) = n.publish(s, payload.into()).await {
                                    error!(%e, "NATS publish IDS alert failed");
                                }
                            });
                        }
                    }
                    for alert in &enriched.c2_alerts {
                        if let Ok(payload) = serde_json::to_vec(alert) {
                            let s = c2_subj.clone();
                            let n = nats.clone();
                            rt.spawn(async move {
                                if let Err(e) = n.publish(s, payload.into()).await {
                                    error!(%e, "NATS publish C2 alert failed");
                                }
                            });
                        }
                    }
                    for alert in &enriched.waf_alerts {
                        if let Ok(payload) = serde_json::to_vec(alert) {
                            let s = waf_subj.clone();
                            let n = nats.clone();
                            rt.spawn(async move {
                                if let Err(e) = n.publish(s, payload.into()).await {
                                    error!(%e, "NATS publish WAF alert failed");
                                }
                            });
                        }
                    }
                }
                if packet_count % 10_000 == 0 {
                    info!(packets = packet_count, "capture stats");
                }
            }
            Err(pcap::Error::TimeoutExpired) => {}
            Err(e) => {
                error!(%e, "pcap error — stopping capture");
                break;
            }
        }
    }
}

// ---------------------------------------------------------------------------
// Enriched packet result
// ---------------------------------------------------------------------------

struct EnrichedPacket {
    ocsf: serde_json::Value,
    ids_alerts: Vec<serde_json::Value>,
    c2_alerts: Vec<serde_json::Value>,
    waf_alerts: Vec<serde_json::Value>,
}

/// Dissect a raw Ethernet frame, run all enrichment engines, and build an
/// enriched OCSF NetworkActivity event plus any IDS alerts.
fn dissect_and_enrich(
    data: &[u8],
    cfg: &Config,
    dpi: &DpiEngine,
    ids: &IdsEngine,
    ipsum: &IpsumLookup,
    waf: &WafEngine,
) -> Option<EnrichedPacket> {
    let eth = EthernetPacket::new(data)?;

    let (l4_proto, src_ip, dst_ip, src_port, dst_port, bytes, l7_payload) =
        match eth.get_ethertype() {
            EtherTypes::Ipv4 => dissect_ipv4(eth.payload())?,
            EtherTypes::Ipv6 => dissect_ipv6(eth.payload())?,
            _ => return None,
        };

    let ts = epoch_secs();
    let event_id = make_event_id(ts, &src_ip, &dst_ip, src_port, dst_port);
    let raw = format!("{src_ip}:{src_port}->{dst_ip}:{dst_port}:{l4_proto}:{ts}");
    let blake3_hash = blake3::hash(raw.as_bytes()).to_hex().to_string();

    // --- DPI classification ---
    let dpi_result = dpi.classify(
        l7_payload.as_deref().unwrap_or(&[]),
        src_port as u16,
        dst_port as u16,
    );

    // --- TLS SNI extraction ---
    let sni = l7_payload
        .as_deref()
        .and_then(tls::parse_client_hello)
        .map(|ch| ch.sni);

    // --- IPsum reputation ---
    let src_rep = ipsum.lookup(&src_ip);
    let dst_rep = ipsum.lookup(&dst_ip);
    let severity = if src_rep.is_malicious || dst_rep.is_malicious { 4 } else { 1 };

    // --- YARA-X payload scan ---
    let mut ids_alerts = Vec::new();
    let mut c2_alerts = Vec::new();
    let mut waf_alerts = Vec::new();
    if let Some(ref payload) = l7_payload {
        if !payload.is_empty() {
            for (rule_name, namespace) in ids.scan(payload) {
                let alert = json!({
                    "tenant_id": cfg.tenant_id,
                    "agent_id": cfg.agent_id,
                    "timestamp": ts,
                    "class_uid": 4001,
                    "activity_id": 5,
                    "severity_id": 4,
                    "category_name": "Network Activity",
                    "type_name": "IDS Alert",
                    "rule_id": rule_name,
                    "rule_namespace": namespace,
                    "src_endpoint": { "ip": &src_ip, "port": src_port },
                    "dst_endpoint": { "ip": &dst_ip, "port": dst_port },
                    "connection_info": { "protocol_name": &dpi_result.protocol },
                });
                if namespace.eq_ignore_ascii_case("suricata")
                    && rule_name.to_lowercase().contains("c2")
                {
                    c2_alerts.push(json!({
                        "tenant_id": cfg.tenant_id,
                        "agent_id": cfg.agent_id,
                        "timestamp": ts,
                        "class_uid": 4001,
                        "activity_id": 6,
                        "severity_id": 4,
                        "category_name": "Network Activity",
                        "type_name": "C2 Detection",
                        "analytic": {
                            "name": "Suricata C2 Rule",
                            "rule": rule_name,
                        },
                        "src_endpoint": { "ip": &src_ip, "port": src_port },
                        "dst_endpoint": { "ip": &dst_ip, "port": dst_port },
                        "connection_info": { "protocol_name": &dpi_result.protocol },
                    }));
                }
                ids_alerts.push(alert);
            }

            if dpi_result.protocol.eq_ignore_ascii_case("HTTP") {
                for (rule_id, msg) in waf.inspect_http_payload(payload) {
                    waf_alerts.push(json!({
                        "tenant_id": cfg.tenant_id,
                        "agent_id": cfg.agent_id,
                        "timestamp": ts,
                        "class_uid": 4001,
                        "activity_id": 5,
                        "severity_id": 3,
                        "category_name": "Network Activity",
                        "type_name": "WAF Alert",
                        "rule_id": rule_id,
                        "rule_message": msg,
                        "src_endpoint": { "ip": &src_ip, "port": src_port },
                        "dst_endpoint": { "ip": &dst_ip, "port": dst_port },
                    }));
                }
            }
        }
    }

    // --- Build enriched OCSF event ---
    let mut ocsf = json!({
        "tenant_id":   cfg.tenant_id,
        "agent_id":    cfg.agent_id,
        "event_id":    event_id,
        "timestamp":   ts,
        "class_uid":   4001,
        "activity_id": 1,
        "severity_id": severity,
        "src_endpoint": {
            "ip": src_ip, "port": src_port,
            "reputation": {
                "is_malicious": src_rep.is_malicious,
                "score": src_rep.threat_score,
                "source": src_rep.source,
            },
        },
        "dst_endpoint": {
            "ip": dst_ip, "port": dst_port,
            "reputation": {
                "is_malicious": dst_rep.is_malicious,
                "score": dst_rep.threat_score,
                "source": dst_rep.source,
            },
        },
        "connection_info": {
            "protocol_name": l4_proto,
            "direction_id": 0,
        },
        "app_protocol": {
            "name": dpi_result.protocol,
            "category": dpi_result.category,
            "confidence": dpi_result.confidence,
            "method": format!("{:?}", dpi_result.method),
        },
        "traffic": { "bytes_in": bytes, "bytes_out": 0 },
        "blake3_hash": blake3_hash,
    });

    if let Some(ref sni_val) = sni {
        ocsf["tls"] = json!({ "sni": sni_val });
    }
    if !ids_alerts.is_empty() {
        ocsf["ids_match_count"] = json!(ids_alerts.len());
    }

    Some(EnrichedPacket { ocsf, ids_alerts, c2_alerts, waf_alerts })
}

// ---------------------------------------------------------------------------
// Dissection helpers — accepts ALL TCP packets (not SYN-only) so DPI, YARA,
// and TLS SNI can inspect payload data.
// ---------------------------------------------------------------------------

fn dissect_ipv4(
    ip_data: &[u8],
) -> Option<(&'static str, String, String, u32, u32, usize, Option<Vec<u8>>)> {
    let ip = Ipv4Packet::new(ip_data)?;
    let src = ip.get_source().to_string();
    let dst = ip.get_destination().to_string();
    let plen = ip.get_total_length() as usize;
    match ip.get_next_level_protocol() {
        IpNextHeaderProtocols::Tcp => {
            let tcp = TcpPacket::new(ip.payload())?;
            let payload = tcp.payload();
            let l7 = if payload.is_empty() { None } else { Some(payload.to_vec()) };
            Some(("TCP", src, dst, tcp.get_source() as u32, tcp.get_destination() as u32, plen, l7))
        }
        IpNextHeaderProtocols::Udp => {
            let udp = UdpPacket::new(ip.payload())?;
            let payload = udp.payload();
            let l7 = if payload.is_empty() { None } else { Some(payload.to_vec()) };
            Some(("UDP", src, dst, udp.get_source() as u32, udp.get_destination() as u32, plen, l7))
        }
        _ => None,
    }
}

fn dissect_ipv6(
    ip_data: &[u8],
) -> Option<(&'static str, String, String, u32, u32, usize, Option<Vec<u8>>)> {
    let ip = Ipv6Packet::new(ip_data)?;
    let src = ip.get_source().to_string();
    let dst = ip.get_destination().to_string();
    let plen = ip.get_payload_length() as usize;
    match ip.get_next_header() {
        IpNextHeaderProtocols::Tcp => {
            let tcp = TcpPacket::new(ip.payload())?;
            let payload = tcp.payload();
            let l7 = if payload.is_empty() { None } else { Some(payload.to_vec()) };
            Some(("TCP", src, dst, tcp.get_source() as u32, tcp.get_destination() as u32, plen, l7))
        }
        IpNextHeaderProtocols::Udp => {
            let udp = UdpPacket::new(ip.payload())?;
            let payload = udp.payload();
            let l7 = if payload.is_empty() { None } else { Some(payload.to_vec()) };
            Some(("UDP", src, dst, udp.get_source() as u32, udp.get_destination() as u32, plen, l7))
        }
        _ => None,
    }
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

fn epoch_secs() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

fn make_event_id(ts: u64, src: &str, dst: &str, sp: u32, dp: u32) -> String {
    let mut h = DefaultHasher::new();
    ts.hash(&mut h);
    src.hash(&mut h);
    dst.hash(&mut h);
    sp.hash(&mut h);
    dp.hash(&mut h);
    format!("{:016x}", h.finish())
}

async fn publish_json(nats: &async_nats::Client, subject: &str, val: &serde_json::Value) {
    if let Ok(payload) = serde_json::to_vec(val) {
        if let Err(e) = nats.publish(subject.to_string(), payload.into()).await {
            error!(%e, "NATS publish failed");
        }
    }
}
