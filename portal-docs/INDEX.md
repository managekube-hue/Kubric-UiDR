# Kubric Platform Documentation

**Version:** v1.0.0-rc1  
**Last Updated:** 2026-02-28

---

## Quick Navigation

### 🚀 Getting Started
- [Deployment Guide](getting-started/DEPLOYMENT.md) - Install and configure Kubric

### 🏗️ Architecture
- [System Architecture](architecture/ARCHITECTURE.md) - Complete platform overview
- [Message Bus (NATS)](architecture/message-bus/) - Event streaming architecture
- [Data Lakehouse](architecture/data-lakehouse/) - ClickHouse + DuckDB
- [Security Root](architecture/security-root/) - PKI and TPM

### 📡 API Reference
- [Layer 1: Go Services](api-reference/LAYER1-API-REFERENCE.md) - K-SVC, VDR, KIC, NOC
- [Layer 2: KAI AI](api-reference/LAYER2-KAI-REFERENCE.md) - 13 AI personas

### 🛡️ SOC (Security Operations)
- [Detection & Response Coverage](soc/DR-COVERAGE.md) - EDR, NDR, ITDR, CDR, VDR
- [Vendor Integrations](soc/VENDOR-INTEGRATIONS.md) - Sigma, YARA, Suricata, Wazuh

### 🔧 NOC (Network Operations)
- [Infrastructure Management](noc/INFRASTRUCTURE.md) - Cluster health, monitoring
- [Agent Management](noc/AGENTS.md) - CoreSec, NetGuard, PerfTrace, Watchdog

### 💼 PSA (Professional Services)
- [ITSM Integration](psa/ITSM.md) - ERPNext ticketing and billing
- [Customer Portal](psa/PORTAL.md) - Self-service and reporting

### ✅ GRC (Governance, Risk, Compliance)
- [Compliance Overview](compliance/COMPLIANCE.md) - ITIL 4 practices
- [200 Framework Registry](compliance/frameworks/K-GRC-FW-000_framework_index.md)
- [NIST 800-53](compliance/frameworks/K-GRC-FW-001_nist_800_53_oscal.md)
- [PCI-DSS](compliance/frameworks/K-GRC-FW-002_pci_dss_oscal.md)
- [ISO 27001](compliance/frameworks/K-GRC-FW-003_iso_27001_oscal.md)
- [SOC 2](compliance/frameworks/K-GRC-FW-004_soc2_oscal.md)

### 🔄 Operations
- [Project Status](operations/PROJECT-STATUS.md) - Current state and roadmap

---

## Documentation by Role

### Security Analyst (SOC)
1. [DR Coverage](soc/DR-COVERAGE.md) - What threats we detect
2. [Vendor Integrations](soc/VENDOR-INTEGRATIONS.md) - Detection tools
3. [NATS Subjects](architecture/message-bus/subject-mapping/) - Event streams

### Infrastructure Engineer (NOC)
1. [Infrastructure Management](noc/INFRASTRUCTURE.md) - Cluster operations
2. [Agent Management](noc/AGENTS.md) - Deploy and monitor agents
3. [Data Lakehouse](architecture/data-lakehouse/) - Storage architecture

### Compliance Officer (GRC)
1. [Framework Registry](compliance/frameworks/K-GRC-FW-000_framework_index.md) - 200 frameworks
2. [Evidence Vault](compliance/frameworks/K-GRC-EV_EVIDENCE_VAULT/) - Audit trails
3. [OSCAL Integration](compliance/frameworks/K-GRC-OSCAL/) - Automated compliance

### Developer
1. [API Reference](api-reference/) - REST APIs and KAI AI
2. [Message Bus](architecture/message-bus/) - NATS event streaming
3. [System Architecture](architecture/ARCHITECTURE.md) - Platform design

### Administrator
1. [Deployment Guide](getting-started/DEPLOYMENT.md) - Installation
2. [Project Status](operations/PROJECT-STATUS.md) - Current state
3. [ITSM Integration](psa/ITSM.md) - Ticketing and billing

---

## File Count: 43 Documentation Files

- Getting Started: 1 file
- Architecture: 26 files
- API Reference: 2 files
- SOC: 2 files
- NOC: 2 files (to be created)
- PSA: 2 files (to be created)
- Compliance: 9 files
- Operations: 1 file
