# K-VENDOR-SUR-001 -- Suricata ET Open Rules Index

## Overview

Suricata is an open-source network IDS/IPS engine. Kubric vendors the **Emerging Threats (ET) Open** ruleset, which provides signature-based detection for malware, C2 traffic, web attacks, and data exfiltration across network flows.

## Source

- **Repository**: Proofpoint ET Open rules (via `rules.emergingthreats.net`)
- **Format**: `.rules` files (Snort/Suricata rule syntax)
- **License**: GPL 2.0 -- rules are loaded as data files, not compiled into Kubric binaries
- **Sync script**: `scripts/vendor-pull.sh suricata`

## Kubric Integration

| Consumer | Integration Method | Notes |
|---|---|---|
| **NetGuard** (Rust agent) | Parsed as data files at startup | Rules are loaded from `vendor/suricata/` and matched against captured pcap payloads |
| **KAI-Hunter** (Python) | References rule SIDs in threat hunt context | Correlates IDS alerts with MITRE ATT&CK techniques |
| **VDR** (Go service) | Indexes rule metadata | Maps Suricata SIDs to CVEs for vulnerability correlation |

## License Boundary

Suricata ET Open rules are GPL 2.0 licensed data files. Kubric reads them as configuration/data at runtime -- no GPL source is linked into or compiled with Kubric proprietary code. This preserves the license boundary.

## Document Map

| Doc ID | Title |
|---|---|
| SUR-002 | Emerging Malware Rules |
| SUR-003 | Emerging C2 Rules |
| SUR-004 | Emerging Web Attack Rules |
| SUR-005 | Emerging Data Exfiltration Rules |
