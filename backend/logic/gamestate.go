package logic

import (
	"echo_trace_server/storage"
	"log"
	"math"
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
	StartTime    time.Time
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
	for i := 0; i < 20; i++ {
		gs.spawnRandomItemInternal()
	}

	// Spawn Phase 1 Supply Drops
	gs.spawnPhaseSupplyDrops(1)
	
	// Spawn Merchant
	gs.spawnMerchant()
}

func (gs *GameState) spawnMerchant() {
	// Center spawn
	pos := Vector2{X: float64(gs.Map.Width)/2, Y: float64(gs.Map.Height)/2}
	// Find walkable near center
	for r := 0; r < 5; r++ {
		if gs.Map.IsWalkable(pos.X+float64(r), pos.Y) {
			pos.X += float64(r)
			break
		}
	}
	
	uid := NewUID()
	gs.Entities[uid] = Entity{
		UID:   uid,
		Type:  EntityTypeMerchant,
		Pos:   pos,
		State: 1,
	}
	log.Printf("Merchant spawned at %v", pos)
}

func (gs *GameState) HandleDropItem(sessionID string, slotIndex int) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	
	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive { return }
	
	if slotIndex < 0 || slotIndex >= len(p.Inventory) { return }
	
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
	if !ok || !p.IsAlive { return }
	
	if slotIndex < 0 || slotIndex >= len(p.Inventory) { return }
	
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
	if val == 0 { val = 50 }
	
	p.Funds += val
	p.Inventory = append(p.Inventory[:slotIndex], p.Inventory[slotIndex+1:]...)
	gs.RecalculateStats(p)
	log.Printf("Player %s sold %s for $%d", p.Name, item.ID, val)
}

func (gs *GameState) HandleBuyItem(sessionID string, itemID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive { return }

	// Check Merchant Distance
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= 3.0 {
			nearMerchant = true
			break
		}
	}
	
	if !nearMerchant { return }

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
	
	if !found { return }
	
	// Cost Logic: Buy Price = Value * 2 (or just hardcoded mapping matching frontend)
	// Frontend has:
	// T1: Shock 100, Med 50, Radar 150
	// T2: Shock 200, Med 100, Radar 300
	// T3: Shock 350, Radar 500
	
	// Let's rely on item.Value if we set it, or simple switch
	cost := 0
	switch itemID {
	case "WPN_SHOCK": cost = 100
	case "SURV_MEDKIT": cost = 50
	case "RECON_RADAR": cost = 150
	case "WPN_SHOCK_T2": cost = 200
	case "SURV_MEDKIT_T2": cost = 100
	case "RECON_RADAR_T2": cost = 300
	case "WPN_SHOCK_T3": cost = 350
	case "RECON_RADAR_T3": cost = 500
	default: cost = 9999
	}
	
	if p.Funds >= cost && len(p.Inventory) < 6 {
		p.Funds -= cost
		
		newItem := targetItem
		newItem.UID = NewUID()
		p.Inventory = append(p.Inventory, newItem)
		
		gs.RecalculateStats(p)
		log.Printf("Player %s bought %s for $%d", p.Name, itemID, cost)
	}
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

	spawnPos := gs.Map.GetRandomSpawnPos()
	p := &Player{
		SessionID:  sessionID,
		Name:       "Unknown",
		Pos:        spawnPos,
		HP:         100,
		MaxHP:      100,
		MoveSpeed:  gs.Config.Gameplay.BaseMoveSpeed,
		ViewRadius: gs.Config.Gameplay.BaseViewRadius,
		HearRadius: 12.0,
		MaxWeight:  10.0,
		Weight:     0.0,
		IsAlive:    true,
		Inventory:  make([]Item, 0),
		Tactic:     "", // Not ready yet
	}
	gs.Players[sessionID] = p
	return p
}

func (gs *GameState) RecalculateStats(p *Player) {
	// Assumes Lock Held
	totalWeight := 0.0
	for _, item := range p.Inventory {
		totalWeight += item.Weight
	}
	p.Weight = totalWeight
	
	ratio := p.Weight / p.MaxWeight
	if ratio > 1.0 { ratio = 1.0 }
	
	// Speed penalty up to 60%
	p.MoveSpeed = gs.Config.Gameplay.BaseMoveSpeed * (1.0 - ratio * 0.6)
	if p.MoveSpeed < 2.0 { p.MoveSpeed = 2.0 }
}

func (gs *GameState) RemovePlayer(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	
	if p, ok := gs.Players[sessionID]; ok {
		// Save to DB
		if p.Name != "Unknown" {
			storage.SavePlayer(p.Name, p.Name, p.Funds, len(p.Inventory))
			log.Printf("Saved player %s data.", p.Name)
		}
		delete(gs.Players, sessionID)
	}
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
	p.IsAlive = false      // Stop physics/interaction
	p.ViewRadius = 100.0   // Spectator Mode
	
	// Save Immediately
	storage.SavePlayer(p.Name, p.Name, p.Funds, 0)
	gs.addEvent("EXTRACTION", p.Name + " escaped with $" + string(rune(lootValue)) + "!")
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

func (gs *GameState) HandleInput(sessionID string, dir Vector2) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	
	if gs.Phase == PhaseInit { return }

	if p, ok := gs.Players[sessionID]; ok && p.IsAlive {
		p.TargetDir = dir
		if dir.X != 0 || dir.Y != 0 {
			p.ChannelingTargetUID = ""
		}
	}
}

func (gs *GameState) HandleInteract(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	
	if gs.Phase == PhaseInit { return }
	
	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive { return }

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
		gs.RespawnTimer = 5.0 // Faster respawn check
		itemCount := 0
		for _, e := range gs.Entities {
			if e.Type == EntityTypeItemDrop { itemCount++ }
		}
		// Higher cap to simulate longer persistence
		if itemCount < 100 {
			gs.spawnRandomItemInternal()
		}
	}

	// 4. Physics
	// Optimized Collision Radius (0.5) to avoid getting stuck
	playerRadius := 0.25 // Visual size is 0.5, so radius 0.25 fits nicely
	for _, p := range gs.Players {
		if !p.IsAlive { continue }
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
	} else if gs.Phase == PhaseEscape {
		gs.startEscapePhase()
		gs.spawnPhaseSupplyDrops(3)
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
		center = Vector2{X: sumX/float64(count), Y: sumY/float64(count)}
	}

	// Spawn 1-2 drops near center
	dropCount := 1
	if phase >= 2 { dropCount = 2 }
	
	for i := 0; i < dropCount; i++ {
		// Random offset from center
		offsetX := (float64(time.Now().UnixNano()%20) - 10) 
		offsetY := (float64(time.Now().UnixNano()%20) - 10)
		pos := Vector2{X: center.X + offsetX, Y: center.Y + offsetY}
		
		// Clamp to map
		if pos.X < 1 { pos.X = 1 }
		if pos.Y < 1 { pos.Y = 1 }
		if pos.X >= float64(gs.Map.Width)-1 { pos.X = float64(gs.Map.Width)-1 }
		if pos.Y >= float64(gs.Map.Height)-1 { pos.Y = float64(gs.Map.Height)-1 }

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
	if !ok { return nil }

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
		visiblePlayers, visibleEntities = gs.AOI.GetVisibleEntities(p, gs.Players, entSlice)
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
	for _, other := range gs.Players {
		if other.SessionID == sessionID || !other.IsAlive { continue }
		
		isMoving := other.TargetDir.X != 0 || other.TargetDir.Y != 0
		if isMoving {
			dist := Distance(p.Pos, other.Pos)
			if dist <= p.HearRadius {
				dir := Vector2{X: other.Pos.X - p.Pos.X, Y: other.Pos.Y - p.Pos.Y}
				len := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)
				if len > 0 { dir.X /= len; dir.Y /= len }
				
				intensity := 1.0 - (dist / p.HearRadius)
				if intensity < 0 { intensity = 0 }

				soundEvents = append(soundEvents, map[string]interface{}{
					"type": "FOOTSTEP",
					"dir": dir,
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
	keys := []string{"WPN_SHOCK", "SURV_MEDKIT", "RECON_RADAR"}
	choice := keys[time.Now().UnixNano()%3]
	item := ItemDB[choice]
	item.UID = NewUID()
	gs.Entities[item.UID] = Entity{
		UID:   item.UID,
		Type:  EntityTypeItemDrop,
		Pos:   gs.Map.GetRandomSpawnPos(),
		State: 1, 
		Extra: item,
	}
}
