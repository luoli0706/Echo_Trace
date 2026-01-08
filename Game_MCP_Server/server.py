import os
import json
import requests
import uvicorn
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

[生存类 / Survival]
- 回光返照 (Repair Kit) -> SURV_REPAIR
- 绝对防御 (Phase Shift/Invincible) -> SURV_PHASE_SHIFT
- 主动防御 (Purge System) -> SURV_PURGE

[侦察类 / Recon]
- 视距提升 (Scope) -> RECON_SCOPE
- 光锥之外 (Rear Sensor) -> RECON_SENSOR
- 全境扫描终端 (Global Scan) -> RECON_SCANNER

[综合类 / Utility]
- 闪灵瞬步 (Blink) -> UTIL_BLINK
- 超视距·追踪 (Radar) -> UTIL_RADAR
- 无形之物 (Stealth) -> UTIL_STEALTH
- 万能许愿机 (Wish Machine) -> UTIL_WISH_MACHINE
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

# --- FastAPI App ---
app = FastAPI()

class WishRequest(BaseModel):
    session_id: str
    wish: str

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
    "command_ai_patrol": command_ai_patrol
}

@app.post("/wish")
def process_wish(req: WishRequest):
    if not AI_API_KEY:
        raise HTTPException(status_code=500, detail="AI_API_KEY not configured")

    # 1. Call LLM
    messages = [
        {"role": "system", "content": f"""You are a Game Master for Echo Trace. Interpret the user's wish and call the appropriate tools to fulfill it. 
        
        CRITICAL: The player's session_id is '{req.session_id}'. You MUST use this exact ID for the 'session_id' parameter in all tool calls. Do NOT ask for it.
        
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

    try:
        resp = requests.post(
            AI_API_URL,
            headers={"Authorization": f"Bearer {AI_API_KEY}", "Content-Type": "application/json"},
            json=payload,
            timeout=10
        )
        resp.raise_for_status()
        data = resp.json()
        print(f"[MCP DEBUG] LLM Response: {json.dumps(data, ensure_ascii=False)}")
    except Exception as e:
        print(f"[MCP DEBUG] LLM Call Error: {str(e)}")
        return {"status": "error", "message": f"LLM Call Failed: {str(e)}"}

    # 2. Process Tool Calls
    results = []
    choice = data.get("choices", [{}])[0]
    message = choice.get("message", {})
    tool_calls = message.get("tool_calls", [])

    if not tool_calls:
        print("[MCP DEBUG] No tool calls in response.")
        return {"status": "ignored", "message": "The wish was heard, but nothing happened."}

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

    return {"status": "success", "results": results}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=9091)
