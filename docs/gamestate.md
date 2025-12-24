# GameState 评审与改进方向（面向 Beta）

> **范围**：服务端单局状态与仿真核心（GameState）  
> **对应版本**：Echo Trace Alpha 0.5 → Beta 1.0  
> **目标对齐**：与 [docs/beta_iteration_plan.md](beta_iteration_plan.md) 的 **稳定性 / 性能 / 可维护性 / 体验**目标一致

---

## 1. 当前实现：`gamestate.go` 的真实职责边界

相关代码：
- [backend/logic/gamestate.go](../backend/logic/gamestate.go)
- [backend/network/room.go](../backend/network/room.go)（tick 驱动与广播）
- [backend/storage/sqlite.go](../backend/storage/sqlite.go)（持久化）
- [backend/logic/item_system.go](../backend/logic/item_system.go)（道具与价值）
- [backend/logic/aoi.go](../backend/logic/aoi.go)（视野过滤）

### 1.1 `GameState` 承担的功能

从结构体字段与方法分布看，[backend/logic/gamestate.go](../backend/logic/gamestate.go) 同时承担：

- **对局状态机**：PhaseInit/Search/Conflict/Escape/Ended，计时器（PhaseTimer、PulseTimer、RespawnTimer）
- **实体与玩家容器**：Players(map)、Entities(map)
- **交互/经济/结算**：购买/出售/拾取/丢弃/撤离结算
- **物理移动与碰撞**：`isWalkableWithRadius` + 每 tick 更新位置
- **交互引导（Channeling）**：电机修复、撤离点引导
- **快照生成**：`GetSnapshot` 负责 AOI、雷达 blips、听觉事件
- **持久化触发**：登录加载、断连保存、撤离即刻保存

这对应 beta 方案里提到的 **God Object（GameState 过载）**：当规则变复杂，维护成本与引入 bug 风险会显著上升。

---

## 2. 核心机制评估（现状优点与缺陷）

### 2.1 ✅ 已实现的优势

- **流程清晰**：Phase 驱动的单局流程简单直观；`UpdateTick` 统一推进时间。
- **锁语义相对一致**：绝大多数 `Handle*` 与 `UpdateTick` 都在 `gs.Mutex.Lock()` 内操作共享状态。
- **可配置的基础参数**：移动速度/视野、tick 等已由 config 驱动（但仍有关键硬编码，见缺陷）。
- **经济闭环基础可用**：拾取→资金、商店购买、撤离结算与写库基本串起来了。

### 2.2 ⚠️ 具体缺陷与瓶颈（带可定位点）

#### A. 稳定性 / 逻辑正确性

1. **BUG-001：撤离事件文本格式化错误**  
   - 在 `ProcessExtraction` 中使用 `string(rune(lootValue))` 拼接整数，必然出现乱码或非预期字符。
   - 该问题已在 beta 方案中记录为关键缺陷。

2. **BUG-002：开局最小玩家数硬编码为 1**  
   - `HandleChooseTactic` 中 `minPlayers := 1` 属于调试值，破坏真实匹配/开局条件。

3. **状态机与配置不一致**  
   - `nextPhase` 内对 PhaseConflict 设置 `PhaseTimer = 9999`，并把电机数量写死为 `spawnMotors(5)`。
   - 但配置里存在 `phases.phase_2_conflict.duration_sec`、`motors_spawn_count`、`motors_required_to_open_exit` 等。
   - 这会导致“调配置无效果”的隐藏问题，后续平衡调参会非常痛。

4. **撤离成功后调用 `RemovePlayer` 的语义风险**  
   - `UpdateTick` 的 channeling loop 正在遍历 `gs.Players`，同时在循环体内调用 `gs.RemovePlayer(p.SessionID)`（该函数内部再次 `Lock()`）。
   - 这在 Go 中会导致 **同一 goroutine 重入锁死（deadlock）** 或逻辑异常：当前 `UpdateTick` 已持有 `gs.Mutex.Lock()`，再次调用 `RemovePlayer` 会阻塞。
   - 这属于 P0 级别的稳定性风险（即便当前跑起来“偶尔没触发”，也属于一碰就炸）。

#### B. 性能

1. **快照计算与广播耦合导致的成本叠加**  
   - `GetSnapshot` 内含 AOI、雷达扫描、听觉扫描，且每 tick 对每个 client 都执行。
   - 在人数上升时成本近似二次增长；与 beta 方案里的“性能优化/降低延迟”目标冲突。

2. **Entity map → slice 的每帧复制**  
   - `entSlice := make([]Entity, 0, len(gs.Entities))` 每次快照都分配+拷贝，会带来额外 GC 压力。

#### C. 体验/平衡

1. **电机修复与撤离引导缺少“被打断”规则**  
   - `HandleInput` 里只要移动就清空 `ChannelingTargetUID`，但并未对“受到攻击/事件”做统一的打断/进度衰减逻辑。
   - beta 方案提出的“进度衰减机制”在当前实现还没有落地接口。

2. **空投位置策略与 Phase 1 滚雪球风险**  
   - `spawnPhaseSupplyDrops` 采用“玩家质心附近 + 随机偏移”的生成策略，可能进一步强化领先者控制资源。
   - 与 beta 方案中的 DESIGN-001（滚雪球）直接相关。

#### D. 可维护性

1. **职责过载（单文件承载过多系统）**：输入、物理、交互、经济、持久化、快照都在一起，变更影响面大。
2. **魔法数字分散**：例如 `HearRadius: 12.0`、`PhaseTimer: 120`、`ExtractionTimer: 3.0`、脉冲 15s、修复速度 `20.0*dt` 等。
3. **类型断言风险**：`ent.Extra.(MotorData)` 等断言在数据不一致时会 panic（未来扩展实体类型时风险上升）。

---

## 3. 改进方向（与 beta 方案对齐）

### 3.1 Direction A：把“规则参数”全面配置化（P0/P1）

目标：消除当前“配置存在但不生效”的情况，让平衡迭代可控。

建议：
- 从 `game_config.json` 读取：
  - Phase 2/3 的持续时间（而不是 9999/120）
  - `motors_spawn_count`、`motors_required_to_open_exit`
  - `motor_decipher_time_sec`（映射为 progress 增长速度）
  - `extraction_cooldown_sec`、`extraction_slots_total` 等（当前逻辑未完整体现）

收益：
- 直接支持 beta 方案中的“平衡调优/快速迭代”。

### 3.2 Direction B：解耦 GameLoop 与系统子模块（P1/P2）

目标：降低 `GameState` 的变更爆炸半径，提升可测试性。

建议拆分：
- `PhaseSystem`：只负责阶段推进与事件
- `InteractionSystem`：Channeling/电机/撤离
- `EconomySystem`：拾取、售卖、购买、结算
- `MovementSystem`：移动与碰撞
- `SnapshotSystem`：AOI/雷达/听觉整合（与 AOI 文档一致）
- `PersistenceGateway`：封装 storage 调用与写库策略

收益：
- 与 beta 方案的 P2-001（逻辑网络解耦）、P2-003（测试覆盖）高度一致。

### 3.3 Direction C：修复锁重入与遍历删除风险（P0）

目标：消除撤离/断连等关键路径上的死锁风险。

建议模式：
- 在 `UpdateTick` 内不要调用会再次 `Lock()` 的方法。
- `RemovePlayer` 提供一个 **不加锁的内部版本**（例如 `removePlayerInternal`），由持锁方调用。
- 或者在 `UpdateTick` 内先收集待移除的 sessionID 列表，循环结束后统一移除。

收益：
- 直接提升稳定性，是 Beta 前必须解决的问题。

### 3.4 Direction D：快照与感知（AOI/声音/雷达）成本治理（P1）

目标：降低 tick 内每客户端快照成本，与 AOI 改造联动。

建议：
- 复用 [docs/aoi.md](aoi.md) 的网格索引，减少可见/听觉查询的全量扫描。
- 将雷达候选集合（电机/撤离点/空投）改为“状态变化时维护集合”，避免每帧遍历所有 Entities。
- 逐步引入 delta 快照（enter/leave/update）为后续 Protobuf 迁移铺路。

---

## 4. 任务拆分与优先级（同 aoi.md 的格式）

### P0：关键缺陷修复（必须在 Beta 前完成）

1. **修复撤离事件字符串格式化（BUG-001）**
2. **移除 minPlayers 调试硬编码（BUG-002）**
3. **修复 UpdateTick 内撤离成功导致的锁重入/死锁风险**
4. **把 motors_fixed 相关阈值与 motors 数量接入 config**

### P1：核心体验与性能优化（Beta 1.0 目标）

1. **Phase 定时逻辑全面配置化**（Phase2/Phase3 duration、pulse interval 等）
2. **Channeling 机制完善**：支持“打断/进度衰减/反馈事件”
3. **快照性能治理**：减少全量扫描与分配（与 AOI 网格索引联动）

### P2：重要能力增强（Beta 1.x/2.0 目标）

1. **系统解耦重构**（Movement/Interaction/Economy/Snapshot）
2. **统一错误处理与日志策略**：对 storage 错误进行分级与回退
3. **可测试性建设**：为 Phase/Extraction/Economy 增加单测与基准

---

## 5. 建议的验收指标（Success Metrics）

- **稳定性**：撤离/断连路径 0 死锁、0 panic（压测 + 回归）
- **配置一致性**：关键玩法参数（阶段时长、电机数量/阈值、交互时长）全部可通过 `game_config.json` 生效
- **性能**：
  - tick 执行时长 P99 显著低于 tick 间隔（50ms）
  - `GetSnapshot` 的 heap allocation 明显下降（pprof 可证）

---

## 6. 最短落地路径（推荐顺序）

1. 先做 P0：修 BUG + 去硬编码 + 修锁重入
2. 再做 P1：参数配置化 + 快照/感知成本治理
3. 最后做 P2：按系统拆分，补测试与基准
