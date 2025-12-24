# Echo Trace (DarkForest-Go) Alpha 0.5

> **状态:** Alpha 0.5 - 经济闭环与UI重构 (Economy & UI Refactor)
> **Latest Update:** v0.5 - Room System, Economy, Persistent Data, I18N.

Echo Trace 是一个基于 Golang 和 Python 的高性能后端游戏 Demo，核心玩法结合了 **迷宫搜刮 + AOI 战争迷雾 + 撤离博弈**。

Echo Trace is a high-performance backend game demo featuring **Maze Scavenging + AOI Fog of War + Extraction Mechanics**.
Built with **Golang** (Server) and **Python/Pygame** (Client).

## 📂 目录结构 (Directory Structure)
```
Echo_Trace/
├── backend/            # Golang Server
│   ├── logic/          # 核心逻辑 (Physics, Maze, AOI, Items)
│   ├── network/        # WebSocket & Room Management
│   ├── storage/        # SQLite Persistence
│   └── main.go         # 入口 (Entry Point)
├── frontend/           # Python Client
│   ├── client/         # 客户端模块 (Net, Render, State, Config, I18n)
│   ├── assets/         # 资源文件 (Images, Locales)
│   └── main.py         # 入口 (Entry Point)
├── game_config.json    # 共享配置参数 (Shared Parameters)
├── protocol.json       # 网络协议定义 (Network Protocol Schema)
└── README.md           # 说明文档 (Documentation)
```

## 🚀 快速开始 (Quick Start)

### 1. 启动服务端 (Start Server)
需要 Go 1.18+。
```bash
cd backend
go mod tidy
go run main.go
```
*默认监听 :8080 端口。*

### 2. 启动客户端 (Start Client)
需要 Python 3.10+。
```bash
cd frontend
pip install pygame-ce websocket-client
python main.py
```
*支持开启多个客户端模拟多玩家。*

## 🎮 玩法指南 (Gameplay Guide)

### 操作 (Controls)
*   **WASD:** 移动角色 (Move Character 🏃)
*   **E:** 拾取物品 (Pick up Item 📦)
*   **F:** 交互 / 修复电机 / 打开商店 (Interact / Fix Motor ⚡ / Shop 💰)
*   **1-3:** 使用物品 (Use Item) / 购买商品 (Buy Item)
*   **Shift + 1-3:** 丢弃物品 (Drop Item)
*   **Ctrl + 1-3:** 出售物品 (Sell Item - 需在商人附近)
*   **ESC:** 暂停菜单 / 退出界面 (Pause / Close Menu)

### 游戏阶段 (Phases)
1.  **搜寻 (SEARCH):** 在黑暗中搜刮物资，寻找商人购买装备。
2.  **冲突 (CONFLICT):** 电机 (⚡) 刷新。修复 2 个电机以开启撤离点，或消灭对手。
    *   *机制:* 电机每 15 秒发出脉冲暴露位置。
3.  **撤离 (ESCAPE):** 出口 (🚪) 开启。到达出口并坚持 3 秒即可撤离胜利。

### 特色系统 (Features)
*   **AOI 战争迷雾:** 只能看到视野范围内的物体。
*   **声音可视化:** 听觉范围内的脚步声会以波纹形式显示方向。
*   **经济系统:** 搜刮物资、撤离带出物品均可获得资金，用于购买高级装备。
*   **高价值空投:** 每阶段开始时在地图中心刷新空投 (🎁)，包含高级装备和大量资金，全图可见。
*   **持久化:** 玩家名称、资金和库存会保存至 SQLite 数据库。

## 🛠 技术栈 (Tech Stack)
*   **Server:** Go (Gorilla WebSocket), Mutex-protected GameState, Grid-based Map, SQLite.
*   **Client:** Pygame CE, Interpolated Rendering, Cyberpunk UI style, I18N support.
*   **Protocol:** JSON-over-WebSocket (Phase-driven state sync).
