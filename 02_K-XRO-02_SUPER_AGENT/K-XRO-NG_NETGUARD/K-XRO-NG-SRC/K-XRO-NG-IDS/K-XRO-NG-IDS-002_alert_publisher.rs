//! K-XRO-NG-IDS-002 — Network IDS alert publisher to NATS.
//!
//! Serialises `NetworkAlert` structs to JSON and publishes them to a
//! per-tenant NATS subject in OCSF-compatible format.
//!
//! # NATS subject
//! ```
//! kubric.{tenant_id}.detection.network_ids.v1
//! ```
//!
//! # Usage
//! ```rust,ignore
//! let publisher = AlertPublisher::new(nats_client, "tenant-abc");
//! let alert = NetworkAlert {
//!     id: uuid::Uuid::new_v4().to_string(),
//!     tenant_id: "tenant-abc".into(),
//!     src_ip: "10.0.0.1".into(),
//!     dst_ip: "185.220.101.1".into(),
//!     src_port: 54321,
//!     dst_port: 443,
//!     proto: "TCP".into(),
//!     rule_name: "TLS_C2_Beacon".into(),
//!     severity: "high".into(),
//!     payload_hex: "16030100aa010000a6...".into(),
//!     timestamp: epoch_ms(),
//! };
//! publisher.publish_alert(alert).await?;
//! ```

#![allow(dead_code)]

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use tracing::{debug, error, info, warn};

// ── NetworkAlert ──────────────────────────────────────────────────────────────

/// A single network IDS detection event.
///
/// Fields follow OCSF Network Activity (class 4001) + detection enrichment.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NetworkAlert {
    /// UUID v4 identifying this specific alert instance.
    pub id: String,

    /// Kubric tenant this event belongs to.
    pub tenant_id: String,

    /// Source IP (IPv4 or IPv6 string).
    pub src_ip: String,
    /// Destination IP.
    pub dst_ip: String,
    /// Source port (0 if ICMP).
    pub src_port: u16,
    /// Destination port (0 if ICMP).
    pub dst_port: u16,
    /// Transport protocol: "TCP", "UDP", "ICMP", …
    pub proto: String,

    /// Matching IDS rule identifier (YARA rule id or Suricata SID as string).
    pub rule_name: String,
    /// Severity: "informational", "low", "medium", "high", "critical".
    pub severity: String,

    /// First `payload_bytes` bytes of the matched packet, hex-encoded.
    pub payload_hex: String,

    /// UTC milliseconds since epoch.
    pub timestamp: u64,

    /// Optional: OCSF class UID (Network Activity = 4001).
    pub class_uid: u32,

    /// Optional: OCSF activity ID (Traffic = 6).
    pub activity_id: u32,
}

impl NetworkAlert {
    /// NATS subject for a given tenant.
    pub fn nats_subject(tenant_id: &str) -> String {
        format!("kubric.{tenant_id}.detection.network_ids.v1")
    }

    /// Current UTC milliseconds since epoch.
    pub fn now_ms() -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_millis() as u64
    }

    /// Construct an alert with automatic ID and timestamp.
    pub fn new(
        tenant_id: &str,
        src_ip: &str,
        dst_ip: &str,
        src_port: u16,
        dst_port: u16,
        proto: &str,
        rule_name: &str,
        severity: &str,
        payload: &[u8],
    ) -> Self {
        let id = {
            use std::collections::hash_map::DefaultHasher;
            use std::hash::{Hash, Hasher};
            let mut h = DefaultHasher::new();
            src_ip.hash(&mut h);
            dst_ip.hash(&mut h);
            src_port.hash(&mut h);
            dst_port.hash(&mut h);
            rule_name.hash(&mut h);
            Self::now_ms().hash(&mut h);
            format!("{:016x}", h.finish())
        };

        let payload_hex = payload[..payload.len().min(64)]
            .iter()
            .map(|b| format!("{b:02x}"))
            .collect::<String>();

        Self {
            id,
            tenant_id: tenant_id.to_string(),
            src_ip: src_ip.to_string(),
            dst_ip: dst_ip.to_string(),
            src_port,
            dst_port,
            proto: proto.to_string(),
            rule_name: rule_name.to_string(),
            severity: severity.to_string(),
            payload_hex,
            timestamp: Self::now_ms(),
            class_uid: 4001, // OCSF Network Activity
            activity_id: 6,  // Traffic
        }
    }

    /// Convert severity string to OCSF severity ID.
    pub fn severity_id(&self) -> u8 {
        match self.severity.to_lowercase().as_str() {
            "informational" => 1,
            "low" => 2,
            "medium" => 3,
            "high" => 4,
            "critical" => 5,
            _ => 0,
        }
    }
}

// ── AlertPublisher ────────────────────────────────────────────────────────────

/// Publishes `NetworkAlert` events to NATS.
pub struct AlertPublisher {
    client: async_nats::Client,
    tenant_id: String,
    subject: String,
    /// Total alerts published since this struct was created.
    pub published_count: Arc<AtomicU64>,
    /// Total publish errors.
    pub error_count: Arc<AtomicU64>,
}

impl AlertPublisher {
    /// Create a publisher bound to a specific tenant.
    pub fn new(client: async_nats::Client, tenant_id: &str) -> Self {
        let subject = NetworkAlert::nats_subject(tenant_id);
        info!(tenant_id, subject = %subject, "AlertPublisher created");
        Self {
            client,
            tenant_id: tenant_id.to_string(),
            subject,
            published_count: Arc::new(AtomicU64::new(0)),
            error_count: Arc::new(AtomicU64::new(0)),
        }
    }

    /// Publish a single `NetworkAlert` to NATS.
    ///
    /// The alert is serialised to JSON before publishing.
    pub async fn publish_alert(&self, alert: NetworkAlert) -> Result<()> {
        let payload = serde_json::to_vec(&alert)
            .context("serialise NetworkAlert to JSON")?;

        self.client
            .publish(self.subject.clone(), payload.into())
            .await
            .with_context(|| format!("NATS publish to {}", self.subject))?;

        self.published_count.fetch_add(1, Ordering::Relaxed);

        debug!(
            id = %alert.id,
            rule = %alert.rule_name,
            src = %alert.src_ip,
            dst = %alert.dst_ip,
            severity = %alert.severity,
            "alert published"
        );

        Ok(())
    }

    /// Publish multiple alerts in sequence.
    ///
    /// Continues publishing even if individual alerts fail — errors are
    /// logged and counted but not propagated until all alerts are attempted.
    /// Returns the number of successfully published alerts.
    pub async fn publish_batch(&self, alerts: &[NetworkAlert]) -> Result<usize> {
        if alerts.is_empty() {
            return Ok(0);
        }

        let mut published = 0;
        let mut last_err: Option<anyhow::Error> = None;

        for alert in alerts {
            match self.publish_alert(alert.clone()).await {
                Ok(()) => published += 1,
                Err(e) => {
                    self.error_count.fetch_add(1, Ordering::Relaxed);
                    warn!(
                        id = %alert.id,
                        rule = %alert.rule_name,
                        err = %e,
                        "failed to publish alert"
                    );
                    last_err = Some(e);
                }
            }
        }

        if let Some(err) = last_err {
            if published == 0 {
                return Err(err).context("all alerts in batch failed to publish");
            }
            warn!(
                published,
                failed = alerts.len() - published,
                "batch publish partial failure"
            );
        }

        info!(
            published,
            total = alerts.len(),
            tenant = %self.tenant_id,
            "alert batch published"
        );

        Ok(published)
    }

    /// Publish an alert derived from a YARA rule match.
    ///
    /// Convenience wrapper that constructs a `NetworkAlert` from the key fields
    /// that are available at match time.
    pub async fn publish_yara_match(
        &self,
        rule_id: &str,
        src_ip: &str,
        dst_ip: &str,
        src_port: u16,
        dst_port: u16,
        proto: &str,
        payload: &[u8],
    ) -> Result<()> {
        let alert = NetworkAlert::new(
            &self.tenant_id,
            src_ip,
            dst_ip,
            src_port,
            dst_port,
            proto,
            rule_id,
            "high", // YARA matches default to high until tuned
            payload,
        );
        self.publish_alert(alert).await
    }

    /// Current NATS subject.
    pub fn subject(&self) -> &str {
        &self.subject
    }

    /// Total alerts successfully published.
    pub fn published(&self) -> u64 {
        self.published_count.load(Ordering::Relaxed)
    }

    /// Total publish failures.
    pub fn errors(&self) -> u64 {
        self.error_count.load(Ordering::Relaxed)
    }
}

// ── Tests ─────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn alert_subject_format() {
        let s = NetworkAlert::nats_subject("tenant-xyz");
        assert_eq!(s, "kubric.tenant-xyz.detection.network_ids.v1");
    }

    #[test]
    fn alert_new_fills_fields() {
        let a = NetworkAlert::new(
            "t1",
            "10.0.0.1",
            "8.8.8.8",
            54321,
            443,
            "TCP",
            "TLS_Beacon",
            "high",
            &[0x16, 0x03, 0x01],
        );
        assert_eq!(a.tenant_id, "t1");
        assert_eq!(a.src_ip, "10.0.0.1");
        assert_eq!(a.dst_ip, "8.8.8.8");
        assert_eq!(a.src_port, 54321);
        assert_eq!(a.dst_port, 443);
        assert_eq!(a.proto, "TCP");
        assert_eq!(a.rule_name, "TLS_Beacon");
        assert_eq!(a.severity, "high");
        assert_eq!(a.class_uid, 4001);
        assert_eq!(a.activity_id, 6);
        assert!(!a.id.is_empty());
        assert!(!a.payload_hex.is_empty());
        assert_eq!(a.payload_hex, "160301");
    }

    #[test]
    fn alert_payload_truncated_at_64_bytes() {
        let big_payload = vec![0xABu8; 200];
        let a = NetworkAlert::new("t", "1.1.1.1", "2.2.2.2", 1, 2, "UDP", "rule", "low", &big_payload);
        // 64 bytes → 128 hex chars
        assert_eq!(a.payload_hex.len(), 128);
    }

    #[test]
    fn severity_id_mapping() {
        let mut a = NetworkAlert::new("t", "a", "b", 1, 2, "TCP", "r", "critical", b"");
        assert_eq!(a.severity_id(), 5);
        a.severity = "medium".into(); assert_eq!(a.severity_id(), 3);
        a.severity = "low".into();    assert_eq!(a.severity_id(), 2);
        a.severity = "high".into();   assert_eq!(a.severity_id(), 4);
        a.severity = "informational".into(); assert_eq!(a.severity_id(), 1);
        a.severity = "unknown".into(); assert_eq!(a.severity_id(), 0);
    }

    #[test]
    fn alert_serializes_to_json() {
        let a = NetworkAlert::new("t", "127.0.0.1", "127.0.0.2", 80, 443, "TCP", "test", "low", b"");
        let json = serde_json::to_string(&a).unwrap();
        assert!(json.contains("\"tenant_id\":\"t\""));
        assert!(json.contains("\"class_uid\":4001"));
    }

    #[test]
    fn now_ms_is_reasonable() {
        let ms = NetworkAlert::now_ms();
        // After year 2020 = 1577836800000 ms
        assert!(ms > 1_577_836_800_000);
    }
}
