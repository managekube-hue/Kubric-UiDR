-- DuckDB ML feature computation for KAI model training and churn/LTV prediction
-- Module: K-DL-DUCK-002
-- Run: duckdb /data/kubric.duckdb < K-DL-DUCK-002_ml_feature_compute.sql

-- Install and load extensions for S3/parquet access
INSTALL httpfs;
LOAD httpfs;

-- Configure S3 credentials from environment
SET s3_region    = (SELECT COALESCE(current_setting('s3.region',    true), 'us-east-1'));
SET s3_access_key_id     = (SELECT COALESCE(current_setting('s3.access_key',    true), ''));
SET s3_secret_access_key = (SELECT COALESCE(current_setting('s3.secret_key',    true), ''));

-- ---------------------------------------------------------------------------
-- Feature table: per-tenant rolling metrics for churn/LTV prediction
-- ---------------------------------------------------------------------------
CREATE OR REPLACE TABLE ml_features AS
WITH tenant_alerts AS (
    SELECT
        tenant_id,
        COUNT(*)  FILTER (WHERE created_at > NOW() - INTERVAL '7 days')                                          AS alerts_7d,
        COUNT(*)  FILTER (WHERE created_at > NOW() - INTERVAL '30 days')                                         AS alerts_30d,
        COUNT(*)  FILTER (WHERE severity = 'critical' AND created_at > NOW() - INTERVAL '7 days')                AS crit_alerts_7d,
        COUNT(*)  FILTER (WHERE severity = 'high'     AND created_at > NOW() - INTERVAL '7 days')                AS high_alerts_7d,
        AVG(
            EXTRACT(EPOCH FROM (closed_at - created_at)) / 3600.0
        )         FILTER (WHERE closed_at IS NOT NULL)                                                           AS avg_mttr_hours,
        COUNT(*)  FILTER (WHERE closed_at IS NOT NULL AND created_at > NOW() - INTERVAL '30 days')               AS closed_30d,
        COUNT(*)  FILTER (WHERE review_status = 'auto_closed'
                          AND   created_at > NOW() - INTERVAL '30 days')                                         AS auto_closed_30d
    FROM read_parquet('s3://kubric-lake/alerts/*.parquet')
    GROUP BY tenant_id
),
tenant_billing AS (
    SELECT
        tenant_id,
        AVG(total_usd)    AS avg_monthly_revenue,
        SUM(total_usd)    AS lifetime_revenue,
        COUNT(*)          AS billing_months,
        MAX(period_end)   AS last_billing_date,
        MIN(period_start) AS first_billing_date
    FROM read_parquet('s3://kubric-lake/billing/*.parquet')
    GROUP BY tenant_id
),
tenant_agents AS (
    SELECT
        tenant_id,
        COUNT(DISTINCT id)                                                           AS total_agents,
        COUNT(DISTINCT id) FILTER (WHERE last_seen_at > NOW() - INTERVAL '24 hours') AS active_agents_24h,
        COUNT(DISTINCT id) FILTER (WHERE last_seen_at > NOW() - INTERVAL '7 days')  AS active_agents_7d,
        AVG(events_per_day)                                                          AS avg_events_per_agent_day
    FROM read_parquet('s3://kubric-lake/agents/*.parquet')
    GROUP BY tenant_id
),
tenant_logins AS (
    SELECT
        tenant_id,
        COUNT(DISTINCT user_id)                                                       AS dau_7d,
        COUNT(*)          FILTER (WHERE logged_at > NOW() - INTERVAL '7 days')        AS logins_7d,
        COUNT(*)          FILTER (WHERE logged_at > NOW() - INTERVAL '30 days')       AS logins_30d,
        MAX(logged_at)                                                                AS last_login_at
    FROM read_parquet('s3://kubric-lake/user_logins/*.parquet')
    GROUP BY tenant_id
)
SELECT
    a.tenant_id,
    -- Alert features
    COALESCE(a.alerts_7d,        0)     AS alerts_7d,
    COALESCE(a.alerts_30d,       0)     AS alerts_30d,
    COALESCE(a.crit_alerts_7d,   0)     AS crit_alerts_7d,
    COALESCE(a.high_alerts_7d,   0)     AS high_alerts_7d,
    COALESCE(a.avg_mttr_hours,   24.0)  AS avg_mttr_hours,
    COALESCE(a.closed_30d,       0)     AS closed_30d,
    COALESCE(a.auto_closed_30d,  0)     AS auto_closed_30d,
    -- Billing features
    COALESCE(b.avg_monthly_revenue, 0)  AS avg_monthly_revenue,
    COALESCE(b.lifetime_revenue,    0)  AS lifetime_revenue,
    COALESCE(b.billing_months,      0)  AS billing_months,
    -- Agent features
    COALESCE(ag.total_agents,         0) AS total_agents,
    COALESCE(ag.active_agents_24h,    0) AS active_agents_24h,
    COALESCE(ag.active_agents_7d,     0) AS active_agents_7d,
    COALESCE(ag.avg_events_per_agent_day, 0) AS avg_events_per_agent_day,
    -- Engagement features
    COALESCE(l.dau_7d,     0)  AS dau_7d,
    COALESCE(l.logins_7d,  0)  AS logins_7d,
    COALESCE(l.logins_30d, 0)  AS logins_30d,
    -- Derived ratio features (avoid division by zero)
    CASE WHEN COALESCE(a.alerts_30d, 0) > 0
         THEN COALESCE(a.closed_30d, 0)::DOUBLE / a.alerts_30d
         ELSE 0.0
    END AS closure_rate_30d,
    CASE WHEN COALESCE(ag.total_agents, 0) > 0
         THEN COALESCE(ag.active_agents_7d, 0)::DOUBLE / ag.total_agents
         ELSE 0.0
    END AS agent_activity_ratio,
    NOW() AS computed_at
FROM tenant_alerts a
LEFT JOIN tenant_billing  b  ON a.tenant_id = b.tenant_id
LEFT JOIN tenant_agents   ag ON a.tenant_id = ag.tenant_id
LEFT JOIN tenant_logins   l  ON a.tenant_id = l.tenant_id;

-- ---------------------------------------------------------------------------
-- Feature table: per-asset vulnerability exposure for risk scoring
-- ---------------------------------------------------------------------------
CREATE OR REPLACE TABLE ml_asset_risk_features AS
WITH asset_vulns AS (
    SELECT
        asset_id,
        tenant_id,
        COUNT(*)                                    AS total_vulns,
        COUNT(*) FILTER (WHERE cvss_v3 >= 9.0)      AS critical_vulns,
        COUNT(*) FILTER (WHERE cvss_v3 >= 7.0)      AS high_vulns,
        COUNT(*) FILTER (WHERE epss_score > 0.5)    AS high_epss_vulns,
        MAX(cvss_v3)                                AS max_cvss,
        AVG(cvss_v3)                                AS avg_cvss,
        MAX(epss_score)                             AS max_epss,
        COUNT(*) FILTER (WHERE exploit_available = true) AS exploitable_vulns,
        MIN(first_seen_at)                          AS oldest_vuln_date
    FROM read_parquet('s3://kubric-lake/vulnerabilities/*.parquet')
    GROUP BY asset_id, tenant_id
),
asset_events AS (
    SELECT
        asset_id,
        COUNT(*) FILTER (WHERE event_time > NOW() - INTERVAL '7 days')  AS events_7d,
        COUNT(*) FILTER (WHERE class_uid IN (1007, 4001))               AS process_net_events_total
    FROM read_parquet('s3://kubric-lake/ocsf_events/*.parquet')
    GROUP BY asset_id
)
SELECT
    v.asset_id,
    v.tenant_id,
    COALESCE(v.total_vulns,       0)     AS total_vulns,
    COALESCE(v.critical_vulns,    0)     AS critical_vulns,
    COALESCE(v.high_vulns,        0)     AS high_vulns,
    COALESCE(v.high_epss_vulns,   0)     AS high_epss_vulns,
    COALESCE(v.max_cvss,          0.0)   AS max_cvss,
    COALESCE(v.avg_cvss,          0.0)   AS avg_cvss,
    COALESCE(v.max_epss,          0.0)   AS max_epss,
    COALESCE(v.exploitable_vulns, 0)     AS exploitable_vulns,
    COALESCE(e.events_7d,         0)     AS events_7d,
    COALESCE(e.process_net_events_total, 0) AS process_net_events_total,
    -- Composite risk score (0-100)
    LEAST(100.0,
        (COALESCE(v.critical_vulns, 0) * 20.0) +
        (COALESCE(v.high_epss_vulns, 0) * 10.0) +
        (COALESCE(v.max_cvss, 0.0) * 3.0)
    ) AS composite_risk_score,
    NOW() AS computed_at
FROM asset_vulns v
LEFT JOIN asset_events e ON v.asset_id = e.asset_id;

-- Export feature tables to parquet for Python model training
COPY ml_features             TO '/tmp/ml_features.parquet'             (FORMAT PARQUET);
COPY ml_asset_risk_features  TO '/tmp/ml_asset_risk_features.parquet'  (FORMAT PARQUET);

-- Summary stats for data quality check
SELECT
    COUNT(*)                     AS tenant_count,
    AVG(alerts_7d)               AS avg_alerts_7d,
    AVG(avg_monthly_revenue)     AS avg_revenue,
    SUM(CASE WHEN billing_months > 0 THEN 1 ELSE 0 END) AS tenants_with_billing
FROM ml_features;
