import pygame
import math, time, os
from datetime import datetime
from client.config import *
from client.i18n import i18n

class Renderer:
    def __init__(self, screen):
        self.screen = screen
        self.assets = {}
        icon_path = os.path.join("frontend", "assets", "icos")
        if not os.path.exists(icon_path): icon_path = "assets/icos"
        def load_icon(name, key):
            try:
                p = os.path.join(icon_path, name)
                if os.path.exists(p):
                    img = pygame.image.load(p).convert_alpha()
                    self.assets[key] = pygame.transform.scale(img, (GRID_SIZE, GRID_SIZE))
            except Exception as e: print(f"Icon error {name}: {e}")
        load_icon("Treasure_Box.png", "ITEM_DROP")
        load_icon("High_value_materials.png", "SUPPLY_DROP")
        load_icon("NPC_Merchant.png", "MERCHANT")

        def get_cjk_font(size):
            for name in ["simhei", "microsoftyahei", "simsun", "wqy-microhei", "arial"]:
                fp = pygame.font.match_font(name)
                if fp: return pygame.font.Font(fp, size)
            return pygame.font.SysFont("arial", size)
        self.font = get_cjk_font(FONT_SIZE); self.hud_font = get_cjk_font(16); self.time_font = pygame.font.SysFont("consolas", 24)
        self.fog_surf = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA)
        self.state = "CONNECT"; self.server_input = "ws://localhost:8080/ws"; self.name_input = "Agent_07"
        self.config_inputs = { "max_players": "6", "motors": "5", "p1_dur": "120", "p2_dur": "180" }
        self.config_active_idx = 0; self.config_keys = ["max_players", "motors", "p1_dur", "p2_dur"]
        self.show_settings = self.show_help = self.show_shop = self.dev_mode = self.spectator_mode = False
        self.cam_offset = [0, 0]
        self.settings_rect = pygame.Rect(WINDOW_WIDTH//2 - 150, WINDOW_HEIGHT//2 - 150, 300, 300)
        self.help_rect = pygame.Rect(WINDOW_WIDTH//2 - 300, WINDOW_HEIGHT//2 - 250, 600, 500)
        self.shop_rect = pygame.Rect(WINDOW_WIDTH//2 - 200, WINDOW_HEIGHT//2 - 200, 400, 400)
        self.dev_mode_rect = pygame.Rect(WINDOW_WIDTH//2 - 120, WINDOW_HEIGHT//2 + 50, 240, 30)
        self.lang_rect = pygame.Rect(WINDOW_WIDTH//2 - 120, WINDOW_HEIGHT//2 + 90, 240, 30)
        self.back_btn_rect = pygame.Rect(WINDOW_WIDTH//2 - 60, WINDOW_HEIGHT//2 + 200, 120, 40)
        self.radar_rect = pygame.Rect(WINDOW_WIDTH - 160, WINDOW_HEIGHT - 160, 150, 150)
        self.pause_rects = {
            "resume": pygame.Rect(WINDOW_WIDTH//2 - 100, 200, 200, 50),
            "settings": pygame.Rect(WINDOW_WIDTH//2 - 100, 270, 200, 50),
            "help": pygame.Rect(WINDOW_WIDTH//2 - 100, 340, 200, 50),
            "quit": pygame.Rect(WINDOW_WIDTH//2 - 100, 450, 200, 50),
        }
        self.results_rects = {
            "spectate": pygame.Rect(WINDOW_WIDTH//2 - 210, WINDOW_HEIGHT//2 + 50, 200, 50),
            "quit": pygame.Rect(WINDOW_WIDTH//2 + 10, WINDOW_HEIGHT//2 + 50, 200, 50),
        }
        self.menu_rects = {}; self.pulse_start_time = 0

    def t(self, key): return i18n.t(key)
    def world_to_screen(self, wx, wy, cam_x, cam_y):
        return int((wx * GRID_SIZE) - cam_x + (WINDOW_WIDTH // 2)), int((wy * GRID_SIZE) - cam_y + (WINDOW_HEIGHT // 2))

    def draw_game(self, state):
        if self.state == "CONNECT": self.draw_connect(); return
        if self.state == "LOGIN": self.draw_login(); return
        if self.state == "MENU": self.draw_menu(); return
        if self.state == "CONFIG": self.draw_config(); return
        self.screen.fill(COLOR_BG)
        if state.phase == 0 and self.state != "PAUSE": self.draw_lobby(state); return
        if getattr(state, "is_extracted", False) and self.spectator_mode:
            cam_x, cam_y = self.cam_offset[0] * GRID_SIZE, self.cam_offset[1] * GRID_SIZE
        else:
            cam_x, cam_y = state.my_pos[0] * GRID_SIZE, state.my_pos[1] * GRID_SIZE
            self.cam_offset = [state.my_pos[0], state.my_pos[1]]
        if state.map_tiles:
            s_c = max(0, int(self.cam_offset[0] - 22)); e_c = int(self.cam_offset[0] + 22)
            s_r = max(0, int(self.cam_offset[1] - 17)); e_r = int(self.cam_offset[1] + 17)
            for y in range(s_r, min(len(state.map_tiles), e_r)):
                for x in range(s_c, min(len(state.map_tiles[0]), e_c)):
                    sx, sy = self.world_to_screen(x, y, cam_x, cam_y)
                    rect = (sx, sy, GRID_SIZE, GRID_SIZE)
                    pygame.draw.rect(self.screen, COLOR_GRID, rect, 1)
                    if state.map_tiles[y][x] == 1:
                        pygame.draw.rect(self.screen, COLOR_WALL, rect); pygame.draw.rect(self.screen, COLOR_WALL_EDGE, rect, 1)
        half = GRID_SIZE // 2
        for ent in state.entities:
            sx, sy = self.world_to_screen(ent["pos"]["x"], ent["pos"]["y"], cam_x, cam_y)
            tl = (sx - half, sy - half)
            if ent["type"] == "ITEM_DROP":
                if "ITEM_DROP" in self.assets: self.screen.blit(self.assets["ITEM_DROP"], tl)
                else: self.draw_text_centered("📦", sx, sy, (255, 255, 0))
            elif ent["type"] == "SUPPLY_DROP":
                pygame.draw.circle(self.screen, COLOR_SUPPLY_DROP, (sx, sy), GRID_SIZE, 1)
                if "SUPPLY_DROP" in self.assets: self.screen.blit(self.assets["SUPPLY_DROP"], tl)
                else: self.draw_text_centered("🎁", sx, sy, COLOR_SUPPLY_DROP)
            elif ent["type"] == "MERCHANT":
                if "MERCHANT" in self.assets: self.screen.blit(self.assets["MERCHANT"], tl)
                else: self.draw_text_centered("💰", sx, sy, (255, 215, 0))
            elif ent["type"] == "MOTOR":
                color = COLOR_MOTOR_DONE if ent["state"] == 2 else COLOR_MOTOR_ACTIVE
                pygame.draw.circle(self.screen, color, (sx, sy), half, 0); self.draw_text_centered("M", sx, sy, (0, 0, 0))
                if ent["state"] != 2:
                    ex = ent.get("extra", {})
                    if ex: self.draw_bar(tl[0], tl[1]-10, ex.get("progress", 0), ex.get("max_progress", 100), (0, 255, 255))
            elif ent["type"] == "EXIT":
                pygame.draw.rect(self.screen, COLOR_EXIT, (tl[0], tl[1], GRID_SIZE, GRID_SIZE), 0); self.draw_text_centered("E", sx, sy, (0, 0, 0))
        rd = GRID_SIZE // 4 
        for pid, p in state.players.items():
            sx, sy = self.world_to_screen(p["pos"]["x"], p["pos"]["y"], cam_x, cam_y)
            pygame.draw.circle(self.screen, COLOR_ENEMY, (sx, sy), rd); self.draw_hp_bar(sx-half, sy-half-5, p["hp"], p["max_hp"])
        if not getattr(state, "is_extracted", False):
            sx, sy = self.world_to_screen(state.my_pos[0], state.my_pos[1], cam_x, cam_y)
            pygame.draw.circle(self.screen, COLOR_SELF, (sx, sy), rd); self.draw_text_centered("ME", sx, sy-10); self.draw_hp_bar(sx-half, sy-half-5, state.my_hp, 100)
        if not self.dev_mode and not self.spectator_mode:
            self.fog_surf.fill(COLOR_FOG); pygame.draw.circle(self.fog_surf, (0,0,0,0), (WINDOW_WIDTH//2, WINDOW_HEIGHT//2), int(state.view_radius * GRID_SIZE))
            self.screen.blit(self.fog_surf, (0,0))
        self.draw_hud(state); self.draw_inventory(state); self.draw_events(state); self.draw_minimap(state)
        if state.my_hp <= 0: self.draw_death_overlay()
        if getattr(state, "is_extracted", False) and not self.spectator_mode: self.draw_spectator_overlay()
        if self.show_shop: self.draw_shop_menu(state)
        if self.state == "PAUSE":
            self.draw_pause_menu()
            if self.show_settings: self.draw_settings_menu()
            if self.show_help: self.draw_help_menu()

    def draw_connect(self):
        self.screen.fill(COLOR_BG); t = self.font.render(self.t("CONNECT_TITLE"), True, (0, 255, 255))
        self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, 200)))
        r = pygame.Rect(WINDOW_WIDTH//2 - 200, 300, 400, 40); pygame.draw.rect(self.screen, (50, 50, 60), r); pygame.draw.rect(self.screen, (0, 255, 255), r, 2)
        self.screen.blit(self.font.render(self.server_input + "|", True, (255, 255, 255)), (r.x + 10, r.y + 5))
        self.screen.blit(self.hud_font.render(self.t("ENTER_URL"), True, (150, 150, 150)), (WINDOW_WIDTH//2 - 150, 360))

    def draw_login(self):
        self.screen.fill(COLOR_BG); t = self.font.render(self.t("LOGIN_TITLE"), True, (0, 255, 255))
        self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, 200)))
        r = pygame.Rect(WINDOW_WIDTH//2 - 150, 300, 300, 40); pygame.draw.rect(self.screen, (50, 50, 60), r); pygame.draw.rect(self.screen, (0, 255, 255), r, 2)
        self.screen.blit(self.font.render(self.name_input + "|", True, (255, 255, 255)), (r.x + 10, r.y + 5))
        self.screen.blit(self.hud_font.render(self.t("ENTER_NAME"), True, (150, 150, 150)), (WINDOW_WIDTH//2 - 150, 360))
        br = pygame.Rect(WINDOW_WIDTH//2 - 60, 400, 120, 40); pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK [B]", True, (255,255,255)), (br.x+25, br.y+10)); self.login_back_rect = br

    def draw_menu(self):
        self.screen.fill(COLOR_BG); t = self.font.render(self.t("MENU_TITLE"), True, (0, 255, 255))
        self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, 100)))
        c_r = pygame.Rect(WINDOW_WIDTH//2 - 150, 250, 300, 50); j_r = pygame.Rect(WINDOW_WIDTH//2 - 150, 330, 300, 50)
        for r, txt in [(c_r, self.t("BTN_CREATE")), (j_r, self.t("BTN_JOIN"))]:
            pygame.draw.rect(self.screen, COLOR_BTN, r); pygame.draw.rect(self.screen, (0, 255, 255), r, 2)
            s = self.font.render(txt, True, (255, 255, 255)); self.screen.blit(s, s.get_rect(center=r.center))
        self.menu_rects = {"create": c_r, "join": j_r}
        br = pygame.Rect(WINDOW_WIDTH//2 - 60, 450, 120, 40); pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK [B]", True, (255,255,255)), (br.x+25, br.y+10)); self.menu_back_rect = br

    def draw_config(self):
        self.screen.fill(COLOR_BG); t = self.font.render(self.t("CONFIG_TITLE"), True, (0, 255, 255))
        self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, 50)))
        y = 150; labels = {"max_players": self.t("LBL_MAX_AGENTS"), "motors": self.t("LBL_MOTORS"), "p1_dur": self.t("LBL_SEARCH"), "p2_dur": self.t("LBL_CONFLICT")}
        for i, key in enumerate(self.config_keys):
            color = (255, 255, 0) if i == self.config_active_idx else (200, 200, 200)
            txt = f"{labels[key]} {self.config_inputs[key]}"
            if i == self.config_active_idx: txt += "|"
            self.screen.blit(self.font.render(txt, True, color), (WINDOW_WIDTH//2 - 150, y)); y += 50
        self.screen.blit(self.hud_font.render(self.t("CONFIG_HINT"), True, (150, 150, 150)), (WINDOW_WIDTH//2 - 200, 500))
        br = pygame.Rect(WINDOW_WIDTH//2 - 60, 550, 120, 40); pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK [B]", True, (255,255,255)), (br.x+25, br.y+10)); self.config_back_rect = br

    def draw_pause_menu(self):
        overlay = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA); overlay.fill((0, 0, 0, 180)); self.screen.blit(overlay, (0,0))
        pygame.draw.rect(self.screen, COLOR_MENU_BG, (WINDOW_WIDTH//2 - 150, 100, 300, 500), border_radius=10)
        t = self.font.render(self.t("PAUSE_TITLE"), True, (0, 255, 255)); self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, 150)))
        lbls = {"resume": self.t("BTN_RESUME"), "settings": self.t("BTN_SETTINGS"), "help": self.t("BTN_HELP"), "quit": self.t("BTN_QUIT")}
        for key, rect in self.pause_rects.items():
            pygame.draw.rect(self.screen, COLOR_BTN, rect, border_radius=5); pygame.draw.rect(self.screen, (0, 255, 255), rect, 1, border_radius=5)
            s = self.hud_font.render(lbls[key], True, (255, 255, 255)); self.screen.blit(s, s.get_rect(center=rect.center))

    def handle_pause_click(self, pos):
        for key, rect in self.pause_rects.items():
            if rect.collidepoint(pos): return key
        return None

    def draw_hud(self, state):
        y = 10;
        for t in [f"HP: {state.my_hp:.0f}%", f"CASH: ${state.funds}", f"POS: {int(state.my_pos[0])},{int(state.my_pos[1])}"]:
            self.screen.blit(self.hud_font.render(t, True, COLOR_HUD_TEXT), (10, y)); y += 20
        phase_map = {1: self.t("PHASE_SEARCH"), 2: self.t("PHASE_CONFLICT"), 3: self.t("PHASE_ESCAPE"), 4: self.t("PHASE_ENDED")}
        p_txt = phase_map.get(state.phase, self.t("PHASE_INIT"))
        s = self.font.render(f"{p_txt} | {int(state.time_left)}s", True, (255, 255, 0)); self.screen.blit(s, s.get_rect(center=(WINDOW_WIDTH//2, 30)))
        self.screen.blit(self.hud_font.render(self.t("HUD_CONTROLS"), True, (150, 150, 150)), (WINDOW_WIDTH - 300, WINDOW_HEIGHT - 30))

    def draw_minimap(self, state):
        pygame.draw.rect(self.screen, COLOR_RADAR_BG, self.radar_rect, border_radius=10); pygame.draw.rect(self.screen, COLOR_RADAR_BORDER, self.radar_rect, 2, border_radius=10)
        scale = 140.0 / 32.0; ox, oy = self.radar_rect.x + 5, self.radar_rect.y + 5
        for blip in state.radar_blips:
            bx, by = blip["pos"]["x"] * scale, blip["pos"]["y"] * scale
            if blip["type"] == "MOTOR": pygame.draw.circle(self.screen, (255,255,0), (int(ox+bx), int(oy+by)), 3)
            elif blip["type"] == "EXIT": pygame.draw.circle(self.screen, (0,255,0), (int(ox+bx), int(oy+by)), 4)
            elif blip["type"] == "SUPPLY_DROP": pygame.draw.rect(self.screen, (255,0,255), (ox+bx-3, oy+by-3, 6, 6))
            elif blip["type"] == "MERCHANT": pygame.draw.rect(self.screen, (255,215,0), (ox+bx-3, oy+by-3, 6, 6))
        sx, sy = state.my_pos[0] * scale, state.my_pos[1] * scale; pygame.draw.circle(self.screen, COLOR_SELF, (int(ox+sx), int(oy+sy)), 3)

    def draw_bar(self, x, y, val, max_val, color):
        pygame.draw.rect(self.screen, (50,50,50), (x, y, GRID_SIZE, 4)); pygame.draw.rect(self.screen, color, (x, y, GRID_SIZE * (val/max_val), 4))

    def draw_hp_bar(self, x, y, hp, max_hp):
        pct = max(0, min(1, hp/max_hp)); pygame.draw.rect(self.screen, (100,0,0), (x, y, GRID_SIZE, 4)); pygame.draw.rect(self.screen, (0,255,0), (x, y, GRID_SIZE * pct, 4))

    def draw_text_centered(self, text, x, y, color=(255,255,255)):
        try: s = self.font.render(text, True, color); self.screen.blit(s, s.get_rect(center=(x, y)))
        except: pass

    def draw_inventory(self, state):
        for i in range(6):
            r = pygame.Rect(300 + i*60, WINDOW_HEIGHT-70, 50, 50); pygame.draw.rect(self.screen, COLOR_INV_BG, r, border_radius=5)
            if i < len(state.my_inventory):
                name = state.my_inventory[i].get("name", "???"); self.screen.blit(self.hud_font.render(name[:3], True, (255,255,255)), (r.x+5, r.y+15))

    def draw_events(self, state):
        y = 100;
        for e in state.events[-5:]:
            msg = e.get('msg', '').replace('\x00', '')
            s = self.hud_font.render(f"> {msg}", True, (255,100,255)); self.screen.blit(s, (WINDOW_WIDTH - s.get_width() - 10, y)); y += 20

    def draw_death_overlay(self):
        s = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA); s.fill((150, 0, 0, 120)); self.screen.blit(s, (0,0))
        t = self.font.render(self.t("DEATH_TITLE"), True, (255,255,255)); self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, WINDOW_HEIGHT//2)))

    def draw_spectator_overlay(self):
        overlay = pygame.Surface((WINDOW_WIDTH, WINDOW_HEIGHT), pygame.SRCALPHA); overlay.fill((0, 0, 0, 150)); self.screen.blit(overlay, (0,0))
        pygame.draw.rect(self.screen, COLOR_MENU_BG, (WINDOW_WIDTH//2 - 250, WINDOW_HEIGHT//2 - 150, 500, 300), border_radius=10)
        t = self.font.render("EXTRACTION SUCCESSFUL", True, (0, 255, 0)); self.screen.blit(t, t.get_rect(center=(WINDOW_WIDTH//2, WINDOW_HEIGHT//2 - 80)))
        lbls = {"spectate": "FREE SPECTATE", "quit": "QUIT TO MENU"}
        for key, rect in self.results_rects.items():
            pygame.draw.rect(self.screen, COLOR_BTN, rect, border_radius=5); pygame.draw.rect(self.screen, (0, 255, 0), rect, 1, border_radius=5)
            s = self.hud_font.render(lbls[key], True, (255, 255, 255)); self.screen.blit(s, s.get_rect(center=rect.center))

    def draw_settings_menu(self):
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.settings_rect, border_radius=10); pygame.draw.rect(self.screen, (255,255,255), self.settings_rect, 2, border_radius=10)
        t = self.font.render(self.t("SETTINGS_TITLE"), True, (255,255,255)); self.screen.blit(t, (self.settings_rect.x+20, self.settings_rect.y+20))
        pygame.draw.rect(self.screen, (0,255,0) if self.dev_mode else (100,100,100), self.dev_mode_rect, 2)
        self.screen.blit(self.hud_font.render(f"{self.t('LBL_DEV_MODE')}: {'ON' if self.dev_mode else 'OFF'}", True, (255,255,255)), (self.dev_mode_rect.x+10, self.dev_mode_rect.y+5))
        pygame.draw.rect(self.screen, (0,255,255), self.lang_rect, 2)
        self.screen.blit(self.hud_font.render(f"{self.t('LBL_LANG')}", True, (255,255,255)), (self.lang_rect.x+10, self.lang_rect.y+5))
        pygame.draw.rect(self.screen, (200, 50, 50), self.back_btn_rect, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), self.back_btn_rect, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK", True, (255,255,255)), (self.back_btn_rect.x+40, self.back_btn_rect.y+10))

    def draw_help_menu(self):
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.help_rect, border_radius=10); pygame.draw.rect(self.screen, (255,255,255), self.help_rect, 2, border_radius=10)
        t = self.font.render(self.t("MANUAL_TITLE"), True, (0,255,255)); self.screen.blit(t, (self.help_rect.x+20, self.help_rect.y+20))
        for i, l in enumerate(i18n.get_list("MANUAL_LINES")): self.screen.blit(self.hud_font.render(l, True, (200,200,200)), (self.help_rect.x+30, self.help_rect.y+80+i*30))
        br = pygame.Rect(self.help_rect.centerx - 60, self.help_rect.bottom - 60, 120, 40)
        pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK", True, (255,255,255)), (br.x+40, br.y+10)); self.help_back_rect = br

    def draw_shop_menu(self, state):
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.shop_rect, border_radius=10); pygame.draw.rect(self.screen, (255,215,0), self.shop_rect, 2, border_radius=10)
        self.screen.blit(self.font.render(self.t("SHOP_TITLE"), True, (255,215,0)), (self.shop_rect.x+20, self.shop_rect.y+20))
        self.screen.blit(self.font.render(f"{self.t('SHOP_FUNDS')} ${state.funds}", True, (0, 255, 0)), (self.shop_rect.x + 200, self.shop_rect.y + 20))
        items = []
        if state.phase == 1: items = [(self.t("ITEM_STUN"), "WPN_SHOCK", 100), (self.t("ITEM_MED"), "SURV_MEDKIT", 50), (self.t("ITEM_SCAN"), "RECON_RADAR", 150)]
        elif state.phase == 2: items = [(self.t("ITEM_TASER"), "WPN_SHOCK_T2", 200), (self.t("ITEM_MED_PLUS"), "SURV_MEDKIT_T2", 100), (self.t("ITEM_SCAN_PRO"), "RECON_RADAR_T2", 300)]
        elif state.phase >= 3: items = [(self.t("ITEM_VOLT"), "WPN_SHOCK_T3", 350), (self.t("ITEM_SCAN_GLOBAL"), "RECON_RADAR_T3", 500), (self.t("ITEM_MED_PLUS"), "SURV_MEDKIT_T2", 100)]
        y = 70;
        for n, pid, c in items:
            color = (255, 255, 255) if state.funds >= c else (100, 100, 100)
            self.screen.blit(self.hud_font.render(f"{n} - ${c} [{items.index((n,pid,c))+1}]", True, color), (self.shop_rect.x+30, self.shop_rect.y+y)); y += 40
        self.screen.blit(self.hud_font.render(self.t("SHOP_HINT"), True, (150, 150, 150)), (self.shop_rect.x+30, self.shop_rect.y+350))
        br = pygame.Rect(self.shop_rect.centerx - 60, self.shop_rect.bottom - 50, 120, 40)
        pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK [B]", True, (255,255,255)), (br.x+25, br.y+10)); self.shop_back_rect = br

    def draw_ui_buttons(self): pass
    def handle_click(self, pos):
        if self.state == "LOGIN":
            if hasattr(self, 'login_back_rect') and self.login_back_rect.collidepoint(pos): self.state = "CONNECT"; return True
        if self.state == "MENU":
            if hasattr(self, 'menu_back_rect') and self.menu_back_rect.collidepoint(pos): self.state = "LOGIN"; return True
        if self.state == "CONFIG":
            if hasattr(self, 'config_back_rect') and self.config_back_rect.collidepoint(pos): self.state = "MENU"; return True
        
        # Results Click
        if hasattr(self, 'results_rects'):
            if self.results_rects["spectate"].collidepoint(pos):
                self.spectator_mode = True
                return True
            if self.results_rects["quit"].collidepoint(pos):
                self.state = "MENU"
                self.spectator_mode = False
                return True

        if self.state == "PAUSE":
            if self.show_settings:
                if self.dev_mode_rect.collidepoint(pos): self.dev_mode = not self.dev_mode; return True
                if self.lang_rect.collidepoint(pos): i18n.set_lang("en" if i18n.lang == "zh" else "zh"); return True
                if self.back_btn_rect.collidepoint(pos): self.show_settings = False; return True
            elif self.show_help:
                if hasattr(self, 'help_back_rect') and self.help_back_rect.collidepoint(pos): self.show_help = False; return True
                if self.back_btn_rect.collidepoint(pos): self.show_help = False; return True
        if self.show_shop:
            if hasattr(self, 'shop_back_rect') and self.shop_back_rect.collidepoint(pos): self.show_shop = False; return True
        return False
    def draw_system_clock(self): pass
    def draw_lobby(self, state):
        self.screen.blit(self.font.render(self.t("LOBBY_HINT"), True, (255,255,255)), (200, 200))
        br = pygame.Rect(WINDOW_WIDTH//2 - 60, 400, 120, 40); pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK [B]", True, (255,255,255)), (br.x+25, br.y+10)); self.lobby_back_rect = br