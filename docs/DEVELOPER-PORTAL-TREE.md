# Developer Portal Documentation Tree

**Baseline:** v1.0.0-rc1 (eb7eadd9b)  
**Date:** Clean state before iteration fatigue  
**Purpose:** Single source of truth for external developers

---

## Clean Documentation Structure (50 files)

### Core Documentation (15 files)

```
docs/
├── ARCHITECTURE.md              # Complete platform architecture
├── DEPLOYMENT.md                # Deployment guides
├── PROJECT-STATUS.md            # Current state (canonical)
├── LAYER1-API-REFERENCE.md      # Go API services reference
├── LAYER2-KAI-REFERENCE.md      # KAI AI layer reference
├── COMPLIANCE.md                # ITIL 4 practices
├── DR-COVERAGE.md               # EDR/NDR/ITDR/CDR/VDR coverage
├── VENDOR-INTEGRATIONS.md       # Sigma, YARA, Suricata, etc.
├── TREE-RESTRUCTURE-SUMMARY.md  # 72-hour complexity reduction
├── EXECUTION-SUMMARY.md         # Implementation summary
├── FINAL-REPORT.md              # Final report
├── BUGFIXES-2026-02-27.md       # Bug fix log
├── BUILD-CONTEXT-FIXES.md       # Build fixes
└── KUBRIC Orchestration.docx.md # Orchestration overview
```

### Message Bus (16 files)

```
docs/message-bus/
├── K-MB-001_NATS_Cluster.yaml
├── K-MB-003_mTLS_Cert_Rotation.md
├── K-MB-004_ZeroMQ_IPC.md
└── subject-mapping/
    ├── K-MB-SUB-001_edr.process.v1.md
    ├── K-MB-SUB-002_edr.file.v1.md
    ├── K-MB-SUB-003_ndr.flow.v1.md
    ├── K-MB-SUB-004_ndr.beacon.v1.md
    ├── K-MB-SUB-005_itdr.auth.v1.md
    ├── K-MB-SUB-006_vdr.vuln.v1.md
    ├── K-MB-SUB-007_grc.drift.v1.md
    ├── K-MB-SUB-008_svc.ticket.v1.md
    ├── K-MB-SUB-009_billing.usage.v1.md
    ├── K-MB-SUB-010_health.score.v1.md
    ├── K-MB-SUB-011_ti.ioc.v1.md
    ├── K-MB-SUB-012_comm.alert.v1.md
    ├── K-MB-SUB-013_security.alert.v1.md
    ├── K-MB-SUB-014_remediation.task.v1.md
    ├── K-MB-SUB-015_asset.provisioned.v1.md
    └── K-MB-SUB-016_grc.ciso.v1.md
```

### Data Lakehouse (7 files)

```
docs/data-lakehouse/
├── clickhouse/
│   ├── K-DL-CH-003_TTL_Cold_Storage.md
│   └── K-DL-CH-005_Arrow_Bulk_Insert.md
├── duckdb/
│   ├── K-DL-DUCK-001_embedded_analytics.md
│   └── K-DL-DUCK-002_ml_feature_compute.sql
└── migrations/
    ├── K-DL-MIG-001_golang_migrate_setup.md
    ├── K-DL-MIG-002_liquibase_k8s.yaml
    └── K-DL-MIG-003_atlas_ci_sync.md
```

### Security Root (2 files)

```
docs/security-root/
├── K-SEC-002_TPM_Root_of_Trust.md
└── K-SEC-004_CA_Setup.md
```

### Historical Archive (10 files - reference only)

```
docs/archive/
├── AUDIT-REMEDIATION-2026-02-27.md
├── AWS-SANDBOX-DEPLOY-1H.md
├── BUILD-CHANGELOG.md
├── EXTERNAL-CLOSURE-RUNBOOK-2026-02-27.md
├── LIVE-PROD-DEPLOYMENT.md
├── NO-VAULT-DEPLOYMENT-WORKAROUND.md
├── NOTION-HANDOFF-BLOCKERS.md
├── PHASED-PRODUCTION-READINESS-PLAN-2026-02-27.md
├── PRODUCTION-REMEDIATION-PLAN.md
└── THOROUGH-AUDIT-2026-02-27.md
```

---

## Developer Portal Navigation

### 1. Getting Started
- Quick Start (from DEPLOYMENT.md)
- Architecture Overview (from ARCHITECTURE.md)
- Installation Guide (from DEPLOYMENT.md)

### 2. Architecture
- System Architecture (ARCHITECTURE.md)
- Message Bus (message-bus/)
- Data Lakehouse (data-lakehouse/)
- Security Root (security-root/)

### 3. API Reference
- Layer 1: Go Services (LAYER1-API-REFERENCE.md)
- Layer 2: KAI AI (LAYER2-KAI-REFERENCE.md)
- NATS Subjects (message-bus/subject-mapping/)

### 4. Detection & Response
- DR Coverage (DR-COVERAGE.md)
- Vendor Integrations (VENDOR-INTEGRATIONS.md)
- Compliance (COMPLIANCE.md)

### 5. Operations
- Deployment (DEPLOYMENT.md)
- Project Status (PROJECT-STATUS.md)
- Bug Fixes (BUGFIXES-2026-02-27.md)

---

## What to EXCLUDE from Portal

### ❌ Do NOT Expose
- `docs/archive/` - Historical iteration fatigue
- `archive/spec-modules/` - 104 outdated specs
- Internal audit documents
- Remediation plans
- Handoff blockers

### ✅ DO Expose
- 50 clean docs from v1.0.0-rc1
- GRC frameworks (07_K-GRC-07_COMPLIANCE/)
- Code (read-only browser)
- Issues (synced from GitHub)

---

## Portal Sync Strategy

### Source of Truth
```
Git Tag: v1.0.0-rc1
Commit: eb7eadd9b
Date: Clean baseline
Files: 50 documentation files
```

### Sync Process
1. Portal pulls from v1.0.0-rc1 tag (not main branch)
2. Parse 50 markdown files
3. Index in PostgreSQL + Meilisearch
4. Render on portal (read-only)
5. Update only when new release tag created

### Release Process
1. Development happens on main branch
2. When ready for external release:
   - Create new tag (v1.0.0-rc2, v1.0.0, etc.)
   - Portal auto-syncs from new tag
   - Developers see clean, stable docs

---

## Implementation Plan

### Phase 1: Extract Clean Docs (1 hour)
```bash
# Extract v1.0.0-rc1 docs to portal-docs/
git archive v1.0.0-rc1 docs/ | tar -x -C portal-docs/

# Add GRC frameworks
git archive v1.0.0-rc1 07_K-GRC-07_COMPLIANCE/ | tar -x -C portal-docs/
```

### Phase 2: Build Portal (3 days)
- Next.js 14 frontend
- PostgreSQL doc index
- Meilisearch full-text search
- GitHub OAuth

### Phase 3: Sync Service (2 days)
- Watch for new release tags
- Pull docs from tag (not main)
- Parse markdown → database
- Update search index

### Phase 4: Security (1 day)
- Read-only code browser
- PR-gated contributions
- Download logging
- Rate limiting

---

## Portal Tech Stack

```
Frontend: Next.js 14 + Tailwind + shadcn/ui
Backend: Go (matches Kubric)
Database: PostgreSQL (doc index)
Search: Meilisearch
Auth: GitHub OAuth
Hosting: Same server as Kubric or separate
```

---

## Next Steps

1. **Confirm baseline:** v1.0.0-rc1 is the clean state? ✅
2. **Portal hosting:** Where? (Same server? Separate domain?)
3. **Build portal:** Start with doc browser + search
4. **Add features:** Code browser, issues, PR flow

**Ready to build?** Say "build portal" and I'll create the implementation.
