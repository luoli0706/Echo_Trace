# AOI 系统评审与改进方向（面向 Beta）

> **范围**：服务端 AOI（视野过滤/兴趣管理）  
> **对应版本**：Echo Trace Alpha 0.5 → Beta 1.0  
> **目标对齐**：与 [docs/beta_iteration_plan.md](beta_iteration_plan.md) 的 **性能 / 稳定性 / 可维护性**目标一致

---

## 1. 当前实现：`aoi.go` 的真实语义

相关代码：
- [backend/logic/aoi.go](../backend/logic/aoi.go)
- [backend/logic/gamestate.go（GetSnapshot）](../backend/logic/gamestate.go)

### 1.1 `AOIManager` 结构与职责

`AOIManager` 当前只保存 `MapWidth/MapHeight`，但在逻辑里并未使用。

### 1.2 `GetVisibleEntities()` 的行为

实现等价于：

- 对 `allPlayers` 做一次全量遍历：
  - 跳过自己
  - 只要对方 `IsAlive==true` 且 `Distance(observer.Pos, p.Pos) <= observer.ViewRadius` 就可见
- 对 `allEntities` 做一次全量遍历：
  - 只要 `Distance(observer.Pos, e.Pos) <= observer.ViewRadius` 就可见

特点：
- **纯距离圆形 AOI**（无墙体遮挡、无视线阻挡、无网格裁剪）
- **无分层**：视觉与听觉、全局事件的“三级 LOD”主要写在设计文档，但实际实现里 AOI 只负责“视觉圆形过滤”

### 1.3 AOI 在服务端的调用链与复杂度

在 [backend/logic/gamestate.go](../backend/logic/gamestate.go) 的 `GetSnapshot(sessionID)` 中：

1. 把 `gs.Entities`（map）复制成 `entSlice`（slice）
2. 调用 `gs.AOI.GetVisibleEntities(p, gs.Players, entSlice)`
3. 额外再做一遍 **Sound Logic**：对 `gs.Players` 全量遍历，计算听觉范围内脚步事件
4. Radar Blips 也会对 `gs.Entities` 再遍历几次（电机/撤离点/空投）

**当房间内有 $P$ 个玩家、$E$ 个实体、每秒 $T$ 次 tick（配置为 20/s）时**：

- 每个客户端快照约为 $O(P + E)$
- 每 tick 给所有客户端广播（约 $P$ 个客户端）时，总开销约 $O(P\cdot(P+E))$

在 Alpha 0.5（max 6 人）还能接受，但 Beta 目标若想“支持更大规模对战/更稳定 tick”，此实现会成为主要 CPU 热点与 GC 热点。

---

## 2. 现存问题清单（按维度）

本节把问题映射到 beta 方案中的目标：**性能、稳定性、可维护性、体验**。

### 2.1 性能瓶颈（P0/P1 关注）

- **O(P\*P + P\*E) 的扩展性差**：人数、实体稍增就出现非线性增长。
- **重复遍历与重复距离计算**：视觉过滤一次、听觉逻辑再扫一次。
- **每次快照都复制实体 map → slice**：`entSlice` 的分配/拷贝会造成额外 GC 压力。
- **map 遍历顺序不稳定**：返回的 `visibleEntities` 顺序可能每帧变化，导致客户端渲染“轻微抖动/闪烁”（尤其 UI 依赖顺序时）。

### 2.2 稳定性与正确性风险（P0 关注）

- **无“墙体遮挡/视线阻挡”**：在迷宫里，玩家可能“隔墙看见”不符合 Fog-of-war 直觉（体验问题也会反过来影响反馈）。
- **AOIGridSize 配置未使用**：`game_config.json` 中 `map.aoi_grid_size` 已存在，但 AOI 完全没用上，配置-实现不一致会引发调参误解。

### 2.3 可维护性与演进阻力（P1/P2 关注）

- AOI 只提供“视觉过滤”，而听觉/雷达/全局事件在 `GetSnapshot` 分散实现，导致“感知系统”难以统一优化。
- 当前 AOI 接口传入 `allEntities []Entity`，但实体源头是 `map[string]Entity`，导致每次快照都有数据整形成本。

---

## 3. AOI 改进方向（从最小可行到长期演进）

### 3.1 Direction A：网格化空间索引（Grid Spatial Index）——最优先

**目标**：把查询复杂度从“全量扫描”变成“只扫描邻近格子”。

建议实现：

- 以 `map.aoi_grid_size` 为 cell 边长（单位按当前坐标系：格/米需统一），把世界划分为 $\lceil W/g \rceil \times \lceil H/g \rceil$。
- 维护两套索引：
  - `cellPlayers[cell] -> []playerID`
  - `cellEntities[cell] -> []entityID`
- 在 `UpdateTick`/移动处理时：当对象跨 cell 时，更新其 cell membership。
- 在 AOI 查询时：
  - 用 observer 的 `ViewRadius` 计算覆盖的 cell 范围（通常 3x3、5x5）
  - 只遍历这些 cell 内的对象，再做一次精确距离过滤

收益：
- 单次查询近似变为 $O(k)$（$k$ 为邻近 cell 内对象数量），大幅改善扩展性。
- 这是 Beta 1.0 最容易落地、收益最高的性能改造。

注意点：
- 需要定义“坐标单位”和“grid_size 的真实含义”（目前代码里 `pos` 是浮点，地图 tile 是 int 网格）。
- 建议输出可见列表时做 **稳定排序**（例如按 `UID`），避免客户端闪烁。

### 3.2 Direction B：感知系统统一（Vision + Hearing + Radar）

**目标**：减少重复扫描、统一 LOD 策略，把 `GetSnapshot` 中的感知逻辑收敛到 AOI/Perception 层。

建议：
- 抽象 `PerceptionService`（或扩展 `AOIManager`）提供单入口：
  - `ComputePerception(observer) -> {visionPlayers, visionEntities, soundEvents, radarBlips}`
- 让 “听觉范围内脚步” 不再全量扫描，而是复用网格索引：
  - 听觉查询用 `HearRadius` 覆盖 cell 范围
  - 只对邻近玩家计算方向与强度
- Radar blips 也可走索引，或者在实体状态变化时维护一个“雷达候选集”（如 motors/exits/supplyDrops 的 ID 集合）

收益：
- 降低每帧重复遍历次数
- 使“三级 LOD 广播”真正可工程化：视觉/听觉/全局事件的职责边界清晰

### 3.3 Direction C：墙体遮挡与视线（Line-of-Sight）

**目标**：在迷宫里做到“隔墙不可见”，让 Fog-of-war 更可信。

可选实现（从易到难）：

1. **格子 LOS（推荐 Beta 1.x）**：
   - 对位于视野半径内的目标格子，用 Bresenham 线段检查是否有墙阻挡
   - 对玩家/关键实体（玩家、道具、交互点）做 LOS，避免对所有 entity 做 LOS 导致太重
2. **Shadowcasting（后续）**：
   - 以 tile 为基础计算可见区域（递归阴影投射/Permissive FOV）
   - 可缓存每 tick 的可见 tile 集，再映射到实体

取舍：
- LOS 正确性提升明显，但 CPU 成本也上升；必须配合 Direction A 的空间索引与“只对重要对象做 LOS”。

### 3.4 Direction D：带宽与快照策略（Interest Management → Deltas）

与 beta 方案的 **NET-001（JSON 冗余）** 目标对齐：

- 在 AOI 层输出“进入/离开视野”的 delta（enter/leave/update），而不是每 tick 全量发。
- 或最小化 payload：实体只发必要字段（pos/type/state），`Extra` 另走按需加载/事件。

这一步通常需要协议层配合（Beta 1.x/2.0 更合适），但 AOI 是天然切入点。

---

## 4. 落地任务拆分与优先级（与 beta 方案一致）

### P0（关键性能/稳定性修复）

1. **AOIGridSize 生效**：实现网格化索引的最小版本（只用于 Vision 查询）。
2. **稳定排序输出**：`visiblePlayers/visibleEntities` 按 `SessionID/UID` 排序，避免客户端闪烁。
3. **减少分配**：避免每次快照都构建 `entSlice`（改为维护一个可复用的实体 slice 缓存，或把 AOI 直接对 map 迭代并做 grid 索引）。

### P1（核心体验/性能提升）

1. **听觉查询走 AOI 索引**：用 `HearRadius` 的 cell 范围替代全量扫描。
2. **感知系统统一入口**：把 `GetSnapshot` 中 Vision/Sound/Radar 的分散逻辑收敛。
3. **压力测试基准**：加基准测试/压测脚本，确保 tick 不抖动。

### P2（重要能力增强）

1. **LOS（墙体遮挡）**：对“玩家 + 关键实体”做 LOS。
2. **Delta 快照**：进入/离开视野的增量协议（为 Protobuf 迁移做准备）。

---

## 5. 验收指标（Success Metrics）

建议把 AOI 改造的成功定义为可量化指标：

- **服务端 tick 稳定性**：tick 执行时长 P99 < 50ms * 30%（留足逻辑/网络余量）
- **AOI 查询耗时**：
  - Alpha 规模（6 人）无回退
  - 扩展到 16 人时，AOI 查询耗时增长近似线性（而非二次）
- **分配与 GC**：每 tick 的 heap allocation 明显下降（可用 `pprof` 观测）
- **一致性**：同一位置与半径下，可见集合稳定（排序稳定 + 逻辑稳定）

---

## 6. 建议的实现顺序（最短路径）

1. **先做 Direction A（网格索引）**：收益最大，风险最小。
2. **接着把 Hearing 迁入索引查询**：立刻减少一轮全量遍历。
3. **再考虑 LOS**：在性能可控的前提下提升 Fog-of-war 的真实性。
4. **最后做 Delta/协议升级**：与后续 Protobuf 迁移同一波推进。

---

## 7. 附：对当前 `aoi.go` 的具体改动建议（接口层）

为了支持演进，建议未来把 AOI API 从“传入全量列表”改为“面向索引”的查询：

- `UpdatePlayerPosition(playerID, oldPos, newPos)`
- `UpdateEntityPosition(entityID, oldPos, newPos)`
- `QueryVision(observerID) -> []playerID, []entityID`
- `QueryHearing(observerID) -> []soundEvent`

这样能把 `GetSnapshot()` 的职责收敛成：组装 payload，而不是承担感知计算。
