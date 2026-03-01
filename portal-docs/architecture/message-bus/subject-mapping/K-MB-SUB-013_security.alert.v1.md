# K-MB-SUB-013 — Security Alert

> NATS subject mapping reference for the unified security alert intake subject within the Kubric UIDR platform.

## Subject Pattern

```
kubric.security.alert.v1
kubric.security.alert.>         # wildcard for all security alert sub-subjects
```

Tokens:
| Position | Token       | Description                                |
|----------|-------------|--------------------------------------------|
| 1        | `kubric`    | Platform root namespace                    |
| 2        | `security`  | Domain — security operations               |
| 3        | `alert`     | Event type — security alert                |
| 4        | `v1`        | Schema version                             |

## Publisher

**All agents** — every detection agent across the Kubric UIDR platform publishes security alerts to this unified subject:

| Agent      | Runtime | Alert Sources                                                   |
|------------|---------|------------------------------------------------------------------|
| CoreSec    | Rust    | EDR detections, FIM changes, process anomalies, syslog analysis  |
| NetGuard   | Rust    | IDS/NTA alerts, TLS anomalies, lateral movement detection        |
| VDR        | Go      | Critical vulnerability findings, exploit prediction triggers     |
| KIC        | Go      | Compliance violations, configuration drift, policy breaches      |

Each agent normalizes its native detection event into the unified OCSF security alert schema before publishing.

## Consumer(s)

| Consumer                   | Runtime  | Role                                                        |
|----------------------------|----------|--------------------------------------------------------------|
| **KAI-TRIAGE** (primary)   | Python   | AI-powered alert triage, enrichment, severity re-scoring, and routing |
| KAI-ANALYST                | Python   | Correlation analysis and threat hunting context              |
| K-SVC Portal               | TypeScript / Next.js | Real-time alert feed for SOC dashboard              |

## Payload

**Format:** JSON — OCSF (Open Cybersecurity Schema Framework) combined security alert.

**Schema version:** `1.0.0`

**Content-Type:** `application/json`

```jsonc
{
  "schema_version": "1.0.0",
  "source_module": "coresec",
  "event_class": "file_activity",
  "severity": "critical",
  "confidence": 92,
  "blake3_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "mitre_techniques": ["T1486", "T1059.001"],
  "affected_asset": {
    "asset_id": "ast_j9k0l1m2",
    "tenant_id": "tnt_a1b2c3d4",
    "hostname": "WORKSTATION-042",
    "ip": "10.0.5.42",
    "os": "Windows 11"
  },
  "raw_event": {
    "process_name": "suspicious.exe",
    "pid": 4812,
    "parent_pid": 1024,
    "file_path": "C:\\Users\\finance\\Documents\\",
    "action": "encrypt",
    "files_affected": 247,
    "duration_seconds": 30
  },
  "timestamp": "2026-02-26T14:29:45Z",
  "dedup_key": "coresec:file_activity:ast_j9k0l1m2:T1486:2026-02-26T14"
}
```

## Fields

| Field                        | Type     | Required | Description                                                             |
|------------------------------|----------|----------|-------------------------------------------------------------------------|
| `schema_version`             | string   | yes      | Payload schema version (semver)                                         |
| `source_module`              | string   | yes      | Originating agent: `coresec`, `netguard`, `vdr`, `kic`                  |
| `event_class`                | string   | yes      | OCSF event class: `file_activity`, `network_activity`, `vulnerability_finding`, `compliance_check` |
| `severity`                   | string   | yes      | Alert severity: `critical`, `high`, `medium`, `low`, `informational`    |
| `confidence`                 | integer  | yes      | Detection confidence score (0-100)                                      |
| `blake3_hash`                | string   | yes      | BLAKE3 hash of the raw event for integrity verification and deduplication |
| `mitre_techniques`           | string[] | no       | MITRE ATT&CK technique IDs mapped by the detection rule                 |
| `affected_asset.asset_id`    | string   | yes      | Unique asset identifier (`ast_` prefix)                                 |
| `affected_asset.tenant_id`   | string   | yes      | Tenant owning the asset (`tnt_` prefix)                                 |
| `affected_asset.hostname`    | string   | yes      | Hostname of the affected system                                         |
| `affected_asset.ip`          | string   | yes      | IP address of the affected system                                       |
| `affected_asset.os`          | string   | no       | Operating system of the affected asset                                  |
| `raw_event`                  | object   | yes      | Source-specific raw event data (schema varies by `source_module`)        |
| `timestamp`                  | datetime | yes      | ISO 8601 UTC timestamp of the detection event                           |
| `dedup_key`                  | string   | yes      | Deterministic deduplication key for alert correlation                    |

### Deduplication Strategy

Alert deduplication operates at two levels:

**1. Publisher-side (agent):** Each agent computes a `dedup_key` from `(source_module, event_class, asset_id, primary_technique, hour_bucket)`. Identical keys within a 1-hour window are suppressed at the agent level.

**2. Consumer-side (KAI-TRIAGE):** KAI-TRIAGE maintains a sliding window deduplication cache (Redis-backed) with a 4-hour TTL. Duplicate alerts are correlated into an existing incident rather than creating a new one.

```
Dedup Key Formula:
  {source_module}:{event_class}:{asset_id}:{primary_technique}:{YYYY-MM-DDTHH}

Example:
  coresec:file_activity:ast_j9k0l1m2:T1486:2026-02-26T14
```

### Alert Fatigue Reduction

KAI-TRIAGE employs the following strategies to reduce alert fatigue:

| Strategy                 | Description                                              |
|--------------------------|----------------------------------------------------------|
| Confidence thresholding  | Alerts below confidence 30 are auto-closed as noise      |
| Correlation grouping     | Related alerts on the same asset within 1 hour are merged into a single incident |
| Severity re-scoring      | AI model re-evaluates severity using asset criticality, threat context, and historical false positive rate |
| Allowlist suppression    | Known benign patterns (e.g., scheduled scans) are suppressed via tenant-configurable allowlists |

## JetStream Configuration

```
Stream:          SECURITY_ALERT
Subjects:        kubric.security.alert.>
Storage:         File
Retention:       Limits
Max Age:         30 days
Max Bytes:       20 GB
Replicas:        3
Discard Policy:  Old
Duplicate Window: 60 seconds
```

## Consumer Groups

| Consumer Group        | Deliver Policy | Ack Policy  | Max Deliver | Filter Subject                |
|-----------------------|---------------|-------------|-------------|-------------------------------|
| `triage-intake`       | All           | Explicit    | 5           | `kubric.security.alert.>`     |
| `analyst-correlator`  | All           | Explicit    | 3           | `kubric.security.alert.>`     |
| `portal-alert-feed`   | New           | Explicit    | 3           | `kubric.security.alert.>`     |

## Example (NATS CLI)

**Publish a security alert:**

```bash
nats pub kubric.security.alert.v1 '{
  "schema_version": "1.0.0",
  "source_module": "coresec",
  "event_class": "file_activity",
  "severity": "critical",
  "confidence": 92,
  "blake3_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "mitre_techniques": ["T1486", "T1059.001"],
  "affected_asset": {
    "asset_id": "ast_j9k0l1m2",
    "tenant_id": "tnt_a1b2c3d4",
    "hostname": "WORKSTATION-042",
    "ip": "10.0.5.42",
    "os": "Windows 11"
  },
  "raw_event": {
    "process_name": "suspicious.exe",
    "pid": 4812,
    "parent_pid": 1024,
    "file_path": "C:\\Users\\finance\\Documents\\",
    "action": "encrypt",
    "files_affected": 247,
    "duration_seconds": 30
  },
  "timestamp": "2026-02-26T14:29:45Z",
  "dedup_key": "coresec:file_activity:ast_j9k0l1m2:T1486:2026-02-26T14"
}'
```

**Subscribe to all security alerts:**

```bash
nats sub "kubric.security.alert.>"
```

**Create the JetStream stream:**

```bash
nats stream add SECURITY_ALERT \
  --subjects "kubric.security.alert.>" \
  --storage file \
  --retention limits \
  --max-age 30d \
  --max-bytes 20GB \
  --replicas 3 \
  --discard old \
  --dupe-window 60s
```

**Create the triage-intake consumer:**

```bash
nats consumer add SECURITY_ALERT triage-intake \
  --deliver all \
  --ack explicit \
  --max-deliver 5 \
  --filter "kubric.security.alert.>"
```

## Notes

- **OCSF Compliance:** All alert payloads conform to the Open Cybersecurity Schema Framework v1.1. The `event_class` field maps directly to OCSF event classes for interoperability with external SIEM/SOAR platforms.
- **BLAKE3 Integrity:** The `blake3_hash` is computed over the serialized `raw_event` object. Consumers can verify event integrity end-to-end without trusting intermediate message bus infrastructure.
- **Volume Estimate:** Expect 10,000-50,000 raw alerts per day per tenant across all agents. After deduplication and noise filtering, KAI-TRIAGE produces approximately 50-200 actionable incidents per day.
- **Priority Processing:** KAI-TRIAGE processes `critical` and `high` severity alerts within a dedicated priority lane (separate NATS consumer with higher fetch batch size) to ensure sub-second triage latency.
- **Downstream Flow:** After triage, actionable alerts flow to `kubric.comm.alert.v1` (notification) and optionally to `kubric.remediation.task.v1` (auto-remediation).
- **Related Subjects:** `kubric.comm.alert.v1` (downstream notification), `kubric.remediation.task.v1` (auto-remediation), `kubric.security.incident.v1` (correlated incident).
