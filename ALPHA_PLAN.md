# Echo Trace 1.0-alpha 重构计划

## 1. 核心目标
将游戏从纯生存撤离玩法转型为 **坦克大战（Tank Battle）** + **生存撤离** 的混合模式。引入战斗、护甲属性，并将游戏进程由“时间驱动”改为“击杀/时间混合驱动”。

## 2. 属性重构 (Backend & Protocol)

### 2.1 玩家属性 (Player Stats)
在 `backend/logic/types.go` 的 `Player` 结构体中新增：
- **HP**: 保持 100 点。
- **Armor (新增)**:
    - 初始值: 50 点（可通过配置调整）。
    - 机制: 受到伤害时优先扣除护甲。
- **Kills (新增)**:
    - 记录单局击杀数，用于结算排行。

### 2.2 游戏状态 (GameState)
在 `GameState` 中新增：
- **TotalKills**: 全场总击杀数，用于驱动阶段流转。
- **MatchTimer**: 总游戏时长计时器（可选，或沿用各阶段倒计时）。

### 2.3 配置文件 (GameConfig)
更新 `game_config.json` 支持：
- `Gameplay.BaseArmor`: 默认 50。
- `Combat.BulletDamage`: 默认 20。
- `Phases.Thresholds`:
    - `Phase2_Kills`: 5
    - `Phase3_Kills`: 15
    - `End_Kills`: 30

## 3. 战斗系统 (Combat System)

### 3.1 射击机制
- **输入**: 玩家按下 `SPACE` (空格键)。
- **弹道**:
    - 方案 A (Alpha版): 射线检测 (Raycast/Hitscan)。射程固定（如 15-20 单位）。
    - 判定: 检测视线方向 (`LookDir`) 上的第一个玩家实体。
- **伤害计算**:
    - 单发伤害: 20 点。
    - 结算顺序: 护甲 -> 血量。
    - 反馈: 击中提示、扣血/甲同步。

### 3.2 击杀与死亡
- 当玩家 HP <= 0:
    - 判定死亡。
    - 攻击者 `Kills + 1`。
    - 全场 `TotalKills + 1`。
    - 广播击杀信息 (`PLAYER_KILLED`)。

## 4. 游戏流程与胜利条件 (Game Loop & Win Conditions)

### 4.1 阶段流转 (Phase Transition)
修改 `logic/loop.go` 或 `gamestate.go` 中的 `UpdateTick`：
- **Phase 1 -> 2**: 当 `TotalKills >= 5` (提前进入) 或 倒计时结束。
- **Phase 2 -> 3**: 当 `TotalKills >= 15` (提前进入) 或 倒计时结束/电机开启。
- **Game Over**:
    - `TotalKills >= 30`。
    - 倒计时结束 (Time Limit)。
    - 玩家成功撤离 (保留原有逻辑作为特殊胜利)。

### 4.2 结算 (End Game)
- 游戏结束时发送 `GAME_OVER_PUSH`。
- 内容: 积分榜 (Scoreboard)，包含所有玩家的击杀数、生存状态、排名。
- 房间关闭: 倒计时 60 秒后自动关闭，或房主手动关闭。

## 5. 前端适配 (Frontend)

- **控制**: 增加 `SPACE` 键监听 -> 发送 `FIRE_REQ`。
- **UI**:
    - 新增 **护甲条** (位于血条上方或下方，颜色区分)。
    - 新增 **击杀计数/阶段进度** (如: "Phase 1: 3/5 Kills").
    - 结算界面显示积分榜。

## 6. 开发步骤

1.  **Backend**: 修改 `Player` 结构体，添加 Armor/Kills 字段。
2.  **Backend**: 实现 `C2S_FIRE_REQ` 协议处理与射线判定逻辑。
3.  **Backend**: 修改伤害逻辑，优先扣甲。
4.  **Backend**: 修改阶段流转逻辑，接入 `TotalKills` 判断。
5.  **Frontend**: 绑定按键，绘制 UI。
