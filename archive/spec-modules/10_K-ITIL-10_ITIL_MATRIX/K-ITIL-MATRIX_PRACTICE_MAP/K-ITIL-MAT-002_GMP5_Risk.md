# K-ITIL-MAT-002 — ITIL 4 GMP5: Risk Management
**Practice Code:** GMP5
**Practice Family:** General Management Practices
**Kubric Reference:** K-ITIL-MAT-002
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Risk Management ensures that an organization understands and effectively handles risks to its services, operations, and assets. It encompasses risk identification, analysis, evaluation, treatment, and ongoing monitoring. In Kubric, risk management is fully instrumented: every detected event generates a quantified risk signal that feeds Monte Carlo loss-exceedance models, SSVC triage decisions, and EPSS exploitability scoring tracked against patch-compliance SLAs.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Risk Identification | CoreSec – Sigma rule evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| Risk Identification | CoreSec – YARA malware scanner | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| Risk Identification | NetGuard – IDS rule loader | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs` |
| Risk Analysis | KAI Risk – pyFAIR Monte Carlo | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py` |
| Risk Quantification | KAI Risk – EPSS scorer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py` |
| Risk Evaluation / Triage | KAI Risk – SSVC decision tree | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py` |
| Risk Treatment | KAI Keeper – remediation planner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py` |
| Risk Monitoring | KAI Sentinel – health score publisher | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py` |
| Continuous Validation | KAI Foresight – LSTM threat baseline | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py` |

---

## 3. RACI Matrix

| Activity | Risk Manager | SOC Analyst | Platform Eng | KAI Agents |
|---|---|---|---|---|
| Define risk appetite | **A** | C | I | I |
| Identify risks from detections | C | **A** | I | **R** |
| Run pyFAIR loss simulations | **A** | C | I | **R** |
| Score CVEs with EPSS | C | **R** | I | **R** |
| Apply SSVC triage outcomes | **A** | **R** | I | R |
| Create remediation tickets | C | **A** | R | **R** |
| Monitor risk trend dashboard | **R** | R | C | **R** |
| Quarterly risk register review | **A** | C | C | I |

---

## 4. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.endpoint.process.v1` | Subscribe | CoreSec OCSF-4007 process events feed risk identification |
| `kubric.{tenant}.endpoint.fim.v1` | Subscribe | CoreSec OCSF-4010 FIM change events |
| `kubric.{tenant}.network.ids.alert.v1` | Subscribe | NetGuard IDS alert publisher output |
| `kubric.{tenant}.risk.score.v1` | Publish | pyFAIR risk output (LEF, LM, loss exceedance) |
| `kubric.{tenant}.risk.ssvc.v1` | Publish | SSVC triage decision per CVE/finding |
| `kubric.{tenant}.risk.epss.v1` | Publish | EPSS enriched vulnerability records |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/risk/score` | GET | Current aggregate risk score |
| `/api/v1/risk/epss/{cve_id}` | GET | EPSS score for a specific CVE |
| `/api/v1/risk/ssvc` | POST | Run SSVC decision for payload `{cve, exploitation, automatable, impact}` |
| `/api/v1/risk/fair-simulation` | POST | Trigger Monte Carlo run; body: `{lef_min, lef_max, lm_min, lm_max, simulations}` |
| `/api/v1/risk/findings` | GET | Paginated open risk findings with SSVC outcomes |

---

## 5. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| Open Immediate-SSVC Findings | 0 open > 24h | KAI Risk SSVC |
| Mean EPSS Score (active fleet CVEs) | < 0.10 | EPSS Scorer |
| pyFAIR 90th Percentile Annual Loss | Within budget | pyFAIR Monte Carlo |
| Risk Register Completeness | 100% | KAI Risk agent register |
| Remediation SLA Compliance (Critical) | ≥ 98% | KAI Keeper planner |
| Mean Time to Risk Score (MTRS) | < 5 min from detection | NATS latency |
| SSVC Triage Coverage | 100% of CVE findings | KAI Risk SSVC |

---

## 6. Risk Treatment Workflow

```
Detection Event (CoreSec / NetGuard)
        |
        v
NATS: kubric.{tenant}.endpoint.process.v1
        |
        v
KAI Triage Agent ──► SSVC Decision (K-KAI-RISK-003)
        |
   ┌────┴──────────────────────────┐
   | SSVC Outcome: Immediate       |
   | SSVC Outcome: Out-of-Cycle    |
   | SSVC Outcome: Scheduled       |
   | SSVC Outcome: Defer           |
   └────────────────────────────────┘
        |
        v
KAI Keeper: Remediation Planner (K-KAI-KP-001)
   → Cortex responder (K-KAI-KP-002)
   → Vault secret fetch (K-KAI-KP-003)
        |
        v
Publish: kubric.{tenant}.risk.ssvc.v1
        |
        v
Grafana Risk Dashboard (Prometheus + ClickHouse)
```

---

## 7. Integration Points

| System | Role in Risk Management |
|---|---|
| **MISP** | Threat intelligence enrichment; IOC correlation for risk identification |
| **TheHive** | Risk findings create cases; severity mapped to SSVC outcome |
| **Cortex** | Automated response actions triggered by KAI Keeper on Immediate findings |
| **Vault** | Secrets accessed during remediation; audit trail for risk treatment evidence |
| **EPSS API** | Probabilistic exploitability scores consumed by `K-KAI-RISK-002_epss_scorer.py` |
| **Wazuh** | Host-based alert feed supplements CoreSec detections in risk register |
| **ClickHouse** | Risk history store; supports trend analysis and loss-exceedance charting |
| **Grafana** | Risk posture dashboards; alert rules on SSVC Immediate findings |

---

## 8. References

- ITIL 4 Foundation: *Risk Management* (GMP5)
- SSVC: CERT/CC Stakeholder-Specific Vulnerability Categorization Guide v2.1
- EPSS: FIRST EPSS v3 Model Documentation
- pyFAIR: Open Group FAIR Standard implementation
- Kubric CoreSec: `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/`
- Kubric KAI Risk: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/`
