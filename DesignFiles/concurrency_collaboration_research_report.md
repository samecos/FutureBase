# 调研阶段-并发与协作技术调研报告

## 半自动化建筑设计平台

**版本**: v1.0  
**日期**: 2025年  
**编写**: 分布式系统专家

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [实时协作算法调研](#2-实时协作算法调研)
3. [并发控制策略](#3-并发控制策略)
4. [分布式一致性](#4-分布式一致性)
5. [冲突检测与解决](#5-冲突检测与解决)
6. [实时同步机制](#6-实时同步机制)
7. [撤销重做架构](#7-撤销重做架构)
8. [技术选型建议](#8-技术选型建议)
9. [架构示意图](#9-架构示意图)
10. [参考资料](#10-参考资料)

---

## 1. 执行摘要

### 1.1 调研背景

半自动化建筑设计平台需要支持多专业工程师并发编辑同一项目，这对系统的并发控制和实时协作能力提出了极高要求。本报告针对以下核心需求进行技术调研：

- 多专业工程师并发编辑同一项目
- 实时协作和同步
- 撤销重做功能（后端存储）
- 脚本执行的任务调度
- 冲突检测和解决

### 1.2 核心结论

| 技术领域 | 推荐方案 | 理由 |
|---------|---------|------|
| 实时协作算法 | **CRDT为主，OT为辅** | 更适合离线协作、去中心化场景 |
| 并发控制 | **乐观锁+MVCC** | 读多写少场景性能更优 |
| 一致性模型 | **因果一致性** | 平衡性能与用户体验 |
| 冲突解决 | **自动合并+人工介入** | 几何冲突需专业判断 |
| 实时同步 | **WebSocket+消息队列** | 业界成熟方案 |
| 撤销重做 | **命令模式+操作日志** | 支持跨用户边界处理 |

---

## 2. 实时协作算法调研

### 2.1 Operational Transformation (OT) 算法

#### 2.1.1 核心原理

OT算法通过"操作转换"来解决并发编辑冲突。当用户执行操作时，系统会将该操作与并发的其他操作进行转换，确保所有用户最终看到一致的结果。

**基本思想**：
- 每个操作在应用到文档前，需要根据已发生的并发操作进行转换
- 转换函数保证：transform(Oa, Ob) = (Oa', Ob')，使得 Oa' 和 Ob' 可以顺序应用而不冲突

```
用户A执行操作 Oa，用户B执行操作 Ob（并发）

OT转换:
  transform(Oa, Ob) → (Oa', Ob')
  
用户A应用: Oa 然后 Ob'
用户B应用: Ob 然后 Oa'

最终结果一致
```

#### 2.1.2 算法伪代码

```python
class OTOperation:
    def __init__(self, type, position, content=None, length=None):
        self.type = type  # 'insert' | 'delete' | 'retain'
        self.position = position
        self.content = content
        self.length = length

class OTTransform:
    """OT转换核心算法"""
    
    @staticmethod
    def transform_insert_insert(op1, op2):
        """两个插入操作的转换"""
        if op1.position <= op2.position:
            # op1在op2之前插入，op2位置后移
            return op1, OTOperation('insert', op2.position + len(op1.content), op2.content)
        else:
            # op2在op1之前插入，op1位置后移
            return OTOperation('insert', op1.position + len(op2.content), op1.content), op2
    
    @staticmethod
    def transform_insert_delete(op1, op2):
        """插入与删除操作的转换"""
        if op1.position <= op2.position:
            return op1, OTOperation('delete', op2.position + len(op1.content), length=op2.length)
        elif op1.position >= op2.position + op2.length:
            return OTOperation('insert', op1.position - op2.length, op1.content), op2
        else:
            # 插入位置在删除范围内，需要特殊处理
            return op1, op2  # 简化处理，实际需更复杂逻辑
    
    @staticmethod
    def transform_operations(op1, op2):
        """通用转换入口"""
        if op1.type == 'insert' and op2.type == 'insert':
            return OTTransform.transform_insert_insert(op1, op2)
        elif op1.type == 'insert' and op2.type == 'delete':
            return OTTransform.transform_insert_delete(op1, op2)
        elif op1.type == 'delete' and op2.type == 'insert':
            op2_new, op1_new = OTTransform.transform_insert_delete(op2, op1)
            return op1_new, op2_new
        # ... 其他组合
```

#### 2.1.3 OT特性

| 特性 | 说明 |
|-----|------|
| TP1 (Transformation Property 1) | 确保操作转换后语义等价 |
| TP2 (Transformation Property 2) | 确保多操作转换的一致性 |
| 收敛性 | 所有副本最终达到一致状态 |
| 意图保持 | 尽可能保持用户的原始意图 |

#### 2.1.4 优缺点

**优点**：
- 成熟稳定，Google Docs等产品验证
- 意图保持能力强
- 元数据开销小

**缺点**：
- 转换函数复杂，随操作类型增加复杂度指数增长
- 依赖中心化服务器协调顺序
- 对延迟和乱序敏感
- 离线支持困难

---

### 2.2 CRDT (Conflict-free Replicated Data Types)

#### 2.2.1 核心原理

CRDT通过特殊的数据结构设计，使得冲突在数学上不可能发生。操作满足交换律、结合律和幂等性，可以任意顺序应用。

**核心性质**：
- **交换律**: merge(A, B) = merge(B, A)
- **结合律**: merge(merge(A, B), C) = merge(A, merge(B, C))
- **幂等性**: merge(A, A) = A

#### 2.2.2 CRDT类型

| 类型 | 用途 | 合并策略 |
|-----|------|---------|
| G-Counter | 只增计数器 | 取各副本最大值 |
| PN-Counter | 可增可减计数器 | 两个G-Counter组合 |
| G-Set | 只增集合 | 并集 |
| 2P-Set | 可增可删集合 | 添加集-删除集 |
| OR-Set | 观察删除集合 | 唯一标识符追踪 |
| LWW-Register | 最后写入获胜 | 时间戳比较 |
| Sequence CRDT | 协作文本编辑 | RGA/LSEQ/Logoot |

#### 2.2.3 Sequence CRDT实现（RGA算法）

```python
import uuid
from typing import List, Optional

class RGANode:
    """RGA (Replicated Growable Array) 节点"""
    def __init__(self, id: str, content: str, origin_left: Optional[str] = None):
        self.id = id                    # 唯一标识符
        self.content = content          # 内容
        self.origin_left = origin_left  # 左邻居ID（用于排序）
        self.is_deleted = False         # 删除标记
        self.timestamp = 0              # 时间戳

class RGASequence:
    """RGA序列CRDT实现"""
    
    def __init__(self):
        self.nodes: dict[str, RGANode] = {}
        self.head = RGANode("head", "", None)
        self.nodes["head"] = self.head
        self.id_counter = 0
    
    def _generate_id(self, replica_id: str) -> str:
        """生成唯一节点ID"""
        self.id_counter += 1
        return f"{replica_id}:{self.id_counter}:{uuid.uuid4().hex[:8]}"
    
    def insert(self, position: int, content: str, replica_id: str) -> RGANode:
        """在指定位置插入内容"""
        # 找到插入位置的左邻居
        visible_nodes = self.get_visible_nodes()
        if position == 0:
            origin_left = "head"
        else:
            origin_left = visible_nodes[position - 1].id
        
        # 创建新节点
        node_id = self._generate_id(replica_id)
        new_node = RGANode(node_id, content, origin_left)
        
        # 添加到节点集合
        self.nodes[node_id] = new_node
        
        return new_node
    
    def delete(self, position: int) -> str:
        """删除指定位置的内容"""
        visible_nodes = self.get_visible_nodes()
        node = visible_nodes[position]
        node.is_deleted = True
        return node.id
    
    def merge(self, other: 'RGASequence'):
        """合并另一个RGA序列"""
        for node_id, node in other.nodes.items():
            if node_id not in self.nodes:
                # 新节点，直接添加
                self.nodes[node_id] = RGANode(
                    node.id, node.content, node.origin_left
                )
                self.nodes[node_id].is_deleted = node.is_deleted
                self.nodes[node_id].timestamp = node.timestamp
            else:
                # 已存在，合并删除状态
                existing = self.nodes[node_id]
                existing.is_deleted = existing.is_deleted or node.is_deleted
                existing.timestamp = max(existing.timestamp, node.timestamp)
    
    def get_visible_nodes(self) -> List[RGANode]:
        """获取可见节点列表（按origin_left排序）"""
        # 使用拓扑排序确保正确的顺序
        visible = [n for n in self.nodes.values() if not n.is_deleted]
        # 简化的排序逻辑（实际需更复杂的排序算法）
        return sorted(visible, key=lambda n: (n.origin_left or "", n.id))
    
    def get_content(self) -> str:
        """获取当前内容"""
        return "".join(n.content for n in self.get_visible_nodes() if n.id != "head")
```

#### 2.2.4 CRDT优缺点

**优点**：
- 无需中心化协调，支持P2P架构
- 天然支持离线编辑和断线重连
- 对网络延迟和乱序不敏感
- 实现正确性可数学证明

**缺点**：
- 元数据开销较大（需存储唯一ID、时间戳等）
- 大文档性能需优化
- 意图保持不如OT精确
- 实现门槛较高

---

### 2.3 OT vs CRDT 对比分析

| 维度 | OT | CRDT |
|-----|-----|------|
| **核心思想** | 转换远程操作以适应本地状态 | 设计特殊数据结构，合并后自然一致 |
| **架构依赖** | 依赖中心化服务器协调顺序 | 支持去中心化/P2P |
| **实现复杂度** | 转换函数复杂（随操作类型增加） | 数据结构设计复杂（一次性成本） |
| **网络适应性** | 对延迟和乱序敏感（需重排序） | 对延迟和乱序不敏感（离线友好） |
| **性能（大文档）** | 较好（无额外元数据） | 需优化（元数据遍历成本） |
| **意图保持** | 强 | 中等 |
| **离线支持** | 困难 | 天然支持 |
| **典型应用** | Google Docs（早期）、Etherpad | Notion、Figma、Yjs、Automerge |

### 2.4 建筑设计平台的选择建议

**推荐：CRDT为主，OT为辅的混合架构**

理由：
1. **离线协作需求**：建筑师可能需要在工地等网络不稳定环境工作
2. **多专业并行**：结构、机电、建筑等专业可独立工作后再合并
3. **版本管理**：CRDT天然支持分支和合并
4. **几何数据**：可使用专门的CRDT类型处理

---

### 2.5 几何数据协作的特殊挑战

#### 2.5.1 挑战分析

| 挑战 | 说明 | 影响 |
|-----|------|------|
| 空间关系复杂 | 墙体、门窗、构件间存在拓扑关系 | 简单合并可能破坏设计意图 |
| 参数化依赖 | 参数驱动几何，修改参数影响多个元素 | 需要传播更新 |
| 精度要求高 | 毫米级精度要求 | 浮点数合并需特殊处理 |
| 语义丰富 | 几何不仅是形状，还包含材料、功能信息 | 需多维度合并 |

#### 2.5.2 几何CRDT设计思路

```python
class GeometryCRDT:
    """几何数据CRDT实现思路"""
    
    def __init__(self):
        self.elements = {}  # 元素CRDT (OR-Map)
        self.relations = {}  # 关系CRDT (OR-Set)
        self.parameters = {}  # 参数CRDT (LWW-Register)
    
    class ElementCRDT:
        """单个几何元素CRDT"""
        def __init__(self, element_id):
            self.id = element_id
            self.geometry = LWWRegister()  # 几何数据（最后写入获胜）
            self.material = LWWRegister()  # 材质
            self.transform = LWWRegister()  # 变换矩阵
            self.version = GCounter()  # 版本计数器
        
        def merge(self, other):
            self.geometry.merge(other.geometry)
            self.material.merge(other.material)
            self.transform.merge(other.transform)
            self.version.merge(other.version)
    
    class RelationCRDT:
        """元素间关系CRDT"""
        def __init__(self):
            self.relations = ORSet()  # 观察删除集合
        
        def add_relation(self, from_id, to_id, relation_type):
            self.relations.add((from_id, to_id, relation_type))
        
        def remove_relation(self, from_id, to_id, relation_type):
            self.relations.remove((from_id, to_id, relation_type))
```

---

## 3. 并发控制策略

### 3.1 乐观锁 vs 悲观锁

#### 3.1.1 核心概念对比

| 特性 | 乐观锁 | 悲观锁 |
|-----|-------|-------|
| **核心思想** | "先信任，后验证" | "先锁定，后操作" |
| **锁定时机** | 提交时检查冲突 | 读取时立即锁定 |
| **冲突处理** | 检测冲突后重试 | 阻塞等待 |
| **并发性能** | 高（无阻塞） | 低（串行化） |
| **实现位置** | 应用层 | 数据库层 |
| **适用场景** | 读多写少、冲突少 | 写多读少、冲突多 |

#### 3.1.2 乐观锁实现

```python
class OptimisticLock:
    """乐观锁实现"""
    
    def __init__(self, db_connection):
        self.db = db_connection
    
    def update_with_version(self, table, id, data, expected_version):
        """
        使用版本号实现乐观锁
        
        SQL示例:
        UPDATE projects 
        SET name = '新项目', version = version + 1 
        WHERE id = 123 AND version = 5
        """
        query = f"""
            UPDATE {table} 
            SET {', '.join(f"{k} = %s" for k in data.keys())}, 
                version = version + 1,
                updated_at = NOW()
            WHERE id = %s AND version = %s
        """
        params = list(data.values()) + [id, expected_version]
        
        result = self.db.execute(query, params)
        
        if result.rowcount == 0:
            # 更新失败，版本号不匹配
            raise ConflictException(
                f"Record {id} was modified by another transaction. "
                f"Expected version: {expected_version}"
            )
        
        return result
    
    def update_with_timestamp(self, table, id, data, expected_timestamp):
        """使用时间戳实现乐观锁"""
        query = f"""
            UPDATE {table} 
            SET {', '.join(f"{k} = %s" for k in data.keys())}, 
                updated_at = NOW()
            WHERE id = %s AND updated_at = %s
        """
        params = list(data.values()) + [id, expected_timestamp]
        
        result = self.db.execute(query, params)
        
        if result.rowcount == 0:
            raise ConflictException("Record was modified by another transaction")
        
        return result
```

#### 3.1.3 悲观锁实现

```python
class PessimisticLock:
    """悲观锁实现"""
    
    def __init__(self, db_connection):
        self.db = db_connection
    
    def acquire_lock(self, resource_type, resource_id, timeout=30):
        """获取悲观锁"""
        lock_key = f"{resource_type}:{resource_id}"
        
        # 使用数据库行锁
        query = """
            SELECT * FROM project_locks 
            WHERE resource_type = %s AND resource_id = %s
            FOR UPDATE NOWAIT
        """
        
        try:
            result = self.db.execute(query, (resource_type, resource_id))
            if not result.fetchone():
                # 创建锁记录
                self.db.execute("""
                    INSERT INTO project_locks (resource_type, resource_id, locked_at, timeout)
                    VALUES (%s, %s, NOW(), %s)
                """, (resource_type, resource_id, timeout))
            return True
        except DatabaseLockException:
            raise LockAcquisitionFailed(f"Could not acquire lock for {lock_key}")
    
    def release_lock(self, resource_type, resource_id):
        """释放悲观锁"""
        self.db.execute("""
            DELETE FROM project_locks 
            WHERE resource_type = %s AND resource_id = %s
        """, (resource_type, resource_id))
    
    def with_lock(self, resource_type, resource_id, operation):
        """上下文管理器方式使用悲观锁"""
        self.acquire_lock(resource_type, resource_id)
        try:
            return operation()
        finally:
            self.release_lock(resource_type, resource_id)
```

### 3.2 MVCC (多版本并发控制)

#### 3.2.1 核心原理

MVCC通过维护数据的多个版本，实现读写不阻塞。每个事务看到的是一个快照，写操作创建新版本而不影响正在进行的读操作。

```
数据记录结构:
+----------------+----------------+----------------+----------------+
|   实际数据      |   创建版本号    |   删除版本号    |   回滚指针     |
+----------------+----------------+----------------+----------------+

事务读取规则:
- 只能看到创建版本号 ≤ 当前事务版本号 且 删除版本号 > 当前事务版本号 的记录
- 或删除版本号为NULL的记录
```

#### 3.2.2 MVCC实现伪代码

```python
class MVCCRecord:
    """MVCC记录结构"""
    def __init__(self, data, tx_id):
        self.data = data
        self.created_by = tx_id      # 创建此版本的事务ID
        self.deleted_by = None       # 删除此版本的事务ID
        self.prev_version = None     # 指向前一个版本的指针

class MVCCDatabase:
    """MVCC数据库实现"""
    
    def __init__(self):
        self.records = {}  # key -> list of MVCCRecord
        self.active_transactions = {}  # tx_id -> Transaction
        self.global_tx_id = 0
    
    def begin_transaction(self):
        """开始事务"""
        self.global_tx_id += 1
        tx_id = self.global_tx_id
        self.active_transactions[tx_id] = {
            'id': tx_id,
            'start_time': time.time(),
            'isolation_level': 'READ_COMMITTED'
        }
        return tx_id
    
    def read(self, key, tx_id):
        """MVCC读操作"""
        if key not in self.records:
            return None
        
        versions = self.records[key]
        
        # 找到对当前事务可见的最新版本
        for record in reversed(versions):
            if self._is_visible(record, tx_id):
                return record.data
        
        return None
    
    def _is_visible(self, record, tx_id):
        """判断记录对当前事务是否可见"""
        # 记录创建者已提交且创建版本号 <= 当前事务ID
        created_visible = record.created_by <= tx_id
        
        # 记录未被删除，或删除者未提交或删除版本号 > 当前事务ID
        not_deleted = record.deleted_by is None or record.deleted_by > tx_id
        
        return created_visible and not_deleted
    
    def write(self, key, data, tx_id):
        """MVCC写操作 - 创建新版本"""
        new_record = MVCCRecord(data, tx_id)
        
        if key in self.records:
            # 标记旧版本为删除
            old_record = self.records[key][-1]
            old_record.deleted_by = tx_id
            
            # 新版本的回滚指针指向旧版本
            new_record.prev_version = old_record
        
        # 添加新版本
        if key not in self.records:
            self.records[key] = []
        self.records[key].append(new_record)
        
        return new_record
    
    def commit(self, tx_id):
        """提交事务"""
        del self.active_transactions[tx_id]
        # 实际实现中需要处理版本清理
    
    def rollback(self, tx_id):
        """回滚事务 - 清理未提交的版本"""
        for key, versions in self.records.items():
            # 删除由本事务创建但未提交的记录
            self.records[key] = [
                v for v in versions 
                if v.created_by != tx_id
            ]
            # 恢复被本事务标记删除的记录
            for v in self.records[key]:
                if v.deleted_by == tx_id:
                    v.deleted_by = None
        
        del self.active_transactions[tx_id]
```

#### 3.2.3 MVCC优势

| 优势 | 说明 |
|-----|------|
| 读写不阻塞 | 读操作不会阻塞写操作，反之亦然 |
| 无锁读 | 读操作无需获取锁，提高并发性能 |
| 快照隔离 | 事务看到一致性的快照视图 |
| 支持时间旅行查询 | 可查询历史版本数据 |

### 3.3 细粒度锁 vs 粗粒度锁

#### 3.3.1 概念对比

| 特性 | 粗粒度锁 | 细粒度锁 |
|-----|---------|---------|
| **锁范围** | 整个数据结构/表 | 单个元素/行 |
| **并发度** | 低（串行化） | 高（并行化） |
| **实现复杂度** | 简单 | 复杂 |
| **死锁风险** | 低 | 高 |
| **内存开销** | 低 | 高 |
| **适用场景** | 低并发、简单操作 | 高并发、复杂操作 |

#### 3.3.2 锁粒度选择策略

```python
class LockGranularityManager:
    """锁粒度管理器"""
    
    def __init__(self):
        self.global_lock = threading.Lock()
        self.segment_locks = [threading.Lock() for _ in range(16)]
        self.element_locks = {}
    
    def get_lock(self, key, granularity='auto'):
        """
        根据策略获取合适的锁
        
        granularity选项:
        - 'coarse': 使用全局锁
        - 'segmented': 使用分段锁
        - 'fine': 使用元素级锁
        - 'auto': 根据负载自动选择
        """
        if granularity == 'coarse':
            return self.global_lock
        
        elif granularity == 'segmented':
            segment = hash(key) % len(self.segment_locks)
            return self.segment_locks[segment]
        
        elif granularity == 'fine':
            if key not in self.element_locks:
                self.element_locks[key] = threading.Lock()
            return self.element_locks[key]
        
        elif granularity == 'auto':
            # 根据当前负载自动选择
            contention_level = self._measure_contention(key)
            if contention_level < 0.1:
                return self.global_lock
            elif contention_level < 0.5:
                segment = hash(key) % len(self.segment_locks)
                return self.segment_locks[segment]
            else:
                if key not in self.element_locks:
                    self.element_locks[key] = threading.Lock()
                return self.element_locks[key]
    
    def _measure_contention(self, key):
        """测量锁竞争程度（简化实现）"""
        # 实际实现需要统计锁等待时间、获取失败次数等
        return 0.3  # 示例返回值
```

### 3.4 专业级别的编辑隔离策略

#### 3.4.1 建筑设计平台的专业隔离模型

```python
class ProfessionalIsolationManager:
    """专业级别编辑隔离管理器"""
    
    def __init__(self):
        self.profession_locks = {}  # 专业级锁
        self.discipline_zones = {}  # 专业工作区
        self.shared_elements = {}   # 共享元素锁
    
    class IsolationLevel(Enum):
        """隔离级别"""
        NONE = 0           # 无隔离，完全协作
        DISCIPLINE = 1     # 专业隔离
        ZONE = 2           # 区域隔离
        ELEMENT = 3        # 元素级隔离
        EXCLUSIVE = 4      # 独占模式
    
    def acquire_edit_rights(self, user_id, profession, target, level):
        """
        获取编辑权限
        
        Args:
            user_id: 用户ID
            profession: 专业（结构/建筑/机电等）
            target: 编辑目标（元素ID/区域ID）
            level: 隔离级别
        """
        if level == self.IsolationLevel.DISCIPLINE:
            # 专业级隔离：不同专业可同时编辑
            return self._acquire_discipline_lock(profession, target)
        
        elif level == self.IsolationLevel.ZONE:
            # 区域级隔离：不同区域可同时编辑
            return self._acquire_zone_lock(user_id, profession, target)
        
        elif level == self.IsolationLevel.ELEMENT:
            # 元素级隔离：单个元素锁定
            return self._acquire_element_lock(user_id, target)
        
        elif level == self.IsolationLevel.EXCLUSIVE:
            # 独占模式：整个项目锁定
            return self._acquire_exclusive_lock(user_id)
    
    def _acquire_discipline_lock(self, profession, target):
        """专业级锁定"""
        # 结构专业可编辑：墙体、柱、梁
        # 建筑专业可编辑：房间、门窗、装饰
        # 机电专业可编辑：管道、设备、线路
        
        profession_permissions = {
            'structural': ['wall', 'column', 'beam', 'slab', 'foundation'],
            'architectural': ['room', 'door', 'window', 'facade', 'finishing'],
            'mep': ['pipe', 'duct', 'equipment', 'cable', 'outlet'],
            'all': []  # 只读权限
        }
        
        element_type = self._get_element_type(target)
        allowed_types = profession_permissions.get(profession, [])
        
        if element_type in allowed_types:
            return LockGrantResult(success=True)
        else:
            return LockGrantResult(
                success=False,
                reason=f"Profession '{profession}' cannot edit '{element_type}'"
            )
```

---

## 4. 分布式一致性

### 4.1 CAP定理在系统中的权衡

#### 4.1.1 CAP定理核心

```
CAP定理：在分布式系统中，最多只能同时满足以下三项中的两项：

┌─────────────────────────────────────────────────────────────┐
│                        CAP定理                               │
│                                                             │
│                    Consistency                              │
│                    (一致性)                                  │
│                          ▲                                  │
│                         / \                                 │
│                        /   \                                │
│                       /     \                               │
│                      /   ▲   \                              │
│                     /   / \   \                             │
│                    /   /   \   \                            │
│                   /   /     \   \                           │
│         Partition<───/───────\───>Availability              │
│         Tolerance    /         (可用性)                      │
│         (分区容错性) /                                       │
│                     /                                        │
│                                                             │
│  注：分区容错性(P)是必选项，实际是在C和A之间做选择             │
└─────────────────────────────────────────────────────────────┘
```

#### 4.1.2 三种系统类型

| 类型 | 特性 | 代表系统 | 适用场景 |
|-----|------|---------|---------|
| **CA** | 一致+可用，无分区容错 | 单节点数据库 | 非分布式系统 |
| **CP** | 一致+分区容错，牺牲可用 | ZooKeeper, etcd, MongoDB | 金融交易、配置中心 |
| **AP** | 可用+分区容错，牺牲一致 | Cassandra, DynamoDB, CouchDB | 社交网络、内容分发 |

### 4.2 一致性级别选择

#### 4.2.1 一致性级别对比

| 一致性级别 | 保证 | 延迟 | 可用性 | 适用场景 |
|-----------|------|------|-------|---------|
| **强一致** | 所有读看到最新写 | 高 | 低 | 金融交易、库存管理 |
| **因果一致** | 因果操作有序 | 中 | 高 | 协作编辑、聊天应用 |
| **读己所写** | 用户看到自己的更新 | 低 | 高 | 用户配置更新 |
| **单调读** | 不会读到旧版本 | 低 | 高 | 时间线、消息历史 |
| **最终一致** | 最终达到一致 | 最低 | 最高 | 社交点赞、计数器 |

#### 4.2.2 建筑设计平台推荐：因果一致性

```python
class CausalConsistencyManager:
    """因果一致性管理器"""
    
    def __init__(self):
        self.vector_clocks = {}  # 向量时钟
        self.dependency_graph = {}  # 依赖图
    
    class VectorClock:
        """向量时钟实现"""
        def __init__(self, node_id):
            self.clock = {node_id: 0}
            self.node_id = node_id
        
        def increment(self):
            self.clock[self.node_id] += 1
            return self
        
        def merge(self, other):
            for node, time in other.clock.items():
                self.clock[node] = max(self.clock.get(node, 0), time)
            return self
        
        def compare(self, other):
            """
            比较两个向量时钟
            返回: 'before', 'after', 'concurrent', 'equal'
            """
            dominates = False
            dominated = False
            
            all_nodes = set(self.clock.keys()) | set(other.clock.keys())
            
            for node in all_nodes:
                v1 = self.clock.get(node, 0)
                v2 = other.clock.get(node, 0)
                
                if v1 > v2:
                    dominates = True
                elif v2 > v1:
                    dominated = True
            
            if dominates and not dominated:
                return 'after'
            elif dominated and not dominates:
                return 'before'
            elif not dominates and not dominated:
                return 'equal'
            else:
                return 'concurrent'
    
    def record_operation(self, user_id, operation, dependencies=None):
        """记录操作及其依赖"""
        # 获取用户的向量时钟
        if user_id not in self.vector_clocks:
            self.vector_clocks[user_id] = self.VectorClock(user_id)
        
        vc = self.vector_clocks[user_id]
        vc.increment()
        
        # 合并依赖的向量时钟
        if dependencies:
            for dep_vc in dependencies:
                vc.merge(dep_vc)
        
        # 记录操作
        op_record = {
            'user_id': user_id,
            'operation': operation,
            'vector_clock': vc.clock.copy(),
            'timestamp': time.time()
        }
        
        return op_record
    
    def can_apply(self, operation, local_vc):
        """检查操作是否可以应用（所有依赖已满足）"""
        op_vc = operation['vector_clock']
        
        for node, time in op_vc.items():
            if local_vc.clock.get(node, 0) < time:
                # 依赖未满足
                return False
        
        return True
```

### 4.3 分布式共识算法

#### 4.3.1 Raft算法

**核心组件**：
1. **Leader选举**：通过超时机制选举Leader
2. **日志复制**：Leader接收写请求并复制到Follower
3. **安全性**：确保已提交的日志不会被覆盖

```python
class RaftNode:
    """Raft节点简化实现"""
    
    def __init__(self, node_id, peers):
        self.id = node_id
        self.peers = peers
        
        # 持久化状态
        self.current_term = 0
        self.voted_for = None
        self.log = []
        
        # 易失状态
        self.state = 'follower'  # follower/candidate/leader
        self.commit_index = 0
        self.last_applied = 0
        
        # Leader状态
        self.next_index = {}
        self.match_index = {}
        
        # 计时器
        self.election_timeout = random.randint(150, 300)  # ms
        self.heartbeat_interval = 50  # ms
    
    def start_election(self):
        """开始选举"""
        self.state = 'candidate'
        self.current_term += 1
        self.voted_for = self.id
        
        votes = 1  # 自己投自己
        
        # 向其他节点请求投票
        for peer in self.peers:
            try:
                response = self.request_vote(peer)
                if response['vote_granted']:
                    votes += 1
            except:
                pass
        
        # 获得多数票则成为Leader
        if votes > len(self.peers) / 2:
            self.become_leader()
    
    def request_vote(self, peer):
        """请求投票RPC"""
        args = {
            'term': self.current_term,
            'candidate_id': self.id,
            'last_log_index': len(self.log) - 1,
            'last_log_term': self.log[-1]['term'] if self.log else 0
        }
        return peer.handle_request_vote(args)
    
    def become_leader(self):
        """成为Leader"""
        self.state = 'leader'
        
        # 初始化Leader状态
        for peer in self.peers:
            self.next_index[peer.id] = len(self.log)
            self.match_index[peer.id] = 0
        
        # 开始发送心跳
        self.start_heartbeat()
    
    def append_entries(self, peer):
        """发送日志条目"""
        prev_log_index = self.next_index[peer.id] - 1
        
        args = {
            'term': self.current_term,
            'leader_id': self.id,
            'prev_log_index': prev_log_index,
            'prev_log_term': self.log[prev_log_index]['term'] if prev_log_index >= 0 else 0,
            'entries': self.log[self.next_index[peer.id]:],
            'leader_commit': self.commit_index
        }
        
        return peer.handle_append_entries(args)
    
    def handle_append_entries(self, args):
        """处理AppendEntries RPC"""
        # 如果term小于当前term，拒绝
        if args['term'] < self.current_term:
            return {'term': self.current_term, 'success': False}
        
        # 重置选举超时
        self.reset_election_timer()
        
        # 检查日志一致性
        if args['prev_log_index'] >= 0:
            if args['prev_log_index'] >= len(self.log):
                return {'term': self.current_term, 'success': False}
            if self.log[args['prev_log_index']]['term'] != args['prev_log_term']:
                return {'term': self.current_term, 'success': False}
        
        # 追加新条目
        self.log = self.log[:args['prev_log_index'] + 1] + args['entries']
        
        # 更新commit_index
        if args['leader_commit'] > self.commit_index:
            self.commit_index = min(args['leader_commit'], len(self.log) - 1)
        
        return {'term': self.current_term, 'success': True}
```

#### 4.3.2 Raft vs Paxos

| 特性 | Raft | Paxos |
|-----|------|-------|
| **可理解性** | 高（设计目标） | 低（公认复杂） |
| **Leader** | 有明确Leader | 无明确Leader |
| **实现难度** | 中等 | 高 |
| **性能** | 中等 | 高（优化后） |
| **应用** | etcd, Consul, TiKV | Chubby, Spanner |

### 4.4 分区容忍策略

#### 4.4.1 网络分区处理策略

```python
class PartitionToleranceManager:
    """分区容忍管理器"""
    
    def __init__(self, consistency_preference='eventual'):
        self.consistency_preference = consistency_preference
        self.partition_state = 'connected'
        self.local_queue = []
        self.conflict_resolver = ConflictResolver()
    
    def handle_partition(self, detected_partition):
        """处理网络分区"""
        self.partition_state = 'partitioned'
        
        if self.consistency_preference == 'strong':
            # CP系统：拒绝写操作
            return self._enter_readonly_mode()
        else:
            # AP系统：继续本地操作
            return self._enter_partition_mode()
    
    def _enter_partition_mode(self):
        """进入分区模式（AP策略）"""
        return {
            'mode': 'partition_tolerant',
            'behavior': {
                'reads': 'local_replica',
                'writes': 'local_queue',
                'sync': 'async_delayed'
            }
        }
    
    def heal_partition(self, other_partition_state):
        """分区恢复处理"""
        self.partition_state = 'healing'
        
        # 检测冲突
        conflicts = self._detect_conflicts(other_partition_state)
        
        if conflicts:
            # 解决冲突
            resolution = self.conflict_resolver.resolve(conflicts)
            self._apply_resolution(resolution)
        
        # 同步状态
        self._synchronize_state(other_partition_state)
        
        self.partition_state = 'connected'
    
    def _detect_conflicts(self, other_state):
        """检测冲突"""
        conflicts = []
        
        for key, local_value in self.local_state.items():
            if key in other_state:
                other_value = other_state[key]
                if local_value != other_value:
                    conflicts.append({
                        'key': key,
                        'local': local_value,
                        'remote': other_value
                    })
        
        return conflicts
```

---

## 5. 冲突检测与解决

### 5.1 冲突类型分类

#### 5.1.1 建筑设计平台冲突类型

| 冲突类型 | 描述 | 示例 | 严重程度 |
|---------|------|------|---------|
| **几何冲突** | 元素空间位置重叠 | 两堵墙重叠、管道穿梁 | 高 |
| **属性冲突** | 同一元素属性不一致 | 墙体厚度不同定义 | 中 |
| **关系冲突** | 元素间关系被破坏 | 门依附的墙被删除 | 高 |
| **参数冲突** | 参数值冲突 | 房间面积与尺寸不匹配 | 中 |
| **版本冲突** | 基于不同版本修改 | A基于v1修改，B基于v2修改 | 中 |
| **语义冲突** | 设计意图冲突 | 结构要求vs建筑要求 | 高 |

#### 5.1.2 冲突检测伪代码

```python
class ConflictDetector:
    """冲突检测器"""
    
    def __init__(self):
        self.spatial_index = RTreeIndex()  # 空间索引
        self.dependency_graph = {}  # 依赖图
    
    def detect_all_conflicts(self, operations):
        """检测所有类型冲突"""
        conflicts = []
        
        for op in operations:
            # 几何冲突检测
            geo_conflicts = self.detect_geometric_conflicts(op)
            conflicts.extend(geo_conflicts)
            
            # 属性冲突检测
            attr_conflicts = self.detect_attribute_conflicts(op)
            conflicts.extend(attr_conflicts)
            
            # 关系冲突检测
            rel_conflicts = self.detect_relation_conflicts(op)
            conflicts.extend(rel_conflicts)
            
            # 参数冲突检测
            param_conflicts = self.detect_parameter_conflicts(op)
            conflicts.extend(param_conflicts)
        
        return conflicts
    
    def detect_geometric_conflicts(self, operation):
        """检测几何冲突"""
        conflicts = []
        
        if operation.type in ['create', 'update']:
            element = operation.target
            bbox = element.get_bounding_box()
            
            # 使用空间索引查找相交元素
            candidates = self.spatial_index.intersection(bbox)
            
            for candidate_id in candidates:
                candidate = self.get_element(candidate_id)
                
                if self._elements_intersect(element, candidate):
                    # 检查是否允许相交
                    if not self._intersection_allowed(element, candidate):
                        conflicts.append(GeometricConflict(
                            element1=element,
                            element2=candidate,
                            intersection_type=self._get_intersection_type(element, candidate)
                        ))
        
        return conflicts
    
    def detect_relation_conflicts(self, operation):
        """检测关系冲突"""
        conflicts = []
        
        if operation.type == 'delete':
            element = operation.target
            
            # 检查是否有依赖此元素的其他元素
            dependents = self.dependency_graph.get_dependents(element.id)
            
            for dependent in dependents:
                conflicts.append(RelationConflict(
                    deleted_element=element,
                    dependent_element=dependent,
                    relation_type=dependent.get_relation_type(element.id)
                ))
        
        return conflicts
    
    def _elements_intersect(self, elem1, elem2):
        """判断两个元素是否相交"""
        # 使用几何库进行相交检测
        return elem1.geometry.intersects(elem2.geometry)
```

### 5.2 自动合并策略

#### 5.2.1 合并策略实现

```python
class ConflictResolver:
    """冲突解决器"""
    
    def __init__(self):
        self.merge_strategies = {
            'geometric': GeometricMergeStrategy(),
            'attribute': AttributeMergeStrategy(),
            'relation': RelationMergeStrategy(),
            'parameter': ParameterMergeStrategy()
        }
    
    def resolve(self, conflicts):
        """解决冲突"""
        resolutions = []
        
        for conflict in conflicts:
            strategy = self.merge_strategies.get(conflict.type)
            
            if strategy and strategy.can_auto_resolve(conflict):
                # 自动解决
                resolution = strategy.resolve(conflict)
                resolutions.append(resolution)
            else:
                # 需要人工介入
                resolutions.append(ManualResolutionRequired(conflict))
        
        return resolutions

class GeometricMergeStrategy:
    """几何合并策略"""
    
    def can_auto_resolve(self, conflict):
        """判断是否可自动解决"""
        # 简单偏移可自动解决
        if conflict.intersection_type == 'overlap':
            return self._can_auto_adjust(conflict)
        return False
    
    def resolve(self, conflict):
        """解决几何冲突"""
        if conflict.intersection_type == 'overlap':
            # 尝试自动调整位置
            adjusted = self._auto_adjust_position(conflict)
            if adjusted:
                return AutoResolution(
                    conflict=conflict,
                    action='adjust_position',
                    result=adjusted
                )
        
        return ManualResolutionRequired(conflict)
    
    def _auto_adjust_position(self, conflict):
        """自动调整元素位置"""
        elem1, elem2 = conflict.element1, conflict.element2
        
        # 计算最小移动距离
        overlap = elem1.geometry.intersection(elem2.geometry)
        
        # 尝试移动优先级较低的元素
        if elem1.priority < elem2.priority:
            move_elem = elem1
            fixed_elem = elem2
        else:
            move_elem = elem2
            fixed_elem = elem1
        
        # 计算移动向量
        move_vector = self._calculate_move_vector(overlap, move_elem, fixed_elem)
        
        # 应用移动
        new_position = move_elem.position + move_vector
        
        return {
            'element': move_elem.id,
            'old_position': move_elem.position,
            'new_position': new_position
        }

class AttributeMergeStrategy:
    """属性合并策略"""
    
    def can_auto_resolve(self, conflict):
        """属性冲突通常需要人工判断"""
        # 数值属性可尝试平均
        if conflict.attribute_type == 'numeric':
            return True
        return False
    
    def resolve(self, conflict):
        """解决属性冲突"""
        if conflict.attribute_type == 'numeric':
            # 数值属性取平均
            merged_value = (conflict.value1 + conflict.value2) / 2
            return AutoResolution(
                conflict=conflict,
                action='average',
                result={'value': merged_value}
            )
        
        # 其他属性需要人工判断
        return ManualResolutionRequired(conflict)
```

### 5.3 人工介入的冲突解决UI设计建议

#### 5.3.1 冲突解决界面设计

```
┌─────────────────────────────────────────────────────────────────┐
│                     冲突解决界面                                 │
├─────────────────────────────────────────────────────────────────┤
│  发现 3 个冲突，已自动解决 1 个，需要人工处理 2 个              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 冲突 #1: 墙体位置重叠 [几何冲突]                         │   │
│  │                                                          │   │
│  │  ┌─────────────┐    ┌─────────────┐                     │   │
│  │  │  版本 A     │    │  版本 B     │                     │   │
│  │  │  位置: (0,0)│    │  位置: (5,0)│                     │   │
│  │  │  厚度: 200  │    │  厚度: 300  │                     │   │
│  │  └─────────────┘    └─────────────┘                     │   │
│  │                                                          │   │
│  │  [采纳A] [采纳B] [合并] [自定义] [标记待处理]            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 冲突 #2: 房间面积与尺寸不匹配 [参数冲突]                 │   │
│  │                                                          │   │
│  │  计算面积: 25.5 m²                                       │   │
│  │  标注面积: 30.0 m²                                       │   │
│  │                                                          │   │
│  │  [更新标注] [调整尺寸] [忽略]                            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  [上一步] [全部采纳A] [全部采纳B] [保存并继续]                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 5.3.2 冲突解决API设计

```python
class ConflictResolutionAPI:
    """冲突解决API"""
    
    def get_conflicts(self, project_id, version_a, version_b):
        """获取两个版本间的冲突列表"""
        pass
    
    def resolve_conflict(self, conflict_id, resolution_type, resolution_data):
        """
        解决单个冲突
        
        resolution_type:
        - 'accept_a': 采纳版本A
        - 'accept_b': 采纳版本B
        - 'merge': 合并
        - 'custom': 自定义
        - 'defer': 延后处理
        """
        pass
    
    def batch_resolve(self, conflict_ids, resolution_type):
        """批量解决冲突"""
        pass
    
    def preview_resolution(self, conflict_id, resolution_type):
        """预览解决结果"""
        pass
```

### 5.4 冲突预防机制

```python
class ConflictPreventionManager:
    """冲突预防管理器"""
    
    def __init__(self):
        self.lock_manager = LockManager()
        self.validation_rules = ValidationRuleEngine()
        self.notification_service = NotificationService()
    
    def prevent_conflicts(self, operation):
        """预防冲突"""
        # 1. 提前锁定
        lock_result = self.lock_manager.acquire_lock(
            resource=operation.target,
            lock_type='intent',
            duration='operation'
        )
        
        if not lock_result.success:
            return PreventionResult(
                success=False,
                reason='Resource already locked by another user',
                conflicting_user=lock_result.holder
            )
        
        # 2. 预验证
        validation = self.validation_rules.validate(operation)
        if not validation.passed:
            return PreventionResult(
                success=False,
                reason='Validation failed',
                violations=validation.violations
            )
        
        # 3. 通知相关用户
        affected_users = self._get_affected_users(operation)
        self.notification_service.notify(
            users=affected_users,
            message=f"User {operation.user_id} is editing {operation.target}"
        )
        
        return PreventionResult(success=True)
    
    def setup_proactive_locks(self, user_id, work_area):
        """设置主动锁定"""
        # 用户进入工作区时，预锁定相关资源
        resources = self._get_resources_in_area(work_area)
        
        for resource in resources:
            self.lock_manager.acquire_lock(
                resource=resource.id,
                lock_type='soft',
                holder=user_id,
                duration='session'
            )
```

---

## 6. 实时同步机制

### 6.1 WebSocket连接管理

#### 6.1.1 WebSocket架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        WebSocket架构                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   ┌─────────┐     ┌─────────────┐     ┌───────────────────────┐   │
│   │ Client  │◄───►│  Load Balancer│◄───►│  WebSocket Server 1   │   │
│   │   A     │     │   (Sticky)    │     │                       │   │
│   └─────────┘     └─────────────┘     └───────────────────────┘   │
│                                              │                      │
│   ┌─────────┐     ┌─────────────┐           │                      │
│   │ Client  │◄───►│  Load Balancer│◄────────┤                      │
│   │   B     │     │   (Sticky)    │         │                      │
│   └─────────┘     └─────────────┘     ┌─────┴─────────────────┐   │
│                                       │  WebSocket Server 2   │   │
│   ┌─────────┐                         │                       │   │
│   │ Client  │◄────────────────────────┤                       │   │
│   │   C     │                         └───────────────────────┘   │
│   └─────────┘                              │                       │
│                                            ▼                       │
│                                     ┌──────────────┐              │
│                                     │ Message Bus  │              │
│                                     │  (Redis/Kafka)│              │
│                                     └──────────────┘              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 6.1.2 WebSocket连接管理实现

```python
import asyncio
import websockets
from typing import Dict, Set
import json

class WebSocketManager:
    """WebSocket连接管理器"""
    
    def __init__(self):
        self.connections: Dict[str, websockets.WebSocketServerProtocol] = {}
        self.project_rooms: Dict[str, Set[str]] = {}  # project_id -> set of user_ids
        self.user_projects: Dict[str, str] = {}  # user_id -> project_id
        self.heartbeat_interval = 30  # 秒
    
    async def register(self, websocket, user_id, project_id):
        """注册新连接"""
        self.connections[user_id] = websocket
        self.user_projects[user_id] = project_id
        
        # 加入项目房间
        if project_id not in self.project_rooms:
            self.project_rooms[project_id] = set()
        self.project_rooms[project_id].add(user_id)
        
        # 通知其他用户
        await self.broadcast_to_project(
            project_id,
            {
                'type': 'user_joined',
                'user_id': user_id,
                'timestamp': time.time()
            },
            exclude=user_id
        )
        
        # 启动心跳检测
        asyncio.create_task(self._heartbeat(user_id))
    
    async def unregister(self, user_id):
        """注销连接"""
        if user_id not in self.connections:
            return
        
        project_id = self.user_projects.get(user_id)
        
        # 从房间移除
        if project_id and project_id in self.project_rooms:
            self.project_rooms[project_id].discard(user_id)
        
        # 清理连接
        del self.connections[user_id]
        if user_id in self.user_projects:
            del self.user_projects[user_id]
        
        # 通知其他用户
        if project_id:
            await self.broadcast_to_project(
                project_id,
                {
                    'type': 'user_left',
                    'user_id': user_id,
                    'timestamp': time.time()
                }
            )
    
    async def broadcast_to_project(self, project_id, message, exclude=None):
        """广播消息到项目房间"""
        if project_id not in self.project_rooms:
            return
        
        tasks = []
        for user_id in self.project_rooms[project_id]:
            if user_id != exclude and user_id in self.connections:
                websocket = self.connections[user_id]
                tasks.append(self._send_safe(websocket, message))
        
        if tasks:
            await asyncio.gather(*tasks, return_exceptions=True)
    
    async def send_to_user(self, user_id, message):
        """发送消息给特定用户"""
        if user_id in self.connections:
            await self._send_safe(self.connections[user_id], message)
    
    async def _send_safe(self, websocket, message):
        """安全发送消息"""
        try:
            await websocket.send(json.dumps(message))
        except websockets.exceptions.ConnectionClosed:
            pass
    
    async def _heartbeat(self, user_id):
        """心跳检测"""
        while user_id in self.connections:
            try:
                websocket = self.connections[user_id]
                await websocket.send(json.dumps({'type': 'ping'}))
                
                # 等待pong响应
                response = await asyncio.wait_for(
                    websocket.recv(),
                    timeout=self.heartbeat_interval
                )
                
                data = json.loads(response)
                if data.get('type') != 'pong':
                    raise Exception('Invalid heartbeat response')
                
                await asyncio.sleep(self.heartbeat_interval)
                
            except (asyncio.TimeoutError, websockets.exceptions.ConnectionClosed):
                # 心跳失败，断开连接
                await self.unregister(user_id)
                break
```

### 6.2 操作广播策略

#### 6.2.1 操作广播实现

```python
class OperationBroadcaster:
    """操作广播器"""
    
    def __init__(self, ws_manager, message_bus):
        self.ws_manager = ws_manager
        self.message_bus = message_bus
        self.operation_buffer = {}  # 操作缓冲区
        self.compression_threshold = 1024  # 压缩阈值
    
    async def broadcast_operation(self, operation, project_id):
        """广播操作到所有相关用户"""
        # 1. 序列化操作
        op_data = self._serialize_operation(operation)
        
        # 2. 压缩大操作
        if len(op_data) > self.compression_threshold:
            op_data = self._compress(op_data)
        
        # 3. 添加元数据
        message = {
            'type': 'operation',
            'operation': op_data,
            'sender': operation.user_id,
            'timestamp': operation.timestamp,
            'sequence': operation.sequence_number,
            'compressed': len(op_data) > self.compression_threshold
        }
        
        # 4. 本地广播
        await self.ws_manager.broadcast_to_project(
            project_id,
            message,
            exclude=operation.user_id
        )
        
        # 5. 发布到消息总线（用于跨服务器广播）
        await self.message_bus.publish(
            f"project:{project_id}:operations",
            message
        )
    
    async def broadcast_cursor_position(self, user_id, project_id, position):
        """广播光标位置（高频低优先级）"""
        message = {
            'type': 'cursor',
            'user_id': user_id,
            'position': position,
            'timestamp': time.time()
        }
        
        # 使用节流，避免过度广播
        await self._throttled_broadcast(
            project_id,
            message,
            interval=0.05  # 50ms节流
        )
    
    async def broadcast_selection(self, user_id, project_id, selection):
        """广播选择状态"""
        message = {
            'type': 'selection',
            'user_id': user_id,
            'elements': selection.element_ids,
            'timestamp': time.time()
        }
        
        await self.ws_manager.broadcast_to_project(project_id, message)
```

### 6.3 断线重连和状态恢复

#### 6.3.1 断线重连机制

```python
class ReconnectionManager:
    """断线重连管理器"""
    
    def __init__(self, state_manager, operation_log):
        self.state_manager = state_manager
        self.operation_log = operation_log
        self.client_states = {}  # 缓存的客户端状态
    
    async def handle_reconnection(self, user_id, project_id, last_sequence):
        """处理客户端重连"""
        # 1. 获取当前项目状态版本
        current_state = await self.state_manager.get_state(project_id)
        current_sequence = current_state.sequence_number
        
        # 2. 检查是否需要同步
        if last_sequence >= current_sequence:
            # 客户端已经是最新状态
            return ReconnectionResult(
                type='no_sync_needed',
                current_sequence=current_sequence
            )
        
        # 3. 计算缺失的操作
        missing_ops = await self.operation_log.get_operations(
            project_id,
            start_sequence=last_sequence + 1,
            end_sequence=current_sequence
        )
        
        # 4. 检查是否可以增量同步
        if len(missing_ops) < 100:  # 阈值可配置
            # 增量同步：发送缺失的操作
            return ReconnectionResult(
                type='incremental_sync',
                operations=missing_ops,
                current_sequence=current_sequence
            )
        else:
            # 全量同步：发送完整状态
            return ReconnectionResult(
                type='full_sync',
                state=current_state,
                current_sequence=current_sequence
            )
    
    async def save_client_state(self, user_id, state):
        """保存客户端状态（用于快速恢复）"""
        self.client_states[user_id] = {
            'state': state,
            'timestamp': time.time()
        }
    
    async def recover_from_snapshot(self, user_id, project_id):
        """从快照恢复"""
        snapshot = await self.state_manager.get_latest_snapshot(project_id)
        
        if not snapshot:
            # 无快照，返回空项目
            return ProjectState.empty(project_id)
        
        # 应用快照后的操作
        ops_after_snapshot = await self.operation_log.get_operations(
            project_id,
            start_sequence=snapshot.sequence + 1
        )
        
        # 重建状态
        state = snapshot.state.copy()
        for op in ops_after_snapshot:
            state = self._apply_operation(state, op)
        
        return state
```

### 6.4 网络延迟优化

#### 6.4.1 延迟优化策略

```python
class LatencyOptimizer:
    """延迟优化器"""
    
    def __init__(self):
        self.local_prediction = LocalPredictionEngine()
        self.operation_buffer = OperationBuffer()
        self.compression = CompressionEngine()
    
    def apply_local_prediction(self, operation):
        """本地预测：立即应用本地操作"""
        # 1. 立即在本地应用操作（不等待服务器确认）
        predicted_result = self.local_prediction.predict(operation)
        
        # 2. 标记为预测状态
        operation.is_predicted = True
        
        # 3. 发送操作到服务器
        self.send_to_server(operation)
        
        return predicted_result
    
    def handle_server_confirmation(self, operation_id, server_result):
        """处理服务器确认"""
        operation = self.operation_buffer.get(operation_id)
        
        if not operation:
            return
        
        if operation.is_predicted:
            # 比较预测结果与实际结果
            if self._results_differ(operation.predicted_result, server_result):
                # 预测错误，需要回滚并重新应用
                self._rollback_and_reapply(operation, server_result)
            else:
                # 预测正确，确认操作
                operation.confirm()
    
    def compress_operations(self, operations):
        """压缩操作序列"""
        # 1. 合并连续操作
        merged = self._merge_consecutive(operations)
        
        # 2. 删除冗余操作
        deduplicated = self._deduplicate(merged)
        
        # 3. 二进制压缩
        compressed = self.compression.compress(deduplicated)
        
        return compressed
    
    def _merge_consecutive(self, operations):
        """合并连续操作"""
        merged = []
        
        for op in operations:
            if merged and self._can_merge(merged[-1], op):
                merged[-1] = self._merge_two(merged[-1], op)
            else:
                merged.append(op)
        
        return merged
```

---

## 7. 撤销重做架构

### 7.1 命令模式实现

#### 7.1.1 命令模式核心结构

```
┌─────────────────────────────────────────────────────────────────┐
│                      命令模式架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐                                               │
│  │   Client     │                                               │
│  └──────┬───────┘                                               │
│         │                                                        │
│         ▼                                                        │
│  ┌──────────────┐     ┌──────────────────────────────────────┐ │
│  │   Invoker    │────►│         Command History              │ │
│  │  (History    │     │  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐  │ │
│  │   Manager)   │     │  │ Cmd │  │ Cmd │  │ Cmd │  │ Cmd │  │ │
│  └──────┬───────┘     │  │  1  │  │  2  │  │  3  │  │  4  │  │ │
│         │             │  └──┬──┘  └──┬──┘  └──┬──┘  └──┬──┘  │ │
│         │             │     │        │        │        │      │ │
│         │             │     └────────┴────────┴────────┘      │ │
│         │             │              ▲                          │ │
│         │             │         Undo Stack                      │ │
│         │             └──────────────────────────────────────┘ │
│         │                                                        │
│         │         execute()                                      │
│         ▼                                                        │
│  ┌──────────────┐                                               │
│  │   Command    │◄──────────────────────────────────────┐      │
│  │  Interface   │                                       │      │
│  └──────┬───────┘                                       │      │
│         │                                               │      │
│    ┌────┴────┬────────────┬────────────┐               │      │
│    ▼         ▼            ▼            ▼               │      │
│ ┌──────┐ ┌──────┐   ┌──────────┐  ┌──────────┐        │      │
│ │Create│ │Delete│   │  Update  │  │ Composite│        │      │
│ │ Wall │ │ Wall │   │  Wall    │  │ Command  │        │      │
│ └──┬───┘ └──┬───┘   └────┬─────┘  └────┬─────┘        │      │
│    │        │            │             │              │      │
│    └────────┴────────────┴─────────────┘              │      │
│                      │                                │      │
│                      ▼                                │      │
│              ┌──────────────┐                         │      │
│              │   Receiver   │                         │      │
│              │  (Project    │◄────────────────────────┘      │
│              │   Model)     │                                │
│              └──────────────┘                                │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

#### 7.1.2 命令模式实现

```python
from abc import ABC, abstractmethod
from typing import List, Any, Optional
from dataclasses import dataclass
import time
import json

class Command(ABC):
    """命令抽象基类"""
    
    def __init__(self, user_id: str, project_id: str):
        self.id = self._generate_id()
        self.user_id = user_id
        self.project_id = project_id
        self.timestamp = time.time()
        self.is_executed = False
        self.metadata = {}
    
    def _generate_id(self) -> str:
        return f"cmd:{int(time.time() * 1000)}:{uuid.uuid4().hex[:8]}"
    
    @abstractmethod
    def execute(self) -> Any:
        """执行命令"""
        pass
    
    @abstractmethod
    def undo(self) -> Any:
        """撤销命令"""
        pass
    
    @abstractmethod
    def redo(self) -> Any:
        """重做命令"""
        pass
    
    @abstractmethod
    def serialize(self) -> dict:
        """序列化命令"""
        pass
    
    @classmethod
    @abstractmethod
    def deserialize(cls, data: dict) -> 'Command':
        """反序列化命令"""
        pass

@dataclass
class CommandResult:
    """命令执行结果"""
    success: bool
    command_id: str
    old_state: Any = None
    new_state: Any = None
    error: Optional[str] = None

class CreateElementCommand(Command):
    """创建元素命令"""
    
    def __init__(self, user_id: str, project_id: str, element_type: str, 
                 properties: dict, position: dict):
        super().__init__(user_id, project_id)
        self.element_type = element_type
        self.properties = properties
        self.position = position
        self.created_element_id = None
        self.old_state = None
    
    def execute(self) -> CommandResult:
        """执行创建"""
        try:
            # 保存旧状态（用于撤销）
            self.old_state = project.get_state()
            
            # 创建元素
            element = project.create_element(
                element_type=self.element_type,
                properties=self.properties,
                position=self.position
            )
            
            self.created_element_id = element.id
            self.is_executed = True
            
            return CommandResult(
                success=True,
                command_id=self.id,
                old_state=self.old_state,
                new_state=project.get_state()
            )
        except Exception as e:
            return CommandResult(
                success=False,
                command_id=self.id,
                error=str(e)
            )
    
    def undo(self) -> CommandResult:
        """撤销创建"""
        try:
            # 删除创建的元素
            project.delete_element(self.created_element_id)
            
            return CommandResult(
                success=True,
                command_id=self.id,
                old_state=project.get_state(),
                new_state=self.old_state
            )
        except Exception as e:
            return CommandResult(
                success=False,
                command_id=self.id,
                error=str(e)
            )
    
    def redo(self) -> CommandResult:
        """重做创建"""
        # 重新执行创建
        return self.execute()
    
    def serialize(self) -> dict:
        return {
            'type': 'CreateElementCommand',
            'id': self.id,
            'user_id': self.user_id,
            'project_id': self.project_id,
            'timestamp': self.timestamp,
            'element_type': self.element_type,
            'properties': self.properties,
            'position': self.position,
            'created_element_id': self.created_element_id
        }
    
    @classmethod
    def deserialize(cls, data: dict) -> 'CreateElementCommand':
        cmd = cls(
            user_id=data['user_id'],
            project_id=data['project_id'],
            element_type=data['element_type'],
            properties=data['properties'],
            position=data['position']
        )
        cmd.id = data['id']
        cmd.timestamp = data['timestamp']
        cmd.created_element_id = data.get('created_element_id')
        return cmd

class UpdateElementCommand(Command):
    """更新元素命令"""
    
    def __init__(self, user_id: str, project_id: str, element_id: str,
                 property_changes: dict):
        super().__init__(user_id, project_id)
        self.element_id = element_id
        self.property_changes = property_changes
        self.old_properties = None
    
    def execute(self) -> CommandResult:
        """执行更新"""
        element = project.get_element(self.element_id)
        
        # 保存旧属性
        self.old_properties = {
            k: element.get_property(k) 
            for k in self.property_changes.keys()
        }
        
        # 应用更新
        for key, value in self.property_changes.items():
            element.set_property(key, value)
        
        self.is_executed = True
        
        return CommandResult(
            success=True,
            command_id=self.id,
            old_state=self.old_properties,
            new_state=self.property_changes
        )
    
    def undo(self) -> CommandResult:
        """撤销更新"""
        element = project.get_element(self.element_id)
        
        # 恢复旧属性
        for key, value in self.old_properties.items():
            element.set_property(key, value)
        
        return CommandResult(
            success=True,
            command_id=self.id,
            old_state=self.property_changes,
            new_state=self.old_properties
        )
    
    def redo(self) -> CommandResult:
        """重做更新"""
        return self.execute()
    
    def serialize(self) -> dict:
        return {
            'type': 'UpdateElementCommand',
            'id': self.id,
            'user_id': self.user_id,
            'project_id': self.project_id,
            'timestamp': self.timestamp,
            'element_id': self.element_id,
            'property_changes': self.property_changes,
            'old_properties': self.old_properties
        }

class CompositeCommand(Command):
    """组合命令（宏命令）"""
    
    def __init__(self, user_id: str, project_id: str, name: str):
        super().__init__(user_id, project_id)
        self.name = name
        self.commands: List[Command] = []
    
    def add_command(self, command: Command):
        """添加子命令"""
        self.commands.append(command)
    
    def execute(self) -> CommandResult:
        """执行所有子命令"""
        results = []
        
        for cmd in self.commands:
            result = cmd.execute()
            results.append(result)
            
            if not result.success:
                # 某个命令失败，回滚已执行的命令
                self._rollback_executed(results)
                return CommandResult(
                    success=False,
                    command_id=self.id,
                    error=f"Sub-command {cmd.id} failed: {result.error}"
                )
        
        self.is_executed = True
        
        return CommandResult(
            success=True,
            command_id=self.id
        )
    
    def undo(self) -> CommandResult:
        """撤销所有子命令（逆序）"""
        for cmd in reversed(self.commands):
            cmd.undo()
        
        return CommandResult(success=True, command_id=self.id)
    
    def redo(self) -> CommandResult:
        """重做所有子命令"""
        return self.execute()
    
    def _rollback_executed(self, results: List[CommandResult]):
        """回滚已执行的命令"""
        for result in reversed(results):
            if result.success:
                # 找到对应的命令并撤销
                cmd = next(c for c in self.commands if c.id == result.command_id)
                cmd.undo()
```

### 7.2 操作日志设计

#### 7.2.1 操作日志结构

```python
class OperationLog:
    """操作日志管理器"""
    
    def __init__(self, storage_backend):
        self.storage = storage_backend
        self.buffer = []
        self.buffer_size = 100
        self.sequence_number = 0
    
    class LogEntry:
        """日志条目"""
        def __init__(self, command: Command, project_version: int):
            self.sequence = 0
            self.timestamp = time.time()
            self.command = command
            self.project_version = project_version
            self.user_id = command.user_id
            self.checksum = self._calculate_checksum()
        
        def _calculate_checksum(self) -> str:
            """计算校验和"""
            data = f"{self.sequence}:{self.timestamp}:{self.command.serialize()}"
            return hashlib.sha256(data.encode()).hexdigest()[:16]
        
        def to_dict(self) -> dict:
            return {
                'sequence': self.sequence,
                'timestamp': self.timestamp,
                'command': self.command.serialize(),
                'project_version': self.project_version,
                'user_id': self.user_id,
                'checksum': self.checksum
            }
    
    async def append(self, command: Command, project_version: int):
        """追加操作到日志"""
        self.sequence_number += 1
        
        entry = self.LogEntry(command, project_version)
        entry.sequence = self.sequence_number
        
        self.buffer.append(entry)
        
        # 缓冲区满，刷写到存储
        if len(self.buffer) >= self.buffer_size:
            await self._flush()
    
    async def _flush(self):
        """刷写缓冲区到存储"""
        if not self.buffer:
            return
        
        # 批量写入
        await self.storage.batch_insert(
            table='operation_log',
            data=[e.to_dict() for e in self.buffer]
        )
        
        # 清空缓冲区
        self.buffer = []
    
    async def get_operations(self, project_id: str, 
                            start_sequence: int = None,
                            end_sequence: int = None,
                            user_id: str = None) -> List[LogEntry]:
        """获取操作日志"""
        query = {'project_id': project_id}
        
        if start_sequence:
            query['sequence'] = {'$gte': start_sequence}
        if end_sequence:
            query['sequence'] = {'$lte': end_sequence}
        if user_id:
            query['user_id'] = user_id
        
        results = await self.storage.find('operation_log', query)
        
        return [self._deserialize_entry(r) for r in results]
    
    async def get_operations_by_time(self, project_id: str,
                                     start_time: float,
                                     end_time: float) -> List[LogEntry]:
        """按时间范围获取操作"""
        query = {
            'project_id': project_id,
            'timestamp': {'$gte': start_time, '$lte': end_time}
        }
        
        results = await self.storage.find('operation_log', query)
        return [self._deserialize_entry(r) for r in results]
```

### 7.3 跨用户撤销的边界处理

#### 7.3.1 撤销边界策略

```python
class UndoBoundaryManager:
    """撤销边界管理器"""
    
    def __init__(self):
        self.boundary_policies = {
            'strict': StrictBoundaryPolicy(),
            'per_user': PerUserBoundaryPolicy(),
            'time_window': TimeWindowBoundaryPolicy(),
            'group': GroupBoundaryPolicy()
        }
    
    class UndoContext:
        """撤销上下文"""
        def __init__(self, user_id: str, project_id: str, policy: str = 'per_user'):
            self.user_id = user_id
            self.project_id = project_id
            self.policy = policy
            self.undo_stack = []
            self.redo_stack = []
    
    def can_undo(self, context: UndoContext) -> bool:
        """检查是否可以撤销"""
        policy = self.boundary_policies[context.policy]
        return policy.can_undo(context)
    
    def get_undo_target(self, context: UndoContext) -> Optional[Command]:
        """获取撤销目标命令"""
        policy = self.boundary_policies[context.policy]
        return policy.get_undo_target(context)

class PerUserBoundaryPolicy:
    """每用户独立撤销边界"""
    
    def can_undo(self, context):
        """用户只能撤销自己的操作"""
        for cmd in reversed(context.undo_stack):
            if cmd.user_id == context.user_id:
                return True
        return False
    
    def get_undo_target(self, context):
        """获取用户可撤销的命令"""
        for cmd in reversed(context.undo_stack):
            if cmd.user_id == context.user_id:
                return cmd
        return None

class TimeWindowBoundaryPolicy:
    """时间窗口撤销边界"""
    
    def __init__(self, window_seconds: float = 300):  # 默认5分钟
        self.window_seconds = window_seconds
    
    def can_undo(self, context):
        """只能撤销最近N分钟内的操作"""
        cutoff_time = time.time() - self.window_seconds
        
        for cmd in reversed(context.undo_stack):
            if cmd.user_id == context.user_id and cmd.timestamp >= cutoff_time:
                return True
        return False
    
    def get_undo_target(self, context):
        """获取时间窗口内的命令"""
        cutoff_time = time.time() - self.window_seconds
        
        for cmd in reversed(context.undo_stack):
            if cmd.user_id == context.user_id and cmd.timestamp >= cutoff_time:
                return cmd
        return None

class GroupBoundaryPolicy:
    """分组撤销边界（事务式）"""
    
    def can_undo(self, context):
        """只能按组撤销"""
        if not context.undo_stack:
            return False
        
        # 获取最后一个命令组
        last_cmd = context.undo_stack[-1]
        return last_cmd.user_id == context.user_id
    
    def get_undo_target(self, context):
        """获取整个命令组"""
        if not context.undo_stack:
            return None
        
        last_cmd = context.undo_stack[-1]
        
        if last_cmd.user_id != context.user_id:
            return None
        
        # 如果是组合命令，返回整个组合
        if isinstance(last_cmd, CompositeCommand):
            return last_cmd
        
        return last_cmd
```

### 7.4 历史数据存储优化

#### 7.4.1 存储优化策略

```python
class HistoryStorageOptimizer:
    """历史数据存储优化器"""
    
    def __init__(self, storage):
        self.storage = storage
        self.compression_enabled = True
        self.snapshot_interval = 100  # 每100个操作创建一个快照
    
    async def optimize_storage(self, project_id: str):
        """优化项目历史存储"""
        # 1. 压缩旧操作日志
        await self._compress_old_logs(project_id)
        
        # 2. 创建快照点
        await self._create_snapshots(project_id)
        
        # 3. 清理冗余历史
        await self._cleanup_redundant_history(project_id)
        
        # 4. 归档冷数据
        await self._archive_cold_data(project_id)
    
    async def _compress_old_logs(self, project_id: str):
        """压缩旧的操作日志"""
        # 获取超过30天的操作
        cutoff_time = time.time() - (30 * 24 * 3600)
        
        old_ops = await self.storage.find(
            'operation_log',
            {
                'project_id': project_id,
                'timestamp': {'$lt': cutoff_time},
                'compressed': {'$ne': True}
            }
        )
        
        # 批量压缩
        compressed_batch = []
        for op in old_ops:
            compressed = self._compress_operation(op)
            compressed['compressed'] = True
            compressed_batch.append(compressed)
        
        if compressed_batch:
            await self.storage.batch_update('operation_log', compressed_batch)
    
    async def _create_snapshots(self, project_id: str):
        """创建快照点"""
        # 获取最新快照
        last_snapshot = await self.storage.find_one(
            'project_snapshots',
            {'project_id': project_id},
            sort=[('sequence', -1)]
        )
        
        last_sequence = last_snapshot['sequence'] if last_snapshot else 0
        
        # 获取需要快照的操作
        ops = await self.storage.find(
            'operation_log',
            {
                'project_id': project_id,
                'sequence': {'$gt': last_sequence}
            },
            sort=[('sequence', 1)]
        )
        
        # 每N个操作创建一个快照
        for i in range(0, len(ops), self.snapshot_interval):
            batch = ops[i:i + self.snapshot_interval]
            
            # 应用操作生成新状态
            state = last_snapshot['state'] if last_snapshot else {}
            for op in batch:
                state = self._apply_operation(state, op)
            
            # 保存快照
            snapshot = {
                'project_id': project_id,
                'sequence': batch[-1]['sequence'],
                'state': state,
                'created_at': time.time()
            }
            
            await self.storage.insert('project_snapshots', snapshot)
    
    async def _cleanup_redundant_history(self, project_id: str):
        """清理冗余历史"""
        # 获取所有快照点
        snapshots = await self.storage.find(
            'project_snapshots',
            {'project_id': project_id},
            sort=[('sequence', 1)]
        )
        
        # 保留最近的快照和关键快照
        snapshots_to_keep = set()
        
        # 保留最新的5个快照
        for snap in snapshots[-5:]:
            snapshots_to_keep.add(snap['sequence'])
        
        # 保留每1000个操作的快照
        for snap in snapshots:
            if snap['sequence'] % 1000 == 0:
                snapshots_to_keep.add(snap['sequence'])
        
        # 删除不需要的快照之间的操作日志
        for i in range(len(snapshots) - 1):
            if snapshots[i]['sequence'] not in snapshots_to_keep:
                await self.storage.delete_many(
                    'operation_log',
                    {
                        'project_id': project_id,
                        'sequence': {
                            '$gt': snapshots[i]['sequence'],
                            '$lte': snapshots[i + 1]['sequence']
                        }
                    }
                )
    
    def _compress_operation(self, operation: dict) -> dict:
        """压缩单个操作"""
        # 使用JSON压缩
        json_str = json.dumps(operation)
        compressed = zlib.compress(json_str.encode(), level=9)
        
        return {
            'compressed_data': base64.b64encode(compressed).decode(),
            'original_size': len(json_str),
            'compressed_size': len(compressed)
        }
```

---

## 8. 技术选型建议

### 8.1 推荐技术栈

| 技术领域 | 推荐方案 | 备选方案 |
|---------|---------|---------|
| **实时协作引擎** | Yjs/Automerge (CRDT) | ShareDB (OT) |
| **并发控制** | 乐观锁 + MVCC | 悲观锁 |
| **一致性协议** | 因果一致性 | 强一致（关键操作） |
| **消息队列** | Redis Pub/Sub + Kafka | RabbitMQ |
| **WebSocket** | Socket.io | 原生WebSocket |
| **存储** | PostgreSQL + Redis | MongoDB |
| **任务调度** | Celery + Redis | Apache Airflow |

### 8.2 架构建议

```
┌─────────────────────────────────────────────────────────────────────┐
│                    建筑设计平台并发协作架构                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │   Web App    │  │  Desktop App │  │  Mobile App  │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                      │
│         └──────────────────┼──────────────────┘                      │
│                            ▼                                         │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                      API Gateway                             │   │
│  │              (Rate Limiting, Authentication)                 │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                            │                                         │
│         ┌──────────────────┼──────────────────┐                      │
│         ▼                  ▼                  ▼                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  REST API    │  │ WebSocket    │  │  GraphQL     │              │
│  │  (CRUD)      │  │ (Real-time)  │  │  (Query)     │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                      │
│         └──────────────────┼──────────────────┘                      │
│                            ▼                                         │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                   Microservices Layer                        │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │   │
│  │  │ Project  │ │Collabor- │ │  Script  │ │ Conflict │       │   │
│  │  │ Service  │ │ ation    │ │ Execution│ │ Resolution│       │   │
│  │  │          │ │ Service  │ │ Service  │ │ Service  │       │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                            │                                         │
│         ┌──────────────────┼──────────────────┐                      │
│         ▼                  ▼                  ▼                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │   PostgreSQL │  │    Redis     │  │    Kafka     │              │
│  │   (Primary)  │  │   (Cache/    │  │  (Event      │              │
│  │              │  │   Session)   │  │   Stream)    │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 8.3 性能目标

| 指标 | 目标值 | 说明 |
|-----|-------|------|
| 操作延迟 | < 50ms | 本地操作到远程同步 |
| 并发用户 | > 100 | 同一项目同时编辑 |
| 冲突解决 | < 200ms | 自动冲突检测和解决 |
| 撤销响应 | < 100ms | 撤销操作响应时间 |
| 断线恢复 | < 2s | 重连后状态同步 |

---

## 9. 架构示意图

### 9.1 整体并发协作架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         并发协作系统整体架构                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                        客户端层                                      │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│   │  │  Web     │  │ Desktop  │  │  Mobile  │  │   CAD    │           │   │
│   │  │  Client  │  │  Client  │  │  Client  │  │  Plugin  │           │   │
│   │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘           │   │
│   └───────┼─────────────┼─────────────┼─────────────┼─────────────────┘   │
│           │             │             │             │                        │
│           └─────────────┴─────────────┴─────────────┘                        │
│                         │                                                    │
│                         ▼                                                    │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      接入网关层                                      │   │
│   │  ┌─────────────────────────────────────────────────────────────┐   │   │
│   │  │  Load Balancer  │  Auth Gateway  │  Rate Limiter  │  WAF   │   │   │
│   │  └─────────────────────────────────────────────────────────────┘   │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                     │                                        │
│           ┌─────────────────────────┼─────────────────────────┐              │
│           ▼                         ▼                         ▼              │
│   ┌──────────────┐          ┌──────────────┐          ┌──────────────┐      │
│   │  REST API    │          │  WebSocket   │          │  GraphQL     │      │
│   │  Service     │          │  Service     │          │  Service     │      │
│   └──────┬───────┘          └──────┬───────┘          └──────┬───────┘      │
│          │                         │                         │              │
│          └─────────────────────────┼─────────────────────────┘              │
│                                    ▼                                         │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      业务服务层                                      │   │
│   │                                                                     │   │
│   │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │   │
│   │  │   Project    │  │ Collaboration│  │    Script    │              │   │
│   │  │   Service    │  │   Service    │  │   Engine     │              │   │
│   │  │              │  │              │  │              │              │   │
│   │  │ - CRUD       │  │ - CRDT/OT    │  │ - Task Queue │              │   │
│   │  │ - Versioning │  │ - Broadcast  │  │ - Execution  │              │   │
│   │  │ - Permission │  │ - Conflict   │  │ - Scheduling │              │   │
│   │  └──────────────┘  └──────────────┘  └──────────────┘              │   │
│   │                                                                     │   │
│   │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │   │
│   │  │   Conflict   │  │    Undo/     │  │ Notification │              │   │
│   │  │  Resolution  │  │    Redo      │  │   Service    │              │   │
│   │  │   Service    │  │   Service    │  │              │              │   │
│   │  │              │  │              │  │              │              │   │
│   │  │ - Detect     │  │ - Command    │  │ - Real-time  │              │   │
│   │  │ - Auto-merge │  │ - History    │  │ - Email      │              │   │
│   │  │ - Manual UI  │  │ - Snapshot   │  │ - Push       │              │   │
│   │  └──────────────┘  └──────────────┘  └──────────────┘              │   │
│   │                                                                     │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                     │                                        │
│           ┌─────────────────────────┼─────────────────────────┐              │
│           ▼                         ▼                         ▼              │
│   ┌──────────────┐          ┌──────────────┐          ┌──────────────┐      │
│   │  PostgreSQL  │          │    Redis     │          │    Kafka     │      │
│   │  (Primary)   │          │   (Cache)    │          │   (Events)   │      │
│   │              │          │              │          │              │      │
│   │ - Projects   │          │ - Sessions   │          │ - Operations │      │
│   │ - Elements   │          │ - Locks      │          │ - Notifications│    │
│   │ - History    │          │ - Rate Limit │          │ - Tasks      │      │
│   └──────────────┘          └──────────────┘          └──────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 9.2 协作服务详细架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         协作服务详细架构                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                     Collaboration Service                            │   │
│   │                                                                     │   │
│   │  ┌─────────────────────────────────────────────────────────────┐   │   │
│   │  │                    Connection Manager                        │   │   │
│   │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │   │   │
│   │  │  │  Client  │  │  Client  │  │  Client  │  │  Client  │   │   │   │
│   │  │  │  Conn 1  │  │  Conn 2  │  │  Conn 3  │  │  Conn N  │   │   │   │
│   │  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │   │   │
│   │  └─────────────────────────────────────────────────────────────┘   │   │
│   │                              │                                       │   │
│   │                              ▼                                       │   │
│   │  ┌─────────────────────────────────────────────────────────────┐   │   │
│   │  │                    Operation Processor                       │   │   │
│   │  │                                                              │   │   │
│   │  │  ┌──────────────┐      ┌──────────────┐      ┌──────────┐   │   │   │
│   │  │  │   Parser     │─────►│  Validator   │─────►│  Router  │   │   │   │
│   │  │  └──────────────┘      └──────────────┘      └────┬─────┘   │   │   │
│   │  │                                                   │         │   │   │
│   │  │                              ┌────────────────────┘         │   │   │
│   │  │                              ▼                               │   │   │
│   │  │  ┌──────────────┐      ┌──────────────┐      ┌──────────┐   │   │   │
│   │  │  │  CRDT Engine │◄────►│  OT Engine   │◄────►│  Merger  │   │   │   │
│   │  │  └──────────────┘      └──────────────┘      └──────────┘   │   │   │
│   │  └─────────────────────────────────────────────────────────────┘   │   │
│   │                              │                                       │   │
│   │                              ▼                                       │   │
│   │  ┌─────────────────────────────────────────────────────────────┐   │   │
│   │  │                    Broadcast Manager                         │   │   │
│   │  │                                                              │   │   │
│   │  │  ┌──────────────┐      ┌──────────────┐      ┌──────────┐   │   │   │
│   │  │  │   Filter     │─────►│  Compressor  │─────►│  Sender  │   │   │   │
│   │  │  └──────────────┘      └──────────────┘      └────┬─────┘   │   │   │
│   │  │                                                   │         │   │   │
│   │  │                              ┌────────────────────┘         │   │   │
│   │  │                              ▼                               │   │   │
│   │  │  ┌──────────────┐      ┌──────────────┐                     │   │   │
│   │  │  │  Redis Pub   │      │  Kafka Topic │                     │   │   │
│   │  │  └──────────────┘      └──────────────┘                     │   │   │
│   │  └─────────────────────────────────────────────────────────────┘   │   │
│   │                                                                     │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. 参考资料

### 10.1 学术论文

1. Ellis, C.A., & Gibbs, S.J. (1989). "Concurrency Control in Groupware Systems"
2. Shapiro, M., Preguiça, N., Baquero, C., & Zawirski, M. (2011). "Conflict-free Replicated Data Types"
3. Ongaro, D., & Ousterhout, J. (2014). "In Search of an Understandable Consensus Algorithm"
4. Gilbert, S., & Lynch, N. (2002). "Brewer's Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web Services"

### 10.2 开源项目

1. **Yjs** - CRDT实现库 (https://github.com/yjs/yjs)
2. **Automerge** - 协作数据类型 (https://github.com/automerge/automerge)
3. **ShareDB** - OT实现 (https://github.com/share/sharedb)
4. **etcd** - Raft实现 (https://github.com/etcd-io/etcd)

### 10.3 技术文章

1. CRDT Implementation Guide - https://velt.dev/blog/crdt-implementation-guide
2. OT vs CRDT Comparison - https://blog.csdn.net/2401_89241768/article/details/154279916
3. MVCC in PostgreSQL - https://minervadb.xyz/demystifying-postgresql-mvcc
4. CAP Theorem - https://www.hellointerview.com/learn/system-design/core-concepts/cap-theorem

---

**报告完成**

*本报告为半自动化建筑设计平台并发与协作技术调研成果，供技术可行性评审使用。*
