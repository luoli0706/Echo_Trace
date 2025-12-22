package logic

import (
	"log"
	"math"
	"sync"
	"time"
)

const (
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
		RespawnTimer: 5.0, // Increased Frequency
		PulseTimer:   15.0,
		GlobalEvents: make([]GlobalEvent, 0),
	}
}

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
	return p
}

func (gs *GameState) RemovePlayer(sessionID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	delete(gs.Players, sessionID)
}

func (gs *GameState) HandleInput(sessionID string, dir Vector2) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
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
	
	p, ok := gs.Players[sessionID]
	if !ok || !p.IsAlive { return }

	interactRange := 2.0
	var targetUID = ""
	
	for uid, e := range gs.Entities {
		if e.Type == EntityTypeMotor && e.State != 2 {
			if Distance(p.Pos, e.Pos) <= interactRange {
				targetUID = uid
				break
			}
		}
	}
	
	if targetUID != "" {
		p.ChannelingTargetUID = targetUID
		log.Printf("Player %s started fixing Motor %s", sessionID, targetUID)
	}
}

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
			if ent, ok := gs.Entities[p.ChannelingTargetUID]; ok && ent.Type == EntityTypeMotor {
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
			} else {
				p.ChannelingTargetUID = ""
			}
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
		// Higher cap
		if itemCount < 30 {
			gs.spawnRandomItemInternal()
		}
	}

	// 4. Physics
	playerRadius := 0.25
	for _, p := range gs.Players {
		if !p.IsAlive { continue }
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
		gs.PhaseTimer = 9999 
		gs.addEvent("PHASE_CHANGE", "Phase 2: Conflict! Fix 2 Motors to escape.")
		gs.spawnMotors(5)
	} else if gs.Phase == PhaseEscape {
		gs.startEscapePhase()
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
	if !ok { return nil }

	entSlice := make([]Entity, 0, len(gs.Entities))
	for _, e := range gs.Entities {
		entSlice = append(entSlice, e)
	}
	visiblePlayers, visibleEntities := gs.AOI.GetVisibleEntities(p, gs.Players, entSlice)

	// Radar Logic: Calculate Blips
	radarBlips := make([]Blip, 0)
	
	// Phase 2: Pulse Motors (First 5 seconds of the 15s cycle)
	isPulseActive := gs.Phase == PhaseConflict && gs.PulseTimer > 10.0
	
	if isPulseActive {
		for _, e := range gs.Entities {
			if e.Type == EntityTypeMotor {
				radarBlips = append(radarBlips, Blip{Type: "MOTOR", Pos: e.Pos})
			}
		}
	}
	// Phase 3: Always show Exit
	if gs.Phase == PhaseEscape {
		for _, e := range gs.Entities {
			if e.Type == EntityTypeExit {
				radarBlips = append(radarBlips, Blip{Type: "EXIT", Pos: e.Pos})
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
