
# 半自动化建筑设计平台 - 详细设计阶段报告
## 服务详细设计文档

---

## 文档信息

| 项目 | 内容 |
|------|------|
| 项目名称 | 半自动化建筑设计平台 |
| 文档版本 | v1.0.0 |
| 设计阶段 | 详细设计阶段 |
| 编写日期 | 2024年 |

---

## 目录

1. [协作服务详细设计](#1-协作服务详细设计)
2. [几何服务详细设计](#2-几何服务详细设计)
3. [属性服务详细设计](#3-属性服务详细设计)
4. [脚本服务详细设计](#4-脚本服务详细设计)
5. [版本服务详细设计](#5-版本服务详细设计)
6. [用户服务详细设计](#6-用户服务详细设计)
7. [服务间集成设计](#7-服务间集成设计)

---

# 1. 协作服务详细设计

## 1.1 服务概述

协作服务负责处理多用户实时协作功能，基于Yjs CRDT算法实现无冲突复制数据类型，支持多用户同时编辑设计文档。

### 核心功能
- 文档协作会话管理
- 用户光标和选区同步
- 操作历史记录
- 冲突自动解决
- 离线编辑支持

## 1.2 gRPC接口定义

### 1.2.1 Proto文件定义

```protobuf
syntax = "proto3";

package collaboration.v1;

option go_package = "github.com/archplatform/collaboration-service/api/v1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

// 协作服务
service CollaborationService {
  // 文档会话管理
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  rpc JoinSession(JoinSessionRequest) returns (stream CollaborationEvent);
  rpc LeaveSession(LeaveSessionRequest) returns (google.protobuf.Empty);
  rpc GetSessionInfo(GetSessionInfoRequest) returns (SessionInfo);
  rpc ListActiveSessions(ListActiveSessionsRequest) returns (ListActiveSessionsResponse);

  // 操作同步
  rpc SyncOperation(stream OperationBatch) returns (stream OperationAck);
  rpc GetMissingOperations(GetMissingOperationsRequest) returns (OperationBatch);

  // 光标和选区
  rpc UpdateCursor(UpdateCursorRequest) returns (google.protobuf.Empty);
  rpc UpdateSelection(UpdateSelectionRequest) returns (google.protobuf.Empty);

  // 历史记录
  rpc GetOperationHistory(GetOperationHistoryRequest) returns (OperationHistory);
  rpc UndoOperation(UndoOperationRequest) returns (UndoOperationResponse);
  rpc RedoOperation(RedoOperationRequest) returns (RedoOperationResponse);

  // 权限管理
  rpc GrantPermission(GrantPermissionRequest) returns (google.protobuf.Empty);
  rpc RevokePermission(RevokePermissionRequest) returns (google.protobuf.Empty);
  rpc CheckPermission(CheckPermissionRequest) returns (PermissionCheckResult);
}

// ==================== 请求/响应消息定义 ====================

message CreateSessionRequest {
  string document_id = 1;
  string user_id = 2;
  string tenant_id = 3;
  SessionType session_type = 4;
  map<string, string> metadata = 5;
}

message CreateSessionResponse {
  string session_id = 1;
  string websocket_url = 2;
  string token = 3;
  google.protobuf.Timestamp expires_at = 4;
}

message JoinSessionRequest {
  string session_id = 1;
  string user_id = 2;
  string user_name = 3;
  string user_avatar = 4;
  ClientInfo client_info = 5;
}

message LeaveSessionRequest {
  string session_id = 1;
  string user_id = 2;
}

message GetSessionInfoRequest {
  string session_id = 1;
}

message ListActiveSessionsRequest {
  string document_id = 1;
  string tenant_id = 2;
  int32 page_size = 3;
  string page_token = 4;
}

message ListActiveSessionsResponse {
  repeated SessionInfo sessions = 1;
  string next_page_token = 2;
}

// ==================== 操作同步消息 ====================

message OperationBatch {
  string session_id = 1;
  string user_id = 2;
  int64 client_clock = 3;
  int64 server_clock = 4;
  repeated Operation operations = 5;
  bytes yjs_update = 6;  // Yjs二进制更新数据
}

message Operation {
  string operation_id = 1;
  OperationType type = 2;
  string target_id = 3;
  bytes data = 4;
  google.protobuf.Timestamp timestamp = 5;
  map<string, string> metadata = 6;
}

message OperationAck {
  string operation_id = 1;
  AckStatus status = 2;
  int64 server_clock = 3;
  string error_message = 4;
}

message GetMissingOperationsRequest {
  string session_id = 1;
  int64 from_clock = 2;
  int64 to_clock = 3;
}

// ==================== 光标和选区消息 ====================

message UpdateCursorRequest {
  string session_id = 1;
  string user_id = 2;
  CursorPosition position = 3;
}

message UpdateSelectionRequest {
  string session_id = 1;
  string user_id = 2;
  SelectionRange selection = 3;
}

// ==================== 历史记录消息 ====================

message GetOperationHistoryRequest {
  string session_id = 1;
  int32 limit = 2;
  string cursor = 3;
}

message OperationHistory {
  repeated HistoricalOperation operations = 1;
  string next_cursor = 2;
  bool has_more = 3;
}

message UndoOperationRequest {
  string session_id = 1;
  string user_id = 2;
  int32 steps = 3;
}

message UndoOperationResponse {
  bool success = 1;
  repeated string undone_operation_ids = 2;
}

message RedoOperationRequest {
  string session_id = 1;
  string user_id = 2;
  int32 steps = 3;
}

message RedoOperationResponse {
  bool success = 1;
  repeated string redone_operation_ids = 2;
}

// ==================== 权限消息 ====================

message GrantPermissionRequest {
  string session_id = 1;
  string user_id = 2;
  PermissionLevel level = 3;
  string granted_by = 4;
}

message RevokePermissionRequest {
  string session_id = 1;
  string user_id = 2;
  string revoked_by = 3;
}

message CheckPermissionRequest {
  string session_id = 1;
  string user_id = 2;
  PermissionAction action = 3;
}

message PermissionCheckResult {
  bool allowed = 1;
  PermissionLevel current_level = 2;
}

// ==================== 事件消息 ====================

message CollaborationEvent {
  oneof event {
    UserJoinedEvent user_joined = 1;
    UserLeftEvent user_left = 2;
    OperationReceivedEvent operation_received = 3;
    CursorUpdatedEvent cursor_updated = 4;
    SelectionUpdatedEvent selection_updated = 5;
    PermissionChangedEvent permission_changed = 6;
    SessionClosedEvent session_closed = 7;
    AwarenessUpdateEvent awareness_update = 8;
  }
}

message UserJoinedEvent {
  UserInfo user = 1;
  google.protobuf.Timestamp joined_at = 2;
}

message UserLeftEvent {
  string user_id = 1;
  google.protobuf.Timestamp left_at = 2;
}

message OperationReceivedEvent {
  Operation operation = 1;
  string from_user_id = 2;
}

message CursorUpdatedEvent {
  string user_id = 1;
  CursorPosition position = 2;
}

message SelectionUpdatedEvent {
  string user_id = 1;
  SelectionRange selection = 2;
}

message PermissionChangedEvent {
  string user_id = 1;
  PermissionLevel old_level = 2;
  PermissionLevel new_level = 3;
  string changed_by = 4;
}

message SessionClosedEvent {
  string reason = 1;
  bool can_reconnect = 2;
}

message AwarenessUpdateEvent {
  map<string, bytes> states = 1;
}

// ==================== 通用数据结构 ====================

message SessionInfo {
  string session_id = 1;
  string document_id = 2;
  SessionType session_type = 3;
  SessionStatus status = 4;
  repeated UserInfo active_users = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp expires_at = 7;
  string created_by = 8;
}

message UserInfo {
  string user_id = 1;
  string user_name = 2;
  string user_avatar = 3;
  PermissionLevel permission_level = 4;
  CursorPosition cursor = 5;
  SelectionRange selection = 6;
  google.protobuf.Timestamp joined_at = 7;
  ClientInfo client_info = 8;
}

message CursorPosition {
  string element_id = 1;
  float x = 2;
  float y = 3;
  float z = 4;
  google.protobuf.Timestamp updated_at = 5;
}

message SelectionRange {
  repeated string element_ids = 1;
  SelectionType type = 2;
  google.protobuf.Timestamp updated_at = 3;
}

message ClientInfo {
  string client_type = 1;  // web, desktop, mobile
  string version = 2;
  string platform = 3;
}

message HistoricalOperation {
  Operation operation = 1;
  string user_id = 2;
  string user_name = 3;
  google.protobuf.Timestamp timestamp = 4;
  bool is_undone = 5;
}

// ==================== 枚举定义 ====================

enum SessionType {
  SESSION_TYPE_UNSPECIFIED = 0;
  SESSION_TYPE_DESIGN = 1;      // 设计协作
  SESSION_TYPE_REVIEW = 2;      // 评审协作
  SESSION_TYPE_PRESENTATION = 3; // 演示模式
}

enum SessionStatus {
  SESSION_STATUS_UNSPECIFIED = 0;
  SESSION_STATUS_ACTIVE = 1;
  SESSION_STATUS_PAUSED = 2;
  SESSION_STATUS_CLOSING = 3;
  SESSION_STATUS_CLOSED = 4;
}

enum OperationType {
  OPERATION_TYPE_UNSPECIFIED = 0;
  OPERATION_TYPE_INSERT = 1;
  OPERATION_TYPE_UPDATE = 2;
  OPERATION_TYPE_DELETE = 3;
  OPERATION_TYPE_TRANSFORM = 4;
  OPERATION_TYPE_PROPERTY_CHANGE = 5;
  OPERATION_TYPE_GEOMETRY_CHANGE = 6;
}

enum AckStatus {
  ACK_STATUS_UNSPECIFIED = 0;
  ACK_STATUS_SUCCESS = 1;
  ACK_STATUS_REJECTED = 2;
  ACK_STATUS_CONFLICT = 3;
  ACK_STATUS_TIMEOUT = 4;
}

enum PermissionLevel {
  PERMISSION_LEVEL_UNSPECIFIED = 0;
  PERMISSION_LEVEL_VIEWER = 1;   // 仅查看
  PERMISSION_LEVEL_COMMENTER = 2; // 可评论
  PERMISSION_LEVEL_EDITOR = 3;   // 可编辑
  PERMISSION_LEVEL_ADMIN = 4;    // 管理员
  PERMISSION_LEVEL_OWNER = 5;    // 所有者
}

enum PermissionAction {
  PERMISSION_ACTION_UNSPECIFIED = 0;
  PERMISSION_ACTION_VIEW = 1;
  PERMISSION_ACTION_COMMENT = 2;
  PERMISSION_ACTION_EDIT = 3;
  PERMISSION_ACTION_MANAGE = 4;
  PERMISSION_ACTION_DELETE = 5;
}

enum SelectionType {
  SELECTION_TYPE_UNSPECIFIED = 0;
  SELECTION_TYPE_SINGLE = 1;
  SELECTION_TYPE_MULTIPLE = 2;
  SELECTION_TYPE_BOX = 3;
  SELECTION_TYPE_LASSO = 4;
}
```

## 1.3 数据库表结构设计

### 1.3.1 PostgreSQL Schema

```sql
-- 启用UUID扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==================== 会话表 ====================
CREATE TABLE collaboration_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    session_type VARCHAR(32) NOT NULL DEFAULT 'design',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    yjs_state BYTEA,  -- Yjs文档状态快照
    server_clock BIGINT DEFAULT 0,

    CONSTRAINT chk_session_type CHECK (session_type IN ('design', 'review', 'presentation')),
    CONSTRAINT chk_session_status CHECK (status IN ('active', 'paused', 'closing', 'closed'))
);

CREATE INDEX idx_sessions_document ON collaboration_sessions(document_id);
CREATE INDEX idx_sessions_tenant ON collaboration_sessions(tenant_id);
CREATE INDEX idx_sessions_status ON collaboration_sessions(status);
CREATE INDEX idx_sessions_expires ON collaboration_sessions(expires_at);

-- ==================== 会话参与者表 ====================
CREATE TABLE session_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    user_name VARCHAR(255),
    user_avatar VARCHAR(500),
    permission_level VARCHAR(32) NOT NULL DEFAULT 'viewer',
    client_type VARCHAR(32),
    client_version VARCHAR(32),
    client_platform VARCHAR(32),
    cursor_position JSONB,
    selection_range JSONB,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,

    CONSTRAINT chk_permission_level CHECK (permission_level IN ('viewer', 'commenter', 'editor', 'admin', 'owner')),
    UNIQUE(session_id, user_id)
);

CREATE INDEX idx_participants_session ON session_participants(session_id);
CREATE INDEX idx_participants_user ON session_participants(user_id);
CREATE INDEX idx_participants_active ON session_participants(session_id, is_active);

-- ==================== 操作日志表 (按时间分区) ====================
CREATE TABLE operation_logs (
    id BIGSERIAL,
    session_id UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    operation_id UUID NOT NULL DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    client_clock BIGINT NOT NULL,
    server_clock BIGINT NOT NULL,
    operation_type VARCHAR(32) NOT NULL,
    target_id UUID,
    operation_data JSONB NOT NULL,
    yjs_update BYTEA,
    metadata JSONB DEFAULT '{}',
    is_undone BOOLEAN DEFAULT FALSE,
    undone_at TIMESTAMP WITH TIME ZONE,
    undone_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- 创建月度分区
CREATE TABLE operation_logs_2024_01 PARTITION OF operation_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE operation_logs_2024_02 PARTITION OF operation_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- ... 更多分区

CREATE INDEX idx_operations_session ON operation_logs(session_id);
CREATE INDEX idx_operations_server_clock ON operation_logs(session_id, server_clock);
CREATE INDEX idx_operations_user ON operation_logs(user_id);
CREATE INDEX idx_operations_target ON operation_logs(target_id);
CREATE INDEX idx_operations_undone ON operation_logs(session_id, is_undone);

-- ==================== 权限表 ====================
CREATE TABLE session_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES collaboration_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    permission_level VARCHAR(32) NOT NULL,
    granted_by UUID NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by UUID,
    is_active BOOLEAN DEFAULT TRUE,

    UNIQUE(session_id, user_id, is_active)
);

CREATE INDEX idx_permissions_session ON session_permissions(session_id);
CREATE INDEX idx_permissions_user ON session_permissions(user_id);

-- ==================== 离线操作队列 ====================
CREATE TABLE offline_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL,
    user_id UUID NOT NULL,
    client_clock BIGINT NOT NULL,
    operation_data JSONB NOT NULL,
    yjs_update BYTEA,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,

    UNIQUE(session_id, user_id, client_clock)
);

CREATE INDEX idx_offline_user ON offline_operations(user_id, synced_at);
CREATE INDEX idx_offline_session ON offline_operations(session_id);

-- ==================== 触发器函数 ====================
-- 自动更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_collaboration_sessions_updated_at
    BEFORE UPDATE ON collaboration_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 自动更新参与者活动时间
CREATE OR REPLACE FUNCTION update_participant_activity()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_activity_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_session_participants_activity
    BEFORE UPDATE ON session_participants
    FOR EACH ROW
    EXECUTE FUNCTION update_participant_activity();
```

### 1.3.2 Redis数据结构

```
# 活跃会话集合
SET active:sessions:{tenant_id} -> [session_id1, session_id2, ...]

# 会话参与者 (Hash)
HSET session:{session_id}:participants {user_id} -> {user_info_json}

# 用户光标位置 (Hash)
HSET session:{session_id}:cursors {user_id} -> {x,y,z,timestamp}

# 用户选区 (Hash)
HSET session:{session_id}:selections {user_id} -> {element_ids,type}

# 服务器时钟计数器
INCR session:{session_id}:clock

# 操作缓存 (Sorted Set, 按server_clock排序)
ZADD session:{session_id}:operations {server_clock} {operation_json}

# 在线状态 (TTL 5分钟)
SETEX user:{user_id}:online 300 {session_id}

# WebSocket连接映射
HSET ws:connections {connection_id} -> {session_id,user_id}

# Yjs文档状态 (Binary, 定期持久化到DB)
SET session:{session_id}:yjs:state {yjs_binary_data}

# 速率限制 (Sliding Window)
ZADD ratelimit:{user_id}:{action} {timestamp} {request_id}
```

## 1.4 核心业务逻辑实现

### 1.4.1 Go服务实现

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/gorilla/websocket"
    "google.golang.org/grpc"
    "google.golang.org/protobuf/types/known/emptypb"
    "google.golang.org/protobuf/types/known/timestamppb"
    "gorm.io/gorm"
)

// ==================== 领域模型 ====================

type Session struct {
    ID          string
    DocumentID  string
    TenantID    string
    SessionType SessionType
    Status      SessionStatus
    CreatedBy   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ExpiresAt   *time.Time
    Metadata    map[string]string
    YjsState    []byte
    ServerClock int64

    Participants map[string]*Participant
    mu           sync.RWMutex
}

type Participant struct {
    ID           string
    UserID       string
    UserName     string
    UserAvatar   string
    Permission   PermissionLevel
    ClientType   string
    Cursor       *CursorPosition
    Selection    *SelectionRange
    JoinedAt     time.Time
    LastActivity time.Time
    IsActive     bool
    Conn         *websocket.Conn
    mu           sync.Mutex
}

type CursorPosition struct {
    ElementID string
    X, Y, Z   float32
    UpdatedAt time.Time
}

type SelectionRange struct {
    ElementIDs []string
    Type       SelectionType
    UpdatedAt  time.Time
}

// ==================== 服务实现 ====================

type CollaborationServer struct {
    pb.UnimplementedCollaborationServiceServer

    db          *gorm.DB
    redis       *redis.Client
    sessions    map[string]*Session
    sessionMu   sync.RWMutex

    // Yjs文档管理器
    yjsManager  *YjsDocumentManager

    // 事件发布器
    eventBus    EventBus

    // 配置
    config      *Config
}

// NewCollaborationServer 创建协作服务实例
func NewCollaborationServer(db *gorm.DB, redis *redis.Client, config *Config) *CollaborationServer {
    server := &CollaborationServer{
        db:         db,
        redis:      redis,
        sessions:   make(map[string]*Session),
        yjsManager: NewYjsDocumentManager(),
        eventBus:   NewNATSEventBus(config.NATSURL),
        config:     config,
    }

    // 启动清理任务
    go server.cleanupTask()

    return server
}

// CreateSession 创建协作会话
func (s *CollaborationServer) CreateSession(
    ctx context.Context, 
    req *pb.CreateSessionRequest,
) (*pb.CreateSessionResponse, error) {
    // 验证用户权限
    if err := s.validateDocumentAccess(ctx, req.DocumentId, req.UserId, PermissionAction_EDIT); err != nil {
        return nil, status.Errorf(codes.PermissionDenied, "无权创建会话: %v", err)
    }

    // 检查是否已存在活跃会话
    existingSession, err := s.findActiveSession(ctx, req.DocumentId)
    if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, status.Errorf(codes.Internal, "查询会话失败: %v", err)
    }

    if existingSession != nil {
        // 返回现有会话
        return &pb.CreateSessionResponse{
            SessionId:    existingSession.ID,
            WebsocketUrl: s.generateWebSocketURL(existingSession.ID),
            Token:        s.generateSessionToken(existingSession.ID, req.UserId),
            ExpiresAt:    timestamppb.New(*existingSession.ExpiresAt),
        }, nil
    }

    // 创建新会话
    session := &Session{
        ID:          uuid.New().String(),
        DocumentID:  req.DocumentId,
        TenantID:    req.TenantId,
        SessionType: req.SessionType,
        Status:      SessionStatus_ACTIVE,
        CreatedBy:   req.UserId,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
        ExpiresAt:   ptr(time.Now().Add(s.config.SessionTTL)),
        Metadata:    req.Metadata,
        ServerClock: 0,
        Participants: make(map[string]*Participant),
    }

    // 持久化到数据库
    dbSession := &models.CollaborationSession{
        ID:          session.ID,
        DocumentID:  session.DocumentID,
        TenantID:    session.TenantID,
        SessionType: session.SessionType.String(),
        Status:      session.Status.String(),
        CreatedBy:   session.CreatedBy,
        ExpiresAt:   session.ExpiresAt,
        Metadata:    session.Metadata,
    }

    if err := s.db.WithContext(ctx).Create(dbSession).Error; err != nil {
        return nil, status.Errorf(codes.Internal, "创建会话失败: %v", err)
    }

    // 缓存到内存
    s.sessionMu.Lock()
    s.sessions[session.ID] = session
    s.sessionMu.Unlock()

    // 发布会话创建事件
    s.eventBus.Publish(ctx, "collaboration.session.created", &SessionCreatedEvent{
        SessionID:  session.ID,
        DocumentID: session.DocumentID,
        UserID:     req.UserId,
    })

    return &pb.CreateSessionResponse{
        SessionId:    session.ID,
        WebsocketUrl: s.generateWebSocketURL(session.ID),
        Token:        s.generateSessionToken(session.ID, req.UserId),
        ExpiresAt:    timestamppb.New(*session.ExpiresAt),
    }, nil
}

// JoinSession 加入会话 (流式)
func (s *CollaborationServer) JoinSession(
    req *pb.JoinSessionRequest, 
    stream pb.CollaborationService_JoinSessionServer,
) error {
    ctx := stream.Context()

    // 验证会话存在
    session, err := s.getSession(ctx, req.SessionId)
    if err != nil {
        return status.Errorf(codes.NotFound, "会话不存在: %v", err)
    }

    // 检查会话状态
    if session.Status != SessionStatus_ACTIVE {
        return status.Errorf(codes.FailedPrecondition, "会话不在活跃状态")
    }

    // 检查用户权限
    if err := s.checkSessionPermission(ctx, req.SessionId, req.UserId, PermissionAction_VIEW); err != nil {
        return status.Errorf(codes.PermissionDenied, "无权加入会话: %v", err)
    }

    // 创建或更新参与者
    participant := &Participant{
        ID:           uuid.New().String(),
        UserID:       req.UserId,
        UserName:     req.UserName,
        UserAvatar:   req.UserAvatar,
        Permission:   s.getUserPermission(ctx, req.SessionId, req.UserId),
        ClientType:   req.ClientInfo.ClientType,
        JoinedAt:     time.Now(),
        LastActivity: time.Now(),
        IsActive:     true,
    }

    session.mu.Lock()
    session.Participants[req.UserId] = participant
    session.mu.Unlock()

    // 持久化参与者信息
    dbParticipant := &models.SessionParticipant{
        SessionID:      session.ID,
        UserID:         participant.UserID,
        UserName:       participant.UserName,
        UserAvatar:     participant.UserAvatar,
        PermissionLevel: participant.Permission.String(),
        ClientType:     participant.ClientType,
        JoinedAt:       participant.JoinedAt,
    }

    if err := s.db.WithContext(ctx).Create(dbParticipant).Error; err != nil {
        return status.Errorf(codes.Internal, "保存参与者信息失败: %v", err)
    }

    // 缓存到Redis
    s.cacheParticipant(ctx, session.ID, participant)

    // 发送用户加入事件给其他参与者
    s.broadcastEvent(ctx, session.ID, &pb.CollaborationEvent{
        Event: &pb.CollaborationEvent_UserJoined{
            UserJoined: &pb.UserJoinedEvent{
                User: &pb.UserInfo{
                    UserId:   participant.UserID,
                    UserName: participant.UserName,
                    UserAvatar: participant.UserAvatar,
                    PermissionLevel: pb.PermissionLevel(participant.Permission),
                },
                JoinedAt: timestamppb.Now(),
            },
        },
    }, req.UserId) // 排除自己

    // 发送当前会话状态给新参与者
    s.sendSessionState(stream, session, participant)

    // 启动事件监听协程
    eventCh := s.eventBus.Subscribe(ctx, fmt.Sprintf("session.%s.events", session.ID))

    // 保持连接并转发事件
    for {
        select {
        case event := <-eventCh:
            if err := stream.Send(event.(*pb.CollaborationEvent)); err != nil {
                log.Printf("发送事件失败: %v", err)
                return err
            }
        case <-ctx.Done():
            s.handleUserLeave(ctx, session.ID, req.UserId)
            return nil
        }
    }
}

// SyncOperation 同步操作 (双向流式)
func (s *CollaborationServer) SyncOperation(
    stream pb.CollaborationService_SyncOperationServer,
) error {
    ctx := stream.Context()

    // 接收操作
    go func() {
        for {
            batch, err := stream.Recv()
            if err != nil {
                log.Printf("接收操作失败: %v", err)
                return
            }

            // 处理操作批次
            if err := s.processOperationBatch(ctx, batch); err != nil {
                // 发送错误确认
                for _, op := range batch.Operations {
                    stream.Send(&pb.OperationAck{
                        OperationId:   op.OperationId,
                        Status:        pb.AckStatus_ACK_STATUS_REJECTED,
                        ErrorMessage:  err.Error(),
                    })
                }
                continue
            }

            // 发送成功确认
            for _, op := range batch.Operations {
                stream.Send(&pb.OperationAck{
                    OperationId:  op.OperationId,
                    Status:       pb.AckStatus_ACK_STATUS_SUCCESS,
                    ServerClock:  batch.ServerClock,
                })
            }
        }
    }()

    // 等待上下文取消
    <-ctx.Done()
    return ctx.Err()
}

// processOperationBatch 处理操作批次
func (s *CollaborationServer) processOperationBatch(
    ctx context.Context, 
    batch *pb.OperationBatch,
) error {
    session, err := s.getSession(ctx, batch.SessionId)
    if err != nil {
        return err
    }

    // 验证用户权限
    if err := s.checkSessionPermission(ctx, batch.SessionId, batch.UserId, PermissionAction_EDIT); err != nil {
        return err
    }

    // 获取并递增服务器时钟
    serverClock := s.incrementServerClock(ctx, session.ID)

    // 应用Yjs更新
    if len(batch.YjsUpdate) > 0 {
        if err := s.yjsManager.ApplyUpdate(session.ID, batch.YjsUpdate); err != nil {
            return fmt.Errorf("应用Yjs更新失败: %w", err)
        }
    }

    // 持久化操作
    for _, op := range batch.Operations {
        dbOp := &models.OperationLog{
            SessionID:     session.ID,
            OperationID:   op.OperationId,
            UserID:        batch.UserId,
            ClientClock:   batch.ClientClock,
            ServerClock:   serverClock,
            OperationType: op.Type.String(),
            TargetID:      &op.TargetId,
            OperationData: op.Data,
            YjsUpdate:     batch.YjsUpdate,
            Metadata:      op.Metadata,
        }

        if err := s.db.WithContext(ctx).Create(dbOp).Error; err != nil {
            return fmt.Errorf("持久化操作失败: %w", err)
        }
    }

    // 广播操作给其他参与者
    for _, op := range batch.Operations {
        s.broadcastEvent(ctx, session.ID, &pb.CollaborationEvent{
            Event: &pb.CollaborationEvent_OperationReceived{
                OperationReceived: &pb.OperationReceivedEvent{
                    Operation:    op,
                    FromUserId:   batch.UserId,
                },
            },
        }, batch.UserId)
    }

    // 更新会话状态
    session.mu.Lock()
    session.ServerClock = serverClock
    session.YjsState = s.yjsManager.GetState(session.ID)
    session.mu.Unlock()

    // 异步保存Yjs状态
    go s.saveYjsState(session.ID, session.YjsState)

    return nil
}

// UndoOperation 撤销操作
func (s *CollaborationServer) UndoOperation(
    ctx context.Context, 
    req *pb.UndoOperationRequest,
) (*pb.UndoOperationResponse, error) {
    // 获取用户的操作历史
    var operations []models.OperationLog
    if err := s.db.WithContext(ctx).
        Where("session_id = ? AND user_id = ? AND is_undone = ?", 
            req.SessionId, req.UserId, false).
        Order("server_clock DESC").
        Limit(int(req.Steps)).
        Find(&operations).Error; err != nil {
        return nil, status.Errorf(codes.Internal, "查询操作历史失败: %v", err)
    }

    if len(operations) == 0 {
        return &pb.UndoOperationResponse{Success: true}, nil
    }

    var undoneIDs []string

    // 执行撤销
    for _, op := range operations {
        // 生成撤销操作
        undoOp := s.generateUndoOperation(&op)

        // 应用撤销
        if err := s.applyUndoOperation(ctx, req.SessionId, undoOp); err != nil {
            log.Printf("撤销操作失败: %v", err)
            continue
        }

        // 标记为已撤销
        op.IsUndone = true
        op.UndoneAt = ptr(time.Now())
        op.UndoneBy = &req.UserId

        if err := s.db.WithContext(ctx).Save(&op).Error; err != nil {
            log.Printf("更新操作状态失败: %v", err)
        }

        undoneIDs = append(undoneIDs, op.OperationID)
    }

    return &pb.UndoOperationResponse{
        Success:             true,
        UndoneOperationIds:  undoneIDs,
    }, nil
}

// ==================== 辅助方法 ====================

func (s *CollaborationServer) getSession(ctx context.Context, sessionID string) (*Session, error) {
    // 先查内存缓存
    s.sessionMu.RLock()
    if session, ok := s.sessions[sessionID]; ok {
        s.sessionMu.RUnlock()
        return session, nil
    }
    s.sessionMu.RUnlock()

    // 查数据库
    var dbSession models.CollaborationSession
    if err := s.db.WithContext(ctx).First(&dbSession, "id = ?", sessionID).Error; err != nil {
        return nil, err
    }

    // 重建内存对象
    session := &Session{
        ID:          dbSession.ID,
        DocumentID:  dbSession.DocumentID,
        TenantID:    dbSession.TenantID,
        ServerClock: dbSession.ServerClock,
        Participants: make(map[string]*Participant),
    }

    // 加载参与者
    var dbParticipants []models.SessionParticipant
    s.db.WithContext(ctx).Where("session_id = ? AND is_active = ?", sessionID, true).Find(&dbParticipants)

    for _, p := range dbParticipants {
        session.Participants[p.UserID] = &Participant{
            UserID:   p.UserID,
            UserName: p.UserName,
            IsActive: p.IsActive,
        }
    }

    // 缓存
    s.sessionMu.Lock()
    s.sessions[sessionID] = session
    s.sessionMu.Unlock()

    return session, nil
}

func (s *CollaborationServer) incrementServerClock(ctx context.Context, sessionID string) int64 {
    // 使用Redis原子递增
    clock, err := s.redis.Incr(ctx, fmt.Sprintf("session:%s:clock", sessionID)).Result()
    if err != nil {
        // 降级到数据库
        s.db.Exec("UPDATE collaboration_sessions SET server_clock = server_clock + 1 WHERE id = ?", sessionID)
        var session models.CollaborationSession
        s.db.First(&session, "id = ?", sessionID)
        return session.ServerClock
    }
    return clock
}

func (s *CollaborationServer) broadcastEvent(
    ctx context.Context, 
    sessionID string, 
    event *pb.CollaborationEvent,
    excludeUserID string,
) {
    // 发布到消息队列
    s.eventBus.Publish(ctx, fmt.Sprintf("session.%s.events", sessionID), event)

    // 更新Redis中的参与者状态
    // ...
}

func (s *CollaborationServer) cleanupTask() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        ctx := context.Background()

        // 清理过期会话
        s.db.WithContext(ctx).
            Model(&models.CollaborationSession{}).
            Where("expires_at < ? AND status = ?", time.Now(), "active").
            Update("status", "closed")

        // 清理不活跃参与者
        s.db.WithContext(ctx).
            Model(&models.SessionParticipant{}).
            Where("last_activity_at < ? AND is_active = ?", 
                time.Now().Add(-10*time.Minute), true).
            Update("is_active", false)
    }
}

// 其他辅助方法...
```

### 1.4.2 Yjs文档管理器

```go
package yjs

import (
    "sync"

    "github.com/yjs/ydoc-go"
)

// YjsDocumentManager 管理Yjs文档
type YjsDocumentManager struct {
    documents map[string]*YjsDocument
    mu        sync.RWMutex
}

type YjsDocument struct {
    ID       string
    Doc      *ydoc.YDoc
    Updates  [][]byte
    mu       sync.RWMutex
}

func NewYjsDocumentManager() *YjsDocumentManager {
    return &YjsDocumentManager{
        documents: make(map[string]*YjsDocument),
    }
}

func (m *YjsDocumentManager) GetOrCreateDocument(docID string) *YjsDocument {
    m.mu.Lock()
    defer m.mu.Unlock()

    if doc, ok := m.documents[docID]; ok {
        return doc
    }

    doc := &YjsDocument{
        ID:      docID,
        Doc:     ydoc.New(),
        Updates: make([][]byte, 0),
    }

    m.documents[docID] = doc
    return doc
}

func (m *YjsDocumentManager) ApplyUpdate(docID string, update []byte) error {
    doc := m.GetOrCreateDocument(docID)

    doc.mu.Lock()
    defer doc.mu.Unlock()

    // 应用Yjs更新
    if err := doc.Doc.ApplyUpdate(update); err != nil {
        return err
    }

    // 保存更新历史
    doc.Updates = append(doc.Updates, update)

    return nil
}

func (m *YjsDocumentManager) GetState(docID string) []byte {
    doc := m.GetOrCreateDocument(docID)

    doc.mu.RLock()
    defer doc.mu.RUnlock()

    return doc.Doc.EncodeStateAsUpdate()
}

func (m *YjsDocumentManager) GetStateVector(docID string) []byte {
    doc := m.GetOrCreateDocument(docID)

    doc.mu.RLock()
    defer doc.mu.RUnlock()

    return doc.Doc.EncodeStateVector()
}

func (m *YjsDocumentManager) MergeUpdates(docID string, updates [][]byte) ([]byte, error) {
    return ydoc.MergeUpdates(updates)
}

func (m *YjsDocumentManager) DiffUpdate(docID string, stateVector []byte) ([]byte, error) {
    doc := m.GetOrCreateDocument(docID)

    doc.mu.RLock()
    defer doc.mu.RUnlock()

    return doc.Doc.EncodeStateAsUpdateWithStateVector(stateVector)
}
```

## 1.5 异常处理策略

```go
package errors

import (
    "errors"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// 协作服务错误类型
var (
    ErrSessionNotFound     = errors.New("会话不存在")
    ErrSessionExpired      = errors.New("会话已过期")
    ErrSessionClosed       = errors.New("会话已关闭")
    ErrPermissionDenied    = errors.New("权限不足")
    ErrInvalidOperation    = errors.New("无效操作")
    ErrConflictDetected    = errors.New("检测到冲突")
    ErrRateLimitExceeded   = errors.New("请求频率超限")
    ErrUserNotInSession    = errors.New("用户不在会话中")
    ErrConcurrentEdit      = errors.New("并发编辑冲突")
    ErrYjsApplyFailed      = errors.New("Yjs更新应用失败")
    ErrOperationTooLarge   = errors.New("操作数据过大")
)

// CollaborationError 协作服务错误
type CollaborationError struct {
    Code       codes.Code
    Message    string
    Details    map[string]interface{}
    Retryable  bool
}

func (e *CollaborationError) Error() string {
    return e.Message
}

func (e *CollaborationError) GRPCStatus() *status.Status {
    st := status.New(e.Code, e.Message)
    // 添加详细信息
    return st
}

// 错误转换函数
func ConvertToGRPCError(err error) error {
    var collabErr *CollaborationError
    if errors.As(err, &collabErr) {
        return collabErr.GRPCStatus().Err()
    }

    switch {
    case errors.Is(err, ErrSessionNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, ErrSessionExpired):
        return status.Error(codes.DeadlineExceeded, err.Error())
    case errors.Is(err, ErrPermissionDenied):
        return status.Error(codes.PermissionDenied, err.Error())
    case errors.Is(err, ErrInvalidOperation):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, ErrConflictDetected):
        return status.Error(codes.Aborted, err.Error())
    case errors.Is(err, ErrRateLimitExceeded):
        return status.Error(codes.ResourceExhausted, err.Error())
    default:
        return status.Error(codes.Internal, err.Error())
    }
}

// 错误处理中间件
func ErrorHandlingInterceptor(
    ctx context.Context, 
    req interface{}, 
    info *grpc.UnaryServerInfo, 
    handler grpc.UnaryHandler,
) (interface{}, error) {
    resp, err := handler(ctx, req)
    if err != nil {
        // 记录错误日志
        log.Printf("gRPC错误 [%s]: %v", info.FullMethod, err)

        // 转换错误
        return nil, ConvertToGRPCError(err)
    }
    return resp, nil
}

// 恢复中间件
func RecoveryInterceptor(
    ctx context.Context, 
    req interface{}, 
    info *grpc.UnaryServerInfo, 
    handler grpc.UnaryHandler,
) (resp interface{}, err error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic recovered [%s]: %v", info.FullMethod, r)
            err = status.Errorf(codes.Internal, "内部服务器错误")
        }
    }()
    return handler(ctx, req)
}

// 重试策略
var RetryPolicy = map[codes.Code]bool{
    codes.DeadlineExceeded: true,
    codes.Unavailable:      true,
    codes.Aborted:          true,
    codes.ResourceExhausted: true,
}

func ShouldRetry(err error) bool {
    st, ok := status.FromError(err)
    if !ok {
        return false
    }
    return RetryPolicy[st.Code()]
}
```

---

---

# 2. 几何服务详细设计

## 2.1 服务概述

几何服务负责处理建筑设计中的几何数据，包括2D/3D几何体的存储、查询、变换和分析。基于PostGIS扩展提供强大的空间数据处理能力。

### 核心功能
- 几何数据存储与管理
- 空间索引与查询
- 几何变换操作
- 几何验证与修复
- 批量几何处理

## 2.2 gRPC接口定义

```protobuf
syntax = "proto3";

package geometry.v1;

option go_package = "github.com/archplatform/geometry-service/api/v1";

import "google/protobuf/struct.proto";
import "google/protobuf/empty.proto";

// 几何服务
service GeometryService {
  // 基础CRUD
  rpc CreateGeometry(CreateGeometryRequest) returns (Geometry);
  rpc GetGeometry(GetGeometryRequest) returns (Geometry);
  rpc UpdateGeometry(UpdateGeometryRequest) returns (Geometry);
  rpc DeleteGeometry(DeleteGeometryRequest) returns (google.protobuf.Empty);
  rpc BatchCreateGeometry(BatchCreateGeometryRequest) returns (BatchGeometryResponse);
  rpc BatchUpdateGeometry(BatchUpdateGeometryRequest) returns (BatchGeometryResponse);
  rpc BatchDeleteGeometry(BatchDeleteGeometryRequest) returns (google.protobuf.Empty);

  // 几何查询
  rpc QueryGeometry(QueryGeometryRequest) returns (GeometryCollection);
  rpc SpatialQuery(SpatialQueryRequest) returns (GeometryCollection);
  rpc NearestQuery(NearestQueryRequest) returns (GeometryCollection);
  rpc IntersectionQuery(IntersectionQueryRequest) returns (GeometryCollection);
  rpc BoundingBoxQuery(BoundingBoxQueryRequest) returns (GeometryCollection);

  // 几何变换
  rpc Transform(TransformRequest) returns (Geometry);
  rpc BatchTransform(BatchTransformRequest) returns (BatchGeometryResponse);

  // 几何分析
  rpc CalculateArea(CalculateRequest) returns (CalculateResponse);
  rpc CalculateVolume(CalculateRequest) returns (CalculateResponse);
  rpc CalculateDistance(DistanceRequest) returns (DistanceResponse);
  rpc ValidateGeometry(ValidateRequest) returns (ValidateResponse);
  rpc RepairGeometry(RepairRequest) returns (Geometry);

  // 几何关系
  rpc CheckIntersection(GeometryRelationRequest) returns (RelationResponse);
  rpc CheckContains(GeometryRelationRequest) returns (RelationResponse);
  rpc CheckWithin(GeometryRelationRequest) returns (RelationResponse);
  rpc CheckTouches(GeometryRelationRequest) returns (RelationResponse);

  // 导入导出
  rpc ImportGeometry(ImportRequest) returns (ImportResponse);
  rpc ExportGeometry(ExportRequest) returns (ExportResponse);
  rpc ConvertFormat(ConvertRequest) returns (ConvertResponse);
}

// ==================== 几何数据消息 ====================

message Geometry {
  string id = 1;
  string element_id = 2;
  string document_id = 3;
  string tenant_id = 4;
  GeometryType type = 5;
  oneof geometry_data {
    PointGeometry point = 6;
    LineGeometry line = 7;
    PolygonGeometry polygon = 8;
    PolylineGeometry polyline = 9;
    CircleGeometry circle = 10;
    ArcGeometry arc = 11;
    MeshGeometry mesh = 12;
    BrepGeometry brep = 13;
    NurbsGeometry nurbs = 14;
    CompositeGeometry composite = 15;
  }
  SpatialReference srid = 16;
  BoundingBox bbox = 17;
  map<string, string> metadata = 18;
  google.protobuf.Struct properties = 19;
  int64 version = 20;
  string created_by = 21;
  string updated_by = 22;
  int64 created_at = 23;
  int64 updated_at = 24;
}

message PointGeometry {
  double x = 1;
  double y = 2;
  double z = 3;
}

message LineGeometry {
  PointGeometry start = 1;
  PointGeometry end = 2;
}

message PolygonGeometry {
  repeated PointGeometry exterior_ring = 1;
  repeated LinearRing interior_rings = 2;
  bool is_planar = 3;
}

message LinearRing {
  repeated PointGeometry points = 1;
}

message PolylineGeometry {
  repeated PointGeometry points = 1;
  bool is_closed = 2;
}

message CircleGeometry {
  PointGeometry center = 1;
  double radius = 2;
  Vector3 normal = 3;
}

message ArcGeometry {
  PointGeometry center = 1;
  double radius = 2;
  double start_angle = 3;
  double end_angle = 4;
  Vector3 normal = 5;
}

message MeshGeometry {
  repeated Vertex vertices = 1;
  repeated Face faces = 2;
  repeated Vector3 normals = 3;
  repeated UVCoordinate uvs = 4;
}

message Vertex {
  double x = 1;
  double y = 2;
  double z = 3;
}

message Face {
  repeated int32 vertex_indices = 1;
  int32 material_index = 2;
  Vector3 normal = 3;
}

message Vector3 {
  double x = 1;
  double y = 2;
  double z = 3;
}

message UVCoordinate {
  double u = 1;
  double v = 2;
}

message BrepGeometry {
  repeated BrepFace faces = 1;
  repeated BrepEdge edges = 2;
  repeated BrepVertex vertices = 3;
  repeated BrepLoop loops = 4;
  bool is_solid = 5;
  bool is_manifold = 6;
}

message BrepFace {
  string surface_id = 1;
  repeated string loop_ids = 2;
  bool orientation = 3;
}

message BrepEdge {
  string curve_id = 1;
  string start_vertex = 2;
  string end_vertex = 3;
  double tolerance = 4;
}

message BrepVertex {
  string id = 1;
  PointGeometry point = 2;
  double tolerance = 3;
}

message BrepLoop {
  string id = 1;
  repeated string edge_ids = 2;
  bool is_outer = 3;
}

message NurbsGeometry {
  int32 degree_u = 1;
  int32 degree_v = 2;
  repeated double knots_u = 3;
  repeated double knots_v = 4;
  repeated NurbsControlPoint control_points = 5;
  repeated double weights = 6;
  bool is_rational = 7;
}

message NurbsControlPoint {
  double x = 1;
  double y = 2;
  double z = 3;
  double w = 4;
}

message CompositeGeometry {
  repeated string child_geometry_ids = 1;
  Transform transform = 2;
}

message Transform {
  repeated double matrix = 1;  // 4x4矩阵，16个元素
}

message BoundingBox {
  double min_x = 1;
  double min_y = 2;
  double min_z = 3;
  double max_x = 4;
  double max_y = 5;
  double max_z = 6;
}

message SpatialReference {
  int32 srid = 1;
  string wkt = 2;
}

// ==================== 请求/响应消息 ====================

message CreateGeometryRequest {
  string element_id = 1;
  string document_id = 2;
  string tenant_id = 3;
  Geometry geometry = 4;
}

message GetGeometryRequest {
  string id = 1;
  string tenant_id = 2;
}

message UpdateGeometryRequest {
  string id = 1;
  string tenant_id = 2;
  Geometry geometry = 3;
  bool create_new_version = 4;
}

message DeleteGeometryRequest {
  string id = 1;
  string tenant_id = 2;
  bool permanent = 3;
}

message BatchCreateGeometryRequest {
  string document_id = 1;
  string tenant_id = 2;
  repeated Geometry geometries = 3;
}

message BatchUpdateGeometryRequest {
  string tenant_id = 1;
  repeated Geometry geometries = 2;
}

message BatchDeleteGeometryRequest {
  string tenant_id = 1;
  repeated string ids = 2;
}

message BatchGeometryResponse {
  repeated Geometry geometries = 1;
  repeated ErrorInfo errors = 2;
  int32 success_count = 3;
  int32 failed_count = 4;
}

message ErrorInfo {
  string id = 1;
  string error_code = 2;
  string error_message = 3;
}

// ==================== 查询消息 ====================

message QueryGeometryRequest {
  string tenant_id = 1;
  string document_id = 2;
  repeated GeometryType types = 3;
  BoundingBox bbox = 4;
  string filter = 5;
  int32 page_size = 6;
  string page_token = 7;
  SortOrder sort_order = 8;
}

message SpatialQueryRequest {
  string tenant_id = 1;
  string document_id = 2;
  Geometry reference_geometry = 3;
  SpatialOperator operator = 4;
  double distance = 5;
  int32 page_size = 6;
  string page_token = 7;
}

message NearestQueryRequest {
  string tenant_id = 1;
  string document_id = 2;
  PointGeometry reference_point = 3;
  int32 limit = 4;
  double max_distance = 5;
}

message IntersectionQueryRequest {
  string tenant_id = 1;
  string document_id = 2;
  Geometry geometry = 3;
  int32 page_size = 4;
  string page_token = 5;
}

message BoundingBoxQueryRequest {
  string tenant_id = 1;
  string document_id = 2;
  BoundingBox bbox = 3;
  bool intersect = 4;  // true=相交, false=包含
  int32 page_size = 5;
  string page_token = 6;
}

message GeometryCollection {
  repeated Geometry geometries = 1;
  int32 total_count = 2;
  string next_page_token = 3;
}

// ==================== 变换消息 ====================

message TransformRequest {
  string id = 1;
  string tenant_id = 2;
  TransformOperation operation = 3;
  oneof params {
    TranslateParams translate = 4;
    RotateParams rotate = 5;
    ScaleParams scale = 6;
    MirrorParams mirror = 7;
    MatrixParams matrix = 8;
  }
}

message TransformOperation {
  enum Type {
    TRANSLATE = 0;
    ROTATE = 1;
    SCALE = 2;
    MIRROR = 3;
    MATRIX = 4;
  }
  Type type = 1;
}

message TranslateParams {
  double dx = 1;
  double dy = 2;
  double dz = 3;
}

message RotateParams {
  PointGeometry center = 1;
  Vector3 axis = 2;
  double angle = 3;  // 弧度
}

message ScaleParams {
  PointGeometry center = 1;
  double sx = 2;
  double sy = 3;
  double sz = 4;
}

message MirrorParams {
  PointGeometry point = 1;
  Vector3 normal = 2;
}

message MatrixParams {
  repeated double values = 1;  // 16个元素
}

message BatchTransformRequest {
  string tenant_id = 1;
  repeated string ids = 2;
  TransformOperation operation = 3;
  oneof params {
    TranslateParams translate = 4;
    RotateParams rotate = 5;
    ScaleParams scale = 6;
    MirrorParams mirror = 7;
    MatrixParams matrix = 8;
  }
}

// ==================== 分析消息 ====================

message CalculateRequest {
  string id = 1;
  string tenant_id = 2;
  CalculationUnit unit = 3;
}

message CalculateResponse {
  double value = 1;
  string unit = 2;
  CalculationPrecision precision = 3;
}

message DistanceRequest {
  Geometry geometry1 = 1;
  Geometry geometry2 = 2;
  DistanceType type = 3;
}

message DistanceResponse {
  double distance = 1;
  PointGeometry closest_point1 = 2;
  PointGeometry closest_point2 = 3;
}

message ValidateRequest {
  string id = 1;
  string tenant_id = 2;
  ValidationLevel level = 3;
}

message ValidateResponse {
  bool is_valid = 1;
  repeated ValidationError errors = 2;
  ValidationLevel checked_level = 3;
}

message ValidationError {
  ValidationErrorType type = 1;
  string message = 2;
  string location = 3;
  Severity severity = 4;
}

message RepairRequest {
  string id = 1;
  string tenant_id = 2;
  RepairStrategy strategy = 3;
}

// ==================== 关系消息 ====================

message GeometryRelationRequest {
  string geometry1_id = 1;
  string geometry2_id = 2;
  string tenant_id = 3;
  double tolerance = 4;
}

message RelationResponse {
  bool result = 1;
  RelationType relation = 2;
  double distance = 3;
}

// ==================== 导入导出消息 ====================

message ImportRequest {
  string tenant_id = 1;
  string document_id = 2;
  FileFormat format = 3;
  bytes data = 4;
  ImportOptions options = 5;
}

message ImportResponse {
  repeated Geometry geometries = 1;
  ImportStatistics statistics = 2;
  repeated ImportWarning warnings = 3;
}

message ExportRequest {
  string tenant_id = 1;
  repeated string ids = 2;
  FileFormat format = 3;
  ExportOptions options = 4;
}

message ExportResponse {
  bytes data = 1;
  string filename = 2;
  string mime_type = 3;
}

message ConvertRequest {
  Geometry geometry = 1;
  FileFormat target_format = 2;
  ConversionOptions options = 3;
}

message ConvertResponse {
  bytes data = 1;
  FileFormat format = 2;
}

// ==================== 枚举定义 ====================

enum GeometryType {
  GEOMETRY_TYPE_UNSPECIFIED = 0;
  POINT = 1;
  LINE = 2;
  POLYGON = 3;
  POLYLINE = 4;
  CIRCLE = 5;
  ARC = 6;
  MESH = 7;
  BREP = 8;
  NURBS = 9;
  COMPOSITE = 10;
}

enum SpatialOperator {
  SPATIAL_OPERATOR_UNSPECIFIED = 0;
  INTERSECTS = 1;
  CONTAINS = 2;
  WITHIN = 3;
  TOUCHES = 4;
  OVERLAPS = 5;
  DISJOINT = 6;
  WITHIN_DISTANCE = 7;
}

enum SortOrder {
  SORT_ORDER_UNSPECIFIED = 0;
  CREATED_AT_ASC = 1;
  CREATED_AT_DESC = 2;
  UPDATED_AT_ASC = 3;
  UPDATED_AT_DESC = 4;
  AREA_ASC = 5;
  AREA_DESC = 6;
}

enum CalculationUnit {
  CALCULATION_UNIT_UNSPECIFIED = 0;
  SQUARE_METERS = 1;
  SQUARE_FEET = 2;
  CUBIC_METERS = 3;
  CUBIC_FEET = 4;
}

enum CalculationPrecision {
  CALCULATION_PRECISION_UNSPECIFIED = 0;
  LOW = 1;
  MEDIUM = 2;
  HIGH = 3;
  EXACT = 4;
}

enum ValidationLevel {
  VALIDATION_LEVEL_UNSPECIFIED = 0;
  BASIC = 1;
  STANDARD = 2;
  STRICT = 3;
}

enum ValidationErrorType {
  VALIDATION_ERROR_TYPE_UNSPECIFIED = 0;
  SELF_INTERSECTION = 1;
  DEGENERATE_GEOMETRY = 2;
  INVALID_TOPOLOGY = 3;
  NAN_VALUES = 4;
  ZERO_LENGTH = 5;
  ZERO_AREA = 6;
  OPEN_SOLID = 7;
  NON_MANIFOLD = 8;
}

enum Severity {
  SEVERITY_UNSPECIFIED = 0;
  INFO = 1;
  WARNING = 2;
  ERROR = 3;
  CRITICAL = 4;
}

enum RepairStrategy {
  REPAIR_STRATEGY_UNSPECIFIED = 0;
  AUTO = 1;
  CONSERVATIVE = 2;
  AGGRESSIVE = 3;
}

enum RelationType {
  RELATION_TYPE_UNSPECIFIED = 0;
  EQUALS = 1;
  DISJOINT = 2;
  INTERSECTS = 3;
  TOUCHES = 4;
  CROSSES = 5;
  WITHIN = 6;
  CONTAINS = 7;
  OVERLAPS = 8;
}

enum FileFormat {
  FILE_FORMAT_UNSPECIFIED = 0;
  OBJ = 1;
  STL = 2;
  PLY = 3;
  FBX = 4;
  GLTF = 5;
  GLB = 6;
  DAE = 7;
  STEP = 8;
  IGES = 9;
  IFC = 10;
  DWG = 11;
  DXF = 12;
  GEOJSON = 13;
  WKT = 14;
  WKB = 15;
}

enum DistanceType {
  DISTANCE_TYPE_UNSPECIFIED = 0;
  EUCLIDEAN = 1;
  MANHATTAN = 2;
  ALONG_SURFACE = 3;
}

message ImportOptions {
  bool merge_duplicates = 1;
  double tolerance = 2;
  bool validate = 3;
  map<string, string> parameters = 4;
}

message ImportStatistics {
  int32 total_imported = 1;
  int32 points_imported = 2;
  int32 lines_imported = 3;
  int32 polygons_imported = 4;
  int32 meshes_imported = 5;
}

message ImportWarning {
  string message = 1;
  string entity_id = 2;
  Severity severity = 3;
}

message ExportOptions {
  bool include_normals = 1;
  bool include_uvs = 2;
  bool triangulate = 3;
  double precision = 4;
  map<string, string> parameters = 5;
}

message ConversionOptions {
  double tolerance = 1;
  bool preserve_topology = 2;
}
```

## 2.3 数据库表结构设计

```sql
-- 启用PostGIS扩展
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- ==================== 几何数据主表 ====================
CREATE TABLE geometries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    element_id UUID NOT NULL,
    document_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    geometry_type VARCHAR(32) NOT NULL,

    -- PostGIS几何字段 (使用Geometry类型支持多种几何)
    geom geometry(GeometryZ, 4326),

    -- 3D几何数据 (BREP/Mesh等复杂几何)
    geometry_3d JSONB,

    -- 边界框 (用于快速筛选)
    bbox geometry(Polygon, 4326),
    bbox_3d BOX3D,

    -- 空间参考系统
    srid INTEGER DEFAULT 4326,

    -- 元数据
    metadata JSONB DEFAULT '{}',
    properties JSONB DEFAULT '{}',

    -- 版本控制
    version BIGINT DEFAULT 1,
    is_deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- 审计字段
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_geometry_type CHECK (geometry_type IN (
        'POINT', 'LINE', 'POLYGON', 'POLYLINE', 'CIRCLE', 'ARC',
        'MESH', 'BREP', 'NURBS', 'COMPOSITE'
    ))
);

-- 空间索引
CREATE INDEX idx_geometries_geom ON geometries USING GIST(geom);
CREATE INDEX idx_geometries_bbox ON geometries USING GIST(bbox);
CREATE INDEX idx_geometries_3d ON geometries USING GIST(bbox_3d);

-- B-tree索引
CREATE INDEX idx_geometries_element ON geometries(element_id);
CREATE INDEX idx_geometries_document ON geometries(document_id);
CREATE INDEX idx_geometries_tenant ON geometries(tenant_id);
CREATE INDEX idx_geometries_type ON geometries(geometry_type);
CREATE INDEX idx_geometries_version ON geometries(document_id, version);
CREATE INDEX idx_geometries_deleted ON geometries(is_deleted) WHERE is_deleted = FALSE;

-- JSONB索引
CREATE INDEX idx_geometries_metadata ON geometries USING GIN(metadata);
CREATE INDEX idx_geometries_properties ON geometries USING GIN(properties);

-- ==================== 几何版本历史表 ====================
CREATE TABLE geometry_versions (
    id BIGSERIAL PRIMARY KEY,
    geometry_id UUID NOT NULL REFERENCES geometries(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    geom geometry(GeometryZ, 4326),
    geometry_3d JSONB,
    change_type VARCHAR(32) NOT NULL,
    change_description TEXT,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(geometry_id, version),
    CONSTRAINT chk_change_type CHECK (change_type IN ('CREATE', 'UPDATE', 'DELETE', 'TRANSFORM'))
);

CREATE INDEX idx_geom_versions_geometry ON geometry_versions(geometry_id);
CREATE INDEX idx_geom_versions_version ON geometry_versions(geometry_id, version);

-- ==================== 几何关系表 ====================
CREATE TABLE geometry_relationships (
    id BIGSERIAL PRIMARY KEY,
    geometry_id_1 UUID NOT NULL REFERENCES geometries(id) ON DELETE CASCADE,
    geometry_id_2 UUID NOT NULL REFERENCES geometries(id) ON DELETE CASCADE,
    relationship_type VARCHAR(32) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(geometry_id_1, geometry_id_2, relationship_type),
    CONSTRAINT chk_relationship_type CHECK (relationship_type IN (
        'PARENT_CHILD', 'SIBLING', 'DEPENDENCY', 'CONSTRAINT', 'REFERENCE'
    ))
);

CREATE INDEX idx_geom_relations_g1 ON geometry_relationships(geometry_id_1);
CREATE INDEX idx_geom_relations_g2 ON geometry_relationships(geometry_id_2);
CREATE INDEX idx_geom_relations_type ON geometry_relationships(relationship_type);

-- ==================== 几何缓存表 (用于复杂计算结果) ====================
CREATE TABLE geometry_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    geometry_id UUID NOT NULL REFERENCES geometries(id) ON DELETE CASCADE,
    cache_type VARCHAR(32) NOT NULL,
    cache_data JSONB NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_cache_type CHECK (cache_type IN (
        'AREA', 'VOLUME', 'CENTROID', 'BOUNDARY', 'CONVEX_HULL', 'BUFFER'
    ))
);

CREATE INDEX idx_geom_cache_geometry ON geometry_cache(geometry_id);
CREATE INDEX idx_geom_cache_type ON geometry_cache(cache_type);
CREATE INDEX idx_geom_cache_expires ON geometry_cache(expires_at);

-- ==================== 空间查询日志表 ====================
CREATE TABLE spatial_query_logs (
    id BIGSERIAL PRIMARY KEY,
    query_type VARCHAR(32) NOT NULL,
    query_params JSONB,
    result_count INTEGER,
    execution_time_ms INTEGER,
    tenant_id UUID,
    user_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_query_logs_type ON spatial_query_logs(query_type);
CREATE INDEX idx_query_logs_created ON spatial_query_logs(created_at);

-- ==================== 触发器函数 ====================

-- 自动更新边界框
CREATE OR REPLACE FUNCTION update_geometry_bbox()
RETURNS TRIGGER AS $$
BEGIN
    -- 更新2D边界框
    NEW.bbox := ST_Envelope(NEW.geom);

    -- 更新3D边界框
    NEW.bbox_3d := ST_3DExtent(NEW.geom)::BOX3D;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_bbox
    BEFORE INSERT OR UPDATE ON geometries
    FOR EACH ROW
    EXECUTE FUNCTION update_geometry_bbox();

-- 版本历史记录
CREATE OR REPLACE FUNCTION log_geometry_version()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        INSERT INTO geometry_versions (
            geometry_id, version, geom, geometry_3d, 
            change_type, changed_by, changed_at
        ) VALUES (
            OLD.id, OLD.version, OLD.geom, OLD.geometry_3d,
            'UPDATE', NEW.updated_by, NOW()
        );

        -- 递增版本号
        NEW.version := OLD.version + 1;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_version
    BEFORE UPDATE ON geometries
    FOR EACH ROW
    WHEN (OLD.geom IS DISTINCT FROM NEW.geom OR 
          OLD.geometry_3d IS DISTINCT FROM NEW.geometry_3d)
    EXECUTE FUNCTION log_geometry_version();

-- 自动更新时间戳
CREATE TRIGGER trigger_geometries_updated_at
    BEFORE UPDATE ON geometries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

## 2.4 核心业务逻辑实现

```go
package geometry

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/paulmach/orb"
    "github.com/paulmach/orb/encoding/wkb"
    "github.com/twpayne/go-geom"
)

// GeometryService 几何服务实现
type GeometryService struct {
    pb.UnimplementedGeometryServiceServer

    db        *pgxpool.Pool
    cache     *GeometryCache
    converter *GeometryConverter
    validator *GeometryValidator
}

// CreateGeometry 创建几何体
func (s *GeometryService) CreateGeometry(
    ctx context.Context, 
    req *pb.CreateGeometryRequest,
) (*pb.Geometry, error) {
    // 验证几何数据
    if err := s.validator.Validate(req.Geometry); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "几何验证失败: %v", err)
    }

    // 转换几何数据
    geomData, err := s.converter.ToPostGIS(req.Geometry)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "几何转换失败: %v", err)
    }

    // 序列化3D几何数据
    geom3DJSON, err := json.Marshal(req.Geometry.GetMesh())
    if err != nil {
        geom3DJSON = []byte("null")
    }

    // 插入数据库
    var id string
    err = s.db.QueryRow(ctx, `
        INSERT INTO geometries (
            element_id, document_id, tenant_id, geometry_type,
            geom, geometry_3d, srid, metadata, properties,
            created_by, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
        RETURNING id
    `, 
        req.ElementId, req.DocumentId, req.TenantId, req.Geometry.Type.String(),
        geomData.WKB, geom3DJSON, geomData.SRID, 
        req.Geometry.Metadata, req.Geometry.Properties,
        req.Geometry.CreatedBy,
    ).Scan(&id)

    if err != nil {
        return nil, status.Errorf(codes.Internal, "创建几何体失败: %v", err)
    }

    // 返回创建的几何体
    return s.GetGeometry(ctx, &pb.GetGeometryRequest{Id: id, TenantId: req.TenantId})
}

// SpatialQuery 空间查询
func (s *GeometryService) SpatialQuery(
    ctx context.Context, 
    req *pb.SpatialQueryRequest,
) (*pb.GeometryCollection, error) {
    // 构建空间查询SQL
    var query string
    var args []interface{}

    // 转换参考几何为WKB
    refWKB, err := s.converter.ToWKB(req.ReferenceGeometry)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "参考几何转换失败: %v", err)
    }

    switch req.Operator {
    case pb.SpatialOperator_INTERSECTS:
        query = `
            SELECT id, element_id, document_id, tenant_id, geometry_type,
                   ST_AsBinary(geom), geometry_3d, bbox, srid, metadata, properties,
                   version, created_by, updated_by, created_at, updated_at
            FROM geometries
            WHERE tenant_id = $1 AND document_id = $2 AND is_deleted = FALSE
              AND ST_Intersects(geom, ST_GeomFromWKB($3, $4))
            ORDER BY ST_Distance(geom, ST_GeomFromWKB($3, $4))
            LIMIT $5 OFFSET $6
        `
        args = []interface{}{req.TenantId, req.DocumentId, refWKB, 4326, 
                            req.PageSize, getOffset(req.PageToken)}

    case pb.SpatialOperator_WITHIN_DISTANCE:
        query = `
            SELECT id, element_id, document_id, tenant_id, geometry_type,
                   ST_AsBinary(geom), geometry_3d, bbox, srid, metadata, properties,
                   version, created_by, updated_by, created_at, updated_at
            FROM geometries
            WHERE tenant_id = $1 AND document_id = $2 AND is_deleted = FALSE
              AND ST_DWithin(geom::geography, ST_GeomFromWKB($3, $4)::geography, $5)
            ORDER BY ST_Distance(geom::geography, ST_GeomFromWKB($3, $4)::geography)
            LIMIT $6 OFFSET $7
        `
        args = []interface{}{req.TenantId, req.DocumentId, refWKB, 4326, 
                            req.Distance, req.PageSize, getOffset(req.PageToken)}

    case pb.SpatialOperator_CONTAINS:
        query = `
            SELECT id, element_id, document_id, tenant_id, geometry_type,
                   ST_AsBinary(geom), geometry_3d, bbox, srid, metadata, properties,
                   version, created_by, updated_by, created_at, updated_at
            FROM geometries
            WHERE tenant_id = $1 AND document_id = $2 AND is_deleted = FALSE
              AND ST_Contains(geom, ST_GeomFromWKB($3, $4))
            LIMIT $5 OFFSET $6
        `
        args = []interface{}{req.TenantId, req.DocumentId, refWKB, 4326,
                            req.PageSize, getOffset(req.PageToken)}

    default:
        return nil, status.Errorf(codes.InvalidArgument, "不支持的空间操作: %v", req.Operator)
    }

    // 执行查询
    rows, err := s.db.Query(ctx, query, args...)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "空间查询失败: %v", err)
    }
    defer rows.Close()

    var geometries []*pb.Geometry
    for rows.Next() {
        geom, err := s.scanGeometry(rows)
        if err != nil {
            continue
        }
        geometries = append(geometries, geom)
    }

    return &pb.GeometryCollection{
        Geometries: geometries,
        TotalCount: int32(len(geometries)),
    }, nil
}

// Transform 几何变换
func (s *GeometryService) Transform(
    ctx context.Context, 
    req *pb.TransformRequest,
) (*pb.Geometry, error) {
    // 获取原几何
    existing, err := s.GetGeometry(ctx, &pb.GetGeometryRequest{
        Id:        req.Id,
        TenantId:  req.TenantId,
    })
    if err != nil {
        return nil, err
    }

    // 构建变换矩阵
    transformMatrix, err := s.buildTransformMatrix(req)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "构建变换矩阵失败: %v", err)
    }

    // 应用变换
    transformed, err := s.applyTransform(existing, transformMatrix)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "应用变换失败: %v", err)
    }

    // 更新数据库
    geomWKB, err := s.converter.ToWKB(transformed)
    if err != nil {
        return nil, err
    }

    _, err = s.db.Exec(ctx, `
        UPDATE geometries 
        SET geom = ST_Transform(ST_GeomFromWKB($1, $2), srid),
            updated_at = NOW(),
            updated_by = $3
        WHERE id = $4 AND tenant_id = $5
    `, geomWKB, 4326, "system", req.Id, req.TenantId)

    if err != nil {
        return nil, status.Errorf(codes.Internal, "更新几何失败: %v", err)
    }

    return s.GetGeometry(ctx, &pb.GetGeometryRequest{Id: req.Id, TenantId: req.TenantId})
}

// ValidateGeometry 验证几何
func (s *GeometryService) ValidateGeometry(
    ctx context.Context, 
    req *pb.ValidateRequest,
) (*pb.ValidateResponse, error) {
    geometry, err := s.GetGeometry(ctx, &pb.GetGeometryRequest{
        Id:       req.Id,
        TenantId: req.TenantId,
    })
    if err != nil {
        return nil, err
    }

    var errors []*pb.ValidationError
    isValid := true

    // 基本验证
    if req.Level >= pb.ValidationLevel_BASIC {
        if errs := s.validator.ValidateBasic(geometry); len(errs) > 0 {
            isValid = false
            errors = append(errors, errs...)
        }
    }

    // 标准验证
    if req.Level >= pb.ValidationLevel_STANDARD {
        if errs := s.validator.ValidateStandard(geometry); len(errs) > 0 {
            isValid = false
            errors = append(errors, errs...)
        }
    }

    // 严格验证
    if req.Level >= pb.ValidationLevel_STRICT {
        if errs := s.validator.ValidateStrict(geometry); len(errs) > 0 {
            isValid = false
            errors = append(errors, errs...)
        }
    }

    return &pb.ValidateResponse{
        IsValid:      isValid,
        Errors:       errors,
        CheckedLevel: req.Level,
    }, nil
}

// 辅助方法
func (s *GeometryService) scanGeometry(row pgx.Row) (*pb.Geometry, error) {
    var g pb.Geometry
    var geomWKB []byte
    var geom3DJSON []byte
    var bboxWKB []byte
    var createdAt, updatedAt interface{}

    err := row.Scan(
        &g.Id, &g.ElementId, &g.DocumentId, &g.TenantId, &g.Type,
        &geomWKB, &geom3DJSON, &bboxWKB, &g.Srid, &g.Metadata, &g.Properties,
        &g.Version, &g.CreatedBy, &g.UpdatedBy, &createdAt, &updatedAt,
    )
    if err != nil {
        return nil, err
    }

    // 解析WKB几何
    if len(geomWKB) > 0 {
        geom, err := wkb.Unmarshal(geomWKB)
        if err == nil {
            g = s.converter.FromOrbGeometry(geom, &g)
        }
    }

    return &g, nil
}

func (s *GeometryService) buildTransformMatrix(req *pb.TransformRequest) ([]float64, error) {
    switch req.Operation.Type {
    case pb.TransformOperation_TRANSLATE:
        p := req.GetTranslate()
        return []float64{
            1, 0, 0, p.Dx,
            0, 1, 0, p.Dy,
            0, 0, 1, p.Dz,
            0, 0, 0, 1,
        }, nil

    case pb.TransformOperation_ROTATE:
        p := req.GetRotate()
        return s.buildRotationMatrix(p.Center, p.Axis, p.Angle)

    case pb.TransformOperation_SCALE:
        p := req.GetScale()
        return s.buildScaleMatrix(p.Center, p.Sx, p.Sy, p.Sz)

    case pb.TransformOperation_MATRIX:
        return req.GetMatrix().Values, nil

    default:
        return nil, fmt.Errorf("不支持的变换类型")
    }
}
```

## 2.5 几何序列化/反序列化

```go
package geometry

import (
    "encoding/binary"
    "fmt"

    "github.com/paulmach/orb"
    "github.com/paulmach/orb/encoding/wkb"
    "github.com/paulmach/orb/encoding/wkt"
    "github.com/twpayne/go-geom"
)

// GeometryConverter 几何转换器
type GeometryConverter struct {
    srid int
}

func NewGeometryConverter(srid int) *GeometryConverter {
    return &GeometryConverter{srid: srid}
}

// ToWKB 将Protobuf几何转换为WKB
func (c *GeometryConverter) ToWKB(g *pb.Geometry) ([]byte, error) {
    orbGeom := c.toOrbGeometry(g)
    return wkb.Marshal(orbGeom, binary.LittleEndian)
}

// FromWKB 从WKB解析几何
func (c *GeometryConverter) FromWKB(wkbData []byte) (*pb.Geometry, error) {
    geom, err := wkb.Unmarshal(wkbData)
    if err != nil {
        return nil, err
    }
    return c.FromOrbGeometry(geom, nil), nil
}

// ToWKT 转换为WKT格式
func (c *GeometryConverter) ToWKT(g *pb.Geometry) (string, error) {
    orbGeom := c.toOrbGeometry(g)
    return wkt.Marshal(orbGeom), nil
}

// ToGeoJSON 转换为GeoJSON格式
func (c *GeometryConverter) ToGeoJSON(g *pb.Geometry) ([]byte, error) {
    orbGeom := c.toOrbGeometry(g)
    return orbGeoJSON.Marshal(orbGeom)
}

// toOrbGeometry 将Protobuf几何转换为orb几何
func (c *GeometryConverter) toOrbGeometry(g *pb.Geometry) orb.Geometry {
    switch g.Type {
    case pb.GeometryType_POINT:
        p := g.GetPoint()
        return orb.Point{p.X, p.Y}

    case pb.GeometryType_LINE:
        l := g.GetLine()
        return orb.LineString{
            {l.Start.X, l.Start.Y},
            {l.End.X, l.End.Y},
        }

    case pb.GeometryType_POLYGON:
        p := g.GetPolygon()
        poly := orb.Polygon{}

        // 外环
        exterior := make(orb.Ring, len(p.ExteriorRing))
        for i, pt := range p.ExteriorRing {
            exterior[i] = orb.Point{pt.X, pt.Y}
        }
        poly = append(poly, exterior)

        // 内环
        for _, ring := range p.InteriorRings {
            interior := make(orb.Ring, len(ring.Points))
            for i, pt := range ring.Points {
                interior[i] = orb.Point{pt.X, pt.Y}
            }
            poly = append(poly, interior)
        }

        return poly

    case pb.GeometryType_POLYLINE:
        pl := g.GetPolyline()
        ls := make(orb.LineString, len(pl.Points))
        for i, pt := range pl.Points {
            ls[i] = orb.Point{pt.X, pt.Y}
        }
        return ls

    default:
        return nil
    }
}

// FromOrbGeometry 从orb几何转换为Protobuf几何
func (c *GeometryConverter) FromOrbGeometry(geom orb.Geometry, template *pb.Geometry) *pb.Geometry {
    g := &pb.Geometry{}
    if template != nil {
        g.Id = template.Id
        g.ElementId = template.ElementId
        g.DocumentId = template.DocumentId
        g.TenantId = template.TenantId
        g.Metadata = template.Metadata
        g.Properties = template.Properties
    }

    switch v := geom.(type) {
    case orb.Point:
        g.Type = pb.GeometryType_POINT
        g.GeometryData = &pb.Geometry_Point{
            Point: &pb.PointGeometry{X: v[0], Y: v[1]},
        }

    case orb.LineString:
        g.Type = pb.GeometryType_POLYLINE
        points := make([]*pb.PointGeometry, len(v))
        for i, pt := range v {
            points[i] = &pb.PointGeometry{X: pt[0], Y: pt[1]}
        }
        g.GeometryData = &pb.Geometry_Polyline{
            Polyline: &pb.PolylineGeometry{Points: points},
        }

    case orb.Polygon:
        g.Type = pb.GeometryType_POLYGON
        poly := &pb.PolygonGeometry{}

        if len(v) > 0 {
            // 外环
            exterior := make([]*pb.PointGeometry, len(v[0]))
            for i, pt := range v[0] {
                exterior[i] = &pb.PointGeometry{X: pt[0], Y: pt[1]}
            }
            poly.ExteriorRing = exterior

            // 内环
            for i := 1; i < len(v); i++ {
                ring := &pb.LinearRing{}
                for _, pt := range v[i] {
                    ring.Points = append(ring.Points, &pb.PointGeometry{X: pt[0], Y: pt[1]})
                }
                poly.InteriorRings = append(poly.InteriorRings, ring)
            }
        }

        g.GeometryData = &pb.Geometry_Polygon{
            Polygon: poly,
        }

    case orb.MultiPolygon:
        // 处理多面几何
        // ...
    }

    return g
}

// ToPostGIS 转换为PostGIS格式
func (c *GeometryConverter) ToPostGIS(g *pb.Geometry) (*PostGISGeometry, error) {
    wkbData, err := c.ToWKB(g)
    if err != nil {
        return nil, err
    }

    return &PostGISGeometry{
        WKB:  wkbData,
        SRID: c.srid,
    }, nil
}

type PostGISGeometry struct {
    WKB  []byte
    SRID int
}

// Mesh序列化
func (c *GeometryConverter) MarshalMesh(mesh *pb.MeshGeometry) ([]byte, error) {
    // 使用自定义二进制格式或标准格式(如glTF)
    return json.Marshal(mesh)
}

func (c *GeometryConverter) UnmarshalMesh(data []byte) (*pb.MeshGeometry, error) {
    var mesh pb.MeshGeometry
    err := json.Unmarshal(data, &mesh)
    return &mesh, err
}

// BREP序列化
func (c *GeometryConverter) MarshalBREP(brep *pb.BrepGeometry) ([]byte, error) {
    return json.Marshal(brep)
}

func (c *GeometryConverter) UnmarshalBREP(data []byte) (*pb.BrepGeometry, error) {
    var brep pb.BrepGeometry
    err := json.Unmarshal(data, &brep)
    return &brep, err
}
```

## 2.6 几何查询优化

```go
package geometry

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
    db *pgxpool.Pool
}

// OptimizeSpatialQuery 优化空间查询
func (o *QueryOptimizer) OptimizeSpatialQuery(
    ctx context.Context,
    queryType string,
    params map[string]interface{},
) (string, []interface{}, error) {

    // 根据查询类型选择最优策略
    switch queryType {
    case "intersection":
        return o.optimizeIntersectionQuery(ctx, params)
    case "nearest":
        return o.optimizeNearestQuery(ctx, params)
    case "bbox":
        return o.optimizeBBoxQuery(ctx, params)
    default:
        return "", nil, fmt.Errorf("未知的查询类型: %s", queryType)
    }
}

// optimizeIntersectionQuery 优化相交查询
func (o *QueryOptimizer) optimizeIntersectionQuery(
    ctx context.Context,
    params map[string]interface{},
) (string, []interface{}, error) {

    // 使用边界框预过滤 + 精确几何检查的两阶段查询
    query := `
        WITH bbox_filtered AS (
            SELECT id, geom
            FROM geometries
            WHERE tenant_id = $1 
              AND document_id = $2 
              AND is_deleted = FALSE
              AND bbox && ST_GeomFromWKB($3, $4)
        )
        SELECT g.id, g.element_id, g.document_id, g.tenant_id, g.geometry_type,
               ST_AsBinary(g.geom), g.geometry_3d, g.bbox, g.srid, g.metadata, g.properties,
               g.version, g.created_by, g.updated_by, g.created_at, g.updated_at
        FROM bbox_filtered bf
        JOIN geometries g ON g.id = bf.id
        WHERE ST_Intersects(bf.geom, ST_GeomFromWKB($3, $4))
        ORDER BY ST_Area(ST_Intersection(bf.geom, ST_GeomFromWKB($3, $4))) DESC
        LIMIT $5 OFFSET $6
    `

    args := []interface{}{
        params["tenant_id"],
        params["document_id"],
        params["geometry_wkb"],
        params["srid"],
        params["limit"],
        params["offset"],
    }

    return query, args, nil
}

// optimizeNearestQuery 优化最近邻查询
func (o *QueryOptimizer) optimizeNearestQuery(
    ctx context.Context,
    params map[string]interface{},
) (string, []interface{}, error) {

    // 使用KNN索引优化最近邻查询
    query := `
        SELECT id, element_id, document_id, tenant_id, geometry_type,
               ST_AsBinary(geom), geometry_3d, bbox, srid, metadata, properties,
               version, created_by, updated_by, created_at, updated_at,
               ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint($3, $4), $5)::geography) as distance
        FROM geometries
        WHERE tenant_id = $1 
          AND document_id = $2 
          AND is_deleted = FALSE
        ORDER BY geom <-> ST_SetSRID(ST_MakePoint($3, $4), $5)
        LIMIT $6
    `

    point := params["point"].(*pb.PointGeometry)
    args := []interface{}{
        params["tenant_id"],
        params["document_id"],
        point.X,
        point.Y,
        params["srid"],
        params["limit"],
    }

    return query, args, nil
}

// optimizeBBoxQuery 优化边界框查询
func (o *QueryOptimizer) optimizeBBoxQuery(
    ctx context.Context,
    params map[string]interface{},
) (string, []interface{}, error) {

    bbox := params["bbox"].(*pb.BoundingBox)

    // 使用边界框索引直接查询
    query := `
        SELECT id, element_id, document_id, tenant_id, geometry_type,
               ST_AsBinary(geom), geometry_3d, bbox, srid, metadata, properties,
               version, created_by, updated_by, created_at, updated_at
        FROM geometries
        WHERE tenant_id = $1 
          AND document_id = $2 
          AND is_deleted = FALSE
          AND bbox && ST_MakeEnvelope($3, $4, $5, $6, $7)
        LIMIT $8 OFFSET $9
    `

    args := []interface{}{
        params["tenant_id"],
        params["document_id"],
        bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY,
        params["srid"],
        params["limit"],
        params["offset"],
    }

    return query, args, nil
}

// 查询缓存策略
func (o *QueryOptimizer) GetCachedQuery(
    ctx context.Context,
    cacheKey string,
) (*QueryCache, error) {
    // 从Redis获取缓存的查询结果
    // ...
    return nil, nil
}

func (o *QueryOptimizer) CacheQueryResult(
    ctx context.Context,
    cacheKey string,
    result *QueryCache,
    ttl time.Duration,
) error {
    // 缓存查询结果到Redis
    // ...
    return nil
}

type QueryCache struct {
    Query      string
    Args       []interface{}
    ResultHash string
    Timestamp  time.Time
}
```

---

---

# 3. 属性服务详细设计

## 3.1 服务概述

属性服务负责管理建筑设计元素的属性数据，支持属性继承、校验、批量操作等高级功能。

### 核心功能
- 属性定义与管理
- 属性继承机制
- 属性校验规则
- 属性批量操作
- 属性模板管理

## 3.2 gRPC接口定义

```protobuf
syntax = "proto3";

package property.v1;

option go_package = "github.com/archplatform/property-service/api/v1";

import "google/protobuf/struct.proto";
import "google/protobuf/empty.proto";

// 属性服务
service PropertyService {
  // 属性定义管理
  rpc CreatePropertyDefinition(CreatePropertyDefinitionRequest) returns (PropertyDefinition);
  rpc GetPropertyDefinition(GetPropertyDefinitionRequest) returns (PropertyDefinition);
  rpc UpdatePropertyDefinition(UpdatePropertyDefinitionRequest) returns (PropertyDefinition);
  rpc DeletePropertyDefinition(DeletePropertyDefinitionRequest) returns (google.protobuf.Empty);
  rpc ListPropertyDefinitions(ListPropertyDefinitionsRequest) returns (ListPropertyDefinitionsResponse);

  // 属性值管理
  rpc SetProperty(SetPropertyRequest) returns (PropertyValue);
  rpc GetProperty(GetPropertyRequest) returns (PropertyValue);
  rpc GetProperties(GetPropertiesRequest) returns (PropertyCollection);
  rpc DeleteProperty(DeletePropertyRequest) returns (google.protobuf.Empty);
  rpc BatchSetProperties(BatchSetPropertiesRequest) returns (BatchPropertyResponse);
  rpc BatchGetProperties(BatchGetPropertiesRequest) returns (BatchPropertyResponse);
  rpc BatchDeleteProperties(BatchDeletePropertiesRequest) returns (google.protobuf.Empty);

  // 属性继承
  rpc GetInheritedProperties(GetInheritedPropertiesRequest) returns (PropertyCollection);
  rpc SetInheritanceRule(SetInheritanceRuleRequest) returns (InheritanceRule);
  rpc GetInheritanceRules(GetInheritanceRulesRequest) returns (InheritanceRulesResponse);
  rpc BreakInheritance(BreakInheritanceRequest) returns (google.protobuf.Empty);
  rpc RestoreInheritance(RestoreInheritanceRequest) returns (google.protobuf.Empty);

  // 属性校验
  rpc ValidateProperty(ValidatePropertyRequest) returns (ValidationResult);
  rpc ValidateProperties(ValidatePropertiesRequest) returns (ValidationResults);
  rpc GetValidationErrors(GetValidationErrorsRequest) returns (ValidationErrorsResponse);

  // 属性模板
  rpc CreatePropertyTemplate(CreatePropertyTemplateRequest) returns (PropertyTemplate);
  rpc GetPropertyTemplate(GetPropertyTemplateRequest) returns (PropertyTemplate);
  rpc UpdatePropertyTemplate(UpdatePropertyTemplateRequest) returns (PropertyTemplate);
  rpc DeletePropertyTemplate(DeletePropertyTemplateRequest) returns (google.protobuf.Empty);
  rpc ApplyTemplate(ApplyTemplateRequest) returns (ApplyTemplateResponse);
  rpc ListPropertyTemplates(ListPropertyTemplatesRequest) returns (ListPropertyTemplatesResponse);

  // 属性查询
  rpc QueryProperties(QueryPropertiesRequest) returns (PropertyCollection);
  rpc SearchByProperty(SearchByPropertyRequest) returns (ElementCollection);
  rpc GetPropertyStatistics(GetPropertyStatisticsRequest) returns (PropertyStatistics);
}

// ==================== 属性定义消息 ====================

message PropertyDefinition {
  string id = 1;
  string name = 2;
  string display_name = 3;
  string description = 4;
  PropertyType type = 5;
  PropertyCategory category = 6;
  PropertyValueType value_type = 7;
  PropertyValue default_value = 8;
  repeated PropertyConstraint constraints = 9;
  bool is_inheritable = 10;
  bool is_readonly = 11;
  bool is_required = 12;
  bool is_visible = 13;
  int32 display_order = 14;
  string group_name = 15;
  map<string, string> metadata = 16;
  string tenant_id = 17;
  string created_by = 18;
  string updated_by = 19;
  int64 created_at = 20;
  int64 updated_at = 21;
}

message PropertyConstraint {
  ConstraintType type = 1;
  oneof value {
    double min_value = 2;
    double max_value = 3;
    string pattern = 4;
    StringList enum_values = 5;
    double precision = 6;
    UnitConstraint unit = 7;
  }
  string error_message = 8;
}

message StringList {
  repeated string values = 1;
}

message UnitConstraint {
  string unit_type = 1;
  string default_unit = 2;
  repeated string allowed_units = 3;
}

// ==================== 属性值消息 ====================

message PropertyValue {
  string id = 1;
  string element_id = 2;
  string property_definition_id = 3;
  string property_name = 4;
  oneof value {
    string string_value = 5;
    double number_value = 6;
    bool boolean_value = 7;
    int64 integer_value = 8;
    ArrayValue array_value = 9;
    ObjectValue object_value = 10;
    UnitValue unit_value = 11;
    ReferenceValue reference_value = 12;
    FormulaValue formula_value = 13;
  }
  string unit = 14;
  PropertySource source = 15;
  string inherited_from = 16;
  bool is_overridden = 17;
  int64 version = 18;
  string modified_by = 19;
  int64 modified_at = 20;
}

message ArrayValue {
  repeated PropertyValue items = 1;
}

message ObjectValue {
  map<string, PropertyValue> properties = 1;
}

message UnitValue {
  double value = 1;
  string unit = 2;
  double base_value = 3;  // 转换为基准单位的值
}

message ReferenceValue {
  string reference_type = 1;
  string reference_id = 2;
  string display_value = 3;
}

message FormulaValue {
  string formula = 1;
  repeated string dependencies = 2;
  double computed_value = 3;
  string unit = 4;
  bool is_valid = 5;
  string error_message = 6;
}

// ==================== 属性继承消息 ====================

message InheritanceRule {
  string id = 1;
  string source_element_id = 2;
  string target_element_id = 3;
  string property_definition_id = 4;
  InheritanceType inheritance_type = 5;
  TransformType transform_type = 6;
  google.protobuf.Struct transform_params = 7;
  bool is_active = 8;
  int32 priority = 9;
  string created_by = 10;
  int64 created_at = 11;
}

message InheritanceInfo {
  string property_id = 1;
  string property_name = 2;
  string inherited_from = 3;
  string source_element_id = 4;
  InheritanceType type = 5;
  bool is_overridden = 6;
  int64 override_at = 7;
  string override_by = 8;
}

// ==================== 属性模板消息 ====================

message PropertyTemplate {
  string id = 1;
  string name = 2;
  string description = 3;
  string category = 4;
  repeated TemplateProperty properties = 5;
  map<string, string> metadata = 6;
  string tenant_id = 7;
  string created_by = 8;
  int64 created_at = 9;
  int64 updated_at = 10;
}

message TemplateProperty {
  string property_definition_id = 1;
  string property_name = 2;
  PropertyValue default_value = 3;
  bool is_required = 4;
  int32 display_order = 5;
}

// ==================== 校验消息 ====================

message ValidationResult {
  bool is_valid = 1;
  string property_id = 2;
  string property_name = 3;
  repeated ValidationError errors = 4;
  repeated ValidationWarning warnings = 5;
}

message ValidationError {
  ErrorCode code = 1;
  string message = 2;
  ConstraintType constraint_type = 3;
  string expected_value = 4;
  string actual_value = 5;
}

message ValidationWarning {
  string message = 1;
  WarningCode code = 2;
}

// ==================== 请求/响应消息 ====================

message CreatePropertyDefinitionRequest {
  string tenant_id = 1;
  PropertyDefinition definition = 2;
}

message GetPropertyDefinitionRequest {
  string id = 1;
  string tenant_id = 2;
}

message UpdatePropertyDefinitionRequest {
  string id = 1;
  string tenant_id = 2;
  PropertyDefinition definition = 3;
}

message DeletePropertyDefinitionRequest {
  string id = 1;
  string tenant_id = 2;
  bool force = 3;
}

message ListPropertyDefinitionsRequest {
  string tenant_id = 1;
  PropertyCategory category = 2;
  PropertyType type = 3;
  int32 page_size = 4;
  string page_token = 5;
}

message ListPropertyDefinitionsResponse {
  repeated PropertyDefinition definitions = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message SetPropertyRequest {
  string element_id = 1;
  string property_definition_id = 2;
  PropertyValue value = 3;
  string modified_by = 4;
  bool skip_validation = 5;
}

message GetPropertyRequest {
  string element_id = 1;
  string property_definition_id = 2;
  bool include_inherited = 3;
}

message GetPropertiesRequest {
  string element_id = 1;
  bool include_inherited = 2;
  PropertyCategory category = 3;
}

message PropertyCollection {
  string element_id = 1;
  repeated PropertyValue properties = 2;
  repeated InheritanceInfo inheritance_info = 3;
}

message DeletePropertyRequest {
  string element_id = 1;
  string property_definition_id = 2;
}

message BatchSetPropertiesRequest {
  repeated SetPropertyRequest requests = 1;
  bool atomic = 2;
  bool skip_validation = 3;
}

message BatchGetPropertiesRequest {
  repeated string element_ids = 1;
  repeated string property_definition_ids = 2;
  bool include_inherited = 3;
}

message BatchDeletePropertiesRequest {
  repeated DeletePropertyRequest requests = 1;
  bool atomic = 2;
}

message BatchPropertyResponse {
  repeated PropertyValue success_values = 1;
  repeated BatchError errors = 2;
  int32 success_count = 3;
  int32 failed_count = 4;
}

message BatchError {
  string element_id = 1;
  string property_definition_id = 2;
  string error_code = 3;
  string error_message = 4;
}

// ==================== 继承相关请求 ====================

message GetInheritedPropertiesRequest {
  string element_id = 1;
  string source_element_id = 2;
  bool include_all = 3;
}

message SetInheritanceRuleRequest {
  string tenant_id = 1;
  InheritanceRule rule = 2;
}

message GetInheritanceRulesRequest {
  string element_id = 1;
  string property_definition_id = 2;
}

message InheritanceRulesResponse {
  repeated InheritanceRule rules = 1;
}

message BreakInheritanceRequest {
  string element_id = 1;
  string property_definition_id = 2;
  string modified_by = 3;
}

message RestoreInheritanceRequest {
  string element_id = 1;
  string property_definition_id = 2;
  string modified_by = 3;
}

// ==================== 校验相关请求 ====================

message ValidatePropertyRequest {
  string element_id = 1;
  string property_definition_id = 2;
  PropertyValue value = 3;
}

message ValidatePropertiesRequest {
  string element_id = 1;
  ValidationLevel level = 2;
}

message ValidationResults {
  repeated ValidationResult results = 1;
  bool all_valid = 2;
  int32 error_count = 3;
  int32 warning_count = 4;
}

message GetValidationErrorsRequest {
  string element_id = 1;
  string property_definition_id = 2;
}

message ValidationErrorsResponse {
  repeated ValidationError errors = 1;
}

// ==================== 模板相关请求 ====================

message CreatePropertyTemplateRequest {
  string tenant_id = 1;
  PropertyTemplate template = 2;
}

message GetPropertyTemplateRequest {
  string id = 1;
  string tenant_id = 2;
}

message UpdatePropertyTemplateRequest {
  string id = 1;
  string tenant_id = 2;
  PropertyTemplate template = 3;
}

message DeletePropertyTemplateRequest {
  string id = 1;
  string tenant_id = 2;
}

message ApplyTemplateRequest {
  string template_id = 1;
  string element_id = 2;
  string tenant_id = 3;
  bool overwrite_existing = 4;
  string applied_by = 5;
}

message ApplyTemplateResponse {
  int32 applied_count = 1;
  int32 skipped_count = 2;
  repeated string applied_property_ids = 3;
  repeated BatchError errors = 4;
}

message ListPropertyTemplatesRequest {
  string tenant_id = 1;
  string category = 2;
  int32 page_size = 3;
  string page_token = 4;
}

message ListPropertyTemplatesResponse {
  repeated PropertyTemplate templates = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ==================== 查询相关请求 ====================

message QueryPropertiesRequest {
  string tenant_id = 1;
  string document_id = 2;
  PropertyFilter filter = 3;
  int32 page_size = 4;
  string page_token = 5;
}

message PropertyFilter {
  string property_definition_id = 1;
  oneof value_filter {
    StringFilter string_filter = 2;
    NumberFilter number_filter = 3;
    BooleanFilter boolean_filter = 4;
    RangeFilter range_filter = 5;
  }
}

message StringFilter {
  string value = 1;
  StringMatchType match_type = 2;
}

message NumberFilter {
  double value = 1;
  NumberComparison comparison = 2;
}

message BooleanFilter {
  bool value = 1;
}

message RangeFilter {
  double min = 1;
  double max = 2;
  bool include_min = 3;
  bool include_max = 4;
}

message SearchByPropertyRequest {
  string tenant_id = 1;
  string document_id = 2;
  string property_definition_id = 3;
  PropertyValue value = 4;
  PropertyMatchType match_type = 5;
}

message ElementCollection {
  repeated string element_ids = 1;
  int32 total_count = 2;
}

message GetPropertyStatisticsRequest {
  string tenant_id = 1;
  string document_id = 2;
  string property_definition_id = 3;
}

message PropertyStatistics {
  string property_definition_id = 1;
  int32 total_count = 2;
  int32 unique_count = 3;
  oneof statistics {
    NumberStatistics number_stats = 4;
    StringStatistics string_stats = 5;
  }
}

message NumberStatistics {
  double min = 1;
  double max = 2;
  double mean = 3;
  double median = 4;
  double std_dev = 5;
  map<string, int32> histogram = 6;
}

message StringStatistics {
  map<string, int32> value_counts = 1;
  repeated string most_common = 2;
}

// ==================== 枚举定义 ====================

enum PropertyType {
  PROPERTY_TYPE_UNSPECIFIED = 0;
  INSTANCE = 1;      // 实例属性
  TYPE = 2;          // 类型属性
  SHARED = 3;        // 共享属性
  PROJECT = 4;       // 项目属性
  SYSTEM = 5;        // 系统属性
}

enum PropertyCategory {
  PROPERTY_CATEGORY_UNSPECIFIED = 0;
  IDENTITY = 1;      // 标识
  DIMENSIONS = 2;    // 尺寸
  MATERIALS = 3;     // 材料
  STRUCTURAL = 4;    // 结构
  MECHANICAL = 5;    // 机械
  ELECTRICAL = 6;    // 电气
  PLUMBING = 7;      // 给排水
  THERMAL = 8;       // 热工
  ACOUSTIC = 9;      // 声学
  COST = 10;         // 成本
  SCHEDULE = 11;     // 进度
  CUSTOM = 12;       // 自定义
}

enum PropertyValueType {
  PROPERTY_VALUE_TYPE_UNSPECIFIED = 0;
  STRING = 1;
  NUMBER = 2;
  BOOLEAN = 3;
  INTEGER = 4;
  ARRAY = 5;
  OBJECT = 6;
  UNIT = 7;
  REFERENCE = 8;
  FORMULA = 9;
  DATE = 10;
  ENUM = 11;
}

enum ConstraintType {
  CONSTRAINT_TYPE_UNSPECIFIED = 0;
  MIN_VALUE = 1;
  MAX_VALUE = 2;
  RANGE = 3;
  PATTERN = 4;
  ENUM = 5;
  PRECISION = 6;
  UNIT = 7;
  REQUIRED = 8;
  UNIQUE = 9;
}

enum PropertySource {
  PROPERTY_SOURCE_UNSPECIFIED = 0;
  DEFAULT = 1;
  INHERITED = 2;
  COMPUTED = 3;
  MANUAL = 4;
  TEMPLATE = 5;
  IMPORTED = 6;
}

enum InheritanceType {
  INHERITANCE_TYPE_UNSPECIFIED = 0;
  DIRECT = 1;
  CASCADING = 2;
  FORMULA = 3;
  MAPPED = 4;
}

enum TransformType {
  TRANSFORM_TYPE_UNSPECIFIED = 0;
  NONE = 1;
  SCALE = 2;
  OFFSET = 3;
  FORMULA = 4;
  LOOKUP = 5;
}

enum ValidationLevel {
  VALIDATION_LEVEL_UNSPECIFIED = 0;
  BASIC = 1;
  STANDARD = 2;
  STRICT = 3;
}

enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;
  REQUIRED_VALUE_MISSING = 1;
  TYPE_MISMATCH = 2;
  MIN_VALUE_VIOLATION = 3;
  MAX_VALUE_VIOLATION = 4;
  PATTERN_MISMATCH = 5;
  ENUM_VALUE_INVALID = 6;
  PRECISION_EXCEEDED = 7;
  UNIT_INVALID = 8;
  FORMULA_ERROR = 9;
  REFERENCE_INVALID = 10;
}

enum WarningCode {
  WARNING_CODE_UNSPECIFIED = 0;
  VALUE_DEPRECATED = 1;
  UNIT_CONVERTED = 2;
  VALUE_ROUNDED = 3;
}

enum StringMatchType {
  STRING_MATCH_TYPE_UNSPECIFIED = 0;
  EXACT = 1;
  CONTAINS = 2;
  STARTS_WITH = 3;
  ENDS_WITH = 4;
  REGEX = 5;
}

enum NumberComparison {
  NUMBER_COMPARISON_UNSPECIFIED = 0;
  EQUAL = 1;
  LESS_THAN = 2;
  LESS_THAN_OR_EQUAL = 3;
  GREATER_THAN = 4;
  GREATER_THAN_OR_EQUAL = 5;
}

enum PropertyMatchType {
  PROPERTY_MATCH_TYPE_UNSPECIFIED = 0;
  EXACT = 1;
  CONTAINS = 2;
  RANGE = 3;
}
```

## 3.3 数据库表结构设计

```sql
-- ==================== 属性定义表 ====================
CREATE TABLE property_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(128) NOT NULL,
    display_name VARCHAR(256),
    description TEXT,
    property_type VARCHAR(32) NOT NULL,
    category VARCHAR(32) NOT NULL,
    value_type VARCHAR(32) NOT NULL,
    default_value JSONB,
    constraints JSONB DEFAULT '[]',
    is_inheritable BOOLEAN DEFAULT TRUE,
    is_readonly BOOLEAN DEFAULT FALSE,
    is_required BOOLEAN DEFAULT FALSE,
    is_visible BOOLEAN DEFAULT TRUE,
    display_order INTEGER DEFAULT 0,
    group_name VARCHAR(128),
    metadata JSONB DEFAULT '{}',
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE,

    CONSTRAINT chk_property_type CHECK (property_type IN ('INSTANCE', 'TYPE', 'SHARED', 'PROJECT', 'SYSTEM')),
    CONSTRAINT chk_category CHECK (category IN ('IDENTITY', 'DIMENSIONS', 'MATERIALS', 'STRUCTURAL', 'MECHANICAL', 'ELECTRICAL', 'PLUMBING', 'THERMAL', 'ACOUSTIC', 'COST', 'SCHEDULE', 'CUSTOM')),
    CONSTRAINT chk_value_type CHECK (value_type IN ('STRING', 'NUMBER', 'BOOLEAN', 'INTEGER', 'ARRAY', 'OBJECT', 'UNIT', 'REFERENCE', 'FORMULA', 'DATE', 'ENUM')),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_prop_defs_tenant ON property_definitions(tenant_id);
CREATE INDEX idx_prop_defs_category ON property_definitions(category);
CREATE INDEX idx_prop_defs_type ON property_definitions(property_type);
CREATE INDEX idx_prop_defs_deleted ON property_definitions(is_deleted) WHERE is_deleted = FALSE;

-- ==================== 属性值表 ====================
CREATE TABLE property_values (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    element_id UUID NOT NULL,
    property_definition_id UUID NOT NULL REFERENCES property_definitions(id),
    string_value TEXT,
    number_value DOUBLE PRECISION,
    boolean_value BOOLEAN,
    integer_value BIGINT,
    array_value JSONB,
    object_value JSONB,
    unit_value JSONB,
    reference_value JSONB,
    formula_value JSONB,
    unit VARCHAR(32),
    property_source VARCHAR(32) DEFAULT 'MANUAL',
    inherited_from UUID,
    is_overridden BOOLEAN DEFAULT FALSE,
    override_at TIMESTAMP WITH TIME ZONE,
    override_by UUID,
    version BIGINT DEFAULT 1,
    modified_by UUID,
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE,

    CONSTRAINT chk_source CHECK (property_source IN ('DEFAULT', 'INHERITED', 'COMPUTED', 'MANUAL', 'TEMPLATE', 'IMPORTED')),
    UNIQUE(element_id, property_definition_id)
);

CREATE INDEX idx_prop_values_element ON property_values(element_id);
CREATE INDEX idx_prop_values_definition ON property_values(property_definition_id);
CREATE INDEX idx_prop_values_source ON property_values(property_source);
CREATE INDEX idx_prop_values_inherited ON property_values(inherited_from);
CREATE INDEX idx_prop_values_deleted ON property_values(is_deleted) WHERE is_deleted = FALSE;

-- 部分索引: 按值类型查询
CREATE INDEX idx_prop_values_number ON property_values(number_value) WHERE number_value IS NOT NULL;
CREATE INDEX idx_prop_values_string ON property_values(string_value) WHERE string_value IS NOT NULL;

-- ==================== 属性继承规则表 ====================
CREATE TABLE inheritance_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_element_id UUID NOT NULL,
    target_element_id UUID NOT NULL,
    property_definition_id UUID NOT NULL REFERENCES property_definitions(id),
    inheritance_type VARCHAR(32) NOT NULL DEFAULT 'DIRECT',
    transform_type VARCHAR(32) DEFAULT 'NONE',
    transform_params JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_inheritance_type CHECK (inheritance_type IN ('DIRECT', 'CASCADING', 'FORMULA', 'MAPPED')),
    CONSTRAINT chk_transform_type CHECK (transform_type IN ('NONE', 'SCALE', 'OFFSET', 'FORMULA', 'LOOKUP'))
);

CREATE INDEX idx_inheritance_source ON inheritance_rules(source_element_id);
CREATE INDEX idx_inheritance_target ON inheritance_rules(target_element_id);
CREATE INDEX idx_inheritance_property ON inheritance_rules(property_definition_id);
CREATE INDEX idx_inheritance_active ON inheritance_rules(is_active) WHERE is_active = TRUE;

-- ==================== 属性模板表 ====================
CREATE TABLE property_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(256) NOT NULL,
    description TEXT,
    category VARCHAR(128),
    properties JSONB NOT NULL DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_deleted BOOLEAN DEFAULT FALSE,

    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_templates_tenant ON property_templates(tenant_id);
CREATE INDEX idx_templates_category ON property_templates(category);

-- ==================== 属性版本历史表 ====================
CREATE TABLE property_value_history (
    id BIGSERIAL PRIMARY KEY,
    property_value_id UUID NOT NULL REFERENCES property_values(id),
    version BIGINT NOT NULL,
    old_value JSONB,
    new_value JSONB,
    change_type VARCHAR(32) NOT NULL,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_change_type CHECK (change_type IN ('CREATE', 'UPDATE', 'DELETE', 'INHERIT', 'OVERRIDE'))
);

CREATE INDEX idx_prop_history_value ON property_value_history(property_value_id);
CREATE INDEX idx_prop_history_version ON property_value_history(property_value_id, version);

-- ==================== 属性校验错误表 ====================
CREATE TABLE property_validation_errors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    element_id UUID NOT NULL,
    property_definition_id UUID NOT NULL,
    error_code VARCHAR(64) NOT NULL,
    error_message TEXT,
    constraint_type VARCHAR(32),
    expected_value TEXT,
    actual_value TEXT,
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_validation_errors_element ON property_validation_errors(element_id);
CREATE INDEX idx_validation_errors_unresolved ON property_validation_errors(is_resolved) WHERE is_resolved = FALSE;

-- ==================== 触发器函数 ====================

-- 属性值变更历史记录
CREATE OR REPLACE FUNCTION log_property_value_change()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO property_value_history (
            property_value_id, version, new_value, change_type, changed_by
        ) VALUES (
            NEW.id, NEW.version, row_to_json(NEW), 'CREATE', NEW.modified_by
        );
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.string_value IS DISTINCT FROM NEW.string_value OR
           OLD.number_value IS DISTINCT FROM NEW.number_value OR
           OLD.boolean_value IS DISTINCT FROM NEW.boolean_value OR
           OLD.array_value IS DISTINCT FROM NEW.array_value OR
           OLD.object_value IS DISTINCT FROM NEW.object_value THEN

            INSERT INTO property_value_history (
                property_value_id, version, old_value, new_value, change_type, changed_by
            ) VALUES (
                NEW.id, NEW.version, row_to_json(OLD), row_to_json(NEW), 
                CASE 
                    WHEN NEW.is_overridden AND NOT OLD.is_overridden THEN 'OVERRIDE'
                    WHEN NEW.property_source = 'INHERITED' AND OLD.property_source != 'INHERITED' THEN 'INHERIT'
                    ELSE 'UPDATE'
                END,
                NEW.modified_by
            );

            NEW.version := OLD.version + 1;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_property_change
    AFTER INSERT OR UPDATE ON property_values
    FOR EACH ROW
    EXECUTE FUNCTION log_property_value_change();
```

## 3.4 属性继承机制实现

```go
package property

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

// InheritanceManager 属性继承管理器
type InheritanceManager struct {
    db        *pgxpool.Pool
    validator *PropertyValidator
}

// GetInheritedProperties 获取继承的属性
func (m *InheritanceManager) GetInheritedProperties(
    ctx context.Context,
    elementID string,
) ([]*pb.PropertyValue, error) {
    // 查询继承规则
    query := `
        SELECT ir.source_element_id, ir.property_definition_id, 
               ir.inheritance_type, ir.transform_type, ir.transform_params,
               pv.string_value, pv.number_value, pv.boolean_value, 
               pv.integer_value, pv.array_value, pv.object_value,
               pv.unit_value, pv.reference_value, pv.formula_value, pv.unit
        FROM inheritance_rules ir
        JOIN property_values pv ON pv.element_id = ir.source_element_id 
            AND pv.property_definition_id = ir.property_definition_id
        WHERE ir.target_element_id = $1 
          AND ir.is_active = TRUE
          AND pv.is_deleted = FALSE
        ORDER BY ir.priority DESC
    `

    rows, err := m.db.Query(ctx, query, elementID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var inheritedProps []*pb.PropertyValue

    for rows.Next() {
        var sourceID, propDefID, inhType, transType string
        var transParams []byte
        var propValue PropertyValueRow

        err := rows.Scan(
            &sourceID, &propDefID, &inhType, &transType, &transParams,
            &propValue.StringValue, &propValue.NumberValue, &propValue.BooleanValue,
            &propValue.IntegerValue, &propValue.ArrayValue, &propValue.ObjectValue,
            &propValue.UnitValue, &propValue.ReferenceValue, &propValue.FormulaValue,
            &propValue.Unit,
        )
        if err != nil {
            continue
        }

        // 应用变换
        transformedValue, err := m.applyTransform(
            &propValue, 
            pb.TransformType(pb.TransformType_value[transType]),
            transParams,
        )
        if err != nil {
            continue
        }

        // 构建属性值
        prop := &pb.PropertyValue{
            PropertyDefinitionId: propDefID,
            Source:               pb.PropertySource_INHERITED,
            InheritedFrom:        sourceID,
        }

        // 设置值
        m.setPropertyValue(prop, transformedValue)

        inheritedProps = append(inheritedProps, prop)
    }

    return inheritedProps, nil
}

// applyTransform 应用继承变换
func (m *InheritanceManager) applyTransform(
    value *PropertyValueRow,
    transformType pb.TransformType,
    params []byte,
) (*PropertyValueRow, error) {

    switch transformType {
    case pb.TransformType_NONE:
        return value, nil

    case pb.TransformType_SCALE:
        var scaleParams struct {
            Factor float64 `json:"factor"`
        }
        if err := json.Unmarshal(params, &scaleParams); err != nil {
            return nil, err
        }

        if value.NumberValue != nil {
            scaled := *value.NumberValue * scaleParams.Factor
            value.NumberValue = &scaled
        }
        return value, nil

    case pb.TransformType_OFFSET:
        var offsetParams struct {
            Value float64 `json:"value"`
        }
        if err := json.Unmarshal(params, &offsetParams); err != nil {
            return nil, err
        }

        if value.NumberValue != nil {
            offset := *value.NumberValue + offsetParams.Value
            value.NumberValue = &offset
        }
        return value, nil

    case pb.TransformType_FORMULA:
        var formulaParams struct {
            Formula string `json:"formula"`
        }
        if err := json.Unmarshal(params, &formulaParams); err != nil {
            return nil, err
        }

        // 执行公式计算
        result, err := m.evaluateFormula(formulaParams.Formula, value)
        if err != nil {
            return nil, err
        }
        value.NumberValue = &result
        return value, nil

    default:
        return value, nil
    }
}

// evaluateFormula 计算公式
func (m *InheritanceManager) evaluateFormula(
    formula string, 
    value *PropertyValueRow,
) (float64, error) {
    // 使用公式引擎计算
    // 这里简化实现，实际应使用专门的公式引擎

    if value.NumberValue == nil {
        return 0, fmt.Errorf("数值为空，无法计算公式")
    }

    // 简单示例: 将公式中的{x}替换为实际值
    // 实际应使用更复杂的表达式解析器
    return *value.NumberValue, nil
}

// CascadeInheritance 级联继承
func (m *InheritanceManager) CascadeInheritance(
    ctx context.Context,
    sourceElementID string,
    propertyDefID string,
    newValue *pb.PropertyValue,
) error {
    // 获取所有子元素
    children, err := m.getChildElements(ctx, sourceElementID)
    if err != nil {
        return err
    }

    // 为每个子元素应用继承
    for _, childID := range children {
        // 检查是否有覆盖
        isOverridden, err := m.isPropertyOverridden(ctx, childID, propertyDefID)
        if err != nil {
            continue
        }

        if isOverridden {
            // 跳过已覆盖的属性
            continue
        }

        // 应用继承值
        _, err = m.db.Exec(ctx, `
            INSERT INTO property_values (
                element_id, property_definition_id, property_source,
                inherited_from, modified_at
            ) VALUES ($1, $2, 'INHERITED', $3, NOW())
            ON CONFLICT (element_id, property_definition_id) 
            DO UPDATE SET 
                property_source = 'INHERITED',
                inherited_from = $3,
                modified_at = NOW()
        `, childID, propertyDefID, sourceElementID)

        if err != nil {
            return err
        }

        // 递归级联
        if err := m.CascadeInheritance(ctx, childID, propertyDefID, newValue); err != nil {
            return err
        }
    }

    return nil
}

// BreakInheritance 断开继承
func (m *InheritanceManager) BreakInheritance(
    ctx context.Context,
    elementID string,
    propertyDefID string,
    modifiedBy string,
) error {
    _, err := m.db.Exec(ctx, `
        UPDATE property_values
        SET is_overridden = TRUE,
            override_at = NOW(),
            override_by = $3,
            property_source = 'MANUAL'
        WHERE element_id = $1 AND property_definition_id = $2
    `, elementID, propertyDefID, modifiedBy)

    return err
}

// RestoreInheritance 恢复继承
func (m *InheritanceManager) RestoreInheritance(
    ctx context.Context,
    elementID string,
    propertyDefID string,
) error {
    // 删除当前值，让继承重新生效
    _, err := m.db.Exec(ctx, `
        DELETE FROM property_values
        WHERE element_id = $1 AND property_definition_id = $2
          AND is_overridden = TRUE
    `, elementID, propertyDefID)

    return err
}

// 辅助方法
func (m *InheritanceManager) getChildElements(
    ctx context.Context,
    parentID string,
) ([]string, error) {
    // 从元素服务获取子元素列表
    // 这里简化实现
    return []string{}, nil
}

func (m *InheritanceManager) isPropertyOverridden(
    ctx context.Context,
    elementID string,
    propertyDefID string,
) (bool, error) {
    var isOverridden bool
    err := m.db.QueryRow(ctx, `
        SELECT is_overridden 
        FROM property_values 
        WHERE element_id = $1 AND property_definition_id = $2
    `, elementID, propertyDefID).Scan(&isOverridden)

    if err != nil {
        return false, err
    }

    return isOverridden, nil
}
```

## 3.5 属性校验规则实现

```go
package property

import (
    "fmt"
    "regexp"
    "strconv"
)

// PropertyValidator 属性校验器
type PropertyValidator struct {
    unitConverter *UnitConverter
}

// Validate 校验属性值
func (v *PropertyValidator) Validate(
    value *pb.PropertyValue,
    definition *pb.PropertyDefinition,
) *pb.ValidationResult {
    result := &pb.ValidationResult{
        IsValid:      true,
        PropertyId:   value.Id,
        PropertyName: definition.Name,
    }

    // 必填校验
    if definition.IsRequired {
        if !v.validateRequired(value) {
            result.IsValid = false
            result.Errors = append(result.Errors, &pb.ValidationError{
                Code:        pb.ErrorCode_REQUIRED_VALUE_MISSING,
                Message:     fmt.Sprintf("属性 '%s' 为必填项", definition.DisplayName),
                ConstraintType: pb.ConstraintType_REQUIRED,
            })
        }
    }

    // 类型校验
    if !v.validateType(value, definition.ValueType) {
        result.IsValid = false
        result.Errors = append(result.Errors, &pb.ValidationError{
            Code:        pb.ErrorCode_TYPE_MISMATCH,
            Message:     fmt.Sprintf("属性 '%s' 类型不匹配，期望类型: %s", 
                definition.DisplayName, definition.ValueType.String()),
            ConstraintType: pb.ConstraintType_ENUM,
            ExpectedValue: definition.ValueType.String(),
            ActualValue:   v.getValueTypeString(value),
        })
        return result
    }

    // 约束校验
    for _, constraint := range definition.Constraints {
        if err := v.validateConstraint(value, constraint); err != nil {
            result.IsValid = false
            result.Errors = append(result.Errors, err)
        }
    }

    return result
}

// validateRequired 必填校验
func (v *PropertyValidator) validateRequired(value *pb.PropertyValue) bool {
    switch v := value.Value.(type) {
    case *pb.PropertyValue_StringValue:
        return v.StringValue != ""
    case *pb.PropertyValue_NumberValue:
        return true
    case *pb.PropertyValue_BooleanValue:
        return true
    case *pb.PropertyValue_IntegerValue:
        return true
    case *pb.PropertyValue_ArrayValue:
        return v.ArrayValue != nil && len(v.ArrayValue.Items) > 0
    case *pb.PropertyValue_ObjectValue:
        return v.ObjectValue != nil && len(v.ObjectValue.Properties) > 0
    case *pb.PropertyValue_UnitValue:
        return v.UnitValue != nil
    case *pb.PropertyValue_ReferenceValue:
        return v.ReferenceValue != nil && v.ReferenceValue.ReferenceId != ""
    default:
        return false
    }
}

// validateType 类型校验
func (v *PropertyValidator) validateType(
    value *pb.PropertyValue, 
    expectedType pb.PropertyValueType,
) bool {
    switch expectedType {
    case pb.PropertyValueType_STRING:
        _, ok := value.Value.(*pb.PropertyValue_StringValue)
        return ok
    case pb.PropertyValueType_NUMBER:
        _, ok := value.Value.(*pb.PropertyValue_NumberValue)
        return ok
    case pb.PropertyValueType_BOOLEAN:
        _, ok := value.Value.(*pb.PropertyValue_BooleanValue)
        return ok
    case pb.PropertyValueType_INTEGER:
        _, ok := value.Value.(*pb.PropertyValue_IntegerValue)
        return ok
    case pb.PropertyValueType_ARRAY:
        _, ok := value.Value.(*pb.PropertyValue_ArrayValue)
        return ok
    case pb.PropertyValueType_OBJECT:
        _, ok := value.Value.(*pb.PropertyValue_ObjectValue)
        return ok
    case pb.PropertyValueType_UNIT:
        _, ok := value.Value.(*pb.PropertyValue_UnitValue)
        return ok
    case pb.PropertyValueType_REFERENCE:
        _, ok := value.Value.(*pb.PropertyValue_ReferenceValue)
        return ok
    case pb.PropertyValueType_FORMULA:
        _, ok := value.Value.(*pb.PropertyValue_FormulaValue)
        return ok
    default:
        return false
    }
}

// validateConstraint 约束校验
func (v *PropertyValidator) validateConstraint(
    value *pb.PropertyValue,
    constraint *pb.PropertyConstraint,
) *pb.ValidationError {

    switch constraint.Type {
    case pb.ConstraintType_MIN_VALUE:
        return v.validateMinValue(value, constraint.GetMinValue())

    case pb.ConstraintType_MAX_VALUE:
        return v.validateMaxValue(value, constraint.GetMaxValue())

    case pb.ConstraintType_PATTERN:
        return v.validatePattern(value, constraint.GetPattern())

    case pb.ConstraintType_ENUM:
        return v.validateEnum(value, constraint.GetEnumValues())

    case pb.ConstraintType_PRECISION:
        return v.validatePrecision(value, constraint.GetPrecision())

    case pb.ConstraintType_UNIT:
        return v.validateUnit(value, constraint.GetUnit())

    default:
        return nil
    }
}

// validateMinValue 最小值校验
func (v *PropertyValidator) validateMinValue(
    value *pb.PropertyValue,
    min float64,
) *pb.ValidationError {
    var actual float64

    switch v := value.Value.(type) {
    case *pb.PropertyValue_NumberValue:
        actual = v.NumberValue
    case *pb.PropertyValue_IntegerValue:
        actual = float64(v.IntegerValue)
    case *pb.PropertyValue_UnitValue:
        actual = v.UnitValue.Value
    default:
        return nil
    }

    if actual < min {
        return &pb.ValidationError{
            Code:           pb.ErrorCode_MIN_VALUE_VIOLATION,
            Message:        fmt.Sprintf("值 %.2f 小于最小值 %.2f", actual, min),
            ConstraintType: pb.ConstraintType_MIN_VALUE,
            ExpectedValue:  fmt.Sprintf(">= %.2f", min),
            ActualValue:    fmt.Sprintf("%.2f", actual),
        }
    }

    return nil
}

// validateMaxValue 最大值校验
func (v *PropertyValidator) validateMaxValue(
    value *pb.PropertyValue,
    max float64,
) *pb.ValidationError {
    var actual float64

    switch v := value.Value.(type) {
    case *pb.PropertyValue_NumberValue:
        actual = v.NumberValue
    case *pb.PropertyValue_IntegerValue:
        actual = float64(v.IntegerValue)
    case *pb.PropertyValue_UnitValue:
        actual = v.UnitValue.Value
    default:
        return nil
    }

    if actual > max {
        return &pb.ValidationError{
            Code:           pb.ErrorCode_MAX_VALUE_VIOLATION,
            Message:        fmt.Sprintf("值 %.2f 大于最大值 %.2f", actual, max),
            ConstraintType: pb.ConstraintType_MAX_VALUE,
            ExpectedValue:  fmt.Sprintf("<= %.2f", max),
            ActualValue:    fmt.Sprintf("%.2f", actual),
        }
    }

    return nil
}

// validatePattern 正则校验
func (v *PropertyValidator) validatePattern(
    value *pb.PropertyValue,
    pattern string,
) *pb.ValidationError {
    strValue, ok := value.Value.(*pb.PropertyValue_StringValue)
    if !ok {
        return nil
    }

    matched, err := regexp.MatchString(pattern, strValue.StringValue)
    if err != nil || !matched {
        return &pb.ValidationError{
            Code:           pb.ErrorCode_PATTERN_MISMATCH,
            Message:        fmt.Sprintf("值 '%s' 不匹配模式 '%s'", strValue.StringValue, pattern),
            ConstraintType: pb.ConstraintType_PATTERN,
            ExpectedValue:  pattern,
            ActualValue:    strValue.StringValue,
        }
    }

    return nil
}

// validateEnum 枚举校验
func (v *PropertyValidator) validateEnum(
    value *pb.PropertyValue,
    enumValues *pb.StringList,
) *pb.ValidationError {
    strValue, ok := value.Value.(*pb.PropertyValue_StringValue)
    if !ok {
        return nil
    }

    validValues := make(map[string]bool)
    for _, ev := range enumValues.Values {
        validValues[ev] = true
    }

    if !validValues[strValue.StringValue] {
        return &pb.ValidationError{
            Code:           pb.ErrorCode_ENUM_VALUE_INVALID,
            Message:        fmt.Sprintf("值 '%s' 不在允许的枚举值中", strValue.StringValue),
            ConstraintType: pb.ConstraintType_ENUM,
            ExpectedValue:  fmt.Sprintf("%v", enumValues.Values),
            ActualValue:    strValue.StringValue,
        }
    }

    return nil
}

// validatePrecision 精度校验
func (v *PropertyValidator) validatePrecision(
    value *pb.PropertyValue,
    precision float64,
) *pb.ValidationError {
    // 实现精度校验
    return nil
}

// validateUnit 单位校验
func (v *PropertyValidator) validateUnit(
    value *pb.PropertyValue,
    unitConstraint *pb.UnitConstraint,
) *pb.ValidationError {
    unitValue, ok := value.Value.(*pb.PropertyValue_UnitValue)
    if !ok {
        return nil
    }

    // 检查单位是否在允许列表中
    allowedUnits := make(map[string]bool)
    for _, u := range unitConstraint.AllowedUnits {
        allowedUnits[u] = true
    }

    if !allowedUnits[unitValue.UnitValue.Unit] {
        return &pb.ValidationError{
            Code:           pb.ErrorCode_UNIT_INVALID,
            Message:        fmt.Sprintf("单位 '%s' 不被允许", unitValue.UnitValue.Unit),
            ConstraintType: pb.ConstraintType_UNIT,
            ExpectedValue:  fmt.Sprintf("%v", unitConstraint.AllowedUnits),
            ActualValue:    unitValue.UnitValue.Unit,
        }
    }

    return nil
}

// getValueTypeString 获取值的类型字符串
func (v *PropertyValidator) getValueTypeString(value *pb.PropertyValue) string {
    switch value.Value.(type) {
    case *pb.PropertyValue_StringValue:
        return "STRING"
    case *pb.PropertyValue_NumberValue:
        return "NUMBER"
    case *pb.PropertyValue_BooleanValue:
        return "BOOLEAN"
    case *pb.PropertyValue_IntegerValue:
        return "INTEGER"
    case *pb.PropertyValue_ArrayValue:
        return "ARRAY"
    case *pb.PropertyValue_ObjectValue:
        return "OBJECT"
    case *pb.PropertyValue_UnitValue:
        return "UNIT"
    case *pb.PropertyValue_ReferenceValue:
        return "REFERENCE"
    case *pb.PropertyValue_FormulaValue:
        return "FORMULA"
    default:
        return "UNKNOWN"
    }
}
```

## 3.6 属性批量操作实现

```go
package property

import (
    "context"
    "sync"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// BatchProcessor 批量处理器
type BatchProcessor struct {
    db        *pgxpool.Pool
    validator *PropertyValidator
    cache     *PropertyCache
}

// BatchSetProperties 批量设置属性
func (p *BatchProcessor) BatchSetProperties(
    ctx context.Context,
    requests []*pb.SetPropertyRequest,
    atomic bool,
) *pb.BatchPropertyResponse {
    response := &pb.BatchPropertyResponse{
        SuccessValues: make([]*pb.PropertyValue, 0),
        Errors:        make([]*pb.BatchError, 0),
    }

    if atomic {
        // 原子操作: 使用事务
        tx, err := p.db.Begin(ctx)
        if err != nil {
            return p.buildAllFailedResponse(requests, err)
        }
        defer tx.Rollback(ctx)

        for _, req := range requests {
            value, err := p.setPropertyInTx(ctx, tx, req)
            if err != nil {
                tx.Rollback(ctx)
                return p.buildAllFailedResponse(requests, err)
            }
            response.SuccessValues = append(response.SuccessValues, value)
        }

        if err := tx.Commit(ctx); err != nil {
            return p.buildAllFailedResponse(requests, err)
        }

        response.SuccessCount = int32(len(response.SuccessValues))
        return response
    }

    // 非原子操作: 并行处理
    var wg sync.WaitGroup
    resultCh := make(chan *batchResult, len(requests))

    // 限制并发数
    sem := make(chan struct{}, 10)

    for _, req := range requests {
        wg.Add(1)
        go func(r *pb.SetPropertyRequest) {
            defer wg.Done()

            sem <- struct{}{}
            defer func() { <-sem }()

            value, err := p.SetProperty(ctx, r)
            resultCh <- &batchResult{
                request: r,
                value:   value,
                err:     err,
            }
        }(req)
    }

    // 等待所有任务完成
    go func() {
        wg.Wait()
        close(resultCh)
    }()

    // 收集结果
    for result := range resultCh {
        if result.err != nil {
            response.Errors = append(response.Errors, &pb.BatchError{
                ElementId:            result.request.ElementId,
                PropertyDefinitionId: result.request.PropertyDefinitionId,
                ErrorCode:            "SET_PROPERTY_FAILED",
                ErrorMessage:         result.err.Error(),
            })
            response.FailedCount++
        } else {
            response.SuccessValues = append(response.SuccessValues, result.value)
            response.SuccessCount++
        }
    }

    return response
}

// BatchGetProperties 批量获取属性
func (p *BatchProcessor) BatchGetProperties(
    ctx context.Context,
    elementIDs []string,
    propertyDefIDs []string,
    includeInherited bool,
) *pb.BatchPropertyResponse {
    response := &pb.BatchPropertyResponse{
        SuccessValues: make([]*pb.PropertyValue, 0),
    }

    // 构建查询
    query := `
        SELECT id, element_id, property_definition_id,
               string_value, number_value, boolean_value, integer_value,
               array_value, object_value, unit_value, reference_value, formula_value,
               unit, property_source, inherited_from, is_overridden, version
        FROM property_values
        WHERE element_id = ANY($1)
    `
    args := []interface{}{elementIDs}

    if len(propertyDefIDs) > 0 {
        query += ` AND property_definition_id = ANY($2)`
        args = append(args, propertyDefIDs)
    }

    query += ` AND is_deleted = FALSE`

    rows, err := p.db.Query(ctx, query, args...)
    if err != nil {
        return &pb.BatchPropertyResponse{
            Errors: []*pb.BatchError{{
                ErrorCode:    "QUERY_FAILED",
                ErrorMessage: err.Error(),
            }},
            FailedCount: int32(len(elementIDs)),
        }
    }
    defer rows.Close()

    for rows.Next() {
        value := &pb.PropertyValue{}
        // 扫描行数据到value
        // ...
        response.SuccessValues = append(response.SuccessValues, value)
    }

    // 如果需要继承属性
    if includeInherited {
        inheritedProps := p.getInheritedPropertiesBatch(ctx, elementIDs)
        response.SuccessValues = append(response.SuccessValues, inheritedProps...)
    }

    response.SuccessCount = int32(len(response.SuccessValues))
    return response
}

// setPropertyInTx 在事务中设置属性
func (p *BatchProcessor) setPropertyInTx(
    ctx context.Context,
    tx pgx.Tx,
    req *pb.SetPropertyRequest,
) (*pb.PropertyValue, error) {
    // 校验属性值
    if !req.SkipValidation {
        definition, err := p.getPropertyDefinition(ctx, req.PropertyDefinitionId)
        if err != nil {
            return nil, err
        }

        result := p.validator.Validate(req.Value, definition)
        if !result.IsValid {
            return nil, fmt.Errorf("属性校验失败: %v", result.Errors)
        }
    }

    // 插入或更新属性值
    var id string
    err := tx.QueryRow(ctx, `
        INSERT INTO property_values (
            element_id, property_definition_id, 
            string_value, number_value, boolean_value, integer_value,
            array_value, object_value, unit_value, reference_value, formula_value,
            unit, property_source, modified_by, modified_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'MANUAL', $13, NOW())
        ON CONFLICT (element_id, property_definition_id) 
        DO UPDATE SET 
            string_value = EXCLUDED.string_value,
            number_value = EXCLUDED.number_value,
            boolean_value = EXCLUDED.boolean_value,
            integer_value = EXCLUDED.integer_value,
            array_value = EXCLUDED.array_value,
            object_value = EXCLUDED.object_value,
            unit_value = EXCLUDED.unit_value,
            reference_value = EXCLUDED.reference_value,
            formula_value = EXCLUDED.formula_value,
            unit = EXCLUDED.unit,
            property_source = 'MANUAL',
            modified_by = EXCLUDED.modified_by,
            modified_at = NOW()
        RETURNING id
    `, 
        req.ElementId, req.PropertyDefinitionId,
        // ... 属性值字段
    ).Scan(&id)

    if err != nil {
        return nil, err
    }

    req.Value.Id = id
    return req.Value, nil
}

// buildAllFailedResponse 构建全部失败的响应
func (p *BatchProcessor) buildAllFailedResponse(
    requests []*pb.SetPropertyRequest,
    err error,
) *pb.BatchPropertyResponse {
    errors := make([]*pb.BatchError, len(requests))
    for i, req := range requests {
        errors[i] = &pb.BatchError{
            ElementId:            req.ElementId,
            PropertyDefinitionId: req.PropertyDefinitionId,
            ErrorCode:            "BATCH_FAILED",
            ErrorMessage:         err.Error(),
        }
    }
    return &pb.BatchPropertyResponse{
        Errors:      errors,
        FailedCount: int32(len(requests)),
    }
}

// getInheritedPropertiesBatch 批量获取继承属性
func (p *BatchProcessor) getInheritedPropertiesBatch(
    ctx context.Context,
    elementIDs []string,
) []*pb.PropertyValue {
    // 实现批量继承属性获取
    return []*pb.PropertyValue{}
}

type batchResult struct {
    request *pb.SetPropertyRequest
    value   *pb.PropertyValue
    err     error
}
```

---

---

# 4. 脚本服务详细设计

## 4.1 服务概述

脚本服务提供安全的脚本执行环境，支持Python、JavaScript等脚本语言，用于自动化设计任务和批量操作。

### 核心功能
- 脚本执行环境管理
- 安全沙箱隔离
- 脚本版本控制
- 脚本权限管理
- 执行结果缓存

## 4.2 gRPC接口定义

```protobuf
syntax = "proto3";

package script.v1;

option go_package = "github.com/archplatform/script-service/api/v1";

import "google/protobuf/struct.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

// 脚本服务
service ScriptService {
  // 脚本管理
  rpc CreateScript(CreateScriptRequest) returns (Script);
  rpc GetScript(GetScriptRequest) returns (Script);
  rpc UpdateScript(UpdateScriptRequest) returns (Script);
  rpc DeleteScript(DeleteScriptRequest) returns (google.protobuf.Empty);
  rpc ListScripts(ListScriptsRequest) returns (ListScriptsResponse);

  // 脚本版本管理
  rpc CreateVersion(CreateVersionRequest) returns (ScriptVersion);
  rpc GetVersion(GetVersionRequest) returns (ScriptVersion);
  rpc ListVersions(ListVersionsRequest) returns (ListVersionsResponse);
  rpc SetDefaultVersion(SetDefaultVersionRequest) returns (Script);
  rpc CompareVersions(CompareVersionsRequest) returns (VersionComparison);

  // 脚本执行
  rpc ExecuteScript(ExecuteScriptRequest) returns (ExecutionResult);
  rpc ExecuteScriptStream(ExecuteScriptRequest) returns (stream ExecutionStreamEvent);
  rpc ExecuteAsync(ExecuteAsyncRequest) returns (AsyncExecutionHandle);
  rpc GetExecutionStatus(GetExecutionStatusRequest) returns (ExecutionStatus);
  rpc CancelExecution(CancelExecutionRequest) returns (google.protobuf.Empty);
  rpc GetExecutionResult(GetExecutionResultRequest) returns (ExecutionResult);
  rpc ListExecutions(ListExecutionsRequest) returns (ListExecutionsResponse);

  // 调试
  rpc DebugScript(DebugScriptRequest) returns (stream DebugEvent);
  rpc SetBreakpoint(SetBreakpointRequest) returns (google.protobuf.Empty);
  rpc RemoveBreakpoint(RemoveBreakpointRequest) returns (google.protobuf.Empty);
  rpc ContinueExecution(ContinueExecutionRequest) returns (google.protobuf.Empty);
  rpc StepExecution(StepExecutionRequest) returns (google.protobuf.Empty);

  // 脚本模板
  rpc GetScriptTemplates(GetScriptTemplatesRequest) returns (ScriptTemplatesResponse);
  rpc CreateFromTemplate(CreateFromTemplateRequest) returns (Script);

  // 脚本库管理
  rpc InstallLibrary(InstallLibraryRequest) returns (Library);
  rpc UninstallLibrary(UninstallLibraryRequest) returns (google.protobuf.Empty);
  rpc ListLibraries(ListLibrariesRequest) returns (ListLibrariesResponse);
}

// ==================== 脚本消息 ====================

message Script {
  string id = 1;
  string name = 2;
  string description = 3;
  ScriptLanguage language = 4;
  string current_version_id = 5;
  ScriptVersion current_version = 6;
  ScriptType type = 7;
  ScriptScope scope = 8;
  repeated string tags = 9;
  map<string, string> metadata = 10;
  string tenant_id = 11;
  string created_by = 12;
  string updated_by = 13;
  int64 created_at = 14;
  int64 updated_at = 15;
  bool is_active = 16;
  ExecutionPermissions permissions = 17;
}

message ScriptVersion {
  string id = 1;
  string script_id = 2;
  string version = 3;
  string code = 4;
  google.protobuf.Struct parameters = 5;
  repeated ScriptInput inputs = 6;
  repeated ScriptOutput outputs = 7;
  string changelog = 8;
  bool is_default = 9;
  string created_by = 10;
  int64 created_at = 11;
  int64 code_size = 12;
  string checksum = 13;
}

message ScriptInput {
  string name = 1;
  string description = 2;
  InputType type = 3;
  bool required = 4;
  google.protobuf.Value default_value = 5;
  repeated string allowed_values = 6;
  ValidationRule validation = 7;
}

message ScriptOutput {
  string name = 1;
  string description = 2;
  OutputType type = 3;
  string schema = 4;
}

message ValidationRule {
  string pattern = 1;
  double min = 2;
  double max = 3;
}

message ExecutionPermissions {
  bool allow_network = 1;
  bool allow_file_system = 2;
  bool allow_database = 3;
  repeated string allowed_apis = 4;
  repeated string allowed_libraries = 5;
  ResourceLimits resource_limits = 6;
}

message ResourceLimits {
  int32 max_cpu_percent = 1;
  int64 max_memory_mb = 2;
  int32 max_execution_time_seconds = 3;
  int32 max_file_size_mb = 4;
  int32 max_network_requests = 5;
}

// ==================== 执行消息 ====================

message ExecutionResult {
  string execution_id = 1;
  string script_id = 2;
  string version_id = 3;
  ExecutionStatus status = 4;
  google.protobuf.Struct outputs = 5;
  repeated LogEntry logs = 6;
  ExecutionMetrics metrics = 7;
  string error_message = 8;
  string error_stack = 9;
  int64 started_at = 10;
  int64 completed_at = 11;
  int64 duration_ms = 12;
}

message ExecutionStreamEvent {
  oneof event {
    LogEntry log = 1;
    ProgressUpdate progress = 2;
    OutputUpdate output = 3;
    ExecutionCompleted completed = 4;
    ExecutionError error = 5;
  }
}

message LogEntry {
  LogLevel level = 1;
  string message = 2;
  int64 timestamp = 3;
  map<string, string> metadata = 4;
}

message ProgressUpdate {
  int32 current = 1;
  int32 total = 2;
  string message = 3;
  double percentage = 4;
}

message OutputUpdate {
  string name = 1;
  google.protobuf.Value value = 2;
}

message ExecutionCompleted {
  google.protobuf.Struct outputs = 1;
  ExecutionMetrics metrics = 2;
}

message ExecutionError {
  string message = 1;
  string stack_trace = 2;
  ErrorType type = 3;
}

message ExecutionMetrics {
  int64 cpu_time_ms = 1;
  int64 memory_peak_mb = 2;
  int64 io_read_bytes = 3;
  int64 io_write_bytes = 4;
  int64 network_bytes = 5;
}

message AsyncExecutionHandle {
  string execution_id = 1;
  string status = 2;
  int64 estimated_completion = 3;
  string result_url = 4;
}

message ExecutionStatus {
  string execution_id = 1;
  ExecutionState state = 2;
  int64 started_at = 3;
  int64 updated_at = 4;
  int64 estimated_completion = 5;
  int32 progress_percent = 6;
  string current_operation = 7;
}

// ==================== 调试消息 ====================

message DebugEvent {
  oneof event {
    BreakpointHit breakpoint_hit = 1;
    VariableUpdate variable_update = 2;
    CallStackUpdate call_stack = 3;
    ExecutionPaused paused = 4;
    ExecutionResumed resumed = 5;
  }
}

message BreakpointHit {
  int32 line = 1;
  int32 column = 2;
  map<string, google.protobuf.Value> local_variables = 3;
}

message VariableUpdate {
  string name = 1;
  google.protobuf.Value value = 2;
  string scope = 3;
}

message CallStackUpdate {
  repeated StackFrame frames = 1;
}

message StackFrame {
  string function_name = 1;
  string file_path = 2;
  int32 line = 3;
  int32 column = 4;
}

// ==================== 请求/响应消息 ====================

message CreateScriptRequest {
  string tenant_id = 1;
  string name = 2;
  string description = 3;
  ScriptLanguage language = 4;
  ScriptType type = 5;
  ScriptScope scope = 6;
  string initial_code = 7;
  ExecutionPermissions permissions = 8;
  string created_by = 9;
}

message GetScriptRequest {
  string id = 1;
  string tenant_id = 2;
  bool include_versions = 3;
}

message UpdateScriptRequest {
  string id = 1;
  string tenant_id = 2;
  string name = 3;
  string description = 4;
  repeated string tags = 5;
  map<string, string> metadata = 6;
  ExecutionPermissions permissions = 7;
  string updated_by = 8;
}

message DeleteScriptRequest {
  string id = 1;
  string tenant_id = 2;
  bool force = 3;
}

message ListScriptsRequest {
  string tenant_id = 1;
  ScriptLanguage language = 2;
  ScriptType type = 3;
  string search = 4;
  repeated string tags = 5;
  int32 page_size = 6;
  string page_token = 7;
}

message ListScriptsResponse {
  repeated Script scripts = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ==================== 版本管理消息 ====================

message CreateVersionRequest {
  string script_id = 1;
  string version = 2;
  string code = 3;
  google.protobuf.Struct parameters = 4;
  repeated ScriptInput inputs = 5;
  repeated ScriptOutput outputs = 6;
  string changelog = 7;
  bool set_as_default = 8;
  string created_by = 9;
}

message GetVersionRequest {
  string script_id = 1;
  string version_id = 2;
}

message ListVersionsRequest {
  string script_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListVersionsResponse {
  repeated ScriptVersion versions = 1;
  string next_page_token = 2;
}

message SetDefaultVersionRequest {
  string script_id = 1;
  string version_id = 2;
}

message CompareVersionsRequest {
  string script_id = 1;
  string version_id_1 = 2;
  string version_id_2 = 3;
}

message VersionComparison {
  ScriptVersion version_1 = 1;
  ScriptVersion version_2 = 2;
  repeated DiffLine diff = 3;
  repeated string added_inputs = 4;
  repeated string removed_inputs = 5;
  repeated string modified_inputs = 6;
}

message DiffLine {
  int32 line_number = 1;
  string old_line = 2;
  string new_line = 3;
  DiffType type = 4;
}

// ==================== 执行请求消息 ====================

message ExecuteScriptRequest {
  string script_id = 1;
  string version_id = 2;
  google.protobuf.Struct inputs = 3;
  ExecutionOptions options = 4;
  string executed_by = 5;
  string tenant_id = 6;
  string document_id = 7;
}

message ExecutionOptions {
  bool capture_output = 1;
  bool capture_logs = 2;
  int32 timeout_seconds = 3;
  int32 max_memory_mb = 4;
  bool use_cache = 5;
  string cache_key = 6;
  ExecutionMode mode = 7;
  repeated string environment_variables = 8;
}

message ExecuteAsyncRequest {
  string script_id = 1;
  string version_id = 2;
  google.protobuf.Struct inputs = 3;
  ExecutionOptions options = 4;
  string callback_url = 5;
  string executed_by = 6;
  string tenant_id = 7;
}

message GetExecutionStatusRequest {
  string execution_id = 1;
}

message CancelExecutionRequest {
  string execution_id = 1;
  string reason = 2;
}

message GetExecutionResultRequest {
  string execution_id = 1;
}

message ListExecutionsRequest {
  string script_id = 1;
  string tenant_id = 2;
  ExecutionState state = 3;
  string executed_by = 4;
  int64 from_time = 5;
  int64 to_time = 6;
  int32 page_size = 7;
  string page_token = 8;
}

message ListExecutionsResponse {
  repeated ExecutionResult executions = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ==================== 调试请求消息 ====================

message DebugScriptRequest {
  string script_id = 1;
  string version_id = 2;
  google.protobuf.Struct inputs = 3;
  string executed_by = 4;
}

message SetBreakpointRequest {
  string execution_id = 1;
  int32 line = 2;
  int32 column = 3;
  string condition = 4;
}

message RemoveBreakpointRequest {
  string execution_id = 1;
  int32 line = 2;
}

message ContinueExecutionRequest {
  string execution_id = 1;
}

message StepExecutionRequest {
  string execution_id = 1;
  StepType step_type = 2;
}

// ==================== 模板消息 ====================

message GetScriptTemplatesRequest {
  ScriptLanguage language = 1;
  ScriptType type = 2;
}

message ScriptTemplatesResponse {
  repeated ScriptTemplate templates = 1;
}

message ScriptTemplate {
  string id = 1;
  string name = 2;
  string description = 3;
  ScriptLanguage language = 4;
  ScriptType type = 5;
  string code_template = 6;
  repeated ScriptInput inputs = 7;
  repeated ScriptOutput outputs = 8;
}

message CreateFromTemplateRequest {
  string template_id = 1;
  string tenant_id = 2;
  string name = 3;
  string created_by = 4;
}

// ==================== 库管理消息 ====================

message InstallLibraryRequest {
  string tenant_id = 1;
  string name = 2;
  string version = 3;
  string source = 4;
}

message UninstallLibraryRequest {
  string tenant_id = 1;
  string name = 2;
}

message ListLibrariesRequest {
  string tenant_id = 1;
  ScriptLanguage language = 2;
}

message ListLibrariesResponse {
  repeated Library libraries = 1;
}

message Library {
  string id = 1;
  string name = 2;
  string version = 3;
  ScriptLanguage language = 4;
  string description = 5;
  string author = 6;
  string license = 7;
  int64 installed_at = 8;
  bool is_system = 9;
}

// ==================== 枚举定义 ====================

enum ScriptLanguage {
  SCRIPT_LANGUAGE_UNSPECIFIED = 0;
  PYTHON = 1;
  JAVASCRIPT = 2;
  TYPESCRIPT = 3;
  LUA = 4;
  GLSL = 5;
  IRONPYTHON = 6;
  CSHARP_SCRIPT = 7;
}

enum ScriptType {
  SCRIPT_TYPE_UNSPECIFIED = 0;
  UTILITY = 1;           // 工具脚本
  AUTOMATION = 2;        // 自动化脚本
  VALIDATION = 3;        // 校验脚本
  GENERATION = 4;        // 生成脚本
  ANALYSIS = 5;          // 分析脚本
  EXPORT = 6;            // 导出脚本
  IMPORT = 7;            // 导入脚本
  SCHEDULED = 8;         // 定时脚本
}

enum ScriptScope {
  SCRIPT_SCOPE_UNSPECIFIED = 0;
  PERSONAL = 1;          // 个人
  PROJECT = 2;           // 项目
  ORGANIZATION = 3;      // 组织
  SYSTEM = 4;            // 系统
}

enum InputType {
  INPUT_TYPE_UNSPECIFIED = 0;
  STRING = 1;
  NUMBER = 2;
  BOOLEAN = 3;
  INTEGER = 4;
  ARRAY = 5;
  OBJECT = 6;
  FILE = 7;
  GEOMETRY = 8;
  ELEMENT = 9;
  ENUM = 10;
}

enum OutputType {
  OUTPUT_TYPE_UNSPECIFIED = 0;
  STRING = 1;
  NUMBER = 2;
  BOOLEAN = 3;
  INTEGER = 4;
  ARRAY = 5;
  OBJECT = 6;
  FILE = 7;
  GEOMETRY = 8;
  ELEMENT = 9;
  REPORT = 10;
}

enum ExecutionState {
  EXECUTION_STATE_UNSPECIFIED = 0;
  PENDING = 1;
  RUNNING = 2;
  PAUSED = 3;
  COMPLETED = 4;
  FAILED = 5;
  CANCELLED = 6;
  TIMEOUT = 7;
}

enum ExecutionMode {
  EXECUTION_MODE_UNSPECIFIED = 0;
  SYNC = 1;
  ASYNC = 2;
  STREAMING = 3;
  DEBUG = 4;
}

enum LogLevel {
  LOG_LEVEL_UNSPECIFIED = 0;
  DEBUG = 1;
  INFO = 2;
  WARNING = 3;
  ERROR = 4;
  CRITICAL = 5;
}

enum ErrorType {
  ERROR_TYPE_UNSPECIFIED = 0;
  SYNTAX_ERROR = 1;
  RUNTIME_ERROR = 2;
  TIMEOUT_ERROR = 3;
  MEMORY_ERROR = 4;
  PERMISSION_ERROR = 5;
  API_ERROR = 6;
  VALIDATION_ERROR = 7;
}

enum DiffType {
  DIFF_TYPE_UNSPECIFIED = 0;
  UNCHANGED = 1;
  ADDED = 2;
  REMOVED = 3;
  MODIFIED = 4;
}

enum StepType {
  STEP_TYPE_UNSPECIFIED = 0;
  OVER = 1;
  INTO = 2;
  OUT = 3;
}
```

## 4.3 数据库表结构设计

```sql
-- ==================== 脚本表 ====================
CREATE TABLE scripts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(256) NOT NULL,
    description TEXT,
    language VARCHAR(32) NOT NULL,
    script_type VARCHAR(32) NOT NULL,
    scope VARCHAR(32) NOT NULL DEFAULT 'PERSONAL',
    current_version_id UUID,
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    allow_network BOOLEAN DEFAULT FALSE,
    allow_file_system BOOLEAN DEFAULT FALSE,
    allow_database BOOLEAN DEFAULT FALSE,
    allowed_apis TEXT[],
    allowed_libraries TEXT[],
    max_cpu_percent INTEGER DEFAULT 50,
    max_memory_mb INTEGER DEFAULT 512,
    max_execution_time_seconds INTEGER DEFAULT 300,
    max_file_size_mb INTEGER DEFAULT 100,
    max_network_requests INTEGER DEFAULT 10,

    CONSTRAINT chk_language CHECK (language IN ('PYTHON', 'JAVASCRIPT', 'TYPESCRIPT', 'LUA', 'GLSL', 'IRONPYTHON', 'CSHARP_SCRIPT')),
    CONSTRAINT chk_script_type CHECK (script_type IN ('UTILITY', 'AUTOMATION', 'VALIDATION', 'GENERATION', 'ANALYSIS', 'EXPORT', 'IMPORT', 'SCHEDULED')),
    CONSTRAINT chk_scope CHECK (scope IN ('PERSONAL', 'PROJECT', 'ORGANIZATION', 'SYSTEM'))
);

CREATE INDEX idx_scripts_tenant ON scripts(tenant_id);
CREATE INDEX idx_scripts_language ON scripts(language);
CREATE INDEX idx_scripts_type ON scripts(script_type);
CREATE INDEX idx_scripts_scope ON scripts(scope);
CREATE INDEX idx_scripts_tags ON scripts USING GIN(tags);
CREATE INDEX idx_scripts_active ON scripts(is_active) WHERE is_active = TRUE;

-- ==================== 脚本版本表 ====================
CREATE TABLE script_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    script_id UUID NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    code TEXT NOT NULL,
    parameters JSONB DEFAULT '{}',
    inputs JSONB DEFAULT '[]',
    outputs JSONB DEFAULT '[]',
    changelog TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    code_size BIGINT,
    checksum VARCHAR(64),

    UNIQUE(script_id, version)
);

CREATE INDEX idx_script_versions_script ON script_versions(script_id);
CREATE INDEX idx_script_versions_default ON script_versions(script_id, is_default) WHERE is_default = TRUE;

-- ==================== 脚本执行记录表 ====================
CREATE TABLE script_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id VARCHAR(64) UNIQUE NOT NULL,
    script_id UUID NOT NULL REFERENCES scripts(id),
    version_id UUID NOT NULL REFERENCES script_versions(id),
    tenant_id UUID NOT NULL,
    document_id UUID,
    inputs JSONB,
    outputs JSONB,
    status VARCHAR(32) NOT NULL,
    logs JSONB DEFAULT '[]',
    metrics JSONB,
    error_message TEXT,
    error_stack TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms BIGINT,
    executed_by UUID NOT NULL,
    sandbox_id VARCHAR(64),
    cache_hit BOOLEAN DEFAULT FALSE,

    CONSTRAINT chk_status CHECK (status IN ('PENDING', 'RUNNING', 'PAUSED', 'COMPLETED', 'FAILED', 'CANCELLED', 'TIMEOUT'))
);

CREATE INDEX idx_executions_script ON script_executions(script_id);
CREATE INDEX idx_executions_tenant ON script_executions(tenant_id);
CREATE INDEX idx_executions_status ON script_executions(status);
CREATE INDEX idx_executions_time ON script_executions(started_at);
CREATE INDEX idx_executions_user ON script_executions(executed_by);

-- ==================== 执行缓存表 ====================
CREATE TABLE script_execution_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    script_id UUID NOT NULL,
    version_id UUID NOT NULL,
    cache_key VARCHAR(256) NOT NULL,
    inputs_hash VARCHAR(64) NOT NULL,
    outputs JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    hit_count INTEGER DEFAULT 0,

    UNIQUE(script_id, version_id, cache_key)
);

CREATE INDEX idx_cache_lookup ON script_execution_cache(script_id, version_id, cache_key);
CREATE INDEX idx_cache_expires ON script_execution_cache(expires_at);

-- ==================== 脚本库表 ====================
CREATE TABLE script_libraries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(128) NOT NULL,
    version VARCHAR(32) NOT NULL,
    language VARCHAR(32) NOT NULL,
    description TEXT,
    author VARCHAR(256),
    license VARCHAR(64),
    source_url TEXT,
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_system BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',

    UNIQUE(tenant_id, name, version)
);

CREATE INDEX idx_libraries_tenant ON script_libraries(tenant_id);
CREATE INDEX idx_libraries_language ON script_libraries(language);

-- ==================== 脚本权限表 ====================
CREATE TABLE script_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    script_id UUID NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    permission VARCHAR(32) NOT NULL,
    granted_by UUID NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT chk_permission CHECK (permission IN ('VIEW', 'EXECUTE', 'EDIT', 'DELETE', 'ADMIN')),
    UNIQUE(script_id, user_id, permission)
);

CREATE INDEX idx_script_permissions_script ON script_permissions(script_id);
CREATE INDEX idx_script_permissions_user ON script_permissions(user_id);

-- ==================== 触发器 ====================
CREATE TRIGGER trigger_scripts_updated_at
    BEFORE UPDATE ON scripts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

## 4.4 沙箱调用实现

```go
package script

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "time"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
)

// SandboxManager 沙箱管理器
type SandboxManager struct {
    docker     *client.Client
    imageCache map[string]string
    networks   []string
}

// SandboxConfig 沙箱配置
type SandboxConfig struct {
    Language        string
    Code            string
    Inputs          map[string]interface{}
    Timeout         time.Duration
    MemoryLimit     int64  // MB
    CPULimit        int64  // percent
    NetworkEnabled  bool
    FileSystemEnabled bool
    AllowedAPIs     []string
    EnvironmentVars map[string]string
}

// SandboxResult 沙箱执行结果
type SandboxResult struct {
    Outputs    map[string]interface{}
    Logs       []LogEntry
    Metrics    ExecutionMetrics
    Error      error
    Duration   time.Duration
}

// NewSandboxManager 创建沙箱管理器
func NewSandboxManager() (*SandboxManager, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return nil, err
    }

    return &SandboxManager{
        docker:     cli,
        imageCache: make(map[string]string),
        networks:   []string{"script-sandbox-net"},
    }, nil
}

// ExecuteInSandbox 在沙箱中执行脚本
func (m *SandboxManager) ExecuteInSandbox(
    ctx context.Context,
    config *SandboxConfig,
) (*SandboxResult, error) {

    // 创建超时上下文
    execCtx, cancel := context.WithTimeout(ctx, config.Timeout)
    defer cancel()

    // 获取或构建镜像
    image, err := m.getSandboxImage(execCtx, config.Language)
    if err != nil {
        return nil, fmt.Errorf("获取沙箱镜像失败: %w", err)
    }

    // 准备容器配置
    containerConfig := &container.Config{
        Image: image,
        Cmd:   m.buildCommand(config),
        Env:   m.buildEnvVars(config),
        Labels: map[string]string{
            "app":       "script-sandbox",
            "language":  config.Language,
            "timestamp": time.Now().Format(time.RFC3339),
        },
    }

    hostConfig := &container.HostConfig{
        Resources: container.Resources{
            Memory:     config.MemoryLimit * 1024 * 1024,
            MemorySwap: config.MemoryLimit * 1024 * 1024,
            CPUQuota:   config.CPULimit * 1000,
        },
        NetworkMode: "none",
        ReadonlyRootfs: true,
        CapDrop: []string{
            "ALL",
        },
        SecurityOpt: []string{
            "no-new-privileges:true",
            "seccomp=./seccomp-profile.json",
        },
    }

    // 如果需要网络访问
    if config.NetworkEnabled {
        hostConfig.NetworkMode = "bridge"
    }

    // 创建容器
    resp, err := m.docker.ContainerCreate(execCtx, containerConfig, hostConfig, nil, nil, "")
    if err != nil {
        return nil, fmt.Errorf("创建沙箱容器失败: %w", err)
    }

    containerID := resp.ID

    // 确保容器被清理
    defer func() {
        removeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        m.docker.ContainerRemove(removeCtx, containerID, types.ContainerRemoveOptions{
            Force: true,
        })
    }()

    // 复制代码到容器
    if err := m.copyCodeToContainer(execCtx, containerID, config); err != nil {
        return nil, fmt.Errorf("复制代码到沙箱失败: %w", err)
    }

    // 启动容器
    if err := m.docker.ContainerStart(execCtx, containerID, types.ContainerStartOptions{}); err != nil {
        return nil, fmt.Errorf("启动沙箱失败: %w", err)
    }

    // 等待执行完成
    statusCh, errCh := m.docker.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)

    select {
    case err := <-errCh:
        if err != nil {
            return nil, fmt.Errorf("沙箱执行错误: %w", err)
        }
    case status := <-statusCh:
        if status.StatusCode != 0 {
            // 获取错误日志
            logs, _ := m.getContainerLogs(execCtx, containerID)
            return nil, fmt.Errorf("脚本执行失败，退出码: %d, 日志: %s", status.StatusCode, logs)
        }
    case <-execCtx.Done():
        return nil, fmt.Errorf("脚本执行超时")
    }

    // 获取执行结果
    result, err := m.getExecutionResult(execCtx, containerID)
    if err != nil {
        return nil, fmt.Errorf("获取执行结果失败: %w", err)
    }

    return result, nil
}

// getSandboxImage 获取沙箱镜像
func (m *SandboxManager) getSandboxImage(ctx context.Context, language string) (string, error) {
    imageMap := map[string]string{
        "PYTHON":       "archplatform/sandbox-python:3.11",
        "JAVASCRIPT":   "archplatform/sandbox-node:18",
        "TYPESCRIPT":   "archplatform/sandbox-node:18-ts",
        "LUA":          "archplatform/sandbox-lua:5.4",
        "CSHARP_SCRIPT": "archplatform/sandbox-dotnet:6.0",
    }

    image, ok := imageMap[language]
    if !ok {
        return "", fmt.Errorf("不支持的语言: %s", language)
    }

    // 检查镜像是否存在
    _, _, err := m.docker.ImageInspectWithRaw(ctx, image)
    if err != nil {
        // 拉取镜像
        pullResp, err := m.docker.ImagePull(ctx, image, types.ImagePullOptions{})
        if err != nil {
            return "", err
        }
        defer pullResp.Close()
        io.Copy(io.Discard, pullResp)
    }

    return image, nil
}

// buildCommand 构建执行命令
func (m *SandboxManager) buildCommand(config *SandboxConfig) []string {
    switch config.Language {
    case "PYTHON":
        return []string{"python", "/sandbox/runner.py"}
    case "JAVASCRIPT", "TYPESCRIPT":
        return []string{"node", "/sandbox/runner.js"}
    case "LUA":
        return []string{"lua", "/sandbox/runner.lua"}
    case "CSHARP_SCRIPT":
        return []string{"dotnet", "/sandbox/runner.dll"}
    default:
        return []string{"/sandbox/runner"}
    }
}

// buildEnvVars 构建环境变量
func (m *SandboxManager) buildEnvVars(config *SandboxConfig) []string {
    envs := []string{
        fmt.Sprintf("SCRIPT_TIMEOUT=%d", int(config.Timeout.Seconds())),
        fmt.Sprintf("SCRIPT_MEMORY_LIMIT=%d", config.MemoryLimit),
    }

    // 序列化输入
    inputsJSON, _ := json.Marshal(config.Inputs)
    envs = append(envs, fmt.Sprintf("SCRIPT_INPUTS=%s", string(inputsJSON)))

    // 允许的API
    if len(config.AllowedAPIs) > 0 {
        apisJSON, _ := json.Marshal(config.AllowedAPIs)
        envs = append(envs, fmt.Sprintf("ALLOWED_APIS=%s", string(apisJSON)))
    }

    // 自定义环境变量
    for k, v := range config.EnvironmentVars {
        envs = append(envs, fmt.Sprintf("%s=%s", k, v))
    }

    return envs
}

// copyCodeToContainer 复制代码到容器
func (m *SandboxManager) copyCodeToContainer(
    ctx context.Context, 
    containerID string, 
    config *SandboxConfig,
) error {
    // 使用Docker SDK的CopyToContainer
    // 创建tar归档包含代码文件
    // ...
    return nil
}

// getContainerLogs 获取容器日志
func (m *SandboxManager) getContainerLogs(ctx context.Context, containerID string) (string, error) {
    options := types.ContainerLogsOptions{
        ShowStdout: true,
        ShowStderr: true,
    }

    logs, err := m.docker.ContainerLogs(ctx, containerID, options)
    if err != nil {
        return "", err
    }
    defer logs.Close()

    buf := new(strings.Builder)
    io.Copy(buf, logs)
    return buf.String(), nil
}

// getExecutionResult 获取执行结果
func (m *SandboxManager) getExecutionResult(
    ctx context.Context, 
    containerID string,
) (*SandboxResult, error) {
    // 从容器中复制结果文件
    // ...
    return &SandboxResult{}, nil
}

// ExecuteStream 流式执行脚本
func (m *SandboxManager) ExecuteStream(
    ctx context.Context,
    config *SandboxConfig,
    eventCh chan<- *pb.ExecutionStreamEvent,
) error {
    // 创建容器并附加到输出流
    // 实时转发日志和进度事件
    // ...
    return nil
}
```

## 4.5 脚本版本管理实现

```go
package script

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/sergi/go-diff/diffmatchpatch"
)

// VersionManager 版本管理器
type VersionManager struct {
    db *pgxpool.Pool
}

// CreateVersion 创建新版本
func (m *VersionManager) CreateVersion(
    ctx context.Context,
    req *pb.CreateVersionRequest,
) (*pb.ScriptVersion, error) {
    // 验证版本号格式
    if !m.isValidVersion(req.Version) {
        return nil, fmt.Errorf("无效的版本号格式: %s", req.Version)
    }

    // 检查版本是否已存在
    var exists bool
    err := m.db.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM script_versions 
            WHERE script_id = $1 AND version = $2
        )
    `, req.ScriptId, req.Version).Scan(&exists)

    if err != nil {
        return nil, err
    }

    if exists {
        return nil, fmt.Errorf("版本 %s 已存在", req.Version)
    }

    // 计算代码校验和
    checksum := m.calculateChecksum(req.Code)

    // 创建版本
    version := &pb.ScriptVersion{
        Id:                generateUUID(),
        ScriptId:          req.ScriptId,
        Version:           req.Version,
        Code:              req.Code,
        Parameters:        req.Parameters,
        Inputs:            req.Inputs,
        Outputs:           req.Outputs,
        Changelog:         req.Changelog,
        IsDefault:         req.SetAsDefault,
        CreatedBy:         req.CreatedBy,
        CreatedAt:         time.Now().Unix(),
        CodeSize:          int64(len(req.Code)),
        Checksum:          checksum,
    }

    // 持久化到数据库
    _, err = m.db.Exec(ctx, `
        INSERT INTO script_versions (
            id, script_id, version, code, parameters, inputs, outputs,
            changelog, is_default, created_by, code_size, checksum
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `,
        version.Id, version.ScriptId, version.Version, version.Code,
        version.Parameters, version.Inputs, version.Outputs,
        version.Changelog, version.IsDefault, version.CreatedBy,
        version.CodeSize, version.Checksum,
    )

    if err != nil {
        return nil, err
    }

    // 如果设置为默认版本，更新脚本
    if req.SetAsDefault {
        _, err = m.db.Exec(ctx, `
            UPDATE scripts 
            SET current_version_id = $1, updated_at = NOW()
            WHERE id = $2
        `, version.Id, req.ScriptId)

        if err != nil {
            return nil, err
        }

        // 取消其他版本的默认状态
        _, err = m.db.Exec(ctx, `
            UPDATE script_versions 
            SET is_default = FALSE
            WHERE script_id = $1 AND id != $2
        `, req.ScriptId, version.Id)

        if err != nil {
            return nil, err
        }
    }

    return version, nil
}

// CompareVersions 比较两个版本
func (m *VersionManager) CompareVersions(
    ctx context.Context,
    req *pb.CompareVersionsRequest,
) (*pb.VersionComparison, error) {
    // 获取两个版本
    v1, err := m.GetVersion(ctx, &pb.GetVersionRequest{
        ScriptId:  req.ScriptId,
        VersionId: req.VersionId1,
    })
    if err != nil {
        return nil, err
    }

    v2, err := m.GetVersion(ctx, &pb.GetVersionRequest{
        ScriptId:  req.ScriptId,
        VersionId: req.VersionId2,
    })
    if err != nil {
        return nil, err
    }

    // 比较代码差异
    dmp := diffmatchpatch.New()
    diffs := dmp.DiffMain(v1.Code, v2.Code, false)

    var diffLines []*pb.DiffLine
    lineNum := 1

    for _, diff := range diffs {
        lines := strings.Split(diff.Text, "
")
        for _, line := range lines {
            if line == "" {
                continue
            }

            var diffType pb.DiffType
            switch diff.Type {
            case diffmatchpatch.DiffEqual:
                diffType = pb.DiffType_UNCHANGED
            case diffmatchpatch.DiffInsert:
                diffType = pb.DiffType_ADDED
            case diffmatchpatch.DiffDelete:
                diffType = pb.DiffType_REMOVED
            }

            diffLines = append(diffLines, &pb.DiffLine{
                LineNumber: int32(lineNum),
                OldLine:    line,
                NewLine:    line,
                Type:       diffType,
            })
            lineNum++
        }
    }

    // 比较输入参数
    v1Inputs := make(map[string]*pb.ScriptInput)
    for _, input := range v1.Inputs {
        v1Inputs[input.Name] = input
    }

    v2Inputs := make(map[string]*pb.ScriptInput)
    for _, input := range v2.Inputs {
        v2Inputs[input.Name] = input
    }

    var addedInputs, removedInputs, modifiedInputs []string

    for name := range v2Inputs {
        if _, ok := v1Inputs[name]; !ok {
            addedInputs = append(addedInputs, name)
        } else if !proto.Equal(v1Inputs[name], v2Inputs[name]) {
            modifiedInputs = append(modifiedInputs, name)
        }
    }

    for name := range v1Inputs {
        if _, ok := v2Inputs[name]; !ok {
            removedInputs = append(removedInputs, name)
        }
    }

    return &pb.VersionComparison{
        Version1:        v1,
        Version2:        v2,
        Diff:            diffLines,
        AddedInputs:     addedInputs,
        RemovedInputs:   removedInputs,
        ModifiedInputs:  modifiedInputs,
    }, nil
}

// RollbackVersion 回滚到指定版本
func (m *VersionManager) RollbackVersion(
    ctx context.Context,
    scriptID string,
    versionID string,
) error {
    // 获取目标版本
    version, err := m.GetVersion(ctx, &pb.GetVersionRequest{
        ScriptId:  scriptID,
        VersionId: versionID,
    })
    if err != nil {
        return err
    }

    // 创建回滚版本
    rollbackVersion := fmt.Sprintf("%s-rollback", version.Version)

    _, err = m.CreateVersion(ctx, &pb.CreateVersionRequest{
        ScriptId:   scriptID,
        Version:    rollbackVersion,
        Code:       version.Code,
        Parameters: version.Parameters,
        Inputs:     version.Inputs,
        Outputs:    version.Outputs,
        Changelog:  fmt.Sprintf("Rollback to version %s", version.Version),
        SetAsDefault: true,
    })

    return err
}

// calculateChecksum 计算代码校验和
func (m *VersionManager) calculateChecksum(code string) string {
    hash := sha256.Sum256([]byte(code))
    return hex.EncodeToString(hash[:])
}

// isValidVersion 验证版本号格式
func (m *VersionManager) isValidVersion(version string) bool {
    // 支持语义化版本: x.y.z 或 x.y.z-prerelease
    pattern := `^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`
    matched, _ := regexp.MatchString(pattern, version)
    return matched
}
```

## 4.6 脚本权限控制实现

```go
package script

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

// PermissionManager 权限管理器
type PermissionManager struct {
    db *pgxpool.Pool
}

// PermissionLevel 权限级别
type PermissionLevel int

const (
    PermissionNone PermissionLevel = iota
    PermissionView
    PermissionExecute
    PermissionEdit
    PermissionDelete
    PermissionAdmin
)

// CheckPermission 检查权限
func (m *PermissionManager) CheckPermission(
    ctx context.Context,
    scriptID string,
    userID string,
    required PermissionLevel,
) (bool, error) {
    // 获取用户权限级别
    userLevel, err := m.getUserPermissionLevel(ctx, scriptID, userID)
    if err != nil {
        return false, err
    }

    return userLevel >= required, nil
}

// getUserPermissionLevel 获取用户权限级别
func (m *PermissionManager) getUserPermissionLevel(
    ctx context.Context,
    scriptID string,
    userID string,
) (PermissionLevel, error) {
    // 查询直接权限
    var permission string
    err := m.db.QueryRow(ctx, `
        SELECT permission FROM script_permissions
        WHERE script_id = $1 AND user_id = $2
          AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY CASE permission
            WHEN 'ADMIN' THEN 5
            WHEN 'DELETE' THEN 4
            WHEN 'EDIT' THEN 3
            WHEN 'EXECUTE' THEN 2
            WHEN 'VIEW' THEN 1
        END DESC
        LIMIT 1
    `, scriptID, userID).Scan(&permission)

    if err == nil {
        return m.parsePermissionLevel(permission), nil
    }

    // 检查脚本所有者
    var ownerID string
    err = m.db.QueryRow(ctx, `
        SELECT created_by FROM scripts WHERE id = $1
    `, scriptID).Scan(&ownerID)

    if err == nil && ownerID == userID {
        return PermissionAdmin, nil
    }

    // 检查脚本范围
    var scope string
    err = m.db.QueryRow(ctx, `
        SELECT scope FROM scripts WHERE id = $1
    `, scriptID).Scan(&scope)

    if err == nil {
        switch scope {
        case "SYSTEM":
            return PermissionExecute, nil
        case "ORGANIZATION":
            // 检查用户是否在同一组织
            // ...
            return PermissionView, nil
        case "PROJECT":
            // 检查用户是否在项目中
            // ...
            return PermissionView, nil
        }
    }

    return PermissionNone, nil
}

// GrantPermission 授予权限
func (m *PermissionManager) GrantPermission(
    ctx context.Context,
    scriptID string,
    userID string,
    permission string,
    grantedBy string,
    expiresAt *time.Time,
) error {
    // 检查授予者是否有权限
    canGrant, err := m.CheckPermission(ctx, scriptID, grantedBy, PermissionAdmin)
    if err != nil {
        return err
    }

    if !canGrant {
        return fmt.Errorf("无权授予权限")
    }

    // 授予权限
    _, err = m.db.Exec(ctx, `
        INSERT INTO script_permissions (
            script_id, user_id, permission, granted_by, expires_at
        ) VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (script_id, user_id, permission) 
        DO UPDATE SET 
            granted_by = EXCLUDED.granted_by,
            granted_at = NOW(),
            expires_at = EXCLUDED.expires_at
    `, scriptID, userID, permission, grantedBy, expiresAt)

    return err
}

// RevokePermission 撤销权限
func (m *PermissionManager) RevokePermission(
    ctx context.Context,
    scriptID string,
    userID string,
    permission string,
    revokedBy string,
) error {
    // 检查撤销者是否有权限
    canRevoke, err := m.CheckPermission(ctx, scriptID, revokedBy, PermissionAdmin)
    if err != nil {
        return err
    }

    if !canRevoke {
        return fmt.Errorf("无权撤销权限")
    }

    _, err = m.db.Exec(ctx, `
        DELETE FROM script_permissions
        WHERE script_id = $1 AND user_id = $2 AND permission = $3
    `, scriptID, userID, permission)

    return err
}

// ValidateExecutionPermissions 验证执行权限
func (m *PermissionManager) ValidateExecutionPermissions(
    ctx context.Context,
    scriptID string,
    userID string,
    requestedPermissions *pb.ExecutionPermissions,
) error {
    // 获取脚本配置
    var scriptPerms ScriptPermissions
    err := m.db.QueryRow(ctx, `
        SELECT allow_network, allow_file_system, allow_database,
               allowed_apis, allowed_libraries,
               max_cpu_percent, max_memory_mb, max_execution_time_seconds
        FROM scripts WHERE id = $1
    `, scriptID).Scan(
        &scriptPerms.AllowNetwork,
        &scriptPerms.AllowFileSystem,
        &scriptPerms.AllowDatabase,
        &scriptPerms.AllowedAPIs,
        &scriptPerms.AllowedLibraries,
        &scriptPerms.MaxCPUPercent,
        &scriptPerms.MaxMemoryMB,
        &scriptPerms.MaxExecutionTimeSeconds,
    )

    if err != nil {
        return err
    }

    // 验证请求权限是否在允许范围内
    if requestedPermissions.AllowNetwork && !scriptPerms.AllowNetwork {
        return fmt.Errorf("脚本不允许网络访问")
    }

    if requestedPermissions.AllowFileSystem && !scriptPerms.AllowFileSystem {
        return fmt.Errorf("脚本不允许文件系统访问")
    }

    if requestedPermissions.AllowDatabase && !scriptPerms.AllowDatabase {
        return fmt.Errorf("脚本不允许数据库访问")
    }

    // 验证API权限
    for _, api := range requestedPermissions.AllowedApis {
        if !contains(scriptPerms.AllowedAPIs, api) {
            return fmt.Errorf("脚本不允许访问API: %s", api)
        }
    }

    // 验证库权限
    for _, lib := range requestedPermissions.AllowedLibraries {
        if !contains(scriptPerms.AllowedLibraries, lib) {
            return fmt.Errorf("脚本不允许使用库: %s", lib)
        }
    }

    return nil
}

// parsePermissionLevel 解析权限级别
func (m *PermissionManager) parsePermissionLevel(permission string) PermissionLevel {
    switch permission {
    case "VIEW":
        return PermissionView
    case "EXECUTE":
        return PermissionExecute
    case "EDIT":
        return PermissionEdit
    case "DELETE":
        return PermissionDelete
    case "ADMIN":
        return PermissionAdmin
    default:
        return PermissionNone
    }
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

type ScriptPermissions struct {
    AllowNetwork            bool
    AllowFileSystem         bool
    AllowDatabase           bool
    AllowedAPIs             []string
    AllowedLibraries        []string
    MaxCPUPercent           int32
    MaxMemoryMB             int64
    MaxExecutionTimeSeconds int32
}
```

---

---

# 5. 版本服务详细设计

## 5.1 服务概述

版本服务负责管理设计文档的版本控制，基于Event Sourcing模式实现完整的变更历史追踪，支持分支管理和快照功能。

### 核心功能
- 版本控制 (类似Git)
- Event Sourcing事件溯源
- 快照管理
- 分支管理
- 变更历史查询

## 5.2 gRPC接口定义

```protobuf
syntax = "proto3";

package version.v1;

option go_package = "github.com/archplatform/version-service/api/v1";

import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

// 版本服务
service VersionService {
  // 版本管理
  rpc CreateVersion(CreateVersionRequest) returns (Version);
  rpc GetVersion(GetVersionRequest) returns (Version);
  rpc ListVersions(ListVersionsRequest) returns (ListVersionsResponse);
  rpc DeleteVersion(DeleteVersionRequest) returns (google.protobuf.Empty);

  // 分支管理
  rpc CreateBranch(CreateBranchRequest) returns (Branch);
  rpc GetBranch(GetBranchRequest) returns (Branch);
  rpc ListBranches(ListBranchesRequest) returns (ListBranchesResponse);
  rpc SwitchBranch(SwitchBranchRequest) returns (Version);
  rpc MergeBranch(MergeBranchRequest) returns (MergeResult);
  rpc DeleteBranch(DeleteBranchRequest) returns (google.protobuf.Empty);

  // 事件溯源
  rpc AppendEvent(AppendEventRequest) returns (Event);
  rpc GetEvents(GetEventsRequest) returns (EventStream);
  rpc GetEventRange(GetEventRangeRequest) returns (EventStream);
  rpc ReplayEvents(ReplayEventsRequest) returns (ReplayResult);

  // 快照管理
  rpc CreateSnapshot(CreateSnapshotRequest) returns (Snapshot);
  rpc GetSnapshot(GetSnapshotRequest) returns (Snapshot);
  rpc ListSnapshots(ListSnapshotsRequest) returns (ListSnapshotsResponse);
  rpc RestoreSnapshot(RestoreSnapshotRequest) returns (Version);
  rpc DeleteSnapshot(DeleteSnapshotRequest) returns (google.protobuf.Empty);

  // 变更历史
  rpc GetChangeHistory(GetChangeHistoryRequest) returns (ChangeHistory);
  rpc GetDiff(GetDiffRequest) returns (DiffResult);
  rpc CompareVersions(CompareVersionsRequest) returns (VersionComparison);

  // 标签管理
  rpc CreateTag(CreateTagRequest) returns (Tag);
  rpc GetTag(GetTagRequest) returns (Tag);
  rpc ListTags(ListTagsRequest) returns (ListTagsResponse);
  rpc DeleteTag(DeleteTagRequest) returns (google.protobuf.Empty);
}

// ==================== 版本消息 ====================

message Version {
  string id = 1;
  string document_id = 2;
  string tenant_id = 3;
  string branch_id = 4;
  string branch_name = 5;
  string parent_version_id = 6;
  string message = 7;
  string author_id = 8;
  string author_name = 9;
  int64 created_at = 10;
  int64 sequence_number = 11;
  string snapshot_id = 12;
  map<string, string> metadata = 13;
  repeated string tag_ids = 14;
}

// ==================== 分支消息 ====================

message Branch {
  string id = 1;
  string document_id = 2;
  string tenant_id = 3;
  string name = 4;
  string description = 5;
  string base_branch_id = 6;
  string base_version_id = 7;
  string head_version_id = 8;
  bool is_default = 9;
  bool is_protected = 10;
  string created_by = 11;
  int64 created_at = 12;
  int64 updated_at = 13;
  map<string, string> metadata = 14;
}

// ==================== 事件消息 ====================

message Event {
  string id = 1;
  string document_id = 2;
  string tenant_id = 3;
  string version_id = 4;
  int64 sequence_number = 5;
  EventType type = 6;
  string entity_type = 7;
  string entity_id = 8;
  google.protobuf.Struct payload = 9;
  string author_id = 10;
  int64 timestamp = 11;
  string correlation_id = 12;
  string causation_id = 13;
  map<string, string> metadata = 14;
}

message EventStream {
  repeated Event events = 1;
  bool has_more = 2;
  string next_cursor = 3;
  int64 total_count = 4;
}

// ==================== 快照消息 ====================

message Snapshot {
  string id = 1;
  string document_id = 2;
  string tenant_id = 3;
  string version_id = 4;
  int64 sequence_number = 5;
  bytes data = 6;
  int64 data_size = 7;
  string checksum = 8;
  CompressionType compression = 9;
  int64 created_at = 10;
  map<string, string> metadata = 11;
}

// ==================== 变更历史消息 ====================

message ChangeHistory {
  string document_id = 1;
  repeated ChangeEntry entries = 2;
  int64 total_count = 3;
  string next_cursor = 4;
}

message ChangeEntry {
  string version_id = 1;
  string author_id = 2;
  string author_name = 3;
  string message = 4;
  int64 timestamp = 5;
  repeated FileChange file_changes = 6;
  int32 insertions = 7;
  int32 deletions = 8;
}

message FileChange {
  string path = 1;
  ChangeType type = 2;
  int32 insertions = 3;
  int32 deletions = 4;
  string old_hash = 5;
  string new_hash = 6;
}

// ==================== 差异消息 ====================

message DiffResult {
  string from_version_id = 1;
  string to_version_id = 2;
  repeated FileDiff file_diffs = 3;
  int32 total_insertions = 4;
  int32 total_deletions = 5;
  int32 files_changed = 6;
}

message FileDiff {
  string path = 1;
  ChangeType type = 2;
  string old_content = 3;
  string new_content = 4;
  repeated DiffHunk hunks = 5;
  bool is_binary = 6;
}

message DiffHunk {
  int32 old_start = 1;
  int32 old_lines = 2;
  int32 new_start = 3;
  int32 new_lines = 4;
  repeated DiffLine lines = 5;
}

message DiffLine {
  LineType type = 1;
  string content = 2;
  int32 old_line_number = 3;
  int32 new_line_number = 4;
}

// ==================== 合并消息 ====================

message MergeResult {
  bool success = 1;
  string merged_version_id = 2;
  repeated MergeConflict conflicts = 3;
  string message = 4;
}

message MergeConflict {
  string path = 1;
  string base_content = 2;
  string ours_content = 3;
  string theirs_content = 4;
  ConflictType type = 5;
}

// ==================== 标签消息 ====================

message Tag {
  string id = 1;
  string document_id = 2;
  string tenant_id = 3;
  string name = 4;
  string description = 5;
  string version_id = 6;
  string created_by = 7;
  int64 created_at = 8;
  map<string, string> metadata = 9;
}

// ==================== 请求/响应消息 ====================

message CreateVersionRequest {
  string document_id = 1;
  string tenant_id = 2;
  string branch_id = 3;
  string message = 4;
  repeated Event events = 5;
  string author_id = 6;
  map<string, string> metadata = 7;
  bool create_snapshot = 8;
}

message GetVersionRequest {
  string id = 1;
  string tenant_id = 2;
  bool include_events = 3;
}

message ListVersionsRequest {
  string document_id = 1;
  string tenant_id = 2;
  string branch_id = 3;
  int32 page_size = 4;
  string page_token = 5;
}

message ListVersionsResponse {
  repeated Version versions = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message DeleteVersionRequest {
  string id = 1;
  string tenant_id = 2;
  bool force = 3;
}

// ==================== 分支请求消息 ====================

message CreateBranchRequest {
  string document_id = 1;
  string tenant_id = 2;
  string name = 3;
  string description = 4;
  string base_branch_id = 5;
  string base_version_id = 6;
  string created_by = 7;
  bool is_protected = 8;
}

message GetBranchRequest {
  string id = 1;
  string tenant_id = 2;
}

message ListBranchesRequest {
  string document_id = 1;
  string tenant_id = 2;
  bool include_merged = 3;
}

message ListBranchesResponse {
  repeated Branch branches = 1;
}

message SwitchBranchRequest {
  string document_id = 1;
  string tenant_id = 2;
  string branch_id = 3;
  string user_id = 4;
}

message MergeBranchRequest {
  string document_id = 1;
  string tenant_id = 2;
  string source_branch_id = 3;
  string target_branch_id = 4;
  string author_id = 5;
  string message = 6;
  MergeStrategy strategy = 7;
}

message DeleteBranchRequest {
  string id = 1;
  string tenant_id = 2;
  bool force = 3;
}

// ==================== 事件请求消息 ====================

message AppendEventRequest {
  string document_id = 1;
  string tenant_id = 2;
  string version_id = 3;
  EventType type = 4;
  string entity_type = 5;
  string entity_id = 6;
  google.protobuf.Struct payload = 7;
  string author_id = 8;
  string correlation_id = 9;
  string causation_id = 10;
  map<string, string> metadata = 11;
}

message GetEventsRequest {
  string document_id = 1;
  string tenant_id = 2;
  string version_id = 3;
  int64 from_sequence = 4;
  int32 limit = 5;
}

message GetEventRangeRequest {
  string document_id = 1;
  string tenant_id = 2;
  int64 from_sequence = 3;
  int64 to_sequence = 4;
}

message ReplayEventsRequest {
  string document_id = 1;
  string tenant_id = 2;
  int64 from_sequence = 3;
  int64 to_sequence = 4;
  bool create_new_version = 5;
}

message ReplayResult {
  bool success = 1;
  string new_version_id = 2;
  int64 events_replayed = 3;
  string message = 4;
}

// ==================== 快照请求消息 ====================

message CreateSnapshotRequest {
  string document_id = 1;
  string tenant_id = 2;
  string version_id = 3;
  bytes data = 4;
  CompressionType compression = 5;
  map<string, string> metadata = 6;
}

message GetSnapshotRequest {
  string id = 1;
  string tenant_id = 2;
  bool decompress = 3;
}

message ListSnapshotsRequest {
  string document_id = 1;
  string tenant_id = 2;
  int32 page_size = 3;
  string page_token = 4;
}

message ListSnapshotsResponse {
  repeated Snapshot snapshots = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message RestoreSnapshotRequest {
  string id = 1;
  string tenant_id = 2;
  string branch_id = 3;
  string author_id = 4;
  string message = 5;
}

message DeleteSnapshotRequest {
  string id = 1;
  string tenant_id = 2;
}

// ==================== 变更历史请求消息 ====================

message GetChangeHistoryRequest {
  string document_id = 1;
  string tenant_id = 2;
  string branch_id = 3;
  string author_id = 4;
  int64 from_time = 5;
  int64 to_time = 6;
  int32 page_size = 7;
  string page_token = 8;
}

message GetDiffRequest {
  string from_version_id = 1;
  string to_version_id = 2;
  string tenant_id = 3;
  repeated string paths = 4;
}

message CompareVersionsRequest {
  string document_id = 1;
  string tenant_id = 2;
  string version_id_1 = 3;
  string version_id_2 = 4;
}

message VersionComparison {
  Version version_1 = 1;
  Version version_2 = 2;
  DiffResult diff = 3;
  repeated string common_ancestors = 4;
  int64 commits_behind = 5;
  int64 commits_ahead = 6;
}

// ==================== 标签请求消息 ====================

message CreateTagRequest {
  string document_id = 1;
  string tenant_id = 2;
  string name = 3;
  string description = 4;
  string version_id = 5;
  string created_by = 6;
  map<string, string> metadata = 7;
}

message GetTagRequest {
  string id = 1;
  string tenant_id = 2;
}

message ListTagsRequest {
  string document_id = 1;
  string tenant_id = 2;
  int32 page_size = 3;
  string page_token = 4;
}

message ListTagsResponse {
  repeated Tag tags = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message DeleteTagRequest {
  string id = 1;
  string tenant_id = 2;
}

// ==================== 枚举定义 ====================

enum EventType {
  EVENT_TYPE_UNSPECIFIED = 0;
  ELEMENT_CREATED = 1;
  ELEMENT_UPDATED = 2;
  ELEMENT_DELETED = 3;
  GEOMETRY_CHANGED = 4;
  PROPERTY_CHANGED = 5;
  RELATIONSHIP_CHANGED = 6;
  METADATA_CHANGED = 7;
  SNAPSHOT_CREATED = 8;
  BRANCH_CREATED = 9;
  BRANCH_MERGED = 10;
  TAG_CREATED = 11;
  TAG_DELETED = 12;
}

enum ChangeType {
  CHANGE_TYPE_UNSPECIFIED = 0;
  ADDED = 1;
  MODIFIED = 2;
  DELETED = 3;
  RENAMED = 4;
  COPIED = 5;
}

enum LineType {
  LINE_TYPE_UNSPECIFIED = 0;
  CONTEXT = 1;
  INSERTION = 2;
  DELETION = 3;
}

enum CompressionType {
  COMPRESSION_TYPE_UNSPECIFIED = 0;
  NONE = 1;
  GZIP = 2;
  ZSTD = 3;
  LZ4 = 4;
}

enum MergeStrategy {
  MERGE_STRATEGY_UNSPECIFIED = 0;
  RECURSIVE = 1;
  ORT = 2;
  OURS = 3;
  THEIRS = 4;
}

enum ConflictType {
  CONFLICT_TYPE_UNSPECIFIED = 0;
  CONTENT = 1;
  RENAME = 2;
  DELETE = 3;
  ADD_ADD = 4;
}
```

## 5.3 Event Sourcing实现

```go
package version

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

// EventStore 事件存储
type EventStore struct {
    db *pgxpool.Pool
}

// Event 领域事件
type Event struct {
    ID             string
    DocumentID     string
    TenantID       string
    VersionID      string
    SequenceNumber int64
    Type           string
    EntityType     string
    EntityID       string
    Payload        map[string]interface{}
    AuthorID       string
    Timestamp      time.Time
    CorrelationID  string
    CausationID    string
    Metadata       map[string]string
}

// AppendEvent 追加事件
func (s *EventStore) AppendEvent(
    ctx context.Context,
    event *Event,
) (*Event, error) {
    // 获取下一个序列号
    var seqNum int64
    err := s.db.QueryRow(ctx, `
        SELECT COALESCE(MAX(sequence_number), 0) + 1
        FROM events
        WHERE document_id = $1
    `, event.DocumentID).Scan(&seqNum)

    if err != nil {
        return nil, fmt.Errorf("获取序列号失败: %w", err)
    }

    event.ID = uuid.New().String()
    event.SequenceNumber = seqNum
    event.Timestamp = time.Now()

    payloadJSON, _ := json.Marshal(event.Payload)
    metadataJSON, _ := json.Marshal(event.Metadata)

    // 插入事件
    _, err = s.db.Exec(ctx, `
        INSERT INTO events (
            id, document_id, tenant_id, version_id, sequence_number,
            event_type, entity_type, entity_id, payload, author_id,
            timestamp, correlation_id, causation_id, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
    `,
        event.ID, event.DocumentID, event.TenantID, event.VersionID,
        event.SequenceNumber, event.Type, event.EntityType, event.EntityID,
        payloadJSON, event.AuthorID, event.Timestamp, event.CorrelationID,
        event.CausationID, metadataJSON,
    )

    if err != nil {
        return nil, fmt.Errorf("追加事件失败: %w", err)
    }

    // 发布事件到消息队列
    go s.publishEvent(event)

    return event, nil
}

// GetEvents 获取事件流
func (s *EventStore) GetEvents(
    ctx context.Context,
    documentID string,
    fromSequence int64,
    limit int32,
) ([]*Event, error) {
    rows, err := s.db.Query(ctx, `
        SELECT id, document_id, tenant_id, version_id, sequence_number,
               event_type, entity_type, entity_id, payload, author_id,
               timestamp, correlation_id, causation_id, metadata
        FROM events
        WHERE document_id = $1 AND sequence_number >= $2
        ORDER BY sequence_number ASC
        LIMIT $3
    `, documentID, fromSequence, limit)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []*Event
    for rows.Next() {
        event := &Event{}
        var payloadJSON, metadataJSON []byte

        err := rows.Scan(
            &event.ID, &event.DocumentID, &event.TenantID, &event.VersionID,
            &event.SequenceNumber, &event.Type, &event.EntityType, &event.EntityID,
            &payloadJSON, &event.AuthorID, &event.Timestamp, &event.CorrelationID,
            &event.CausationID, &metadataJSON,
        )
        if err != nil {
            continue
        }

        json.Unmarshal(payloadJSON, &event.Payload)
        json.Unmarshal(metadataJSON, &event.Metadata)

        events = append(events, event)
    }

    return events, nil
}

// GetEventsByVersion 获取版本的所有事件
func (s *EventStore) GetEventsByVersion(
    ctx context.Context,
    versionID string,
) ([]*Event, error) {
    rows, err := s.db.Query(ctx, `
        SELECT id, document_id, tenant_id, version_id, sequence_number,
               event_type, entity_type, entity_id, payload, author_id,
               timestamp, correlation_id, causation_id, metadata
        FROM events
        WHERE version_id = $1
        ORDER BY sequence_number ASC
    `, versionID)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []*Event
    for rows.Next() {
        event := &Event{}
        var payloadJSON, metadataJSON []byte

        err := rows.Scan(
            &event.ID, &event.DocumentID, &event.TenantID, &event.VersionID,
            &event.SequenceNumber, &event.Type, &event.EntityType, &event.EntityID,
            &payloadJSON, &event.AuthorID, &event.Timestamp, &event.CorrelationID,
            &event.CausationID, &metadataJSON,
        )
        if err != nil {
            continue
        }

        json.Unmarshal(payloadJSON, &event.Payload)
        json.Unmarshal(metadataJSON, &event.Metadata)

        events = append(events, event)
    }

    return events, nil
}

// ReplayEvents 重放事件
func (s *EventStore) ReplayEvents(
    ctx context.Context,
    documentID string,
    fromSequence int64,
    toSequence int64,
) (*ReplayResult, error) {
    // 获取事件范围
    events, err := s.GetEvents(ctx, documentID, fromSequence, int32(toSequence-fromSequence+1))
    if err != nil {
        return nil, err
    }

    // 创建聚合根
    aggregate := NewDocumentAggregate(documentID)

    // 重放事件
    for _, event := range events {
        if err := aggregate.Apply(event); err != nil {
            return &ReplayResult{
                Success:        false,
                EventsReplayed: int64(len(events)),
                Message:        fmt.Sprintf("重放事件失败: %v", err),
            }, nil
        }
    }

    return &ReplayResult{
        Success:        true,
        EventsReplayed: int64(len(events)),
        Message:        "事件重放成功",
    }, nil
}

// publishEvent 发布事件
func (s *EventStore) publishEvent(event *Event) {
    // 发布到消息队列 (Kafka/NATS)
    // ...
}

// DocumentAggregate 文档聚合根
type DocumentAggregate struct {
    DocumentID     string
    Version        int64
    Elements       map[string]*Element
    Relationships  map[string]*Relationship
    Metadata       map[string]string
}

func NewDocumentAggregate(documentID string) *DocumentAggregate {
    return &DocumentAggregate{
        DocumentID:    documentID,
        Elements:      make(map[string]*Element),
        Relationships: make(map[string]*Relationship),
        Metadata:      make(map[string]string),
    }
}

// Apply 应用事件
func (a *DocumentAggregate) Apply(event *Event) error {
    switch event.Type {
    case "ELEMENT_CREATED":
        return a.applyElementCreated(event)
    case "ELEMENT_UPDATED":
        return a.applyElementUpdated(event)
    case "ELEMENT_DELETED":
        return a.applyElementDeleted(event)
    case "GEOMETRY_CHANGED":
        return a.applyGeometryChanged(event)
    case "PROPERTY_CHANGED":
        return a.applyPropertyChanged(event)
    default:
        return fmt.Errorf("未知事件类型: %s", event.Type)
    }
}

func (a *DocumentAggregate) applyElementCreated(event *Event) error {
    elementID := event.EntityID
    element := &Element{
        ID:   elementID,
        Type: event.Payload["element_type"].(string),
        Data: event.Payload["data"].(map[string]interface{}),
    }
    a.Elements[elementID] = element
    return nil
}

func (a *DocumentAggregate) applyElementUpdated(event *Event) error {
    elementID := event.EntityID
    element, ok := a.Elements[elementID]
    if !ok {
        return fmt.Errorf("元素不存在: %s", elementID)
    }

    // 更新元素数据
    updates := event.Payload["updates"].(map[string]interface{})
    for key, value := range updates {
        element.Data[key] = value
    }

    return nil
}

func (a *DocumentAggregate) applyElementDeleted(event *Event) error {
    elementID := event.EntityID
    delete(a.Elements, elementID)
    return nil
}

func (a *DocumentAggregate) applyGeometryChanged(event *Event) error {
    // 应用几何变更
    return nil
}

func (a *DocumentAggregate) applyPropertyChanged(event *Event) error {
    // 应用属性变更
    return nil
}
```

## 5.4 快照管理实现

```go
package version

import (
    "bytes"
    "compress/gzip"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/klauspost/compress/zstd"
)

// SnapshotManager 快照管理器
type SnapshotManager struct {
    db          *pgxpool.Pool
    storage     SnapshotStorage
    compression CompressionType
}

// SnapshotStorage 快照存储接口
type SnapshotStorage interface {
    Save(ctx context.Context, id string, data []byte) error
    Load(ctx context.Context, id string) ([]byte, error)
    Delete(ctx context.Context, id string) error
}

// CreateSnapshot 创建快照
func (m *SnapshotManager) CreateSnapshot(
    ctx context.Context,
    documentID string,
    versionID string,
    sequenceNumber int64,
    data []byte,
    compression CompressionType,
) (*Snapshot, error) {
    // 压缩数据
    compressedData, err := m.compressData(data, compression)
    if err != nil {
        return nil, fmt.Errorf("压缩数据失败: %w", err)
    }

    // 计算校验和
    checksum := m.calculateChecksum(compressedData)

    // 创建快照记录
    snapshot := &Snapshot{
        ID:             generateUUID(),
        DocumentID:     documentID,
        VersionID:      versionID,
        SequenceNumber: sequenceNumber,
        DataSize:       int64(len(data)),
        CompressedSize: int64(len(compressedData)),
        Checksum:       checksum,
        Compression:    compression,
        CreatedAt:      time.Now(),
    }

    // 保存到对象存储
    if err := m.storage.Save(ctx, snapshot.ID, compressedData); err != nil {
        return nil, fmt.Errorf("保存快照失败: %w", err)
    }

    // 持久化元数据
    _, err = m.db.Exec(ctx, `
        INSERT INTO snapshots (
            id, document_id, version_id, sequence_number,
            data_size, compressed_size, checksum, compression_type, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `,
        snapshot.ID, snapshot.DocumentID, snapshot.VersionID, snapshot.SequenceNumber,
        snapshot.DataSize, snapshot.CompressedSize, snapshot.Checksum, 
        compression.String(), snapshot.CreatedAt,
    )

    if err != nil {
        // 回滚对象存储
        m.storage.Delete(ctx, snapshot.ID)
        return nil, fmt.Errorf("保存快照元数据失败: %w", err)
    }

    return snapshot, nil
}

// GetSnapshot 获取快照
func (m *SnapshotManager) GetSnapshot(
    ctx context.Context,
    snapshotID string,
    decompress bool,
) (*Snapshot, []byte, error) {
    // 查询快照元数据
    var snapshot Snapshot
    var compressionType string

    err := m.db.QueryRow(ctx, `
        SELECT id, document_id, version_id, sequence_number,
               data_size, compressed_size, checksum, compression_type, created_at
        FROM snapshots WHERE id = $1
    `, snapshotID).Scan(
        &snapshot.ID, &snapshot.DocumentID, &snapshot.VersionID, &snapshot.SequenceNumber,
        &snapshot.DataSize, &snapshot.CompressedSize, &snapshot.Checksum,
        &compressionType, &snapshot.CreatedAt,
    )

    if err != nil {
        return nil, nil, fmt.Errorf("快照不存在: %w", err)
    }

    snapshot.Compression = parseCompressionType(compressionType)

    // 从对象存储加载
    compressedData, err := m.storage.Load(ctx, snapshotID)
    if err != nil {
        return nil, nil, fmt.Errorf("加载快照数据失败: %w", err)
    }

    // 验证校验和
    if checksum := m.calculateChecksum(compressedData); checksum != snapshot.Checksum {
        return nil, nil, fmt.Errorf("快照校验和验证失败")
    }

    if !decompress {
        return &snapshot, compressedData, nil
    }

    // 解压数据
    data, err := m.decompressData(compressedData, snapshot.Compression)
    if err != nil {
        return nil, nil, fmt.Errorf("解压数据失败: %w", err)
    }

    return &snapshot, data, nil
}

// RestoreSnapshot 恢复快照
func (m *SnapshotManager) RestoreSnapshot(
    ctx context.Context,
    snapshotID string,
    branchID string,
    authorID string,
    message string,
) (*Version, error) {
    // 获取快照
    snapshot, data, err := m.GetSnapshot(ctx, snapshotID, true)
    if err != nil {
        return nil, err
    }

    // 重放快照数据
    aggregate := NewDocumentAggregate(snapshot.DocumentID)
    if err := aggregate.LoadFromSnapshot(data); err != nil {
        return nil, fmt.Errorf("加载快照数据失败: %w", err)
    }

    // 创建新版本
    version := &Version{
        ID:             generateUUID(),
        DocumentID:     snapshot.DocumentID,
        BranchID:       branchID,
        ParentVersionID: snapshot.VersionID,
        Message:        message,
        AuthorID:       authorID,
        CreatedAt:      time.Now(),
        SequenceNumber: snapshot.SequenceNumber,
        SnapshotID:     snapshotID,
    }

    // 持久化版本
    _, err = m.db.Exec(ctx, `
        INSERT INTO versions (
            id, document_id, branch_id, parent_version_id, message,
            author_id, created_at, sequence_number, snapshot_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `,
        version.ID, version.DocumentID, version.BranchID, version.ParentVersionID,
        version.Message, version.AuthorID, version.CreatedAt,
        version.SequenceNumber, version.SnapshotID,
    )

    if err != nil {
        return nil, fmt.Errorf("创建版本失败: %w", err)
    }

    // 更新分支头指针
    _, err = m.db.Exec(ctx, `
        UPDATE branches SET head_version_id = $1, updated_at = NOW()
        WHERE id = $2
    `, version.ID, branchID)

    if err != nil {
        return nil, fmt.Errorf("更新分支失败: %w", err)
    }

    return version, nil
}

// compressData 压缩数据
func (m *SnapshotManager) compressData(data []byte, compression CompressionType) ([]byte, error) {
    switch compression {
    case CompressionType_NONE:
        return data, nil

    case CompressionType_GZIP:
        var buf bytes.Buffer
        writer := gzip.NewWriter(&buf)
        if _, err := writer.Write(data); err != nil {
            return nil, err
        }
        writer.Close()
        return buf.Bytes(), nil

    case CompressionType_ZSTD:
        encoder, err := zstd.NewWriter(nil)
        if err != nil {
            return nil, err
        }
        return encoder.EncodeAll(data, nil), nil

    default:
        return data, nil
    }
}

// decompressData 解压数据
func (m *SnapshotManager) decompressData(data []byte, compression CompressionType) ([]byte, error) {
    switch compression {
    case CompressionType_NONE:
        return data, nil

    case CompressionType_GZIP:
        reader, err := gzip.NewReader(bytes.NewReader(data))
        if err != nil {
            return nil, err
        }
        defer reader.Close()
        return io.ReadAll(reader)

    case CompressionType_ZSTD:
        decoder, err := zstd.NewReader(nil)
        if err != nil {
            return nil, err
        }
        defer decoder.Close()
        return decoder.DecodeAll(data, nil)

    default:
        return data, nil
    }
}

// calculateChecksum 计算校验和
func (m *SnapshotManager) calculateChecksum(data []byte) string {
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

// CleanupOldSnapshots 清理旧快照
func (m *SnapshotManager) CleanupOldSnapshots(
    ctx context.Context,
    documentID string,
    keepCount int,
) error {
    // 获取要删除的快照
    rows, err := m.db.Query(ctx, `
        SELECT id FROM snapshots
        WHERE document_id = $1
        ORDER BY created_at DESC
        OFFSET $2
    `, documentID, keepCount)

    if err != nil {
        return err
    }
    defer rows.Close()

    var snapshotIDs []string
    for rows.Next() {
        var id string
        if err := rows.Scan(&id); err != nil {
            continue
        }
        snapshotIDs = append(snapshotIDs, id)
    }

    // 删除快照
    for _, id := range snapshotIDs {
        // 删除对象存储
        if err := m.storage.Delete(ctx, id); err != nil {
            continue
        }

        // 删除元数据
        m.db.Exec(ctx, `DELETE FROM snapshots WHERE id = $1`, id)
    }

    return nil
}
```

## 5.5 分支管理实现

```go
package version

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// BranchManager 分支管理器
type BranchManager struct {
    db        *pgxpool.Pool
    eventStore *EventStore
}

// Branch 分支
type Branch struct {
    ID            string
    DocumentID    string
    TenantID      string
    Name          string
    Description   string
    BaseBranchID  string
    BaseVersionID string
    HeadVersionID string
    IsDefault     bool
    IsProtected   bool
    CreatedBy     string
    CreatedAt     time.Time
    UpdatedAt     time.Time
    Metadata      map[string]string
}

// CreateBranch 创建分支
func (m *BranchManager) CreateBranch(
    ctx context.Context,
    documentID string,
    tenantID string,
    name string,
    description string,
    baseBranchID string,
    baseVersionID string,
    createdBy string,
    isProtected bool,
) (*Branch, error) {
    // 检查分支名是否已存在
    var exists bool
    err := m.db.QueryRow(ctx, `
        SELECT EXISTS(
            SELECT 1 FROM branches 
            WHERE document_id = $1 AND name = $2
        )
    `, documentID, name).Scan(&exists)

    if err != nil {
        return nil, err
    }

    if exists {
        return nil, fmt.Errorf("分支 '%s' 已存在", name)
    }

    // 如果没有指定基础版本，使用分支的当前版本
    if baseVersionID == "" && baseBranchID != "" {
        err = m.db.QueryRow(ctx, `
            SELECT head_version_id FROM branches WHERE id = $1
        `, baseBranchID).Scan(&baseVersionID)

        if err != nil {
            return nil, fmt.Errorf("基础分支不存在: %w", err)
        }
    }

    branch := &Branch{
        ID:            generateUUID(),
        DocumentID:    documentID,
        TenantID:      tenantID,
        Name:          name,
        Description:   description,
        BaseBranchID:  baseBranchID,
        BaseVersionID: baseVersionID,
        HeadVersionID: baseVersionID,
        IsProtected:   isProtected,
        CreatedBy:     createdBy,
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }

    // 持久化分支
    _, err = m.db.Exec(ctx, `
        INSERT INTO branches (
            id, document_id, tenant_id, name, description,
            base_branch_id, base_version_id, head_version_id,
            is_protected, created_by, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `,
        branch.ID, branch.DocumentID, branch.TenantID, branch.Name, branch.Description,
        branch.BaseBranchID, branch.BaseVersionID, branch.HeadVersionID,
        branch.IsProtected, branch.CreatedBy, branch.CreatedAt, branch.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("创建分支失败: %w", err)
    }

    return branch, nil
}

// MergeBranch 合并分支
func (m *BranchManager) MergeBranch(
    ctx context.Context,
    documentID string,
    sourceBranchID string,
    targetBranchID string,
    authorID string,
    message string,
    strategy MergeStrategy,
) (*MergeResult, error) {
    // 获取源分支和目标分支
    sourceBranch, err := m.GetBranch(ctx, sourceBranchID)
    if err != nil {
        return nil, fmt.Errorf("源分支不存在: %w", err)
    }

    targetBranch, err := m.GetBranch(ctx, targetBranchID)
    if err != nil {
        return nil, fmt.Errorf("目标分支不存在: %w", err)
    }

    // 检查目标分支是否受保护
    if targetBranch.IsProtected {
        // 检查用户是否有权限
        hasPermission, err := m.checkMergePermission(ctx, targetBranchID, authorID)
        if err != nil || !hasPermission {
            return nil, fmt.Errorf("无权合并到受保护分支")
        }
    }

    // 获取共同祖先
    commonAncestor, err := m.findCommonAncestor(ctx, sourceBranchID, targetBranchID)
    if err != nil {
        return nil, fmt.Errorf("查找共同祖先失败: %w", err)
    }

    // 获取变更事件
    sourceEvents, err := m.eventStore.GetEvents(ctx, documentID, 
        commonAncestor.SequenceNumber+1, 10000)
    if err != nil {
        return nil, err
    }

    targetEvents, err := m.eventStore.GetEvents(ctx, documentID,
        commonAncestor.SequenceNumber+1, 10000)
    if err != nil {
        return nil, err
    }

    // 检测冲突
    conflicts := m.detectConflicts(sourceEvents, targetEvents)

    if len(conflicts) > 0 {
        return &MergeResult{
            Success:   false,
            Conflicts: conflicts,
            Message:   "检测到合并冲突",
        }, nil
    }

    // 执行合并
    mergedVersion, err := m.createMergeVersion(
        ctx, documentID, targetBranchID, sourceBranch.HeadVersionID,
        targetBranch.HeadVersionID, authorID, message,
    )

    if err != nil {
        return nil, fmt.Errorf("创建合并版本失败: %w", err)
    }

    return &MergeResult{
        Success:          true,
        MergedVersionID:  mergedVersion.ID,
        Message:          "合并成功",
    }, nil
}

// findCommonAncestor 查找共同祖先
func (m *BranchManager) findCommonAncestor(
    ctx context.Context,
    branchID1 string,
    branchID2 string,
) (*Version, error) {
    // 获取分支的所有祖先版本
    ancestors1, err := m.getBranchAncestors(ctx, branchID1)
    if err != nil {
        return nil, err
    }

    ancestors2, err := m.getBranchAncestors(ctx, branchID2)
    if err != nil {
        return nil, err
    }

    // 找到最新的共同祖先
    ancestorSet := make(map[string]*Version)
    for _, v := range ancestors1 {
        ancestorSet[v.ID] = v
    }

    for _, v := range ancestors2 {
        if _, ok := ancestorSet[v.ID]; ok {
            return v, nil
        }
    }

    return nil, fmt.Errorf("未找到共同祖先")
}

// getBranchAncestors 获取分支的所有祖先版本
func (m *BranchManager) getBranchAncestors(
    ctx context.Context,
    branchID string,
) ([]*Version, error) {
    branch, err := m.GetBranch(ctx, branchID)
    if err != nil {
        return nil, err
    }

    var ancestors []*Version
    currentVersionID := branch.HeadVersionID

    for currentVersionID != "" {
        version, err := m.getVersion(ctx, currentVersionID)
        if err != nil {
            break
        }

        ancestors = append(ancestors, version)
        currentVersionID = version.ParentVersionID
    }

    return ancestors, nil
}

// detectConflicts 检测冲突
func (m *BranchManager) detectConflicts(
    sourceEvents []*Event,
    targetEvents []*Event,
) []*MergeConflict {
    var conflicts []*MergeConflict

    // 按实体ID分组
    sourceChanges := make(map[string][]*Event)
    for _, e := range sourceEvents {
        key := fmt.Sprintf("%s:%s", e.EntityType, e.EntityID)
        sourceChanges[key] = append(sourceChanges[key], e)
    }

    targetChanges := make(map[string][]*Event)
    for _, e := range targetEvents {
        key := fmt.Sprintf("%s:%s", e.EntityType, e.EntityID)
        targetChanges[key] = append(targetChanges[key], e)
    }

    // 检测同一实体的并发修改
    for key, sourceEvts := range sourceChanges {
        if targetEvts, ok := targetChanges[key]; ok {
            // 检查是否有冲突
            if m.hasConflictingChanges(sourceEvts, targetEvts) {
                parts := strings.Split(key, ":")
                conflicts = append(conflicts, &MergeConflict{
                    Path: parts[1],
                    Type: ConflictType_CONTENT,
                })
            }
        }
    }

    return conflicts
}

// hasConflictingChanges 检查变更是否冲突
func (m *BranchManager) hasConflictingChanges(
    sourceEvents []*Event,
    targetEvents []*Event,
) bool {
    // 简化实现: 如果两个分支都修改了同一实体的同一属性，则认为是冲突
    sourceProps := make(map[string]bool)
    for _, e := range sourceEvents {
        if prop, ok := e.Payload["property"].(string); ok {
            sourceProps[prop] = true
        }
    }

    for _, e := range targetEvents {
        if prop, ok := e.Payload["property"].(string); ok {
            if sourceProps[prop] {
                return true
            }
        }
    }

    return false
}

// createMergeVersion 创建合并版本
func (m *BranchManager) createMergeVersion(
    ctx context.Context,
    documentID string,
    branchID string,
    sourceVersionID string,
    targetVersionID string,
    authorID string,
    message string,
) (*Version, error) {
    version := &Version{
        ID:              generateUUID(),
        DocumentID:      documentID,
        BranchID:        branchID,
        ParentVersionID: targetVersionID,
        Message:         message,
        AuthorID:        authorID,
        CreatedAt:       time.Now(),
    }

    // 持久化版本
    _, err := m.db.Exec(ctx, `
        INSERT INTO versions (
            id, document_id, branch_id, parent_version_id,
            message, author_id, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `,
        version.ID, version.DocumentID, version.BranchID,
        version.ParentVersionID, version.Message, version.AuthorID, version.CreatedAt,
    )

    if err != nil {
        return nil, err
    }

    // 更新分支头指针
    _, err = m.db.Exec(ctx, `
        UPDATE branches SET head_version_id = $1, updated_at = NOW()
        WHERE id = $2
    `, version.ID, branchID)

    if err != nil {
        return nil, err
    }

    return version, nil
}

// GetBranch 获取分支
func (m *BranchManager) GetBranch(ctx context.Context, branchID string) (*Branch, error) {
    var branch Branch
    err := m.db.QueryRow(ctx, `
        SELECT id, document_id, tenant_id, name, description,
               base_branch_id, base_version_id, head_version_id,
               is_default, is_protected, created_by, created_at, updated_at
        FROM branches WHERE id = $1
    `, branchID).Scan(
        &branch.ID, &branch.DocumentID, &branch.TenantID, &branch.Name, &branch.Description,
        &branch.BaseBranchID, &branch.BaseVersionID, &branch.HeadVersionID,
        &branch.IsDefault, &branch.IsProtected, &branch.CreatedBy,
        &branch.CreatedAt, &branch.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return &branch, nil
}

// checkMergePermission 检查合并权限
func (m *BranchManager) checkMergePermission(
    ctx context.Context,
    branchID string,
    userID string,
) (bool, error) {
    // 检查用户是否是分支管理员或项目管理员
    // ...
    return true, nil
}
```

---

---

# 6. 用户服务详细设计

## 6.1 服务概述

用户服务负责管理用户账户、认证授权、租户管理和权限控制，是整个系统的安全核心。

### 核心功能
- 用户账户管理
- 认证与授权 (OAuth2/JWT)
- 多租户管理
- 角色权限管理
- 审计日志

## 6.2 gRPC接口定义

```protobuf
syntax = "proto3";

package user.v1;

option go_package = "github.com/archplatform/user-service/api/v1";

import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

// 用户服务
service UserService {
  // 用户管理
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc GetUserByEmail(GetUserByEmailRequest) returns (User);
  rpc UpdateUser(UpdateUserRequest) returns (User);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc SearchUsers(SearchUsersRequest) returns (ListUsersResponse);

  // 认证
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc Logout(LogoutRequest) returns (google.protobuf.Empty);
  rpc RefreshToken(RefreshTokenRequest) returns (LoginResponse);
  rpc VerifyToken(VerifyTokenRequest) returns (VerifyTokenResponse);
  rpc ChangePassword(ChangePasswordRequest) returns (google.protobuf.Empty);
  rpc ResetPassword(ResetPasswordRequest) returns (google.protobuf.Empty);
  rpc VerifyEmail(VerifyEmailRequest) returns (google.protobuf.Empty);
  rpc ResendVerification(ResendVerificationRequest) returns (google.protobuf.Empty);

  // OAuth
  rpc OAuthLogin(OAuthLoginRequest) returns (OAuthLoginResponse);
  rpc OAuthCallback(OAuthCallbackRequest) returns (LoginResponse);
  rpc LinkOAuthAccount(LinkOAuthAccountRequest) returns (google.protobuf.Empty);
  rpc UnlinkOAuthAccount(UnlinkOAuthAccountRequest) returns (google.protobuf.Empty);

  // 租户管理
  rpc CreateTenant(CreateTenantRequest) returns (Tenant);
  rpc GetTenant(GetTenantRequest) returns (Tenant);
  rpc UpdateTenant(UpdateTenantRequest) returns (Tenant);
  rpc DeleteTenant(DeleteTenantRequest) returns (google.protobuf.Empty);
  rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
  rpc SwitchTenant(SwitchTenantRequest) returns (LoginResponse);
  rpc InviteUserToTenant(InviteUserToTenantRequest) returns (Invitation);
  rpc AcceptInvitation(AcceptInvitationRequest) returns (google.protobuf.Empty);

  // 角色权限
  rpc CreateRole(CreateRoleRequest) returns (Role);
  rpc GetRole(GetRoleRequest) returns (Role);
  rpc UpdateRole(UpdateRoleRequest) returns (Role);
  rpc DeleteRole(DeleteRoleRequest) returns (google.protobuf.Empty);
  rpc ListRoles(ListRolesRequest) returns (ListRolesResponse);
  rpc AssignRole(AssignRoleRequest) returns (google.protobuf.Empty);
  rpc RevokeRole(RevokeRoleRequest) returns (google.protobuf.Empty);
  rpc GetUserRoles(GetUserRolesRequest) returns (UserRolesResponse);
  rpc CheckPermission(CheckPermissionRequest) returns (PermissionCheckResponse);
  rpc GetPermissions(GetPermissionsRequest) returns (PermissionsResponse);

  // API密钥
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (APIKey);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);

  // 审计日志
  rpc GetAuditLogs(GetAuditLogsRequest) returns (AuditLogResponse);
  rpc GetUserActivity(GetUserActivityRequest) returns (UserActivityResponse);
}

// ==================== 用户消息 ====================

message User {
  string id = 1;
  string email = 2;
  string username = 3;
  string display_name = 4;
  string avatar_url = 5;
  UserStatus status = 6;
  bool email_verified = 7;
  string timezone = 8;
  string locale = 9;
  map<string, string> metadata = 10;
  repeated string tenant_ids = 11;
  string current_tenant_id = 12;
  int64 created_at = 13;
  int64 updated_at = 14;
  int64 last_login_at = 15;
}

message UserProfile {
  string user_id = 1;
  string bio = 2;
  string company = 3;
  string job_title = 4;
  string phone = 5;
  string website = 6;
  map<string, string> social_links = 7;
  map<string, string> preferences = 8;
}

// ==================== 认证消息 ====================

message LoginRequest {
  string email = 1;
  string password = 2;
  string tenant_id = 3;
  string device_id = 4;
  string ip_address = 5;
  string user_agent = 6;
}

message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  string token_type = 3;
  int64 expires_in = 4;
  User user = 5;
  repeated Tenant tenants = 6;
}

message LogoutRequest {
  string refresh_token = 1;
  bool all_devices = 2;
}

message RefreshTokenRequest {
  string refresh_token = 1;
}

message VerifyTokenRequest {
  string token = 1;
}

message VerifyTokenResponse {
  bool valid = 1;
  string user_id = 2;
  string tenant_id = 3;
  repeated string permissions = 4;
  int64 expires_at = 5;
}

message ChangePasswordRequest {
  string user_id = 1;
  string old_password = 2;
  string new_password = 3;
}

message ResetPasswordRequest {
  string email = 1;
}

message VerifyEmailRequest {
  string token = 1;
}

message ResendVerificationRequest {
  string email = 1;
}

// ==================== OAuth消息 ====================

message OAuthLoginRequest {
  OAuthProvider provider = 1;
  string redirect_url = 2;
  string state = 3;
}

message OAuthLoginResponse {
  string auth_url = 1;
  string state = 2;
}

message OAuthCallbackRequest {
  OAuthProvider provider = 1;
  string code = 2;
  string state = 3;
}

message LinkOAuthAccountRequest {
  string user_id = 1;
  OAuthProvider provider = 2;
  string provider_user_id = 3;
  map<string, string> provider_data = 4;
}

message UnlinkOAuthAccountRequest {
  string user_id = 1;
  OAuthProvider provider = 2;
}

message OAuthAccount {
  OAuthProvider provider = 1;
  string provider_user_id = 2;
  string email = 3;
  string username = 4;
  string avatar_url = 5;
  int64 linked_at = 6;
}

// ==================== 租户消息 ====================

message Tenant {
  string id = 1;
  string name = 2;
  string slug = 3;
  string description = 4;
  TenantStatus status = 5;
  TenantPlan plan = 6;
  string owner_id = 7;
  map<string, string> settings = 8;
  TenantLimits limits = 9;
  int64 created_at = 10;
  int64 updated_at = 11;
  int64 expires_at = 12;
}

message TenantLimits {
  int32 max_users = 1;
  int32 max_projects = 2;
  int64 max_storage_gb = 3;
  int32 max_api_calls_per_day = 4;
}

message Invitation {
  string id = 1;
  string tenant_id = 2;
  string email = 3;
  string role_id = 4;
  string invited_by = 5;
  int64 invited_at = 6;
  int64 expires_at = 7;
  string token = 8;
  InvitationStatus status = 9;
}

// ==================== 角色权限消息 ====================

message Role {
  string id = 1;
  string tenant_id = 2;
  string name = 3;
  string description = 4;
  RoleType type = 5;
  repeated string permissions = 6;
  map<string, string> metadata = 7;
  int64 created_at = 8;
  int64 updated_at = 9;
}

message Permission {
  string id = 1;
  string resource = 2;
  string action = 3;
  string description = 4;
  PermissionScope scope = 5;
}

message PermissionCheckRequest {
  string user_id = 1;
  string tenant_id = 2;
  string resource = 3;
  string action = 4;
  map<string, string> context = 5;
}

message PermissionCheckResponse {
  bool allowed = 1;
  repeated string required_permissions = 2;
  string reason = 3;
}

// ==================== API密钥消息 ====================

message APIKey {
  string id = 1;
  string tenant_id = 2;
  string user_id = 3;
  string name = 4;
  string key_preview = 5;
  repeated string permissions = 6;
  int64 created_at = 7;
  int64 expires_at = 8;
  int64 last_used_at = 9;
  string created_by = 10;
}

// ==================== 审计日志消息 ====================

message AuditLog {
  string id = 1;
  string tenant_id = 2;
  string user_id = 3;
  string action = 4;
  string resource_type = 5;
  string resource_id = 6;
  google.protobuf.Struct details = 7;
  string ip_address = 8;
  string user_agent = 9;
  int64 timestamp = 10;
  ActionResult result = 11;
  string error_message = 12;
}

message UserActivity {
  string user_id = 1;
  string action = 2;
  string resource_type = 3;
  string resource_id = 4;
  int64 timestamp = 5;
  string ip_address = 6;
}

// ==================== 请求/响应消息 ====================

message CreateUserRequest {
  string email = 1;
  string username = 2;
  string password = 3;
  string display_name = 4;
  map<string, string> metadata = 5;
  string tenant_id = 6;
  string invited_by = 7;
}

message GetUserRequest {
  string id = 1;
}

message GetUserByEmailRequest {
  string email = 1;
}

message UpdateUserRequest {
  string id = 1;
  string display_name = 2;
  string avatar_url = 3;
  string timezone = 4;
  string locale = 5;
  map<string, string> metadata = 6;
  UserProfile profile = 7;
}

message DeleteUserRequest {
  string id = 1;
  bool permanent = 2;
}

message ListUsersRequest {
  string tenant_id = 1;
  UserStatus status = 2;
  int32 page_size = 3;
  string page_token = 4;
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message SearchUsersRequest {
  string tenant_id = 1;
  string query = 2;
  int32 page_size = 3;
  string page_token = 4;
}

// ==================== 租户请求消息 ====================

message CreateTenantRequest {
  string name = 1;
  string slug = 2;
  string description = 3;
  string owner_id = 4;
  TenantPlan plan = 5;
  map<string, string> settings = 6;
}

message GetTenantRequest {
  string id = 1;
}

message UpdateTenantRequest {
  string id = 1;
  string name = 2;
  string description = 3;
  map<string, string> settings = 4;
  TenantLimits limits = 5;
}

message DeleteTenantRequest {
  string id = 1;
  bool force = 2;
}

message ListTenantsRequest {
  string user_id = 1;
  TenantStatus status = 2;
}

message ListTenantsResponse {
  repeated Tenant tenants = 1;
}

message SwitchTenantRequest {
  string user_id = 1;
  string tenant_id = 2;
}

message InviteUserToTenantRequest {
  string tenant_id = 1;
  string email = 2;
  string role_id = 3;
  string invited_by = 4;
  int64 expires_in_hours = 5;
}

message AcceptInvitationRequest {
  string token = 1;
  string password = 2;
}

// ==================== 角色请求消息 ====================

message CreateRoleRequest {
  string tenant_id = 1;
  string name = 2;
  string description = 3;
  RoleType type = 4;
  repeated string permissions = 5;
  map<string, string> metadata = 6;
}

message GetRoleRequest {
  string id = 1;
  string tenant_id = 2;
}

message UpdateRoleRequest {
  string id = 1;
  string tenant_id = 2;
  string name = 3;
  string description = 4;
  repeated string permissions = 5;
  map<string, string> metadata = 6;
}

message DeleteRoleRequest {
  string id = 1;
  string tenant_id = 2;
}

message ListRolesRequest {
  string tenant_id = 1;
  RoleType type = 2;
}

message ListRolesResponse {
  repeated Role roles = 1;
}

message AssignRoleRequest {
  string user_id = 1;
  string tenant_id = 2;
  string role_id = 3;
  string assigned_by = 4;
}

message RevokeRoleRequest {
  string user_id = 1;
  string tenant_id = 2;
  string role_id = 3;
}

message GetUserRolesRequest {
  string user_id = 1;
  string tenant_id = 2;
}

message UserRolesResponse {
  repeated Role roles = 1;
}

message GetPermissionsRequest {
  string tenant_id = 1;
  string resource = 2;
}

message PermissionsResponse {
  repeated Permission permissions = 1;
}

// ==================== API密钥请求消息 ====================

message CreateAPIKeyRequest {
  string tenant_id = 1;
  string user_id = 2;
  string name = 3;
  repeated string permissions = 4;
  int64 expires_in_days = 5;
}

message ListAPIKeysRequest {
  string tenant_id = 1;
  string user_id = 2;
}

message ListAPIKeysResponse {
  repeated APIKey api_keys = 1;
}

message RevokeAPIKeyRequest {
  string id = 1;
  string tenant_id = 2;
}

// ==================== 审计日志请求消息 ====================

message GetAuditLogsRequest {
  string tenant_id = 1;
  string user_id = 2;
  string action = 3;
  string resource_type = 4;
  int64 from_time = 5;
  int64 to_time = 6;
  int32 page_size = 7;
  string page_token = 8;
}

message AuditLogResponse {
  repeated AuditLog logs = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message GetUserActivityRequest {
  string user_id = 1;
  int64 from_time = 2;
  int64 to_time = 3;
  int32 page_size = 4;
  string page_token = 5;
}

message UserActivityResponse {
  repeated UserActivity activities = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ==================== 枚举定义 ====================

enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;
  ACTIVE = 1;
  INACTIVE = 2;
  SUSPENDED = 3;
  PENDING_VERIFICATION = 4;
  DELETED = 5;
}

enum TenantStatus {
  TENANT_STATUS_UNSPECIFIED = 0;
  ACTIVE = 1;
  INACTIVE = 2;
  SUSPENDED = 3;
  PENDING_PAYMENT = 4;
  EXPIRED = 5;
}

enum TenantPlan {
  TENANT_PLAN_UNSPECIFIED = 0;
  FREE = 1;
  STARTER = 2;
  PROFESSIONAL = 3;
  ENTERPRISE = 4;
}

enum RoleType {
  ROLE_TYPE_UNSPECIFIED = 0;
  SYSTEM = 1;
  TENANT = 2;
  CUSTOM = 3;
}

enum PermissionScope {
  PERMISSION_SCOPE_UNSPECIFIED = 0;
  GLOBAL = 1;
  TENANT = 2;
  PROJECT = 3;
  RESOURCE = 4;
}

enum OAuthProvider {
  OAUTH_PROVIDER_UNSPECIFIED = 0;
  GOOGLE = 1;
  GITHUB = 2;
  MICROSOFT = 3;
  APPLE = 4;
  SAML = 5;
}

enum InvitationStatus {
  INVITATION_STATUS_UNSPECIFIED = 0;
  PENDING = 1;
  ACCEPTED = 2;
  EXPIRED = 3;
  REVOKED = 4;
}

enum ActionResult {
  ACTION_RESULT_UNSPECIFIED = 0;
  SUCCESS = 1;
  FAILURE = 2;
  DENIED = 3;
  ERROR = 4;
}
```

## 6.3 数据库表结构设计

```sql
-- ==================== 用户表 ====================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(256) UNIQUE NOT NULL,
    username VARCHAR(128) UNIQUE,
    password_hash VARCHAR(256),
    display_name VARCHAR(256),
    avatar_url TEXT,
    status VARCHAR(32) DEFAULT 'PENDING_VERIFICATION',
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(256),
    email_verification_expires_at TIMESTAMP WITH TIME ZONE,
    password_reset_token VARCHAR(256),
    password_reset_expires_at TIMESTAMP WITH TIME ZONE,
    timezone VARCHAR(64) DEFAULT 'UTC',
    locale VARCHAR(16) DEFAULT 'en',
    metadata JSONB DEFAULT '{}',
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    last_login_ip INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT chk_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED', 'PENDING_VERIFICATION', 'DELETED'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_verification ON users(email_verification_token) WHERE email_verification_token IS NOT NULL;

-- ==================== 用户资料表 ====================
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio TEXT,
    company VARCHAR(256),
    job_title VARCHAR(128),
    phone VARCHAR(32),
    website VARCHAR(256),
    social_links JSONB DEFAULT '{}',
    preferences JSONB DEFAULT '{}',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ==================== 租户表 ====================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(256) NOT NULL,
    slug VARCHAR(128) UNIQUE NOT NULL,
    description TEXT,
    status VARCHAR(32) DEFAULT 'ACTIVE',
    plan VARCHAR(32) DEFAULT 'FREE',
    owner_id UUID NOT NULL REFERENCES users(id),
    settings JSONB DEFAULT '{}',
    limits JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT chk_tenant_status CHECK (status IN ('ACTIVE', 'INACTIVE', 'SUSPENDED', 'PENDING_PAYMENT', 'EXPIRED')),
    CONSTRAINT chk_plan CHECK (plan IN ('FREE', 'STARTER', 'PROFESSIONAL', 'ENTERPRISE'))
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_owner ON tenants(owner_id);

-- ==================== 租户成员表 ====================
CREATE TABLE tenant_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    invited_by UUID REFERENCES users(id),
    is_active BOOLEAN DEFAULT TRUE,

    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_tenant_members_tenant ON tenant_members(tenant_id);
CREATE INDEX idx_tenant_members_user ON tenant_members(user_id);

-- ==================== OAuth账户表 ====================
CREATE TABLE oauth_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(256) NOT NULL,
    email VARCHAR(256),
    username VARCHAR(128),
    avatar_url TEXT,
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMP WITH TIME ZONE,
    provider_data JSONB DEFAULT '{}',
    linked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(provider, provider_user_id),
    CONSTRAINT chk_provider CHECK (provider IN ('GOOGLE', 'GITHUB', 'MICROSOFT', 'APPLE', 'SAML'))
);

CREATE INDEX idx_oauth_user ON oauth_accounts(user_id);

-- ==================== 角色表 ====================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    role_type VARCHAR(32) DEFAULT 'CUSTOM',
    permissions TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_system BOOLEAN DEFAULT FALSE,

    CONSTRAINT chk_role_type CHECK (role_type IN ('SYSTEM', 'TENANT', 'CUSTOM')),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);

-- ==================== 用户角色表 ====================
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,

    UNIQUE(user_id, tenant_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_tenant ON user_roles(tenant_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);

-- ==================== 权限定义表 ====================
CREATE TABLE permissions (
    id VARCHAR(128) PRIMARY KEY,
    resource VARCHAR(128) NOT NULL,
    action VARCHAR(64) NOT NULL,
    description TEXT,
    scope VARCHAR(32) DEFAULT 'TENANT',

    CONSTRAINT chk_scope CHECK (scope IN ('GLOBAL', 'TENANT', 'PROJECT', 'RESOURCE')),
    UNIQUE(resource, action)
);

-- ==================== 邀请表 ====================
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(256) NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id),
    invited_by UUID NOT NULL REFERENCES users(id),
    invited_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    token VARCHAR(256) UNIQUE NOT NULL,
    status VARCHAR(32) DEFAULT 'PENDING',
    accepted_at TIMESTAMP WITH TIME ZONE,
    accepted_by UUID REFERENCES users(id),

    CONSTRAINT chk_invitation_status CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED', 'REVOKED'))
);

CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_email ON invitations(email);

-- ==================== API密钥表 ====================
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    key_hash VARCHAR(256) NOT NULL,
    key_preview VARCHAR(16) NOT NULL,
    permissions TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL REFERENCES users(id),

    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);

-- ==================== 刷新令牌表 ====================
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash VARCHAR(256) UNIQUE NOT NULL,
    device_id VARCHAR(128),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token_hash);

-- ==================== 审计日志表 (按时间分区) ====================
CREATE TABLE audit_logs (
    id BIGSERIAL,
    tenant_id UUID NOT NULL,
    user_id UUID,
    action VARCHAR(128) NOT NULL,
    resource_type VARCHAR(128),
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    result VARCHAR(32) DEFAULT 'SUCCESS',
    error_message TEXT,

    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- 创建月度分区
CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE audit_logs_2024_02 PARTITION OF audit_logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- ==================== 系统权限初始化数据 ====================
INSERT INTO permissions (id, resource, action, description, scope) VALUES
-- 用户管理
('user:create', 'user', 'create', '创建用户', 'TENANT'),
('user:read', 'user', 'read', '读取用户信息', 'TENANT'),
('user:update', 'user', 'update', '更新用户信息', 'TENANT'),
('user:delete', 'user', 'delete', '删除用户', 'TENANT'),
-- 项目管理
('project:create', 'project', 'create', '创建项目', 'TENANT'),
('project:read', 'project', 'read', '读取项目信息', 'TENANT'),
('project:update', 'project', 'update', '更新项目信息', 'TENANT'),
('project:delete', 'project', 'delete', '删除项目', 'TENANT'),
-- 文档管理
('document:create', 'document', 'create', '创建文档', 'PROJECT'),
('document:read', 'document', 'read', '读取文档', 'PROJECT'),
('document:update', 'document', 'update', '更新文档', 'PROJECT'),
('document:delete', 'document', 'delete', '删除文档', 'PROJECT'),
-- 协作
('collaboration:join', 'collaboration', 'join', '加入协作', 'PROJECT'),
('collaboration:edit', 'collaboration', 'edit', '编辑协作', 'PROJECT'),
('collaboration:admin', 'collaboration', 'admin', '管理协作', 'PROJECT'),
-- 脚本
('script:execute', 'script', 'execute', '执行脚本', 'TENANT'),
('script:create', 'script', 'create', '创建脚本', 'TENANT'),
('script:admin', 'script', 'admin', '管理脚本', 'TENANT'),
-- 租户管理
('tenant:admin', 'tenant', 'admin', '租户管理', 'TENANT'),
('tenant:billing', 'tenant', 'billing', '账单管理', 'TENANT');

-- ==================== 系统角色初始化数据 ====================
INSERT INTO roles (id, name, description, role_type, permissions, is_system) VALUES
('role:admin', 'Administrator', '系统管理员', 'SYSTEM', 
 ARRAY['user:create', 'user:read', 'user:update', 'user:delete',
       'project:create', 'project:read', 'project:update', 'project:delete',
       'document:create', 'document:read', 'document:update', 'document:delete',
       'collaboration:join', 'collaboration:edit', 'collaboration:admin',
       'script:execute', 'script:create', 'script:admin',
       'tenant:admin', 'tenant:billing'], TRUE),
('role:editor', 'Editor', '编辑者', 'SYSTEM',
 ARRAY['user:read', 'project:read', 'document:create', 'document:read', 
       'document:update', 'collaboration:join', 'collaboration:edit',
       'script:execute'], TRUE),
('role:viewer', 'Viewer', '查看者', 'SYSTEM',
 ARRAY['user:read', 'project:read', 'document:read', 'collaboration:join'], TRUE);
```

## 6.4 认证授权实现

```go
package auth

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
type AuthService struct {
    db            *pgxpool.Pool
    jwtSecret     []byte
    tokenExpiry   time.Duration
    refreshExpiry time.Duration
}

// TokenClaims JWT声明
type TokenClaims struct {
    UserID   string   `json:"user_id"`
    Email    string   `json:"email"`
    TenantID string   `json:"tenant_id"`
    Roles    []string `json:"roles"`
    jwt.RegisteredClaims
}

// Login 用户登录
func (s *AuthService) Login(
    ctx context.Context,
    email string,
    password string,
    tenantID string,
    deviceInfo *DeviceInfo,
) (*LoginResponse, error) {
    // 获取用户
    user, err := s.getUserByEmail(ctx, email)
    if err != nil {
        return nil, fmt.Errorf("用户不存在")
    }

    // 检查账户状态
    if user.Status != "ACTIVE" {
        return nil, fmt.Errorf("账户未激活或已被禁用")
    }

    // 检查是否被锁定
    if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
        return nil, fmt.Errorf("账户已锁定，请稍后重试")
    }

    // 验证密码
    if err := bcrypt.CompareHashAndPassword(
        []byte(user.PasswordHash), 
        []byte(password),
    ); err != nil {
        // 增加失败次数
        s.incrementFailedAttempts(ctx, user.ID)
        return nil, fmt.Errorf("密码错误")
    }

    // 重置失败次数
    s.resetFailedAttempts(ctx, user.ID)

    // 更新最后登录时间
    s.updateLastLogin(ctx, user.ID, deviceInfo)

    // 生成令牌
    accessToken, err := s.generateAccessToken(user, tenantID)
    if err != nil {
        return nil, fmt.Errorf("生成访问令牌失败: %w", err)
    }

    refreshToken, err := s.generateRefreshToken(user.ID, tenantID, deviceInfo)
    if err != nil {
        return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
    }

    // 获取用户租户列表
    tenants, err := s.getUserTenants(ctx, user.ID)
    if err != nil {
        return nil, err
    }

    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    int64(s.tokenExpiry.Seconds()),
        User:         user,
        Tenants:      tenants,
    }, nil
}

// generateAccessToken 生成访问令牌
func (s *AuthService) generateAccessToken(
    user *User,
    tenantID string,
) (string, error) {
    // 获取用户角色和权限
    roles, err := s.getUserRoles(ctx, user.ID, tenantID)
    if err != nil {
        return "", err
    }

    claims := TokenClaims{
        UserID:   user.ID,
        Email:    user.Email,
        TenantID: tenantID,
        Roles:    roles,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "archplatform",
            Subject:   user.ID,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

// generateRefreshToken 生成刷新令牌
func (s *AuthService) generateRefreshToken(
    userID string,
    tenantID string,
    deviceInfo *DeviceInfo,
) (string, error) {
    // 生成随机令牌
    tokenBytes := make([]byte, 32)
    if _, err := rand.Read(tokenBytes); err != nil {
        return "", err
    }
    token := base64.URLEncoding.EncodeToString(tokenBytes)

    // 计算哈希
    tokenHash := hashToken(token)

    // 保存到数据库
    _, err := s.db.Exec(ctx, `
        INSERT INTO refresh_tokens (
            user_id, tenant_id, token_hash, device_id, 
            ip_address, user_agent, expires_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `,
        userID, tenantID, tokenHash, deviceInfo.DeviceID,
        deviceInfo.IPAddress, deviceInfo.UserAgent,
        time.Now().Add(s.refreshExpiry),
    )

    if err != nil {
        return "", err
    }

    return token, nil
}

// VerifyToken 验证令牌
func (s *AuthService) VerifyToken(tokenString string) (*TokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{},
        func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return s.jwtSecret, nil
        },
    )

    if err != nil {
        return nil, fmt.Errorf("无效的令牌: %w", err)
    }

    if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("无效的令牌声明")
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(refreshToken string) (*LoginResponse, error) {
    // 验证刷新令牌
    tokenHash := hashToken(refreshToken)

    var userID, tenantID string
    err := s.db.QueryRow(ctx, `
        SELECT user_id, tenant_id FROM refresh_tokens
        WHERE token_hash = $1 AND expires_at > NOW() AND revoked_at IS NULL
    `, tokenHash).Scan(&userID, &tenantID)

    if err != nil {
        return nil, fmt.Errorf("无效的刷新令牌")
    }

    // 获取用户信息
    user, err := s.getUserByID(ctx, userID)
    if err != nil {
        return nil, err
    }

    // 生成新令牌
    accessToken, err := s.generateAccessToken(user, tenantID)
    if err != nil {
        return nil, err
    }

    newRefreshToken, err := s.generateRefreshToken(userID, tenantID, nil)
    if err != nil {
        return nil, err
    }

    // 撤销旧刷新令牌
    s.db.Exec(ctx, `
        UPDATE refresh_tokens SET revoked_at = NOW()
        WHERE token_hash = $1
    `, tokenHash)

    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: newRefreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    int64(s.tokenExpiry.Seconds()),
        User:         user,
    }, nil
}

// CheckPermission 检查权限
func (s *AuthService) CheckPermission(
    ctx context.Context,
    userID string,
    tenantID string,
    resource string,
    action string,
) (bool, error) {
    // 获取用户在该租户的所有权限
    var permissions []string

    rows, err := s.db.Query(ctx, `
        SELECT DISTINCT UNNEST(r.permissions)
        FROM user_roles ur
        JOIN roles r ON ur.role_id = r.id
        WHERE ur.user_id = $1 AND ur.tenant_id = $2
    `, userID, tenantID)

    if err != nil {
        return false, err
    }
    defer rows.Close()

    for rows.Next() {
        var perm string
        if err := rows.Scan(&perm); err != nil {
            continue
        }
        permissions = append(permissions, perm)
    }

    // 检查是否有权限
    requiredPerm := fmt.Sprintf("%s:%s", resource, action)
    for _, perm := range permissions {
        if perm == requiredPerm || perm == "*:*" {
            return true, nil
        }
    }

    return false, nil
}

// hashToken 哈希令牌
func hashToken(token string) string {
    hash, _ := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
    return string(hash)
}

// DeviceInfo 设备信息
type DeviceInfo struct {
    DeviceID  string
    IPAddress string
    UserAgent string
}
```

## 6.5 租户管理实现

```go
package tenant

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// TenantManager 租户管理器
type TenantManager struct {
    db *pgxpool.Pool
}

// CreateTenant 创建租户
func (m *TenantManager) CreateTenant(
    ctx context.Context,
    name string,
    slug string,
    description string,
    ownerID string,
    plan TenantPlan,
    settings map[string]string,
) (*Tenant, error) {
    // 验证slug格式
    slug = strings.ToLower(strings.TrimSpace(slug))
    if !isValidSlug(slug) {
        return nil, fmt.Errorf("无效的租户标识")
    }

    // 检查slug是否已存在
    var exists bool
    err := m.db.QueryRow(ctx, `
        SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)
    `, slug).Scan(&exists)

    if err != nil {
        return nil, err
    }

    if exists {
        return nil, fmt.Errorf("租户标识 '%s' 已存在", slug)
    }

    // 设置默认限制
    limits := getDefaultLimits(plan)

    tenant := &Tenant{
        ID:          generateUUID(),
        Name:        name,
        Slug:        slug,
        Description: description,
        Status:      "ACTIVE",
        Plan:        plan,
        OwnerID:     ownerID,
        Settings:    settings,
        Limits:      limits,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    // 持久化租户
    _, err = m.db.Exec(ctx, `
        INSERT INTO tenants (
            id, name, slug, description, status, plan,
            owner_id, settings, limits, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `,
        tenant.ID, tenant.Name, tenant.Slug, tenant.Description,
        tenant.Status, tenant.Plan, tenant.OwnerID,
        tenant.Settings, tenant.Limits, tenant.CreatedAt, tenant.UpdatedAt,
    )

    if err != nil {
        return nil, fmt.Errorf("创建租户失败: %w", err)
    }

    // 添加所有者到租户
    _, err = m.db.Exec(ctx, `
        INSERT INTO tenant_members (tenant_id, user_id, joined_at, is_active)
        VALUES ($1, $2, NOW(), TRUE)
    `, tenant.ID, ownerID)

    if err != nil {
        return nil, fmt.Errorf("添加租户成员失败: %w", err)
    }

    // 为所有者分配管理员角色
    _, err = m.db.Exec(ctx, `
        INSERT INTO user_roles (user_id, tenant_id, role_id, assigned_by, assigned_at)
        VALUES ($1, $2, 'role:admin', $1, NOW())
    `, ownerID, tenant.ID)

    if err != nil {
        return nil, fmt.Errorf("分配角色失败: %w", err)
    }

    return tenant, nil
}

// InviteUser 邀请用户
func (m *TenantManager) InviteUser(
    ctx context.Context,
    tenantID string,
    email string,
    roleID string,
    invitedBy string,
    expiresInHours int64,
) (*Invitation, error) {
    // 检查邀请者权限
    hasPermission, err := m.checkInvitePermission(ctx, tenantID, invitedBy)
    if err != nil || !hasPermission {
        return nil, fmt.Errorf("无权邀请用户")
    }

    // 检查租户成员数限制
    currentMembers, err := m.getMemberCount(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    limits, err := m.getTenantLimits(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    if currentMembers >= limits.MaxUsers {
        return nil, fmt.Errorf("租户成员数已达上限")
    }

    // 生成邀请令牌
    token := generateSecureToken(32)

    invitation := &Invitation{
        ID:        generateUUID(),
        TenantID:  tenantID,
        Email:     email,
        RoleID:    roleID,
        InvitedBy: invitedBy,
        InvitedAt: time.Now(),
        ExpiresAt: time.Now().Add(time.Duration(expiresInHours) * time.Hour),
        Token:     token,
        Status:    "PENDING",
    }

    // 持久化邀请
    _, err = m.db.Exec(ctx, `
        INSERT INTO invitations (
            id, tenant_id, email, role_id, invited_by,
            invited_at, expires_at, token, status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `,
        invitation.ID, invitation.TenantID, invitation.Email,
        invitation.RoleID, invitation.InvitedBy, invitation.InvitedAt,
        invitation.ExpiresAt, invitation.Token, invitation.Status,
    )

    if err != nil {
        return nil, fmt.Errorf("创建邀请失败: %w", err)
    }

    // 发送邀请邮件
    go m.sendInvitationEmail(invitation)

    return invitation, nil
}

// AcceptInvitation 接受邀请
func (m *TenantManager) AcceptInvitation(
    ctx context.Context,
    token string,
    password string,
) error {
    // 验证邀请
    invitation, err := m.getInvitationByToken(ctx, token)
    if err != nil {
        return fmt.Errorf("无效的邀请")
    }

    if invitation.Status != "PENDING" {
        return fmt.Errorf("邀请已失效")
    }

    if invitation.ExpiresAt.Before(time.Now()) {
        m.updateInvitationStatus(ctx, invitation.ID, "EXPIRED")
        return fmt.Errorf("邀请已过期")
    }

    // 创建用户或获取现有用户
    var userID string
    existingUser, err := m.getUserByEmail(ctx, invitation.Email)
    if err != nil {
        // 创建新用户
        userID, err = m.createUser(ctx, invitation.Email, password)
        if err != nil {
            return fmt.Errorf("创建用户失败: %w", err)
        }
    } else {
        userID = existingUser.ID
    }

    // 添加用户到租户
    _, err = m.db.Exec(ctx, `
        INSERT INTO tenant_members (tenant_id, user_id, joined_at, invited_by, is_active)
        VALUES ($1, $2, NOW(), $3, TRUE)
        ON CONFLICT (tenant_id, user_id) DO UPDATE SET is_active = TRUE
    `, invitation.TenantID, userID, invitation.InvitedBy)

    if err != nil {
        return fmt.Errorf("添加租户成员失败: %w", err)
    }

    // 分配角色
    _, err = m.db.Exec(ctx, `
        INSERT INTO user_roles (user_id, tenant_id, role_id, assigned_by, assigned_at)
        VALUES ($1, $2, $3, $4, NOW())
    `, userID, invitation.TenantID, invitation.RoleID, invitation.InvitedBy)

    if err != nil {
        return fmt.Errorf("分配角色失败: %w", err)
    }

    // 更新邀请状态
    m.updateInvitationStatus(ctx, invitation.ID, "ACCEPTED")

    return nil
}

// getDefaultLimits 获取默认限制
func getDefaultLimits(plan TenantPlan) *TenantLimits {
    switch plan {
    case TenantPlan_FREE:
        return &TenantLimits{
            MaxUsers:          3,
            MaxProjects:       5,
            MaxStorageGB:      1,
            MaxAPICallsPerDay: 1000,
        }
    case TenantPlan_STARTER:
        return &TenantLimits{
            MaxUsers:          10,
            MaxProjects:       20,
            MaxStorageGB:      10,
            MaxAPICallsPerDay: 10000,
        }
    case TenantPlan_PROFESSIONAL:
        return &TenantLimits{
            MaxUsers:          50,
            MaxProjects:       100,
            MaxStorageGB:      100,
            MaxAPICallsPerDay: 100000,
        }
    case TenantPlan_ENTERPRISE:
        return &TenantLimits{
            MaxUsers:          -1, // 无限制
            MaxProjects:       -1,
            MaxStorageGB:      -1,
            MaxAPICallsPerDay: -1,
        }
    default:
        return getDefaultLimits(TenantPlan_FREE)
    }
}

// isValidSlug 验证slug格式
func isValidSlug(slug string) bool {
    if len(slug) < 3 || len(slug) > 63 {
        return false
    }

    // 只允许小写字母、数字和连字符
    for _, c := range slug {
        if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
            return false
        }
    }

    return true
}
```

---

---

# 7. 服务间集成设计

## 7.1 服务调用链设计

### 7.1.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Gateway                                      │
│                    (Kong / Ambassador / Istio Ingress)                       │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Service Mesh (Istio)                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │   User   │  │ Document │  │Geometry  │  │ Property │  │ Version  │      │
│  │ Service  │  │ Service  │  │ Service  │  │ Service  │  │ Service  │      │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘      │
│       │             │             │             │             │             │
│       └─────────────┴─────────────┴─────────────┴─────────────┘             │
│                              gRPC / HTTP2                                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Message Queue (Kafka/NATS)                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ collaboration│  │  geometry    │  │   property   │  │   version    │     │
│  │   events     │  │   events     │  │   events     │  │   events     │     │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Data Layer                                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │PostgreSQL│  │  Redis   │  │  MinIO   │  │Elasticsearch│  │ ClickHouse │  │
│  │(Primary) │  │ (Cache)  │  │ (Object) │  │  (Search)   │  │  (Analytics)│  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.1.2 服务间通信模式

```go
package integration

import (
    "context"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/keepalive"
)

// ServiceClient 服务客户端管理
type ServiceClient struct {
    connections map[string]*grpc.ClientConn
    mu          sync.RWMutex
}

// GetClient 获取服务客户端
func (sc *ServiceClient) GetClient(serviceName string) (*grpc.ClientConn, error) {
    sc.mu.RLock()
    if conn, ok := sc.connections[serviceName]; ok {
        sc.mu.RUnlock()
        return conn, nil
    }
    sc.mu.RUnlock()

    // 创建新连接
    conn, err := sc.createConnection(serviceName)
    if err != nil {
        return nil, err
    }

    sc.mu.Lock()
    sc.connections[serviceName] = conn
    sc.mu.Unlock()

    return conn, nil
}

// createConnection 创建gRPC连接
func (sc *ServiceClient) createConnection(serviceName string) (*grpc.ClientConn, error) {
    // 使用服务发现获取地址
    address := sc.resolveService(serviceName)

    // 连接配置
    kacp := keepalive.ClientParameters{
        Time:                10 * time.Second,
        Timeout:             20 * time.Second,
        PermitWithoutStream: true,
    }

    conn, err := grpc.Dial(address,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithKeepaliveParams(kacp),
        grpc.WithDefaultServiceConfig(`{
            "loadBalancingConfig": [{"round_robin": {}}],
            "healthCheckConfig": {"serviceName": ""}
        }`),
        grpc.WithUnaryInterceptor(chainUnaryInterceptors(
            tracingInterceptor,
            metricsInterceptor,
            retryInterceptor,
        )),
    )

    if err != nil {
        return nil, err
    }

    return conn, nil
}

// resolveService 服务发现
func (sc *ServiceClient) resolveService(serviceName string) string {
    // 使用Kubernetes DNS或Consul进行服务发现
    // 格式: <service>.<namespace>.svc.cluster.local:<port>
    return fmt.Sprintf("%s.%s.svc.cluster.local:50051", 
        serviceName, getNamespace())
}
```

### 7.1.3 同步调用模式

```go
package integration

import (
    "context"
    "fmt"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/status"
)

// SyncCallOptions 同步调用选项
type SyncCallOptions struct {
    Timeout       time.Duration
    RetryCount    int
    RetryInterval time.Duration
    CircuitBreaker bool
}

// DefaultSyncCallOptions 默认选项
var DefaultSyncCallOptions = &SyncCallOptions{
    Timeout:       5 * time.Second,
    RetryCount:    3,
    RetryInterval: 100 * time.Millisecond,
    CircuitBreaker: true,
}

// CallService 同步调用服务
func CallService(
    ctx context.Context,
    serviceName string,
    method string,
    request interface{},
    response interface{},
    opts *SyncCallOptions,
) error {
    if opts == nil {
        opts = DefaultSyncCallOptions
    }

    // 创建带超时的上下文
    callCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
    defer cancel()

    // 获取服务客户端
    client, err := serviceClient.GetClient(serviceName)
    if err != nil {
        return fmt.Errorf("获取服务客户端失败: %w", err)
    }

    // 执行调用（带重试）
    var lastErr error
    for i := 0; i <= opts.RetryCount; i++ {
        if i > 0 {
            time.Sleep(opts.RetryInterval * time.Duration(i))
        }

        err = client.Invoke(callCtx, method, request, response)
        if err == nil {
            return nil
        }

        lastErr = err

        // 检查是否可重试
        if !isRetryableError(err) {
            break
        }
    }

    return fmt.Errorf("服务调用失败(重试%d次): %w", opts.RetryCount, lastErr)
}

// isRetryableError 检查错误是否可重试
func isRetryableError(err error) bool {
    if err == nil {
        return false
    }

    st, ok := status.FromError(err)
    if !ok {
        return true // 未知错误，尝试重试
    }

    switch st.Code() {
    case codes.DeadlineExceeded,
        codes.Unavailable,
        codes.Aborted,
        codes.ResourceExhausted:
        return true
    default:
        return false
    }
}

// 服务调用示例
func ExampleServiceCall() {
    ctx := context.Background()

    // 调用几何服务
    req := &geometry.GetGeometryRequest{
        Id:       "geo-123",
        TenantId: "tenant-456",
    }

    var resp geometry.Geometry

    err := CallService(ctx, "geometry-service", "/geometry.v1.GeometryService/GetGeometry",
        req, &resp, &SyncCallOptions{
            Timeout:    3 * time.Second,
            RetryCount: 2,
        })

    if err != nil {
        // 处理错误
    }
}
```

### 7.1.4 异步事件模式

```go
package integration

import (
    "context"
    "encoding/json"

    "github.com/nats-io/nats.go"
    "github.com/segmentio/kafka-go"
)

// EventBus 事件总线
type EventBus struct {
    natsConn *nats.Conn
    kafkaWriter *kafka.Writer
}

// PublishEvent 发布事件
func (eb *EventBus) PublishEvent(
    ctx context.Context,
    topic string,
    event interface{},
) error {
    // 序列化事件
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    // 发布到NATS（实时事件）
    if eb.natsConn != nil {
        err = eb.natsConn.Publish(topic, data)
        if err != nil {
            return err
        }
    }

    // 发布到Kafka（持久化事件）
    if eb.kafkaWriter != nil {
        err = eb.kafkaWriter.WriteMessages(ctx, kafka.Message{
            Topic: topic,
            Key:   []byte(getEventKey(event)),
            Value: data,
        })
        if err != nil {
            return err
        }
    }

    return nil
}

// SubscribeEvent 订阅事件
func (eb *EventBus) SubscribeEvent(
    ctx context.Context,
    topic string,
    groupID string,
    handler func(ctx context.Context, event []byte) error,
) error {
    // Kafka消费者
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{"kafka:9092"},
        Topic:   topic,
        GroupID: groupID,
    })

    go func() {
        defer reader.Close()

        for {
            select {
            case <-ctx.Done():
                return
            default:
                msg, err := reader.ReadMessage(ctx)
                if err != nil {
                    continue
                }

                // 处理消息
                if err := handler(ctx, msg.Value); err != nil {
                    // 记录错误，继续处理下一条
                }
            }
        }
    }()

    return nil
}

// DomainEvent 领域事件
type DomainEvent struct {
    EventID       string                 `json:"event_id"`
    EventType     string                 `json:"event_type"`
    AggregateID   string                 `json:"aggregate_id"`
    AggregateType string                 `json:"aggregate_type"`
    Timestamp     int64                  `json:"timestamp"`
    Version       int64                  `json:"version"`
    Payload       map[string]interface{} `json:"payload"`
    Metadata      map[string]string      `json:"metadata"`
}

// 事件订阅示例
func ExampleEventSubscription() {
    ctx := context.Background()

    eventBus := &EventBus{}

    // 订阅几何变更事件
    eventBus.SubscribeEvent(ctx, "geometry.changed", "property-service",
        func(ctx context.Context, event []byte) error {
            var geoEvent DomainEvent
            if err := json.Unmarshal(event, &geoEvent); err != nil {
                return err
            }

            // 处理几何变更，更新相关属性
            // ...

            return nil
        })
}
```

## 7.2 分布式事务实现

### 7.2.1 Saga模式实现

```go
package transaction

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// Saga 分布式事务编排器
type Saga struct {
    ID          string
    Steps       []SagaStep
    Status      SagaStatus
    CurrentStep int
    StartTime   time.Time
    EndTime     *time.Time
    CompensationLog []CompensationRecord
}

// SagaStep Saga步骤
type SagaStep struct {
    Name         string
    Action       func(ctx context.Context) error
    Compensation func(ctx context.Context) error
    Status       StepStatus
}

// SagaStatus Saga状态
type SagaStatus string

const (
    SagaStatusPending     SagaStatus = "PENDING"
    SagaStatusRunning     SagaStatus = "RUNNING"
    SagaStatusCompleted   SagaStatus = "COMPLETED"
    SagaStatusFailed      SagaStatus = "FAILED"
    SagaStatusCompensating SagaStatus = "COMPENSATING"
    SagaStatusCompensated SagaStatus = "COMPENSATED"
)

// StepStatus 步骤状态
type StepStatus string

const (
    StepStatusPending    StepStatus = "PENDING"
    StepStatusRunning    StepStatus = "RUNNING"
    StepStatusCompleted  StepStatus = "COMPLETED"
    StepStatusFailed     StepStatus = "FAILED"
    StepStatusCompensated StepStatus = "COMPENSATED"
)

// CompensationRecord 补偿记录
type CompensationRecord struct {
    StepName    string
    StepIndex   int
    Timestamp   time.Time
    Success     bool
    Error       string
}

// NewSaga 创建Saga
func NewSaga() *Saga {
    return &Saga{
        ID:        uuid.New().String(),
        Steps:     make([]SagaStep, 0),
        Status:    SagaStatusPending,
        StartTime: time.Now(),
        CompensationLog: make([]CompensationRecord, 0),
    }
}

// AddStep 添加步骤
func (s *Saga) AddStep(
    name string,
    action func(ctx context.Context) error,
    compensation func(ctx context.Context) error,
) {
    s.Steps = append(s.Steps, SagaStep{
        Name:         name,
        Action:       action,
        Compensation: compensation,
        Status:       StepStatusPending,
    })
}

// Execute 执行Saga
func (s *Saga) Execute(ctx context.Context) error {
    s.Status = SagaStatusRunning

    for i, step := range s.Steps {
        s.CurrentStep = i
        step.Status = StepStatusRunning

        // 执行步骤
        if err := step.Action(ctx); err != nil {
            step.Status = StepStatusFailed
            s.Status = SagaStatusFailed

            // 触发补偿
            if err := s.compensate(ctx, i-1); err != nil {
                return fmt.Errorf("Saga执行失败且补偿失败: %w", err)
            }

            return fmt.Errorf("Saga步骤 '%s' 执行失败: %w", step.Name, err)
        }

        step.Status = StepStatusCompleted
    }

    s.Status = SagaStatusCompleted
    endTime := time.Now()
    s.EndTime = &endTime

    return nil
}

// compensate 执行补偿
func (s *Saga) compensate(ctx context.Context, lastCompletedStep int) error {
    s.Status = SagaStatusCompensating

    // 逆序执行补偿
    for i := lastCompletedStep; i >= 0; i-- {
        step := s.Steps[i]

        if step.Compensation != nil {
            record := CompensationRecord{
                StepName:  step.Name,
                StepIndex: i,
                Timestamp: time.Now(),
            }

            if err := step.Compensation(ctx); err != nil {
                record.Success = false
                record.Error = err.Error()
                s.CompensationLog = append(s.CompensationLog, record)

                // 补偿失败，需要人工介入
                return fmt.Errorf("补偿步骤 '%s' 失败: %w", step.Name, err)
            }

            record.Success = true
            s.CompensationLog = append(s.CompensationLog, record)
            step.Status = StepStatusCompensated
        }
    }

    s.Status = SagaStatusCompensated
    return nil
}

// Saga使用示例: 创建设计文档
func ExampleCreateDocumentSaga() {
    ctx := context.Background()

    saga := NewSaga()

    var documentID string
    var geometryID string

    // 步骤1: 创建文档
    saga.AddStep("create_document",
        func(ctx context.Context) error {
            // 调用文档服务创建文档
            doc, err := documentService.CreateDocument(ctx, &CreateDocumentRequest{
                Name: "New Design",
            })
            if err != nil {
                return err
            }
            documentID = doc.Id
            return nil
        },
        func(ctx context.Context) error {
            // 补偿: 删除文档
            if documentID != "" {
                return documentService.DeleteDocument(ctx, &DeleteDocumentRequest{
                    Id: documentID,
                })
            }
            return nil
        },
    )

    // 步骤2: 创建初始几何
    saga.AddStep("create_geometry",
        func(ctx context.Context) error {
            // 调用几何服务创建几何
            geo, err := geometryService.CreateGeometry(ctx, &CreateGeometryRequest{
                DocumentId: documentID,
                Geometry:   &Geometry{Type: GeometryType_POINT},
            })
            if err != nil {
                return err
            }
            geometryID = geo.Id
            return nil
        },
        func(ctx context.Context) error {
            // 补偿: 删除几何
            if geometryID != "" {
                return geometryService.DeleteGeometry(ctx, &DeleteGeometryRequest{
                    Id: geometryID,
                })
            }
            return nil
        },
    )

    // 步骤3: 设置默认属性
    saga.AddStep("set_default_properties",
        func(ctx context.Context) error {
            // 调用属性服务设置属性
            _, err := propertyService.BatchSetProperties(ctx, &BatchSetPropertiesRequest{
                Requests: []*SetPropertyRequest{
                    {
                        ElementId:            geometryID,
                        PropertyDefinitionId: "prop_name",
                        Value: &PropertyValue{StringValue: "Default Name"},
                    },
                },
            })
            return err
        },
        func(ctx context.Context) error {
            // 补偿: 删除属性
            return propertyService.BatchDeleteProperties(ctx, &BatchDeletePropertiesRequest{
                Requests: []*DeletePropertyRequest{
                    {ElementId: geometryID, PropertyDefinitionId: "prop_name"},
                },
            })
        },
    )

    // 执行Saga
    if err := saga.Execute(ctx); err != nil {
        // 处理错误
    }
}
```

### 7.2.2 TCC模式实现

```go
package transaction

import (
    "context"
    "fmt"
)

// TCC 事务协调器
type TCC struct {
    Participants []TCCParticipant
}

// TCCParticipant TCC参与者
type TCCParticipant struct {
    Name    string
    Try     func(ctx context.Context) error
    Confirm func(ctx context.Context) error
    Cancel  func(ctx context.Context) error
}

// Execute 执行TCC事务
func (t *TCC) Execute(ctx context.Context) error {
    // 阶段1: Try
    confirmed := make([]bool, len(t.Participants))

    for i, p := range t.Participants {
        if err := p.Try(ctx); err != nil {
            // Try失败，执行Cancel
            for j := i - 1; j >= 0; j-- {
                if confirmed[j] {
                    if cancelErr := t.Participants[j].Cancel(ctx); cancelErr != nil {
                        // 记录Cancel失败，需要人工处理
                    }
                }
            }
            return fmt.Errorf("TCC Try阶段失败 [%s]: %w", p.Name, err)
        }
        confirmed[i] = true
    }

    // 阶段2: Confirm
    for _, p := range t.Participants {
        if err := p.Confirm(ctx); err != nil {
            // Confirm失败，记录错误，需要人工处理
            return fmt.Errorf("TCC Confirm阶段失败 [%s]: %w", p.Name, err)
        }
    }

    return nil
}
```

## 7.3 熔断降级策略

### 7.3.1 熔断器实现

```go
package resilience

import (
    "context"
    "errors"
    "sync"
    "time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
    StateClosed CircuitBreakerState = iota    // 关闭，正常
    StateOpen                                 // 打开，熔断
    StateHalfOpen                             // 半开，试探
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
    name          string
    state         CircuitBreakerState
    failureCount  int
    successCount  int
    lastFailureTime time.Time

    // 配置
    failureThreshold    int           // 失败阈值
    successThreshold    int           // 成功阈值（半开状态）
    timeoutDuration     time.Duration // 熔断持续时间
    halfOpenMaxCalls    int           // 半开状态最大调用数

    mu sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
    return &CircuitBreaker{
        name:             name,
        state:            StateClosed,
        failureThreshold: config.FailureThreshold,
        successThreshold: config.SuccessThreshold,
        timeoutDuration:  config.TimeoutDuration,
        halfOpenMaxCalls: config.HalfOpenMaxCalls,
    }
}

// Execute 执行带熔断保护的操作
func (cb *CircuitBreaker) Execute(
    ctx context.Context,
    operation func(ctx context.Context) error,
    fallback func(ctx context.Context) error,
) error {
    cb.mu.Lock()

    // 检查状态
    switch cb.state {
    case StateOpen:
        // 检查是否可以切换到半开状态
        if time.Since(cb.lastFailureTime) > cb.timeoutDuration {
            cb.state = StateHalfOpen
            cb.failureCount = 0
            cb.successCount = 0
        } else {
            cb.mu.Unlock()
            // 执行降级逻辑
            if fallback != nil {
                return fallback(ctx)
            }
            return errors.New("circuit breaker is open")
        }

    case StateHalfOpen:
        if cb.failureCount+cb.successCount >= cb.halfOpenMaxCalls {
            cb.mu.Unlock()
            if fallback != nil {
                return fallback(ctx)
            }
            return errors.New("circuit breaker half-open limit reached")
        }
    }

    cb.mu.Unlock()

    // 执行操作
    err := operation(ctx)

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.recordFailure()
    } else {
        cb.recordSuccess()
    }

    return err
}

// recordFailure 记录失败
func (cb *CircuitBreaker) recordFailure() {
    cb.failureCount++
    cb.lastFailureTime = time.Now()

    switch cb.state {
    case StateClosed:
        if cb.failureCount >= cb.failureThreshold {
            cb.state = StateOpen
        }
    case StateHalfOpen:
        cb.state = StateOpen
    }
}

// recordSuccess 记录成功
func (cb *CircuitBreaker) recordSuccess() {
    cb.successCount++

    switch cb.state {
    case StateHalfOpen:
        if cb.successCount >= cb.successThreshold {
            cb.state = StateClosed
            cb.failureCount = 0
            cb.successCount = 0
        }
    case StateClosed:
        cb.failureCount = 0
    }
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    return cb.state
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
    FailureThreshold    int
    SuccessThreshold    int
    TimeoutDuration     time.Duration
    HalfOpenMaxCalls    int
}

// DefaultCircuitBreakerConfig 默认配置
var DefaultCircuitBreakerConfig = &CircuitBreakerConfig{
    FailureThreshold: 5,
    SuccessThreshold: 3,
    TimeoutDuration:  30 * time.Second,
    HalfOpenMaxCalls: 3,
}

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
    breakers map[string]*CircuitBreaker
    mu       sync.RWMutex
}

// GetBreaker 获取或创建熔断器
func (m *CircuitBreakerManager) GetBreaker(
    name string,
    config *CircuitBreakerConfig,
) *CircuitBreaker {
    m.mu.RLock()
    if cb, ok := m.breakers[name]; ok {
        m.mu.RUnlock()
        return cb
    }
    m.mu.RUnlock()

    m.mu.Lock()
    defer m.mu.Unlock()

    // 双重检查
    if cb, ok := m.breakers[name]; ok {
        return cb
    }

    cb := NewCircuitBreaker(name, config)
    m.breakers[name] = cb
    return cb
}

// 全局熔断器管理器
var globalCBManager = &CircuitBreakerManager{
    breakers: make(map[string]*CircuitBreaker),
}
```

### 7.3.2 降级策略实现

```go
package resilience

import (
    "context"
    "encoding/json"
    "time"
)

// FallbackStrategy 降级策略
type FallbackStrategy int

const (
    FallbackReturnDefault FallbackStrategy = iota
    FallbackReturnCache
    FallbackReturnEmpty
    FallbackReturnError
)

// FallbackConfig 降级配置
type FallbackConfig struct {
    Strategy      FallbackStrategy
    DefaultValue  interface{}
    CacheKey      string
    CacheDuration time.Duration
}

// ExecuteWithFallback 执行带降级的操作
func ExecuteWithFallback(
    ctx context.Context,
    operation func(ctx context.Context) (interface{}, error),
    config *FallbackConfig,
) (interface{}, error) {
    result, err := operation(ctx)
    if err == nil {
        return result, nil
    }

    // 执行降级
    switch config.Strategy {
    case FallbackReturnDefault:
        return config.DefaultValue, nil

    case FallbackReturnCache:
        // 从缓存获取
        cached, cacheErr := getFromCache(ctx, config.CacheKey)
        if cacheErr == nil {
            return cached, nil
        }
        return config.DefaultValue, nil

    case FallbackReturnEmpty:
        return nil, nil

    case FallbackReturnError:
        return nil, err

    default:
        return nil, err
    }
}

// 降级示例
func ExampleFallback() {
    ctx := context.Background()

    // 获取几何数据，带缓存降级
    result, err := ExecuteWithFallback(ctx,
        func(ctx context.Context) (interface{}, error) {
            return geometryService.GetGeometry(ctx, req)
        },
        &FallbackConfig{
            Strategy:      FallbackReturnCache,
            CacheKey:      "geometry:" + req.Id,
            CacheDuration: 5 * time.Minute,
            DefaultValue:  &geometry.Geometry{},
        },
    )

    if err != nil {
        // 处理错误
    }
}
```

## 7.4 超时重试策略

### 7.4.1 重试策略实现

```go
package resilience

import (
    "context"
    "fmt"
    "math"
    "math/rand"
    "time"
)

// RetryPolicy 重试策略
type RetryPolicy struct {
    MaxRetries      int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
    RandomizationFactor float64
    RetryableErrors []error
}

// DefaultRetryPolicy 默认重试策略
var DefaultRetryPolicy = &RetryPolicy{
    MaxRetries:          3,
    InitialInterval:     100 * time.Millisecond,
    MaxInterval:         30 * time.Second,
    Multiplier:          2.0,
    RandomizationFactor: 0.1,
}

// Retry 执行带重试的操作
func Retry(
    ctx context.Context,
    operation func(ctx context.Context) error,
    policy *RetryPolicy,
) error {
    if policy == nil {
        policy = DefaultRetryPolicy
    }

    var lastErr error

    for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
        if attempt > 0 {
            // 计算退避时间
            backoff := policy.calculateBackoff(attempt)

            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(backoff):
            }
        }

        err := operation(ctx)
        if err == nil {
            return nil
        }

        lastErr = err

        // 检查是否可重试
        if !policy.isRetryable(err) {
            return err
        }

        // 最后一次尝试失败
        if attempt == policy.MaxRetries {
            break
        }
    }

    return fmt.Errorf("操作失败(重试%d次): %w", policy.MaxRetries, lastErr)
}

// calculateBackoff 计算退避时间
func (p *RetryPolicy) calculateBackoff(attempt int) time.Duration {
    if attempt <= 0 {
        return 0
    }

    // 指数退避
    interval := float64(p.InitialInterval) * math.Pow(p.Multiplier, float64(attempt-1))

    // 限制最大间隔
    if interval > float64(p.MaxInterval) {
        interval = float64(p.MaxInterval)
    }

    // 添加随机抖动
    delta := p.RandomizationFactor * interval
    minInterval := interval - delta
    maxInterval := interval + delta

    // 生成随机间隔
    return time.Duration(minInterval + (rand.Float64() * (maxInterval - minInterval)))
}

// isRetryable 检查错误是否可重试
func (p *RetryPolicy) isRetryable(err error) bool {
    if err == nil {
        return false
    }

    // 如果没有指定可重试错误，则默认所有错误都可重试
    if len(p.RetryableErrors) == 0 {
        return true
    }

    for _, retryableErr := range p.RetryableErrors {
        if errors.Is(err, retryableErr) {
            return true
        }
    }

    return false
}

// RetryWithResult 带结果的重试
func RetryWithResult[T any](
    ctx context.Context,
    operation func(ctx context.Context) (T, error),
    policy *RetryPolicy,
) (T, error) {
    var result T

    err := Retry(ctx, func(ctx context.Context) error {
        var err error
        result, err = operation(ctx)
        return err
    }, policy)

    return result, err
}

// 重试示例
func ExampleRetry() {
    ctx := context.Background()

    // 配置重试策略
    policy := &RetryPolicy{
        MaxRetries:      5,
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     5 * time.Second,
        Multiplier:      2.0,
    }

    // 执行带重试的操作
    err := Retry(ctx, func(ctx context.Context) error {
        return callExternalService(ctx)
    }, policy)

    if err != nil {
        // 处理最终失败
    }
}
```

### 7.4.2 超时控制实现

```go
package resilience

import (
    "context"
    "fmt"
    "time"
)

// TimeoutConfig 超时配置
type TimeoutConfig struct {
    Timeout         time.Duration
    GracefulTimeout time.Duration
}

// ExecuteWithTimeout 执行带超时的操作
func ExecuteWithTimeout(
    parentCtx context.Context,
    operation func(ctx context.Context) error,
    config *TimeoutConfig,
) error {
    ctx, cancel := context.WithTimeout(parentCtx, config.Timeout)
    defer cancel()

    done := make(chan error, 1)

    go func() {
        done <- operation(ctx)
    }()

    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        // 超时处理
        if config.GracefulTimeout > 0 {
            // 等待优雅关闭
            select {
            case err := <-done:
                return err
            case <-time.After(config.GracefulTimeout):
                return fmt.Errorf("operation timed out and graceful shutdown failed")
            }
        }
        return fmt.Errorf("operation timed out after %v", config.Timeout)
    }
}

// DeadlineManager 截止时间管理器
type DeadlineManager struct {
    defaultTimeout time.Duration
    maxTimeout     time.Duration
}

// NewDeadlineManager 创建截止时间管理器
func NewDeadlineManager(defaultTimeout, maxTimeout time.Duration) *DeadlineManager {
    return &DeadlineManager{
        defaultTimeout: defaultTimeout,
        maxTimeout:     maxTimeout,
    }
}

// GetDeadline 获取截止时间
func (dm *DeadlineManager) GetDeadline(ctx context.Context) (time.Time, bool) {
    deadline, ok := ctx.Deadline()
    if !ok {
        // 没有设置截止时间，使用默认
        return time.Now().Add(dm.defaultTimeout), true
    }

    // 检查剩余时间
    remaining := time.Until(deadline)
    if remaining > dm.maxTimeout {
        // 剩余时间太长，限制为最大超时
        return time.Now().Add(dm.maxTimeout), true
    }

    return deadline, true
}

// ServiceTimeoutConfig 各服务超时配置
var ServiceTimeoutConfig = map[string]*TimeoutConfig{
    "user-service": {
        Timeout:         5 * time.Second,
        GracefulTimeout: 2 * time.Second,
    },
    "document-service": {
        Timeout:         10 * time.Second,
        GracefulTimeout: 3 * time.Second,
    },
    "geometry-service": {
        Timeout:         15 * time.Second,
        GracefulTimeout: 5 * time.Second,
    },
    "property-service": {
        Timeout:         10 * time.Second,
        GracefulTimeout: 3 * time.Second,
    },
    "collaboration-service": {
        Timeout:         30 * time.Second,
        GracefulTimeout: 10 * time.Second,
    },
    "script-service": {
        Timeout:         60 * time.Second,
        GracefulTimeout: 10 * time.Second,
    },
    "version-service": {
        Timeout:         10 * time.Second,
        GracefulTimeout: 3 * time.Second,
    },
}
```

## 7.5 服务网格配置

### 7.5.1 Istio配置

```yaml
# VirtualService - 路由配置
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: archplatform-routes
  namespace: archplatform
spec:
  hosts:
    - "api.archplatform.com"
  gateways:
    - archplatform-gateway
  http:
    # 用户服务路由
    - match:
        - uri:
            prefix: /api/v1/users
      route:
        - destination:
            host: user-service
            port:
              number: 50051
      timeout: 5s
      retries:
        attempts: 3
        perTryTimeout: 2s
        retryOn: gateway-error,connect-failure,refused-stream

    # 几何服务路由
    - match:
        - uri:
            prefix: /api/v1/geometry
      route:
        - destination:
            host: geometry-service
            port:
              number: 50051
      timeout: 15s
      retries:
        attempts: 3
        perTryTimeout: 5s

    # 协作服务路由 - WebSocket支持
    - match:
        - uri:
            prefix: /api/v1/collaboration
      route:
        - destination:
            host: collaboration-service
            port:
              number: 50051
      timeout: 300s

---
# DestinationRule - 负载均衡和连接池
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: archplatform-destinations
  namespace: archplatform
spec:
  host: "*.archplatform.svc.cluster.local"
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 100
        http2MaxRequests: 1000
        maxRequestsPerConnection: 100
        maxRetries: 3
    loadBalancer:
      simple: LEAST_CONN
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s

---
# CircuitBreaker - 熔断配置
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: geometry-service-circuit-breaker
  namespace: archplatform
spec:
  host: geometry-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 50
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 50
    outlierDetection:
      consecutiveErrors: 5
      interval: 10s
      baseEjectionTime: 30s
      maxEjectionPercent: 50

---
# Rate Limiting - 限流配置
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: rate-limit-filter
  namespace: istio-system
spec:
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        context: GATEWAY
        listener:
          filterChain:
            filter:
              name: envoy.filters.network.http_connection_manager
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.local_ratelimit
          typed_config:
            "@type": type.googleapis.com/udpa.type.v1.TypedStruct
            type_url: type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
            value:
              stat_prefix: http_local_rate_limiter
              token_bucket:
                max_tokens: 1000
                tokens_per_fill: 100
                fill_interval: 1s
              filter_enabled:
                runtime_key: local_rate_limit_enabled
                default_value:
                  numerator: 100
                  denominator: HUNDRED
              filter_enforced:
                runtime_key: local_rate_limit_enforced
                default_value:
                  numerator: 100
                  denominator: HUNDRED
```

### 7.5.2 监控与追踪

```yaml
# Prometheus监控配置
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: archplatform-metrics
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app.kubernetes.io/part-of: archplatform
  endpoints:
    - port: metrics
      interval: 15s
      path: /metrics

---
# Jaeger分布式追踪
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: archplatform-tracing
  namespace: observability
spec:
  strategy: production
  storage:
    type: elasticsearch
    options:
      es:
        server-urls: http://elasticsearch:9200
  ingress:
    enabled: true
```

---

## 附录

### A. 技术选型总结

| 组件 | 技术选型 | 版本 | 说明 |
|------|----------|------|------|
| 服务框架 | gRPC + Go | 1.56+ | 高性能RPC通信 |
| 服务网格 | Istio | 1.18+ | 服务治理 |
| 消息队列 | Kafka + NATS | 3.5+ / 2.10+ | 事件驱动架构 |
| 数据库 | PostgreSQL + PostGIS | 15+ | 主数据库 |
| 缓存 | Redis Cluster | 7.0+ | 分布式缓存 |
| 对象存储 | MinIO | 2023+ | 文件存储 |
| 搜索引擎 | Elasticsearch | 8.0+ | 全文搜索 |
| 时序数据库 | ClickHouse | 23+ | 分析数据 |
| 容器编排 | Kubernetes | 1.27+ | 容器管理 |
| API网关 | Kong / Istio Ingress | 3.0+ | 流量入口 |
| 监控 | Prometheus + Grafana | 2.45+ / 10.0+ | 监控告警 |
| 追踪 | Jaeger | 1.47+ | 分布式追踪 |
| 日志 | Loki + Fluentd | 2.9+ | 日志收集 |

### B. 服务端口分配

| 服务 | gRPC端口 | HTTP端口 | 指标端口 |
|------|----------|----------|----------|
| user-service | 50051 | 8080 | 9090 |
| document-service | 50052 | 8081 | 9091 |
| geometry-service | 50053 | 8082 | 9092 |
| property-service | 50054 | 8083 | 9093 |
| collaboration-service | 50055 | 8084 | 9094 |
| script-service | 50056 | 8085 | 9095 |
| version-service | 50057 | 8086 | 9096 |

---

*文档结束*
