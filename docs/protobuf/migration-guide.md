# JSON → Protobuf 迁移指南

本指南帮助开发者将现有的 JSON 客户端代码迁移到 Protobuf 协议。

## 为什么迁移？

### 性能收益

| 指标 | JSON (旧版) | Protobuf (新版) | 提升 |
|------|------------|----------------|------|
| 单次移动消息 | ~100 bytes | ~20 bytes | **80%** |
| 单次状态快照 | ~2KB | ~500B | **75%** |
| 序列化时间 | ~0.5ms | ~0.1ms | **80%** |
| 10 玩家带宽 | 400KB/s | 100KB/s | **75%** |

### 向后兼容

- **房间管理仍使用 JSON**：`CREATE_ROOM`, `JOIN_ROOM`, `LIST_ROOMS`, `LEAVE_ROOM`
- **双协议并存**：服务器同时支持 JSON 和 Protobuf 消息
- **渐进式迁移**：可以逐步迁移而不破坏现有代码

---

## 迁移步骤

### 1. 安装依赖

#### Python 客户端

```bash
pip install protobuf==6.33.3
```

#### JavaScript 客户端

```bash
npm install protobufjs@^7.0.0
```

#### Go 服务器

```bash
go get google.golang.org/protobuf@v1.36.11
```

---

### 2. 生成代码

#### Python

```bash
protoc --python_out=. echo_trace.proto
```

生成 `echo_trace_pb2.py`。

#### JavaScript

```bash
npx pbjs -t static-module -w commonjs -o echo_trace.js echo_trace.proto
npx pbts -o echo_trace.d.ts echo_trace.js
```

#### Go

```bash
protoc --go_out=. --go_opt=paths=source_relative echo_trace.proto
```

---

### 3. 代码迁移

### Python 客户端

#### 旧代码（JSON）

```python
# 发送移动指令
payload = {
    "dir": {"x": 1.0, "y": 0.0},
    "look_dir": {"x": 0.707, "y": 0.707}
}
message = json.dumps({"type": 2001, "payload": payload})
ws.send(message)  # Text Frame
```

#### 新代码（Protobuf）

```python
from proto import echo_trace_pb2 as pb

# 发送移动指令
move_input = pb.MoveInput(
    dir=pb.Vector2(x=1.0, y=0.0),
    look_dir=pb.Vector2(x=0.707, y=0.707),
    has_look_dir=True
)

envelope = pb.Envelope(
    type=2001,
    payload=move_input.SerializeToString()
)

ws.send(envelope.SerializeToString(), binary=True)  # Binary Frame
```

#### 接收消息（兼容处理）

```python
def on_message(ws, message):
    # 检测消息类型
    if isinstance(message, bytes):
        # Protobuf 消息
        envelope = pb.Envelope()
        envelope.ParseFromString(message)
        
        if envelope.type == 3001:  # STATE_SNAPSHOT
            snapshot = pb.StateSnapshot()
            snapshot.ParseFromString(envelope.payload)
            handle_snapshot(snapshot)
    else:
        # JSON 消息（房间管理）
        data = json.loads(message)
        handle_json_message(data)
```

---

### JavaScript 客户端

#### 旧代码（JSON）

```javascript
// 发送移动指令
const payload = {
    dir: { x: 1.0, y: 0.0 },
    look_dir: { x: 0.707, y: 0.707 }
};
const message = JSON.stringify({ type: 2001, payload });
ws.send(message);  // Text Frame
```

#### 新代码（Protobuf）

```javascript
const protobuf = require('protobufjs');
const root = protobuf.loadSync('echo_trace.proto');

const MoveInput = root.lookupType('echo_trace.MoveInput');
const Envelope = root.lookupType('echo_trace.Envelope');

// 发送移动指令
const moveInput = MoveInput.create({
    dir: { x: 1.0, y: 0.0 },
    lookDir: { x: 0.707, y: 0.707 },
    hasLookDir: true
});

const envelope = Envelope.create({
    type: 2001,
    payload: MoveInput.encode(moveInput).finish()
});

ws.send(Envelope.encode(envelope).finish());  // Binary Frame
```

---

### Go 服务器

#### 旧代码（JSON）

```go
// 解析移动指令
var msg map[string]interface{}
json.Unmarshal(data, &msg)

payload := msg["payload"].(map[string]interface{})
dirX := payload["dir"].(map[string]interface{})["x"].(float64)
dirY := payload["dir"].(map[string]interface{})["y"].(float64)
```

#### 新代码（Protobuf）

```go
import pb "echo_trace_server/proto"
import "google.golang.org/protobuf/proto"

// 解析移动指令
var envelope pb.Envelope
proto.Unmarshal(data, &envelope)

if envelope.Type == 2001 {
    var moveInput pb.MoveInput
    proto.Unmarshal(envelope.Payload, &moveInput)
    
    dirX := moveInput.Dir.X
    dirY := moveInput.Dir.Y
}
```

---

## 消息类型映射

### C2S 消息

| JSON 字段 | Protobuf 字段 | 类型变化 |
|-----------|--------------|---------|
| `type` | `Envelope.type` | `string` → `int32` |
| `payload.dir` | `MoveInput.dir` | `object` → `Vector2` |
| `payload.slot_index` | `UseItemInput.slot_index` | `number` → `int32` |
| `payload.tactic` | `TacticInput.tactic` | `string` → `string` |

### S2C 消息

| JSON 字段 | Protobuf 字段 | 类型变化 |
|-----------|--------------|---------|
| `phase` | `StateSnapshot.phase` | `number` → `int32` |
| `my_state.pos` | `PlayerSnapshot.pos` | `object` → `Vector2` |
| `entities[].uid` | `EntitySnapshot.uid` | `string` → `string` |
| `events[].type` | `GameEvent.type` | `string` → `string` |

---

## 常见问题

### Q1: 房间管理为什么还用 JSON？

**A**: 房间管理操作（创建、加入、列出、离开）频率低，可读性优先。使用 JSON 方便调试和 Web UI 集成。

---

### Q2: 如何处理双协议兼容？

**A**: 服务器通过 WebSocket 消息类型区分：
- **Text Frame**：JSON 消息（房间管理）
- **Binary Frame**：Protobuf 消息（游戏数据）

客户端检测 `message` 类型：
```python
if isinstance(message, bytes):
    # Protobuf
else:
    # JSON
```

---

### Q3: 如何调试 Protobuf 消息？

**方法 1**: 使用 `protobuf-inspector`

```bash
pip install protobuf-inspector
protobuf-inspector < message.bin
```

**方法 2**: 转换为 JSON

```python
from google.protobuf.json_format import MessageToJson

snapshot = pb.StateSnapshot()
snapshot.ParseFromString(data)
print(MessageToJson(snapshot))
```

---

### Q4: Protobuf 消息大小如何估算？

**规则**：
- 每个字段：1-2 bytes（tag + wire type）
- `int32`：1-5 bytes（varint 编码）
- `double`：8 bytes
- `string`：1 + length bytes
- `repeated`：每个元素重复以上开销

**示例**（MoveInput）：
```
dir.x (8) + dir.y (8) + look_dir.x (8) + look_dir.y (8) + has_look_dir (1) = ~33 bytes
```

---

### Q5: 如何处理协议版本不兼容？

**策略**：
1. 使用 `Envelope.version` 字段（预留）
2. 服务器检测版本号，拒绝旧版本连接
3. 客户端显示"请更新"提示

```protobuf
message Envelope {
  int32 version = 3;  // 协议版本
  // ...
}
```

---

## 性能优化建议

### 1. 减少消息发送频率

#### 旧代码（每帧发送）

```python
def game_loop():
    while running:
        send_move(dir_x, dir_y, look_x, look_y)  # 60 FPS = 3600 msg/min
        time.sleep(1/60)
```

#### 新代码（仅在变化时发送）

```python
last_dir = None
last_look = None

def game_loop():
    while running:
        current_dir = (dir_x, dir_y)
        current_look = (look_x, look_y)
        
        if current_dir != last_dir or current_look != last_look:
            send_move(dir_x, dir_y, look_x, look_y)
            last_dir = current_dir
            last_look = current_look
        
        time.sleep(1/60)
```

---

### 2. 批量发送消息

```python
# 将多个操作合并到一个 Envelope
commands = []
commands.append(create_move_message())
commands.append(create_fire_message())

# 发送批量消息（需服务器支持）
batch = pb.BatchMessage(commands=commands)
ws.send(batch.SerializeToString())
```

---

### 3. 压缩 StateSnapshot

服务器端：
```go
import "github.com/golang/snappy"

data := proto.Marshal(snapshot)
compressed := snappy.Encode(nil, data)
ws.WriteMessage(websocket.BinaryMessage, compressed)
```

客户端：
```python
import snappy

compressed_data = ws.recv()
data = snappy.decompress(compressed_data)
snapshot.ParseFromString(data)
```

---

## 迁移检查清单

- [ ] 安装 Protobuf 依赖
- [ ] 生成客户端代码
- [ ] 替换移动指令（MOVE_REQ）
- [ ] 替换射击指令（FIRE_REQ）
- [ ] 替换道具使用（USE_ITEM_REQ）
- [ ] 替换 StateSnapshot 解析
- [ ] 保留 JSON 房间管理
- [ ] 添加双协议兼容检测
- [ ] 测试完整游戏流程
- [ ] 性能测试（带宽、延迟）
- [ ] 调试工具集成

---

## 示例代码仓库

完整示例代码见：
- Python 客户端：`frontend/client/network.py`
- Go 服务器：`backend/network/client.go`
- Protobuf 定义：`proto/echo_trace.proto`

---

## 下一步

- [消息类型](/protobuf/message-types.html) - 完整消息类型目录
- [游戏操作 API](/api/game-actions.html) - C2S 消息详细说明
- [状态快照 API](/api/state-snapshot.html) - StateSnapshot 结构详解
