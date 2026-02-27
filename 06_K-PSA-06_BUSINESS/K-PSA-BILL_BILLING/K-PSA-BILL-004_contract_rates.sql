-- PostgreSQL DDL: MSP contract rate tables
-- Requires: btree_gist extension for exclusion constraint
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS msp_contracts (
    id                  UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    contract_type       TEXT        NOT NULL CHECK (contract_type IN ('standard','enterprise','government','nonprofit')),
    hle_list_price_usd  NUMERIC(10,2) NOT NULL DEFAULT 1500.00,
    discount_pct        NUMERIC(5,2) NOT NULL DEFAULT 0.00
        CHECK (discount_pct >= 0 AND discount_pct <= 100),
    min_hle             NUMERIC(6,2) NOT NULL DEFAULT 1.00,
    max_hle             NUMERIC(6,2),
    overage_rate_usd    NUMERIC(10,2),
    payment_terms       TEXT        NOT NULL DEFAULT 'net30'
        CHECK (payment_terms IN ('net30','net60','prepaid','monthly')),
    billing_currency    TEXT        NOT NULL DEFAULT 'USD',
    effective_from      DATE        NOT NULL DEFAULT CURRENT_DATE,
    effective_to        DATE,
    notes               TEXT,
    signed_by           TEXT,
    signed_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT no_overlapping_contracts EXCLUDE USING gist (
        tenant_id WITH =,
        daterange(effective_from, COALESCE(effective_to, 'infinity'::date), '[]') WITH &&
    )
);

-- Automatically update updated_at on row modification
CREATE OR REPLACE FUNCTION msp_contracts_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER msp_contracts_updated_at
BEFORE UPDATE ON msp_contracts
FOR EACH ROW EXECUTE FUNCTION msp_contracts_set_updated_at();

-- Per-contract volume commitments and overage rates
CREATE TABLE IF NOT EXISTS contract_volumes (
    id              UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    contract_id     UUID        NOT NULL REFERENCES msp_contracts(id) ON DELETE CASCADE,
    metric          TEXT        NOT NULL
        CHECK (metric IN ('agent_seats','events_M','ml_calls','storage_gb')),
    included_qty    NUMERIC(12,2) NOT NULL,
    overage_rate_usd NUMERIC(10,4),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Billing usage summary: populated by usage_aggregator monthly run
CREATE TABLE IF NOT EXISTS billing_usage_summary (
    id              UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_key      TEXT        NOT NULL,              -- e.g. "2025-01"
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    agent_seats     NUMERIC(12,4) NOT NULL DEFAULT 0,
    events_ingested NUMERIC(18,4) NOT NULL DEFAULT 0,
    ml_calls        NUMERIC(18,4) NOT NULL DEFAULT 0,
    storage_gb      NUMERIC(12,4) NOT NULL DEFAULT 0,
    total_usd       NUMERIC(10,2) NOT NULL DEFAULT 0,
    invoice_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT billing_usage_summary_uq UNIQUE (tenant_id, period_key)
);

CREATE INDEX IF NOT EXISTS idx_msp_contracts_tenant     ON msp_contracts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_msp_contracts_effective  ON msp_contracts(effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_contract_volumes_contract ON contract_volumes(contract_id);
CREATE INDEX IF NOT EXISTS idx_billing_summary_tenant   ON billing_usage_summary(tenant_id, period_key);

-- Row-level security: tenants see only their own contracts
ALTER TABLE msp_contracts ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_usage_summary ENABLE ROW LEVEL SECURITY;

CREATE POLICY msp_contracts_tenant_isolation ON msp_contracts
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE POLICY billing_summary_tenant_isolation ON billing_usage_summary
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
