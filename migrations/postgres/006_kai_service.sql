-- =============================================================================
-- Kubric — PostgreSQL Migration 006
-- K-KAI service tables: incidents, agent sessions, and long-running operations.
-- =============================================================================
-- These three tables back the KAI orchestration layer (Layer 2):
--   kai_incidents       — correlated incidents from the detection engine
--   kai_agent_sessions  — per-invocation records for every KAI persona
--   kai_operations      — Temporal workflow execution tracker
--
-- Apply with:
--   psql $DATABASE_URL -f migrations/postgres/006_kai_service.sql
-- Or via Supabase dashboard → SQL Editor → Run
--
-- Prerequisite migrations:
--   001_core_tables.sql          — set_updated_at()
--   002_contract_rate_tables.sql
--   003_oscal_ingestion.sql
--   004 (db/migrations folder)   — app_tenant_id(), kubric_superuser role
--   005_kai_missing_tables.sql   — kai_alerts, kai_patch_jobs, kai_drift_events
--
-- NOTE: tenant_id is TEXT (not UUID) throughout this migration to match the
-- KAI Python layer which represents tenants as opaque string identifiers
-- (e.g. "acme-corp", "tenant-abc123") rather than UUIDs.
-- =============================================================================

BEGIN;

-- ─── Ensure helper functions are present (idempotent re-definitions) ─────────
-- set_updated_at() is first defined in 001_core_tables.sql.
-- app_tenant_id()  is first defined in db/migrations/002_tenant_rls.up.sql.
-- Both are recreated here as CREATE OR REPLACE so this file is self-contained
-- when replayed in a clean schema.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION app_tenant_id() RETURNS TEXT
    LANGUAGE sql STABLE PARALLEL SAFE AS
$$
    SELECT coalesce(current_setting('app.current_tenant_id', true), '')
$$;

-- Ensure the kubric_superuser bypass role exists.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kubric_superuser') THEN
        CREATE ROLE kubric_superuser BYPASSRLS NOLOGIN;
        COMMENT ON ROLE kubric_superuser IS
            'Kubric service role — bypasses RLS for migrations and admin ops.';
    END IF;
END $$;

-- =============================================================================
-- TABLE 1 — kai_incidents
-- Correlated incidents produced by the KAI detection engine.  Each row
-- represents a deduplicated, enriched incident that may span multiple raw
-- events (stored as the event_ids array).  TheHive and Shuffle integration
-- IDs are written back by KAI-COMM once external cases are created.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_incidents (
    id                    UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id             TEXT        NOT NULL,
    title                 TEXT        NOT NULL,
    severity              TEXT        NOT NULL
                          CHECK (severity IN ('CRITICAL','HIGH','MEDIUM','LOW','INFO')),
    status                TEXT        NOT NULL DEFAULT 'open'
                          CHECK (status IN ('open','investigating','resolved','false_positive')),
    source                TEXT        NOT NULL,             -- correlation_engine|falco|wazuh|sigma|yara
    event_ids             TEXT[]      NOT NULL DEFAULT '{}',
    mitre_techniques      TEXT[]      NOT NULL DEFAULT '{}',
    assigned_to           TEXT,
    thehive_case_id       TEXT,
    shuffle_execution_id  TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at           TIMESTAMPTZ
);

-- Indexes — kai_incidents
CREATE INDEX IF NOT EXISTS idx_kai_incidents_tenant_status
    ON kai_incidents (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_kai_incidents_tenant_created
    ON kai_incidents (tenant_id, created_at DESC);

-- Fast path for the most common incident dashboard query: all open incidents per tenant
CREATE INDEX IF NOT EXISTS idx_kai_incidents_tenant_open
    ON kai_incidents (tenant_id, created_at DESC)
    WHERE status = 'open';

-- =============================================================================
-- TABLE 2 — kai_agent_sessions
-- Tracks every invocation of a KAI persona agent.  Written at the start of
-- each agent execution (status=pending/running) and updated on completion.
-- Enables per-tenant audit trails, latency profiling, and cost attribution
-- by model_used.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_agent_sessions (
    id               UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id        TEXT        NOT NULL,
    persona          TEXT        NOT NULL
                     CHECK (persona IN (
                         'triage','analyst','hunter','keeper','risk','invest',
                         'sentinel','foresight','house','bill','comm','deploy','simulate'
                     )),
    trigger_subject  TEXT        NOT NULL,               -- NATS subject or API path that triggered the session
    payload          JSONB       NOT NULL DEFAULT '{}',
    result           JSONB,
    status           TEXT        NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','running','completed','failed')),
    duration_ms      INTEGER,
    model_used       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ
);

-- Indexes — kai_agent_sessions
CREATE INDEX IF NOT EXISTS idx_kai_agent_sessions_tenant_persona
    ON kai_agent_sessions (tenant_id, persona);

CREATE INDEX IF NOT EXISTS idx_kai_agent_sessions_tenant_created
    ON kai_agent_sessions (tenant_id, created_at DESC);

-- Narrow partial index for the worker polling query: in-flight sessions only
CREATE INDEX IF NOT EXISTS idx_kai_agent_sessions_inflight
    ON kai_agent_sessions (status, created_at)
    WHERE status IN ('pending','running');

-- =============================================================================
-- TABLE 3 — kai_operations
-- Long-running Temporal workflow execution tracker.  Each row represents a
-- single workflow run.  The Temporal workflow ID and run ID are written once
-- the workflow is accepted; status is advanced by the activity worker.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_operations (
    id                    UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id             TEXT        NOT NULL,
    workflow_type         TEXT        NOT NULL,           -- PatchWorkflow|RemediationWorkflow|SBOMWorkflow|BillingWorkflow
    temporal_workflow_id  TEXT        UNIQUE,
    temporal_run_id       TEXT,
    status                TEXT        NOT NULL DEFAULT 'pending',
    input                 JSONB       NOT NULL DEFAULT '{}',
    result                JSONB,
    error                 TEXT,
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes — kai_operations
CREATE INDEX IF NOT EXISTS idx_kai_operations_tenant_status
    ON kai_operations (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_kai_operations_tenant_created
    ON kai_operations (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kai_operations_workflow_type
    ON kai_operations (tenant_id, workflow_type);

-- Covering index for in-flight workflow lookup by temporal worker
CREATE INDEX IF NOT EXISTS idx_kai_operations_inflight
    ON kai_operations (status, created_at)
    WHERE status IN ('pending','running');

-- =============================================================================
-- Auto-update updated_at triggers
-- Applied to all three tables which carry an updated_at column.
-- Matches the pattern established in 005_kai_missing_tables.sql.
-- =============================================================================
DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['kai_incidents','kai_agent_sessions','kai_operations'] LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS tg_%I_updated_at ON %I;
            CREATE TRIGGER tg_%I_updated_at
                BEFORE UPDATE ON %I
                FOR EACH ROW EXECUTE FUNCTION set_updated_at();
        ', t, t, t, t);
    END LOOP;
END;
$$;

-- =============================================================================
-- Row Level Security (RLS) — tenant isolation
-- Policy design follows db/migrations/002_tenant_rls.up.sql and 005:
--   • app_tenant_id() = '' (service/admin context) → all rows visible.
--   • app_tenant_id() set to a tenant string → only matching rows visible.
--   • kubric_superuser role bypasses all policies (BYPASSRLS).
--
-- NOTE: tenant_id is TEXT here (not UUID), so the cast in 005 is not needed;
-- direct equality comparison is used.
-- =============================================================================

-- ─── kai_incidents ────────────────────────────────────────────────────────────
ALTER TABLE kai_incidents ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_incidents FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_incidents_tenant_isolation ON kai_incidents;
CREATE POLICY kai_incidents_tenant_isolation ON kai_incidents
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── kai_agent_sessions ───────────────────────────────────────────────────────
ALTER TABLE kai_agent_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_agent_sessions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_agent_sessions_tenant_isolation ON kai_agent_sessions;
CREATE POLICY kai_agent_sessions_tenant_isolation ON kai_agent_sessions
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── kai_operations ───────────────────────────────────────────────────────────
ALTER TABLE kai_operations ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_operations FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_operations_tenant_isolation ON kai_operations;
CREATE POLICY kai_operations_tenant_isolation ON kai_operations
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- =============================================================================
-- Grant kubric_superuser full access (bypasses RLS by role attribute, but
-- explicit GRANT ensures the role can run DDL and DML in all contexts).
-- =============================================================================
GRANT ALL ON kai_incidents      TO kubric_superuser;
GRANT ALL ON kai_agent_sessions TO kubric_superuser;
GRANT ALL ON kai_operations     TO kubric_superuser;

COMMIT;

-- =============================================================================
-- Python / Go source files that reference each new table
-- =============================================================================
--
-- kai_incidents
--   kai/agents/triage.py           — inserts row on triage completion
--   kai/agents/analyst.py          — reads open incidents for correlation
--   internal/noc/handler_agent.go  — reads/writes incidents via REST API
--
-- kai_agent_sessions
--   kai/core/subscriber.py         — inserts pending session before dispatch,
--                                    updates completed_at + result on finish
--   kai/api/main.py                — POST /v1/triage, /v1/remediate, etc.
--                                    each handler creates + closes a session row
--
-- kai_operations
--   kai/workflows/billing.py       — submit_billing() inserts + updates rows
--   internal/noc/handler_agent.go  — PatchWorkflow + RemediationWorkflow runners
--                                    write temporal_workflow_id / temporal_run_id
-- =============================================================================
