package logic

import "math"

// QuadtreeEntity 四叉树中存储的实体接口
type QuadtreeEntity struct {
	UID string
	X   float64
	Y   float64
}

// Quadtree 用于空间索引的四叉树结构
type Quadtree struct {
	Boundary Rect
	Capacity int
	Entities []QuadtreeEntity
	Divided  bool

	// 子节点 (按顺时针: NE, SE, SW, NW)
	NE *Quadtree
	SE *Quadtree
	SW *Quadtree
	NW *Quadtree
}

// Rect 矩形区域
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// NewQuadtree 创建新的四叉树
func NewQuadtree(boundary Rect, capacity int) *Quadtree {
	return &Quadtree{
		Boundary: boundary,
		Capacity: capacity,
		Entities: make([]QuadtreeEntity, 0, capacity),
		Divided:  false,
	}
}

// Contains 检查点是否在矩形内
func (r *Rect) Contains(x, y float64) bool {
	return x >= r.X-r.Width &&
		x <= r.X+r.Width &&
		y >= r.Y-r.Height &&
		y <= r.Y+r.Height
}

// Intersects 检查两个矩形是否相交
func (r *Rect) Intersects(other Rect) bool {
	return !(other.X-other.Width > r.X+r.Width ||
		other.X+other.Width < r.X-r.Width ||
		other.Y-other.Height > r.Y+r.Height ||
		other.Y+other.Height < r.Y-r.Height)
}

// Insert 向四叉树插入实体
func (qt *Quadtree) Insert(entity QuadtreeEntity) bool {
	// 检查实体是否在边界内
	if !qt.Boundary.Contains(entity.X, entity.Y) {
		return false
	}

	// 如果未达到容量且未分割，直接插入
	if len(qt.Entities) < qt.Capacity && !qt.Divided {
		qt.Entities = append(qt.Entities, entity)
		return true
	}

	// 需要分割
	if !qt.Divided {
		qt.subdivide()
	}

	// 尝试插入到子节点
	if qt.NE.Insert(entity) {
		return true
	}
	if qt.SE.Insert(entity) {
		return true
	}
	if qt.SW.Insert(entity) {
		return true
	}
	if qt.NW.Insert(entity) {
		return true
	}

	return false
}

// subdivide 分割四叉树
func (qt *Quadtree) subdivide() {
	x := qt.Boundary.X
	y := qt.Boundary.Y
	w := qt.Boundary.Width / 2
	h := qt.Boundary.Height / 2

	qt.NE = NewQuadtree(Rect{x + w, y - h, w, h}, qt.Capacity)
	qt.SE = NewQuadtree(Rect{x + w, y + h, w, h}, qt.Capacity)
	qt.SW = NewQuadtree(Rect{x - w, y + h, w, h}, qt.Capacity)
	qt.NW = NewQuadtree(Rect{x - w, y - h, w, h}, qt.Capacity)

	qt.Divided = true

	// 重新插入现有实体
	oldEntities := qt.Entities
	qt.Entities = make([]QuadtreeEntity, 0, qt.Capacity)

	for _, entity := range oldEntities {
		if !qt.NE.Insert(entity) && !qt.SE.Insert(entity) &&
			!qt.SW.Insert(entity) && !qt.NW.Insert(entity) {
			// 如果所有子节点都插入失败，保留在当前节点
			qt.Entities = append(qt.Entities, entity)
		}
	}
}

// Query 查询矩形范围内的实体
func (qt *Quadtree) Query(range_ Rect, found []QuadtreeEntity) []QuadtreeEntity {
	if found == nil {
		found = make([]QuadtreeEntity, 0)
	}

	// 如果范围不相交，直接返回
	if !qt.Boundary.Intersects(range_) {
		return found
	}

	// 检查当前节点的实体
	for _, entity := range qt.Entities {
		if range_.Contains(entity.X, entity.Y) {
			found = append(found, entity)
		}
	}

	// 如果已分割，递归查询子节点
	if qt.Divided {
		found = qt.NE.Query(range_, found)
		found = qt.SE.Query(range_, found)
		found = qt.SW.Query(range_, found)
		found = qt.NW.Query(range_, found)
	}

	return found
}

// QueryRadius 查询圆形范围内的实体
func (qt *Quadtree) QueryRadius(x, y, radius float64) []QuadtreeEntity {
	// 先用矩形查询
	rect := Rect{
		X:      x,
		Y:      y,
		Width:  radius,
		Height: radius,
	}

	candidates := qt.Query(rect, nil)

	// 过滤出圆形范围内的实体
	result := make([]QuadtreeEntity, 0)
	radiusSq := radius * radius

	for _, entity := range candidates {
		dx := entity.X - x
		dy := entity.Y - y
		distSq := dx*dx + dy*dy

		if distSq <= radiusSq {
			result = append(result, entity)
		}
	}

	return result
}

// Remove 从四叉树移除实体
func (qt *Quadtree) Remove(entity QuadtreeEntity) bool {
	if !qt.Boundary.Contains(entity.X, entity.Y) {
		return false
	}

	// 从当前节点移除
	for i, e := range qt.Entities {
		if e.UID == entity.UID {
			qt.Entities = append(qt.Entities[:i], qt.Entities[i+1:]...)
			return true
		}
	}

	// 如果已分割，从子节点移除
	if qt.Divided {
		if qt.NE.Remove(entity) {
			return true
		}
		if qt.SE.Remove(entity) {
			return true
		}
		if qt.SW.Remove(entity) {
			return true
		}
		if qt.NW.Remove(entity) {
			return true
		}
	}

	return false
}

// Clear 清空四叉树
func (qt *Quadtree) Clear() {
	qt.Entities = make([]QuadtreeEntity, 0, qt.Capacity)
	qt.Divided = false
	qt.NE = nil
	qt.SE = nil
	qt.SW = nil
	qt.NW = nil
}

// Rebuild 重建四叉树 (当实体移动后需要调用)
func (qt *Quadtree) Rebuild(entities []QuadtreeEntity) {
	qt.Clear()
	for _, entity := range entities {
		qt.Insert(entity)
	}
}

// GetNearby 获取附近的实体 (用于 AOI)
func (qt *Quadtree) GetNearby(x, y, viewRadius float64) []QuadtreeEntity {
	return qt.QueryRadius(x, y, viewRadius)
}

// GetVisibleEntities 获取可见实体 (带视野限制)
func (qt *Quadtree) GetVisibleEntities(observerX, observerY, viewRadius float64) []QuadtreeEntity {
	candidates := qt.QueryRadius(observerX, observerY, viewRadius)
	visible := make([]QuadtreeEntity, 0)

	for _, entity := range candidates {
		// 检查距离
		dx := entity.X - observerX
		dy := entity.Y - observerY
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance <= viewRadius {
			visible = append(visible, entity)
		}
	}

	return visible
}
