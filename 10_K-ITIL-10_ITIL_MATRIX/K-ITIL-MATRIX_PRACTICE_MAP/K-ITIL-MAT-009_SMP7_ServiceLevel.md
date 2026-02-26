# K-ITIL-MAT-009 — ITIL 4 SMP7: Service Level Management
**Practice Code:** SMP7
**Practice Family:** Service Management Practices
**Kubric Reference:** K-ITIL-MAT-009
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Service Level Management sets clear, business-based targets for service levels, ensures services are delivered to those targets, and provides an ongoing review mechanism for continual improvement. In the context of a security operations platform, service levels govern detection latency, response times, agent availability, and customer health scores. Kubric tracks SLOs continuously via Prometheus metrics from PerfTrace, Watchdog agent status events, and the KAI Sentinel health score publisher, with billing-grade SLA evidence produced by the KAI Bill agent.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Service Level Target Definition | KAI Sentinel – health score publisher | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py` |
| SLA Evidence and Reporting | KAI Bill – ClickHouse audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |
| Customer Health Prediction | KAI Sentinel – churn risk model | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py` |
| Credential Health Check | KAI Sentinel – HIBP credential score | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py` |
| SLA Billing Calculation | KAI Bill – HLE calculator | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py` |
| Invoice Generation | KAI Bill – invoice renderer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py` |
| Platform Availability Monitoring | Watchdog orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| Operational Metrics | PerfTrace Prometheus | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs` |
| LTV / Churn Simulation | KAI Simulate – LTV predictor | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py` |
| SLA Notification | KAI Comm – Twilio SMS | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py` |

---

## 3. Service Level Definitions

| Service Level Indicator (SLI) | Service Level Objective (SLO) | SLA Threshold |
|---|---|---|
| Agent Fleet Availability | ≥ 99.5% uptime per month | < 99.0% triggers SLA credit |
| Detection Event Latency (MTTD) | p95 < 5 min | > 15 min breach → incident SLA violation |
| Triage to Case Creation (MTTT) | p95 < 10 min | > 30 min breach → case SLA violation |
| API p99 Response Time | < 200 ms | > 1000 ms sustained → service degradation |
| Incident P1 MTTR | < 4 hours | > 8 hours → SLA breach |
| Data Ingestion Freshness | < 2 min lag | > 10 min lag → data SLA breach |
| Customer Health Score | ≥ 70 / 100 | < 50 → churn-risk escalation |

---

## 4. RACI Matrix

| Activity | Service Level Manager | CSM | Platform Eng | KAI Agents |
|---|---|---|---|---|
| Define SLIs and SLOs | **A** | C | C | I |
| Publish SLA evidence reports | **A** | C | I | **R** |
| Monitor health score trends | C | **A** | I | **R** |
| Trigger SLA breach notifications | **A** | R | I | **R** |
| Investigate SLA breach root cause | C | **A** | **R** | R |
| Calculate HLE and invoice | **A** | C | I | **R** |
| Run churn prediction model | C | **A** | I | **R** |
| Review HIBP credential health | C | **A** | C | **R** |
| Quarterly SLM review | **A** | R | C | I |

---

## 5. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Direction | Description |
|---|---|---|
| `kubric.{tenant}.agent.status.v1` | Subscribe | Watchdog agent status drives fleet availability SLI |
| `kubric.{tenant}.metrics.host.v1` | Subscribe | PerfTrace metrics feed performance SLIs |
| `kubric.{tenant}.sla.health.v1` | Publish | Sentinel health score events per tenant |
| `kubric.{tenant}.sla.breach.v1` | Publish | SLA breach event: sli_name, threshold, observed_value |
| `kubric.{tenant}.billing.invoice.v1` | Publish | Monthly invoice generation trigger |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/sla/health` | GET | Current tenant health score (0–100) |
| `/api/v1/sla/report` | GET | Monthly SLA compliance report (ClickHouse backed) |
| `/api/v1/sla/breaches` | GET | List of SLA breach events with evidence |
| `/api/v1/sla/availability` | GET | Agent fleet availability percentage (rolling 30 days) |
| `/api/v1/sla/billing` | GET | Current billing period usage and HLE calculation |
| `/api/v1/sla/invoice/{period}` | GET | Rendered invoice for a billing period (PDF/JSON) |
| `/api/v1/sla/churn-risk` | GET | Churn risk score and contributing factors |
| `/api/v1/sla/hibp-score` | GET | HIBP credential breach health score |

---

## 6. SLA Measurement Architecture

```
Data Sources
  ├── Watchdog: agent.status.v1 → fleet availability SLI
  ├── PerfTrace: metrics.host.v1 → performance SLIs
  ├── KAI Triage: incident.triage.v1 → MTTD, MTTT SLIs
  └── FastAPI: request logs → API latency SLI
        |
        v
KAI Sentinel (K-KAI-SEN-001)
  ├── Health Score calculation (weighted SLI composite)
  ├── Churn Risk Model (K-KAI-SEN-002)
  └── HIBP Credential Score (K-KAI-SEN-003)
        |
  Publish: kubric.{tenant}.sla.health.v1
        |
        v
KAI Bill Agent
  ├── ClickHouse audit (K-KAI-BL-002): evidence queries
  ├── HLE calculation (K-KAI-BL-003): hosting liability estimate
  └── Invoice renderer (K-KAI-BL-004): PDF/JSON invoice
        |
  Publish: kubric.{tenant}.billing.invoice.v1
        |
        v
Grafana SLO dashboards
  ├── Error budget burn rate
  ├── SLI trend charts
  └── Fleet availability heatmap

SLA Breach → KAI Comm (Twilio SMS) → Service Level Manager
```

---

## 7. KPIs and Metrics

| KPI | Target | Measurement |
|---|---|---|
| Composite Health Score (p50 across tenants) | ≥ 75 | KAI Sentinel publisher |
| Monthly SLA Compliance Rate | ≥ 99.5% | ClickHouse SLA audit |
| Churn Risk Score (high-risk tenants) | < 5% of fleet | KAI Sentinel churn model |
| HIBP Breach Score (0 = clean) | ≤ 0.05 | HIBP credential score |
| Invoice Generation Success | 100% by 1st of month | KAI Bill invoice renderer |
| SLA Breach Response Time | < 4h acknowledgement | SLA breach NATS event → Comm agent |
| Error Budget Remaining (fleet availability) | > 20% at mid-month | Grafana SLO dashboard |

---

## 8. Integration Points

| System | SLM Role |
|---|---|
| **ClickHouse** | Primary SLA evidence store; queried by KAI Bill for audit data |
| **Grafana** | SLO dashboards, error budget burn, fleet availability charts |
| **Prometheus** | Raw SLI data from PerfTrace and FastAPI |
| **HIBP API** | Credential breach health checks via `K-KAI-SEN-003` |
| **Stripe** | Payment processing integrated with invoice renderer |
| **Twilio** | SLA breach notifications to service level managers |
| **Watchdog** | Primary source of fleet availability measurement |

---

## 9. References

- ITIL 4 Foundation: *Service Level Management* (SMP7)
- Google SRE: Site Reliability Engineering, Chapter 4 (SLOs)
- Have I Been Pwned API: https://haveibeenpwned.com/API/v3
- Kubric Sentinel: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/`
- Kubric Bill: `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/`
