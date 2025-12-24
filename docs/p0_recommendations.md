# P0 级修改意见整合（来自 docs 评估文档）

> **目的**：把当前 docs 内所有“必须在 Beta 前完成”的 P0 事项合并成一份可直接开工的清单（去重、补充落地建议与验收口径）。  
> **日期**：2025-12-24

**来源文档**：
- [docs/beta_iteration_plan.md](beta_iteration_plan.md)
- [docs/gamestate.md](gamestate.md)
- [docs/aoi.md](aoi.md)
- [docs/types.md](types.md)
- [docs/item_system.md](item_system.md)
- [docs/maze.md](maze.md)

---

## P0 清单（合并版）

### P0-01 修复撤离事件字符串格式化（BUG-001）

- **问题**：撤离事件文案使用 `string(rune(lootValue))` 拼接整数，必然乱码。
- **涉及**：[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现**：使用 `strconv.Itoa(lootValue)` 或 `fmt.Sprintf("%d", lootValue)`。
- **验收**：撤离事件显示金额正确，且无乱码；包含回归用例（至少本地双人撤离一次）。

### P0-02 移除调试硬编码开局条件（BUG-002）

- **问题**：`minPlayers := 1` 是调试值，破坏真实开局规则。
- **涉及**：[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现**：
  - 最小人数从配置读取（建议新增 `server.min_players_to_start` 或复用现有字段，保持单一事实来源）。
  - 若短期不改 config schema，也至少把 `1` 改为 `2` 并集中为常量。
- **验收**：未达到最小人数不会自动开局；达到后可开局。

### P0-03 修复撤离成功路径的锁重入/死锁风险（关键稳定性）

- **问题**：`UpdateTick` 已持有 `gs.Mutex.Lock()`，但在遍历 `gs.Players` 时调用 `gs.RemovePlayer()`，该函数内部再次 `Lock()`，存在死锁风险。
- **涉及**：[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现（任选其一，推荐 A）**：
  - **A**：在 `UpdateTick` 中先收集 `toRemove []sessionID`，循环结束后统一移除（移除逻辑不再二次加锁）。
  - **B**：提供 `removePlayerInternal()`（不加锁版本），仅允许在持锁上下文调用。
- **验收**：撤离成功后服务端不会卡死；连续多次撤离/断连压力下 0 死锁。

### P0-04 修复房间状态竞争（BUG-003，来自 Beta 方案）

- **问题**：房间主循环中的阶段切换/状态变量存在并发读写风险（beta 方案已标注）。
- **涉及**：[backend/network/room.go](../backend/network/room.go)
- **建议实现**：
  - 明确 `lastPhase` 与 `GameState.Phase` 的同步策略（统一在同一把锁下读写，或只在单 goroutine 内维护）。
  - 对 `client.Send` 的非阻塞写入要有可观测性（至少计数/日志采样），避免“静默丢包”难排查。
- **验收**：在多客户端连接/断开/阶段切换时无数据竞争（建议用 `-race` 本地跑一次）。

### P0-05 断线重连最小方案（NET-002，来自 Beta 方案）

- **问题**：网络波动会导致玩家资产与局内状态不可恢复。
- **涉及**：[backend/network/client.go](../backend/network/client.go)、[backend/network/manager.go](../backend/network/manager.go)、[backend/network/room.go](../backend/network/room.go)、[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现（最小闭环）**：
  - 允许客户端携带 `session_id` 重新连接；在一定 TTL 内恢复 Player（位置/资金/背包/阶段）。
  - TTL 到期则按“断连淘汰/保存资产”规则处理（规则要明确且可配置）。
- **验收**：断线后在 TTL 内重连可回到同房间；超过 TTL 行为符合预期且有日志。

### P0-06 游戏结束条件与收尾（来自 Beta 方案）

- **问题**：缺少明确的 PhaseEnded/房间清理与胜负结算闭环。
- **涉及**：[backend/logic/gamestate.go](../backend/logic/gamestate.go)、[backend/network/room.go](../backend/network/room.go)
- **建议实现（最小闭环）**：
  - 定义结束条件（时间到/名额耗尽/仅剩 0-1 名存活等）。
  - 结束后广播结算事件，并进入 `PhaseEnded`，停止关键系统（掉落刷新/交互等），房间进入可销毁状态。
- **验收**：任意结束条件触发后对局可正常结束，客户端收到明确结算事件，房间可回收。

### P0-07 AOI：让 `aoi_grid_size` 真正生效 + 稳定输出 + 降低分配

- **问题**：AOI 当前全量扫描、重复计算、每快照复制 entities，且输出顺序不稳定；同时 config 中 `map.aoi_grid_size` 未生效。
- **涉及**：[backend/logic/aoi.go](../backend/logic/aoi.go)、[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现（最小可行）**：
  - 建立网格索引（Vision 查询先按 cell 粗筛，再做距离精筛）。
  - `visiblePlayers/visibleEntities` 做稳定排序（按 SessionID/UID）。
  - 取消每快照 `entSlice` 的新分配（缓存或改为索引直查）。
- **验收**：AOI 查询耗时随玩家/实体增长近似线性；同场景下客户端不因顺序抖动闪烁。

### P0-08 Map：出生点死循环保护 + 关键实体投放可达

- **问题**：`GetRandomSpawnPos()` 在高墙密度下可能长时间循环；且地图不保证连通，关键实体可能不可达。
- **涉及**：[backend/logic/maze.go](../backend/logic/maze.go)、[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现（最小可行）**：
  - `GetRandomSpawnPos` 增加最大尝试次数与失败回退。
  - 关键实体（商人/电机/撤离点/空投）至少保证投放在同一连通区域（最大连通分量）。
- **验收**：任何配置下不会生成卡死；关键实体对玩家可达。

### P0-09 Item：掉落规则配置化（禁越级）+ MaxUses 行为一致

- **问题**：Phase1 空投可能越级掉落（滚雪球风险）；`MaxUses` 字段与实际行为不一致。
- **涉及**：[backend/logic/item_system.go](../backend/logic/item_system.go)、[backend/logic/gamestate.go](../backend/logic/gamestate.go)
- **建议实现（最小可行）**：
  - 将“空投 tier 规则/是否越级”接入配置，并默认禁止 Phase1 越级。
  - 引入 `UsesLeft` 或等价字段，让多次使用道具有真实的消耗逻辑。
- **验收**：Phase1 不再掉落高阶道具（除非配置允许）；多次使用道具能正确递减并在用尽后移除。

### P0-10 Types：补齐 `GameConfig` 与 `game_config.json` 的一致性

- **问题**：Go 侧 `GameConfig` 未覆盖 JSON 的多数字段，导致调参“表面成功、实际无效”。
- **涉及**：[backend/logic/types.go](../backend/logic/types.go)、[game_config.json](../game_config.json)、以及引用这些字段的逻辑（如 gamestate/item）。
- **建议实现（最小可行）**：
  - 把 `game_config.json` 里已存在的字段补齐到 `GameConfig`（至少覆盖目前被硬编码使用的：inventory/hearRadius/phase3/combat/weight thresholds 等）。
  - 把对应硬编码改为读取 config。
- **验收**：修改 [game_config.json](../game_config.json) 的关键字段能在运行时体现，无需改代码。

---

## 推荐落地顺序（降低返工）

1. P0-03（死锁）→ P0-01（乱码）→ P0-02（开局条件）
2. P0-10（补齐 config）
3. P0-08（spawn/可达）与 P0-09（掉落/越级）
4. P0-04（room 竞态）与 P0-05（重连）与 P0-06（结束收尾）
5. P0-07（AOI 网格/排序/分配）

---

## P0 完成的统一验收口径

- **稳定性**：撤离/断连/阶段切换/重连路径 0 死锁、0 panic
- **一致性**：配置可驱动关键参数（至少开局条件/阶段时长/空投规则/听觉半径/背包容量）
- **可玩性**：关键实体可达，游戏可结束且可回收房间
- **性能底线**：在目标人数（例如 6→16 的扩展测试）下 tick 不显著抖动，且 AOI/快照不出现明显非线性爆炸
