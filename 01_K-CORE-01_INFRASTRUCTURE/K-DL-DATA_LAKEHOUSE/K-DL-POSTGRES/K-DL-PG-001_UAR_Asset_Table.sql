-- PostgreSQL UAR (User Access Review) and asset inventory tables
-- Module: K-DL-PG-001
-- Depends on: tenants(id), users(id) tables existing

CREATE TABLE IF NOT EXISTS assets (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    hostname            TEXT        NOT NULL,
    fqdn                TEXT,
    ip_address          INET,
    mac_address         TEXT,
    os_family           TEXT        CHECK (os_family IN
                                    ('windows','linux','macos','network_device',
                                     'cloud_vm','container','iot')),
    os_version          TEXT,
    asset_type          TEXT        CHECK (asset_type IN
                                    ('server','workstation','network','cloud_vm',
                                     'iot','mobile','container')),
    criticality_tier    INTEGER     CHECK (criticality_tier BETWEEN 1 AND 5),
    managed_by          TEXT        NOT NULL DEFAULT 'kubric',
    cloud_provider      TEXT        CHECK (cloud_provider IN ('aws','azure','gcp','on_prem')),
    cloud_instance_id   TEXT,
    cloud_account_id    TEXT,
    cloud_region        TEXT,
    serial_number       TEXT,
    tags                JSONB       NOT NULL DEFAULT '{}',
    agent_id            UUID,
    last_seen_at        TIMESTAMPTZ,
    is_active           BOOLEAN     NOT NULL DEFAULT true,
    deprovisioned_at    TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Asset indexes
CREATE INDEX IF NOT EXISTS idx_assets_tenant         ON assets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_assets_hostname       ON assets(tenant_id, hostname);
CREATE INDEX IF NOT EXISTS idx_assets_ip             ON assets(tenant_id, ip_address);
CREATE INDEX IF NOT EXISTS idx_assets_active         ON assets(tenant_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_assets_agent          ON assets(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_assets_criticality    ON assets(tenant_id, criticality_tier);
CREATE INDEX IF NOT EXISTS idx_assets_tags           ON assets USING gin(tags);
CREATE INDEX IF NOT EXISTS idx_assets_last_seen      ON assets(tenant_id, last_seen_at DESC);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION assets_update_timestamp()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_assets_updated_at ON assets;
CREATE TRIGGER trg_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION assets_update_timestamp();

-- User Access Review table
CREATE TABLE IF NOT EXISTS user_access_reviews (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL,
    user_email      TEXT        NOT NULL,
    resource_type   TEXT        NOT NULL,   -- 'asset' | 'dashboard' | 'api_key' | 'alert' | 'report'
    resource_id     TEXT        NOT NULL,
    access_level    TEXT        NOT NULL,   -- 'read' | 'write' | 'admin' | 'execute'
    granted_at      TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ,
    review_status   TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (review_status IN ('pending','approved','revoked','flagged')),
    reviewed_by     UUID        REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    expiry_at       TIMESTAMPTZ,
    risk_score      NUMERIC(4,2) CHECK (risk_score BETWEEN 0 AND 10),
    risk_flags      JSONB       NOT NULL DEFAULT '[]',  -- array of risk flag strings
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- UAR indexes
CREATE INDEX IF NOT EXISTS idx_uar_tenant_status ON user_access_reviews(tenant_id, review_status);
CREATE INDEX IF NOT EXISTS idx_uar_user          ON user_access_reviews(user_id);
CREATE INDEX IF NOT EXISTS idx_uar_user_email    ON user_access_reviews(tenant_id, user_email);
CREATE INDEX IF NOT EXISTS idx_uar_resource      ON user_access_reviews(tenant_id, resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_uar_expiry        ON user_access_reviews(expiry_at)
    WHERE review_status = 'approved' AND expiry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_uar_risk          ON user_access_reviews(tenant_id, risk_score DESC)
    WHERE risk_score >= 7;

-- Privileged access audit log (append-only)
CREATE TABLE IF NOT EXISTS privileged_access_log (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL,
    user_email      TEXT        NOT NULL,
    asset_id        UUID        REFERENCES assets(id),
    resource_type   TEXT        NOT NULL,
    resource_id     TEXT        NOT NULL,
    action          TEXT        NOT NULL,  -- 'login' | 'command' | 'file_access' | 'data_export'
    outcome         TEXT        NOT NULL CHECK (outcome IN ('success','failure','blocked')),
    session_id      TEXT,
    src_ip          INET,
    details         JSONB       NOT NULL DEFAULT '{}',
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pal_tenant ON privileged_access_log(tenant_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_pal_user   ON privileged_access_log(user_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_pal_asset  ON privileged_access_log(asset_id, recorded_at DESC)
    WHERE asset_id IS NOT NULL;
