# K-MB-SUB-008 — Service Ticket (PSA)

> NATS subject mapping reference for PSA (Professional Services Automation) ticket events in the Kubric UIDR platform.

## Subject Pattern

```
kubric.svc.ticket.v1
kubric.svc.ticket.v1.<tenant_id>
kubric.svc.ticket.v1.<tenant_id>.<ticket_id>
```

Wildcard subscription: `kubric.svc.ticket.>`

## Publisher

| Component | Language | Description |
|-----------|----------|-------------|
| **K-SVC** | Go | Kubric Service Management bridge. Integrates with the PSA platform (Zammad) to publish ticket lifecycle events (creation, updates, assignments, resolution) to NATS. Also receives automated ticket creation requests from other Kubric subsystems (KAI-TRIAGE, KAI-KEEPER) and translates them into Zammad API calls. |

## Consumer(s)

| Consumer | Language | Role |
|----------|----------|------|
| **KAI-CLERK** | Python (CrewAI) | Automated service desk AI agent. Performs time entry automation, SLA tracking, ticket triage/routing, response drafting, and billing integration. Ensures every security event that requires human action has a tracked, billable ticket. |

## Payload (OCSF Class)

- **OCSF Class**: Incident (`8002`)
- **OCSF Category**: Application Activity
- **Schema Version**: OCSF 1.1
- **Serialization**: Protobuf 3 (wire), JSON (debug/replay)

The OCSF Incident class is extended with PSA-specific fields for ticket management, SLA tracking, and billing.

## Fields

| # | Field | Protobuf Type | OCSF Mapping | Description |
|---|-------|---------------|--------------|-------------|
| 1 | `ticket_id` | `string` | `finding_info.uid` | Unique ticket identifier from Zammad (e.g., `"ZAM-2024-001847"`). |
| 2 | `customer_id` | `string` | `metadata.tenant_uid` | Customer/tenant identifier. Maps to Zammad organization. |
| 3 | `severity` | `uint32` | `severity_id` | Ticket severity: 1=Low, 2=Medium, 3=High, 4=Critical. Aligns with OCSF severity_id. |
| 4 | `type` | `string` | `type_name` | Ticket type: `"incident"`, `"alert"`, `"vulnerability"`, `"compliance"`, `"change_request"`, `"service_request"`. |
| 5 | `description` | `string` | `message` | Ticket description/summary text. For auto-generated tickets, contains structured context from the originating event. |
| 6 | `assigned_to` | `string` | `assignee.user.name` | Assigned analyst or team (e.g., `"soc-tier1"`, `"jsmith"`, `"kai-auto"`). Value `"kai-auto"` indicates automated handling. |
| 7 | `sla_deadline` | `uint64` | `unmapped.sla_deadline` | SLA response deadline as Unix epoch nanoseconds. Computed from ticket severity and customer SLA tier. |
| 8 | `tenant_id` | `string` | `metadata.tenant_uid` | Kubric tenant identifier (same as `customer_id` in most deployments). |
| 9 | `timestamp_ns` | `uint64` | `time` | Ticket event timestamp in nanoseconds since Unix epoch. |
| 10 | `status` | `string` | `status` | Ticket status: `"new"`, `"open"`, `"pending"`, `"on_hold"`, `"resolved"`, `"closed"`. |
| 11 | `priority` | `uint32` | `unmapped.priority` | Internal priority ranking (1=Highest, 2=High, 3=Normal, 4=Low). May differ from severity based on SLA tier. |
| 12 | `source_event` | `string` | `unmapped.source_event` | NATS subject of the originating event (e.g., `"kubric.ndr.beacon.v1.tenant-acme.sensor-01"`). Links ticket to its root cause. |
| 13 | `source_msg_id` | `string` | `unmapped.source_msg_id` | `Nats-Msg-Id` of the originating event for full traceability. |
| 14 | `time_entries_min` | `uint32` | `unmapped.time_entries_min` | Total time logged against this ticket in minutes. Updated on each time entry. |
| 15 | `billable` | `bool` | `unmapped.billable` | Whether time entries for this ticket are billable to the customer. |
| 16 | `tags` | `repeated string` | `unmapped.tags` | Ticket tags for categorization (e.g., `["c2-beacon", "auto-generated", "mitre-t1071"]`). |

## PSA Integration (Zammad)

K-SVC integrates bidirectionally with Zammad as the PSA platform:

### Zammad to NATS (ticket lifecycle events)

```
Zammad Webhook
      |
      v
  K-SVC (Go)
      |
      +-- Normalize to OCSF Incident
      +-- Publish to kubric.svc.ticket.v1.<tenant_id>.<ticket_id>
```

K-SVC registers a webhook in Zammad that fires on ticket create, update, and close events. Each webhook payload is normalized into the OCSF Incident schema and published to NATS.

### NATS to Zammad (automated ticket creation)

```
KAI-TRIAGE / KAI-KEEPER
      |
      +-- NATS Request: kubric.svc.ticket.v1.create
      |
      v
  K-SVC (Go)
      |
      +-- Create ticket via Zammad REST API
      +-- Publish confirmation to kubric.svc.ticket.v1.<tenant_id>.<ticket_id>
```

Other Kubric subsystems create tickets by publishing a request to `kubric.svc.ticket.v1.create`. K-SVC consumes these requests, creates the ticket in Zammad, and publishes the confirmed ticket event with the assigned `ticket_id`.

### Zammad configuration

| Setting | Value |
|---------|-------|
| Zammad API endpoint | `https://zammad.kubric.internal/api/v1` |
| Authentication | API token (stored in Vault at `secret/k-svc/zammad-api-token`) |
| Webhook events | `ticket.create`, `ticket.update`, `ticket.close` |
| Organization mapping | Kubric `tenant_id` maps to Zammad Organization |
| Group mapping | Ticket `type` maps to Zammad Group (e.g., `incident` -> `SOC`, `vulnerability` -> `VulnMgmt`) |

## Automated Billing Time Tracking

KAI-CLERK automates time entry management for MSP/MSSP billing:

### Auto-time rules

| Event Type | Auto-Time Entry | Duration | Billable | Description |
|------------|-----------------|----------|----------|-------------|
| Alert triage (automated) | Yes | 5 min | Yes | KAI-TRIAGE automated alert analysis and enrichment |
| Alert triage (manual escalation) | Yes (start timer) | Actual | Yes | Timer starts on assignment, stops on next status change |
| Vulnerability scan | Yes | 15 min per host | Yes | Scheduled scan execution and result processing |
| Compliance check | Yes | 10 min per framework | Yes | Automated compliance assessment |
| Auto-remediation | Yes | 10 min | Yes | Automated patch or config fix execution |
| Manual remediation | Yes (start timer) | Actual | Yes | Timer tracks analyst hands-on time |
| Ticket response | Yes | 5 min | Yes | Automated response draft + review |
| Monthly report | Yes | 30 min per tenant | Yes | Automated report generation |

### Time entry flow

```
kubric.svc.ticket.v1 (ticket event)
        |
        v
   KAI-CLERK
        |
        +-- Determine time entry rule
        |       |
        |       +-- Fixed-time event --> Create time entry with preset duration
        |       +-- Timer event     --> Start/stop timer, compute actual duration
        |
        +-- Create time entry in Zammad via K-SVC
        |       POST /api/v1/ticket_articles
        |       + time_unit field
        |
        +-- Update billing system
        |       --> Aggregate time entries per customer per billing period
        |       --> Generate invoice line items
        |
        +-- Publish updated ticket event with time_entries_min
```

### SLA tracking

| SLA Tier | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| **Platinum** | 15 min response, 1h resolution | 30 min response, 4h resolution | 2h response, 8h resolution | 4h response, 24h resolution |
| **Gold** | 30 min response, 2h resolution | 1h response, 8h resolution | 4h response, 24h resolution | 8h response, 48h resolution |
| **Silver** | 1h response, 4h resolution | 2h response, 24h resolution | 8h response, 48h resolution | 24h response, 72h resolution |

KAI-CLERK monitors `sla_deadline` on every ticket event. When a ticket approaches its SLA deadline (configurable warning threshold: 75% of SLA time elapsed), KAI-CLERK:
1. Sends escalation notifications via Slack/PagerDuty.
2. Re-prioritizes the ticket in Zammad.
3. Publishes an updated ticket event with elevated priority.

## JetStream Configuration

```json
{
  "stream": {
    "name": "KUBRIC_SVC",
    "subjects": ["kubric.svc.>"],
    "retention": "limits",
    "max_age": "365d",
    "max_bytes": 10737418240,
    "max_msg_size": 65536,
    "storage": "file",
    "num_replicas": 3,
    "discard": "old",
    "duplicate_window": "2m",
    "deny_delete": true,
    "deny_purge": true
  }
}
```

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| `name` | `KUBRIC_SVC` | Dedicated stream for service management events. |
| `subjects` | `kubric.svc.>` | Captures all service management event types. |
| `max_age` | `365d` | Ticket history retained for 1 year for billing reconciliation and audit purposes. |
| `max_bytes` | `10 GB` (10737418240) | Ticket events are low volume (hundreds to low thousands per day). 10 GB is generous for 1-year retention. |
| `max_msg_size` | `64 KB` | Ticket descriptions can be verbose; 64 KB accommodates detailed incident narratives. |
| `deny_delete` | `true` | Billing audit trail integrity — individual messages cannot be deleted. |
| `deny_purge` | `true` | Billing audit trail integrity — stream cannot be purged. |

## Consumer Groups

| Consumer Name | Durable | Filter Subject | Deliver Policy | Ack Policy | Max Deliver | Ack Wait |
|---------------|---------|----------------|----------------|------------|-------------|----------|
| `kai-clerk-ticket` | Yes | `kubric.svc.ticket.v1.>` | `all` | `explicit` | 5 | 60s |
| `billing-aggregator` | Yes | `kubric.svc.ticket.v1.>` | `all` | `explicit` | 10 | 120s |
| `sla-monitor` | Yes | `kubric.svc.ticket.v1.>` | `all` | `explicit` | 5 | 15s |
| `siem-export-svc` | Yes | `kubric.svc.>` | `all` | `explicit` | 3 | 60s |

- **kai-clerk-ticket**: Primary consumer for automated time entry, ticket routing, and response drafting.
- **billing-aggregator**: Aggregates time entries into billing periods for invoice generation. Aggressive retry to ensure no billable time is lost.
- **sla-monitor**: Low-latency consumer (15s ack wait) dedicated to SLA deadline tracking and escalation.
- **siem-export-svc**: External SIEM forwarding for service management audit trails.

## Example (NATS CLI)

### Publish a test ticket event

```bash
nats pub kubric.svc.ticket.v1.tenant-acme.ZAM-2024-001847 \
  --header="Nats-Msg-Id:ticket-$(uuidgen)" \
  '{"ticket_id":"ZAM-2024-001847","customer_id":"tenant-acme","severity":4,"type":"incident","description":"C2 beacon detected: 192.168.1.42 -> 198.51.100.77 (score: 0.92). Automated containment initiated. Analyst review required for lateral movement assessment.","assigned_to":"soc-tier1","sla_deadline":1718903600000000000,"tenant_id":"tenant-acme","timestamp_ns":1718900000000000000,"status":"new","priority":1,"source_event":"kubric.ndr.beacon.v1.tenant-acme.sensor-01","source_msg_id":"beacon-a1b2c3d4","time_entries_min":5,"billable":true,"tags":["c2-beacon","auto-generated","mitre-t1071","critical"]}'
```

### Request automated ticket creation

```bash
nats req kubric.svc.ticket.v1.create \
  '{"customer_id":"tenant-acme","severity":3,"type":"vulnerability","description":"CVE-2024-3094 (xz-utils) detected on host-01. SSVC: ACT. Auto-patch initiated.","assigned_to":"kai-auto","tags":["auto-generated","cve-2024-3094","auto-patch"]}'
```

### Subscribe to all ticket events

```bash
nats sub "kubric.svc.ticket.>"
```

### Create the SVC stream

```bash
nats stream add KUBRIC_SVC \
  --subjects="kubric.svc.>" \
  --retention=limits \
  --max-age=365d \
  --max-bytes=10737418240 \
  --max-msg-size=65536 \
  --storage=file \
  --replicas=3 \
  --discard=old \
  --dupe-window=2m \
  --deny-delete \
  --deny-purge
```

### Create the KAI-CLERK consumer

```bash
nats consumer add KUBRIC_SVC kai-clerk-ticket \
  --filter="kubric.svc.ticket.v1.>" \
  --ack=explicit \
  --deliver=all \
  --max-deliver=5 \
  --wait=60s \
  --pull \
  --durable="kai-clerk-ticket"
```

### Check SLA status

```bash
nats consumer info KUBRIC_SVC sla-monitor
```

## Notes

- **Bidirectional bridge**: K-SVC is both a publisher and a consumer on the NATS bus. It publishes ticket events from Zammad webhooks and consumes ticket creation requests from other Kubric subsystems. This bidirectional pattern is unique among Kubric NATS subjects.
- **Request-reply for creation**: Ticket creation uses NATS request-reply pattern (`nats req`) rather than fire-and-forget publish. The requesting agent receives a reply containing the assigned `ticket_id` for correlation tracking.
- **Zammad dependency**: K-SVC abstracts the PSA platform behind a clean NATS interface. If Zammad is replaced with another PSA tool in the future, only K-SVC needs to be updated; all consumers (KAI-CLERK, billing, SLA monitor) continue to work unchanged.
- **Billing accuracy**: KAI-CLERK applies conservative rounding (always round up to the nearest 5-minute increment) for auto-generated time entries to ensure billing accuracy favors the MSP.
- **Tenant isolation**: Ticket events include `tenant_id` in both the subject hierarchy and payload. KAI-CLERK processes tickets across all tenants but generates billing reports per tenant.
- **Wire format**: Production payloads use Protobuf 3. JSON examples above are for illustration only.
