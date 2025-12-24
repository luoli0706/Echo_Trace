# Types / Schema 评审与改进方向（面向 Beta）

> **范围**：核心数据结构（Player/Entity/Item/Config）与协议/序列化一致性  
> **对应版本**：Echo Trace Alpha 0.5 → Beta 1.0

---

## 1. 当前实现：`types.go` 的真实角色

相关代码：
- [backend/logic/types.go](../backend/logic/types.go)
- [backend/main.go](../backend/main.go)（config Unmarshal）
- [backend/logic/gamestate.go](../backend/logic/gamestate.go)（快照/逻辑使用）

`types.go` 定义了：

- 数学/坐标：`Vector2`
- 世界实体：`Entity`（含 `Extra interface{}`）
- 玩家：`Player`
- 道具：`Item`
- 配置：`GameConfig`（仅覆盖 config 的一小部分字段）

---

## 2. 主要问题与风险

### 2.1 配置结构与 `game_config.json` 不一致（高优先级）

现状：`GameConfig` 只定义了少数字段（tick、maxPlayers、map尺寸、基础速度/视野、phase1/phase2 duration + motors_spawn_count）。

但 `game_config.json` 中还有大量字段（inventory_size、safe_slot_count、hear_radius、weight thresholds、combat、phase3 等）。由于 Go 的 JSON 反序列化会忽略未知字段，所以不会报错，但会造成：

- **调参不生效**（例如库存大小、听觉半径等仍在代码中硬编码）
- **设计与实现脱节**，平衡迭代变得不可控

这与 beta 方案里提到的“魔法数字/配置硬编码”是同一个根因。

### 2.2 `Entity.Extra interface{}` 带来的类型安全与序列化风险

- 逻辑层大量依赖 `ent.Extra.(MotorData)`、`.(Item)`、`.(SupplyDropData)`。
- 一旦 Extra 被误写、未来引入新的实体类型，或者后续做协议升级/重放/录制，就会出现：
  - 运行时 panic
  - 难以版本化（schema evolution 困难）

### 2.3 Player/Entity 作为快照 payload 的耦合问题

- `GetSnapshot` 直接把 `*Player` 作为 `self` 返回。
- `Player` 同时承担“服务端内部状态”（例如 TargetDir/Velocity）与“网络对外状态”。目前通过 `json:"-"` 屏蔽了一部分，但长期会越来越难控。

### 2.4 事件/枚举缺少强类型

- entity type、phase 等用 string/int 混合，且协议里 type code 也用 magic number。
- 对未来 Protobuf 迁移不友好。

---

## 3. 改进方向

### 3.1 Direction A：把 `GameConfig` 补齐并消除硬编码（P0/P1）

- 让 `GameConfig` 覆盖 `game_config.json` 的全部字段（至少覆盖当前已经在 JSON 中出现的字段）。
- 逻辑层的硬编码（inventory size、hear radius、phase3 等）统一改为读取 config。

收益：
- 配置成为唯一事实来源（single source of truth）
- 支持 beta 方案的平衡调参与快速迭代

### 3.2 Direction B：分离“内部模型”和“网络 DTO”（P1/P2）

- 定义 `SnapshotSelfDTO`、`SnapshotPlayerDTO`、`SnapshotEntityDTO`，只包含网络需要的字段。
- 内部 `Player`/`Entity` 保留服务端计算需要的字段，不再直接作为 JSON payload。

收益：
- 降低协议改动对逻辑的影响
- 更容易引入 delta 快照、压缩与版本化

### 3.3 Direction C：替换 `Entity.Extra`（P1/P2）

推荐两种方向：

1. **显式 payload 结构**：
   - `type Entity struct { ... Motor *MotorData; Item *Item; SupplyDrop *SupplyDropData }`（只会有一个非空）
2. **RawMessage + 版本化**：
   - `Extra json.RawMessage`，并按 `Type` 解码到不同结构（支持 schema version）

### 3.4 Direction D：为 Protobuf 迁移做准备（P2/P3）

- 把 phase、entity type、消息 type code 统一到 enum
- 为 schema 增加版本字段（例如 `snapshot_version`）

---

## 4. 任务拆分与优先级

### P0：关键一致性修复

1. 补齐 `GameConfig` 至少覆盖现有 `game_config.json` 字段
2. 把 inventory/hearRadius/phase3 等关键参数改为 config 驱动

### P1：核心可维护性提升

1. DTO 化快照输出，避免直接序列化内部对象
2. 替换 `Entity.Extra interface{}` 的强断言使用方式

### P2：长期演进准备

1. enum 化与 schema version
2. 与 Protobuf 迁移方案合并推进

---

## 5. 验收指标（Success Metrics）

- **一致性**：`game_config.json` 的关键参数调整能在运行时体现（至少不需要改代码）
- **稳定性**：实体 Extra 不会因类型不匹配导致 panic
- **可维护性**：协议字段变更只影响 DTO 层，不影响核心仿真逻辑
