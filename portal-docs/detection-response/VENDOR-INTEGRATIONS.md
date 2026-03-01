# Kubric Vendor Detection Asset Integrations

> **Consolidated from:** `00_K-VENDOR-00_DETECTION_ASSETS/` (17 vendor subdirectories, 75 files)
> **Source:** `archive/spec-modules/00_K-VENDOR-00_DETECTION_ASSETS/`

---

## Overview

Kubric integrates 17 open-source detection asset sources for threat detection, vulnerability scanning, compliance auditing, and forensic investigation. Assets are **never bundled in git** — they are synced at setup time via `scripts/sync-vendor-assets.sh` into `vendor/` directories.

### Integration Strategy

| Strategy | Description | Vendors |
|----------|-------------|---------|
| **Direct Import** | Source code linked (Apache-2.0 / MIT / BSD) | Sigma, YARA, Falco, Zeek, Nuclei, MISP, MITRE, osquery, OpenSCAP, OSCAL |
| **API Only** | Process isolation, REST/gRPC boundary (AGPL/GPL) | Wazuh, TheHive, Cortex, Velociraptor, Shuffle, BloodHound |
| **Subprocess** | CLI invocation, output parsing | Rudder, OpenSCAP (`oscap` CLI) |

---

## Vendor Matrix

| Vendor | License | Integration | Consumers | Key Capability |
|--------|---------|-------------|-----------|----------------|
| **Sigma** (SigmaHQ) | Apache-2.0 | Direct — Rust `SigmaEngine`, hot-reload YAML | CoreSec, KAI-Triage | Log-based detection rules (~3,000 rules) |
| **YARA** (Yara-Rules) | BSD/Apache-2.0 | Direct — `yara-x` Rust crate | NetGuard, CoreSec, VDR | File/memory pattern matching, PII/DLP |
| **Suricata** (ET Open) | GPL-2.0 (data) | Direct — rule files | NetGuard IDS engine | Network intrusion detection |
| **Falco** (CNCF) | Apache-2.0 | Direct | PerfTrace, KAI-Triage | Runtime syscall/K8s audit detection |
| **Zeek** (CNCF) | BSD-3-Clause | Direct — spawned process, JSON tailing | NetGuard | Protocol analysis, intel framework |
| **Nuclei** (ProjectDiscovery) | MIT | Subprocess — CLI | VDR | Vulnerability scanning (~8,000 templates) |
| **MISP** (CIRCL) | CC0 | Direct — JSON data files | VDR, KAI-Hunter, NetGuard | Threat intel taxonomies, galaxies, warninglists |
| **MITRE** ATT&CK/CWE/CAPEC | CC BY 4.0 | Direct — STIX 2.1 JSON | VDR, KAI-Risk, NOC | ~800 techniques, ~1,400 CWEs, kill-chain mapping |
| **osquery** (LF/Meta) | Apache-2.0 | Direct — query packs | CoreSec, KAI-Hunter | SQL-queryable endpoint telemetry |
| **OpenSCAP** (NIST/RH) | LGPL-2.1 + PD | Subprocess — `oscap` CLI | CoreSec, KAI-Risk | CIS benchmarks, DISA STIGs |
| **OSCAL** (NIST) | Public Domain | Direct — JSON catalogs | KAI-Risk, K-SVC, NOC | NIST 800-53, PCI DSS, ISO 27001, SOC 2 machine-readable controls |
| **Wazuh** | GPL-2.0 | API + syslog only | CoreSec, KAI-Triage | HIDS, log analysis, FIM, SCA |
| **TheHive** (StrangeBee) | AGPL-3.0 | REST API only | KAI-Triage, KAI-Keeper | Case management, alert dedup |
| **Cortex** (StrangeBee) | AGPL-3.0 | REST API only | KAI-Triage, KAI-Analyst | Observable analysis (VT, AbuseIPDB, Shodan) + responders |
| **Velociraptor** (Rapid7) | AGPL-3.0 | HTTP/subprocess only | CoreSec, KAI-Hunter | VQL threat hunting, forensic collection |
| **Shuffle** (SOAR) | AGPL-3.0 | REST API only | KAI-Keeper, KAI-Comm | SOAR workflows (phishing, isolation, escalation) |
| **BloodHound** CE (SpecterOps) | Apache-2.0 | REST API + Cypher | KAI-Hunter, KAI-Risk | AD/Azure attack path analysis |

---

## Detection Categories

### Endpoint Detection (EDR)
- **Sigma**: Windows (Sysmon, PowerShell, scheduled tasks), Cloud (AWS/Azure/GCP audit), SaaS (M365, Okta, GitHub), Hunting (LOL, anomalous auth, beaconing)
- **Wazuh**: Process monitoring (Sysmon 61600-61699, PowerShell 91800-91899, auditd 80700-80799), AD rules (brute force, Kerberos, DCSync), SCA policies (CIS, PCI-DSS, HIPAA)
- **Falco**: System rules (shell-in-container, privilege escalation, credential access), K8s audit rules (ServiceAccount creation, attach/exec, ConfigMap sensitive data)
- **osquery**: IR packs (processes, network, startup items, browser extensions), FIM packs (Linux/Windows/macOS critical paths)
- **YARA**: Malware signatures (ransomware, RATs, rootkits), PII/DLP rules (credit cards, SSNs, API keys)

### Network Detection (NDR)
- **Suricata**: Emerging malware, C2 beaconing (JA3/JA3S), web attacks (SQLi, XSS, RCE), data exfiltration
- **Zeek**: Base protocols (conn, dns, http, ssl, smtp, files, notice), Intel framework (IOC matching from MISP), HTTP scripts (webshell, exe downloads), JA3/JA4 TLS fingerprinting

### Threat Intelligence
- **MISP**: Taxonomies (TLP, kill-chain, VERIS), Galaxies (threat-actor, mitre-attack-pattern, ransomware), Warninglists (cloud/CDN FP suppression), Objects (file, ip-port, vulnerability)
- **MITRE**: ATT&CK Enterprise STIX (800 techniques, 140 groups), CWE STIX (CVE→CWE→technique bridge), CAPEC STIX (attack pattern linking)

### Attack Path Analysis
- **BloodHound**: Windows AD Cypher queries (DA paths, Kerberoasting, DCSync, GPO abuse), Azure Cypher queries (Global Admin paths, Key Vault access, cross-tenant trust)

### Vulnerability Scanning
- **Nuclei**: CVE templates (with EPSS enrichment), Cloud misconfig (AWS ~200, Azure ~100, GCP ~80, K8s ~60), HTTP/API security, SaaS templates

### Compliance & Hardening
- **OpenSCAP**: CIS Benchmarks (Ubuntu, RHEL, Windows, K8s), DISA STIGs (CAT I/II/III → P1/P2/P3)
- **OSCAL**: NIST 800-53 Rev 5 (1,189 controls), PCI DSS v4.0, ISO 27001:2022 Annex A, SOC 2 Type II TSC

### SOAR & Response
- **Cortex**: Analyzers (VirusTotal, AbuseIPDB, Shodan, MISP), Responders (Wazuh active response, Velociraptor collection, DNS sinkhole, firewall block)
- **Shuffle**: Workflows (Phishing Response, Endpoint Isolation, Malware Containment, Brute Force Mitigation)
- **TheHive**: Case schema (custom fields: kubric_tenant, kiss_score, hunt_id), Alert dedup (type+source+sourceRef)
- **Velociraptor**: Threat hunting (ProcessTree, LateralMovement, LSASS), Forensic collection (KapeFiles, MFT, EventLogs, memory acquisition)

### Config Management
- **Rudder** (Normation): GPL-3.0 server (API-only), Apache-2.0 agent (direct deploy). Techniques: SSH hardening, firewall baseline, NTP sync, log forwarding, user account policy

---

## License Compliance Summary

| Risk Level | License | Rule | Vendors |
|------------|---------|------|---------|
| **Safe** | Apache-2.0, MIT, BSD, CC0, PD | Direct import OK | Sigma, YARA, Falco, Zeek, Nuclei, MISP, MITRE, osquery, OSCAL, BloodHound CE |
| **Caution** | LGPL-2.1 | Dynamic link or subprocess only | OpenSCAP |
| **Restricted** | GPL-2.0 | Data files OK, no source import | Suricata (ET Open rules), Wazuh, osquery (dual) |
| **Isolated** | AGPL-3.0 | API boundary, process isolation, no source integration | TheHive, Cortex, Velociraptor, Shuffle |
| **Dual** | GPL-3.0 server / Apache-2.0 agent | Server API-only, agent direct OK | Rudder |

---

## Asset Sync

All vendor assets are fetched at setup time, not stored in git:

```bash
# Full sync
./scripts/sync-vendor-assets.sh

# Individual vendor sync scripts
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-006_sync_script.sh
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-005_sync_script.sh
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-006_sync_script.sh
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-006_sync_script.sh
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-004_sync_script.sh
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-006_sync_script.sh
./00_K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-004_sync_script.sh
```

---

## Severity Mapping

All vendor detection outputs normalize to OCSF severity levels:

| OCSF Severity | Sigma `level` | Wazuh `rule.level` | Falco `priority` | Suricata `severity` |
|---------------|---------------|---------------------|-------------------|---------------------|
| 5 — Critical | critical | 13-15 | Emergency/Alert | 1 |
| 4 — High | high | 10-12 | Critical/Error | 1-2 |
| 3 — Medium | medium | 7-9 | Warning | 2-3 |
| 2 — Low | low | 4-6 | Notice | 3 |
| 1 — Info | informational | 1-3 | Informational/Debug | 4 |

---

*Full vendor spec files preserved in `archive/spec-modules/00_K-VENDOR-00_DETECTION_ASSETS/`*
