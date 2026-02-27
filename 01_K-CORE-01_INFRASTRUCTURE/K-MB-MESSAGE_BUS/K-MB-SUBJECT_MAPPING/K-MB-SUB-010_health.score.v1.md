# K-MB-SUB-010 — health.score.v1

---

## 1. Overview

| Field              | Value                                                           |
|--------------------|------------------------------------------------------------------|
| **Subject**        | `kubric.<tenant_id>.health.score.v1` (canonical) and `kai.health.<tenant_id>` (legacy alias — both active) |
| **Stream**         | `HEALTH-SCORE`                                                  |
| **Version**        | v1                                                              |
| **Purpose**        | Carries periodic composite security health scores computed for each tenant. The score (0–100) aggregates five leading and lagging indicators: alert volume over 24 hours, mean-time-to-remediate over 7 days, patch lag in days, open configuration drift count, and open critical vulnerability count. It feeds the customer-facing dashboard, the churn-risk ML model, and the SLA compliance tracker to provide a real-time view of posture degradation. |
| **Owner**          | KAI Sentinel / Platform Intelligence                            |
| **SLA**            | Published every 5 minutes per active tenant; at-least-once delivery; consumers tolerate out-of-order arrivals by sorting on `computed_at` |

---

## 2. NATS Configuration (YAML)

```yaml
stream:
  name: HEALTH-SCORE
  subjects:
    - "kubric.*.health.score.v1"
    - "kai.health.*"
  retention: limits
  max_age: 7776000       # 90 days
  max_bytes: 5368709120  # 5 GB
  replicas: 3
  storage: file
  discard: old
  duplicate_window: 120s

consumer:
  name: dashboard-api
  durable: dashboard-api
  ack_policy: explicit
  max_deliver: 5
  ack_wait: 30s
  deliver_policy: all
```

> Additional durable consumers `churn-risk-model` and `sla-tracker` are registered against the same stream with identical ack settings.

---

## 3. Message Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "kubric/health/score/v1",
  "title": "HealthScoreEvent",
  "type": "object",
  "required": ["tenant_id", "score", "components", "computed_at"],
  "additionalProperties": false,
  "properties": {
    "tenant_id": {
      "type": "string",
      "format": "uuid",
      "description": "UUID v4 of the tenant whose posture is being scored."
    },
    "score": {
      "type": "integer",
      "minimum": 0,
      "maximum": 100,
      "description": "Composite health score. Higher is healthier. Derived via weighted geometric mean of normalised component scores."
    },
    "components": {
      "type": "object",
      "required": [
        "alert_volume_24h",
        "mttr_hours_7d",
        "patch_lag_days",
        "drift_count_open",
        "open_critical_vulns"
      ],
      "additionalProperties": false,
      "properties": {
        "alert_volume_24h": {
          "type": "integer",
          "minimum": 0,
          "description": "Total security alerts generated across all detection engines in the past 24 hours."
        },
        "mttr_hours_7d": {
          "type": "number",
          "minimum": 0,
          "description": "Mean time to remediate (hours) over the trailing 7 days across all closed remediation tasks."
        },
        "patch_lag_days": {
          "type": "number",
          "minimum": 0,
          "description": "Weighted average days since critical/high CVEs became available in NVD, weighted by CVSS score."
        },
        "drift_count_open": {
          "type": "integer",
          "minimum": 0,
          "description": "Number of unresolved configuration drift findings in the GRC module."
        },
        "open_critical_vulns": {
          "type": "integer",
          "minimum": 0,
          "description": "Count of open CVSS 9.0+ vulnerabilities with no accepted risk or active remediation task."
        }
      }
    },
    "computed_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 UTC timestamp of when this score was computed."
    }
  }
}
```

---

## 4. Example Payload (JSON)

### 4a — Healthy tenant, all indicators nominal
```json
{
  "tenant_id": "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
  "score": 87,
  "components": {
    "alert_volume_24h": 14,
    "mttr_hours_7d": 3.2,
    "patch_lag_days": 4.1,
    "drift_count_open": 2,
    "open_critical_vulns": 0
  },
  "computed_at": "2026-02-26T14:00:07.123Z"
}
```

### 4b — At-risk tenant triggering churn model
```json
{
  "tenant_id": "f9e8d7c6-b5a4-4938-8271-fedcba012345",
  "score": 31,
  "components": {
    "alert_volume_24h": 312,
    "mttr_hours_7d": 47.6,
    "patch_lag_days": 38.0,
    "drift_count_open": 89,
    "open_critical_vulns": 14
  },
  "computed_at": "2026-02-26T14:00:09.884Z"
}
```

### 4c — Mid-range tenant, approaching SLA boundary
```json
{
  "tenant_id": "11223344-5566-4778-8990-aabbccddeeff",
  "score": 62,
  "components": {
    "alert_volume_24h": 57,
    "mttr_hours_7d": 8.9,
    "patch_lag_days": 12.5,
    "drift_count_open": 11,
    "open_critical_vulns": 3
  },
  "computed_at": "2026-02-26T14:00:11.207Z"
}
```

---

## 5. Publisher Details

| Attribute          | Value                                                                                                                    |
|--------------------|--------------------------------------------------------------------------------------------------------------------------|
| **Service**        | KAI Sentinel health score publisher                                                                                       |
| **File**           | `K-KAI-SEN-001_health_score_publisher.py`                                                                                |
| **Trigger**        | APScheduler cron job firing every 5 minutes; iterates all active tenants using `asyncio.gather` with concurrency limit of 50 |
| **Subject format** | `kubric.{tenant_id}.health.score.v1` (primary) and `kai.health.{tenant_id}` (legacy alias) — both published per cycle   |
| **Library**        | `nats.py` async JetStream `publish_async`; `Nats-Msg-Id` = `{tenant_id}:{computed_at_epoch_ms}`                         |
| **Serialisation**  | UTF-8 JSON                                                                                                               |
| **Headers**        | `Kubric-Tenant: {tenant_id}`, `Kubric-Schema-Version: v1`                                                               |

Source data is pulled from ClickHouse `security.alert_events` (alert volume, MTTR) and the Kubric CMDB REST API (patch lag, drift, open critical vulns). Compute time per tenant is approximately 50 ms.

---

## 6. Consumer Details

### dashboard-api
| Attribute          | Value                                                                                                          |
|--------------------|----------------------------------------------------------------------------------------------------------------|
| **Role**           | Writes each score to Redis `health:{tenant_id}:latest` (JSON string, no TTL). The WebSocket layer reads from Redis, decoupling read latency from NATS fan-out. |
| **Batch size**     | 100 messages per pull; flushed immediately on any score change (delta > 0 from last stored value)              |
| **Error behaviour**| NACKs on Redis write failure; dead-letters after 5 redeliveries                                                |

### churn-risk-model
| Attribute          | Value                                                                                                          |
|--------------------|----------------------------------------------------------------------------------------------------------------|
| **File**           | `K-KAI-SEN-002_churn_risk_model.py`                                                                            |
| **Role**           | Features the `components` object into a scikit-learn gradient-boosted churn predictor. Stores predictions in Postgres `churn_risk` table. Fires Slack `#cs-alerts` when P(churn) > 0.75. |
| **Batch size**     | 50 messages per pull                                                                                           |
| **Error behaviour**| NACKs with 10 s delay on model inference failure; dead-letters after 5 redeliveries                            |

### sla-tracker
| Attribute          | Value                                                                                                          |
|--------------------|----------------------------------------------------------------------------------------------------------------|
| **Role**           | Evaluates `score` against the tenant contracted SLA tier: bronze >= 50, silver >= 65, gold >= 80. Emits `kubric.<tenant_id>.sla.breach.v1` when score falls below tier threshold for two consecutive 5-minute windows. |
| **Batch size**     | 200 messages per pull                                                                                          |
| **Error behaviour**| NACKs with 5 s delay on Postgres write errors; dead-letters after 5 redeliveries                               |

---

## 7. Retention Policy

| Parameter            | Value                                                                                                  |
|----------------------|--------------------------------------------------------------------------------------------------------|
| **Stream retention** | `limits`                                                                                               |
| **max_age**          | 7,776,000 seconds (90 days)                                                                            |
| **max_bytes**        | 5 GB                                                                                                   |
| **Replicas**         | 3                                                                                                      |
| **Storage tier**     | `file`                                                                                                 |
| **Archive policy**   | A background ClickHouse sink consumer (`health-ch-archive`, separate durable) streams all events to `health.score_history`. Rows older than 90 days are tiered to S3 via ClickHouse TTL, retained indefinitely for trend analytics and SLA dispute resolution. |

---

## 8. Error Handling

| Scenario                                  | Behaviour                                                                                                                                   |
|-------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| **ClickHouse unreachable during compute** | Publisher emits a synthetic stale score using last known values from Redis with an internal `_data_stale: true` extension field. Downstream consumers treat stale payloads as a no-update signal rather than a posture alarm. |
| **CMDB API timeout**                      | Publisher uses last-known patch_lag and drift values from a 10-minute Redis cache. A `_partial_staleness` array field lists which components used cached data. |
| **Schema validation failure**             | Invalid messages forwarded immediately to `kubric.<tenant_id>.dlq.health.score.v1` with header `Kubric-Error-Reason: SCHEMA_INVALID`. Never retried. |
| **Max redeliveries exceeded**             | NATS routes to `kubric.<tenant_id>.dlq.health.score.v1`. PagerDuty alert triggers when DLQ depth > 5 per tenant within 10 minutes. |
| **Score regression (>20 point drop)**    | `dashboard-api` consumer emits `kubric.<tenant_id>.ops.score_regression.v1`, triggering an automated email to the tenant security contact via the comm-agent pipeline. |
| **Publisher missed full cycle**           | Watchdog in `K-KAI-SEN-001` increments Prometheus counter `health_score_missed_cycles_total{tenant}`. Alertmanager pages after 3 consecutive missed cycles. |
| **Dead-letter subject**                   | `kubric.<tenant_id>.dlq.health.score.v1`                                                                                                    |
