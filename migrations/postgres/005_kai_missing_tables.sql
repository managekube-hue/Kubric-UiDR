-- =============================================================================
-- Kubric — PostgreSQL Migration 005
-- K-KAI missing tables: alerts, patch jobs, drift events, and baselines.
-- =============================================================================
-- These four tables are referenced throughout the KAI orchestration layer but
-- had no CREATE TABLE definition anywhere in the schema prior to this file.
--
-- Apply with:
--   psql $DATABASE_URL -f migrations/postgres/005_kai_missing_tables.sql
-- Or via Supabase dashboard → SQL Editor → Run
--
-- Prerequisite migrations:
--   001_core_tables.sql      — tenants, users (FK targets), set_updated_at()
--   002_contract_rate_tables.sql
--   003_oscal_ingestion.sql
--   004 (db/migrations folder) — app_tenant_id(), kubric_superuser role
--
-- NOTE: pgvector extension is required by kai_embeddings defined in
-- K-KAI-RAG-001_vector_search.sql.  The extension is declared below with
-- IF NOT EXISTS so this statement is fully idempotent regardless of run order.
-- =============================================================================

BEGIN;

-- ─── pgvector extension (idempotent) ─────────────────────────────────────────
-- Required by kai_embeddings (K-KAI-RAG-001_vector_search.sql).
-- Safe to run here even if that migration has already been applied; IF NOT
-- EXISTS guarantees a no-op on subsequent runs.
CREATE EXTENSION IF NOT EXISTS vector;

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
-- TABLE 1 — kai_alerts
-- Security alerts ingested from NATS subject security.alert.v1 and from the
-- coresec Sigma/YARA engine.  The triage agent (K-KAI-TR-001) and health score
-- publisher (K-KAI-SEN-001) are the primary readers and writers.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_alerts (
    id              UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       TEXT        NOT NULL REFERENCES kubric_tenants(tenant_id) ON DELETE CASCADE,
    alert_source    TEXT        NOT NULL,                        -- sigma_match|yara_match|behavioral|anomaly|threat_intel
    severity        TEXT        NOT NULL CHECK (severity IN ('critical','high','medium','low','info')),
    title           TEXT        NOT NULL,
    description     TEXT,
    mitre_tactic    TEXT,
    mitre_technique TEXT,
    asset_id        UUID,
    cve_id          TEXT,
    raw_event       JSONB,
    status          TEXT        NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','in_progress','resolved','false_positive')),
    assigned_to     UUID        REFERENCES users(id),
    incident_id     UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at       TIMESTAMPTZ
);

-- Indexes — kai_alerts
CREATE INDEX IF NOT EXISTS idx_kai_alerts_tenant        ON kai_alerts (tenant_id);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_severity      ON kai_alerts (severity);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_status        ON kai_alerts (status);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_asset         ON kai_alerts (asset_id);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_cve           ON kai_alerts (cve_id);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_source        ON kai_alerts (alert_source);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_incident      ON kai_alerts (incident_id);
CREATE INDEX IF NOT EXISTS idx_kai_alerts_assigned      ON kai_alerts (assigned_to);
-- Covering index for the most common triage query: open alerts per tenant sorted by time
CREATE INDEX IF NOT EXISTS idx_kai_alerts_tenant_open   ON kai_alerts (tenant_id, created_at DESC)
    WHERE status = 'open';

-- =============================================================================
-- TABLE 2 — kai_patch_jobs
-- Patch workflow jobs tracked by Temporal (K-KAI-WF-TEMP-001_patch_workflow.go).
-- Each row represents a single patching workflow execution for one asset.
-- Written by the Temporal activity workers; read by the housekeeper (K-KAI-HS-001)
-- and the drift event table via FK.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_patch_jobs (
    id                   UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id            TEXT    NOT NULL REFERENCES kubric_tenants(tenant_id) ON DELETE CASCADE,
    asset_id             UUID    NOT NULL,
    cve_ids              TEXT[]  NOT NULL DEFAULT '{}',
    patch_ids            TEXT[]  NOT NULL DEFAULT '{}',
    status               TEXT    NOT NULL DEFAULT 'pending'
                         CHECK (status IN (
                             'pending','validating','snapshotting','applying',
                             'verifying','complete','failed','rolled_back'
                         )),
    temporal_workflow_id TEXT    UNIQUE,
    temporal_run_id      TEXT,
    snapshot_id          TEXT,
    applied_by           UUID    REFERENCES users(id),
    started_at           TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    error_message        TEXT,
    rollback_reason      TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes — kai_patch_jobs
CREATE INDEX IF NOT EXISTS idx_kai_patch_jobs_tenant         ON kai_patch_jobs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_kai_patch_jobs_asset          ON kai_patch_jobs (asset_id);
CREATE INDEX IF NOT EXISTS idx_kai_patch_jobs_status         ON kai_patch_jobs (status);
CREATE INDEX IF NOT EXISTS idx_kai_patch_jobs_applied_by     ON kai_patch_jobs (applied_by);
-- Covering index for active workflow lookup by Temporal worker
CREATE INDEX IF NOT EXISTS idx_kai_patch_jobs_tenant_pending ON kai_patch_jobs (tenant_id, created_at DESC)
    WHERE status IN ('pending','validating','snapshotting','applying','verifying');
-- GIN index for CVE array containment queries (e.g. WHERE cve_ids @> '{CVE-2023-12345}')
CREATE INDEX IF NOT EXISTS idx_kai_patch_jobs_cve_ids        ON kai_patch_jobs USING gin (cve_ids);

-- =============================================================================
-- TABLE 3 — kai_drift_events
-- GRC compliance drift events streamed from NATS subject grc.drift.v1.
-- Written by the coresec compliance scanner; read by the housekeeper
-- (K-KAI-HS-001) for remediation scheduling and by the health score publisher
-- (K-KAI-SEN-001) for compliance sub-score computation.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_drift_events (
    id             UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id      TEXT        NOT NULL REFERENCES kubric_tenants(tenant_id) ON DELETE CASCADE,
    asset_id       UUID        NOT NULL,
    control_id     TEXT        NOT NULL,
    framework      TEXT        NOT NULL
                   CHECK (framework IN ('cis','nist','soc2','pci_dss','hipaa')),
    expected_value JSONB       NOT NULL,
    actual_value   JSONB       NOT NULL,
    severity       TEXT        NOT NULL
                   CHECK (severity IN ('critical','high','medium','low')),
    status         TEXT        NOT NULL DEFAULT 'open'
                   CHECK (status IN ('open','remediated','accepted','false_positive')),
    remediated_by  UUID        REFERENCES users(id),
    patch_job_id   UUID        REFERENCES kai_patch_jobs(id),
    detected_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    remediated_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes — kai_drift_events
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_tenant      ON kai_drift_events (tenant_id);
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_asset       ON kai_drift_events (asset_id);
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_status      ON kai_drift_events (status);
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_framework   ON kai_drift_events (framework);
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_control     ON kai_drift_events (control_id);
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_patch_job   ON kai_drift_events (patch_job_id);
-- Covering index used by health score publisher: open drift in last N hours per tenant
CREATE INDEX IF NOT EXISTS idx_kai_drift_events_tenant_open ON kai_drift_events (tenant_id, created_at DESC)
    WHERE status = 'open';

-- =============================================================================
-- TABLE 4 — kai_baselines
-- Asset configuration baselines captured by the housekeeper (K-KAI-HS-001)
-- for drift detection.  Versioned with a self-referencing previous_id FK so
-- baseline history is preserved.  The criticality check (K-KAI-HS-003) and
-- rollback manager (K-KAI-HS-004) read the current baseline before applying
-- any change.
-- =============================================================================
CREATE TABLE IF NOT EXISTS kai_baselines (
    id               UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id        TEXT    NOT NULL REFERENCES kubric_tenants(tenant_id) ON DELETE CASCADE,
    asset_id         UUID    NOT NULL,
    baseline_type    TEXT    NOT NULL
                     CHECK (baseline_type IN (
                         'os_config','network_config','package_list',
                         'service_list','cis_benchmark'
                     )),
    framework        TEXT,
    baseline_data    JSONB   NOT NULL,
    checksum         TEXT    NOT NULL,                           -- SHA-256 of baseline_data
    version          INTEGER NOT NULL DEFAULT 1,
    captured_by      TEXT    NOT NULL DEFAULT 'housekeeper',
    captured_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_current       BOOLEAN NOT NULL DEFAULT true,
    previous_id      UUID    REFERENCES kai_baselines(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes — kai_baselines
CREATE INDEX IF NOT EXISTS idx_kai_baselines_tenant        ON kai_baselines (tenant_id);
CREATE INDEX IF NOT EXISTS idx_kai_baselines_asset         ON kai_baselines (asset_id);
CREATE INDEX IF NOT EXISTS idx_kai_baselines_type          ON kai_baselines (baseline_type);
CREATE INDEX IF NOT EXISTS idx_kai_baselines_checksum      ON kai_baselines (checksum);
-- Fast lookup of the active baseline for an asset (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_kai_baselines_current       ON kai_baselines (tenant_id, asset_id)
    WHERE is_current = true;

-- =============================================================================
-- Auto-update updated_at triggers
-- Only applied to tables that carry an updated_at column:
--   kai_alerts, kai_patch_jobs
-- (kai_drift_events and kai_baselines are append-only; they have no updated_at.)
-- =============================================================================
DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['kai_alerts','kai_patch_jobs'] LOOP
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
-- Policy design follows db/migrations/002_tenant_rls.up.sql:
--   • app_tenant_id() = '' (service/admin context) → all rows visible.
--   • app_tenant_id() set to a UUID → only matching tenant rows visible.
--   • kubric_superuser role bypasses all policies (BYPASSRLS).
-- =============================================================================

-- ─── kai_alerts ──────────────────────────────────────────────────────────────
ALTER TABLE kai_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_alerts FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_alerts_tenant_isolation ON kai_alerts;
CREATE POLICY kai_alerts_tenant_isolation ON kai_alerts
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id::TEXT = app_tenant_id()
    );

-- ─── kai_patch_jobs ──────────────────────────────────────────────────────────
ALTER TABLE kai_patch_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_patch_jobs FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_patch_jobs_tenant_isolation ON kai_patch_jobs;
CREATE POLICY kai_patch_jobs_tenant_isolation ON kai_patch_jobs
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id::TEXT = app_tenant_id()
    );

-- ─── kai_drift_events ────────────────────────────────────────────────────────
ALTER TABLE kai_drift_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_drift_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_drift_events_tenant_isolation ON kai_drift_events;
CREATE POLICY kai_drift_events_tenant_isolation ON kai_drift_events
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id::TEXT = app_tenant_id()
    );

-- ─── kai_baselines ───────────────────────────────────────────────────────────
ALTER TABLE kai_baselines ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_baselines FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_baselines_tenant_isolation ON kai_baselines;
CREATE POLICY kai_baselines_tenant_isolation ON kai_baselines
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id::TEXT = app_tenant_id()
    );

-- =============================================================================
-- Grant kubric_superuser full access (bypasses RLS by role attribute, but
-- explicit GRANT ensures the role can run DDL and DML in all contexts).
-- =============================================================================
GRANT ALL ON kai_alerts       TO kubric_superuser;
GRANT ALL ON kai_patch_jobs   TO kubric_superuser;
GRANT ALL ON kai_drift_events TO kubric_superuser;
GRANT ALL ON kai_baselines    TO kubric_superuser;

COMMIT;

-- =============================================================================
-- Python / Go source files that reference each new table
-- =============================================================================
--
-- kai_alerts
--   03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/
--     K-KAI-TR-001_triage_agent.py         — consumes NATS security.alert.v1;
--                                            downstream alert rows feed this table
--   03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/
--     K-KAI-SEN-001_health_score_publisher.py — queries open alert counts per tenant
--                                               for security sub-score computation
--
-- kai_patch_jobs
--   K-KAI-WF-TEMP-001_patch_workflow.go    — Temporal workflow that inserts rows,
--                                            advances status through the state
--                                            machine, and records snapshot_id /
--                                            temporal_workflow_id / temporal_run_id
--   03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/
--     K-KAI-HS-001_housekeeper.py          — polls pending patch jobs via
--                                            /api/patches?status=pending (NOC API
--                                            which reads this table)
--
-- kai_drift_events
--   03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/
--     K-KAI-SEN-001_health_score_publisher.py — SELECT COUNT(*) FROM kai_drift_events
--                                               WHERE tenant_id=$1
--                                               AND created_at > NOW()-INTERVAL '24 hours'
--   03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/
--     K-KAI-HS-001_housekeeper.py          — polls open drift events via
--                                            /api/clusters/drift?status=pending
--
-- kai_baselines
--   03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/
--     K-KAI-HS-003_criticality_check.py    — reads current baseline to validate
--                                            whether a proposed change is safe
--                                            given asset criticality tier
--     K-KAI-HS-004_rollback.py             — reads latest baseline snapshot
--                                            (kai_rollback_snapshots is the
--                                            runtime snapshot store; kai_baselines
--                                            holds the canonical config baseline
--                                            used to detect drift post-rollback)
-- =============================================================================
