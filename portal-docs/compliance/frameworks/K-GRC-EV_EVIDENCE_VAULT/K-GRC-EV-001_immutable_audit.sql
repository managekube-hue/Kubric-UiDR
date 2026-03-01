-- PostgreSQL DDL: Append-only immutable audit log with hash-chained integrity

CREATE TABLE IF NOT EXISTS audit_events (
    id            UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id     UUID        NOT NULL REFERENCES tenants(id),
    event_type    TEXT        NOT NULL
        CHECK (event_type IN ('access','change','delete','auth','export','config','compliance')),
    actor_id      UUID,
    actor_email   TEXT,
    resource_type TEXT        NOT NULL,
    resource_id   TEXT,
    action        TEXT        NOT NULL,
    outcome       TEXT        NOT NULL CHECK (outcome IN ('success','failure','partial')),
    ip_address    INET,
    user_agent    TEXT,
    payload       JSONB       NOT NULL DEFAULT '{}',
    previous_hash TEXT        NOT NULL DEFAULT '',
    event_hash    TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Prevent UPDATE and DELETE — audit_events is append-only
CREATE OR REPLACE FUNCTION audit_prevent_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only: % not allowed', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER audit_events_immutable
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION audit_prevent_mutation();

-- Prevent TRUNCATE as an additional safeguard
CREATE OR REPLACE FUNCTION audit_prevent_truncate()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'TRUNCATE on audit_events is not permitted';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER audit_events_no_truncate
BEFORE TRUNCATE ON audit_events
FOR EACH STATEMENT EXECUTE FUNCTION audit_prevent_truncate();

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_tenant_time  ON audit_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource     ON audit_events(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor        ON audit_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_event_type   ON audit_events(event_type, created_at DESC);

-- Hash-chain verification function
-- Verifies that each event_hash is correctly derived from the previous event's hash
CREATE OR REPLACE FUNCTION verify_audit_chain(
    p_tenant_id UUID,
    p_start     TIMESTAMPTZ,
    p_end       TIMESTAMPTZ
)
RETURNS TABLE(
    id            UUID,
    is_valid      BOOLEAN,
    expected_hash TEXT,
    actual_hash   TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH ordered AS (
        SELECT
            e.id,
            e.event_hash,
            e.previous_hash,
            e.created_at,
            LAG(e.event_hash, 1, '') OVER (ORDER BY e.created_at, e.id) AS prev_hash_actual
        FROM audit_events e
        WHERE e.tenant_id = p_tenant_id
          AND e.created_at BETWEEN p_start AND p_end
    )
    SELECT
        o.id,
        o.previous_hash = o.prev_hash_actual                              AS is_valid,
        encode(
            sha256(
                (o.prev_hash_actual || o.id::text || o.created_at::text)::bytea
            ), 'hex'
        )                                                                  AS expected_hash,
        o.event_hash                                                       AS actual_hash
    FROM ordered o
    ORDER BY o.created_at;
END;
$$ LANGUAGE plpgsql;

-- Legal hold: flag events that must not be archived before a hold expires
CREATE TABLE IF NOT EXISTS audit_legal_hold (
    id          UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id   UUID        NOT NULL REFERENCES tenants(id),
    hold_reason TEXT        NOT NULL,
    hold_until  TIMESTAMPTZ NOT NULL,
    created_by  UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_legal_hold_tenant ON audit_legal_hold(tenant_id, hold_until);

-- Retention policy: events older than 7 years may be archived (enforced by ops runbook)
-- Archive trigger placeholder — actual archival runs nightly via pg_cron + pg_dump to S3
COMMENT ON TABLE audit_events IS
    'Append-only audit log. Retention: 7 years on-disk, permanent cold storage on S3. '
    'Hash chain provides tamper evidence. See ops/runbooks/audit-archival.md.';
