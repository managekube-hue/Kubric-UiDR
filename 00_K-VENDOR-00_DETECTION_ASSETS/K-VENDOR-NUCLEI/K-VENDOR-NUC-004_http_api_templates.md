# K-VENDOR-NUC-004 -- HTTP & API Security Templates

## Overview

HTTP/API templates detect security issues in web applications and REST APIs. These templates check for authentication bypasses, information disclosure, insecure headers, exposed debug endpoints, and API-specific vulnerabilities.

## Template Categories

| Category | Description | Example Checks |
|---|---|---|
| **Auth bypass** | Missing or weak authentication | Open admin panels, default credentials, JWT misconfig |
| **Info disclosure** | Sensitive data in responses | Stack traces, `.env` files, `.git/config` exposure |
| **Header security** | Missing security headers | No CSP, missing HSTS, permissive CORS |
| **API security** | REST/GraphQL vulnerabilities | Introspection enabled, rate limit bypass, BOLA/IDOR |
| **Debug endpoints** | Exposed development tools | phpinfo, Actuator, Swagger UI, debug toolbar |
| **Tech detection** | Technology fingerprinting | Server version, framework detection, CMS version |

## Template Structure

```yaml
id: git-config-exposure
info:
  name: Git Config File Exposure
  author: projectdiscovery
  severity: medium
  description: Exposed .git/config may leak repository URLs and credentials
  tags: exposure,git,config

http:
  - method: GET
    path:
      - "{{BaseURL}}/.git/config"
    matchers-condition: and
    matchers:
      - type: word
        words:
          - "[core]"
      - type: status
        status:
          - 200
```

## Kubric Integration

VDR runs HTTP/API templates against tenant web assets. These scans complement the Suricata web attack rules (SUR-004) -- Suricata detects inbound attacks, while Nuclei proactively identifies exploitable weaknesses.

The scan pipeline prioritizes templates based on the tenant's technology stack:
- **WordPress sites** --> CMS-specific templates + plugin vulnerability checks
- **API-first services** --> GraphQL introspection, OpenAPI validation, BOLA testing
- **Legacy apps** --> Default credential checks, EOL framework detection

Results feed into VDR's vulnerability prioritization engine, which combines Nuclei severity with EPSS probability and CVSS impact for risk-ranked remediation lists.

## MITRE ATT&CK Mapping

- T1190 -- Exploit Public-Facing Application
- T1592 -- Gather Victim Host Information
- T1589 -- Gather Victim Identity Information
