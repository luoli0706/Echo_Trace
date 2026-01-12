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
// Uses quadtree for spatial acceleration.
func (aoi *AOIManager) GetVisibleEntities(observer *Player, gameMap *GameMap, allPlayers map[string]*Player, allEntities []Entity, fovDegrees float64, qt *Quadtree) ([]*Player, []Entity) {
	visiblePlayers := make([]*Player, 0)
	visibleEntities := make([]Entity, 0)

	// Default cone: 90° total => 45° half-angle.
	if fovDegrees <= 0 {
		fovDegrees = 90.0
	}
	halfAngleRad := (fovDegrees * math.Pi / 180.0) / 2.0
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

	// 使用四叉树加速：先从四叉树查询附近实体
	if qt != nil {
		nearbyEntities := qt.QueryRadius(observer.Pos.X, observer.Pos.Y, observer.ViewRadius)

		// 从四叉树结果中提取可见实体
		for _, e := range nearbyEntities {
			// 跳过观察者自己
			if e.UID == observer.SessionID {
				continue
			}

			if inConeAndLOS(Vector2{X: e.X, Y: e.Y}) {
				// 检查是否为玩家实体
				if p, ok := allPlayers[e.UID]; ok && p.IsAlive {
					visiblePlayers = append(visiblePlayers, p)
				} else {
					// 静态实体
					if entity, exists := findEntityByUID(allEntities, e.UID); exists {
						visibleEntities = append(visibleEntities, entity)
					}
				}
			}
		}
	} else {
		// Fallback: 原有的暴力遍历逻辑
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
	}

	return visiblePlayers, visibleEntities
}

// findEntityByUID 从实体列表中查找指定 UID 的实体
func findEntityByUID(entities []Entity, uid string) (Entity, bool) {
	for _, e := range entities {
		if e.UID == uid {
			return e, true
		}
	}
	return Entity{}, false
}

// Distance helper
func Distance(p1, p2 Vector2) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}
