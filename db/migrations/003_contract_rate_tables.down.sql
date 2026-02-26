BEGIN;
DROP POLICY IF EXISTS tenant_isolation ON contract_rate_tables;
DROP INDEX IF EXISTS idx_rate_active;
DROP INDEX IF EXISTS idx_rate_history;
DROP TABLE IF EXISTS contract_rate_tables CASCADE;
COMMIT;
