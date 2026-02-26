# K-MB-SUB-010 — Health Score

> NATS subject mapping reference for the Kubric Security Score (KiSS) health telemetry within the Kubric UIDR platform.

## Subject Pattern

```
kubric.health.score.v1
kubric.health.score.>           # wildcard for all health score sub-subjects
```

Tokens:
| Position | Token       | Description                              |
|----------|-------------|------------------------------------------|
| 1        | `kubric`    | Platform root namespace                  |
| 2        | `health`    | Domain — platform and tenant health      |
| 3        | `score`     | Event type — computed health score       |
| 4        | `v1`        | Schema version                           |

## Publisher

**KAI-SENTINEL** — the dedicated health monitoring and scoring agent within the KAI crew. KAI-SENTINEL continuously computes the Kubric Security Score (KiSS) by aggregating signals from all detection and compliance modules.

Publish cadence:
| Trigger              | Frequency            |
|----------------------|----------------------|
| Periodic refresh     | Every 60 seconds     |
| Score-change event   | On any component score delta >= 5 points |
| Manual recalculation | On-demand via API    |

## Consumer(s)

| Consumer                     | Runtime               | Role                                              |
|------------------------------|------------------------|----------------------------------------------------|
| **K-SVC Portal** (primary)   | TypeScript / Next.js   | Renders real-time health dashboard via EventSource SSE stream |
| KAI-COMM                     | JavaScript / Vapi      | Fires voice/chat alerts when score drops below threshold |
| KAI-ANALYST                  | Python                 | Trend analysis and churn prediction model input    |

## Payload

**Format:** JSON — Kubric Security Score (KiSS) envelope.

**Schema version:** `1.0.0`

**Content-Type:** `application/json`

```jsonc
{
  "schema_version": "1.0.0",
  "tenant_id": "tnt_a1b2c3d4",
  "overall_score": 82,
  "component_scores": {
    "edr": 90,
    "ndr": 78,
    "vdr": 75,
    "compliance": 88,
    "patching": 79
  },
  "risk_level": "medium",
  "trend": "improving",
  "churn_probability": 0.12,
  "timestamp": "2026-02-26T14:30:00Z",
  "scoring_model_version": "2.1.0"
}
```

## Fields

| Field                          | Type     | Required | Description                                                         |
|--------------------------------|----------|----------|---------------------------------------------------------------------|
| `schema_version`               | string   | yes      | Payload schema version (semver)                                     |
| `tenant_id`                    | string   | yes      | Unique tenant identifier (`tnt_` prefix)                            |
| `overall_score`                | integer  | yes      | Composite KiSS score, range 0-100                                   |
| `component_scores.edr`         | integer  | yes      | Endpoint Detection and Response sub-score (0-100)                   |
| `component_scores.ndr`         | integer  | yes      | Network Detection and Response sub-score (0-100)                    |
| `component_scores.vdr`         | integer  | yes      | Vulnerability Detection and Response sub-score (0-100)              |
| `component_scores.compliance`  | integer  | yes      | Compliance posture sub-score (0-100)                                |
| `component_scores.patching`    | integer  | yes      | Patch management sub-score (0-100)                                  |
| `risk_level`                   | string   | yes      | Derived risk level: `critical`, `high`, `medium`, `low`, `minimal`  |
| `trend`                        | string   | yes      | Score trajectory: `improving`, `stable`, `declining`                |
| `churn_probability`            | float    | yes      | ML-predicted churn probability (0.0-1.0) for the tenant             |
| `timestamp`                    | datetime | yes      | ISO 8601 UTC timestamp of score computation                         |
| `scoring_model_version`        | string   | no       | Version of the KiSS scoring model used                              |

### Risk Level Mapping

| Overall Score | Risk Level  |
|---------------|-------------|
| 90-100        | `minimal`   |
| 75-89         | `low`       |
| 50-74         | `medium`    |
| 25-49         | `high`      |
| 0-24          | `critical`  |

### EventSource Streaming to Frontend

The K-SVC Portal backend subscribes to `kubric.health.score.>` via a NATS subscription and fans out to authenticated browser clients over Server-Sent Events (SSE):

```
NATS (kubric.health.score.v1)
  --> K-SVC API (Next.js API route /api/health/stream)
    --> SSE EventSource per tenant session
      --> React dashboard component re-render
```

Each SSE event is filtered by `tenant_id` so tenants only receive their own score updates.

## JetStream Configuration

```
Stream:          HEALTH_SCORE
Subjects:        kubric.health.score.>
Storage:         File
Retention:       Limits
Max Age:         30 days
Max Bytes:       2 GB
Replicas:        3
Discard Policy:  Old
Max Msgs Per Subject: 1000
Duplicate Window: 5 seconds
```

## Consumer Groups

| Consumer Group      | Deliver Policy   | Ack Policy  | Max Deliver | Filter Subject              |
|---------------------|-----------------|-------------|-------------|-----------------------------|
| `portal-sse`        | Last Per Subject | Explicit    | 3           | `kubric.health.score.>`     |
| `comm-threshold`    | All             | Explicit    | 5           | `kubric.health.score.>`     |
| `analyst-trend`     | All             | Explicit    | 3           | `kubric.health.score.>`     |

## Example (NATS CLI)

**Publish a health score update:**

```bash
nats pub kubric.health.score.v1 '{
  "schema_version": "1.0.0",
  "tenant_id": "tnt_a1b2c3d4",
  "overall_score": 82,
  "component_scores": {
    "edr": 90,
    "ndr": 78,
    "vdr": 75,
    "compliance": 88,
    "patching": 79
  },
  "risk_level": "medium",
  "trend": "improving",
  "churn_probability": 0.12,
  "timestamp": "2026-02-26T14:30:00Z",
  "scoring_model_version": "2.1.0"
}'
```

**Subscribe to all health score events:**

```bash
nats sub "kubric.health.score.>"
```

**Create the JetStream stream:**

```bash
nats stream add HEALTH_SCORE \
  --subjects "kubric.health.score.>" \
  --storage file \
  --retention limits \
  --max-age 30d \
  --max-bytes 2GB \
  --replicas 3 \
  --discard old \
  --dupe-window 5s
```

**Create the portal-sse consumer:**

```bash
nats consumer add HEALTH_SCORE portal-sse \
  --deliver last-per-subject \
  --ack explicit \
  --max-deliver 3 \
  --filter "kubric.health.score.>"
```

## Notes

- **Scoring Weights:** The `overall_score` is a weighted average of component scores. Default weights: EDR 25%, NDR 20%, VDR 20%, Compliance 20%, Patching 15%. Weights are configurable per tenant contract tier.
- **Churn Prediction:** The `churn_probability` field is computed by KAI-ANALYST's ML model and attached by KAI-SENTINEL before publishing. A probability above 0.35 triggers a proactive customer success outreach via KAI-COMM.
- **Threshold Alerts:** When `overall_score` drops below 50, KAI-COMM automatically places a Vapi voice call to the tenant's designated security contact.
- **Historical Trend:** The `trend` field is computed by comparing the current score against the 7-day rolling average. A delta of more than 10 points triggers the `declining` or `improving` label.
- **Last-Value Semantics:** The `portal-sse` consumer uses `Last Per Subject` delivery so newly connected dashboard clients immediately receive the most recent score without waiting for the next publish cycle.
- **Related Subjects:** `kubric.health.component.>` (detailed per-component breakdown), `kubric.health.incident.>` (health-impacting incident correlation).
