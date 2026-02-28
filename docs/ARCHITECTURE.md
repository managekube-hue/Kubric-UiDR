# Kubric Platform Architecture

> Multi-tenant MSSP/MDR security platform. 5 Rust agents, 4 Go API services,
> 13 KAI AI personas (Python/CrewAI), Next.js 14 portal, full K8s/GitOps infra.

---

## System Overview

```
                         +-----------------+
                         |   Next.js 14    |
                         |   Portal (L5)   |
                         | Authentik OIDC  |
                         +--------+--------+
                                  |
                    +-------------+-------------+
                    |                           |
              +-----+------+            +------+------+
              |  Go APIs   |            | Stripe      |
              |  K-SVC     |            | Billing     |
              |  VDR/KIC   |            +------+------+
              |  NOC       |                   |
              +-----+------+            +------+------+
                    |                   |  KAI Python  |
                    |                   |  13 Personas |
              +-----+------+           |  CrewAI      |
              |   NATS     |<--------->+------+------+
              | JetStream  |                   |
              +-----+------+           +------+------+
                    |                   |  Temporal    |
         +----------+----------+       |  Workflows   |
         |          |          |       +--------------+
    +----+---+ +---+----+ +---+----+
    |CoreSec | |NetGuard| |PerfTrace|
    | (Rust) | | (Rust) | | (Rust)  |
    +--------+ +--------+ +--------+
    |Watchdog| |Provisioning|
    | (Rust) | |  (Rust)    |
    +--------+ +------------+
```

---

## Agent Architecture (Rust)

All agents share:
- **Async runtime**: tokio (full features)
- **Messaging**: async-nats (JetStream)
- **Serialization**: serde + serde_json + prost (protobuf)
- **Integrity**: blake3 hashing, ed25519 signatures
- **Compression**: zstd (dictionary-based delta)
- **Observability**: tracing + tracing-subscriber + OpenTelemetry (OTLP)
- **Config**: dotenvy + config crate
- **OCSF compliance**: All events use OCSF class UIDs

### CoreSec (agents/coresec/)

Endpoint detection and response agent. Monitors processes, files, and system activity.

```
coresec/src/
  main.rs          -- entry point
  agent.rs         -- 5-sec sysinfo poll loop, NATS publish
  config.rs        -- env-based configuration
  event.rs         -- ProcessEvent (OCSF 4007)
  fim.rs           -- File Integrity Monitoring (inotify/ReadDirectoryChangesW)
  governor.rs      -- Token-bucket rate limiter for NATS publishing
  detection/
    mod.rs         -- DetectionEngine combinator (Sigma + YARA)
    sigma.rs       -- Native Sigma YAML evaluator (no external crate)
    yara.rs        -- yara-x 0.12 scanner
  hooks/
    mod.rs         -- Platform abstraction, HookEvent enum, enum dispatch
    ebpf.rs        -- Linux eBPF via aya (tracepoints: execve, openat2)
    etw.rs         -- Windows ETW via Win32 FFI (Kernel-Process provider)
  ml/
    mod.rs         -- Module exports
    anomaly.rs     -- candle feedforward NN (12-64-32-1), safetensors model
```

**Key design decisions:**
- **Enum dispatch** for hooks (not `dyn Trait`) to avoid E0038 with async fn
- **cfg-gated** platform code: ebpf behind `#[cfg(all(target_os = "linux", feature = "ebpf"))]`, etw behind `#[cfg(target_os = "windows")]`
- **Graceful fallback**: if no kernel hooks available, falls back to sysinfo polling
- **ML scoring** is optional: returns 0.0 score if no model file present
- **FIM** uses platform-native APIs (inotify on Linux, ReadDirectoryChangesW on Windows)

### NetGuard (agents/netguard/)

Network detection and response sensor. 10G packet capture, DPI, IDS, TLS inspection.

```
netguard/src/
  main.rs          -- entry point
  capture.rs       -- pcap capture loop, pnet dissection, OCSF 4001
  config.rs        -- interface selection, NATS URL
  dpi.rs           -- DPI engine (nDPI + heuristic fallback)
  ndpi_ffi.rs      -- nDPI dynamic loading via dlopen/LoadLibraryA
  ids.rs           -- Signature-based IDS, Suricata-compatible rules
  tls.rs           -- TLS handshake parser, SNI, JA3 fingerprints
  rita_client.rs   -- RITA HTTP client (GPL-3.0 boundary)
  ipsum_lookup.rs  -- In-memory IP reputation database
```

**Key design decisions:**
- **nDPI GPL boundary**: loaded at runtime via dlopen, zero compile-time link
- **Heuristic fallback**: when nDPI unavailable, uses byte-pattern + port-based L7 classification
- **RITA GPL boundary**: HTTP-only communication, no Go imports from activecm/rita
- **IPsum lookup**: in-memory hash map, threat_score >= 2 threshold (present in 2+ blocklist feeds)

**Supported DPI protocols:**
TLS, HTTP, DNS, SSH, SMB, SMB2, RDP, SMTP, IMAP, FTP, MySQL, PostgreSQL

### PerfTrace (agents/perftrace/)

System performance monitoring agent. Collects host metrics at 10-second intervals.

```
perftrace/src/
  main.rs          -- entry point
  agent.rs         -- 10-sec collection loop, OCSF 4004 events, OTel spans
  metrics.rs       -- HostMetric struct definition
```

**Metrics collected:**
cpu_percent, memory_used/total, disk_read/write_bytes, net_rx/tx_bytes, load_avg_1m

### Watchdog (agents/watchdog/)

OTA update and agent lifecycle manager. TUF-based secure updates with delta compression.

```
watchdog/src/
  main.rs          -- entry point
  config.rs        -- tenant_id, tuf_repo_url, update_interval
  manifest.rs      -- blake3 + ed25519 binary manifest verification
  orchestrator.rs  -- Update loop: check → download → verify → apply → restart
  tuf_updater.rs   -- TUF update client with delta patch support
  zstd_delta.rs    -- Dictionary-based zstd delta compression
```

**Update flow:**
1. Check TUF repository for new version metadata
2. Download delta patch (`.delta.zst`) or full binary
3. Apply delta: decompress using old binary as dictionary
4. Verify new binary against signed manifest (blake3 hash + ed25519 sig)
5. Atomic replace + restart target agent
6. Rollback if post-install verification fails

### Provisioning (agents/provisioning/)

Agent onboarding and endpoint enrollment. Generates platform-specific install scripts.

```
provisioning/src/
  main.rs            -- entry point
  config.rs          -- nats_url, api_base_url, binary_dir, vault_addr
  registration.rs    -- NATS-based agent registration
  install_script.rs  -- Cross-platform install script generator
  fingerprint.rs     -- blake3 binary validation
```

**Install script targets:**
- **Linux**: bash + systemd service unit + blake3 hash verification
- **Windows**: PowerShell + SC.exe Windows Service + env vars
- **macOS**: bash + launchd plist + Vault auth

**Registration flow:**
1. New endpoint downloads and runs install script
2. Agent binary downloaded from TUF repository
3. blake3 hash verified against expected value
4. Agent sends registration request via NATS
5. Provisioning service validates binary fingerprint
6. On approval: issues NATS token + Vault AppRole credentials
7. Agent starts with tenant-specific configuration

---

## Go API Services (Layer 1)

All services share:
- **Router**: chi with middleware chain
- **Auth**: JWT (HS256) with tenant context extraction
- **RBAC**: 4 roles (admin, analyst, agent, readonly)
- **Rate limiting**: per-IP token bucket (100 req/s, burst 200)
- **Database**: pgx (Postgres) with Row-Level Security
- **Messaging**: async NATS publish for event streaming

| Service | Port | Responsibility |
|---------|------|---------------|
| K-SVC | 8080 | Tenant CRUD, billing integration |
| VDR | 8081 | Vulnerability findings + EPSS enrichment |
| KIC | 8082 | Compliance assessments |
| NOC | 8083 | Cluster + agent management |

### Billing (services/k-svc/billing/)

| Component | Description |
|-----------|-------------|
| `stripe.go` | Stripe SDK integration: customers, subscriptions, usage records, checkout, billing portal |
| `metering.go` | NATS usage event subscriber, ClickHouse aggregation, SHA-256 Merkle root for audit trail |

---

## KAI AI Orchestration (Layer 2)

Python-based AI layer using CrewAI framework. 13 specialized personas, each with a
focused role, goal, and backstory for LLM-driven security operations.

### Personas

| Persona | Module | Responsibility |
|---------|--------|---------------|
| TRIAGE | `kai/agents/triage.py` | Alert classification, severity scoring, SOC L1 automation |
| SENTINEL | `kai/agents/sentinel.py` | Continuous threat monitoring, anomaly detection |
| KEEPER | `kai/agents/keeper.py` | Knowledge base maintenance, runbook updates |
| COMM | `kai/agents/comm.py` | Incident communication, PSA ticket creation, Vapi voice alerts |
| FORESIGHT | `kai/agents/foresight.py` | Predictive analysis, trend forecasting |
| HOUSE | `kai/agents/house.py` | Infrastructure health, resource optimization |
| BILL | `kai/agents/bill.py` | Billing analysis, cost optimization, invoice review |
| ANALYST | `kai/agents/analyst.py` | Deep-dive security analysis, forensic investigation |
| HUNTER | `kai/agents/hunter.py` | Proactive threat hunting, IOC correlation |
| INVEST | `kai/agents/invest.py` | Incident investigation, root cause analysis |
| SIMULATE | `kai/agents/simulate.py` | Attack simulation, adversary emulation planning |
| RISK | `kai/agents/risk.py` | Risk quantification, FAIR framework analysis |
| DEPLOY | `kai/agents/deploy.py` | Deployment orchestration, rollout validation |

### LLM chain
Ollama (local) -> vLLM -> OpenAI -> Anthropic (fallback)

### PSA Integration (kai/psa/)
- **ERPNext REST client**: Complete ITSM solution with customer portal, issue tracking, billing, asset management, contracts
- **QBR PDF reports**: reportlab rendering (exec summary, risk trends, metrics), MinIO upload
- **n8n workflows**: Multi-channel support (email-to-ticket, Slack/Teams notifications, escalations)

### Temporal Workflows
- **Billing**: ClickHouse usage aggregation -> Stripe invoice creation -> audit record
- **Remediation**: host isolation -> CVE patching -> service restart

---

## Detection & Threat Intelligence (Layer 3)

### Sigma Engine
Native YAML evaluator (no external crate). Loads all `*.yml` / `*.yaml` from `vendor/sigma/rules/`.
- **Field modifiers**: `|contains`, `|startswith`, `|endswith`, `|re`, `|contains|all`
- **Boolean conditions**: `selection`, `not`, `and`, `or`, `1 of X*`
- **Field mapping**: Image -> executable, CommandLine -> cmdline, User -> user

### YARA-X Engine
Pure-Rust YARA v4 (yara-x 0.12). Loads all `*.yar` / `*.yara` from `vendor/yara-rules/`.
Single Compiler pass handles 25,000+ rules.

### TI Feed Pipeline (Python)
7 feed pullers with configurable cadences:

| Feed | Cadence | Source |
|------|---------|--------|
| NVD | 1h | NIST NVD API |
| CISA KEV | 24h | CISA Known Exploited Vulnerabilities |
| EPSS | 24h | FIRST EPSS CSV (via polars) |
| OTX | 15m | AlienVault OTX (paginated) |
| AbuseIPDB | 30m | AbuseIPDB API |
| IPsum | 1h | IPsum blocklist aggregator |
| MISP | 1h | MISP REST + PyMISP fallback |

All feeds write to ClickHouse via clickhouse-connect.

### RITA Beacon Analysis
GPL-3.0 boundary maintained via:
- Standalone Go module (`services/netguard/rita/go.mod`)
- HTTP-only communication with RITA sidecar (port 4096)
- Docker Compose sidecar: activecm/rita:v5.0 + MongoDB 7
- `make check-gpl-boundary` CI target verifies zero activecm/rita imports

---

## Infrastructure (Layer 4)

### Kubernetes (Kustomize + Helm)
- Base manifests for all 6 services
- Dev overlay: 1 replica, tag `dev`
- Prod overlay: k-svc 3 replicas, others 2 replicas

### GitOps (ArgoCD)
App-of-apps pattern: 6 Helm applications + 1 Kustomize application.
Automated sync + prune + selfHeal.

### TLS (cert-manager)
Let's Encrypt staging + production ClusterIssuers.
Certificates: app/api/grafana/n8n.kubric.security

### Secrets (External Secrets Operator)
Vault KV-v2 -> K8s Secrets. 6 ExternalSecrets for JWT, Stripe, DB URLs, TI API keys.

### SSO (Authentik)
OAuth2/OIDC provider. 4 groups: admin, analyst, agent, readonly.
Maps to JWT groups claim consumed by Go RBAC middleware.

### Observability (Grafana)
- 2 dashboards: platform health + per-tenant customer view
- 6 alert rules: NATS lag, ClickHouse throughput, Vault seal, heartbeat, Temporal, pod crash
- Datasources: ClickHouse, Prometheus, Loki

### Backup & DR
4 backup targets (ClickHouse, Vault, Postgres, Neo4j) -> MinIO S3.
Restore drill test: backup -> list -> restore to temp -> verify row count.

---

## Data Flow

```
Endpoint/Network
      |
  [Agent] -- OCSF events --> [NATS JetStream]
      |                            |
      |                    +-------+-------+
      |                    |               |
      |             [ClickHouse]    [KAI Personas]
      |              (analytics)    (AI triage)
      |                    |               |
      |                    v               v
      |             [Grafana]     [PSA/Zammad]
      |             [Alerts]      [Tickets]
      |                    |               |
      +--------------------+-------+-------+
                                   |
                            [Next.js Portal]
                            (customer view)

                                   |
                            [ERPNext ITSM]
                            (tickets/billing)
```

### OCSF Event Classes

| Class UID | Name | Source Agent |
|-----------|------|-------------|
| 4007 | ProcessEvent | CoreSec |
| 4010 | FimEvent | CoreSec |
| 4001 | NetworkActivity | NetGuard |
| 4004 | HostMetric | PerfTrace |

### NATS Subject Schema
```
kubric.{tenant_id}.endpoint.process.v1    -- CoreSec process events
kubric.{tenant_id}.detection.sigma.v1     -- Sigma rule matches
kubric.{tenant_id}.detection.yara.v1      -- YARA rule matches
kubric.{tenant_id}.network.flow.v1        -- NetGuard network flows
kubric.{tenant_id}.perf.host.v1           -- PerfTrace host metrics
kubric.{tenant_id}.watchdog.status.v1     -- Watchdog update status
kubric.provisioning.register              -- Agent registration requests
kubric.{tenant_id}.tenant.{created|updated|deleted}
kubric.{tenant_id}.vuln.{created|updated}
kubric.{tenant_id}.compliance.{created|updated}
```

---

## License Boundaries

| Component | License | Boundary Mechanism |
|-----------|---------|-------------------|
| nDPI | LGPL-3.0 | Runtime dlopen (zero compile-time link) |
| RITA | GPL-3.0 | HTTP-only communication (standalone Go module) |
| Sigma rules | DRL-1.1 | Used as data files only (no code import) |
| YARA rules | BSD/Apache | Used as data files only |
| Nuclei templates | MIT | Subprocess execution (no Go import) |
| Falco rules | Apache 2.0 | Config files only |

---

## Test Coverage

| Component | Tests | Framework |
|-----------|-------|-----------|
| CoreSec | 29 | `cargo test -p coresec` |
| NetGuard | 29 | `cargo test -p netguard` |
| PerfTrace | 0 | (metrics are data structs) |
| Provisioning | 12 | `cargo test -p provisioning` |
| Watchdog | 11 | `cargo test -p watchdog` |
| Go APIs | per-service | `go test ./...` |
| KAI PSA | 4 | `pytest kai/tests/` |
| Billing | 2 | `go test ./services/k-svc/billing/...` |
| **Total Rust** | **81** | `cargo test --workspace` |

---

## Build & Verify

```bash
# Rust workspace
cargo check --workspace
cargo test --workspace

# Go services
go test -mod=mod ./...

# RITA client (isolated module)
cd services/netguard/rita && go build ./...

# GPL boundary check
make check-gpl-boundary

# Python KAI
pip install -e "kai[all]"
pytest kai/tests/

# Frontend
cd frontend && npm install && npm run build
```
