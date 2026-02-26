-- K-DL-PG-004 — OSCAL Control Ingestion
-- NIST 800-53 rev5 control storage for GRC compliance module
-- Loaded from vendor/oscal/nist.gov/SP800-53/rev5/*.json

CREATE TABLE IF NOT EXISTS oscal_controls (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_id      TEXT NOT NULL,                 -- e.g. 'NIST-SP-800-53-rev5'
    control_id      TEXT NOT NULL,                 -- e.g. 'AC-1', 'SC-7'
    control_class   TEXT,                          -- 'SP800-53'
    title           TEXT NOT NULL,
    family          TEXT NOT NULL,                 -- e.g. 'Access Control', 'System and Communications Protection'
    priority        TEXT,                          -- P1, P2, P3
    baseline_impact TEXT[],                        -- {'LOW', 'MODERATE', 'HIGH'}
    description     TEXT NOT NULL,
    guidance        TEXT,
    parameters      JSONB DEFAULT '[]',            -- Parameter definitions
    parts           JSONB DEFAULT '[]',            -- Assessment objectives, implementation guidance
    links           JSONB DEFAULT '[]',            -- Related controls, references
    properties      JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (catalog_id, control_id)
);

-- Control enhancements (sub-controls like AC-1(1), SC-7(3))
CREATE TABLE IF NOT EXISTS oscal_control_enhancements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_control  UUID NOT NULL REFERENCES oscal_controls(id),
    enhancement_id  TEXT NOT NULL,                 -- e.g. 'AC-2(1)'
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    baseline_impact TEXT[],
    withdrawn       BOOLEAN DEFAULT false,
    incorporated_into TEXT,                        -- If withdrawn, control it merged into
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-tenant control implementation status
CREATE TABLE IF NOT EXISTS tenant_control_status (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
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
    evidence_refs       TEXT[],                    -- Links to evidence artifacts
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
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Indexes
CREATE INDEX idx_oscal_family ON oscal_controls (family);
CREATE INDEX idx_oscal_baseline ON oscal_controls USING GIN (baseline_impact);
CREATE INDEX idx_tenant_control ON tenant_control_status (tenant_id, status);

-- Compliance posture summary view
CREATE OR REPLACE VIEW compliance_posture AS
SELECT
    tcs.tenant_id,
    oc.family,
    count(*) AS total_controls,
    count(*) FILTER (WHERE tcs.status = 'implemented') AS implemented,
    count(*) FILTER (WHERE tcs.status = 'partially_implemented') AS partial,
    count(*) FILTER (WHERE tcs.status = 'not_implemented') AS not_implemented,
    count(*) FILTER (WHERE tcs.status = 'not_applicable') AS not_applicable,
    round(
        100.0 * count(*) FILTER (WHERE tcs.status IN ('implemented', 'not_applicable'))
        / NULLIF(count(*), 0), 1
    ) AS compliance_pct
FROM tenant_control_status tcs
JOIN oscal_controls oc ON oc.id = tcs.control_id
GROUP BY tcs.tenant_id, oc.family;
