package logic

import (
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	AIStatePatrol      = 0
	AIStateCommandMove = 1
	AIStateCombat      = 2
)

// A* Node
type node struct {
	X, Y   int
	G, H   float64
	Parent *node
}

func (gs *GameState) UpdateNingByeAI(dt float64) {
	// Find the AI Entity
	var aiEnt Entity
	var aiData NingByeAI
	found := false

	for _, e := range gs.Entities {
		if e.Type == EntityTypeNingBye {
			aiEnt = e
			if data, ok := e.Extra.(NingByeAI); ok {
				aiData = data
				found = true
			}
			break
		}
	}

	if !found {
		return // No AI spawned yet
	}

	// Update State
	// 1. Sensing
	target := gs.scanForThreat(aiEnt.Pos, aiData.SensingRadius)
	now := time.Now()

	if aiData.State == AIStateCombat {
		// Combat Logic
		if target != nil {
			aiData.CombatTargetID = target.SessionID
			aiData.LostTargetTime = time.Time{} // Reset lost timer
			// Fire Logic
			gs.aiCombatBehavior(&aiEnt, &aiData, target, now)
		} else {
			// Target lost?
			if aiData.LostTargetTime.IsZero() {
				aiData.LostTargetTime = now
			}
			timeout := 3.0
			if gs.Config != nil && gs.Config.AI.NingBye.LostTargetTimeoutSec > 0 {
				timeout = gs.Config.AI.NingBye.LostTargetTimeoutSec
			}
			if now.Sub(aiData.LostTargetTime).Seconds() > timeout {
				log.Println("[AI] Lost target, returning to patrol.")
				aiData.State = AIStatePatrol
				aiData.CombatTargetID = ""
				// Resume command if valid? For now just Patrol.
				if aiData.TargetPos.X != 0 || aiData.TargetPos.Y != 0 {
					aiData.State = AIStateCommandMove
				}
			}
		}
	} else {
		// Patrol or Command
		if target != nil {
			log.Printf("[AI] Threat detected! Engaging %s", target.Name)
			aiData.State = AIStateCombat
			aiData.CombatTargetID = target.SessionID
		} else {
			// Movement Logic
			gs.aiMoveBehavior(&aiEnt, &aiData, dt)
		}
	}

	// Sync back
	aiEnt.Extra = aiData
	gs.Entities[aiEnt.UID] = aiEnt
}

func (gs *GameState) scanForThreat(pos Vector2, radius float64) *Player {
	var bestTarget *Player
	minDist := radius * radius

	// 使用四叉树加速查询附近玩家
	var candidatePlayers []*Player
	if gs.Quadtree != nil {
		nearbyEntities := gs.Quadtree.QueryRadius(pos.X, pos.Y, radius)
		for _, e := range nearbyEntities {
			if p, ok := gs.Players[e.UID]; ok && p.IsAlive && !p.IsExtracted {
				candidatePlayers = append(candidatePlayers, p)
			}
		}
	} else {
		// Fallback：遍历所有玩家
		for _, p := range gs.Players {
			if !p.IsAlive || p.IsExtracted {
				continue
			}
			candidatePlayers = append(candidatePlayers, p)
		}
	}

	for _, p := range candidatePlayers {
		// Check distance
		dist2 := (p.Pos.X-pos.X)*(p.Pos.X-pos.X) + (p.Pos.Y-pos.Y)*(p.Pos.Y-pos.Y)

		// Condition 1: Explicit Threat Flag (within sensing radius)
		isTarget := p.IsThreat && dist2 <= minDist

		// Condition 2: Too Close (Self-Defense)
		distThreshold := 3.0
		if gs.Config != nil && gs.Config.AI.NingBye.ThreatScanDistance > 0 {
			distThreshold = gs.Config.AI.NingBye.ThreatScanDistance
		}
		if !isTarget && dist2 <= distThreshold*distThreshold {
			isTarget = true
		}

		if isTarget {
			// Check Line of Sight
			if gs.hasLOS(pos, p.Pos) {
				bestTarget = p
				minDist = dist2
			}
		}
	}
	return bestTarget
}

func (gs *GameState) aiCombatBehavior(ent *Entity, data *NingByeAI, target *Player, now time.Time) {
	// Simple Aim and Fire
	// Check Reload
	reload := 0.5 // Default player
	if gs.Config != nil {
		reload = gs.Config.Combat.ReloadTimeSec
	}
	// AI has half reload time logic, but here we use Config.AI value directly
	if data.ReloadTimeSec > 0 {
		reload = data.ReloadTimeSec
	}

	if now.Sub(data.LastFireTime).Seconds() < reload {
		return
	}

	// Fire!
	data.LastFireTime = now

	// Create Projectile
	dir := Vector2{X: target.Pos.X - ent.Pos.X, Y: target.Pos.Y - ent.Pos.Y}
	dist := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)
	if dist > 0 {
		dir.X /= dist
		dir.Y /= dist
	}

	speed := 12.0
	radius := 0.4
	lifetimeSec := 5.0
	spawnOffset := 0.8
	if gs.Config != nil {
		if gs.Config.AI.NingBye.ProjectileSpeed > 0 {
			speed = gs.Config.AI.NingBye.ProjectileSpeed
		}
		if gs.Config.AI.NingBye.ProjectileRadius > 0 {
			radius = gs.Config.AI.NingBye.ProjectileRadius
		}
		if gs.Config.AI.NingBye.ProjectileLifetimeSec > 0 {
			lifetimeSec = gs.Config.AI.NingBye.ProjectileLifetimeSec
		}
		if gs.Config.AI.NingBye.ProjectileSpawnOffset > 0 {
			spawnOffset = gs.Config.AI.NingBye.ProjectileSpawnOffset
		}
	}
	uid := NewUID()

	proj := ProjectileData{
		OwnerID:          "AI_NINGBYE",
		Velocity:         Vector2{X: dir.X * speed, Y: dir.Y * speed},
		Damage:           data.Damage,
		Radius:           radius,
		Lifetime:         now.Add(time.Duration(lifetimeSec * float64(time.Second))),
		BouncesLeft:      0,
		ArmorPenetration: data.ArmorPenetration,
	}

	gs.Entities[uid] = Entity{
		UID:   uid,
		Type:  EntityTypeProjectile,
		Pos:   Vector2{X: ent.Pos.X + dir.X*spawnOffset, Y: ent.Pos.Y + dir.Y*spawnOffset}, // Spawn offset
		State: 1,
		Extra: proj,
	}
}

func (gs *GameState) aiMoveBehavior(ent *Entity, data *NingByeAI, dt float64) {
	// If Command Move, check if we have a path to TargetPos
	if data.State == AIStateCommandMove {
		// Check arrival
		arrivalDist := 0.5
		if gs.Config != nil && gs.Config.AI.NingBye.ArrivalDistance > 0 {
			arrivalDist = gs.Config.AI.NingBye.ArrivalDistance
		}
		dist := Distance(ent.Pos, data.TargetPos)
		if dist < arrivalDist {
			log.Println("[AI] Command target reached. Switching to Patrol.")
			data.State = AIStatePatrol
			data.TargetPos = Vector2{}
			data.PatrolPath = nil
			return
		}

		// Re-calc path if empty or stale (simple logic: if no path, calc one)
		if len(data.PatrolPath) == 0 {
			path := gs.findPath(ent.Pos, data.TargetPos)
			if len(path) > 0 {
				data.PatrolPath = path
			} else {
				// Cannot reach? Abort to patrol.
				data.State = AIStatePatrol
			}
		}
	} else {
		// Patrol: Random Walk along connected components
		// If path empty, pick a random walkable neighbor
		if len(data.PatrolPath) == 0 {
			// Get neighbors
			currX, currY := int(ent.Pos.X), int(ent.Pos.Y)
			candidates := []Vector2{}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := currX+dx, currY+dy
					if gs.Map.IsWalkable(float64(nx), float64(ny)) {
						candidates = append(candidates, Vector2{X: float64(nx) + 0.5, Y: float64(ny) + 0.5})
					}
				}
			}
			if len(candidates) > 0 {
				next := candidates[rand.Intn(len(candidates))]
				data.PatrolPath = []Vector2{next}
			}
		}
	}

	// Execute Path Move
	if len(data.PatrolPath) > 0 {
		target := data.PatrolPath[0]
		// Move towards target
		dir := Vector2{X: target.X - ent.Pos.X, Y: target.Y - ent.Pos.Y}
		dist := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y)

		moveDist := data.MoveSpeed * dt
		if gs.Config != nil && gs.Config.AI.NingBye.MoveSpeed > 0 {
			moveDist = gs.Config.AI.NingBye.MoveSpeed * dt
		}

		if dist <= moveDist {
			ent.Pos = target
			data.PatrolPath = data.PatrolPath[1:]
		} else {
			dir.X /= dist
			dir.Y /= dist
			ent.Pos.X += dir.X * moveDist
			ent.Pos.Y += dir.Y * moveDist
		}
	}
}

// Simple BFS/A* Pathfinding
func (gs *GameState) findPath(start, end Vector2) []Vector2 {
	sx, sy := int(start.X), int(start.Y)
	ex, ey := int(end.X), int(end.Y)

	// Boundary check
	if !gs.Map.IsWalkable(float64(ex), float64(ey)) {
		return nil
	}

	type Point struct{ X, Y int }
	queue := []Point{{sx, sy}}
	cameFrom := make(map[Point]Point)
	cameFrom[Point{sx, sy}] = Point{-1, -1}

	found := false

	// Max steps to prevent lag
	steps := 0
	maxSteps := 500
	if gs.Config != nil && gs.Config.AI.NingBye.PathfindingMaxSteps > 0 {
		maxSteps = gs.Config.AI.NingBye.PathfindingMaxSteps
	}
	for len(queue) > 0 && steps < maxSteps {
		curr := queue[0]
		queue = queue[1:]
		steps++

		if curr.X == ex && curr.Y == ey {
			found = true
			break
		}

		// Neighbors (4-dir for simpler pathing, or 8-dir)
		dirs := []Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		for _, d := range dirs {
			next := Point{curr.X + d.X, curr.Y + d.Y}
			if _, visited := cameFrom[next]; !visited {
				if gs.Map.IsWalkable(float64(next.X), float64(next.Y)) {
					cameFrom[next] = curr
					queue = append(queue, next)
				}
			}
		}
	}

	if !found {
		return nil
	}

	// Reconstruct
	path := []Vector2{}
	curr := Point{ex, ey}
	for curr.X != -1 {
		path = append([]Vector2{{X: float64(curr.X) + 0.5, Y: float64(curr.Y) + 0.5}}, path...)
		curr = cameFrom[curr]
		if curr.X == sx && curr.Y == sy {
			break
		}
	}
	return path
}

func (gs *GameState) SetPlayerThreat(sessionID string, isThreat bool) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}
	p.IsThreat = isThreat
	return true
}

func (gs *GameState) CommandAI(targetX, targetY float64) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	// Find AI
	for uid, e := range gs.Entities {
		if e.Type == EntityTypeNingBye {
			data := e.Extra.(NingByeAI)
			data.State = AIStateCommandMove
			data.TargetPos = Vector2{X: targetX, Y: targetY}
			data.PatrolPath = nil // Force recalc
			e.Extra = data
			gs.Entities[uid] = e
			log.Printf("[AI] Commanded to move to %.1f, %.1f", targetX, targetY)
			return true
		}
	}
	return false
}
