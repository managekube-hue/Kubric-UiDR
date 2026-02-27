-- K-KAI-GD-002: KAI Action Queue DDL
-- Guardrail approval workflow for KAI persona-initiated remediation actions.
-- Supports auto-expiry of stale pending actions via trigger function.

-- ---------------------------------------------------------------------------
-- Enum-like CHECK types
-- ---------------------------------------------------------------------------
-- action_type: isolate_host | block_ip | run_responder | patch_system
--              | rotate_secret | send_notification
-- status:      pending | approved | rejected | executing | done | failed

-- ---------------------------------------------------------------------------
-- Table: kai_action_queue
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS kai_action_queue (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       TEXT        NOT NULL,
    action_type     TEXT        NOT NULL
                        CHECK (action_type IN (
                            'isolate_host',
                            'block_ip',
                            'run_responder',
                            'patch_system',
                            'rotate_secret',
                            'send_notification'
                        )),
    payload         JSONB       NOT NULL,
    criticality     INT         NOT NULL CHECK (criticality BETWEEN 1 AND 5),
    requires_mfa    BOOLEAN     NOT NULL DEFAULT FALSE,
    approved_by     TEXT,
    approved_at     TIMESTAMPTZ,
    executed_at     TIMESTAMPTZ,
    status          TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN (
                            'pending',
                            'approved',
                            'rejected',
                            'executing',
                            'done',
                            'failed'
                        )),
    error_msg       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    CONSTRAINT kai_action_queue_pkey PRIMARY KEY (id)
);

COMMENT ON TABLE kai_action_queue IS
    'KAI guardrail approval queue — every AI-initiated action must pass through this table before execution.';

COMMENT ON COLUMN kai_action_queue.criticality IS
    '1=Minimal, 2=Low, 3=Medium, 4=High, 5=Critical';

COMMENT ON COLUMN kai_action_queue.requires_mfa IS
    'When true, the approving user must authenticate with MFA before the action is set to approved.';

COMMENT ON COLUMN kai_action_queue.expires_at IS
    'If populated and NOW() > expires_at while status=pending, the row is automatically set to rejected.';

-- ---------------------------------------------------------------------------
-- Indexes
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_kai_aq_tenant_id
    ON kai_action_queue (tenant_id);

CREATE INDEX IF NOT EXISTS idx_kai_aq_status
    ON kai_action_queue (status);

CREATE INDEX IF NOT EXISTS idx_kai_aq_criticality
    ON kai_action_queue (criticality);

CREATE INDEX IF NOT EXISTS idx_kai_aq_tenant_status
    ON kai_action_queue (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_kai_aq_created_at
    ON kai_action_queue (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_kai_aq_expires_at
    ON kai_action_queue (expires_at)
    WHERE expires_at IS NOT NULL AND status = 'pending';

-- ---------------------------------------------------------------------------
-- Function: kai_expire_stale_actions
-- Marks pending actions whose expires_at has passed as 'rejected'.
-- Intended to be called by a periodic scheduler (pg_cron / cron trigger).
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION kai_expire_stale_actions()
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_expired_count INTEGER;
BEGIN
    UPDATE kai_action_queue
    SET    status    = 'rejected',
           error_msg = 'Auto-rejected: action expired at ' || expires_at::TEXT
    WHERE  status    = 'pending'
      AND  expires_at IS NOT NULL
      AND  expires_at < NOW();

    GET DIAGNOSTICS v_expired_count = ROW_COUNT;

    RAISE NOTICE 'kai_expire_stale_actions: % row(s) expired', v_expired_count;
    RETURN v_expired_count;
END;
$$;

COMMENT ON FUNCTION kai_expire_stale_actions() IS
    'Sets all pending kai_action_queue rows whose expires_at < NOW() to status=rejected. '
    'Returns the number of rows updated.';

-- ---------------------------------------------------------------------------
-- Trigger function: automatically expire on INSERT / UPDATE
-- Called on row INSERT and when status or expires_at changes.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION kai_action_expiry_check()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Only apply to rows transitioning to or staying at 'pending'
    IF NEW.status = 'pending'
       AND NEW.expires_at IS NOT NULL
       AND NEW.expires_at < NOW()
    THEN
        NEW.status    := 'rejected';
        NEW.error_msg := 'Auto-rejected at insert/update: expires_at ' || NEW.expires_at::TEXT || ' is in the past.';
    END IF;
    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- Trigger: kai_action_expiry_trigger
-- ---------------------------------------------------------------------------
DROP TRIGGER IF EXISTS kai_action_expiry_trigger ON kai_action_queue;

CREATE TRIGGER kai_action_expiry_trigger
    BEFORE INSERT OR UPDATE OF status, expires_at
    ON kai_action_queue
    FOR EACH ROW
    EXECUTE FUNCTION kai_action_expiry_check();

COMMENT ON TRIGGER kai_action_expiry_trigger ON kai_action_queue IS
    'Prevents stale actions from entering or remaining in pending state if their expiry has passed.';
