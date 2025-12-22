# Echo Trace (DarkForest-Go) Alpha

这是一个基于 Golang 后端和 Python (Pygame) 前端的高性能游戏 Demo。
核心玩法是 **迷宫搜刮 + AOI 视野博弈 + 撤离机制**。

## 目录结构
```
Echo_Trace/
├── backend/            # Golang 服务端
│   ├── logic/          # 游戏核心逻辑 (Map, AOI, Physics)
│   ├── network/        # WebSocket 网络层
│   └── main.go         # 入口
├── frontend/           # Python 客户端
│   ├── client/         # 模块代码
│   └── main.py         # 入口
├── game_config.json    # 游戏数值配置
├── protocol.json       # 通信协议定义
└── README.md           # 本文档
```

## 快速开始

### 1. 启动服务端
确保已安装 Go 1.18+。

```bash
cd backend
# 安装依赖
go mod tidy
# 运行
go run main.go
```
*服务端默认监听 :8080 端口。*

### 2. 启动客户端
确保已安装 Python 3.10+。

```bash
cd frontend
# 安装依赖
pip install pygame-ce websocket-client
# 运行
python main.py
```

### 3. 操作说明
*   **WASD:** 移动玩家
*   **视野:** 你只能看到以自己为中心的圆圈内的内容。
*   **目标:** 探索迷宫，避开红色鬼脸 (👹)，寻找出口。

## 调试与开发
*   修改 `game_config.json` 可以实时调整地图大小、视野半径等参数（需重启服务端生效）。
*   可以使用 `--bot-count 10` (待实现) 来添加机器人。
