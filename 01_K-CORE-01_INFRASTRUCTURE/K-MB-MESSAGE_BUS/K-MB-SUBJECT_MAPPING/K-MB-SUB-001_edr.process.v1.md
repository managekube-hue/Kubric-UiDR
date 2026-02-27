# K-MB-SUB-001 — edr.process.v1

## Overview

This subject carries process-execution telemetry emitted by the **coresec** Rust-based EDR sensor
running on every managed endpoint. Every time a process is created, a `ProcessEvent` message is
published to this subject. Downstream systems use it for behavioural triage, SIEM forwarding, and
ML anomaly-detection model training.

**Subject pattern:** `kubric.<tenant_id>.edr.process.v1`

**Example resolved subject:** `kubric.acme_corp.edr.process.v1`

The `<tenant_id>` token is a lowercase alphanumeric slug (3-64 chars) assigned at tenant
provisioning time. Multi-tenant consumers subscribe via wildcard `kubric.*.edr.process.v1`.

---

## NATS Configuration (YAML)

```yaml
# Stream: EDR-PROCESS — apply with: nats stream add --config edr-process-stream.yaml
stream:
  name: EDR-PROCESS
  subjects:
    - "kubric.*.edr.process.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 604800000000000    # 7 days in nanoseconds
  max_bytes: 53687091200      # 50 GiB
  max_msg_size: 65536         # 64 KiB per message
  max_msgs: -1
  discard: old
  duplicate_window: 120000000000  # 2-minute dedup window

# Consumer: triage-consumer (CrewAI Triage Agent)
consumer:
  stream_name: EDR-PROCESS
  name: triage-consumer
  durable_name: triage-consumer
  filter_subject: "kubric.*.edr.process.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000       # 30 seconds
  max_deliver: 5
  max_ack_pending: 1000
  replay_policy: instant
  deliver_group: triage-workers
  flow_control: true
  idle_heartbeat: 5000000000

# Consumer: siem-forwarder-edr-process
consumer:
  stream_name: EDR-PROCESS
  name: siem-forwarder-edr-process
  durable_name: siem-forwarder-edr-process
  filter_subject: "kubric.*.edr.process.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 5000
  replay_policy: instant

# Consumer: ml-pipeline-edr-process
consumer:
  stream_name: EDR-PROCESS
  name: ml-pipeline-edr-process
  durable_name: ml-pipeline-edr-process
  filter_subject: "kubric.*.edr.process.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 120000000000      # 2 minutes for batch feature-extraction
  max_deliver: 3
  max_ack_pending: 10000
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/edr/process/v1",
  "title": "ProcessEvent",
  "description": "Process-execution event captured by the coresec EDR sensor.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id", "asset_id", "timestamp",
    "pid", "ppid", "executable", "cmdline", "user", "sha256", "parent_executable"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id": {
      "type": "string", "format": "uuid",
      "description": "UUID v4; also used as Nats-Msg-Id for stream deduplication."
    },
    "tenant_id":  { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "asset_id":   { "type": "string", "pattern": "^[a-z0-9_-]{3,128}$" },
    "timestamp":  { "type": "string", "format": "date-time" },
    "pid":        { "type": "integer", "minimum": 1,  "maximum": 4194304 },
    "ppid":       { "type": "integer", "minimum": 0,  "maximum": 4194304,
                    "description": "Parent PID. 0 = kernel-spawned or orphaned." },
    "executable": { "type": "string", "maxLength": 4096 },
    "cmdline":    { "type": "string", "maxLength": 32768 },
    "user":       { "type": "string", "maxLength": 256 },
    "sha256":     { "type": "string", "pattern": "^[a-f0-9]{64}$" },
    "parent_executable": { "type": "string", "maxLength": 4096 },
    "integrity_level": {
      "type": "string",
      "enum": ["untrusted", "low", "medium", "high", "system"],
      "description": "Windows integrity level. Omitted on non-Windows platforms."
    },
    "signed":   { "type": "boolean" },
    "signer":   { "type": "string", "maxLength": 512 },
    "truncated": {
      "type": "boolean", "default": false,
      "description": "True when cmdline was truncated to fit max_msg_size."
    },
    "env_vars": {
      "type": "object",
      "additionalProperties": { "type": "string" },
      "description": "Env var subset captured per sensor policy (PATH, TEMP, etc.)."
    },
    "sensor_version": { "type": "string" }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "tenant_id": "acme_corp",
  "asset_id": "win-endpoint-0042",
  "timestamp": "2026-02-26T14:23:07.841Z",
  "pid": 4821,
  "ppid": 1024,
  "executable": "C:\Windows\System32\cmd.exe",
  "cmdline": "cmd.exe /c whoami && net localgroup administrators",
  "user": "ACMECORP\jsmith",
  "sha256": "b94f6f125c79e3a5ffaa826f584c10d52ada669e6762051b826b55776d05a8ad",
  "parent_executable": "C:\Windows\explorer.exe",
  "integrity_level": "medium",
  "signed": true,
  "signer": "Microsoft Windows",
  "truncated": false,
  "env_vars": {
    "PATH": "C:\Windows\System32;C:\Windows",
    "TEMP": "C:\Users\jsmith\AppData\Local\Temp"
  },
  "sensor_version": "2.4.1"
}
```

NATS message headers:

```
Nats-Msg-Id: f47ac10b-58cc-4372-a567-0e02b2c3d479
Content-Type: application/json
X-Kubric-Schema: edr.process.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Asset: win-endpoint-0042
X-Kubric-Sensor: coresec/2.4.1
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/coresec` (Rust) |
| Source module | `services/coresec/src/publishers/process_publisher.rs` |
| Transport | NATS JetStream `PublishAsync` with `Nats-Msg-Id` header |
| Peak rate | Up to 5,000 events/sec per tenant under heavy endpoint activity |
| Batching | Events published individually; no client-side batching |
| Offline buffer | 50,000-event in-memory ring buffer; tail-drop after 30 s unavailability; persistent fallback to `/var/lib/coresec/wal.db` (SQLite WAL) |
| TLS | mTLS with per-sensor client cert issued by Kubric internal CA |
| Credentials | NATS NKey per sensor; rotated every 24 h via Vault path `nats/creds/coresec/<asset_id>` |
| Oversized messages | `cmdline` truncated to 16,384 chars; `truncated:true` appended before re-publishing |

---

## Consumer Details

### triage-consumer — CrewAI Triage Agent

| Field | Value |
|---|---|
| Service | `services/kai/agents/triage_agent.py` |
| Consumer type | Push-based durable, queue group `triage-workers` |
| Ack policy | Explicit; `msg.ack()` called after triage crew finishes |
| Max delivers | 5 — on exhaustion, JetStream MAX_DELIVERIES advisory fires |
| Processing | Score >= 0.7 triggers `SecurityAlert` to `kubric.<tenant_id>.security.alert.v1` |
| Parallelism | Up to 10 worker replicas per queue group |

### siem-forwarder-edr-process

| Field | Value |
|---|---|
| Service | `services/siem_forwarder/` |
| Consumer type | Push-based durable |
| Ack policy | Explicit |
| Processing | Transforms to CEF/JSON; POSTs to tenant SIEM endpoint in `config/siem/<tenant_id>.yaml` |
| Retry | HTTP 5xx: NAK with 10 s backoff (doubling); >3 failures dead-letter to `kubric.<tenant_id>.dlq.siem.edr.process` |

### ml-pipeline-edr-process

| Field | Value |
|---|---|
| Service | `services/ml/` (Python) |
| Consumer type | Push-based durable |
| Ack policy | Explicit |
| Processing | 60 s windowed feature batches for process-lineage anomaly detection; results to ClickHouse `ml_scores` |
| Circuit breaker | Pauses 60 s on ClickHouse write p99 > 5 s for 10 consecutive messages |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 7 days | Full triage backlog replay SLA |
| `max_bytes` | 50 GiB | ~20 M events at avg 2.5 KiB |
| `max_msgs` | Unlimited | Governed by age/byte caps |
| `discard` | `old` | Oldest evicted first under pressure |
| Replicas | 3 | Tolerates one JetStream node loss |
| Storage | File | Persistent across restarts |
| Duplicate window | 2 minutes | Suppresses sensor retransmit duplicates |

---

## Error Handling

**Publisher side**

- `503 No Responders` (stream unavailable): sensor writes to SQLite WAL; background task replays on
  reconnect in arrival order.
- Message > 64 KiB: `cmdline` truncated to 16,384 chars; `truncated:true` set.
- All publish errors increment `coresec_nats_publish_errors_total` Prometheus counter on `:9090/metrics`.

**Consumer side**

- `triage-consumer`: panics NAK with exponential backoff from 5 s; exhausted deliveries surface via
  `$JS.EVENT.ADVISORY.>` advisory consumed by the NOC agent.
- `siem-forwarder`: dead-letter to `kubric.<tenant_id>.dlq.siem.edr.process`.
- `ml-pipeline`: circuit-breaker suspends consumption 60 s on sustained ClickHouse latency.

**Schema validation**

Consumers validate JSON via `jsonschema` (Python) or `serde` (Rust) before processing. Invalid
messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
