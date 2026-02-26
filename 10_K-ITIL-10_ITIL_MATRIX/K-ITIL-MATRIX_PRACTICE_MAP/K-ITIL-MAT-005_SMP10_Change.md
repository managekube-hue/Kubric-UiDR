# K-ITIL-MAT-005 — ITIL 4 SMP10: Change Enablement
**Practice Code:** SMP10
**Practice Family:** Service Management Practices
**Kubric Reference:** K-ITIL-MAT-005
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Change Enablement maximizes the number of successful service and product changes by ensuring risks are properly assessed and changes are authorized. ITIL 4 distinguishes between standard, normal, and emergency changes. Kubric operationalizes change enablement through GitOps (ArgoCD), SaltStack fleet rollouts, Ansible configuration management, and AI-assisted change impact assessment via the KAI Housekeeper and Deploy agents, with the Criticality Check guardrail preventing deployment to high-availability workloads without approval.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Change Request / Standard Change | ArgoCD GitOps pipeline | `services/k-svc/` + ArgoCD application manifests |
| Normal Change – Impact Assessment | KAI Housekeeper – criticality check | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py` |
| Normal Change – Deployment | KAI Deploy Agent – fleet rollout | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py` |
| Normal Change – Configuration | KAI Housekeeper – Ansible runner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py` |
| Emergency Change – Agent Patch | KAI Deploy – SaltStack client | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py` |
| Change Rollback | KAI Housekeeper – rollback | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py` |
| Change Scheduling (Standard) | KAI Deploy Agent – deploy agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-001_deploy_agent.py` |
| Change Audit Trail | ClickHouse – KAI Bill audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |
| Guardrails | KAI Guardrails – criticality 5 | `03_K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py` |

---

## 3. Change Types in Kubric

| Change Type | Authorization | Toolchain | Approval Gate |
|---|---|---|---|
| **Standard** | Pre-approved; auto-deploy via ArgoCD | GitOps → ArgoCD | Git PR merge = approval |
| **Normal** | Change Advisory Board (CAB) review | KAI Deploy + SaltStack/Ansible | Criticality check + human sign-off |
| **Emergency** | CISO or on-call incident commander | KAI Deploy → SaltStack | Criticality 5 guardrail check only |
| **Agent Hotfix** | Platform Eng lead | SaltStack client | Watchdog health check post-deploy |

---

## 4. RACI Matrix

| Activity | Change Manager | Platform Eng | CISO | KAI Agents |
|---|---|---|---|---|
| Raise change request | C | **A** | I | R |
| Assess impact (criticality) | **A** | R | C | **R** |
| CAB review for normal changes | **A** | R | C | I |
| Schedule and execute deployment | C | **A** | I | **R** |
| Run post-deploy health checks | C | **R** | I | **R** |
| Authorize emergency change | C | C | **A** | R |
| Execute rollback | C | **A** | I | **R** |
| Record change audit entry | I | C | I | **R** |

---

## 5. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.agent.heartbeat` | Subscribe | Watchdog monitors agent liveness post-deployment |
| `kubric.{tenant}.agent.status.v1` | Subscribe | Post-deploy health confirmation |
| `kubric.{tenant}.change.deploy.v1` | Publish | Deployment event record: version, agent_type, host |
| `kubric.{tenant}.change.rollback.v1` | Publish | Rollback event: trigger, agent_type, previous_version |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/change/requests` | GET | List pending change requests |
| `/api/v1/change/deploy` | POST | Trigger fleet rollout; body: `{agent_type, version, target_hosts}` |
| `/api/v1/change/rollback` | POST | Initiate rollback; body: `{agent_type, target_hosts, reason}` |
| `/api/v1/change/criticality-check` | POST | Run criticality assessment; body: `{service, change_type}` |
| `/api/v1/change/audit` | GET | ClickHouse-backed change audit log |
| `/api/v1/change/status/{job_id}` | GET | SaltStack job status for a deployment run |

---

## 6. Change Pipeline Flow

```
Engineer raises PR (Git)
        |
        v
ArgoCD detects diff → sync
  (Standard change: auto-apply if health checks pass)
        |
        v
KAI Deploy Agent (K-KAI-DEP-001)
   └── Criticality Check (K-KAI-HS-003)
         |
   ┌─────┴───────────────────────────────────┐
   | Criticality < 5                         |
   | → SaltStack fleet rollout               |
   | → Ansible config convergence            |
   |                                         |
   | Criticality = 5 (production critical)   |
   | → Guardrail BLOCKS auto-deploy          |
   | → Requires manual CAB approval          |
   └─────────────────────────────────────────┘
        |
        v
Post-deploy health check
   Watchdog: kubric.{tenant}.agent.status.v1
        |
   ┌────┴──────────────────────────────────┐
   | All agents healthy →                  |
   | ClickHouse audit record               |
   |                                       |
   | Agent stale/offline →                 |
   | KAI Housekeeper rollback              |
   └───────────────────────────────────────┘
```

---

## 7. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| Change Success Rate | ≥ 98% | ClickHouse change audit |
| Change Failure Rate (rollback triggered) | < 2% | KAI Housekeeper rollback events |
| Emergency Change Volume | < 5% of total changes | Change audit log |
| Mean Lead Time for Standard Change | < 30 min | ArgoCD sync → health-check completion |
| Mean Lead Time for Normal Change | < 2 business days | CAB approval → deploy complete |
| Post-Deploy Watchdog Confirmation | 100% | Watchdog `agent.status.v1` |
| Criticality 5 Guardrail Blocks | Logged, reviewed | KAI Guardrails `criticality_5.py` |

---

## 8. Integration Points

| System | Change Enablement Role |
|---|---|
| **ArgoCD** | GitOps change automation; pull-based continuous delivery |
| **SaltStack** | Imperative fleet configuration and emergency patching |
| **Ansible** | Idempotent configuration management for infrastructure changes |
| **Vault** | Secrets injected per-deployment at runtime; no secrets in Git |
| **cert-manager** | Certificate rotation triggered as standard changes |
| **ESO** | Secret sync between Vault and Kubernetes; part of every deploy |
| **Watchdog** | Post-change health verification gate |
| **ClickHouse** | Immutable change audit log |
| **Grafana** | Change frequency and failure-rate dashboards |

---

## 9. References

- ITIL 4 Foundation: *Change Enablement* (SMP10)
- ArgoCD GitOps: https://argo-cd.readthedocs.io
- SaltStack: https://docs.saltproject.io
- Ansible: https://docs.ansible.com
- Kubric Deploy: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/`
- Kubric Guardrails: `03_K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/`
