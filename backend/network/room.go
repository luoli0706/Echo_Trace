package network

import (
	"log"
	"sync"
	"time"

	"echo_trace_server/logic"
)

type Room struct {
	ID         string
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	GameState  *logic.GameState
	Config     *logic.GameConfig
	Mutex      sync.RWMutex
}

func NewRoom(id string, cfg *logic.GameConfig) *Room {
	return &Room{
		ID:         id,
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		GameState:  logic.NewGameState(cfg),
		Config:     cfg,
	}
}

func (r *Room) Run() {
	ticker := time.NewTicker(time.Duration(r.Config.Server.TickRateMs) * time.Millisecond)
	defer ticker.Stop()

	log.Printf("Room %s started. Tick: %dms", r.ID, r.Config.Server.TickRateMs)

	for {
		select {
		case client := <-r.Register:
			r.Mutex.Lock()
			r.Clients[client] = true
			p := r.GameState.AddPlayer(client.SessionID)

			// Send Game Start (Map Info)
			startMsg := map[string]interface{}{
				"type": 3001,
				"payload": map[string]interface{}{
					"map_width":  r.GameState.Map.Width,
					"map_height": r.GameState.Map.Height,
					"spawn_pos":  p.Pos,
					"map_tiles":  r.GameState.Map.Tiles,
				},
			}
			client.SendJSON(startMsg)
			r.Mutex.Unlock()

		case client := <-r.Unregister:
			r.Mutex.Lock()
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				r.GameState.RemovePlayer(client.SessionID)
				close(client.Send)
			}
			r.Mutex.Unlock()

		case <-ticker.C:
			// 1. Update Physics
			dt := float64(r.Config.Server.TickRateMs) / 1000.0
			r.GameState.UpdateTick(dt)

			// 2. Broadcast State (AOI filtered)
			r.Mutex.RLock()
			for client := range r.Clients {
				snapshot := r.GameState.GetSnapshot(client.SessionID)
				msg := map[string]interface{}{
					"type":    3002,
					"payload": snapshot,
				}
				
				select {
				case client.Send <- toJSON(msg):
				default:
				}
			}
			r.Mutex.RUnlock()
		}
	}
}
