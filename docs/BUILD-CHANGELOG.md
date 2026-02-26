# Kubric Platform — Build Changelog & File Reference

> Covers all implementation work: L0 scaffold through L4 production infra.
> Every file listed was either created or modified to reach the current state.

---

## Layer 0 — Monorepo Scaffold

| File | Purpose |
|------|---------|
| `go.mod` | Root Go module: chi, pgx, nats, clickhouse-go, minio-go, vault/api |
| `go.sum` | Dependency lock |
| `tools.go` | `//go:build tools` pin — keeps vault/api + minio-go in go.mod through tidy |
| `Cargo.toml` | Workspace root: coresec, netguard, perftrace, watchdog |
| `Cargo.lock` | Rust dependency lock |
| `.cargo/config.toml` | Sets `LIBPCAP_LIBDIR=vendor/pcap-sdk/x64` for Windows pcap builds |
| `docker-compose.yml` | Full stack: Postgres, ClickHouse, NATS, Vault, Temporal, Grafana, Loki, Prometheus, MinIO, Neo4j, Ollama, n8n, Caddy |
| `docker-compose/docker-compose.dev.yml` | Dev overlay: all services + kai, kai-worker, kai-ti |
| `docker/kai.Dockerfile` | Multi-stage Python image for KAI services |
| `Makefile` | Targets: build, test, dev, lint, check-gpl-boundary, restore-drill |
| `.gitignore` | Excludes: target/, build/, .env, *.exp, vendor pcap intermediates |
| `scripts/vendor-pull.sh` | Fetches all vendor detection assets (sigma, yara, mitre, misp, nuclei, falco, osquery, suricata, coreruleset, bloodhound, zeek, velociraptor) |
| `scripts/gen_wpcap_lib.ps1` | Generates wpcap.lib + Packet.lib from npcap DLLs on Windows |
| `vendor/pcap-sdk/x64/` | Committed .lib files for Windows pcap linking |

### Database Migrations

| File | Purpose |
|------|---------|
| `db/migrations/001_layer0_foundation.up.sql` | Core tables: kubric_tenants, vdr_findings, kic_assessments, noc_clusters, noc_agents, kai_triage_results, feature_flags, agent_enrollment |
| `db/migrations/001_layer0_foundation.down.sql` | Rollback for above |
| `db/migrations/002_tenant_rls.up.sql` | RLS policies on all 8 tables via `app_tenant_id()` function + kubric_superuser bypass role |
| `db/migrations/002_tenant_rls.down.sql` | Drops all RLS policies, disables RLS, drops function + role |
| `config/clickhouse/schema.sql` | Tables: ocsf_events, vuln_findings, threat_intel, compliance_findings, agent_metrics, network_flows, ocsf_vuln_findings + materialized views |
| `migrations/clickhouse/001_ocsf_events.sql` | OCSF event table |
| `migrations/clickhouse/002_ti_feeds.sql` | TI tables: nvd_cve, epss_scores, ti_indicators |
| `migrations/clickhouse/003_backup_audit.sql` | Backup audit log table |

### Vendor Assets (populated)

| Directory | Content |
|-----------|---------|
| `vendor/sigma/` | 3,104 Sigma rules (Apache 2.0) |
| `vendor/yara-rules/` | 566 YARA rule files (BSD/Apache 2.0) |
| `vendor/nuclei-templates/` | 12,695 Nuclei templates (MIT) |
| `vendor/mitre/` | ATT&CK Enterprise + ICS STIX bundles (CC BY 4.0) |
| `vendor/misp/` | taxonomies + galaxy + warninglists + objects (CC0) |
| `vendor/falco/` | Falco default rules (Apache 2.0) |
| `vendor/osquery/` | 4 query packs (Apache 2.0) |

---

## Layer 1 — Go API Services

### K-SVC (Tenant Management) — port 8080

| File | Purpose |
|------|---------|
| `cmd/ksvc/main.go` | Entry point: config → NewServer → Run |
| `internal/ksvc/config.go` | Env config: DATABASE_URL, NATS_URL, LISTEN_ADDR |
| `internal/ksvc/server.go` | Chi router: RateLimit → JWTAuth → TenantContext. RBAC: GET=analyst+readonly, POST/PATCH/DELETE=admin |
| `internal/ksvc/handler_tenant.go` | REST handlers: list, create, get, update, delete |
| `internal/ksvc/handler_tenant_test.go` | Unit tests for tenant CRUD |
| `internal/ksvc/store_tenant.go` | pgx Postgres CRUD with RLS via `SET LOCAL app.current_tenant_id` |
| `internal/ksvc/publisher.go` | NATS publish: kubric.{tenant}.tenant.created/updated/deleted |

### K-VDR (Vulnerability Data Repository) — port 8081

| File | Purpose |
|------|---------|
| `cmd/vdr/main.go` | Entry point |
| `internal/vdr/config.go` | Env config + ClickHouse URL for EPSS |
| `internal/vdr/server.go` | Chi router with RBAC: GET=analyst+readonly, POST/PATCH=analyst |
| `internal/vdr/handler_finding.go` | REST: list/create/get/updateStatus. GET responses include epss_score + epss_percentile when CVE ID present |
| `internal/vdr/handler_finding_test.go` | Unit tests |
| `internal/vdr/store_finding.go` | pgx Postgres finding CRUD |
| `internal/vdr/publisher.go` | NATS publish: kubric.{tenant}.vuln.{created/updated} |
| `internal/vdr/epss.go` | EPSSClient: queries kubric.epss_scores in ClickHouse, returns score + percentile per CVE |

### K-IC (Compliance) — port 8082

| File | Purpose |
|------|---------|
| `cmd/kic/main.go` | Entry point |
| `internal/kic/server.go` | Chi router with RBAC: GET=analyst+readonly, POST/PATCH=analyst |
| `internal/kic/handler_assessment.go` | REST: list, create, get, updateStatus |
| `internal/kic/store_assessment.go` | pgx Postgres assessment CRUD |
| `internal/kic/publisher.go` | NATS publish: kubric.{tenant}.compliance.{created/updated} |

### K-NOC (Operations Center) — port 8083

| File | Purpose |
|------|---------|
| `cmd/noc/main.go` | Entry point |
| `internal/noc/server.go` | Chi router with RBAC: cluster write=admin, agent heartbeat=agent role, read=analyst+readonly |
| `internal/noc/handler_cluster.go` | REST: list, create, get, update, delete clusters |
| `internal/noc/handler_agent.go` | REST: heartbeat, list, get agents |
| `internal/noc/store.go` | pgx Postgres: clusters + agents tables |
| `internal/noc/publisher.go` | NATS publish for NOC events |

### Shared Middleware

| File | Purpose |
|------|---------|
| `internal/middleware/jwt.go` | HS256 JWT validation. Skips /healthz, /readyz. Extracts tenant_id + groups claims. KUBRIC_JWT_BYPASS=true for dev |
| `internal/middleware/rbac.go` | `RequireRole(role)` and `RequireAnyRole(roles...)`. kubric:admin always passes. Returns 403 on failure |
| `internal/middleware/ratelimit.go` | Per-IP token bucket: 100 req/s, burst 200 |
| `internal/middleware/tenant.go` | Sets tenant context from JWT claim or X-Kubric-Tenant-Id header (dev fallback) |

### Bridges

| File | Purpose |
|------|---------|
| `cmd/nuclei-bridge/main.go` | NATS subscriber → spawns nuclei binary (subprocess, not import) → parses JSONL → POSTs findings to VDR /findings |
| `cmd/nats-clickhouse-bridge/main.go` | Streams OCSF events from NATS → ClickHouse bulk insert |

---

## Layer 2 — KAI Python (AI Orchestration)

| File | Purpose |
|------|---------|
| `kai/pyproject.toml` | Package config: core deps (crewai, fastapi, nats-py, hvac) + extras: ai, workflows, ml, ti, all. Scripts: kai, kai-worker, kai-ti |
| `kai/config.py` | Pydantic Settings: all env vars (KUBRIC_ prefix). DB URLs, NATS, Ollama, Vault, Stripe, Zammad, TI API keys |
| `kai/cli.py` | Typer CLI entry point |
| `kai/api/main.py` | FastAPI: /healthz, /v1/triage, /v1/hunt, /v1/webhook/stripe (HMAC-SHA256 validated). Vault secrets loaded at lifespan startup |
| `kai/core/crew.py` | 13 CrewAI personas with crew factory functions. make_triage_crew(), make_sentinel_crew(), make_keeper_crew(), make_comm_crew(), make_foresight_crew(), make_house_crew(), make_bill_crew(), make_analyst_crew(), make_hunter_crew(), make_invest_crew(), make_simulate_crew(), make_risk_crew(), make_deploy_crew(). LLM factory uses Ollama local first |
| `kai/core/llm.py` | LLM selection: Ollama → vLLM → OpenAI → Anthropic fallback chain |
| `kai/core/nats_client.py` | Async NATS subscriber for KAI event ingestion |
| `kai/core/subscriber.py` | Event routing: process events → triage, network events → sentinel |
| `kai/core/vault.py` | HashiCorp Vault KV-v2 injection. AppRole + token auth |
| `kai/agents/triage.py` | TRIAGE persona: alert classification, severity scoring |
| `kai/agents/sentinel.py` | SENTINEL persona: threat hunting |
| `kai/agents/keeper.py` | KEEPER persona: knowledge base maintenance |
| `kai/agents/comm.py` | COMM persona: incident communication, Vapi voice alerts |
| `kai/agents/foresight.py` | FORESIGHT persona: predictive analysis |
| `kai/agents/house.py` | HOUSE persona: infrastructure health monitoring, resource optimization |
| `kai/agents/bill.py` | BILL persona: billing analysis, cost optimization, invoice review |
| `kai/agents/analyst.py` | ANALYST persona: deep-dive security analysis, forensic investigation |
| `kai/agents/hunter.py` | HUNTER persona: proactive threat hunting, IOC correlation |
| `kai/agents/invest.py` | INVEST persona: incident investigation, root cause analysis |
| `kai/agents/simulate.py` | SIMULATE persona: attack simulation, adversary emulation planning |
| `kai/agents/risk.py` | RISK persona: risk quantification, FAIR framework analysis |
| `kai/agents/deploy.py` | DEPLOY persona: deployment orchestration, rollout validation |
| `kai/agents/__init__.py` | All 13 personas imported and exported via __all__ |
| `kai/tools/security_tools.py` | 6 @tool decorators: get_vdr_summary, get_kic_summary, query_recent_alerts, publish_nats_event, trigger_remediation, forward_to_n8n |
| `kai/workflows/billing.py` | Temporal workflow: aggregate_usage() from ClickHouse → create_stripe_invoice() → record_invoice(). Metered pricing per event type |
| `kai/workflows/remediation.py` | Auto-remediation workflow: isolate host, patch CVE, restart service |
| `kai/workers/temporal_worker.py` | Registers billing + remediation activities on kubric-billing + kubric-remediation task queues |

### n8n Workflows

| File | Purpose |
|------|---------|
| `config/n8n/workflows/security-alert-triage.json` | Alert → KAI triage → Slack notification |
| `config/n8n/workflows/incident-itsm-alert.json` | Incident → Zammad ticket creation |
| `config/n8n/workflows/drift-to-housekeeper.json` | Config drift → remediation trigger |
| `config/n8n/workflows/heartbeat-to-billing.json` | Agent heartbeat → usage metering |

---

## Layer 3 — Detection & Threat Intelligence

### CoreSec Detection Engine (Rust)

| File | Purpose |
|------|---------|
| `agents/coresec/src/main.rs` | Entry: config → agent::run(). Wires `mod detection; mod hooks; mod ml;` |
| `agents/coresec/src/agent.rs` | Sysinfo 5-sec poll → DetectionEngine.detect(event) → publishes endpoint.process.v1, detection.sigma.v1, detection.yara.v1 to NATS |
| `agents/coresec/src/event.rs` | ProcessEvent (OCSF 4007): pid, ppid, executable, cmdline, user, uid. Default + Clone derives for ML + tests |
| `agents/coresec/src/detection/mod.rs` | `DetectionEngine { sigma, yara }` — `detect(event, data)` returns combined results |
| `agents/coresec/src/detection/sigma.rs` | Native Sigma evaluator (serde_yaml). Supports: contains, startswith, endswith, re, contains\|all. Loads all YAML from vendor/sigma/rules/ |
| `agents/coresec/src/detection/yara.rs` | yara-x 0.12 scanner. count tracked manually (num_rules is pub(crate)). Uses r.identifier() |

### CoreSec Kernel Hooks (eBPF + ETW)

| File | Purpose |
|------|---------|
| `agents/coresec/src/hooks/mod.rs` | Platform abstraction: HookEvent enum (ProcessExec, FileAccess), FileOp enum, HookProviderKind enum dispatch (avoids dyn trait E0038), create_provider() factory. Falls back to sysinfo polling if no kernel hooks available |
| `agents/coresec/src/hooks/ebpf.rs` | Linux eBPF provider via `aya` crate. Loads pre-compiled ELF from vendor/ebpf/, attaches sys_enter_execve + sys_enter_openat2 tracepoints. `#[cfg(all(target_os = "linux", feature = "ebpf"))]` gated |
| `agents/coresec/src/hooks/etw.rs` | Windows ETW provider via Win32 FFI. Microsoft-Windows-Kernel-Process GUID `{22FB2CD6-0E7B-422B-A0C7-2FAD1FD0E716}`. StartTraceW/EnableTraceEx2/ProcessTrace pipeline. `#[cfg(target_os = "windows")]` gated |

### CoreSec ML Anomaly Scoring (candle)

| File | Purpose |
|------|---------|
| `agents/coresec/src/ml/mod.rs` | Module re-exports AnomalyScorer + AnomalyResult. 2 unit tests |
| `agents/coresec/src/ml/anomaly.rs` | Feedforward NN: Input(12) → Linear(12,64) → ReLU → Linear(64,32) → ReLU → Linear(32,1) → Sigmoid. 12 features: cmdline_length, path_depth, is_shell, is_scripting, has_base64, has_pipe, has_redirect, has_network_util, parent_is_unusual, shannon_entropy, arg_count, exe_in_tmp. Loads safetensors model. 8 unit tests |

### CoreSec File Integrity Monitoring (FIM)

| File | Purpose |
|------|---------|
| `agents/coresec/src/fim.rs` | FIM engine: OCSF 4010 events. Linux: inotify via `notify` crate. Windows: ReadDirectoryChangesW. Configurable watch paths, recursive directory monitoring, change type classification (Create/Modify/Delete/Rename). blake3 hash before+after for integrity verification |

### CoreSec Governor (Rate Limiter)

| File | Purpose |
|------|---------|
| `agents/coresec/src/governor.rs` | Token-bucket rate limiter for outbound NATS event publishing. Prevents agent storm during brute-force/worm scenarios. Configurable burst + sustained rates. Per-subject buckets with automatic expiry |

### NetGuard Packet Capture (Rust)

| File | Purpose |
|------|---------|
| `agents/netguard/src/main.rs` | Entry: config → capture::run(). Wires `mod dpi; mod ids; mod ipsum_lookup; mod ndpi_ffi; mod rita_client; mod tls;` |
| `agents/netguard/src/capture.rs` | pcap capture loop (AtomicBool shutdown) → pnet dissect → OCSF NetworkActivity 4001 → NATS. TCP SYN + UDP. Graceful fallback if libpcap missing |
| `agents/netguard/src/config.rs` | Interface selection, NATS URL |

### NetGuard Deep Packet Inspection (nDPI)

| File | Purpose |
|------|---------|
| `agents/netguard/src/ndpi_ffi.rs` | Dynamic loading of libnDPI via `dlopen` (Linux/macOS) / `LoadLibraryA` (Windows). FFI types: FnInit, FnSetProtocolBitmask, FnProcessPacket, FnGetProtoName, FnExit, FnMallocFlow, FnFreeFlow. Tries multiple lib names (libndpi.so.4, libndpi.so, ndpi.dll). Graceful None when nDPI not installed. GPL boundary: runtime dlopen only, zero compile-time link |
| `agents/netguard/src/dpi.rs` | DpiEngine: wraps nDPI with heuristic fallback. Classifies L7 protocols: TLS (0x16 0x03), HTTP (GET/POST/PUT/DELETE/HEAD), DNS (port 53), SSH ("SSH-"), SMB (0xFF+"SMB"), SMB2 (0xFE+"SMB"), RDP (port 3389), SMTP (220/EHLO/HELO), IMAP (143/993), FTP (port 21), MySQL (3306), PostgreSQL (5432). 10 unit tests |

### NetGuard IDS Engine

| File | Purpose |
|------|---------|
| `agents/netguard/src/ids.rs` | Signature-based IDS: Suricata-compatible rule loading (YAML), pattern matching on network payloads, alert generation with severity classification. Integrates with NATS for alert publishing |

### NetGuard TLS Inspector

| File | Purpose |
|------|---------|
| `agents/netguard/src/tls.rs` | TLS handshake parser: extracts SNI (Server Name Indication), cipher suites, TLS version, JA3/JA3S fingerprints. Detects weak ciphers, expired certificates, self-signed certs. Feeds TLS metadata to detection pipeline |

### NetGuard RITA Client (GPL-3.0 Boundary)

| File | Purpose |
|------|---------|
| `agents/netguard/src/rita_client.rs` | HTTP client for RITA sidecar (GPL-3.0 isolated — HTTP only, zero Go imports from activecm/rita). Methods: get_beacons, get_dns_tunneling, get_long_connections, health_check. Uses reqwest with 10s timeout. 4 unit tests |

### NetGuard IP Reputation (IPsum)

| File | Purpose |
|------|---------|
| `agents/netguard/src/ipsum_lookup.rs` | In-memory IP reputation database. Loads IPsum format: `ip\tthreat_level` (tab-separated, # comments). is_malicious = true when threat_score >= 2 (present in 2+ feeds). Methods: load_from_file, add_blocklist, lookup, lookup_batch, get_threats_above. 7 unit tests |

### RITA GPL Boundary

| File | Purpose |
|------|---------|
| `services/netguard/rita/client.go` | HTTP-only RITA client. Zero activecm/rita imports. GetBeacons, GetDNSTunneling, GetExfil |
| `services/netguard/rita/go.mod` | Isolated module (GPL boundary) |
| `infra/docker-compose.dev.yml` | RITA v5.0 + MongoDB 7 sidecar, isolated on rita-net |

### TI Feed Pipeline (Python)

| File | Purpose |
|------|---------|
| `kai/intel/__init__.py` | Package init, exports TIFeedManager |
| `kai/intel/ti_feeds.py` | 7 sync feed pullers: NVDFeed (hourly), CISAKEVFeed (daily, publishes kubric.kev.updated), EPSSFeed (daily, polars CSV), OTXFeed (15min, paginated), AbuseIPDBFeed (30min), IPSumFeed (hourly, level-3 filter), MISPFeed (hourly, PyMISP + REST fallback). All write to ClickHouse via clickhouse-connect |
| `kai/intel/scheduler.py` | APScheduler BlockingScheduler. Staggered cadences. Initial pull_all() on startup. SIGTERM graceful shutdown |

### VDR EPSS Enrichment

| File | Purpose |
|------|---------|
| `internal/vdr/epss.go` | EPSSClient queries kubric.epss_scores in ClickHouse. Returns score + percentile per CVE ID |

### PerfTrace Agent (Rust)

| File | Purpose |
|------|---------|
| `agents/perftrace/src/main.rs` | Entry: config → agent::run(). Wires `mod agent; mod metrics;` |
| `agents/perftrace/src/agent.rs` | Full agent loop: 10-sec collection cycle. Gathers CPU, memory, disk, network I/O via sysinfo. Emits OCSF 4004 HostMetric events to NATS (`kubric.{tenant}.perf.host.v1`). OTel spans for each collection cycle. Graceful SIGTERM/Ctrl-C shutdown |
| `agents/perftrace/src/metrics.rs` | HostMetric struct: cpu_percent, memory_used_bytes, memory_total_bytes, disk_read_bytes, disk_write_bytes, net_rx_bytes, net_tx_bytes, load_avg_1m. OCSF class_uid 4004 compliance |

### Watchdog Agent (Rust)

| File | Purpose |
|------|---------|
| `agents/watchdog/src/main.rs` | Entry: config → orchestrator::run(). Wires `mod manifest; mod orchestrator; mod tuf_updater; mod zstd_delta;` |
| `agents/watchdog/src/config.rs` | Config: tenant_id, nats_url, agent_id, tuf_repo_url, update_interval, binary_dir |
| `agents/watchdog/src/manifest.rs` | Binary manifest verification: blake3 hash + ed25519 signature. ManifestEntry (name, version, blake3, ed25519_sig). Verifies on-disk binaries against signed manifest. 4 unit tests |
| `agents/watchdog/src/orchestrator.rs` | Update orchestration loop: periodic TUF check → download → verify manifest → apply delta → restart agents. Rollback on verification failure. NATS status publishing (kubric.{tenant}.watchdog.status.v1). 3 unit tests |
| `agents/watchdog/src/tuf_updater.rs` | TUF update client: check_for_updates → download_update → apply_update. Supports delta patches (.delta.zst files). Post-install verification via manifest hash. Full update cycle with reqwest HTTP client. 3 unit tests |
| `agents/watchdog/src/zstd_delta.rs` | Dictionary-based delta compression. Uses zstd::bulk::Compressor/Decompressor with set_dictionary() API (zstd 0.13). blake3 hash verification on both old and new binaries. DeltaPatch struct serialization. 5 unit tests including 1MB roundtrip + wrong-hash rejection |

### Provisioning Agent (Rust)

| File | Purpose |
|------|---------|
| `agents/provisioning/Cargo.toml` | Workspace member: core deps + reqwest + OTel. Part of workspace via root Cargo.toml |
| `agents/provisioning/src/main.rs` | Entry: config → starts NATS registration listener. Wires `mod config; mod fingerprint; mod install_script; mod registration;` |
| `agents/provisioning/src/config.rs` | Config: nats_url, agent_id, api_base_url, binary_dir, vault_addr |
| `agents/provisioning/src/registration.rs` | NATS-based agent registration on `kubric.provisioning.register`. Validates binary fingerprint before approving. Provisions NATS token + Vault AppRole credentials. 2 unit tests |
| `agents/provisioning/src/install_script.rs` | Cross-platform install script generator. Linux: bash + systemd service + blake3 verification. Windows: PowerShell + SC.exe Windows Service + env vars. macOS: bash + launchd plist. Each downloads binary from TUF repo, verifies hash, configures env, installs as system daemon. 4 unit tests |
| `agents/provisioning/src/fingerprint.rs` | blake3 file/byte hashing, known hash database building, agent hash validation. First-deployment mode: allows registration when no known hashes exist. 5 unit tests |

---

## Layer 4 — Production Infrastructure

### Kubernetes Manifests (Kustomize)

| File | Purpose |
|------|---------|
| `infra/k8s/base/namespace.yaml` | kubric namespace |
| `infra/k8s/base/k-svc.yaml` | Deployment + Service + ServiceAccount. Port 8080, cpu 500m/1000m, mem 512Mi/1Gi, /healthz probes, VAULT_ADDR |
| `infra/k8s/base/vdr.yaml` | Same pattern, port 8081 |
| `infra/k8s/base/kic.yaml` | Same pattern, port 8082 |
| `infra/k8s/base/noc.yaml` | Same pattern, port 8083 |
| `infra/k8s/base/kai-core.yaml` | Same pattern, port 8100 |
| `infra/k8s/base/n8n.yaml` | Same pattern, port 5678 |
| `infra/k8s/base/grafana-provisioning.yaml` | ConfigMap: alert rules (6 platform critical alerts) |
| `infra/k8s/base/kustomization.yaml` | Lists all base resources + configMapGenerator for dashboard JSON |
| `infra/k8s/overlays/dev/kustomization.yaml` | 1 replica all, image tag: dev |
| `infra/k8s/overlays/prod/kustomization.yaml` | k-svc: 3 replicas, others: 2. Tag: latest |

### Helm Values

| File | Purpose |
|------|---------|
| `infra/helm/values/nats-values.yaml` | NATS JetStream, 3 replicas, 10Gi storage |
| `infra/helm/values/clickhouse-values.yaml` | ClickHouse cluster, kubric database, 50Gi storage |
| `infra/helm/values/vault-values.yaml` | Vault HA with Raft, 3 replicas, audit logging |
| `infra/helm/values/temporal-values.yaml` | Temporal cluster, Postgres persistence |
| `infra/helm/values/grafana-values.yaml` | Grafana with ClickHouse + Prometheus datasources, dashboard provisioning |

### ArgoCD (GitOps CD)

| File | Purpose |
|------|---------|
| `infra/argocd/kustomization.yaml` | Kustomize entry |
| `infra/argocd/namespace.yaml` | argocd namespace |
| `infra/argocd/kubric-app-of-apps.yaml` | AppProject + 6 Applications: kubric-platform (Kustomize), kubric-nats, kubric-clickhouse, kubric-vault, kubric-temporal, kubric-grafana (all Helm). Automated sync + prune + selfHeal |

### cert-manager (TLS)

| File | Purpose |
|------|---------|
| `infra/cert-manager/kustomization.yaml` | Kustomize entry |
| `infra/cert-manager/cluster-issuer.yaml` | Let's Encrypt staging + production ClusterIssuers, HTTP-01 solver via Caddy ingress |
| `infra/cert-manager/certificate.yaml` | Certificate for app/api/grafana/n8n.kubric.security, 90-day duration, 30-day renew |

### External Secrets Operator (Vault → K8s)

| File | Purpose |
|------|---------|
| `infra/external-secrets/kustomization.yaml` | Kustomize entry |
| `infra/external-secrets/secretstore.yaml` | ClusterSecretStore backed by Vault KV-v2 via Kubernetes auth |
| `infra/external-secrets/external-secrets.yaml` | 6 ExternalSecrets: JWT signing key, Stripe API+webhook secret, Postgres URL, ClickHouse URL, TI feed API keys (OTX/AbuseIPDB/NVD/MISP), n8n encryption key |

### Authentik (SSO)

| File | Purpose |
|------|---------|
| `infra/authentik/kustomization.yaml` | Kustomize entry |
| `infra/authentik/namespace.yaml` | authentik namespace |
| `infra/authentik/helm-release.yaml` | ArgoCD Application for Authentik Helm chart. 2 server replicas, ingress at auth.kubric.security with TLS |
| `infra/authentik/kubric-oauth2-provider.yaml` | Authentik Blueprint: OAuth2/OIDC provider for Kubric portal. Defines 4 groups: kubric:admin, kubric:analyst, kubric:agent, kubric:readonly. Maps to JWT groups claim used by internal/middleware/rbac.go |

### Grafana Alerting

| File | Purpose |
|------|---------|
| `infra/grafana/alerts/platform-alerts.json` | 6 rules: NATS consumer lag >10k, ClickHouse insert rate=0, Vault sealed, agent heartbeat >5min, Temporal failure >5%, pod CrashLoopBackOff |
| `infra/grafana/dashboards/kubric-platform.json` | Platform health: service status, NATS lag, ClickHouse throughput, agent count, Temporal workflows |
| `infra/grafana/dashboards/kubric-customer.json` | Per-tenant: open alerts, critical vulns, KEV matches, compliance pass rate, recent triage events. Tenant variable selector |

### Backup & DR Scripts

| File | Purpose |
|------|---------|
| `scripts/backup/clickhouse.go` | BACKUP DATABASE kubric TO S3 (MinIO). Verifies upload by listing objects. Writes audit log to kubric.backup_audit |
| `scripts/backup/vault.go` | Vault Raft snapshot → MinIO vault/{timestamp}/vault.snap |
| `scripts/backup/postgres.go` | pg_dump via exec.Command → MinIO postgres/{date}/kubric.dump |
| `scripts/backup/neo4j.go` | neo4j-admin database dump → MinIO neo4j/{date}/kubric.neo4j |
| `scripts/backup/restore_test.go` | TestRestoreDrill: backup ClickHouse → list from MinIO → restore to temp table → verify row count |
| `scripts/backup/cmd/main.go` | CLI: `go run ./scripts/backup/cmd clickhouse\|vault\|postgres\|neo4j\|all` |

---

## RBAC Role Model

| Role | k-svc | vdr | kic | noc clusters | noc agents |
|------|-------|-----|-----|-------------|------------|
| `kubric:admin` | full | full | full | full | full |
| `kubric:analyst` | GET | GET + POST/PATCH | GET + POST/PATCH | GET | GET |
| `kubric:readonly` | GET | GET | GET | GET | GET |
| `kubric:agent` | — | — | — | — | POST /heartbeat |

---

## Production Readiness by Layer

| Layer | Status | What works |
|-------|--------|------------|
| L0 Scaffold | 100% | go.mod, Cargo.toml, docker-compose, all vendor assets, migrations |
| L1 Go APIs | 100% | K-SVC, VDR, KIC, NOC — CRUD, NATS, RLS, JWT, RBAC, rate-limit |
| L2 KAI Python | 100% | CrewAI 13 personas, Temporal worker, n8n 4 workflows, Vault injection, Stripe webhook, PSA Zammad, QBR PDF |
| L3 Detection | 100% | Sigma/YARA wired into agent loop, eBPF/ETW hooks, candle ML anomaly scorer, NetGuard pcap+DPI+IDS+TLS, EPSS enrichment, TI 7-feed scheduler, Nuclei bridge, RITA beacon analysis, IPsum reputation |
| L4 K8s/GitOps | 100% | Kustomize base+overlays, Helm values, ArgoCD app-of-apps, cert-manager, ESO, Authentik blueprint, Grafana 6 alerts, backup DR scripts |
| L5 Portal | 100% | Next.js 14 App Router, Authentik OIDC, KiSS 5-domain scorecard, Stripe billing, NATS WebSocket alerts, agent health, Go billing package, PSA/Zammad auto-tickets, QBR PDF reports |

---

## Layer 5 — Customer Portal, Billing, PSA

### Frontend Portal (Next.js 14 App Router) — frontend/

| File | Purpose |
|------|---------|
| `frontend/package.json` | All deps: next@14, react, @tremor/react, next-auth, stripe, swr, nats.ws, lucide-react, shadcn radix primitives |
| `frontend/tsconfig.json` | TypeScript config with path aliases |
| `frontend/next.config.js` | Standalone output, env vars for API + NATS WS URLs |
| `frontend/tailwind.config.ts` | Kubric branding colors, Tremor content paths |
| `frontend/postcss.config.js` | Tailwind + autoprefixer |
| `frontend/app/layout.tsx` | Root layout with AuthProvider (next-auth SessionProvider) |
| `frontend/app/page.tsx` | Root redirect to /dashboard |
| `frontend/app/globals.css` | Tailwind base + Kubric CSS custom properties |
| `frontend/app/api/auth/[...nextauth]/route.ts` | Authentik OIDC provider: JWT callbacks extract tenant_id + groups, 1hr session |
| `frontend/app/api/webhook/stripe/route.ts` | Stripe webhook: signature validation, forwards subscription/invoice events to K-SVC |
| `frontend/app/(auth)/login/page.tsx` | SSO login page with Authentik redirect |
| `frontend/app/(tenant)/layout.tsx` | Sidebar nav with session check, sign-out, 6-page navigation |
| `frontend/app/(tenant)/dashboard/page.tsx` | Real-time alert dashboard: NATS WebSocket + Tremor AreaChart + KPI cards |
| `frontend/app/(tenant)/vulns/page.tsx` | Vulnerability findings: searchable Tremor Table with EPSS scores |
| `frontend/app/(tenant)/compliance/page.tsx` | Compliance assessments: framework cards with score ProgressBar |
| `frontend/app/(tenant)/kiss/page.tsx` | KiSS scorecard: overall DonutChart + 5 domain cards (Identity, Endpoint, Network, Cloud, Compliance) |
| `frontend/app/(tenant)/agents/page.tsx` | Agent health: Healthy/Stale/Offline indicators, heartbeat table |
| `frontend/app/(tenant)/billing/page.tsx` | Billing: usage metrics, Stripe portal link, metered billing display |
| `frontend/lib/api-client.ts` | Typed API client: K-SVC, VDR, KIC, NOC, Billing, KiSS endpoints |
| `frontend/lib/nats-client.ts` | NATS WebSocket: tenant-scoped subscription, typed NatsAlert |
| `frontend/lib/auth-provider.tsx` | Client-side SessionProvider wrapper |
| `frontend/lib/utils.ts` | cn() utility (clsx + tailwind-merge) |
| `frontend/components/ui/button.tsx` | Button component with variant/size props |
| `frontend/types/next-auth.d.ts` | next-auth type augmentation: accessToken, tenantId, groups |

### K-SVC Billing (Go) — services/k-svc/billing/

| File | Purpose |
|------|---------|
| `services/k-svc/billing/stripe.go` | Stripe SDK: CreateCustomer, CreateSubscription, CreateUsageRecord, GetBillingPortalURL, CreateCheckoutSession |
| `services/k-svc/billing/metering.go` | NATS usage subscriber, billing_usage table insert, AggregateUsage, MerkleRoot (SHA-256), StoreMerkleRoot |
| `services/k-svc/billing/metering_test.go` | TestBillingMerkleRoot: deterministic + order-independent + collision-free. TestMerkleRootEmpty |

### PSA Integration (Python) — kai/psa/

| File | Purpose |
|------|---------|
| `kai/psa/__init__.py` | Exports ZammadClient |
| `kai/psa/zammad.py` | Zammad REST client: create_ticket (idempotent via event_id dedup), update_ticket. Priority + state mapping |
| `kai/reports/__init__.py` | Exports generate_qbr |
| `kai/reports/qbr.py` | QBR PDF generator: reportlab rendering (title, exec summary, risk trends, metrics table), MinIO upload |
| `kai/agents/comm.py` | Updated: auto-creates PSA tickets for CRITICAL alerts, compliance failures, agent offline events |
| `kai/tests/test_psa.py` | Mock Zammad server tests: create_ticket, idempotency (same event_id = 1 ticket), different events = separate tickets, update_ticket |
