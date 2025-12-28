import sys
import queue
import pygame
from client.network import NetworkClient
from client.gamestate import GameState
from client.renderer import Renderer
from client.config import WINDOW_WIDTH, WINDOW_HEIGHT

# Default Server
DEFAULT_SERVER_URL = "ws://localhost:8080/ws"

def main():
    pygame.init()
    screen = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
    pygame.display.set_caption("Echo Trace Client [Alpha 0.5]")
    clock = pygame.time.Clock()

    recv_q = queue.Queue()
    net = None

    state = GameState()
    renderer = Renderer(screen)
    input_dir = [0, 0]

    running = True
    while running:
        dt = 1.0 / 60.0
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            
            # --- State: CONNECT ---
            if renderer.state == "CONNECT":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_RETURN:
                        url = renderer.server_input.strip() or DEFAULT_SERVER_URL
                        print(f"Connecting to {url}...")
                        try:
                            net = NetworkClient(url, recv_q)
                            net.start()
                            renderer.state = "LOGIN"
                        except Exception as e:
                            print(f"Connection failed: {e}")
                    elif event.key == pygame.K_BACKSPACE:
                        renderer.server_input = renderer.server_input[:-1]
                    else:
                        renderer.server_input += event.unicode
                continue

            # --- State: LOGIN ---
            if renderer.state == "LOGIN":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_RETURN:
                        name = renderer.name_input.strip() or "Agent_47"
                        if net: net.send({"type": 1001, "payload": {"name": name}})
                        renderer.state = "MENU"
                    elif event.key == pygame.K_BACKSPACE:
                        renderer.name_input = renderer.name_input[:-1]
                    else:
                        renderer.name_input += event.unicode
                continue

            # --- State: MENU ---
            if renderer.state == "MENU":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    if renderer.menu_rects.get("create").collidepoint(event.pos):
                        renderer.enter_config_editor()
                        renderer.state = "CONFIG"
                    elif renderer.menu_rects.get("join").collidepoint(event.pos):
                        renderer.menu_message = ""
                        if net:
                            net.send({"type": 1013, "payload": {}})
                            renderer.state = "ROOM_LIST"
                continue

            # --- State: ROOM_LIST ---
            if renderer.state == "ROOM_LIST":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_ESCAPE:
                        renderer.state = "MENU"
                    elif event.key == pygame.K_r:
                        if net: net.send({"type": 1013, "payload": {}})
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
                                net.send({"type": 1011, "payload": {"room_id": rid}})
                elif event.type == pygame.MOUSEBUTTONDOWN:
                    if renderer.room_list_refresh_rect and renderer.room_list_refresh_rect.collidepoint(event.pos):
                        if net: net.send({"type": 1013, "payload": {}})
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
                                    net.send({"type": 1011, "payload": {"room_id": rid}})
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
                            if net:
                                net.send({"type": 1010, "payload": {"room_name": rn, "config": renderer.config_data}})
                        continue
                    if renderer.config_back_rect and renderer.config_back_rect.collidepoint(event.pos):
                        renderer.state = "MENU"
                        continue

                    # Select row
                    for idx, rect in getattr(renderer, "config_row_rects", []):
                        if rect.collidepoint(event.pos):
                            renderer.config_focus = "table"
                            renderer.config_selected = idx
                            renderer.config_editing = False
                            renderer.config_edit_buffer = ""
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
                        else:
                            renderer.room_name_input += event.unicode
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
                            if renderer.config_rows and renderer.config_selected < len(renderer.config_rows):
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
                                    if net:
                                        net.send({"type": 1010, "payload": {"room_name": rn, "config": renderer.config_data}})
                            else:
                                # Start/commit row editing
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
                                renderer.config_edit_buffer += event.unicode
                continue

            # --- State: GAME ---
            if renderer.state == "GAME":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    if event.button == 1:
                        # Lobby Back Button
                        if state.phase == 0 and hasattr(renderer, 'lobby_back_rect') and renderer.lobby_back_rect.collidepoint(event.pos):
                            renderer.state = "MENU"
                            state = GameState() # Reset
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
                            for ent in state.entities:
                                if ent["type"] == "MERCHANT":
                                    d = ((state.my_pos[0]-ent["pos"]["x"])**2 + (state.my_pos[1]-ent["pos"]["y"])**2)**0.5
                                    if d <= 2.0: near_merchant = True; break
                            if near_merchant: renderer.show_shop = True
                            elif net: net.send({"type": 2003, "payload": {}}) # Interact
                        
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

                elif event.type == pygame.MOUSEWHEEL:
                    if renderer.pause_view() == "item_manual":
                        renderer.scroll_item_manual(-event.y * 40)

                continue

        # Network
        if net:
            while not recv_q.empty():
                msg = recv_q.get()
                mt, pl = msg.get("type"), msg.get("payload")
                if mt == 1012:
                    renderer.state = "GAME"
                    state.config = pl.get("config")
                    renderer.menu_message = ""
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
