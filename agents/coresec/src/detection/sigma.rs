#![allow(dead_code)]
//! Native Sigma rule evaluator for CoreSec.
//!
//! Loads all `*.yml` / `*.yaml` files from a directory tree and evaluates
//! OCSF [`ProcessEvent`] fields against each rule at runtime.
//!
//! # Supported detection syntax
//! - Field modifiers: `|contains`, `|startswith`, `|endswith`, `|re`,
//!   `|contains|all`
//! - Condition operators: `and`, `or`, `not`, `1 of <pattern>`,
//!   `all of <pattern>`
//! - Multi-value lists (OR semantics unless `|contains|all`)
//!
//! # sigma-rust 0.8 note
//! The `sigma-rust` crate is not yet published to crates.io.  This module
//! provides a compatible evaluator using `serde_yaml` until an upstream crate
//! stabilises.

use std::collections::HashMap;
use std::fs;
use std::path::Path;

use anyhow::Result;
use regex::Regex;
use serde::Deserialize;
use serde_yaml::Value;
use tracing::warn;

use crate::event::ProcessEvent;

// ── Public types ─────────────────────────────────────────────────────────────

/// A single Sigma rule match returned by [`SigmaEngine::evaluate`].
#[derive(Debug, Clone)]
pub struct SigmaMatch {
    pub rule_id: String,
    pub title:   String,
    pub tags:    Vec<String>,
    pub level:   String,
}

// ── Internal structures ───────────────────────────────────────────────────────

/// One field matcher after parsing a detection group entry.
#[derive(Debug)]
enum FieldModifier {
    Contains,
    ContainsAll,
    StartsWith,
    EndsWith,
    Re,
    Exact,
}

#[derive(Debug)]
struct FieldMatcher {
    field:    String,
    modifier: FieldModifier,
    values:   Vec<String>,
}

#[derive(Debug)]
struct DetectionGroup {
    matchers: Vec<FieldMatcher>,
}

#[derive(Debug)]
struct SigmaRule {
    id:        String,
    title:     String,
    tags:      Vec<String>,
    level:     String,
    groups:    HashMap<String, DetectionGroup>,
    condition: String,
}

// ── YAML deserialization helpers ──────────────────────────────────────────────

/// Loosely-typed Sigma rule file top level.
#[derive(Debug, Deserialize)]
struct RuleFile {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    tags: Vec<String>,
    #[serde(default)]
    level: String,
    /// The `detection:` block.  We keep it as a raw [`Value`] because groups
    /// can have arbitrary names and nested types.
    detection: Option<Value>,
}

fn parse_string_or_list(v: &Value) -> Vec<String> {
    match v {
        Value::String(s) => vec![s.clone()],
        Value::Sequence(seq) => seq
            .iter()
            .filter_map(|x| x.as_str().map(String::from))
            .collect(),
        Value::Bool(b) => vec![b.to_string()],
        Value::Number(n) => vec![n.to_string()],
        _ => vec![],
    }
}

fn parse_detection_group(map: &serde_yaml::Mapping) -> DetectionGroup {
    let mut matchers = Vec::new();
    for (k, v) in map {
        let key = match k.as_str() {
            Some(s) => s.to_string(),
            None => continue,
        };
        // Split key into field + modifiers, e.g. "CommandLine|contains|all"
        let parts: Vec<&str> = key.splitn(4, '|').collect();
        let field = parts[0].to_string();
        let modifier = match parts.get(1).copied() {
            Some("contains") if parts.get(2).copied() == Some("all") => {
                FieldModifier::ContainsAll
            }
            Some("contains") => FieldModifier::Contains,
            Some("startswith") => FieldModifier::StartsWith,
            Some("endswith") => FieldModifier::EndsWith,
            Some("re") => FieldModifier::Re,
            _ => FieldModifier::Exact,
        };
        let values = parse_string_or_list(v);
        matchers.push(FieldMatcher { field, modifier, values });
    }
    DetectionGroup { matchers }
}

fn parse_rule(rf: RuleFile) -> Option<SigmaRule> {
    let detection = rf.detection.as_ref()?;
    let det_map = detection.as_mapping()?;

    let condition = det_map
        .get("condition")
        .and_then(|v| v.as_str())
        .unwrap_or("selection")
        .to_string();

    let mut groups: HashMap<String, DetectionGroup> = HashMap::new();
    for (k, v) in det_map {
        let name = k.as_str()?;
        if name == "condition" {
            continue;
        }
        if let Some(m) = v.as_mapping() {
            groups.insert(name.to_string(), parse_detection_group(m));
        }
    }

    Some(SigmaRule {
        id:        rf.id,
        title:     rf.title,
        tags:      rf.tags,
        level:     rf.level,
        groups,
        condition,
    })
}

// ── Event field extraction ────────────────────────────────────────────────────

/// Map Sigma field names (case-insensitive) to [`ProcessEvent`] string values.
fn extract_field<'a>(event: &'a ProcessEvent, field: &str) -> Option<String> {
    match field.to_lowercase().as_str() {
        "image" | "process.executable" | "processpath" => {
            Some(event.executable.clone())
        }
        "commandline" | "process.command_line" | "cmdline" => {
            Some(event.cmdline.clone())
        }
        "user" | "username" | "subject.user.name" => Some(event.user.clone()),
        "parentimage" | "parent_process" => Some(event.executable.clone()),
        "processid" | "pid" => Some(event.pid.to_string()),
        "parentprocessid" | "ppid" => Some(event.ppid.to_string()),
        "tenantid" | "tenant_id" => Some(event.tenant_id.clone()),
        _ => None,
    }
}

// ── Condition evaluation ──────────────────────────────────────────────────────

fn matches_group(group: &DetectionGroup, event: &ProcessEvent) -> bool {
    // All matchers in a group must match (AND semantics within a group).
    group.matchers.iter().all(|m| matches_matcher(m, event))
}

fn matches_matcher(m: &FieldMatcher, event: &ProcessEvent) -> bool {
    let haystack = match extract_field(event, &m.field) {
        Some(v) => v.to_lowercase(),
        None => return false,
    };
    match m.modifier {
        FieldModifier::Contains => m
            .values
            .iter()
            .any(|v| haystack.contains(v.to_lowercase().as_str())),
        FieldModifier::ContainsAll => m
            .values
            .iter()
            .all(|v| haystack.contains(v.to_lowercase().as_str())),
        FieldModifier::StartsWith => m
            .values
            .iter()
            .any(|v| haystack.starts_with(v.to_lowercase().as_str())),
        FieldModifier::EndsWith => m
            .values
            .iter()
            .any(|v| haystack.ends_with(v.to_lowercase().as_str())),
        FieldModifier::Re => m.values.iter().any(|v| {
            Regex::new(v).map(|re| re.is_match(&haystack)).unwrap_or(false)
        }),
        FieldModifier::Exact => m
            .values
            .iter()
            .any(|v| haystack == v.to_lowercase()),
    }
}

/// Evaluate a condition string against the populated group results map.
///
/// Handles:
/// - bare group names: `selection`
/// - `not selection`
/// - `A and B`, `A or B`
/// - `1 of selection*` / `all of selection*`
fn eval_condition(
    condition: &str,
    groups: &HashMap<String, DetectionGroup>,
    event: &ProcessEvent,
) -> bool {
    let cond = condition.trim();

    // "1 of X*" — any group whose name matches the pattern
    if let Some(rest) = cond.strip_prefix("1 of ") {
        let pattern = rest.trim().trim_end_matches('*');
        return groups
            .iter()
            .filter(|(k, _)| k.starts_with(pattern))
            .any(|(_, g)| matches_group(g, event));
    }
    // "all of X*"
    if let Some(rest) = cond.strip_prefix("all of ") {
        let pattern = rest.trim().trim_end_matches('*');
        let candidates: Vec<_> = groups
            .iter()
            .filter(|(k, _)| k.starts_with(pattern))
            .collect();
        return !candidates.is_empty()
            && candidates.iter().all(|(_, g)| matches_group(g, event));
    }

    // Split on " or " first (lowest precedence)
    if cond.contains(" or ") {
        return cond
            .split(" or ")
            .any(|part| eval_condition(part.trim(), groups, event));
    }
    // Split on " and "
    if cond.contains(" and ") {
        return cond
            .split(" and ")
            .all(|part| eval_condition(part.trim(), groups, event));
    }
    // "not X"
    if let Some(rest) = cond.strip_prefix("not ") {
        return !eval_condition(rest.trim(), groups, event);
    }
    // Bare group name
    if let Some(group) = groups.get(cond) {
        return matches_group(group, event);
    }
    false
}

// ── SigmaEngine ───────────────────────────────────────────────────────────────

/// In-memory Sigma detection engine.
pub struct SigmaEngine {
    rules: Vec<SigmaRule>,
}

impl SigmaEngine {
    /// Creates an engine with no rules (used as a safe fallback in tests).
    pub fn empty() -> Self {
        SigmaEngine { rules: vec![] }
    }

    /// Returns the number of loaded rules.
    pub fn rule_count(&self) -> usize {
        self.rules.len()
    }

    /// Recursively walks `dir`, parsing every `*.yml` / `*.yaml` file as a
    /// Sigma rule.  Malformed files are skipped with a warning.
    pub fn load_from_dir(dir: &str) -> Result<Self> {
        let mut rules = Vec::new();
        load_dir(Path::new(dir), &mut rules);
        Ok(SigmaEngine { rules })
    }

    /// Evaluates all loaded rules against the given event and returns the
    /// matching rules.
    pub fn evaluate(&self, event: &ProcessEvent) -> Vec<SigmaMatch> {
        self.rules
            .iter()
            .filter(|r| eval_condition(&r.condition, &r.groups, event))
            .map(|r| SigmaMatch {
                rule_id: r.id.clone(),
                title:   r.title.clone(),
                tags:    r.tags.clone(),
                level:   r.level.clone(),
            })
            .collect()
    }
}

fn load_dir(path: &Path, out: &mut Vec<SigmaRule>) {
    let entries = match fs::read_dir(path) {
        Ok(e) => e,
        Err(_) => return, // directory missing — not an error in CI
    };
    for entry in entries.flatten() {
        let p = entry.path();
        if p.is_dir() {
            load_dir(&p, out);
        } else {
            let ext = p.extension().and_then(|e| e.to_str()).unwrap_or("");
            if ext == "yml" || ext == "yaml" {
                match fs::read_to_string(&p) {
                    Ok(contents) => match serde_yaml::from_str::<RuleFile>(&contents) {
                        Ok(rf) => {
                            if let Some(rule) = parse_rule(rf) {
                                out.push(rule);
                            }
                        }
                        Err(e) => {
                            warn!(path = %p.display(), error = %e, "sigma: skipping malformed rule");
                        }
                    },
                    Err(e) => {
                        warn!(path = %p.display(), error = %e, "sigma: cannot read file");
                    }
                }
            }
        }
    }
}
