# K-VENDOR-THV-002 -- TheHive Case Schema

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Incident case management data model            |
| Format      | JSON objects via TheHive REST API              |
| Consumer    | KAI-KEEPER, KAI-COMM, KAI-ANALYST              |

## Purpose

TheHive cases represent confirmed security incidents. KAI-KEEPER
creates and manages cases through the REST API when alerts are
promoted during triage.

## Case Object Fields

| Field            | Type     | Description                          |
|------------------|----------|--------------------------------------|
| title            | string   | Case title (auto-generated or manual)|
| description      | string   | Incident narrative                   |
| severity         | int      | 1 (Low) to 4 (Critical)             |
| tlp              | int      | Traffic Light Protocol (0-3)         |
| pap              | int      | Permissible Actions Protocol (0-3)   |
| status           | string   | New, InProgress, Resolved, Closed    |
| tags             | string[] | MITRE IDs, tenant, source labels     |
| customFields     | object   | Kubric tenant_id, kiss_score, etc.   |
| tasks            | object[] | Remediation checklist items          |
| observables      | object[] | Attached IOCs with data types        |

## Integration Flow

1. KAI-KEEPER POSTs to `https://thehive-{tenant}/api/v1/case`.
2. Remediation tasks are added via `POST /api/v1/case/{id}/task`.
3. Observables from the source alert are attached to the case.
4. KAI-COMM polls case status changes for escalation routing.
5. Case closure triggers metrics update on KAI-HOUSE dashboard.

## Custom Fields (Kubric)

| Custom Field   | Type    | Purpose                            |
|----------------|---------|------------------------------------|
| kubric_tenant  | string  | Multi-tenant isolation key         |
| kiss_score     | float   | KAI-SENTINEL health score         |
| hunt_id        | string  | Link to originating KAI-HUNTER run |
| auto_remediated| boolean | Whether KAI-KEEPER auto-resolved   |
