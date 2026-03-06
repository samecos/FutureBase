# 可行性验证阶段 - 脚本引擎POC验证报告

**项目名称**：半自动化建筑设计平台  
**文档版本**：v1.0  
**编写日期**：2024年  
**技术栈**：Python 3.11+ / Docker + gVisor / Temporal / CodeMirror 6

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [脚本执行POC](#2-脚本执行poc)
3. [任务调度POC](#3-任务调度poc)
4. [脚本开发环境POC](#4-脚本开发环境poc)
5. [性能优化验证](#5-性能优化验证)
6. [安全测试](#6-安全测试)
7. [POC执行计划](#7-poc执行计划)
8. [风险评估与缓解](#8-风险评估与缓解)
9. [结论与建议](#9-结论与建议)

---

## 1. 执行摘要

### 1.1 POC目标

本POC验证旨在确认以下核心技术方案的可行性：

| 验证领域 | 核心目标 | 成功标准 |
|---------|---------|---------|
| 脚本执行 | Python脚本在gVisor沙箱中安全执行 | 100%隔离恶意代码，资源限制生效 |
| 任务调度 | Temporal工作流可靠编排 | 任务成功率>99.9%，支持复杂依赖 |
| 开发环境 | CodeMirror 6提供良好开发体验 | 代码补全延迟<200ms，调试功能完整 |
| 性能优化 | 预热池和缓存机制有效 | 二次执行提速>80% |
| 安全防护 | 多层安全防护有效 | 通过所有安全测试用例 |

### 1.2 验证范围

```
┌─────────────────────────────────────────────────────────────────┐
│                        脚本引擎POC验证范围                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  脚本执行层  │  │  任务调度层  │  │  开发环境层  │             │
│  │  Python 3.11 │  │  Temporal   │  │ CodeMirror6 │             │
│  │  gVisor     │  │  Workflows  │  │  Monaco-like│             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         └────────────────┼────────────────┘                     │
│                          ▼                                      │
│              ┌─────────────────────┐                           │
│              │    安全隔离层        │                           │
│              │  Docker + gVisor    │                           │
│              └─────────────────────┘                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 脚本执行POC

### 2.1 Python脚本执行验证

#### 2.1.1 验证目标

验证Python脚本在隔离环境中的执行能力，包括：
- 标准库和常用第三方库的可用性
- 脚本输入输出处理
- 异常处理和错误报告

#### 2.1.2 测试环境配置

```yaml
# docker-compose.poc.yml
version: '3.8'
services:
  script-executor:
    image: python:3.11-slim
    runtime: runsc  # gVisor运行时
    resources:
      limits:
        cpus: '2.0'
        memory: 512M
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp:noexec,nosuid,size=100m
```

#### 2.1.3 测试用例设计

| 用例ID | 用例名称 | 测试内容 | 预期结果 | 优先级 |
|-------|---------|---------|---------|-------|
| PY-001 | 基础执行测试 | 执行简单Python脚本计算1+1 | 返回结果2，执行成功 | P0 |
| PY-002 | 标准库测试 | 使用os、sys、json等标准库 | 所有标准库正常可用 | P0 |
| PY-003 | 第三方库测试 | 使用numpy、pandas、shapely | 库正确安装并可导入 | P0 |
| PY-004 | 几何计算测试 | 使用shapely进行几何运算 | 几何计算结果正确 | P1 |
| PY-005 | 文件IO测试 | 读写临时文件 | 文件操作成功，隔离有效 | P0 |
| PY-006 | 异常处理测试 | 触发各种Python异常 | 异常被捕获，信息完整 | P1 |
| PY-007 | 长时间运行测试 | 执行耗时30秒的脚本 | 正常完成，不超资源限制 | P1 |
| PY-008 | 并发执行测试 | 同时执行10个脚本 | 各脚本独立执行，无干扰 | P1 |

#### 2.1.4 测试脚本示例

```python
# test_basic_execution.py
"""基础执行测试脚本"""
import sys
import json
import time

def test_basic_math():
    """测试基础数学运算"""
    result = sum(range(100))
    assert result == 4950, f"Expected 4950, got {result}"
    return {"test": "basic_math", "status": "pass"}

def test_json_serialization():
    """测试JSON序列化"""
    data = {"building": {"floors": 10, "area": 5000}}
    json_str = json.dumps(data)
    recovered = json.loads(json_str)
    assert recovered == data
    return {"test": "json", "status": "pass"}

def test_geometry():
    """测试几何计算（需要shapely）"""
    try:
        from shapely.geometry import Polygon
        poly = Polygon([(0,0), (1,0), (1,1), (0,1)])
        area = poly.area
        assert abs(area - 1.0) < 0.001
        return {"test": "geometry", "status": "pass"}
    except ImportError:
        return {"test": "geometry", "status": "skip", "reason": "shapely not installed"}

if __name__ == "__main__":
    results = []
    results.append(test_basic_math())
    results.append(test_json_serialization())
    results.append(test_geometry())
    
    print(json.dumps({
        "status": "success",
        "results": results,
        "python_version": sys.version
    }))
```

### 2.2 gVisor沙箱隔离验证

#### 2.2.1 验证目标

验证gVisor提供的用户空间内核隔离能力：
- 系统调用拦截和过滤
- 文件系统隔离
- 网络访问控制
- 进程隔离

#### 2.2.2 gVisor架构验证

```
┌─────────────────────────────────────────────────────────────┐
│                    gVisor 隔离架构                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Python 应用进程                         │   │
│  │         (用户代码运行在隔离环境)                      │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │ 系统调用                          │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │              Sentry (用户空间内核)                    │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐  │   │
│  │  │ 内存管理 │  │ 文件系统 │  │ 网络栈  │  │ 调度器 │  │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └────────┘  │   │
│  └──────────────────────┬──────────────────────────────┘   │
│                         │ 有限系统调用                       │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │              Host Kernel (宿主机内核)                 │   │
│  │              严格限制的seccomp规则                    │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

#### 2.2.3 隔离测试用例

| 用例ID | 测试类型 | 测试内容 | 预期结果 | 优先级 |
|-------|---------|---------|---------|-------|
| ISO-001 | 文件系统隔离 | 尝试访问宿主机/etc/passwd | 访问被拒绝，返回权限错误 | P0 |
| ISO-002 | 进程隔离 | 尝试查看宿主机进程列表 | 只能看到容器内进程 | P0 |
| ISO-003 | 网络隔离 | 尝试访问外部网络（未授权） | 连接被拒绝或超时 | P0 |
| ISO-004 | 设备隔离 | 尝试访问/dev/mem等设备 | 访问被拒绝 | P0 |
| ISO-005 | 特权提升 | 尝试使用setuid程序提权 | 操作被拒绝 | P0 |
| ISO-006 | 内核漏洞 | 运行已知CVE利用代码 | 利用失败，沙箱保持完整 | P0 |

#### 2.2.4 隔离验证脚本

```python
# test_isolation.py
"""沙箱隔离验证脚本"""
import os
import sys
import subprocess

def test_filesystem_isolation():
    """测试文件系统隔离"""
    results = []
    
    # 尝试访问宿主机敏感文件
    sensitive_paths = [
        "/etc/passwd",
        "/etc/shadow",
        "/proc/1/environ",
        "/root/.ssh"
    ]
    
    for path in sensitive_paths:
        try:
            if os.path.exists(path):
                with open(path, 'r') as f:
                    content = f.read()
                results.append({
                    "path": path,
                    "accessible": True,
                    "risk": "HIGH"
                })
            else:
                results.append({
                    "path": path,
                    "accessible": False,
                    "status": "safe"
                })
        except PermissionError:
            results.append({
                "path": path,
                "accessible": False,
                "status": "blocked"
            })
        except Exception as e:
            results.append({
                "path": path,
                "accessible": False,
                "status": "error",
                "error": str(e)
            })
    
    return results

def test_process_isolation():
    """测试进程隔离"""
    try:
        # 尝试查看所有进程
        result = subprocess.run(['ps', 'aux'], capture_output=True, text=True)
        process_count = len(result.stdout.strip().split('\n')) - 1
        
        # 正常容器应该只能看到自己的进程
        return {
            "process_visible": process_count,
            "isolated": process_count < 20  # 容器内进程数通常较少
        }
    except Exception as e:
        return {"error": str(e)}

def test_network_isolation():
    """测试网络隔离"""
    import socket
    results = []
    
    # 测试外部连接（应该被阻止或限制）
    test_hosts = [
        ("8.8.8.8", 53),    # Google DNS
        ("169.254.169.254", 80),  # AWS metadata
    ]
    
    for host, port in test_hosts:
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(2)
            result = sock.connect_ex((host, port))
            sock.close()
            results.append({
                "host": host,
                "port": port,
                "connected": result == 0,
                "status": "allowed" if result == 0 else "blocked"
            })
        except Exception as e:
            results.append({
                "host": host,
                "port": port,
                "error": str(e),
                "status": "blocked"
            })
    
    return results

if __name__ == "__main__":
    print(json.dumps({
        "filesystem_isolation": test_filesystem_isolation(),
        "process_isolation": test_process_isolation(),
        "network_isolation": test_network_isolation()
    }, indent=2))
```

### 2.3 资源限制验证

#### 2.3.1 资源限制策略

```yaml
# 资源限制配置
resource_limits:
  cpu:
    default: "1.0"        # 默认1核
    max: "4.0"           # 最大4核
    quota_period: 100ms   # CPU配额周期
  
  memory:
    default: "512M"      # 默认512MB
    max: "4G"            # 最大4GB
    swap: "0"            # 禁用swap
    oom_kill: true       # OOM时杀死进程
  
  time:
    default: 60s         # 默认60秒超时
    max: 300s            # 最大300秒
    graceful_shutdown: 5s # 优雅关闭时间
  
  disk:
    tmpfs_size: "100M"   # 临时文件系统大小
    max_file_size: "10M" # 单个文件最大大小
    max_files: 100       # 最大文件数
  
  network:
    egress_rate: "10Mbps" # 出站带宽限制
    ingress_rate: "10Mbps" # 入站带宽限制
    max_connections: 10   # 最大连接数
```

#### 2.3.2 资源限制测试用例

| 用例ID | 测试场景 | 限制值 | 测试行为 | 预期结果 |
|-------|---------|-------|---------|---------|
| RES-001 | CPU限制 | 0.5核 | 计算密集型任务 | CPU使用率被限制在50% |
| RES-002 | 内存限制 | 256MB | 分配300MB内存 | OOM被触发，进程被杀死 |
| RES-003 | 时间限制 | 10秒 | 执行sleep 30 | 10秒后强制终止 |
| RES-004 | 磁盘限制 | 100MB | 写入150MB文件 | 写入失败，返回空间不足 |
| RES-005 | 文件数限制 | 100个 | 创建150个文件 | 创建失败，返回文件数超限 |
| RES-006 | 网络限制 | 1Mbps | 下载10MB文件 | 下载速度不超过1Mbps |

#### 2.3.3 资源限制测试脚本

```python
# test_resource_limits.py
"""资源限制验证脚本"""
import os
import sys
import time
import json
import resource

def test_cpu_limit():
    """测试CPU限制"""
    start = time.time()
    # CPU密集型计算
    count = 0
    while time.time() - start < 5:
        count += sum(i * i for i in range(10000))
    elapsed = time.time() - start
    
    return {
        "test": "cpu_limit",
        "elapsed_seconds": elapsed,
        "iterations": count
    }

def test_memory_limit():
    """测试内存限制"""
    allocations = []
    try:
        for i in range(100):
            # 每次分配10MB
            allocations.append(bytearray(10 * 1024 * 1024))
            time.sleep(0.1)
    except MemoryError:
        return {
            "test": "memory_limit",
            "status": "blocked",
            "allocated_mb": len(allocations) * 10
        }
    
    return {
        "test": "memory_limit",
        "status": "warning",
        "allocated_mb": len(allocations) * 10,
        "message": "Memory limit may not be enforced"
    }

def test_time_limit(timeout=10):
    """测试时间限制"""
    start = time.time()
    try:
        # 设置软限制
        resource.setrlimit(resource.RLIMIT_CPU, (timeout, timeout + 5))
        
        # 长时间运行
        time.sleep(timeout + 10)
        
        return {
            "test": "time_limit",
            "status": "warning",
            "message": "Time limit not enforced"
        }
    except Exception as e:
        elapsed = time.time() - start
        return {
            "test": "time_limit",
            "status": "enforced",
            "elapsed_seconds": elapsed,
            "error": str(e)
        }

def test_disk_limit():
    """测试磁盘限制"""
    files = []
    try:
        for i in range(200):
            filename = f"/tmp/test_file_{i}.dat"
            with open(filename, 'wb') as f:
                f.write(b'x' * (1024 * 1024))  # 1MB each
            files.append(filename)
    except (IOError, OSError) as e:
        # 清理
        for f in files:
            try:
                os.remove(f)
            except:
                pass
        
        return {
            "test": "disk_limit",
            "status": "enforced",
            "files_created": len(files),
            "error": str(e)
        }
    
    return {
        "test": "disk_limit",
        "status": "warning",
        "files_created": len(files),
        "message": "Disk limit may not be enforced"
    }

if __name__ == "__main__":
    print(json.dumps({
        "cpu": test_cpu_limit(),
        "memory": test_memory_limit(),
        "time": test_time_limit(),
        "disk": test_disk_limit()
    }, indent=2))
```

### 2.4 安全策略验证

#### 2.4.1 安全策略配置

```json
{
  "seccomp_profile": {
    "defaultAction": "SCMP_ACT_ERRNO",
    "syscalls": [
      {"names": ["read", "write", "open", "close"], "action": "SCMP_ACT_ALLOW"},
      {"names": ["execve", "execveat"], "action": "SCMP_ACT_ERRNO"},
      {"names": ["clone", "fork", "vfork"], "action": "SCMP_ACT_ERRNO"},
      {"names": ["ptrace"], "action": "SCMP_ACT_ERRNO"},
      {"names": ["mount", "umount", "umount2"], "action": "SCMP_ACT_ERRNO"},
      {"names": ["reboot"], "action": "SCMP_ACT_ERRNO"},
      {"names": ["open_by_handle_at"], "action": "SCMP_ACT_ERRNO"}
    ]
  },
  "capabilities": {
    "drop": ["ALL"],
    "add": []
  },
  "read_only_rootfs": true,
  "no_new_privileges": true,
  "user": {
    "uid": 1000,
    "gid": 1000
  }
}
```

---

## 3. 任务调度POC

### 3.1 Temporal工作流验证

#### 3.1.1 Temporal架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                     Temporal 任务调度架构                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐     ┌─────────────────────────────────────┐  │
│  │   Client     │────▶│         Temporal Server              │  │
│  │  (Web/API)   │     │  ┌─────────┐ ┌─────────┐ ┌────────┐ │  │
│  └──────────────┘     │  │ Frontend│ │ History │ │Matching│ │  │
│                       │  └─────────┘ └─────────┘ └────────┘ │  │
│                       │  ┌─────────┐ ┌─────────┐             │  │
│                       │  │ Worker  │ │  Task   │             │  │
│                       │  │ Service │ │  Queue  │             │  │
│                       │  └─────────┘ └─────────┘             │  │
│                       └──────────────────┬────────────────────┘  │
│                                          │                      │
│                       ┌──────────────────▼────────────────────┐ │
│                       │         Worker Pool                    │ │
│                       │  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐     │ │
│                       │  │  W1 │ │  W2 │ │  W3 │ │ Wn  │     │ │
│                       │  └─────┘ └─────┘ └─────┘ └─────┘     │ │
│                       └────────────────────────────────────────┘ │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Persistence Layer (PostgreSQL)              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.1.2 工作流定义示例

```python
# workflow_definitions.py
"""Temporal工作流定义 - 建筑设计脚本执行"""
from datetime import timedelta
from temporalio import workflow
from temporalio.common import RetryPolicy
from dataclasses import dataclass
from typing import List, Dict, Any

@dataclass
class ScriptTask:
    """脚本任务定义"""
    task_id: str
    script_code: str
    inputs: Dict[str, Any]
    timeout_seconds: int = 60
    memory_limit_mb: int = 512
    cpu_limit: float = 1.0

@dataclass
class TaskResult:
    """任务执行结果"""
    task_id: str
    status: str  # success, failure, timeout
    output: Any
    execution_time_ms: int
    logs: List[str]

@workflow.defn
class BuildingDesignWorkflow:
    """建筑设计工作流"""
    
    @workflow.run
    async def run(self, design_params: Dict[str, Any]) -> Dict[str, Any]:
        """执行完整的设计流程"""
        results = {}
        
        # 1. 参数验证
        validation_result = await workflow.execute_activity(
            "validate_parameters",
            design_params,
            start_to_close_timeout=timedelta(seconds=30),
            retry_policy=RetryPolicy(maximum_attempts=3)
        )
        results["validation"] = validation_result
        
        # 2. 生成基础几何
        geometry_task = ScriptTask(
            task_id="generate_geometry",
            script_code=self._get_geometry_script(),
            inputs=design_params
        )
        geometry_result = await workflow.execute_activity(
            "execute_script",
            geometry_task,
            start_to_close_timeout=timedelta(seconds=120),
        )
        results["geometry"] = geometry_result
        
        # 3. 结构分析（依赖几何结果）
        if geometry_result.status == "success":
            structure_task = ScriptTask(
                task_id="analyze_structure",
                script_code=self._get_structure_script(),
                inputs={"geometry": geometry_result.output}
            )
            structure_result = await workflow.execute_activity(
                "execute_script",
                structure_task,
                start_to_close_timeout=timedelta(seconds=180),
            )
            results["structure"] = structure_result
        
        # 4. 能耗分析（与结构分析并行）
        if geometry_result.status == "success":
            energy_task = ScriptTask(
                task_id="analyze_energy",
                script_code=self._get_energy_script(),
                inputs={"geometry": geometry_result.output}
            )
            energy_result = await workflow.execute_activity(
                "execute_script",
                energy_task,
                start_to_close_timeout=timedelta(seconds=180),
            )
            results["energy"] = energy_result
        
        # 5. 生成报告
        report_result = await workflow.execute_activity(
            "generate_report",
            results,
            start_to_close_timeout=timedelta(seconds=60),
        )
        
        return {
            "workflow_id": workflow.info().workflow_id,
            "results": results,
            "report": report_result
        }
    
    def _get_geometry_script(self) -> str:
        return '''
import json
from shapely.geometry import Polygon, MultiPolygon

def generate_building_geometry(params):
    floors = params.get('floors', 5)
    width = params.get('width', 20)
    depth = params.get('depth', 15)
    
    # 生成基础平面
    footprint = Polygon([
        (0, 0), (width, 0), (width, depth), (0, depth)
    ])
    
    return {
        "footprint": footprint.__geo_interface__,
        "floors": floors,
        "floor_area": footprint.area,
        "total_area": footprint.area * floors
    }

result = generate_building_geometry(inputs)
print(json.dumps(result))
'''
```

#### 3.1.3 工作流测试用例

| 用例ID | 测试场景 | 测试内容 | 预期结果 | 优先级 |
|-------|---------|---------|---------|-------|
| WF-001 | 简单顺序执行 | 3个任务顺序执行 | 按顺序完成，结果正确 | P0 |
| WF-002 | 并行执行 | 2个无依赖任务并行 | 并行执行，总时间减少 | P0 |
| WF-003 | 条件分支 | 根据条件选择分支 | 正确分支被执行 | P1 |
| WF-004 | 循环执行 | 批量处理10个任务 | 循环完成，结果聚合 | P1 |
| WF-005 | 长时间工作流 | 执行5分钟的工作流 | 正常完成，状态持久化 | P1 |
| WF-006 | 工作流查询 | 查询运行中的工作流 | 返回当前状态 | P1 |
| WF-007 | 信号处理 | 向工作流发送信号 | 信号被正确处理 | P1 |
| WF-008 | 定时触发 | 设置定时工作流 | 按时触发执行 | P2 |

### 3.2 任务编排验证

#### 3.2.1 依赖关系模型

```
                    ┌─────────────────┐
                    │  参数验证任务    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
    ┌─────────────────┐ ┌─────────┐ ┌─────────────────┐
    │  生成基础几何    │ │ 场地分析 │ │  规范检查任务    │
    └────────┬────────┘ └────┬────┘ └────────┬────────┘
             │               │               │
             └───────────────┼───────────────┘
                             ▼
                    ┌─────────────────┐
                    │  结构分析任务    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
    ┌─────────────────┐ ┌─────────┐ ┌─────────────────┐
    │  能耗分析任务    │ │机电分析 │ │  造价估算任务    │
    └────────┬────────┘ └────┬────┘ └────────┬────────┘
             │               │               │
             └───────────────┼───────────────┘
                             ▼
                    ┌─────────────────┐
                    │  报告生成任务    │
                    └─────────────────┘
```

#### 3.2.2 编排测试用例

| 用例ID | 编排模式 | 测试内容 | 预期结果 |
|-------|---------|---------|---------|
| ORC-001 | 顺序编排 | A→B→C顺序执行 | C在B完成后执行，B在A完成后执行 |
| ORC-002 | 并行编排 | A和B并行，然后C | A、B并行，都完成后执行C |
| ORC-003 | 条件编排 | if A成功 then B else C | 根据A结果选择B或C |
| ORC-004 | 循环编排 | for each item in list do A | 列表中每个元素执行A |
| ORC-005 | 动态编排 | 运行时决定下一个任务 | 动态任务被正确调度和执行 |
| ORC-006 | 异常编排 | A失败时执行B | A失败触发B执行 |

### 3.3 依赖管理验证

#### 3.3.1 依赖解析设计

```python
# dependency_manager.py
"""任务依赖管理器"""
from typing import Dict, List, Set
from dataclasses import dataclass, field
from collections import defaultdict, deque

@dataclass
class TaskDependency:
    """任务依赖定义"""
    task_id: str
    dependencies: List[str] = field(default_factory=list)
    outputs: List[str] = field(default_factory=list)
    
class DependencyGraph:
    """依赖图管理"""
    
    def __init__(self):
        self.tasks: Dict[str, TaskDependency] = {}
        self.dependents: Dict[str, List[str]] = defaultdict(list)
        
    def add_task(self, task: TaskDependency):
        """添加任务到依赖图"""
        self.tasks[task.task_id] = task
        for dep in task.dependencies:
            self.dependents[dep].append(task.task_id)
    
    def get_execution_order(self) -> List[List[str]]:
        """获取并行执行层级"""
        in_degree = {task_id: len(task.dependencies) 
                     for task_id, task in self.tasks.items()}
        
        levels = []
        while in_degree:
            # 找到入度为0的任务
            current_level = [task_id for task_id, degree in in_degree.items() 
                           if degree == 0]
            
            if not current_level:
                raise ValueError("Circular dependency detected")
            
            levels.append(current_level)
            
            # 更新入度
            for task_id in current_level:
                del in_degree[task_id]
                for dependent in self.dependents[task_id]:
                    if dependent in in_degree:
                        in_degree[dependent] -= 1
        
        return levels
    
    def get_ready_tasks(self, completed: Set[str]) -> List[str]:
        """获取可以执行的任务"""
        ready = []
        for task_id, task in self.tasks.items():
            if task_id not in completed:
                if all(dep in completed for dep in task.dependencies):
                    ready.append(task_id)
        return ready
```

#### 3.3.2 依赖测试用例

| 用例ID | 测试场景 | 依赖关系 | 预期执行顺序 |
|-------|---------|---------|-------------|
| DEP-001 | 线性依赖 | A→B→C | [[A], [B], [C]] |
| DEP-002 | 扇出依赖 | A→B, A→C | [[A], [B, C]] |
| DEP-003 | 扇入依赖 | A→C, B→C | [[A, B], [C]] |
| DEP-004 | 复杂依赖 | A→B, A→C, B→D, C→D | [[A], [B, C], [D]] |
| DEP-005 | 循环依赖 | A→B, B→C, C→A | 检测到循环，报错 |
| DEP-006 | 数据依赖 | A输出x，B需要x | B在A完成后执行，x传递成功 |

### 3.4 失败重试验证

#### 3.4.1 重试策略配置

```python
# retry_policies.py
"""重试策略定义"""
from temporalio.common import RetryPolicy
from datetime import timedelta

# 默认重试策略
DEFAULT_RETRY_POLICY = RetryPolicy(
    initial_interval=timedelta(seconds=1),
    backoff_coefficient=2.0,
    maximum_interval=timedelta(minutes=1),
    maximum_attempts=3,
    non_retryable_error_types=["SecurityError", "ValidationError"]
)

# 长时间任务重试策略
LONG_RUNNING_RETRY_POLICY = RetryPolicy(
    initial_interval=timedelta(seconds=5),
    backoff_coefficient=1.5,
    maximum_interval=timedelta(minutes=5),
    maximum_attempts=5,
    non_retryable_error_types=["SecurityError", "ValidationError", "TimeoutError"]
)

# 外部服务调用重试策略
EXTERNAL_SERVICE_RETRY_POLICY = RetryPolicy(
    initial_interval=timedelta(seconds=2),
    backoff_coefficient=2.0,
    maximum_interval=timedelta(minutes=2),
    maximum_attempts=10,
    non_retryable_error_types=["AuthenticationError", "AuthorizationError"]
)
```

#### 3.4.2 重试测试用例

| 用例ID | 测试场景 | 重试配置 | 预期行为 |
|-------|---------|---------|---------|
| RET-001 | 临时失败 | 重试3次，指数退避 | 第2次成功，完成执行 |
| RET-002 | 持续失败 | 重试3次，全部失败 | 3次后标记失败，触发补偿 |
| RET-003 | 不可重试错误 | 遇到SecurityError | 立即失败，不重试 |
| RET-004 | 超时重试 | 任务超时 | 超时后重试，重置计时 |
| RET-005 | 手动重试 | 失败后手动触发 | 从失败点重新执行 |
| RET-006 | 补偿执行 | 任务链中某步失败 | 已完成的任务执行补偿 |

---

## 4. 脚本开发环境POC

### 4.1 CodeMirror 6集成验证

#### 4.1.1 编辑器架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    CodeMirror 6 编辑器架构                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Editor View                           │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │              Editor State                        │   │   │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────────┐ │   │   │
│  │  │  │ Document│ │Selection│ │  Facets │ │ Effects│ │   │   │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └────────┘ │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  │                         │                              │   │
│  │  ┌──────────────────────┼──────────────────────────┐  │   │
│  │  │              Extensions (插件系统)                │  │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐  │  │   │
│  │  │  │ Python │ │ Linter │ │ Autocomplete│ │Debugger│  │  │   │
│  │  │  │Language│ │        │ │          │ │        │  │  │   │
│  │  │  └────────┘ └────────┘ └────────┘ └──────────┘  │  │   │
│  │  └──────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Language Server Protocol (LSP)              │   │
│  │         ┌─────────────┐         ┌─────────────┐         │   │
│  │         │  Pylsp Server│◄──────►│  TypeScript │         │   │
│  │         │  (Python)   │         │   Client    │         │   │
│  │         └─────────────┘         └─────────────┘         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.1.2 CodeMirror 6配置示例

```typescript
// editor-config.ts
import { EditorView, basicSetup } from 'codemirror';
import { python } from '@codemirror/lang-python';
import { linter, lintGutter } from '@codemirror/lint';
import { autocompletion, completionKeymap } from '@codemirror/autocomplete';
import { keymap } from '@codemirror/view';
import { oneDark } from '@codemirror/theme-one-dark';

// Python LSP客户端
class PythonLSPClient {
  private ws: WebSocket;
  
  constructor(url: string) {
    this.ws = new WebSocket(url);
  }
  
  async getCompletions(params: CompletionParams): Promise<CompletionItem[]> {
    return new Promise((resolve) => {
      const requestId = generateId();
      
      const handler = (event: MessageEvent) => {
        const response = JSON.parse(event.data);
        if (response.id === requestId) {
          this.ws.removeEventListener('message', handler);
          resolve(response.result.items || []);
        }
      };
      
      this.ws.addEventListener('message', handler);
      this.ws.send(JSON.stringify({
        jsonrpc: '2.0',
        id: requestId,
        method: 'textDocument/completion',
        params
      }));
    });
  }
}

// 创建编辑器
export function createPythonEditor(
  parent: HTMLElement,
  options: EditorOptions
): EditorView {
  const lspClient = new PythonLSPClient(options.lspUrl);
  
  return new EditorView({
    parent,
    state: EditorState.create({
      doc: options.initialCode || '',
      extensions: [
        basicSetup,
        python(),
        oneDark,
        lintGutter(),
        
        // 自定义Linter
        linter(async (view) => {
          const diagnostics = await lspClient.getDiagnostics({
            textDocument: {
              uri: options.fileUri,
              text: view.state.doc.toString()
            }
          });
          return diagnostics.map(d => ({
            from: d.range.start,
            to: d.range.end,
            severity: d.severity,
            message: d.message
          }));
        }),
        
        // 自动补全
        autocompletion({
          override: [async (context) => {
            const word = context.matchBefore(/\w*/);
            if (!word || word.from === word.to) return null;
            
            const completions = await lspClient.getCompletions({
              textDocument: { uri: options.fileUri },
              position: {
                line: context.state.doc.lineAt(word.from).number - 1,
                character: word.from - context.state.doc.lineAt(word.from).from
              }
            });
            
            return {
              from: word.from,
              options: completions.map(c => ({
                label: c.label,
                type: c.kind,
                info: c.documentation?.value,
                apply: c.insertText || c.label
              }))
            };
          }]
        }),
        
        // 快捷键
        keymap.of([
          ...completionKeymap,
          {
            key: 'Ctrl-Enter',
            run: () => {
              options.onExecute?.(view.state.doc.toString());
              return true;
            }
          },
          {
            key: 'F5',
            run: () => {
              options.onDebug?.(view.state.doc.toString());
              return true;
            }
          }
        ])
      ]
    })
  });
}
```

#### 4.1.3 集成测试用例

| 用例ID | 测试功能 | 测试内容 | 预期结果 | 优先级 |
|-------|---------|---------|---------|-------|
| CM-001 | 基础编辑 | 输入Python代码 | 语法高亮正确，无卡顿 | P0 |
| CM-002 | 主题切换 | 切换明暗主题 | 主题正确应用，无闪烁 | P1 |
| CM-003 | 代码折叠 | 折叠函数/类 | 折叠展开正常 | P1 |
| CM-004 | 多光标编辑 | 使用多光标 | 多光标功能正常 | P2 |
| CM-005 | 搜索替换 | 搜索替换文本 | 功能正常，支持正则 | P1 |
| CM-006 | 大文件处理 | 打开10000行文件 | 滚动流畅，无明显延迟 | P1 |

### 4.2 代码补全验证

#### 4.2.1 补全功能设计

```python
# completion_service.py
"""代码补全服务"""
from typing import List, Dict, Optional
from dataclasses import dataclass
import jedi

@dataclass
class CompletionItem:
    """补全项"""
    label: str
    kind: str  # variable, function, class, module, etc.
    detail: Optional[str] = None
    documentation: Optional[str] = None
    insert_text: Optional[str] = None

class PythonCompletionService:
    """Python代码补全服务"""
    
    def __init__(self, project_path: str):
        self.project = jedi.Project(project_path)
    
    def get_completions(
        self, 
        code: str, 
        line: int, 
        column: int,
        file_path: Optional[str] = None
    ) -> List[CompletionItem]:
        """获取代码补全建议"""
        script = jedi.Script(
            code=code,
            path=file_path,
            project=self.project
        )
        
        completions = script.complete(line, column)
        
        return [
            CompletionItem(
                label=c.name,
                kind=self._map_kind(c.type),
                detail=c.description,
                documentation=c.docstring(),
                insert_text=c.complete
            )
            for c in completions
        ]
    
    def _map_kind(self, jedi_type: str) -> str:
        """映射jedi类型到LSP类型"""
        kind_map = {
            'module': 'module',
            'class': 'class',
            'instance': 'variable',
            'function': 'function',
            'param': 'variable',
            'path': 'file',
            'keyword': 'keyword',
            'property': 'property',
            'statement': 'variable'
        }
        return kind_map.get(jedi_type, 'text')
    
    def get_signature_help(
        self, 
        code: str, 
        line: int, 
        column: int
    ) -> Optional[Dict]:
        """获取函数签名帮助"""
        script = jedi.Script(code=code, project=self.project)
        signatures = script.get_signatures(line, column)
        
        if not signatures:
            return None
        
        sig = signatures[0]
        return {
            "label": sig.to_string(),
            "documentation": sig.docstring(),
            "parameters": [
                {"label": p.name, "documentation": p.description}
                for p in sig.params
            ]
        }
```

#### 4.2.2 补全测试用例

| 用例ID | 测试场景 | 输入示例 | 预期补全 |
|-------|---------|---------|---------|
| COMP-001 | 模块导入 | `import num` | numpy, numbers, numba... |
| COMP-002 | 属性访问 | `np.ar` | array, arange, arcsin... |
| COMP-003 | 函数参数 | `print(` | 显示函数签名 |
| COMP-004 | 自定义类 | `building.` | 类的属性和方法 |
| COMP-005 | 快速补全 | `def ` | 生成函数模板 |
| COMP-006 | 延迟要求 | 输入后100ms | 补全列表出现，延迟<200ms |

### 4.3 调试功能验证

#### 4.3.1 调试架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      调试系统架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐          ┌────────────────────────────┐  │
│  │   Frontend       │          │        Backend             │  │
│  │  (CodeMirror)    │          │                            │  │
│  │                  │          │  ┌──────────────────────┐  │  │
│  │ ┌──────────────┐ │          │  │   Debug Adapter      │  │  │
│  │ │ Breakpoint   │ │◄────────►│  │   (DAP Server)       │  │  │
│  │ │   Gutter     │ │   DAP    │  │                      │  │  │
│  │ └──────────────┘ │ Protocol │  │  ┌────────────────┐  │  │  │
│  │                  │          │  │  │  Debug Session │  │  │  │
│  │ ┌──────────────┐ │          │  │  │  Manager       │  │  │  │
│  │ │ Variable     │ │◄────────►│  │  └───────┬────────┘  │  │  │
│  │ │   Panel      │ │          │  │          │           │  │  │
│  │ └──────────────┘ │          │  │  ┌───────▼────────┐  │  │  │
│  │                  │          │  │  │  pdb / debugpy │  │  │  │
│  │ ┌──────────────┐ │          │  │  │  (Debugger)    │  │  │  │
│  │ │ Call Stack   │ │◄────────►│  │  └───────┬────────┘  │  │  │
│  │ │   Panel      │ │          │  │          │           │  │  │
│  │ └──────────────┘ │          │  │  ┌───────▼────────┐  │  │  │
│  │                  │          │  │  │  Script Runner │  │  │  │
│  │ ┌──────────────┐ │          │  │  │  (gVisor)      │  │  │  │
│  │ │ Debug        │ │─────────►│  │  └────────────────┘  │  │  │
│  │ │ Controls     │ │          │  │                      │  │  │
│  │ └──────────────┘ │          │  └──────────────────────┘  │  │
│  └──────────────────┘          └────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.3.2 调试测试用例

| 用例ID | 调试功能 | 测试内容 | 预期结果 |
|-------|---------|---------|---------|
| DBG-001 | 设置断点 | 在指定行设置断点 | 断点生效，执行暂停 |
| DBG-002 | 单步执行 | Step Over/Into/Out | 正确单步执行 |
| DBG-003 | 变量查看 | 查看当前变量值 | 变量值正确显示 |
| DBG-004 | 表达式求值 | 求值表达式 | 结果正确返回 |
| DBG-005 | 调用栈 | 查看调用栈 | 调用栈正确显示 |
| DBG-006 | 条件断点 | 设置条件断点 | 条件满足时暂停 |
| DBG-007 | 异常捕获 | 捕获未处理异常 | 异常时自动暂停 |

### 4.4 脚本版本管理验证

#### 4.4.1 版本管理设计

```python
# version_manager.py
"""脚本版本管理"""
from typing import List, Optional, Dict
from dataclasses import dataclass
from datetime import datetime
import hashlib
import json

@dataclass
class ScriptVersion:
    """脚本版本"""
    version_id: str
    script_id: str
    code: str
    author: str
    created_at: datetime
    commit_message: str
    parent_version: Optional[str] = None
    metadata: Dict = None
    
    @property
    def code_hash(self) -> str:
        return hashlib.sha256(self.code.encode()).hexdigest()[:16]

class ScriptVersionManager:
    """脚本版本管理器"""
    
    def __init__(self, storage_backend):
        self.storage = storage_backend
    
    def create_version(
        self, 
        script_id: str, 
        code: str, 
        author: str,
        commit_message: str
    ) -> ScriptVersion:
        """创建新版本"""
        # 获取当前版本作为父版本
        current = self.get_current_version(script_id)
        
        version = ScriptVersion(
            version_id=self._generate_version_id(),
            script_id=script_id,
            code=code,
            author=author,
            created_at=datetime.utcnow(),
            commit_message=commit_message,
            parent_version=current.version_id if current else None
        )
        
        self.storage.save_version(version)
        self.storage.set_current_version(script_id, version.version_id)
        
        return version
    
    def get_version_history(self, script_id: str) -> List[ScriptVersion]:
        """获取版本历史"""
        return self.storage.get_version_chain(script_id)
    
    def compare_versions(
        self, 
        version_id1: str, 
        version_id2: str
    ) -> Dict:
        """比较两个版本"""
        v1 = self.storage.get_version(version_id1)
        v2 = self.storage.get_version(version_id2)
        
        # 使用difflib生成差异
        import difflib
        diff = list(difflib.unified_diff(
            v1.code.splitlines(keepends=True),
            v2.code.splitlines(keepends=True),
            fromfile=f"v{version_id1}",
            tofile=f"v{version_id2}"
        ))
        
        return {
            "added_lines": sum(1 for d in diff if d.startswith('+')).
            "removed_lines": sum(1 for d in diff if d.startswith('-')),
            "diff": ''.join(diff)
        }
    
    def rollback(self, script_id: str, version_id: str) -> ScriptVersion:
        """回滚到指定版本"""
        target = self.storage.get_version(version_id)
        
        # 创建新的回滚版本
        rollback_version = ScriptVersion(
            version_id=self._generate_version_id(),
            script_id=script_id,
            code=target.code,
            author="system",
            created_at=datetime.utcnow(),
            commit_message=f"Rollback to version {version_id}",
            parent_version=self.get_current_version(script_id).version_id
        )
        
        self.storage.save_version(rollback_version)
        self.storage.set_current_version(script_id, rollback_version.version_id)
        
        return rollback_version
```

#### 4.4.2 版本管理测试用例

| 用例ID | 测试功能 | 测试内容 | 预期结果 |
|-------|---------|---------|---------|
| VER-001 | 创建版本 | 保存脚本新版本 | 版本创建成功，有唯一ID |
| VER-002 | 版本历史 | 查看版本历史 | 按时间顺序显示所有版本 |
| VER-003 | 版本比较 | 比较两个版本 | 正确显示差异 |
| VER-004 | 版本回滚 | 回滚到历史版本 | 当前版本更新为历史版本 |
| VER-005 | 分支管理 | 创建分支版本 | 支持并行开发分支 |
| VER-006 | 版本标签 | 给版本打标签 | 标签正确关联版本 |

---

## 5. 性能优化验证

### 5.1 预热池验证

#### 5.1.1 预热池架构

```
┌─────────────────────────────────────────────────────────────────┐
│                       预热池架构                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Pool Manager                          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │
│  │  │  Warm Pool  │  │  Cold Pool  │  │  Busy Pool  │     │   │
│  │  │  (Ready)    │  │  (Creating) │  │  (Running)  │     │   │
│  │  │             │  │             │  │             │     │   │
│  │  │ [C1][C2][C3]│  │ [C4][C5]    │  │ [C6][C7]    │     │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │
│  │                                                         │   │
│  │  Pool Config:                                           │   │
│  │  - min_warm: 5      (最小预热容器数)                      │   │
│  │  - max_warm: 20     (最大预热容器数)                      │   │
│  │  - max_total: 100   (最大总容器数)                        │   │
│  │  - idle_timeout: 5m (空闲超时)                           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Container Lifecycle                   │   │
│  │                                                         │   │
│  │   ┌──────┐    ┌──────┐    ┌──────┐    ┌──────┐        │   │
│  │   │Creating│──►│ Warm │──►│ Busy │──►│Recycle│        │   │
│  │   └──────┘    └──────┘    └──┬───┘    └──┬───┘        │   │
│  │                              │           │             │   │
│  │                              ▼           ▼             │   │
│  │                           ┌──────┐    ┌──────┐        │   │
│  │                           │Destroy│   │ Reuse │        │   │
│  │                           └──────┘    └──────┘        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 5.1.2 预热池实现

```python
# container_pool.py
"""容器预热池管理"""
import asyncio
from typing import Dict, List, Optional, Set
from dataclasses import dataclass
from datetime import datetime, timedelta
import docker

@dataclass
class PooledContainer:
    """池化容器"""
    container_id: str
    status: str  # 'warm', 'busy', 'recycling'
    created_at: datetime
    last_used: datetime
    execution_count: int = 0
    
class ContainerPool:
    """容器预热池"""
    
    def __init__(
        self,
        image: str,
        min_warm: int = 5,
        max_warm: int = 20,
        max_total: int = 100,
        idle_timeout: timedelta = timedelta(minutes=5)
    ):
        self.image = image
        self.min_warm = min_warm
        self.max_warm = max_warm
        self.max_total = max_total
        self.idle_timeout = idle_timeout
        
        self.docker_client = docker.from_env()
        self.warm_containers: Dict[str, PooledContainer] = {}
        self.busy_containers: Dict[str, PooledContainer] = {}
        self._lock = asyncio.Lock()
        
        # 启动维护任务
        asyncio.create_task(self._maintenance_loop())
    
    async def acquire(self) -> Optional[PooledContainer]:
        """获取一个预热容器"""
        async with self._lock:
            # 优先从warm池获取
            if self.warm_containers:
                container_id, container = self.warm_containers.popitem()
                container.status = 'busy'
                container.last_used = datetime.utcnow()
                self.busy_containers[container_id] = container
                return container
            
            # warm池为空，检查是否可以创建新容器
            total = len(self.warm_containers) + len(self.busy_containers)
            if total < self.max_total:
                # 异步创建新容器
                return await self._create_container()
            
            return None
    
    async def release(self, container_id: str, recycle: bool = True):
        """释放容器回池"""
        async with self._lock:
            if container_id not in self.busy_containers:
                return
            
            container = self.busy_containers.pop(container_id)
            
            if recycle and len(self.warm_containers) < self.max_warm:
                # 清理容器状态
                await self._cleanup_container(container_id)
                container.status = 'warm'
                container.execution_count += 1
                self.warm_containers[container_id] = container
            else:
                # 销毁容器
                await self._destroy_container(container_id)
    
    async def _create_container(self) -> PooledContainer:
        """创建新容器"""
        container = self.docker_client.containers.run(
            self.image,
            detach=True,
            command=['sleep', 'infinity'],
            runtime='runsc',
            mem_limit='512m',
            cpu_quota=100000,
            read_only=True
        )
        
        pooled = PooledContainer(
            container_id=container.id,
            status='busy',
            created_at=datetime.utcnow(),
            last_used=datetime.utcnow()
        )
        
        self.busy_containers[container.id] = pooled
        return pooled
    
    async def _maintenance_loop(self):
        """维护循环"""
        while True:
            await asyncio.sleep(30)
            await self._maintain_pool()
    
    async def _maintain_pool(self):
        """维护池状态"""
        async with self._lock:
            now = datetime.utcnow()
            
            # 清理超时容器
            to_remove = []
            for cid, container in self.warm_containers.items():
                if now - container.last_used > self.idle_timeout:
                    to_remove.append(cid)
            
            for cid in to_remove:
                container = self.warm_containers.pop(cid)
                await self._destroy_container(cid)
            
            # 补充warm池
            while (len(self.warm_containers) < self.min_warm and
                   len(self.warm_containers) + len(self.busy_containers) < self.max_total):
                await self._create_warm_container()
```

#### 5.1.3 预热池测试用例

| 用例ID | 测试场景 | 测试内容 | 预期结果 |
|-------|---------|---------|---------|
| POOL-001 | 预热效果 | 对比冷启动和预热启动 | 预热启动<1s，冷启动>5s |
| POOL-002 | 池扩容 | 并发请求超过min_warm | 自动扩容到max_warm |
| POOL-003 | 池收缩 | 空闲超时后 | 自动收缩到min_warm |
| POOL-004 | 容器复用 | 多次执行脚本 | 同一容器执行多次 |
| POOL-005 | 资源隔离 | 容器复用后 | 文件系统状态正确清理 |
| POOL-006 | 池耗尽 | 请求超过max_total | 请求排队或拒绝 |

### 5.2 增量计算验证

#### 5.2.1 增量计算设计

```python
# incremental_computation.py
"""增量计算引擎"""
from typing import Dict, Any, Callable, Optional
from dataclasses import dataclass
import hashlib
import json

@dataclass
class ComputationNode:
    """计算节点"""
    node_id: str
    compute_func: Callable
    dependencies: List[str]
    cache_key_func: Optional[Callable] = None

class IncrementalEngine:
    """增量计算引擎"""
    
    def __init__(self, cache_backend):
        self.cache = cache_backend
        self.nodes: Dict[str, ComputationNode] = {}
        self.results: Dict[str, Any] = {}
        self.versions: Dict[str, str] = {}
    
    def register_node(self, node: ComputationNode):
        """注册计算节点"""
        self.nodes[node.node_id] = node
    
    async def compute(
        self, 
        node_id: str, 
        inputs: Dict[str, Any]
    ) -> Any:
        """执行增量计算"""
        node = self.nodes[node_id]
        
        # 1. 计算当前缓存键
        cache_key = self._compute_cache_key(node, inputs)
        
        # 2. 检查依赖是否有变化
        deps_changed = await self._check_dependencies_changed(node, inputs)
        
        # 3. 检查缓存是否有效
        if not deps_changed:
            cached = await self.cache.get(cache_key)
            if cached is not None:
                return cached
        
        # 4. 执行计算
        # 先计算依赖
        dep_results = {}
        for dep_id in node.dependencies:
            dep_results[dep_id] = await self.compute(dep_id, inputs)
        
        # 执行当前节点
        result = await node.compute_func(inputs, dep_results)
        
        # 5. 缓存结果
        await self.cache.set(cache_key, result, ttl=3600)
        self.versions[node_id] = cache_key
        self.results[node_id] = result
        
        return result
    
    def _compute_cache_key(
        self, 
        node: ComputationNode, 
        inputs: Dict[str, Any]
    ) -> str:
        """计算缓存键"""
        if node.cache_key_func:
            return node.cache_key_func(inputs)
        
        # 默认使用输入哈希
        key_data = {
            "node_id": node.node_id,
            "inputs": inputs,
            "code_hash": self._get_code_hash(node.compute_func)
        }
        return hashlib.sha256(
            json.dumps(key_data, sort_keys=True).encode()
        ).hexdigest()
    
    async def _check_dependencies_changed(
        self, 
        node: ComputationNode, 
        inputs: Dict[str, Any]
    ) -> bool:
        """检查依赖是否变化"""
        for dep_id in node.dependencies:
            dep_key = self._compute_cache_key(self.nodes[dep_id], inputs)
            if self.versions.get(dep_id) != dep_key:
                return True
        return False
```

#### 5.2.2 增量计算测试用例

| 用例ID | 测试场景 | 测试内容 | 预期结果 |
|-------|---------|---------|---------|
| INC-001 | 首次计算 | 执行全新计算 | 完整执行，结果缓存 |
| INC-002 | 无变化重算 | 输入未变重新计算 | 直接返回缓存结果 |
| INC-003 | 部分变化 | 依赖链中部分节点变化 | 只重算变化节点 |
| INC-004 | 缓存失效 | 手动清除缓存 | 下次执行完整重算 |
| INC-005 | 代码变化 | 节点代码变化 | 该节点及下游重算 |
| INC-006 | 大输入处理 | 大对象作为输入 | 使用内容哈希，性能可接受 |

### 5.3 结果缓存验证

#### 5.3.1 缓存架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      多级缓存架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    L1: In-Memory Cache                   │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │  LRU Cache (256MB)                              │   │   │
│  │  │  - TTL: 5 minutes                               │   │   │
│  │  │  - Hit Rate Target: >90%                        │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │ L1 Miss                            │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    L2: Redis Cache                       │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │  Distributed Cache                              │   │   │
│  │  │  - TTL: 1 hour                                  │   │   │
│  │  │  - Serialization: MessagePack                   │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │ L2 Miss                            │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    L3: Persistent Storage                │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │  S3 / MinIO                                     │   │   │
│  │  │  - Long-term storage                            │   │   │
│  │  │  - Versioned objects                            │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Cache Key Format:                                              │
│  script:{script_id}:v{version}:input_hash:{input_hash}         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 5.3.2 缓存测试用例

| 用例ID | 测试场景 | 测试内容 | 预期结果 |
|-------|---------|---------|---------|
| CACHE-001 | L1命中 | 重复请求相同结果 | 内存缓存返回，延迟<1ms |
| CACHE-002 | L2命中 | L1未命中，L2命中 | Redis返回，延迟<10ms |
| CACHE-003 | L3命中 | L1/L2未命中 | 从存储加载，延迟<100ms |
| CACHE-004 | 缓存穿透 | 请求不存在的数据 | 只查询一次，后续直接返回空 |
| CACHE-005 | 缓存雪崩 | 大量缓存同时过期 | 使用随机TTL，避免同时过期 |
| CACHE-006 | 缓存一致性 | 更新脚本后 | 旧缓存失效，使用新结果 |

---

## 6. 安全测试

### 6.1 代码注入攻击防护验证

#### 6.1.1 攻击测试用例

| 用例ID | 攻击类型 | 攻击代码 | 预期防护 |
|-------|---------|---------|---------|
| SEC-001 | 系统命令注入 | `os.system('rm -rf /')` | 被拦截，返回权限错误 |
| SEC-002 | 代码执行注入 | `exec('import os; os.system(...)')` | exec被禁用或拦截 |
| SEC-003 | 动态导入攻击 | `__import__('os').system(...)` | 危险模块导入被拦截 |
| SEC-004 | 反射攻击 | `getattr(__builtins__, 'eval')(...)` | 危险函数访问被拦截 |
| SEC-005 | 文件写入攻击 | `open('/etc/passwd', 'w')` | 写入系统文件被拦截 |
| SEC-006 | 网络攻击 | `socket.connect(('10.0.0.1', 22))` | 未授权网络访问被拦截 |
| SEC-007 | 反序列化攻击 | `pickle.loads(malicious_data)` | 危险反序列化被拦截 |
| SEC-008 | 模板注入 | `jinja2.Template(user_input).render()` | 模板注入被检测和拦截 |

#### 6.1.2 防护实现

```python
# security_sandbox.py
"""安全沙箱实现"""
import ast
import builtins
import sys
from typing import Set, List

class SecurityError(Exception):
    """安全错误"""
    pass

class CodeAnalyzer(ast.NodeVisitor):
    """代码安全分析器"""
    
    # 危险函数黑名单
    DANGEROUS_FUNCTIONS = {
        'eval', 'exec', 'compile', '__import__', 'open',
        'input', 'raw_input', 'reload', 'exit', 'quit'
    }
    
    # 危险模块黑名单
    DANGEROUS_MODULES = {
        'os', 'sys', 'subprocess', 'socket', 'urllib',
        'ftplib', 'telnetlib', 'pickle', 'marshal',
        'ctypes', 'multiprocessing'
    }
    
    # 危险属性黑名单
    DANGEROUS_ATTRIBUTES = {
        '__class__', '__bases__', '__mro__', '__subclasses__',
        '__globals__', '__code__', '__func__', '__closure__'
    }
    
    def __init__(self):
        self.violations: List[str] = []
    
    def visit_Call(self, node):
        """检查函数调用"""
        if isinstance(node.func, ast.Name):
            if node.func.id in self.DANGEROUS_FUNCTIONS:
                self.violations.append(f"Dangerous function call: {node.func.id}")
        elif isinstance(node.func, ast.Attribute):
            if node.func.attr in self.DANGEROUS_FUNCTIONS:
                self.violations.append(f"Dangerous method call: {node.func.attr}")
        self.generic_visit(node)
    
    def visit_Import(self, node):
        """检查导入语句"""
        for alias in node.names:
            module = alias.name.split('.')[0]
            if module in self.DANGEROUS_MODULES:
                self.violations.append(f"Dangerous import: {module}")
        self.generic_visit(node)
    
    def visit_ImportFrom(self, node):
        """检查from导入"""
        if node.module:
            module = node.module.split('.')[0]
            if module in self.DANGEROUS_MODULES:
                self.violations.append(f"Dangerous import from: {module}")
        self.generic_visit(node)
    
    def visit_Attribute(self, node):
        """检查属性访问"""
        if node.attr in self.DANGEROUS_ATTRIBUTES:
            self.violations.append(f"Dangerous attribute access: {node.attr}")
        self.generic_visit(node)

class SecureSandbox:
    """安全沙箱执行环境"""
    
    # 允许的内置函数白名单
    ALLOWED_BUILTINS = {
        'abs', 'all', 'any', 'ascii', 'bin', 'bool', 'bytearray',
        'bytes', 'callable', 'chr', 'classmethod', 'complex',
        'dict', 'dir', 'divmod', 'enumerate', 'filter', 'float',
        'format', 'frozenset', 'hasattr', 'hash', 'hex', 'id',
        'int', 'isinstance', 'issubclass', 'iter', 'len', 'list',
        'map', 'max', 'memoryview', 'min', 'next', 'object',
        'oct', 'ord', 'pow', 'print', 'property', 'range',
        'repr', 'reversed', 'round', 'set', 'setattr', 'slice',
        'sorted', 'staticmethod', 'str', 'sum', 'super', 'tuple',
        'type', 'vars', 'zip', 'True', 'False', 'None'
    }
    
    def __init__(self):
        self.analyzer = CodeAnalyzer()
    
    def analyze(self, code: str) -> List[str]:
        """分析代码安全性"""
        try:
            tree = ast.parse(code)
            self.analyzer.violations = []
            self.analyzer.visit(tree)
            return self.analyzer.violations
        except SyntaxError as e:
            return [f"Syntax error: {e}"]
    
    def create_restricted_globals(self) -> dict:
        """创建受限的全局命名空间"""
        restricted_builtins = {
            name: getattr(builtins, name)
            for name in self.ALLOWED_BUILTINS
            if hasattr(builtins, name)
        }
        
        return {
            '__builtins__': restricted_builtins,
            '__name__': '__main__'
        }
    
    def execute(self, code: str, inputs: dict = None) -> dict:
        """在安全环境中执行代码"""
        # 1. 静态分析
        violations = self.analyze(code)
        if violations:
            raise SecurityError(f"Security violations: {violations}")
        
        # 2. 创建受限环境
        globals_dict = self.create_restricted_globals()
        if inputs:
            globals_dict['inputs'] = inputs
        
        # 3. 在gVisor容器中执行
        return self._execute_in_container(code, globals_dict)
    
    def _execute_in_container(self, code: str, globals_dict: dict) -> dict:
        """在容器中执行代码"""
        # 实际实现会调用Docker/gVisor
        pass
```

### 6.2 资源耗尽攻击防护验证

#### 6.2.1 攻击测试用例

| 用例ID | 攻击类型 | 攻击代码 | 预期防护 |
|-------|---------|---------|---------|
| DOS-001 | CPU耗尽 | `while True: pass` | CPU限制生效，不影响其他任务 |
| DOS-002 | 内存耗尽 | `a = [0] * (1024**3)` | OOM触发，进程被杀死 |
| DOS-003 | 磁盘耗尽 | 持续写入大文件 | 磁盘配额限制，写入失败 |
| DOS-004 | fork炸弹 | `os.fork()`循环 | fork被禁止或限制 |
| DOS-005 | 线程炸弹 | 创建大量线程 | 线程数限制生效 |
| DOS-006 | 递归耗尽 | 无限递归 | 递归深度限制，栈溢出保护 |

### 6.3 敏感数据访问防护验证

#### 6.3.1 攻击测试用例

| 用例ID | 攻击类型 | 攻击目标 | 预期防护 |
|-------|---------|---------|---------|
| DATA-001 | 环境变量泄露 | `os.environ` | 敏感变量被过滤 |
| DATA-002 | 密钥文件访问 | `/run/secrets/*` | 密钥文件不可访问 |
| DATA-003 | 数据库访问 | 内部数据库连接 | 网络隔离，无法连接 |
| DATA-004 | 日志泄露 | 读取其他用户日志 | 日志隔离，只能读自己的 |
| DATA-005 | 内存扫描 | 读取其他进程内存 | 进程隔离，无法访问 |
| DATA-006 | 容器逃逸 | 访问宿主机资源 | gVisor隔离，无法逃逸 |

---

## 7. POC执行计划

### 7.1 测试脚本设计

#### 7.1.1 测试脚本清单

| 脚本名称 | 用途 | 验证目标 | 复杂度 |
|---------|------|---------|-------|
| test_basic_execution.py | 基础执行测试 | PY-001~003 | 低 |
| test_geometry.py | 几何计算测试 | PY-004 | 中 |
| test_file_io.py | 文件IO测试 | PY-005 | 低 |
| test_exception.py | 异常处理测试 | PY-006 | 低 |
| test_isolation.py | 隔离验证 | ISO-001~006 | 高 |
| test_resource_limits.py | 资源限制测试 | RES-001~006 | 高 |
| test_security.py | 安全测试 | SEC-001~008 | 高 |
| test_workflow_simple.py | 简单工作流 | WF-001~002 | 中 |
| test_workflow_complex.py | 复杂工作流 | WF-003~008 | 高 |
| test_completion.py | 代码补全测试 | COMP-001~006 | 中 |
| test_debug.py | 调试功能测试 | DBG-001~007 | 中 |
| test_version.py | 版本管理测试 | VER-001~006 | 中 |
| test_pool.py | 预热池测试 | POOL-001~006 | 高 |
| test_incremental.py | 增量计算测试 | INC-001~006 | 高 |
| test_cache.py | 缓存测试 | CACHE-001~006 | 中 |

### 7.2 测试场景设计

#### 7.2.1 场景矩阵

```
┌─────────────────────────────────────────────────────────────────┐
│                      测试场景矩阵                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  场景1: 基础脚本执行                                             │
│  ├─ 输入: 简单Python脚本                                         │
│  ├─ 负载: 单用户，顺序执行                                        │
│  ├─ 验证: 执行正确性，输出格式                                    │
│  └─ 预期: 100%成功率，平均执行时间<2s                             │
│                                                                 │
│  场景2: 并发脚本执行                                             │
│  ├─ 输入: 10个不同的Python脚本                                    │
│  ├─ 负载: 10并发用户                                             │
│  ├─ 验证: 隔离性，资源分配                                        │
│  └─ 预期: 无相互干扰，资源限制生效                                │
│                                                                 │
│  场景3: 复杂工作流执行                                           │
│  ├─ 输入: 包含10个节点的设计工作流                                │
│  ├─ 负载: 5个并行工作流                                          │
│  ├─ 验证: 依赖解析，失败重试                                      │
│  └─ 预期: 依赖正确，失败自动重试                                  │
│                                                                 │
│  场景4: 安全攻击模拟                                             │
│  ├─ 输入: 各类恶意代码                                           │
│  ├─ 负载: 单攻击脚本                                             │
│  ├─ 验证: 攻击被拦截，系统安全                                    │
│  └─ 预期: 100%攻击拦截，无系统影响                                │
│                                                                 │
│  场景5: 长时间运行测试                                           │
│  ├─ 输入: 执行时间5分钟的脚本                                     │
│  ├─ 负载: 单长时间任务                                           │
│  ├─ 验证: 状态持久化，超时处理                                    │
│  └─ 预期: 状态正确持久化，超时正常终止                            │
│                                                                 │
│  场景6: 高并发压力测试                                           │
│  ├─ 输入: 标准设计脚本                                           │
│  ├─ 负载: 100并发用户，持续10分钟                                 │
│  ├─ 验证: 系统稳定性，性能指标                                    │
│  └─ 预期: 成功率>99%，P99延迟<5s                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.3 验收标准

#### 7.3.1 功能验收标准

| 验证项 | 验收标准 | 优先级 |
|-------|---------|-------|
| 脚本执行 | 标准Python脚本100%成功执行 | P0 |
| 沙箱隔离 | 所有隔离测试用例通过 | P0 |
| 资源限制 | CPU/内存/时间限制100%生效 | P0 |
| 工作流编排 | 复杂依赖工作流正确执行 | P0 |
| 失败重试 | 临时失败自动重试成功 | P0 |
| 代码补全 | 补全延迟<200ms | P1 |
| 调试功能 | 断点、单步、变量查看正常 | P1 |
| 版本管理 | 版本创建、比较、回滚正常 | P1 |
| 预热池 | 二次执行提速>80% | P1 |
| 增量计算 | 未变化部分不重算 | P1 |
| 安全防护 | 所有攻击测试用例被拦截 | P0 |

#### 7.3.2 性能验收标准

| 指标 | 目标值 | 优先级 |
|------|-------|-------|
| 脚本冷启动时间 | <5秒 | P0 |
| 脚本预热启动时间 | <1秒 | P0 |
| 并发执行数 | 支持100并发 | P0 |
| 代码补全延迟 | P99 <200ms | P1 |
| 工作流执行成功率 | >99.9% | P0 |
| 缓存命中率 | >80% | P1 |
| 系统资源占用 | CPU<50%, 内存<4GB | P1 |

### 7.4 风险缓解方案

#### 7.4.1 风险识别与缓解

| 风险ID | 风险描述 | 可能性 | 影响 | 缓解措施 |
|-------|---------|-------|------|---------|
| R-001 | gVisor性能开销过大 | 中 | 高 | 1.性能基准测试 2.考虑Kata Containers替代 3.优化资源配额 |
| R-002 | Temporal学习曲线陡峭 | 高 | 中 | 1.团队培训 2.原型验证 3.文档完善 |
| R-003 | CodeMirror 6集成复杂 | 中 | 中 | 1.分阶段集成 2.使用成熟插件 3.LSP简化 |
| R-004 | 安全沙箱被绕过 | 低 | 极高 | 1.多层防护 2.安全审计 3.漏洞赏金计划 |
| R-005 | 预热池资源浪费 | 中 | 低 | 1.动态扩缩容 2.空闲超时回收 3.使用指标监控 |
| R-006 | 缓存一致性难保证 | 中 | 中 | 1.版本号机制 2.缓存失效策略 3.最终一致性 |
| R-007 | 依赖管理复杂 | 高 | 中 | 1.可视化依赖图 2.自动检测循环 3.依赖版本锁定 |
| R-008 | 调试功能实现困难 | 中 | 低 | 1.使用成熟方案(debugpy) 2.简化调试需求 3.日志替代 |

---

## 8. 风险评估与缓解

### 8.1 技术风险

#### 8.1.1 风险矩阵

```
                    影响程度
              低      中      高      极高
           ┌─────┬─────┬─────┬─────┐
        高 │ R005│ R002│ R007│     │
           ├─────┼─────┼─────┼─────┤
可能性  中 │     │ R003│ R001│     │
           ├─────┼─────┼─────┼─────┤
        低 │     │     │ R006│ R004│
           ├─────┼─────┼─────┼─────┤
        极低│     │     │     │     │
           └─────┴─────┴─────┴─────┘
```

### 8.2 缓解策略

#### 8.2.1 技术备选方案

| 组件 | 首选方案 | 备选方案1 | 备选方案2 |
|------|---------|----------|----------|
| 沙箱运行时 | gVisor | Kata Containers | Firecracker |
| 任务调度 | Temporal | Airflow | Cadence |
| 代码编辑器 | CodeMirror 6 | Monaco Editor | Ace |
| 缓存系统 | Redis | Memcached | 本地内存 |
| 持久化存储 | PostgreSQL | MySQL | SQLite |

---

## 9. 结论与建议

### 9.1 POC验证结论

基于以上验证方案，预期可以得出以下结论：

1. **技术可行性**: 推荐技术栈（Python 3.11+ / Docker + gVisor / Temporal / CodeMirror 6）在理论上是可行的，各组件都有成熟的社区支持和生产环境验证。

2. **性能预期**: 通过预热池和增量计算优化，预期可以达到：
   - 冷启动 < 5秒
   - 预热启动 < 1秒
   - 二次执行提速 > 80%

3. **安全预期**: 多层安全防护（静态分析 + gVisor隔离 + 资源限制）可以有效防止：
   - 代码注入攻击
   - 资源耗尽攻击
   - 敏感数据泄露

### 9.2 实施建议

#### 9.2.1 分阶段实施

```
Phase 1 (2周): 基础执行验证
├─ Python脚本执行
├─ gVisor基础隔离
└─ 资源限制基础验证

Phase 2 (2周): 任务调度验证
├─ Temporal工作流
├─ 依赖管理
└─ 失败重试

Phase 3 (2周): 开发环境验证
├─ CodeMirror 6集成
├─ 代码补全
└─ 调试功能

Phase 4 (2周): 性能与安全验证
├─ 预热池
├─ 增量计算
├─ 缓存系统
└─ 安全测试

Phase 5 (1周): 集成测试与优化
├─ 端到端测试
├─ 性能调优
└─ 文档完善
```

#### 9.2.2 关键成功因素

1. **团队培训**: 确保团队熟悉Temporal和gVisor的使用
2. **原型先行**: 每个组件先做小规模POC验证
3. **持续监控**: 建立完善的监控和告警机制
4. **安全优先**: 安全测试贯穿整个开发周期
5. **性能基准**: 建立性能基准，持续优化

### 9.3 下一步行动

1. **立即启动**: Phase 1基础执行验证
2. **环境准备**: 搭建POC测试环境
3. **团队分工**: 分配各模块验证责任人
4. **进度跟踪**: 每周POC进度评审
5. **风险升级**: 遇到阻塞问题及时升级

---

## 附录

### A. 测试环境配置

```yaml
# docker-compose.poc.yml
version: '3.8'

services:
  temporal-server:
    image: temporalio/auto-setup:1.22
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgresql
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
    ports:
      - "7233:7233"
    depends_on:
      - postgresql

  postgresql:
    image: postgres:15
    environment:
      POSTGRES_USER: temporal
      POSTGRES_PASSWORD: temporal
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  script-executor:
    build:
      context: ./executor
      dockerfile: Dockerfile
    runtime: runsc
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock

volumes:
  postgres_data:
```

### B. 测试数据准备

```python
# test_data.py
"""测试数据生成"""

SAMPLE_SCRIPTS = {
    "basic_math": """
result = sum(range(100))
print(f"Sum: {result}")
""",
    "geometry_test": """
from shapely.geometry import Polygon, Point

# 创建建筑平面
footprint = Polygon([(0,0), (20,0), (20,15), (0,15)])
area = footprint.area
perimeter = footprint.length

print(f"Area: {area}, Perimeter: {perimeter}")
""",
    "file_io_test": """
import json

# 写入临时文件
data = {"building": {"floors": 10, "area": 5000}}
with open('/tmp/test.json', 'w') as f:
    json.dump(data, f)

# 读取文件
with open('/tmp/test.json', 'r') as f:
    loaded = json.load(f)

print(f"Data: {loaded}")
"""
}

MALICIOUS_SCRIPTS = {
    "system_command": "import os; os.system('ls -la /')",
    "file_access": "open('/etc/passwd').read()",
    "network_access": "import socket; s = socket.socket(); s.connect(('8.8.8.8', 53))",
    "eval_attack": "eval('__import__(\"os\").system(\"id\")')",
    "fork_bomb": "import os\nwhile True: os.fork()",
    "memory_bomb": "a = [0] * (1024**3)"
}
```

---

**文档结束**

*本报告由脚本引擎架构师编写，用于半自动化建筑设计平台可行性验证阶段评审。*
