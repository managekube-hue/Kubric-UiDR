# KAI Layer 2 — API & Agent Reference

> **Version:** 0.2.0
> **Status:** Complete — import chain verified 2026-02-25
> **Package root:** `kai/`
> **Default port:** `:8100`

---

## Quick start

```bash
# Base install (API + NATS + config only, no heavy AI deps)
pip install -e "kai/"

# Full Layer 2 install (adds crewai, langchain, temporalio, clickhouse-connect, ansible-runner)
pip install -e "kai/[layer2]"

# Start FastAPI server
kai serve
# or:
uvicorn kai.api.main:app --host 0.0.0.0 --port 8100 --reload

# Start Temporal worker (separate process)
kai-worker
```

Environment variables (all prefixed `KUBRIC_`):

| Variable | Default | Description |
|---|---|---|
| `KUBRIC_TENANT_ID` | `default` | Fallback tenant ID |
| `KUBRIC_NATS_URL` | `nats://127.0.0.1:4222` | NATS server |
| `KUBRIC_OLLAMA_URL` | `http://127.0.0.1:11434` | Ollama (local LLM) |
| `KUBRIC_VLLM_URL` | `http://127.0.0.1:8000` | vLLM (fallback) |
| `KUBRIC_OPENAI_API_KEY` | `""` | OpenAI (cloud fallback) |
| `KUBRIC_ANTHROPIC_API_KEY` | `""` | Anthropic (cloud fallback) |
| `KUBRIC_MODEL_NAME` | `llama3.2` | Model name for Ollama/vLLM |
| `KUBRIC_VDR_URL` | `http://127.0.0.1:8081` | VDR service base URL |
| `KUBRIC_KIC_URL` | `http://127.0.0.1:8082` | KIC service base URL |
| `KUBRIC_NOC_URL` | `http://127.0.0.1:8083` | NOC service base URL |
| `KUBRIC_KSVC_URL` | `http://127.0.0.1:8080` | K-SVC base URL |
| `KUBRIC_TEMPORAL_ADDRESS` | `127.0.0.1:7233` | Temporal frontend |
| `KUBRIC_VAPI_API_KEY` | `""` | Vapi voice escalation |
| `KUBRIC_VAPI_PHONE_NUMBER_ID` | `""` | Vapi outbound number |
| `KUBRIC_ALERT_PHONE_NUMBER` | `""` | Destination phone (SMS/voice) |
| `KUBRIC_TWILIO_SID` | `""` | Twilio account SID |
| `KUBRIC_TWILIO_TOKEN` | `""` | Twilio auth token |
| `KUBRIC_TWILIO_FROM` | `""` | Twilio from number |
| `KUBRIC_STRIPE_API_KEY` | `""` | Stripe secret key |
| `KUBRIC_N8N_BASE_URL` | `http://127.0.0.1:5678` | n8n base URL |
| `KUBRIC_ZAMMAD_URL` | `""` | Zammad ticketing URL |
| `KUBRIC_ZAMMAD_TOKEN` | `""` | Zammad API token |
| `KUBRIC_COMPOSIO_API_KEY` | `""` | Composio integration key |
| `KUBRIC_KNOWN_TENANTS` | *(uses TENANT_ID)* | Comma-separated tenant list for FORESIGHT loop |
| `KUBRIC_AUTO_REMEDIATE` | `false` | Auto-submit safe plans to Temporal |
| `KUBRIC_FORESIGHT_INTERVAL` | `1800` | Seconds between FORESIGHT sweeps |
| `KUBRIC_API_HOST` | `0.0.0.0` | Bind host |
| `KUBRIC_API_PORT` | `8100` | Bind port |

---

## HTTP Endpoints

### `GET /healthz`

Liveness probe. Always returns 200.

```json
{"status": "ok", "tenant_id": "demo-tenant"}
```

---

### `GET /readyz`

Readiness probe. Checks NATS connectivity and Ollama reachability.

```json
{
  "status": "ok",
  "checks": {
    "nats": "ok",
    "ollama": "ok"
  }
}
```

`status` is `"degraded"` if any check fails. Returns 200 in both cases (let the load balancer decide).

---

### `POST /v1/triage`

Enrich a raw OCSF event through **KAI-TRIAGE**.

**Request:**
```json
{
  "event": {
    "class_uid": 1001,
    "severity_id": 3,
    "message": "Suspicious process spawn detected",
    "actor": {"process": {"name": "powershell.exe", "cmd_line": "IEX (...)"}},
    "device": {"hostname": "ws-12"}
  },
  "tenant_id": "acme-corp"
}
```

**Response:**
```json
{
  "triage_id": "3f2a...",
  "tenant_id": "acme-corp",
  "severity": "HIGH",
  "mitre_techniques": ["T1059.001", "T1086"],
  "summary": "PowerShell execution via encoded command — likely stage 2 downloader",
  "recommended_action": "Isolate host ws-12 and collect memory dump",
  "confidence": 0.87,
  "model_used": "crewai/ollama/llama3.2"
}
```

---

### `POST /v1/score/{tenant_id}`

Compute the **KiSS health score** for a tenant via **KAI-SENTINEL**.

```
KiSS = vuln × 0.30 + compliance × 0.25 + detection × 0.25 + response × 0.20
```

**Response:**
```json
{
  "tenant_id": "acme-corp",
  "computed_at": 1708900000000,
  "kiss_score": 74.5,
  "vuln_score": 68.0,
  "compliance_score": 87.5,
  "detection_score": 72.0,
  "response_score": 80.0
}
```

---

### `GET /v1/insights/{tenant_id}`

AI-generated security narrative for a QBR / weekly digest. Calls **KAI-SENTINEL** then prompts Ollama.

**Response:**
```json
{
  "tenant_id": "acme-corp",
  "generated_at": 1708900000000,
  "narrative": "Tenant posture improved week-over-week. Three critical CVEs remain unpatched in the web tier. Detection coverage is strong.",
  "top_risks": [
    "CVE-2024-1234 (CVSS 9.8) — unpatched nginx",
    "CIS benchmark drift in container runtime",
    "Elevated brute-force traffic from ASN 12345"
  ],
  "recommended_actions": [
    "Patch nginx to ≥ 1.25.4 within 72 h",
    "Re-run KIC scan after patch to close compliance gap"
  ]
}
```

---

### `POST /v1/remediate`

Generate a remediation plan via **KAI-KEEPER**. Optionally triggers a Temporal `RemediationWorkflow`.

**Request:**
```json
{
  "finding_id": "vdr-abc123",
  "tenant_id": "acme-corp",
  "finding": {
    "cve_id": "CVE-2024-1234",
    "package": "nginx",
    "installed_version": "1.24.0",
    "fixed_version": "1.25.4",
    "severity": "CRITICAL"
  },
  "auto_apply": false
}
```

**Response:**
```json
{
  "plan_id": "plan-uuid",
  "tenant_id": "acme-corp",
  "remediation_type": "package_update",
  "steps": [
    "1. Notify change control window",
    "2. Run ansible playbook: update_nginx.yml",
    "3. Verify nginx -v == 1.25.4",
    "4. Close finding vdr-abc123"
  ],
  "ansible_playbook": "- hosts: web_tier\n  tasks:\n  - name: upgrade nginx\n    apt: name=nginx=1.25.4 state=present",
  "estimated_risk": "low",
  "auto_safe": true,
  "status": "pending"
}
```

If `KUBRIC_AUTO_REMEDIATE=true` and `auto_safe=true`, `status` will be `"submitted"` and the Temporal workflow will have been launched.

---

### `POST /v1/billing/run`

Trigger a **KAI-CLERK** billing aggregation for a tenant. Submits a Temporal `BillingWorkflow` (falls back to direct async execution if Temporal is unavailable).

**Request:**
```json
{
  "tenant_id": "acme-corp",
  "stripe_customer_id": "cus_abc123",
  "billing_period": "2026-02"
}
```

**Response:**
```json
{"status": "submitted", "tenant_id": "acme-corp", "period": "2026-02"}
```

Pricing model (ClickHouse aggregation from `kubric.ocsf_events`):

| Metric | Rate |
|---|---|
| Process events | $5 / 10,000 |
| Network events | $3 / 10,000 |
| Vulnerability findings | $10 / 100 |

---

### `POST /v1/n8n/callback`

Bridge from KAI to n8n for ITSM fan-out (Slack, email, PagerDuty).

**Request:**
```json
{
  "event_type": "triage.enriched",
  "tenant_id": "acme-corp",
  "severity": "CRITICAL",
  "summary": "Ransomware precursor detected on ws-12",
  "incident_id": "triage-uuid"
}
```

Forwards to `{KUBRIC_N8N_BASE_URL}/webhook/kubric-incident`. Non-fatal on n8n failure.

---

## NATS Message Bus

### Subjects consumed (subscriber routing)

| Subject pattern | Handler |
|---|---|
| `kubric.edr.>` | KAI-TRIAGE |
| `kubric.itdr.>` | KAI-TRIAGE |
| `kubric.ndr.>` | KAI-TRIAGE |
| `kubric.vdr.>` | KAI-KEEPER |
| `kubric.grc.>` | KAI-KEEPER |
| `kubric.comm.>` | KAI-COMM |

Layer 1 publishes on `kubric.{tenant_id}.{category}.{class}.v1` — the subscriber strips and routes.

### Subjects published by KAI

| Subject | Publisher | When |
|---|---|---|
| `kubric.kai.triage.enriched` | KAI-TRIAGE | After every enrichment |
| `kubric.health.score.{tenant_id}` | KAI-SENTINEL | After every score computation |
| `kubric.kai.keeper.plan` | KAI-KEEPER | After every remediation plan |
| `kubric.kai.foresight.risk.{tenant_id}` | KAI-FORESIGHT | Every `KUBRIC_FORESIGHT_INTERVAL` seconds |
| `kubric.kai.comm.escalated` | KAI-COMM | After voice/SMS escalation |

---

## AI Agents

### KAI-TRIAGE (`kai/agents/triage.py`)

**Persona:** Senior SOC Analyst
**Trigger:** NATS `kubric.edr.*` / `kubric.ndr.*` / `kubric.itdr.*`
**LLM task:** Classify severity, extract MITRE techniques, produce 2-sentence summary + remediation hint
**CrewAI tools:** `publish_nats_event`
**Output fields:** `severity`, `mitre_techniques[]`, `summary`, `recommended_action`, `confidence`

---

### KAI-SENTINEL (`kai/agents/sentinel.py`)

**Persona:** Security Posture Analyst
**Trigger:** `POST /v1/score/{tenant_id}` or `GET /v1/insights/{tenant_id}`
**LLM task:** Pull live VDR + KIC + ClickHouse metrics, compute KiSS sub-scores, return JSON
**CrewAI tools:** `get_vdr_summary`, `get_kic_summary`, `query_recent_alerts`
**Output fields:** `kiss_score`, `vuln_score`, `compliance_score`, `detection_score`, `response_score`

---

### KAI-KEEPER (`kai/agents/keeper.py`)

**Persona:** DevSecOps Remediation Engineer
**Trigger:** NATS `kubric.vdr.*` / `kubric.grc.*` or `POST /v1/remediate`
**LLM task:** Generate step-by-step fix, optional Ansible playbook, rate auto-safety
**CrewAI tools:** `get_vdr_summary`, `trigger_remediation`
**Output fields:** `remediation_type`, `steps[]`, `ansible_playbook`, `estimated_risk`, `auto_safe`

---

### KAI-COMM (`kai/agents/comm.py`)

**Persona:** Security Communications Officer
**Trigger:** NATS `kubric.comm.*`
**Actions:**
- CRITICAL → Vapi voice call + Twilio SMS
- HIGH → Twilio SMS only
- All → n8n ITSM webhook
**CrewAI tools:** `forward_to_n8n`

---

### KAI-FORESIGHT (`kai/agents/foresight.py`)

**Persona:** Threat Intelligence Analyst
**Trigger:** Background loop — runs every `KUBRIC_FORESIGHT_INTERVAL` seconds (default 30 min)
**LLM task:** Analyse 24h alert trend, project next-24h risk probability per severity tier
**CrewAI tools:** `query_recent_alerts`, `get_vdr_summary`
**Output fields:** `risk_level`, `trend`, `prediction_24h`, `key_findings[]`

---

## LLM Fallback Chain

```
1. Ollama  http://127.0.0.1:11434  (local, ollama/llama3.2)
2. vLLM    http://127.0.0.1:8000   (local, same model)
3. OpenAI  api.openai.com          (if KUBRIC_OPENAI_API_KEY set)
4. Anthropic                        (if KUBRIC_ANTHROPIC_API_KEY set)
```

Source: `kai/core/llm.py` — `complete()` and `complete_json()`.

---

## Temporal Workflows

Both workflows degrade gracefully: if Temporal is unreachable, they execute the activity functions directly as async coroutines.

### `RemediationWorkflow` (`kai/workflows/remediation.py`)

**Queue:** `kubric-remediation`
**Activities:**

| Activity | What it does |
|---|---|
| `validate_finding` | Checks finding still open in VDR |
| `run_ansible` | Runs ansible-runner with the generated playbook |
| `verify_remediation` | Layer 3 placeholder — always returns `True` |
| `close_finding` | PATCH `VDR /findings/{id}` → status=`closed` |

---

### `BillingWorkflow` (`kai/workflows/billing.py`)

**Queue:** `kubric-billing`
**Activities:**

| Activity | What it does |
|---|---|
| `aggregate_usage` | ClickHouse query on `kubric.ocsf_events` PARTITION `YYYY-MM` |
| `create_stripe_invoice` | POST `/v1/invoiceitems` + `/v1/invoices` + finalize |
| `record_invoice` | POST K-SVC `/v1/invoices` for audit log |

---

## CrewAI Tools (`kai/tools/security_tools.py`)

| Tool name | HTTP call | Returns |
|---|---|---|
| `get_vdr_summary` | `GET vdr:8081/findings` | `{"critical":N, "high":N, ...}` |
| `get_kic_summary` | `GET kic:8082/assessments` | `{"framework":"CIS_v8", "pass_rate":87.5}` |
| `query_recent_alerts` | ClickHouse `kubric.kai_triage_results` | `{"CRITICAL":N, "total":N}` |
| `publish_nats_event` | NATS publish | `"published"` |
| `trigger_remediation` | Calls `submit_remediation()` | `"submitted"` |
| `forward_to_n8n` | `POST n8n_base_url/webhook/kubric-incident` | `"forwarded"` |

---

## ClickHouse Tables Required

```sql
-- KAI triage results (written by KAI-TRIAGE via nats-clickhouse-bridge or direct insert)
CREATE TABLE IF NOT EXISTS kubric.kai_triage_results (
    tenant_id       LowCardinality(String),
    triage_id       String,
    timestamp       DateTime64(3, 'UTC'),
    severity        LowCardinality(String),
    mitre_techniques Array(String),
    summary         String,
    confidence      Float32,
    model_used      String,
    _inserted_at    DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, timestamp, triage_id)
TTL timestamp + INTERVAL 90 DAY;
```

See `docs/NOTION-HANDOFF-BLOCKERS.md` Step 5 for the base `kubric.ocsf_events` DDL.

---

## File Map

```
kai/
├── __init__.py                 # version = "0.2.0"
├── config.py                   # Settings (pydantic-settings, all KUBRIC_* env vars)
├── cli.py                      # `kai serve` entrypoint
├── api/
│   ├── __init__.py
│   └── main.py                 # FastAPI app, lifespan, all 8 routes
├── core/
│   ├── __init__.py
│   ├── nats_client.py          # _NATSManager singleton
│   ├── llm.py                  # complete() / complete_json() with fallback chain
│   ├── subscriber.py           # NATS wildcard subscription → agent dispatch
│   └── crew.py                 # CrewAI crew factories for all 5 personas
├── agents/
│   ├── __init__.py             # re-exports all 5 agents
│   ├── triage.py               # KAI-TRIAGE
│   ├── sentinel.py             # KAI-SENTINEL
│   ├── keeper.py               # KAI-KEEPER
│   ├── comm.py                 # KAI-COMM
│   └── foresight.py            # KAI-FORESIGHT
├── tools/
│   ├── __init__.py
│   └── security_tools.py       # 6 CrewAI @tool wrappers
├── workflows/
│   ├── __init__.py
│   ├── billing.py              # BillingWorkflow + activities
│   └── remediation.py          # RemediationWorkflow + activities
└── workers/
    ├── __init__.py
    └── temporal_worker.py      # `kai-worker` CLI — runs both Temporal queues
```
