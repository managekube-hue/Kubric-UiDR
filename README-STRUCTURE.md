# Kubric Platform - File Structure Reference

This repository uses a modular architecture organized into 10 main modules with standardized naming conventions.

## Module Overview

### K-CORE-01: Infrastructure
- **K-HW-R740_HARDWARE**: Hardware configuration documentation for Dell R740 systems
- **K-NET-NETWORKING**: Network configuration (Caddyfile, Prometheus)
- **K-HV-PROXMOX_HYPERVISOR**: Proxmox virtualization and Ceph storage setup
- **K-K8S-KUBERNETES**: Kubernetes manifests and deployments
- **K-DL-DATA_LAKEHOUSE**: ClickHouse and PostgreSQL database configurations
- **K-MB-MESSAGE_BUS**: NATS JetStream messaging system setup
- **K-SEC-SECURITY_ROOT**: Security infrastructure (Vault, PKI, Blake3)

### K-XRO-02: Super Agent
High-performance security agents built in Rust with eBPF kernel integration.
- **K-XRO-CS_CORESEC**: Kernel security monitoring agent
- **K-XRO-NG_NETGUARD**: Network detection and response agent  
- **K-XRO-PT_PERFTRACE**: Performance profiling agent
- **K-XRO-WD_WATCHDOG**: Agent orchestration and patching
- **K-XRO-PV_PROVISIONING**: Agent registration and deployment

### K-KAI-03: Orchestration
AI-driven incident response and automation.
- **K-KAI-CP_CREWAI_PERSONAS**: CrewAI-based intelligent agents
  - K-KAI-TRIAGE: Incident triage agent
  - K-KAI-HOUSE: Remediation housekeeper agent
  - K-KAI-BILL: Billing clerk agent
  - K-KAI-COMM: Communications agent
- **K-KAI-WF_WORKFLOW**: Workflow engines (n8n and Temporal)
- **K-KAI-GD_GUARDRAILS**: Safety and approval controls
- **K-KAI-RAG**: Retrieval-Augmented Generation for CISO assistant
- **K-KAI-AUDIT**: Decision history and audit trails

### K-SOC-04: Security
Detection, correlation, and forensics.
- **K-SOC-DET_DETECTION**: Sigma rules, scanner integrations
- **K-SOC-TI_THREAT_INTEL**: Threat intelligence feeds
- **K-SOC-VULN_VULNERABILITY**: Vulnerability scanning
- **K-SOC-IS_INCIDENT_STITCH**: Graph-based incident correlation
- **K-SOC-FR_FORENSICS**: Forensic evidence collection

### K-NOC-05: Operations
Operational management and observability.
- **K-NOC-CM_CONFIG_MGMT**: Ansible playbooks for configuration management
- **K-NOC-BR_BACKUP_DR**: Backup and disaster recovery
- **K-NOC-PM_PERFORMANCE**: Performance monitoring with OpenTelemetry
- **K-NOC-PT_PATCH_MGMT**: Patch management and deltas

### K-PSA-06: Business
Professional services and billing.
- **K-PSA-ITSM**: IT service management and ticketing
- **K-PSA-BILL_BILLING**: Billing and invoicing
- **K-PSA-CRM_CPQ**: Customer relationship management
- **K-PSA-PTL_PORTAL**: Web portal (Next.js, React)
- **K-PSA-BI_BUSINESS_INTEL**: Business intelligence and QBR

### K-GRC-07: Compliance
Governance, risk, and compliance.
- **K-GRC-OSCAL**: OSCAL framework and NIST control mappings
- **K-GRC-EV_EVIDENCE_VAULT**: Immutable evidence storage
- **K-GRC-SCS_SUPPLY_CHAIN**: SBOM and supply chain security
- **K-GRC-CA_COMPLIANCE_AUTO**: Automated compliance validation

### K-DEV-08: Development
Development tools and CI/CD infrastructure.
- **K-DEV-LOCAL_LOCAL_STACK**: Docker Compose for local development
- **K-DEV-BLD_BUILD_TOOLCHAIN**: Build tools (Makefile, toolchains)
- **K-DEV-CICD**: GitHub Actions workflows
- **K-DEV-GIT_GITOPS**: GitOps and pre-commit hooks
- **K-DEV-DOC_DOCUMENTATION**: Architecture docs and licenses

### K-API-09: API Reference
API specifications and protocols.
- **K-API-OPENAPI**: OpenAPI 3.0 specifications
- **K-API-PB_PROTOBUF**: Protocol Buffer definitions

### K-ITIL-10: ITIL Matrix
Compliance and process mapping.
- **K-ITIL-01_FRAMEWORK**: ITIL 4 framework documents
- **K-ITIL-02_GMP_MAP**: General Management Practices mapping
- **K-ITIL-03_SMP_MAP**: Service Management Practices mapping
- **K-ITIL-04_TMP_MAP**: Technology Management Practices mapping
- **K-ITIL-05_AUDIT_READINESS**: Audit and KIC evidence collection

## File Naming Convention

All files follow the pattern: `K-[MODULE]-[SUBMODULE]-###_filename`

Examples:
- `K-K8S-001_namespace.yaml` - Kubernetes namespace
- `K-DL-CH-001_config.xml` - ClickHouse configuration
- `K-NOC-CM-ANS-001_isolate_host.yml` - Ansible playbook
- `K-DEV-CICD-GHA-001_build_agents.yml` - GitHub Actions workflow

## Navigation Tips

1. **Infrastructure setup**: Start in `K-CORE-01_INFRASTRUCTURE/`
2. **Build agents**: See `K-XRO-02_SUPER_AGENT/`
3. **Deploy services**: Check `K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/`
4. **Security operations**: Review `K-SOC-04_SECURITY/`
5. **Operational tasks**: Find in `K-NOC-05_OPERATIONS/`
6. **Development**: Tools in `K-DEV-08_DEVELOPMENT/`

---

For detailed documentation, see K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/
