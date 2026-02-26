# K-VENDOR-MISP-001 -- MISP Threat Intelligence Index

## Overview

MISP (Malware Information Sharing Platform) provides structured threat intelligence data. Kubric vendors four MISP community data repositories: taxonomies, galaxies, warninglists, and object templates. These are CC0-licensed JSON datasets consumed as read-only reference data.

## Sources

| Repository | Content | License |
|---|---|---|
| `MISP/misp-taxonomies` | Classification vocabularies | CC0 |
| `MISP/misp-galaxy` | Threat actor & malware clusters | CC0 |
| `MISP/misp-warninglists` | Known-good/known-bad IP/domain lists | CC0 |
| `MISP/misp-objects` | Structured object templates | CC0 |

**Sync script**: `scripts/vendor-pull.sh misp`

## Kubric Integration

| Consumer | Integration Method | Notes |
|---|---|---|
| **VDR** (Go service) | REST API pull with pagination | Fetches IOCs from MISP instances; uses taxonomies for classification |
| **KAI-Hunter** (Python) | Reads galaxy clusters for threat actor attribution | Maps IOCs to APT groups during proactive hunts |
| **KAI-Analyst** (Python) | Enriches alerts with MISP context | Adds TLP marking, threat actor info, and kill chain phase |
| **NetGuard** (Rust agent) | Warninglist lookup for IP/domain reputation | Cross-references flow destinations against known-bad lists |

## License Notes

All MISP community data repositories are CC0 (public domain). No attribution or license compliance requirements apply.

## Document Map

| Doc ID | Title |
|---|---|
| MISP-002 | Taxonomies |
| MISP-003 | Galaxies |
| MISP-004 | Warninglists |
| MISP-005 | Object Templates |
