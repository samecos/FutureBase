-- ============================================
-- Architecture Design Platform - Database Initialization
-- ============================================

-- Create schemas
CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS geometry;
CREATE SCHEMA IF NOT EXISTS versioning;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS analytics;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- Core Tables - Tenants and Users
-- ============================================

-- Tenants table
CREATE TABLE IF NOT EXISTS core.tenants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL,
    slug                VARCHAR(100) UNIQUE NOT NULL,
    description         TEXT,
    logo_url            VARCHAR(500),
    plan_type           VARCHAR(50) NOT NULL DEFAULT 'free' 
                        CHECK (plan_type IN ('free', 'basic', 'professional', 'enterprise')),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'suspended', 'deleted')),
    max_projects        INTEGER NOT NULL DEFAULT 5,
    max_storage_gb      INTEGER NOT NULL DEFAULT 10,
    max_users           INTEGER NOT NULL DEFAULT 10,
    storage_used_bytes  BIGINT NOT NULL DEFAULT 0,
    settings            JSONB DEFAULT '{}',
    billing_info        JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    created_by          UUID,
    updated_by          UUID
);

-- Users table
CREATE TABLE IF NOT EXISTS core.users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    email               VARCHAR(255) NOT NULL,
    username            VARCHAR(100) NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    first_name          VARCHAR(100),
    last_name           VARCHAR(100),
    avatar_url          VARCHAR(500),
    phone               VARCHAR(50),
    role                VARCHAR(50) NOT NULL DEFAULT 'member'
                        CHECK (role IN ('super_admin', 'admin', 'manager', 'designer', 'viewer', 'member')),
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'inactive', 'suspended', 'pending')),
    email_verified      BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at       TIMESTAMPTZ,
    login_count         INTEGER NOT NULL DEFAULT 0,
    preferences         JSONB DEFAULT '{}',
    mfa_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_secret          VARCHAR(255),
    password_changed_at TIMESTAMPTZ,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    
    UNIQUE(tenant_id, email),
    UNIQUE(tenant_id, username)
);

-- Teams table
CREATE TABLE IF NOT EXISTS core.teams (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    color               VARCHAR(7) DEFAULT '#1890FF',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    
    UNIQUE(tenant_id, name)
);

-- Team members table
CREATE TABLE IF NOT EXISTS core.team_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id             UUID NOT NULL REFERENCES core.teams(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role                VARCHAR(50) NOT NULL DEFAULT 'member'
                        CHECK (role IN ('leader', 'member')),
    joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(team_id, user_id)
);

-- ============================================
-- Core Tables - Projects and Designs
-- ============================================

-- Projects table
CREATE TABLE IF NOT EXISTS core.projects (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    project_code        VARCHAR(100),
    status              VARCHAR(50) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'in_progress', 'under_review', 'approved', 'archived', 'deleted')),
    project_type        VARCHAR(100) NOT NULL DEFAULT 'building'
                        CHECK (project_type IN ('building', 'interior', 'landscape', 'urban', 'industrial', 'other')),
    visibility          VARCHAR(20) NOT NULL DEFAULT 'private'
                        CHECK (visibility IN ('private', 'team', 'organization', 'public')),
    thumbnail_url       VARCHAR(500),
    tags                TEXT[] DEFAULT '{}',
    location            JSONB,
    area_total_sqm      DECIMAL(15, 2),
    budget_currency     VARCHAR(3) DEFAULT 'CNY',
    budget_amount       DECIMAL(18, 2),
    start_date          DATE,
    target_end_date     DATE,
    actual_end_date     DATE,
    progress_percent    INTEGER DEFAULT 0 CHECK (progress_percent BETWEEN 0 AND 100),
    settings            JSONB DEFAULT '{}',
    custom_fields       JSONB DEFAULT '{}',
    metadata            JSONB DEFAULT '{}',
    version_count       INTEGER NOT NULL DEFAULT 0,
    current_version_id  UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES core.users(id),
    updated_by          UUID REFERENCES core.users(id),
    
    UNIQUE(tenant_id, project_code)
);

-- Project members table
CREATE TABLE IF NOT EXISTS core.project_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role                VARCHAR(50) NOT NULL DEFAULT 'viewer'
                        CHECK (role IN ('owner', 'manager', 'editor', 'reviewer', 'viewer')),
    permissions         JSONB DEFAULT '{}',
    joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_by           UUID REFERENCES core.users(id),
    
    UNIQUE(project_id, user_id)
);

-- Designs table
CREATE TABLE IF NOT EXISTS core.designs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    design_type         VARCHAR(100) NOT NULL DEFAULT 'floor_plan'
                        CHECK (design_type IN ('floor_plan', 'elevation', 'section', '3d_model', 'detail', 'sketch', 'concept', 'other')),
    file_format         VARCHAR(50),
    file_size_bytes     BIGINT,
    file_hash           VARCHAR(64),
    storage_path        VARCHAR(1000),
    thumbnail_url       VARCHAR(500),
    status              VARCHAR(50) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'in_progress', 'under_review', 'approved', 'archived')),
    scale               VARCHAR(50),
    unit                VARCHAR(20) DEFAULT 'mm'
                        CHECK (unit IN ('mm', 'cm', 'm', 'inch', 'foot')),
    bounds_min_x        DECIMAL(18, 6),
    bounds_min_y        DECIMAL(18, 6),
    bounds_max_x        DECIMAL(18, 6),
    bounds_max_y        DECIMAL(18, 6),
    element_count       INTEGER NOT NULL DEFAULT 0,
    layer_count         INTEGER NOT NULL DEFAULT 0,
    version_count       INTEGER NOT NULL DEFAULT 0,
    current_version_id  UUID,
    parent_design_id    UUID REFERENCES core.designs(id),
    is_template         BOOLEAN NOT NULL DEFAULT FALSE,
    template_category   VARCHAR(100),
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES core.users(id),
    updated_by          UUID REFERENCES core.users(id)
);

-- Design versions table
CREATE TABLE IF NOT EXISTS core.design_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    design_id           UUID NOT NULL REFERENCES core.designs(id) ON DELETE CASCADE,
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    version_number      INTEGER NOT NULL,
    version_name        VARCHAR(255),
    description         TEXT,
    change_summary      TEXT,
    snapshot_id         UUID,
    file_path           VARCHAR(1000),
    file_size_bytes     BIGINT,
    file_hash           VARCHAR(64),
    element_count       INTEGER NOT NULL DEFAULT 0,
    is_major_version    BOOLEAN NOT NULL DEFAULT FALSE,
    is_published        BOOLEAN NOT NULL DEFAULT FALSE,
    published_at        TIMESTAMPTZ,
    published_by        UUID REFERENCES core.users(id),
    parent_version_id   UUID REFERENCES core.design_versions(id),
    merge_source_id     UUID REFERENCES core.design_versions(id),
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    
    UNIQUE(design_id, version_number)
);

-- ============================================
-- Core Tables - Layers and Elements
-- ============================================

-- Layers table
CREATE TABLE IF NOT EXISTS core.layers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    design_id           UUID NOT NULL REFERENCES core.designs(id) ON DELETE CASCADE,
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    display_order       INTEGER NOT NULL DEFAULT 0,
    is_visible          BOOLEAN NOT NULL DEFAULT TRUE,
    is_locked           BOOLEAN NOT NULL DEFAULT FALSE,
    is_printable        BOOLEAN NOT NULL DEFAULT TRUE,
    color               VARCHAR(7) DEFAULT '#000000',
    line_type           VARCHAR(50) DEFAULT 'solid',
    line_weight         DECIMAL(5, 2) DEFAULT 0.25,
    transparency        INTEGER DEFAULT 0 CHECK (transparency BETWEEN 0 AND 100),
    element_count       INTEGER NOT NULL DEFAULT 0,
    parent_layer_id     UUID REFERENCES core.layers(id),
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    updated_by          UUID REFERENCES core.users(id),
    
    UNIQUE(design_id, name)
);

-- Elements table
CREATE TABLE IF NOT EXISTS core.elements (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    design_id           UUID NOT NULL REFERENCES core.designs(id) ON DELETE CASCADE,
    layer_id            UUID REFERENCES core.layers(id),
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    element_type        VARCHAR(100) NOT NULL
                        CHECK (element_type IN (
                            'wall', 'door', 'window', 'column', 'beam', 'slab', 'roof',
                            'stair', 'railing', 'furniture', 'equipment', 'text', 'dimension',
                            'line', 'polyline', 'circle', 'arc', 'rectangle', 'polygon',
                            'hatch', 'block', 'group', 'reference', 'other'
                        )),
    element_subtype     VARCHAR(100),
    name                VARCHAR(255),
    description         TEXT,
    properties          JSONB DEFAULT '{}',
    style               JSONB DEFAULT '{}',
    transform           JSONB DEFAULT '{"x": 0, "y": 0, "z": 0, "rotation": 0, "scaleX": 1, "scaleY": 1}',
    bounds_min_x        DECIMAL(18, 6),
    bounds_min_y        DECIMAL(18, 6),
    bounds_max_x        DECIMAL(18, 6),
    bounds_max_y        DECIMAL(18, 6),
    z_index             INTEGER DEFAULT 0,
    is_visible          BOOLEAN NOT NULL DEFAULT TRUE,
    is_locked           BOOLEAN NOT NULL DEFAULT FALSE,
    is_selectable       BOOLEAN NOT NULL DEFAULT TRUE,
    parent_element_id   UUID REFERENCES core.elements(id),
    reference_id        UUID,
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    updated_by          UUID REFERENCES core.users(id),
    deleted_at          TIMESTAMPTZ
);

-- ============================================
-- Collaboration Tables
-- ============================================

-- Collaboration sessions table
CREATE TABLE IF NOT EXISTS collaboration_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id         UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    session_type        VARCHAR(32) NOT NULL DEFAULT 'design'
                        CHECK (session_type IN ('design', 'review', 'presentation')),
    status              VARCHAR(32) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'paused', 'closing', 'closed')),
    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,
    metadata            JSONB DEFAULT '{}',
    yjs_state           BYTEA,
    server_clock        BIGINT DEFAULT 0
);

-- Session participants table
CREATE TABLE IF NOT EXISTS session_participants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL,
    user_name           VARCHAR(255),
    user_avatar         VARCHAR(500),
    permission_level    VARCHAR(32) NOT NULL DEFAULT 'viewer'
                        CHECK (permission_level IN ('viewer', 'commenter', 'editor', 'admin', 'owner')),
    client_type         VARCHAR(32),
    client_version      VARCHAR(32),
    client_platform     VARCHAR(32),
    cursor_position     JSONB,
    selection_range     JSONB,
    joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active           BOOLEAN DEFAULT TRUE,
    
    UNIQUE(session_id, user_id)
);

-- Operation logs table (partitioned by time)
CREATE TABLE IF NOT EXISTS operation_logs (
    id                  BIGSERIAL,
    session_id          UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    operation_id        UUID NOT NULL DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    client_clock        BIGINT NOT NULL,
    server_clock        BIGINT NOT NULL,
    operation_type      VARCHAR(32) NOT NULL
                        CHECK (operation_type IN (
                            'insert', 'update', 'delete', 'transform',
                            'property_change', 'geometry_change', 'style_change', 'layer_change'
                        )),
    target_id           UUID,
    operation_data      JSONB NOT NULL,
    yjs_update          BYTEA,
    metadata            JSONB DEFAULT '{}',
    is_undone           BOOLEAN DEFAULT FALSE,
    undone_at           TIMESTAMPTZ,
    undone_by           UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partition
CREATE TABLE IF NOT EXISTS operation_logs_2024_01 PARTITION OF operation_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ============================================
-- Permissions and Audit
-- ============================================

-- Permissions table
CREATE TABLE IF NOT EXISTS core.permissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(100) UNIQUE NOT NULL,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    resource_type       VARCHAR(100) NOT NULL,
    action              VARCHAR(100) NOT NULL,
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Roles table
CREATE TABLE IF NOT EXISTS core.roles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID REFERENCES core.tenants(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    is_default          BOOLEAN NOT NULL DEFAULT FALSE,
    permissions         JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, name)
);

-- User roles table
CREATE TABLE IF NOT EXISTS core.user_roles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role_id             UUID NOT NULL REFERENCES core.roles(id) ON DELETE CASCADE,
    scope_type          VARCHAR(50) NOT NULL DEFAULT 'tenant'
                        CHECK (scope_type IN ('tenant', 'project', 'team', 'design')),
    scope_id            UUID,
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by          UUID REFERENCES core.users(id),
    expires_at          TIMESTAMPTZ,
    
    UNIQUE(user_id, role_id, scope_type, scope_id)
);

-- Audit logs table (partitioned by time)
CREATE TABLE IF NOT EXISTS audit.audit_logs (
    id                  UUID,
    tenant_id           UUID NOT NULL,
    action              VARCHAR(100) NOT NULL
                        CHECK (action IN ('CREATE', 'READ', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'EXPORT', 'IMPORT', 'SHARE', 'PERMISSION_CHANGE')),
    entity_type         VARCHAR(100) NOT NULL,
    entity_id           UUID,
    before_data         JSONB,
    after_data          JSONB,
    changed_fields      TEXT[],
    user_id             UUID,
    user_name           VARCHAR(255),
    user_email          VARCHAR(255),
    request_id          UUID,
    session_id          UUID,
    correlation_id      UUID,
    source_ip           INET,
    user_agent          TEXT,
    source_service      VARCHAR(100),
    api_endpoint        VARCHAR(500),
    http_method         VARCHAR(10),
    success             BOOLEAN NOT NULL DEFAULT TRUE,
    error_code          VARCHAR(100),
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partition
CREATE TABLE IF NOT EXISTS audit.audit_logs_2024_01 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ============================================
-- Indexes
-- ============================================

-- Users indexes
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON core.users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON core.users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON core.users(status);

-- Projects indexes
CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON core.projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON core.projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_project_type ON core.projects(project_type);

-- Designs indexes
CREATE INDEX IF NOT EXISTS idx_designs_project_id ON core.designs(project_id);
CREATE INDEX IF NOT EXISTS idx_designs_tenant_id ON core.designs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_designs_design_type ON core.designs(design_type);
CREATE INDEX IF NOT EXISTS idx_designs_is_template ON core.designs(is_template) WHERE is_template = TRUE;

-- Layers indexes
CREATE INDEX IF NOT EXISTS idx_layers_design_id ON core.layers(design_id);
CREATE INDEX IF NOT EXISTS idx_layers_display_order ON core.layers(design_id, display_order);

-- Elements indexes
CREATE INDEX IF NOT EXISTS idx_elements_design_id ON core.elements(design_id);
CREATE INDEX IF NOT EXISTS idx_elements_layer_id ON core.elements(layer_id);
CREATE INDEX IF NOT EXISTS idx_elements_element_type ON core.elements(element_type);

-- Collaboration indexes
CREATE INDEX IF NOT EXISTS idx_sessions_document ON collaboration_sessions(document_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant ON collaboration_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON collaboration_sessions(status);
CREATE INDEX IF NOT EXISTS idx_participants_session ON session_participants(session_id);
CREATE INDEX IF NOT EXISTS idx_operations_session ON operation_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_operations_server_clock ON operation_logs(session_id, server_clock);

-- ============================================
-- Triggers
-- ============================================

-- Update updated_at trigger function
CREATE OR REPLACE FUNCTION core.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to tables
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON core.users
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_tenants_updated_at
    BEFORE UPDATE ON core.tenants
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_projects_updated_at
    BEFORE UPDATE ON core.projects
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_designs_updated_at
    BEFORE UPDATE ON core.designs
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_layers_updated_at
    BEFORE UPDATE ON core.layers
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_elements_updated_at
    BEFORE UPDATE ON core.elements
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

-- ============================================
-- Insert Default Data
-- ============================================

-- Insert default permissions
INSERT INTO core.permissions (code, name, resource_type, action, is_system) VALUES
('project:create', 'Create Project', 'project', 'create', true),
('project:read', 'Read Project', 'project', 'read', true),
('project:update', 'Update Project', 'project', 'update', true),
('project:delete', 'Delete Project', 'project', 'delete', true),
('design:create', 'Create Design', 'design', 'create', true),
('design:read', 'Read Design', 'design', 'read', true),
('design:update', 'Update Design', 'design', 'update', true),
('design:delete', 'Delete Design', 'design', 'delete', true),
('user:create', 'Create User', 'user', 'create', true),
('user:read', 'Read User', 'user', 'read', true),
('user:update', 'Update User', 'user', 'update', true),
('user:delete', 'Delete User', 'user', 'delete', true)
ON CONFLICT (code) DO NOTHING;

-- Insert system roles
INSERT INTO core.roles (id, name, description, is_system, is_default, permissions) VALUES
('00000000-0000-0000-0000-000000000001', 'Super Admin', 'Full system access', true, false, '["project:create", "project:read", "project:update", "project:delete", "user:create", "user:read", "user:update", "user:delete"]'),
('00000000-0000-0000-0000-000000000002', 'Admin', 'Administrative access', true, false, '["project:create", "project:read", "project:update", "project:delete", "user:read", "user:update"]'),
('00000000-0000-0000-0000-000000000003', 'Designer', 'Can create and edit designs', true, false, '["project:read", "design:create", "design:read", "design:update", "design:delete"]'),
('00000000-0000-0000-0000-000000000004', 'Viewer', 'Read-only access', true, true, '["project:read", "design:read"]')
ON CONFLICT DO NOTHING;
