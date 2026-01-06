# Echo Trace (DarkForest-Go)

Echo Trace 是一个基于 **Go（服务端）+ Python/Pygame（客户端）** 的多人在线对战 Demo。项目当前正在从“搜刮/撤离”玩法，过渡到 **AOI 战争迷雾 + 多人在线坦克大战**：在迷宫地形中进行视野受限的信息对抗与弹道战斗。

本仓库以“代码即事实”为准：玩法数值由 `config/*.json`（分模块配置）与服务端逻辑共同决定；客户端负责 UI、输入与渲染。

## 🧭 当前方向（基于近期更新日志）

近期迭代重点（Alpha 1.0 / 1.1）：

- 坦克大战核心重构：弹丸（Projectile）与命中判定、墙体反弹（Bounce）物理
- 开火节奏：装填/冷却时间调整，并在前端加入开火频率限制（防止过量发送）
- 复活（Respawn）逻辑修复
- “残影视野”机制：死亡/离开视野后仍保留约 2 秒的可见性残留（Residual Vision）

说明：旧的搜刮/经济/撤离相关能力仍可能存在于代码中，但产品主轴正在向“坦克战斗 + 迷雾 AOI”收敛。

## 📂 目录结构

```
Echo_Trace/
├── backend/             # Go 服务端（WebSocket + Room + GameLoop + SQLite）
├── frontend/            # Python 客户端（Pygame UI/渲染 + WebSocket 客户端）
├── config/              # 分拆后的配置（server/map/gameplay/items/tactics/combat/phases）
├── game_config.json     # 旧版单文件配置（兼容读取/回退）
├── protocol.json        # WebSocket 协议定义
└── README.md
```

## 🚀 快速开始

### 1) 启动服务端

需要 Go（本项目 `backend/main.go` 通过 `-port` 启动监听）。

```bash
cd backend
go mod tidy
go run main.go
```

- 默认监听：`:8080`
- WebSocket：`ws://<host>:<port>/ws`
- 健康检查：`http://<host>:<port>/health`

如果需要与线上端口对齐，可指定端口：

```bash
go run main.go -port 9191
```

### 2) 启动客户端（源码运行）

需要 Python（建议 3.10+；当前开发环境使用 3.13 也可运行）。

```bash
cd frontend
python -m pip install -r requirements.txt
python main.py
```

客户端 CONNECT 页面需要输入 `ws://<host>:<port>/ws`（示例：`ws://127.0.0.1:8080/ws` 或 `ws://139.224.0.73:9191/ws`）。

### 3) 打包客户端（Windows EXE）

```powershell
cd frontend
./build_exe.ps1 -Clean
```

产物位置：`frontend/dist/Echo_Trace_Client.exe`

## 🎮 游戏流程（概览）

- CONNECT：连接 WebSocket；可选填 Resume ID（`session_id`）用于冷启动续局
- LOGIN：填写“特工代号”（玩家名）
- MENU：创建房间 / 列表加入房间
- 游戏内：战术选择 → 迷雾 AOI 规则下的坦克对战（弹道/反弹/复活等）

更详细的流程说明见 [GameFlow.md](./GameFlow.md)（阶段命名与具体事件仍在迭代，以实现为准）。

## 🎯 基础操作

- `WASD` 移动；鼠标决定面向
- `Space` 开火（存在开火冷却/装填时间，前端会做限频）
- `E` 拾取；`F` 交互/商店
- `1-6` 使用道具 /（商店打开时）购买；`Shift+1-6` 丢弃；`Ctrl+1-6` 出售
- `ESC` 暂停菜单

## 🏠 房间与身份规则

- `session_id` 由“特工代号（玩家名）”派生（如 `agent_<name>`）
- 同一房间内特工代号必须唯一：重复会被拒绝（守护机制）
- 断线重连：服务端在 `server.disconnect_grace_sec` 宽限期内允许同 `session_id` 重连恢复

## 🧩 技术要点

- Server：Go + Gorilla WebSocket；房间隔离；tick 驱动 GameLoop；SQLite 持久化（运行时生成 `backend/game.db`）
- Client：Pygame-ce + websocket-client；状态机 UI（CONNECT/LOGIN/MENU/ROOM_LIST/CONFIG/GAME/PAUSE）
- Protocol：JSON over WebSocket（详见 `protocol.json`）
