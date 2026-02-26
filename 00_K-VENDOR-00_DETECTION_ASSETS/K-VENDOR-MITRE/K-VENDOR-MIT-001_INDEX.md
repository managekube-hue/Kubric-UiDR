# K-VENDOR-MIT-001 -- MITRE ATT&CK / CWE / CAPEC Index

## Overview

MITRE provides three foundational knowledge bases for cybersecurity: **ATT&CK** (adversary tactics and techniques), **CWE** (Common Weakness Enumeration), and **CAPEC** (Common Attack Pattern Enumeration). Kubric vendors these as STIX 2.1 JSON bundles for offline querying and correlation.

## Sources

| Dataset | Format | License | Sync Method |
|---|---|---|---|
| **ATT&CK Enterprise** | STIX 2.1 JSON bundle | CC BY 4.0 | `curl` from `mitre/cti` GitHub |
| **ATT&CK ICS** | STIX 2.1 JSON bundle | CC BY 4.0 | `curl` from `mitre/cti` GitHub |
| **CWE** | STIX 2.1 JSON (converted from XML) | CC BY 4.0 | Converted from MITRE CWE XML |
| **CAPEC** | STIX 2.1 JSON (converted from XML) | CC BY 4.0 | Converted from MITRE CAPEC XML |

**Sync script**: `scripts/vendor-pull.sh mitre`

## Kubric Integration

| Consumer | Usage |
|---|---|
| **VDR** (Go service) | Maps CVEs to ATT&CK techniques and CWE weaknesses |
| **KAI-Hunter** (Python) | Uses ATT&CK techniques as hypothesis anchors for threat hunts |
| **KAI-Analyst** (Python) | Builds attack chains using ATT&CK tactic/technique sequences |
| **KAI-Risk** (Python) | Maps CAPEC attack patterns to FAIR threat event frequencies |
| **NOC Dashboard** | Displays ATT&CK heatmap of detected techniques per tenant |

## License Notes

All MITRE datasets are CC BY 4.0. Attribution is required when redistributing. Kubric includes the MITRE attribution notice in all generated reports and dashboard footers.

## Document Map

| Doc ID | Title |
|---|---|
| MIT-002 | Enterprise ATT&CK STIX Bundle |
| MIT-003 | CWE in STIX 2.1 Format |
| MIT-004 | CAPEC in STIX 2.1 Format |
