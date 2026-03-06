-- ============================================
-- User Service - Initial Schema
-- Version: 1.0.0
-- ============================================

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS core;

-- Roles table
CREATE TABLE core.roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(255),
    permissions JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Users table
CREATE TABLE core.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url VARCHAR(500),
    phone VARCHAR(20),

    -- Status flags
    enabled BOOLEAN DEFAULT TRUE,
    account_non_expired BOOLEAN DEFAULT TRUE,
    account_non_locked BOOLEAN DEFAULT TRUE,
    credentials_non_expired BOOLEAN DEFAULT TRUE,
    email_verified BOOLEAN DEFAULT FALSE,

    -- MFA
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(255),

    -- Login tracking
    last_login_at TIMESTAMP WITH TIME ZONE,
    last_login_ip VARCHAR(45),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,

    -- Tokens
    refresh_token VARCHAR(500),
    refresh_token_expires_at TIMESTAMP WITH TIME ZONE,

    -- Preferences
    preferences JSONB DEFAULT '{}',

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_by UUID,

    -- Soft delete
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_by UUID
);

-- User-Roles junction table
CREATE TABLE core.user_roles (
    user_id UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES core.roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    granted_by UUID,
    PRIMARY KEY (user_id, role_id)
);

-- Password history (for preventing reuse)
CREATE TABLE core.password_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    changed_by UUID
);

-- Login audit log
CREATE TABLE core.login_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES core.users(id),
    username VARCHAR(50),
    ip_address VARCHAR(45),
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    mfa_used BOOLEAN DEFAULT FALSE,
    session_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Email verification tokens
CREATE TABLE core.email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Password reset tokens
CREATE TABLE core.password_resets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    ip_address VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_username ON core.users(username);
CREATE INDEX idx_users_email ON core.users(email);
CREATE INDEX idx_users_created_at ON core.users(created_at);
CREATE INDEX idx_users_last_login ON core.users(last_login_at);
CREATE INDEX idx_login_audit_user ON core.login_audit(user_id);
CREATE INDEX idx_login_audit_created ON core.login_audit(created_at);

-- Insert default roles
INSERT INTO core.roles (name, description, permissions) VALUES
    ('USER', 'Standard user with basic permissions', '["project:read", "project:write"]'),
    ('ADMIN', 'Administrator with elevated permissions', '["project:read", "project:write", "project:delete", "user:manage"]'),
    ('OWNER', 'Project owner with full permissions', '["project:*", "user:manage", "billing:manage"]');

-- Insert system user (for migrations and audit)
INSERT INTO core.users (
    id, username, email, password, first_name, last_name,
    enabled, email_verified, created_at
) VALUES (
    '00000000-0000-0000-0000-000000000000',
    'system',
    'system@archplatform.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqhmM6JGKpS4G3R1G2JH8YpfB0Bqy', -- 'system' hashed
    'System',
    'Administrator',
    TRUE,
    TRUE,
    CURRENT_TIMESTAMP
);

-- Comments
COMMENT ON TABLE core.users IS 'User accounts for the platform';
COMMENT ON TABLE core.roles IS 'User roles and permissions';
COMMENT ON TABLE user_service.login_audit IS 'Audit log for login attempts';
