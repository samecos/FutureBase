# 启动剩余后端服务的脚本
$ErrorActionPreference = "Continue"

$MAVEN_HOME = "C:\apache-maven\apache-maven-3.9.6"
$GO_BIN = "C:\Program Files\Go\bin"

$BACKEND_DIR = "D:\code\FutureBase\backend"
$SERVICES_DIR = "$BACKEND_DIR\services"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "启动剩余后端服务" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Java 服务列表（排除已启动的 user-service 和 project-service）
$JAVA_SERVICES = @(
    @{Name="property-service"; Port=8083; GrpcPort=9094},
    @{Name="version-service"; Port=8084; GrpcPort=9095},
    @{Name="search-service"; Port=8089; GrpcPort=9096}
)

# 启动 Java 服务
foreach ($SERVICE in $JAVA_SERVICES) {
    $SERVICE_NAME = $SERVICE.Name
    $PORT = $SERVICE.Port
    $GRPC_PORT = $SERVICE.GrpcPort

    Write-Host ""
    Write-Host "启动 $SERVICE_NAME (端口: $PORT, gRPC: $GRPC_PORT)..." -ForegroundColor Yellow

    Set-Location -Path "$SERVICES_DIR\$SERVICE_NAME"

    # 检查依赖是否已复制
    if (-not (Test-Path "target\dependency")) {
        Write-Host "  复制依赖..." -ForegroundColor Gray
        & "$MAVEN_HOME\bin\mvn.cmd" dependency:copy-dependencies -DoutputDirectory=target\dependency -q
    }

    # 编译
    & "$MAVEN_HOME\bin\mvn.cmd" compile -q

    if ($LASTEXITCODE -eq 0) {
        # 设置环境变量并启动
        $env:GRPC_PORT = $GRPC_PORT
        $PROCESS = Start-Process -FilePath "java" -ArgumentList "-cp", "target/classes;target/dependency/*", "com.archplatform.$($SERVICE_NAME.Replace('-', '')).$($SERVICE_NAME.Replace('-service', '').Replace('-', '').Substring(0,1).ToUpper() + $SERVICE_NAME.Replace('-service', '').Replace('-', '').Substring(1))ServiceApplication" -PassThru -WindowStyle Hidden
        Write-Host "  ✓ $SERVICE_NAME 启动 (PID: $($PROCESS.Id))" -ForegroundColor Green
    } else {
        Write-Host "  ✗ $SERVICE_NAME 编译失败" -ForegroundColor Red
    }
}

# Go 服务列表
$GO_SERVICES = @(
    @{Name="collaboration-service"; Port=8085},
    @{Name="geometry-service"; Port=8086},
    @{Name="file-service"; Port=8088},
    @{Name="notification-service"; Port=8090}
)

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "构建并启动 Go 服务" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

foreach ($SERVICE in $GO_SERVICES) {
    $SERVICE_NAME = $SERVICE.Name
    $PORT = $SERVICE.Port

    Write-Host ""
    Write-Host "构建并启动 $SERVICE_NAME (端口: $PORT)..." -ForegroundColor Yellow

    Set-Location -Path "$SERVICES_DIR\$SERVICE_NAME"

    # 检查是否有 cmd/server/main.go
    if (Test-Path "cmd\server\main.go") {
        # 创建 bin 目录
        New-Item -ItemType Directory -Force -Path "bin" | Out-Null

        # 构建
        & "$GO_BIN\go.exe" build -o "bin\$SERVICE_NAME.exe" ./cmd/server 2>&1 | Out-Null

        if ($LASTEXITCODE -eq 0) {
            # 启动
            $PROCESS = Start-Process -FilePath "bin\$SERVICE_NAME.exe" -PassThru -WindowStyle Hidden
            Write-Host "  ✓ $SERVICE_NAME 启动 (PID: $($PROCESS.Id))" -ForegroundColor Green
        } else {
            Write-Host "  ✗ $SERVICE_NAME 构建失败" -ForegroundColor Red
        }
    } else {
        Write-Host "  ⚠ $SERVICE_NAME 没有 main.go，跳过" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "所有服务启动完成" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

Set-Location -Path $BACKEND_DIR
