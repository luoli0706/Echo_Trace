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
- 超级穿甲弹 (AP Ammo) [T1] -> WPN_AP_AMMO
- 光学折射效应 (Reflect/Bounce Ammo) [T2] -> WPN_BOUNCE_AMMO
- 超视距·电磁炮 (Railgun) [T3] -> WPN_RAILGUN
- 超视距·精通天理 (Heaven Ray) [T4] -> WPN_HEAVEN_RAY

[生存类 / Survival]
- 回光返照 (Repair Kit) [T1] -> SURV_REPAIR
- 绝对防御 (Phase Shift/Invincible) [T2] -> SURV_PHASE_SHIFT
- 主动防御 (Purge System) [T3] -> SURV_PURGE
- 我就是柠白号 (NingBye Mode) [T4] -> SURV_NINGBYE_MODE

[侦察类 / Recon]
- 视距提升 (Scope) [T1] -> RECON_SCOPE
- 光锥之外 (Rear Sensor) [T2] -> RECON_SENSOR
- 全境扫描终端 (Global Scan) [T3] -> RECON_SCANNER
- 全频段阻塞干扰 (Global Jammer) [T4] -> RECON_JAMMER

[综合类 / Utility]
- 闪灵瞬步 (Blink) [T1] -> UTIL_BLINK
- 超视距·追踪 (Radar) [T2] -> UTIL_RADAR
- 残缺的万能许愿机 (Broken Wish Machine) [T2] -> UTIL_WISH_MACHINE
- 无形之物 (Stealth) [T3] -> UTIL_STEALTH
- 万能许愿机 (Universal Wish Machine) [T4] -> UTIL_WISH_MACHINE_FULL (Not implemented separately, logic handled by CLI)
- 开发者遗忘的命令行 (Developer Forgotten CLI) [T4] -> UTIL_DEV_FORGOTTEN_CLI
"""

MAP_CONTEXT = """
Game Map Information:
- Size: 48x48 grid.
- Coordinates: (0,0) is Top-Left, (47,47) is Bottom-Right.
- Center is approximately (24,24).
- Obstacles: Procedurally generated maze (density ~0.2).
- Key Locations:
  - Motors (Phase 2) are scattered.
  - Exits (Phase 3) usually appear at edges.
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

def modify_global_health(hp: float) -> str:
    """Set the health of ALL players in the game to a specific value."""
    return _call_backend("/admin/player/health/global", {"hp": hp})

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
    # Correctly remove markdown links [Text](URL) -> Text
    text = re.sub(r"\[([^\]]+)\]\([^\)]+\)", r"\1", text)
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
            "name": "modify_global_health",
            "description": "[T4 Exclusive] Set the HP of ALL players in the game.",
            "parameters": {
                "type": "object",
                "properties": {
                    "hp": {"type": "number", "description": "Target HP value (0-500)"}
                },
                "required": ["hp"]
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
            "description": "[T4 Exclusive] Move a player to a target coordinate. Use when user asks to relocate someone to X,Y.",
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
    "modify_global_health": modify_global_health,
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

    # Determine Device Tier and Permissions
    is_t4 = (item_id == "UTIL_DEV_FORGOTTEN_CLI")
    device_name = "Developer CLI (Tier 4)" if is_t4 else "Residual Wish Machine (Tier 2)"
    
    # Filter Tools based on Tier
    allowed_tools = []
    for t in TOOLS_SCHEMA:
        name = t["function"]["name"]
        
        # T4 Exclusive Tools
        if name in ("move_player_to_coordinate", "modify_global_health", "command_ai_patrol", "set_player_threat"):
            if not is_t4:
                continue # Skip for T2
        
        allowed_tools.append(t)

    # Context Prompt
    permission_context = ""
    if is_t4:
        permission_context = """
        TIER 4 ACCESS GRANTED:
        - You have FULL ACCESS to all tools.
        - You can move players, command AI, set global stats, and spawn ANY item (Tier 1-4).
        """
    else:
        permission_context = """
        TIER 2 ACCESS (RESTRICTED):
        - You can ONLY modify self stats (HP, Armor, Speed) and spawn items up to Tier 3.
        - You CANNOT move players, command AI, or spawn Tier 4 items.
        - If the user asks for restricted actions, politely deny them citing 'Insufficient Permissions (Tier 2 Device)'.
        """

    # 1. Call LLM
    messages = [
        {"role": "system", "content": f"""You are a Game Master for Echo Trace. Interpret the user's wish and call the appropriate tools to fulfill it.

CRITICAL RULES:
- Device Used: {device_name}.
- Wishing Player Session ID: '{req.session_id}'.
- Do NOT ask confirmation. Act or Deny.
- If impossible/denied, reply with short plain-text reason.
- **IMPORTANT**: At the very end of your text response, provide a very concise summary (max 10 words) of what you actually did, prefixed with '[[SUMMARY]]'.
  - Example: "Teleporting you. [[SUMMARY]]Teleported to (24,24)"
  - Example: "Granting items. [[SUMMARY]]Added Railgun & Ammo"

{permission_context}

{MAP_CONTEXT}

ITEM LIST (Use 'add_items_to_inventory' with IDs):
{ITEM_MAPPING}

If valid, execute. If ambiguous, guess."""},
        {"role": "user", "content": f"I wish for: {req.wish}"}
    ]

    payload = {
        "model": "deepseek-chat", 
        "messages": messages,
        "tools": allowed_tools
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
        return {"status": "needs_retry", "results": [], "玩家可见回应": sanitize_plain_text(message.get("content", "指令无法执行（权限不足或意图不明）。"))}

    for tc in tool_calls:
        func_name = tc["function"]["name"]
        args_str = tc["function"]["arguments"]
        print(f"[MCP DEBUG] Tool Call: {func_name} Args: {args_str}")
        try:
            args = json.loads(args_str)
            if func_name in TOOL_MAP:
                if "session_id" not in args and "session_id" in tc["function"]["arguments"]: 
                     pass
                
                if not is_t4 and func_name in ("move_player_to_coordinate", "modify_global_health"):
                     results.append(f"{func_name}: DENIED (Tier 2 Restriction)")
                     continue

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
    if ok_count == 0 and len(results) > 0:
        reply = "指令执行失败或被拒绝。"
    
    # Process LLM Text for Summary
    llm_text = message.get("content", "")
    action_summary = ""
    
    if llm_text:
        # Check for [[SUMMARY]] token
        match = re.search(r"\[\[SUMMARY\]\](.*)", llm_text, re.DOTALL)
        if match:
            action_summary = match.group(1).strip()
            # Remove summary from visible reply
            llm_text = llm_text.replace(match.group(0), "").strip()
        
        reply = f"{sanitize_plain_text(llm_text)}"

    # Fallback if LLM didn't provide summary, use heuristic
    if not action_summary:
        summary_parts = []
        for tc in tool_calls:
            fn = tc["function"]["name"]
            if fn == "add_items_to_inventory": summary_parts.append("物资配发")
            elif fn == "modify_player_health": summary_parts.append("生命调整")
            elif fn == "move_player_to_coordinate": summary_parts.append("传送")
            elif fn == "command_ai_patrol": summary_parts.append("AI部署")
            elif fn == "set_player_threat": summary_parts.append("威胁更新")
            elif fn == "modify_global_health": summary_parts.append("世界重塑")
        if summary_parts:
            action_summary = " & ".join(summary_parts)
        else:
            action_summary = "System Action"

    # Prefix with Tier info
    tier_prefix = "[T4]" if is_t4 else "[T2]"
    action_summary = f"{tier_prefix} {action_summary}"

    return {"status": "success", "results": results, "玩家可见回应": reply, "action_summary": action_summary}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=9091)
