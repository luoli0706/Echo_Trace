# 🎮 Echo Trace

<div align="center">

**基于迷雾战争的多人在线战术对抗游戏**

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://golang.org/)
[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python)](https://python.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[功能特性](#-功能特性) • [快速开始](#-快速开始) • [游戏玩法](#-游戏玩法) • [开发文档](#-开发文档)

</div>

---

## 📖 游戏简介

**Echo Trace** 是一款融合了**战争迷雾（FOG）**、**空间感知（AOI）** 和 **AI 对抗**的多人在线战术游戏。玩家在动态生成的迷宫地图中搜刮资源、对抗 AI Boss，并在有限视野下与其他玩家展开策略博弈。

### 🎯 核心玩法
- **战术选择**：开局选择侦察、生存或攻击倾向，获得永久加成
- **有限视野**：基于视锥的 FOG 系统，需通过雷达脉冲探测敌人
- **AI Boss**：柠白号（NingBye）巡逻战场，发射高穿透子弹威胁所有玩家
- **道具系统**：T1-T4 稀有度道具，包括护盾、干扰器、许愿装置等
- **MCP 集成**：通过自然语言与 AI 交互，触发特殊指令

### 🏆 游戏目标
- **Phase 1 (搜索阶段)**：搜刮物资，修复引擎
- **Phase 2 (冲突阶段)**：雷达脉冲激活，玩家互相暴露位置
- **Phase 3 (逃离阶段)**：地图缩圈，最后的生存竞赛

---

## ✨ 功能特性

### 🔥 v1.3.0 - 性能优化与协议升级
- **四叉树空间索引**：AOI 查询性能提升 50-90%
- **Protobuf 二进制协议**：带宽占用降低 75%，延迟更低
- **空房间自动回收**：房间管理更高效
- **战术永久加成**：RECON +20% 视野 | DEFENSE +20% HP | TRAP +20% 护甲穿透

### 🤖 AI 系统
- **柠白号 Boss**：巡逻索敌，高伤害重型火力，周期性雷达脉冲
- **A* 寻路**：基于连通分量的智能移动
- **MCP 自然语言接口**：通过"许愿装置"与 DeepSeek LLM 交互

### 🎨 技术亮点
- **Go + WebSocket**：高性能游戏服务器（50ms Tick）
- **Python + Pygame**：跨平台客户端渲染
- **SQLite**：玩家数据持久化
- **双协议支持**：JSON（房间管理）+ Protobuf（游戏数据）

---

## 🚀 快速开始

### 📋 环境依赖

| 组件 | 版本要求 | 用途 |
|------|---------|------|
| **Go** | 1.23+ | 游戏服务器 |
| **Python** | 3.10+ | 游戏客户端 + MCP Server |
| **Protoc** | 6.x+ | Protobuf 编译（可选） |

### 📦 安装步骤

#### 1️⃣ 克隆仓库
```bash
git clone <repository_url>
cd Echo_Trace
```

#### 2️⃣ 配置 MCP Server（可选）
```bash
cd Game_MCP_Server
pip install -r requirements.txt

# 复制环境变量模板
cp .env.example .env

# 编辑 .env，填入 DeepSeek API Key
# AI_API_KEY=your_deepseek_api_key_here
```

#### 3️⃣ 安装 Go 依赖
```bash
cd ../backend
go mod download
```

#### 4️⃣ 安装 Python 依赖
```bash
cd ../frontend
pip install -r requirements.txt
```

### ▶️ 启动游戏

#### 方式一：完整体验（含 MCP）
```bash
# Terminal 1: 启动 MCP Server
cd Game_MCP_Server
python server.py
# 监听: http://localhost:9091

# Terminal 2: 启动游戏服务器
cd backend
go run .
# 监听: :8080

# Terminal 3: 启动客户端
cd frontend
python main.py
```

#### 方式二：基础游戏（不含 MCP）
```bash
# Terminal 1: 启动游戏服务器
cd backend
go run .

# Terminal 2: 启动客户端
cd frontend
python main.py
```

---

## 🎮 游戏玩法

### 🕹️ 基础操作

| 按键 | 功能 |
|------|------|
| **WASD** | 移动角色 |
| **鼠标** | 控制朝向/视野方向 |
| **Space** | 开火（注意弹药与冷却） |
| **E** | 拾取物品 |
| **F** | 交互（商人/引擎） |
| **1-6** | 使用道具栏物品 |
| **Shift+1-6** | 丢弃物品 |
| **Ctrl+1-6** | 出售物品（需在商人附近） |
| **ESC** | 打开设置菜单 |

### 📊 战术倾向（开局选择）

在游戏开始前，使用 **数字键 1-3** 选择战术倾向：

| 倾向 | 快捷键 | 效果 |
|------|-------|------|
| **侦察 (RECON)** | 1 | 👁️ 视野半径 +20% |
| **生存 (DEFENSE)** | 2 | 🛡️ 最大生命值 +20% |
| **攻击 (TRAP)** | 3 | ⚔️ 护甲穿透 +20% |

### 🎯 游戏阶段

#### Phase 0: 战术选择 (开局)
- 使用数字键 **1-3** 选择战术倾向
- 获得永久属性加成（侦察/生存/攻击）
- 所有玩家完成选择后自动进入 Phase 1

#### Phase 1: 搜索阶段 (默认 5 分钟)
- 搜刮地图上的物资（护盾、弹药、道具）
- 寻找并修复引擎（通过 **F 键交互**）
- 与商人交易升级装备
- 躲避 AI Boss "柠白号"
- **目标**：激活全部引擎进入下一阶段

#### Phase 2: 冲突阶段 (默认 5 分钟)
- 引擎定期发出雷达脉冲（默认 15 秒/次，持续 5 秒）
- 玩家位置周期性暴露在雷达上
- 竞争稀缺物资，高强度 PvP 与 PvE
- 柠白号加强巡逻，战斗频率提升

#### Phase 3: 逃离阶段 (默认 2 分钟)
- 地图边缘开始缩圈（伤害区）
- 撤离点开启（先到先得）
- 最后的生存竞赛
- **目标**：到达撤离点并按 **F 键** 撤离

#### Phase 4: 游戏结束
- 显示统计数据（击杀、存活时间、道具使用等）
- 房间在 60 秒后自动关闭

### 🛠️ 道具系统

游戏内道具分为 **4 个等级**（T1-T4），按类型划分为 **攻击、生存、侦察、综合** 四大类。稀有度越高，效果越强，刷新概率越低。

#### T1 - 基础道具
- **超级穿甲弹**：3 发强化射击，无视 20% 护甲造成真实伤害
- **回光返照**：立即修复 +30% HP / +20 护甲，超出上限部分转化为临时护甲
- **视距提升**：视野半径 +20%，持续 15 秒
- **闪灵瞬步**：向履带朝向瞬移 2.0m，不可穿墙

#### T2 - 进阶道具
- **光学折射效应**：3 发可反弹子弹，碰墙最多反弹 3 次，具有 50% 护甲穿透
- **绝对防御**：进入 3 秒无敌状态（相位偏移），期间无法移动和攻击
- **光锥之外**：牺牲正前方 90° 视野，获得侧后方 270° 视野，持续 10 秒
- **超视距·追踪**：小地图每 2 秒闪烁显示最近玩家红点
- **残缺的万能许愿机**：向 MCP 发送简单指令（受限权限）

#### T3 - 稀有道具
- **超视距·电磁炮**：1 发瞬时射线，无视障碍造成 75 伤害，预警 0.5 秒
- **主动防御**：清除所有标记/追踪状态，2 秒全免疫
- **全境扫描终端**：锁定最近玩家并全图同步坐标，持续 30 秒
- **无形之物**：完全透明化，攻击或撞击后失效

#### T4 - 传说道具
- **超视距·精通天理**：轨道打击，对超大半径区域造成 125 初始 + 100 持续伤害，全图广播预警
- **我就是柠白号**：瞬间同步 Boss 级属性（HP/护甲完全同步为柠白号数值）
- **全频段阻塞干扰**：全图黑盒化，屏蔽所有雷达和小地图 30 秒
- **万能许愿机**：向 MCP 发送指令，实现战术干预（传送、清除障碍、交换位置等）
- **开发者遗忘的命令行**：继承许愿机能力，额外获得系统级权限（修改参数、召唤实体、操控 Boss）

### 🤖 AI Boss - 柠白号

**特性**：
- 🚶 随机巡逻地图连通区域
- 👁️ 3 米感知半径，检测到玩家自动追击
- 💥 重型子弹：50 伤害 + 75% 护甲穿透
- 📡 周期性雷达脉冲（30 秒一次，持续 3 秒）

**应对策略**：
- 保持距离，利用视野盲区
- 使用迷彩道具隐身
- 引诱 Boss 攻击其他玩家
- 利用"柠白号控制器"操纵 Boss

---

## 🏠 房间管理

### 创建房间
1. 主菜单 → **创建房间**
2. 配置游戏参数：
   - 地图大小（50x50 ~ 200x200）
   - 玩家上限（1-20）
   - 阶段时长
   - 物品生成率
3. **导出设置**按钮：保存配置到剪贴板
4. **导入设置**按钮：从剪贴板加载配置

### 加入房间
- **加入行动**：自动重连上次房间（断线重连）
- **房间列表**：查看所有活跃房间
- 断线后有 30 秒宽限期，超时会被踢出

### 房间回收
- ✅ 游戏结束后 5 秒自动回收
- ✅ 所有玩家离开后立即释放
- ✅ **退出房间**按钮（设置菜单）：立即退出返回大厅

---

## 🔧 开发文档

### 📁 项目结构
```
Echo_Trace/
├── backend/              # Go 游戏服务器
│   ├── logic/           # 游戏逻辑（状态机、AI、物品）
│   ├── network/         # WebSocket 通信
│   ├── storage/         # SQLite 数据库
│   └── proto/           # Protobuf 生成代码
├── frontend/            # Python/Pygame 客户端
│   ├── client/          # 网络层、渲染器
│   ├── assets/          # 贴图、音效、本地化
│   └── proto/           # Protobuf Python 代码
├── Game_MCP_Server/     # AI 许愿装置后端
│   ├── server.py        # FastAPI 服务
│   └── .env.example     # 环境变量模板
├── proto/               # Protobuf 协议定义
│   └── echo_trace.proto
└── docs/                # 开发者文档（VuePress）
```

### 🛠️ 构建与编译

#### 编译服务器
```bash
cd backend
go build -o echo_trace_server .
./echo_trace_server
```

#### 打包客户端（PyInstaller）
```bash
cd frontend
pyinstaller --onefile --windowed main.py
# 输出: dist/main.exe
```

#### 重新生成 Protobuf 代码
```bash
# 需要先安装 protoc (https://github.com/protocolbuffers/protobuf/releases)
protoc --go_out=backend --go_opt=paths=source_relative proto/echo_trace.proto
protoc --python_out=frontend proto/echo_trace.proto
```

### 📡 网络协议

#### WebSocket 端口
- 游戏服务器：`:8080/ws`
- MCP Server：`http://localhost:9091`

#### 消息类型
- **JSON**（文本帧）：房间管理（CREATE_ROOM, JOIN_ROOM, LIST_ROOMS）
- **Protobuf**（二进制帧）：游戏数据（MOVE, FIRE, StateSnapshot）

#### 协议文档
详见 [BREAKING_CHANGES.md](BREAKING_CHANGES.md) 和 [PROTOBUF_COMPLETION.md](PROTOBUF_COMPLETION.md)

### 🧪 测试与调试

#### 单元测试
```bash
# Go 后端测试
cd backend
go test ./...

# Python 前端测试
cd frontend
pytest
```

#### 性能分析
```bash
# Go pprof
go run . -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Python cProfile
python -m cProfile -o output.prof main.py
```

---

## 🗺️ 路线图

### ✅ v1.3.0 (当前版本)
- [x] 四叉树空间索引
- [x] Protobuf 二进制协议
- [x] 战术永久加成
- [x] 房间生命周期优化

### 🚧 v1.4.0 (计划中)
- [ ] S2C StateSnapshot Protobuf 迁移
- [ ] 增量状态更新（Delta Snapshot）
- [ ] 客户端预测与服务器和解
- [ ] 观战模式

### 🔮 v2.0.0 (未来)
- [ ] 排行榜与赛季系统
- [ ] 自定义地图编辑器
- [ ] 更多 AI 单位与 Boss
- [ ] 团队模式（2v2v2）

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发流程
1. Fork 项目
2. 创建特性分支：`git checkout -b feature/AmazingFeature`
3. 提交更改：`git commit -m 'Add some AmazingFeature'`
4. 推送分支：`git push origin feature/AmazingFeature`
5. 提交 Pull Request

### 代码规范
- **Go**：遵循 `gofmt` 和 `golint`
- **Python**：遵循 PEP 8，使用 `black` 格式化
- **Commit**：使用约定式提交（Conventional Commits）

---

## 📄 许可证

本项目采用 **MIT License** 开源协议。详见 [LICENSE](LICENSE) 文件。

---

## 🙏 致谢

- **Go + Gorilla WebSocket**：高性能网络框架
- **Pygame**：跨平台游戏引擎
- **DeepSeek**：自然语言 AI 支持
- **Protobuf**：高效序列化协议

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给个 Star！**

Made with ❤️ by Echo Trace Team

</div>

