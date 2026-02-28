# K-VENDOR-OSC-005 -- SOC 2 Type II OSCAL Mapping

## Overview

SOC 2 Type II is an auditing framework based on the AICPA Trust Services Criteria (TSC). It evaluates security, availability, processing integrity, confidentiality, and privacy controls over time. Kubric maps TSC criteria to automated evidence sources to streamline SOC 2 audits for MSSP tenants.

## Trust Services Criteria Mapping

| Category | Criteria | Kubric Evidence Source |
|---|---|---|
| **Security** | CC6.1 - Logical access controls | K-SVC RBAC, Authentik SSO/MFA |
| **Security** | CC6.6 - Boundary protection | NetGuard IDS, Suricata rules |
| **Security** | CC6.8 - Malicious software prevention | YARA scanning, Suricata malware rules |
| **Security** | CC7.1 - Detection of unauthorized changes | CoreSec FIM (OCSF FileActivity) |
| **Security** | CC7.2 - Monitoring for anomalies | KAI alert pipeline, KAI-Hunter proactive hunts |
| **Security** | CC7.3 - Security incident evaluation | KAI-Analyst automated investigation |
| **Security** | CC8.1 - Change management | ArgoCD GitOps, Watchdog manifest sync |
| **Availability** | A1.2 - Environmental protections | Infrastructure telemetry via PerfTrace agent |
| **Confidentiality** | C1.1 - Confidential information identification | YARA PII rules (YAR-003), DLP alerts |
| **Confidentiality** | C1.2 - Confidential information disposal | CoreSec FIM tracking of deletions |

## Data Structure

SOC 2 criteria are mapped as an OSCAL profile referencing NIST 800-53 controls:

```json
{
  "profile": {
    "metadata": { "title": "SOC 2 Type II - Trust Services Criteria" },
    "imports": [{ "href": "nist-800-53-rev5-catalog.json" }],
    "modify": {
      "set-parameters": [
        { "param-id": "cc7-1_prm_1", "values": ["continuous monitoring"] }
      ]
    }
  }
}
```

## Kubric Integration

K-SVC provides a `/api/v1/compliance/soc2` endpoint that returns:

- **Evidence timeline** -- Continuous log of control-relevant events over the audit period
- **Control effectiveness** -- Percentage of time each control was operating effectively
- **Exception log** -- Incidents where controls failed, with remediation timestamps

This data supports the SOC 2 Type II requirement of demonstrating control effectiveness over a period (typically 6-12 months), not just at a point in time. KAI-Risk uses SOC 2 gap data to quantify operational risk for service provider tenants.
