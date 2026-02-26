//! Unit tests for the CoreSec detection engine.
//!
//! These tests are designed to be CI-safe: they do not require any real Sigma
//! or YARA rule files to be present.  When the vendor directories are missing
//! the engines fall back to empty rule sets, which is the expected behaviour
//! in a fresh checkout.

use super::sigma::SigmaEngine;
use super::yara::YaraEngine;
use super::DetectionEngine;
use crate::event::ProcessEvent;

/// Construct a minimal [`ProcessEvent`] for testing purposes.
fn benign_event() -> ProcessEvent {
    ProcessEvent {
        tenant_id:   "test-tenant".into(),
        agent_id:    "agent-001".into(),
        event_id:    "evt-001".into(),
        timestamp:   "2025-01-01T00:00:00Z".into(),
        class_uid:   4007,
        severity_id: 1,
        activity_id: 1,
        pid:         1234,
        ppid:        1,
        executable:  "/bin/ls".into(),
        cmdline:     "ls -la /tmp".into(),
        user:        "nobody".into(),
        blake3_hash: "abc123".into(),
    }
}

#[test]
fn sigma_engine_loads_without_panic() {
    // vendor/sigma/rules/ may not exist in CI — must never panic.
    let engine = SigmaEngine::load_from_dir("vendor/sigma/rules")
        .unwrap_or_else(|_| SigmaEngine::empty());
    // rule_count() returns a usize — always >= 0.
    let _ = engine.rule_count();
}

#[test]
fn sigma_engine_evaluates_benign_event() {
    let engine = SigmaEngine::load_from_dir("vendor/sigma/rules")
        .unwrap_or_else(|_| SigmaEngine::empty());
    let hits = engine.evaluate(&benign_event());
    // With an empty rule set there must be zero matches.
    if engine.rule_count() == 0 {
        assert!(hits.is_empty(), "empty engine must not produce matches");
    }
}

#[test]
fn yara_engine_loads_without_panic() {
    // vendor/yara-rules/ may not exist in CI — must never panic.
    let engine = YaraEngine::load_from_dir("vendor/yara-rules")
        .unwrap_or_else(|_| YaraEngine::empty());
    let _ = engine.rule_count();
}

#[test]
fn yara_engine_scans_empty_data() {
    let engine = YaraEngine::empty();
    let hits = engine.scan(b"");
    assert!(hits.is_empty(), "scanning empty bytes must return no matches");
}

#[test]
fn yara_engine_scans_benign_data() {
    let engine = YaraEngine::load_from_dir("vendor/yara-rules")
        .unwrap_or_else(|_| YaraEngine::empty());
    // Benign string — YARA must not panic regardless of rule count.
    let _ = engine.scan(b"/bin/ls\x00ls -la /tmp\x00nobody");
}

#[test]
fn detection_engine_detect_returns_no_matches_on_benign() {
    let engine = DetectionEngine::new("vendor/sigma/rules", "vendor/yara-rules")
        .unwrap_or_else(|_| DetectionEngine::empty());
    let event = benign_event();
    let (sigma_hits, yara_hits) = engine.detect(&event);
    // With empty rule sets (typical CI) both slices must be empty.
    if engine.sigma.rule_count() == 0 {
        assert!(sigma_hits.is_empty());
    }
    if engine.yara.rule_count() == 0 {
        assert!(yara_hits.is_empty());
    }
}

#[test]
fn event_to_bytes_contains_executable_and_cmdline() {
    let event = benign_event();
    let bytes = DetectionEngine::event_to_bytes(&event);
    let s = String::from_utf8_lossy(&bytes);
    assert!(s.contains("/bin/ls"));
    assert!(s.contains("ls -la /tmp"));
    assert!(s.contains("nobody"));
}

#[test]
fn sigma_inline_rule_matches_known_pattern() {
    // Write a temporary Sigma rule to a temp file and verify the engine
    // correctly matches a crafted event.
    use std::io::Write;
    let rule_yaml = r#"
title: Test PowerShell Detection
id: test-rule-001
status: test
tags:
  - attack.execution
  - attack.t1059.001
level: high
detection:
  selection:
    CommandLine|contains:
      - powershell.exe
      - pwsh
  condition: selection
"#;
    let dir = std::env::temp_dir().join("kubric_sigma_test");
    std::fs::create_dir_all(&dir).unwrap();
    let rule_path = dir.join("test_rule.yml");
    {
        let mut f = std::fs::File::create(&rule_path).unwrap();
        f.write_all(rule_yaml.as_bytes()).unwrap();
    }

    let engine = SigmaEngine::load_from_dir(dir.to_str().unwrap())
        .expect("should load temp sigma rule");
    assert_eq!(engine.rule_count(), 1, "expected exactly 1 rule");

    // Matching event
    let mut ev = benign_event();
    ev.cmdline = "powershell.exe -enc dGVzdA==".into();
    let hits = engine.evaluate(&ev);
    assert_eq!(hits.len(), 1, "expected 1 sigma hit for powershell cmdline");
    assert_eq!(hits[0].rule_id, "test-rule-001");
    assert_eq!(hits[0].level, "high");

    // Non-matching event
    let clean_ev = benign_event();
    let no_hits = engine.evaluate(&clean_ev);
    assert!(no_hits.is_empty(), "ls -la must not trigger powershell rule");

    // Cleanup
    let _ = std::fs::remove_dir_all(&dir);
}
