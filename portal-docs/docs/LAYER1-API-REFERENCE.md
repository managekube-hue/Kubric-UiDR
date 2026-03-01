# Kubric-UiDR Layer 1 — API Reference

> **Status:** Live and tested against Supabase (2026-02-25)
> All four services share the same PostgreSQL instance (`KUBRIC_DATABASE_URL`) and auto-migrate their tables on startup.

---

## Services Overview

| Service | Binary | Default Port | Responsibility |
|---|---|---|---|
| **K-SVC** | `cmd/ksvc` | `:8080` | Tenant / customer lifecycle |
| **VDR** | `cmd/vdr` | `:8081` | Vulnerability findings intake |
| **KIC** | `cmd/kic` | `:8082` | Compliance assessment intake |
| **NOC** | `cmd/noc` | `:8083` | Cluster health + agent heartbeat |

---

## Environment Variables (all services)

| Variable | Used By | Default | Description |
|---|---|---|---|
| `KUBRIC_DATABASE_URL` | All | `postgres://postgres:postgres@127.0.0.1:5432/kubric` | PostgreSQL DSN (Supabase-compatible) |
| `KUBRIC_NATS_URL` | All | `nats://127.0.0.1:4222` | NATS server for event publishing (non-fatal if unavailable) |
| `KSVC_LISTEN_ADDR` | K-SVC | `:8080` | HTTP bind address |
| `VDR_LISTEN_ADDR` | VDR | `:8081` | HTTP bind address |
| `KIC_LISTEN_ADDR` | KIC | `:8082` | HTTP bind address |
| `NOC_LISTEN_ADDR` | NOC | `:8083` | HTTP bind address |

### Supabase connection string format
```
KUBRIC_DATABASE_URL=postgresql://postgres.[project-ref]:[password]@aws-0-[region].pooler.supabase.com:5432/postgres
```

---

## Common Endpoints (all services)

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness probe — always 200 |
| `GET` | `/readyz` | Readiness probe — 200 if Postgres reachable, 503 otherwise |

**Response (200):**
```json
{"status": "ok"}
```

**Response (503):**
```json
{"status": "postgres unavailable", "error": "..."}
```

---

## K-SVC — Tenant API (`:8080`)

Manages tenant/customer records. Every other service validates `tenant_id` against the schema enforced here.

### Postgres Table
```sql
kubric_tenants (tenant_id TEXT PRIMARY KEY, name, plan, status, created_at, updated_at)
```

### NATS Events
```
kubric.{tenant_id}.tenant.lifecycle.v1
{"tenant_id": "acme-corp", "action": "created|updated|deleted"}
```

### Endpoints

#### `POST /tenants` — Create tenant
```json
{
  "tenant_id": "acme-corp",
  "name":      "Acme Corporation",
  "plan":      "pro"
}
```
- `tenant_id`: required, lowercase alphanumeric + hyphens, 2–63 chars
- `name`: required
- `plan`: optional, defaults to `"starter"`

**Response 201:**
```json
{
  "tenant_id":  "acme-corp",
  "name":       "Acme Corporation",
  "plan":       "pro",
  "status":     "active",
  "created_at": "2026-02-25T20:37:15Z",
  "updated_at": "2026-02-25T20:37:15Z"
}
```

#### `GET /tenants` — List tenants
Query params: `?limit=50` (max 500)

#### `GET /tenants/{tenantID}` — Get tenant
Returns 404 if not found.

#### `PATCH /tenants/{tenantID}` — Update tenant
All fields optional. Empty strings leave existing values unchanged.
```json
{"name": "New Name", "plan": "enterprise", "status": "suspended"}
```

#### `DELETE /tenants/{tenantID}` — Delete tenant
Returns 204 No Content.

---

## VDR — Vulnerability Findings API (`:8081`)

Intake API for normalized vulnerability findings from Nuclei, Trivy, Grype, or manual entry.

### Postgres Table
```sql
vdr_findings (id UUID PK, tenant_id, target, scanner, severity, cve_id, title, description, status, raw_json, created_at, updated_at)
-- Indexes: tenant_id, severity
```

### NATS Events
```
kubric.{tenant_id}.vuln.finding.v1
{"finding_id": "uuid", "tenant_id": "acme-corp", "severity": "critical", "action": "created|status_changed"}
```

### Valid Field Values

| Field | Allowed Values |
|---|---|
| `scanner` | `nuclei` `trivy` `grype` `manual` |
| `severity` | `critical` `high` `medium` `low` `informational` |
| `status` | `open` `acknowledged` `resolved` `false-positive` |

### Endpoints

#### `POST /findings` — Submit a finding
```json
{
  "tenant_id":   "acme-corp",
  "target":      "nginx:1.19.0",
  "scanner":     "trivy",
  "severity":    "critical",
  "cve_id":      "CVE-2021-23017",
  "title":       "Off-by-one in nginx resolver",
  "description": "CVSS 9.8 — remote code execution via crafted DNS response",
  "raw_json":    "{...}"
}
```
- `status` is always set to `"open"` on creation
- `cve_id`, `description`, `raw_json` are optional

**Response 201:** Full finding object with UUID `id`.

#### `GET /findings` — List findings
Query params: `?tenant_id=acme-corp&severity=critical&status=open&limit=50`
- `tenant_id` is required
- `severity` and `status` are optional filters

#### `GET /findings/{findingID}` — Get finding
Returns 404 if not found.

#### `PATCH /findings/{findingID}` — Update triage status
```json
{"status": "acknowledged"}
```

---

## KIC — Compliance Assessment API (`:8082`)

Intake API for control assessment results from kube-bench, Lula, OpenSCAP, or manual audits.

### Postgres Table
```sql
kic_assessments (id UUID PK, tenant_id, framework, control_id, title, status, evidence_json, assessed_by, assessed_at, created_at, updated_at)
-- Indexes: tenant_id, framework
```

### NATS Events
```
kubric.{tenant_id}.compliance.assessment.v1
{"assessment_id": "uuid", "tenant_id": "acme-corp", "framework": "CIS-K8s-1.8", "control_id": "CIS.5.1.1", "status": "pass", "action": "created|status_changed"}
```

### Valid Field Values

| Field | Allowed Values |
|---|---|
| `framework` | `NIST-800-53` `CIS-K8s-1.8` `PCI-DSS-4.0` `SOC2` `ISO-27001` |
| `status` | `pass` `fail` `not-applicable` `not-reviewed` |
| `assessed_by` | `lula` `kube-bench` `openscap` `manual` |

### Endpoints

#### `POST /assessments` — Submit an assessment
```json
{
  "tenant_id":     "acme-corp",
  "framework":     "CIS-K8s-1.8",
  "control_id":    "CIS.5.1.1",
  "title":         "Ensure RBAC is enabled on the API server",
  "status":        "pass",
  "assessed_by":   "kube-bench",
  "assessed_at":   "2026-02-25T20:00:00Z",
  "evidence_json": "{\"check\":\"kube-bench v0.7.3\",\"result\":\"PASS\"}"
}
```
- `assessed_at` is optional (defaults to now)
- `status` defaults to `"not-reviewed"` if omitted
- `assessed_by` defaults to `"manual"` if omitted

**Response 201:** Full assessment object with UUID `id`.

#### `GET /assessments` — List assessments
Query params: `?tenant_id=acme-corp&framework=CIS-K8s-1.8&status=fail&limit=50`

#### `GET /assessments/{assessmentID}` — Get assessment

#### `PATCH /assessments/{assessmentID}` — Update status + evidence
```json
{
  "status":       "pass",
  "evidence_json": "{\"remediation\": \"RBAC enabled 2026-02-25\"}"
}
```
- `evidence_json` is optional — omitting it preserves the existing value

---

## NOC — Operations API (`:8083`)

Tracks registered clusters and agent health via heartbeat. Used by the Kubric dashboard and alerting.

### Postgres Tables
```sql
noc_clusters (id UUID PK, tenant_id, name, provider, version, status, last_seen, created_at)
noc_agents   (id UUID PK, tenant_id, cluster_id, hostname, agent_type, version, status, last_heartbeat, created_at)
-- Unique constraint on noc_agents (tenant_id, hostname, agent_type) — enables upsert heartbeats
```

### NATS Events
```
kubric.{tenant_id}.noc.cluster.v1  → {"cluster_id","tenant_id","status","action":"registered|updated|removed"}
kubric.{tenant_id}.noc.agent.v1    → {"agent_id","tenant_id","hostname","agent_type","action":"heartbeat|registered"}
```

### Valid Field Values

| Field | Allowed Values |
|---|---|
| `provider` | `k8s` `eks` `gke` `aks` `proxmox` |
| cluster `status` | `healthy` `degraded` `critical` `unknown` |
| `agent_type` | `coresec` `netguard` `perftrace` `watchdog` |

### Cluster Endpoints

#### `POST /clusters` — Register a cluster
```json
{
  "tenant_id": "acme-corp",
  "name":      "prod-k8s",
  "provider":  "k8s",
  "version":   "1.29.2"
}
```
- `status` starts as `"unknown"` until updated
- `provider` defaults to `"k8s"`

#### `GET /clusters` — List clusters
Query params: `?tenant_id=acme-corp&limit=50`

#### `GET /clusters/{clusterID}` — Get cluster

#### `PATCH /clusters/{clusterID}` — Update status/version
```json
{"status": "healthy", "version": "1.30.0"}
```
Both fields are optional. Refreshes `last_seen` on every call.

#### `DELETE /clusters/{clusterID}` — Deregister cluster
Returns 204 No Content.

### Agent Endpoints

#### `POST /agents/heartbeat` — Agent heartbeat (upsert)
Called by every Kubric agent binary on a 30–60 second tick.
- First call: creates the agent record
- Subsequent calls: updates `last_heartbeat`, `version`, and `status = "online"`
- Identity key: `(tenant_id, hostname, agent_type)` — no stored ID required

```json
{
  "tenant_id":  "acme-corp",
  "cluster_id": "dc3a7083-9dc3-4a9f-b2c5-1dcb125e93d6",
  "hostname":   "node-01.acme.internal",
  "agent_type": "coresec",
  "version":    "0.1.1"
}
```
- `cluster_id` is optional (empty string for standalone agents)

**Response 200:** Agent object with stable `id` (same UUID across all heartbeats from same node).

#### `GET /agents` — List agents
Query params: `?tenant_id=acme-corp&cluster_id=uuid&limit=100`

#### `GET /agents/{agentID}` — Get agent by UUID

---

## Error Responses

All endpoints return consistent JSON error bodies:
```json
{"error": "descriptive error message"}
```

| HTTP Code | Meaning |
|---|---|
| 400 | Malformed JSON request body |
| 404 | Resource not found |
| 409 | Conflict (duplicate tenant_id on create) |
| 422 | Validation failure (invalid field value, bad tenant_id format) |
| 500 | Internal server error (Postgres failure) |
| 503 | Service unavailable (Postgres unreachable — readyz only) |

---

## Running Locally (Dev Stack)

### Prerequisites
- Go ≥ 1.22
- PostgreSQL (or Supabase free tier)
- NATS (optional — services start without it)

### Start all 4 services
```bash
export KUBRIC_DATABASE_URL="postgresql://postgres:[password]@[host]:5432/postgres"
export KUBRIC_NATS_URL="nats://127.0.0.1:4222"

go run ./cmd/ksvc &   # :8080
go run ./cmd/vdr  &   # :8081
go run ./cmd/kic  &   # :8082
go run ./cmd/noc  &   # :8083
```

### Verify all services are ready
```bash
curl http://localhost:8080/readyz
curl http://localhost:8081/readyz
curl http://localhost:8082/readyz
curl http://localhost:8083/readyz
```

All should return `{"status":"ok"}`.

### Database tables created automatically on startup
- `kubric_tenants`
- `vdr_findings`
- `kic_assessments`
- `noc_clusters`
- `noc_agents`

---

## NATS Subject Map (Layer 1)

| Subject | Publisher | Trigger |
|---|---|---|
| `kubric.{tid}.tenant.lifecycle.v1` | K-SVC | Tenant create / update / delete |
| `kubric.{tid}.vuln.finding.v1` | VDR | Finding submitted or triaged |
| `kubric.{tid}.compliance.assessment.v1` | KIC | Assessment submitted or re-assessed |
| `kubric.{tid}.noc.cluster.v1` | NOC | Cluster registered / status updated / removed |
| `kubric.{tid}.noc.agent.v1` | NOC | Agent heartbeat received |

> All subjects follow the locked format: `kubric.{tenant_id}.{category}.{class}.v1`
