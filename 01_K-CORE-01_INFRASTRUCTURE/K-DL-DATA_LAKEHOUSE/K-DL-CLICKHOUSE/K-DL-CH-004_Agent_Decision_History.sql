-- ClickHouse AI agent decision audit trail
-- Module: K-DL-CH-004
-- Purpose: Immutable audit log of every KAI agent decision for explainability, compliance, and model tuning

CREATE TABLE IF NOT EXISTS agent_decision_history
(
    decision_id         UUID,
    tenant_id           UUID,
    correlation_id      UUID,
    agent_persona       LowCardinality(String),   -- triage | analyst | hunter | responder | patch
    input_payload       String,
    output_payload      String,
    decision_type       LowCardinality(String),   -- triage | escalate | close | remediate | defer
    decision_rationale  String,
    confidence_score    Float32,
    model_name          LowCardinality(String),
    model_version       LowCardinality(String),
    execution_ms        Int32,
    token_count_in      Nullable(Int32),
    token_count_out     Nullable(Int32),
    parent_decision_id  Nullable(UUID),
    ssvc_outcome        LowCardinality(String) DEFAULT '',  -- DEFER | SCHEDULED | OUT_OF_CYCLE | IMMEDIATE
    epss_score          Nullable(Float32),
    cvss_score          Nullable(Float32),
    created_at          DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
PARTITION BY (toYYYYMM(created_at), tenant_id)
ORDER BY (tenant_id, agent_persona, created_at)
TTL created_at + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;

ALTER TABLE agent_decision_history ADD INDEX IF NOT EXISTS idx_correlation (correlation_id)    TYPE bloom_filter GRANULARITY 4;
ALTER TABLE agent_decision_history ADD INDEX IF NOT EXISTS idx_decision_type (decision_type)  TYPE set(8)       GRANULARITY 1;
ALTER TABLE agent_decision_history ADD INDEX IF NOT EXISTS idx_ssvc (ssvc_outcome)            TYPE set(8)       GRANULARITY 1;

-- Materialized view: daily agent decision metrics per tenant
CREATE MATERIALIZED VIEW IF NOT EXISTS agent_decision_daily_mv
ENGINE = SummingMergeTree()
ORDER BY (tenant_id, agent_persona, decision_type, date)
AS SELECT
    tenant_id,
    agent_persona,
    decision_type,
    toDate(created_at)          AS date,
    count()                     AS decision_count,
    avg(confidence_score)       AS avg_confidence,
    avg(execution_ms)           AS avg_execution_ms,
    sum(token_count_in)         AS total_tokens_in,
    sum(token_count_out)        AS total_tokens_out
FROM agent_decision_history
GROUP BY tenant_id, agent_persona, decision_type, toDate(created_at);

-- Materialized view: SSVC outcome distribution per tenant per day
CREATE MATERIALIZED VIEW IF NOT EXISTS agent_ssvc_daily_mv
ENGINE = SummingMergeTree()
ORDER BY (tenant_id, ssvc_outcome, date)
AS SELECT
    tenant_id,
    ssvc_outcome,
    toDate(created_at) AS date,
    count()            AS outcome_count,
    avg(cvss_score)    AS avg_cvss,
    avg(epss_score)    AS avg_epss
FROM agent_decision_history
WHERE ssvc_outcome != ''
GROUP BY tenant_id, ssvc_outcome, toDate(created_at);

-- Materialized view: model performance per model version
CREATE MATERIALIZED VIEW IF NOT EXISTS agent_model_perf_mv
ENGINE = SummingMergeTree()
ORDER BY (model_name, model_version, date)
AS SELECT
    model_name,
    model_version,
    toDate(created_at)       AS date,
    count()                  AS call_count,
    avg(confidence_score)    AS avg_confidence,
    avg(execution_ms)        AS avg_latency_ms,
    sum(token_count_in)      AS total_input_tokens,
    sum(token_count_out)     AS total_output_tokens
FROM agent_decision_history
GROUP BY model_name, model_version, toDate(created_at);
