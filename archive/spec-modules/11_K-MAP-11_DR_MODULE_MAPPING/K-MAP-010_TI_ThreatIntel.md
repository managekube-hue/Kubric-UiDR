# K-MAP-010 — Threat Intelligence (TI)
**Discipline:** Threat Intelligence
**Abbreviation:** TI
**Kubric Reference:** K-MAP-010
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Threat Intelligence (TI) is the collection, processing, analysis, and dissemination of information about current and emerging threats to an organization's assets. TI encompasses tactical IOCs (IP addresses, domains, file hashes), operational TTPs (threat actor campaigns and procedures), and strategic intelligence (sector threat landscape). Kubric integrates TI throughout the platform: NetGuard uses the ipsum IP blocklist for real-time flow enrichment, KAI Invest queries MISP galaxy for threat actor mapping, KAI Foresight uses EPSS and LSTM for predictive intelligence, and KAI Analyst applies Cortex-based external enrichment on any observable.

---

## 2. Kubric Modules

| TI Sub-Function | Module | File Path |
|---|---|---|
| Network IOC (real-time) | NetGuard ipsum IP lookup | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs` |
| File Hash TI | CoreSec YARA (hash patterns) | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| Threat Actor Galaxy | KAI Invest MISP galaxy query | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py` |
| TI Investigation Graph | KAI Invest graph investigation | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py` |
| Predictive TI (EPSS) | KAI Foresight EPSS enrichment | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-002_epss_enrichment.py` |
| Predictive TI (LSTM) | KAI Foresight LSTM baseline | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py` |
| Observable Enrichment | KAI Analyst Cortex chain | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py` |
| Observable Enrichment | KAI Analyst enrichment | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py` |
| GeoIP Resolution | KAI Libs GeoIP2 | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-010_geoip2_resolver.py` |
| Vendor TI Pull | scripts/vendor-pull.sh | `scripts/vendor-pull.sh` |
| TI Store | ClickHouse + MISP | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py` |

---

## 3. TI Feed Coverage

| Feed | Type | Update Frequency | Kubric Consumer |
|---|---|---|---|
| ipsum IP blocklist | Tactical IOC (IP) | Daily via vendor-pull.sh | NetGuard ipsum_lookup |
| MISP instance | All TI types | Real-time / event-driven | KAI Invest MISP galaxy |
| MISP galaxy | Threat actor TTPs | Weekly galaxy update | KAI Invest galaxy query |
| EPSS v3 API | CVE exploitability score | Daily | KAI Foresight EPSS enrichment |
| YARA rules | File signatures | Via vendor-pull.sh | CoreSec YARA scanner |
| Sigma rules | Behavioral detection | Via vendor-pull.sh | CoreSec Sigma evaluator |
| Nuclei templates | Vuln scanning templates | Via vendor-pull.sh | Nuclei bridge |
| GeoIP2 MaxMind | IP geolocation | Monthly database update | KAI Libs GeoIP2 |

---

## 4. Data Flow Diagram

```
External TI Sources
│
├── ipsum blocklist ──► vendor-pull.sh ──► vendor/ipsum-blocklist
│   └── NetGuard at startup: load into memory hash set
│        └── Per-flow: src/dst IP lookup → threat_score field
│
├── MISP instance ──► KAI Invest (K-KAI-IV-001)
│   ├── Galaxy: threat actor → MITRE ATT&CK TTP map
│   └── IOC events: IP, domain, hash, URL
│
├── EPSS API ──► KAI Foresight (K-KAI-FS-002)
│   └── CVE → EPSS score → ClickHouse vuln enrichment
│
└── Sigma/YARA/Nuclei vendors ──► vendor-pull.sh
    └── Updated rule sets → CoreSec reload on next startup

Internal TI Production
│
├── CoreSec malware detections (YARA hits) → MISP feedback loop
├── NetGuard IDS alerts → MISP observable creation
└── KAI Analyst: Cortex analyzers produce TI enrichment
    └── VirusTotal, Shodan, PassiveDNS, AbuseIPDB, MISP lookup

TI Dissemination
├── NATS: kubric.{tenant}.network.ids.alert.v1 (IOC in context)
├── TheHive cases: observables with TI enrichment
├── ClickHouse: kubric.ti_iocs (enrichment cache)
└── Grafana: TI coverage dashboard
```

---

## 5. MITRE ATT&CK TI Coverage

| TI Type | ATT&CK Mapping | Kubric Source |
|---|---|---|
| IP blocklist hit | Initial Access, C2 | NetGuard ipsum lookup |
| Malware family (YARA) | T1566, T1204, T1486 | CoreSec YARA scanner |
| Threat actor galaxy | All tactics (per actor TTPs) | KAI Invest MISP galaxy |
| EPSS-scored CVE | Exploitation techniques | KAI Foresight EPSS |
| LSTM anomaly baseline | Novel attack patterns | KAI Foresight LSTM |
| Cortex VirusTotal | T1059, T1078, T1190 | KAI Analyst Cortex |
| Sigma rules | Execution, Persistence, Priv Esc, Lateral Movement | CoreSec Sigma |

---

## 6. Integration Points

| System | TI Role |
|---|---|
| **MISP** | Central TI platform; IOC storage and galaxy feeds |
| **Cortex** | Automated TI enrichment (VirusTotal, Shodan, PassiveDNS) |
| **TheHive** | TI-enriched observables attached to cases |
| **ClickHouse** | TI cache and enrichment history |
| **EPSS API** | Exploitability probability scores |
| **MaxMind GeoIP2** | IP geolocation enrichment |
| **ipsum** | Community-maintained IP threat blocklist |

---

## 7. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| MISP instance | MISP 2.4+ with galaxy module enabled; API key in Vault |
| ipsum blocklist | Run `scripts/vendor-pull.sh` before first NetGuard start |
| EPSS access | Public FIRST API; no key required |
| GeoIP2 database | MaxMind account + license for GeoLite2-City.mmdb download |
| Cortex | Analyzers enabled: MISP_2_1, VirusTotal_GetReport, Shodan_Host, PassiveDNS_Circl |
| vendor-pull.sh | Scheduled weekly; run as CI step on release |
