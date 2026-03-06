-- ============================================
-- Project Service - Initial Schema
-- Version: 1.0.0
-- ============================================

CREATE SCHEMA IF NOT EXISTS project_service;

-- Projects table
CREATE TABLE project_service.projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL,
    
    -- Project metadata
    location VARCHAR(255),
    client_name VARCHAR(200),
    project_number VARCHAR(100),
    tags JSONB DEFAULT '[]',
    
    -- Settings
    settings JSONB DEFAULT '{
        "defaultUnitSystem": "METRIC",
        "autoSaveInterval": 30000,
        "maxVersions": 50
    }',
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' 
        CHECK (status IN ('ACTIVE', 'ARCHIVED', 'DELETED')),
    visibility VARCHAR(20) NOT NULL DEFAULT 'PRIVATE'
        CHECK (visibility IN ('PUBLIC', 'PRIVATE', 'TEAM')),
    
    -- Statistics (cached)
    member_count INTEGER DEFAULT 1,
    file_count INTEGER DEFAULT 0,
    total_storage_bytes BIGINT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit
    created_by UUID,
    updated_by UUID,
    archived_by UUID,
    deleted_by UUID
);

-- Project members table
CREATE TABLE project_service.project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES project_service.projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'VIEWER'
        CHECK (role IN ('OWNER', 'ADMIN', 'EDITOR', 'VIEWER')),
    
    -- Permissions override (JSON for flexibility)
    permissions JSONB DEFAULT '{}',
    
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    joined_by UUID,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID,
    
    UNIQUE (project_id, user_id)
);

-- Design files table
CREATE TABLE project_service.design_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES project_service.projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- File metadata
    file_type VARCHAR(50) NOT NULL,
    mime_type VARCHAR(100),
    file_size_bytes BIGINT,
    storage_key VARCHAR(500),
    
    -- Version control
    current_version_id UUID,
    version_count INTEGER DEFAULT 1,
    
    -- Status
    status VARCHAR(20) DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'LOCKED', 'ARCHIVED')),
    
    -- Locking
    locked_by UUID,
    locked_at TIMESTAMP WITH TIME ZONE,
    lock_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Thumbnail
    thumbnail_url VARCHAR(500),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- File versions table
CREATE TABLE project_service.file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES project_service.design_files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    
    -- Storage
    storage_key VARCHAR(500) NOT NULL,
    file_size_bytes BIGINT,
    checksum VARCHAR(64),
    
    -- Change info
    change_summary TEXT,
    changes JSONB,
    
    -- Creator
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (file_id, version_number)
);

-- Project invitations
CREATE TABLE project_service.project_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES project_service.projects(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'VIEWER',
    token VARCHAR(255) NOT NULL UNIQUE,
    
    invited_by UUID,
    invited_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    accepted_at TIMESTAMP WITH TIME ZONE,
    accepted_by UUID,
    
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    UNIQUE (project_id, email)
);

-- Project activity log
CREATE TABLE project_service.project_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES project_service.projects(id) ON DELETE CASCADE,
    user_id UUID,
    
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    
    details JSONB,
    ip_address VARCHAR(45),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_projects_owner ON project_service.projects(owner_id);
CREATE INDEX idx_projects_status ON project_service.projects(status);
CREATE INDEX idx_projects_created ON project_service.projects(created_at);
CREATE INDEX idx_project_members_project ON project_service.project_members(project_id);
CREATE INDEX idx_project_members_user ON project_service.project_members(user_id);
CREATE INDEX idx_design_files_project ON project_service.design_files(project_id);
CREATE INDEX idx_design_files_status ON project_service.design_files(status);
CREATE INDEX idx_project_activity_project ON project_service.project_activity(project_id);
CREATE INDEX idx_project_activity_created ON project_service.project_activity(created_at);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION project_service.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER projects_updated_at
    BEFORE UPDATE ON project_service.projects
    FOR EACH ROW
    EXECUTE FUNCTION project_service.update_updated_at();

CREATE TRIGGER project_members_updated_at
    BEFORE UPDATE ON project_service.project_members
    FOR EACH ROW
    EXECUTE FUNCTION project_service.update_updated_at();

-- Comments
COMMENT ON TABLE project_service.projects IS 'Architecture design projects';
COMMENT ON TABLE project_service.project_members IS 'Project membership and roles';
COMMENT ON TABLE project_service.design_files IS 'Design files within projects';
