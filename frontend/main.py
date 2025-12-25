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
                        renderer.state = "CONFIG"
                    elif renderer.menu_rects.get("join").collidepoint(event.pos):
                        if net: net.send({"type": 1011, "payload": {}})
                continue

            # --- State: CONFIG ---
            if renderer.state == "CONFIG":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_UP: renderer.config_active_idx = (renderer.config_active_idx - 1) % len(renderer.config_keys)
                    elif event.key == pygame.K_DOWN: renderer.config_active_idx = (renderer.config_active_idx + 1) % len(renderer.config_keys)
                    elif event.key == pygame.K_RETURN:
                        # Deploy
                        payload = {k: float(v) for k, v in renderer.config_inputs.items()}
                        if net: net.send({"type": 1010, "payload": payload})
                    elif event.key == pygame.K_BACKSPACE:
                        key = renderer.config_keys[renderer.config_active_idx]
                        renderer.config_inputs[key] = renderer.config_inputs[key][:-1]
                    else:
                        key = renderer.config_keys[renderer.config_active_idx]
                        renderer.config_inputs[key] += event.unicode
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
                        elif event.key >= pygame.K_1 and event.key <= pygame.K_3:
                            ids = ["WPN_SHOCK", "SURV_MEDKIT", "RECON_RADAR"] 
                            if state.phase == 2: ids = ["WPN_SHOCK_T2", "SURV_MEDKIT_T2", "RECON_RADAR_T2"]
                            elif state.phase >= 3: ids = ["WPN_SHOCK_T3", "RECON_RADAR_T3", "SURV_MEDKIT_T2"]
                            
                            idx = event.key-pygame.K_1
                            if idx < len(ids):
                                net.send({"type": 2007, "payload": {"item_id": ids[idx]}})

                if event.type == pygame.KEYUP:
                    if event.key in (pygame.K_w, pygame.K_s): input_dir[1] = 0
                    if event.key in (pygame.K_a, pygame.K_d): input_dir[0] = 0

            # --- State: PAUSE ---
            if renderer.state == "PAUSE":
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_ESCAPE:
                        renderer.show_settings = renderer.show_help = False
                        renderer.state = "GAME"
                elif event.type == pygame.MOUSEBUTTONDOWN:
                    if renderer.show_settings or renderer.show_help:
                         renderer.handle_click(event.pos)
                    else:
                        action = renderer.handle_pause_click(event.pos)
                        if action == "resume": renderer.state = "GAME"
                        elif action == "settings": renderer.show_settings = True
                        elif action == "help": renderer.show_help = True
                        elif action == "quit":
                            renderer.state = "MENU"
                            state = GameState() # Reset local
                            renderer.show_settings = renderer.show_help = False

        # Network
        if net:
            while not recv_q.empty():
                msg = recv_q.get()
                mt, pl = msg.get("type"), msg.get("payload")
                if mt == 1012: renderer.state = "GAME"; state.config = pl.get("config")
                elif mt == 3001: 
                    state.map_tiles = pl["map_tiles"]
                    state.my_pos = [pl["spawn_pos"]["x"], pl["spawn_pos"]["y"]]
                elif mt == 3002: state.update_from_server(pl)

        # Logic
        if renderer.state == "GAME" and net:
            if getattr(state, "is_extracted", False) and renderer.spectator_mode:
                # Free Spectate Camera Movement
                speed = 10.0 * (1.0/60.0) # approx dt
                renderer.cam_offset[0] += input_dir[0] * speed
                renderer.cam_offset[1] += input_dir[1] * speed
            elif state.phase > 0 and not renderer.show_shop:
                net.send({"type": 2001, "payload": {"dir": {"x": float(input_dir[0]), "y": float(input_dir[1])}}})

        renderer.draw_game(state)
        pygame.display.flip()
        clock.tick(60)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()
