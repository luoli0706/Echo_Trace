package logic

import (
	"math"
)

// ResolveMovement handles Circle vs TileMap collision with sliding
func (gs *GameState) ResolveMovement(pos Vector2, delta Vector2, radius float64) Vector2 {
	// Try full move
	target := Vector2{X: pos.X + delta.X, Y: pos.Y + delta.Y}
	
	// Check collision at target
	if !gs.checkCollision(target, radius) {
		return target
	}

	// Collision detected. Try sliding.
	// 1. Try X axis only
	targetX := Vector2{X: pos.X + delta.X, Y: pos.Y}
	if !gs.checkCollision(targetX, radius) {
		return targetX
	}

	// 2. Try Y axis only
	targetY := Vector2{X: pos.X, Y: pos.Y + delta.Y}
	if !gs.checkCollision(targetY, radius) {
		return targetY
	}

	// 3. Blocked
	return pos
}

func (gs *GameState) checkCollision(pos Vector2, radius float64) bool {
	// Check map bounds
	if pos.X < radius || pos.X > float64(gs.Map.Width)-radius ||
	   pos.Y < radius || pos.Y > float64(gs.Map.Height)-radius {
		return true
	}

	// Check surrounding tiles
	minX := int(pos.X - radius)
	maxX := int(pos.X + radius)
	minY := int(pos.Y - radius)
	maxY := int(pos.Y + radius)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !gs.Map.IsWalkable(float64(x)+0.5, float64(y)+0.5) {
				// Wall found. Check Circle vs AABB
				if CircleAABB(pos, radius, x, y) {
					return true
				}
			}
		}
	}
	return false
}

// CircleAABB checks overlap between circle at pos/radius and tile at tx,ty (1x1 size)
func CircleAABB(c Vector2, r float64, tx, ty int) bool {
	// Find closest point on AABB to Circle center
	closestX := math.Max(float64(tx), math.Min(c.X, float64(tx+1)))
	closestY := math.Max(float64(ty), math.Min(c.Y, float64(ty+1)))

	distanceX := c.X - closestX
	distanceY := c.Y - closestY

	return (distanceX*distanceX + distanceY*distanceY) < (r * r)
}
