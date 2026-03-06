# 代码规范快速开始指南

> 5 分钟上手 ArchPlatform 代码规范

## 🚀 快速设置

### 1. 安装必要工具

```bash
# 运行设置脚本（安装 pre-commit、Go 工具等）
make setup-hooks

# 或手动安装
cd backend

# Python 工具
pip install pre-commit

# Go 工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
```

### 2. 配置 Git Hooks

```bash
# 安装 pre-commit hooks
pre-commit install

# 测试 hooks
pre-commit run --all-files
```

---

## 📝 常用命令

### 代码格式化

```bash
# 格式化所有代码
make fmt

# 仅格式化 Java
make fmt-java

# 仅格式化 Go
make fmt-go

# 检查格式（CI 使用）
make check-fmt
```

### 代码检查

```bash
# 运行所有检查
make lint

# Java 代码规范检查
make lint-java

# Go 代码检查
make lint-go
```

### 完整检查流程

```bash
# 提交前的完整检查
make check-fmt  # 检查格式
make lint       # 检查规范
make test       # 运行测试
```

---

## 🎯 IDE 配置

### IntelliJ IDEA

1. **安装插件**
   - `google-java-format` - Google Java 格式化
   - `CheckStyle-IDEA` - 代码规范检查

2. **导入代码样式**
   ```
   Settings → Editor → Code Style → Java → ⚙️ → Import Scheme
   → 选择: https://raw.githubusercontent.com/google/styleguide/gh-pages/intellij-java-google-style.xml
   ```

3. **配置 Checkstyle**
   ```
   Settings → Tools → Checkstyle
   → Configuration File → + 
   → 选择: backend/checkstyle.xml
   ```

4. **保存自动格式化**
   ```
   Settings → Tools → Actions on Save
   ✓ Reformat code
   ✓ Optimize imports
   ✓ Rearrange code
   ```

### VS Code

**settings.json:**
```json
{
  "editor.formatOnSave": true,
  "java.format.settings.url": "https://raw.githubusercontent.com/google/styleguide/gh-pages/eclipse-java-google-style.xml",
  "java.format.settings.profile": "GoogleStyle",
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "editor.codeActionsOnSave": {
    "source.organizeImports": true
  }
}
```

**推荐插件:**
- Extension Pack for Java
- Go
- Prettier
- Checkstyle for Java

---

## ✅ 提交前检查清单

提交代码前，请运行：

```bash
make check-fmt && make lint && make test
```

或手动检查：

- [ ] `make fmt` - 代码已格式化
- [ ] `make lint` - 规范检查通过
- [ ] `make test` - 测试通过
- [ ] 无敏感信息（密码、密钥）
- [ ] 提交信息符合规范 (`feat:`, `fix:`, `docs:` 等)

---

## 🔧 跳过检查（紧急情况）

```bash
# 跳过 pre-commit hooks
git commit --no-verify -m "your message"

# Maven 跳过代码检查
mvn clean package -Dspotless.check.skip=true -Dcheckstyle.skip=true

# Go 跳过 lint
go build -tags skip_lint
```

---

## 🆘 常见问题

### Q: pre-commit 安装失败

```bash
# Windows 可能需要管理员权限
pip install --user pre-commit

# 然后添加到 PATH
# C:\Users\<username>\AppData\Roaming\Python\Python3x\Scripts
```

### Q: Spotless 格式化失败

```bash
# 确保在项目目录下运行
cd services/user-service
mvn spotless:apply

# 或从 backend 目录
make fmt-java
```

### Q: golangci-lint 找不到

```bash
# 确认 GOPATH/bin 在 PATH 中
export PATH=$PATH:$(go env GOPATH)/bin

# Windows (PowerShell)
$env:PATH += ";$(go env GOPATH)\bin"
```

### Q: Checkstyle 报错太多

1. 先运行格式化：`make fmt-java`
2. 查看具体错误：`mvn checkstyle:checkstyle`
3. 报告位置：`target/checkstyle-result.xml`

---

## 📊 代码质量指标

项目目标：

| 指标 | 目标 | 当前 |
|------|------|------|
| 测试覆盖率 | > 70% | 待统计 |
| Checkstyle 错误 | 0 | 待统计 |
| Go Lint 问题 | 0 | 待统计 |
| 代码重复率 | < 5% | 待统计 |

---

## 📚 详细文档

- [完整代码规范](CODING_STANDARDS.md)
- [Java 规范](https://google.github.io/styleguide/javaguide.html)
- [Go 规范](https://golang.org/doc/effective_go)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

## 💡 最佳实践

1. **频繁提交**：小步快跑，每次提交一个逻辑单元
2. **先格式化后提交**：养成 `make fmt` 习惯
3. **本地检查**：提交前运行 `make lint`
4. **及时修复**：发现问题立即修复，不要堆积

---

有问题？查看 [TROUBLESHOOTING.md](TROUBLESHOOTING.md) 或联系团队。
