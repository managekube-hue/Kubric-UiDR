# Kubric-UiDR Production Remediation Plan
**Generated:** 2026-02-27
**Based on:** Full codebase audit — XRO, KAI, NOC, SOC, PSA, GRC, Supabase, Docker, K8s
**Estimated timeline to production-ready:** 18 working days across 6 phases

---

## HOW TO READ THIS PLAN

Each task is tagged with one of:
- `[CLAUDE WRITES]` — I execute this automatically, no user input needed
- `[YOU GET KEY]` — You must obtain a credential/URL and tell me; I then wire it
- `[YOU RUN]` — A shell or GUI action you execute (I show exact command)
- `[YOU DECIDE]` — An architectural or procurement decision only you can make

---

## PHASE 0 — IMMEDIATE CODE FIXES (Day 1, ~2 hours, zero external deps)

These fixes require no credentials, no external services. They are pure code changes I write now.

### P0-1 [CLAUDE WRITES] Fix 3 critical NATS subject naming violations
**Files to edit:**
- `kai/agents/triage.py:57` — change `"kubric.kai.triage.enriched"` → `f"kubric.{tenant_id}.kai.triage.enriched.v1"`
- `kai/agents/keeper.py:64` — change `"kubric.kai.keeper.plan"` → `f"kubric.{tenant_id}.kai.keeper.plan.v1"`
- `kai/agents/comm.py:102` — change `"kubric.kai.comm.dispatched"` → `f"kubric.{tenant_id}.kai.comm.dispatched.v1"`

### P0-2 [CLAUDE WRITES] Fix hardcoded service URLs in security_tools.py
**File:** `kai/tools/security_tools.py:33-36`
Change 4 hardcoded `http://vdr:8081` etc. to use `settings.vdr_url`, `settings.kic_url`, `settings.noc_url`, `settings.ksvc_url` from `kai/config.py` (variables already declared there, just unused)

### P0-3 [CLAUDE WRITES] Fix PostgreSQL schema type mismatches
Three migrations use `UUID` where `kubric_tenants.tenant_id` is `TEXT`:

**File `db/migrations/003_contract_rate_tables.up.sql:9-10`:**
```sql
-- CHANGE FROM:
tenant_id  UUID NOT NULL REFERENCES kubric_tenants(id),
-- CHANGE TO:
tenant_id  TEXT NOT NULL REFERENCES kubric_tenants(tenant_id),
```

**File `db/migrations/004_oscal_ingestion.up.sql:41-42`:**
```sql
-- CHANGE FROM:
tenant_id  UUID NOT NULL REFERENCES kubric_tenants(id),
-- CHANGE TO:
tenant_id  TEXT NOT NULL REFERENCES kubric_tenants(tenant_id),
```

**File `migrations/postgres/005_kai_missing_tables.sql:69`:**
```sql
-- CHANGE FROM:
tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
-- CHANGE TO:
tenant_id  TEXT NOT NULL REFERENCES kubric_tenants(tenant_id) ON DELETE CASCADE,
```

### P0-4 [CLAUDE WRITES] Create missing migration 004 to fix sequence gap
**Create:** `migrations/postgres/004_placeholder.sql`
```sql
-- Placeholder to maintain monotonic migration sequence (005 follows 003 without gap)
-- golang-migrate enforces sequential numbering; this stub maintains sequence integrity.
SELECT 1;
```

### P0-5 [CLAUDE WRITES] Delete the `nul` Windows artifact
**Delete:** `nul` (0-byte Windows null device artifact at repo root, created 2026-02-27)

### P0-6 [CLAUDE WRITES] Scaffold 9 empty Portal component files
All 9 files are 0 bytes and will break `next build` if imported.
Add `export default function X() { return null; }` stub to each:
- `06_K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-001_AssetCard.tsx`
- `K-PSA-PTL-DASH-002_DeploymentWizard.tsx`
- `K-PSA-PTL-DASH-003_ActionApproval.tsx`
- `K-PSA-PTL-DASH-004_KissScorecard.tsx`
- `K-PSA-PTL-DASH-005_RiskDashboard.tsx`
- `K-PSA-PTL-DASH-006_BillingChart.tsx`
- `K-PSA-PTL-LIB-001_api_client.ts`
- `K-PSA-PTL-LIB-002_nats_eventsource.ts`
- `K-PSA-PTL-THEME-001_tenant_branding.css`

### P0-7 [CLAUDE WRITES] Fix Zammad `go:build ignore` tag
`06_K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_zammad_bridge.go` has `//go:build ignore` at line 1. Remove that tag and promote the file to `internal/psa/zammad/` so it compiles and is usable.

### P0-8 [CLAUDE WRITES] Fix verify_remediation stub
**File:** `kai/workflows/remediation.py:72-76`
Implement actual re-scan by calling VDR API to check finding status, then optionally triggering a Nuclei re-scan if `KUBRIC_VERIFY_WITH_NUCLEI=true`. Currently always returns `True`.

### P0-9 [CLAUDE WRITES] Remove unused Go dependencies from go.mod
Run `go mod tidy` equivalent logic — remove packages with zero imports:
- `github.com/neo4j/neo4j-go-driver/v5` → move to `internal/itdr/neo4j/` stub that actually imports it, OR remove
- `github.com/ossf/scorecard/v4` → wire into `K-GRC-SCS-001` supply chain, OR remove
- `github.com/marcboeker/go-duckdb` → remove (deferred to L3 analytics)
- `github.com/sigstore/sigstore` → wire into CI image signing stub, OR remove

**Decision required — see P1 items for which ones to wire vs remove.**

### P0-10 [CLAUDE WRITES] Wire Zammad credentials to Vault in Python
**File:** `kai/psa/zammad.py:31-32`
Implement Vault secret fetch pattern matching the Go services — read from `secret/kubric/psa/zammad` with env var fallback:
```python
# Pattern: Vault first, env var fallback
zammad_secret = vault_client.secrets.kv.read_secret("kubric/psa/zammad")
self.url = zammad_secret["url"] or os.getenv("KUBRIC_ZAMMAD_URL")
self.api_token = zammad_secret["api_token"] or os.getenv("KUBRIC_ZAMMAD_API_TOKEN")
```

---

## PHASE 1 — CREDENTIAL GATHERING (Day 1-2, user-gated)

You must obtain each of these before Phase 2. I show exactly where to get each one and where it goes.

### P1-1 [YOU GET KEY] Supabase / PostgreSQL connection string
**Where to get it:**
1. Go to [supabase.com](https://supabase.com) → your project → Settings → Database
2. Click "Connection string" → select "URI" mode
3. Copy the `postgresql://postgres:[YOUR-PASSWORD]@aws-0-us-west-2.pooler.supabase.com:6543/postgres` string

**Where it goes:**
```
.env → KUBRIC_DATABASE_URL=postgresql://postgres:YOURPASSWORD@aws-0-us-west-2.pooler.supabase.com:6543/postgres
infra/k8s/secrets/kubric-db-credentials.yaml → data.url (base64-encoded)
```

**Note:** Password was shared in a previous session — use that or rotate at Supabase → Settings → Database → Reset password.

---

### P1-2 [YOU GET KEY] Stripe API keys (billing)
**Where to get them:**
1. Go to [dashboard.stripe.com](https://dashboard.stripe.com) → Developers → API keys
2. Copy **Secret key** (`sk_live_...` for prod, `sk_test_...` for staging)
3. Go to Developers → Webhooks → Add endpoint
   - URL: `https://kai.yourdomain.com/stripe/webhook`
   - Events: `customer.subscription.created`, `customer.subscription.updated`, `customer.subscription.deleted`, `invoice.payment_succeeded`, `invoice.payment_failed`
4. Copy the **Webhook signing secret** (`whsec_...`)

**Where they go:**
```
.env → KUBRIC_STRIPE_SECRET_KEY=sk_live_...
.env → KUBRIC_STRIPE_WEBHOOK_SECRET=whsec_...
Vault path: secret/kubric/stripe → fields: secret_key, webhook_secret
K8s: infra/k8s/secrets/kubric-stripe-secret.yaml (SealedSecret)
```

---

### P1-3 [YOU GET KEY] Twilio credentials (SMS alerting)
**Where to get them:**
1. Go to [console.twilio.com](https://console.twilio.com)
2. Dashboard → Account Info: copy **Account SID** (`ACxxx`) and **Auth Token**
3. Phone Numbers → Buy a number (or use existing) → copy the `+1...` number

**Where they go:**
```
.env → KUBRIC_TWILIO_ACCOUNT_SID=ACxxx
.env → KUBRIC_TWILIO_AUTH_TOKEN=...
.env → KUBRIC_TWILIO_FROM_NUMBER=+1...
.env → KUBRIC_ALERT_PHONE_NUMBER=+1... (your on-call number)
Vault path: secret/kubric/twilio → fields: account_sid, auth_token, from_number
```

---

### P1-4 [YOU GET KEY] Vapi.ai credentials (voice alerting)
**Where to get them:**
1. Go to [vapi.ai](https://vapi.ai) → Dashboard → API Keys
2. Copy your **Private Key**
3. Create an assistant in Vapi dashboard for "Kubric Security Alert" with a simple prompt
4. Copy the **Assistant ID**

**Where they go:**
```
.env → KUBRIC_VAPI_API_KEY=...
.env → KUBRIC_VAPI_ASSISTANT_ID=...
Vault path: secret/kubric/vapi → fields: api_key, assistant_id
```

---

### P1-5 [YOU GET KEY] AlienVault OTX API key (threat intelligence)
**Where to get it:**
1. Go to [otx.alienvault.com](https://otx.alienvault.com) → Your Name → Settings
2. Copy **OTX Key** (40-char hex string)

**Where it goes:**
```
.env → KUBRIC_OTX_API_KEY=...
Vault path: secret/kubric/ti → field: otx_api_key
docker-compose → kai-ti service KUBRIC_OTX_API_KEY env var
```

---

### P1-6 [YOU GET KEY] AbuseIPDB API key (IP reputation)
**Where to get it:**
1. Go to [abuseipdb.com](https://www.abuseipdb.com) → Account → API
2. Click **Create Key** → copy it

**Where it goes:**
```
.env → KUBRIC_ABUSEIPDB_API_KEY=...
Vault path: secret/kubric/ti → field: abuseipdb_api_key
```

---

### P1-7 [YOU GET KEY] NVD API key (CVE data — 10x rate limit)
**Where to get it:**
1. Go to [nvd.nist.gov/developers/request-an-api-key](https://nvd.nist.gov/developers/request-an-api-key)
2. Submit form → check email → copy API key

**Where it goes:**
```
.env → KUBRIC_NVD_API_KEY=...
Vault path: secret/kubric/ti → field: nvd_api_key
```

---

### P1-8 [YOU DECIDE] MISP instance
**Two options:**
- **Option A (recommended for MSP):** Deploy MISP in docker-compose — I add it as a service. Takes 15 min. Free.
- **Option B:** Use an existing MISP instance URL + API key.

If Option A: tell me "add MISP to docker-compose" and I write the service definition.
If Option B: provide `MISP_URL=https://misp.yourdomain.com` and `MISP_API_KEY=...`

**Where it goes:**
```
.env → KUBRIC_MISP_URL=...
.env → KUBRIC_MISP_API_KEY=...
Vault path: secret/kubric/ti → fields: misp_url, misp_api_key
```

---

### P1-9 [YOU GET KEY] Zammad API credentials (PSA ticketing)
**Two options:**
- **Option A (recommended):** Deploy Zammad via docker-compose — I add the service. Free.
- **Option B:** Use an existing Zammad instance.

1. In Zammad: Admin → API → click "New Access Token" → name it "kubric-kai" → check all permissions
2. Copy the token

**Where it goes:**
```
.env → KUBRIC_ZAMMAD_URL=https://zammad.yourdomain.com
.env → KUBRIC_ZAMMAD_API_TOKEN=...
Vault path: secret/kubric/psa/zammad → fields: url, api_token
```

---

### P1-10 [YOU GET KEY] Ollama + model (local LLM)
**Where to install it:**
1. Download from [ollama.com/download](https://ollama.com/download) — select your OS
2. Install and run: `ollama serve`
3. Pull the model: `ollama pull llama3.2` (2.0 GB download)

For production on your Proxmox cluster:
```bash
# Run on your GPU/inference node
curl -fsSL https://ollama.com/install.sh | sh
ollama pull llama3.2
ollama pull llama3.1:70b  # Optional: larger model for better analysis
```

**Where it goes:**
```
.env → KUBRIC_OLLAMA_URL=http://YOUR_OLLAMA_HOST:11434
docker-compose → kai service KUBRIC_OLLAMA_URL env var
```

---

### P1-11 [YOU DECIDE] OpenAI / Anthropic fallback LLMs
If you want cloud LLM fallback (when Ollama is unavailable):
- OpenAI: [platform.openai.com/api-keys](https://platform.openai.com/api-keys) → Create new key
- Anthropic: [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys) → Create key

**Where they go:**
```
.env → KUBRIC_OPENAI_API_KEY=sk-...
.env → KUBRIC_ANTHROPIC_API_KEY=sk-ant-...
Vault path: secret/kubric/llm → fields: openai_api_key, anthropic_api_key
```

---

### P1-12 [YOU DECIDE] HashiCorp Vault — production deployment
**For production (Proxmox K8s):** Use the included Helm chart — Vault Helm with Raft storage. I configure the `infra/helm/values/vault-values.yaml`.
- Requires: 3 nodes for HA Raft, each node needs at minimum 1 CPU + 2Gi memory
- Unseal keys will be generated on first init — **store them offline in a password manager**

**For dev (docker-compose):** Already deployed in dev-mode. Acceptable for local development only.

**Action for production:**
```bash
# [YOU RUN] — after K8s cluster is up
helm repo add hashicorp https://helm.releases.hashicorp.com
helm install vault hashicorp/vault \
  --namespace vault --create-namespace \
  -f infra/helm/values/vault-values.yaml

# Then initialise (captures unseal keys):
kubectl -n vault exec vault-0 -- vault operator init \
  -key-shares=5 -key-threshold=3 \
  -format=json > vault-init.json

# *** SAVE vault-init.json somewhere OFFLINE and SECURE ***
# *** NEVER commit vault-init.json to git ***
```

---

### P1-13 [YOU DECIDE] Temporal — cloud or self-hosted
- **Option A:** [Temporal Cloud](https://cloud.temporal.io) — managed service, $25/mo. Easiest.
  - Sign up → create namespace `kubric-prod`
  - Download mTLS certs from Temporal Cloud dashboard
  - Give me: namespace endpoint, client cert path, client key path
- **Option B:** Self-hosted on K8s (included in docker-compose already).
  - Set `KUBRIC_TEMPORAL_URL=temporal:7233` (already default)

---

### P1-14 [YOU GET KEY] GreyNoise, Shodan, Censys (optional TI enrichment)
These provide internet-scale context for IP addresses. All have free tiers.
- **GreyNoise:** [viz.greynoise.io/account](https://viz.greynoise.io/account) → API Key
- **Shodan:** [account.shodan.io](https://account.shodan.io) → API Key (free: 1 query/sec)
- **Censys:** [app.censys.io/account/api](https://app.censys.io/account/api) → API ID + Secret

```
.env → KUBRIC_GREYNOISE_API_KEY=...
.env → KUBRIC_SHODAN_API_KEY=...
.env → KUBRIC_CENSYS_API_ID=...
.env → KUBRIC_CENSYS_SECRET=...
Vault path: secret/kubric/ti/enrichment
```

---

## PHASE 2 — SECRETS INFRASTRUCTURE (Day 3-4)

### P2-1 [CLAUDE WRITES] Create `.env` from `.env.example` with all slots
Expand `/.env.example` to include all 40 environment variables needed across all services, with clear comments on where each comes from (linked to P1 items above).

### P2-2 [CLAUDE WRITES] Update docker-compose to load from `.env`
Replace all 14 hardcoded `dev_password` values in `docker-compose/docker-compose.dev.yml` with `${VARIABLE_NAME}` references that load from `.env`.

**Current (wrong):**
```yaml
POSTGRES_PASSWORD: dev_password
```
**Fixed:**
```yaml
POSTGRES_PASSWORD: ${KUBRIC_DB_PASSWORD}
```

### P2-3 [CLAUDE WRITES] Create all missing K8s Secrets (as SealedSecret templates)
Generate all 9 missing Secret/ConfigMap stubs that K8s manifests reference:

| Secret Name | Referenced By | Fields |
|---|---|---|
| `kubric-db-credentials` | k-svc.yaml | url, username, password |
| `kubric-auth-secrets` | web.yaml | nextauth-secret, authentik-client-id, authentik-client-secret |
| `kubric-stripe-secret` | k-svc.yaml | secret-key, webhook-secret |
| `kubric-twilio-secret` | kai-core.yaml | account-sid, auth-token, from-number |
| `postgresql-credentials` | postgresql-statefulset.yaml | username, password |
| `postgresql-tls` | postgresql-statefulset.yaml | server.crt, server.key |
| `nats-tls` | nats-statefulset.yaml | server.crt, server.key, nkey.seed |
| `clickhouse-credentials` | clickhouse-statefulset.yaml | username, password |
| `kubric-vault-token` | all services | token |

I write the `infra/k8s/secrets/` directory with one YAML per Secret, with placeholder values and instructions to run `kubeseal`.

### P2-4 [YOU RUN] Seal all secrets with kubeseal
After kubeseal is installed on your cluster:
```bash
# Install kubeseal CLI
# macOS: brew install kubeseal
# Linux:
KUBESEAL_VERSION=0.27.0
wget "https://github.com/bitnami-labs/sealed-secrets/releases/download/v${KUBESEAL_VERSION}/kubeseal-${KUBESEAL_VERSION}-linux-amd64.tar.gz"
tar xf kubeseal-*.tar.gz && sudo mv kubeseal /usr/local/bin/

# Seal all secrets (run from repo root)
for f in infra/k8s/secrets/*.yaml; do
  kubeseal --format yaml < "$f" > "${f%.yaml}-sealed.yaml"
done
```
Tell me when done — I move the `-sealed.yaml` files to the right locations and update `kustomization.yaml`.

### P2-5 [CLAUDE WRITES] Deploy SealedSecrets controller to K8s
Add to `infra/k8s/base/kustomization.yaml`:
```yaml
helmCharts:
  - name: sealed-secrets
    repo: https://bitnami-labs.github.io/sealed-secrets
    version: "2.15.3"
    namespace: kube-system
```

### P2-6 [CLAUDE WRITES] Configure Vault Kubernetes auth method
I write the Vault initialization script `scripts/vault-setup.sh` that:
1. Enables the Kubernetes auth method: `vault auth enable kubernetes`
2. Configures cluster endpoint: `vault write auth/kubernetes/config kubernetes_host=...`
3. Creates 5 policies: `kubric-ksvc`, `kubric-vdr`, `kubric-kic`, `kubric-noc`, `kubric-kai`
4. Creates 5 roles binding ServiceAccounts to policies
5. Writes initial secrets to Vault KV paths (secrets provided in Phase 1)

**You run it once after Vault is initialized:**
```bash
# [YOU RUN] — with VAULT_ADDR and VAULT_TOKEN set
bash scripts/vault-setup.sh
```

### P2-7 [CLAUDE WRITES] Deploy External Secrets Operator (ESO)
Add ESO to kustomization so n8n and other services can pull secrets from Vault automatically:
```yaml
# infra/k8s/base/kustomization.yaml addtion
helmCharts:
  - name: external-secrets
    repo: https://charts.external-secrets.io
    version: "0.9.13"
    namespace: external-secrets
    releaseName: external-secrets
```
I also write the `SecretStore` and `ExternalSecret` resources for each service.

---

## PHASE 3 — DATABASE SCHEMA REMEDIATION (Day 4, ~1 hour)

### P3-1 [CLAUDE WRITES] Run P0-3 and P0-4 fixes (already planned)
After the code fixes in Phase 0, run migrations in order against Supabase:

```bash
# [YOU RUN] — requires migrate CLI
# Install: https://github.com/golang-migrate/migrate/releases
# Windows: scoop install migrate or choco install golang-migrate

export KUBRIC_DATABASE_URL="postgresql://postgres:YOURPASSWORD@aws-0-us-west-2.pooler.supabase.com:6543/postgres"

# Apply db/migrations/ (Layer 0 — Foundation)
migrate -database "${KUBRIC_DATABASE_URL}" \
        -path db/migrations up

# Apply migrations/postgres/ (Layer 1-2 extension)
migrate -database "${KUBRIC_DATABASE_URL}" \
        -path migrations/postgres up
```

### P3-2 [CLAUDE WRITES] Apply ClickHouse schemas
```bash
# [YOU RUN] — once ClickHouse is running
for sql in migrations/clickhouse/*.sql; do
  clickhouse-client --host localhost --query "$(cat $sql)"
done
```

### P3-3 [CLAUDE WRITES] Write Supabase RLS validation test
Create `scripts/validate-schema.sql` — a read-only test that:
1. Verifies all 17 tables exist
2. Verifies RLS is enabled on each
3. Checks that `app_tenant_id()` function exists
4. Tests tenant isolation with a mock SET LOCAL

Run with: `psql $KUBRIC_DATABASE_URL -f scripts/validate-schema.sql`

---

## PHASE 4 — DETECTION ASSETS (Day 5-6, mostly automated)

All 120,000+ detection rules are pulled as git submodules. Zero manual rule writing required.

### P4-1 [CLAUDE WRITES] Update `.gitmodules` and Makefile `vendor-pull` target
I write a complete `vendor-pull` Makefile target and `.gitmodules` that pulls all 8 asset sources:

```bash
# [YOU RUN ONCE] — pulls all detection assets (~2GB total)
make vendor-pull
```

The target pulls:
| Asset | Source | Destination | Count |
|---|---|---|---|
| Sigma rules | `github.com/SigmaHQ/sigma` | `vendor/sigma/` | ~3,000 YAML |
| YARA rules | `github.com/Yara-Rules/rules` | `vendor/yara-rules/` | ~1,500 .yar |
| Nuclei templates | `github.com/projectdiscovery/nuclei-templates` | `vendor/nuclei-templates/` | ~8,000 YAML |
| Falco rules | `github.com/falcosecurity/rules` | `vendor/falco/` | ~500 YAML |
| MISP taxonomies | `github.com/MISP/misp-taxonomies` | `vendor/misp/` | ~100 JSON |
| MITRE ATT&CK | STIX bundle download | `vendor/mitre/` | ~600 techniques |
| osquery packs | `github.com/osquery/osquery` | `vendor/osquery/` | ~50 JSON |
| Suricata ET | Emerging Threats download | `vendor/suricata/` | ~50,000 rules |

### P4-2 [CLAUDE WRITES] Add Dockerfile COPY for Nuclei templates
`Dockerfile.api` already copies sigma/yara but is missing nuclei:
```dockerfile
# Add to nuclei-bridge build stage
COPY --chown=kubric:kubric vendor/nuclei-templates/ /opt/kubric/vendor/nuclei-templates/
```

### P4-3 [CLAUDE WRITES] Add Nuclei binary to nuclei-bridge image
The distroless image has no nuclei binary. Fix: use a sidecar pattern or switch the nuclei-bridge base image:
```dockerfile
# Change nuclei-bridge runtime base from distroless to:
FROM projectdiscovery/nuclei:latest AS nuclei
FROM debian:bookworm-slim
COPY --from=nuclei /usr/local/bin/nuclei /usr/local/bin/nuclei
```

### P4-4 [CLAUDE WRITES] Wire Falco DaemonSet to vendor/falco/ rules
`infra/k8s/security-tools/falco-daemonset.yaml` references rules but vendor dir was empty. After P4-1 populates it, I update the Falco K8s ConfigMap to mount the rules correctly.

### P4-5 [CLAUDE WRITES] Create missing ClickHouse table: `kubric.kai_triage_results`
This table is referenced in `kai/tools/security_tools.py:120` but has no DDL anywhere.
I add the DDL to `migrations/clickhouse/005_kai_triage_results.sql`:
```sql
CREATE TABLE IF NOT EXISTS kubric.kai_triage_results (
    tenant_id     LowCardinality(String),
    triage_id     String,
    severity      LowCardinality(String),
    timestamp     DateTime64(3, 'UTC'),
    techniques    Array(String),
    confidence    Float32,
    model_used    String,
    summary       String,
    source_ids    Array(String)
) ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(timestamp))
ORDER BY (tenant_id, toStartOfHour(timestamp), triage_id)
TTL toDateTime(timestamp) + INTERVAL 365 DAY;
```

---

## PHASE 5 — AGENT CODE COMPLETION (Day 7-12)

### P5-1 [CLAUDE WRITES] Implement eBPF build pipeline
eBPF programs must be pre-compiled. I create:
- `agents/coresec/ebpf/execve_hook.c` — process execution tracepoint (attaches to `sys_enter_execve`)
- `agents/coresec/ebpf/openat2_hook.c` — file open FIM tracepoint
- `agents/coresec/ebpf/Makefile` — compiles with `clang -target bpf`
- `.github/workflows/build-ebpf.yml` — CI job that compiles and pushes `.o` artifacts
- `Makefile` `build-ebpf` target

**[YOU RUN] to pre-build:** (Requires Linux with clang/LLVM — can be done in a docker container)
```bash
make build-ebpf
# OR via Docker:
docker run --rm -v $(pwd):/src ubuntu:22.04 bash -c \
  "apt-get install -y clang llvm libbpf-dev && cd /src && make build-ebpf"
```
Output: `vendor/ebpf/execve_hook.o` and `vendor/ebpf/openat2_hook.o`

### P5-2 [CLAUDE WRITES] Implement ETW real kernel provider (Windows)
Replace WMI fallback in `agents/coresec/src/hooks/etw.rs` with real ETW session:
- Enable Windows Security event log provider: `{54849625-5478-4994-A5BA-3E3B0328C30D}`
- Use `StartTraceW` + `ProcessTrace` Win32 APIs via FFI
- Parse Security EventID 4688 (process creation) and 4663 (file access)
- Falls back to WMI if not running as Administrator

### P5-3 [CLAUDE WRITES] Implement disk I/O metrics in PerfTrace
`agents/perftrace/src/agent.rs:176-177` — currently hardcoded 0.
Use `sysinfo` `Disks::refresh_list()` to get cumulative read/write bytes and compute delta per interval.

### P5-4 [CLAUDE WRITES] Wire Vault provisioning in Rust agent registration
`agents/provisioning/src/registration.rs:181-195` generates placeholder tokens.
Implement real Vault AppRole credential provisioning:
1. POST to `$VAULT_ADDR/v1/auth/approle/role/kubric-agent-{tenant_id}/role-id` → get `role_id`
2. POST to generate `secret_id`
3. Return to registering agent as NATS credential
This requires the Vault setup from P2-6 to be complete.

### P5-5 [CLAUDE WRITES] Implement remaining 5 priority KAI personas
In priority order for an MSP CISO assistant:

**ANALYST (Senior Threat Analyst):** Correlates incidents across tenants (anonymised). Publishes to `kubric.{tenant_id}.kai.analyst.v1`.

**HUNTER (Threat Hunter):** Queries ClickHouse for behavioral anomalies (beacon patterns, lateral movement). Triggers RITA scan. Publishes to `kubric.{tenant_id}.kai.hunter.v1`.

**RISK (FAIR Cyber Risk Analyst):** Quantifies financial risk using `pyfair`. Inputs: asset count, threat frequency, vulnerability severity. Output: Annual Loss Expectancy ($). Publishes to `kubric.{tenant_id}.kai.risk.v1`.

**INVEST (Digital Forensics Investigator):** Orchestrates evidence collection via `K-SOC-FR-001_evidence_capture.go`. Builds forensic chain report. Publishes to `kubric.{tenant_id}.kai.invest.v1`.

**BILL (Billing Reconciliation):** Aggregates usage from ClickHouse and reconciles against Stripe invoices. Triggers dispute alerts. Publishes to `kubric.{tenant_id}.kai.billing.v1`.

**[YOU DECIDE]:** Do you want SIMULATE (Purple Team) and DEPLOY (Deployment Security) personas in Phase 5 or defer to Phase 6? They require additional tooling (attack simulation frameworks).

### P5-6 [CLAUDE WRITES] Implement Authentik SSO for Portal
**[YOU GET KEY] first — Authentik setup:**
1. Authentik is already declared in `infra/authentik/` — I write the Helm values
2. Deploy: `helm install authentik authentik/authentik -f infra/authentik/values.yaml`
3. First-run wizard: `https://authentik.yourdomain.com/if/flow/initial-setup/`
4. Create OAuth2 Provider: Applications → Providers → Create → OAuth2/OIDC
   - Name: `kubric-portal`
   - Client ID: auto-generated (copy this)
   - Client Secret: auto-generated (copy this)
   - Redirect URIs: `https://portal.yourdomain.com/api/auth/callback/authentik`
5. Give me: **Client ID**, **Client Secret**, **Authentik URL**

**Then [CLAUDE WRITES]:**
- `Dockerfile.web` build args for `NEXTAUTH_URL`, `AUTHENTIK_CLIENT_ID`, `AUTHENTIK_CLIENT_SECRET`
- Next.js `[...nextauth].ts` route handler
- Portal login page with Authentik OIDC redirect
- Tenant-scoped JWT claims propagation to API calls

### P5-7 [CLAUDE WRITES] Implement Next.js Portal components
After scaffold stubs (P0-6), implement real components:

| Component | What it shows | Data source |
|---|---|---|
| `AssetCard.tsx` | Agent status, OS, last seen | NOC API :8083/agents |
| `DeploymentWizard.tsx` | Step-by-step agent install | Provisioning API, install scripts |
| `ActionApproval.tsx` | Pending auto-remediation approvals | KAI-KEEPER Temporal queue |
| `KissScorecard.tsx` | KiSS score gauge (0-100) with breakdown | KAI-SENTINEL NATS + ClickHouse |
| `RiskDashboard.tsx` | FAIR risk, open criticals, timeline | KAI-RISK output + VDR API |
| `BillingChart.tsx` | Monthly usage by service, invoice history | K-SVC billing API + Stripe |
| `api_client.ts` | Typed fetch client for all 4 Go APIs | — |
| `nats_eventsource.ts` | WebSocket NATS EventSource for live updates | NATS WebSocket (port 9222) |
| `tenant_branding.css` | Per-tenant colour palette, logo URL | K-SVC tenant metadata |

### P5-8 [CLAUDE WRITES] Wire Neo4j Go driver into ITDR
`github.com/neo4j/neo4j-go-driver/v5` is declared in go.mod but never imported. Wire it into:
- `K-SOC-ID-002_neo4j_graph.go` — BloodHound graph queries
- `internal/itdr/neo4j/client.go` — Cypher query helpers for attack path analysis

**[YOU GET KEY]:** Neo4j connection details
- If using the docker-compose Neo4j: `neo4j://neo4j:dev_password@neo4j:7687`
- If using Neo4j AuraDB: [console.neo4j.io](https://console.neo4j.io) → New Instance → copy connection URI
```
.env → KUBRIC_NEO4J_URI=neo4j://localhost:7687
.env → KUBRIC_NEO4J_USER=neo4j
.env → KUBRIC_NEO4J_PASSWORD=...
```

### P5-9 [CLAUDE WRITES] Wire MinIO Go driver into backup scripts
`github.com/minio/minio-go/v7` is in go.mod but `scripts/backup/*.go` uses the MinIO HTTP API directly. Clean this up by using the official Go client in all 4 backup files:
```go
import "github.com/minio/minio-go/v7"
```

### P5-10 [CLAUDE WRITES] Wire OpenSSF Scorecard into GRC supply chain
`github.com/ossf/scorecard/v4` is declared. Wire it into `K-GRC-SCS-001` to run automatic supply chain scores on every dependency update:
- Trigger: `kubric.{tenant_id}.grc.supply-chain.scan.v1` NATS event
- Runs scorecard on the repo URL (extracted from SBOM package URLs)
- Publishes score to ClickHouse `kubric.ocsf_compliance_findings`

### P5-11 [CLAUDE WRITES] Implement 11 stub TI integrations in SOC layer
Promote from stub to working code:
1. `K-SOC-TI-002_abuseipdb.py` — bulk IP blacklist (uses P1-6 key)
2. `K-SOC-TI-003_malware_bazaar.py` — MalwareBazaar hash lookups (free, no key)
3. `K-SOC-TI-004_phishing_tank.py` — PhishingTank URL reputation (free)
4. `K-SOC-TI-007_stix2_parser.py` — parse STIX2 bundles into IOC records
5. `K-SOC-TI-008_stix2_validator.py` — validate STIX2 before ingestion
6. `K-SOC-TI-009_shodan_enrich.py` — enrich source IPs (uses P1-14 key)
7. `K-SOC-TI-010_censys_discovery.py` — internet scan discovery (uses P1-14 key)
8. `K-SOC-TI-011_greynoise_filter.py` — noise reduction (uses P1-14 key)
9. `K-SOC-TI-013_misp_pymisp_client.py` — full pyMISP event/attribute pull
10. `K-SOC-TI-015_ipsum_blocklist.py` — updated IPsum feed sync to ClickHouse
11. `K-SOC-TI-012_wiz_cloud_ti.py` — Wiz cloud risk integration [YOU DECIDE: do you have Wiz? Skip if not]

### P5-12 [CLAUDE WRITES] Implement Trivy, Grype, EPSS, NVD in SOC-VULN
Promote from stub to working code:
- `K-SOC-VULN-002_epss_worker.py` — pull EPSS CSV daily → ClickHouse `epss_scores` table
- `K-SOC-VULN-003_cve_priority.sql` — materialized view joining NVD + EPSS + KEV for priority score
- `K-SOC-VULN-004_trivy_scanner.go` — subprocess Trivy → VDR findings
- `K-SOC-VULN-005_grype_db.go` — subprocess Grype → VDR findings
- `K-SOC-VULN-011_nvd_api_puller.go` — NVD REST API → ClickHouse `nvd_cve` (uses P1-7 key)

### P5-13 [CLAUDE WRITES] Implement OSCAL ingest (GRC)
Promote 5 Python files from stub to working code:
- `K-GRC-OSCAL-001_nist_ingest.py` — parse NIST 800-53 Rev 5 OSCAL catalog
- `K-GRC-OSCAL-002_soc2_mapper.py` — map SOC2 TSC to OSCAL control IDs
- `K-GRC-OSCAL-004_compliance_trestle.py` — generate trestle SSP from assessment results
- `K-GRC-OSCAL-005_regscale_ingest.py` [YOU DECIDE: do you have RegScale? Skip if not]

### P5-14 [CLAUDE WRITES] Implement backup encryption
Add AES-256-GCM encryption to all 4 backup scripts using Vault Transit secrets engine:
```go
// Pattern: encrypt before upload
ciphertext := vault.Transit.Encrypt("kubric-backup", plaintext)
// Store only ciphertext in MinIO
```
Add matching decrypt function to restore scripts.

**[YOU RUN] — after Vault setup:**
```bash
vault secrets enable transit
vault write -f transit/keys/kubric-backup type=aes256-gcm96
```

---

## PHASE 6 — KUBERNETES PRODUCTION HARDENING (Day 13-16)

### P6-1 [CLAUDE WRITES] Add HEALTHCHECK to all Dockerfiles
Add to each Dockerfile:
```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:PORT/healthz || exit 1
```

### P6-2 [CLAUDE WRITES] Add build version ARGs to all Dockerfiles
```dockerfile
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
LABEL org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$GIT_COMMIT" \
      org.opencontainers.image.created="$BUILD_DATE"
```
Wire into CI: `.github/workflows/build-agents.yml` passes `--build-arg VERSION=$(git describe --tags)`.

### P6-3 [CLAUDE WRITES] Configure cert-manager + TLS for all services
```bash
# [YOU RUN]
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set installCRDs=true
```
I write:
- `ClusterIssuer` for Let's Encrypt production (requires your domain + DNS challenge)
- `Certificate` resources for each service ingress
- Ingress objects with TLS annotations

**[YOU GET KEY]:** Your domain name and DNS provider
```
Example: yourdomain.com, DNS provider: Cloudflare
Cloudflare API token: (Settings → API Tokens → Create Token)
```

### P6-4 [CLAUDE WRITES] Implement mTLS between services via Cilium
`infra/k8s/mesh/cilium-cni.yaml` already exists. I complete it:
- Enable Hubble for observability
- Add `CiliumNetworkPolicy` for service-to-service mTLS
- Restrict: kai can only call vdr:8081, kic:8082, noc:8083, ksvc:8080
- Restrict: external traffic only via Ingress

### P6-5 [CLAUDE WRITES] Add Vault IRSA/Kubernetes service account bindings
Complete the ServiceAccount → Vault role bindings that are currently annotated but not configured.
Each service gets:
- ServiceAccount with `vault.hashicorp.com/agent-inject: "true"`
- Vault role mapped to that SA: `vault write auth/kubernetes/role/kubric-ksvc ...`
- Policy granting read on `secret/kubric/*`

### P6-6 [CLAUDE WRITES] Add resource limits to docker-compose
```yaml
deploy:
  resources:
    limits:
      memory: 1G
      cpus: '1.0'
    reservations:
      memory: 512M
```
Add to each service in docker-compose.dev.yml.

### P6-7 [CLAUDE WRITES] Configure ArgoCD App-of-Apps
`infra/argocd/` directory exists but app definitions are incomplete. I write:
- `Application` CRD for each service pointing to `infra/k8s/overlays/prod`
- Sync policy: automated + self-heal
- Health check hooks

**[YOU RUN]:**
```bash
kubectl apply -n argocd -f infra/argocd/app-of-apps.yaml
```

### P6-8 [CLAUDE WRITES] Configure Woodpecker CI for self-hosted pipeline
`infra/woodpecker/` CI config. Woodpecker replaces GitHub Actions for private Gitea:
- Build agents on push to main
- Run Rust tests → Go tests → Python tests in parallel
- Build Docker images and push to Gitea Container Registry
- Trigger ArgoCD sync on successful build

---

## PHASE 7 — OBSERVABILITY & ALERTING (Day 16-18)

### P7-1 [CLAUDE WRITES] Complete Grafana dashboards
`infra/grafana/dashboards/kubric-platform.json` and `kubric-customer.json` — implement both:
- **Platform dashboard:** ClickHouse event rates per tenant, NATS subject throughput, K8s pod health, backup status
- **Customer dashboard:** KiSS score timeline, open vulnerabilities by severity, compliance drift, incident count

### P7-2 [CLAUDE WRITES] Configure Loki alerts for critical events
Add alert rules to Loki that trigger KAI-COMM on:
- Pod crash loops (3+ restarts in 5 min)
- Migration failures
- NATS disconnection
- Backup failures (from `kubric.backup_audit`)

### P7-3 [CLAUDE WRITES] Create operational runbook
`docs/OPERATOR-RUNBOOK.md`:
- Day-0 setup commands in order
- Common failures and fixes
- Backup/restore procedure (step by step)
- Vault unseal procedure
- Adding a new tenant (API call sequence)
- Updating detection rules (make vendor-pull + rolling deploy)

---

## PRODUCTION READINESS CHECKLIST

When all phases are complete, verify:

```
INFRASTRUCTURE
[ ] K8s cluster running (3+ nodes on Proxmox)
[ ] Vault initialised, unsealed, policies applied
[ ] All SealedSecrets decrypted and accessible
[ ] PostgreSQL StatefulSet running, migrations applied
[ ] ClickHouse StatefulSet running, schemas applied
[ ] NATS JetStream cluster running, streams created
[ ] cert-manager issuing Let's Encrypt certs
[ ] ArgoCD syncing from Gitea

SERVICES
[ ] K-SVC (:8080) responds to GET /healthz
[ ] VDR (:8081) responds to GET /healthz
[ ] KIC (:8082) responds to GET /healthz
[ ] NOC (:8083) responds to GET /healthz
[ ] KAI (:8100) responds to GET /healthz
[ ] Portal (:3000) loads login page with Authentik SSO
[ ] Temporal UI accessible at :8088

DETECTION
[ ] make vendor-pull completes — check file counts:
    vendor/sigma/rules: ls -R | grep ".yml" | wc -l    → > 3000
    vendor/yara-rules: ls -R | grep ".yar" | wc -l     → > 1500
    vendor/nuclei-templates: ls -R | grep ".yaml" | wc -l → > 8000
[ ] Sigma engine loads rules: cargo test -- sigma
[ ] YARA engine loads rules: cargo test -- yara
[ ] Nuclei bridge runs a scan against test target

MULTI-TENANT
[ ] Create 2 test tenants via K-SVC API
[ ] Write a finding for tenant A
[ ] Confirm tenant B cannot see tenant A's finding
[ ] Confirm KiSS scores are per-tenant

INTEGRATIONS
[ ] Stripe webhook receives test event
[ ] Twilio sends test SMS
[ ] Zammad creates test ticket from KAI-COMM
[ ] TI feeds pull at least 1 IOC from each source

BACKUPS
[ ] Run backup: make db-backup
[ ] Verify upload in MinIO console (:9001)
[ ] Run restore drill: make restore-drill
[ ] Confirm RTO < 1 hour
```

---

## IMPLEMENTATION ORDER (summary)

| Phase | Days | Claude does | You do |
|---|---|---|---|
| **Phase 0** | Day 1 | 10 code fixes | Review + approve commits |
| **Phase 1** | Day 1-2 | Nothing | Gather 14 credentials from 10 services |
| **Phase 2** | Day 3-4 | .env, docker-compose, K8s secrets, Vault scripts | Run kubeseal, run vault-setup.sh |
| **Phase 3** | Day 4 | Migration fixes, ClickHouse DDL | Run migrate CLI, validate schema |
| **Phase 4** | Day 5-6 | vendor-pull Makefile, Dockerfile fixes, DDL | Run make vendor-pull |
| **Phase 5** | Day 7-12 | All 14 code items (agents, portal, TI, GRC, NOC) | Provide Authentik Client ID/Secret, eBPF build, decisions |
| **Phase 6** | Day 13-16 | Hardening, certs, ArgoCD, Woodpecker | Install cert-manager, ArgoCD, run kubeseal |
| **Phase 7** | Day 16-18 | Dashboards, alerts, runbook | Review and test |

---

## START HERE — TELL ME

Reply with:
1. Which Phase 0 items to execute now (or "all of Phase 0 now")
2. Credentials from Phase 1 that you already have
3. Decisions on: MISP (deploy vs existing), Temporal (cloud vs self-hosted), Zammad (deploy vs existing), Wiz Cloud TI (yes/no), SIMULATE/DEPLOY personas (yes/no)
4. Your domain name (for TLS cert-manager in Phase 6)
5. Whether your Proxmox K8s cluster already exists or needs bootstrap (if new, I write the bootstrap script)
