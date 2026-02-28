# K-VENDOR-MIT-003 -- CWE in STIX 2.1 Format

## Overview

The Common Weakness Enumeration (CWE) is a categorized list of software and hardware weakness types. Kubric converts CWE data into STIX 2.1 format to enable graph-based linking between CVEs, weaknesses, and ATT&CK techniques.

## Purpose in Kubric

CWE serves as the **bridge between vulnerabilities and adversary behavior**:

```
CVE-2024-XXXX  --(has weakness)-->  CWE-79 (XSS)
CWE-79         --(enables)-------->  T1059.007 (JavaScript Execution)
T1059.007      --(belongs to)----->  TA0002 (Execution)
```

This three-hop chain lets Kubric automatically map any CVE to relevant ATT&CK tactics.

## Data Structure

```json
{
  "type": "weakness",
  "id": "weakness--cwe-79",
  "name": "Improper Neutralization of Input During Web Page Generation",
  "external_references": [
    { "source_name": "cwe", "external_id": "CWE-79" }
  ],
  "x_capec_mapping": ["CAPEC-86"]
}
```

## Key CWE Categories Used by Kubric

| CWE ID | Name | Related ATT&CK |
|---|---|---|
| CWE-79 | Cross-Site Scripting (XSS) | T1059.007 |
| CWE-89 | SQL Injection | T1190 |
| CWE-78 | OS Command Injection | T1059 |
| CWE-22 | Path Traversal | T1083 |
| CWE-502 | Deserialization of Untrusted Data | T1190 |
| CWE-287 | Improper Authentication | T1078 |
| CWE-798 | Hard-coded Credentials | T1552.001 |

## Kubric Integration

VDR's vulnerability enrichment pipeline works as follows:

1. **NVD/EPSS pull** -- Fetches CVE records with CWE IDs and EPSS scores
2. **CWE lookup** -- Resolves each CWE ID to its STIX object, extracting CAPEC mappings
3. **CAPEC-to-ATT&CK bridge** -- Uses CAPEC relationships (MIT-004) to link to ATT&CK techniques
4. **Risk scoring** -- KAI-Risk factors the CWE category into FAIR vulnerability estimates

This enables the NOC dashboard to display vulnerabilities organized by ATT&CK tactic, not just CVSS score.
