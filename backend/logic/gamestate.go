package logic

import (
	"echo_trace_server/storage"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	PhaseInit     = 0
	PhaseSearch   = 1
	PhaseConflict = 2
	PhaseEscape   = 3
	PhaseEnded    = 4
)

type GlobalEvent struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
}

type Blip struct {
	Type string  `json:"type"`
	Pos  Vector2 `json:"pos"`
}

type GameState struct {
	Config       *GameConfig
	Map          *GameMap
	Players      map[string]*Player
	Entities     map[string]Entity
	AOI          *AOIManager
	Phase        int
	PhaseTimer   float64
	RespawnTimer float64
	PulseTimer   float64
	GlobalEvents []GlobalEvent
	MotorsFixed  int
	Mutex        sync.RWMutex

	// Phase 0 State
	StartTime time.Time
}

func NewGameState(cfg *GameConfig) *GameState {
	m := NewGameMap(cfg.Map.Width, cfg.Map.Height, cfg.Map.WallDensity)
	return &GameState{
		Config:       cfg,
		Map:          m,
		Players:      make(map[string]*Player),
		Entities:     make(map[string]Entity),
		AOI:          NewAOIManager(cfg.Map.Width, cfg.Map.Height),
		Phase:        PhaseInit, // Start in Init Phase
		PhaseTimer:   float64(cfg.Phases.Phase1.Duration),
		RespawnTimer: 5.0,
		PulseTimer:   15.0,
		GlobalEvents: make([]GlobalEvent, 0),
		StartTime:    time.Now(),
	}
}

func (gs *GameState) HandleChooseTactic(sessionID, tactic string) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.Phase != PhaseInit {
		return false
	}

	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}

	// Validate Tactic
	if tactic != "RECON" && tactic != "DEFENSE" && tactic != "TRAP" {
		tactic = "RECON" // Default
	}
	p.Tactic = tactic
	gs.RecalculateStats(p)
	// Ensure current HP doesn't exceed new MaxHP.
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}

	// Add Starting Gear based on Tactic (Example)
	// p.Inventory = append(p.Inventory, ...)

	// Check if we should start
	readyCount := 0
	for _, pl := range gs.Players {
		if pl.Tactic != "" {
			readyCount++
		}
	}

	// Start condition: Min players reached (e.g. 1 for debug, 2 for real)
	minPlayers := 1 // Debug setting
	if readyCount >= minPlayers {
		gs.StartGame()
		return true
	}
	return false
}

func (gs *GameState) StartGame() {
	// Assumes Lock is HELD by caller
	gs.Phase = PhaseSearch
	gs.PhaseTimer = float64(gs.Config.Phases.Phase1.Duration)
	gs.addEvent("GAME_START", "The Hunt Begins! Search for supplies.")

	// Spawn Initial Items
	initial := 20
	if gs.Config != nil && gs.Config.Items.InitialWorldItemCount > 0 {
		initial = gs.Config.Items.InitialWorldItemCount
	}
	for i := 0; i < initial; i++ {
		gs.spawnRandomItemInternal()
	}

	// Spawn Phase 1 Supply Drops
	gs.spawnPhaseSupplyDrops(1)

	// Spawn Merchant
	gs.spawnOrMoveMerchantForPhase(1)
	gs.refreshAllPlayersShopStockForPhase(1)
}

func (gs *GameState) merchantAnchorForPhase(phaseIdx int) Vector2 {
	// Fixed per phase (deterministic) anchors, expressed as fractions of map size.
	// Phase1: center-ish, Phase2: upper-left-ish, Phase3: lower-right-ish.
	w := float64(gs.Map.Width)
	h := float64(gs.Map.Height)
	switch phaseIdx {
	case 2:
		return Vector2{X: w * 0.25, Y: h * 0.25}
	case 3:
		return Vector2{X: w * 0.75, Y: h * 0.75}
	default:
		return Vector2{X: w * 0.50, Y: h * 0.50}
	}
}

func (gs *GameState) spawnOrMoveMerchantForPhase(phaseIdx int) {
	// Remove existing merchants.
	for uid, e := range gs.Entities {
		if e.Type == EntityTypeMerchant {
			delete(gs.Entities, uid)
		}
	}

	anchor := gs.merchantAnchorForPhase(phaseIdx)
	pos := Vector2{X: math.Floor(anchor.X), Y: math.Floor(anchor.Y)}
	if pos.X < 1 {
		pos.X = 1
	}
	if pos.Y < 1 {
		pos.Y = 1
	}
	if pos.X >= float64(gs.Map.Width)-1 {
		pos.X = float64(gs.Map.Width) - 2
	}
	if pos.Y >= float64(gs.Map.Height)-1 {
		pos.Y = float64(gs.Map.Height) - 2
	}

	// Find nearby walkable tile in a small spiral.
	best := pos
	found := false
	for r := 0; r <= 6 && !found; r++ {
		for dy := -r; dy <= r && !found; dy++ {
			for dx := -r; dx <= r && !found; dx++ {
				x := pos.X + float64(dx)
				y := pos.Y + float64(dy)
				if gs.Map.IsWalkable(x, y) {
					best = Vector2{X: x, Y: y}
					found = true
				}
			}
		}
	}

	uid := NewUID()
	gs.Entities[uid] = Entity{UID: uid, Type: EntityTypeMerchant, Pos: best, State: 1}
	log.Printf("Merchant spawned for phase %d at %v", phaseIdx, best)
}

func (gs *GameState) refreshAllPlayersShopStockForPhase(phaseIdx int) {
	for _, p := range gs.Players {
		if p == nil {
			continue
		}
		p.ShopStock = gs.generateShopStock(phaseIdx, p.Tactic)
		// New phase => free refresh available again.
		p.ShopFreeRefreshUsedPhase = 0
	}
}

func (gs *GameState) HandleDropItem(sessionID string, slotIndex int) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive {
		return
	}

	if slotIndex < 0 || slotIndex >= len(p.Inventory) {
		return
	}

	item := p.Inventory[slotIndex]
	// Drop logic
	uid := NewUID()
	gs.Entities[uid] = Entity{
		UID:   uid,
		Type:  EntityTypeItemDrop,
		Pos:   p.Pos,
		State: 1,
		Extra: item,
	}

	// Remove from inv
	p.Inventory = append(p.Inventory[:slotIndex], p.Inventory[slotIndex+1:]...)
	gs.RecalculateStats(p)
	log.Printf("Player %s dropped %s", p.Name, item.ID)
}

func (gs *GameState) HandleSellItem(sessionID string, slotIndex int) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive {
		return
	}

	if slotIndex < 0 || slotIndex >= len(p.Inventory) {
		return
	}

	// Check Merchant Distance
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= 3.0 {
			nearMerchant = true
			break
		}
	}

	if !nearMerchant {
		return // Must be near merchant (or maybe via Radio? Prompt says "find merchant")
	}

	item := p.Inventory[slotIndex]
	val := item.Value
	if val == 0 {
		val = 50
	}

	p.Funds += val
	p.Inventory = append(p.Inventory[:slotIndex], p.Inventory[slotIndex+1:]...)
	gs.RecalculateStats(p)
	log.Printf("Player %s sold %s for $%d", p.Name, item.ID, val)
}

func (gs *GameState) HandleBuyItem(sessionID string, itemID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive {
		return
	}

	// Check Merchant Distance
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= 3.0 {
			nearMerchant = true
			break
		}
	}

	if !nearMerchant {
		return
	}

	// Must be in current shop stock.
	allowed := false
	for _, sid := range p.ShopStock {
		if sid == itemID {
			allowed = true
			break
		}
	}
	if !allowed {
		return
	}

	// Find Item Config
	// Iterate ItemDB (Global var in item_system.go)
	// We need to access ItemDB. It is in package logic.
	var targetItem Item
	found := false
	for _, it := range ItemDB {
		if it.ID == itemID {
			targetItem = it
			found = true
			break
		}
	}

	if !found {
		return
	}

	// Cost Logic: use item.Value (fallback: 50 * tier)
	cost := targetItem.Value
	if cost <= 0 {
		cost = 50 * targetItem.Tier
	}

	if p.Funds >= cost && len(p.Inventory) < p.InventoryCap {
		p.Funds -= cost

		newItem := targetItem
		newItem.UID = NewUID()
		p.Inventory = append(p.Inventory, newItem)

		gs.RecalculateStats(p)
		log.Printf("Player %s bought %s for $%d", p.Name, itemID, cost)
	}
}

func (gs *GameState) HandleShopRefresh(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.Phase == PhaseInit {
		return
	}

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive {
		return
	}

	// Must be near merchant.
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= 3.0 {
			nearMerchant = true
			break
		}
	}
	if !nearMerchant {
		return
	}

	phaseIdx := gs.currentLootPhaseIndex()

	// One free refresh per phase per player.
	free := p.ShopFreeRefreshUsedPhase != gs.Phase
	if free {
		p.ShopFreeRefreshUsedPhase = gs.Phase
	} else {
		cost := 120
		if gs.Config != nil && gs.Config.Items.MerchantRefreshCost > 0 {
			cost = gs.Config.Items.MerchantRefreshCost
		}
		if p.Funds < cost {
			return
		}
		p.Funds -= cost
	}

	p.ShopStock = gs.generateShopStock(phaseIdx, p.Tactic)
}

func (gs *GameState) SetPlayerName(sessionID, name string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if p, ok := gs.Players[sessionID]; ok {
		p.Name = name
		// Load from DB
		funds, _ := storage.LoadPlayer(p.Name)
		p.Funds = funds
		log.Printf("Player %s (%s) loaded with $%d", sessionID, name, funds)
	}
}

func (gs *GameState) AddPlayer(sessionID string) *Player {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	// Reconnect: if a player with the same sessionID exists, resume it.
	if existing, ok := gs.Players[sessionID]; ok && existing != nil {
		existing.Disconnected = false
		existing.DisconnectedAt = time.Time{}
		// Keep inventory/pos/funds/tactic/etc.
		return existing
	}

	spawnPos := gs.Map.GetRandomSpawnPos()
	invCap := gs.Config.Gameplay.InventorySize
	if invCap <= 0 {
		invCap = 6
	}
	baseHP := gs.Config.Gameplay.BaseMaxHP
	if baseHP <= 0 {
		baseHP = 100
	}
	baseHear := gs.Config.Gameplay.HearRadius
	if baseHear <= 0 {
		baseHear = 12.0
	}
	baseMaxWeight := gs.Config.Gameplay.BaseMaxWeight
	if baseMaxWeight <= 0 {
		baseMaxWeight = 10.0
	}
	p := &Player{
		SessionID:     sessionID,
		Name:          "Unknown",
		Pos:           spawnPos,
		LookDir:       Vector2{X: 1, Y: 0},
		HP:            baseHP,
		MaxHP:         baseHP,
		MoveSpeed:     gs.Config.Gameplay.BaseMoveSpeed,
		ViewRadius:    gs.Config.Gameplay.BaseViewRadius,
		HearRadius:    baseHear,
		MaxWeight:     baseMaxWeight,
		Weight:        0.0,
		IsAlive:       true,
		Inventory:     make([]Item, 0),
		Tactic:        "", // Not ready yet
		InventoryCap:  invCap,
		BuffSpeedMult: 1.0,
	}
	gs.Players[sessionID] = p
	gs.RecalculateStats(p)
	return p
}

func (gs *GameState) MarkPlayerDisconnected(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	if p, ok := gs.Players[sessionID]; ok && p != nil {
		p.Disconnected = true
		p.DisconnectedAt = time.Now()
		// Cancel any channeling/extraction while offline.
		p.ChannelingTargetUID = ""
		p.IsExtracting = false
	}
}

func (gs *GameState) removePlayerLocked(sessionID string) {
	if p, ok := gs.Players[sessionID]; ok {
		// Save to DB
		if p != nil && p.Name != "Unknown" {
			storage.SavePlayer(p.Name, p.Name, p.Funds, len(p.Inventory))
			log.Printf("Saved player %s data.", p.Name)
		}
		delete(gs.Players, sessionID)
	}
}

func (gs *GameState) RecalculateStats(p *Player) {
	// Assumes Lock Held
	now := time.Now()
	invCap := gs.Config.Gameplay.InventorySize
	if invCap <= 0 {
		invCap = 6
	}
	baseHP := gs.Config.Gameplay.BaseMaxHP
	if baseHP <= 0 {
		baseHP = 100
	}
	baseHear := gs.Config.Gameplay.HearRadius
	if baseHear <= 0 {
		baseHear = 12.0
	}
	baseMaxWeight := gs.Config.Gameplay.BaseMaxWeight
	if baseMaxWeight <= 0 {
		baseMaxWeight = 10.0
	}

	maxHPMult := 1.0
	moveMult := 1.0
	viewMult := 1.0
	hearMult := 1.0
	if p.Tactic != "" {
		switch p.Tactic {
		case "RECON":
			if gs.Config.Tactics.Recon.MaxHPMult > 0 {
				maxHPMult = gs.Config.Tactics.Recon.MaxHPMult
			}
			if gs.Config.Tactics.Recon.MoveSpeedMult > 0 {
				moveMult = gs.Config.Tactics.Recon.MoveSpeedMult
			}
			if gs.Config.Tactics.Recon.ViewRadiusMult > 0 {
				viewMult = gs.Config.Tactics.Recon.ViewRadiusMult
			}
			if gs.Config.Tactics.Recon.HearRadiusMult > 0 {
				hearMult = gs.Config.Tactics.Recon.HearRadiusMult
			}
		case "DEFENSE":
			if gs.Config.Tactics.Defense.MaxHPMult > 0 {
				maxHPMult = gs.Config.Tactics.Defense.MaxHPMult
			}
			if gs.Config.Tactics.Defense.MoveSpeedMult > 0 {
				moveMult = gs.Config.Tactics.Defense.MoveSpeedMult
			}
			if gs.Config.Tactics.Defense.ViewRadiusMult > 0 {
				viewMult = gs.Config.Tactics.Defense.ViewRadiusMult
			}
			if gs.Config.Tactics.Defense.HearRadiusMult > 0 {
				hearMult = gs.Config.Tactics.Defense.HearRadiusMult
			}
		case "TRAP":
			if gs.Config.Tactics.Trap.MaxHPMult > 0 {
				maxHPMult = gs.Config.Tactics.Trap.MaxHPMult
			}
			if gs.Config.Tactics.Trap.MoveSpeedMult > 0 {
				moveMult = gs.Config.Tactics.Trap.MoveSpeedMult
			}
			if gs.Config.Tactics.Trap.ViewRadiusMult > 0 {
				viewMult = gs.Config.Tactics.Trap.ViewRadiusMult
			}
			if gs.Config.Tactics.Trap.HearRadiusMult > 0 {
				hearMult = gs.Config.Tactics.Trap.HearRadiusMult
			}
		}
	}

	// Base stats (recomputed each time)
	invBonus := 0
	if now.Before(p.BuffInvCapUntil) {
		invBonus = p.BuffInvCapBonus
	}
	maxWeightBonus := 0.0
	if now.Before(p.BuffMaxWeightUntil) {
		maxWeightBonus = p.BuffMaxWeightBonus
	}
	viewBonus := 0.0
	if now.Before(p.BuffViewUntil) {
		viewBonus = p.BuffViewBonus
	}
	hearBonus := 0.0
	if now.Before(p.BuffHearUntil) {
		hearBonus = p.BuffHearBonus
	}
	speedBuffMult := 1.0
	if now.Before(p.BuffSpeedUntil) && p.BuffSpeedMult > 0 {
		speedBuffMult = p.BuffSpeedMult
	}

	p.InventoryCap = invCap + invBonus
	p.MaxWeight = baseMaxWeight + maxWeightBonus
	p.MaxHP = baseHP * maxHPMult
	p.ViewRadius = (gs.Config.Gameplay.BaseViewRadius * viewMult) + viewBonus
	p.HearRadius = (baseHear * hearMult) + hearBonus

	// Weight always depends on what you carry.
	totalWeight := 0.0
	for _, item := range p.Inventory {
		totalWeight += item.Weight
	}
	p.Weight = totalWeight

	ratio := p.Weight / p.MaxWeight
	if ratio > 1.0 {
		ratio = 1.0
	}

	// Speed penalty up to 60%
	p.MoveSpeed = (gs.Config.Gameplay.BaseMoveSpeed * moveMult * speedBuffMult) * (1.0 - ratio*0.6)
	if p.MoveSpeed < 2.0 {
		p.MoveSpeed = 2.0
	}
}

func (gs *GameState) RemovePlayer(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	gs.removePlayerLocked(sessionID)
}

func (gs *GameState) ProcessExtraction(p *Player) {
	lootValue := 0
	for _, item := range p.Inventory {
		val := item.Value
		if val == 0 {
			// Fallback if not set
			val = 50 * item.Tier
		}
		lootValue += val
	}

	p.Funds += lootValue
	p.Inventory = []Item{} // Clear inventory on extract
	p.IsExtracted = true
	p.IsAlive = false    // Stop physics/interaction
	p.ViewRadius = 100.0 // Spectator Mode

	// Save Immediately
	storage.SavePlayer(p.Name, p.Name, p.Funds, 0)
	gs.addEvent("EXTRACTION", fmt.Sprintf("%s escaped with $%d!", p.Name, lootValue))
	log.Printf("Player %s extracted. Funds: %d (+%d)", p.Name, p.Funds, lootValue)
}

func (gs *GameState) HandleDevSkipPhase() {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.Phase != PhaseEnded {
		gs.PhaseTimer = 0
		gs.addEvent("DEV", "Phase Skipped by Developer!")
		// UpdateTick will handle the transition
	}
}

func (gs *GameState) HandleInput(sessionID string, dir Vector2, lookDir Vector2, hasLookDir bool) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.Phase == PhaseInit {
		return
	}

	if p, ok := gs.Players[sessionID]; ok && p.IsAlive {
		p.TargetDir = dir
		if hasLookDir {
			// Normalize (ignore zero vector)
			l2 := lookDir.X*lookDir.X + lookDir.Y*lookDir.Y
			if l2 > 0 {
				invLen := 1.0 / math.Sqrt(l2)
				p.LookDir = Vector2{X: lookDir.X * invLen, Y: lookDir.Y * invLen}
			}
		} else if dir.X != 0 || dir.Y != 0 {
			// Fallback: if client doesn't send look_dir, face movement direction.
			l2 := dir.X*dir.X + dir.Y*dir.Y
			if l2 > 0 {
				invLen := 1.0 / math.Sqrt(l2)
				p.LookDir = Vector2{X: dir.X * invLen, Y: dir.Y * invLen}
			}
		}
		if dir.X != 0 || dir.Y != 0 {
			p.ChannelingTargetUID = ""
		}
	}
}

func (gs *GameState) HandleInteract(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.Phase == PhaseInit {
		return
	}

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive {
		return
	}

	interactRange := 2.0
	var targetUID = ""

	for uid, e := range gs.Entities {
		if (e.Type == EntityTypeMotor && e.State != 2) || (e.Type == EntityTypeExit && e.State == 1) {
			if Distance(p.Pos, e.Pos) <= interactRange {
				targetUID = uid
				break
			}
		}
	}

	if targetUID != "" {
		p.ChannelingTargetUID = targetUID
		// If Exit, start extraction timer
		if gs.Entities[targetUID].Type == EntityTypeExit {
			p.IsExtracting = true
			p.ExtractionTimer = 3.0 // 3 seconds
			log.Printf("Player %s started extraction...", sessionID)
		} else {
			log.Printf("Player %s started fixing Motor %s", sessionID, targetUID)
		}
	}
}

func (gs *GameState) UpdateTick(dt float64) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	// 0. Disconnect cleanup (kick after grace period)
	graceSec := 0
	if gs.Config != nil {
		graceSec = gs.Config.Server.DisconnectGraceSec
	}
	if graceSec > 0 {
		now := time.Now()
		deadline := time.Duration(graceSec) * time.Second
		for sid, p := range gs.Players {
			if p == nil {
				continue
			}
			if p.Disconnected && !p.DisconnectedAt.IsZero() && now.Sub(p.DisconnectedAt) > deadline {
				gs.addEvent("PLAYER_KICK", fmt.Sprintf("%s disconnected too long and was removed.", p.Name))
				gs.removePlayerLocked(sid)
			}
		}
	}

	if gs.Phase == PhaseInit {
		return
	}

	// 1. Phase Logic
	if gs.Phase != PhaseEnded {
		gs.PhaseTimer -= dt
		if gs.PhaseTimer <= 0 {
			gs.nextPhase()
		}
	}

	// 1.1 Motor Pulse Logic
	if gs.Phase == PhaseConflict {
		gs.PulseTimer -= dt
		if gs.PulseTimer <= 0 {
			gs.PulseTimer = 15.0
			gs.addEvent("MOTOR_PULSE", "Motors are emitting a signal!")
		}
	}

	// 2. Channeling Logic
	for _, p := range gs.Players {
		if p == nil {
			continue
		}
		if p.Disconnected {
			// No channeling progress while offline.
			p.IsExtracting = false
			p.ChannelingTargetUID = ""
			continue
		}
		if p.IsAlive && p.ChannelingTargetUID != "" {
			ent, ok := gs.Entities[p.ChannelingTargetUID]
			if !ok {
				p.ChannelingTargetUID = ""
				p.IsExtracting = false
				continue
			}

			if ent.Type == EntityTypeMotor {
				data := ent.Extra.(MotorData)
				data.Progress += 20.0 * dt
				if data.Progress >= data.MaxProgress {
					data.Progress = data.MaxProgress
					ent.State = 2
					gs.MotorsFixed++
					gs.addEvent("MOTOR_FIXED", "A Motor has been repaired!")
					p.ChannelingTargetUID = ""
					if gs.MotorsFixed >= 2 && gs.Phase == PhaseConflict {
						gs.startEscapePhase()
					}
				}
				ent.Extra = data
				gs.Entities[p.ChannelingTargetUID] = ent
			} else if ent.Type == EntityTypeExit {
				// Extraction Logic
				if p.IsExtracting {
					p.ExtractionTimer -= dt
					if p.ExtractionTimer <= 0 {
						// Success
						gs.ProcessExtraction(p)
						p.ChannelingTargetUID = ""
						p.IsExtracting = false
						// Do NOT remove player to allow spectating
						// gs.RemovePlayer(p.SessionID)
					}
				}
			}
		} else {
			p.IsExtracting = false
		}
	}

	// 3. Item Respawn (Optimized)
	gs.RespawnTimer -= dt
	if gs.RespawnTimer <= 0 {
		interval := 5.0
		if gs.Config != nil && gs.Config.Items.RespawnIntervalSec > 0 {
			interval = gs.Config.Items.RespawnIntervalSec
		}
		gs.RespawnTimer = interval
		itemCount := 0
		for _, e := range gs.Entities {
			if e.Type == EntityTypeItemDrop {
				itemCount++
			}
		}
		cap := 60
		if gs.Config != nil {
			phaseIdx := 1
			if gs.Phase == PhaseConflict {
				phaseIdx = 2
			} else if gs.Phase == PhaseEscape {
				phaseIdx = 3
			}
			switch phaseIdx {
			case 1:
				if gs.Config.Items.MaxWorldItemCount.Phase1 > 0 {
					cap = gs.Config.Items.MaxWorldItemCount.Phase1
				}
			case 2:
				if gs.Config.Items.MaxWorldItemCount.Phase2 > 0 {
					cap = gs.Config.Items.MaxWorldItemCount.Phase2
				}
			case 3:
				if gs.Config.Items.MaxWorldItemCount.Phase3 > 0 {
					cap = gs.Config.Items.MaxWorldItemCount.Phase3
				}
			}
		}

		if itemCount < cap {
			gs.spawnRandomItemInternal()
		}
	}

	// 4. Physics
	// Optimized Collision Radius (0.5) to avoid getting stuck
	playerRadius := 0.25 // Visual size is 0.5, so radius 0.25 fits nicely
	// Recompute stats each tick so timed buffs expire correctly and weight penalties stay accurate.
	for _, p := range gs.Players {
		if p != nil && p.IsAlive && !p.Disconnected {
			gs.RecalculateStats(p)
		}
	}
	for _, p := range gs.Players {
		if p == nil || !p.IsAlive || p.Disconnected {
			continue
		}
		if p.TargetDir.X != 0 || p.TargetDir.Y != 0 {
			len := math.Sqrt(p.TargetDir.X*p.TargetDir.X + p.TargetDir.Y*p.TargetDir.Y)
			if len > 0 {
				p.TargetDir.X /= len
				p.TargetDir.Y /= len
			}

			delta := Vector2{
				X: p.TargetDir.X * p.MoveSpeed * dt,
				Y: p.TargetDir.Y * p.MoveSpeed * dt,
			}
			p.Pos = gs.ResolveMovement(p.Pos, delta, playerRadius)
		}
	}
}

func (gs *GameState) nextPhase() {
	gs.Phase++
	if gs.Phase == PhaseConflict {
		gs.PhaseTimer = 9999
		gs.PulseTimer = 15.0 // Ensure immediate pulse on start
		gs.addEvent("PHASE_CHANGE", "Phase 2: Conflict! Fix 2 Motors to escape.")
		gs.spawnMotors(5)
		gs.spawnPhaseSupplyDrops(2)
		gs.spawnOrMoveMerchantForPhase(2)
		gs.refreshAllPlayersShopStockForPhase(2)
	} else if gs.Phase == PhaseEscape {
		gs.startEscapePhase()
		gs.spawnPhaseSupplyDrops(3)
		gs.spawnOrMoveMerchantForPhase(3)
		gs.refreshAllPlayersShopStockForPhase(3)
	}
}

func (gs *GameState) spawnPhaseSupplyDrops(phase int) {
	// Calculate Centroid
	count := 0
	sumX, sumY := 0.0, 0.0
	for _, p := range gs.Players {
		if p.IsAlive {
			sumX += p.Pos.X
			sumY += p.Pos.Y
			count++
		}
	}

	center := gs.Map.GetRandomSpawnPos()
	if count > 0 {
		center = Vector2{X: sumX / float64(count), Y: sumY / float64(count)}
	}

	// Spawn 1-2 drops near center
	dropCount := 1
	if phase >= 2 {
		dropCount = 2
	}

	for i := 0; i < dropCount; i++ {
		// Random offset from center
		offsetX := (float64(time.Now().UnixNano()%20) - 10)
		offsetY := (float64(time.Now().UnixNano()%20) - 10)
		pos := Vector2{X: center.X + offsetX, Y: center.Y + offsetY}

		// Clamp to map
		if pos.X < 1 {
			pos.X = 1
		}
		if pos.Y < 1 {
			pos.Y = 1
		}
		if pos.X >= float64(gs.Map.Width)-1 {
			pos.X = float64(gs.Map.Width) - 1
		}
		if pos.Y >= float64(gs.Map.Height)-1 {
			pos.Y = float64(gs.Map.Height) - 1
		}

		if !gs.checkCollision(pos, 0.5) {
			gs.SpawnSupplyDrop(pos, phase)
		} else {
			// Fallback
			gs.SpawnSupplyDrop(gs.Map.GetRandomSpawnPos(), phase)
		}
	}
}

func (gs *GameState) startEscapePhase() {
	gs.Phase = PhaseEscape
	gs.PhaseTimer = 120
	gs.addEvent("PHASE_CHANGE", "Phase 3: ESCAPE! The Exit has opened.")
	gs.spawnExit()
}

func (gs *GameState) spawnMotors(count int) {
	for i := 0; i < count; i++ {
		pos := gs.Map.GetRandomSpawnPos()
		uid := NewUID()
		gs.Entities[uid] = Entity{
			UID:   uid,
			Type:  EntityTypeMotor,
			Pos:   pos,
			State: 0,
			Extra: MotorData{MaxProgress: 100},
		}
	}
}

func (gs *GameState) spawnExit() {
	pos := gs.Map.GetRandomSpawnPos()
	uid := NewUID()
	gs.Entities[uid] = Entity{
		UID:   uid,
		Type:  EntityTypeExit,
		Pos:   pos,
		State: 1,
	}
}

func (gs *GameState) addEvent(t, msg string) {
	gs.GlobalEvents = append(gs.GlobalEvents, GlobalEvent{Type: t, Msg: msg})
	if len(gs.GlobalEvents) > 5 {
		gs.GlobalEvents = gs.GlobalEvents[1:]
	}
}

func (gs *GameState) GetSnapshot(sessionID string) map[string]interface{} {
	gs.Mutex.RLock()
	defer gs.Mutex.RUnlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return nil
	}

	entSlice := make([]Entity, 0, len(gs.Entities))
	for _, e := range gs.Entities {
		entSlice = append(entSlice, e)
	}

	var visiblePlayers []*Player
	var visibleEntities []Entity

	if p.IsExtracted {
		// Spectator Mode: See All
		for _, pl := range gs.Players {
			visiblePlayers = append(visiblePlayers, pl)
		}
		visibleEntities = entSlice
	} else {
		visiblePlayers, visibleEntities = gs.AOI.GetVisibleEntities(p, gs.Map, gs.Players, entSlice)
	}

	// Radar Logic: Calculate Blips
	radarBlips := make([]Blip, 0)

	// Phase 2: Pulse Motors
	isPulseActive := gs.Phase == PhaseConflict && gs.PulseTimer > 10.0

	if isPulseActive {
		for _, e := range gs.Entities {
			if e.Type == EntityTypeMotor {
				radarBlips = append(radarBlips, Blip{Type: "MOTOR", Pos: e.Pos})
			}
		}
	}
	if gs.Phase == PhaseEscape {
		for _, e := range gs.Entities {
			if e.Type == EntityTypeExit {
				radarBlips = append(radarBlips, Blip{Type: "EXIT", Pos: e.Pos})
			}
		}
	}

	// Always Show Supply Drops
	for _, e := range gs.Entities {
		if e.Type == EntityTypeSupplyDrop {
			radarBlips = append(radarBlips, Blip{Type: "SUPPLY_DROP", Pos: e.Pos})
		}
	}

	// Sound Logic (Hearing)
	soundEvents := make([]map[string]interface{}, 0)
	now := time.Now()
	for _, other := range gs.Players {
		if other.SessionID == sessionID || !other.IsAlive {
			continue
		}
		// Silent buff: do not emit footsteps at all.
		if now.Before(other.BuffSilentUntil) {
			continue
		}

		isMoving := other.TargetDir.X != 0 || other.TargetDir.Y != 0
		if isMoving {
			dist := Distance(p.Pos, other.Pos)
			if dist <= p.HearRadius {
				dir := Vector2{X: other.Pos.X - p.Pos.X, Y: other.Pos.Y - p.Pos.Y}
				len := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)
				if len > 0 {
					dir.X /= len
					dir.Y /= len
				}

				intensity := 1.0 - (dist / p.HearRadius)
				if intensity < 0 {
					intensity = 0
				}

				// Jammer buff: scramble perceived direction + dampen intensity.
				if now.Before(other.BuffJammerUntil) {
					ang := rand.Float64() * 2.0 * math.Pi
					dir = Vector2{X: math.Cos(ang), Y: math.Sin(ang)}
					intensity *= 0.25
				}

				soundEvents = append(soundEvents, map[string]interface{}{
					"type":      "FOOTSTEP",
					"dir":       dir,
					"intensity": intensity,
				})
			}
		}
	}

	return map[string]interface{}{
		"timestamp": 0,
		"phase":     gs.Phase,
		"time_left": gs.PhaseTimer,
		"events":    gs.GlobalEvents,
		"self":      p,
		"vision": map[string]interface{}{
			"players":  visiblePlayers,
			"entities": visibleEntities,
		},
		"radar_blips": radarBlips,
		"sound": map[string]interface{}{
			"events": soundEvents,
		},
	}
}

func (gs *GameState) spawnRandomItemInternal() {
	_ = gs.spawnWeightedRandomItemAt(gs.Map.GetRandomSpawnPos())
}
