# K-MB-SUB-012 — Communication Alert

> NATS subject mapping reference for alert notification dispatch within the Kubric UIDR platform.

## Subject Pattern

```
kubric.comm.alert.v1
kubric.comm.alert.>             # wildcard for all communication alert sub-subjects
```

Tokens:
| Position | Token       | Description                                    |
|----------|-------------|------------------------------------------------|
| 1        | `kubric`    | Platform root namespace                        |
| 2        | `comm`      | Domain — communications and notifications      |
| 3        | `alert`     | Event type — alert notification trigger        |
| 4        | `v1`        | Schema version                                 |

## Publisher

**KAI-TRIAGE** — the AI-driven alert triage agent within the KAI crew. After triaging and enriching a security alert, KAI-TRIAGE determines the notification priority and publishes a communication alert to trigger downstream delivery channels.

Publish triggers:
| Trigger                  | Description                                              |
|--------------------------|----------------------------------------------------------|
| New critical/high alert  | Immediately after triage classification                  |
| Escalation timeout       | When acknowledgment is not received within SLA window    |
| Status change            | Alert status transitions (e.g., open to investigating)   |
| Manual escalation        | Analyst-initiated escalation via K-SVC Portal            |

## Consumer(s)

| Consumer                    | Runtime             | Role                                                      |
|-----------------------------|---------------------|------------------------------------------------------------|
| **KAI-COMM** (primary)      | JavaScript / Vapi   | Orchestrates multi-channel notification delivery           |
| K-SVC Portal                | TypeScript / Next.js | In-app notification bell and alert feed                    |

## Payload

**Format:** JSON — Alert notification payload with escalation metadata.

**Schema version:** `1.0.0`

**Content-Type:** `application/json`

```jsonc
{
  "schema_version": "1.0.0",
  "tenant_id": "tnt_a1b2c3d4",
  "alert_id": "alt_e5f6g7h8",
  "severity": "critical",
  "title": "Ransomware Behavior Detected on WORKSTATION-042",
  "description": "CoreSec FIM detected rapid encryption of files in C:\\Users\\finance across 247 files in 30 seconds. Pattern matches LockBit 3.0 TTP (T1486).",
  "affected_assets": [
    {
      "asset_id": "ast_j9k0l1m2",
      "hostname": "WORKSTATION-042",
      "ip": "10.0.5.42"
    }
  ],
  "recommended_action": "Immediately isolate WORKSTATION-042 from the network and initiate forensic imaging. Do not power off the machine.",
  "escalation_level": 1,
  "escalation_matrix": {
    "level_1": { "channel": "slack", "target": "#soc-alerts", "sla_minutes": 5 },
    "level_2": { "channel": "voice", "target": "+1-555-0142", "sla_minutes": 15 },
    "level_3": { "channel": "voice", "target": "+1-555-0199", "sla_minutes": 30 }
  },
  "timestamp": "2026-02-26T14:30:00Z"
}
```

## Fields

| Field                    | Type       | Required | Description                                                            |
|--------------------------|------------|----------|------------------------------------------------------------------------|
| `schema_version`         | string     | yes      | Payload schema version (semver)                                        |
| `tenant_id`              | string     | yes      | Unique tenant identifier (`tnt_` prefix)                               |
| `alert_id`               | string     | yes      | Unique alert identifier (`alt_` prefix) from KAI-TRIAGE                |
| `severity`               | string     | yes      | Alert severity: `critical`, `high`, `medium`, `low`, `informational`   |
| `title`                  | string     | yes      | Human-readable alert title (max 256 characters)                        |
| `description`            | string     | yes      | Detailed description with context, evidence, and MITRE technique refs  |
| `affected_assets`        | object[]   | yes      | List of impacted assets with `asset_id`, `hostname`, and `ip`          |
| `recommended_action`     | string     | yes      | AI-generated recommended response action                               |
| `escalation_level`       | integer    | yes      | Current escalation tier (1 = initial, increments on SLA breach)        |
| `escalation_matrix`      | object     | yes      | Per-level escalation targets with channel, contact, and SLA            |
| `timestamp`              | datetime   | yes      | ISO 8601 UTC timestamp of notification trigger                         |

### Communication Channels

KAI-COMM dispatches notifications across multiple channels based on the escalation matrix and tenant preferences:

| Channel        | Provider     | Use Case                                      |
|----------------|-------------|------------------------------------------------|
| Vapi Voice     | Vapi AI     | Critical severity — direct voice call to SOC lead with AI briefing |
| Twilio SMS     | Twilio      | High severity — SMS to on-call analyst with alert summary         |
| Email          | SendGrid    | All severities — detailed email with full context and links       |
| Slack Webhook  | Slack API   | All severities — alert card posted to designated SOC channel      |

### Escalation Matrix

The escalation matrix defines progressive notification tiers. If acknowledgment (`ack`) is not received within the SLA window for the current level, KAI-COMM re-publishes the alert with an incremented `escalation_level`:

```
Level 1 (0 min)   --> Slack #soc-alerts              (SLA: 5 min)
Level 2 (5 min)   --> Vapi voice call to on-call      (SLA: 15 min)
Level 3 (20 min)  --> Vapi voice call to SOC manager   (SLA: 30 min)
Level 4 (50 min)  --> Vapi voice call to CISO + email  (terminal)
```

## JetStream Configuration

```
Stream:          COMM_ALERT
Subjects:        kubric.comm.alert.>
Storage:         File
Retention:       Interest
Max Age:         7 days
Max Bytes:       2 GB
Replicas:        3
Discard Policy:  Old
Duplicate Window: 10 seconds
```

## Consumer Groups

| Consumer Group        | Deliver Policy | Ack Policy  | Max Deliver | Filter Subject            |
|-----------------------|---------------|-------------|-------------|---------------------------|
| `comm-dispatcher`     | All           | Explicit    | 10          | `kubric.comm.alert.>`     |
| `portal-notification` | All           | Explicit    | 5           | `kubric.comm.alert.>`     |

## Example (NATS CLI)

**Publish a communication alert:**

```bash
nats pub kubric.comm.alert.v1 '{
  "schema_version": "1.0.0",
  "tenant_id": "tnt_a1b2c3d4",
  "alert_id": "alt_e5f6g7h8",
  "severity": "critical",
  "title": "Ransomware Behavior Detected on WORKSTATION-042",
  "description": "CoreSec FIM detected rapid encryption of files across 247 files in 30 seconds. Pattern matches LockBit 3.0 TTP (T1486).",
  "affected_assets": [
    {"asset_id": "ast_j9k0l1m2", "hostname": "WORKSTATION-042", "ip": "10.0.5.42"}
  ],
  "recommended_action": "Immediately isolate WORKSTATION-042 from the network.",
  "escalation_level": 1,
  "escalation_matrix": {
    "level_1": {"channel": "slack", "target": "#soc-alerts", "sla_minutes": 5},
    "level_2": {"channel": "voice", "target": "+1-555-0142", "sla_minutes": 15},
    "level_3": {"channel": "voice", "target": "+1-555-0199", "sla_minutes": 30}
  },
  "timestamp": "2026-02-26T14:30:00Z"
}'
```

**Subscribe to all communication alerts:**

```bash
nats sub "kubric.comm.alert.>"
```

**Create the JetStream stream:**

```bash
nats stream add COMM_ALERT \
  --subjects "kubric.comm.alert.>" \
  --storage file \
  --retention interest \
  --max-age 7d \
  --max-bytes 2GB \
  --replicas 3 \
  --discard old \
  --dupe-window 10s
```

**Create the comm-dispatcher consumer:**

```bash
nats consumer add COMM_ALERT comm-dispatcher \
  --deliver all \
  --ack explicit \
  --max-deliver 10 \
  --filter "kubric.comm.alert.>"
```

## Notes

- **Delivery Guarantee:** The `comm-dispatcher` consumer has `max-deliver: 10` to ensure notification delivery even under transient provider outages (Vapi, Twilio, SendGrid, Slack).
- **Acknowledgment Flow:** KAI-COMM publishes an acknowledgment event to `kubric.comm.ack.v1` when a human acknowledges the alert. This stops the escalation timer.
- **Quiet Hours:** Tenant-configurable quiet hours suppress non-critical notifications. Critical and high severity alerts bypass quiet hours.
- **Rate Limiting:** KAI-COMM enforces per-tenant rate limits (default: 20 voice calls/hour, 100 SMS/hour) to prevent alert fatigue and cost overruns.
- **Vapi AI Briefing:** For voice calls, KAI-COMM feeds the alert payload to Vapi's AI voice agent, which verbally briefs the recipient with a natural-language summary and asks for verbal acknowledgment.
- **Audit Log:** All notification attempts, deliveries, and acknowledgments are logged to `kubric.comm.audit.v1` for SLA compliance reporting.
- **Related Subjects:** `kubric.comm.ack.v1` (alert acknowledgment), `kubric.comm.audit.v1` (delivery audit trail), `kubric.security.alert.v1` (upstream alert source).
