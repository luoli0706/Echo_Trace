# Maze / Map 评审与改进方向（面向 Beta）

> **范围**：服务端地图生成与碰撞基础（GameMap / Maze）  
> **对应版本**：Echo Trace Alpha 0.5 → Beta 1.0

---

## 1. 当前实现：`maze.go` 的真实语义

相关代码：
- [backend/logic/maze.go](../backend/logic/maze.go)
- [backend/logic/gamestate.go](../backend/logic/gamestate.go)（spawn、碰撞）

### 1.1 地图生成

`NewGameMap(width,height,density)` 当前并非“迷宫算法（Prim）”，而是：

- 边界强制为墙
- 内部格子以 `rand.Float64() < density` 随机生成墙

结论：这是 **随机障碍地图**，不是“保证连通/走廊结构”的迷宫。

### 1.2 碰撞与可行走判定

`IsWalkable(x,y)`：将 float 坐标取整到 tile，检查是否为 TileEmpty。

### 1.3 出生点

`GetRandomSpawnPos()`：在非边界区域随机找一个空地 tile（死循环直到找到）。

---

## 2. 主要问题与风险

### 2.1 体验与玩法层面

1. **不保证连通性**  
   - 随机墙很可能形成封闭区域/孤岛，导致：
     - 玩家出生在不可达区域
     - 电机/撤离点/商人/空投可能与玩家隔绝
   - 这会直接破坏“搜刮→冲突→撤离”的核心 loop。

2. **与设计文档不一致**  
   - README/蓝图提到 Prim 迷宫，但实际实现不是。
   - 会导致后续调参/扩展（AOI、LOS、NavMesh）缺乏可预测结构。

### 2.2 稳定性

1. **spawn 可能长时间循环**  
   - 当 density 较高或地图较小，`GetRandomSpawnPos` 可能循环很久（极端情况近似卡死）。

2. **缺少“出生点最小距离”**  
   - 可能出现多玩家贴脸出生，导致不符合预期的开局体验。

### 2.3 可维护性/演进

1. **地图缺少元数据**  
   - 没有房间/走廊/关键点的结构信息，后续做：
     - AI 寻路（NavMesh）
     - LOS（遮挡）
     - 资源投放策略
     都会更难。

---

## 3. 改进方向

### 3.1 Direction A：保证连通性（Beta 必做）

最小可行方案（P0/P1）：

- 地图生成后做一次 flood fill/ BFS，从任意空地出发标记可达区域。
- 若可达空地比例低于阈值（例如 < 60%），则重新生成。
- 出生点、关键实体（商人/电机/撤离点/空投）只从“最大连通分量”中采样。

收益：立刻消除“不可达/卡局”的核心风险。

### 3.2 Direction B：实现真正迷宫（Prim/DFS）或“房间+走廊”生成

- 若希望强迷宫体验：实现 Prim/DFS 生成完美迷宫，再按配置加少量破墙形成环路。
- 若希望对战更可控：实现“rooms + corridors”并允许配置房间数量/大小。

### 3.3 Direction C：Spawn 策略升级

- 增加：
  - `GetRandomSpawnPosWithMinDistance(existing[]pos, minDist)`
  - `GetSpawnPosNear(center, radius)`（用于空投/商人策略）
- 把“重要点位”投放转为可配置策略（与 beta 方案的平衡调优一致）。

### 3.4 Direction D：为 LOS/NavMesh 做准备（P2/P3）

- 生成并缓存：
  - 可行走 tile 集合
  - 距离场/热点（可用于资源投放与环境威胁）
  - 简易 nav graph（或 navmesh 前置）

---

## 4. 任务拆分与优先级

### P0：关键缺陷修复

1. `GetRandomSpawnPos` 增加最大尝试次数与失败回退（避免死循环）
2. 关键实体投放只在连通区域（或最大连通分量）内

### P1：核心体验优化

1. 加入连通性检测与重生成
2. 出生点最小距离约束

### P2：重要功能增强

1. 引入 Prim/DFS 或 rooms+corridors 生成器（与设计文档一致）
2. 为 LOS/NavMesh 提供必要的 map 元数据

---

## 5. 验收指标（Success Metrics）

- **可玩性**：任意玩家出生点到商人/电机/撤离点至少存在一条路径
- **稳定性**：地图生成与 spawn 无长时间卡顿（有上限）
- **体验**：开局出生点满足最小距离，减少贴脸遭遇
