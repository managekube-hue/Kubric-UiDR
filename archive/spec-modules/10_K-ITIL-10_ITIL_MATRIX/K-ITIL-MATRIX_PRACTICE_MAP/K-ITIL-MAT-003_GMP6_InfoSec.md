# K-ITIL-MAT-003 — ITIL 4 GMP6: Information Security Management
**Practice Code:** GMP6
**Practice Family:** General Management Practices
**Kubric Reference:** K-ITIL-MAT-003
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Information Security Management ensures that an organization's information and related assets are protected against threats such as unauthorized access, modification, disclosure, or destruction. It encompasses confidentiality, integrity, and availability (CIA) across all service components. Kubric implements ISM as a continuously evaluated, agent-enforced control layer: eBPF hooks, FIM watchdogs, network anomaly detection, and ML-powered process scoring work together to detect and respond to policy violations in near-real-time.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Confidentiality – Access Control | Authentik RBAC (JWT enforcement) | ArgoCD / `services/k-svc/` |
| Confidentiality – Secret Management | HashiCorp Vault injector | KAI Keeper `K-KAI-KP-003_vault_secret_fetcher.py` |
| Integrity – File Integrity Monitoring | CoreSec FIM (inotify + BLAKE3) | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| Integrity – Baseline Hashing | CoreSec BLAKE3 baseline | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs` |
| Availability – Agent Health | Watchdog orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| Threat Detection – Process | CoreSec eBPF execve hook | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs` |
| Threat Detection – Network | NetGuard IDS alert publisher | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs` |
| Threat Detection – Malware | CoreSec YARA + ML scorer | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| Threat Detection – Sigma | CoreSec Sigma evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| Incident Response Integration | KAI Triage → TheHive | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| Intelligence Integration | MISP galaxy query | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py` |
| Security Monitoring | ClickHouse long-term retention | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py` |
| OCSF Event Normalization | CoreSec OCSF event bridge | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-002_ocsf_event_bridge.rs` |

---

## 3. RACI Matrix

| Activity | CISO | SOC Analyst | Platform Eng | KAI Agents |
|---|---|---|---|---|
| Define ISM policy | **A** | C | C | I |
| Configure FIM watch paths | **A** | C | **R** | I |
| Tune Sigma detection rules | C | **A** | **R** | R |
| Respond to YARA/ML alerts | C | **A** | C | **R** |
| Manage Vault secret lifecycle | **A** | I | **R** | R |
| Perform IOC enrichment (MISP) | C | **R** | I | **R** |
| Maintain RBAC policies (Authentik) | **A** | C | **R** | I |
| Escalate critical ISM events | **A** | **R** | C | R |
| Review OCSF event bridge mapping | C | C | **A** | R |

---

## 4. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.endpoint.fim.v1` | Publish (CoreSec) | OCSF-4010 File Activity events; BLAKE3 old/new hash |
| `kubric.{tenant}.endpoint.process.v1` | Publish (CoreSec) | OCSF-4007 Process Activity; exe, cmdline, pid, ppid, blake3_hash |
| `kubric.{tenant}.endpoint.malware.v1` | Publish (CoreSec) | YARA + ML positive detections |
| `kubric.{tenant}.network.ids.alert.v1` | Publish (NetGuard) | Suricata-style IDS rule match alerts |
| `kubric.{tenant}.network.tls.anomaly.v1` | Publish (NetGuard) | TLS SNI anomalies detected by `K-XRO-NG-PCAP-002_tls_sni.rs` |
| `kubric.{tenant}.agent.heartbeat` | Pub/Sub (Watchdog) | Agent liveness heartbeats |
| `kubric.{tenant}.agent.status.v1` | Publish (Watchdog) | Aggregated agent health: healthy / stale / offline |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/ism/fim-events` | GET | Recent FIM changes; filter by path, severity |
| `/api/v1/ism/process-alerts` | GET | Sigma / YARA triggered process alerts |
| `/api/v1/ism/network-alerts` | GET | IDS and TLS anomaly alerts from NetGuard |
| `/api/v1/ism/ioc-lookup` | POST | MISP IOC lookup; body `{indicator, type}` |
| `/api/v1/ism/vault/status` | GET | Vault unsealed + KV path accessibility check |
| `/api/v1/ism/rbac/policy` | GET | Active Authentik RBAC policy export |

---

## 5. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| FIM Change Events – Unreviewed | 0 open > 1h (P1 paths) | CoreSec NATS `endpoint.fim.v1` |
| YARA Detection Response Time | < 15 min to case creation | KAI Triage → TheHive |
| Sigma Rule Hit Rate (false positives) | < 5% of volume | CoreSec Sigma evaluator |
| OCSF Event Normalization Coverage | 100% of published events | OCSF event bridge |
| Agent Fleet Integrity (all heartbeating) | 100% | Watchdog `agent.status.v1` |
| Vault Secret Rotation Compliance | 100% within TTL | KAI Keeper Vault fetcher |
| MISP IOC Enrichment Latency | < 30 s from alert | KAI Invest MISP query |
| RBAC Policy Violations Detected | 0 unresolved | Authentik audit log |

---

## 6. Information Security Control Flow

```
Host Events                          Network Events
    |                                      |
CoreSec eBPF (execve, openat2)       NetGuard PCAP+DPI+IDS
    |                                      |
    ├── FIM: inotify + BLAKE3              ├── Flow analysis (5-tuple)
    ├── Sigma rule evaluation              ├── TLS SNI inspection
    ├── YARA malware scan                  ├── IP threat intel lookup
    └── ML candle inference                └── IDS rule match
    |                                      |
    └──────────── NATS JetStream ──────────┘
                        |
              KAI Triage Agent
                  |        |
           llama3   OCSF-4007
           reasoning  analyzer
                  |
            SSVC + EPSS scoring
                  |
         ┌────────────────────┐
         | Case: TheHive       |
         | Enrich: MISP        |
         | Respond: Cortex     |
         | Store: ClickHouse   |
         └────────────────────┘
```

---

## 7. Integration Points

| System | ISM Role |
|---|---|
| **Wazuh** | Host-based IDS; complements CoreSec eBPF with agent-based detection |
| **TheHive** | Case management for confirmed ISM incidents |
| **MISP** | Threat intelligence platform; IOC and galaxy enrichment |
| **Cortex** | Automated response (isolation, block, quarantine) triggered by KAI Keeper |
| **Vault** | Secrets management; all agent credentials injected at runtime |
| **Authentik** | SSO + RBAC; JWT validation on all Kubric API routes |
| **cert-manager** | TLS certificate lifecycle for all inter-service mTLS |
| **ESO (External Secrets Operator)** | Syncs Vault secrets to Kubernetes secrets |
| **ClickHouse** | Immutable, columnar event store for long-term ISM audit trail |

---

## 8. References

- ITIL 4 Foundation: *Information Security Management* (GMP6)
- ISO/IEC 27001:2022 – Annex A Controls
- NIST SP 800-53 Rev 5 – Security and Privacy Controls
- OCSF Schema: https://schema.ocsf.io
- Kubric CoreSec: `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/`
- Kubric NetGuard: `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/`
