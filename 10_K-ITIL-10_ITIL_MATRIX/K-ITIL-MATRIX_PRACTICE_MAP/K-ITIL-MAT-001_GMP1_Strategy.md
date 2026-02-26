# K-ITIL-MAT-001 — ITIL 4 GMP1: Strategy Management
**Practice Code:** GMP1
**Practice Family:** General Management Practices
**Kubric Reference:** K-ITIL-MAT-001
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Strategy Management establishes the direction of an organization's services and ensures that products and services meet defined organizational goals. It involves analyzing the environment, defining strategic intent, and translating direction into actionable portfolios. Within a security platform context, strategy management governs how security capabilities are invested, measured, and continuously improved to deliver stakeholder value.

---

## 2. Kubric Platform Mapping

The Kubric platform realizes Strategy Management through the KAI orchestration layer and its multi-persona CrewAI agent framework. Strategic decisions are data-driven, pulling from ClickHouse telemetry, pyFAIR risk models, and real-time agent health dashboards.

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Strategic Assessment | KAI Risk Agent – pyFAIR model | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py` |
| Strategic Planning | KAI Sentinel – health score publisher | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py` |
| Portfolio Management | KAI Simulate – LTV predictor | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py` |
| Strategic Measurement | PerfTrace – Prometheus /metrics | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs` |
| Risk-Informed Direction | KAI Risk – SSVC decision tree | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py` |
| Horizon Scanning | KAI Foresight – LSTM baseline | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py` |
| Strategic Communication | KAI Comm Agent – VAPI / Twilio | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py` |

---

## 3. RACI Matrix

| Activity | CISO | Platform Eng | SOC Lead | KAI Orchestrator |
|---|---|---|---|---|
| Define security strategy | **A** | C | C | I |
| Maintain risk-appetite thresholds | **A** | I | C | R |
| Publish strategic KPI dashboard | C | **R** | C | **R** |
| EPSS-based investment prioritization | A | C | **R** | **R** |
| Quarterly strategy review | **A** | C | C | I |
| Produce pyFAIR loss scenarios | C | I | C | **R** |

*R = Responsible, A = Accountable, C = Consulted, I = Informed*

---

## 4. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.agent.status.v1` | Subscribe | Watchdog agent health — feeds strategic health score |
| `kubric.{tenant}.metrics.host.v1` | Subscribe | PerfTrace host telemetry — platform capacity inputs |
| `kubric.{tenant}.risk.score.v1` | Publish | pyFAIR loss exceedance outputs |
| `kubric.{tenant}.strategy.kpi.v1` | Publish | Strategic KPI events for Grafana |

### REST API Endpoints (FastAPI)

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/strategy/risk-score` | GET | Current tenant risk score from pyFAIR |
| `/api/v1/strategy/health` | GET | Aggregated platform health (Sentinel) |
| `/api/v1/strategy/epss-top-n` | GET | Top-N EPSS enriched vulnerabilities |
| `/api/v1/strategy/ltv` | GET | Predicted lifetime value / churn risk |
| `/api/v1/strategy/ssvc` | POST | SSVC triage decision for a given CVE |

---

## 5. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| Mean Risk Score (pyFAIR LEF×LM) | < 0.35 | KAI Risk Agent |
| Agent Fleet Availability | ≥ 99.5% | Watchdog → NATS `agent.status.v1` |
| EPSS p90 Patch SLA Compliance | ≥ 95% | KAI Foresight EPSS enrichment |
| Strategic Dashboard Freshness | < 5 min lag | PerfTrace Prometheus scrape |
| Unmitigated Critical Findings (SSVC Immediate) | 0 open > 24h | KAI Risk SSVC |
| Threat Horizon Forecast Accuracy (LSTM MRE) | < 0.12 | KAI Foresight Hikari trainer |
| Customer Health Score p50 | ≥ 75 | KAI Sentinel health score publisher |

---

## 6. Integration Points

| System | Role in Strategy Management |
|---|---|
| **Grafana** | Strategic KPI dashboard frontend; consumes Prometheus endpoint `:9090/metrics` from PerfTrace |
| **ClickHouse** | Long-term telemetry store; KAI Bill agent queries audit logs for cost allocation |
| **Vault (HashiCorp)** | Secure storage of API keys consumed by KAI Keeper and strategy agents |
| **ArgoCD** | GitOps source of truth; strategic capacity changes tracked as Git commits |
| **MISP** | Threat intelligence feeds inform horizon scanning in KAI Foresight |
| **EPSS API** | Exploitability probability scores pulled by `K-KAI-FS-002_epss_enrichment.py` |
| **Authentik** | Identity provider enforcing RBAC on all `/api/v1/strategy/*` routes |

---

## 7. Configuration Reference

| Environment Variable | Agent | Purpose |
|---|---|---|
| `KUBRIC_TENANT_ID` | All | Tenant namespace for all NATS subjects |
| `KUBRIC_NATS_URL` | All | NATS broker endpoint |
| `FAIR_SIMULATION_RUNS` | KAI Risk | Monte Carlo iteration count (default: 10000) |
| `EPSS_API_URL` | KAI Foresight | EPSS enrichment endpoint |
| `LSTM_MODEL_PATH` | KAI Foresight | Hikari LSTM model weights path |
| `SENTINEL_HEALTH_THRESHOLD` | KAI Sentinel | Minimum acceptable health score (0–100) |

---

## 8. References

- ITIL 4 Foundation: *Strategy Management* (GMP1)
- Kubric Agent Architecture: `02_K-XRO-02_SUPER_AGENT/`
- KAI Orchestration Layer: `03_K-KAI-03_ORCHESTRATION/`
- pyFAIR documentation: https://pyfair.readthedocs.io
- EPSS: https://www.first.org/epss
- SSVC: https://certcc.github.io/SSVC
