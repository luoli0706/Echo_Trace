import pygame
import math
import time
from datetime import datetime

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
COLOR_MENU_BG = (20, 20, 30, 230)
COLOR_BTN = (50, 50, 60)
COLOR_INV_BG = (30, 30, 40, 200)
COLOR_RADAR_BG = (0, 20, 30, 200)
COLOR_RADAR_BORDER = (0, 200, 255)

COLOR_ITEM_OFFENSE = (255, 100, 100)
COLOR_ITEM_SURVIVAL = (100, 255, 100)
COLOR_ITEM_RECON = (100, 100, 255)
COLOR_SUPPLY_DROP = (255, 0, 255)

COLOR_MOTOR_ACTIVE = (255, 255, 0)
COLOR_MOTOR_DONE = (0, 255, 255)
COLOR_EXIT = (0, 255, 0)

class Renderer:
    def __init__(self, screen):
        self.screen = screen
        self.font = pygame.font.SysFont("segoeuiemoji", FONT_SIZE) 
        if not self.font: self.font = pygame.font.SysFont("arial", FONT_SIZE)
        
        self.hud_font = pygame.font.SysFont("consolas", 16)
        self.time_font = pygame.font.SysFont("consolas", 24)
        self.fog_surf = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
        
        # UI State
        self.show_settings = False
        self.show_help = False
        self.show_shop = False
        self.dev_mode = False
        
        self.settings_rect = pygame.Rect(WINDOW_WIDTH//2 - 150, WINDOW_HEIGHT//2 - 100, 300, 250)
        self.help_rect = pygame.Rect(WINDOW_WIDTH//2 - 300, WINDOW_HEIGHT//2 - 250, 600, 500)
        self.shop_rect = pygame.Rect(WINDOW_WIDTH//2 - 200, WINDOW_HEIGHT//2 - 200, 400, 400)
        
        self.gear_rect = pygame.Rect(WINDOW_WIDTH - 40, 50, 30, 30)
        self.help_btn_rect = pygame.Rect(WINDOW_WIDTH - 80, 50, 30, 30)
        self.shop_btn_rect = pygame.Rect(WINDOW_WIDTH - 120, 50, 30, 30)
        
        self.dev_mode_rect = pygame.Rect(WINDOW_WIDTH//2 - 120, WINDOW_HEIGHT//2 + 50, 240, 30)
        
        self.radar_rect = pygame.Rect(WINDOW_WIDTH - 160, WINDOW_HEIGHT - 160, 150, 150)
        self.pulse_start_time = 0
        
        # Login UI
        self.name_input_text = ""
        self.login_active = True

    def world_to_screen(self, wx, wy, cam_x, cam_y):
        sx = (wx * GRID_SIZE) - cam_x + (WINDOW_WIDTH // 2)
        sy = (wy * GRID_SIZE) - cam_y + (WINDOW_HEIGHT // 2)
        return int(sx), int(sy)

    def trigger_pulse(self):
        self.pulse_start_time = time.time()

    def draw_game(self, state):
        if self.login_active:
            self.draw_login()
            return

        if state.phase == 0:
            self.draw_lobby(state)
            return

        self.screen.fill(COLOR_BG)
        cam_x = state.my_pos[0] * GRID_SIZE
        cam_y = state.my_pos[1] * GRID_SIZE

        # 1. Map
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

        # 2. Entities
        for ent in state.entities:
            ex, ey = ent["pos"]["x"], ent["pos"]["y"]
            sx, sy = self.world_to_screen(ex, ey, cam_x, cam_y)
            
            if ent["type"] == "ITEM_DROP":
                item_data = ent.get("extra")
                color = (255, 255, 0)
                if item_data:
                    itype = item_data.get("type")
                    if itype == "OFFENSE": color = COLOR_ITEM_OFFENSE
                    elif itype == "SURVIVAL": color = COLOR_ITEM_SURVIVAL
                    elif itype == "RECON": color = COLOR_ITEM_RECON
                self.draw_text_centered("📦", sx, sy, color)
            
            elif ent["type"] == "SUPPLY_DROP":
                self.draw_text_centered("🎁", sx, sy, COLOR_SUPPLY_DROP)
                # Draw a glow
                pygame.draw.circle(self.screen, COLOR_SUPPLY_DROP, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE, 1)

            elif ent["type"] == "MOTOR":
                color = COLOR_MOTOR_DONE if ent["state"] == 2 else COLOR_MOTOR_ACTIVE
                self.draw_text_centered("⚡", sx, sy, color)
                if ent["state"] != 2:
                    extra = ent.get("extra", {})
                    if extra:
                        prog = extra.get("progress", 0)
                        max_p = extra.get("max_progress", 100)
                        self.draw_bar(sx, sy-10, prog, max_p, (0, 255, 255))
            
            elif ent["type"] == "EXIT":
                self.draw_text_centered("🚪", sx, sy, COLOR_EXIT)

        # 3. Players (Smaller Size: 0.5)
        player_draw_radius = GRID_SIZE // 4 
        for pid, p in state.players.items():
            px, py = p["pos"]["x"], p["pos"]["y"]
            sx, sy = self.world_to_screen(px, py, cam_x, cam_y)
            # Center the circle in the grid cell
            center = (sx + GRID_SIZE//2, sy + GRID_SIZE//2)
            pygame.draw.circle(self.screen, COLOR_ENEMY, center, player_draw_radius)
            self.draw_text_centered("👹", sx, sy) # Text might be too big now?
            self.draw_hp_bar(sx, sy - 5, p["hp"], p["max_hp"])

        # 4. Self
        sx, sy = self.world_to_screen(state.my_pos[0], state.my_pos[1], cam_x, cam_y)
        center = (sx + GRID_SIZE//2, sy + GRID_SIZE//2)
        pygame.draw.circle(self.screen, COLOR_SELF, center, player_draw_radius)
        self.draw_text_centered("🏃", sx, sy)
        self.draw_hp_bar(sx, sy - 5, state.my_hp, 100)

        # 5. Fog (Skip if Dev Mode)
        if not self.dev_mode:
            self.fog_surf.fill(COLOR_FOG)
            view_px = int(state.view_radius * GRID_SIZE)
            pygame.draw.circle(self.fog_surf, (0,0,0,0), (WINDOW_WIDTH//2, WINDOW_HEIGHT//2), view_px)
            self.screen.blit(self.fog_surf, (0,0))

        # 6. Pulse Phantoms (Through Fog)
        for blip in state.radar_blips:
            bx, by = blip["pos"]["x"], blip["pos"]["y"]
            sx, sy = self.world_to_screen(bx, by, cam_x, cam_y)
            
            if blip["type"] == "MOTOR":
                self.draw_text_centered("⚡", sx, sy, COLOR_MOTOR_ACTIVE)
                pygame.draw.circle(self.screen, COLOR_MOTOR_ACTIVE, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE, 1)
            elif blip["type"] == "EXIT":
                self.draw_text_centered("🚪", sx, sy, COLOR_EXIT)
            elif blip["type"] == "SUPPLY_DROP":
                self.draw_text_centered("🎁", sx, sy, COLOR_SUPPLY_DROP)
                pygame.draw.circle(self.screen, COLOR_SUPPLY_DROP, (sx + GRID_SIZE//2, sy + GRID_SIZE//2), GRID_SIZE, 1)

        # 7. Sound Indicators (Hear Radius)
        center_x, center_y = WINDOW_WIDTH//2, WINDOW_HEIGHT//2
        for snd in state.sound_events:
            dx, dy = snd["dir"]["x"], snd["dir"]["y"]
            intensity = snd["intensity"]
            radius = 100 + (1.0-intensity) * 200 
            angle = math.atan2(dy, dx)
            ix = center_x + math.cos(angle) * 200
            iy = center_y + math.sin(angle) * 200
            
            pygame.draw.circle(self.screen, (255, 255, 255), (int(ix), int(iy)), int(10 * intensity))

        # 8. UI Layers
        self.draw_hud(state)
        self.draw_inventory(state)
        self.draw_events(state)
        self.draw_system_clock()
        self.draw_minimap(state)
        self.draw_ui_buttons()
        
        if self.show_settings: self.draw_settings_menu()
        if self.show_help: self.draw_help_menu()
        if self.show_shop: self.draw_shop_menu(state)

    def draw_login(self):
        self.screen.fill(COLOR_BG)
        title = self.font.render("ECHO TRACE - LOGIN", True, (0, 255, 255))
        rect = title.get_rect(center=(WINDOW_WIDTH//2, WINDOW_HEIGHT//2 - 50))
        self.screen.blit(title, rect)
        
        input_rect = pygame.Rect(WINDOW_WIDTH//2 - 100, WINDOW_HEIGHT//2, 200, 40)
        pygame.draw.rect(self.screen, (50, 50, 60), input_rect)
        pygame.draw.rect(self.screen, (0, 255, 255), input_rect, 2)
        
        name_surf = self.font.render(self.name_input_text + "|", True, (255, 255, 255))
        self.screen.blit(name_surf, (input_rect.x + 10, input_rect.y + 5))
        
        hint = self.hud_font.render("Enter Name and Press Enter", True, (150, 150, 150))
        self.screen.blit(hint, (WINDOW_WIDTH//2 - 100, WINDOW_HEIGHT//2 + 50))

    def draw_lobby(self, state):
        self.screen.fill(COLOR_BG)
        
        title = self.font.render("ECHO TRACE - TACTICAL SETUP", True, (0, 255, 255))
        rect = title.get_rect(center=(WINDOW_WIDTH//2, 100))
        self.screen.blit(title, rect)

        if state.tactic_chosen:
            msg = self.font.render("Waiting for other players...", True, (255, 255, 0))
            rect = msg.get_rect(center=(WINDOW_WIDTH//2, WINDOW_HEIGHT//2))
            self.screen.blit(msg, rect)
        else:
            prompt = self.hud_font.render("Select your Loadout (Press 1, 2, or 3):", True, (200, 200, 200))
            self.screen.blit(prompt, (WINDOW_WIDTH//2 - 150, 200))
            
            opts = [
                "1. RECON (Scanner + Light Armor)",
                "2. DEFENSE (MedKit + Heavy Armor)",
                "3. TRAP (Stun Gun + Trap Kit)"
            ]
            y = 250
            for opt in opts:
                surf = self.hud_font.render(opt, True, (255, 255, 255))
                self.screen.blit(surf, (WINDOW_WIDTH//2 - 120, y))
                y += 40

    def draw_minimap(self, state):
        # Background
        pygame.draw.rect(self.screen, COLOR_RADAR_BG, self.radar_rect, border_radius=75)
        pygame.draw.circle(self.screen, COLOR_RADAR_BORDER, self.radar_rect.center, 75, 2)
        
        # Scale: Map 32x32 -> Radar 150x150. Scale ~4.5
        scale = 150.0 / max(state.map_width, state.map_height)
        offset_x = self.radar_rect.centerx - (state.map_width * scale)/2
        offset_y = self.radar_rect.centery - (state.map_height * scale)/2

        # Draw Blips
        for blip in state.radar_blips:
            bx, by = blip["pos"]["x"], blip["pos"]["y"]
            mx = int(offset_x + bx * scale)
            my = int(offset_y + by * scale)
            
            if blip["type"] == "MOTOR":
                pygame.draw.circle(self.screen, COLOR_MOTOR_ACTIVE, (mx, my), 3)
            elif blip["type"] == "EXIT":
                pygame.draw.circle(self.screen, COLOR_EXIT, (mx, my), 4)
            elif blip["type"] == "SUPPLY_DROP":
                # Distinct Square for Supply Drop
                pygame.draw.rect(self.screen, COLOR_SUPPLY_DROP, (mx-4, my-4, 8, 8))
                pygame.draw.rect(self.screen, (255, 255, 255), (mx-4, my-4, 8, 8), 1)

        # Draw Self
        mx = int(offset_x + state.my_pos[0] * scale)
        my = int(offset_y + state.my_pos[1] * scale)
        pygame.draw.circle(self.screen, COLOR_SELF, (mx, my), 3)

    # ... [Keep existing draw_bar, draw_hp_bar, draw_text_centered, draw_hud, draw_system_clock, etc.]
    # Re-pasting them for completeness to ensure file correctness.
    
    def draw_bar(self, x, y, val, max_val, color):
        width = GRID_SIZE
        height = 4
        pct = max(0, min(1, val / max_val)) if max_val > 0 else 0
        pygame.draw.rect(self.screen, (50, 50, 50), (x, y, width, height))
        pygame.draw.rect(self.screen, color, (x, y, width * pct, height))

    def draw_hp_bar(self, x, y, hp, max_hp):
        width = GRID_SIZE
        height = 4
        pct = max(0, min(1, hp / max_hp)) if max_hp > 0 else 0
        r = min(255, max(0, int(255 * (1 - pct) * 2)))
        g = min(255, max(0, int(255 * pct * 2)))
        pygame.draw.rect(self.screen, COLOR_HP_BG, (x, y, width, height))
        pygame.draw.rect(self.screen, (r, g, 0), (x, y, width * pct, height))

    def draw_text_centered(self, text, x, y, color=(255, 255, 255)):
        surf = self.font.render(text, True, color)
        rect = surf.get_rect(center=(x + GRID_SIZE//2, y + GRID_SIZE//2))
        self.screen.blit(surf, rect)

    def draw_hud(self, state):
        texts = [
            f"HP: {state.my_hp:.0f}%",
            f"FUNDS: ${state.funds}",
            f"Pos: ({state.my_pos[0]:.1f}, {state.my_pos[1]:.1f})",
            f"View: {state.view_radius}m",
            "Controls: WASD, E Pickup, F Fix, B Shop",
        ]
        if self.dev_mode:
            texts.append("DEV MODE: F9 Skip Phase")
            
        y = 10
        for t in texts:
            surf = self.hud_font.render(t, True, COLOR_HUD_TEXT)
            self.screen.blit(surf, (10, y))
            y += 20
        phase_map = {1: "SEARCH", 2: "CONFLICT", 3: "ESCAPE", 4: "ENDED"}
        phase_txt = phase_map.get(getattr(state, "phase", 1), "UNKNOWN")
        phase_surf = self.font.render(f"PHASE: {phase_txt} | Time: {state.time_left:.0f}", True, (255, 255, 0))
        p_rect = phase_surf.get_rect(center=(WINDOW_WIDTH//2, 30))
        self.screen.blit(phase_surf, p_rect)

    def draw_system_clock(self):
        now_str = datetime.now().strftime("%H:%M:%S")
        surf = self.time_font.render(now_str, True, (0, 255, 255))
        self.screen.blit(surf, (WINDOW_WIDTH - surf.get_width() - 10, 10))

    def draw_ui_buttons(self):
        pygame.draw.rect(self.screen, COLOR_BTN, self.gear_rect, border_radius=5)
        self.draw_text_centered("⚙️", self.gear_rect.x, self.gear_rect.y)
        pygame.draw.rect(self.screen, COLOR_BTN, self.help_btn_rect, border_radius=5)
        self.draw_text_centered("?", self.help_btn_rect.x, self.help_btn_rect.y)
        pygame.draw.rect(self.screen, COLOR_BTN, self.shop_btn_rect, border_radius=5)
        self.draw_text_centered("$", self.shop_btn_rect.x, self.shop_btn_rect.y)

    def draw_events(self, state):
        y = 100
        for evt in state.events:
            txt = f"[{evt.get('type')}] {evt.get('msg')}"
            surf = self.hud_font.render(txt, True, (255, 100, 255))
            self.screen.blit(surf, (WINDOW_WIDTH - surf.get_width() - 10, y))
            y += 20

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
                color = (200, 200, 200)
                if itype == "OFFENSE": color = COLOR_ITEM_OFFENSE
                elif itype == "SURVIVAL": color = COLOR_ITEM_SURVIVAL
                elif itype == "RECON": color = COLOR_ITEM_RECON
                name_surf = self.hud_font.render(item.get("name", "???")[:4], True, color)
                self.screen.blit(name_surf, (rect[0]+5, rect[1]+15))

    def draw_settings_menu(self):
        s = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
        s.fill((0,0,0,150))
        self.screen.blit(s, (0,0))
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.settings_rect, border_radius=10)
        pygame.draw.rect(self.screen, (255,255,255), self.settings_rect, 2, border_radius=10)
        title = self.hud_font.render("SETTINGS", True, (255,255,255))
        self.screen.blit(title, (self.settings_rect.x + 20, self.settings_rect.y + 20))
        
        # Dev Mode Toggle
        dev_color = (0, 255, 0) if self.dev_mode else (100, 100, 100)
        pygame.draw.rect(self.screen, dev_color, self.dev_mode_rect, 2)
        dev_txt = self.hud_font.render(f"Developer Mode: {'ON' if self.dev_mode else 'OFF'}", True, (255, 255, 255))
        self.screen.blit(dev_txt, (self.dev_mode_rect.x + 10, self.dev_mode_rect.y + 5))

        opts = ["Volume: [||||||  ]", "Graphics: [High]", "Quit Game"]
        y_off = 100
        for o in opts:
            opt_surf = self.hud_font.render(o, True, (200,200,200))
            self.screen.blit(opt_surf, (self.settings_rect.x + 30, self.settings_rect.y + y_off))
            y_off += 30

    def draw_shop_menu(self, state):
        s = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
        s.fill((0,0,0,200))
        self.screen.blit(s, (0,0))
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.shop_rect, border_radius=10)
        pygame.draw.rect(self.screen, (255, 215, 0), self.shop_rect, 2, border_radius=10)
        
        title = self.font.render("BLACK MARKET", True, (255, 215, 0))
        self.screen.blit(title, (self.shop_rect.x + 20, self.shop_rect.y + 20))
        
        funds = self.font.render(f"Your Funds: ${state.funds}", True, (0, 255, 0))
        self.screen.blit(funds, (self.shop_rect.x + 200, self.shop_rect.y + 20))

        # Example Shop Items
        items = [
            ("Stun Gun (T1)", "WPN_SHOCK", 100),
            ("MedKit (T1)", "SURV_MEDKIT", 50),
            ("Scanner (T1)", "RECON_RADAR", 150),
        ]
        
        y_off = 70
        for name, pid, cost in items:
            color = (255, 255, 255)
            if state.funds < cost: color = (100, 100, 100)
            
            txt = f"{name} - ${cost} [Press {items.index((name, pid, cost)) + 1}]"
            surf = self.hud_font.render(txt, True, color)
            self.screen.blit(surf, (self.shop_rect.x + 30, self.shop_rect.y + y_off))
            y_off += 40
            
        hint = self.hud_font.render("Press 1-3 to Buy. B to Close.", True, (150, 150, 150))
        self.screen.blit(hint, (self.shop_rect.x + 30, self.shop_rect.y + 350))

    def handle_click(self, pos):
        if self.gear_rect.collidepoint(pos):
            self.show_settings = not self.show_settings
            self.show_help = False
            self.show_shop = False
            return True
        if self.help_btn_rect.collidepoint(pos):
            self.show_help = not self.show_help
            self.show_settings = False
            self.show_shop = False
            return True
        if self.shop_btn_rect.collidepoint(pos):
            self.show_shop = not self.show_shop
            self.show_settings = False
            self.show_help = False
            return True
            
        if self.show_settings and self.dev_mode_rect.collidepoint(pos):
            self.dev_mode = not self.dev_mode
            return True
            
        return False