
# 概要设计阶段 - 并发协作架构设计报告

## 半自动化建筑设计平台

---

## 文档信息

| 项目 | 内容 |
|------|------|
| 文档版本 | v1.0 |
| 设计阶段 | 概要设计阶段 |
| 设计范围 | 并发协作架构 |
| 关键技术 | CRDT、乐观锁、MVCC、因果一致性 |

---

## 目录

1. [协作架构总体设计](#1-协作架构总体设计)
2. [CRDT引擎设计](#2-crdt引擎设计)
3. [实时同步机制设计](#3-实时同步机制设计)
4. [并发控制设计](#4-并发控制设计)
5. [一致性设计](#5-一致性设计)
6. [性能优化设计](#6-性能优化设计)
7. [容错设计](#7-容错设计)

---

## 1. 协作架构总体设计

### 1.1 协作系统分层架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        协作系统分层架构                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      应用层 (Application Layer)                  │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │   │
│  │  │  画布协作    │  │  属性协作   │  │  评论协作   │  │ 光标同步 │ │   │
│  │  │  Canvas     │  │  Property   │  │  Comment    │  │ Cursor  │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              ▲                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      协作引擎层 (Collaboration Engine)           │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │   │
│  │  │  CRDT引擎   │  │  操作转换   │  │  冲突解决   │  │ 状态管理 │ │   │
│  │  │  CRDT       │  │  OT         │  │  Resolver   │  │ State   │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              ▲                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      通信层 (Communication Layer)                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │   │
│  │  │ WebSocket   │  │ 消息队列    │  │ 广播服务    │  │ 房间管理 │ │   │
│  │  │ Gateway     │  │ Pub/Sub     │  │ Broadcast   │  │ Room    │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              ▲                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      存储层 (Storage Layer)                      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │   │
│  │  │ 文档存储    │  │ 操作日志    │  │ 快照存储    │  │ 版本控制 │ │   │
│  │  │ Document    │  │ Operation   │  │ Snapshot    │  │ Version │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 协作数据流设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           协作数据流架构                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   客户端A                    服务端                     客户端B              │
│  ┌─────────┐              ┌─────────┐               ┌─────────┐            │
│  │ 用户操作 │              │         │               │ 用户操作 │            │
│  └────┬────┘              │         │               └────┬────┘            │
│       ▼                   │         │                    ▼                 │
│  ┌─────────┐              │         │               ┌─────────┐            │
│  │本地CRDT │              │         │               │本地CRDT │            │
│  │  更新   │              │         │               │  更新   │            │
│  └────┬────┘              │         │               └────┬────┘            │
│       ▼                   │         │                    ▼                 │
│  ┌─────────┐    WS       ┌─────────┐      WS        ┌─────────┐            │
│  │操作消息  │◄──────────►│ 网关    │◄──────────────►│操作消息  │            │
│  │(JSON)   │             │         │                │(JSON)   │            │
│  └────┬────┘             │  ┌───┐  │                └────▲────┘            │
│       │                  │  │   │  │                     │                 │
│       │                  └───┼───┼──┘                     │                 │
│       │                      │   │                         │                 │
│       │    ┌─────────────────┘   └─────────────────┐      │                 │
│       │    ▼                                       ▼      │                 │
│       │  ┌─────────┐                           ┌─────────┐│                 │
│       └──►│ Redis   │◄─────────────────────────►│ Kafka   │┘                 │
│          │ Pub/Sub │      持久化/广播            │ Topic   │                  │
│          └─────────┘                           └─────────┘                  │
│                                                                             │
│  数据流说明:                                                                 │
│  1. 用户操作 → 本地CRDT更新 → 生成操作消息                                   │
│  2. 操作消息 → WebSocket → 网关 → Redis Pub/Sub                              │
│  3. Redis广播 → 其他客户端 + Kafka持久化                                     │
│  4. 客户端接收 → 本地CRDT合并 → UI更新                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.3 协作状态管理

#### 1.3.1 状态机设计

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        协作会话状态机                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│    ┌─────────┐    连接     ┌─────────┐   加入文档   ┌─────────┐        │
│    │  OFFLINE │ ─────────► │CONNECTED│ ───────────► │ SYNCING │        │
│    │  离线    │            │ 已连接   │              │ 同步中   │        │
│    └─────────┘            └────┬────┘              └────┬────┘        │
│         ▲                      │                        │              │
│         │                      │ 连接失败               │ 同步完成      │
│         │                      ▼                        ▼              │
│         │               ┌─────────┐              ┌─────────┐           │
│         │               │  ERROR  │              │  ACTIVE │           │
│         │               │ 错误状态 │              │ 协作中  │◄─────────┤
│         │               └─────────┘              └────┬────┘           │
│         │                                             │                │
│         │              ┌──────────────────────────────┤                │
│         │              │         网络中断              │                │
│         │              ▼                              │                │
│         │        ┌─────────┐    重连成功              │                │
│         └────────┤RECONNECT│ ────────────────────────┘                │
│                  │重连中   │                                           │
│                  └─────────┘    重连失败                               │
│                       │                                               │
│                       └──────────────────────────────────────────────►│
│                                                                         │
│  状态说明:                                                               │
│  - OFFLINE:  初始状态，未建立连接                                        │
│  - CONNECTED: WebSocket连接成功                                          │
│  - SYNCING:  正在同步文档初始状态                                        │
│  - ACTIVE:   正常协作状态，可收发操作                                    │
│  - RECONNECT: 网络中断，尝试重连                                         │
│  - ERROR:    发生错误，需要人工干预                                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 1.3.2 状态管理架构

```typescript
// 协作状态管理架构
interface CollaborationState {
  // 会话状态
  session: SessionState;
  // 文档状态
  document: DocumentState;
  // 用户状态
  users: Map<string, UserState>;
  // 操作队列
  pendingOps: Operation[];
  // 确认队列
  ackQueue: Map<string, Operation>;
}

interface SessionState {
  status: 'offline' | 'connected' | 'syncing' | 'active' | 'reconnect' | 'error';
  sessionId: string;
  documentId: string;
  userId: string;
  lastActivity: number;
  reconnectAttempts: number;
}

interface DocumentState {
  // CRDT文档状态
  crdt: Y.Doc;
  // 版本向量
  versionVector: VersionVector;
  // 最后同步时间
  lastSyncTime: number;
  // 本地操作计数
  localOpCount: number;
}
```

### 1.4 协作会话管理

#### 1.4.1 会话生命周期

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        协作会话生命周期                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  阶段1: 会话建立                                                          │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐              │
│  │ 用户认证 │───►│ 权限校验 │───►│ 创建会话 │───►│ 加入房间 │              │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘              │
│       │              │              │              │                    │
│       ▼              ▼              ▼              ▼                    │
│  JWT验证         RBAC检查      Session记录      Room注册                │
│  Token解析       文档权限      心跳机制         用户列表                  │
│                                                                         │
│  阶段2: 会话维护                                                          │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐              │
│  │ 心跳检测 │◄──►│ 状态同步 │◄──►│ 操作转发 │◄──►│ 冲突处理 │              │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘              │
│  (30s间隔)      (增量更新)      (广播机制)      (CRDT合并)               │
│                                                                         │
│  阶段3: 会话结束                                                          │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐              │
│  │ 离开通知 │───►│ 状态保存 │───►│ 资源释放 │───►│ 会话销毁 │              │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘              │
│       │              │              │              │                    │
│       ▼              ▼              ▼              ▼                    │
│  广播离开事件    持久化CRDT     关闭WebSocket    清理内存                 │
│  更新用户列表    保存快照       取消订阅         删除会话记录              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 1.4.2 房间管理设计

```typescript
// 房间管理架构
interface RoomManager {
  // 房间集合
  rooms: Map<string, Room>;

  // 创建房间
  createRoom(documentId: string): Room;

  // 加入房间
  joinRoom(documentId: string, userId: string, ws: WebSocket): void;

  // 离开房间
  leaveRoom(documentId: string, userId: string): void;

  // 广播消息
  broadcast(documentId: string, message: Message, exclude?: string[]): void;

  // 获取房间用户
  getRoomUsers(documentId: string): UserInfo[];
}

interface Room {
  documentId: string;
  users: Map<string, UserConnection>;
  crdt: Y.Doc;
  versionVector: VersionVector;
  createdAt: number;
  lastActivity: number;
}

interface UserConnection {
  userId: string;
  userName: string;
  userColor: string;
  ws: WebSocket;
  joinedAt: number;
  cursor?: CursorPosition;
  selection?: SelectionRange;
}
```

---

## 2. CRDT引擎设计

### 2.1 CRDT数据类型设计

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      CRDT数据类型体系                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Yjs CRDT 类型系统                            │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │   │
│  │  │  Y.Array    │    │  Y.Map      │    │  Y.Text     │         │   │
│  │  │  有序列表   │    │  键值映射   │    │  富文本     │         │   │
│  │  │             │    │             │    │             │         │   │
│  │  │ • 图层列表  │    │ • 元素属性  │    │ • 注释内容  │         │   │
│  │  │ • 元素集合  │    │ • 元数据    │    │ • 描述文本  │         │   │
│  │  │ • 历史记录  │    │ • 配置项    │    │ • 富文本    │         │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘         │   │
│  │         │                  │                  │                 │   │
│  │         └──────────────────┼──────────────────┘                 │   │
│  │                            ▼                                    │   │
│  │                   ┌─────────────┐                               │   │
│  │                   │  Y.Doc      │                               │   │
│  │                   │  文档根节点  │                               │   │
│  │                   │             │                               │   │
│  │                   │ • 版本控制  │                               │   │
│  │                   │ • 事务管理  │                               │   │
│  │                   │ • 事件监听  │                               │   │
│  │                   └─────────────┘                               │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  类型映射关系:                                                           │
│  ┌────────────────┬─────────────────┬────────────────────────────────┐ │
│  │ 业务概念        │ CRDT类型        │ 说明                            │ │
│  ├────────────────┼─────────────────┼────────────────────────────────┤ │
│  │ 建筑图层        │ Y.Array<Layer>  │ 有序图层列表，支持重排序         │ │
│  │ 图层元素        │ Y.Array<Element>│ 图层内元素集合                   │ │
│  │ 元素属性        │ Y.Map           │ 几何/样式/业务属性               │ │
│  │ 元素几何        │ Y.Map           │ 坐标、尺寸、变换矩阵             │ │
│  │ 注释内容        │ Y.Text          │ 协作评论，支持富文本             │ │
│  │ 用户光标        │ Y.Map           │ 光标位置和选择范围               │ │
│  │ 操作历史        │ Y.Array         │ 操作日志，用于撤销重做           │ │
│  └────────────────┴─────────────────┴────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 几何数据CRDT实现

#### 2.2.1 几何数据结构

```typescript
// 几何数据CRDT实现
interface GeometryCRDT {
  // 使用Y.Map存储几何属性
  type: 'rectangle' | 'circle' | 'polygon' | 'line' | 'path';

  // 位置属性 (Y.Map)
  position: Y.Map<{
    x: number;      // X坐标
    y: number;      // Y坐标
    z: number;      // Z坐标（层高）
  }>;

  // 尺寸属性 (Y.Map)
  size: Y.Map<{
    width: number;  // 宽度
    height: number; // 高度
    depth: number;  // 深度（3D）
  }>;

  // 变换矩阵 (Y.Array)
  transform: Y.Array<number>; // [a, b, c, d, e, f] 2D变换矩阵

  // 旋转角度 (Y.Map)
  rotation: Y.Map<{
    angle: number;  // 旋转角度
    cx: number;     // 旋转中心X
    cy: number;     // 旋转中心Y
  }>;

  // 顶点集合 (Y.Array) - 用于多边形和路径
  points: Y.Array<Point>;
}

// 建筑元素CRDT文档结构
interface BuildingElementCRDT {
  // 元素ID (唯一标识)
  id: string;

  // 元素类型
  elementType: 'wall' | 'door' | 'window' | 'room' | 'furniture' | 'annotation';

  // 几何数据 (Y.Map)
  geometry: Y.Map<GeometryCRDT>;

  // 样式属性 (Y.Map)
  style: Y.Map<{
    fill: string;
    stroke: string;
    strokeWidth: number;
    opacity: number;
  }>;

  // 业务属性 (Y.Map)
  properties: Y.Map<{
    name: string;
    description: string;
    tags: Y.Array<string>;
    metadata: Y.Map<any>;
  }>;

  // 层级关系
  layerId: string;
  parentId: string | null;
  children: Y.Array<string>;

  // 版本信息
  version: number;
  createdAt: number;
  updatedAt: number;
  createdBy: string;
  updatedBy: string;
}
```

#### 2.2.2 几何操作CRDT算法

```typescript
// 几何操作CRDT实现
class GeometryCRDTEngine {
  private doc: Y.Doc;
  private elements: Y.Map<Y.Map<any>>;

  constructor() {
    this.doc = new Y.Doc();
    this.elements = this.doc.getMap('elements');
  }

  // 创建元素 - 使用LWW-Element-Set (Last-Write-Wins Element Set)
  createElement(elementId: string, geometry: GeometryData): void {
    this.doc.transact(() => {
      const element = new Y.Map();

      // 设置几何属性
      const geomMap = new Y.Map();
      geomMap.set('type', geometry.type);
      geomMap.set('position', this.createPositionMap(geometry.position));
      geomMap.set('size', this.createSizeMap(geometry.size));
      geomMap.set('transform', new Y.Array(geometry.transform || [1,0,0,1,0,0]));

      element.set('geometry', geomMap);
      element.set('version', 1);
      element.set('updatedAt', Date.now());

      // LWW: 使用时间戳作为最后写入胜出的依据
      this.elements.set(elementId, element);
    });
  }

  // 更新元素位置 - 使用Y.Map的merge语义
  updatePosition(elementId: string, newPosition: Position): void {
    this.doc.transact(() => {
      const element = this.elements.get(elementId);
      if (!element) return;

      const geometry = element.get('geometry') as Y.Map<any>;
      const position = geometry.get('position') as Y.Map<number>;

      // Y.Map自动处理并发更新，使用last-write-wins语义
      position.set('x', newPosition.x);
      position.set('y', newPosition.y);

      // 递增版本
      element.set('version', (element.get('version') as number) + 1);
      element.set('updatedAt', Date.now());
    });
  }

  // 批量更新 - 使用单一事务保证原子性
  batchUpdate(updates: ElementUpdate[]): void {
    this.doc.transact(() => {
      updates.forEach(update => {
        const element = this.elements.get(update.elementId);
        if (element) {
          // 应用更新...
          this.applyUpdate(element, update);
        }
      });
    }, this); // 第二个参数是origin，用于追踪操作来源
  }

  // 删除元素 - 使用墓碑标记实现软删除
  deleteElement(elementId: string): void {
    this.doc.transact(() => {
      const element = this.elements.get(elementId);
      if (element) {
        // 设置删除标记而非真正删除
        element.set('_deleted', true);
        element.set('deletedAt', Date.now());
        element.set('deletedBy', this.currentUserId);
      }
    });
  }

  // 获取更新操作 - 用于网络同步
  getUpdate(): Uint8Array {
    return Y.encodeStateAsUpdate(this.doc);
  }

  // 应用远程更新 - CRDT合并
  applyUpdate(update: Uint8Array): void {
    Y.applyUpdate(this.doc, update);
  }
}
```

### 2.3 属性数据CRDT实现

#### 2.3.1 属性数据结构设计

```typescript
// 属性数据CRDT实现
interface PropertyCRDT {
  // 基础属性 (Y.Map) - 使用LWW-Register
  basic: Y.Map<string | number | boolean>;

  // 嵌套对象 (Y.Map) - 递归CRDT结构
  nested: Y.Map<Y.Map<any>>;

  // 数组属性 (Y.Array) - 使用RGA (Replicated Growable Array)
  arrays: Map<string, Y.Array<any>>;

  // 文本属性 (Y.Text) - 使用YATA (Yet Another Transformation Approach)
  texts: Map<string, Y.Text>;
}

// 建筑元素属性定义
interface BuildingProperties {
  // 基础信息
  basic: {
    name: string;           // 元素名称
    category: string;       // 分类
    level: number;          // 楼层
    area: number;           // 面积
  };

  // 材料信息
  material: {
    type: string;           // 材料类型
    color: string;          // 颜色
    texture: string;        // 纹理
    cost: number;           // 成本
  };

  // 结构信息
  structure: {
    loadBearing: boolean;   // 是否承重
    thickness: number;      // 厚度
    height: number;         // 高度
  };

  // 标签列表
  tags: string[];

  // 描述文本
  description: string;
}
```

#### 2.3.2 属性CRDT操作实现

```typescript
// 属性CRDT引擎
class PropertyCRDTEngine {
  private properties: Y.Map<Y.Map<any>>;

  constructor(doc: Y.Doc) {
    this.properties = doc.getMap('properties');
  }

  // 设置属性值 - LWW-Register语义
  setProperty(elementId: string, path: string, value: any): void {
    const elementProps = this.getOrCreateElementProps(elementId);

    // 解析路径 (e.g., "material.color")
    const keys = path.split('.');
    let current = elementProps;

    for (let i = 0; i < keys.length - 1; i++) {
      const key = keys[i];
      if (!current.has(key)) {
        current.set(key, new Y.Map());
      }
      current = current.get(key) as Y.Map<any>;
    }

    // 设置最终值 - LWW语义自动处理并发
    current.set(keys[keys.length - 1], value);
  }

  // 数组操作 - RGA语义
  arrayOperation(
    elementId: string, 
    arrayPath: string, 
    operation: 'insert' | 'delete' | 'move',
    params: ArrayOperationParams
  ): void {
    const elementProps = this.getOrCreateElementProps(elementId);
    let array = elementProps.get(arrayPath) as Y.Array<any>;

    if (!array) {
      array = new Y.Array();
      elementProps.set(arrayPath, array);
    }

    switch (operation) {
      case 'insert':
        // RGA插入 - 在指定位置插入，并发插入按origin排序
        array.insert(params.index, [params.value]);
        break;
      case 'delete':
        // RGA删除 - 标记删除，保留位置
        array.delete(params.index, params.length || 1);
        break;
      case 'move':
        // 移动元素 - 先删除后插入，保持因果一致性
        const [moved] = array.slice(params.fromIndex, params.fromIndex + 1);
        array.delete(params.fromIndex, 1);
        array.insert(params.toIndex, [moved]);
        break;
    }
  }

  // 文本操作 - YATA语义
  textOperation(
    elementId: string,
    textPath: string,
    operation: 'insert' | 'delete',
    params: TextOperationParams
  ): void {
    const elementProps = this.getOrCreateElementProps(elementId);
    let text = elementProps.get(textPath) as Y.Text;

    if (!text) {
      text = new Y.Text();
      elementProps.set(textPath, text);
    }

    switch (operation) {
      case 'insert':
        // YATA插入 - 基于位置的并发文本编辑
        text.insert(params.index, params.content);
        break;
      case 'delete':
        // YATA删除 - 标记删除区间
        text.delete(params.index, params.length);
        break;
    }
  }

  // 获取属性值
  getProperty(elementId: string, path: string): any {
    const elementProps = this.properties.get(elementId);
    if (!elementProps) return undefined;

    const keys = path.split('.');
    let current: any = elementProps;

    for (const key of keys) {
      if (current instanceof Y.Map) {
        current = current.get(key);
      } else {
        return undefined;
      }
    }

    return current;
  }

  private getOrCreateElementProps(elementId: string): Y.Map<any> {
    let props = this.properties.get(elementId);
    if (!props) {
      props = new Y.Map();
      this.properties.set(elementId, props);
    }
    return props;
  }
}
```

### 2.4 CRDT同步协议

#### 2.4.1 同步协议架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      CRDT同步协议架构                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     同步协议分层                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │  应用层协议 (Application Protocol)                       │   │   │
│  │  │  • 操作语义定义                                          │   │   │
│  │  │  • 业务逻辑封装                                          │   │   │
│  │  │  • 冲突解决策略                                          │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │  CRDT层协议 (CRDT Protocol)                              │   │   │
│  │  │  • Yjs Update编码                                        │   │   │
│  │  │  • State Vector交换                                      │   │   │
│  │  │  • Delta计算                                             │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │  传输层协议 (Transport Protocol)                         │   │   │
│  │  │  • WebSocket消息帧                                       │   │   │
│  │  │  • 二进制数据传输                                        │   │   │
│  │  │  • 心跳与ACK机制                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  消息类型定义:                                                           │
│  ┌─────────────────┬──────────────┬───────────────────────────────────┐ │
│  │ 消息类型         │ 消息格式      │ 说明                              │ │
│  ├─────────────────┼──────────────┼───────────────────────────────────┤ │
│  │ SYNC_STEP1      │ {sv: SV}     │ 客户端发送StateVector请求同步      │ │
│  │ SYNC_STEP2      │ {update: []} │ 服务端返回缺失的update             │ │
│  │ UPDATE          │ {update: []} │ 实时操作更新广播                   │ │
│  │ AWARENESS       │ {aw: {...}}  │ 用户状态更新（光标、选择等）       │ │
│  │ AUTH            │ {token: ...} │ 连接认证                           │ │
│  │ ERROR           │ {code, msg}  │ 错误响应                           │ │
│  └─────────────────┴──────────────┴───────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 2.4.2 同步协议实现

```typescript
// CRDT同步协议实现
class CRDTSyncProtocol {
  private doc: Y.Doc;
  private provider: WebsocketProvider;
  private awareness: awarenessProtocol.Awareness;

  constructor(doc: Y.Doc, wsUrl: string) {
    this.doc = doc;
    this.provider = new WebsocketProvider(wsUrl, 'room-name', doc);
    this.awareness = this.provider.awareness;

    this.setupHandlers();
  }

  // 设置事件处理器
  private setupHandlers(): void {
    // 同步状态监听
    this.provider.on('sync', (isSynced: boolean) => {
      console.log('Sync status:', isSynced ? 'synced' : 'syncing');
    });

    // 更新监听
    this.doc.on('update', (update: Uint8Array, origin: any) => {
      // 本地更新，origin为当前实例
      if (origin === this) {
        this.onLocalUpdate(update);
      } else {
        // 远程更新
        this.onRemoteUpdate(update);
      }
    });

    // Awareness更新监听
    this.awareness.on('change', ({ added, updated, removed }: any) => {
      const changedClients = [...added, ...updated, ...removed];
      const states = this.awareness.getStates();

      changedClients.forEach(clientId => {
        const state = states.get(clientId);
        if (state) {
          this.onAwarenessChange(clientId, state);
        }
      });
    });
  }

  // 本地更新处理
  private onLocalUpdate(update: Uint8Array): void {
    // 更新已通过provider自动发送
    // 可在此处添加本地UI更新逻辑
  }

  // 远程更新处理
  private onRemoteUpdate(update: Uint8Array): void {
    // CRDT自动合并，应用更新
    Y.applyUpdate(this.doc, update);

    // 触发UI更新
    this.emit('documentChange', {
      update,
      timestamp: Date.now()
    });
  }

  // Awareness变化处理
  private onAwarenessChange(clientId: number, state: any): void {
    this.emit('awarenessChange', {
      clientId,
      user: state.user,
      cursor: state.cursor,
      selection: state.selection
    });
  }

  // 设置本地用户状态
  setLocalState(state: AwarenessState): void {
    this.awareness.setLocalState(state);
  }

  // 获取所有用户状态
  getAllStates(): Map<number, AwarenessState> {
    return this.awareness.getStates();
  }

  // 手动同步 - 用于断线重连
  async sync(): Promise<void> {
    const stateVector = Y.encodeStateVector(this.doc);

    // 发送同步请求
    this.provider.sendSyncStep1(stateVector);

    return new Promise((resolve) => {
      const handler = (isSynced: boolean) => {
        if (isSynced) {
          this.provider.off('sync', handler);
          resolve();
        }
      };
      this.provider.on('sync', handler);
    });
  }

  // 销毁连接
  destroy(): void {
    this.awareness.destroy();
    this.provider.destroy();
  }
}

// 消息编码/解码
class MessageCodec {
  // 编码消息
  static encode(message: SyncMessage): Uint8Array {
    const encoder = encoding.createEncoder();
    encoding.writeVarUint(encoder, message.type);

    switch (message.type) {
      case MessageType.SYNC_STEP1:
        encoding.writeVarUint8Array(encoder, message.stateVector);
        break;
      case MessageType.SYNC_STEP2:
      case MessageType.UPDATE:
        encoding.writeVarUint8Array(encoder, message.update);
        break;
      case MessageType.AWARENESS:
        encoding.writeVarUint8Array(encoder, message.awarenessUpdate);
        break;
    }

    return encoding.toUint8Array(encoder);
  }

  // 解码消息
  static decode(data: Uint8Array): SyncMessage {
    const decoder = decoding.createDecoder(data);
    const type = decoding.readVarUint(decoder) as MessageType;

    switch (type) {
      case MessageType.SYNC_STEP1:
        return {
          type,
          stateVector: decoding.readVarUint8Array(decoder)
        };
      case MessageType.SYNC_STEP2:
      case MessageType.UPDATE:
        return {
          type,
          update: decoding.readVarUint8Array(decoder)
        };
      case MessageType.AWARENESS:
        return {
          type,
          awarenessUpdate: decoding.readVarUint8Array(decoder)
        };
      default:
        throw new Error(`Unknown message type: ${type}`);
    }
  }
}
```

---

## 3. 实时同步机制设计

### 3.1 WebSocket网关设计

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      WebSocket网关架构                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     WebSocket网关层                              │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                    接入层 (Ingress)                      │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │   │
│  │  │  │ 负载均衡器   │  │ SSL终端     │  │ 连接管理器   │     │   │   │
│  │  │  │ (Nginx/HA)  │  │ (TLS 1.3)   │  │ (连接池)    │     │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                    网关核心 (Core)                       │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │   │
│  │  │  │ 消息路由器   │  │ 会话管理器   │  │ 认证授权     │     │   │   │
│  │  │  │ Router      │  │ Session     │  │ Auth        │     │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │   │
│  │  │  │ 协议处理器   │  │ 限流器      │  │ 监控统计     │     │   │   │
│  │  │  │ Protocol    │  │ Rate Limiter│  │ Metrics     │     │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                    后端集成 (Backend)                    │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │   │   │
│  │  │  │ Redis Pub/Sub│  │ Kafka Topic │  │ 协作服务     │     │   │   │
│  │  │  │ (实时广播)   │  │ (持久化)    │  │ (CRDT处理)  │     │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘     │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  网关特性:                                                               │
│  • 支持10万+并发连接                                                     │
│  • 水平扩展能力                                                          │
│  • 自动故障转移                                                          │
│  • 消息QoS保证                                                           │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 3.1.1 WebSocket网关实现

```typescript
// WebSocket网关实现
class CollaborationGateway {
  private wss: WebSocket.Server;
  private sessionManager: SessionManager;
  private messageRouter: MessageRouter;
  private redisPubSub: RedisPubSub;
  private kafkaProducer: KafkaProducer;

  constructor(config: GatewayConfig) {
    this.sessionManager = new SessionManager();
    this.messageRouter = new MessageRouter();
    this.redisPubSub = new RedisPubSub(config.redis);
    this.kafkaProducer = new KafkaProducer(config.kafka);
  }

  // 启动网关
  async start(): Promise<void> {
    this.wss = new WebSocket.Server({
      port: config.port,
      perMessageDeflate: true,  // 启用压缩
      maxPayload: 10 * 1024 * 1024,  // 10MB最大消息
    });

    this.wss.on('connection', this.handleConnection.bind(this));

    // 订阅Redis频道
    this.redisPubSub.subscribe('collaboration:*', this.handleRedisMessage.bind(this));

    console.log(`WebSocket Gateway started on port ${config.port}`);
  }

  // 处理新连接
  private async handleConnection(ws: WebSocket, req: IncomingMessage): Promise<void> {
    try {
      // 1. 认证
      const token = this.extractToken(req);
      const user = await this.authenticate(token);

      if (!user) {
        ws.close(1008, 'Authentication failed');
        return;
      }

      // 2. 创建会话
      const session = this.sessionManager.createSession({
        userId: user.id,
        userName: user.name,
        ws,
        ip: req.socket.remoteAddress
      });

      // 3. 设置消息处理器
      ws.on('message', (data) => this.handleMessage(session, data));
      ws.on('close', (code, reason) => this.handleDisconnect(session, code, reason));
      ws.on('error', (error) => this.handleError(session, error));

      // 4. 发送连接成功消息
      this.send(session, {
        type: 'CONNECTED',
        sessionId: session.id,
        timestamp: Date.now()
      });

      // 5. 启动心跳检测
      this.startHeartbeat(session);

    } catch (error) {
      console.error('Connection handling error:', error);
      ws.close(1011, 'Internal server error');
    }
  }

  // 处理消息
  private async handleMessage(session: Session, data: WebSocket.Data): Promise<void> {
    try {
      // 限流检查
      if (!this.rateLimiter.allow(session.userId)) {
        this.send(session, {
          type: 'ERROR',
          code: 'RATE_LIMITED',
          message: 'Too many requests'
        });
        return;
      }

      // 解析消息
      const message = JSON.parse(data.toString()) as ClientMessage;

      // 路由消息
      switch (message.type) {
        case 'JOIN_DOCUMENT':
          await this.handleJoinDocument(session, message);
          break;
        case 'LEAVE_DOCUMENT':
          await this.handleLeaveDocument(session, message);
          break;
        case 'OPERATION':
          await this.handleOperation(session, message);
          break;
        case 'CURSOR_UPDATE':
          await this.handleCursorUpdate(session, message);
          break;
        case 'PING':
          this.send(session, { type: 'PONG', timestamp: Date.now() });
          break;
        default:
          this.send(session, {
            type: 'ERROR',
            code: 'UNKNOWN_MESSAGE_TYPE',
            message: `Unknown message type: ${message.type}`
          });
      }

    } catch (error) {
      console.error('Message handling error:', error);
      this.send(session, {
        type: 'ERROR',
        code: 'INVALID_MESSAGE',
        message: 'Failed to process message'
      });
    }
  }

  // 处理加入文档
  private async handleJoinDocument(session: Session, message: JoinDocumentMessage): Promise<void> {
    const { documentId } = message;

    // 权限检查
    const hasPermission = await this.checkPermission(session.userId, documentId, 'read');
    if (!hasPermission) {
      this.send(session, {
        type: 'ERROR',
        code: 'PERMISSION_DENIED',
        message: 'No permission to access this document'
      });
      return;
    }

    // 加入房间
    session.documentId = documentId;
    this.sessionManager.joinRoom(documentId, session);

    // 订阅文档频道
    const channel = `collaboration:${documentId}`;
    await this.redisPubSub.subscribe(channel);

    // 获取文档初始状态
    const documentState = await this.getDocumentState(documentId);

    // 发送初始状态
    this.send(session, {
      type: 'DOCUMENT_STATE',
      documentId,
      state: documentState,
      collaborators: this.getCollaborators(documentId)
    });

    // 广播用户加入
    this.broadcastToRoom(documentId, {
      type: 'USER_JOINED',
      userId: session.userId,
      userName: session.userName,
      timestamp: Date.now()
    }, [session.id]);

    // 记录到Kafka
    this.kafkaProducer.send('document-join', {
      userId: session.userId,
      documentId,
      timestamp: Date.now()
    });
  }

  // 处理操作消息
  private async handleOperation(session: Session, message: OperationMessage): Promise<void> {
    const { documentId, operation } = message;

    // 验证操作
    if (session.documentId !== documentId) {
      this.send(session, {
        type: 'ERROR',
        code: 'INVALID_DOCUMENT',
        message: 'Not joined to this document'
      });
      return;
    }

    // 添加元数据
    const enrichedOp = {
      ...operation,
      userId: session.userId,
      timestamp: Date.now(),
      operationId: generateUUID()
    };

    // 发布到Redis
    const channel = `collaboration:${documentId}`;
    await this.redisPubSub.publish(channel, {
      type: 'OPERATION',
      operation: enrichedOp
    });

    // 持久化到Kafka
    this.kafkaProducer.send('document-operations', {
      documentId,
      operation: enrichedOp
    });

    // 发送ACK
    this.send(session, {
      type: 'OPERATION_ACK',
      operationId: enrichedOp.operationId,
      timestamp: Date.now()
    });
  }

  // 广播到房间
  private broadcastToRoom(documentId: string, message: ServerMessage, exclude: string[] = []): void {
    const sessions = this.sessionManager.getRoomSessions(documentId);

    sessions.forEach(session => {
      if (!exclude.includes(session.id)) {
        this.send(session, message);
      }
    });
  }

  // 发送消息
  private send(session: Session, message: ServerMessage): void {
    if (session.ws.readyState === WebSocket.OPEN) {
      session.ws.send(JSON.stringify(message));
    }
  }

  // 心跳检测
  private startHeartbeat(session: Session): void {
    const interval = setInterval(() => {
      if (session.ws.readyState === WebSocket.OPEN) {
        if (Date.now() - session.lastPong > 60000) {
          // 超时未响应，关闭连接
          clearInterval(interval);
          session.ws.close(1001, 'Heartbeat timeout');
        } else {
          this.send(session, { type: 'PING' });
        }
      } else {
        clearInterval(interval);
      }
    }, 30000);
  }

  // 处理断开连接
  private handleDisconnect(session: Session, code: number, reason: string): void {
    console.log(`Session ${session.id} disconnected: ${code} - ${reason}`);

    // 离开房间
    if (session.documentId) {
      this.broadcastToRoom(session.documentId, {
        type: 'USER_LEFT',
        userId: session.userId,
        timestamp: Date.now()
      }, [session.id]);

      this.sessionManager.leaveRoom(session.documentId, session);
    }

    // 销毁会话
    this.sessionManager.destroySession(session.id);
  }
}
```

### 3.2 操作广播机制

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      操作广播机制设计                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     广播流程架构                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │   客户端A                    Redis                    客户端B    │   │
│  │  ┌─────────┐              ┌─────────┐               ┌─────────┐ │   │
│  │  │ 本地操作 │              │         │               │ 接收操作 │ │   │
│  │  └────┬────┘              │         │               ▲         │ │   │
│  │       │                   │  ┌───┐  │               │         │ │   │
│  │       │  1.发布操作        │  │   │  │  2.订阅广播    │         │ │   │
│  │       ├──────────────────►│  │   │  ├───────────────┘         │ │   │
│  │       │                   └───┼───┼──┘                         │ │   │
│  │       │                       │   │                             │ │   │
│  │       │                       ▼   ▼                             │ │   │
│  │       │                    ┌─────────┐                          │ │   │
│  │       │                    │  Kafka  │  3.持久化                │ │   │
│  │       │                    │ Topic   │                          │ │   │
│  │       │                    └─────────┘                          │ │   │
│  │       │                                                         │ │   │
│  │       │  4.ACK确认                                              │ │   │
│  │       ◄─────────────────────────────────────────────────────────┘ │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  广播策略:                                                               │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │ 策略1: 直接广播 (Direct Broadcast)                                 │ │
│  │ • 适用: 光标移动、选择变化等高频低优先级操作                        │ │
│  │ • 特点: 无ACK，可能丢包，不影响一致性                               │ │
│  │ • 实现: WebSocket直接发送，不等待确认                               │ │
│  ├───────────────────────────────────────────────────────────────────┤ │
│  │ 策略2: 可靠广播 (Reliable Broadcast)                               │ │
│  │ • 适用: 元素创建、删除、属性修改等操作                              │ │
│  │ • 特点: 需要ACK，超时重传，保证到达                                 │ │
│  │ • 实现: Redis Pub/Sub + 本地队列重传                                │ │
│  ├───────────────────────────────────────────────────────────────────┤ │
│  │ 策略3: 有序广播 (Ordered Broadcast)                                │ │
│  │ • 适用: 依赖顺序的操作序列                                          │ │
│  │ • 特点: 全局有序，因果一致性保证                                    │ │
│  │ • 实现: 序列号 + 等待队列 + CRDT合并                                │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 3.2.1 广播机制实现

```typescript
// 操作广播管理器
class OperationBroadcastManager {
  private redis: Redis;
  private pendingAcks: Map<string, PendingAck>;
  private sequenceNumber: number = 0;
  private messageQueue: Map<string, QueuedMessage[]>;

  constructor(redis: Redis) {
    this.redis = redis;
    this.pendingAcks = new Map();
    this.messageQueue = new Map();
  }

  // 广播操作
  async broadcast(
    documentId: string, 
    operation: Operation, 
    reliability: ReliabilityLevel
  ): Promise<void> {
    const message: BroadcastMessage = {
      id: generateUUID(),
      documentId,
      operation,
      sequenceNumber: ++this.sequenceNumber,
      timestamp: Date.now(),
      reliability
    };

    switch (reliability) {
      case ReliabilityLevel.FIRE_AND_FORGET:
        await this.fireAndForget(documentId, message);
        break;
      case ReliabilityLevel.AT_LEAST_ONCE:
        await this.atLeastOnce(documentId, message);
        break;
      case ReliabilityLevel.EXACTLY_ONCE:
        await this.exactlyOnce(documentId, message);
        break;
    }
  }

  // 直接广播 - 不保证送达
  private async fireAndForget(documentId: string, message: BroadcastMessage): Promise<void> {
    const channel = `collab:ff:${documentId}`;
    await this.redis.publish(channel, JSON.stringify(message));
  }

  // 至少一次广播 - 保证送达但可能重复
  private async atLeastOnce(documentId: string, message: BroadcastMessage): Promise<void> {
    const channel = `collab:alo:${documentId}`;

    // 存储待确认消息
    this.pendingAcks.set(message.id, {
      message,
      attempts: 0,
      timeout: setTimeout(() => this.retry(message), 5000)
    });

    // 发布消息
    await this.redis.publish(channel, JSON.stringify(message));

    // 同时存储到Redis用于重放
    await this.redis.setex(
      `pending:${message.id}`,
      300, // 5分钟过期
      JSON.stringify(message)
    );
  }

  // 精确一次广播 - 保证不重复
  private async exactlyOnce(documentId: string, message: BroadcastMessage): Promise<void> {
    const channel = `collab:eo:${documentId}`;
    const dedupKey = `dedup:${documentId}:${message.id}`;

    // 去重检查
    const isDuplicate = await this.redis.set(dedupKey, '1', 'EX', 3600, 'NX');
    if (!isDuplicate) {
      console.log('Duplicate message detected:', message.id);
      return;
    }

    // 存储操作日志
    await this.redis.lpush(
      `ops:${documentId}`,
      JSON.stringify(message)
    );
    await this.redis.ltrim(`ops:${documentId}`, 0, 9999); // 保留最近10000条

    // 发布消息
    await this.redis.publish(channel, JSON.stringify(message));
  }

  // 处理ACK
  handleAck(messageId: string): void {
    const pending = this.pendingAcks.get(messageId);
    if (pending) {
      clearTimeout(pending.timeout);
      this.pendingAcks.delete(messageId);
      this.redis.del(`pending:${messageId}`);
    }
  }

  // 重试机制
  private retry(message: BroadcastMessage): void {
    const pending = this.pendingAcks.get(message.id);
    if (!pending) return;

    pending.attempts++;

    if (pending.attempts >= 5) {
      // 超过最大重试次数
      console.error('Max retry exceeded for message:', message.id);
      this.pendingAcks.delete(message.id);
      return;
    }

    // 重新发布
    const channel = `collab:alo:${message.documentId}`;
    this.redis.publish(channel, JSON.stringify(message));

    // 设置下一次重试
    const delay = Math.min(1000 * Math.pow(2, pending.attempts), 30000);
    pending.timeout = setTimeout(() => this.retry(message), delay);
  }

  // 有序消息处理
  async processOrderedMessage(
    documentId: string, 
    message: BroadcastMessage,
    callback: (msg: BroadcastMessage) => void
  ): Promise<void> {
    let queue = this.messageQueue.get(documentId);
    if (!queue) {
      queue = [];
      this.messageQueue.set(documentId, queue);
    }

    // 获取期望的序列号
    const expectedSeq = await this.redis.get(`seq:${documentId}`);
    const expected = expectedSeq ? parseInt(expectedSeq) : 1;

    if (message.sequenceNumber === expected) {
      // 顺序正确，立即处理
      callback(message);
      await this.redis.set(`seq:${documentId}`, expected + 1);

      // 检查队列中是否有可处理的消息
      this.processQueue(documentId, callback);
    } else if (message.sequenceNumber > expected) {
      // 消息超前，放入队列等待
      queue.push({ message, receivedAt: Date.now() });
      queue.sort((a, b) => a.message.sequenceNumber - b.message.sequenceNumber);

      // 请求缺失的消息
      this.requestMissingMessages(documentId, expected, message.sequenceNumber - 1);
    } else {
      // 消息滞后，可能是重复消息
      console.log('Out of order message:', message.sequenceNumber, 'expected:', expected);
    }
  }

  // 处理队列中的消息
  private async processQueue(
    documentId: string, 
    callback: (msg: BroadcastMessage) => void
  ): Promise<void> {
    const queue = this.messageQueue.get(documentId);
    if (!queue) return;

    const expectedSeq = await this.redis.get(`seq:${documentId}`);
    let expected = expectedSeq ? parseInt(expectedSeq) : 1;

    while (queue.length > 0 && queue[0].message.sequenceNumber === expected) {
      const item = queue.shift()!;
      callback(item.message);
      expected++;
    }

    await this.redis.set(`seq:${documentId}`, expected);
  }

  // 请求缺失的消息
  private async requestMissingMessages(
    documentId: string, 
    from: number, 
    to: number
  ): Promise<void> {
    // 从Redis获取历史操作
    const ops = await this.redis.lrange(`ops:${documentId}`, 0, -1);

    for (const opStr of ops) {
      const op = JSON.parse(opStr) as BroadcastMessage;
      if (op.sequenceNumber >= from && op.sequenceNumber <= to) {
        // 重新发布缺失的消息
        await this.redis.publish(`collab:eo:${documentId}`, opStr);
      }
    }
  }
}

// 可靠性级别
enum ReliabilityLevel {
  FIRE_AND_FORGET = 0,   // 直接广播
  AT_LEAST_ONCE = 1,     // 至少一次
  EXACTLY_ONCE = 2       // 精确一次
}
```

### 3.3 本地预测优化

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      本地预测优化架构                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     乐观更新流程                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  用户操作                                                          │   │
│  │     │                                                             │   │
│  │     ▼                                                             │   │
│  │  ┌─────────┐     ┌─────────┐     ┌─────────┐                     │   │
│  │  │ 本地CRDT │────►│ UI更新   │     │ 显示结果 │                     │   │
│  │  │ 更新    │     │         │────►│ (乐观)  │                     │   │
│  │  └─────────┘     └─────────┘     └─────────┘                     │   │
│  │     │                                              即时反馈        │   │
│  │     │                                                             │   │
│  │     ▼                                              ◄────────────  │   │
│  │  ┌─────────┐     ┌─────────┐     ┌─────────┐                     │   │
│  │  │ 发送操作 │────►│ 等待ACK │────►│ 确认成功 │                     │   │
│  │  │ 到服务端│     │         │     │ 保持状态 │                     │   │
│  │  └─────────┘     └─────────┘     └─────────┘                     │   │
│  │     │                                ▲                           │   │
│  │     │ 失败                           │                           │   │
│  │     ▼                                │                           │   │
│  │  ┌─────────┐     ┌─────────┐         │                           │   │
│  │  │ 回滚操作 │────►│ UI回滚   │─────────┘                           │   │
│  │  │ (Undo)  │     │         │                                     │   │
│  │  └─────────┘     └─────────┘                                     │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  预测策略:                                                               │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │ 策略1: 即时预测 (Immediate Prediction)                             │ │
│  │ • 用户操作立即应用到本地CRDT和UI                                   │ │
│  │ • 操作发送到服务端，等待确认                                        │ │
│  │ • 冲突时回滚并重新应用                                              │ │
│  │ • 适用: 移动、缩放、属性修改                                        │ │
│  ├───────────────────────────────────────────────────────────────────┤ │
│  │ 策略2: 延迟预测 (Deferred Prediction)                              │ │
│  │ • 用户操作先发送到服务端                                            │ │
│  │ • 收到确认后再更新本地状态                                          │ │
│  │ • 适用: 删除操作、批量操作                                          │ │
│  ├───────────────────────────────────────────────────────────────────┤ │
│  │ 策略3: 混合预测 (Hybrid Prediction)                                │ │
│  │ • 高频操作使用即时预测                                              │ │
│  │ • 关键操作使用延迟预测                                              │ │
│  │ • 智能切换策略                                                      │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 3.3.1 本地预测实现

```typescript
// 乐观更新管理器
class OptimisticUpdateManager {
  private doc: Y.Doc;
  private pendingOps: Map<string, PendingOperation>;
  private undoManager: Y.UndoManager;
  private conflictResolver: ConflictResolver;

  constructor(doc: Y.Doc) {
    this.doc = doc;
    this.pendingOps = new Map();
    this.undoManager = new Y.UndoManager(doc.getMap('elements'));
    this.conflictResolver = new ConflictResolver();
  }

  // 执行乐观更新
  async executeOptimistic<T>(
    operation: Operation,
    localUpdate: () => T,
    sendToServer: () => Promise<ServerResponse>
  ): Promise<T> {
    const opId = generateUUID();

    // 1. 开始捕获撤销堆栈
    this.undoManager.stopCapturing();

    // 2. 执行本地更新
    const result = localUpdate();

    // 3. 记录待确认操作
    this.pendingOps.set(opId, {
      id: opId,
      operation,
      undoStackDepth: this.undoManager.undoStack.length,
      timestamp: Date.now()
    });

    // 4. 发送到服务端
    try {
      const response = await sendToServer();

      if (response.success) {
        // 确认成功
        this.confirmOperation(opId);
      } else {
        // 服务端拒绝，回滚
        await this.rollbackOperation(opId);
        throw new Error(response.error);
      }

      return result;

    } catch (error) {
      // 网络错误，回滚
      await this.rollbackOperation(opId);
      throw error;
    }
  }

  // 确认操作
  private confirmOperation(opId: string): void {
    const pending = this.pendingOps.get(opId);
    if (pending) {
      // 清理撤销堆栈中已确认的操作
      this.undoManager.clear();
      this.pendingOps.delete(opId);
    }
  }

  // 回滚操作
  private async rollbackOperation(opId: string): Promise<void> {
    const pending = this.pendingOps.get(opId);
    if (!pending) return;

    // 撤销到操作前的状态
    while (this.undoManager.undoStack.length > pending.undoStackDepth) {
      this.undoManager.undo();
    }

    this.pendingOps.delete(opId);
  }

  // 处理服务端冲突
  handleServerConflict(opId: string, serverState: any): void {
    const pending = this.pendingOps.get(opId);
    if (!pending) return;

    // 回滚本地操作
    this.rollbackOperation(opId);

    // 应用服务端状态
    Y.applyUpdate(this.doc, serverState.update);

    // 尝试重新应用本地操作（如果需要）
    const resolution = this.conflictResolver.resolve(
      pending.operation,
      serverState.operation
    );

    if (resolution.shouldReapply) {
      this.reapplyOperation(resolution.transformedOp);
    }
  }

  // 重新应用操作
  private reapplyOperation(operation: Operation): void {
    // 重新执行操作
    this.doc.transact(() => {
      this.applyOperation(operation);
    });
  }

  // 批量乐观更新
  async executeBatchOptimistic(
    operations: Operation[],
    localUpdates: () => void,
    sendToServer: () => Promise<ServerResponse>
  ): Promise<void> {
    const batchId = generateUUID();

    // 1. 批量执行本地更新
    this.doc.transact(() => {
      localUpdates();
    });

    // 2. 记录批量操作
    this.pendingOps.set(batchId, {
      id: batchId,
      operations,
      isBatch: true,
      timestamp: Date.now()
    });

    // 3. 发送到服务端
    try {
      const response = await sendToServer();

      if (response.success) {
        this.confirmOperation(batchId);
      } else {
        await this.rollbackOperation(batchId);
        throw new Error(response.error);
      }
    } catch (error) {
      await this.rollbackOperation(batchId);
      throw error;
    }
  }

  // 获取待确认操作列表
  getPendingOperations(): PendingOperation[] {
    return Array.from(this.pendingOps.values());
  }

  // 清理过期操作
  cleanupExpiredOperations(maxAge: number = 30000): void {
    const now = Date.now();

    this.pendingOps.forEach((pending, opId) => {
      if (now - pending.timestamp > maxAge) {
        this.rollbackOperation(opId);
      }
    });
  }
}

// 冲突解决器
class ConflictResolver {
  resolve(localOp: Operation, serverOp: Operation): ConflictResolution {
    // 基于操作类型选择解决策略
    switch (localOp.type) {
      case 'move':
        return this.resolveMoveConflict(localOp, serverOp);
      case 'resize':
        return this.resolveResizeConflict(localOp, serverOp);
      case 'property':
        return this.resolvePropertyConflict(localOp, serverOp);
      default:
        return { shouldReapply: false };
    }
  }

  private resolveMoveConflict(localOp: Operation, serverOp: Operation): ConflictResolution {
    if (serverOp.type === 'move' && localOp.targetId === serverOp.targetId) {
      // 同一元素的移动冲突，使用服务器状态
      return { shouldReapply: false };
    }
    // 不同元素，可以重新应用
    return { shouldReapply: true, transformedOp: localOp };
  }

  private resolveResizeConflict(localOp: Operation, serverOp: Operation): ConflictResolution {
    // 类似移动冲突处理
    return { shouldReapply: false };
  }

  private resolvePropertyConflict(localOp: Operation, serverOp: Operation): ConflictResolution {
    if (localOp.propertyPath === serverOp.propertyPath) {
      // 同一属性的冲突，使用LWW策略
      if (localOp.timestamp > serverOp.timestamp) {
        return { shouldReapply: true, transformedOp: localOp };
      }
      return { shouldReapply: false };
    }
    // 不同属性，可以重新应用
    return { shouldReapply: true, transformedOp: localOp };
  }
}
```

### 3.4 断线重连恢复

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      断线重连恢复机制                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     重连状态机                                    │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │    ┌─────────┐    断线检测    ┌─────────┐   尝试重连   ┌────────┐│   │
│  │    │ CONNECTED│─────────────►│DISCONNECTED│──────────►│RECONNECT││   │
│  │    │  已连接  │               │  已断开   │            │ 重连中  ││   │
│  │    └─────────┘               └─────────┘            └────┬───┘│   │
│  │         ▲                                                 │    │   │
│  │         │                                                 │    │   │
│  │         │            重连成功                             │    │   │
│  │         └─────────────────────────────────────────────────┘    │   │
│  │                              │                                  │   │
│  │                              ▼                                  │   │
│  │    ┌─────────┐    同步完成   ┌─────────┐    恢复状态   ┌────────┐│   │
│  │    │  ACTIVE │◄─────────────│ SYNCING │◄─────────────│ RESUME ││   │
│  │    │ 协作中  │               │ 同步中   │              │ 恢复中  ││   │
│  │    └─────────┘               └─────────┘              └────────┘│   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  恢复策略:                                                               │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │ 策略1: 增量同步 (Delta Sync)                                       │ │
│  │ • 发送本地StateVector到服务端                                      │ │
│  │ • 服务端返回缺失的更新                                              │ │
│  │ • 应用更新并合并                                                    │ │
│  │ • 适用: 短时间断线 (< 30秒)                                         │ │
│  ├───────────────────────────────────────────────────────────────────┤ │
│  │ 策略2: 快照恢复 (Snapshot Recovery)                                │ │
│  │ • 获取服务端完整文档快照                                            │ │
│  │ • 重置本地CRDT状态                                                  │ │
│  │ • 重新应用本地未确认操作                                            │ │
│  │ • 适用: 长时间断线 (> 30秒)                                         │ │
│  ├───────────────────────────────────────────────────────────────────┤ │
│  │ 策略3: 操作重放 (Operation Replay)                                 │ │
│  │ • 从服务端获取缺失的操作历史                                        │ │
│  │ • 按顺序重放操作                                                    │ │
│  │ • 合并本地未确认操作                                                │ │
│  │ • 适用: 需要完整操作历史场景                                        │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 3.4.1 断线重连实现

```typescript
// 断线重连管理器
class ReconnectionManager {
  private doc: Y.Doc;
  private provider: WebsocketProvider;
  private state: ReconnectionState;
  private pendingOps: Operation[];
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 10;
  private reconnectDelay: number = 1000;

  constructor(doc: Y.Doc, provider: WebsocketProvider) {
    this.doc = doc;
    this.provider = provider;
    this.state = ReconnectionState.CONNECTED;
    this.pendingOps = [];

    this.setupEventHandlers();
  }

  private setupEventHandlers(): void {
    // 连接断开
    this.provider.on('connection-close', () => {
      this.onDisconnect();
    });

    // 连接错误
    this.provider.on('connection-error', (error) => {
      this.onConnectionError(error);
    });

    // 连接成功
    this.provider.on('status', ({ status }) => {
      if (status === 'connected') {
        this.onReconnect();
      }
    });

    // 同步完成
    this.provider.on('sync', (isSynced) => {
      if (isSynced && this.state === ReconnectionState.SYNCING) {
        this.onSyncComplete();
      }
    });
  }

  // 断开处理
  private onDisconnect(): void {
    this.state = ReconnectionState.DISCONNECTED;

    // 保存当前状态向量
    const stateVector = Y.encodeStateVector(this.doc);
    localStorage.setItem('pendingStateVector', toBase64(stateVector));

    // 保存未确认操作
    localStorage.setItem('pendingOps', JSON.stringify(this.pendingOps));

    // 开始重连
    this.attemptReconnect();
  }

  // 尝试重连
  private async attemptReconnect(): Promise<void> {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.state = ReconnectionState.FAILED;
      this.emit('reconnect-failed');
      return;
    }

    this.state = ReconnectionState.RECONNECTING;
    this.reconnectAttempts++;

    // 指数退避
    const delay = Math.min(
      this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      30000
    );

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      this.provider.connect();
    }, delay);
  }

  // 重连成功
  private async onReconnect(): Promise<void> {
    console.log('Reconnected, starting recovery...');
    this.state = ReconnectionState.SYNCING;

    // 获取断线前的状态向量
    const savedStateVector = localStorage.getItem('pendingStateVector');

    if (savedStateVector) {
      // 增量同步
      await this.performDeltaSync(fromBase64(savedStateVector));
    } else {
      // 快照恢复
      await this.performSnapshotRecovery();
    }
  }

  // 增量同步
  private async performDeltaSync(lastStateVector: Uint8Array): Promise<void> {
    // 发送状态向量请求差异更新
    this.provider.sendSyncStep1(lastStateVector);

    // 等待同步完成
    await this.waitForSync();

    // 恢复未确认操作
    await this.reapplyPendingOps();
  }

  // 快照恢复
  private async performSnapshotRecovery(): Promise<void> {
    // 获取服务端完整状态
    const serverState = await this.fetchServerState();

    // 重置本地文档
    const newDoc = new Y.Doc();
    Y.applyUpdate(newDoc, serverState);

    // 迁移观察者
    this.migrateObservers(newDoc);

    // 尝试重新应用本地操作
    await this.reapplyPendingOps();
  }

  // 同步完成
  private onSyncComplete(): void {
    this.state = ReconnectionState.ACTIVE;
    this.reconnectAttempts = 0;

    // 清理本地存储
    localStorage.removeItem('pendingStateVector');
    localStorage.removeItem('pendingOps');

    this.emit('reconnect-success');
  }

  // 重新应用未确认操作
  private async reapplyPendingOps(): Promise<void> {
    const savedOps = localStorage.getItem('pendingOps');
    if (!savedOps) return;

    const ops = JSON.parse(savedOps) as Operation[];

    for (const op of ops) {
      // 检查操作是否已被服务端应用
      const isApplied = await this.checkOperationApplied(op.id);

      if (!isApplied) {
        // 重新应用操作
        this.doc.transact(() => {
          this.applyOperation(op);
        });

        // 重新发送
        this.provider.sendOperation(op);
      }
    }

    this.pendingOps = [];
  }

  // 检查操作是否已应用
  private async checkOperationApplied(opId: string): Promise<boolean> {
    // 通过服务端API查询
    const response = await fetch(`/api/operations/${opId}/status`);
    const { applied } = await response.json();
    return applied;
  }

  // 等待同步
  private waitForSync(timeout: number = 30000): Promise<void> {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        reject(new Error('Sync timeout'));
      }, timeout);

      const handler = (isSynced: boolean) => {
        if (isSynced) {
          clearTimeout(timer);
          this.provider.off('sync', handler);
          resolve();
        }
      };

      this.provider.on('sync', handler);
    });
  }

  // 添加待确认操作
  addPendingOp(operation: Operation): void {
    this.pendingOps.push(operation);
  }

  // 确认操作
  confirmOp(opId: string): void {
    this.pendingOps = this.pendingOps.filter(op => op.id !== opId);
  }

  // 获取当前状态
  getState(): ReconnectionState {
    return this.state;
  }

  // 获取重连尝试次数
  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }
}

// 重连状态
enum ReconnectionState {
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  RECONNECTING = 'reconnecting',
  SYNCING = 'syncing',
  ACTIVE = 'active',
  FAILED = 'failed'
}
```

---

## 4. 并发控制设计

### 4.1 乐观锁实现

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      乐观锁架构设计                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     乐观锁工作流程                                │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  客户端A                    服务端                     客户端B   │   │
│  │  ┌─────────┐              ┌─────────┐               ┌─────────┐ │   │
│  │  │ 读取v=1 │─────────────►│ 版本v=1 │◄──────────────│ 读取v=1 │ │   │
│  │  └─────────┘              └─────────┘               └─────────┘ │   │
│  │       │                        │                        │       │   │
│  │       │                        │                        │       │   │
│  │       ▼                        ▼                        ▼       │   │
│  │  ┌─────────┐              ┌─────────┐               ┌─────────┐ │   │
│  │  │ 修改数据 │              │         │               │ 修改数据 │ │   │
│  │  │ v=1→v=2 │              │         │               │ v=1→v=3 │ │   │
│  │  └─────────┘              │         │               └─────────┘ │   │
│  │       │                        │                        │       │   │
│  │       │                        │                        │       │   │
│  │       ▼                        ▼                        ▼       │   │
│  │  ┌─────────┐              ┌─────────┐               ┌─────────┐ │   │
│  │  │ 提交v=2 │─────────────►│ 检查v=1 │◄──────────────│ 提交v=3 │ │   │
│  │  │ 成功    │              │ 通过    │               │ 冲突!   │ │   │
│  │  └─────────┘              └─────────┘               └─────────┘ │   │
│  │       │                        │                        │       │   │
│  │       │                        │                        │       │   │
│  │       ▼                        ▼                        ▼       │   │
│  │  ┌─────────┐              ┌─────────┐               ┌─────────┐ │   │
│  │  │ 更新成功 │              │ 版本v=2 │               │ 重试    │ │   │
│  │  │         │              │         │               │ 读取v=2 │ │   │
│  │  └─────────┘              └─────────┘               └─────────┘ │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  乐观锁特点:                                                             │
│  • 无阻塞: 读取不加锁，提高并发性能                                       │
│  • 冲突检测: 提交时检查版本，检测冲突                                     │
│  • 冲突处理: 失败时重试或回滚                                             │
│  • 适用场景: 读多写少，冲突概率低的场景                                   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 4.1.1 乐观锁实现

```typescript
// 乐观锁管理器
class OptimisticLockManager {
  private versionStore: Map<string, VersionInfo>;
  private conflictHandler: ConflictHandler;

  constructor() {
    this.versionStore = new Map();
    this.conflictHandler = new ConflictHandler();
  }

  // 获取带版本的数据
  async getWithVersion<T>(key: string): Promise<VersionedData<T>> {
    const versionInfo = this.versionStore.get(key);

    if (!versionInfo) {
      // 从存储加载
      const data = await this.loadFromStorage(key);
      const version = this.generateVersion();

      versionInfo = {
        data,
        version,
        timestamp: Date.now(),
        updateCount: 0
      };

      this.versionStore.set(key, versionInfo);
    }

    return {
      data: versionInfo.data,
      version: versionInfo.version,
      timestamp: versionInfo.timestamp
    };
  }

  // 带版本检查更新
  async updateWithVersion<T>(
    key: string,
    newData: T,
    expectedVersion: string
  ): Promise<UpdateResult<T>> {
    const versionInfo = this.versionStore.get(key);

    if (!versionInfo) {
      return {
        success: false,
        error: 'Key not found',
        conflict: false
      };
    }

    // 版本检查
    if (versionInfo.version !== expectedVersion) {
      // 版本冲突
      const conflict = this.detectConflict(versionInfo.data, newData);

      if (conflict.hasConflict) {
        // 尝试自动合并
        const merged = this.conflictHandler.attemptMerge(
          versionInfo.data,
          newData,
          conflict
        );

        if (merged.success) {
          // 合并成功
          const newVersion = this.generateVersion();
          versionInfo.data = merged.data;
          versionInfo.version = newVersion;
          versionInfo.timestamp = Date.now();
          versionInfo.updateCount++;

          await this.persistToStorage(key, versionInfo);

          return {
            success: true,
            data: merged.data,
            version: newVersion,
            merged: true
          };
        } else {
          // 合并失败，返回冲突
          return {
            success: false,
            error: 'Version conflict detected',
            conflict: true,
            currentVersion: versionInfo.version,
            currentData: versionInfo.data
          };
        }
      }
    }

    // 版本匹配，正常更新
    const newVersion = this.generateVersion();
    versionInfo.data = newData;
    versionInfo.version = newVersion;
    versionInfo.timestamp = Date.now();
    versionInfo.updateCount++;

    await this.persistToStorage(key, versionInfo);

    return {
      success: true,
      data: newData,
      version: newVersion,
      merged: false
    };
  }

  // 批量更新
  async batchUpdate<T>(
    updates: BatchUpdateRequest<T>[]
  ): Promise<BatchUpdateResult<T>> {
    const results: UpdateResult<T>[] = [];
    const failed: BatchUpdateRequest<T>[] = [];

    // 按key分组，避免同一key的并发更新
    const grouped = this.groupByKey(updates);

    for (const [key, group] of grouped) {
      // 获取当前版本
      const versioned = await this.getWithVersion<T>(key);

      // 检查所有更新的版本是否一致
      const versions = new Set(group.map(u => u.expectedVersion));

      if (versions.size > 1) {
        // 同一key有多个不同版本的更新，需要串行处理
        for (const update of group) {
          const result = await this.updateWithVersion(
            key,
            update.data,
            update.expectedVersion
          );
          results.push(result);

          if (!result.success) {
            failed.push(update);
          }
        }
      } else {
        // 版本一致，可以批量处理
        const mergedData = this.mergeUpdates(group.map(g => g.data));
        const result = await this.updateWithVersion(
          key,
          mergedData,
          Array.from(versions)[0]
        );
        results.push(result);
      }
    }

    return {
      success: failed.length === 0,
      results,
      failed
    };
  }

  // 生成版本号
  private generateVersion(): string {
    // 使用时间戳+随机数+计数器
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substr(2, 5);
    const counter = (this.updateCounter++).toString(36);
    return `${timestamp}-${random}-${counter}`;
  }

  // 检测冲突
  private detectConflict(current: any, incoming: any): ConflictInfo {
    const changes = this.compareObjects(current, incoming);

    return {
      hasConflict: changes.conflicting.length > 0,
      conflictingFields: changes.conflicting,
      nonConflictingFields: changes.nonConflicting
    };
  }

  // 比较对象差异
  private compareObjects(current: any, incoming: any): ChangeAnalysis {
    const conflicting: string[] = [];
    const nonConflicting: string[] = [];

    const currentFields = new Set(Object.keys(current));
    const incomingFields = new Set(Object.keys(incoming));

    // 检查所有字段
    for (const field of new Set([...currentFields, ...incomingFields])) {
      if (current[field] !== incoming[field]) {
        // 字段值不同，可能存在冲突
        if (this.isConflictingChange(current[field], incoming[field])) {
          conflicting.push(field);
        } else {
          nonConflicting.push(field);
        }
      }
    }

    return { conflicting, nonConflicting };
  }

  // 判断是否为冲突性修改
  private isConflictingChange(current: any, incoming: any): boolean {
    // 如果两者都是对象，递归检查
    if (typeof current === 'object' && typeof incoming === 'object') {
      const analysis = this.compareObjects(current, incoming);
      return analysis.conflicting.length > 0;
    }

    // 基本类型，值不同即为冲突
    return current !== incoming;
  }

  private updateCounter: number = 0;
}

// 版本信息
interface VersionInfo {
  data: any;
  version: string;
  timestamp: number;
  updateCount: number;
}

// 带版本的数据
interface VersionedData<T> {
  data: T;
  version: string;
  timestamp: number;
}

// 更新结果
interface UpdateResult<T> {
  success: boolean;
  data?: T;
  version?: string;
  merged?: boolean;
  error?: string;
  conflict?: boolean;
  currentVersion?: string;
  currentData?: T;
}
```

### 4.2 版本向量设计

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      版本向量架构                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     版本向量原理                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  版本向量 (Version Vector):                                       │   │
│  │  V = {A: 3, B: 2, C: 1}                                          │   │
│  │                                                                 │   │
│  │  含义:                                                            │   │
│  │  • 副本A执行了3次更新                                              │   │
│  │  • 副本B执行了2次更新                                              │   │
│  │  • 副本C执行了1次更新                                              │   │
│  │                                                                 │   │
│  │  比较规则:                                                        │   │
│  │  ┌─────────────────────────────────────────────────────────────┐ │   │
│  │  │ V1 = {A: 3, B: 2}                                           │ │   │
│  │  │ V2 = {A: 3, B: 2, C: 1}                                     │ │   │
│  │  │                                                             │ │   │
│  │  │ V2 ≥ V1 (V2包含V1的所有更新) → V2是V1的后继                  │ │   │
│  │  │ V1 ≱ V2 (V1缺少C的更新) → 并发关系，需要合并                 │ │   │
│  │  └─────────────────────────────────────────────────────────────┘ │   │
│  │                                                                 │   │
│  │  并发检测:                                                        │   │
│  │  V1 = {A: 3, B: 2}       V2 = {A: 2, B: 3}                       │   │
│  │       ↓                        ↓                                │   │
│  │  V1[A] > V2[A] 且 V1[B] < V2[B] → 并发冲突!                      │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  版本向量在CRDT中的应用:                                                  │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  客户端A (Site A)              客户端B (Site B)                  │   │
│  │  ┌─────────────────┐          ┌─────────────────┐              │   │
│  │  │ 操作1: V={A:1}  │          │ 操作1: V={B:1}  │              │   │
│  │  │ 操作2: V={A:2}  │          │ 操作2: V={B:2}  │              │   │
│  │  │ 操作3: V={A:3}  │          │ 操作3: V={B:3}  │              │   │
│  │  └────────┬────────┘          └────────┬────────┘              │   │
│  │           │                            │                       │   │
│  │           └────────────┬───────────────┘                       │   │
│  │                        ▼                                       │   │
│  │                 ┌─────────────┐                                │   │
│  │                 │  同步合并    │                                │   │
│  │                 │ V={A:3,B:3} │                                │   │
│  │                 └─────────────┘                                │   │
│  │                                                                 │   │
│  │  合并规则:                                                       │   │
│  │  V_merged[site] = max(V1[site], V2[site])                       │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 4.2.1 版本向量实现

```typescript
// 版本向量实现
class VersionVector {
  private vector: Map<string, number>;

  constructor(initial?: Record<string, number>) {
    this.vector = new Map();
    if (initial) {
      Object.entries(initial).forEach(([site, counter]) => {
        this.vector.set(site, counter);
      });
    }
  }

  // 递增指定站点的计数器
  increment(siteId: string): void {
    const current = this.vector.get(siteId) || 0;
    this.vector.set(siteId, current + 1);
  }

  // 获取站点计数器值
  get(siteId: string): number {
    return this.vector.get(siteId) || 0;
  }

  // 比较两个版本向量
  compare(other: VersionVector): VectorComparison {
    let hasGreater = false;
    let hasLess = false;

    // 获取所有站点
    const allSites = new Set([...this.vector.keys(), ...other.vector.keys()]);

    for (const site of allSites) {
      const thisValue = this.get(site);
      const otherValue = other.get(site);

      if (thisValue > otherValue) {
        hasGreater = true;
      } else if (thisValue < otherValue) {
        hasLess = true;
      }
    }

    if (hasGreater && !hasLess) {
      return VectorComparison.GREATER;  // this > other
    } else if (!hasGreater && hasLess) {
      return VectorComparison.LESS;     // this < other
    } else if (!hasGreater && !hasLess) {
      return VectorComparison.EQUAL;    // this == other
    } else {
      return VectorComparison.CONCURRENT; // 并发冲突
    }
  }

  // 合并版本向量
  merge(other: VersionVector): VersionVector {
    const merged = new VersionVector();

    // 取每个站点的最大值
    const allSites = new Set([...this.vector.keys(), ...other.vector.keys()]);

    for (const site of allSites) {
      merged.vector.set(site, Math.max(this.get(site), other.get(site)));
    }

    return merged;
  }

  // 检查是否包含另一个版本向量
  contains(other: VersionVector): boolean {
    return this.compare(other) === VectorComparison.GREATER ||
           this.compare(other) === VectorComparison.EQUAL;
  }

  // 序列化
  toJSON(): Record<string, number> {
    const result: Record<string, number> = {};
    this.vector.forEach((value, site) => {
      result[site] = value;
    });
    return result;
  }

  // 反序列化
  static fromJSON(json: Record<string, number>): VersionVector {
    return new VersionVector(json);
  }

  // 克隆
  clone(): VersionVector {
    return new VersionVector(this.toJSON());
  }

  // 转换为字符串
  toString(): string {
    const entries = Array.from(this.vector.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([site, counter]) => `${site}:${counter}`)
      .join(',');
    return `{${entries}}`;
  }
}

// 带版本向量的操作
interface VersionedOperation {
  id: string;
  type: string;
  data: any;
  versionVector: VersionVector;
  siteId: string;
  timestamp: number;
}

// 版本向量比较结果
enum VectorComparison {
  LESS = -1,      // this < other
  EQUAL = 0,      // this == other
  GREATER = 1,    // this > other
  CONCURRENT = 2  // 并发（不可比较）
}

// 版本向量管理器
class VersionVectorManager {
  private localSiteId: string;
  private localVector: VersionVector;
  private operationLog: VersionedOperation[];

  constructor(siteId: string) {
    this.localSiteId = siteId;
    this.localVector = new VersionVector();
    this.operationLog = [];
  }

  // 创建新操作
  createOperation(type: string, data: any): VersionedOperation {
    // 递增本地计数器
    this.localVector.increment(this.localSiteId);

    const operation: VersionedOperation = {
      id: generateUUID(),
      type,
      data,
      versionVector: this.localVector.clone(),
      siteId: this.localSiteId,
      timestamp: Date.now()
    };

    // 记录操作
    this.operationLog.push(operation);

    return operation;
  }

  // 接收远程操作
  receiveOperation(operation: VersionedOperation): ReceiveResult {
    // 检查是否已存在
    if (this.operationLog.some(op => op.id === operation.id)) {
      return { status: 'duplicate' };
    }

    // 比较版本向量
    const comparison = this.localVector.compare(operation.versionVector);

    switch (comparison) {
      case VectorComparison.LESS:
      case VectorComparison.EQUAL:
        // 本地落后于远程，直接应用
        this.applyOperation(operation);
        return { status: 'applied' };

      case VectorComparison.GREATER:
        // 本地领先于远程，远程操作已过时
        return { status: 'outdated' };

      case VectorComparison.CONCURRENT:
        // 并发冲突，需要合并
        return this.handleConcurrentOperation(operation);
    }
  }

  // 应用操作
  private applyOperation(operation: VersionedOperation): void {
    // 合并版本向量
    this.localVector = this.localVector.merge(operation.versionVector);

    // 记录操作
    this.operationLog.push(operation);

    // 按版本向量排序操作日志
    this.sortOperationLog();
  }

  // 处理并发操作
  private handleConcurrentOperation(operation: VersionedOperation): ReceiveResult {
    // 使用CRDT合并
    // 这里需要根据操作类型调用相应的CRDT合并逻辑

    // 合并版本向量
    this.localVector = this.localVector.merge(operation.versionVector);

    // 记录操作
    this.operationLog.push(operation);

    // 重新排序
    this.sortOperationLog();

    return {
      status: 'merged',
      conflict: true,
      resolution: 'crdt-merge'
    };
  }

  // 排序操作日志（拓扑排序）
  private sortOperationLog(): void {
    this.operationLog.sort((a, b) => {
      const comparison = a.versionVector.compare(b.versionVector);

      if (comparison === VectorComparison.LESS) return -1;
      if (comparison === VectorComparison.GREATER) return 1;

      // 并发或相等，按时间戳排序
      return a.timestamp - b.timestamp;
    });
  }

  // 获取操作历史
  getOperationHistory(since?: VersionVector): VersionedOperation[] {
    if (!since) {
      return [...this.operationLog];
    }

    // 返回since之后的操作
    return this.operationLog.filter(op => {
      const comparison = op.versionVector.compare(since);
      return comparison === VectorComparison.GREATER ||
             comparison === VectorComparison.CONCURRENT;
    });
  }

  // 获取当前版本向量
  getCurrentVector(): VersionVector {
    return this.localVector.clone();
  }
}

// 接收结果
interface ReceiveResult {
  status: 'applied' | 'duplicate' | 'outdated' | 'merged';
  conflict?: boolean;
  resolution?: string;
}
```

### 4.3 冲突检测机制

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      冲突检测机制设计                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     冲突检测层次                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  层次1: 语法冲突检测 (Syntactic Conflict Detection)              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ • 同一元素的并发删除和修改                               │   │   │
│  │  │ • 同一属性的并发修改                                     │   │   │
│  │  │ • 违反约束的修改（如负尺寸）                             │   │   │
│  │  │ • 检测方法: 版本向量比较 + 操作类型分析                   │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  层次2: 语义冲突检测 (Semantic Conflict Detection)               │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ • 元素重叠检测（建筑设计中的空间冲突）                   │   │   │
│  │  │ • 结构约束违反（承重墙删除）                             │   │   │
│  │  │ • 业务规则违反（面积超限）                               │   │   │
│  │  │ • 检测方法: 几何计算 + 规则引擎                          │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  层次3: 意图冲突检测 (Intent Conflict Detection)                 │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ • 设计意图冲突（风格不一致）                             │   │   │
│  │  │ • 功能冲突（门和墙在同一位置）                           │   │   │
│  │  │ • 美学冲突（颜色搭配问题）                               │   │   │
│  │  │ • 检测方法: AI分析 + 设计规则库                          │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  冲突检测流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  操作A ──┐                                                       │   │
│  │          ├──► 版本向量比较 ──► 并发? ──► 是 ──► 冲突检测器        │   │
│  │  操作B ──┘                       │                              │   │
│  │                                  └─► 否 ──► 正常应用              │   │
│  │                                                                 │   │
│  │  冲突检测器:                                                     │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │   │
│  │  │ 语法冲突检测 │───►│ 语义冲突检测 │───►│ 意图冲突检测 │          │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘          │   │
│  │        │                  │                  │                  │   │
│  │        ▼                  ▼                  ▼                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                    冲突解决器                            │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 4.3.1 冲突检测实现

```typescript
// 冲突检测器
class ConflictDetector {
  private syntaxDetector: SyntaxConflictDetector;
  private semanticDetector: SemanticConflictDetector;
  private intentDetector: IntentConflictDetector;

  constructor() {
    this.syntaxDetector = new SyntaxConflictDetector();
    this.semanticDetector = new SemanticConflictDetector();
    this.intentDetector = new IntentConflictDetector();
  }

  // 检测冲突
  async detectConflicts(
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): Promise<ConflictReport> {
    const conflicts: Conflict[] = [];

    // 1. 语法冲突检测
    const syntaxConflicts = this.syntaxDetector.detect(localOp, remoteOp);
    conflicts.push(...syntaxConflicts);

    // 2. 语义冲突检测
    const semanticConflicts = await this.semanticDetector.detect(
      localOp, 
      remoteOp, 
      document
    );
    conflicts.push(...semanticConflicts);

    // 3. 意图冲突检测（异步，可能较慢）
    const intentConflicts = await this.intentDetector.detect(
      localOp, 
      remoteOp, 
      document
    );
    conflicts.push(...intentConflicts);

    return {
      hasConflict: conflicts.length > 0,
      conflicts,
      severity: this.calculateSeverity(conflicts)
    };
  }

  // 计算冲突严重程度
  private calculateSeverity(conflicts: Conflict[]): ConflictSeverity {
    if (conflicts.some(c => c.level === 'critical')) {
      return ConflictSeverity.CRITICAL;
    }
    if (conflicts.some(c => c.level === 'high')) {
      return ConflictSeverity.HIGH;
    }
    if (conflicts.some(c => c.level === 'medium')) {
      return ConflictSeverity.MEDIUM;
    }
    return ConflictSeverity.LOW;
  }
}

// 语法冲突检测器
class SyntaxConflictDetector {
  detect(localOp: Operation, remoteOp: Operation): Conflict[] {
    const conflicts: Conflict[] = [];

    // 检查1: 同一元素的删除和修改
    if (this.isDeleteModifyConflict(localOp, remoteOp)) {
      conflicts.push({
        type: 'DELETE_MODIFY',
        level: 'high',
        description: 'Element deleted and modified concurrently',
        elements: [localOp.targetId],
        autoResolvable: false
      });
    }

    // 检查2: 同一属性的并发修改
    if (this.isPropertyConflict(localOp, remoteOp)) {
      conflicts.push({
        type: 'PROPERTY_CONFLICT',
        level: 'medium',
        description: 'Same property modified concurrently',
        elements: [localOp.targetId],
        properties: [localOp.propertyPath],
        autoResolvable: true
      });
    }

    // 检查3: 父子关系冲突
    if (this.isHierarchyConflict(localOp, remoteOp)) {
      conflicts.push({
        type: 'HIERARCHY_CONFLICT',
        level: 'high',
        description: 'Parent-child relationship conflict',
        elements: [localOp.targetId, remoteOp.targetId],
        autoResolvable: false
      });
    }

    return conflicts;
  }

  private isDeleteModifyConflict(a: Operation, b: Operation): boolean {
    return a.targetId === b.targetId && 
           ((a.type === 'delete' && b.type !== 'delete') ||
            (b.type === 'delete' && a.type !== 'delete'));
  }

  private isPropertyConflict(a: Operation, b: Operation): boolean {
    return a.targetId === b.targetId &&
           a.type === 'property' &&
           b.type === 'property' &&
           a.propertyPath === b.propertyPath;
  }

  private isHierarchyConflict(a: Operation, b: Operation): boolean {
    // 检查是否形成循环引用或无效的父子关系
    return a.type === 'setParent' && b.type === 'setParent' &&
           (a.targetId === b.data.parentId || b.targetId === a.data.parentId);
  }
}

// 语义冲突检测器
class SemanticConflictDetector {
  async detect(
    localOp: Operation, 
    remoteOp: Operation,
    document: Document
  ): Promise<Conflict[]> {
    const conflicts: Conflict[] = [];

    // 检查1: 几何重叠
    const overlapConflict = await this.checkGeometricOverlap(
      localOp, 
      remoteOp, 
      document
    );
    if (overlapConflict) {
      conflicts.push(overlapConflict);
    }

    // 检查2: 结构约束
    const structureConflict = await this.checkStructureConstraint(
      localOp, 
      remoteOp, 
      document
    );
    if (structureConflict) {
      conflicts.push(structureConflict);
    }

    // 检查3: 业务规则
    const businessConflict = await this.checkBusinessRules(
      localOp, 
      remoteOp, 
      document
    );
    if (businessConflict) {
      conflicts.push(businessConflict);
    }

    return conflicts;
  }

  private async checkGeometricOverlap(
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): Promise<Conflict | null> {
    // 获取操作后的元素几何信息
    const localElement = await this.getElementAfterOp(localOp, document);
    const remoteElement = await this.getElementAfterOp(remoteOp, document);

    if (!localElement || !remoteElement) return null;

    // 几何重叠检测
    const overlap = this.calculateOverlap(localElement, remoteElement);

    if (overlap.hasOverlap) {
      return {
        type: 'GEOMETRIC_OVERLAP',
        level: overlap.severity,
        description: `Elements overlap by ${overlap.area} square units`,
        elements: [localOp.targetId, remoteOp.targetId],
        overlapInfo: overlap,
        autoResolvable: false
      };
    }

    return null;
  }

  private calculateOverlap(elem1: Element, elem2: Element): OverlapInfo {
    // 使用几何库计算重叠
    const rect1 = this.getBoundingRect(elem1);
    const rect2 = this.getBoundingRect(elem2);

    const intersection = this.rectIntersection(rect1, rect2);

    if (!intersection) {
      return { hasOverlap: false };
    }

    const overlapArea = intersection.width * intersection.height;
    const minArea = Math.min(rect1.width * rect1.height, rect2.width * rect2.height);
    const overlapRatio = overlapArea / minArea;

    return {
      hasOverlap: true,
      area: overlapArea,
      ratio: overlapRatio,
      severity: overlapRatio > 0.5 ? 'critical' : overlapRatio > 0.2 ? 'high' : 'medium'
    };
  }

  private async checkStructureConstraint(
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): Promise<Conflict | null> {
    // 检查是否删除了承重墙等关键结构
    if (localOp.type === 'delete' || remoteOp.type === 'delete') {
      const targetId = localOp.type === 'delete' ? localOp.targetId : remoteOp.targetId;
      const element = document.getElement(targetId);

      if (element?.properties?.loadBearing) {
        return {
          type: 'STRUCTURE_CONSTRAINT',
          level: 'critical',
          description: 'Cannot delete load-bearing element',
          elements: [targetId],
          autoResolvable: false
        };
      }
    }

    return null;
  }

  private async checkBusinessRules(
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): Promise<Conflict | null> {
    // 检查业务规则，如房间面积限制等
    // 这里可以实现具体的业务规则检查
    return null;
  }

  private getBoundingRect(element: Element): Rectangle {
    return {
      x: element.geometry.position.x,
      y: element.geometry.position.y,
      width: element.geometry.size.width,
      height: element.geometry.size.height
    };
  }

  private rectIntersection(r1: Rectangle, r2: Rectangle): Rectangle | null {
    const x1 = Math.max(r1.x, r2.x);
    const y1 = Math.max(r1.y, r2.y);
    const x2 = Math.min(r1.x + r1.width, r2.x + r2.width);
    const y2 = Math.min(r1.y + r1.height, r2.y + r2.height);

    if (x1 >= x2 || y1 >= y2) {
      return null;
    }

    return {
      x: x1,
      y: y1,
      width: x2 - x1,
      height: y2 - y1
    };
  }
}

// 冲突定义
interface Conflict {
  type: string;
  level: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  elements: string[];
  properties?: string[];
  autoResolvable: boolean;
  overlapInfo?: OverlapInfo;
}

interface ConflictReport {
  hasConflict: boolean;
  conflicts: Conflict[];
  severity: ConflictSeverity;
}

enum ConflictSeverity {
  NONE = 0,
  LOW = 1,
  MEDIUM = 2,
  HIGH = 3,
  CRITICAL = 4
}
```

### 4.4 冲突解决策略

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      冲突解决策略设计                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     冲突解决策略矩阵                              │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  ┌────────────────┬─────────────────┬─────────────────────────┐ │   │
│  │  │ 冲突类型        │ 解决策略         │ 说明                     │ │   │
│  │  ├────────────────┼─────────────────┼─────────────────────────┤ │   │
│  │  │ 属性冲突        │ LWW (Last-Write-│ 时间戳较晚的胜出          │ │   │
│  │  │                │ Wins)           │ 或用户选择               │ │   │
│  │  ├────────────────┼─────────────────┼─────────────────────────┤ │   │
│  │  │ 几何位置冲突    │ 空间合并         │ 保留两个元素，标记重叠   │ │   │
│  │  │                │ (Spatial Merge) │ 用户手动调整             │ │   │
│  │  ├────────────────┼─────────────────┼─────────────────────────┤ │   │
│  │  │ 删除-修改冲突   │ 删除优先         │ 删除操作胜出             │ │   │
│  │  │                │ (Delete Wins)   │ 或提示用户               │ │   │
│  │  ├────────────────┼─────────────────┼─────────────────────────┤ │   │
│  │  │ 结构约束冲突    │ 拒绝操作         │ 违反约束的操作被拒绝     │ │   │
│  │  │                │ (Reject)        │ 提示用户修改             │ │   │
│  │  ├────────────────┼─────────────────┼─────────────────────────┤ │   │
│  │  │ 意图冲突        │ 人工介入         │ 需要用户决策             │ │   │
│  │  │                │ (Manual)        │ 提供冲突视图             │ │   │
│  │  └────────────────┴─────────────────┴─────────────────────────┘ │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  冲突解决流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  检测到冲突                                                      │   │
│  │       │                                                         │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 自动解决?   │────否──► 人工介入 ──► 冲突视图 ──► 用户决策    │   │
│  │  └─────────────┘                                                │   │
│  │       │是                                                       │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 应用策略    │                                                │   │
│  │  │ • LWW       │                                                │   │
│  │  │ • 合并      │                                                │   │
│  │  │ • 拒绝      │                                                │   │
│  │  └─────────────┘                                                │   │
│  │       │                                                         │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 通知各方    │                                                │   │
│  │  │ 更新状态    │                                                │   │
│  │  └─────────────┘                                                │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 4.4.1 冲突解决实现

```typescript
// 冲突解决器
class ConflictResolver {
  private strategies: Map<string, ConflictResolutionStrategy>;
  private userDecisionQueue: UserDecisionRequest[];

  constructor() {
    this.strategies = new Map();
    this.userDecisionQueue = [];

    // 注册默认策略
    this.registerStrategy('PROPERTY_CONFLICT', new LWWStrategy());
    this.registerStrategy('DELETE_MODIFY', new DeleteWinsStrategy());
    this.registerStrategy('GEOMETRIC_OVERLAP', new SpatialMergeStrategy());
    this.registerStrategy('STRUCTURE_CONSTRAINT', new RejectStrategy());
  }

  // 注册解决策略
  registerStrategy(conflictType: string, strategy: ConflictResolutionStrategy): void {
    this.strategies.set(conflictType, strategy);
  }

  // 解决冲突
  async resolve(
    conflict: Conflict,
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): Promise<ResolutionResult> {
    const strategy = this.strategies.get(conflict.type);

    if (!strategy) {
      return {
        success: false,
        requiresUserDecision: true,
        reason: 'No strategy available for this conflict type'
      };
    }

    // 检查是否可以自动解决
    if (conflict.autoResolvable) {
      return strategy.resolve(conflict, localOp, remoteOp, document);
    }

    // 需要用户决策
    const decisionRequest: UserDecisionRequest = {
      conflictId: generateUUID(),
      conflict,
      localOp,
      remoteOp,
      options: strategy.getOptions(conflict, localOp, remoteOp),
      timeout: 30000 // 30秒超时
    };

    this.userDecisionQueue.push(decisionRequest);

    return {
      success: false,
      requiresUserDecision: true,
      decisionId: decisionRequest.conflictId
    };
  }

  // 处理用户决策
  async handleUserDecision(decisionId: string, choice: UserChoice): Promise<ResolutionResult> {
    const requestIndex = this.userDecisionQueue.findIndex(r => r.conflictId === decisionId);

    if (requestIndex === -1) {
      return {
        success: false,
        error: 'Decision request not found or expired'
      };
    }

    const request = this.userDecisionQueue[requestIndex];
    this.userDecisionQueue.splice(requestIndex, 1);

    const strategy = this.strategies.get(request.conflict.type);
    if (!strategy) {
      return {
        success: false,
        error: 'Strategy not found'
      };
    }

    return strategy.applyUserChoice(request, choice);
  }

  // 获取待决策列表
  getPendingDecisions(): UserDecisionRequest[] {
    return [...this.userDecisionQueue];
  }
}

// LWW (Last-Write-Wins) 策略
class LWWStrategy implements ConflictResolutionStrategy {
  resolve(
    conflict: Conflict,
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): ResolutionResult {
    // 比较时间戳
    const localTime = localOp.timestamp;
    const remoteTime = remoteOp.timestamp;

    if (localTime > remoteTime) {
      return {
        success: true,
        winningOp: localOp,
        strategy: 'LWW',
        reason: 'Local operation has later timestamp'
      };
    } else if (remoteTime > localTime) {
      return {
        success: true,
        winningOp: remoteOp,
        strategy: 'LWW',
        reason: 'Remote operation has later timestamp'
      };
    } else {
      // 时间戳相同，使用site ID作为决胜
      const winningOp = localOp.siteId > remoteOp.siteId ? localOp : remoteOp;
      return {
        success: true,
        winningOp,
        strategy: 'LWW-tiebreaker',
        reason: 'Same timestamp, used site ID as tiebreaker'
      };
    }
  }

  getOptions(conflict: Conflict, localOp: Operation, remoteOp: Operation): ConflictOption[] {
    return [
      { id: 'local', label: 'Keep local change', description: `Value: ${localOp.data}` },
      { id: 'remote', label: 'Keep remote change', description: `Value: ${remoteOp.data}` },
      { id: 'merge', label: 'Merge values', description: 'Combine both values' }
    ];
  }

  applyUserChoice(request: UserDecisionRequest, choice: UserChoice): ResolutionResult {
    switch (choice.optionId) {
      case 'local':
        return {
          success: true,
          winningOp: request.localOp,
          strategy: 'LWW-user-choice'
        };
      case 'remote':
        return {
          success: true,
          winningOp: request.remoteOp,
          strategy: 'LWW-user-choice'
        };
      case 'merge':
        return {
          success: true,
          mergedOp: this.mergeOperations(request.localOp, request.remoteOp),
          strategy: 'LWW-merge'
        };
      default:
        return {
          success: false,
          error: 'Invalid user choice'
        };
    }
  }

  private mergeOperations(a: Operation, b: Operation): Operation {
    // 合并两个操作的数据
    return {
      ...a,
      data: { ...a.data, ...b.data },
      merged: true,
      sources: [a.id, b.id]
    };
  }
}

// 删除优先策略
class DeleteWinsStrategy implements ConflictResolutionStrategy {
  resolve(
    conflict: Conflict,
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): ResolutionResult {
    const deleteOp = localOp.type === 'delete' ? localOp : remoteOp;

    return {
      success: true,
      winningOp: deleteOp,
      strategy: 'DELETE_WINS',
      reason: 'Delete operation takes precedence'
    };
  }

  getOptions(conflict: Conflict, localOp: Operation, remoteOp: Operation): ConflictOption[] {
    const deleteOp = localOp.type === 'delete' ? localOp : remoteOp;
    const modifyOp = localOp.type === 'delete' ? remoteOp : localOp;

    return [
      { 
        id: 'delete', 
        label: 'Delete element', 
        description: 'Remove the element completely' 
      },
      { 
        id: 'keep', 
        label: 'Keep and modify', 
        description: `Preserve with changes: ${modifyOp.data}` 
      },
      { 
        id: 'duplicate', 
        label: 'Duplicate element', 
        description: 'Create a copy with modifications' 
      }
    ];
  }

  applyUserChoice(request: UserDecisionRequest, choice: UserChoice): ResolutionResult {
    // 实现用户选择的应用逻辑
    return {
      success: true,
      strategy: 'DELETE_WINS-user-choice'
    };
  }
}

// 空间合并策略
class SpatialMergeStrategy implements ConflictResolutionStrategy {
  resolve(
    conflict: Conflict,
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): ResolutionResult {
    // 保留两个元素，标记重叠区域
    const overlapMarker = this.createOverlapMarker(conflict.overlapInfo!);

    return {
      success: true,
      winningOp: null,
      additionalOps: [overlapMarker],
      strategy: 'SPATIAL_MERGE',
      reason: 'Both elements preserved with overlap marker',
      requiresManualAdjustment: true
    };
  }

  private createOverlapMarker(overlapInfo: OverlapInfo): Operation {
    return {
      id: generateUUID(),
      type: 'createMarker',
      data: {
        markerType: 'OVERLAP_WARNING',
        area: overlapInfo.area,
        severity: overlapInfo.severity
      }
    };
  }

  getOptions(conflict: Conflict, localOp: Operation, remoteOp: Operation): ConflictOption[] {
    return [
      { id: 'keep-both', label: 'Keep both elements', description: 'Mark overlap area' },
      { id: 'keep-local', label: 'Keep local only', description: 'Remove remote element' },
      { id: 'keep-remote', label: 'Keep remote only', description: 'Remove local element' },
      { id: 'auto-adjust', label: 'Auto-adjust positions', description: 'Move elements to avoid overlap' }
    ];
  }

  applyUserChoice(request: UserDecisionRequest, choice: UserChoice): ResolutionResult {
    // 实现空间调整逻辑
    return {
      success: true,
      strategy: 'SPATIAL_MERGE-user-choice'
    };
  }
}

// 拒绝策略
class RejectStrategy implements ConflictResolutionStrategy {
  resolve(
    conflict: Conflict,
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): ResolutionResult {
    // 拒绝违反约束的操作
    const violatingOp = this.findViolatingOp(conflict, localOp, remoteOp);

    return {
      success: true,
      winningOp: violatingOp === localOp ? remoteOp : localOp,
      rejectedOp: violatingOp,
      strategy: 'REJECT',
      reason: `Operation violates constraint: ${conflict.description}`
    };
  }

  private findViolatingOp(conflict: Conflict, localOp: Operation, remoteOp: Operation): Operation {
    // 根据冲突类型判断哪个操作违反了约束
    if (conflict.type === 'STRUCTURE_CONSTRAINT') {
      return localOp.type === 'delete' ? localOp : remoteOp;
    }
    return localOp;
  }

  getOptions(conflict: Conflict, localOp: Operation, remoteOp: Operation): ConflictOption[] {
    return [
      { id: 'cancel', label: 'Cancel operation', description: 'Keep original state' },
      { id: 'modify', label: 'Modify to comply', description: 'Adjust operation to meet constraints' }
    ];
  }

  applyUserChoice(request: UserDecisionRequest, choice: UserChoice): ResolutionResult {
    return {
      success: true,
      strategy: 'REJECT-user-choice'
    };
  }
}

// 策略接口
interface ConflictResolutionStrategy {
  resolve(
    conflict: Conflict,
    localOp: Operation,
    remoteOp: Operation,
    document: Document
  ): ResolutionResult;

  getOptions(conflict: Conflict, localOp: Operation, remoteOp: Operation): ConflictOption[];

  applyUserChoice(request: UserDecisionRequest, choice: UserChoice): ResolutionResult;
}

// 解决结果
interface ResolutionResult {
  success: boolean;
  winningOp?: Operation | null;
  rejectedOp?: Operation;
  mergedOp?: Operation;
  additionalOps?: Operation[];
  strategy?: string;
  reason?: string;
  requiresUserDecision?: boolean;
  decisionId?: string;
  requiresManualAdjustment?: boolean;
  error?: string;
}
```

---

## 5. 一致性设计

### 5.1 因果一致性实现

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      因果一致性架构                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     因果一致性原理                                │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  因果一致性 (Causal Consistency):                                │   │
│  │  如果操作A在操作B之前发生（A happens-before B），               │   │
│  │  那么所有节点必须先看到A，再看到B。                              │   │
│  │                                                                 │   │
│  │  Happens-Before关系:                                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  1. 同一进程: A → B (A在B之前执行)                       │   │   │
│  │  │                                                         │   │   │
│  │  │  2. 发送-接收: send(m) → receive(m)                     │   │   │
│  │  │                                                         │   │   │
│  │  │  3. 传递性: A → B 且 B → C ⟹ A → C                      │   │   │
│  │  │                                                         │   │   │
│  │  │  示例:                                                   │   │   │
│  │  │  用户A: 创建元素X ──► 修改元素X                           │   │   │
│  │  │              │                                       │   │   │
│  │  │              ▼                                       │   │   │
│  │  │  用户B: 看到元素X ──► 删除元素X                         │   │   │
│  │  │                                                         │   │   │
│  │  │  因果链: 创建X → 修改X → 看到X → 删除X                   │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  并发操作 (Concurrent Operations):                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  用户A: ──────► 修改元素X                                │   │   │
│  │  │                                                         │   │   │
│  │  │  用户B: ──────► 修改元素Y                                │   │   │
│  │  │                                                         │   │   │
│  │  │  如果A和B没有因果关系，它们是并发的                       │   │   │
│  │  │  并发操作的执行顺序可以不同，但最终状态一致               │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  因果一致性保证:                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  1. 读-写因果: 读取操作能看到之前的写入                          │   │
│  │  2. 写-读因果: 写入操作对后续读取可见                            │   │
│  │  3. 传递因果: 因果链上的操作顺序一致                             │   │
│  │  4. 并发独立: 并发操作的顺序可以不同                             │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 5.1.1 因果一致性实现

```typescript
// 因果一致性管理器
class CausalConsistencyManager {
  private vectorClock: VectorClock;
  private dependencyGraph: DependencyGraph;
  private pendingOperations: Map<string, PendingOperation>;
  private appliedOperations: Set<string>;

  constructor(siteId: string) {
    this.vectorClock = new VectorClock(siteId);
    this.dependencyGraph = new DependencyGraph();
    this.pendingOperations = new Map();
    this.appliedOperations = new Set();
  }

  // 执行本地操作
  executeLocal(operation: Operation): CausalOperation {
    // 递增向量时钟
    this.vectorClock.increment();

    // 创建带因果信息的操作
    const causalOp: CausalOperation = {
      ...operation,
      vectorClock: this.vectorClock.clone(),
      dependencies: this.getCurrentDependencies(),
      timestamp: Date.now()
    };

    // 记录到依赖图
    this.dependencyGraph.addOperation(causalOp);
    this.appliedOperations.add(causalOp.id);

    return causalOp;
  }

  // 接收远程操作
  receiveRemote(operation: CausalOperation): ReceiveResult {
    // 检查是否已应用
    if (this.appliedOperations.has(operation.id)) {
      return { status: 'duplicate' };
    }

    // 检查依赖是否满足
    const dependenciesMet = this.checkDependencies(operation.dependencies);

    if (!dependenciesMet) {
      // 依赖未满足，加入等待队列
      this.pendingOperations.set(operation.id, {
        operation,
        missingDependencies: this.getMissingDependencies(operation.dependencies)
      });

      return { 
        status: 'pending',
        reason: 'Dependencies not met'
      };
    }

    // 依赖满足，应用操作
    return this.applyOperation(operation);
  }

  // 检查依赖是否满足
  private checkDependencies(dependencies: string[]): boolean {
    return dependencies.every(depId => this.appliedOperations.has(depId));
  }

  // 获取缺失的依赖
  private getMissingDependencies(dependencies: string[]): string[] {
    return dependencies.filter(depId => !this.appliedOperations.has(depId));
  }

  // 应用操作
  private applyOperation(operation: CausalOperation): ReceiveResult {
    // 更新向量时钟
    this.vectorClock.merge(operation.vectorClock);

    // 记录操作
    this.dependencyGraph.addOperation(operation);
    this.appliedOperations.add(operation.id);

    // 从等待队列移除
    this.pendingOperations.delete(operation.id);

    // 检查等待队列中是否有可应用的操作
    this.processPendingOperations();

    return {
      status: 'applied',
      operation
    };
  }

  // 处理等待中的操作
  private processPendingOperations(): void {
    let progress = true;

    while (progress) {
      progress = false;

      for (const [id, pending] of this.pendingOperations) {
        if (this.checkDependencies(pending.operation.dependencies)) {
          this.applyOperation(pending.operation);
          progress = true;
        }
      }
    }
  }

  // 获取当前依赖
  private getCurrentDependencies(): string[] {
    // 返回最近应用的操作ID作为依赖
    return Array.from(this.appliedOperations).slice(-10);
  }

  // 获取因果历史
  getCausalHistory(operationId: string): string[] {
    return this.dependencyGraph.getAncestors(operationId);
  }

  // 检查因果关系
  isCausallyRelated(op1: string, op2: string): boolean {
    return this.dependencyGraph.hasPath(op1, op2) ||
           this.dependencyGraph.hasPath(op2, op1);
  }

  // 获取并发操作
  getConcurrentOperations(operationId: string): string[] {
    const allOps = Array.from(this.appliedOperations);
    return allOps.filter(opId => {
      if (opId === operationId) return false;
      return !this.isCausallyRelated(opId, operationId);
    });
  }
}

// 向量时钟实现
class VectorClock {
  private clock: Map<string, number>;
  private siteId: string;

  constructor(siteId: string) {
    this.siteId = siteId;
    this.clock = new Map();
    this.clock.set(siteId, 0);
  }

  // 递增本地时钟
  increment(): void {
    const current = this.clock.get(this.siteId) || 0;
    this.clock.set(this.siteId, current + 1);
  }

  // 合并其他时钟
  merge(other: VectorClock): void {
    for (const [site, time] of other.clock) {
      const current = this.clock.get(site) || 0;
      this.clock.set(site, Math.max(current, time));
    }
  }

  // 比较时钟
  compare(other: VectorClock): ClockComparison {
    let dominates = false;
    let isDominated = false;

    const allSites = new Set([...this.clock.keys(), ...other.clock.keys()]);

    for (const site of allSites) {
      const thisTime = this.clock.get(site) || 0;
      const otherTime = other.clock.get(site) || 0;

      if (thisTime > otherTime) dominates = true;
      if (otherTime > thisTime) isDominated = true;
    }

    if (dominates && !isDominated) return ClockComparison.GREATER;
    if (!dominates && isDominated) return ClockComparison.LESS;
    if (!dominates && !isDominated) return ClockComparison.EQUAL;
    return ClockComparison.CONCURRENT;
  }

  // 克隆
  clone(): VectorClock {
    const cloned = new VectorClock(this.siteId);
    for (const [site, time] of this.clock) {
      cloned.clock.set(site, time);
    }
    return cloned;
  }

  // 序列化
  toJSON(): Record<string, number> {
    const result: Record<string, number> = {};
    this.clock.forEach((time, site) => {
      result[site] = time;
    });
    return result;
  }
}

// 依赖图
class DependencyGraph {
  private nodes: Map<string, GraphNode>;

  constructor() {
    this.nodes = new Map();
  }

  addOperation(operation: CausalOperation): void {
    this.nodes.set(operation.id, {
      id: operation.id,
      dependencies: operation.dependencies,
      dependents: []
    });

    // 更新依赖关系的反向链接
    for (const depId of operation.dependencies) {
      const depNode = this.nodes.get(depId);
      if (depNode) {
        depNode.dependents.push(operation.id);
      }
    }
  }

  // 获取祖先节点（所有依赖）
  getAncestors(nodeId: string): string[] {
    const ancestors: string[] = [];
    const visited = new Set<string>();
    const queue = [nodeId];

    while (queue.length > 0) {
      const current = queue.shift()!;
      const node = this.nodes.get(current);

      if (node && !visited.has(current)) {
        visited.add(current);
        ancestors.push(current);
        queue.push(...node.dependencies);
      }
    }

    return ancestors;
  }

  // 检查是否存在路径
  hasPath(from: string, to: string): boolean {
    const ancestors = this.getAncestors(to);
    return ancestors.includes(from);
  }
}

// 时钟比较结果
enum ClockComparison {
  LESS = -1,
  EQUAL = 0,
  GREATER = 1,
  CONCURRENT = 2
}

// 带因果信息的操作
interface CausalOperation extends Operation {
  vectorClock: VectorClock;
  dependencies: string[];
}

interface GraphNode {
  id: string;
  dependencies: string[];
  dependents: string[];
}
```

### 5.2 读写一致性保证

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      读写一致性保证                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     读写一致性模型                                │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  模型1: 读己之写 (Read Your Writes)                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  客户端: 写入A=1 ──► 读取A ──► 必须返回1                 │   │   │
│  │  │                                                         │   │   │
│  │  │  实现: 本地缓存 + 写操作确认                              │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  模型2: 单调读 (Monotonic Reads)                               │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  客户端: 读取v1 ──► 读取v2 ──► v2必须≥v1                │   │   │
│  │  │                                                         │   │   │
│  │  │  实现: 会话绑定 + 版本向量跟踪                            │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  模型3: 单调写 (Monotonic Writes)                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  客户端: 写入w1 ──► 写入w2 ──► w2必须在w1之后应用       │   │   │
│  │  │                                                         │   │   │
│  │  │  实现: 操作排序 + 依赖追踪                                │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  模型4: 写后读 (Writes Follow Reads)                           │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  客户端: 读取A ──► 写入B ──► B必须看到A的更新           │   │   │
│  │  │                                                         │   │   │
│  │  │  实现: 读版本传播 + 写依赖建立                            │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  实现架构:                                                               │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │   │
│  │  │ 写操作      │───►│ 本地缓存    │───►│ 确认等待    │         │   │
│  │  │             │    │ (读己之写)  │    │ (ACK机制)   │         │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘         │   │
│  │                                                         │       │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │       │   │
│  │  │ 读操作      │───►│ 版本检查    │───►│ 会话状态    │◄─┘       │   │
│  │  │             │    │ (单调读)    │    │ 跟踪        │          │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘          │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 5.2.1 读写一致性实现

```typescript
// 读写一致性管理器
class ReadWriteConsistencyManager {
  private sessionState: SessionState;
  localWriteCache: Map<string, CachedWrite>;
  private readVersions: Map<string, VersionVector>;
  private pendingWrites: Map<string, PendingWrite>;

  constructor(sessionId: string) {
    this.sessionState = {
      sessionId,
      writeSequence: 0,
      readSequence: 0
    };
    this.localWriteCache = new Map();
    this.readVersions = new Map();
    this.pendingWrites = new Map();
  }

  // 写操作 - 读己之写保证
  async write(key: string, value: any): Promise<WriteResult> {
    const writeId = this.generateWriteId();

    // 1. 立即更新本地缓存（读己之写）
    this.localWriteCache.set(key, {
      writeId,
      value,
      timestamp: Date.now(),
      confirmed: false
    });

    // 2. 发送写请求
    const writePromise = this.sendWriteRequest({
      writeId,
      key,
      value,
      sessionId: this.sessionState.sessionId,
      sequence: ++this.sessionState.writeSequence
    });

    // 3. 等待确认
    try {
      const result = await writePromise;

      // 4. 标记为已确认
      const cached = this.localWriteCache.get(key);
      if (cached && cached.writeId === writeId) {
        cached.confirmed = true;
        cached.serverVersion = result.version;
      }

      return {
        success: true,
        writeId,
        version: result.version
      };

    } catch (error) {
      // 写失败，从缓存移除
      this.localWriteCache.delete(key);
      throw error;
    }
  }

  // 读操作 - 单调读保证
  async read(key: string): Promise<ReadResult> {
    // 1. 检查本地写缓存（读己之写）
    const cachedWrite = this.localWriteCache.get(key);
    if (cachedWrite) {
      return {
        value: cachedWrite.value,
        source: 'local-cache',
        version: cachedWrite.serverVersion || 'pending'
      };
    }

    // 2. 获取上次读取的版本（单调读）
    const lastReadVersion = this.readVersions.get(key);

    // 3. 发送读请求，携带版本信息
    const result = await this.sendReadRequest({
      key,
      sessionId: this.sessionState.sessionId,
      minVersion: lastReadVersion,
      sequence: ++this.sessionState.readSequence
    });

    // 4. 更新读取版本
    this.readVersions.set(key, result.version);

    return {
      value: result.value,
      source: 'server',
      version: result.version
    };
  }

  // 批量读 - 保证一致性
  async batchRead(keys: string[]): Promise<BatchReadResult> {
    // 收集所有需要的版本信息
    const versionRequirements = new Map<string, VersionVector>();

    for (const key of keys) {
      const version = this.readVersions.get(key);
      if (version) {
        versionRequirements.set(key, version);
      }
    }

    // 发送批量读请求
    const result = await this.sendBatchReadRequest({
      keys,
      versionRequirements,
      sessionId: this.sessionState.sessionId
    });

    // 更新读取版本
    for (const [key, item] of result.items) {
      this.readVersions.set(key, item.version);
    }

    return result;
  }

  // 同步等待写确认
  async waitForWriteConfirmation(writeId: string, timeout: number = 5000): Promise<void> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      for (const [key, cached] of this.localWriteCache) {
        if (cached.writeId === writeId && cached.confirmed) {
          return;
        }
      }

      await this.sleep(50);
    }

    throw new Error('Write confirmation timeout');
  }

  // 强制刷新 - 确保所有写操作已确认
  async flush(): Promise<void> {
    const pendingWrites = Array.from(this.localWriteCache.entries())
      .filter(([_, cached]) => !cached.confirmed);

    await Promise.all(
      pendingWrites.map(([_, cached]) => 
        this.waitForWriteConfirmation(cached.writeId)
      )
    );
  }

  private generateWriteId(): string {
    return `${this.sessionState.sessionId}-${this.sessionState.writeSequence}-${Date.now()}`;
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // 模拟网络请求
  private async sendWriteRequest(request: WriteRequest): Promise<WriteResponse> {
    // 实际实现中这里会调用API
    return { version: new VersionVector(this.sessionState.sessionId) };
  }

  private async sendReadRequest(request: ReadRequest): Promise<ReadResponse> {
    // 实际实现中这里会调用API
    return { value: null, version: new VersionVector(this.sessionState.sessionId) };
  }

  private async sendBatchReadRequest(request: BatchReadRequest): Promise<BatchReadResult> {
    // 实际实现中这里会调用API
    return { items: new Map() };
  }
}

// 会话状态
interface SessionState {
  sessionId: string;
  writeSequence: number;
  readSequence: number;
}

// 缓存的写操作
interface CachedWrite {
  writeId: string;
  value: any;
  timestamp: number;
  confirmed: boolean;
  serverVersion?: VersionVector;
}
```

### 5.3 跨服务一致性

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      跨服务一致性设计                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Saga模式 - 分布式事务                         │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  协作服务                    文档服务                    通知服务 │   │
│  │  ┌─────────┐               ┌─────────┐               ┌─────────┐│   │
│  │  │ 开始Saga │──────────────►│ 保存文档 │──────────────►│ 发送通知 ││   │
│  │  └─────────┘               └─────────┘               └─────────┘│   │
│  │       │                         │                         │      │   │
│  │       │ 成功                    │ 成功                    │ 成功  │   │
│  │       │                         │                         │      │   │
│  │       ▼                         ▼                         ▼      │   │
│  │  ┌─────────────────────────────────────────────────────────┐    │   │
│  │  │                    Saga完成                              │    │   │
│  │  └─────────────────────────────────────────────────────────┘    │   │
│  │                                                                 │   │
│  │  补偿流程 (失败时):                                              │   │
│  │  ┌─────────┐               ┌─────────┐               ┌─────────┐│   │
│  │  │ 开始Saga │──────────────►│ 保存文档 │──X(失败)────►│ 补偿:   ││   │
│  │  └─────────┘               └─────────┘               │撤销保存 ││   │
│  │       │                         │                    └─────────┘│   │
│  │       │                         ▼                               │   │
│  │       │                    ┌─────────┐                          │   │
│  │       └───────────────────►│ Saga失败│                          │   │
│  │                            └─────────┘                          │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  最终一致性实现:                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │   │
│  │  │ 操作执行    │───►│ 事件发布    │───►│ 异步处理    │         │   │
│  │  │             │    │ (Kafka)     │    │ (消费者)    │         │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘         │   │
│  │       │                                            │            │   │
│  │       │ 同步提交                                   │ 最终一致   │   │
│  │       ▼                                            ▼            │   │
│  │  ┌─────────────┐                          ┌─────────────┐      │   │
│  │  │ 主数据存储  │                          │ 从数据存储  │      │   │
│  │  │ (强一致)    │                          │ (最终一致)  │      │   │
│  │  └─────────────┘                          └─────────────┘      │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 5.3.1 跨服务一致性实现

```typescript
// Saga编排器
class SagaOrchestrator {
  private sagas: Map<string, SagaInstance>;
  private compensations: Map<string, CompensationHandler>;

  constructor() {
    this.sagas = new Map();
    this.compensations = new Map();
  }

  // 开始Saga
  async startSaga<T>(definition: SagaDefinition<T>): Promise<SagaResult<T>> {
    const sagaId = generateUUID();
    const instance: SagaInstance = {
      id: sagaId,
      status: 'running',
      steps: [],
      currentStep: 0,
      data: definition.initialData
    };

    this.sagas.set(sagaId, instance);

    try {
      for (let i = 0; i < definition.steps.length; i++) {
        const step = definition.steps[i];
        instance.currentStep = i;

        // 执行步骤
        const result = await step.action(instance.data);

        // 记录步骤结果
        instance.steps.push({
          stepIndex: i,
          status: 'success',
          result,
          timestamp: Date.now()
        });

        // 更新Saga数据
        instance.data = { ...instance.data, ...result };
      }

      instance.status = 'completed';

      return {
        success: true,
        sagaId,
        data: instance.data
      };

    } catch (error) {
      // 执行补偿
      instance.status = 'compensating';
      await this.compensate(sagaId, instance);

      return {
        success: false,
        sagaId,
        error: error.message,
        compensated: true
      };
    }
  }

  // 执行补偿
  private async compensate(sagaId: string, instance: SagaInstance): Promise<void> {
    // 反向执行补偿
    for (let i = instance.currentStep; i >= 0; i--) {
      const step = instance.steps[i];

      if (step.compensation) {
        try {
          await step.compensation(step.result);
          step.compensated = true;
        } catch (compError) {
          // 补偿失败，记录需要人工干预
          step.compensationFailed = true;
          console.error(`Compensation failed for saga ${sagaId}, step ${i}:`, compError);
        }
      }
    }

    instance.status = 'compensated';
  }
}

// 事件溯源 - 保证跨服务一致性
class EventSourcingManager {
  private eventStore: EventStore;
  private projections: Map<string, Projection>;
  private eventBus: EventBus;

  constructor(eventStore: EventStore, eventBus: EventBus) {
    this.eventStore = eventStore;
    this.projections = new Map();
    this.eventBus = eventBus;
  }

  // 发布事件
  async publishEvent(event: DomainEvent): Promise<void> {
    // 1. 持久化事件
    await this.eventStore.append(event);

    // 2. 发布到事件总线
    await this.eventBus.publish(event);

    // 3. 更新投影
    await this.updateProjections(event);
  }

  // 更新投影
  private async updateProjections(event: DomainEvent): Promise<void> {
    for (const [name, projection] of this.projections) {
      if (projection.handles(event.type)) {
        await projection.apply(event);
      }
    }
  }

  // 注册投影
  registerProjection(name: string, projection: Projection): void {
    this.projections.set(name, projection);
  }

  // 重放事件 - 用于恢复状态
  async replayEvents(aggregateId: string, fromVersion?: number): Promise<any> {
    const events = await this.eventStore.getEvents(aggregateId, fromVersion);

    let state = {};
    for (const event of events) {
      state = this.applyEvent(state, event);
    }

    return state;
  }

  private applyEvent(state: any, event: DomainEvent): any {
    // 根据事件类型应用状态变更
    switch (event.type) {
      case 'ElementCreated':
        return { ...state, [event.data.id]: event.data };
      case 'ElementUpdated':
        return { 
          ...state, 
          [event.data.id]: { ...state[event.data.id], ...event.data.updates } 
        };
      case 'ElementDeleted':
        const { [event.data.id]: _, ...rest } = state;
        return rest;
      default:
        return state;
    }
  }
}

// 领域事件
interface DomainEvent {
  id: string;
  type: string;
  aggregateId: string;
  version: number;
  data: any;
  timestamp: number;
  metadata: {
    userId: string;
    sessionId: string;
    correlationId: string;
  };
}
```

### 5.4 最终一致性场景

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      最终一致性场景                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     最终一致性适用场景                            │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  场景1: 用户在线状态                                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 用户A在线 ──► 广播到所有客户端                            │   │   │
│  │  │ 允许短暂不一致，最终所有客户端看到正确状态                 │   │   │
│  │  │ 一致性要求: 低 (几秒延迟可接受)                           │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  场景2: 操作历史记录                                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 操作执行 ──► 异步写入历史数据库                            │   │   │
│  │  │ 历史记录允许延迟写入，不影响实时协作                       │   │   │
│  │  │ 一致性要求: 中 (分钟级延迟可接受)                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  场景3: 统计信息更新                                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 元素数量统计 ──► 异步聚合计算                              │   │   │
│  │  │ 统计数据不需要实时精确                                     │   │   │
│  │  │ 一致性要求: 低 (小时级延迟可接受)                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  场景4: 全文搜索索引                                              │   │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 文档变更 ──► 异步更新搜索索引                             │   │   │
│  │  │ 搜索索引允许短暂不一致                                     │   │   │
│  │  │ 一致性要求: 中 (秒级延迟可接受)                           │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  最终一致性实现策略:                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │   │
│  │  │ 主写从读    │    │ 异步复制    │    │ 冲突解决    │         │   │
│  │  │             │    │             │    │             │         │   │
│  │  │ 写入主节点  │    │ 主到从复制  │    │ 版本向量    │         │   │
│  │  │ 读取从节点  │    │ 延迟容忍    │    │ LWW策略     │         │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘         │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. 性能优化设计

### 6.1 批量操作优化

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      批量操作优化设计                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     批量操作策略                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  策略1: 操作合并 (Operation Merging)                             │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 原始操作:                                                │   │   │
│  │  │  1. 移动(x=10,y=10) ──► 2. 移动(x=20,y=20)              │   │   │
│  │  │                                                         │   │   │
│  │  │ 合并后:                                                 │   │   │
│  │  │  1. 移动(x=20,y=20)  (跳过中间状态)                     │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略2: 操作去重 (Operation Deduplication)                       │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 原始操作:                                                │   │   │
│  │  │  1. 设置颜色=红色 ──► 2. 设置颜色=蓝色 ──► 3. 设置颜色=红色│   │   │
│  │  │                                                         │   │   │
│  │  │ 去重后:                                                 │   │   │
│  │  │  1. 设置颜色=红色  (最终结果)                           │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略3: 批量传输 (Batch Transmission)                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 单条发送: 100个操作 × 1KB = 100KB + 100次网络往返       │   │   │
│  │  │                                                         │   │   │
│  │  │ 批量发送: 100个操作打包 = 100KB + 1次网络往返           │   │   │
│  │  │ 节省: 99次网络往返，减少延迟                            │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  批量操作实现:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  客户端                    批量队列                  服务端      │   │
│  │  ┌─────────┐              ┌─────────┐              ┌─────────┐  │   │
│  │  │ 操作1   │─────────────►│         │              │         │  │   │
│  │  │ 操作2   │─────────────►│  收集   │──定时/数量──►│ 批量处理 │  │   │
│  │  │ 操作3   │─────────────►│  窗口   │              │         │  │   │
│  │  └─────────┘              └─────────┘              └─────────┘  │   │
│  │       │                       │                        │        │   │
│  │       │                       │ 合并去重               │        │   │
│  │       │                       ▼                        ▼        │   │
│  │       │                  ┌─────────┐              ┌─────────┐   │   │
│  │       │                  │ 优化后  │              │ 批量ACK │   │   │
│  │       │                  │ 操作集  │              │         │   │   │
│  │       │                  └─────────┘              └─────────┘   │   │
│  │       │                                                         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 6.1.1 批量操作实现

```typescript
// 批量操作管理器
class BatchOperationManager {
  private operationBuffer: Operation[];
  private batchConfig: BatchConfig;
  private flushTimer: NodeJS.Timeout | null;
  private pendingFlush: Promise<void> | null;

  constructor(config: BatchConfig) {
    this.operationBuffer = [];
    this.batchConfig = config;
    this.flushTimer = null;
    this.pendingFlush = null;
  }

  // 添加操作到批量队列
  addOperation(operation: Operation): void {
    // 尝试合并操作
    const merged = this.tryMerge(operation);

    if (!merged) {
      this.operationBuffer.push(operation);
    }

    // 检查是否需要立即刷新
    if (this.shouldFlush()) {
      this.flush();
    } else {
      this.scheduleFlush();
    }
  }

  // 尝试合并操作
  private tryMerge(newOp: Operation): boolean {
    for (let i = this.operationBuffer.length - 1; i >= 0; i--) {
      const existingOp = this.operationBuffer[i];

      // 检查是否可以合并
      if (this.canMerge(existingOp, newOp)) {
        const merged = this.mergeOperations(existingOp, newOp);

        if (merged) {
          // 替换或删除原操作
          if (merged.skip) {
            this.operationBuffer.splice(i, 1);
          } else {
            this.operationBuffer[i] = merged.operation;
          }
          return true;
        }
      }
    }

    return false;
  }

  // 检查操作是否可以合并
  private canMerge(op1: Operation, op2: Operation): boolean {
    // 同一目标元素
    if (op1.targetId !== op2.targetId) return false;

    // 同一操作类型
    if (op1.type !== op2.type) return false;

    // 检查具体操作类型的合并规则
    switch (op1.type) {
      case 'move':
        return true; // 移动操作可以合并
      case 'resize':
        return true; // 缩放操作可以合并
      case 'property':
        return op1.propertyPath === op2.propertyPath; // 同一属性可以合并
      default:
        return false;
    }
  }

  // 合并两个操作
  private mergeOperations(op1: Operation, op2: Operation): MergeResult | null {
    switch (op1.type) {
      case 'move':
        // 保留最终位置
        return {
          operation: {
            ...op2,
            mergedFrom: [op1.id, op2.id]
          }
        };

      case 'property':
        // 如果最终值与初始值相同，可以跳过
        if (op1.oldValue !== undefined && op2.newValue === op1.oldValue) {
          return { skip: true };
        }

        // 合并属性修改
        return {
          operation: {
            ...op2,
            oldValue: op1.oldValue, // 保留原始值
            mergedFrom: [op1.id, op2.id]
          }
        };

      default:
        return null;
    }
  }

  // 检查是否应该立即刷新
  private shouldFlush(): boolean {
    return (
      this.operationBuffer.length >= this.batchConfig.maxBatchSize ||
      this.getBufferSize() >= this.batchConfig.maxBatchBytes
    );
  }

  // 获取缓冲区大小（字节）
  private getBufferSize(): number {
    return this.operationBuffer.reduce((size, op) => {
      return size + JSON.stringify(op).length;
    }, 0);
  }

  // 调度刷新
  private scheduleFlush(): void {
    if (this.flushTimer) return;

    this.flushTimer = setTimeout(() => {
      this.flush();
    }, this.batchConfig.flushInterval);
  }

  // 刷新缓冲区
  async flush(): Promise<void> {
    // 清除定时器
    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
      this.flushTimer = null;
    }

    // 等待之前的刷新完成
    if (this.pendingFlush) {
      await this.pendingFlush;
    }

    // 执行刷新
    this.pendingFlush = this.doFlush();
    await this.pendingFlush;
    this.pendingFlush = null;
  }

  // 实际刷新操作
  private async doFlush(): Promise<void> {
    if (this.operationBuffer.length === 0) return;

    // 复制并清空缓冲区
    const batch = [...this.operationBuffer];
    this.operationBuffer = [];

    // 批量发送
    try {
      await this.sendBatch(batch);
    } catch (error) {
      // 发送失败，重新加入缓冲区
      this.operationBuffer.unshift(...batch);
      throw error;
    }
  }

  // 发送批量操作
  private async sendBatch(operations: Operation[]): Promise<void> {
    const batchMessage: BatchMessage = {
      type: 'BATCH_OPERATION',
      operations,
      count: operations.length,
      timestamp: Date.now()
    };

    // 发送批量消息
    await this.transport.send(batchMessage);
  }

  // 立即刷新（用于关键操作）
  async flushImmediately(): Promise<void> {
    await this.flush();
  }

  // 销毁
  destroy(): void {
    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
    }
    this.flush();
  }
}

// 批量配置
interface BatchConfig {
  maxBatchSize: number;      // 最大批量大小
  maxBatchBytes: number;     // 最大批量字节数
  flushInterval: number;     // 刷新间隔（毫秒）
}

// 合并结果
interface MergeResult {
  operation?: Operation;
  skip?: boolean;
}
```

### 6.2 增量更新优化

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      增量更新优化设计                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     增量更新原理                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  全量更新 vs 增量更新:                                           │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 全量更新: 1000个元素 × 1KB = 1MB                        │   │   │
│  │  │                                                         │   │   │
│  │  │ 增量更新: 只传输变更的元素                              │   │   │
│  │  │          10个变更元素 × 1KB = 10KB                      │   │   │
│  │  │          节省: 99% 带宽                                  │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  增量更新策略:                                                   │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │                                                         │   │   │
│  │  │  1. 差异计算 (Diff Calculation)                         │   │   │
│  │  │     • 对比前后状态，找出变化                            │   │   │
│  │  │     • 使用高效的diff算法                                 │   │   │
│  │  │                                                         │   │   │
│  │  │  2. 变更压缩 (Change Compression)                       │   │   │
│  │  │     • 只传输变更的字段                                   │   │   │
│  │  │     • 使用delta编码                                     │   │   │
│  │  │                                                         │   │   │
│  │  │  3. 分层更新 (Layered Updates)                          │   │   │
│  │  │     • 几何数据单独更新                                   │   │   │
│  │  │     • 属性数据单独更新                                   │   │   │
│  │  │     • 按需加载                                          │   │   │
│  │  │                                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  增量更新流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  旧状态 ──► 差异计算 ──► 变更集 ──► 压缩编码 ──► 传输           │   │
│  │     │                                              │            │   │
│  │     │                                              ▼            │   │
│  │     │                                         网络传输          │   │
│  │     │                                              │            │   │
│  │     │                                              ▼            │   │
│  │     └───────────── 解码应用 ◄──────── 解压解码 ◄────────        │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 6.2.1 增量更新实现

```typescript
// 增量更新管理器
class IncrementalUpdateManager {
  private stateCache: Map<string, any>;
  private diffEngine: DiffEngine;
  private compressionEngine: CompressionEngine;

  constructor() {
    this.stateCache = new Map();
    this.diffEngine = new DiffEngine();
    this.compressionEngine = new CompressionEngine();
  }

  // 计算增量更新
  computeIncrementalUpdate(key: string, newState: any): IncrementalUpdate {
    // 获取缓存的旧状态
    const oldState = this.stateCache.get(key);

    if (!oldState) {
      // 首次更新，返回全量
      const fullUpdate: IncrementalUpdate = {
        type: 'full',
        data: newState,
        timestamp: Date.now()
      };

      this.stateCache.set(key, this.cloneState(newState));
      return fullUpdate;
    }

    // 计算差异
    const diff = this.diffEngine.computeDiff(oldState, newState);

    // 压缩差异
    const compressed = this.compressionEngine.compress(diff);

    // 更新缓存
    this.stateCache.set(key, this.cloneState(newState));

    return {
      type: 'incremental',
      diff: compressed,
      baseVersion: this.getStateVersion(oldState),
      timestamp: Date.now()
    };
  }

  // 应用增量更新
  applyIncrementalUpdate(currentState: any, update: IncrementalUpdate): any {
    if (update.type === 'full') {
      return update.data;
    }

    // 解压差异
    const diff = this.compressionEngine.decompress(update.diff);

    // 应用差异
    return this.diffEngine.applyDiff(currentState, diff);
  }

  // 克隆状态
  private cloneState(state: any): any {
    return JSON.parse(JSON.stringify(state));
  }

  // 获取状态版本
  private getStateVersion(state: any): string {
    // 使用哈希作为版本标识
    return this.hashState(state);
  }

  private hashState(state: any): string {
    // 简单的哈希实现
    const str = JSON.stringify(state);
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return hash.toString(16);
  }
}

// 差异引擎
class DiffEngine {
  // 计算两个对象的差异
  computeDiff(oldObj: any, newObj: any): Diff {
    const changes: Change[] = [];

    this.computeDiffRecursive(oldObj, newObj, '', changes);

    return {
      changes,
      timestamp: Date.now()
    };
  }

  private computeDiffRecursive(
    oldVal: any, 
    newVal: any, 
    path: string, 
    changes: Change[]
  ): void {
    // 相同值，无变化
    if (oldVal === newVal) return;

    // 类型不同，完全替换
    if (typeof oldVal !== typeof newVal) {
      changes.push({
        type: 'replace',
        path,
        oldValue: oldVal,
        newValue: newVal
      });
      return;
    }

    // 基本类型
    if (typeof oldVal !== 'object' || oldVal === null || newVal === null) {
      changes.push({
        type: 'replace',
        path,
        oldValue: oldVal,
        newValue: newVal
      });
      return;
    }

    // 数组
    if (Array.isArray(oldVal) && Array.isArray(newVal)) {
      this.computeArrayDiff(oldVal, newVal, path, changes);
      return;
    }

    // 对象
    this.computeObjectDiff(oldVal, newVal, path, changes);
  }

  private computeObjectDiff(
    oldObj: Record<string, any>, 
    newObj: Record<string, any>, 
    basePath: string, 
    changes: Change[]
  ): void {
    const allKeys = new Set([...Object.keys(oldObj), ...Object.keys(newObj)]);

    for (const key of allKeys) {
      const path = basePath ? `${basePath}.${key}` : key;

      if (!(key in oldObj)) {
        // 新增属性
        changes.push({
          type: 'add',
          path,
          newValue: newObj[key]
        });
      } else if (!(key in newObj)) {
        // 删除属性
        changes.push({
          type: 'remove',
          path,
          oldValue: oldObj[key]
        });
      } else {
        // 递归比较
        this.computeDiffRecursive(oldObj[key], newObj[key], path, changes);
      }
    }
  }

  private computeArrayDiff(
    oldArr: any[], 
    newArr: any[], 
    path: string, 
    changes: Change[]
  ): void {
    const maxLen = Math.max(oldArr.length, newArr.length);

    for (let i = 0; i < maxLen; i++) {
      const itemPath = `${path}[${i}]`;

      if (i >= oldArr.length) {
        // 新增元素
        changes.push({
          type: 'add',
          path: itemPath,
          newValue: newArr[i]
        });
      } else if (i >= newArr.length) {
        // 删除元素
        changes.push({
          type: 'remove',
          path: itemPath,
          oldValue: oldArr[i]
        });
      } else {
        // 递归比较
        this.computeDiffRecursive(oldArr[i], newArr[i], itemPath, changes);
      }
    }
  }

  // 应用差异
  applyDiff(obj: any, diff: Diff): any {
    const result = this.clone(obj);

    for (const change of diff.changes) {
      this.applyChange(result, change);
    }

    return result;
  }

  private applyChange(obj: any, change: Change): void {
    const pathParts = change.path.split('.');
    let current = obj;

    // 遍历到父对象
    for (let i = 0; i < pathParts.length - 1; i++) {
      const part = pathParts[i];
      const arrayMatch = part.match(/^(\w+)\[(\d+)\]$/);

      if (arrayMatch) {
        const key = arrayMatch[1];
        const index = parseInt(arrayMatch[2]);
        current = current[key][index];
      } else {
        current = current[part];
      }
    }

    // 应用变更
    const lastPart = pathParts[pathParts.length - 1];
    const arrayMatch = lastPart.match(/^(\w+)\[(\d+)\]$/);

    if (arrayMatch) {
      const key = arrayMatch[1];
      const index = parseInt(arrayMatch[2]);

      if (change.type === 'remove') {
        current[key].splice(index, 1);
      } else {
        current[key][index] = change.newValue;
      }
    } else {
      if (change.type === 'add' || change.type === 'replace') {
        current[lastPart] = change.newValue;
      } else if (change.type === 'remove') {
        delete current[lastPart];
      }
    }
  }

  private clone(obj: any): any {
    return JSON.parse(JSON.stringify(obj));
  }
}

// 差异定义
interface Diff {
  changes: Change[];
  timestamp: number;
}

interface Change {
  type: 'add' | 'remove' | 'replace';
  path: string;
  oldValue?: any;
  newValue?: any;
}

interface IncrementalUpdate {
  type: 'full' | 'incremental';
  data?: any;
  diff?: any;
  baseVersion?: string;
  timestamp: number;
}
```

### 6.3 内存优化

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      内存优化设计                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     内存优化策略                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  策略1: 对象池 (Object Pooling)                                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 频繁创建/销毁的对象使用对象池复用                          │   │   │
│  │  │ • 几何对象                                               │   │   │
│  │  │ • 操作对象                                               │   │   │
│  │  │ • 消息对象                                               │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略2: 惰性加载 (Lazy Loading)                                  │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 只在需要时加载数据                                        │   │   │
│  │  │ • 视口外元素不渲染                                        │   │   │
│  │  │ • 历史记录分页加载                                        │   │   │
│  │  │ • 缩略图按需生成                                          │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略3: 数据压缩 (Data Compression)                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 内存中数据使用压缩格式存储                                │   │   │
│  │  │ • 坐标使用Float32Array                                   │   │   │
│  │  │ • 字符串使用字典压缩                                      │   │   │
│  │  │ • 二进制数据使用TypedArray                               │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略4: 垃圾回收优化 (GC Optimization)                           │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 减少GC压力                                              │   │   │
│  │  │ • 避免频繁创建临时对象                                    │   │   │
│  │  │ • 使用WeakMap/WeakSet                                    │   │   │
│  │  │ • 手动释放不再使用的引用                                  │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  内存使用监控:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │   │
│  │  │ 内存采样    │───►│ 阈值检查    │───►│ 预警/清理   │         │   │
│  │  │ (定期)      │    │             │    │             │         │   │
│  │  └─────────────┘    └─────────────┘    └─────────────┘         │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 6.3.1 内存优化实现

```typescript
// 对象池
class ObjectPool<T> {
  private pool: T[];
  private createFn: () => T;
  private resetFn: (obj: T) => void;
  private maxSize: number;

  constructor(
    createFn: () => T,
    resetFn: (obj: T) => void,
    initialSize: number = 10,
    maxSize: number = 100
  ) {
    this.createFn = createFn;
    this.resetFn = resetFn;
    this.maxSize = maxSize;
    this.pool = [];

    // 预创建对象
    for (let i = 0; i < initialSize; i++) {
      this.pool.push(createFn());
    }
  }

  // 获取对象
  acquire(): T {
    if (this.pool.length > 0) {
      return this.pool.pop()!;
    }
    return this.createFn();
  }

  // 释放对象
  release(obj: T): void {
    if (this.pool.length < this.maxSize) {
      this.resetFn(obj);
      this.pool.push(obj);
    }
  }

  // 批量释放
  releaseAll(objs: T[]): void {
    for (const obj of objs) {
      this.release(obj);
    }
  }

  // 获取池大小
  get size(): number {
    return this.pool.length;
  }
}

// 内存管理器
class MemoryManager {
  private objectPools: Map<string, ObjectPool<any>>;
  private memoryThreshold: number;
  private cleanupInterval: number;
  private intervalId: NodeJS.Timeout | null;

  constructor(threshold: number = 512 * 1024 * 1024) { // 512MB
    this.objectPools = new Map();
    this.memoryThreshold = threshold;
    this.cleanupInterval = 30000; // 30秒
  }

  // 注册对象池
  registerPool<T>(name: string, pool: ObjectPool<T>): void {
    this.objectPools.set(name, pool);
  }

  // 启动监控
  startMonitoring(): void {
    this.intervalId = setInterval(() => {
      this.checkMemory();
    }, this.cleanupInterval);
  }

  // 停止监控
  stopMonitoring(): void {
    if (this.intervalId) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }

  // 检查内存使用
  private checkMemory(): void {
    if (typeof window !== 'undefined' && (performance as any).memory) {
      const memory = (performance as any).memory;
      const usedMB = memory.usedJSHeapSize / 1024 / 1024;
      const totalMB = memory.totalJSHeapSize / 1024 / 1024;

      console.log(`Memory: ${usedMB.toFixed(2)}MB / ${totalMB.toFixed(2)}MB`);

      if (memory.usedJSHeapSize > this.memoryThreshold) {
        this.performCleanup();
      }
    }
  }

  // 执行清理
  private performCleanup(): void {
    console.log('Performing memory cleanup...');

    // 清理对象池
    for (const [name, pool] of this.objectPools) {
      const beforeSize = pool.size;
      // 保留一半对象
      while (pool.size > 5) {
        // 对象池会自动管理
      }
      console.log(`Pool ${name}: ${beforeSize} -> ${pool.size}`);
    }

    // 触发垃圾回收（如果可用）
    if (globalThis.gc) {
      globalThis.gc();
    }
  }

  // 获取内存报告
  getMemoryReport(): MemoryReport {
    const pools: Record<string, number> = {};

    for (const [name, pool] of this.objectPools) {
      pools[name] = pool.size;
    }

    return {
      pools,
      timestamp: Date.now()
    };
  }
}

// 内存报告
interface MemoryReport {
  pools: Record<string, number>;
  timestamp: number;
}

// 惰性加载管理器
class LazyLoadManager {
  private loadedKeys: Set<string>;
  private loadCallbacks: Map<string, () => Promise<any>>;

  constructor() {
    this.loadedKeys = new Set();
    this.loadCallbacks = new Map();
  }

  // 注册加载回调
  register(key: string, callback: () => Promise<any>): void {
    this.loadCallbacks.set(key, callback);
  }

  // 加载数据
  async load(key: string): Promise<any> {
    if (this.loadedKeys.has(key)) {
      return null; // 已加载
    }

    const callback = this.loadCallbacks.get(key);
    if (!callback) {
      throw new Error(`No load callback registered for key: ${key}`);
    }

    const result = await callback();
    this.loadedKeys.add(key);

    return result;
  }

  // 检查是否已加载
  isLoaded(key: string): boolean {
    return this.loadedKeys.has(key);
  }

  // 卸载数据
  unload(key: string): void {
    this.loadedKeys.delete(key);
  }
}
```

### 6.4 网络优化

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      网络优化设计                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     网络优化策略                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  策略1: WebSocket连接复用                                        │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 单一WebSocket连接处理多个文档的协作                       │   │   │
│  │  │ 减少连接建立开销                                          │   │   │
│  │  │ 连接池管理                                                │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略2: 消息压缩                                                │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 使用permessage-deflate压缩WebSocket消息                 │   │   │
│  │  │ 二进制格式替代JSON                                        │   │   │
│  │  │ Yjs使用高效的二进制编码                                   │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略3: 自适应心跳                                              │   │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 网络良好: 延长心跳间隔 (60s)                              │   │   │
│  │  │ 网络较差: 缩短心跳间隔 (10s)                              │   │   │
│  │  │ 动态调整以平衡及时性和开销                                │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略4: 智能重连                                                │   │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 指数退避重连策略                                          │   │   │
│  │  │ 网络状态检测                                              │   │   │
│  │  │ 快速恢复机制                                              │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  网络质量监控:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  指标:                                                           │   │
│  │  • 延迟 (Latency)                                               │   │
│  │  • 丢包率 (Packet Loss)                                         │   │
│  │  • 带宽 (Bandwidth)                                             │   │
│  │  • 抖动 (Jitter)                                                │   │
│  │                                                                 │   │
│  │  自适应调整:                                                     │   │
│  │  • 延迟高 → 增加批处理大小                                      │   │
│  │  • 丢包高 → 启用更可靠传输                                      │   │
│  │  • 带宽低 → 增加压缩级别                                        │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 6.4.1 网络优化实现

```typescript
// 网络优化管理器
class NetworkOptimizationManager {
  private ws: WebSocket;
  private compressionEnabled: boolean;
  private heartbeatInterval: number;
  private latencyHistory: number[];
  private lastPingTime: number;

  constructor(ws: WebSocket) {
    this.ws = ws;
    this.compressionEnabled = true;
    this.heartbeatInterval = 30000;
    this.latencyHistory = [];
    this.lastPingTime = 0;

    this.setupPingPong();
  }

  // 设置心跳检测
  private setupPingPong(): void {
    setInterval(() => {
      if (this.ws.readyState === WebSocket.OPEN) {
        this.lastPingTime = Date.now();
        this.ws.send(JSON.stringify({ type: 'ping', timestamp: this.lastPingTime }));
      }
    }, this.heartbeatInterval);

    // 监听pong响应
    this.ws.addEventListener('message', (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'pong') {
        const latency = Date.now() - data.timestamp;
        this.recordLatency(latency);
      }
    });
  }

  // 记录延迟
  private recordLatency(latency: number): void {
    this.latencyHistory.push(latency);

    // 只保留最近100个样本
    if (this.latencyHistory.length > 100) {
      this.latencyHistory.shift();
    }

    // 自适应调整
    this.adaptToNetworkConditions();
  }

  // 根据网络状况自适应调整
  private adaptToNetworkConditions(): void {
    const avgLatency = this.getAverageLatency();
    const packetLoss = this.estimatePacketLoss();

    // 调整心跳间隔
    if (avgLatency < 50 && packetLoss < 0.01) {
      // 网络良好，延长心跳
      this.heartbeatInterval = Math.min(this.heartbeatInterval * 1.1, 60000);
    } else if (avgLatency > 200 || packetLoss > 0.05) {
      // 网络较差，缩短心跳
      this.heartbeatInterval = Math.max(this.heartbeatInterval * 0.8, 10000);
    }

    // 调整压缩
    this.compressionEnabled = avgLatency > 100;
  }

  // 计算平均延迟
  private getAverageLatency(): number {
    if (this.latencyHistory.length === 0) return 0;

    const sum = this.latencyHistory.reduce((a, b) => a + b, 0);
    return sum / this.latencyHistory.length;
  }

  // 估计丢包率
  private estimatePacketLoss(): number {
    // 基于超时次数估计
    // 简化实现
    return 0;
  }

  // 发送消息（带压缩）
  send(message: any): void {
    let data: string | ArrayBuffer;

    if (this.compressionEnabled && typeof message === 'object') {
      // 使用MessagePack压缩
      data = this.compress(message);
    } else {
      data = JSON.stringify(message);
    }

    this.ws.send(data);
  }

  // 压缩消息
  private compress(message: any): ArrayBuffer {
    // 使用MessagePack或其他二进制编码
    // 这里简化实现
    const json = JSON.stringify(message);
    const encoder = new TextEncoder();
    return encoder.encode(json).buffer;
  }

  // 获取网络状态报告
  getNetworkReport(): NetworkReport {
    return {
      averageLatency: this.getAverageLatency(),
      heartbeatInterval: this.heartbeatInterval,
      compressionEnabled: this.compressionEnabled,
      timestamp: Date.now()
    };
  }
}

// 网络报告
interface NetworkReport {
  averageLatency: number;
  heartbeatInterval: number;
  compressionEnabled: boolean;
  timestamp: number;
}
```

---

## 7. 容错设计

### 7.1 故障检测

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      故障检测架构                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     故障检测层次                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  层次1: 连接层故障检测                                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ • WebSocket连接断开检测                                  │   │   │
│  │  │ • 心跳超时检测                                           │   │   │
│  │  │ • 网络状态变化监听                                       │   │   │
│  │  │ 检测方法: ping/pong, onclose事件, online/offline事件    │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  层次2: 服务层故障检测                                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ • 服务端无响应检测                                       │   │   │
│  │  │ • 错误响应码检测                                         │   │   │
│  │  │ • 超时检测                                               │   │   │
│  │  │ 检测方法: 请求超时, 错误码分析                          │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                              ▼                                  │   │
│  │  层次3: 数据层故障检测                                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ • 数据不一致检测                                         │   │   │
│  │  │ • 版本冲突检测                                           │   │   │
│  │  │ • 校验和验证                                             │   │   │
│  │  │ 检测方法: 版本向量比较, 校验和计算                      │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  故障检测流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  心跳检测 ──► 超时? ──► 是 ──► 标记故障 ──► 触发恢复            │   │
│  │     │           │                                            │   │   │
│  │     │           └─► 否 ──► 继续检测                           │   │   │
│  │     │                                                        │   │   │
│  │     ▼                                                        │   │   │
│  │  响应检测 ──► 异常? ──► 是 ──► 分析类型 ──► 选择恢复策略       │   │   │
│  │                                                         │      │   │   │
│  │                                                         ▼      │   │   │
│  │                                                    执行恢复    │   │   │
│  │                                                                 │   │   │
│  └─────────────────────────────────────────────────────────────────┘   │   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 7.1.1 故障检测实现

```typescript
// 故障检测器
class FaultDetector {
  private heartbeatInterval: number;
  private heartbeatTimeout: number;
  private lastHeartbeat: number;
  private failureThreshold: number;
  private failureCount: number;
  private isHealthy: boolean;
  private listeners: Map<string, FaultListener[]>;

  constructor(config: FaultDetectorConfig) {
    this.heartbeatInterval = config.heartbeatInterval || 30000;
    this.heartbeatTimeout = config.heartbeatTimeout || 60000;
    this.failureThreshold = config.failureThreshold || 3;
    this.lastHeartbeat = Date.now();
    this.failureCount = 0;
    this.isHealthy = true;
    this.listeners = new Map();
  }

  // 启动检测
  start(): void {
    this.scheduleHeartbeatCheck();
  }

  // 记录心跳
  recordHeartbeat(): void {
    this.lastHeartbeat = Date.now();
    this.failureCount = 0;

    if (!this.isHealthy) {
      this.isHealthy = true;
      this.emit('recovered', { timestamp: Date.now() });
    }
  }

  // 调度心跳检查
  private scheduleHeartbeatCheck(): void {
    setInterval(() => {
      this.checkHeartbeat();
    }, this.heartbeatInterval);
  }

  // 检查心跳
  private checkHeartbeat(): void {
    const elapsed = Date.now() - this.lastHeartbeat;

    if (elapsed > this.heartbeatTimeout) {
      this.failureCount++;

      if (this.failureCount >= this.failureThreshold) {
        this.reportFailure('HEARTBEAT_TIMEOUT', {
          elapsed,
          threshold: this.heartbeatTimeout,
          failureCount: this.failureCount
        });
      }
    }
  }

  // 报告故障
  private reportFailure(type: FailureType, details: any): void {
    this.isHealthy = false;

    const failure: Failure = {
      type,
      timestamp: Date.now(),
      details,
      severity: this.calculateSeverity(type)
    };

    this.emit('failure', failure);
  }

  // 检测错误响应
  detectErrorResponse(error: Error, context: any): void {
    const errorType = this.classifyError(error);

    if (errorType === 'TRANSIENT') {
      // 临时错误，增加失败计数
      this.failureCount++;

      if (this.failureCount >= this.failureThreshold) {
        this.reportFailure('TRANSIENT_ERROR', { error: error.message, context });
      }
    } else if (errorType === 'PERMANENT') {
      // 永久错误，立即报告
      this.reportFailure('PERMANENT_ERROR', { error: error.message, context });
    }
  }

  // 分类错误
  private classifyError(error: Error): ErrorType {
    // 根据错误类型分类
    if (error.message.includes('timeout') || 
        error.message.includes('network') ||
        error.message.includes('ECONNRESET')) {
      return 'TRANSIENT';
    }

    if (error.message.includes('authentication') ||
        error.message.includes('permission') ||
        error.message.includes('not found')) {
      return 'PERMANENT';
    }

    return 'UNKNOWN';
  }

  // 计算严重程度
  private calculateSeverity(type: FailureType): FailureSeverity {
    switch (type) {
      case 'HEARTBEAT_TIMEOUT':
        return 'HIGH';
      case 'TRANSIENT_ERROR':
        return 'MEDIUM';
      case 'PERMANENT_ERROR':
        return 'CRITICAL';
      default:
        return 'LOW';
    }
  }

  // 添加监听器
  on(event: string, listener: FaultListener): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(listener);
  }

  // 触发事件
  private emit(event: string, data: any): void {
    const listeners = this.listeners.get(event);
    if (listeners) {
      listeners.forEach(listener => listener(data));
    }
  }

  // 获取健康状态
  getHealthStatus(): HealthStatus {
    return {
      isHealthy: this.isHealthy,
      failureCount: this.failureCount,
      lastHeartbeat: this.lastHeartbeat,
      timestamp: Date.now()
    };
  }
}

// 故障类型
type FailureType = 'HEARTBEAT_TIMEOUT' | 'TRANSIENT_ERROR' | 'PERMANENT_ERROR';
type ErrorType = 'TRANSIENT' | 'PERMANENT' | 'UNKNOWN';
type FailureSeverity = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

// 故障定义
interface Failure {
  type: FailureType;
  timestamp: number;
  details: any;
  severity: FailureSeverity;
}

// 健康状态
interface HealthStatus {
  isHealthy: boolean;
  failureCount: number;
  lastHeartbeat: number;
  timestamp: number;
}

type FaultListener = (data: any) => void;

interface FaultDetectorConfig {
  heartbeatInterval?: number;
  heartbeatTimeout?: number;
  failureThreshold?: number;
}
```

### 7.2 故障恢复

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      故障恢复架构                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     故障恢复策略                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  策略1: 自动重连 (Auto Reconnect)                                │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 指数退避策略:                                           │   │   │
│  │  │  第1次: 1秒后重连                                       │   │   │
│  │  │  第2次: 2秒后重连                                       │   │   │
│  │  │  第3次: 4秒后重连                                       │   │   │
│  │  │  ...                                                    │   │   │
│  │  │  最大间隔: 30秒                                         │   │   │
│  │  │  最大重试: 10次                                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略2: 状态恢复 (State Recovery)                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 1. 从本地存储恢复未确认操作                             │   │   │
│  │  │ 2. 与服务端同步状态                                     │   │   │
│  │  │ 3. 重新应用本地操作                                     │   │   │
│  │  │ 4. 恢复用户界面状态                                     │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略3: 优雅降级 (Graceful Degradation)                        │   │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 协作功能不可用时:                                       │   │   │
│  │  │ • 切换到本地编辑模式                                    │   │   │
│  │  │ • 禁用实时同步                                          │   │   │
│  │  │ • 提供手动同步按钮                                      │   │   │
│  │  │ • 保存到本地草稿                                        │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  故障恢复流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  故障检测 ──► 分类故障 ──► 选择策略 ──► 执行恢复               │   │
│  │     │                                              │            │   │
│  │     │                                              ▼            │   │
│  │     │                                         恢复成功?         │   │
│  │     │                                              │            │   │
│  │     │           是 ◄───────────────────────────────┘            │   │
│  │     │           │                                              │   │
│  │     │           ▼                                              │   │
│  │     │      恢复正常 ──► 通知用户                               │   │
│  │     │                                                          │   │
│  │     └───────────┐                                              │   │
│  │                 否                                             │   │
│  │                 ▼                                              │   │
│  │            启用降级 ──► 通知用户 ──► 等待人工干预              │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 7.2.1 故障恢复实现

```typescript
// 故障恢复管理器
class RecoveryManager {
  private reconnectAttempts: number;
  private maxReconnectAttempts: number;
  private baseReconnectDelay: number;
  private maxReconnectDelay: number;
  private recoveryStrategies: Map<string, RecoveryStrategy>;
  private stateManager: StateRecoveryManager;

  constructor(config: RecoveryConfig) {
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = config.maxReconnectAttempts || 10;
    this.baseReconnectDelay = config.baseReconnectDelay || 1000;
    this.maxReconnectDelay = config.maxReconnectDelay || 30000;
    this.recoveryStrategies = new Map();
    this.stateManager = new StateRecoveryManager();

    this.registerDefaultStrategies();
  }

  // 注册默认恢复策略
  private registerDefaultStrategies(): void {
    this.registerStrategy('CONNECTION_LOST', new ConnectionRecoveryStrategy());
    this.registerStrategy('STATE_MISMATCH', new StateRecoveryStrategy());
    this.registerStrategy('DATA_CORRUPTION', new DataRecoveryStrategy());
  }

  // 注册恢复策略
  registerStrategy(failureType: string, strategy: RecoveryStrategy): void {
    this.recoveryStrategies.set(failureType, strategy);
  }

  // 执行恢复
  async recover(failure: Failure): Promise<RecoveryResult> {
    console.log(`Recovering from failure: ${failure.type}`);

    const strategy = this.recoveryStrategies.get(failure.type);

    if (!strategy) {
      return {
        success: false,
        error: `No recovery strategy for failure type: ${failure.type}`
      };
    }

    try {
      const result = await strategy.execute(failure);

      if (result.success) {
        this.reconnectAttempts = 0;
      }

      return result;
    } catch (error) {
      return {
        success: false,
        error: error.message
      };
    }
  }

  // 自动重连
  async autoReconnect(connectFn: () => Promise<void>): Promise<boolean> {
    while (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;

      // 计算退避延迟
      const delay = this.calculateReconnectDelay();
      console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

      await this.sleep(delay);

      try {
        await connectFn();
        console.log('Reconnected successfully');
        this.reconnectAttempts = 0;
        return true;
      } catch (error) {
        console.error(`Reconnect attempt ${this.reconnectAttempts} failed:`, error);
      }
    }

    console.error('Max reconnect attempts reached');
    return false;
  }

  // 计算重连延迟
  private calculateReconnectDelay(): number {
    // 指数退避
    const exponentialDelay = this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    // 添加随机抖动
    const jitter = Math.random() * 1000;

    // 限制最大延迟
    return Math.min(exponentialDelay + jitter, this.maxReconnectDelay);
  }

  // 恢复状态
  async recoverState(): Promise<StateRecoveryResult> {
    return this.stateManager.recover();
  }

  // 保存状态（用于恢复）
  async saveState(state: any): Promise<void> {
    await this.stateManager.save(state);
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

// 连接恢复策略
class ConnectionRecoveryStrategy implements RecoveryStrategy {
  async execute(failure: Failure): Promise<RecoveryResult> {
    // 实现连接恢复逻辑
    return { success: true };
  }
}

// 状态恢复策略
class StateRecoveryStrategy implements RecoveryStrategy {
  async execute(failure: Failure): Promise<RecoveryResult> {
    // 实现状态恢复逻辑
    return { success: true };
  }
}

// 数据恢复策略
class DataRecoveryStrategy implements RecoveryStrategy {
  async execute(failure: Failure): Promise<RecoveryResult> {
    // 实现数据恢复逻辑
    return { success: true };
  }
}

// 状态恢复管理器
class StateRecoveryManager {
  private storage: Storage;

  constructor() {
    this.storage = localStorage;
  }

  // 保存状态
  async save(state: any): Promise<void> {
    const recoveryData: RecoveryData = {
      state,
      timestamp: Date.now(),
      version: '1.0'
    };

    this.storage.setItem('collaboration:recovery', JSON.stringify(recoveryData));
  }

  // 恢复状态
  async recover(): Promise<StateRecoveryResult> {
    const saved = this.storage.getItem('collaboration:recovery');

    if (!saved) {
      return {
        success: false,
        error: 'No saved state found'
      };
    }

    try {
      const recoveryData: RecoveryData = JSON.parse(saved);

      // 检查数据时效性
      const age = Date.now() - recoveryData.timestamp;
      const maxAge = 24 * 60 * 60 * 1000; // 24小时

      if (age > maxAge) {
        return {
          success: false,
          error: 'Saved state is too old'
        };
      }

      return {
        success: true,
        state: recoveryData.state,
        timestamp: recoveryData.timestamp
      };

    } catch (error) {
      return {
        success: false,
        error: 'Failed to parse saved state'
      };
    }
  }

  // 清除保存的状态
  clear(): void {
    this.storage.removeItem('collaboration:recovery');
  }
}

// 恢复策略接口
interface RecoveryStrategy {
  execute(failure: Failure): Promise<RecoveryResult>;
}

// 恢复结果
interface RecoveryResult {
  success: boolean;
  error?: string;
  details?: any;
}

// 状态恢复结果
interface StateRecoveryResult extends RecoveryResult {
  state?: any;
  timestamp?: number;
}

// 恢复数据
interface RecoveryData {
  state: any;
  timestamp: number;
  version: string;
}

// 恢复配置
interface RecoveryConfig {
  maxReconnectAttempts?: number;
  baseReconnectDelay?: number;
  maxReconnectDelay?: number;
}
```

### 7.3 数据修复

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      数据修复机制                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     数据修复策略                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  策略1: 校验和验证 (Checksum Validation)                         │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 计算数据校验和，检测数据损坏                             │   │   │
│  │  │ 校验和不匹配时触发修复流程                               │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略2: 版本对比修复 (Version-based Repair)                      │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 对比多个副本的版本向量                                   │   │   │
│  │  │ 使用最新版本作为权威来源                                 │   │   │
│  │  │ 合并可合并的并发更新                                     │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略3: 操作日志重放 (Operation Log Replay)                      │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 从操作日志重建状态                                       │   │   │
│  │  │ 按顺序重放所有操作                                       │   │   │
│  │  │ 确保最终一致性                                           │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  策略4: 快照恢复 (Snapshot Recovery)                             │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 从最近的快照恢复                                         │   │   │
│  │  │ 重放快照后的操作                                         │   │   │
│  │  │ 快速恢复到一致状态                                       │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  数据修复流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  检测到数据问题                                                  │   │
│  │       │                                                         │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 诊断问题    │───► 确定修复策略                               │   │
│  │  └─────────────┘                                                │   │
│  │       │                                                         │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 执行修复    │                                                │   │
│  │  │ • 获取权威源 │                                                │   │
│  │  │ • 应用修复   │                                                │   │
│  │  │ • 验证结果   │                                                │   │
│  │  └─────────────┘                                                │   │
│  │       │                                                         │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 修复成功?   │───► 是 ──► 记录修复日志 ──► 完成              │   │
│  │  └─────────────┘                                                │   │
│  │       │                                                         │   │
│  │       否                                                        │   │
│  │       ▼                                                         │   │
│  │  ┌─────────────┐                                                │   │
│  │  │ 人工介入    │                                                │   │
│  │  └─────────────┘                                                │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 7.3.1 数据修复实现

```typescript
// 数据修复管理器
class DataRepairManager {
  private checksumValidator: ChecksumValidator;
  private versionComparator: VersionComparator;
  private operationReplayer: OperationReplayer;
  private snapshotManager: SnapshotManager;

  constructor() {
    this.checksumValidator = new ChecksumValidator();
    this.versionComparator = new VersionComparator();
    this.operationReplayer = new OperationReplayer();
    this.snapshotManager = new SnapshotManager();
  }

  // 验证数据完整性
  async validateData(data: any): Promise<ValidationResult> {
    const checksum = this.checksumValidator.calculate(data);
    const storedChecksum = await this.getStoredChecksum(data.id);

    if (checksum !== storedChecksum) {
      return {
        valid: false,
        error: 'Checksum mismatch',
        expectedChecksum: storedChecksum,
        actualChecksum: checksum
      };
    }

    return { valid: true };
  }

  // 修复数据
  async repairData(documentId: string): Promise<RepairResult> {
    console.log(`Starting data repair for document: ${documentId}`);

    // 1. 获取所有可用副本
    const replicas = await this.getReplicas(documentId);

    // 2. 选择权威副本
    const authoritative = this.selectAuthoritativeReplica(replicas);

    // 3. 执行修复
    const repairLog: RepairLogEntry[] = [];

    for (const replica of replicas) {
      if (replica.id !== authoritative.id) {
        const result = await this.repairReplica(replica, authoritative);
        repairLog.push({
          replicaId: replica.id,
          action: result.action,
          timestamp: Date.now()
        });
      }
    }

    return {
      success: true,
      authoritativeReplica: authoritative.id,
      repairLog
    };
  }

  // 选择权威副本
  private selectAuthoritativeReplica(replicas: Replica[]): Replica {
    // 按版本向量排序，选择最新的
    return replicas.sort((a, b) => {
      const comparison = this.versionComparator.compare(a.version, b.version);
      return -comparison; // 降序
    })[0];
  }

  // 修复副本
  private async repairReplica(replica: Replica, authoritative: Replica): Promise<RepairAction> {
    // 计算差异
    const diff = this.calculateDiff(replica.data, authoritative.data);

    if (diff.isEmpty) {
      return { action: 'none', reason: 'No difference' };
    }

    // 应用修复
    replica.data = this.applyDiff(replica.data, diff);
    replica.version = authoritative.version;

    await this.saveReplica(replica);

    return {
      action: 'repaired',
      changes: diff.changes.length
    };
  }

  // 从快照恢复
  async recoverFromSnapshot(documentId: string, snapshotId: string): Promise<RecoveryResult> {
    // 1. 获取快照
    const snapshot = await this.snapshotManager.getSnapshot(snapshotId);

    if (!snapshot) {
      return {
        success: false,
        error: 'Snapshot not found'
      };
    }

    // 2. 获取快照后的操作
    const operations = await this.getOperationsSince(documentId, snapshot.timestamp);

    // 3. 重建状态
    let state = snapshot.state;
    for (const op of operations) {
      state = this.operationReplayer.apply(state, op);
    }

    // 4. 保存恢复后的状态
    await this.saveDocumentState(documentId, state);

    return {
      success: true,
      state,
      operationsReplayed: operations.length
    };
  }

  // 计算差异
  private calculateDiff(source: any, target: any): Diff {
    // 使用差异引擎计算
    return new DiffEngine().computeDiff(source, target);
  }

  // 应用差异
  private applyDiff(source: any, diff: Diff): any {
    return new DiffEngine().applyDiff(source, diff);
  }

  // 获取副本
  private async getReplicas(documentId: string): Promise<Replica[]> {
    // 从存储获取所有副本
    return [];
  }

  // 保存副本
  private async saveReplica(replica: Replica): Promise<void> {
    // 保存到存储
  }

  // 获取存储的校验和
  private async getStoredChecksum(id: string): Promise<string> {
    // 从元数据获取
    return '';
  }

  // 获取快照后的操作
  private async getOperationsSince(documentId: string, since: number): Promise<Operation[]> {
    // 从操作日志获取
    return [];
  }

  // 保存文档状态
  private async saveDocumentState(documentId: string, state: any): Promise<void> {
    // 保存到存储
  }
}

// 校验和验证器
class ChecksumValidator {
  calculate(data: any): string {
    const str = JSON.stringify(data);
    return this.hashString(str);
  }

  private hashString(str: string): string {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return hash.toString(16);
  }
}

// 版本比较器
class VersionComparator {
  compare(v1: VersionVector, v2: VersionVector): number {
    return v1.compare(v2);
  }
}

// 操作重放器
class OperationReplayer {
  apply(state: any, operation: Operation): any {
    // 根据操作类型应用
    switch (operation.type) {
      case 'create':
        return { ...state, [operation.targetId]: operation.data };
      case 'update':
        return {
          ...state,
          [operation.targetId]: { ...state[operation.targetId], ...operation.data }
        };
      case 'delete':
        const { [operation.targetId]: _, ...rest } = state;
        return rest;
      default:
        return state;
    }
  }
}

// 快照管理器
class SnapshotManager {
  async getSnapshot(snapshotId: string): Promise<Snapshot | null> {
    // 从存储获取快照
    return null;
  }

  async createSnapshot(documentId: string, state: any): Promise<Snapshot> {
    const snapshot: Snapshot = {
      id: generateUUID(),
      documentId,
      state,
      timestamp: Date.now()
    };

    // 保存快照
    return snapshot;
  }
}

// 副本定义
interface Replica {
  id: string;
  documentId: string;
  data: any;
  version: VersionVector;
  lastUpdated: number;
}

// 快照定义
interface Snapshot {
  id: string;
  documentId: string;
  state: any;
  timestamp: number;
}

// 修复结果
interface RepairResult {
  success: boolean;
  authoritativeReplica: string;
  repairLog: RepairLogEntry[];
}

interface RepairLogEntry {
  replicaId: string;
  action: RepairAction;
  timestamp: number;
}

interface RepairAction {
  action: 'none' | 'repaired';
  reason?: string;
  changes?: number;
}

// 验证结果
interface ValidationResult {
  valid: boolean;
  error?: string;
  expectedChecksum?: string;
  actualChecksum?: string;
}

// 恢复结果
interface RecoveryResult {
  success: boolean;
  state?: any;
  operationsReplayed?: number;
  error?: string;
}
```

### 7.4 降级策略

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      降级策略设计                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     降级策略层次                                  │   │
│  ├─────────────────────────────────────────────────────────────────┤   │
│  │                                                                 │   │
│  │  级别1: 功能降级 (Feature Degradation)                           │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 协作功能不可用时:                                       │   │   │
│  │  │ • 禁用实时同步                                          │   │   │
│  │  │ • 禁用光标显示                                          │   │   │
│  │  │ • 禁用在线用户列表                                      │   │   │
│  │  │ • 保留本地编辑功能                                      │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  级别2: 性能降级 (Performance Degradation)                       │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 系统负载过高时:                                         │   │   │
│  │  │ • 降低同步频率                                          │   │   │
│  │  │ • 减少历史记录保存                                      │   │   │
│  │  │ • 禁用非必要功能                                        │   │   │
│  │  │ • 启用流控                                              │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  级别3: 服务降级 (Service Degradation)                           │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 部分服务不可用时:                                       │   │   │
│  │  │ • 切换到备用服务                                        │   │   │
│  │  │ • 使用本地缓存                                          │   │   │
│  │  │ • 提供只读模式                                          │   │   │
│  │  │ • 排队等待恢复                                          │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  │  级别4: 完全降级 (Full Degradation)                              │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ 核心服务不可用时:                                       │   │   │
│  │  │ • 切换到离线模式                                        │   │   │
│  │  │ • 保存到本地存储                                        │   │   │
│  │  │ • 提供导出功能                                          │   │   │
│  │  │ • 提示用户稍后重试                                      │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  降级决策流程:                                                           │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                                                                 │   │
│  │  故障检测 ──► 评估影响 ──► 选择降级级别 ──► 执行降级           │   │
│  │     │                                              │            │   │
│  │     │                                              ▼            │   │
│  │     │                                         通知用户          │   │
│  │     │                                              │            │   │
│  │     │                                              ▼            │   │
│  │     └───────────── 监控恢复 ◄──────── 持续检查 ◄────────        │   │
│  │                                                                 │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 7.4.1 降级策略实现

```typescript
// 降级管理器
class DegradationManager {
  private currentLevel: DegradationLevel;
  private strategies: Map<DegradationLevel, DegradationStrategy>;
  private healthMonitor: HealthMonitor;

  constructor() {
    this.currentLevel = DegradationLevel.NORMAL;
    this.strategies = new Map();
    this.healthMonitor = new HealthMonitor();

    this.registerStrategies();
    this.startMonitoring();
  }

  // 注册降级策略
  private registerStrategies(): void {
    this.strategies.set(DegradationLevel.FEATURE, new FeatureDegradationStrategy());
    this.strategies.set(DegradationLevel.PERFORMANCE, new PerformanceDegradationStrategy());
    this.strategies.set(DegradationLevel.SERVICE, new ServiceDegradationStrategy());
    this.strategies.set(DegradationLevel.FULL, new FullDegradationStrategy());
  }

  // 启动监控
  private startMonitoring(): void {
    this.healthMonitor.on('healthChange', (status: HealthStatus) => {
      this.evaluateDegradation(status);
    });
  }

  // 评估是否需要降级
  private evaluateDegradation(status: HealthStatus): void {
    const newLevel = this.determineDegradationLevel(status);

    if (newLevel !== this.currentLevel) {
      this.applyDegradation(newLevel);
    }
  }

  // 确定降级级别
  private determineDegradationLevel(status: HealthStatus): DegradationLevel {
    if (!status.servicesAvailable) {
      return DegradationLevel.FULL;
    }

    if (status.latency > 5000 || status.errorRate > 0.5) {
      return DegradationLevel.SERVICE;
    }

    if (status.latency > 1000 || status.errorRate > 0.1) {
      return DegradationLevel.PERFORMANCE;
    }

    if (!status.realtimeAvailable) {
      return DegradationLevel.FEATURE;
    }

    return DegradationLevel.NORMAL;
  }

  // 应用降级
  private async applyDegradation(level: DegradationLevel): Promise<void> {
    console.log(`Applying degradation level: ${level}`);

    const strategy = this.strategies.get(level);

    if (strategy) {
      await strategy.apply();
    }

    this.currentLevel = level;

    // 通知用户
    this.notifyUser(level);
  }

  // 通知用户
  private notifyUser(level: DegradationLevel): void {
    const messages: Record<DegradationLevel, string> = {
      [DegradationLevel.NORMAL]: '所有功能正常',
      [DegradationLevel.FEATURE]: '实时协作功能暂时不可用，您可以继续本地编辑',
      [DegradationLevel.PERFORMANCE]: '系统响应较慢，部分功能可能受限',
      [DegradationLevel.SERVICE]: '部分服务不可用，正在使用备用服务',
      [DegradationLevel.FULL]: '服务暂时不可用，已切换到离线模式'
    };

    // 显示通知
    this.showNotification(messages[level], level);
  }

  // 显示通知
  private showNotification(message: string, level: DegradationLevel): void {
    // 实现通知显示逻辑
    console.log(`[${level}] ${message}`);
  }

  // 恢复服务
  async recover(): Promise<void> {
    if (this.currentLevel !== DegradationLevel.NORMAL) {
      await this.applyDegradation(DegradationLevel.NORMAL);
    }
  }

  // 获取当前降级级别
  getCurrentLevel(): DegradationLevel {
    return this.currentLevel;
  }
}

// 功能降级策略
class FeatureDegradationStrategy implements DegradationStrategy {
  async apply(): Promise<void> {
    // 禁用实时同步
    disableRealtimeSync();

    // 禁用光标显示
    disableCursorDisplay();

    // 禁用在线用户列表
    disableUserList();

    // 启用本地编辑
    enableLocalEdit();
  }
}

// 性能降级策略
class PerformanceDegradationStrategy implements DegradationStrategy {
  async apply(): Promise<void> {
    // 降低同步频率
    reduceSyncFrequency();

    // 减少历史记录
    reduceHistoryRetention();

    // 启用流控
    enableRateLimiting();
  }
}

// 服务降级策略
class ServiceDegradationStrategy implements DegradationStrategy {
  async apply(): Promise<void> {
    // 切换到备用服务
    switchToBackupService();

    // 使用本地缓存
    enableLocalCache();

    // 启用只读模式
    enableReadOnlyMode();
  }
}

// 完全降级策略
class FullDegradationStrategy implements DegradationStrategy {
  async apply(): Promise<void> {
    // 切换到离线模式
    enableOfflineMode();

    // 保存到本地存储
    enableLocalStorage();

    // 提供导出功能
    enableExportFeature();
  }
}

// 降级策略接口
interface DegradationStrategy {
  apply(): Promise<void>;
}

// 降级级别
enum DegradationLevel {
  NORMAL = 'normal',
  FEATURE = 'feature',
  PERFORMANCE = 'performance',
  SERVICE = 'service',
  FULL = 'full'
}

// 健康监控器
class HealthMonitor extends EventEmitter {
  private checkInterval: number;
  private intervalId: NodeJS.Timeout | null;

  constructor(checkInterval: number = 5000) {
    super();
    this.checkInterval = checkInterval;
    this.intervalId = null;
  }

  start(): void {
    this.intervalId = setInterval(() => {
      this.checkHealth();
    }, this.checkInterval);
  }

  stop(): void {
    if (this.intervalId) {
      clearInterval(this.intervalId);
    }
  }

  private async checkHealth(): Promise<void> {
    const status: HealthStatus = {
      realtimeAvailable: await this.checkRealtimeService(),
      servicesAvailable: await this.checkServices(),
      latency: await this.measureLatency(),
      errorRate: await this.calculateErrorRate()
    };

    this.emit('healthChange', status);
  }

  private async checkRealtimeService(): Promise<boolean> {
    // 检查实时服务状态
    return true;
  }

  private async checkServices(): Promise<boolean> {
    // 检查服务状态
    return true;
  }

  private async measureLatency(): Promise<number> {
    // 测量延迟
    return 0;
  }

  private async calculateErrorRate(): Promise<number> {
    // 计算错误率
    return 0;
  }
}

// 健康状态
interface HealthStatus {
  realtimeAvailable: boolean;
  servicesAvailable: boolean;
  latency: number;
  errorRate: number;
}

// 辅助函数
function disableRealtimeSync(): void {}
function disableCursorDisplay(): void {}
function disableUserList(): void {}
function enableLocalEdit(): void {}
function reduceSyncFrequency(): void {}
function reduceHistoryRetention(): void {}
function enableRateLimiting(): void {}
function switchToBackupService(): void {}
function enableLocalCache(): void {}
function enableReadOnlyMode(): void {}
function enableOfflineMode(): void {}
function enableLocalStorage(): void {}
function enableExportFeature(): void {}

// EventEmitter 简化实现
class EventEmitter {
  private listeners: Map<string, Function[]> = new Map();

  on(event: string, listener: Function): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(listener);
  }

  emit(event: string, data: any): void {
    const listeners = this.listeners.get(event);
    if (listeners) {
      listeners.forEach(listener => listener(data));
    }
  }
}

---

## 总结

本报告详细设计了半自动化建筑设计平台的并发协作架构，涵盖以下核心内容：

### 关键技术选型
- **CRDT引擎**: 使用Yjs实现，支持Y.Array、Y.Map、Y.Text等数据类型
- **并发控制**: 乐观锁 + MVCC + 版本向量
- **一致性模型**: 因果一致性，支持读写一致性保证
- **消息队列**: Redis Pub/Sub + Kafka

### 核心设计亮点
1. **分层协作架构**: 应用层、协作引擎层、通信层、存储层清晰分离
2. **完善的CRDT实现**: 几何数据和属性数据的CRDT专用实现
3. **实时同步机制**: WebSocket网关、操作广播、本地预测、断线重连
4. **多层次冲突处理**: 语法、语义、意图三层冲突检测与解决
5. **全面容错设计**: 故障检测、恢复、数据修复、降级策略

### 性能优化措施
- 批量操作合并与去重
- 增量更新减少传输
- 内存池与惰性加载
- 自适应网络优化

### 可靠性保障
- 多层级故障检测
- 自动重连与状态恢复
- 数据校验与修复
- 优雅降级机制

---

*文档版本: v1.0*
*设计阶段: 概要设计阶段*
*完成日期: 2024年*
