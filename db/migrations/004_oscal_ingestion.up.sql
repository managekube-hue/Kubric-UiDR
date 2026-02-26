-- K-DL-PG-004 — OSCAL Control Ingestion
-- NIST 800-53 rev5 control storage for GRC compliance module

BEGIN;

CREATE TABLE IF NOT EXISTS oscal_controls (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id      TEXT NOT NULL,                 -- e.g. 'NIST-SP-800-53-rev5'
    control_id      TEXT NOT NULL,                 -- e.g. 'AC-1', 'SC-7'
    control_class   TEXT,                          -- 'SP800-53'
    title           TEXT NOT NULL,
    family          TEXT NOT NULL,                 -- e.g. 'Access Control'
    priority        TEXT,                          -- P1, P2, P3
    baseline_impact TEXT[],                        -- {'LOW', 'MODERATE', 'HIGH'}
    description     TEXT NOT NULL,
    guidance        TEXT,
    parameters      JSONB DEFAULT '[]',
    parts           JSONB DEFAULT '[]',
    links           JSONB DEFAULT '[]',
    properties      JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (catalog_id, control_id)
);

-- Control enhancements (sub-controls like AC-1(1), SC-7(3))
CREATE TABLE IF NOT EXISTS oscal_control_enhancements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_control  UUID NOT NULL REFERENCES oscal_controls(id),
    enhancement_id  TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    baseline_impact TEXT[],
    withdrawn       BOOLEAN DEFAULT false,
    incorporated_into TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-tenant control implementation status
CREATE TABLE IF NOT EXISTS tenant_control_status (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES kubric_tenants(id),
    control_id      UUID NOT NULL REFERENCES oscal_controls(id),
    status          TEXT NOT NULL CHECK (status IN (
        'not_implemented',
        'planned',
        'partially_implemented',
        'implemented',
        'not_applicable'
    )),
    implementation_description TEXT,
    responsible_role    TEXT,
    evidence_refs       TEXT[],
    last_assessed       TIMESTAMPTZ,
    assessor            TEXT,
    risk_accepted       BOOLEAN DEFAULT false,
    risk_notes          TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, control_id)
);

-- RLS: tenant isolation
ALTER TABLE tenant_control_status ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_control_status
    USING (
        current_setting('app.current_tenant_id', true) IS NULL
        OR current_setting('app.current_tenant_id', true) = ''
        OR tenant_id = current_setting('app.current_tenant_id')::uuid
    );

-- Indexes
CREATE INDEX idx_oscal_family ON oscal_controls (family);
CREATE INDEX idx_oscal_baseline ON oscal_controls USING GIN (baseline_impact);
CREATE INDEX idx_tenant_control ON tenant_control_status (tenant_id, status);

COMMIT;
