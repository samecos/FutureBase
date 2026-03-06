# 可行性验证阶段 - 数据存储POC验证报告

## 半自动化建筑设计平台

---

**文档版本**: v1.0  
**编制日期**: 2024年  
**文档状态**: 可行性验证阶段  

---

## 目录

1. [概述](#1-概述)
2. [数据模型POC](#2-数据模型poc)
3. [版本控制POC](#3-版本控制poc)
4. [并发控制POC](#4-并发控制poc)
5. [性能测试方案](#5-性能测试方案)
6. [多租户隔离验证](#6-多租户隔离验证)
7. [POC执行计划](#7-poc执行计划)
8. [风险评估与建议](#8-风险评估与建议)

---

## 1. 概述

### 1.1 技术栈选型

| 组件 | 推荐技术 | 备选方案 | 选型理由 |
|------|----------|----------|----------|
| 主数据库 | **YugabyteDB** | CockroachDB | 分布式PostgreSQL兼容，内置PostGIS支持，全球分布式部署 |
| 几何存储 | **PostGIS 3.x** | - | 行业标准，支持3D几何、空间索引、复杂空间运算 |
| 版本控制 | **Event Sourcing** | Git LFS | 完整历史追溯，支持时间旅行查询 |
| 缓存 | **Redis Cluster** | Memcached | 高性能缓存，支持发布订阅、分布式锁 |
| 对象存储 | **MinIO/S3** | - | 大文件存储（模型文件、渲染结果） |

### 1.2 POC验证目标

```
┌─────────────────────────────────────────────────────────────────┐
│                    POC验证目标矩阵                               │
├─────────────────┬───────────────────────────────────────────────┤
│ 功能性验证      │ ✓ 数据模型完整性                             │
│                 │ ✓ 版本控制准确性                             │
│                 │ ✓ 并发控制正确性                             │
│                 │ ✓ 多租户隔离有效性                           │
├─────────────────┼───────────────────────────────────────────────┤
│ 性能验证        │ ✓ 几何查询 < 100ms (95th percentile)         │
│                 │ ✓ 版本历史查询 < 200ms                       │
│                 │ ✓ 并发写入 > 1000 TPS                        │
│                 │ ✓ 大数据量支持 100万+ 构件                   │
├─────────────────┼───────────────────────────────────────────────┤
│ 可靠性验证      │ ✓ 数据一致性保证                             │
│                 │ ✓ 故障恢复能力                               │
│                 │ ✓ 数据备份/恢复                              │
└─────────────────┴───────────────────────────────────────────────┘
```

---

## 2. 数据模型POC

### 2.1 核心实体ER图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           核心实体关系图                                      │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────┐         ┌──────────────┐         ┌──────────────┐
    │   tenants    │         │   projects   │         │   users      │
    ├──────────────┤         ├──────────────┤         ├──────────────┤
    │ tenant_id(PK)│◄────────│ project_id   │◄────────│ user_id(PK)  │
    │ name         │    1:N  │ tenant_id(FK)│    N:M  │ email        │
    │ config       │         │ name         │         │ profile      │
    │ created_at   │         │ created_by   │         │ created_at   │
    └──────────────┘         │ settings     │         └──────────────┘
                             │ version_seq  │                ▲
                             └──────────────┘                │
                                      ▲                       │
                                      │ 1:N                   │ N:M
                                      │                       │
                             ┌────────┴───────┐      ┌────────┴───────┐
                             │    elements    │      │ project_members│
                             ├────────────────┤      ├────────────────┤
                             │ element_id(PK) │      │ member_id(PK)  │
                             │ project_id(FK) │      │ project_id(FK) │
                             │ element_type   │      │ user_id(FK)    │
                             │ geometry(3D)   │      │ role           │
                             │ properties     │      │ permissions    │
                             │ current_ver    │      └────────────────┘
                             └───────┬────────┘
                                     │
                                     │ 1:N
                                     ▼
                             ┌────────┴────────┐
                             │ element_versions│
                             ├─────────────────┤
                             │ version_id(PK)  │
                             │ element_id(FK)  │
                             │ version_number  │
                             │ geometry(3D)    │
                             │ properties      │
                             │ event_id(FK)    │
                             │ created_at      │
                             │ created_by      │
                             └─────────────────┘

    ┌──────────────┐         ┌──────────────┐         ┌──────────────┐
    │    events    │         │  snapshots   │         │   branches   │
    ├──────────────┤         ├──────────────┤         ├──────────────┤
    │ event_id(PK) │◄────────│ snapshot_id  │         │ branch_id(PK)│
    │ project_id   │         │ project_id   │◄────────│ project_id   │
    │ event_type   │         │ branch_id    │    N:1  │ name         │
    │ payload      │         │ version_seq  │         │ base_version │
    │ timestamp    │         │ data         │         │ head_version │
    │ user_id      │         │ created_at   │         │ is_default   │
    │ version_seq  │         └──────────────┘         └──────────────┘
    └──────────────┘
```

### 2.2 核心表结构设计

#### 2.2.1 租户表 (tenants)

```sql
-- ============================================
-- 租户表 - 多租户隔离的基础
-- ============================================
CREATE TABLE tenants (
    tenant_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(100) UNIQUE NOT NULL,  -- URL友好的标识
    
    -- 租户配置 (JSONB存储灵活配置)
    config          JSONB DEFAULT '{
        "max_projects": 100,
        "max_storage_gb": 100,
        "max_users": 50,
        "features": ["basic", "collaboration"]
    }'::jsonb,
    
    -- 配额和使用统计
    quota           JSONB DEFAULT '{}'::jsonb,
    usage_stats     JSONB DEFAULT '{}'::jsonb,
    
    -- 状态管理
    status          VARCHAR(20) DEFAULT 'active' 
                    CHECK (status IN ('active', 'suspended', 'deleted')),
    
    -- 时间戳
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,  -- 软删除
    
    -- 约束
    CONSTRAINT valid_tenant_slug CHECK (slug ~ '^[a-z0-9-]+$')
);

-- 索引
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
```

#### 2.2.2 项目表 (projects)

```sql
-- ============================================
-- 项目表 - 建筑设计项目主表
-- ============================================
CREATE TABLE projects (
    project_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- 基本信息
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    
    -- 项目设置
    settings        JSONB DEFAULT '{
        "units": "metric",
        "coordinate_system": "WGS84",
        "precision": 0.001,
        "auto_save_interval": 300
    }'::jsonb,
    
    -- 版本控制
    version_seq     BIGINT DEFAULT 0,  -- 全局事件序列号
    current_branch  UUID,              -- 当前分支
    
    -- 统计信息
    stats           JSONB DEFAULT '{
        "element_count": 0,
        "total_versions": 0,
        "last_activity": null
    }'::jsonb,
    
    -- 创建者
    created_by      UUID NOT NULL,
    
    -- 时间戳
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    
    -- 约束
    CONSTRAINT unique_project_name_per_tenant 
        UNIQUE (tenant_id, name) 
        WHERE deleted_at IS NULL
);

-- 索引
CREATE INDEX idx_projects_tenant ON projects(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_created_by ON projects(created_by);
CREATE INDEX idx_projects_version_seq ON projects(project_id, version_seq);

-- GIN索引用于JSONB查询
CREATE INDEX idx_projects_settings_gin ON projects USING GIN(settings);
CREATE INDEX idx_projects_stats_gin ON projects USING GIN(stats);
```

#### 2.2.3 构件表 (elements) - 含PostGIS几何数据

```sql
-- ============================================
-- 构件表 - 建筑设计元素（墙、门、窗等）
-- ============================================
CREATE TABLE elements (
    element_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(project_id),
    
    -- 构件分类
    element_type    VARCHAR(50) NOT NULL 
                    CHECK (element_type IN (
                        'wall', 'door', 'window', 'floor', 'roof',
                        'column', 'beam', 'stair', 'railing', 'furniture',
                        'equipment', 'annotation', 'group'
                    )),
    
    -- 构件子类型
    element_subtype VARCHAR(50),
    
    -- 分类编码 (支持多种分类体系)
    classification  JSONB DEFAULT '{}'::jsonb,  -- { "uniclass": "...", "omniclass": "..." }
    
    -- 当前版本引用
    current_version UUID,
    version_count   INT DEFAULT 1,
    
    -- 空间关系
    parent_id       UUID REFERENCES elements(element_id),
    level_id        UUID,  -- 楼层引用
    
    -- 空间索引边界框 (用于快速空间查询)
    bbox_2d         GEOMETRY(POLYGON, 3857),  -- Web Mercator投影
    bbox_3d         GEOMETRY(POLYHEDRALSURFACEZ, 3857),
    
    -- 可见性控制
    is_visible      BOOLEAN DEFAULT TRUE,
    is_locked       BOOLEAN DEFAULT FALSE,
    
    -- 元数据
    metadata        JSONB DEFAULT '{}'::jsonb,
    
    -- 创建信息
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    
    -- 乐观锁版本
    lock_version    INT DEFAULT 1
);

-- PostGIS空间索引
CREATE INDEX idx_elements_bbox_2d ON elements USING GIST(bbox_2d);
CREATE INDEX idx_elements_bbox_3d ON elements USING GIST(bbox_3d);

-- 常规索引
CREATE INDEX idx_elements_project ON elements(project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_elements_type ON elements(element_type);
CREATE INDEX idx_elements_parent ON elements(parent_id);
CREATE INDEX idx_elements_level ON elements(level_id);
CREATE INDEX idx_elements_current_ver ON elements(current_version);

-- GIN索引
CREATE INDEX idx_elements_classification ON elements USING GIN(classification);
CREATE INDEX idx_elements_metadata ON elements USING GIN(metadata);

-- 复合索引
CREATE INDEX idx_elements_project_type ON elements(project_id, element_type) 
    WHERE deleted_at IS NULL;
```

#### 2.2.4 构件版本表 (element_versions)

```sql
-- ============================================
-- 构件版本表 - 存储每个版本的完整数据
-- ============================================
CREATE TABLE element_versions (
    version_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    element_id      UUID NOT NULL REFERENCES elements(element_id),
    
    -- 版本信息
    version_number  INT NOT NULL,
    version_type    VARCHAR(20) DEFAULT 'major' 
                    CHECK (version_type IN ('major', 'minor', 'auto')),
    
    -- 几何数据 (PostGIS)
    geometry_2d     GEOMETRY(GEOMETRY, 3857),  -- 2D几何
    geometry_3d     GEOMETRY(GEOMETRYZ, 3857), -- 3D几何 (含Z坐标)
    
    -- 几何哈希 (用于快速比较)
    geometry_hash   VARCHAR(64),
    
    -- 属性数据
    properties      JSONB NOT NULL DEFAULT '{}'::jsonb,
    properties_hash VARCHAR(64),
    
    -- 完整属性快照 (用于快速恢复)
    full_snapshot   JSONB,
    
    -- 关联事件
    event_id        UUID NOT NULL,
    
    -- 变更摘要
    change_summary  JSONB,  -- { "changed_fields": [...], "change_type": "..." }
    
    -- 创建信息
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    
    -- 约束
    CONSTRAINT unique_element_version UNIQUE (element_id, version_number)
);

-- 索引
CREATE INDEX idx_elem_ver_element ON element_versions(element_id);
CREATE INDEX idx_elem_ver_event ON element_versions(event_id);
CREATE INDEX idx_elem_ver_created ON element_versions(created_at);
CREATE INDEX idx_elem_ver_geom_hash ON element_versions(geometry_hash);
CREATE INDEX idx_elem_ver_props_hash ON element_versions(properties_hash);

-- 空间索引
CREATE INDEX idx_elem_ver_geom_2d ON element_versions USING GIST(geometry_2d);
CREATE INDEX idx_elem_ver_geom_3d ON element_versions USING GIST(geometry_3d);

-- GIN索引
CREATE INDEX idx_elem_ver_properties ON element_versions USING GIN(properties);
CREATE INDEX idx_elem_ver_change_summary ON element_versions USING GIN(change_summary);

-- 分区策略 (按时间范围分区，提高历史查询性能)
-- CREATE TABLE element_versions_partitioned (LIKE element_versions) 
-- PARTITION BY RANGE (created_at);
```

#### 2.2.5 事件表 (events) - Event Sourcing核心

```sql
-- ============================================
-- 事件表 - Event Sourcing存储
-- ============================================
CREATE TABLE events (
    -- 事件ID (全局唯一)
    event_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 项目上下文
    project_id      UUID NOT NULL REFERENCES projects(project_id),
    tenant_id       UUID NOT NULL,  -- 冗余存储，便于租户级查询
    
    -- 事件序列号 (项目内单调递增)
    version_seq     BIGINT NOT NULL,
    
    -- 事件类型
    event_type      VARCHAR(50) NOT NULL 
                    CHECK (event_type IN (
                        -- 元素操作
                        'element.created', 'element.updated', 'element.deleted',
                        'element.restored', 'element.copied', 'element.moved',
                        -- 属性操作
                        'property.changed', 'property.added', 'property.removed',
                        -- 几何操作
                        'geometry.changed', 'transform.applied',
                        -- 项目操作
                        'project.created', 'project.updated', 'project.deleted',
                        'project.snapshot', 'project.branch_created',
                        'project.branch_merged', 'project.branch_switched',
                        -- 批量操作
                        'batch.created', 'batch.updated', 'batch.deleted'
                    )),
    
    -- 事件载荷 (核心数据)
    payload         JSONB NOT NULL,
    
    -- 事件元数据
    metadata        JSONB DEFAULT '{}'::jsonb,  -- { "client_version": "...", "source": "..." }
    
    -- 影响范围
    affected_elements UUID[] DEFAULT '{}',  -- 受影响的元素ID列表
    
    -- 因果关系
    correlation_id  UUID,  -- 关联ID (用于追踪请求链)
    causation_id    UUID,  -- 因果关系ID (导致此事件的事件)
    
    -- 执行信息
    executed_by     UUID NOT NULL,
    executed_at     TIMESTAMPTZ DEFAULT NOW(),
    
    -- 事务信息 (用于幂等性)
    transaction_id  UUID,
    
    -- 约束
    CONSTRAINT unique_project_version_seq UNIQUE (project_id, version_seq)
);

-- 核心索引
CREATE INDEX idx_events_project_seq ON events(project_id, version_seq);
CREATE INDEX idx_events_project_type ON events(project_id, event_type);
CREATE INDEX idx_events_tenant ON events(tenant_id);
CREATE INDEX idx_events_executed_at ON events(executed_at);
CREATE INDEX idx_events_correlation ON events(correlation_id);
CREATE INDEX idx_events_transaction ON events(transaction_id);

-- GIN索引 (用于JSONB查询)
CREATE INDEX idx_events_payload ON events USING GIN(payload jsonb_path_ops);
CREATE INDEX idx_events_metadata ON events USING GIN(metadata);
CREATE INDEX idx_events_affected ON events USING GIN(affected_elements);

-- 复合索引 (常用查询模式)
CREATE INDEX idx_events_project_time ON events(project_id, executed_at);
CREATE INDEX idx_events_project_type_time ON events(project_id, event_type, executed_at);

-- 分区策略建议 (按project_id哈希分区，提高并发写入)
-- 或按时间分区，便于历史数据归档
```

#### 2.2.6 快照表 (snapshots)

```sql
-- ============================================
-- 快照表 - 项目状态快照 (加速重建)
-- ============================================
CREATE TABLE snapshots (
    snapshot_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(project_id),
    
    -- 快照类型
    snapshot_type   VARCHAR(20) DEFAULT 'full' 
                    CHECK (snapshot_type IN ('full', 'incremental', 'delta')),
    
    -- 版本范围
    version_from    BIGINT NOT NULL,  -- 起始事件序列号
    version_to      BIGINT NOT NULL,  -- 结束事件序列号
    
    -- 快照数据
    -- 方案1: 存储在表中 (适合小项目)
    snapshot_data   JSONB,
    
    -- 方案2: 存储在外部对象存储 (适合大项目)
    storage_path    VARCHAR(500),     -- S3/MinIO路径
    storage_size    BIGINT,           -- 文件大小(字节)
    checksum        VARCHAR(64),      -- 数据校验和
    
    -- 包含的元素统计
    element_count   INT,
    element_ids     UUID[],           -- 包含的元素ID列表
    
    -- 创建信息
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    
    -- 访问统计
    access_count    INT DEFAULT 0,
    last_accessed   TIMESTAMPTZ
);

-- 索引
CREATE INDEX idx_snapshots_project ON snapshots(project_id);
CREATE INDEX idx_snapshots_version_range ON snapshots(project_id, version_from, version_to);
CREATE INDEX idx_snapshots_type ON snapshots(snapshot_type);
CREATE INDEX idx_snapshots_created ON snapshots(created_at);
```

#### 2.2.7 分支表 (branches)

```sql
-- ============================================
-- 分支表 - 支持Git式分支管理
-- ============================================
CREATE TABLE branches (
    branch_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(project_id),
    
    -- 分支信息
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    
    -- 版本指针
    base_version    BIGINT NOT NULL,  -- 基于哪个版本创建
    head_version    BIGINT NOT NULL,  -- 当前最新版本
    
    -- 父分支 (用于追踪分支关系)
    parent_branch   UUID REFERENCES branches(branch_id),
    merged_from     UUID[],           -- 合并来源分支
    
    -- 状态
    is_default      BOOLEAN DEFAULT FALSE,
    is_protected    BOOLEAN DEFAULT FALSE,  -- 受保护分支
    status          VARCHAR(20) DEFAULT 'active' 
                    CHECK (status IN ('active', 'merged', 'deleted')),
    
    -- 创建信息
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_branches_project ON branches(project_id);
CREATE INDEX idx_branches_default ON branches(project_id, is_default) WHERE is_default = TRUE;
CREATE INDEX idx_branches_status ON branches(status);

-- 约束
CREATE UNIQUE INDEX idx_unique_default_branch ON branches(project_id) 
    WHERE is_default = TRUE;
```

### 2.3 PostGIS几何数据存储验证

#### 2.3.1 几何类型设计

```sql
-- ============================================
-- PostGIS几何类型验证
-- ============================================

-- 启用PostGIS扩展
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS postgis_raster;  -- 如需栅格支持

-- 验证几何类型支持
DO $$
DECLARE
    test_geom GEOMETRY;
BEGIN
    -- 2D点
    test_geom := ST_GeomFromText('POINT(0 0)', 3857);
    RAISE NOTICE '2D Point: %', ST_AsText(test_geom);
    
    -- 3D点
    test_geom := ST_GeomFromText('POINT Z(0 0 10)', 3857);
    RAISE NOTICE '3D Point: %', ST_AsText(test_geom);
    
    -- 3D线
    test_geom := ST_GeomFromText('LINESTRING Z(0 0 0, 10 0 5, 10 10 10)', 3857);
    RAISE NOTICE '3D Line: %', ST_AsText(test_geom);
    
    -- 3D面 (墙体示例)
    test_geom := ST_GeomFromText('
        POLYHEDRALSURFACE Z ((
            0 0 0, 10 0 0, 10 0 3, 0 0 3, 0 0 0
        ), (
            0 0 0, 0 0.2 0, 0 0.2 3, 0 0 3, 0 0 0
        ))
    ', 3857);
    RAISE NOTICE '3D Wall: %', ST_GeometryType(test_geom);
    
END $$;
```

#### 2.3.2 几何数据存储方案对比

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      几何数据存储方案对比                                     │
├──────────────────┬────────────────┬────────────────┬────────────────────────┤
│ 方案             │ 存储格式       │ 适用场景       │ 优缺点                 │
├──────────────────┼────────────────┼────────────────┼────────────────────────┤
│ PostGIS原生      │ GEOMETRY       │ 标准构件       │ ✓ 完整空间索引支持     │
│                  │                │                │ ✓ 空间运算效率高       │
│                  │                │                │ ✗ 复杂几何存储开销大   │
├──────────────────┼────────────────┼────────────────┼────────────────────────┤
│ WKB二进制        │ BYTEA          │ 复杂BIM模型    │ ✓ 存储紧凑             │
│                  │                │                │ ✗ 无法直接空间查询     │
│                  │                │                │ ✗ 需转换后运算         │
├──────────────────┼────────────────┼────────────────┼────────────────────────┤
│ 混合方案(推荐)   │ GEOMETRY+JSONB │ 大型项目       │ ✓ 简化几何存PostGIS    │
│                  │                │                │ ✓ 复杂几何存JSONB      │
│                  │                │                │ ✓ 灵活性与性能平衡     │
├──────────────────┼────────────────┼────────────────┼────────────────────────┤
│ 外部存储         │ S3+引用        │ 超大模型       │ ✓ 无数据库大小限制     │
│                  │                │                │ ✗ 需要额外IO           │
│                  │                │                │ ✗ 一致性管理复杂       │
└──────────────────┴────────────────┴────────────────┴────────────────────────┘
```

#### 2.3.3 混合存储方案实现

```sql
-- ============================================
-- 混合几何存储方案
-- ============================================

-- 扩展构件版本表支持混合存储
ALTER TABLE element_versions ADD COLUMN IF NOT EXISTS 
    geometry_complex JSONB;  -- 复杂几何数据(如BIM实体)

-- 几何大小阈值(超过则使用JSONB存储简化版)
CREATE OR REPLACE FUNCTION get_geometry_storage_threshold() 
RETURNS INT AS $$
BEGIN
    RETURN 10000;  -- 10000个顶点阈值
END;
$$ LANGUAGE plpgsql;

-- 智能几何存储函数
CREATE OR REPLACE FUNCTION store_element_geometry(
    p_element_id UUID,
    p_geometry GEOMETRY,
    p_complex_data JSONB DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_version_id UUID;
    v_vertex_count INT;
BEGIN
    -- 计算几何复杂度
    v_vertex_count := ST_NPoints(p_geometry);
    
    -- 插入版本记录
    INSERT INTO element_versions (
        element_id,
        version_number,
        geometry_3d,
        geometry_complex,
        geometry_hash,
        properties,
        event_id,
        created_by
    ) VALUES (
        p_element_id,
        (SELECT COALESCE(MAX(version_number), 0) + 1 
         FROM element_versions WHERE element_id = p_element_id),
        CASE WHEN v_vertex_count <= get_geometry_storage_threshold() 
             THEN p_geometry ELSE NULL END,
        CASE WHEN v_vertex_count > get_geometry_storage_threshold() 
             THEN p_complex_data ELSE NULL END,
        MD5(ST_AsBinary(p_geometry)),
        '{}'::jsonb,
        gen_random_uuid(),  -- 临时事件ID
        'system'::uuid
    )
    RETURNING version_id INTO v_version_id;
    
    RETURN v_version_id;
END;
$$ LANGUAGE plpgsql;
```

### 2.4 索引策略验证

#### 2.4.1 索引设计矩阵

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         索引策略设计矩阵                                      │
├────────────────────┬────────────────────┬────────────────────────────────────┤
│ 表                 │ 索引类型           │ 索引字段/用途                      │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│ tenants            │ B-Tree             │ tenant_id (PK), slug (UNIQUE)      │
│                    │ B-Tree (Partial)   │ status WHERE deleted_at IS NULL    │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│ projects           │ B-Tree             │ project_id (PK), tenant_id         │
│                    │ B-Tree (Composite) │ (tenant_id, name) UNIQUE           │
│                    │ B-Tree             │ version_seq (用于事件序列)         │
│                    │ GIN                │ settings, stats (JSONB查询)        │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│ elements           │ B-Tree             │ element_id (PK), project_id        │
│                    │ B-Tree             │ element_type, parent_id, level_id  │
│                    │ GIST (PostGIS)     │ bbox_2d, bbox_3d (空间查询)        │
│                    │ GIN                │ classification, metadata (JSONB)   │
│                    │ B-Tree (Composite) │ (project_id, element_type)         │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│ element_versions   │ B-Tree             │ version_id (PK)                    │
│                    │ B-Tree (Composite) │ (element_id, version_number) UNIQUE│
│                    │ GIST (PostGIS)     │ geometry_2d, geometry_3d           │
│                    │ B-Tree             │ geometry_hash, properties_hash     │
│                    │ GIN                │ properties, change_summary         │
│                    │ B-Tree             │ created_at (时间范围查询)          │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│ events             │ B-Tree             │ event_id (PK)                      │
│                    │ B-Tree (Composite) │ (project_id, version_seq) UNIQUE   │
│                    │ B-Tree             │ project_id, event_type             │
│                    │ GIN                │ payload, metadata, affected_elements│
│                    │ B-Tree             │ executed_at, correlation_id        │
├────────────────────┼────────────────────┼────────────────────────────────────┤
│ snapshots          │ B-Tree             │ snapshot_id (PK)                   │
│                    │ B-Tree (Composite) │ (project_id, version_from, version_to)│
│                    │ B-Tree             │ snapshot_type                      │
└────────────────────┴────────────────────┴────────────────────────────────────┘
```

#### 2.4.2 索引创建脚本

```sql
-- ============================================
-- 完整索引创建脚本
-- ============================================

-- 租户表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tenants_status 
    ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tenants_slug 
    ON tenants(slug) WHERE deleted_at IS NULL;

-- 项目表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_tenant 
    ON projects(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_version_seq 
    ON projects(project_id, version_seq);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_settings_gin 
    ON projects USING GIN(settings);

-- 构件表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elements_project 
    ON elements(project_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elements_bbox_2d 
    ON elements USING GIST(bbox_2d);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elements_bbox_3d 
    ON elements USING GIST(bbox_3d);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elements_classification 
    ON elements USING GIN(classification);

-- 构件版本表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elem_ver_element 
    ON element_versions(element_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elem_ver_geom_3d 
    ON element_versions USING GIST(geometry_3d);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elem_ver_properties 
    ON element_versions USING GIN(properties);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_elem_ver_created 
    ON element_versions(created_at);

-- 事件表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_project_seq 
    ON events(project_id, version_seq);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_payload 
    ON events USING GIN(payload jsonb_path_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_executed_at 
    ON events(executed_at);

-- 快照表索引
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snapshots_version_range 
    ON snapshots(project_id, version_from, version_to);
```

---


## 3. 版本控制POC

### 3.1 Event Sourcing架构设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Event Sourcing 架构图                                    │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
    │   Command   │────▶│   Event     │────▶│   Event     │
    │   Handler   │     │   Store     │     │   Bus       │
    └─────────────┘     └─────────────┘     └──────┬──────┘
           │                                         │
           │                                         ▼
           │                              ┌─────────────┐
           │                              │  Projectors │
           │                              │  (Read Model)│
           │                              └─────────────┘
           │                                         │
           ▼                                         ▼
    ┌─────────────┐                         ┌─────────────┐
    │  Aggregate  │◄────────────────────────│   Query     │
    │  (Write)    │                         │   Handler   │
    └─────────────┘                         └─────────────┘
           │
           ▼
    ┌─────────────┐
    │  Snapshot   │
    │   Store     │
    └─────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  事件流转:                                                                   │
│  Command → Validation → Event Generation → Event Store → Projection → Query  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Event Sourcing实现验证

#### 3.2.1 事件发布函数

```sql
-- ============================================
-- 事件发布核心函数
-- ============================================

CREATE OR REPLACE FUNCTION publish_event(
    p_project_id UUID,
    p_event_type VARCHAR(50),
    p_payload JSONB,
    p_executed_by UUID,
    p_metadata JSONB DEFAULT '{}'::jsonb,
    p_affected_elements UUID[] DEFAULT '{}',
    p_correlation_id UUID DEFAULT NULL,
    p_causation_id UUID DEFAULT NULL,
    p_transaction_id UUID DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_event_id UUID;
    v_version_seq BIGINT;
    v_tenant_id UUID;
BEGIN
    -- 获取租户ID
    SELECT tenant_id INTO v_tenant_id
    FROM projects WHERE project_id = p_project_id;
    
    IF v_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Project not found: %', p_project_id;
    END IF;
    
    -- 获取下一个版本序列号 (使用行锁保证顺序)
    UPDATE projects 
    SET version_seq = version_seq + 1,
        stats = jsonb_set(
            stats, 
            '{total_versions}', 
            to_jsonb((stats->>'total_versions')::int + 1)
        )
    WHERE project_id = p_project_id
    RETURNING version_seq INTO v_version_seq;
    
    -- 插入事件
    INSERT INTO events (
        project_id,
        tenant_id,
        version_seq,
        event_type,
        payload,
        metadata,
        affected_elements,
        correlation_id,
        causation_id,
        transaction_id,
        executed_by,
        executed_at
    ) VALUES (
        p_project_id,
        v_tenant_id,
        v_version_seq,
        p_event_type,
        p_payload,
        p_metadata,
        p_affected_elements,
        COALESCE(p_correlation_id, gen_random_uuid()),
        p_causation_id,
        COALESCE(p_transaction_id, gen_random_uuid()),
        p_executed_by,
        NOW()
    )
    RETURNING event_id INTO v_event_id;
    
    -- 触发投影更新 (异步)
    PERFORM pg_notify('event_published', json_build_object(
        'event_id', v_event_id,
        'project_id', p_project_id,
        'event_type', p_event_type,
        'version_seq', v_version_seq
    )::text);
    
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;
```

#### 3.2.2 元素创建事件处理

```sql
-- ============================================
-- 元素创建命令处理
-- ============================================

CREATE OR REPLACE FUNCTION handle_create_element(
    p_project_id UUID,
    p_element_type VARCHAR(50),
    p_geometry_3d GEOMETRY,
    p_properties JSONB,
    p_created_by UUID,
    p_parent_id UUID DEFAULT NULL,
    p_classification JSONB DEFAULT '{}'::jsonb
) RETURNS UUID AS $$
DECLARE
    v_element_id UUID;
    v_event_id UUID;
    v_version_id UUID;
    v_bbox_3d GEOMETRY;
    v_bbox_2d GEOMETRY;
BEGIN
    -- 生成元素ID
    v_element_id := gen_random_uuid();
    
    -- 计算边界框
    v_bbox_3d := ST_3DExtent(p_geometry_3d)::GEOMETRY;
    v_bbox_2d := ST_Envelope(p_geometry_3d);
    
    -- 创建元素记录
    INSERT INTO elements (
        element_id,
        project_id,
        element_type,
        parent_id,
        bbox_2d,
        bbox_3d,
        classification,
        created_by
    ) VALUES (
        v_element_id,
        p_project_id,
        p_element_type,
        p_parent_id,
        v_bbox_2d,
        v_bbox_3d,
        p_classification,
        p_created_by
    );
    
    -- 创建初始版本
    INSERT INTO element_versions (
        element_id,
        version_number,
        geometry_3d,
        geometry_hash,
        properties,
        properties_hash,
        full_snapshot,
        event_id,
        created_by
    ) VALUES (
        v_element_id,
        1,
        p_geometry_3d,
        MD5(ST_AsBinary(p_geometry_3d)),
        p_properties,
        MD5(p_properties::text),
        jsonb_build_object(
            'element_id', v_element_id,
            'geometry', ST_AsGeoJSON(p_geometry_3d)::jsonb,
            'properties', p_properties
        ),
        gen_random_uuid(),  -- 临时
        p_created_by
    )
    RETURNING version_id INTO v_version_id;
    
    -- 更新元素的当前版本
    UPDATE elements 
    SET current_version = v_version_id,
        updated_at = NOW()
    WHERE element_id = v_element_id;
    
    -- 发布事件
    v_event_id := publish_event(
        p_project_id,
        'element.created',
        jsonb_build_object(
            'element_id', v_element_id,
            'element_type', p_element_type,
            'parent_id', p_parent_id,
            'geometry_hash', MD5(ST_AsBinary(p_geometry_3d)),
            'properties_hash', MD5(p_properties::text)
        ),
        p_created_by,
        '{}'::jsonb,
        ARRAY[v_element_id]
    );
    
    -- 更新版本的事件引用
    UPDATE element_versions SET event_id = v_event_id 
    WHERE version_id = v_version_id;
    
    -- 更新项目统计
    UPDATE projects 
    SET stats = jsonb_set(
        stats,
        '{element_count}',
        to_jsonb((stats->>'element_count')::int + 1)
    )
    WHERE project_id = p_project_id;
    
    RETURN v_element_id;
END;
$$ LANGUAGE plpgsql;
```

#### 3.2.3 元素更新事件处理

```sql
-- ============================================
-- 元素更新命令处理
-- ============================================

CREATE OR REPLACE FUNCTION handle_update_element(
    p_element_id UUID,
    p_geometry_3d GEOMETRY DEFAULT NULL,
    p_properties JSONB DEFAULT NULL,
    p_updated_by UUID,
    p_expected_version INT DEFAULT NULL  -- 乐观锁检查
) RETURNS UUID AS $$
DECLARE
    v_project_id UUID;
    v_current_version INT;
    v_version_id UUID;
    v_event_id UUID;
    v_old_geom_hash VARCHAR(64);
    v_old_props_hash VARCHAR(64);
    v_new_geom_hash VARCHAR(64);
    v_new_props_hash VARCHAR(64);
    v_change_type VARCHAR(20);
    v_changed_fields TEXT[];
    v_bbox_3d GEOMETRY;
    v_bbox_2d GEOMETRY;
BEGIN
    -- 获取项目ID和当前版本
    SELECT e.project_id, ev.version_number, 
           ev.geometry_hash, ev.properties_hash
    INTO v_project_id, v_current_version, v_old_geom_hash, v_old_props_hash
    FROM elements e
    JOIN element_versions ev ON e.current_version = ev.version_id
    WHERE e.element_id = p_element_id;
    
    IF v_project_id IS NULL THEN
        RAISE EXCEPTION 'Element not found: %', p_element_id;
    END IF;
    
    -- 乐观锁检查
    IF p_expected_version IS NOT NULL AND p_expected_version != v_current_version THEN
        RAISE EXCEPTION 'Version conflict: expected %, found %', 
            p_expected_version, v_current_version;
    END IF;
    
    -- 计算新哈希值
    IF p_geometry_3d IS NOT NULL THEN
        v_new_geom_hash := MD5(ST_AsBinary(p_geometry_3d));
    ELSE
        v_new_geom_hash := v_old_geom_hash;
    END IF;
    
    IF p_properties IS NOT NULL THEN
        v_new_props_hash := MD5(p_properties::text);
    ELSE
        v_new_props_hash := v_old_props_hash;
    END IF;
    
    -- 确定变更类型
    IF v_new_geom_hash != v_old_geom_hash AND v_new_props_hash != v_old_props_hash THEN
        v_change_type := 'both';
        v_changed_fields := ARRAY['geometry', 'properties'];
    ELSIF v_new_geom_hash != v_old_geom_hash THEN
        v_change_type := 'geometry';
        v_changed_fields := ARRAY['geometry'];
    ELSIF v_new_props_hash != v_old_props_hash THEN
        v_change_type := 'properties';
        v_changed_fields := ARRAY['properties'];
    ELSE
        RAISE EXCEPTION 'No changes detected for element %', p_element_id;
    END IF;
    
    -- 计算新边界框
    IF p_geometry_3d IS NOT NULL THEN
        v_bbox_3d := ST_3DExtent(p_geometry_3d)::GEOMETRY;
        v_bbox_2d := ST_Envelope(p_geometry_3d);
    END IF;
    
    -- 创建新版本
    INSERT INTO element_versions (
        element_id,
        version_number,
        geometry_3d,
        geometry_hash,
        properties,
        properties_hash,
        change_summary,
        created_by
    ) VALUES (
        p_element_id,
        v_current_version + 1,
        COALESCE(p_geometry_3d, (SELECT geometry_3d FROM element_versions 
                                  WHERE element_id = p_element_id 
                                  ORDER BY version_number DESC LIMIT 1)),
        v_new_geom_hash,
        COALESCE(p_properties, (SELECT properties FROM element_versions 
                                WHERE element_id = p_element_id 
                                ORDER BY version_number DESC LIMIT 1)),
        v_new_props_hash,
        jsonb_build_object(
            'change_type', v_change_type,
            'changed_fields', v_changed_fields,
            'previous_version', v_current_version
        ),
        p_updated_by
    )
    RETURNING version_id INTO v_version_id;
    
    -- 更新元素记录
    UPDATE elements 
    SET current_version = v_version_id,
        version_count = version_count + 1,
        bbox_2d = COALESCE(v_bbox_2d, bbox_2d),
        bbox_3d = COALESCE(v_bbox_3d, bbox_3d),
        updated_at = NOW(),
        lock_version = lock_version + 1
    WHERE element_id = p_element_id;
    
    -- 发布事件
    v_event_id := publish_event(
        v_project_id,
        'element.updated',
        jsonb_build_object(
            'element_id', p_element_id,
            'change_type', v_change_type,
            'changed_fields', v_changed_fields,
            'from_version', v_current_version,
            'to_version', v_current_version + 1
        ),
        p_updated_by,
        '{}'::jsonb,
        ARRAY[p_element_id]
    );
    
    -- 更新版本的事件引用
    UPDATE element_versions SET event_id = v_event_id 
    WHERE version_id = v_version_id;
    
    RETURN v_event_id;
END;
$$ LANGUAGE plpgsql;
```

### 3.3 快照策略验证

#### 3.3.1 快照策略架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         快照策略架构                                          │
└─────────────────────────────────────────────────────────────────────────────┘

    事件流:  [E1] → [E2] → [E3] → [E4] → [E5] → [E6] → [E7] → [E8] → [E9]
             │      │      │      │      │      │      │      │      │
             └──────┴──────┴──────┴──────┴──────┴──────┴──────┴──────┘
                                                                         
    全量快照: [S1@E3]                              [S2@E6]      [S3@E9]
              │                                      │           │
              └──────── 包含E1-E3的所有状态 ──────────┘           │
                                                                 
    增量快照:              [D1@E4-E5]                [D2@E7-E8]
                           │                         │
                           └─ 仅包含变更增量 ─────────┘

    恢复流程:
    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
    │ 加载S2  │ →  │ 应用D2  │ →  │ 应用E9  │ →  │ 当前状态 │
    │ (基础)  │    │ (增量)  │    │ (最新)  │    │         │
    └─────────┘    └─────────┘    └─────────┘    └─────────┘
```

#### 3.3.2 快照创建函数

```sql
-- ============================================
-- 快照创建函数
-- ============================================

CREATE OR REPLACE FUNCTION create_snapshot(
    p_project_id UUID,
    p_snapshot_type VARCHAR(20) DEFAULT 'full',
    p_version_from BIGINT DEFAULT NULL,
    p_version_to BIGINT DEFAULT NULL,
    p_created_by UUID DEFAULT 'system'::uuid
) RETURNS UUID AS $$
DECLARE
    v_snapshot_id UUID;
    v_actual_version_from BIGINT;
    v_actual_version_to BIGINT;
    v_snapshot_data JSONB;
    v_element_count INT;
    v_element_ids UUID[];
BEGIN
    -- 确定版本范围
    IF p_version_to IS NULL THEN
        SELECT version_seq INTO v_actual_version_to
        FROM projects WHERE project_id = p_project_id;
    ELSE
        v_actual_version_to := p_version_to;
    END IF;
    
    IF p_snapshot_type = 'full' THEN
        v_actual_version_from := 0;
    ELSIF p_version_from IS NULL THEN
        -- 查找上一个快照的结束版本
        SELECT version_to INTO v_actual_version_from
        FROM snapshots
        WHERE project_id = p_project_id
        ORDER BY version_to DESC
        LIMIT 1;
        
        v_actual_version_from := COALESCE(v_actual_version_from, 0);
    ELSE
        v_actual_version_from := p_version_from;
    END IF;
    
    -- 收集元素数据
    IF p_snapshot_type = 'full' THEN
        -- 全量快照: 收集所有元素的当前版本
        SELECT 
            jsonb_object_agg(e.element_id::text, jsonb_build_object(
                'element_type', e.element_type,
                'version_number', ev.version_number,
                'geometry', ST_AsGeoJSON(ev.geometry_3d)::jsonb,
                'properties', ev.properties,
                'classification', e.classification,
                'parent_id', e.parent_id
            )),
            COUNT(*),
            array_agg(e.element_id)
        INTO v_snapshot_data, v_element_count, v_element_ids
        FROM elements e
        JOIN element_versions ev ON e.current_version = ev.version_id
        WHERE e.project_id = p_project_id
        AND e.deleted_at IS NULL;
        
    ELSIF p_snapshot_type = 'incremental' THEN
        -- 增量快照: 仅收集变更的元素
        SELECT 
            jsonb_object_agg(e.element_id::text, jsonb_build_object(
                'element_type', e.element_type,
                'version_number', ev.version_number,
                'geometry', ST_AsGeoJSON(ev.geometry_3d)::jsonb,
                'properties', ev.properties,
                'change_summary', ev.change_summary
            )),
            COUNT(*),
            array_agg(e.element_id)
        INTO v_snapshot_data, v_element_count, v_element_ids
        FROM elements e
        JOIN element_versions ev ON e.current_version = ev.version_id
        WHERE e.project_id = p_project_id
        AND e.element_id IN (
            SELECT DISTINCT unnest(affected_elements)
            FROM events
            WHERE project_id = p_project_id
            AND version_seq > v_actual_version_from
            AND version_seq <= v_actual_version_to
        );
    END IF;
    
    -- 插入快照记录
    INSERT INTO snapshots (
        project_id,
        snapshot_type,
        version_from,
        version_to,
        snapshot_data,
        element_count,
        element_ids,
        created_by
    ) VALUES (
        p_project_id,
        p_snapshot_type,
        v_actual_version_from,
        v_actual_version_to,
        v_snapshot_data,
        v_element_count,
        v_element_ids,
        p_created_by
    )
    RETURNING snapshot_id INTO v_snapshot_id;
    
    -- 记录快照创建事件
    PERFORM publish_event(
        p_project_id,
        'project.snapshot',
        jsonb_build_object(
            'snapshot_id', v_snapshot_id,
            'snapshot_type', p_snapshot_type,
            'version_from', v_actual_version_from,
            'version_to', v_actual_version_to,
            'element_count', v_element_count
        ),
        p_created_by
    );
    
    RETURN v_snapshot_id;
END;
$$ LANGUAGE plpgsql;
```

#### 3.3.3 自动快照策略

```sql
-- ============================================
-- 自动快照策略配置
-- ============================================

CREATE TABLE snapshot_policies (
    policy_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID REFERENCES projects(project_id),
    
    -- 触发条件
    trigger_type    VARCHAR(20) NOT NULL 
                    CHECK (trigger_type IN ('interval', 'event_count', 'manual')),
    
    -- 间隔触发 (分钟)
    interval_minutes INT,
    
    -- 事件数量触发
    event_threshold INT,
    
    -- 快照类型
    snapshot_type   VARCHAR(20) DEFAULT 'incremental',
    
    -- 保留策略
    keep_count      INT DEFAULT 10,  -- 保留最近N个快照
    keep_days       INT DEFAULT 30,  -- 保留N天
    
    -- 状态
    is_active       BOOLEAN DEFAULT TRUE,
    
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- 自动快照调度函数
CREATE OR REPLACE FUNCTION check_and_create_snapshot(p_project_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_policy snapshot_policies%ROWTYPE;
    v_last_snapshot snapshots%ROWTYPE;
    v_event_count BIGINT;
    v_should_snapshot BOOLEAN := FALSE;
BEGIN
    -- 获取项目策略
    SELECT * INTO v_policy
    FROM snapshot_policies
    WHERE project_id = p_project_id
    AND is_active = TRUE
    ORDER BY created_at DESC
    LIMIT 1;
    
    IF v_policy.policy_id IS NULL THEN
        -- 使用默认策略
        v_policy.trigger_type := 'event_count';
        v_policy.event_threshold := 100;
        v_policy.snapshot_type := 'incremental';
    END IF;
    
    -- 获取上次快照
    SELECT * INTO v_last_snapshot
    FROM snapshots
    WHERE project_id = p_project_id
    ORDER BY version_to DESC
    LIMIT 1;
    
    -- 检查触发条件
    IF v_policy.trigger_type = 'interval' THEN
        IF v_last_snapshot.snapshot_id IS NULL OR
           v_last_snapshot.created_at < NOW() - (v_policy.interval_minutes || ' minutes')::INTERVAL THEN
            v_should_snapshot := TRUE;
        END IF;
        
    ELSIF v_policy.trigger_type = 'event_count' THEN
        IF v_last_snapshot.snapshot_id IS NULL THEN
            -- 检查总事件数
            SELECT COUNT(*) INTO v_event_count
            FROM events WHERE project_id = p_project_id;
        ELSE
            -- 检查新增事件数
            SELECT COUNT(*) INTO v_event_count
            FROM events 
            WHERE project_id = p_project_id
            AND version_seq > v_last_snapshot.version_to;
        END IF;
        
        IF v_event_count >= v_policy.event_threshold THEN
            v_should_snapshot := TRUE;
        END IF;
    END IF;
    
    -- 创建快照
    IF v_should_snapshot THEN
        PERFORM create_snapshot(
            p_project_id,
            v_policy.snapshot_type,
            CASE WHEN v_policy.snapshot_type = 'incremental' 
                 THEN v_last_snapshot.version_to ELSE NULL END,
            NULL,
            'system'::uuid
        );
        
        -- 清理旧快照
        PERFORM cleanup_old_snapshots(p_project_id, v_policy.keep_count, v_policy.keep_days);
        
        RETURN TRUE;
    END IF;
    
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- 清理旧快照
CREATE OR REPLACE FUNCTION cleanup_old_snapshots(
    p_project_id UUID,
    p_keep_count INT,
    p_keep_days INT
) RETURNS INT AS $$
DECLARE
    v_deleted_count INT := 0;
BEGIN
    -- 删除超出保留数量的旧快照
    WITH to_delete AS (
        SELECT snapshot_id
        FROM snapshots
        WHERE project_id = p_project_id
        ORDER BY created_at DESC
        OFFSET p_keep_count
    )
    DELETE FROM snapshots
    WHERE snapshot_id IN (SELECT snapshot_id FROM to_delete);
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    
    -- 删除超出保留时间的快照
    DELETE FROM snapshots
    WHERE project_id = p_project_id
    AND created_at < NOW() - (p_keep_days || ' days')::INTERVAL;
    
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;
```

### 3.4 时间旅行查询验证

#### 3.4.1 时间旅行查询函数

```sql
-- ============================================
-- 时间旅行查询 - 获取指定版本的项目状态
-- ============================================

CREATE OR REPLACE FUNCTION get_project_at_version(
    p_project_id UUID,
    p_target_version BIGINT
) RETURNS TABLE (
    element_id UUID,
    element_type VARCHAR(50),
    version_number INT,
    geometry_3d GEOMETRY,
    properties JSONB,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    -- 方法1: 从事件重建 (精确但较慢)
    RETURN QUERY
    WITH element_latest_version AS (
        SELECT DISTINCT ON (ev.element_id)
            ev.element_id,
            ev.version_number,
            ev.geometry_3d,
            ev.properties,
            ev.created_at
        FROM element_versions ev
        JOIN events e ON ev.event_id = e.event_id
        WHERE e.project_id = p_project_id
        AND e.version_seq <= p_target_version
        AND ev.element_id IN (
            -- 只包含未被删除的元素
            SELECT DISTINCT unnest(affected_elements)
            FROM events
            WHERE project_id = p_project_id
            AND version_seq <= p_target_version
            AND event_type != 'element.deleted'
        )
        ORDER BY ev.element_id, ev.version_number DESC
    )
    SELECT 
        e.element_id,
        e.element_type,
        elv.version_number,
        elv.geometry_3d,
        elv.properties,
        elv.created_at
    FROM elements e
    JOIN element_latest_version elv ON e.element_id = elv.element_id
    WHERE e.project_id = p_project_id
    AND e.deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 优化的快照辅助时间旅行查询
-- ============================================

CREATE OR REPLACE FUNCTION get_project_at_version_optimized(
    p_project_id UUID,
    p_target_version BIGINT
) RETURNS TABLE (
    element_id UUID,
    element_type VARCHAR(50),
    version_number INT,
    geometry_3d GEOMETRY,
    properties JSONB,
    created_at TIMESTAMPTZ
) AS $$
DECLARE
    v_base_snapshot snapshots%ROWTYPE;
    v_base_version BIGINT;
BEGIN
    -- 查找最近的快照
    SELECT * INTO v_base_snapshot
    FROM snapshots
    WHERE project_id = p_project_id
    AND version_to <= p_target_version
    ORDER BY version_to DESC
    LIMIT 1;
    
    IF v_base_snapshot.snapshot_id IS NULL THEN
        -- 无可用快照，从事件重建
        v_base_version := 0;
    ELSE
        v_base_version := v_base_snapshot.version_to;
        
        -- 返回快照中的元素
        RETURN QUERY
        SELECT 
            (key)::UUID,
            (value->>'element_type')::VARCHAR(50),
            (value->>'version_number')::INT,
            ST_GeomFromGeoJSON(value->>'geometry'),
            value->'properties',
            (value->>'created_at')::TIMESTAMPTZ
        FROM jsonb_each(v_base_snapshot.snapshot_data);
    END IF;
    
    -- 应用快照后的事件
    RETURN QUERY
    WITH events_after_snapshot AS (
        SELECT 
            e.event_type,
            e.payload,
            e.affected_elements,
            e.version_seq
        FROM events e
        WHERE e.project_id = p_project_id
        AND e.version_seq > v_base_version
        AND e.version_seq <= p_target_version
        ORDER BY e.version_seq
    )
    SELECT * FROM apply_events_to_state(
        p_project_id, 
        v_base_version, 
        p_target_version
    );
    
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 时间点查询 - 获取指定时间的状态
-- ============================================

CREATE OR REPLACE FUNCTION get_project_at_time(
    p_project_id UUID,
    p_target_time TIMESTAMPTZ
) RETURNS TABLE (
    element_id UUID,
    element_type VARCHAR(50),
    version_number INT,
    geometry_3d GEOMETRY,
    properties JSONB
) AS $$
DECLARE
    v_target_version BIGINT;
BEGIN
    -- 查找目标时间对应的版本号
    SELECT MAX(version_seq) INTO v_target_version
    FROM events
    WHERE project_id = p_project_id
    AND executed_at <= p_target_time;
    
    IF v_target_version IS NULL THEN
        v_target_version := 0;
    END IF;
    
    -- 使用版本查询
    RETURN QUERY
    SELECT * FROM get_project_at_version(p_project_id, v_target_version);
END;
$$ LANGUAGE plpgsql;
```

### 3.5 历史数据压缩验证

#### 3.5.1 压缩策略设计

```sql
-- ============================================
-- 历史数据压缩策略
-- ============================================

-- 压缩配置表
CREATE TABLE compression_policies (
    policy_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID REFERENCES projects(project_id),
    
    -- 压缩触发条件
    compress_after_days     INT DEFAULT 30,     -- N天后压缩
    compress_after_versions INT DEFAULT 100,    -- N个版本后压缩
    
    -- 压缩级别
    compression_level       VARCHAR(20) DEFAULT 'medium'
                            CHECK (compression_level IN ('low', 'medium', 'high')),
    
    -- 保留策略
    keep_original_for_days  INT DEFAULT 7,      -- 保留原始数据N天
    
    is_active               BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ DEFAULT NOW()
);

-- 压缩历史表
CREATE TABLE element_versions_compressed (
    compressed_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    element_id      UUID NOT NULL,
    
    -- 版本范围
    version_from    INT NOT NULL,
    version_to      INT NOT NULL,
    
    -- 压缩数据 (使用PostgreSQL内置压缩)
    compressed_data BYTEA NOT NULL,
    compression_method VARCHAR(20) DEFAULT 'pglz',  -- pglz, lz4
    original_size   BIGINT,
    compressed_size BIGINT,
    
    -- 摘要信息 (不解压即可查询)
    version_count   INT,
    change_summary  JSONB,
    
    -- 元数据
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    compressed_at   TIMESTAMPTZ DEFAULT NOW()
);

-- 压缩函数
CREATE OR REPLACE FUNCTION compress_element_history(
    p_element_id UUID,
    p_compression_level VARCHAR(20) DEFAULT 'medium'
) RETURNS UUID AS $$
DECLARE
    v_compressed_id UUID;
    v_versions JSONB;
    v_compressed BYTEA;
    v_original_size BIGINT;
    v_version_from INT;
    v_version_to INT;
BEGIN
    -- 收集历史版本
    SELECT 
        jsonb_agg(jsonb_build_object(
            'version_number', version_number,
            'geometry', ST_AsGeoJSON(geometry_3d)::jsonb,
            'properties', properties,
            'change_summary', change_summary,
            'created_at', created_at
        ) ORDER BY version_number),
        MIN(version_number),
        MAX(version_number)
    INTO v_versions, v_version_from, v_version_to
    FROM element_versions
    WHERE element_id = p_element_id
    AND version_number < (
        -- 保留最近N个版本不压缩
        SELECT MAX(version_number) - 10
        FROM element_versions
        WHERE element_id = p_element_id
    );
    
    IF v_versions IS NULL OR jsonb_array_length(v_versions) = 0 THEN
        RETURN NULL;
    END IF;
    
    v_original_size := octet_length(v_versions::text);
    
    -- 压缩数据
    v_compressed := pglz_compress(v_versions::text::bytea);
    
    -- 插入压缩记录
    INSERT INTO element_versions_compressed (
        element_id,
        version_from,
        version_to,
        compressed_data,
        original_size,
        compressed_size,
        version_count,
        change_summary
    ) VALUES (
        p_element_id,
        v_version_from,
        v_version_to,
        v_compressed,
        v_original_size,
        octet_length(v_compressed),
        jsonb_array_length(v_versions),
        jsonb_build_object(
            'total_versions', jsonb_array_length(v_versions),
            'compressed_at', NOW()
        )
    )
    RETURNING compressed_id INTO v_compressed_id;
    
    -- 可选: 删除原始版本 (根据保留策略)
    -- DELETE FROM element_versions
    -- WHERE element_id = p_element_id
    -- AND version_number BETWEEN v_version_from AND v_version_to;
    
    RETURN v_compressed_id;
END;
$$ LANGUAGE plpgsql;

-- 解压函数
CREATE OR REPLACE FUNCTION decompress_element_history(
    p_compressed_id UUID
) RETURNS JSONB AS $$
DECLARE
    v_compressed BYTEA;
    v_decompressed TEXT;
BEGIN
    SELECT compressed_data INTO v_compressed
    FROM element_versions_compressed
    WHERE compressed_id = p_compressed_id;
    
    IF v_compressed IS NULL THEN
        RETURN NULL;
    END IF;
    
    v_decompressed := pglz_decompress(v_compressed);
    
    RETURN v_decompressed::jsonb;
END;
$$ LANGUAGE plpgsql;
```

---


## 4. 并发控制POC

### 4.1 并发控制架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       并发控制架构图                                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                         并发访问场景                                 │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   用户A ──┐                                                         │
    │           │    ┌─────────────┐                                      │
    │           ├───▶│  乐观锁     │────┐                                 │
    │   用户B ──┤    │  (版本号)   │    │                                 │
    │           │    └─────────────┘    │                                 │
    │           │                       │                                 │
    │   用户C ──┤    ┌─────────────┐    │    ┌─────────────┐             │
    │           ├───▶│  MVCC       │────┼───▶│  数据库     │             │
    │           │    │  (多版本)   │    │    │  YugabyteDB │             │
    │   用户D ──┘    └─────────────┘    │    └─────────────┘             │
    │                                   │                                 │
    │           ┌─────────────┐         │                                 │
    │           │  分布式锁   │─────────┘                                 │
    │           │  (Redis)    │                                           │
    │           └─────────────┘                                           │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

### 4.2 乐观锁实现验证

#### 4.2.1 乐观锁机制设计

```sql
-- ============================================
-- 乐观锁实现
-- ============================================

-- 在elements表中已添加 lock_version 字段
-- ALTER TABLE elements ADD COLUMN lock_version INT DEFAULT 1;

-- 乐观锁更新函数
CREATE OR REPLACE FUNCTION update_element_with_optimistic_lock(
    p_element_id UUID,
    p_expected_version INT,
    p_geometry_3d GEOMETRY DEFAULT NULL,
    p_properties JSONB DEFAULT NULL,
    p_updated_by UUID
) RETURNS JSONB AS $$
DECLARE
    v_current_version INT;
    v_project_id UUID;
    v_result JSONB;
BEGIN
    -- 获取当前版本
    SELECT lock_version, project_id 
    INTO v_current_version, v_project_id
    FROM elements 
    WHERE element_id = p_element_id
    FOR UPDATE;  -- 行锁
    
    -- 版本检查
    IF v_current_version IS NULL THEN
        RETURN jsonb_build_object(
            'success', FALSE,
            'error', 'Element not found',
            'error_code', 'NOT_FOUND'
        );
    END IF;
    
    IF v_current_version != p_expected_version THEN
        RETURN jsonb_build_object(
            'success', FALSE,
            'error', 'Version conflict detected',
            'error_code', 'VERSION_CONFLICT',
            'expected_version', p_expected_version,
            'current_version', v_current_version
        );
    END IF;
    
    -- 执行更新
    PERFORM handle_update_element(
        p_element_id,
        p_geometry_3d,
        p_properties,
        p_updated_by
    );
    
    -- 递增锁版本
    UPDATE elements 
    SET lock_version = lock_version + 1
    WHERE element_id = p_element_id;
    
    RETURN jsonb_build_object(
        'success', TRUE,
        'new_version', v_current_version + 1,
        'element_id', p_element_id
    );
END;
$$ LANGUAGE plpgsql;
```

#### 4.2.2 批量乐观锁更新

```sql
-- ============================================
-- 批量乐观锁更新
-- ============================================

CREATE OR REPLACE FUNCTION batch_update_elements(
    p_updates JSONB,  -- [{"element_id": "...", "expected_version": N, "properties": {...}}, ...]
    p_updated_by UUID
) RETURNS JSONB AS $$
DECLARE
    v_update JSONB;
    v_element_id UUID;
    v_expected_version INT;
    v_results JSONB := '[]'::jsonb;
    v_success_count INT := 0;
    v_conflict_count INT := 0;
BEGIN
    FOR v_update IN SELECT jsonb_array_elements(p_updates)
    LOOP
        v_element_id := (v_update->>'element_id')::UUID;
        v_expected_version := (v_update->>'expected_version')::INT;
        
        BEGIN
            -- 尝试单个更新
            PERFORM update_element_with_optimistic_lock(
                v_element_id,
                v_expected_version,
                NULL,  -- geometry
                v_update->'properties',
                p_updated_by
            );
            
            v_success_count := v_success_count + 1;
            v_results := v_results || jsonb_build_object(
                'element_id', v_element_id,
                'status', 'success'
            );
            
        EXCEPTION WHEN OTHERS THEN
            v_conflict_count := v_conflict_count + 1;
            v_results := v_results || jsonb_build_object(
                'element_id', v_element_id,
                'status', 'conflict',
                'error', SQLERRM
            );
        END;
    END LOOP;
    
    RETURN jsonb_build_object(
        'total', jsonb_array_length(p_updates),
        'success', v_success_count,
        'conflicts', v_conflict_count,
        'results', v_results
    );
END;
$$ LANGUAGE plpgsql;
```

### 4.3 MVCC行为验证

#### 4.3.1 MVCC测试脚本

```sql
-- ============================================
-- MVCC行为验证测试
-- ============================================

-- 测试1: 读已提交隔离级别
-- 会话A
BEGIN ISOLATION LEVEL READ COMMITTED;
SELECT * FROM elements WHERE element_id = 'test-id';
-- 返回: version=1

-- 会话B (同时)
BEGIN;
UPDATE elements SET properties = '{"updated": true}'::jsonb 
WHERE element_id = 'test-id';
COMMIT;

-- 会话A (再次查询)
SELECT * FROM elements WHERE element_id = 'test-id';
-- 返回: version=2 (读取到最新已提交数据)
COMMIT;

-- 测试2: 可重复读隔离级别
-- 会话A
BEGIN ISOLATION LEVEL REPEATABLE READ;
SELECT * FROM elements WHERE element_id = 'test-id';
-- 返回: version=1

-- 会话B (同时)
BEGIN;
UPDATE elements SET properties = '{"updated": true}'::jsonb 
WHERE element_id = 'test-id';
COMMIT;

-- 会话A (再次查询)
SELECT * FROM elements WHERE element_id = 'test-id';
-- 返回: version=1 (事务内保持一致的快照)
COMMIT;

-- 测试3: 序列化隔离级别
-- 会话A
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT * FROM elements WHERE project_id = 'project-1';

-- 会话B
BEGIN ISOLATION LEVEL SERIALIZABLE;
INSERT INTO elements (element_id, project_id, ...) VALUES (...);
COMMIT;

-- 会话A (尝试更新)
UPDATE elements SET ... WHERE project_id = 'project-1';
-- 可能报错: could not serialize access due to read/write dependencies
COMMIT;
```

#### 4.3.2 MVCC配置优化

```sql
-- ============================================
-- MVCC配置优化 (YugabyteDB/CockroachDB)
-- ============================================

-- 设置合适的vacuum参数
ALTER SYSTEM SET autovacuum_vacuum_scale_factor = 0.1;
ALTER SYSTEM SET autovacuum_analyze_scale_factor = 0.05;

-- 设置事务ID回卷保护
ALTER SYSTEM SET autovacuum_freeze_min_age = 50000000;
ALTER SYSTEM SET vacuum_freeze_table_age = 150000000;

-- YugabyteDB特定配置
-- SET yb_transaction_priority_upper_bound = 0.5;
-- SET yb_transaction_priority_lower_bound = 0.1;

-- 监控MVCC状态
CREATE OR REPLACE VIEW mvcc_status AS
SELECT
    schemaname,
    relname,
    n_live_tup,
    n_dead_tup,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    vacuum_count,
    autovacuum_count
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY n_dead_tup DESC;
```

### 4.4 冲突检测机制验证

#### 4.4.1 冲突检测架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       冲突检测机制                                            │
└─────────────────────────────────────────────────────────────────────────────┘

    编辑A ──┐
            │    ┌─────────────┐
            ├───▶│  操作队列   │
            │    │  (Redis)    │
    编辑B ──┤    └──────┬──────┘
            │           │
            │    ┌──────▼──────┐
            └───▶│  冲突检测   │
                 │  引擎       │
                 └──────┬──────┘
                        │
           ┌────────────┼────────────┐
           │            │            │
           ▼            ▼            ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │ 无冲突   │ │ 可合并   │ │ 冲突     │
    │ 直接应用 │ │ 自动合并 │ │ 人工解决 │
    └──────────┘ └──────────┘ └──────────┘
```

#### 4.4.2 冲突检测函数

```sql
-- ============================================
-- 冲突检测函数
-- ============================================

-- 冲突类型枚举
CREATE TYPE conflict_type AS ENUM (
    'NO_CONFLICT',
    'GEOMETRY_CONFLICT',
    'PROPERTY_CONFLICT',
    'DELETE_CONFLICT',
    'DEPENDENCY_CONFLICT'
);

-- 冲突检测表
CREATE TABLE detected_conflicts (
    conflict_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL,
    
    -- 冲突双方
    source_event_id UUID NOT NULL,
    target_event_id UUID NOT NULL,
    
    -- 冲突详情
    conflict_type   conflict_type NOT NULL,
    element_ids     UUID[],
    
    -- 冲突描述
    description     TEXT,
    resolution_data JSONB,  -- 用于自动合并的数据
    
    -- 状态
    status          VARCHAR(20) DEFAULT 'pending'
                    CHECK (status IN ('pending', 'auto_resolved', 'manual_resolved', 'rejected')),
    
    -- 时间戳
    detected_at     TIMESTAMPTZ DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    resolved_by     UUID
);

-- 冲突检测函数
CREATE OR REPLACE FUNCTION detect_conflict(
    p_event_id_1 UUID,
    p_event_id_2 UUID
) RETURNS conflict_type AS $$
DECLARE
    v_event_1 events%ROWTYPE;
    v_event_2 events%ROWTYPE;
    v_common_elements UUID[];
    v_conflict conflict_type := 'NO_CONFLICT';
BEGIN
    -- 获取事件详情
    SELECT * INTO v_event_1 FROM events WHERE event_id = p_event_id_1;
    SELECT * INTO v_event_2 FROM events WHERE event_id = p_event_id_2;
    
    -- 检查是否有共同影响的元素
    SELECT array_agg(elem) INTO v_common_elements
    FROM (
        SELECT unnest(v_event_1.affected_elements)
        INTERSECT
        SELECT unnest(v_event_2.affected_elements)
    ) t(elem);
    
    IF v_common_elements IS NULL OR array_length(v_common_elements, 1) = 0 THEN
        RETURN 'NO_CONFLICT';
    END IF;
    
    -- 检查删除冲突
    IF v_event_1.event_type = 'element.deleted' OR 
       v_event_2.event_type = 'element.deleted' THEN
        RETURN 'DELETE_CONFLICT';
    END IF;
    
    -- 检查几何冲突
    IF (v_event_1.payload ? 'geometry_hash' AND v_event_2.payload ? 'geometry_hash') OR
       (v_event_1.event_type LIKE '%geometry%' OR v_event_2.event_type LIKE '%geometry%') THEN
        v_conflict := 'GEOMETRY_CONFLICT';
    END IF;
    
    -- 检查属性冲突
    IF v_event_1.payload ? 'properties' AND v_event_2.payload ? 'properties' THEN
        IF v_conflict = 'NO_CONFLICT' THEN
            v_conflict := 'PROPERTY_CONFLICT';
        END IF;
    END IF;
    
    -- 记录冲突
    IF v_conflict != 'NO_CONFLICT' THEN
        INSERT INTO detected_conflicts (
            project_id,
            source_event_id,
            target_event_id,
            conflict_type,
            element_ids,
            description
        ) VALUES (
            v_event_1.project_id,
            p_event_id_1,
            p_event_id_2,
            v_conflict,
            v_common_elements,
            format('Conflict detected between events %s and %s on elements %s',
                   p_event_id_1, p_event_id_2, v_common_elements)
        );
    END IF;
    
    RETURN v_conflict;
END;
$$ LANGUAGE plpgsql;

-- 批量冲突检测
CREATE OR REPLACE FUNCTION detect_conflicts_in_range(
    p_project_id UUID,
    p_version_from BIGINT,
    p_version_to BIGINT
) RETURNS TABLE (
    conflict_id UUID,
    conflict_type conflict_type,
    element_ids UUID[],
    description TEXT
) AS $$
BEGIN
    RETURN QUERY
    WITH event_pairs AS (
        SELECT 
            e1.event_id as event_id_1,
            e2.event_id as event_id_2,
            e1.affected_elements as elements_1,
            e2.affected_elements as elements_2
        FROM events e1
        JOIN events e2 ON e1.project_id = e2.project_id
            AND e1.version_seq < e2.version_seq
            AND e1.version_seq BETWEEN p_version_from AND p_version_to
            AND e2.version_seq BETWEEN p_version_from AND p_version_to
        WHERE e1.project_id = p_project_id
    )
    SELECT 
        gen_random_uuid(),
        CASE 
            WHEN ep.elements_1 && ep.elements_2 THEN 'PROPERTY_CONFLICT'::conflict_type
            ELSE 'NO_CONFLICT'::conflict_type
        END,
        ep.elements_1 && ep.elements_2 as common_elements,
        'Detected potential conflict'::TEXT
    FROM event_pairs ep
    WHERE ep.elements_1 && ep.elements_2;
END;
$$ LANGUAGE plpgsql;
```

### 4.5 数据一致性验证

#### 4.5.1 一致性检查函数

```sql
-- ============================================
-- 数据一致性验证
-- ============================================

-- 一致性检查表
CREATE TABLE consistency_checks (
    check_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID,
    check_type      VARCHAR(50) NOT NULL,
    status          VARCHAR(20) DEFAULT 'running'
                    CHECK (status IN ('running', 'passed', 'failed')),
    details         JSONB,
    started_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

-- 1. 事件序列一致性检查
CREATE OR REPLACE FUNCTION check_event_sequence_consistency(
    p_project_id UUID
) RETURNS JSONB AS $$
DECLARE
    v_gaps BIGINT[];
    v_duplicates BIGINT[];
    v_result JSONB;
BEGIN
    -- 检查序列号是否有缺失
    WITH sequence_check AS (
        SELECT version_seq,
               LAG(version_seq) OVER (ORDER BY version_seq) as prev_seq
        FROM events
        WHERE project_id = p_project_id
    )
    SELECT array_agg(version_seq) INTO v_gaps
    FROM sequence_check
    WHERE prev_seq IS NOT NULL AND version_seq != prev_seq + 1;
    
    -- 检查序列号是否有重复
    SELECT array_agg(version_seq) INTO v_duplicates
    FROM (
        SELECT version_seq, COUNT(*) as cnt
        FROM events
        WHERE project_id = p_project_id
        GROUP BY version_seq
        HAVING COUNT(*) > 1
    ) t;
    
    v_result := jsonb_build_object(
        'check_name', 'event_sequence_consistency',
        'gaps_found', COALESCE(array_length(v_gaps, 1), 0),
        'gaps', v_gaps,
        'duplicates_found', COALESCE(array_length(v_duplicates, 1), 0),
        'duplicates', v_duplicates,
        'is_consistent', v_gaps IS NULL AND v_duplicates IS NULL
    );
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- 2. 元素版本一致性检查
CREATE OR REPLACE FUNCTION check_element_version_consistency(
    p_project_id UUID
) RETURNS JSONB AS $$
DECLARE
    v_orphaned_versions INT;
    v_missing_versions INT;
    v_result JSONB;
BEGIN
    -- 检查孤儿版本 (没有对应元素的版本)
    SELECT COUNT(*) INTO v_orphaned_versions
    FROM element_versions ev
    LEFT JOIN elements e ON ev.element_id = e.element_id
    WHERE e.element_id IS NULL OR e.deleted_at IS NOT NULL;
    
    -- 检查缺失版本号
    WITH version_gaps AS (
        SELECT element_id, version_number,
               LAG(version_number) OVER (PARTITION BY element_id ORDER BY version_number) as prev_version
        FROM element_versions ev
        JOIN elements e ON ev.element_id = e.element_id
        WHERE e.project_id = p_project_id
    )
    SELECT COUNT(*) INTO v_missing_versions
    FROM version_gaps
    WHERE prev_version IS NOT NULL AND version_number != prev_version + 1;
    
    v_result := jsonb_build_object(
        'check_name', 'element_version_consistency',
        'orphaned_versions', v_orphaned_versions,
        'version_gaps', v_missing_versions,
        'is_consistent', v_orphaned_versions = 0 AND v_missing_versions = 0
    );
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- 3. 事件与版本一致性检查
CREATE OR REPLACE FUNCTION check_event_version_consistency(
    p_project_id UUID
) RETURNS JSONB AS $$
DECLARE
    v_mismatched INT;
    v_result JSONB;
BEGIN
    -- 检查事件与版本记录是否匹配
    SELECT COUNT(*) INTO v_mismatched
    FROM events e
    LEFT JOIN element_versions ev ON e.event_id = ev.event_id
    WHERE e.project_id = p_project_id
    AND e.event_type LIKE 'element.%'
    AND ev.version_id IS NULL;
    
    v_result := jsonb_build_object(
        'check_name', 'event_version_consistency',
        'events_without_versions', v_mismatched,
        'is_consistent', v_mismatched = 0
    );
    
    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- 综合一致性检查
CREATE OR REPLACE FUNCTION run_consistency_checks(
    p_project_id UUID DEFAULT NULL
) RETURNS TABLE (
    check_name TEXT,
    status TEXT,
    details JSONB
) AS $$
DECLARE
    v_check_id UUID;
BEGIN
    -- 创建检查记录
    INSERT INTO consistency_checks (project_id, check_type)
    VALUES (p_project_id, 'full')
    RETURNING check_id INTO v_check_id;
    
    -- 执行各项检查
    RETURN QUERY
    SELECT 'event_sequence'::TEXT, 
           CASE WHEN (result->>'is_consistent')::boolean THEN 'passed' ELSE 'failed' END,
           result
    FROM (SELECT check_event_sequence_consistency(p_project_id) as result) t;
    
    RETURN QUERY
    SELECT 'element_version'::TEXT,
           CASE WHEN (result->>'is_consistent')::boolean THEN 'passed' ELSE 'failed' END,
           result
    FROM (SELECT check_element_version_consistency(p_project_id) as result) t;
    
    RETURN QUERY
    SELECT 'event_version'::TEXT,
           CASE WHEN (result->>'is_consistent')::boolean THEN 'passed' ELSE 'failed' END,
           result
    FROM (SELECT check_event_version_consistency(p_project_id) as result) t;
    
    -- 更新检查记录
    UPDATE consistency_checks
    SET status = 'completed',
        completed_at = NOW()
    WHERE check_id = v_check_id;
END;
$$ LANGUAGE plpgsql;
```

#### 4.5.2 分布式锁实现 (Redis)

```sql
-- ============================================
-- Redis分布式锁 (用于应用层)
-- ============================================

/*
-- Redis Lua脚本: 获取锁
-- SET resource_name my_random_value NX PX 30000

-- 获取锁
local function acquire_lock(redis, lock_key, lock_value, expire_ms)
    return redis:set(lock_key, lock_value, "NX", "PX", expire_ms)
end

-- 释放锁 (使用Lua保证原子性)
local release_lock_script = [[
    if redis.call("get", KEYS[1]) == ARGV[1] then
        return redis.call("del", KEYS[1])
    else
        return 0
    end
]]

-- 续约锁
local extend_lock_script = [[
    if redis.call("get", KEYS[1]) == ARGV[1] then
        return redis.call("pexpire", KEYS[1], ARGV[2])
    else
        return 0
    end
]]

-- 锁键命名规范
-- lock:project:{project_id}:element:{element_id}
-- lock:project:{project_id}:batch:{batch_id}
-- lock:project:{project_id}:global
*/

-- PostgreSQL Advisory Lock (轻量级锁)
CREATE OR REPLACE FUNCTION acquire_advisory_lock(
    p_lock_id BIGINT
) RETURNS BOOLEAN AS $$
BEGIN
    PERFORM pg_advisory_lock(p_lock_id);
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION release_advisory_lock(
    p_lock_id BIGINT
) RETURNS BOOLEAN AS $$
BEGIN
    PERFORM pg_advisory_unlock(p_lock_id);
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- 生成锁ID (基于项目ID和元素ID)
CREATE OR REPLACE FUNCTION get_lock_id(
    p_project_id UUID,
    p_element_id UUID DEFAULT NULL
) RETURNS BIGINT AS $$
BEGIN
    IF p_element_id IS NULL THEN
        -- 项目级锁
        RETURN ('x' || substr(md5(p_project_id::text), 1, 16))::bit(64)::bigint;
    ELSE
        -- 元素级锁
        RETURN ('x' || substr(md5(p_project_id::text || p_element_id::text), 1, 16))::bit(64)::bigint;
    END IF;
END;
$$ LANGUAGE plpgsql;
```

---


## 5. 性能测试方案

### 5.1 测试环境规划

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       性能测试环境架构                                        │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        测试环境配置                                  │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   ┌─────────────┐     ┌─────────────┐     ┌─────────────┐          │
    │   │  负载生成器  │────▶│  应用服务器  │────▶│  数据库集群  │          │
    │   │  (JMeter)   │     │  (3节点)    │     │  (3节点)    │          │
    │   └─────────────┘     └─────────────┘     └──────┬──────┘          │
    │                                                   │                 │
    │                                            ┌──────▼──────┐          │
    │                                            │  Redis集群  │          │
    │                                            │  (3主3从)   │          │
    │                                            └─────────────┘          │
    │                                                                     │
    │   硬件配置:                                                         │
    │   - 数据库节点: 8 vCPU, 32GB RAM, 500GB SSD                         │
    │   - 应用节点: 4 vCPU, 16GB RAM                                      │
    │   - 网络: 10Gbps 内网                                               │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

### 5.2 几何数据查询性能测试

#### 5.2.1 测试用例设计

```sql
-- ============================================
-- 几何查询性能测试用例
-- ============================================

-- 测试1: 空间范围查询性能
-- 目标: 查询指定3D空间范围内的所有构件
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT element_id, element_type, bbox_3d
FROM elements
WHERE project_id = 'test-project-id'
AND bbox_3d && ST_3DMakeBox(
    ST_MakePoint(0, 0, 0),
    ST_MakePoint(100, 100, 50)
);

-- 预期结果: < 50ms (有索引)

-- 测试2: 精确几何查询性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT ev.element_id, ev.geometry_3d, ev.properties
FROM element_versions ev
JOIN elements e ON ev.element_id = e.element_id
WHERE e.project_id = 'test-project-id'
AND e.element_type = 'wall'
AND ev.version_number = (
    SELECT MAX(version_number) 
    FROM element_versions 
    WHERE element_id = ev.element_id
);

-- 预期结果: < 100ms

-- 测试3: 空间关系查询 (相邻构件)
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT e1.element_id, e2.element_id,
       ST_3DDistance(e1.bbox_3d, e2.bbox_3d) as distance
FROM elements e1
JOIN elements e2 ON e1.element_id < e2.element_id
WHERE e1.project_id = 'test-project-id'
AND e2.project_id = 'test-project-id'
AND ST_3DDWithin(e1.bbox_3d, e2.bbox_3d, 1.0);

-- 预期结果: < 500ms (1000构件)

-- 测试4: 复杂几何运算性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT 
    element_id,
    ST_Volume(geometry_3d) as volume,
    ST_Area(ST_3DPerimeter(geometry_3d)) as surface_area
FROM element_versions
WHERE element_id IN (
    SELECT element_id FROM elements 
    WHERE project_id = 'test-project-id'
    LIMIT 100
);

-- 预期结果: < 200ms (100个构件)
```

#### 5.2.2 性能测试脚本

```python
# ============================================
# 几何查询性能测试脚本 (Python + psycopg2)
# ============================================

"""
性能测试脚本 - 几何数据查询
"""
import psycopg2
import time
import statistics
from concurrent.futures import ThreadPoolExecutor
import json

class GeometryQueryPerformanceTest:
    def __init__(self, db_config):
        self.db_config = db_config
        self.results = []
    
    def test_spatial_range_query(self, project_id, iterations=100):
        """测试空间范围查询性能"""
        query = """
        SELECT element_id, element_type 
        FROM elements
        WHERE project_id = %s
        AND bbox_3d && ST_3DMakeBox(
            ST_MakePoint(%s, %s, %s),
            ST_MakePoint(%s, %s, %s)
        )
        """
        
        times = []
        conn = psycopg2.connect(**self.db_config)
        cur = conn.cursor()
        
        for i in range(iterations):
            # 随机生成查询范围
            x, y, z = i * 10, i * 5, i * 2
            
            start = time.perf_counter()
            cur.execute(query, (project_id, x, y, z, x+50, y+50, z+20))
            rows = cur.fetchall()
            elapsed = (time.perf_counter() - start) * 1000
            times.append(elapsed)
        
        cur.close()
        conn.close()
        
        return {
            'test_name': 'spatial_range_query',
            'iterations': iterations,
            'min_ms': min(times),
            'max_ms': max(times),
            'avg_ms': statistics.mean(times),
            'p50_ms': statistics.median(times),
            'p95_ms': sorted(times)[int(len(times)*0.95)],
            'p99_ms': sorted(times)[int(len(times)*0.99)]
        }
    
    def test_concurrent_queries(self, project_id, concurrent_users=50, queries_per_user=20):
        """测试并发查询性能"""
        def worker(user_id):
            times = []
            conn = psycopg2.connect(**self.db_config)
            cur = conn.cursor()
            
            for q in range(queries_per_user):
                start = time.perf_counter()
                cur.execute("""
                    SELECT element_id, element_type, bbox_3d
                    FROM elements
                    WHERE project_id = %s
                    LIMIT 100 OFFSET %s
                """, (project_id, q * 100))
                rows = cur.fetchall()
                elapsed = (time.perf_counter() - start) * 1000
                times.append(elapsed)
            
            cur.close()
            conn.close()
            return times
        
        all_times = []
        with ThreadPoolExecutor(max_workers=concurrent_users) as executor:
            futures = [executor.submit(worker, i) for i in range(concurrent_users)]
            for future in futures:
                all_times.extend(future.result())
        
        return {
            'test_name': 'concurrent_queries',
            'concurrent_users': concurrent_users,
            'queries_per_user': queries_per_user,
            'total_queries': len(all_times),
            'avg_ms': statistics.mean(all_times),
            'p95_ms': sorted(all_times)[int(len(all_times)*0.95)],
            'p99_ms': sorted(all_times)[int(len(all_times)*0.99)]
        }

# 测试配置
DB_CONFIG = {
    'host': 'localhost',
    'port': 5433,
    'database': 'arch_platform',
    'user': 'test_user',
    'password': 'test_pass'
}

# 验收标准
ACCEPTANCE_CRITERIA = {
    'spatial_range_query': {
        'p95_ms': 100,  # 95%查询 < 100ms
        'p99_ms': 200   # 99%查询 < 200ms
    },
    'concurrent_queries': {
        'p95_ms': 150,
        'p99_ms': 300
    }
}
```

### 5.3 版本历史查询性能测试

#### 5.3.1 测试用例设计

```sql
-- ============================================
-- 版本历史查询性能测试
-- ============================================

-- 测试1: 单元素版本历史查询
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT version_number, created_at, change_summary
FROM element_versions
WHERE element_id = 'test-element-id'
ORDER BY version_number DESC;

-- 预期结果: < 20ms (1000个版本)

-- 测试2: 项目时间旅行查询
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT * FROM get_project_at_version(
    'test-project-id',
    5000  -- 目标版本
);

-- 预期结果: < 500ms (10万构件)

-- 测试3: 事件流重建查询
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT event_id, event_type, payload, executed_at
FROM events
WHERE project_id = 'test-project-id'
AND version_seq BETWEEN 1000 AND 2000
ORDER BY version_seq;

-- 预期结果: < 100ms (1000个事件)

-- 测试4: 快照加载性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT snapshot_data
FROM snapshots
WHERE project_id = 'test-project-id'
ORDER BY version_to DESC
LIMIT 1;

-- 预期结果: < 50ms
```

#### 5.3.2 版本历史性能测试脚本

```python
# ============================================
# 版本历史查询性能测试
# ============================================

class VersionHistoryPerformanceTest:
    def __init__(self, db_config):
        self.db_config = db_config
    
    def test_element_version_history(self, element_id, iterations=100):
        """测试单元素版本历史查询"""
        query = """
        SELECT version_number, created_at, change_summary
        FROM element_versions
        WHERE element_id = %s
        ORDER BY version_number DESC
        """
        
        times = []
        conn = psycopg2.connect(**self.db_config)
        cur = conn.cursor()
        
        for _ in range(iterations):
            start = time.perf_counter()
            cur.execute(query, (element_id,))
            rows = cur.fetchall()
            elapsed = (time.perf_counter() - start) * 1000
            times.append(elapsed)
        
        cur.close()
        conn.close()
        
        return {
            'test_name': 'element_version_history',
            'iterations': iterations,
            'avg_ms': statistics.mean(times),
            'p95_ms': sorted(times)[int(len(times)*0.95)]
        }
    
    def test_time_travel_query(self, project_id, target_versions, iterations=50):
        """测试时间旅行查询性能"""
        times = []
        conn = psycopg2.connect(**self.db_config)
        cur = conn.cursor()
        
        for target_version in target_versions[:iterations]:
            start = time.perf_counter()
            cur.execute(
                "SELECT * FROM get_project_at_version(%s, %s)",
                (project_id, target_version)
            )
            rows = cur.fetchall()
            elapsed = (time.perf_counter() - start) * 1000
            times.append(elapsed)
        
        cur.close()
        conn.close()
        
        return {
            'test_name': 'time_travel_query',
            'target_versions': len(target_versions),
            'avg_ms': statistics.mean(times),
            'p95_ms': sorted(times)[int(len(times)*0.95)]
        }

# 验收标准
VERSION_ACCEPTANCE_CRITERIA = {
    'element_version_history': {
        'p95_ms': 50,
        'p99_ms': 100
    },
    'time_travel_query': {
        'p95_ms': 500,
        'p99_ms': 1000
    }
}
```

### 5.4 并发写入性能测试

#### 5.4.1 测试用例设计

```sql
-- ============================================
-- 并发写入性能测试
-- ============================================

-- 测试1: 单元素并发更新冲突率
-- 使用多个会话同时更新同一元素

-- 会话A
BEGIN;
SELECT lock_version FROM elements WHERE element_id = 'test-id' FOR UPDATE;
-- ... 业务处理 ...
UPDATE elements SET lock_version = lock_version + 1 WHERE element_id = 'test-id';
COMMIT;

-- 测试2: 批量插入性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
WITH inserted AS (
    INSERT INTO elements (element_id, project_id, element_type, bbox_3d, created_by)
    SELECT 
        gen_random_uuid(),
        'test-project-id',
        'wall',
        ST_3DMakeBox(ST_MakePoint(i*10, 0, 0), ST_MakePoint(i*10+5, 3, 3)),
        'test-user'
    FROM generate_series(1, 1000) i
    RETURNING element_id
)
SELECT COUNT(*) FROM inserted;

-- 预期结果: > 500 inserts/sec

-- 测试3: 事件写入吞吐量
-- 批量事件插入
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
INSERT INTO events (project_id, tenant_id, version_seq, event_type, payload, executed_by)
SELECT 
    'test-project-id',
    'test-tenant-id',
    next_version_seq,
    'element.updated',
    '{}'::jsonb,
    'test-user'
FROM generate_series(1, 1000) i,
     LATERAL (SELECT (version_seq + i) as next_version_seq FROM projects 
              WHERE project_id = 'test-project-id' FOR UPDATE) p;

-- 预期结果: > 1000 events/sec
```

#### 5.4.2 并发写入测试脚本

```python
# ============================================
# 并发写入性能测试
# ============================================

import threading
import queue
from concurrent.futures import ThreadPoolExecutor, as_completed

class ConcurrentWriteTest:
    def __init__(self, db_config):
        self.db_config = db_config
        self.conflict_count = 0
        self.success_count = 0
        self.lock = threading.Lock()
    
    def test_concurrent_element_updates(
        self, 
        project_id, 
        element_id, 
        concurrent_users=100,
        updates_per_user=10
    ):
        """测试并发元素更新冲突率"""
        
        def update_worker(user_id):
            local_success = 0
            local_conflict = 0
            
            conn = psycopg2.connect(**self.db_config)
            cur = conn.cursor()
            
            for i in range(updates_per_user):
                try:
                    # 获取当前版本
                    cur.execute(
                        "SELECT lock_version FROM elements WHERE element_id = %s",
                        (element_id,)
                    )
                    row = cur.fetchone()
                    if not row:
                        continue
                    
                    current_version = row[0]
                    
                    # 尝试更新
                    cur.execute("""
                        UPDATE elements 
                        SET lock_version = lock_version + 1,
                            updated_at = NOW()
                        WHERE element_id = %s AND lock_version = %s
                    """, (element_id, current_version))
                    
                    if cur.rowcount > 0:
                        conn.commit()
                        local_success += 1
                    else:
                        conn.rollback()
                        local_conflict += 1
                        
                except Exception as e:
                    conn.rollback()
                    local_conflict += 1
            
            cur.close()
            conn.close()
            
            with self.lock:
                self.success_count += local_success
                self.conflict_count += local_conflict
            
            return local_success, local_conflict
        
        # 执行并发测试
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=concurrent_users) as executor:
            futures = [
                executor.submit(update_worker, i) 
                for i in range(concurrent_users)
            ]
            for future in as_completed(futures):
                future.result()
        
        elapsed = time.time() - start_time
        total_ops = self.success_count + self.conflict_count
        
        return {
            'test_name': 'concurrent_element_updates',
            'concurrent_users': concurrent_users,
            'updates_per_user': updates_per_user,
            'total_attempts': total_ops,
            'success_count': self.success_count,
            'conflict_count': self.conflict_count,
            'conflict_rate': self.conflict_count / total_ops if total_ops > 0 else 0,
            'throughput_ops_sec': total_ops / elapsed,
            'elapsed_seconds': elapsed
        }
    
    def test_event_write_throughput(
        self,
        project_id,
        total_events=10000,
        concurrent_writers=50
    ):
        """测试事件写入吞吐量"""
        
        def event_worker(writer_id, event_count):
            conn = psycopg2.connect(**self.db_config)
            cur = conn.cursor()
            
            for i in range(event_count):
                cur.execute("""
                    INSERT INTO events 
                    (project_id, tenant_id, version_seq, event_type, payload, executed_by)
                    SELECT %s, tenant_id, version_seq + 1, 'test.event', '{}', %s
                    FROM projects WHERE project_id = %s
                """, (project_id, f'writer-{writer_id}', project_id))
            
            conn.commit()
            cur.close()
            conn.close()
            
            return event_count
        
        events_per_writer = total_events // concurrent_writers
        
        start_time = time.time()
        
        with ThreadPoolExecutor(max_workers=concurrent_writers) as executor:
            futures = [
                executor.submit(event_worker, i, events_per_writer)
                for i in range(concurrent_writers)
            ]
            total_written = sum(f.result() for f in as_completed(futures))
        
        elapsed = time.time() - start_time
        
        return {
            'test_name': 'event_write_throughput',
            'total_events': total_written,
            'concurrent_writers': concurrent_writers,
            'throughput_events_sec': total_written / elapsed,
            'elapsed_seconds': elapsed
        }

# 验收标准
WRITE_ACCEPTANCE_CRITERIA = {
    'concurrent_element_updates': {
        'conflict_rate_max': 0.1,  # 冲突率 < 10%
        'throughput_ops_sec_min': 500
    },
    'event_write_throughput': {
        'throughput_events_sec_min': 1000
    }
}
```

### 5.5 大数据量测试 (100万+构件)

#### 5.5.1 数据生成脚本

```sql
-- ============================================
-- 大数据量测试数据生成
-- ============================================

-- 生成100万构件的测试数据
CREATE OR REPLACE FUNCTION generate_test_data(
    p_project_id UUID,
    p_element_count INT DEFAULT 1000000
) RETURNS VOID AS $$
DECLARE
    v_batch_size INT := 10000;
    v_batches INT;
    v_tenant_id UUID;
BEGIN
    SELECT tenant_id INTO v_tenant_id 
    FROM projects WHERE project_id = p_project_id;
    
    v_batches := CEIL(p_element_count::float / v_batch_size);
    
    FOR i IN 1..v_batches LOOP
        -- 批量插入元素
        INSERT INTO elements (
            element_id, project_id, element_type, 
            bbox_2d, bbox_3d, created_by
        )
        SELECT 
            gen_random_uuid(),
            p_project_id,
            CASE (random() * 8)::int 
                WHEN 0 THEN 'wall'
                WHEN 1 THEN 'door'
                WHEN 2 THEN 'window'
                WHEN 3 THEN 'floor'
                WHEN 4 THEN 'column'
                WHEN 5 THEN 'beam'
                WHEN 6 THEN 'stair'
                ELSE 'furniture'
            END,
            ST_MakeEnvelope(
                random() * 1000, random() * 1000,
                random() * 1000 + 10, random() * 1000 + 10,
                3857
            ),
            ST_3DMakeBox(
                ST_MakePoint(random() * 1000, random() * 1000, 0),
                ST_MakePoint(random() * 1000 + 10, random() * 1000 + 10, random() * 50)
            ),
            'test-user'::uuid
        FROM generate_series(1, LEAST(v_batch_size, p_element_count - (i-1)*v_batch_size));
        
        -- 为每个元素创建初始版本
        INSERT INTO element_versions (
            element_id, version_number, geometry_3d, 
            geometry_hash, properties, event_id, created_by
        )
        SELECT 
            e.element_id,
            1,
            ST_SetSRID(ST_MakePoint(
                ST_XMin(e.bbox_3d) + 5,
                ST_YMin(e.bbox_3d) + 5,
                ST_ZMax(e.bbox_3d) / 2
            ), 3857),
            md5(random()::text),
            jsonb_build_object(
                'name', 'Element ' || row_number() OVER (),
                'material', 'concrete',
                'cost', random() * 1000
            ),
            gen_random_uuid(),
            'test-user'::uuid
        FROM elements e
        WHERE e.project_id = p_project_id
        AND e.created_at > NOW() - INTERVAL '1 minute'
        AND NOT EXISTS (
            SELECT 1 FROM element_versions ev 
            WHERE ev.element_id = e.element_id
        );
        
        -- 更新元素的当前版本
        UPDATE elements e
        SET current_version = ev.version_id
        FROM element_versions ev
        WHERE e.element_id = ev.element_id
        AND e.project_id = p_project_id
        AND e.current_version IS NULL;
        
        RAISE NOTICE 'Batch %/% completed', i, v_batches;
        COMMIT;
    END LOOP;
    
    -- 更新项目统计
    UPDATE projects 
    SET stats = jsonb_set(stats, '{element_count}', to_jsonb(p_element_count))
    WHERE project_id = p_project_id;
    
    RAISE NOTICE 'Generated % elements for project %', p_element_count, p_project_id;
END;
$$ LANGUAGE plpgsql;
```

#### 5.5.2 大数据量性能测试

```python
# ============================================
# 大数据量性能测试
# ============================================

class LargeScalePerformanceTest:
    def __init__(self, db_config):
        self.db_config = db_config
    
    def test_query_performance_at_scale(self, project_id):
        """测试大规模数据下的查询性能"""
        conn = psycopg2.connect(**self.db_config)
        cur = conn.cursor()
        
        results = {}
        
        # 测试1: 计数查询
        start = time.perf_counter()
        cur.execute(
            "SELECT COUNT(*) FROM elements WHERE project_id = %s",
            (project_id,)
        )
        count = cur.fetchone()[0]
        results['count_query_ms'] = (time.perf_counter() - start) * 1000
        results['total_elements'] = count
        
        # 测试2: 分页查询
        start = time.perf_counter()
        cur.execute("""
            SELECT element_id, element_type, bbox_3d
            FROM elements
            WHERE project_id = %s
            ORDER BY element_id
            LIMIT 100 OFFSET 0
        """, (project_id,))
        rows = cur.fetchall()
        results['pagination_query_ms'] = (time.perf_counter() - start) * 1000
        
        # 测试3: 聚合查询
        start = time.perf_counter()
        cur.execute("""
            SELECT element_type, COUNT(*) as count
            FROM elements
            WHERE project_id = %s
            GROUP BY element_type
        """, (project_id,))
        rows = cur.fetchall()
        results['aggregation_query_ms'] = (time.perf_counter() - start) * 1000
        
        # 测试4: 空间范围查询
        start = time.perf_counter()
        cur.execute("""
            SELECT element_id, element_type
            FROM elements
            WHERE project_id = %s
            AND bbox_3d && ST_3DMakeBox(
                ST_MakePoint(0, 0, 0),
                ST_MakePoint(100, 100, 50)
            )
        """, (project_id,))
        rows = cur.fetchall()
        results['spatial_query_ms'] = (time.perf_counter() - start) * 1000
        results['spatial_query_results'] = len(rows)
        
        cur.close()
        conn.close()
        
        return results
    
    def test_storage_efficiency(self, project_id):
        """测试存储效率"""
        conn = psycopg2.connect(**self.db_config)
        cur = conn.cursor()
        
        # 获取表大小
        cur.execute("""
            SELECT 
                schemaname,
                relname,
                pg_size_pretty(pg_total_relation_size(relid)) as total_size,
                pg_total_relation_size(relid) as size_bytes,
                n_live_tup as row_count
            FROM pg_stat_user_tables
            WHERE relname IN ('elements', 'element_versions', 'events')
            ORDER BY pg_total_relation_size(relid) DESC
        """)
        
        storage_stats = cur.fetchall()
        
        cur.close()
        conn.close()
        
        return {
            'storage_stats': storage_stats,
            'storage_efficiency_per_element': 
                sum(s[3] for s in storage_stats) / 
                max(sum(s[4] for s in storage_stats), 1)
        }

# 大数据量验收标准
SCALE_ACCEPTANCE_CRITERIA = {
    'count_query_ms': 100,      # 计数查询 < 100ms
    'pagination_query_ms': 50,  # 分页查询 < 50ms
    'aggregation_query_ms': 500, # 聚合查询 < 500ms
    'spatial_query_ms': 200,    # 空间查询 < 200ms
}
```

---


## 6. 多租户隔离验证

### 6.1 多租户架构设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       多租户隔离架构                                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                         应用层                                       │
    │   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐               │
    │   │ 租户A   │  │ 租户B   │  │ 租户C   │  │ 租户D   │               │
    │   │ 用户    │  │ 用户    │  │ 用户    │  │ 用户    │               │
    │   └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘               │
    │        │            │            │            │                     │
    │        └────────────┴────────────┴────────────┘                     │
    │                         │                                          │
    │                  ┌──────▼──────┐                                   │
    │                  │ 租户上下文   │                                   │
    │                  │ (Tenant ID) │                                   │
    │                  └──────┬──────┘                                   │
    └─────────────────────────┼─────────────────────────────────────────┘
                              │
    ┌─────────────────────────┼─────────────────────────────────────────┐
    │                         ▼                                         │
    │                    数据访问层                                      │
    │   ┌─────────────────────────────────────────────────────────┐    │
    │   │  Row Level Security (RLS)                               │    │
    │   │  - 自动过滤租户数据                                      │    │
    │   │  - 防止跨租户访问                                        │    │
    │   └─────────────────────────────────────────────────────────┘    │
    │                              │                                    │
    │                         ┌────▼────┐                               │
    │                         │ 数据库   │                               │
    │                         └─────────┘                               │
    └───────────────────────────────────────────────────────────────────┘
```

### 6.2 Schema级隔离验证

#### 6.2.1 Schema隔离方案

```sql
-- ============================================
-- Schema级隔离方案 (可选方案)
-- ============================================

-- 方案: 每个租户一个Schema
-- 优点: 完全隔离，可独立备份/恢复
-- 缺点: 维护复杂，跨租户查询困难

-- 创建租户Schema
CREATE OR REPLACE FUNCTION create_tenant_schema(p_tenant_id UUID)
RETURNS VOID AS $$
DECLARE
    v_schema_name TEXT;
BEGIN
    v_schema_name := 'tenant_' || replace(p_tenant_id::text, '-', '_');
    
    -- 创建Schema
    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema_name);
    
    -- 创建租户表
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.projects (
            project_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            created_at TIMESTAMPTZ DEFAULT NOW()
        )
    ', v_schema_name);
    
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.elements (
            element_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            project_id UUID NOT NULL,
            element_type VARCHAR(50) NOT NULL,
            geometry_3d GEOMETRY(GEOMETRYZ, 3857),
            created_at TIMESTAMPTZ DEFAULT NOW()
        )
    ', v_schema_name);
    
    -- 设置权限
    EXECUTE format('GRANT USAGE ON SCHEMA %I TO app_user', v_schema_name);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO app_user', v_schema_name);
    
END;
$$ LANGUAGE plpgsql;

-- 切换租户Schema的函数
CREATE OR REPLACE FUNCTION set_tenant_schema(p_tenant_id UUID)
RETURNS VOID AS $$
DECLARE
    v_schema_name TEXT;
BEGIN
    v_schema_name := 'tenant_' || replace(p_tenant_id::text, '-', '_');
    EXECUTE format('SET search_path TO %I, public', v_schema_name);
END;
$$ LANGUAGE plpgsql;
```

#### 6.2.2 Schema隔离测试

```sql
-- ============================================
-- Schema隔离验证测试
-- ============================================

-- 测试1: 创建租户Schema
SELECT create_tenant_schema('11111111-1111-1111-1111-111111111111');
SELECT create_tenant_schema('22222222-2222-2222-2222-222222222222');

-- 测试2: 验证Schema隔离
-- 切换到租户1的Schema
SELECT set_tenant_schema('11111111-1111-1111-1111-111111111111');

-- 插入数据到租户1
INSERT INTO projects (name) VALUES ('Tenant 1 Project');

-- 切换到租户2的Schema
SELECT set_tenant_schema('22222222-2222-2222-2222-222222222222');

-- 验证租户2看不到租户1的数据
SELECT COUNT(*) FROM projects;  -- 应该返回0

-- 插入数据到租户2
INSERT INTO projects (name) VALUES ('Tenant 2 Project');

-- 测试3: 验证权限隔离
-- 尝试从租户2访问租户1的Schema (应该失败)
SELECT * FROM tenant_11111111_1111_1111_1111_111111111111.projects;
-- 预期: ERROR: permission denied for schema tenant_11111111_1111_1111_1111_111111111111
```

### 6.3 行级安全(RLS)验证

#### 6.3.1 RLS策略配置

```sql
-- ============================================
-- 行级安全(RLS)实现
-- ============================================

-- 启用RLS
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE elements ENABLE ROW LEVEL SECURITY;
ALTER TABLE element_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE events ENABLE ROW LEVEL SECURITY;
ALTER TABLE snapshots ENABLE ROW LEVEL SECURITY;

-- 创建租户ID设置函数
CREATE OR REPLACE FUNCTION current_tenant_id()
RETURNS UUID AS $$
BEGIN
    -- 从会话变量获取当前租户ID
    RETURN NULLIF(current_setting('app.current_tenant_id', TRUE), '')::UUID;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 设置当前租户ID
CREATE OR REPLACE FUNCTION set_current_tenant_id(p_tenant_id UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::text, FALSE);
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- RLS策略定义
-- ============================================

-- 项目表RLS策略
CREATE POLICY tenant_isolation_projects ON projects
    FOR ALL
    TO app_user
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- 构件表RLS策略 (通过项目关联)
CREATE POLICY tenant_isolation_elements ON elements
    FOR ALL
    TO app_user
    USING (
        project_id IN (
            SELECT project_id FROM projects 
            WHERE tenant_id = current_tenant_id()
        )
    )
    WITH CHECK (
        project_id IN (
            SELECT project_id FROM projects 
            WHERE tenant_id = current_tenant_id()
        )
    );

-- 构件版本表RLS策略
CREATE POLICY tenant_isolation_element_versions ON element_versions
    FOR ALL
    TO app_user
    USING (
        element_id IN (
            SELECT e.element_id FROM elements e
            JOIN projects p ON e.project_id = p.project_id
            WHERE p.tenant_id = current_tenant_id()
        )
    );

-- 事件表RLS策略
CREATE POLICY tenant_isolation_events ON events
    FOR ALL
    TO app_user
    USING (
        tenant_id = current_tenant_id()
    )
    WITH CHECK (
        tenant_id = current_tenant_id()
    );

-- 快照表RLS策略
CREATE POLICY tenant_isolation_snapshots ON snapshots
    FOR ALL
    TO app_user
    USING (
        project_id IN (
            SELECT project_id FROM projects 
            WHERE tenant_id = current_tenant_id()
        )
    );

-- 管理员绕过RLS
CREATE POLICY admin_bypass_projects ON projects
    FOR ALL
    TO admin_user
    USING (TRUE);

-- ============================================
-- 租户成员权限策略
-- ============================================

-- 创建项目成员权限检查函数
CREATE OR REPLACE FUNCTION has_project_permission(
    p_project_id UUID,
    p_permission TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    v_user_id UUID;
    v_member_permissions JSONB;
BEGIN
    -- 获取当前用户ID
    v_user_id := NULLIF(current_setting('app.current_user_id', TRUE), '')::UUID;
    
    IF v_user_id IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- 检查项目成员权限
    SELECT permissions INTO v_member_permissions
    FROM project_members
    WHERE project_id = p_project_id
    AND user_id = v_user_id;
    
    IF v_member_permissions IS NULL THEN
        RETURN FALSE;
    END IF;
    
    RETURN v_member_permissions ? p_permission;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 基于权限的RLS策略
CREATE POLICY permission_based_elements ON elements
    FOR UPDATE
    TO app_user
    USING (
        project_id IN (
            SELECT project_id FROM projects 
            WHERE tenant_id = current_tenant_id()
        )
        AND has_project_permission(project_id, 'element:write')
    );
```

#### 6.3.2 RLS测试验证

```sql
-- ============================================
-- RLS验证测试
-- ============================================

-- 测试1: 基本租户隔离
-- 设置租户上下文
SELECT set_current_tenant_id('11111111-1111-1111-1111-111111111111');

-- 查询应该只返回租户1的数据
SELECT project_id, name FROM projects;

-- 测试2: 跨租户访问防护
-- 尝试直接访问其他租户的数据 (应该返回空)
SELECT * FROM projects WHERE tenant_id = '22222222-2222-2222-2222-222222222222';

-- 测试3: 插入数据自动关联租户
-- 应该自动设置tenant_id
INSERT INTO projects (tenant_id, name, created_by)
VALUES (current_tenant_id(), 'New Project', 'user-1'::uuid);

-- 测试4: 权限检查
-- 设置用户上下文
SELECT set_config('app.current_user_id', 'user-1', FALSE);

-- 检查用户是否有项目写入权限
SELECT has_project_permission('project-id-1', 'element:write');

-- 测试5: 绕过RLS (管理员)
-- 切换到管理员用户
SET ROLE admin_user;

-- 管理员可以看到所有租户数据
SELECT tenant_id, COUNT(*) as project_count 
FROM projects 
GROUP BY tenant_id;

-- 恢复普通用户
SET ROLE app_user;
```

### 6.4 跨租户数据访问防护验证

#### 6.4.1 防护机制设计

```sql
-- ============================================
-- 跨租户访问防护机制
-- ============================================

-- 1. 触发器防护 - 阻止跨租户操作
CREATE OR REPLACE FUNCTION prevent_cross_tenant_access()
RETURNS TRIGGER AS $$
DECLARE
    v_current_tenant UUID;
    v_target_tenant UUID;
BEGIN
    v_current_tenant := current_tenant_id();
    
    -- 获取目标记录的租户ID
    IF TG_OP = 'DELETE' OR TG_OP = 'UPDATE' THEN
        IF TG_TABLE_NAME = 'projects' THEN
            v_target_tenant := OLD.tenant_id;
        ELSIF TG_TABLE_NAME = 'elements' THEN
            SELECT tenant_id INTO v_target_tenant
            FROM projects WHERE project_id = OLD.project_id;
        END IF;
    END IF;
    
    -- 检查跨租户访问
    IF v_current_tenant IS NOT NULL AND 
       v_target_tenant IS NOT NULL AND 
       v_current_tenant != v_target_tenant THEN
        RAISE EXCEPTION 'Cross-tenant access denied: current=%, target=%',
            v_current_tenant, v_target_tenant;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 应用触发器
CREATE TRIGGER trg_prevent_cross_tenant_projects
    BEFORE UPDATE OR DELETE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION prevent_cross_tenant_access();

-- 2. 审计日志 - 记录跨租户访问尝试
CREATE TABLE cross_tenant_access_audit (
    audit_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempted_at    TIMESTAMPTZ DEFAULT NOW(),
    user_id         UUID,
    source_tenant   UUID,
    target_tenant   UUID,
    target_table    TEXT,
    target_record   UUID,
    action_type     TEXT,
    blocked         BOOLEAN DEFAULT TRUE,
    client_ip       INET
);

-- 审计触发器
CREATE OR REPLACE FUNCTION audit_cross_tenant_access()
RETURNS TRIGGER AS $$
DECLARE
    v_current_tenant UUID;
    v_target_tenant UUID;
BEGIN
    v_current_tenant := current_tenant_id();
    
    IF TG_TABLE_NAME = 'projects' THEN
        v_target_tenant := NEW.tenant_id;
    END IF;
    
    IF v_current_tenant IS NOT NULL AND 
       v_target_tenant IS NOT NULL AND 
       v_current_tenant != v_target_tenant THEN
        
        INSERT INTO cross_tenant_access_audit (
            user_id, source_tenant, target_tenant,
            target_table, target_record, action_type
        ) VALUES (
            NULLIF(current_setting('app.current_user_id', TRUE), '')::UUID,
            v_current_tenant,
            v_target_tenant,
            TG_TABLE_NAME,
            COALESCE(NEW.project_id, OLD.project_id),
            TG_OP
        );
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- 3. API层防护 (应用层代码)
/*
// 伪代码示例
class TenantContext {
    private static ThreadLocal<UUID> currentTenant = new ThreadLocal<>();
    
    public static void setCurrentTenant(UUID tenantId) {
        // 验证用户是否有权限访问该租户
        if (!hasTenantAccess(getCurrentUser(), tenantId)) {
            throw new UnauthorizedException("Access denied to tenant: " + tenantId);
        }
        currentTenant.set(tenantId);
        
        // 同步到数据库会话
        jdbcTemplate.execute("SELECT set_current_tenant_id('" + tenantId + "')");
    }
}

// 每个请求开始时设置租户上下文
@Interceptor
public class TenantInterceptor {
    @Override
    public void intercept(Request request) {
        UUID tenantId = extractTenantFromRequest(request);
        TenantContext.setCurrentTenant(tenantId);
    }
}
*/
```

#### 6.4.2 跨租户防护测试

```sql
-- ============================================
-- 跨租户防护验证测试
-- ============================================

-- 测试1: 尝试跨租户更新 (应该失败)
SELECT set_current_tenant_id('11111111-1111-1111-1111-111111111111');

-- 尝试更新租户2的项目 (应该被RLS阻止)
DO $$
DECLARE
    v_project_id UUID;
BEGIN
    -- 获取租户2的项目ID
    SELECT project_id INTO v_project_id
    FROM projects
    WHERE tenant_id = '22222222-2222-2222-2222-222222222222'
    LIMIT 1;
    
    -- 尝试更新 (应该失败)
    UPDATE projects SET name = 'Hacked' WHERE project_id = v_project_id;
    
    RAISE NOTICE 'Update succeeded - SECURITY ISSUE!';
EXCEPTION WHEN insufficient_privilege THEN
    RAISE NOTICE 'Update blocked by RLS - GOOD!';
END $$;

-- 测试2: 检查审计日志
SELECT * FROM cross_tenant_access_audit 
WHERE attempted_at > NOW() - INTERVAL '1 hour'
ORDER BY attempted_at DESC;

-- 测试3: 直接SQL注入尝试
-- 尝试绕过RLS
DO $$
BEGIN
    -- 尝试修改会话变量
    PERFORM set_config('app.current_tenant_id', '22222222-2222-2222-2222-222222222222', FALSE);
    
    -- 验证是否生效
    IF current_tenant_id() = '22222222-2222-2222-2222-222222222222'::UUID THEN
        RAISE NOTICE 'Tenant ID changed - check security settings!';
    END IF;
END $$;

-- 测试4: 批量操作隔离
-- 批量更新应该只影响当前租户的数据
SELECT set_current_tenant_id('11111111-1111-1111-1111-111111111111');

-- 这个更新应该只影响租户1的数据
UPDATE projects SET settings = settings || '{"updated": true}'::jsonb;

-- 验证其他租户数据未被修改
SELECT tenant_id, COUNT(*) as updated_count
FROM projects
WHERE settings ? 'updated'
GROUP BY tenant_id;
-- 预期: 只有租户1有更新记录
```

### 6.5 多租户性能影响测试

```sql
-- ============================================
-- RLS性能影响测试
-- ============================================

-- 测试1: 比较有无RLS的查询性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT * FROM projects WHERE tenant_id = '11111111-1111-1111-1111-111111111111';
-- 有RLS时应该自动应用此过滤

-- 测试2: 大量租户下的查询性能
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
SELECT tenant_id, COUNT(*) as project_count
FROM projects
GROUP BY tenant_id
ORDER BY project_count DESC
LIMIT 10;

-- 测试3: 租户数据分布查询
SELECT 
    tenant_id,
    COUNT(DISTINCT project_id) as project_count,
    COUNT(*) as element_count,
    pg_size_pretty(sum(pg_column_size(geometry_3d))) as geometry_size
FROM elements e
JOIN projects p ON e.project_id = p.project_id
GROUP BY tenant_id
ORDER BY element_count DESC;

-- 预期: RLS开销 < 5%
```

---


## 7. POC执行计划

### 7.1 测试环境搭建

#### 7.1.1 环境架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       POC测试环境架构                                         │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        测试环境拓扑                                  │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   ┌─────────────────────────────────────────────────────────────┐  │
    │   │                    Kubernetes集群                            │  │
    │   │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │  │
    │   │  │ YugabyteDB  │  │ YugabyteDB  │  │ YugabyteDB  │         │  │
    │   │  │  Master     │  │  TServer 1  │  │  TServer 2  │         │  │
    │   │  │  (Leader)   │  │             │  │             │         │  │
    │   │  └─────────────┘  └─────────────┘  └─────────────┘         │  │
    │   │                                                             │  │
    │   │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │  │
    │   │  │ Redis       │  │ Redis       │  │ Redis       │         │  │
    │   │  │ Master 1    │  │ Master 2    │  │ Master 3    │         │  │
    │   │  └─────────────┘  └─────────────┘  └─────────────┘         │  │
    │   └─────────────────────────────────────────────────────────────┘  │
    │                                                                     │
    │   ┌─────────────────────────────────────────────────────────────┐  │
    │   │                    测试工具服务器                            │  │
    │   │  - JMeter (负载测试)                                         │  │
    │   │  - Python测试脚本                                            │  │
    │   │  - 监控工具 (Prometheus + Grafana)                           │  │
    │   └─────────────────────────────────────────────────────────────┘  │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

#### 7.1.2 环境搭建脚本

```bash
#!/bin/bash
# ============================================
# POC测试环境搭建脚本
# ============================================

# 1. 部署YugabyteDB集群
deploy_yugabyte() {
    helm repo add yugabytedb https://charts.yugabyte.com
    helm repo update
    
    helm install yb-demo yugabytedb/yugabyte \
        --set resource.master.requests.cpu=2 \
        --set resource.master.requests.memory=4Gi \
        --set resource.tserver.requests.cpu=4 \
        --set resource.tserver.requests.memory=8Gi \
        --set replicas.master=3 \
        --set replicas.tserver=3 \
        --set enablePostGIS=true \
        --namespace poc-test \
        --create-namespace
}

# 2. 部署Redis集群
deploy_redis() {
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm repo update
    
    helm install redis-cluster bitnami/redis-cluster \
        --set cluster.nodes=6 \
        --set cluster.replicas=1 \
        --set password=redis-password \
        --namespace poc-test
}

# 3. 初始化数据库
init_database() {
    # 等待YugabyteDB就绪
    kubectl wait --for=condition=ready pod -l app=yb-tserver -n poc-test --timeout=300s
    
    # 获取YSQL连接信息
    YSQL_IP=$(kubectl get svc -n poc-test yb-tservers -o jsonpath='{.spec.clusterIP}')
    
    # 创建数据库和用户
    ysqlsh -h $YSQL_IP -c "CREATE DATABASE arch_platform;"
    ysqlsh -h $YSQL_IP -c "CREATE USER poc_test WITH PASSWORD 'poc_password';"
    ysqlsh -h $YSQL_IP -c "GRANT ALL PRIVILEGES ON DATABASE arch_platform TO poc_test;"
    
    # 启用PostGIS
    ysqlsh -h $YSQL_IP -d arch_platform -c "CREATE EXTENSION IF NOT EXISTS postgis;"
    ysqlsh -h $YSQL_IP -d arch_platform -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
    
    echo "Database initialized successfully!"
}

# 4. 运行数据库迁移
run_migrations() {
    # 执行所有DDL脚本
    ysqlsh -h $YSQL_IP -d arch_platform -f /migrations/01_create_tables.sql
    ysqlsh -h $YSQL_IP -d arch_platform -f /migrations/02_create_indexes.sql
    ysqlsh -h $YSQL_IP -d arch_platform -f /migrations/03_create_functions.sql
    ysqlsh -h $YSQL_IP -d arch_platform -f /migrations/04_setup_rls.sql
    
    echo "Migrations completed!"
}

# 主流程
main() {
    echo "Starting POC environment setup..."
    deploy_yugabyte
    deploy_redis
    init_database
    run_migrations
    echo "POC environment setup completed!"
}

main
```

### 7.2 测试数据集准备

#### 7.2.1 测试数据规格

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       测试数据集规格                                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        数据集规模                                    │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   数据集类型          数量              用途                        │
    │   ─────────────────────────────────────────────────────────────    │
    │   租户                10个              多租户隔离测试               │
    │   项目/租户           5-10个            功能测试                     │
    │   构件/项目           1万-10万          中等规模测试                 │
    │   构件/项目           100万             大规模性能测试               │
    │   版本/构件           平均10个          版本历史测试                 │
    │   事件/项目           1000万            Event Sourcing测试           │
    │                                                                     │
    │   总数据量估算:                                                     │
    │   - 100万构件 × 平均10版本 = 1000万版本记录                         │
    │   - 1000万事件                                                        │
    │   - 预计存储: 100-200GB                                              │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

#### 7.2.2 数据生成脚本

```sql
-- ============================================
-- 测试数据生成脚本
-- ============================================

-- 生成测试租户
INSERT INTO tenants (tenant_id, name, slug, config)
SELECT 
    gen_random_uuid(),
    'Test Tenant ' || i,
    'test-tenant-' || i,
    jsonb_build_object(
        'max_projects', 100,
        'max_storage_gb', 1000,
        'max_users', 100
    )
FROM generate_series(1, 10) i;

-- 为每个租户生成测试项目
DO $$
DECLARE
    v_tenant RECORD;
    v_project_id UUID;
BEGIN
    FOR v_tenant IN SELECT tenant_id FROM tenants LOOP
        FOR j IN 1..5 LOOP
            v_project_id := gen_random_uuid();
            
            INSERT INTO projects (
                project_id, tenant_id, name, created_by
            ) VALUES (
                v_project_id,
                v_tenant.tenant_id,
                'Project ' || j || ' - ' || v_tenant.tenant_id::text[:8],
                gen_random_uuid()
            );
            
            -- 为每个项目生成构件
            PERFORM generate_test_data(v_project_id, 10000);
        END LOOP;
    END LOOP;
END $$;

-- 生成版本历史数据
DO $$
DECLARE
    v_element RECORD;
BEGIN
    FOR v_element IN 
        SELECT element_id FROM elements 
        WHERE random() < 0.5  -- 50%的构件有多个版本
    LOOP
        -- 为每个元素生成2-10个版本
        FOR v IN 2..(2 + (random() * 8)::int) LOOP
            INSERT INTO element_versions (
                element_id, version_number, geometry_3d,
                geometry_hash, properties, event_id, created_by
            )
            SELECT 
                v_element.element_id,
                v,
                ST_SetSRID(ST_MakePoint(
                    random() * 1000,
                    random() * 1000,
                    random() * 50
                ), 3857),
                md5(random()::text),
                jsonb_build_object(
                    'version', v,
                    'updated_at', NOW()
                ),
                gen_random_uuid(),
                gen_random_uuid()
            FROM element_versions ev
            WHERE ev.element_id = v_element.element_id
            AND ev.version_number = v - 1;
        END LOOP;
    END LOOP;
END $$;
```

### 7.3 测试脚本设计

#### 7.3.1 测试脚本清单

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       POC测试脚本清单                                         │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        测试脚本目录                                  │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   /poc-tests/                                                       │
    │   ├── 01_data_model/                                                │
    │   │   ├── test_table_creation.sql      # 表结构创建测试             │
    │   │   ├── test_postgis_geometry.sql    # PostGIS几何类型测试        │
    │   │   ├── test_index_performance.sql   # 索引性能测试               │
    │   │   └── test_constraints.sql         # 约束验证测试               │
    │   │                                                                 │
    │   ├── 02_version_control/                                           │
    │   │   ├── test_event_sourcing.py       # Event Sourcing功能测试     │
    │   │   ├── test_snapshot_creation.sql   # 快照创建测试               │
    │   │   ├── test_time_travel.sql         # 时间旅行查询测试           │
    │   │   └── test_compression.sql         # 历史压缩测试               │
    │   │                                                                 │
    │   ├── 03_concurrency/                                               │
    │   │   ├── test_optimistic_lock.py      # 乐观锁测试                 │
    │   │   ├── test_mvcc_behavior.sql       # MVCC行为测试               │
    │   │   ├── test_conflict_detection.py   # 冲突检测测试               │
    │   │   └── test_consistency.py          # 一致性检查测试             │
    │   │                                                                 │
    │   ├── 04_performance/                                               │
    │   │   ├── perf_geometry_query.py       # 几何查询性能测试           │
    │   │   ├── perf_version_history.py      # 版本历史性能测试           │
    │   │   ├── perf_concurrent_write.py     # 并发写入性能测试           │
    │   │   └── perf_large_scale.py          # 大规模数据测试             │
    │   │                                                                 │
    │   ├── 05_multitenancy/                                              │
    │   │   ├── test_schema_isolation.sql    # Schema隔离测试             │
    │   │   ├── test_rls_policy.sql          # RLS策略测试                │
    │   │   └── test_cross_tenant.py         # 跨租户防护测试             │
    │   │                                                                 │
    │   └── run_all_tests.sh                 # 测试执行脚本               │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

#### 7.3.2 自动化测试执行脚本

```bash
#!/bin/bash
# ============================================
# POC自动化测试执行脚本
# ============================================

set -e

# 配置
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5433}
DB_NAME=${DB_NAME:-arch_platform}
DB_USER=${DB_USER:-poc_test}
DB_PASS=${DB_PASS:-poc_password}

TEST_RESULTS_DIR="./test-results/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$TEST_RESULTS_DIR"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试计数器
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# 执行SQL测试
run_sql_test() {
    local test_file=$1
    local test_name=$(basename "$test_file" .sql)
    
    echo -e "${YELLOW}Running SQL test: $test_name${NC}"
    
    if PGPASSWORD=$DB_PASS psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME \
        -f "$test_file" > "$TEST_RESULTS_DIR/${test_name}.log" 2>&1; then
        echo -e "${GREEN}✓ PASSED: $test_name${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAILED: $test_name${NC}"
        ((TESTS_FAILED++))
    fi
    ((TESTS_TOTAL++))
}

# 执行Python测试
run_python_test() {
    local test_file=$1
    local test_name=$(basename "$test_file" .py)
    
    echo -e "${YELLOW}Running Python test: $test_name${NC}"
    
    if python3 "$test_file" > "$TEST_RESULTS_DIR/${test_name}.log" 2>&1; then
        echo -e "${GREEN}✓ PASSED: $test_name${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAILED: $test_name${NC}"
        ((TESTS_FAILED++))
    fi
    ((TESTS_TOTAL++))
}

# 主测试流程
echo "=============================================="
echo "POC Test Suite Execution"
echo "=============================================="
echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo "Results: $TEST_RESULTS_DIR"
echo "=============================================="

# 1. 数据模型测试
echo ""
echo "## Phase 1: Data Model Tests"
for test in 01_data_model/*.sql; do
    run_sql_test "$test"
done

# 2. 版本控制测试
echo ""
echo "## Phase 2: Version Control Tests"
for test in 02_version_control/*.sql; do
    run_sql_test "$test"
done
for test in 02_version_control/*.py; do
    run_python_test "$test"
done

# 3. 并发控制测试
echo ""
echo "## Phase 3: Concurrency Tests"
for test in 03_concurrency/*.sql; do
    run_sql_test "$test"
done
for test in 03_concurrency/*.py; do
    run_python_test "$test"
done

# 4. 性能测试
echo ""
echo "## Phase 4: Performance Tests"
for test in 04_performance/*.py; do
    run_python_test "$test"
done

# 5. 多租户测试
echo ""
echo "## Phase 5: Multi-tenancy Tests"
for test in 05_multitenancy/*.sql; do
    run_sql_test "$test"
done
for test in 05_multitenancy/*.py; do
    run_python_test "$test"
done

# 生成测试报告
echo ""
echo "=============================================="
echo "Test Summary"
echo "=============================================="
echo -e "Total Tests:  $TESTS_TOTAL"
echo -e "${GREEN}Passed:       $TESTS_PASSED${NC}"
echo -e "${RED}Failed:       $TESTS_FAILED${NC}"
echo "=============================================="

# 生成JSON报告
cat > "$TEST_RESULTS_DIR/summary.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "database": "$DB_HOST:$DB_PORT/$DB_NAME",
    "total_tests": $TESTS_TOTAL,
    "passed": $TESTS_PASSED,
    "failed": $TESTS_FAILED,
    "success_rate": $(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc)%
}
EOF

# 返回退出码
if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
```

### 7.4 验收标准

#### 7.4.1 功能验收标准

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       功能验收标准矩阵                                        │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        功能验收标准                                  │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   测试项                    验收标准                    权重       │
    │   ─────────────────────────────────────────────────────────────    │
    │                                                                     │
    │   数据模型                                                          │
    │   ├── 表结构完整性         所有表创建成功，约束生效        必须     │
    │   ├── PostGIS几何支持      支持3D几何类型和运算            必须     │
    │   ├── 索引有效性           查询使用索引，性能达标          必须     │
    │   └── 外键约束             级联删除/更新正确               必须     │
    │                                                                     │
    │   版本控制                                                          │
    │   ├── Event Sourcing       事件正确存储和回放              必须     │
    │   ├── 快照功能             快照创建和加载正常              必须     │
    │   ├── 时间旅行查询         可查询任意历史版本              必须     │
    │   └── 历史压缩             压缩/解压功能正常               可选     │
    │                                                                     │
    │   并发控制                                                          │
    │   ├── 乐观锁               冲突检测和版本检查正确          必须     │
    │   ├── MVCC行为             隔离级别正确，无脏读            必须     │
    │   ├── 冲突检测             自动检测操作冲突                必须     │
    │   └── 数据一致性           一致性检查通过                  必须     │
    │                                                                     │
    │   多租户隔离                                                        │
    │   ├── Schema隔离           Schema级隔离正确                可选     │
    │   ├── RLS策略              行级安全生效                    必须     │
    │   └── 跨租户防护           无法访问其他租户数据            必须     │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

#### 7.4.2 性能验收标准

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       性能验收标准                                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        性能验收指标                                  │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   测试场景                  指标              目标值        优先级   │
    │   ─────────────────────────────────────────────────────────────    │
    │                                                                     │
    │   几何查询                                                          │
    │   ├── 空间范围查询          p95延迟          < 100ms        P0      │
    │   ├── 空间范围查询          p99延迟          < 200ms        P0      │
    │   ├── 精确几何查询          p95延迟          < 50ms         P0      │
    │   └── 空间关系查询          1000构件         < 500ms        P1      │
    │                                                                     │
    │   版本历史                                                          │
    │   ├── 单元素版本历史        p95延迟          < 50ms         P0      │
    │   ├── 时间旅行查询          10万构件         < 500ms        P0      │
    │   └── 事件流重建            1000事件         < 100ms        P1      │
    │                                                                     │
    │   并发写入                                                          │
    │   ├── 单元素更新            冲突率           < 10%          P0      │
    │   ├── 事件写入              吞吐量           > 1000 TPS     P0      │
    │   └── 批量插入              吞吐量           > 500 ops/s    P1      │
    │                                                                     │
    │   大规模数据                                                        │
    │   ├── 计数查询              100万构件        < 100ms        P0      │
    │   ├── 分页查询              100万构件        < 50ms         P0      │
    │   └── 存储效率              每构件           < 200KB        P1      │
    │                                                                     │
    │   多租户性能                                                        │
    │   └── RLS开销               查询延迟增加     < 10%          P1      │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

#### 7.4.3 验收检查清单

```markdown
# POC验收检查清单

## 数据模型 (Data Model)
- [ ] 所有核心表创建成功
- [ ] PostGIS扩展正确启用
- [ ] 空间索引(GIST)创建成功
- [ ] 外键约束正确配置
- [ ] JSONB索引(GIN)创建成功

## 版本控制 (Version Control)
- [ ] Event Sourcing事件正确存储
- [ ] 事件序列号单调递增
- [ ] 快照创建功能正常
- [ ] 快照加载功能正常
- [ ] 时间旅行查询返回正确结果
- [ ] 历史数据压缩/解压正常

## 并发控制 (Concurrency)
- [ ] 乐观锁冲突检测正确
- [ ] MVCC隔离级别生效
- [ ] 无脏读/幻读问题
- [ ] 冲突检测机制工作正常
- [ ] 数据一致性检查通过

## 性能 (Performance)
- [ ] 几何查询p95 < 100ms
- [ ] 版本历史查询p95 < 50ms
- [ ] 时间旅行查询 < 500ms (10万构件)
- [ ] 事件写入吞吐量 > 1000 TPS
- [ ] 100万构件计数查询 < 100ms
- [ ] 并发更新冲突率 < 10%

## 多租户 (Multi-tenancy)
- [ ] RLS策略正确生效
- [ ] 租户数据完全隔离
- [ ] 跨租户访问被拒绝
- [ ] RLS性能开销 < 10%

## 总体评估
- [ ] 所有P0需求满足
- [ ] 关键风险已识别
- [ ] 技术方案可行
```

---

## 8. 风险评估与建议

### 8.1 技术风险分析

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       技术风险分析                                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        风险矩阵                                      │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   风险项                    概率    影响    风险等级    缓解措施     │
    │   ─────────────────────────────────────────────────────────────    │
    │                                                                     │
    │   1. PostGIS 3D性能         中      高      高         预计算边界框 │
    │      - 复杂3D几何查询可能较慢                                       │
    │      - 建议: 使用简化几何进行初步过滤                               │
    │                                                                     │
    │   2. Event Sourcing存储膨胀 高      中      高         快照+压缩     │
    │      - 大量事件导致存储增长                                         │
    │      - 建议: 定期快照，压缩历史                                     │
    │                                                                     │
    │   3. 分布式事务一致性       中      高      高         Saga模式      │
    │      - 跨服务操作可能不一致                                         │
    │      - 建议: 使用Saga模式处理长事务                                 │
    │                                                                     │
    │   4. 并发冲突率过高         中      中      中         操作合并      │
    │      - 多人同时编辑同一元素                                         │
    │      - 建议: 实现操作合并/自动冲突解决                              │
    │                                                                     │
    │   5. RLS性能开销            低      低      低         查询优化      │
    │      - 行级安全可能增加查询开销                                     │
    │      - 建议: 优化RLS策略，使用索引                                  │
    │                                                                     │
    │   6. 大数据量查询性能       中      中      中         分区+缓存     │
    │      - 百万级构件查询可能变慢                                       │
    │      - 建议: 表分区，Redis缓存热点数据                              │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

### 8.2 技术建议

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       技术建议汇总                                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        架构建议                                      │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   1. 数据库选型建议                                                 │
    │      ✓ 推荐: YugabyteDB (PostgreSQL兼容 + 分布式)                   │
    │      ✓ 备选: CockroachDB (类似特性)                                 │
    │      ✓ 内置PostGIS支持，无需额外配置                                │
    │                                                                     │
    │   2. 存储优化建议                                                   │
    │      ✓ 使用混合存储: PostGIS存简化几何，复杂BIM存对象存储           │
    │      ✓ 实施表分区: 按时间分区历史数据                               │
    │      ✓ 定期快照: 每1000个事件创建快照                               │
    │      ✓ 历史压缩: 30天前的版本自动压缩                               │
    │                                                                     │
    │   3. 性能优化建议                                                   │
    │      ✓ 空间索引: 所有几何字段必须创建GIST索引                       │
    │      ✓ 查询缓存: Redis缓存热点项目数据                              │
    │      ✓ 连接池: 使用PgBouncer管理数据库连接                          │
    │      ✓ 读写分离: 查询走只读副本                                     │
    │                                                                     │
    │   4. 并发控制建议                                                   │
    │      ✓ 乐观锁: 元素更新使用版本号检查                               │
    │      ✓ 操作合并: 实现CRDT风格的操作合并                             │
    │      ✓ 分布式锁: Redis实现全局锁                                    │
    │      ✓ 冲突解决: 提供可视化冲突解决界面                             │
    │                                                                     │
    │   5. 多租户建议                                                     │
    │      ✓ 行级安全: 使用PostgreSQL RLS实现隔离                         │
    │      ✓ 租户上下文: 应用层设置租户ID                                 │
    │      ✓ 审计日志: 记录所有跨租户访问尝试                             │
    │      ✓ 资源配额: 每个租户设置存储/计算配额                          │
    │                                                                     │
    │   6. 监控建议                                                       │
    │      ✓ 数据库监控: pg_stat_statements, 慢查询日志                   │
    │      ✓ 应用监控: 请求延迟, 错误率, 吞吐量                           │
    │      ✓ 业务监控: 活跃项目数, 并发用户数                             │
    │      ✓ 告警: 性能下降, 存储不足, 错误激增                           │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

### 8.3 后续工作建议

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       后续工作建议                                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────┐
    │                        后续行动计划                                  │
    ├─────────────────────────────────────────────────────────────────────┤
    │                                                                     │
    │   阶段1: POC验证 (2周)                                              │
    │   ├── 搭建测试环境                                                  │
    │   ├── 执行所有测试脚本                                              │
    │   ├── 收集性能数据                                                  │
    │   └── 输出POC验证报告                                               │
    │                                                                     │
    │   阶段2: 原型开发 (4周)                                             │
    │   ├── 实现核心数据模型                                              │
    │   ├── 实现Event Sourcing框架                                        │
    │   ├── 实现版本控制API                                               │
    │   └── 实现基本几何操作                                              │
    │                                                                     │
    │   阶段3: 性能优化 (2周)                                             │
    │   ├── 数据库性能调优                                                │
    │   ├── 缓存策略实现                                                  │
    │   ├── 查询优化                                                      │
    │   └── 压力测试                                                      │
    │                                                                     │
    │   阶段4: 生产准备 (2周)                                             │
    │   ├── 部署脚本开发                                                  │
    │   ├── 监控告警配置                                                  │
    │   ├── 备份恢复方案                                                  │
    │   └── 运维文档编写                                                  │
    │                                                                     │
    │   总计: 10周                                                        │
    │                                                                     │
    └─────────────────────────────────────────────────────────────────────┘
```

---

## 附录

### A. 参考资料

1. YugabyteDB官方文档: https://docs.yugabyte.com/
2. PostGIS官方文档: https://postgis.net/documentation/
3. Event Sourcing模式: https://martinfowler.com/eaaDev/EventSourcing.html
4. PostgreSQL RLS: https://www.postgresql.org/docs/current/ddl-rowsecurity.html

### B. 术语表

| 术语 | 说明 |
|------|------|
| Event Sourcing | 事件溯源，以事件序列记录状态变更的模式 |
| MVCC | 多版本并发控制，数据库并发控制机制 |
| RLS | 行级安全，PostgreSQL的行级访问控制 |
| GIST | 通用搜索树，PostgreSQL的空间索引类型 |
| CRDT | 无冲突复制数据类型，分布式数据结构 |

### C. 文档变更记录

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| v1.0 | 2024 | 初始版本 | 数据库架构师 |

---

**文档结束**

