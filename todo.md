# Echo Trace 开发计划 | Development Plan

> **状态 (Status):** Alpha 版实现完成 (Alpha Implementation Complete)
> **目标 (Objective):** 使实现逻辑符合 `GameFlow.md` 和 `blueprint.md` 的设计期望。 (Align implementation with `GameFlow.md` and `blueprint.md`.)

## 🚨 核心差异修复 (Critical Gaps - High Priority)

### 1. 后端：游戏循环与机制 (Backend: Game Loop & Mechanics)
- [x] **实现阶段 0 (大厅/战术选择) | Implement Phase 0 (Lobby/Tactical Choice)**
    - 增加了 `PhaseInit` 状态。游戏现在会等待玩家发送 `CHOOSE_TACTIC_REQ` 战术选择请求后再开始。
- [x] **实现动态负重系统 | Implement Dynamic Weight System**
    - 实现了 `RecalculateStats`（重新计算属性）和 `Weight`（负重）逻辑。玩家的移动速度现在会随着背包物品重量增加而动态降低。
- [x] **实现声音/噪音传播 (LOD 第二层) | Implement Sound/Noise Propagation (LOD Layer 2)**
    - 在 `GetSnapshot` 中实现了 `HearRadius`（听觉半径）校验，并能为范围内的移动玩家生成 `FOOTSTEP`（脚步声）事件。
- [x] **高价值物资点 (空投) | High-Value Supply Drops**
    - *逻辑:* 每个阶段开始时，在玩家中心位置刷新物资箱。
    - *掉落:* 使用下一阶段的掉落权重。数量 1~3。
    - *信号:* 需要在雷达上持续显示。
- [x] **经济系统 | Economy System**
    - *数据:* 在 `Player` 结构体中增加 `Funds` 字段。
    - *获取:* 拾取物资时增加资金。

### 2. 前端：UI 与反馈 (Frontend: UI & Feedback)
- [x] **阶段与计时器 UI | Phase & Timer UI**
    - 在 `draw_hud` 中实现，现在顶部会显示当前游戏阶段和剩余时间。
- [ ] **交互进度 UI | Interaction Progress UI**
    - *当前进度:* 在 `renderer.py` 中实现了基础进度条，但仍需进一步美化。
- [x] **雷达/光点渲染 | Radar/Blip Rendering**
    - 实现了 `radar_blips`（雷达光点）的渲染。
    - 实现了声音指示器（波纹效果）的渲染，用于展示听到的脚步声方向。
    - *新增:* 需支持 `SUPPLY_DROP` 类型的雷达显示。
- [x] **大厅界面 | Lobby Interface**
    - 在 `renderer.py` 中实现了 `draw_lobby`（绘制大厅）功能，支持战术倾向选择。
- [x] **资金显示 UI | Funds Display UI**
    - 在 HUD 中实时显示当前资金。
- [x] **物理表现优化 | Visual Physics Optimization**
    - 将玩家渲染大小缩小为 0.5 格，以匹配新的碰撞判定。

## 🛠 重构任务 (Refactoring Tasks)
- [ ] **网络与逻辑解耦 | Decouple Network from Logic**
    - `Room` 结构体目前仍承担了部分逻辑，虽然在 Alpha 阶段可行，但未来建议迁移至 `GameServer`。
- [x] **AOI 优化 | AOI Optimization**
    - 视觉系统已采用 `AOIManager`。声音系统目前使用简单的距离校验（对于少量玩家已经足够）。
- [x] **碰撞判定优化 | Collision Logic Update**
    - 后端：将碰撞检测半径缩小至 0.5。

## 📅 开发路线图 (Roadmap)
1.  **第一阶段 (Sprint 1):** 阶段 0 实现 + 基础 UI。 (Phase 0 Implementation + Basic UI. - **Completed**)
2.  **第二阶段 (Sprint 2):** 负重与声音物理系统 + 雷达 UI。 (Weight & Sound Physics + Radar UI. - **Completed**)
3.  **第三阶段 (Sprint 3):** 经济系统、空投机制与物理优化。 (Economy, Supply Drops, and Physics Optimization. - **Completed**)
