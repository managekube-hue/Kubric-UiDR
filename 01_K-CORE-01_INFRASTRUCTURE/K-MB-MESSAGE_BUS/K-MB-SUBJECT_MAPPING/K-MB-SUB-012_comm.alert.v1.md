# K-MB-SUB-012 — comm.alert.v1

---

## 1. Overview

| Field              | Value                                                           |
|--------------------|------------------------------------------------------------------|
| **Subject**        | `kubric.<tenant_id>.comm.alert.v1`                              |
| **Stream**         | `COMM-ALERT`                                                    |
| **Version**        | v1                                                              |
| **Purpose**        | Carries outbound alert notification events dispatched by the communications agent after channel routing. Each message records a single notification attempt — the resolved channel (email, SMS, voice, or internal NATS), the full message body as transmitted, the target recipient, and the delivery status at time of publication. Consumers persist audit trails, track delivery receipts from upstream provider callbacks, and feed the triage acknowledgement listener for SOC workflow automation. |
| **Owner**          | KAI Communications / SOC Workflow                               |
| **SLA**            | At-least-once delivery; critical-priority messages must be persisted by `audit-log-writer` within 30 seconds of publication |

---

## 2. NATS Configuration (YAML)

```yaml
stream:
  name: COMM-ALERT
  subjects:
    - "kubric.*.comm.alert.v1"
  retention: limits
  max_age: 7776000       # 90 days
  max_bytes: 5368709120  # 5 GB
  replicas: 3
  storage: file
  discard: old
  duplicate_window: 120s

consumer:
  name: audit-log-writer
  durable: audit-log-writer
  ack_policy: explicit
  max_deliver: 5
  ack_wait: 30s
  deliver_policy: all
```

> Additional durable consumers `delivery-tracker` and `triage-ack-listener` share this stream with identical ack settings.

---

## 3. Message Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "kubric/comm/alert/v1",
  "title": "CommAlertEvent",
  "type": "object",
  "required": [
    "tenant_id", "alert_id", "channel", "priority",
    "recipient", "message", "sent_at", "delivery_status"
  ],
  "additionalProperties": false,
  "properties": {
    "tenant_id": {
      "type": "string",
      "format": "uuid",
      "description": "UUID v4 of the tenant this notification belongs to."
    },
    "alert_id": {
      "type": "string",
      "format": "uuid",
      "description": "UUID v4 correlating this comm event to the originating security alert."
    },
    "channel": {
      "type": "string",
      "enum": ["email", "sms", "voice", "nats"],
      "description": "Delivery channel used for this notification attempt."
    },
    "priority": {
      "type": "string",
      "enum": ["critical", "high", "medium", "low"],
      "description": "Notification urgency tier, which determines routing and retry escalation."
    },
    "recipient": {
      "type": "string",
      "maxLength": 512,
      "description": "Channel-specific recipient address: email address, E.164 phone number, or NATS reply subject."
    },
    "message": {
      "type": "string",
      "maxLength": 2000,
      "description": "Full notification message body as transmitted to the provider."
    },
    "sent_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp of when the dispatch attempt was made."
    },
    "delivery_status": {
      "type": "string",
      "enum": ["pending", "sent", "delivered", "failed"],
      "description": "Delivery state at time of publication. Updated by delivery-tracker on provider callback."
    },
    "provider_message_id": {
      "type": ["string", "null"],
      "maxLength": 512,
      "description": "Upstream provider message identifier for delivery receipt correlation. Null until provider confirms acceptance."
    }
  }
}
```

---

## 4. Example Payload (JSON)

### 4a — Critical email notification for P1 ransomware alert
```json
{
  "tenant_id": "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9",
  "alert_id": "f1e2d3c4-b5a6-4978-8021-9a8b7c6d5e4f",
  "channel": "email",
  "priority": "critical",
  "recipient": "soc-lead@acme-corp.com",
  "message": "[CRITICAL] Kubric Platform Alert: Ransomware payload detected on HOST-WIN-PROD-042 (10.0.1.42). SHA-256: a3f5c8e1d2b4f769. Immediate isolation required. View details: https://console.kubric.ai/alerts/f1e2d3c4",
  "sent_at": "2026-02-26T02:47:13.221Z",
  "delivery_status": "delivered",
  "provider_message_id": "SENDGRID-MSG-01HX7K9P3VQRMN5FZEJ8"
}
```

### 4b — SMS alert for high-priority authentication anomaly
```json
{
  "tenant_id": "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9",
  "alert_id": "a9b8c7d6-e5f4-4321-b098-765432109fed",
  "channel": "sms",
  "priority": "high",
  "recipient": "+14255551234",
  "message": "KUBRIC ALERT: Brute-force detected on admin@acme-corp.com from 203.0.113.47. 147 failed logins in 60s. Reply ACK to acknowledge.",
  "sent_at": "2026-02-26T09:12:44.089Z",
  "delivery_status": "sent",
  "provider_message_id": "TWILIO-SM-d95f8c3a7b1e4092a53f"
}
```

### 4c — Internal NATS notification to triage agent
```json
{
  "tenant_id": "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9",
  "alert_id": "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e",
  "channel": "nats",
  "priority": "medium",
  "recipient": "kubric.e5f6a7b8.agent.triage.inbound",
  "message": "{\"event\":\"sigma_match\",\"rule\":\"T1003.001_LSASS_Dump\",\"asset\":\"HOST-WIN-DEV-017\",\"severity\":\"medium\"}",
  "sent_at": "2026-02-26T11:30:05.774Z",
  "delivery_status": "delivered",
  "provider_message_id": null
}
```

### 4d — Failed voice call escalation
```json
{
  "tenant_id": "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9",
  "alert_id": "f1e2d3c4-b5a6-4978-8021-9a8b7c6d5e4f",
  "channel": "voice",
  "priority": "critical",
  "recipient": "+12065559876",
  "message": "Kubric critical alert: Ransomware detected. Asset HOST-WIN-PROD-042 requires immediate isolation. Press 1 to acknowledge.",
  "sent_at": "2026-02-26T02:52:00.000Z",
  "delivery_status": "failed",
  "provider_message_id": "TWILIO-CALL-CA3f8e9d7a6b5c4d2e1f"
}
```

---

## 5. Publisher Details

| Attribute          | Value                                                                                                                    |
|--------------------|--------------------------------------------------------------------------------------------------------------------------|
| **Service**        | KAI communications agent                                                                                                  |
| **File**           | `K-KAI-CM-001_comm_agent.py`                                                                                             |
| **Trigger**        | Publishes immediately after routing an alert to the selected delivery channel. Routing decision is made by a priority-channel matrix: critical always fires email+SMS+voice simultaneously; high fires email+SMS; medium fires email only; low fires email with 30-minute delay. |
| **Subject format** | `kubric.{tenant_id}.comm.alert.v1`                                                                                       |
| **Library**        | `nats.py` async JetStream `publish_async`; `Nats-Msg-Id` = `{alert_id}:{channel}:{sent_at_epoch_ms}`                    |
| **Serialisation**  | UTF-8 JSON                                                                                                               |
| **Headers**        | `Kubric-Tenant: {tenant_id}`, `Kubric-Schema-Version: v1`, `Kubric-Priority: {priority}`, `Kubric-Channel: {channel}`   |

The comm-agent publishes one event per channel per alert, so a single critical alert generates up to 3 separate `comm.alert.v1` events (email, SMS, voice). The `alert_id` field links them for correlation.

---

## 6. Consumer Details

### audit-log-writer
| Attribute          | Value                                                                                                          |
|--------------------|----------------------------------------------------------------------------------------------------------------|
| **Role**           | Persists every comm event to Postgres `audit.comm_notifications` table for compliance logging and SOC review. Provides idempotent upsert on `(alert_id, channel, sent_at)` primary key. |
| **Batch size**     | 200 messages per pull                                                                                          |
| **Error behaviour**| NACKs on Postgres write timeout; dead-letters after 5 redeliveries                                            |

### delivery-tracker
| Attribute          | Value                                                                                                          |
|--------------------|----------------------------------------------------------------------------------------------------------------|
| **Role**           | Subscribes to provider delivery receipt webhooks (SendGrid, Twilio) via a separate HTTP listener and updates `delivery_status` in Postgres by correlating on `provider_message_id`. Publishes a `comm.delivery_update.v1` event when status transitions to `delivered` or `failed`. |
| **Batch size**     | 100 messages per pull (registers initial records that webhooks later update)                                   |
| **Error behaviour**| NACKs with 10 s delay on webhook registration failure; dead-letters after 5 redeliveries                       |

### triage-ack-listener
| Attribute          | Value                                                                                                          |
|--------------------|----------------------------------------------------------------------------------------------------------------|
| **Role**           | Monitors `delivery_status: delivered` events for channel `sms` and `voice`. When a recipient replies ACK (via Twilio webhook callback), marks the associated alert as acknowledged in Postgres and publishes `kubric.<tenant_id>.triage.ack.v1` to advance the SOC workflow. |
| **Batch size**     | 50 messages per pull                                                                                           |
| **Filter**         | Subject filter: `kubric.*.comm.alert.v1`, message filter on `channel IN (sms, voice)` post-receive           |
| **Error behaviour**| NACKs with 5 s delay; dead-letters after 5 redeliveries                                                       |

---

## 7. Retention Policy

| Parameter            | Value                                                                                     |
|----------------------|-------------------------------------------------------------------------------------------|
| **Stream retention** | `limits`                                                                                  |
| **max_age**          | 7,776,000 seconds (90 days)                                                               |
| **max_bytes**        | 5 GB                                                                                      |
| **Replicas**         | 3                                                                                         |
| **Storage tier**     | `file`                                                                                    |
| **Archive policy**   | Postgres `audit.comm_notifications` table retains all records indefinitely for compliance. Rows are exported to S3 in Parquet format on a monthly schedule via `pg_cron`. |

---

## 8. Error Handling

| Scenario                              | Behaviour                                                                                                                                      |
|---------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------|
| **Provider API failure (SendGrid/Twilio down)** | comm-agent retries delivery with exponential backoff (base 5 s, max 300 s). After 3 failed attempts on `critical` priority, automatically escalates to the next channel in the priority-channel matrix. |
| **NATS publish failure**              | comm-agent queues the event in a local SQLite fallback store and retries on the next NATS reconnect (max 30-minute window). Prometheus counter `comm_publish_failures_total` increments. |
| **Schema validation failure**         | Invalid messages forwarded immediately to `kubric.<tenant_id>.dlq.comm.alert.v1` with `Kubric-Error-Reason: SCHEMA_INVALID`. Never retried. |
| **Max redeliveries exceeded**         | NATS routes to `kubric.<tenant_id>.dlq.comm.alert.v1`. PagerDuty alert fires when DLQ depth > 5 for `critical` priority messages. |
| **Message body >2000 chars**          | comm-agent truncates the body to 1997 characters and appends `...` before publishing. Original full body is stored in the Postgres audit record. |
| **All escalation channels exhausted** | comm-agent emits `kubric.<tenant_id>.ops.comm_escalation_failed.v1` which triggers a PagerDuty incident directly via the PagerDuty Events API. |
| **Dead-letter subject**               | `kubric.<tenant_id>.dlq.comm.alert.v1`                                                                                                         |
