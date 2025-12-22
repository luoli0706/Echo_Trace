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

// GetVisibleEntities returns entities within viewRadius of observer
func (aoi *AOIManager) GetVisibleEntities(observer *Player, allPlayers map[string]*Player, allEntities []Entity) ([]*Player, []Entity) {
	visiblePlayers := make([]*Player, 0)
	visibleEntities := make([]Entity, 0)

	// Check Players
	for _, p := range allPlayers {
		if p.SessionID == observer.SessionID {
			continue
		}
		if p.IsAlive && Distance(observer.Pos, p.Pos) <= observer.ViewRadius {
			visiblePlayers = append(visiblePlayers, p)
		}
	}

	// Check Static Entities
	for _, e := range allEntities {
		if Distance(observer.Pos, e.Pos) <= observer.ViewRadius {
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
