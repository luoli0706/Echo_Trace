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

    def update_from_server(self, payload):
        if "self" in payload:
            s = payload["self"]
            self.self_id = s.get("session_id")
            self.my_pos = [s["pos"]["x"], s["pos"]["y"]]
            self.my_hp = s["hp"]
            self.view_radius = s["view_radius"]

        if "vision" in payload:
            self.players = {}
            for p in payload["vision"]["players"]:
                self.players[p["session_id"]] = p
