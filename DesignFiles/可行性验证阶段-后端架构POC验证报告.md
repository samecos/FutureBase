# 可行性验证阶段 - 后端架构POC验证报告

**项目名称**：半自动化建筑设计平台  
**文档版本**：v1.0  
**编写日期**：2024年  
**文档类型**：技术可行性验证报告

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [技术方案验证](#2-技术方案验证)
3. [核心组件POC](#3-核心组件poc)
4. [性能验证方案](#4-性能验证方案)
5. [集成验证方案](#5-集成验证方案)
6. [关键技术风险验证](#6-关键技术风险验证)
7. [POC执行计划](#7-poc执行计划)
8. [结论与建议](#8-结论与建议)

---

## 1. 执行摘要

### 1.1 POC目标

本POC验证旨在确认半自动化建筑设计平台后端架构的核心技术可行性，验证以下关键技术方案：

| 验证项 | 目标 | 验收标准 |
|--------|------|----------|
| 微服务架构 | 服务拆分合理性 | 服务边界清晰，接口契约稳定 |
| CRDT协作引擎 | 多人实时协作 | 无冲突合并，延迟<100ms |
| WebSocket通信 | 实时消息广播 | 连接稳定，支持100+并发 |
| 事件驱动架构 | 异步消息处理 | 消息可靠投递，顺序保证 |

### 1.2 验证范围

```
┌─────────────────────────────────────────────────────────────────┐
│                      POC验证范围                                 │
├─────────────────────────────────────────────────────────────────┤
│  ✅ 包含                                                        │
│  ├── 核心服务POC（协作、几何、脚本调度）                        │
│  ├── CRDT算法核心实现                                           │
│  ├── WebSocket连接管理                                          │
│  ├── 服务间通信（gRPC + 消息队列）                              │
│  └── 性能基准测试                                               │
│                                                                 │
│  ❌ 不包含                                                      │
│  ├── 完整业务功能实现                                           │
│  ├── 生产级安全认证                                             │
│  └── 完整监控告警体系                                           │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 技术方案验证

### 2.1 微服务拆分方案验证

#### 2.1.1 服务边界定义

基于领域驱动设计（DDD）原则，建议将系统拆分为以下核心服务：

```
┌─────────────────────────────────────────────────────────────────────┐
│                         微服务架构全景图                              │
└─────────────────────────────────────────────────────────────────────┘

                              ┌─────────────┐
                              │   API网关   │
                              │  (Kong/Envoy)│
                              └──────┬──────┘
                                     │
           ┌─────────────────────────┼─────────────────────────┐
           │                         │                         │
           ▼                         ▼                         ▼
    ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
    │  用户服务   │          │  项目服务   │          │  协作服务   │
    │  (Go)       │          │  (Go)       │          │  (Go)       │
    │             │          │             │          │  CRDT核心   │
    │ - 认证授权  │          │ - 项目管理  │          │ - 实时同步  │
    │ - 用户管理  │          │ - 版本控制  │          │ - 冲突解决  │
    └──────┬──────┘          └──────┬──────┘          └──────┬──────┘
           │                         │                         │
           │    ┌────────────────────┼────────────────────┐   │
           │    │                    │                    │   │
           ▼    ▼                    ▼                    ▼   ▼
    ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
    │  几何服务   │          │  脚本服务   │          │  文件服务   │
    │  (Go)       │          │  (Java/SB)  │          │  (Go)       │
    │             │          │             │          │             │
    │ - BRep处理  │          │ - 任务调度  │          │ - 文件存储  │
    │ - 网格计算  │          │ - 脚本执行  │          │ - CDN分发   │
    └──────┬──────┘          └──────┬──────┘          └──────┬──────┘
           │                         │                         │
           └─────────────────────────┼─────────────────────────┘
                                     │
                              ┌──────┴──────┐
                              │  基础设施层  │
                              ├─────────────┤
                              │ PostgreSQL  │
                              │    Redis    │
                              │    Kafka    │
                              │   MinIO     │
                              └─────────────┘
```

#### 2.1.2 服务接口契约

**协作服务 API 契约示例：**

```protobuf
// collaboration.proto
syntax = "proto3";
package collaboration;

service CollaborationService {
  // 加入协作会话
  rpc JoinSession(JoinRequest) returns (JoinResponse);
  
  // 发送操作（客户端流）
  rpc SendOperation(stream Operation) returns (stream OperationAck);
  
  // 订阅变更（服务端流）
  rpc SubscribeChanges(SubscribeRequest) returns (stream ChangeEvent);
  
  // 获取文档状态
  rpc GetDocumentState(DocumentId) returns (DocumentState);
  
  // 离开会话
  rpc LeaveSession(LeaveRequest) returns (LeaveResponse);
}

message Operation {
  string operation_id = 1;
  string user_id = 2;
  string document_id = 3;
  int64 timestamp = 4;
  int64 vector_clock = 5;
  
  oneof payload {
    WallOperation wall_op = 10;
    ElementOperation element_op = 11;
    TransformOperation transform_op = 12;
  }
}

message WallOperation {
  string wall_id = 1;
  ActionType action = 2;
  Point3D start_point = 3;
  Point3D end_point = 4;
  float height = 5;
  float thickness = 6;
  map<string, string> properties = 7;
}

enum ActionType {
  CREATE = 0;
  UPDATE = 1;
  DELETE = 2;
}

message Point3D {
  double x = 1;
  double y = 2;
  double z = 3;
}
```

#### 2.1.3 服务拆分验证标准

| 验证项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| 服务内聚性 | 代码审查 | 每个服务有明确的单一职责 |
| 服务耦合度 | 依赖分析 | 服务间仅通过API通信，无直接数据库共享 |
| 接口稳定性 | 契约测试 | 接口变更不影响其他服务 |
| 独立部署 | CI/CD验证 | 单个服务可独立构建和部署 |

### 2.2 CRDT协作引擎POC设计

#### 2.2.1 CRDT选型分析

```
┌─────────────────────────────────────────────────────────────────────┐
│                     CRDT算法选型对比                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. State-based CRDT (CvRDT)                                        │
│     ✅ 优点：实现简单，易于理解                                       │
│     ❌ 缺点：状态传输量大，不适合高频更新                              │
│     适用：低频变更，强一致性场景                                      │
│                                                                     │
│  2. Operation-based CRDT (CmRDT)  ★ 推荐                            │
│     ✅ 优点：传输增量，带宽友好                                       │
│     ✅ 优点：实时性好，延迟低                                         │
│     ⚠️  注意：需要可靠广播保证                                         │
│     适用：高频协作编辑场景                                            │
│                                                                     │
│  3. Delta-state CRDT                                                  │
│     ✅ 优点：平衡了State和Operation的优点                             │
│     ⚠️  缺点：实现复杂度高                                            │
│     适用：大规模分布式场景                                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**推荐方案**：Operation-based CRDT (CmRDT)

#### 2.2.2 CRDT核心架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CRDT协作引擎架构                                  │
└─────────────────────────────────────────────────────────────────────┘

    客户端A                    协作服务                    客户端B
       │                          │                          │
       │  1. 本地操作              │                          │
       │ ──────────────────────▶  │                          │
       │                          │                          │
       │  2. 生成Op + VC          │                          │
       │ ──────────────────────▶  │                          │
       │                          │                          │
       │                          │  3. 应用本地              │
       │                          │ ──────────────────────▶  │
       │                          │                          │
       │                          │  4. 广播Op               │
       │                          │ ──────────────────────▶  │
       │                          │                          │
       │  5. 接收远程Op           │                          │
       │ ◀──────────────────────  │                          │
       │                          │                          │
       │  6. 转换 + 应用          │                          │
       │                          │                          │

┌─────────────────────────────────────────────────────────────────────┐
│                        CRDT核心组件                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │
│  │ 操作生成器  │───▶│ 向量时钟    │───▶│ 操作转换器  │             │
│  │ Operation   │    │ VectorClock │    │ OT Engine   │             │
│  │ Generator   │    │             │    │             │             │
│  └─────────────┘    └─────────────┘    └─────────────┘             │
│         │                  │                  │                     │
│         ▼                  ▼                  ▼                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │
│  │ 状态管理器  │    │ 冲突解决器  │    │ 历史记录器  │             │
│  │ StateMgr    │    │ Conflict    │    │ HistoryLog  │             │
│  │             │    │ Resolver    │    │             │             │
│  └─────────────┘    └─────────────┘    └─────────────┘             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 2.2.3 墙体编辑CRDT实现示例

```go
// wall_crdt.go - 墙体对象的CRDT实现
package crdt

import (
    "sync"
    "time"
)

// WallID 墙体唯一标识
type WallID string

// WallState 墙体状态（LWW-Register实现）
type WallState struct {
    ID        WallID
    StartPoint Point3D
    EndPoint   Point3D
    Height     float64
    Thickness  float64
    Properties map[string]string
    
    // CRDT元数据
    Timestamp  int64      // 最后更新时间
    ActorID    string     // 最后修改者
    Version    VectorClock // 向量时钟
}

// WallOperation 墙体操作
type WallOperation struct {
    OpID      string
    WallID    WallID
    Type      OpType
    Timestamp int64
    ActorID   string
    VectorClock VectorClock
    
    // 操作数据
    Data      WallData
}

type OpType int

const (
    OpCreate OpType = iota
    OpUpdate
    OpDelete
)

// WallDocument 墙体文档（OR-Set实现）
type WallDocument struct {
    mu       sync.RWMutex
    walls    map[WallID]*WallState
    tombstone map[WallID]int64 // 删除标记
    history  []WallOperation   // 操作历史
    
    localActor string
    vectorClock VectorClock
}

// NewWallDocument 创建新的墙体文档
func NewWallDocument(actorID string) *WallDocument {
    return &WallDocument{
        walls:       make(map[WallID]*WallState),
        tombstone:   make(map[WallID]int64),
        history:     make([]WallOperation, 0),
        localActor:  actorID,
        vectorClock: NewVectorClock(),
    }
}

// ApplyLocalOperation 应用本地操作
func (wd *WallDocument) ApplyLocalOperation(op WallOperation) error {
    wd.mu.Lock()
    defer wd.mu.Unlock()
    
    // 1. 递增本地向量时钟
    wd.vectorClock = wd.vectorClock.Increment(wd.localActor)
    op.VectorClock = wd.vectorClock
    op.Timestamp = time.Now().UnixMilli()
    
    // 2. 应用到本地状态
    if err := wd.applyOperation(op); err != nil {
        return err
    }
    
    // 3. 记录历史
    wd.history = append(wd.history, op)
    
    return nil
}

// ApplyRemoteOperation 应用远程操作（核心CRDT逻辑）
func (wd *WallDocument) ApplyRemoteOperation(op WallOperation) error {
    wd.mu.Lock()
    defer wd.mu.Unlock()
    
    // 1. 更新向量时钟（取最大值）
    wd.vectorClock = wd.vectorClock.Merge(op.VectorClock)
    
    // 2. 检查操作是否已应用（幂等性）
    if wd.isOperationApplied(op.OpID) {
        return nil // 已应用，忽略
    }
    
    // 3. 应用操作
    if err := wd.applyOperation(op); err != nil {
        return err
    }
    
    // 4. 记录历史
    wd.history = append(wd.history, op)
    
    return nil
}

// applyOperation 内部操作应用
func (wd *WallDocument) applyOperation(op WallOperation) error {
    switch op.Type {
    case OpCreate:
        return wd.applyCreate(op)
    case OpUpdate:
        return wd.applyUpdate(op)
    case OpDelete:
        return wd.applyDelete(op)
    default:
        return fmt.Errorf("unknown operation type: %v", op.Type)
    }
}

// applyUpdate 应用更新操作（LWW语义）
func (wd *WallDocument) applyUpdate(op WallOperation) error {
    existing, exists := wd.walls[op.WallID]
    if !exists {
        return fmt.Errorf("wall not found: %s", op.WallID)
    }
    
    // LWW (Last-Write-Wins) 冲突解决
    // 比较向量时钟决定胜负
    cmp := existing.Version.Compare(op.VectorClock)
    
    if cmp < 0 {
        // 新操作胜出
        wd.walls[op.WallID] = &WallState{
            ID:         op.WallID,
            StartPoint: op.Data.StartPoint,
            EndPoint:   op.Data.EndPoint,
            Height:     op.Data.Height,
            Thickness:  op.Data.Thickness,
            Properties: mergeProperties(existing.Properties, op.Data.Properties),
            Timestamp:  op.Timestamp,
            ActorID:    op.ActorID,
            Version:    op.VectorClock,
        }
    }
    // 否则保留现有状态（旧操作）
    
    return nil
}

// GetState 获取当前文档状态
func (wd *WallDocument) GetState() map[WallID]*WallState {
    wd.mu.RLock()
    defer wd.mu.RUnlock()
    
    result := make(map[WallID]*WallState)
    for id, wall := range wd.walls {
        // 跳过已删除的墙体
        if _, deleted := wd.tombstone[id]; !deleted {
            result[id] = wall
        }
    }
    return result
}

// VectorClock 向量时钟实现
type VectorClock map[string]int64

func NewVectorClock() VectorClock {
    return make(map[string]int64)
}

func (vc VectorClock) Increment(actor string) VectorClock {
    newVC := make(VectorClock)
    for k, v := range vc {
        newVC[k] = v
    }
    newVC[actor] = vc[actor] + 1
    return newVC
}

func (vc VectorClock) Merge(other VectorClock) VectorClock {
    merged := make(VectorClock)
    
    // 合并所有actor
    allActors := make(map[string]bool)
    for k := range vc {
        allActors[k] = true
    }
    for k := range other {
        allActors[k] = true
    }
    
    // 取最大值
    for actor := range allActors {
        v1, ok1 := vc[actor]
        v2, ok2 := other[actor]
        
        if !ok1 {
            merged[actor] = v2
        } else if !ok2 {
            merged[actor] = v1
        } else {
            if v1 > v2 {
                merged[actor] = v1
            } else {
                merged[actor] = v2
            }
        }
    }
    
    return merged
}

// Compare 比较两个向量时钟
// 返回: -1 (vc < other), 0 (并发/相等), 1 (vc > other)
func (vc VectorClock) Compare(other VectorClock) int {
    dominates := false
    dominated := false
    
    allActors := make(map[string]bool)
    for k := range vc {
        allActors[k] = true
    }
    for k := range other {
        allActors[k] = true
    }
    
    for actor := range allActors {
        v1 := vc[actor]
        v2 := other[actor]
        
        if v1 > v2 {
            dominates = true
        } else if v2 > v1 {
            dominated = true
        }
    }
    
    if dominates && !dominated {
        return 1
    } else if !dominates && dominated {
        return -1
    } else {
        return 0 // 并发或相等
    }
}
```

### 2.3 WebSocket实时通信POC设计

#### 2.3.1 WebSocket架构设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                    WebSocket实时通信架构                             │
└─────────────────────────────────────────────────────────────────────┘

                              ┌─────────────┐
                              │  Load Balancer
                              │   (Nginx/HAProxy)
                              │  WebSocket支持
                              └──────┬──────┘
                                     │
           ┌─────────────────────────┼─────────────────────────┐
           │                         │                         │
           ▼                         ▼                         ▼
    ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
    │ 协作服务    │          │  协作服务   │          │  协作服务   │
    │ 实例 1      │          │   实例 2    │          │   实例 3    │
    │             │          │             │          │             │
    │ ┌─────────┐ │          │ ┌─────────┐ │          │ ┌─────────┐ │
    │ │WS Hub   │ │◀────────▶│ │WS Hub   │ │◀────────▶│ │WS Hub   │ │
    │ │(本地)   │ │  Redis   │ │(本地)   │ │  Redis   │ │(本地)   │ │
    │ └─────────┘ │ Pub/Sub  │ └─────────┘ │ Pub/Sub  │ └─────────┘ │
    │ ┌─────────┐ │          │ ┌─────────┐ │          │ ┌─────────┐ │
    │ │CRDT引擎 │ │          │ │CRDT引擎 │ │          │ │CRDT引擎 │ │
    │ └─────────┘ │          │ └─────────┘ │          │ └─────────┘ │
    └──────┬──────┘          └──────┬──────┘          └──────┬──────┘
           │                         │                         │
           └─────────────────────────┼─────────────────────────┘
                                     │
                              ┌──────┴──────┐
                              │  Redis Cluster
                              │  - 会话存储
                              │  - 消息广播
                              │  - 状态同步
                              └─────────────┘
```

#### 2.3.2 WebSocket Hub实现

```go
// websocket_hub.go - WebSocket连接管理
package websocket

import (
    "context"
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/redis/go-redis/v9"
)

// Hub 管理所有WebSocket连接
type Hub struct {
    // 本地连接管理
    clients    map[string]*Client        // userID -> Client
    rooms      map[string]map[string]bool // roomID -> set of userIDs
    
    // 通道
    register   chan *Client
    unregister chan *Client
    broadcast  chan Message
    
    // Redis用于跨实例通信
    redis      *redis.Client
    pubsub     *redis.PubSub
    
    // 配置
    config     HubConfig
    
    mu         sync.RWMutex
    ctx        context.Context
    cancel     context.CancelFunc
}

type HubConfig struct {
    MaxConnectionsPerUser int           // 每用户最大连接数
    MaxConnectionsPerRoom int           // 每房间最大连接数
    WriteTimeout          time.Duration // 写入超时
    PingInterval          time.Duration // 心跳间隔
    MessageBufferSize     int           // 消息缓冲区大小
}

// Client 表示一个WebSocket客户端
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    userID   string
    roomID   string
    send     chan []byte
    
    // 状态
    lastPing time.Time
    mu       sync.Mutex
}

// Message 消息结构
type Message struct {
    Type      string          `json:"type"`       // message, operation, presence
    RoomID    string          `json:"room_id"`
    UserID    string          `json:"user_id"`
    Timestamp int64           `json:"timestamp"`
    Payload   json.RawMessage `json:"payload"`
}

// NewHub 创建新的Hub
func NewHub(redisClient *redis.Client, config HubConfig) *Hub {
    ctx, cancel := context.WithCancel(context.Background())
    
    hub := &Hub{
        clients:    make(map[string]*Client),
        rooms:      make(map[string]map[string]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan Message, config.MessageBufferSize),
        redis:      redisClient,
        config:     config,
        ctx:        ctx,
        cancel:     cancel,
    }
    
    // 订阅Redis频道
    hub.pubsub = redisClient.Subscribe(ctx, "collaboration:broadcast")
    
    return hub
}

// Run 启动Hub主循环
func (h *Hub) Run() {
    // 启动Redis消息监听
    go h.handleRedisMessages()
    
    // 启动心跳检查
    go h.heartbeatChecker()
    
    for {
        select {
        case client := <-h.register:
            h.handleRegister(client)
            
        case client := <-h.unregister:
            h.handleUnregister(client)
            
        case message := <-h.broadcast:
            h.handleBroadcast(message)
            
        case <-h.ctx.Done():
            return
        }
    }
}

// handleRegister 处理客户端注册
func (h *Hub) handleRegister(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    // 检查用户是否已有连接
    if existing, ok := h.clients[client.userID]; ok {
        // 关闭旧连接
        close(existing.send)
        delete(h.clients, client.userID)
    }
    
    // 注册新连接
    h.clients[client.userID] = client
    
    // 加入房间
    if client.roomID != "" {
        if h.rooms[client.roomID] == nil {
            h.rooms[client.roomID] = make(map[string]bool)
        }
        h.rooms[client.roomID][client.userID] = true
    }
    
    // 发布用户上线事件到Redis
    h.publishPresence(client.roomID, client.userID, "online")
}

// handleBroadcast 处理消息广播
func (h *Hub) handleBroadcast(message Message) {
    h.mu.RLock()
    roomUsers, ok := h.rooms[message.RoomID]
    h.mu.RUnlock()
    
    if !ok {
        return
    }
    
    // 序列化消息
    data, err := json.Marshal(message)
    if err != nil {
        log.Printf("Failed to marshal message: %v", err)
        return
    }
    
    // 发送给房间内所有用户
    for userID := range roomUsers {
        // 跳过发送者
        if userID == message.UserID {
            continue
        }
        
        h.mu.RLock()
        client, ok := h.clients[userID]
        h.mu.RUnlock()
        
        if ok {
            select {
            case client.send <- data:
                // 发送成功
            default:
                // 客户端缓冲区满，关闭连接
                h.handleUnregister(client)
            }
        }
    }
    
    // 发布到Redis（跨实例广播）
    h.redis.Publish(h.ctx, "collaboration:broadcast", data)
}

// handleRedisMessages 处理Redis订阅消息
func (h *Hub) handleRedisMessages() {
    ch := h.pubsub.Channel()
    
    for msg := range ch {
        var message Message
        if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
            continue
        }
        
        // 只处理来自其他实例的消息
        h.handleBroadcast(message)
    }
}

// heartbeatChecker 心跳检查
func (h *Hub) heartbeatChecker() {
    ticker := time.NewTicker(h.config.PingInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            h.checkHeartbeats()
        case <-h.ctx.Done():
            return
        }
    }
}

// checkHeartbeats 检查所有连接的心跳
func (h *Hub) checkHeartbeats() {
    h.mu.RLock()
    clients := make([]*Client, 0, len(h.clients))
    for _, client := range h.clients {
        clients = append(clients, client)
    }
    h.mu.RUnlock()
    
    deadline := time.Now().Add(-h.config.PingInterval * 3)
    
    for _, client := range clients {
        client.mu.Lock()
        lastPing := client.lastPing
        client.mu.Unlock()
        
        if lastPing.Before(deadline) {
            // 心跳超时，关闭连接
            h.handleUnregister(client)
        } else {
            // 发送ping
            client.sendPing()
        }
    }
}

// Client方法

// readPump 读取循环
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.mu.Lock()
        c.lastPing = time.Now()
        c.mu.Unlock()
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })
    
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }
        
        // 处理消息
        c.handleMessage(message)
    }
}

// writePump 写入循环
func (c *Client) writePump() {
    ticker := time.NewTicker(c.hub.config.PingInterval)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteTimeout))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            c.conn.WriteMessage(websocket.TextMessage, message)
            
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteTimeout))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
            
        case <-c.hub.ctx.Done():
            return
        }
    }
}

// sendPing 发送ping
func (c *Client) sendPing() {
    select {
    case c.send <- []byte(`{"type":"ping"}`):
    default:
    }
}
```

### 2.4 事件驱动架构验证

#### 2.4.1 事件驱动架构设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                     事件驱动架构设计                                 │
└─────────────────────────────────────────────────────────────────────┘

    服务层                      消息层                      消费者
       │                          │                          │
       │  1. 产生事件              │                          │
       │ ──────────────────────▶  │                          │
       │                          │                          │
       │                          │  2. 路由到Topic           │
       │                          │  ┌────────────────┐      │
       │                          │  │ Kafka/NATS     │      │
       │                          │  │ - 持久化       │      │
       │                          │  │ - 分区         │      │
       │                          │  │ - 复制         │      │
       │                          │  └────────┬───────┘      │
       │                          │           │              │
       │                          │  3. 消费   │              │
       │                          │ ──────────┼──────────▶   │
       │                          │           │              │
       │                          │           ▼              │
       │                          │  ┌────────────────┐      │
       │                          │  │ 消费者组       │      │
       │                          │  │ - 协作服务     │      │
       │                          │  │ - 通知服务     │      │
       │                          │  │ - 审计服务     │      │
       │                          │  └────────────────┘      │

┌─────────────────────────────────────────────────────────────────────┐
│                        事件类型定义                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Domain Events (领域事件)                                           │
│  ├── WallCreated      - 墙体创建                                   │
│  ├── WallUpdated      - 墙体更新                                   │
│  ├── WallDeleted      - 墙体删除                                   │
│  ├── ElementTransformed - 元素变换                                 │
│  └── ProjectExported  - 项目导出                                   │
│                                                                     │
│  Integration Events (集成事件)                                      │
│  ├── UserJoinedRoom   - 用户加入房间                               │
│  ├── UserLeftRoom     - 用户离开房间                               │
│  ├── OperationApplied - 操作已应用                                 │
│  └── StateSynced      - 状态已同步                                 │
│                                                                     │
│  System Events (系统事件)                                           │
│  ├── ServiceStarted   - 服务启动                                   │
│  ├── HealthCheck      - 健康检查                                   │
│  └── ErrorOccurred    - 错误发生                                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 2.4.2 Kafka Topic设计

```yaml
# Kafka Topic配置
topics:
  # 协作事件 - 高吞吐量，多分区
  collaboration.operations:
    partitions: 12
    replication: 3
    retention: 7d
    compression: lz4
    
  # 项目事件 - 中等吞吐量
  project.events:
    partitions: 6
    replication: 3
    retention: 30d
    compression: snappy
    
  # 通知事件 - 低延迟
  notification.events:
    partitions: 3
    replication: 3
    retention: 1d
    compression: none
    
  # 审计日志 - 长期存储
  audit.logs:
    partitions: 6
    replication: 3
    retention: 365d
    compression: zstd
    
  # 死信队列
  dlq.events:
    partitions: 3
    replication: 3
    retention: 14d
```



---

## 3. 核心组件POC

### 3.1 协作服务POC

#### 3.1.1 POC目标

验证多用户并发编辑同一墙体的场景，确保：
- 操作无冲突合并
- 实时同步延迟<100ms
- 支持100+并发用户

#### 3.1.2 协作服务架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                      协作服务POC架构                                 │
└─────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                        协作服务                               │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
    │  │ WebSocket   │  │   CRDT      │  │   Session   │         │
    │  │   Handler   │──│   Engine    │──│   Manager   │         │
    │  └─────────────┘  └─────────────┘  └─────────────┘         │
    │         │                │                │                 │
    │         ▼                ▼                ▼                 │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
    │  │  Operation  │  │   State     │  │   Presence  │         │
    │  │   Router    │  │   Store     │  │   Tracker   │         │
    │  └─────────────┘  └─────────────┘  └─────────────┘         │
    │                                                             │
    │  ┌─────────────────────────────────────────────────────┐   │
    │  │              Event Publisher                        │   │
    │  │         (Kafka/NATS Integration)                    │   │
    │  └─────────────────────────────────────────────────────┘   │
    └─────────────────────────────────────────────────────────────┘
                              │
           ┌──────────────────┼──────────────────┐
           │                  │                  │
           ▼                  ▼                  ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │   Redis     │    │   Kafka     │    │ PostgreSQL  │
    │  (Session)  │    │  (Events)   │    │  (History)  │
    └─────────────┘    └─────────────┘    └─────────────┘
```

#### 3.1.3 多用户墙体编辑POC代码

```go
// collaboration_service_poc.go
package poc

import (
    "context"
    "sync"
    "testing"
    "time"
)

// CollaborationServicePOC 协作服务POC
type CollaborationServicePOC struct {
    crdtEngine    *CRDTEngine
    sessionMgr    *SessionManager
    wsHub         *WebSocketHub
    eventBus      EventBus
    
    // 指标收集
    metrics       *CollaborationMetrics
}

// CollaborationMetrics 协作指标
type CollaborationMetrics struct {
    mu                    sync.RWMutex
    OperationLatency      []time.Duration    // 操作延迟
    SyncLatency           []time.Duration    // 同步延迟
    ConflictCount         int                // 冲突次数
    ConcurrentUsers       int                // 并发用户数
    OperationsPerSecond   float64            // 每秒操作数
}

// MultiUserWallEditTest 多用户墙体编辑测试
func (poc *CollaborationServicePOC) MultiUserWallEditTest(ctx context.Context, config TestConfig) (*TestResult, error) {
    result := &TestResult{
        TestName: "MultiUserWallEdit",
        StartTime: time.Now(),
    }
    
    // 1. 创建协作房间
    roomID := "test-room-" + uuid.New().String()
    document := NewWallDocument("server")
    
    // 2. 模拟多个用户
    users := make([]*SimulatedUser, config.UserCount)
    for i := 0; i < config.UserCount; i++ {
        users[i] = &SimulatedUser{
            ID:       fmt.Sprintf("user-%d", i),
            Document: NewWallDocument(fmt.Sprintf("user-%d", i)),
            RoomID:   roomID,
        }
    }
    
    // 3. 启动用户操作模拟
    var wg sync.WaitGroup
    operationCount := config.OperationCount
    
    for _, user := range users {
        wg.Add(1)
        go func(u *SimulatedUser) {
            defer wg.Done()
            
            for i := 0; i < operationCount; i++ {
                select {
                case <-ctx.Done():
                    return
                default:
                }
                
                // 生成随机墙体操作
                op := u.GenerateRandomWallOperation()
                
                // 记录开始时间
                start := time.Now()
                
                // 应用本地操作
                if err := u.Document.ApplyLocalOperation(op); err != nil {
                    result.Errors = append(result.Errors, err)
                    continue
                }
                
                // 发送到服务器
                if err := poc.sendOperation(ctx, roomID, u.ID, op); err != nil {
                    result.Errors = append(result.Errors, err)
                    continue
                }
                
                // 记录本地延迟
                localLatency := time.Since(start)
                poc.metrics.AddOperationLatency(localLatency)
                
                // 等待随机间隔
                time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
            }
        }(user)
    }
    
    // 4. 启动广播接收
    broadcastWg := sync.WaitGroup{}
    for _, user := range users {
        broadcastWg.Add(1)
        go func(u *SimulatedUser) {
            defer broadcastWg.Done()
            
            receivedOps := 0
            expectedOps := (config.UserCount - 1) * operationCount
            
            timeout := time.After(config.TestDuration)
            
            for receivedOps < expectedOps {
                select {
                case op := <-poc.receiveOperation(ctx, u.RoomID, u.ID):
                    start := time.Now()
                    
                    // 应用远程操作
                    if err := u.Document.ApplyRemoteOperation(op); err != nil {
                        result.Errors = append(result.Errors, err)
                        continue
                    }
                    
                    // 记录同步延迟
                    syncLatency := time.Since(start)
                    poc.metrics.AddSyncLatency(syncLatency)
                    receivedOps++
                    
                case <-timeout:
                    return
                case <-ctx.Done():
                    return
                }
            }
        }(user)
    }
    
    // 5. 等待完成
    wg.Wait()
    time.Sleep(2 * time.Second) // 等待最后消息
    broadcastWg.Wait()
    
    // 6. 验证一致性
    result.ConsistencyCheck = poc.verifyConsistency(users)
    result.Metrics = poc.metrics.GetSnapshot()
    result.EndTime = time.Now()
    
    return result, nil
}

// verifyConsistency 验证所有用户文档一致性
func (poc *CollaborationServicePOC) verifyConsistency(users []*SimulatedUser) ConsistencyResult {
    result := ConsistencyResult{
        Passed: true,
    }
    
    if len(users) < 2 {
        return result
    }
    
    // 获取参考状态
    reference := users[0].Document.GetState()
    
    for i, user := range users[1:] {
        state := user.Document.GetState()
        
        // 比较墙体数量
        if len(reference) != len(state) {
            result.Passed = false
            result.Differences = append(result.Differences, 
                fmt.Sprintf("User %d: wall count mismatch (ref: %d, actual: %d)", 
                    i+1, len(reference), len(state)))
            continue
        }
        
        // 比较每个墙体
        for wallID, refWall := range reference {
            actualWall, ok := state[wallID]
            if !ok {
                result.Passed = false
                result.Differences = append(result.Differences,
                    fmt.Sprintf("User %d: missing wall %s", i+1, wallID))
                continue
            }
            
            // 比较墙体属性
            if !refWall.Equals(actualWall) {
                result.Passed = false
                result.Differences = append(result.Differences,
                    fmt.Sprintf("User %d: wall %s properties differ", i+1, wallID))
            }
        }
    }
    
    return result
}

// TestConfig 测试配置
type TestConfig struct {
    UserCount      int           // 用户数量
    OperationCount int           // 每个用户的操作数
    TestDuration   time.Duration // 测试持续时间
}

// TestResult 测试结果
type TestResult struct {
    TestName         string
    StartTime        time.Time
    EndTime          time.Time
    Metrics          MetricsSnapshot
    ConsistencyCheck ConsistencyResult
    Errors           []error
}

// ConsistencyResult 一致性检查结果
type ConsistencyResult struct {
    Passed      bool
    Differences []string
}
```

### 3.2 几何服务POC

#### 3.2.1 几何数据模型设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                      几何数据模型                                    │
└─────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                     GeometryDocument                         │
    │                    建筑设计文档                              │
    ├─────────────────────────────────────────────────────────────┤
    │  - id: string                                               │
    │  - version: int                                             │
    │  - elements: Map<ElementID, Element>                        │
    │  - layers: Layer[]                                          │
    │  - metadata: DocumentMetadata                               │
    └─────────────────────────────────────────────────────────────┘
                              │
           ┌──────────────────┼──────────────────┐
           │                  │                  │
           ▼                  ▼                  ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │    Wall     │    │   Element   │    │   Group     │
    │    (墙体)    │    │   (元素)    │    │   (编组)    │
    ├─────────────┤    ├─────────────┤    ├─────────────┤
    │ startPoint  │    │ transform   │    │ elements    │
    │ endPoint    │    │ geometry    │    │ transform   │
    │ height      │    │ material    │    │ name        │
    │ thickness   │    │ visibility  │    │             │
    │ material    │    │             │    │             │
    └─────────────┘    └─────────────┘    └─────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                      几何数据序列化                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. Protocol Buffers (推荐用于服务间通信)                           │
│     ✅ 高效二进制编码                                               │
│     ✅ 强类型，向后兼容                                             │
│     ✅ 代码生成                                                     │
│                                                                     │
│  2. FlatBuffers (推荐用于大模型传输)                                │
│     ✅ 零拷贝反序列化                                               │
│     ✅ 内存映射友好                                                 │
│     ✅ 适合BRep数据                                                 │
│                                                                     │
│  3. glTF 2.0 (推荐用于前端渲染)                                     │
│     ✅ 行业标准                                                     │
│     ✅ 高效网格传输                                                 │
│     ✅ 材质支持                                                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.2.2 几何服务Proto定义

```protobuf
// geometry.proto
syntax = "proto3";
package geometry;

// 几何服务
service GeometryService {
  // 文档操作
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  rpc UpdateDocument(UpdateDocumentRequest) returns (Document);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  
  // 元素操作
  rpc CreateElement(CreateElementRequest) returns (Element);
  rpc UpdateElement(UpdateElementRequest) returns (Element);
  rpc DeleteElement(DeleteElementRequest) returns (Empty);
  rpc GetElement(GetElementRequest) returns (Element);
  rpc ListElements(ListElementsRequest) returns (ElementList);
  
  // 几何计算
  rpc CalculateBoundingBox(CalculateBoundingBoxRequest) returns (BoundingBox);
  rpc GenerateMesh(GenerateMeshRequest) returns (Mesh);
  rpc BooleanOperation(BooleanOperationRequest) returns (Geometry);
  
  // 导入导出
  rpc ImportGeometry(ImportGeometryRequest) returns (Document);
  rpc ExportGeometry(ExportGeometryRequest) returns (ExportResult);
}

// 基础几何类型
message Point3D {
  double x = 1;
  double y = 2;
  double z = 3;
}

message Vector3D {
  double x = 1;
  double y = 2;
  double z = 3;
}

message Transform {
  // 4x4变换矩阵（列优先）
  repeated double matrix = 1;
}

// 墙体定义
message Wall {
  string id = 1;
  Point3D start_point = 2;
  Point3D end_point = 3;
  double height = 4;
  double thickness = 5;
  double base_elevation = 6;
  
  // 墙体属性
  WallType type = 7;
  Material material = 8;
  
  // 开口（门窗）
  repeated Opening openings = 9;
  
  // 元数据
  map<string, string> properties = 10;
}

enum WallType {
  WALL_TYPE_STANDARD = 0;
  WALL_TYPE_CURTAIN = 1;
  WALL_TYPE_RETAINING = 2;
  WALL_TYPE_SHEAR = 3;
}

message Opening {
  string id = 1;
  OpeningType type = 2;
  double width = 3;
  double height = 4;
  double sill_height = 5;
  double distance_from_start = 6;
}

enum OpeningType {
  OPENING_TYPE_DOOR = 0;
  OPENING_TYPE_WINDOW = 1;
}

// 网格数据
message Mesh {
  string id = 1;
  
  // 顶点数据
  repeated float vertices = 2;  // [x,y,z, x,y,z, ...]
  
  // 法线数据
  repeated float normals = 3;   // [nx,ny,nz, nx,ny,nz, ...]
  
  // UV坐标
  repeated float uvs = 4;       // [u,v, u,v, ...]
  
  // 索引数据
  repeated uint32 indices = 5;  // 三角形索引
  
  // 材质
  Material material = 6;
  
  // 包围盒
  BoundingBox bounding_box = 7;
}

// 材质定义
message Material {
  string id = 1;
  string name = 2;
  
  // PBR材质属性
  Color base_color = 3;
  float metallic = 4;
  float roughness = 5;
  float opacity = 6;
  
  // 纹理
  string base_color_texture = 7;
  string normal_texture = 8;
  string metallic_roughness_texture = 9;
}

message Color {
  float r = 1;
  float g = 2;
  float b = 3;
  float a = 4;
}

// 包围盒
message BoundingBox {
  Point3D min = 1;
  Point3D max = 2;
  Point3D center = 3;
}

// 文档定义
message Document {
  string id = 1;
  string name = 2;
  int32 version = 3;
  repeated Element elements = 4;
  repeated Layer layers = 5;
  DocumentMetadata metadata = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
}

message Element {
  string id = 1;
  ElementType type = 2;
  string name = 3;
  Transform transform = 4;
  oneof geometry {
    Wall wall = 10;
    Mesh mesh = 11;
    BRep brep = 12;
  }
  string layer_id = 20;
  bool visible = 21;
  Material material = 22;
  map<string, string> properties = 30;
}

enum ElementType {
  ELEMENT_TYPE_WALL = 0;
  ELEMENT_TYPE_FLOOR = 1;
  ELEMENT_TYPE_ROOF = 2;
  ELEMENT_TYPE_COLUMN = 3;
  ELEMENT_TYPE_BEAM = 4;
  ELEMENT_TYPE_DOOR = 5;
  ELEMENT_TYPE_WINDOW = 6;
  ELEMENT_TYPE_FURNITURE = 7;
  ELEMENT_TYPE_MESH = 100;
  ELEMENT_TYPE_BREP = 101;
}

// BRep几何（边界表示）
message BRep {
  string id = 1;
  bytes data = 2;  // OpenCASCADE/OpenNURBS序列化数据
  string format = 3;  // "opencascade", "opennurbs", "step"
}
```

#### 3.2.3 几何数据传输优化

```go
// geometry_transport.go - 几何数据传输优化
package geometry

import (
    "bytes"
    "compress/gzip"
    "io"
)

// GeometryCompressor 几何数据压缩器
type GeometryCompressor struct {
    level int
}

func NewGeometryCompressor(level int) *GeometryCompressor {
    return &GeometryCompressor{level: level}
}

// CompressMesh 压缩网格数据
func (gc *GeometryCompressor) CompressMesh(mesh *Mesh) ([]byte, error) {
    // 1. 使用Draco压缩（如果可用）
    // 2. 回退到gzip压缩
    
    data, err := proto.Marshal(mesh)
    if err != nil {
        return nil, err
    }
    
    return gc.compress(data)
}

// compress gzip压缩
func (gc *GeometryCompressor) compress(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    writer, err := gzip.NewWriterLevel(&buf, gc.level)
    if err != nil {
        return nil, err
    }
    
    if _, err := writer.Write(data); err != nil {
        return nil, err
    }
    
    if err := writer.Close(); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}

// GeometryLODManager LOD（细节层次）管理器
type GeometryLODManager struct {
    levels map[int]LODConfig
}

type LODConfig struct {
    Level           int
    MaxTriangles    int
    ErrorThreshold  float64
    Simplification  float64
}

// GenerateLOD 生成不同细节层次的几何数据
func (lm *GeometryLODManager) GenerateLOD(mesh *Mesh, targetLevel int) (*Mesh, error) {
    config, ok := lm.levels[targetLevel]
    if !ok {
        return nil, fmt.Errorf("LOD level %d not configured", targetLevel)
    }
    
    // 获取当前三角形数量
    currentTriangles := len(mesh.Indices) / 3
    
    if currentTriangles <= config.MaxTriangles {
        // 不需要简化
        return mesh, nil
    }
    
    // 使用网格简化算法
    // 这里可以集成meshoptimizer等库
    simplifiedMesh, err := lm.simplifyMesh(mesh, config.Simplification)
    if err != nil {
        return nil, err
    }
    
    return simplifiedMesh, nil
}

// StreamingGeometryProvider 流式几何数据提供器
// 用于大模型的分块传输
type StreamingGeometryProvider struct {
    chunkSize int  // 每个块的大小（字节）
}

// GetGeometryStream 获取几何数据流
func (sgp *StreamingGeometryProvider) GetGeometryStream(documentID string, elementIDs []string) (<-chan GeometryChunk, error) {
    chunkChan := make(chan GeometryChunk, 10)
    
    go func() {
        defer close(chunkChan)
        
        for _, elementID := range elementIDs {
            // 获取元素几何数据
            geometry, err := sgp.fetchGeometry(documentID, elementID)
            if err != nil {
                chunkChan <- GeometryChunk{
                    Error: err,
                }
                continue
            }
            
            // 分块发送
            chunks := sgp.splitIntoChunks(geometry)
            for i, chunk := range chunks {
                chunkChan <- GeometryChunk{
                    ElementID:   elementID,
                    ChunkIndex:  i,
                    TotalChunks: len(chunks),
                    Data:        chunk,
                }
            }
        }
    }()
    
    return chunkChan, nil
}

type GeometryChunk struct {
    ElementID   string
    ChunkIndex  int
    TotalChunks int
    Data        []byte
    Error       error
}
```

### 3.3 脚本调度服务POC

#### 3.3.1 脚本调度架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                     脚本调度服务架构                                 │
└─────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                      API Layer                               │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
    │  │  Script     │  │   Task      │  │   Queue     │         │
    │  │  Upload     │  │   Submit    │  │   Status    │         │
    │  └─────────────┘  └─────────────┘  └─────────────┘         │
    └─────────────────────────────────────────────────────────────┘
                              │
                              ▼
    ┌─────────────────────────────────────────────────────────────┐
    │                    Scheduler Core                            │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
    │  │   Task      │  │  Resource   │  │   Priority  │         │
    │  │   Queue     │  │  Manager    │  │   Queue     │         │
    │  │  (Redis)    │  │             │  │             │         │
    │  └─────────────┘  └─────────────┘  └─────────────┘         │
    └─────────────────────────────────────────────────────────────┘
                              │
           ┌──────────────────┼──────────────────┐
           │                  │                  │
           ▼                  ▼                  ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │  Worker     │    │   Worker    │    │   Worker    │
    │  Pool 1     │    │   Pool 2    │    │   Pool 3    │
    │  (Python)   │    │  (C#)       │    │  (Go)       │
    │             │    │             │    │             │
    │ - Rhino     │    │ - Revit     │    │ - Geometry  │
    │ - Grasshopper│   │ - Dynamo    │    │ - Analysis  │
    └─────────────┘    └─────────────┘    └─────────────┘
```

#### 3.3.2 脚本调度服务实现

```go
// script_scheduler.go
package scheduler

import (
    "context"
    "fmt"
    "time"
)

// ScriptScheduler 脚本调度器
type ScriptScheduler struct {
    taskQueue      TaskQueue
    workerPools    map[string]WorkerPool
    resourceMgr    ResourceManager
    eventBus       EventBus
    metrics        SchedulerMetrics
}

// Task 任务定义
type Task struct {
    ID            string
    Type          TaskType
    Priority      int           // 1-10, 10最高
    Script        ScriptInfo
    Input         TaskInput
    Resources     ResourceRequirements
    Timeout       time.Duration
    RetryPolicy   RetryPolicy
    
    // 状态
    Status        TaskStatus
    CreatedAt     time.Time
    StartedAt     *time.Time
    CompletedAt   *time.Time
    Result        *TaskResult
    Error         error
}

type TaskType string

const (
    TaskTypeGeometryGeneration TaskType = "geometry_generation"
    TaskTypeAnalysis           TaskType = "analysis"
    TaskTypeOptimization       TaskType = "optimization"
    TaskTypeExport             TaskType = "export"
    TaskTypeValidation         TaskType = "validation"
)

type TaskStatus string

const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusQueued     TaskStatus = "queued"
    TaskStatusRunning    TaskStatus = "running"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusFailed     TaskStatus = "failed"
    TaskStatusCancelled  TaskStatus = "cancelled"
    TaskStatusTimeout    TaskStatus = "timeout"
)

// ScriptInfo 脚本信息
type ScriptInfo struct {
    ID          string
    Name        string
    Runtime     ScriptRuntime
    Code        string
    Dependencies []string
    EntryPoint  string
}

type ScriptRuntime string

const (
    RuntimePython    ScriptRuntime = "python"
    RuntimeCSharp    ScriptRuntime = "csharp"
    RuntimeGrasshopper ScriptRuntime = "grasshopper"
    RuntimeDynamo    ScriptRuntime = "dynamo"
)

// ResourceRequirements 资源需求
type ResourceRequirements struct {
    CPU        float64       // CPU核心数
    Memory     int64         // 内存(MB)
    GPU        bool          // 是否需要GPU
    DiskSpace  int64         // 磁盘空间(MB)
    Timeout    time.Duration // 超时时间
}

// SubmitTask 提交任务
func (s *ScriptScheduler) SubmitTask(ctx context.Context, task *Task) (*Task, error) {
    // 1. 验证任务
    if err := s.validateTask(task); err != nil {
        return nil, fmt.Errorf("task validation failed: %w", err)
    }
    
    // 2. 生成任务ID
    task.ID = generateTaskID()
    task.Status = TaskStatusPending
    task.CreatedAt = time.Now()
    
    // 3. 计算优先级分数
    priorityScore := s.calculatePriorityScore(task)
    
    // 4. 加入队列
    if err := s.taskQueue.Enqueue(ctx, task, priorityScore); err != nil {
        return nil, fmt.Errorf("failed to enqueue task: %w", err)
    }
    
    task.Status = TaskStatusQueued
    
    // 5. 发布任务提交事件
    s.eventBus.Publish(TaskSubmittedEvent{
        TaskID:   task.ID,
        Type:     task.Type,
        Priority: task.Priority,
    })
    
    return task, nil
}

// Start 启动调度器
func (s *ScriptScheduler) Start(ctx context.Context) error {
    // 1. 启动任务调度循环
    go s.scheduleLoop(ctx)
    
    // 2. 启动worker池
    for runtime, pool := range s.workerPools {
        if err := pool.Start(ctx); err != nil {
            return fmt.Errorf("failed to start worker pool for %s: %w", runtime, err)
        }
    }
    
    // 3. 启动资源监控
    go s.resourceMonitor(ctx)
    
    return nil
}

// scheduleLoop 调度循环
func (s *ScriptScheduler) scheduleLoop(ctx context.Context) {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.processNextTask(ctx)
        }
    }
}

// processNextTask 处理下一个任务
func (s *ScriptScheduler) processNextTask(ctx context.Context) {
    // 1. 获取可用资源
    availableResources := s.resourceMgr.GetAvailableResources()
    
    // 2. 从队列获取任务
    task, err := s.taskQueue.Dequeue(ctx, availableResources)
    if err != nil {
        if err != ErrQueueEmpty {
            log.Printf("Failed to dequeue task: %v", err)
        }
        return
    }
    
    // 3. 获取对应的worker池
    pool, ok := s.workerPools[task.Script.Runtime]
    if !ok {
        s.failTask(task, fmt.Errorf("no worker pool for runtime: %s", task.Script.Runtime))
        return
    }
    
    // 4. 分配worker
    worker, err := pool.AcquireWorker(ctx, task.Resources)
    if err != nil {
        // 资源不足，重新入队
        s.taskQueue.Requeue(ctx, task)
        return
    }
    
    // 5. 执行任务
    go s.executeTask(ctx, task, worker)
}

// executeTask 执行任务
func (s *ScriptScheduler) executeTask(ctx context.Context, task *Task, worker Worker) {
    defer worker.Release()
    
    // 更新任务状态
    task.Status = TaskStatusRunning
    now := time.Now()
    task.StartedAt = &now
    
    // 创建任务上下文（带超时）
    taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
    defer cancel()
    
    // 执行脚本
    result, err := worker.Execute(taskCtx, task)
    
    // 处理结果
    if err != nil {
        if taskCtx.Err() == context.DeadlineExceeded {
            task.Status = TaskStatusTimeout
        } else {
            task.Status = TaskStatusFailed
        }
        task.Error = err
        
        // 重试逻辑
        if s.shouldRetry(task) {
            s.retryTask(task)
            return
        }
    } else {
        task.Status = TaskStatusCompleted
        task.Result = result
    }
    
    completedAt := time.Now()
    task.CompletedAt = &completedAt
    
    // 发布任务完成事件
    s.eventBus.Publish(TaskCompletedEvent{
        TaskID: task.ID,
        Status: task.Status,
        Duration: completedAt.Sub(*task.StartedAt),
    })
    
    // 更新指标
    s.metrics.RecordTaskCompletion(task)
}

// WorkerPool Worker池接口
type WorkerPool interface {
    Start(ctx context.Context) error
    Stop() error
    AcquireWorker(ctx context.Context, requirements ResourceRequirements) (Worker, error)
    GetStats() PoolStats
}

// Worker 工作器接口
type Worker interface {
    Execute(ctx context.Context, task *Task) (*TaskResult, error)
    Release()
    GetID() string
    GetRuntime() ScriptRuntime
}

// PythonWorkerPool Python Worker池实现
type PythonWorkerPool struct {
    workers     chan *PythonWorker
    maxWorkers  int
    scriptPath  string
    sandboxConfig SandboxConfig
}

type PythonWorker struct {
    id       string
    pool     *PythonWorkerPool
    executor *PythonExecutor
    busy     bool
    mu       sync.Mutex
}

func (w *PythonWorker) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    w.mu.Lock()
    w.busy = true
    w.mu.Unlock()
    
    defer func() {
        w.mu.Lock()
        w.busy = false
        w.mu.Unlock()
    }()
    
    // 准备执行环境
    env := w.prepareEnvironment(task)
    
    // 执行脚本
    output, err := w.executor.Run(ctx, task.Script.Code, env, task.Input)
    if err != nil {
        return nil, err
    }
    
    // 解析结果
    result, err := w.parseResult(output)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

### 3.4 API网关POC

#### 3.4.1 API网关架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                      API网关架构                                     │
└─────────────────────────────────────────────────────────────────────┘

                              ┌─────────────┐
                              │   Clients   │
                              │  Web/Mobile │
                              └──────┬──────┘
                                     │
                              ┌──────┴──────┐
                              │  CDN/WAF    │
                              └──────┬──────┘
                                     │
    ┌────────────────────────────────┼────────────────────────────────┐
    │                           API Gateway                           │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
    │  │   Routing   │  │   Auth      │  │   Rate      │             │
    │  │   Engine    │──│   Middleware│──│   Limiter   │             │
    │  └─────────────┘  └─────────────┘  └─────────────┘             │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
    │  │   Load      │  │   Circuit   │  │   Request   │             │
    │  │   Balancer  │──│   Breaker   │──│   Transform │             │
    │  └─────────────┘  └─────────────┘  └─────────────┘             │
    └────────────────────────────────┼────────────────────────────────┘
                                     │
           ┌─────────────────────────┼─────────────────────────┐
           │                         │                         │
           ▼                         ▼                         ▼
    ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
    │ 协作服务    │          │  几何服务   │          │  脚本服务   │
    │ gRPC/HTTP   │          │  gRPC/HTTP  │          │  gRPC/HTTP  │
    └─────────────┘          └─────────────┘          └─────────────┘
```

#### 3.4.2 Kong API网关配置示例

```yaml
# kong.yml - Kong网关配置
_format_version: "3.0"

services:
  # 协作服务
  - name: collaboration-service
    url: http://collaboration-service:8080
    routes:
      - name: collaboration-routes
        paths:
          - /api/v1/collaboration
        strip_path: false
    plugins:
      - name: rate-limiting
        config:
          minute: 1000
          policy: redis
          redis_host: redis
      - name: jwt
        config:
          uri_param_names: []
          cookie_names: []
          key_claim_name: iss
          secret_is_base64: false
          claims_to_verify:
            - exp
      - name: cors
        config:
          origins:
            - "https://app.example.com"
          methods:
            - GET
            - POST
            - PUT
            - DELETE
            - PATCH
          headers:
            - Authorization
            - Content-Type
          max_age: 3600

  # 几何服务
  - name: geometry-service
    url: http://geometry-service:8080
    routes:
      - name: geometry-routes
        paths:
          - /api/v1/geometry
        strip_path: false
    plugins:
      - name: rate-limiting
        config:
          minute: 500
      - name: request-transformer
        config:
          add:
            headers:
              - X-Service-Name:geometry

  # 脚本服务
  - name: script-service
    url: http://script-service:8080
    routes:
      - name: script-routes
        paths:
          - /api/v1/scripts
        strip_path: false
    plugins:
      - name: rate-limiting
        config:
          minute: 100
      - name: request-size-limiting
        config:
          allowed_payload_size: 50  # MB

  # WebSocket服务
  - name: websocket-service
    url: http://collaboration-service:8081
    protocol: ws
    routes:
      - name: websocket-routes
        paths:
          - /ws
        strip_path: false

upstreams:
  - name: collaboration-upstream
    targets:
      - target: collaboration-service-1:8080
        weight: 100
      - target: collaboration-service-2:8080
        weight: 100
      - target: collaboration-service-3:8080
        weight: 100
    healthchecks:
      active:
        healthy:
          interval: 10
          successes: 2
        unhealthy:
          interval: 10
          http_failures: 3
        http_path: /health
```



---

## 4. 性能验证方案

### 4.1 性能目标定义

| 指标类别 | 指标名称 | 目标值 | 测量方法 |
|----------|----------|--------|----------|
| 响应时间 | 本地操作反馈 | <16ms | 客户端测量 |
| 响应时间 | 远程广播延迟 | <100ms | 端到端测量 |
| 并发能力 | 每项目并发用户 | 100+ | 压力测试 |
| 吞吐量 | 每秒操作数 | 1000+ | 负载测试 |
| 资源使用 | CPU使用率 | <70% | 监控 |
| 资源使用 | 内存使用 | <4GB/实例 | 监控 |

### 4.2 响应时间测试方案

#### 4.2.1 测试架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                    响应时间测试架构                                  │
└─────────────────────────────────────────────────────────────────────┘

    ┌─────────────┐                    ┌─────────────┐
    │  Test       │                    │   Test      │
    │  Client 1   │◀──────────────────▶│   Client N  │
    │  (k6)       │   同步时间(NTP)    │   (k6)      │
    └──────┬──────┘                    └──────┬──────┘
           │                                  │
           │  1. 发送操作                      │
           │ ────────────────────────────────▶│
           │                                  │
           │  2. 接收广播                      │
           │ ◀────────────────────────────────│
           │                                  │
           │  3. 计算延迟                      │
           │  Latency = T_receive - T_send    │
           │                                  │

┌─────────────────────────────────────────────────────────────────────┐
│                        测试工具栈                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  负载生成: k6 (https://k6.io)                                       │
│  ├── 支持WebSocket测试                                              │
│  ├── JavaScript脚本                                                 │
│  ├── 分布式执行                                                     │
│  └── 实时指标收集                                                   │
│                                                                     │
│  监控: Prometheus + Grafana                                         │
│  ├── 服务指标采集                                                   │
│  ├── 自定义业务指标                                                 │
│  └── 可视化仪表盘                                                   │
│                                                                     │
│  追踪: Jaeger/Zipkin                                                │
│  ├── 分布式追踪                                                     │
│  ├── 性能瓶颈定位                                                   │
│  └── 调用链分析                                                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 4.2.2 k6测试脚本

```javascript
// performance_test.js - k6性能测试脚本
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter, Gauge } from 'k6/metrics';
import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// 自定义指标
const localFeedbackLatency = new Trend('local_feedback_latency');
const broadcastLatency = new Trend('broadcast_latency');
const operationSuccessRate = new Rate('operation_success_rate');
const activeConnections = new Gauge('active_connections');
const operationsSent = new Counter('operations_sent');
const operationsReceived = new Counter('operations_received');

// 测试配置
export const options = {
  scenarios: {
    // 渐进式负载测试
    ramp_up: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 20 },   // 预热
        { duration: '5m', target: 50 },   // 正常负载
        { duration: '5m', target: 100 },  // 峰值负载
        { duration: '2m', target: 0 },    // 冷却
      ],
      gracefulRampDown: '30s',
    },
    // 压力测试
    stress_test: {
      executor: 'constant-vus',
      vus: 100,
      duration: '10m',
      startTime: '15m',  // 在ramp_up之后开始
    },
  },
  thresholds: {
    // 性能阈值
    'local_feedback_latency': ['p(95)<16'],      // 95%本地反馈<16ms
    'broadcast_latency': ['p(95)<100'],          // 95%广播延迟<100ms
    'operation_success_rate': ['rate>0.99'],     // 99%成功率
    http_req_duration: ['p(95)<50'],             // HTTP请求<50ms
  },
};

const WS_URL = __ENV.WS_URL || 'ws://localhost:8081/ws';
const ROOM_ID = __ENV.ROOM_ID || 'test-room';

export default function () {
  const userId = `user-${__VU}-${randomString(8)}`;
  const operations = [];
  const pendingOperations = new Map();
  
  const res = ws.connect(WS_URL, null, function (socket) {
    activeConnections.add(1);
    
    socket.on('open', function () {
      console.log(`User ${userId} connected`);
      
      // 加入房间
      socket.send(JSON.stringify({
        type: 'join',
        room_id: ROOM_ID,
        user_id: userId,
      }));
      
      // 开始发送操作
      startSendingOperations(socket, userId, pendingOperations);
    });
    
    socket.on('message', function (message) {
      const data = JSON.parse(message);
      const receiveTime = Date.now();
      
      switch (data.type) {
        case 'operation_ack':
          // 本地操作确认
          const localSendTime = pendingOperations.get(data.operation_id);
          if (localSendTime) {
            const latency = receiveTime - localSendTime;
            localFeedbackLatency.add(latency);
            pendingOperations.delete(data.operation_id);
            operationSuccessRate.add(1);
          }
          break;
          
        case 'remote_operation':
          // 远程操作广播
          operationsReceived.add(1);
          const originalSendTime = data.original_timestamp;
          if (originalSendTime) {
            const latency = receiveTime - originalSendTime;
            broadcastLatency.add(latency);
          }
          break;
          
        case 'error':
          console.error(`Error: ${data.message}`);
          operationSuccessRate.add(0);
          break;
      }
    });
    
    socket.on('close', function () {
      activeConnections.add(-1);
      console.log(`User ${userId} disconnected`);
    });
    
    socket.on('error', function (e) {
      console.error(`WebSocket error: ${e.error()}`);
      operationSuccessRate.add(0);
    });
    
    // 测试持续时间
    sleep(randomIntBetween(30, 60));
    socket.close();
  });
  
  check(res, { 'WebSocket connection successful': (r) => r && r.status === 101 });
}

function startSendingOperations(socket, userId, pendingOperations) {
  // 每100-500ms发送一个操作
  const interval = setInterval(() => {
    const operation = generateWallOperation(userId);
    const sendTime = Date.now();
    
    pendingOperations.set(operation.id, sendTime);
    
    socket.send(JSON.stringify({
      type: 'operation',
      operation_id: operation.id,
      timestamp: sendTime,
      payload: operation,
    }));
    
    operationsSent.add(1);
  }, randomIntBetween(100, 500));
  
  // 清理定时器
  socket.on('close', () => clearInterval(interval));
}

function generateWallOperation(userId) {
  return {
    id: `op-${randomString(16)}`,
    type: 'wall_update',
    wall_id: `wall-${randomIntBetween(1, 100)}`,
    data: {
      start_point: {
        x: randomIntBetween(0, 1000),
        y: randomIntBetween(0, 1000),
        z: 0,
      },
      end_point: {
        x: randomIntBetween(0, 1000),
        y: randomIntBetween(0, 1000),
        z: 0,
      },
      height: randomIntBetween(200, 400),
      thickness: randomIntBetween(10, 30),
    },
    user_id: userId,
  };
}
```

### 4.3 并发用户测试方案

#### 4.3.1 测试场景设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                    并发用户测试场景                                  │
└─────────────────────────────────────────────────────────────────────┘

场景1: 基础并发测试
├── 用户数量: 10, 50, 100, 200
├── 持续时间: 10分钟
├── 操作频率: 每个用户2-5次/秒
└── 验证目标: 系统稳定性，无崩溃

场景2: 峰值负载测试
├── 用户数量: 100 → 200 → 300（阶梯增长）
├── 持续时间: 每阶梯5分钟
├── 操作频率: 每个用户5次/秒
└── 验证目标: 性能衰减曲线

场景3: 突发流量测试
├── 基础用户: 50
├── 突发用户: 额外100用户（1分钟内加入）
├── 持续时间: 突发后持续5分钟
└── 验证目标: 弹性伸缩能力

场景4: 长时间稳定性测试
├── 用户数量: 100
├── 持续时间: 24小时
├── 操作频率: 每个用户1次/秒
└── 验证目标: 内存泄漏，连接稳定性

场景5: 混合负载测试
├── 协作编辑用户: 60%
├── 只读查看用户: 30%
├── 脚本执行用户: 10%
├── 总用户数: 100
└── 验证目标: 多场景混合下的性能
```

#### 4.3.2 并发测试执行计划

```yaml
# concurrent_test_plan.yaml
phases:
  - name: warmup
    duration: 2m
    users: 10
    ramp_up: 30s
    
  - name: normal_load
    duration: 5m
    users: 50
    operations_per_second: 200
    
  - name: peak_load
    duration: 5m
    users: 100
    operations_per_second: 500
    
  - name: stress_test
    duration: 3m
    users: 200
    operations_per_second: 1000
    
  - name: cooldown
    duration: 2m
    users: 0

success_criteria:
  - metric: p95_latency
    threshold: 100ms
    
  - metric: error_rate
    threshold: 1%
    
  - metric: connection_drop_rate
    threshold: 0.1%
    
  - metric: memory_growth
    threshold: 10MB/hour
```

### 4.4 几何数据传输性能测试

#### 4.4.1 测试数据规格

| 模型类型 | 墙体数量 | 顶点数 | 数据大小 | 适用场景 |
|----------|----------|--------|----------|----------|
| 小型公寓 | 20-30 | 10K | 1-2MB | 单元测试 |
| 中型住宅 | 50-100 | 100K | 10-20MB | 功能测试 |
| 大型商业 | 200-500 | 1M | 100-200MB | 压力测试 |
| 超大型项目 | 1000+ | 10M+ | 1GB+ | 极限测试 |

#### 4.4.2 传输性能测试脚本

```go
// geometry_transport_test.go
package performance

import (
    "context"
    "testing"
    "time"
)

// GeometryTransportBenchmark 几何数据传输基准测试
func BenchmarkGeometryTransport(b *testing.B) {
    testCases := []struct {
        name        string
        wallCount   int
        vertexCount int
    }{
        {"Small_Apartment", 30, 10000},
        {"Medium_House", 100, 100000},
        {"Large_Commercial", 500, 1000000},
    }
    
    for _, tc := range testCases {
        b.Run(tc.name, func(b *testing.B) {
            // 生成测试数据
            document := generateTestDocument(tc.wallCount, tc.vertexCount)
            
            b.Run("Protobuf_Serialization", func(b *testing.B) {
                for i := 0; i < b.N; i++ {
                    data, err := proto.Marshal(document)
                    if err != nil {
                        b.Fatal(err)
                    }
                    b.SetBytes(int64(len(data)))
                }
            })
            
            b.Run("Protobuf_With_Compression", func(b *testing.B) {
                compressor := NewGeometryCompressor(gzip.BestSpeed)
                for i := 0; i < b.N; i++ {
                    data, err := compressor.CompressMesh(document.ToMesh())
                    if err != nil {
                        b.Fatal(err)
                    }
                    b.SetBytes(int64(len(data)))
                }
            })
            
            b.Run("Streaming_Transfer", func(b *testing.B) {
                provider := &StreamingGeometryProvider{chunkSize: 64 * 1024}
                for i := 0; i < b.N; i++ {
                    elementIDs := getAllElementIDs(document)
                    chunkChan, err := provider.GetGeometryStream(document.Id, elementIDs)
                    if err != nil {
                        b.Fatal(err)
                    }
                    
                    totalBytes := 0
                    for chunk := range chunkChan {
                        if chunk.Error != nil {
                            b.Fatal(chunk.Error)
                        }
                        totalBytes += len(chunk.Data)
                    }
                    b.SetBytes(int64(totalBytes))
                }
            })
        })
    }
}

// TransferLatencyTest 传输延迟测试
func TestTransferLatency(t *testing.T) {
    ctx := context.Background()
    
    testCases := []struct {
        name         string
        dataSize     int64
        maxLatency   time.Duration
    }{
        {"1MB_Model", 1 * 1024 * 1024, 500 * time.Millisecond},
        {"10MB_Model", 10 * 1024 * 1024, 2 * time.Second},
        {"100MB_Model", 100 * 1024 * 1024, 10 * time.Second},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // 生成测试数据
            data := make([]byte, tc.dataSize)
            rand.Read(data)
            
            // 测量传输时间
            start := time.Now()
            
            // 模拟传输
            err := transferData(ctx, data)
            if err != nil {
                t.Fatalf("Transfer failed: %v", err)
            }
            
            latency := time.Since(start)
            
            t.Logf("Transfer latency: %v (data size: %d MB)", 
                latency, tc.dataSize/1024/1024)
            
            if latency > tc.maxLatency {
                t.Errorf("Latency %v exceeds threshold %v", latency, tc.maxLatency)
            }
            
            // 计算吞吐量
            throughput := float64(tc.dataSize) / latency.Seconds() / 1024 / 1024
            t.Logf("Throughput: %.2f MB/s", throughput)
        })
    }
}
```

---

## 5. 集成验证方案

### 5.1 服务间通信验证

#### 5.1.1 gRPC通信验证

```go
// grpc_integration_test.go
package integration

import (
    "context"
    "testing"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// TestGRPCServiceCommunication gRPC服务通信测试
func TestGRPCServiceCommunication(t *testing.T) {
    ctx := context.Background()
    
    // 1. 建立连接
    conn, err := grpc.Dial(
        "collaboration-service:8080",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(100*1024*1024),
            grpc.MaxCallSendMsgSize(100*1024*1024),
        ),
    )
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()
    
    client := pb.NewCollaborationServiceClient(conn)
    
    t.Run("Unary_Call", func(t *testing.T) {
        req := &pb.GetDocumentStateRequest{
            DocumentId: "test-doc-1",
        }
        
        start := time.Now()
        resp, err := client.GetDocumentState(ctx, req)
        latency := time.Since(start)
        
        if err != nil {
            t.Fatalf("RPC failed: %v", err)
        }
        
        t.Logf("Unary call latency: %v", latency)
        
        if latency > 50*time.Millisecond {
            t.Errorf("Latency %v exceeds 50ms threshold", latency)
        }
        
        if resp.DocumentId != req.DocumentId {
            t.Errorf("Document ID mismatch")
        }
    })
    
    t.Run("Streaming_Call", func(t *testing.T) {
        stream, err := client.SubscribeChanges(ctx, &pb.SubscribeRequest{
            DocumentId: "test-doc-1",
            UserId:     "test-user",
        })
        if err != nil {
            t.Fatalf("Failed to subscribe: %v", err)
        }
        
        // 接收变更事件
        eventCount := 0
        timeout := time.After(5 * time.Second)
        
        for {
            select {
            case <-timeout:
                t.Logf("Received %d events", eventCount)
                return
            default:
                event, err := stream.Recv()
                if err != nil {
                    t.Logf("Stream ended: %v", err)
                    return
                }
                eventCount++
                t.Logf("Received event: %s", event.Type)
            }
        }
    })
    
    t.Run("Bidirectional_Streaming", func(t *testing.T) {
        stream, err := client.SendOperation(ctx)
        if err != nil {
            t.Fatalf("Failed to create stream: %v", err)
        }
        
        // 发送操作
        go func() {
            for i := 0; i < 10; i++ {
                op := &pb.Operation{
                    OperationId: fmt.Sprintf("op-%d", i),
                    UserId:      "test-user",
                    DocumentId:  "test-doc-1",
                    Timestamp:   time.Now().UnixMilli(),
                }
                
                if err := stream.Send(op); err != nil {
                    t.Logf("Send error: %v", err)
                    return
                }
                
                time.Sleep(100 * time.Millisecond)
            }
            stream.CloseSend()
        }()
        
        // 接收确认
        ackCount := 0
        for {
            ack, err := stream.Recv()
            if err != nil {
                break
            }
            ackCount++
            t.Logf("Received ack: %s", ack.OperationId)
        }
        
        if ackCount != 10 {
            t.Errorf("Expected 10 acks, got %d", ackCount)
        }
    })
}
```

#### 5.1.2 消息队列集成验证

```go
// message_queue_test.go
package integration

import (
    "context"
    "testing"
    "time"
)

// TestKafkaIntegration Kafka集成测试
func TestKafkaIntegration(t *testing.T) {
    ctx := context.Background()
    
    // 创建生产者
    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": "kafka:9092",
        "acks":              "all",
        "retries":           3,
    })
    if err != nil {
        t.Fatalf("Failed to create producer: %v", err)
    }
    defer producer.Close()
    
    // 创建消费者
    consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers": "kafka:9092",
        "group.id":          "test-group",
        "auto.offset.reset": "earliest",
    })
    if err != nil {
        t.Fatalf("Failed to create consumer: %v", err)
    }
    defer consumer.Close()
    
    topic := "test-collaboration-events"
    
    t.Run("Message_Produce_Consume", func(t *testing.T) {
        // 订阅主题
        consumer.Subscribe(topic, nil)
        
        // 发送消息
        messageCount := 100
        sentMessages := make(map[string]bool)
        
        for i := 0; i < messageCount; i++ {
            key := fmt.Sprintf("key-%d", i)
            value := fmt.Sprintf("value-%d", i)
            
            err := producer.Produce(&kafka.Message{
                TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
                Key:            []byte(key),
                Value:          []byte(value),
            }, nil)
            
            if err != nil {
                t.Fatalf("Failed to produce message: %v", err)
            }
            
            sentMessages[key] = true
        }
        
        producer.Flush(5000)
        
        // 消费消息
        receivedCount := 0
        timeout := time.After(30 * time.Second)
        
        for receivedCount < messageCount {
            select {
            case <-timeout:
                t.Fatalf("Timeout waiting for messages. Received %d/%d", receivedCount, messageCount)
            default:
                msg, err := consumer.ReadMessage(100 * time.Millisecond)
                if err != nil {
                    continue
                }
                
                key := string(msg.Key)
                if !sentMessages[key] {
                    t.Errorf("Received unexpected message: %s", key)
                }
                
                receivedCount++
                delete(sentMessages, key)
            }
        }
        
        t.Logf("Successfully produced and consumed %d messages", messageCount)
    })
    
    t.Run("Message_Ordering", func(t *testing.T) {
        // 发送有序消息
        for i := 0; i < 10; i++ {
            key := "ordered-key"  // 相同key确保分区一致
            value := fmt.Sprintf("%d", i)
            
            producer.Produce(&kafka.Message{
                TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
                Key:            []byte(key),
                Value:          []byte(value),
            }, nil)
        }
        
        producer.Flush(5000)
        
        // 验证顺序
        expectedValue := 0
        timeout := time.After(10 * time.Second)
        
        for expectedValue < 10 {
            select {
            case <-timeout:
                t.Fatalf("Timeout waiting for ordered messages")
            default:
                msg, err := consumer.ReadMessage(100 * time.Millisecond)
                if err != nil {
                    continue
                }
                
                if string(msg.Key) != "ordered-key" {
                    continue
                }
                
                value, _ := strconv.Atoi(string(msg.Value))
                if value != expectedValue {
                    t.Errorf("Order violation: expected %d, got %d", expectedValue, value)
                }
                expectedValue++
            }
        }
    })
}
```

### 5.2 数据一致性验证

```go
// consistency_test.go
package integration

import (
    "context"
    "sync"
    "testing"
    "time"
)

// TestEventualConsistency 最终一致性测试
func TestEventualConsistency(t *testing.T) {
    ctx := context.Background()
    
    // 创建多个客户端
    clientCount := 5
    clients := make([]*CollaborationClient, clientCount)
    
    for i := 0; i < clientCount; i++ {
        clients[i] = NewCollaborationClient(fmt.Sprintf("user-%d", i))
        if err := clients[i].Connect(ctx); err != nil {
            t.Fatalf("Failed to connect client %d: %v", i, err)
        }
        defer clients[i].Disconnect()
    }
    
    roomID := "consistency-test-room"
    
    // 所有客户端加入同一房间
    for _, client := range clients {
        if err := client.JoinRoom(ctx, roomID); err != nil {
            t.Fatalf("Failed to join room: %v", err)
        }
    }
    
    t.Run("Concurrent_Edits_Consistency", func(t *testing.T) {
        operationCount := 20
        var wg sync.WaitGroup
        
        // 每个客户端并发发送操作
        for i, client := range clients {
            wg.Add(1)
            go func(idx int, c *CollaborationClient) {
                defer wg.Done()
                
                for j := 0; j < operationCount; j++ {
                    op := &WallOperation{
                        WallID: fmt.Sprintf("wall-%d", idx),
                        Action: ActionUpdate,
                        Data: WallData{
                            Height: float64(j * 10),
                        },
                    }
                    
                    if err := c.SendOperation(ctx, op); err != nil {
                        t.Errorf("Client %d failed to send operation: %v", idx, err)
                    }
                    
                    time.Sleep(50 * time.Millisecond)
                }
            }(i, client)
        }
        
        wg.Wait()
        
        // 等待同步完成
        time.Sleep(2 * time.Second)
        
        // 验证所有客户端状态一致
        referenceState := clients[0].GetDocumentState()
        
        for i, client := range clients[1:] {
            state := client.GetDocumentState()
            
            if !statesEqual(referenceState, state) {
                t.Errorf("Client %d state differs from reference", i+1)
                t.Logf("Reference: %+v", referenceState)
                t.Logf("Actual: %+v", state)
            }
        }
        
        t.Logf("All %d clients have consistent state", clientCount)
    })
    
    t.Run("Offline_Recovery_Consistency", func(t *testing.T) {
        // 客户端1发送操作后断开
        client1 := clients[0]
        
        for i := 0; i < 5; i++ {
            op := &WallOperation{
                WallID: "offline-test-wall",
                Action: ActionUpdate,
                Data: WallData{Height: float64(i * 100)},
            }
            client1.SendOperation(ctx, op)
        }
        
        // 断开连接
        client1.Disconnect()
        
        // 其他客户端继续操作
        for _, client := range clients[1:] {
            for i := 0; i < 5; i++ {
                op := &WallOperation{
                    WallID: "offline-test-wall",
                    Action: ActionUpdate,
                    Data: WallData{Height: float64(500 + i * 10)},
                }
                client.SendOperation(ctx, op)
            }
        }
        
        time.Sleep(1 * time.Second)
        
        // 客户端1重新连接
        client1.Connect(ctx)
        client1.JoinRoom(ctx, roomID)
        
        // 等待状态同步
        time.Sleep(2 * time.Second)
        
        // 验证状态一致
        client1State := client1.GetDocumentState()
        referenceState := clients[1].GetDocumentState()
        
        if !statesEqual(client1State, referenceState) {
            t.Errorf("Client 1 state inconsistent after reconnection")
        }
    })
}

func statesEqual(a, b map[string]*WallState) bool {
    if len(a) != len(b) {
        return false
    }
    
    for id, stateA := range a {
        stateB, ok := b[id]
        if !ok {
            return false
        }
        
        if stateA.Height != stateB.Height {
            return false
        }
    }
    
    return true
}
```

### 5.3 故障恢复和容错验证

```go
// fault_tolerance_test.go
package integration

import (
    "context"
    "testing"
    "time"
)

// TestFaultTolerance 故障容错测试
func TestFaultTolerance(t *testing.T) {
    ctx := context.Background()
    
    t.Run("Service_Restart_Recovery", func(t *testing.T) {
        // 创建客户端
        client := NewCollaborationClient("test-user")
        if err := client.Connect(ctx); err != nil {
            t.Fatalf("Failed to connect: %v", err)
        }
        
        // 加入房间并发送操作
        client.JoinRoom(ctx, "fault-test-room")
        
        for i := 0; i < 5; i++ {
            client.SendOperation(ctx, &WallOperation{
                WallID: fmt.Sprintf("wall-%d", i),
                Action: ActionCreate,
            })
        }
        
        // 模拟服务重启
        t.Log("Simulating service restart...")
        restartService("collaboration-service")
        
        // 等待服务恢复
        time.Sleep(5 * time.Second)
        
        // 验证客户端自动重连
        if err := client.WaitForReconnect(ctx, 10*time.Second); err != nil {
            t.Fatalf("Client failed to reconnect: %v", err)
        }
        
        // 验证可以继续操作
        if err := client.SendOperation(ctx, &WallOperation{
            WallID: "post-restart-wall",
            Action: ActionCreate,
        }); err != nil {
            t.Fatalf("Failed to send operation after restart: %v", err)
        }
        
        t.Log("Service restart recovery successful")
    })
    
    t.Run("Network_Partition_Recovery", func(t *testing.T) {
        // 创建两个客户端
        client1 := NewCollaborationClient("user-1")
        client2 := NewCollaborationClient("user-2")
        
        client1.Connect(ctx)
        client2.Connect(ctx)
        
        client1.JoinRoom(ctx, "partition-test-room")
        client2.JoinRoom(ctx, "partition-test-room")
        
        // 模拟网络分区
        t.Log("Simulating network partition...")
        partitionNetwork("client-1")
        
        // 客户端1发送操作（无法广播）
        for i := 0; i < 3; i++ {
            client1.SendOperation(ctx, &WallOperation{
                WallID: "partitioned-wall",
                Action: ActionUpdate,
                Data:   WallData{Height: float64(i * 100)},
            })
        }
        
        // 客户端2也发送操作
        for i := 0; i < 3; i++ {
            client2.SendOperation(ctx, &WallOperation{
                WallID: "partitioned-wall",
                Action: ActionUpdate,
                Data:   WallData{Height: float64(500 + i * 10)},
            })
        }
        
        // 恢复网络
        t.Log("Restoring network...")
        restoreNetwork("client-1")
        time.Sleep(2 * time.Second)
        
        // 验证最终一致性
        state1 := client1.GetDocumentState()
        state2 := client2.GetDocumentState()
        
        if !statesEqual(state1, state2) {
            t.Logf("State 1: %+v", state1)
            t.Logf("State 2: %+v", state2)
            t.Error("States inconsistent after partition recovery")
        } else {
            t.Log("Network partition recovery successful")
        }
    })
    
    t.Run("Message_Loss_Recovery", func(t *testing.T) {
        client := NewCollaborationClient("test-user")
        client.Connect(ctx)
        client.JoinRoom(ctx, "message-loss-test-room")
        
        // 启用消息丢失模拟
        client.EnableMessageLossSimulation(0.2) // 20%丢失率
        
        operationCount := 50
        sentCount := 0
        
        for i := 0; i < operationCount; i++ {
            err := client.SendOperation(ctx, &WallOperation{
                WallID: fmt.Sprintf("loss-test-wall-%d", i),
                Action: ActionCreate,
            })
            
            if err == nil {
                sentCount++
            }
        }
        
        // 禁用消息丢失
        client.DisableMessageLossSimulation()
        
        // 等待重传完成
        time.Sleep(3 * time.Second)
        
        // 验证所有操作最终到达
        state := client.GetDocumentState()
        receivedCount := len(state)
        
        t.Logf("Sent: %d, Received: %d", sentCount, receivedCount)
        
        if receivedCount < sentCount {
            t.Errorf("Message loss not recovered: sent %d, received %d", sentCount, receivedCount)
        }
    })
}

// CircuitBreakerTest 熔断器测试
func TestCircuitBreaker(t *testing.T) {
    ctx := context.Background()
    
    // 配置熔断器
    cb := circuitbreaker.New(circuitbreaker.Config{
        FailureThreshold:    5,
        SuccessThreshold:    2,
        Timeout:             5 * time.Second,
        HalfOpenMaxRequests: 3,
    })
    
    t.Run("Circuit_Opens_On_Failures", func(t *testing.T) {
        // 模拟失败
        for i := 0; i < 5; i++ {
            err := cb.Execute(func() error {
                return fmt.Errorf("simulated error")
            })
            
            if i < 4 && cb.State() != circuitbreaker.StateClosed {
                t.Errorf("Circuit opened too early at iteration %d", i)
            }
        }
        
        // 验证熔断器打开
        if cb.State() != circuitbreaker.StateOpen {
            t.Errorf("Expected circuit to be open, got %v", cb.State())
        }
        
        // 验证后续请求被快速拒绝
        err := cb.Execute(func() error {
            return nil
        })
        
        if err != circuitbreaker.ErrCircuitOpen {
            t.Errorf("Expected ErrCircuitOpen, got %v", err)
        }
    })
    
    t.Run("Circuit_Closes_After_Recovery", func(t *testing.T) {
        // 等待超时进入半开状态
        time.Sleep(6 * time.Second)
        
        if cb.State() != circuitbreaker.StateHalfOpen {
            t.Errorf("Expected half-open state, got %v", cb.State())
        }
        
        // 发送成功请求
        for i := 0; i < 2; i++ {
            err := cb.Execute(func() error {
                return nil
            })
            
            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
        }
        
        // 验证熔断器关闭
        if cb.State() != circuitbreaker.StateClosed {
            t.Errorf("Expected circuit to be closed, got %v", cb.State())
        }
    })
}
```



---

## 6. 关键技术风险验证

### 6.1 CRDT算法实现复杂度评估

#### 6.1.1 复杂度分析

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CRDT实现复杂度评估                                │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────┬────────────┬────────────┬─────────────────────┐
│     组件            │  复杂度    │  风险等级  │     缓解措施        │
├─────────────────────┼────────────┼────────────┼─────────────────────┤
│ 向量时钟实现        │    中      │    低      │ 使用成熟库/参考实现 │
│ 操作序列化          │    低      │    低      │ Protobuf标准化      │
│ LWW寄存器           │    低      │    低      │ 简单可靠            │
│ OR-Set实现          │    中      │    中      │ 需要充分测试        │
│ 操作转换(OT)        │    高      │    高      │ 考虑使用现成库      │
│ 冲突解决策略        │    高      │    高      │ 领域专家参与设计    │
│ 历史压缩/垃圾回收   │    高      │    中      │ 渐进式实现          │
└─────────────────────┴────────────┴────────────┴─────────────────────┘
```

#### 6.1.2 风险评估矩阵

| 风险项 | 发生概率 | 影响程度 | 风险等级 | 缓解策略 |
|--------|----------|----------|----------|----------|
| CRDT算法bug导致数据不一致 | 中 | 高 | **高** | 1) 使用成熟CRDT库<br>2) 大量自动化测试<br>3) 数据校验机制 |
| 向量时钟溢出 | 低 | 高 | **中** | 1) 使用64位时间戳<br>2) 定期时钟同步<br>3) 溢出检测 |
| 操作历史无限增长 | 中 | 中 | **中** | 1) 历史压缩<br>2) 定期快照<br>3) 垃圾回收 |
| 复杂场景冲突无法解决 | 中 | 高 | **高** | 1) 领域专家设计<br>2) 用户干预机制<br>3) 回滚能力 |

#### 6.1.3 推荐CRDT库选型

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CRDT库选型对比                                    │
└─────────────────────────────────────────────────────────────────────┘

1. Yjs (JavaScript) - 前端首选
   ✅ 成熟的CRDT实现
   ✅ 丰富的文档和社区
   ✅ 支持多种数据类型
   ❌ 仅JavaScript，需要与后端桥接
   适用：前端协作编辑

2. Automerge (Rust/JS) - 跨平台
   ✅ Rust核心，性能优秀
   ✅ 支持多种语言绑定
   ✅ 活跃开发
   ⚠️ API仍在演进
   适用：跨平台协作

3. 自研实现 (Go) - 定制化
   ✅ 完全可控
   ✅ 针对建筑领域优化
   ❌ 开发成本高
   ❌ 需要大量测试
   适用：特殊需求场景

推荐方案：
├── 前端：Yjs (成熟稳定)
├── 后端：参考Automerge设计自研
└── 通信：自定义协议桥接
```

### 6.2 WebSocket连接管理挑战

#### 6.2.1 连接管理挑战分析

```
┌─────────────────────────────────────────────────────────────────────┐
│                 WebSocket连接管理挑战                                │
└─────────────────────────────────────────────────────────────────────┘

挑战1: 大规模连接管理
┌─────────────────────────────────────────────────────────────────┐
│  问题: 10000+并发连接，每个连接需要维护状态                      │
│  影响: 内存消耗、CPU负载、连接稳定性                             │
│                                                                 │
│  解决方案:                                                      │
│  ├── 连接池化：复用goroutine处理多个连接                         │
│  ├── 水平扩展：多实例分担连接负载                                │
│  ├── 心跳优化：自适应心跳间隔                                    │
│  └── 连接分级：活跃/空闲/休眠状态管理                            │
└─────────────────────────────────────────────────────────────────┘

挑战2: 跨实例消息广播
┌─────────────────────────────────────────────────────────────────┐
│  问题: 用户A连接实例1，用户B连接实例2，如何广播？                 │
│  影响: 消息丢失、延迟增加                                        │
│                                                                 │
│  解决方案:                                                      │
│  ├── Redis Pub/Sub：轻量级广播                                   │
│  ├── Kafka：可靠消息传递                                         │
│  ├── 一致性哈希：相同房间路由到相同实例                          │
│  └── 消息确认机制：确保送达                                      │
└─────────────────────────────────────────────────────────────────┘

挑战3: 连接异常处理
┌─────────────────────────────────────────────────────────────────┐
│  问题: 网络抖动、客户端崩溃、服务器重启                          │
│  影响: 数据丢失、状态不一致                                      │
│                                                                 │
│  解决方案:                                                      │
│  ├── 自动重连：指数退避策略                                      │
│  ├── 会话恢复：Redis存储会话状态                                 │
│  ├── 操作重放：客户端缓存未确认操作                              │
│  └── 优雅关闭：确保消息发送完成                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 6.2.2 连接管理性能指标

| 指标 | 目标值 | 测试方法 |
|------|--------|----------|
| 单实例最大连接数 | 10000 | 压力测试 |
| 连接建立时间 | <100ms | 性能测试 |
| 消息广播延迟(P99) | <50ms | 端到端测试 |
| 内存/连接 | <10KB | 内存分析 |
| 重连成功率 | >99.9% | 故障注入测试 |

### 6.3 大规模并发性能瓶颈识别

#### 6.3.1 潜在瓶颈分析

```
┌─────────────────────────────────────────────────────────────────────┐
│                    性能瓶颈分析图                                    │
└─────────────────────────────────────────────────────────────────────┘

    用户请求
       │
       ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  API网关    │────▶│  负载均衡器 │────▶│  协作服务   │
│  (Kong)     │     │  (Nginx)    │     │  (Go)       │
│             │     │             │     │             │
│ 瓶颈:配置   │     │ 瓶颈:连接数 │     │ 瓶颈:CPU/内存│
│ 缓解:调优   │     │ 缓解:调参   │     │ 缓解:扩展   │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                          ┌────────────────────┼────────────────────┐
                          │                    │                    │
                          ▼                    ▼                    ▼
                   ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
                   │   Redis     │      │   Kafka     │      │ PostgreSQL  │
                   │  (Session)  │      │  (Events)   │      │  (History)  │
                   │             │      │             │      │             │
                   │瓶颈:内存    │      │瓶颈:磁盘I/O │      │瓶颈:连接池  │
                   │缓解:集群    │      │缓解:分区    │      │缓解:读写分离│
                   └─────────────┘      └─────────────┘      └─────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                        瓶颈识别清单                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. CPU瓶颈                                                         │
│     症状: CPU使用率持续>80%                                         │
│     原因: CRDT计算、序列化/反序列化、消息处理                        │
│     缓解: 优化算法、使用协程池、水平扩展                             │
│                                                                     │
│  2. 内存瓶颈                                                        │
│     症状: 内存持续增长，频繁GC                                       │
│     原因: 操作历史缓存、连接状态、消息缓冲区                         │
│     缓解: 历史压缩、对象池、流式处理                                 │
│                                                                     │
│  3. 网络瓶颈                                                        │
│     症状: 网络带宽饱和，延迟增加                                     │
│     原因: 大数据传输、广播风暴、消息序列化                           │
│     缓解: 数据压缩、增量传输、CDN                                    │
│                                                                     │
│  4. 数据库瓶颈                                                      │
│     症状: 查询慢，连接池耗尽                                         │
│     原因: 大量写入、复杂查询、锁竞争                                 │
│     缓解: 读写分离、分库分表、缓存                                   │
│                                                                     │
│  5. 消息队列瓶颈                                                    │
│     症状: 消息堆积，消费延迟                                         │
│     原因: 生产者过快、消费者慢、分区不均                             │
│     缓解: 增加分区、优化消费者、背压机制                             │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 6.3.2 性能调优建议

| 层面 | 调优项 | 建议配置 |
|------|--------|----------|
| Go运行时 | GOMAXPROCS | 设置为CPU核心数 |
| Go运行时 | GC目标 | 100% (默认) |
| HTTP | Keep-Alive | 启用，超时30s |
| WebSocket | 读缓冲区 | 4KB |
| WebSocket | 写缓冲区 | 4KB |
| Redis | 连接池 | 100连接 |
| Redis | 超时 | 5s |
| Kafka | 批量大小 | 16KB |
| Kafka | linger.ms | 10ms |

---

## 7. POC执行计划

### 7.1 POC环境搭建方案

#### 7.1.1 基础设施架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                    POC环境基础设施                                   │
└─────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────┐
    │                    Kubernetes Cluster                        │
    │                     (Minikube/Kind)                          │
    ├─────────────────────────────────────────────────────────────┤
    │                                                             │
    │  Namespace: poc-architecture                                │
    │                                                             │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
    │  │ API Gateway │  │Collaboration│  │  Geometry   │         │
    │  │   (Kong)    │  │  Service    │  │  Service    │         │
    │  │  1 replica  │  │  3 replicas │  │  2 replicas │         │
    │  │  1CPU/1GB   │  │  2CPU/4GB   │  │  2CPU/4GB   │         │
    │  └─────────────┘  └─────────────┘  └─────────────┘         │
    │                                                             │
    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
    │  │   Script    │  │   Redis     │  │   Kafka     │         │
    │  │  Service    │  │  Cluster    │  │  Cluster    │         │
    │  │  2 replicas │  │  3 replicas │  │  3 brokers  │         │
    │  │  2CPU/4GB   │  │  1CPU/2GB   │  │  2CPU/4GB   │         │
    │  └─────────────┘  └─────────────┘  └─────────────┘         │
    │                                                             │
    │  ┌─────────────┐  ┌─────────────┐                          │
    │  │ PostgreSQL  │  │  Prometheus │                          │
    │  │  Primary    │  │  + Grafana  │                          │
    │  │  1CPU/2GB   │  │  1CPU/2GB   │                          │
    │  └─────────────┘  └─────────────┘                          │
    │                                                             │
    └─────────────────────────────────────────────────────────────┘
```

#### 7.1.2 Docker Compose开发环境

```yaml
# docker-compose.poc.yml
version: '3.8'

services:
  # API网关
  kong:
    image: kong:3.4
    environment:
      KONG_DATABASE: "off"
      KONG_DECLARATIVE_CONFIG: /kong/declarative/kong.yml
      KONG_PROXY_ACCESS_LOG: /dev/stdout
      KONG_ADMIN_ACCESS_LOG: /dev/stdout
      KONG_PROXY_ERROR_LOG: /dev/stderr
      KONG_ADMIN_ERROR_LOG: /dev/stderr
      KONG_PLUGINS: bundled
    ports:
      - "8000:8000"
      - "8443:8443"
      - "8001:8001"
      - "8444:8444"
    volumes:
      - ./kong.yml:/kong/declarative/kong.yml:ro
    networks:
      - poc-network

  # 协作服务
  collaboration-service:
    build:
      context: ./services/collaboration
      dockerfile: Dockerfile
    environment:
      - REDIS_URL=redis:6379
      - KAFKA_BROKERS=kafka:9092
      - SERVICE_PORT=8080
      - WS_PORT=8081
    ports:
      - "8080:8080"
      - "8081:8081"
    depends_on:
      - redis
      - kafka
    deploy:
      replicas: 3
    networks:
      - poc-network

  # 几何服务
  geometry-service:
    build:
      context: ./services/geometry
      dockerfile: Dockerfile
    environment:
      - POSTGRES_URL=postgresql://postgres:password@postgres:5432/geometry
      - SERVICE_PORT=8080
    ports:
      - "8082:8080"
    depends_on:
      - postgres
    deploy:
      replicas: 2
    networks:
      - poc-network

  # 脚本服务
  script-service:
    build:
      context: ./services/script
      dockerfile: Dockerfile
    environment:
      - REDIS_URL=redis:6379
      - KAFKA_BROKERS=kafka:9092
      - SERVICE_PORT=8080
    ports:
      - "8083:8080"
    depends_on:
      - redis
      - kafka
    deploy:
      replicas: 2
    networks:
      - poc-network

  # Redis
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    networks:
      - poc-network

  # Kafka
  kafka:
    image: confluentinc/cp-kafka:7.5.0
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9092:9092"
    depends_on:
      - zookeeper
    networks:
      - poc-network

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
    networks:
      - poc-network

  # PostgreSQL
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: architecture
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - poc-network

  # Prometheus
  prometheus:
    image: prom/prometheus:v2.47.0
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - poc-network

  # Grafana
  grafana:
    image: grafana/grafana:10.1.0
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./grafana/datasources:/etc/grafana/provisioning/datasources:ro
    networks:
      - poc-network

  # k6负载测试
  k6:
    image: grafana/k6:latest
    volumes:
      - ./tests:/tests
    command: run /tests/performance_test.js
    networks:
      - poc-network
    profiles:
      - test

volumes:
  redis-data:
  postgres-data:
  prometheus-data:
  grafana-data:

networks:
  poc-network:
    driver: bridge
```

### 7.2 测试用例设计

#### 7.2.1 功能测试用例

| 用例ID | 用例名称 | 前置条件 | 测试步骤 | 预期结果 | 优先级 |
|--------|----------|----------|----------|----------|--------|
| TC001 | 单用户墙体创建 | 服务正常运行 | 1. 连接WebSocket<br>2. 发送墙体创建操作 | 墙体创建成功，返回确认 | P0 |
| TC002 | 多用户并发编辑 | 2+用户在线 | 1. 用户A编辑墙体<br>2. 用户B同时编辑 | 无冲突，状态一致 | P0 |
| TC003 | 离线恢复 | 用户离线后重连 | 1. 用户A离线<br>2. 其他用户编辑<br>3. 用户A重连 | 状态同步，无数据丢失 | P0 |
| TC004 | 几何数据传输 | 大模型数据 | 1. 上传100MB模型<br>2. 多用户下载 | 传输完整，无损坏 | P1 |
| TC005 | 脚本执行 | 脚本服务正常 | 1. 提交脚本任务<br>2. 等待执行完成 | 任务完成，结果正确 | P1 |

#### 7.2.2 性能测试用例

| 用例ID | 用例名称 | 负载配置 | 预期指标 | 通过标准 |
|--------|----------|----------|----------|----------|
| PT001 | 本地反馈延迟 | 1用户，100操作/秒 | P95 < 16ms | 通过 |
| PT002 | 广播延迟 | 50用户，同时操作 | P95 < 100ms | 通过 |
| PT003 | 并发用户 | 100用户，10分钟 | 无连接丢失 | 通过 |
| PT004 | 峰值负载 | 200用户，5分钟 | 错误率<1% | 通过 |
| PT005 | 几何传输 | 100MB模型 | 传输时间<10s | 通过 |

### 7.3 验收标准和通过条件

#### 7.3.1 功能验收标准

```
┌─────────────────────────────────────────────────────────────────────┐
│                    功能验收标准                                      │
└─────────────────────────────────────────────────────────────────────┘

P0 (必须满足)
├── 协作服务
│   ├── ✅ 单用户墙体CRUD操作正常
│   ├── ✅ 多用户并发编辑无冲突
│   ├── ✅ 离线后状态正确恢复
│   └── ✅ WebSocket连接稳定(>99.9%)
│
├── 几何服务
│   ├── ✅ 几何数据序列化/反序列化正确
│   ├── ✅ 大模型(100MB)传输完整
│   └── ✅ 网格生成正确
│
└── 脚本服务
    ├── ✅ 脚本任务提交成功
    ├── ✅ 任务执行完成
    └── ✅ 结果正确返回

P1 (应该满足)
├── 协作服务
│   ├── ✅ 100+用户并发支持
│   └── ✅ 操作历史可追溯
│
├── 几何服务
│   ├── ✅ LOD生成正确
│   └── ✅ 增量更新有效
│
└── 脚本服务
    ├── ✅ 任务优先级生效
    └── ✅ 资源限制有效
```

#### 7.3.2 性能验收标准

| 指标 | 目标值 | 可接受值 | 验收结果 |
|------|--------|----------|----------|
| 本地反馈延迟(P95) | <16ms | <50ms | 待测试 |
| 广播延迟(P95) | <100ms | <200ms | 待测试 |
| 并发用户数 | 100+ | 50+ | 待测试 |
| 操作成功率 | >99.9% | >99% | 待测试 |
| 连接恢复成功率 | >99.9% | >99% | 待测试 |
| 几何传输完整性 | 100% | 100% | 待测试 |

### 7.4 时间和资源估算

#### 7.4.1 POC时间计划

```
┌─────────────────────────────────────────────────────────────────────┐
│                    POC时间计划 (4周)                                 │
└─────────────────────────────────────────────────────────────────────┘

第1周: 环境搭建 + 基础组件
┌─────────────────────────────────────────────────────────────────────┐
│ 周一    │ 周二    │ 周三    │ 周四    │ 周五    │                   │
├─────────┼─────────┼─────────┼─────────┼─────────┤                   │
│ 环境搭建 │ 环境搭建 │ 协作服务 │ 协作服务 │ 代码审查 │                   │
│ Docker  │ K8s配置 │ 基础框架 │ CRDT核心 │         │                   │
└─────────┴─────────┴─────────┴─────────┴─────────┘                   │

第2周: 核心服务开发
┌─────────────────────────────────────────────────────────────────────┐
│ 周一    │ 周二    │ 周三    │ 周四    │ 周五    │                   │
├─────────┼─────────┼─────────┼─────────┼─────────┤                   │
│ WebSocket│ 几何服务 │ 几何服务 │ 脚本服务 │ 代码审查 │                   │
│ 连接管理 │ 数据模型 │ API实现 │ 调度框架 │         │                   │
└─────────┴─────────┴─────────┴─────────┴─────────┘                   │

第3周: 集成 + 功能测试
┌─────────────────────────────────────────────────────────────────────┐
│ 周一    │ 周二    │ 周三    │ 周四    │ 周五    │                   │
├─────────┼─────────┼─────────┼─────────┼─────────┤                   │
│ 服务集成 │ 消息队列 │ 功能测试 │ 功能测试 │ Bug修复  │                   │
│ gRPC    │ 集成    │ 用例执行 │ 用例执行 │         │                   │
└─────────┴─────────┴─────────┴─────────┴─────────┘                   │

第4周: 性能测试 + 报告
┌─────────────────────────────────────────────────────────────────────┐
│ 周一    │ 周二    │ 周三    │ 周四    │ 周五    │                   │
├─────────┼─────────┼─────────┼─────────┼─────────┤                   │
│ 性能测试 │ 性能测试 │ 性能优化 │ 报告编写 │ 评审汇报 │                   │
│ 基准测试 │ 压力测试 │         │         │         │                   │
└─────────┴─────────┴─────────┴─────────┴─────────┘                   │
```

#### 7.4.2 资源需求

| 资源类型 | 需求 | 说明 |
|----------|------|------|
| 后端开发工程师 | 2人 | Go/Java开发 |
| DevOps工程师 | 1人 | 环境搭建、CI/CD |
| 测试工程师 | 1人 | 测试用例设计、执行 |
| 云服务器 | 5台 | 4核8G，用于POC环境 |
| 开发机 | 4台 | 开发使用 |

#### 7.4.3 成本估算

| 项目 | 数量 | 单价(月) | 总计(月) |
|------|------|----------|----------|
| 云服务器 | 5台 | ¥500 | ¥2,500 |
| 开发人员 | 4人 | ¥25,000 | ¥100,000 |
| 第三方服务 | - | - | ¥5,000 |
| **总计** | - | - | **¥107,500** |

*注：POC周期为1个月，总成本约¥107,500*

---

## 8. 结论与建议

### 8.1 可行性结论

基于以上分析，半自动化建筑设计平台后端架构的技术方案具有以下可行性：

| 技术方案 | 可行性 | 风险等级 | 建议 |
|----------|--------|----------|------|
| 微服务架构 | ✅ 可行 | 低 | 采用，服务边界清晰 |
| CRDT协作引擎 | ✅ 可行 | 中 | 采用，建议使用成熟库 |
| WebSocket通信 | ✅ 可行 | 中 | 采用，注意连接管理 |
| 事件驱动架构 | ✅ 可行 | 低 | 采用，Kafka成熟稳定 |
| gRPC服务通信 | ✅ 可行 | 低 | 采用，性能优秀 |

### 8.2 关键建议

1. **CRDT实现建议**
   - 前端采用Yjs成熟方案
   - 后端参考Automerge设计自研
   - 投入充足时间进行测试验证

2. **性能优化建议**
   - 优先保证本地反馈<16ms
   - 广播延迟目标<100ms
   - 预留水平扩展能力

3. **风险缓解建议**
   - 建立完善的监控体系
   - 设计降级方案
   - 准备回滚机制

### 8.3 下一步行动

1. ✅ 批准POC执行计划
2. 🔄 搭建POC环境
3. ⏳ 开发核心组件
4. ⏳ 执行测试验证
5. ⏳ 输出验证报告

---

**文档结束**

*本报告由后端架构团队编写，用于技术可行性评审。*

