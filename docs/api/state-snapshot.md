# 状态快照 API

服务器每 50ms 向客户端推送游戏状态快照 (StateSnapshot)，包含玩家、实体、游戏阶段等信息。

::: warning 当前协议
v1.3.0 状态快照仍使用 **JSON 格式**，Protobuf 迁移计划在 v1.4.0 完成。
:::

---

## S2C: 服务器到客户端消息

### StateSnapshot 结构

```json
{
  "phase": 1,
  "phase_remaining_time": 285.5,
  "players": [...],
  "entities": [...],
  "radar_active": false,
  "blips": [],
  "shrink_circle": null
}
```

---

## 字段说明

### phase

当前游戏阶段

- **类型**: `int`
- **取值**:
  - `0`: Phase 0 - 战术选择
  - `1`: Phase 1 - 搜索阶段
  - `2`: Phase 2 - 冲突阶段
  - `3`: Phase 3 - 逃离阶段
  - `4`: Phase 4 - 游戏结束

### phase_remaining_time

当前阶段剩余时间（秒）

- **类型**: `float`
- **示例**: `285.5` (还剩 4 分 45.5 秒)

---

### players

所有玩家的状态数组

**PlayerState 结构**:

```json
{
  "id": "player-uuid",
  "name": "Player1",
  "x": 50.0,
  "y": 30.0,
  "hp": 80,
  "max_hp": 100,
  "armor": 20,
  "angle": 45.0,
  "look_angle": 60.0,
  "view_radius": 15.0,
  "is_dead": false,
  "tactic": "RECON",
  "inventory": [...]
}
```

**字段详解**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 玩家唯一标识符 (UUID) |
| `name` | string | 玩家昵称 |
| `x`, `y` | float | 世界坐标位置 |
| `hp` | int | 当前生命值 |
| `max_hp` | int | 最大生命值（受战术加成影响） |
| `armor` | int | 当前护甲值 |
| `angle` | float | 坦克履带朝向角度（度数，0-360） |
| `look_angle` | float | 炮塔朝向角度（度数，0-360） |
| `view_radius` | float | 视野半径（米） |
| `is_dead` | bool | 是否死亡 |
| `tactic` | string | 战术倾向 (`"RECON"` / `"DEFENSE"` / `"TRAP"` / `null`) |
| `inventory` | array | 道具栏（见下文） |

**inventory 道具栏**:

```json
[
  {
    "name": "超级穿甲弹",
    "tier": 1,
    "category": "offense"
  },
  null,  // 空槽位
  {
    "name": "绝对防御",
    "tier": 2,
    "category": "survival"
  },
  null,
  null,
  null
]
```

---

### entities

所有实体（子弹、物品、AI）的状态数组

**Entity 类型**:

#### 1. 子弹 (Bullet)

```json
{
  "type": "bullet",
  "id": "bullet-123",
  "x": 55.0,
  "y": 32.0,
  "vel_x": 10.0,
  "vel_y": 5.0,
  "owner_id": "player-uuid",
  "damage": 25,
  "armor_penetration": 0.5
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `"bullet"` |
| `id` | string | 子弹唯一标识符 |
| `x`, `y` | float | 当前位置 |
| `vel_x`, `vel_y` | float | 速度向量（米/秒） |
| `owner_id` | string | 射击者 ID |
| `damage` | int | 基础伤害 |
| `armor_penetration` | float | 护甲穿透率 (0.0-1.0) |

#### 2. 物品 (Item)

```json
{
  "type": "item",
  "id": "item-456",
  "x": 60.0,
  "y": 40.0,
  "item_name": "光学折射效应",
  "tier": 2,
  "category": "offense"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `"item"` |
| `id` | string | 物品唯一标识符 |
| `x`, `y` | float | 地面位置 |
| `item_name` | string | 道具名称 |
| `tier` | int | 稀有度等级 (1-4) |
| `category` | string | 道具类别（`offense/survival/recon/utility`） |

#### 3. AI Boss (NingBye)

```json
{
  "type": "ai",
  "id": "ai-ningbye",
  "x": 70.0,
  "y": 50.0,
  "hp": 180,
  "max_hp": 200,
  "angle": 90.0,
  "state": "patrol"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `"ai"` |
| `id` | string | AI 唯一标识符 |
| `x`, `y` | float | 当前位置 |
| `hp`, `max_hp` | int | 生命值 |
| `angle` | float | 朝向角度 |
| `state` | string | 行为状态 (`patrol` / `chase` / `attack`) |

#### 4. 引擎 (Engine)

```json
{
  "type": "engine",
  "id": "engine-1",
  "x": 80.0,
  "y": 60.0,
  "is_repaired": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `"engine"` |
| `id` | string | 引擎唯一标识符 |
| `x`, `y` | float | 位置 |
| `is_repaired` | bool | 是否已修复 |

#### 5. 撤离点 (Evacuation)

```json
{
  "type": "evac",
  "id": "evac-1",
  "x": 90.0,
  "y": 70.0,
  "capacity": 5,
  "current_count": 2
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 固定为 `"evac"` |
| `id` | string | 撤离点唯一标识符 |
| `x`, `y` | float | 位置 |
| `capacity` | int | 容量上限 |
| `current_count` | int | 当前已撤离人数 |

---

### radar_active

雷达脉冲是否激活

- **类型**: `bool`
- **说明**: Phase 2/3 周期性激活（15 秒一次，持续 5 秒）
- **影响**: 激活时所有玩家位置暴露在小地图

### blips

雷达脉冲探测到的玩家位置

```json
[
  {"id": "player-uuid-1", "x": 50.0, "y": 30.0},
  {"id": "player-uuid-2", "x": 60.0, "y": 40.0}
]
```

- **类型**: `array`
- **说明**: 仅在 `radar_active = true` 时有数据
- **用途**: 在小地图上显示红点

### shrink_circle

Phase 3 缩圈信息

```json
{
  "center_x": 50.0,
  "center_y": 50.0,
  "current_radius": 30.0,
  "target_radius": 10.0,
  "damage_per_second": 10
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `center_x`, `center_y` | float | 安全区中心坐标 |
| `current_radius` | float | 当前安全区半径 |
| `target_radius` | float | 最终安全区半径 |
| `damage_per_second` | int | 圈外每秒伤害 |

- **类型**: `object` 或 `null`
- **说明**: Phase 3 时不为 `null`，其他阶段为 `null`

---

## 客户端处理示例

### Python 接收和解析

```python
import asyncio
import websockets
import json

async def receive_state(websocket):
    async for message in websocket:
        if isinstance(message, str):
            # JSON 状态快照
            snapshot = json.loads(message)
            
            # 更新游戏阶段
            current_phase = snapshot["phase"]
            time_left = snapshot["phase_remaining_time"]
            print(f"Phase {current_phase} - 剩余时间: {time_left:.1f}s")
            
            # 渲染玩家
            for player in snapshot["players"]:
                render_player(player)
            
            # 渲染实体
            for entity in snapshot["entities"]:
                if entity["type"] == "bullet":
                    render_bullet(entity)
                elif entity["type"] == "item":
                    render_item(entity)
                elif entity["type"] == "ai":
                    render_ai(entity)
            
            # 雷达脉冲
            if snapshot["radar_active"]:
                for blip in snapshot["blips"]:
                    draw_radar_blip(blip["x"], blip["y"])
            
            # 缩圈
            if snapshot["shrink_circle"]:
                draw_shrink_circle(snapshot["shrink_circle"])
```

### JavaScript 接收和解析

```javascript
ws.onmessage = (event) => {
  if (typeof event.data === 'string') {
    const snapshot = JSON.parse(event.data);
    
    // 更新 UI
    updatePhaseDisplay(snapshot.phase, snapshot.phase_remaining_time);
    
    // 渲染玩家
    snapshot.players.forEach(player => {
      renderPlayer(player);
    });
    
    // 渲染实体
    snapshot.entities.forEach(entity => {
      switch (entity.type) {
        case 'bullet':
          renderBullet(entity);
          break;
        case 'item':
          renderItem(entity);
          break;
        case 'ai':
          renderAI(entity);
          break;
      }
    });
    
    // 小地图雷达点
    if (snapshot.radar_active) {
      clearRadarBlips();
      snapshot.blips.forEach(blip => {
        addRadarBlip(blip.x, blip.y);
      });
    }
    
    // 缩圈警告
    if (snapshot.shrink_circle) {
      showShrinkCircleWarning(snapshot.shrink_circle);
    }
  }
};
```

---

## 性能优化

### 1. 增量更新（计划中）

当前每帧发送完整快照，v1.4.0 将支持增量更新：

```json
{
  "type": "delta",
  "updated_players": [
    {"id": "player-1", "x": 50.5, "hp": 75}
  ],
  "removed_entities": ["bullet-123", "item-456"]
}
```

### 2. 视野裁剪

客户端只渲染视野范围内的实体：

```python
def render_entities(entities, player_x, player_y, view_radius):
    for entity in entities:
        distance = math.sqrt((entity['x'] - player_x)**2 + (entity['y'] - player_y)**2)
        if distance <= view_radius:
            render(entity)
```

### 3. 插值平滑

避免位置更新跳变：

```python
def lerp(start, end, alpha):
    return start + (end - start) * alpha

# 平滑移动
player.x = lerp(player.x, snapshot['x'], 0.3)
player.y = lerp(player.y, snapshot['y'], 0.3)
```

---

## v1.4.0 Protobuf 迁移计划

### 新 StateSnapshot 定义

```protobuf
message StateSnapshot {
  int32 phase = 1;
  float phase_remaining_time = 2;
  repeated PlayerState players = 3;
  repeated Entity entities = 4;
  bool radar_active = 5;
  repeated RadarBlip blips = 6;
  optional ShrinkCircle shrink_circle = 7;
}

message PlayerState {
  string id = 1;
  string name = 2;
  float x = 3;
  float y = 4;
  int32 hp = 5;
  int32 max_hp = 6;
  int32 armor = 7;
  float angle = 8;
  float look_angle = 9;
  float view_radius = 10;
  bool is_dead = 11;
  optional string tactic = 12;
  repeated Item inventory = 13;
}
```

### 预期收益

| 指标 | JSON | Protobuf | 提升 |
|------|------|----------|------|
| 快照大小 (20 实体) | ~2000 bytes | ~500 bytes | 75% |
| 序列化时间 | ~0.8 ms | ~0.2 ms | 75% |
| 带宽占用 (20 Hz) | 40 KB/s | 10 KB/s | 75% |

---

## 相关文档

- [游戏操作 API](/api/game-actions.html) - 客户端发送的操作指令
- [Protobuf 协议](/protobuf/) - 消息格式详解
- [游戏阶段](/game-logic/phases.html) - Phase 0-4 详细说明
