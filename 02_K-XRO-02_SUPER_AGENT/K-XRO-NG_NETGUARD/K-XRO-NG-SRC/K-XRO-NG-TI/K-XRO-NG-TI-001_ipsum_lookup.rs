//! K-XRO-NG-TI-001 — IPsum community IP reputation lookup.
//!
//! Integrates the [IPsum](https://github.com/stamparm/ipsum) aggregated
//! IP blocklist — a daily-updated list of IPs sourced from 100+ threat feeds.
//!
//! # Threat level semantics
//! The IPsum file format is `<ip>\t<count>` where `count` is the number of
//! threat-intelligence feeds that have listed this IP.
//!
//! | Count | Interpretation                                   |
//! |-------|--------------------------------------------------|
//! | 1     | Mentioned in one feed — low confidence           |
//! | 2     | Confirmed by two feeds — moderate confidence     |
//! | 3+    | High confidence malicious — `is_malicious()` = true |
//!
//! # Loading
//! - `IpsumBlocklist::load_from_file(path)` — synchronous, reads local file
//! - `IpsumBlocklist::load_from_url(url)` — async, downloads with `reqwest`
//! - `IpsumBlocklist::from_env()` — reads `KUBRIC_IPSUM_PATH` or live URL
//!
//! # Hot path
//! Most lookups happen on the packet capture hot path.  The underlying
//! `HashMap<IpAddr, u8>` is O(1) and allocation-free per lookup.

#![allow(dead_code)]

use std::collections::HashMap;
use std::net::IpAddr;
use std::path::Path;
use std::str::FromStr;
use serde::Serialize;
use tracing::{info, warn};

// ── IpsumEntry ────────────────────────────────────────────────────────────────

/// A single IPsum blocklist entry.
#[derive(Debug, Clone, Serialize)]
pub struct IpsumEntry {
    /// The IP address.
    pub ip: IpAddr,
    /// Threat level = number of feeds listing this IP (1–10+).
    pub threat_level: u8,
}

// ── IpsumBlocklist ────────────────────────────────────────────────────────────

/// In-memory IPsum blocklist for fast per-packet IP reputation lookups.
pub struct IpsumBlocklist {
    entries: HashMap<IpAddr, u8>,
    source_label: String,
}

impl IpsumBlocklist {
    /// Create an empty blocklist.
    pub fn empty() -> Self {
        Self {
            entries: HashMap::new(),
            source_label: "empty".into(),
        }
    }

    /// Load the IPsum file from `path`.
    ///
    /// Format: tab-separated `<ip>\t<count>`, one per line.
    /// Lines starting with `#` or that are empty are skipped.
    pub fn load_from_file(path: &str) -> anyhow::Result<Self> {
        let file_path = Path::new(path);
        if !file_path.exists() {
            warn!(path, "IPsum file not found — returning empty blocklist");
            return Ok(Self::empty());
        }

        let content = std::fs::read_to_string(file_path)
            .map_err(|e| anyhow::anyhow!("read {}: {}", path, e))?;

        let mut entries = HashMap::new();
        for line in content.lines() {
            let line = line.trim();
            if line.is_empty() || line.starts_with('#') {
                continue;
            }
            let mut parts = line.splitn(2, '\t');
            let ip_str = match parts.next() {
                Some(s) => s.trim(),
                None => continue,
            };
            let level_str = match parts.next() {
                Some(s) => s.trim(),
                None => "1", // treat as level 1 if no tab
            };
            if let (Ok(ip), Ok(level)) = (IpAddr::from_str(ip_str), level_str.parse::<u8>()) {
                entries.insert(ip, level);
            }
        }

        info!(entries = entries.len(), path, "IPsum blocklist loaded from file");
        Ok(Self {
            entries,
            source_label: path.to_string(),
        })
    }

    /// Asynchronously download and parse the IPsum file from `url`.
    ///
    /// Typically: `https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt`
    pub async fn load_from_url(url: &str) -> anyhow::Result<Self> {
        let response = reqwest::get(url)
            .await
            .map_err(|e| anyhow::anyhow!("GET {}: {}", url, e))?;

        if !response.status().is_success() {
            anyhow::bail!("GET {}: HTTP {}", url, response.status());
        }

        let body = response
            .text()
            .await
            .map_err(|e| anyhow::anyhow!("read body {}: {}", url, e))?;

        let mut entries = HashMap::new();
        for line in body.lines() {
            let line = line.trim();
            if line.is_empty() || line.starts_with('#') {
                continue;
            }
            let mut parts = line.splitn(2, '\t');
            let ip_str = match parts.next() {
                Some(s) => s.trim(),
                None => continue,
            };
            let level_str = match parts.next() {
                Some(s) => s.trim(),
                None => "1",
            };
            if let (Ok(ip), Ok(level)) = (IpAddr::from_str(ip_str), level_str.parse::<u8>()) {
                entries.insert(ip, level);
            }
        }

        info!(entries = entries.len(), url, "IPsum blocklist loaded from URL");
        Ok(Self {
            entries,
            source_label: url.to_string(),
        })
    }

    /// Load from environment or use the default IPsum GitHub URL.
    ///
    /// `KUBRIC_IPSUM_PATH` → local file path (synchronous load)
    /// `KUBRIC_IPSUM_URL`  → alternative URL
    /// Falls back to the official IPsum master URL.
    pub async fn from_env() -> Self {
        if let Ok(path) = std::env::var("KUBRIC_IPSUM_PATH") {
            match Self::load_from_file(&path) {
                Ok(bl) => return bl,
                Err(e) => warn!(%e, "failed to load IPsum from KUBRIC_IPSUM_PATH"),
            }
        }

        let url = std::env::var("KUBRIC_IPSUM_URL").unwrap_or_else(|_| {
            "https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt".into()
        });

        match Self::load_from_url(&url).await {
            Ok(bl) => bl,
            Err(e) => {
                warn!(%e, "failed to load IPsum from URL — using empty blocklist");
                Self::empty()
            }
        }
    }

    /// Look up a single IP address.
    ///
    /// Returns `Some(threat_level)` if found, `None` if the IP is not listed.
    pub fn lookup(&self, ip: IpAddr) -> Option<u8> {
        self.entries.get(&ip).copied()
    }

    /// Returns `true` if `ip` has a threat level >= 3 (listed in 3+ feeds).
    pub fn is_malicious(&self, ip: IpAddr) -> bool {
        self.lookup(ip).map(|level| level >= 3).unwrap_or(false)
    }

    /// Batch lookup — returns a map of found IPs to their threat levels.
    ///
    /// IPs not in the blocklist are omitted from the result.
    pub fn batch_lookup(&self, ips: &[IpAddr]) -> HashMap<IpAddr, u8> {
        ips.iter()
            .filter_map(|ip| self.lookup(*ip).map(|level| (*ip, level)))
            .collect()
    }

    /// List all IPs with threat_level >= `threshold`.
    pub fn entries_above(&self, threshold: u8) -> Vec<IpsumEntry> {
        self.entries
            .iter()
            .filter(|(_, &level)| level >= threshold)
            .map(|(&ip, &threat_level)| IpsumEntry { ip, threat_level })
            .collect()
    }

    /// Number of entries loaded.
    pub fn count(&self) -> usize {
        self.entries.len()
    }

    /// Source label (file path or URL used to load this blocklist).
    pub fn source(&self) -> &str {
        &self.source_label
    }

    /// Add or update a single entry programmatically (e.g., from a live NATS feed).
    pub fn insert(&mut self, ip: IpAddr, threat_level: u8) {
        self.entries.insert(ip, threat_level);
    }

    /// Remove an entry (e.g., after receiving a TI retraction).
    pub fn remove(&mut self, ip: &IpAddr) -> Option<u8> {
        self.entries.remove(ip)
    }

    /// Merge another blocklist into this one.
    ///
    /// For IPs present in both lists the higher threat level wins.
    pub fn merge(&mut self, other: &IpsumBlocklist) {
        for (ip, &level) in &other.entries {
            let existing = self.entries.entry(*ip).or_insert(0);
            *existing = (*existing).max(level);
        }
    }

    /// Return a deduplicated `Vec<IpsumEntry>` sorted by threat level descending.
    pub fn top_threats(&self, limit: usize) -> Vec<IpsumEntry> {
        let mut entries: Vec<IpsumEntry> = self
            .entries
            .iter()
            .map(|(&ip, &threat_level)| IpsumEntry { ip, threat_level })
            .collect();
        entries.sort_unstable_by(|a, b| b.threat_level.cmp(&a.threat_level));
        entries.truncate(limit);
        entries
    }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    fn make_ip(s: &str) -> IpAddr {
        IpAddr::from_str(s).unwrap()
    }

    #[test]
    fn empty_lookup_returns_none() {
        let bl = IpsumBlocklist::empty();
        assert_eq!(bl.lookup(make_ip("8.8.8.8")), None);
        assert!(!bl.is_malicious(make_ip("8.8.8.8")));
        assert_eq!(bl.count(), 0);
    }

    #[test]
    fn load_from_file_parses_correctly() {
        let dir = std::env::temp_dir().join("ipsum_test");
        let _ = std::fs::create_dir_all(&dir);
        let path = dir.join("ipsum.txt");
        let mut f = std::fs::File::create(&path).unwrap();
        writeln!(f, "# IPsum threat intelligence").unwrap();
        writeln!(f, "1.2.3.4\t5").unwrap();
        writeln!(f, "5.6.7.8\t1").unwrap();
        writeln!(f, "2001:db8::1\t3").unwrap();
        writeln!(f, "# comment line").unwrap();
        writeln!(f, "").unwrap();
        drop(f);

        let bl = IpsumBlocklist::load_from_file(path.to_str().unwrap()).unwrap();
        assert_eq!(bl.count(), 3);
        assert_eq!(bl.lookup(make_ip("1.2.3.4")), Some(5));
        assert_eq!(bl.lookup(make_ip("5.6.7.8")), Some(1));
        assert!(bl.is_malicious(make_ip("1.2.3.4")));
        assert!(!bl.is_malicious(make_ip("5.6.7.8")));
        assert!(bl.is_malicious(make_ip("2001:db8::1")));

        let _ = std::fs::remove_dir_all(&dir);
    }

    #[test]
    fn load_from_missing_file_returns_empty() {
        let bl = IpsumBlocklist::load_from_file("/nonexistent/ipsum.txt").unwrap();
        assert_eq!(bl.count(), 0);
    }

    #[test]
    fn batch_lookup_only_returns_found_ips() {
        let mut bl = IpsumBlocklist::empty();
        bl.insert(make_ip("10.0.0.1"), 4);
        bl.insert(make_ip("10.0.0.2"), 2);

        let ips = vec![
            make_ip("10.0.0.1"),
            make_ip("10.0.0.2"),
            make_ip("10.0.0.3"),
        ];
        let result = bl.batch_lookup(&ips);
        assert_eq!(result.len(), 2);
        assert_eq!(result[&make_ip("10.0.0.1")], 4);
        assert_eq!(result[&make_ip("10.0.0.2")], 2);
        assert!(!result.contains_key(&make_ip("10.0.0.3")));
    }

    #[test]
    fn entries_above_threshold() {
        let mut bl = IpsumBlocklist::empty();
        bl.insert(make_ip("1.1.1.1"), 1);
        bl.insert(make_ip("2.2.2.2"), 3);
        bl.insert(make_ip("3.3.3.3"), 5);

        let high = bl.entries_above(3);
        assert_eq!(high.len(), 2);
        let ips: Vec<_> = high.iter().map(|e| e.ip).collect();
        assert!(ips.contains(&make_ip("2.2.2.2")));
        assert!(ips.contains(&make_ip("3.3.3.3")));
    }

    #[test]
    fn merge_takes_higher_level() {
        let mut a = IpsumBlocklist::empty();
        a.insert(make_ip("1.1.1.1"), 2);
        a.insert(make_ip("2.2.2.2"), 4);

        let mut b = IpsumBlocklist::empty();
        b.insert(make_ip("1.1.1.1"), 5);
        b.insert(make_ip("3.3.3.3"), 1);

        a.merge(&b);
        assert_eq!(a.lookup(make_ip("1.1.1.1")), Some(5)); // took higher
        assert_eq!(a.lookup(make_ip("2.2.2.2")), Some(4)); // unchanged
        assert_eq!(a.lookup(make_ip("3.3.3.3")), Some(1)); // added
        assert_eq!(a.count(), 3);
    }

    #[test]
    fn remove_entry() {
        let mut bl = IpsumBlocklist::empty();
        bl.insert(make_ip("1.1.1.1"), 5);
        assert_eq!(bl.remove(&make_ip("1.1.1.1")), Some(5));
        assert_eq!(bl.lookup(make_ip("1.1.1.1")), None);
    }

    #[test]
    fn top_threats_sorted() {
        let mut bl = IpsumBlocklist::empty();
        bl.insert(make_ip("1.1.1.1"), 3);
        bl.insert(make_ip("2.2.2.2"), 9);
        bl.insert(make_ip("3.3.3.3"), 1);

        let top = bl.top_threats(2);
        assert_eq!(top.len(), 2);
        assert_eq!(top[0].threat_level, 9);
        assert_eq!(top[1].threat_level, 3);
    }

    #[test]
    fn source_label_preserved() {
        let bl = IpsumBlocklist::load_from_file("/tmp/nonexistent").unwrap();
        assert_eq!(bl.source(), "empty");
    }
}
