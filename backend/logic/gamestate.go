package logic

import (
	"echo_trace_server/storage"
	"fmt"
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
	TotalKills   int
	Mutex        sync.RWMutex

	// Phase 0 State
	StartTime time.Time
}

func NewGameState(cfg *GameConfig) *GameState {
	m := NewGameMap(cfg.Map.Width, cfg.Map.Height, cfg.Map.WallDensity)
	respawn := 5.0
	if cfg != nil && cfg.Items.RespawnIntervalSec > 0 {
		respawn = cfg.Items.RespawnIntervalSec
	}
	pulse := 15.0
	if cfg != nil && cfg.Phases.Phase2.PulseIntervalSec > 0 {
		pulse = float64(cfg.Phases.Phase2.PulseIntervalSec)
	}
	return &GameState{
		Config:       cfg,
		Map:          m,
		Players:      make(map[string]*Player),
		Entities:     make(map[string]Entity),
		AOI:          NewAOIManager(cfg.Map.Width, cfg.Map.Height),
		Phase:        PhaseInit, // Start in Init Phase
		PhaseTimer:   float64(cfg.Phases.Phase1.Duration),
		RespawnTimer: respawn,
		PulseTimer:   pulse,
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
	if p.Armor > p.MaxArmor {
		p.Armor = p.MaxArmor
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

	// Start condition: Min players reached
	minPlayers := 1
	if gs.Config != nil && gs.Config.Server.MinPlayersToStart > 0 {
		minPlayers = gs.Config.Server.MinPlayersToStart
	}
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
	initial := 15
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
		gs.refreshShopForPlayer(p, phaseIdx)
		// New phase => free refresh available again.
		p.ShopFreeRefreshUsedPhase = 0
	}
}

func (gs *GameState) refreshShopForPlayer(p *Player, phaseIdx int) {
	if p == nil {
		return
	}
	stock := gs.generateShopStock(phaseIdx, p.Tactic)
	prices := make([]int, 0, len(stock))
	types := make([]string, 0, len(stock))
	for _, itemID := range stock {
		prices = append(prices, gs.shopItemCost(itemID))
		types = append(types, gs.shopItemType(itemID))
	}
	p.ShopStock = stock
	p.ShopPrices = prices
	p.ShopTypes = types
}

func (gs *GameState) shopItemType(itemID string) string {
	for _, it := range ItemDB {
		if it.ID == itemID {
			return it.Type
		}
	}
	return ""
}

func (gs *GameState) shopItemCost(itemID string) int {
	// Cost Logic: use item.Value (fallback: 50 * tier)
	for _, it := range ItemDB {
		if it.ID == itemID {
			cost := it.Value
			if cost <= 0 {
				cost = 50 * it.Tier
			}
			if cost < 0 {
				cost = 0
			}
			return cost
		}
	}
	return 0
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
	merchantRange := 3.0
	if gs.Config != nil && gs.Config.Gameplay.MerchantInteractRange > 0 {
		merchantRange = gs.Config.Gameplay.MerchantInteractRange
	}
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= merchantRange {
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
	merchantRange := 3.0
	if gs.Config != nil && gs.Config.Gameplay.MerchantInteractRange > 0 {
		merchantRange = gs.Config.Gameplay.MerchantInteractRange
	}
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= merchantRange {
			nearMerchant = true
			break
		}
	}

	if !nearMerchant {
		p.ClientMsg = "You must be near the Merchant."
		return
	}

	// Must be in current shop stock.
	allowed := false
	stockIdx := -1
	for i, sid := range p.ShopStock {
		if sid == itemID {
			allowed = true
			stockIdx = i
			break
		}
	}
	if !allowed {
		p.ClientMsg = "Item not available in your shop."
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
		p.ClientMsg = "Unknown item."
		return
	}

	cost := 0
	if stockIdx >= 0 && stockIdx < len(p.ShopPrices) {
		cost = p.ShopPrices[stockIdx]
	}
	if cost <= 0 {
		cost = targetItem.Value
		if cost <= 0 {
			cost = 50 * targetItem.Tier
		}
	}

	if len(p.Inventory) >= p.InventoryCap {
		p.ClientMsg = "Inventory is full."
		return
	}
	if p.Funds < cost {
		p.ClientMsg = fmt.Sprintf("Not enough funds ($%d needed).", cost)
		return
	}

	if p.Funds >= cost {
		p.Funds -= cost

		newItem := targetItem
		newItem.UID = NewUID()
		p.Inventory = append(p.Inventory, newItem)

		gs.RecalculateStats(p)
		p.ClientMsg = ""
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

	// Must be near merchant
	merchantRange := 3.0
	if gs.Config != nil && gs.Config.Gameplay.MerchantInteractRange > 0 {
		merchantRange = gs.Config.Gameplay.MerchantInteractRange
	}
	nearMerchant := false
	for _, e := range gs.Entities {
		if e.Type == EntityTypeMerchant && Distance(p.Pos, e.Pos) <= merchantRange {
			nearMerchant = true
			break
		}
	}
	if !nearMerchant {
		p.ClientMsg = "You must be near the Merchant."
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
			p.ClientMsg = fmt.Sprintf("Not enough funds to refresh ($%d needed).", cost)
			return
		}
		p.Funds -= cost
	}

	gs.refreshShopForPlayer(p, phaseIdx)
	p.ClientMsg = ""
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
	baseArmor := gs.Config.Combat.BaseArmor
	if baseArmor <= 0 {
		baseArmor = 50.0
	}
	p := &Player{
		SessionID:     sessionID,
		Name:          "Unknown",
		Pos:           spawnPos,
		LookDir:       Vector2{X: 1, Y: 0},
		HP:            baseHP,
		MaxHP:         baseHP,
		Armor:         baseArmor,
		MaxArmor:      baseArmor,
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

	p.InventoryCap = invCap + invBonus + p.ExtraInvCap
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

	// Overweight view reduce rule.
	if gs.Config != nil && gs.Config.Gameplay.WeightThresholdViewReduce > 0 {
		thr := gs.Config.Gameplay.WeightThresholdViewReduce
		if thr < 1.0 && ratio > thr {
			// Linearly reduce up to 50% as ratio approaches 1.0.
			k := (ratio - thr) / (1.0 - thr)
			if k < 0 {
				k = 0
			}
			if k > 1 {
				k = 1
			}
			reduce := 1.0 - 0.5*k
			p.ViewRadius *= reduce
		}
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

func (gs *GameState) HandleFire(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive {
		return
	}

	// Check if Phase Shifted (Invincible) -> Cannot Shoot
	if time.Now().Before(p.BuffInvincibleUntil) {
		p.ClientMsg = "System Locked (Phase Shift)"
		return
	}
	
	// Check Invisible -> Shoot breaks invisibility
	if time.Now().Before(p.BuffInvisibleUntil) {
		p.BuffInvisibleUntil = time.Time{} // Clear buff
		p.ClientMsg = "Stealth Broken"
	}

	// Rate Limit
	reloadSec := 0.5
	if gs.Config != nil && gs.Config.Combat.ReloadTimeSec > 0 {
		reloadSec = gs.Config.Combat.ReloadTimeSec
	}
	nextFire := p.LastFireTime.Add(time.Duration(reloadSec * float64(time.Second)))
	if time.Now().Before(nextFire) {
		return // Reloading
	}
	p.LastFireTime = time.Now()

	// Params
	speed := 15.0 // Slightly faster for better feel
	damage := 20.0
	radius := 0.3
	if gs.Config != nil {
		if gs.Config.Combat.BulletDamage > 0 {
			damage = gs.Config.Combat.BulletDamage
		}
	}
	
	// Ammo Logic
	ammoType := p.AmmoType
	if p.AmmoCount > 0 {
		p.AmmoCount--
		if p.AmmoCount <= 0 {
			p.AmmoType = "" // Revert to normal after using up ammo
		}
	} else {
		ammoType = "" // Normal ammo
	}

	// RAILGUN: Hitscan, Wall Penetration
	if ammoType == "RAILGUN" {
		gs.fireRailgun(p, 75.0) // 75 Damage
		return
	}

	// PROJECTILE: Normal, AP, Bounce
	uid := NewUID()
	dir := p.LookDir
	l := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)
	if l > 0 {
		dir.X /= l
		dir.Y /= l
	}
	spawnPos := Vector2{
		X: p.Pos.X + dir.X*0.6,
		Y: p.Pos.Y + dir.Y*0.6,
	}
	
	// Modifiers based on AmmoType
	lifetimeSec := 5.0
	defaultBounces := 0
	if gs.Config != nil {
		if gs.Config.Combat.ProjectileLifetimeSec > 0 {
			lifetimeSec = gs.Config.Combat.ProjectileLifetimeSec
		}
		defaultBounces = gs.Config.Combat.DefaultBounces
	}

	projData := ProjectileData{
		OwnerID:     sessionID,
		Velocity:    Vector2{X: dir.X * speed, Y: dir.Y * speed},
		Damage:      damage,
		Radius:      radius,
		Lifetime:    time.Now().Add(time.Duration(lifetimeSec * float64(time.Second))),
		BouncesLeft: defaultBounces,
	}
	
	if ammoType == "AP" {
		projData.Damage = 40 
	} else if ammoType == "BOUNCE" {
		projData.Damage = 25
		projData.BouncesLeft += 100 // Bounce ammo adds lots of bounces
	}

	gs.Entities[uid] = Entity{
		UID:   uid,
		Type:  EntityTypeProjectile,
		Pos:   spawnPos,
		State: 1, 
		Extra: projData,
	}
}

func (gs *GameState) fireRailgun(p *Player, damage float64) {
	// Instant hitscan, infinite range, ignores walls (penetration decay?)
	// For Alpha: infinite range, hits first target, ignores walls.
	// Visual: Add a "LASER" event?
	
	bestTarget := (*Player)(nil)
	bestDist := 999.0
	
	for _, t := range gs.Players {
		if t.SessionID == p.SessionID || !t.IsAlive { continue }
		
		// Line vs Circle check
		// P + t*Dir = C
		// (C - P) dot Dir_perp ... simple distance check to ray
		toTarget := Vector2{X: t.Pos.X - p.Pos.X, Y: t.Pos.Y - p.Pos.Y}
		distAlong := toTarget.X*p.LookDir.X + toTarget.Y*p.LookDir.Y
		if distAlong < 0 { continue } // Behind
		
		perpDist2 := (toTarget.X*toTarget.X + toTarget.Y*toTarget.Y) - (distAlong*distAlong)
		if perpDist2 < 0.5*0.5 { // Hit radius
			if distAlong < bestDist {
				bestDist = distAlong
				bestTarget = t
			}
		}
	}
	
	if bestTarget != nil {
		gs.applyDamageLocked(p, bestTarget, damage)
		gs.addEvent("RAILGUN", fmt.Sprintf("Railgun fired by %s!", p.Name))
	} else {
		gs.addEvent("RAILGUN", fmt.Sprintf("Railgun fired by %s (Miss)", p.Name))
	}
}

func (gs *GameState) hasLOS(start, end Vector2) bool {
	// Simple step-based LOS check
	dist := Distance(start, end)
	if dist < 1.0 {
		return true
	}
	steps := int(dist * 2) // Check every 0.5 units
	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		checkPos := Vector2{
			X: start.X + (end.X-start.X)*t,
			Y: start.Y + (end.Y-start.Y)*t,
		}
		if !gs.Map.IsWalkable(checkPos.X, checkPos.Y) {
			return false
		}
	}
	return true
}

func (gs *GameState) applyDamageLocked(attacker, victim *Player, damage float64) {
	// Check Invincibility
	if time.Now().Before(victim.BuffInvincibleUntil) {
		return // No Damage
	}

	// Armor reduces damage first
	if victim.Armor > 0 {
		if victim.Armor >= damage {
			victim.Armor -= damage
			damage = 0
		} else {
			damage -= victim.Armor
			victim.Armor = 0
		}
	}

	if damage > 0 {
		victim.HP -= damage
	}

	if victim.HP <= 0 {
		victim.HP = 0
		victim.IsAlive = false // Explicitly set to false
		if !victim.IsDead { // Prevent multiple death calls
			attacker.Kills++
			gs.TotalKills++
			gs.addEvent("PLAYER_KILLED", fmt.Sprintf("%s was eliminated by %s!", victim.Name, attacker.Name))
			gs.handleDeath(victim)
			gs.checkPhaseThresholds()
		}
	}
}

func (gs *GameState) handleDeath(p *Player) {
	if p == nil {
		return
	}
	if p.IsDead {
		return
	}
	p.IsAlive = false
	p.IsDead = true
	p.RespawnTimer = time.Now().Add(5 * time.Second) // Hardcoded 5s or config

	// Drop Inventory
	for _, item := range p.Inventory {
		uid := NewUID()
		ent := Entity{
			UID:   uid,
			Type:  EntityTypeItemDrop,
			Pos:   p.Pos,
			State: 1,
			Extra: item,
		}
		gs.Entities[uid] = ent
	}
	p.Inventory = []Item{}
	name := p.Name
	if name == "" || name == "Unknown" {
		name = p.SessionID
	}
	gs.addEvent("DEATH", fmt.Sprintf("%s 死亡 (5s 后重生)", name))
	log.Printf("Player %s died. Respawning in 5s...", p.SessionID)
}

func (gs *GameState) checkPhaseThresholds() {
	if gs.Config == nil {
		return
	}
	ts := gs.Config.Phases.Thresholds
	if gs.TotalKills >= ts.EndGameKills && gs.Phase != PhaseEnded {
		gs.endGame()
	} else if gs.TotalKills >= ts.Phase3Kills && gs.Phase < PhaseEscape {
		gs.startEscapePhase()
	} else if gs.TotalKills >= ts.Phase2Kills && gs.Phase < PhaseConflict {
		gs.nextPhase()
	}
}

func (gs *GameState) endGame() {
	gs.Phase = PhaseEnded
	gs.PhaseTimer = 60.0 // 1 minute until room close
	gs.addEvent("GAME_OVER", "The match has ended! Check the scoreboard.")
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

	// Overweight immobilize rule: if weight ratio exceeds threshold, block interactions (e.g., motor decipher).
	if gs.Config != nil && gs.Config.Gameplay.WeightThresholdImmobilize > 0 {
		ratio := 0.0
		if p.MaxWeight > 0 {
			ratio = p.Weight / p.MaxWeight
		}
		if ratio >= gs.Config.Gameplay.WeightThresholdImmobilize {
			p.ClientMsg = "Too heavy to interact."
			return
		}
	}

	interactRange := 2.0
	if gs.Config != nil && gs.Config.Gameplay.InteractRange > 0 {
		interactRange = gs.Config.Gameplay.InteractRange
	}
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
			extractSec := 3.0
			if gs.Config != nil && gs.Config.Phases.Phase3.ExtractionChannelTimeSec > 0 {
				extractSec = gs.Config.Phases.Phase3.ExtractionChannelTimeSec
			}
			p.ExtractionTimer = extractSec
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
			if p == nil { continue }
			if p.Disconnected && !p.DisconnectedAt.IsZero() && now.Sub(p.DisconnectedAt) > deadline {
				gs.addEvent("PLAYER_KICK", fmt.Sprintf("%s disconnected too long and was removed.", p.Name))
				gs.removePlayerLocked(sid)
			}
		}
	}
	
	// Respawn Logic
	now := time.Now()
	for _, p := range gs.Players {
		if p.IsDead && now.After(p.RespawnTimer) {
			p.IsDead = false
			p.IsAlive = true
			p.HP = p.MaxHP
			p.Armor = p.MaxArmor
			// Spawn Logic: Try positions far from death point (p.Pos is current pos)
			p.Pos = gs.Map.GetFarSpawnPos(p.Pos, 15.0)
			gs.addEvent("RESPAWN", fmt.Sprintf("%s 重返战场!", p.Name))
			log.Printf("RESPAWN: Player %s revived at %v", p.Name, p.Pos)
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
	} else {
		gs.PhaseTimer -= dt
		if gs.PhaseTimer <= 0 {
			// Room close logic could go here, for now just stay at 0
			gs.PhaseTimer = 0
		}
	}

	// 1.1 Motor Pulse Logic
	if gs.Phase == PhaseConflict {
		gs.PulseTimer -= dt
		if gs.PulseTimer <= 0 {
			interval := 15.0
			if gs.Config != nil && gs.Config.Phases.Phase2.PulseIntervalSec > 0 {
				interval = float64(gs.Config.Phases.Phase2.PulseIntervalSec)
			}
			gs.PulseTimer = interval
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
				decipherSec := 20
				if gs.Config != nil && gs.Config.Phases.Phase2.MotorDecipherTimeSec > 0 {
					decipherSec = gs.Config.Phases.Phase2.MotorDecipherTimeSec
				}
				speed := 100.0 / float64(decipherSec)
				data.Progress += speed * dt
				if data.Progress >= data.MaxProgress {
					data.Progress = data.MaxProgress
					ent.State = 2
					gs.MotorsFixed++
					gs.addEvent("MOTOR_FIXED", "A Motor has been repaired!")
					p.ChannelingTargetUID = ""
					req := 2
					if gs.Config != nil {
						req = gs.Config.Phases.Phase2.MotorsRequiredToOpenExit
					}
					if req < 0 {
						req = 0
					}
					if gs.MotorsFixed >= req && gs.Phase == PhaseConflict {
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
	// Circle radius used for collision (walls + player-player)
	playerRadius := 0.25
	if gs.Config != nil && gs.Config.Gameplay.PlayerCollisionRadius > 0 {
		playerRadius = gs.Config.Gameplay.PlayerCollisionRadius
	}
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
	// Resolve player-vs-player overlaps after wall collision.
	gs.resolvePlayerOverlaps(playerRadius)

	// 5. Update Projectiles
	gs.updateProjectiles(dt)
}

func (gs *GameState) updateProjectiles(dt float64) {
	toRemove := []string{}
	now := time.Now()

	for uid, e := range gs.Entities {
		if e.Type != EntityTypeProjectile {
			continue
		}

		data, ok := e.Extra.(ProjectileData)
		if !ok {
			toRemove = append(toRemove, uid)
			continue
		}
		
		// Check Lifetime
		if now.After(data.Lifetime) {
			toRemove = append(toRemove, uid)
			continue
		}

		// Move
		nextPos := Vector2{
			X: e.Pos.X + data.Velocity.X*dt,
			Y: e.Pos.Y + data.Velocity.Y*dt,
		}

		// 1. Check Wall Collision (with bounce)
		if gs.checkCollision(nextPos, data.Radius) {
			if data.BouncesLeft > 0 {
				// Calculate Bounce
				// Try move X
				hitX := gs.checkCollision(Vector2{X: nextPos.X, Y: e.Pos.Y}, data.Radius)
				hitY := gs.checkCollision(Vector2{X: e.Pos.X, Y: nextPos.Y}, data.Radius)
				
				if hitX {
					data.Velocity.X = -data.Velocity.X
				}
				if hitY {
					data.Velocity.Y = -data.Velocity.Y
				}
				if !hitX && !hitY {
					data.Velocity.X = -data.Velocity.X
					data.Velocity.Y = -data.Velocity.Y
				}
				
				data.BouncesLeft--
				e.Extra = data
				
				// Move out of wall in same frame to avoid "stutter"
				nextPos = Vector2{
					X: e.Pos.X + data.Velocity.X*dt,
					Y: e.Pos.Y + data.Velocity.Y*dt,
				}
				// If still colliding after reflection, just stay put to avoid tunneling
				if gs.checkCollision(nextPos, data.Radius) {
					nextPos = e.Pos
				}
			} else {
				// No bounces left, destroy
				toRemove = append(toRemove, uid)
				continue
			}
		}

		// 2. Check Player Collision
		hitPlayer := false
		for _, p := range gs.Players {
			if !p.IsAlive || p.SessionID == data.OwnerID {
				continue
			}
			// Simple Circle-Circle
			dist := Distance(nextPos, p.Pos)
			// Player hit radius approx 0.4
			if dist < (data.Radius + 0.4) {
				// Hit!
				owner, hasOwner := gs.Players[data.OwnerID]
				if hasOwner {
					gs.applyDamageLocked(owner, p, data.Damage)
				}
				hitPlayer = true
				break
			}
		}

		if hitPlayer {
			toRemove = append(toRemove, uid)
		} else {
			// Update pos
			e.Pos = nextPos
			gs.Entities[uid] = e
		}
	}

	for _, uid := range toRemove {
		delete(gs.Entities, uid)
	}
}

func (gs *GameState) resolvePlayerOverlaps(playerRadius float64) {
	// Simple circle-circle separation. Player count is small (<= max players), so O(n^2) is fine.
	players := make([]*Player, 0, len(gs.Players))
	for _, p := range gs.Players {
		if p == nil || !p.IsAlive || p.Disconnected {
			continue
		}
		players = append(players, p)
	}

	minDist := playerRadius * 2.0
	minDist2 := minDist * minDist
	// A few relaxation iterations makes separation stable.
	for iter := 0; iter < 3; iter++ {
		movedAny := false
		for i := 0; i < len(players); i++ {
			for j := i + 1; j < len(players); j++ {
				a := players[i]
				b := players[j]
				dx := b.Pos.X - a.Pos.X
				dy := b.Pos.Y - a.Pos.Y
				d2 := dx*dx + dy*dy
				if d2 >= minDist2 {
					continue
				}

				// Deterministic direction if perfectly overlapping.
				if d2 < 1e-9 {
					if a.SessionID < b.SessionID {
						dx, dy = 1.0, 0.0
					} else {
						dx, dy = -1.0, 0.0
					}
					d2 = 1.0
				}

				d := math.Sqrt(d2)
				nx := dx / d
				ny := dy / d
				push := (minDist - d) * 0.5
				if push <= 0 {
					continue
				}

				newA := Vector2{X: a.Pos.X - nx*push, Y: a.Pos.Y - ny*push}
				newB := Vector2{X: b.Pos.X + nx*push, Y: b.Pos.Y + ny*push}

				// Avoid pushing into walls. If one side is blocked, push the other more.
				aBlocked := gs.checkCollision(newA, playerRadius)
				bBlocked := gs.checkCollision(newB, playerRadius)
				switch {
				case !aBlocked && !bBlocked:
					a.Pos = newA
					b.Pos = newB
					movedAny = true
				case !aBlocked && bBlocked:
					newA2 := Vector2{X: a.Pos.X - nx*(2.0*push), Y: a.Pos.Y - ny*(2.0*push)}
					if !gs.checkCollision(newA2, playerRadius) {
						a.Pos = newA2
						movedAny = true
					}
				case aBlocked && !bBlocked:
					newB2 := Vector2{X: b.Pos.X + nx*(2.0*push), Y: b.Pos.Y + ny*(2.0*push)}
					if !gs.checkCollision(newB2, playerRadius) {
						b.Pos = newB2
						movedAny = true
					}
				default:
					// Both blocked by walls; leave as-is.
				}
			}
		}
		if !movedAny {
			break
		}
	}
}

func (gs *GameState) nextPhase() {
	gs.Phase++
	if gs.Phase == PhaseConflict {
		// Phase 2 ends on whichever comes first:
		// - the phase timer expires, OR
		// - enough motors are repaired to open the exit.
		p2 := 180
		if gs.Config != nil && gs.Config.Phases.Phase2.Duration > 0 {
			p2 = gs.Config.Phases.Phase2.Duration
		}
		gs.PhaseTimer = float64(p2)
		interval := 15.0
		if gs.Config != nil && gs.Config.Phases.Phase2.PulseIntervalSec > 0 {
			interval = float64(gs.Config.Phases.Phase2.PulseIntervalSec)
		}
		gs.PulseTimer = interval // Ensure immediate pulse window on start
		req := 2
		if gs.Config != nil {
			req = gs.Config.Phases.Phase2.MotorsRequiredToOpenExit
		}
		if req < 0 {
			req = 0
		}
		gs.addEvent("PHASE_CHANGE", fmt.Sprintf("Phase 2: Conflict! Fix %d Motors to escape.", req))
		mcount := 5
		if gs.Config != nil && gs.Config.Phases.Phase2.MotorsSpawnCount >= 0 {
			mcount = gs.Config.Phases.Phase2.MotorsSpawnCount
		}
		gs.spawnMotors(mcount)
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
	dur := 120
	if gs.Config != nil && gs.Config.Phases.Phase3.DurationSec > 0 {
		dur = gs.Config.Phases.Phase3.DurationSec
	}
	gs.PhaseTimer = float64(dur)
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
	cap := 5
	if gs.Config != nil && gs.Config.Server.GlobalEventsMax > 0 {
		cap = gs.Config.Server.GlobalEventsMax
	}
	if len(gs.GlobalEvents) > cap {
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
		fov := 90.0
		if gs.Config != nil && gs.Config.Gameplay.FOVDegrees > 0 {
			fov = gs.Config.Gameplay.FOVDegrees
		}
		visiblePlayers, visibleEntities = gs.AOI.GetVisibleEntities(p, gs.Map, gs.Players, entSlice, fov)
	}

	// Radar Logic: Calculate Blips
	radarBlips := make([]Blip, 0)

	// Phase 2: Pulse Motors (active window is configurable)
	pulseActiveWindow := 5.0
	pulseInterval := 15.0
	if gs.Config != nil {
		if gs.Config.Phases.Phase2.PulseActiveWindowSec > 0 {
			pulseActiveWindow = float64(gs.Config.Phases.Phase2.PulseActiveWindowSec)
		}
		if gs.Config.Phases.Phase2.PulseIntervalSec > 0 {
			pulseInterval = float64(gs.Config.Phases.Phase2.PulseIntervalSec)
		}
	}
	startThreshold := pulseInterval - pulseActiveWindow
	if startThreshold < 0 {
		startThreshold = 0
	}
	isPulseActive := gs.Phase == PhaseConflict && gs.PulseTimer > startThreshold

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
		if other.SessionID == sessionID || !other.IsAlive {
			continue
		}
		// Silent buff removed in refactor. Footsteps always emitted unless logic changes.
		
		isMoving := other.TargetDir.X != 0 || other.TargetDir.Y != 0
		if isMoving {
			dist := Distance(p.Pos, other.Pos)
			hearRadius := p.HearRadius
			if gs.Config != nil && gs.Config.Gameplay.WeightThresholdNoiseDouble > 0 {
				thr := gs.Config.Gameplay.WeightThresholdNoiseDouble
				ratio := 0.0
				if other.MaxWeight > 0 {
					ratio = other.Weight / other.MaxWeight
				}
				if ratio >= thr {
					hearRadius *= 2.0
				}
			}
			if dist <= hearRadius {
				dir := Vector2{X: other.Pos.X - p.Pos.X, Y: other.Pos.Y - p.Pos.Y}
				len := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)
				if len > 0 {
					dir.X /= len
					dir.Y /= len
				}
				
				intensity := 1.0 - (dist / hearRadius)
				if intensity < 0 {
					intensity = 0
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
		"timestamp":   0,
		"phase":       gs.Phase,
		"time_left":   gs.PhaseTimer,
		"total_kills": gs.TotalKills,
		"events":      gs.GlobalEvents,
		"self":        p,
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
