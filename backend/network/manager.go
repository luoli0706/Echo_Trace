package network

import (
	"log"
	"sync"

	"echo_trace_server/logic"
)

type RoomManager struct {
	Rooms    map[string]*Room
	Register chan *Client // Clients waiting to join a room? No, clients manage themselves via Req.
	Mutex    sync.RWMutex
}

var GlobalManager *RoomManager

func InitManager() {
	GlobalManager = &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) CreateRoom(id string, cfg *logic.GameConfig) *Room {
	rm.Mutex.Lock()
	defer rm.Mutex.Unlock()

	room := NewRoom(id, cfg)
	rm.Rooms[id] = room
	go room.Run()
	log.Printf("Created Room %s", id)
	return room
}

func (rm *RoomManager) GetRoom(id string) *Room {
	rm.Mutex.RLock()
	defer rm.Mutex.RUnlock()
	return rm.Rooms[id]
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
