# Echo Trace (DarkForest-Go)

Echo Trace 是一个基于 **Go（服务端）+ Python/Pygame（客户端）** 的多人在线对战 Demo。项目当前正在从“搜刮/撤离”玩法，过渡到 **AOI 战争迷雾 + 多人在线坦克大战**：在迷宫地形中进行视野受限的信息对抗与弹道战斗。

本仓库引入了 **AI 辅助博弈 (MCP Server)** 与 **高威胁 AI 单位**，使游戏从单纯的玩家对抗演变为“玩家 vs 环境 (PvE) vs AI 指令”的多维战场。

## 🧭 当前方向（Ver 1.2 - 终极插件与 AI 更新）

- **AI Boss - 柠白号 (NingBye's Tunk)**：第一阶段即生成的巨型威胁单位，具备巡逻、索敌、高穿透重型火力及雷达脉冲机制。
- **万能许愿机 (MCP Integration)**：集成 DeepSeek LLM，通过 **Model Context Protocol (MCP)** 允许玩家使用自然语言改变战场逻辑。
- **Tier 4 终极道具**：引入“精通天理”、“全频段阻塞”、“我就是柠白号”等足以逆转局势的传说级插件。
- **坦克大战核心重构**：完善弹丸（Projectile）命中判定、护甲穿透物理及残留视野机制。

## 📂 目录结构

```
Echo_Trace/
├── backend/             # Go 服务端（WebSocket + Room + GameLoop + SQLite）
├── frontend/            # Python 客户端（Pygame UI/渲染 + WebSocket 客户端）
├── Game_MCP_Server/     # Python MCP 服务端（FastAPI + DeepSeek LLM）
├── config/              # 分拆后的配置（server/map/gameplay/items/ai等）
├── items_pro.md         # 最新的 T1-T4 道具设计规范
├── protocol.json        # WebSocket 协议定义
└── README.md
```

## 🚀 快速开始

### 1) 启动 MCP 服务端（指令解析）
用于处理“万能许愿机”的自然语言请求。需要 Python 3.10+。

```bash
cd Game_MCP_Server
pip install -r requirements.txt
# 在 .env 中配置 AI_API_KEY
python server.py
```
默认监听：`http://localhost:9091`

### 2) 启动游戏服务端
需要 Go 环境。

```bash
cd backend
go run .
```
默认监听：`:8080`

### 3) 启动客户端
```bash
cd frontend
python main.py
```

## 🎮 核心系统

### 🤖 柠白号 AI (Boss)
- **生成**：游戏开始后在随机位置生成，1.5 倍于普通玩家体型。
- **行为**：沿连通分量随机巡逻；检测到 3m 内玩家或标记为 `Threat` 的玩家时进入战斗。
- **火力**：发射 50 伤害的重型子弹，具有 75% 护甲穿透。
- **侦测**：每 30 秒发出一次持续 3 秒的小地图雷达脉冲。

### 🌟 万能许愿机 (Wish Machine)
- **获取**：掉落、商店购买或在设置页面通过“获得一个许愿机”按钮（测试用）获取。
- **使用**：点击道具槽位激活输入框（上限 20 字），支持中文。
- **逻辑**：请求由 MCP Server 发送至 LLM，可实现：补充状态、发放道具、传送、命令 Boss 巡逻至特定坐标、修改玩家威胁度等。

## 🎯 基础操作

- `WASD` 移动；鼠标决定面向
- `Space` 开火（注意装填冷却）
- `E` 拾取；`F` 交互 /（靠近商人时）打开商店
- `1-6` 使用道具 / 激活许愿机输入框；`Shift+1-6` 丢弃；`Ctrl+1-6` 出售
- `ESC` 暂停菜单（可在 Settings 中快速获取测试道具）

## 🏠 技术要点

- **MCP Bridge**：游戏后端通过 HTTP 异步通知 MCP Server，实现非阻塞的 AI 指令理解。
- **Armor Penetration**：新增穿透计算公式，使重型火力和 AP 弹药对高护甲单位更具威胁。
- **Global Jamming**：T4 道具可触发全图信号阻塞，屏蔽所有玩家雷达并显示 "SIGNAL LOST"。
- **Stat Synchronization**：服务端 Tick (50ms) 驱动的状态全量/增量同步。