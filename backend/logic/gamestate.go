package logic

import (
	"log"
	"math"
	"sync"
	"time"
)

// Game Phases
const (
	PhaseSearch   = 1
	PhaseConflict = 2
	PhaseEscape   = 3
	PhaseEnded    = 4
)

// Global Event Struct
type GlobalEvent struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
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
	GlobalEvents []GlobalEvent
	Mutex        sync.RWMutex
}

func NewGameState(cfg *GameConfig) *GameState {
	m := NewGameMap(cfg.Map.Width, cfg.Map.Height, cfg.Map.WallDensity)
	return &GameState{
		Config:       cfg,
		Map:          m,
		Players:      make(map[string]*Player),
		Entities:     make(map[string]Entity),
		AOI:          NewAOIManager(cfg.Map.Width, cfg.Map.Height),
		Phase:        PhaseSearch,
		PhaseTimer:   float64(cfg.Phases.Phase1.Duration),
		RespawnTimer: 10.0,
	}
}

// AddPlayer spawns a new player
func (gs *GameState) AddPlayer(sessionID string) *Player {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	spawnPos := gs.Map.GetRandomSpawnPos()
	p := &Player{
		SessionID:  sessionID,
		Pos:        spawnPos,
		HP:         100,
		MaxHP:      100,
		MoveSpeed:  gs.Config.Gameplay.BaseMoveSpeed,
		ViewRadius: gs.Config.Gameplay.BaseViewRadius,
		IsAlive:    true,
		Inventory:  make([]Item, 0),
	}
	gs.Players[sessionID] = p
	log.Printf("Player %s spawned at %v", sessionID, spawnPos)
	return p
}

// RemovePlayer cleans up
func (gs *GameState) RemovePlayer(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	delete(gs.Players, sessionID)
}

// HandleInput updates player target direction
func (gs *GameState) HandleInput(sessionID string, dir Vector2) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	if p, ok := gs.Players[sessionID]; ok && p.IsAlive {
		p.TargetDir = dir
	}
}

// UpdateTick runs physics and logic
func (gs *GameState) UpdateTick(dt float64) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	// 1. Phase Logic
	if gs.Phase != PhaseEnded {
		gs.PhaseTimer -= dt
		if gs.PhaseTimer <= 0 {
			gs.nextPhase()
		}
	}

	// 2. Item Respawn Logic
	gs.RespawnTimer -= dt
	if gs.RespawnTimer <= 0 {
		gs.RespawnTimer = 10.0 // Check every 10s
		// Count items
		itemCount := 0
		for _, e := range gs.Entities {
			if e.Type == EntityTypeItemDrop {
				itemCount++
			}
		}
		
		// Refill up to 20 items
		if itemCount < 20 {
			gs.spawnRandomItemInternal() // Use internal helper to avoid double lock if needed, or careful calling
			log.Printf("Respawned item. Total: %d", itemCount+1)
		}
	}

	// 3. Physics
	playerRadius := 0.25 // Smoother collision
	for _, p := range gs.Players {
		if !p.IsAlive {
			continue
		}
		if p.TargetDir.X != 0 || p.TargetDir.Y != 0 {
			len := math.Sqrt(p.TargetDir.X*p.TargetDir.X + p.TargetDir.Y*p.TargetDir.Y)
			if len > 0 {
				p.TargetDir.X /= len
				p.TargetDir.Y /= len
			}
			newX := p.Pos.X + p.TargetDir.X*p.MoveSpeed*dt
			if gs.isWalkableWithRadius(newX, p.Pos.Y, playerRadius) {
				p.Pos.X = newX
			}
			newY := p.Pos.Y + p.TargetDir.Y*p.MoveSpeed*dt
			if gs.isWalkableWithRadius(p.Pos.X, newY, playerRadius) {
				p.Pos.Y = newY
			}
		}
	}
}

func (gs *GameState) nextPhase() {
	gs.Phase++
	if gs.Phase == PhaseConflict {
		gs.PhaseTimer = float64(gs.Config.Phases.Phase2.Duration)
		gs.addEvent("PHASE_CHANGE", "Phase 2: Conflict Started! Motors Active.")
	} else if gs.Phase == PhaseEscape {
		gs.PhaseTimer = 9999 
		gs.addEvent("PHASE_CHANGE", "Phase 3: Escape! Find the exit.")
	}
}

func (gs *GameState) addEvent(t, msg string) {
	gs.GlobalEvents = append(gs.GlobalEvents, GlobalEvent{Type: t, Msg: msg})
	if len(gs.GlobalEvents) > 5 {
		gs.GlobalEvents = gs.GlobalEvents[1:]
	}
}

func (gs *GameState) isWalkableWithRadius(x, y, r float64) bool {
	if !gs.Map.IsWalkable(x, y) { return false }
	if !gs.Map.IsWalkable(x+r, y) || !gs.Map.IsWalkable(x-r, y) ||
	   !gs.Map.IsWalkable(x, y+r) || !gs.Map.IsWalkable(x, y-r) {
		return false
	}
	return true
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

	visiblePlayers, visibleEntities := gs.AOI.GetVisibleEntities(p, gs.Players, entSlice)

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
	}
}

// spawnRandomItemInternal is a helper that doesn't lock, assuming caller has lock
func (gs *GameState) spawnRandomItemInternal() {
	keys := []string{"WPN_SHOCK", "SURV_MEDKIT", "RECON_RADAR"}
	choice := keys[time.Now().UnixNano()%3]
	
	// Access ItemDB from item_system.go (assuming it's exported or in same package)
	item := ItemDB[choice]
	item.UID = NewUID()
	
	entity := Entity{
		UID:   item.UID,
		Type:  EntityTypeItemDrop,
		Pos:   gs.Map.GetRandomSpawnPos(),
		State: 1, 
		Extra: item,
	}
	gs.Entities[entity.UID] = entity
}