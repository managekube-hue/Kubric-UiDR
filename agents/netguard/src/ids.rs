//! NetGuard IDS — loads YARA rules for network payload scanning and publishes alerts.
//!
//! This module loads YARA rules from the vendor directory and scans TCP/UDP
//! payloads for malicious patterns.  Alerts are published to NATS on
//! `kubric.{tenant_id}.detection.network_ids.v1`.

use serde::Serialize;
use tracing::{info, warn};

#[derive(Debug, Clone)]
struct SuricataRule {
    sid: String,
    msg: String,
    content_terms: Vec<Vec<u8>>,
}

#[derive(Debug, Clone)]
pub struct SuricataAssetValidation {
    pub missing_files: Vec<String>,
}

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
    suricata_rules: Vec<SuricataRule>,
}

impl IdsEngine {
    /// Create an empty engine (no rules loaded).
    pub fn empty() -> Self {
        let compiler = yara_x::Compiler::new();
        Self {
            rules: compiler.build(),
            count: 0,
            suricata_rules: Vec::new(),
        }
    }

    /// Load YARA rules from a directory, intended for network-oriented rules.
    pub fn load_from_dir(dir: &str) -> anyhow::Result<Self> {
        Self::load_from_dirs(dir, dir)
    }

    /// Load YARA and Suricata rules from their respective directories.
    pub fn load_from_dirs(yara_dir: &str, suricata_dir: &str) -> anyhow::Result<Self> {
        let path = std::path::Path::new(yara_dir);
        if !path.exists() {
            info!(dir = yara_dir, "IDS YARA rule directory not found — using empty engine");
            return Ok(Self::empty());
        }

        let mut compiler = yara_x::Compiler::new();
        let mut count = 0;
        load_dir(path, &mut compiler, &mut count);

        let mut suricata_rules = Vec::new();
        load_suricata_rules(std::path::Path::new(suricata_dir), &mut suricata_rules);

        info!(yara_rules = count, suricata_rules = suricata_rules.len(), yara_dir, suricata_dir, "IDS rules loaded");
        Ok(Self {
            rules: compiler.build(),
            count,
            suricata_rules,
        })
    }

    /// Scan a network payload and return matching rule identifiers.
    pub fn scan(&self, data: &[u8]) -> Vec<(String, String)> {
        let mut out: Vec<(String, String)> = Vec::new();

        let mut scanner = yara_x::Scanner::new(&self.rules);
        if let Ok(results) = scanner.scan(data) {
            out.extend(results
            .matching_rules()
            .map(|rule| {
                (
                    rule.identifier().to_string(),
                    rule.namespace().to_string(),
                )
            }));
        }

        for rule in &self.suricata_rules {
            if rule.content_terms.is_empty() {
                continue;
            }
            let matched = rule
                .content_terms
                .iter()
                .all(|term| contains_bytes(data, term));
            if matched {
                out.push((format!("{}:{}", rule.sid, rule.msg), "suricata".to_string()));
            }
        }

        out
    }

    pub fn rule_count(&self) -> usize {
        self.count + self.suricata_rules.len()
    }

    pub fn validate_suricata_assets(dir: &str) -> SuricataAssetValidation {
        let root = std::path::Path::new(dir);
        if !root.exists() {
            return SuricataAssetValidation {
                missing_files: vec!["suricata directory missing".to_string()],
            };
        }

        let mut missing = Vec::new();
        let c2 = root.join("emerging-c2.rules");
        if !c2.exists() {
            missing.push("emerging-c2.rules".to_string());
        }

        let glob = std::fs::read_dir(root)
            .ok()
            .map(|it| {
                it.flatten()
                    .map(|e| e.path())
                    .filter(|p| {
                        p.file_name()
                            .and_then(|n| n.to_str())
                            .map(|n| n.starts_with("emerging-") && n.ends_with(".rules"))
                            .unwrap_or(false)
                    })
                    .count()
            })
            .unwrap_or(0);

        if glob == 0 {
            missing.push("emerging-*.rules".to_string());
        }

        SuricataAssetValidation { missing_files: missing }
    }
}

fn contains_bytes(haystack: &[u8], needle: &[u8]) -> bool {
    if needle.is_empty() {
        return true;
    }
    haystack.windows(needle.len()).any(|w| w == needle)
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

fn load_suricata_rules(path: &std::path::Path, out: &mut Vec<SuricataRule>) {
    let entries = match std::fs::read_dir(path) {
        Ok(e) => e,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            load_suricata_rules(&p, out);
        } else if let Some(ext) = p.extension() {
            let ext = ext.to_string_lossy().to_lowercase();
            if ext == "rules" {
                if let Ok(src) = std::fs::read_to_string(&p) {
                    for line in src.lines() {
                        if let Some(rule) = parse_suricata_rule(line) {
                            out.push(rule);
                        }
                    }
                }
            }
        }
    }
}

fn parse_suricata_rule(line: &str) -> Option<SuricataRule> {
    let trimmed = line.trim();
    if trimmed.is_empty() || trimmed.starts_with('#') {
        return None;
    }
    let start = trimmed.find('(')?;
    let end = trimmed.rfind(')')?;
    let body = &trimmed[start + 1..end];

    let mut sid = String::new();
    let mut msg = String::new();
    let mut content_terms: Vec<Vec<u8>> = Vec::new();

    for part in body.split(';') {
        let kv = part.trim();
        if let Some(v) = kv.strip_prefix("sid:") {
            sid = v.trim().to_string();
        } else if let Some(v) = kv.strip_prefix("msg:") {
            msg = v.trim().trim_matches('"').to_string();
        } else if let Some(v) = kv.strip_prefix("content:") {
            let txt = v.trim().trim_matches('"');
            content_terms.push(txt.as_bytes().to_vec());
        }
    }

    if sid.is_empty() || content_terms.is_empty() {
        return None;
    }
    Some(SuricataRule {
        sid,
        msg,
        content_terms,
    })
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
