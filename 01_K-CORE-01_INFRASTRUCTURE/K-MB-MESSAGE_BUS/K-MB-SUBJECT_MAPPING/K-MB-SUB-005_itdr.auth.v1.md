# K-MB-SUB-005 — itdr.auth.v1

## Overview

This subject carries identity and authentication events normalised from identity-provider webhooks.
The **identity service** listens to Okta Event Hooks and Azure AD audit-log webhooks, normalises
raw provider payloads into `AuthEvent` messages, and publishes them here. Consumers are the analyst
agent (UEBA correlation), the SIEM forwarder, and the triage agent (cross-domain login/EDR
correlation).

**Subject pattern:** `kubric.<tenant_id>.itdr.auth.v1`

**Example resolved subject:** `kubric.acme_corp.itdr.auth.v1`

Multi-tenant wildcard: `kubric.*.itdr.auth.v1`

---

## NATS Configuration (YAML)

```yaml
# Stream: ITDR-AUTH — apply with: nats stream add --config itdr-auth-stream.yaml
stream:
  name: ITDR-AUTH
  subjects:
    - "kubric.*.itdr.auth.v1"
  retention: limits
  storage: file
  num_replicas: 3
  max_age: 7776000000000000   # 90 days in nanoseconds
  max_bytes: 21474836480      # 20 GiB
  max_msg_size: 16384         # 16 KiB per message
  max_msgs: -1
  discard: old
  duplicate_window: 120000000000
  # 90-day retention satisfies SOC 2 Type II and ISO 27001 audit lookback

# Consumer: analyst-agent-auth
consumer:
  stream_name: ITDR-AUTH
  name: analyst-agent-auth
  durable_name: analyst-agent-auth
  filter_subject: "kubric.*.itdr.auth.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 45000000000       # 45 seconds
  max_deliver: 5
  max_ack_pending: 2000
  replay_policy: instant
  deliver_group: analyst-workers
  flow_control: true
  idle_heartbeat: 5000000000

# Consumer: siem-forwarder-itdr-auth
consumer:
  stream_name: ITDR-AUTH
  name: siem-forwarder-itdr-auth
  durable_name: siem-forwarder-itdr-auth
  filter_subject: "kubric.*.itdr.auth.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 60000000000
  max_deliver: 3
  max_ack_pending: 5000
  replay_policy: instant

# Consumer: triage-agent-auth
consumer:
  stream_name: ITDR-AUTH
  name: triage-agent-auth
  durable_name: triage-agent-auth
  filter_subject: "kubric.*.itdr.auth.v1"
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30000000000
  max_deliver: 5
  max_ack_pending: 1000
  replay_policy: instant
```

---

## Message Schema (JSON)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12",
  "$id": "https://schemas.kubric.io/itdr/auth/v1",
  "title": "AuthEvent",
  "description": "Normalised identity authentication event from Okta, Azure AD, or other IdP webhook.",
  "type": "object",
  "required": [
    "schema_version", "event_id", "tenant_id",
    "user_id", "email", "source_ip", "outcome", "provider", "timestamp"
  ],
  "additionalProperties": false,
  "properties": {
    "schema_version": { "type": "string", "const": "1.0" },
    "event_id":  { "type": "string", "format": "uuid" },
    "tenant_id": { "type": "string", "pattern": "^[a-z0-9_]{3,64}$" },
    "user_id":   { "type": "string", "maxLength": 256,
                   "description": "Provider-internal user identifier (Okta UID, AAD Object ID, etc.)." },
    "email":     { "type": "string", "format": "email" },
    "source_ip": { "type": "string", "format": "ipv4" },
    "device_id": { "type": "string", "maxLength": 256 },
    "outcome": {
      "type": "string",
      "enum": [
        "success", "failure", "mfa_challenge", "mfa_success", "mfa_failure",
        "lockout", "password_reset", "session_hijack_suspected"
      ]
    },
    "provider": {
      "type": "string",
      "enum": ["okta", "azure_ad", "google_workspace", "ping_identity", "generic_oidc"]
    },
    "timestamp":   { "type": "string", "format": "date-time" },
    "geo_country": { "type": "string", "maxLength": 2,
                     "description": "ISO 3166-1 alpha-2 country of source IP." },
    "geo_city":    { "type": "string", "maxLength": 128 },
    "user_agent":  { "type": "string", "maxLength": 1024 },
    "application": { "type": "string", "maxLength": 256 },
    "failure_reason": { "type": "string", "maxLength": 512 },
    "risk_level": {
      "type": "string",
      "enum": ["none", "low", "medium", "high"],
      "description": "Risk level from the IdP own risk engine."
    },
    "is_suspicious": { "type": "boolean",
                       "description": "True when Kubric ITDR rules flagged this event." },
    "itdr_rule_ids": { "type": "array", "items": { "type": "string" } }
  }
}
```

---

## Example Payload (JSON)

```json
{
  "schema_version": "1.0",
  "event_id": "e7f8a9b0-c1d2-3456-ef01-234567890abc",
  "tenant_id": "acme_corp",
  "user_id": "00u1a2b3c4d5e6f7g8h9",
  "email": "jsmith@acmecorp.com",
  "source_ip": "91.108.4.100",
  "device_id": null,
  "outcome": "success",
  "provider": "okta",
  "timestamp": "2026-02-26T03:17:42.000Z",
  "geo_country": "RU",
  "geo_city": "Moscow",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
  "application": "Kubric MSP Portal",
  "failure_reason": null,
  "risk_level": "high",
  "is_suspicious": true,
  "itdr_rule_ids": ["ITDR-001_impossible_travel", "ITDR-007_new_country_first_seen"]
}
```

NATS message headers:

```
Nats-Msg-Id: e7f8a9b0-c1d2-3456-ef01-234567890abc
Content-Type: application/json
X-Kubric-Schema: itdr.auth.v1
X-Kubric-Tenant: acme_corp
X-Kubric-Provider: okta
X-Kubric-Outcome: success
X-Kubric-Suspicious: true
```

---

## Publisher Details

| Field | Value |
|---|---|
| Service | `services/identity/` (Python FastAPI) |
| Source module | `services/identity/publishers/auth_publisher.py` |
| Transport | NATS JetStream `PublishAsync` with `Nats-Msg-Id` header |
| Webhook receiver | `/webhooks/okta` and `/webhooks/azure_ad`; HMAC-SHA256 signature validation |
| Normalisation | Per-provider mappers in `services/identity/normalizers/<provider>.py` |
| ITDR rules | Engine runs inline before publish; populates `is_suspicious` and `itdr_rule_ids` |
| Rate | Typically 10-500 events/sec; bursts to 5,000/sec during credential-stuffing attacks |
| Offline buffer | PostgreSQL `auth_event_buffer` table (max 100,000 rows); replayed on reconnect |
| Credentials | mTLS + NKey; Vault path `nats/creds/identity/<tenant_id>` |

---

## Consumer Details

### analyst-agent-auth

| Field | Value |
|---|---|
| Service | `services/kai/agents/analyst_agent.py` |
| Consumer type | Push-based durable, queue group `analyst-workers` |
| Processing | `is_suspicious:true` events enriched with UEBA from ClickHouse; correlated with concurrent EDR events; confirmed threats to `kubric.<tenant_id>.security.alert.v1` |
| UEBA timeout | Query timeout > 10 s causes processing without UEBA context; confidence downgraded to `medium` |

### siem-forwarder-itdr-auth

| Field | Value |
|---|---|
| Service | `services/siem_forwarder/` |
| Processing | All auth events forwarded regardless of suspicion flag |
| Dead-letter | `kubric.<tenant_id>.dlq.siem.itdr.auth` |

### triage-agent-auth

| Field | Value |
|---|---|
| Service | `services/kai/agents/triage_agent.py` |
| Processing | Correlates suspicious auth with active endpoint alerts to identify post-compromise lateral movement |

---

## Retention Policy

| Parameter | Value | Rationale |
|---|---|---|
| Retention type | `limits` | Age + byte ceiling |
| `max_age` | 90 days | SOC 2 Type II and ISO 27001 audit evidence lookback |
| `max_bytes` | 20 GiB | Auth events are low-volume but must persist 90 days |
| `discard` | `old` | Oldest records evicted when byte ceiling hit |
| Replicas | 3 | Single-node failure tolerance |
| Storage | File | Persistent |
| Duplicate window | 2 minutes | IdP webhook retry deduplication |

---

## Error Handling

**Publisher side**

- Webhook HMAC validation failure: HTTP 401; event not published; counted in `identity_webhook_auth_failures_total`.
- Normalisation failure (unknown provider schema): best-effort mapping; `"schema_version": "1.0-partial"` set; event still published.
- NATS unavailability: PostgreSQL `auth_event_buffer` (max 100,000 rows); relay-on-reconnect in order.

**Consumer side**

- `analyst-agent-auth`: UEBA query timeout > 10 s processes without UEBA; alert confidence set to `medium`.
- `triage-agent-auth`: no correlated EDR event within 15 minutes logs to identity audit trail without escalation.
- `siem-forwarder`: dead-letter `kubric.<tenant_id>.dlq.siem.itdr.auth`.

**Schema validation**

Invalid messages routed to `kubric.<tenant_id>.dlq.schema.invalid` with `X-Kubric-Validation-Error` header.
