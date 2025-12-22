package logic

import (
	"log"
	"math"
	"sync"
)

// GameState manages the world simulation
type GameState struct {
	Config   *GameConfig
	Map      *GameMap
	Players  map[string]*Player
	Entities []Entity
	AOI      *AOIManager
	Mutex    sync.RWMutex
}

func NewGameState(cfg *GameConfig) *GameState {
	m := NewGameMap(cfg.Map.Width, cfg.Map.Height, cfg.Map.WallDensity)
	return &GameState{
		Config:   cfg,
		Map:      m,
		Players:  make(map[string]*Player),
		Entities: make([]Entity, 0),
		AOI:      NewAOIManager(cfg.Map.Width, cfg.Map.Height),
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

// HandleAttack processes an attack request
func (gs *GameState) HandleAttack(attackerID string, targetID string) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	attacker, ok := gs.Players[attackerID]
	if !ok || !attacker.IsAlive {
		return false
	}

	// Basic Melee Attack Logic for MVP
	// 1. Find Target
	var target *Player
	if targetID != "" {
		target = gs.Players[targetID]
	} else {
		// Auto-target nearest in range/vision (Simplified)
		minDist := 2.0 // Melee range
		for _, p := range gs.Players {
			if p.SessionID != attackerID && p.IsAlive {
				d := Distance(attacker.Pos, p.Pos)
				if d < minDist && d <= attacker.ViewRadius {
					target = p
					minDist = d
				}
			}
		}
	}

	if target != nil {
		// 2. Deal Damage
		dmg := 25.0 // Hardcoded base damage for MVP
		target.HP -= dmg
		log.Printf("Player %s hit %s for %.1f dmg. Target HP: %.1f", attackerID, target.SessionID, dmg, target.HP)
		
		if target.HP <= 0 {
			target.HP = 0
			target.IsAlive = false
			log.Printf("Player %s KILLED %s", attackerID, target.SessionID)
			// TODO: Drop items
		}
		return true
	}
	
	return false
}


// UpdateTick runs physics and logic (called every 50ms)
func (gs *GameState) UpdateTick(dt float64) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	playerRadius := 0.3 // Player size

	for _, p := range gs.Players {
		if !p.IsAlive {
			continue
		}

		// Movement Logic with improved Collision
		if p.TargetDir.X != 0 || p.TargetDir.Y != 0 {
			// Normalize dir
			len := math.Sqrt(p.TargetDir.X*p.TargetDir.X + p.TargetDir.Y*p.TargetDir.Y)
			if len > 0 {
				p.TargetDir.X /= len
				p.TargetDir.Y /= len
			}

			// Try X Movement
			newX := p.Pos.X + p.TargetDir.X*p.MoveSpeed*dt
			if gs.isWalkableWithRadius(newX, p.Pos.Y, playerRadius) {
				p.Pos.X = newX
			}

			// Try Y Movement
			newY := p.Pos.Y + p.TargetDir.Y*p.MoveSpeed*dt
			if gs.isWalkableWithRadius(p.Pos.X, newY, playerRadius) {
				p.Pos.Y = newY
			}
		}
	}
}

// isWalkableWithRadius checks if a circle is in a valid position
func (gs *GameState) isWalkableWithRadius(x, y, r float64) bool {
	// Check center
	if !gs.Map.IsWalkable(x, y) {
		return false
	}
	// Check corners (approximate circle with box)
	if !gs.Map.IsWalkable(x+r, y) || !gs.Map.IsWalkable(x-r, y) ||
	   !gs.Map.IsWalkable(x, y+r) || !gs.Map.IsWalkable(x, y-r) {
		return false
	}
	return true
}

// GetSnapshot generates the view for a specific player
func (gs *GameState) GetSnapshot(sessionID string) map[string]interface{} {
	gs.Mutex.RLock()
	defer gs.Mutex.RUnlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return nil
	}

	visiblePlayers, visibleEntities := gs.AOI.GetVisibleEntities(p, gs.Players, gs.Entities)

	return map[string]interface{}{
		"timestamp": 0,
		"self":      p,
		"vision": map[string]interface{}{
			"players":  visiblePlayers,
			"entities": visibleEntities,
		},
	}
}