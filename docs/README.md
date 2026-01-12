---
home: true
title: 首页
heroImage: null
heroText: Echo Trace API 文档
tagline: 多人在线战术对抗游戏开发者文档
actions:
  - text: 快速开始
    link: /api/
    type: primary
  - text: Protobuf 协议
    link: /protobuf/
    type: secondary

features:
  - title: WebSocket API
    details: 基于 gorilla/websocket 的实时通信协议，支持 JSON 和 Protobuf 双模式
  - title: Protobuf 优化
    details: v1.3.0 采用 Protocol Buffers 实现游戏数据序列化，带宽占用降低 75%
  - title: 空间索引
    details: 四叉树 AOI 算法，查询性能提升 50-90%，支持大规模实体管理
  - title: MCP 集成
    details: Model Context Protocol 支持自然语言交互，玩家可通过许愿装置发送 AI 指令
  - title: 游戏逻辑
    details: 5 阶段游戏流程，T1-T4 道具系统，AI Boss 战术系统
  - title: 持久化
    details: SQLite 数据库存储玩家数据，支持断线重连和历史记录查询

footer: MIT Licensed | Copyright © 2026 Echo Trace Team
---

## 📚 文档导航

### 🔌 API 参考
- [WebSocket API 概述](/api/) - 连接、消息格式、错误处理
- [房间管理](/api/room-management.html) - 创建、加入、列表查询
- [游戏操作](/api/game-actions.html) - 移动、射击、道具使用
- [状态快照](/api/state-snapshot.html) - 服务器推送的游戏状态

### 📦 Protobuf 协议
- [协议定义](/protobuf/) - Message 结构、字段说明
- [消息类型](/protobuf/message-types.html) - C2S/S2C 消息完整列表
- [迁移指南](/protobuf/migration-guide.html) - 从 JSON 迁移到 Protobuf

### 🎮 游戏逻辑
- [游戏阶段](/game-logic/phases.html) - Phase 0-4 流程详解
- [道具系统](/game-logic/items.html) - T1-T4 道具效果与平衡
- [AI Boss](/game-logic/ai-boss.html) - 柠白号行为树与战术
- [空间索引](/game-logic/spatial-indexing.html) - 四叉树 AOI 实现

## 🚀 快速开始

### 连接到游戏服务器

```javascript
// JavaScript WebSocket 客户端示例
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  // 发送 JSON 消息（房间管理）
  ws.send(JSON.stringify({
    type: 'LIST_ROOMS'
  }));
};

ws.onmessage = (event) => {
  if (event.data instanceof Blob) {
    // 接收 Protobuf 二进制消息（游戏数据）
    event.data.arrayBuffer().then(buffer => {
      const envelope = Envelope.decode(new Uint8Array(buffer));
      console.log('收到游戏消息:', envelope);
    });
  } else {
    // 接收 JSON 消息（房间管理响应）
    const data = JSON.parse(event.data);
    console.log('房间列表:', data);
  }
};
```

### Python 客户端示例

```python
import websocket
from proto import echo_trace_pb2

ws = websocket.WebSocket()
ws.connect("ws://localhost:8080/ws")

# 发送移动指令（Protobuf）
move_msg = echo_trace_pb2.MoveInput(
    dir=echo_trace_pb2.Vector2(x=1.0, y=0.0),
    look_dir=echo_trace_pb2.Vector2(x=1.0, y=0.0)
)
envelope = echo_trace_pb2.Envelope(
    msg_type=2001,  # MOVE
    data=move_msg.SerializeToString()
)
ws.send_binary(envelope.SerializeToString())
```

## 📖 版本历史

### v1.3.0 (2026-01-12)
- ✅ Protobuf 二进制协议迁移
- ✅ 四叉树空间索引优化
- ✅ 战术永久加成系统
- ✅ 房间自动回收机制

### v1.2.0
- AI Boss 柠白号战术升级
- MCP 许愿装置集成
- T4 传说道具系统

### v1.1.0
- 多阶段游戏流程
- FOG 战争迷雾系统
- 道具生成平衡调整

## 🔗 相关链接

- [项目 GitHub](https://github.com/your-repo/echo-trace)
- [问题反馈](https://github.com/your-repo/echo-trace/issues)
- [更新日志](https://github.com/your-repo/echo-trace/releases)
