use serde::{Deserialize, Serialize};

/// Canonical OCSF process activity event (class 4007).
/// tenant_id is first-class — set at the agent before NATS publish.
///
/// NATS subject: kubric.{tenant_id}.endpoint.process.v1
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct ProcessEvent {
    pub tenant_id: String,
    pub agent_id: String,
    pub event_id: String,
    pub timestamp: String,       // RFC3339
    pub class_uid: u32,          // OCSF 4007
    pub severity_id: u8,
    pub activity_id: u8,
    pub pid: u32,
    pub ppid: u32,
    pub executable: String,
    pub cmdline: String,
    pub user: String,
    pub blake3_hash: String,     // hash of raw event bytes
}

impl ProcessEvent {
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{tenant_id}.endpoint.process.v1")
    }
}
