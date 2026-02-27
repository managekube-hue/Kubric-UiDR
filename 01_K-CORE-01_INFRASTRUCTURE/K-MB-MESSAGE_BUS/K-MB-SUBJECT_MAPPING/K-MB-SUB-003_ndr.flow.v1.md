# K-MB-SUB-003 — ndr.flow.v1

## Overview

This subject carries network-flow telemetry produced by the **netguard** Go service. A `FlowEvent`
represents a completed or sampled bidirectional network conversation (5-tuple plus metadata). These
events feed the RITA beacon-detection analyzer, the KAI hunter agent for threat hunting, and the
SIEM for network-layer visibility.

**Subject pattern:** `kubric.<tenant_id>.ndr.flow.v1`

**Example resolved subject:** `kubric.acme_corp.ndr.flow.v1`

Multi-tenant wildcard: `kubric.*.ndr.flow.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: NDR-FLOW — apply with: nats stream add --config ndr-flow-stream.yaml
stream:
  name: NDR-FLOW
  subjects:
    - "kubric.*.ndr.flow.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 259200000000000    # 3 days in nanoseconds
  max_bytes: 107374182400     # 100 GiB (flow telemetry is the highest-volume stream)
  max_msg_size: 8192          # 8 KiB — flows are compact
  max_msgs: -1
  discard: old
  duplicate_window: 60000000000  # 1-minute dedup window

# Consumer: rita-beacon-analyzer
consumer:
  stream_name: NDR-FLOW
  name: rita-beacon-analyzer
  durable_name: rita-beacon-analyzer
  filter_subject: "kubric.*.ndr.flow.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000
  max_deliver: 3
  max_ack_pending: 20000
  replay_policy: instant
  deliver_group: rita-workers
  flow_control: true
  idle_heartbeat: 5000000000

# Consumer: hunter-agent-ndr-flow
consumer:
  stream_name: NDR-FLOW
  name: hunter-agent-ndr-flow
  durable_name: hunter-agent-ndr-flow
  filter_subject: "kubric.*.ndr.flow.v1"
  deliver_policy: last_per_subject
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 5000
  replay_policy: instant

# Consumer: siem-forwarder-ndr-flow
consumer:
  stream_name: NDR-FLOW
  name: siem-forwarder-ndr-flow
  durable_name: siem-forwarder-ndr-flow
  filter_subject: "kubric.*.ndr.flow.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 10000
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/ndr/flow/v1",
  "title": "FlowEvent",
  "description": "Completed or sampled network-flow event produced by netguard.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id", "timestamp",
    "src_ip", "dst_ip", "src_port", "dst_port", "protocol",
    "bytes_in", "bytes_out", "duration_ms", "direction"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":  { "type": "string", "format": "uuid" },
    "tenant_id": { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "timestamp": { "type": "string", "format": "date-time",
                   "description": "ISO 8601 UTC timestamp of flow completion or last sample." },
    "src_ip":    { "type": "string", "format": "ipv4" },
    "dst_ip":    { "type": "string", "format": "ipv4" },
    "src_port":  { "type": "integer", "minimum": 0, "maximum": 65535 },
    "dst_port":  { "type": "integer", "minimum": 0, "maximum": 65535 },
    "protocol":  { "type": "string", "enum": ["tcp", "udp", "icmp", "other"] },
    "bytes_in":  { "type": "integer", "minimum": 0 },
    "bytes_out": { "type": "integer", "minimum": 0 },
    "packets_in":  { "type": "integer", "minimum": 0 },
    "packets_out": { "type": "integer", "minimum": 0 },
    "duration_ms": { "type": "integer", "minimum": 0,
                     "description": "Flow duration in milliseconds." },
    "direction": {
      "type": "string",
      "enum": ["inbound", "outbound", "lateral", "unknown"],
      "description": "Direction relative to monitored network boundary."
    },
    "tcp_flags": { "type": "string", "pattern": "^[FSRPAUEC]{0,8}$" },
    "vlan_id":   { "type": "integer", "minimum": 0, "maximum": 4094 },
    "sensor_id": { "type": "string" },
    "sampled":   { "type": "boolean",
                   "description": "True when 1-in-N sampling was active on the capture link." },
    "geo_src": {
      "type": "object",
      "properties": {
        "country": { "type": "string", "maxLength": 2 },
        "asn":     { "type": "integer" },
        "org":     { "type": "string" }
      }
    },
    "geo_dst": {
      "type": "object",
      "properties": {
        "country": { "type": "string", "maxLength": 2 },
        "asn":     { "type": "integer" },
        "org":     { "type": "string" }
      }
    }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "c3d4e5f6-a7b8-9012-cdef-1234567890ab",
  "tenant_id": "acme_corp",
  "timestamp": "2026-02-26T15:00:00.000Z",
  "src_ip": "10.0.1.55",
  "dst_ip": "185.220.101.47",
  "src_port": 52341,
  "dst_port": 443,
  "protocol": "tcp",
  "bytes_in": 512,
  "bytes_out": 4096,
  "packets_in": 8,
  "packets_out": 12,
  "duration_ms": 3200,
  "direction": "outbound",
  "tcp_flags": "SFPA",
  "vlan_id": 100,
  "sensor_id": "netguard-probe-01",
  "sampled": false,
  "geo_src": { "country": "US", "asn": 64512, "org": "ACME Corp Internal" },
  "geo_dst": { "country": "NL", "asn": 209763, "org": "Tor Exit Node AS209763" }
}
```

NATS message headers:

```
Nats-Msg-Id: c3d4e5f6-a7b8-9012-cdef-1234567890ab
Content-Type: application/json
X-Kubric-Schema: ndr.flow.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Sensor: netguard-probe-01
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/netguard` (Go) |
| Source module | `services/netguard/publisher/flow_publisher.go` |
| Transport | NATS JetStream `PublishAsync` |
| Flow source | AF_PACKET/XDP capture or NetFlow v9/IPFIX collector |
| Peak rate | Up to 50,000 events/sec for a busy 1 Gbps link segment |
| Sampling | For links > 1 Gbps, 1-in-10 sampling applies; `sampled:true` field added |
| GeoIP enrichment | MaxMind GeoLite2, refreshed weekly; applied in-process before publish |
| Credentials | mTLS + NKey; Vault path `nats/creds/netguard/<sensor_id>` |

---

## Consumer Details

### rita-beacon-analyzer

| Field | Value |
|---|---|
| Service | `services/netguard/rita/` (Go) |
| Consumer type | Push-based durable, queue group `rita-workers` |
| Processing | Flows buffered into 1-hour analysis windows per (src_ip, dst_ip, dst_port) tuple; RITA scores run at window end; beacons >= 0.85 published to `kubric.<tenant_id>.ndr.beacon.v1` |
| Ack policy | Explicit; acked after flow committed to RITA MongoDB buffer |

### hunter-agent-ndr-flow

| Field | Value |
|---|---|
| Service | `services/kai/agents/hunter_agent.py` |
| Consumer type | Push-based durable |
| Processing | Correlates flows with IOC watchlist; hits escalate to `kubric.<tenant_id>.security.alert.v1` |

### siem-forwarder-ndr-flow

| Field | Value |
|---|---|
| Service | `services/siem_forwarder/` |
| Processing | Transforms to LEEF/JSON; forwards to tenant SIEM |
| Dead-letter | `kubric.<tenant_id>.dlq.siem.ndr.flow` |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 3 days | RITA 1 h windows; 3 days covers incident scope replay |
| `max_bytes` | 100 GiB | Highest-volume stream on the bus |
| `discard` | `old` | Oldest flows evicted under pressure |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent |
| Duplicate window | 1 minute | Shorter than EDR; probe retransmits are less common |

---

## Error Handling

**Publisher side**

- Private RFC-1918 addresses are published without `geo_src`/`geo_dst` (normal for lateral flows).
- NATS unavailability: in-memory ring buffer (200,000 flows) with tail-drop; `netguard_nats_dropped_flows_total` tracks drops.
- Malformed NetFlow/IPFIX packets discarded and counted in `netguard_malformed_netflow_total`.

**Consumer side**

- `rita-beacon-analyzer`: MongoDB write failures cause NAK; after `max_deliver=3` advisory fires.
- `hunter-agent-ndr-flow`: stale watchlist tolerated up to 5 minutes before `kubric.internal.health.degraded` fires.
- `siem-forwarder`: dead-letter `kubric.<tenant_id>.dlq.siem.ndr.flow`.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
