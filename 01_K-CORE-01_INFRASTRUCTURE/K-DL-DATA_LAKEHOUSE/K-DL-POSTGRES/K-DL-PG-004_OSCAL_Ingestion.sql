-- PostgreSQL OSCAL control catalog ingestion tables
-- Module: K-DL-PG-004
-- Purpose: Store NIST/CIS/PCI/SOC2 control catalogs and per-tenant compliance assessment results
-- Depends on: tenants(id), users(id) tables existing

-- ---------------------------------------------------------------------------
-- OSCAL control catalogs (one row per framework version import)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oscal_catalogs (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    catalog_name    TEXT        NOT NULL,
    catalog_version TEXT        NOT NULL,
    framework       TEXT        NOT NULL
                    CHECK (framework IN
                           ('NIST-800-53','CIS-K8s','PCI-DSS','SOC2','ISO-27001','HIPAA',
                            'CIS-AWS','CIS-Azure','CIS-GCP','FEDRAMP','CMMC')),
    uuid            TEXT        NOT NULL UNIQUE,  -- OSCAL document UUID
    last_modified   TIMESTAMPTZ,
    metadata        JSONB       NOT NULL DEFAULT '{}',
    raw_json_path   TEXT,       -- S3 or local path to full OSCAL JSON
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- OSCAL controls (control catalog entries)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oscal_controls (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    catalog_id      UUID        NOT NULL REFERENCES oscal_catalogs(id) ON DELETE CASCADE,
    control_id      TEXT        NOT NULL,   -- e.g. 'AC-1', 'AC-2(1)', '1.1.1'
    family          TEXT,                   -- e.g. 'AC', 'IA', 'CM'
    family_name     TEXT,                   -- e.g. 'Access Control'
    title           TEXT        NOT NULL,
    class           TEXT        NOT NULL DEFAULT 'SP800-53',
    priority        TEXT,                   -- P1 | P2 | P3
    baseline_impact TEXT,                   -- LOW | MODERATE | HIGH
    parameters      JSONB       NOT NULL DEFAULT '[]',
    parts           JSONB       NOT NULL DEFAULT '[]',
    links           JSONB       NOT NULL DEFAULT '[]',
    related_controls TEXT[]     NOT NULL DEFAULT '{}',
    guidance        TEXT,
    enhancement_of  TEXT,       -- parent control_id if this is a control enhancement
    UNIQUE (catalog_id, control_id)
);

CREATE INDEX IF NOT EXISTS idx_oscal_controls_catalog    ON oscal_controls(catalog_id, control_id);
CREATE INDEX IF NOT EXISTS idx_oscal_controls_family     ON oscal_controls(catalog_id, family);
CREATE INDEX IF NOT EXISTS idx_oscal_controls_baseline   ON oscal_controls(catalog_id, baseline_impact)
    WHERE baseline_impact IS NOT NULL;

-- ---------------------------------------------------------------------------
-- OSCAL assessment results (per-tenant control compliance status)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oscal_assessment_results (
    id                      UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id               UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    catalog_id              UUID        NOT NULL REFERENCES oscal_catalogs(id),
    control_id              TEXT        NOT NULL,
    implementation_status   TEXT        NOT NULL
                            CHECK (implementation_status IN (
                                'implemented','partial','planned',
                                'not-applicable','not-implemented'
                            )),
    assessment_tool         TEXT,       -- 'kubric_auto' | 'manual' | 'tenable' | 'crowdstrike'
    evidence                JSONB       NOT NULL DEFAULT '{}',
    remarks                 TEXT,
    risk_acceptance         BOOLEAN     NOT NULL DEFAULT false,
    risk_acceptance_reason  TEXT,
    assessed_by             UUID        REFERENCES users(id),
    assessed_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_review_at          TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, catalog_id, control_id)
);

CREATE INDEX IF NOT EXISTS idx_oscal_results_tenant     ON oscal_assessment_results(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oscal_results_status     ON oscal_assessment_results(tenant_id, implementation_status);
CREATE INDEX IF NOT EXISTS idx_oscal_results_catalog    ON oscal_assessment_results(tenant_id, catalog_id);
CREATE INDEX IF NOT EXISTS idx_oscal_results_review     ON oscal_assessment_results(next_review_at)
    WHERE next_review_at IS NOT NULL AND implementation_status != 'not-applicable';

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION oscal_results_update_timestamp()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_oscal_results_updated_at ON oscal_assessment_results;
CREATE TRIGGER trg_oscal_results_updated_at
    BEFORE UPDATE ON oscal_assessment_results
    FOR EACH ROW EXECUTE FUNCTION oscal_results_update_timestamp();

-- ---------------------------------------------------------------------------
-- OSCAL POA&M (Plan of Action and Milestones) — tracks remediation tasks
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS oscal_poam_items (
    id                      UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id               UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assessment_result_id    UUID        NOT NULL REFERENCES oscal_assessment_results(id) ON DELETE CASCADE,
    catalog_id              UUID        NOT NULL REFERENCES oscal_catalogs(id),
    control_id              TEXT        NOT NULL,
    title                   TEXT        NOT NULL,
    description             TEXT,
    priority                TEXT        NOT NULL DEFAULT 'P3'
                            CHECK (priority IN ('P1','P2','P3','P4')),
    status                  TEXT        NOT NULL DEFAULT 'open'
                            CHECK (status IN ('open','in-progress','resolved','risk-accepted','deferred')),
    due_date                DATE,
    resolved_date           DATE,
    assigned_to             UUID        REFERENCES users(id),
    remediation_steps       TEXT,
    milestones              JSONB       NOT NULL DEFAULT '[]',
    risk_rating             TEXT        CHECK (risk_rating IN ('critical','high','medium','low','informational')),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_poam_tenant  ON oscal_poam_items(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_poam_due     ON oscal_poam_items(due_date)
    WHERE status NOT IN ('resolved','risk-accepted');

-- ---------------------------------------------------------------------------
-- Compliance score summary view (per tenant per framework)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_compliance_scores AS
SELECT
    ar.tenant_id,
    c.framework,
    c.catalog_version,
    COUNT(*)                                                                    AS total_controls,
    COUNT(*) FILTER (WHERE ar.implementation_status = 'implemented')           AS implemented,
    COUNT(*) FILTER (WHERE ar.implementation_status = 'partial')               AS partial,
    COUNT(*) FILTER (WHERE ar.implementation_status = 'planned')               AS planned,
    COUNT(*) FILTER (WHERE ar.implementation_status = 'not-implemented')       AS not_implemented,
    COUNT(*) FILTER (WHERE ar.implementation_status = 'not-applicable')        AS not_applicable,
    ROUND(
        100.0 * COUNT(*) FILTER (WHERE ar.implementation_status = 'implemented')
        / NULLIF(COUNT(*) FILTER (WHERE ar.implementation_status != 'not-applicable'), 0),
        1
    ) AS compliance_pct
FROM oscal_assessment_results ar
JOIN oscal_catalogs c ON ar.catalog_id = c.id
GROUP BY ar.tenant_id, c.framework, c.catalog_version;
