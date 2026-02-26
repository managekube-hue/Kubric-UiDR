# K-MAP-012 — Backup & Disaster Recovery (BDR)
**Discipline:** Backup & Disaster Recovery
**Abbreviation:** BDR
**Kubric Reference:** K-MAP-012
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Backup and Disaster Recovery (BDR) ensures that data and systems can be restored to an operational state following a destructive event — whether a ransomware attack, accidental deletion, hardware failure, or natural disaster. BDR in a security context requires both the technical capability to restore systems and the detection capability to identify when a destructive event has occurred. Kubric integrates BDR through the KAI Keeper agent (which triggers backup verification and restoration workflows), CoreSec FIM (detecting backup file modifications or mass deletions that indicate ransomware), Vault (providing the secret keys needed for restoration), and Watchdog (confirming system liveness after recovery).

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Ransomware / Destructive Event Detection | CoreSec YARA scanner | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| Mass Change / Deletion Detection | CoreSec FIM inotify | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| Backup Integrity Monitoring | CoreSec FIM BLAKE3 | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs` |
| Recovery Orchestration | KAI Keeper remediation planner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py` |
| Recovery Credential Fetch | KAI Keeper Vault fetcher | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py` |
| Post-Recovery Health Check | Watchdog agent orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| Configuration Restore (GitOps) | ArgoCD hard sync | `services/k-svc/` manifests |
| Config Convergence | KAI Housekeeper Ansible runner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py` |
| DR Notification | KAI Comm VAPI + Twilio | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py` |
| Memory Forensics (pre-wipe) | CoreSec memory snapshot | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs` |

---

## 3. Data Flow Diagram

```
Destructive Event Indicators
│
├── CoreSec YARA: ransomware signature match
├── CoreSec FIM: mass MODIFY events (encryption)
├── CoreSec FIM: mass DELETE events (wiper)
└── NetGuard IDS: C2 associated with ransomware actors
        │
        ▼
KAI Triage: P1 case creation in TheHive
        │
        ▼
┌───────────────────────────────────────────────────────┐
│  DR Response Workflow                                │
│                                                     │
│  1. CoreSec: memory snapshot (forensic preservation) │
│  2. KAI Keeper: isolation via Cortex responder       │
│  3. KAI Comm: VAPI phone + Twilio SMS to on-call     │
│  4. KAI Keeper + Vault: fetch backup encryption keys │
│  5. Restore from verified clean backup               │
│  6. ArgoCD: hard sync to restore K8s desired state   │
│  7. KAI House: Ansible convergence on host configs   │
│  8. Watchdog: verify all agents healthy post-restore │
└───────────────────────────────────────────────────────┘
        │
        ▼
NATS: kubric.{tenant}.agent.status.v1 (all healthy)
ClickHouse: incident record + DR timeline
Post-Incident Review → Problem record
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric BDR Response |
|---|---|---|---|
| Impact | T1486 | Data Encrypted for Impact | YARA: ransomware signatures; FIM: mass modify |
| Impact | T1485 | Data Destruction | FIM: mass DELETE events |
| Impact | T1490 | Inhibit System Recovery | FIM: backup directory deletion; IDS: VSS attacks |
| Impact | T1491 | Defacement | FIM: web root modification at high rate |
| Defense Evasion | T1070 | Indicator Removal | FIM: log file deletion before encryption |
| Collection | T1005 | Data from Local System (pre-exfil) | NetGuard: high-volume egress before encryption |

---

## 5. BDR RTO / RPO Targets

| Component | RTO | RPO | Recovery Method |
|---|---|---|---|
| KAI API (Kubernetes) | < 15 min | Last ArgoCD sync | ArgoCD hard sync |
| ClickHouse | < 4h | 24h (daily backup) | S3 restore + replay NATS |
| PostgreSQL | < 2h | 1h (WAL streaming) | WAL-G restore |
| Vault | < 30 min | Near-zero (raft snapshot) | Vault operator unseal + raft restore |
| CoreSec/NetGuard agents | < 10 min | Stateless | SaltStack re-deploy |
| NATS JetStream | < 15 min | Last persisted message | JetStream cluster failover |

---

## 6. Integration Points

| System | BDR Role |
|---|---|
| **Vault** | Backup encryption keys and recovery credentials |
| **ArgoCD** | K8s state restoration via hard sync to Git |
| **TheHive** | DR incident case management |
| **Cortex** | Automated isolation responder to stop active ransomware |
| **Watchdog** | Post-recovery verification gate |
| **ClickHouse** | Backup of all security events; DR evidence retention |
| **Ansible** | Host configuration re-convergence after bare-metal restore |

---

## 7. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| Backup solution | External backup agent (Velero for K8s, pgBackRest for PG, ClickHouse backup) configured |
| Vault HA | Raft-based Vault HA with automatic unseal (AWS KMS or equivalent) |
| Off-site backup | Backup S3 bucket in separate region / cloud account |
| DR runbook | Stored in Git and reviewed quarterly |
| BDR test | Full DR test executed at minimum annually; results logged to TheHive |
| YARA ransomware rules | Current family signatures in `vendor/yara-rules/ransomware/` |
