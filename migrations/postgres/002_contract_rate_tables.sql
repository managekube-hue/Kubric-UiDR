-- K-DL-PG-003 — Contract Rate Tables
-- PSA billing rate storage — per-tenant, per-service, per-tier pricing
-- Used by KAI-CLERK for automated invoice line item generation

CREATE TABLE IF NOT EXISTS contract_rate_tables (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    contract_id     UUID NOT NULL,
    effective_from  DATE NOT NULL,
    effective_to    DATE,                          -- NULL = currently active
    rate_type       TEXT NOT NULL CHECK (rate_type IN (
        'per_endpoint',      -- Per monitored endpoint per month
        'per_user',          -- Per identity monitored per month
        'per_gb_ingested',   -- Per GB of telemetry ingested
        'per_scan',          -- Per vulnerability scan
        'per_incident',      -- Per incident response engagement
        'flat_monthly',      -- Flat monthly retainer
        'overage'            -- Overage rate above included units
    )),
    service_module  TEXT NOT NULL CHECK (service_module IN (
        'edr', 'ndr', 'itdr', 'cdr', 'sdr', 'adr', 'ddr',
        'vdr', 'mdr', 'npm', 'uem', 'mdm', 'apm', 'grc',
        'kai', 'backup', 'ti', 'full_stack'
    )),
    tier            TEXT NOT NULL DEFAULT 'professional' CHECK (tier IN (
        'starter', 'professional', 'enterprise', 'custom'
    )),
    unit_price_cents    INTEGER NOT NULL,          -- Price in cents (USD)
    included_units      INTEGER DEFAULT 0,         -- Units included in flat rate
    overage_price_cents INTEGER DEFAULT 0,         -- Per-unit overage price
    currency            TEXT NOT NULL DEFAULT 'USD',
    billing_frequency   TEXT NOT NULL DEFAULT 'monthly' CHECK (billing_frequency IN (
        'monthly', 'quarterly', 'annually'
    )),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RLS policy: tenants can only see their own rates
ALTER TABLE contract_rate_tables ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON contract_rate_tables
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Index for active rate lookup
CREATE INDEX idx_rate_active ON contract_rate_tables (tenant_id, service_module, tier)
    WHERE effective_to IS NULL;

-- Index for historical rate queries
CREATE INDEX idx_rate_history ON contract_rate_tables (tenant_id, effective_from, effective_to);

-- Billing usage aggregation view
CREATE OR REPLACE VIEW billing_usage_summary AS
SELECT
    t.id AS tenant_id,
    t.name AS tenant_name,
    crt.service_module,
    crt.tier,
    crt.unit_price_cents,
    crt.included_units,
    COALESCE(bu.actual_units, 0) AS actual_units,
    GREATEST(0, COALESCE(bu.actual_units, 0) - crt.included_units) AS overage_units,
    crt.unit_price_cents * crt.included_units
        + crt.overage_price_cents * GREATEST(0, COALESCE(bu.actual_units, 0) - crt.included_units)
        AS total_cents
FROM contract_rate_tables crt
JOIN tenants t ON t.id = crt.tenant_id
LEFT JOIN LATERAL (
    SELECT count(*) AS actual_units
    FROM billing_usage bu
    WHERE bu.tenant_id = crt.tenant_id
      AND bu.service_module = crt.service_module
      AND bu.usage_date >= date_trunc('month', CURRENT_DATE)
) bu ON true
WHERE crt.effective_to IS NULL;

-- Seed: default rate card
INSERT INTO contract_rate_tables (tenant_id, contract_id, effective_from, rate_type, service_module, tier, unit_price_cents, included_units, overage_price_cents)
SELECT
    '00000000-0000-0000-0000-000000000000'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    '2024-01-01',
    'per_endpoint',
    module,
    'professional',
    1500,  -- $15.00/endpoint
    100,   -- 100 included
    1200   -- $12.00 overage
FROM unnest(ARRAY['edr', 'ndr', 'vdr', 'full_stack']) AS module
ON CONFLICT DO NOTHING;
