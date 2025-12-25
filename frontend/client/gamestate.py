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
        self.my_hp = 100
        self.view_radius = 5.0
        self.my_inventory = []
        self.funds = 0
        self.is_extracted = False
        
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
            self.my_pos = [s["pos"]["x"], s["pos"]["y"]]
            self.my_hp = s["hp"]
            self.view_radius = s["view_radius"]
            self.funds = s.get("funds", 0)
            self.is_extracted = s.get("is_extracted", False)
            inv = s.get("inventory")
            self.my_inventory = inv if inv is not None else []

        if "vision" in payload:
            self.players = {}
            for p in payload["vision"]["players"]:
                self.players[p["session_id"]] = p
            
            self.entities = payload["vision"]["entities"]