# Item System 评审与改进方向（面向 Beta）

> **范围**：服务端道具/掉落/使用/价值/UID 生成（Item System）  
> **对应版本**：Echo Trace Alpha 0.5 → Beta 1.0  
> **目标对齐**：与 [docs/beta_iteration_plan.md](beta_iteration_plan.md) 的 **稳定性 / 性能 / 可维护性 / 体验（平衡）**一致

---

## 1. 当前实现：`item_system.go` 的真实职责

相关代码：
- [backend/logic/item_system.go](../backend/logic/item_system.go)
- [backend/logic/gamestate.go](../backend/logic/gamestate.go)（购买/出售/撤离结算等）
- [backend/storage/sqlite.go](../backend/storage/sqlite.go)（资金持久化）

`item_system.go` 当前包含：

- 道具静态表 `ItemDB`（内置 3 类道具，含 Tier/Weight/Value）
- 可选的价值覆盖 `LoadItemValues()`（从 `../item_values.json` 读取）
- 掉落生成：`SpawnSupplyDrop`、`SpawnRandomItem`（由 `GameState` 调用）
- 拾取与使用：`HandlePickup`、`HandleUseItem`
- 简易战斗：`findNearestEnemy`、`handleDeath`（死亡掉落）
- UID 生成：`NewUID()`（atomic + time）

---

## 2. 已实现优势

- **闭环可跑**：拾取→加钱/入包，商店购买（在 gamestate），撤离结算与写库已串通。
- **数据结构直观**：Item 里有 Tier/Weight/Value，便于后续平衡。
- **全局 UID 生成简单可用**：对单进程房间足够。

---

## 3. 主要问题与风险

### 3.1 稳定性/正确性

1. **并发修改玩家视野的 goroutine 风险**  
   - `HandleUseItem` 的雷达道具通过 `go func(){ time.Sleep(); gs.Mutex.Lock(); ... }` 延迟回收 buff。
   - 风险点：
     - 玩家退出/被移除后仍可能被修改（目前闭包持有 `*Player` 指针）。
     - 如果未来把 `Player` 对象复用或变为值拷贝，会出现更隐蔽 bug。
   - 建议：buff/持续效果应由 `UpdateTick` 统一推进（时间轮/状态字段），避免额外 goroutine。

2. **`Entity.Extra` 强类型断言易 panic**  
   - `HandlePickup` 中 `ent.Extra.(Item)`、`.(SupplyDropData)`。
   - 一旦 Extra 被序列化/反序列化或误写（或未来新增实体类型复用 Extra），会直接 panic。

3. **“MaxUses”没有真正使用**  
   - 当前 `MaxUses>1` 只是注释，实际每次使用都直接删物品。
   - 这会让配置与表现不一致，影响平衡与玩家理解。

### 3.2 性能与可扩展性

1. **随机数与选择方式不稳定/偏差**  
   - 多处用 `time.Now().UnixNano() % n` 选择候选项。
   - 会造成：
     - 可预测性更强（尤其在相近时间调用）
     - 分布不均（取模偏差在某些 n 下更明显）
   - 建议统一使用 `math/rand.Rand`（带 seed），或直接用 `rand.Intn(n)`。

2. **拾取扫描是全量 Entities**  
   - `HandlePickup` 遍历 `gs.Entities` 找距离最近的掉落/空投。
   - 随着实体增多，拾取成本上涨；与 AOI 网格索引方向一致，应复用空间索引。

3. **UID 生成包含时间戳 + 自增，字符串较长且频繁分配**  
   - 对 alpha 足够，但在实体量大时会制造较多短命字符串。
   - Beta 可考虑：
     - 改为 `uint64` ID（网络层再转字符串）
     - 或保持字符串但减少生成频率/长度

### 3.3 体验与平衡

1. **空投越级与资金曲线过陡**  
   - `SpawnSupplyDrop` 的 tier = phase+1（Phase1 就可能出 Tier2），且 Funds = 500 * tier。
   - 与 beta 方案的 DESIGN-001（滚雪球）高度相关。
   - 建议把空投掉落规则完全配置化，并引入“稀有度/权重/保底/反滚雪球”策略。

2. **拾取附带随机资金 20-80**  
   - 随机性会放大运气对经济曲线的影响，且与 item.Value 的经济体系叠加后更难控。

3. **攻击判定与视野耦合过弱**  
   - `findNearestEnemy` 要求 `d <= attacker.ViewRadius`，但没有 LOS/墙体遮挡，会出现“隔墙打人”的体验问题（需与 AOI/LOS 改造联动）。

---

## 4. 改进方向（工程化落地）

### 4.1 Direction A：配置驱动的掉落与经济曲线（P0/P1）

- 将以下规则从硬编码迁移到 config：
  - 空投资金、空投物品数量、tier 规则（是否允许越级）
  - 普通掉落的物品池与权重
  - 拾取时的资金奖励（可改为与物品价值绑定或取消随机钱）
- 引入 **权重池**（weighted random）替代简单 candidates 列表。

### 4.2 Direction B：道具效果系统化（Buff/Duration/Charges）（P1/P2）

- 将“持续效果”改为 Player 上的状态字段：
  - `activeEffects[]`（type, magnitude, expiresAt）
  - `charges`（替代 MaxUses 直接删物品）
- 由 `UpdateTick` 统一处理效果过期，移除额外 goroutine。

### 4.3 Direction C：空间索引复用（P1）

- 拾取查询复用 AOI 网格索引：只扫描玩家附近 cell 的掉落实体。

### 4.4 Direction D：类型安全（P1/P2）

- 去掉 `Entity.Extra interface{}` 的强断言使用方式：
  - 方案 1：为每个实体类型定义固定 payload 结构（MotorPayload/SupplyDropPayload/ItemDropPayload）
  - 方案 2：Extra 使用 `json.RawMessage` + 按 type 解码

---

## 5. 任务拆分与优先级

### P0：关键缺陷/一致性修复

1. 掉落规则配置化：禁止 Phase1 越级掉落（或可配置开关）
2. `MaxUses` 行为与数据一致（最小实现：增加 `UsesLeft` 字段并递减）

### P1：核心体验与性能

1. 持续效果从 goroutine 改为 tick 驱动（效果列表/过期机制）
2. 拾取与附近查询走空间索引
3. 经济曲线调整：减少“随机钱”对局内决策的噪声

### P2：可维护性与长期演进

1. 类型安全的实体 payload 结构
2. 权重池 + 反滚雪球机制（依据玩家资金/领先程度动态调整）

---

## 6. 验收指标（Success Metrics）

- **稳定性**：道具使用/拾取/空投不会触发 panic（Extra 解码安全）
- **一致性**：MaxUses/持续时间与实际表现一致，且可通过 config 调整
- **平衡**：Phase1 领先者的资金/装备优势曲线可控（可通过局内统计验证）
- **性能**：实体量增大时，拾取查询耗时不随 Entities 线性飙升（索引生效）
