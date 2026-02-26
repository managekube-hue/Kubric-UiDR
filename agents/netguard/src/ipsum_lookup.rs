//! IP reputation lookup — checks IP addresses against threat intelligence
//! feeds for known malicious indicators.
//!
//! Supports:
//!   - IPsum aggregated blocklist (https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt)
//!   - Custom tenant-specific blocklists loaded from file
//!   - NATS-based live updates from the KAI TI feed
//!
//! The lookup is in-memory for fast path checking during packet capture.

use std::collections::{HashMap, HashSet};
use std::net::IpAddr;
use std::path::Path;
use std::str::FromStr;
use serde::Serialize;
use tracing::{info, warn};

/// Threat intelligence match result.
#[derive(Debug, Clone, Serialize)]
pub struct IpReputation {
    pub ip: String,
    pub is_malicious: bool,
    pub source: String,
    pub threat_score: u8,
    pub lists_matched: Vec<String>,
}

/// In-memory IP reputation database.
pub struct IpsumLookup {
    /// IP -> (threat_level, sources)
    ips: HashMap<IpAddr, (u8, Vec<String>)>,
}

impl IpsumLookup {
    /// Create an empty lookup table.
    pub fn empty() -> Self {
        Self {
            ips: HashMap::new(),
        }
    }

    /// Load the IPsum blocklist file.
    ///
    /// Format: `ip\tthreat_level` (one per line, # comments)
    /// The threat_level is 1-10 indicating how many feeds include this IP.
    pub fn load_from_file(path: &str) -> Self {
        let mut ips = HashMap::new();

        let file_path = Path::new(path);
        if !file_path.exists() {
            warn!(path, "IPsum file not found — IP reputation disabled");
            return Self { ips };
        }

        match std::fs::read_to_string(file_path) {
            Ok(content) => {
                for line in content.lines() {
                    let line = line.trim();
                    if line.is_empty() || line.starts_with('#') {
                        continue;
                    }
                    let parts: Vec<&str> = line.split('\t').collect();
                    if parts.len() >= 2 {
                        if let (Ok(ip), Ok(level)) = (
                            IpAddr::from_str(parts[0]),
                            parts[1].parse::<u8>(),
                        ) {
                            ips.insert(ip, (level, vec!["ipsum".into()]));
                        }
                    }
                }
                info!(entries = ips.len(), path, "IPsum blocklist loaded");
            }
            Err(e) => {
                warn!(path, %e, "failed to read IPsum file");
            }
        }

        Self { ips }
    }

    /// Load from environment variable KUBRIC_IPSUM_PATH.
    pub fn from_env() -> Self {
        let path = std::env::var("KUBRIC_IPSUM_PATH")
            .unwrap_or_else(|_| "vendor/ipsum/ipsum.txt".to_string());
        Self::load_from_file(&path)
    }

    /// Add IPs from a custom blocklist (e.g., tenant-specific).
    pub fn add_blocklist(&mut self, name: &str, ips: &[&str]) {
        for ip_str in ips {
            if let Ok(ip) = IpAddr::from_str(ip_str) {
                let entry = self.ips.entry(ip).or_insert((0, Vec::new()));
                entry.0 = entry.0.saturating_add(1);
                entry.1.push(name.to_string());
            }
        }
    }

    /// Check an IP address against the reputation database.
    pub fn lookup(&self, ip_str: &str) -> IpReputation {
        let ip = match IpAddr::from_str(ip_str) {
            Ok(ip) => ip,
            Err(_) => {
                return IpReputation {
                    ip: ip_str.to_string(),
                    is_malicious: false,
                    source: "invalid_ip".into(),
                    threat_score: 0,
                    lists_matched: Vec::new(),
                };
            }
        };

        match self.ips.get(&ip) {
            Some((level, sources)) => IpReputation {
                ip: ip_str.to_string(),
                is_malicious: *level >= 2, // Present in 2+ feeds
                source: sources.join(","),
                threat_score: *level,
                lists_matched: sources.clone(),
            },
            None => IpReputation {
                ip: ip_str.to_string(),
                is_malicious: false,
                source: "clean".into(),
                threat_score: 0,
                lists_matched: Vec::new(),
            },
        }
    }

    /// Batch lookup for multiple IPs.
    pub fn lookup_batch(&self, ips: &[&str]) -> Vec<IpReputation> {
        ips.iter().map(|ip| self.lookup(ip)).collect()
    }

    /// Returns the number of loaded IPs.
    pub fn count(&self) -> usize {
        self.ips.len()
    }

    /// Get all IPs with threat_score >= threshold.
    pub fn get_threats_above(&self, threshold: u8) -> Vec<(String, u8)> {
        self.ips
            .iter()
            .filter(|(_, (level, _))| *level >= threshold)
            .map(|(ip, (level, _))| (ip.to_string(), *level))
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn empty_lookup_returns_clean() {
        let db = IpsumLookup::empty();
        let result = db.lookup("8.8.8.8");
        assert!(!result.is_malicious);
        assert_eq!(result.threat_score, 0);
    }

    #[test]
    fn add_blocklist_and_lookup() {
        let mut db = IpsumLookup::empty();
        db.add_blocklist("custom", &["1.2.3.4", "5.6.7.8"]);
        db.add_blocklist("another", &["1.2.3.4"]);

        let result = db.lookup("1.2.3.4");
        assert!(result.is_malicious); // present in 2 lists
        assert_eq!(result.threat_score, 2);
        assert_eq!(result.lists_matched.len(), 2);

        let clean = db.lookup("10.0.0.1");
        assert!(!clean.is_malicious);
    }

    #[test]
    fn invalid_ip_returns_clean() {
        let db = IpsumLookup::empty();
        let result = db.lookup("not-an-ip");
        assert!(!result.is_malicious);
        assert_eq!(result.source, "invalid_ip");
    }

    #[test]
    fn batch_lookup() {
        let mut db = IpsumLookup::empty();
        db.add_blocklist("test", &["1.2.3.4"]);

        let results = db.lookup_batch(&["1.2.3.4", "8.8.8.8"]);
        assert_eq!(results.len(), 2);
        assert_eq!(results[0].threat_score, 1);
        assert_eq!(results[1].threat_score, 0);
    }

    #[test]
    fn count_entries() {
        let mut db = IpsumLookup::empty();
        assert_eq!(db.count(), 0);
        db.add_blocklist("test", &["1.2.3.4", "5.6.7.8", "9.10.11.12"]);
        assert_eq!(db.count(), 3);
    }

    #[test]
    fn threats_above_threshold() {
        let mut db = IpsumLookup::empty();
        db.add_blocklist("a", &["1.2.3.4", "5.6.7.8"]);
        db.add_blocklist("b", &["1.2.3.4"]);
        db.add_blocklist("c", &["1.2.3.4"]);

        let high_threats = db.get_threats_above(3);
        assert_eq!(high_threats.len(), 1);
        assert_eq!(high_threats[0].1, 3);
    }

    #[test]
    fn load_from_missing_file() {
        let db = IpsumLookup::load_from_file("/nonexistent/ipsum.txt");
        assert_eq!(db.count(), 0);
    }
}
