# ITIL 4 Compliance Matrix

This document maps Kubric platform capabilities to ITIL 4 practices.

## Service Value Chain (SVC)

| SVC Activity | Kubric Component | Mapping |
|---|---|---|
| **Plan** | Triage Agent + Reports | Analyzes incident patterns, forecasts |
| **Improve** | CISO Assistant + Feedback loops | Learns from incidents, updates runbooks |
| **Engage** | Comm Agent + Portal | Stakeholder communication and portal |
| **Design & Transition** | Deployment Wizard + Terraform | Service modeling and deployment |
| **Obtain/Build** | Agent Provisioning + Update Framework | Acquisition and configuration mgmt |
| **Deliver & Support** | KAI Orchestration + Portal | Incident response and support |

## Core Practices Mapped

### Incident Management
- **Detection**: CoreSec & NetGuard agents
- **Triage**: Triage Agent (CrewAI)
- **Assignment**: Automated via severity + routing rules
- **Resolution**: Housekeeper Agent (Ansible playbooks)
- **Closure**: Temporal workflows with approval gates
- **Post-Incident Review**: ClickHouse analytics dashboard

### Problem Management
- **Root Cause Analysis**: CISO Assistant RAG with threat intel
- **Trend Analysis**: ClickHouse historical queries
- **Known Error Database**: PostgreSQL error catalog
- **Prevention**: Patch automation via CVE scoring (EPSS)

### Change Management
- **Change Planning**: Terraform IaC for infrastructure changes
- **CAB Approval**: Optional approval gates in Temporal workflows
- **Testing**: Staging environment via terraform/environments
- **Deployment**: GitOps via Flux or ArgoCD (extensible)
- **Back-out Plans**: Automated rollback with Ansible

### Configuration Management
- **CMDB**: PostgreSQL UAR + inventory integration
- **Version Control**: Git for all IaC and configs
- **Baseline Tracking**: Osquery + FleetDM for compliance
- **Drift Detection**: Automated with housekeeper remediation

### Service Level Management
- **SLA Definitions**: Contract templates in docs/
- **SLA Tracking**: Metrics in ClickHouse, dashboards in Tremor
- **Breach Escalation**: Comm Agent escalation workflows
- **Reporting**: Monthly SLA reports to stakeholders

### Availability Management
- **Monitoring**: Prometheus + OpenTelemetry
- **Redundancy**: 3-node NATS, Ceph, Kubernetes HA
- **Backup**: Restic incremental backups, Kopia snapshots
- **RTO/RPO**: Defined in operational runbooks

### Continuity Management
- **Business Impact Analysis**: Risk quantification (FAIR) in GRC module
- **DR Plans**: Documented in terraform/environments (prod + staging)
- **Recovery Procedures**: Ansible playbooks in deployments/ansible/playbooks/

### Information Security Management
- **Policy Framework**: OSIBL + NIST 800-53 mappings
- **Access Control**: PostgreSQL RLS + Blake3 fingerprints
- **Compliance Monitoring**: OSCAL ingestion + evidence vault
- **Vulnerability Management**: Grype + Syft + SBOM generation

### Supplier Management
- **Third-party SLAs**: Tracked in contract system
- **Vendor Risk**: Integrated threat intelligence (OTX, AbuseIPDB)
- **Service Reviews**: Escalation workflows

### IT Asset Management
- **Discovery**: Osquery + FleetDM queries
- **Tracking**: PostgreSQL inventory
- **Deprecation**: Automated removal from inventory
- **Reporting**: ClickHouse asset analytics

---

**Note**: This matrix assumes optional extensions for CMDB, ITSM ticketing backend, and GitOps. Core module provides foundation.

Generated: 2026-02-12
