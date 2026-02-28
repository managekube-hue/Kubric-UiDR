# K-MAP-014 — Unified Endpoint Management (UEM)
**Discipline:** Unified Endpoint Management
**Abbreviation:** UEM
**Kubric Reference:** K-MAP-014
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Unified Endpoint Management (UEM) consolidates the management of diverse endpoint types — servers, workstations, containers, and edge devices — into a single operational workflow covering inventory, configuration, patching, policy enforcement, and health monitoring. UEM bridges the gap between IT operations and security by ensuring that endpoints are consistently configured, up-to-date, and continuously monitored. Kubric implements UEM through the CoreSec agent (runtime monitoring and FIM), the KAI Deploy agent (fleet patching via SaltStack), the KAI Housekeeper agent (configuration convergence via Ansible), and the Watchdog agent (fleet health and liveness monitoring).

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Endpoint Inventory | Watchdog agent orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| Fleet Health Monitoring | Watchdog heartbeat processing | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| Endpoint Runtime Monitoring | CoreSec main entry point | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs` |
| Software / Process Inventory | CoreSec sysinfo process poll | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs` |
| Endpoint Configuration Mgmt | KAI Housekeeper Ansible runner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py` |
| Endpoint Patching | KAI Deploy SaltStack client | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py` |
| Fleet Patch Rollout | KAI Deploy fleet rollout | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py` |
| Endpoint Performance | PerfTrace host metrics | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs` |
| Endpoint File Integrity | CoreSec FIM | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/` |
| Endpoint Rollback | KAI Housekeeper rollback | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py` |
| Policy Guardrails | KAI Guardrails criticality 5 | `03_K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py` |
| Endpoint Change Audit | KAI Bill ClickHouse audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |

---

## 3. Data Flow Diagram

```
Endpoint Fleet
│
├── CoreSec agents: runtime telemetry every 5 s
│   └── NATS: kubric.{tenant}.endpoint.process.v1
│              kubric.{tenant}.endpoint.fim.v1
│
├── PerfTrace agents: performance metrics every 15 s
│   └── NATS: kubric.{tenant}.metrics.host.v1
│   └── Prometheus: :9090/metrics
│
└── All agents: heartbeat every 10 s
    └── NATS: kubric.{tenant}.agent.heartbeat

                    ▼
          Watchdog Agent
          ├── Fleet inventory (registered agents map)
          ├── Health classification: healthy / stale / offline
          └── NATS: kubric.{tenant}.agent.status.v1 (every 15 s)

                    ▼
          KAI UEM Actions
          ├── Deploy agent: SaltStack patch rollout
          ├── Housekeeper: Ansible config convergence
          │    ├── Detect drift from baseline
          │    └── Idempotent apply
          └── Rollback: revert failed patches

ClickHouse: kubric.agent_status_history (fleet inventory)
ClickHouse: kubric.change_audit (patch history)
Grafana: fleet health heatmap, version drift chart
```

---

## 4. UEM KPIs

| KPI | Target | Source |
|---|---|---|
| Fleet Coverage (agents deployed) | 100% of managed hosts | Watchdog inventory |
| Agent Availability | ≥ 99.5%/month | Watchdog `agent.status.v1` |
| Hosts on Latest Agent Version | ≥ 95% | ClickHouse change_audit |
| Mean Patch Time (Critical CVE) | < 24h from SSVC Immediate | KAI Deploy + Change audit |
| Configuration Compliance | ≥ 98% | KAI Housekeeper Ansible idempotency |
| Stale Agents (>30 min no heartbeat) | 0 | Watchdog stale threshold |

---

## 5. Integration Points

| System | UEM Role |
|---|---|
| **SaltStack** | Imperative fleet management; package install/update |
| **Ansible** | Idempotent host configuration State |
| **Watchdog** | Fleet health and liveness monitoring |
| **ClickHouse** | Endpoint change history and compliance evidence |
| **Grafana** | Fleet health, version dashboards |
| **Vault** | Deployment credentials and SSH keys |
| **TheHive** | Endpoint compliance failure cases |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| SaltStack master | Deployed and accessible from KAI namespace; all endpoints as minions |
| Ansible inventory | Dynamic or static inventory covering all managed hosts |
| Watchdog | Configured with `STALE_THRESHOLD=30s`, `OFFLINE_THRESHOLD=120s` |
| ClickHouse | `kubric.agent_status_history` and `kubric.change_audit` tables |
| Agent packages | CoreSec, NetGuard, PerfTrace binaries available in SaltStack fileserver |
