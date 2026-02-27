# K-MB-SUB-007 — grc.drift.v1

## Overview

This subject carries compliance-drift events produced by the **GRC compliance service**. A
`DriftEvent` is published when a configuration control check finds that an asset deviates from its
expected state under CIS Benchmarks, NIST 800-53, SOC 2, or other frameworks. Consumers are the
housekeeper automated-remediation agent, the drift remediation workflow, and the audit log service.

**Subject pattern:** `kubric.<tenant_id>.grc.drift.v1`

**Example resolved subject:** `kubric.acme_corp.grc.drift.v1`

Multi-tenant wildcard: `kubric.*.grc.drift.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: GRC-DRIFT — apply with: nats stream add --config grc-drift-stream.yaml
stream:
  name: GRC-DRIFT
  subjects:
    - "kubric.*.grc.drift.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 31536000000000000  # 365 days in nanoseconds
  max_bytes: 5368709120       # 5 GiB (drift events are low-frequency)
  max_msg_size: 32768         # 32 KiB per message
  max_msgs: -1
  discard: old
  duplicate_window: 3600000000000  # 1-hour dedup (same control checked hourly)
  # 365-day retention supports annual SOC 2 / ISO 27001 audit cycles

# Consumer: housekeeper-agent
consumer:
  stream_name: GRC-DRIFT
  name: housekeeper-agent
  durable_name: housekeeper-agent
  filter_subject: "kubric.*.grc.drift.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 120000000000      # 2 minutes — remediation scripts can be slow
  max_deliver: 5
  max_ack_pending: 200
  replay_policy: instant
  deliver_group: housekeeper-workers
  flow_control: true
  idle_heartbeat: 15000000000

# Consumer: drift-remediation-workflow
consumer:
  stream_name: GRC-DRIFT
  name: drift-remediation-workflow
  durable_name: drift-remediation-workflow
  filter_subject: "kubric.*.grc.drift.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 500
  replay_policy: instant

# Consumer: audit-log-grc
consumer:
  stream_name: GRC-DRIFT
  name: audit-log-grc
  durable_name: audit-log-grc
  filter_subject: "kubric.*.grc.drift.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000
  max_deliver: 10             # Audit writes must be as reliable as possible
  max_ack_pending: 1000
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/grc/drift/v1",
  "title": "DriftEvent",
  "description": "Compliance-control drift event from the GRC compliance service.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id", "asset_id",
    "control_id", "framework", "expected_value", "actual_value",
    "severity", "detected_at"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":  { "type": "string", "format": "uuid" },
    "tenant_id": { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "asset_id":  { "type": "string", "pattern": "^[a-z0-9_-]{3,128}$" },
    "control_id": { "type": "string", "maxLength": 64,
                    "description": "Framework-specific control ID (e.g. CIS-1.1.1, AC-2, CC6.1)." },
    "control_title": { "type": "string", "maxLength": 512 },
    "framework": {
      "type": "string",
      "enum": ["cis", "nist_800_53", "soc2", "iso_27001", "pci_dss", "hipaa"]
    },
    "framework_version": { "type": "string", "maxLength": 32 },
    "expected_value": { "type": "string", "maxLength": 4096,
                        "description": "Required config value per policy. JSON-encoded for structured values." },
    "actual_value":   { "type": "string", "maxLength": 4096,
                        "description": "Observed value on asset. JSON-encoded for structured values." },
    "severity": {
      "type": "string",
      "enum": ["info", "low", "medium", "high", "critical"]
    },
    "detected_at":      { "type": "string", "format": "date-time" },
    "first_detected_at": { "type": "string", "format": "date-time",
                           "description": "When this exact drift condition was first observed. Used for SLA age tracking." },
    "check_method": {
      "type": "string",
      "enum": ["agent_poll", "api_query", "configuration_scan", "manual"]
    },
    "remediation_script": { "type": "string", "maxLength": 256,
                             "description": "Relative path in `config/grc/remediation/`. Null if no automated remediation." },
    "auto_remediable": { "type": "boolean" },
    "remediation_risk": {
      "type": "string",
      "enum": ["none", "low", "medium", "high"]
    },
    "asset_criticality": {
      "type": "string",
      "enum": ["low", "medium", "high", "critical"]
    },
    "tags": { "type": "array", "items": { "type": "string" } }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "a1b2c3d4-e5f6-7890-abcd-000000000001",
  "tenant_id": "acme_corp",
  "asset_id": "linux-server-web-01",
  "control_id": "CIS-5.2.1",
  "control_title": "Ensure SSH MaxAuthTries is set to 4 or less",
  "framework": "cis",
  "framework_version": "CIS Linux v3.0",
  "expected_value": "4",
  "actual_value": "6",
  "severity": "medium",
  "detected_at": "2026-02-26T06:00:00.000Z",
  "first_detected_at": "2026-02-20T06:00:00.000Z",
  "check_method": "agent_poll",
  "remediation_script": "cis/linux/5.2.1_ssh_maxauthtries.sh",
  "auto_remediable": true,
  "remediation_risk": "low",
  "asset_criticality": "high",
  "tags": ["ssh", "access_control", "authentication"]
}
```

NATS message headers:

```
Nats-Msg-Id: a1b2c3d4-e5f6-7890-abcd-000000000001
Content-Type: application/json
X-Kubric-Schema: grc.drift.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Framework: cis
X-Kubric-Control: CIS-5.2.1
X-Kubric-AutoRemediable: true
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/grc/` (Python) |
| Source module | `services/grc/publishers/drift_publisher.py` |
| Transport | NATS JetStream `PublishAsync` |
| Check schedule | Hourly compliance scans; full framework re-evaluation every 24 h |
| Dedup key | `uuid5(NAMESPACE_DNS, "{tenant_id}:{asset_id}:{control_id}:{drift_date}")` — same drift on same day is deduplicated |
| Check engine | SSH/WinRM for network assets; coresec agent API for endpoint assets |
| Publish gate | Only drifted controls (actual != expected) are published; passing controls written directly to ClickHouse |
| Credentials | mTLS + NKey; Vault path `nats/creds/grc/<tenant_id>` |

---

## Consumer Details

### housekeeper-agent

| Field | Value |
|---|---|
| Service | `services/kai/agents/housekeeper_agent.py` |
| Consumer type | Push-based durable, queue group `housekeeper-workers` |
| Processing | When `auto_remediable:true` and `remediation_risk` in `[none, low]` and asset not in `change_freeze`, executes `remediation_script` via coresec agent command API; re-check scheduled at T+15 min |
| Change-freeze skip | Assets tagged `change_freeze` are always skipped; event acked with log entry |
| Approval gate | `critical` criticality + `remediation_risk:high` always requires human approval via change request before execution |

### drift-remediation-workflow

| Field | Value |
|---|---|
| Service | `services/grc/remediation_workflow.py` |
| Consumer type | Push-based durable |
| Processing | Non-auto-remediable drifts routed through approval workflow; creates change request in PSA and notifies asset owner |

### audit-log-grc

| Field | Value |
|---|---|
| Service | `services/audit_log/` (Python) |
| Consumer type | Push-based durable |
| Processing | Writes all drift events to ClickHouse `grc_audit_log` with cryptographic hash chain for tamper evidence |
| Max delivers | 10 — audit log writes are retried aggressively; failure fires critical PagerDuty alert |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 365 days | Full calendar year for annual compliance audit cycles |
| `max_bytes` | 5 GiB | Drift events are low-frequency; supports years of history |
| `discard` | `old` | Oldest records evicted when byte ceiling reached |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent; audit evidence must survive restarts |
| Duplicate window | 1 hour | Prevents same drift reported multiple times in one hourly scan |

**Note:** The `audit-log-grc` consumer writes all events to ClickHouse for indefinite long-term
storage, independent of the 365-day NATS stream retention window.

---

## Error Handling

**Publisher side**

- SSH/WinRM connection failure: asset marked `scan_unreachable` in GRC state table; `kubric.internal.health.degraded` emitted after 2 consecutive failed scans.
- Multi-line `actual_value` output is JSON-encoded before insertion into the string field.

**Consumer side**

- `housekeeper-agent`: remediation failure NAKs; after max_deliver=5, advisory fires and PSA ticket raised at `high` priority.
- `housekeeper-agent`: `change_freeze` assets skipped; event acked with log entry only.
- `drift-remediation-workflow`: PSA unavailability dead-letters to `kubric.<tenant_id>.dlq.grc.drift`.
- `audit-log-grc`: ClickHouse write failure NAKs with 10 retries; on total failure a critical PagerDuty alert fires.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
