package logic

import "time"

type Vector2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

const (
	EntityTypePlayer     = "PLAYER"
	EntityTypeItemDrop   = "ITEM_DROP"
	EntityTypeMotor      = "MOTOR"
	EntityTypeExit       = "EXIT"
	EntityTypeSupplyDrop = "SUPPLY_DROP"
	EntityTypeMerchant   = "MERCHANT"
)

type Entity struct {
	UID   string      `json:"uid"`
	Type  string      `json:"type"`
	Pos   Vector2     `json:"pos"`
	State int         `json:"state"` // For Motors: 0=Inactive, 1=Active, 2=Done
	Extra interface{} `json:"extra,omitempty"`
}

type MotorData struct {
	Progress    float64 `json:"progress"` // 0.0 to 100.0
	MaxProgress float64 `json:"max_progress"`
}

type SupplyDropData struct {
	Funds int    `json:"funds"`
	Items []Item `json:"items"`
}

type Player struct {
	SessionID string  `json:"session_id"`
	Name      string  `json:"name"`
	Pos       Vector2 `json:"pos"`
	// LookDir is the player's facing / vision direction as a unit vector.
	// It is used for AOI / Fog of War cone visibility.
	LookDir    Vector2 `json:"look_dir"`
	HP         float64 `json:"hp"`
	MaxHP      float64 `json:"max_hp"`
	MoveSpeed  float64 `json:"move_speed"`
	ViewRadius float64 `json:"view_radius"`
	HearRadius float64 `json:"hear_radius"`
	IsAlive    bool    `json:"is_alive"`
	Tactic     string  `json:"tactic"`

	InventoryCap int `json:"inventory_cap"`

	MaxWeight float64 `json:"max_weight"`
	Weight    float64 `json:"weight"`
	Funds     int     `json:"funds"`

	Velocity                 Vector2  `json:"-"`
	TargetDir                Vector2  `json:"-"`
	Inventory                []Item   `json:"inventory"`
	ShopStock                []string `json:"shop_stock"`
	ShopFreeRefreshUsedPhase int      `json:"-"`

	// Timed buffs (server-authoritative). These are not serialized to clients.
	BuffSpeedMult            float64   `json:"-"`
	BuffSpeedUntil           time.Time `json:"-"`
	BuffViewBonus            float64   `json:"-"`
	BuffViewUntil            time.Time `json:"-"`
	BuffHearBonus            float64   `json:"-"`
	BuffHearUntil            time.Time `json:"-"`
	BuffInvCapBonus          int       `json:"-"`
	BuffInvCapUntil          time.Time `json:"-"`
	BuffMaxWeightBonus       float64   `json:"-"`
	BuffMaxWeightUntil       time.Time `json:"-"`
	BuffDamageReduction      float64   `json:"-"`
	BuffDamageReductionUntil time.Time `json:"-"`
	BuffSilentUntil          time.Time `json:"-"`
	BuffJammerUntil          time.Time `json:"-"`

	// Interaction State
	ChannelingTargetUID string `json:"channeling_target"` // UID of entity being interacted with

	// Extraction
	IsExtracting    bool    `json:"is_extracting"`
	ExtractionTimer float64 `json:"extraction_timer"`
	IsExtracted     bool    `json:"is_extracted"`
}

type Item struct {
	UID     string  `json:"uid"`
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	Tier    int     `json:"tier"`
	MaxUses int     `json:"max_uses"`
	Weight  float64 `json:"weight"`
	Value   int     `json:"value"`
}

type GameConfig struct {
	Server struct {
		TickRateMs int `json:"tick_rate_ms"`
		MaxPlayers int `json:"max_players_per_room"`
	} `json:"server"`
	Map struct {
		Width       int     `json:"width"`
		Height      int     `json:"height"`
		AOIGridSize int     `json:"aoi_grid_size"`
		WallDensity float64 `json:"wall_density"`
	} `json:"map"`
	Gameplay struct {
		InventorySize  int     `json:"inventory_size"`
		BaseMoveSpeed  float64 `json:"base_move_speed"`
		BaseViewRadius float64 `json:"base_view_radius"`
		HearRadius     float64 `json:"hear_radius"`
		BaseMaxHP      float64 `json:"base_max_hp"`
		BaseMaxWeight  float64 `json:"base_max_weight"`
	} `json:"gameplay"`
	Items struct {
		InitialWorldItemCount int     `json:"initial_world_item_count"`
		RespawnIntervalSec    float64 `json:"respawn_interval_sec"`
		MerchantStockSize     int     `json:"merchant_stock_size"`
		MerchantRefreshCost   int     `json:"merchant_refresh_cost"`
		MaxWorldItemCount     struct {
			Phase1 int `json:"phase_1"`
			Phase2 int `json:"phase_2"`
			Phase3 int `json:"phase_3"`
		} `json:"max_world_item_count"`
		TierWeightsByPhase struct {
			Phase1 struct {
				T1 float64 `json:"tier_1"`
				T2 float64 `json:"tier_2"`
				T3 float64 `json:"tier_3"`
			} `json:"phase_1"`
			Phase2 struct {
				T1 float64 `json:"tier_1"`
				T2 float64 `json:"tier_2"`
				T3 float64 `json:"tier_3"`
			} `json:"phase_2"`
			Phase3 struct {
				T1 float64 `json:"tier_1"`
				T2 float64 `json:"tier_2"`
				T3 float64 `json:"tier_3"`
			} `json:"phase_3"`
		} `json:"tier_weights_by_phase"`
		ScavengeShareByPhase struct {
			Phase1 float64 `json:"phase_1"`
			Phase2 float64 `json:"phase_2"`
			Phase3 float64 `json:"phase_3"`
		} `json:"scavenge_share_by_phase"`
		TacticFocusShare float64 `json:"tactic_focus_share"`
	} `json:"items"`
	Tactics struct {
		Recon struct {
			MaxHPMult        float64 `json:"max_hp_mult"`
			MoveSpeedMult    float64 `json:"move_speed_mult"`
			ViewRadiusMult   float64 `json:"view_radius_mult"`
			HearRadiusMult   float64 `json:"hear_radius_mult"`
			HealEffectMult   float64 `json:"heal_effect_mult"`
			DamageEffectMult float64 `json:"damage_effect_mult"`
			ReconEffectMult  float64 `json:"recon_effect_mult"`
		} `json:"RECON"`
		Defense struct {
			MaxHPMult        float64 `json:"max_hp_mult"`
			MoveSpeedMult    float64 `json:"move_speed_mult"`
			ViewRadiusMult   float64 `json:"view_radius_mult"`
			HearRadiusMult   float64 `json:"hear_radius_mult"`
			HealEffectMult   float64 `json:"heal_effect_mult"`
			DamageEffectMult float64 `json:"damage_effect_mult"`
			ReconEffectMult  float64 `json:"recon_effect_mult"`
		} `json:"DEFENSE"`
		Trap struct {
			MaxHPMult        float64 `json:"max_hp_mult"`
			MoveSpeedMult    float64 `json:"move_speed_mult"`
			ViewRadiusMult   float64 `json:"view_radius_mult"`
			HearRadiusMult   float64 `json:"hear_radius_mult"`
			HealEffectMult   float64 `json:"heal_effect_mult"`
			DamageEffectMult float64 `json:"damage_effect_mult"`
			ReconEffectMult  float64 `json:"recon_effect_mult"`
		} `json:"TRAP"`
	} `json:"tactics"`
	Phases struct {
		Phase1 struct {
			Duration int `json:"duration_sec"`
		} `json:"phase_1_search"`
		Phase2 struct {
			Duration         int `json:"duration_sec"`
			MotorsSpawnCount int `json:"motors_spawn_count"`
		} `json:"phase_2_conflict"`
	} `json:"phases"`
}
