package logic

import (
	"log"
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
	if p, ok := gs.Players[sessionID]; ok {
		p.TargetDir = dir
	}
}

// UpdateTick runs physics and logic (called every 50ms)
func (gs *GameState) UpdateTick(dt float64) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	for _, p := range gs.Players {
		if !p.IsAlive {
			continue
		}

		// Simple Movement: Pos += Dir * Speed * dt
		if p.TargetDir.X != 0 || p.TargetDir.Y != 0 {
			newX := p.Pos.X + p.TargetDir.X*p.MoveSpeed*dt
			newY := p.Pos.Y + p.TargetDir.Y*p.MoveSpeed*dt

			// Check Collision
			if gs.Map.IsWalkable(newX, newY) {
				p.Pos.X = newX
				p.Pos.Y = newY
			} else {
				// Slide along walls (simplified: try X only, then Y only)
				if gs.Map.IsWalkable(newX, p.Pos.Y) {
					p.Pos.X = newX
				} else if gs.Map.IsWalkable(p.Pos.X, newY) {
					p.Pos.Y = newY
				}
			}
		}
	}
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
		"timestamp": 0, // TODO: Real timestamp
		"self":      p,
		"vision": map[string]interface{}{
			"players":  visiblePlayers,
			"entities": visibleEntities,
		},
	}
}
