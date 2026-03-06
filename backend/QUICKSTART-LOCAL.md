# FutureBase 后端本地启动指南

## 环境准备

已安装的工具：
- ✅ Java 21
- ✅ Maven 3.9.6 (`C:\apache-maven\apache-maven-3.9.6`)
- ✅ Go 1.26 (`C:\Program Files\Go`)
- ✅ Docker (用于基础设施服务)

## 快速启动

### 1. 启动基础设施服务

由于 Docker Hub 连接问题，需要手动拉取镜像或使用本地镜像：

```powershell
cd D:\code\FutureBase\backend\deployments\docker

# 启动基础设施（如果镜像已存在）
docker-compose -f docker-compose.yml up -d postgres redis kafka elasticsearch minio temporal zookeeper nats
```

### 2. 构建并启动后端服务

使用提供的 PowerShell 脚本：

```powershell
cd D:\code\FutureBase\backend
.\scripts\start-backend.ps1
```

或者手动构建和启动：

#### 构建 Java 服务

```powershell
# 设置 Maven 路径
$MAVEN_HOME = "C:\apache-maven\apache-maven-3.9.6"

# 构建 user-service
cd D:\code\FutureBase\backend\services\user-service
& "$MAVEN_HOME\bin\mvn.cmd" clean compile -DskipTests
& "$MAVEN_HOME\bin\mvn.cmd" jar:jar

# 启动 user-service
java -jar target\user-service-1.0.0.jar
```

#### 构建 Go 服务

```powershell
# 设置 Go 路径
$GO_BIN = "C:\Program Files\Go\bin"

# 构建 collaboration-service
cd D:\code\FutureBase\backend\services\collaboration-service
& "$GO_BIN\go.exe" build -o bin\collaboration-service.exe ./cmd/server

# 启动 collaboration-service
.\bin\collaboration-service.exe
```

## 服务端口

| 服务 | 端口 | 类型 |
|------|------|------|
| user-service | 8081 | Java |
| project-service | 8082 | Java |
| property-service | 8083 | Java |
| version-service | 8084 | Java |
| search-service | 8089 | Java |
| collaboration-service | 8085 | Go |
| geometry-service | 8086 | Go |
| script-service | 8087 | Go |
| file-service | 8088 | Go |
| notification-service | 8090 | Go |
| analytics-service | 8091 | Go |

## 基础设施端口

| 服务 | 端口 |
|------|------|
| PostgreSQL | 5432 |
| Redis | 6379 |
| Kafka | 9092 |
| Elasticsearch | 9200 |
| MinIO | 9000 / 9001 |
| Temporal | 7233 |
| NATS | 4222 |

## 已完成的修复

1. ✅ 安装了 Maven 3.9.6
2. ✅ 安装了 Go 1.26
3. ✅ 修复了 protobuf 文件问题：
   - `geometry.proto`: 添加了 `google/protobuf/field_mask.proto` 导入
   - `user.proto`: 添加了 `google/protobuf/struct.proto` 导入，修复了 `PaginationRequest` 引用
4. ✅ 修复了 `RegisterRequest.java`: 添加了 `java.util.UUID` 导入
5. ✅ 为所有 Java 服务添加了 `protobuf-java` 和 `javax.annotation-api` 依赖
6. ✅ 成功构建 `user-service` 和 `project-service`

## 已知问题

1. Docker Hub 连接问题 - 无法拉取新镜像
2. 部分 Java 服务测试代码有编译错误（使用 `-DskipTests` 跳过）
3. 部分 Go 服务可能需要额外的依赖配置

## 下一步

1. 确保基础设施容器正常运行
2. 运行 `start-backend.ps1` 脚本启动所有服务
3. 访问 API 文档（如果有）测试服务
