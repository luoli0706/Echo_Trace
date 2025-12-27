package logic

import (
	"math"
)

// AOIManager handles visibility calculations
type AOIManager struct {
	MapWidth  int
	MapHeight int
}

func NewAOIManager(width, height int) *AOIManager {
	return &AOIManager{
		MapWidth:  width,
		MapHeight: height,
	}
}

// GetVisibleEntities returns entities within the observer's vision cone and view radius.
// Vision is blocked by wall tiles (LOS).
func (aoi *AOIManager) GetVisibleEntities(observer *Player, gameMap *GameMap, allPlayers map[string]*Player, allEntities []Entity) ([]*Player, []Entity) {
	visiblePlayers := make([]*Player, 0)
	visibleEntities := make([]Entity, 0)

	// Default cone: 90° total => 45° half-angle.
	halfAngleRad := math.Pi / 4.0
	cosHalf := math.Cos(halfAngleRad)
	look := observer.LookDir
	lookLen2 := look.X*look.X + look.Y*look.Y
	if lookLen2 == 0 {
		look = Vector2{X: 1, Y: 0}
	} else {
		inv := 1.0 / math.Sqrt(lookLen2)
		look = Vector2{X: look.X * inv, Y: look.Y * inv}
	}

	inConeAndLOS := func(target Vector2) bool {
		// Radius check
		dx := target.X - observer.Pos.X
		dy := target.Y - observer.Pos.Y
		if dx*dx+dy*dy > observer.ViewRadius*observer.ViewRadius {
			return false
		}
		// Cone check
		len2 := dx*dx + dy*dy
		if len2 == 0 {
			return true
		}
		invLen := 1.0 / math.Sqrt(len2)
		ux := dx * invLen
		uy := dy * invLen
		dot := ux*look.X + uy*look.Y
		if dot < cosHalf {
			return false
		}
		// LOS check
		if gameMap == nil {
			return true
		}
		return gameMap.HasLineOfSight(observer.Pos, target)
	}

	// Check Players
	for _, p := range allPlayers {
		if p.SessionID == observer.SessionID {
			continue
		}
		if p.IsAlive && inConeAndLOS(p.Pos) {
			visiblePlayers = append(visiblePlayers, p)
		}
	}

	// Check Static Entities
	for _, e := range allEntities {
		if inConeAndLOS(e.Pos) {
			visibleEntities = append(visibleEntities, e)
		}
	}

	return visiblePlayers, visibleEntities
}

// Distance helper
func Distance(p1, p2 Vector2) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}
