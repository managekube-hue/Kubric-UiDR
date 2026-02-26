-- =============================================================================
-- Migration 002 — Tenant Row-Level Security Policies (L0-7 RLS activation)
-- =============================================================================
-- Activates PostgreSQL RLS on all business tables from migration 001.
-- Works in conjunction with internal/db/tenant_tx.go which sets:
--   SET LOCAL app.current_tenant_id = '<tenant_id>'
-- before every data-modifying transaction.
--
-- Policy design:
--   • When app.current_tenant_id IS NULL or '' (admin/migration/service ops):
--     ALL rows are visible — no tenant filter applied.
--   • When app.current_tenant_id IS SET to a tenant ID:
--     Only rows belonging to that tenant are visible.
--   • kubric_superuser PostgreSQL role bypasses all policies (BYPASSRLS).
--
-- Run:  psql $KUBRIC_DATABASE_URL -f 002_tenant_rls.up.sql
-- =============================================================================

BEGIN;

-- ─── Helper: expose app.current_tenant_id safely ─────────────────────────────
-- Returns the current tenant ID or empty string if unset.
-- Used inside all RLS policy USING clauses.
CREATE OR REPLACE FUNCTION app_tenant_id() RETURNS TEXT
    LANGUAGE sql STABLE PARALLEL SAFE AS
$$
    SELECT coalesce(current_setting('app.current_tenant_id', true), '')
$$;

-- ─── Superuser / service role — bypasses all RLS ─────────────────────────────
-- Create the role if it doesn't already exist (idempotent).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kubric_superuser') THEN
        CREATE ROLE kubric_superuser BYPASSRLS NOLOGIN;
        COMMENT ON ROLE kubric_superuser IS
            'Kubric service role — bypasses RLS for migrations and admin ops.';
    END IF;
END $$;

-- ─── Table 1: kubric_tenants ──────────────────────────────────────────────────
-- Tenants can only see their own record.
-- Service role (app_tenant_id()='') sees all tenants.
ALTER TABLE kubric_tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubric_tenants FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kubric_tenants_tenant_isolation ON kubric_tenants;
CREATE POLICY kubric_tenants_tenant_isolation ON kubric_tenants
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 2: vdr_findings ───────────────────────────────────────────────────
ALTER TABLE vdr_findings ENABLE ROW LEVEL SECURITY;
ALTER TABLE vdr_findings FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS vdr_findings_tenant_isolation ON vdr_findings;
CREATE POLICY vdr_findings_tenant_isolation ON vdr_findings
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 3: kic_assessments ────────────────────────────────────────────────
ALTER TABLE kic_assessments ENABLE ROW LEVEL SECURITY;
ALTER TABLE kic_assessments FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kic_assessments_tenant_isolation ON kic_assessments;
CREATE POLICY kic_assessments_tenant_isolation ON kic_assessments
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 4: noc_clusters ───────────────────────────────────────────────────
ALTER TABLE noc_clusters ENABLE ROW LEVEL SECURITY;
ALTER TABLE noc_clusters FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS noc_clusters_tenant_isolation ON noc_clusters;
CREATE POLICY noc_clusters_tenant_isolation ON noc_clusters
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 5: noc_agents ─────────────────────────────────────────────────────
ALTER TABLE noc_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE noc_agents FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS noc_agents_tenant_isolation ON noc_agents;
CREATE POLICY noc_agents_tenant_isolation ON noc_agents
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 6: kai_triage_results ─────────────────────────────────────────────
ALTER TABLE kai_triage_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE kai_triage_results FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS kai_triage_results_tenant_isolation ON kai_triage_results;
CREATE POLICY kai_triage_results_tenant_isolation ON kai_triage_results
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 7: feature_flags ──────────────────────────────────────────────────
-- Tenants can only read/modify their own flags.
-- Global flags (tenant_id = '') are readable by all — no tenant context needed.
ALTER TABLE feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS feature_flags_tenant_isolation ON feature_flags;
CREATE POLICY feature_flags_tenant_isolation ON feature_flags
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = ''              -- global flag visible to everyone
        OR tenant_id = app_tenant_id()
    );

-- ─── Table 8: agent_enrollment ───────────────────────────────────────────────
ALTER TABLE agent_enrollment ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_enrollment FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS agent_enrollment_tenant_isolation ON agent_enrollment;
CREATE POLICY agent_enrollment_tenant_isolation ON agent_enrollment
    AS PERMISSIVE
    FOR ALL
    USING (
        app_tenant_id() = ''
        OR tenant_id = app_tenant_id()
    );

-- ─── Grant kubric_superuser bypass on all tables ─────────────────────────────
GRANT ALL ON kubric_tenants    TO kubric_superuser;
GRANT ALL ON vdr_findings       TO kubric_superuser;
GRANT ALL ON kic_assessments    TO kubric_superuser;
GRANT ALL ON noc_clusters       TO kubric_superuser;
GRANT ALL ON noc_agents         TO kubric_superuser;
GRANT ALL ON kai_triage_results TO kubric_superuser;
GRANT ALL ON feature_flags      TO kubric_superuser;
GRANT ALL ON agent_enrollment   TO kubric_superuser;

COMMIT;
