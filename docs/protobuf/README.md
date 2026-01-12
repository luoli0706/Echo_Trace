# Protobuf 协议定义

Echo Trace v1.3.0 采用 **Protocol Buffers** 进行游戏数据序列化，大幅降低带宽占用和序列化开销。

## 协议文件

完整协议定义位于：`proto/echo_trace.proto`

## 消息封装

所有 Protobuf 消息都通过 `Envelope` 封装：

```protobuf
message Envelope {
  int32 msg_type = 1;  // 消息类型代码
  bytes data = 2;      // 序列化后的消息体
}
```

### 消息类型代码

| 类型 | 代码 | 方向 | 描述 |
|------|------|------|------|
| MOVE | 2001 | C2S | 移动/朝向 |
| USE_ITEM | 2002 | C2S | 使用道具 |
| INTERACT | 2003 | C2S | 交互 |
| PICKUP | 2004 | C2S | 拾取 |
| DROP | 2005 | C2S | 丢弃 |
| SELL | 2006 | C2S | 出售 |
| BUY | 2007 | C2S | 购买 |
| SHOP_REFRESH | 2008 | C2S | 刷新商店 |
| CHOOSE_TACTIC | 2009 | C2S | 选择战术 |
| FIRE | 2010 | C2S | 射击 |
| STATE_SNAPSHOT | 3001 | S2C | 状态快照 |

## 基础类型

### Vector2

```protobuf
message Vector2 {
  float x = 1;
  float y = 2;
}
```

### Item

```protobuf
message Item {
  string id = 1;       // 道具 ID（如 "WPN_AP_AMMO"）
  string name = 2;     // 显示名称
  int32 tier = 3;      // 稀有度（1-4）
  int32 uses_left = 4; // 剩余使用次数
}
```

## C2S 消息（客户端 → 服务器）

### MoveInput (2001)

```protobuf
message MoveInput {
  Vector2 dir = 1;         // 移动方向（归一化）
  Vector2 look_dir = 2;    // 朝向（归一化）
  bool has_look_dir = 3;   // 是否包含朝向
}
```

**示例**：
```python
move_msg = echo_trace_pb2.MoveInput(
    dir=echo_trace_pb2.Vector2(x=1.0, y=0.0),  # 向右移动
    look_dir=echo_trace_pb2.Vector2(x=0.707, y=0.707),  # 朝向右上
    has_look_dir=True
)
```

### FireInput (2010)

```protobuf
message FireInput {
  // 无额外字段，服务器根据玩家朝向发射子弹
}
```

### UseItemInput (2002)

```protobuf
message UseItemInput {
  int32 slot_index = 1;  // 道具栏位置（1-6）
}
```

### InteractInput (2003)

```protobuf
message InteractInput {
  // 无字段，自动检测附近可交互实体
}
```

### PickupInput (2004)

```protobuf
message PickupInput {
  // 无字段，拾取最近的掉落物
}
```

### DropInput (2005)

```protobuf
message DropInput {
  int32 slot_index = 1;  // 要丢弃的道具栏位置
}
```

### SellInput (2006)

```protobuf
message SellInput {
  int32 slot_index = 1;  // 要出售的道具栏位置
}
```

### BuyInput (2007)

```protobuf
message BuyInput {
  string item_id = 1;  // 要购买的道具 ID
}
```

### ShopRefreshInput (2008)

```protobuf
message ShopRefreshInput {
  // 无字段，刷新商店货物
}
```

### ChooseTacticInput (2009)

```protobuf
message ChooseTacticInput {
  string tactic = 1;  // "RECON" / "DEFENSE" / "TRAP"
}
```

## S2C 消息（服务器 → 客户端）

### StateSnapshot (3001)

完整的游戏状态快照，每 50ms 推送一次。

```protobuf
message StateSnapshot {
  int32 phase = 1;              // 当前阶段（0-4）
  float time_left = 2;          // 阶段剩余时间（秒）
  
  PlayerState self = 3;         // 自己的状态
  repeated PlayerState others = 4;  // 其他可见玩家
  repeated Entity entities = 5; // 可见实体（引擎、商人等）
  repeated Item ground_items = 6;  // 地面掉落物
  
  repeated GlobalEvent events = 7;  // 全局事件
  repeated Blip radar_blips = 8;    // 雷达信号
  
  bool is_jammer_active = 9;    // T4 全频段干扰是否激活
}
```

#### PlayerState

```protobuf
message PlayerState {
  string session_id = 1;
  string name = 2;
  Vector2 pos = 3;
  Vector2 vel = 4;
  Vector2 look_dir = 5;
  
  float hp = 6;
  float max_hp = 7;
  float armor = 8;
  float max_armor = 9;
  
  int32 kills = 10;
  int32 credits = 11;
  
  repeated Item inventory = 12;  // 道具栏（最多 6 个）
  
  bool is_dead = 13;
  bool is_extracted = 14;
  
  string tactic = 15;  // 战术倾向
}
```

#### Entity

```protobuf
message Entity {
  string uid = 1;
  string type = 2;  // "MOTOR" / "SHOP" / "EXIT" / "AI"
  Vector2 pos = 3;
  int32 state = 4;  // 实体状态（如引擎修复进度）
}
```

#### GlobalEvent

```protobuf
message GlobalEvent {
  string type = 1;  // "PHASE_CHANGE" / "PLAYER_KILL" / "MOTOR_PULSE"
  string msg = 2;   // 事件消息
}
```

#### Blip

```protobuf
message Blip {
  string type = 1;  // "MOTOR" / "PLAYER" / "AI"
  Vector2 pos = 2;  // 雷达上的位置
}
```

## 序列化/反序列化

### Go (服务器)

```go
import pb "echo_trace_server/proto"

// 发送消息
moveInput := &pb.MoveInput{
    Dir: &pb.Vector2{X: 1.0, Y: 0.0},
    LookDir: &pb.Vector2{X: 1.0, Y: 0.0},
    HasLookDir: true,
}
data, _ := proto.Marshal(moveInput)
envelope := &pb.Envelope{
    MsgType: 2001,
    Data: data,
}
binary, _ := proto.Marshal(envelope)
conn.WriteMessage(websocket.BinaryMessage, binary)

// 接收消息
_, message, _ := conn.ReadMessage()
envelope := &pb.Envelope{}
proto.Unmarshal(message, envelope)

switch envelope.MsgType {
case 2001: // MOVE
    moveInput := &pb.MoveInput{}
    proto.Unmarshal(envelope.Data, moveInput)
}
```

### Python (客户端)

```python
from proto import echo_trace_pb2

# 发送消息
move_msg = echo_trace_pb2.MoveInput(
    dir=echo_trace_pb2.Vector2(x=1.0, y=0.0),
    look_dir=echo_trace_pb2.Vector2(x=1.0, y=0.0),
    has_look_dir=True
)
envelope = echo_trace_pb2.Envelope(
    msg_type=2001,
    data=move_msg.SerializeToString()
)
ws.send_binary(envelope.SerializeToString())

# 接收消息
data = ws.recv()
envelope = echo_trace_pb2.Envelope()
envelope.ParseFromString(data)

if envelope.msg_type == 3001:  # STATE_SNAPSHOT
    snapshot = echo_trace_pb2.StateSnapshot()
    snapshot.ParseFromString(envelope.data)
    print(f"Phase: {snapshot.phase}, HP: {snapshot.self.hp}")
```

## 性能对比

### 单个 StateSnapshot 大小（10 玩家场景）

| 协议 | 大小 | 压缩率 |
|------|------|--------|
| JSON | ~8 KB | - |
| **Protobuf** | **~2 KB** | ⬇️ 75% |

### 序列化性能

| 操作 | JSON | Protobuf | 提升 |
|------|------|----------|------|
| 编码 | 1.2 ms | 0.3 ms | 4x |
| 解码 | 1.5 ms | 0.4 ms | 3.75x |

## 编译 Protobuf 代码

### 安装 protoc

**Windows**:
```bash
choco install protoc
```

**Linux**:
```bash
sudo apt install protobuf-compiler
```

### 生成代码

```bash
# Go
protoc --go_out=backend --go_opt=paths=source_relative proto/echo_trace.proto

# Python
protoc --python_out=frontend proto/echo_trace.proto
```

## 下一步

- [消息类型详解](/protobuf/message-types.html)
- [迁移指南（JSON → Protobuf）](/protobuf/migration-guide.html)
