# K-MAP-017 — Governance, Risk & Compliance (GRC)
**Discipline:** Governance, Risk & Compliance
**Abbreviation:** GRC
**Kubric Reference:** K-MAP-017
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Governance, Risk and Compliance (GRC) provides the organizational framework for ensuring that an organization achieves its strategic objectives, manages its risks appropriately, and operates in accordance with applicable laws and regulations. GRC encompasses policy management, risk identification and treatment, compliance evidence collection, audit readiness, and regulatory reporting. Kubric's GRC capability is built on a foundation of continuous control evidence production (all K-XRO agents produce auditable OCSF events), risk quantification through pyFAIR and SSVC, compliance crosswalk documentation (K-ITIL-AUD-002), and ClickHouse-backed audit retention meeting SOC 2 and ISO 27001 requirements.

---

## 2. Kubric Modules

| GRC Sub-Capability | Module | File Path |
|---|---|---|
| Risk Quantification (FAIR) | KAI Risk pyFAIR model | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py` |
| Exploitability Probability | KAI Risk EPSS scorer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py` |
| Vulnerability Triage (SSVC) | KAI Risk SSVC decision | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py` |
| Compliance Evidence | All K-XRO agents (OCSF events) | `02_K-XRO-02_SUPER_AGENT/` |
| Audit Trail Store | ClickHouse / KAI Bill audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |
| SLA / Service Level Evidence | KAI Bill HLE calculator | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py` |
| Compliance Reports | KAI Bill invoice renderer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py` |
| Control Evidence Map | K-ITIL-AUD-001 | `10_K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-001_KIC_evidence_map.md` |
| SOC2/ISO Crosswalk | K-ITIL-AUD-002 | `10_K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-002_soc2_iso_crosswalk.cs` |
| ITIL Practice Mapping | K-ITIL-MAT-001 to 009 | `10_K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/` |
| Change Control Evidence | ArgoCD Git history | Git repository (immutable) |
| Secret Management Evidence | Vault audit log | HashiCorp Vault |
| Access Control Evidence | Authentik event log | Authentik + ClickHouse |

---

## 3. GRC Framework Coverage

| Framework | Kubric Coverage Level | Evidence Source |
|---|---|---|
| SOC 2 Type II (CC + A1) | Full TSC coverage via KIC-001 to KIC-011 | ClickHouse, Vault audit, Authentik, Git |
| ISO/IEC 27001:2022 | Annex A controls mapped via K-ITIL-AUD-002 | Same as SOC 2 |
| ITIL 4 | GMP1, GMP5, GMP6, SMP1, SMP3, SMP7, SMP10, SMP12, TMP2 | K-ITIL-MAT-001 to 009 |
| MITRE ATT&CK | T1-T40 coverage via EDR + NDR + CFDR | K-MAP-001 to 016 |
| CIS Controls v8 | CIS-1,5,7,8,10,13,16,17,18 addressed | KIC evidence map |
| NIST SP 800-53 | Selected controls via ITIL/SOC2 crosswalk | K-ITIL-AUD-002 |
| FAIR (Open Group) | Monte Carlo loss quantification | KAI Risk pyFAIR |
| SSVC | Stakeholder-specific vulnerability triage | KAI Risk SSVC |

---

## 4. Data Flow Diagram

```
Control Evidence Production (continuous)
│
├── CoreSec → NATS → ClickHouse
│   (process events, FIM, malware, OCSF-normalized)
│
├── NetGuard → NATS → ClickHouse
│   (flow events, IDS alerts, TLS anomalies)
│
├── PerfTrace → Prometheus + NATS → ClickHouse
│   (host metrics, availability evidence)
│
├── Watchdog → NATS → ClickHouse
│   (agent status, fleet availability SLI)
│
├── Vault → Audit log → ClickHouse
│   (secret access evidence)
│
└── Authentik → Event log → ClickHouse
    (authentication and RBAC evidence)

                    ▼
          GRC Evidence Repository
          ClickHouse (13-month retention)
          + Git (indefinite change history)
          + TheHive (5-year incident records)

                    ▼
          KAI Risk Agent
          ├── pyFAIR: annual loss exceedance
          ├── EPSS: CVE exploitation probability
          └── SSVC: triage decision audit trail

                    ▼
          KAI Bill Agent
          ├── ClickHouse audit query
          ├── HLE calculation
          └── Compliance report / invoice rendering

                    ▼
          Grafana / Auditor Portal
          └── Evidence export on demand
```

---

## 5. GRC KPIs

| KPI | Target | Source |
|---|---|---|
| Control Evidence Completeness | 100% of KIC controls with evidence | K-ITIL-AUD-001 |
| Audit Readiness Score | ≥ 95% | KAI Bill ClickHouse audit |
| Open SSVC Immediate Findings | 0 > 24h | KAI Risk SSVC |
| pyFAIR Annual Loss within Budget | Yes | KAI Risk pyFAIR |
| SOC 2 Evidence Retention | 13 months continuous | ClickHouse TTL |
| RBAC Policy Violations | 0 unresolved | Authentik audit log |
| Change Control Compliance | 100% via GitOps | ArgoCD + Git |

---

## 6. Integration Points

| System | GRC Role |
|---|---|
| **ClickHouse** | Primary GRC evidence store; 13-month retention for all event types |
| **Vault** | Secret management audit; non-repudiation of credential access |
| **Authentik** | Access control evidence; RBAC policy enforcement audit |
| **ArgoCD + Git** | Change management evidence; immutable GitOps history |
| **TheHive** | Incident management evidence; 5-year case retention |
| **EPSS API** | Exploitability probability for risk register |
| **MISP** | Threat intelligence context for risk assessments |
| **Grafana** | GRC executive dashboard; SLO, risk score, compliance metrics |

---

## 7. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| ClickHouse | 13-month TTL on all event tables |
| Vault | Audit log enabled; exported to durable storage |
| Authentik | Event log exported to ClickHouse |
| ArgoCD | All production changes via Git PR with reviewer |
| pyFAIR config | Risk appetite thresholds defined in `FAIR_LEF_*` and `FAIR_LM_*` env vars |
| SSVC config | Decision tree inputs configured: exploitation, automatable, mission impact |
| Auditor access | Read-only Grafana and ClickHouse credentials provisioned |
