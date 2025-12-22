import pygame

WINDOW_WIDTH = 1024
WINDOW_HEIGHT = 768
GRID_SIZE = 24
FONT_SIZE = 20

# Colors
COLOR_BG = (10, 10, 15)
COLOR_GRID = (30, 30, 40)
COLOR_WALL = (60, 60, 70)
COLOR_WALL_EDGE = (0, 200, 255)
COLOR_SELF = (0, 255, 128)
COLOR_ENEMY = (255, 0, 60)
COLOR_HUD_TEXT = (200, 255, 255)
COLOR_FOG = (0, 0, 0)

class Renderer:
    def __init__(self, screen):
        self.screen = screen
        self.font = pygame.font.SysFont("segoeuiemoji", FONT_SIZE)
        self.hud_font = pygame.font.SysFont("consolas", 16)
        self.fog_surf = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)

    def world_to_screen(self, wx, wy, cam_x, cam_y):
        sx = (wx * GRID_SIZE) - cam_x + (WINDOW_WIDTH // 2)
        sy = (wy * GRID_SIZE) - cam_y + (WINDOW_HEIGHT // 2)
        return int(sx), int(sy)

    def draw_game(self, state):
        self.screen.fill(COLOR_BG)
        cam_x = state.my_pos[0] * GRID_SIZE
        cam_y = state.my_pos[1] * GRID_SIZE

        # Draw Map
        start_col = max(0, int(state.my_pos[0] - (WINDOW_WIDTH/GRID_SIZE/2)) - 2)
        end_col = int(state.my_pos[0] + (WINDOW_WIDTH/GRID_SIZE/2)) + 2
        start_row = max(0, int(state.my_pos[1] - (WINDOW_HEIGHT/GRID_SIZE/2)) - 2)
        end_row = int(state.my_pos[1] + (WINDOW_HEIGHT/GRID_SIZE/2)) + 2
        
        if state.map_tiles:
            for y in range(start_row, min(len(state.map_tiles), end_row)):
                for x in range(start_col, min(len(state.map_tiles[0]), end_col)):
                    tile = state.map_tiles[y][x]
                    sx, sy = self.world_to_screen(x, y, cam_x, cam_y)
                    rect = (sx, sy, GRID_SIZE, GRID_SIZE)
                    pygame.draw.rect(self.screen, COLOR_GRID, rect, 1)
                    if tile == 1:
                        pygame.draw.rect(self.screen, COLOR_WALL, rect)
                        pygame.draw.rect(self.screen, COLOR_WALL_EDGE, rect, 1)

        # Draw Players
        for pid, p in state.players.items():
            px, py = p["pos"]["x"], p["pos"]["y"]
            sx, sy = self.world_to_screen(px, py, cam_x, cam_y)
            pygame.draw.circle(self.screen, COLOR_ENEMY, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE//2 - 2)
            self.draw_text_centered("👹", sx, sy)

        # Draw Self
        sx, sy = self.world_to_screen(state.my_pos[0], state.my_pos[1], cam_x, cam_y)
        pygame.draw.circle(self.screen, COLOR_SELF, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE//2 - 2)
        self.draw_text_centered("🏃", sx, sy)

        # Draw Fog
        self.fog_surf.fill(COLOR_FOG)
        view_px = int(state.view_radius * GRID_SIZE)
        pygame.draw.circle(self.fog_surf, (0,0,0,0), (WINDOW_WIDTH//2, WINDOW_HEIGHT//2), view_px)
        self.screen.blit(self.fog_surf, (0,0))

        # HUD
        self.draw_hud(state)

    def draw_text_centered(self, text, x, y):
        surf = self.font.render(text, True, (255, 255, 255))
        rect = surf.get_rect(center=(x + GRID_SIZE//2, y + GRID_SIZE//2))
        self.screen.blit(surf, rect)

    def draw_hud(self, state):
        texts = [
            f"HP: {state.my_hp:.0f}%",
            f"Pos: ({state.my_pos[0]:.1f}, {state.my_pos[1]:.1f})",
            f"View: {state.view_radius}m",
            "Controls: WASD to Move",
        ]
        y = 10
        for t in texts:
            surf = self.hud_font.render(t, True, COLOR_HUD_TEXT)
            self.screen.blit(surf, (10, y))
            y += 20
