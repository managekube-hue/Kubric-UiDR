# Notion Sync Guardrails

This repository intentionally separates:

- **Website shell (manual control):** `docs/src/pages/**`, `docs/src/components/**`, `docs/src/css/**`
- **Notion-synced docs content:** `docs/docs/**`

## Why

Hero sections, headlines, navigation framing, and onboarding presentation are managed manually for brand and UX consistency.

Operational documentation content can evolve through Notion sync.

## Enforcement

Both workflows are configured to auto-commit only `docs/docs/**`:

- `.github/workflows/docu-notion.yml`
- `.github/workflows/sync-notion-docs.yml`

This prevents Notion automation from overwriting homepage/contributor/platform/open-source/contact hero and headline layers.
