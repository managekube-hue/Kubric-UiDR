# Kubric-UiDR Phased Production Readiness Plan

Date: 2026-02-27  
Goal: Reach 100% production-ready state for multi-tenant, multi-agent MSP operations with customer portal, billing visibility, ticket automation, and real-time module monitoring.

---

## 0) Architecture Decision (Resolves Zammad vs ERPNext Conflict)

Decision:
- **ERPNext = customer-facing portal system of record** (customers, contracts, invoices, SLA views, portal UX).
- **Zammad = ITSM execution adapter** (ticket lifecycle + bidirectional updates), optional during migration.

Target model:
- Portal pages and billing visibility are served from Next.js + K-SVC APIs.
- ERPNext holds account/billing/customer master records.
- AI agents create/update incidents through a unified ticket abstraction; adapter routes to Zammad first, ERPNext helpdesk next.

Why this is correct:
- Your repo already has mature Zammad adapters (`internal/psa/zammad.go`) and billing pages in frontend.
- ERPNext is the better customer portal and billing workflow anchor.
- This avoids a rewrite while letting you phase migration safely.

---

## 1) What Copilot Can Do vs What You Must Do

## Copilot Can Do (Code + Config + Docs)
1. Implement ticket provider abstraction (`ticketing.Provider`) and route Zammad/ERPNext adapters behind it.
2. Build ERPNext API adapter (customer, invoice summary, ticket sync endpoints).
3. Wire Next.js tenant pages for modules: MDM, XDR, NDR, NPM, compliance, billing, agents.
4. Add real-time event visualization panels (NATS WebSocket + charts + status cards).
5. Add AI agent-to-ticket sync workflows (create/update/comment/close with correlation IDs).
6. Patch K8s/Docker manifests and CI workflows (immutable tags, port-correct health checks, env handling).
7. Add multi-tenant contract tests and RLS safety tests.
8. Add deployment runbooks, smoke tests, and release checklists.

## You Must Do (External Systems / Credentials / Runtime Ops)
1. Install local toolchains on build/deploy runners:
   - Node.js LTS + npm
   - Go toolchain
   - Rust toolchain
   - Docker + kubectl + kustomize
2. Provide production credentials/secrets:
   - ERPNext API host/key/secret
   - Zammad URL/token (if retained)
   - Supabase/PG credentials
   - Stripe keys + webhook secret
   - Auth provider secrets (NextAuth/OIDC)
3. Provision infrastructure and DNS/TLS:
   - Kubernetes cluster access
   - Registry credentials
   - Domain, cert-manager issuer, ingress DNS
4. Approve architecture and cutover choices:
   - Zammad-only, ERPNext-only, or dual-run transition period
5. Execute production approvals and change windows.

---

## 2) Phased Remediation Plan

## Phase 1 — Environment Unblock + Baseline (Day 1)
Outcomes:
- Toolchains available; build system unblocked.

Copilot actions:
- Add deterministic scripts (`make` + PowerShell) for frontend/API/agents build checks.
- Add preflight script for required CLIs/env vars.

You must do:
- Install missing CLIs and confirm they resolve in PATH.
- Provide `.env`/secret values for non-dev environments.

Exit criteria:
- `npm ci` and `npm run build` succeed in frontend.
- `go test ./...` and `cargo check --workspace` run on CI runners.

## Phase 2 — Portal Reliability + Visualization (Day 2-4)
Outcomes:
- Next.js portal supports reliable dashboarding and module visibility.

Copilot actions:
- Complete tenant module dashboards: MDM, XDR, NDR, NPM, billing, compliance, agents.
- Normalize API client layer and error boundaries.
- Add real-time feeds for alerts/agents/tickets with reconnection logic.

You must do:
- Confirm final KPI list per module and customer SLA metrics.

Exit criteria:
- Portal displays module health and live updates for each tenant.

## Phase 3 — Ticketing and AI Automation Layer (Day 4-7)
Outcomes:
- AI agents can open/update/resolve tickets with full audit chain.

Copilot actions:
- Implement `TicketProvider` interface.
- Keep Zammad adapter active; add ERPNext adapter.
- Route AI incident actions through unified adapter and store event correlation IDs.

You must do:
- Provide ERPNext and/or Zammad credentials.
- Confirm workflow ownership rules (auto-close, approval gates, escalation).

Exit criteria:
- End-to-end incident -> AI action -> ticket update -> portal visibility works in staging.

## Phase 4 — Billing + Customer Visibility (Day 7-10)
Outcomes:
- Customer portal billing and service visibility are accurate.

Copilot actions:
- Align portal billing pages with ERPNext/Stripe aggregation APIs.
- Implement contract-based usage views by module.
- Add invoice timeline and payment state cards.

You must do:
- Provide billing policy mapping and finance approval.

Exit criteria:
- Billing and module usage match source systems for at least 3 test tenants.

## Phase 5 — Production Hardening (Day 10-13)
Outcomes:
- Deployment safety and tenant isolation are enforceable.

Copilot actions:
- Patch CI/CD health checks and deployment manifests.
- Replace mutable image tags with immutable tags/digests.
- Add security and tenancy regression tests.

You must do:
- Approve rollback policy and production release windows.

Exit criteria:
- Green CI + passing smoke + rollback drill completed.

## Phase 6 — Cutover + Hypercare (Day 13-16)
Outcomes:
- Production launch with controlled migration path.

Copilot actions:
- Create cutover checklist and migration toggles.
- Add observability dashboards and incident runbooks.

You must do:
- Execute go-live approvals, monitor with on-call staff, confirm customer comms.

Exit criteria:
- 72-hour stable operations post-go-live.

---

## 3) Multi-Agent Interface Build Plan

UI scope (tenant context required):
- Agent Fleet: health, versions, heartbeat, stale/offline state.
- Module Matrix: MDM, XDR, NDR, NPM status by tenant.
- Incident Stream: real-time alerts + ticket links + remediation state.
- Billing Visibility: usage by module and invoice history.
- Automation Control: pending actions, approval queue, execution logs.

Back-end contracts:
- NATS subjects must remain tenant-scoped (`kubric.{tenant}.<domain>.<event>.v1`).
- Every automation action emits immutable audit events.
- Ticket updates must include external ticket IDs and workflow correlation IDs.

---

## 4) Deployment Checklist (Must Pass)

1. Toolchain checks pass.
2. Frontend build passes (`npm ci`, `npm run build`).
3. API/agent build and tests pass.
4. Kubernetes manifests validate and deploy in staging.
5. Tenant isolation tests pass.
6. Ticket sync tests pass (create/update/resolve roundtrip).
7. Billing reconciliation sample tenants pass.
8. SLO dashboard and alerting configured.

---

## 5) Immediate Next 48 Hours (Practical Sequence)

1. Install missing Node/npm and Go/Rust/Docker/K8s CLIs.
2. Run frontend build and capture errors.
3. Implement ticket provider abstraction + dual adapter (Zammad active, ERPNext ready).
4. Finalize portal module pages and real-time widgets.
5. Run staging e2e for incident -> AI -> ticket -> portal -> billing link.

---

## 6) Current Blockers in This Shell (Observed)

The active environment currently lacks these executables:
- `node`, `npm`
- `go`
- `cargo`, `rustc`
- `docker`, `kubectl`, `kustomize`

So I can write/fix code and docs now, but runtime build/deploy commands must wait until these are installed and available in PATH.
