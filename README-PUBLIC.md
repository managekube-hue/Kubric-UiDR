# Kubric - Unified Detection and Response Platform

**Open-source, self-hosted security operations platform combining EDR, NDR, ITDR, CDR, VDR, and AI-driven orchestration.**

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-compose-blue.svg)](docker-compose.yml)
[![Version](https://img.shields.io/badge/version-1.0.0--rc1-green.svg)](https://github.com/managekube-hue/Kubric-UiDR/releases)

---

## What is Kubric?

Kubric is a **complete security operations platform** that unifies:

- 🛡️ **SOC** - Security Operations (EDR, NDR, ITDR, CDR, VDR)
- 🔧 **NOC** - Network Operations (infrastructure monitoring)
- 💼 **PSA** - Professional Services (ITSM, billing, portal)
- ✅ **GRC** - Governance, Risk, Compliance (200 frameworks)
- 🤖 **KAI** - AI Orchestration (13 personas, auto-remediation)

**Single platform. Single deployment. Complete coverage.**

---

## Quick Start

```bash
# Clone repository
git clone https://github.com/managekube-hue/Kubric-UiDR.git
cd Kubric-UiDR

# Configure
cp .env.example .env
nano .env

# Start infrastructure
docker compose up -d

# Access Grafana
open http://localhost:3000
```

**Full guide:** [QUICKSTART.md](QUICKSTART.md)

---

## Features

### Detection & Response
- **EDR** - eBPF/ETW process monitoring, Sigma/YARA detection
- **NDR** - 10G packet capture, DPI, Suricata IDS, TLS inspection
- **ITDR** - BloodHound AD graph, identity threat detection
- **CDR** - Container runtime security (Falco)
- **VDR** - Vulnerability scanning (Nuclei, Grype)

### AI Orchestration
- **13 KAI Personas** - TRIAGE, SENTINEL, KEEPER, COMM, ANALYST, HUNTER, etc.
- **Auto-remediation** - Temporal workflows for patching and isolation
- **Threat Intelligence** - NVD, CISA KEV, EPSS, OTX, AbuseIPDB, MISP

### Compliance
- **200 Frameworks** - NIST 800-53, PCI-DSS, ISO 27001, SOC 2, HIPAA, etc.
- **OSCAL Support** - Machine-readable compliance catalogs
- **Evidence Vault** - Immutable audit trails with BLAKE3 signatures

### ITSM
- **ERPNext** - Customer portal, ticketing, billing, asset management
- **n8n** - Workflow automation (email, Slack, Teams)
- **Billing** - Usage-based metering and invoicing

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Agents (Rust)                                  │
│  ├─ CoreSec (EDR)                              │
│  ├─ NetGuard (NDR)                             │
│  ├─ PerfTrace (Metrics)                        │
│  └─ Watchdog (OTA Updates)                     │
└─────────────────┬───────────────────────────────┘
                  │ OCSF Events
                  ↓
┌─────────────────────────────────────────────────┐
│  Message Bus (NATS JetStream)                   │
└─────────────────┬───────────────────────────────┘
                  │
        ┌─────────┴─────────┐
        ↓                   ↓
┌───────────────┐   ┌───────────────┐
│  ClickHouse   │   │  KAI AI       │
│  (Events)     │   │  (13 Personas)│
└───────────────┘   └───────────────┘
        │                   │
        ↓                   ↓
┌───────────────┐   ┌───────────────┐
│  Grafana      │   │  ERPNext      │
│  (Dashboards) │   │  (ITSM)       │
└───────────────┘   └───────────────┘
```

**Full architecture:** [portal-docs/architecture/ARCHITECTURE.md](portal-docs/architecture/ARCHITECTURE.md)

---

## Documentation

### Getting Started
- [Quick Start Guide](QUICKSTART.md) - 5-minute setup
- [Deployment Guide](portal-docs/getting-started/DEPLOYMENT.md) - Full installation

### By Role
- **Security Analyst** → [SOC Documentation](portal-docs/soc/)
- **Infrastructure Engineer** → [NOC Documentation](portal-docs/noc/)
- **Compliance Officer** → [GRC Documentation](portal-docs/compliance/)
- **Developer** → [API Reference](portal-docs/api-reference/)

### Complete Index
📖 **[Documentation Index](portal-docs/INDEX.md)** - All 48 documentation files

---

## Requirements

### Minimum
- **Docker Desktop** 24+ with Compose V2
- **RAM:** 16 GB
- **CPU:** 4 cores
- **Disk:** 100 GB

### Recommended (Production)
- **RAM:** 32 GB
- **CPU:** 8 cores
- **Disk:** 500 GB SSD
- **Network:** 10 Gbps (for NetGuard)

---

## Technology Stack

| Layer | Technology |
|-------|-----------|
| **Agents** | Rust (tokio, aya, yara-x) |
| **APIs** | Go (chi, pgx, NATS) |
| **AI** | Python (CrewAI, Ollama, Temporal) |
| **Frontend** | Next.js 14, Tailwind CSS |
| **Message Bus** | NATS JetStream |
| **Storage** | ClickHouse, PostgreSQL, Neo4j |
| **Observability** | Prometheus, Loki, Grafana |
| **ITSM** | ERPNext, n8n |

---

## Deployment Options

### Docker Compose (Recommended)
```bash
# Development
docker compose up -d

# Production
docker compose -f docker-compose.prod.yml up -d
```

### Proxmox + Ceph (Enterprise)
- Bare-metal cluster deployment
- Ceph distributed storage
- High availability
- See: [Deployment Topologies](portal-docs/getting-started/DEPLOYMENT.md)

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup
```bash
# Install dependencies
make dev-setup

# Build all services
make build

# Run tests
make test

# Run linters
make lint
```

---

## Roadmap

- [x] Core platform (EDR, NDR, ITDR, CDR, VDR)
- [x] AI orchestration (13 KAI personas)
- [x] 200 compliance frameworks
- [x] ERPNext ITSM integration
- [ ] Mobile app (iOS/Android)
- [ ] Kubernetes operator
- [ ] SaaS offering
- [ ] Marketplace for detection rules

---

## Community

- **GitHub Discussions:** https://github.com/managekube-hue/Kubric-UiDR/discussions
- **Issues:** https://github.com/managekube-hue/Kubric-UiDR/issues
- **Documentation:** [portal-docs/](portal-docs/)

---

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

---

## Acknowledgments

Built with open-source components:
- Sigma, YARA, Suricata, Wazuh, Falco, Zeek
- NATS, ClickHouse, PostgreSQL, Neo4j
- Grafana, Prometheus, Loki
- ERPNext, n8n, Temporal
- Ollama, CrewAI

---

**Made with ❤️ for the security community**
