-- Core platform rate tables for data lakehouse billing engine
-- Module: K-DL-PG-003
-- Depends on: tenants(id) table existing

-- ---------------------------------------------------------------------------
-- Platform pricing tiers (HLE = Headless Linux Endpoint unit)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS platform_pricing_tiers (
    id                      UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tier_name               TEXT        NOT NULL UNIQUE,
    min_hle                 NUMERIC(8,2)  NOT NULL DEFAULT 0,
    max_hle                 NUMERIC(8,2),           -- NULL means unlimited
    list_price_per_hle_usd  NUMERIC(10,2) NOT NULL DEFAULT 1500.00,
    discount_pct            NUMERIC(5,2)  NOT NULL DEFAULT 0.00,
    effective_price_per_hle NUMERIC(10,2) GENERATED ALWAYS AS
                                (list_price_per_hle_usd * (1 - discount_pct / 100.0)) STORED,
    features                JSONB       NOT NULL DEFAULT '{}',
    is_active               BOOLEAN     NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO platform_pricing_tiers
    (tier_name, min_hle, max_hle, list_price_per_hle_usd, discount_pct, features)
VALUES
    ('starter',      0,    5,  1500.00,  0.00, '{"agents": 50,   "retention_days": 90,   "ml_calls_mo": 10000,  "support": "email"}'),
    ('growth',       5,   10,  1500.00,  5.00, '{"agents": 100,  "retention_days": 180,  "ml_calls_mo": 50000,  "support": "email"}'),
    ('professional',10,   20,  1500.00, 10.00, '{"agents": 200,  "retention_days": 365,  "ml_calls_mo": 200000, "support": "slack"}'),
    ('enterprise',  20,   50,  1500.00, 15.00, '{"agents": -1,   "retention_days": 730,  "ml_calls_mo": -1,     "support": "dedicated"}'),
    ('unlimited',   50, NULL,  1500.00, 20.00, '{"agents": -1,   "retention_days": 1825, "ml_calls_mo": -1,     "support": "dedicated"}')
ON CONFLICT (tier_name) DO UPDATE
    SET discount_pct           = EXCLUDED.discount_pct,
        features               = EXCLUDED.features,
        updated_at             = NOW();

-- ---------------------------------------------------------------------------
-- Tenant contracts (links a tenant to a pricing tier with override fields)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tenant_contracts (
    id                      UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id               UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tier_id                 UUID        NOT NULL REFERENCES platform_pricing_tiers(id),
    contract_start          DATE        NOT NULL,
    contract_end            DATE,
    committed_hle           NUMERIC(8,2) NOT NULL DEFAULT 0,
    negotiated_price_override NUMERIC(10,2),       -- NULL = use tier effective price
    annual_commit_usd       NUMERIC(12,2),
    auto_renew              BOOLEAN     NOT NULL DEFAULT true,
    status                  TEXT        NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','suspended','cancelled','expired')),
    stripe_subscription_id  TEXT,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, contract_start)
);

CREATE INDEX IF NOT EXISTS idx_contracts_tenant  ON tenant_contracts(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_contracts_active  ON tenant_contracts(status, contract_end)
    WHERE status = 'active';

-- ---------------------------------------------------------------------------
-- Billing usage summary (one row per tenant per billing period)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS billing_usage_summary (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_start        DATE        NOT NULL,
    period_end          DATE        NOT NULL,
    agent_seats         NUMERIC(10,2) NOT NULL DEFAULT 0,
    events_ingested     BIGINT      NOT NULL DEFAULT 0,
    ml_calls            BIGINT      NOT NULL DEFAULT 0,
    storage_gb          NUMERIC(10,2) NOT NULL DEFAULT 0,
    hle_units           NUMERIC(8,4) NOT NULL DEFAULT 0,
    list_price_usd      NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_usd        NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_usd           NUMERIC(12,2) NOT NULL DEFAULT 0,
    stripe_invoice_id   TEXT,
    stripe_charge_id    TEXT,
    invoice_url         TEXT,
    status              TEXT        NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','finalized','paid','void','past_due')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_billing_tenant   ON billing_usage_summary(tenant_id, period_start DESC);
CREATE INDEX IF NOT EXISTS idx_billing_status   ON billing_usage_summary(status, period_end)
    WHERE status IN ('draft','past_due');
CREATE INDEX IF NOT EXISTS idx_billing_stripe   ON billing_usage_summary(stripe_invoice_id)
    WHERE stripe_invoice_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Billing line items (detailed breakdown per invoice)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS billing_line_items (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    billing_summary_id  UUID        NOT NULL
                        REFERENCES billing_usage_summary(id) ON DELETE CASCADE,
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sku                 TEXT        NOT NULL,   -- 'agent_seat' | 'event_ingest_10M' | 'ml_call_1k' | 'storage_gb'
    description         TEXT        NOT NULL,
    quantity            NUMERIC(14,4) NOT NULL DEFAULT 0,
    unit_price_usd      NUMERIC(10,4) NOT NULL DEFAULT 0,
    subtotal_usd        NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_pct        NUMERIC(5,2)  NOT NULL DEFAULT 0,
    total_usd           NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bli_summary ON billing_line_items(billing_summary_id);
CREATE INDEX IF NOT EXISTS idx_bli_tenant  ON billing_line_items(tenant_id);

-- ---------------------------------------------------------------------------
-- Auto-update updated_at trigger
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION billing_update_timestamp()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_pricing_updated_at  ON platform_pricing_tiers;
CREATE TRIGGER trg_pricing_updated_at
    BEFORE UPDATE ON platform_pricing_tiers
    FOR EACH ROW EXECUTE FUNCTION billing_update_timestamp();

DROP TRIGGER IF EXISTS trg_contracts_updated_at ON tenant_contracts;
CREATE TRIGGER trg_contracts_updated_at
    BEFORE UPDATE ON tenant_contracts
    FOR EACH ROW EXECUTE FUNCTION billing_update_timestamp();
