# ArchPlatform 代码规范指南

> 本项目的代码规范标准，所有贡献者必须遵守。

## 📋 目录

- [通用规范](#通用规范)
- [Java 规范](#java-规范)
- [Go 规范](#go-规范)
- [Protocol Buffers 规范](#protocol-buffers-规范)
- [数据库规范](#数据库规范)
- [Git 提交规范](#git-提交规范)

---

## 通用规范

### 基本准则

1. **可读性优先**：代码是写给人看的，顺便给机器执行
2. **自解释代码**：通过清晰的命名和结构表达意图，避免过度注释
3. **单一职责**：每个函数/类只做一件事
4. **DRY 原则**：Don't Repeat Yourself

### 文件格式

- **编码**：UTF-8 (无 BOM)
- **换行**：LF (`\n`)
- **缩进**：
  - Java：4 空格
  - Go：Tab
  - YAML/JSON：2 空格
- **行尾**：无尾随空格
- **文件结尾**：保留一个空行

---

## Java 规范

### 代码格式化

我们使用 **Google Java Format** 进行代码格式化。

#### 格式化命令

```bash
# 格式化单个服务
cd services/user-service
mvn spotless:apply

# 检查格式（CI 使用）
mvn spotless:check

# 跳过格式化（快速构建）
mvn clean package -Dspotless.check.skip=true
```

#### 代码样式

```java
// ✅ Good：遵循 Google Java Format
public class UserService {
  private static final int MAX_RETRY = 3;
  
  private final UserRepository userRepository;
  
  @Autowired
  public UserService(UserRepository userRepository) {
    this.userRepository = userRepository;
  }
  
  public User createUser(CreateUserRequest request) {
    validateRequest(request);
    
    User user = User.builder()
        .username(request.getUsername())
        .email(request.getEmail())
        .build();
    
    return userRepository.save(user);
  }
}

// ❌ Bad：不符合规范
public class user_service {
    private final static int maxRetry=3;
    @Autowired
    private UserRepository userRepository;
    
    public User createUser(CreateUserRequest request){
        validateRequest(request);
        User user=User.builder().username(request.getUsername()).email(request.getEmail()).build();
        return userRepository.save(user);
    }
}
```

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 类名 | PascalCase | `UserService`, `ProjectController` |
| 方法名 | camelCase | `createUser()`, `getProjectById()` |
| 变量名 | camelCase | `userName`, `projectList` |
| 常量 | UPPER_SNAKE_CASE | `MAX_RETRY`, `DEFAULT_PAGE_SIZE` |
| 包名 | 全小写 | `com.archplatform.user.service` |
| 数据库字段 | snake_case | `created_at`, `user_id` |

### 代码结构

```java
package com.archplatform.user.service;

// 1. 导入语句（按顺序：java, javax, org, com, 其他）
import java.util.Optional;
import java.util.UUID;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.archplatform.user.entity.User;
import com.archplatform.user.repository.UserRepository;

/**
 * 2. Javadoc 类注释
 * 用户服务，处理用户相关的业务逻辑。
 *
 * @author ArchPlatform Team
 * @since 1.0.0
 */
@Service
@Transactional(readOnly = true)
public class UserService {

  // 3. 静态常量
  private static final int DEFAULT_PAGE_SIZE = 20;
  private static final String USER_CACHE_PREFIX = "user:";

  // 4. 实例变量
  private final UserRepository userRepository;
  private final CacheManager cacheManager;

  // 5. 构造函数
  public UserService(UserRepository userRepository, CacheManager cacheManager) {
    this.userRepository = userRepository;
    this.cacheManager = cacheManager;
  }

  // 6. 公共方法
  @Transactional
  public User createUser(CreateUserRequest request) {
    // 实现
  }

  // 7. 私有方法
  private void validateRequest(CreateUserRequest request) {
    // 实现
  }
}
```

### Checkstyle 检查

```bash
# 运行 Checkstyle 检查
mvn checkstyle:check

# 生成 Checkstyle 报告
mvn checkstyle:checkstyle
# 报告位置: target/checkstyle-result.xml
```

---

## Go 规范

### 代码格式化

Go 使用内置的 `gofmt` 工具。

```bash
# 格式化代码
go fmt ./...

# 自动导入管理
goimports -w .

# 完整 lint 检查
golangci-lint run

# 自动修复问题
golangci-lint run --fix
```

### 代码样式

```go
// ✅ Good
type UserService struct {
    repo   UserRepository
    cache  CacheManager
}

func NewUserService(repo UserRepository, cache CacheManager) *UserService {
    return &UserService{
        repo:  repo,
        cache: cache,
    }
}

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    if err := s.validateRequest(req); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }

    user := &User{
        ID:       uuid.New(),
        Username: req.Username,
        Email:    req.Email,
    }

    return s.repo.Save(ctx, user)
}

// ❌ Bad
func (this *UserService) CreateUser(req CreateUserRequest)(*User,error){
    user:=&User{ID:uuid.New(),Username:req.Username}
    return s.repo.Save(context.Background(),user)
}
```

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 接口 | 名词/形容词 | `Reader`, `Handler`, `Stringer` |
| 结构体 | PascalCase | `UserService`, `ProjectConfig` |
| 函数 | camelCase/PascalCase | `createUser()`, `NewUserService()` |
| 变量 | camelCase | `userName`, `projectList` |
| 常量 | camelCase/PascalCase | `maxRetry`, `DefaultPageSize` |
| 包名 | 全小写，单数 | `user`, `project`, `geometry` |
| 错误变量 | Err 前缀 | `ErrNotFound`, `ErrInvalidInput` |
| 私有函数 | camelCase | `validateInput()`, `parseToken()` |

### 项目结构

```
service-name/
├── cmd/
│   └── server/          # 应用程序入口
│       └── main.go
├── internal/            # 私有代码
│   ├── config/          # 配置
│   ├── handler/         # HTTP 处理器
│   ├── models/          # 数据模型
│   └── storage/         # 数据存储
├── pkg/                 # 公共库
│   ├── errors/          # 错误定义
│   └── utils/           # 工具函数
├── api/                 # API 定义
├── go.mod
├── go.sum
└── README.md
```

### 错误处理

```go
// ✅ Good：包装错误提供上下文
if err := db.Query(ctx, query); err != nil {
    return fmt.Errorf("failed to query user %s: %w", userID, err)
}

// ✅ Good：定义 sentinel 错误
var ErrUserNotFound = errors.New("user not found")

if errors.Is(err, ErrUserNotFound) {
    return nil, ErrUserNotFound
}

// ❌ Bad：丢失原始错误信息
if err != nil {
    return errors.New("failed")
}
```

---

## Protocol Buffers 规范

### 文件组织

```
shared/proto/
├── common.proto         # 通用类型
├── user.proto          # 用户服务
├── project.proto       # 项目服务
└── collaboration.proto # 协作服务
```

### 命名规范

```protobuf
// ✅ Good
syntax = "proto3";

package archplatform.user.v1;

option go_package = "github.com/archplatform/backend/shared/proto/user;userpb";
option java_package = "com.archplatform.user.proto";
option java_multiple_files = true;

// 消息名：PascalCase
message CreateUserRequest {
  // 字段名：snake_case
  string user_id = 1;
  string email = 2;
  
  // 枚举名：TYPE_FORMAT
  enum UserStatus {
    USER_STATUS_UNSPECIFIED = 0;
    USER_STATUS_ACTIVE = 1;
    USER_STATUS_INACTIVE = 2;
  }
  
  UserStatus status = 3;
}

// RPC 名：PascalCase
service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (User);
}
```

---

## 数据库规范

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 表名 | snake_case, 复数 | `users`, `project_members` |
| 字段 | snake_case | `created_at`, `user_id` |
| 主键 | id 或 {table}_id | `id`, `project_id` |
| 外键 | {table}_id | `user_id`, `project_id` |
| 索引 | idx_{table}_{field} | `idx_users_email` |
| 唯一索引 | uk_{table}_{field} | `uk_users_username` |

### 字段顺序

```sql
CREATE TABLE users (
    -- 1. 主键
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- 2. 业务字段
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    
    -- 3. 状态字段
    status VARCHAR(20) DEFAULT 'active',
    
    -- 4. 审计字段
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    
    -- 5. 软删除
    deleted_at TIMESTAMP,
    deleted_by UUID
);
```

### 必需字段

所有表必须包含：
- `id` - 主键
- `created_at` - 创建时间
- `updated_at` - 更新时间
- `deleted_at` - 软删除标记

---

## Git 提交规范

### 提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建/工具相关 |

### 示例

```bash
# 功能提交
git commit -m "feat(user): add MFA support with TOTP"

# Bug 修复
git commit -m "fix(project): resolve concurrent lock issue

When multiple users try to acquire the same lock simultaneously,
the previous implementation could cause a race condition.

Closes #123"

# 文档更新
git commit -m "docs(api): update OpenAPI spec for file upload"
```

### 分支命名

```
feature/user-mfa           # 新功能
fix/project-lock-race      # Bug 修复
refactor/geometry-engine   # 重构
docs/api-examples          # 文档
```

---

## 🔧 工具配置

### IDE 配置

#### IntelliJ IDEA

1. 安装插件：
   - Google Java Format
   - CheckStyle-IDEA
   - Save Actions

2. 配置代码样式：
   ```
   Editor → Code Style → Java → Scheme → Import Scheme → GoogleStyle
   ```

3. 配置保存自动格式化：
   ```
   Tools → Actions on Save → Reformat code ✅
   ```

#### VS Code

1. 安装插件：
   - Extension Pack for Java
   - Go
   - Prettier

2. 配置 settings.json：
   ```json
   {
     "editor.formatOnSave": true,
     "java.format.settings.url": "https://raw.githubusercontent.com/google/styleguide/gh-pages/eclipse-java-google-style.xml",
     "go.formatTool": "goimports",
     "go.lintTool": "golangci-lint"
   }
   ```

---

## ✅ 提交前检查清单

提交代码前，请确认：

- [ ] 代码已格式化（`mvn spotless:apply` 或 `go fmt`）
- [ ] 代码规范检查通过（`mvn checkstyle:check` 或 `golangci-lint run`）
- [ ] 单元测试通过（`mvn test` 或 `go test ./...`）
- [ ] 无敏感信息泄露（密码、密钥等）
- [ ] 提交信息符合规范

---

## 📚 参考资源

- [Google Java Style Guide](https://google.github.io/styleguide/javaguide.html)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Conventional Commits](https://www.conventionalcommits.org/)
