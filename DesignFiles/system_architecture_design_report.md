# 半自动化建筑设计平台 - 系统架构设计报告

## 文档信息

| 项目 | 内容 |
|------|------|
| 文档名称 | 概要设计阶段-系统架构设计报告 |
| 版本 | v1.0 |
| 阶段 | 概要设计阶段 |
| 目标读者 | 技术评审委员会、开发团队、运维团队 |

---

## 目录

1. [整体架构设计](#1-整体架构设计)
2. [核心服务设计](#2-核心服务设计)
3. [API设计](#3-api设计)
4. [事件驱动架构](#4-事件驱动架构)
5. [实时协作架构](#5-实时协作架构)
6. [部署架构](#6-部署架构)
7. [非功能性设计](#7-非功能性设计)

---

## 1. 整体架构设计

### 1.1 系统分层架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              展示层 (Presentation Layer)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │  Web前端    │  │  桌面端     │  │  移动端     │  │  第三方集成客户端   │ │
│  │  (React)    │  │  (Electron) │  │  (ReactNative)│  │  (API/SDK)         │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           网关层 (Gateway Layer)                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │  API Gateway (Kong/Nginx)                                               ││
│  │  - 认证鉴权 │ 限流熔断 │ 路由转发 │ 协议转换 │ 日志监控                  ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │  WebSocket Gateway                                                      ││
│  │  - 连接管理 │ 消息路由 │ 心跳检测 │ 负载均衡                              ││
│  └─────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         BFF层 (Backend For Frontend)                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │  GraphQL Gateway (Node.js/Apollo)                                       ││
│  │  - 数据聚合 │ 字段裁剪 │ 缓存优化 │ 查询优化                              ││
│  └─────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           应用层 (Application Layer)                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐│
│  │ 协作服务 │ │ 几何服务 │ │ 属性服务 │ │ 脚本服务 │ │ 版本服务 │ │用户服务││
│  │ (Go)     │ │ (Go)     │ │ (Java)   │ │ (Go)     │ │ (Java)   │ │(Java)  ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────┘│
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │ 项目服务 │ │ 文件服务 │ │ 通知服务 │ │ 搜索服务 │ │ 分析服务 │            │
│  │ (Java)   │ │ (Go)     │ │ (Go)     │ │ (Java)   │ │ (Go)     │            │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘            │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            领域层 (Domain Layer)                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │  领域模型 │ 领域服务 │ 领域事件 │ 值对象 │ 聚合根                          ││
│  └─────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        基础设施层 (Infrastructure Layer)                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────────┐ │
│  │ PostgreSQL│ │ MongoDB  │ │  Redis   │ │  MinIO   │ │  Elasticsearch     │ │
│  │ (主数据)  │ │ (文档)   │ │ (缓存)   │ │ (对象存储)│ │ (全文搜索)         │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────────┐ │
│  │  Kafka   │ │  NATS    │ │ClickHouse│ │  InfluxDB│ │  Prometheus/Grafana│ │
│  │ (事件流)  │ │ (消息)   │ │ (分析)   │ │ (时序数据)│ │ (监控)             │ │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 微服务拆分方案

| 服务名称 | 技术栈 | 职责 | 关键指标 |
|---------|--------|------|---------|
| **协作服务** | Go | 实时协作、CRDT同步、操作广播 | 延迟 < 50ms |
| **几何服务** | Go | 几何计算、BIM解析、空间查询 | 吞吐量 > 1000 ops/s |
| **属性服务** | Java/Spring Boot | 属性管理、参数化设计、规则引擎 | 并发 > 500 |
| **脚本服务** | Go | 脚本执行、沙箱环境、API网关 | 隔离安全 |
| **版本服务** | Java/Spring Boot | 版本控制、变更追踪、分支管理 | 存储优化 |
| **用户服务** | Java/Spring Boot | 用户管理、权限控制、组织架构 | 认证 < 100ms |
| **项目服务** | Java/Spring Boot | 项目管理、资源分配、工作流 | 事务一致 |
| **文件服务** | Go | 文件上传、格式转换、预览生成 | 支持 2GB+ |
| **通知服务** | Go | 消息推送、邮件、Webhook | 实时投递 |
| **搜索服务** | Java/ES | 全文搜索、智能推荐、语义分析 | 查询 < 200ms |
| **分析服务** | Go | 数据分析、报表生成、性能监控 | 异步处理 |

### 1.3 服务间通信方式

```
┌─────────────────────────────────────────────────────────────────┐
│                      服务间通信架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  同步通信 (Synchronous)                                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  gRPC (内部服务)                                         │   │
│  │  - 服务发现: Consul/Etcd                                 │   │
│  │  - 负载均衡: Client-side LB                              │   │
│  │  - 超时控制: 5s default, 30s max                         │   │
│  │  - 重试策略: 3次指数退避                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  异步通信 (Asynchronous)                                        │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Apache Kafka (事件流)                                   │   │
│  │  - 领域事件发布/订阅                                     │   │
│  │  - 事件溯源存储                                          │   │
│  │  - 分区: 按聚合根ID哈希                                  │   │
│  │  - 副本: 3                                               │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  NATS (实时消息)                                         │   │
│  │  - 协作操作广播                                          │   │
│  │  - 通知推送                                              │   │
│  │  - 模式: JetStream持久化                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  实时通信 (Real-time)                                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  WebSocket + CRDT                                        │   │
│  │  - 双向实时通信                                          │   │
│  │  - 操作转换与同步                                        │   │
│  │  - 心跳: 30s                                             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.4 数据流设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            核心数据流                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. 用户操作数据流                                                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│  │ 客户端  │───▶│ WS网关  │───▶│协作服务 │───▶│ CRDT引擎│───▶│ 广播    │   │
│  │ 操作    │    │         │    │         │    │         │    │ 给其他  │   │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    │ 客户端  │   │
│                                                              └─────────┘   │
│                                                                             │
│  2. 几何数据处理流                                                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│  │ BIM文件 │───▶│文件服务 │───▶│几何服务 │───▶│解析引擎 │───▶│存储到   │   │
│  │ 上传    │    │ 接收    │    │ 处理    │    │         │    │ MongoDB │   │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                                                                             │
│  3. 属性变更事件流                                                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│  │属性变更 │───▶│属性服务 │───▶│ Kafka   │───▶│订阅服务 │───▶│通知用户 │   │
│  │ 请求    │    │ 验证    │    │ 事件    │    │ 处理    │    │         │   │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                                                                             │
│  4. 版本控制数据流                                                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│  │提交快照 │───▶│版本服务 │───▶│差异计算 │───▶│存储版本 │───▶│更新引用 │   │
│  │         │    │ 接收    │    │         │    │ 链      │    │         │   │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                                                                             │
│  5. 脚本执行数据流                                                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│  │脚本提交 │───▶│脚本服务 │───▶│沙箱执行 │───▶│结果处理 │───▶│状态更新 │   │
│  │         │    │ 验证    │    │ 环境    │    │         │    │         │   │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. 核心服务设计

### 2.1 协作服务（Collaboration Service）

```
┌─────────────────────────────────────────────────────────────────┐
│                     协作服务架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  API Layer (REST/WebSocket)                              │   │
│  │  - POST /api/v1/sessions          创建协作会话           │   │
│  │  - GET  /api/v1/sessions/{id}     获取会话信息           │   │
│  │  - WS   /ws/v1/sessions/{id}      WebSocket连接          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Session Manager                                         │   │
│  │  - 会话生命周期管理                                      │   │
│  │  - 参与者管理 (加入/离开/权限)                           │   │
│  │  - 心跳检测与超时处理                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  CRDT Engine                                             │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │ 文档CRDT    │  │ 几何CRDT    │  │ 属性CRDT    │     │   │
│  │  │ (Yjs格式)   │  │ (自定义)    │  │ (JSON CRDT) │     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  │  - 操作本地执行                                          │   │
│  │  - 变更广播                                              │   │
│  │  - 状态合并                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Message Router                                          │   │
│  │  - NATS发布/订阅                                         │   │
│  │  - 消息路由与过滤                                        │   │
│  │  - 消息顺序保证                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Persistence Layer                                       │   │
│  │  - Redis: 会话状态缓存                                   │   │
│  │  - PostgreSQL: 会话持久化                                │   │
│  │  - Kafka: 操作日志                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**核心接口定义:**

```protobuf
// collaboration.proto
syntax = "proto3";
package collaboration;

service CollaborationService {
  // 会话管理
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc GetSession(GetSessionRequest) returns (Session);
  rpc CloseSession(CloseSessionRequest) returns (Empty);

  // 参与者管理
  rpc JoinSession(JoinSessionRequest) returns (Participant);
  rpc LeaveSession(LeaveSessionRequest) returns (Empty);
  rpc UpdateCursor(UpdateCursorRequest) returns (Empty);

  // 操作同步
  rpc SyncOperations(stream Operation) returns (stream Operation);
  rpc GetDocumentState(GetDocumentStateRequest) returns (DocumentState);
}

message CreateSessionRequest {
  string project_id = 1;
  string document_id = 2;
  string creator_id = 3;
  SessionType type = 4;
}

message Operation {
  string session_id = 1;
  string client_id = 2;
  int64 timestamp = 3;
  bytes crdt_update = 4;
  OperationType type = 5;
}

enum OperationType {
  INSERT = 0;
  DELETE = 1;
  UPDATE = 2;
  CURSOR = 3;
  SELECTION = 4;
}
```

### 2.2 几何服务（Geometry Service）

```
┌─────────────────────────────────────────────────────────────────┐
│                     几何服务架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  API Layer                                               │   │
│  │  - GET  /api/v1/geometry/{id}      获取几何数据          │   │
│  │  - POST /api/v1/geometry/query     空间查询              │   │
│  │  - POST /api/v1/geometry/boolean   布尔运算              │   │
│  │  - POST /api/v1/geometry/transform 几何变换              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  BIM Parser Engine                                       │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ IFC解析器 │ │ Revit解析│ │ DWG解析  │ │ 其他格式 │   │   │
│  │  │          │ │ 器       │ │ 器       │ │          │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  │  - 多格式支持                                            │   │
│  │  - 增量解析                                              │   │
│  │  - 内存优化 (流式处理)                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Geometry Kernel                                         │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ OpenCASCADE│ │ CGAL    │ │ 自定义算法│ │ 空间索引 │   │   │
│  │  │          │ │          │ │          │ │ (R-tree) │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  │  - NURBS曲面计算                                         │   │
│  │  - 布尔运算                                              │   │
│  │  - 碰撞检测                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Spatial Query Engine                                    │   │
│  │  - 空间索引 (R-tree/八叉树)                              │   │
│  │  - 范围查询                                              │   │
│  │  - 最近邻搜索                                            │   │
│  │  - 射线检测                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Cache & Storage                                         │   │
│  │  - Redis: 热点几何缓存                                   │   │
│  │  - MongoDB: 几何文档存储                                 │   │
│  │  - MinIO: 原始文件存储                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**核心接口定义:**

```protobuf
// geometry.proto
syntax = "proto3";
package geometry;

service GeometryService {
  // 几何数据管理
  rpc GetGeometry(GetGeometryRequest) returns (Geometry);
  rpc CreateGeometry(CreateGeometryRequest) returns (Geometry);
  rpc UpdateGeometry(UpdateGeometryRequest) returns (Geometry);
  rpc DeleteGeometry(DeleteGeometryRequest) returns (Empty);

  // 空间查询
  rpc SpatialQuery(SpatialQueryRequest) returns (GeometryCollection);
  rpc Raycast(RaycastRequest) returns (RaycastResult);
  rpc NearestNeighbor(NearestNeighborRequest) returns (Geometry);

  // 几何运算
  rpc BooleanOperation(BooleanOperationRequest) returns (Geometry);
  rpc Transform(TransformRequest) returns (Geometry);
  rpc ComputeVolume(ComputeVolumeRequest) returns (VolumeResult);
  rpc ComputeArea(ComputeAreaRequest) returns (AreaResult);

  // BIM解析
  rpc ParseBIMFile(ParseBIMFileRequest) returns (stream ParseProgress);
  rpc GetParseStatus(GetParseStatusRequest) returns (ParseStatus);
}

message SpatialQueryRequest {
  string project_id = 1;
  BoundingBox bbox = 2;
  repeated string filters = 3;
  int32 limit = 4;
}

message BooleanOperationRequest {
  string geometry_id_1 = 1;
  string geometry_id_2 = 2;
  BooleanOperationType operation = 3; // UNION, INTERSECTION, DIFFERENCE
}
```

### 2.3 属性服务（Property Service）

```
┌─────────────────────────────────────────────────────────────────┐
│                     属性服务架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  API Layer                                               │   │
│  │  - CRUD 属性定义                                         │   │
│  │  - 属性值管理                                            │   │
│  │  - 参数化规则执行                                        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Property Schema Manager                                 │   │
│  │  - 属性模板定义                                          │   │
│  │  - 属性继承与覆盖                                        │   │
│  │  - 属性验证规则                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Rule Engine                                             │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ Drools   │ │ 表达式引擎│ │ 约束求解 │ │ 单位转换 │   │   │
│  │  │          │ │ (Aviator)│ │ 器       │ │ 器       │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  │  - 参数化设计规则                                        │   │
│  │  - 属性联动计算                                          │   │
│  │  - 约束检查                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Event Publisher                                         │   │
│  │  - 属性变更事件                                          │   │
│  │  - 规则触发事件                                          │   │
│  │  - 异常告警事件                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Storage Layer                                           │   │
│  │  - PostgreSQL: 属性结构化数据                            │   │
│  │  - MongoDB: 动态属性文档                                 │   │
│  │  - Redis: 属性缓存                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.4 脚本服务（Script Service）

```
┌─────────────────────────────────────────────────────────────────┐
│                     脚本服务架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  API Layer                                               │   │
│  │  - POST /api/v1/scripts          创建脚本                │   │
│  │  - POST /api/v1/scripts/{id}/run 执行脚本                │   │
│  │  - GET  /api/v1/scripts/{id}/status 获取状态             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Script Manager                                          │   │
│  │  - 脚本版本管理                                          │   │
│  │  - 脚本依赖解析                                          │   │
│  │  - 脚本缓存                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Sandbox Environment                                     │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ gVisor   │ │ Firecracker│ │ 资源限制 │ │ 网络隔离 │   │   │
│  │  │          │ │          │ │ (cgroup) │ │          │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  │  - 安全沙箱                                              │   │
│  │  - CPU/内存限制                                          │   │
│  │  - 执行超时控制                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Script Runtime                                          │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐                │   │
│  │  │ Python   │ │ JavaScript│ │ C#       │                │   │
│  │  │ (IronPython│ │ (V8)    │ │ (Roslyn) │                │   │
│  │  └──────────┘ └──────────┘ └──────────┘                │   │
│  │  - 多语言支持                                            │   │
│  │  - API绑定                                               │   │
│  │  - 调试支持                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Execution Queue                                         │   │
│  │  - 任务队列 (Redis/RabbitMQ)                             │   │
│  │  - 优先级调度                                            │   │
│  │  - 结果回调                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.5 版本服务（Version Service）

```
┌─────────────────────────────────────────────────────────────────┐
│                     版本服务架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  API Layer                                               │   │
│  │  - POST /api/v1/versions         创建版本                │   │
│  │  - GET  /api/v1/versions/{id}    获取版本                │   │
│  │  - POST /api/v1/versions/{id}/diff 对比版本              │   │
│  │  - POST /api/v1/branches         创建分支                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Version Control Core                                    │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ 快照存储 │ │ 差异计算 │ │ 合并引擎 │ │ 冲突解决 │   │   │
│  │  │          │ │ (Myers)  │ │          │ │          │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  │  - 版本链管理                                            │   │
│  │  - 分支与合并                                            │   │
│  │  - 标签管理                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Change Tracker                                          │   │
│  │  - 操作日志记录                                          │   │
│  │  - 变更聚合                                              │   │
│  │  - 变更回放                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Storage Layer                                           │   │
│  │  - PostgreSQL: 版本元数据                                │   │
│  │  - MinIO: 版本快照存储                                   │   │
│  │  - Kafka: 变更事件流                                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.6 用户服务（User Service）

```
┌─────────────────────────────────────────────────────────────────┐
│                     用户服务架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  API Layer                                               │   │
│  │  - 用户CRUD                                              │   │
│  │  - 认证授权                                              │   │
│  │  - 组织架构                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Authentication & Authorization                          │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ OAuth2   │ │ JWT      │ │ RBAC     │ │ ABAC     │   │   │
│  │  │          │ │          │ │          │ │          │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  │  - SSO集成                                               │   │
│  │  - 多因素认证                                            │   │
│  │  - 细粒度权限控制                                        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Organization Manager                                    │   │
│  │  - 租户管理                                              │   │
│  │  - 部门层级                                              │   │
│  │  - 团队管理                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Storage Layer                                           │   │
│  │  - PostgreSQL: 用户数据                                  │   │
│  │  - Redis: 会话/令牌缓存                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```


---

## 3. API设计

### 3.1 RESTful API设计规范

```yaml
# API设计规范

# 基础规范
BaseURL: https://api.archdesign.com
Version: v1
Format: JSON
Encoding: UTF-8

# HTTP方法规范
Methods:
  GET:    获取资源（幂等）
  POST:   创建资源
  PUT:    全量更新资源（幂等）
  PATCH:  部分更新资源
  DELETE: 删除资源（幂等）

# URL设计规范
URL_Pattern: /api/{version}/{resource}/{id}/{sub-resource}

# 响应格式
Response_Format:
  success:
    code: 200/201/204
    data: {}
    meta:
      page: 1
      page_size: 20
      total: 100
  error:
    code: 400/401/403/404/500
    error:
      type: "ValidationError"
      message: "详细错误信息"
      details: []

# 分页规范
Pagination:
  default_page_size: 20
  max_page_size: 100
  parameters:
    page: 当前页码
    page_size: 每页数量
    sort: 排序字段 (-created_at 表示倒序)

# 过滤规范
Filtering:
  format: ?field__operator=value
  operators:
    eq: 等于
    ne: 不等于
    gt: 大于
    gte: 大于等于
    lt: 小于
    lte: 小于等于
    in: 在列表中
    contains: 包含
    startswith: 以...开头
```

**核心RESTful API示例:**

```yaml
# 项目API
/projects:
  GET:
    summary: 获取项目列表
    parameters:
      - name: name__contains
        in: query
        description: 项目名称模糊查询
      - name: status
        in: query
        description: 项目状态过滤
    responses:
      200:
        description: 项目列表

  POST:
    summary: 创建项目
    requestBody:
      content:
        application/json:
          schema:
            type: object
            required: [name, owner_id]
            properties:
              name: { type: string }
              description: { type: string }
              owner_id: { type: string }
              template_id: { type: string }

/projects/{id}:
  GET:
    summary: 获取项目详情

  PUT:
    summary: 更新项目

  DELETE:
    summary: 删除项目

/projects/{id}/members:
  GET:
    summary: 获取项目成员

  POST:
    summary: 添加项目成员

# 文档API
/documents:
  GET: 获取文档列表
  POST: 创建文档

/documents/{id}:
  GET: 获取文档详情
  PUT: 更新文档
  DELETE: 删除文档

/documents/{id}/export:
  POST:
    summary: 导出文档
    requestBody:
      content:
        application/json:
          schema:
            type: object
            properties:
              format: { enum: [IFC, DWG, PDF, OBJ] }
              options: { type: object }
```

### 3.2 GraphQL API设计（BFF层）

```graphql
# schema.graphql

# 标量类型
scalar DateTime
scalar JSON
scalar Geometry
scalar UUID

# 枚举类型
enum ProjectStatus {
  DRAFT
  ACTIVE
  ARCHIVED
  DELETED
}

enum Permission {
  READ
  WRITE
  ADMIN
  OWNER
}

# 接口类型
interface Node {
  id: ID!
  createdAt: DateTime!
  updatedAt: DateTime!
}

# 对象类型
type User implements Node {
  id: ID!
  email: String!
  name: String!
  avatar: String
  createdAt: DateTime!
  updatedAt: DateTime!
  projects: ProjectConnection!
  permissions: [Permission!]!
}

type Project implements Node {
  id: ID!
  name: String!
  description: String
  status: ProjectStatus!
  owner: User!
  members: [ProjectMember!]!
  documents: DocumentConnection!
  createdAt: DateTime!
  updatedAt: DateTime!

  # 聚合字段
  documentCount: Int!
  storageUsed: Float!
}

type Document implements Node {
  id: ID!
  name: String!
  project: Project!
  type: String!
  version: DocumentVersion!
  versions: [DocumentVersion!]!
  geometry: Geometry
  properties: JSON
  collaborators: [User!]!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type DocumentVersion {
  id: ID!
  document: Document!
  version: String!
  author: User!
  message: String
  createdAt: DateTime!
  changes: [Change!]!
}

type Change {
  id: ID!
  type: String!
  target: String!
  before: JSON
  after: JSON
}

type ProjectMember {
  user: User!
  permission: Permission!
  joinedAt: DateTime!
}

# 连接类型 (分页)
type ProjectConnection {
  edges: [ProjectEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type ProjectEdge {
  node: Project!
  cursor: String!
}

type DocumentConnection {
  edges: [DocumentEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type DocumentEdge {
  node: Document!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# 查询类型
type Query {
  # 节点查询
  node(id: ID!): Node
  nodes(ids: [ID!]!): [Node]!

  # 用户查询
  me: User!
  user(id: ID!): User

  # 项目查询
  project(id: ID!): Project
  projects(
    first: Int = 20
    after: String
    filter: ProjectFilter
    sort: ProjectSort
  ): ProjectConnection!

  # 文档查询
  document(id: ID!): Document
  documents(
    projectId: ID!
    first: Int = 20
    after: String
    filter: DocumentFilter
  ): DocumentConnection!

  # 搜索
  search(query: String!, types: [String!], limit: Int = 10): SearchResult!
}

# 输入类型
input ProjectFilter {
  status: ProjectStatus
  nameContains: String
  ownerId: ID
}

input ProjectSort {
  field: ProjectSortField!
  direction: SortDirection!
}

enum ProjectSortField {
  NAME
  CREATED_AT
  UPDATED_AT
}

enum SortDirection {
  ASC
  DESC
}

input DocumentFilter {
  type: String
  nameContains: String
}

# 变更类型
type Mutation {
  # 项目操作
  createProject(input: CreateProjectInput!): Project!
  updateProject(id: ID!, input: UpdateProjectInput!): Project!
  deleteProject(id: ID!): Boolean!
  addProjectMember(projectId: ID!, userId: ID!, permission: Permission!): ProjectMember!
  removeProjectMember(projectId: ID!, userId: ID!): Boolean!

  # 文档操作
  createDocument(input: CreateDocumentInput!): Document!
  updateDocument(id: ID!, input: UpdateDocumentInput!): Document!
  deleteDocument(id: ID!): Boolean!
  commitDocumentVersion(id: ID!, message: String): DocumentVersion!

  # 协作操作
  joinCollaborationSession(documentId: ID!): CollaborationSession!
  leaveCollaborationSession(sessionId: ID!): Boolean!
}

input CreateProjectInput {
  name: String!
  description: String
  templateId: ID
}

input UpdateProjectInput {
  name: String
  description: String
  status: ProjectStatus
}

input CreateDocumentInput {
  projectId: ID!
  name: String!
  type: String!
  initialData: JSON
}

input UpdateDocumentInput {
  name: String
  geometry: Geometry
  properties: JSON
}

# 订阅类型
type Subscription {
  # 协作会话订阅
  collaborationOperations(sessionId: ID!): Operation!

  # 文档变更订阅
  documentChanged(documentId: ID!): DocumentChange!

  # 通知订阅
  notificationReceived: Notification!

  # 用户状态订阅
  userPresenceChanged(projectId: ID!): UserPresence!
}

type Operation {
  id: ID!
  type: String!
  clientId: String!
  timestamp: DateTime!
  data: JSON!
}

type DocumentChange {
  documentId: ID!
  changeType: String!
  data: JSON!
}

type Notification {
  id: ID!
  type: String!
  title: String!
  message: String!
  data: JSON
  createdAt: DateTime!
}

type UserPresence {
  userId: ID!
  documentId: ID
  status: PresenceStatus!
  cursor: CursorPosition
}

enum PresenceStatus {
  ONLINE
  AWAY
  OFFLINE
}

type CursorPosition {
  x: Float!
  y: Float!
  z: Float
}

# 搜索结果联合类型
union SearchResultItem = Project | Document | User

type SearchResult {
  items: [SearchResultItem!]!
  totalCount: Int!
  facets: [SearchFacet!]!
}

type SearchFacet {
  field: String!
  values: [FacetValue!]!
}

type FacetValue {
  value: String!
  count: Int!
}
```

### 3.3 gRPC内部服务接口

```protobuf
// common.proto
syntax = "proto3";
package common;

option go_package = "github.com/archdesign/api/common";

message Empty {}

message Pagination {
  int32 page = 1;
  int32 page_size = 2;
}

message PageInfo {
  int32 total = 1;
  int32 page = 2;
  int32 page_size = 3;
  int32 total_pages = 4;
}

message Error {
  string code = 1;
  string message = 2;
  map<string, string> details = 3;
}

// user_service.proto
syntax = "proto3";
package user;

import "common.proto";

option go_package = "github.com/archdesign/api/user";

service UserService {
  rpc GetUser(GetUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc UpdateUser(UpdateUserRequest) returns (User);
  rpc DeleteUser(DeleteUserRequest) returns (common.Empty);

  rpc Authenticate(AuthenticateRequest) returns (AuthResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (TokenValidation);
  rpc RefreshToken(RefreshTokenRequest) returns (AuthResponse);

  rpc GetPermissions(GetPermissionsRequest) returns (Permissions);
  rpc CheckPermission(CheckPermissionRequest) returns (PermissionCheck);
}

message User {
  string id = 1;
  string email = 2;
  string name = 3;
  string avatar = 4;
  string organization_id = 5;
  repeated string roles = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
}

message GetUserRequest {
  string id = 1;
}

message ListUsersRequest {
  common.Pagination pagination = 1;
  string organization_id = 2;
  string search = 3;
}

message ListUsersResponse {
  repeated User users = 1;
  common.PageInfo page_info = 2;
}

message CreateUserRequest {
  string email = 1;
  string name = 2;
  string password = 3;
  string organization_id = 4;
}

message UpdateUserRequest {
  string id = 1;
  string name = 2;
  string avatar = 3;
}

message DeleteUserRequest {
  string id = 1;
}

message AuthenticateRequest {
  string email = 1;
  string password = 2;
  string mfa_code = 3;
}

message AuthResponse {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_at = 3;
  User user = 4;
}

message ValidateTokenRequest {
  string token = 1;
}

message TokenValidation {
  bool valid = 1;
  string user_id = 2;
  repeated string permissions = 3;
  int64 expires_at = 4;
}

message GetPermissionsRequest {
  string user_id = 1;
  string resource_type = 2;
  string resource_id = 3;
}

message Permissions {
  repeated string permissions = 1;
}

message CheckPermissionRequest {
  string user_id = 1;
  string permission = 2;
  string resource_type = 3;
  string resource_id = 4;
}

message PermissionCheck {
  bool allowed = 1;
}
```

### 3.4 API版本管理策略

```
┌─────────────────────────────────────────────────────────────────┐
│                     API版本管理策略                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  版本策略: URL路径版本控制                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  /api/v1/projects     - 当前稳定版本                     │   │
│  │  /api/v2/projects     - 新版本 (开发中)                  │   │
│  │  /api/beta/projects   - 预览版本                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  版本生命周期:                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Beta ──▶ v1.0 ──▶ v1.x ──▶ v2.0 Beta ──▶ v2.0         │   │
│  │   │       │        │          │          │              │   │
│  │   │       │        │          │          │              │   │
│  │  3个月   12个月   18个月     3个月      长期支持        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  兼容性保证:                                                     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  ✓ 向后兼容: 新增字段、新增端点                          │   │
│  │  ✗ 破坏性变更: 删除字段、修改字段类型、修改行为          │   │
│  │  ⚠ 废弃流程: 标记废弃 ▶ 保留6个月 ▶ 通知迁移 ▶ 移除    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  版本迁移支持:                                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  - 兼容性适配层 (Adapter Pattern)                        │   │
│  │  - 自动化迁移工具                                        │   │
│  │  - 版本对比文档                                          │   │
│  │  - 弃用通知机制                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```


---

## 4. 事件驱动架构

### 4.1 领域事件定义

```protobuf
// events.proto
syntax = "proto3";
package events;

option go_package = "github.com/archdesign/api/events";

// 基础事件结构
message DomainEvent {
  string event_id = 1;
  string event_type = 2;
  string aggregate_id = 3;
  string aggregate_type = 4;
  int64 timestamp = 5;
  string version = 6;
  bytes payload = 7;
  map<string, string> metadata = 8;
}

// ============ 用户领域事件 ============

message UserCreated {
  string user_id = 1;
  string email = 2;
  string name = 3;
  string organization_id = 4;
  int64 created_at = 5;
}

message UserUpdated {
  string user_id = 1;
  string name = 2;
  string avatar = 3;
  int64 updated_at = 4;
}

message UserDeleted {
  string user_id = 1;
  int64 deleted_at = 2;
}

// ============ 项目领域事件 ============

message ProjectCreated {
  string project_id = 1;
  string name = 2;
  string owner_id = 3;
  string organization_id = 4;
  int64 created_at = 5;
}

message ProjectUpdated {
  string project_id = 1;
  string name = 2;
  string description = 3;
  string status = 4;
  int64 updated_at = 5;
}

message ProjectDeleted {
  string project_id = 1;
  int64 deleted_at = 2;
}

message ProjectMemberAdded {
  string project_id = 1;
  string user_id = 2;
  string permission = 3;
  int64 added_at = 4;
}

message ProjectMemberRemoved {
  string project_id = 1;
  string user_id = 2;
  int64 removed_at = 3;
}

// ============ 文档领域事件 ============

message DocumentCreated {
  string document_id = 1;
  string project_id = 2;
  string name = 3;
  string type = 4;
  string creator_id = 5;
  int64 created_at = 6;
}

message DocumentUpdated {
  string document_id = 1;
  string name = 2;
  int64 updated_at = 3;
}

message DocumentDeleted {
  string document_id = 1;
  int64 deleted_at = 2;
}

// ============ 版本领域事件 ============

message VersionCommitted {
  string version_id = 1;
  string document_id = 2;
  string author_id = 3;
  string message = 4;
  string parent_version_id = 5;
  int64 committed_at = 6;
}

message BranchCreated {
  string branch_id = 1;
  string document_id = 2;
  string name = 3;
  string base_version_id = 4;
  string creator_id = 5;
  int64 created_at = 6;
}

message BranchMerged {
  string source_branch_id = 1;
  string target_branch_id = 2;
  string merge_commit_id = 3;
  string merged_by = 4;
  int64 merged_at = 5;
}

// ============ 协作领域事件 ============

message CollaborationSessionStarted {
  string session_id = 1;
  string document_id = 2;
  string user_id = 3;
  int64 started_at = 4;
}

message CollaborationSessionEnded {
  string session_id = 1;
  string document_id = 2;
  int64 ended_at = 3;
}

message UserJoinedSession {
  string session_id = 1;
  string user_id = 2;
  int64 joined_at = 3;
}

message UserLeftSession {
  string session_id = 1;
  string user_id = 2;
  int64 left_at = 3;
}

// ============ 属性领域事件 ============

message PropertyChanged {
  string property_id = 1;
  string target_id = 2;
  string target_type = 3;
  string property_name = 4;
  bytes old_value = 5;
  bytes new_value = 6;
  string changed_by = 7;
  int64 changed_at = 8;
}

message RuleTriggered {
  string rule_id = 1;
  string target_id = 2;
  string trigger_type = 3;
  repeated PropertyChange changes = 4;
  int64 triggered_at = 5;
}

message PropertyChange {
  string property_name = 1;
  bytes old_value = 2;
  bytes new_value = 3;
}

// ============ 几何领域事件 ============

message GeometryCreated {
  string geometry_id = 1;
  string document_id = 2;
  string type = 3;
  bytes data = 4;
  int64 created_at = 5;
}

message GeometryUpdated {
  string geometry_id = 1;
  bytes data = 2;
  int64 updated_at = 3;
}

message GeometryDeleted {
  string geometry_id = 1;
  int64 deleted_at = 2;
}

message BIMFileParsed {
  string file_id = 1;
  string document_id = 2;
  string format = 3;
  int32 element_count = 4;
  int64 parsed_at = 5;
}

// ============ 脚本领域事件 ============

message ScriptExecuted {
  string execution_id = 1;
  string script_id = 2;
  string executor_id = 3;
  string status = 4;
  bytes result = 5;
  int64 started_at = 6;
  int64 completed_at = 7;
}

message ScriptFailed {
  string execution_id = 1;
  string script_id = 2;
  string error_code = 3;
  string error_message = 4;
  int64 failed_at = 5;
}

// ============ 通知领域事件 ============

message NotificationSent {
  string notification_id = 1;
  string user_id = 2;
  string type = 3;
  string title = 4;
  string message = 5;
  int64 sent_at = 6;
}

message NotificationRead {
  string notification_id = 1;
  string user_id = 2;
  int64 read_at = 3;
}
```

### 4.2 事件发布/订阅机制

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        事件发布/订阅架构                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Apache Kafka                                  │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │ user-events │  │project-events│  │doc-events   │  │collab-events│ │   │
│  │  │ 分区: 6     │  │ 分区: 12    │  │ 分区: 24    │  │ 分区: 48   │ │   │
│  │  │ 副本: 3     │  │ 副本: 3     │  │ 副本: 3     │  │ 副本: 3    │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │version-events│  │prop-events  │  │geo-events   │  │script-events│ │   │
│  │  │ 分区: 12    │  │ 分区: 12    │  │ 分区: 18    │  │ 分区: 6    │ │   │
│  │  │ 副本: 3     │  │ 副本: 3     │  │ 副本: 3     │  │ 副本: 3    │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        事件发布者 (Producers)                        │   │
│  │                                                                     │   │
│  │  服务层 ──▶ 领域事件 ──▶ EventPublisher ──▶ Kafka Producer          │   │
│  │                                                                     │   │
│  │  发布模式:                                                          │   │
│  │  1. 事务性发布: 数据库事务 + 事件发布 (Outbox Pattern)              │   │
│  │  2. 异步发布: 批量发送 + 重试机制                                   │   │
│  │  3. 顺序保证: 相同聚合根ID的事件发送到同一分区                      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        事件订阅者 (Consumers)                        │   │
│  │                                                                     │   │
│  │  Consumer Group: user-service                                        │   │
│  │    ├─ 订阅: user-events                                              │   │
│  │    └─ 处理: 用户相关事件                                             │   │
│  │                                                                     │   │
│  │  Consumer Group: notification-service                                │   │
│  │    ├─ 订阅: user-events, project-events, collab-events              │   │
│  │    └─ 处理: 发送通知                                                 │   │
│  │                                                                     │   │
│  │  Consumer Group: audit-service                                       │   │
│  │    ├─ 订阅: * (所有事件)                                             │   │
│  │    └─ 处理: 审计日志                                                 │   │
│  │                                                                     │   │
│  │  Consumer Group: search-service                                      │   │
│  │    ├─ 订阅: project-events, doc-events                              │   │
│  │    └─ 处理: 更新搜索索引                                            │   │
│  │                                                                     │   │
│  │  消费模式:                                                          │   │
│  │  1. 至少一次投递 + 幂等处理                                         │   │
│  │  2. 批量消费 + 手动提交偏移量                                       │   │
│  │  3. 死信队列处理失败消息                                            │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.3 Saga分布式事务

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Saga分布式事务设计                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Saga编排模式: 编排式Saga (Orchestration)                                │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Saga编排器 (Saga Orchestrator)                  │   │
│  │                                                                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │ Saga定义    │  │ 状态机      │  │ 补偿逻辑    │  │ 超时管理   │ │   │
│  │  │ (DSL)       │  │             │  │ 注册表      │  │            │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  │                                                                     │   │
│  │  技术选型: Temporal / Cadence / 自研Saga框架                        │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  典型Saga示例: 创建项目并初始化资源                                          │
│                                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐   │
│  │ 开始    │───▶│创建项目 │───▶│创建文档 │───▶│设置权限 │───▶│发送通知 │   │
│  │ Saga    │    │记录     │    │模板     │    │         │    │         │   │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘   │
│                      │              │              │              │        │
│                      ▼              ▼              ▼              ▼        │
│                 ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐      │
│                 │删除项目 │   │删除文档 │   │撤销权限 │   │(无需补偿)│      │
│                 │记录     │   │         │   │         │   │         │      │
│                 └─────────┘   └─────────┘   └─────────┘   └─────────┘      │
│                                                                             │
│  Saga状态持久化:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PostgreSQL: saga_instance 表                                       │   │
│  │  - saga_id, status, current_step, started_at, completed_at          │   │
│  │  - saga_step: step_name, status, input, output, compensation        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  超时与重试策略:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  步骤超时: 30s (默认), 可配置                                       │   │
│  │  Saga超时: 5min (默认), 可配置                                      │   │
│  │  重试策略: 3次指数退避 (1s, 2s, 4s)                                 │   │
│  │  死信处理: 人工介入 / 自动回滚                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.4 事件溯源设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         事件溯源架构                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  事件溯源适用场景:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  ✓ 文档版本历史 (Document Version History)                          │   │
│  │  ✓ 协作操作日志 (Collaboration Operation Log)                       │   │
│  │  ✓ 属性变更审计 (Property Change Audit)                             │   │
│  │  ✗ 用户基本信息 (非频繁变更)                                        │   │
│  │  ✗ 静态配置数据                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  事件存储设计:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  Kafka (事件流存储)                                                 │   │
│  │  ├── Topic: document-events                                         │   │
│  │  │   ├── 保留策略: 永久保留 (compact + delete)                       │   │
│  │  │   └── 分区: 按 document_id 哈希                                   │   │
│  │  └── Topic: operation-events                                        │   │
│  │      ├── 保留策略: 7天 (可配置)                                     │   │
│  │      └── 分区: 按 session_id 哈希                                   │   │
│  │                                                                     │   │
│  │  PostgreSQL (事件快照存储)                                          │   │
│  │  ├── 表: event_store                                                │   │
│  │  │   ├── event_id (UUID PK)                                         │   │
│  │  │   ├── aggregate_id (UUID)                                        │   │
│  │  │   ├── aggregate_type (VARCHAR)                                   │   │
│  │  │   ├── event_type (VARCHAR)                                       │   │
│  │  │   ├── event_version (INT)                                        │   │
│  │  │   ├── payload (JSONB)                                            │   │
│  │  │   ├── metadata (JSONB)                                           │   │
│  │  │   ├── occurred_at (TIMESTAMP)                                    │   │
│  │  │   └── sequence_number (BIGINT)                                   │   │
│  │  └── 索引: (aggregate_id, sequence_number) UNIQUE                   │   │
│  │                                                                     │   │
│  │  MongoDB (物化视图)                                                 │   │
│  │  ├── Collection: document_snapshots                                 │   │
│  │  │   ├── document_id                                                │   │
│  │  │   ├── version                                                    │   │
│  │  │   ├── state (完整文档状态)                                       │   │
│  │  │   └── created_at                                                 │   │
│  │  └── 快照策略: 每100个事件创建一次快照                              │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  事件回放与重建:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  重建聚合根状态:                                                    │   │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐          │   │
│  │  │读取快照 │───▶│读取事件 │───▶│应用事件 │───▶│当前状态 │          │   │
│  │  │(可选)   │    │(快照后) │    │(reduce) │    │         │          │   │
│  │  └─────────┘    └─────────┘    └─────────┘    └─────────┘          │   │
│  │                                                                     │   │
│  │  时间旅行查询:                                                      │   │
│  │  GET /api/v1/documents/{id}/at/{timestamp}                          │   │
│  │  GET /api/v1/documents/{id}/versions/{version}                      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  投影处理器 (Projections):                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐         │   │
│  │  │ Event Store │─────▶│ Projection  │─────▶│ Read Model  │         │   │
│  │  │             │      │ Processor   │      │ (MongoDB)   │         │   │
│  │  └─────────────┘      └─────────────┘      └─────────────┘         │   │
│  │         │                   │                   │                   │   │
│  │         │            ┌──────┴──────┐           │                   │   │
│  │         │            ▼             ▼           │                   │   │
│  │         │      ┌─────────┐   ┌─────────┐       │                   │   │
│  │         │      │文档投影 │   │搜索投影 │       │                   │   │
│  │         │      └─────────┘   └─────────┘       │                   │   │
│  │         │                                        │                   │   │
│  │         └────────────────────────────────────────┘                   │   │
│  │                      (事件驱动更新)                                  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```


---

## 5. 实时协作架构

### 5.1 CRDT协作引擎设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CRDT协作引擎架构                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  CRDT类型选择:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  文档内容: Yjs (YATA算法) - 基于YATA的文本CRDT                      │   │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐                         │   │
│  │  │ Y.Text  │    │ Y.Array │    │ Y.Map   │                         │   │
│  │  │ 文本    │    │ 数组    │    │ 映射    │                         │   │
│  │  └─────────┘    └─────────┘    └─────────┘                         │   │
│  │                                                                     │   │
│  │  几何数据: 自定义CRDT - 基于LWW-Register和OR-Set                    │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │ LWW-Register│  │ OR-Set      │  │ G-Counter   │                 │   │
│  │  │ (单值)      │  │ (集合)      │  │ (计数器)    │                 │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │
│  │                                                                     │   │
│  │  属性数据: JSON-CRDT - 基于RGA的JSON CRDT                           │   │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐                         │   │
│  │  │ JSON    │    │ 嵌套    │    │ 类型    │                         │   │
│  │  │ 对象    │    │ 对象    │    │ 安全    │                         │   │
│  │  └─────────┘    └─────────┘    └─────────┘                         │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  CRDT引擎核心组件:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    CRDT Document                             │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ 文档CRDT    │  │ 几何CRDT    │  │ 属性CRDT    │         │   │   │
│  │  │  │ (Y.Doc)     │  │ (Custom)    │  │ (JSON-CRDT) │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │                                                              │   │   │
│  │  │  - 本地状态管理                                              │   │   │
│  │  │  - 更新应用                                                  │   │   │
│  │  │  - 状态编码/解码                                             │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Update Manager                            │   │   │
│  │  │                                                              │   │   │
│  │  │  - 本地更新队列                                              │   │   │
│  │  │  - 远程更新合并                                              │   │   │
│  │  │  - 更新去重 (基于Vector Clock)                               │   │   │
│  │  │  - 更新压缩 (Delta Encoding)                                 │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Sync Protocol                             │   │   │
│  │  │                                                              │   │   │
│  │  │  - 状态向量交换 (State Vector)                               │   │   │
│  │  │  - 差异更新 (Diff Update)                                    │   │   │
│  │  │  - 快照同步 (Snapshot Sync)                                  │   │   │
│  │  │  - 增量同步 (Incremental Sync)                               │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Awareness Module                          │   │   │
│  │  │                                                              │   │   │
│  │  │  - 用户光标位置                                              │   │   │
│  │  │  - 用户选择范围                                              │   │   │
│  │  │  - 用户在线状态                                              │   │   │
│  │  │  - 用户权限信息                                              │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  Vector Clock实现:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  每个客户端维护一个逻辑时钟:                                        │   │
│  │  {                                                                  │   │
│  │    "client-1": 10,  // 客户端1的时钟                                │   │
│  │    "client-2": 5,   // 客户端2的时钟                                │   │
│  │    "client-3": 8   // 客户端3的时钟                                 │   │
│  │  }                                                                  │   │
│  │                                                                     │   │
│  │  用于:                                                              │   │
│  │  - 检测并发操作                                                     │   │
│  │  - 确定因果关系                                                     │   │
│  │  - 更新去重                                                         │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 WebSocket网关设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        WebSocket网关架构                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    WebSocket Gateway Cluster                         │   │
│  │                                                                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │ Gateway-1   │  │ Gateway-2   │  │ Gateway-3   │  │ Gateway-N  │ │   │
│  │  │ (Pod)       │  │ (Pod)       │  │ (Pod)       │  │ (Pod)      │ │   │
│  │  │ 10k conn    │  │ 10k conn    │  │ 10k conn    │  │ 10k conn   │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  │                                                                     │   │
│  │  负载均衡: 基于Client ID的粘性会话 (Sticky Session)                 │   │
│  │  技术栈: Go + Gorilla WebSocket / Gobwas/ws                         │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  网关内部架构:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │   │
│  │  │ Connection  │    │ Connection  │    │ Connection  │             │   │
│  │  │ Manager     │    │ Pool        │    │ Handler     │             │   │
│  │  │             │    │             │    │             │             │   │
│  │  │ - 连接注册  │    │ - 连接复用  │    │ - 消息解析  │             │   │
│  │  │ - 心跳管理  │    │ - 资源限制  │    │ - 协议升级  │             │   │
│  │  │ - 异常处理  │    │ - 负载均衡  │    │ - 消息路由  │             │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘             │   │
│  │                                                                     │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │   │
│  │  │ Message     │    │ Rate        │    │ Auth        │             │   │
│  │  │ Router      │    │ Limiter     │    │ Middleware  │             │   │
│  │  │             │    │             │    │             │             │   │
│  │  │ - 会话路由  │    │ - 令牌桶    │    │ - JWT验证   │             │   │
│  │  │ - 广播路由  │    │ - 用户限流  │    │ - 权限检查  │             │   │
│  │  │ - 服务路由  │    │ - IP限流    │    │ - 会话绑定  │             │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘             │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  消息协议设计:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  消息格式 (JSON):                                                   │   │
│  │  {                                                                  │   │
│  │    "type": "operation" | "awareness" | "sync" | "auth" | "ack",     │   │
│  │    "session_id": "uuid",                                            │   │
│  │    "client_id": "uuid",                                             │   │
│  │    "timestamp": 1699123456789,                                      │   │
│  │    "seq": 123,          // 序列号，用于消息排序和去重               │   │
│  │    "payload": { ... }                                               │   │
│  │  }                                                                  │   │
│  │                                                                     │   │
│  │  消息类型:                                                          │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ Type        │ Description                                     │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ operation   │ CRDT操作更新                                    │  │   │
│  │  │ awareness   │ 用户状态更新 (光标、选择、在线状态)             │  │   │
│  │  │ sync        │ 同步请求/响应                                   │  │   │
│  │  │ auth        │ 认证消息                                        │  │   │
│  │  │ ack         │ 消息确认                                        │  │   │
│  │  │ error       │ 错误消息                                        │  │   │
│  │  │ ping/pong   │ 心跳消息                                        │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  跨网关消息路由:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌──────────┐      ┌──────────┐      ┌──────────┐                  │   │
│  │  │ Gateway  │◄────►│  Redis   │◄────►│ Gateway  │                  │   │
│  │  │    A     │      │ Pub/Sub  │      │    B     │                  │   │
│  │  └──────────┘      └──────────┘      └──────────┘                  │   │
│  │       │                                   │                        │   │
│  │       ▼                                   ▼                        │   │
│  │  ┌──────────┐                        ┌──────────┐                  │   │
│  │  │ Client 1 │                        │ Client 2 │                  │   │
│  │  └──────────┘                        └──────────┘                  │   │
│  │                                                                     │   │
│  │  路由策略:                                                          │   │
│  │  - 基于session_id的频道订阅                                         │   │
│  │  - 消息广播到所有订阅该session的网关                                │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.3 操作广播机制

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        操作广播机制                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  广播流程:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  Client A                    Server                    Client B     │   │
│  │     │                          │                          │         │   │
│  │     │  1. 本地执行操作          │                          │         │   │
│  │     │  2. 生成CRDT更新          │                          │         │   │
│  │     ├─────────────────────────▶│                          │         │   │
│  │     │    WS: operation msg     │                          │         │   │
│  │     │                          │  3. 验证操作               │         │   │
│  │     │                          │  4. 应用CRDT更新           │         │   │
│  │     │                          │  5. 持久化到事件存储       │         │   │
│  │     │                          ├─────────────────────────▶│         │   │
│  │     │                          │    WS: broadcast msg     │         │   │
│  │     │                          │                          │  6. 应用 │   │
│  │     │                          │                          │     更新 │   │
│  │     │◄─────────────────────────┤                          │         │   │
│  │     │    WS: ack msg           │                          │         │   │
│  │     │                          │                          │         │   │
│  │  7. 确认发送                    │                          │         │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  广播策略:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  1. 全量广播 (Full Broadcast)                                       │   │
│  │     - 适用: 小型协作会话 (< 10人)                                   │   │
│  │     - 方式: 发送给会话中所有参与者                                  │   │
│  │                                                                     │   │
│  │  2. 兴趣广播 (Interest-Based Broadcast)                             │   │
│  │     - 适用: 大型协作会话                                            │   │
│  │     - 方式: 基于视口/选择集过滤接收者                               │   │
│  │     - 实现: 空间索引 + 用户兴趣注册                                 │   │
│  │                                                                     │   │
│  │  3. 分层广播 (Hierarchical Broadcast)                               │   │
│  │     - 适用: 超大型文档                                              │   │
│  │     - 方式: 按文档区域划分广播域                                    │   │
│  │     - 实现: 八叉树空间分区                                          │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  消息顺序保证:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  服务器端顺序保证:                                                  │   │
│  │  - 每个会话维护一个全局序列号生成器                                 │   │
│  │  - 消息按序列号顺序广播                                             │   │
│  │  - 客户端按序列号顺序应用                                           │   │
│  │                                                                     │   │
│  │  客户端乱序处理:                                                    │   │
│  │  - 消息缓冲区 (Message Buffer)                                      │   │
│  │  - 等待缺失消息 (最多等待 500ms)                                    │   │
│  │  - 超时后请求重传                                                   │   │
│  │                                                                     │   │
│  │  序列号管理:                                                        │   │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐                         │   │
│  │  │ Seq: 1  │───▶│ Seq: 2  │───▶│ Seq: 3  │ ──▶ ...                 │   │
│  │  │ (应用)  │    │ (应用)  │    │ (等待)  │                         │   │
│  │  └─────────┘    └─────────┘    └─────────┘                         │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  延迟优化策略:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  1. 操作合并 (Operation Merging)                                    │   │
│  │     - 连续输入合并为一个操作                                        │   │
│  │     - 减少网络往返次数                                              │   │
│  │                                                                     │   │
│  │  2. 增量更新 (Delta Updates)                                        │   │
│  │     - 只发送变更的部分                                              │   │
│  │     - 使用CRDT的diff算法                                            │   │
│  │                                                                     │   │
│  │  3. 批量发送 (Batch Sending)                                        │   │
│  │     - 16ms窗口内合并多个操作                                        │   │
│  │     - 保持60fps的流畅体验                                           │   │
│  │                                                                     │   │
│  │  4. 预测执行 (Optimistic Execution)                                 │   │
│  │     - 本地立即执行，不等待服务器确认                                │   │
│  │     - 冲突时回滚并重新应用                                          │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.4 冲突解决策略

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        冲突解决策略                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  CRDT自动冲突解决:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  CRDT保证: 强最终一致性 (Strong Eventual Consistency)               │   │
│  │                                                                     │   │
│  │  冲突自动解决原则:                                                  │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ CRDT类型    │ 冲突解决策略                                    │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ LWW-Register│ Last-Write-Wins (基于时间戳)                    │  │   │
│  │  │ G-Counter   │ 数值相加                                        │  │   │
│  │  │ PN-Counter  │ 正计数器+负计数器分别合并                       │  │   │
│  │  │ OR-Set      │ 添加集-删除集，删除优先                         │  │   │
│  │  │ LWW-Element │ 元素级别LWW                                     │  │   │
│  │  │ YATA (文本) │ 基于位置的自动合并                              │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  业务层冲突处理:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  场景1: 同时编辑同一属性                                             │   │
│  │  ┌─────────┐         ┌─────────┐         ┌─────────┐               │   │
│  │  │ User A  │         │ Server  │         │ User B  │               │   │
│  │  │ 改高度  │────────▶│         │◀────────│ 改高度  │               │   │
│  │  │ =3000   │         │         │         │ =3500   │               │   │
│  │  └─────────┘         └─────────┘         └─────────┘               │   │
│  │         │                 │                 │                       │   │
│  │         │                 ▼                 │                       │   │
│  │         │         ┌─────────────┐          │                       │   │
│  │         │         │ LWW策略     │          │                       │   │
│  │         │         │ 取最新值    │          │                       │   │
│  │         │         └─────────────┘          │                       │   │
│  │         │                 │                 │                       │   │
│  │         └────────┬────────┴────────┬────────┘                       │   │
│  │                  ▼                 ▼                                │   │
│  │            ┌─────────┐       ┌─────────┐                           │   │
│  │            │ 值=3500 │       │ 值=3500 │                           │   │
│  │            └─────────┘       └─────────┘                           │   │
│  │                                                                     │   │
│  │  场景2: 删除与修改冲突                                               │   │
│  │  ┌─────────┐         ┌─────────┐         ┌─────────┐               │   │
│  │  │ User A  │         │ Server  │         │ User B  │               │   │
│  │  │ 删除墙  │────────▶│         │◀────────│ 改墙高   │               │   │
│  │  └─────────┘         └─────────┘         └─────────┘               │   │
│  │         │                 │                 │                       │   │
│  │         │                 ▼                 │                       │   │
│  │         │         ┌─────────────┐          │                       │   │
│  │         │         │ Tombstone   │          │                       │   │
│  │         │         │ 策略        │          │                       │   │
│  │         │         │ 删除优先    │          │                       │   │
│  │         │         └─────────────┘          │                       │   │
│  │         │                 │                 │                       │   │
│  │         └─────────────────┴─────────────────┘                       │   │
│  │                           │                                         │   │
│  │                           ▼                                         │   │
│  │                     ┌─────────┐                                     │   │
│  │                     │ 墙已删除│                                     │   │
│  │                     │ 通知B   │                                     │   │
│  │                     └─────────┘                                     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  手动冲突解决UI:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  当自动解决不可行时 (如复杂合并冲突):                               │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    冲突解决对话框                            │   │   │
│  │  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │   │   │
│  │  │  │ 版本 A (我的)   │  │ 版本 B (他们的) │  │ 合并结果    │ │   │   │
│  │  │  │                 │  │                 │  │             │ │   │   │
│  │  │  │ [显示差异]      │  │ [显示差异]      │  │ [可编辑]    │ │   │   │
│  │  │  └─────────────────┘  └─────────────────┘  └─────────────┘ │   │   │
│  │  │                                                              │   │   │
│  │  │  [接受我的] [接受他们的] [合并编辑] [取消]                   │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  触发条件:                                                          │   │
│  │  - 分支合并冲突                                                     │   │
│  │  - 复杂结构化数据冲突                                               │   │
│  │  - 业务规则冲突                                                     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```


---

## 6. 部署架构

### 6.1 Kubernetes部署方案

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Kubernetes部署架构                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         K8s Cluster                                  │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                     Ingress Layer                            │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ Ingress     │  │ Ingress     │  │ SSL         │         │   │   │
│  │  │  │ Controller  │  │ Controller  │  │ Termination │         │   │   │
│  │  │  │ (Nginx)     │  │ (Nginx)     │  │ (cert-manager)│       │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                     Gateway Layer                            │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ API Gateway │  │ API Gateway │  │ HPA         │         │   │   │
│  │  │  │ (Kong)      │  │ (Kong)      │  │ (2-10 pods) │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ WS Gateway  │  │ WS Gateway  │  │ HPA         │         │   │   │
│  │  │  │ (Go)        │  │ (Go)        │  │ (3-20 pods) │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                   Microservices Layer                        │   │   │
│  │  │                                                              │   │   │
│  │  │  Namespace: app                                              │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ User Svc    │  │ Project Svc │  │ Version Svc │         │   │   │
│  │  │  │ (Java)      │  │ (Java)      │  │ (Java)      │         │   │   │
│  │  │  │ 2 replicas  │  │ 2 replicas  │  │ 2 replicas  │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ Collab Svc  │  │ Geometry Svc│  │ Property Svc│         │   │   │
│  │  │  │ (Go)        │  │ (Go)        │  │ (Java)      │         │   │   │
│  │  │  │ 3 replicas  │  │ 3 replicas  │  │ 2 replicas  │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ Script Svc  │  │ File Svc    │  │ Search Svc  │         │   │   │
│  │  │  │ (Go)        │  │ (Go)        │  │ (Java)      │         │   │   │
│  │  │  │ 2 replicas  │  │ 3 replicas  │  │ 2 replicas  │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │                                                              │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Data Layer                                │   │   │
│  │  │                                                              │   │   │
│  │  │  Namespace: data                                             │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ PostgreSQL  │  │ MongoDB     │  │ Redis       │         │   │   │
│  │  │  │ (StatefulSet)│ │ (StatefulSet)│ │ (StatefulSet)│        │   │   │
│  │  │  │ 3 replicas  │  │ 3 replicas  │  │ 3 replicas  │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ Kafka       │  │ MinIO       │  │ ES          │         │   │   │
│  │  │  │ (Strimzi)   │  │ (StatefulSet)│ │ (StatefulSet)│        │   │   │
│  │  │  │ 3 brokers   │  │ 4 replicas  │  │ 3 nodes     │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  │                                                              │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  命名空间划分:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  - ingress: 入口控制器                                               │   │
│  │  - gateway: API网关和WebSocket网关                                   │   │
│  │  - app: 业务微服务                                                   │   │
│  │  - data: 数据存储服务                                                │   │
│  │  - monitoring: 监控和日志                                            │   │
│  │  - istio-system: Service Mesh (可选)                                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 服务网格设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Service Mesh架构                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  技术选型: Istio                                                            │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Control Plane                             │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │   │   │
│  │  │  │ istiod      │  │ Citadel     │  │ Galley      │         │   │   │
│  │  │  │ (Pilot)     │  │ (安全)      │  │ (配置验证)  │         │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │                     Data Plane                               │   │   │
│  │  │                                                              │   │   │
│  │  │  ┌─────────┐    ┌─────────┐    ┌─────────┐                  │   │   │
│  │  │  │ Service │    │ Service │    │ Service │                  │   │   │
│  │  │  │ Pod     │    │ Pod     │    │ Pod     │                  │   │   │
│  │  │  │ ┌─────┐ │    │ ┌─────┐ │    │ ┌─────┐ │                  │   │   │
│  │  │  │ │ App │ │    │ │ App │ │    │ │ App │ │                  │   │   │
│  │  │  │ └──┬──┘ │    │ └──┬──┘ │    │ └──┬──┘ │                  │   │   │
│  │  │  │ ┌──┴──┐ │    │ ┌──┴──┐ │    │ ┌──┴──┐ │                  │   │   │
│  │  │  │ │Envoy│ │◄──►│ │Envoy│ │◄──►│ │Envoy│ │                  │   │   │
│  │  │  │ │Sidecar│   │ │Sidecar│   │ │Sidecar│                  │   │   │
│  │  │  │ └─────┘ │    │ └─────┘ │    │ └─────┘ │                  │   │   │
│  │  │  └─────────┘    └─────────┘    └─────────┘                  │   │   │
│  │  │                                                              │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  Service Mesh功能:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  流量管理:                                                          │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ 功能        │ 配置示例                                        │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ 负载均衡    │ round-robin, least-conn, random                 │  │   │
│  │  │ 流量分割    │ 90% v1, 10% v2 (金丝雀发布)                     │  │   │
│  │  │ 超时控制    │ 5s timeout, 3 retries                           │  │   │
│  │  │ 熔断器      │ 5 consecutive errors, 30s recovery              │  │   │
│  │  │ 故障注入    │ 10% delay, 1% abort (混沌测试)                  │  │   │
│  │  │ 镜像流量    │ 复制流量到影子服务                              │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  安全:                                                              │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ 功能        │ 说明                                            │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ mTLS        │ 服务间自动双向TLS认证                           │  │   │
│  │  │ 认证        │ JWT验证, OAuth2集成                             │  │   │
│  │  │ 授权        │ RBAC策略控制服务访问                            │  │   │
│  │  │ 审计日志    │ 记录所有服务间调用                              │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  可观测性:                                                          │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ 功能        │ 工具                                            │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ 指标        │ Prometheus + Grafana                            │  │   │
│  │  │ 分布式追踪  │ Jaeger / Zipkin                                 │  │   │
│  │  │ 日志        │ Fluentd + Elasticsearch + Kibana                │  │   │
│  │  │ 服务拓扑    │ Kiali                                           │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  Istio配置示例:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  # VirtualService - 流量路由                                        │   │
│  │  apiVersion: networking.istio.io/v1beta1                            │   │
│  │  kind: VirtualService                                               │   │
│  │  metadata:                                                          │   │
│  │    name: collaboration-service                                      │   │
│  │  spec:                                                              │   │
│  │    hosts:                                                           │   │
│  │    - collaboration-service                                          │   │
│  │    http:                                                            │   │
│  │    - route:                                                         │   │
│  │      - destination:                                                 │   │
│  │          host: collaboration-service                                │   │
│  │          subset: v1                                                 │   │
│  │        weight: 90                                                   │   │
│  │      - destination:                                                 │   │
│  │          host: collaboration-service                                │   │
│  │          subset: v2                                                 │   │
│  │        weight: 10                                                   │   │
│  │      timeout: 5s                                                    │   │
│  │      retries:                                                       │   │
│  │        attempts: 3                                                  │   │
│  │        perTryTimeout: 2s                                            │   │
│  │                                                                     │   │
│  │  # DestinationRule - 负载均衡和连接池                               │   │
│  │  apiVersion: networking.istio.io/v1beta1                            │   │
│  │  kind: DestinationRule                                              │   │
│  │  metadata:                                                          │   │
│  │    name: collaboration-service                                      │   │
│  │  spec:                                                              │   │
│  │    host: collaboration-service                                      │   │
│  │    trafficPolicy:                                                   │   │
│  │      loadBalancer:                                                  │   │
│  │        simple: LEAST_CONN                                           │   │
│  │      connectionPool:                                                │   │
│  │        tcp:                                                         │   │
│  │          maxConnections: 100                                        │   │
│  │        http:                                                        │   │
│  │          http1MaxPendingRequests: 50                                │   │
│  │      outlierDetection:                                              │   │
│  │        consecutiveErrors: 5                                         │   │
│  │        interval: 30s                                                │   │
│  │        baseEjectionTime: 30s                                        │   │
│  │    subsets:                                                         │   │
│  │    - name: v1                                                       │   │
│  │      labels:                                                        │   │
│  │        version: v1                                                  │   │
│  │    - name: v2                                                       │   │
│  │      labels:                                                        │   │
│  │        version: v2                                                  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.3 负载均衡策略

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        负载均衡策略                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  多层负载均衡架构:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  Layer 1: DNS负载均衡 (Global)                                      │   │
│  │  ┌─────────────┐                                                    │   │
│  │  │ Cloudflare  │  地理位置路由 + 健康检查                            │   │
│  │  │ / Route53   │                                                    │   │
│  │  └─────────────┘                                                    │   │
│  │         │                                                           │   │
│  │         ▼                                                           │   │
│  │  Layer 2: L4负载均衡 (Regional)                                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │ MetalLB     │  │ AWS NLB     │  │ 阿里云SLB   │                 │   │
│  │  │ (Bare Metal)│  │ (Cloud)     │  │ (Cloud)     │                 │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │
│  │         │                                                           │   │
│  │         ▼                                                           │   │
│  │  Layer 3: L7负载均衡 (Kubernetes)                                   │   │
│  │  ┌─────────────┐  ┌─────────────┐                                   │   │
│  │  │ Nginx       │  │ Kong        │  Ingress Controller               │   │
│  │  │ Ingress     │  │ Ingress     │                                   │   │
│  │  └─────────────┘  └─────────────┘                                   │   │
│  │         │                                                           │   │
│  │         ▼                                                           │   │
│  │  Layer 4: 服务发现负载均衡 (Service Mesh)                           │   │
│  │  ┌─────────────┐  ┌─────────────┐                                   │   │
│  │  │ Istio Envoy │  │ Client-side │                                   │   │
│  │  │ Sidecar LB  │  │ gRPC LB     │                                   │   │
│  │  └─────────────┘  └─────────────┘                                   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  各服务负载均衡策略:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌─────────────┬──────────────────┬─────────────────────────────┐  │   │
│  │  │ 服务        │ 负载均衡算法     │ 特殊配置                    │  │   │
│  │  ├─────────────┼──────────────────┼─────────────────────────────┤  │   │
│  │  │ API Gateway │ Round Robin      │ 基于请求hash的会话保持      │  │   │
│  │  │ WS Gateway  │ IP Hash          │ 粘性会话，同一客户端固定    │  │   │
│  │  │ Collab Svc  │ Least Connection │ 考虑连接数，避免过载        │  │   │
│  │  │ Geometry Svc│ Round Robin      │ CPU密集型，均匀分配         │  │   │
│  │  │ Property Svc│ Random           │ 无状态服务                  │  │   │
│  │  │ Script Svc  │ Least Response   │ 考虑执行队列长度            │  │   │
│  │  │ File Svc    │ Round Robin      │ 大文件上传需要会话保持      │  │   │
│  │  └─────────────┴──────────────────┴─────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  WebSocket会话保持:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  问题: WebSocket连接需要保持长连接，不能随意切换后端                 │   │
│  │                                                                     │   │
│  │  解决方案:                                                          │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ Client  │───▶ 连接时携带 client_id                               │   │
│  │  └────┬────┘                                                        │   │
│  │       │                                                             │   │
│  │       ▼                                                             │   │
│  │  ┌─────────┐    基于 client_id 计算 hash                            │   │
│  │  │  LB     │───▶ 路由到固定的后端实例                               │   │
│  │  └─────────┘                                                        │   │
│  │       │                                                             │   │
│  │       ▼                                                             │   │
│  │  ┌─────────┐    同一 client_id 始终路由到同一 Pod                    │   │
│  │  │ Pod X   │◄─── 保证会话连续性                                     │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  │  实现方式:                                                          │   │
│  │  - Nginx: ip_hash 或 hash $arg_client_id consistent                │   │
│  │  - Istio: consistentHash (基于header)                              │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.4 自动扩缩容

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        自动扩缩容设计                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  HPA (Horizontal Pod Autoscaler)配置:                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  apiVersion: autoscaling/v2                                         │   │
│  │  kind: HorizontalPodAutoscaler                                      │   │
│  │  metadata:                                                          │   │
│  │    name: collaboration-service-hpa                                  │   │
│  │  spec:                                                              │   │
│  │    scaleTargetRef:                                                  │   │
│  │      apiVersion: apps/v1                                            │   │
│  │      kind: Deployment                                               │   │
│  │      name: collaboration-service                                    │   │
│  │    minReplicas: 3                                                   │   │
│  │    maxReplicas: 50                                                  │   │
│  │    metrics:                                                         │   │
│  │    - type: Resource                                                 │   │
│  │      resource:                                                      │   │
│  │        name: cpu                                                    │   │
│  │        target:                                                      │   │
│  │          type: Utilization                                          │   │
│  │          averageUtilization: 70                                     │   │
│  │    - type: Resource                                                 │   │
│  │      resource:                                                      │   │
│  │        name: memory                                                 │   │
│  │        target:                                                      │   │
│  │          type: Utilization                                          │   │
│  │          averageUtilization: 80                                     │   │
│  │    - type: Pods                                                     │   │
│  │      pods:                                                          │   │
│  │        metric:                                                      │   │
│  │          name: websocket_connections                                │   │
│  │        target:                                                      │   │
│  │          type: AverageValue                                         │   │
│  │          averageValue: "8000"                                       │   │
│  │    behavior:                                                        │   │
│  │      scaleUp:                                                       │   │
│  │        stabilizationWindowSeconds: 60                               │   │
│  │        policies:                                                    │   │
│  │        - type: Percent                                              │   │
│  │          value: 100                                                 │   │
│  │          periodSeconds: 60                                          │   │
│  │      scaleDown:                                                     │   │
│  │        stabilizationWindowSeconds: 300                              │   │
│  │        policies:                                                    │   │
│  │        - type: Percent                                              │   │
│  │          value: 10                                                  │   │
│  │          periodSeconds: 60                                          │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  各服务HPA配置:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌─────────────┬─────────┬─────────┬─────────────────────────────┐  │   │
│  │  │ 服务        │ Min     │ Max     │ 扩缩容指标                  │  │   │
│  │  ├─────────────┼─────────┼─────────┼─────────────────────────────┤  │   │
│  │  │ API Gateway │ 2       │ 10      │ CPU 70%, RPS 1000           │  │   │
│  │  │ WS Gateway  │ 3       │ 50      │ WS连接数 8000, CPU 70%      │  │   │
│  │  │ Collab Svc  │ 3       │ 30      │ CPU 70%, 内存 80%           │  │   │
│  │  │ Geometry Svc│ 2       │ 20      │ CPU 70%, 队列长度 100       │  │   │
│  │  │ Property Svc│ 2       │ 15      │ CPU 70%, 内存 80%           │  │   │
│  │  │ Script Svc  │ 2       │ 20      │ 执行队列长度 50, CPU 70%    │  │   │
│  │  │ File Svc    │ 3       │ 20      │ CPU 70%, 带宽使用率 80%     │  │   │
│  │  │ Search Svc  │ 2       │ 10      │ CPU 70%, 查询延迟 100ms     │  │   │
│  │  └─────────────┴─────────┴─────────┴─────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  VPA (Vertical Pod Autoscaler)配置:                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  适用场景: 有状态服务 (数据库、缓存)                                 │   │
│  │                                                                     │   │
│  │  apiVersion: autoscaling.k8s.io/v1                                  │   │
│  │  kind: VerticalPodAutoscaler                                        │   │
│  │  metadata:                                                          │   │
│  │    name: redis-vpa                                                  │   │
│  │  spec:                                                              │   │
│  │    targetRef:                                                       │   │
│  │      apiVersion: apps/v1                                            │   │
│  │      kind: StatefulSet                                              │   │
│  │      name: redis                                                    │   │
│  │    updatePolicy:                                                    │   │
│  │      updateMode: "Auto"                                             │   │
│  │    resourcePolicy:                                                  │   │
│  │      containerPolicies:                                             │   │
│  │      - containerName: redis                                         │   │
│  │        minAllowed:                                                  │   │
│  │          cpu: 100m                                                  │   │
│  │          memory: 512Mi                                              │   │
│  │        maxAllowed:                                                  │   │
│  │          cpu: 4000m                                                 │   │
│  │          memory: 16Gi                                               │   │
│  │        controlledResources: ["cpu", "memory"]                       │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  Cluster Autoscaler:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  触发条件:                                                          │   │
│  │  - Pod处于Pending状态，资源不足                                     │   │
│  │  - 节点利用率低于阈值，可以缩容                                     │   │
│  │                                                                     │   │
│  │  节点池配置:                                                        │   │
│  │  ┌─────────────┬─────────────┬─────────────┬─────────────────────┐  │   │
│  │  │ 节点池      │ 实例类型    │ 最小/最大   │ 用途                │  │   │
│  │  ├─────────────┼─────────────┼─────────────┼─────────────────────┤  │   │
│  │  │ general     │ 4C8G        │ 3/20        │ 通用服务            │  │   │
│  │  │ compute     │ 8C16G       │ 2/30        │ 计算密集型          │  │   │
│  │  │ memory      │ 4C32G       │ 2/15        │ 内存密集型          │  │   │
│  │  │ gpu         │ GPU实例     │ 0/5         │ AI/渲染任务         │  │   │
│  │  └─────────────┴─────────────┴─────────────┴─────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  KEDA (事件驱动自动扩缩容):                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  适用场景: 基于Kafka消息队列的自动扩缩容                             │   │
│  │                                                                     │   │
│  │  apiVersion: keda.sh/v1alpha1                                       │   │
│  │  kind: ScaledObject                                                 │   │
│  │  metadata:                                                          │   │
│  │    name: script-processor-scaler                                    │   │
│  │  spec:                                                              │   │
│  │    scaleTargetRef:                                                  │   │
│  │      name: script-processor                                         │   │
│  │    minReplicaCount: 2                                               │   │
│  │    maxReplicaCount: 50                                              │   │
│  │    triggers:                                                        │   │
│  │    - type: kafka                                                    │   │
│  │      metadata:                                                      │   │
│  │        bootstrapServers: kafka:9092                                 │   │
│  │        consumerGroup: script-processor-group                        │   │
│  │        topic: script-execution-requests                             │   │
│  │        lagThreshold: "100"                                          │   │
│  │        activationLagThreshold: "10"                                 │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```


---

## 7. 非功能性设计

### 7.1 性能设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        性能设计                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  缓存策略:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  多级缓存架构:                                                      │   │
│  │                                                                     │   │
│  │  L1: 本地缓存 (Caffeine/Go-Cache)                                   │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │  - 服务实例本地缓存                                           │   │   │
│  │  │  - 容量: 10MB per instance                                    │   │   │
│  │  │  - TTL: 5 minutes                                             │   │   │
│  │  │  - 适用: 热点数据、配置信息                                   │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  L2: 分布式缓存 (Redis Cluster)                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │  - 共享缓存层                                                 │   │   │
│  │  │  - 容量: 32GB cluster                                         │   │   │
│  │  │  - 策略:                                                      │   │   │
│  │  │    ├── Cache-Aside (旁路缓存)                                 │   │   │
│  │  │    ├── Write-Through (写穿透)                                 │   │   │
│  │  │    └── Write-Behind (异步写)                                  │   │   │
│  │  │  - 适用: 用户会话、几何数据缓存、搜索结果                     │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                              │                                      │   │
│  │                              ▼                                      │   │
│  │  L3: CDN缓存 (静态资源)                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │  - 静态资源、模型预览图                                       │   │   │
│  │  │  - TTL: 1 day                                                 │   │   │
│  │  │  - 全球节点分发                                               │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  缓存Key设计:                                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  命名规范: {service}:{entity}:{id}:{version}                        │   │
│  │                                                                     │   │
│  │  示例:                                                              │   │
│  │  - user:profile:{user_id}:v1                                        │   │
│  │  - geometry:doc:{doc_id}:v{version}                                 │   │
│  │  - project:members:{project_id}:v1                                  │   │
│  │                                                                     │   │
│  │  缓存失效策略:                                                      │   │
│  │  - 主动失效: 数据变更时发送缓存失效消息                             │   │
│  │  - 被动失效: TTL到期自动失效                                        │   │
│  │  - 版本控制: 数据版本号变化时自动失效                               │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  异步处理:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  异步任务场景:                                                      │   │
│  │  ┌────────────────┬─────────────────────────────────────────────┐  │   │
│  │  │ 场景           │ 处理方式                                    │  │   │
│  │  ├────────────────┼─────────────────────────────────────────────┤  │   │
│  │  │ BIM文件解析    │ 异步任务队列，进度通知                      │  │   │
│  │  │ 几何计算       │ 计算队列，结果回调                          │  │   │
│  │  │ 脚本执行       │ 沙箱执行，异步返回                          │  │   │
│  │  │ 报表生成       │ 后台任务，完成后通知                        │  │   │
│  │  │ 批量导出       │ 任务队列，文件生成后下载                    │  │   │
│  │  │ 搜索索引更新   │ 事件驱动，异步更新                          │  │   │
│  │  │ 通知发送       │ 消息队列，批量处理                          │  │   │
│  │  └────────────────┴─────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  任务队列设计:                                                      │   │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │   │
│  │  │ 生产者  │───▶│  Redis  │───▶│ 消费者  │───▶│ 结果存储│         │   │
│  │  │         │    │  Queue  │    │  Worker │    │         │         │   │
│  │  └─────────┘    └─────────┘    └─────────┘    └─────────┘         │   │
│  │                                                                     │   │
│  │  技术选型: Redis Streams / RabbitMQ                                 │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  数据库优化:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  PostgreSQL优化:                                                    │   │
│  │  - 读写分离: 主库写，从库读 (Streaming Replication)                 │   │
│  │  - 连接池: PgBouncer (max 1000 connections)                         │   │
│  │  - 分区表: 按时间分区 (events, logs)                                │   │
│  │  - 索引优化: B-tree, GIN, GiST (几何索引)                           │   │
│  │                                                                     │   │
│  │  MongoDB优化:                                                       │   │
│  │  - 分片集群: 按project_id分片                                       │   │
│  │  - 复合索引: 常用查询字段组合                                       │   │
│  │  - 预聚合: 统计类查询使用物化视图                                   │   │
│  │                                                                     │   │
│  │  Redis优化:                                                         │   │
│  │  - 集群模式: 6节点 (3主3从)                                         │   │
│  │  - 内存策略: allkeys-lru (LRU淘汰)                                  │   │
│  │  - Pipeline: 批量操作减少RTT                                        │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  性能指标目标:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌────────────────┬─────────────────────────────────────────────┐  │   │
│  │  │ 指标           │ 目标值                                      │  │   │
│  │  ├────────────────┼─────────────────────────────────────────────┤  │   │
│  │  │ API响应时间    │ P99 < 200ms                                 │  │   │
│  │  │ WebSocket延迟  │ < 50ms (同区域)                             │  │   │
│  │  │ 几何计算       │ 简单操作 < 100ms, 复杂操作 < 5s             │  │   │
│  │  │ BIM解析        │ 100MB文件 < 30s                             │  │   │
│  │  │ 搜索查询       │ < 200ms                                     │  │   │
│  │  │ 并发用户数     │ 单实例支持 1000 并发                        │  │   │
│  │  │ 系统吞吐量     │ 10,000 QPS                                  │  │   │
│  │  └────────────────┴─────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.2 可扩展性设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        可扩展性设计                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  水平扩展策略:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  无状态服务:                                                        │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ 特征    │  不存储会话状态，请求可在任意实例处理                  │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 服务    │  API Gateway, Geometry Service, Property Service     │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 扩展    │  水平扩展，HPA自动扩缩容                               │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  │  有状态服务:                                                        │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ 特征    │  维护会话状态，需要粘性路由                            │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 服务    │  WebSocket Gateway, Collaboration Service            │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 扩展    │  会话分片，基于client_id路由                           │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  │  数据分片:                                                          │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ 策略    │  按project_id或user_id分片                             │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 实现    │  MongoDB Sharding, PostgreSQL Partitioning           │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 路由    │  应用层路由或数据库代理                                │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  多租户架构:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  租户隔离策略:                                                      │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │ 方案A: 共享数据库，租户ID隔离 (推荐)                          │   │   │
│  │  │  - 所有租户共享同一数据库实例                                 │   │   │
│  │  │  - 每张表包含tenant_id字段                                    │   │   │
│  │  │  - 应用层自动添加tenant_id过滤                                │   │   │
│  │  │  - 优点: 成本低，易于管理                                     │   │   │
│  │  │  - 缺点: 需要严格的数据隔离保证                               │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │ 方案B: 独立Schema隔离                                         │   │   │
│  │  │  - 每个租户独立Schema                                         │   │   │
│  │  │  - 共享数据库实例                                             │   │   │
│  │  │  - 优点: 数据隔离更好                                         │   │   │
│  │  │  - 缺点: Schema管理复杂                                       │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────┐   │   │
│  │  │ 方案C: 独立数据库实例 (VIP租户)                               │   │   │
│  │  │  - 大客户独立数据库实例                                       │   │   │
│  │  │  - 优点: 最高隔离级别，可定制                                 │   │   │
│  │  │  - 缺点: 成本高                                               │   │   │
│  │  └─────────────────────────────────────────────────────────────┘   │   │
│  │                                                                     │   │
│  │  实施方案: 混合策略                                                 │   │
│  │  - 普通租户: 方案A                                                  │   │
│  │  - 企业租户: 方案B                                                  │   │
│  │  - VIP租户:  方案C                                                  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  插件化架构:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  扩展点设计:                                                        │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │ 导入插件    │  │ 导出插件    │  │ 计算插件    │  │ 检查插件   │ │   │
│  │  │             │  │             │  │             │  │            │ │   │
│  │  │ - IFC导入   │  │ - IFC导出   │  │ - 能耗计算  │  │ - 规范检查 │ │   │
│  │  │ - Revit导入 │  │ - DWG导出   │  │ - 结构分析  │  │ - 碰撞检测 │ │   │
│  │  │ - 其他格式  │  │ - PDF导出   │  │ - 其他算法  │  │ - 其他规则 │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  │                                                                     │   │
│  │  插件接口:                                                          │   │
│  │  ```go                                                              │   │
│  │  type Plugin interface {                                            │   │
│  │      Name() string                                                  │   │
│  │      Version() string                                               │   │
│  │      Initialize(config Config) error                                │   │
│  │      Execute(ctx context.Context, input Input) (Output, error)      │   │
│  │  }                                                                  │   │
│  │  ```                                                                │   │
│  │                                                                     │   │
│  │  插件管理:                                                          │   │
│  │  - 动态加载/卸载                                                    │   │
│  │  - 版本管理                                                         │   │
│  │  - 沙箱执行                                                         │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.3 可用性设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        可用性设计                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  高可用架构:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  服务层高可用:                                                      │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ 多实例  │  每个服务至少2个实例，跨可用区部署                     │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 健康检查│  Liveness + Readiness Probe                            │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 故障转移│  K8s自动重启，Istio熔断                                │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 优雅关闭│  PreStop Hook，处理完当前请求                          │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  │  数据层高可用:                                                      │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ PostgreSQL │ 主从复制 + 自动故障转移 (Patroni)                   │   │
│  │  ├─────────┤                                                        │   │
│  │  │ MongoDB │ Replica Set (3节点) + 仲裁节点                         │   │
│  │  ├─────────┤                                                        │   │
│  │  │ Redis   │ Cluster模式 (6节点，3主3从)                            │   │
│  │  ├─────────┤                                                        │   │
│  │  │ Kafka   │ 3 Broker + 3 ZooKeeper，副本因子3                      │   │
│  │  ├─────────┤                                                        │   │
│  │  │ MinIO   │ 分布式模式 (4节点)，纠删码保护                         │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  多活架构:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌───────────────┐              ┌───────────────┐                   │   │
│  │  │  Region A     │              │  Region B     │                   │   │
│  │  │  (主)         │◄────────────►│  (备)         │                   │   │
│  │  │               │   数据同步   │               │                   │   │
│  │  │  ┌─────────┐  │              │  ┌─────────┐  │                   │   │
│  │  │  │ K8s     │  │              │  │ K8s     │  │                   │   │
│  │  │  │ Cluster │  │              │  │ Cluster │  │                   │   │
│  │  │  └─────────┘  │              │  └─────────┘  │                   │   │
│  │  │  ┌─────────┐  │              │  ┌─────────┐  │                   │   │
│  │  │  │ Database│  │              │  │ Database│  │                   │   │
│  │  │  │ (Master)│  │              │  │ (Replica)│  │                   │   │
│  │  │  └─────────┘  │              │  └─────────┘  │                   │   │
│  │  └───────────────┘              └───────────────┘                   │   │
│  │                                                                     │   │
│  │  数据同步:                                                          │   │
│  │  - PostgreSQL: 流复制 (Streaming Replication)                       │   │
│  │  - MongoDB: 副本集同步                                              │   │
│  │  - Kafka: MirrorMaker 2 跨集群复制                                  │   │
│  │                                                                     │   │
│  │  故障切换:                                                          │   │
│  │  - DNS切换: 健康检查失败时切换DNS记录                               │   │
│  │  - 切换时间: RTO < 5分钟                                            │   │
│  │  - 数据丢失: RPO < 1分钟                                            │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  灾备设计:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  备份策略:                                                          │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ 数据类型    │ 备份策略                                        │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ PostgreSQL  │ 每日全量 + 实时WAL归档                          │  │   │
│  │  │ MongoDB     │ 每日全量 + Oplog增量                            │  │   │
│  │  │ 文件存储    │ 跨区域复制 + 版本控制                           │  │   │
│  │  │ Kafka       │ 数据保留7天，重要主题持久化到对象存储           │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  恢复演练:                                                          │   │
│  │  - 每季度进行一次灾备演练                                           │   │
│  │  - 验证备份数据可恢复性                                             │   │
│  │  - 记录RTO/RPO实际值                                                │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  可用性指标:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  ┌────────────────┬─────────────────────────────────────────────┐  │   │
│  │  │ 指标           │ 目标                                        │  │   │
│  │  ├────────────────┼─────────────────────────────────────────────┤  │   │
│  │  │ 系统可用性     │ 99.9% (年停机时间 < 8.76小时)               │  │   │
│  │  │ 核心服务可用性 │ 99.95% (年停机时间 < 4.38小时)              │  │   │
│  │  │ 数据持久性     │ 99.9999999% (11个9)                         │  │   │
│  │  │ RTO            │ < 5分钟                                     │  │   │
│  │  │ RPO            │ < 1分钟                                     │  │   │
│  │  └────────────────┴─────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.4 安全性设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        安全性设计                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  身份认证与授权:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  认证方式:                                                          │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ 方式        │ 说明                                            │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ JWT         │ 访问令牌，有效期15分钟                          │  │   │
│  │  │ Refresh Token│ 刷新令牌，有效期7天                            │  │   │
│  │  │ OAuth2      │ 第三方登录 (Google, Microsoft)                  │  │   │
│  │  │ SAML        │ 企业SSO集成                                     │  │   │
│  │  │ MFA         │ 多因素认证 (TOTP, SMS)                          │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  授权模型: RBAC + ABAC                                              │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ RBAC    │  角色: Owner, Admin, Editor, Viewer                  │   │
│  │  ├─────────┤                                                        │   │
│  │  │ ABAC    │  属性: 部门、项目、时间、位置等                      │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  数据安全:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  传输加密:                                                          │   │
│  │  - TLS 1.3 (外部通信)                                               │   │
│  │  - mTLS (服务间通信，Istio)                                         │   │
│  │                                                                     │   │
│  │  存储加密:                                                          │   │
│  │  - 数据库: 透明数据加密 (TDE)                                       │   │
│  │  - 对象存储: 服务端加密 (SSE-S3)                                    │   │
│  │  - 密钥管理: HashiCorp Vault / AWS KMS                              │   │
│  │                                                                     │   │
│  │  敏感数据处理:                                                      │   │
│  │  - 密码: bcrypt哈希存储                                             │   │
│  │  - PII: 脱敏显示，加密存储                                          │   │
│  │  - API密钥: 加密存储，定期轮换                                      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  网络安全:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  网络隔离:                                                          │   │
│  │  ┌─────────┐                                                        │   │
│  │  │ 公网    │  Ingress, API Gateway                                  │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 内网    │  微服务 (不可直接从公网访问)                           │   │
│  │  ├─────────┤                                                        │   │
│  │  │ 数据网  │  数据库 (仅允许内网访问)                               │   │
│  │  └─────────┘                                                        │   │
│  │                                                                     │   │
│  │  安全组规则:                                                        │   │
│  │  - 最小权限原则                                                     │   │
│  │  - 端口白名单                                                       │   │
│  │  - IP白名单 (管理接口)                                              │   │
│  │                                                                     │   │
│  │  DDoS防护:                                                          │   │
│  │  - Cloudflare / AWS Shield                                          │   │
│  │  - 速率限制 (API Gateway)                                           │   │
│  │  - WAF规则                                                          │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  代码安全:                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  安全开发流程:                                                      │   │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │   │
│  │  │ 代码扫描│───▶│ 依赖检查│───▶│ 安全测试│───▶│ 镜像扫描│         │   │
│  │  │ (SAST)  │    │ (SCA)   │    │ (DAST)  │    │         │         │   │
│  │  └─────────┘    └─────────┘    └─────────┘    └─────────┘         │   │
│  │                                                                     │   │
│  │  工具链:                                                            │   │
│  │  - SAST: SonarQube, Semgrep                                         │   │
│  │  - SCA: Snyk, OWASP Dependency Check                                │   │
│  │  - DAST: OWASP ZAP                                                  │   │
│  │  - 镜像扫描: Trivy, Clair                                           │   │
│  │                                                                     │   │
│  │  安全编码规范:                                                      │   │
│  │  - OWASP Top 10防护                                                 │   │
│  │  - 输入验证与净化                                                   │   │
│  │  - SQL注入防护 (使用ORM/参数化查询)                                 │   │
│  │  - XSS防护 (输出编码)                                               │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  审计与监控:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  审计日志:                                                          │   │
│  │  - 用户操作日志 (登录、数据访问、修改)                              │   │
│  │  - 管理员操作日志                                                   │   │
│  │  - 保留期限: 1年                                                    │   │
│  │                                                                     │   │
│  │  安全监控:                                                          │   │
│  │  - 异常登录检测 (异地、频繁失败)                                    │   │
│  │  - 数据访问异常 (批量导出、非工作时间)                              │   │
│  │  - 告警机制 (PagerDuty, 钉钉)                                       │   │
│  │                                                                     │   │
│  │  合规性:                                                            │   │
│  │  - 等保2.0 (三级)                                                   │   │
│  │  - ISO 27001                                                        │   │
│  │  - GDPR (数据保护)                                                  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  脚本服务安全:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │  沙箱隔离:                                                          │   │
│  │  ┌─────────────┬─────────────────────────────────────────────────┐  │   │
│  │  │ 技术        │ 说明                                            │  │   │
│  │  ├─────────────┼─────────────────────────────────────────────────┤  │   │
│  │  │ gVisor      │ 用户态内核，系统调用拦截                        │  │   │
│  │  │ Firecracker │ MicroVM，轻量级虚拟机                           │  │   │
│  │  │ seccomp     │ 系统调用过滤                                    │  │   │
│  │  │ AppArmor    │ 强制访问控制                                    │  │   │
│  │  └─────────────┴─────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  资源限制:                                                          │   │
│  │  - CPU: 1 core                                                      │   │
│  │  - 内存: 512MB                                                      │   │
│  │  - 磁盘: 100MB                                                      │   │
│  │  - 网络: 禁止外网访问                                               │   │
│  │  - 执行时间: 30秒超时                                               │   │
│  │                                                                     │   │
│  │  API权限控制:                                                       │   │
│  │  - 白名单机制 (只允许调用特定API)                                   │   │
│  │  - 调用频率限制                                                     │   │
│  │  - 敏感操作审计                                                     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 附录

### A. 技术选型清单

| 类别 | 技术 | 版本 | 用途 |
|------|------|------|------|
| 编程语言 | Go | 1.21+ | 高性能服务 |
| 编程语言 | Java | 17+ | 业务服务 |
| 框架 | Spring Boot | 3.x | Java微服务 |
| 框架 | Gin | 1.9+ | Go Web框架 |
| 数据库 | PostgreSQL | 15+ | 主数据库 |
| 数据库 | MongoDB | 6.0+ | 文档存储 |
| 缓存 | Redis | 7.0+ | 分布式缓存 |
| 消息队列 | Apache Kafka | 3.5+ | 事件流 |
| 消息队列 | NATS | 2.10+ | 实时消息 |
| 对象存储 | MinIO | 最新 | 文件存储 |
| 搜索 | Elasticsearch | 8.x | 全文搜索 |
| 容器编排 | Kubernetes | 1.28+ | 容器管理 |
| 服务网格 | Istio | 1.19+ | 服务治理 |
| 网关 | Kong | 3.x | API网关 |
| 监控 | Prometheus/Grafana | 最新 | 监控告警 |
| 追踪 | Jaeger | 最新 | 分布式追踪 |
| 日志 | ELK Stack | 8.x | 日志分析 |

### B. 接口版本演进计划

| 版本 | 状态 | 发布时间 | 主要特性 |
|------|------|----------|----------|
| v1.0 | 稳定 | 2024-Q1 | 基础CRUD、协作 |
| v1.1 | 开发中 | 2024-Q2 | 高级查询、批量操作 |
| v2.0 Beta | 计划中 | 2024-Q3 | GraphQL增强、实时订阅 |
| v2.0 | 计划中 | 2024-Q4 | 全新API设计 |

### C. 性能基准测试计划

| 测试项 | 目标 | 测试工具 |
|--------|------|----------|
| API响应时间 | P99 < 200ms | k6 |
| WebSocket并发 | 10k连接/实例 | WebSocket Bench |
| 几何计算 | 1000 ops/s | 自定义 |
| BIM解析 | 100MB < 30s | 自定义 |
| 数据库查询 | < 50ms | pgbench |

---

## 文档修订记录

| 版本 | 日期 | 修订人 | 修订内容 |
|------|------|--------|----------|
| v1.0 | 2024-01 | 架构组 | 初始版本 |

---

*本文档为半自动化建筑设计平台概要设计阶段的系统架构设计报告，用于技术评审和开发指导。*
