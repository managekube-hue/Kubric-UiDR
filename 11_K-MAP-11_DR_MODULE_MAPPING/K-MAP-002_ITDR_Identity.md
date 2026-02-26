# K-MAP-002 — Identity Threat Detection & Response (ITDR)
**Discipline:** Identity Threat Detection & Response
**Abbreviation:** ITDR
**Kubric Reference:** K-MAP-002
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Identity Threat Detection and Response (ITDR) focuses on protecting identity infrastructure — authentication systems, directory services, privileged accounts, and access tokens — from compromise and abuse. ITDR detects anomalous authentication patterns, credential theft indicators, lateral movement via identity, and privilege escalation through identity systems. Kubric implements ITDR through Authentik (OIDC/RBAC enforcement), the KAI Sentinel credential health module (HIBP), the KAI Invest graph investigation engine (which traces identity-based lateral movement), and MISP galaxy correlation for known identity-targeting threat actors.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Authentication Enforcement | Authentik OIDC provider | `services/k-svc/` (JWT middleware) |
| RBAC Policy Enforcement | Authentik + FastAPI JWT | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py` |
| Credential Breach Detection | KAI Sentinel HIBP score | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py` |
| Identity Threat Hunting | KAI Invest graph investigation | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py` |
| Threat Actor Correlation | KAI Invest MISP galaxy | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py` |
| Credential File Monitoring | CoreSec FIM (/etc/passwd, /etc/shadow) | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| sudo / SUID Monitoring | CoreSec Sigma evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| Secret Access Audit | HashiCorp Vault audit log | KAI Keeper `K-KAI-KP-003_vault_secret_fetcher.py` |
| Identity Anomaly Triage | KAI Triage OCSF analyzer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py` |

---

## 3. Data Flow Diagram

```
Identity Events
│
├── Authentik: login, logout, failed auth, permission deny
│       └── Syslog export ──► ClickHouse: kubric.auth_events
│
├── CoreSec FIM: /etc/passwd, /etc/shadow, /etc/sudoers
│       └── NATS: kubric.{tenant}.endpoint.fim.v1
│
├── Vault Audit Log: secret read, denied access
│       └── Syslog ──► ClickHouse: kubric.vault_audit
│
├── CoreSec Sigma: sudo abuse, SUID execution
│       └── NATS: kubric.{tenant}.endpoint.process.v1
│
└── KAI Sentinel: HIBP credential breach check
        NATS: kubric.{tenant}.sla.health.v1 (health score component)

                    │
                    ▼
          KAI Triage Agent
           ├── OCSF class analyzer
           ├── KISS priority score
           └── TheHive case creation
                    │
          KAI Invest Agent
           ├── MISP galaxy query (threat actor TTP map)
           └── Graph investigation
                ├── Correlate: auth_events → process_events → fim_events
                └── Identify lateral movement paths
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Credential Access | T1003 | OS Credential Dumping | FIM: /etc/shadow access + Sigma |
| Credential Access | T1110 | Brute Force | Authentik failed-auth event rate |
| Credential Access | T1555 | Credentials from Password Stores | Vault audit: unauthorized access |
| Privilege Escalation | T1548 | Abuse Elevation Control Mechanism | Sigma: sudo -l, SUID exec |
| Privilege Escalation | T1078 | Valid Accounts (misuse) | Authentik anomaly + HIBP score |
| Lateral Movement | T1021 | Remote Services | NetGuard flow + Invest graph investigation |
| Lateral Movement | T1550 | Use Alternate Authentication Material | KAI Invest graph + Sigma |
| Defense Evasion | T1108 | Redundant Access | FIM: new SSH key, cron addition |
| Persistence | T1098 | Account Manipulation | FIM: /etc/passwd, /etc/shadow |
| Persistence | T1136 | Create Account | FIM + Sigma: useradd detection |
| Initial Access | T1078 | Valid Accounts | Authentik + HIBP credential breach |
| Discovery | T1087 | Account Discovery | Sigma: /etc/passwd read, getent |

---

## 5. Integration Points

| System | ITDR Role |
|---|---|
| **Authentik** | Primary IdP; OIDC token issuance; RBAC enforcement; event logging |
| **HIBP API** | Credential breach scoring (KAI Sentinel) |
| **MISP** | Credential-targeting threat actor galaxy correlation |
| **Vault** | Secret access audit; unauthorized access = ITDR signal |
| **TheHive** | Cases created for identity-based incidents |
| **Cortex** | HIBP, VirusTotal analyzers for credential investigation |
| **CoreSec** | FIM on identity-related files; Sigma on privilege escalation patterns |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| Authentik | Deployed as OIDC provider; event logging enabled to syslog/ClickHouse |
| Vault | Audit logging enabled (file or syslog backend) |
| CoreSec FIM paths | Must include `/etc/passwd`, `/etc/shadow`, `/etc/sudoers`, `~/.ssh/authorized_keys` |
| HIBP API key | Set `HIBP_API_KEY` in Vault; consumed by `K-KAI-SEN-003` |
| MISP URL and API key | Set in Vault; consumed by `K-KAI-IV-001` |
| ClickHouse | `kubric.auth_events` table for Authentik log ingestion |
