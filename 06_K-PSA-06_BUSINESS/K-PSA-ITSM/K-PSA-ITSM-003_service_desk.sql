-- PostgreSQL DDL: ITSM service desk ticket tables

CREATE TABLE IF NOT EXISTS service_tickets (
    id                  UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ticket_number       TEXT        NOT NULL UNIQUE,
    title               TEXT        NOT NULL,
    description         TEXT,
    priority            TEXT        NOT NULL DEFAULT 'P3'
        CHECK (priority IN ('P1','P2','P3','P4')),
    state               TEXT        NOT NULL DEFAULT 'open'
        CHECK (state IN ('open','in_progress','pending','resolved','closed','cancelled')),
    category            TEXT
        CHECK (category IN ('security_incident','vulnerability','compliance',
                            'change_request','service_request','inquiry')),
    assigned_to         UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_by          UUID        REFERENCES users(id) ON DELETE SET NULL,
    external_ticket_id  TEXT,
    external_system     TEXT
        CHECK (external_system IN ('zammad','connectwise','autotask','jira','servicenow')),
    sla_breach_at       TIMESTAMPTZ,
    resolved_at         TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,
    related_alert_ids   UUID[],
    related_incident_id UUID,
    attachments         JSONB       NOT NULL DEFAULT '[]',
    custom_fields       JSONB       NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-generate ticket numbers: TK-YYYYMM-NNNN
CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START 1000 INCREMENT 1;

CREATE OR REPLACE FUNCTION next_ticket_number()
RETURNS TEXT AS $$
BEGIN
    RETURN 'TK-' || to_char(NOW(), 'YYYYMM') || '-' || lpad(nextval('ticket_number_seq')::text, 4, '0');
END;
$$ LANGUAGE plpgsql;

-- Auto-set sla_breach_at based on priority when inserting a new ticket
CREATE OR REPLACE FUNCTION service_tickets_set_sla()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.sla_breach_at IS NULL THEN
        NEW.sla_breach_at := CASE NEW.priority
            WHEN 'P1' THEN NEW.created_at + INTERVAL '4 hours'
            WHEN 'P2' THEN NEW.created_at + INTERVAL '8 hours'
            WHEN 'P3' THEN NEW.created_at + INTERVAL '24 hours'
            WHEN 'P4' THEN NEW.created_at + INTERVAL '72 hours'
            ELSE NEW.created_at + INTERVAL '24 hours'
        END;
    END IF;
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER service_tickets_sla_and_ts
BEFORE INSERT OR UPDATE ON service_tickets
FOR EACH ROW EXECUTE FUNCTION service_tickets_set_sla();

-- Event log for every state change, comment, or escalation
CREATE TABLE IF NOT EXISTS ticket_events (
    id          UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    ticket_id   UUID        NOT NULL REFERENCES service_tickets(id) ON DELETE CASCADE,
    event_type  TEXT        NOT NULL
        CHECK (event_type IN ('state_change','comment','assignment','sla_breach','escalation')),
    from_state  TEXT,
    to_state    TEXT,
    actor_id    UUID        REFERENCES users(id) ON DELETE SET NULL,
    comment     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Zammad integration: track sync status
CREATE TABLE IF NOT EXISTS zammad_ticket_sync (
    ticket_id        UUID        NOT NULL REFERENCES service_tickets(id) ON DELETE CASCADE,
    zammad_ticket_id TEXT        NOT NULL,
    synced_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sync_status      TEXT        NOT NULL DEFAULT 'synced'
        CHECK (sync_status IN ('synced','pending','error')),
    PRIMARY KEY (ticket_id)
);

CREATE INDEX IF NOT EXISTS idx_tickets_tenant_state    ON service_tickets(tenant_id, state);
CREATE INDEX IF NOT EXISTS idx_tickets_sla             ON service_tickets(sla_breach_at)
    WHERE state NOT IN ('resolved','closed','cancelled');
CREATE INDEX IF NOT EXISTS idx_tickets_priority        ON service_tickets(priority, state);
CREATE INDEX IF NOT EXISTS idx_ticket_events_ticket    ON ticket_events(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_events_type      ON ticket_events(event_type, created_at DESC);

ALTER TABLE service_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY tickets_tenant_isolation ON service_tickets
    USING (tenant_id = current_setting('app.tenant_id', true)::uuid
        OR current_setting('app.role', true) = 'admin');

CREATE POLICY ticket_events_tenant_isolation ON ticket_events
    USING (ticket_id IN (
        SELECT id FROM service_tickets
        WHERE tenant_id = current_setting('app.tenant_id', true)::uuid
    ) OR current_setting('app.role', true) = 'admin');
