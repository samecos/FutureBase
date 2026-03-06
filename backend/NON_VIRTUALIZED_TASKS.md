# 无虚拟化环境下的开发任务

> 适用于 Hyper-V 等无法开启嵌套虚拟化的环境

## ✅ 可以执行的任务

### 1. 代码构建 (Build)
```bash
# Java 服务本地构建
make build-java
# 或
cd services/user-service && ./mvnw clean package -DskipTests

# Go 服务本地构建  
make build-go
# 或
cd services/collaboration-service && go build ./cmd/server
```

### 2. 单元测试 (Testing)
```bash
# 运行所有测试
make test

# Java 测试
make test-java

# Go 测试
make test-go
```

### 3. 代码质量 (Code Quality)
```bash
# 代码格式化
make fmt-go

# 静态分析 (需要安装 golangci-lint)
make lint-go

# Java 代码规范检查
make lint-java
```

### 4. 代码生成 (Code Generation)
```bash
# 从 protobuf 生成代码 (需要安装 protoc)
make proto-generate

# 生成 OpenAPI 客户端
make generate-clients
```

### 5. 项目统计
```bash
make stats        # 查看代码统计
make clean        # 清理构建产物
```

---

## 🔧 建议执行的任务清单

### 优先级 1: 核心代码完善
- [ ] **添加缺失的单元测试** - Java 服务需要更多测试覆盖
- [ ] **完善 Go 服务错误处理** - 添加更详细的错误码和日志
- [ ] **添加数据库 Migration 脚本** - Flyway/Liquibase 脚本
- [ ] **添加 API 校验注解** - Jakarta Validation

### 优先级 2: 开发体验
- [ ] **配置 IDE 代码格式化** - .editorconfig, checkstyle.xml
- [ ] **添加 Pre-commit Hooks** - 代码提交前自动检查
- [ ] **完善错误码定义** - 统一的错误码体系
- [ ] **添加多语言支持** - i18n 资源文件

### 优先级 3: 文档和工具
- [ ] **编写 API 使用示例** - Postman 集合
- [ ] **添加数据库 ER 图** - dbdiagram.io 或 PlantUML
- [ ] **完善操作手册** - 故障排查指南
- [ ] **添加性能测试脚本** - JMeter/K6 配置

### 优先级 4: 安全加固
- [ ] **添加 OWASP 依赖检查** - 安全漏洞扫描
- [ ] **配置 Spring Security** - 更细粒度的权限控制
- [ ] **添加输入验证** - SQL 注入、XSS 防护
- [ ] **敏感信息脱敏** - 日志中的密码、Token 处理

---

## 🚀 立即可执行的操作

### 操作 A: 验证所有服务能否编译
```powershell
# 检查所有 Java 服务
$services = @('user-service','project-service','property-service','version-service','search-service')
foreach ($svc in $services) {
    Write-Host "Building $svc..." -ForegroundColor Cyan
    Set-Location services/$svc
    ./mvnw compile -q
    Set-Location ../..
}

# 检查所有 Go 服务
$goServices = @('collaboration-service','geometry-service','script-service','file-service','notification-service','analytics-service')
foreach ($svc in $goServices) {
    Write-Host "Building $svc..." -ForegroundColor Cyan
    Set-Location services/$svc
    go build -o bin/$svc.exe ./cmd/server
    Set-Location ../..
}
```

### 操作 B: 生成测试数据脚本
```bash
# 创建测试数据生成脚本
make db-seed
```

### 操作 C: 验证配置文件
```bash
# 验证所有 YAML 配置
make validate-config
```

---

## 📝 无需虚拟化的代码开发

### 可以编写的新功能

1. **扩展 Protocol Buffers 定义**
   - 添加更多 gRPC 服务定义
   - 扩展事件消息类型

2. **完善 Domain 层**
   - 添加领域事件
   - 完善值对象

3. **添加 Query 对象**
   - CQRS 查询优化
   - 搜索条件对象

4. **编写 Repository 实现**
   - 自定义查询方法
   - 批量操作优化

5. **添加 Service 层逻辑**
   - 业务规则校验
   - 事务管理

---

## 🎯 推荐当前执行的任务

我建议按以下顺序执行（无需虚拟化）：

1. **立即执行**: 验证所有服务能否本地编译 ✅
2. **添加测试**: 为关键 Service 类编写单元测试
3. **完善文档**: 添加 JavaDoc/GoDoc 注释
4. **代码检查**: 运行静态分析，修复潜在问题
5. **配置优化**: 添加 IDE 配置和代码规范

需要我帮你开始哪个任务？
