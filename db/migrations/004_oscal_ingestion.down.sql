BEGIN;
DROP POLICY IF EXISTS tenant_isolation ON tenant_control_status;
DROP INDEX IF EXISTS idx_oscal_family;
DROP INDEX IF EXISTS idx_oscal_baseline;
DROP INDEX IF EXISTS idx_tenant_control;
DROP TABLE IF EXISTS tenant_control_status CASCADE;
DROP TABLE IF EXISTS oscal_control_enhancements CASCADE;
DROP TABLE IF EXISTS oscal_controls CASCADE;
COMMIT;
