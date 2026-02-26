//! K-XRO-NG-DPI-002 — L7 application protocol classifier.
//!
//! Combines two classification strategies in priority order:
//!
//! 1. **nDPI** (if available) — deep packet inspection via `NdpiWrapper`
//! 2. **Heuristics** — well-known port mappings + payload byte-signature matching
//!
//! # Supported protocols
//! HTTP, HTTPS/TLS, DNS, SSH, SMB, RDP, SMTP, IMAP, FTP, MySQL, PostgreSQL,
//! Redis, MongoDB, Kafka, NATS, and an open `Unknown(String)` variant.
//!
//! # Confidence scoring
//! | Source                | Confidence |
//! |-----------------------|-----------|
//! | nDPI (app_protocol≠0) | 100        |
//! | Payload byte signature | 90        |
//! | Well-known dst port   | 70        |
//! | Well-known src port   | 60        |
//! | No match              | 0          |

#![allow(dead_code)]

use std::fmt;
use serde::{Deserialize, Serialize};

// ── Protocol enumeration ──────────────────────────────────────────────────────

/// Identified application-layer protocol.
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum L7Protocol {
    HTTP,
    HTTPS,
    DNS,
    SSH,
    /// SMB / CIFS (Windows file sharing, port 445).
    SMB,
    /// Remote Desktop Protocol (port 3389).
    RDP,
    SMTP,
    IMAP,
    FTP,
    MySQL,
    PostgreSQL,
    Redis,
    MongoDB,
    Kafka,
    NATS,
    /// All other protocols with the best-effort name.
    Unknown(String),
}

impl fmt::Display for L7Protocol {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            L7Protocol::Unknown(s) => write!(f, "Unknown({s})"),
            _ => write!(f, "{self:?}"),
        }
    }
}

// ── ClassificationResult ─────────────────────────────────────────────────────

/// Full classification output.
#[derive(Debug, Clone, Serialize)]
pub struct ClassificationResult {
    /// Identified protocol.
    pub protocol: L7Protocol,
    /// 0–100 confidence.
    pub confidence: u8,
    /// Human-readable indicators that led to this classification.
    pub indicators: Vec<String>,
}

impl ClassificationResult {
    fn new(protocol: L7Protocol, confidence: u8, indicators: Vec<String>) -> Self {
        Self { protocol, confidence, indicators }
    }

    fn unknown() -> Self {
        Self::new(L7Protocol::Unknown(String::new()), 0, Vec::new())
    }
}

// ── Port tables ───────────────────────────────────────────────────────────────

/// Map a standard port number to its canonical protocol.
fn port_to_protocol(port: u16) -> Option<L7Protocol> {
    match port {
        80 | 8080 | 8000 | 3000 => Some(L7Protocol::HTTP),
        443 | 8443 | 9443 => Some(L7Protocol::HTTPS),
        53 => Some(L7Protocol::DNS),
        22 => Some(L7Protocol::SSH),
        445 | 139 => Some(L7Protocol::SMB),
        3389 => Some(L7Protocol::RDP),
        25 | 587 | 465 => Some(L7Protocol::SMTP),
        143 | 993 => Some(L7Protocol::IMAP),
        20 | 21 => Some(L7Protocol::FTP),
        3306 => Some(L7Protocol::MySQL),
        5432 => Some(L7Protocol::PostgreSQL),
        6379 => Some(L7Protocol::Redis),
        27017 | 27018 | 27019 => Some(L7Protocol::MongoDB),
        9092 => Some(L7Protocol::Kafka),
        4222 | 8222 => Some(L7Protocol::NATS),
        _ => None,
    }
}

// ── Payload signature matching ────────────────────────────────────────────────

/// Check payload bytes for protocol magic signatures.
fn classify_by_payload(payload: &[u8]) -> Option<(L7Protocol, Vec<String>)> {
    if payload.is_empty() {
        return None;
    }

    // HTTP request methods
    if payload.starts_with(b"GET ") || payload.starts_with(b"POST ")
        || payload.starts_with(b"PUT ") || payload.starts_with(b"DELETE ")
        || payload.starts_with(b"HEAD ") || payload.starts_with(b"OPTIONS ")
        || payload.starts_with(b"PATCH ")
    {
        let method = payload.splitn(2, |&b| b == b' ').next().unwrap_or(b"");
        return Some((
            L7Protocol::HTTP,
            vec![format!("HTTP method: {}", String::from_utf8_lossy(method))],
        ));
    }

    // HTTP response
    if payload.starts_with(b"HTTP/1.") || payload.starts_with(b"HTTP/2") {
        return Some((L7Protocol::HTTP, vec!["HTTP response header".into()]));
    }

    // TLS ClientHello: record type 0x16, version 0x03XX, handshake type 0x01
    if payload.len() >= 6 && payload[0] == 0x16 && payload[1] == 0x03 && payload[5] == 0x01 {
        return Some((L7Protocol::HTTPS, vec!["TLS ClientHello record".into()]));
    }

    // TLS ServerHello / other handshake
    if payload.len() >= 3 && payload[0] == 0x16 && payload[1] == 0x03 {
        return Some((L7Protocol::HTTPS, vec!["TLS handshake record type=0x16".into()]));
    }

    // SSH banner
    if payload.starts_with(b"SSH-") {
        let version = payload
            .splitn(2, |&b| b == b'\n')
            .next()
            .unwrap_or(b"SSH-?");
        return Some((
            L7Protocol::SSH,
            vec![format!("SSH banner: {}", String::from_utf8_lossy(version).trim())],
        ));
    }

    // DNS: query class IN (0x0001), opcode 0 in flags
    if payload.len() >= 12 {
        let flags = u16::from_be_bytes([payload[2], payload[3]]);
        let qr = (flags >> 15) & 1;
        let opcode = (flags >> 11) & 0xf;
        let qdcount = u16::from_be_bytes([payload[4], payload[5]]);
        if opcode == 0 && qdcount > 0 && qdcount < 20 {
            return Some((
                L7Protocol::DNS,
                vec![format!("DNS {} flags={:#06x}", if qr == 0 { "query" } else { "response" }, flags)],
            ));
        }
    }

    // MySQL greeting (server → client starts with 4-byte length + protocol byte ≥ 9)
    if payload.len() >= 5 {
        let pkt_len = u32::from_le_bytes([payload[0], payload[1], payload[2], 0]) as usize;
        let proto_ver = payload[4];
        if payload[3] == 0
            && (proto_ver == 10 || proto_ver == 9)
            && pkt_len + 4 <= payload.len()
        {
            return Some((
                L7Protocol::MySQL,
                vec![format!("MySQL greeting proto_ver={proto_ver}")],
            ));
        }
    }

    // Redis inline command: starts with '*' (RESP array) or '+' / '-' / ':'
    if !payload.is_empty()
        && matches!(payload[0], b'*' | b'+' | b'-' | b':' | b'$')
        && payload.contains(&b'\r')
    {
        return Some((L7Protocol::Redis, vec!["RESP protocol byte".into()]));
    }

    // SMTP greeting / commands
    if payload.starts_with(b"220 ") || payload.starts_with(b"EHLO ")
        || payload.starts_with(b"HELO ") || payload.starts_with(b"MAIL FROM:")
    {
        return Some((L7Protocol::SMTP, vec!["SMTP greeting/command".into()]));
    }

    // IMAP greeting
    if payload.starts_with(b"* OK") || payload.starts_with(b"* BYE") {
        return Some((L7Protocol::IMAP, vec!["IMAP untagged response".into()]));
    }

    // FTP greeting / commands
    if payload.starts_with(b"220-") || payload.starts_with(b"220 ")
        || payload.starts_with(b"USER ") || payload.starts_with(b"PASS ")
        || payload.starts_with(b"PASV")
    {
        return Some((L7Protocol::FTP, vec!["FTP greeting/command".into()]));
    }

    // PostgreSQL startup message: length (4) + magic 0x0003_0000
    if payload.len() >= 8 {
        let magic = u32::from_be_bytes([payload[4], payload[5], payload[6], payload[7]]);
        if magic == 0x0003_0000 {
            return Some((L7Protocol::PostgreSQL, vec!["PostgreSQL startup message".into()]));
        }
    }

    // MongoDB wire protocol: opcodes OP_QUERY=2004, OP_MSG=2013
    if payload.len() >= 16 {
        let opcode = i32::from_le_bytes([payload[12], payload[13], payload[14], payload[15]]);
        if opcode == 2004 || opcode == 2013 || opcode == 2001 {
            return Some((L7Protocol::MongoDB, vec![format!("MongoDB opcode={opcode}")]));
        }
    }

    // Kafka request: ApiKey in [0,80], ApiVersion in [0,12], followed by correlation_id
    if payload.len() >= 8 {
        let api_key = i16::from_be_bytes([payload[0], payload[1]]);
        let api_ver = i16::from_be_bytes([payload[2], payload[3]]);
        if api_key >= 0 && api_key <= 80 && api_ver >= 0 && api_ver <= 12 {
            return Some((
                L7Protocol::Kafka,
                vec![format!("Kafka ApiKey={api_key} ApiVersion={api_ver}")],
            ));
        }
    }

    // NATS: client protocol starts with "CONNECT"
    if payload.starts_with(b"CONNECT ") || payload.starts_with(b"INFO ") {
        return Some((L7Protocol::NATS, vec!["NATS client/server greeting".into()]));
    }

    // SMB: NetBIOS session or raw SMB
    if payload.len() >= 4 && (&payload[0..4] == b"\x00\x00\x00\x00" || &payload[4..8] == b"\xffSMB") {
        return Some((L7Protocol::SMB, vec!["SMB protocol magic".into()]));
    }
    if payload.len() >= 4 && &payload[0..4] == b"\xfeSMB" {
        return Some((L7Protocol::SMB, vec!["SMB2/3 protocol magic".into()]));
    }

    // RDP: x.224 Connection Request TPDU (0x0e,0xe0)
    if payload.len() >= 11 && payload[5] == 0xe0 {
        return Some((L7Protocol::RDP, vec!["RDP x.224 Connection Request".into()]));
    }

    None
}

// ── L7Classifier ─────────────────────────────────────────────────────────────

/// L7 application-protocol classifier.
///
/// Uses heuristic payload signatures and port mappings.
/// Optionally integrates nDPI for high-confidence deep packet inspection.
pub struct L7Classifier {
    /// Maximum payload bytes to inspect.
    max_inspect_bytes: usize,
}

impl Default for L7Classifier {
    fn default() -> Self {
        Self::new()
    }
}

impl L7Classifier {
    /// Create a classifier that inspects up to 512 bytes of payload.
    pub fn new() -> Self {
        Self { max_inspect_bytes: 512 }
    }

    /// Create a classifier with a custom inspection window.
    pub fn with_max_inspect(max_inspect_bytes: usize) -> Self {
        Self { max_inspect_bytes }
    }

    /// Classify a flow by source port, destination port, and payload bytes.
    ///
    /// Classification priority:
    /// 1. Payload byte signature (confidence 90)
    /// 2. Destination port mapping (confidence 70)
    /// 3. Source port mapping (confidence 60)
    /// 4. Unknown (confidence 0)
    pub fn classify(
        &self,
        src_port: u16,
        dst_port: u16,
        payload: &[u8],
    ) -> ClassificationResult {
        let trimmed = if payload.len() > self.max_inspect_bytes {
            &payload[..self.max_inspect_bytes]
        } else {
            payload
        };

        // ── 1. Payload signature ────────────────────────────────────────────
        if !trimmed.is_empty() {
            if let Some((proto, indicators)) = classify_by_payload(trimmed) {
                return ClassificationResult::new(proto, 90, indicators);
            }
        }

        // ── 2. Destination port ─────────────────────────────────────────────
        if let Some(proto) = port_to_protocol(dst_port) {
            return ClassificationResult::new(
                proto,
                70,
                vec![format!("dst_port={dst_port}")],
            );
        }

        // ── 3. Source port ──────────────────────────────────────────────────
        if let Some(proto) = port_to_protocol(src_port) {
            return ClassificationResult::new(
                proto,
                60,
                vec![format!("src_port={src_port}")],
            );
        }

        ClassificationResult::unknown()
    }

    /// Promote a low-confidence nDPI result into `ClassificationResult`.
    ///
    /// Merges nDPI output (confidence 100 when app_protocol != 0) with the
    /// heuristic fallback.
    pub fn classify_with_ndpi_result(
        &self,
        src_port: u16,
        dst_port: u16,
        payload: &[u8],
        ndpi_proto_name: Option<&str>,
        ndpi_confidence: u8,
    ) -> ClassificationResult {
        if let Some(name) = ndpi_proto_name {
            if ndpi_confidence > 0 {
                // Map common nDPI names to our enum
                let proto = match name.to_uppercase().as_str() {
                    "HTTP" => L7Protocol::HTTP,
                    "HTTPS" | "TLS" | "SSL" => L7Protocol::HTTPS,
                    "DNS" => L7Protocol::DNS,
                    "SSH" => L7Protocol::SSH,
                    "SMB" | "CIFS" => L7Protocol::SMB,
                    "RDP" => L7Protocol::RDP,
                    "SMTP" => L7Protocol::SMTP,
                    "IMAP" => L7Protocol::IMAP,
                    "FTP" | "FTP_DATA" | "FTP_CONTROL" => L7Protocol::FTP,
                    "MYSQL" => L7Protocol::MySQL,
                    "POSTGRES" | "POSTGRESQL" => L7Protocol::PostgreSQL,
                    "REDIS" => L7Protocol::Redis,
                    "MONGODB" => L7Protocol::MongoDB,
                    "KAFKA" => L7Protocol::Kafka,
                    _ => L7Protocol::Unknown(name.to_string()),
                };
                return ClassificationResult::new(
                    proto,
                    ndpi_confidence,
                    vec![format!("nDPI: {name}")],
                );
            }
        }
        self.classify(src_port, dst_port, payload)
    }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    fn clf() -> L7Classifier { L7Classifier::new() }

    #[test]
    fn classify_http_get() {
        let r = clf().classify(54321, 80, b"GET /index.html HTTP/1.1\r\nHost: example.com\r\n\r\n");
        assert_eq!(r.protocol, L7Protocol::HTTP);
        assert_eq!(r.confidence, 90);
        assert!(!r.indicators.is_empty());
    }

    #[test]
    fn classify_http_response() {
        let r = clf().classify(80, 54321, b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n");
        assert_eq!(r.protocol, L7Protocol::HTTP);
    }

    #[test]
    fn classify_tls_client_hello() {
        // Record type 0x16 = TLS, version 0x0303 = TLS 1.2, handshake type 0x01
        let pkt = vec![0x16u8, 0x03, 0x03, 0x00, 0x7a, 0x01, 0x00, 0x00, 0x70];
        let r = clf().classify(12345, 443, &pkt);
        assert_eq!(r.protocol, L7Protocol::HTTPS);
        assert_eq!(r.confidence, 90);
    }

    #[test]
    fn classify_ssh_banner() {
        let r = clf().classify(12345, 22, b"SSH-2.0-OpenSSH_8.9\r\n");
        assert_eq!(r.protocol, L7Protocol::SSH);
        assert_eq!(r.confidence, 90);
    }

    #[test]
    fn classify_smtp_ehlo() {
        let r = clf().classify(12345, 25, b"EHLO mail.example.com\r\n");
        assert_eq!(r.protocol, L7Protocol::SMTP);
    }

    #[test]
    fn classify_redis_resp() {
        let r = clf().classify(12345, 6379, b"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n");
        assert_eq!(r.protocol, L7Protocol::Redis);
    }

    #[test]
    fn classify_postgres_startup() {
        let mut pkt = vec![0u8; 8];
        pkt[0..4].copy_from_slice(&[0, 0, 0, 8]);   // length
        pkt[4..8].copy_from_slice(&[0x00, 0x03, 0x00, 0x00]); // protocol version
        let r = clf().classify(12345, 5432, &pkt);
        assert_eq!(r.protocol, L7Protocol::PostgreSQL);
    }

    #[test]
    fn classify_by_dst_port_fallback() {
        let r = clf().classify(12345, 3306, b"");
        assert_eq!(r.protocol, L7Protocol::MySQL);
        assert_eq!(r.confidence, 70);
        assert!(r.indicators[0].contains("dst_port=3306"));
    }

    #[test]
    fn classify_by_src_port_fallback() {
        let r = clf().classify(27017, 54321, b"");
        assert_eq!(r.protocol, L7Protocol::MongoDB);
        assert_eq!(r.confidence, 60);
    }

    #[test]
    fn classify_unknown_returns_zero_confidence() {
        let r = clf().classify(54321, 54322, b"");
        assert_eq!(r.confidence, 0);
        assert!(matches!(r.protocol, L7Protocol::Unknown(_)));
    }

    #[test]
    fn classify_nats_greeting() {
        let r = clf().classify(12345, 4222, b"INFO {\"server_id\":\"abc\"}\r\n");
        assert_eq!(r.protocol, L7Protocol::NATS);
    }

    #[test]
    fn classify_ftp_greeting() {
        let r = clf().classify(12345, 21, b"220 FTP server ready\r\n");
        assert_eq!(r.protocol, L7Protocol::FTP);
    }

    #[test]
    fn classify_with_ndpi_result_override() {
        let r = clf().classify_with_ndpi_result(
            12345, 443, b"", Some("TLS"), 100
        );
        assert_eq!(r.protocol, L7Protocol::HTTPS);
        assert_eq!(r.confidence, 100);
    }

    #[test]
    fn classify_with_ndpi_zero_confidence_falls_back() {
        let r = clf().classify_with_ndpi_result(
            12345, 22, b"SSH-2.0-OpenSSH_9.0\r\n", Some("SSH"), 0
        );
        assert_eq!(r.protocol, L7Protocol::SSH);
        assert_eq!(r.confidence, 90); // payload signature
    }

    #[test]
    fn l7protocol_display() {
        assert_eq!(L7Protocol::HTTP.to_string(), "HTTP");
        assert_eq!(L7Protocol::Unknown("QUIC".into()).to_string(), "Unknown(QUIC)");
    }

    #[test]
    fn smb2_magic() {
        let mut pkt = vec![0u8; 8];
        pkt[0..4].copy_from_slice(b"\xfeSMB");
        let r = clf().classify(45678, 445, &pkt);
        assert_eq!(r.protocol, L7Protocol::SMB);
        assert_eq!(r.confidence, 90);
    }
}
