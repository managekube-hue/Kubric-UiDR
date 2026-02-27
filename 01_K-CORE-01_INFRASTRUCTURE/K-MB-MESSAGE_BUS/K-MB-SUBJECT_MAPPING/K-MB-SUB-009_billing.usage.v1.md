# K-MB-SUB-009 — billing.usage.v1

---

## 1. Overview

| Field              | Value                                                           |
|--------------------|------------------------------------------------------------------|
| **Subject**        | `kubric.<tenant_id>.billing.usage.v1`                           |
| **Stream**         | `BILLING-USAGE`                                                 |
| **Version**        | v1                                                              |
| **Purpose**        | Carries per-tenant metered usage events for every billable action performed on the Kubric platform. Each message represents one discrete billing unit: an agent seat consumed, an event ingested into the pipeline, an ML inference call, or a storage delta. The billing-clerk consumer aggregates these events into monthly invoices while the ClickHouse audit consumer persists the raw grain for cost analytics and chargeback reporting. |
| **Owner**          | Platform Engineering / Billing                                   |
| **SLA**            | At-least-once delivery; idempotency key = `tenant_id + metric_type + timestamp` |

---

## 2. NATS Configuration (YAML)

```yaml
stream:
  name: BILLING-USAGE
  subjects:
    - "kubric.*.billing.usage.v1"
  retention: limits
  max_age: 34128000       # 395 days — full fiscal year + 30-day dispute buffer
  max_bytes: 10737418240  # 10 GB
  replicas: 3
  storage: file
  discard: old
  duplicate_window: 120s

consumer:
  name: billing-clerk
  durable: billing-clerk
  ack_policy: explicit
  max_deliver: 5
  ack_wait: 30s
  deliver_policy: all
```

> A second consumer `clickhouse-audit` shares the same stream with identical `ack_policy`, `max_deliver`, and `ack_wait` settings, bound to its own durable subscription.

---

## 3. Message Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "kubric/billing/usage/v1",
  "title": "BillingUsageEvent",
  "type": "object",
  "required": ["tenant_id", "metric_type", "quantity", "unit_price_usd", "timestamp"],
  "additionalProperties": false,
  "properties": {
    "tenant_id": {
      "type": "string",
      "format": "uuid",
      "description": "UUID v4 identifying the Kubric tenant."
    },
    "metric_type": {
      "type": "string",
      "enum": ["agent_seat", "event_ingested", "ml_call", "storage_gb"],
      "description": "The category of billable resource consumed."
    },
    "quantity": {
      "type": "number",
      "minimum": 0,
      "description": "Dimensionless count or fractional GB of the consumed resource."
    },
    "unit_price_usd": {
      "type": "number",
      "minimum": 0,
      "description": "Contracted unit price in USD at time of consumption."
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp of when the usage event occurred."
    },
    "agent_id": {
      "type": ["string", "null"],
      "format": "uuid",
      "description": "Optional KAI agent persona UUID that incurred the usage. Null for infrastructure-level metrics."
    }
  }
}
```

---

## 4. Example Payload (JSON)

### 4a — ML inference call from analyst agent
```json
{
  "tenant_id": "d4e7f1a2-3b8c-4d9e-a012-56789abcdef0",
  "metric_type": "ml_call",
  "quantity": 1,
  "unit_price_usd": 0.0035,
  "timestamp": "2026-02-26T14:32:07.441Z",
  "agent_id": "8f3c1d2a-7e4b-4f0c-9a1d-123456789abc"
}
```

### 4b — Storage delta from SIEM pipeline
```json
{
  "tenant_id": "d4e7f1a2-3b8c-4d9e-a012-56789abcdef0",
  "metric_type": "storage_gb",
  "quantity": 4.73,
  "unit_price_usd": 0.023,
  "timestamp": "2026-02-26T00:00:00.000Z",
  "agent_id": null
}
```

### 4c — Agent seat provisioned for the month
```json
{
  "tenant_id": "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e",
  "metric_type": "agent_seat",
  "quantity": 1,
  "unit_price_usd": 149.00,
  "timestamp": "2026-02-01T00:00:00.000Z",
  "agent_id": "2e5f7a8b-9c0d-4e1f-b2a3-fedcba987654"
}
```

### 4d — Bulk event ingestion batch
```json
{
  "tenant_id": "d4e7f1a2-3b8c-4d9e-a012-56789abcdef0",
  "metric_type": "event_ingested",
  "quantity": 15827,
  "unit_price_usd": 0.000004,
  "timestamp": "2026-02-26T14:35:00.000Z",
  "agent_id": null
}
```

---

## 5. Publisher Details

| Attribute          | Value                                                                                                              |
|--------------------|--------------------------------------------------------------------------------------------------------------------|
| **Service**        | KAI FastAPI server                                                                                                 |
| **File**           | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py`                                            |
| **Trigger**        | Every successful `POST /agents/{persona}/trigger` response — published immediately after agent task acceptance via a FastAPI `BackgroundTask` |
| **Subject format** | `kubric.{tenant_id}.billing.usage.v1`                                                                              |
| **Library**        | `nats.py` async client, JetStream `publish_async` with `Nats-Msg-Id` header set to `{tenant_id}:{metric_type}:{timestamp}` |
| **Serialisation**  | UTF-8 JSON, no compression                                                                                         |
| **Headers**        | `Kubric-Tenant: {tenant_id}`, `Kubric-Schema-Version: v1`                                                         |

Publishing is fire-and-forget from the HTTP path. A periodic reconciler (`K-KAI-BL-003_usage_reconciler.py`) re-derives missing events from the API audit log and re-publishes within 15 minutes if NATS was unavailable. The `duplicate_window: 120s` stream setting deduplicates bursts from HTTP client retries.

---

## 6. Consumer Details

### billing-clerk
| Attribute          | Value                                                                                  |
|--------------------|----------------------------------------------------------------------------------------|
| **File**           | `K-KAI-BL-001_billing_clerk.py`                                                        |
| **Group**          | Exclusive durable — single active consumer instance                                    |
| **Action**         | Accumulates usage events into a Redis sorted set keyed `billing:{tenant_id}:{metric_type}:{month}`. At month-end close, flushes aggregates to the invoicing REST service, then ACKs the full batch. Performs idempotency check via Redis `SETNX` before insertion. |
| **Batch size**     | 500 messages per pull                                                                  |
| **Error behaviour**| NACKs with 5 s delay on transient Redis failures; dead-letters after 5 redeliveries   |

### clickhouse-audit
| Attribute          | Value                                                                                  |
|--------------------|----------------------------------------------------------------------------------------|
| **File**           | `K-KAI-BL-002_clickhouse_audit.py`                                                     |
| **Group**          | Exclusive durable — single active consumer instance                                    |
| **Action**         | Bulk-inserts raw usage rows into ClickHouse `billing.usage_raw` via the async HTTP native interface. Maintains a 2-second flush window or 1000-row batch, whichever is reached first. |
| **Batch size**     | 1000 messages per pull                                                                 |
| **Error behaviour**| NACKs with exponential backoff (base 2 s, max 120 s) on ClickHouse write errors; dead-letters after 5 redeliveries |

---

## 7. Retention Policy

| Parameter            | Value                                                                                  |
|----------------------|----------------------------------------------------------------------------------------|
| **Stream retention** | `limits` — age and byte caps enforced by NATS server                                  |
| **max_age**          | 34,128,000 seconds (395 days)                                                          |
| **max_bytes**        | 10 GB — oldest messages discarded (`discard: old`) when breached                       |
| **Replicas**         | 3 — survives single-node loss without data loss                                        |
| **Storage tier**     | `file` — messages survive NATS server restarts                                         |
| **Archive policy**   | ClickHouse `billing.usage_raw` provides permanent cold storage beyond the 395-day stream window. ClickHouse TTL policy tiers rows older than 2 years to S3-backed object storage. |

---

## 8. Error Handling

| Scenario                        | Behaviour                                                                                              |
|---------------------------------|--------------------------------------------------------------------------------------------------------|
| **Publish failure (NATS down)** | FastAPI logs the error to the structured app log; `K-KAI-BL-003_usage_reconciler.py` runs every 15 minutes to detect gaps by comparing API audit log entries against ClickHouse rows and re-publishes any missing events. |
| **Schema validation failure**   | Consumer validates on receipt. Invalid messages are forwarded immediately to `kubric.<tenant_id>.dlq.billing.usage.v1` with header `Kubric-Error-Reason: SCHEMA_INVALID`. No retry attempted. |
| **Max redeliveries exceeded**   | After 5 failed delivery attempts NATS routes the message to `kubric.<tenant_id>.dlq.billing.usage.v1`. PagerDuty alert fires when DLQ depth exceeds 10 messages. |
| **Duplicate events**            | NATS `duplicate_window: 120s` drops exact-duplicate publish attempts within a 2-minute window. The billing-clerk performs an additional Redis-level idempotency check before inserting into the invoice accumulator. |
| **ClickHouse unavailable**      | `clickhouse-audit` suspends consumption and retries with exponential backoff (base 2 s, max 120 s). Messages remain safely in stream within the `max_age` window. No data loss. |
| **Invoice period close race**   | billing-clerk uses a Redis distributed lock `billing:close:{tenant_id}:{month}` with a 10-minute TTL to prevent concurrent close operations across restarts. |
| **Dead-letter subject**         | `kubric.<tenant_id>.dlq.billing.usage.v1` — monitored by `K-KAI-BL-004_dlq_reaper.py`; exposes `/admin/dlq/replay` endpoint for authorised on-call re-injection. |
