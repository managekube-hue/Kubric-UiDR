//! RITA HTTP client — communicates with the RITA beacon analysis sidecar.
//!
//! RITA is GPL 3.0 — this module communicates via HTTP ONLY.
//! **No imports from `github.com/activecm/rita` are permitted.**
//!
//! RITA exposes a REST API on port 4096 (configurable via KUBRIC_RITA_URL).
//! This client fetches beacon scores, DNS tunneling indicators, and data
//! exfiltration signals.

use serde::{Deserialize, Serialize};
use tracing::{info, warn};

/// RITA beacon detection result.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Beacon {
    pub src_ip: String,
    pub dst_ip: String,
    pub score: f64,
    pub connections: u64,
    pub avg_bytes: u64,
    pub ts_score: f64,
    pub ds_score: f64,
    pub dur_score: f64,
}

/// RITA DNS tunneling detection result.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DnsTunnel {
    pub fqdn: String,
    pub src_ip: String,
    pub score: f64,
    pub query_count: u64,
    pub unique_subdomains: u64,
}

/// RITA long-connection / data exfil detection result.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LongConnection {
    pub src_ip: String,
    pub dst_ip: String,
    pub duration_secs: u64,
    pub bytes_sent: u64,
    pub bytes_received: u64,
}

/// HTTP client for the RITA sidecar.
pub struct RitaClient {
    base_url: String,
    client: reqwest::Client,
}

impl RitaClient {
    /// Create a new RITA client.
    pub fn new(base_url: &str) -> Self {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_secs(10))
            .build()
            .unwrap_or_default();
        info!(url = base_url, "RITA client initialised");
        Self {
            base_url: base_url.trim_end_matches('/').to_string(),
            client,
        }
    }

    /// Create from environment variable KUBRIC_RITA_URL (default: http://rita:4096).
    pub fn from_env() -> Self {
        let url = std::env::var("KUBRIC_RITA_URL")
            .unwrap_or_else(|_| "http://rita:4096".to_string());
        Self::new(&url)
    }

    /// Fetch beacons for a database/tenant.
    pub async fn get_beacons(&self, database: &str) -> Vec<Beacon> {
        let url = format!("{}/api/v1/{}/beacons", self.base_url, database);
        match self.client.get(&url).send().await {
            Ok(resp) => {
                if resp.status().is_success() {
                    match resp.json::<Vec<Beacon>>().await {
                        Ok(beacons) => {
                            info!(count = beacons.len(), db = database, "RITA beacons fetched");
                            beacons
                        }
                        Err(e) => {
                            warn!(%e, "RITA beacon parse failed");
                            Vec::new()
                        }
                    }
                } else {
                    warn!(status = %resp.status(), "RITA beacons request failed");
                    Vec::new()
                }
            }
            Err(e) => {
                warn!(%e, "RITA beacons request error");
                Vec::new()
            }
        }
    }

    /// Fetch DNS tunneling indicators.
    pub async fn get_dns_tunneling(&self, database: &str) -> Vec<DnsTunnel> {
        let url = format!("{}/api/v1/{}/dns/tunneling", self.base_url, database);
        match self.client.get(&url).send().await {
            Ok(resp) if resp.status().is_success() => {
                resp.json().await.unwrap_or_default()
            }
            Ok(resp) => {
                warn!(status = %resp.status(), "RITA DNS tunneling request failed");
                Vec::new()
            }
            Err(e) => {
                warn!(%e, "RITA DNS tunneling request error");
                Vec::new()
            }
        }
    }

    /// Fetch long connections (potential data exfiltration).
    pub async fn get_long_connections(&self, database: &str) -> Vec<LongConnection> {
        let url = format!("{}/api/v1/{}/long-connections", self.base_url, database);
        match self.client.get(&url).send().await {
            Ok(resp) if resp.status().is_success() => {
                resp.json().await.unwrap_or_default()
            }
            Ok(resp) => {
                warn!(status = %resp.status(), "RITA long connections request failed");
                Vec::new()
            }
            Err(e) => {
                warn!(%e, "RITA long connections request error");
                Vec::new()
            }
        }
    }

    /// Health check — returns true if RITA is reachable.
    pub async fn health_check(&self) -> bool {
        let url = format!("{}/api/v1/health", self.base_url);
        matches!(self.client.get(&url).send().await, Ok(r) if r.status().is_success())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn beacon_serializes() {
        let b = Beacon {
            src_ip: "10.0.0.1".into(),
            dst_ip: "185.100.87.42".into(),
            score: 0.95,
            connections: 1440,
            avg_bytes: 256,
            ts_score: 0.98,
            ds_score: 0.92,
            dur_score: 0.95,
        };
        let json = serde_json::to_string(&b).unwrap();
        assert!(json.contains("0.95"));
        assert!(json.contains("185.100.87.42"));
    }

    #[test]
    fn dns_tunnel_serializes() {
        let t = DnsTunnel {
            fqdn: "data.evil.com".into(),
            src_ip: "10.0.0.5".into(),
            score: 0.88,
            query_count: 5000,
            unique_subdomains: 2500,
        };
        let json = serde_json::to_string(&t).unwrap();
        assert!(json.contains("data.evil.com"));
    }

    #[test]
    fn client_from_env_default() {
        // Without KUBRIC_RITA_URL set, defaults to http://rita:4096
        let client = RitaClient::from_env();
        assert!(client.base_url.contains("rita"));
    }

    #[test]
    fn client_custom_url() {
        let client = RitaClient::new("http://localhost:4096/");
        // Trailing slash should be stripped
        assert_eq!(client.base_url, "http://localhost:4096");
    }
}
