---
home: true
title: 首页
heroImage: null
heroText: Echo Trace API 文档
tagline: 多人在线战术对抗游戏开发者文档 - 高性能 WebSocket + Protobuf + AI 集成
actions:
  - text: 快速开始
    link: /api/
    type: primary
  - text: Protobuf 协议
    link: /protobuf/
    type: secondary
  - text: 游戏逻辑
    link: /game-logic/phases.html
    type: secondary

features:
  - title: 📡 双协议支持
    details: JSON (房间管理) + Protobuf (游戏数据)，兼顾可读性与性能，带宽占用降低 75%
  - title: ⚡ 实时战斗
    details: 50ms Tick 游戏循环，基于 gorilla/websocket 的低延迟通信，支持 20+ 玩家同时在线
  - title: 🤖 智能 AI Boss
    details: A* 寻路 + 行为树，柠白号 (NingBye) 自主巡逻与追击，重型武器 + 雷达脉冲威胁
  - title: 🚀 四叉树优化
    details: AOI 查询性能提升 50-90%，支持大规模实体管理，O(log N) 复杂度空间索引
  - title: 🎁 丰富道具系统
    details: T1-T4 稀有度等级，攻击/生存/侦察/综合四大类，40+ 道具效果平衡设计
  - title: 💬 MCP 集成
    details: Model Context Protocol + DeepSeek AI，自然语言指令解析，许愿装置实现战术干预
  - title: 🎮 5 阶段流程
    details: Phase 0 战术选择 → Phase 1 搜索 → Phase 2 冲突 → Phase 3 逃离 → Phase 4 结算
  - title: 🔒 断线重连
    details: 30 秒宽限期自动重连，SQLite 持久化玩家数据，游戏进度不丢失
  - title: 🛠️ 跨平台支持
    details: Go 1.23+ 后端 + Python 3.10+ 前端，Windows/Linux/macOS 全平台兼容

footer: MIT Licensed | Copyright © 2026 Echo Trace Team
---

## 📚 文档导航

### 🔌 API 参考
- [WebSocket API 概述](/api/) - 连接、消息格式、心跳机制、错误处理
- [房间管理 API](/api/room-management.html) - CREATE_ROOM, JOIN_ROOM, LIST_ROOMS, LEAVE_ROOM
- [游戏操作 API](/api/game-actions.html) - MOVE, FIRE, USE_ITEM, INTERACT, DROP
- [状态快照 API](/api/state-snapshot.html) - StateSnapshot 结构、PlayerState、Entity 类型
- [MCP Server API](/api/mcp-server.html) - HTTP 接口、DeepSeek 集成、许愿指令示例

### 📦 Protobuf 协议
- [协议定义](/protobuf/) - Envelope 结构、Vector2、MoveInput、StateSnapshot
- [消息类型目录](/protobuf/message-types.html) - C2S/S2C 完整消息列表与字段说明
- [迁移指南](/protobuf/migration-guide.html) - JSON → Protobuf 升级步骤、Breaking Changes

### 🎮 游戏逻辑
- [游戏阶段](/game-logic/phases.html) - Phase 0-4 详解、引擎系统、雷达脉冲、缩圈机制
- [道具系统](/game-logic/items.html) - T1-T4 道具完整列表、效果、刷新概率、平衡设计
- [AI Boss 行为](/game-logic/ai-boss.html) - 柠白号巡逻逻辑、追击算法、雷达脉冲、战术应对
- [空间索引](/game-logic/spatial-indexing.html) - 四叉树 AOI 实现、性能基准测试、算法优化

## 🚀 快速开始

### 连接到游戏服务器

```javascript
// JavaScript WebSocket 客户端示例
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('已连接到服务器');
  
  // 发送 JSON 消息（房间管理）
  ws.send(JSON.stringify({
    type: 1003,  // LIST_ROOMS
    payload: {}
  }));
};

ws.onmessage = (event) => {
  if (event.data instanceof Blob) {
    // 接收 Protobuf 二进制消息（游戏数据）
    event.data.arrayBuffer().then(buffer => {
      const envelope = Envelope.decode(new Uint8Array(buffer));
      if (envelope.type === 3001) {  // STATE_SNAPSHOT
        console.log('收到状态快照:', envelope);
      }
    });
  } else {
    // 接收 JSON 消息（房间管理响应）
    const data = JSON.parse(event.data);
    console.log('房间列表:', data.payload.rooms);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('连接已关闭');
};
```

### Python 客户端示例

```python
import asyncio
import websockets
from proto import echo_trace_pb2 as pb

async def game_client():
    uri = "ws://localhost:8080/ws"
    async with websockets.connect(uri) as websocket:
        # 发送移动指令（Protobuf）
        move_input = pb.MoveInput(
            dir_x=1.0, dir_y=0.0,
            look_x=0.5, look_y=0.866
        )
        envelope = pb.Envelope(
            type=2001,  # MOVE_REQ
            payload=move_input.SerializeToString()
        )
        await websocket.send(envelope.SerializeToString())
        
        # 接收状态快照
        while True:
            message = await websocket.recv()
            if isinstance(message, bytes):
                envelope = pb.Envelope()
                envelope.ParseFromString(message)
                print(f"收到消息类型: {envelope.type}")

asyncio.run(game_client())
```

### Go 服务器示例

```go
package main

import (
    "github.com/gorilla/websocket"
    pb "echo_trace/proto"
    "google.golang.org/protobuf/proto"
)

func handleClient(conn *websocket.Conn) {
    for {
        messageType, data, err := conn.ReadMessage()
        if err != nil {
            break
        }
        
        if messageType == websocket.BinaryMessage {
            // 解析 Protobuf 消息
            var envelope pb.Envelope
            proto.Unmarshal(data, &envelope)
            
            switch envelope.Type {
            case 2001:  // MOVE_REQ
                var moveInput pb.MoveInput
                proto.Unmarshal(envelope.Payload, &moveInput)
                // 处理移动逻辑
            case 2010:  // FIRE_REQ
                // 处理开火逻辑
            }
        } else {
            // 处理 JSON 消息（房间管理）
            // ...
        }
    }
}
```

## 📖 版本历史

### v1.3.0 (2026-01-12) - 性能优化与协议升级
- ✅ **Protobuf 二进制协议迁移**：C2S 游戏消息全面支持 Protobuf，带宽占用降低 75%
- ✅ **四叉树空间索引优化**：AOI 查询性能提升 50-90%，支持大规模实体管理
- ✅ **战术永久加成系统**：RECON (+20% 视野) / DEFENSE (+20% HP) / TRAP (+20% 护甲穿透)
- ✅ **房间自动回收机制**：游戏结束 60 秒后自动释放，所有玩家离开立即回收
- ✅ **VitePress 文档系统**：完整 API 参考、Protobuf 协议文档、游戏逻辑详解

### v1.2.0 (2025-12) - AI 与道具系统升级
- AI Boss 柠白号战术升级：A* 寻路 + 行为树优化
- MCP 许愿装置集成：DeepSeek AI + FastAPI 自然语言接口
- T4 传说道具系统：轨道打击、柠白号同步、全频段干扰、开发者命令行
- 雷达脉冲机制：Phase 2/3 周期性位置暴露
- 地图缩圈系统：Phase 3 边缘伤害区收缩

### v1.1.0 (2025-10) - 核心玩法实现
- 多阶段游戏流程：Phase 0 战术选择 → Phase 1 搜索 → Phase 2 冲突 → Phase 3 逃离
- FOG 战争迷雾系统：视锥视野 + LOS 视线检测
- 道具生成平衡调整：T1-T3 道具稀有度权重优化
- 引擎修复机制：Phase 1 → Phase 2 转换条件
- 撤离点系统：Phase 3 多撤离点先到先得

### v1.0.0 (2025-08) - 初始版本
- WebSocket 通信基础框架
- JSON 消息协议
- 基础移动与战斗系统
- 简单 AI 巡逻逻辑

## 🎯 核心特性详解

### 双协议设计

| 协议 | 使用场景 | 优势 | 消息示例 |
|------|---------|------|----------|
| **JSON** | 房间管理（CREATE/JOIN/LIST） | 可读性强，易于调试 | `{"type": 1002, "payload": {...}}` |
| **Protobuf** | 游戏数据（MOVE/FIRE/STATE） | 体积小，序列化快 | 二进制流 (~20 bytes) |

**带宽对比**：
- JSON 移动消息：~100 bytes
- Protobuf 移动消息：~20 bytes
- **节省 80% 带宽**

### 四叉树空间索引

**传统方案问题**：
- Grid-based AOI：O(N) 遍历所有实体
- 100 实体场景：~10ms 查询时间

**四叉树方案**：
- 递归四分空间：O(log N) 查询
- 100 实体场景：~0.5ms 查询时间
- **性能提升 95%**

### MCP 自然语言接口

**许愿装置道具**：
- **残缺版 (T2)**：受限权限（小范围传送、查询信息）
- **完整版 (T4)**：中等权限（清除障碍、交换位置）
- **开发者版 (T4)**：系统级权限（修改参数、操控 Boss）

**示例指令**：
```
玩家：传送到最近的引擎
AI 解析：{"action": "teleport", "params": {"target_x": 50.0, "target_y": 30.0}}
游戏执行：玩家瞬移至坐标 (50.0, 30.0)
```

## 🛠️ 开发工具

### Protobuf 编译工具

```bash
# 安装 protoc (https://github.com/protocolbuffers/protobuf/releases)
# 下载 protoc-6.33.3-win64.zip (Windows)
# 下载 protoc-6.33.3-linux-x86_64.zip (Linux)

# 编译 Go 代码
protoc --go_out=backend --go_opt=paths=source_relative proto/echo_trace.proto

# 编译 Python 代码
protoc --python_out=frontend proto/echo_trace.proto
```

### 性能分析工具

```bash
# Go pprof (CPU 性能分析)
go run . -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Python cProfile
python -m cProfile -o output.prof main.py
python -m pstats output.prof
```

### 测试工具

```bash
# Go 单元测试
cd backend
go test -v -cover ./...

# Python 测试 (pytest)
cd frontend
pytest --cov=client --cov=ui
```

## 🔗 相关链接

- 📖 [玩家指南](../PLAYER_GUIDE.md) - 完整游戏玩法、道具系统、战术策略
- 🛠️ [开发者指南](../DEVELOPER_GUIDE.md) - 项目架构、技术栈、贡献指南
- 🐛 [问题反馈](https://github.com/your-repo/echo-trace/issues) - Bug 报告与功能建议
- 📝 [更新日志](https://github.com/your-repo/echo-trace/releases) - 版本发布记录
- 💬 [讨论区](https://github.com/your-repo/echo-trace/discussions) - 社区交流

## 📊 性能基准

| 场景 | JSON 协议 | Protobuf 协议 | 提升 |
|------|----------|--------------|------|
| 10 玩家移动消息 | 200 KB/s | 50 KB/s | **75%** |
| 状态快照 (20 实体) | 2000 bytes | 500 bytes | **75%** |
| 序列化时间 | 0.5 ms | 0.1 ms | **80%** |
| AOI 查询 (100 实体) | 10 ms | 0.5 ms | **95%** |

## 🎓 学习路径

### 新手开发者
1. 阅读 [玩家指南](../PLAYER_GUIDE.md) 了解游戏机制
2. 查看 [API 概述](/api/) 学习通信协议
3. 运行官方客户端示例代码
4. 尝试修改道具参数或创建新道具

### 进阶开发者
1. 深入 [Protobuf 协议](/protobuf/) 理解序列化机制
2. 研究 [四叉树空间索引](/game-logic/spatial-indexing.html) 算法
3. 优化 AI Boss 行为树
4. 实现客户端预测与服务器和解

### 高级开发者
1. 贡献核心功能（Delta Snapshot、观战模式）
2. 优化网络协议（增量更新、压缩算法）
3. 设计新游戏模式（团队战、排位赛）
4. 编写性能分析工具

---

<div align="center">

**📘 开始探索 Echo Trace 的技术世界！**

如有疑问，请访问 [讨论区](https://github.com/your-repo/echo-trace/discussions)

</div>

