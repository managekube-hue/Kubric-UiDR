# K-ITIL-AUD-001 — Kubric Infra Controls (KIC) Evidence Map
**Document Code:** K-ITIL-AUD-001
**Practice Family:** Audit Readiness
**Kubric Reference:** K-ITIL-AUDIT_AUDIT_READINESS
**Last Updated:** 2026-02-26

---

## 1. Purpose

This document maps every Kubric infrastructure control to the audit evidence it produces, the agent or service responsible for generating that evidence, the relevant compliance frameworks (SOC 2 Type II, ISO/IEC 27001:2022, CIS Controls v8), and the storage location for auditor retrieval. It is intended for use during annual certification audits and ad-hoc control effectiveness reviews.

---

## 2. Evidence Production Architecture

```
Evidence Sources                    Evidence Store
─────────────────────               ────────────────
CoreSec   ──► NATS ──────────────►  ClickHouse (event tables)
NetGuard  ──► NATS ──────────────►  ClickHouse (flow + alert tables)
PerfTrace ──► Prometheus ────────►  Grafana (metrics + alerts)
Watchdog  ──► NATS ──────────────►  ClickHouse (agent_status table)
KAI Agents ─► ClickHouse + PG ──►  Audit log tables
ArgoCD    ──────────────────────►  Git history (immutable)
Vault     ──────────────────────►  Vault audit log (syslog)
Authentik ──────────────────────►  Authentik event log
```

---

## 3. KIC Evidence Map

### KIC-001: Asset Inventory and Discovery

| Attribute | Value |
|---|---|
| Control Description | All managed hosts and network devices are inventoried and tracked |
| Evidence Type | Agent registration records, Watchdog heartbeat history |
| Evidence Location | ClickHouse table: `kubric.agent_status_history` |
| Evidence Retention | 13 months rolling |
| Frequency | Continuous; Watchdog sweeps every 15 seconds |
| Kubric Module | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| SOC 2 TSC | CC6.1 (Logical and Physical Access Controls) |
| ISO 27001 | A.8.1 – Asset Inventory |
| CIS Control | CIS-1: Inventory and Control of Enterprise Assets |

---

### KIC-002: File Integrity Monitoring

| Attribute | Value |
|---|---|
| Control Description | Critical file system paths are monitored for unauthorized changes using BLAKE3 hashing |
| Evidence Type | OCSF-4010 FIM change events with old_hash, new_hash, path, timestamp |
| Evidence Location | NATS subject: `kubric.{tenant}.endpoint.fim.v1`; ClickHouse table: `kubric.fim_events` |
| Evidence Retention | 13 months rolling |
| Frequency | Real-time (inotify kernel events) |
| Kubric Module | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| Default Watch Paths | `/etc`, `/usr/bin`, `/usr/sbin` (configurable via `KUBRIC_FIM_PATHS`) |
| SOC 2 TSC | CC7.2 (System Monitoring) |
| ISO 27001 | A.8.8 – Technical Vulnerability Management |
| CIS Control | CIS-10: Malware Defenses |

---

### KIC-003: Process Execution Monitoring

| Attribute | Value |
|---|---|
| Control Description | All new process executions are captured with pid, ppid, exe, cmdline, BLAKE3 hash |
| Evidence Type | OCSF-4007 Process Activity events |
| Evidence Location | NATS: `kubric.{tenant}.endpoint.process.v1`; ClickHouse: `kubric.process_events` |
| Evidence Retention | 13 months rolling |
| Frequency | Real-time (eBPF on Linux; sysinfo fallback every 5s) |
| Kubric Module | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs` |
| SOC 2 TSC | CC7.2 (System Monitoring) |
| ISO 27001 | A.8.15 – Logging |
| CIS Control | CIS-8: Audit Log Management |

---

### KIC-004: Network Traffic Analysis

| Attribute | Value |
|---|---|
| Control Description | Network flows are captured and analyzed for anomalies, IDS rule matches, and TLS anomalies |
| Evidence Type | FlowEvent (5-tuple, bytes, packets, duration), IDS alerts, TLS SNI anomalies |
| Evidence Location | NATS: `kubric.{tenant}.network.flow.v1`, `kubric.{tenant}.network.ids.alert.v1` |
| Evidence Retention | Flow data: 90 days; IDS alerts: 13 months |
| Frequency | Real-time (libpcap / AF_PACKET ring / DPDK) |
| Kubric Module | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/` |
| SOC 2 TSC | CC7.2, CC7.3 |
| ISO 27001 | A.8.16 – Monitoring Activities |
| CIS Control | CIS-13: Network Monitoring and Defense |

---

### KIC-005: Malware Detection

| Attribute | Value |
|---|---|
| Control Description | All process launches are scanned with YARA rules and ML anomaly scoring |
| Evidence Type | YARA match records: rule_name, file_hash, scan_score, timestamp |
| Evidence Location | NATS: `kubric.{tenant}.endpoint.malware.v1`; ClickHouse: `kubric.malware_events` |
| Evidence Retention | 13 months |
| Frequency | Real-time on file access (YARA) + every 5s poll (ML) |
| Kubric Module | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/` |
| SOC 2 TSC | CC7.1 (Vulnerability Identification) |
| ISO 27001 | A.8.7 – Protection Against Malware |
| CIS Control | CIS-10: Malware Defenses |

---

### KIC-006: Access Control and Authentication

| Attribute | Value |
|---|---|
| Control Description | All Kubric API access is protected by JWT (Authentik OIDC) with RBAC enforcement |
| Evidence Type | Authentik event log: login, logout, permission checks; JWT audit in FastAPI |
| Evidence Location | Authentik event log (syslog export); PostgreSQL: `kubric.api_access_log` |
| Evidence Retention | 13 months |
| Frequency | Per-request |
| Kubric Module | `services/k-svc/` (JWT middleware); Authentik IDP |
| SOC 2 TSC | CC6.1, CC6.2 |
| ISO 27001 | A.9.1 – Access Control Policy |
| CIS Control | CIS-5: Account Management |

---

### KIC-007: Secret Management

| Attribute | Value |
|---|---|
| Control Description | All credentials and secrets are stored in HashiCorp Vault; no secrets committed to Git |
| Evidence Type | Vault audit log: secret reads, rotations, access denials |
| Evidence Location | Vault audit log (syslog/file); ESO sync records in Kubernetes events |
| Evidence Retention | 13 months |
| Frequency | Per-access event |
| Kubric Module | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py` |
| SOC 2 TSC | CC6.1, CC6.3 |
| ISO 27001 | A.9.4 – System and Application Access Control |
| CIS Control | CIS-16: Application Software Security |

---

### KIC-008: Change and Deployment Control

| Attribute | Value |
|---|---|
| Control Description | All production changes require Git PR with review; ArgoCD enforces GitOps |
| Evidence Type | Git commit history (immutable); ArgoCD sync events; ClickHouse deploy audit |
| Evidence Location | Git repository (immutable log); ClickHouse: `kubric.change_audit` |
| Evidence Retention | Indefinite (Git) + 13 months (ClickHouse) |
| Frequency | Per deployment event |
| Kubric Module | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/` |
| SOC 2 TSC | CC8.1 (Change Management) |
| ISO 27001 | A.8.32 – Change Management |
| CIS Control | CIS-18: Penetration Testing (change validation) |

---

### KIC-009: Vulnerability Management

| Attribute | Value |
|---|---|
| Control Description | CVEs are enriched with EPSS, triaged with SSVC, and tracked to remediation |
| Evidence Type | SSVC triage decisions; EPSS scores; remediation plan records; patch SLA compliance |
| Evidence Location | ClickHouse: `kubric.vuln_findings`; KAI Risk SSVC records |
| Evidence Retention | 13 months |
| Frequency | Daily EPSS refresh; real-time SSVC on new findings |
| Kubric Module | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/` |
| SOC 2 TSC | CC7.1 |
| ISO 27001 | A.8.8 – Technical Vulnerability Management |
| CIS Control | CIS-7: Continuous Vulnerability Management |

---

### KIC-010: Incident Management and Response

| Attribute | Value |
|---|---|
| Control Description | Security incidents are detected, triaged, and tracked with full audit trail |
| Evidence Type | TheHive case records; triage events; escalation logs; resolution notes |
| Evidence Location | TheHive database; ClickHouse: `kubric.incident_events` |
| Evidence Retention | 5 years (TheHive cases); 13 months (ClickHouse) |
| Frequency | Per incident |
| Kubric Module | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/` |
| SOC 2 TSC | CC7.3, CC7.4, CC7.5 |
| ISO 27001 | A.5.24 – Incident Management Planning |
| CIS Control | CIS-17: Incident Response Management |

---

### KIC-011: Platform Availability and Performance

| Attribute | Value |
|---|---|
| Control Description | Platform availability and performance SLOs are continuously measured |
| Evidence Type | Prometheus time-series metrics; Watchdog agent status history |
| Evidence Location | Prometheus TSDB (30 days); ClickHouse: `kubric.agent_status_history` (13 months) |
| Evidence Retention | 13 months |
| Frequency | 15-second Watchdog sweep; 15-second PerfTrace collection |
| Kubric Module | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/` + `K-XRO-WD_WATCHDOG/` |
| SOC 2 TSC | A1.1, A1.2, A1.3 (Availability) |
| ISO 27001 | A.8.6 – Capacity Management |
| CIS Control | CIS-16: Application Software Security |

---

## 4. Auditor Access Procedure

1. Request read-only Grafana viewer credentials from Platform Engineering.
2. Query ClickHouse via `K-KAI-API-004_clickhouse_connect.py` with tenant scope.
3. Export TheHive case records for specified date range via TheHive API.
4. Download Vault audit log export from the designated log bucket.
5. Review Git history for change evidence: `git log --format="%H %an %ae %s" --since="[date]"`.
6. Request Authentik event log export for specified period.

---

## 5. References

- SOC 2 Trust Services Criteria (2017 rev.)
- ISO/IEC 27001:2022 Annex A
- CIS Controls v8: https://www.cisecurity.org/cis-controls
- NIST SP 800-53 Rev 5
- Kubric platform source: `02_K-XRO-02_SUPER_AGENT/` and `03_K-KAI-03_ORCHESTRATION/`
