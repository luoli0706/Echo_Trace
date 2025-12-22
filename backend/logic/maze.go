package logic

import (
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
