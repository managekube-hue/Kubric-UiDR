# K-MB-SUB-008 — svc.ticket.v1

## Overview

This subject carries service-desk ticket lifecycle events normalised from PSA webhook payloads. The
**PSA integration service** processes webhooks from ConnectWise Manage and Autotask PSA, maps them
to `TicketEvent` messages, and publishes them here. Consumers are the comm agent (notifications),
the billing clerk (time accrual), and the SLA tracker (breach monitoring and escalation).

**Subject pattern:** `kubric.<tenant_id>.svc.ticket.v1`

**Example resolved subject:** `kubric.acme_corp.svc.ticket.v1`

Multi-tenant wildcard: `kubric.*.svc.ticket.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: SVC-TICKET — apply with: nats stream add --config svc-ticket-stream.yaml
stream:
  name: SVC-TICKET
  subjects:
    - "kubric.*.svc.ticket.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 31536000000000000  # 365 days in nanoseconds
  max_bytes: 10737418240      # 10 GiB
  max_msg_size: 16384         # 16 KiB per message
  max_msgs: -1
  discard: old
  duplicate_window: 30000000000   # 30-second dedup (PSA webhooks can double-fire)

# Consumer: comm-agent-ticket
consumer:
  stream_name: SVC-TICKET
  name: comm-agent-ticket
  durable_name: comm-agent-ticket
  filter_subject: "kubric.*.svc.ticket.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000
  max_deliver: 5
  max_ack_pending: 500
  replay_policy: instant
  deliver_group: comm-workers
  flow_control: true
  idle_heartbeat: 10000000000

# Consumer: billing-clerk
consumer:
  stream_name: SVC-TICKET
  name: billing-clerk
  durable_name: billing-clerk
  filter_subject: "kubric.*.svc.ticket.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000       # 60 s — billing writes use PostgreSQL transactions
  max_deliver: 5
  max_ack_pending: 200
  replay_policy: instant

# Consumer: sla-tracker
consumer:
  stream_name: SVC-TICKET
  name: sla-tracker
  durable_name: sla-tracker
  filter_subject: "kubric.*.svc.ticket.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 15000000000       # 15 s — SLA tracking must be low-latency
  max_deliver: 10             # SLA writes are critical; retry aggressively
  max_ack_pending: 1000
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/svc/ticket/v1",
  "title": "TicketEvent",
  "description": "Service-desk ticket lifecycle event from ConnectWise or Autotask PSA webhook.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id", "ticket_id",
    "event_type", "title", "priority", "status", "created_at", "source_system"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":   { "type": "string", "format": "uuid",
                    "description": "uuid5(NAMESPACE_DNS, '{source_system}:{ticket_id}:{event_type}:{updated_at}') for idempotency." },
    "tenant_id":  { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "ticket_id":  { "type": "string", "maxLength": 64 },
    "event_type": {
      "type": "string",
      "enum": [
        "created", "updated", "status_changed", "priority_changed",
        "assigned", "resolved", "closed", "reopened",
        "sla_breach_imminent", "sla_breached"
      ]
    },
    "title":   { "type": "string", "maxLength": 512 },
    "description": { "type": "string", "maxLength": 8192 },
    "priority": {
      "type": "string",
      "enum": ["P1", "P2", "P3", "P4"],
      "description": "P1=Critical (15 min), P2=High (1 h), P3=Medium (4 h), P4=Low (NBD)."
    },
    "status": {
      "type": "string",
      "enum": ["new", "in_progress", "waiting_customer", "waiting_vendor", "resolved", "closed"]
    },
    "assigned_to":   { "type": "string", "maxLength": 256 },
    "assigned_team": { "type": "string", "maxLength": 256 },
    "created_at":    { "type": "string", "format": "date-time" },
    "updated_at":    { "type": "string", "format": "date-time" },
    "resolved_at":   { "type": "string", "format": "date-time" },
    "sla_breach_at": { "type": "string", "format": "date-time",
                       "description": "SLA breach deadline for current priority level." },
    "sla_breached":  { "type": "boolean" },
    "time_to_breach_minutes": { "type": "integer", "minimum": -99999,
      "description": "Minutes until SLA breach. Negative = already breached." },
    "source_system": {
      "type": "string",
      "enum": ["connectwise", "autotask", "servicenow", "freshdesk", "manual", "kubric_internal"]
    },
    "source_ticket_url": { "type": "string", "format": "uri", "maxLength": 2048 },
    "linked_asset_ids":  { "type": "array", "items": { "type": "string" } },
    "linked_alert_ids":  { "type": "array", "items": { "type": "string", "format": "uuid" } },
    "contact_email": { "type": "string", "format": "email" },
    "company_name":  { "type": "string", "maxLength": 256 },
    "category":      { "type": "string", "maxLength": 128 },
    "billable":      { "type": "boolean" },
    "time_logged_hours": { "type": "number", "minimum": 0 }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "b2c3d4e5-f6a7-8901-bcde-f01234567890",
  "tenant_id": "acme_corp",
  "ticket_id": "CW-88421",
  "event_type": "sla_breach_imminent",
  "title": "[P1] Suspected ransomware on WIN-ENDPOINT-0042 — immediate response required",
  "description": "KAI opened this ticket after Cobalt Strike beacon score 0.94 correlated with anomalous process execution. Asset isolated. Technician action required.",
  "priority": "P1",
  "status": "in_progress",
  "assigned_to": "soc-analyst@kubric-noc.com",
  "assigned_team": "SOC Tier 2",
  "created_at": "2026-02-26T14:35:00.000Z",
  "updated_at": "2026-02-26T14:49:00.000Z",
  "resolved_at": null,
  "sla_breach_at": "2026-02-26T14:50:00.000Z",
  "sla_breached": false,
  "time_to_breach_minutes": 1,
  "source_system": "connectwise",
  "source_ticket_url": "https://cw.acmecorp-msp.com/v4_6_release/services/system_io/Service/fv_sr100_request.rails?sId=88421",
  "linked_asset_ids": ["win-endpoint-0042"],
  "linked_alert_ids": ["d5e6f7a8-b9c0-1234-def0-1234567890ab"],
  "contact_email": "it-director@acmecorp.com",
  "company_name": "ACME Corp",
  "category": "Security Incident",
  "billable": false,
  "time_logged_hours": 0.25
}
```

NATS message headers:

```
Nats-Msg-Id: b2c3d4e5-f6a7-8901-bcde-f01234567890
Content-Type: application/json
X-Kubric-Schema: svc.ticket.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Ticket: CW-88421
X-Kubric-Priority: P1
X-Kubric-Event-Type: sla_breach_imminent
X-Kubric-SLA-Breach-At: 2026-02-26T14:50:00.000Z
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/psa_integration/` (Python FastAPI) |
| Source module | `services/psa_integration/publishers/ticket_publisher.py` |
| Transport | NATS JetStream `PublishAsync` with `Nats-Msg-Id` header |
| Webhook receiver | `/webhooks/connectwise` and `/webhooks/autotask`; validated with PSA HMAC or API key headers |
| Normalisation | `services/psa_integration/normalizers/connectwise.py` and `autotask.py` map provider status/priority enums to Kubric canonical schema |
| Dedup key | `uuid5(NAMESPACE_DNS, "{source_system}:{ticket_id}:{event_type}:{updated_at}")` — same update cannot duplicate within 30-second window |
| Internal tickets | Kubric-created tickets (patch workflow, housekeeper) published directly with `source_system:kubric_internal` |
| Offline buffer | PostgreSQL `ticket_event_buffer` (max 50,000 rows); replayed on reconnect |
| Credentials | mTLS + NKey; Vault path `nats/creds/psa/<tenant_id>` |

---

## Consumer Details

### comm-agent-ticket

| Field | Value |
|---|---|
| Service | `services/kai/agents/comm_agent.py` |
| Consumer type | Push-based durable, queue group `comm-workers` |
| Processing | P1/P2 tickets send immediate notifications via `kubric.<tenant_id>.comm.alert.v1`; `sla_breach_imminent` escalates to on-call engineer and customer contact; `resolved`/`closed` generates closure summary email |
| Fallback | Notification delivery failures after max_deliver: dead-letter to `kubric.<tenant_id>.dlq.comm.ticket`; fallback SMS via Twilio |

### billing-clerk

| Field | Value |
|---|---|
| Service | `services/billing/billing_clerk.py` (Python) |
| Consumer type | Push-based durable |
| Processing | Reconciles `time_logged_hours` changes against tenant service contract; `billable:true` updates monthly accrual ledger; final billing entry on `closed` events |
| Failure handling | PostgreSQL deadlock or constraint violation NAKs with 5 s backoff; after max_deliver=5 billing team alerted for manual reconciliation |

### sla-tracker

| Field | Value |
|---|---|
| Service | `services/svc/sla_tracker.py` (Python) |
| Consumer type | Push-based durable |
| Processing | On `created`/`priority_changed`: upserts SLA deadline into Redis sorted set keyed by `sla_breach_at` epoch; scheduler checks every 30 s for imminent breaches and re-publishes `sla_breach_imminent` events; on `resolved`/`closed`: removes from schedule and records SLA adherence |
| Max delivers | 10 — SLA tracking failures must be retried aggressively |
| Redis failure | After max_deliver=10 a critical alert fires (missed SLA tracking directly impacts contractual obligations) |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 365 days | Full year for MSP contract, SLA audit, and billing lookback |
| `max_bytes` | 10 GiB | Moderate volume; multi-year headroom at typical MSP scale |
| `discard` | `old` | Oldest records evicted when byte ceiling reached |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent; billing and SLA evidence must survive restarts |
| Duplicate window | 30 seconds | PSA webhook double-fire suppression |

---

## Error Handling

**Publisher side**

- Webhook signature validation failure: HTTP 401; event not published; `psa_integration_webhook_auth_failures_total` incremented.
- Normalisation failure (unknown PSA status code): maps to `status:in_progress` as safe default; `"normalisation_warning": "<reason>"` added.
- NATS unavailability: PostgreSQL `ticket_event_buffer` (max 50,000 rows) with ordered replay on reconnect.

**Consumer side**

- `comm-agent-ticket`: delivery failure after all retries dead-letters to `kubric.<tenant_id>.dlq.comm.ticket` and triggers Twilio SMS fallback to on-call engineer.
- `billing-clerk`: PostgreSQL failure NAKs with 5 s backoff; after max_deliver=5 billing team alerted for manual reconciliation.
- `sla-tracker`: Redis unavailability causes NAK; after 10 deliveries critical PagerDuty alert fires.
- SLA circular events: `sla_breach_imminent` re-publications by sla-tracker are idempotent due to the 30-second dedup window; comm and billing consumers handle duplicates gracefully.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
