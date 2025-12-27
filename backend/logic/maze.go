package logic

import (
	"math"
	"math/rand"
	"time"
)

// Tile types
const (
	TileEmpty = 0
	TileWall  = 1
)

type GameMap struct {
	Width  int
	Height int
	Tiles  [][]int // 0: Walkable, 1: Wall
}

func NewGameMap(width, height int, density float64) *GameMap {
	rand.Seed(time.Now().UnixNano())
	tiles := make([][]int, height)
	for y := 0; y < height; y++ {
		tiles[y] = make([]int, width)
		for x := 0; x < width; x++ {
			// Borders are always walls
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				tiles[y][x] = TileWall
			} else {
				// Random walls based on density
				if rand.Float64() < density {
					tiles[y][x] = TileWall
				} else {
					tiles[y][x] = TileEmpty
				}
			}
		}
	}
	return &GameMap{
		Width:  width,
		Height: height,
		Tiles:  tiles,
	}
}

// IsWalkable checks collision with grid
func (m *GameMap) IsWalkable(x, y float64) bool {
	// Simple grid collision
	gridX := int(x)
	gridY := int(y)

	if gridX < 0 || gridX >= m.Width || gridY < 0 || gridY >= m.Height {
		return false
	}
	return m.Tiles[gridY][gridX] == TileEmpty
}

func (m *GameMap) GetRandomSpawnPos() Vector2 {
	for {
		x := rand.Intn(m.Width-2) + 1
		y := rand.Intn(m.Height-2) + 1
		if m.Tiles[y][x] == TileEmpty {
			return Vector2{X: float64(x) + 0.5, Y: float64(y) + 0.5}
		}
	}
}

// HasLineOfSight returns true if the segment from 'from' to 'to' does not cross any wall tiles.
// Coordinates are in world space where integer grid cells correspond to tiles.
func (m *GameMap) HasLineOfSight(from, to Vector2) bool {
	// Clamp quick out-of-bounds
	if from.X < 0 || from.Y < 0 || to.X < 0 || to.Y < 0 {
		return false
	}
	if from.X >= float64(m.Width) || to.X >= float64(m.Width) || from.Y >= float64(m.Height) || to.Y >= float64(m.Height) {
		return false
	}

	fx, fy := from.X, from.Y
	tx, ty := to.X, to.Y
	dx := tx - fx
	dy := ty - fy

	// Same tile
	startX, startY := int(fx), int(fy)
	endX, endY := int(tx), int(ty)
	if startX == endX && startY == endY {
		return true
	}

	stepX := 0
	stepY := 0
	if dx > 0 {
		stepX = 1
	} else if dx < 0 {
		stepX = -1
	}
	if dy > 0 {
		stepY = 1
	} else if dy < 0 {
		stepY = -1
	}

	// Avoid division by zero
	invDx := math.Inf(1)
	invDy := math.Inf(1)
	if dx != 0 {
		invDx = 1.0 / math.Abs(dx)
	}
	if dy != 0 {
		invDy = 1.0 / math.Abs(dy)
	}

	// tMax is the distance along the ray where we cross the first vertical/horizontal grid boundary.
	x := startX
	y := startY

	// Starting cell itself should not block (player stands in walkable).

	var tMaxX, tMaxY float64
	var tDeltaX, tDeltaY float64

	if stepX != 0 {
		var nextV float64
		if stepX > 0 {
			nextV = float64(x + 1)
		} else {
			nextV = float64(x)
		}
		tMaxX = math.Abs(nextV-fx) * invDx
		tDeltaX = 1.0 * invDx
	} else {
		tMaxX = math.Inf(1)
		tDeltaX = math.Inf(1)
	}

	if stepY != 0 {
		var nextH float64
		if stepY > 0 {
			nextH = float64(y + 1)
		} else {
			nextH = float64(y)
		}
		tMaxY = math.Abs(nextH-fy) * invDy
		tDeltaY = 1.0 * invDy
	} else {
		tMaxY = math.Inf(1)
		tDeltaY = math.Inf(1)
	}

	// Traverse grid cells until we reach the destination cell.
	maxSteps := m.Width*m.Height + 8
	for i := 0; i < maxSteps; i++ {
		if tMaxX < tMaxY {
			x += stepX
			tMaxX += tDeltaX
		} else {
			y += stepY
			tMaxY += tDeltaY
		}

		if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
			return false
		}
		if m.Tiles[y][x] == TileWall {
			return false
		}
		if x == endX && y == endY {
			return true
		}
	}

	return false
}
