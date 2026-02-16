-- Row-Level Security (RLS) Policies

-- Enable RLS on all tables
ALTER TABLE kubric_core.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubric_core.roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubric_core.permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubric_core.user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubric_core.role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE kubric_core.audit_log ENABLE ROW LEVEL SECURITY;

-- Users can only see their own user record
CREATE POLICY users_select_own ON kubric_core.users
FOR SELECT
USING (id = current_user_id());

CREATE POLICY users_update_own ON kubric_core.users
FOR UPDATE
USING (id = current_user_id());

-- Roles are readable to all authenticated users
CREATE POLICY roles_select_all ON kubric_core.roles
FOR SELECT
USING (true);

-- Permissions are readable to all authenticated users
CREATE POLICY permissions_select_all ON kubric_core.permissions
FOR SELECT
USING (true);

-- User-Role mappings viewable by the user or admins
CREATE POLICY user_roles_select ON kubric_core.user_roles
FOR SELECT
USING (user_id = current_user_id() OR has_admin_role());

-- Audit log viewable only by admins
CREATE POLICY audit_log_select ON kubric_core.audit_log
FOR SELECT
USING (has_admin_role());

-- Create security functions
CREATE OR REPLACE FUNCTION current_user_id() RETURNS UUID AS $$
SELECT (current_setting('app.current_user_id', true))::UUID;
$$ LANGUAGE SQL;

CREATE OR REPLACE FUNCTION has_admin_role() RETURNS BOOLEAN AS $$
SELECT EXISTS (
    SELECT 1
    FROM kubric_core.user_roles ur
    JOIN kubric_core.roles r ON ur.role_id = r.id
    WHERE ur.user_id = current_user_id()
    AND r.name = 'admin'
);
$$ LANGUAGE SQL;
