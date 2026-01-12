# 🎮 Echo Trace

<div align="center">

**基于迷雾战争的多人在线战术对抗游戏**

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://golang.org/)
[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python)](https://python.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[功能特性](#-功能特性) • [快速开始](#-快速开始) • [文档](#-文档)

</div>

---

## 📖 项目简介

**Echo Trace** 是一款融合了**战争迷雾（FOG）**、**空间感知（AOI）** 和 **AI 对抗**的多人在线战术游戏。玩家在动态生成的迷宫地图中搜刮资源、对抗 AI Boss，并在有限视野下与其他玩家展开策略博弈。

### 核心特性

- 🔍 **战术选择**：开局选择侦察、生存或攻击倾向，获得永久加成
- 👁️ **有限视野**：基于视锥的 FOG 系统，需通过雷达脉冲探测敌人
- 🤖 **AI Boss**：柠白号（NingBye）巡逻战场，威胁所有玩家
- 🎁 **道具系统**：T1-T4 稀有度道具，包括护盾、干扰器、许愿装置等
- 💬 **MCP 集成**：通过自然语言与 AI 交互，触发特殊指令

---

## ✨ 功能特性

### v1.3.0 - 性能优化与协议升级

- ⚡ **四叉树空间索引**：AOI 查询性能提升 50-90%
- 📦 **Protobuf 二进制协议**：带宽占用降低 75%，延迟更低
- 🏠 **空房间自动回收**：房间管理更高效
- 🎯 **战术永久加成**：RECON +20% 视野 | DEFENSE +20% HP | TRAP +20% 护甲穿透

### 技术架构

- **Go 1.23+ + WebSocket**：高性能游戏服务器（50ms Tick）
- **Python 3.10+ + Pygame**：跨平台客户端渲染
- **Protocol Buffers**：二进制序列化协议
- **SQLite**：玩家数据持久化
- **FastAPI + DeepSeek**：MCP 自然语言接口

---

## 🚀 快速开始

### 📋 环境依赖

| 组件 | 版本要求 | 用途 |
|------|---------|------|
| **Go** | 1.23+ | 游戏服务器 |
| **Python** | 3.10+ | 游戏客户端 + MCP Server |
| **Protoc** | 6.x+ | Protobuf 编译（可选） |

### 📦 安装与启动

#### 1️⃣ 克隆仓库

```bash
git clone <repository_url>
cd Echo_Trace
```

#### 2️⃣ 安装依赖

```bash
# 后端依赖
cd backend
go mod download

# 前端依赖
cd ../frontend
pip install -r requirements.txt

# MCP Server（可选）
cd ../Game_MCP_Server
pip install -r requirements.txt
cp .env.example .env
# 编辑 .env 文件，填入 DeepSeek API Key
```

#### 3️⃣ 启动服务

```bash
# Terminal 1: 游戏服务器
cd backend
go run .

# Terminal 2: 游戏客户端
cd frontend
python main.py

# Terminal 3 (可选): MCP Server
cd Game_MCP_Server
python server.py
```

---

## 📚 文档

### 玩家指南

详细的游戏玩法、道具系统、战术选择和 AI Boss 应对策略，请查看：

👉 **[PLAYER_GUIDE.md](PLAYER_GUIDE.md)** - 玩家完整指南

包含内容：
- 🌌 游戏世界观和背景故事
- 🕹️ 完整游戏流程（Phase 0-4）
- 🛠️ 道具系统详解（T1-T4 全道具）
- 🎯 战术选择指南（RECON/DEFENSE/TRAP）
- 🤖 AI Boss 应对技巧
- 🏆 获胜策略与常见问题解答

### 开发者文档

项目架构、技术栈、构建流程和贡献指南，请查看：

👉 **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)** - 开发者完整指南

包含内容：
- 🏗️ 项目架构（Client-Server-MCP）
- 💻 技术栈详解（Go/Python/Protobuf/FastAPI）
- 🔧 开发环境配置
- 📡 Protobuf 使用指南
- 🤖 MCP 服务器开发
- 🧪 测试与调试
- 🤝 贡献指南和代码规范

### API 文档

在线 API 参考文档（VitePress），请启动文档服务器：

```bash
cd docs
npm install     # 首次运行
npm run docs:dev

# 访问: http://localhost:9962
```

包含内容：
- 📡 WebSocket API 参考
- 📦 Protobuf 协议文档
- 🎮 游戏逻辑详解
- 📝 版本更新日志

---
## 📄 许可证

本项目采用 **MIT License** 开源协议。详见 [LICENSE](LICENSE) 文件。

---

## 🙏 致谢

- **Go + Gorilla WebSocket**：高性能网络框架
- **Pygame**：跨平台游戏引擎
- **DeepSeek**：自然语言 AI 支持
- **Protobuf**：高效序列化协议
- **VitePress**：优雅的文档生成工具

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给个 Star！**

Made with ❤️ by Echo Trace Team

</div>

