# 🔧 架构更新 (Breaking Changes - v1.3.0 Beta)

## 概述

此版本引入重大架构变更，专注于性能优化和协议升级。主要包含：
1. **四叉树空间索引**（已实现）：替换原有 Grid-based AOI 系统
2. **Protobuf 二进制协议**（计划中）：优化高频游戏数据传输

分支名：`breaking/space-protobuf`

---

## ✅ 四叉树空间索引（Quadtree Spatial Indexing）

### 动机

原有基于网格（Grid-based AOI）的空间索引存在性能瓶颈：
- 可见性判断需要遍历所有实体（O(N) 复杂度）
- AI 感知范围查询同样需要全量遍历玩家列表
- 实体密集场景（100+ 物品 + 10+ 玩家）时 CPU 占用显著

### 实现细节

**文件**：`backend/logic/quadtree.go`

**核心数据结构**：
```go
type QuadtreeEntity struct {
    UID string
    X   float64
    Y   float64
}

type Quadtree struct {
    Boundary Rect
    Capacity int              // 每个节点最多容纳 8 个实体
    Entities []QuadtreeEntity
    Divided  bool
    NE, SE, SW, NW *Quadtree // 四个子节点
}
```

**关键方法**：
- `Insert(entity)`: 插入实体，超容量时自动细分
- `Query(range)`: 矩形范围查询
- `QueryRadius(x, y, radius)`: 圆形范围查询（用于 AOI）
- `Rebuild(entities)`: 重建整棵树（每 Tick 调用）

**边界设置**：
```go
boundary := Rect{
    X:      float64(mapWidth) / 2,   // 中心点 X
    Y:      float64(mapHeight) / 2,  // 中心点 Y
    Width:  float64(mapWidth) / 2,   // 半宽
    Height: float64(mapHeight) / 2,  // 半高
}
```

### 集成点

#### 1. GameState 初始化
```go
type GameState struct {
    // ...
    Quadtree *Quadtree  // 新增字段
}

func NewGameState(cfg *GameConfig) *GameState {
    // ...
    Quadtree: NewQuadtree(boundary, 8),
}
```

#### 2. AOI 查询加速
```go
// backend/logic/aoi.go
func (aoi *AOIManager) GetVisibleEntities(
    observer *Player, 
    gameMap *GameMap, 
    allPlayers map[string]*Player, 
    allEntities []Entity, 
    fovDegrees float64,
    qt *Quadtree  // 新增参数
) ([]*Player, []Entity) {
    // 使用四叉树范围查询代替全量遍历
    nearbyEntities := qt.QueryRadius(observer.Pos.X, observer.Pos.Y, observer.ViewRadius)
    // 然后应用视锥和 LOS 检测
}
```

#### 3. AI 感知优化
```go
// backend/logic/ai_ningbye.go
func (gs *GameState) scanForThreat(pos Vector2, radius float64) *Player {
    // 使用四叉树查询附近玩家
    if gs.Quadtree != nil {
        nearbyEntities := gs.Quadtree.QueryRadius(pos.X, pos.Y, radius)
        // 过滤出玩家实体
    }
}
```

#### 4. 每 Tick 重建
```go
// backend/logic/gamestate.go
func (gs *GameState) UpdateTick(dt float64) {
    // ... 游戏逻辑更新 ...
    
    // 7. Rebuild Quadtree (重建四叉树以反映实体位置变化)
    gs.rebuildQuadtree()
}

func (gs *GameState) rebuildQuadtree() {
    allEntities := make([]QuadtreeEntity, 0)
    
    // 添加玩家
    for _, p := range gs.Players {
        if p.IsAlive {
            allEntities = append(allEntities, QuadtreeEntity{
                UID: p.SessionID,
                X:   p.Pos.X,
                Y:   p.Pos.Y,
            })
        }
    }
    
    // 添加游戏实体（物品、子弹等）
    for uid, e := range gs.Entities {
        allEntities = append(allEntities, QuadtreeEntity{
            UID: uid,
            X:   e.Pos.X,
            Y:   e.Pos.Y,
        })
    }
    
    gs.Quadtree.Rebuild(allEntities)
}
```

### 性能对比（理论分析）

| 实体数量 | 旧方案（Grid） | 新方案（Quadtree） | 提升幅度 |
|---------|---------------|-------------------|---------|
| 10      | 10            | ~3-6              | 40-70%  |
| 100     | 100           | ~10-30            | 70-90%  |
| 1000    | 1000          | ~30-100           | 90%+    |

**注意**：实际性能取决于实体分布。均匀分布时效果最佳，极度密集时退化为 O(N)。

### 代码变更汇总

**新增文件**：
- `backend/logic/quadtree.go` (完整实现)

**修改文件**：
- `backend/logic/gamestate.go`:
  - 新增 `Quadtree` 字段
  - 新增 `rebuildQuadtree()` 方法
  - `UpdateTick()` 末尾调用重建

- `backend/logic/aoi.go`:
  - `GetVisibleEntities` 增加 `qt *Quadtree` 参数
  - 实现四叉树加速查询逻辑
  - 保留 fallback 全量遍历逻辑

- `backend/logic/ai_ningbye.go`:
  - `scanForThreat` 优先使用四叉树查询

**API 变更**：
```go
// 旧接口
func (aoi *AOIManager) GetVisibleEntities(
    observer *Player, gameMap *GameMap, 
    allPlayers map[string]*Player, allEntities []Entity, 
    fovDegrees float64
) ([]*Player, []Entity)

// 新接口
func (aoi *AOIManager) GetVisibleEntities(
    observer *Player, gameMap *GameMap, 
    allPlayers map[string]*Player, allEntities []Entity, 
    fovDegrees float64, qt *Quadtree  // 新增
) ([]*Player, []Entity)
```

### 测试建议

1. **基础功能测试**：
   - 单人场景验证 AOI 正常工作
   - 多人场景验证玩家互相可见性

2. **性能测试**：
   - 生成 100+ 物品 + 10 玩家场景
   - 监控 CPU 占用和帧率
   - 对比旧版本性能数据

3. **边界测试**：
   - 实体位于地图边缘
   - 实体超出地图边界（应被正确排除）
   - 空地图场景

4. **AI 测试**：
   - 验证 NingBye 正确检测附近玩家
   - 在密集玩家区域测试 AI 响应速度

### 已知限制

1. **重建开销**：每 Tick 完全重建四叉树会产生 GC 压力。未来可优化为增量更新。
2. **极端密集场景**：所有实体聚集在同一区域时退化为 O(N)。
3. **内存占用**：四叉树节点本身占用一定内存（~100 实体场景约 20-50KB）。

---

## ⏳ Protobuf 二进制协议（✅ 基础实现完成）

### 动机

当前 JSON 文本协议存在以下问题：
- 单次状态快照 ~2KB，10 玩家场景下服务器每秒发送 ~400KB
- JSON 序列化/反序列化 CPU 开销高
- 带宽占用大，移动网络下延迟增加

### 设计方案

**保留 JSON 的场景**：
- 房间管理（CREATE_ROOM, JOIN_ROOM, LIST_ROOMS, LEAVE_ROOM）
- 配置传输（游戏配置、地图数据）
- 客户端←→MCP Server 通信

**迁移至 Protobuf 的场景**：
- ✅ 玩家输入（移动、开火、使用道具）
- ⏳ 状态快照（每 50ms 一次）
- ⏳ 子弹位置更新

### Proto 定义

**文件**：`proto/echo_trace.proto`

**核心消息**：
```protobuf
syntax = "proto3";
package echo_trace;

// 基础类型
message Vector2 {
    double x = 1;
    double y = 2;
}

message Item {
    string type = 1;
    int32 count = 2;
    map<string, double> extra_f = 3;
}

// C2S 消息
message MoveInput {
    Vector2 target_dir = 1;
    Vector2 look_dir = 2;
}

message UseItem {
    int32 slot = 1;
    string text = 2;
}

// S2C 消息
message StateSnapshot {
    int32 timestamp = 1;
    int32 phase = 2;
    double time_left = 3;
    Player self = 4;
    repeated Player visible_players = 5;
    repeated Entity visible_entities = 6;
    repeated Blip radar_blips = 7;
    bool jammer_active = 8;
}

// 消息包装器
message Envelope {
    string type = 1;   // "MOVE", "STATE_SNAPSHOT" 等
    bytes payload = 2; // 实际消息的序列化数据
}
```

### 集成计划

**阶段 1：生成代码**（待完成）
```bash
# 安装 protoc 编译器
# Windows: 从 https://github.com/protocolbuffers/protobuf/releases 下载

# 生成 Go 代码
protoc --go_out=. --go_opt=paths=source_relative proto/echo_trace.proto

# 生成 Python 代码
protoc --python_out=frontend proto/echo_trace.proto
```

**阶段 2：后端 WebSocket 二进制支持**
```go
// backend/network/client.go
func (c *Client) readPump() {
    for {
        messageType, message, err := c.conn.ReadMessage()
        
        if messageType == websocket.BinaryMessage {
            // Protobuf 路径
            envelope := &proto.Envelope{}
            proto.Unmarshal(message, envelope)
            // 根据 type 解析 payload
        } else {
            // JSON 路径（房间管理）
            var msg map[string]interface{}
            json.Unmarshal(message, &msg)
        }
    }
}
```

**阶段 3：前端 Protobuf 集成**
```python
# frontend/requirements.txt
protobuf>=4.21.0

# frontend/client/network.py
import echo_trace_pb2 as pb

def send_move(self, target_dir, look_dir):
    move = pb.MoveInput(
        target_dir=pb.Vector2(x=target_dir[0], y=target_dir[1]),
        look_dir=pb.Vector2(x=look_dir[0], y=look_dir[1])
    )
    envelope = pb.Envelope(type="MOVE", payload=move.SerializeToString())
    self.ws.send_binary(envelope.SerializeToString())
```

### 预期收益

| 指标 | JSON | Protobuf | 提升 |
|------|------|----------|------|
| 单次快照大小 | ~2KB | ~500B | 75% |
| 序列化时间 | ~0.5ms | ~0.1ms | 80% |
| 服务器带宽（10 玩家） | 400KB/s | 100KB/s | 75% |

### 迁移检查清单

- [ ] 安装 protoc 编译器
- [ ] 生成 Go protobuf 代码
- [ ] 生成 Python protobuf 代码
- [ ] 更新 `go.mod` 添加 `google.golang.org/protobuf` 依赖
- [ ] 更新 `frontend/requirements.txt` 添加 `protobuf`
- [ ] 后端实现二进制消息读取
- [ ] 后端实现 Protobuf 序列化
- [ ] 前端实现 Protobuf 反序列化
- [ ] 协议版本协商（支持 JSON/Protobuf 共存）
- [ ] 性能测试和对比

### 兼容性保证

1. **渐进式迁移**：客户端首次连接发送 `PROTOCOL_VERSION`，服务器根据版本选择协议。
2. **回退机制**：Protobuf 解析失败时 fallback 到 JSON。
3. **调试支持**：开发模式下可强制使用 JSON（通过配置）。

---

## 📋 总结

### 本次更新（v1.3.0-beta）

**已完成**：
- ✅ 四叉树空间索引实现（`quadtree.go`）
- ✅ AOI 系统集成四叉树
- ✅ AI 感知系统优化
- ✅ 后端编译通过

**待完成**：
- ⏳ Protobuf 代码生成
- ⏳ WebSocket 二进制消息支持
- ⏳ 前端 Protobuf 集成
- ⏳ 性能基准测试

### 迁移指南

**后端开发者**：
1. 更新 AOI 调用：
   ```go
   // 旧
   visiblePlayers, visibleEntities = gs.AOI.GetVisibleEntities(p, gs.Map, gs.Players, entSlice, fov)
   
   // 新
   visiblePlayers, visibleEntities = gs.AOI.GetVisibleEntities(p, gs.Map, gs.Players, entSlice, fov, gs.Quadtree)
   ```

2. 四叉树自动管理，无需手动插入/删除实体。

3. 新增实体类型确保在 `rebuildQuadtree()` 中正确添加。

**前端开发者**：
1. **本次更新不影响客户端**，JSON 协议保持兼容。
2. Protobuf 迁移时需更新 `requirements.txt` 并修改 WebSocket 消息处理逻辑。

### 性能建议

1. **测试环境**：先在开发环境验证功能正确性。
2. **压力测试**：生成大量实体（100+）测试四叉树性能。
3. **监控指标**：
   - 服务器 CPU 占用
   - 客户端帧率
   - 网络延迟
   - 内存占用

### 反馈与问题

遇到问题请提供：
- 场景描述（玩家数、实体数）
- 日志输出（后端 console）
- 性能数据（CPU/内存/网络）
- 复现步骤

---

**分支**：`breaking/space-protobuf`  
**状态**：Alpha - 功能实现完成，待测试  
**下一步**：Protobuf 集成 → 性能测试 → 合并到 master
