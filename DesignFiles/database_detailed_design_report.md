# 详细设计阶段 - 数据库详细设计报告

## 半自动化建筑设计平台

**版本**: 1.0  
**日期**: 2024年  
**架构师**: 数据库架构团队

---

## 目录

1. [数据库架构概述](#1-数据库架构概述)
2. [完整数据库DDL](#2-完整数据库ddl)
3. [存储过程和函数](#3-存储过程和函数)
4. [视图设计](#4-视图设计)
5. [数据迁移脚本](#5-数据迁移脚本)
6. [性能优化](#6-性能优化)
7. [备份恢复脚本](#7-备份恢复脚本)
8. [数据库监控](#8-数据库监控)

---

## 1. 数据库架构概述

### 1.1 技术栈选型

| 组件 | 技术选型 | 用途 |
|------|----------|------|
| 主数据库 | YugabyteDB / CockroachDB | 分布式事务数据存储 |
| 几何存储 | PostgreSQL + PostGIS | 几何数据存储与空间查询 |
| 版本控制 | Event Sourcing + 快照 | 历史版本管理 |
| 缓存层 | Redis Cluster | 热点数据缓存 |
| 搜索引擎 | Elasticsearch | 全文搜索 |

### 1.2 数据库拓扑结构

```
                    ┌─────────────────────────────────────┐
                    │           应用服务层                 │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────┴──────────────────────┐
                    │           缓存层 (Redis)             │
                    └──────────────┬──────────────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
┌───────▼────────┐      ┌──────────▼──────────┐    ┌──────────▼──────────┐
│  YugabyteDB    │      │   PostgreSQL        │    │   Elasticsearch    │
│  (主数据库)     │      │   + PostGIS         │    │   (搜索引擎)        │
│                │      │   (几何存储)         │    │                    │
│ • 用户数据      │      │                     │    │ • 全文索引          │
│ • 项目数据      │      │ • 几何对象          │    │ • 属性搜索          │
│ • 权限数据      │      │ • 空间关系          │    │ • 聚合分析          │
│ • 审计日志      │      │ • 版本快照          │    │                    │
└────────────────┘      └─────────────────────┘    └─────────────────────┘
```

### 1.3 数据模型总览

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          核心实体关系图                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐              │
│   │   tenants   │────▶│   users     │◀────│   teams     │              │
│   │  (租户表)    │     │  (用户表)   │     │  (团队表)   │              │
│   └─────────────┘     └──────┬──────┘     └─────────────┘              │
│                              │                                          │
│                              │ owns                                     │
│                              ▼                                          │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐              │
│   │  projects   │◀────│  designs    │────▶│  versions   │              │
│   │  (项目表)   │     │  (设计表)   │     │ (版本表)    │              │
│   └─────────────┘     └──────┬──────┘     └─────────────┘              │
│                              │                                          │
│                              │ contains                                 │
│                              ▼                                          │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐              │
│   │   layers    │◀────│  elements   │────▶│ geometries  │              │
│   │  (图层表)   │     │ (元素表)    │     │ (几何表)    │              │
│   └─────────────┘     └─────────────┘     └─────────────┘              │
│                                                                         │
│   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐              │
│   │ permissions │◀────│ audit_logs  │     │  events     │              │
│   │  (权限表)   │     │ (审计日志)  │     │ (事件表)    │              │
│   └─────────────┘     └─────────────┘     └─────────────┘              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. 完整数据库DDL

### 2.1 数据库创建与基础配置

```sql
-- ============================================
-- 数据库创建与基础配置
-- ============================================

-- 创建主数据库
CREATE DATABASE archdesign_platform 
    WITH ENCODING = 'UTF8' 
    LC_COLLATE = 'en_US.UTF-8' 
    LC_CTYPE = 'en_US.UTF-8';

-- 创建几何数据库（PostgreSQL + PostGIS）
CREATE DATABASE archdesign_geometry 
    WITH ENCODING = 'UTF8';

-- 连接到几何数据库并启用PostGIS扩展
\c archdesign_geometry;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS postgis_raster;

-- 创建自定义Schema
CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS geometry;
CREATE SCHEMA IF NOT EXISTS versioning;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS analytics;

-- 设置搜索路径
SET search_path TO core, geometry, versioning, audit, analytics, public;
```

### 2.2 租户与用户模块

```sql
-- ============================================
-- 租户与用户模块 DDL
-- ============================================

-- 租户表
CREATE TABLE core.tenants (
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

-- 用户表
CREATE TABLE core.users (
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

-- 团队表
CREATE TABLE core.teams (
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

-- 团队成员关联表
CREATE TABLE core.team_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id             UUID NOT NULL REFERENCES core.teams(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role                VARCHAR(50) NOT NULL DEFAULT 'member'
                        CHECK (role IN ('leader', 'member')),
    joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(team_id, user_id)
);

-- API密钥表
CREATE TABLE core.api_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    user_id             UUID REFERENCES core.users(id) ON DELETE SET NULL,
    name                VARCHAR(255) NOT NULL,
    key_hash            VARCHAR(255) NOT NULL,
    key_prefix          VARCHAR(20) NOT NULL,
    scopes              TEXT[] NOT NULL DEFAULT '{}',
    expires_at          TIMESTAMPTZ,
    last_used_at        TIMESTAMPTZ,
    usage_count         INTEGER NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id)
);

-- 用户会话表
CREATE TABLE core.user_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    token               VARCHAR(500) NOT NULL,
    ip_address          INET,
    user_agent          TEXT,
    device_info         JSONB,
    expires_at          TIMESTAMPTZ NOT NULL,
    last_activity_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_valid            BOOLEAN NOT NULL DEFAULT TRUE
);

-- 索引定义
CREATE INDEX idx_users_tenant_id ON core.users(tenant_id);
CREATE INDEX idx_users_email ON core.users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON core.users(status);
CREATE INDEX idx_users_role ON core.users(role);
CREATE INDEX idx_teams_tenant_id ON core.teams(tenant_id);
CREATE INDEX idx_team_members_team_id ON core.team_members(team_id);
CREATE INDEX idx_team_members_user_id ON core.team_members(user_id);
CREATE INDEX idx_api_keys_tenant_id ON core.api_keys(tenant_id);
CREATE INDEX idx_api_keys_key_prefix ON core.api_keys(key_prefix);
CREATE INDEX idx_user_sessions_user_id ON core.user_sessions(user_id);
CREATE INDEX idx_user_sessions_token ON core.user_sessions(token);
CREATE INDEX idx_user_sessions_expires ON core.user_sessions(expires_at);

-- 部分索引：活跃租户
CREATE INDEX idx_tenants_active ON core.tenants(id) WHERE status = 'active';

-- 部分索引：活跃用户
CREATE INDEX idx_users_active ON core.users(id) WHERE status = 'active' AND deleted_at IS NULL;
```

### 2.3 项目与设计模块

```sql
-- ============================================
-- 项目与设计模块 DDL
-- ============================================

-- 项目表
CREATE TABLE core.projects (
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
    location            JSONB,  -- {country, city, address, coordinates: {lat, lng}}
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

-- 项目成员关联表
CREATE TABLE core.project_members (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role                VARCHAR(50) NOT NULL DEFAULT 'viewer'
                        CHECK (role IN ('owner', 'manager', 'editor', 'reviewer', 'viewer')),
    permissions         JSONB DEFAULT '{}',  -- 细粒度权限
    joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_by           UUID REFERENCES core.users(id),
    
    UNIQUE(project_id, user_id)
);

-- 设计表（设计文档/文件）
CREATE TABLE core.designs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    design_type         VARCHAR(100) NOT NULL DEFAULT 'floor_plan'
                        CHECK (design_type IN ('floor_plan', 'elevation', 'section', '3d_model', 'detail', 'sketch', 'concept', 'other')),
    file_format         VARCHAR(50),  -- dwg, dxf, ifc, revit, sketchup, etc.
    file_size_bytes     BIGINT,
    file_hash           VARCHAR(64),
    storage_path        VARCHAR(1000),
    thumbnail_url       VARCHAR(500),
    status              VARCHAR(50) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'in_progress', 'under_review', 'approved', 'archived')),
    scale               VARCHAR(50),  -- 1:100, 1:50, etc.
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

-- 设计版本表
CREATE TABLE core.design_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    design_id           UUID NOT NULL REFERENCES core.designs(id) ON DELETE CASCADE,
    project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
    version_number      INTEGER NOT NULL,
    version_name        VARCHAR(255),
    description         TEXT,
    change_summary      TEXT,
    snapshot_id         UUID,  -- 关联到几何数据库的快照
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

-- 索引定义
CREATE INDEX idx_projects_tenant_id ON core.projects(tenant_id);
CREATE INDEX idx_projects_status ON core.projects(status);
CREATE INDEX idx_projects_project_type ON core.projects(project_type);
CREATE INDEX idx_projects_created_by ON core.projects(created_by);
CREATE INDEX idx_projects_created_at ON core.projects(created_at);
CREATE INDEX idx_project_members_project_id ON core.project_members(project_id);
CREATE INDEX idx_project_members_user_id ON core.project_members(user_id);
CREATE INDEX idx_designs_project_id ON core.designs(project_id);
CREATE INDEX idx_designs_tenant_id ON core.designs(tenant_id);
CREATE INDEX idx_designs_design_type ON core.designs(design_type);
CREATE INDEX idx_designs_status ON core.designs(status);
CREATE INDEX idx_designs_is_template ON core.designs(is_template) WHERE is_template = TRUE;
CREATE INDEX idx_design_versions_design_id ON core.design_versions(design_id);
CREATE INDEX idx_design_versions_version_number ON core.design_versions(version_number);
CREATE INDEX idx_design_versions_is_published ON core.design_versions(is_published) WHERE is_published = TRUE;

-- 复合索引
CREATE INDEX idx_projects_tenant_status ON core.projects(tenant_id, status);
CREATE INDEX idx_designs_project_type ON core.designs(project_id, design_type);
CREATE INDEX idx_design_versions_design_created ON core.design_versions(design_id, created_at DESC);

-- GIN索引（JSONB查询）
CREATE INDEX idx_projects_location ON core.projects USING GIN(location);
CREATE INDEX idx_projects_tags ON core.projects USING GIN(tags);
CREATE INDEX idx_projects_settings ON core.projects USING GIN(settings);
CREATE INDEX idx_designs_metadata ON core.designs USING GIN(metadata);
```

### 2.4 图层与元素模块

```sql
-- ============================================
-- 图层与元素模块 DDL
-- ============================================

-- 图层表
CREATE TABLE core.layers (
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

-- 元素表（建筑元素：墙、门、窗等）
CREATE TABLE core.elements (
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
    properties          JSONB DEFAULT '{}',  -- 元素属性（材质、厚度等）
    style               JSONB DEFAULT '{}',  -- 样式信息
    transform           JSONB DEFAULT '{"x": 0, "y": 0, "z": 0, "rotation": 0, "scaleX": 1, "scaleY": 1}',  -- 变换矩阵
    bounds_min_x        DECIMAL(18, 6),
    bounds_min_y        DECIMAL(18, 6),
    bounds_max_x        DECIMAL(18, 6),
    bounds_max_y        DECIMAL(18, 6),
    z_index             INTEGER DEFAULT 0,
    is_visible          BOOLEAN NOT NULL DEFAULT TRUE,
    is_locked           BOOLEAN NOT NULL DEFAULT FALSE,
    is_selectable       BOOLEAN NOT NULL DEFAULT TRUE,
    parent_element_id   UUID REFERENCES core.elements(id),
    reference_id        UUID,  -- 引用外部元素
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    updated_by          UUID REFERENCES core.users(id),
    deleted_at          TIMESTAMPTZ
);

-- 元素属性历史表（用于追踪属性变更）
CREATE TABLE core.element_properties_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    element_id          UUID NOT NULL REFERENCES core.elements(id) ON DELETE CASCADE,
    version_id          UUID NOT NULL REFERENCES core.design_versions(id),
    properties          JSONB NOT NULL,
    changed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by          UUID REFERENCES core.users(id)
);

-- 元素关系表（元素之间的连接关系）
CREATE TABLE core.element_relations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_element_id   UUID NOT NULL REFERENCES core.elements(id) ON DELETE CASCADE,
    target_element_id   UUID NOT NULL REFERENCES core.elements(id) ON DELETE CASCADE,
    relation_type       VARCHAR(100) NOT NULL
                        CHECK (relation_type IN (
                            'connected', 'adjacent', 'contains', 'part_of', 
                            'supports', 'supported_by', 'aligned_with', 'parallel_to'
                        )),
    properties          JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(source_element_id, target_element_id, relation_type)
);

-- 索引定义
CREATE INDEX idx_layers_design_id ON core.layers(design_id);
CREATE INDEX idx_layers_project_id ON core.layers(project_id);
CREATE INDEX idx_layers_display_order ON core.layers(design_id, display_order);
CREATE INDEX idx_elements_design_id ON core.elements(design_id);
CREATE INDEX idx_elements_layer_id ON core.elements(layer_id);
CREATE INDEX idx_elements_element_type ON core.elements(element_type);
CREATE INDEX idx_elements_project_id ON core.elements(project_id);
CREATE INDEX idx_elements_parent ON core.elements(parent_element_id);
CREATE INDEX idx_elements_bounds ON core.elements USING GIST (
    BOX(POINT(bounds_min_x, bounds_min_y), POINT(bounds_max_x, bounds_max_y))
);
CREATE INDEX idx_element_history_element ON core.element_properties_history(element_id);
CREATE INDEX idx_element_history_version ON core.element_properties_history(version_id);
CREATE INDEX idx_element_relations_source ON core.element_relations(source_element_id);
CREATE INDEX idx_element_relations_target ON core.element_relations(target_element_id);

-- GIN索引
CREATE INDEX idx_elements_properties ON core.elements USING GIN(properties);
CREATE INDEX idx_elements_style ON core.elements USING GIN(style);
CREATE INDEX idx_elements_metadata ON core.elements USING GIN(metadata);
```

### 2.5 几何数据模块（PostGIS）

```sql
-- ============================================
-- 几何数据模块 DDL (PostGIS)
-- ============================================

-- 几何对象主表
CREATE TABLE geometry.geometries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    element_id          UUID NOT NULL,  -- 关联到core.elements
    design_id           UUID NOT NULL,
    project_id          UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    geometry_type       VARCHAR(50) NOT NULL
                        CHECK (geometry_type IN (
                            'point', 'multipoint', 'linestring', 'multilinestring',
                            'polygon', 'multipolygon', 'geometrycollection', 'curve', 'surface'
                        )),
    -- 2D几何（WGS84坐标系，SRID=4326）
    geom_2d             GEOMETRY(GEOMETRY, 4326),
    -- 3D几何（本地坐标系）
    geom_3d             GEOMETRY(GEOMETRYZ, 0),
    -- 简化几何（用于快速渲染）
    geom_simplified     GEOMETRY(GEOMETRY, 4326),
    -- 边界框（用于快速碰撞检测）
    bbox                GEOMETRY(POLYGON, 4326),
    -- 几何属性
    area                DECIMAL(18, 6),
    length              DECIMAL(18, 6),
    perimeter           DECIMAL(18, 6),
    centroid            GEOMETRY(POINT, 4326),
    -- 顶点数量（用于复杂度评估）
    vertex_count        INTEGER,
    -- 精度设置
    precision_mm        DECIMAL(10, 4) DEFAULT 1.0,
    -- 元数据
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version             INTEGER NOT NULL DEFAULT 1
);

-- 几何快照表（用于版本控制）
CREATE TABLE geometry.geometry_snapshots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id         UUID NOT NULL,  -- 关联到versioning.snapshots
    geometry_id         UUID NOT NULL REFERENCES geometry.geometries(id),
    element_id          UUID NOT NULL,
    design_id           UUID NOT NULL,
    geom_2d             GEOMETRY(GEOMETRY, 4326),
    geom_3d             GEOMETRY(GEOMETRYZ, 0),
    bbox                GEOMETRY(POLYGON, 4326),
    area                DECIMAL(18, 6),
    length              DECIMAL(18, 6),
    vertex_count        INTEGER,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(snapshot_id, geometry_id)
);

-- 空间索引表（R-tree索引加速空间查询）
CREATE TABLE geometry.spatial_index (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    geometry_id         UUID NOT NULL REFERENCES geometry.geometries(id),
    project_id          UUID NOT NULL,
    design_id           UUID NOT NULL,
    element_type        VARCHAR(100),
    -- 网格索引（用于快速区域查询）
    grid_x              INTEGER,
    grid_y              INTEGER,
    grid_level          INTEGER DEFAULT 0,
    -- 边界框坐标（用于非空间查询）
    min_x               DECIMAL(18, 6),
    min_y               DECIMAL(18, 6),
    max_x               DECIMAL(18, 6),
    max_y               DECIMAL(18, 6),
    
    UNIQUE(geometry_id, grid_level)
);

-- 空间关系表（预计算的空间关系）
CREATE TABLE geometry.spatial_relations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_geometry_id  UUID NOT NULL REFERENCES geometry.geometries(id) ON DELETE CASCADE,
    target_geometry_id  UUID NOT NULL REFERENCES geometry.geometries(id) ON DELETE CASCADE,
    relation_type       VARCHAR(50) NOT NULL
                        CHECK (relation_type IN (
                            'intersects', 'contains', 'within', 'touches', 
                            'crosses', 'overlaps', 'equals', 'disjoint', 'distance'
                        )),
    distance_mm         DECIMAL(18, 6),
    overlap_area        DECIMAL(18, 6),
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(source_geometry_id, target_geometry_id, relation_type)
);

-- PostGIS空间索引
CREATE INDEX idx_geometries_geom_2d ON geometry.geometries USING GIST(geom_2d);
CREATE INDEX idx_geometries_geom_3d ON geometry.geometries USING GIST(geom_3d);
CREATE INDEX idx_geometries_bbox ON geometry.geometries USING GIST(bbox);
CREATE INDEX idx_geometries_centroid ON geometry.geometries USING GIST(centroid);
CREATE INDEX idx_geometries_element ON geometry.geometries(element_id);
CREATE INDEX idx_geometries_design ON geometry.geometries(design_id);
CREATE INDEX idx_geometries_project ON geometry.geometries(project_id);
CREATE INDEX idx_geometries_type ON geometry.geometries(geometry_type);

CREATE INDEX idx_geometry_snapshots_snapshot ON geometry.geometry_snapshots(snapshot_id);
CREATE INDEX idx_geometry_snapshots_geom ON geometry.geometry_snapshots USING GIST(geom_2d);

CREATE INDEX idx_spatial_index_project ON geometry.spatial_index(project_id);
CREATE INDEX idx_spatial_index_design ON geometry.spatial_index(design_id);
CREATE INDEX idx_spatial_index_grid ON geometry.spatial_index(grid_x, grid_y, grid_level);
CREATE INDEX idx_spatial_index_bounds ON geometry.spatial_index(min_x, min_y, max_x, max_y);

CREATE INDEX idx_spatial_relations_source ON geometry.spatial_relations(source_geometry_id);
CREATE INDEX idx_spatial_relations_target ON geometry.spatial_relations(target_geometry_id);

-- BRIN索引（用于时间序列数据）
CREATE INDEX idx_geometries_created_brin ON geometry.geometries USING BRIN(created_at);
```

### 2.6 版本控制模块（Event Sourcing）

```sql
-- ============================================
-- 版本控制模块 DDL (Event Sourcing)
-- ============================================

-- 事件存储表（Event Store）
CREATE TABLE versioning.events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 聚合根信息
    aggregate_type      VARCHAR(100) NOT NULL,  -- 'project', 'design', 'element', etc.
    aggregate_id        UUID NOT NULL,
    -- 租户隔离
    tenant_id           UUID NOT NULL,
    -- 事件信息
    event_type          VARCHAR(200) NOT NULL,
    event_version       INTEGER NOT NULL DEFAULT 1,
    -- 事件数据
    payload             JSONB NOT NULL,
    metadata            JSONB DEFAULT '{}',
    -- 版本控制
    sequence_number     BIGINT NOT NULL,  -- 聚合内的事件序号
    global_sequence     BIGSERIAL,  -- 全局事件序号
    -- 因果链
    correlation_id      UUID,  -- 关联ID（用于追踪请求链）
    causation_id        UUID,  -- 因果ID（导致此事件的事件ID）
    -- 时间戳
    occurred_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 操作者
    user_id             UUID REFERENCES core.users(id),
    session_id          UUID,
    -- 来源
    source_ip           INET,
    source_service      VARCHAR(100),
    
    UNIQUE(aggregate_type, aggregate_id, sequence_number)
);

-- 快照表
CREATE TABLE versioning.snapshots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type      VARCHAR(100) NOT NULL,
    aggregate_id        UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    -- 快照版本
    version             INTEGER NOT NULL,
    sequence_number     BIGINT NOT NULL,  -- 快照对应的事件序号
    -- 快照数据
    state               JSONB NOT NULL,
    metadata            JSONB DEFAULT '{}',
    -- 统计信息
    event_count         INTEGER NOT NULL,  -- 从上一个快照到现在的事件数
    state_size_bytes    INTEGER,
    -- 时间戳
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,  -- 快照过期时间
    -- 创建者
    created_by          UUID REFERENCES core.users(id),
    
    UNIQUE(aggregate_type, aggregate_id, version)
);

-- 变更集表（用于批量操作）
CREATE TABLE versioning.change_sets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    project_id          UUID NOT NULL,
    design_id           UUID,
    name                VARCHAR(255),
    description         TEXT,
    -- 变更状态
    status              VARCHAR(50) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'applying', 'applied', 'failed', 'reverted')),
    -- 变更内容
    changes             JSONB NOT NULL,  -- [{event_type, aggregate_type, aggregate_id, payload}]
    -- 执行信息
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    error_message       TEXT,
    -- 回滚信息
    can_revert          BOOLEAN NOT NULL DEFAULT FALSE,
    reverted_at         TIMESTAMPTZ,
    reverted_by         UUID REFERENCES core.users(id),
    -- 时间戳
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 操作历史表（用于撤销/重做）
CREATE TABLE versioning.operation_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    project_id          UUID NOT NULL,
    design_id           UUID,
    user_id             UUID NOT NULL REFERENCES core.users(id),
    -- 操作信息
    operation_type      VARCHAR(100) NOT NULL,
    operation_name      VARCHAR(255),
    description         TEXT,
    -- 操作数据
    before_state        JSONB,
    after_state         JSONB,
    affected_elements   UUID[],
    -- 撤销/重做
    can_undo            BOOLEAN NOT NULL DEFAULT TRUE,
    undone_at           TIMESTAMPTZ,
    undone_by           UUID REFERENCES core.users(id),
    redo_of             UUID REFERENCES versioning.operation_history(id),
    -- 时间戳
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    session_id          UUID
);

-- 索引定义
CREATE INDEX idx_events_aggregate ON versioning.events(aggregate_type, aggregate_id);
CREATE INDEX idx_events_sequence ON versioning.events(aggregate_type, aggregate_id, sequence_number);
CREATE INDEX idx_events_global_seq ON versioning.events(global_sequence);
CREATE INDEX idx_events_tenant ON versioning.events(tenant_id);
CREATE INDEX idx_events_type ON versioning.events(event_type);
CREATE INDEX idx_events_occurred ON versioning.events(occurred_at);
CREATE INDEX idx_events_correlation ON versioning.events(correlation_id);
CREATE INDEX idx_events_user ON versioning.events(user_id);

CREATE INDEX idx_snapshots_aggregate ON versioning.snapshots(aggregate_type, aggregate_id);
CREATE INDEX idx_snapshots_version ON versioning.snapshots(aggregate_type, aggregate_id, version);
CREATE INDEX idx_snapshots_sequence ON versioning.snapshots(sequence_number);
CREATE INDEX idx_snapshots_tenant ON versioning.snapshots(tenant_id);
CREATE INDEX idx_snapshots_expires ON versioning.snapshots(expires_at);

CREATE INDEX idx_change_sets_tenant ON versioning.change_sets(tenant_id);
CREATE INDEX idx_change_sets_project ON versioning.change_sets(project_id);
CREATE INDEX idx_change_sets_status ON versioning.change_sets(status);
CREATE INDEX idx_change_sets_created ON versioning.change_sets(created_at);

CREATE INDEX idx_operation_history_project ON versioning.operation_history(project_id);
CREATE INDEX idx_operation_history_design ON versioning.operation_history(design_id);
CREATE INDEX idx_operation_history_user ON versioning.operation_history(user_id);
CREATE INDEX idx_operation_history_created ON versioning.operation_history(created_at DESC);
CREATE INDEX idx_operation_history_undo ON versioning.operation_history(undone_at) WHERE undone_at IS NULL;

-- GIN索引
CREATE INDEX idx_events_payload ON versioning.events USING GIN(payload);
CREATE INDEX idx_events_metadata ON versioning.events USING GIN(metadata);
CREATE INDEX idx_snapshots_state ON versioning.snapshots USING GIN(state);
CREATE INDEX idx_change_sets_changes ON versioning.change_sets USING GIN(changes);

-- BRIN索引（时间序列）
CREATE INDEX idx_events_created_brin ON versioning.events USING BRIN(occurred_at);
CREATE INDEX idx_snapshots_created_brin ON versioning.snapshots USING BRIN(created_at);
```

### 2.7 权限与访问控制模块

```sql
-- ============================================
-- 权限与访问控制模块 DDL
-- ============================================

-- 权限定义表
CREATE TABLE core.permissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(100) UNIQUE NOT NULL,
    name                VARCHAR(255) NOT NULL,
    description         TEXT,
    resource_type       VARCHAR(100) NOT NULL  -- 'project', 'design', 'element', etc.
                        CHECK (resource_type IN ('system', 'tenant', 'project', 'design', 'element', 'team', 'user')),
    action              VARCHAR(100) NOT NULL  -- 'create', 'read', 'update', 'delete', etc.
                        CHECK (action IN ('create', 'read', 'update', 'delete', 'manage', 'execute', 'share', 'admin')),
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 角色定义表
CREATE TABLE core.roles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID REFERENCES core.tenants(id) ON DELETE CASCADE,  -- NULL表示系统角色
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    is_system           BOOLEAN NOT NULL DEFAULT FALSE,
    is_default          BOOLEAN NOT NULL DEFAULT FALSE,
    permissions         JSONB NOT NULL DEFAULT '[]',  -- 权限代码列表
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, name)
);

-- 用户角色关联表
CREATE TABLE core.user_roles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
    role_id             UUID NOT NULL REFERENCES core.roles(id) ON DELETE CASCADE,
    scope_type          VARCHAR(50) NOT NULL DEFAULT 'tenant'  -- 'tenant', 'project', 'team'
                        CHECK (scope_type IN ('tenant', 'project', 'team', 'design')),
    scope_id            UUID,  -- 如果scope_type不是'tenant'，则需要指定scope_id
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by          UUID REFERENCES core.users(id),
    expires_at          TIMESTAMPTZ,
    
    UNIQUE(user_id, role_id, scope_type, scope_id)
);

-- 资源权限表（细粒度权限）
CREATE TABLE core.resource_permissions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type       VARCHAR(100) NOT NULL,
    resource_id         UUID NOT NULL,
    permission_code     VARCHAR(100) NOT NULL REFERENCES core.permissions(code),
    -- 权限主体
    principal_type      VARCHAR(50) NOT NULL  -- 'user', 'team', 'role'
                        CHECK (principal_type IN ('user', 'team', 'role')),
    principal_id        UUID NOT NULL,
    -- 权限设置
    is_allowed          BOOLEAN NOT NULL DEFAULT TRUE,
    conditions          JSONB DEFAULT '{}',  -- 权限条件（如时间限制、IP限制等）
    -- 继承设置
    inherit_to_children BOOLEAN NOT NULL DEFAULT TRUE,
    -- 时间戳
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID REFERENCES core.users(id),
    expires_at          TIMESTAMPTZ,
    
    UNIQUE(resource_type, resource_id, permission_code, principal_type, principal_id)
);

-- 访问控制列表缓存表
CREATE TABLE core.acl_cache (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    resource_type       VARCHAR(100) NOT NULL,
    resource_id         UUID NOT NULL,
    permissions         JSONB NOT NULL,  -- {permission_code: boolean}
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    
    UNIQUE(user_id, resource_type, resource_id)
);

-- 权限检查日志
CREATE TABLE core.permission_audit (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    resource_type       VARCHAR(100) NOT NULL,
    resource_id         UUID NOT NULL,
    permission_code     VARCHAR(100) NOT NULL,
    is_allowed          BOOLEAN NOT NULL,
    reason              TEXT,
    checked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_ip           INET,
    request_id          UUID
);

-- 索引定义
CREATE INDEX idx_permissions_resource ON core.permissions(resource_type);
CREATE INDEX idx_permissions_action ON core.permissions(action);
CREATE INDEX idx_roles_tenant ON core.roles(tenant_id);
CREATE INDEX idx_user_roles_user ON core.user_roles(user_id);
CREATE INDEX idx_user_roles_role ON core.user_roles(role_id);
CREATE INDEX idx_resource_permissions_resource ON core.resource_permissions(resource_type, resource_id);
CREATE INDEX idx_resource_permissions_principal ON core.resource_permissions(principal_type, principal_id);
CREATE INDEX idx_acl_cache_user ON core.acl_cache(user_id);
CREATE INDEX idx_acl_cache_expires ON core.acl_cache(expires_at);
CREATE INDEX idx_permission_audit_user ON core.permission_audit(user_id);
CREATE INDEX idx_permission_audit_resource ON core.permission_audit(resource_type, resource_id);
CREATE INDEX idx_permission_audit_checked ON core.permission_audit(checked_at);

-- GIN索引
CREATE INDEX idx_roles_permissions ON core.roles USING GIN(permissions);
CREATE INDEX idx_resource_permissions_conditions ON core.resource_permissions USING GIN(conditions);
CREATE INDEX idx_acl_cache_permissions ON core.acl_cache USING GIN(permissions);
```

### 2.8 审计日志模块

```sql
-- ============================================
-- 审计日志模块 DDL
-- ============================================

-- 审计日志主表（按时间分区）
CREATE TABLE audit.audit_logs (
    id                  UUID,
    tenant_id           UUID NOT NULL,
    -- 操作信息
    action              VARCHAR(100) NOT NULL
                        CHECK (action IN ('CREATE', 'READ', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'EXPORT', 'IMPORT', 'SHARE', 'PERMISSION_CHANGE')),
    entity_type         VARCHAR(100) NOT NULL,
    entity_id           UUID,
    -- 变更详情
    before_data         JSONB,
    after_data          JSONB,
    changed_fields      TEXT[],
    -- 操作者信息
    user_id             UUID,
    user_name           VARCHAR(255),
    user_email          VARCHAR(255),
    -- 请求信息
    request_id          UUID,
    session_id          UUID,
    correlation_id      UUID,
    -- 来源信息
    source_ip           INET,
    user_agent          TEXT,
    source_service      VARCHAR(100),
    api_endpoint        VARCHAR(500),
    http_method         VARCHAR(10),
    -- 结果
    success             BOOLEAN NOT NULL DEFAULT TRUE,
    error_code          VARCHAR(100),
    error_message       TEXT,
    -- 时间戳（分区键）
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 主键包含分区键
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建分区表（按月分区）
CREATE TABLE audit.audit_logs_2024_01 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE audit.audit_logs_2024_02 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE audit.audit_logs_2024_03 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE audit.audit_logs_2024_04 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE audit.audit_logs_2024_05 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE audit.audit_logs_2024_06 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE audit.audit_logs_2024_07 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE audit.audit_logs_2024_08 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE audit.audit_logs_2024_09 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE audit.audit_logs_2024_10 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE audit.audit_logs_2024_11 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE audit.audit_logs_2024_12 PARTITION OF audit.audit_logs
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- 审计日志归档表
CREATE TABLE audit.audit_logs_archive (
    LIKE audit.audit_logs INCLUDING ALL,
    archived_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archive_reason      VARCHAR(100)
);

-- 登录历史表
CREATE TABLE audit.login_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    -- 登录信息
    login_type          VARCHAR(50) NOT NULL DEFAULT 'password'
                        CHECK (login_type IN ('password', 'sso', 'api_key', 'mfa', 'oauth')),
    success             BOOLEAN NOT NULL,
    -- 失败信息
    failure_reason      VARCHAR(200),
    -- 会话信息
    session_id          UUID,
    token_id            UUID,
    -- 来源信息
    ip_address          INET,
    user_agent          TEXT,
    device_fingerprint  VARCHAR(255),
    geo_location        JSONB,  -- {country, city, coordinates}
    -- 时间戳
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    logout_at           TIMESTAMPTZ
);

-- 数据访问日志
CREATE TABLE audit.data_access_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    -- 访问信息
    access_type         VARCHAR(50) NOT NULL  -- 'query', 'export', 'download'
                        CHECK (access_type IN ('query', 'export', 'download', 'api_call')),
    resource_type       VARCHAR(100) NOT NULL,
    resource_id         UUID,
    -- 查询详情
    query_params        JSONB,
    result_count        INTEGER,
    -- 时间戳
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms         INTEGER
);

-- 索引定义
CREATE INDEX idx_audit_logs_tenant ON audit.audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_action ON audit.audit_logs(action);
CREATE INDEX idx_audit_logs_entity ON audit.audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_user ON audit.audit_logs(user_id);
CREATE INDEX idx_audit_logs_request ON audit.audit_logs(request_id);
CREATE INDEX idx_audit_logs_correlation ON audit.audit_logs(correlation_id);
CREATE INDEX idx_audit_logs_created ON audit.audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_source_ip ON audit.audit_logs(source_ip);

CREATE INDEX idx_login_history_tenant ON audit.login_history(tenant_id);
CREATE INDEX idx_login_history_user ON audit.login_history(user_id);
CREATE INDEX idx_login_history_created ON audit.login_history(created_at DESC);
CREATE INDEX idx_login_history_session ON audit.login_history(session_id);
CREATE INDEX idx_login_history_ip ON audit.login_history(ip_address);

CREATE INDEX idx_data_access_tenant ON audit.data_access_log(tenant_id);
CREATE INDEX idx_data_access_user ON audit.data_access_log(user_id);
CREATE INDEX idx_data_access_resource ON audit.data_access_log(resource_type, resource_id);
CREATE INDEX idx_data_access_created ON audit.data_access_log(created_at DESC);

-- GIN索引
CREATE INDEX idx_audit_logs_before ON audit.audit_logs USING GIN(before_data);
CREATE INDEX idx_audit_logs_after ON audit.audit_logs USING GIN(after_data);
CREATE INDEX idx_login_history_geo ON audit.login_history USING GIN(geo_location);
CREATE INDEX idx_data_access_params ON audit.data_access_log USING GIN(query_params);

-- BRIN索引（时间序列优化）
CREATE INDEX idx_audit_logs_created_brin ON audit.audit_logs USING BRIN(created_at);
CREATE INDEX idx_login_history_created_brin ON audit.login_history USING BRIN(created_at);
```

### 2.9 触发器定义

```sql
-- ============================================
-- 触发器定义
-- ============================================

-- 更新时间戳触发器函数
CREATE OR REPLACE FUNCTION core.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为需要自动更新updated_at的表创建触发器
CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON core.users
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_tenants_updated_at
    BEFORE UPDATE ON core.tenants
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

CREATE TRIGGER trigger_teams_updated_at
    BEFORE UPDATE ON core.teams
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

CREATE TRIGGER trigger_roles_updated_at
    BEFORE UPDATE ON core.roles
    FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();

-- 软删除触发器函数
CREATE OR REPLACE FUNCTION core.soft_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- 更新deleted_at而不是真正删除
    UPDATE TG_TABLE_NAME SET deleted_at = NOW() WHERE id = OLD.id;
    RETURN NULL;  -- 阻止实际删除
END;
$$ LANGUAGE plpgsql;

-- 审计日志触发器函数
CREATE OR REPLACE FUNCTION audit.log_changes()
RETURNS TRIGGER AS $$
DECLARE
    v_old_data JSONB;
    v_new_data JSONB;
    v_changed_fields TEXT[];
    v_user_id UUID;
    v_tenant_id UUID;
BEGIN
    -- 获取当前用户信息（从session或application设置）
    v_user_id := NULLIF(current_setting('app.current_user_id', TRUE), '')::UUID;
    v_tenant_id := NULLIF(current_setting('app.current_tenant_id', TRUE), '')::UUID;
    
    IF TG_OP = 'DELETE' THEN
        v_old_data := to_jsonb(OLD);
        v_new_data := NULL;
        v_changed_fields := NULL;
        
        INSERT INTO audit.audit_logs (
            id, tenant_id, action, entity_type, entity_id,
            before_data, after_data, changed_fields, user_id,
            created_at
        ) VALUES (
            gen_random_uuid(), v_tenant_id, 'DELETE', TG_TABLE_NAME, OLD.id,
            v_old_data, v_new_data, v_changed_fields, v_user_id,
            NOW()
        );
        RETURN OLD;
        
    ELSIF TG_OP = 'UPDATE' THEN
        v_old_data := to_jsonb(OLD);
        v_new_data := to_jsonb(NEW);
        
        -- 计算变更字段
        SELECT array_agg(key) INTO v_changed_fields
        FROM jsonb_each(v_new_data)
        WHERE v_old_data->key IS DISTINCT FROM value;
        
        -- 如果只有updated_at变化，不记录审计
        IF v_changed_fields = ARRAY['updated_at'] THEN
            RETURN NEW;
        END IF;
        
        INSERT INTO audit.audit_logs (
            id, tenant_id, action, entity_type, entity_id,
            before_data, after_data, changed_fields, user_id,
            created_at
        ) VALUES (
            gen_random_uuid(), v_tenant_id, 'UPDATE', TG_TABLE_NAME, NEW.id,
            v_old_data, v_new_data, v_changed_fields, v_user_id,
            NOW()
        );
        RETURN NEW;
        
    ELSIF TG_OP = 'INSERT' THEN
        v_old_data := NULL;
        v_new_data := to_jsonb(NEW);
        v_changed_fields := NULL;
        
        INSERT INTO audit.audit_logs (
            id, tenant_id, action, entity_type, entity_id,
            before_data, after_data, changed_fields, user_id,
            created_at
        ) VALUES (
            gen_random_uuid(), v_tenant_id, 'CREATE', TG_TABLE_NAME, NEW.id,
            v_old_data, v_new_data, v_changed_fields, v_user_id,
            NOW()
        );
        RETURN NEW;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 为关键表创建审计触发器
CREATE TRIGGER trigger_projects_audit
    AFTER INSERT OR UPDATE OR DELETE ON core.projects
    FOR EACH ROW EXECUTE FUNCTION audit.log_changes();

CREATE TRIGGER trigger_designs_audit
    AFTER INSERT OR UPDATE OR DELETE ON core.designs
    FOR EACH ROW EXECUTE FUNCTION audit.log_changes();

CREATE TRIGGER trigger_elements_audit
    AFTER INSERT OR UPDATE OR DELETE ON core.elements
    FOR EACH ROW EXECUTE FUNCTION audit.log_changes();

-- 元素计数自动更新触发器
CREATE OR REPLACE FUNCTION core.update_element_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE core.layers SET element_count = element_count + 1 WHERE id = NEW.layer_id;
        UPDATE core.designs SET element_count = element_count + 1 WHERE id = NEW.design_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE core.layers SET element_count = GREATEST(element_count - 1, 0) WHERE id = OLD.layer_id;
        UPDATE core.designs SET element_count = GREATEST(element_count - 1, 0) WHERE id = OLD.design_id;
    ELSIF TG_OP = 'UPDATE' AND NEW.layer_id IS DISTINCT FROM OLD.layer_id THEN
        UPDATE core.layers SET element_count = GREATEST(element_count - 1, 0) WHERE id = OLD.layer_id;
        UPDATE core.layers SET element_count = element_count + 1 WHERE id = NEW.layer_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_elements_count
    AFTER INSERT OR UPDATE OR DELETE ON core.elements
    FOR EACH ROW EXECUTE FUNCTION core.update_element_count();

-- 版本计数自动更新触发器
CREATE OR REPLACE FUNCTION core.update_version_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE core.designs SET version_count = version_count + 1, current_version_id = NEW.id 
        WHERE id = NEW.design_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE core.designs SET version_count = GREATEST(version_count - 1, 0) 
        WHERE id = OLD.design_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_design_versions_count
    AFTER INSERT OR DELETE ON core.design_versions
    FOR EACH ROW EXECUTE FUNCTION core.update_version_count();

-- 租户存储使用统计触发器
CREATE OR REPLACE FUNCTION core.update_tenant_storage()
RETURNS TRIGGER AS $$
DECLARE
    v_size_diff BIGINT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        v_size_diff := COALESCE(NEW.file_size_bytes, 0);
    ELSIF TG_OP = 'DELETE' THEN
        v_size_diff := -COALESCE(OLD.file_size_bytes, 0);
    ELSIF TG_OP = 'UPDATE' THEN
        v_size_diff := COALESCE(NEW.file_size_bytes, 0) - COALESCE(OLD.file_size_bytes, 0);
    END IF;
    
    IF v_size_diff != 0 THEN
        UPDATE core.tenants 
        SET storage_used_bytes = GREATEST(storage_used_bytes + v_size_diff, 0)
        WHERE id = COALESCE(NEW.tenant_id, OLD.tenant_id);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_designs_storage
    AFTER INSERT OR UPDATE OR DELETE ON core.designs
    FOR EACH ROW EXECUTE FUNCTION core.update_tenant_storage();

CREATE TRIGGER trigger_design_versions_storage
    AFTER INSERT OR UPDATE OR DELETE ON core.design_versions
    FOR EACH ROW EXECUTE FUNCTION core.update_tenant_storage();

-- 几何数据自动计算触发器
CREATE OR REPLACE FUNCTION geometry.auto_compute_geometry_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- 计算面积（仅对多边形类型）
    IF GeometryType(NEW.geom_2d) IN ('POLYGON', 'MULTIPOLYGON') THEN
        NEW.area := ST_Area(NEW.geom_2d)::DECIMAL(18, 6);
    END IF;
    
    -- 计算长度
    IF GeometryType(NEW.geom_2d) IN ('LINESTRING', 'MULTILINESTRING') THEN
        NEW.length := ST_Length(NEW.geom_2d)::DECIMAL(18, 6);
    END IF;
    
    -- 计算边界框
    NEW.bbox := ST_Envelope(NEW.geom_2d);
    
    -- 计算质心
    NEW.centroid := ST_Centroid(NEW.geom_2d);
    
    -- 计算顶点数量
    NEW.vertex_count := ST_NPoints(NEW.geom_2d);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_geometries_stats
    BEFORE INSERT OR UPDATE ON geometry.geometries
    FOR EACH ROW EXECUTE FUNCTION geometry.auto_compute_geometry_stats();

-- 事件发布触发器（用于Event Sourcing）
CREATE OR REPLACE FUNCTION versioning.publish_event()
RETURNS TRIGGER AS $$
DECLARE
    v_event_payload JSONB;
    v_sequence_number BIGINT;
BEGIN
    -- 获取下一个序列号
    SELECT COALESCE(MAX(sequence_number), 0) + 1 INTO v_sequence_number
    FROM versioning.events
    WHERE aggregate_type = TG_TABLE_NAME AND aggregate_id = NEW.id;
    
    -- 构建事件payload
    v_event_payload := jsonb_build_object(
        'table', TG_TABLE_NAME,
        'operation', TG_OP,
        'data', to_jsonb(NEW)
    );
    
    -- 插入事件
    INSERT INTO versioning.events (
        aggregate_type, aggregate_id, tenant_id, event_type,
        payload, sequence_number, occurred_at
    ) VALUES (
        TG_TABLE_NAME, NEW.id, NEW.tenant_id, 
        TG_OP || '_' || TG_TABLE_NAME,
        v_event_payload, v_sequence_number, NOW()
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```


---

## 3. 存储过程和函数

### 3.1 几何操作函数

```sql
-- ============================================
-- 几何操作函数
-- ============================================

-- 创建几何对象
CREATE OR REPLACE FUNCTION geometry.create_geometry(
    p_element_id UUID,
    p_design_id UUID,
    p_project_id UUID,
    p_tenant_id UUID,
    p_geometry_type VARCHAR(50),
    p_wkt_2d TEXT,
    p_wkt_3d TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT '{}'
)
RETURNS UUID AS $$
DECLARE
    v_geometry_id UUID;
    v_geom_2d GEOMETRY;
    v_geom_3d GEOMETRY;
BEGIN
    -- 解析WKT
    v_geom_2d := ST_GeomFromText(p_wkt_2d, 4326);
    
    IF p_wkt_3d IS NOT NULL THEN
        v_geom_3d := ST_GeomFromText(p_wkt_3d, 0);
    END IF;
    
    -- 插入几何数据
    INSERT INTO geometry.geometries (
        element_id, design_id, project_id, tenant_id,
        geometry_type, geom_2d, geom_3d, metadata
    ) VALUES (
        p_element_id, p_design_id, p_project_id, p_tenant_id,
        p_geometry_type, v_geom_2d, v_geom_3d, p_metadata
    ) RETURNING id INTO v_geometry_id;
    
    -- 更新空间索引
    PERFORM geometry.update_spatial_index(v_geometry_id);
    
    RETURN v_geometry_id;
END;
$$ LANGUAGE plpgsql;

-- 更新几何对象
CREATE OR REPLACE FUNCTION geometry.update_geometry(
    p_geometry_id UUID,
    p_wkt_2d TEXT DEFAULT NULL,
    p_wkt_3d TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_geom_2d GEOMETRY;
    v_geom_3d GEOMETRY;
BEGIN
    IF p_wkt_2d IS NOT NULL THEN
        v_geom_2d := ST_GeomFromText(p_wkt_2d, 4326);
    END IF;
    
    IF p_wkt_3d IS NOT NULL THEN
        v_geom_3d := ST_GeomFromText(p_wkt_3d, 0);
    END IF;
    
    UPDATE geometry.geometries SET
        geom_2d = COALESCE(v_geom_2d, geom_2d),
        geom_3d = COALESCE(v_geom_3d, geom_3d),
        metadata = COALESCE(p_metadata, metadata),
        version = version + 1,
        updated_at = NOW()
    WHERE id = p_geometry_id;
    
    -- 更新空间索引
    PERFORM geometry.update_spatial_index(p_geometry_id);
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- 更新空间索引
CREATE OR REPLACE FUNCTION geometry.update_spatial_index(p_geometry_id UUID)
RETURNS VOID AS $$
DECLARE
    v_rec RECORD;
    v_bbox GEOMETRY;
    v_grid_size INTEGER := 100;  -- 网格大小（米）
BEGIN
    SELECT geom_2d, project_id, design_id INTO v_rec
    FROM geometry.geometries WHERE id = p_geometry_id;
    
    IF v_rec.geom_2d IS NULL THEN
        RETURN;
    END IF;
    
    v_bbox := ST_Envelope(v_rec.geom_2d);
    
    -- 删除旧索引
    DELETE FROM geometry.spatial_index WHERE geometry_id = p_geometry_id;
    
    -- 插入新索引（多级网格）
    FOR i IN 0..3 LOOP
        INSERT INTO geometry.spatial_index (
            geometry_id, project_id, design_id,
            grid_x, grid_y, grid_level,
            min_x, min_y, max_x, max_y
        ) VALUES (
            p_geometry_id, v_rec.project_id, v_rec.design_id,
            FLOOR(ST_XMin(v_bbox) / (v_grid_size * (2^i)))::INTEGER,
            FLOOR(ST_YMin(v_bbox) / (v_grid_size * (2^i)))::INTEGER,
            i,
            ST_XMin(v_bbox), ST_YMin(v_bbox),
            ST_XMax(v_bbox), ST_YMax(v_bbox)
        );
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 空间查询：获取指定区域内的几何对象
CREATE OR REPLACE FUNCTION geometry.query_by_bbox(
    p_min_x DECIMAL,
    p_min_y DECIMAL,
    p_max_x DECIMAL,
    p_max_y DECIMAL,
    p_project_id UUID DEFAULT NULL,
    p_design_id UUID DEFAULT NULL
)
RETURNS TABLE (
    geometry_id UUID,
    element_id UUID,
    geometry_type VARCHAR(50),
    geom_2d GEOMETRY,
    area DECIMAL,
    length DECIMAL
) AS $$
DECLARE
    v_bbox GEOMETRY;
BEGIN
    v_bbox := ST_MakeEnvelope(p_min_x, p_min_y, p_max_x, p_max_y, 4326);
    
    RETURN QUERY
    SELECT 
        g.id, g.element_id, g.geometry_type, g.geom_2d, g.area, g.length
    FROM geometry.geometries g
    WHERE g.bbox && v_bbox  -- 边界框相交
      AND (p_project_id IS NULL OR g.project_id = p_project_id)
      AND (p_design_id IS NULL OR g.design_id = p_design_id)
      AND ST_Intersects(g.geom_2d, v_bbox)  -- 精确相交检测
    ORDER BY ST_Area(g.bbox) DESC;
END;
$$ LANGUAGE plpgsql;

-- 空间查询：获取指定点半径内的几何对象
CREATE OR REPLACE FUNCTION geometry.query_by_radius(
    p_center_x DECIMAL,
    p_center_y DECIMAL,
    p_radius_meters DECIMAL,
    p_project_id UUID DEFAULT NULL
)
RETURNS TABLE (
    geometry_id UUID,
    element_id UUID,
    geometry_type VARCHAR(50),
    distance_meters DECIMAL,
    geom_2d GEOMETRY
) AS $$
DECLARE
    v_center GEOMETRY;
    v_radius_degrees DECIMAL;
BEGIN
    v_center := ST_SetSRID(ST_MakePoint(p_center_x, p_center_y), 4326);
    -- 粗略转换：1度约等于111公里
    v_radius_degrees := p_radius_meters / 111000.0;
    
    RETURN QUERY
    SELECT 
        g.id, g.element_id, g.geometry_type,
        ST_Distance(g.geom_2d::GEOGRAPHY, v_center::GEOGRAPHY)::DECIMAL as distance_meters,
        g.geom_2d
    FROM geometry.geometries g
    WHERE ST_DWithin(g.geom_2d, v_center, v_radius_degrees)
      AND (p_project_id IS NULL OR g.project_id = p_project_id)
    ORDER BY distance_meters;
END;
$$ LANGUAGE plpgsql;

-- 几何简化
CREATE OR REPLACE FUNCTION geometry.simplify_geometry(
    p_geometry_id UUID,
    p_tolerance_meters DECIMAL DEFAULT 1.0
)
RETURNS GEOMETRY AS $$
DECLARE
    v_geom GEOMETRY;
    v_simplified GEOMETRY;
BEGIN
    SELECT geom_2d INTO v_geom FROM geometry.geometries WHERE id = p_geometry_id;
    
    IF v_geom IS NULL THEN
        RETURN NULL;
    END IF;
    
    -- 使用Douglas-Peucker算法简化
    v_simplified := ST_SimplifyPreserveTopology(v_geom, p_tolerance_meters / 111000.0);
    
    -- 更新简化几何
    UPDATE geometry.geometries 
    SET geom_simplified = v_simplified 
    WHERE id = p_geometry_id;
    
    RETURN v_simplified;
END;
$$ LANGUAGE plpgsql;

-- 计算几何属性
CREATE OR REPLACE FUNCTION geometry.compute_properties(p_geometry_id UUID)
RETURNS JSONB AS $$
DECLARE
    v_geom GEOMETRY;
    v_result JSONB;
BEGIN
    SELECT geom_2d INTO v_geom FROM geometry.geometries WHERE id = p_geometry_id;
    
    IF v_geom IS NULL THEN
        RETURN '{}'::JSONB;
    END IF;
    
    v_result := jsonb_build_object(
        'centroid', ST_AsText(ST_Centroid(v_geom)),
        'bounding_box', ST_AsText(ST_Envelope(v_geom)),
        'vertex_count', ST_NPoints(v_geom),
        'geometry_type', GeometryType(v_geom)
    );
    
    -- 根据几何类型添加特定属性
    IF GeometryType(v_geom) IN ('POLYGON', 'MULTIPOLYGON') THEN
        v_result := v_result || jsonb_build_object(
            'area', ST_Area(v_geom::GEOGRAPHY),
            'perimeter', ST_Perimeter(v_geom::GEOGRAPHY)
        );
    ELSIF GeometryType(v_geom) IN ('LINESTRING', 'MULTILINESTRING') THEN
        v_result := v_result || jsonb_build_object(
            'length', ST_Length(v_geom::GEOGRAPHY)
        );
    END IF;
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- 批量导入几何数据
CREATE OR REPLACE FUNCTION geometry.batch_import_geometries(
    p_geometries JSONB,  -- [{element_id, design_id, project_id, tenant_id, wkt, type, metadata}]
    p_batch_size INTEGER DEFAULT 1000
)
RETURNS TABLE (imported_count INTEGER, failed_count INTEGER, errors TEXT[]) AS $$
DECLARE
    v_geom RECORD;
    v_imported INTEGER := 0;
    v_failed INTEGER := 0;
    v_errors TEXT[] := ARRAY[]::TEXT[];
BEGIN
    FOR v_geom IN SELECT * FROM jsonb_array_elements(p_geometries)
    LOOP
        BEGIN
            PERFORM geometry.create_geometry(
                (v_geom->>'element_id')::UUID,
                (v_geom->>'design_id')::UUID,
                (v_geom->>'project_id')::UUID,
                (v_geom->>'tenant_id')::UUID,
                v_geom->>'type',
                v_geom->>'wkt',
                v_geom->>'wkt_3d',
                COALESCE(v_geom->'metadata', '{}')
            );
            v_imported := v_imported + 1;
        EXCEPTION WHEN OTHERS THEN
            v_failed := v_failed + 1;
            v_errors := array_append(v_errors, 
                format('Element %s: %s', v_geom->>'element_id', SQLERRM));
        END;
        
        -- 每批提交
        IF v_imported % p_batch_size = 0 THEN
            COMMIT;
        END IF;
    END LOOP;
    
    RETURN QUERY SELECT v_imported, v_failed, v_errors;
END;
$$ LANGUAGE plpgsql;

-- 导出几何为GeoJSON
CREATE OR REPLACE FUNCTION geometry.export_to_geojson(
    p_project_id UUID DEFAULT NULL,
    p_design_id UUID DEFAULT NULL,
    p_element_ids UUID[] DEFAULT NULL
)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    SELECT jsonb_build_object(
        'type', 'FeatureCollection',
        'features', jsonb_agg(
            jsonb_build_object(
                'type', 'Feature',
                'id', g.id,
                'geometry', ST_AsGeoJSON(g.geom_2d)::JSONB,
                'properties', jsonb_build_object(
                    'element_id', g.element_id,
                    'design_id', g.design_id,
                    'geometry_type', g.geometry_type,
                    'area', g.area,
                    'length', g.length
                ) || COALESCE(g.metadata, '{}')
            )
        )
    ) INTO v_result
    FROM geometry.geometries g
    WHERE (p_project_id IS NULL OR g.project_id = p_project_id)
      AND (p_design_id IS NULL OR g.design_id = p_design_id)
      AND (p_element_ids IS NULL OR g.element_id = ANY(p_element_ids));
    
    RETURN COALESCE(v_result, '{"type": "FeatureCollection", "features": []}'::JSONB);
END;
$$ LANGUAGE plpgsql;
```

### 3.2 版本控制函数

```sql
-- ============================================
-- 版本控制函数 (Event Sourcing)
-- ============================================

-- 追加事件
CREATE OR REPLACE FUNCTION versioning.append_event(
    p_aggregate_type VARCHAR(100),
    p_aggregate_id UUID,
    p_tenant_id UUID,
    p_event_type VARCHAR(200),
    p_payload JSONB,
    p_metadata JSONB DEFAULT '{}',
    p_correlation_id UUID DEFAULT NULL,
    p_causation_id UUID DEFAULT NULL,
    p_user_id UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_event_id UUID;
    v_sequence_number BIGINT;
BEGIN
    -- 获取下一个序列号
    SELECT COALESCE(MAX(sequence_number), 0) + 1 INTO v_sequence_number
    FROM versioning.events
    WHERE aggregate_type = p_aggregate_type AND aggregate_id = p_aggregate_id;
    
    -- 插入事件
    INSERT INTO versioning.events (
        id, aggregate_type, aggregate_id, tenant_id, event_type,
        payload, metadata, sequence_number, correlation_id, causation_id,
        user_id, occurred_at
    ) VALUES (
        gen_random_uuid(), p_aggregate_type, p_aggregate_id, p_tenant_id, p_event_type,
        p_payload, p_metadata, v_sequence_number, p_correlation_id, p_causation_id,
        p_user_id, NOW()
    ) RETURNING id INTO v_event_id;
    
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;

-- 获取聚合状态（通过重放事件）
CREATE OR REPLACE FUNCTION versioning.get_aggregate_state(
    p_aggregate_type VARCHAR(100),
    p_aggregate_id UUID,
    p_up_to_sequence BIGINT DEFAULT NULL
)
RETURNS JSONB AS $$
DECLARE
    v_state JSONB := '{}';
    v_event RECORD;
BEGIN
    FOR v_event IN
        SELECT event_type, payload
        FROM versioning.events
        WHERE aggregate_type = p_aggregate_type
          AND aggregate_id = p_aggregate_id
          AND (p_up_to_sequence IS NULL OR sequence_number <= p_up_to_sequence)
        ORDER BY sequence_number
    LOOP
        -- 根据事件类型应用状态变更
        v_state := versioning.apply_event(v_state, v_event.event_type, v_event.payload);
    END LOOP;
    
    RETURN v_state;
END;
$$ LANGUAGE plpgsql;

-- 应用事件到状态（事件处理器）
CREATE OR REPLACE FUNCTION versioning.apply_event(
    p_state JSONB,
    p_event_type VARCHAR(200),
    p_payload JSONB
)
RETURNS JSONB AS $$
DECLARE
    v_new_state JSONB;
BEGIN
    v_new_state := p_state;
    
    -- 根据事件类型处理
    CASE p_event_type
        WHEN 'PROJECT_CREATED' THEN
            v_new_state := p_payload;
        WHEN 'PROJECT_UPDATED' THEN
            v_new_state := v_new_state || p_payload;
        WHEN 'ELEMENT_CREATED' THEN
            v_new_state := jsonb_set(
                v_new_state, 
                ARRAY['elements'], 
                COALESCE(v_new_state->'elements', '[]'::JSONB) || jsonb_build_array(p_payload)
            );
        WHEN 'ELEMENT_UPDATED' THEN
            -- 更新元素
            v_new_state := versioning.update_element_in_state(v_new_state, p_payload);
        WHEN 'ELEMENT_DELETED' THEN
            -- 删除元素
            v_new_state := versioning.remove_element_from_state(v_new_state, p_payload->>'element_id');
        ELSE
            -- 默认：合并payload
            v_new_state := v_new_state || p_payload;
    END CASE;
    
    RETURN v_new_state;
END;
$$ LANGUAGE plpgsql;

-- 创建快照
CREATE OR REPLACE FUNCTION versioning.create_snapshot(
    p_aggregate_type VARCHAR(100),
    p_aggregate_id UUID,
    p_tenant_id UUID,
    p_version INTEGER DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_snapshot_id UUID;
    v_state JSONB;
    v_sequence_number BIGINT;
    v_event_count INTEGER;
    v_snapshot_version INTEGER;
BEGIN
    -- 获取当前状态
    v_state := versioning.get_aggregate_state(p_aggregate_type, p_aggregate_id);
    
    -- 获取最后事件序号
    SELECT MAX(sequence_number) INTO v_sequence_number
    FROM versioning.events
    WHERE aggregate_type = p_aggregate_type AND aggregate_id = p_aggregate_id;
    
    -- 计算版本号
    SELECT COALESCE(MAX(version), 0) + 1 INTO v_snapshot_version
    FROM versioning.snapshots
    WHERE aggregate_type = p_aggregate_type AND aggregate_id = p_aggregate_id;
    
    IF p_version IS NOT NULL THEN
        v_snapshot_version := p_version;
    END IF;
    
    -- 计算事件数量（从上一个快照）
    SELECT COUNT(*) INTO v_event_count
    FROM versioning.events
    WHERE aggregate_type = p_aggregate_type
      AND aggregate_id = p_aggregate_id
      AND sequence_number > COALESCE(
          (SELECT MAX(sequence_number) FROM versioning.snapshots 
           WHERE aggregate_type = p_aggregate_type AND aggregate_id = p_aggregate_id),
          0
      );
    
    -- 插入快照
    INSERT INTO versioning.snapshots (
        aggregate_type, aggregate_id, tenant_id, version,
        sequence_number, state, event_count, state_size_bytes, expires_at
    ) VALUES (
        p_aggregate_type, p_aggregate_id, p_tenant_id, v_snapshot_version,
        v_sequence_number, v_state, v_event_count, pg_column_size(v_state), NOW() + INTERVAL '30 days'
    ) RETURNING id INTO v_snapshot_id;
    
    RETURN v_snapshot_id;
END;
$$ LANGUAGE plpgsql;

-- 从快照恢复状态
CREATE OR REPLACE FUNCTION versioning.restore_from_snapshot(
    p_aggregate_type VARCHAR(100),
    p_aggregate_id UUID,
    p_version INTEGER DEFAULT NULL
)
RETURNS JSONB AS $$
DECLARE
    v_snapshot RECORD;
    v_state JSONB;
BEGIN
    -- 获取快照
    SELECT * INTO v_snapshot
    FROM versioning.snapshots
    WHERE aggregate_type = p_aggregate_type
      AND aggregate_id = p_aggregate_id
      AND (p_version IS NULL OR version = p_version)
    ORDER BY version DESC
    LIMIT 1;
    
    IF v_snapshot IS NULL THEN
        RETURN NULL;
    END IF;
    
    v_state := v_snapshot.state;
    
    -- 重放快照之后的事件
    FOR v_event IN
        SELECT event_type, payload
        FROM versioning.events
        WHERE aggregate_type = p_aggregate_type
          AND aggregate_id = p_aggregate_id
          AND sequence_number > v_snapshot.sequence_number
        ORDER BY sequence_number
    LOOP
        v_state := versioning.apply_event(v_state, v_event.event_type, v_event.payload);
    END LOOP;
    
    RETURN v_state;
END;
$$ LANGUAGE plpgsql;

-- 撤销操作
CREATE OR REPLACE FUNCTION versioning.undo_operation(
    p_operation_id UUID,
    p_user_id UUID
)
RETURNS BOOLEAN AS $$
DECLARE
    v_operation RECORD;
    v_inverse_operation JSONB;
BEGIN
    SELECT * INTO v_operation
    FROM versioning.operation_history
    WHERE id = p_operation_id AND undone_at IS NULL;
    
    IF v_operation IS NULL OR NOT v_operation.can_undo THEN
        RETURN FALSE;
    END IF;
    
    -- 构建逆向操作
    v_inverse_operation := jsonb_build_object(
        'before_state', v_operation.after_state,
        'after_state', v_operation.before_state
    );
    
    -- 应用逆向操作
    PERFORM versioning.apply_event('{}', v_operation.operation_type || '_UNDO', v_inverse_operation);
    
    -- 标记为已撤销
    UPDATE versioning.operation_history
    SET undone_at = NOW(), undone_by = p_user_id
    WHERE id = p_operation_id;
    
    -- 记录重做链
    INSERT INTO versioning.operation_history (
        tenant_id, project_id, design_id, user_id,
        operation_type, operation_name, description,
        before_state, after_state, redo_of, can_undo
    ) VALUES (
        v_operation.tenant_id, v_operation.project_id, v_operation.design_id, p_user_id,
        v_operation.operation_type, 'UNDO: ' || v_operation.operation_name, 
        'Undo of: ' || v_operation.description,
        v_operation.after_state, v_operation.before_state, p_operation_id, FALSE
    );
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- 重做操作
CREATE OR REPLACE FUNCTION versioning.redo_operation(
    p_undo_operation_id UUID,
    p_user_id UUID
)
RETURNS BOOLEAN AS $$
DECLARE
    v_undo_operation RECORD;
    v_original_operation RECORD;
BEGIN
    SELECT * INTO v_undo_operation
    FROM versioning.operation_history
    WHERE id = p_undo_operation_id AND redo_of IS NOT NULL;
    
    IF v_undo_operation IS NULL THEN
        RETURN FALSE;
    END IF;
    
    SELECT * INTO v_original_operation
    FROM versioning.operation_history
    WHERE id = v_undo_operation.redo_of;
    
    -- 重新应用原始操作
    PERFORM versioning.apply_event('{}', v_original_operation.operation_type, jsonb_build_object(
        'before_state', v_original_operation.before_state,
        'after_state', v_original_operation.after_state
    ));
    
    -- 记录新操作
    INSERT INTO versioning.operation_history (
        tenant_id, project_id, design_id, user_id,
        operation_type, operation_name, description,
        before_state, after_state, can_undo
    ) VALUES (
        v_original_operation.tenant_id, v_original_operation.project_id, 
        v_original_operation.design_id, p_user_id,
        v_original_operation.operation_type, 'REDO: ' || v_original_operation.operation_name,
        'Redo of: ' || v_original_operation.description,
        v_original_operation.before_state, v_original_operation.after_state, TRUE
    );
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- 获取版本历史
CREATE OR REPLACE FUNCTION versioning.get_version_history(
    p_aggregate_type VARCHAR(100),
    p_aggregate_id UUID,
    p_limit INTEGER DEFAULT 100,
    p_offset INTEGER DEFAULT 0
)
RETURNS TABLE (
    version INTEGER,
    sequence_number BIGINT,
    event_type VARCHAR(200),
    user_id UUID,
    occurred_at TIMESTAMPTZ,
    change_summary TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.version,
        s.sequence_number,
        'SNAPSHOT'::VARCHAR(200) as event_type,
        s.created_by as user_id,
        s.created_at as occurred_at,
        'Snapshot created'::TEXT as change_summary
    FROM versioning.snapshots s
    WHERE s.aggregate_type = p_aggregate_type AND s.aggregate_id = p_aggregate_id
    
    UNION ALL
    
    SELECT 
        NULL::INTEGER as version,
        e.sequence_number,
        e.event_type,
        e.user_id,
        e.occurred_at,
        LEFT(e.payload::TEXT, 200) as change_summary
    FROM versioning.events e
    WHERE e.aggregate_type = p_aggregate_type 
      AND e.aggregate_id = p_aggregate_id
      AND e.sequence_number > COALESCE(
          (SELECT MAX(sequence_number) FROM versioning.snapshots 
           WHERE aggregate_type = p_aggregate_type AND aggregate_id = p_aggregate_id),
          0
      )
    
    ORDER BY sequence_number DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- 清理过期快照
CREATE OR REPLACE FUNCTION versioning.cleanup_expired_snapshots(
    p_batch_size INTEGER DEFAULT 1000
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM versioning.snapshots
        WHERE expires_at < NOW()
        LIMIT p_batch_size
        RETURNING id
    )
    SELECT COUNT(*) INTO v_deleted FROM deleted;
    
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 辅助函数：在状态中更新元素
CREATE OR REPLACE FUNCTION versioning.update_element_in_state(
    p_state JSONB,
    p_element JSONB
)
RETURNS JSONB AS $$
DECLARE
    v_elements JSONB;
    v_element_id TEXT;
    v_index INTEGER;
BEGIN
    v_element_id := p_element->>'id';
    v_elements := COALESCE(p_state->'elements', '[]'::JSONB);
    
    -- 查找元素索引
    SELECT i INTO v_index
    FROM generate_series(0, jsonb_array_length(v_elements) - 1) AS i
    WHERE v_elements->i->>'id' = v_element_id;
    
    IF v_index IS NOT NULL THEN
        -- 更新元素
        v_elements := jsonb_set(
            v_elements, 
            ARRAY[v_index::TEXT], 
            v_elements->v_index || p_element
        );
    ELSE
        -- 添加新元素
        v_elements := v_elements || jsonb_build_array(p_element);
    END IF;
    
    RETURN jsonb_set(p_state, ARRAY['elements'], v_elements);
END;
$$ LANGUAGE plpgsql;

-- 辅助函数：从状态中移除元素
CREATE OR REPLACE FUNCTION versioning.remove_element_from_state(
    p_state JSONB,
    p_element_id TEXT
)
RETURNS JSONB AS $$
DECLARE
    v_elements JSONB;
BEGIN
    v_elements := COALESCE(p_state->'elements', '[]'::JSONB);
    
    -- 过滤掉指定元素
    v_elements := (
        SELECT jsonb_agg(elem)
        FROM jsonb_array_elements(v_elements) AS elem
        WHERE elem->>'id' != p_element_id
    );
    
    RETURN jsonb_set(p_state, ARRAY['elements'], COALESCE(v_elements, '[]'::JSONB));
END;
$$ LANGUAGE plpgsql;
```

### 3.3 权限检查函数

```sql
-- ============================================
-- 权限检查函数
-- ============================================

-- 检查用户权限
CREATE OR REPLACE FUNCTION core.check_permission(
    p_user_id UUID,
    p_resource_type VARCHAR(100),
    p_resource_id UUID,
    p_permission_code VARCHAR(100)
)
RETURNS BOOLEAN AS $$
DECLARE
    v_has_permission BOOLEAN := FALSE;
    v_tenant_id UUID;
    v_is_super_admin BOOLEAN;
BEGIN
    -- 获取用户租户
    SELECT tenant_id INTO v_tenant_id FROM core.users WHERE id = p_user_id;
    
    -- 检查是否是超级管理员
    SELECT role = 'super_admin' INTO v_is_super_admin
    FROM core.users WHERE id = p_user_id;
    
    -- 超级管理员拥有所有权限
    IF v_is_super_admin THEN
        RETURN TRUE;
    END IF;
    
    -- 检查ACL缓存
    SELECT (permissions->p_permission_code)::BOOLEAN INTO v_has_permission
    FROM core.acl_cache
    WHERE user_id = p_user_id 
      AND resource_type = p_resource_type 
      AND resource_id = p_resource_id
      AND expires_at > NOW();
    
    IF v_has_permission IS NOT NULL THEN
        RETURN v_has_permission;
    END IF;
    
    -- 检查直接权限
    SELECT EXISTS (
        SELECT 1 FROM core.resource_permissions
        WHERE resource_type = p_resource_type
          AND resource_id = p_resource_id
          AND permission_code = p_permission_code
          AND principal_type = 'user'
          AND principal_id = p_user_id
          AND is_allowed = TRUE
          AND (expires_at IS NULL OR expires_at > NOW())
    ) INTO v_has_permission;
    
    IF v_has_permission THEN
        RETURN TRUE;
    END IF;
    
    -- 检查组权限
    SELECT EXISTS (
        SELECT 1 FROM core.resource_permissions rp
        JOIN core.team_members tm ON rp.principal_id = tm.team_id
        WHERE rp.resource_type = p_resource_type
          AND rp.resource_id = p_resource_id
          AND rp.permission_code = p_permission_code
          AND rp.principal_type = 'team'
          AND tm.user_id = p_user_id
          AND rp.is_allowed = TRUE
          AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
    ) INTO v_has_permission;
    
    IF v_has_permission THEN
        RETURN TRUE;
    END IF;
    
    -- 检查角色权限
    SELECT EXISTS (
        SELECT 1 FROM core.resource_permissions rp
        JOIN core.user_roles ur ON rp.principal_id = ur.role_id
        WHERE rp.resource_type = p_resource_type
          AND rp.resource_id = p_resource_id
          AND rp.permission_code = p_permission_code
          AND rp.principal_type = 'role'
          AND ur.user_id = p_user_id
          AND rp.is_allowed = TRUE
          AND (rp.expires_at IS NULL OR rp.expires_at > NOW())
    ) INTO v_has_permission;
    
    -- 缓存结果
    INSERT INTO core.acl_cache (user_id, resource_type, resource_id, permissions, expires_at)
    VALUES (
        p_user_id, p_resource_type, p_resource_id,
        jsonb_build_object(p_permission_code, v_has_permission),
        NOW() + INTERVAL '5 minutes'
    )
    ON CONFLICT (user_id, resource_type, resource_id) 
    DO UPDATE SET 
        permissions = core.acl_cache.permissions || jsonb_build_object(p_permission_code, v_has_permission),
        computed_at = NOW(),
        expires_at = NOW() + INTERVAL '5 minutes';
    
    RETURN v_has_permission;
END;
$$ LANGUAGE plpgsql;

-- 批量检查权限
CREATE OR REPLACE FUNCTION core.check_permissions(
    p_user_id UUID,
    p_resource_type VARCHAR(100),
    p_resource_id UUID,
    p_permission_codes TEXT[]
)
RETURNS TABLE (permission_code VARCHAR(100), is_allowed BOOLEAN) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        code::VARCHAR(100),
        core.check_permission(p_user_id, p_resource_type, p_resource_id, code)
    FROM unnest(p_permission_codes) AS code;
END;
$$ LANGUAGE plpgsql;

-- 获取用户所有权限
CREATE OR REPLACE FUNCTION core.get_user_permissions(
    p_user_id UUID,
    p_resource_type VARCHAR(100) DEFAULT NULL,
    p_resource_id UUID DEFAULT NULL
)
RETURNS TABLE (
    resource_type VARCHAR(100),
    resource_id UUID,
    permission_code VARCHAR(100),
    granted_via VARCHAR(50)
) AS $$
BEGIN
    -- 直接权限
    RETURN QUERY
    SELECT 
        rp.resource_type::VARCHAR(100),
        rp.resource_id,
        rp.permission_code::VARCHAR(100),
        'direct'::VARCHAR(50) as granted_via
    FROM core.resource_permissions rp
    WHERE rp.principal_type = 'user'
      AND rp.principal_id = p_user_id
      AND rp.is_allowed = TRUE
      AND (p_resource_type IS NULL OR rp.resource_type = p_resource_type)
      AND (p_resource_id IS NULL OR rp.resource_id = p_resource_id)
      AND (rp.expires_at IS NULL OR rp.expires_at > NOW());
    
    -- 组权限
    RETURN QUERY
    SELECT 
        rp.resource_type::VARCHAR(100),
        rp.resource_id,
        rp.permission_code::VARCHAR(100),
        'team'::VARCHAR(50) as granted_via
    FROM core.resource_permissions rp
    JOIN core.team_members tm ON rp.principal_id = tm.team_id
    WHERE rp.principal_type = 'team'
      AND tm.user_id = p_user_id
      AND rp.is_allowed = TRUE
      AND (p_resource_type IS NULL OR rp.resource_type = p_resource_type)
      AND (p_resource_id IS NULL OR rp.resource_id = p_resource_id)
      AND (rp.expires_at IS NULL OR rp.expires_at > NOW());
    
    -- 角色权限
    RETURN QUERY
    SELECT 
        rp.resource_type::VARCHAR(100),
        rp.resource_id,
        rp.permission_code::VARCHAR(100),
        'role'::VARCHAR(50) as granted_via
    FROM core.resource_permissions rp
    JOIN core.user_roles ur ON rp.principal_id = ur.role_id
    WHERE rp.principal_type = 'role'
      AND ur.user_id = p_user_id
      AND rp.is_allowed = TRUE
      AND (p_resource_type IS NULL OR rp.resource_type = p_resource_type)
      AND (p_resource_id IS NULL OR rp.resource_id = p_resource_id)
      AND (rp.expires_at IS NULL OR rp.expires_at > NOW());
END;
$$ LANGUAGE plpgsql;

-- 授予权限
CREATE OR REPLACE FUNCTION core.grant_permission(
    p_resource_type VARCHAR(100),
    p_resource_id UUID,
    p_permission_code VARCHAR(100),
    p_principal_type VARCHAR(50),
    p_principal_id UUID,
    p_granted_by UUID,
    p_conditions JSONB DEFAULT '{}',
    p_expires_at TIMESTAMPTZ DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_permission_id UUID;
BEGIN
    INSERT INTO core.resource_permissions (
        resource_type, resource_id, permission_code,
        principal_type, principal_id, is_allowed,
        conditions, created_by, expires_at
    ) VALUES (
        p_resource_type, p_resource_id, p_permission_code,
        p_principal_type, p_principal_id, TRUE,
        p_conditions, p_granted_by, p_expires_at
    )
    ON CONFLICT (resource_type, resource_id, permission_code, principal_type, principal_id)
    DO UPDATE SET 
        is_allowed = TRUE,
        conditions = p_conditions,
        expires_at = p_expires_at
    RETURNING id INTO v_permission_id;
    
    -- 清除ACL缓存
    DELETE FROM core.acl_cache WHERE resource_type = p_resource_type AND resource_id = p_resource_id;
    
    RETURN v_permission_id;
END;
$$ LANGUAGE plpgsql;

-- 撤销权限
CREATE OR REPLACE FUNCTION core.revoke_permission(
    p_resource_type VARCHAR(100),
    p_resource_id UUID,
    p_permission_code VARCHAR(100),
    p_principal_type VARCHAR(50),
    p_principal_id UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE core.resource_permissions
    SET is_allowed = FALSE
    WHERE resource_type = p_resource_type
      AND resource_id = p_resource_id
      AND permission_code = p_permission_code
      AND principal_type = p_principal_type
      AND principal_id = p_principal_id;
    
    -- 清除ACL缓存
    DELETE FROM core.acl_cache WHERE resource_type = p_resource_type AND resource_id = p_resource_id;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- 清除用户ACL缓存
CREATE OR REPLACE FUNCTION core.invalidate_acl_cache(
    p_user_id UUID DEFAULT NULL,
    p_resource_type VARCHAR(100) DEFAULT NULL,
    p_resource_id UUID DEFAULT NULL
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM core.acl_cache
    WHERE (p_user_id IS NULL OR user_id = p_user_id)
      AND (p_resource_type IS NULL OR resource_type = p_resource_type)
      AND (p_resource_id IS NULL OR resource_id = p_resource_id);
    
    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 检查项目访问权限
CREATE OR REPLACE FUNCTION core.check_project_access(
    p_user_id UUID,
    p_project_id UUID,
    p_required_role VARCHAR(50) DEFAULT 'viewer'
)
RETURNS BOOLEAN AS $$
DECLARE
    v_has_access BOOLEAN;
    v_project_tenant_id UUID;
    v_user_tenant_id UUID;
    v_user_role VARCHAR(50);
BEGIN
    -- 获取项目租户
    SELECT tenant_id INTO v_project_tenant_id FROM core.projects WHERE id = p_project_id;
    
    -- 获取用户租户
    SELECT tenant_id, role INTO v_user_tenant_id, v_user_role 
    FROM core.users WHERE id = p_user_id;
    
    -- 检查租户匹配
    IF v_project_tenant_id != v_user_tenant_id THEN
        RETURN FALSE;
    END IF;
    
    -- 检查是否是超级管理员或租户管理员
    IF v_user_role IN ('super_admin', 'admin') THEN
        RETURN TRUE;
    END IF;
    
    -- 检查项目成员权限
    SELECT role INTO v_user_role
    FROM core.project_members
    WHERE project_id = p_project_id AND user_id = p_user_id;
    
    IF v_user_role IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- 角色权限映射
    CASE p_required_role
        WHEN 'viewer' THEN RETURN TRUE;
        WHEN 'editor' THEN RETURN v_user_role IN ('editor', 'manager', 'owner');
        WHEN 'manager' THEN RETURN v_user_role IN ('manager', 'owner');
        WHEN 'owner' THEN RETURN v_user_role = 'owner';
        ELSE RETURN FALSE;
    END CASE;
END;
$$ LANGUAGE plpgsql;
```

### 3.4 审计日志函数

```sql
-- ============================================
-- 审计日志函数
-- ============================================

-- 记录审计日志
CREATE OR REPLACE FUNCTION audit.log_audit(
    p_tenant_id UUID,
    p_action VARCHAR(100),
    p_entity_type VARCHAR(100),
    p_entity_id UUID,
    p_before_data JSONB DEFAULT NULL,
    p_after_data JSONB DEFAULT NULL,
    p_user_id UUID DEFAULT NULL,
    p_request_id UUID DEFAULT NULL,
    p_source_ip INET DEFAULT NULL,
    p_success BOOLEAN DEFAULT TRUE,
    p_error_message TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
    v_user_name VARCHAR(255);
    v_user_email VARCHAR(255);
    v_changed_fields TEXT[];
BEGIN
    -- 获取用户信息
    IF p_user_id IS NOT NULL THEN
        SELECT first_name || ' ' || last_name, email 
        INTO v_user_name, v_user_email
        FROM core.users WHERE id = p_user_id;
    END IF;
    
    -- 计算变更字段
    IF p_before_data IS NOT NULL AND p_after_data IS NOT NULL THEN
        SELECT array_agg(key) INTO v_changed_fields
        FROM jsonb_each(p_after_data)
        WHERE p_before_data->key IS DISTINCT FROM value;
    END IF;
    
    INSERT INTO audit.audit_logs (
        id, tenant_id, action, entity_type, entity_id,
        before_data, after_data, changed_fields,
        user_id, user_name, user_email, request_id,
        source_ip, success, error_message, created_at
    ) VALUES (
        gen_random_uuid(), p_tenant_id, p_action, p_entity_type, p_entity_id,
        p_before_data, p_after_data, v_changed_fields,
        p_user_id, v_user_name, v_user_email, p_request_id,
        p_source_ip, p_success, p_error_message, NOW()
    ) RETURNING id INTO v_log_id;
    
    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql;

-- 记录登录历史
CREATE OR REPLACE FUNCTION audit.log_login(
    p_tenant_id UUID,
    p_user_id UUID,
    p_login_type VARCHAR(50),
    p_success BOOLEAN,
    p_failure_reason VARCHAR(200) DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_session_id UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
    v_geo_location JSONB;
BEGIN
    -- 这里可以集成IP地理位置服务
    v_geo_location := '{}'::JSONB;
    
    INSERT INTO audit.login_history (
        tenant_id, user_id, login_type, success, failure_reason,
        ip_address, user_agent, session_id, geo_location
    ) VALUES (
        p_tenant_id, p_user_id, p_login_type, p_success, p_failure_reason,
        p_ip_address, p_user_agent, p_session_id, v_geo_location
    ) RETURNING id INTO v_log_id;
    
    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql;

-- 记录登出
CREATE OR REPLACE FUNCTION audit.log_logout(
    p_session_id UUID
)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE audit.login_history
    SET logout_at = NOW()
    WHERE session_id = p_session_id AND logout_at IS NULL;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- 查询审计日志
CREATE OR REPLACE FUNCTION audit.query_audit_logs(
    p_tenant_id UUID,
    p_start_date TIMESTAMPTZ DEFAULT NULL,
    p_end_date TIMESTAMPTZ DEFAULT NULL,
    p_user_id UUID DEFAULT NULL,
    p_action VARCHAR(100) DEFAULT NULL,
    p_entity_type VARCHAR(100) DEFAULT NULL,
    p_entity_id UUID DEFAULT NULL,
    p_limit INTEGER DEFAULT 100,
    p_offset INTEGER DEFAULT 0
)
RETURNS TABLE (
    id UUID,
    action VARCHAR(100),
    entity_type VARCHAR(100),
    entity_id UUID,
    user_name VARCHAR(255),
    created_at TIMESTAMPTZ,
    success BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        al.id, al.action, al.entity_type, al.entity_id,
        al.user_name, al.created_at, al.success
    FROM audit.audit_logs al
    WHERE al.tenant_id = p_tenant_id
      AND (p_start_date IS NULL OR al.created_at >= p_start_date)
      AND (p_end_date IS NULL OR al.created_at <= p_end_date)
      AND (p_user_id IS NULL OR al.user_id = p_user_id)
      AND (p_action IS NULL OR al.action = p_action)
      AND (p_entity_type IS NULL OR al.entity_type = p_entity_type)
      AND (p_entity_id IS NULL OR al.entity_id = p_entity_id)
    ORDER BY al.created_at DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- 归档旧审计日志
CREATE OR REPLACE FUNCTION audit.archive_old_logs(
    p_older_than_days INTEGER DEFAULT 90,
    p_batch_size INTEGER DEFAULT 10000
)
RETURNS INTEGER AS $$
DECLARE
    v_archived INTEGER := 0;
    v_batch INTEGER;
BEGIN
    LOOP
        WITH archived AS (
            DELETE FROM audit.audit_logs
            WHERE created_at < NOW() - (p_older_than_days || ' days')::INTERVAL
            AND id IN (
                SELECT id FROM audit.audit_logs
                WHERE created_at < NOW() - (p_older_than_days || ' days')::INTERVAL
                LIMIT p_batch_size
            )
            RETURNING *, 'AGE'::VARCHAR(100) as archive_reason
        )
        INSERT INTO audit.audit_logs_archive
        SELECT * FROM archived;
        
        GET DIAGNOSTICS v_batch = ROW_COUNT;
        v_archived := v_archived + v_batch;
        
        EXIT WHEN v_batch < p_batch_size;
    END LOOP;
    
    RETURN v_archived;
END;
$$ LANGUAGE plpgsql;

-- 清理已归档的审计日志
CREATE OR REPLACE FUNCTION audit.purge_archived_logs(
    p_older_than_days INTEGER DEFAULT 365
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM audit.audit_logs_archive
    WHERE archived_at < NOW() - (p_older_than_days || ' days')::INTERVAL;
    
    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 获取登录统计
CREATE OR REPLACE FUNCTION audit.get_login_stats(
    p_tenant_id UUID,
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS TABLE (
    total_logins BIGINT,
    successful_logins BIGINT,
    failed_logins BIGINT,
    unique_users BIGINT,
    avg_logins_per_user NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::BIGINT as total_logins,
        COUNT(*) FILTER (WHERE success = TRUE)::BIGINT as successful_logins,
        COUNT(*) FILTER (WHERE success = FALSE)::BIGINT as failed_logins,
        COUNT(DISTINCT user_id)::BIGINT as unique_users,
        ROUND(COUNT(*)::NUMERIC / NULLIF(COUNT(DISTINCT user_id), 0), 2) as avg_logins_per_user
    FROM audit.login_history
    WHERE tenant_id = p_tenant_id
      AND created_at BETWEEN p_start_date AND p_end_date;
END;
$$ LANGUAGE plpgsql;

-- 获取用户活动报告
CREATE OR REPLACE FUNCTION audit.get_user_activity_report(
    p_tenant_id UUID,
    p_user_id UUID,
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS TABLE (
    action VARCHAR(100),
    action_count BIGINT,
    first_action TIMESTAMPTZ,
    last_action TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        al.action::VARCHAR(100),
        COUNT(*)::BIGINT as action_count,
        MIN(al.created_at) as first_action,
        MAX(al.created_at) as last_action
    FROM audit.audit_logs al
    WHERE al.tenant_id = p_tenant_id
      AND al.user_id = p_user_id
      AND al.created_at BETWEEN p_start_date AND p_end_date
    GROUP BY al.action
    ORDER BY action_count DESC;
END;
$$ LANGUAGE plpgsql;

-- 创建分区（按月）
CREATE OR REPLACE FUNCTION audit.create_monthly_partition(
    p_year INTEGER,
    p_month INTEGER
)
RETURNS TEXT AS $$
DECLARE
    v_partition_name TEXT;
    v_start_date DATE;
    v_end_date DATE;
BEGIN
    v_partition_name := format('audit_logs_%s_%s', p_year, LPAD(p_month::TEXT, 2, '0'));
    v_start_date := make_date(p_year, p_month, 1);
    v_end_date := v_start_date + INTERVAL '1 month';
    
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS audit.%I PARTITION OF audit.audit_logs
         FOR VALUES FROM (%L) TO (%L)',
        v_partition_name, v_start_date, v_end_date
    );
    
    RETURN v_partition_name;
END;
$$ LANGUAGE plpgsql;

-- 删除旧分区
CREATE OR REPLACE FUNCTION audit.drop_old_partition(
    p_older_than_months INTEGER DEFAULT 12
)
RETURNS INTEGER AS $$
DECLARE
    v_partition RECORD;
    v_dropped INTEGER := 0;
BEGIN
    FOR v_partition IN
        SELECT inhrelid::regclass::text as partition_name
        FROM pg_inherits
        WHERE inhparent = 'audit.audit_logs'::regclass
    LOOP
        -- 检查分区是否过期
        IF v_partition.partition_name ~ 'audit_logs_(\d{4})_(\d{2})' THEN
            v_dropped := v_dropped + 1;
            EXECUTE format('DROP TABLE IF EXISTS %I', v_partition.partition_name);
        END IF;
    END LOOP;
    
    RETURN v_dropped;
END;
$$ LANGUAGE plpgsql;
```


---

## 4. 视图设计

### 4.1 项目视图

```sql
-- ============================================
-- 项目视图
-- ============================================

-- 项目详情视图
CREATE OR REPLACE VIEW core.v_project_details AS
SELECT 
    p.id,
    p.tenant_id,
    t.name as tenant_name,
    p.name,
    p.description,
    p.project_code,
    p.status,
    p.project_type,
    p.visibility,
    p.thumbnail_url,
    p.tags,
    p.location,
    p.area_total_sqm,
    p.budget_currency,
    p.budget_amount,
    p.start_date,
    p.target_end_date,
    p.actual_end_date,
    p.progress_percent,
    p.settings,
    p.custom_fields,
    p.metadata,
    p.version_count,
    p.current_version_id,
    p.created_at,
    p.updated_at,
    p.created_by,
    u1.first_name || ' ' || u1.last_name as created_by_name,
    p.updated_by,
    u2.first_name || ' ' || u2.last_name as updated_by_name,
    -- 成员数量
    (SELECT COUNT(*) FROM core.project_members pm WHERE pm.project_id = p.id) as member_count,
    -- 设计数量
    (SELECT COUNT(*) FROM core.designs d WHERE d.project_id = p.id AND d.deleted_at IS NULL) as design_count,
    -- 最近活动时间
    (SELECT MAX(created_at) FROM audit.audit_logs al 
     WHERE al.entity_type = 'projects' AND al.entity_id = p.id) as last_activity_at
FROM core.projects p
LEFT JOIN core.tenants t ON p.tenant_id = t.id
LEFT JOIN core.users u1 ON p.created_by = u1.id
LEFT JOIN core.users u2 ON p.updated_by = u2.id
WHERE p.deleted_at IS NULL;

-- 项目成员视图
CREATE OR REPLACE VIEW core.v_project_members AS
SELECT 
    pm.id,
    pm.project_id,
    p.name as project_name,
    pm.user_id,
    u.username,
    u.email,
    u.first_name || ' ' || u.last_name as full_name,
    u.avatar_url,
    pm.role,
    pm.permissions,
    pm.joined_at,
    pm.joined_by,
    u2.first_name || ' ' || u2.last_name as joined_by_name,
    -- 用户状态
    u.status as user_status,
    -- 最后登录
    u.last_login_at
FROM core.project_members pm
JOIN core.projects p ON pm.project_id = p.id
JOIN core.users u ON pm.user_id = u.id
LEFT JOIN core.users u2 ON pm.joined_by = u2.id
WHERE p.deleted_at IS NULL AND u.deleted_at IS NULL;

-- 项目统计视图
CREATE OR REPLACE VIEW analytics.v_project_statistics AS
SELECT 
    p.id as project_id,
    p.tenant_id,
    p.name as project_name,
    p.status,
    p.project_type,
    -- 设计统计
    COUNT(DISTINCT d.id) FILTER (WHERE d.deleted_at IS NULL) as total_designs,
    COUNT(DISTINCT d.id) FILTER (WHERE d.status = 'approved' AND d.deleted_at IS NULL) as approved_designs,
    COUNT(DISTINCT d.id) FILTER (WHERE d.status = 'draft' AND d.deleted_at IS NULL) as draft_designs,
    -- 元素统计
    COALESCE(SUM(d.element_count), 0) as total_elements,
    -- 版本统计
    COALESCE(SUM(d.version_count), 0) as total_versions,
    -- 成员统计
    COUNT(DISTINCT pm.user_id) as total_members,
    -- 活动统计
    COUNT(DISTINCT al.id) as total_activities,
    COUNT(DISTINCT al.id) FILTER (WHERE al.created_at > NOW() - INTERVAL '7 days') as activities_last_7_days,
    -- 时间统计
    p.created_at,
    EXTRACT(DAY FROM (NOW() - p.created_at)) as project_age_days,
    CASE 
        WHEN p.actual_end_date IS NOT NULL THEN EXTRACT(DAY FROM (p.actual_end_date - p.created_at))
        ELSE NULL 
    END as project_duration_days
FROM core.projects p
LEFT JOIN core.designs d ON p.id = d.project_id
LEFT JOIN core.project_members pm ON p.id = pm.project_id
LEFT JOIN audit.audit_logs al ON p.id = al.entity_id AND al.entity_type = 'projects'
WHERE p.deleted_at IS NULL
GROUP BY p.id, p.tenant_id, p.name, p.status, p.project_type, p.created_at, p.actual_end_date;

-- 项目时间线视图
CREATE OR REPLACE VIEW core.v_project_timeline AS
SELECT 
    p.id as project_id,
    p.name as project_name,
    'created'::VARCHAR(50) as event_type,
    p.created_at as event_date,
    'Project created' as event_description,
    p.created_by as user_id,
    u.first_name || ' ' || u.last_name as user_name
FROM core.projects p
LEFT JOIN core.users u ON p.created_by = u.id
WHERE p.deleted_at IS NULL

UNION ALL

SELECT 
    p.id as project_id,
    p.name as project_name,
    'design_added'::VARCHAR(50) as event_type,
    d.created_at as event_date,
    'Design "' || d.name || '" added' as event_description,
    d.created_by as user_id,
    u.first_name || ' ' || u.last_name as user_name
FROM core.projects p
JOIN core.designs d ON p.id = d.project_id
LEFT JOIN core.users u ON d.created_by = u.id
WHERE p.deleted_at IS NULL AND d.deleted_at IS NULL

UNION ALL

SELECT 
    p.id as project_id,
    p.name as project_name,
    'member_joined'::VARCHAR(50) as event_type,
    pm.joined_at as event_date,
    u2.first_name || ' ' || u2.last_name || ' joined as ' || pm.role as event_description,
    pm.user_id,
    u.first_name || ' ' || u.last_name as user_name
FROM core.projects p
JOIN core.project_members pm ON p.id = pm.project_id
JOIN core.users u ON pm.user_id = u.id
LEFT JOIN core.users u2 ON pm.joined_by = u2.id
WHERE p.deleted_at IS NULL AND u.deleted_at IS NULL

ORDER BY event_date DESC;
```

### 4.2 版本历史视图

```sql
-- ============================================
-- 版本历史视图
-- ============================================

-- 设计版本详情视图
CREATE OR REPLACE VIEW core.v_design_version_details AS
SELECT 
    dv.id,
    dv.design_id,
    d.name as design_name,
    d.design_type,
    dv.project_id,
    p.name as project_name,
    dv.tenant_id,
    dv.version_number,
    dv.version_name,
    dv.description,
    dv.change_summary,
    dv.snapshot_id,
    dv.file_path,
    dv.file_size_bytes,
    pg_size_pretty(dv.file_size_bytes) as file_size_readable,
    dv.file_hash,
    dv.element_count,
    dv.is_major_version,
    dv.is_published,
    dv.published_at,
    dv.published_by,
    pub.first_name || ' ' || pub.last_name as published_by_name,
    dv.parent_version_id,
    pv.version_number as parent_version_number,
    dv.merge_source_id,
    mv.version_number as merge_source_version_number,
    dv.metadata,
    dv.created_at,
    dv.created_by,
    creator.first_name || ' ' || creator.last_name as created_by_name,
    -- 版本比较信息
    CASE 
        WHEN dv.parent_version_id IS NOT NULL THEN
            (SELECT COUNT(*) FROM versioning.events e
             WHERE e.aggregate_type = 'designs'
               AND e.aggregate_id = dv.design_id
               AND e.sequence_number > (
                   SELECT sequence_number FROM versioning.snapshots s
                   WHERE s.aggregate_type = 'designs' AND s.aggregate_id = dv.design_id
                   ORDER BY version DESC LIMIT 1
               ))
        ELSE dv.element_count
    END as changes_since_parent
FROM core.design_versions dv
JOIN core.designs d ON dv.design_id = d.id
JOIN core.projects p ON dv.project_id = p.id
LEFT JOIN core.design_versions pv ON dv.parent_version_id = pv.id
LEFT JOIN core.design_versions mv ON dv.merge_source_id = mv.id
LEFT JOIN core.users pub ON dv.published_by = pub.id
LEFT JOIN core.users creator ON dv.created_by = creator.id;

-- 版本对比视图
CREATE OR REPLACE VIEW core.v_version_comparison AS
SELECT 
    v1.id as version1_id,
    v1.version_number as version1_number,
    v2.id as version2_id,
    v2.version_number as version2_number,
    v1.design_id,
    d.name as design_name,
    -- 元素变化统计
    (SELECT COUNT(*) FROM versioning.events e
     WHERE e.aggregate_type = 'elements'
       AND e.aggregate_id IN (SELECT id FROM core.elements WHERE design_id = v1.design_id)
       AND e.occurred_at BETWEEN v1.created_at AND v2.created_at
       AND e.event_type = 'ELEMENT_CREATED') as elements_added,
    (SELECT COUNT(*) FROM versioning.events e
     WHERE e.aggregate_type = 'elements'
       AND e.aggregate_id IN (SELECT id FROM core.elements WHERE design_id = v1.design_id)
       AND e.occurred_at BETWEEN v1.created_at AND v2.created_at
       AND e.event_type = 'ELEMENT_DELETED') as elements_removed,
    (SELECT COUNT(*) FROM versioning.events e
     WHERE e.aggregate_type = 'elements'
       AND e.aggregate_id IN (SELECT id FROM core.elements WHERE design_id = v1.design_id)
       AND e.occurred_at BETWEEN v1.created_at AND v2.created_at
       AND e.event_type = 'ELEMENT_UPDATED') as elements_modified,
    -- 文件大小变化
    v2.file_size_bytes - v1.file_size_bytes as size_change_bytes,
    pg_size_pretty(ABS(v2.file_size_bytes - v1.file_size_bytes)) as size_change_readable,
    -- 时间差
    EXTRACT(EPOCH FROM (v2.created_at - v1.created_at)) / 3600 as hours_between
FROM core.design_versions v1
JOIN core.design_versions v2 ON v1.design_id = v2.design_id AND v2.version_number > v1.version_number
JOIN core.designs d ON v1.design_id = d.id;

-- 事件流视图
CREATE OR REPLACE VIEW versioning.v_event_stream AS
SELECT 
    e.id,
    e.aggregate_type,
    e.aggregate_id,
    e.tenant_id,
    t.name as tenant_name,
    e.event_type,
    e.event_version,
    e.payload,
    e.metadata,
    e.sequence_number,
    e.global_sequence,
    e.correlation_id,
    e.causation_id,
    e.occurred_at,
    e.recorded_at,
    e.user_id,
    u.first_name || ' ' || u.last_name as user_name,
    e.source_ip,
    e.source_service,
    -- 事件分类
    CASE 
        WHEN e.event_type LIKE '%CREATE%' THEN 'CREATE'
        WHEN e.event_type LIKE '%UPDATE%' THEN 'UPDATE'
        WHEN e.event_type LIKE '%DELETE%' THEN 'DELETE'
        ELSE 'OTHER'
    END as event_category,
    -- payload摘要
    LEFT(e.payload::TEXT, 200) as payload_summary
FROM versioning.events e
LEFT JOIN core.tenants t ON e.tenant_id = t.id
LEFT JOIN core.users u ON e.user_id = u.id;

-- 快照状态视图
CREATE OR REPLACE VIEW versioning.v_snapshot_states AS
SELECT 
    s.id,
    s.aggregate_type,
    s.aggregate_id,
    s.tenant_id,
    s.version,
    s.sequence_number,
    s.state,
    s.event_count,
    pg_size_pretty(s.state_size_bytes) as state_size_readable,
    s.state_size_bytes,
    s.created_at,
    s.expires_at,
    s.created_by,
    u.first_name || ' ' || u.last_name as created_by_name,
    -- 事件重放时间估算（假设每个事件0.1ms）
    (s.event_count * 0.1)::INTEGER as estimated_replay_ms,
    -- 是否需要新快照
    CASE WHEN s.event_count > 100 THEN TRUE ELSE FALSE END as needs_new_snapshot
FROM versioning.snapshots s
LEFT JOIN core.users u ON s.created_by = u.id;

-- 操作历史视图
CREATE OR REPLACE VIEW versioning.v_operation_history AS
SELECT 
    oh.id,
    oh.tenant_id,
    t.name as tenant_name,
    oh.project_id,
    p.name as project_name,
    oh.design_id,
    d.name as design_name,
    oh.user_id,
    u.first_name || ' ' || u.last_name as user_name,
    u.avatar_url as user_avatar,
    oh.operation_type,
    oh.operation_name,
    oh.description,
    oh.before_state,
    oh.after_state,
    oh.affected_elements,
    array_length(oh.affected_elements, 1) as affected_element_count,
    oh.can_undo,
    oh.undone_at,
    oh.undone_by,
    undo_user.first_name || ' ' || undo_user.last_name as undone_by_name,
    oh.redo_of,
    oh.created_at,
    oh.session_id,
    -- 状态
    CASE 
        WHEN oh.undone_at IS NOT NULL THEN 'undone'
        WHEN oh.redo_of IS NOT NULL THEN 'redo'
        ELSE 'active'
    END as operation_status
FROM versioning.operation_history oh
LEFT JOIN core.tenants t ON oh.tenant_id = t.id
LEFT JOIN core.projects p ON oh.project_id = p.id
LEFT JOIN core.designs d ON oh.design_id = d.id
LEFT JOIN core.users u ON oh.user_id = u.id
LEFT JOIN core.users undo_user ON oh.undone_by = undo_user.id;
```

### 4.3 审计追踪视图

```sql
-- ============================================
-- 审计追踪视图
-- ============================================

-- 审计日志详情视图
CREATE OR REPLACE VIEW audit.v_audit_log_details AS
SELECT 
    al.id,
    al.tenant_id,
    t.name as tenant_name,
    al.action,
    al.entity_type,
    al.entity_id,
    -- 实体名称（动态查询）
    CASE al.entity_type
        WHEN 'projects' THEN (SELECT name FROM core.projects WHERE id = al.entity_id)
        WHEN 'designs' THEN (SELECT name FROM core.designs WHERE id = al.entity_id)
        WHEN 'elements' THEN (SELECT name FROM core.elements WHERE id = al.entity_id)
        WHEN 'users' THEN (SELECT username FROM core.users WHERE id = al.entity_id)
        ELSE NULL
    END as entity_name,
    al.before_data,
    al.after_data,
    al.changed_fields,
    array_length(al.changed_fields, 1) as change_count,
    al.user_id,
    al.user_name,
    al.user_email,
    al.request_id,
    al.session_id,
    al.correlation_id,
    al.source_ip,
    al.user_agent,
    al.source_service,
    al.api_endpoint,
    al.http_method,
    al.success,
    al.error_code,
    al.error_message,
    al.created_at,
    -- 时间格式化
    TO_CHAR(al.created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at_formatted,
    -- 相对时间
    CASE 
        WHEN al.created_at > NOW() - INTERVAL '1 minute' THEN 'Just now'
        WHEN al.created_at > NOW() - INTERVAL '1 hour' THEN 
            EXTRACT(MINUTE FROM (NOW() - al.created_at))::TEXT || ' minutes ago'
        WHEN al.created_at > NOW() - INTERVAL '1 day' THEN 
            EXTRACT(HOUR FROM (NOW() - al.created_at))::TEXT || ' hours ago'
        ELSE TO_CHAR(al.created_at, 'YYYY-MM-DD')
    END as relative_time
FROM audit.audit_logs al
LEFT JOIN core.tenants t ON al.tenant_id = t.id;

-- 登录历史详情视图
CREATE OR REPLACE VIEW audit.v_login_history_details AS
SELECT 
    lh.id,
    lh.tenant_id,
    t.name as tenant_name,
    lh.user_id,
    u.username,
    u.email,
    u.first_name || ' ' || u.last_name as full_name,
    lh.login_type,
    lh.success,
    lh.failure_reason,
    lh.session_id,
    lh.token_id,
    lh.ip_address,
    lh.user_agent,
    lh.device_fingerprint,
    lh.geo_location,
    lh.created_at as login_at,
    lh.logout_at,
    -- 会话时长
    CASE 
        WHEN lh.logout_at IS NOT NULL THEN 
            EXTRACT(EPOCH FROM (lh.logout_at - lh.created_at)) / 60
        ELSE NULL
    END as session_duration_minutes,
    -- 状态
    CASE 
        WHEN lh.logout_at IS NOT NULL THEN 'logged_out'
        WHEN lh.success = FALSE THEN 'failed'
        ELSE 'active'
    END as session_status,
    -- 风险评分（简单规则）
    CASE 
        WHEN lh.success = FALSE THEN 50
        WHEN lh.geo_location->>'country' != (
            SELECT geo_location->>'country' 
            FROM audit.login_history 
            WHERE user_id = lh.user_id AND success = TRUE 
            ORDER BY created_at DESC LIMIT 1 OFFSET 1
        ) THEN 30
        ELSE 0
    END as risk_score
FROM audit.login_history lh
LEFT JOIN core.tenants t ON lh.tenant_id = t.id
LEFT JOIN core.users u ON lh.user_id = u.id;

-- 安全审计视图（异常检测）
CREATE OR REPLACE VIEW audit.v_security_audit AS
SELECT 
    'failed_login'::VARCHAR(50) as alert_type,
    lh.ip_address as source,
    COUNT(*) as event_count,
    MIN(lh.created_at) as first_seen,
    MAX(lh.created_at) as last_seen,
    'Multiple failed login attempts' as description,
    80 as severity  -- 0-100
FROM audit.login_history lh
WHERE lh.success = FALSE
  AND lh.created_at > NOW() - INTERVAL '1 hour'
GROUP BY lh.ip_address
HAVING COUNT(*) >= 5

UNION ALL

SELECT 
    'unusual_access_time'::VARCHAR(50) as alert_type,
    lh.user_id::TEXT as source,
    COUNT(*) as event_count,
    MIN(lh.created_at) as first_seen,
    MAX(lh.created_at) as last_seen,
    'Login outside normal hours' as description,
    50 as severity
FROM audit.login_history lh
WHERE lh.success = TRUE
  AND lh.created_at > NOW() - INTERVAL '24 hours'
  AND EXTRACT(HOUR FROM lh.created_at) NOT BETWEEN 6 AND 22
GROUP BY lh.user_id

UNION ALL

SELECT 
    'permission_change'::VARCHAR(50) as alert_type,
    al.user_id::TEXT as source,
    COUNT(*) as event_count,
    MIN(al.created_at) as first_seen,
    MAX(al.created_at) as last_seen,
    'Multiple permission changes' as description,
    60 as severity
FROM audit.audit_logs al
WHERE al.action = 'PERMISSION_CHANGE'
  AND al.created_at > NOW() - INTERVAL '1 hour'
GROUP BY al.user_id
HAVING COUNT(*) >= 3;

-- 数据访问审计视图
CREATE OR REPLACE VIEW audit.v_data_access_audit AS
SELECT 
    dal.id,
    dal.tenant_id,
    t.name as tenant_name,
    dal.user_id,
    u.username,
    u.first_name || ' ' || u.last_name as full_name,
    dal.access_type,
    dal.resource_type,
    dal.resource_id,
    -- 资源名称
    CASE dal.resource_type
        WHEN 'projects' THEN (SELECT name FROM core.projects WHERE id = dal.resource_id)
        WHEN 'designs' THEN (SELECT name FROM core.designs WHERE id = dal.resource_id)
        ELSE NULL
    END as resource_name,
    dal.query_params,
    dal.result_count,
    dal.duration_ms,
    dal.created_at,
    -- 性能评级
    CASE 
        WHEN dal.duration_ms < 100 THEN 'excellent'
        WHEN dal.duration_ms < 500 THEN 'good'
        WHEN dal.duration_ms < 1000 THEN 'fair'
        ELSE 'poor'
    END as performance_rating
FROM audit.data_access_log dal
LEFT JOIN core.tenants t ON dal.tenant_id = t.id
LEFT JOIN core.users u ON dal.user_id = u.id;

-- 合规性报告视图
CREATE OR REPLACE VIEW audit.v_compliance_report AS
SELECT 
    al.tenant_id,
    t.name as tenant_name,
    DATE(al.created_at) as activity_date,
    COUNT(*) as total_activities,
    COUNT(*) FILTER (WHERE al.action = 'CREATE') as creations,
    COUNT(*) FILTER (WHERE al.action = 'UPDATE') as updates,
    COUNT(*) FILTER (WHERE al.action = 'DELETE') as deletions,
    COUNT(*) FILTER (WHERE al.action = 'EXPORT') as exports,
    COUNT(*) FILTER (WHERE al.action = 'SHARE') as shares,
    COUNT(DISTINCT al.user_id) as unique_users,
    COUNT(*) FILTER (WHERE al.success = FALSE) as failed_actions,
    -- 数据导出总量估算
    SUM(CASE WHEN al.action = 'EXPORT' THEN pg_column_size(al.after_data) ELSE 0 END) as estimated_export_bytes
FROM audit.audit_logs al
LEFT JOIN core.tenants t ON al.tenant_id = t.id
GROUP BY al.tenant_id, t.name, DATE(al.created_at)
ORDER BY activity_date DESC;
```

### 4.4 统计报表视图

```sql
-- ============================================
-- 统计报表视图
-- ============================================

-- 租户使用统计视图
CREATE OR REPLACE VIEW analytics.v_tenant_usage_stats AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    t.plan_type,
    t.status,
    -- 用户统计
    COUNT(DISTINCT u.id) FILTER (WHERE u.deleted_at IS NULL) as total_users,
    COUNT(DISTINCT u.id) FILTER (WHERE u.status = 'active' AND u.deleted_at IS NULL) as active_users,
    COUNT(DISTINCT u.id) FILTER (WHERE u.last_login_at > NOW() - INTERVAL '7 days') as users_active_last_7_days,
    -- 项目统计
    COUNT(DISTINCT p.id) FILTER (WHERE p.deleted_at IS NULL) as total_projects,
    COUNT(DISTINCT p.id) FILTER (WHERE p.status = 'in_progress' AND p.deleted_at IS NULL) as active_projects,
    -- 设计统计
    COUNT(DISTINCT d.id) FILTER (WHERE d.deleted_at IS NULL) as total_designs,
    -- 元素统计
    COALESCE(SUM(d.element_count), 0) as total_elements,
    -- 存储统计
    t.storage_used_bytes,
    pg_size_pretty(t.storage_used_bytes) as storage_used_readable,
    t.max_storage_gb * 1024 * 1024 * 1024 as max_storage_bytes,
    ROUND(t.storage_used_bytes::NUMERIC / (t.max_storage_gb * 1024 * 1024 * 1024) * 100, 2) as storage_usage_percent,
    -- 活动统计
    COUNT(DISTINCT al.id) as total_activities,
    COUNT(DISTINCT al.id) FILTER (WHERE al.created_at > NOW() - INTERVAL '30 days') as activities_last_30_days,
    -- 时间统计
    t.created_at as tenant_created_at,
    EXTRACT(DAY FROM (NOW() - t.created_at)) as tenant_age_days
FROM core.tenants t
LEFT JOIN core.users u ON t.id = u.tenant_id
LEFT JOIN core.projects p ON t.id = p.tenant_id
LEFT JOIN core.designs d ON p.id = d.project_id
LEFT JOIN audit.audit_logs al ON t.id = al.tenant_id
GROUP BY t.id, t.name, t.plan_type, t.status, t.storage_used_bytes, t.max_storage_gb, t.created_at;

-- 用户活动统计视图
CREATE OR REPLACE VIEW analytics.v_user_activity_stats AS
SELECT 
    u.id as user_id,
    u.tenant_id,
    t.name as tenant_name,
    u.username,
    u.email,
    u.first_name || ' ' || u.last_name as full_name,
    u.role,
    u.status,
    -- 项目参与
    COUNT(DISTINCT pm.project_id) as projects_count,
    COUNT(DISTINCT pm.project_id) FILTER (WHERE pm.role = 'owner') as owned_projects,
    -- 设计贡献
    COUNT(DISTINCT d.id) FILTER (WHERE d.created_by = u.id AND d.deleted_at IS NULL) as designs_created,
    COUNT(DISTINCT d.id) FILTER (WHERE d.updated_by = u.id AND d.deleted_at IS NULL) as designs_updated,
    -- 元素操作
    COUNT(DISTINCT e.id) FILTER (WHERE e.created_by = u.id AND e.deleted_at IS NULL) as elements_created,
    -- 活动统计
    COUNT(DISTINCT al.id) as total_activities,
    COUNT(DISTINCT al.id) FILTER (WHERE al.created_at > NOW() - INTERVAL '7 days') as activities_last_7_days,
    COUNT(DISTINCT al.id) FILTER (WHERE al.created_at > NOW() - INTERVAL '30 days') as activities_last_30_days,
    -- 登录统计
    COUNT(DISTINCT lh.id) as total_logins,
    MAX(lh.created_at) as last_login_at,
    -- 会话统计
    AVG(EXTRACT(EPOCH FROM (lh.logout_at - lh.created_at)) / 60) 
        FILTER (WHERE lh.logout_at IS NOT NULL) as avg_session_duration_minutes,
    -- 时间统计
    u.created_at as user_created_at,
    u.last_login_at,
    EXTRACT(DAY FROM (NOW() - u.created_at)) as user_age_days,
    CASE 
        WHEN u.last_login_at > NOW() - INTERVAL '7 days' THEN 'active'
        WHEN u.last_login_at > NOW() - INTERVAL '30 days' THEN 'moderate'
        ELSE 'inactive'
    END as activity_level
FROM core.users u
LEFT JOIN core.tenants t ON u.tenant_id = t.id
LEFT JOIN core.project_members pm ON u.id = pm.user_id
LEFT JOIN core.designs d ON u.id = d.created_by OR u.id = d.updated_by
LEFT JOIN core.elements e ON u.id = e.created_by
LEFT JOIN audit.audit_logs al ON u.id = al.user_id
LEFT JOIN audit.login_history lh ON u.id = lh.user_id
WHERE u.deleted_at IS NULL
GROUP BY u.id, u.tenant_id, t.name, u.username, u.email, u.first_name, u.last_name, 
         u.role, u.status, u.created_at, u.last_login_at;

-- 设计类型分布视图
CREATE OR REPLACE VIEW analytics.v_design_type_distribution AS
SELECT 
    d.design_type,
    COUNT(*) as design_count,
    ROUND(COUNT(*)::NUMERIC / NULLIF(SUM(COUNT(*)) OVER (), 0) * 100, 2) as percentage,
    AVG(d.element_count)::INTEGER as avg_elements,
    AVG(d.version_count)::INTEGER as avg_versions,
    pg_size_pretty(AVG(d.file_size_bytes)::BIGINT) as avg_file_size,
    SUM(d.file_size_bytes) as total_size_bytes,
    pg_size_pretty(SUM(d.file_size_bytes)) as total_size_readable
FROM core.designs d
WHERE d.deleted_at IS NULL
GROUP BY d.design_type
ORDER BY design_count DESC;

-- 元素类型分布视图
CREATE OR REPLACE VIEW analytics.v_element_type_distribution AS
SELECT 
    e.element_type,
    COUNT(*) as element_count,
    ROUND(COUNT(*)::NUMERIC / NULLIF(SUM(COUNT(*)) OVER (), 0) * 100, 2) as percentage,
    COUNT(DISTINCT e.design_id) as designs_using,
    COUNT(DISTINCT e.project_id) as projects_using,
    AVG(g.area)::DECIMAL(18, 2) as avg_area,
    AVG(g.length)::DECIMAL(18, 2) as avg_length
FROM core.elements e
LEFT JOIN geometry.geometries g ON e.id = g.element_id
WHERE e.deleted_at IS NULL
GROUP BY e.element_type
ORDER BY element_count DESC;

-- 时间趋势分析视图
CREATE OR REPLACE VIEW analytics.v_time_trends AS
SELECT 
    DATE_TRUNC('day', created_at)::DATE as activity_date,
    COUNT(DISTINCT id) FILTER (WHERE entity_type = 'projects') as projects_created,
    COUNT(DISTINCT id) FILTER (WHERE entity_type = 'designs') as designs_created,
    COUNT(DISTINCT id) FILTER (WHERE entity_type = 'elements') as elements_created,
    COUNT(DISTINCT user_id) as unique_active_users,
    COUNT(*) as total_activities
FROM audit.audit_logs
WHERE action = 'CREATE'
  AND created_at > NOW() - INTERVAL '90 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY activity_date DESC;

-- 性能指标视图
CREATE OR REPLACE VIEW analytics.v_performance_metrics AS
SELECT 
    'query_response_time'::VARCHAR(50) as metric_name,
    'ms'::VARCHAR(20) as unit,
    AVG(duration_ms)::DECIMAL(10, 2) as avg_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) as p50_value,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_value,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99_value,
    MAX(duration_ms) as max_value,
    COUNT(*) as sample_count
FROM audit.data_access_log
WHERE created_at > NOW() - INTERVAL '24 hours'

UNION ALL

SELECT 
    'login_response_time'::VARCHAR(50) as metric_name,
    'ms'::VARCHAR(20) as unit,
    AVG(duration_ms)::DECIMAL(10, 2) as avg_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms) as p50_value,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_value,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99_value,
    MAX(duration_ms) as max_value,
    COUNT(*) as sample_count
FROM audit.login_history
WHERE created_at > NOW() - INTERVAL '24 hours';

-- 存储使用趋势视图
CREATE OR REPLACE VIEW analytics.v_storage_trends AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    DATE(dv.created_at) as date,
    SUM(dv.file_size_bytes) as daily_storage_bytes,
    pg_size_pretty(SUM(dv.file_size_bytes)) as daily_storage_readable,
    SUM(SUM(dv.file_size_bytes)) OVER (
        PARTITION BY t.id 
        ORDER BY DATE(dv.created_at) 
        ROWS UNBOUNDED PRECEDING
    ) as cumulative_storage_bytes
FROM core.tenants t
JOIN core.projects p ON t.id = p.tenant_id
JOIN core.designs d ON p.id = d.project_id
JOIN core.design_versions dv ON d.id = dv.design_id
WHERE dv.created_at > NOW() - INTERVAL '90 days'
GROUP BY t.id, t.name, DATE(dv.created_at)
ORDER BY date DESC;

-- 实时仪表板视图
CREATE OR REPLACE VIEW analytics.v_dashboard_summary AS
SELECT 
    -- 总体统计
    (SELECT COUNT(*) FROM core.tenants WHERE status = 'active') as active_tenants,
    (SELECT COUNT(*) FROM core.users WHERE status = 'active' AND deleted_at IS NULL) as active_users,
    (SELECT COUNT(*) FROM core.projects WHERE status IN ('in_progress', 'draft') AND deleted_at IS NULL) as active_projects,
    (SELECT COUNT(*) FROM core.designs WHERE deleted_at IS NULL) as total_designs,
    (SELECT COALESCE(SUM(element_count), 0) FROM core.designs WHERE deleted_at IS NULL) as total_elements,
    -- 今日统计
    (SELECT COUNT(*) FROM audit.audit_logs WHERE created_at > NOW() - INTERVAL '24 hours') as activities_today,
    (SELECT COUNT(DISTINCT user_id) FROM audit.login_history WHERE created_at > NOW() - INTERVAL '24 hours' AND success = TRUE) as users_logged_in_today,
    (SELECT COUNT(*) FROM core.projects WHERE created_at > NOW() - INTERVAL '24 hours') as projects_created_today,
    (SELECT COUNT(*) FROM core.designs WHERE created_at > NOW() - INTERVAL '24 hours' AND deleted_at IS NULL) as designs_created_today,
    -- 本周统计
    (SELECT COUNT(*) FROM audit.audit_logs WHERE created_at > NOW() - INTERVAL '7 days') as activities_this_week,
    (SELECT COUNT(*) FROM core.projects WHERE created_at > NOW() - INTERVAL '7 days') as projects_created_this_week,
    -- 系统健康
    (SELECT COUNT(*) FROM core.users WHERE failed_login_attempts >= 5) as locked_accounts,
    (SELECT COUNT(*) FROM audit.login_history WHERE success = FALSE AND created_at > NOW() - INTERVAL '1 hour') as failed_logins_last_hour,
    -- 更新时间
    NOW() as last_updated;
```

---

## 5. 数据迁移脚本

### 5.1 初始数据脚本

```sql
-- ============================================
-- 初始数据脚本
-- ============================================

-- 插入系统权限
INSERT INTO core.permissions (code, name, description, resource_type, action, is_system) VALUES
-- 系统级权限
('system:admin', 'System Administration', 'Full system access', 'system', 'admin', TRUE),
('system:monitor', 'System Monitoring', 'View system metrics', 'system', 'read', TRUE),
-- 租户级权限
('tenant:manage', 'Manage Tenant', 'Full tenant management', 'tenant', 'manage', TRUE),
('tenant:read', 'View Tenant', 'View tenant information', 'tenant', 'read', TRUE),
-- 项目级权限
('project:create', 'Create Project', 'Create new projects', 'project', 'create', FALSE),
('project:read', 'View Project', 'View project details', 'project', 'read', FALSE),
('project:update', 'Update Project', 'Update project information', 'project', 'update', FALSE),
('project:delete', 'Delete Project', 'Delete projects', 'project', 'delete', FALSE),
('project:manage', 'Manage Project', 'Full project management', 'project', 'manage', FALSE),
('project:share', 'Share Project', 'Share project with others', 'project', 'share', FALSE),
-- 设计级权限
('design:create', 'Create Design', 'Create new designs', 'design', 'create', FALSE),
('design:read', 'View Design', 'View design details', 'design', 'read', FALSE),
('design:update', 'Update Design', 'Update design', 'design', 'update', FALSE),
('design:delete', 'Delete Design', 'Delete designs', 'design', 'delete', FALSE),
('design:export', 'Export Design', 'Export design files', 'design', 'export', FALSE),
('design:publish', 'Publish Design', 'Publish design versions', 'design', 'execute', FALSE),
-- 元素级权限
('element:create', 'Create Element', 'Create new elements', 'element', 'create', FALSE),
('element:read', 'View Element', 'View element details', 'element', 'read', FALSE),
('element:update', 'Update Element', 'Update elements', 'element', 'update', FALSE),
('element:delete', 'Delete Element', 'Delete elements', 'element', 'delete', FALSE),
-- 团队级权限
('team:manage', 'Manage Team', 'Manage team members', 'team', 'manage', FALSE),
('team:read', 'View Team', 'View team information', 'team', 'read', FALSE),
-- 用户级权限
('user:manage', 'Manage Users', 'Manage user accounts', 'user', 'manage', FALSE),
('user:read', 'View Users', 'View user information', 'user', 'read', FALSE);

-- 插入系统角色
INSERT INTO core.roles (name, description, is_system, is_default, permissions) VALUES
('Super Admin', 'Full system access', TRUE, FALSE, 
    '["system:admin", "system:monitor", "tenant:manage", "tenant:read", "project:manage", "design:manage", "element:manage", "user:manage"]'),
('Tenant Admin', 'Full tenant management', TRUE, FALSE, 
    '["tenant:manage", "tenant:read", "project:manage", "design:manage", "element:manage", "user:manage", "team:manage"]'),
('Project Manager', 'Project management', TRUE, FALSE, 
    '["project:manage", "project:read", "project:update", "project:share", "design:create", "design:read", "design:update", "design:publish", "element:create", "element:read", "element:update"]'),
('Designer', 'Design work', TRUE, TRUE, 
    '["project:read", "design:create", "design:read", "design:update", "design:export", "element:create", "element:read", "element:update"]'),
('Viewer', 'Read-only access', TRUE, FALSE, 
    '["project:read", "design:read", "element:read"]'),
('Reviewer', 'Review and comment', TRUE, FALSE, 
    '["project:read", "design:read", "design:export", "element:read"]');

-- 插入示例租户
INSERT INTO core.tenants (name, slug, description, plan_type, max_projects, max_storage_gb, max_users, settings) VALUES
('Demo Architecture Firm', 'demo-arch-firm', 'Demo tenant for architecture firm', 'professional', 50, 100, 20, 
    '{"theme": "light", "language": "zh-CN", "timezone": "Asia/Shanghai"}'::JSONB),
('Sample Construction Co', 'sample-construction', 'Demo tenant for construction company', 'enterprise', 100, 500, 50,
    '{"theme": "dark", "language": "en-US", "timezone": "America/New_York"}'::JSONB);

-- 插入示例用户（密码需要哈希处理，这里使用占位符）
INSERT INTO core.users (tenant_id, email, username, password_hash, first_name, last_name, role, email_verified, preferences) VALUES
((SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'), 'admin@demo-arch.com', 'admin', '$2b$12$placeholder_hash', 'Admin', 'User', 'admin', TRUE,
    '{"notifications": {"email": true, "push": true}, "editor": {"gridVisible": true, "snapToGrid": true}}'::JSONB),
((SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'), 'designer1@demo-arch.com', 'designer1', '$2b$12$placeholder_hash', 'John', 'Designer', 'designer', TRUE,
    '{"notifications": {"email": true, "push": false}, "editor": {"gridVisible": true, "snapToGrid": false}}'::JSONB),
((SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'), 'viewer1@demo-arch.com', 'viewer1', '$2b$12$placeholder_hash', 'Jane', 'Viewer', 'viewer', TRUE,
    '{"notifications": {"email": false, "push": false}}'::JSONB);

-- 插入示例项目
INSERT INTO core.projects (tenant_id, name, description, project_code, status, project_type, visibility, location, area_total_sqm, tags, created_by) VALUES
((SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'), 'Downtown Office Building', 'Modern office building in downtown area', 'PRJ-2024-001', 'in_progress', 'building', 'team',
    '{"country": "China", "city": "Shanghai", "address": "123 Main St", "coordinates": {"lat": 31.2304, "lng": 121.4737}}'::JSONB,
    5000.00, ARRAY['office', 'commercial', 'high-rise'],
    (SELECT id FROM core.users WHERE username = 'admin')),
((SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'), 'Residential Complex', 'Luxury residential complex', 'PRJ-2024-002', 'draft', 'building', 'private',
    '{"country": "China", "city": "Beijing", "address": "456 Park Ave", "coordinates": {"lat": 39.9042, "lng": 116.4074}}'::JSONB,
    12000.00, ARRAY['residential', 'luxury'],
    (SELECT id FROM core.users WHERE username = 'admin'));

-- 插入项目成员
INSERT INTO core.project_members (project_id, user_id, role, joined_by) VALUES
((SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'), 
 (SELECT id FROM core.users WHERE username = 'admin'), 'owner', 
 (SELECT id FROM core.users WHERE username = 'admin')),
((SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'), 
 (SELECT id FROM core.users WHERE username = 'designer1'), 'editor', 
 (SELECT id FROM core.users WHERE username = 'admin')),
((SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'), 
 (SELECT id FROM core.users WHERE username = 'viewer1'), 'viewer', 
 (SELECT id FROM core.users WHERE username = 'admin'));

-- 插入示例设计
INSERT INTO core.designs (project_id, tenant_id, name, description, design_type, file_format, status, scale, unit, created_by) VALUES
((SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'Floor Plan - Level 1', 'Ground floor plan', 'floor_plan', 'dwg', 'in_progress', '1:100', 'mm',
 (SELECT id FROM core.users WHERE username = 'designer1')),
((SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'Elevation - North', 'North elevation view', 'elevation', 'dwg', 'draft', '1:50', 'mm',
 (SELECT id FROM core.users WHERE username = 'designer1'));

-- 插入示例图层
INSERT INTO core.layers (design_id, project_id, tenant_id, name, description, display_order, color, line_type, line_weight, created_by) VALUES
((SELECT id FROM core.designs WHERE name = 'Floor Plan - Level 1'),
 (SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'Walls', 'Structural walls', 1, '#000000', 'solid', 0.5,
 (SELECT id FROM core.users WHERE username = 'designer1')),
((SELECT id FROM core.designs WHERE name = 'Floor Plan - Level 1'),
 (SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'Doors', 'Door openings', 2, '#0000FF', 'solid', 0.25,
 (SELECT id FROM core.users WHERE username = 'designer1')),
((SELECT id FROM core.designs WHERE name = 'Floor Plan - Level 1'),
 (SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'Windows', 'Window openings', 3, '#00FF00', 'solid', 0.25,
 (SELECT id FROM core.users WHERE username = 'designer1'));

-- 插入示例元素
INSERT INTO core.elements (design_id, layer_id, project_id, tenant_id, element_type, element_subtype, name, properties, created_by) VALUES
((SELECT id FROM core.designs WHERE name = 'Floor Plan - Level 1'),
 (SELECT id FROM core.layers WHERE name = 'Walls'),
 (SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'wall', 'concrete', 'Wall-001', '{"thickness": 200, "height": 3000, "material": "concrete"}'::JSONB,
 (SELECT id FROM core.users WHERE username = 'designer1')),
((SELECT id FROM core.designs WHERE name = 'Floor Plan - Level 1'),
 (SELECT id FROM core.layers WHERE name = 'Doors'),
 (SELECT id FROM core.projects WHERE project_code = 'PRJ-2024-001'),
 (SELECT id FROM core.tenants WHERE slug = 'demo-arch-firm'),
 'door', 'single', 'Door-001', '{"width": 900, "height": 2100, "material": "wood"}'::JSONB,
 (SELECT id FROM core.users WHERE username = 'designer1'));

-- 确认数据插入
SELECT 'Initial data inserted successfully' as status;
```

### 5.2 版本升级脚本

```sql
-- ============================================
-- 版本升级脚本
-- ============================================

-- 版本: 1.0.0 -> 1.1.0
-- 描述: 添加设计评论功能

DO $$
BEGIN
    -- 检查是否已存在评论表
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables 
                   WHERE table_schema = 'core' AND table_name = 'design_comments') THEN
        
        -- 创建设计评论表
        CREATE TABLE core.design_comments (
            id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            design_id           UUID NOT NULL REFERENCES core.designs(id) ON DELETE CASCADE,
            project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
            tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
            parent_comment_id   UUID REFERENCES core.design_comments(id),
            content             TEXT NOT NULL,
            -- 评论位置（可选，用于标注评论）
            position_x          DECIMAL(18, 6),
            position_y          DECIMAL(18, 6),
            position_z          DECIMAL(18, 6),
            -- 关联元素
            element_id          UUID REFERENCES core.elements(id),
            -- 评论状态
            status              VARCHAR(50) NOT NULL DEFAULT 'open'
                                CHECK (status IN ('open', 'resolved', 'closed')),
            resolved_at         TIMESTAMPTZ,
            resolved_by         UUID REFERENCES core.users(id),
            -- 时间戳
            created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            created_by          UUID REFERENCES core.users(id),
            updated_by          UUID REFERENCES core.users(id),
            deleted_at          TIMESTAMPTZ
        );
        
        -- 创建索引
        CREATE INDEX idx_design_comments_design ON core.design_comments(design_id);
        CREATE INDEX idx_design_comments_project ON core.design_comments(project_id);
        CREATE INDEX idx_design_comments_status ON core.design_comments(status);
        CREATE INDEX idx_design_comments_parent ON core.design_comments(parent_comment_id);
        CREATE INDEX idx_design_comments_element ON core.design_comments(element_id);
        
        -- 创建更新时间戳触发器
        CREATE TRIGGER trigger_design_comments_updated_at
            BEFORE UPDATE ON core.design_comments
            FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();
            
        RAISE NOTICE 'Created design_comments table';
    END IF;
END $$;

-- 版本: 1.1.0 -> 1.2.0
-- 描述: 添加设计审批工作流

DO $$
BEGIN
    -- 检查是否已存在审批表
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables 
                   WHERE table_schema = 'core' AND table_name = 'design_approvals') THEN
        
        -- 创建审批表
        CREATE TABLE core.design_approvals (
            id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            design_id           UUID NOT NULL REFERENCES core.designs(id) ON DELETE CASCADE,
            design_version_id   UUID NOT NULL REFERENCES core.design_versions(id),
            project_id          UUID NOT NULL REFERENCES core.projects(id) ON DELETE CASCADE,
            tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
            -- 审批流程
            workflow_name       VARCHAR(255) NOT NULL DEFAULT 'standard',
            -- 审批状态
            status              VARCHAR(50) NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'in_review', 'approved', 'rejected', 'escalated')),
            -- 审批人
            submitted_by        UUID NOT NULL REFERENCES core.users(id),
            submitted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            reviewed_by         UUID REFERENCES core.users(id),
            reviewed_at         TIMESTAMPTZ,
            -- 审批意见
            review_comments     TEXT,
            -- 时间戳
            created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        
        -- 创建审批历史表
        CREATE TABLE core.design_approval_history (
            id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            approval_id         UUID NOT NULL REFERENCES core.design_approvals(id) ON DELETE CASCADE,
            from_status         VARCHAR(50),
            to_status           VARCHAR(50) NOT NULL,
            changed_by          UUID REFERENCES core.users(id),
            comments            TEXT,
            created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        
        -- 创建索引
        CREATE INDEX idx_design_approvals_design ON core.design_approvals(design_id);
        CREATE INDEX idx_design_approvals_status ON core.design_approvals(status);
        CREATE INDEX idx_design_approvals_submitted ON core.design_approvals(submitted_at);
        CREATE INDEX idx_design_approval_history_approval ON core.design_approval_history(approval_id);
        
        -- 创建触发器
        CREATE TRIGGER trigger_design_approvals_updated_at
            BEFORE UPDATE ON core.design_approvals
            FOR EACH ROW EXECUTE FUNCTION core.update_updated_at_column();
            
        RAISE NOTICE 'Created design_approvals tables';
    END IF;
END $$;

-- 版本: 1.2.0 -> 1.3.0
-- 描述: 添加通知系统

DO $$
BEGIN
    -- 检查是否已存在通知表
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables 
                   WHERE table_schema = 'core' AND table_name = 'notifications') THEN
        
        -- 创建通知表
        CREATE TABLE core.notifications (
            id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            tenant_id           UUID NOT NULL REFERENCES core.tenants(id) ON DELETE CASCADE,
            user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
            -- 通知类型
            notification_type   VARCHAR(100) NOT NULL
                                CHECK (notification_type IN ('info', 'warning', 'success', 'error')),
            category            VARCHAR(100) NOT NULL
                                CHECK (category IN ('project', 'design', 'approval', 'comment', 'system', 'mention')),
            -- 通知内容
            title               VARCHAR(255) NOT NULL,
            message             TEXT NOT NULL,
            -- 关联资源
            resource_type       VARCHAR(100),
            resource_id         UUID,
            -- 操作链接
            action_url          VARCHAR(500),
            action_text         VARCHAR(100),
            -- 状态
            is_read             BOOLEAN NOT NULL DEFAULT FALSE,
            read_at             TIMESTAMPTZ,
            -- 时间戳
            created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            expires_at          TIMESTAMPTZ
        );
        
        -- 创建用户通知设置表
        CREATE TABLE core.notification_settings (
            id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            user_id             UUID NOT NULL REFERENCES core.users(id) ON DELETE CASCADE,
            category            VARCHAR(100) NOT NULL,
            email_enabled       BOOLEAN NOT NULL DEFAULT TRUE,
            push_enabled        BOOLEAN NOT NULL DEFAULT TRUE,
            in_app_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
            updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            
            UNIQUE(user_id, category)
        );
        
        -- 创建索引
        CREATE INDEX idx_notifications_user ON core.notifications(user_id);
        CREATE INDEX idx_notifications_unread ON core.notifications(user_id, is_read) WHERE is_read = FALSE;
        CREATE INDEX idx_notifications_created ON core.notifications(created_at DESC);
        CREATE INDEX idx_notifications_resource ON core.notifications(resource_type, resource_id);
        
        RAISE NOTICE 'Created notifications tables';
    END IF;
END $$;

-- 版本: 1.3.0 -> 1.4.0
-- 描述: 优化几何数据存储

DO $$
BEGIN
    -- 添加新的几何索引列
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_schema = 'geometry' AND table_name = 'geometries' AND column_name = 'simplified_level') THEN
        
        ALTER TABLE geometry.geometries ADD COLUMN simplified_level INTEGER DEFAULT 0;
        ALTER TABLE geometry.geometries ADD COLUMN quad_key VARCHAR(100);
        
        -- 创建四叉树索引
        CREATE INDEX idx_geometries_quad_key ON geometry.geometries(quad_key);
        
        RAISE NOTICE 'Added geometry optimization columns';
    END IF;
END $$;

-- 记录版本升级历史
CREATE TABLE IF NOT EXISTS core.schema_migrations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version             VARCHAR(20) NOT NULL UNIQUE,
    description         TEXT,
    applied_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by          VARCHAR(255),
    checksum            VARCHAR(64),
    execution_time_ms   INTEGER
);

-- 插入当前版本记录
INSERT INTO core.schema_migrations (version, description, applied_by) VALUES
('1.1.0', 'Add design comments feature', CURRENT_USER),
('1.2.0', 'Add design approval workflow', CURRENT_USER),
('1.3.0', 'Add notification system', CURRENT_USER),
('1.4.0', 'Optimize geometry storage', CURRENT_USER)
ON CONFLICT (version) DO NOTHING;

SELECT 'Schema migration completed successfully' as status;
```

### 5.3 数据清理脚本

```sql
-- ============================================
-- 数据清理脚本
-- ============================================

-- 清理已删除的软删除数据
CREATE OR REPLACE FUNCTION core.purge_deleted_data(
    p_older_than_days INTEGER DEFAULT 30,
    p_dry_run BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (table_name TEXT, rows_deleted BIGINT) AS $$
DECLARE
    v_table RECORD;
    v_count BIGINT;
    v_sql TEXT;
BEGIN
    FOR v_table IN 
        SELECT table_schema, table_name 
        FROM information_schema.tables 
        WHERE table_schema IN ('core', 'geometry', 'versioning')
          AND table_type = 'BASE TABLE'
    LOOP
        -- 检查表是否有deleted_at列
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_schema = v_table.table_schema 
              AND table_name = v_table.table_name 
              AND column_name = 'deleted_at'
        ) THEN
            v_sql := format(
                'SELECT COUNT(*) FROM %I.%I WHERE deleted_at < NOW() - INTERVAL ''%s days''',
                v_table.table_schema, v_table.table_name, p_older_than_days
            );
            
            EXECUTE v_sql INTO v_count;
            
            IF v_count > 0 THEN
                table_name := v_table.table_schema || '.' || v_table.table_name;
                rows_deleted := v_count;
                RETURN NEXT;
                
                IF NOT p_dry_run THEN
                    v_sql := format(
                        'DELETE FROM %I.%I WHERE deleted_at < NOW() - INTERVAL ''%s days''',
                        v_table.table_schema, v_table.table_name, p_older_than_days
                    );
                    EXECUTE v_sql;
                END IF;
            END IF;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 清理过期的会话
CREATE OR REPLACE FUNCTION core.cleanup_expired_sessions(
    p_batch_size INTEGER DEFAULT 1000
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM core.user_sessions
    WHERE expires_at < NOW()
    LIMIT p_batch_size;
    
    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 清理过期的API密钥
CREATE OR REPLACE FUNCTION core.cleanup_expired_api_keys(
    p_batch_size INTEGER DEFAULT 100
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    UPDATE core.api_keys
    SET is_active = FALSE
    WHERE expires_at < NOW() AND is_active = TRUE
    LIMIT p_batch_size;
    
    GET DIAGNOSTICS v_updated = ROW_COUNT;
    RETURN v_updated;
END;
$$ LANGUAGE plpgsql;

-- 清理过期的ACL缓存
CREATE OR REPLACE FUNCTION core.cleanup_expired_acl_cache(
    p_batch_size INTEGER DEFAULT 10000
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM core.acl_cache
    WHERE expires_at < NOW()
    LIMIT p_batch_size;
    
    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 清理孤儿几何数据
CREATE OR REPLACE FUNCTION geometry.cleanup_orphan_geometries(
    p_dry_run BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (orphan_count BIGINT, deleted_count BIGINT) AS $$
DECLARE
    v_orphan_count BIGINT;
    v_deleted_count BIGINT := 0;
BEGIN
    -- 查找没有对应元素的几何数据
    SELECT COUNT(*) INTO v_orphan_count
    FROM geometry.geometries g
    LEFT JOIN core.elements e ON g.element_id = e.id
    WHERE e.id IS NULL OR e.deleted_at IS NOT NULL;
    
    IF NOT p_dry_run AND v_orphan_count > 0 THEN
        DELETE FROM geometry.geometries g
        WHERE EXISTS (
            SELECT 1 FROM core.elements e 
            WHERE e.id = g.element_id 
              AND (e.id IS NULL OR e.deleted_at IS NOT NULL)
        );
        
        GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    END IF;
    
    orphan_count := v_orphan_count;
    deleted_count := v_deleted_count;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- 清理过期的快照
CREATE OR REPLACE FUNCTION versioning.cleanup_old_snapshots(
    p_keep_count INTEGER DEFAULT 10,
    p_older_than_days INTEGER DEFAULT 30
)
RETURNS TABLE (aggregate_type VARCHAR(100), aggregate_id UUID, deleted_count INTEGER) AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    FOR aggregate_type, aggregate_id IN
        SELECT DISTINCT s.aggregate_type, s.aggregate_id
        FROM versioning.snapshots s
    LOOP
        WITH ranked_snapshots AS (
            SELECT id, ROW_NUMBER() OVER (ORDER BY version DESC) as rn
            FROM versioning.snapshots
            WHERE aggregate_type = aggregate_type AND aggregate_id = aggregate_id
        )
        DELETE FROM versioning.snapshots
        WHERE id IN (
            SELECT id FROM ranked_snapshots
            WHERE rn > p_keep_count
        )
        AND created_at < NOW() - (p_older_than_days || ' days')::INTERVAL;
        
        GET DIAGNOSTICS v_deleted = ROW_COUNT;
        
        IF v_deleted > 0 THEN
            deleted_count := v_deleted;
            RETURN NEXT;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 清理旧事件（保留最近N个）
CREATE OR REPLACE FUNCTION versioning.cleanup_old_events(
    p_keep_per_aggregate INTEGER DEFAULT 1000,
    p_older_than_days INTEGER DEFAULT 90
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER := 0;
    v_batch INTEGER;
BEGIN
    LOOP
        WITH events_to_delete AS (
            SELECT e.id
            FROM versioning.events e
            JOIN (
                SELECT aggregate_type, aggregate_id, 
                       MAX(sequence_number) as max_seq
                FROM versioning.events
                WHERE occurred_at < NOW() - (p_older_than_days || ' days')::INTERVAL
                GROUP BY aggregate_type, aggregate_id
            ) latest ON e.aggregate_type = latest.aggregate_type 
                    AND e.aggregate_id = latest.aggregate_id
            WHERE e.sequence_number <= latest.max_seq - p_keep_per_aggregate
            LIMIT 10000
        )
        DELETE FROM versioning.events
        WHERE id IN (SELECT id FROM events_to_delete);
        
        GET DIAGNOSTICS v_batch = ROW_COUNT;
        v_deleted := v_deleted + v_batch;
        
        EXIT WHEN v_batch = 0;
    END LOOP;
    
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 运行所有清理任务
CREATE OR REPLACE FUNCTION core.run_all_cleanup_tasks(
    p_dry_run BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (task_name TEXT, result TEXT) AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- 清理过期会话
    IF p_dry_run THEN
        SELECT COUNT(*) INTO v_count FROM core.user_sessions WHERE expires_at < NOW();
        task_name := 'cleanup_expired_sessions';
        result := format('Would delete %s expired sessions', v_count);
        RETURN NEXT;
    ELSE
        task_name := 'cleanup_expired_sessions';
        result := format('Deleted %s expired sessions', core.cleanup_expired_sessions());
        RETURN NEXT;
    END IF;
    
    -- 清理过期ACL缓存
    IF p_dry_run THEN
        SELECT COUNT(*) INTO v_count FROM core.acl_cache WHERE expires_at < NOW();
        task_name := 'cleanup_expired_acl_cache';
        result := format('Would delete %s expired cache entries', v_count);
        RETURN NEXT;
    ELSE
        task_name := 'cleanup_expired_acl_cache';
        result := format('Deleted %s expired cache entries', core.cleanup_expired_acl_cache());
        RETURN NEXT;
    END IF;
    
    -- 归档审计日志
    IF p_dry_run THEN
        SELECT COUNT(*) INTO v_count FROM audit.audit_logs WHERE created_at < NOW() - INTERVAL '90 days';
        task_name := 'archive_old_audit_logs';
        result := format('Would archive %s old audit logs', v_count);
        RETURN NEXT;
    ELSE
        task_name := 'archive_old_audit_logs';
        result := format('Archived %s old audit logs', audit.archive_old_logs());
        RETURN NEXT;
    END IF;
    
    -- 清理过期快照
    IF NOT p_dry_run THEN
        task_name := 'cleanup_old_snapshots';
        result := 'Cleaned up old snapshots';
        RETURN NEXT;
    END IF;
END;
$$ LANGUAGE plpgsql;
```


---

## 6. 性能优化

### 6.1 查询优化建议

```sql
-- ============================================
-- 查询优化建议
-- ============================================

-- 1. 使用EXPLAIN ANALYZE分析慢查询
-- 示例：分析项目查询性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT p.*, 
       COUNT(DISTINCT d.id) as design_count,
       COUNT(DISTINCT pm.user_id) as member_count
FROM core.projects p
LEFT JOIN core.designs d ON p.id = d.project_id AND d.deleted_at IS NULL
LEFT JOIN core.project_members pm ON p.id = pm.project_id
WHERE p.tenant_id = '00000000-0000-0000-0000-000000000000'
  AND p.deleted_at IS NULL
GROUP BY p.id;

-- 2. 优化分页查询（使用键集分页替代OFFSET）
-- 低效：SELECT * FROM core.projects ORDER BY created_at DESC LIMIT 10 OFFSET 10000;
-- 高效：使用最后一条记录的时间戳
CREATE OR REPLACE FUNCTION core.get_projects_paginated(
    p_tenant_id UUID,
    p_last_created_at TIMESTAMPTZ DEFAULT NULL,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    created_at TIMESTAMPTZ,
    has_more BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id, p.name, p.created_at,
        EXISTS (
            SELECT 1 FROM core.projects p2 
            WHERE p2.tenant_id = p_tenant_id 
              AND p2.deleted_at IS NULL
              AND p2.created_at < p.created_at
            LIMIT 1
        ) as has_more
    FROM core.projects p
    WHERE p.tenant_id = p_tenant_id
      AND p.deleted_at IS NULL
      AND (p_last_created_at IS NULL OR p.created_at < p_last_created_at)
    ORDER BY p.created_at DESC
    LIMIT p_limit + 1;
END;
$$ LANGUAGE plpgsql;

-- 3. 预计算常用聚合
CREATE MATERIALIZED VIEW analytics.mv_project_stats AS
SELECT 
    p.id as project_id,
    p.tenant_id,
    COUNT(DISTINCT d.id) FILTER (WHERE d.deleted_at IS NULL) as design_count,
    COUNT(DISTINCT pm.user_id) as member_count,
    SUM(d.element_count) as total_elements,
    MAX(d.updated_at) as last_design_update
FROM core.projects p
LEFT JOIN core.designs d ON p.id = d.project_id
LEFT JOIN core.project_members pm ON p.id = pm.project_id
WHERE p.deleted_at IS NULL
GROUP BY p.id, p.tenant_id;

-- 创建唯一索引用于并发刷新
CREATE UNIQUE INDEX idx_mv_project_stats_project ON analytics.mv_project_stats(project_id);

-- 刷新物化视图
REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.mv_project_stats;

-- 4. 使用LATERAL JOIN优化相关子查询
-- 获取每个项目的最新设计
SELECT p.id, p.name, latest_design.*
FROM core.projects p
LEFT JOIN LATERAL (
    SELECT d.id, d.name, d.created_at
    FROM core.designs d
    WHERE d.project_id = p.id AND d.deleted_at IS NULL
    ORDER BY d.created_at DESC
    LIMIT 1
) latest_design ON TRUE
WHERE p.deleted_at IS NULL;

-- 5. 批量操作优化
CREATE OR REPLACE FUNCTION core.batch_update_elements(
    p_element_ids UUID[],
    p_updates JSONB
)
RETURNS INTEGER AS $$
DECLARE
    v_updated INTEGER;
BEGIN
    UPDATE core.elements
    SET 
        properties = properties || p_updates->'properties',
        updated_at = NOW(),
        updated_by = (p_updates->>'updated_by')::UUID
    WHERE id = ANY(p_element_ids);
    
    GET DIAGNOSTICS v_updated = ROW_COUNT;
    RETURN v_updated;
END;
$$ LANGUAGE plpgsql;

-- 6. 空间查询优化（使用合适的索引策略）
-- 使用空间索引进行边界框查询
CREATE OR REPLACE FUNCTION geometry.fast_bbox_query(
    p_min_x DECIMAL,
    p_min_y DECIMAL,
    p_max_x DECIMAL,
    p_max_y DECIMAL,
    p_project_id UUID
)
RETURNS TABLE (geometry_id UUID, element_id UUID) AS $$
DECLARE
    v_bbox GEOMETRY;
BEGIN
    v_bbox := ST_MakeEnvelope(p_min_x, p_min_y, p_max_x, p_max_y, 4326);
    
    RETURN QUERY
    SELECT g.id, g.element_id
    FROM geometry.geometries g
    WHERE g.bbox && v_bbox  -- 使用空间索引的边界框查询
      AND g.project_id = p_project_id
      AND ST_Intersects(g.geom_2d, v_bbox)  -- 精确相交检测
    LIMIT 1000;
END;
$$ LANGUAGE plpgsql;

-- 7. 查询提示和优化器设置
SET enable_seqscan = off;  -- 强制使用索引（仅用于测试）
SET work_mem = '256MB';    -- 增加排序和哈希操作的内存
SET effective_cache_size = '4GB';  -- 设置有效缓存大小
```

### 6.2 索引优化建议

```sql
-- ============================================
-- 索引优化建议
-- ============================================

-- 1. 复合索引策略
-- 为常见查询模式创建复合索引
CREATE INDEX idx_projects_tenant_status_created 
    ON core.projects(tenant_id, status, created_at DESC);

CREATE INDEX idx_designs_project_type_status 
    ON core.designs(project_id, design_type, status);

CREATE INDEX idx_elements_design_layer_type 
    ON core.elements(design_id, layer_id, element_type);

-- 2. 部分索引（Partial Indexes）
-- 只为活跃记录创建索引
CREATE INDEX idx_users_active_email 
    ON core.users(email) 
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE INDEX idx_projects_active 
    ON core.projects(created_at DESC) 
    WHERE status IN ('in_progress', 'draft') AND deleted_at IS NULL;

-- 3. 表达式索引
-- 为常用表达式创建索引
CREATE INDEX idx_users_lower_email 
    ON core.users(LOWER(email));

CREATE INDEX idx_projects_name_lower 
    ON core.projects(LOWER(name));

-- 4. 覆盖索引（Covering Indexes）
-- 包含查询所需的所有列
CREATE INDEX idx_designs_covering 
    ON core.designs(project_id, status, name, design_type, created_at)
    INCLUDE (element_count, version_count, thumbnail_url);

-- 5. GIN索引（用于JSONB和数组）
-- JSONB索引
CREATE INDEX idx_projects_settings_gin 
    ON core.projects USING GIN(settings jsonb_path_ops);

CREATE INDEX idx_elements_properties_gin 
    ON core.elements USING GIN(properties jsonb_path_ops);

-- 数组索引
CREATE INDEX idx_projects_tags_gin 
    ON core.projects USING GIN(tags);

-- 6. BRIN索引（用于大表的时间序列数据）
-- 适用于按时间顺序插入的大表
CREATE INDEX idx_audit_logs_created_brin 
    ON audit.audit_logs USING BRIN(created_at)
    WITH (pages_per_range = 128);

CREATE INDEX idx_events_occurred_brin 
    ON versioning.events USING BRIN(occurred_at)
    WITH (pages_per_range = 128);

-- 7. 空间索引（PostGIS）
-- R-tree GiST索引
CREATE INDEX idx_geometries_geom_gist 
    ON geometry.geometries USING GIST(geom_2d);

-- 边界框索引
CREATE INDEX idx_geometries_bbox_gist 
    ON geometry.geometries USING GIST(bbox);

-- 8. 索引维护和监控
-- 查看索引使用情况
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit')
ORDER BY pg_relation_size(indexrelid) DESC;

-- 查找未使用的索引
SELECT 
    schemaname,
    tablename,
    indexname
FROM pg_stat_user_indexes
WHERE idx_scan = 0 
  AND schemaname IN ('core', 'geometry', 'versioning', 'audit')
ORDER BY pg_relation_size(indexrelid) DESC;

-- 重建索引
REINDEX INDEX CONCURRENTLY idx_projects_tenant_id;

-- 分析表和索引统计信息
ANALYZE core.projects;
ANALYZE core.designs;
ANALYZE core.elements;
ANALYZE geometry.geometries;
```

### 6.3 分区策略实现

```sql
-- ============================================
-- 分区策略实现
-- ============================================

-- 1. 审计日志表分区（按时间范围分区）
-- 创建主分区表
CREATE TABLE audit.audit_logs_partitioned (
    id                  UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    action              VARCHAR(100) NOT NULL,
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
    created_at          TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建分区管理函数
CREATE OR REPLACE FUNCTION audit.create_monthly_partition(
    p_year INTEGER,
    p_month INTEGER
)
RETURNS TEXT AS $$
DECLARE
    v_partition_name TEXT;
    v_start_date DATE;
    v_end_date DATE;
BEGIN
    v_partition_name := format('audit_logs_%s_%s', p_year, LPAD(p_month::TEXT, 2, '0'));
    v_start_date := make_date(p_year, p_month, 1);
    v_end_date := v_start_date + INTERVAL '1 month';
    
    -- 检查分区是否已存在
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE schemaname = 'audit' AND tablename = v_partition_name
    ) THEN
        EXECUTE format(
            'CREATE TABLE audit.%I PARTITION OF audit.audit_logs_partitioned
             FOR VALUES FROM (%L) TO (%L)',
            v_partition_name, v_start_date, v_end_date
        );
        
        -- 为分区创建索引
        EXECUTE format(
            'CREATE INDEX idx_%s_tenant ON audit.%I(tenant_id)',
            v_partition_name, v_partition_name
        );
        
        EXECUTE format(
            'CREATE INDEX idx_%s_created ON audit.%I(created_at DESC)',
            v_partition_name, v_partition_name
        );
        
        RAISE NOTICE 'Created partition: %', v_partition_name;
    END IF;
    
    RETURN v_partition_name;
END;
$$ LANGUAGE plpgsql;

-- 自动创建未来12个月的分区
DO $$
DECLARE
    i INTEGER;
    v_year INTEGER;
    v_month INTEGER;
BEGIN
    FOR i IN 0..12 LOOP
        v_year := EXTRACT(YEAR FROM (NOW() + (i || ' months')::INTERVAL))::INTEGER;
        v_month := EXTRACT(MONTH FROM (NOW() + (i || ' months')::INTERVAL))::INTEGER;
        PERFORM audit.create_monthly_partition(v_year, v_month);
    END LOOP;
END $$;

-- 2. 事件表分区（按租户ID哈希分区）
CREATE TABLE versioning.events_partitioned (
    id                  UUID NOT NULL,
    aggregate_type      VARCHAR(100) NOT NULL,
    aggregate_id        UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    event_type          VARCHAR(200) NOT NULL,
    event_version       INTEGER NOT NULL DEFAULT 1,
    payload             JSONB NOT NULL,
    metadata            JSONB DEFAULT '{}',
    sequence_number     BIGINT NOT NULL,
    global_sequence     BIGSERIAL,
    correlation_id      UUID,
    causation_id        UUID,
    occurred_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id             UUID,
    session_id          UUID,
    source_ip           INET,
    source_service      VARCHAR(100),
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

-- 创建8个哈希分区
DO $$
DECLARE
    i INTEGER;
BEGIN
    FOR i IN 0..7 LOOP
        EXECUTE format(
            'CREATE TABLE versioning.events_p%s PARTITION OF versioning.events_partitioned
             FOR VALUES WITH (MODULUS 8, REMAINDER %s)',
            i, i
        );
    END LOOP;
END $$;

-- 3. 几何数据分区（按项目ID范围分区）
CREATE TABLE geometry.geometries_partitioned (
    id                  UUID NOT NULL,
    element_id          UUID NOT NULL,
    design_id           UUID NOT NULL,
    project_id          UUID NOT NULL,
    tenant_id           UUID NOT NULL,
    geometry_type       VARCHAR(50) NOT NULL,
    geom_2d             GEOMETRY(GEOMETRY, 4326),
    geom_3d             GEOMETRY(GEOMETRYZ, 0),
    geom_simplified     GEOMETRY(GEOMETRY, 4326),
    bbox                GEOMETRY(POLYGON, 4326),
    area                DECIMAL(18, 6),
    length              DECIMAL(18, 6),
    perimeter           DECIMAL(18, 6),
    centroid            GEOMETRY(POINT, 4326),
    vertex_count        INTEGER,
    precision_mm        DECIMAL(10, 4) DEFAULT 1.0,
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version             INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (id, project_id)
) PARTITION BY RANGE (project_id);

-- 4. 分区维护函数
CREATE OR REPLACE FUNCTION audit.archive_old_partition(
    p_partition_name TEXT,
    p_archive_table TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_archive_table TEXT;
BEGIN
    v_archive_table := COALESCE(p_archive_table, p_partition_name || '_archive');
    
    -- 创建归档表
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS audit.%I (LIKE audit.%I INCLUDING ALL)',
        v_archive_table, p_partition_name
    );
    
    -- 将数据移动到归档表
    EXECUTE format(
        'INSERT INTO audit.%I SELECT * FROM audit.%I',
        v_archive_table, p_partition_name
    );
    
    -- 删除分区
    EXECUTE format(
        'DROP TABLE audit.%I',
        p_partition_name
    );
    
    RAISE NOTICE 'Archived partition % to %', p_partition_name, v_archive_table;
    RETURN TRUE;
EXCEPTION WHEN OTHERS THEN
    RAISE WARNING 'Failed to archive partition %: %', p_partition_name, SQLERRM;
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- 5. 分区监控视图
CREATE OR REPLACE VIEW analytics.v_partition_stats AS
SELECT 
    schemaname,
    tablename as partition_name,
    pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) as total_size,
    pg_total_relation_size(schemaname || '.' || tablename) as size_bytes,
    (SELECT COUNT(*) FROM information_schema.columns 
     WHERE table_schema = schemaname AND table_name = tablename) as column_count,
    (SELECT reltuples::BIGINT FROM pg_class WHERE oid = (schemaname || '.' || tablename)::regclass) as estimated_rows
FROM pg_tables
WHERE schemaname IN ('audit', 'versioning', 'geometry')
  AND tablename LIKE '%_20%'  -- 分区表命名模式
ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC;
```

### 6.4 缓存策略实现

```sql
-- ============================================
-- 缓存策略实现
-- ============================================

-- 1. 数据库内缓存表
-- 查询结果缓存
CREATE TABLE core.query_cache (
    cache_key           VARCHAR(255) PRIMARY KEY,
    query_hash          VARCHAR(64) NOT NULL,
    result_data         JSONB NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    hit_count           INTEGER NOT NULL DEFAULT 0,
    last_accessed_at    TIMESTAMPTZ
);

-- 创建过期索引
CREATE INDEX idx_query_cache_expires ON core.query_cache(expires_at);

-- 2. 缓存管理函数
CREATE OR REPLACE FUNCTION core.get_cached_result(
    p_cache_key VARCHAR(255)
)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    SELECT result_data INTO v_result
    FROM core.query_cache
    WHERE cache_key = p_cache_key
      AND expires_at > NOW();
    
    IF v_result IS NOT NULL THEN
        -- 更新访问统计
        UPDATE core.query_cache
        SET hit_count = hit_count + 1,
            last_accessed_at = NOW()
        WHERE cache_key = p_cache_key;
    END IF;
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION core.set_cached_result(
    p_cache_key VARCHAR(255),
    p_query_hash VARCHAR(64),
    p_result_data JSONB,
    p_ttl_seconds INTEGER DEFAULT 300
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO core.query_cache (cache_key, query_hash, result_data, expires_at)
    VALUES (p_cache_key, p_query_hash, p_result_data, NOW() + (p_ttl_seconds || ' seconds')::INTERVAL)
    ON CONFLICT (cache_key) 
    DO UPDATE SET 
        query_hash = p_query_hash,
        result_data = p_result_data,
        expires_at = NOW() + (p_ttl_seconds || ' seconds')::INTERVAL,
        created_at = NOW(),
        hit_count = 0,
        last_accessed_at = NULL;
END;
$$ LANGUAGE plpgsql;

-- 3. 项目列表缓存函数
CREATE OR REPLACE FUNCTION core.get_projects_with_cache(
    p_tenant_id UUID,
    p_user_id UUID,
    p_status VARCHAR(50) DEFAULT NULL,
    p_limit INTEGER DEFAULT 20,
    p_use_cache BOOLEAN DEFAULT TRUE
)
RETURNS JSONB AS $$
DECLARE
    v_cache_key VARCHAR(255);
    v_query_hash VARCHAR(64);
    v_cached_result JSONB;
    v_result JSONB;
BEGIN
    -- 生成缓存键
    v_cache_key := format('projects:%s:%s:%s:%s', p_tenant_id, p_user_id, COALESCE(p_status, 'all'), p_limit);
    v_query_hash := md5(v_cache_key);
    
    -- 尝试从缓存获取
    IF p_use_cache THEN
        v_cached_result := core.get_cached_result(v_cache_key);
        IF v_cached_result IS NOT NULL THEN
            RETURN jsonb_build_object('data', v_cached_result, 'from_cache', true);
        END IF;
    END IF;
    
    -- 查询数据库
    SELECT jsonb_agg(row_to_json(t))
    INTO v_result
    FROM (
        SELECT 
            p.id, p.name, p.description, p.status, p.project_type,
            p.thumbnail_url, p.progress_percent, p.created_at, p.updated_at,
            (SELECT COUNT(*) FROM core.designs d WHERE d.project_id = p.id AND d.deleted_at IS NULL) as design_count,
            (SELECT COUNT(*) FROM core.project_members pm WHERE pm.project_id = p.id) as member_count
        FROM core.projects p
        WHERE p.tenant_id = p_tenant_id
          AND p.deleted_at IS NULL
          AND core.check_project_access(p_user_id, p.id, 'viewer')
          AND (p_status IS NULL OR p.status = p_status)
        ORDER BY p.updated_at DESC
        LIMIT p_limit
    ) t;
    
    -- 缓存结果
    IF p_use_cache THEN
        PERFORM core.set_cached_result(v_cache_key, v_query_hash, v_result, 60);
    END IF;
    
    RETURN jsonb_build_object('data', v_result, 'from_cache', false);
END;
$$ LANGUAGE plpgsql;

-- 4. 缓存清理函数
CREATE OR REPLACE FUNCTION core.cleanup_expired_cache(
    p_batch_size INTEGER DEFAULT 1000
)
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM core.query_cache
    WHERE expires_at < NOW()
    LIMIT p_batch_size;
    
    GET DIAGNOSTICS v_deleted = ROW_COUNT;
    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- 5. 缓存统计视图
CREATE OR REPLACE VIEW analytics.v_cache_stats AS
SELECT 
    COUNT(*) as total_entries,
    COUNT(*) FILTER (WHERE expires_at > NOW()) as active_entries,
    COUNT(*) FILTER (WHERE expires_at <= NOW()) as expired_entries,
    SUM(hit_count) as total_hits,
    AVG(hit_count)::INTEGER as avg_hits,
    MAX(hit_count) as max_hits,
    pg_size_pretty(SUM(pg_column_size(result_data))) as total_cache_size
FROM core.query_cache;

-- 6. Redis缓存集成函数（伪代码，实际需要在应用层实现）
/*
-- Redis缓存键命名规范
project:{project_id}:details
design:{design_id}:versions
user:{user_id}:permissions
element:{element_id}:geometry
spatial:{project_id}:bbox:{min_x}:{min_y}:{max_x}:{max_y}

-- 缓存TTL设置
项目详情: 5分钟
设计列表: 2分钟
用户权限: 10分钟
几何数据: 1小时（不变数据）
空间查询: 30秒（热点区域）
*/

-- 7. 缓存预热函数
CREATE OR REPLACE FUNCTION core.warmup_cache(
    p_tenant_id UUID,
    p_warmup_type VARCHAR(50) DEFAULT 'all'
)
RETURNS TABLE (cache_type TEXT, entries_warmed INTEGER) AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- 预热项目列表缓存
    IF p_warmup_type IN ('all', 'projects') THEN
        SELECT COUNT(*) INTO v_count
        FROM core.projects
        WHERE tenant_id = p_tenant_id AND deleted_at IS NULL;
        
        cache_type := 'projects';
        entries_warmed := v_count;
        RETURN NEXT;
    END IF;
    
    -- 预热活跃用户缓存
    IF p_warmup_type IN ('all', 'users') THEN
        SELECT COUNT(*) INTO v_count
        FROM core.users
        WHERE tenant_id = p_tenant_id AND status = 'active' AND deleted_at IS NULL;
        
        cache_type := 'users';
        entries_warmed := v_count;
        RETURN NEXT;
    END IF;
END;
$$ LANGUAGE plpgsql;
```

---

## 7. 备份恢复脚本

### 7.1 全量备份脚本

```bash
#!/bin/bash
# ============================================
# 全量备份脚本 (full_backup.sh)
# ============================================

# 配置
DB_NAME="archdesign_platform"
DB_USER="postgres"
DB_HOST="localhost"
DB_PORT="5433"
BACKUP_DIR="/backup/postgresql"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/full_${DB_NAME}_${DATE}.sql"

# 创建备份目录
mkdir -p "${BACKUP_DIR}"

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${BACKUP_DIR}/backup.log"
}

log "Starting full backup of ${DB_NAME}..."

# 执行全量备份
pg_dump \
    --host="${DB_HOST}" \
    --port="${DB_PORT}" \
    --username="${DB_USER}" \
    --dbname="${DB_NAME}" \
    --format=custom \
    --compress=9 \
    --verbose \
    --file="${BACKUP_FILE}.dump" \
    2>> "${BACKUP_DIR}/backup.log"

if [ $? -eq 0 ]; then
    log "Full backup completed successfully: ${BACKUP_FILE}.dump"
    
    # 计算文件大小
    FILE_SIZE=$(du -h "${BACKUP_FILE}.dump" | cut -f1)
    log "Backup file size: ${FILE_SIZE}"
    
    # 生成校验和
    sha256sum "${BACKUP_FILE}.dump" > "${BACKUP_FILE}.sha256"
    log "Checksum generated: ${BACKUP_FILE}.sha256"
    
    # 清理旧备份（保留最近30天）
    find "${BACKUP_DIR}" -name "full_${DB_NAME}_*.dump" -mtime +30 -delete
    find "${BACKUP_DIR}" -name "full_${DB_NAME}_*.sha256" -mtime +30 -delete
    log "Old backups cleaned up"
    
    exit 0
else
    log "ERROR: Full backup failed!"
    exit 1
fi
```

### 7.2 增量备份脚本

```bash
#!/bin/bash
# ============================================
# 增量备份脚本 (incremental_backup.sh)
# ============================================

# 配置
DB_NAME="archdesign_platform"
DB_USER="postgres"
DB_HOST="localhost"
DB_PORT="5433"
BACKUP_DIR="/backup/postgresql"
WAL_ARCHIVE_DIR="/backup/postgresql/wal"
DATE=$(date +%Y%m%d_%H%M%S)

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${BACKUP_DIR}/backup.log"
}

log "Starting incremental backup..."

# 创建WAL归档目录
mkdir -p "${WAL_ARCHIVE_DIR}"

# 使用pg_basebackup进行增量备份
pg_basebackup \
    --host="${DB_HOST}" \
    --port="${DB_PORT}" \
    --username="${DB_USER}" \
    --pgdata="${BACKUP_DIR}/incremental_${DATE}" \
    --format=tar \
    --gzip \
    --progress \
    --wal-method=fetch \
    2>> "${BACKUP_DIR}/backup.log"

if [ $? -eq 0 ]; then
    log "Incremental backup completed successfully"
    
    # 记录备份信息
    echo "${DATE}" > "${BACKUP_DIR}/last_incremental_backup.txt"
    
    # 清理旧增量备份（保留最近7天）
    find "${BACKUP_DIR}" -name "incremental_*" -mtime +7 -exec rm -rf {} \;
    log "Old incremental backups cleaned up"
    
    exit 0
else
    log "ERROR: Incremental backup failed!"
    exit 1
fi
```

### 7.3 数据恢复脚本

```bash
#!/bin/bash
# ============================================
# 数据恢复脚本 (restore_backup.sh)
# ============================================

# 配置
DB_NAME="archdesign_platform"
DB_USER="postgres"
DB_HOST="localhost"
DB_PORT="5433"
BACKUP_DIR="/backup/postgresql"

# 显示用法
usage() {
    echo "Usage: $0 <backup_file> [options]"
    echo "Options:"
    echo "  --full          Restore full backup (default)"
    echo "  --schema-only   Restore schema only"
    echo "  --data-only     Restore data only"
    echo "  --target-db     Target database name"
    exit 1
}

# 检查参数
if [ $# -lt 1 ]; then
    usage
fi

BACKUP_FILE="$1"
shift

RESTORE_MODE="full"
TARGET_DB="${DB_NAME}"

# 解析选项
while [[ $# -gt 0 ]]; do
    case $1 in
        --full) RESTORE_MODE="full" ;;
        --schema-only) RESTORE_MODE="schema" ;;
        --data-only) RESTORE_MODE="data" ;;
        --target-db) TARGET_DB="$2"; shift ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
    shift
done

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 验证备份文件
if [ ! -f "${BACKUP_FILE}" ]; then
    log "ERROR: Backup file not found: ${BACKUP_FILE}"
    exit 1
fi

# 验证校验和
if [ -f "${BACKUP_FILE}.sha256" ]; then
    log "Verifying backup checksum..."
    sha256sum -c "${BACKUP_FILE}.sha256"
    if [ $? -ne 0 ]; then
        log "ERROR: Backup file checksum verification failed!"
        exit 1
    fi
    log "Checksum verification passed"
fi

log "Starting restore of ${BACKUP_FILE} to ${TARGET_DB}..."

# 创建目标数据库（如果不存在）
psql \
    --host="${DB_HOST}" \
    --port="${DB_PORT}" \
    --username="${DB_USER}" \
    --dbname="postgres" \
    --command="DROP DATABASE IF EXISTS ${TARGET_DB}; CREATE DATABASE ${TARGET_DB};"

# 执行恢复
case ${RESTORE_MODE} in
    full)
        pg_restore \
            --host="${DB_HOST}" \
            --port="${DB_PORT}" \
            --username="${DB_USER}" \
            --dbname="${TARGET_DB}" \
            --verbose \
            --jobs=4 \
            "${BACKUP_FILE}"
        ;;
    schema)
        pg_restore \
            --host="${DB_HOST}" \
            --port="${DB_PORT}" \
            --username="${DB_USER}" \
            --dbname="${TARGET_DB}" \
            --verbose \
            --schema-only \
            "${BACKUP_FILE}"
        ;;
    data)
        pg_restore \
            --host="${DB_HOST}" \
            --port="${DB_PORT}" \
            --username="${DB_USER}" \
            --dbname="${TARGET_DB}" \
            --verbose \
            --data-only \
            --disable-triggers \
            "${BACKUP_FILE}"
        ;;
esac

if [ $? -eq 0 ]; then
    log "Restore completed successfully"
    
    # 验证恢复
    TABLE_COUNT=$(psql \
        --host="${DB_HOST}" \
        --port="${DB_PORT}" \
        --username="${DB_USER}" \
        --dbname="${TARGET_DB}" \
        --tuples-only \
        --command="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema IN ('core', 'geometry', 'versioning', 'audit');" | xargs)
    
    log "Restored database contains ${TABLE_COUNT} tables"
    
    exit 0
else
    log "ERROR: Restore failed!"
    exit 1
fi
```

### 7.4 数据验证脚本

```sql
-- ============================================
-- 数据验证脚本
-- ============================================

-- 1. 数据完整性检查函数
CREATE OR REPLACE FUNCTION core.validate_data_integrity()
RETURNS TABLE (check_name TEXT, status TEXT, details TEXT) AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- 检查孤儿项目成员
    check_name := 'orphan_project_members';
    SELECT COUNT(*) INTO v_count
    FROM core.project_members pm
    LEFT JOIN core.projects p ON pm.project_id = p.id
    WHERE p.id IS NULL OR p.deleted_at IS NOT NULL;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'No orphan project members found';
    ELSE
        status := 'FAIL';
        details := format('%s orphan project members found', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查孤儿设计
    check_name := 'orphan_designs';
    SELECT COUNT(*) INTO v_count
    FROM core.designs d
    LEFT JOIN core.projects p ON d.project_id = p.id
    WHERE p.id IS NULL OR p.deleted_at IS NOT NULL;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'No orphan designs found';
    ELSE
        status := 'FAIL';
        details := format('%s orphan designs found', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查孤儿元素
    check_name := 'orphan_elements';
    SELECT COUNT(*) INTO v_count
    FROM core.elements e
    LEFT JOIN core.designs d ON e.design_id = d.id
    WHERE d.id IS NULL OR d.deleted_at IS NOT NULL;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'No orphan elements found';
    ELSE
        status := 'FAIL';
        details := format('%s orphan elements found', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查孤儿几何数据
    check_name := 'orphan_geometries';
    SELECT COUNT(*) INTO v_count
    FROM geometry.geometries g
    LEFT JOIN core.elements e ON g.element_id = e.id
    WHERE e.id IS NULL OR e.deleted_at IS NOT NULL;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'No orphan geometries found';
    ELSE
        status := 'FAIL';
        details := format('%s orphan geometries found', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查无效的用户状态
    check_name := 'invalid_user_status';
    SELECT COUNT(*) INTO v_count
    FROM core.users
    WHERE status NOT IN ('active', 'inactive', 'suspended', 'pending');
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'All users have valid status';
    ELSE
        status := 'FAIL';
        details := format('%s users have invalid status', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查负值计数
    check_name := 'negative_counts';
    SELECT COUNT(*) INTO v_count
    FROM core.designs
    WHERE element_count < 0 OR version_count < 0;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'No negative counts found';
    ELSE
        status := 'FAIL';
        details := format('%s records have negative counts', v_count);
    END IF;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- 2. 数据一致性检查
CREATE OR REPLACE FUNCTION core.validate_data_consistency()
RETURNS TABLE (check_name TEXT, status TEXT, details TEXT) AS $$
DECLARE
    v_count INTEGER;
    v_expected INTEGER;
    v_actual INTEGER;
BEGIN
    -- 检查项目元素计数一致性
    check_name := 'project_element_count';
    SELECT COUNT(*) INTO v_count
    FROM (
        SELECT p.id, p.element_count as expected,
               (SELECT COALESCE(SUM(d.element_count), 0) FROM core.designs d WHERE d.project_id = p.id AND d.deleted_at IS NULL) as actual
        FROM core.projects p
        WHERE p.deleted_at IS NULL
        HAVING p.element_count != (SELECT COALESCE(SUM(d.element_count), 0) FROM core.designs d WHERE d.project_id = p.id AND d.deleted_at IS NULL)
    ) discrepancies;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'Project element counts are consistent';
    ELSE
        status := 'FAIL';
        details := format('%s projects have inconsistent element counts', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查设计版本计数一致性
    check_name := 'design_version_count';
    SELECT COUNT(*) INTO v_count
    FROM (
        SELECT d.id, d.version_count as expected,
               (SELECT COUNT(*) FROM core.design_versions dv WHERE dv.design_id = d.id) as actual
        FROM core.designs d
        WHERE d.deleted_at IS NULL
        HAVING d.version_count != (SELECT COUNT(*) FROM core.design_versions dv WHERE dv.design_id = d.id)
    ) discrepancies;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'Design version counts are consistent';
    ELSE
        status := 'FAIL';
        details := format('%s designs have inconsistent version counts', v_count);
    END IF;
    RETURN NEXT;
    
    -- 检查租户存储使用一致性
    check_name := 'tenant_storage_usage';
    SELECT COUNT(*) INTO v_count
    FROM (
        SELECT t.id, t.storage_used_bytes as expected,
               COALESCE((SELECT SUM(d.file_size_bytes) FROM core.designs d 
                        JOIN core.projects p ON d.project_id = p.id 
                        WHERE p.tenant_id = t.id AND d.deleted_at IS NULL), 0) +
               COALESCE((SELECT SUM(dv.file_size_bytes) FROM core.design_versions dv 
                        JOIN core.designs d ON dv.design_id = d.id
                        JOIN core.projects p ON d.project_id = p.id
                        WHERE p.tenant_id = t.id), 0) as actual
        FROM core.tenants t
        HAVING ABS(t.storage_used_bytes - (
            COALESCE((SELECT SUM(d.file_size_bytes) FROM core.designs d 
                     JOIN core.projects p ON d.project_id = p.id 
                     WHERE p.tenant_id = t.id AND d.deleted_at IS NULL), 0) +
            COALESCE((SELECT SUM(dv.file_size_bytes) FROM core.design_versions dv 
                     JOIN core.designs d ON dv.design_id = d.id
                     JOIN core.projects p ON d.project_id = p.id
                     WHERE p.tenant_id = t.id), 0)
        )) > 1024 * 1024  -- 允许1MB误差
    ) discrepancies;
    
    IF v_count = 0 THEN
        status := 'PASS';
        details := 'Tenant storage usage is consistent';
    ELSE
        status := 'FAIL';
        details := format('%s tenants have inconsistent storage usage', v_count);
    END IF;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- 3. 备份验证函数
CREATE OR REPLACE FUNCTION core.validate_backup(
    p_backup_file_path TEXT
)
RETURNS TABLE (validation_step TEXT, status TEXT, details TEXT) AS $$
BEGIN
    -- 文件存在性检查
    validation_step := 'file_exists';
    status := 'INFO';
    details := format('Checking backup file: %s', p_backup_file_path);
    RETURN NEXT;
    
    -- 校验和验证
    validation_step := 'checksum';
    status := 'INFO';
    details := 'Checksum validation would be performed here';
    RETURN NEXT;
    
    -- 表数量检查
    validation_step := 'table_count';
    SELECT 
        CASE WHEN COUNT(*) > 0 THEN 'PASS' ELSE 'FAIL' END,
        format('Found %s tables in expected schemas', COUNT(*))
    INTO status, details
    FROM information_schema.tables
    WHERE table_schema IN ('core', 'geometry', 'versioning', 'audit');
    RETURN NEXT;
    
    -- 关键表数据检查
    validation_step := 'critical_tables';
    SELECT 
        CASE 
            WHEN (SELECT COUNT(*) FROM core.tenants) > 0 
                 AND (SELECT COUNT(*) FROM core.users) > 0 
            THEN 'PASS' 
            ELSE 'WARN' 
        END,
        format('Tenants: %s, Users: %s', 
               (SELECT COUNT(*) FROM core.tenants),
               (SELECT COUNT(*) FROM core.users))
    INTO status, details;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- 4. 运行所有验证
SELECT * FROM core.validate_data_integrity();
SELECT * FROM core.validate_data_consistency();
```

---

## 8. 数据库监控

### 8.1 监控指标定义

```sql
-- ============================================
-- 监控指标定义
-- ============================================

-- 创建监控指标表
CREATE TABLE IF NOT EXISTS monitoring.metrics (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name         VARCHAR(100) NOT NULL,
    metric_type         VARCHAR(50) NOT NULL  -- 'counter', 'gauge', 'histogram'
                        CHECK (metric_type IN ('counter', 'gauge', 'histogram')),
    metric_value        DECIMAL(18, 6) NOT NULL,
    metric_labels       JSONB DEFAULT '{}',
    collected_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建监控指标收集函数
CREATE OR REPLACE FUNCTION monitoring.collect_metrics()
RETURNS VOID AS $$
BEGIN
    -- 数据库连接数
    INSERT INTO monitoring.metrics (metric_name, metric_type, metric_value, metric_labels)
    SELECT 
        'db_connections_active',
        'gauge',
        COUNT(*)::DECIMAL,
        jsonb_build_object('state', state)
    FROM pg_stat_activity
    GROUP BY state;
    
    -- 表统计
    INSERT INTO monitoring.metrics (metric_name, metric_type, metric_value, metric_labels)
    SELECT 
        'table_row_count',
        'gauge',
        reltuples::DECIMAL,
        jsonb_build_object('schema', schemaname, 'table', relname)
    FROM pg_stat_user_tables
    WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit');
    
    -- 表大小
    INSERT INTO monitoring.metrics (metric_name, metric_type, metric_value, metric_labels)
    SELECT 
        'table_size_bytes',
        'gauge',
        pg_total_relation_size(schemaname || '.' || relname)::DECIMAL,
        jsonb_build_object('schema', schemaname, 'table', relname)
    FROM pg_stat_user_tables
    WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit');
    
    -- 索引使用情况
    INSERT INTO monitoring.metrics (metric_name, metric_type, metric_value, metric_labels)
    SELECT 
        'index_scan_count',
        'counter',
        idx_scan::DECIMAL,
        jsonb_build_object('schema', schemaname, 'index', indexrelname)
    FROM pg_stat_user_indexes
    WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit');
    
    -- 清理旧指标（保留24小时）
    DELETE FROM monitoring.metrics WHERE collected_at < NOW() - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;

-- 创建监控视图
CREATE OR REPLACE VIEW monitoring.v_current_metrics AS
SELECT 
    metric_name,
    metric_type,
    metric_value,
    metric_labels,
    collected_at
FROM monitoring.metrics
WHERE collected_at > NOW() - INTERVAL '5 minutes'
ORDER BY collected_at DESC;

-- 数据库健康检查视图
CREATE OR REPLACE VIEW monitoring.v_database_health AS
SELECT 
    'connection_count' as metric,
    (SELECT COUNT(*) FROM pg_stat_activity) as current_value,
    (SELECT setting::INTEGER FROM pg_settings WHERE name = 'max_connections') as max_value,
    ROUND((SELECT COUNT(*)::DECIMAL / NULLIF((SELECT setting::INTEGER FROM pg_settings WHERE name = 'max_connections'), 0) * 100), 2) as usage_percent,
    CASE 
        WHEN (SELECT COUNT(*)::DECIMAL / NULLIF((SELECT setting::INTEGER FROM pg_settings WHERE name = 'max_connections'), 0)) > 0.8 THEN 'WARNING'
        WHEN (SELECT COUNT(*)::DECIMAL / NULLIF((SELECT setting::INTEGER FROM pg_settings WHERE name = 'max_connections'), 0)) > 0.9 THEN 'CRITICAL'
        ELSE 'OK'
    END as status

UNION ALL

SELECT 
    'database_size' as metric,
    (SELECT pg_database_size(current_database())) as current_value,
    NULL as max_value,
    NULL as usage_percent,
    'INFO' as status

UNION ALL

SELECT 
    'active_transactions' as metric,
    (SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active') as current_value,
    NULL as max_value,
    NULL as usage_percent,
    'INFO' as status

UNION ALL

SELECT 
    'idle_transactions' as metric,
    (SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'idle in transaction') as current_value,
    NULL as max_value,
    NULL as usage_percent,
    CASE 
        WHEN (SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'idle in transaction') > 10 THEN 'WARNING'
        ELSE 'OK'
    END as status;
```

### 8.2 告警规则配置

```sql
-- ============================================
-- 告警规则配置
-- ============================================

-- 创建告警规则表
CREATE TABLE IF NOT EXISTS monitoring.alert_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name           VARCHAR(100) NOT NULL UNIQUE,
    description         TEXT,
    metric_name         VARCHAR(100) NOT NULL,
    condition_type      VARCHAR(50) NOT NULL  -- 'gt', 'lt', 'eq', 'between'
                        CHECK (condition_type IN ('gt', 'lt', 'eq', 'between', 'contains')),
    threshold_value     DECIMAL(18, 6) NOT NULL,
    threshold_value_2   DECIMAL(18, 6),  -- 用于between条件
    duration_minutes    INTEGER DEFAULT 5,
    severity            VARCHAR(20) NOT NULL DEFAULT 'warning'
                        CHECK (severity IN ('info', 'warning', 'critical')),
    is_enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    notification_channels TEXT[] DEFAULT ARRAY['email'],
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建告警历史表
CREATE TABLE IF NOT EXISTS monitoring.alert_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id             UUID NOT NULL REFERENCES monitoring.alert_rules(id),
    alert_name          VARCHAR(100) NOT NULL,
    severity            VARCHAR(20) NOT NULL,
    metric_value        DECIMAL(18, 6),
    message             TEXT,
    is_resolved         BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 插入默认告警规则
INSERT INTO monitoring.alert_rules (rule_name, description, metric_name, condition_type, threshold_value, severity) VALUES
('high_connection_count', '数据库连接数过高', 'db_connections_active', 'gt', 80, 'warning'),
('critical_connection_count', '数据库连接数严重过高', 'db_connections_active', 'gt', 90, 'critical'),
('large_table_size', '表大小超过1GB', 'table_size_bytes', 'gt', 1073741824, 'warning'),
('unused_index', '索引未被使用', 'index_scan_count', 'eq', 0, 'info'),
('slow_query', '慢查询数量', 'slow_query_count', 'gt', 10, 'warning');

-- 告警检查函数
CREATE OR REPLACE FUNCTION monitoring.check_alerts()
RETURNS TABLE (alert_triggered BOOLEAN, alert_name VARCHAR(100), severity VARCHAR(20), message TEXT) AS $$
DECLARE
    v_rule RECORD;
    v_metric_value DECIMAL;
    v_alert_triggered BOOLEAN;
BEGIN
    FOR v_rule IN SELECT * FROM monitoring.alert_rules WHERE is_enabled = TRUE
    LOOP
        v_alert_triggered := FALSE;
        
        -- 获取当前指标值
        SELECT metric_value INTO v_metric_value
        FROM monitoring.metrics
        WHERE metric_name = v_rule.metric_name
        ORDER BY collected_at DESC
        LIMIT 1;
        
        -- 检查条件
        CASE v_rule.condition_type
            WHEN 'gt' THEN
                IF v_metric_value > v_rule.threshold_value THEN
                    v_alert_triggered := TRUE;
                END IF;
            WHEN 'lt' THEN
                IF v_metric_value < v_rule.threshold_value THEN
                    v_alert_triggered := TRUE;
                END IF;
            WHEN 'eq' THEN
                IF v_metric_value = v_rule.threshold_value THEN
                    v_alert_triggered := TRUE;
                END IF;
            WHEN 'between' THEN
                IF v_metric_value BETWEEN v_rule.threshold_value AND v_rule.threshold_value_2 THEN
                    v_alert_triggered := TRUE;
                END IF;
        END CASE;
        
        -- 记录告警
        IF v_alert_triggered THEN
            INSERT INTO monitoring.alert_history (rule_id, alert_name, severity, metric_value, message)
            VALUES (
                v_rule.id,
                v_rule.rule_name,
                v_rule.severity,
                v_metric_value,
                format('Alert: %s (value: %s, threshold: %s)', 
                       v_rule.description, v_metric_value, v_rule.threshold_value)
            );
            
            alert_triggered := TRUE;
            alert_name := v_rule.rule_name;
            severity := v_rule.severity;
            message := format('%s: Current value %s exceeds threshold %s', 
                            v_rule.description, v_metric_value, v_rule.threshold_value);
            RETURN NEXT;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 告警视图
CREATE OR REPLACE VIEW monitoring.v_active_alerts AS
SELECT 
    ah.id,
    ah.alert_name,
    ah.severity,
    ah.metric_value,
    ah.message,
    ah.created_at,
    EXTRACT(EPOCH FROM (NOW() - ah.created_at)) / 60 as duration_minutes
FROM monitoring.alert_history ah
WHERE ah.is_resolved = FALSE
ORDER BY 
    CASE ah.severity 
        WHEN 'critical' THEN 1 
        WHEN 'warning' THEN 2 
        ELSE 3 
    END,
    ah.created_at DESC;
```

### 8.3 性能分析查询

```sql
-- ============================================
-- 性能分析查询
-- ============================================

-- 1. 慢查询分析
CREATE OR REPLACE VIEW monitoring.v_slow_queries AS
SELECT 
    queryid,
    query,
    calls,
    ROUND(total_exec_time::NUMERIC, 2) as total_time_ms,
    ROUND(mean_exec_time::NUMERIC, 2) as avg_time_ms,
    ROUND(max_exec_time::NUMERIC, 2) as max_time_ms,
    ROUND(min_exec_time::NUMERIC, 2) as min_time_ms,
    rows,
    ROUND((100 * total_exec_time / sum(total_exec_time) OVER ())::NUMERIC, 2) as percent_time
FROM pg_stat_statements
WHERE query NOT LIKE '%pg_stat%'
ORDER BY total_exec_time DESC
LIMIT 50;

-- 2. 表扫描统计
CREATE OR REPLACE VIEW monitoring.v_table_scans AS
SELECT 
    schemaname,
    relname as table_name,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    n_live_tup,
    n_dead_tup,
    CASE 
        WHEN seq_scan > 0 AND idx_scan > 0 THEN 
            ROUND((seq_scan::NUMERIC / (seq_scan + idx_scan) * 100), 2)
        WHEN seq_scan > 0 THEN 100
        ELSE 0
    END as seq_scan_percent,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit')
ORDER BY seq_scan DESC;

-- 3. 锁等待分析
CREATE OR REPLACE VIEW monitoring.v_lock_waits AS
SELECT 
    blocked_locks.pid AS blocked_pid,
    blocked_activity.usename AS blocked_user,
    blocking_locks.pid AS blocking_pid,
    blocking_activity.usename AS blocking_user,
    blocked_activity.query AS blocked_statement,
    blocking_activity.query AS blocking_statement,
    blocked_activity.application_name AS blocked_application,
    blocking_activity.application_name AS blocking_application,
    blocked_activity.wait_event_type AS blocked_wait_event_type,
    blocked_activity.wait_event AS blocked_wait_event,
    NOW() - blocked_activity.query_start AS blocked_duration
FROM pg_catalog.pg_locks blocked_locks
JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
    AND blocking_locks.relation = blocked_locks.relation
    AND blocking_locks.pid != blocked_locks.pid
JOIN pg_catalog.pg_stat_activity blocking_activity ON blocking_activity.pid = blocking_locks.pid
WHERE NOT blocked_locks.granted;

-- 4. 索引效率分析
CREATE OR REPLACE VIEW monitoring.v_index_efficiency AS
SELECT 
    schemaname,
    relname as table_name,
    indexrelname as index_name,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
    CASE 
        WHEN idx_scan > 0 THEN ROUND((idx_tup_read::NUMERIC / idx_scan), 2)
        ELSE 0
    END as avg_tuples_per_scan,
    CASE 
        WHEN idx_scan = 0 THEN 'UNUSED'
        WHEN idx_scan < 100 THEN 'LOW_USAGE'
        ELSE 'ACTIVE'
    END as usage_status
FROM pg_stat_user_indexes
WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit')
ORDER BY idx_scan DESC;

-- 5. 缓存命中率分析
CREATE OR REPLACE VIEW monitoring.v_cache_hit_ratio AS
SELECT 
    schemaname,
    relname as table_name,
    heap_blks_read,
    heap_blks_hit,
    CASE 
        WHEN heap_blks_hit + heap_blks_read > 0 THEN
            ROUND((heap_blks_hit::NUMERIC / (heap_blks_hit + heap_blks_read) * 100), 2)
        ELSE 0
    END as cache_hit_ratio,
    CASE 
        WHEN heap_blks_hit + heap_blks_read > 0 AND 
             (heap_blks_hit::NUMERIC / (heap_blks_hit + heap_blks_read)) < 0.95 THEN 'LOW'
        ELSE 'OK'
    END as status
FROM pg_statio_user_tables
WHERE schemaname IN ('core', 'geometry', 'versioning', 'audit')
ORDER BY heap_blks_read DESC;

-- 6. 连接池监控
CREATE OR REPLACE VIEW monitoring.v_connection_pool AS
SELECT 
    datname as database,
    usename as username,
    application_name,
    client_addr,
    state,
    COUNT(*) as connection_count,
    AVG(EXTRACT(EPOCH FROM (NOW() - backend_start)))::INTEGER as avg_connection_age_seconds,
    MAX(EXTRACT(EPOCH FROM (NOW() - query_start)))::INTEGER as max_query_duration_seconds
FROM pg_stat_activity
WHERE datname IS NOT NULL
GROUP BY datname, usename, application_name, client_addr, state
ORDER BY connection_count DESC;

-- 7. 数据库大小趋势（需要定期收集）
CREATE TABLE IF NOT EXISTS monitoring.db_size_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_name       VARCHAR(100) NOT NULL,
    size_bytes          BIGINT NOT NULL,
    collected_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION monitoring.collect_db_size()
RETURNS VOID AS $$
BEGIN
    INSERT INTO monitoring.db_size_history (database_name, size_bytes)
    SELECT datname, pg_database_size(datname)
    FROM pg_database
    WHERE datname NOT IN ('template0', 'template1');
    
    -- 清理旧数据（保留90天）
    DELETE FROM monitoring.db_size_history WHERE collected_at < NOW() - INTERVAL '90 days';
END;
$$ LANGUAGE plpgsql;

-- 8. 性能报告生成函数
CREATE OR REPLACE FUNCTION monitoring.generate_performance_report(
    p_start_date TIMESTAMPTZ DEFAULT NOW() - INTERVAL '24 hours',
    p_end_date TIMESTAMPTZ DEFAULT NOW()
)
RETURNS JSONB AS $$
DECLARE
    v_report JSONB;
BEGIN
    SELECT jsonb_build_object(
        'report_period', jsonb_build_object(
            'start', p_start_date,
            'end', p_end_date
        ),
        'database_health', (
            SELECT jsonb_agg(row_to_json(t))
            FROM monitoring.v_database_health t
        ),
        'slow_queries', (
            SELECT jsonb_agg(row_to_json(t))
            FROM monitoring.v_slow_queries t
            LIMIT 10
        ),
        'table_scans', (
            SELECT jsonb_agg(row_to_json(t))
            FROM monitoring.v_table_scans t
            LIMIT 10
        ),
        'cache_hit_ratio', (
            SELECT jsonb_agg(row_to_json(t))
            FROM monitoring.v_cache_hit_ratio t
            WHERE status = 'LOW'
        ),
        'active_alerts', (
            SELECT jsonb_agg(row_to_json(t))
            FROM monitoring.v_active_alerts t
        ),
        'generated_at', NOW()
    ) INTO v_report;
    
    RETURN v_report;
END;
$$ LANGUAGE plpgsql;
```

---

## 附录

### A. 数据库ER图描述

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              半自动化建筑设计平台 - 数据库ER图                              │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐                        │
│  │  tenants    │◀────────│   users     │◀────────│   teams     │                        │
│  │  (租户)      │         │  (用户)     │         │  (团队)     │                        │
│  └──────┬──────┘         └──────┬──────┘         └─────────────┘                        │
│         │                       │                                                       │
│         │                       │ owns                                                  │
│         │                       ▼                                                       │
│         │              ┌─────────────────┐                                              │
│         │              │  user_roles     │                                              │
│         │              │  (用户角色)      │                                              │
│         │              └─────────────────┘                                              │
│         │                       │                                                       │
│         │                       ▼                                                       │
│         │              ┌─────────────────┐                                              │
│         │              │     roles       │                                              │
│         │              │    (角色)       │                                              │
│         │              └─────────────────┘                                              │
│         │                                                                               │
│         │                       │                                                       │
│         │                       ▼                                                       │
│         │              ┌─────────────────┐                                              │
│         │              │  permissions    │                                              │
│         │              │   (权限)        │                                              │
│         │              └─────────────────┘                                              │
│         │                                                                               │
│         │                       │                                                       │
│         │                       ▼                                                       │
│         │              ┌─────────────────┐                                              │
│         │              │resource_perms   │                                              │
│         │              │ (资源权限)      │                                              │
│         │              └─────────────────┘                                              │
│         │                                                                               │
│         ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │                         projects                            │                        │
│  │                         (项目)                              │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│         │                              ▲                                                │
│         │ contains                     │                                                │
│         ▼                              │                                                │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │                         designs                             │                        │
│  │                         (设计)                              │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│         │                              ▲                                                │
│         │ has versions                 │                                                │
│         ▼                              │                                                │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │                     design_versions                         │                        │
│  │                       (设计版本)                             │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│         │                              ▲                                                │
│         │ contains                     │                                                │
│         ▼                              │                                                │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │                         layers                              │                        │
│  │                         (图层)                              │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│         │                              ▲                                                │
│         │ contains                     │                                                │
│         ▼                              │                                                │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │                        elements                             │                        │
│  │                        (元素)                               │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│         │                              ▲                                                │
│         │ has geometry               │                                                  │
│         ▼                              │                                                │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │  ┌─────────────────────────────────────────────────────┐   │                        │
│  │  │              geometry.geometries                    │   │                        │
│  │  │                 (几何数据)                          │   │                        │
│  │  └─────────────────────────────────────────────────────┘   │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │  ┌─────────────────────────────────────────────────────┐   │                        │
│  │  │              versioning.events                      │   │                        │
│  │  │                 (事件存储)                          │   │                        │
│  │  └─────────────────────────────────────────────────────┘   │                        │
│  │  ┌─────────────────────────────────────────────────────┐   │                        │
│  │  │              versioning.snapshots                   │   │                        │
│  │  │                 (快照存储)                          │   │                        │
│  │  └─────────────────────────────────────────────────────┘   │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────┐                        │
│  │  ┌─────────────────────────────────────────────────────┐   │                        │
│  │  │              audit.audit_logs                       │   │                        │
│  │  │                 (审计日志)                          │   │                        │
│  │  └─────────────────────────────────────────────────────┘   │                        │
│  │  ┌─────────────────────────────────────────────────────┐   │                        │
│  │  │              audit.login_history                    │   │                        │
│  │  │                 (登录历史)                          │   │                        │
│  │  └─────────────────────────────────────────────────────┘   │                        │
│  └─────────────────────────────────────────────────────────────┘                        │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

### B. 表结构汇总

| Schema | 表名 | 用途 | 主要索引 |
|--------|------|------|----------|
| core | tenants | 租户信息 | id (PK), slug (UK) |
| core | users | 用户信息 | id (PK), tenant_id+email (UK) |
| core | teams | 团队信息 | id (PK), tenant_id+name (UK) |
| core | team_members | 团队成员 | id (PK), team_id+user_id (UK) |
| core | projects | 项目信息 | id (PK), tenant_id+project_code (UK) |
| core | project_members | 项目成员 | id (PK), project_id+user_id (UK) |
| core | designs | 设计文档 | id (PK), project_id (IDX) |
| core | design_versions | 设计版本 | id (PK), design_id+version_number (UK) |
| core | layers | 图层信息 | id (PK), design_id+name (UK) |
| core | elements | 建筑元素 | id (PK), design_id (IDX), bbox (GIST) |
| core | permissions | 权限定义 | id (PK), code (UK) |
| core | roles | 角色定义 | id (PK), tenant_id+name (UK) |
| core | user_roles | 用户角色 | id (PK), user_id+role_id+scope (UK) |
| core | resource_permissions | 资源权限 | id (PK), resource+perm+principal (UK) |
| core | api_keys | API密钥 | id (PK), key_prefix (IDX) |
| core | user_sessions | 用户会话 | id (PK), token (IDX), expires_at (IDX) |
| geometry | geometries | 几何对象 | id (PK), geom_2d (GIST), bbox (GIST) |
| geometry | geometry_snapshots | 几何快照 | id (PK), snapshot_id (IDX) |
| geometry | spatial_index | 空间索引 | id (PK), grid (IDX), bounds (IDX) |
| geometry | spatial_relations | 空间关系 | id (PK), source+target+type (UK) |
| versioning | events | 事件存储 | id (PK), aggregate+seq (UK), global_seq (IDX) |
| versioning | snapshots | 快照存储 | id (PK), aggregate+version (UK) |
| versioning | change_sets | 变更集 | id (PK), project_id (IDX) |
| versioning | operation_history | 操作历史 | id (PK), project_id (IDX) |
| audit | audit_logs | 审计日志 | id+created_at (PK), tenant_id (IDX), created_at (BRIN) |
| audit | login_history | 登录历史 | id (PK), user_id (IDX), created_at (BRIN) |
| audit | data_access_log | 数据访问 | id (PK), user_id (IDX), created_at (IDX) |

---

**文档版本**: 1.0  
**最后更新**: 2024年  
**作者**: 数据库架构团队
