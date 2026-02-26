-- =============================================================================
-- Migration 001 — Layer 0 Foundation Schema ROLLBACK
-- =============================================================================
-- Drops all tables created by 001_layer0_foundation.up.sql.
-- WARNING: Destructive — all data will be lost.
-- =============================================================================

BEGIN;

DROP TABLE IF EXISTS agent_enrollment     CASCADE;
DROP TABLE IF EXISTS feature_flags        CASCADE;
DROP TABLE IF EXISTS kai_triage_results   CASCADE;
DROP TABLE IF EXISTS noc_agents           CASCADE;
DROP TABLE IF EXISTS noc_clusters         CASCADE;
DROP TABLE IF EXISTS kic_assessments      CASCADE;
DROP TABLE IF EXISTS vdr_findings         CASCADE;
DROP TABLE IF EXISTS kubric_tenants       CASCADE;

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "pgcrypto";

COMMIT;
