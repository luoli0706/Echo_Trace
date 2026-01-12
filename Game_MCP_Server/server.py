import os
import json
import requests
import uvicorn
import re
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from dotenv import load_dotenv
from fastmcp import FastMCP

# 加载环境变量
load_dotenv()

AI_API_KEY = os.getenv("AI_API_KEY")
AI_API_URL = os.getenv("AI_API_URL")

# 初始化 MCP (用于管理工具)
mcp = FastMCP("Game MCP Server")
GAME_BACKEND_URL = "http://localhost:8080"

# --- Item Mapping (Context for LLM) ---
ITEM_MAPPING = """
Available Items (Name -> ID):
[攻击类 / Offense]
- 超级穿甲弹 (AP Ammo) -> WPN_AP_AMMO
- 光学折射效应 (Reflect/Bounce Ammo) -> WPN_BOUNCE_AMMO
- 超视距·电磁炮 (Railgun) -> WPN_RAILGUN
- 超视距·精通天理 (Heaven Ray) -> WPN_HEAVEN_RAY

[生存类 / Survival]
- 回光返照 (Repair Kit) -> SURV_REPAIR
- 绝对防御 (Phase Shift/Invincible) -> SURV_PHASE_SHIFT
- 主动防御 (Purge System) -> SURV_PURGE
- 我就是柠白号 (NingBye Mode) -> SURV_NINGBYE_MODE

[侦察类 / Recon]
- 视距提升 (Scope) -> RECON_SCOPE
- 光锥之外 (Rear Sensor) -> RECON_SENSOR
- 全境扫描终端 (Global Scan) -> RECON_SCANNER
- 全频段阻塞干扰 (Global Jammer) -> RECON_JAMMER

[综合类 / Utility]
- 闪灵瞬步 (Blink) -> UTIL_BLINK
- 超视距·追踪 (Radar) -> UTIL_RADAR
- 无形之物 (Stealth) -> UTIL_STEALTH
- 残缺的万能许愿机 (Broken Wish Machine) -> UTIL_WISH_MACHINE
- 开发者遗忘的命令行 (Developer Forgotten CLI) -> UTIL_DEV_FORGOTTEN_CLI
"""

# --- Tools Definition ---
def _call_backend(endpoint: str, data: dict) -> str:
    try:
        url = f"{GAME_BACKEND_URL}{endpoint}"
        resp = requests.post(url, json=data)
        if resp.status_code == 200:
            return f"Success: {resp.text}"
        return f"Failed (Status {resp.status_code}): {resp.text}"
    except requests.exceptions.RequestException as e:
        return f"Error calling game backend: {str(e)}"

# Define raw functions first
def modify_player_health(session_id: str, hp: float) -> str:
    """Modify player's health (0-500)."""
    return _call_backend("/admin/player/health", {"session_id": session_id, "hp": hp})

def modify_player_armor(session_id: str, armor: float) -> str:
    """Modify player's armor (0-250)."""
    return _call_backend("/admin/player/armor", {"session_id": session_id, "armor": armor})

def modify_inventory_capacity(session_id: str, capacity: int) -> str:
    """Modify inventory capacity (0-8)."""
    return _call_backend("/admin/player/inventory/capacity", {"session_id": session_id, "capacity": capacity})

def add_items_to_inventory(session_id: str, item_ids: list[str]) -> str:
    """Add items (max 2) to inventory. IDs: WPN_AP_AMMO, WPN_RAILGUN, SURV_MEDKIT, UTIL_WISH_MACHINE, etc."""
    # Forbidden: Wishing for more wishes
    if "UTIL_WISH_MACHINE" in item_ids or "UTIL_DEV_FORGOTTEN_CLI" in item_ids:
        return "DENIED: You cannot wish for a Wish Device. (Causality Paradox prevented)"
        
    return _call_backend("/admin/player/inventory/item", {"session_id": session_id, "item_ids": item_ids})

def modify_player_speed(session_id: str, multiplier: float, duration_sec: float) -> str:
    """Modify speed multiplier for a duration."""
    return _call_backend("/admin/player/speed", {"session_id": session_id, "multiplier": multiplier, "duration": duration_sec})

def modify_special_attribute(session_id: str, stealth_mode: bool) -> str:
    """Set special attribute (stealth)."""
    return f"Set stealth to {stealth_mode} (Mock)"

def set_player_threat(session_id: str, is_threat: bool) -> str:
    """Mark a player as a 'Threat' to the NingBye AI system."""
    return _call_backend("/admin/player/threat", {"session_id": session_id, "is_threat": is_threat})

def command_ai_patrol(target_x: float, target_y: float) -> str:
    """Command the NingBye AI unit to move to a specific coordinate."""
    return _call_backend("/admin/ai/command", {"target_x": target_x, "target_y": target_y})

def move_player_to_coordinate(session_id: str, x: float, y: float) -> str:
    """Move a player to a target coordinate (server validates collision)."""
    return _call_backend("/admin/player/move", {"session_id": session_id, "x": x, "y": y})

def sanitize_plain_text(text: str) -> str:
    if not text:
        return ""
    # Strip common Markdown tokens; keep only plain text.
    text = text.replace("```", "")
    text = text.replace("`", "")
    text = text.replace("**", "")
    text = text.replace("*", "")
    text = text.replace("_", "")
    text = re.sub(r"^#{1,6}\\s+", "", text, flags=re.MULTILINE)
    text = re.sub(r"\\[([^\\]]+)\\]\\([^\\)]+\\)", r"\\1", text)
    return text.strip()

# --- FastAPI App ---
app = FastAPI()

class WishRequest(BaseModel):
    session_id: str
    wish: str
    item_id: str | None = None

# Manually construct OpenAI-compatible tool definitions based on our mcp tools
# (Since we know what they are, hardcoding schema is safer than relying on internal fastmcp APIs for this demo)
TOOLS_SCHEMA = [
    {
        "type": "function",
        "function": {
            "name": "modify_player_health",
            "description": "Modify player's HP. Use when user asks to heal, set hp, become invincible (high hp).",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string"},
                    "hp": {"type": "number", "description": "Target HP value (0-500)"}
                },
                "required": ["session_id", "hp"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "modify_player_armor",
            "description": "Modify player's Armor.",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string"},
                    "armor": {"type": "number", "description": "Target Armor value (0-250)"}
                },
                "required": ["session_id", "armor"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "modify_inventory_capacity",
            "description": "Expand or shrink inventory bag size.",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string"},
                    "capacity": {"type": "integer", "description": "New slot count (0-8)"}
                },
                "required": ["session_id", "capacity"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "add_items_to_inventory",
            "description": f"Give items to player. Refer to the System Prompt for the list of valid Item IDs.",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string"},
                    "item_ids": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "List of Item IDs (e.g. WPN_RAILGUN, RECON_SCANNER)"
                    }
                },
                "required": ["session_id", "item_ids"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "modify_player_speed",
            "description": "Change player movement speed.",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string"},
                    "multiplier": {"type": "number", "description": "Speed factor (e.g. 1.5, 2.0)"},
                    "duration_sec": {"type": "number", "description": "Duration in seconds"}
                },
                "required": ["session_id", "multiplier", "duration_sec"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "set_player_threat",
            "description": "Mark or unmark player as a threat to AI. Use when user wants to be attacked or ignored by AI.",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string"},
                    "is_threat": {"type": "boolean", "description": "True to be targeted, False to be ignored"}
                },
                "required": ["session_id", "is_threat"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "command_ai_patrol",
            "description": "Command the NingBye AI to move to a specific location (Route 2).",
            "parameters": {
                "type": "object",
                "properties": {
                    "target_x": {"type": "number", "description": "Target X coordinate"},
                    "target_y": {"type": "number", "description": "Target Y coordinate"}
                },
                "required": ["target_x", "target_y"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "move_player_to_coordinate",
            "description": "Move a player to a target coordinate. Use when user asks to relocate someone to X,Y.",
            "parameters": {
                "type": "object",
                "properties": {
                    "session_id": {"type": "string", "description": "Target player's session_id"},
                    "x": {"type": "number", "description": "Target X coordinate"},
                    "y": {"type": "number", "description": "Target Y coordinate"}
                },
                "required": ["session_id", "x", "y"]
            }
        }
    }
]

TOOL_MAP = {
    "modify_player_health": modify_player_health,
    "modify_player_armor": modify_player_armor,
    "modify_inventory_capacity": modify_inventory_capacity,
    "add_items_to_inventory": add_items_to_inventory,
    "modify_player_speed": modify_player_speed,
    "modify_special_attribute": modify_special_attribute,
    "set_player_threat": set_player_threat,
    "command_ai_patrol": command_ai_patrol,
    "move_player_to_coordinate": move_player_to_coordinate
}

@app.post("/wish")
def process_wish(req: WishRequest):
    if not AI_API_KEY:
        raise HTTPException(status_code=500, detail="AI_API_KEY not configured")

    item_id = (req.item_id or "UTIL_WISH_MACHINE").strip()

    # 1. Call LLM
    messages = [
        {"role": "system", "content": f"""You are a Game Master for Echo Trace. Interpret the user's wish and call the appropriate tools to fulfill it.

CRITICAL RULES:
- Do NOT ask the player for confirmation. Infer intent and act.
- Do NOT ask for session_id; it is provided.
- If it is impossible to execute, respond with a short plain-text reason and ask the player to re-enter the wish.

CONTEXT:
- Wishing player's session_id is '{req.session_id}'.
- Wish device item_id is '{item_id}'.

If the user asks for items, use 'add_items_to_inventory' with the IDs from the list below.

{ITEM_MAPPING}

If the user's request requires multiple actions (e.g. "expand inventory AND give item"), you MUST generate multiple tool calls in the same response.

If the wish is ambiguous, make a best-effort guess. Be generous but within limits."""},
        {"role": "user", "content": f"I wish for: {req.wish}"}
    ]

    payload = {
        "model": "deepseek-chat", # or whichever model is available
        "messages": messages,
        "tools": TOOLS_SCHEMA
    }

    def call_llm(msgs: list[dict]):
        p = dict(payload)
        p["messages"] = msgs
        resp = requests.post(
            AI_API_URL,
            headers={"Authorization": f"Bearer {AI_API_KEY}", "Content-Type": "application/json"},
            json=p,
            timeout=10,
        )
        resp.raise_for_status()
        return resp.json()

    try:
        data = call_llm(messages)
        print(f"[MCP DEBUG] LLM Response: {json.dumps(data, ensure_ascii=False)}")
    except Exception as e:
        print(f"[MCP DEBUG] LLM Call Error: {str(e)}")
        return {"status": "error", "results": [], "玩家可见回应": sanitize_plain_text("许愿系统暂时不可用，请稍后重试。")}

    # 2. Process Tool Calls
    results = []
    choice = data.get("choices", [{}])[0]
    message = choice.get("message", {})
    tool_calls = message.get("tool_calls", [])

    if not tool_calls:
        print("[MCP DEBUG] No tool calls in response. Retrying once...")
        retry_messages = [
            messages[0],
            {"role": "system", "content": "Second attempt: You MUST call at least one available tool if possible. If impossible, reply with a short plain-text reason asking the player to re-enter."},
            messages[1],
        ]
        try:
            data = call_llm(retry_messages)
            choice = data.get("choices", [{}])[0]
            message = choice.get("message", {})
            tool_calls = message.get("tool_calls", [])
        except Exception as e:
            print(f"[MCP DEBUG] LLM Retry Error: {str(e)}")
            tool_calls = []

    if not tool_calls:
        return {"status": "needs_retry", "results": [], "玩家可见回应": sanitize_plain_text("我无法执行这个愿望（缺少可用指令或信息不足）。请换一种说法重新输入。")}

    for tc in tool_calls:
        func_name = tc["function"]["name"]
        args_str = tc["function"]["arguments"]
        print(f"[MCP DEBUG] Tool Call: {func_name} Args: {args_str}")
        try:
            args = json.loads(args_str)
            if func_name in TOOL_MAP:
                # Inject session_id if missing (though Prompt usually adds it)
                if "session_id" not in args:
                    args["session_id"] = req.session_id
                
                # Execute Tool
                res = TOOL_MAP[func_name](**args)
                print(f"[MCP DEBUG] Tool Result: {res}")
                results.append(f"{func_name}: {res}")
            else:
                results.append(f"{func_name}: Unknown Tool")
        except Exception as e:
            print(f"[MCP DEBUG] Tool Error: {e}")
            results.append(f"{func_name}: Error {str(e)}")

    ok_count = sum(1 for r in results if isinstance(r, str) and "Success:" in r)
    reply = f"已执行 {len(results)} 个指令（成功 {ok_count} 个）。"
    if ok_count == 0:
        reply = "指令已尝试执行，但未成功生效。你可以换一种说法重新输入愿望。"

    return {"status": "success", "results": results, "玩家可见回应": sanitize_plain_text(reply)}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=9091)
