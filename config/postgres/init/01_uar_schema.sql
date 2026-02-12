-- User Account Registry (UAR) schema
-- Stores all identity and access control data

CREATE SCHEMA IF NOT EXISTS kubric_core;

-- Users/Identities table
CREATE TABLE kubric_core.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    blake3_fingerprint VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NULL
);

-- Roles table
CREATE TABLE kubric_core.roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Permissions table
CREATE TABLE kubric_core.permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) UNIQUE NOT NULL,
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User-Role mapping (many-to-many)
CREATE TABLE kubric_core.user_roles (
    user_id UUID NOT NULL REFERENCES kubric_core.users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES kubric_core.roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

-- Role-Permission mapping (many-to-many)
CREATE TABLE kubric_core.role_permissions (
    role_id UUID NOT NULL REFERENCES kubric_core.roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES kubric_core.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Audit log
CREATE TABLE kubric_core.audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NULL REFERENCES kubric_core.users(id) ON DELETE SET NULL,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    resource_id VARCHAR(255) NULL,
    changes JSONB NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_users_blake3 ON kubric_core.users(blake3_fingerprint);
CREATE INDEX idx_users_email ON kubric_core.users(email);
CREATE INDEX idx_user_roles_user ON kubric_core.user_roles(user_id);
CREATE INDEX idx_user_roles_role ON kubric_core.user_roles(role_id);
CREATE INDEX idx_role_permissions_role ON kubric_core.role_permissions(role_id);
CREATE INDEX idx_audit_log_user ON kubric_core.audit_log(user_id);
CREATE INDEX idx_audit_log_timestamp ON kubric_core.audit_log(timestamp);
