-- =============================================================================
-- Migration 002 — Tenant RLS Policies ROLLBACK
-- =============================================================================

BEGIN;

-- Remove policies
DROP POLICY IF EXISTS kubric_tenants_tenant_isolation    ON kubric_tenants;
DROP POLICY IF EXISTS vdr_findings_tenant_isolation       ON vdr_findings;
DROP POLICY IF EXISTS kic_assessments_tenant_isolation    ON kic_assessments;
DROP POLICY IF EXISTS noc_clusters_tenant_isolation       ON noc_clusters;
DROP POLICY IF EXISTS noc_agents_tenant_isolation         ON noc_agents;
DROP POLICY IF EXISTS kai_triage_results_tenant_isolation ON kai_triage_results;
DROP POLICY IF EXISTS feature_flags_tenant_isolation      ON feature_flags;
DROP POLICY IF EXISTS agent_enrollment_tenant_isolation   ON agent_enrollment;

-- Disable RLS (tables remain, data is untouched)
ALTER TABLE kubric_tenants    DISABLE ROW LEVEL SECURITY;
ALTER TABLE vdr_findings       DISABLE ROW LEVEL SECURITY;
ALTER TABLE kic_assessments    DISABLE ROW LEVEL SECURITY;
ALTER TABLE noc_clusters       DISABLE ROW LEVEL SECURITY;
ALTER TABLE noc_agents         DISABLE ROW LEVEL SECURITY;
ALTER TABLE kai_triage_results DISABLE ROW LEVEL SECURITY;
ALTER TABLE feature_flags      DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_enrollment   DISABLE ROW LEVEL SECURITY;

-- Drop helper function
DROP FUNCTION IF EXISTS app_tenant_id();

-- Drop service role (will fail if other objects depend on it — use CASCADE if needed)
DROP ROLE IF EXISTS kubric_superuser;

COMMIT;
