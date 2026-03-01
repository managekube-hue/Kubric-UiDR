# Kubric Documentation Inventory - Pre-Production

**Date:** 2025-02-28  
**Purpose:** Complete inventory before developer portal creation  
**Status:** Days away from production

---

## Documentation Statistics

| Category | Count | Location | Status |
|----------|-------|----------|--------|
| **Active Docs** | 57 | `docs/` | ✅ Current |
| **GRC Frameworks** | 9 | `07_K-GRC-07_COMPLIANCE/` | ✅ 200 frameworks |
| **Archived Specs** | 104 | `archive/spec-modules/` | ✅ Historical |
| **Go Code Files** | 68 | `internal/`, `cmd/` | ✅ Implementation |
| **Rust Agents** | 5 | `agents/` | ✅ CoreSec, NetGuard, etc. |
| **Python KAI** | 13 personas | `kai/agents/` | ✅ AI orchestration |
| **TOTAL DOCS** | **170+** | Multiple | **MASSIVE** |

---

## What EXISTS (Current State)

### 1. Core Documentation (`docs/`)

#### Architecture & Design
- `ARCHITECTURE.md` - Complete platform architecture
- `DEPLOYMENT.md` - Deployment guides
- `PROJECT-STATUS.md` - Current state (canonical)
- `LAYER1-API-REFERENCE.md` - Go API services
- `LAYER2-KAI-REFERENCE.md` - KAI AI layer

#### Compliance & Security
- `COMPLIANCE.md` - ITIL 4 practices
- `DR-COVERAGE.md` - EDR/NDR/ITDR/CDR/VDR coverage
- `VENDOR-INTEGRATIONS.md` - Sigma, YARA, Suricata, etc.

#### Implementation Records
- `TREE-RESTRUCTURE-SUMMARY.md` - 72-hour complexity reduction
- `ITSM-SIMPLIFICATION.md` - Zammad removal decision
- `ERPNEXT-INTEGRATION.md` - Complete ITSM solution
- `BUGFIXES-2026-02-27.md` - Bug fix log
- `CRITICAL-STATUS-VERIFICATION.md` - Pre-production verification

#### Historical Archive (`docs/archive/`)
- 10 audit/remediation documents
- Production readiness plans
- Deployment workarounds
- Handoff documentation

### 2. GRC Frameworks (`07_K-GRC-07_COMPLIANCE/`)

#### Framework Registry
- `K-GRC-FW-000_framework_index.md` - 200 frameworks master index
- `K-GRC-FW-001_nist_800_53_oscal.md` - NIST 800-53
- `K-GRC-FW-002_pci_dss_oscal.md` - PCI-DSS
- `K-GRC-FW-003_iso_27001_oscal.md` - ISO 27001
- `K-GRC-FW-004_soc2_oscal.md` - SOC 2

#### Implementation Modules
- `K-GRC-CA/` - Compliance assessor
- `K-GRC-EV/` - Evidence vault
- `K-GRC-OSCAL/` - OSCAL ingestion
- `K-GRC-SCS/` - Supply chain security (SBOM)

### 3. Archived Specifications (`archive/spec-modules/`)

#### Vendor Detection Assets (104 files)
- `00_K-VENDOR-00_DETECTION_ASSETS/` - Sigma, YARA, Suricata, Wazuh, etc.
- `10_K-ITIL-10_ITIL_MATRIX/` - ITIL 4 practice mapping
- `11_K-MAP-11_DR_MODULE_MAPPING/` - DR module master index

### 4. Technical Documentation (Embedded in Code)

#### Go Services (`internal/`, `cmd/`)
- 68 Go files with inline documentation
- API endpoint definitions
- Database schemas
- NATS subject hierarchies

#### Rust Agents (`agents/`)
- CoreSec: eBPF/ETW hooks, Sigma/YARA engines
- NetGuard: DPI, IDS, TLS inspection
- PerfTrace: Metrics collection
- Watchdog: OTA updates
- Provisioning: Agent enrollment

#### Python KAI (`kai/`)
- 13 AI personas with role/goal/backstory
- Temporal workflows
- PSA integrations (ERPNext, n8n)

### 5. Configuration & Infrastructure

#### Docker Compose
- `docker-compose.yml` - Development stack
- `docker-compose.prod.yml` - Production hardening

#### AI-Readable Configs
- `.kubric/stack.yaml` - Service inventory
- `.kubric/deployment-rules.yaml` - Auto-scale rules
- `.kubric/health-checks.yaml` - Monitoring thresholds

#### Database Migrations
- `migrations/postgres/` - Schema evolution
- `migrations/clickhouse/` - Event tables

---

## What WILL EXIST (Developer Portal Requirements)

### 1. User-Facing Documentation

#### Getting Started
- [ ] Quick Start Guide (5 min to first deployment)
- [ ] Installation Guide (Proxmox, Docker, bare-metal)
- [ ] Configuration Guide (environment variables, secrets)
- [ ] First Agent Deployment

#### User Guides
- [ ] Admin Guide (tenant management, billing)
- [ ] Analyst Guide (SOC workflows, investigation)
- [ ] Developer Guide (API integration, webhooks)
- [ ] Compliance Officer Guide (GRC workflows, evidence)

#### Reference Documentation
- [ ] API Reference (OpenAPI/Swagger for all 7 services)
- [ ] NATS Subject Reference (complete hierarchy)
- [ ] Database Schema Reference (Postgres, ClickHouse)
- [ ] Agent Configuration Reference (all 5 agents)

#### Deployment Guides
- [ ] Single-Node Deployment (Docker Compose)
- [ ] Multi-Node Deployment (Docker Swarm)
- [ ] Proxmox Cluster Deployment (production)
- [ ] AWS/Azure/GCP Deployment (cloud)
- [ ] Air-Gapped Deployment (offline)

### 2. Developer Documentation

#### Contributing
- [ ] Contribution Guidelines
- [ ] Code Style Guide (Go, Rust, Python)
- [ ] PR Template & Review Process
- [ ] Testing Requirements
- [ ] Security Guidelines

#### Architecture Deep Dives
- [ ] Agent Architecture (eBPF, ETW, hooks)
- [ ] Message Bus Architecture (NATS JetStream)
- [ ] Data Lake Architecture (ClickHouse partitioning)
- [ ] AI Orchestration Architecture (KAI personas)
- [ ] Multi-Tenancy Architecture (RLS, isolation)

#### Integration Guides
- [ ] Webhook Integration
- [ ] SIEM Integration (Splunk, Elastic, QRadar)
- [ ] SOAR Integration (Cortex, TheHive)
- [ ] Ticketing Integration (ERPNext, Jira)
- [ ] Custom Detection Rules (Sigma, YARA)

### 3. Operational Documentation

#### Runbooks
- [ ] Incident Response Runbook
- [ ] Backup & Restore Runbook
- [ ] Disaster Recovery Runbook
- [ ] Scaling Runbook
- [ ] Upgrade Runbook

#### Troubleshooting
- [ ] Common Issues & Solutions
- [ ] Log Analysis Guide
- [ ] Performance Tuning Guide
- [ ] Network Troubleshooting
- [ ] Agent Troubleshooting

#### Monitoring & Alerting
- [ ] Grafana Dashboard Guide
- [ ] Alert Configuration Guide
- [ ] SLA Monitoring Guide
- [ ] Capacity Planning Guide

### 4. Compliance Documentation

#### Framework Guides (per framework)
- [ ] NIST 800-53 Implementation Guide
- [ ] PCI-DSS Implementation Guide
- [ ] ISO 27001 Implementation Guide
- [ ] SOC 2 Implementation Guide
- [ ] HIPAA Implementation Guide
- [ ] (195 more frameworks...)

#### Audit Support
- [ ] Evidence Collection Guide
- [ ] Audit Readiness Checklist
- [ ] Control Mapping Guide
- [ ] Compliance Reporting Guide

### 5. Training Materials

#### Video Tutorials
- [ ] Platform Overview (10 min)
- [ ] First Deployment (15 min)
- [ ] SOC Analyst Workflow (20 min)
- [ ] Compliance Workflow (20 min)
- [ ] Custom Detection Rules (15 min)

#### Workshops
- [ ] Admin Workshop (4 hours)
- [ ] Analyst Workshop (4 hours)
- [ ] Developer Workshop (8 hours)
- [ ] Compliance Workshop (4 hours)

---

## Developer Portal Structure (Proposed)

```
Developer Portal (Your Website)
├─ Home
│   ├─ Platform Overview
│   ├─ Quick Start (5 min)
│   └─ Latest Release Notes
│
├─ Documentation
│   ├─ Getting Started
│   │   ├─ Installation
│   │   ├─ Configuration
│   │   └─ First Deployment
│   │
│   ├─ User Guides
│   │   ├─ Admin Guide
│   │   ├─ Analyst Guide
│   │   ├─ Developer Guide
│   │   └─ Compliance Guide
│   │
│   ├─ Reference
│   │   ├─ API Reference (OpenAPI)
│   │   ├─ NATS Subjects
│   │   ├─ Database Schema
│   │   └─ Agent Config
│   │
│   ├─ Architecture
│   │   ├─ System Overview
│   │   ├─ Agent Architecture
│   │   ├─ Message Bus
│   │   ├─ Data Lake
│   │   └─ AI Orchestration
│   │
│   └─ Compliance
│       ├─ Framework Index (200)
│       ├─ NIST 800-53
│       ├─ PCI-DSS
│       ├─ ISO 27001
│       └─ SOC 2
│
├─ Code Browser (Read-Only)
│   ├─ Browse Repository
│   ├─ Search Code
│   ├─ View Commits
│   └─ Download Release
│
├─ Issues
│   ├─ Browse Issues (synced from GitHub)
│   ├─ Create Issue (creates GitHub issue)
│   └─ Track Status
│
├─ Contribute
│   ├─ Contribution Guidelines
│   ├─ Submit PR (opens GitHub PR)
│   ├─ Code Style Guide
│   └─ Security Guidelines
│
└─ Community
    ├─ Forum/Discussions
    ├─ Roadmap
    ├─ Changelog
    └─ Support
```

---

## Security Model (Your Requirements)

### Read-Only Access
- Documentation: Full read access
- Code: Browse only, no clone/download without logging
- Issues: View only, create via portal (creates GitHub issue)

### Controlled Contributions
- No direct repo access
- All contributions via PR workflow
- PR review required before merge
- Automated security scanning on PR

### Malicious Actor Mitigation
- No direct commit access (PR-only)
- No repo deletion capability
- Download logging & rate limiting
- IP-based access control (optional)
- Code watermarking (optional)

### Source Code Protection
- No local clone without authentication
- Download requires account + logging
- Release binaries only (no source by default)
- Enterprise customers get source access

---

## Sync Strategy

### GitHub → Portal (One-Way)

```
GitHub Repo (Source of Truth)
    ↓ Webhook on push
Portal Sync Service
    ↓ Pull changes
Portal Database
    ↓ Render
Developer Portal (Read-Only)
```

### Portal → GitHub (PR-Gated)

```
Developer Portal
    ↓ Submit contribution
GitHub PR Created
    ↓ Review + CI
Approved & Merged
    ↓ Webhook
Portal Syncs Changes
```

---

## Technology Stack (Proposed)

### Portal Backend
- **Framework:** Go (matches Kubric stack)
- **Database:** PostgreSQL (documentation index)
- **Search:** Meilisearch or Typesense
- **Auth:** OAuth2 (GitHub, Google, SSO)

### Portal Frontend
- **Framework:** Next.js 14 (matches Kubric frontend)
- **UI:** Tailwind CSS + shadcn/ui
- **Docs Rendering:** MDX or Markdoc
- **Code Highlighting:** Shiki

### Sync Service
- **Language:** Go
- **Trigger:** GitHub webhooks + scheduled (hourly)
- **Storage:** Git bare repo + PostgreSQL index
- **Search Index:** Meilisearch

---

## Immediate Actions Required

### 1. Documentation Audit (1 day)
- [ ] Categorize all 170+ docs
- [ ] Identify gaps (user guides, API reference)
- [ ] Mark outdated/deprecated docs
- [ ] Create documentation roadmap

### 2. Portal Design (2 days)
- [ ] Wireframe portal structure
- [ ] Design navigation/search
- [ ] Plan sync architecture
- [ ] Define security model

### 3. Sync Service (3 days)
- [ ] Build GitHub webhook receiver
- [ ] Implement doc parser (Markdown → DB)
- [ ] Build search indexer
- [ ] Create PR creation API

### 4. Portal MVP (5 days)
- [ ] Documentation browser
- [ ] Search functionality
- [ ] Issue tracker (read-only)
- [ ] PR submission flow

---

## Questions for You

1. **Portal Hosting:** Where will developer portal live? (Same server as Kubric? Separate?)
2. **Authentication:** GitHub OAuth only? Or support Google/SSO?
3. **Source Access:** Who gets source code access? (Enterprise only? Open source?)
4. **Download Logging:** Track who downloads what? IP logging?
5. **Forum/Community:** Do you want built-in discussions or use GitHub Discussions?
6. **Release Cadence:** How often do you want docs synced? (Real-time webhook? Hourly? Daily?)

---

**Next Step:** Tell me your priorities and I'll create the implementation plan.
