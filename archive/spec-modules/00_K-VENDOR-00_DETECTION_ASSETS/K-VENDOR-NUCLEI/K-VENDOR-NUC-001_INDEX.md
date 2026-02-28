# K-VENDOR-NUC-001 -- Nuclei Templates Index

## Overview

Nuclei is a fast, template-based vulnerability scanner by ProjectDiscovery. Kubric vendors the community template repository and invokes the Nuclei binary via subprocess to perform active vulnerability scanning against tenant-scoped targets.

## Source

- **Repository**: `https://github.com/projectdiscovery/nuclei-templates.git`
- **Format**: YAML template files
- **License**: MIT (allows direct integration)
- **Sync script**: `scripts/vendor-pull.sh nuclei`
- **Approximate count**: ~8,000 templates

## Kubric Integration

| Consumer | Integration Method | Notes |
|---|---|---|
| **VDR** (Go service) | Subprocess execution of `nuclei` binary | Runs scheduled scans; parses JSON output |
| **KAI-Hunter** (Python) | Triggers targeted scans via VDR API | Uses specific template IDs during threat hunts |
| **KAI-Analyst** (Python) | References scan results in investigation reports | Confirms exploitability of detected CVEs |
| **NOC Dashboard** | Displays scan results per tenant | Filterable by severity, template category, target |

## License Notes

Nuclei templates are MIT-licensed. The Nuclei binary itself is also MIT-licensed. Both can be directly integrated without copyleft restrictions. Kubric invokes Nuclei as a subprocess for operational isolation.

## Template Categories

| Doc ID | Category |
|---|---|
| NUC-002 | CVE-specific vulnerability templates |
| NUC-003 | Cloud misconfiguration templates |
| NUC-004 | HTTP/API security templates |
| NUC-005 | SaaS application templates |
