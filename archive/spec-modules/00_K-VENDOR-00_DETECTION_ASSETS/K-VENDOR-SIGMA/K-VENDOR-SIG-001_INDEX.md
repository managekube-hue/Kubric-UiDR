# K-VENDOR-SIG-001 -- Sigma Rule Index

| Field | Value |
|-------|-------|
| **Asset Class** | Sigma Detection Rules |
| **Engine** | `agents/coresec/src/detection/sigma.rs` (`SigmaEngine`) |
| **Rule Store** | `vendor/sigma/rules/` (subdirs: `windows/`, `linux/`, `cloud/`, `network/`) |
| **Format** | YAML (`.yml` / `.yaml`) -- SigmaHQ-compatible schema |

## Overview

This index catalogues all Sigma rule families vendored into the Kubric platform.
CoreSec's native `SigmaEngine` loads every rule file recursively from the rule
store at agent startup and evaluates them against OCSF `ProcessEvent` telemetry
in real time. Malformed files are skipped with a tracing warning so a single bad
rule never blocks the detection pipeline.

## Rule Categories

| Doc ID | Category | Description |
|--------|----------|-------------|
| SIG-002 | Windows Built-in | Process creation, service installs, PowerShell, WMI |
| SIG-003 | Cloud | AWS CloudTrail, Azure Activity, GCP Audit rules |
| SIG-004 | SaaS | Microsoft 365, Google Workspace, Okta, GitHub audit |
| SIG-005 | Hunting | Proactive threat-hunt queries and anomaly detections |

## CoreSec Integration

```text
DetectionEngine::new(sigma_dir, yara_dir)
  --> SigmaEngine::load_from_dir(sigma_dir)
        --> recursively walks vendor/sigma/rules/**/*.yml
        --> serde_yaml::from_str::<RuleFile> for each file
        --> parse_rule() builds in-memory SigmaRule structs
  --> SigmaEngine::evaluate(event) called per ProcessEvent
        --> returns Vec<SigmaMatch> { rule_id, title, tags, level }
```

## Supported Detection Syntax

| Feature | Example |
|---------|---------|
| Field modifiers | `CommandLine\|contains`, `Image\|endswith` |
| Compound modifiers | `CommandLine\|contains\|all` |
| Regex modifier | `FieldName\|re` |
| Condition logic | `selection and not filter`, `1 of selection*`, `all of them` |
| Multi-value lists | OR semantics by default; AND with `\|contains\|all` |

## Hot Reload

CoreSec uses `notify` / `notify-debouncer-mini` to watch the rule store
directory. When a rule file is added, modified, or removed, the engine
reloads the full rule set without restarting the agent process.

## Severity Mapping

Sigma `level` values map directly to OCSF `severity_id` on the resulting alert:

| Sigma Level | OCSF severity_id | Kubric Tier |
|-------------|-------------------|-------------|
| informational | 1 | Telemetry |
| low | 2 | Low |
| medium | 3 | Medium |
| high | 4 | High |
| critical | 5 | Critical |
