-- PostgreSQL Row-Level Security policies for multi-tenant isolation
-- Module: K-DL-PG-002
-- Apply to: assets, user_access_reviews, kai_alerts, kai_patch_jobs, kai_incidents, privileged_access_log

-- ---------------------------------------------------------------------------
-- Helper function: return current tenant context from session variable
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION app_tenant_id() RETURNS TEXT
    LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')
$$;

-- ---------------------------------------------------------------------------
-- Helper function: check whether the current role bypasses RLS
-- Used in USING clauses to allow service roles full access without setting session var.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION is_service_role() RETURNS BOOLEAN
    LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT current_user = ANY (ARRAY['kubric_superuser', 'kubric_migration', 'postgres'])
$$;

-- ---------------------------------------------------------------------------
-- ASSETS
-- ---------------------------------------------------------------------------
ALTER TABLE assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS assets_tenant_isolation ON assets;
CREATE POLICY assets_tenant_isolation ON assets
    AS PERMISSIVE FOR ALL
    USING (
        is_service_role()
        OR app_tenant_id() IS NULL
        OR tenant_id::TEXT = app_tenant_id()
    );

GRANT ALL ON assets TO kubric_superuser;
GRANT SELECT, INSERT, UPDATE, DELETE ON assets TO kubric_app;

-- ---------------------------------------------------------------------------
-- USER_ACCESS_REVIEWS
-- ---------------------------------------------------------------------------
ALTER TABLE user_access_reviews ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_access_reviews FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS uar_tenant_isolation ON user_access_reviews;
CREATE POLICY uar_tenant_isolation ON user_access_reviews
    AS PERMISSIVE FOR ALL
    USING (
        is_service_role()
        OR app_tenant_id() IS NULL
        OR tenant_id::TEXT = app_tenant_id()
    );

GRANT ALL ON user_access_reviews TO kubric_superuser;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_access_reviews TO kubric_app;

-- ---------------------------------------------------------------------------
-- PRIVILEGED_ACCESS_LOG (read-only for app role once written; append via INSERT)
-- ---------------------------------------------------------------------------
ALTER TABLE privileged_access_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE privileged_access_log FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pal_tenant_isolation ON privileged_access_log;
CREATE POLICY pal_tenant_isolation ON privileged_access_log
    AS PERMISSIVE FOR ALL
    USING (
        is_service_role()
        OR app_tenant_id() IS NULL
        OR tenant_id::TEXT = app_tenant_id()
    );

GRANT ALL ON privileged_access_log TO kubric_superuser;
GRANT SELECT, INSERT ON privileged_access_log TO kubric_app;

-- ---------------------------------------------------------------------------
-- KAI_ALERTS (if table exists; idempotent via DO block)
-- ---------------------------------------------------------------------------
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'kai_alerts') THEN
        EXECUTE 'ALTER TABLE kai_alerts ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE kai_alerts FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS kai_alerts_tenant_isolation ON kai_alerts';
        EXECUTE $p$
            CREATE POLICY kai_alerts_tenant_isolation ON kai_alerts
                AS PERMISSIVE FOR ALL
                USING (
                    is_service_role()
                    OR app_tenant_id() IS NULL
                    OR tenant_id::TEXT = app_tenant_id()
                )
        $p$;
        EXECUTE 'GRANT ALL ON kai_alerts TO kubric_superuser';
        EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON kai_alerts TO kubric_app';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- KAI_PATCH_JOBS
-- ---------------------------------------------------------------------------
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'kai_patch_jobs') THEN
        EXECUTE 'ALTER TABLE kai_patch_jobs ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE kai_patch_jobs FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS kai_patch_jobs_tenant_isolation ON kai_patch_jobs';
        EXECUTE $p$
            CREATE POLICY kai_patch_jobs_tenant_isolation ON kai_patch_jobs
                AS PERMISSIVE FOR ALL
                USING (
                    is_service_role()
                    OR app_tenant_id() IS NULL
                    OR tenant_id::TEXT = app_tenant_id()
                )
        $p$;
        EXECUTE 'GRANT ALL ON kai_patch_jobs TO kubric_superuser';
        EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON kai_patch_jobs TO kubric_app';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- KAI_INCIDENTS
-- ---------------------------------------------------------------------------
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'kai_incidents') THEN
        EXECUTE 'ALTER TABLE kai_incidents ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE kai_incidents FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS kai_incidents_tenant_isolation ON kai_incidents';
        EXECUTE $p$
            CREATE POLICY kai_incidents_tenant_isolation ON kai_incidents
                AS PERMISSIVE FOR ALL
                USING (
                    is_service_role()
                    OR app_tenant_id() IS NULL
                    OR tenant_id::TEXT = app_tenant_id()
                )
        $p$;
        EXECUTE 'GRANT ALL ON kai_incidents TO kubric_superuser';
        EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON kai_incidents TO kubric_app';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- OSCAL_ASSESSMENT_RESULTS
-- ---------------------------------------------------------------------------
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = 'public' AND tablename = 'oscal_assessment_results') THEN
        EXECUTE 'ALTER TABLE oscal_assessment_results ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE oscal_assessment_results FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS oscal_results_tenant_isolation ON oscal_assessment_results';
        EXECUTE $p$
            CREATE POLICY oscal_results_tenant_isolation ON oscal_assessment_results
                AS PERMISSIVE FOR ALL
                USING (
                    is_service_role()
                    OR app_tenant_id() IS NULL
                    OR tenant_id::TEXT = app_tenant_id()
                )
        $p$;
        EXECUTE 'GRANT ALL ON oscal_assessment_results TO kubric_superuser';
        EXECUTE 'GRANT SELECT, INSERT, UPDATE ON oscal_assessment_results TO kubric_app';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Verify policies are applied
-- ---------------------------------------------------------------------------
SELECT
    schemaname,
    tablename,
    rowsecurity AS rls_enabled,
    forcerowsecurity AS rls_forced
FROM pg_tables
WHERE schemaname = 'public'
  AND tablename IN (
    'assets', 'user_access_reviews', 'privileged_access_log',
    'kai_alerts', 'kai_patch_jobs', 'kai_incidents',
    'oscal_assessment_results'
  )
ORDER BY tablename;
