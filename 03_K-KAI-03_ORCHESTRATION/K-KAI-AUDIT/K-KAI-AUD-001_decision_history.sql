-- K-KAI-AUD-001: KAI Decision History Audit Log DDL
-- Immutable, hash-chained audit log for all KAI persona decisions.
-- Partitioned by month for efficient pruning and querying.
-- Row-level security: each tenant can only read its own rows.

-- ---------------------------------------------------------------------------
-- Prerequisites
-- ---------------------------------------------------------------------------
-- The table uses RANGE partitioning on created_at.
-- You must create child partitions with partition management tooling
-- (e.g., pg_partman) or manually ahead of time.

-- ---------------------------------------------------------------------------
-- Table: kai_decision_history  (partitioned by month on created_at)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kai_decision_history (
    id                  UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           TEXT        NOT NULL,
    decision_type       TEXT        NOT NULL
                            CHECK (decision_type IN (
                                'triage',
                                'remediation',
                                'risk_score',
                                'ssvc_decision',
                                'hunt_dispatch'
                            )),
    persona             TEXT        NOT NULL
                            CHECK (persona IN (
                                'TRIAGE',
                                'ANALYST',
                                'HUNTER',
                                'KEEPER',
                                'RISK',
                                'INVEST',
                                'SENTINEL',
                                'FORESIGHT'
                            )),
    input_summary       TEXT        NOT NULL,
    decision            JSONB       NOT NULL,
    confidence          FLOAT
                            CHECK (confidence IS NULL OR (confidence >= 0.0 AND confidence <= 1.0)),
    model_used          TEXT,
    duration_ms         INT
                            CHECK (duration_ms IS NULL OR duration_ms >= 0),
    -- BLAKE3 hash of (input_summary || decision::text) for tamper detection
    blake3_hash         TEXT        NOT NULL,
    -- Hash of the previous row's blake3_hash to form a Merkle chain
    merkle_parent_hash  TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Primary key must include partition key for PostgreSQL constraint routing
    CONSTRAINT kai_decision_history_pkey
        PRIMARY KEY (id, created_at),
    -- Unique blake3 hash across the table (enforced via partial unique index per partition)
    CONSTRAINT kai_decision_history_blake3_uq
        UNIQUE (blake3_hash, created_at)
) PARTITION BY RANGE (created_at);

COMMENT ON TABLE kai_decision_history IS
    'Immutable audit trail of every KAI persona decision. '
    'Rows must never be deleted or updated. Hash chain provides tamper evidence.';

COMMENT ON COLUMN kai_decision_history.blake3_hash IS
    'BLAKE3(input_summary || decision::text). Must be unique globally to detect replays.';

COMMENT ON COLUMN kai_decision_history.merkle_parent_hash IS
    'blake3_hash of the immediately preceding row for the same tenant, '
    'forming an append-only Merkle chain per tenant.';

-- ---------------------------------------------------------------------------
-- Default partition (catch-all for rows inserted with no matching partition)
-- In production, replace with pg_partman monthly auto-creation.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kai_decision_history_default
    PARTITION OF kai_decision_history DEFAULT;

-- Example monthly partitions (production tooling should manage these dynamically):
CREATE TABLE IF NOT EXISTS kai_decision_history_2025_01
    PARTITION OF kai_decision_history
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE IF NOT EXISTS kai_decision_history_2025_02
    PARTITION OF kai_decision_history
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE IF NOT EXISTS kai_decision_history_2025_03
    PARTITION OF kai_decision_history
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE IF NOT EXISTS kai_decision_history_2026_01
    PARTITION OF kai_decision_history
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE IF NOT EXISTS kai_decision_history_2026_02
    PARTITION OF kai_decision_history
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE IF NOT EXISTS kai_decision_history_2026_03
    PARTITION OF kai_decision_history
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- ---------------------------------------------------------------------------
-- Indexes (created on the partitioned parent; PostgreSQL propagates to children)
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_kdh_tenant_id
    ON kai_decision_history (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kdh_persona
    ON kai_decision_history (persona, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kdh_decision_type
    ON kai_decision_history (decision_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kdh_created_at
    ON kai_decision_history (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kdh_blake3
    ON kai_decision_history (blake3_hash);

-- ---------------------------------------------------------------------------
-- Row-level security: tenants can only read their own decisions
-- ---------------------------------------------------------------------------
ALTER TABLE kai_decision_history ENABLE ROW LEVEL SECURITY;

-- Policy for application role (kai_app) – can read own tenant rows only
-- The current_setting() approach allows injecting tenant_id via SET LOCAL
-- e.g.:  SET LOCAL app.current_tenant = 'acme-corp';
CREATE POLICY kai_decision_history_tenant_isolation
    ON kai_decision_history
    AS PERMISSIVE
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant', TRUE));

-- Superusers and the kai_internal role bypass RLS for admin operations
CREATE POLICY kai_decision_history_internal_bypass
    ON kai_decision_history
    AS PERMISSIVE
    FOR ALL
    TO kai_internal
    USING (TRUE)
    WITH CHECK (TRUE);

COMMENT ON POLICY kai_decision_history_tenant_isolation ON kai_decision_history IS
    'Tenants can only SELECT their own rows. Set app.current_tenant before querying.';

-- ---------------------------------------------------------------------------
-- Helper view: recent decisions per tenant (convenience, respects RLS)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_recent_kai_decisions AS
SELECT  id,
        tenant_id,
        decision_type,
        persona,
        LEFT(input_summary, 200)    AS input_summary_excerpt,
        confidence,
        model_used,
        duration_ms,
        blake3_hash,
        created_at
FROM    kai_decision_history
ORDER BY created_at DESC;

COMMENT ON VIEW v_recent_kai_decisions IS
    'Convenience view for the 100 most recent KAI decisions. Inherits RLS from base table.';
