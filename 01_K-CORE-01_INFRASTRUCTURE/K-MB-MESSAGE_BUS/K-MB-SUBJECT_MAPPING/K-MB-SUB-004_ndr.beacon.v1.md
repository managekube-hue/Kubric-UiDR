# K-MB-SUB-004 — ndr.beacon.v1

## Overview

This subject carries beacon-detection results produced by the RITA analyzer inside **netguard**. A
`BeaconEvent` is published when RITA identifies a (src_ip, dst_ip) pair exhibiting statistically
periodic connection behaviour consistent with C2 callback traffic. Consumers are the KAI hunter
agent, the incident manager, and the KAI orchestrator.

**Subject pattern:** `kubric.<tenant_id>.ndr.beacon.v1`

**Example resolved subject:** `kubric.acme_corp.ndr.beacon.v1`

Multi-tenant wildcard: `kubric.*.ndr.beacon.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: NDR-BEACON — apply with: nats stream add --config ndr-beacon-stream.yaml
stream:
  name: NDR-BEACON
  subjects:
    - "kubric.*.ndr.beacon.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 2592000000000000   # 30 days in nanoseconds
  max_msgs: 1000000           # 1 million beacon records maximum
  max_bytes: -1               # governed by message count and age
  max_msg_size: 16384         # 16 KiB per message
  discard: old
  duplicate_window: 300000000000  # 5-minute dedup window (beacons re-scored hourly)

# Consumer: hunter-agent-beacon
consumer:
  stream_name: NDR-BEACON
  name: hunter-agent-beacon
  durable_name: hunter-agent-beacon
  filter_subject: "kubric.*.ndr.beacon.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 5
  max_ack_pending: 1000
  replay_policy: instant
  deliver_group: hunter-workers
  flow_control: true
  idle_heartbeat: 10000000000

# Consumer: incident-manager-beacon
consumer:
  stream_name: NDR-BEACON
  name: incident-manager-beacon
  durable_name: incident-manager-beacon
  filter_subject: "kubric.*.ndr.beacon.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000
  max_deliver: 5
  max_ack_pending: 500
  replay_policy: instant

# Consumer: kai-orchestrator-beacon
consumer:
  stream_name: NDR-BEACON
  name: kai-orchestrator-beacon
  durable_name: kai-orchestrator-beacon
  filter_subject: "kubric.*.ndr.beacon.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 200
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/ndr/beacon/v1",
  "title": "BeaconEvent",
  "description": "C2 beacon-detection result produced by RITA analysis of NDR flow data.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id",
    "src_ip", "dst_ip", "score", "connection_count",
    "jitter_ms", "interval_ms", "protocol",
    "analysis_window_start", "analysis_window_end"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":  { "type": "string", "format": "uuid" },
    "tenant_id": { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "src_ip":    { "type": "string", "format": "ipv4",
                   "description": "Internal host IP suspected of beaconing." },
    "dst_ip":    { "type": "string", "format": "ipv4",
                   "description": "External destination receiving beacon traffic." },
    "dst_port":  { "type": "integer", "minimum": 0, "maximum": 65535 },
    "protocol":  { "type": "string", "enum": ["tcp", "udp", "icmp", "dns", "other"] },
    "score": {
      "type": "number", "minimum": 0.0, "maximum": 1.0,
      "description": "RITA beacon score. >= 0.85 is high confidence."
    },
    "connection_count": { "type": "integer", "minimum": 1 },
    "interval_ms": { "type": "integer", "minimum": 0,
                     "description": "Modal interval between connections in milliseconds." },
    "jitter_ms":   { "type": "integer", "minimum": 0,
                     "description": "Std deviation of connection interval. Low = automated traffic." },
    "bytes_per_connection_avg": { "type": "number", "minimum": 0 },
    "duration_avg_ms":          { "type": "number", "minimum": 0 },
    "analysis_window_start": { "type": "string", "format": "date-time" },
    "analysis_window_end":   { "type": "string", "format": "date-time" },
    "dst_hostname": { "type": "string", "maxLength": 253 },
    "geo_dst": {
      "type": "object",
      "properties": {
        "country": { "type": "string", "maxLength": 2 },
        "asn":     { "type": "integer" },
        "org":     { "type": "string" }
      }
    },
    "threat_intel_match": { "type": "boolean" },
    "threat_intel_tags":  { "type": "array", "items": { "type": "string" } },
    "rita_version": { "type": "string" }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "d5e6f7a8-b9c0-1234-def0-1234567890ab",
  "tenant_id": "acme_corp",
  "src_ip": "10.0.2.101",
  "dst_ip": "185.220.101.47",
  "dst_port": 443,
  "protocol": "tcp",
  "score": 0.94,
  "connection_count": 288,
  "interval_ms": 300000,
  "jitter_ms": 1200,
  "bytes_per_connection_avg": 1024.5,
  "duration_avg_ms": 450,
  "analysis_window_start": "2026-02-26T13:00:00.000Z",
  "analysis_window_end":   "2026-02-26T14:00:00.000Z",
  "dst_hostname": "gate.darkleaf-c2.net",
  "geo_dst": { "country": "NL", "asn": 209763, "org": "Hosting Services AS209763" },
  "threat_intel_match": true,
  "threat_intel_tags": ["c2", "cobalt_strike"],
  "rita_version": "4.7.1"
}
```

NATS message headers:

```
Nats-Msg-Id: d5e6f7a8-b9c0-1234-def0-1234567890ab
Content-Type: application/json
X-Kubric-Schema: ndr.beacon.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Score: 0.94
X-Kubric-TI-Match: true
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/netguard/rita/` (Go) |
| Source module | `services/netguard/rita/beacon_publisher.go` |
| Transport | NATS JetStream `Publish` (synchronous — beacons are critical) |
| Trigger | Published at end of each 1-hour RITA window; only if `score >= 0.75` |
| Dedup key | Same (tenant_id, src_ip, dst_ip, dst_port) within 5-minute window is suppressed |
| TI enrichment | `dst_ip` checked against `services/ti/` IOC API before publish; failures default to `threat_intel_match:false` + tag `__ti_lookup_failed` |
| Credentials | mTLS + NKey; Vault path `nats/creds/netguard/rita/<sensor_id>` |

---

## Consumer Details

### hunter-agent-beacon

| Field | Value |
|---|---|
| Service | `services/kai/agents/hunter_agent.py` |
| Consumer type | Push-based durable, queue group `hunter-workers` |
| Processing | score >= 0.85 triggers automated threat-hunting runbook via CrewAI; results correlated with EDR-PROCESS events; confirmed threats published to `kubric.<tenant_id>.security.alert.v1` |
| Max delivers | 5 |

### incident-manager-beacon

| Field | Value |
|---|---|
| Service | `services/incident_manager/` (Python) |
| Consumer type | Push-based durable |
| Processing | `score >= 0.9` or `threat_intel_match:true` auto-opens incident and notifies on-call via `kubric.<tenant_id>.comm.alert.v1` |
| Fallback | PagerDuty webhook triggered directly if incident tracker API is unavailable after max_deliver=5 |

### kai-orchestrator-beacon

| Field | Value |
|---|---|
| Service | `services/kai/orchestrator.py` |
| Processing | Decides whether to dispatch full multi-agent investigation crew based on score, asset criticality, and tenant response plan |
| Low-score handling | score 0.75-0.84 logged to ClickHouse `beacon_log`; no active alert raised |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Message count + age ceiling |
| `max_age` | 30 days | Trend analysis and re-investigation window |
| `max_msgs` | 1,000,000 | Beacons are low-volume; practical count cap |
| `max_bytes` | Unlimited | Governed by age and count caps |
| `discard` | `old` | Oldest records evicted when limit reached |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent |
| Duplicate window | 5 minutes | Suppresses hourly re-publication of unchanged scores |

---

## Error Handling

**Publisher side**

- RITA MongoDB query error: analysis window retried once after 5 minutes; on second failure `kubric.internal.health.degraded` emitted.
- TI lookup failure: `threat_intel_match:false`; `threat_intel_tags:["__ti_lookup_failed"]` added.

**Consumer side**

- `hunter-agent-beacon`: EDR-PROCESS correlation timeout escalates beacon as network-only alert without process context.
- `incident-manager-beacon`: API unavailability causes NAK with 30 s backoff; after max_deliver=5 a PagerDuty fallback webhook fires.
- `kai-orchestrator-beacon`: scores 0.75-0.84 written to ClickHouse `beacon_log`; no active alert.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
