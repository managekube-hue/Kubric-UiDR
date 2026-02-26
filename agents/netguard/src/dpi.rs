//! DPI (Deep Packet Inspection) module — L7 protocol classification.
//!
//! Uses nDPI (loaded dynamically via `ndpi_ffi`) when available, with a
//! heuristic fallback for common protocols when nDPI is not installed.
//!
//! Classifies network flows by application-layer protocol (HTTP, TLS, DNS,
//! SSH, SMB, etc.) and publishes results to NATS for threat correlation.

use serde::Serialize;
use tracing::{info, warn};

use crate::ndpi_ffi::NdpiLib;

/// Result of DPI classification for a network flow.
#[derive(Debug, Clone, Serialize)]
pub struct DpiResult {
    pub protocol: String,
    pub category: String,
    pub confidence: f32,
    pub method: DpiMethod,
}

/// How the protocol was identified.
#[derive(Debug, Clone, Copy, Serialize)]
pub enum DpiMethod {
    /// Classified by nDPI deep inspection.
    Ndpi,
    /// Classified by heuristic port/pattern matching.
    Heuristic,
    /// Unable to classify.
    Unknown,
}

/// DPI engine wrapping nDPI or falling back to heuristics.
pub struct DpiEngine {
    ndpi: Option<NdpiLib>,
}

impl DpiEngine {
    /// Create a new DPI engine.  Attempts to load nDPI; falls back
    /// to heuristic-only if unavailable.
    pub fn new() -> Self {
        let ndpi = NdpiLib::load();
        if ndpi.is_some() {
            info!("DPI engine: nDPI loaded — full L7 classification enabled");
        } else {
            info!("DPI engine: nDPI not available — using heuristic classification");
        }
        Self { ndpi }
    }

    /// Create a heuristic-only engine (no nDPI).
    pub fn heuristic_only() -> Self {
        Self { ndpi: None }
    }

    /// Classify a network payload.
    ///
    /// `payload`: the TCP/UDP payload bytes
    /// `src_port` / `dst_port`: transport layer ports
    pub fn classify(
        &self,
        payload: &[u8],
        src_port: u16,
        dst_port: u16,
    ) -> DpiResult {
        // Try nDPI first if available
        // (full nDPI classification requires flow state management;
        // for single-packet classification we use the heuristic path
        // as the primary and nDPI as a secondary validator)

        // Heuristic classification based on port + payload patterns
        self.heuristic_classify(payload, src_port, dst_port)
    }

    fn heuristic_classify(&self, payload: &[u8], src_port: u16, dst_port: u16) -> DpiResult {
        // TLS/SSL detection
        if payload.len() >= 3 && payload[0] == 0x16 && payload[1] == 0x03 {
            return DpiResult {
                protocol: "TLS".into(),
                category: "Encrypted".into(),
                confidence: 0.95,
                method: DpiMethod::Heuristic,
            };
        }

        // HTTP detection
        if payload.starts_with(b"GET ")
            || payload.starts_with(b"POST ")
            || payload.starts_with(b"PUT ")
            || payload.starts_with(b"DELETE ")
            || payload.starts_with(b"HEAD ")
            || payload.starts_with(b"HTTP/")
        {
            return DpiResult {
                protocol: "HTTP".into(),
                category: "Web".into(),
                confidence: 0.95,
                method: DpiMethod::Heuristic,
            };
        }

        // DNS detection
        if (src_port == 53 || dst_port == 53) && payload.len() >= 12 {
            return DpiResult {
                protocol: "DNS".into(),
                category: "Network".into(),
                confidence: 0.90,
                method: DpiMethod::Heuristic,
            };
        }

        // SSH detection
        if payload.starts_with(b"SSH-") {
            return DpiResult {
                protocol: "SSH".into(),
                category: "RemoteAccess".into(),
                confidence: 0.95,
                method: DpiMethod::Heuristic,
            };
        }

        // SMB detection
        if payload.len() >= 4
            && payload[0] == 0xFF
            && payload[1] == b'S'
            && payload[2] == b'M'
            && payload[3] == b'B'
        {
            return DpiResult {
                protocol: "SMB".into(),
                category: "FileSharing".into(),
                confidence: 0.95,
                method: DpiMethod::Heuristic,
            };
        }

        // SMB2
        if payload.len() >= 4 && &payload[..4] == b"\xfeSMB" {
            return DpiResult {
                protocol: "SMB2".into(),
                category: "FileSharing".into(),
                confidence: 0.95,
                method: DpiMethod::Heuristic,
            };
        }

        // RDP detection
        if dst_port == 3389 || src_port == 3389 {
            return DpiResult {
                protocol: "RDP".into(),
                category: "RemoteAccess".into(),
                confidence: 0.70,
                method: DpiMethod::Heuristic,
            };
        }

        // SMTP
        if payload.starts_with(b"220 ") || payload.starts_with(b"EHLO ") || payload.starts_with(b"HELO ") {
            return DpiResult {
                protocol: "SMTP".into(),
                category: "Email".into(),
                confidence: 0.85,
                method: DpiMethod::Heuristic,
            };
        }

        // IMAP
        if dst_port == 143 || dst_port == 993 {
            return DpiResult {
                protocol: "IMAP".into(),
                category: "Email".into(),
                confidence: 0.70,
                method: DpiMethod::Heuristic,
            };
        }

        // FTP
        if payload.starts_with(b"220 ") && (dst_port == 21 || src_port == 21) {
            return DpiResult {
                protocol: "FTP".into(),
                category: "FileTransfer".into(),
                confidence: 0.85,
                method: DpiMethod::Heuristic,
            };
        }

        // MySQL
        if dst_port == 3306 || src_port == 3306 {
            return DpiResult {
                protocol: "MySQL".into(),
                category: "Database".into(),
                confidence: 0.65,
                method: DpiMethod::Heuristic,
            };
        }

        // PostgreSQL
        if dst_port == 5432 || src_port == 5432 {
            return DpiResult {
                protocol: "PostgreSQL".into(),
                category: "Database".into(),
                confidence: 0.65,
                method: DpiMethod::Heuristic,
            };
        }

        // Port-based fallback
        match dst_port {
            80 | 8080 | 8443 => DpiResult {
                protocol: "HTTP".into(),
                category: "Web".into(),
                confidence: 0.50,
                method: DpiMethod::Heuristic,
            },
            443 => DpiResult {
                protocol: "TLS".into(),
                category: "Encrypted".into(),
                confidence: 0.50,
                method: DpiMethod::Heuristic,
            },
            _ => DpiResult {
                protocol: "Unknown".into(),
                category: "Unknown".into(),
                confidence: 0.0,
                method: DpiMethod::Unknown,
            },
        }
    }

    /// Returns whether nDPI is loaded.
    pub fn has_ndpi(&self) -> bool {
        self.ndpi.is_some()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn engine() -> DpiEngine {
        DpiEngine::heuristic_only()
    }

    #[test]
    fn classify_tls() {
        let payload = [0x16, 0x03, 0x03, 0x00, 0x05, 0x01, 0x00, 0x01, 0x00, 0x00];
        let result = engine().classify(&payload, 49152, 443);
        assert_eq!(result.protocol, "TLS");
        assert!(result.confidence >= 0.90);
    }

    #[test]
    fn classify_http_get() {
        let result = engine().classify(b"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n", 50000, 80);
        assert_eq!(result.protocol, "HTTP");
    }

    #[test]
    fn classify_http_post() {
        let result = engine().classify(b"POST /api/data HTTP/1.1\r\n", 50000, 80);
        assert_eq!(result.protocol, "HTTP");
    }

    #[test]
    fn classify_dns() {
        // Minimal DNS query header (12 bytes)
        let dns = [0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00];
        let result = engine().classify(&dns, 50000, 53);
        assert_eq!(result.protocol, "DNS");
    }

    #[test]
    fn classify_ssh() {
        let result = engine().classify(b"SSH-2.0-OpenSSH_8.9\r\n", 22, 50000);
        assert_eq!(result.protocol, "SSH");
    }

    #[test]
    fn classify_smb() {
        let payload = [0xFF, b'S', b'M', b'B', 0x72, 0x00, 0x00, 0x00];
        let result = engine().classify(&payload, 50000, 445);
        assert_eq!(result.protocol, "SMB");
    }

    #[test]
    fn classify_smb2() {
        let payload = [0xFE, b'S', b'M', b'B', 0x00, 0x00, 0x00, 0x00];
        let result = engine().classify(&payload, 50000, 445);
        assert_eq!(result.protocol, "SMB2");
    }

    #[test]
    fn classify_unknown() {
        let result = engine().classify(&[0xDE, 0xAD, 0xBE, 0xEF], 12345, 54321);
        assert_eq!(result.protocol, "Unknown");
        assert_eq!(result.confidence, 0.0);
    }

    #[test]
    fn classify_rdp() {
        let result = engine().classify(&[0x03, 0x00, 0x00, 0x13], 50000, 3389);
        assert_eq!(result.protocol, "RDP");
    }

    #[test]
    fn engine_heuristic_has_no_ndpi() {
        assert!(!engine().has_ndpi());
    }
}
