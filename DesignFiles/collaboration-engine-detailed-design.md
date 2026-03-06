# 详细设计阶段 - 协作引擎详细设计报告

## 文档信息

| 项目 | 内容 |
|------|------|
| 文档名称 | 协作引擎详细设计报告 |
| 版本 | v1.0 |
| 阶段 | 详细设计阶段 |
| 适用范围 | 半自动化建筑设计平台 |

---

## 目录

1. [CRDT数据结构详细设计](#1-crdt数据结构详细设计)
2. [同步协议详细设计](#2-同步协议详细设计)
3. [WebSocket网关详细设计](#3-websocket网关详细设计)
4. [操作转换详细设计](#4-操作转换详细设计)
5. [冲突解决详细设计](#5-冲突解决详细设计)
6. [性能优化详细设计](#6-性能优化详细设计)
7. [代码实现示例](#7-代码实现示例)

---

## 1. CRDT数据结构详细设计

### 1.1 设计概述

基于Yjs框架，我们设计自定义的CRDT数据结构来支持建筑设计领域的协作需求。核心设计原则：

- **分层设计**：文档 → 图层 → 元素 → 属性
- **类型安全**：TypeScript类型定义确保数据一致性
- **可扩展性**：支持自定义几何类型和属性类型
- **高效同步**：增量更新，最小化传输数据

### 1.2 Yjs文档结构设计

```
Y.Doc (项目文档)
├── Y.Map (元数据层)
│   ├── projectId: string
│   ├── version: string
│   ├── createdAt: number
│   ├── updatedAt: number
│   └── versionVector: Y.Map<clientId, counter>
│
├── Y.Map (用户状态层)
│   └── userStates: Y.Map<clientId, UserState>
│       ├── cursor: { x, y, z }
│       ├── selection: string[]
│       ├── viewport: { center, zoom }
│       └── lastActive: number
│
├── Y.Map (图层管理层)
│   └── layers: Y.Map<layerId, Layer>
│       ├── name: string
│       ├── visible: boolean
│       ├── locked: boolean
│       ├── opacity: number
│       ├── order: number
│       └── elements: Y.Map<elementId, Element>
│
└── Y.Map (撤销/重做历史)
    └── undoManager: Y.UndoManager
```

### 1.3 几何数据CRDT类型定义

```typescript
// ============================================
// 基础类型定义
// ============================================

/**
 * 三维点坐标
 */
interface Point3D {
  x: number;
  y: number;
  z: number;
}

/**
 * 变换矩阵 (4x4)
 */
interface TransformMatrix {
  m11: number; m12: number; m13: number; m14: number;
  m21: number; m22: number; m23: number; m24: number;
  m31: number; m32: number; m33: number; m34: number;
  m41: number; m42: number; m43: number; m44: number;
}

/**
 * 边界框
 */
interface BoundingBox {
  min: Point3D;
  max: Point3D;
}

// ============================================
// 几何元素基类
// ============================================

/**
 * 几何元素基类 - 所有建筑元素的抽象基类
 */
interface GeometryElement {
  // 唯一标识
  id: string;
  
  // 元素类型
  type: GeometryType;
  
  // 空间变换
  transform: TransformMatrix;
  
  // 边界框（缓存，用于快速碰撞检测）
  boundingBox: BoundingBox;
  
  // 所属图层
  layerId: string;
  
  // 创建信息
  createdBy: string;
  createdAt: number;
  
  // 修改信息
  modifiedBy: string;
  modifiedAt: number;
  
  // 版本向量（用于冲突检测）
  versionVector: VersionVector;
  
  // 锁定状态
  lockInfo?: LockInfo;
}

/**
 * 版本向量 - 用于因果排序
 */
interface VersionVector {
  [clientId: string]: number;
}

/**
 * 锁定信息
 */
interface LockInfo {
  clientId: string;
  timestamp: number;
  expiresAt: number;
}

/**
 * 几何元素类型枚举
 */
enum GeometryType {
  // 基础几何
  POINT = 'point',
  LINE = 'line',
  POLYLINE = 'polyline',
  POLYGON = 'polygon',
  CIRCLE = 'circle',
  ARC = 'arc',
  
  // 建筑专用
  WALL = 'wall',
  DOOR = 'door',
  WINDOW = 'window',
  COLUMN = 'column',
  BEAM = 'beam',
  SLAB = 'slab',
  ROOF = 'roof',
  STAIR = 'stair',
  RAILING = 'railing',
  
  // 空间元素
  ROOM = 'room',
  ZONE = 'zone',
  
  // 注释元素
  DIMENSION = 'dimension',
  TEXT = 'text',
  MARKER = 'marker',
  
  // 组/块
  GROUP = 'group',
  BLOCK = 'block',
  
  // 外部引用
  XREF = 'xref'
}

// ============================================
// 具体几何类型定义
// ============================================

/**
 * 墙体元素
 */
interface WallElement extends GeometryElement {
  type: GeometryType.WALL;
  
  // 起点和终点
  startPoint: Point3D;
  endPoint: Point3D;
  
  // 墙体属性
  height: number;
  thickness: number;
  baseOffset: number;
  
  // 结构属性
  structural: boolean;
  bearing: boolean;
  
  // 材质
  material: MaterialRef;
  
  // 开洞（门、窗）
  openings: Opening[];
  
  // 连接关系
  connections: WallConnection[];
}

/**
 * 墙体连接
 */
interface WallConnection {
  wallId: string;
  connectionType: 'T' | 'L' | 'X' | 'end';
  position: Point3D;
}

/**
 * 开洞（门/窗）
 */
interface Opening {
  id: string;
  type: 'door' | 'window';
  
  // 相对于墙体起点的位置
  distance: number;
  
  // 洞口尺寸
  width: number;
  height: number;
  sillHeight: number;
  
  // 关联元素ID
  elementId: string;
}

/**
 * 房间/空间元素
 */
interface RoomElement extends GeometryElement {
  type: GeometryType.ROOM;
  
  // 边界（由墙体或其他边界围成）
  boundaries: RoomBoundary[];
  
  // 楼层
  level: number;
  
  // 空间属性
  area: number;
  volume: number;
  perimeter: number;
  
  // 功能属性
  roomType: string;
  roomNumber: string;
  roomName: string;
  
  // 高度设置
  height: number;
  offsetFromLevel: number;
  
  // 计算点（用于面积计算）
  calculationPoint: Point3D;
}

/**
 * 房间边界
 */
interface RoomBoundary {
  elementId: string;
  elementType: 'wall' | 'line' | 'curve';
  side: 'left' | 'right';
}

/**
 * 尺寸标注元素
 */
interface DimensionElement extends GeometryElement {
  type: GeometryType.DIMENSION;
  
  // 标注点
  referencePoints: Point3D[];
  
  // 标注线位置
  dimensionLine: {
    start: Point3D;
    end: Point3D;
  };
  
  // 标注值
  value: number;
  
  // 标注样式
  style: DimensionStyle;
}

/**
 * 标注样式
 */
interface DimensionStyle {
  precision: number;
  unit: 'mm' | 'cm' | 'm' | 'ft' | 'in';
  textHeight: number;
  arrowSize: number;
  extensionLineOffset: number;
  extensionLineExtension: number;
}

// ============================================
// 组/块定义
// ============================================

/**
 * 组元素 - 可嵌套
 */
interface GroupElement extends GeometryElement {
  type: GeometryType.GROUP;
  
  // 组名称
  name: string;
  
  // 子元素ID列表
  children: string[];
  
  // 组边界框（包含所有子元素）
  groupBoundingBox: BoundingBox;
  
  // 是否可爆炸
  explodable: boolean;
}

/**
 * 块引用（外部定义的可复用元素）
 */
interface BlockElement extends GeometryElement {
  type: GeometryType.BLOCK;
  
  // 块定义ID
  blockDefId: string;
  
  // 块参数（动态块）
  parameters: BlockParameter[];
  
  // 属性覆盖
  attributeOverrides: Map<string, any>;
}

/**
 * 块参数
 */
interface BlockParameter {
  name: string;
  type: 'number' | 'string' | 'boolean' | 'point';
  value: any;
  constraints?: ParameterConstraint;
}

/**
 * 参数约束
 */
interface ParameterConstraint {
  min?: number;
  max?: number;
  enum?: any[];
  regex?: string;
}
```

### 1.4 属性数据CRDT类型定义

```typescript
// ============================================
// 属性系统类型定义
// ============================================

/**
 * 元素属性容器
 */
interface ElementProperties {
  // 元素ID
  elementId: string;
  
  // 属性集
  propertySets: Map<string, PropertySet>;
  
  // 扩展属性（用户自定义）
  customProperties: Map<string, CustomProperty>;
  
  // 属性版本
  propertyVersion: number;
  
  // 最后修改
  lastModified: {
    by: string;
    at: number;
    propertyKey: string;
  };
}

/**
 * 属性集 - 按类别组织的属性
 */
interface PropertySet {
  // 属性集名称
  name: string;
  
  // 属性集类型（如：Identity, Dimensions, Materials等）
  type: PropertySetType;
  
  // 属性定义
  properties: Map<string, Property>;
  
  // 是否可编辑
  editable: boolean;
  
  // 来源（系统/用户/插件）
  source: 'system' | 'user' | 'plugin';
}

/**
 * 属性集类型
 */
enum PropertySetType {
  IDENTITY = 'identity',           // 标识信息
  GEOMETRY = 'geometry',           // 几何属性
  DIMENSIONS = 'dimensions',       // 尺寸信息
  MATERIALS = 'materials',         // 材质信息
  STRUCTURAL = 'structural',       // 结构属性
  THERMAL = 'thermal',             // 热工性能
  ACOUSTIC = 'acoustic',           // 声学性能
  FIRE = 'fire',                   // 防火性能
  COST = 'cost',                   // 成本信息
  SCHEDULING = 'scheduling',       // 进度信息
  CUSTOM = 'custom'                // 自定义属性
}

/**
 * 单个属性定义
 */
interface Property {
  // 属性名
  name: string;
  
  // 显示标签
  displayName: string;
  
  // 属性值
  value: PropertyValue;
  
  // 数据类型
  dataType: PropertyDataType;
  
  // 单位
  unit?: string;
  
  // 是否只读
  readOnly: boolean;
  
  // 是否必填
  required: boolean;
  
  // 约束条件
  constraints?: PropertyConstraint;
  
  // 描述
  description?: string;
}

/**
 * 属性数据类型
 */
enum PropertyDataType {
  STRING = 'string',
  NUMBER = 'number',
  INTEGER = 'integer',
  BOOLEAN = 'boolean',
  DATE = 'date',
  TIME = 'time',
  DATETIME = 'datetime',
  POINT = 'point',
  VECTOR = 'vector',
  COLOR = 'color',
  ENUM = 'enum',
  ARRAY = 'array',
  OBJECT = 'object',
  REFERENCE = 'reference',
  URL = 'url',
  FILE = 'file'
}

/**
 * 属性值类型
 */
type PropertyValue = 
  | string 
  | number 
  | boolean 
  | Date 
  | Point3D 
  | Color 
  | string[] 
  | any[] 
  | object 
  | null;

/**
 * 颜色类型
 */
interface Color {
  r: number;
  g: number;
  b: number;
  a?: number;
}

/**
 * 属性约束
 */
interface PropertyConstraint {
  // 数值约束
  min?: number;
  max?: number;
  step?: number;
  
  // 字符串约束
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  
  // 枚举值
  enum?: any[];
  
  // 默认值
  default?: PropertyValue;
  
  // 计算公式
  formula?: string;
}

/**
 * 自定义属性
 */
interface CustomProperty {
  // 属性键
  key: string;
  
  // 属性值
  value: PropertyValue;
  
  // 数据类型
  dataType: PropertyDataType;
  
  // 创建者
  createdBy: string;
  
  // 创建时间
  createdAt: number;
  
  // 版本历史
  history: PropertyHistoryEntry[];
}

/**
 * 属性历史条目
 */
interface PropertyHistoryEntry {
  value: PropertyValue;
  modifiedBy: string;
  modifiedAt: number;
  reason?: string;
}

// ============================================
// 材质引用定义
// ============================================

/**
 * 材质引用
 */
interface MaterialRef {
  // 材质ID
  materialId: string;
  
  // 材质名称
  name: string;
  
  // 应用到元素的哪个部分
  application: MaterialApplication;
  
  // 材质参数覆盖
  parameterOverrides?: Map<string, any>;
}

/**
 * 材质应用位置
 */
enum MaterialApplication {
  SURFACE = 'surface',       // 表面
  CORE = 'core',             // 核心
  FINISH_EXTERIOR = 'finish_exterior',  // 外饰面
  FINISH_INTERIOR = 'finish_interior',  // 内饰面
  STRUCTURE = 'structure',   // 结构
  INSULATION = 'insulation', // 保温层
  ALL = 'all'                // 全部
}
```

### 1.5 CRDT元数据设计

```typescript
// ============================================
// CRDT元数据类型定义
// ============================================

/**
 * 文档级元数据
 */
interface DocumentMetadata {
  // 文档标识
  docId: string;
  projectId: string;
  
  // 版本信息
  schemaVersion: string;
  documentVersion: number;
  
  // 全局版本向量
  globalVersionVector: VersionVector;
  
  // 客户端注册表
  clientRegistry: ClientRegistry;
  
  // 同步状态
  syncState: SyncState;
  
  // 统计信息
  statistics: DocumentStatistics;
}

/**
 * 客户端注册表
 */
interface ClientRegistry {
  // 已注册客户端
  clients: Map<string, ClientInfo>;
  
  // 客户端ID分配计数器
  clientIdCounter: number;
  
  // 最后清理时间
  lastCleanup: number;
}

/**
 * 客户端信息
 */
interface ClientInfo {
  // 客户端ID
  clientId: string;
  
  // 用户信息
  userId: string;
  userName: string;
  
  // 连接信息
  connected: boolean;
  connectedAt: number;
  lastSeen: number;
  
  // 客户端能力
  capabilities: ClientCapabilities;
  
  // 客户端版本向量
  versionVector: VersionVector;
}

/**
 * 客户端能力
 */
interface ClientCapabilities {
  // 支持的CRDT类型
  supportedTypes: string[];
  
  // 最大消息大小
  maxMessageSize: number;
  
  // 是否支持压缩
  compression: boolean;
  
  // 是否支持二进制
  binarySupport: boolean;
}

/**
 * 同步状态
 */
interface SyncState {
  // 最后同步时间
  lastSyncTime: number;
  
  // 待发送更新
  pendingUpdates: number;
  
  // 待确认更新
  unconfirmedUpdates: number;
  
  // 同步模式
  mode: 'realtime' | 'batch' | 'manual';
  
  // 同步延迟（毫秒）
  syncDelay: number;
}

/**
 * 文档统计
 */
interface DocumentStatistics {
  // 元素统计
  elementCount: number;
  
  // 图层统计
  layerCount: number;
  
  // 用户统计
  activeUsers: number;
  totalUsers: number;
  
  // 操作统计
  operationCount: number;
  
  // 文档大小
  documentSize: number;
  
  // 历史记录大小
  historySize: number;
}

/**
 * 操作元数据
 */
interface OperationMetadata {
  // 操作ID（全局唯一）
  operationId: string;
  
  // 操作类型
  type: OperationType;
  
  // 操作来源
  origin: {
    clientId: string;
    userId: string;
    timestamp: number;
  };
  
  // 操作前的版本向量
  beforeVector: VersionVector;
  
  // 操作后的版本向量
  afterVector: VersionVector;
  
  // 影响的元素
  affectedElements: string[];
  
  // 操作依赖
  dependencies: string[];
  
  // 操作优先级
  priority: OperationPriority;
}

/**
 * 操作类型
 */
enum OperationType {
  // 元素操作
  ELEMENT_CREATE = 'element_create',
  ELEMENT_UPDATE = 'element_update',
  ELEMENT_DELETE = 'element_delete',
  ELEMENT_TRANSFORM = 'element_transform',
  
  // 属性操作
  PROPERTY_SET = 'property_set',
  PROPERTY_DELETE = 'property_delete',
  
  // 图层操作
  LAYER_CREATE = 'layer_create',
  LAYER_UPDATE = 'layer_update',
  LAYER_DELETE = 'layer_delete',
  LAYER_REORDER = 'layer_reorder',
  
  // 组操作
  GROUP_CREATE = 'group_create',
  GROUP_ADD = 'group_add',
  GROUP_REMOVE = 'group_remove',
  GROUP_UNGROUP = 'group_ungroup',
  
  // 选择操作
  SELECTION_CHANGE = 'selection_change',
  
  // 视口操作
  VIEWPORT_CHANGE = 'viewport_change'
}

/**
 * 操作优先级
 */
enum OperationPriority {
  CRITICAL = 0,    // 关键操作，立即同步
  HIGH = 1,        // 高优先级
  NORMAL = 2,      // 普通优先级
  LOW = 3,         // 低优先级，可批量
  BACKGROUND = 4   // 后台操作
}

/**
 * 冲突元数据
 */
interface ConflictMetadata {
  // 冲突ID
  conflictId: string;
  
  // 冲突类型
  type: ConflictType;
  
  // 冲突元素
  elementId: string;
  
  // 冲突的操作
  operations: string[];
  
  // 冲突检测时间
  detectedAt: number;
  
  // 冲突解决状态
  status: ConflictStatus;
  
  // 解决结果
  resolution?: ConflictResolution;
}

/**
 * 冲突类型
 */
enum ConflictType {
  // 并发修改冲突
  CONCURRENT_EDIT = 'concurrent_edit',
  
  // 删除后修改冲突
  EDIT_AFTER_DELETE = 'edit_after_delete',
  
  // 移动冲突
  MOVE_CONFLICT = 'move_conflict',
  
  // 属性冲突
  PROPERTY_CONFLICT = 'property_conflict',
  
  // 结构冲突
  STRUCTURE_CONFLICT = 'structure_conflict',
  
  // 依赖冲突
  DEPENDENCY_CONFLICT = 'dependency_conflict'
}

/**
 * 冲突状态
 */
enum ConflictStatus {
  DETECTED = 'detected',       // 已检测到
  AUTO_RESOLVED = 'auto_resolved', // 自动解决
  PENDING = 'pending',         // 等待人工处理
  RESOLVED = 'resolved',       // 已解决
  IGNORED = 'ignored'          // 已忽略
}

/**
 * 冲突解决结果
 */
interface ConflictResolution {
  // 解决方式
  method: 'auto_merge' | 'last_write_wins' | 'manual' | 'revert';
  
  // 解决者
  resolvedBy: string;
  
  // 解决时间
  resolvedAt: number;
  
  // 解决结果值
  result: any;
  
  // 丢弃的操作
  discardedOperations: string[];
}
```

---

## 2. 同步协议详细设计

### 2.1 同步消息格式定义

```typescript
// ============================================
// 同步消息基础类型
// ============================================

/**
 * 同步消息基类
 */
interface SyncMessage {
  // 消息ID
  messageId: string;
  
  // 消息类型
  type: MessageType;
  
  // 发送者信息
  sender: {
    clientId: string;
    userId: string;
  };
  
  // 时间戳
  timestamp: number;
  
  // 版本向量
  versionVector: VersionVector;
  
  // 消息负载
  payload: MessagePayload;
  
  // 消息元数据
  metadata?: MessageMetadata;
}

/**
 * 消息类型
 */
enum MessageType {
  // 连接管理
  CONNECT = 'connect',
  CONNECT_ACK = 'connect_ack',
  DISCONNECT = 'disconnect',
  
  // 同步请求
  SYNC_REQUEST = 'sync_request',
  SYNC_RESPONSE = 'sync_response',
  
  // 增量更新
  UPDATE = 'update',
  UPDATE_ACK = 'update_ack',
  
  // 全量同步
  FULL_SYNC = 'full_sync',
  FULL_SYNC_ACK = 'full_sync_ack',
  
  // 心跳
  PING = 'ping',
  PONG = 'pong',
  
  // 冲突
  CONFLICT_DETECTED = 'conflict_detected',
  CONFLICT_RESOLVED = 'conflict_resolved',
  
  // 状态
  STATE_BROADCAST = 'state_broadcast',
  
  // 错误
  ERROR = 'error'
}

/**
 * 消息负载类型
 */
type MessagePayload = 
  | ConnectPayload
  | SyncRequestPayload
  | SyncResponsePayload
  | UpdatePayload
  | FullSyncPayload
  | ConflictPayload
  | StateBroadcastPayload
  | ErrorPayload;

/**
 * 消息元数据
 */
interface MessageMetadata {
  // 压缩方式
  compression?: 'none' | 'gzip' | 'deflate';
  
  // 编码方式
  encoding?: 'json' | 'binary' | 'base64';
  
  // 消息大小
  size?: number;
  
  // 优先级
  priority?: number;
  
  // 重试次数
  retryCount?: number;
}

// ============================================
// 连接管理消息
// ============================================

/**
 * 连接请求负载
 */
interface ConnectPayload {
  // 文档ID
  documentId: string;
  
  // 认证令牌
  authToken: string;
  
  // 客户端信息
  clientInfo: {
    clientType: 'web' | 'desktop' | 'mobile';
    version: string;
    capabilities: ClientCapabilities;
  };
  
  // 初始状态请求
  initialState?: {
    includeHistory: boolean;
    maxHistorySize: number;
  };
}

/**
 * 连接确认负载
 */
interface ConnectAckPayload {
  // 分配的客户端ID
  clientId: string;
  
  // 连接状态
  status: 'success' | 'failed';
  
  // 失败原因
  reason?: string;
  
  // 服务器信息
  serverInfo: {
    serverId: string;
    version: string;
    maxClients: number;
    currentClients: number;
  };
  
  // 初始同步数据
  initialSync?: InitialSyncData;
}

/**
 * 初始同步数据
 */
interface InitialSyncData {
  // 文档状态向量
  stateVector: Uint8Array;
  
  // 差异更新
  diffUpdate?: Uint8Array;
  
  // 活跃客户端列表
  activeClients: ClientInfo[];
  
  // 文档元数据
  documentMetadata: DocumentMetadata;
}

// ============================================
// 同步请求/响应消息
// ============================================

/**
 * 同步请求负载
 */
interface SyncRequestPayload {
  // 请求的同步类型
  syncType: 'incremental' | 'full' | 'state_vector';
  
  // 客户端当前状态向量
  clientStateVector: Uint8Array;
  
  // 请求的范围
  scope?: {
    layers?: string[];
    elements?: string[];
    timeRange?: { start: number; end: number };
  };
}

/**
 * 同步响应负载
 */
interface SyncResponsePayload {
  // 同步状态
  status: 'success' | 'no_changes' | 'partial' | 'error';
  
  // 服务器状态向量
  serverStateVector: Uint8Array;
  
  // 更新数据
  updates: UpdateData[];
  
  // 缺失的数据（部分同步时）
  missingData?: MissingDataInfo;
}

/**
 * 更新数据
 */
interface UpdateData {
  // 更新类型
  type: 'diff' | 'full' | 'patch';
  
  // 更新内容
  data: Uint8Array;
  
  // 更新范围
  scope: string;
  
  // 更新大小
  size: number;
}

/**
 * 缺失数据信息
 */
interface MissingDataInfo {
  // 缺失的元素ID
  missingElements: string[];
  
  // 缺失的图层ID
  missingLayers: string[];
  
  // 获取完整数据的方式
  retrievalMethod: 'request_full' | 'lazy_load';
}
```

### 2.2 增量更新协议

```typescript
// ============================================
// 增量更新协议
// ============================================

/**
 * 增量更新消息
 */
interface UpdatePayload {
  // 更新ID
  updateId: string;
  
  // 更新类型
  updateType: UpdateType;
  
  // 更新操作列表
  operations: Operation[];
  
  // 更新前的状态向量
  beforeStateVector: Uint8Array;
  
  // 更新后的状态向量
  afterStateVector: Uint8Array;
  
  // 依赖的更新
  dependencies: string[];
  
  // 更新时间戳
  timestamp: number;
}

/**
 * 更新类型
 */
enum UpdateType {
  // 本地操作
  LOCAL = 'local',
  
  // 远程操作
  REMOTE = 'remote',
  
  // 批量操作
  BATCH = 'batch',
  
  // 历史操作
  HISTORY = 'history',
  
  // 合并操作
  MERGED = 'merged'
}

/**
 * 操作定义
 */
interface Operation {
  // 操作ID
  id: string;
  
  // 操作类型
  type: OperationType;
  
  // 目标元素
  target: {
    elementId: string;
    layerId?: string;
  };
  
  // 操作数据
  data: OperationData;
  
  // 操作前的状态（用于撤销）
  previousState?: any;
  
  // 操作元数据
  metadata: OperationMetadata;
}

/**
 * 操作数据类型
 */
type OperationData = 
  | CreateOperationData
  | UpdateOperationData
  | DeleteOperationData
  | TransformOperationData
  | PropertyOperationData;

/**
 * 创建操作数据
 */
interface CreateOperationData {
  // 元素类型
  elementType: GeometryType;
  
  // 初始属性
  initialProperties: Partial<GeometryElement>;
  
  // 父元素ID（用于嵌套）
  parentId?: string;
}

/**
 * 更新操作数据
 */
interface UpdateOperationData {
  // 更新路径
  path: string;
  
  // 新值
  value: any;
  
  // 旧值（用于撤销）
  oldValue: any;
  
  // 更新模式
  mode: 'set' | 'add' | 'remove' | 'merge';
}

/**
 * 删除操作数据
 */
interface DeleteOperationData {
  // 被删除元素的完整数据（用于恢复）
  deletedData: GeometryElement;
  
  // 级联删除的子元素
  cascadeDeleted: string[];
}

/**
 * 变换操作数据
 */
interface TransformOperationData {
  // 变换类型
  transformType: 'translate' | 'rotate' | 'scale' | 'matrix';
  
  // 变换参数
  parameters: TransformParameters;
  
  // 变换中心点
  center?: Point3D;
  
  // 是否相对变换
  relative: boolean;
}

/**
 * 变换参数
 */
interface TransformParameters {
  // 平移
  translate?: { x: number; y: number; z: number };
  
  // 旋转（欧拉角或四元数）
  rotate?: { x: number; y: number; z: number } | Quaternion;
  
  // 缩放
  scale?: { x: number; y: number; z: number };
  
  // 矩阵
  matrix?: TransformMatrix;
}

/**
 * 四元数
 */
interface Quaternion {
  x: number;
  y: number;
  z: number;
  w: number;
}

/**
 * 属性操作数据
 */
interface PropertyOperationData {
  // 属性集名称
  propertySet: string;
  
  // 属性名
  propertyName: string;
  
  // 属性值
  value: PropertyValue;
  
  // 旧值
  oldValue?: PropertyValue;
}

// ============================================
// 增量更新协议流程
// ============================================

/**
 * 增量更新协议状态机
 */
enum UpdateProtocolState {
  IDLE = 'idle',
  PENDING = 'pending',
  SENDING = 'sending',
  WAITING_ACK = 'waiting_ack',
  CONFIRMED = 'confirmed',
  FAILED = 'failed'
}

/**
 * 增量更新管理器
 */
interface IncrementalUpdateManager {
  // 当前状态
  state: UpdateProtocolState;
  
  // 待发送队列
  pendingQueue: Operation[];
  
  // 发送中操作
  inFlight: Map<string, InFlightOperation>;
  
  // 已确认操作
  confirmed: Set<string>;
  
  // 发送操作
  send(operations: Operation[]): Promise<void>;
  
  // 接收操作
  receive(update: UpdatePayload): Promise<void>;
  
  // 确认操作
  acknowledge(updateId: string): void;
  
  // 重试操作
  retry(updateId: string): Promise<void>;
}

/**
 * 发送中操作
 */
interface InFlightOperation {
  operation: Operation;
  sentAt: number;
  retryCount: number;
  timeoutId: number;
}

// ============================================
// 更新确认消息
// ============================================

/**
 * 更新确认负载
 */
interface UpdateAckPayload {
  // 确认的更新ID
  updateId: string;
  
  // 确认类型
  ackType: 'full' | 'partial' | 'rejected';
  
  // 服务器状态向量
  serverStateVector: Uint8Array;
  
  // 拒绝原因
  rejectReason?: string;
  
  // 冲突信息
  conflicts?: ConflictInfo[];
}

/**
 * 冲突信息
 */
interface ConflictInfo {
  // 冲突操作ID
  operationId: string;
  
  // 冲突类型
  type: ConflictType;
  
  // 冲突详情
  details: any;
}
```

### 2.3 全量同步协议

```typescript
// ============================================
// 全量同步协议
// ============================================

/**
 * 全量同步消息
 */
interface FullSyncPayload {
  // 同步ID
  syncId: string;
  
  // 同步类型
  syncType: 'initial' | 'recovery' | 'forced';
  
  // 文档快照
  snapshot: DocumentSnapshot;
  
  // 状态向量
  stateVector: Uint8Array;
  
  // 分块信息（大文档分块传输）
  chunks?: ChunkInfo;
}

/**
 * 文档快照
 */
interface DocumentSnapshot {
  // 文档元数据
  metadata: DocumentMetadata;
  
  // 图层数据
  layers: LayerSnapshot[];
  
  // 元素数据
  elements: ElementSnapshot[];
  
  // 属性数据
  properties: PropertySnapshot[];
  
  // 用户状态
  userStates: UserStateSnapshot[];
  
  // 历史记录（可选）
  history?: HistorySnapshot;
}

/**
 * 图层快照
 */
interface LayerSnapshot {
  id: string;
  name: string;
  visible: boolean;
  locked: boolean;
  opacity: number;
  order: number;
  elementIds: string[];
}

/**
 * 元素快照
 */
interface ElementSnapshot {
  id: string;
  type: GeometryType;
  layerId: string;
  data: any;
  versionVector: VersionVector;
}

/**
 * 属性快照
 */
interface PropertySnapshot {
  elementId: string;
  propertySets: any;
  customProperties: any;
}

/**
 * 用户状态快照
 */
interface UserStateSnapshot {
  clientId: string;
  userId: string;
  cursor?: Point3D;
  selection: string[];
  viewport: any;
}

/**
 * 历史记录快照
 */
interface HistorySnapshot {
  operations: Operation[];
  maxSize: number;
  currentIndex: number;
}

/**
 * 分块信息
 */
interface ChunkInfo {
  // 总分块数
  totalChunks: number;
  
  // 当前块索引
  currentChunk: number;
  
  // 块大小
  chunkSize: number;
  
  // 校验和
  checksum: string;
}

// ============================================
// 全量同步确认
// ============================================

/**
 * 全量同步确认负载
 */
interface FullSyncAckPayload {
  // 同步ID
  syncId: string;
  
  // 状态
  status: 'success' | 'partial' | 'failed';
  
  // 接收的状态向量
  receivedStateVector: Uint8Array;
  
  // 差异（如果有）
  diff?: Uint8Array;
  
  // 失败原因
  error?: string;
  
  // 需要的额外块
  requestedChunks?: number[];
}

// ============================================
// 全量同步管理器
// ============================================

/**
 * 全量同步管理器
 */
interface FullSyncManager {
  // 当前同步状态
  state: FullSyncState;
  
  // 同步配置
  config: FullSyncConfig;
  
  // 执行全量同步
  performFullSync(type: 'initial' | 'recovery' | 'forced'): Promise<void>;
  
  // 处理分块
  handleChunk(chunk: ChunkData): void;
  
  // 验证同步
  verifySync(): boolean;
}

/**
 * 全量同步状态
 */
enum FullSyncState {
  IDLE = 'idle',
  REQUESTING = 'requesting',
  RECEIVING = 'receiving',
  PROCESSING = 'processing',
  VERIFYING = 'verifying',
  COMPLETED = 'completed',
  FAILED = 'failed'
}

/**
 * 全量同步配置
 */
interface FullSyncConfig {
  // 分块大小（字节）
  chunkSize: number;
  
  // 超时时间（毫秒）
  timeout: number;
  
  // 最大重试次数
  maxRetries: number;
  
  // 压缩启用
  enableCompression: boolean;
  
  // 包含历史记录
  includeHistory: boolean;
  
  // 历史记录最大数量
  maxHistoryItems: number;
}
```

### 2.4 冲突解决协议

```typescript
// ============================================
// 冲突解决协议
// ============================================

/**
 * 冲突检测消息
 */
interface ConflictPayload {
  // 冲突ID
  conflictId: string;
  
  // 冲突类型
  type: ConflictType;
  
  // 冲突元素
  elementId: string;
  
  // 冲突的操作
  operations: ConflictingOperation[];
  
  // 冲突详情
  details: ConflictDetails;
  
  // 建议的解决方案
  suggestions: ConflictSuggestion[];
}

/**
 * 冲突操作
 */
interface ConflictingOperation {
  // 操作ID
  operationId: string;
  
  // 操作来源
  source: {
    clientId: string;
    userId: string;
    userName: string;
  };
  
  // 操作时间
  timestamp: number;
  
  // 操作内容
  operation: Operation;
  
  // 操作影响
  impact: OperationImpact;
}

/**
 * 操作影响
 */
interface OperationImpact {
  // 影响的属性
  affectedProperties: string[];
  
  // 影响范围
  scope: 'element' | 'layer' | 'document';
  
  // 严重程度
  severity: 'low' | 'medium' | 'high' | 'critical';
}

/**
 * 冲突详情
 */
interface ConflictDetails {
  // 属性冲突详情
  propertyConflicts?: PropertyConflictDetail[];
  
  // 结构冲突详情
  structureConflicts?: StructureConflictDetail[];
  
  // 依赖冲突详情
  dependencyConflicts?: DependencyConflictDetail[];
}

/**
 * 属性冲突详情
 */
interface PropertyConflictDetail {
  // 属性路径
  propertyPath: string;
  
  // 冲突值
  values: {
    clientId: string;
    value: any;
  }[];
  
  // 属性类型
  dataType: PropertyDataType;
}

/**
 * 结构冲突详情
 */
interface StructureConflictDetail {
  // 结构变化类型
  changeType: 'add' | 'remove' | 'reorder' | 'reparent';
  
  // 涉及的元素
  elements: string[];
  
  // 结构变化描述
  description: string;
}

/**
 * 依赖冲突详情
 */
interface DependencyConflictDetail {
  // 依赖的元素
  dependencyElement: string;
  
  // 依赖类型
  dependencyType: string;
  
  // 冲突描述
  description: string;
}

/**
 * 冲突建议
 */
interface ConflictSuggestion {
  // 建议ID
  id: string;
  
  // 建议类型
  type: 'auto_merge' | 'last_write_wins' | 'manual_merge' | 'revert';
  
  // 建议描述
  description: string;
  
  // 预览结果
  preview: any;
  
  // 置信度
  confidence: number;
}

/**
 * 冲突解决消息
 */
interface ConflictResolvedPayload {
  // 冲突ID
  conflictId: string;
  
  // 解决方式
  resolution: ConflictResolution;
  
  // 解决结果
  result: any;
  
  // 通知的客户端
  notifiedClients: string[];
}

// ============================================
// 冲突解决管理器
// ============================================

/**
 * 冲突解决管理器
 */
interface ConflictResolutionManager {
  // 活跃冲突
  activeConflicts: Map<string, ConflictMetadata>;
  
  // 冲突历史
  conflictHistory: ConflictMetadata[];
  
  // 检测冲突
  detectConflict(operations: Operation[]): ConflictMetadata | null;
  
  // 自动解决
  autoResolve(conflict: ConflictMetadata): ConflictResolution | null;
  
  // 手动解决
  manualResolve(conflictId: string, resolution: ConflictResolution): void;
  
  // 通知冲突
  notifyConflict(conflict: ConflictMetadata): void;
  
  // 应用解决
  applyResolution(conflictId: string): void;
}
```

---

## 3. WebSocket网关详细设计

### 3.1 连接管理实现

```typescript
// ============================================
// 连接管理器
// ============================================

/**
 * WebSocket连接管理器
 */
class ConnectionManager {
  // 活跃连接映射
  private connections: Map<string, ClientConnection> = new Map();
  
  // 用户到连接的映射（一个用户可能有多个连接）
  private userConnections: Map<string, Set<string>> = new Map();
  
  // 文档到连接的映射
  private documentConnections: Map<string, Set<string>> = new Map();
  
  // 连接配置
  private config: ConnectionConfig;
  
  // 统计信息
  private stats: ConnectionStats;
  
  constructor(config: ConnectionConfig) {
    this.config = config;
    this.stats = {
      totalConnections: 0,
      activeConnections: 0,
      peakConnections: 0,
      totalMessages: 0,
      totalBytes: 0
    };
  }
  
  /**
   * 注册新连接
   */
  async registerConnection(
    ws: WebSocket,
    request: ConnectionRequest
  ): Promise<ClientConnection> {
    const clientId = this.generateClientId();
    const connection: ClientConnection = {
      id: clientId,
      userId: request.userId,
      userName: request.userName,
      documentId: request.documentId,
      ws: ws,
      state: ConnectionState.CONNECTING,
      connectedAt: Date.now(),
      lastActivity: Date.now(),
      messageCount: 0,
      byteCount: 0,
      capabilities: request.capabilities,
      subscriptions: new Set(),
      metadata: {}
    };
    
    // 存储连接
    this.connections.set(clientId, connection);
    
    // 更新用户连接映射
    if (!this.userConnections.has(request.userId)) {
      this.userConnections.set(request.userId, new Set());
    }
    this.userConnections.get(request.userId)!.add(clientId);
    
    // 更新文档连接映射
    if (!this.documentConnections.has(request.documentId)) {
      this.documentConnections.set(request.documentId, new Set());
    }
    this.documentConnections.get(request.documentId)!.add(clientId);
    
    // 更新统计
    this.stats.totalConnections++;
    this.stats.activeConnections++;
    this.stats.peakConnections = Math.max(
      this.stats.peakConnections,
      this.stats.activeConnections
    );
    
    // 设置连接事件处理
    this.setupConnectionHandlers(connection);
    
    connection.state = ConnectionState.CONNECTED;
    
    return connection;
  }
  
  /**
   * 注销连接
   */
  async unregisterConnection(clientId: string, reason?: string): Promise<void> {
    const connection = this.connections.get(clientId);
    if (!connection) return;
    
    connection.state = ConnectionState.DISCONNECTING;
    
    // 清理用户连接映射
    const userConns = this.userConnections.get(connection.userId);
    if (userConns) {
      userConns.delete(clientId);
      if (userConns.size === 0) {
        this.userConnections.delete(connection.userId);
      }
    }
    
    // 清理文档连接映射
    const docConns = this.documentConnections.get(connection.documentId);
    if (docConns) {
      docConns.delete(clientId);
      if (docConns.size === 0) {
        this.documentConnections.delete(connection.documentId);
      }
    }
    
    // 关闭WebSocket
    if (connection.ws.readyState === WebSocket.OPEN) {
      connection.ws.close(1000, reason || 'Normal closure');
    }
    
    // 移除连接
    this.connections.delete(clientId);
    
    // 更新统计
    this.stats.activeConnections--;
    
    // 广播用户离开
    this.broadcastUserLeft(connection);
  }
  
  /**
   * 获取文档的所有连接
   */
  getDocumentConnections(documentId: string): ClientConnection[] {
    const clientIds = this.documentConnections.get(documentId);
    if (!clientIds) return [];
    
    return Array.from(clientIds)
      .map(id => this.connections.get(id))
      .filter((conn): conn is ClientConnection => conn !== undefined);
  }
  
  /**
   * 获取用户的所有连接
   */
  getUserConnections(userId: string): ClientConnection[] {
    const clientIds = this.userConnections.get(userId);
    if (!clientIds) return [];
    
    return Array.from(clientIds)
      .map(id => this.connections.get(id))
      .filter((conn): conn is ClientConnection => conn !== undefined);
  }
  
  /**
   * 生成客户端ID
   */
  private generateClientId(): string {
    return `client_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 设置连接事件处理
   */
  private setupConnectionHandlers(connection: ClientConnection): void {
    const ws = connection.ws;
    
    // 消息处理
    ws.on('message', (data: WebSocket.Data) => {
      this.handleMessage(connection, data);
    });
    
    // 关闭处理
    ws.on('close', (code: number, reason: string) => {
      this.handleClose(connection, code, reason);
    });
    
    // 错误处理
    ws.on('error', (error: Error) => {
      this.handleError(connection, error);
    });
    
    // 设置超时检查
    this.setupTimeoutCheck(connection);
  }
  
  /**
   * 处理消息
   */
  private async handleMessage(
    connection: ClientConnection,
    data: WebSocket.Data
  ): Promise<void> {
    try {
      connection.lastActivity = Date.now();
      connection.messageCount++;
      
      // 解析消息
      const message = this.parseMessage(data);
      
      // 更新统计
      this.stats.totalMessages++;
      this.stats.totalBytes += Buffer.byteLength(data.toString());
      
      // 处理消息
      await this.processMessage(connection, message);
    } catch (error) {
      this.sendError(connection, 'MESSAGE_PARSE_ERROR', error.message);
    }
  }
  
  /**
   * 处理连接关闭
   */
  private handleClose(
    connection: ClientConnection,
    code: number,
    reason: string
  ): void {
    console.log(`Connection ${connection.id} closed: ${code} - ${reason}`);
    this.unregisterConnection(connection.id, reason);
  }
  
  /**
   * 处理错误
   */
  private handleError(connection: ClientConnection, error: Error): void {
    console.error(`Connection ${connection.id} error:`, error);
    this.sendError(connection, 'CONNECTION_ERROR', error.message);
  }
  
  /**
   * 设置超时检查
   */
  private setupTimeoutCheck(connection: ClientConnection): void {
    const checkInterval = setInterval(() => {
      const inactiveTime = Date.now() - connection.lastActivity;
      
      if (inactiveTime > this.config.inactivityTimeout) {
        console.log(`Connection ${connection.id} timed out`);
        this.unregisterConnection(connection.id, 'Inactivity timeout');
        clearInterval(checkInterval);
      }
    }, 30000); // 每30秒检查一次
  }
  
  /**
   * 解析消息
   */
  private parseMessage(data: WebSocket.Data): SyncMessage {
    const text = data.toString();
    return JSON.parse(text) as SyncMessage;
  }
  
  /**
   * 处理消息
   */
  private async processMessage(
    connection: ClientConnection,
    message: SyncMessage
  ): Promise<void> {
    // 根据消息类型路由到相应的处理器
    switch (message.type) {
      case MessageType.PING:
        this.handlePing(connection);
        break;
      case MessageType.SYNC_REQUEST:
        await this.handleSyncRequest(connection, message.payload as SyncRequestPayload);
        break;
      case MessageType.UPDATE:
        await this.handleUpdate(connection, message.payload as UpdatePayload);
        break;
      // ... 其他消息类型
      default:
        this.sendError(connection, 'UNKNOWN_MESSAGE_TYPE', `Unknown type: ${message.type}`);
    }
  }
  
  /**
   * 发送错误消息
   */
  private sendError(
    connection: ClientConnection,
    code: string,
    message: string
  ): void {
    const errorMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.ERROR,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        code,
        message,
        timestamp: Date.now()
      } as ErrorPayload
    };
    
    this.sendMessage(connection, errorMessage);
  }
  
  /**
   * 发送消息
   */
  sendMessage(connection: ClientConnection, message: SyncMessage): void {
    if (connection.ws.readyState === WebSocket.OPEN) {
      const data = JSON.stringify(message);
      connection.ws.send(data);
      connection.byteCount += Buffer.byteLength(data);
    }
  }
  
  /**
   * 广播消息到文档
   */
  broadcastToDocument(
    documentId: string,
    message: SyncMessage,
    excludeClientId?: string
  ): void {
    const connections = this.getDocumentConnections(documentId);
    
    for (const conn of connections) {
      if (conn.id !== excludeClientId && conn.state === ConnectionState.CONNECTED) {
        this.sendMessage(conn, message);
      }
    }
  }
  
  /**
   * 生成消息ID
   */
  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  // ... 其他方法
}

// ============================================
// 类型定义
// ============================================

/**
 * 客户端连接
 */
interface ClientConnection {
  id: string;
  userId: string;
  userName: string;
  documentId: string;
  ws: WebSocket;
  state: ConnectionState;
  connectedAt: number;
  lastActivity: number;
  messageCount: number;
  byteCount: number;
  capabilities: ClientCapabilities;
  subscriptions: Set<string>;
  metadata: Record<string, any>;
}

/**
 * 连接状态
 */
enum ConnectionState {
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
  DISCONNECTING = 'disconnecting',
  DISCONNECTED = 'disconnected'
}

/**
 * 连接配置
 */
interface ConnectionConfig {
  maxConnections: number;
  maxConnectionsPerUser: number;
  maxConnectionsPerDocument: number;
  inactivityTimeout: number;
  heartbeatInterval: number;
  messageSizeLimit: number;
  rateLimit: {
    messagesPerSecond: number;
    burstSize: number;
  };
}

/**
 * 连接请求
 */
interface ConnectionRequest {
  userId: string;
  userName: string;
  documentId: string;
  capabilities: ClientCapabilities;
  authToken: string;
}

/**
 * 连接统计
 */
interface ConnectionStats {
  totalConnections: number;
  activeConnections: number;
  peakConnections: number;
  totalMessages: number;
  totalBytes: number;
}
```

### 3.2 消息路由实现

```typescript
// ============================================
// 消息路由器
// ============================================

/**
 * 消息路由器
 */
class MessageRouter {
  // 路由表
  private routes: Map<MessageType, MessageHandler> = new Map();
  
  // 中间件链
  private middlewares: Middleware[] = [];
  
  // 消息转换器
  private transformers: Map<string, MessageTransformer> = new Map();
  
  // 消息过滤器
  private filters: MessageFilter[] = [];
  
  constructor() {
    this.setupDefaultRoutes();
  }
  
  /**
   * 注册路由
   */
  registerRoute(type: MessageType, handler: MessageHandler): void {
    this.routes.set(type, handler);
  }
  
  /**
   * 注册中间件
   */
  use(middleware: Middleware): void {
    this.middlewares.push(middleware);
  }
  
  /**
   * 注册过滤器
   */
  addFilter(filter: MessageFilter): void {
    this.filters.push(filter);
  }
  
  /**
   * 路由消息
   */
  async route(
    connection: ClientConnection,
    message: SyncMessage
  ): Promise<void> {
    // 执行中间件链
    const context: MessageContext = {
      connection,
      message,
      metadata: {}
    };
    
    // 前置中间件
    for (const middleware of this.middlewares) {
      const result = await middleware.before(context);
      if (result === false) {
        // 中间件阻止消息处理
        return;
      }
    }
    
    // 过滤检查
    for (const filter of this.filters) {
      if (!filter.allow(context)) {
        this.handleFilteredMessage(context, filter);
        return;
      }
    }
    
    // 转换消息
    const transformedMessage = await this.transformMessage(message);
    context.message = transformedMessage;
    
    // 查找处理器
    const handler = this.routes.get(message.type);
    if (!handler) {
      throw new Error(`No handler for message type: ${message.type}`);
    }
    
    // 执行处理器
    try {
      await handler.handle(context);
    } catch (error) {
      await this.handleError(context, error);
    }
    
    // 后置中间件
    for (const middleware of this.middlewares) {
      await middleware.after(context);
    }
  }
  
  /**
   * 设置默认路由
   */
  private setupDefaultRoutes(): void {
    // 连接管理
    this.registerRoute(MessageType.CONNECT, new ConnectHandler());
    this.registerRoute(MessageType.DISCONNECT, new DisconnectHandler());
    
    // 同步
    this.registerRoute(MessageType.SYNC_REQUEST, new SyncRequestHandler());
    this.registerRoute(MessageType.SYNC_RESPONSE, new SyncResponseHandler());
    
    // 更新
    this.registerRoute(MessageType.UPDATE, new UpdateHandler());
    this.registerRoute(MessageType.UPDATE_ACK, new UpdateAckHandler());
    
    // 全量同步
    this.registerRoute(MessageType.FULL_SYNC, new FullSyncHandler());
    this.registerRoute(MessageType.FULL_SYNC_ACK, new FullSyncAckHandler());
    
    // 心跳
    this.registerRoute(MessageType.PING, new PingHandler());
    this.registerRoute(MessageType.PONG, new PongHandler());
    
    // 冲突
    this.registerRoute(MessageType.CONFLICT_DETECTED, new ConflictDetectedHandler());
    this.registerRoute(MessageType.CONFLICT_RESOLVED, new ConflictResolvedHandler());
    
    // 状态广播
    this.registerRoute(MessageType.STATE_BROADCAST, new StateBroadcastHandler());
  }
  
  /**
   * 转换消息
   */
  private async transformMessage(message: SyncMessage): Promise<SyncMessage> {
    let transformed = message;
    
    // 应用所有适用的转换器
    for (const [key, transformer] of this.transformers) {
      if (transformer.appliesTo(message)) {
        transformed = await transformer.transform(transformed);
      }
    }
    
    return transformed;
  }
  
  /**
   * 处理过滤的消息
   */
  private handleFilteredMessage(
    context: MessageContext,
    filter: MessageFilter
  ): void {
    // 发送过滤通知
    const response: SyncMessage = {
      messageId: generateId(),
      type: MessageType.ERROR,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        code: 'MESSAGE_FILTERED',
        message: `Message filtered by: ${filter.name}`,
        details: filter.getFilterReason(context)
      } as ErrorPayload
    };
    
    context.connection.ws.send(JSON.stringify(response));
  }
  
  /**
   * 处理错误
   */
  private async handleError(context: MessageContext, error: Error): Promise<void> {
    console.error('Message handling error:', error);
    
    const response: SyncMessage = {
      messageId: generateId(),
      type: MessageType.ERROR,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        code: 'HANDLER_ERROR',
        message: error.message,
        stack: process.env.NODE_ENV === 'development' ? error.stack : undefined
      } as ErrorPayload
    };
    
    context.connection.ws.send(JSON.stringify(response));
  }
}

// ============================================
// 消息处理器基类
// ============================================

/**
 * 消息处理器接口
 */
interface MessageHandler {
  handle(context: MessageContext): Promise<void>;
}

/**
 * 连接处理器
 */
class ConnectHandler implements MessageHandler {
  async handle(context: MessageContext): Promise<void> {
    const payload = context.message.payload as ConnectPayload;
    
    // 验证认证令牌
    const user = await this.authenticate(payload.authToken);
    if (!user) {
      throw new Error('Authentication failed');
    }
    
    // 更新连接信息
    context.connection.userId = user.id;
    context.connection.userName = user.name;
    context.connection.capabilities = payload.clientInfo.capabilities;
    
    // 发送连接确认
    const ackMessage: SyncMessage = {
      messageId: generateId(),
      type: MessageType.CONNECT_ACK,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        clientId: context.connection.id,
        status: 'success',
        serverInfo: {
          serverId: 'server-1',
          version: '1.0.0',
          maxClients: 100,
          currentClients: 50
        }
      } as ConnectAckPayload
    };
    
    context.connection.ws.send(JSON.stringify(ackMessage));
    
    // 广播用户加入
    await this.broadcastUserJoined(context.connection);
  }
  
  private async authenticate(token: string): Promise<{ id: string; name: string } | null> {
    // 实现认证逻辑
    return { id: 'user-1', name: 'User 1' };
  }
  
  private async broadcastUserJoined(connection: ClientConnection): Promise<void> {
    // 实现广播逻辑
  }
}

/**
 * 同步请求处理器
 */
class SyncRequestHandler implements MessageHandler {
  async handle(context: MessageContext): Promise<void> {
    const payload = context.message.payload as SyncRequestPayload;
    
    // 获取文档状态
    const document = await this.getDocument(context.connection.documentId);
    
    // 计算差异
    const diff = await this.computeDiff(
      document.stateVector,
      payload.clientStateVector
    );
    
    // 发送同步响应
    const response: SyncMessage = {
      messageId: generateId(),
      type: MessageType.SYNC_RESPONSE,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: document.versionVector,
      payload: {
        status: diff.length > 0 ? 'success' : 'no_changes',
        serverStateVector: document.stateVector,
        updates: diff.map(update => ({
          type: 'diff',
          data: update,
          scope: 'document',
          size: update.length
        }))
      } as SyncResponsePayload
    };
    
    context.connection.ws.send(JSON.stringify(response));
  }
  
  private async getDocument(documentId: string): Promise<any> {
    // 实现获取文档逻辑
    return {};
  }
  
  private async computeDiff(
    serverVector: Uint8Array,
    clientVector: Uint8Array
  ): Promise<Uint8Array[]> {
    // 实现差异计算逻辑
    return [];
  }
}

/**
 * 更新处理器
 */
class UpdateHandler implements MessageHandler {
  async handle(context: MessageContext): Promise<void> {
    const payload = context.message.payload as UpdatePayload;
    
    // 验证更新
    const validation = await this.validateUpdate(payload);
    if (!validation.valid) {
      throw new Error(`Invalid update: ${validation.reason}`);
    }
    
    // 应用更新到文档
    const result = await this.applyUpdate(context.connection.documentId, payload);
    
    // 广播更新到其他客户端
    await this.broadcastUpdate(context.connection, payload);
    
    // 发送确认
    const ackMessage: SyncMessage = {
      messageId: generateId(),
      type: MessageType.UPDATE_ACK,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: result.newVersionVector,
      payload: {
        updateId: payload.updateId,
        ackType: 'full',
        serverStateVector: result.stateVector
      } as UpdateAckPayload
    };
    
    context.connection.ws.send(JSON.stringify(ackMessage));
  }
  
  private async validateUpdate(payload: UpdatePayload): Promise<ValidationResult> {
    // 实现验证逻辑
    return { valid: true };
  }
  
  private async applyUpdate(
    documentId: string,
    payload: UpdatePayload
  ): Promise<UpdateResult> {
    // 实现应用更新逻辑
    return {
      newVersionVector: {},
      stateVector: new Uint8Array()
    };
  }
  
  private async broadcastUpdate(
    sourceConnection: ClientConnection,
    payload: UpdatePayload
  ): Promise<void> {
    // 实现广播逻辑
  }
}

/**
 * 心跳处理器
 */
class PingHandler implements MessageHandler {
  async handle(context: MessageContext): Promise<void> {
    const pongMessage: SyncMessage = {
      messageId: generateId(),
      type: MessageType.PONG,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        serverTime: Date.now()
      }
    };
    
    context.connection.ws.send(JSON.stringify(pongMessage));
  }
}

// ============================================
// 辅助类型
// ============================================

interface MessageContext {
  connection: ClientConnection;
  message: SyncMessage;
  metadata: Record<string, any>;
}

interface Middleware {
  before(context: MessageContext): Promise<boolean | void>;
  after(context: MessageContext): Promise<void>;
}

interface MessageFilter {
  name: string;
  allow(context: MessageContext): boolean;
  getFilterReason(context: MessageContext): string;
}

interface MessageTransformer {
  appliesTo(message: SyncMessage): boolean;
  transform(message: SyncMessage): Promise<SyncMessage>;
}

interface ValidationResult {
  valid: boolean;
  reason?: string;
}

interface UpdateResult {
  newVersionVector: VersionVector;
  stateVector: Uint8Array;
}

function generateId(): string {
  return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}
```

### 3.3 心跳机制实现

```typescript
// ============================================
// 心跳管理器
// ============================================

/**
 * 心跳管理器
 */
class HeartbeatManager {
  // 连接的心跳状态
  private heartbeatStates: Map<string, HeartbeatState> = new Map();
  
  // 配置
  private config: HeartbeatConfig;
  
  // 定时器
  private checkInterval: NodeJS.Timeout | null = null;
  
  // 连接管理器引用
  private connectionManager: ConnectionManager;
  
  constructor(config: HeartbeatConfig, connectionManager: ConnectionManager) {
    this.config = config;
    this.connectionManager = connectionManager;
  }
  
  /**
   * 启动心跳管理
   */
  start(): void {
    // 启动定期检查
    this.checkInterval = setInterval(() => {
      this.checkHeartbeats();
    }, this.config.checkInterval);
    
    console.log('Heartbeat manager started');
  }
  
  /**
   * 停止心跳管理
   */
  stop(): void {
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
      this.checkInterval = null;
    }
    
    console.log('Heartbeat manager stopped');
  }
  
  /**
   * 注册连接的心跳
   */
  registerHeartbeat(clientId: string): void {
    this.heartbeatStates.set(clientId, {
      clientId,
      lastPingTime: Date.now(),
      lastPongTime: Date.now(),
      missedPongs: 0,
      latency: 0,
      status: 'healthy'
    });
  }
  
  /**
   * 注销连接的心跳
   */
  unregisterHeartbeat(clientId: string): void {
    this.heartbeatStates.delete(clientId);
  }
  
  /**
   * 处理ping消息
   */
  handlePing(clientId: string): void {
    const state = this.heartbeatStates.get(clientId);
    if (state) {
      state.lastPingTime = Date.now();
      state.missedPongs = 0;
    }
  }
  
  /**
   * 处理pong消息
   */
  handlePong(clientId: string, serverTime: number): void {
    const state = this.heartbeatStates.get(clientId);
    if (state) {
      const now = Date.now();
      state.lastPongTime = now;
      state.missedPongs = 0;
      
      // 计算延迟（往返时间的一半）
      state.latency = (now - serverTime) / 2;
      
      // 更新状态
      this.updateHeartbeatStatus(state);
    }
  }
  
  /**
   * 发送ping到客户端
   */
  async sendPing(connection: ClientConnection): Promise<void> {
    const pingMessage: SyncMessage = {
      messageId: generateId(),
      type: MessageType.PING,
      sender: { clientId: 'server', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {}
    };
    
    this.connectionManager.sendMessage(connection, pingMessage);
  }
  
  /**
   * 检查所有心跳
   */
  private checkHeartbeats(): void {
    const now = Date.now();
    
    for (const [clientId, state] of this.heartbeatStates) {
      const timeSinceLastPong = now - state.lastPongTime;
      
      // 检查是否超时
      if (timeSinceLastPong > this.config.pongTimeout) {
        state.missedPongs++;
        
        if (state.missedPongs >= this.config.maxMissedPongs) {
          // 连接可能已断开
          this.handleDeadConnection(clientId, state);
        } else {
          // 发送ping检查
          const connection = this.getConnection(clientId);
          if (connection) {
            this.sendPing(connection);
          }
        }
      }
      
      // 更新状态
      this.updateHeartbeatStatus(state);
    }
  }
  
  /**
   * 更新心跳状态
   */
  private updateHeartbeatStatus(state: HeartbeatState): void {
    const latency = state.latency;
    
    if (latency < 100) {
      state.status = 'excellent';
    } else if (latency < 300) {
      state.status = 'good';
    } else if (latency < 500) {
      state.status = 'fair';
    } else {
      state.status = 'poor';
    }
  }
  
  /**
   * 处理可能断开的连接
   */
  private handleDeadConnection(clientId: string, state: HeartbeatState): void {
    console.warn(`Connection ${clientId} appears dead, closing...`);
    
    // 注销心跳
    this.unregisterHeartbeat(clientId);
    
    // 关闭连接
    this.connectionManager.unregisterConnection(clientId, 'Heartbeat timeout');
  }
  
  /**
   * 获取连接
   */
  private getConnection(clientId: string): ClientConnection | undefined {
    // 从连接管理器获取
    return undefined; // 实际实现中返回真实连接
  }
  
  /**
   * 获取心跳统计
   */
  getStats(): HeartbeatStats {
    const states = Array.from(this.heartbeatStates.values());
    
    return {
      totalConnections: states.length,
      healthyConnections: states.filter(s => s.status === 'healthy').length,
      averageLatency: states.reduce((sum, s) => sum + s.latency, 0) / states.length || 0,
      maxLatency: Math.max(...states.map(s => s.latency), 0),
      minLatency: Math.min(...states.map(s => s.latency), Infinity)
    };
  }
}

// ============================================
// 心跳类型定义
// ============================================

interface HeartbeatState {
  clientId: string;
  lastPingTime: number;
  lastPongTime: number;
  missedPongs: number;
  latency: number;
  status: 'excellent' | 'good' | 'fair' | 'poor' | 'dead' | 'healthy';
}

interface HeartbeatConfig {
  checkInterval: number;      // 检查间隔（毫秒）
  pingInterval: number;       // ping发送间隔（毫秒）
  pongTimeout: number;        // pong超时时间（毫秒）
  maxMissedPongs: number;     // 最大允许丢失的pong数
}

interface HeartbeatStats {
  totalConnections: number;
  healthyConnections: number;
  averageLatency: number;
  maxLatency: number;
  minLatency: number;
}
```

### 3.4 断线重连实现

```typescript
// ============================================
// 重连管理器
// ============================================

/**
 * 重连管理器
 */
class ReconnectionManager {
  // 重连尝试记录
  private reconnectionAttempts: Map<string, ReconnectionAttempt> = new Map();
  
  // 会话状态缓存
  private sessionCache: Map<string, SessionState> = new Map();
  
  // 配置
  private config: ReconnectionConfig;
  
  // 连接管理器引用
  private connectionManager: ConnectionManager;
  
  constructor(config: ReconnectionConfig, connectionManager: ConnectionManager) {
    this.config = config;
    this.connectionManager = connectionManager;
  }
  
  /**
   * 保存会话状态
   */
  saveSessionState(connection: ClientConnection): void {
    const sessionState: SessionState = {
      clientId: connection.id,
      userId: connection.userId,
      documentId: connection.documentId,
      lastActivity: connection.lastActivity,
      subscriptions: Array.from(connection.subscriptions),
      viewport: connection.metadata.viewport,
      selection: connection.metadata.selection,
      versionVector: connection.metadata.versionVector,
      savedAt: Date.now()
    };
    
    this.sessionCache.set(connection.id, sessionState);
    
    // 设置过期清理
    setTimeout(() => {
      this.sessionCache.delete(connection.id);
    }, this.config.sessionTimeout);
  }
  
  /**
   * 恢复会话状态
   */
  async restoreSessionState(
    newClientId: string,
    oldClientId: string
  ): Promise<SessionState | null> {
    const sessionState = this.sessionCache.get(oldClientId);
    
    if (!sessionState) {
      return null;
    }
    
    // 检查会话是否过期
    if (Date.now() - sessionState.savedAt > this.config.sessionTimeout) {
      this.sessionCache.delete(oldClientId);
      return null;
    }
    
    // 迁移到新客户端ID
    const restoredState: SessionState = {
      ...sessionState,
      clientId: newClientId,
      restoredAt: Date.now()
    };
    
    // 清除旧会话
    this.sessionCache.delete(oldClientId);
    
    return restoredState;
  }
  
  /**
   * 处理重连请求
   */
  async handleReconnection(
    ws: WebSocket,
    request: ReconnectionRequest
  ): Promise<ReconnectionResult> {
    const { oldClientId, authToken, documentId } = request;
    
    // 验证认证
    const user = await this.authenticate(authToken);
    if (!user) {
      return {
        success: false,
        reason: 'Authentication failed'
      };
    }
    
    // 尝试恢复会话
    const sessionState = await this.restoreSessionState(
      this.generateClientId(),
      oldClientId
    );
    
    if (!sessionState) {
      // 会话已过期，需要全量同步
      return {
        success: true,
        sessionRestored: false,
        requiresFullSync: true,
        reason: 'Session expired'
      };
    }
    
    // 检查文档是否一致
    if (sessionState.documentId !== documentId) {
      return {
        success: true,
        sessionRestored: false,
        requiresFullSync: true,
        reason: 'Document changed'
      };
    }
    
    // 恢复成功
    return {
      success: true,
      sessionRestored: true,
      requiresFullSync: false,
      sessionState,
      missedUpdates: await this.getMissedUpdates(sessionState)
    };
  }
  
  /**
   * 获取错过的更新
   */
  private async getMissedUpdates(
    sessionState: SessionState
  ): Promise<UpdatePayload[]> {
    // 获取从lastActivity之后的所有更新
    const updates = await this.fetchUpdatesSince(
      sessionState.documentId,
      sessionState.lastActivity
    );
    
    return updates;
  }
  
  /**
   * 获取指定时间后的更新
   */
  private async fetchUpdatesSince(
    documentId: string,
    since: number
  ): Promise<UpdatePayload[]> {
    // 实现获取更新逻辑
    return [];
  }
  
  /**
   * 记录重连尝试
   */
  recordAttempt(clientId: string): void {
    const attempt = this.reconnectionAttempts.get(clientId);
    
    if (attempt) {
      attempt.count++;
      attempt.lastAttempt = Date.now();
    } else {
      this.reconnectionAttempts.set(clientId, {
        clientId,
        count: 1,
        firstAttempt: Date.now(),
        lastAttempt: Date.now()
      });
    }
  }
  
  /**
   * 检查是否允许重连
   */
  canReconnect(clientId: string): boolean {
    const attempt = this.reconnectionAttempts.get(clientId);
    
    if (!attempt) {
      return true;
    }
    
    // 检查重连次数
    if (attempt.count >= this.config.maxAttempts) {
      // 检查是否在冷却期内
      const cooldownEnd = attempt.lastAttempt + this.config.cooldownPeriod;
      if (Date.now() < cooldownEnd) {
        return false;
      }
      
      // 重置计数
      this.reconnectionAttempts.delete(clientId);
      return true;
    }
    
    return true;
  }
  
  /**
   * 获取重连延迟
   */
  getReconnectionDelay(attemptCount: number): number {
    // 指数退避
    const baseDelay = this.config.baseDelay;
    const maxDelay = this.config.maxDelay;
    
    const delay = Math.min(
      baseDelay * Math.pow(2, attemptCount),
      maxDelay
    );
    
    // 添加随机抖动
    const jitter = Math.random() * 0.1 * delay;
    
    return delay + jitter;
  }
  
  /**
   * 清理重连记录
   */
  cleanupAttempts(): void {
    const now = Date.now();
    
    for (const [clientId, attempt] of this.reconnectionAttempts) {
      // 清理过期的记录
      if (now - attempt.lastAttempt > this.config.attemptRecordTTL) {
        this.reconnectionAttempts.delete(clientId);
      }
    }
  }
  
  /**
   * 生成客户端ID
   */
  private generateClientId(): string {
    return `client_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 认证
   */
  private async authenticate(token: string): Promise<any> {
    // 实现认证逻辑
    return { id: 'user-1', name: 'User 1' };
  }
}

// ============================================
// 重连类型定义
// ============================================

interface ReconnectionConfig {
  maxAttempts: number;        // 最大重连次数
  baseDelay: number;          // 基础延迟（毫秒）
  maxDelay: number;           // 最大延迟（毫秒）
  sessionTimeout: number;     // 会话超时时间（毫秒）
  cooldownPeriod: number;     // 冷却期（毫秒）
  attemptRecordTTL: number;   // 重连记录保留时间（毫秒）
}

interface ReconnectionRequest {
  oldClientId: string;
  authToken: string;
  documentId: string;
  lastKnownVersionVector: VersionVector;
}

interface ReconnectionResult {
  success: boolean;
  sessionRestored?: boolean;
  requiresFullSync?: boolean;
  reason?: string;
  sessionState?: SessionState;
  missedUpdates?: UpdatePayload[];
}

interface ReconnectionAttempt {
  clientId: string;
  count: number;
  firstAttempt: number;
  lastAttempt: number;
}

interface SessionState {
  clientId: string;
  userId: string;
  documentId: string;
  lastActivity: number;
  subscriptions: string[];
  viewport?: any;
  selection?: string[];
  versionVector?: VersionVector;
  savedAt: number;
  restoredAt?: number;
}
```

---

## 4. 操作转换详细设计

### 4.1 操作定义

```typescript
// ============================================
// 操作类型定义
// ============================================

/**
 * 操作基类
 */
interface Operation {
  // 操作ID
  id: string;
  
  // 操作类型
  type: OperationType;
  
  // 操作来源
  origin: OperationOrigin;
  
  // 操作目标
  target: OperationTarget;
  
  // 操作数据
  data: OperationData;
  
  // 操作元数据
  metadata: OperationMetadata;
}

/**
 * 操作来源
 */
interface OperationOrigin {
  clientId: string;
  userId: string;
  userName: string;
  timestamp: number;
  sequenceNumber: number;
}

/**
 * 操作目标
 */
interface OperationTarget {
  // 目标类型
  type: 'element' | 'layer' | 'document' | 'property' | 'viewport';
  
  // 目标ID
  id: string;
  
  // 父目标ID（用于嵌套）
  parentId?: string;
  
  // 属性路径（用于属性操作）
  propertyPath?: string[];
}

/**
 * 操作数据联合类型
 */
type OperationData =
  | CreateElementData
  | UpdateElementData
  | DeleteElementData
  | TransformElementData
  | PropertyData
  | LayerData
  | SelectionData
  | ViewportData;

/**
 * 创建元素操作数据
 */
interface CreateElementData {
  elementType: GeometryType;
  initialData: Partial<GeometryElement>;
  parentId?: string;
  index?: number;
}

/**
 * 更新元素操作数据
 */
interface UpdateElementData {
  path: string;
  value: any;
  oldValue: any;
  updateMode: 'set' | 'add' | 'remove' | 'merge' | 'increment';
}

/**
 * 删除元素操作数据
 */
interface DeleteElementData {
  elementData: GeometryElement;
  cascadeDelete: boolean;
}

/**
 * 变换元素操作数据
 */
interface TransformElementData {
  transformType: 'translate' | 'rotate' | 'scale' | 'matrix';
  parameters: TransformParameters;
  relative: boolean;
  center?: Point3D;
}

/**
 * 属性操作数据
 */
interface PropertyData {
  propertySet: string;
  propertyName: string;
  value: PropertyValue;
  oldValue?: PropertyValue;
}

/**
 * 图层操作数据
 */
interface LayerData {
  action: 'create' | 'update' | 'delete' | 'reorder' | 'lock' | 'unlock' | 'show' | 'hide';
  layerData?: any;
  newIndex?: number;
}

/**
 * 选择操作数据
 */
interface SelectionData {
  action: 'set' | 'add' | 'remove' | 'clear';
  elementIds: string[];
}

/**
 * 视口操作数据
 */
interface ViewportData {
  action: 'pan' | 'zoom' | 'rotate' | 'set';
  center?: Point3D;
  zoom?: number;
  rotation?: number;
}

// ============================================
// 操作工厂
// ============================================

/**
 * 操作工厂
 */
class OperationFactory {
  private sequenceCounter: Map<string, number> = new Map();
  
  /**
   * 创建操作
   */
  createOperation(
    type: OperationType,
    clientId: string,
    userId: string,
    userName: string,
    target: OperationTarget,
    data: OperationData
  ): Operation {
    const sequenceNumber = this.getNextSequenceNumber(clientId);
    
    return {
      id: this.generateOperationId(),
      type,
      origin: {
        clientId,
        userId,
        userName,
        timestamp: Date.now(),
        sequenceNumber
      },
      target,
      data,
      metadata: {
        operationId: this.generateOperationId(),
        type,
        origin: {
          clientId,
          userId,
          timestamp: Date.now()
        },
        beforeVector: {},
        afterVector: {},
        affectedElements: [target.id],
        dependencies: [],
        priority: this.getPriority(type)
      }
    };
  }
  
  /**
   * 创建元素操作
   */
  createCreateElementOperation(
    clientId: string,
    userId: string,
    userName: string,
    elementType: GeometryType,
    initialData: Partial<GeometryElement>,
    parentId?: string
  ): Operation {
    return this.createOperation(
      OperationType.ELEMENT_CREATE,
      clientId,
      userId,
      userName,
      {
        type: 'element',
        id: initialData.id || this.generateElementId(),
        parentId
      },
      {
        elementType,
        initialData,
        parentId
      } as CreateElementData
    );
  }
  
  /**
   * 更新元素操作
   */
  createUpdateElementOperation(
    clientId: string,
    userId: string,
    userName: string,
    elementId: string,
    path: string,
    value: any,
    oldValue: any
  ): Operation {
    return this.createOperation(
      OperationType.ELEMENT_UPDATE,
      clientId,
      userId,
      userName,
      {
        type: 'element',
        id: elementId
      },
      {
        path,
        value,
        oldValue,
        updateMode: 'set'
      } as UpdateElementData
    );
  }
  
  /**
   * 删除元素操作
   */
  createDeleteElementOperation(
    clientId: string,
    userId: string,
    userName: string,
    element: GeometryElement,
    cascadeDelete: boolean = false
  ): Operation {
    return this.createOperation(
      OperationType.ELEMENT_DELETE,
      clientId,
      userId,
      userName,
      {
        type: 'element',
        id: element.id
      },
      {
        elementData: element,
        cascadeDelete
      } as DeleteElementData
    );
  }
  
  /**
   * 变换元素操作
   */
  createTransformOperation(
    clientId: string,
    userId: string,
    userName: string,
    elementId: string,
    transformType: 'translate' | 'rotate' | 'scale' | 'matrix',
    parameters: TransformParameters,
    relative: boolean = true
  ): Operation {
    return this.createOperation(
      OperationType.ELEMENT_TRANSFORM,
      clientId,
      userId,
      userName,
      {
        type: 'element',
        id: elementId
      },
      {
        transformType,
        parameters,
        relative
      } as TransformElementData
    );
  }
  
  /**
   * 获取下一个序列号
   */
  private getNextSequenceNumber(clientId: string): number {
    const current = this.sequenceCounter.get(clientId) || 0;
    const next = current + 1;
    this.sequenceCounter.set(clientId, next);
    return next;
  }
  
  /**
   * 生成操作ID
   */
  private generateOperationId(): string {
    return `op_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 生成元素ID
   */
  private generateElementId(): string {
    return `elem_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 获取操作优先级
   */
  private getPriority(type: OperationType): OperationPriority {
    switch (type) {
      case OperationType.ELEMENT_DELETE:
        return OperationPriority.CRITICAL;
      case OperationType.ELEMENT_CREATE:
      case OperationType.ELEMENT_UPDATE:
        return OperationPriority.HIGH;
      case OperationType.ELEMENT_TRANSFORM:
        return OperationPriority.NORMAL;
      default:
        return OperationPriority.LOW;
    }
  }
}
```

### 4.2 操作应用

```typescript
// ============================================
// 操作应用器
// ============================================

/**
 * 操作应用器
 */
class OperationApplier {
  // CRDT文档
  private ydoc: Y.Doc;
  
  // 元素映射
  private elements: Y.Map<Y.Map<any>>;
  
  // 属性映射
  private properties: Y.Map<Y.Map<any>>;
  
  // 图层映射
  private layers: Y.Map<Y.Map<any>>;
  
  // 撤销管理器
  private undoManager: Y.UndoManager;
  
  // 事件发射器
  private eventEmitter: EventEmitter;
  
  constructor(ydoc: Y.Doc) {
    this.ydoc = ydoc;
    this.elements = ydoc.getMap('elements');
    this.properties = ydoc.getMap('properties');
    this.layers = ydoc.getMap('layers');
    this.undoManager = new Y.UndoManager([this.elements, this.properties]);
    this.eventEmitter = new EventEmitter();
  }
  
  /**
   * 应用操作
   */
  applyOperation(operation: Operation): OperationResult {
    const transaction = this.ydoc.transact(() => {
      switch (operation.type) {
        case OperationType.ELEMENT_CREATE:
          return this.applyCreateElement(operation);
        case OperationType.ELEMENT_UPDATE:
          return this.applyUpdateElement(operation);
        case OperationType.ELEMENT_DELETE:
          return this.applyDeleteElement(operation);
        case OperationType.ELEMENT_TRANSFORM:
          return this.applyTransformElement(operation);
        case OperationType.PROPERTY_SET:
          return this.applySetProperty(operation);
        case OperationType.LAYER_CREATE:
          return this.applyCreateLayer(operation);
        case OperationType.LAYER_UPDATE:
          return this.applyUpdateLayer(operation);
        default:
          throw new Error(`Unknown operation type: ${operation.type}`);
      }
    }, operation.origin.clientId);
    
    // 触发事件
    this.eventEmitter.emit('operationApplied', {
      operation,
      result: transaction
    });
    
    return {
      success: true,
      operation,
      affectedElements: operation.metadata.affectedElements
    };
  }
  
  /**
   * 应用创建元素操作
   */
  private applyCreateElement(operation: Operation): any {
    const data = operation.data as CreateElementData;
    const elementId = operation.target.id;
    
    // 检查元素是否已存在
    if (this.elements.has(elementId)) {
      throw new Error(`Element ${elementId} already exists`);
    }
    
    // 创建元素YMap
    const elementYMap = new Y.Map<any>();
    
    // 设置基本属性
    elementYMap.set('id', elementId);
    elementYMap.set('type', data.elementType);
    elementYMap.set('createdBy', operation.origin.userId);
    elementYMap.set('createdAt', operation.origin.timestamp);
    elementYMap.set('modifiedBy', operation.origin.userId);
    elementYMap.set('modifiedAt', operation.origin.timestamp);
    elementYMap.set('versionVector', {
      [operation.origin.clientId]: operation.origin.sequenceNumber
    });
    
    // 设置初始数据
    for (const [key, value] of Object.entries(data.initialData)) {
      if (value !== undefined) {
        elementYMap.set(key, this.toYjsValue(value));
      }
    }
    
    // 添加到元素映射
    this.elements.set(elementId, elementYMap);
    
    // 如果指定了父元素，添加到父元素
    if (data.parentId) {
      const parent = this.elements.get(data.parentId);
      if (parent) {
        const children = parent.get('children') || new Y.Array<string>();
        children.push([elementId]);
        parent.set('children', children);
      }
    }
    
    // 添加到图层
    if (data.initialData.layerId) {
      const layer = this.layers.get(data.initialData.layerId);
      if (layer) {
        const layerElements = layer.get('elements') || new Y.Map();
        layerElements.set(elementId, true);
        layer.set('elements', layerElements);
      }
    }
    
    return { elementId, elementYMap };
  }
  
  /**
   * 应用更新元素操作
   */
  private applyUpdateElement(operation: Operation): any {
    const data = operation.data as UpdateElementData;
    const elementId = operation.target.id;
    
    // 获取元素
    const element = this.elements.get(elementId);
    if (!element) {
      throw new Error(`Element ${elementId} not found`);
    }
    
    // 解析路径
    const pathParts = data.path.split('.');
    
    // 应用更新
    this.setValueAtPath(element, pathParts, this.toYjsValue(data.value));
    
    // 更新修改信息
    element.set('modifiedBy', operation.origin.userId);
    element.set('modifiedAt', operation.origin.timestamp);
    
    // 更新版本向量
    const versionVector = element.get('versionVector') || {};
    versionVector[operation.origin.clientId] = operation.origin.sequenceNumber;
    element.set('versionVector', versionVector);
    
    return { elementId, path: data.path, value: data.value };
  }
  
  /**
   * 应用删除元素操作
   */
  private applyDeleteElement(operation: Operation): any {
    const data = operation.data as DeleteElementData;
    const elementId = operation.target.id;
    
    // 获取元素
    const element = this.elements.get(elementId);
    if (!element) {
      // 元素可能已被其他操作删除
      return { elementId, alreadyDeleted: true };
    }
    
    // 级联删除子元素
    if (data.cascadeDelete) {
      const children = element.get('children') as Y.Array<string>;
      if (children) {
        for (let i = children.length - 1; i >= 0; i--) {
          const childId = children.get(i);
          this.elements.delete(childId);
        }
      }
    }
    
    // 从父元素中移除
    const parentId = element.get('parentId');
    if (parentId) {
      const parent = this.elements.get(parentId);
      if (parent) {
        const children = parent.get('children') as Y.Array<string>;
        if (children) {
          const index = children.toArray().indexOf(elementId);
          if (index >= 0) {
            children.delete(index, 1);
          }
        }
      }
    }
    
    // 从图层中移除
    const layerId = element.get('layerId');
    if (layerId) {
      const layer = this.layers.get(layerId);
      if (layer) {
        const layerElements = layer.get('elements') as Y.Map<boolean>;
        if (layerElements) {
          layerElements.delete(elementId);
        }
      }
    }
    
    // 删除元素
    this.elements.delete(elementId);
    
    // 删除属性
    this.properties.delete(elementId);
    
    return { elementId };
  }
  
  /**
   * 应用变换元素操作
   */
  private applyTransformElement(operation: Operation): any {
    const data = operation.data as TransformElementData;
    const elementId = operation.target.id;
    
    // 获取元素
    const element = this.elements.get(elementId);
    if (!element) {
      throw new Error(`Element ${elementId} not found`);
    }
    
    // 获取当前变换
    const currentTransform = element.get('transform') as TransformMatrix || this.getIdentityMatrix();
    
    // 计算新变换
    let newTransform: TransformMatrix;
    
    switch (data.transformType) {
      case 'translate':
        newTransform = this.applyTranslation(
          currentTransform,
          data.parameters.translate!,
          data.relative
        );
        break;
      case 'rotate':
        newTransform = this.applyRotation(
          currentTransform,
          data.parameters.rotate!,
          data.center,
          data.relative
        );
        break;
      case 'scale':
        newTransform = this.applyScale(
          currentTransform,
          data.parameters.scale!,
          data.center,
          data.relative
        );
        break;
      case 'matrix':
        newTransform = data.parameters.matrix!;
        break;
      default:
        throw new Error(`Unknown transform type: ${data.transformType}`);
    }
    
    // 应用新变换
    element.set('transform', newTransform);
    
    // 更新修改信息
    element.set('modifiedBy', operation.origin.userId);
    element.set('modifiedAt', operation.origin.timestamp);
    
    // 更新边界框
    this.updateBoundingBox(element);
    
    return { elementId, transform: newTransform };
  }
  
  /**
   * 应用设置属性操作
   */
  private applySetProperty(operation: Operation): any {
    const data = operation.data as PropertyData;
    const elementId = operation.target.id;
    
    // 获取或创建属性集
    let elementProperties = this.properties.get(elementId);
    if (!elementProperties) {
      elementProperties = new Y.Map();
      this.properties.set(elementId, elementProperties);
    }
    
    // 获取或创建属性集
    let propertySet = elementProperties.get(data.propertySet) as Y.Map<any>;
    if (!propertySet) {
      propertySet = new Y.Map();
      elementProperties.set(data.propertySet, propertySet);
    }
    
    // 设置属性值
    propertySet.set(data.propertyName, this.toYjsValue(data.value));
    
    // 更新修改信息
    propertySet.set('_modifiedBy', operation.origin.userId);
    propertySet.set('_modifiedAt', operation.origin.timestamp);
    
    return {
      elementId,
      propertySet: data.propertySet,
      propertyName: data.propertyName,
      value: data.value
    };
  }
  
  /**
   * 在路径上设置值
   */
  private setValueAtPath(
    target: Y.Map<any>,
    path: string[],
    value: any
  ): void {
    if (path.length === 1) {
      target.set(path[0], value);
      return;
    }
    
    const [first, ...rest] = path;
    let next = target.get(first);
    
    if (!next) {
      next = new Y.Map();
      target.set(first, next);
    }
    
    this.setValueAtPath(next, rest, value);
  }
  
  /**
   * 转换为Yjs值
   */
  private toYjsValue(value: any): any {
    if (value === null || value === undefined) {
      return value;
    }
    
    if (Array.isArray(value)) {
      const yarray = new Y.Array();
      yarray.push(value.map(v => this.toYjsValue(v)));
      return yarray;
    }
    
    if (typeof value === 'object') {
      const ymap = new Y.Map();
      for (const [key, val] of Object.entries(value)) {
        ymap.set(key, this.toYjsValue(val));
      }
      return ymap;
    }
    
    return value;
  }
  
  /**
   * 应用平移
   */
  private applyTranslation(
    matrix: TransformMatrix,
    translation: { x: number; y: number; z: number },
    relative: boolean
  ): TransformMatrix {
    const translationMatrix: TransformMatrix = {
      m11: 1, m12: 0, m13: 0, m14: 0,
      m21: 0, m22: 1, m23: 0, m24: 0,
      m31: 0, m32: 0, m33: 1, m34: 0,
      m41: translation.x, m42: translation.y, m43: translation.z, m44: 1
    };
    
    if (relative) {
      return this.multiplyMatrices(matrix, translationMatrix);
    } else {
      return translationMatrix;
    }
  }
  
  /**
   * 应用旋转
   */
  private applyRotation(
    matrix: TransformMatrix,
    rotation: { x: number; y: number; z: number },
    center?: Point3D,
    relative: boolean = true
  ): TransformMatrix {
    // 实现旋转矩阵计算
    // 简化为单位矩阵
    return matrix;
  }
  
  /**
   * 应用缩放
   */
  private applyScale(
    matrix: TransformMatrix,
    scale: { x: number; y: number; z: number },
    center?: Point3D,
    relative: boolean = true
  ): TransformMatrix {
    const scaleMatrix: TransformMatrix = {
      m11: scale.x, m12: 0, m13: 0, m14: 0,
      m21: 0, m22: scale.y, m23: 0, m24: 0,
      m31: 0, m32: 0, m33: scale.z, m34: 0,
      m41: 0, m42: 0, m43: 0, m44: 1
    };
    
    if (relative) {
      return this.multiplyMatrices(matrix, scaleMatrix);
    } else {
      return scaleMatrix;
    }
  }
  
  /**
   * 矩阵乘法
   */
  private multiplyMatrices(a: TransformMatrix, b: TransformMatrix): TransformMatrix {
    return {
      m11: a.m11 * b.m11 + a.m12 * b.m21 + a.m13 * b.m31 + a.m14 * b.m41,
      m12: a.m11 * b.m12 + a.m12 * b.m22 + a.m13 * b.m32 + a.m14 * b.m42,
      m13: a.m11 * b.m13 + a.m12 * b.m23 + a.m13 * b.m33 + a.m14 * b.m43,
      m14: a.m11 * b.m14 + a.m12 * b.m24 + a.m13 * b.m34 + a.m14 * b.m44,
      m21: a.m21 * b.m11 + a.m22 * b.m21 + a.m23 * b.m31 + a.m24 * b.m41,
      m22: a.m21 * b.m12 + a.m22 * b.m22 + a.m23 * b.m32 + a.m24 * b.m42,
      m23: a.m21 * b.m13 + a.m22 * b.m23 + a.m23 * b.m33 + a.m24 * b.m43,
      m24: a.m21 * b.m14 + a.m22 * b.m24 + a.m23 * b.m34 + a.m24 * b.m44,
      m31: a.m31 * b.m11 + a.m32 * b.m21 + a.m33 * b.m31 + a.m34 * b.m41,
      m32: a.m31 * b.m12 + a.m32 * b.m22 + a.m33 * b.m32 + a.m34 * b.m42,
      m33: a.m31 * b.m13 + a.m32 * b.m23 + a.m33 * b.m33 + a.m34 * b.m43,
      m34: a.m31 * b.m14 + a.m32 * b.m24 + a.m33 * b.m34 + a.m34 * b.m44,
      m41: a.m41 * b.m11 + a.m42 * b.m21 + a.m43 * b.m31 + a.m44 * b.m41,
      m42: a.m41 * b.m12 + a.m42 * b.m22 + a.m43 * b.m32 + a.m44 * b.m42,
      m43: a.m41 * b.m13 + a.m42 * b.m23 + a.m43 * b.m33 + a.m44 * b.m43,
      m44: a.m41 * b.m14 + a.m42 * b.m24 + a.m43 * b.m34 + a.m44 * b.m44
    };
  }
  
  /**
   * 获取单位矩阵
   */
  private getIdentityMatrix(): TransformMatrix {
    return {
      m11: 1, m12: 0, m13: 0, m14: 0,
      m21: 0, m22: 1, m23: 0, m24: 0,
      m31: 0, m32: 0, m33: 1, m34: 0,
      m41: 0, m42: 0, m43: 0, m44: 1
    };
  }
  
  /**
   * 更新边界框
   */
  private updateBoundingBox(element: Y.Map<any>): void {
    // 实现边界框更新逻辑
    // 根据元素类型和变换计算新的边界框
  }
  
  // ... 其他辅助方法
}

// ============================================
// 操作结果类型
// ============================================

interface OperationResult {
  success: boolean;
  operation: Operation;
  affectedElements: string[];
  error?: string;
}
```



### 4.3 操作转换

```typescript
// ============================================
// 操作转换器
// ============================================

/**
 * 操作转换器 - 实现OT算法
 */
class OperationTransformer {
  /**
   * 转换操作
   * 将操作op1相对于操作op2进行转换，使得op1可以在op2应用后仍然有效
   */
  transform(op1: Operation, op2: Operation): Operation {
    // 如果操作不冲突，直接返回原操作
    if (!this.hasConflict(op1, op2)) {
      return op1;
    }
    
    // 根据操作类型进行转换
    switch (op1.type) {
      case OperationType.ELEMENT_UPDATE:
        return this.transformUpdateOperation(op1, op2);
      case OperationType.ELEMENT_DELETE:
        return this.transformDeleteOperation(op1, op2);
      case OperationType.ELEMENT_TRANSFORM:
        return this.transformTransformOperation(op1, op2);
      case OperationType.PROPERTY_SET:
        return this.transformPropertyOperation(op1, op2);
      default:
        // 其他操作类型，默认返回原操作
        return op1;
    }
  }
  
  /**
   * 转换更新操作
   */
  private transformUpdateOperation(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as UpdateElementData;
    
    switch (op2.type) {
      case OperationType.ELEMENT_UPDATE:
        return this.transformUpdateVsUpdate(op1, op2);
      case OperationType.ELEMENT_DELETE:
        // 如果元素被删除，更新操作变为无效
        return this.createNoOp(op1);
      case OperationType.ELEMENT_TRANSFORM:
        // 变换可能影响更新路径
        return this.transformUpdateVsTransform(op1, op2);
      default:
        return op1;
    }
  }
  
  /**
   * 转换更新vs更新
   */
  private transformUpdateVsUpdate(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as UpdateElementData;
    const data2 = op2.data as UpdateElementData;
    
    // 如果更新的是同一路径，需要合并
    if (data1.path === data2.path) {
      // 使用LWW（Last Write Wins）策略
      if (op1.origin.timestamp > op2.origin.timestamp) {
        // op1更新，保留
        return op1;
      } else {
        // op2更新，op1需要调整
        return this.createNoOp(op1);
      }
    }
    
    // 如果更新的是不同路径，没有冲突
    return op1;
  }
  
  /**
   * 转换更新vs变换
   */
  private transformUpdateVsTransform(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as UpdateElementData;
    const data2 = op2.data as TransformElementData;
    
    // 如果更新的是变换相关的属性，需要调整
    if (data1.path.startsWith('transform') || data1.path.startsWith('position')) {
      // 创建转换后的操作
      return {
        ...op1,
        data: {
          ...data1,
          // 可能需要调整值以考虑变换的影响
          value: this.adjustValueForTransform(data1.value, data2)
        }
      };
    }
    
    return op1;
  }
  
  /**
   * 转换删除操作
   */
  private transformDeleteOperation(
    op1: Operation,
    op2: Operation
  ): Operation {
    switch (op2.type) {
      case OperationType.ELEMENT_DELETE:
        // 如果删除的是同一元素，去重
        if (op1.target.id === op2.target.id) {
          return this.createNoOp(op1);
        }
        return op1;
      case OperationType.ELEMENT_UPDATE:
      case OperationType.ELEMENT_TRANSFORM:
        // 删除操作优先
        return op1;
      default:
        return op1;
    }
  }
  
  /**
   * 转换变换操作
   */
  private transformTransformOperation(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as TransformElementData;
    
    switch (op2.type) {
      case OperationType.ELEMENT_TRANSFORM:
        return this.transformTransformVsTransform(op1, op2);
      case OperationType.ELEMENT_DELETE:
        // 元素被删除，变换无效
        return this.createNoOp(op1);
      default:
        return op1;
    }
  }
  
  /**
   * 转换变换vs变换
   */
  private transformTransformVsTransform(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as TransformElementData;
    const data2 = op2.data as TransformElementData;
    
    // 合并两个变换
    const mergedTransform = this.mergeTransforms(data1, data2);
    
    return {
      ...op1,
      data: mergedTransform
    };
  }
  
  /**
   * 转换属性操作
   */
  private transformPropertyOperation(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as PropertyData;
    
    switch (op2.type) {
      case OperationType.PROPERTY_SET:
        return this.transformPropertyVsProperty(op1, op2);
      case OperationType.ELEMENT_DELETE:
        // 元素被删除，属性操作无效
        return this.createNoOp(op1);
      default:
        return op1;
    }
  }
  
  /**
   * 转换属性vs属性
   */
  private transformPropertyVsProperty(
    op1: Operation,
    op2: Operation
  ): Operation {
    const data1 = op1.data as PropertyData;
    const data2 = op2.data as PropertyData;
    
    // 如果设置的是同一属性
    if (data1.propertySet === data2.propertySet &&
        data1.propertyName === data2.propertyName) {
      // 使用LWW策略
      if (op1.origin.timestamp > op2.origin.timestamp) {
        return op1;
      } else {
        return this.createNoOp(op1);
      }
    }
    
    return op1;
  }
  
  /**
   * 检查操作是否冲突
   */
  private hasConflict(op1: Operation, op2: Operation): boolean {
    // 不同目标的操作不冲突
    if (op1.target.id !== op2.target.id) {
      return false;
    }
    
    // 检查操作类型组合
    const conflictPairs: Array<[OperationType, OperationType]> = [
      [OperationType.ELEMENT_UPDATE, OperationType.ELEMENT_UPDATE],
      [OperationType.ELEMENT_UPDATE, OperationType.ELEMENT_DELETE],
      [OperationType.ELEMENT_DELETE, OperationType.ELEMENT_UPDATE],
      [OperationType.ELEMENT_TRANSFORM, OperationType.ELEMENT_TRANSFORM],
      [OperationType.PROPERTY_SET, OperationType.PROPERTY_SET]
    ];
    
    return conflictPairs.some(
      pair => (op1.type === pair[0] && op2.type === pair[1]) ||
              (op1.type === pair[1] && op2.type === pair[0])
    );
  }
  
  /**
   * 创建空操作
   */
  private createNoOp(originalOp: Operation): Operation {
    return {
      ...originalOp,
      type: OperationType.ELEMENT_UPDATE, // 使用一个无害的操作类型
      data: {
        path: 'noop',
        value: null,
        oldValue: null,
        updateMode: 'set'
      } as UpdateElementData,
      metadata: {
        ...originalOp.metadata,
        isNoOp: true
      }
    };
  }
  
  /**
   * 调整值以考虑变换
   */
  private adjustValueForTransform(
    value: any,
    transform: TransformElementData
  ): any {
    // 实现值调整逻辑
    // 例如：如果值是位置，需要应用逆变换
    return value;
  }
  
  /**
   * 合并变换
   */
  private mergeTransforms(
    t1: TransformElementData,
    t2: TransformElementData
  ): TransformElementData {
    // 实现变换合并逻辑
    // 例如：两个平移可以合并为一个平移
    if (t1.transformType === 'translate' && t2.transformType === 'translate') {
      return {
        ...t1,
        parameters: {
          translate: {
            x: (t1.parameters.translate?.x || 0) + (t2.parameters.translate?.x || 0),
            y: (t1.parameters.translate?.y || 0) + (t2.parameters.translate?.y || 0),
            z: (t1.parameters.translate?.z || 0) + (t2.parameters.translate?.z || 0)
          }
        }
      };
    }
    
    // 其他情况，保留第一个变换
    return t1;
  }
}

// ============================================
// 操作转换管理器
// ============================================

/**
 * 操作转换管理器
 */
class OperationTransformManager {
  private transformer: OperationTransformer;
  private pendingOperations: Map<string, Operation[]> = new Map();
  
  constructor() {
    this.transformer = new OperationTransformer();
  }
  
  /**
   * 转换操作列表
   */
  transformOperations(
    localOps: Operation[],
    remoteOps: Operation[]
  ): Operation[] {
    const transformedOps: Operation[] = [];
    
    for (const localOp of localOps) {
      let transformedOp = localOp;
      
      // 对每个远程操作进行转换
      for (const remoteOp of remoteOps) {
        transformedOp = this.transformer.transform(transformedOp, remoteOp);
      }
      
      // 如果不是空操作，添加到结果
      if (!transformedOp.metadata.isNoOp) {
        transformedOps.push(transformedOp);
      }
    }
    
    return transformedOps;
  }
  
  /**
   * 转换单个操作
   */
  transformOperation(
    operation: Operation,
    againstOperations: Operation[]
  ): Operation {
    let transformedOp = operation;
    
    for (const againstOp of againstOperations) {
      transformedOp = this.transformer.transform(transformedOp, againstOp);
    }
    
    return transformedOp;
  }
  
  /**
   * 添加待处理操作
   */
  addPendingOperation(clientId: string, operation: Operation): void {
    if (!this.pendingOperations.has(clientId)) {
      this.pendingOperations.set(clientId, []);
    }
    this.pendingOperations.get(clientId)!.push(operation);
  }
  
  /**
   * 获取并清除待处理操作
   */
  getAndClearPendingOperations(clientId: string): Operation[] {
    const ops = this.pendingOperations.get(clientId) || [];
    this.pendingOperations.delete(clientId);
    return ops;
  }
}
```

### 4.4 操作广播

```typescript
// ============================================
// 操作广播器
// ============================================

/**
 * 操作广播器
 */
class OperationBroadcaster {
  // 连接管理器
  private connectionManager: ConnectionManager;
  
  // 广播配置
  private config: BroadcastConfig;
  
  // 批处理队列
  private batchQueue: Map<string, Operation[]> = new Map();
  
  // 批处理定时器
  private batchTimers: Map<string, NodeJS.Timeout> = new Map();
  
  // 统计信息
  private stats: BroadcastStats;
  
  constructor(connectionManager: ConnectionManager, config: BroadcastConfig) {
    this.connectionManager = connectionManager;
    this.config = config;
    this.stats = {
      totalBroadcasts: 0,
      totalOperations: 0,
      batchedOperations: 0,
      immediateBroadcasts: 0
    };
  }
  
  /**
   * 广播操作
   */
  async broadcast(
    sourceClientId: string,
    documentId: string,
    operation: Operation
  ): Promise<void> {
    // 检查是否需要立即广播
    if (this.shouldBroadcastImmediately(operation)) {
      await this.broadcastImmediately(sourceClientId, documentId, [operation]);
      this.stats.immediateBroadcasts++;
    } else {
      // 添加到批处理队列
      this.addToBatch(sourceClientId, documentId, operation);
      this.stats.batchedOperations++;
    }
    
    this.stats.totalOperations++;
  }
  
  /**
   * 批量广播操作
   */
  async broadcastBatch(
    sourceClientId: string,
    documentId: string,
    operations: Operation[]
  ): Promise<void> {
    // 按优先级分组
    const grouped = this.groupByPriority(operations);
    
    // 立即广播高优先级操作
    if (grouped.critical.length > 0) {
      await this.broadcastImmediately(sourceClientId, documentId, grouped.critical);
    }
    
    if (grouped.high.length > 0) {
      await this.broadcastImmediately(sourceClientId, documentId, grouped.high);
    }
    
    // 批量处理普通和低优先级操作
    if (grouped.normal.length > 0) {
      this.addToBatch(sourceClientId, documentId, ...grouped.normal);
    }
    
    if (grouped.low.length > 0) {
      this.addToBatch(sourceClientId, documentId, ...grouped.low);
    }
  }
  
  /**
   * 立即广播
   */
  private async broadcastImmediately(
    sourceClientId: string,
    documentId: string,
    operations: Operation[]
  ): Promise<void> {
    // 创建更新消息
    const updateMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.UPDATE,
      sender: { clientId: sourceClientId, userId: '' },
      timestamp: Date.now(),
      versionVector: this.computeVersionVector(operations),
      payload: {
        updateId: this.generateUpdateId(),
        updateType: UpdateType.REMOTE,
        operations,
        beforeStateVector: new Uint8Array(),
        afterStateVector: new Uint8Array(),
        dependencies: [],
        timestamp: Date.now()
      } as UpdatePayload
    };
    
    // 广播到文档（排除源客户端）
    this.connectionManager.broadcastToDocument(
      documentId,
      updateMessage,
      sourceClientId
    );
    
    this.stats.totalBroadcasts++;
  }
  
  /**
   * 添加到批处理队列
   */
  private addToBatch(
    sourceClientId: string,
    documentId: string,
    ...operations: Operation[]
  ): void {
    const batchKey = `${documentId}:${sourceClientId}`;
    
    if (!this.batchQueue.has(batchKey)) {
      this.batchQueue.set(batchKey, []);
    }
    
    this.batchQueue.get(batchKey)!.push(...operations);
    
    // 设置批处理定时器
    this.scheduleBatchFlush(batchKey, documentId, sourceClientId);
  }
  
  /**
   * 调度批处理刷新
   */
  private scheduleBatchFlush(
    batchKey: string,
    documentId: string,
    sourceClientId: string
  ): void {
    // 清除现有定时器
    if (this.batchTimers.has(batchKey)) {
      clearTimeout(this.batchTimers.get(batchKey)!);
    }
    
    // 设置新定时器
    const timer = setTimeout(() => {
      this.flushBatch(batchKey, documentId, sourceClientId);
    }, this.config.batchInterval);
    
    this.batchTimers.set(batchKey, timer);
  }
  
  /**
   * 刷新批处理队列
   */
  private flushBatch(
    batchKey: string,
    documentId: string,
    sourceClientId: string
  ): void {
    const operations = this.batchQueue.get(batchKey);
    if (!operations || operations.length === 0) {
      return;
    }
    
    // 清空队列
    this.batchQueue.delete(batchKey);
    this.batchTimers.delete(batchKey);
    
    // 广播
    this.broadcastImmediately(sourceClientId, documentId, operations);
  }
  
  /**
   * 检查是否应该立即广播
   */
  private shouldBroadcastImmediately(operation: Operation): boolean {
    // 关键和高优先级操作立即广播
    return operation.metadata.priority <= OperationPriority.HIGH;
  }
  
  /**
   * 按优先级分组
   */
  private groupByPriority(operations: Operation[]): GroupedOperations {
    const grouped: GroupedOperations = {
      critical: [],
      high: [],
      normal: [],
      low: []
    };
    
    for (const op of operations) {
      switch (op.metadata.priority) {
        case OperationPriority.CRITICAL:
          grouped.critical.push(op);
          break;
        case OperationPriority.HIGH:
          grouped.high.push(op);
          break;
        case OperationPriority.NORMAL:
          grouped.normal.push(op);
          break;
        case OperationPriority.LOW:
        case OperationPriority.BACKGROUND:
          grouped.low.push(op);
          break;
      }
    }
    
    return grouped;
  }
  
  /**
   * 计算版本向量
   */
  private computeVersionVector(operations: Operation[]): VersionVector {
    const vector: VersionVector = {};
    
    for (const op of operations) {
      const clientId = op.origin.clientId;
      const seqNum = op.origin.sequenceNumber;
      
      if (!vector[clientId] || vector[clientId] < seqNum) {
        vector[clientId] = seqNum;
      }
    }
    
    return vector;
  }
  
  /**
   * 生成消息ID
   */
  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 生成更新ID
   */
  private generateUpdateId(): string {
    return `upd_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 获取统计信息
   */
  getStats(): BroadcastStats {
    return { ...this.stats };
  }
  
  /**
   * 强制刷新所有批处理队列
   */
  flushAllBatches(): void {
    for (const [batchKey, timer] of this.batchTimers) {
      clearTimeout(timer);
      
      const [documentId, sourceClientId] = batchKey.split(':');
      this.flushBatch(batchKey, documentId, sourceClientId);
    }
  }
}

// ============================================
// 广播类型定义
// ============================================

interface BroadcastConfig {
  batchInterval: number;      // 批处理间隔（毫秒）
  maxBatchSize: number;       // 最大批处理大小
  compressionThreshold: number; // 压缩阈值（字节）
  enableCompression: boolean; // 是否启用压缩
}

interface BroadcastStats {
  totalBroadcasts: number;
  totalOperations: number;
  batchedOperations: number;
  immediateBroadcasts: number;
}

interface GroupedOperations {
  critical: Operation[];
  high: Operation[];
  normal: Operation[];
  low: Operation[];
}
```

---

## 5. 冲突解决详细设计

### 5.1 冲突检测算法

```typescript
// ============================================
// 冲突检测器
// ============================================

/**
 * 冲突检测器
 */
class ConflictDetector {
  // 冲突规则
  private conflictRules: ConflictRule[] = [];
  
  constructor() {
    this.setupDefaultRules();
  }
  
  /**
   * 检测冲突
   */
  detect(
    operation1: Operation,
    operation2: Operation
  ): ConflictMetadata | null {
    // 检查操作是否针对同一目标
    if (operation1.target.id !== operation2.target.id) {
      return null;
    }
    
    // 应用冲突规则
    for (const rule of this.conflictRules) {
      const conflict = rule.check(operation1, operation2);
      if (conflict) {
        return conflict;
      }
    }
    
    return null;
  }
  
  /**
   * 批量检测冲突
   */
  detectBatch(operations: Operation[]): ConflictMetadata[] {
    const conflicts: ConflictMetadata[] = [];
    
    for (let i = 0; i < operations.length; i++) {
      for (let j = i + 1; j < operations.length; j++) {
        const conflict = this.detect(operations[i], operations[j]);
        if (conflict) {
          conflicts.push(conflict);
        }
      }
    }
    
    return conflicts;
  }
  
  /**
   * 添加冲突规则
   */
  addRule(rule: ConflictRule): void {
    this.conflictRules.push(rule);
  }
  
  /**
   * 设置默认规则
   */
  private setupDefaultRules(): void {
    // 并发编辑规则
    this.addRule({
      name: 'concurrent_edit',
      check: (op1, op2) => {
        // 检查是否是并发编辑
        if (this.isConcurrent(op1, op2) &&
            op1.type === OperationType.ELEMENT_UPDATE &&
            op2.type === OperationType.ELEMENT_UPDATE) {
          const data1 = op1.data as UpdateElementData;
          const data2 = op2.data as UpdateElementData;
          
          // 如果编辑的是同一路径
          if (data1.path === data2.path) {
            return this.createConflict(
              ConflictType.CONCURRENT_EDIT,
              op1.target.id,
              [op1, op2],
              {
                propertyPath: data1.path,
                values: [
                  { clientId: op1.origin.clientId, value: data1.value },
                  { clientId: op2.origin.clientId, value: data2.value }
                ]
              }
            );
          }
        }
        return null;
      }
    });
    
    // 删除后编辑规则
    this.addRule({
      name: 'edit_after_delete',
      check: (op1, op2) => {
        if (op1.type === OperationType.ELEMENT_DELETE &&
            (op2.type === OperationType.ELEMENT_UPDATE ||
             op2.type === OperationType.ELEMENT_TRANSFORM)) {
          return this.createConflict(
            ConflictType.EDIT_AFTER_DELETE,
            op1.target.id,
            [op1, op2],
            {
              deletedBy: op1.origin.clientId,
              editedBy: op2.origin.clientId
            }
          );
        }
        return null;
      }
    });
    
    // 属性冲突规则
    this.addRule({
      name: 'property_conflict',
      check: (op1, op2) => {
        if (op1.type === OperationType.PROPERTY_SET &&
            op2.type === OperationType.PROPERTY_SET) {
          const data1 = op1.data as PropertyData;
          const data2 = op2.data as PropertyData;
          
          if (data1.propertySet === data2.propertySet &&
              data1.propertyName === data2.propertyName) {
            return this.createConflict(
              ConflictType.PROPERTY_CONFLICT,
              op1.target.id,
              [op1, op2],
              {
                propertySet: data1.propertySet,
                propertyName: data1.propertyName,
                values: [
                  { clientId: op1.origin.clientId, value: data1.value },
                  { clientId: op2.origin.clientId, value: data2.value }
                ]
              }
            );
          }
        }
        return null;
      }
    });
    
    // 结构冲突规则
    this.addRule({
      name: 'structure_conflict',
      check: (op1, op2) => {
        // 检查父子关系冲突
        if (this.hasStructuralConflict(op1, op2)) {
          return this.createConflict(
            ConflictType.STRUCTURE_CONFLICT,
            op1.target.id,
            [op1, op2],
            {
              description: 'Structural relationship conflict'
            }
          );
        }
        return null;
      }
    });
    
    // 依赖冲突规则
    this.addRule({
      name: 'dependency_conflict',
      check: (op1, op2) => {
        // 检查依赖关系冲突
        if (this.hasDependencyConflict(op1, op2)) {
          return this.createConflict(
            ConflictType.DEPENDENCY_CONFLICT,
            op1.target.id,
            [op1, op2],
            {
              description: 'Dependency relationship conflict'
            }
          );
        }
        return null;
      }
    });
  }
  
  /**
   * 检查是否并发
   */
  private isConcurrent(op1: Operation, op2: Operation): boolean {
    // 检查版本向量
    const vv1 = op1.metadata.beforeVector;
    const vv2 = op2.metadata.beforeVector;
    
    // 如果op1不知道op2，且op2不知道op1，则是并发
    const op1KnowsOp2 = vv1[op2.origin.clientId] !== undefined &&
                        vv1[op2.origin.clientId] >= op2.origin.sequenceNumber;
    const op2KnowsOp1 = vv2[op1.origin.clientId] !== undefined &&
                        vv2[op1.origin.clientId] >= op1.origin.sequenceNumber;
    
    return !op1KnowsOp2 && !op2KnowsOp1;
  }
  
  /**
   * 检查结构冲突
   */
  private hasStructuralConflict(op1: Operation, op2: Operation): boolean {
    // 检查是否涉及父子关系的冲突
    if (op1.type === OperationType.ELEMENT_CREATE &&
        op2.type === OperationType.ELEMENT_DELETE) {
      const data1 = op1.data as CreateElementData;
      if (data1.parentId === op2.target.id) {
        return true;
      }
    }
    return false;
  }
  
  /**
   * 检查依赖冲突
   */
  private hasDependencyConflict(op1: Operation, op2: Operation): boolean {
    // 检查元素之间的依赖关系
    // 例如：墙体和开洞的依赖
    return false;
  }
  
  /**
   * 创建冲突
   */
  private createConflict(
    type: ConflictType,
    elementId: string,
    operations: Operation[],
    details: any
  ): ConflictMetadata {
    return {
      conflictId: this.generateConflictId(),
      type,
      elementId,
      operations: operations.map(op => op.id),
      detectedAt: Date.now(),
      status: ConflictStatus.DETECTED,
      details
    };
  }
  
  /**
   * 生成冲突ID
   */
  private generateConflictId(): string {
    return `conflict_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// ============================================
// 冲突规则类型
// ============================================

interface ConflictRule {
  name: string;
  check(op1: Operation, op2: Operation): ConflictMetadata | null;
}
```

### 5.2 自动合并策略

```typescript
// ============================================
// 自动合并器
// ============================================

/**
 * 自动合并器
 */
class AutoMerger {
  // 合并策略
  private mergeStrategies: Map<ConflictType, MergeStrategy> = new Map();
  
  constructor() {
    this.setupDefaultStrategies();
  }
  
  /**
   * 尝试自动合并
   */
  tryMerge(conflict: ConflictMetadata): ConflictResolution | null {
    const strategy = this.mergeStrategies.get(conflict.type);
    
    if (!strategy) {
      return null;
    }
    
    // 检查是否可以自动合并
    if (!strategy.canAutoMerge(conflict)) {
      return null;
    }
    
    // 执行合并
    return strategy.merge(conflict);
  }
  
  /**
   * 注册合并策略
   */
  registerStrategy(type: ConflictType, strategy: MergeStrategy): void {
    this.mergeStrategies.set(type, strategy);
  }
  
  /**
   * 设置默认策略
   */
  private setupDefaultStrategies(): void {
    // 并发编辑策略 - LWW
    this.registerStrategy(ConflictType.CONCURRENT_EDIT, {
      canAutoMerge: (conflict) => {
        // 数值类型可以自动合并
        const details = conflict.details;
        if (details.values) {
          const firstValue = details.values[0].value;
          return typeof firstValue === 'number' ||
                 typeof firstValue === 'boolean';
        }
        return false;
      },
      merge: (conflict) => {
        const details = conflict.details;
        const values = details.values;
        
        // 使用最后写入获胜
        const lastValue = values.reduce((latest, current) =>
          current.timestamp > latest.timestamp ? current : latest
        );
        
        return {
          method: 'last_write_wins',
          resolvedBy: 'system',
          resolvedAt: Date.now(),
          result: {
            propertyPath: details.propertyPath,
            value: lastValue.value
          },
          discardedOperations: values
            .filter(v => v.clientId !== lastValue.clientId)
            .map(v => v.operationId)
        };
      }
    });
    
    // 属性冲突策略 - 数值合并
    this.registerStrategy(ConflictType.PROPERTY_CONFLICT, {
      canAutoMerge: (conflict) => {
        const details = conflict.details;
        const values = details.values;
        
        // 数值可以合并
        return values.every(v => typeof v.value === 'number');
      },
      merge: (conflict) => {
        const details = conflict.details;
        const values = details.values;
        
        // 数值取最大
        const maxValue = Math.max(...values.map(v => v.value));
        
        return {
          method: 'auto_merge',
          resolvedBy: 'system',
          resolvedAt: Date.now(),
          result: {
            propertySet: details.propertySet,
            propertyName: details.propertyName,
            value: maxValue
          },
          discardedOperations: []
        };
      }
    });
    
    // 变换合并策略
    this.registerStrategy(ConflictType.CONCURRENT_EDIT, {
      canAutoMerge: (conflict) => {
        // 变换操作可以合并
        return true;
      },
      merge: (conflict) => {
        // 合并变换
        return {
          method: 'auto_merge',
          resolvedBy: 'system',
          resolvedAt: Date.now(),
          result: {
            merged: true
          },
          discardedOperations: []
        };
      }
    });
  }
  
  /**
   * 数值合并策略
   */
  private mergeNumericValues(values: any[]): any {
    // 取平均值
    const sum = values.reduce((acc, v) => acc + v.value, 0);
    return sum / values.length;
  }
  
  /**
   * 数组合并策略
   */
  private mergeArrays(arrays: any[][]): any[] {
    // 合并并去重
    const merged = arrays.flat();
    return [...new Set(merged)];
  }
  
  /**
   * 对象合并策略
   */
  private mergeObjects(objects: object[]): object {
    // 深度合并
    return objects.reduce((merged, obj) => ({ ...merged, ...obj }), {});
  }
}

// ============================================
// 合并策略类型
// ============================================

interface MergeStrategy {
  canAutoMerge(conflict: ConflictMetadata): boolean;
  merge(conflict: ConflictMetadata): ConflictResolution;
}
```

### 5.3 人工介入流程

```typescript
// ============================================
// 人工冲突解决管理器
// ============================================

/**
 * 人工冲突解决管理器
 */
class ManualConflictResolver {
  // 待解决冲突队列
  private pendingConflicts: Map<string, ConflictMetadata> = new Map();
  
  // 冲突解决回调
  private resolutionCallbacks: Map<string, (resolution: ConflictResolution) => void> = new Map();
  
  // 通知管理器
  private notificationManager: NotificationManager;
  
  // 超时配置
  private timeoutConfig: ConflictTimeoutConfig;
  
  constructor(
    notificationManager: NotificationManager,
    timeoutConfig: ConflictTimeoutConfig
  ) {
    this.notificationManager = notificationManager;
    this.timeoutConfig = timeoutConfig;
  }
  
  /**
   * 提交冲突等待人工解决
   */
  async submitForManualResolution(
    conflict: ConflictMetadata,
    options: ResolutionOptions
  ): Promise<ConflictResolution> {
    // 添加到待解决队列
    this.pendingConflicts.set(conflict.conflictId, conflict);
    
    // 更新冲突状态
    conflict.status = ConflictStatus.PENDING;
    
    // 发送通知
    await this.notifyConflict(conflict, options);
    
    // 返回Promise，等待解决
    return new Promise((resolve, reject) => {
      // 设置超时
      const timeoutId = setTimeout(() => {
        this.pendingConflicts.delete(conflict.conflictId);
        this.resolutionCallbacks.delete(conflict.conflictId);
        reject(new Error('Conflict resolution timeout'));
      }, this.timeoutConfig.resolutionTimeout);
      
      // 注册回调
      this.resolutionCallbacks.set(conflict.conflictId, (resolution) => {
        clearTimeout(timeoutId);
        resolve(resolution);
      });
    });
  }
  
  /**
   * 通知冲突
   */
  private async notifyConflict(
    conflict: ConflictMetadata,
    options: ResolutionOptions
  ): Promise<void> {
    // 构建通知内容
    const notification: ConflictNotification = {
      conflictId: conflict.conflictId,
      type: conflict.type,
      elementId: conflict.elementId,
      description: this.buildConflictDescription(conflict),
      options: this.buildResolutionOptions(conflict),
      preview: await this.buildConflictPreview(conflict),
      urgency: options.urgency || 'normal',
      deadline: Date.now() + this.timeoutConfig.resolutionTimeout
    };
    
    // 发送通知给相关用户
    for (const userId of options.notifyUsers) {
      await this.notificationManager.send(userId, notification);
    }
  }
  
  /**
   * 构建冲突描述
   */
  private buildConflictDescription(conflict: ConflictMetadata): string {
    switch (conflict.type) {
      case ConflictType.CONCURRENT_EDIT:
        return `并发编辑冲突：多个用户同时编辑了元素 ${conflict.elementId}`;
      case ConflictType.EDIT_AFTER_DELETE:
        return `删除后编辑冲突：元素 ${conflict.elementId} 已被删除，但仍有编辑操作`;
      case ConflictType.PROPERTY_CONFLICT:
        return `属性冲突：元素 ${conflict.elementId} 的属性存在冲突`;
      case ConflictType.STRUCTURE_CONFLICT:
        return `结构冲突：元素 ${conflict.elementId} 的结构关系存在冲突`;
      case ConflictType.DEPENDENCY_CONFLICT:
        return `依赖冲突：元素 ${conflict.elementId} 的依赖关系存在冲突`;
      default:
        return `未知冲突：元素 ${conflict.elementId}`;
    }
  }
  
  /**
   * 构建解决选项
   */
  private buildResolutionOptions(conflict: ConflictMetadata): ResolutionOption[] {
    const options: ResolutionOption[] = [];
    
    // 保留我的修改
    options.push({
      id: 'keep_mine',
      label: '保留我的修改',
      description: '使用您的修改覆盖其他修改',
      action: 'keep_local'
    });
    
    // 保留对方的修改
    options.push({
      id: 'keep_theirs',
      label: '保留对方的修改',
      description: '接受其他用户的修改',
      action: 'keep_remote'
    });
    
    // 合并修改
    options.push({
      id: 'merge',
      label: '合并修改',
      description: '尝试自动合并两个修改',
      action: 'merge'
    });
    
    // 手动编辑
    options.push({
      id: 'manual',
      label: '手动编辑',
      description: '手动编辑解决冲突',
      action: 'manual_edit'
    });
    
    // 撤销操作
    options.push({
      id: 'revert',
      label: '撤销我的操作',
      description: '撤销您的修改，保留原始状态',
      action: 'revert'
    });
    
    return options;
  }
  
  /**
   * 构建冲突预览
   */
  private async buildConflictPreview(
    conflict: ConflictMetadata
  ): Promise<ConflictPreview> {
    // 获取冲突元素的当前状态
    const currentState = await this.getElementState(conflict.elementId);
    
    // 获取冲突的操作详情
    const operations = await this.getOperations(conflict.operations);
    
    return {
      elementId: conflict.elementId,
      currentState,
      operations: operations.map(op => ({
        clientId: op.origin.clientId,
        userName: op.origin.userName,
        timestamp: op.origin.timestamp,
        changes: this.extractChanges(op)
      }))
    };
  }
  
  /**
   * 解决冲突
   */
  resolveConflict(
    conflictId: string,
    resolution: ConflictResolution
  ): void {
    const conflict = this.pendingConflicts.get(conflictId);
    if (!conflict) {
      throw new Error(`Conflict ${conflictId} not found`);
    }
    
    // 更新冲突状态
    conflict.status = ConflictStatus.RESOLVED;
    conflict.resolution = resolution;
    
    // 从待解决队列移除
    this.pendingConflicts.delete(conflictId);
    
    // 触发回调
    const callback = this.resolutionCallbacks.get(conflictId);
    if (callback) {
      callback(resolution);
      this.resolutionCallbacks.delete(conflictId);
    }
    
    // 广播解决结果
    this.broadcastResolution(conflict, resolution);
  }
  
  /**
   * 获取元素状态
   */
  private async getElementState(elementId: string): Promise<any> {
    // 实现获取元素状态逻辑
    return {};
  }
  
  /**
   * 获取操作详情
   */
  private async getOperations(operationIds: string[]): Promise<Operation[]> {
    // 实现获取操作逻辑
    return [];
  }
  
  /**
   * 提取变更
   */
  private extractChanges(operation: Operation): any {
    // 实现提取变更逻辑
    return operation.data;
  }
  
  /**
   * 广播解决结果
   */
  private broadcastResolution(
    conflict: ConflictMetadata,
    resolution: ConflictResolution
  ): void {
    // 实现广播逻辑
  }
  
  /**
   * 获取待解决冲突列表
   */
  getPendingConflicts(): ConflictMetadata[] {
    return Array.from(this.pendingConflicts.values());
  }
  
  /**
   * 获取冲突详情
   */
  getConflictDetails(conflictId: string): ConflictMetadata | null {
    return this.pendingConflicts.get(conflictId) || null;
  }
}

// ============================================
// 人工解决类型定义
// ============================================

interface ResolutionOptions {
  notifyUsers: string[];
  urgency?: 'low' | 'normal' | 'high' | 'critical';
  autoResolveTimeout?: number;
}

interface ConflictNotification {
  conflictId: string;
  type: ConflictType;
  elementId: string;
  description: string;
  options: ResolutionOption[];
  preview: ConflictPreview;
  urgency: string;
  deadline: number;
}

interface ResolutionOption {
  id: string;
  label: string;
  description: string;
  action: string;
}

interface ConflictPreview {
  elementId: string;
  currentState: any;
  operations: OperationPreview[];
}

interface OperationPreview {
  clientId: string;
  userName: string;
  timestamp: number;
  changes: any;
}

interface ConflictTimeoutConfig {
  resolutionTimeout: number;
  notificationTimeout: number;
}

interface NotificationManager {
  send(userId: string, notification: ConflictNotification): Promise<void>;
}
```

### 5.4 冲突通知机制

```typescript
// ============================================
// 冲突通知管理器
// ============================================

/**
 * 冲突通知管理器
 */
class ConflictNotificationManager implements NotificationManager {
  // WebSocket连接管理器
  private connectionManager: ConnectionManager;
  
  // 通知配置
  private config: NotificationConfig;
  
  // 通知历史
  private notificationHistory: Map<string, NotificationRecord> = new Map();
  
  constructor(connectionManager: ConnectionManager, config: NotificationConfig) {
    this.connectionManager = connectionManager;
    this.config = config;
  }
  
  /**
   * 发送通知
   */
  async send(
    userId: string,
    notification: ConflictNotification
  ): Promise<void> {
    // 记录通知
    this.recordNotification(notification);
    
    // 获取用户连接
    const connections = this.connectionManager.getUserConnections(userId);
    
    // 创建通知消息
    const message: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.CONFLICT_DETECTED,
      sender: { clientId: 'system', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        conflictId: notification.conflictId,
        type: notification.type,
        elementId: notification.elementId,
        description: notification.description,
        options: notification.options,
        preview: notification.preview,
        urgency: notification.urgency,
        deadline: notification.deadline
      } as ConflictPayload
    };
    
    // 发送给所有连接
    for (const connection of connections) {
      this.connectionManager.sendMessage(connection, message);
    }
    
    // 如果用户不在线，可能需要其他通知方式
    if (connections.length === 0) {
      await this.sendOfflineNotification(userId, notification);
    }
  }
  
  /**
   * 发送离线通知
   */
  private async sendOfflineNotification(
    userId: string,
    notification: ConflictNotification
  ): Promise<void> {
    // 可以集成邮件、推送等通知方式
    console.log(`User ${userId} is offline, conflict ${notification.conflictId} queued`);
  }
  
  /**
   * 广播冲突解决
   */
  async broadcastResolution(
    documentId: string,
    conflict: ConflictMetadata,
    resolution: ConflictResolution
  ): Promise<void> {
    const message: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.CONFLICT_RESOLVED,
      sender: { clientId: 'system', userId: 'system' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        conflictId: conflict.conflictId,
        resolution,
        result: resolution.result
      } as ConflictResolvedPayload
    };
    
    // 广播给文档的所有用户
    this.connectionManager.broadcastToDocument(documentId, message);
  }
  
  /**
   * 记录通知
   */
  private recordNotification(notification: ConflictNotification): void {
    this.notificationHistory.set(notification.conflictId, {
      conflictId: notification.conflictId,
      sentAt: Date.now(),
      notifiedUsers: [],
      status: 'sent'
    });
  }
  
  /**
   * 获取通知历史
   */
  getNotificationHistory(conflictId?: string): NotificationRecord[] {
    if (conflictId) {
      const record = this.notificationHistory.get(conflictId);
      return record ? [record] : [];
    }
    return Array.from(this.notificationHistory.values());
  }
  
  /**
   * 生成消息ID
   */
  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// ============================================
// 通知类型定义
// ============================================

interface NotificationConfig {
  enablePush: boolean;
  enableEmail: boolean;
  enableInApp: boolean;
  batchNotifications: boolean;
  batchInterval: number;
}

interface NotificationRecord {
  conflictId: string;
  sentAt: number;
  notifiedUsers: string[];
  status: 'sent' | 'delivered' | 'read' | 'expired';
  readAt?: number;
}
```

---

## 6. 性能优化详细设计

### 6.1 批量操作优化

```typescript
// ============================================
// 批量操作管理器
// ============================================

/**
 * 批量操作管理器
 */
class BatchOperationManager {
  // 批处理队列
  private batchQueue: Operation[] = [];
  
  // 批处理定时器
  private batchTimer: NodeJS.Timeout | null = null;
  
  // 配置
  private config: BatchConfig;
  
  // 操作应用器
  private operationApplier: OperationApplier;
  
  // 统计信息
  private stats: BatchStats;
  
  constructor(config: BatchConfig, operationApplier: OperationApplier) {
    this.config = config;
    this.operationApplier = operationApplier;
    this.stats = {
      totalBatches: 0,
      totalOperations: 0,
      averageBatchSize: 0,
      maxBatchSize: 0
    };
  }
  
  /**
   * 添加操作到批处理队列
   */
  addOperation(operation: Operation): void {
    this.batchQueue.push(operation);
    
    // 检查是否达到批处理阈值
    if (this.batchQueue.length >= this.config.maxBatchSize) {
      this.flushBatch();
    } else {
      // 设置定时器
      this.scheduleBatchFlush();
    }
  }
  
  /**
   * 批量添加操作
   */
  addOperations(operations: Operation[]): void {
    this.batchQueue.push(...operations);
    
    if (this.batchQueue.length >= this.config.maxBatchSize) {
      this.flushBatch();
    } else {
      this.scheduleBatchFlush();
    }
  }
  
  /**
   * 调度批处理刷新
   */
  private scheduleBatchFlush(): void {
    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
    }
    
    this.batchTimer = setTimeout(() => {
      this.flushBatch();
    }, this.config.batchInterval);
  }
  
  /**
   * 刷新批处理队列
   */
  flushBatch(): OperationResult[] {
    if (this.batchQueue.length === 0) {
      return [];
    }
    
    // 清除定时器
    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }
    
    // 获取当前队列
    const operations = [...this.batchQueue];
    this.batchQueue = [];
    
    // 优化操作顺序
    const optimizedOperations = this.optimizeOperationOrder(operations);
    
    // 合并可合并的操作
    const mergedOperations = this.mergeOperations(optimizedOperations);
    
    // 批量应用
    const results: OperationResult[] = [];
    
    // 使用Yjs事务批量应用
    this.operationApplier['ydoc'].transact(() => {
      for (const operation of mergedOperations) {
        try {
          const result = this.operationApplier.applyOperation(operation);
          results.push(result);
        } catch (error) {
          results.push({
            success: false,
            operation,
            affectedElements: [],
            error: error.message
          });
        }
      }
    }, 'batch');
    
    // 更新统计
    this.stats.totalBatches++;
    this.stats.totalOperations += operations.length;
    this.stats.maxBatchSize = Math.max(this.stats.maxBatchSize, operations.length);
    this.stats.averageBatchSize = this.stats.totalOperations / this.stats.totalBatches;
    
    return results;
  }
  
  /**
   * 优化操作顺序
   */
  private optimizeOperationOrder(operations: Operation[]): Operation[] {
    // 按优先级排序
    const priorityOrder = [
      OperationPriority.CRITICAL,
      OperationPriority.HIGH,
      OperationPriority.NORMAL,
      OperationPriority.LOW,
      OperationPriority.BACKGROUND
    ];
    
    return operations.sort((a, b) => {
      const priorityDiff = priorityOrder.indexOf(a.metadata.priority) -
                          priorityOrder.indexOf(b.metadata.priority);
      if (priorityDiff !== 0) {
        return priorityDiff;
      }
      
      // 同优先级按时间排序
      return a.origin.timestamp - b.origin.timestamp;
    });
  }
  
  /**
   * 合并操作
   */
  private mergeOperations(operations: Operation[]): Operation[] {
    const merged: Operation[] = [];
    const mergeMap: Map<string, Operation[]> = new Map();
    
    // 按目标分组
    for (const op of operations) {
      const key = `${op.target.id}:${op.type}`;
      if (!mergeMap.has(key)) {
        mergeMap.set(key, []);
      }
      mergeMap.get(key)!.push(op);
    }
    
    // 尝试合并
    for (const [key, ops] of mergeMap) {
      if (ops.length === 1) {
        merged.push(ops[0]);
        continue;
      }
      
      // 尝试合并同类型的操作
      const mergedOp = this.tryMergeOperations(ops);
      if (mergedOp) {
        merged.push(mergedOp);
      } else {
        merged.push(...ops);
      }
    }
    
    return merged;
  }
  
  /**
   * 尝试合并操作
   */
  private tryMergeOperations(operations: Operation[]): Operation | null {
    if (operations.length === 0) return null;
    if (operations.length === 1) return operations[0];
    
    const firstOp = operations[0];
    
    // 只合并更新操作
    if (firstOp.type !== OperationType.ELEMENT_UPDATE) {
      return null;
    }
    
    // 检查是否都是同一属性的更新
    const path = (firstOp.data as UpdateElementData).path;
    const allSamePath = operations.every(op =>
      op.type === OperationType.ELEMENT_UPDATE &&
      (op.data as UpdateElementData).path === path
    );
    
    if (!allSamePath) {
      return null;
    }
    
    // 合并为最后一个值
    const lastOp = operations[operations.length - 1];
    return {
      ...lastOp,
      metadata: {
        ...lastOp.metadata,
        mergedFrom: operations.map(op => op.id)
      }
    };
  }
  
  /**
   * 获取统计信息
   */
  getStats(): BatchStats {
    return { ...this.stats };
  }
}

// ============================================
// 批处理类型定义
// ============================================

interface BatchConfig {
  maxBatchSize: number;       // 最大批处理大小
  batchInterval: number;      // 批处理间隔（毫秒）
  enableMerging: boolean;     // 是否启用操作合并
  maxMergeWindow: number;     // 最大合并窗口（毫秒）
}

interface BatchStats {
  totalBatches: number;
  totalOperations: number;
  averageBatchSize: number;
  maxBatchSize: number;
}
```

### 6.2 增量更新优化

```typescript
// ============================================
// 增量更新管理器
// ============================================

/**
 * 增量更新管理器
 */
class IncrementalUpdateManager {
  // Yjs文档
  private ydoc: Y.Doc;
  
  // 状态向量缓存
  private stateVectorCache: Map<string, Uint8Array> = new Map();
  
  // 更新缓存
  private updateCache: LRUCache<string, Uint8Array>;
  
  // 差异计算缓存
  private diffCache: Map<string, Uint8Array>;
  
  // 配置
  private config: IncrementalConfig;
  
  constructor(ydoc: Y.Doc, config: IncrementalConfig) {
    this.ydoc = ydoc;
    this.config = config;
    this.updateCache = new LRUCache(config.cacheSize);
    this.diffCache = new Map();
  }
  
  /**
   * 计算增量更新
   */
  computeIncrementalUpdate(
    clientStateVector: Uint8Array
  ): Uint8Array {
    const cacheKey = this.hashStateVector(clientStateVector);
    
    // 检查缓存
    if (this.diffCache.has(cacheKey)) {
      return this.diffCache.get(cacheKey)!;
    }
    
    // 计算差异
    const diff = Y.encodeStateAsUpdate(this.ydoc, clientStateVector);
    
    // 如果差异较小，直接返回
    if (diff.length <= this.config.minCompressionSize) {
      this.diffCache.set(cacheKey, diff);
      return diff;
    }
    
    // 压缩差异
    const compressed = this.compressUpdate(diff);
    
    // 缓存结果
    if (this.diffCache.size < this.config.maxCacheEntries) {
      this.diffCache.set(cacheKey, compressed);
    }
    
    return compressed;
  }
  
  /**
   * 压缩更新
   */
  private compressUpdate(update: Uint8Array): Uint8Array {
    // 使用gzip压缩
    if (this.config.compressionEnabled && update.length > this.config.compressionThreshold) {
      return gzip(update);
    }
    return update;
  }
  
  /**
   * 解压更新
   */
  decompressUpdate(update: Uint8Array): Uint8Array {
    // 检查是否是压缩数据
    if (this.isCompressed(update)) {
      return gunzip(update);
    }
    return update;
  }
  
  /**
   * 检查是否压缩
   */
  private isCompressed(data: Uint8Array): boolean {
    // gzip魔数: 0x1f 0x8b
    return data.length > 2 && data[0] === 0x1f && data[1] === 0x8b;
  }
  
  /**
   * 获取状态向量
   */
  getStateVector(): Uint8Array {
    return Y.encodeStateVector(this.ydoc);
  }
  
  /**
   * 应用增量更新
   */
  applyIncrementalUpdate(update: Uint8Array): void {
    // 解压（如果需要）
    const decompressed = this.decompressUpdate(update);
    
    // 应用更新
    Y.applyUpdate(this.ydoc, decompressed);
  }
  
  /**
   * 分块大更新
   */
  chunkLargeUpdate(update: Uint8Array, chunkSize: number): Uint8Array[] {
    if (update.length <= chunkSize) {
      return [update];
    }
    
    const chunks: Uint8Array[] = [];
    for (let i = 0; i < update.length; i += chunkSize) {
      chunks.push(update.slice(i, i + chunkSize));
    }
    
    return chunks;
  }
  
  /**
   * 合并更新
   */
  mergeUpdates(updates: Uint8Array[]): Uint8Array {
    return Y.mergeUpdates(updates);
  }
  
  /**
   * 哈希状态向量
   */
  private hashStateVector(stateVector: Uint8Array): string {
    // 简单的哈希实现
    let hash = 0;
    for (let i = 0; i < stateVector.length; i++) {
      const char = stateVector[i];
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return hash.toString(16);
  }
  
  /**
   * 清理缓存
   */
  cleanupCache(): void {
    this.diffCache.clear();
    this.stateVectorCache.clear();
  }
}

// ============================================
// LRU缓存实现
// ============================================

class LRUCache<K, V> {
  private cache: Map<K, V>;
  private maxSize: number;
  
  constructor(maxSize: number) {
    this.cache = new Map();
    this.maxSize = maxSize;
  }
  
  get(key: K): V | undefined {
    const value = this.cache.get(key);
    if (value !== undefined) {
      // 移动到最近使用
      this.cache.delete(key);
      this.cache.set(key, value);
    }
    return value;
  }
  
  set(key: K, value: V): void {
    if (this.cache.has(key)) {
      this.cache.delete(key);
    } else if (this.cache.size >= this.maxSize) {
      // 移除最旧的
      const firstKey = this.cache.keys().next().value;
      this.cache.delete(firstKey);
    }
    this.cache.set(key, value);
  }
  
  has(key: K): boolean {
    return this.cache.has(key);
  }
  
  clear(): void {
    this.cache.clear();
  }
  
  get size(): number {
    return this.cache.size;
  }
}

// ============================================
// 增量更新配置
// ============================================

interface IncrementalConfig {
  cacheSize: number;              // 缓存大小
  maxCacheEntries: number;        // 最大缓存条目数
  compressionEnabled: boolean;    // 是否启用压缩
  compressionThreshold: number;   // 压缩阈值（字节）
  minCompressionSize: number;     // 最小压缩大小
  chunkSize: number;              // 分块大小
}

// 简单的gzip/gunzip占位符
function gzip(data: Uint8Array): Uint8Array {
  // 实际实现使用zlib或其他压缩库
  return data;
}

function gunzip(data: Uint8Array): Uint8Array {
  // 实际实现使用zlib或其他压缩库
  return data;
}
```

### 6.3 内存管理优化

```typescript
// ============================================
// 内存管理器
// ============================================

/**
 * 内存管理器
 */
class MemoryManager {
  // Yjs文档
  private ydoc: Y.Doc;
  
  // 配置
  private config: MemoryConfig;
  
  // 垃圾回收定时器
  private gcTimer: NodeJS.Timeout | null = null;
  
  // 内存使用统计
  private stats: MemoryStats;
  
  // 观察者
  private observers: Map<string, () => void> = new Map();
  
  constructor(ydoc: Y.Doc, config: MemoryConfig) {
    this.ydoc = ydoc;
    this.config = config;
    this.stats = {
      currentSize: 0,
      peakSize: 0,
      gcCount: 0,
      freedBytes: 0
    };
    
    this.setupGarbageCollection();
    this.setupObservers();
  }
  
  /**
   * 设置垃圾回收
   */
  private setupGarbageCollection(): void {
    if (this.config.gcInterval > 0) {
      this.gcTimer = setInterval(() => {
        this.performGarbageCollection();
      }, this.config.gcInterval);
    }
  }
  
  /**
   * 设置观察者
   */
  private setupObservers(): void {
    // 监听文档变化
    this.ydoc.on('update', () => {
      this.updateStats();
    });
  }
  
  /**
   * 执行垃圾回收
   */
  performGarbageCollection(): void {
    const beforeSize = this.getDocumentSize();
    
    // 清理未引用的元素
    this.cleanupUnreferencedElements();
    
    // 清理过期缓存
    this.cleanupExpiredCache();
    
    // 压缩文档
    this.compactDocument();
    
    const afterSize = this.getDocumentSize();
    const freed = beforeSize - afterSize;
    
    this.stats.gcCount++;
    this.stats.freedBytes += freed;
    
    console.log(`GC completed: freed ${freed} bytes`);
  }
  
  /**
   * 清理未引用的元素
   */
  private cleanupUnreferencedElements(): void {
    const elements = this.ydoc.getMap('elements');
    const layers = this.ydoc.getMap('layers');
    const referencedElements = new Set<string>();
    
    // 收集所有被引用的元素
    for (const [layerId, layer] of layers.entries()) {
      const layerElements = (layer as Y.Map<any>).get('elements') as Y.Map<boolean>;
      if (layerElements) {
        for (const elementId of layerElements.keys()) {
          referencedElements.add(elementId);
        }
      }
    }
    
    // 删除未引用的元素
    for (const elementId of elements.keys()) {
      if (!referencedElements.has(elementId)) {
        // 检查元素是否被锁定
        const element = elements.get(elementId) as Y.Map<any>;
        const lockInfo = element.get('lockInfo');
        
        if (!lockInfo || this.isLockExpired(lockInfo)) {
          elements.delete(elementId);
          console.log(`Removed unreferenced element: ${elementId}`);
        }
      }
    }
  }
  
  /**
   * 检查锁是否过期
   */
  private isLockExpired(lockInfo: LockInfo): boolean {
    return Date.now() > lockInfo.expiresAt;
  }
  
  /**
   * 清理过期缓存
   */
  private cleanupExpiredCache(): void {
    // 清理各种缓存
    // 实际实现取决于具体的缓存机制
  }
  
  /**
   * 压缩文档
   */
  private compactDocument(): void {
    // Yjs自动管理内存，这里可以做一些额外的优化
    // 例如：清理历史记录、合并重复数据等
  }
  
  /**
   * 获取文档大小
   */
  getDocumentSize(): number {
    const stateVector = Y.encodeStateAsUpdate(this.ydoc);
    return stateVector.length;
  }
  
  /**
   * 更新统计
   */
  private updateStats(): void {
    this.stats.currentSize = this.getDocumentSize();
    this.stats.peakSize = Math.max(this.stats.peakSize, this.stats.currentSize);
  }
  
  /**
   * 获取内存统计
   */
  getStats(): MemoryStats {
    return { ...this.stats };
  }
  
  /**
   * 检查内存使用
   */
  checkMemoryUsage(): MemoryStatus {
    const currentSize = this.getDocumentSize();
    const usage = currentSize / this.config.maxDocumentSize;
    
    if (usage > 0.9) {
      return { status: 'critical', usage, message: 'Memory usage critical' };
    } else if (usage > 0.75) {
      return { status: 'warning', usage, message: 'Memory usage high' };
    } else {
      return { status: 'normal', usage, message: 'Memory usage normal' };
    }
  }
  
  /**
   * 释放内存
   */
  releaseMemory(): void {
    this.performGarbageCollection();
    
    // 强制垃圾回收（如果环境支持）
    if (global.gc) {
      global.gc();
    }
  }
  
  /**
   * 销毁
   */
  destroy(): void {
    if (this.gcTimer) {
      clearInterval(this.gcTimer);
      this.gcTimer = null;
    }
    
    // 清理观察者
    for (const [key, cleanup] of this.observers) {
      cleanup();
    }
    this.observers.clear();
  }
}

// ============================================
// 内存管理类型定义
// ============================================

interface MemoryConfig {
  maxDocumentSize: number;    // 最大文档大小（字节）
  gcInterval: number;         // 垃圾回收间隔（毫秒）
  cacheTTL: number;           // 缓存过期时间（毫秒）
  enableCompression: boolean; // 是否启用压缩
}

interface MemoryStats {
  currentSize: number;
  peakSize: number;
  gcCount: number;
  freedBytes: number;
}

interface MemoryStatus {
  status: 'normal' | 'warning' | 'critical';
  usage: number;
  message: string;
}
```

### 6.4 网络传输优化

```typescript
// ============================================
// 网络传输优化器
// ============================================

/**
 * 网络传输优化器
 */
class NetworkOptimizer {
  // 配置
  private config: NetworkConfig;
  
  // 传输统计
  private stats: NetworkStats;
  
  // 压缩器
  private compressor: Compressor;
  
  // 连接质量监测
  private connectionQuality: ConnectionQuality;
  
  constructor(config: NetworkConfig) {
    this.config = config;
    this.stats = {
      totalBytesSent: 0,
      totalBytesReceived: 0,
      compressedBytesSent: 0,
      compressedBytesReceived: 0,
      averageLatency: 0,
      packetLoss: 0
    };
    this.compressor = new Compressor(config.compression);
    this.connectionQuality = { latency: 0, bandwidth: 0, packetLoss: 0 };
  }
  
  /**
   * 优化发送数据
   */
  optimizeForSend(data: Uint8Array): OptimizedData {
    const originalSize = data.length;
    
    // 根据连接质量选择优化策略
    if (this.connectionQuality.bandwidth < this.config.lowBandwidthThreshold) {
      // 低带宽：高压缩
      return this.compressWithLevel(data, 'high');
    } else if (originalSize > this.config.compressionThreshold) {
      // 大数据：中等压缩
      return this.compressWithLevel(data, 'medium');
    }
    
    // 小数据：不压缩
    return {
      data,
      compressed: false,
      originalSize,
      compressedSize: originalSize
    };
  }
  
  /**
   * 优化接收数据
   */
  optimizeForReceive(data: OptimizedData): Uint8Array {
    if (data.compressed) {
      return this.compressor.decompress(data.data);
    }
    return data.data;
  }
  
  /**
   * 分块传输
   */
  chunkForTransmission(data: Uint8Array, maxChunkSize: number): DataChunk[] {
    if (data.length <= maxChunkSize) {
      return [{
        index: 0,
        total: 1,
        data,
        checksum: this.calculateChecksum(data)
      }];
    }
    
    const chunks: DataChunk[] = [];
    const totalChunks = Math.ceil(data.length / maxChunkSize);
    
    for (let i = 0; i < totalChunks; i++) {
      const start = i * maxChunkSize;
      const end = Math.min(start + maxChunkSize, data.length);
      const chunk = data.slice(start, end);
      
      chunks.push({
        index: i,
        total: totalChunks,
        data: chunk,
        checksum: this.calculateChecksum(chunk)
      });
    }
    
    return chunks;
  }
  
  /**
   * 重新组装分块
   */
  reassembleChunks(chunks: DataChunk[]): Uint8Array {
    // 按索引排序
    chunks.sort((a, b) => a.index - b.index);
    
    // 验证完整性
    const totalSize = chunks.reduce((sum, chunk) => sum + chunk.data.length, 0);
    const result = new Uint8Array(totalSize);
    
    let offset = 0;
    for (const chunk of chunks) {
      // 验证校验和
      if (!this.verifyChecksum(chunk.data, chunk.checksum)) {
        throw new Error(`Checksum mismatch for chunk ${chunk.index}`);
      }
      
      result.set(chunk.data, offset);
      offset += chunk.data.length;
    }
    
    return result;
  }
  
  /**
   * 压缩数据
   */
  private compressWithLevel(
    data: Uint8Array,
    level: 'low' | 'medium' | 'high'
  ): OptimizedData {
    const compressed = this.compressor.compress(data, level);
    
    // 如果压缩后更大，返回原始数据
    if (compressed.length >= data.length) {
      return {
        data,
        compressed: false,
        originalSize: data.length,
        compressedSize: data.length
      };
    }
    
    return {
      data: compressed,
      compressed: true,
      originalSize: data.length,
      compressedSize: compressed.length
    };
  }
  
  /**
   * 计算校验和
   */
  private calculateChecksum(data: Uint8Array): string {
    // 简单的CRC32实现
    let crc = 0xffffffff;
    for (const byte of data) {
      crc ^= byte;
      for (let i = 0; i < 8; i++) {
        crc = (crc >>> 1) ^ (0xedb88320 & -(crc & 1));
      }
    }
    return (~crc >>> 0).toString(16);
  }
  
  /**
   * 验证校验和
   */
  private verifyChecksum(data: Uint8Array, checksum: string): boolean {
    return this.calculateChecksum(data) === checksum;
  }
  
  /**
   * 更新连接质量
   */
  updateConnectionQuality(latency: number, bandwidth: number, packetLoss: number): void {
    this.connectionQuality = { latency, bandwidth, packetLoss };
    
    // 更新统计
    this.stats.averageLatency = (this.stats.averageLatency * 0.9) + (latency * 0.1);
    this.stats.packetLoss = packetLoss;
  }
  
  /**
   * 获取传输统计
   */
  getStats(): NetworkStats {
    return { ...this.stats };
  }
  
  /**
   * 获取连接质量
   */
  getConnectionQuality(): ConnectionQuality {
    return { ...this.connectionQuality };
  }
}

// ============================================
// 压缩器
// ============================================

class Compressor {
  private config: CompressionConfig;
  
  constructor(config: CompressionConfig) {
    this.config = config;
  }
  
  compress(data: Uint8Array, level: 'low' | 'medium' | 'high'): Uint8Array {
    // 根据级别选择压缩算法
    switch (level) {
      case 'low':
        return this.fastCompress(data);
      case 'medium':
        return this.balancedCompress(data);
      case 'high':
        return this.maximumCompress(data);
      default:
        return data;
    }
  }
  
  decompress(data: Uint8Array): Uint8Array {
    // 检测压缩格式并解压
    if (this.isGzip(data)) {
      return gunzip(data);
    }
    return data;
  }
  
  private fastCompress(data: Uint8Array): Uint8Array {
    // 快速压缩（低CPU占用）
    return gzip(data, { level: 1 });
  }
  
  private balancedCompress(data: Uint8Array): Uint8Array {
    // 平衡压缩
    return gzip(data, { level: 6 });
  }
  
  private maximumCompress(data: Uint8Array): Uint8Array {
    // 最大压缩（高CPU占用）
    return gzip(data, { level: 9 });
  }
  
  private isGzip(data: Uint8Array): boolean {
    return data.length > 2 && data[0] === 0x1f && data[1] === 0x8b;
  }
}

// ============================================
// 网络优化类型定义
// ============================================

interface NetworkConfig {
  compressionThreshold: number;     // 压缩阈值
  lowBandwidthThreshold: number;    // 低带宽阈值
  maxChunkSize: number;             // 最大分块大小
  compression: CompressionConfig;
}

interface CompressionConfig {
  enabled: boolean;
  algorithm: 'gzip' | 'deflate' | 'brotli';
  defaultLevel: number;
}

interface NetworkStats {
  totalBytesSent: number;
  totalBytesReceived: number;
  compressedBytesSent: number;
  compressedBytesReceived: number;
  averageLatency: number;
  packetLoss: number;
}

interface ConnectionQuality {
  latency: number;
  bandwidth: number;
  packetLoss: number;
}

interface OptimizedData {
  data: Uint8Array;
  compressed: boolean;
  originalSize: number;
  compressedSize: number;
}

interface DataChunk {
  index: number;
  total: number;
  data: Uint8Array;
  checksum: string;
}

// gzip with level option
function gzip(data: Uint8Array, options?: { level?: number }): Uint8Array {
  // 实际实现使用zlib
  return data;
}
```

---

## 7. 代码实现示例

### 7.1 CRDT操作代码

```typescript
// ============================================
// CRDT操作实现示例
// ============================================

import * as Y from 'yjs';

/**
 * CRDT文档管理器
 */
export class CRDTDocumentManager {
  private ydoc: Y.Doc;
  private elements: Y.Map<Y.Map<any>>;
  private layers: Y.Map<Y.Map<any>>;
  private properties: Y.Map<Y.Map<any>>;
  private undoManager: Y.UndoManager;
  
  constructor() {
    this.ydoc = new Y.Doc();
    this.elements = this.ydoc.getMap('elements');
    this.layers = this.ydoc.getMap('layers');
    this.properties = this.ydoc.getMap('properties');
    this.undoManager = new Y.UndoManager([this.elements, this.properties]);
  }
  
  /**
   * 创建墙体元素
   */
  createWall(params: WallCreationParams): string {
    const elementId = `wall_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    this.ydoc.transact(() => {
      const wall = new Y.Map();
      
      // 基本属性
      wall.set('id', elementId);
      wall.set('type', 'wall');
      wall.set('layerId', params.layerId);
      
      // 几何属性
      wall.set('startPoint', this.createYPoint(params.startPoint));
      wall.set('endPoint', this.createYPoint(params.endPoint));
      wall.set('height', params.height);
      wall.set('thickness', params.thickness);
      
      // 变换
      wall.set('transform', this.createIdentityTransform());
      
      // 元数据
      wall.set('createdBy', params.userId);
      wall.set('createdAt', Date.now());
      wall.set('modifiedBy', params.userId);
      wall.set('modifiedAt', Date.now());
      
      // 开洞数组
      wall.set('openings', new Y.Array());
      
      // 添加到元素映射
      this.elements.set(elementId, wall);
      
      // 添加到图层
      const layer = this.layers.get(params.layerId);
      if (layer) {
        const layerElements = layer.get('elements') as Y.Map<boolean> || new Y.Map();
        layerElements.set(elementId, true);
        layer.set('elements', layerElements);
      }
    }, params.userId);
    
    return elementId;
  }
  
  /**
   * 更新元素属性
   */
  updateElement(
    elementId: string,
    path: string,
    value: any,
    userId: string
  ): boolean {
    const element = this.elements.get(elementId);
    if (!element) {
      throw new Error(`Element ${elementId} not found`);
    }
    
    this.ydoc.transact(() => {
      // 设置属性值
      this.setNestedValue(element, path.split('.'), this.toYjsValue(value));
      
      // 更新修改信息
      element.set('modifiedBy', userId);
      element.set('modifiedAt', Date.now());
      
      // 更新版本向量
      const versionVector = element.get('versionVector') || {};
      versionVector[userId] = (versionVector[userId] || 0) + 1;
      element.set('versionVector', versionVector);
    }, userId);
    
    return true;
  }
  
  /**
   * 删除元素
   */
  deleteElement(elementId: string, userId: string): boolean {
    const element = this.elements.get(elementId);
    if (!element) {
      return false;
    }
    
    this.ydoc.transact(() => {
      // 从图层中移除
      const layerId = element.get('layerId');
      if (layerId) {
        const layer = this.layers.get(layerId);
        if (layer) {
          const layerElements = layer.get('elements') as Y.Map<boolean>;
          if (layerElements) {
            layerElements.delete(elementId);
          }
        }
      }
      
      // 删除元素
      this.elements.delete(elementId);
      
      // 删除属性
      this.properties.delete(elementId);
    }, userId);
    
    return true;
  }
  
  /**
   * 变换元素
   */
  transformElement(
    elementId: string,
    transform: TransformData,
    userId: string
  ): boolean {
    const element = this.elements.get(elementId);
    if (!element) {
      throw new Error(`Element ${elementId} not found`);
    }
    
    this.ydoc.transact(() => {
      const currentTransform = element.get('transform') as TransformMatrix;
      let newTransform: TransformMatrix;
      
      switch (transform.type) {
        case 'translate':
          newTransform = this.applyTranslation(
            currentTransform,
            transform.params as TranslationParams
          );
          break;
        case 'rotate':
          newTransform = this.applyRotation(
            currentTransform,
            transform.params as RotationParams
          );
          break;
        case 'scale':
          newTransform = this.applyScale(
            currentTransform,
            transform.params as ScaleParams
          );
          break;
        default:
          throw new Error(`Unknown transform type: ${transform.type}`);
      }
      
      element.set('transform', newTransform);
      element.set('modifiedBy', userId);
      element.set('modifiedAt', Date.now());
    }, userId);
    
    return true;
  }
  
  /**
   * 设置嵌套值
   */
  private setNestedValue(target: Y.Map<any>, path: string[], value: any): void {
    if (path.length === 1) {
      target.set(path[0], value);
      return;
    }
    
    const [first, ...rest] = path;
    let next = target.get(first);
    
    if (!next) {
      next = new Y.Map();
      target.set(first, next);
    }
    
    this.setNestedValue(next, rest, value);
  }
  
  /**
   * 转换为Yjs值
   */
  private toYjsValue(value: any): any {
    if (value === null || value === undefined) {
      return value;
    }
    
    if (Array.isArray(value)) {
      const yarray = new Y.Array();
      yarray.push(value.map(v => this.toYjsValue(v)));
      return yarray;
    }
    
    if (typeof value === 'object') {
      const ymap = new Y.Map();
      for (const [key, val] of Object.entries(value)) {
        ymap.set(key, this.toYjsValue(val));
      }
      return ymap;
    }
    
    return value;
  }
  
  /**
   * 创建YPoint
   */
  private createYPoint(point: Point3D): Y.Map<number> {
    const ypoint = new Y.Map<number>();
    ypoint.set('x', point.x);
    ypoint.set('y', point.y);
    ypoint.set('z', point.z);
    return ypoint;
  }
  
  /**
   * 创建单位变换矩阵
   */
  private createIdentityTransform(): TransformMatrix {
    return {
      m11: 1, m12: 0, m13: 0, m14: 0,
      m21: 0, m22: 1, m23: 0, m24: 0,
      m31: 0, m32: 0, m33: 1, m34: 0,
      m41: 0, m42: 0, m43: 0, m44: 1
    };
  }
  
  /**
   * 应用平移
   */
  private applyTranslation(
    matrix: TransformMatrix,
    params: TranslationParams
  ): TransformMatrix {
    // 矩阵乘法实现
    return {
      m11: matrix.m11,
      m12: matrix.m12,
      m13: matrix.m13,
      m14: matrix.m14,
      m21: matrix.m21,
      m22: matrix.m22,
      m23: matrix.m23,
      m24: matrix.m24,
      m31: matrix.m31,
      m32: matrix.m32,
      m33: matrix.m33,
      m34: matrix.m34,
      m41: matrix.m41 + params.x,
      m42: matrix.m42 + params.y,
      m43: matrix.m43 + params.z,
      m44: matrix.m44
    };
  }
  
  /**
   * 应用旋转
   */
  private applyRotation(
    matrix: TransformMatrix,
    params: RotationParams
  ): TransformMatrix {
    // 简化的旋转实现
    // 实际实现需要完整的3D旋转矩阵计算
    return matrix;
  }
  
  /**
   * 应用缩放
   */
  private applyScale(
    matrix: TransformMatrix,
    params: ScaleParams
  ): TransformMatrix {
    return {
      m11: matrix.m11 * params.x,
      m12: matrix.m12,
      m13: matrix.m13,
      m14: matrix.m14,
      m21: matrix.m21,
      m22: matrix.m22 * params.y,
      m23: matrix.m23,
      m24: matrix.m24,
      m31: matrix.m31,
      m32: matrix.m32,
      m33: matrix.m33 * params.z,
      m34: matrix.m34,
      m41: matrix.m41,
      m42: matrix.m42,
      m43: matrix.m43,
      m44: matrix.m44
    };
  }
  
  /**
   * 获取文档状态向量
   */
  getStateVector(): Uint8Array {
    return Y.encodeStateVector(this.ydoc);
  }
  
  /**
   * 应用远程更新
   */
  applyUpdate(update: Uint8Array): void {
    Y.applyUpdate(this.ydoc, update);
  }
  
  /**
   * 编码文档状态
   */
  encodeState(): Uint8Array {
    return Y.encodeStateAsUpdate(this.ydoc);
  }
  
  /**
   * 撤销
   */
  undo(): void {
    this.undoManager.undo();
  }
  
  /**
   * 重做
   */
  redo(): void {
    this.undoManager.redo();
  }
  
  /**
   * 销毁
   */
  destroy(): void {
    this.ydoc.destroy();
  }
}

// ============================================
// 类型定义
// ============================================

interface WallCreationParams {
  layerId: string;
  userId: string;
  startPoint: Point3D;
  endPoint: Point3D;
  height: number;
  thickness: number;
}

interface Point3D {
  x: number;
  y: number;
  z: number;
}

interface TransformMatrix {
  m11: number; m12: number; m13: number; m14: number;
  m21: number; m22: number; m23: number; m24: number;
  m31: number; m32: number; m33: number; m34: number;
  m41: number; m42: number; m43: number; m44: number;
}

interface TransformData {
  type: 'translate' | 'rotate' | 'scale';
  params: TranslationParams | RotationParams | ScaleParams;
}

interface TranslationParams {
  x: number;
  y: number;
  z: number;
}

interface RotationParams {
  x: number;
  y: number;
  z: number;
}

interface ScaleParams {
  x: number;
  y: number;
  z: number;
}
```

### 7.2 同步协议代码

```typescript
// ============================================
// 同步协议实现示例
// ============================================

import WebSocket from 'ws';
import { EventEmitter } from 'events';

/**
 * 同步协议处理器
 */
export class SyncProtocolHandler extends EventEmitter {
  private ws: WebSocket;
  private documentId: string;
  private clientId: string;
  private state: SyncProtocolState;
  private pendingUpdates: Map<string, PendingUpdate>;
  private messageQueue: SyncMessage[];
  private heartbeatInterval: NodeJS.Timeout | null;
  
  constructor(ws: WebSocket, documentId: string) {
    super();
    this.ws = ws;
    this.documentId = documentId;
    this.clientId = '';
    this.state = SyncProtocolState.DISCONNECTED;
    this.pendingUpdates = new Map();
    this.messageQueue = [];
    this.heartbeatInterval = null;
    
    this.setupWebSocketHandlers();
  }
  
  /**
   * 连接
   */
  async connect(authToken: string, initialStateVector: Uint8Array): Promise<void> {
    this.state = SyncProtocolState.CONNECTING;
    
    // 发送连接请求
    const connectMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.CONNECT,
      sender: { clientId: '', userId: '' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        documentId: this.documentId,
        authToken,
        clientInfo: {
          clientType: 'web',
          version: '1.0.0',
          capabilities: {
            supportedTypes: ['yjs'],
            maxMessageSize: 1024 * 1024,
            compression: true,
            binarySupport: true
          }
        },
        initialState: {
          includeHistory: false,
          maxHistorySize: 100
        }
      } as ConnectPayload
    };
    
    this.sendMessage(connectMessage);
    
    // 等待连接确认
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Connection timeout'));
      }, 10000);
      
      const onConnectAck = (message: SyncMessage) => {
        if (message.type === MessageType.CONNECT_ACK) {
          clearTimeout(timeout);
          const payload = message.payload as ConnectAckPayload;
          this.clientId = payload.clientId;
          this.state = SyncProtocolState.CONNECTED;
          this.startHeartbeat();
          resolve();
        }
      };
      
      this.once('connect_ack', onConnectAck);
    });
  }
  
  /**
   * 同步文档
   */
  async sync(stateVector: Uint8Array): Promise<Uint8Array[]> {
    this.state = SyncProtocolState.SYNCING;
    
    // 发送同步请求
    const syncRequest: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.SYNC_REQUEST,
      sender: { clientId: this.clientId, userId: '' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        syncType: 'incremental',
        clientStateVector: stateVector
      } as SyncRequestPayload
    };
    
    this.sendMessage(syncRequest);
    
    // 等待同步响应
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Sync timeout'));
      }, 30000);
      
      const onSyncResponse = (message: SyncMessage) => {
        if (message.type === MessageType.SYNC_RESPONSE) {
          clearTimeout(timeout);
          const payload = message.payload as SyncResponsePayload;
          this.state = SyncProtocolState.SYNCED;
          resolve(payload.updates.map(u => u.data));
        }
      };
      
      this.once('sync_response', onSyncResponse);
    });
  }
  
  /**
   * 发送更新
   */
  async sendUpdate(update: Uint8Array): Promise<void> {
    const updateId = this.generateUpdateId();
    
    const updateMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.UPDATE,
      sender: { clientId: this.clientId, userId: '' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        updateId,
        updateType: UpdateType.LOCAL,
        operations: [],
        beforeStateVector: new Uint8Array(),
        afterStateVector: update,
        dependencies: [],
        timestamp: Date.now()
      } as UpdatePayload
    };
    
    // 添加到待确认队列
    this.pendingUpdates.set(updateId, {
      updateId,
      message: updateMessage,
      sentAt: Date.now(),
      retryCount: 0
    });
    
    this.sendMessage(updateMessage);
    
    // 设置超时重试
    this.scheduleRetry(updateId);
  }
  
  /**
   * 断开连接
   */
  disconnect(reason?: string): void {
    this.state = SyncProtocolState.DISCONNECTING;
    
    // 停止心跳
    this.stopHeartbeat();
    
    // 发送断开消息
    const disconnectMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.DISCONNECT,
      sender: { clientId: this.clientId, userId: '' },
      timestamp: Date.now(),
      versionVector: {},
      payload: { reason }
    };
    
    this.sendMessage(disconnectMessage);
    
    // 关闭WebSocket
    this.ws.close(1000, reason);
    
    this.state = SyncProtocolState.DISCONNECTED;
  }
  
  /**
   * 设置WebSocket处理器
   */
  private setupWebSocketHandlers(): void {
    this.ws.on('message', (data: WebSocket.Data) => {
      this.handleMessage(data);
    });
    
    this.ws.on('close', (code: number, reason: string) => {
      this.emit('close', code, reason);
    });
    
    this.ws.on('error', (error: Error) => {
      this.emit('error', error);
    });
  }
  
  /**
   * 处理消息
   */
  private handleMessage(data: WebSocket.Data): void {
    try {
      const message = JSON.parse(data.toString()) as SyncMessage;
      
      // 处理消息确认
      if (message.type === MessageType.UPDATE_ACK) {
        this.handleUpdateAck(message.payload as UpdateAckPayload);
      }
      
      // 处理远程更新
      if (message.type === MessageType.UPDATE) {
        this.handleRemoteUpdate(message.payload as UpdatePayload);
      }
      
      // 发射事件
      this.emit(message.type.toLowerCase(), message);
      this.emit('message', message);
    } catch (error) {
      this.emit('error', error);
    }
  }
  
  /**
   * 处理更新确认
   */
  private handleUpdateAck(payload: UpdateAckPayload): void {
    const pending = this.pendingUpdates.get(payload.updateId);
    if (pending) {
      this.pendingUpdates.delete(payload.updateId);
      this.emit('update_confirmed', payload);
    }
  }
  
  /**
   * 处理远程更新
   */
  private handleRemoteUpdate(payload: UpdatePayload): void {
    // 发送确认
    const ackMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.UPDATE_ACK,
      sender: { clientId: this.clientId, userId: '' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {
        updateId: payload.updateId,
        ackType: 'full',
        serverStateVector: payload.afterStateVector
      } as UpdateAckPayload
    };
    
    this.sendMessage(ackMessage);
    
    // 发射事件
    this.emit('remote_update', payload);
  }
  
  /**
   * 发送消息
   */
  private sendMessage(message: SyncMessage): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      // 队列消息
      this.messageQueue.push(message);
    }
  }
  
  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      this.sendPing();
    }, 30000);
  }
  
  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
  
  /**
   * 发送ping
   */
  private sendPing(): void {
    const pingMessage: SyncMessage = {
      messageId: this.generateMessageId(),
      type: MessageType.PING,
      sender: { clientId: this.clientId, userId: '' },
      timestamp: Date.now(),
      versionVector: {},
      payload: {}
    };
    
    this.sendMessage(pingMessage);
  }
  
  /**
   * 调度重试
   */
  private scheduleRetry(updateId: string): void {
    setTimeout(() => {
      const pending = this.pendingUpdates.get(updateId);
      if (pending && pending.retryCount < 3) {
        pending.retryCount++;
        this.sendMessage(pending.message);
        this.scheduleRetry(updateId);
      } else if (pending) {
        this.pendingUpdates.delete(updateId);
        this.emit('update_failed', updateId);
      }
    }, 5000);
  }
  
  /**
   * 生成消息ID
   */
  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  /**
   * 生成更新ID
   */
  private generateUpdateId(): string {
    return `upd_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
}

// ============================================
// 类型定义
// ============================================

enum SyncProtocolState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  SYNCING = 'syncing',
  SYNCED = 'synced',
  DISCONNECTING = 'disconnecting'
}

interface PendingUpdate {
  updateId: string;
  message: SyncMessage;
  sentAt: number;
  retryCount: number;
}
```

### 7.3 冲突解决代码

```typescript
// ============================================
// 冲突解决实现示例
// ============================================

/**
 * 冲突解决引擎
 */
export class ConflictResolutionEngine {
  private detector: ConflictDetector;
  private autoMerger: AutoMerger;
  private manualResolver: ManualConflictResolver;
  private notificationManager: ConflictNotificationManager;
  
  constructor(
    notificationManager: ConflictNotificationManager,
    manualResolver: ManualConflictResolver
  ) {
    this.detector = new ConflictDetector();
    this.autoMerger = new AutoMerger();
    this.manualResolver = manualResolver;
    this.notificationManager = notificationManager;
  }
  
  /**
   * 处理操作冲突
   */
  async resolveConflict(
    localOp: Operation,
    remoteOp: Operation
  ): Promise<ConflictResolution> {
    // 检测冲突
    const conflict = this.detector.detect(localOp, remoteOp);
    
    if (!conflict) {
      // 没有冲突，可以并发执行
      return {
        method: 'auto_merge',
        resolvedBy: 'system',
        resolvedAt: Date.now(),
        result: null,
        discardedOperations: []
      };
    }
    
    // 尝试自动合并
    const autoResolution = this.autoMerger.tryMerge(conflict);
    if (autoResolution) {
      conflict.status = ConflictStatus.AUTO_RESOLVED;
      conflict.resolution = autoResolution;
      
      // 广播解决结果
      await this.notificationManager.broadcastResolution(
        '', // documentId
        conflict,
        autoResolution
      );
      
      return autoResolution;
    }
    
    // 需要人工解决
    return this.manualResolver.submitForManualResolution(conflict, {
      notifyUsers: [localOp.origin.userId, remoteOp.origin.userId],
      urgency: this.calculateUrgency(conflict),
      autoResolveTimeout: 300000 // 5分钟
    });
  }
  
  /**
   * 批量解决冲突
   */
  async resolveBatchConflicts(operations: Operation[]): Promise<ConflictResolution[]> {
    const conflicts = this.detector.detectBatch(operations);
    const resolutions: ConflictResolution[] = [];
    
    for (const conflict of conflicts) {
      // 获取冲突的操作
      const conflictOps = operations.filter(op =>
        conflict.operations.includes(op.id)
      );
      
      if (conflictOps.length >= 2) {
        const resolution = await this.resolveConflict(
          conflictOps[0],
          conflictOps[1]
        );
        resolutions.push(resolution);
      }
    }
    
    return resolutions;
  }
  
  /**
   * 计算紧急程度
   */
  private calculateUrgency(conflict: ConflictMetadata): string {
    // 根据冲突类型和影响范围计算紧急程度
    switch (conflict.type) {
      case ConflictType.EDIT_AFTER_DELETE:
        return 'high';
      case ConflictType.STRUCTURE_CONFLICT:
        return 'critical';
      case ConflictType.DEPENDENCY_CONFLICT:
        return 'high';
      default:
        return 'normal';
    }
  }
  
  /**
   * 应用解决结果
   */
  applyResolution(
    conflict: ConflictMetadata,
    resolution: ConflictResolution,
    documentManager: CRDTDocumentManager
  ): void {
    switch (resolution.method) {
      case 'last_write_wins':
        this.applyLastWriteWins(conflict, resolution, documentManager);
        break;
      case 'auto_merge':
        this.applyAutoMerge(conflict, resolution, documentManager);
        break;
      case 'manual':
        this.applyManualResolution(conflict, resolution, documentManager);
        break;
      case 'revert':
        this.applyRevert(conflict, resolution, documentManager);
        break;
    }
  }
  
  /**
   * 应用LWW解决
   */
  private applyLastWriteWins(
    conflict: ConflictMetadata,
    resolution: ConflictResolution,
    documentManager: CRDTDocumentManager
  ): void {
    // LWW已经在检测时确定，这里只需确认应用
    const result = resolution.result;
    if (result && result.propertyPath) {
      documentManager.updateElement(
        conflict.elementId,
        result.propertyPath,
        result.value,
        resolution.resolvedBy
      );
    }
  }
  
  /**
   * 应用自动合并
   */
  private applyAutoMerge(
    conflict: ConflictMetadata,
    resolution: ConflictResolution,
    documentManager: CRDTDocumentManager
  ): void {
    const result = resolution.result;
    
    // 应用合并后的值
    if (result) {
      for (const [key, value] of Object.entries(result)) {
        documentManager.updateElement(
          conflict.elementId,
          key,
          value,
          resolution.resolvedBy
        );
      }
    }
  }
  
  /**
   * 应用人工解决
   */
  private applyManualResolution(
    conflict: ConflictMetadata,
    resolution: ConflictResolution,
    documentManager: CRDTDocumentManager
  ): void {
    // 应用人工选择的值
    const result = resolution.result;
    
    if (result && result.manualValue) {
      documentManager.updateElement(
        conflict.elementId,
        result.propertyPath || '',
        result.manualValue,
        resolution.resolvedBy
      );
    }
  }
  
  /**
   * 应用撤销
   */
  private applyRevert(
    conflict: ConflictMetadata,
    resolution: ConflictResolution,
    documentManager: CRDTDocumentManager
  ): void {
    // 撤销到冲突前的状态
    // 实际实现可能需要保存冲突前的状态
    console.log(`Reverting changes for conflict ${conflict.conflictId}`);
  }
}

/**
 * 冲突检测器实现
 */
class ConflictDetector {
  private rules: ConflictRule[];
  
  constructor() {
    this.rules = this.setupRules();
  }
  
  detect(op1: Operation, op2: Operation): ConflictMetadata | null {
    // 快速检查：不同目标不冲突
    if (op1.target.id !== op2.target.id) {
      return null;
    }
    
    // 应用规则
    for (const rule of this.rules) {
      const conflict = rule.check(op1, op2);
      if (conflict) {
        return conflict;
      }
    }
    
    return null;
  }
  
  detectBatch(operations: Operation[]): ConflictMetadata[] {
    const conflicts: ConflictMetadata[] = [];
    
    for (let i = 0; i < operations.length; i++) {
      for (let j = i + 1; j < operations.length; j++) {
        const conflict = this.detect(operations[i], operations[j]);
        if (conflict) {
          conflicts.push(conflict);
        }
      }
    }
    
    return conflicts;
  }
  
  private setupRules(): ConflictRule[] {
    return [
      // 并发编辑规则
      {
        name: 'concurrent_edit',
        check: (op1, op2) => {
          if (op1.type === 'ELEMENT_UPDATE' && op2.type === 'ELEMENT_UPDATE') {
            const data1 = op1.data as UpdateElementData;
            const data2 = op2.data as UpdateElementData;
            
            if (data1.path === data2.path) {
              return {
                conflictId: `conflict_${Date.now()}`,
                type: ConflictType.CONCURRENT_EDIT,
                elementId: op1.target.id,
                operations: [op1.id, op2.id],
                detectedAt: Date.now(),
                status: ConflictStatus.DETECTED,
                details: {
                  propertyPath: data1.path,
                  values: [
                    { clientId: op1.origin.clientId, value: data1.value },
                    { clientId: op2.origin.clientId, value: data2.value }
                  ]
                }
              };
            }
          }
          return null;
        }
      }
    ];
  }
}

/**
 * 自动合并器实现
 */
class AutoMerger {
  private strategies: Map<string, MergeStrategy>;
  
  constructor() {
    this.strategies = new Map();
    this.setupStrategies();
  }
  
  tryMerge(conflict: ConflictMetadata): ConflictResolution | null {
    const strategy = this.strategies.get(conflict.type);
    
    if (strategy && strategy.canAutoMerge(conflict)) {
      return strategy.merge(conflict);
    }
    
    return null;
  }
  
  private setupStrategies(): void {
    // LWW策略
    this.strategies.set(ConflictType.CONCURRENT_EDIT, {
      canAutoMerge: (conflict) => {
        const values = conflict.details.values;
        return values && values.every(v => typeof v.value === 'number');
      },
      merge: (conflict) => {
        const values = conflict.details.values;
        const lastValue = values.reduce((latest, current) =>
          current.timestamp > latest.timestamp ? current : latest
        );
        
        return {
          method: 'last_write_wins',
          resolvedBy: 'system',
          resolvedAt: Date.now(),
          result: {
            propertyPath: conflict.details.propertyPath,
            value: lastValue.value
          },
          discardedOperations: values
            .filter(v => v.clientId !== lastValue.clientId)
            .map(v => v.operationId)
        };
      }
    });
  }
}

// ============================================
// 类型定义
// ============================================

interface ConflictRule {
  name: string;
  check(op1: Operation, op2: Operation): ConflictMetadata | null;
}

interface MergeStrategy {
  canAutoMerge(conflict: ConflictMetadata): boolean;
  merge(conflict: ConflictMetadata): ConflictResolution;
}
```

---

## 附录

### A. 术语表

| 术语 | 说明 |
|------|------|
| CRDT | 无冲突复制数据类型，支持最终一致性的分布式数据结构 |
| Yjs | 一个流行的CRDT实现库 |
| OT | 操作转换，一种协作编辑算法 |
| LWW | Last Write Wins，最后写入获胜策略 |
| 版本向量 | 用于追踪分布式系统中事件因果关系的向量时钟 |
| 因果一致性 | 保证因果相关操作按正确顺序执行的一致性模型 |

### B. 参考资料

1. Yjs官方文档: https://docs.yjs.dev/
2. CRDT技术论文: "A comprehensive study of Convergent and Commutative Replicated Data Types"
3. WebSocket协议: RFC 6455
4. Operational Transformation: "Operational Transformation in Real-Time Group Editors"

### C. 设计决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| CRDT库 | Yjs | 成熟稳定，社区活跃，TypeScript支持好 |
| 一致性模型 | 因果一致性 | 平衡一致性和性能，适合建筑设计场景 |
| 冲突解决 | 自动+人工 | 简单冲突自动解决，复杂冲突人工介入 |
| 传输协议 | WebSocket | 实时双向通信，低延迟 |

---

**文档结束**

*本报告为半自动化建筑设计平台协作引擎的详细设计文档，供详细设计评审使用。*
