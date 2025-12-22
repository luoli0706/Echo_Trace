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
    pygame.display.set_caption("Echo Trace Client [Refactored]")
    clock = pygame.time.Clock()

    recv_q = queue.Queue()
    net = NetworkClient(SERVER_URL, recv_q)
    net.start()

    state = GameState()
    renderer = Renderer(screen)
    input_dir = [0, 0]

    running = True
    while running:
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            
            if event.type == pygame.KEYDOWN:
                if event.key == pygame.K_w: input_dir[1] = -1
                elif event.key == pygame.K_s: input_dir[1] = 1
                elif event.key == pygame.K_a: input_dir[0] = -1
                elif event.key == pygame.K_d: input_dir[0] = 1
            
            if event.type == pygame.KEYUP:
                if event.key in (pygame.K_w, pygame.K_s): input_dir[1] = 0
                if event.key in (pygame.K_a, pygame.K_d): input_dir[0] = 0

        # Process Network
        while not recv_q.empty():
            msg = recv_q.get()
            msg_type = msg.get("type")
            payload = msg.get("payload")

            if msg_type == 3001: 
                state.map_tiles = payload["map_tiles"]
                print(f"Map Loaded: {len(state.map_tiles)}x{len(state.map_tiles[0])}")
            elif msg_type == 3002: 
                state.update_from_server(payload)

        # Send Input
        move_req = {
            "type": 2001, 
            "payload": {"dir": {"x": float(input_dir[0]), "y": float(input_dir[1])}}
        }
        net.send(move_req)

        # Render
        renderer.draw_game(state)
        pygame.display.flip()
        clock.tick(60)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()
