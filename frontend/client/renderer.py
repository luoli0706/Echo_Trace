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
COLOR_HP_LOW = (255, 0, 0)
COLOR_HP_HIGH = (0, 255, 0)
COLOR_MENU_BG = (20, 20, 30, 230)
COLOR_BTN = (50, 50, 60)
COLOR_INV_BG = (30, 30, 40, 200)

# Item Type Colors
COLOR_ITEM_OFFENSE = (255, 100, 100)
COLOR_ITEM_SURVIVAL = (100, 255, 100)
COLOR_ITEM_RECON = (100, 100, 255)

class Renderer:
    def __init__(self, screen):
        self.screen = screen
        self.font = pygame.font.SysFont("segoeuiemoji", FONT_SIZE) 
        if not self.font:
             self.font = pygame.font.SysFont("arial", FONT_SIZE)
        
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

        # Draw Items
        for ent in state.entities:
            if ent["type"] == "ITEM_DROP":
                ex, ey = ent["pos"]["x"], ent["pos"]["y"]
                sx, sy = self.world_to_screen(ex, ey, cam_x, cam_y)
                
                # Determine color based on item type if available in 'extra'
                # Note: 'extra' might be a dict or map in JSON
                color = (255, 255, 0)
                item_data = ent.get("extra")
                if item_data:
                    itype = item_data.get("type")
                    if itype == "OFFENSE": color = COLOR_ITEM_OFFENSE
                    elif itype == "SURVIVAL": color = COLOR_ITEM_SURVIVAL
                    elif itype == "RECON": color = COLOR_ITEM_RECON
                
                self.draw_text_centered("📦", sx, sy, color)

        # Draw Players
        for pid, p in state.players.items():
            px, py = p["pos"]["x"], p["pos"]["y"]
            sx, sy = self.world_to_screen(px, py, cam_x, cam_y)
            pygame.draw.circle(self.screen, COLOR_ENEMY, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE//2 - 2)
            self.draw_text_centered("👹", sx, sy)
            self.draw_hp_bar(sx, sy - 5, p["hp"], p["max_hp"])

        # Draw Self
        sx, sy = self.world_to_screen(state.my_pos[0], state.my_pos[1], cam_x, cam_y)
        pygame.draw.circle(self.screen, COLOR_SELF, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE//2 - 2)
        self.draw_text_centered("🏃", sx, sy)
        self.draw_hp_bar(sx, sy - 5, state.my_hp, 100)

        # Draw Fog
        self.fog_surf.fill(COLOR_FOG)
        view_px = int(state.view_radius * GRID_SIZE)
        pygame.draw.circle(self.fog_surf, (0,0,0,0), (WINDOW_WIDTH//2, WINDOW_HEIGHT//2), view_px)
        self.screen.blit(self.fog_surf, (0,0))

        # HUD
        self.draw_hud(state)
        self.draw_inventory(state)
        self.draw_ui()

    def draw_hp_bar(self, x, y, hp, max_hp):
        width = GRID_SIZE
        height = 4
        pct = max(0, min(1, hp / max_hp)) if max_hp > 0 else 0
        
        # Color Interpolation (Red to Green)
        r = min(255, max(0, int(255 * (1 - pct) * 2)))
        g = min(255, max(0, int(255 * pct * 2)))
        color = (r, g, 0)
        
        pygame.draw.rect(self.screen, COLOR_HP_BG, (x, y, width, height))
        pygame.draw.rect(self.screen, color, (x, y, width * pct, height))

    def draw_text_centered(self, text, x, y, color=(255, 255, 255)):
        surf = self.font.render(text, True, color)
        rect = surf.get_rect(center=(x + GRID_SIZE//2, y + GRID_SIZE//2))
        self.screen.blit(surf, rect)

    def draw_hud(self, state):
        # Top Left Info
        texts = [
            f"HP: {state.my_hp:.0f}%",
            f"Pos: ({state.my_pos[0]:.1f}, {state.my_pos[1]:.1f})",
            f"View: {state.view_radius}m",
            "Controls: WASD Move, SPACE Attack, E Pickup",
        ]
        y = 10
        for t in texts:
            surf = self.hud_font.render(t, True, COLOR_HUD_TEXT)
            self.screen.blit(surf, (10, y))
            y += 20
            
        # Phase Indicator (Top Center)
        phase_map = {1: "SEARCH", 2: "CONFLICT", 3: "ESCAPE", 4: "ENDED"}
        phase_txt = phase_map.get(getattr(state, "phase", 1), "UNKNOWN")
        # Note: state.phase needs to be synced from gamestate.py update_from_server
        
        phase_surf = self.font.render(f"PHASE: {phase_txt}", True, (255, 255, 0))
        p_rect = phase_surf.get_rect(center=(WINDOW_WIDTH//2, 30))
        self.screen.blit(phase_surf, p_rect)

    def draw_inventory(self, state):
        slot_size = 50
        padding = 10
        count = 6
        total_w = count * slot_size + (count-1) * padding
        start_x = (WINDOW_WIDTH - total_w) // 2
        y = WINDOW_HEIGHT - slot_size - 20
        
        for i in range(count):
            rect = (start_x + i * (slot_size + padding), y, slot_size, slot_size)
            pygame.draw.rect(self.screen, COLOR_INV_BG, rect, border_radius=5)
            pygame.draw.rect(self.screen, (100, 100, 100), rect, 1, border_radius=5)
            
            hint = self.hud_font.render(str(i+1), True, (150, 150, 150))
            self.screen.blit(hint, (rect[0]+2, rect[1]+2))
            
            if i < len(state.my_inventory):
                item = state.my_inventory[i]
                itype = item.get("type", "UNKNOWN")
                
                # Icon Color
                color = (200, 200, 200)
                if itype == "OFFENSE": color = COLOR_ITEM_OFFENSE
                elif itype == "SURVIVAL": color = COLOR_ITEM_SURVIVAL
                elif itype == "RECON": color = COLOR_ITEM_RECON
                
                name_surf = self.hud_font.render(item.get("name", "???")[:4], True, color)
                self.screen.blit(name_surf, (rect[0]+5, rect[1]+15))

    def draw_ui(self):
        # Gear Icon
        pygame.draw.rect(self.screen, COLOR_BTN, self.gear_rect, border_radius=5)
        self.draw_text_centered("⚙️", self.gear_rect.x, self.gear_rect.y) 

        # Settings Menu
        if self.show_settings:
            s = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
            s.fill((0,0,0,150))
            self.screen.blit(s, (0,0))
            
            pygame.draw.rect(self.screen, COLOR_MENU_BG, self.settings_rect, border_radius=10)
            pygame.draw.rect(self.screen, (255,255,255), self.settings_rect, 2, border_radius=10)
            
            title = self.hud_font.render("SETTINGS", True, (255,255,255))
            self.screen.blit(title, (self.settings_rect.x + 20, self.settings_rect.y + 20))
            
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
            return True
        return False