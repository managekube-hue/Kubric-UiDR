# Portal Documentation Tree

**Source:** Git tag `v1.0.0-rc1` (eb7eadd9b)  
**Files:** 58 clean documentation files  
**Purpose:** External developer portal sync

---

## What's Here

```
portal-docs/
├── docs/                           # 50 core docs
│   ├── ARCHITECTURE.md
│   ├── DEPLOYMENT.md
│   ├── PROJECT-STATUS.md
│   ├── LAYER1-API-REFERENCE.md
│   ├── LAYER2-KAI-REFERENCE.md
│   ├── COMPLIANCE.md
│   ├── DR-COVERAGE.md
│   ├── VENDOR-INTEGRATIONS.md
│   ├── message-bus/                # NATS subject hierarchy
│   ├── data-lakehouse/             # ClickHouse, DuckDB
│   ├── security-root/              # PKI, TPM
│   └── archive/                    # Historical (optional)
│
└── 07_K-GRC-07_COMPLIANCE/         # 8 GRC framework docs
    ├── K-GRC-FW-000_framework_index.md
    ├── K-GRC-FW-001_nist_800_53_oscal.md
    ├── K-GRC-FW-002_pci_dss_oscal.md
    ├── K-GRC-FW-003_iso_27001_oscal.md
    └── K-GRC-FW-004_soc2_oscal.md
```

---

## Sync Instructions

### Your Portal Should Pull From

**Git Repository:** `https://github.com/YOUR-ORG/Kubric-UiDR`  
**Directory:** `portal-docs/`  
**Branch:** `main` (or create `portal` branch)  
**Frequency:** On push to main, or hourly

### Sync Command (Example)

```bash
# Clone repo
git clone https://github.com/YOUR-ORG/Kubric-UiDR.git

# Pull portal docs
cd Kubric-UiDR/portal-docs

# Sync to your portal
rsync -av docs/ /var/www/portal/docs/
rsync -av 07_K-GRC-07_COMPLIANCE/ /var/www/portal/compliance/
```

### Webhook Sync (Recommended)

```yaml
# GitHub webhook → your portal
POST https://your-portal.com/api/sync
{
  "ref": "refs/heads/main",
  "repository": "Kubric-UiDR",
  "commits": [...]
}

# Your portal pulls portal-docs/ and rebuilds
```

---

## What to Exclude

**DO NOT sync these directories:**
- `archive/spec-modules/` (104 outdated specs)
- `docs/archive/` (optional - historical only)
- `.git/` (obviously)
- `vendor/` (detection rules - separate sync)

---

## Portal Structure Mapping

### Your Portal Navigation → Our Files

```
Portal: Getting Started
  └─ docs/DEPLOYMENT.md

Portal: Architecture
  ├─ docs/ARCHITECTURE.md
  ├─ docs/message-bus/
  ├─ docs/data-lakehouse/
  └─ docs/security-root/

Portal: API Reference
  ├─ docs/LAYER1-API-REFERENCE.md
  └─ docs/LAYER2-KAI-REFERENCE.md

Portal: Detection & Response
  ├─ docs/DR-COVERAGE.md
  └─ docs/VENDOR-INTEGRATIONS.md

Portal: Compliance
  └─ 07_K-GRC-07_COMPLIANCE/

Portal: Operations
  ├─ docs/PROJECT-STATUS.md
  └─ docs/BUGFIXES-2026-02-27.md
```

---

## Update Process

### When We Update Docs

1. We commit to `main` branch
2. We update `portal-docs/` directory
3. GitHub webhook triggers your portal sync
4. Your portal pulls latest `portal-docs/`
5. Your portal rebuilds documentation

### When We Release

1. We create new tag (v1.0.0, v1.1.0, etc.)
2. We update `portal-docs/` from new tag
3. Your portal syncs automatically
4. Developers see stable release docs

---

## Security Notes

### Read-Only Access
- Your portal pulls from GitHub (read-only)
- Developers cannot push to this repo
- All contributions via PR workflow

### PR Workflow
- Developer submits via your portal
- Your portal creates GitHub PR
- We review and merge
- Portal syncs updated docs

---

## File Count

- **Core docs:** 50 files
- **GRC frameworks:** 8 files
- **Total:** 58 files
- **Size:** ~2 MB

---

## Sync Endpoint

**Point your sync to:**
```
Repository: https://github.com/YOUR-ORG/Kubric-UiDR
Directory: portal-docs/
Branch: main
```

**That's it.** Everything you need is in `portal-docs/`.
