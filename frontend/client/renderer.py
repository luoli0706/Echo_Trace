import pygame
import math, time, os
from datetime import datetime
from client.config import *
from client.i18n import i18n
from client.item_manual import CATEGORY_ORDER, get_item_abbr, get_item_name, get_item_use

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
        self.show_shop = self.dev_mode = self.spectator_mode = False
        # Pause UI routing stack: ["root" -> "settings"/"help"/"item_manual"].
        self.pause_route = []
        # Item manual scroll state
        self.item_manual_scroll = 0
        self.item_manual_content_height = 0
        self.mouse_sensitivity = 1.0
        self.look_angle = 0.0  # radians
        self.fov_degrees = 90.0
        self.fov_ray_count = 120
        # Rendering policy:
        # - Map tiles (including walls) are always rendered.
        # - Fog keeps non-FOV area fully black.
        # - Inside FOV wedge, map remains fully visible (not blocked by walls).
        # - World entities are visible ONLY if inside wedge AND not blocked by walls.
        self.fov_blocked_by_walls = False
        self.hide_world_entities = False
        self.cam_offset = [0, 0]
        self.settings_rect = pygame.Rect(WINDOW_WIDTH//2 - 150, WINDOW_HEIGHT//2 - 150, 300, 300)
        self.help_rect = pygame.Rect(WINDOW_WIDTH//2 - 300, WINDOW_HEIGHT//2 - 250, 600, 500)
        self.shop_rect = pygame.Rect(WINDOW_WIDTH//2 - 200, WINDOW_HEIGHT//2 - 200, 400, 400)
        self.dev_mode_rect = pygame.Rect(WINDOW_WIDTH//2 - 120, WINDOW_HEIGHT//2 + 50, 240, 30)
        self.lang_rect = pygame.Rect(WINDOW_WIDTH//2 - 120, WINDOW_HEIGHT//2 + 90, 240, 30)
        self.sens_minus_rect = pygame.Rect(WINDOW_WIDTH//2 - 120, WINDOW_HEIGHT//2 + 130, 40, 30)
        self.sens_plus_rect = pygame.Rect(WINDOW_WIDTH//2 + 80, WINDOW_HEIGHT//2 + 130, 40, 30)
        self.sens_value_rect = pygame.Rect(WINDOW_WIDTH//2 - 70, WINDOW_HEIGHT//2 + 130, 140, 30)
        self.back_btn_rect = pygame.Rect(WINDOW_WIDTH//2 - 60, WINDOW_HEIGHT//2 + 200, 120, 40)
        self.radar_rect = pygame.Rect(WINDOW_WIDTH - 160, WINDOW_HEIGHT - 160, 150, 150)
        self.pause_rects = {
            "resume": pygame.Rect(WINDOW_WIDTH//2 - 100, 200, 200, 50),
            "settings": pygame.Rect(WINDOW_WIDTH//2 - 100, 270, 200, 50),
            "item_manual": pygame.Rect(WINDOW_WIDTH//2 - 100, 340, 200, 50),
            "help": pygame.Rect(WINDOW_WIDTH//2 - 100, 410, 200, 50),
            "quit": pygame.Rect(WINDOW_WIDTH//2 - 100, 480, 200, 50),
        }
        self.results_rects = {
            "spectate": pygame.Rect(WINDOW_WIDTH//2 - 210, WINDOW_HEIGHT//2 + 50, 200, 50),
            "quit": pygame.Rect(WINDOW_WIDTH//2 + 10, WINDOW_HEIGHT//2 + 50, 200, 50),
        }
        self.menu_rects = {}; self.pulse_start_time = 0

    def pause_open(self):
        if not self.pause_route:
            self.pause_route = ["root"]

    def pause_view(self):
        if not self.pause_route:
            return "root"
        return self.pause_route[-1]

    def pause_push(self, view: str):
        self.pause_open()
        if view and view != self.pause_view():
            self.pause_route.append(view)
        if view == "item_manual":
            self.item_manual_scroll = 0

    def pause_pop(self):
        if len(self.pause_route) > 1:
            self.pause_route.pop()
        if self.pause_view() != "item_manual":
            self.item_manual_scroll = 0

    def t(self, key): return i18n.t(key)
    def world_to_screen(self, wx, wy, cam_x, cam_y):
        return int((wx * GRID_SIZE) - cam_x + (WINDOW_WIDTH // 2)), int((wy * GRID_SIZE) - cam_y + (WINDOW_HEIGHT // 2))

    def _wrap_angle(self, a):
        # Normalize to [-pi, pi)
        while a >= math.pi:
            a -= 2*math.pi
        while a < -math.pi:
            a += 2*math.pi
        return a

    def update_look_from_mouse(self, mouse_pos, dt, state):
        # Only update in active gameplay (not in dev/spectator overlays)
        if self.state != "GAME":
            return
        if getattr(state, "is_extracted", False) and self.spectator_mode:
            return
        if self.show_shop or self.state == "PAUSE":
            return

        mx, my = mouse_pos
        cx, cy = WINDOW_WIDTH // 2, WINDOW_HEIGHT // 2
        dx = mx - cx
        dy = my - cy
        if dx == 0 and dy == 0:
            return

        target = math.atan2(dy, dx)
        diff = self._wrap_angle(target - self.look_angle)
        # Sensitivity acts as response speed (higher -> faster snap)
        alpha = max(0.0, min(1.0, dt * 12.0 * float(self.mouse_sensitivity)))
        self.look_angle = self._wrap_angle(self.look_angle + diff * alpha)

    def get_look_dir(self):
        return (math.cos(self.look_angle), math.sin(self.look_angle))

    def _raycast_to_wall(self, origin_w, dir_w, max_dist, tiles):
        # Simple incremental raymarch in world space against wall tiles.
        # Map is small (32x32), so this is fast enough and stable.
        ox, oy = origin_w
        dx, dy = dir_w
        step = 0.12  # world units (~1/8 tile)
        dist = 0.0
        lastx, lasty = ox, oy
        h = len(tiles)
        w = len(tiles[0]) if h > 0 else 0
        while dist <= max_dist:
            x = ox + dx * dist
            y = oy + dy * dist
            gx = int(x)
            gy = int(y)
            if gx < 0 or gy < 0 or gx >= w or gy >= h:
                return lastx, lasty
            if tiles[gy][gx] == 1:
                return lastx, lasty
            lastx, lasty = x, y
            dist += step
        return lastx, lasty

    def _has_line_of_sight(self, origin_w, target_w, tiles):
        if not tiles:
            return True
        ox, oy = origin_w
        tx, ty = target_w
        dx = tx - ox
        dy = ty - oy
        dist = math.sqrt(dx*dx + dy*dy)
        if dist <= 1e-6:
            return True
        inv = 1.0 / dist
        ux, uy = dx * inv, dy * inv
        step = 0.12
        d = 0.0
        h = len(tiles)
        w = len(tiles[0]) if h > 0 else 0
        while d <= dist:
            x = ox + ux * d
            y = oy + uy * d
            gx = int(x)
            gy = int(y)
            if gx < 0 or gy < 0 or gx >= w or gy >= h:
                return False
            if tiles[gy][gx] == 1:
                return False
            d += step
        return True

    def _is_world_pos_visible(self, state, wx, wy):
        # Visibility rule for entities/blips: within view radius, inside FOV wedge,
        # and NOT occluded by walls.
        ox, oy = state.my_pos[0], state.my_pos[1]
        dx = wx - ox
        dy = wy - oy
        rr = float(state.view_radius)
        if dx*dx + dy*dy > rr*rr:
            return False
        # Cone check
        half = math.radians(self.fov_degrees) / 2.0
        cos_half = math.cos(half)
        ldx, ldy = self.get_look_dir()
        dlen2 = dx*dx + dy*dy
        if dlen2 <= 1e-9:
            in_cone = True
        else:
            inv = 1.0 / math.sqrt(dlen2)
            ux, uy = dx * inv, dy * inv
            in_cone = (ux * ldx + uy * ldy) >= cos_half
        if not in_cone:
            return False
        return self._has_line_of_sight((ox, oy), (wx, wy), state.map_tiles)

    def _compute_fov_polygon_screen(self, state):
        # Returns a list of screen points forming a polygon fan (center + ray endpoints)
        cx, cy = WINDOW_WIDTH // 2, WINDOW_HEIGHT // 2
        half = math.radians(self.fov_degrees) / 2.0
        rays = max(12, int(self.fov_ray_count))
        pts = [(cx, cy)]

        tiles = state.map_tiles
        origin_w = (state.my_pos[0], state.my_pos[1])
        max_dist = float(state.view_radius)

        for i in range(rays):
            t = 0.0 if rays == 1 else (i / (rays - 1))
            ang = self.look_angle - half + (2.0 * half * t)
            dir_w = (math.cos(ang), math.sin(ang))
            if self.fov_blocked_by_walls and tiles:
                ex, ey = self._raycast_to_wall(origin_w, dir_w, max_dist, tiles)
                dist = math.sqrt((ex-origin_w[0])**2 + (ey-origin_w[1])**2)
            else:
                dist = max_dist
            sx = int(cx + dir_w[0] * dist * GRID_SIZE)
            sy = int(cy + dir_w[1] * dist * GRID_SIZE)
            pts.append((sx, sy))

        return pts

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
            if self.hide_world_entities:
                continue
            if not self._is_world_pos_visible(state, ent["pos"]["x"], ent["pos"]["y"]):
                continue
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
            if self.hide_world_entities:
                continue
            if not self._is_world_pos_visible(state, p["pos"]["x"], p["pos"]["y"]):
                continue
            sx, sy = self.world_to_screen(p["pos"]["x"], p["pos"]["y"], cam_x, cam_y)
            pygame.draw.circle(self.screen, COLOR_ENEMY, (sx, sy), rd); self.draw_hp_bar(sx-half, sy-half-5, p["hp"], p["max_hp"])
        if not getattr(state, "is_extracted", False):
            sx, sy = self.world_to_screen(state.my_pos[0], state.my_pos[1], cam_x, cam_y)
            pygame.draw.circle(self.screen, COLOR_SELF, (sx, sy), rd); self.draw_text_centered("ME", sx, sy-10); self.draw_hp_bar(sx-half, sy-half-5, state.my_hp, 100)
        if not self.dev_mode and not self.spectator_mode:
            # Keep outside-FOV fully black; inside FOV wedge fully visible.
            self.fog_surf.fill((0, 0, 0, 255))
            poly = self._compute_fov_polygon_screen(state)
            if len(poly) >= 3:
                pygame.draw.polygon(self.fog_surf, (0, 0, 0, 0), poly)
            else:
                pygame.draw.circle(self.fog_surf, (0,0,0,0), (WINDOW_WIDTH//2, WINDOW_HEIGHT//2), int(state.view_radius * GRID_SIZE))
            self.screen.blit(self.fog_surf, (0,0))
        self.draw_hud(state); self.draw_inventory(state); self.draw_events(state); self.draw_minimap(state)
        if state.my_hp <= 0: self.draw_death_overlay()
        if getattr(state, "is_extracted", False) and not self.spectator_mode: self.draw_spectator_overlay()
        if self.show_shop: self.draw_shop_menu(state)
        if self.state == "PAUSE":
            self.draw_pause_menu()
            view = self.pause_view()
            if view == "settings":
                self.draw_settings_menu()
            elif view == "help":
                self.draw_help_menu()
            elif view == "item_manual":
                self.draw_item_manual_menu()

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
        lbls = {
            "resume": self.t("BTN_RESUME"),
            "settings": self.t("BTN_SETTINGS"),
            "item_manual": self.t("BTN_ITEM_MANUAL"),
            "help": self.t("BTN_HELP"),
            "quit": self.t("BTN_QUIT"),
        }
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
        if not getattr(self, "hide_world_entities", False):
            for blip in state.radar_blips:
                bxw, byw = blip["pos"]["x"], blip["pos"]["y"]
                if not self._is_world_pos_visible(state, bxw, byw):
                    continue
                bx, by = bxw * scale, byw * scale
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
        cap = int(getattr(state, "inventory_cap", 6) or 6)
        # If a timed buff expires and cap shrinks, keep showing overflow items so players can still see/sell/drop them.
        cap = max(cap, len(state.my_inventory))
        for i in range(cap):
            r = pygame.Rect(300 + i*60, WINDOW_HEIGHT-70, 50, 50); pygame.draw.rect(self.screen, COLOR_INV_BG, r, border_radius=5)
            if i < len(state.my_inventory):
                iid = state.my_inventory[i].get("id") or state.my_inventory[i].get("ID")
                if iid:
                    ab = get_item_abbr(iid)
                else:
                    name = state.my_inventory[i].get("name", "???")
                    ab = name[:3]
                self.screen.blit(self.hud_font.render(ab, True, (255,255,255)), (r.x+5, r.y+15))

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

        # Mouse Sensitivity
        pygame.draw.rect(self.screen, (0,255,255), self.sens_minus_rect, 2)
        pygame.draw.rect(self.screen, (0,255,255), self.sens_plus_rect, 2)
        pygame.draw.rect(self.screen, (0,255,255), self.sens_value_rect, 2)
        self.screen.blit(self.hud_font.render("-", True, (255,255,255)), (self.sens_minus_rect.x+14, self.sens_minus_rect.y+5))
        self.screen.blit(self.hud_font.render("+", True, (255,255,255)), (self.sens_plus_rect.x+14, self.sens_plus_rect.y+5))
        sens_txt = f"{self.t('LBL_MOUSE_SENS')}: {self.mouse_sensitivity:.1f}"
        self.screen.blit(self.hud_font.render(sens_txt, True, (255,255,255)), (self.sens_value_rect.x+10, self.sens_value_rect.y+5))

        pygame.draw.rect(self.screen, (200, 50, 50), self.back_btn_rect, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), self.back_btn_rect, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK", True, (255,255,255)), (self.back_btn_rect.x+40, self.back_btn_rect.y+10))

    def draw_help_menu(self):
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.help_rect, border_radius=10); pygame.draw.rect(self.screen, (255,255,255), self.help_rect, 2, border_radius=10)
        t = self.font.render(self.t("MANUAL_TITLE"), True, (0,255,255)); self.screen.blit(t, (self.help_rect.x+20, self.help_rect.y+20))
        for i, l in enumerate(i18n.get_list("MANUAL_LINES")): self.screen.blit(self.hud_font.render(l, True, (200,200,200)), (self.help_rect.x+30, self.help_rect.y+80+i*30))
        br = pygame.Rect(self.help_rect.centerx - 60, self.help_rect.bottom - 60, 120, 40)
        pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK", True, (255,255,255)), (br.x+40, br.y+10)); self.help_back_rect = br

    def draw_item_manual_menu(self):
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.help_rect, border_radius=10); pygame.draw.rect(self.screen, (255,255,255), self.help_rect, 2, border_radius=10)
        t = self.font.render(self.t("ITEM_MANUAL_TITLE"), True, (0,255,255)); self.screen.blit(t, (self.help_rect.x+20, self.help_rect.y+20))

        # Scrollable content area
        content_rect = pygame.Rect(self.help_rect.x + 20, self.help_rect.y + 70, self.help_rect.width - 40, self.help_rect.height - 140)
        x0 = content_rect.x
        max_w = content_rect.width

        def wrap_lines(text: str):
            # CJK-friendly wrapping: wrap by character width.
            lines = []
            for para in str(text).split("\n"):
                if para == "":
                    lines.append("")
                    continue
                cur = ""
                for ch in para:
                    test = cur + ch
                    if cur and self.hud_font.size(test)[0] > max_w:
                        lines.append(cur)
                        cur = ch
                    else:
                        cur = test
                if cur:
                    lines.append(cur)
            return lines

        def measure_content_height():
            yy = content_rect.y
            for cat, ids in CATEGORY_ORDER:
                yy += 22
                for iid in ids:
                    ab = get_item_abbr(iid)
                    nm = get_item_name(iid)
                    use = get_item_use(iid)
                    for _ in wrap_lines(f"{ab}  {nm} ({iid})"):
                        yy += 22
                    if use:
                        for _ in wrap_lines(f"- {use}"):
                            yy += 22
                    yy += 6
                yy += 8
            return max(0, yy - content_rect.y)

        self.item_manual_content_height = measure_content_height()
        max_scroll = max(0, self.item_manual_content_height - content_rect.height)
        if self.item_manual_scroll < 0:
            self.item_manual_scroll = 0
        if self.item_manual_scroll > max_scroll:
            self.item_manual_scroll = max_scroll

        prev_clip = self.screen.get_clip()
        self.screen.set_clip(content_rect)

        y = content_rect.y - int(self.item_manual_scroll)
        for cat, ids in CATEGORY_ORDER:
            self.screen.blit(self.hud_font.render(f"[{cat}]", True, (255,215,0)), (x0, y)); y += 22
            for iid in ids:
                ab = get_item_abbr(iid)
                nm = get_item_name(iid)
                use = get_item_use(iid)
                for ln in wrap_lines(f"{ab}  {nm} ({iid})"):
                    self.screen.blit(self.hud_font.render(ln, True, (255,255,255)), (x0, y))
                    y += 22
                if use:
                    for ln in wrap_lines(f"- {use}"):
                        self.screen.blit(self.hud_font.render(ln, True, (180,180,180)), (x0, y))
                        y += 22
                y += 6
            y += 8

        self.screen.set_clip(prev_clip)

        br = pygame.Rect(self.help_rect.centerx - 60, self.help_rect.bottom - 60, 120, 40)
        pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK", True, (255,255,255)), (br.x+40, br.y+10)); self.item_manual_back_rect = br

    def scroll_item_manual(self, delta_px: int):
        if self.pause_view() != "item_manual":
            return
        self.item_manual_scroll += int(delta_px)

    def draw_shop_menu(self, state):
        pygame.draw.rect(self.screen, COLOR_MENU_BG, self.shop_rect, border_radius=10); pygame.draw.rect(self.screen, (255,215,0), self.shop_rect, 2, border_radius=10)
        self.screen.blit(self.font.render(self.t("SHOP_TITLE"), True, (255,215,0)), (self.shop_rect.x+20, self.shop_rect.y+20))
        self.screen.blit(self.font.render(f"{self.t('SHOP_FUNDS')} ${state.funds}", True, (0, 255, 0)), (self.shop_rect.x + 200, self.shop_rect.y + 20))
        items = []
        stock = getattr(state, "shop_stock", []) or []
        for iid in stock[:6]:
            items.append((f"{get_item_abbr(iid)} {get_item_name(iid)}", iid))
        y = 70;
        for idx, it in enumerate(items):
            n, pid = it
            self.screen.blit(self.hud_font.render(f"{idx+1}. {n}", True, (255,255,255)), (self.shop_rect.x+30, self.shop_rect.y+y)); y += 34
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
            view = self.pause_view()
            if view == "settings":
                if self.dev_mode_rect.collidepoint(pos): self.dev_mode = not self.dev_mode; return True
                if self.lang_rect.collidepoint(pos): i18n.set_lang("en" if i18n.lang == "zh" else "zh"); return True
                if self.sens_minus_rect.collidepoint(pos):
                    self.mouse_sensitivity = max(0.1, round(self.mouse_sensitivity - 0.1, 1)); return True
                if self.sens_plus_rect.collidepoint(pos):
                    self.mouse_sensitivity = min(5.0, round(self.mouse_sensitivity + 0.1, 1)); return True
                if self.back_btn_rect.collidepoint(pos): self.pause_pop(); return True
            elif view == "help":
                if hasattr(self, 'help_back_rect') and self.help_back_rect.collidepoint(pos): self.pause_pop(); return True
                if self.back_btn_rect.collidepoint(pos): self.pause_pop(); return True
            elif view == "item_manual":
                if hasattr(self, 'item_manual_back_rect') and self.item_manual_back_rect.collidepoint(pos): self.pause_pop(); return True
        if self.show_shop:
            if hasattr(self, 'shop_back_rect') and self.shop_back_rect.collidepoint(pos): self.show_shop = False; return True
        return False
    def draw_system_clock(self): pass
    def draw_lobby(self, state):
        self.screen.blit(self.font.render(self.t("LOBBY_HINT"), True, (255,255,255)), (200, 200))
        br = pygame.Rect(WINDOW_WIDTH//2 - 60, 400, 120, 40); pygame.draw.rect(self.screen, (200, 50, 50), br, border_radius=5); pygame.draw.rect(self.screen, (255, 255, 255), br, 2, border_radius=5)
        self.screen.blit(self.hud_font.render("BACK [B]", True, (255,255,255)), (br.x+25, br.y+10)); self.lobby_back_rect = br