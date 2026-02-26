//! K-XRO-NG-PCAP-001 — Network flow analyzer.
//!
//! Aggregates individual packets into bidirectional 5-tuple flows, computes
//! per-flow statistics (bytes, packets, duration, TCP flags), and emits
//! `FlowEvent`s when a flow is explicitly closed (TCP FIN/RST) or when the
//! flow idle-timeout expires.
//!
//! # Cargo dependencies
//! ```toml
//! serde    = { version = "1", features = ["derive"] }
//! tracing  = "0.1"
//! ```

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fmt;
use std::net::IpAddr;
use std::time::{Duration, Instant};
use tracing::debug;

// ─────────────────────────────────────────────────────────────────────────────
// Packet — minimal input type
// ─────────────────────────────────────────────────────────────────────────────

/// Protocol numbers (IP protocol field).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[repr(u8)]
pub enum Protocol {
    Tcp = 6,
    Udp = 17,
    Icmp = 1,
    Other(u8),
}

impl Protocol {
    pub fn from_u8(v: u8) -> Self {
        match v {
            6 => Self::Tcp,
            17 => Self::Udp,
            1 => Self::Icmp,
            x => Self::Other(x),
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Tcp => "TCP",
            Self::Udp => "UDP",
            Self::Icmp => "ICMP",
            Self::Other(_) => "OTHER",
        }
    }
}

impl fmt::Display for Protocol {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// Minimal packet representation passed into the flow analyzer.
#[derive(Debug, Clone)]
pub struct Packet {
    pub src_ip: IpAddr,
    pub dst_ip: IpAddr,
    pub src_port: u16,
    pub dst_port: u16,
    pub protocol: Protocol,
    /// Total payload + header bytes.
    pub length: usize,
    /// TCP flags (OR of TCP_FLAG_* constants below).
    pub tcp_flags: u8,
    /// Capture timestamp (defaults to Instant::now() if not set externally).
    pub captured_at: Instant,
}

/// TCP flag bitmask constants.
pub mod tcp_flags {
    pub const FIN: u8 = 0x01;
    pub const SYN: u8 = 0x02;
    pub const RST: u8 = 0x04;
    pub const PSH: u8 = 0x08;
    pub const ACK: u8 = 0x10;
    pub const URG: u8 = 0x20;
}

// ─────────────────────────────────────────────────────────────────────────────
// FlowKey — canonical 5-tuple
// ─────────────────────────────────────────────────────────────────────────────

/// A canonical 5-tuple flow key.
///
/// To handle bidirectional flows, packets are normalised so that the
/// lower (IP, port) is always stored as "src".
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct FlowKey {
    pub src_ip: IpAddr,
    pub dst_ip: IpAddr,
    pub src_port: u16,
    pub dst_port: u16,
    pub proto: u8,
}

impl FlowKey {
    /// Build a normalised key from a packet (canonical lower-endpoint = src).
    pub fn from_packet(pkt: &Packet) -> Self {
        let proto = match pkt.protocol {
            Protocol::Tcp => 6,
            Protocol::Udp => 17,
            Protocol::Icmp => 1,
            Protocol::Other(n) => n,
        };
        // Normalise so that flows are treated bidirectionally
        if (pkt.src_ip, pkt.src_port) <= (pkt.dst_ip, pkt.dst_port) {
            Self {
                src_ip: pkt.src_ip,
                dst_ip: pkt.dst_ip,
                src_port: pkt.src_port,
                dst_port: pkt.dst_port,
                proto,
            }
        } else {
            Self {
                src_ip: pkt.dst_ip,
                dst_ip: pkt.src_ip,
                src_port: pkt.dst_port,
                dst_port: pkt.src_port,
                proto,
            }
        }
    }

    pub fn protocol_str(&self) -> &'static str {
        Protocol::from_u8(self.proto).as_str()
    }
}

impl fmt::Display for FlowKey {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{}:{} → {}:{} ({})",
            self.src_ip, self.src_port, self.dst_ip, self.dst_port, self.protocol_str()
        )
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Flow — per-flow mutable state
// ─────────────────────────────────────────────────────────────────────────────

/// State of a single bidirectional network flow.
#[derive(Debug, Clone)]
pub struct Flow {
    pub src_ip: IpAddr,
    pub dst_ip: IpAddr,
    pub src_port: u16,
    pub dst_port: u16,
    pub proto: Protocol,
    pub start_time: Instant,
    pub last_time: Instant,
    pub bytes: u64,
    pub packets: u64,
    /// Accumulated TCP flags (OR of all packets in the flow).
    pub flags: u8,
}

impl Flow {
    fn new(pkt: &Packet) -> Self {
        Self {
            src_ip: pkt.src_ip,
            dst_ip: pkt.dst_ip,
            src_port: pkt.src_port,
            dst_port: pkt.dst_port,
            proto: pkt.protocol,
            start_time: pkt.captured_at,
            last_time: pkt.captured_at,
            bytes: pkt.length as u64,
            packets: 1,
            flags: pkt.tcp_flags,
        }
    }

    fn update(&mut self, pkt: &Packet) {
        self.last_time = pkt.captured_at;
        self.bytes += pkt.length as u64;
        self.packets += 1;
        self.flags |= pkt.tcp_flags;
    }

    fn duration_ms(&self) -> u64 {
        self.last_time
            .duration_since(self.start_time)
            .as_millis() as u64
    }

    fn is_tcp_closed(&self) -> bool {
        // Flow considered closed if FIN or RST observed
        self.proto == Protocol::Tcp
            && (self.flags & (tcp_flags::FIN | tcp_flags::RST)) != 0
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// FlowEvent — emitted when a flow is complete
// ─────────────────────────────────────────────────────────────────────────────

/// Reason a flow was exported.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum FlowEndReason {
    /// TCP FIN or RST seen.
    TcpClosed,
    /// Flow idle for longer than the configured timeout.
    IdleTimeout,
    /// Forced flush (e.g. agent shutdown).
    Forced,
}

/// Final connection state label (simplified).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConnectionState {
    Established,
    Closed,
    Reset,
    Incomplete,
}

/// A completed flow record suitable for NATS publication or anomaly scoring.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FlowEvent {
    pub key: FlowKeySerializable,
    pub duration_ms: u64,
    pub bytes: u64,
    pub packets: u64,
    pub tcp_flags: u8,
    pub connection_state: ConnectionState,
    pub end_reason: FlowEndReason,
    /// Unix millis of flow start.
    pub start_ms: u64,
    /// Unix millis of flow end.
    pub end_ms: u64,
}

/// JSON-serialisable version of FlowKey (IpAddr implements Serialize already
/// but we store the string representation for readability).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FlowKeySerializable {
    pub src_ip: String,
    pub dst_ip: String,
    pub src_port: u16,
    pub dst_port: u16,
    pub proto: String,
}

impl FlowKeySerializable {
    fn from_flow(flow: &Flow) -> Self {
        Self {
            src_ip: flow.src_ip.to_string(),
            dst_ip: flow.dst_ip.to_string(),
            src_port: flow.src_port,
            dst_port: flow.dst_port,
            proto: flow.proto.as_str().to_string(),
        }
    }
}

fn connection_state(flow: &Flow) -> ConnectionState {
    if flow.proto != Protocol::Tcp {
        return ConnectionState::Established;
    }
    let has_syn = flow.flags & tcp_flags::SYN != 0;
    let has_fin = flow.flags & tcp_flags::FIN != 0;
    let has_rst = flow.flags & tcp_flags::RST != 0;
    if has_rst {
        ConnectionState::Reset
    } else if has_syn && has_fin {
        ConnectionState::Closed
    } else if has_syn {
        ConnectionState::Incomplete
    } else {
        ConnectionState::Established
    }
}

/// Convert a flow + end reason to a `FlowEvent`.
fn flow_to_event(flow: &Flow, reason: FlowEndReason) -> FlowEvent {
    use std::time::{SystemTime, UNIX_EPOCH};
    // We can't convert Instant to UNIX time directly, so we use
    // SystemTime::now() ± elapsed difference as an approximation.
    let now_ms = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as u64;
    let elapsed_since_last = flow.last_time.elapsed().as_millis() as u64;
    let elapsed_since_start = flow.start_time.elapsed().as_millis() as u64;
    let end_ms = now_ms.saturating_sub(elapsed_since_last);
    let start_ms = now_ms.saturating_sub(elapsed_since_start);

    FlowEvent {
        key: FlowKeySerializable::from_flow(flow),
        duration_ms: flow.duration_ms(),
        bytes: flow.bytes,
        packets: flow.packets,
        tcp_flags: flow.flags,
        connection_state: connection_state(flow),
        end_reason: reason,
        start_ms,
        end_ms,
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// FlowAnalyzer
// ─────────────────────────────────────────────────────────────────────────────

/// Aggregates packets into flows and emits `FlowEvent`s.
///
/// # Example
/// ```rust
/// let mut analyzer = FlowAnalyzer::new(Duration::from_secs(120));
/// for pkt in packet_stream {
///     if let Some(event) = analyzer.process_packet(&pkt) {
///         publish(event);
///     }
/// }
/// // Flush all remaining flows on shutdown
/// for event in analyzer.expire_flows() {
///     publish(event);
/// }
/// ```
pub struct FlowAnalyzer {
    flows: HashMap<FlowKey, Flow>,
    /// A flow is expired if idle for longer than this duration.
    timeout: Duration,
}

impl FlowAnalyzer {
    /// Create a new analyzer with the given idle-flow timeout.
    pub fn new(timeout: Duration) -> Self {
        Self {
            flows: HashMap::new(),
            timeout,
        }
    }

    /// Process a single packet.
    ///
    /// * Upserts the flow for the packet's 5-tuple.
    /// * Returns `Some(FlowEvent)` if the packet completes the flow (TCP FIN/RST),
    ///   `None` otherwise.
    pub fn process_packet(&mut self, pkt: &Packet) -> Option<FlowEvent> {
        let key = FlowKey::from_packet(pkt);

        let flow = self.flows.entry(key.clone()).or_insert_with(|| Flow::new(pkt));
        flow.update(pkt);

        let closed = flow.is_tcp_closed();
        if closed {
            let flow = self.flows.remove(&key)?;
            let event = flow_to_event(&flow, FlowEndReason::TcpClosed);
            debug!(key = %key, "Flow closed (TCP FIN/RST)");
            return Some(event);
        }

        None
    }

    /// Flush all flows that have been idle longer than the configured timeout.
    /// Call this periodically (e.g. every 30 seconds) to prevent the flow map
    /// from growing without bound.
    pub fn expire_flows(&mut self) -> Vec<FlowEvent> {
        let timeout = self.timeout;
        let mut expired = Vec::new();
        self.flows.retain(|key, flow| {
            if flow.last_time.elapsed() >= timeout {
                debug!(key = %key, "Flow expired (idle timeout)");
                expired.push(flow_to_event(flow, FlowEndReason::IdleTimeout));
                false
            } else {
                true
            }
        });
        expired
    }

    /// Forcefully flush all flows regardless of age (e.g. on shutdown).
    pub fn flush_all(&mut self) -> Vec<FlowEvent> {
        let events: Vec<FlowEvent> = self
            .flows
            .values()
            .map(|f| flow_to_event(f, FlowEndReason::Forced))
            .collect();
        self.flows.clear();
        events
    }

    /// Number of active flows currently tracked.
    pub fn active_flow_count(&self) -> usize {
        self.flows.len()
    }

    /// Change the idle timeout at runtime.
    pub fn set_timeout(&mut self, timeout: Duration) {
        self.timeout = timeout;
    }

    /// Iterator over active flow keys (for diagnostics).
    pub fn active_keys(&self) -> impl Iterator<Item = &FlowKey> {
        self.flows.keys()
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::{IpAddr, Ipv4Addr};

    fn ip(a: u8, b: u8, c: u8, d: u8) -> IpAddr {
        IpAddr::V4(Ipv4Addr::new(a, b, c, d))
    }

    fn pkt(src: IpAddr, dst: IpAddr, sp: u16, dp: u16, proto: Protocol, len: usize, flags: u8) -> Packet {
        Packet {
            src_ip: src,
            dst_ip: dst,
            src_port: sp,
            dst_port: dp,
            protocol: proto,
            length: len,
            tcp_flags: flags,
            captured_at: Instant::now(),
        }
    }

    #[test]
    fn new_flow_created_on_first_packet() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        let p = pkt(ip(10,0,0,1), ip(10,0,0,2), 12345, 80, Protocol::Tcp, 100, tcp_flags::SYN);
        let ev = a.process_packet(&p);
        assert!(ev.is_none(), "SYN alone should not close flow");
        assert_eq!(a.active_flow_count(), 1);
    }

    #[test]
    fn flow_accumulates_packets() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        let src = ip(10,0,0,1);
        let dst = ip(10,0,0,2);
        for _ in 0..5 {
            let p = pkt(src, dst, 12345, 80, Protocol::Tcp, 200, tcp_flags::ACK);
            a.process_packet(&p);
        }
        assert_eq!(a.active_flow_count(), 1);
    }

    #[test]
    fn tcp_fin_closes_flow() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        let src = ip(10,0,0,1);
        let dst = ip(10,0,0,2);
        a.process_packet(&pkt(src, dst, 1234, 80, Protocol::Tcp, 100, tcp_flags::SYN));
        a.process_packet(&pkt(src, dst, 1234, 80, Protocol::Tcp, 50, tcp_flags::ACK));
        let ev = a.process_packet(&pkt(src, dst, 1234, 80, Protocol::Tcp, 20, tcp_flags::FIN | tcp_flags::ACK));
        assert!(ev.is_some(), "FIN should close the flow");
        let ev = ev.unwrap();
        assert_eq!(ev.end_reason, FlowEndReason::TcpClosed);
        assert_eq!(a.active_flow_count(), 0);
    }

    #[test]
    fn tcp_rst_closes_flow() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        let src = ip(172,16,0,1);
        let dst = ip(172,16,0,2);
        a.process_packet(&pkt(src, dst, 9999, 443, Protocol::Tcp, 100, tcp_flags::SYN));
        let ev = a.process_packet(&pkt(src, dst, 9999, 443, Protocol::Tcp, 20, tcp_flags::RST));
        assert!(ev.is_some());
        assert_eq!(ev.unwrap().end_reason, FlowEndReason::TcpClosed);
    }

    #[test]
    fn expiry_removes_idle_flows() {
        let mut a = FlowAnalyzer::new(Duration::from_millis(0)); // immediate expiry
        let p = pkt(ip(1,2,3,4), ip(5,6,7,8), 1111, 22, Protocol::Tcp, 60, 0);
        a.process_packet(&p);
        assert_eq!(a.active_flow_count(), 1);
        let expired = a.expire_flows();
        assert_eq!(expired.len(), 1);
        assert_eq!(expired[0].end_reason, FlowEndReason::IdleTimeout);
        assert_eq!(a.active_flow_count(), 0);
    }

    #[test]
    fn flush_all_clears_all_flows() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        for port in 1000..1005u16 {
            let p = pkt(ip(10,0,0,1), ip(10,0,0,2), port, 80, Protocol::Udp, 100, 0);
            a.process_packet(&p);
        }
        assert_eq!(a.active_flow_count(), 5);
        let evs = a.flush_all();
        assert_eq!(evs.len(), 5);
        assert_eq!(a.active_flow_count(), 0);
        assert!(evs.iter().all(|e| e.end_reason == FlowEndReason::Forced));
    }

    #[test]
    fn bidirectional_flow_same_key() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        let src = ip(192,168,1,1);
        let dst = ip(192,168,1,2);
        // Forward packet
        a.process_packet(&pkt(src, dst, 5000, 80, Protocol::Tcp, 100, tcp_flags::SYN));
        // Reverse packet — should update the same flow
        a.process_packet(&pkt(dst, src, 80, 5000, Protocol::Tcp, 100, tcp_flags::SYN | tcp_flags::ACK));
        assert_eq!(a.active_flow_count(), 1, "bidirectional traffic should be one flow");
    }

    #[test]
    fn udp_flow_does_not_close_on_flags() {
        let mut a = FlowAnalyzer::new(Duration::from_secs(60));
        let p = pkt(ip(10,0,0,1), ip(8,8,8,8), 54321, 53, Protocol::Udp, 80, 0);
        let ev = a.process_packet(&p);
        assert!(ev.is_none(), "UDP flows do not auto-close");
        assert_eq!(a.active_flow_count(), 1);
    }

    #[test]
    fn flow_event_bytes_and_packets() {
        let mut a = FlowAnalyzer::new(Duration::from_millis(0));
        let src = ip(10,0,0,1);
        let dst = ip(10,0,0,2);
        a.process_packet(&pkt(src, dst, 9000, 9001, Protocol::Udp, 500, 0));
        a.process_packet(&pkt(src, dst, 9000, 9001, Protocol::Udp, 300, 0));
        let evs = a.expire_flows();
        assert_eq!(evs.len(), 1);
        assert_eq!(evs[0].bytes, 800);
        assert_eq!(evs[0].packets, 2);
    }

    #[test]
    fn flow_key_display() {
        let key = FlowKey {
            src_ip: ip(10, 0, 0, 1),
            dst_ip: ip(10, 0, 0, 2),
            src_port: 1234,
            dst_port: 80,
            proto: 6,
        };
        let s = key.to_string();
        assert!(s.contains("10.0.0.1"));
        assert!(s.contains("TCP"));
    }

    #[test]
    fn protocol_from_u8() {
        assert_eq!(Protocol::from_u8(6), Protocol::Tcp);
        assert_eq!(Protocol::from_u8(17), Protocol::Udp);
        assert_eq!(Protocol::from_u8(1), Protocol::Icmp);
        assert!(matches!(Protocol::from_u8(255), Protocol::Other(255)));
    }
}
