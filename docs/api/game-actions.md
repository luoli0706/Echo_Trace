# 游戏操作 API

游戏过程中的玩家操作 API，包括移动、射击、道具使用等。这些操作使用 **Protobuf 二进制协议**。

---

## C2S: 客户端到服务器消息

### MOVE_REQ (2001)

玩家移动指令。

**消息体**: `MoveInput`

```protobuf
message MoveInput {
  float dir_x = 1;     // 移动方向 X 分量 (-1.0 ~ 1.0)
  float dir_y = 2;     // 移动方向 Y 分量 (-1.0 ~ 1.0)
  float look_x = 3;    // 视野方向 X 分量 (归一化)
  float look_y = 4;    // 视野方向 Y 分量 (归一化)
}
```

**字段说明**:
- `dir_x`, `dir_y`: 移动方向向量（WASD 输入转换）
  - W: (0, -1)
  - S: (0, 1)
  - A: (-1, 0)
  - D: (1, 0)
  - 斜向移动需归一化（如 WD: (0.707, -0.707)）
- `look_x`, `look_y`: 视野朝向（通常由鼠标位置计算）

**示例 (Python)**:

```python
import math
from proto import echo_trace_pb2 as pb

# 计算移动方向
dir_x, dir_y = 0.0, 0.0
if keys[pygame.K_w]: dir_y -= 1.0
if keys[pygame.K_s]: dir_y += 1.0
if keys[pygame.K_a]: dir_x -= 1.0
if keys[pygame.K_d]: dir_x += 1.0

# 归一化
length = math.sqrt(dir_x**2 + dir_y**2)
if length > 0:
    dir_x /= length
    dir_y /= length

# 计算视野方向
mouse_x, mouse_y = pygame.mouse.get_pos()
look_x = mouse_x - player_x
look_y = mouse_y - player_y
look_length = math.sqrt(look_x**2 + look_y**2)
if look_length > 0:
    look_x /= look_length
    look_y /= look_length

# 发送移动指令
move_input = pb.MoveInput(
    dir_x=dir_x, dir_y=dir_y,
    look_x=look_x, look_y=look_y
)
envelope = pb.Envelope(
    type=2001,
    payload=move_input.SerializeToString()
)
await ws.send(envelope.SerializeToString())
```

---

### FIRE_REQ (2010)

开火指令（无参数）。

**消息体**: 无

**示例 (Python)**:

```python
envelope = pb.Envelope(
    type=2010,
    payload=b""  # 空载荷
)
await ws.send(envelope.SerializeToString())
```

**限制**:
- 需要消耗弹药（默认弹夹容量 10 发）
- 射击冷却时间（默认 0.5 秒）
- 打空后自动装填（装填时间 2 秒）

---

### USE_ITEM_REQ (2002)

使用道具栏中的道具。

**消息体**: `UseItemInput`

```protobuf
message UseItemInput {
  int32 slot = 1;  // 道具栏位置 (0-5)
}
```

**示例 (Python)**:

```python
# 使用物品栏第 1 个道具
use_item = pb.UseItemInput(slot=0)
envelope = pb.Envelope(
    type=2002,
    payload=use_item.SerializeToString()
)
await ws.send(envelope.SerializeToString())
```

**限制**:
- 道具栏范围: 0-5（6 个槽位）
- 使用后道具自动消失（消耗型道具）
- 部分道具有冷却时间

---

### DROP_REQ (2005)

丢弃道具栏中的道具。

**消息体**: `DropInput`

```protobuf
message DropInput {
  int32 slot = 1;  // 道具栏位置 (0-5)
}
```

**示例 (Python)**:

```python
# 丢弃物品栏第 3 个道具
drop_input = pb.DropInput(slot=2)
envelope = pb.Envelope(
    type=2005,
    payload=drop_input.SerializeToString()
)
await ws.send(envelope.SerializeToString())
```

**效果**:
- 道具掉落在玩家当前位置
- 其他玩家可以拾取
- 丢弃的道具槽位变为空

---

### CHOOSE_TACTIC_REQ (2006)

Phase 0 阶段选择战术倾向。

**消息体**: `TacticInput`

```protobuf
message TacticInput {
  string tactic = 1;  // "RECON" | "DEFENSE" | "TRAP"
}
```

**战术效果**:
- `RECON`: 视野半径 +20%
- `DEFENSE`: 最大生命值 +20%
- `TRAP`: 护甲穿透 +20%

**示例 (Python)**:

```python
# 选择侦察倾向
tactic_input = pb.TacticInput(tactic="RECON")
envelope = pb.Envelope(
    type=2006,
    payload=tactic_input.SerializeToString()
)
await ws.send(envelope.SerializeToString())
```

**限制**:
- 仅在 Phase 0 有效
- 每局游戏只能选择一次
- 所有玩家完成选择后自动进入 Phase 1

---

### BUY_REQ (2007)

从商人处购买道具。

**消息体**: `BuyInput`

```protobuf
message BuyInput {
  int32 item_index = 1;  // 商店道具索引 (0-5)
}
```

**示例 (Python)**:

```python
# 购买商店第 1 个道具
buy_input = pb.BuyInput(item_index=0)
envelope = pb.Envelope(
    type=2007,
    payload=buy_input.SerializeToString()
)
await ws.send(envelope.SerializeToString())
```

**限制**:
- 必须在商人附近（5 米以内）
- 需要足够的金币
- 道具栏必须有空位

---

### SELL_REQ (2008)

向商人出售道具。

**消息体**: `SellInput`

```protobuf
message SellInput {
  int32 slot = 1;  // 道具栏位置 (0-5)
}
```

**示例 (Python)**:

```python
# 出售物品栏第 2 个道具
sell_input = pb.SellInput(slot=1)
envelope = pb.Envelope(
    type=2008,
    payload=sell_input.SerializeToString()
)
await ws.send(envelope.SerializeToString())
```

**价格**:
- T1: 10 金币
- T2: 25 金币
- T3: 50 金币
- T4: 100 金币

---

## 操作流程示例

### 完整游戏循环

```python
import asyncio
import pygame
from proto import echo_trace_pb2 as pb

async def game_loop(websocket):
    clock = pygame.time.Clock()
    running = True
    
    while running:
        # 处理输入
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            elif event.type == pygame.KEYDOWN:
                # 使用道具 (1-6 键)
                if pygame.K_1 <= event.key <= pygame.K_6:
                    slot = event.key - pygame.K_1
                    await use_item(websocket, slot)
                
                # 开火 (Space)
                elif event.key == pygame.K_SPACE:
                    await fire(websocket)
                
                # 交互 (F 键)
                elif event.key == pygame.K_f:
                    await interact(websocket)
        
        # 发送移动指令（每帧）
        await send_move(websocket)
        
        # 接收状态快照
        await receive_state(websocket)
        
        clock.tick(60)  # 60 FPS

async def use_item(ws, slot):
    use_item = pb.UseItemInput(slot=slot)
    envelope = pb.Envelope(
        type=2002,
        payload=use_item.SerializeToString()
    )
    await ws.send(envelope.SerializeToString())

async def fire(ws):
    envelope = pb.Envelope(type=2010, payload=b"")
    await ws.send(envelope.SerializeToString())

async def send_move(ws):
    # 计算移动和视野方向（省略细节）
    dir_x, dir_y, look_x, look_y = calculate_directions()
    
    move_input = pb.MoveInput(
        dir_x=dir_x, dir_y=dir_y,
        look_x=look_x, look_y=look_y
    )
    envelope = pb.Envelope(
        type=2001,
        payload=move_input.SerializeToString()
    )
    await ws.send(envelope.SerializeToString())
```

---

## 错误处理

服务器可能返回的错误：

| 错误码 | 名称 | 说明 |
|--------|------|------|
| 4004 | PLAYER_DEAD | 玩家已死亡，无法操作 |
| 4005 | INVALID_SLOT | 道具栏位置无效 |
| 4006 | ITEM_NOT_FOUND | 道具不存在 |
| 4007 | COOLDOWN_ACTIVE | 道具或武器冷却中 |
| 4008 | INSUFFICIENT_AMMO | 弹药不足 |
| 4009 | NOT_IN_RANGE | 不在交互范围内 |

**示例错误响应 (JSON)**:

```json
{
  "type": "ERROR",
  "error_code": 4007,
  "message": "武器冷却中，请等待 0.5 秒"
}
```

---

## 性能优化建议

### 1. 批量发送移动指令

避免每帧发送，降低到 20-30 次/秒：

```python
last_move_time = 0
MOVE_INTERVAL = 0.05  # 50ms

current_time = time.time()
if current_time - last_move_time >= MOVE_INTERVAL:
    await send_move(websocket)
    last_move_time = current_time
```

### 2. 预测性客户端

客户端立即渲染移动结果，等待服务器确认：

```python
# 客户端预测
player.x += dir_x * speed * dt
player.y += dir_y * speed * dt

# 服务器确认后校正
if server_pos != predicted_pos:
    player.x = lerp(player.x, server_pos.x, 0.5)
    player.y = lerp(player.y, server_pos.y, 0.5)
```

### 3. 输入缓冲

延迟发送非关键操作，避免网络拥塞：

```python
input_buffer = []

# 缓冲输入
input_buffer.append(("use_item", 0))
input_buffer.append(("drop", 2))

# 批量发送
async def flush_buffer(ws):
    for action, param in input_buffer:
        await send_action(ws, action, param)
    input_buffer.clear()
```

---

## 相关文档

- [WebSocket API 概述](/api/) - 连接管理
- [状态快照 API](/api/state-snapshot.html) - 服务器推送的状态数据
- [Protobuf 协议](/protobuf/) - 消息格式详解
- [游戏阶段](/game-logic/phases.html) - Phase 0-4 流程
