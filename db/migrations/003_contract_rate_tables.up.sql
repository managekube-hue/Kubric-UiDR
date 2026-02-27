-- K-DL-PG-003 — Contract Rate Tables
-- PSA billing rate storage — per-tenant, per-service, per-tier pricing
-- Used by KAI-CLERK for automated invoice line item generation

BEGIN;

CREATE TABLE IF NOT EXISTS contract_rate_tables (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL REFERENCES kubric_tenants(tenant_id),
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
    USING (
        current_setting('app.current_tenant_id', true) IS NULL
        OR current_setting('app.current_tenant_id', true) = ''
        OR tenant_id = current_setting('app.current_tenant_id', true)
    );

-- Index for active rate lookup
CREATE INDEX idx_rate_active ON contract_rate_tables (tenant_id, service_module, tier)
    WHERE effective_to IS NULL;

-- Index for historical rate queries
CREATE INDEX idx_rate_history ON contract_rate_tables (tenant_id, effective_from, effective_to);

COMMIT;
