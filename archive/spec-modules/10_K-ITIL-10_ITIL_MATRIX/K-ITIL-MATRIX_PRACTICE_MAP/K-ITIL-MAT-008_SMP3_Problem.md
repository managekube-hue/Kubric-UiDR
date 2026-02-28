# K-ITIL-MAT-008 — ITIL 4 SMP3: Problem Management
**Practice Code:** SMP3
**Practice Family:** Service Management Practices
**Kubric Reference:** K-ITIL-MAT-008
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Problem Management identifies, analyzes, and controls the underlying causes of incidents to prevent recurrence and minimize impact. It operates in three modes: reactive (triggered by incident patterns), proactive (driven by trend analysis), and error control (managing known errors). Kubric supports all three modes: the KAI Invest agent performs graph-based investigation to find root causes, the LSTM Foresight agent detects anomalies before they manifest as incidents, and the Sigma rule engine provides structured detection logic that serves as evidence for known-error records.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Problem Identification (reactive) | KAI Triage – OCSF analyzer (pattern grouping) | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py` |
| Problem Investigation | KAI Invest – graph investigation | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py` |
| Threat Context for Root Cause | KAI Invest – MISP galaxy query | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py` |
| Proactive Problem Detection | KAI Foresight – LSTM baseline | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py` |
| Proactive Problem Detection | KAI Foresight – EPSS enrichment | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-002_epss_enrichment.py` |
| ML Anomaly Detection | KAI Foresight – Hikari trainer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-003_hikari_trainer.py` |
| Known-Error Detection (process) | CoreSec – Sigma evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| Known-Error Detection (network) | NetGuard – IDS rule loader | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs` |
| Forensic Evidence Collection | CoreSec – memory snapshot | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs` |
| Root Cause Hunting | KAI Hunter – Velociraptor artifacts | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-001_velociraptor_artifacts.py` |
| Sigma Threat Hunting | KAI Hunter – Sigma hunting runner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-002_sigma_hunting_runner.py` |
| Trend Analysis Storage | ClickHouse | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py` |

---

## 3. RACI Matrix

| Activity | Problem Manager | SOC Lead | Platform Eng | KAI Agents |
|---|---|---|---|---|
| Identify problem from incident pattern | **A** | R | I | **R** |
| Log problem record in TheHive | **A** | R | I | R |
| Perform graph investigation | **A** | R | I | **R** |
| Collect forensic artifacts | C | **A** | **R** | R |
| Run Sigma hunting campaign | C | **A** | I | **R** |
| Correlate with MISP threat galaxy | C | **R** | I | **R** |
| Document known error | **A** | C | I | R |
| Propose and implement workaround | **A** | C | **R** | R |
| Validate LSTM anomaly alert | **A** | R | I | **R** |
| Close problem record | **A** | C | I | I |

---

## 4. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.endpoint.process.v1` | Subscribe | Historical process events feed problem investigation |
| `kubric.{tenant}.network.flow.v1` | Subscribe | Network flow history for lateral movement investigation |
| `kubric.{tenant}.problem.identified.v1` | Publish | Problem record created: source_incidents[], root_cause_hypothesis |
| `kubric.{tenant}.problem.known-error.v1` | Publish | Known error record with workaround and Sigma rule reference |
| `kubric.{tenant}.hunt.result.v1` | Publish | Sigma / Velociraptor hunting result set |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/problems` | GET | Open problem records with incident linkage |
| `/api/v1/problems/{id}` | GET | Problem detail: root cause analysis, timeline, workarounds |
| `/api/v1/problems/{id}/investigate` | POST | Trigger KAI Invest graph investigation |
| `/api/v1/problems/{id}/hunt` | POST | Launch Sigma/Velociraptor hunting campaign |
| `/api/v1/problems/known-errors` | GET | Known-error database |
| `/api/v1/problems/trend` | GET | ClickHouse incident pattern query (top recurring event types) |

---

## 5. Problem Investigation Flow

```
Multiple Related Incidents
        |
        v
KAI Triage: incident pattern grouping (OCSF class + asset cluster)
        |
        v
Problem Record Created → TheHive
        |
        v
   ┌────┴──────────────────────────────────────────────────┐
   | Reactive Investigation                               |
   | KAI Invest: graph investigation (K-KAI-IV-002)       |
   |   └── Correlate: process parents, network flows,     |
   |        FIM changes, login events                     |
   | KAI Invest: MISP galaxy query (K-KAI-IV-001)         |
   |   └── Map to threat actor TTPs                       |
   └────────────────────────────────────────────────────────┘
        |
        v
   ┌────┴──────────────────────────────────────────────────┐
   | Forensic Collection                                  |
   | CoreSec: memory snapshot (K-XRO-CS-FR-001)           |
   | KAI Hunter: Velociraptor artifacts (K-KAI-HU-001)    |
   | KAI Hunter: Sigma hunting runner (K-KAI-HU-002)      |
   └────────────────────────────────────────────────────────┘
        |
        v
Root Cause Identified
   → Known Error Record
   → Sigma rule updated / new rule authored
   → Problem closed with PIR (Post-Incident Review) link

   ┌────────────────────────────────────────┐
   | Proactive: LSTM detects deviation      |
   | from baseline → Problem raised before  |
   | incident occurs (Foresight K-KAI-FS-*)  |
   └─────────────────────────────────────────┘
```

---

## 6. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| Problem Backlog Age (open > 30 days) | 0 | TheHive problem records |
| Recurring Incident Rate (same root cause) | < 3% | ClickHouse incident pattern query |
| Mean Time to Root Cause | < 72h for P1 problems | Problem record timeline |
| Known-Error Coverage (documented workaround) | 100% | Known-error database |
| Proactive Problem Detection Rate | ≥ 20% of problems | LSTM anomaly → problem record |
| Hunting Campaign Hit Rate | ≥ 15% productive | KAI Hunter sigma_hunting_runner |
| Sigma Rule Bank Growth (new rules from PIR) | ≥ 2 rules/quarter | Sigma rule directory |

---

## 7. Integration Points

| System | Problem Management Role |
|---|---|
| **TheHive** | Problem records linked to source incidents; known-error tracking |
| **MISP** | Threat intelligence provides TTP context for root cause analysis |
| **Velociraptor** | Live artifact collection for forensic root cause evidence |
| **Cortex** | Automated analysis responders during investigation |
| **ClickHouse** | Long-term event store; enables historical pattern queries |
| **Wazuh** | Additional host telemetry source for problem investigation |
| **Grafana** | Problem trend dashboards; recurring event heatmaps |

---

## 8. References

- ITIL 4 Foundation: *Problem Management* (SMP3)
- Velociraptor: https://docs.velociraptor.app
- Sigma rules: https://github.com/SigmaHQ/sigma
- Kubric Hunter: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/`
- Kubric Invest: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/`
- Kubric Foresight: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/`
