# Kubric Documentation Tree for Notion

**Copy this structure into Notion for public-facing documentation**

---

## 📚 Documentation Structure (48 Files)

```
🏠 Home
├─ README-PUBLIC.md (Overview)
├─ QUICKSTART.md (5-minute setup)
└─ LICENSE

🚀 Getting Started
├─ Quick Start Guide (QUICKSTART.md)
├─ Installation Requirements
├─ Docker Compose Setup
└─ First Agent Deployment

🏗️ Architecture
├─ System Overview (ARCHITECTURE.md)
├─ Message Bus (NATS)
│  ├─ Cluster Configuration
│  ├─ mTLS Setup
│  └─ Subject Mapping (16 subjects)
│     ├─ EDR: Process Events
│     ├─ EDR: File Events
│     ├─ NDR: Network Flows
│     ├─ NDR: Beacon Detection
│     ├─ ITDR: Auth Events
│     ├─ VDR: Vulnerabilities
│     ├─ GRC: Compliance Drift
│     ├─ PSA: Ticketing
│     ├─ Billing: Usage
│     ├─ Health: Scores
│     ├─ TI: IOCs
│     ├─ COMM: Alerts
│     ├─ Security: Alerts
│     ├─ Remediation: Tasks
│     ├─ Assets: Provisioned
│     └─ GRC: CISO Assistant
├─ Data Lakehouse
│  ├─ ClickHouse (TTL, Arrow)
│  ├─ DuckDB (Analytics, ML)
│  └─ Migrations (Golang, Liquibase, Atlas)
└─ Security Root
   ├─ TPM Root of Trust
   └─ CA Setup

📡 API Reference
├─ Layer 1: Go Services
│  ├─ K-SVC (Tenant Management)
│  ├─ VDR (Vulnerability Detection)
│  ├─ KIC (Compliance)
│  └─ NOC (Infrastructure)
└─ Layer 2: KAI AI
   ├─ 13 Personas
   ├─ Temporal Workflows
   └─ RAG Integration

🛡️ SOC (Security Operations)
├─ Detection & Response Coverage
│  ├─ EDR (Endpoint)
│  ├─ NDR (Network)
│  ├─ ITDR (Identity)
│  ├─ CDR (Container)
│  └─ VDR (Vulnerability)
└─ Vendor Integrations
   ├─ Sigma Rules
   ├─ YARA Rules
   ├─ Suricata IDS
   ├─ Wazuh Decoders
   ├─ Falco Rules
   ├─ Zeek Scripts
   ├─ Nuclei Templates
   ├─ BloodHound
   ├─ Cortex
   ├─ TheHive
   └─ Velociraptor

🔧 NOC (Network Operations)
├─ Infrastructure Management
│  ├─ Cluster Health
│  ├─ Resource Monitoring
│  ├─ Capacity Planning
│  └─ Alerting
└─ Agent Management
   ├─ CoreSec (EDR)
   ├─ NetGuard (NDR)
   ├─ PerfTrace (Metrics)
   ├─ Watchdog (OTA Updates)
   ├─ Provisioning (Enrollment)
   ├─ Deployment
   ├─ Configuration
   └─ Troubleshooting

💼 PSA (Professional Services)
├─ ITSM Integration
│  ├─ ERPNext Setup
│  ├─ Issue Tracking
│  ├─ Billing & Invoicing
│  ├─ Asset Management
│  ├─ Contract Management
│  └─ Knowledge Base
└─ Customer Portal
   ├─ Dashboard
   ├─ Self-Service
   ├─ Security Posture
   ├─ Compliance Reports
   ├─ API Access
   └─ Webhooks

✅ GRC (Governance, Risk, Compliance)
├─ Compliance Overview (ITIL 4)
└─ 200 Framework Registry
   ├─ Framework Index
   ├─ NIST 800-53
   ├─ PCI-DSS
   ├─ ISO 27001
   ├─ SOC 2
   ├─ Compliance Automation
   │  ├─ Lula Validator
   │  ├─ OpenSCAP
   │  └─ Kyverno Policies
   ├─ Evidence Vault
   │  ├─ Immutable Audit
   │  ├─ BLAKE3 Signing
   │  ├─ Legal Hold
   │  └─ Evidence Export
   ├─ OSCAL Integration
   │  ├─ NIST Ingest
   │  ├─ SOC 2 Mapper
   │  ├─ ISO Mapping
   │  ├─ Compliance Trestle
   │  └─ RegScale Ingest
   └─ Supply Chain Security
      ├─ SBOM (Syft)
      ├─ Grype Scanner
      ├─ OpenSSF Scorecard
      ├─ SBOM Generation
      ├─ Sigstore Cosign
      ├─ OSV API Check
      ├─ Dependency Track
      └─ CycloneDX SBOM

🔄 Operations
└─ Project Status
   ├─ Current State
   ├─ Roadmap
   └─ Known Issues

🤝 Community
├─ Contributing Guide
├─ Code of Conduct
├─ GitHub Discussions
└─ Issue Tracker
```

---

## 📋 Notion Page Structure

### Top-Level Pages

1. **🏠 Home**
   - Overview
   - Quick Start
   - Key Features
   - Architecture Diagram

2. **🚀 Getting Started**
   - Prerequisites
   - Installation
   - Configuration
   - First Deployment

3. **📖 Documentation**
   - Architecture
   - API Reference
   - SOC
   - NOC
   - PSA
   - GRC
   - Operations

4. **💻 Developer**
   - API Reference
   - Contributing
   - Code Examples
   - Webhooks

5. **🛡️ Security**
   - Detection Coverage
   - Vendor Integrations
   - Threat Intelligence
   - Incident Response

6. **✅ Compliance**
   - Framework Registry
   - NIST 800-53
   - PCI-DSS
   - ISO 27001
   - SOC 2

7. **🤝 Community**
   - GitHub
   - Discussions
   - Issues
   - Roadmap

---

## 🔗 Links to Include

### GitHub
- Repository: `https://github.com/managekube-hue/Kubric-UiDR`
- Issues: `https://github.com/managekube-hue/Kubric-UiDR/issues`
- Discussions: `https://github.com/managekube-hue/Kubric-UiDR/discussions`
- Releases: `https://github.com/managekube-hue/Kubric-UiDR/releases`

### Documentation
- Portal Docs: `https://github.com/managekube-hue/Kubric-UiDR/tree/main/portal-docs`
- Quick Start: `https://github.com/managekube-hue/Kubric-UiDR/blob/main/QUICKSTART.md`
- Architecture: `https://github.com/managekube-hue/Kubric-UiDR/blob/main/portal-docs/architecture/ARCHITECTURE.md`

---

## 📊 Badges to Add

```markdown
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-compose-blue.svg)](docker-compose.yml)
[![Version](https://img.shields.io/badge/version-1.0.0--rc1-green.svg)](releases)
[![Stars](https://img.shields.io/github/stars/managekube-hue/Kubric-UiDR?style=social)](https://github.com/managekube-hue/Kubric-UiDR)
```

---

## 🎨 Notion Formatting Tips

### Use Callouts
```
💡 Tip: Start with infrastructure-only deployment first
⚠️ Warning: Requires 16GB RAM minimum
✅ Success: All services healthy
```

### Use Toggle Lists
- Collapse long sections
- Keep navigation clean
- Show/hide details

### Use Code Blocks
- Syntax highlighting
- Copy button
- Language tags

### Use Tables
- Feature comparison
- Service ports
- API endpoints

---

## 📦 Files to Upload to Notion

1. `README-PUBLIC.md` → Home page
2. `QUICKSTART.md` → Getting Started
3. `portal-docs/INDEX.md` → Documentation index
4. All files from `portal-docs/` → Respective sections

---

**Total: 48 documentation files organized into 7 main sections**
