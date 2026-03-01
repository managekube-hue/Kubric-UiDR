# K-MB-SUB-016 — `kubric.grc.ciso.v1`

## Purpose

Audit trail for **CISO-Assistant** AI queries.  Every question submitted through the customer
portal generates a NATS event containing the query text, AI confidence score, cited OSCAL
control IDs, and response timestamp.  Consumers downstream use this stream for compliance
evidence archival, analyst dashboards, and SIEM forwarding.

## Subject pattern

```
kubric.grc.ciso.v1
kubric.grc.ciso.v1.<tenant_id>
```

| Token | Description |
|-------|-------------|
| `kubric` | Platform root |
| `grc` | Domain — Governance, Risk & Compliance |
| `ciso` | Event type — CISO-Assistant interaction |
| `v1` | Schema version |
| `<tenant_id>` | Tenant slug (lowercase, hyphenated) |

Wildcard subscribe: `kubric.grc.ciso.>`

## Payload (JSON)

```json
{
  "type":       "ciso.query.v1",
  "tenant_id":  "acme-corp",
  "question":   "What controls cover multi-factor authentication?",
  "confidence": 0.92,
  "sources":    ["IA-2", "CC6.1", "A.9.4.2"],
  "timestamp":  "2026-02-27T18:45:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `ciso.query.v1` |
| `tenant_id` | string | Tenant identifier (matches JWT claim) |
| `question` | string | Original user question (≤ 2 000 chars) |
| `confidence` | float | AI confidence score 0.0 – 1.0 |
| `sources` | string[] | OSCAL control IDs cited in the answer |
| `timestamp` | string | RFC 3339 UTC timestamp |

## JetStream configuration

Captured by the existing **KUBRIC_GRC** stream (`kubric.grc.>`).

| Setting | Value |
|---------|-------|
| Stream | `KUBRIC_GRC` |
| Subjects | `kubric.grc.>` |
| Retention | 365 days |
| `deny_delete` | `true` |
| `deny_purge` | `true` |

### Consumer groups

| Consumer | Filter subject | Purpose |
|----------|---------------|---------|
| `ciso-audit-trail` | `kubric.grc.ciso.v1.>` | Immutable audit evidence for GRC compliance |
| `compliance-dashboard` | `kubric.grc.ciso.v1.>` | Customer-facing CISO query history |
| `siem-export-grc` | `kubric.grc.>` | External SIEM forwarding (shared with drift events) |

## Producer

| Service | Language | File |
|---------|----------|------|
| KIC API (CISO handler) | Go | `internal/kic/handler_ciso.go` |

## Related subjects

- [`kubric.grc.drift.v1`](K-MB-SUB-007_grc.drift.v1.md) — compliance posture drift events
