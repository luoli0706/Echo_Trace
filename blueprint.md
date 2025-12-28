# Echo Trace 技术蓝图（与当前代码一致）

> 本文档以当前仓库实现为准（Go 后端 + Python/Pygame 客户端）。详细玩法流程见 [GameFlow.md](./GameFlow.md)，数值配置见 [game_config.json](./game_config.json)，道具设计见 [Items.md](./Items.md)。

---

## 1. 项目定位

- **核心循环**：迷宫搜刮 → 购买/使用道具 → 修电机 → 撤离结算（资金持久化）
- **对抗形式**：多人混战（FFA）
- **信息核心**：90°扇形视野 + 视线遮挡（墙体阻挡实体可见性），服务器端 AOI 过滤避免信息泄露

---

## 2. 总体架构

### 2.1 组件拆分

- **后端（Go）**
  - WebSocket 入口与房间管理：`backend/network/*`
  - 游戏逻辑与状态：`backend/logic/*`
  - 数据持久化：`backend/storage/*`（SQLite，默认 `game.db`）
- **前端（Python + Pygame）**
  - 网络收发：`frontend/client/network.py`
  - 状态缓存：`frontend/client/gamestate.py`
  - 渲染/UI：`frontend/client/renderer.py`
  - 主循环与输入：`frontend/main.py`

### 2.2 数据流（关键）

1. 客户端发送输入（移动、面向、使用道具、拾取/交互、商店操作等）
2. 服务端 `GameLoop` 每 tick：
   - `UpdateTick(dt)`：推进阶段计时、交互引导、物理移动、道具补充等
   - 对每个玩家生成快照 `GetSnapshot(sessionID)`
3. 快照通过 WebSocket 广播给对应客户端
4. 客户端只渲染“能看到/能听到”的信息

---

## 3. 房间与主循环

### 3.1 房间

- **创建/加入**：客户端菜单 `CREATE_ROOM(1010)` / `JOIN_ROOM(1011)`
- **房间配置**：
  - 服务端启动时读取 `game_config.json` 作为默认值
  - 创建房间时可通过 payload 覆盖（支持 legacy 平铺字段，也支持 `payload.config` 结构化覆盖）

### 3.2 GameLoop（Tick + 输入通道）

- `backend/logic/loop.go`：`GameLoop.Run()`
  - `InputChan`：串行处理玩家输入（Move/UseItem/Interact/Pickup/Drop/Buy/Sell/Refresh/Tactic/Login）
  - `Ticker`：按 `server.tick_rate_ms` 调用 `UpdateTick(dt)`
  - `SnapshotChan`：按玩家生成快照并交给网络层发送

---

## 4. 视野 / AOI / 反作弊

### 4.1 服务端 AOI：90°扇形 + LOS

- `backend/logic/aoi.go`：
  - 半角 45°（总角度 90°）
  - 半径 `player.view_radius`
  - 视线遮挡：墙体阻挡（LOS）
- **重要约束**：服务端只把“扇形内且无遮挡”的玩家/实体下发给客户端，避免客户端通过网络包窥探墙后实体

### 4.2 客户端 Fog 策略

- 扇形外：全黑
- 扇形内：地图可见（墙体/地形照常渲染）
- 实体可见性：只显示扇形内且无遮挡的实体（与服务端一致）

---

## 5. 阶段状态机（以当前实现为准）

- Phase 0：大厅/选战术（客户端按 1/2/3 选择 RECON/DEFENSE/TRAP）
- Phase 1（Search）：按配置倒计时结束后进入 Phase 2
- Phase 2（Conflict）：
  - 刷新电机与补给箱
  - 需要修复 2 个电机触发撤离阶段
  - “电机脉冲”事件会周期性产生（当前用于事件提示与电机雷达提示）
- Phase 3（Escape）：
  - 刷新出口（Exit），玩家持续交互 3 秒撤离成功
- Phase 4（Ended）：存在枚举但当前流程主要在撤离后进入观战/或房间结束逻辑

> 注：配置文件里存在“视野衰减、撤离名额限制”等字段，但当前代码未完整实现这些机制；文档不做强描述，避免误导。

---

## 6. 道具/经济系统

### 6.1 道具库与刷新

- `backend/logic/item_system.go`：`ItemDB` + 按阶段 Tier 权重 + 按战术偏好调整类别概率
- 拾取掉落：拾取道具会额外获得一笔随机资金（当前实现）

### 6.2 战术（Tactic）影响

- 影响：初始属性倍率（血量/移速/视距/听距）与道具效果倍率（治疗/伤害/侦察）
- 配置：`game_config.json` → `tactics.RECON/DEFENSE/TRAP`

### 6.3 负重（Weight）

- `player.weight` 由背包物品重量求和
- 速度惩罚：负重比例越高，移速越低（当前上限惩罚 60%，最低不低于 2.0）
- 配置里存在更复杂阈值字段，但当前版本主要使用“比例→速度惩罚”这一条

### 6.4 “被动道具”已改为限时增益

- 为了与负重/背包上限互动一致，原先的被动效果均改为“使用后持续一段时间”的服务端 Buff
- Buff 由服务端字段维护，每 tick 重新计算，确保到期一定回退

---

## 7. 商人系统（NPC Merchant）

- 每个玩家有独立的 `shop_stock`（服务端生成并下发）
- 购买限制：只能买自己当前 `shop_stock` 里的商品
- 刷新规则：
  - **每阶段每玩家**第一次刷新免费
  - 后续刷新扣除 `items.merchant_refresh_cost`
- 位置规则：商人在每个阶段会移动到该阶段的固定锚点（进入下一阶段时换点）

---

## 8. 持久化

- `backend/storage/*`：按玩家名字保存资金等数据
- 撤离结算：背包道具按价值结算为资金，清空背包并进入观战态
