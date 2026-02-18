---
title: Documentation
---

Complete technical reference for Kubric UIDR. This page sits in front of the full repository-mirrored documentation tree.

## Documentation Lanes

Each lane maps 1:1 to the repository and remains Notion-sync compatible.

| Lane | Scope | Notion | Repository Docs |
| --- | --- | --- | --- |
| K-CORE-01_INFRASTRUCTURE | Data, compute, networking, and security foundations | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-CORE-01_INFRASTRUCTURE) |
| K-VENDOR-00_DETECTION_ASSETS | Upstream detection assets and vendor-integrated libraries | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-VENDOR-00_DETECTION_ASSETS) |
| K-XRO-02_SUPER_AGENT | Core runtime agents and execution primitives | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-XRO-02_SUPER_AGENT) |
| K-KAI-03_ORCHESTRATION | AI workflows, guardrails, and orchestration logic | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-KAI-03_ORCHESTRATION) |
| K-SOC-04_SECURITY | Detection, forensics, incident stitching, threat intel, vulnerability | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-SOC-04_SECURITY) |
| K-NOC-05_OPERATIONS | Operational controls, inventory, backup/DR, patching, performance | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-NOC-05_OPERATIONS) |
| K-PSA-06_BUSINESS | ITSM, billing, CRM/CPQ, portal, and business intelligence | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-PSA-06_BUSINESS) |
| K-GRC-07_COMPLIANCE | Framework mapping, evidence, OSCAL, and supply-chain controls | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-GRC-07_COMPLIANCE) |
| K-DEV-08_DEVELOPMENT | Build toolchain, CI/CD, local stacks, tests, and tools | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-DEV-08_DEVELOPMENT) |
| K-API-09_API_REFERENCE | API subjects, OpenAPI contracts, and protobuf references | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-API-09_API_REFERENCE) |
| K-ITIL-10_ITIL_MATRIX | ITIL practice-to-module operational mapping | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-ITIL-10_ITIL_MATRIX) |
| K-MAP-11_DR_MODULE_MAPPING | Detection & response module mapping index | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-MAP-11_DR_MODULE_MAPPING) |
| K-DEPLOY-12_TOPOLOGIES | Small/medium/large topologies and deployment assets | [Open in Notion](https://www.notion.so/kubric) | [Open docs](/docs/K-DEPLOY-12_TOPOLOGIES) |

## Integration Strategy

- **Direct Import** — MIT / Apache 2.0 / BSD dependencies in Go/Rust/TS packages.
- **Vendor Data** — GPL and policy/rule datasets stored as versioned assets.
- **Subprocess** — AGPL tools isolated as sidecars or controlled subprocesses.
- **REST Pull** — External API data retrieved and normalized into platform storage.
- **FFI Binding** — Native/C libraries exposed via audited bindings.
