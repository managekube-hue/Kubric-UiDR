//! NetGuard IDS — loads YARA rules for network payload scanning and publishes alerts.
//!
//! This module loads YARA rules from the vendor directory and scans TCP/UDP
//! payloads for malicious patterns.  Alerts are published to NATS on
//! `kubric.{tenant_id}.detection.network_ids.v1`.

use serde::Serialize;
use tracing::{info, warn};

/// A network IDS match result.
#[derive(Debug, Clone, Serialize)]
pub struct IdsAlert {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: u64,
    pub rule_id: String,
    pub rule_name: String,
    pub src_ip: String,
    pub dst_ip: String,
    pub src_port: u32,
    pub dst_port: u32,
    pub protocol: String,
    pub severity: String,
    pub class_uid: u32,
}

impl IdsAlert {
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{}.detection.network_ids.v1", tenant_id)
    }
}

/// Network IDS engine backed by YARA-X rules.
pub struct IdsEngine {
    rules: yara_x::Rules,
    count: usize,
}

impl IdsEngine {
    /// Create an empty engine (no rules loaded).
    pub fn empty() -> Self {
        let compiler = yara_x::Compiler::new();
        Self {
            rules: compiler.build(),
            count: 0,
        }
    }

    /// Load YARA rules from a directory, intended for network-oriented rules.
    pub fn load_from_dir(dir: &str) -> anyhow::Result<Self> {
        let path = std::path::Path::new(dir);
        if !path.exists() {
            info!(dir, "IDS rule directory not found — using empty engine");
            return Ok(Self::empty());
        }

        let mut compiler = yara_x::Compiler::new();
        let mut count = 0;
        load_dir(path, &mut compiler, &mut count);

        info!(rules = count, dir, "IDS YARA rules loaded");
        Ok(Self {
            rules: compiler.build(),
            count,
        })
    }

    /// Scan a network payload and return matching rule identifiers.
    pub fn scan(&self, data: &[u8]) -> Vec<(String, String)> {
        let mut scanner = yara_x::Scanner::new(&self.rules);
        let results = match scanner.scan(data) {
            Ok(r) => r,
            Err(_) => return Vec::new(),
        };
        results
            .matching_rules()
            .map(|rule| {
                (
                    rule.identifier().to_string(),
                    rule.namespace().to_string(),
                )
            })
            .collect()
    }

    pub fn rule_count(&self) -> usize {
        self.count
    }
}

fn load_dir(path: &std::path::Path, compiler: &mut yara_x::Compiler, count: &mut usize) {
    let entries = match std::fs::read_dir(path) {
        Ok(e) => e,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            load_dir(&p, compiler, count);
        } else if let Some(ext) = p.extension() {
            let ext = ext.to_string_lossy().to_lowercase();
            if ext == "yar" || ext == "yara" {
                match std::fs::read_to_string(&p) {
                    Ok(src) => {
                        if compiler.add_source(src.as_str()).is_ok() {
                            *count += 1;
                        } else {
                            warn!(file = %p.display(), "IDS YARA compile error — skipping");
                        }
                    }
                    Err(_) => {}
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ids_engine_empty_scans_cleanly() {
        let engine = IdsEngine::empty();
        assert_eq!(engine.rule_count(), 0);
        let hits = engine.scan(b"GET /malware HTTP/1.1");
        assert!(hits.is_empty());
    }

    #[test]
    fn ids_engine_loads_missing_dir() {
        let engine = IdsEngine::load_from_dir("/nonexistent/ids/rules").unwrap();
        assert_eq!(engine.rule_count(), 0);
    }
}
