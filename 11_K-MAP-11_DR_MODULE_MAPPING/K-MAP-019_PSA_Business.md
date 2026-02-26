# K-MAP-019 — Professional Services Automation (PSA)
**Discipline:** Professional Services Automation
**Abbreviation:** PSA
**Kubric Reference:** K-MAP-019
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Professional Services Automation (PSA) encompasses the business management and operational software that Managed Service Providers (MSPs) and MSSPs use to run their service delivery businesses. PSA tools typically cover ticketing, time tracking, billing, contract management, resource scheduling, project management, and customer health monitoring. Kubric integrates a PSA capability layer through the KAI Bill agent (billing, HLE calculation, invoice generation), the KAI Sentinel agent (customer health scoring, churn prediction), the KAI Simulate agent (LTV prediction, churn simulation, dynamic pricing), and the KAI Comm agent (automated customer communications via VAPI and Twilio).

---

## 2. Kubric Modules

| PSA Sub-Capability | Module | File Path |
|---|---|---|
| Billing Administration | KAI Bill – billing clerk | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-001_billing_clerk.py` |
| Usage Audit (ClickHouse) | KAI Bill – ClickHouse audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |
| Hosting Liability Estimate | KAI Bill – HLE calculator | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py` |
| Invoice Generation | KAI Bill – invoice renderer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py` |
| Customer Health Scoring | KAI Sentinel – health score publisher | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py` |
| Churn Risk Prediction | KAI Sentinel – churn risk model | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py` |
| Credential Health (NPS signal) | KAI Sentinel – HIBP credential score | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py` |
| LTV Prediction | KAI Simulate – LTV predictor | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py` |
| Churn Simulation | KAI Simulate – churn simulator | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-002_churn_simulator.py` |
| Dynamic Pricing | KAI Simulate – dynamic pricing | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-003_dynamic_pricing.py` |
| Customer Communications | KAI Comm – comm agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py` |
| Voice Escalation | KAI Comm – VAPI phone | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-002_vapi_phone.py` |
| SMS Notification | KAI Comm – Twilio SMS | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py` |

---

## 3. Data Flow Diagram

```
Service Delivery Data Sources
│
├── Watchdog: agent.status.v1 → fleet availability per tenant
├── Triage: incident.triage.v1 → incident volume and severity per tenant
├── ClickHouse: all event tables → usage metrics
│
└── External:
    ├── HIBP API → credential health
    └── Stripe → payment processing
                    │
                    ▼
KAI Bill Agent (monthly cycle)
├── ClickHouse audit: query event volume by tenant
├── HLE calculator: hosting cost allocation
└── Invoice renderer: PDF/JSON invoice → Stripe charge
    └── NATS: kubric.{tenant}.billing.invoice.v1

KAI Sentinel Agent (continuous)
├── Health score: composite (availability + incident + credential)
├── Churn risk model: ML prediction of churn probability (30-day)
└── HIBP credential score: breach exposure index
    └── NATS: kubric.{tenant}.sla.health.v1

KAI Simulate Agent (on-demand / scheduled)
├── LTV predictor: predicted lifetime value per customer segment
├── Churn simulator: what-if scenario modeling
└── Dynamic pricing: usage-based pricing recommendation
    └── Grafana: PSA business intelligence dashboard

KAI Comm Agent (event-driven)
├── SLA breach → Twilio SMS to account manager
├── Churn risk high → Comm agent prepares outreach
└── P1 incident → VAPI phone call to customer on-call
```

---

## 4. PSA KPIs

| KPI | Target | Source |
|---|---|---|
| Invoice Generation SLA (1st of month) | 100% on time | KAI Bill invoice renderer |
| Customer Health Score p50 | ≥ 75/100 | KAI Sentinel health score |
| High-Churn-Risk Customers (< 50 health) | < 5% of fleet | KAI Sentinel churn model |
| 30-Day LTV Forecast Accuracy (MAPE) | < 10% | KAI Simulate LTV predictor |
| SLA Breach Notification Time | < 4h from breach | KAI Comm Twilio SMS |
| Credential Breach Score p90 | < 0.10 | KAI Sentinel HIBP |
| Dynamic Pricing Model Update Frequency | Monthly | KAI Simulate pricing |

---

## 5. Integration Points

| System | PSA Role |
|---|---|
| **ClickHouse** | Usage data source for billing and health computation |
| **Stripe** | Payment processing; invoice renderer triggers Stripe charge |
| **Twilio** | Customer SMS notifications (SLA breach, renewal, security alert) |
| **VAPI** | AI voice calls for P1 escalations and outreach |
| **HIBP API** | Credential breach signal for customer health score |
| **Grafana** | PSA executive dashboard: ARR, churn forecast, health heatmap |
| **PostgreSQL** | Customer contract and subscription data |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| ClickHouse | `kubric.event_usage_by_tenant` view or materialized table |
| Stripe | API key in Vault at `stripe/secret_key` |
| Twilio | Account SID + Auth Token in Vault at `twilio/*` |
| VAPI | API key + phone number in Vault at `vapi/*` |
| HIBP API key | Vault at `hibp/api_key` |
| Sentinel schedule | `SENTINEL_HEALTH_INTERVAL=300` (every 5 min) |
| Billing cycle | `BILLING_DAY=1` env var for monthly invoice trigger |
