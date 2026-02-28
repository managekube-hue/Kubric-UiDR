# K-VENDOR-OSC-003 -- PCI DSS v4.0 OSCAL Profile

## Overview

PCI DSS (Payment Card Industry Data Security Standard) v4.0 defines security requirements for organizations that handle cardholder data. Kubric maps PCI DSS requirements to OSCAL controls and Kubric detection capabilities to provide continuous compliance visibility.

## PCI DSS Requirements Mapped to Kubric

| Req | Title | Kubric Evidence Source |
|---|---|---|
| **1** | Network Security Controls | NetGuard flow monitoring, Suricata IDS |
| **2** | Secure Configurations | Watchdog manifest enforcement, OpenSCAP CIS |
| **3** | Protect Stored Account Data | YARA PII rules (YAR-003), CoreSec FIM |
| **4** | Encrypt Transmission | NetGuard TLS version tracking, cert validation |
| **5** | Malware Protection | YARA malware rules (YAR-002), Suricata ET rules |
| **6** | Secure Development | Nuclei CVE scans (NUC-002), VDR EPSS scoring |
| **7** | Restrict Access | K-SVC RBAC, Authentik SSO/MFA |
| **8** | Identify & Authenticate | Authentik identity provider, K-SVC JWT |
| **9** | Physical Access | Manual attestation (out of scope for software) |
| **10** | Log & Monitor | CoreSec OCSF logging, KAI alert pipeline |
| **11** | Test Security | Nuclei scheduled scans, NetGuard IDS |
| **12** | Organizational Policies | Manual attestation + KAI-Risk reports |

## Data Structure

The PCI DSS OSCAL profile references NIST 800-53 controls and adds PCI-specific parameter values:

```json
{
  "profile": {
    "metadata": { "title": "PCI DSS v4.0" },
    "imports": [{ "href": "nist-800-53-rev5-catalog.json" }],
    "modify": {
      "set-parameters": [
        { "param-id": "ac-7_prm_1", "values": ["6 attempts"] }
      ]
    }
  }
}
```

## Kubric Integration

K-SVC generates PCI DSS compliance reports via `/api/v1/compliance/pci-dss`. Each requirement shows:

- **Status**: Met / Partially Met / Not Met / Not Applicable
- **Evidence**: Links to detection events, scan results, or configuration snapshots
- **Last assessed**: Timestamp of the most recent automated check

KAI-Risk includes PCI DSS scope in loss magnitude estimates for tenants processing cardholder data.
