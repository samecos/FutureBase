# FutureBase 后端服务启动脚本
# Start Backend Services Script

$ErrorActionPreference = "Stop"

# 颜色定义
$GREEN = "Green"
$RED = "Red"
$YELLOW = "Yellow"
$CYAN = "Cyan"

# 设置路径
$MAVEN_HOME = "C:\apache-maven\apache-maven-3.9.6"
$GO_BIN = "C:\Program Files\Go\bin"
$env:PATH = "$MAVEN_HOME\bin;$GO_BIN;$env:PATH"

$BACKEND_DIR = "D:\code\FutureBase\backend"
$SERVICES_DIR = "$BACKEND_DIR\services"

Write-Host "========================================" -ForegroundColor $CYAN
Write-Host "FutureBase 后端服务启动脚本" -ForegroundColor $CYAN
Write-Host "========================================" -ForegroundColor $CYAN

# 检查 Java
Write-Host ""
Write-Host "检查 Java 环境..." -ForegroundColor $YELLOW
try {
    $JAVA_VERSION = java -version 2>&1 | Select-String "version" | ForEach-Object { $_.ToString() }
    Write-Host "✓ Java: $JAVA_VERSION" -ForegroundColor $GREEN
} catch {
    Write-Host "✗ Java 未安装或不在 PATH 中" -ForegroundColor $RED
    exit 1
}

# 检查 Maven
Write-Host ""
Write-Host "检查 Maven 环境..." -ForegroundColor $YELLOW
try {
    $MVN_VERSION = & "$MAVEN_HOME\bin\mvn.cmd" -version 2>&1 | Select-String "Apache Maven" | ForEach-Object { $_.ToString().Split()[2] }
    Write-Host "✓ Maven: $MVN_VERSION" -ForegroundColor $GREEN
} catch {
    Write-Host "✗ Maven 未找到: $MAVEN_HOME" -ForegroundColor $RED
    exit 1
}

# 检查 Go
Write-Host ""
Write-Host "检查 Go 环境..." -ForegroundColor $YELLOW
try {
    $GO_VERSION = & "$GO_BIN\go.exe" version 2>&1
    Write-Host "✓ Go: $GO_VERSION" -ForegroundColor $GREEN
} catch {
    Write-Host "✗ Go 未找到: $GO_BIN" -ForegroundColor $RED
    exit 1
}

# Java 服务列表
$JAVA_SERVICES = @(
    @{Name="user-service"; Port=8081; Description="用户服务 - JWT认证、MFA、RBAC"},
    @{Name="project-service"; Port=8082; Description="项目服务 - 项目管理、成员角色"},
    @{Name="property-service"; Port=8083; Description="属性服务 - MVEL规则、单位转换"},
    @{Name="version-service"; Port=8084; Description="版本服务 - Git-like分支、合并冲突"},
    @{Name="search-service"; Port=8089; Description="搜索服务 - Elasticsearch全文搜索"}
)

# Go 服务列表
$GO_SERVICES = @(
    @{Name="collaboration-service"; Port=8085; Description="协作服务 - Yjs CRDT、WebSocket"},
    @{Name="geometry-service"; Port=8086; Description="几何服务 - PostGIS、布尔运算"},
    @{Name="script-service"; Port=8087; Description="脚本服务 - Python沙箱、gVisor"},
    @{Name="file-service"; Port=8088; Description="文件服务 - MinIO、分片上传"},
    @{Name="notification-service"; Port=8090; Description="通知服务 - WebSocket、邮件、Webhooks"},
    @{Name="analytics-service"; Port=8091; Description="分析服务 - 事件追踪、ClickHouse"}
)

# 构建 Java 服务
Write-Host ""
Write-Host "========================================" -ForegroundColor $CYAN
Write-Host "构建 Java 服务" -ForegroundColor $CYAN
Write-Host "========================================" -ForegroundColor $CYAN

$JAVA_FAILED = @()
foreach ($SERVICE in $JAVA_SERVICES) {
    $SERVICE_NAME = $SERVICE.Name
    Write-Host ""
    Write-Host "构建 $SERVICE_NAME..." -ForegroundColor $YELLOW

    Set-Location -Path "$SERVICES_DIR\$SERVICE_NAME"

    # 编译
    & "$MAVEN_HOME\bin\mvn.cmd" clean compile -DskipTests -q 2>&1 | Out-Null

    if ($LASTEXITCODE -eq 0) {
        # 打包
        & "$MAVEN_HOME\bin\mvn.cmd" jar:jar -q 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  ✓ $SERVICE_NAME 构建成功" -ForegroundColor $GREEN
        } else {
            Write-Host "  ✗ $SERVICE_NAME 打包失败" -ForegroundColor $RED
            $JAVA_FAILED += $SERVICE_NAME
        }
    } else {
        Write-Host "  ✗ $SERVICE_NAME 编译失败" -ForegroundColor $RED
        $JAVA_FAILED += $SERVICE_NAME
    }
}

# 构建 Go 服务
Write-Host ""
Write-Host "========================================" -ForegroundColor $CYAN
Write-Host "构建 Go 服务" -ForegroundColor $CYAN
Write-Host "========================================" -ForegroundColor $CYAN

$GO_FAILED = @()
foreach ($SERVICE in $GO_SERVICES) {
    $SERVICE_NAME = $SERVICE.Name
    Write-Host ""
    Write-Host "构建 $SERVICE_NAME..." -ForegroundColor $YELLOW

    Set-Location -Path "$SERVICES_DIR\$SERVICE_NAME"

    # 检查是否有 cmd/server/main.go
    if (Test-Path "cmd\server\main.go") {
        # 创建 bin 目录
        New-Item -ItemType Directory -Force -Path "bin" | Out-Null

        # 构建
        & "$GO_BIN\go.exe" build -o "bin\$SERVICE_NAME.exe" ./cmd/server 2>&1 | Out-Null

        if ($LASTEXITCODE -eq 0) {
            Write-Host "  ✓ $SERVICE_NAME 构建成功" -ForegroundColor $GREEN
        } else {
            Write-Host "  ✗ $SERVICE_NAME 构建失败" -ForegroundColor $RED
            $GO_FAILED += $SERVICE_NAME
        }
    } else {
        Write-Host "  ⚠ $SERVICE_NAME 没有 main.go，跳过" -ForegroundColor $YELLOW
    }
}

# 启动服务
Write-Host ""
Write-Host "========================================" -ForegroundColor $CYAN
Write-Host "启动服务" -ForegroundColor $CYAN
Write-Host "========================================" -ForegroundColor $CYAN

Write-Host ""
Write-Host "注意: 请确保以下基础设施服务已启动:" -ForegroundColor $YELLOW
Write-Host "  - PostgreSQL (端口: 5432)" -ForegroundColor $YELLOW
Write-Host "  - Redis (端口: 6379)" -ForegroundColor $YELLOW
Write-Host "  - Kafka (端口: 9092)" -ForegroundColor $YELLOW
Write-Host "  - Elasticsearch (端口: 9200)" -ForegroundColor $YELLOW
Write-Host "  - MinIO (端口: 9000)" -ForegroundColor $YELLOW
Write-Host "  - Temporal (端口: 7233)" -ForegroundColor $YELLOW
Write-Host ""
Write-Host "可以使用以下命令启动基础设施:" -ForegroundColor $CYAN
Write-Host "  cd D:\code\FutureBase\backend\deployments\docker" -ForegroundColor $CYAN
Write-Host "  docker-compose -f docker-compose.yml up -d postgres redis kafka elasticsearch minio temporal" -ForegroundColor $CYAN
Write-Host ""

# 启动 Java 服务
Write-Host "启动 Java 服务..." -ForegroundColor $YELLOW
$JAVA_PROCESSES = @()
foreach ($SERVICE in $JAVA_SERVICES) {
    $SERVICE_NAME = $SERVICE.Name
    $PORT = $SERVICE.Port

    if ($JAVA_FAILED -contains $SERVICE_NAME) {
        Write-Host "  ⚠ 跳过 $SERVICE_NAME (构建失败)" -ForegroundColor $YELLOW
        continue
    }

    $JAR_PATH = "$SERVICES_DIR\$SERVICE_NAME\target\$SERVICE_NAME-1.0.0.jar"
    if (Test-Path $JAR_PATH) {
        Write-Host "  启动 $SERVICE_NAME (端口: $PORT)..." -ForegroundColor $GREEN
        $PROCESS = Start-Process -FilePath "java" -ArgumentList "-jar", "`"$JAR_PATH`"" -WorkingDirectory "$SERVICES_DIR\$SERVICE_NAME" -PassThru -WindowStyle Hidden
        $JAVA_PROCESSES += @{Name=$SERVICE_NAME; Process=$PROCESS; Port=$PORT}
    } else {
        Write-Host "  ✗ $SERVICE_NAME JAR 文件不存在" -ForegroundColor $RED
    }
}

# 启动 Go 服务
Write-Host ""
Write-Host "启动 Go 服务..." -ForegroundColor $YELLOW
$GO_PROCESSES = @()
foreach ($SERVICE in $GO_SERVICES) {
    $SERVICE_NAME = $SERVICE.Name
    $PORT = $SERVICE.Port

    if ($GO_FAILED -contains $SERVICE_NAME) {
        Write-Host "  ⚠ 跳过 $SERVICE_NAME (构建失败)" -ForegroundColor $YELLOW
        continue
    }

    $EXE_PATH = "$SERVICES_DIR\$SERVICE_NAME\bin\$SERVICE_NAME.exe"
    if (Test-Path $EXE_PATH) {
        Write-Host "  启动 $SERVICE_NAME (端口: $PORT)..." -ForegroundColor $GREEN
        $PROCESS = Start-Process -FilePath $EXE_PATH -WorkingDirectory "$SERVICES_DIR\$SERVICE_NAME" -PassThru -WindowStyle Hidden
        $GO_PROCESSES += @{Name=$SERVICE_NAME; Process=$PROCESS; Port=$PORT}
    } else {
        Write-Host "  ⚠ $SERVICE_NAME 可执行文件不存在，跳过" -ForegroundColor $YELLOW
    }
}

# 保存进程信息到文件
$ALL_PROCESSES = $JAVA_PROCESSES + $GO_PROCESSES
$PROCESS_INFO = $ALL_PROCESSES | ForEach-Object {
    "$($_.Name):$($_.Process.Id):$($_.Port)"
}
$PROCESS_INFO | Out-File -FilePath "$BACKEND_DIR\running-services.txt" -Encoding UTF8

Write-Host ""
Write-Host "========================================" -ForegroundColor $CYAN
Write-Host "服务启动完成" -ForegroundColor $CYAN
Write-Host "========================================" -ForegroundColor $CYAN
Write-Host ""
Write-Host "已启动的服务信息已保存到: $BACKEND_DIR\running-services.txt" -ForegroundColor $GREEN
Write-Host ""
Write-Host "服务列表:" -ForegroundColor $CYAN
foreach ($P in $ALL_PROCESSES) {
    Write-Host "  - $($P.Name) (PID: $($P.Process.Id), 端口: $($P.Port))" -ForegroundColor $GREEN
}
Write-Host ""
Write-Host "停止所有服务命令:" -ForegroundColor $YELLOW
Write-Host "  Stop-Process -Id <PID>" -ForegroundColor $CYAN
Write-Host ""
Write-Host "或使用:" -ForegroundColor $YELLOW
Write-Host "  Get-Content $BACKEND_DIR\running-services.txt | ForEach-Object { `" -ForegroundColor $CYAN
Write-Host "    $parts = $_.Split(':')" -ForegroundColor $CYAN
Write-Host "    Stop-Process -Id $parts[1] -Force -ErrorAction SilentlyContinue" -ForegroundColor $CYAN
Write-Host "  }" -ForegroundColor $CYAN
Write-Host ""

Set-Location -Path $BACKEND_DIR
