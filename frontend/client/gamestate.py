class GameState:
    def __init__(self):
        self.map_width = 32
        self.map_height = 32
        self.map_tiles = []
        self.self_id = None
        self.players = {} 
        self.entities = [] 
        
        # Self State
        self.my_pos = [0, 0]
        self.my_name = ""
        self.my_hp = 100
        self.my_max_hp = 100
        self.my_armor = 0
        self.my_max_armor = 50
        self.my_kills = 0
        self.my_ammo_type = ""
        self.my_ammo_count = 0
        self.view_radius = 5.0
        self.my_inventory = []
        self.inventory_cap = 6
        self.funds = 0
        self.is_extracted = False

        # Merchant
        self.shop_stock = []
        self.shop_prices = []
        self.shop_types = []

        # Ephemeral UI toast (from server self.client_msg)
        self.toast_msg = ""
        self.toast_until = 0.0
        
        # Global State
        self.phase = 0 # Default Init
        self.time_left = 0
        self.events = []
        self.radar_blips = []
        self.sound_events = []
        
        # Client State
        self.config = {}
        self.tactic_chosen = False

    def update_from_server(self, payload):
        # Global
        self.phase = payload.get("phase", 0)
        self.time_left = payload.get("time_left", 0)
        self.total_kills = payload.get("total_kills", 0)
        evts = payload.get("events")
        self.events = evts if evts is not None else []
        
        blips = payload.get("radar_blips")
        self.radar_blips = blips if blips is not None else []
        
        snd = payload.get("sound")
        if snd:
            self.sound_events = snd.get("events", [])
        else:
            self.sound_events = []

        if "self" in payload:
            s = payload["self"]
            self.self_id = s.get("session_id")
            nm = s.get("name")
            if isinstance(nm, str):
                self.my_name = nm
            self.my_pos = [s["pos"]["x"], s["pos"]["y"]]
            self.my_hp = s["hp"]
            self.my_max_hp = s.get("max_hp", 100)
            self.my_armor = s.get("armor", 0)
            self.my_max_armor = s.get("max_armor", 0)
            self.my_kills = s.get("kills", 0)
            self.my_ammo_type = s.get("ammo_type", "")
            self.my_ammo_count = s.get("ammo_count", 0)
            self.view_radius = s["view_radius"]
            self.funds = s.get("funds", 0)
            self.is_extracted = s.get("is_extracted", False)
            inv = s.get("inventory")
            self.my_inventory = inv if inv is not None else []
            self.inventory_cap = s.get("inventory_cap", 6)
            ss = s.get("shop_stock")
            self.shop_stock = ss if ss is not None else []

            sp = s.get("shop_prices")
            self.shop_prices = sp if isinstance(sp, list) else []

            st = s.get("shop_types")
            self.shop_types = st if isinstance(st, list) else []

            msg = s.get("client_msg")
            if isinstance(msg, str) and msg and msg != self.toast_msg:
                # Set/update toast; renderer will display it briefly.
                try:
                    import time as _time
                    self.toast_until = float(_time.time()) + 3.0
                except Exception:
                    self.toast_until = 0.0
                self.toast_msg = msg

        if "vision" in payload:
            self.players = {}
            for p in payload["vision"]["players"]:
                self.players[p["session_id"]] = p
            
            self.entities = payload["vision"]["entities"]