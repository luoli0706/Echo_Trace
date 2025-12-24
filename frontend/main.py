import sys
import queue
import pygame
from client.network import NetworkClient
from client.gamestate import GameState
from client.renderer import Renderer, WINDOW_WIDTH, WINDOW_HEIGHT

# Default Server (Used if not changed in Connect UI)
DEFAULT_SERVER_URL = "ws://localhost:8080/ws"

def main():
    pygame.init()
    screen = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
    pygame.display.set_caption("Echo Trace Client [Alpha 0.5 - Rooms]")
    clock = pygame.time.Clock()

    recv_q = queue.Queue()
    # Net is initialized later in CONNECT state
    net = None

    state = GameState()
    renderer = Renderer(screen)
    input_dir = [0, 0]

    running = True
    while running:
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            
            # --- State: CONNECT ---
            if renderer.state == "CONNECT":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_RETURN:
                        url = renderer.server_input.strip()
                        if not url: url = DEFAULT_SERVER_URL
                        print(f"Connecting to {url}...")
                        net = NetworkClient(url, recv_q)
                        net.start()
                        renderer.state = "LOGIN"
                    elif event.key == pygame.K_BACKSPACE:
                        renderer.server_input = renderer.server_input[:-1]
                    else:
                        renderer.server_input += event.unicode
                continue

            # --- State: LOGIN ---
            if renderer.state == "LOGIN":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_RETURN:
                        name = renderer.name_input.strip()
                        if not name: name = "Agent_47"
                        print(f"Logging in as {name}...")
                        # Send Login (1001)
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
                    pos = event.pos
                    if renderer.menu_rects.get("create", pygame.Rect(0,0,0,0)).collidepoint(pos):
                        renderer.state = "CONFIG"
                    elif renderer.menu_rects.get("join", pygame.Rect(0,0,0,0)).collidepoint(pos):
                        # Join Random (1011)
                        if net: net.send({"type": 1011, "payload": {}})
                continue

            # --- State: CONFIG (Create Room) ---
            if renderer.state == "CONFIG":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_UP:
                        renderer.config_active_idx = (renderer.config_active_idx - 1) % len(renderer.config_keys)
                    elif event.key == pygame.K_DOWN:
                        renderer.config_active_idx = (renderer.config_active_idx + 1) % len(renderer.config_keys)
                    elif event.key == pygame.K_RETURN:
                        # Create Room (1010)
                        payload = {}
                        try:
                            payload["max_players"] = float(renderer.config_inputs["max_players"])
                            payload["motors"] = float(renderer.config_inputs["motors"])
                            payload["phase1_dur"] = float(renderer.config_inputs["p1_dur"])
                            payload["phase2_dur"] = float(renderer.config_inputs["p2_dur"])
                        except ValueError:
                            pass
                        if net: net.send({"type": 1010, "payload": payload})
                    elif event.key == pygame.K_BACKSPACE:
                        key = renderer.config_keys[renderer.config_active_idx]
                        renderer.config_inputs[key] = renderer.config_inputs[key][:-1]
                    else:
                        key = renderer.config_keys[renderer.config_active_idx]
                        renderer.config_inputs[key] += event.unicode
                continue

            # --- State: GAME (Playing) ---
            if renderer.state == "GAME":
                if event.type == pygame.MOUSEBUTTONDOWN:
                    if event.button == 1:
                        renderer.handle_click(event.pos)

                if event.type == pygame.KEYDOWN:
                    # Shop
                    if renderer.show_shop:
                        if event.key >= pygame.K_1 and event.key <= pygame.K_3:
                            items = []
                            if state.phase == 1:
                                items = ["WPN_SHOCK", "SURV_MEDKIT", "RECON_RADAR"]
                            elif state.phase == 2:
                                items = ["WPN_SHOCK_T2", "SURV_MEDKIT_T2", "RECON_RADAR_T2"]
                            elif state.phase >= 3:
                                items = ["WPN_SHOCK_T3", "RECON_RADAR_T3", "SURV_MEDKIT_T2"]
                                
                            idx = event.key - pygame.K_1
                            if idx < len(items):
                                net.send({"type": 2007, "payload": {"item_id": items[idx]}})
                        # 'b' to close handled below
                        
                    # Toggle Shop/Settings
                    if event.key == pygame.K_b:
                        renderer.show_shop = not renderer.show_shop
                        renderer.show_settings = False
                        renderer.show_help = False
                        
                    if renderer.show_shop: continue

                    if event.key == pygame.K_w: input_dir[1] = -1
                    elif event.key == pygame.K_s: input_dir[1] = 1
                    elif event.key == pygame.K_a: input_dir[0] = -1
                    elif event.key == pygame.K_d: input_dir[0] = 1
                    
                    elif event.key == pygame.K_e:
                        net.send({"type": 2004, "payload": {}}) # Pickup
                    elif event.key == pygame.K_f:
                        net.send({"type": 2003, "payload": {}}) # Interact
                    
                    # Number Keys
                    elif event.key >= pygame.K_1 and event.key <= pygame.K_6:
                        mods = pygame.key.get_mods()
                        if state.phase == 0 and not state.tactic_chosen:
                            if event.key <= pygame.K_3:
                                tactic_map = {pygame.K_1: "RECON", pygame.K_2: "DEFENSE", pygame.K_3: "TRAP"}
                                tactic = tactic_map.get(event.key)
                                if tactic:
                                    net.send({"type": 2006, "payload": {"tactic": tactic}})
                                    state.tactic_chosen = True
                        else:
                            slot = event.key - pygame.K_1
                            if mods & pygame.KMOD_SHIFT:
                                # Drop
                                net.send({"type": 2005, "payload": {"slot_index": slot}})
                            elif mods & pygame.KMOD_CTRL:
                                # Sell
                                net.send({"type": 2008, "payload": {"slot_index": slot}})
                            else:
                                # Use
                                net.send({"type": 2002, "payload": {"slot_index": slot}})
                
                if event.type == pygame.KEYUP:
                    if event.key in (pygame.K_w, pygame.K_s): input_dir[1] = 0
                    if event.key in (pygame.K_a, pygame.K_d): input_dir[0] = 0

        # Network Handling
        if net:
            while not recv_q.empty():
                msg = recv_q.get()
                msg_type = msg.get("type")
                payload = msg.get("payload")

                if msg_type == 1012: # ROOM_JOINED
                    print(f"Joined Room: {payload.get('room_id')}")
                    state.config = payload.get("config", {})
                    renderer.state = "GAME"
                    
                elif msg_type == 1001:
                    state.self_id = payload.get("session_id")
                    print(f"Logged in as {state.self_id}")

                elif msg_type == 3001: 
                    state.map_tiles = payload["map_tiles"]
                    state.my_pos = [payload["spawn_pos"]["x"], payload["spawn_pos"]["y"]]
                    state.my_inventory = payload.get("inventory", [])
                    print(f"Map Loaded: {len(state.map_tiles)}x{len(state.map_tiles[0])}")
                elif msg_type == 3002: 
                    state.update_from_server(payload)

        # Game Loop Logic (Only if in GAME state)
        if renderer.state == "GAME" and net and state.phase > 0:
            move_req = {
                "type": 2001, 
                "payload": {"dir": {"x": float(input_dir[0]), "y": float(input_dir[1])}}
            }
            net.send(move_req)

        renderer.draw_game(state)
        pygame.display.flip()
        clock.tick(60)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()
