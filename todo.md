# Echo Trace Development Plan | 开发计划

> **Status:** Sprint 4 重构与优化中 (Optimization & Refactoring)
> **Objective:** 实现 docs/p0_recommendations.md 中的关键架构升级。

## 🚨 P0: Critical Architecture Decoupling | 关键架构解耦 (Current Focus)

- [x] **Refactor GameLoop | 重构游戏循环**
    - [x] Create `logic/loop.go` with `GameLoop` struct.
    - [x] Implement `Run()` with Ticker and Channel handling.
    - [x] Decouple `Room` from direct ticking.
- [x] **Actor Model for Input | 演员模型输入处理**
    - [x] Define `PlayerInput` struct.
    - [x] Replace direct method calls in `client.go` with `GameLoop.InputChan`.
- [x] **Snapshot Broadcasting | 快照广播**
    - [x] Implement `SnapshotChan` in `GameLoop`.
    - [x] Update `Room` to listen and broadcast snapshots.

## 🚨 P0: Physics Engine Upgrade | 物理引擎升级

- [x] **New Collision System | 新碰撞系统**
    - [x] Create `logic/physics.go`.
    - [x] Implement `CircleAABB` collision detection.
    - [x] Implement `ResolveMovement` with sliding vectors.
- [x] **Integrate Physics | 集成物理**
    - [x] Replace `UpdateTick` movement logic.
    - [x] Remove old `isWalkableWithRadius`.

## 🚨 P1: Protocol Optimization (Protobuf) | 协议优化

- [ ] **Define Schema | 定义 Schema**
    - [ ] Create `.proto` files for `InputEvent` and `StateSnapshot`.
- [ ] **Generate Code | 生成代码**
    - [ ] Setup `protoc` workflow.
- [ ] **Migrate Network | 迁移网络层**
    - [ ] Update `client.go` to use binary messages.
    - [ ] Update frontend to parse Protobuf.

## 📅 Sprint 3: Economy & Loop (Completed Items)

- [x] High-Value Supply Drops (Logic & Radar).
- [x] Process Extraction (Funds Settlement).
- [x] SQLite Persistence.
- [x] Shop System & UI.
- [x] Developer Mode.
- [x] Player Name Input.

## 📅 Sprint 4 Remaining Tasks

1.  **Verify Stability:** Run stress tests on new GameLoop.
2.  **Protobuf Migration:** Start defining `.proto` files.
