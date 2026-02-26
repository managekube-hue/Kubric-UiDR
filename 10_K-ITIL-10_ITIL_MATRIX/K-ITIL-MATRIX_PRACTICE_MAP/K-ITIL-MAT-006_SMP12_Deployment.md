# K-ITIL-MAT-006 — ITIL 4 SMP12: Deployment Management
**Practice Code:** SMP12
**Practice Family:** Service Management Practices
**Kubric Reference:** K-ITIL-MAT-006
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Deployment Management moves new or changed hardware, software, documentation, processes, or any other service component to live environments. It is closely linked to Change Enablement (SMP10) but focuses specifically on the technical mechanics of moving components through pipeline stages. Kubric implements deployment management through a GitOps-first model using ArgoCD, SaltStack for fleet agent deployments, and the KAI Deploy agent for orchestrated multi-host rollouts, with automated health gates preventing broken deployments from reaching production.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Deployment Pipeline (infrastructure) | ArgoCD GitOps sync | `services/k-svc/` manifests |
| Agent Fleet Deployment | KAI Deploy – SaltStack client | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py` |
| Multi-Host Fleet Rollout | KAI Deploy – fleet rollout | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py` |
| Deployment Orchestration | KAI Deploy Agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-001_deploy_agent.py` |
| Post-Deploy Verification | Watchdog – agent orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| Certificate Deployment | cert-manager + ESO | Kubernetes infra layer |
| Secret Injection | HashiCorp Vault + ESO | Kubernetes infra layer |
| Deployment Rollback | KAI Housekeeper – rollback | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py` |
| Pre-Deploy Config Validation | KAI Housekeeper – criticality check | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py` |

---

## 3. Deployment Environments

| Environment | GitOps Branch | Promotion Gate | Auto-Deploy |
|---|---|---|---|
| **Development** | `dev` | Unit + integration tests (cargo test) | Yes |
| **Staging** | `staging` | Integration tests + Watchdog health check | Yes |
| **Pre-Production** | `preprod` | Performance test + Criticality check | Manual approval |
| **Production** | `main` | Full CAB for normal changes; Criticality 5 guardrail | Manual approval (Normal/Emergency) |

---

## 4. RACI Matrix

| Activity | Platform Eng | Release Manager | SOC Lead | KAI Agents |
|---|---|---|---|---|
| Build and package agent binaries | **A** | C | I | I |
| Commit to GitOps repo | **A** | R | I | I |
| ArgoCD sync (standard deploy) | I | C | I | **R** |
| SaltStack fleet rollout | **A** | C | I | **R** |
| Post-deploy health verification | C | C | I | **R** |
| Approve prod deployment | C | **A** | C | I |
| Monitor deployment metrics | C | C | C | **R** |
| Initiate rollback on failure | **A** | C | I | **R** |

---

## 5. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.agent.heartbeat` | Subscribe | Post-deploy heartbeats prove agent liveness |
| `kubric.{tenant}.agent.status.v1` | Subscribe | Watchdog status: healthy/stale/offline per agent |
| `kubric.{tenant}.change.deploy.v1` | Publish | Deployment event: agent_type, version, host, timestamp |
| `kubric.{tenant}.change.rollback.v1` | Publish | Rollback event: trigger_reason, target_hosts |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/deploy/agents` | GET | Current agent versions across the fleet |
| `/api/v1/deploy/rollout` | POST | Initiate SaltStack fleet rollout |
| `/api/v1/deploy/status/{job_id}` | GET | SaltStack job status |
| `/api/v1/deploy/health` | GET | Post-deploy health summary from Watchdog |
| `/api/v1/deploy/rollback` | POST | Rollback to previous known-good version |
| `/api/v1/deploy/history` | GET | Deployment history from ClickHouse audit |

---

## 6. Deployment Pipeline

```
Developer commits code
        |
        v
Git Pull Request → CI (cargo test / pytest)
        |
        v
Merge to branch → ArgoCD detects diff
        |
   ┌────┴──────────────────────────────────────────┐
   | Dev/Staging: auto-sync                        |
   | Prod: manual sync after CAB approval          |
   └────────────────────────────────────────────────┘
        |
        v
KAI Deploy Agent (K-KAI-DEP-001)
   ├── Criticality Check (K-KAI-HS-003)
   ├── SaltStack fleet rollout (K-KAI-DEP-002)
   │     └── Target: all minions with role=coresec|netguard|perftrace
   └── Canary gate: 5% hosts first, 30-min soak
        |
        v
Watchdog monitors heartbeats
  kubric.{tenant}.agent.heartbeat
        |
   ┌────┴──────────────────────────────────────────┐
   | All agents healthy within 5 min →             |
   | Complete rollout, publish deploy.v1            |
   |                                               |
   | Any agent stale/offline →                     |
   | KAI Housekeeper rollback (K-KAI-HS-004)        |
   | Publish change.rollback.v1                    |
   └───────────────────────────────────────────────┘
        |
        v
ClickHouse audit (K-KAI-BL-002_clickhouse_audit.py)
Grafana deployment dashboard
```

---

## 7. Agent Deployment Specifications

| Agent | Binary | Deploy Method | Health Check |
|---|---|---|---|
| **CoreSec** | `coresec` (Rust) | SaltStack state | NATS heartbeat on `agent.heartbeat` |
| **NetGuard** | `netguard` (Rust) | SaltStack state | NATS heartbeat on `agent.heartbeat` |
| **PerfTrace** | `perftrace` (Rust) | SaltStack state | HTTP `/healthz` on `:9090` |
| **Watchdog** | `watchdog` (Rust) | SaltStack state | NATS `agent.status.v1` self-report |
| **KAI API** | Python/FastAPI | ArgoCD / Kubernetes | HTTP `/healthz` |

---

## 8. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| Deployment Success Rate | ≥ 99% | ClickHouse deploy audit |
| Canary Gate Pass Rate | ≥ 95% without human intervention | KAI Deploy fleet rollout |
| Time to Production (standard change) | < 45 min | Git merge → Watchdog healthy |
| Rollback Frequency | < 1% of deployments | Housekeeper rollback events |
| Post-Deploy Health Confirmation Time | < 5 min | Watchdog sweep interval × 2 |
| Fleet Version Drift | 0 hosts on N-2 or older | SaltStack grains report |

---

## 9. Integration Points

| System | Deployment Management Role |
|---|---|
| **ArgoCD** | GitOps reconciliation engine; manages Kubernetes manifest state |
| **SaltStack** | Imperative fleet management; binary distribution and service restart |
| **Vault** | Runtime secret injection; no credentials baked into images |
| **cert-manager** | TLS certificate provisioning during deployment |
| **ESO** | Kubernetes secret sync from Vault for pod deployments |
| **Watchdog** | Post-deploy liveness verification gate |
| **ClickHouse** | Deployment audit trail and rollback history |
| **Grafana** | Deployment frequency, failure rate, and fleet version dashboards |

---

## 10. References

- ITIL 4 Foundation: *Deployment Management* (SMP12)
- ArgoCD: https://argo-cd.readthedocs.io
- SaltStack: https://docs.saltproject.io
- HashiCorp Vault: https://developer.hashicorp.com/vault
- Kubric Deploy agents: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/`
