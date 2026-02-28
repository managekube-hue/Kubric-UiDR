# K-MAP-009 — Managed Detection & Response (MDR)
**Discipline:** Managed Detection & Response
**Abbreviation:** MDR
**Kubric Reference:** K-MAP-009
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Managed Detection and Response (MDR) delivers security monitoring and incident response as a service, typically combining technology with human expertise. MDR providers operate on behalf of customers, providing 24/7 threat monitoring, alert triage, investigation, and incident response. Kubric is designed to function as the technology backbone for an MDR service delivery model: the full KAI agent persona stack (Triage, Analyst, Hunter, Invest, Keeper, Risk, Foresight, Sentinel, Comm) automates the routine work of an MDR SOC, while the Watchdog agent provides the fleet availability guarantees that underpin MDR service-level commitments.

---

## 2. Kubric Modules

| MDR Function | Module | File Path |
|---|---|---|
| 24/7 Detection | CoreSec (all sub-modules) | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/` |
| 24/7 Detection | NetGuard (all sub-modules) | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/` |
| Alert Triage | KAI Triage + llama3 + OCSF + KISS | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/` |
| Investigation | KAI Analyst (Cortex, observables) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/` |
| Threat Hunting | KAI Hunter (Velociraptor, Sigma) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/` |
| TI Enrichment | KAI Invest (MISP, graph) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/` |
| Containment & Remediation | KAI Keeper (planner, Cortex, Vault) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/` |
| Risk Scoring | KAI Risk (pyFAIR, EPSS, SSVC) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/` |
| Threat Forecasting | KAI Foresight (LSTM, EPSS, Hikari) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/` |
| Health Monitoring | KAI Sentinel (health, churn, HIBP) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/` |
| Escalation | KAI Comm (VAPI, Twilio) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/` |
| Fleet Health | Watchdog orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| SLA Evidence | KAI Bill (ClickHouse, HLE, invoice) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/` |
| Platform Metrics | PerfTrace (sysinfo, perf, OTEL) | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/` |

---

## 3. MDR Service Tiers

| Tier | Coverage | KAI Personas Active | SLA MTTR |
|---|---|---|---|
| **Essential** | EDR + FIM + Watchdog | Triage, Sentinel | P1: 4h, P2: 8h |
| **Standard** | EDR + NDR + VDR + TI | Triage, Analyst, Risk, Invest, Keeper | P1: 2h, P2: 4h |
| **Advanced** | Full platform including hunting | All personas | P1: 1h, P2: 2h |
| **Enterprise** | Advanced + MDR-as-a-Service custom | All + custom crew | Negotiated SLA |

---

## 4. Data Flow Diagram

```
Customer Environment
│
├── CoreSec agents (EDR)            ┐
├── NetGuard agents (NDR)           │  NATS JetStream
├── PerfTrace agents (metrics)      ├──────────────────────►
└── Watchdog (fleet health)         ┘
                                    │
                       ┌────────────┴──────────────────────┐
                       │        KAI MDR Ops Center         │
                       │                                   │
                       │  T1: KAI Triage                   │
                       │   ├── Auto-triage all alerts      │
                       │   ├── llama3 reasoning            │
                       │   └── KISS priority + TheHive     │
                       │                                   │
                       │  T2: KAI Analyst + Invest         │
                       │   ├── Cortex enrichment chain     │
                       │   ├── MISP galaxy correlation     │
                       │   └── Graph investigation         │
                       │                                   │
                       │  T3: KAI Hunter                   │
                       │   ├── Velociraptor artifacts       │
                       │   └── Sigma hunting campaigns     │
                       │                                   │
                       │  Response: KAI Keeper             │
                       │   ├── Remediation planner         │
                       │   ├── Cortex contain/isolate      │
                       │   └── Vault-fetched credentials   │
                       │                                   │
                       │  Escalation: KAI Comm             │
                       │   ├── VAPI phone (P1)             │
                       │   └── Twilio SMS                  │
                       └───────────────────────────────────┘
                                    │
                       Customer Portal (FastAPI + Socket.IO)
                       SLA Reporting (KAI Bill)
                       Grafana dashboards
```

---

## 5. MDR SLA Metrics

| KPI | Essential | Standard | Advanced |
|---|---|---|---|
| MTTD (Mean Time to Detect) | < 15 min | < 10 min | < 5 min |
| MTTT (Mean Time to Triage) | < 30 min | < 15 min | < 10 min |
| MTTR P1 | < 4h | < 2h | < 1h |
| Agent Fleet Availability | ≥ 99% | ≥ 99.5% | ≥ 99.9% |
| Monthly Health Report | Yes | Yes | Yes |
| Threat Hunting (proactive) | No | Quarterly | Monthly |

---

## 6. Integration Points

| System | MDR Role |
|---|---|
| **TheHive** | MDR case management platform; all tiers |
| **Cortex** | Analyst response automation; all tiers |
| **MISP** | Threat intelligence enrichment; Standard and above |
| **Velociraptor** | Forensic collection; Advanced tier |
| **VAPI** | P1 voice escalation; Standard and above |
| **Twilio** | SMS notification; all tiers |
| **ClickHouse** | MDR evidence and SLA audit store |
| **Grafana** | Customer-facing security posture dashboard |
| **KAI Bill** | Monthly SLA and invoice generation per customer |

---

## 7. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| All K-XRO agents | CoreSec, NetGuard, PerfTrace, Watchdog deployed on all endpoints |
| NATS JetStream | 3-node cluster for MDR reliability |
| TheHive | Production instance with customer isolation (tenant separation) |
| Cortex | Analyzers and responders configured per MDR tier |
| Velociraptor | Server installed; agents enrolled for Advanced tier |
| VAPI | Phone number provisioned and MDR on-call schedule configured |
| Authentik | Customer portal RBAC; read-only customer role defined |
