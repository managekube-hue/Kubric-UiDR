# K-MB-SUB-015 — asset.provisioned.v1

---

## 1. Overview

| Field              | Value                                                           |
|--------------------|------------------------------------------------------------------|
| **Subject**        | `kubric.<tenant_id>.asset.provisioned.v1`                       |
| **Stream**         | `ASSET-EVENTS`                                                  |
| **Version**        | v1                                                              |
| **Purpose**        | Carries the full lifecycle of managed assets in the Kubric CMDB. Each message represents a provisioning, update, or deprovisioning event for a single asset (server, workstation, network device, cloud VM, container, IoT, or mobile device). It is the authoritative source of truth for asset inventory across the platform: the housekeeper agent uses it to schedule vulnerability scans and drift checks, billing-clerk uses it for seat counting, the vulnerability scanner scheduler uses it to route scan jobs, and the CMDB writer synchronises records to the asset database. |
| **Owner**          | NOC Inventory / Platform Engineering                            |
| **SLA**            | At-least-once delivery; `provisioned` and `deprovisioned` events must be reflected in the CMDB within 60 seconds of publication |

---

## 2. NATS Configuration (YAML)

```yaml
stream:
  name: ASSET-EVENTS
  subjects:
    - "kubric.*.asset.provisioned.v1"
  retention: limits
  max_age: 0            # unlimited — asset lineage must be preserved indefinitely
  max_msgs: 10000000    # 10 million messages per stream (approx 50 bytes avg = 500 MB)
  replicas: 3
  storage: file
  discard: old
  duplicate_window: 120s

consumer:
  name: housekeeper-agent
  durable: housekeeper-agent
  ack_policy: explicit
  max_deliver: 5
  ack_wait: 30s
  deliver_policy: all
```

> Additional durable consumers `billing-clerk`, `vuln-scanner-scheduler`, and `asset-cmdb-writer` share this stream with identical ack settings.

---

## 3. Message Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "kubric/asset/provisioned/v1",
  "title": "AssetProvisionedEvent",
  "type": "object",
  "required": [
    "tenant_id", "asset_id", "hostname", "ip_address",
    "os_family", "os_version", "asset_type", "criticality_tier",
    "managed_by", "tags", "provisioned_at", "event_type"
  ],
  "additionalProperties": false,
  "properties": {
    "tenant_id": { "type": "string", "format": "uuid" },
    "asset_id": { "type": "string", "format": "uuid" },
    "hostname": { "type": "string", "maxLength": 253 },
    "ip_address": {
      "type": "string",
      "description": "Primary IPv4 or IPv6 address."
    },
    "os_family": {
      "type": "string",
      "enum": ["windows", "linux", "macos", "network_device", "cloud_vm", "container"]
    },
    "os_version": {
      "type": "string",
      "maxLength": 128,
      "description": "Full OS version string, e.g. 'Windows Server 2022 21H2' or 'Ubuntu 22.04.3 LTS'."
    },
    "asset_type": {
      "type": "string",
      "enum": ["server", "workstation", "network", "cloud_vm", "iot", "mobile"]
    },
    "criticality_tier": {
      "type": "integer",
      "enum": [1, 2, 3, 4, 5],
      "description": "Asset criticality: 1=crown-jewel/business-critical, 5=non-production/test."
    },
    "managed_by": {
      "type": "string",
      "maxLength": 128,
      "description": "Team or system responsible, e.g. 'noc-team', 'platform-engineering', 'intune-mdm'."
    },
    "mac_address": {
      "type": ["string", "null"],
      "pattern": "^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$",
      "description": "Primary MAC address. Null for cloud VMs and containers."
    },
    "serial_number": {
      "type": ["string", "null"],
      "maxLength": 128,
      "description": "Hardware serial number. Null for VMs and containers."
    },
    "cloud_provider": {
      "type": ["string", "null"],
      "enum": ["aws", "azure", "gcp", null],
      "description": "Cloud hyperscaler for cloud_vm and container types. Null for physical assets."
    },
    "tags": {
      "type": "object",
      "additionalProperties": {"type": "string"},
      "description": "Free-form key-value metadata, e.g. {\"env\": \"production\", \"app\": \"payments\", \"cost_center\": \"CC-1042\"}."
    },
    "provisioned_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp when asset was first provisioned."
    },
    "deprovisioned_at": {
      "type": ["string", "null"],
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp when asset was decommissioned. Null if still active."
    },
    "event_type": {
      "type": "string",
      "enum": ["provisioned", "updated", "deprovisioned"],
      "description": "Type of asset lifecycle event represented by this message."
    }
  }
}
```

---

## 4. Example Payload (JSON)

### 4a — Windows production server initially provisioned
```json
{
  "tenant_id": "e6f7a8b9-c0d1-4e2f-b3a4-c5d6e7f8a9b0",
  "asset_id": "cc3344ff-5566-4778-99aa-bbccddeeff00",
  "hostname": "HOST-WIN-PROD-042.acme-corp.internal",
  "ip_address": "10.0.1.42",
  "os_family": "windows",
  "os_version": "Windows Server 2022 21H2 (Build 20348.2340)",
  "asset_type": "server",
  "criticality_tier": 1,
  "managed_by": "platform-engineering",
  "mac_address": "00:1A:2B:3C:4D:5E",
  "serial_number": "MXQ91502VH",
  "cloud_provider": null,
  "tags": {
    "env": "production",
    "app": "payments-api",
    "cost_center": "CC-1042",
    "datacenter": "DC-EUW1",
    "patch_ring": "ring-2"
  },
  "provisioned_at": "2025-11-15T08:30:00.000Z",
  "deprovisioned_at": null,
  "event_type": "provisioned"
}
```

### 4b — AWS cloud VM updated (new IP after elastic IP reassignment)
```json
{
  "tenant_id": "e6f7a8b9-c0d1-4e2f-b3a4-c5d6e7f8a9b0",
  "asset_id": "aa1122bb-3344-4556-7788-99aabbccddee",
  "hostname": "ip-172-31-24-88.eu-west-1.compute.internal",
  "ip_address": "172.31.24.88",
  "os_family": "cloud_vm",
  "os_version": "Amazon Linux 2023 (kernel 6.1.79)",
  "asset_type": "cloud_vm",
  "criticality_tier": 2,
  "managed_by": "devops-team",
  "mac_address": null,
  "serial_number": null,
  "cloud_provider": "aws",
  "tags": {
    "env": "staging",
    "aws:instance-id": "i-0a1b2c3d4e5f67890",
    "aws:region": "eu-west-1",
    "app": "data-pipeline",
    "cost_center": "CC-2017"
  },
  "provisioned_at": "2026-01-10T14:22:00.000Z",
  "deprovisioned_at": null,
  "event_type": "updated"
}
```

### 4c — macOS workstation enrolled via Jamf MDM
```json
{
  "tenant_id": "e6f7a8b9-c0d1-4e2f-b3a4-c5d6e7f8a9b0",
  "asset_id": "bb2233cc-ddee-4ff0-a112-233445566778",
  "hostname": "MBP-JSMITH-2024.local",
  "ip_address": "10.10.5.214",
  "os_family": "macos",
  "os_version": "macOS Sequoia 15.2 (24C101)",
  "asset_type": "workstation",
  "criticality_tier": 3,
  "managed_by": "jamf-mdm",
  "mac_address": "F8:63:3F:A1:B2:C3",
  "serial_number": "C02Z1234ABCD",
  "cloud_provider": null,
  "tags": {
    "env": "corporate",
    "department": "engineering",
    "user": "jsmith",
    "mdm": "jamf-pro",
    "patch_ring": "ring-4"
  },
  "provisioned_at": "2026-02-10T09:15:00.000Z",
  "deprovisioned_at": null,
  "event_type": "provisioned"
}
```

### 4d — Deprovisioned network device (switch retired)
```json
{
  "tenant_id": "e6f7a8b9-c0d1-4e2f-b3a4-c5d6e7f8a9b0",
  "asset_id": "99aabb00-ccdd-4eef-f112-233445566778",
  "hostname": "SW-CORE-DC1-02.acme-corp.internal",
  "ip_address": "10.0.0.5",
  "os_family": "network_device",
  "os_version": "Cisco IOS-XE 17.12.1",
  "asset_type": "network",
  "criticality_tier": 1,
  "managed_by": "noc-team",
  "mac_address": "A8:B4:56:12:34:56",
  "serial_number": "FHH2309K01J",
  "cloud_provider": null,
  "tags": {
    "env": "production",
    "datacenter": "DC-EUW1",
    "role": "core-switch",
    "vendor": "cisco",
    "cost_center": "CC-1001"
  },
  "provisioned_at": "2018-06-20T00:00:00.000Z",
  "deprovisioned_at": "2026-02-26T16:00:00.000Z",
  "event_type": "deprovisioned"
}
```

---

## 5. Publisher Details

| Attribute          | Value                                                                                                                              |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------|
| **Service A**      | NOC inventory service                                                                                                              |
| **Service B**      | MDM integration (Microsoft Intune / Jamf webhook consumer)                                                                         |
| **Trigger A**      | NOC inventory service publishes on manual asset import, network discovery scan completion, or CMDB record update via the NOC REST API |
| **Trigger B**      | Intune/Jamf webhook receiver publishes on device enrollment, compliance status change, or device retirement                         |
| **Subject format** | `kubric.{tenant_id}.asset.provisioned.v1`                                                                                          |
| **Library**        | `nats.py` async JetStream `publish_async`; `Nats-Msg-Id` = `{tenant_id}:{asset_id}:{event_type}:{provisioned_at_epoch_ms}`        |
| **Serialisation**  | UTF-8 JSON                                                                                                                         |
| **Headers**        | `Kubric-Tenant: {tenant_id}`, `Kubric-Schema-Version: v1`, `Kubric-Asset-Type: {asset_type}`, `Kubric-Event-Type: {event_type}`   |

---

## 6. Consumer Details

### housekeeper-agent
| Attribute          | Value                                                                                                              |
|--------------------|--------------------------------------------------------------------------------------------------------------------|
| **Role**           | On `event_type: provisioned`, schedules an initial vulnerability scan job for the new asset within 4 hours. On `event_type: deprovisioned`, cancels any pending scan/drift tasks and marks the asset as inactive in the vulnerability schedule. Registers the asset in the configuration drift baseline. |
| **Batch size**     | 100 messages per pull                                                                                              |
| **Error behaviour**| NACKs on scan scheduler write failure; dead-letters after 5 redeliveries                                          |

### billing-clerk
| Attribute          | Value                                                                                                              |
|--------------------|--------------------------------------------------------------------------------------------------------------------|
| **Role**           | Counts `agent_seat` billable units by tracking active managed assets per tenant. On `provisioned`, increments the seat counter. On `deprovisioned`, decrements it. Uses the Redis counter `billing:seats:{tenant_id}` as the authoritative count for monthly invoice generation. |
| **Batch size**     | 500 messages per pull                                                                                              |
| **Error behaviour**| NACKs on Redis INCR/DECR failure; dead-letters after 5 redeliveries                                               |

### vuln-scanner-scheduler
| Attribute          | Value                                                                                                              |
|--------------------|--------------------------------------------------------------------------------------------------------------------|
| **Role**           | Maintains a priority queue of assets due for vulnerability scanning, weighted by `criticality_tier`. Tier-1 assets are scanned every 24 hours; Tier-5 every 14 days. On `updated` events, re-evaluates scan priority if `criticality_tier` or `os_family` changed. |
| **Batch size**     | 200 messages per pull                                                                                              |
| **Error behaviour**| NACKs on Postgres priority queue write failure; dead-letters after 5 redeliveries                                  |

### asset-cmdb-writer
| Attribute          | Value                                                                                                              |
|--------------------|--------------------------------------------------------------------------------------------------------------------|
| **Role**           | Upserts asset records into Postgres `cmdb.assets` table on `provisioned` and `updated` events. Soft-deletes (sets `is_active=false`, populates `deprovisioned_at`) on `deprovisioned` events. Maintains full change history in `cmdb.asset_history` audit table. |
| **Batch size**     | 200 messages per pull                                                                                              |
| **Error behaviour**| NACKs on Postgres constraint violation or write timeout; dead-letters after 5 redeliveries                         |

---

## 7. Retention Policy

| Parameter            | Value                                                                                                  |
|----------------------|--------------------------------------------------------------------------------------------------------|
| **Stream retention** | `limits`                                                                                               |
| **max_age**          | 0 (unlimited) — full asset lineage must be preserved for compliance and forensic investigations        |
| **max_msgs**         | 10,000,000 — approximately 500 MB at 50-byte average message size; discard oldest when exceeded        |
| **Replicas**         | 3                                                                                                      |
| **Storage tier**     | `file`                                                                                                 |
| **Archive policy**   | Postgres `cmdb.assets` and `cmdb.asset_history` tables provide the primary persistent store. ClickHouse `cmdb.asset_events` table retains all raw events for large-scale analytics queries (fleet-wide OS version distribution, criticality tier changes over time). ClickHouse rows are never deleted. |

---

## 8. Error Handling

| Scenario                                    | Behaviour                                                                                                                                    |
|---------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| **NOC inventory API outage**                | The NOC service queues asset events locally in SQLite with a 24-hour TTL and replays them on reconnect. Prometheus counter `asset_inventory_publish_failures_total` increments. PagerDuty alert at 10 consecutive failures. |
| **Intune/Jamf webhook replay**              | MDM webhook receiver is idempotent on `asset_id + event_type + provisioned_at`. Duplicate webhooks within the `duplicate_window` are silently discarded by NATS deduplication. |
| **Schema validation failure**               | Invalid messages forwarded immediately to `kubric.<tenant_id>.dlq.asset.provisioned.v1` with `Kubric-Error-Reason: SCHEMA_INVALID`. Never retried. |
| **Max redeliveries exceeded**               | NATS routes to `kubric.<tenant_id>.dlq.asset.provisioned.v1`. PagerDuty alert at DLQ depth > 20. |
| **Deprovisioned asset with open tasks**     | housekeeper-agent detects open remediation tasks for the deprovisioned asset (query via Postgres) and automatically cancels them by publishing `status: cancelled` task events to `kubric.<tenant_id>.remediation.task.v1`. |
| **max_msgs limit reached**                  | NATS discards the oldest messages (`discard: old`). asset-cmdb-writer ensures all asset state is persisted to Postgres before NATS messages age out, providing a durable recovery point if replay is needed. |
| **IP address conflict (duplicate asset)**   | asset-cmdb-writer enforces a unique index on `(tenant_id, ip_address)` in Postgres. On conflict, the newer record wins and a `Kubric-Warning: IP_CONFLICT` header is logged to the ops event stream. |
| **Dead-letter subject**                     | `kubric.<tenant_id>.dlq.asset.provisioned.v1`                                                                                                |
