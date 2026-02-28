# K-MAP-018 — KAI AI Orchestration Layer
**Discipline:** AI-Driven Security Orchestration (KAI)
**Abbreviation:** KAI
**Kubric Reference:** K-MAP-018
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

The KAI (Kubric AI) layer is the cognitive orchestration engine of the Kubric platform. It implements a multi-agent AI framework using CrewAI, where specialized "persona" agents — each with a defined role, toolset, and behavioral guardrails — collaborate to automate the complete security operations pipeline: detection triage, investigation, threat intelligence enrichment, risk quantification, remediation planning, customer health scoring, billing, and forecasting. KAI bridges raw telemetry from the K-XRO Rust agents (CoreSec, NetGuard, PerfTrace, Watchdog) with structured, actionable security outcomes delivered through TheHive, Grafana, and direct communication channels.

---

## 2. Kubric KAI Persona Map

| Persona | Role | Key Capabilities | File Path |
|---|---|---|---|
| **Triage** | L1 alert triage | llama3 reasoning, OCSF analysis, KISS priority, TheHive case creation | `K-KAI-TRIAGE/` |
| **Analyst** | L2 investigation | Cortex analyzer chain, observable enrichment | `K-KAI-ANALYST/` |
| **Hunter** | Proactive hunting | Velociraptor artifacts, Sigma hunting runner | `K-KAI-HUNTER/` |
| **Invest** | Threat intelligence | MISP galaxy query, graph investigation | `K-KAI-INVEST/` |
| **Keeper** | Remediation | Remediation planner, Cortex subprocess, Vault secret fetch | `K-KAI-KEEPER/` |
| **Risk** | Risk quantification | pyFAIR Monte Carlo, EPSS scorer, SSVC decision | `K-KAI-RISK/` |
| **Foresight** | Predictive analytics | LSTM baseline, EPSS enrichment, Hikari ML trainer | `K-KAI-FORESIGHT/` |
| **Sentinel** | Customer health | Health score publisher, churn risk model, HIBP credential score | `K-KAI-SENTINEL/` |
| **Comm** | Communication | Comm agent, VAPI phone, Twilio SMS | `K-KAI-COMM/` |
| **House** | Configuration management | Housekeeper, Ansible runner, criticality check, rollback | `K-KAI-HOUSE/` |
| **Deploy** | Deployment | Deploy agent, SaltStack client, fleet rollout | `K-KAI-DEPLOY/` |
| **Bill** | Billing | Billing clerk, ClickHouse audit, HLE calculator, invoice renderer | `K-KAI-BILL/` |
| **Simulate** | Business simulation | LTV predictor, churn simulator, dynamic pricing | `K-KAI-SIMULATE/` |

---

## 3. KAI Architecture Overview

```
External Events (NATS JetStream)
kubric.{tenant}.endpoint.*
kubric.{tenant}.network.*
kubric.{tenant}.metrics.*
kubric.{tenant}.agent.*
        │
        ▼
┌────────────────────────────────────────────────────────────────┐
│  KAI API Layer (FastAPI + Python)                             │
│  ├── REST API: /api/v1/* (Authentik JWT-protected)            │
│  ├── WebSocket: Socket.IO real-time push                     │
│  ├── NATS subscriber: nats-py client                         │
│  ├── Kafka consumer: aiokafka (event streaming)              │
│  ├── asyncpg: PostgreSQL                                     │
│  └── ClickHouse connect: analytics store                     │
└────────────────────────────────────────────────────────────────┘
        │
        ▼
┌────────────────────────────────────────────────────────────────┐
│  CrewAI Persona Execution Engine                              │
│                                                              │
│  Crew: SOC Operations                                        │
│  ├── Triage ──► Analyst ──► Hunter (as needed)              │
│  └── Invest ──► Keeper ──► Comm                             │
│                                                              │
│  Crew: Risk & Compliance                                     │
│  ├── Risk ──► Foresight                                      │
│  └── Sentinel ──► Bill                                      │
│                                                              │
│  Crew: Platform Operations                                   │
│  ├── House ──► Deploy                                        │
│  └── Simulate                                               │
└────────────────────────────────────────────────────────────────┘
        │
        ▼
Outputs:
├── TheHive: cases, tasks, observables
├── NATS: kubric.{tenant}.incident.*
│         kubric.{tenant}.risk.*
│         kubric.{tenant}.sla.*
├── ClickHouse: all audit records
├── Twilio / VAPI: notifications and escalations
└── Grafana: real-time dashboards
```

---

## 4. Guardrails Framework

| Guardrail | File | Purpose |
|---|---|---|
| Criticality 5 | `K-KAI-GD-003_criticality_5.py` | Block automated actions on criticality-5 resources |
| Prompt Injection | `K-KAI-GD-004_prompt_injection.py` | Detect and block prompt injection in LLM inputs |

---

## 5. KAI Data Libraries

| Library | File | Use Case |
|---|---|---|
| Polars DataFrame | `K-KAI-LIBS-001_polars_dataframe.py` | High-performance in-memory analytics |
| PyArrow / Parquet | `K-KAI-LIBS-002_pyarrow_parquet.py` | Columnar data export and pipeline |
| fastparquet | `K-KAI-LIBS-003_fastparquet_io.py` | Parquet I/O for large event datasets |
| orjson | `K-KAI-LIBS-004_orjson_serializer.py` | Fast JSON serialization |
| ujson | `K-KAI-LIBS-005_ujson_fallback.py` | JSON fallback serializer |
| msgpack | `K-KAI-LIBS-006_msgpack_encoder.py` | Binary message encoding |
| dpkt | `K-KAI-LIBS-007_dpkt_pcap_parser.py` | PCAP file parsing |
| scapy | `K-KAI-LIBS-008_scapy_probe.py` | Network probe and test packet crafting |
| pcap capture | `K-KAI-LIBS-009_pcap_capture.py` | Live capture utility |
| GeoIP2 | `K-KAI-LIBS-010_geoip2_resolver.py` | IP geolocation enrichment |

---

## 6. KAI MITRE ATT&CK Contribution

KAI agents do not directly detect attacks — that is the role of the K-XRO agents. However KAI enables:

| ATT&CK Contribution | KAI Persona | Mechanism |
|---|---|---|
| Detection coverage breadth | Triage + Analyst | Enrich every alert with MISP galaxy TTP mapping |
| Detection quality | Triage (llama3) | LLM-assisted reasoning reduces false-positive rate |
| Proactive detection | Hunter + Foresight | Hunting campaigns + LSTM anomaly detection |
| Threat actor attribution | Invest | MISP galaxy → threat actor identification |
| Response automation | Keeper | Cortex responder automation |
| Predictive coverage | Foresight | EPSS + LSTM trending |

---

## 7. Integration Points

| System | KAI Role |
|---|---|
| **NATS** | Primary event bus; all KAI agents subscribe and publish |
| **TheHive** | Case management output of Triage, Analyst, Hunter |
| **MISP** | TI input for Invest and Analyst |
| **Cortex** | Analysis and response actions from Analyst and Keeper |
| **Vault** | Runtime secret fetch by Keeper; all API keys |
| **ClickHouse** | Audit trail for all KAI actions and decisions |
| **Velociraptor** | Forensic collection triggered by Hunter |
| **Twilio / VAPI** | Escalation channels for Comm persona |
| **SaltStack** | Fleet management executed by Deploy persona |
| **Ansible** | Config convergence executed by House persona |

---

## 8. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| Kubernetes | ≥ 1.27; KAI runs as Kubernetes Deployments |
| Python | ≥ 3.11 for all KAI services |
| NATS JetStream | 3-node HA cluster |
| PostgreSQL | ≥ 14 for transactional data |
| ClickHouse | ≥ 23.x for analytics store |
| Vault | KV v2, PKI, and Kubernetes auth method configured |
| Authentik | OIDC provider; JWT middleware enforced on /api/v1/* |
| CrewAI | Installed in KAI Python environment |
| LLM backend | llama3 via Ollama or compatible OpenAI-compatible endpoint |
