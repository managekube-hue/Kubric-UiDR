# Kubric-UiDR — Canonical Project Status

Last updated: 2026-02-27  
Current baseline commit: `bc6a737445a8`  
Baseline tags on commit: `grc-200fw-baseline`, `full-baseline-2026-02-27`

---

## Purpose

This is the **single source of truth** for current repo state.

Use this file first for:
- current readiness
- latest verified assertions
- active priorities
- handoff context (solo continuity between sessions)

All other audit/remediation docs are historical snapshots unless this file says otherwise.

---

## Current State (2026-02-27)

### Verified baseline

- `ops-batch-06` passes end-to-end (all assertions green in latest run).
- Rust dependency cleanup completed for unused/deferred crates.
- eBPF object handling is wired in code and image build path.
- Windows hook no-op fallback removed; process events emitted.
- PerfTrace disk I/O no longer hardcoded to zero.
- CISO-Assistant GRC integration wired across API, service bridge, NATS subject docs, and frontend API client.
- 200-framework GRC registry baseline is present and validated.

### Runtime/deploy wiring confirmed

- `kic` service wired in compose on `:8082` with `KAI_RAG_URL`.
- Distroless KIC runtime image present in `build/kic/Dockerfile`.
- `Dockerfile.api` contains KIC runtime stage.

### Known open item outside baseline gate

- Local `docker build -f Dockerfile.web` has recent failing attempts in terminal history and should be handled in a separate focused pass (not blocking current `ops-batch-06` baseline verification).

---

## Review Entry Points (minimal)

For humans or AI reviewers, start here in order:

1. This file (`docs/PROJECT-STATUS.md`)
2. `scripts/bootstrap/ops-batch-06-audit-verify.ps1`
3. `docs/AUDIT-REMEDIATION-2026-02-27.md` (historical implementation detail)

---

## Historical Snapshot Docs

- `docs/THOROUGH-AUDIT-2026-02-27.md` — original broad audit snapshot.
- `docs/PRODUCTION-REMEDIATION-PLAN.md` — long-form remediation program plan.
- `docs/AUDIT-REMEDIATION-2026-02-27.md` — remediation implementation record.

These are retained for traceability but should not be treated as the day-to-day source of truth.

---

## Update Protocol (keep repo understandable)

When new work is completed:

1. Update this file first (state, baseline commit/tag, pass/fail summary).
2. Add or move baseline tag(s) to the new commit.
3. Run `ops-batch-06` and record result summary here.
4. Only then update deep-dive docs if needed.

This keeps the repo easy to explain without forcing reviewers through multiple large documents.
