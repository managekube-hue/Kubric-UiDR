# K-MAP-011 — Configuration Drift Detection & Response (CFDR)
**Discipline:** Configuration Drift Detection & Response
**Abbreviation:** CFDR
**Kubric Reference:** K-MAP-011
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Configuration Drift Detection and Response (CFDR) identifies and remediates unauthorized or unintended changes to system configurations, infrastructure state, and security baselines. Configuration drift is a persistent source of vulnerabilities: a single misconfigured file, unexpected package installation, or manual Kubernetes resource edit can introduce significant risk. Kubric implements CFDR through ArgoCD (GitOps enforcement of Kubernetes desired state), CoreSec FIM (real-time file change detection with BLAKE3 baselines), the KAI Housekeeper agent (Ansible-based configuration convergence), and the KAI Criticality 5 guardrail (blocking unauthorized changes to critical resources).

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Kubernetes Drift Detection | ArgoCD sync & health check | `services/k-svc/` manifests |
| File System Drift Detection | CoreSec FIM inotify | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| File Hash Baseline | CoreSec FIM BLAKE3 | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs` |
| Config File Detection (Sigma) | CoreSec Sigma evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| Config Convergence | KAI Housekeeper Ansible runner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py` |
| Criticality Assessment | KAI Housekeeper criticality check | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py` |
| Config Rollback | KAI Housekeeper rollback | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py` |
| Deploy Guardrail | KAI Guardrails criticality 5 | `03_K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py` |
| Change Audit | KAI Bill ClickHouse audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |
| Secret Drift | Vault + ESO | HashiCorp Vault audit log |

---

## 3. Data Flow Diagram

```
Configuration State Sources
│
├── Kubernetes Manifests (Git)
│   └── ArgoCD: desired state = Git HEAD
│        ├── Sync: out-of-sync resource → drift alert
│        ├── Health: degraded resource → health alert
│        └── NATS: kubric.{tenant}.change.deploy.v1
│
├── Host File System (CoreSec FIM)
│   ├── inotify: /etc, /usr/bin, /etc/systemd, /etc/cron.d
│   ├── BLAKE3: hash mismatch = unauthorized change
│   └── NATS: kubric.{tenant}.endpoint.fim.v1
│
├── Application Config (CoreSec Sigma)
│   └── Sigma rules: unexpected config file write patterns
│
└── Secret State (Vault)
    └── ESO sync: Vault → K8s Secret
         └── Drift: K8s secret != Vault value → ESO re-sync

                    ▼
          Configuration Drift Event
                    │
          ┌─────────┴─────────────────────────────────────┐
          | Automated Response:                          |
          | KAI Housekeeper                              |
          |  ├── Criticality check (< 5):               |
          |  │    → Ansible runner: converge to baseline |
          |  └── Criticality = 5:                        |
          |       → Guardrail blocks; human approval     |
          |                                              |
          | If rollback needed:                          |
          | KAI Housekeeper rollback → ArgoCD hard sync  |
          └─────────────────────────────────────────────── |
                    │
          NATS: kubric.{tenant}.change.deploy.v1
          ClickHouse: kubric.change_audit
          TheHive: drift incident case
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric CFDR Response |
|---|---|---|---|
| Defense Evasion | T1562.001 | Disable or Modify Tools | FIM: agent binary modification |
| Defense Evasion | T1070.006 | Timestomping | FIM: mtime vs. content hash mismatch |
| Persistence | T1543.003 | Windows Service / systemd | FIM: /etc/systemd/system modification |
| Persistence | T1053.003 | Cron Job | FIM: /etc/cron.d modification |
| Privilege Escalation | T1548.003 | sudo and sudoedit | FIM: /etc/sudoers modification |
| Initial Access | T1078 | Valid Accounts (stale credential) | Vault audit: stale secret not rotated |
| Impact | T1490 | Inhibit System Recovery | FIM: backup config deletion |
| Defense Evasion | T1578 | Modify Cloud Infrastructure | ArgoCD drift detection |

---

## 5. Integration Points

| System | CFDR Role |
|---|---|
| **ArgoCD** | GitOps desired-state enforcement; primary Kubernetes drift detection |
| **Ansible** | Host configuration convergence via KAI Housekeeper |
| **Vault** | Secret state authority; ESO drift re-sync |
| **TheHive** | Unauthorized configuration change cases |
| **ClickHouse** | Configuration change audit trail |
| **Grafana** | ArgoCD sync status dashboard; FIM change rate charts |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| ArgoCD | Installed and managing all production Kubernetes manifests |
| CoreSec FIM paths | `/etc`, `/usr/bin`, `/usr/sbin`, `/etc/systemd`, `/etc/cron.d` at minimum |
| Ansible inventory | All managed hosts registered; KAI Housekeeper has SSH/API access |
| Vault + ESO | All Kubernetes secrets managed by ESO from Vault |
| ClickHouse | `kubric.change_audit` table with ArgoCD sync event schema |
| Criticality 5 config | `KUBRIC_CRITICALITY_THRESHOLD=5` in KAI Housekeeper environment |
