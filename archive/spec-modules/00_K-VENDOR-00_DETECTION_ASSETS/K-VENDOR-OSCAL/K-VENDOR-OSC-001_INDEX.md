# K-VENDOR-OSC-001 -- OSCAL Compliance Frameworks Index

## Overview

OSCAL (Open Security Controls Assessment Language) is a NIST-developed machine-readable format for expressing security control catalogs, baselines, and assessment results. Kubric vendors OSCAL content to automate compliance mapping and continuous control assessment across multiple frameworks.

## Source

- **Repository**: `https://github.com/usnistgov/oscal-content.git`
- **Format**: JSON and XML (Kubric consumes JSON)
- **License**: Public domain (US Government work)
- **Sync script**: `scripts/vendor-pull.sh oscal`

## Kubric Integration

| Consumer | Usage |
|---|---|
| **KAI-Risk** (Python) | Maps controls to FAIR risk scenarios; identifies control gaps |
| **K-SVC** (Go service) | Generates compliance posture reports per tenant |
| **NOC Dashboard** | Displays control coverage heatmaps by framework |
| **KAI-Analyst** (Python) | References control requirements in investigation reports |

## Supported Frameworks

| Doc ID | Framework | Applicability |
|---|---|---|
| OSC-002 | NIST 800-53 Rev 5 | Federal / FedRAMP tenants |
| OSC-003 | PCI DSS v4.0 | Payment card processing tenants |
| OSC-004 | ISO 27001:2022 | International / enterprise tenants |
| OSC-005 | SOC 2 Type II | SaaS / service provider tenants |

## License Notes

OSCAL content produced by NIST is public domain. Framework-specific mappings created by Kubric are proprietary but reference public control IDs.
