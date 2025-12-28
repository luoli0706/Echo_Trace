package network

import (
	"log"
	"strconv"
	"sync"
	"time"

	"echo_trace_server/logic"
)

type RoomManager struct {
	Rooms    map[string]*Room
	Register chan *Client // Clients waiting to join a room? No, clients manage themselves via Req.
	Mutex    sync.RWMutex
}

var GlobalManager *RoomManager

var defaultConfig logic.GameConfig
var hasDefaultConfig bool

// SetDefaultConfig sets the baseline config used for newly created rooms.
// The value is copied to avoid sharing mutable state.
func SetDefaultConfig(cfg *logic.GameConfig) {
	if cfg == nil {
		hasDefaultConfig = false
		defaultConfig = logic.GameConfig{}
		return
	}
	defaultConfig = *cfg
	hasDefaultConfig = true
}

func getDefaultConfigClone() *logic.GameConfig {
	if hasDefaultConfig {
		cfg := defaultConfig
		return &cfg
	}
	return &logic.GameConfig{}
}

func InitManager() {
	GlobalManager = &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) roomNameTakenLocked(name string) bool {
	for _, r := range rm.Rooms {
		if r != nil && r.Name == name {
			return true
		}
	}
	return false
}

func (rm *RoomManager) CreateRoom(name string, cfg *logic.GameConfig) (*Room, string, bool) {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	if name == "" {
		return nil, "", false
	}
	if rm.roomNameTakenLocked(name) {
		return nil, "", false
	}

	id := "room_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	room := NewRoom(id, name, cfg)
	rm.Rooms[id] = room
	go room.Run()
	log.Printf("Created Room %s (%s)", id, name)
	return room, id, true
}

func (rm *RoomManager) GetRoom(id string) *Room {
	rm.Mutex.RLock()
	defer rm.Mutex.RUnlock()
	return rm.Rooms[id]
}

func (rm *RoomManager) GetRoomByName(name string) *Room {
	rm.Mutex.RLock()
	defer rm.Mutex.RUnlock()
	for _, r := range rm.Rooms {
		if r != nil && r.Name == name {
			return r
		}
	}
	return nil
}

// ListRooms returns a list of room IDs (for simple joining)
func (rm *RoomManager) ListRooms() []string {
	rm.Mutex.RLock()
	defer rm.Mutex.RUnlock()
	keys := make([]string, 0, len(rm.Rooms))
	for k := range rm.Rooms {
		keys = append(keys, k)
	}
	return keys
}

func (rm *RoomManager) ListRoomSummaries() []map[string]interface{} {
	rm.Mutex.RLock()
	defer rm.Mutex.RUnlock()
	rooms := make([]map[string]interface{}, 0, len(rm.Rooms))
	for _, r := range rm.Rooms {
		if r == nil {
			continue
		}
		playerCount := 0
		phase := -1
		maxPlayers := 0
		mapW := 0
		mapH := 0
		r.Mutex.RLock()
		playerCount = len(r.Clients)
		if r.GameLoop != nil && r.GameLoop.GameState != nil {
			phase = int(r.GameLoop.GameState.Phase)
		}
		if r.Config != nil {
			maxPlayers = r.Config.Server.MaxPlayers
			mapW = r.Config.Map.Width
			mapH = r.Config.Map.Height
		}
		r.Mutex.RUnlock()
		rooms = append(rooms, map[string]interface{}{
			"room_id":     r.ID,
			"room_name":   r.Name,
			"players":     playerCount,
			"max_players": maxPlayers,
			"phase":       phase,
			"map_width":   mapW,
			"map_height":  mapH,
		})
	}
	return rooms
}
