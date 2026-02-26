//! NetGuard packet capture loop.
//!
//! Opens a pcap capture on the configured interface, dissects each frame
//! with pnet, and publishes OCSF NetworkActivity (class 4001) events to NATS.
//!
//! # Graceful degradation
//! If libpcap is unavailable (no permissions, WinPcap/npcap not installed, etc.)
//! the capture falls back to logging-only mode.  NATS is still required.

use anyhow::{Context, Result};
use serde_json::json;
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
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

/// NATS subject for network activity events.
fn network_subject(tenant_id: &str) -> String {
    format!("kubric.{tenant_id}.network.activity.v1")
}

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

    // Shared shutdown flag — set by the async signal handler, read by the
    // blocking pcap loop via AtomicBool.
    let shutdown = Arc::new(AtomicBool::new(false));
    let shutdown_signal = shutdown.clone();

    tokio::spawn(async move {
        tokio::signal::ctrl_c().await.ok();
        shutdown_signal.store(true, Ordering::Relaxed);
    });

    // Attempt to open the pcap capture.
    let cap_result = open_capture(&cfg.interface);

    match cap_result {
        Err(err) => {
            warn!(%err, interface = %cfg.interface,
                "pcap open failed — running in degraded (no-capture) mode");
            // Wait for shutdown signal
            while !shutdown.load(Ordering::Relaxed) {
                tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;
            }
        }
        Ok(cap) => {
            // Move the blocking capture loop to a dedicated OS thread so it
            // doesn't block the tokio runtime.
            let cfg_clone = cfg.clone();
            let shutdown_clone = shutdown.clone();
            let nats_clone = nats.clone();

            let handle = tokio::task::spawn_blocking(move || {
                run_capture_blocking(cap, &nats_clone, &cfg_clone, &shutdown_clone);
            });

            // Wait until shutdown flag is set
            while !shutdown.load(Ordering::Relaxed) {
                tokio::time::sleep(tokio::time::Duration::from_millis(200)).await;
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
        .timeout(500) // ms — keeps the loop responsive to shutdown flag
        .open()
        .context("pcap open")
}

/// Blocking capture loop — runs on a dedicated OS thread.
fn run_capture_blocking(
    mut cap: pcap::Capture<pcap::Active>,
    nats: &async_nats::Client,
    cfg: &Config,
    shutdown: &AtomicBool,
) {
    let subject = network_subject(&cfg.tenant_id);
    let rt = tokio::runtime::Handle::current();
    let mut packet_count: u64 = 0;

    while !shutdown.load(Ordering::Relaxed) {
        match cap.next_packet() {
            Ok(packet) => {
                packet_count += 1;
                if let Some(event) = dissect_packet(packet.data, cfg) {
                    if let Ok(payload) = serde_json::to_vec(&event) {
                        let subject = subject.clone();
                        let nats = nats.clone();
                        rt.spawn(async move {
                            if let Err(e) = nats.publish(subject, payload.into()).await {
                                error!(%e, "NATS publish network event failed");
                            }
                        });
                    }
                }
                if packet_count % 10_000 == 0 {
                    info!(packets = packet_count, "capture stats");
                }
            }
            Err(pcap::Error::TimeoutExpired) => {
                // Normal — loop back and check shutdown flag
            }
            Err(e) => {
                error!(%e, "pcap error — stopping capture");
                break;
            }
        }
    }
}

/// Dissect a raw Ethernet frame and return an OCSF NetworkActivity JSON value.
/// Returns None for non-IP frames or frames that can't be fully parsed.
fn dissect_packet(data: &[u8], cfg: &Config) -> Option<serde_json::Value> {
    let eth = EthernetPacket::new(data)?;

    let (proto, src_ip, dst_ip, src_port, dst_port, bytes) = match eth.get_ethertype() {
        EtherTypes::Ipv4 => {
            let ip = Ipv4Packet::new(eth.payload())?;
            let src = ip.get_source().to_string();
            let dst = ip.get_destination().to_string();
            let plen = ip.get_total_length() as usize;
            match ip.get_next_level_protocol() {
                IpNextHeaderProtocols::Tcp => {
                    let tcp = TcpPacket::new(ip.payload())?;
                    // Only TCP SYN (new connections)
                    if tcp.get_flags() & 0x02 == 0 {
                        return None;
                    }
                    ("TCP", src, dst, tcp.get_source() as u32, tcp.get_destination() as u32, plen)
                }
                IpNextHeaderProtocols::Udp => {
                    let udp = UdpPacket::new(ip.payload())?;
                    ("UDP", src, dst, udp.get_source() as u32, udp.get_destination() as u32, plen)
                }
                _ => return None,
            }
        }
        EtherTypes::Ipv6 => {
            let ip = Ipv6Packet::new(eth.payload())?;
            let src = ip.get_source().to_string();
            let dst = ip.get_destination().to_string();
            let plen = ip.get_payload_length() as usize;
            match ip.get_next_header() {
                IpNextHeaderProtocols::Tcp => {
                    let tcp = TcpPacket::new(ip.payload())?;
                    if tcp.get_flags() & 0x02 == 0 {
                        return None;
                    }
                    ("TCP", src, dst, tcp.get_source() as u32, tcp.get_destination() as u32, plen)
                }
                IpNextHeaderProtocols::Udp => {
                    let udp = UdpPacket::new(ip.payload())?;
                    ("UDP", src, dst, udp.get_source() as u32, udp.get_destination() as u32, plen)
                }
                _ => return None,
            }
        }
        _ => return None,
    };

    let ts = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();

    let mut h = DefaultHasher::new();
    ts.hash(&mut h);
    src_ip.hash(&mut h);
    dst_ip.hash(&mut h);
    src_port.hash(&mut h);
    dst_port.hash(&mut h);
    let event_id = format!("{:016x}", h.finish());

    let raw = format!("{src_ip}:{src_port}->{dst_ip}:{dst_port}:{proto}:{ts}");
    let blake3_hash = blake3::hash(raw.as_bytes()).to_hex().to_string();

    // OCSF NetworkActivity — class_uid 4001, activity_id 1 (Open)
    Some(json!({
        "tenant_id":   cfg.tenant_id,
        "agent_id":    cfg.agent_id,
        "event_id":    event_id,
        "timestamp":   ts,
        "class_uid":   4001,
        "activity_id": 1,
        "severity_id": 1,
        "src_endpoint": { "ip": src_ip, "port": src_port },
        "dst_endpoint": { "ip": dst_ip, "port": dst_port },
        "connection_info": {
            "protocol_name": proto,
            "direction_id": 0
        },
        "traffic": { "bytes_in": bytes, "bytes_out": 0 },
        "blake3_hash": blake3_hash,
    }))
}
