package logic

type Vector2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

const (
	EntityTypePlayer   = "PLAYER"
	EntityTypeItemDrop = "ITEM_DROP"
	EntityTypeMotor    = "MOTOR"
	EntityTypeExit     = "EXIT"
)

type Entity struct {
	UID   string      `json:"uid"`
	Type  string      `json:"type"`
	Pos   Vector2     `json:"pos"`
	State int         `json:"state"`
	Extra interface{} `json:"extra,omitempty"` 
}

type Player struct {
	SessionID  string  `json:"session_id"`
	Pos        Vector2 `json:"pos"`
	HP         float64 `json:"hp"`
	MaxHP      float64 `json:"max_hp"`
	MoveSpeed  float64 `json:"move_speed"`
	ViewRadius float64 `json:"view_radius"`
	IsAlive    bool    `json:"is_alive"`
	Tactic     string  `json:"tactic"`
	
	Velocity   Vector2 `json:"-"`
	TargetDir  Vector2 `json:"-"`
	Inventory  []Item  `json:"inventory"`
}

type Item struct {
	UID     string `json:"uid"`
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	MaxUses int    `json:"max_uses"`
}

type GameConfig struct {
	Server struct {
		TickRateMs      int `json:"tick_rate_ms"`
		MaxPlayers      int `json:"max_players_per_room"`
	} `json:"server"`
	Map struct {
		Width       int     `json:"width"`
		Height      int     `json:"height"`
		AOIGridSize int     `json:"aoi_grid_size"`
		WallDensity float64 `json:"wall_density"`
	} `json:"map"`
	Gameplay struct {
		BaseMoveSpeed  float64 `json:"base_move_speed"`
		BaseViewRadius float64 `json:"base_view_radius"`
	} `json:"gameplay"`
	Phases struct {
		Phase1 struct { Duration int `json:"duration_sec"` } `json:"phase_1_search"`
		Phase2 struct { Duration int `json:"duration_sec"` } `json:"phase_2_conflict"`
	} `json:"phases"`
}
