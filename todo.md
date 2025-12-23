# Echo Trace 开发计划 | Development Plan

> **状态 (Status):** Sprint 3 进行中 (Sprint 3 In Progress)
> **目标 (Objective):** Sprint 3 - 经济体系与博弈闭环 (Economy & Game Loop)

## 📅 Sprint 3: 经济体系与博弈闭环 (Economy & Game Loop)

### 1. 后端：经济结算与持久化 (Backend: Settlement & Persistence)
- [x] **高价值空投逻辑 | High-Value Supply Drops**
    - *优化:* 已实现重心刷新和物品生成。
    - *雷达:* 已实现。
- [x] **撤离处理 | Process Extraction**
    - *实现:* 在 `ProcessExtraction` 中实现了基础的 "物品 -> 资金" 转换逻辑。
    - *逻辑:* 撤离时清空背包并保存资金。
- [x] **SQLite 持久化层 | SQLite Persistence Layer**
    - *基础:* 已集成 SQLite。
    - *任务:* 玩家断开连接或撤离时保存数据。
    - *任务:* 玩家登录 (`SetPlayerName`) 时加载数据。

### 2. 前端：交互与反馈 (Frontend: UI & Feedback)
- [ ] **交互进度条 | Interaction Progress Bar**
    - *目标:* 为破译电机和激活撤离点添加环形或长条进度反馈。
- [x] **资金面板美化 | Funds Panel**
    - *目标:* HUD 已显示资金。

### 3. 协议与连接 (Protocol & Connection)
- [x] **玩家名称输入 | Player Name Input**
    - *前端:* 启动时请求用户输入名称。
    - *协议:* 增加了 `LOGIN_REQ` 处理逻辑。
    - *后端:* 绑定 SessionID 与 Name，用于数据库存储。

## 📅 Sprint 4: 性能优化与重构 (Optimization & Refactoring)

### 1. 架构解耦 (Architecture)
- [ ] **逻辑与网络分离 | Decouple Logic from Network**
    - *重构:* 将 `Room` 拆分为 `GameLoop` (Simulation) 和 `NetworkManager`。

### 2. 物理与碰撞 (Physics)
- [ ] **高级碰撞判定 | Advanced Collision**
    - *升级:* 从网格判定升级为 AABB 或 圆形碰撞判定。
    - *参数:* 严格执行 0.5 半径。

### 3. 协议升级 (Protocol Migration)
- [ ] **Protobuf 迁移 | Protobuf Migration**
    - *定义:* 编写 `.proto` 文件。
    - *替换:* 替换 JSON 序列化，优化带宽。
