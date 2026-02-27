# K-MB-SUB-002 — edr.file.v1

## Overview

This subject carries file-system activity events emitted by the **coresec** Rust-based EDR sensor.
Each create, modify, delete, or rename operation on a monitored path produces a `FileEvent` message.
Downstream consumers are the YARA scanning worker (which checks file hashes and content against
rule sets) and the SIEM forwarder.

**Subject pattern:** `kubric.<tenant_id>.edr.file.v1`

**Example resolved subject:** `kubric.acme_corp.edr.file.v1`

Multi-tenant wildcard: `kubric.*.edr.file.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: EDR-FILE — apply with: nats stream add --config edr-file-stream.yaml
stream:
  name: EDR-FILE
  subjects:
    - "kubric.*.edr.file.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 604800000000000    # 7 days in nanoseconds
  max_bytes: 32212254720      # 30 GiB (file events are smaller than process events)
  max_msg_size: 32768         # 32 KiB per message
  max_msgs: -1
  discard: old
  duplicate_window: 120000000000  # 2-minute dedup window

# Consumer: yara-scanner
consumer:
  stream_name: EDR-FILE
  name: yara-scanner
  durable_name: yara-scanner
  filter_subject: "kubric.*.edr.file.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000       # 60 s — YARA scan on large files can be slow
  max_deliver: 3
  max_ack_pending: 500
  replay_policy: instant
  deliver_group: yara-workers
  flow_control: true
  idle_heartbeat: 10000000000

# Consumer: siem-forwarder-edr-file
consumer:
  stream_name: EDR-FILE
  name: siem-forwarder-edr-file
  durable_name: siem-forwarder-edr-file
  filter_subject: "kubric.*.edr.file.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 5000
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/edr/file/v1",
  "title": "FileEvent",
  "description": "File-system activity event captured by the coresec EDR sensor.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id", "asset_id",
    "timestamp", "path", "operation", "user", "process_pid"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":  { "type": "string", "format": "uuid" },
    "tenant_id": { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "asset_id":  { "type": "string", "pattern": "^[a-z0-9_-]{3,128}$" },
    "timestamp": { "type": "string", "format": "date-time" },
    "path":      { "type": "string", "maxLength": 4096,
                   "description": "Absolute filesystem path of the affected file." },
    "old_path":  { "type": "string", "maxLength": 4096,
                   "description": "Pre-rename source path. Present only when operation=rename." },
    "operation": {
      "type": "string",
      "enum": ["create", "modify", "delete", "rename"],
      "description": "File-system operation type."
    },
    "sha256": {
      "type": "string", "pattern": "^[a-f0-9]{64}$",
      "description": "SHA-256 of file contents at event time. Null for delete operations."
    },
    "size_bytes": { "type": "integer", "minimum": 0,
                    "description": "File size in bytes at event time. 0 for deletes." },
    "user":       { "type": "string", "maxLength": 256 },
    "process_pid": { "type": "integer", "minimum": 1, "maximum": 4194304 },
    "process_executable": { "type": "string", "maxLength": 4096 },
    "extension":  { "type": "string", "maxLength": 32,
                    "description": "Lowercase file extension without dot (e.g. exe, ps1, docx)." },
    "is_hidden":  { "type": "boolean" },
    "yara_tags":  {
      "type": "array", "items": { "type": "string" },
      "description": "YARA rule tags matched by yara-scanner consumer post-event."
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
  "event_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "acme_corp",
  "asset_id": "win-endpoint-0042",
  "timestamp": "2026-02-26T14:31:55.200Z",
  "path": "C:\Users\jsmith\AppData\Local\Temp\invoice_Q1.exe",
  "old_path": null,
  "operation": "create",
  "sha256": "3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4",
  "size_bytes": 458752,
  "user": "ACMECORP\jsmith",
  "process_pid": 7340,
  "process_executable": "C:\Program Files\Google\Chrome\Application\chrome.exe",
  "extension": "exe",
  "is_hidden": false,
  "yara_tags": [],
  "sensor_version": "2.4.1"
}
```

NATS message headers:

```
Nats-Msg-Id: a1b2c3d4-e5f6-7890-abcd-ef1234567890
Content-Type: application/json
X-Kubric-Schema: edr.file.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Asset: win-endpoint-0042
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/coresec` (Rust) |
| Source module | `services/coresec/src/publishers/file_publisher.rs` |
| Transport | NATS JetStream `PublishAsync` with `Nats-Msg-Id` header |
| Monitored paths | Configured per-tenant in `config/coresec/<tenant_id>/file_monitor_paths.yaml`; defaults: `%TEMP%`, `%APPDATA%`, `C:\Windows\System32`, `/tmp`, `/etc` |
| SHA-256 | Computed synchronously for files <= 100 MiB; omitted for larger files to avoid sensor blocking |
| Peak rate | Up to 2,000 events/sec per tenant during mass file operations |
| Offline buffer | Same SQLite WAL as SubjectId-001 at `/var/lib/coresec/wal.db` |
| TLS / Credentials | mTLS + NKey; Vault path `nats/creds/coresec/<asset_id>` |

---

## Consumer Details

### yara-scanner

| Field | Value |
|---|---|
| Service | `services/yara_scanner/` (Python) |
| Consumer type | Push-based durable, queue group `yara-workers` |
| Ack policy | Explicit |
| Processing | For `operation` in `[create, modify]`, fetches the file from the asset via coresec file-fetch API and runs it against YARA rule set at `config/yara/rules/`. Matches populate `yara_tags` and a `SecurityAlert` is published to `kubric.<tenant_id>.security.alert.v1`. |
| Ack wait | 60 s — allows time to fetch and scan files up to ~50 MiB |
| Max delivers | 3 |
| File deleted before scan | Acked with `yara_tags: ["__file_deleted_before_scan"]`; no alert raised |
| File too large (>200 MiB) | Acked with `yara_tags: ["__file_too_large"]`; metric incremented |

### siem-forwarder-edr-file

| Field | Value |
|---|---|
| Service | `services/siem_forwarder/` |
| Consumer type | Push-based durable |
| Ack policy | Explicit |
| Processing | Transforms to CEF/JSON and POSTs to tenant SIEM endpoint |
| Dead-letter | `kubric.<tenant_id>.dlq.siem.edr.file` after 3 failed HTTP attempts |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 7 days | Matches EDR-PROCESS for correlated investigation window |
| `max_bytes` | 30 GiB | File events are smaller; lower ceiling than process stream |
| `max_msgs` | Unlimited | Governed by age/byte caps |
| `discard` | `old` | Oldest messages evicted first |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent |
| Duplicate window | 2 minutes | Prevents sensor retransmit duplicates |

---

## Error Handling

**Publisher side**

- SHA-256 computation failure (file locked): `sha256` omitted; `"sha256_error": "<reason>"` added. Event still published.
- Paths excluded by sensor policy are silently dropped at the sensor; this is not an error condition.
- NATS unavailability: same SQLite WAL offline buffer as the process publisher.

**Consumer side**

- `yara-scanner`: file deleted before scan — acked with `__file_deleted_before_scan` tag; no alert.
- `yara-scanner`: file > 200 MiB — acked with `__file_too_large`; `yara_scanner_oversized_files_total` incremented.
- `siem-forwarder`: dead-letter `kubric.<tenant_id>.dlq.siem.edr.file` for operator review.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
