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
	EntityTypeProjectile = "PROJECTILE"
	EntityTypeNingBye    = "NING_BYE"
)

type ProjectileData struct {
	OwnerID     string    `json:"owner_id"`
	Velocity    Vector2   `json:"velocity"`
	Damage      float64   `json:"damage"`
	Radius      float64   `json:"radius"`
	Lifetime    time.Time `json:"lifetime"`
	BouncesLeft int       `json:"bounces_left"`
	// AI Specifics
	ArmorPenetration float64 `json:"armor_penetration"` // 0.0 to 1.0
	// T4 Specifics
	OnDeathExplode   bool    `json:"on_death_explode"`
	ExplosionRadius  float64 `json:"explosion_radius"`
}

// NingByeAI holds the state for the boss unit
type NingByeAI struct {
	State            int       `json:"state"` // 0: Patrol, 1: CommandMove, 2: Combat
	HP               float64   `json:"hp"`
	MaxHP            float64   `json:"max_hp"`
	Armor            float64   `json:"armor"`
	MaxArmor         float64   `json:"max_armor"`
	MoveSpeed        float64   `json:"move_speed"`
	Damage           float64   `json:"damage"`
	ArmorPenetration float64   `json:"armor_penetration"`
	ReloadTimeSec    float64   `json:"reload_time_sec"`
	TargetPos        Vector2   `json:"target_pos"`      // For Command Move
	PatrolPath       []Vector2 `json:"-"`               // Current movement path
	CombatTargetID   string    `json:"combat_target"`   // Player ID
	LostTargetTime   time.Time `json:"-"`               // When did we lose the target?
	LastFireTime     time.Time `json:"-"`
	SensingRadius    float64   `json:"sensing_radius"`
}

type Entity struct {
	UID   string      `json:"uid"`
	Type  string      `json:"type"`
	Pos   Vector2     `json:"pos"`
	State int         `json:"state"` // For Motors: 0=Inactive, 1=Active, 2=Done. For AI: Syncs logic state.
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
	LookDir      Vector2 `json:"look_dir"`
	HP           float64 `json:"hp"`
	MaxHP        float64 `json:"max_hp"`
	Armor        float64 `json:"armor"`
	MaxArmor     float64 `json:"max_armor"`
	Kills        int     `json:"kills"`
	MoveSpeed    float64 `json:"move_speed"`
	ViewRadius   float64 `json:"view_radius"`
	HearRadius   float64 `json:"hear_radius"`
	IsAlive      bool    `json:"is_alive"`
	IsDead       bool    `json:"is_dead"`
	RespawnTimer time.Time `json:"respawn_timer"`
	DeathHandled bool    `json:"-"`
	Tactic       string  `json:"tactic"`

	InventoryCap int `json:"inventory_cap"`
	ExtraInvCap  int `json:"extra_inv_cap"` // Permanent bonus from Admin/Upgrades

	MaxWeight float64 `json:"max_weight"`
	Weight    float64 `json:"weight"`
	Funds     int     `json:"funds"`
	
	// Threat System
	IsThreat bool `json:"is_threat"` // Controlled by MCP

	Velocity                 Vector2  `json:"-"`
	TargetDir                Vector2  `json:"-"`
	Inventory                []Item   `json:"inventory"`
	ShopStock                []string `json:"shop_stock"`
	ShopPrices               []int    `json:"shop_prices,omitempty"`
	ShopTypes                []string `json:"shop_types,omitempty"`
	ShopFreeRefreshUsedPhase int      `json:"-"`

	// ClientMsg is an ephemeral message intended for the owning client UI only.
	// It should not be used for global announcements.
	ClientMsg string `json:"client_msg,omitempty"`
	
	// LastAIAction is a persistent summary of the last action taken by the AI for this player.
	// Rendered independently on the HUD.
	LastAIAction string `json:"last_ai_action,omitempty"`

	// Timed buffs (server-authoritative).
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
	
	// New Item Buffs
	BuffInvincibleUntil      time.Time `json:"buff_invincible_until"` // Abs Defense
	BuffInvisibleUntil       time.Time `json:"buff_invisible_until"`  // Stealth
	BuffVisionInvertUntil    time.Time `json:"buff_vision_invert_until"` // T2 Recon
	BuffScanUntil            time.Time `json:"buff_scan_until"` // T3 Recon (user)
	BuffNingByeUntil         time.Time `json:"buff_ning_bye_until"` // T4 Survival
	
	// Ammo State
	AmmoType      string    `json:"ammo_type"` // "", "AP", "BOUNCE"
	AmmoCount     int       `json:"ammo_count"`
	LastFireTime  time.Time `json:"-"`
	
	// Interaction State
	ChannelingTargetUID string `json:"channeling_target"` // UID of entity being interacted with

	// Extraction
	IsExtracting    bool    `json:"is_extracting"`
	ExtractionTimer float64 `json:"extraction_timer"`
	IsExtracted     bool    `json:"is_extracted"`

	// Reconnect / disconnect handling (server-authoritative)
	Disconnected   bool      `json:"-"`
	DisconnectedAt time.Time `json:"-"`
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
		TickRateMs               int  `json:"tick_rate_ms"`
		MaxPlayers               int  `json:"max_players_per_room"`
		MinPlayersToStart        int  `json:"min_players_to_start"`
		WaitForPlayersTimeoutSec int  `json:"wait_for_players_timeout_sec"`
		DisconnectGraceSec       int  `json:"disconnect_grace_sec"`
		GlobalEventsMax          int  `json:"global_events_max"`
		DebugLogColor            bool `json:"debug_log_color"`
	} `json:"server"`
	Map struct {
		Width       int     `json:"width"`
		Height      int     `json:"height"`
		WallDensity float64 `json:"wall_density"`
	} `json:"map"`
	Gameplay struct {
		InventorySize              int     `json:"inventory_size"`
		SafeSlotCount              int     `json:"safe_slot_count"`
		PlayerCollisionRadius      float64 `json:"player_collision_radius"`
		InteractRange              float64 `json:"interact_range"`
		MerchantInteractRange      float64 `json:"merchant_interact_range"`
		FOVDegrees                 float64 `json:"fov_degrees"`
		FOVRayCount                int     `json:"fov_ray_count"`
		BaseMoveSpeed              float64 `json:"base_move_speed"`
		BaseViewRadius             float64 `json:"base_view_radius"`
		HearRadius                 float64 `json:"hear_radius"`
		BaseMaxHP                  float64 `json:"base_max_hp"`
		BaseMaxWeight              float64 `json:"base_max_weight"`
		WeightThresholdNoiseDouble float64 `json:"weight_threshold_noise_double"`
		WeightThresholdViewReduce  float64 `json:"weight_threshold_view_reduce"`
		WeightThresholdImmobilize  float64 `json:"weight_threshold_immobilize"`
		RespawnTimeSec             float64 `json:"respawn_time_sec"`
		RespawnFarDistance         float64 `json:"respawn_far_distance"`
		ItemDurations              struct {
			PhaseShiftSec   float64 `json:"phase_shift_sec"`
			PurgeSec        float64 `json:"purge_sec"`
			ReconScopeSec   float64 `json:"recon_scope_sec"`
			ReconSensorSec  float64 `json:"recon_sensor_sec"`
			ReconScannerSec float64 `json:"recon_scanner_sec"`
			BlinkDistance   float64 `json:"blink_distance"`
			StealthSec      float64 `json:"stealth_sec"`
		} `json:"item_durations"`
	} `json:"gameplay"`
	Items struct {
		InitialWorldItemCount     int     `json:"initial_world_item_count"`
		RespawnIntervalSec        float64 `json:"respawn_interval_sec"`
		MerchantStockSize         int     `json:"merchant_stock_size"`
		MerchantRefreshCost       int     `json:"merchant_refresh_cost"`
		MerchantSpawnSearchRadius int     `json:"merchant_spawn_search_radius"`
		DefaultSellValue          int     `json:"default_sell_value"`
		DefaultBuyMultiplier      int     `json:"default_buy_multiplier"`
		PickupRange               float64 `json:"pickup_range"`
		FundsGainMin              int     `json:"funds_gain_min"`
		FundsGainMax              int     `json:"funds_gain_max"`
		ShopMaxAttempts           int     `json:"shop_max_attempts"`
		ShopT4Chance              float64 `json:"shop_t4_chance"`
		SupplyDropT4Chance        float64 `json:"supply_drop_t4_chance"`
		SupplyDropFundsBase       int     `json:"supply_drop_funds_base"`
		MaxWorldItemCount         struct {
			Phase1 int `json:"phase_1"`
			Phase2 int `json:"phase_2"`
			Phase3 int `json:"phase_3"`
		} `json:"max_world_item_count"`
		SupplyDropCount struct {
			Phase1 int `json:"phase_1"`
			Phase2 int `json:"phase_2"`
			Phase3 int `json:"phase_3"`
		} `json:"supply_drop_count"`
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
	AI struct {
		NingBye struct {
			HP                    float64 `json:"hp"`
			Armor                 float64 `json:"armor"`
			MoveSpeed             float64 `json:"move_speed"`
			Damage                float64 `json:"damage"`
			ArmorPenetration      float64 `json:"armor_penetration"`
			ReloadTimeSec         float64 `json:"reload_time_sec"`
			SensingRadiusRatio    float64 `json:"sensing_radius_ratio"`
			PulseIntervalSec      float64 `json:"pulse_interval_sec"`
			LegendaryDropChance   float64 `json:"legendary_drop_chance"`
			LostTargetTimeoutSec  float64 `json:"lost_target_timeout_sec"`
			ThreatScanDistance    float64 `json:"threat_scan_distance"`
			ProjectileSpeed       float64 `json:"projectile_speed"`
			ProjectileRadius      float64 `json:"projectile_radius"`
			ProjectileLifetimeSec float64 `json:"projectile_lifetime_sec"`
			ProjectileSpawnOffset float64 `json:"projectile_spawn_offset"`
			ArrivalDistance       float64 `json:"arrival_distance"`
			PathfindingMaxSteps   int     `json:"pathfinding_max_steps"`
		} `json:"ning_bye"`
	} `json:"ai"`
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
	Combat struct {
		BaseAttackDamage         float64 `json:"base_attack_damage"`
		BulletDamage             float64 `json:"bullet_damage"`
		BaseArmor                float64 `json:"base_armor"`
		ProjectileLifetimeSec    float64 `json:"projectile_lifetime_sec"`
		ReloadTimeSec            float64 `json:"reload_time_sec"`
		DefaultBounces           int     `json:"default_bounces"`
		AdvancedReconDurationSec float64 `json:"advanced_recon_duration_sec"`
		AttackRequiresVision     bool    `json:"attack_requires_vision"`
		ProjectileSpeed          float64 `json:"projectile_speed"`
		ProjectileRadius         float64 `json:"projectile_radius"`
		AmmoAPDamage             float64 `json:"ammo_ap_damage"`
		AmmoBounceDamage         float64 `json:"ammo_bounce_damage"`
		AmmoBounceBonus          int     `json:"ammo_bounce_bonus"`
		RailgunDamage            float64 `json:"railgun_damage"`
		RailgunTrapPenetration   float64 `json:"railgun_trap_penetration"`
	} `json:"combat"`
	Phases struct {
		Thresholds struct {
			Phase2Kills int `json:"phase_2_kills"`
			Phase3Kills int `json:"phase_3_kills"`
			EndGameKills int `json:"end_game_kills"`
		} `json:"thresholds"`
		Phase1 struct {			Duration int `json:"duration_sec"`
		} `json:"phase_1_search"`
		Phase2 struct {
			Duration                 int `json:"duration_sec"`
			MotorsSpawnCount         int `json:"motors_spawn_count"`
			MotorsRequiredToOpenExit int `json:"motors_required_to_open_exit"`
			MotorDecipherTimeSec     int `json:"motor_decipher_time_sec"`
			PulseIntervalSec         int `json:"pulse_interval_sec"`
			PulseActiveWindowSec     int `json:"pulse_active_window_sec"`
		} `json:"phase_2_conflict"`
		Phase3 struct {
			DurationSec               int     `json:"duration_sec"`
			ExtractionChannelTimeSec  float64 `json:"extraction_channel_time_sec"`
			ExtractionSlotsTotal      int     `json:"extraction_slots_total"`
			ExtractionCooldownSec     int     `json:"extraction_cooldown_sec"`
			GlobalPulseIntervalSec    int     `json:"global_pulse_interval_sec"`
			ViewRadiusDecayRatePerSec float64 `json:"view_radius_decay_rate_per_sec"`
		} `json:"phase_3_escape"`
		Phase4 struct {
			DurationSec int `json:"duration_sec"`
		} `json:"phase_4_ended"`
	} `json:"phases"`
}
