# K-MB-SUB-006 — vdr.vuln.v1

## Overview

This subject carries vulnerability-discovery events produced by the **VDR** (Vulnerability Detection
and Response) service. The service integrates with Nuclei templates (network-layer scanning) and
Tenable.io / Tenable SC (agent-based host scanning). Each discovered vulnerability produces a
`VulnEvent` referencing the asset, CVE, CVSS v3 score, EPSS probability, and KEV catalog status.

**Subject pattern:** `kubric.<tenant_id>.vdr.vuln.v1`

**Example resolved subject:** `kubric.acme_corp.vdr.vuln.v1`

Multi-tenant wildcard: `kubric.*.vdr.vuln.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: VDR-VULN — apply with: nats stream add --config vdr-vuln-stream.yaml
stream:
  name: VDR-VULN
  subjects:
    - "kubric.*.vdr.vuln.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 15552000000000000  # 180 days in nanoseconds
  max_bytes: 10737418240      # 10 GiB
  max_msg_size: 32768         # 32 KiB per message
  max_msgs: -1
  discard: old
  duplicate_window: 3600000000000  # 1-hour dedup (same CVE/asset within 1 h = duplicate)

# Consumer: risk-agent-vuln
consumer:
  stream_name: VDR-VULN
  name: risk-agent-vuln
  durable_name: risk-agent-vuln
  filter_subject: "kubric.*.vdr.vuln.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 5
  max_ack_pending: 500
  replay_policy: instant
  deliver_group: risk-workers
  flow_control: true
  idle_heartbeat: 15000000000

# Consumer: ssvc-decision-engine
consumer:
  stream_name: VDR-VULN
  name: ssvc-decision-engine
  durable_name: ssvc-decision-engine
  filter_subject: "kubric.*.vdr.vuln.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000
  max_deliver: 3
  max_ack_pending: 1000
  replay_policy: instant

# Consumer: patch-workflow
consumer:
  stream_name: VDR-VULN
  name: patch-workflow
  durable_name: patch-workflow
  filter_subject: "kubric.*.vdr.vuln.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 5
  max_ack_pending: 200
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/vdr/vuln/v1",
  "title": "VulnEvent",
  "description": "Vulnerability-discovery event from Nuclei or Tenable integration.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id", "asset_id",
    "cve_id", "cvss_v3", "epss_score", "severity",
    "remediation_available", "scan_timestamp", "scanner"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":  { "type": "string", "format": "uuid" },
    "tenant_id": { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "asset_id":  { "type": "string", "pattern": "^[a-z0-9_-]{3,128}$" },
    "cve_id": {
      "type": "string", "pattern": "^CVE-[0-9]{4}-[0-9]{4,}$",
      "description": "CVE identifier in canonical form."
    },
    "cvss_v3":  { "type": "number", "minimum": 0.0, "maximum": 10.0 },
    "cvss_vector": { "type": "string", "maxLength": 128,
                     "description": "Full CVSS v3.x vector string." },
    "epss_score": { "type": "number", "minimum": 0.0, "maximum": 1.0,
                    "description": "EPSS exploit probability as of scan date." },
    "epss_percentile": { "type": "number", "minimum": 0.0, "maximum": 100.0 },
    "severity": {
      "type": "string",
      "enum": ["info", "low", "medium", "high", "critical"]
    },
    "port":    { "type": "integer", "minimum": 0, "maximum": 65535 },
    "service": { "type": "string", "maxLength": 128 },
    "plugin_id": { "type": "string", "maxLength": 64,
                   "description": "Tenable plugin ID or Nuclei template ID." },
    "remediation_available": { "type": "boolean" },
    "remediation_notes": { "type": "string", "maxLength": 2048 },
    "kev_listed":  { "type": "boolean",
                     "description": "True if CVE appears in CISA KEV catalog." },
    "kev_due_date": { "type": "string", "format": "date",
                      "description": "CISA KEV remediation deadline (ISO date)." },
    "kev_stale":   { "type": "boolean",
                     "description": "True when KEV catalog cache is older than 24 hours." },
    "scan_timestamp": { "type": "string", "format": "date-time" },
    "scanner": {
      "type": "string",
      "enum": ["nuclei", "tenable_io", "tenable_sc", "openvas", "manual"]
    },
    "scanner_version": { "type": "string", "maxLength": 64 },
    "first_seen": { "type": "string", "format": "date-time",
                    "description": "When this CVE was first observed on this asset." },
    "asset_criticality": {
      "type": "string",
      "enum": ["low", "medium", "high", "critical"]
    }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "f9a0b1c2-d3e4-5678-f012-34567890abcd",
  "tenant_id": "acme_corp",
  "asset_id": "web-server-prod-01",
  "cve_id": "CVE-2024-3094",
  "cvss_v3": 10.0,
  "cvss_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
  "epss_score": 0.9731,
  "epss_percentile": 99.8,
  "severity": "critical",
  "port": 22,
  "service": "OpenSSH_9.6p1 embedded XZ liblzma",
  "plugin_id": "nuclei:xz-utils-backdoor-detection",
  "remediation_available": true,
  "remediation_notes": "Upgrade xz-utils to >= 5.6.2. Distro patches available for all major Linux distributions.",
  "kev_listed": true,
  "kev_due_date": "2024-04-19",
  "kev_stale": false,
  "scan_timestamp": "2026-02-26T08:00:00.000Z",
  "scanner": "nuclei",
  "scanner_version": "3.2.4",
  "first_seen": "2026-02-24T08:00:00.000Z",
  "asset_criticality": "critical"
}
```

NATS message headers:

```
Nats-Msg-Id: f9a0b1c2-d3e4-5678-f012-34567890abcd
Content-Type: application/json
X-Kubric-Schema: vdr.vuln.v1
X-Kubric-Tenant: acme_corp
X-Kubric-CVE: CVE-2024-3094
X-Kubric-CVSS: 10.0
X-Kubric-KEV: true
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/vdr/` (Python) |
| Source module | `services/vdr/publishers/vuln_publisher.py` |
| Transport | NATS JetStream `PublishAsync` with `Nats-Msg-Id` header |
| Scan scheduling | Nuclei scans every 24 h per tenant; Tenable scans driven by Tenable API schedule |
| EPSS enrichment | Fetched daily from `https://api.first.org/data/v1/epss`; cached in Redis |
| KEV enrichment | CISA KEV catalog fetched daily; cached locally |
| Dedup key | `uuid5(NAMESPACE_DNS, "{tenant_id}:{asset_id}:{cve_id}:{scan_date}")` — same CVE/asset/day = same UUID |
| Credentials | mTLS + NKey; Vault path `nats/creds/vdr/<tenant_id>` |

---

## Consumer Details

### risk-agent-vuln

| Field | Value |
|---|---|
| Service | `services/kai/agents/risk_agent.py` |
| Consumer type | Push-based durable, queue group `risk-workers` |
| Processing | Composite risk score = CVSS v3 x EPSS x asset criticality x age-in-environment; written to ClickHouse `vuln_risk`; score >= 8.5 + `kev_listed:true` escalates to `kubric.<tenant_id>.security.alert.v1` |

### ssvc-decision-engine

| Field | Value |
|---|---|
| Service | `services/vdr/ssvc/` (Python) |
| Consumer type | Push-based durable |
| Processing | CISA SSVC decision tree: exploitation status, automatable, technical impact; outputs `track`, `track-closely`, `attend`, or `act`; results to ClickHouse `ssvc_decisions` |
| Exploitation lookup failure | Defaults to `exploitation:poc` (worst-case) to avoid under-prioritising |

### patch-workflow

| Field | Value |
|---|---|
| Service | `services/patch_workflow/` (Python) |
| Consumer type | Push-based durable |
| Processing | `critical` + `remediation_available:true` + `kev_listed:true` creates patch task in PSA and publishes to `kubric.<tenant_id>.svc.ticket.v1` |
| PSA failure | After max_deliver failures, event written to `kubric.<tenant_id>.dlq.patch.workflow` |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 180 days | Vulnerability age trend analysis and compliance evidence |
| `max_bytes` | 10 GiB | Low-volume (thousands per scan cycle) |
| `discard` | `old` | Oldest findings evicted under pressure |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent |
| Duplicate window | 1 hour | Same CVE/asset pair not processed twice in one scan run |

---

## Error Handling

**Publisher side**

- EPSS lookup failure: `epss_score:null` published; event is not blocked.
- KEV refresh failure: stale cache used up to 24 h; `kev_stale:true` added to payload.
- Nuclei scan timeout for an asset: asset flagged in `vdr_scan_coverage_gaps` metric.

**Consumer side**

- `risk-agent-vuln`: ClickHouse write failure NAKs with 30 s backoff; after max_deliver=5, advisory fires.
- `ssvc-decision-engine`: NVD API down defaults to `poc` exploitation status.
- `patch-workflow`: PSA unavailability dead-letters to `kubric.<tenant_id>.dlq.patch.workflow`.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
