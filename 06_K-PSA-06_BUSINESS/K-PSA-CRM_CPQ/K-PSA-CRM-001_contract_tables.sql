-- PostgreSQL DDL: CRM sales opportunities and CPQ quote tables

CREATE TABLE IF NOT EXISTS sales_opportunities (
    id               UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id        UUID        REFERENCES tenants(id) ON DELETE SET NULL,
    prospect_name    TEXT        NOT NULL,
    prospect_email   TEXT,
    prospect_company TEXT,
    stage            TEXT        NOT NULL DEFAULT 'lead'
        CHECK (stage IN ('lead','qualified','proposal','negotiation','closed_won','closed_lost')),
    est_monthly_usd  NUMERIC(10,2),
    assigned_to      UUID        REFERENCES users(id) ON DELETE SET NULL,
    close_date       DATE,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION sales_opportunities_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER sales_opportunities_updated_at
BEFORE UPDATE ON sales_opportunities
FOR EACH ROW EXECUTE FUNCTION sales_opportunities_set_updated_at();

-- CPQ quote lines stored as JSONB: [{"description":"","quantity":1,"unit_price":0,"amount":0}]
CREATE TABLE IF NOT EXISTS sales_quotes (
    id               UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    opportunity_id   UUID        REFERENCES sales_opportunities(id) ON DELETE SET NULL,
    quote_number     TEXT        UNIQUE NOT NULL,
    line_items       JSONB       NOT NULL DEFAULT '[]',
    subtotal_usd     NUMERIC(10,2) NOT NULL,
    discount_pct     NUMERIC(5,2) DEFAULT 0,
    tax_pct          NUMERIC(5,2) DEFAULT 0,
    total_usd        NUMERIC(10,2) NOT NULL,
    status           TEXT        NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','sent','accepted','rejected','expired')),
    valid_until      DATE,
    created_by       UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE TRIGGER sales_quotes_updated_at
BEFORE UPDATE ON sales_quotes
FOR EACH ROW EXECUTE FUNCTION sales_opportunities_set_updated_at();

-- Auto-generate quote numbers: QT-YYYYMM-NNNN
CREATE SEQUENCE IF NOT EXISTS quote_number_seq START 1000 INCREMENT 1;

CREATE OR REPLACE FUNCTION next_quote_number()
RETURNS TEXT AS $$
BEGIN
    RETURN 'QT-' || to_char(NOW(), 'YYYYMM') || '-' || lpad(nextval('quote_number_seq')::text, 4, '0');
END;
$$ LANGUAGE plpgsql;

-- Quote acceptance audit trail
CREATE TABLE IF NOT EXISTS quote_acceptance_log (
    id          UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    quote_id    UUID        NOT NULL REFERENCES sales_quotes(id),
    action      TEXT        NOT NULL CHECK (action IN ('sent','viewed','accepted','rejected','expired')),
    actor_email TEXT,
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opportunities_stage    ON sales_opportunities(stage);
CREATE INDEX IF NOT EXISTS idx_opportunities_assigned ON sales_opportunities(assigned_to);
CREATE INDEX IF NOT EXISTS idx_quotes_opportunity     ON sales_quotes(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_quotes_status          ON sales_quotes(status);
CREATE INDEX IF NOT EXISTS idx_quote_log_quote        ON quote_acceptance_log(quote_id);

ALTER TABLE sales_opportunities ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_quotes ENABLE ROW LEVEL SECURITY;

CREATE POLICY opp_tenant_isolation ON sales_opportunities
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid
        OR current_setting('app.role', true) = 'admin');

CREATE POLICY quote_tenant_isolation ON sales_quotes
    USING (opportunity_id IN (
        SELECT id FROM sales_opportunities
        WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
    ) OR current_setting('app.role', true) = 'admin');
