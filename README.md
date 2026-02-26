# Kubric-UiDR

**Unified Detection and Response (UiDR) Platform**

Kubric is a self-hosted, open-architecture security operations platform that combines EDR, NDR, ITDR, CDR, GRC, and AI-driven orchestration into a single deployable stack.

---

## Repository Layout

| Directory | Module | Description |
|-----------|--------|-------------|
| `00_K-VENDOR-00_DETECTION_ASSETS` | VENDOR | Third-party detection asset integrations (Sigma, Wazuh, Suricata, YARA, Falco, Zeek, MITRE ATT&CK, …) |
| `01_K-CORE-01_INFRASTRUCTURE` | CORE | Bare-metal / Proxmox cluster, Ceph storage, networking, PKI, and message bus (NATS) |
| `02_K-XRO-02_SUPER_AGENT` | XRO | CoreSec eBPF agent, NetGuard 10 G sensor, and sidecar modules |
| `03_K-KAI-03_ORCHESTRATION` | KAI | AI orchestration layer and model-tiering strategy |
| `04_K-SOC-04_SECURITY` | SOC | Detection, identity threat, incident stitching, threat intel, and vulnerability management |
| `05_K-NOC-05_OPERATIONS` | NOC | Backup/DR, config management, MDM, and performance monitoring |
| `06_K-PSA-06_BUSINESS` | PSA | Billing, business intelligence, ITSM, and client portal |
| `07_K-GRC-07_COMPLIANCE` | GRC | OSCAL-backed frameworks (NIST 800-53, PCI-DSS, ISO 27001, SOC 2), evidence vault, and SBOM |
| `08_K-DEV-08_DEVELOPMENT` | DEV | CI/CD, GitOps, and developer documentation |
| `09_K-API-09_API_REFERENCE` | API | NATS subject hierarchy and API reference |
| `10_K-ITIL-10_ITIL_MATRIX` | ITIL | ITIL 4 practice mapping and audit-readiness evidence |
| `11_K-MAP-11_DR_MODULE_MAPPING` | MAP | Master index mapping all DR modules (EDR, NDR, CDR, …) |
| `12_K-DEPLOY-12_TOPOLOGIES` | DEPLOY | Deployment topology guides |

### Code

| Path | Language | Purpose |
|------|----------|---------|
| `cmd/validate` | Go | Validates upstream intelligence XML/YAML assets |
| `cmd/intel-normalize` | Go | Normalises upstream intelligence into canonical NDJSON |
| `cmd/nats-clickhouse-bridge` | Go | Bridges NATS JetStream events into ClickHouse |
| `internal/intel` | Go | Shared canonical intelligence types |
| `deployments/` | Terraform / Helm / Ansible | Infrastructure-as-code for all topologies |
| `config/` | YAML | Runtime configuration |
| `scripts/` | Bash | Operational helper scripts |
| `third_party/intelligence/` | XML / YAML | Upstream detection content (Wazuh decoders, Sigma rules, …) |

---

## Quick Start

```bash
# Bootstrap developer toolchain (Ubuntu / Codespaces)
# See DEVELOPER-BOOTSTRAP.md for full instructions

make dev          # Start local docker-compose stack
make build        # Build all Go binaries
make test         # Run all tests
make lint         # Run linters
```

---

## Key Technologies

- **Runtime**: Kubernetes (self-hosted on Proxmox + Ceph)
- **Message bus**: NATS JetStream (mTLS)
- **Storage**: ClickHouse (events), PostgreSQL (state), VictoriaMetrics (metrics)
- **Agents**: Rust (CoreSec eBPF, NetGuard)
- **Orchestration**: Go + KAI AI layer
- **Detection**: Sigma, Wazuh, Suricata, YARA, Falco, Zeek, Tetragon
- **Compliance**: OSCAL, SBOM, OpenSSF Scorecard

---

## Documentation

- [`DEVELOPER-BOOTSTRAP.md`](DEVELOPER-BOOTSTRAP.md) — toolchain setup
- [`08_K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/`](08_K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/) — architecture and module docs
- [`11_K-MAP-11_DR_MODULE_MAPPING/`](11_K-MAP-11_DR_MODULE_MAPPING/) — DR module master index
