# 构建所有 Java 服务的 PowerShell 脚本
# Build All Java Services Script

$ErrorActionPreference = "Continue"

# 设置 Maven 路径
$MAVEN_HOME = "C:\apache-maven\apache-maven-3.9.6"
$env:PATH = "$MAVEN_HOME\bin;$env:PATH"

# Java 服务列表
$JAVA_SERVICES = @(
    "user-service",
    "project-service",
    "property-service",
    "version-service",
    "search-service"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Building Java Services" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$FAILED_SERVICES = @()

foreach ($SERVICE in $JAVA_SERVICES) {
    Write-Host ""
    Write-Host "Building $SERVICE..." -ForegroundColor Green
    Write-Host "----------------------------------------"

    Set-Location -Path "D:\code\FutureBase\backend\services\$SERVICE"

    # 清理并编译
    & "$MAVEN_HOME\bin\mvn.cmd" clean compile -DskipTests -q

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $SERVICE compiled successfully" -ForegroundColor Green

        # 打包 JAR
        & "$MAVEN_HOME\bin\mvn.cmd" jar:jar -q

        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ $SERVICE packaged successfully" -ForegroundColor Green
        } else {
            Write-Host "✗ $SERVICE packaging failed" -ForegroundColor Red
            $FAILED_SERVICES += $SERVICE
        }
    } else {
        Write-Host "✗ $SERVICE compilation failed" -ForegroundColor Red
        $FAILED_SERVICES += $SERVICE
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Build Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

if ($FAILED_SERVICES.Count -eq 0) {
    Write-Host "All services built successfully!" -ForegroundColor Green
} else {
    Write-Host "Failed services:" -ForegroundColor Red
    foreach ($SERVICE in $FAILED_SERVICES) {
        Write-Host "  - $SERVICE" -ForegroundColor Red
    }
}

Set-Location -Path "D:\code\FutureBase\backend"
