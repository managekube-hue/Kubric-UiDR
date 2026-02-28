# K-VENDOR-MIT-004 -- CAPEC in STIX 2.1 Format

## Overview

The Common Attack Pattern Enumeration and Classification (CAPEC) catalogs known attack patterns -- the methods adversaries use to exploit weaknesses. CAPEC links CWE weaknesses to ATT&CK techniques, completing the vulnerability-to-behavior mapping chain.

## Bridging Role

```
CVE  -->  CWE (weakness)  -->  CAPEC (attack pattern)  -->  ATT&CK (technique)
```

CAPEC is the middle connector that translates "what is weak" (CWE) into "how it is attacked" (ATT&CK).

## Data Structure

```json
{
  "type": "attack-pattern",
  "id": "attack-pattern--capec-66",
  "name": "SQL Injection",
  "description": "Attacker manipulates SQL queries via user input",
  "external_references": [
    { "source_name": "capec", "external_id": "CAPEC-66" }
  ],
  "x_cwe_mapping": ["CWE-89", "CWE-564"],
  "x_mitre_mapping": ["T1190"]
}
```

## Key CAPEC Patterns Used by Kubric

| CAPEC ID | Name | CWE Link | ATT&CK Link |
|---|---|---|---|
| CAPEC-66 | SQL Injection | CWE-89 | T1190 |
| CAPEC-86 | XSS via HTTP Headers | CWE-79 | T1059.007 |
| CAPEC-126 | Path Traversal | CWE-22 | T1083 |
| CAPEC-153 | Input Data Manipulation | CWE-20 | T1565 |
| CAPEC-242 | Code Injection | CWE-94 | T1059 |
| CAPEC-560 | Execute Unauthorized Software | CWE-284 | T1203 |

## Kubric Integration

VDR uses CAPEC as the final link in its CVE enrichment chain:

1. A CVE references one or more CWE IDs (from NVD)
2. Each CWE maps to one or more CAPEC attack patterns
3. Each CAPEC maps to ATT&CK techniques via `x_mitre_mapping`
4. The resulting ATT&CK technique IDs are attached to the vulnerability record

KAI-Risk uses CAPEC attack pattern descriptions to model threat event frequency in FAIR assessments. The attack pattern's prevalence and required skill level inform the probability estimates published to `kubric.kai.risk.assessment`.

## License

CAPEC is licensed under CC BY 4.0, consistent with all MITRE datasets. Attribution is included in Kubric-generated vulnerability reports.
