use regex::Regex;
use tracing::{info, warn};

#[derive(Debug, Clone)]
struct WafRule {
    id: String,
    msg: String,
    regex: Regex,
}

pub struct WafEngine {
    rules: Vec<WafRule>,
}

impl WafEngine {
    pub fn empty() -> Self {
        Self { rules: Vec::new() }
    }

    pub fn load_from_dir(dir: &str) -> Self {
        let root = std::path::Path::new(dir);
        if !root.exists() {
            info!(dir, "WAF CRS directory not found; WAF disabled");
            return Self::empty();
        }

        let mut rules = Vec::new();
        load_rules(root, &mut rules);
        info!(rules = rules.len(), dir, "WAF rules loaded");
        Self { rules }
    }

    pub fn inspect_http_payload(&self, payload: &[u8]) -> Vec<(String, String)> {
        if self.rules.is_empty() || payload.is_empty() {
            return Vec::new();
        }
        let text = match std::str::from_utf8(payload) {
            Ok(v) => v,
            Err(_) => return Vec::new(),
        };

        let mut out = Vec::new();
        for rule in &self.rules {
            if rule.regex.is_match(text) {
                out.push((rule.id.clone(), rule.msg.clone()));
            }
        }
        out
    }
}

fn load_rules(dir: &std::path::Path, out: &mut Vec<WafRule>) {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return,
    };

    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            load_rules(&p, out);
            continue;
        }
        if p.extension().and_then(|e| e.to_str()) != Some("conf") {
            continue;
        }
        if let Ok(content) = std::fs::read_to_string(&p) {
            for line in content.lines() {
                if let Some(rule) = parse_seclang_rule(line) {
                    out.push(rule);
                }
            }
        }
    }
}

fn parse_seclang_rule(line: &str) -> Option<WafRule> {
    let line = line.trim();
    if !line.starts_with("SecRule") {
        return None;
    }

    let rx_idx = line.find("@rx ")?;
    let after = &line[rx_idx + 4..];
    let end_quote = after.find('"')?;
    let pattern = after[..end_quote].trim();
    if pattern.is_empty() {
        return None;
    }

    let id = extract_action_value(line, "id").unwrap_or_else(|| "unknown".to_string());
    let msg = extract_action_value(line, "msg").unwrap_or_else(|| "WAF Rule".to_string());

    match Regex::new(pattern) {
        Ok(regex) => Some(WafRule { id, msg, regex }),
        Err(e) => {
            warn!(error = %e, pattern, "invalid CRS regex; skipping rule");
            None
        }
    }
}

fn extract_action_value(line: &str, key: &str) -> Option<String> {
    let k = format!("{key}:");
    let pos = line.find(&k)?;
    let rest = &line[pos + k.len()..];

    if let Some(stripped) = rest.strip_prefix('"') {
        let end = stripped.find('"')?;
        return Some(stripped[..end].to_string());
    }

    if let Some(stripped) = rest.strip_prefix('\'') {
        let end = stripped.find('\'')?;
        return Some(stripped[..end].to_string());
    }

    let end = rest.find(',').unwrap_or(rest.len());
    Some(rest[..end].trim().to_string())
}
