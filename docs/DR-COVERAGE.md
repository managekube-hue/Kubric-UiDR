# Kubric DR Module Coverage Map

> **Consolidated from:** `11_K-MAP-11_DR_MODULE_MAPPING/` (21 files, K-MAP-000 through K-MAP-020)
> **Source:** `archive/spec-modules/11_K-MAP-11_DR_MODULE_MAPPING/`

---

## Overview

Kubric implements **20 Detection & Response disciplines** across a unified agent-bus-AI architecture:

```
Agent Layer (Rust)  →  NATS JetStream  →  KAI Orchestration (Python/Go)  →  Data & Integration Layer
CoreSec | NetGuard     kubric.events.*     13 CrewAI persona agents         ClickHouse | PostgreSQL
PerfTrace | Watchdog   kubric.metrics.*    Temporal durable workflows       Neo4j | MinIO | Qdrant
```

---

## DR Discipline Matrix

| # | Discipline | Primary Agents / Modules | MITRE Coverage | Status |
|---|-----------|--------------------------|----------------|--------|
| 001 | **EDR** — Endpoint | CoreSec (eBPF, BLAKE3 FIM, Sigma, YARA, Candle ML) | 14 techniques | Complete |
| 002 | **ITDR** — Identity | Authentik OIDC, KAI Sentinel HIBP, KAI Invest graph | 12 techniques | Complete |
| 003 | **NDR** — Network | NetGuard (libpcap/AF_PACKET/DPDK, nDPI, IDS, ipsum) | 14 techniques | Complete |
| 004 | **CDR** — Cloud | K8s-native CoreSec/NetGuard, ArgoCD drift, Vault hygiene | 11 techniques | Complete |
| 005 | **SDR** — SaaS | Authentik SSO, KAI Sentinel, KAI Invest, MISP SaaS TI | 7 techniques | Complete |
| 006 | **ADR** — Application | NetGuard L7 DPI (SQLi/XSS/RCE), CoreSec, PerfTrace, KAI Guardrails | 9 techniques | Complete |
| 007 | **DDR** — Data | CoreSec FIM (BLAKE3), YARA ransomware, NetGuard exfil, ClickHouse audit | 11 techniques | Complete |
| 008 | **VDR** — Vulnerability | KAI Risk (pyFAIR, EPSS v3, SSVC), Nuclei, MISP, KAI Keeper | 5 techniques | Complete |
| 009 | **MDR** — Managed | Full platform as managed service (4 tiers: Essential→Enterprise) | All | Complete |
| 010 | **TI** — Threat Intel | ipsum, MISP, EPSS/LSTM, Cortex enrichment, YARA/Sigma/Nuclei | 8 feed types | Complete |
| 011 | **CFDR** — Config Drift | ArgoCD GitOps, CoreSec FIM baselines, KAI Housekeeper Ansible | 8 techniques | Complete |
| 012 | **BDR** — Backup/DR | YARA+FIM ransomware → isolation → backup restore → ArgoCD sync | Component-level RPO/RTO | Complete |
| 013 | **NPM** — Network Perf | NetGuard capture + PerfTrace NIC stats | 7 Prometheus metrics | Complete |
| 014 | **UEM** — Endpoint Mgmt | Watchdog, CoreSec, KAI Deploy (SaltStack), KAI Housekeeper | Fleet-level | Complete |
| 015 | **MDM** — Mobile | NetGuard network-side + Authentik conditional access + HIBP | Network-side only | Complete |
| 016 | **APM** — App Perf | PerfTrace (sysinfo, perf_event_open, Prometheus, OTEL), FastAPI, Socket.IO | 10 metrics | Complete |
| 017 | **GRC** — Governance | SOC 2, ISO 27001, ITIL 4, MITRE, CIS v8, NIST 800-53, FAIR, SSVC | Full framework set | Complete |
| 018 | **KAI** — AI Layer | 13 CrewAI personas, 3 crew groups, Guardrails framework | Orchestration | Complete |
| 019 | **PSA** — Business | KAI Bill (Stripe), KAI Sentinel (health/churn), KAI Simulate (LTV), KAI Comm | MSSP ops | Complete |
| 020 | **LCM** — License | SBOM (cargo-sbom, syft, cyclonedx), 21 components, regulatory compliance | Compliance | Complete |

---

## EDR — Endpoint Detection & Response (K-MAP-001)

**Agent:** CoreSec (Rust, eBPF)

| Module | Path | Capability |
|--------|------|------------|
| eBPF hooks | `hoco/coresec/src/ebpf/` | Process creation, file access, network connect syscalls |
| BLAKE3 FIM | `hoco/coresec/src/fim.rs` | File integrity monitoring with cryptographic hashing |
| Sigma Engine | `hoco/coresec/src/sigma.rs` | Hot-reload YAML rule evaluation |
| YARA Scanner | `hoco/coresec/src/yara.rs` | Pattern matching on files/memory |
| Candle ML | `hoco/coresec/src/ml/` | Anomaly detection (inference only) |
| Governor | `hoco/coresec/src/governor.rs` | Rate control, CPU budget enforcement |

**Data Flow:** Kernel eBPF → User-space processing → OCSF normalization → NATS `kubric.events.endpoint.*` → KAI Triage

**MITRE Coverage:** T1059 (Command Interpreter), T1053 (Scheduled Task), T1055 (Process Injection), T1003 (Credential Dumping), T1547 (Boot Persistence), T1068 (Privilege Escalation), +8 more

**Prerequisites:** Linux kernel ≥ 5.4, CAP_BPF, CAP_PERFMON

---

## NDR — Network Detection & Response (K-MAP-003)

**Agent:** NetGuard (Rust)

| Module | Path | Capability |
|--------|------|------------|
| Capture | `hoco/netguard/src/capture/` | libpcap / AF_PACKET / DPDK packet capture |
| Flow Analysis | `hoco/netguard/src/flow/` | TCP/UDP flow tracking, session reassembly |
| TLS Inspector | `hoco/netguard/src/tls/` | SNI extraction, JA3/JA4 fingerprinting |
| nDPI DPI | `hoco/netguard/src/dpi/` | Deep packet inspection, L7 classification |
| IDS Engine | `hoco/netguard/src/ids.rs` | Suricata rule evaluation |
| ipsum Lookup | `hoco/netguard/src/ipsum/` | Known-bad IP blocklist matching |

**Data Flow:** NIC → Capture → Flow/DPI → IDS rules → OCSF → NATS `kubric.events.network.*`

**Prerequisites:** Promiscuous NIC, CAP_NET_RAW, libndpi.so

---

## VDR — Vulnerability Detection & Response (K-MAP-008)

**Pipeline:** Nuclei scan → EPSS v3 enrichment → pyFAIR Monte Carlo → SSVC decision tree → KAI Keeper remediation

| SSVC Outcome | Action | SLA |
|--------------|--------|-----|
| Immediate | Auto-patch or isolate | < 24 hours |
| Out-of-Cycle | Emergency change window | < 72 hours |
| Scheduled | Next maintenance window | < 30 days |
| Defer | Monitor only | Risk accepted |

---

## KAI AI Orchestration Layer (K-MAP-018)

### CrewAI Persona Agents (13)

| Persona | Role | Crew Group |
|---------|------|------------|
| **KAI Triage** | Alert classification, OCSF normalization, KISS scoring | SOC Operations |
| **KAI Analyst** | Deep investigation, Cortex enrichment chain | SOC Operations |
| **KAI Hunter** | Proactive threat hunting, Velociraptor/Sigma campaigns | SOC Operations |
| **KAI Invest** | Graph-based investigation, Neo4j topology traversal | SOC Operations |
| **KAI Keeper** | Automated remediation, approval gates | SOC Operations |
| **KAI Comm** | Stakeholder communication, VAPI/Twilio escalation | SOC Operations |
| **KAI Risk** | pyFAIR quantification, EPSS/SSVC scoring | Risk & Compliance |
| **KAI Foresight** | LSTM time-series prediction, anomaly forecasting | Risk & Compliance |
| **KAI Sentinel** | Fleet health scoring, churn risk, HIBP monitoring | Risk & Compliance |
| **KAI Bill** | HLE billing, invoice generation, Stripe integration | Risk & Compliance |
| **KAI House** | Infrastructure convergence, Ansible playbooks | Platform Operations |
| **KAI Deploy** | SaltStack patching, canary deployment, Criticality 5 guardrail | Platform Operations |
| **KAI Simulate** | LTV prediction, churn simulation, dynamic pricing | Platform Operations |

---

## BDR — Backup & Disaster Recovery (K-MAP-012)

### RTO/RPO Targets

| Component | RPO | RTO | Backup Method |
|-----------|-----|-----|---------------|
| KAI API State | 5 min | 15 min | PostgreSQL WAL streaming |
| PostgreSQL | 1 hour | 2 hours | pg_basebackup + WAL |
| ClickHouse | 4 hours | 4 hours | Partition backup to MinIO |
| Neo4j | 4 hours | 4 hours | neo4j-admin dump |
| MinIO | 0 (replicated) | 15 min | Erasure coding (4+2) |
| Vault | 1 hour | 30 min | Raft snapshot |
| NATS | 0 (JetStream) | 5 min | Stream replication |

### DR Response Workflow
Detection (YARA+FIM) → Memory Snapshot → Network Isolation → VAPI/SMS Alert → Backup Restore → ArgoCD Hard Sync → Ansible Convergence → Watchdog Verification

---

## GRC Framework Coverage (K-MAP-017)

| Framework | Coverage | Evidence Source |
|-----------|----------|-----------------|
| SOC 2 Type II | Full TSC (CC1-CC9, A1, C1, PI1) | KIC-001 through KIC-011 |
| ISO 27001:2022 | Annex A (93 controls via crosswalk) | Crosswalk registry |
| ITIL 4 | 9 practices mapped | Practice map |
| MITRE ATT&CK | Enterprise matrix (14 tactics) | CoreSec + NetGuard |
| CIS Controls v8 | 18 control groups | Agent telemetry |
| NIST 800-53 Rev 5 | 1,189 controls (OSCAL) | OSCAL catalog mapping |
| FAIR | Risk quantification | pyFAIR Monte Carlo |
| SSVC | Decision trees | VDR pipeline |

---

## License Compliance (K-MAP-020)

**SBOM Pipeline:** `cargo-sbom` (Rust) + `syft` (containers) + `cyclonedx-bom` (Python) → CycloneDX JSON → per-release artifact

**Regulatory:** SOC 2 (CC2.2 software inventory), ISO 27001 (A.8.1.1 asset inventory), GDPR (Art. 30 records), HIPAA (§164.312 access controls), PCI DSS (Req 6 secure development), NIS2 (Art. 21 supply chain)

**KPIs:** 0 GPL/AGPL components in proprietary code paths | SBOM generated per release | License audit quarterly

---

## NATS Subject Taxonomy

```
kubric.events.endpoint.*      # CoreSec EDR events
kubric.events.network.*       # NetGuard NDR events
kubric.events.identity.*      # ITDR events
kubric.events.cloud.*         # CDR events
kubric.events.application.*   # ADR events
kubric.events.data.*          # DDR events
kubric.metrics.*              # PerfTrace, Watchdog
kubric.risk.*                 # KAI Risk, EPSS, SSVC
kubric.incident.*             # Incident lifecycle
kubric.strategy.*             # Strategy management
kubric.ism.*                  # InfoSec management
kubric.change.*               # Change enablement
kubric.deploy.*               # Deployment management
kubric.billing.*              # KAI Bill, invoices
```

---

*Full DR module mapping specs preserved in `archive/spec-modules/11_K-MAP-11_DR_MODULE_MAPPING/`*
