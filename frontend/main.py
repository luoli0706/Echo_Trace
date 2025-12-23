import sys
import queue
import pygame
from client.network import NetworkClient
from client.gamestate import GameState
from client.renderer import Renderer, WINDOW_WIDTH, WINDOW_HEIGHT

SERVER_URL = "ws://localhost:8080/ws"

def main():
    pygame.init()
    screen = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
    pygame.display.set_caption("Echo Trace Client [Alpha 0.4 - Phases]")
    clock = pygame.time.Clock()

    # Name Input (Simple Console)
    player_name = input("Enter your Agent Name: ").strip()
    if not player_name: player_name = "Agent_47"

    recv_q = queue.Queue()
    net = NetworkClient(SERVER_URL, recv_q)
    net.start()
    
    # Send Login
    net.send({"type": 1001, "payload": {"name": player_name}})

    state = GameState()
    renderer = Renderer(screen)
    input_dir = [0, 0]

    running = True
    while running:
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            
            if event.type == pygame.MOUSEBUTTONDOWN:
                if event.button == 1:
                    renderer.handle_click(event.pos)

            if event.type == pygame.KEYDOWN:
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
                    if state.phase == 0 and not state.tactic_chosen:
                        if event.key <= pygame.K_3:
                            tactic_map = {pygame.K_1: "RECON", pygame.K_2: "DEFENSE", pygame.K_3: "TRAP"}
                            tactic = tactic_map.get(event.key)
                            if tactic:
                                net.send({"type": 2006, "payload": {"tactic": tactic}})
                                state.tactic_chosen = True
                    else:
                        slot = event.key - pygame.K_1
                        net.send({"type": 2002, "payload": {"slot_index": slot}})
            
            if event.type == pygame.KEYUP:
                if event.key in (pygame.K_w, pygame.K_s): input_dir[1] = 0
                if event.key in (pygame.K_a, pygame.K_d): input_dir[0] = 0

        while not recv_q.empty():
            msg = recv_q.get()
            msg_type = msg.get("type")
            payload = msg.get("payload")

            if msg_type == 1001:
                state.config = payload.get("config", {})
                state.self_id = payload.get("session_id")
                print(f"Logged in as {state.self_id}")

            elif msg_type == 3001: 
                state.map_tiles = payload["map_tiles"]
                state.my_pos = [payload["spawn_pos"]["x"], payload["spawn_pos"]["y"]]
                state.my_inventory = payload.get("inventory", [])
                print(f"Map Loaded: {len(state.map_tiles)}x{len(state.map_tiles[0])}")
            elif msg_type == 3002: 
                state.update_from_server(payload)

        # Input Handling for Phase 0
        if state.phase == 0:
             # Process events for Tactic Selection
             # Note: We already processed events above, but we need to check specific keys here if we didn't store them.
             # Actually, the event loop above already consumed events. We need to handle keys INSIDE the event loop.
             pass 

        if state.phase > 0:
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