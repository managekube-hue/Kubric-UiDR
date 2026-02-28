# K-MAP-005 — SaaS Detection & Response (SDR)
**Discipline:** SaaS Detection & Response
**Abbreviation:** SDR
**Kubric Reference:** K-MAP-005
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

SaaS Detection and Response (SDR) monitors and responds to threats within Software-as-a-Service applications — including unauthorized access, data exfiltration via SaaS APIs, OAuth token abuse, and shadow IT. SDR requires visibility into SaaS audit logs, user behavior analytics, and integration with the organization's identity provider. Kubric implements SDR through the KAI Sentinel health score module (which incorporates credential breach data from HIBP), MISP-backed threat intelligence about SaaS-targeting adversaries, Authentik OIDC for centralized SaaS identity brokering, and the KAI Invest graph investigation engine for tracing multi-SaaS lateral movement.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| SaaS Identity Brokering | Authentik OIDC | `services/k-svc/` |
| Credential Breach Monitor | KAI Sentinel HIBP | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py` |
| Customer Health / Anomaly | KAI Sentinel health score | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py` |
| SaaS Threat Actor Intel | KAI Invest MISP galaxy | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py` |
| OAuth/Token Investigation | KAI Invest graph investigation | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py` |
| SaaS Risk Scoring | KAI Risk EPSS | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py` |
| SaaS Alert Triage | KAI Triage agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| SaaS Churn Signal | KAI Sentinel churn risk | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py` |
| API Monitoring | KAI API FastAPI | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py` |

---

## 3. Data Flow Diagram

```
SaaS Applications
│
├── Authentik: SSO audit events (OAuth grants, revocations, failed auth)
│     └── Syslog ──► ClickHouse: kubric.auth_events
│
├── Kubric API: access log (per-request JWT validation results)
│     └── PostgreSQL: kubric.api_access_log
│
├── External: HIBP credential breach data
│     └── KAI Sentinel (K-KAI-SEN-003) polls HIBP API
│          └── NATS: kubric.{tenant}.sla.health.v1
│
└── External: MISP SaaS threat intelligence
      └── KAI Invest (K-KAI-IV-001) queries MISP galaxy
           └── Threat actor TTP enrichment

                    ▼
          KAI Triage Agent
           ├── OCSF class for SaaS events
           ├── KISS priority score
           └── TheHive case (suspicious OAuth, MFA bypass)

          KAI Invest Graph
           └── Cross-correlate: SaaS auth → process → network
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Initial Access | T1078.004 | Valid Accounts: Cloud | Authentik: anomalous login (new geo/time) |
| Initial Access | T1566.002 | Spearphishing Link (OAuth consent) | KAI Invest: OAuth grant anomaly |
| Credential Access | T1528 | Steal Application Access Token | Authentik audit + Vault audit |
| Persistence | T1098.001 | Additional Cloud Credentials | Authentik: unexpected OAuth app grant |
| Defense Evasion | T1550.001 | Use Alternate Auth Material: App Access Token | KAI Invest graph |
| Exfiltration | T1567 | Exfiltration to Cloud Storage | NetGuard: egress to cloud provider IP |
| Collection | T1213.003 | Data from Cloud Storage | Authentik: bulk API access |

---

## 5. Integration Points

| System | SDR Role |
|---|---|
| **Authentik** | SSO broker for all SaaS; OAuth audit events |
| **HIBP API** | Credential breach monitoring for SaaS accounts |
| **MISP** | SaaS-targeting threat actor TTPs |
| **TheHive** | SaaS incident cases |
| **ClickHouse** | SaaS audit log retention |
| **Grafana** | SaaS adoption, anomaly, health dashboards |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| Authentik | OIDC/SAML configured for each SaaS application |
| HIBP API key | Set in Vault; `K-KAI-SEN-003` uses k-v key `hibp/api_key` |
| MISP connectivity | Network access to MISP instance from KAI namespace |
| ClickHouse | `kubric.auth_events` receiving Authentik event export |
| Sentinel schedule | Health score publishing interval: 5 min (configurable) |
