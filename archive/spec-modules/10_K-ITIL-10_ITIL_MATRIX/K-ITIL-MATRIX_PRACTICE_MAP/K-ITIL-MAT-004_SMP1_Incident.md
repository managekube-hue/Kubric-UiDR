# K-ITIL-MAT-004 — ITIL 4 SMP1: Incident Management
**Practice Code:** SMP1
**Practice Family:** Service Management Practices
**Kubric Reference:** K-ITIL-MAT-004
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Incident Management minimizes negative impact on services by restoring normal operation as quickly as possible following an unplanned interruption or reduction in quality. In Kubric, every security detection event — from a Sigma-matched process execution to a YARA-positive binary scan — is treated as a potential incident. The KAI Triage agent automatically classifies, enriches, and routes detections to TheHive case management, with escalation paths driven by SSVC priority and on-call paging via VAPI/Twilio.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Incident Detection – Endpoint | CoreSec Sigma evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| Incident Detection – Malware | CoreSec YARA + ML inference | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| Incident Detection – FIM | CoreSec inotify watcher | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| Incident Detection – Network | NetGuard IDS alert publisher | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs` |
| Incident Triage | KAI Triage Agent + llama3 | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| Incident Classification | KAI OCSF analyzer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py` |
| Priority Scoring | KAI KISS calculator | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-004_kiss_calculator.py` |
| Incident Investigation | KAI Analyst – Cortex chain | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py` |
| Observable Enrichment | KAI Analyst – observable enrichment | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py` |
| Forensic Capture | CoreSec memory snapshot | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs` |
| Incident Escalation | KAI Comm Agent – VAPI phone | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-002_vapi_phone.py` |
| Incident Notification | KAI Comm Agent – Twilio SMS | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py` |
| Incident Resolution | KAI Keeper – remediation planner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py` |
| Rate Control | CoreSec Governor token bucket | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-GOVERNOR/K-XRO-CS-GV-001_token_bucket.rs` |

---

## 3. RACI Matrix

| Activity | SOC Analyst | Incident Manager | Platform Eng | KAI Agents |
|---|---|---|---|---|
| Detect incident | I | I | I | **R** |
| Classify and triage | **A** | C | I | **R** |
| Enrich with MISP/Cortex | **R** | C | I | **R** |
| Create TheHive case | **R** | **A** | I | R |
| Determine priority (KISS) | C | **A** | I | **R** |
| Notify stakeholders | C | **A** | I | **R** |
| Escalate P1 (phone/SMS) | **R** | **A** | I | R |
| Initiate forensic capture | **A** | C | **R** | R |
| Resolve and close | **R** | **A** | C | R |
| Post-incident review | C | **A** | C | I |

---

## 4. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Payload Schema |
|---|---|---|
| `kubric.{tenant}.endpoint.process.v1` | Subscribe | OCSF 4007 — pid, ppid, exe, cmdline, blake3_hash |
| `kubric.{tenant}.endpoint.fim.v1` | Subscribe | OCSF 4010 — path, old_hash, new_hash, severity |
| `kubric.{tenant}.endpoint.malware.v1` | Subscribe | YARA match: rule_name, file_path, scan_score |
| `kubric.{tenant}.network.ids.alert.v1` | Subscribe | IDS alert: rule_id, src_ip, dst_ip, proto, message |
| `kubric.{tenant}.incident.triage.v1` | Publish | Triage result: class, severity, ssvc_outcome, case_id |
| `kubric.{tenant}.incident.escalate.v1` | Publish | Escalation trigger: on_call_group, channel, priority |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/incidents` | GET | List open incidents with SSVC priority filter |
| `/api/v1/incidents/{id}` | GET | Full incident detail: events, observables, timeline |
| `/api/v1/incidents/{id}/triage` | POST | Manually trigger KAI Triage re-evaluation |
| `/api/v1/incidents/{id}/escalate` | POST | Trigger VAPI/Twilio escalation chain |
| `/api/v1/incidents/{id}/close` | POST | Close incident with resolution notes |
| `/api/v1/incidents/stats` | GET | MTTR, MTTD, volume by severity, closure rate |

---

## 5. KPIs and Metrics

| KPI | Target | Measurement |
|---|---|---|
| Mean Time to Detect (MTTD) | < 5 min | Detection event timestamp vs. alert timestamp |
| Mean Time to Triage (MTTT) | < 10 min | Alert timestamp vs. TheHive case creation |
| Mean Time to Respond (MTTR) | P1 < 30 min, P2 < 4h | Case open to first responder action |
| Mean Time to Resolve (MTTResolve) | P1 < 4h, P2 < 24h | Case created to case closed |
| False Positive Rate | < 5% | Closed as false positive / total cases |
| Escalation Rate | < 10% of P2 cases | Incidents requiring VAPI/Twilio escalation |
| OCSF Normalization Rate | 100% | Events passing OCSF bridge validation |
| Forensic Capture Success Rate | ≥ 95% on P1 | Memory snapshot completion rate |

---

## 6. Incident Lifecycle Flow

```
Detection Sources
  CoreSec (eBPF/Sigma/YARA/FIM)
  NetGuard (IDS/DPI/TLS)
        |
        v  NATS JetStream
        |
  KAI Triage Agent
   ├── llama3 reasoning (K-KAI-TR-002)
   ├── OCSF class analyzer (K-KAI-TR-003)
   └── KISS priority score (K-KAI-TR-004)
        |
        v
  Severity: P1 / P2 / P3 / P4
        |
  ┌─────┴──────────────────────────────────────┐
  |                                             |
  P1/P2: Immediate                        P3/P4: Queue
  ├── TheHive case created                 └── TheHive case created
  ├── KAI Analyst enrichment                   └── SOC review queue
  │    ├── Cortex analyzer chain
  │    └── Observable enrichment
  ├── KAI Comm: VAPI phone (P1)
  ├── KAI Comm: Twilio SMS
  └── KAI Keeper: remediation plan
        |
        v
  Resolution → Post-Incident Review
  → ClickHouse audit record
```

---

## 7. Integration Points

| System | Incident Management Role |
|---|---|
| **TheHive** | Primary case management platform; cases auto-created by KAI Triage |
| **Cortex** | Analyzer chain for observable enrichment; supported analyzers: VirusTotal, Shodan, MISP, AbuseIPDB |
| **MISP** | IOC lookup and Galaxy threat actor correlation |
| **Wazuh** | Supplementary host alert feed; alerts bridged via NATS |
| **VAPI** | AI-voice incident bridge for P1 on-call escalation |
| **Twilio** | SMS escalation for P1/P2 incidents to on-call groups |
| **ClickHouse** | Incident history, MTTR trends, SLA tracking |
| **Grafana** | Incident dashboard; real-time alert state and MTTR scorecards |
| **Velociraptor** | Live forensic artifact collection triggered during investigation |

---

## 8. References

- ITIL 4 Foundation: *Incident Management* (SMP1)
- OCSF Schema 1.1: https://schema.ocsf.io
- TheHive Project: https://thehive-project.org
- Cortex: https://github.com/TheHive-Project/Cortex
- Kubric Triage: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/`
- Kubric CoreSec: `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/`
