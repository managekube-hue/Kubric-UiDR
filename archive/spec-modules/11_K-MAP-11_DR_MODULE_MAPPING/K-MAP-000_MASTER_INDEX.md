# K-MAP-000 — Master Index: Kubric Detection & Response Module Mapping
**Document Code:** K-MAP-000
**Series:** K-MAP-11_DR_MODULE_MAPPING
**Last Updated:** 2026-02-26

---

## 1. Purpose

This master index provides a comprehensive overview of how the Kubric Unified Infrastructure Detection and Response (UiDR) platform maps to every major Detection & Response (DR) discipline. Each numbered K-MAP document provides depth on one discipline, including the Kubric modules responsible, data flows, MITRE ATT&CK coverage, and deployment prerequisites.

---

## 2. Platform Architecture Summary

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         KUBRIC UiDR PLATFORM                               │
│                                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐  │
│  │  AGENT LAYER (Rust — K-XRO Super Agent)                            │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │  │
│  │  │ CoreSec  │  │ NetGuard │  │PerfTrace │  │    Watchdog      │   │  │
│  │  │ eBPF FIM │  │ PCAP IDS │  │ Metrics  │  │ Fleet Orchestr.  │   │  │
│  │  │ YARA ML  │  │ DPI TLS  │  │ OTEL     │  │ Health Monitor   │   │  │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘   │  │
│  └───────┼─────────────┼─────────────┼──────────────────┼─────────────┘  │
│          │             │             │                  │                  │
│          └─────────────┴─────────────┴──────────────────┘                 │
│                                  │                                         │
│                         NATS JetStream                                     │
│                   kubric.{tenant}.*  subjects                              │
│                                  │                                         │
│  ┌───────────────────────────────┼─────────────────────────────────────┐  │
│  │  KAI ORCHESTRATION LAYER (Python — FastAPI + CrewAI)               │  │
│  │                               │                                    │  │
│  │  Triage  Analyst  Hunter  Invest  Keeper  Risk  Foresight          │  │
│  │  Sentinel  Comm  House  Deploy  Bill  Simulate                     │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
│                                  │                                         │
│  ┌───────────────────────────────┼─────────────────────────────────────┐  │
│  │  DATA & INTEGRATION LAYER                                          │  │
│  │  ClickHouse  PostgreSQL  NATS  Kafka  Prometheus  OTEL             │  │
│  │  TheHive  MISP  Cortex  Wazuh  Vault  Authentik  ArgoCD           │  │
│  └─────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. DR Discipline Index

| K-MAP Doc | Discipline | Abbreviation | Primary Kubric Agents | Status |
|---|---|---|---|---|
| K-MAP-001 | Endpoint Detection & Response | EDR | CoreSec, KAI Triage | Complete |
| K-MAP-002 | Identity Threat Detection & Response | ITDR | Authentik, KAI Invest, KAI Triage | Complete |
| K-MAP-003 | Network Detection & Response | NDR | NetGuard, KAI Analyst | Complete |
| K-MAP-004 | Cloud Detection & Response | CDR | KAI Deploy, ArgoCD, KAI Risk | Complete |
| K-MAP-005 | SaaS Detection & Response | SDR | KAI Sentinel, KAI Invest | Complete |
| K-MAP-006 | Application Detection & Response | ADR | NetGuard DPI, KAI Triage | Complete |
| K-MAP-007 | Data Detection & Response | DDR | CoreSec FIM, ClickHouse, KAI Keeper | Complete |
| K-MAP-008 | Vulnerability Detection & Response | VDR | KAI Risk, KAI Foresight | Complete |
| K-MAP-009 | Managed Detection & Response | MDR | Full KAI stack, Watchdog | Complete |
| K-MAP-010 | Threat Intelligence | TI | KAI Invest, MISP, KAI Foresight | Complete |
| K-MAP-011 | Configuration Drift Detection & Response | CFDR | KAI House, ArgoCD | Complete |
| K-MAP-012 | Backup & Disaster Recovery | BDR | KAI Keeper, Vault, Watchdog | Complete |
| K-MAP-013 | Network Performance Management | NPM | NetGuard, PerfTrace | Complete |
| K-MAP-014 | Unified Endpoint Management | UEM | CoreSec, KAI Deploy, SaltStack | Complete |
| K-MAP-015 | Mobile Device Management | MDM | KAI Deploy, Sentinel | Complete |
| K-MAP-016 | Application Performance Management | APM | PerfTrace, OTEL | Complete |
| K-MAP-017 | Governance, Risk & Compliance | GRC | KAI Risk, KAI Bill | Complete |
| K-MAP-018 | KAI AI Orchestration Layer | KAI | Full KAI CrewAI stack | Complete |
| K-MAP-019 | Professional Services Automation | PSA | KAI Bill, Sentinel, Simulate | Complete |
| K-MAP-020 | License & Compliance Management | LCM | KAI Bill, ClickHouse | Complete |

---

## 4. Core NATS Subject Taxonomy

| Subject Pattern | Publisher | Consumer |
|---|---|---|
| `kubric.{tenant}.endpoint.process.v1` | CoreSec | KAI Triage, KAI Analyst |
| `kubric.{tenant}.endpoint.fim.v1` | CoreSec | KAI Triage, KAI Keeper |
| `kubric.{tenant}.endpoint.malware.v1` | CoreSec | KAI Triage |
| `kubric.{tenant}.network.flow.v1` | NetGuard | KAI Analyst, ClickHouse |
| `kubric.{tenant}.network.ids.alert.v1` | NetGuard | KAI Triage |
| `kubric.{tenant}.network.tls.anomaly.v1` | NetGuard | KAI Triage |
| `kubric.{tenant}.metrics.host.v1` | PerfTrace | Prometheus, ClickHouse |
| `kubric.{tenant}.agent.heartbeat` | All agents | Watchdog |
| `kubric.{tenant}.agent.status.v1` | Watchdog | KAI Sentinel, Grafana |
| `kubric.{tenant}.incident.triage.v1` | KAI Triage | TheHive, KAI Keeper |
| `kubric.{tenant}.risk.score.v1` | KAI Risk | Grafana, KAI Sentinel |

---

## 5. MITRE ATT&CK Coverage Summary

| Tactic | Kubric Modules with Coverage |
|---|---|
| Initial Access (TA0001) | NetGuard IDS, TLS SNI, IP threat intel |
| Execution (TA0002) | CoreSec eBPF execve, Sigma evaluator, YARA |
| Persistence (TA0003) | CoreSec FIM (inotify + BLAKE3), Sigma |
| Privilege Escalation (TA0004) | CoreSec eBPF openat2, Sigma rules |
| Defense Evasion (TA0005) | CoreSec YARA, ML anomaly (Candle), eBPF |
| Credential Access (TA0006) | KAI Sentinel HIBP, KAI Invest MISP |
| Discovery (TA0007) | NetGuard flow anomaly, CoreSec Sigma |
| Lateral Movement (TA0008) | NetGuard flow analysis, DPI, KAI Invest graph |
| Collection (TA0009) | CoreSec FIM, NetGuard TLS SNI |
| Command and Control (TA0011) | NetGuard IP threat intel, DPI L7 classifier |
| Exfiltration (TA0010) | NetGuard flow volume anomaly, DPI |
| Impact (TA0040) | CoreSec YARA (ransomware), FIM volume |

---

## 6. Deployment Quick Reference

| Component | Min Spec | Notes |
|---|---|---|
| CoreSec agent | Linux x86_64, kernel ≥ 5.4 | eBPF requires CAP_BPF; sysinfo fallback available |
| NetGuard agent | Linux x86_64, libpcap | DPDK optional for ≥ 10 Gbps environments |
| PerfTrace agent | Any OS with sysinfo | perf_event_open requires Linux ≥ 3.10 |
| Watchdog agent | Any OS | Heartbeat interval: 10 s; status sweep: 15 s |
| KAI stack | Kubernetes ≥ 1.27 | Requires NATS JetStream, ClickHouse, PostgreSQL |
| NATS JetStream | 3-node cluster | Enables durable, ordered event delivery |
| ClickHouse | 3-node cluster | TTL policies per event type; columnar compression |

---

## 7. Navigation

See individual K-MAP-NNN documents in this directory for full discipline mappings. Each document follows the standard structure:

1. Discipline Definition
2. Kubric Modules (with file paths)
3. Data Flow Diagram (ASCII)
4. MITRE ATT&CK Coverage
5. Integration Points
6. Deployment Prerequisites
