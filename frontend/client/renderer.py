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
COLOR_HP_BG = (50, 0, 0)
COLOR_HP_FG = (0, 255, 0)
COLOR_MENU_BG = (20, 20, 30, 230)
COLOR_BTN = (50, 50, 60)
COLOR_BTN_HOVER = (70, 70, 80)

class Renderer:
    def __init__(self, screen):
        self.screen = screen
        self.font = pygame.font.SysFont("segoeuiemoji", FONT_SIZE) # Emoji capable
        # Fallback if needed, but modern Win10/11 usually has Segoe UI Emoji
        
        self.hud_font = pygame.font.SysFont("consolas", 16)
        self.fog_surf = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
        
        # UI State
        self.show_settings = False
        self.settings_rect = pygame.Rect(WINDOW_WIDTH//2 - 150, WINDOW_HEIGHT//2 - 100, 300, 200)
        self.gear_rect = pygame.Rect(WINDOW_WIDTH - 40, 10, 30, 30)

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
            # HP Bar
            self.draw_hp_bar(sx, sy - 5, p["hp"], p["max_hp"])

        # Draw Self
        sx, sy = self.world_to_screen(state.my_pos[0], state.my_pos[1], cam_x, cam_y)
        pygame.draw.circle(self.screen, COLOR_SELF, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE//2 - 2)
        self.draw_text_centered("🏃", sx, sy)
        # Self HP Bar (redundant with HUD but good for focus)
        self.draw_hp_bar(sx, sy - 5, state.my_hp, 100)

        # Draw Fog
        self.fog_surf.fill(COLOR_FOG)
        view_px = int(state.view_radius * GRID_SIZE)
        pygame.draw.circle(self.fog_surf, (0,0,0,0), (WINDOW_WIDTH//2, WINDOW_HEIGHT//2), view_px)
        self.screen.blit(self.fog_surf, (0,0))

        # HUD
        self.draw_hud(state)
        
        # UI Overlay (Settings)
        self.draw_ui()

    def draw_hp_bar(self, x, y, hp, max_hp):
        width = GRID_SIZE
        height = 4
        pct = max(0, min(1, hp / max_hp)) if max_hp > 0 else 0
        pygame.draw.rect(self.screen, COLOR_HP_BG, (x, y, width, height))
        pygame.draw.rect(self.screen, COLOR_HP_FG, (x, y, width * pct, height))

    def draw_text_centered(self, text, x, y):
        surf = self.font.render(text, True, (255, 255, 255))
        rect = surf.get_rect(center=(x + GRID_SIZE//2, y + GRID_SIZE//2))
        self.screen.blit(surf, rect)

    def draw_hud(self, state):
        texts = [
            f"HP: {state.my_hp:.0f}%",
            f"Pos: ({state.my_pos[0]:.1f}, {state.my_pos[1]:.1f})",
            f"View: {state.view_radius}m",
            "Controls: WASD to Move, SPACE to Attack",
        ]
        y = 10
        for t in texts:
            surf = self.hud_font.render(t, True, COLOR_HUD_TEXT)
            self.screen.blit(surf, (10, y))
            y += 20

    def draw_ui(self):
        # Gear Icon
        pygame.draw.rect(self.screen, COLOR_BTN, self.gear_rect, border_radius=5)
        self.draw_text_centered("⚙️", self.gear_rect.x, self.gear_rect.y) # Use Emoji or text

        # Settings Menu
        if self.show_settings:
            s = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
            s.fill((0,0,0,150))
            self.screen.blit(s, (0,0))
            
            pygame.draw.rect(self.screen, COLOR_MENU_BG, self.settings_rect, border_radius=10)
            pygame.draw.rect(self.screen, (255,255,255), self.settings_rect, 2, border_radius=10)
            
            # Title
            title = self.hud_font.render("SETTINGS", True, (255,255,255))
            self.screen.blit(title, (self.settings_rect.x + 20, self.settings_rect.y + 20))
            
            # Placeholder Options
            opts = ["Volume: [||||||  ]", "Graphics: [High]", "Quit Game"]
            y_off = 60
            for o in opts:
                opt_surf = self.hud_font.render(o, True, (200,200,200))
                self.screen.blit(opt_surf, (self.settings_rect.x + 30, self.settings_rect.y + y_off))
                y_off += 30

    def handle_click(self, pos):
        if self.gear_rect.collidepoint(pos):
            self.show_settings = not self.show_settings
            return True
        if self.show_settings and self.settings_rect.collidepoint(pos):
            # Handle menu clicks (Placeholder)
            return True
        return False