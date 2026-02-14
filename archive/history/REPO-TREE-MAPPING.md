# Kubric UIDR Canonical Tree Mapping

This is the single source of truth for repository structure and DocuNotion mapping.

## 1) Source Tree (Git)

- App shell and site config live at repository root.
- Technical docs content used by Docusaurus and DocuNotion sync lives in `docs/`.
- Marketing/routes pages live in `src/pages/`.

### Canonical roots
- Site config: `docusaurus.config.ts`
- Sidebar tree: `sidebars.js`
- Notion sync workflow: `.github/workflows/sync-notion-docs.yml`
- Route inventory export: `ROUTES-BREADCRUMBS.{md,json,csv}`

## 2) Published Route Tree (Docusaurus)

- Marketing routes:
  - `/`
  - `/platform`
  - `/platform/soc`
  - `/platform/noc`
  - `/platform/grc`
  - `/platform/psa`
  - `/platform/kai`
  - `/contributors`
  - `/open-source`
  - `/contact`

- Documentation root:
  - `/docs/intro`

- Module roots:
  - `/docs/K-CORE-01_INFRASTRUCTURE`
  - `/docs/K-XRO-02_SUPER_AGENT`
  - `/docs/K-KAI-03_ORCHESTRATION`
  - `/docs/K-SOC-04_SECURITY`
  - `/docs/K-NOC-05_OPERATIONS`
  - `/docs/K-GRC-07_COMPLIANCE`
  - `/docs/K-PSA-06_BUSINESS`
  - `/docs/K-DEV-08_DEVELOPMENT`
  - `/docs/K-API-09_API_REFERENCE`
  - `/docs/K-ITIL-10_ITIL_MATRIX`

## 3) DocuNotion 1:1 Mapping Policy

- Only `docs/` content is synced from Notion.
- Website shell (`src/pages`, styling, nav, hero sections) remains code-managed.
- Auto-commit scope for sync is docs content only.
- Route inventory files are generated artifacts for sharing with Notion.

## 4) Production Workflow Decision

- Use a single Git-integrated Vercel project/domain for deployment.
- Do not trigger direct Vercel deploy API calls from sync workflow.
- Keep one deployment path to avoid domain/link sprawl.

## 5) Legacy Files

The following files are retained for historical context but are not canonical:
- `README-STRUCTURE.md`
- `MIGRATION-SUMMARY.md`

Use this file + `README.md` for current operational guidance.
