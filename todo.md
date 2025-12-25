# 回声追踪 开发计划 | Echo Trace Development Plan

> **状态 (Status):** Sprint 3 修正中 (Sprint 3 Fixes)
> **目标 (Objective):** Sprint 4 - 性能优化与代码重构 (Optimization & Refactoring)

## 📅 Sprint 3: 经济体系与博弈闭环 (Economy & Game Loop)

### 1. 后端：经济结算与持久化 (Backend: Settlement & Persistence)
- [x] **高价值空投逻辑 | High-Value Supply Drops**
    - *优化:* 已实现重心刷新和物品生成。
    - *雷达:* 已实现持续显示。
    - *修正:* 图标已优化（带边框方形），生成上限已提高。
- [x] **撤离处理 | Process Extraction**
    - *实现:* 在 `ProcessExtraction` 中实现了基础的 "物品 -> 资金" 转换逻辑。
    - *逻辑:* 撤离时清空背包并保存资金。
- [x] **SQLite 持久化层 | SQLite Persistence Layer**
    - *基础:* 已集成 SQLite。
    - *任务:* 玩家断开连接或撤离时保存数据。
    - *任务:* 玩家登录 (`SetPlayerName`) 时加载数据。
- [x] **道具价值系统 | Item Value System**
    - *数据:* 创建了 `item_values.json`。
    - *逻辑:* 后端加载价值，撤离时按价值结算。
- [x] **商店系统 | Shop System**
    - *逻辑:* 后端处理 `BUY_ITEM_REQ`。
    - *限制:* 只能购买当前 Phase 对应 Tier 的物品。

### 2. 前端：交互与反馈 (Frontend: UI & Feedback)
- [ ] **交互进度条 | Interaction Progress Bar**
    - *目标:* 为破译电机和激活撤离点添加环形或长条进度反馈。
- [x] **资金面板美化 | Funds Panel**
    - *目标:* HUD 已显示资金。
- [x] **商店界面 | Shop UI**
    - *实现:* 按 B 打开黑市界面，支持购买基础道具。
- [x] **开发者模式 | Developer Mode**
    - *实现:* 设置中开启。
    - *功能:* F9 跳过阶段，去雾高亮。
    - *修正:* F9 跳过阶段后，雷达脉冲立即生效。

### 3. 协议与连接 (Protocol & Connection)
- [x] **玩家名称输入 | Player Name Input**
    - *前端:* 启动时请求用户输入名称。
    - *协议:* 增加了 `LOGIN_REQ` 处理逻辑。
    - *后端:* 绑定 SessionID 与 Name，用于数据库存储。

## 📅 Sprint 4: 性能优化与重构 (Optimization & Refactoring)

### 1. 架构解耦 (Architecture)
- [x] **逻辑与网络分离 | Decouple Logic from Network**
    - *重构:* 将 `Room` 拆分为 `GameLoop` (Simulation) 和 `NetworkManager`。
    - *实现:* 创建了 `GameLoop`，使用 Channel 通信。

### 2. 物理与碰撞 (Physics)
- [x] **高级碰撞判定 | Advanced Collision**
    - *升级:* 从网格判定升级为 AABB 或 圆形碰撞判定。
    - *参数:* 严格执行 0.5 半径。
    - *实现:* `physics.go` 实现了 `ResolveMovement` (Circle-AABB with Sliding)。

### 3. 协议升级 (Protocol Migration)
- [ ] **Protobuf 迁移 | Protobuf Migration**
    - *定义:* 编写 `.proto` 文件。
    - *替换:* 替换 JSON 序列化，优化带宽。