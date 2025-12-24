# 回声追踪（暗林围猎）游戏流程设计 | Echo Trace Game Flow Design

> **注意 (Note):** 文档中的数值仅为示例，实际参数请参考 `game_config.json`。
> The values in this document are examples. Refer to `game_config.json` for actual parameters.

---

### 1. 初始化与匹配阶段 | Initialization & Matchmaking

这是房间实例从虚无中诞生的起点。
The starting point where room instances are born.

*   **服务端行为 (Server Behavior):**
    *   **迷宫生成 (Maze Gen):** 使用 Prim 算法生成 `32x32` 网格地图。
    *   **资源投放 (Spawning):** 随机撒下“宝藏点”(Item Drops) 和“电机点”(Motors)。在地图中心附近刷新“商人”(Merchant)。
    *   **玩家入场 (Player Entry):** 等待玩家连接。玩家需输入名称 (Name) 并登录。
    *   **大厅 (Lobby):** 玩家进入房间后，需选择战术 (Tactic: Recon/Defense/Trap)。
    *   **出生点 (Spawn):** 在安全区域随机分配出生点。

*   **阶段转换 (Transition):** 倒计时结束或全员准备就绪后，广播 `GAME_START`。

---

### 2. 第一阶段：暗影潜伏期 | Phase 1: Search

**持续时间 (Duration):** 120s
**氛围 (Vibe):** 黑暗、寂静、搜刮。 Dark, Silent, Scavenging.

*   **核心玩法 (Core Gameplay):**
    *   **搜刮 (Scavenge):** 寻找宝箱 (📦) 获取道具和少量资金。
    *   **交易 (Trade):** 寻找商人 (💰)，使用资金购买 T1 级装备（电击枪、急救包、扫描仪）。
    *   **空投 (Supply Drop):** 阶段开始时，地图中心刷新高价值空投 (🎁)，全图雷达可见。
*   **机制 (Mechanics):**
    *   **迷雾 (Fog):** 仅显示视野半径内的物体。
    *   **负重 (Weight):** 背包越重，移动速度越慢。
    *   **声音 (Sound):** 奔跑会产生脚步声波纹，被附近玩家侦测到。

---

### 3. 第二阶段：噪波冲突期 | Phase 2: Conflict

**持续时间 (Duration):** 180s
**氛围 (Vibe):** 紧张、争夺。 Tense, Competitive.

*   **阶段触发 (Trigger):** 倒计时归零。刷新新的 T2 级空投。
*   **核心玩法 (Core Gameplay):**
    *   **电机 (Motors):** 5 个电机 (⚡/M) 激活。玩家需修复电机。
    *   **脉冲 (Pulse):** 电机每 15 秒发出全图脉冲，暴露位置。
    *   **博弈 (Combat):** 使用 T2 级武器争夺电机控制权。
*   **目标 (Objective):** 修复至少 2 个电机以激活出口。

---

### 4. 第三阶段：终局撤离期 | Phase 3: Escape

**触发条件 (Trigger):** 2 个电机被修复。刷新 T3 级空投。
**氛围 (Vibe):** 绝望、冲刺。 Desperate, Rush.

*   **核心玩法 (Core Gameplay):**
    *   **出口 (Exit):** 出口 (🚪/E) 开启，位置全图广播。
    *   **撤离 (Extract):** 到达出口并保持交互 3 秒。
    *   **视野衰减 (Decay):** 玩家视野半径随时间逐渐缩小。
*   **名额限制 (Slots):** 仅限 2 人撤离。

---

### 5. 结算与清理 | Settlement

*   **胜利 (Win):** 成功撤离。
    *   **奖励:** 背包内所有物品按价值转换为资金 (Funds) 并保存到数据库。
*   **失败 (Loss):** 死亡或超时。
    *   **惩罚:** 丢失背包内物品（除安全箱外），资金不变。
*   **清理 (Cleanup):** 保存数据，断开连接，销毁房间。
