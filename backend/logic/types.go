package logic

// Vector2 represents a 2D position
type Vector2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// EntityType enum
const (
	EntityTypePlayer   = "PLAYER"
	EntityTypeItemDrop = "ITEM_DROP"
	EntityTypeMotor    = "MOTOR"
	EntityTypeExit     = "EXIT"
)

// Entity represents any object in the world
type Entity struct {
	UID   string  `json:"uid"`
	Type  string  `json:"type"`
	Pos   Vector2 `json:"pos"`
	State int     `json:"state"` // Generic state (e.g., progress)
}

// Player represents a connected user in the game simulation
type Player struct {
	SessionID  string  `json:"session_id"`
	Pos        Vector2 `json:"pos"`
	HP         float64 `json:"hp"`
	MaxHP      float64 `json:"max_hp"`
	MoveSpeed  float64 `json:"move_speed"`
	ViewRadius float64 `json:"view_radius"`
	IsAlive    bool    `json:"is_alive"`
	Tactic     string  `json:"tactic"`
	
	// Internal state (not always synced)
	Velocity    Vector2 `json:"-"`
	TargetDir   Vector2 `json:"-"` // From client input
	Inventory   []Item  `json:"-"` // Simplified for MVP
}

// Item simplified
type Item struct {
	UID  string `json:"uid"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Config structs (mirrors game_config.json)
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
}
