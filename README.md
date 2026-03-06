# FutureBase - 半自动化建筑设计平台

<p align="center">
  <img src="https://img.shields.io/badge/Architecture-Microservices-blue.svg" alt="Architecture">
  <img src="https://img.shields.io/badge/Java-17-orange.svg" alt="Java">
  <img src="https://img.shields.io/badge/Go-1.21+-cyan.svg" alt="Go">
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
  <br>
  <img src="https://github.com/samecos/FutureBase/actions/workflows/ci.yml/badge.svg" alt="CI">
  <img src="https://github.com/samecos/FutureBase/actions/workflows/cd.yml/badge.svg" alt="CD">
</p>

<p align="center">
  <b>企业级建筑设计协作平台 | 实时协作 | 几何计算 | 脚本自动化</b>
</p>

---

## 📖 项目简介

**FutureBase** 是一个面向建筑设计团队的综合性 SaaS 解决方案，提供实时协作、版本控制、高级几何处理和脚本自动化功能。平台采用微服务架构，结合 Java 和 Go 技术栈，支持海量并发和大规模数据处理。

### 🎯 核心特性

| 特性 | 描述 | 技术亮点 |
|------|------|----------|
| 🔄 **实时协作** | 多人同时编辑设计文档 | Yjs CRDT + WebSocket |
| 🏗️ **几何引擎** | 2D/3D 几何计算与布尔运算 | PostGIS + OCCT/CGAL |
| 📝 **脚本自动化** | Python 脚本执行与编排 | gVisor 沙箱 + Temporal 工作流 |
| 📊 **版本控制** | 类 Git 的分支与合并 | 事件溯源 + 快照 |
| 🔐 **企业安全** | RBAC 权限 + MFA 认证 | JWT + TOTP |
| 📈 **实时监控** | 性能指标与日志追踪 | Prometheus + Grafana |

---

## 🏗️ 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              前端层 (Frontend)                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │
│  │ 脚本编辑器   │  │ 工作流编排   │  │ 执行监控     │  │ 设计画布         │ │
│  │ CodeMirror 6 │  │ 可视化界面   │  │ 实时日志     │  │ 3D/2D 渲染       │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼ REST API / WebSocket
┌─────────────────────────────────────────────────────────────────────────────┐
│                         API 网关层 (Kong Gateway)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │
│  │ 认证授权     │  │ 请求路由     │  │ 速率限制     │  │ 负载均衡         │ │
│  │ JWT/OAuth2   │  │ 路径匹配     │  │ 令牌桶       │  │ 轮询/最小连接    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
        ┌──────────────────────────────┼──────────────────────────────┐
        │                              │                              │
        ▼                              ▼                              ▼
┌───────────────┐            ┌──────────────────┐            ┌──────────────────┐
│  Java 服务    │            │ Go 服务          │            │ 外部服务         │
│  (Spring Boot)│            │ (高性能微服务)   │            │                  │
├───────────────┤            ├──────────────────┤            ├──────────────────┤
│ • user-service│            │ • collaboration  │            │ • Temporal       │
│ • project-svc │            │ • geometry       │            │ • Kafka          │
│ • property-svc│            │ • script         │            │ • Elasticsearch  │
│ • version-svc │            │ • file           │            │                  │
│ • search-svc  │            │ • notification   │            │                  │
│               │            │ • analytics      │            │                  │
└───────┬───────┘            └────────┬─────────┘            └──────────────────┘
        │                             │
        └─────────────────────────────┘
                      │
        ┌─────────────┼─────────────┬─────────────┐
        ▼             ▼             ▼             ▼
┌──────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ PostgreSQL   │ │  Redis   │ │  MinIO   │ │  Kafka   │
│ + PostGIS    │ │ (缓存)   │ │ (对象)   │ │ (消息)   │
└──────────────┘ └──────────┘ └──────────┘ └──────────┘
```

### 技术栈

#### 后端服务

| 服务类型 | 技术栈 | 服务列表 |
|----------|--------|----------|
| **Java 服务** | Spring Boot 3.x, JDK 17 | 用户、项目、属性、版本、搜索 |
| **Go 服务** | Go 1.21+, Gin/Echo | 协作、几何、脚本、文件、通知、分析 |

#### 数据存储

| 组件 | 技术选型 | 用途 |
|------|----------|------|
| **主数据库** | PostgreSQL + PostGIS | 事务数据 + 几何数据存储 |
| **分布式 SQL** | YugabyteDB / CockroachDB | 多租户数据存储 |
| **缓存层** | Redis Cluster | 热点数据、会话缓存 |
| **对象存储** | MinIO | 文件、日志、大对象 |
| **搜索引擎** | Elasticsearch | 全文搜索、聚合分析 |
| **消息队列** | Apache Kafka | 事件流、异步处理 |

#### 基础设施

| 组件 | 技术选型 |
|------|----------|
| **API 网关** | Kong (JWT、速率限制、CORS) |
| **工作流引擎** | Temporal ( durable execution ) |
| **容器编排** | Kubernetes |
| **监控** | Prometheus + Grafana |
| **日志** | ELK Stack / Loki |
| **脚本沙箱** | gVisor (runsc) + Kubernetes |

---

## 🚀 快速开始

### 环境要求

- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **Make**: 4.0+
- **JDK**: 17 (Java 开发)
- **Go**: 1.21+ (Go 开发)
- **Maven**: 3.8+

### 1. 克隆仓库

```bash
git clone https://github.com/samecos/FutureBase.git
cd FutureBase
```

### 2. 启动基础设施

```bash
cd backend

# 启动所有基础设施服务 (PostgreSQL, Redis, MinIO, Kafka, etc.)
docker-compose -f deployments/docker/docker-compose.full.yml up -d

# 检查服务状态
make health-check
```

### 3. 构建服务

```bash
# 构建所有服务
make build

# 仅构建 Java 服务
make build-java

# 仅构建 Go 服务
make build-go
```

### 4. 运行测试

```bash
# 运行单元测试
make test

# 运行集成测试
make test-integration

# 运行 E2E 测试
make test-e2e

# 生成测试覆盖率报告
make test-coverage
```

### 5. 启动服务

```bash
# 使用 Docker Compose 启动所有服务
make docker-compose-up

# 查看日志
make docker-compose-logs

# 停止服务
make docker-compose-down
```

### 6. 访问服务

| 服务 | 地址 | 说明 |
|------|------|------|
| Kong Gateway | http://localhost:8000 | API 入口 |
| Kong Admin | http://localhost:8001 | 网关管理 |
| Grafana | http://localhost:3000 | 监控面板 (admin/admin) |
| Prometheus | http://localhost:9090 | 指标查询 |

---

## 📦 服务详情

### Java 服务 (Spring Boot)

| 服务 | 端口 | 描述 | 核心功能 |
|------|------|------|----------|
| **user-service** | 8081 | 用户管理 | JWT 认证、MFA(TOTP)、RBAC 权限 |
| **project-service** | 8082 | 项目管理 | CRUD、成员角色、项目锁定 |
| **property-service** | 8083 | 属性引擎 | MVEL 规则引擎、单位换算 |
| **version-service** | 8084 | 版本控制 | Git-like 分支、合并冲突检测 |
| **search-service** | 8089 | 全文搜索 | Elasticsearch 集成、聚合查询 |

### Go 服务 (高性能)

| 服务 | 端口 | 描述 | 核心功能 |
|------|------|------|----------|
| **collaboration-service** | 8091 | 实时协作 | Yjs CRDT、WebSocket 广播 |
| **geometry-service** | 8092 | 几何处理 | PostGIS、布尔运算、简化 |
| **script-service** | 8095 | 脚本执行 | Python 沙箱、Temporal 编排 |
| **file-service** | 8096 | 文件管理 | MinIO、分片上传、缩略图 |
| **notification-service** | 8097 | 通知服务 | WebSocket、邮件、Webhook |
| **analytics-service** | 8090 | 分析服务 | 事件追踪、ClickHouse |

---

## 🔧 开发指南

### 项目结构

```
FutureBase/
├── backend/                      # 后端代码
│   ├── services/                 # 微服务目录
│   │   ├── user-service/         # Java - 用户服务
│   │   ├── project-service/      # Java - 项目服务
│   │   ├── property-service/     # Java - 属性服务
│   │   ├── version-service/      # Java - 版本服务
│   │   ├── search-service/       # Java - 搜索服务
│   │   ├── collaboration-service/# Go - 协作服务
│   │   ├── geometry-service/     # Go - 几何服务
│   │   ├── script-service/       # Go - 脚本服务
│   │   ├── file-service/         # Go - 文件服务
│   │   ├── notification-service/ # Go - 通知服务
│   │   └── analytics-service/    # Go - 分析服务
│   ├── shared/                   # 共享组件
│   │   ├── proto/                # Protocol Buffers
│   │   ├── models/               # 数据模型
│   │   └── errors/               # 错误码定义
│   ├── deployments/              # 部署配置
│   │   ├── docker/               # Docker Compose
│   │   ├── k8s/                  # Kubernetes 清单
│   │   └── scripts/              # 部署脚本
│   ├── docs/                     # 文档
│   │   ├── ARCHITECTURE.md       # 架构文档
│   │   ├── CODING_STANDARDS.md   # 编码规范
│   │   └── TESTING.md            # 测试指南
│   ├── tests/                    # 测试
│   │   ├── integration/          # 集成测试
│   │   └── e2e/                  # E2E 测试
│   └── Makefile                  # 构建自动化
├── DesignFiles/                  # 设计文档
│   ├── architecture_diagrams.md  # 架构图
│   ├── database_detailed_design_report.md
│   └── collaboration-engine-detailed-design.md
└── README.md                     # 本文件
```

### 开发工作流

```bash
# 1. 创建功能分支
git checkout -b feature/your-feature-name

# 2. 开发并测试
cd backend/services/your-service
# 编写代码...

# 3. 运行测试
make test

# 4. 提交代码
git commit -m "feat: add your feature"

# 5. 推送并创建 PR
git push origin feature/your-feature-name
```

### 代码规范

项目遵循严格的编码规范：

- **Java**: Google Java Style + Spring 最佳实践
- **Go**: Standard Go Project Layout + Effective Go
- **数据库**: PostgreSQL 规范 + 索引优化
- **API**: RESTful 设计 + OpenAPI 3.0

详见 [backend/docs/CODING_STANDARDS.md](backend/docs/CODING_STANDARDS.md)

---

## 🔐 安全架构

### 多层安全防护

```
Layer 5: 应用层安全
├── 代码签名验证
├── 静态分析扫描 (Bandit/Semgrep)
└── 危险函数黑名单

Layer 4: 容器运行时安全 (gVisor)
├── Sentry (用户空间内核)
├── Gofer (文件代理)
└── 系统调用过滤

Layer 3: 系统调用过滤 (seccomp-bpf)
├── 白名单模式
└── 阻止 ~40 个危险 syscall

Layer 2: 资源限制 (cgroups v2)
├── CPU/内存/IO 限制
└── Linux Capabilities

Layer 1: 命名空间隔离
├── PID/Mount/Network/IPC/UTS/User Namespace
```

### 认证授权

- **认证**: JWT Tokens + Refresh Token 机制
- **授权**: RBAC (5 级角色: OWNER/ADMIN/EDITOR/VIEWER/GUEST)
- **MFA**: TOTP 多因素认证
- **加密**: BCrypt 密码加密, TLS 传输加密

---

## 📊 监控与可观测性

### 指标监控 (Prometheus + Grafana)

```bash
# 查看服务健康状态
curl http://localhost:8081/actuator/health
curl http://localhost:8091/health

# 查看 Prometheus 指标
curl http://localhost:8081/actuator/prometheus
```

### 日志聚合

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f user-service
```

### 链路追踪 (Jaeger)

访问 http://localhost:16686 查看分布式追踪。

---

## 🚢 部署

### Docker Compose (开发环境)

```bash
# 启动所有服务
make docker-compose-up

# 查看日志
make docker-compose-logs

# 停止服务
make docker-compose-down
```

### Kubernetes (生产环境)

```bash
# 部署到开发环境
make k8s-deploy ENV=dev

# 查看状态
make k8s-status

# 查看特定服务日志
make k8s-logs SERVICE=user-service

# 端口转发
make k8s-port-forward SERVICE=user-service PORT=8081
```

### 生产环境要求

| 资源 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 16 核 | 32 核+ |
| 内存 | 32 GB | 64 GB+ |
| 磁盘 | 500 GB SSD | 1 TB NVMe |
| 网络 | 1 Gbps | 10 Gbps |

---

## 🧪 测试策略

### 测试金字塔

```
        /\
       /  \
      / E2E \          (浏览器/Postman Newman)
     /--------\
    / 集成测试 \        (服务间交互)
   /------------\
  /   单元测试    \      (JUnit, Go testing)
 /----------------\
```

### 测试命令

```bash
# 单元测试
make test

# 集成测试 (需要运行中的服务)
make test-integration

# E2E 测试 (需要完整环境)
make test-e2e

# 性能测试 (k6)
make test-performance

# 覆盖率报告
make test-coverage
```

---

## 📚 API 文档

### OpenAPI/Swagger

```bash
# 本地启动 API 文档
make api-docs
```

然后访问 http://localhost:8080/swagger-ui.html

### 主要 API 分组

| API | 路径前缀 | 描述 |
|-----|----------|------|
| 用户 API | `/api/v1/users` | 注册、登录、权限管理 |
| 项目 API | `/api/v1/projects` | 项目 CRUD、成员管理 |
| 设计 API | `/api/v1/designs` | 设计文档操作 |
| 几何 API | `/api/v1/geometry` | 几何计算与布尔运算 |
| 脚本 API | `/api/v1/scripts` | 脚本提交与执行 |
| 文件 API | `/api/v1/files` | 上传、下载、管理 |

---

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 贡献流程

1. **Fork** 仓库
2. 创建 **Feature Branch** (`git checkout -b feature/amazing-feature`)
3. **Commit** 你的更改 (`git commit -m 'feat: add amazing feature'`)
4. **Push** 到分支 (`git push origin feature/amazing-feature`)
5. 创建 **Pull Request**

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式 (不影响代码功能)
refactor: 重构
test: 测试相关
chore: 构建/工具相关
```

### 代码审查

- 所有 PR 需要至少 1 个审查者批准
- CI 检查必须通过
- 代码覆盖率不得低于 80%

---

## 📋 路线图

### 2024 Q1-Q2

- [x] 核心微服务架构
- [x] 实时协作引擎 (Yjs)
- [x] 几何计算服务
- [x] 脚本执行沙箱

### 2024 Q3-Q4

- [ ] AI 辅助设计功能
- [ ] BIM 集成支持
- [ ] 移动端适配
- [ ] 插件生态系统

### 2025

- [ ] 多云部署支持
- [ ] 边缘计算节点
- [ ] 智能优化算法
- [ ] 开源社区版

---

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。

---

## 🙏 致谢

- [Spring Boot](https://spring.io/projects/spring-boot) - 优秀的 Java 框架
- [Go](https://golang.org/) - 高效的编程语言
- [Kong](https://konghq.com/) - 强大的 API 网关
- [Temporal](https://temporal.io/) - 可靠的工作流引擎
- [Yjs](https://yjs.dev/) - CRDT 协作框架
- [PostGIS](https://postgis.net/) - 空间数据库扩展

---

## 📞 联系我们

- **项目主页**: https://github.com/samecos/FutureBase
- **文档**: https://docs.futurebase.com
- **问题反馈**: https://github.com/samecos/FutureBase/issues
- **邮件**: maluki@163.com

---

<p align="center">
  <b>Built with ❤️ by the FutureBase Team</b>
</p>
