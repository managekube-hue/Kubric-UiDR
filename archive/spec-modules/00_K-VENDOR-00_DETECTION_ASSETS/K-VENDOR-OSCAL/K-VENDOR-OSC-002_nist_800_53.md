# K-VENDOR-OSC-002 -- NIST 800-53 Rev 5 OSCAL Catalog

## Overview

NIST Special Publication 800-53 Rev 5 defines security and privacy controls for federal information systems. The OSCAL catalog provides all 1,189 controls in machine-readable JSON, enabling automated gap analysis and continuous compliance monitoring.

## Data Structure

```json
{
  "catalog": {
    "uuid": "...",
    "metadata": { "title": "NIST SP 800-53 Rev 5" },
    "groups": [
      {
        "id": "ac",
        "title": "Access Control",
        "controls": [
          {
            "id": "ac-1",
            "title": "Policy and Procedures",
            "parts": [{ "name": "statement", "prose": "..." }],
            "params": [{ "id": "ac-1_prm_1", "label": "organization-defined..." }]
          }
        ]
      }
    ]
  }
}
```

## Key Control Families Mapped to Kubric

| Family | ID | Kubric Component Providing Evidence |
|---|---|---|
| **Access Control** | AC | K-SVC JWT/RBAC enforcement, Authentik SSO |
| **Audit & Accountability** | AU | CoreSec OCSF event logging, NATS audit trail |
| **Configuration Management** | CM | Watchdog manifest enforcement, ArgoCD GitOps |
| **Incident Response** | IR | KAI-Analyst automated investigation |
| **Risk Assessment** | RA | KAI-Risk FAIR quantification, VDR EPSS scoring |
| **System & Info Integrity** | SI | CoreSec FIM, NetGuard IDS, YARA scanning |
| **System & Comms Protection** | SC | NetGuard TLS inspection, Vault secret management |

## Kubric Integration

K-SVC exposes a `/api/v1/compliance/nist-800-53` endpoint that returns the current control implementation status per tenant. For each control:

1. **Automated evidence** -- Kubric maps detection events, configuration checks, and agent telemetry to specific controls
2. **Gap identification** -- Controls without automated evidence are flagged as requiring manual attestation
3. **Continuous monitoring** -- Control status updates in real-time as new events flow through NATS

KAI-Risk references control gaps when calculating residual risk in FAIR assessments.
