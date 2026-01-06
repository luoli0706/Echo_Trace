import sys
import queue
import json
from pathlib import Path
import os
import pygame
from client.network import NetworkClient
from client.gamestate import GameState
from client.renderer import Renderer
from client.config import WINDOW_WIDTH, WINDOW_HEIGHT
import time

# Default Server (optional, can be provided via env)
DEFAULT_SERVER_URL = os.environ.get("ECHO_TRACE_SERVER_URL", "").strip()

CLIENT_STATE_PATH = Path.home() / ".echo_trace_client.json"


def _load_client_state():
    try:
        with open(CLIENT_STATE_PATH, "r", encoding="utf-8") as f:
            data = json.load(f)
            return data if isinstance(data, dict) else {}
    except Exception:
        return {}


def _save_client_state(data: dict):
    try:
        with open(CLIENT_STATE_PATH, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
    except Exception:
        pass

def main():
    pygame.init()
    # Enable IME-friendly text input (TEXTINPUT) so Chinese/Japanese input works.
    pygame.key.start_text_input()

    # Enable clipboard for Ctrl+C/Ctrl+V in text inputs.
    try:
        pygame.scrap.init()
        try:
            pygame.scrap.set_mode(pygame.SCRAP_CLIPBOARD)
        except Exception:
            pass
    except Exception:
        pass

    screen = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
    pygame.display.set_caption("Echo Trace Client [Alpha 0.5]")
    clock = pygame.time.Clock()

    recv_q = queue.Queue()
    net = None
    connect_in_progress = False
    connect_started_at = 0.0

    persisted = _load_client_state()
    persisted_session_id = str(persisted.get("session_id") or "")
    persisted_name = str(persisted.get("name") or "")
    persisted_last_room_id = str(persisted.get("last_room_id") or "")

    state = GameState()
    renderer = Renderer(screen)
    persisted_server_url = str(persisted.get("server_url") or "").strip()
    if persisted_server_url:
        renderer.server_input = persisted_server_url
    elif DEFAULT_SERVER_URL:
        renderer.server_input = DEFAULT_SERVER_URL
    renderer.server_cursor = len(renderer.server_input)
    if persisted_name:
        renderer.name_input = persisted_name
    # Cold-start resume is gated: user must provide Resume ID (session_id) on CONNECT.
    renderer.resume_id_input = ""
    renderer.resume_cursor = 0
    input_dir = [0, 0]

    def _append_text(dst: str, text: str, max_len: int = 120) -> str:
        if not text:
            return dst
        # Filter surrogate code points (can appear during IME composition on some systems).
        filtered = "".join(ch for ch in str(text) if not ("\ud800" <= ch <= "\udfff"))
        if not filtered:
            return dst
        out = dst + filtered
        if len(out) > max_len:
            out = out[:max_len]
        return out

    def _insert_at(dst: str, cursor: int, text: str, max_len: int) -> tuple[str, int]:
        cursor = max(0, min(int(cursor), len(dst)))
        out = dst[:cursor] + text + dst[cursor:]
        if len(out) > max_len:
            out = out[:max_len]
        cursor = min(cursor + len(text), len(out))
        return out, cursor

    def _delete_left(dst: str, cursor: int) -> tuple[str, int]:
        cursor = max(0, min(int(cursor), len(dst)))
        if cursor <= 0:
            return dst, 0
        return dst[:cursor - 1] + dst[cursor:], cursor - 1

    def _delete_right(dst: str, cursor: int) -> tuple[str, int]:
        cursor = max(0, min(int(cursor), len(dst)))
        if cursor >= len(dst):
            return dst, cursor
        return dst[:cursor] + dst[cursor + 1 :], cursor

    def _cursor_from_click(text: str, font, rect_x: int, click_x: int, padding_x: int = 10) -> int:
        # Convert mouse x to caret index by measuring substring width.
        x = max(0, int(click_x) - int(rect_x) - int(padding_x))
        if x <= 0:
            return 0
        # Fast path: end.
        if font.size(text)[0] <= x:
            return len(text)
        # Linear scan (strings are short).
        for i in range(1, len(text) + 1):
            if font.size(text[:i])[0] >= x:
                return i
        return len(text)

    def _clip_get_text() -> str:
        try:
            if pygame.scrap.get_init():
                raw = pygame.scrap.get(pygame.SCRAP_TEXT)
                if raw is None:
                    return ""
                if isinstance(raw, (bytes, bytearray)):
                    try:
                        return raw.decode("utf-8", errors="ignore")
                    except Exception:
                        return raw.decode(errors="ignore")
                return str(raw)
        except Exception:
            return ""
        return ""

    def _clip_set_text(text: str) -> None:
        try:
            if pygame.scrap.get_init():
                pygame.scrap.put(pygame.SCRAP_TEXT, str(text).encode("utf-8"))
        except Exception:
            pass

    running = True
    while running:
        dt = 1.0 / 60.0
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            
            # --- State: CONNECT ---
            if renderer.state == "CONNECT":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    pos = event.pos
                    if getattr(renderer, "connect_server_rect", None) and renderer.connect_server_rect.collidepoint(pos):
                        renderer.connect_focus = "server"
                        renderer.server_cursor = _cursor_from_click(renderer.server_input, renderer.font, renderer.connect_server_rect.x, pos[0])
                        continue
                    if getattr(renderer, "connect_resume_rect", None) and renderer.connect_resume_rect.collidepoint(pos):
                        renderer.connect_focus = "resume_id"
                        renderer.resume_cursor = _cursor_from_click(renderer.resume_id_input, renderer.font, renderer.connect_resume_rect.x, pos[0])
                        continue

                if event.type == pygame.KEYDOWN:
                    # Ctrl+C / Ctrl+V for copy/paste
                    if event.mod & pygame.KMOD_CTRL:
                        if event.key == pygame.K_c:
                            if renderer.connect_focus == "server":
                                _clip_set_text(renderer.server_input)
                            else:
                                _clip_set_text(renderer.resume_id_input)
                            continue
                        if event.key == pygame.K_v:
                            clip = _clip_get_text().replace("\r", "").replace("\n", "").strip()
                            if renderer.connect_focus == "server":
                                renderer.server_input, renderer.server_cursor = _insert_at(
                                    renderer.server_input, getattr(renderer, "server_cursor", len(renderer.server_input)), clip, max_len=200
                                )
                            else:
                                renderer.resume_id_input, renderer.resume_cursor = _insert_at(
                                    renderer.resume_id_input, getattr(renderer, "resume_cursor", len(renderer.resume_id_input)), clip, max_len=80
                                )
                            continue

                    if event.key == pygame.K_TAB:
                        renderer.connect_focus = "resume_id" if renderer.connect_focus == "server" else "server"
                    elif event.key == pygame.K_RETURN:
                        url = renderer.server_input.strip() or DEFAULT_SERVER_URL
                        if not url:
                            renderer.menu_message = "请填写服务器地址，例如 ws://139.224.0.73:9191/ws"
                            continue
                        resume_id = renderer.resume_id_input.strip()
                        try:
                            if connect_in_progress:
                                continue
                            connect_in_progress = True
                            connect_started_at = time.monotonic()
                            renderer.menu_message = f"正在连接: {url}"

                            # Stop any previous client first.
                            if net:
                                try:
                                    net.stop()
                                except Exception:
                                    pass

                            # Only resume when Resume ID is explicitly provided.
                            net = NetworkClient(url, recv_q, session_id=(resume_id or ""), player_name=persisted_name)
                            if resume_id and persisted_last_room_id:
                                net.set_auto_join(persisted_last_room_id)
                            net.start()
                            persisted["server_url"] = url
                            _save_client_state(persisted)
                        except Exception as e:
                            print(f"Connection failed: {e}")
                            renderer.menu_message = f"连接失败: {e}"
                            connect_in_progress = False
                    elif event.key == pygame.K_LEFT:
                        if renderer.connect_focus == "server":
                            renderer.server_cursor = max(0, int(getattr(renderer, "server_cursor", 0)) - 1)
                        else:
                            renderer.resume_cursor = max(0, int(getattr(renderer, "resume_cursor", 0)) - 1)
                    elif event.key == pygame.K_RIGHT:
                        if renderer.connect_focus == "server":
                            renderer.server_cursor = min(len(renderer.server_input), int(getattr(renderer, "server_cursor", 0)) + 1)
                        else:
                            renderer.resume_cursor = min(len(renderer.resume_id_input), int(getattr(renderer, "resume_cursor", 0)) + 1)
                    elif event.key == pygame.K_HOME:
                        if renderer.connect_focus == "server":
                            renderer.server_cursor = 0
                        else:
                            renderer.resume_cursor = 0
                    elif event.key == pygame.K_END:
                        if renderer.connect_focus == "server":
                            renderer.server_cursor = len(renderer.server_input)
                        else:
                            renderer.resume_cursor = len(renderer.resume_id_input)
                    elif event.key == pygame.K_BACKSPACE:
                        if renderer.connect_focus == "server":
                            renderer.server_input, renderer.server_cursor = _delete_left(
                                renderer.server_input, getattr(renderer, "server_cursor", len(renderer.server_input))
                            )
                        else:
                            renderer.resume_id_input, renderer.resume_cursor = _delete_left(
                                renderer.resume_id_input, getattr(renderer, "resume_cursor", len(renderer.resume_id_input))
                            )
                    elif event.key == pygame.K_DELETE:
                        if renderer.connect_focus == "server":
                            renderer.server_input, renderer.server_cursor = _delete_right(
                                renderer.server_input, getattr(renderer, "server_cursor", len(renderer.server_input))
                            )
                        else:
                            renderer.resume_id_input, renderer.resume_cursor = _delete_right(
                                renderer.resume_id_input, getattr(renderer, "resume_cursor", len(renderer.resume_id_input))
                            )
                elif event.type == pygame.TEXTINPUT:
                    if renderer.connect_focus == "server":
                        renderer.server_input, renderer.server_cursor = _insert_at(
                            renderer.server_input,
                            getattr(renderer, "server_cursor", len(renderer.server_input)),
                            _append_text("", event.text, max_len=200),
                            max_len=200,
                        )
                    else:
                        renderer.resume_id_input, renderer.resume_cursor = _insert_at(
                            renderer.resume_id_input,
                            getattr(renderer, "resume_cursor", len(renderer.resume_id_input)),
                            _append_text("", event.text, max_len=80),
                            max_len=80,
                        )
                continue

            # --- State: LOGIN ---
            if renderer.state == "LOGIN":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_RETURN:
                        name = renderer.name_input.strip() or "Agent_47"
                        persisted_name = name
                        persisted["name"] = persisted_name
                        _save_client_state(persisted)
                        if net:
                            net.set_identity(player_name=persisted_name)
                        renderer.state = "MENU"
                    elif event.key == pygame.K_BACKSPACE:
                        renderer.name_input = renderer.name_input[:-1]
                elif event.type == pygame.TEXTINPUT:
                    renderer.name_input = _append_text(renderer.name_input, event.text, max_len=40)
                continue

            # --- State: MENU ---
            if renderer.state == "MENU":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    if renderer.menu_rects.get("create").collidepoint(event.pos):
                        if not net or not getattr(net, "connected", False):
                            renderer.menu_message = "未连接服务器：请先在 CONNECT 页面连接成功。"
                            continue
                        renderer.enter_config_editor()
                        renderer.state = "CONFIG"
                    elif renderer.menu_rects.get("join").collidepoint(event.pos):
                        renderer.menu_message = ""
                        if not net or not getattr(net, "connected", False):
                            renderer.menu_message = "未连接服务器：请先在 CONNECT 页面连接成功。"
                        else:
                            net.send({"type": 1013, "payload": {}})
                            renderer.state = "ROOM_LIST"
                continue

            # --- State: ROOM_LIST ---
            if renderer.state == "ROOM_LIST":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_ESCAPE:
                        renderer.state = "MENU"
                    elif event.key == pygame.K_r:
                        if net and getattr(net, "connected", False):
                            net.send({"type": 1013, "payload": {}})
                        else:
                            renderer.menu_message = "未连接服务器：无法刷新房间列表。"
                    elif event.key == pygame.K_UP:
                        renderer.room_list_selected = max(0, renderer.room_list_selected - 1)
                        if renderer.room_list_selected < renderer.room_list_scroll:
                            renderer.room_list_scroll = max(0, renderer.room_list_selected)
                    elif event.key == pygame.K_DOWN:
                        renderer.room_list_selected = min(max(0, len(renderer.rooms) - 1), renderer.room_list_selected + 1)
                        visible = 10
                        if renderer.room_list_selected >= renderer.room_list_scroll + visible:
                            renderer.room_list_scroll = max(0, renderer.room_list_selected - visible + 1)
                    elif event.key == pygame.K_RETURN:
                        if renderer.rooms and renderer.room_list_selected < len(renderer.rooms):
                            rid = renderer.rooms[renderer.room_list_selected].get("room_id")
                            if rid and net:
                                payload = {"room_id": rid}
                                if persisted_name:
                                    payload["name"] = persisted_name
                                net.send({"type": 1011, "payload": payload})
                elif event.type == pygame.MOUSEBUTTONDOWN:
                    if renderer.room_list_refresh_rect and renderer.room_list_refresh_rect.collidepoint(event.pos):
                        if net and getattr(net, "connected", False):
                            net.send({"type": 1013, "payload": {}})
                        else:
                            renderer.menu_message = "未连接服务器：无法刷新房间列表。"
                        continue
                    if renderer.room_list_back_rect and renderer.room_list_back_rect.collidepoint(event.pos):
                        renderer.state = "MENU"
                        continue
                    for idx, rect in getattr(renderer, "room_list_row_rects", []):
                        if rect.collidepoint(event.pos):
                            renderer.room_list_selected = idx
                            # double click / click-to-join convenience
                            if net and idx < len(renderer.rooms):
                                rid = renderer.rooms[idx].get("room_id")
                                if rid:
                                    payload = {"room_id": rid}
                                    if persisted_name:
                                        payload["name"] = persisted_name
                                    net.send({"type": 1011, "payload": payload})
                            break
                    # Mouse wheel (older pygame)
                    if event.button in (4, 5):
                        renderer.room_list_scroll = max(0, renderer.room_list_scroll + (-1 if event.button == 4 else 1))
                elif event.type == pygame.MOUSEWHEEL:
                    renderer.room_list_scroll = max(0, renderer.room_list_scroll - event.y)
                continue

            # --- State: CONFIG ---
            if renderer.state == "CONFIG":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    if renderer.config_create_rect and renderer.config_create_rect.collidepoint(event.pos):
                        rn = renderer.room_name_input.strip()
                        if not rn:
                            renderer.menu_message = "必须填写房间名。"
                        else:
                            renderer.menu_message = ""
                            if not net or not getattr(net, "connected", False):
                                renderer.menu_message = "未连接服务器：无法创建房间。"
                            else:
                                payload = {"room_name": rn, "config": renderer.config_data}
                                if persisted_name:
                                    payload["name"] = persisted_name
                                net.send({"type": 1010, "payload": payload})
                        continue
                    if renderer.config_back_rect and renderer.config_back_rect.collidepoint(event.pos):
                        if getattr(renderer, "config_view", "index") == "edit":
                            renderer._set_config_index_rows()
                            renderer.config_focus = "table"
                        else:
                            renderer.state = "MENU"
                        continue

                    # Select row
                    for idx, rect in getattr(renderer, "config_row_rects", []):
                        if rect.collidepoint(event.pos):
                            renderer.config_focus = "table"
                            renderer.config_selected = idx
                            renderer.config_editing = False
                            renderer.config_edit_buffer = ""

                            # Index view: click-to-open category
                            if getattr(renderer, "config_view", "index") == "index":
                                if idx < len(getattr(renderer, "config_rows", []) or []):
                                    row = renderer.config_rows[idx]
                                    if isinstance(row, dict) and row.get("is_section"):
                                        renderer.open_config_section(row.get("section_key"))
                            break

                    # Mouse wheel (older pygame)
                    if event.button in (4, 5):
                        renderer.config_scroll = max(0, renderer.config_scroll + (-1 if event.button == 4 else 1))
                    continue

                if event.type == pygame.MOUSEWHEEL:
                    renderer.config_scroll = max(0, renderer.config_scroll - event.y)
                    continue

                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_ESCAPE:
                        if getattr(renderer, "config_view", "index") == "edit":
                            renderer._set_config_index_rows()
                            renderer.config_focus = "table"
                        else:
                            renderer.state = "MENU"
                        continue

                    if event.key == pygame.K_TAB:
                        renderer.config_focus = "table" if renderer.config_focus == "room_name" else "room_name"
                        renderer.config_editing = False
                        renderer.config_edit_buffer = ""
                        continue

                    # Focus: room name
                    if renderer.config_focus == "room_name":
                        if event.key == pygame.K_RETURN:
                            renderer.config_focus = "table"
                        elif event.key == pygame.K_BACKSPACE:
                            renderer.room_name_input = renderer.room_name_input[:-1]
                        continue

                    # Focus: table
                    if renderer.config_focus == "table":
                        if event.key == pygame.K_UP:
                            renderer.config_selected = max(0, renderer.config_selected - 1)
                            if renderer.config_selected < renderer.config_scroll:
                                renderer.config_scroll = max(0, renderer.config_selected)
                            renderer.config_editing = False
                        elif event.key == pygame.K_DOWN:
                            renderer.config_selected = min(max(0, len(renderer.config_rows) - 1), renderer.config_selected + 1)
                            visible = 16
                            if renderer.config_selected >= renderer.config_scroll + visible:
                                renderer.config_scroll = max(0, renderer.config_selected - visible + 1)
                            renderer.config_editing = False
                        elif event.key == pygame.K_SPACE:
                            # Toggle bool
                            if getattr(renderer, "config_view", "index") == "edit" and renderer.config_rows and renderer.config_selected < len(renderer.config_rows):
                                row = renderer.config_rows[renderer.config_selected]
                                if row.get("editable", True) and isinstance(row.get("value"), bool):
                                    v = not bool(row.get("value"))
                                    row["value"] = v
                                    try:
                                        renderer._set_by_path(renderer.config_data, row["path"], v)
                                    except Exception:
                                        pass
                        elif event.key == pygame.K_RETURN:
                            mods = pygame.key.get_mods()
                            if (mods & pygame.KMOD_CTRL) or (mods & pygame.KMOD_LCTRL):
                                # Create room
                                rn = renderer.room_name_input.strip()
                                if not rn:
                                    renderer.menu_message = "必须填写房间名。"
                                else:
                                    renderer.menu_message = ""
                                    if not net or not getattr(net, "connected", False):
                                        renderer.menu_message = "未连接服务器：无法创建房间。"
                                    else:
                                        payload = {"room_name": rn, "config": renderer.config_data}
                                        if persisted_name:
                                            payload["name"] = persisted_name
                                        net.send({"type": 1010, "payload": payload})
                            else:
                                # Index view: Enter opens selected category
                                if getattr(renderer, "config_view", "index") == "index":
                                    if renderer.config_rows and renderer.config_selected < len(renderer.config_rows):
                                        row = renderer.config_rows[renderer.config_selected]
                                        if isinstance(row, dict) and row.get("is_section"):
                                            renderer.open_config_section(row.get("section_key"))
                                    continue

                                # Edit view: Start/commit row editing
                                if not renderer.config_rows or renderer.config_selected >= len(renderer.config_rows):
                                    continue
                                row = renderer.config_rows[renderer.config_selected]
                                if not row.get("editable", True):
                                    continue
                                if renderer.config_editing:
                                    # Commit
                                    s = renderer.config_edit_buffer
                                    old = row.get("value")
                                    try:
                                        if isinstance(old, bool):
                                            v = s.strip().lower() in ("1", "true", "yes", "y", "on")
                                        elif isinstance(old, int):
                                            v = int(float(s.strip() or "0"))
                                        elif isinstance(old, float):
                                            v = float(s.strip() or "0")
                                        else:
                                            v = s
                                        v2, clamp_msg = renderer.clamp_config_value(row["path"], v, old)
                                        row["value"] = v2
                                        renderer._set_by_path(renderer.config_data, row["path"], v2)
                                        renderer.menu_message = clamp_msg or ""
                                    except Exception:
                                        renderer.menu_message = "值解析失败：请检查类型。"
                                    renderer.config_editing = False
                                    renderer.config_edit_buffer = ""
                                else:
                                    renderer.config_editing = True
                                    renderer.config_edit_buffer = str(row.get("value", ""))
                        elif event.key == pygame.K_BACKSPACE:
                            if renderer.config_editing:
                                renderer.config_edit_buffer = renderer.config_edit_buffer[:-1]
                        else:
                            if renderer.config_editing:
                                pass
                elif event.type == pygame.TEXTINPUT:
                    # IME-friendly text entry
                    if renderer.config_focus == "room_name":
                        renderer.room_name_input = _append_text(renderer.room_name_input, event.text, max_len=40)
                    elif renderer.config_focus == "table" and renderer.config_editing:
                        renderer.config_edit_buffer = _append_text(renderer.config_edit_buffer, event.text, max_len=60)
                continue

            # --- State: GAME ---
            if renderer.state == "GAME":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    if event.button == 1:
                        # Lobby Back Button
                        if state.phase == 0 and hasattr(renderer, 'lobby_back_rect') and renderer.lobby_back_rect.collidepoint(event.pos):
                            renderer.state = "MENU"
                            state = GameState() # Reset
                            if net:
                                net.clear_auto_join()
                            persisted_last_room_id = ""
                            persisted.pop("last_room_id", None)
                            _save_client_state(persisted)
                            continue
                            
                        renderer.handle_click(event.pos)

                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_ESCAPE:
                        renderer.state = "PAUSE"
                        renderer.pause_open()
                        continue
                    
                    if event.key == pygame.K_F9 and renderer.dev_mode:
                        if net: net.send({"type": 9001, "payload": {}})

                    # Gameplay Inputs
                    if not renderer.show_shop:
                        if event.key == pygame.K_w: input_dir[1] = -1
                        elif event.key == pygame.K_s: input_dir[1] = 1
                        elif event.key == pygame.K_a: input_dir[0] = -1
                        elif event.key == pygame.K_d: input_dir[0] = 1
                        elif event.key == pygame.K_e:
                            if net: net.send({"type": 2004, "payload": {}}) # Pickup
                        elif event.key == pygame.K_f:
                            # Merchant Check
                            near_merchant = False
                            merchant_range = 2.0
                            try:
                                cfg = getattr(state, "config", None) or {}
                                merchant_range = float(cfg.get("gameplay", {}).get("merchant_interact_range", merchant_range))
                            except Exception:
                                pass
                            for ent in state.entities:
                                if ent["type"] == "MERCHANT":
                                    d = ((state.my_pos[0]-ent["pos"]["x"])**2 + (state.my_pos[1]-ent["pos"]["y"])**2)**0.5
                                    if d <= merchant_range: near_merchant = True; break
                            if near_merchant: renderer.show_shop = True
                            elif net: net.send({"type": 2003, "payload": {}}) # Interact
                        elif event.key == pygame.K_SPACE:
                            if net: net.send({"type": 2010, "payload": {}}) # Fire
                        
                        # Number Keys
                        elif event.key >= pygame.K_1 and event.key <= pygame.K_6:
                            slot = event.key - pygame.K_1
                            mods = pygame.key.get_mods()
                            if state.phase == 0 and not state.tactic_chosen:
                                if event.key <= pygame.K_3:
                                    t = {pygame.K_1: "RECON", pygame.K_2: "DEFENSE", pygame.K_3: "TRAP"}.get(event.key)
                                    if net: net.send({"type": 2006, "payload": {"tactic": t}})
                                    state.tactic_chosen = True
                            else:
                                if mods & pygame.KMOD_SHIFT:
                                    if net: net.send({"type": 2005, "payload": {"slot_index": slot}})
                                elif mods & pygame.KMOD_CTRL or mods & pygame.KMOD_LCTRL:
                                    if net: net.send({"type": 2008, "payload": {"slot_index": slot}})
                                else:
                                    if net: net.send({"type": 2002, "payload": {"slot_index": slot}})
                    else:
                        # Shop is open
                        if event.key == pygame.K_f or event.key == pygame.K_ESCAPE: renderer.show_shop = False
                        elif event.key == pygame.K_r:
                            if net: net.send({"type": 2009, "payload": {}})
                        elif event.key >= pygame.K_1 and event.key <= pygame.K_6:
                            idx = event.key - pygame.K_1
                            mods = pygame.key.get_mods()
                            if mods & pygame.KMOD_CTRL or mods & pygame.KMOD_LCTRL:
                                if net: net.send({"type": 2008, "payload": {"slot_index": idx}})
                            else:
                                stock = getattr(state, "shop_stock", []) or []
                                if idx < len(stock):
                                    net.send({"type": 2007, "payload": {"item_id": stock[idx]}})

                if event.type == pygame.KEYUP:
                    if event.key in (pygame.K_w, pygame.K_s): input_dir[1] = 0
                    if event.key in (pygame.K_a, pygame.K_d): input_dir[0] = 0

            # --- State: PAUSE ---
            if renderer.state == "PAUSE":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_ESCAPE:
                        if renderer.pause_view() != "root":
                            renderer.pause_pop()
                        else:
                            renderer.state = "GAME"

                elif event.type == pygame.MOUSEBUTTONDOWN:
                    # Mouse wheel (older pygame) for manual scrolling
                    if renderer.pause_view() == "item_manual" and event.button in (4, 5):
                        renderer.scroll_item_manual(-40 if event.button == 4 else 40)
                        continue

                    if renderer.pause_view() != "root":
                        renderer.handle_click(event.pos)
                    else:
                        action = renderer.handle_pause_click(event.pos)
                        if action == "resume":
                            renderer.state = "GAME"
                        elif action == "settings":
                            renderer.pause_push("settings")
                        elif action == "help":
                            renderer.pause_push("help")
                        elif action == "item_manual":
                            renderer.pause_push("item_manual")
                        elif action == "quit":
                            renderer.state = "MENU"
                            state = GameState()  # Reset local
                            renderer.pause_route = []
                            if net:
                                net.clear_auto_join()
                            persisted_last_room_id = ""
                            persisted.pop("last_room_id", None)
                            _save_client_state(persisted)

                elif event.type == pygame.MOUSEWHEEL:
                    if renderer.pause_view() == "item_manual":
                        renderer.scroll_item_manual(-event.y * 40)

                continue

        # CONNECT progress handling must run outside the per-event loop.
        # Otherwise it becomes unreachable because CONNECT consumes events with `continue`.
        if connect_in_progress and net is not None:
            if getattr(net, "connected", False):
                connect_in_progress = False
                renderer.menu_message = ""
                renderer.state = "LOGIN"
            else:
                last_err = str(getattr(net, "last_error", "") or "").strip()
                if last_err:
                    renderer.menu_message = f"连接失败: {last_err}".strip()
                    # Stop auto-retry loop from hiding the failure behind repeated reconnects.
                    connect_in_progress = False
                else:
                    # Fail fast if connect takes too long (common when port is blocked).
                    try:
                        timeout_sec = float(getattr(net, "connect_timeout_sec", 0.0) or 0.0)
                    except Exception:
                        timeout_sec = 0.0
                    if timeout_sec > 0 and connect_started_at > 0:
                        if (time.monotonic() - connect_started_at) > (timeout_sec + 0.5):
                            renderer.menu_message = f"连接超时（>{timeout_sec:.1f}s）：请检查端口/防火墙/地址是否正确".strip()
                            try:
                                net.stop()
                            except Exception:
                                pass
                            connect_in_progress = False

        # Network
        if net:
            while not recv_q.empty():
                msg = recv_q.get()
                mt, pl = msg.get("type"), msg.get("payload")
                if mt == 1001 and isinstance(pl, dict):
                    sid = str(pl.get("session_id") or "")
                    if sid:
                        persisted_session_id = sid
                        persisted["session_id"] = sid
                        _save_client_state(persisted)
                        net.set_identity(session_id=sid)
                if mt == 1012:
                    renderer.state = "GAME"
                    state.config = pl.get("config")
                    renderer.menu_message = ""

                    # Remember the room_id for reconnect auto-join.
                    rid = str(pl.get("room_id") or "")
                    if rid:
                        persisted_last_room_id = rid
                        persisted["last_room_id"] = rid
                        _save_client_state(persisted)
                        net.set_auto_join(rid)

                    # Ensure server sees our display name after joining.
                    if persisted_name:
                        net.send({"type": 1001, "payload": {"name": persisted_name}})
                elif mt == 3001:
                    state.map_tiles = pl["map_tiles"]
                    state.my_pos = [pl["spawn_pos"]["x"], pl["spawn_pos"]["y"]]
                elif mt == 3002:
                    state.update_from_server(pl)
                elif mt == 1014:
                    renderer.rooms = pl.get("rooms", []) or []
                    renderer.room_list_selected = 0
                    renderer.room_list_scroll = 0
                elif mt == 4001:
                    renderer.menu_message = (pl.get("msg") if isinstance(pl, dict) else str(pl))

        # Logic
        if renderer.state == "GAME" and net:
            renderer.update_look_from_mouse(pygame.mouse.get_pos(), dt, state)
            if getattr(state, "is_extracted", False) and renderer.spectator_mode:
                # Free Spectate Camera Movement
                speed = 10.0 * (1.0/60.0) # approx dt
                renderer.cam_offset[0] += input_dir[0] * speed
                renderer.cam_offset[1] += input_dir[1] * speed
            elif state.phase > 0 and not renderer.show_shop:
                lx, ly = renderer.get_look_dir()
                net.send({"type": 2001, "payload": {"dir": {"x": float(input_dir[0]), "y": float(input_dir[1])}, "look_dir": {"x": float(lx), "y": float(ly)}}})

        renderer.draw_game(state)
        pygame.display.flip()
        clock.tick(60)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()
