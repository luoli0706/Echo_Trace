# MCP Server API

Model Context Protocol (MCP) 服务器提供自然语言接口，玩家可通过许愿装置道具发送 AI 指令，实现战术干预。

---

## 服务概述

- **端口**: `http://localhost:9091`
- **框架**: FastAPI
- **LLM**: DeepSeek Chat API
- **权限等级**: 3 级（残缺版、完整版、开发者版）

---

## HTTP API 端点

### POST /wish

接收玩家许愿指令，调用 DeepSeek LLM 解析并执行。

**请求格式**:

```http
POST /wish HTTP/1.1
Content-Type: application/json

{
  "player_id": "550e8400-e29b-41d4-a716-446655440000",
  "wish_text": "传送到最近的引擎",
  "permission_level": 2
}
```

**请求字段**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `player_id` | string | ✅ | 玩家 UUID |
| `wish_text` | string | ✅ | 自然语言指令 |
| `permission_level` | int | ✅ | 权限等级 (1-3) |

**响应格式**:

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

**响应字段**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | bool | 执行是否成功 |
| `action` | string | 操作类型 |
| `params` | object | 操作参数 |
| `message` | string | 用户可读消息 |

---

## 权限等级

### Level 1: 残缺的万能许愿机 (T2)

**允许的操作**:
- 🔍 **查询信息**: 查询最近的引擎、物资、玩家位置
- 🚀 **小范围传送**: 传送距离 < 10m
- 💪 **临时属性提升**: 属性提升 < 20%，持续 < 10s

**示例指令**:
```json
{
  "wish_text": "告诉我最近的引擎在哪",
  "permission_level": 1
}
```

**响应**:
```json
{
  "success": true,
  "action": "query",
  "params": {
    "result": "最近的引擎位于坐标 (45.0, 28.0)，距离你 8.5 米"
  },
  "message": "最近的引擎位于坐标 (45.0, 28.0)，距离你 8.5 米"
}
```

---

### Level 2: 万能许愿机 (T4)

**允许的操作**:
- 🌀 **中等范围传送**: 传送距离 < 50m
- 🔧 **清除障碍**: 清除指定范围内的墙体（5m 半径）
- 🔄 **交换位置**: 与指定玩家交换位置
- 🛡️ **临时增益**: 属性提升 50%，持续 30s

**示例指令**:
```json
{
  "wish_text": "传送到撤离点",
  "permission_level": 2
}
```

**响应**:
```json
{
  "success": true,
  "action": "teleport",
  "params": {
    "target_x": 90.0,
    "target_y": 70.0
  },
  "message": "已传送到撤离点 (90.0, 70.0)"
}
```

---

### Level 3: 开发者遗忘的命令行 (T4)

**允许的操作**:
- 🚀 **无限制传送**: 传送到地图任意位置
- 🎮 **修改游戏参数**: 修改移动速度、HP 上限、视野半径等
- 👾 **召唤实体**: 生成物品、弹药、护盾
- 🤖 **操控 Boss**: 让柠白号巡逻指定位置或攻击指定玩家

**示例指令**:
```json
{
  "wish_text": "让柠白号巡逻撤离点",
  "permission_level": 3
}
```

**响应**:
```json
{
  "success": true,
  "action": "control_boss",
  "params": {
    "target_x": 90.0,
    "target_y": 70.0
  },
  "message": "柠白号已被引导至撤离点巡逻"
}
```

---

## 操作类型

### teleport (传送)

**参数**:
```json
{
  "target_x": 50.0,
  "target_y": 30.0
}
```

**限制**:
- Level 1: 距离 < 10m
- Level 2: 距离 < 50m
- Level 3: 无限制

---

### query (查询)

**参数**:
```json
{
  "result": "查询结果文本"
}
```

**可查询内容**:
- 最近的引擎/物资/玩家
- 当前阶段剩余时间
- 自己的属性状态

---

### heal (治疗)

**参数**:
```json
{
  "hp": 30,
  "armor": 20
}
```

**限制**:
- Level 1: HP/护甲 < 30
- Level 2: HP/护甲 < 50
- Level 3: 无限制

---

### speed_boost (速度提升)

**参数**:
```json
{
  "multiplier": 1.5,
  "duration": 10.0
}
```

**限制**:
- Level 1: 倍率 < 1.2，持续 < 10s
- Level 2: 倍率 < 1.5，持续 < 30s
- Level 3: 无限制

---

### clear_obstacle (清除障碍)

**参数**:
```json
{
  "center_x": 50.0,
  "center_y": 30.0,
  "radius": 5.0
}
```

**限制**:
- Level 1: 不可用
- Level 2: 半径 < 5m
- Level 3: 无限制

---

### spawn_item (生成物品)

**参数**:
```json
{
  "item_name": "超级穿甲弹",
  "tier": 1,
  "x": 50.0,
  "y": 30.0
}
```

**限制**:
- Level 1/2: 不可用
- Level 3: 无限制

---

### control_boss (操控 Boss)

**参数**:
```json
{
  "command": "patrol" | "attack",
  "target_x": 90.0,
  "target_y": 70.0,
  "target_player_id": "player-uuid"
}
```

**限制**:
- Level 1/2: 不可用
- Level 3: 无限制

---

## DeepSeek 集成

### Prompt 模板

```python
SYSTEM_PROMPT = """
你是 Echo Trace 游戏的 AI 助手。玩家会发送许愿指令，你需要将其转换为游戏操作。

权限等级：
- Level 1 (残缺版): 查询信息、小范围传送 (<10m)、临时属性提升 (<20%)
- Level 2 (完整版): 中等范围传送 (<50m)、清除障碍、交换位置
- Level 3 (开发者版): 无限制传送、修改参数、召唤实体、操控 Boss

玩家指令: "{wish_text}"
权限等级: Level {permission_level}

请返回 JSON 格式的操作，包含以下字段：
{
  "action": "teleport|heal|query|speed_boost|clear_obstacle|spawn_item|control_boss",
  "params": {...},
  "message": "操作说明"
}

如果权限不足，返回：
{
  "action": "error",
  "params": {},
  "message": "权限不足，此操作需要 Level X 权限"
}
"""

def parse_wish(wish_text: str, permission_level: int) -> dict:
    response = deepseek.chat.completions.create(
        model="deepseek-chat",
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT.format(
                wish_text=wish_text,
                permission_level=permission_level
            )}
        ]
    )
    return json.loads(response.choices[0].message.content)
```

---

## 游戏逻辑调用

MCP Server 需要调用游戏服务器的内部 API：

### 传送玩家

```python
import requests

def teleport_player(player_id: str, x: float, y: float):
    response = requests.post(
        "http://localhost:8080/api/teleport",
        json={
            "player_id": player_id,
            "x": x,
            "y": y
        }
    )
    return response.json()
```

### 修改玩家属性

```python
def modify_player(player_id: str, hp: int = None, speed: float = None):
    response = requests.post(
        "http://localhost:8080/api/modify_player",
        json={
            "player_id": player_id,
            "hp": hp,
            "speed_multiplier": speed
        }
    )
    return response.json()
```

### 生成物品

```python
def spawn_item(item_name: str, tier: int, x: float, y: float):
    response = requests.post(
        "http://localhost:8080/api/spawn_item",
        json={
            "item_name": item_name,
            "tier": tier,
            "x": x,
            "y": y
        }
    )
    return response.json()
```

---

## 安全限制

### 1. 权限验证

```python
def validate_permission(action: str, level: int) -> bool:
    LEVEL_1_ACTIONS = ["query", "teleport_short", "heal_minor"]
    LEVEL_2_ACTIONS = LEVEL_1_ACTIONS + ["teleport_medium", "clear_obstacle"]
    LEVEL_3_ACTIONS = LEVEL_2_ACTIONS + ["teleport_any", "spawn_item", "control_boss"]
    
    if level == 1:
        return action in LEVEL_1_ACTIONS
    elif level == 2:
        return action in LEVEL_2_ACTIONS
    elif level == 3:
        return action in LEVEL_3_ACTIONS
    
    return False
```

### 2. 频率限制

```python
from collections import defaultdict
from time import time

wish_cooldown = defaultdict(list)
MAX_WISHES_PER_MINUTE = 5

def check_rate_limit(player_id: str) -> bool:
    now = time()
    # 清理 1 分钟前的记录
    wish_cooldown[player_id] = [
        t for t in wish_cooldown[player_id] if now - t < 60
    ]
    
    if len(wish_cooldown[player_id]) >= MAX_WISHES_PER_MINUTE:
        return False  # 超过限制
    
    wish_cooldown[player_id].append(now)
    return True
```

### 3. 内容过滤

```python
BANNED_KEYWORDS = ["delete", "shutdown", "crash", "hack"]

def validate_wish_text(wish_text: str) -> bool:
    lower_text = wish_text.lower()
    for keyword in BANNED_KEYWORDS:
        if keyword in lower_text:
            return False
    return True
```

---

## 客户端集成

### Python 示例

```python
import requests

async def use_wish_item(wish_text: str, permission_level: int):
    response = requests.post(
        "http://localhost:9091/wish",
        json={
            "player_id": player_id,
            "wish_text": wish_text,
            "permission_level": permission_level
        }
    )
    result = response.json()
    
    if result["success"]:
        print(f"✅ {result['message']}")
    else:
        print(f"❌ {result['message']}")
```

### JavaScript 示例

```javascript
async function useWishItem(wishText, permissionLevel) {
  const response = await fetch('http://localhost:9091/wish', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      player_id: playerId,
      wish_text: wishText,
      permission_level: permissionLevel
    })
  });
  
  const result = await response.json();
  if (result.success) {
    showMessage(result.message, 'success');
  } else {
    showMessage(result.message, 'error');
  }
}

// 使用示例
useWishItem("传送到最近的引擎", 1);
```

---

## 性能优化

### 1. LLM 响应缓存

```python
from functools import lru_cache

@lru_cache(maxsize=100)
def cached_parse_wish(wish_text: str, permission_level: int):
    return parse_wish(wish_text, permission_level)
```

### 2. 异步处理

```python
from fastapi import BackgroundTasks

@app.post("/wish")
async def handle_wish(request: WishRequest, background_tasks: BackgroundTasks):
    # 立即返回响应
    background_tasks.add_task(execute_wish, request)
    return {"status": "processing"}
```

---

## 错误处理

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 403 | 权限不足 |
| 429 | 请求频率过高 |
| 500 | 服务器内部错误 |

**错误响应示例**:

```json
{
  "success": false,
  "error_code": 403,
  "message": "权限不足，此操作需要 Level 3 权限"
}
```

---

## 相关文档

- [游戏操作 API](/api/game-actions.html) - USE_ITEM_REQ 调用许愿装置
- [道具系统](/game-logic/items.html) - 许愿装置道具详解
- [开发者指南](../../DEVELOPER_GUIDE.md) - MCP 服务器开发
