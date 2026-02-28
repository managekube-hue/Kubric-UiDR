# Kubric ITIL 4 Compliance Matrix

> **Consolidated from:** `10_K-ITIL-10_ITIL_MATRIX/` (2 subdirectories, 11 files)
> **Source:** `archive/spec-modules/10_K-ITIL-10_ITIL_MATRIX/`

---

## Overview

Kubric maps 9 ITIL 4 practices across 3 practice categories to platform modules, KAI AI agents, and NATS event streams. Every practice has defined RACI matrices, REST endpoints, NATS subjects, and KPIs.

---

## Audit Readiness

### KIC Evidence Map (KIC-001 → KIC-011)

| Control | Description | Agent/Service | Frameworks | Storage |
|---------|-------------|---------------|------------|---------|
| KIC-001 | Asset Inventory | Watchdog | SOC 2 CC6.1, ISO A.8.1, CIS 1 | ClickHouse |
| KIC-002 | Vulnerability Management | VDR + Nuclei | SOC 2 CC7.1, ISO A.12.6, CIS 7 | ClickHouse + PostgreSQL |
| KIC-003 | Change Control | ArgoCD + KAI Deploy | SOC 2 CC8.1, ISO A.12.1.2, CIS 4 | Git + ClickHouse |
| KIC-004 | Access Control | Authentik + Vault | SOC 2 CC6.1-6.3, ISO A.9, CIS 5-6 | PostgreSQL |
| KIC-005 | Incident Response | KAI Triage→Keeper | SOC 2 CC7.3-7.4, ISO A.16, CIS 17 | ClickHouse + TheHive |
| KIC-006 | Log Integrity | CoreSec FIM + BLAKE3 | SOC 2 CC7.2, ISO A.12.4, CIS 8 | ClickHouse |
| KIC-007 | Encryption | Vault PKI + TLS | SOC 2 CC6.7, ISO A.10, CIS 3 | Vault |
| KIC-008 | Backup & Recovery | MinIO + ClickHouse | SOC 2 A1.2, ISO A.12.3, CIS 11 | MinIO |
| KIC-009 | Network Security | NetGuard + Suricata | SOC 2 CC6.6, ISO A.13, CIS 12-13 | ClickHouse |
| KIC-010 | Endpoint Protection | CoreSec + PerfTrace | SOC 2 CC6.8, ISO A.12.2, CIS 10 | ClickHouse |
| KIC-011 | Compliance Reporting | KIC + OSCAL | SOC 2 CC4.1, ISO A.18, CIS 18 | PostgreSQL + ClickHouse |

Evidence retention: **13 months rolling** across all controls.

### SOC 2 / ISO 27001 / ITIL 4 Crosswalk

Structural mapping between SOC 2 Trust Service Criteria, ISO 27001:2022 Annex A controls, KIC controls (KIC-001 through KIC-011), and ITIL 4 practices. Implemented as a C# crosswalk registry with LINQ query helpers (`BySoc2Criterion`, `ByIso27001Control`, `ByItil4Practice`) and JSON export.

---

## ITIL 4 Practice Map

### General Management Practices

#### GMP1 — Strategy Management
- **Modules:** KAI Risk (pyFAIR Monte Carlo), KAI Sentinel (fleet health), KAI Simulate (LTV predictor), PerfTrace, SSVC, LSTM
- **Endpoints:** `/api/v1/strategy/*`
- **NATS:** `kubric.strategy.*`
- **KPIs:** Mean Risk Score < 0.35 | Fleet Availability ≥ 99.5% | EPSS p90 Patch SLA ≥ 95%

#### GMP5 — Risk Management
- **Pipeline:** CoreSec/Sigma/YARA → Identification → KAI Risk pyFAIR/EPSS/SSVC → Analysis/Quantification → KAI Keeper → Treatment → KAI Sentinel → Monitoring → KAI Foresight LSTM → Validation
- **Endpoints:** `/api/v1/risk/*`
- **NATS:** `kubric.risk.*`
- **KPIs:** Risk Register Coverage ≥ 95% | Quantified Risks ≥ 80% | Treatment SLA ≥ 90%

#### GMP6 — Information Security Management
- **CIA Triad Mapping:**
  - **Confidentiality:** Authentik RBAC, Vault secrets, encryption at rest
  - **Integrity:** CoreSec FIM + BLAKE3 hashing
  - **Availability:** Watchdog health monitoring, auto-restart
- **Threat Detection:** eBPF (CoreSec), IDS (NetGuard/Suricata), YARA, Sigma, Candle ML anomaly
- **Endpoints:** `/api/v1/ism/*`
- **NATS:** `kubric.ism.*`
- **KPIs:** MTTD < 5 min | FP Rate < 5% | Encryption Coverage 100%

### Service Management Practices

#### SMP1 — Incident Management
- **Lifecycle:** Detection (CoreSec, NetGuard) → Triage (KAI Triage — llama3, OCSF normalize, KISS scoring) → Investigation (KAI Analyst — Cortex chain) → Forensic Capture (memory snapshot) → Escalation (VAPI/Twilio) → Resolution (KAI Keeper)
- **Endpoints:** `/api/v1/incidents/*`
- **NATS:** `kubric.incident.*`
- **KPIs:** MTTD < 5 min | MTTT < 10 min | MTTR P1 < 30 min

#### SMP3 — Problem Management
- **Three Modes:**
  - **Reactive:** KAI Invest graph investigation (neo4j topology traversal)
  - **Proactive:** LSTM Foresight anomaly detection (predict before impact)
  - **Error Control:** Sigma known-error records, root cause tagging
- **Hunting:** KAI Hunter runs Velociraptor + Sigma hunting campaigns
- **KPIs:** Backlog Age 0 > 30d | Recurring Incident Rate < 3% | Proactive Detection ≥ 20%

#### SMP7 — Service Level Management
- **SLI/SLO/SLA:**
  - Fleet Availability ≥ 99.5%
  - MTTD p95 < 5 min
  - API p99 < 200 ms
  - P1 MTTR < 4 hours
- **Modules:** KAI Sentinel (health score, churn risk, HIBP credential score), KAI Bill (HLE calculation, invoice generation)

#### SMP10 — Change Enablement
- **Change Types:**
  - **Standard:** ArgoCD GitOps (auto-approved)
  - **Normal:** SaltStack fleet rollout with CAB approval
  - **Emergency:** KAI Deploy with Criticality 5 guardrail (blocks auto-deploy)
- **KPIs:** Change Success Rate ≥ 98% | Change Failure Rate < 2%

#### SMP12 — Deployment Management
- **Pipeline:** GitOps-first (ArgoCD) + SaltStack fleet agent deploys
- **Environments:** Dev → Staging → PreProd → Prod
- **Canary Gate:** 5% hosts, 30-min soak period
- **Agent Specs:** CoreSec, NetGuard, PerfTrace, Watchdog, KAI API
- **KPIs:** Deployment Success Rate ≥ 99% | Time-to-Prod < 45 min

### Technical Management Practices

#### TMP2 — Infrastructure & Platform Management
- **Observability Stack:**
  - **PerfTrace:** sysinfo + perf_event_open + Prometheus + OTEL
  - **Watchdog:** process health, restart orchestration
  - **CoreSec:** eBPF map pressure monitoring
  - **NetGuard:** AF_PACKET/DPDK capture statistics
- **Data Platform:** ClickHouse (events), PostgreSQL (state), FastAPI, asyncpg, aiokafka, Socket.IO
- **KPIs:** CPU p95 < 70% | Memory p95 < 80% | API p99 < 200 ms

---

## RACI Summary

| Practice | KAI Agent (Responsible) | Platform Module (Accountable) | NOC Dashboard (Consulted) | SOC Team (Informed) |
|----------|-------------------------|-------------------------------|---------------------------|---------------------|
| Strategy | KAI Risk, KAI Simulate | K-SVC | Grafana | Slack/Email |
| Risk | KAI Risk, KAI Foresight | VDR, KIC | Grafana | TheHive alerts |
| InfoSec | KAI Triage | CoreSec, NetGuard | Grafana | VAPI/Twilio |
| Incident | KAI Triage → KAI Keeper | NOC | Grafana | VAPI/Twilio, Slack |
| Problem | KAI Invest, KAI Hunter | NOC | Grafana | TheHive |
| Service Level | KAI Sentinel, KAI Bill | K-SVC | Grafana | n8n workflows |
| Change | KAI Deploy | ArgoCD, SaltStack | Grafana | Git notifications |
| Deployment | KAI Deploy, KAI House | ArgoCD, Docker | Grafana | Slack |
| Infrastructure | PerfTrace, Watchdog | Prometheus, Grafana | Grafana | Alertmanager |

---

*Full ITIL practice specs preserved in `archive/spec-modules/10_K-ITIL-10_ITIL_MATRIX/`*
