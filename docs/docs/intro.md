---
title: Documentation
---

Complete technical reference for Kubric UIDR. This page sits in front of the full repository-mirrored documentation tree.

## Documentation Lanes

Each lane maps 1:1 to the repository and remains Notion-sync compatible.

| Lane | Scope | Read more |
| --- | --- | --- |
| K-CORE-01_INFRASTRUCTURE | Data, compute, networking, and security foundations | [Read more](/docs/K-CORE-01_INFRASTRUCTURE) |
| K-VENDOR-00_DETECTION_ASSETS | Upstream detection assets and vendor-integrated libraries | [Read more](/docs/K-VENDOR-00_DETECTION_ASSETS) |
| K-XRO-02_SUPER_AGENT | Core runtime agents and execution primitives | [Read more](/docs/K-XRO-02_SUPER_AGENT) |
| K-KAI-03_ORCHESTRATION | AI workflows, guardrails, and orchestration logic | [Read more](/docs/K-KAI-03_ORCHESTRATION) |
| K-SOC-04_SECURITY | Detection, forensics, incident stitching, threat intel, vulnerability | [Read more](/docs/K-SOC-04_SECURITY) |
| K-NOC-05_OPERATIONS | Operational controls, inventory, backup/DR, patching, performance | [Read more](/docs/K-NOC-05_OPERATIONS) |
| K-PSA-06_BUSINESS | ITSM, billing, CRM/CPQ, portal, and business intelligence | [Read more](/docs/K-PSA-06_BUSINESS) |
| K-GRC-07_COMPLIANCE | Framework mapping, evidence, OSCAL, and supply-chain controls | [Read more](/docs/K-GRC-07_COMPLIANCE) |
| K-DEV-08_DEVELOPMENT | Build toolchain, CI/CD, local stacks, tests, and tools | [Read more](/docs/K-DEV-08_DEVELOPMENT) |
| K-API-09_API_REFERENCE | API subjects, OpenAPI contracts, and protobuf references | [Read more](/docs/K-API-09_API_REFERENCE) |
| K-ITIL-10_ITIL_MATRIX | ITIL practice-to-module operational mapping | [Read more](/docs/K-ITIL-10_ITIL_MATRIX) |
| K-MAP-11_DR_MODULE_MAPPING | Detection & response module mapping index | [Read more](/docs/K-MAP-11_DR_MODULE_MAPPING) |
| K-DEPLOY-12_TOPOLOGIES | Small/medium/large topologies and deployment assets | [Read more](/docs/K-DEPLOY-12_TOPOLOGIES) |

## Integration Strategy

- **Direct Import** — MIT / Apache 2.0 / BSD dependencies in Go/Rust/TS packages.
- **Vendor Data** — GPL and policy/rule datasets stored as versioned assets.
- **Subprocess** — AGPL tools isolated as sidecars or controlled subprocesses.
- **REST Pull** — External API data retrieved and normalized into platform storage.
- **FFI Binding** — Native/C libraries exposed via audited bindings.
