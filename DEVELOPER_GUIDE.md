# 🛠️ Echo Trace 开发者指南

<div align="center">

**为贡献者和维护者提供的完整技术文档**

</div>

---

## 📖 目录

- [项目架构](#-项目架构)
- [技术栈详解](#-技术栈详解)
- [开发环境配置](#-开发环境配置)
- [构建和运行流程](#-构建和运行流程)
- [Protobuf 使用指南](#-protobuf-使用指南)
- [MCP 服务器开发](#-mcp-服务器开发)
- [代码结构说明](#-代码结构说明)
- [网络协议详解](#-网络协议详解)
- [性能优化技术](#-性能优化技术)
- [测试与调试](#-测试与调试)
- [贡献指南](#-贡献指南)
- [API 文档生成](#-api-文档生成)

---

## 🏗️ 项目架构

### 总体架构

Echo Trace 采用 **客户端-服务器 (Client-Server)** 架构，分为三个主要模块：

```
┌─────────────────┐         WebSocket          ┌──────────────────┐
│                 │    (JSON + Protobuf)       │                  │
│  Python Client  │ ◄─────────────────────────► │   Go Server      │
│   (Pygame UI)   │                             │  (Game Logic)    │
│                 │                             │                  │
└─────────────────┘                             └────────┬─────────┘
                                                         │
                                                         │ HTTP API
                                                         │
                                                  ┌──────▼──────────┐
                                                  │                 │
                                                  │  MCP Server     │
                                                  │  (FastAPI +     │
                                                  │   DeepSeek AI)  │
                                                  │                 │
                                                  └─────────────────┘
```

### 模块职责

#### 1. **Backend (Go 游戏服务器)**
- **路径**：`backend/`
- **语言**：Go 1.23+
- **职责**：
  - 游戏逻辑处理（移动、战斗、AI、物品）
  - 状态同步（50ms Tick）
  - WebSocket 通信管理
  - 房间生命周期管理
  - SQLite 数据持久化
  - 四叉树空间索引

#### 2. **Frontend (Python 游戏客户端)**
- **路径**：`frontend/`
- **语言**：Python 3.10+
- **职责**：
  - 渲染游戏画面（Pygame）
  - 用户输入处理
  - WebSocket 客户端
  - 本地化和资源管理
  - UI 界面（主菜单、设置、HUD）

#### 3. **Game_MCP_Server (AI 许愿装置后端)**
- **路径**：`Game_MCP_Server/`
- **语言**：Python 3.10+ (FastAPI)
- **职责**：
  - 自然语言指令解析
  - DeepSeek LLM 集成
  - 游戏逻辑调用接口
  - 权限控制和安全验证

#### 4. **Proto (协议定义)**
- **路径**：`proto/`
- **语言**：Protobuf
- **职责**：
  - 定义 C2S/S2C 消息格式
  - 跨语言序列化标准
  - 版本兼容性管理

#### 5. **Docs (开发者文档)**
- **路径**：`docs/`
- **工具**：VitePress 1.6.4
- **职责**：
  - API 参考文档
  - Protobuf 协议文档
  - 游戏逻辑文档
  - 版本更新日志

---

## 💻 技术栈详解

### 后端技术栈

| 组件 | 版本 | 用途 | 官方文档 |
|------|------|------|----------|
| **Go** | 1.23+ | 游戏服务器核心语言 | [golang.org](https://golang.org) |
| **Gorilla WebSocket** | v1.5.0 | WebSocket 通信库 | [github.com/gorilla/websocket](https://github.com/gorilla/websocket) |
| **Protocol Buffers** | v1.36.11 | 二进制序列化协议 | [protobuf.dev](https://protobuf.dev) |
| **SQLite** | 3.x | 玩家数据持久化 | [sqlite.org](https://sqlite.org) |
| **A* Pathfinding** | 自实现 | AI 寻路算法 | - |
| **Quadtree** | 自实现 | 空间索引（AOI） | - |

### 前端技术栈

| 组件 | 版本 | 用途 | 官方文档 |
|------|------|------|----------|
| **Python** | 3.10+ | 客户端核心语言 | [python.org](https://python.org) |
| **Pygame** | 2.5.0+ | 游戏渲染引擎 | [pygame.org](https://pygame.org) |
| **websockets** | 12.0+ | WebSocket 客户端库 | [websockets.readthedocs.io](https://websockets.readthedocs.io) |
| **protobuf** | 6.33.3 | Protobuf Python 支持 | [protobuf.dev](https://protobuf.dev) |
| **PyInstaller** | 5.x+ | 打包可执行文件（可选） | [pyinstaller.org](https://pyinstaller.org) |

### MCP 服务器技术栈

| 组件 | 版本 | 用途 | 官方文档 |
|------|------|------|----------|
| **FastAPI** | 0.100+ | HTTP API 框架 | [fastapi.tiangolo.com](https://fastapi.tiangolo.com) |
| **DeepSeek** | API v1 | 自然语言 LLM | [platform.deepseek.com](https://platform.deepseek.com) |
| **Uvicorn** | 0.20+ | ASGI 服务器 | [uvicorn.org](https://uvicorn.org) |
| **python-dotenv** | 1.0+ | 环境变量管理 | - |

### 文档技术栈

| 组件 | 版本 | 用途 | 官方文档 |
|------|------|------|----------|
| **VitePress** | 1.6.4 | 文档生成器 | [vitepress.dev](https://vitepress.dev) |
| **Vue** | 3.5.13 | 文档 UI 框架 | [vuejs.org](https://vuejs.org) |
| **Node.js** | 18.x+ | JavaScript 运行时 | [nodejs.org](https://nodejs.org) |

---

## 🔧 开发环境配置

### 1. Go 环境

#### 安装 Go 1.23+

**Windows**：
```powershell
# 下载安装器
# https://go.dev/dl/

# 验证安装
go version
# 输出: go version go1.23.0 windows/amd64
```

**Linux/macOS**：
```bash
# 使用包管理器
sudo apt install golang-1.23  # Ubuntu/Debian
brew install go@1.23          # macOS

# 或下载二进制
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz

# 配置环境变量
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

#### 配置 Go 模块代理（可选，加速下载）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GO111MODULE=on
```

---

### 2. Python 环境

#### 安装 Python 3.10+

**Windows**：
```powershell
# 下载安装器
# https://www.python.org/downloads/

# 验证安装
python --version
# 输出: Python 3.10.0
```

**Linux/macOS**：
```bash
# Ubuntu/Debian
sudo apt install python3.10 python3.10-venv python3-pip

# macOS
brew install python@3.10

# 验证安装
python3 --version
```

#### 创建虚拟环境（推荐）

```bash
# 创建虚拟环境
python3 -m venv venv

# 激活虚拟环境
# Windows
.\venv\Scripts\activate

# Linux/macOS
source venv/bin/activate

# 升级 pip
pip install --upgrade pip
```

---

### 3. Protoc 编译器（可选，用于修改协议）

#### 安装 Protocol Buffers 编译器

**Windows**：
```powershell
# 下载预编译二进制
# https://github.com/protocolbuffers/protobuf/releases

# 下载 protoc-6.33.3-win64.zip
# 解压到 C:\protoc
# 添加 C:\protoc\bin 到 PATH 环境变量

# 验证安装
protoc --version
# 输出: libprotoc 6.33.3
```

**Linux/macOS**：
```bash
# Ubuntu/Debian
sudo apt install protobuf-compiler

# macOS
brew install protobuf

# 或下载二进制
wget https://github.com/protocolbuffers/protobuf/releases/download/v6.33.3/protoc-6.33.3-linux-x86_64.zip
unzip protoc-6.33.3-linux-x86_64.zip -d $HOME/.local
export PATH="$PATH:$HOME/.local/bin"

# 验证安装
protoc --version
```

---

### 4. 数据库（SQLite，无需额外安装）

SQLite 已集成在 Go 和 Python 标准库中，无需额外配置。

---

### 5. IDE 推荐

#### GoLand / VS Code (Go 开发)
- **GoLand**：JetBrains 专业 IDE，强大的代码补全和调试
- **VS Code**：轻量级，安装 Go 扩展（`golang.go`）

#### PyCharm / VS Code (Python 开发)
- **PyCharm**：JetBrains 专业 IDE，适合大型 Python 项目
- **VS Code**：轻量级，安装 Python 扩展（`ms-python.python`）

---

## 🚀 构建和运行流程

### 完整启动流程

#### 步骤 1：克隆仓库

```bash
git clone <repository_url>
cd Echo_Trace
```

#### 步骤 2：安装 Go 依赖

```bash
cd backend
go mod download

# 验证依赖
go mod verify
```

#### 步骤 3：安装 Python 依赖

```bash
cd ../frontend
pip install -r requirements.txt

# 依赖列表:
# pygame>=2.5.0
# websockets>=12.0
# protobuf>=6.33.3
```

#### 步骤 4：配置 MCP Server（可选）

```bash
cd ../Game_MCP_Server
pip install -r requirements.txt

# 创建环境变量文件
cp .env.example .env

# 编辑 .env 文件
# AI_API_KEY=your_deepseek_api_key_here
# AI_BASE_URL=https://api.deepseek.com
```

---

### 启动服务

#### Terminal 1: MCP Server (可选)

```bash
cd Game_MCP_Server
python server.py

# 输出:
# INFO:     Uvicorn running on http://localhost:9091
```

#### Terminal 2: 游戏服务器

```bash
cd backend
go run .

# 输出:
# 2024/01/15 14:30:00 Server started on :8080
# 2024/01/15 14:30:00 Quadtree AOI initialized
# 2024/01/15 14:30:00 Database initialized
```

#### Terminal 3: 游戏客户端

```bash
cd frontend
python main.py

# Pygame 窗口启动
```

---

### 构建可执行文件

#### 构建服务器（Go）

```bash
cd backend

# Windows
go build -o echo_trace_server.exe .

# Linux/macOS
go build -o echo_trace_server .

# 运行
./echo_trace_server
```

#### 构建客户端（PyInstaller）

```bash
cd frontend

# 安装 PyInstaller
pip install pyinstaller

# 打包单文件可执行文件
pyinstaller --onefile --windowed main.py

# 输出路径: dist/main.exe (Windows) 或 dist/main (Linux/macOS)
```

---

## 📡 Protobuf 使用指南

### Protobuf 架构

Echo Trace 使用 **Protocol Buffers** 作为高频游戏数据的序列化协议，相比 JSON 具有以下优势：
- **体积更小**：二进制编码，减少 75% 带宽占用
- **性能更高**：序列化/反序列化速度快 5-10 倍
- **类型安全**：强类型定义，避免运行时错误

### 协议文件结构

```
proto/
└── echo_trace.proto   # 完整协议定义
```

### 消息类型定义

#### 消息信封（Envelope）

所有 Protobuf 消息都包裹在 `Envelope` 中：

```protobuf
message Envelope {
  int32 type = 1;        // 消息类型 ID (2001-2099)
  bytes payload = 2;     // 序列化后的消息体
}
```

#### C2S 消息类型

| 类型 ID | 名称 | 消息体 | 用途 |
|--------|------|--------|------|
| 2001 | MOVE_REQ | MoveInput | 移动指令 |
| 2002 | USE_ITEM_REQ | UseItemInput | 使用道具 |
| 2005 | DROP_REQ | DropInput | 丢弃道具 |
| 2006 | CHOOSE_TACTIC_REQ | TacticInput | 选择战术 |
| 2007 | BUY_REQ | BuyInput | 购买道具 |
| 2008 | SELL_REQ | SellInput | 出售道具 |
| 2010 | FIRE_REQ | - | 开火 |

#### S2C 消息类型（计划中）

| 类型 ID | 名称 | 消息体 | 用途 |
|--------|------|--------|------|
| 3001 | STATE_SNAPSHOT | StateSnapshot | 游戏状态快照 |

### 代码生成

#### 生成 Go 代码

```bash
# 安装 protoc-gen-go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 生成代码
protoc --go_out=backend --go_opt=paths=source_relative proto/echo_trace.proto

# 输出: backend/proto/echo_trace.pb.go
```

#### 生成 Python 代码

```bash
# protoc 自带 Python 插件

# 生成代码
protoc --python_out=frontend proto/echo_trace.proto

# 输出: frontend/proto/echo_trace_pb2.py
```

### 使用示例

#### Go 后端（接收消息）

```go
package network

import (
    pb "echo_trace/proto"
    "google.golang.org/protobuf/proto"
)

func (c *Client) handleProtobufMessage(data []byte) error {
    // 解析 Envelope
    var envelope pb.Envelope
    if err := proto.Unmarshal(data, &envelope); err != nil {
        return err
    }

    // 根据类型分发
    switch envelope.Type {
    case 2001: // MOVE_REQ
        var moveInput pb.MoveInput
        if err := proto.Unmarshal(envelope.Payload, &moveInput); err != nil {
            return err
        }
        // 处理移动
        c.room.gameState.HandleMove(c.playerID, moveInput)
        
    case 2010: // FIRE_REQ
        c.room.gameState.HandleFire(c.playerID)
    }
    
    return nil
}
```

#### Python 前端（发送消息）

```python
from proto import echo_trace_pb2 as pb

class NetworkClient:
    async def send_move(self, dir_x, dir_y, look_x, look_y):
        """发送移动指令（Protobuf）"""
        # 构造消息体
        move_input = pb.MoveInput(
            dir_x=dir_x,
            dir_y=dir_y,
            look_x=look_x,
            look_y=look_y
        )
        
        # 包裹在 Envelope 中
        envelope = pb.Envelope(
            type=2001,  # MOVE_REQ
            payload=move_input.SerializeToString()
        )
        
        # 发送二进制消息
        await self.ws.send(envelope.SerializeToString())

    async def send_fire(self):
        """发送开火指令（Protobuf）"""
        envelope = pb.Envelope(
            type=2010,  # FIRE_REQ
            payload=b""  # 无消息体
        )
        await self.ws.send(envelope.SerializeToString())
```

### 协议兼容性

#### 双协议支持

Echo Trace 同时支持 **JSON** 和 **Protobuf**：

| 功能模块 | 协议 | 原因 |
|---------|------|------|
| 房间管理 | JSON | 低频操作，可读性更重要 |
| 游戏数据 | Protobuf | 高频传输，性能更重要 |

#### JSON 房间管理消息

```json
// CREATE_ROOM (C2S)
{
  "type": 1001,
  "payload": {
    "room_name": "My Room",
    "max_players": 10,
    "map_width": 100,
    "map_height": 100
  }
}

// JOIN_ROOM (C2S)
{
  "type": 1002,
  "payload": {
    "room_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### 性能对比

| 指标 | JSON | Protobuf | 提升 |
|------|------|----------|------|
| 单次移动消息 | ~100 bytes | ~20 bytes | **80%** |
| 单次状态快照 | ~2KB | ~500B | **75%** |
| 序列化时间 | ~0.5ms | ~0.1ms | **80%** |
| 10玩家带宽 | 400KB/s | 100KB/s | **75%** |

---

## 🤖 MCP 服务器开发

### MCP 架构

**MCP (Model Context Protocol)** 是 Echo Trace 的 AI 许愿装置后端，使用 FastAPI 和 DeepSeek LLM 实现自然语言指令解析和执行。

### 文件结构

```
Game_MCP_Server/
├── server.py           # FastAPI 主程序
├── .env.example        # 环境变量模板
├── .env                # 实际环境变量（需手动创建）
└── requirements.txt    # Python 依赖
```

### 环境变量配置

```bash
# .env 文件内容
AI_API_KEY=your_deepseek_api_key_here
AI_BASE_URL=https://api.deepseek.com
AI_MODEL=deepseek-chat
```

### FastAPI 接口

#### POST /wish

接收玩家许愿指令，调用 DeepSeek LLM 解析并执行。

**请求格式**：
```json
{
  "player_id": "uuid-string",
  "wish_text": "传送到最近的引擎",
  "permission_level": 1  // 1: 残缺版, 2: 完整版, 3: 开发者版
}
```

**响应格式**：
```json
{
  "success": true,
  "action": "teleport",
  "params": {
    "target_x": 50.0,
    "target_y": 30.0
  },
  "message": "已传送到坐标 (50.0, 30.0)"
}
```

### DeepSeek 集成

#### Prompt 模板

```python
SYSTEM_PROMPT = """
你是 Echo Trace 游戏的 AI 助手。玩家会发送许愿指令，你需要将其转换为游戏操作。

权限等级：
- Level 1 (残缺版): 查询信息、小范围传送 (<10m)、临时属性提升 (<20%)
- Level 2 (完整版): 中等范围传送 (<50m)、清除障碍、交换位置
- Level 3 (开发者版): 无限制传送、修改参数、召唤实体、操控 Boss

返回 JSON 格式：
{
  "action": "teleport|heal|speed_boost|spawn_item|...",
  "params": {...},
  "message": "操作说明"
}
"""

def parse_wish(wish_text: str, permission_level: int) -> dict:
    response = deepseek.chat.completions.create(
        model="deepseek-chat",
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": f"Level {permission_level}: {wish_text}"}
        ]
    )
    return json.loads(response.choices[0].message.content)
```

### 游戏逻辑调用

MCP Server 需要调用游戏服务器的接口来执行操作：

```python
import requests

GAME_SERVER_URL = "http://localhost:8080"

def execute_action(player_id: str, action: dict):
    if action["action"] == "teleport":
        # 调用游戏服务器传送接口
        requests.post(f"{GAME_SERVER_URL}/api/teleport", json={
            "player_id": player_id,
            "x": action["params"]["target_x"],
            "y": action["params"]["target_y"]
        })
    
    elif action["action"] == "heal":
        requests.post(f"{GAME_SERVER_URL}/api/heal", json={
            "player_id": player_id,
            "amount": action["params"]["hp"]
        })
```

### 安全限制

#### 权限等级验证

```python
def validate_permission(wish_action: str, level: int) -> bool:
    LEVEL_1_ACTIONS = ["query", "teleport_short", "buff_minor"]
    LEVEL_2_ACTIONS = LEVEL_1_ACTIONS + ["teleport_medium", "clear_obstacle"]
    LEVEL_3_ACTIONS = LEVEL_2_ACTIONS + ["teleport_any", "modify_param", "spawn"]
    
    if level == 1:
        return wish_action in LEVEL_1_ACTIONS
    elif level == 2:
        return wish_action in LEVEL_2_ACTIONS
    elif level == 3:
        return wish_action in LEVEL_3_ACTIONS
    
    return False
```

#### 防止滥用

```python
# 限制玩家每分钟最多 5 次许愿
from collections import defaultdict
from time import time

wish_cooldown = defaultdict(list)

def check_rate_limit(player_id: str) -> bool:
    now = time()
    # 清理 1 分钟前的记录
    wish_cooldown[player_id] = [t for t in wish_cooldown[player_id] if now - t < 60]
    
    if len(wish_cooldown[player_id]) >= 5:
        return False  # 超过限制
    
    wish_cooldown[player_id].append(now)
    return True
```

---

## 📂 代码结构说明

### Backend 结构

```
backend/
├── main.go                 # 入口文件
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖锁文件
├── internal/
│   ├── game/
│   │   ├── state.go        # GameState 核心逻辑
│   │   ├── player.go       # 玩家实体
│   │   ├── entity.go       # 物品/子弹实体
│   │   ├── actions.go      # 玩家操作处理
│   │   ├── ai_ningbye.go   # AI Boss 行为
│   │   └── items.go        # 道具系统
│   ├── network/
│   │   ├── client.go       # WebSocket 客户端管理
│   │   ├── room.go         # 房间管理
│   │   └── server.go       # WebSocket 服务器
│   ├── quadtree/
│   │   └── quadtree.go     # 四叉树空间索引
│   ├── aoi/
│   │   └── aoi.go          # 视野管理（AOI）
│   ├── pathfinding/
│   │   └── astar.go        # A* 寻路算法
│   └── storage/
│       └── database.go     # SQLite 数据库
├── proto/
│   └── echo_trace.pb.go    # Protobuf 生成代码
└── config/
    └── config.go           # 配置文件解析
```

### Frontend 结构

```
frontend/
├── main.py                 # 入口文件
├── requirements.txt        # Python 依赖
├── client/
│   ├── network.py          # WebSocket 客户端
│   ├── renderer.py         # Pygame 渲染器
│   ├── input_handler.py    # 用户输入处理
│   └── game_state.py       # 客户端游戏状态
├── ui/
│   ├── main_menu.py        # 主菜单界面
│   ├── hud.py              # 游戏内 HUD
│   └── settings.py         # 设置菜单
├── assets/
│   ├── textures/           # 贴图资源
│   ├── sounds/             # 音效资源
│   └── locales/            # 本地化文件
└── proto/
    └── echo_trace_pb2.py   # Protobuf Python 代码
```

### 关键文件详解

#### `backend/internal/game/state.go`

游戏核心逻辑，管理所有游戏状态：

```go
type GameState struct {
    Players        map[string]*Player
    Entities       []Entity
    Quadtree       *Quadtree
    Phase          int
    PhaseStartTime time.Time
    Config         *GameConfig
}

// 每 50ms 调用一次
func (gs *GameState) Tick() {
    // 1. 更新玩家移动
    // 2. 更新子弹飞行
    // 3. 碰撞检测
    // 4. AI 行为更新
    // 5. 重建四叉树
    // 6. 阶段转换检查
}
```

#### `frontend/client/network.py`

WebSocket 客户端，处理消息收发：

```python
class NetworkClient:
    async def connect(self, url: str):
        self.ws = await websockets.connect(url)
        asyncio.create_task(self._receive_loop())
    
    async def _receive_loop(self):
        async for message in self.ws:
            if isinstance(message, bytes):
                # Protobuf 消息
                self._handle_protobuf(message)
            else:
                # JSON 消息
                self._handle_json(message)
```

---

## 🌐 网络协议详解

### WebSocket 端点

- **游戏服务器**：`ws://localhost:8080/ws`
- **MCP 服务器**：`http://localhost:9091`

### 连接流程

```
Client                          Server
  │                                │
  ├─────── WebSocket Handshake ────►
  │◄──────── 101 Switching ─────────┤
  │                                │
  ├─────── {"type": 1002} ──────────► JOIN_ROOM
  │◄─────── {"type": 9002} ─────────┤ JOIN_SUCCESS
  │                                │
  ├─────── Protobuf (2001) ─────────► MOVE_REQ
  │◄─────── JSON StateSnapshot ─────┤ (50ms Tick)
  │                                │
```

### 心跳机制

```python
# 客户端每 30 秒发送 PING
async def heartbeat():
    while True:
        await asyncio.sleep(30)
        await ws.ping()

# 服务器响应 PONG
# 超过 60 秒无心跳则断开连接
```

### 错误码

| 错误码 | 名称 | 说明 |
|--------|------|------|
| 4000 | ROOM_NOT_FOUND | 房间不存在 |
| 4001 | ROOM_FULL | 房间已满 |
| 4002 | INVALID_MESSAGE | 消息格式错误 |
| 4003 | UNAUTHORIZED | 未授权操作 |
| 4004 | PLAYER_DEAD | 玩家已死亡 |

---

## ⚡ 性能优化技术

### 1. 四叉树空间索引

**问题**：原有 Grid-based AOI 需要 O(N) 遍历所有实体。

**解决方案**：使用四叉树将复杂度降低到 O(log N)。

**实现**：
```go
// backend/internal/quadtree/quadtree.go
type Quadtree struct {
    Boundary Rect
    Capacity int                // 每个节点最多 8 个实体
    Entities []QuadtreeEntity
    Divided  bool
    NE, SE, SW, NW *Quadtree
}

// 矩形范围查询
func (qt *Quadtree) Query(queryRange Rect) []QuadtreeEntity {
    found := []QuadtreeEntity{}
    if !qt.Boundary.Intersects(queryRange) {
        return found  // 剪枝
    }
    // 递归查询子节点
    // ...
}
```

**性能提升**：
- 10 玩家 + 100 物品：**50% 提升**（20ms → 10ms）
- 20 玩家 + 200 物品：**90% 提升**（100ms → 10ms）

---

### 2. Protobuf 二进制协议

**问题**：JSON 序列化开销大，带宽占用高。

**解决方案**：使用 Protobuf 进行二进制序列化。

**带宽对比**：
```
// JSON 移动消息 (~100 bytes)
{"type": "move", "dir_x": 1.0, "dir_y": 0.0, "look_x": 0.5, "look_y": 0.866}

// Protobuf 移动消息 (~20 bytes)
\x08\xd1\x0f\x12\x10\x00\x00\x80?\x00\x00\x00\x00\x00\x00\x00?\x85\xebQ\xb8\x1e\t@
```

**延迟降低**：
- 序列化时间：0.5ms → 0.1ms
- 网络传输时间：10ms → 2.5ms（在 100KB/s 带宽下）

---

### 3. 对象池（Object Pooling）

**问题**：频繁创建/销毁子弹对象导致 GC 压力。

**解决方案**：预分配对象池，复用对象。

```go
type BulletPool struct {
    pool []*Bullet
    mu   sync.Mutex
}

func (bp *BulletPool) Get() *Bullet {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    
    if len(bp.pool) > 0 {
        bullet := bp.pool[len(bp.pool)-1]
        bp.pool = bp.pool[:len(bp.pool)-1]
        return bullet
    }
    return &Bullet{}  // 池为空时创建新对象
}

func (bp *BulletPool) Put(bullet *Bullet) {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    
    bullet.Reset()  // 重置状态
    bp.pool = append(bp.pool, bullet)
}
```

---

### 4. 增量状态更新（计划中）

**当前问题**：每 Tick 发送完整 StateSnapshot，带宽浪费。

**计划方案**：仅发送变化的实体（Delta Snapshot）。

```protobuf
message DeltaSnapshot {
  repeated PlayerUpdate updated_players = 1;
  repeated EntityUpdate updated_entities = 2;
  repeated string removed_entities = 3;
}

message PlayerUpdate {
  string id = 1;
  optional float x = 2;  // 仅更新变化的字段
  optional float y = 3;
  optional float hp = 4;
}
```

**预期收益**：带宽占用再降低 50-70%。

---

## 🧪 测试与调试

### 单元测试

#### Go 后端测试

```bash
cd backend

# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/quadtree

# 带覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**测试示例**：
```go
// backend/internal/quadtree/quadtree_test.go
func TestQuadtreeInsert(t *testing.T) {
    qt := NewQuadtree(Rect{X: 50, Y: 50, Width: 50, Height: 50}, 4)
    
    entity := QuadtreeEntity{UID: "test", X: 30, Y: 30}
    inserted := qt.Insert(entity)
    
    if !inserted {
        t.Errorf("Failed to insert entity")
    }
}
```

---

#### Python 前端测试

```bash
cd frontend

# 安装 pytest
pip install pytest pytest-asyncio

# 运行所有测试
pytest

# 带覆盖率
pytest --cov=client --cov=ui
```

**测试示例**：
```python
# frontend/tests/test_network.py
import pytest
from client.network import NetworkClient

@pytest.mark.asyncio
async def test_connect():
    client = NetworkClient()
    await client.connect("ws://localhost:8080/ws")
    assert client.ws is not None
```

---

### 性能分析

#### Go pprof

```bash
cd backend

# CPU 性能分析
go run . -cpuprofile=cpu.prof
# 运行一段时间后停止
go tool pprof cpu.prof

# 内存分析
go run . -memprofile=mem.prof
go tool pprof mem.prof
```

**pprof 交互命令**：
```
(pprof) top10       # 显示前 10 个热点函数
(pprof) list Tick   # 显示 Tick 函数的代码
(pprof) web         # 生成可视化图表（需要 Graphviz）
```

---

#### Python cProfile

```bash
cd frontend

# 性能分析
python -m cProfile -o output.prof main.py

# 查看结果
python -m pstats output.prof
```

---

### 日志系统

#### 后端日志

```go
// backend/main.go
import "log"

log.Println("[INFO] Server started on :8080")
log.Printf("[DEBUG] Player %s joined room %s", playerID, roomID)
log.Printf("[ERROR] Failed to parse message: %v", err)
```

**日志级别**：
- `[INFO]`：常规信息
- `[DEBUG]`：调试信息（生产环境关闭）
- `[WARN]`：警告信息
- `[ERROR]`：错误信息

---

#### 前端日志

```python
# frontend/main.py
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

logger.info("Game started")
logger.debug(f"Received message: {msg}")
logger.error(f"Connection failed: {e}")
```

---

## 🤝 贡献指南

### 开发流程

1. **Fork 项目**
   ```bash
   # 在 GitHub 上点击 Fork 按钮
   ```

2. **克隆仓库**
   ```bash
   git clone https://github.com/YOUR_USERNAME/Echo_Trace.git
   cd Echo_Trace
   ```

3. **创建特性分支**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **编写代码**
   - 遵循代码规范（见下文）
   - 添加单元测试
   - 更新文档

5. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

6. **推送分支**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **创建 Pull Request**
   - 在 GitHub 上创建 PR
   - 填写 PR 模板
   - 等待代码审查

---

### 代码规范

#### Go 代码规范

- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码风格
- 函数注释使用 `//` 开头
- 导出函数首字母大写
- 错误处理：始终检查 `error` 返回值

**示例**：
```go
// UpdatePlayerPosition updates the player's position and returns an error if out of bounds.
func (gs *GameState) UpdatePlayerPosition(playerID string, x, y float64) error {
    player, ok := gs.Players[playerID]
    if !ok {
        return fmt.Errorf("player not found: %s", playerID)
    }
    
    player.X = x
    player.Y = y
    return nil
}
```

---

#### Python 代码规范

- 遵循 **PEP 8** 规范
- 使用 `black` 自动格式化
- 使用类型注解（Type Hints）
- 函数文档字符串使用 `"""` 包裹

**示例**：
```python
def calculate_distance(x1: float, y1: float, x2: float, y2: float) -> float:
    """计算两点之间的欧几里得距离。
    
    Args:
        x1, y1: 第一个点的坐标
        x2, y2: 第二个点的坐标
    
    Returns:
        两点之间的距离
    """
    return math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2)
```

---

### Commit 消息规范

使用 **约定式提交 (Conventional Commits)**：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型（type）**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构代码
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**：
```
feat(backend): implement quadtree spatial indexing

- Replace grid-based AOI with quadtree
- Reduce query complexity from O(N) to O(log N)
- Add unit tests for quadtree operations

Closes #42
```

---

### Pull Request 检查清单

- [ ] 代码通过所有单元测试
- [ ] 新功能添加了单元测试
- [ ] 代码符合项目规范（`gofmt`, `black`）
- [ ] 更新了相关文档
- [ ] Commit 消息符合约定式提交规范
- [ ] PR 描述清晰，说明了更改内容和原因

---

## 📚 API 文档生成

### VitePress 文档系统

Echo Trace 使用 **VitePress 1.6.4** 构建开发者文档。

### 启动文档服务器

```bash
cd docs

# 安装依赖（首次）
npm install

# 启动开发服务器
npm run docs:dev

# 访问: http://localhost:9962
```

### 构建静态文档

```bash
cd docs

# 构建
npm run docs:build

# 输出: docs/.vitepress/dist/

# 预览构建结果
npm run docs:preview
```

### 文档结构

```
docs/
├── .vitepress/
│   ├── config.mjs          # VitePress 配置
│   └── cache/              # 缓存文件（gitignore）
├── README.md               # 文档首页
├── api/
│   ├── README.md           # API 概览
│   ├── room-management.md  # 房间管理 API
│   ├── game-actions.md     # 游戏操作 API
│   └── ...
├── protobuf/
│   ├── README.md           # Protobuf 概览
│   ├── message-types.md    # 消息类型目录
│   └── migration-guide.md  # 迁移指南
└── game-logic/
    ├── phases.md           # 游戏阶段
    ├── items.md            # 道具系统
    └── ...
```

### 添加新文档页面

1. 在对应目录创建 Markdown 文件
2. 更新 `.vitepress/config.mjs` 的侧边栏配置

```javascript
// docs/.vitepress/config.mjs
export default {
  themeConfig: {
    sidebar: {
      '/api/': [
        {
          text: 'API 参考',
          items: [
            { text: 'API 概览', link: '/api/' },
            { text: '房间管理', link: '/api/room-management' },
            { text: '新增页面', link: '/api/new-page' }  // 新增
          ]
        }
      ]
    }
  }
}
```

---

<div align="center">

**🛠️ 感谢你为 Echo Trace 做出贡献！**

如需玩家指南，请查看 [PLAYER_GUIDE.md](PLAYER_GUIDE.md)

</div>
