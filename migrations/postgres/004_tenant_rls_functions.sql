-- =============================================================================
-- Kubric — PostgreSQL Migration 004
-- Tenant RLS helper functions and policies for tables created in 002 & 003.
-- =============================================================================
-- This migration bridges 003_oscal_ingestion.sql → 005_kai_missing_tables.sql.
-- It ensures the app_tenant_id() helper function, kubric_superuser role, and
-- RLS policies exist for all tables that carry a tenant_id column.
--
-- Prerequisite: 001_core_tables.sql (tenants table must exist)
-- =============================================================================

BEGIN;

-- ─── Helper: expose app.current_tenant_id safely ─────────────────────────────
-- Returns the current tenant ID or empty string if unset.
-- Used inside all RLS policy USING clauses.
-- CREATE OR REPLACE is idempotent — safe to re-run.
CREATE OR REPLACE FUNCTION app_tenant_id() RETURNS TEXT
    LANGUAGE sql STABLE PARALLEL SAFE AS
$$
    SELECT coalesce(current_setting('app.current_tenant_id', true), '')
$$;

-- ─── Helper: auto-update updated_at on row mutation ──────────────────────────
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

-- ─── Superuser / service role — bypasses all RLS ─────────────────────────────
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kubric_superuser') THEN
        CREATE ROLE kubric_superuser BYPASSRLS NOLOGIN;
        COMMENT ON ROLE kubric_superuser IS
            'Kubric service role — bypasses RLS for migrations and admin ops.';
    END IF;
END $$;

-- ─── RLS: contract_rate_tables (from 002_contract_rate_tables.sql) ───────────
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_name = 'contract_rate_tables') THEN
        EXECUTE 'ALTER TABLE contract_rate_tables ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE contract_rate_tables FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS contract_rate_tables_tenant_isolation ON contract_rate_tables';
        EXECUTE $p$
            CREATE POLICY contract_rate_tables_tenant_isolation ON contract_rate_tables
                AS PERMISSIVE FOR ALL
                USING (
                    app_tenant_id() = ''
                    OR tenant_id = app_tenant_id()
                )
        $p$;
        EXECUTE 'GRANT ALL ON contract_rate_tables TO kubric_superuser';
    END IF;
END $$;

-- ─── RLS: oscal_controls (from 003_oscal_ingestion.sql) — no tenant_id ──────
-- oscal_controls is a global reference table (no tenant_id column).
-- No RLS needed.

-- ─── RLS: oscal_control_enhancements — no tenant_id ─────────────────────────
-- Also a global reference table. No RLS needed.

-- ─── RLS: tenant_control_status (from 003_oscal_ingestion.sql) ──────────────
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_name = 'tenant_control_status') THEN
        EXECUTE 'ALTER TABLE tenant_control_status ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE tenant_control_status FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS tenant_control_status_tenant_isolation ON tenant_control_status';
        EXECUTE $p$
            CREATE POLICY tenant_control_status_tenant_isolation ON tenant_control_status
                AS PERMISSIVE FOR ALL
                USING (
                    app_tenant_id() = ''
                    OR tenant_id = app_tenant_id()
                )
        $p$;
        EXECUTE 'GRANT ALL ON tenant_control_status TO kubric_superuser';
    END IF;
END $$;

COMMIT;
