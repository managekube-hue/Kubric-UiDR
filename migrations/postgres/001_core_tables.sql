-- =============================================================================
-- Kubric — Supabase/PostgreSQL Migration L0-7
-- All 8 core platform tables with RLS policies, indexes, and constraints.
-- =============================================================================
-- Apply with:
--   psql $DATABASE_URL -f migrations/postgres/001_core_tables.sql
-- Or via Supabase dashboard → SQL Editor → Run
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";         -- text search on alert summaries

-- =============================================================================
-- TABLE 1 — tenants
-- Master account record for every MSP customer.
-- K-SVC is the authoritative writer; all other services read via JOIN.
-- =============================================================================
CREATE TABLE IF NOT EXISTS tenants (
    id                UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug              TEXT        NOT NULL UNIQUE,           -- URL-safe identifier (e.g. "acme-corp")
    display_name      TEXT        NOT NULL,
    plan              TEXT        NOT NULL DEFAULT 'starter' CHECK (plan IN ('starter','growth','enterprise')),
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    status            TEXT        NOT NULL DEFAULT 'active'  CHECK (status IN ('active','suspended','cancelled')),
    kiss_score        FLOAT4      DEFAULT 0.0 CHECK (kiss_score BETWEEN 0 AND 100),
    max_endpoints     INTEGER     NOT NULL DEFAULT 10,
    timezone          TEXT        NOT NULL DEFAULT 'UTC',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants (slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants (status);

-- =============================================================================
-- TABLE 2 — users
-- Portal users linked to a tenant.  Auth is handled by Supabase auth.users;
-- this table stores the Kubric-specific profile and RBAC role.
-- =============================================================================
CREATE TABLE IF NOT EXISTS users (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    supabase_uid    UUID        UNIQUE,                      -- maps to auth.users.id
    email           TEXT        NOT NULL,
    full_name       TEXT,
    role            TEXT        NOT NULL DEFAULT 'viewer'
                    CHECK (role IN ('owner','admin','analyst','viewer','api_service')),
    mfa_enabled     BOOLEAN     NOT NULL DEFAULT false,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_email_tenant UNIQUE (tenant_id, email)
);

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_supabase ON users (supabase_uid);

-- =============================================================================
-- TABLE 3 — agents
-- Registered endpoint and sensor agents managed by NOC service.
-- =============================================================================
CREATE TABLE IF NOT EXISTS agents (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    hostname        TEXT        NOT NULL,
    agent_type      TEXT        NOT NULL CHECK (agent_type IN ('coresec','netguard','perftrace','watchdog')),
    version         TEXT        NOT NULL DEFAULT '0.0.0',
    platform        TEXT                                     -- linux/windows/macos
                    CHECK (platform IN ('linux','windows','macos','container','unknown')),
    ip_address      INET,
    mac_address     MACADDR,
    status          TEXT        NOT NULL DEFAULT 'online'
                    CHECK (status IN ('online','offline','degraded','updating','quarantined')),
    last_heartbeat  TIMESTAMPTZ,
    labels          JSONB       NOT NULL DEFAULT '{}',       -- k:v metadata for grouping
    config          JSONB       NOT NULL DEFAULT '{}',       -- agent-specific config overrides
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_agent_hostname_tenant UNIQUE (tenant_id, hostname)
);

CREATE INDEX IF NOT EXISTS idx_agents_tenant    ON agents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_agents_status    ON agents (status);
CREATE INDEX IF NOT EXISTS idx_agents_type      ON agents (agent_type);
CREATE INDEX IF NOT EXISTS idx_agents_heartbeat ON agents (last_heartbeat);

-- =============================================================================
-- TABLE 4 — assessments
-- Security control assessments managed by KIC service.
-- =============================================================================
CREATE TABLE IF NOT EXISTS assessments (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    framework       TEXT        NOT NULL                     -- CIS, NIST, SOC2, ISO27001, PCI-DSS
                    CHECK (framework IN ('CIS_v8','NIST_CSF','SOC2','ISO27001','PCI_DSS','HIPAA','GDPR','CUSTOM')),
    profile         TEXT,                                    -- e.g. "CIS Level 2"
    status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','in_progress','complete','failed')),
    total_controls  INTEGER     NOT NULL DEFAULT 0,
    passed          INTEGER     NOT NULL DEFAULT 0,
    failed          INTEGER     NOT NULL DEFAULT 0,
    not_applicable  INTEGER     NOT NULL DEFAULT 0,
    pass_rate       FLOAT4      GENERATED ALWAYS AS
                    (CASE WHEN total_controls > 0
                          THEN (passed::FLOAT4 / total_controls) * 100
                          ELSE 0 END) STORED,
    findings_json   JSONB       NOT NULL DEFAULT '[]',       -- array of control findings
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assessments_tenant    ON assessments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_assessments_framework ON assessments (framework);
CREATE INDEX IF NOT EXISTS idx_assessments_status    ON assessments (status);

-- =============================================================================
-- TABLE 5 — findings
-- Vulnerability and risk findings from VDR service.
-- Maps to OCSF class 2002 (Vulnerability Finding).
-- =============================================================================
CREATE TABLE IF NOT EXISTS findings (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    agent_id        UUID        REFERENCES agents (id) ON DELETE SET NULL,
    source          TEXT        NOT NULL CHECK (source IN ('nuclei','trivy','grype','syft','manual','kics','osquery','openscap')),
    cve_id          TEXT,
    title           TEXT        NOT NULL,
    description     TEXT,
    severity        TEXT        NOT NULL DEFAULT 'UNKNOWN'
                    CHECK (severity IN ('CRITICAL','HIGH','MEDIUM','LOW','INFO','UNKNOWN')),
    cvss_score      FLOAT4      CHECK (cvss_score BETWEEN 0 AND 10),
    epss_score      FLOAT4      CHECK (epss_score BETWEEN 0 AND 1),
    asset           TEXT,                                    -- hostname / image / package
    asset_type      TEXT        CHECK (asset_type IN ('host','container','package','iac','cloud_resource')),
    status          TEXT        NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','in_remediation','remediated','accepted_risk','false_positive')),
    remediation_plan_id UUID,
    remediated_at   TIMESTAMPTZ,
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw_payload     JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_findings_tenant   ON findings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings (severity);
CREATE INDEX IF NOT EXISTS idx_findings_status   ON findings (status);
CREATE INDEX IF NOT EXISTS idx_findings_cve      ON findings (cve_id);
CREATE INDEX IF NOT EXISTS idx_findings_asset    ON findings (asset);

-- =============================================================================
-- TABLE 6 — incidents
-- Active security incidents created by KAI-TRIAGE.
-- Maps to OCSF class 4001 (Security Finding).
-- =============================================================================
CREATE TABLE IF NOT EXISTS incidents (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    triage_id       TEXT,                                    -- KAI-TRIAGE result ID
    title           TEXT        NOT NULL,
    summary         TEXT,                                    -- LLM-generated summary
    severity        TEXT        NOT NULL DEFAULT 'MEDIUM'
                    CHECK (severity IN ('CRITICAL','HIGH','MEDIUM','LOW','INFO')),
    status          TEXT        NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open','investigating','contained','resolved','false_positive')),
    assigned_to     UUID        REFERENCES users (id) ON DELETE SET NULL,
    mitre_techniques TEXT[]     NOT NULL DEFAULT '{}',
    confidence      FLOAT4      CHECK (confidence BETWEEN 0 AND 1),
    source_events   JSONB       NOT NULL DEFAULT '[]',       -- array of OCSF event IDs
    timeline        JSONB       NOT NULL DEFAULT '[]',       -- array of timestamped actions
    resolved_at     TIMESTAMPTZ,
    mttr_seconds    INTEGER     GENERATED ALWAYS AS
                    (EXTRACT(EPOCH FROM (resolved_at - created_at))::INTEGER) STORED,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_incidents_tenant   ON incidents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents (severity);
CREATE INDEX IF NOT EXISTS idx_incidents_status   ON incidents (status);
CREATE INDEX IF NOT EXISTS idx_incidents_assigned ON incidents (assigned_to);
-- GIN index for MITRE technique array queries
CREATE INDEX IF NOT EXISTS idx_incidents_mitre ON incidents USING gin (mitre_techniques);

-- =============================================================================
-- TABLE 7 — audit_log
-- Immutable append-only ledger of all platform actions.
-- No deletes or updates allowed (enforced by trigger).
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id              BIGSERIAL   PRIMARY KEY,
    tenant_id       UUID        REFERENCES tenants (id) ON DELETE SET NULL,
    user_id         UUID        REFERENCES users (id) ON DELETE SET NULL,
    service         TEXT        NOT NULL,                    -- ksvc, kic, noc, vdr, kai, portal
    action          TEXT        NOT NULL,                    -- CREATE_TENANT, DELETE_AGENT, etc.
    resource_type   TEXT,
    resource_id     TEXT,
    ip_address      INET,
    user_agent      TEXT,
    request_id      TEXT,
    outcome         TEXT        NOT NULL DEFAULT 'success' CHECK (outcome IN ('success','failure','error')),
    details         JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prevent updates and deletes on audit_log
CREATE OR REPLACE FUNCTION audit_log_immutable()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log rows are immutable';
END;
$$;

DROP TRIGGER IF EXISTS tg_audit_log_immutable ON audit_log;
CREATE TRIGGER tg_audit_log_immutable
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_immutable();

CREATE INDEX IF NOT EXISTS idx_audit_tenant  ON audit_log (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user    ON audit_log (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action  ON audit_log (service, action);

-- =============================================================================
-- TABLE 8 — billing_invoices
-- Metered usage invoices created by KAI-CLERK (BillingWorkflow).
-- =============================================================================
CREATE TABLE IF NOT EXISTS billing_invoices (
    id                  UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id           UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    billing_period      TEXT        NOT NULL,                -- YYYY-MM
    stripe_invoice_id   TEXT        UNIQUE,
    status              TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','created','paid','void','error','skipped')),
    process_events      BIGINT      NOT NULL DEFAULT 0,
    network_events      BIGINT      NOT NULL DEFAULT 0,
    vuln_findings       INTEGER     NOT NULL DEFAULT 0,
    total_usd           NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_items          JSONB       NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_invoice_tenant_period UNIQUE (tenant_id, billing_period)
);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant ON billing_invoices (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoices_period ON billing_invoices (billing_period);

-- =============================================================================
-- Row Level Security (RLS) — tenant isolation
-- All service calls must set app.current_tenant_id via SET LOCAL.
-- =============================================================================

ALTER TABLE tenants           ENABLE ROW LEVEL SECURITY;
ALTER TABLE users             ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents            ENABLE ROW LEVEL SECURITY;
ALTER TABLE assessments       ENABLE ROW LEVEL SECURITY;
ALTER TABLE findings          ENABLE ROW LEVEL SECURITY;
ALTER TABLE incidents         ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log         ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_invoices  ENABLE ROW LEVEL SECURITY;

-- Service role (backend) bypasses RLS — applied by authenticating with service_role key.
-- Portal/JWT customers see only their own tenant rows.

CREATE POLICY tenant_isolation ON users
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

CREATE POLICY tenant_isolation ON agents
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

CREATE POLICY tenant_isolation ON assessments
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

CREATE POLICY tenant_isolation ON findings
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

CREATE POLICY tenant_isolation ON incidents
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

CREATE POLICY tenant_isolation ON audit_log
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

CREATE POLICY tenant_isolation ON billing_invoices
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::UUID);

-- =============================================================================
-- Auto-update updated_at timestamp helper
-- =============================================================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['tenants','users','agents','assessments','findings','incidents','billing_invoices'] LOOP
        EXECUTE format('
            DROP TRIGGER IF EXISTS tg_%I_updated_at ON %I;
            CREATE TRIGGER tg_%I_updated_at
                BEFORE UPDATE ON %I
                FOR EACH ROW EXECUTE FUNCTION set_updated_at();
        ', t, t, t, t);
    END LOOP;
END;
$$;
