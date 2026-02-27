# K-MB-SUB-011 — ti.ioc.v1

---

## 1. Overview

| Field              | Value                                                           |
|--------------------|------------------------------------------------------------------|
| **Subject**        | `kubric.<tenant_id>.ti.ioc.v1`                                  |
| **Stream**         | `TI-IOC`                                                        |
| **Version**        | v1                                                              |
| **Purpose**        | Carries enriched Indicators of Compromise sourced from tenant-accessible threat intelligence feeds. Each message represents a single IOC with MISP enrichment metadata: confidence score, TLP classification, descriptive tags, source attribution, temporal validity window, and optional MISP Galaxy Cluster mapping to MITRE ATT&CK or ransomware families. Downstream consumers enforce real-time network blocklists, augment SIEM correlation rules, and drive threat hunting pivot queries against the telemetry data lake. |
| **Owner**          | KAI Threat Intelligence / Investigations                         |
| **SLA**            | At-least-once delivery; TLP:red IOCs must reach `blocklist-publisher` within 60 seconds of publication |

---

## 2. NATS Configuration (YAML)

```yaml
stream:
  name: TI-IOC
  subjects:
    - "kubric.*.ti.ioc.v1"
  retention: limits
  max_age: 2592000        # 30 days — standard IOC viability window
  max_bytes: 21474836480  # 20 GB
  replicas: 3
  storage: file
  discard: old
  duplicate_window: 120s

consumer:
  name: hunter-agent
  durable: hunter-agent
  ack_policy: explicit
  max_deliver: 5
  ack_wait: 30s
  deliver_policy: all
```

> Additional durable consumers `siem-forwarder` and `blocklist-publisher` share this stream. `blocklist-publisher` uses `deliver_policy: new` to process only fresh IOCs rather than replaying history on restart.

---

## 3. Message Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "kubric/ti/ioc/v1",
  "title": "ThreatIntelIOCEvent",
  "type": "object",
  "required": [
    "tenant_id", "ioc_type", "value", "confidence",
    "tlp", "tags", "source", "first_seen", "last_seen"
  ],
  "additionalProperties": false,
  "properties": {
    "tenant_id": {
      "type": "string",
      "format": "uuid",
      "description": "UUID v4 of the tenant this IOC is scoped to."
    },
    "ioc_type": {
      "type": "string",
      "enum": ["ip", "domain", "hash_md5", "hash_sha256", "url", "email"],
      "description": "Indicator category defining how the value should be matched."
    },
    "value": {
      "type": "string",
      "minLength": 1,
      "maxLength": 2048,
      "description": "Raw indicator value, e.g. an IP address, domain name, or hex hash string."
    },
    "confidence": {
      "type": "integer",
      "minimum": 0,
      "maximum": 100,
      "description": "Publisher confidence score: 0=unknown, 50=low-fidelity open source, 90+=confirmed by multiple independent sources."
    },
    "tlp": {
      "type": "string",
      "enum": ["white", "green", "amber", "red"],
      "description": "Traffic Light Protocol classification governing redistribution scope."
    },
    "tags": {
      "type": "array",
      "items": {"type": "string", "maxLength": 128},
      "description": "Free-form classification tags, e.g. ['ransomware', 'cobalt-strike', 'c2']."
    },
    "source": {
      "type": "string",
      "maxLength": 256,
      "description": "Feed or intelligence source identifier, e.g. 'MISP-Community', 'AlienVault-OTX', 'internal-hunt'."
    },
    "first_seen": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp of first confirmed observation."
    },
    "last_seen": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp of most recent confirmation of indicator activity."
    },
    "misp_galaxy_cluster": {
      "type": ["string", "null"],
      "maxLength": 512,
      "description": "MISP Galaxy Cluster label, e.g. 'mitre-attack:T1059.001 - PowerShell' or 'ransomware:LockBit 3.0'. Null if no cluster applies."
    }
  }
}
```

---

## 4. Example Payload (JSON)

### 4a — Cobalt Strike C2 IP from MISP community feed
```json
{
  "tenant_id": "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f",
  "ioc_type": "ip",
  "value": "185.220.101.45",
  "confidence": 92,
  "tlp": "amber",
  "tags": ["cobalt-strike", "c2", "threatgroup:TA505", "malware:beacon"],
  "source": "MISP-Community",
  "first_seen": "2026-02-20T08:14:33.000Z",
  "last_seen": "2026-02-26T12:00:01.000Z",
  "misp_galaxy_cluster": "mitre-attack:T1071.001 - Application Layer Protocol: Web Protocols"
}
```

### 4b — LockBit 3.0 dropper (SHA-256), TLP:red, internally confirmed
```json
{
  "tenant_id": "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f",
  "ioc_type": "hash_sha256",
  "value": "a3f5c8e1d2b4f769ac1e3d5b7f2a4c6e8d0b2f4a6c8e0d2b4f6a8c0e2d4f6a8",
  "confidence": 99,
  "tlp": "red",
  "tags": ["lockbit3", "ransomware", "dropper", "stage1"],
  "source": "internal-hunt",
  "first_seen": "2026-02-25T22:47:10.000Z",
  "last_seen": "2026-02-25T22:47:10.000Z",
  "misp_galaxy_cluster": "ransomware:LockBit 3.0"
}
```

### 4c — Phishing domain targeting finance sector, TLP:green
```json
{
  "tenant_id": "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f",
  "ioc_type": "domain",
  "value": "secure-login-portal.kubric-phish.xyz",
  "confidence": 78,
  "tlp": "green",
  "tags": ["phishing", "credential-harvest", "industry:finance", "kit:evilginx2"],
  "source": "AlienVault-OTX",
  "first_seen": "2026-02-24T14:00:00.000Z",
  "last_seen": "2026-02-26T09:32:17.000Z",
  "misp_galaxy_cluster": null
}
```

### 4d — Malicious URL delivering macro-enabled document
```json
{
  "tenant_id": "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f",
  "ioc_type": "url",
  "value": "https://cdn.malicious-delivery.ru/docs/invoice_2026_02.xlsm",
  "confidence": 85,
  "tlp": "amber",
  "tags": ["macro-dropper", "initial-access", "spearphish-attachment"],
  "source": "Mandiant-Advantage",
  "first_seen": "2026-02-23T16:20:00.000Z",
  "last_seen": "2026-02-26T10:45:00.000Z",
  "misp_galaxy_cluster": "mitre-attack:T1566.001 - Phishing: Spearphishing Attachment"
}
```

---

## 5. Publisher Details

| Attribute          | Value                                                                                                                     |
|--------------------|---------------------------------------------------------------------------------------------------------------------------|
| **Service**        | KAI Investigations MISP galaxy query enrichment loop                                                                       |
| **File**           | `K-KAI-IV-001_misp_galaxy_query.py`                                                                                       |
| **Trigger**        | Continuous polling loop with a 60-second base interval against MISP REST API (`/attributes/restSearch`). Also receives on-demand trigger via NATS request/reply from the hunter agent. |
| **Subject format** | `kubric.{tenant_id}.ti.ioc.v1`                                                                                            |
| **Library**        | `nats.py` async JetStream `publish_async`; `Nats-Msg-Id` = `{tenant_id}:{ioc_type}:{sha256(value)[:16]}:{last_seen_epoch}` |
| **Serialisation**  | UTF-8 JSON                                                                                                                |
| **Headers**        | `Kubric-Tenant: {tenant_id}`, `Kubric-Schema-Version: v1`, `Kubric-TLP: {tlp}`, `Kubric-IOC-Type: {ioc_type}`            |

The `Nats-Msg-Id` hash deduplicates within the 120-second window. A Redis set `ti:seen:{tenant_id}` with 30-day TTL handles cross-window deduplication for IOCs that reappear in different polling cycles.

---

## 6. Consumer Details

### hunter-agent
| Attribute          | Value                                                                                                      |
|--------------------|------------------------------------------------------------------------------------------------------------|
| **Role**           | Submits pivot queries against ClickHouse `edr.process_events`, `ndr.flow_events`, and `edr.file_events` for each received IOC. Matches are published to `kubric.<tenant_id>.security.alert.v1` with `type: threat_intel`. |
| **Batch size**     | 25 messages per pull (latency-optimised for TLP:red)                                                       |
| **Error behaviour**| NACKs on ClickHouse query timeout >5 s; dead-letters after 5 redeliveries                                  |

### siem-forwarder
| Attribute          | Value                                                                                                      |
|--------------------|------------------------------------------------------------------------------------------------------------|
| **Role**           | Translates each IOC to CEF/LEEF format and forwards to the tenant SIEM via syslog-TLS or REST. TLP:red indicators are never forwarded to external SIEM instances. Only TLP:white and TLP:green may be forwarded without restriction. |
| **Batch size**     | 200 messages per pull                                                                                      |
| **Error behaviour**| NACKs with 15 s delay when SIEM endpoint is unreachable; dead-letters after 5 redeliveries                 |

### blocklist-publisher
| Attribute          | Value                                                                                                      |
|--------------------|------------------------------------------------------------------------------------------------------------|
| **Role**           | For `ioc_type: ip` and `ioc_type: domain` with `confidence >= 70`, pushes entries to pfSense pfBlockerNG and Fortinet FortiGate dynamic address group APIs. Maintains a dedicated fast-path goroutine for TLP:red entries to satisfy the 60-second SLA. Latency tracked via `ti_blocklist_publish_latency_seconds` histogram. |
| **Batch size**     | 50 messages per pull                                                                                       |
| **Error behaviour**| NACKs with 5 s delay on firewall API errors; dead-letters after 5 redeliveries                             |

---

## 7. Retention Policy

| Parameter            | Value                                                                                                   |
|----------------------|---------------------------------------------------------------------------------------------------------|
| **Stream retention** | `limits`                                                                                                |
| **max_age**          | 2,592,000 seconds (30 days) — aligns with standard external feed IOC expiry windows                     |
| **max_bytes**        | 20 GB — high-volume feeds can produce millions of IOC updates per month                                  |
| **Replicas**         | 3                                                                                                       |
| **Storage tier**     | `file`                                                                                                  |
| **Archive policy**   | ClickHouse sink consumer `ti-ch-archive` (separate durable) streams all events to `ti.ioc_history`. ClickHouse TTL tiers rows older than 1 year to S3-backed cold storage, retained indefinitely for retrospective hunt queries. |

---

## 8. Error Handling

| Scenario                              | Behaviour                                                                                                                                          |
|---------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| **MISP API unreachable**              | Publisher backs off with exponential retry (base 5 s, max 300 s). Prometheus counter `ti_misp_poll_failures_total` increments; PagerDuty critical alert at 5 consecutive failures. |
| **TLP:red delivery SLA breach**       | `blocklist-publisher` tracks publish-to-consume latency via `Nats-Time-Stamp` header. If latency > 90 s, emits `kubric.ops.ti.sla_breach.v1` and sends Slack `#soc-critical` alert with IOC value redacted to first octet. |
| **Schema validation failure**         | Invalid messages forwarded immediately to `kubric.<tenant_id>.dlq.ti.ioc.v1` with `Kubric-Error-Reason: SCHEMA_INVALID`. Never retried. |
| **Max redeliveries exceeded**         | NATS routes to `kubric.<tenant_id>.dlq.ti.ioc.v1`. PagerDuty alert at gauge `ti_dlq_depth{tenant}` > 20. |
| **Cross-window duplicate IOC**        | Redis `SETNX ti:seen:{tenant_id}:{msg_id}` with 30-day TTL prevents duplicate hunt pivots or duplicate blocklist entries. |
| **Confidence below blocklist threshold** | `blocklist-publisher` silently skips IOCs with `confidence < 70` and increments `ti_blocklist_skipped_low_confidence_total`. IOCs are still processed by `hunter-agent` for pivot context. |
| **Dead-letter subject**               | `kubric.<tenant_id>.dlq.ti.ioc.v1`                                                                                                                 |
