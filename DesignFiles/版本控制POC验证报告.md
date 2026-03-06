# 可行性验证阶段 - 版本控制POC验证报告

## 半自动化建筑设计平台

---

**文档版本**: v1.0  
**编制日期**: 2024年  
**文档状态**: 可行性验证阶段  

---

## 目录

1. [概述](#1-概述)
2. [撤销重做POC](#2-撤销重做poc)
3. [历史版本管理POC](#3-历史版本管理poc)
4. [权限控制POC](#4-权限控制poc)
5. [账号隔离验证](#5-账号隔离验证)
6. [审计追踪POC](#6-审计追踪poc)
7. [POC执行计划](#7-poc执行计划)
8. [附录](#8-附录)

---

## 1. 概述

### 1.1 验证目标

本POC验证旨在确认推荐技术栈在半自动化建筑设计平台中的技术可行性，验证以下核心能力：

| 验证领域 | 验证目标 | 技术栈 |
|---------|---------|--------|
| 撤销重做 | 支持无限级撤销/重做，批量操作 | 命令模式 + 操作日志 |
| 版本管理 | 高效存储历史版本，快速回滚 | 全量快照 + 增量变更 |
| 权限控制 | 细粒度RBAC权限，数据隔离 | Casbin + 行级隔离 |
| 账号隔离 | 多租户安全隔离 | Keycloak + OAuth2.0 |
| 审计追踪 | 完整操作日志，可追溯 | 审计日志系统 |

### 1.2 验证范围

```
┌─────────────────────────────────────────────────────────────────┐
│                      版本控制系统POC验证范围                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  撤销重做   │  │  版本管理   │  │  权限控制   │             │
│  │  模块POC   │  │  模块POC   │  │  模块POC   │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│  ┌─────────────┐  ┌─────────────┐                              │
│  │  账号隔离   │  │  审计追踪   │                              │
│  │  验证POC   │  │  模块POC   │                              │
│  └─────────────┘  └─────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 验证环境

| 环境项 | 配置 |
|-------|------|
| 后端框架 | Spring Boot 3.x / Node.js 18+ |
| 数据库 | PostgreSQL 15+ (主库) + Redis 7+ (缓存) |
| 权限引擎 | Casbin 2.x |
| 身份认证 | Keycloak 22+ |
| 消息队列 | RabbitMQ / Kafka |
| 存储 | MinIO (对象存储) |

---

## 2. 撤销重做POC

### 2.1 命令模式实现验证

#### 2.1.1 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                      命令模式架构图                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────────┐                                              │
│   │   Client     │  (UI/Controller)                             │
│   └──────┬───────┘                                              │
│          │ execute()                                            │
│          ▼                                                      │
│   ┌──────────────┐         ┌─────────────────────────────┐     │
│   │  Invoker     │◄───────│      Command History        │     │
│   │  (History    │ 管理    │  ┌─────────┐  ┌─────────┐  │     │
│   │   Manager)   │         │  │ Undo栈  │  │ Redo栈  │  │     │
│   └──────┬───────┘         │  │ [Cmd3]  │  │         │  │     │
│          │                 │  │ [Cmd2]  │  │         │  │     │
│          │ execute()       │  │ [Cmd1]  │  │         │  │     │
│          ▼                 │  └─────────┘  └─────────┘  │     │
│   ┌──────────────┐         └─────────────────────────────┘     │
│   │   Command    │◄──────────────┐                              │
│   │  (Interface) │               │                              │
│   │  +execute()  │               │                              │
│   │  +undo()     │               │                              │
│   │  +redo()     │               │                              │
│   └──────┬───────┘               │                              │
│          │                       │                              │
│    ┌─────┴─────┬────────────────┘                              │
│    │           │                                               │
│    ▼           ▼                                               │
│ ┌──────┐  ┌────────┐  ┌────────┐  ┌────────┐                  │
│ │Create│  │ Update │  │ Delete │  │ Batch  │  Concrete         │
│ │Cmd   │  │  Cmd   │  │  Cmd   │  │  Cmd   │  Commands         │
│ └──────┘  └────────┘  └────────┘  └────────┘                  │
│    │           │           │           │                       │
│    └───────────┴───────────┴───────────┘                       │
│                    │                                            │
│                    ▼                                            │
│            ┌──────────────┐                                    │
│            │   Receiver   │  (业务对象/模型)                    │
│            │  (Element    │                                    │
│            │   Model)     │                                    │
│            └──────────────┘                                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 2.1.2 核心接口定义

```typescript
// 命令接口定义
interface ICommand {
  readonly id: string;
  readonly type: CommandType;
  readonly timestamp: number;
  readonly userId: string;
  readonly projectId: string;
  
  // 执行命令
  execute(): Promise<CommandResult>;
  
  // 撤销命令
  undo(): Promise<CommandResult>;
  
  // 重做命令（默认调用execute，可覆盖）
  redo(): Promise<CommandResult>;
  
  // 获取命令描述（用于UI显示）
  getDescription(): string;
  
  // 序列化为JSON（用于持久化）
  serialize(): CommandSnapshot;
  
  // 从JSON反序列化
  deserialize(snapshot: CommandSnapshot): ICommand;
}

// 命令执行结果
interface CommandResult {
  success: boolean;
  data?: any;
  error?: Error;
  affectedElements?: string[];
}

// 命令类型枚举
enum CommandType {
  CREATE_ELEMENT = 'CREATE_ELEMENT',
  UPDATE_ELEMENT = 'UPDATE_ELEMENT',
  DELETE_ELEMENT = 'DELETE_ELEMENT',
  MOVE_ELEMENT = 'MOVE_ELEMENT',
  RESIZE_ELEMENT = 'RESIZE_ELEMENT',
  BATCH_OPERATION = 'BATCH_OPERATION',
  PROPERTY_CHANGE = 'PROPERTY_CHANGE',
  STYLE_CHANGE = 'STYLE_CHANGE'
}
```

#### 2.1.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| CMD-001 | 创建元素命令执行 | 1. 创建CreateElementCommand<br>2. 调用execute()<br>3. 验证元素创建 | 元素成功创建，命令入Undo栈 |
| CMD-002 | 更新元素命令执行 | 1. 创建UpdateElementCommand<br>2. 调用execute()<br>3. 验证元素更新 | 元素属性更新，命令入Undo栈 |
| CMD-003 | 删除元素命令执行 | 1. 创建DeleteElementCommand<br>2. 调用execute()<br>3. 验证元素删除 | 元素被标记删除，命令入Undo栈 |
| CMD-004 | 命令撤销操作 | 1. 执行命令<br>2. 调用undo()<br>3. 验证状态恢复 | 状态恢复到执行前，命令移入Redo栈 |
| CMD-005 | 命令重做操作 | 1. 执行并撤销命令<br>2. 调用redo()<br>3. 验证状态恢复 | 状态恢复到撤销前 |
| CMD-006 | 命令序列化 | 1. 创建命令<br>2. 调用serialize()<br>3. 调用deserialize() | 序列化和反序列化结果一致 |

### 2.2 操作日志存储验证

#### 2.2.1 存储架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    操作日志存储架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│   │   内存缓存   │    │   持久化层   │    │   归档存储   │     │
│   │  (Redis)    │    │ (PostgreSQL) │    │  (MinIO)    │     │
│   │             │    │              │    │             │     │
│   │ ┌────────┐  │    │ ┌──────────┐ │    │ ┌────────┐  │     │
│   │ │活跃操作│  │───►│ │操作日志表│ │───►│ │历史归档│  │     │
│   │ │队列   │  │    │ │          │ │    │ │文件   │  │     │
│   │ └────────┘  │    │ └──────────┘ │    │ └────────┘  │     │
│   │             │    │              │    │             │     │
│   │ TTL: 1小时  │    │ 保留: 90天   │    │ 永久保留   │     │
│   └──────────────┘    └──────────────┘    └──────────────┘     │
│                                                                 │
│   存储策略:                                                     │
│   - 热数据: Redis (最近1小时操作)                               │
│   - 温数据: PostgreSQL (最近90天)                              │
│   - 冷数据: MinIO (90天前归档)                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 2.2.2 数据库表结构

```sql
-- 操作日志主表
CREATE TABLE operation_logs (
    id BIGSERIAL PRIMARY KEY,
    command_id VARCHAR(64) NOT NULL UNIQUE,
    command_type VARCHAR(50) NOT NULL,
    project_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    session_id VARCHAR(64),
    
    -- 操作前状态快照
    before_state JSONB,
    
    -- 操作后状态快照
    after_state JSONB,
    
    -- 变更详情 (增量)
    changes JSONB NOT NULL,
    
    -- 操作元数据
    metadata JSONB,
    
    -- 执行结果
    result_status VARCHAR(20) NOT NULL,
    result_message TEXT,
    
    -- 时间戳
    executed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    duration_ms INTEGER,
    
    -- 索引
    CONSTRAINT idx_op_logs_project_time 
        UNIQUE (project_id, executed_at, id)
);

-- 创建索引
CREATE INDEX idx_op_logs_user ON operation_logs(user_id, executed_at DESC);
CREATE INDEX idx_op_logs_command_type ON operation_logs(command_type);
CREATE INDEX idx_op_logs_session ON operation_logs(session_id);

-- 分区表 (按项目ID+时间分区)
CREATE TABLE operation_logs_2024_q1 PARTITION OF operation_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');
```

#### 2.2.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| LOG-001 | 操作日志写入 | 1. 执行操作命令<br>2. 检查日志表记录 | 日志正确记录，包含完整变更信息 |
| LOG-002 | 操作日志查询 | 1. 按项目ID查询<br>2. 按时间范围查询<br>3. 按用户查询 | 查询结果准确，性能<100ms |
| LOG-003 | 批量日志写入 | 1. 批量执行1000个命令<br>2. 监控写入性能 | 写入TPS > 500 |
| LOG-004 | 日志归档 | 1. 模拟90天前数据<br>2. 触发归档任务<br>3. 验证归档结果 | 数据正确归档，原表数据清理 |
| LOG-005 | 日志压缩 | 1. 生成大量操作日志<br>2. 启用压缩<br>3. 验证存储空间 | 压缩率 > 60% |

### 2.3 撤销栈/重做栈管理验证

#### 2.3.1 栈管理架构

```
┌─────────────────────────────────────────────────────────────────┐
│                   撤销/重做栈管理架构                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    HistoryManager                        │  │
│   │  ┌─────────────────┐        ┌─────────────────┐        │  │
│   │  │    Undo Stack   │        │    Redo Stack   │        │  │
│   │  │  ┌───────────┐  │        │  ┌───────────┐  │        │  │
│   │  │  │  [Top]    │  │        │  │  [Top]    │  │        │  │
│   │  │  │ Command N │  │◄──────►│  │ Command N │  │        │  │
│   │  │  │ Command N-1│  │  undo  │  │ Command N-1│  │        │  │
│   │  │  │    ...    │  │        │  │    ...    │  │        │  │
│   │  │  │ Command 1 │  │  redo  │  │ Command 1 │  │        │  │
│   │  │  │ [Bottom]  │  │───────►│  │ [Bottom]  │  │        │  │
│   │  │  └───────────┘  │        │  └───────────┘  │        │  │
│   │  │  Max: 1000     │        │  Max: 1000     │        │  │
│   │  └─────────────────┘        └─────────────────┘        │  │
│   └─────────────────────────────────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    栈操作状态机                          │  │
│   │                                                         │  │
│   │    ┌─────────┐    execute    ┌─────────┐               │  │
│   │    │  IDLE   │──────────────►│ EXECUTED│               │  │
│   │    └────┬────┘               └────┬────┘               │  │
│   │         │                         │                    │  │
│   │         │ undo                    │ undo               │  │
│   │         ▼                         ▼                    │  │
│   │    ┌─────────┐               ┌─────────┐               │  │
│   │    │ UNDONE  │◄──────────────┤         │               │  │
│   │    └────┬────┘               └─────────┘               │  │
│   │         │                                              │  │
│   │         │ redo                                         │  │
│   │         ▼                                              │  │
│   │    ┌─────────┐                                          │  │
│   │    │REDONE   │                                          │  │
│   │    └─────────┘                                          │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 2.3.2 核心算法实现

```typescript
class HistoryManager {
  private undoStack: ICommand[] = [];
  private redoStack: ICommand[] = [];
  private maxStackSize: number = 1000;
  private currentGroup: string | null = null;
  private groupedCommands: ICommand[] = [];
  
  // 执行命令
  async execute(command: ICommand): Promise<CommandResult> {
    const result = await command.execute();
    
    if (result.success) {
      // 清空重做栈（新操作后重做栈失效）
      this.redoStack = [];
      
      // 检查是否在分组操作中
      if (this.currentGroup) {
        this.groupedCommands.push(command);
      } else {
        this.pushToUndoStack(command);
      }
      
      // 持久化操作日志
      await this.persistOperation(command, result);
    }
    
    return result;
  }
  
  // 撤销操作
  async undo(): Promise<CommandResult | null> {
    if (this.undoStack.length === 0) {
      return null;
    }
    
    const command = this.undoStack.pop()!;
    const result = await command.undo();
    
    if (result.success) {
      this.redoStack.push(command);
    } else {
      // 撤销失败，恢复命令到Undo栈
      this.undoStack.push(command);
    }
    
    return result;
  }
  
  // 重做操作
  async redo(): Promise<CommandResult | null> {
    if (this.redoStack.length === 0) {
      return null;
    }
    
    const command = this.redoStack.pop()!;
    const result = await command.redo();
    
    if (result.success) {
      this.undoStack.push(command);
    } else {
      // 重做失败，恢复命令到Redo栈
      this.redoStack.push(command);
    }
    
    return result;
  }
  
  // 开始批量操作分组
  beginGroup(groupId: string): void {
    this.currentGroup = groupId;
    this.groupedCommands = [];
  }
  
  // 结束批量操作分组
  endGroup(): void {
    if (this.groupedCommands.length > 0) {
      const batchCommand = new BatchCommand(
        this.currentGroup!,
        this.groupedCommands
      );
      this.pushToUndoStack(batchCommand);
    }
    this.currentGroup = null;
    this.groupedCommands = [];
  }
  
  // 压入Undo栈（带容量限制）
  private pushToUndoStack(command: ICommand): void {
    if (this.undoStack.length >= this.maxStackSize) {
      // 移除最旧的命令并归档
      const oldCommand = this.undoStack.shift()!;
      this.archiveCommand(oldCommand);
    }
    this.undoStack.push(command);
  }
  
  // 获取当前状态
  getState(): HistoryState {
    return {
      canUndo: this.undoStack.length > 0,
      canRedo: this.redoStack.length > 0,
      undoCount: this.undoStack.length,
      redoCount: this.redoStack.length,
      undoDescriptions: this.undoStack.map(c => c.getDescription()),
      redoDescriptions: this.redoStack.map(c => c.getDescription())
    };
  }
}
```

#### 2.3.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| STACK-001 | 基本撤销重做 | 1. 执行3个命令<br>2. 撤销2次<br>3. 重做1次 | Undo栈:1, Redo栈:1 |
| STACK-002 | 栈容量限制 | 1. 执行1001个命令 | 最旧命令被归档，Undo栈保持1000 |
| STACK-003 | 新操作清空Redo栈 | 1. 执行命令A<br>2. 撤销<br>3. 执行命令B | Redo栈被清空 |
| STACK-004 | 撤销失败恢复 | 1. 执行命令<br>2. 模拟撤销失败<br>3. 验证栈状态 | 命令保留在Undo栈 |
| STACK-005 | 状态查询 | 1. 执行多个命令<br>2. 调用getState() | 返回准确的栈状态信息 |

### 2.4 批量操作撤销验证

#### 2.4.1 批量命令架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    批量操作命令架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    BatchCommand                          │  │
│   │  (组合模式实现批量操作)                                   │  │
│   │                                                         │  │
│   │  ┌─────────────────────────────────────────────────┐   │  │
│   │  │  execute(): Promise<CommandResult>              │   │  │
│   │  │  {                                              │   │  │
│   │  │    // 事务性执行所有子命令                       │   │  │
│   │  │    for (cmd of subCommands) {                   │   │  │
│   │  │      result = await cmd.execute();              │   │  │
│   │  │      if (!result.success) {                     │   │  │
│   │  │        await this.rollback(executed);           │   │  │
│   │  │        return failure;                          │   │  │
│   │  │      }                                          │   │  │
│   │  │      executed.push(cmd);                        │   │  │
│   │  │    }                                            │   │  │
│   │  │    return success;                              │   │  │
│   │  │  }                                              │   │  │
│   │  │                                                 │   │  │
│   │  │  undo(): Promise<CommandResult>                 │   │  │
│   │  │  {                                              │   │  │
│   │  │    // 逆序撤销所有子命令                         │   │  │
│   │  │    for (cmd of reverse(subCommands)) {          │   │  │
│   │  │      await cmd.undo();                          │   │  │
│   │  │    }                                            │   │  │
│   │  │  }                                              │   │  │
│   │  └─────────────────────────────────────────────────┘   │  │
│   │                                                         │  │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │  │
│   │  │ SubCmd1 │  │ SubCmd2 │  │ SubCmd3 │  │ SubCmdN │   │  │
│   │  │(Create) │  │(Update) │  │(Move)   │  │(Delete) │   │  │
│   │  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   批量操作场景:                                                  │
│   - 多选元素批量移动                                             │
│   - 批量属性修改                                                 │
│   - 复制粘贴多个元素                                             │
│   - 批量删除                                                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 2.4.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| BATCH-001 | 批量创建撤销 | 1. 批量创建10个元素<br>2. 执行撤销 | 所有元素被删除 |
| BATCH-002 | 批量更新撤销 | 1. 批量更新5个元素属性<br>2. 执行撤销 | 所有元素属性恢复 |
| BATCH-003 | 批量操作原子性 | 1. 批量操作(模拟中间失败)<br>2. 验证状态 | 所有操作回滚，状态一致 |
| BATCH-004 | 批量操作重做 | 1. 批量操作<br>2. 撤销<br>3. 重做 | 批量操作重新执行 |
| BATCH-005 | 嵌套批量操作 | 1. 批量操作内包含子批量<br>2. 执行撤销 | 正确撤销所有层级 |

---

## 3. 历史版本管理POC

### 3.1 快照生成验证

#### 3.1.1 快照架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    快照生成架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    快照生成流程                          │  │
│   │                                                         │  │
│   │   ┌─────────┐    ┌─────────┐    ┌─────────┐            │  │
│   │   │ 触发条件 │───►│ 收集数据 │───►│ 序列化  │            │  │
│   │   └─────────┘    └─────────┘    └────┬────┘            │  │
│   │                                      │                  │  │
│   │   ┌─────────┐    ┌─────────┐    ┌────▼────┐            │  │
│   │   │ 完成通知 │◄───│ 存储快照 │◄───│ 压缩优化 │            │  │
│   │   └─────────┘    └─────────┘    └─────────┘            │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   触发条件:                                                      │
│   - 手动保存 (Ctrl+S)                                           │
│   - 自动保存 (每5分钟)                                          │
│   - 关键操作后 (导入/导出完成)                                   │
│   - 用户登出前                                                  │
│                                                                 │
│   快照内容:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  {                                                      │  │
│   │    "version": "1.0",                                    │  │
│   │    "projectId": "proj_123",                             │  │
│   │    "snapshotId": "snap_456",                            │  │
│   │    "timestamp": "2024-01-15T10:30:00Z",                 │  │
│   │    "createdBy": "user_789",                             │  │
│   │    "description": "自动保存",                            │  │
│   │    "elements": [...],      // 所有元素完整数据          │  │
│   │    "metadata": {...},      // 项目元数据                │  │
│   │    "checksum": "sha256:..." // 完整性校验               │  │
│   │  }                                                      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.1.2 快照存储表结构

```sql
-- 版本快照表
CREATE TABLE project_snapshots (
    id BIGSERIAL PRIMARY KEY,
    snapshot_id VARCHAR(64) NOT NULL UNIQUE,
    project_id VARCHAR(64) NOT NULL,
    version_number INTEGER NOT NULL,
    
    -- 快照类型
    snapshot_type VARCHAR(20) NOT NULL, -- FULL, INCREMENTAL
    
    -- 快照元数据
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    description TEXT,
    tags VARCHAR(50)[],
    
    -- 存储信息
    storage_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    checksum VARCHAR(128) NOT NULL,
    compression_type VARCHAR(20) DEFAULT 'gzip',
    
    -- 统计信息
    element_count INTEGER NOT NULL,
    change_count INTEGER DEFAULT 0,
    
    -- 父快照（增量快照用）
    parent_snapshot_id VARCHAR(64),
    
    -- 是否已归档
    is_archived BOOLEAN DEFAULT FALSE,
    archived_at TIMESTAMP WITH TIME ZONE,
    
    -- 约束
    CONSTRAINT idx_snapshots_project_version 
        UNIQUE (project_id, version_number),
    CONSTRAINT fk_parent_snapshot 
        FOREIGN KEY (parent_snapshot_id) 
        REFERENCES project_snapshots(snapshot_id)
);

-- 创建索引
CREATE INDEX idx_snapshots_project ON project_snapshots(project_id, created_at DESC);
CREATE INDEX idx_snapshots_type ON project_snapshots(snapshot_type);
CREATE INDEX idx_snapshots_created_by ON project_snapshots(created_by);
```

#### 3.1.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| SNAP-001 | 手动快照生成 | 1. 调用createSnapshot()<br>2. 验证快照文件<br>3. 验证数据库记录 | 快照正确生成，数据完整 |
| SNAP-002 | 自动快照生成 | 1. 等待自动保存触发<br>2. 验证快照生成 | 按配置间隔自动生成 |
| SNAP-003 | 大项目快照 | 1. 准备10万元素项目<br>2. 生成快照<br>3. 监控性能 | 生成时间<30s，内存<2GB |
| SNAP-004 | 快照完整性校验 | 1. 生成快照<br>2. 修改快照文件<br>3. 验证校验失败 | 校验失败被检测 |
| SNAP-005 | 并发快照 | 1. 同时触发多个快照请求<br>2. 验证结果 | 排队执行，无数据损坏 |

### 3.2 增量存储验证

#### 3.2.1 增量存储架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    增量存储架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   存储策略: 混合存储 (全量快照 + 增量变更)                        │
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                                                         │  │
│   │   V1 (全量)    V2 (增量)    V3 (增量)    V4 (全量)      │  │
│   │   ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐          │  │
│   │   │█████│◄────│  Δ  │◄────│  Δ  │◄────│█████│          │  │
│   │   │█████│     │  Δ  │     │  Δ  │     │█████│          │  │
│   │   │█████│     └─────┘     └─────┘     │█████│          │  │
│   │   └─────┘                              └─────┘          │  │
│   │    10MB      +0.5MB      +0.3MB       10.8MB            │  │
│   │                                                         │  │
│   │   策略: 每10个增量生成一个全量快照                        │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   增量变更结构:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  {                                                      │  │
│   │    "baseSnapshotId": "snap_v1",                         │  │
│   │    "changes": [                                         │  │
│   │      {                                                  │  │
│   │        "type": "CREATE",                                │  │
│   │        "elementId": "elem_001",                         │  │
│   │        "data": {...}                                    │  │
│   │      },                                                 │  │
│   │      {                                                  │  │
│   │        "type": "UPDATE",                                │  │
│   │        "elementId": "elem_002",                         │  │
│   │        "before": {...},                                 │  │
│   │        "after": {...}                                   │  │
│   │      },                                                 │  │
│   │      {                                                  │  │
│   │        "type": "DELETE",                                │  │
│   │        "elementId": "elem_003",                         │  │
│   │        "before": {...}                                  │  │
│   │      }                                                  │  │
│   │    ]                                                    │  │
│   │  }                                                      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.2.2 差异计算算法

```typescript
// 差异计算服务
class DiffService {
  
  // 计算两个版本之间的差异
  calculateDiff(
    before: ProjectState, 
    after: ProjectState
  ): ChangeSet {
    const changes: Change[] = [];
    
    const beforeMap = new Map(before.elements.map(e => [e.id, e]));
    const afterMap = new Map(after.elements.map(e => [e.id, e]));
    
    // 检测新增元素
    for (const [id, element] of afterMap) {
      if (!beforeMap.has(id)) {
        changes.push({
          type: ChangeType.CREATE,
          elementId: id,
          data: element
        });
      }
    }
    
    // 检测删除元素
    for (const [id, element] of beforeMap) {
      if (!afterMap.has(id)) {
        changes.push({
          type: ChangeType.DELETE,
          elementId: id,
          before: element
        });
      }
    }
    
    // 检测修改元素
    for (const [id, afterElement] of afterMap) {
      const beforeElement = beforeMap.get(id);
      if (beforeElement && !this.isEqual(beforeElement, afterElement)) {
        changes.push({
          type: ChangeType.UPDATE,
          elementId: id,
          before: beforeElement,
          after: afterElement,
          diff: this.calculateObjectDiff(beforeElement, afterElement)
        });
      }
    }
    
    return {
      baseVersion: before.version,
      targetVersion: after.version,
      changes,
      changeCount: changes.length
    };
  }
  
  // 应用变更集到基础版本
  applyDiff(baseState: ProjectState, changeSet: ChangeSet): ProjectState {
    const elementMap = new Map(baseState.elements.map(e => [e.id, e]));
    
    for (const change of changeSet.changes) {
      switch (change.type) {
        case ChangeType.CREATE:
          elementMap.set(change.elementId, change.data);
          break;
        case ChangeType.UPDATE:
          elementMap.set(change.elementId, change.after);
          break;
        case ChangeType.DELETE:
          elementMap.delete(change.elementId);
          break;
      }
    }
    
    return {
      ...baseState,
      version: changeSet.targetVersion,
      elements: Array.from(elementMap.values())
    };
  }
  
  // 压缩连续变更
  compressChanges(changes: Change[]): Change[] {
    const compressed: Change[] = [];
    const elementChanges = new Map<string, Change[]>();
    
    // 按元素分组
    for (const change of changes) {
      if (!elementChanges.has(change.elementId)) {
        elementChanges.set(change.elementId, []);
      }
      elementChanges.get(change.elementId)!.push(change);
    }
    
    // 压缩每个元素的变更链
    for (const [elementId, elementChangeList] of elementChanges) {
      const compressedChange = this.compressElementChanges(elementChangeList);
      if (compressedChange) {
        compressed.push(compressedChange);
      }
    }
    
    return compressed;
  }
}
```

#### 3.2.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| DIFF-001 | 差异计算 | 1. 准备两个版本<br>2. 计算差异<br>3. 验证结果 | 正确识别增删改 |
| DIFF-002 | 差异应用 | 1. 应用差异到基础版<br>2. 验证结果 | 结果与目标版本一致 |
| DIFF-003 | 差异压缩 | 1. 生成连续变更链<br>2. 压缩差异<br>3. 验证结果 | 压缩后功能等价 |
| DIFF-004 | 增量存储空间 | 1. 对比全量vs增量存储 | 增量存储节省>70% |
| DIFF-005 | 增量链恢复 | 1. 从增量链恢复版本<br>2. 验证性能 | 10个增量恢复<5s |

### 3.3 版本对比验证

#### 3.3.1 版本对比架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    版本对比架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    版本对比流程                          │  │
│   │                                                         │  │
│   │   ┌─────────┐     ┌─────────┐     ┌─────────┐          │  │
│   │   │ Version │     │ Version │     │  Diff   │          │  │
│   │   │   A     │────►│   B     │────►│ Engine  │          │  │
│   │   └─────────┘     └─────────┘     └────┬────┘          │  │
│   │                                        │                │  │
│   │                    ┌───────────────────┘                │  │
│   │                    ▼                                    │  │
│   │   ┌─────────┐     ┌─────────┐     ┌─────────┐          │  │
│   │   │ 统计    │◄────│ 结构化  │◄────│ 原始    │          │  │
│   │   │ 报告    │     │ 差异    │     │ 差异    │          │  │
│   │   └─────────┘     └─────────┘     └─────────┘          │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   对比结果结构:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  {                                                      │  │
│   │    "versionA": "v1.0",                                  │  │
│   │    "versionB": "v1.5",                                  │  │
│   │    "summary": {                                         │  │
│   │      "totalChanges": 150,                               │  │
│   │      "created": 20,                                     │  │
│   │      "updated": 100,                                    │  │
│   │      "deleted": 30                                      │  │
│   │    },                                                   │  │
│   │    "changesByType": {                                   │  │
│   │      "wall": 50, "door": 30, "window": 20, ...          │  │
│   │    },                                                   │  │
│   │    "detailedChanges": [...],                            │  │
│   │    "visualDiff": {                                      │  │
│   │      "addedElements": [...],                            │  │
│   │      "removedElements": [...],                          │  │
│   │      "modifiedElements": [...]                          │  │
│   │    }                                                    │  │
│   │  }                                                      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.3.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| COMP-001 | 版本对比基础 | 1. 选择两个版本<br>2. 执行对比<br>3. 验证结果 | 正确显示差异统计 |
| COMP-002 | 可视化对比 | 1. 对比两个版本<br>2. 查看可视化差异 | 新增/删除/修改元素正确高亮 |
| COMP-003 | 属性级对比 | 1. 对比元素属性变化<br>2. 查看详细差异 | 显示具体属性变更 |
| COMP-004 | 大版本对比 | 1. 对比1000+变更版本<br>2. 监控性能 | 对比完成<10s |
| COMP-005 | 跨快照对比 | 1. 对比非连续版本<br>2. 验证结果 | 正确累积中间变更 |

### 3.4 版本回滚验证

#### 3.4.1 回滚架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    版本回滚架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    版本回滚流程                          │  │
│   │                                                         │  │
│   │   1. 选择目标版本                                        │  │
│   │        │                                                │  │
│   │        ▼                                                │  │
│   │   2. 检查依赖关系                                        │  │
│   │        │                                                │  │
│   │        ▼                                                │  │
│   │   3. 创建当前版本备份                                    │  │
│   │        │                                                │  │
│   │        ▼                                                │  │
│   │   4. 执行回滚操作                                        │  │
│   │        │                                                │  │
│   │        ▼                                                │  │
│   │   5. 验证回滚结果                                        │  │
│   │        │                                                │  │
│   │        ▼                                                │  │
│   │   6. 更新当前版本指针                                    │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   回滚策略:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  策略类型        适用场景              实现方式          │  │
│   │  ─────────────────────────────────────────────────────  │  │
│   │  全量替换        小项目/紧急回滚      直接加载快照      │  │
│   │  增量回滚        大项目/精确回滚      逆向应用变更      │  │
│   │  混合回滚        通用场景             智能选择策略      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   安全机制:                                                      │
│   - 回滚前自动备份当前版本                                       │
│   - 支持回滚撤销（Undo Rollback）                                │
│   - 冲突检测和提示                                               │
│   - 回滚操作审计日志                                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.4.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| ROLL-001 | 基本回滚 | 1. 选择历史版本<br>2. 执行回滚<br>3. 验证状态 | 项目恢复到目标版本 |
| ROLL-002 | 回滚撤销 | 1. 执行回滚<br>2. 执行撤销回滚<br>3. 验证状态 | 恢复到回滚前状态 |
| ROLL-003 | 并发回滚防护 | 1. 多人同时尝试回滚<br>2. 验证处理 | 排队执行或拒绝后发起者 |
| ROLL-004 | 大项目回滚 | 1. 回滚10万元素项目<br>2. 监控性能 | 回滚完成<60s |
| ROLL-005 | 回滚冲突检测 | 1. 模拟冲突场景<br>2. 执行回滚<br>3. 验证提示 | 正确检测并提示冲突 |

---

## 4. 权限控制POC

### 4.1 RBAC模型实现验证

#### 4.1.1 RBAC架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    RBAC权限模型架构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    RBAC核心模型                          │  │
│   │                                                         │  │
│   │   ┌─────────┐      ┌─────────┐      ┌─────────┐        │  │
│   │   │  User   │─────►│  Role   │─────►│Permission│        │  │
│   │   │ (用户)  │  N:M │ (角色)  │  N:M │ (权限)   │        │  │
│   │   └─────────┘      └─────────┘      └─────────┘        │  │
│   │        │                                            │   │  │
│   │        │                                            │   │  │
│   │        ▼                                            ▼   │  │
│   │   ┌─────────┐                                  ┌────────┐│  │
│   │   │ Project │                                  │Action  ││  │
│   │   │ (项目)  │                                  │(操作)  ││  │
│   │   └─────────┘                                  └────────┘│  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   角色定义:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  角色          权限范围                        适用对象   │  │
│   │  ─────────────────────────────────────────────────────  │  │
│   │  项目所有者    全部权限                        项目负责人 │  │
│   │  项目管理员    除删除项目外的全部权限          项目管理员 │  │
│   │  建筑师        设计相关操作                    设计人员   │  │
│   │  结构工程师    结构专业操作                    结构人员   │  │
│   │  机电工程师    机电专业操作                    机电人员   │  │
│   │  审图人员      查看+批注                       审图专家   │  │
│   │  访客          仅查看                          外部人员   │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   权限粒度:                                                      │
│   - 项目级: project:read, project:write, project:delete        │
│   - 专业级: discipline:architecture:write                      │
│   - 元素级: element:elem_123:update                            │
│   - 操作级: operation:export, operation:import                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.1.2 Casbin策略配置

```ini
# Casbin模型定义 (rbac_model.conf)
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _
g2 = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
```

```csv
# Casbin策略定义 (rbac_policy.csv)
# 格式: p, 角色, 域(项目), 对象, 操作

# 项目所有者权限
p, owner, {project}, project, read
p, owner, {project}, project, write
p, owner, {project}, project, delete
p, owner, {project}, element, *
p, owner, {project}, version, *
p, owner, {project}, export, *
p, owner, {project}, import, *
p, owner, {project}, member, *

# 建筑师权限
p, architect, {project}, project, read
p, architect, {project}, element:architecture, *
p, architect, {project}, element:structure, read
p, architect, {project}, element:mep, read
p, architect, {project}, version, read
p, architect, {project}, export, read

# 结构工程师权限
p, structural_eng, {project}, project, read
p, structural_eng, {project}, element:architecture, read
p, structural_eng, {project}, element:structure, *
p, structural_eng, {project}, element:mep, read
p, structural_eng, {project}, version, read

# 访客权限
p, visitor, {project}, project, read
p, visitor, {project}, element, read
p, visitor, {project}, version, read

# 角色继承
# g, 用户ID, 角色, 项目ID
g, user_001, owner, project_001
g, user_002, architect, project_001
g, user_003, structural_eng, project_001
g, user_004, visitor, project_001
```

#### 4.1.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| RBAC-001 | 角色权限验证 | 1. 为用户分配角色<br>2. 验证各角色权限 | 权限与角色定义一致 |
| RBAC-002 | 角色继承 | 1. 配置角色继承<br>2. 验证权限传递 | 子角色继承父角色权限 |
| RBAC-003 | 动态权限变更 | 1. 运行时修改权限<br>2. 验证生效 | 权限变更即时生效 |
| RBAC-004 | 权限缓存 | 1. 高频权限检查<br>2. 监控性能 | 缓存命中率>90% |
| RBAC-005 | 权限拒绝 | 1. 无权限用户尝试操作<br>2. 验证拒绝 | 正确拒绝并返回403 |

### 4.2 项目级权限验证

#### 4.2.1 项目权限架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    项目级权限架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    项目权限层级                          │  │
│   │                                                         │  │
│   │   ┌─────────────────────────────────────────────────┐  │  │
│   │   │              Organization (组织)                 │  │  │
│   │   │  ┌─────────────────────────────────────────┐   │  │  │
│   │   │  │           Project A (项目A)              │   │  │  │
│   │   │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐   │   │  │  │
│   │   │  │  │ Member1 │ │ Member2 │ │ Member3 │   │   │  │  │
│   │   │  │  │ (Owner) │ │(Architect)│ │(Visitor)│   │   │  │  │
│   │   │  │  └─────────┘ └─────────┘ └─────────┘   │   │  │  │
│   │   │  └─────────────────────────────────────────┘   │  │  │
│   │   │  ┌─────────────────────────────────────────┐   │  │  │
│   │   │  │           Project B (项目B)              │   │  │  │
│   │   │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐   │   │  │  │
│   │   │  │  │ Member1 │ │ Member4 │ │ Member5 │   │   │  │  │
│   │   │  │  │ (Admin) │ │(Structural)│ │(MEP)   │   │   │  │  │
│   │   │  │  └─────────┘ └─────────┘ └─────────┘   │   │  │  │
│   │   │  └─────────────────────────────────────────┘   │  │  │
│   │   └─────────────────────────────────────────────────┘  │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   权限检查流程:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  1. 获取用户身份                                          │  │
│   │       │                                                  │  │
│   │       ▼                                                  │  │
│   │  2. 查询用户在项目中的角色                                 │  │
│   │       │                                                  │  │
│   │       ▼                                                  │  │
│   │  3. 检查角色是否拥有操作权限                               │  │
│   │       │                                                  │  │
│   │       ▼                                                  │  │
│   │  4. 返回检查结果 (Allow/Deny)                              │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.2.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| PROJ-001 | 项目访问控制 | 1. 用户A访问项目X<br>2. 用户A访问项目Y(无权限) | 项目X允许，项目Y拒绝 |
| PROJ-002 | 项目成员管理 | 1. 所有者添加成员<br>2. 非所有者尝试添加 | 所有者成功，其他拒绝 |
| PROJ-003 | 项目删除保护 | 1. 所有者删除项目<br>2. 管理员尝试删除 | 所有者成功，管理员拒绝 |
| PROJ-004 | 跨项目隔离 | 1. 项目A成员访问项目B数据 | 数据完全隔离，访问拒绝 |
| PROJ-005 | 项目转让 | 1. 所有者转让项目<br>2. 验证新所有者权限 | 权限正确转移 |

### 4.3 专业级数据隔离验证

#### 4.3.1 专业隔离架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    专业级数据隔离架构                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    专业数据隔离模型                      │  │
│   │                                                         │  │
│   │   ┌─────────────────────────────────────────────────┐  │  │
│   │   │              Project (项目)                      │  │  │
│   │   │                                                 │  │  │
│   │   │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │  │  │
│   │   │  │Architecture│ │ Structure │ │   MEP    │        │  │  │
│   │   │  │ (建筑)    │ │ (结构)   │ │ (机电)   │        │  │  │
│   │   │  │          │ │          │ │          │        │  │  │
│   │   │  │ ┌──────┐ │ │ ┌──────┐ │ │ ┌──────┐ │        │  │  │
│   │   │  │ │ Walls│ │ │ │Beams │ │ │ │HVAC │ │        │  │  │
│   │   │  │ │Doors │ │ │ │Cols │ │ │ │Plumb│ │        │  │  │
│   │   │  │ │Windows│ │ │ │Slabs│ │ │ │Elect│ │        │  │  │
│   │   │  │ └──────┘ │ │ └──────┘ │ │ └──────┘ │        │  │  │
│   │   │  └──────────┘ └──────────┘ └──────────┘        │  │  │
│   │   │                                                 │  │  │
│   │   └─────────────────────────────────────────────────┘  │  │
│   │                                                         │  │
│   │   数据隔离策略:                                          │  │
│   │   ┌─────────────────────────────────────────────────┐  │  │
│   │   │  隔离级别    实现方式              适用场景       │  │  │
│   │   │  ─────────────────────────────────────────────  │  │  │
│   │   │  表级隔离    专业分表              大数据量      │  │  │
│   │   │  行级隔离    discipline字段过滤    通用方案      │  │  │
│   │   │  列级隔离    视图/字段权限         敏感数据      │  │  │
│   │   └─────────────────────────────────────────────────┘  │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   权限矩阵:                                                      │
│   ┌────────────────┬────────┬──────────┬────────┬────────┐     │
│   │ 角色/专业      │ 建筑   │ 结构     │ 机电   │ 其他   │     │
│   ├────────────────┼────────┼──────────┼────────┼────────┤     │
│   │ 建筑师         │ RW     │ R        │ R      │ R      │     │
│   │ 结构工程师     │ R      │ RW       │ R      │ R      │     │
│   │ 机电工程师     │ R      │ R        │ RW     │ R      │     │
│   │ 审图人员       │ R      │ R        │ R      │ R      │     │
│   └────────────────┴────────┴──────────┴────────┴────────┘     │
│   (R=Read, W=Write)                                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.3.2 行级隔离实现

```sql
-- 元素表（带专业隔离）
CREATE TABLE elements (
    id VARCHAR(64) PRIMARY KEY,
    project_id VARCHAR(64) NOT NULL,
    discipline VARCHAR(20) NOT NULL, -- 专业字段
    element_type VARCHAR(50) NOT NULL,
    name VARCHAR(200),
    properties JSONB,
    geometry JSONB,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by VARCHAR(64),
    updated_at TIMESTAMP WITH TIME ZONE,
    
    -- 复合索引
    CONSTRAINT idx_elements_project_discipline 
        UNIQUE (project_id, discipline, id)
);

-- 创建行级安全策略
CREATE POLICY discipline_isolation_policy ON elements
    FOR ALL
    USING (
        -- 检查用户是否有该专业的访问权限
        EXISTS (
            SELECT 1 FROM user_discipline_permissions udp
            WHERE udp.user_id = current_setting('app.current_user_id')::VARCHAR
            AND udp.project_id = elements.project_id
            AND udp.discipline = elements.discipline
            AND udp.permission_level IN ('read', 'write')
        )
        OR
        -- 项目所有者绕过检查
        EXISTS (
            SELECT 1 FROM project_members pm
            WHERE pm.user_id = current_setting('app.current_user_id')::VARCHAR
            AND pm.project_id = elements.project_id
            AND pm.role = 'owner'
        )
    );

-- 启用行级安全
ALTER TABLE elements ENABLE ROW LEVEL SECURITY;
```

#### 4.3.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| DISC-001 | 专业读取隔离 | 1. 建筑师查询元素<br>2. 验证返回结果 | 只能看到建筑专业元素 |
| DISC-002 | 专业写入隔离 | 1. 建筑师修改结构元素<br>2. 验证结果 | 写入被拒绝 |
| DISC-003 | 跨专业查看 | 1. 配置跨专业权限<br>2. 验证访问 | 可查看指定专业 |
| DISC-004 | 专业管理员 | 1. 项目管理员查询<br>2. 验证返回 | 可查看所有专业 |
| DISC-005 | 专业数据统计 | 1. 各专业用户执行统计<br>2. 验证结果 | 只统计有权限专业 |

### 4.4 细粒度操作权限验证

#### 4.4.1 细粒度权限架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    细粒度操作权限架构                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   权限粒度层级:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                                                         │  │
│   │   Level 1: 系统级                                        │  │
│   │   └── system:admin, system:user, system:guest          │  │
│   │                                                         │  │
│   │   Level 2: 组织级                                        │  │
│   │   └── org:{orgId}:admin, org:{orgId}:member            │  │
│   │                                                         │  │
│   │   Level 3: 项目级                                        │  │
│   │   └── project:{projId}:owner, project:{projId}:member  │  │
│   │                                                         │  │
│   │   Level 4: 专业级                                        │  │
│   │   └── discipline:{disc}:write, discipline:{disc}:read  │  │
│   │                                                         │  │
│   │   Level 5: 元素级                                        │  │
│   │   └── element:{elemId}:update, element:{elemId}:delete │  │
│   │                                                         │  │
│   │   Level 6: 操作级                                        │  │
│   │   └── operation:export, operation:import,              │  │
│   │       operation:share, operation:publish               │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   操作权限定义:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  操作          权限标识                    描述          │  │
│   │  ─────────────────────────────────────────────────────  │  │
│   │  创建元素      element:create            新建设计元素   │  │
│   │  修改元素      element:update            修改元素属性   │  │
│   │  删除元素      element:delete            删除元素       │  │
│   │  查看元素      element:read              查看元素详情   │  │
│   │  导出项目      project:export            导出项目文件   │  │
│   │  导入项目      project:import            导入项目文件   │  │
│   │  分享项目      project:share             分享项目链接   │  │
│   │  发布版本      version:publish           发布正式版本   │  │
│   │  回滚版本      version:rollback          回滚到历史版本 │  │
│   │  管理成员      project:member:manage     管理项目成员   │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.4.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| FINE-001 | 元素级权限 | 1. 配置元素级权限<br>2. 验证访问控制 | 精确到单个元素 |
| FINE-002 | 操作级权限 | 1. 配置导出权限<br>2. 验证导出操作 | 无权限用户导出被拒绝 |
| FINE-003 | 条件权限 | 1. 配置时间限制权限<br>2. 验证不同时段 | 限制时段外访问拒绝 |
| FINE-004 | 权限组合 | 1. 配置多维度权限<br>2. 验证权限计算 | 权限正确组合生效 |
| FINE-005 | 权限继承 | 1. 配置继承权限<br>2. 验证子对象权限 | 子对象继承父对象权限 |

---

## 5. 账号隔离验证

### 5.1 多租户隔离验证

#### 5.1.1 多租户架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    多租户隔离架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   租户隔离策略: 混合模式 (共享数据库 + 行级隔离)                  │
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    数据库层                              │  │
│   │                                                         │  │
│   │   ┌─────────────────────────────────────────────────┐  │  │
│   │   │              Shared Database                     │  │  │
│   │   │                                                 │  │  │
│   │   │  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │  │  │
│   │   │  │ Tenant A    │  │ Tenant B    │  │ Tenant C│ │  │  │
│   │   │  │ Data        │  │ Data        │  │ Data    │ │  │  │
│   │   │  │ (tenant_id) │  │ (tenant_id) │  │(tenant_id)│  │  │
│   │   │  └─────────────┘  └─────────────┘  └─────────┘ │  │  │
│   │   │                                                 │  │  │
│   │   │  隔离方式: 行级安全策略(RLS) + 应用层过滤        │  │  │
│   │   └─────────────────────────────────────────────────┘  │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    应用层                                │  │
│   │                                                         │  │
│   │   ┌─────────┐     ┌─────────┐     ┌─────────┐          │  │
│   │   │ Tenant  │     │ Tenant  │     │ Tenant  │          │  │
│   │   │ Context │     │ Context │     │ Context │          │  │
│   │   │   A     │     │   B     │     │   C     │          │  │
│   │   └────┬────┘     └────┬────┘     └────┬────┘          │  │
│   │        │               │               │                │  │
│   │        └───────────────┼───────────────┘                │  │
│   │                        │                                │  │
│   │                        ▼                                │  │
│   │              ┌─────────────────┐                        │  │
│   │              │  Tenant Filter  │                        │  │
│   │              │  (Middleware)   │                        │  │
│   │              └─────────────────┘                        │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    存储层                                │  │
│   │                                                         │  │
│   │   租户A存储: /storage/tenant-a/...                      │  │
│   │   租户B存储: /storage/tenant-b/...                      │  │
│   │   租户C存储: /storage/tenant-c/...                      │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 5.1.2 租户隔离实现

```sql
-- 租户信息表
CREATE TABLE tenants (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    domain VARCHAR(200) UNIQUE,
    status VARCHAR(20) DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 所有业务表添加租户ID
CREATE TABLE projects (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES tenants(id),
    name VARCHAR(200) NOT NULL,
    -- ... 其他字段
);

-- 租户行级安全策略
CREATE POLICY tenant_isolation_policy ON projects
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::VARCHAR);

-- 设置当前租户ID的函数
CREATE OR REPLACE FUNCTION set_current_tenant(tenant_id VARCHAR)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant_id, false);
END;
$$ LANGUAGE plpgsql;
```

#### 5.1.3 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| TENANT-001 | 数据隔离 | 1. 租户A创建数据<br>2. 租户B查询数据 | 租户B看不到租户A数据 |
| TENANT-002 | 跨租户访问 | 1. 租户A尝试访问租户B资源 | 访问被拒绝 |
| TENANT-003 | 租户上下文 | 1. 切换租户上下文<br>2. 验证数据可见性 | 数据随租户切换 |
| TENANT-004 | 租户存储隔离 | 1. 各租户上传文件<br>2. 验证存储路径 | 文件存储在各自目录 |
| TENANT-005 | 租户删除 | 1. 删除租户<br>2. 验证数据清理 | 租户数据完全清理 |

### 5.2 跨账号访问防护验证

#### 5.2.1 访问防护架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    跨账号访问防护架构                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   防护层级:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                                                         │  │
│   │   Layer 1: 认证层                                        │  │
│   │   ├── JWT Token 验证                                     │  │
│   │   ├── Token 过期检查                                     │  │
│   │   └── Token 签名验证                                     │  │
│   │                                                         │  │
│   │   Layer 2: 授权层                                        │  │
│   │   ├── 用户-资源关系验证                                  │  │
│   │   ├── 角色权限检查                                       │  │
│   │   └── 资源所有权验证                                     │  │
│   │                                                         │  │
│   │   Layer 3: 数据层                                        │  │
│   │   ├── 行级安全策略                                       │  │
│   │   ├── 查询过滤器                                         │  │
│   │   └── 字段级脱敏                                         │  │
│   │                                                         │  │
│   │   Layer 4: 审计层                                        │  │
│   │   ├── 访问日志记录                                       │  │
│   │   ├── 异常访问告警                                       │  │
│   │   └── 访问频率限制                                       │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   防护机制:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  机制              实现              防护目标            │  │
│   │  ─────────────────────────────────────────────────────  │  │
│   │  URL参数校验      正则匹配          防止ID遍历          │  │
│   │  资源所有权检查   数据库查询        防止越权访问        │  │
│   │  会话绑定         Token-IP绑定      防止会话劫持        │  │
│   │  操作频率限制     滑动窗口限流      防止暴力破解        │  │
│   │  敏感操作验证     二次认证          防止关键操作被篡改  │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 5.2.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| XACC-001 | ID遍历防护 | 1. 尝试遍历项目ID<br>2. 验证访问结果 | 无权限项目访问拒绝 |
| XACC-002 | URL参数篡改 | 1. 篡改URL中的用户ID<br>2. 验证防护 | 篡改被检测，请求拒绝 |
| XACC-003 | 会话劫持防护 | 1. 复制Token到不同IP<br>2. 验证访问 | Token验证失败 |
| XACC-004 | 水平越权 | 1. 用户A访问用户B资源<br>2. 验证结果 | 访问被拒绝 |
| XACC-005 | 垂直越权 | 1. 普通用户尝试管理员操作<br>2. 验证结果 | 操作被拒绝 |

### 5.3 会话管理验证

#### 5.3.1 会话管理架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    会话管理架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    会话生命周期                          │  │
│   │                                                         │  │
│   │   ┌─────────┐    login    ┌─────────┐   activity      │  │
│   │   │  匿名   │────────────►│  活跃   │◄─────────┐      │  │
│   │   └─────────┘             └────┬────┘          │      │  │
│   │                                │               │      │  │
│   │                           idle │ timeout       │      │  │
│   │                                ▼               │      │  │
│   │                           ┌─────────┐          │      │  │
│   │                    refresh│  空闲   │──────────┘      │  │
│   │                    ┌──────┤         │                  │  │
│   │                    │      └────┬────┘                  │  │
│   │                    │    expire │                       │  │
│   │                    │           ▼                       │  │
│   │                    │      ┌─────────┐   logout         │  │
│   │                    └─────►│  过期   │────────►┌─────┐ │  │
│   │                           └─────────┘         │ 结束 │ │  │
│   │                                               └─────┘ │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   会话存储:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  {                                                      │  │
│   │    "sessionId": "sess_abc123",                          │  │
│   │    "userId": "user_456",                                │  │
│   │    "tenantId": "tenant_789",                            │  │
│   │    "roles": ["architect", "project_member"],            │  │
│   │    "permissions": [...],                                │  │
│   │    "createdAt": "2024-01-15T10:00:00Z",                 │  │
│   │    "expiresAt": "2024-01-15T18:00:00Z",                 │  │
│   │    "lastActivity": "2024-01-15T12:00:00Z",              │  │
│   │    "ipAddress": "192.168.1.100",                        │  │
│   │    "userAgent": "Mozilla/5.0...",                       │  │
│   │    "concurrentSessions": 3                              │  │
│   │  }                                                      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 5.3.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| SESS-001 | 会话创建 | 1. 用户登录<br>2. 验证会话创建 | 会话正确创建，Token返回 |
| SESS-002 | 会话过期 | 1. 等待会话过期<br>2. 尝试访问 | 访问被拒绝，提示重新登录 |
| SESS-003 | 会话续期 | 1. 活跃使用会话<br>2. 验证过期时间 | 过期时间自动延长 |
| SESS-004 | 多设备登录 | 1. 同一用户多设备登录<br>2. 验证会话数 | 支持配置的最大会话数 |
| SESS-005 | 会话踢出 | 1. 管理员踢出用户会话<br>2. 验证结果 | 会话立即失效 |
| SESS-006 | 会话并发限制 | 1. 超出最大会话数登录<br>2. 验证处理 | 按策略处理(拒绝/踢出最旧) |

---

## 6. 审计追踪POC

### 6.1 操作日志记录验证

#### 6.1.1 审计日志架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    审计日志架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    审计日志收集流程                      │  │
│   │                                                         │  │
│   │   ┌─────────┐    ┌─────────┐    ┌─────────┐            │  │
│   │   │ 操作源  │───►│ 拦截器  │───►│ 格式化  │            │  │
│   │   │         │    │(AOP)    │    │         │            │  │
│   │   └─────────┘    └────┬────┘    └────┬────┘            │  │
│   │                        │              │                 │  │
│   │                        ▼              ▼                 │  │
│   │                   ┌─────────────────────────┐           │  │
│   │                   │      消息队列            │           │  │
│   │                   │    (RabbitMQ/Kafka)     │           │  │
│   │                   └───────────┬─────────────┘           │  │
│   │                               │                         │  │
│   │              ┌────────────────┼────────────────┐        │  │
│   │              ▼                ▼                ▼        │  │
│   │        ┌─────────┐      ┌─────────┐      ┌─────────┐   │  │
│   │        │实时分析 │      │日志存储 │      │告警通知 │   │  │
│   │        │(Flink) │      │(ES/DB) │      │(Webhook)│   │  │
│   │        └─────────┘      └─────────┘      └─────────┘   │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   审计日志内容:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  {                                                      │  │
│   │    "logId": "audit_001",                                │  │
│   │    "timestamp": "2024-01-15T10:30:00.123Z",             │  │
│   │    "level": "INFO",                                     │  │
│   │    "category": "DATA_ACCESS",                           │  │
│   │    "action": "ELEMENT_UPDATE",                          │  │
│   │    "status": "SUCCESS",                                 │  │
│   │    "actor": {                                           │  │
│   │      "userId": "user_123",                              │  │
│   │      "username": "zhangsan",                            │  │
│   │      "tenantId": "tenant_456",                          │  │
│   │      "ipAddress": "192.168.1.100",                      │  │
│   │      "userAgent": "Mozilla/5.0..."                     │  │
│   │    },                                                   │  │
│   │    "resource": {                                        │  │
│   │      "type": "ELEMENT",                                 │  │
│   │      "id": "elem_789",                                  │  │
│   │      "projectId": "proj_abc"                            │  │
│   │    },                                                   │  │
│   │    "details": {                                         │  │
│   │      "before": {...},                                   │  │
│   │      "after": {...},                                    │  │
│   │      "changes": [...]                                   │  │
│   │    },                                                   │  │
│   │    "duration": 150,                                     │  │
│   │    "sessionId": "sess_xyz"                              │  │
│   │  }                                                      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 6.1.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| AUDIT-001 | 操作日志记录 | 1. 执行各类操作<br>2. 检查审计日志 | 所有操作被记录 |
| AUDIT-002 | 日志完整性 | 1. 检查日志字段<br>2. 验证必填字段 | 所有必填字段存在 |
| AUDIT-003 | 异步日志 | 1. 高频操作<br>2. 监控日志延迟 | 延迟<1s |
| AUDIT-004 | 日志持久化 | 1. 模拟系统故障<br>2. 验证日志不丢失 | 日志可靠持久化 |
| AUDIT-005 | 敏感数据脱敏 | 1. 检查敏感操作日志<br>2. 验证脱敏 | 敏感信息被脱敏 |

### 6.2 变更追踪验证

#### 6.2.1 变更追踪架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    变更追踪架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    变更追踪流程                          │  │
│   │                                                         │  │
│   │   变更前状态 ──► 执行变更 ──► 变更后状态 ──► 计算差异   │  │
│   │        │                                              │  │
│   │        ▼                                              │  │
│   │   ┌─────────┐     ┌─────────┐     ┌─────────┐        │  │
│   │   │ 快照A   │────►│ 变更操作 │────►│ 快照B   │        │  │
│   │   └─────────┘     └─────────┘     └─────────┘        │  │
│   │                                        │              │  │
│   │                                        ▼              │  │
│   │                              ┌─────────────────┐      │  │
│   │                              │  差异计算引擎   │      │  │
│   │                              │                 │      │  │
│   │                              │ ┌─────────────┐ │      │  │
│   │                              │ │ 新增字段    │ │      │  │
│   │                              │ │ 删除字段    │ │      │  │
│   │                              │ │ 修改字段    │ │      │  │
│   │                              │ │ 字段路径    │ │      │  │
│   │                              │ │ 旧值/新值   │ │      │  │
│   │                              │ └─────────────┘ │      │  │
│   │                              └─────────────────┘      │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   变更记录结构:                                                  │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  {                                                      │  │
│   │    "changeId": "chg_001",                               │  │
│   │    "changeType": "UPDATE",                              │  │
│   │    "entityType": "ELEMENT",                             │  │
│   │    "entityId": "elem_123",                              │  │
│   │    "changes": [                                         │  │
│   │      {                                                  │  │
│   │        "field": "properties.width",                     │  │
│   │        "oldValue": 100,                                 │  │
│   │        "newValue": 150,                                 │  │
│   │        "changeType": "MODIFIED"                         │  │
│   │      },                                                 │  │
│   │      {                                                  │  │
│   │        "field": "properties.height",                    │  │
│   │        "oldValue": null,                                │  │
│   │        "newValue": 200,                                 │  │
│   │        "changeType": "ADDED"                            │  │
│   │      }                                                  │  │
│   │    ],                                                   │  │
│   │    "changedBy": "user_456",                             │  │
│   │    "changedAt": "2024-01-15T10:30:00Z"                  │  │
│   │  }                                                      │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 6.2.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| TRACK-001 | 字段级变更追踪 | 1. 修改元素属性<br>2. 检查变更记录 | 精确到字段级变更 |
| TRACK-002 | 嵌套对象变更 | 1. 修改嵌套对象属性<br>2. 验证追踪 | 正确追踪嵌套变更 |
| TRACK-003 | 数组变更 | 1. 修改数组元素<br>2. 验证追踪 | 正确追踪数组变更 |
| TRACK-004 | 批量变更 | 1. 批量修改多个元素<br>2. 验证追踪 | 每个元素变更独立记录 |
| TRACK-005 | 变更回滚 | 1. 根据变更记录回滚<br>2. 验证结果 | 正确回滚到变更前 |

### 6.3 审计查询验证

#### 6.3.1 审计查询架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    审计查询架构                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │                    审计查询接口                          │  │
│   │                                                         │  │
│   │   GET /api/audit/logs                                   │  │
│   │   Query Parameters:                                     │  │
│   │   - startTime: 开始时间                                  │  │
│   │   - endTime: 结束时间                                    │  │
│   │   - userId: 用户ID                                       │  │
│   │   - projectId: 项目ID                                    │  │
│   │   - action: 操作类型                                     │  │
│   │   - resourceType: 资源类型                               │  │
│   │   - resourceId: 资源ID                                   │  │
│   │   - status: 操作状态                                     │  │
│   │   - page: 页码                                           │  │
│   │   - pageSize: 每页数量                                   │  │
│   │                                                         │  │
│   │   Response:                                             │  │
│   │   {                                                     │  │
│   │     "total": 1000,                                      │  │
│   │     "page": 1,                                          │  │
│   │     "pageSize": 20,                                     │  │
│   │     "data": [...]                                       │  │
│   │   }                                                     │  │
│   │                                                         │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
│   查询优化:                                                      │
│   ┌─────────────────────────────────────────────────────────┐  │
│   │  策略              实现              效果               │  │
│   │  ─────────────────────────────────────────────────────  │  │
│   │  时间分区          按月分区表        查询效率提升10x    │  │
│   │  复合索引          多字段索引        覆盖常见查询       │  │
│   │  全文检索          Elasticsearch     支持模糊搜索       │  │
│   │  缓存热点          Redis缓存        常用查询<100ms      │  │
│   │  预聚合            物化视图          统计查询秒级响应   │  │
│   └─────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 6.3.2 验证测试用例

| 用例ID | 用例名称 | 测试步骤 | 预期结果 |
|--------|---------|---------|---------|
| QUERY-001 | 基础查询 | 1. 按时间范围查询<br>2. 验证结果 | 返回正确数据 |
| QUERY-002 | 组合查询 | 1. 多条件组合查询<br>2. 验证结果 | 正确应用所有条件 |
| QUERY-003 | 分页查询 | 1. 大数据量分页查询<br>2. 验证性能 | 每页<500ms |
| QUERY-004 | 聚合查询 | 1. 执行统计聚合查询<br>2. 验证结果 | 统计结果正确 |
| QUERY-005 | 导出查询 | 1. 导出大量审计日志<br>2. 验证性能 | 导出完成<30s |

---

## 7. POC执行计划

### 7.1 测试场景设计

#### 7.1.1 测试场景矩阵

```
┌─────────────────────────────────────────────────────────────────┐
│                    测试场景矩阵                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────────┬──────────────────────────────────────────┐  │
│   │   测试领域   │              测试场景                     │  │
│   ├──────────────┼──────────────────────────────────────────┤  │
│   │              │ T1: 单用户连续操作撤销重做               │  │
│   │   撤销重做   │ T2: 多用户并发操作冲突处理               │  │
│   │              │ T3: 批量操作(100+元素)撤销重做           │  │
│   │              │ T4: 极端场景(1000+操作栈)                │  │
│   ├──────────────┼──────────────────────────────────────────┤  │
│   │              │ T5: 大项目(10万元素)快照生成             │  │
│   │   版本管理   │ T6: 增量存储空间效率验证                 │  │
│   │              │ T7: 版本对比性能测试                     │  │
│   │              │ T8: 版本回滚可靠性测试                   │  │
│   ├──────────────┼──────────────────────────────────────────┤  │
│   │              │ T9: RBAC权限模型完整验证                 │  │
│   │   权限控制   │ T10: 专业级数据隔离验证                  │  │
│   │              │ T11: 细粒度权限(元素级)验证              │  │
│   │              │ T12: 权限缓存性能测试                    │  │
│   ├──────────────┼──────────────────────────────────────────┤  │
│   │              │ T13: 多租户数据隔离验证                  │  │
│   │   账号隔离   │ T14: 跨账号访问防护测试                  │  │
│   │              │ T15: 会话生命周期管理测试                │  │
│   │              │ T16: 并发会话限制测试                    │  │
│   ├──────────────┼──────────────────────────────────────────┤  │
│   │              │ T17: 操作日志完整性验证                  │  │
│   │   审计追踪   │ T18: 变更追踪准确性验证                  │  │
│   │              │ T19: 审计查询性能测试                    │  │
│   │              │ T20: 日志归档和压缩测试                  │  │
│   └──────────────┴──────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 测试数据准备

#### 7.2.1 测试数据集

| 数据集名称 | 数据规模 | 用途 |
|-----------|---------|------|
| 小型项目 | 100元素 | 功能验证 |
| 中型项目 | 1万元素 | 性能基准 |
| 大型项目 | 10万元素 | 压力测试 |
| 超大型项目 | 50万元素 | 极限测试 |
| 多租户数据 | 10租户×1万元素 | 隔离测试 |
| 历史版本链 | 100版本 | 版本管理测试 |
| 操作日志 | 100万条 | 审计查询测试 |

#### 7.2.2 测试环境配置

```yaml
# 测试环境配置
environment:
  # 数据库
  postgresql:
    version: "15"
    cpu: 4
    memory: 8GB
    storage: 100GB
    
  redis:
    version: "7"
    cpu: 2
    memory: 4GB
    
  # 应用服务器
  app_server:
    instances: 2
    cpu: 4
    memory: 8GB
    
  # 负载测试工具
  k8s:
    enabled: true
    namespace: poc-test
```

### 7.3 验收标准

#### 7.3.1 功能验收标准

| 验收项 | 验收标准 | 优先级 |
|-------|---------|--------|
| 撤销重做 | 支持无限级撤销/重做，批量操作原子性 | P0 |
| 快照生成 | 10万元素项目快照<30s | P0 |
| 增量存储 | 相比全量存储节省>70%空间 | P0 |
| 权限检查 | 单次检查<10ms | P0 |
| 数据隔离 | 租户间数据100%隔离 | P0 |
| 审计日志 | 所有操作100%记录 | P0 |

#### 7.3.2 性能验收标准

| 性能指标 | 目标值 | 测试方法 |
|---------|-------|---------|
| 撤销操作延迟 | <100ms | 执行撤销，测量响应时间 |
| 重做操作延迟 | <100ms | 执行重做，测量响应时间 |
| 快照生成速度 | >3000元素/s | 大项目快照生成 |
| 版本对比速度 | <10s(1万变更) | 大版本对比 |
| 权限检查TPS | >1000 | 并发权限检查 |
| 审计查询响应 | <500ms | 复杂条件查询 |

#### 7.3.3 可靠性验收标准

| 可靠性指标 | 目标值 | 测试方法 |
|-----------|-------|---------|
| 数据一致性 | 100% | 并发操作后验证 |
| 日志完整性 | 100% | 故障恢复后验证 |
| 系统可用性 | 99.9% | 7×24小时运行 |
| 故障恢复时间 | <5分钟 | 模拟故障恢复 |

### 7.4 执行时间表

```
┌─────────────────────────────────────────────────────────────────┐
│                    POC执行时间表                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   第1周: 环境准备和基础验证                                      │
│   ├─ Day 1-2: 测试环境搭建                                      │
│   ├─ Day 3-4: 测试数据准备                                      │
│   └─ Day 5: 基础功能冒烟测试                                    │
│                                                                 │
│   第2周: 撤销重做POC                                            │
│   ├─ Day 1-2: 命令模式实现验证                                  │
│   ├─ Day 3: 操作日志存储验证                                    │
│   ├─ Day 4: 撤销栈/重做栈管理验证                               │
│   └─ Day 5: 批量操作撤销验证 + 周总结                           │
│                                                                 │
│   第3周: 历史版本管理POC                                        │
│   ├─ Day 1: 快照生成验证                                        │
│   ├─ Day 2: 增量存储验证                                        │
│   ├─ Day 3: 版本对比验证                                        │
│   └─ Day 4-5: 版本回滚验证 + 周总结                             │
│                                                                 │
│   第4周: 权限控制POC                                            │
│   ├─ Day 1-2: RBAC模型实现验证                                  │
│   ├─ Day 3: 项目级权限验证                                      │
│   ├─ Day 4: 专业级数据隔离验证                                  │
│   └─ Day 5: 细粒度操作权限验证 + 周总结                         │
│                                                                 │
│   第5周: 账号隔离和审计追踪POC                                  │
│   ├─ Day 1: 多租户隔离验证                                      │
│   ├─ Day 2: 跨账号访问防护验证                                  │
│   ├─ Day 3: 会话管理验证                                        │
│   ├─ Day 4: 操作日志记录验证                                    │
│   └─ Day 5: 变更追踪和审计查询验证 + 周总结                     │
│                                                                 │
│   第6周: 综合测试和报告                                         │
│   ├─ Day 1-2: 集成测试和性能压测                                │
│   ├─ Day 3: 问题修复和回归测试                                  │
│   ├─ Day 4: 编写验证报告                                        │
│   └─ Day 5: 评审和汇报                                          │
│                                                                 │
│   总计: 6周 (30个工作日)                                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. 附录

### 8.1 术语表

| 术语 | 说明 |
|-----|------|
| POC | Proof of Concept，概念验证 |
| RBAC | Role-Based Access Control，基于角色的访问控制 |
| JWT | JSON Web Token |
| RLS | Row Level Security，行级安全 |
| AOP | Aspect-Oriented Programming，面向切面编程 |
| TPS | Transactions Per Second，每秒事务数 |

### 8.2 参考文档

1. Casbin官方文档: https://casbin.org/
2. Keycloak官方文档: https://www.keycloak.org/documentation
3. PostgreSQL RLS文档: https://www.postgresql.org/docs/current/ddl-rowsecurity.html

### 8.3 风险与应对

| 风险 | 影响 | 应对措施 |
|-----|------|---------|
| 性能不达标 | 高 | 提前进行性能基准测试，准备优化方案 |
| 数据隔离漏洞 | 高 | 增加安全测试覆盖，引入第三方安全审计 |
| 技术栈兼容性问题 | 中 | POC阶段充分验证集成点 |
| 进度延期 | 中 | 预留缓冲时间，优先级管理 |

---

**文档结束**

*本报告由版本控制系统专家编制，用于半自动化建筑设计平台可行性验证阶段技术评审。*
