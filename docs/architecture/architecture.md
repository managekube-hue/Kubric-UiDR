# Kubric Platform Architecture

This document describes the overall architecture of the Kubric Platform, organized into 9 modules.

## Module Structure

### K-CORE-01: Infrastructure
Core infrastructure components that form the foundation of the platform.
- **Blake3 Hardware Fingerprint**: Device identification via Go
- **PostgreSQL UAR**: User Account Registry with Row-Level Security
- **ClickHouse OCSF**: Event storage and analytics
- **NATS JetStream**: Event streaming and messaging
- **HashiCorp Vault**: Secrets management
- **Proxmox/Ceph**: Infrastructure hypervisor and distributed storage
- **Kubernetes**: Container orchestration

### K-XRO-02: Super Agent
Next-generation security agents with isolation and performance optimization.
- **CoreSec (eBPF)**: Operating system security monitoring in Rust
- **NetGuard (NDR)**: Network detection and response
- **PerfTrace (Metrics)**: Performance tracking and profiling
- **Watchdog Orchestrator**: Agent lifecycle management
- **Zstd Delta Patcher**: Efficient binary updates
- **Go-tuf Update Framework**: Secure update distribution
- **Agent Registration**: Provisioning and onboarding

### K-KAI-03: Orchestration
AI-driven orchestration engine for remediation and automation.
- **Triage Agent**: Uses CrewAI + Llama 3.1 for incident analysis
- **Housekeeper Agent**: Automated remediation via Ansible
- **Billing Clerk Agent**: Usage metering and cost calculation
- **Comm Agent**: Communications via Vapi/Deepgram/Twilio
- **n8n Workflows**: Visual workflow automation
- **Temporal.io**: Workflow orchestration engine
- **CISO Assistant**: RAG-based compliance advisor
- **Merkle Tree Signer**: Cryptographic attestation

### K-SOC-04: Security
Security orchestration and correlation engine.
- **Sigma Rule Compiler**: Detection rule compilation (Go)
- **Nuclei Scanner**: Vulnerability scanning (Go)
- **MITRE ATT&CK Mapper**: Adversary tactic mapping
- **EPSS API**: Exploit prediction scoring
- **AlienVault OTX**: Open threat exchange integration
- **AbuseIPDB**: Reputation lookups
- **Incident Stitching**: Graph-based correlation
- **YARA Integration**: Malware detection

### K-NOC-05: Operations
Network and operations center functions.
- **Ansible Playbooks**: Infrastructure automation
- **Osquery/FleetDM**: Host drift detection
- **Restic**: Incremental backups
- **Kopia**: Snapshot management
- **OpenTelemetry**: Observability pipeline
- **Anomaly Detection**: ML-based detection

### K-PSA-06: Business
Professional services automation and billing.
- **Ticketing State Machine**: Incident/ticket lifecycle
- **SLA Tracker**: Service level agreement monitoring
- **Metered Usage Aggregator**: Billing data processing
- **PDF Invoice Generator**: Automated invoicing
- **Next.js Portal**: React-based web interface
- **Tremor Dashboard**: Data visualization
- **Deployment Wizard**: Guided onboarding

### K-GRC-07: Compliance
Governance, risk, and compliance management.
- **OSCAL Framework**: NIST format compliance mapping
- **NIST 800-53 Mapper**: Control family tracking
- **FAIR Risk Quantification**: Quantitative risk analysis
- **SBOM Generation**: Software bill of materials
- **Grype Scanner**: Vulnerability auditing
- **OpenSSF Scorecard**: Supply chain security assessment
- **Evidence Vault**: Immutable audit logs

### K-DEV-08: Development
Development tools and CI/CD orchestration.
- **Rust Toolchain**: eBPF and high-performance components
- **Go Workspace**: Backend services and tools
- **Python Virtual Environment**: AI and data processing
- **Next.js Setup**: Frontend framework
- **GitHub Actions**: CI/CD automation
- **Pre-commit Hooks**: Code quality gates
- **Architecture Documentation**: Design guides

### K-API-09: API Reference
External and internal API specifications.
- **OpenAPI 3.0**: REST API specifications
- **Protocol Buffers**: gRPC service definitions

## Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      KUBERNETES CLUSTER                      │
├──────────────┬──────────────┬──────────────┬─────────────────┤
│   Control    │   Worker     │   Worker     │   Worker        │
│   Plane      │   01         │   02         │   03            │
├──────────────┼──────────────┴──────────────┴─────────────────┤
│              │                                                │
│   etcd       │  Namespace: kubric                            │
│   API        │  ├── NATS (StatefulSet, replicas=3)          │
│   Controller │  ├── ClickHouse (StatefulSet, replicas=1)    │
│   Scheduler  │  ├── PostgreSQL (StatefulSet, replicas=1)    │
│              │  ├── API (Deployment, replicas=3)            │
│              │  ├── KAI Orchestration (Deployment, replicas=2)
│              │  ├── Web Portal (Deployment, replicas=3)     │
│              │  └── Ingress/Caddy                           │
│              │                                               │
│              │  Storage: Ceph RBD (500Gi PVC)               │
└──────────────┴───────────────────────────────────────────────┘

                        Infrastructure Layer
┌──────────────────────────────────────────────────────────────┐
│                      PROXMOX HYPERVISOR                       │
├──────────────────────────────────────────────────────────────┤
│  Ceph Storage NODES                                           │
│  ├── OSD 1              ├── OSD 2              ├── OSD 3     │
│  └── Mon 1             └── Mon 2              └── Mon 3      │
└──────────────────────────────────────────────────────────────┘
```

## Data Flow

1. **Agent → Event Stream**: Agents publish events via NATS
2. **Event Storage**: NATS → ClickHouse via streaming consumer
3. **Incident Detection**: Sigma rules compiled and executed
4. **Triage**: Triage Agent (CrewAI) analyzes via ClickHouse queries
5. **Remediation**: Housekeeper Agent executes Ansible playbooks
6. **Billing**: Heartbeat events aggregated in ClickHouse
7. **Compliance**: Evidence stored in PostgreSQL with RLS

## Technology Stack by Language

| Language | Usage | Modules |
|----------|-------|---------|
| **Go** | Backend services, scanning, state machines, billing | Core, XRO, KAI, SOC, NOC, PSA, GRC |
| **Rust** | eBPF agents, high-performance monitoring | XRO, SOC, GRC |
| **Python** | AI agents, threat intelligence, compliance | KAI, SOC, NOC, GRC |
| **TypeScript/React** | Web portals, dashboards | PSA |
| **SQL** | Schema, queries, RLS | Core, KAI, SOC, NOC |
| **YAML** | Config, orchestration, infrastructure-as-code | All |
| **JSON** | n8n workflows, API specs | KAI, API |
| **Proto** | gRPC definitions | API |
| **Markdown** | Documentation | Dev |

---

**Kubric Platform v1.0** | Last Updated: 2026-02-12
