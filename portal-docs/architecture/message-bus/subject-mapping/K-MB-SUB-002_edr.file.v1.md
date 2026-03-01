# K-MB-SUB-002 — EDR File Activity (FIM)

> NATS subject mapping reference for File Integrity Monitoring telemetry in the Kubric UIDR platform.

## Subject Pattern

```
kubric.edr.file.v1
kubric.edr.file.v1.<tenant_id>
kubric.edr.file.v1.<tenant_id>.<host_id>
```

Wildcard subscription: `kubric.edr.file.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **CoreSec FIM** | Rust | File Integrity Monitoring module within the CoreSec agent. Watches configured paths via `inotify` (Linux) / `ReadDirectoryChangesW` (Windows) and computes BLAKE3 + SHA-256 hashes on every create, modify, delete, or permission change. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-TRIAGE** | Python (CrewAI) | Correlates file changes with process events to build causal chains (e.g., which process dropped a binary). Triggers alerts on suspicious file writes in sensitive directories. |
| **KIC** | Go | Kubric Integrity & Compliance engine. Compares file state against known-good baselines and compliance frameworks (CIS, STIG). Drift events from FIM feed directly into GRC compliance posture. |

## Payload (OCSF Class)

- **OCSF Class**: File Activity (`4008`)
- **OCSF Category**: System Activity
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `path` | `string` | `file.path` | Full absolute path to the file. |
| 2 | `filename` | `string` | `file.name` | Base filename extracted from path. |
| 3 | `size` | `uint64` | `file.size` | File size in bytes after the operation. |
| 4 | `action` | `uint32` | `activity_id` | OCSF activity: 1=Create, 2=Read, 3=Update, 4=Delete, 5=Rename, 6=SetAttributes. |
| 5 | `hash_blake3` | `string` | `file.hashes[].value` (algorithm=BLAKE3) | BLAKE3 hash of file contents (fast, used for internal dedup). |
| 6 | `hash_sha256` | `string` | `file.hashes[].value` (algorithm=SHA-256) | SHA-256 hash of file contents (used for IOC matching and compliance). |
| 7 | `user` | `string` | `actor.user.name` | Username of the account that performed the file operation. |
| 8 | `mode` | `string` | `file.attributes` | Unix file permissions as octal string (e.g., `"0644"`). On Windows, maps to ACL summary. |
| 9 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier. |
| 10 | `host_id` | `string` | `device.uid` | Unique host identifier assigned by CoreSec. |
| 11 | `timestamp_ns` | `uint64` | `time` | Event timestamp in nanoseconds since Unix epoch. |
| 12 | `prev_hash_sha256` | `string` | `file.hashes[].value` (prev) | SHA-256 hash before modification (empty on create). Used for diff detection. |
| 13 | `severity_id` | `uint32` | `severity_id` | OCSF severity: 0=Unknown through 5=Critical. |

## JetStream Configuration

FIM file events share the **KUBRIC_EDR** JetStream stream with process events (K-MB-SUB-001). Both subjects fall under the `kubric.edr.>` wildcard.

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
    "duplicate_window": "2m"
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_EDR` | **Shared stream** — process (`kubric.edr.process.>`) and file (`kubric.edr.file.>`) events coexist in the same stream. This simplifies stream management and enables cross-subject replay. |
| `subjects` | `kubric.edr.>` | Single wildcard captures all current and future EDR event types. |
| `max_age` | `7d` | Raw FIM events retained for 7 days. KIC snapshots baseline state to object storage for long-term compliance evidence. |
| `max_bytes` | `50 GB` | Combined budget for all EDR events. FIM events are typically smaller than process events but can spike during large deployments or `apt upgrade`. |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-triage-file` | Yes | `kubric.edr.file.v1.>` | `all` | `explicit` | 5 | 30s |
| `kic-fim-drift` | Yes | `kubric.edr.file.v1.>` | `all` | `explicit` | 5 | 60s |
| `siem-export-edr` | Yes | `kubric.edr.>` | `all` | `explicit` | 3 | 60s |

- **kai-triage-file**: Correlates file events with process telemetry. When a file write occurs in `/tmp`, `/dev/shm`, or user home directories, KAI-TRIAGE cross-references the writing PID against the process stream to build full attack narratives.
- **kic-fim-drift**: KIC compares each file event against the declared baseline. Deviations generate `kubric.grc.drift.v1` events (see K-MB-SUB-007) that feed the compliance remediation pipeline.
- **siem-export-edr**: Shared consumer across all EDR events for external SIEM forwarding.

## Compliance Correlation Use Case

```
FIM Event (kubric.edr.file.v1)
    |
    v
KIC Baseline Comparison
    |
    +-- Match baseline  --> No action
    |
    +-- Drift detected  --> Publish kubric.grc.drift.v1
                                |
                                v
                           KAI-KEEPER
                                |
                                +-- Auto-remediate (SaltStack/Ansible)
                                +-- Create compliance finding
                                +-- Notify via kubric.svc.ticket.v1
```

**Supported frameworks**: CIS Benchmarks (Level 1 & 2), DISA STIG, PCI-DSS (Req 11.5), HIPAA (164.312(c)(1)), SOC 2 Type II (CC6.1).

KIC maintains per-host baselines in its local state store. On each FIM event:
1. Hash is compared against the expected value for that path.
2. If the hash differs and the change is not in an approved change window, a drift event is emitted.
3. KAI-KEEPER evaluates drift severity and triggers remediation or creates an exception request.

## Example (NATS CLI)

### Publish a test FIM event

```bash
nats pub kubric.edr.file.v1.tenant-acme.host-01 \
  --header="Nats-Msg-Id:fim-$(uuidgen)" \
  '{"path":"/etc/passwd","filename":"passwd","size":2847,"action":3,"hash_blake3":"a1b2c3...","hash_sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","user":"root","mode":"0644","tenant_id":"tenant-acme","host_id":"host-01","timestamp_ns":1718900000000000000,"severity_id":4}'
```

### Subscribe to FIM events only

```bash
nats sub "kubric.edr.file.>"
```

### Create the KIC consumer

```bash
nats consumer add KUBRIC_EDR kic-fim-drift \
  --filter="kubric.edr.file.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=60s \
  --pull \
  --durable="kic-fim-drift"
```

### Check consumer lag

```bash
nats consumer info KUBRIC_EDR kic-fim-drift
```

## Notes

- **Shared stream**: File events share the `KUBRIC_EDR` stream with process events. Consumer filter subjects isolate the two event types. This avoids stream proliferation while keeping logical separation at the consumer level.
- **Hash strategy**: BLAKE3 is computed first (fast, ~3 GB/s on modern CPUs) for internal deduplication and change detection. SHA-256 is computed in parallel for IOC matching against threat intelligence feeds and compliance evidence.
- **Large file handling**: Files larger than 100 MB are hashed asynchronously. The initial FIM event is published with `hash_blake3=""` and `hash_sha256=""`, followed by a second event with populated hashes once computation completes. The `Nats-Msg-Id` differentiates the two.
- **Excluded paths**: CoreSec FIM excludes noisy paths by default (`/proc`, `/sys`, `/var/log/journal`, temp build dirs). Exclusion lists are configurable per-tenant via the CoreSec policy pushed from the Kubric control plane.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
