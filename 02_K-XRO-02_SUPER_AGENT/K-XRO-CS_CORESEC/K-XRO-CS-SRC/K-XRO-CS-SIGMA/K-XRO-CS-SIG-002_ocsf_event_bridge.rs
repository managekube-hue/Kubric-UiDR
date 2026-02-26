//! K-XRO-CS-SIG-002 — OCSF (Open Cybersecurity Schema Framework) event bridge.
//!
//! Converts internal Kubric event types to OCSF-standard structures for
//! interoperability with SIEM platforms, data lakes, and third-party tooling.
//!
//! # OCSF class UIDs used
//! * 4001 — File System Activity
//! * 4003 — Network Activity
//! * 4007 — Process Activity
//! * 2004 — Detection Finding
//!
//! Reference: https://schema.ocsf.io/
//!
//! # Cargo dependencies
//! ```toml
//! serde      = { version = "1", features = ["derive"] }
//! serde_json = "1"
//! ```

use serde::{Deserialize, Serialize};
use serde_json::{json, Map, Value};
use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};

// ─────────────────────────────────────────────────────────────────────────────
// Shared event types that we map FROM
// ─────────────────────────────────────────────────────────────────────────────

/// Internal process event (mirrors CoreSec ProcessEvent).
#[derive(Debug, Clone, Default)]
pub struct ProcessEvent {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: u64,
    pub exe: String,
    pub cmdline: String,
    pub pid: u32,
    pub ppid: u32,
    pub uid: u32,
    pub user: String,
    pub severity: String,
}

/// Internal FIM (File Integrity Monitoring) event.
#[derive(Debug, Clone, Default)]
pub struct FimEvent {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: u64,
    pub path: String,
    pub old_hash: String,
    pub new_hash: String,
    pub severity: String,
    /// Activity: "create" | "modify" | "delete"
    pub activity: String,
}

/// Internal network flow event.
#[derive(Debug, Clone, Default)]
pub struct NetworkEvent {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: u64,
    pub src_ip: String,
    pub dst_ip: String,
    pub src_port: u16,
    pub dst_port: u16,
    pub proto: String,
    pub bytes_sent: u64,
    pub bytes_received: u64,
    pub severity: String,
}

// ─────────────────────────────────────────────────────────────────────────────
// OCSF severity
// ─────────────────────────────────────────────────────────────────────────────

/// OCSF severity identifiers.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum OcsfSeverity {
    Unknown = 0,
    Informational = 1,
    Low = 2,
    Medium = 3,
    High = 4,
    Critical = 5,
}

impl OcsfSeverity {
    /// Convert a string severity label to the OCSF numeric ID.
    pub fn from_str(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "informational" | "info" => Self::Informational,
            "low" => Self::Low,
            "medium" | "med" => Self::Medium,
            "high" => Self::High,
            "critical" => Self::Critical,
            _ => Self::Unknown,
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Unknown => "Unknown",
            Self::Informational => "Informational",
            Self::Low => "Low",
            Self::Medium => "Medium",
            Self::High => "High",
            Self::Critical => "Critical",
        }
    }

    pub fn id(&self) -> u8 {
        *self as u8
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// OCSF event
// ─────────────────────────────────────────────────────────────────────────────

/// An OCSF-compliant event object ready for serialisation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcsfEvent {
    /// OCSF class UID (e.g. 4007 = Process Activity).
    pub class_uid: u32,
    /// OCSF class name string (informational).
    pub class_name: String,
    /// Severity identifier (0–5).
    pub severity_id: u8,
    /// Severity label.
    pub severity: String,
    /// Activity identifier (class-specific integer).
    pub activity_id: u32,
    /// Activity name string.
    pub activity_name: String,
    /// Unix epoch milliseconds.
    pub time: u64,
    /// OCSF metadata object.
    pub metadata: OcsfMetadata,
    /// OCSF observables array (key indicators).
    pub observables: Vec<OcsfObservable>,
    /// Fields that don't map to a standard OCSF attribute.
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub unmapped: Map<String, Value>,
}

/// OCSF metadata sub-object.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcsfMetadata {
    pub version: String,
    pub product: OcsfProduct,
    pub event_code: String,
}

/// OCSF product sub-object within metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcsfProduct {
    pub name: String,
    pub vendor_name: String,
    pub version: String,
}

/// OCSF observable — a notable value extracted from the event.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcsfObservable {
    pub name: String,
    #[serde(rename = "type")]
    pub type_name: String,
    pub type_id: u32,
    pub value: String,
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper constructors
// ─────────────────────────────────────────────────────────────────────────────

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_millis() as u64
}

fn kubric_metadata(event_code: &str) -> OcsfMetadata {
    OcsfMetadata {
        version: "1.1.0".to_string(),
        product: OcsfProduct {
            name: "Kubric XDR".to_string(),
            vendor_name: "Kubric".to_string(),
            version: env!("CARGO_PKG_VERSION").to_string(),
        },
        event_code: event_code.to_string(),
    }
}

fn observable(name: &str, type_name: &str, type_id: u32, value: &str) -> OcsfObservable {
    OcsfObservable {
        name: name.to_string(),
        type_name: type_name.to_string(),
        type_id,
        value: value.to_string(),
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Mapping functions
// ─────────────────────────────────────────────────────────────────────────────

/// Map a `ProcessEvent` to OCSF class 4007 — Process Activity.
///
/// Activity IDs:
/// * 1 = Launch
/// * 2 = Terminate
/// * 4 = Open
pub fn map_process_event(event: &ProcessEvent) -> OcsfEvent {
    let severity = OcsfSeverity::from_str(&event.severity);
    let ts = if event.timestamp > 0 { event.timestamp } else { now_ms() };

    let mut observables = vec![
        observable("process.exe", "File", 23, &event.exe),
        observable("process.command_line", "Other", 99, &event.cmdline),
        observable("actor.user.uid", "User", 4, &event.uid.to_string()),
    ];
    if !event.user.is_empty() {
        observables.push(observable("actor.user.name", "User", 4, &event.user));
    }

    let mut unmapped = Map::new();
    unmapped.insert("pid".to_string(), json!(event.pid));
    unmapped.insert("ppid".to_string(), json!(event.ppid));
    unmapped.insert("agent_id".to_string(), json!(event.agent_id));
    unmapped.insert("tenant_id".to_string(), json!(event.tenant_id));

    OcsfEvent {
        class_uid: 4007,
        class_name: "Process Activity".to_string(),
        severity_id: severity.id(),
        severity: severity.as_str().to_string(),
        activity_id: 1, // Launch
        activity_name: "Launch".to_string(),
        time: ts,
        metadata: kubric_metadata(&event.event_id),
        observables,
        unmapped,
    }
}

/// Map a `FimEvent` to OCSF class 4001 — File System Activity.
///
/// Activity IDs:
/// * 1 = Create
/// * 3 = Modify
/// * 4 = Delete
pub fn map_fim_event(event: &FimEvent) -> OcsfEvent {
    let severity = OcsfSeverity::from_str(&event.severity);
    let ts = if event.timestamp > 0 { event.timestamp } else { now_ms() };

    let activity_id = match event.activity.to_lowercase().as_str() {
        "create" => 1u32,
        "modify" | "modified" => 3,
        "delete" | "deleted" => 4,
        _ => 99,
    };
    let activity_name = match activity_id {
        1 => "Create",
        3 => "Modify",
        4 => "Delete",
        _ => "Unknown",
    };

    let observables = vec![
        observable("file.path", "File", 23, &event.path),
        observable("file.hashes.old", "Other", 99, &event.old_hash),
        observable("file.hashes.new", "Other", 99, &event.new_hash),
    ];

    let mut unmapped = Map::new();
    unmapped.insert("agent_id".to_string(), json!(event.agent_id));
    unmapped.insert("tenant_id".to_string(), json!(event.tenant_id));

    OcsfEvent {
        class_uid: 4001,
        class_name: "File System Activity".to_string(),
        severity_id: severity.id(),
        severity: severity.as_str().to_string(),
        activity_id,
        activity_name: activity_name.to_string(),
        time: ts,
        metadata: kubric_metadata(&event.event_id),
        observables,
        unmapped,
    }
}

/// Map a `NetworkEvent` to OCSF class 4003 — Network Activity.
///
/// Activity ID 6 = Traffic (generic flow).
pub fn map_network_event(event: &NetworkEvent) -> OcsfEvent {
    let severity = OcsfSeverity::from_str(&event.severity);
    let ts = if event.timestamp > 0 { event.timestamp } else { now_ms() };

    let observables = vec![
        observable("src_endpoint.ip", "IP Address", 2, &event.src_ip),
        observable("dst_endpoint.ip", "IP Address", 2, &event.dst_ip),
        observable("src_endpoint.port", "Port", 10, &event.src_port.to_string()),
        observable("dst_endpoint.port", "Port", 10, &event.dst_port.to_string()),
        observable("connection_info.protocol_name", "Other", 99, &event.proto),
    ];

    let mut unmapped = Map::new();
    unmapped.insert("bytes_sent".to_string(), json!(event.bytes_sent));
    unmapped.insert("bytes_received".to_string(), json!(event.bytes_received));
    unmapped.insert("agent_id".to_string(), json!(event.agent_id));
    unmapped.insert("tenant_id".to_string(), json!(event.tenant_id));

    OcsfEvent {
        class_uid: 4003,
        class_name: "Network Activity".to_string(),
        severity_id: severity.id(),
        severity: severity.as_str().to_string(),
        activity_id: 6, // Traffic
        activity_name: "Traffic".to_string(),
        time: ts,
        metadata: kubric_metadata(&event.event_id),
        observables,
        unmapped,
    }
}

/// Map a detection rule match to OCSF class 2004 — Detection Finding.
///
/// Activity ID 1 = Create (new finding reported).
pub fn map_detection_finding(
    rule_id: &str,
    rule_title: &str,
    severity: &str,
    tenant_id: &str,
    agent_id: &str,
) -> OcsfEvent {
    let sev = OcsfSeverity::from_str(severity);

    let observables = vec![
        observable("finding.uid", "Other", 99, rule_id),
        observable("finding.title", "Other", 99, rule_title),
    ];

    let mut unmapped = Map::new();
    unmapped.insert("tenant_id".to_string(), json!(tenant_id));
    unmapped.insert("agent_id".to_string(), json!(agent_id));
    unmapped.insert("rule_id".to_string(), json!(rule_id));

    OcsfEvent {
        class_uid: 2004,
        class_name: "Detection Finding".to_string(),
        severity_id: sev.id(),
        severity: sev.as_str().to_string(),
        activity_id: 1, // Create
        activity_name: "Create".to_string(),
        time: now_ms(),
        metadata: kubric_metadata(rule_id),
        observables,
        unmapped,
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Serialisation helpers
// ─────────────────────────────────────────────────────────────────────────────

/// Serialise an `OcsfEvent` to a compact JSON string.
pub fn to_json(event: &OcsfEvent) -> String {
    serde_json::to_string(event).unwrap_or_else(|e| format!("{{\"error\": \"{e}\"}}"))
}

/// Serialise an `OcsfEvent` to a pretty-printed JSON string.
pub fn to_json_pretty(event: &OcsfEvent) -> String {
    serde_json::to_string_pretty(event).unwrap_or_else(|e| format!("{{\"error\": \"{e}\"}}"))
}

/// Parse a JSON string back into an `OcsfEvent`.
pub fn from_json(json_str: &str) -> Result<OcsfEvent, serde_json::Error> {
    serde_json::from_str(json_str)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    fn sample_process_event() -> ProcessEvent {
        ProcessEvent {
            tenant_id: "acme".into(),
            agent_id: "agent-001".into(),
            event_id: "ev-001".into(),
            timestamp: 1_700_000_000_000,
            exe: "/bin/bash".into(),
            cmdline: "bash -c 'id'".into(),
            pid: 12345,
            ppid: 1,
            uid: 0,
            user: "root".into(),
            severity: "high".into(),
        }
    }

    fn sample_fim_event() -> FimEvent {
        FimEvent {
            tenant_id: "acme".into(),
            agent_id: "agent-001".into(),
            event_id: "fim-001".into(),
            timestamp: 1_700_000_000_000,
            path: "/etc/passwd".into(),
            old_hash: "abc123".into(),
            new_hash: "def456".into(),
            severity: "critical".into(),
            activity: "modify".into(),
        }
    }

    fn sample_network_event() -> NetworkEvent {
        NetworkEvent {
            tenant_id: "acme".into(),
            agent_id: "agent-001".into(),
            event_id: "net-001".into(),
            timestamp: 1_700_000_000_000,
            src_ip: "10.0.0.5".into(),
            dst_ip: "203.0.113.1".into(),
            src_port: 54321,
            dst_port: 443,
            proto: "TCP".into(),
            bytes_sent: 1024,
            bytes_received: 4096,
            severity: "medium".into(),
        }
    }

    #[test]
    fn process_event_class_uid() {
        let ev = map_process_event(&sample_process_event());
        assert_eq!(ev.class_uid, 4007);
        assert_eq!(ev.class_name, "Process Activity");
    }

    #[test]
    fn process_event_severity_high() {
        let ev = map_process_event(&sample_process_event());
        assert_eq!(ev.severity_id, OcsfSeverity::High.id());
        assert_eq!(ev.severity, "High");
    }

    #[test]
    fn process_event_observables_contain_exe() {
        let ev = map_process_event(&sample_process_event());
        let has_exe = ev
            .observables
            .iter()
            .any(|o| o.name == "process.exe" && o.value == "/bin/bash");
        assert!(has_exe);
    }

    #[test]
    fn fim_event_class_uid() {
        let ev = map_fim_event(&sample_fim_event());
        assert_eq!(ev.class_uid, 4001);
    }

    #[test]
    fn fim_modify_activity_id() {
        let ev = map_fim_event(&sample_fim_event());
        assert_eq!(ev.activity_id, 3, "modify → activity_id 3");
        assert_eq!(ev.activity_name, "Modify");
    }

    #[test]
    fn fim_event_severity_critical() {
        let ev = map_fim_event(&sample_fim_event());
        assert_eq!(ev.severity_id, OcsfSeverity::Critical.id());
    }

    #[test]
    fn network_event_class_uid() {
        let ev = map_network_event(&sample_network_event());
        assert_eq!(ev.class_uid, 4003);
        assert_eq!(ev.activity_id, 6);
    }

    #[test]
    fn network_event_observables_contain_ips() {
        let ev = map_network_event(&sample_network_event());
        let src = ev.observables.iter().find(|o| o.name == "src_endpoint.ip");
        let dst = ev.observables.iter().find(|o| o.name == "dst_endpoint.ip");
        assert!(src.is_some());
        assert!(dst.is_some());
        assert_eq!(src.unwrap().value, "10.0.0.5");
        assert_eq!(dst.unwrap().value, "203.0.113.1");
    }

    #[test]
    fn detection_finding_class_uid() {
        let ev = map_detection_finding("SIGMA-001", "Suspicious curl", "high", "acme", "ag-01");
        assert_eq!(ev.class_uid, 2004);
        assert_eq!(ev.class_name, "Detection Finding");
    }

    #[test]
    fn to_json_roundtrip() {
        let ev = map_process_event(&sample_process_event());
        let json_str = to_json(&ev);
        let reparsed = from_json(&json_str).expect("re-parse failed");
        assert_eq!(reparsed.class_uid, 4007);
        assert_eq!(reparsed.severity_id, 4);
    }

    #[test]
    fn ocsf_severity_from_str() {
        assert_eq!(OcsfSeverity::from_str("high").id(), 4);
        assert_eq!(OcsfSeverity::from_str("CRITICAL").id(), 5);
        assert_eq!(OcsfSeverity::from_str("unknown_xyz").id(), 0);
    }

    #[test]
    fn ocsf_severity_as_str() {
        assert_eq!(OcsfSeverity::High.as_str(), "High");
        assert_eq!(OcsfSeverity::Unknown.as_str(), "Unknown");
    }

    #[test]
    fn json_contains_metadata_product() {
        let ev = map_process_event(&sample_process_event());
        let json_str = to_json(&ev);
        assert!(json_str.contains("Kubric XDR"), "metadata.product.name missing");
    }

    #[test]
    fn fim_create_delete_activity_ids() {
        let mut e = sample_fim_event();
        e.activity = "create".into();
        assert_eq!(map_fim_event(&e).activity_id, 1);
        e.activity = "delete".into();
        assert_eq!(map_fim_event(&e).activity_id, 4);
    }
}
