# 项目名称：DarkForest-Go (Alpha Version)

> **当前版本状态：** 核心设计已参数化，详细流程请参阅 [GameFlow.md](./GameFlow.md)，配置参数请参阅 [game_config.json](./game_config.json)。

### 项目定位：基于Golang的高性能后端游戏开发面试Demo

### 核心玩法：FFA（自由竞技）+ 迷宫搜刮 + 撤离博弈

---

## 1. 架构设计 (Architecture)

* **服务端 (Server):** 使用 Golang 编写，负责所有核心逻辑（物理、战斗、视野、状态机）。
* **客户端 (Client):** 使用 Python (Pygame/Curses) 编写，仅负责 UI 渲染及简单的指令输入，不参与逻辑计算。
* **通信协议:** 基于 WebSocket 的 JSON 协议（Alpha版为了调试方便，后续可优化为 Protobuf）。
* **配置驱动:** 启动时加载 `game_config.json`，支持热重载（可选）。
* **并发模型:** 
  * **Room Loop:** 每个房间一个主 Goroutine，通过 `select` 监听 Ticker 和 input channel。
  * **Thread Safety:** 采用无锁 Actor 模型或最小化 `sync.RWMutex` 保护共享状态。

---

## 2. 核心系统逻辑 (Core Systems)

详细参数见 `game_config.json`。

### 2.1 地图与移动

* **迷宫生成:** 服务端启动时利用 Prim 算法生成 `map.width` x `map.height` 的二维网格地图。
* **坐标系统:** 使用浮点数坐标，但基于网格进行 AOI（感兴趣区域）裁剪。
* **移动校验:** 服务端每 `server.tick_rate_ms` 更新一次坐标，校验玩家是否穿墙。
* **静默移动:** 速度低于 `silent_speed_threshold` 时不触发听觉广播。

### 2.2 AOI 视野过滤系统 (关键亮点)

* **网格管理:** 地图划分为 `map.aoi_grid_size` 大小的格子 (Grid)，每格管理其中的玩家和道具。
* **三级 LOD 广播:**
  1. **视觉层:** `gameplay.base_view_radius` (如 5m)，同步所有精确坐标及动作。
  2. **听觉层:** `gameplay.hear_radius` (如 12m)，仅同步噪音源方向（UI显示波纹）。
  3. **全局层:** 仅同步撤离点开启等大事件或 Phase III 的全图脉冲。
* **单向透明:** 道具触发的坐标广播不触发目标方的“反向视野”。

### 2.3 动态负重与 Debuff

* **携带计算:** `MoveSpeed = BaseSpeed * (1 - CurrentWeight/MaxWeight)`。
* **阈值影响 (可配置):** 
  - 负重 > `weight_threshold_noise_double`: 移动产生噪音半径翻倍。
  - 负重 > `weight_threshold_view_reduce`: 视野半径缩小。
  - 负重 = `weight_threshold_immobilize`: 无法进行交互（如破译）。

### 2.4 道具组件系统 (Item Component System)

* **设计理念:** 组合优于继承。道具效果由组件堆叠定义。
* `AttackComponent`: 
  - **消耗型攻击:** 必须消耗道具才能发动。
  - **机制:** 校验 TargetID 是否在 `ViewRadius` 内 -> 造成 `combat.base_attack_damage` -> 销毁道具。
* `ScannerComponent`: 
  - **主动侦察:** 赋予短时间全图或局部雷达。
  - **标记诅咒:** 将指定/随机玩家坐标向**全员广播**持续 `advanced_recon_duration_sec` 秒，配合远程武器实现“超视距打击”。
* `StunComponent`: 命中目标后使其视野协议中断 N 秒。
* `StealthComponent`: 消除移动噪音。

---

## 3. 单局进程管理 (Game Session State)

详细流程状态机请参阅 [GameFlow.md](./GameFlow.md)。

1. **Phase 0: 初始化与匹配**
   * 房间创建、资源投放。
   * 玩家连接并进行**战术倾向选择**（侦察/防御/陷阱）。
   * 随机出生点分配（保证最小间距）。

2. **Phase I: 初始搜刮 (潜行期)**
   * 战争迷雾笼罩，依靠视野搜刮。
   * 积累资源，动态负重生效。

3. **Phase II: 冲突爆发 (噪波期)**
   * 电机刷新，破译触发持续噪音。
   * 道具博弈白热化，击杀掉落。

4. **Phase III: 终局撤离 (逃生期)**
   * 撤离点随机开放（名额限制）。
   * 全图脉冲定期暴露位置。
   * 视野持续衰减。

5. **Phase IV: 结算与清理**
   * 撤离成功：道具带出，转化为赏金。
   * 失败/死亡：仅保留**安全箱**内物品。
   * 数据持久化与房间销毁。

---

## 4. 技术栈实现细节

### 服务端 (Go) 核心结构

* `RoomManager`: 管理所有活动中的房间 `map[string]*Room`。
* `Room`: 核心逻辑容器，包含 `Players`, `Map`, `BroadcastQueue`。
* `Config`: 单例结构体，映射 `game_config.json`。
* `Ticker`: 主循环 `for range time.NewTicker(...)`。

### 客户端 (Python) 模拟行为

* **输入:** 监听键盘 (WASD) 和 道具快捷键 (1-5)。
* **预测插值:** 客户端收到服务端坐标后进行线性平滑处理，减少网络抖动感。
* **调试模式:** 支持 `--bot-count 50` 参数启动无头模式进行压测。

---

## 5. Alpha 版交付物清单

1. **`game_config.json`**: 游戏数值配置文件。
2. **`protocol.json`**: 定义 `MoveReq`, `UseItemReq`, `SyncStatePush` 等协议格式。
3. **`main.go`**: 服务端入口，加载配置，处理连接与 Room 分发。
4. **`logic/aoi.go`**: 实现基于网格的 AOI 算法。
5. **`logic/maze.go`**: 迷宫生成逻辑。
6. **`client_sim.py`**: 基于 Pygame 的可视化 UI，支持多实例启动。

---

## 6. 后续迭代方向 (Roadmap)

* **Beta:** 接入 Redis 记录玩家总资产排行榜。
* **RC:** 将消息序列化由 JSON 迁移至 Protobuf，压测单机支撑房间数。