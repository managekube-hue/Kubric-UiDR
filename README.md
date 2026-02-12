# Kubric Security Platform

A comprehensive, modular security orchestration and response platform built on enterprise-grade infrastructure.

## Overview

Kubric is a next-generation security operations platform designed for modern cloud-native environments. It combines:

- **K-CORE**: Distributed infrastructure (Kubernetes, PostgreSQL, ClickHouse, NATS)
- **K-XRO**: High-performance security agents (eBPF + Rust)
- **K-KAI**: AI-driven orchestration (CrewAI, Temporal)
- **K-SOC**: Security detection and correlation
- **K-NOC**: Network operations and observability
- **K-PSA**: Professional services and billing
- **K-GRC**: Compliance and governance
- **K-DEV**: Development infrastructure
- **K-API**: Unified REST and gRPC APIs

## Quick Start

### Prerequisites

- Kubernetes 1.24+
- kubectl configured
- Helm 3.10+
- Docker (for development)
- Terraform (for infrastructure)
- Ansible 2.9+

### Development Environment (Docker Compose)

```bash
make dev
# Services available:
# - PostgreSQL: localhost:5432
# - NATS: localhost:4222
# - ClickHouse: localhost:8123
# - Prometheus: localhost:9090
```

### Production Deployment (Kubernetes)

```bash
make bootstrap
# This will:
# 1. Create kubric namespace
# 2. Deploy NATS, ClickHouse, PostgreSQL
# 3. Initialize databases
# 4. Deploy API and services
```

## Project Structure

```
kubric-platform/
├── deployments/           # Infrastructure-as-Code
│   ├── k8s/              # Kubernetes manifests
│   ├── terraform/        # Terraform modules
│   ├── ansible/          # Automation playbooks
│   └── helm/             # Helm charts
├── config/               # Service configurations
├── docs/                 # Documentation
├── scripts/              # Utility scripts
├── docker-compose/       # Development environments
├── .github/workflows/    # CI/CD pipelines
└── Makefile             # Development tasks
```

## Key Features

### Security Agents (K-XRO)
- **CoreSec**: eBPF-based kernel monitoring (Rust)
- **NetGuard**: Network detection and response (Rust)
- **PerfTrace**: Performance profiling (Rust)

### AI-Driven Orchestration (K-KAI)
- **Triage Agent**: Incident analysis with CrewAI + Llama 3.1
- **Housekeeper Agent**: Automated remediation via Ansible
- **Billing Clerk**: Usage aggregation and invoicing
- **Comm Agent**: Multi-channel notifications

### Infrastructure (K-CORE)
- Hardware identity via Blake3 fingerprinting
- Centralized User Account Registry with RLS
- High-throughput event bus (NATS JetStream)
- Distributed analytics (ClickHouse)
- Secure secrets management (Vault)

## Technology Stack

| Layer | Technologies |
|-------|---|
| **Frontend** | Next.js, React, Tremor |
| **Backend** | Go, Temporal, NATS |
| **Security Agents** | Rust, eBPF, Pcap |
| **AI/ML** | Python, CrewAI, LangChain, Llama 3.1 |
| **Data** | PostgreSQL, ClickHouse |
| **Infrastructure** | Kubernetes, Terraform, Ansible, Proxmox, Ceph |

## Documentation

- [Architecture Guide](docs/architecture/architecture.md)
- [Core Requirements](docs/srs/core_requirements.md)
- [API Reference](docs/api/kubric_gateway_v1.yaml)
- [ITIL Compliance](docs/itil/)

## Development Commands

```bash
make dev                # Start development environment
make bootstrap          # Deploy to Kubernetes
make test               # Run all tests
make lint               # Run linters
make clean              # Clean build artifacts
```

## Security

- FIPS 140-2 cryptographic operations
- HIPAA multi-tenant isolation
- SOC 2 Type II operational controls
- GDPR data residency and right to erasure
- NIST 800-53 control mappings

## Contributing

Pre-commit hooks and code style requirements:

```bash
make pre-commit-install
make lint
```

## License

Copyright (c) 2025 Kubric Contributors. See [LICENSE](LICENSE) for details.

---

**Version**: 1.0.0 | **Last Updated**: 2026-02-12
