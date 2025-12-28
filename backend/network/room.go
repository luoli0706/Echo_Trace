package network

import (
	"log"
	"sync"

	"echo_trace_server/logic"
)

type Room struct {
	ID         string
	Name       string
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	GameLoop   *logic.GameLoop
	Config     *logic.GameConfig
	Mutex      sync.RWMutex
}

func NewRoom(id string, name string, cfg *logic.GameConfig) *Room {
	r := &Room{
		ID:         id,
		Name:       name,
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		GameLoop:   logic.NewGameLoop(cfg),
		Config:     cfg,
	}
	return r
}

func (r *Room) Run() {
	// Start Game Loop
	go r.GameLoop.Run()
	log.Printf("Room %s started. Tick: %dms", r.ID, r.Config.Server.TickRateMs)

	lastPhase := logic.PhaseInit

	for {
		select {
		case client := <-r.Register:
			r.Mutex.Lock()
			// If the same session_id is already connected, drop the old connection.
			for other := range r.Clients {
				if other != nil && other.SessionID == client.SessionID {
					delete(r.Clients, other)
					close(other.Send)
				}
			}
			r.Clients[client] = true
			// Direct call to GameState is safe (Mutex)
			p := r.GameLoop.GameState.AddPlayer(client.SessionID)
			if client.PlayerName != "" {
				p.Name = client.PlayerName
			}

			// Send Login Response
			loginMsg := map[string]interface{}{
				"type": 1001,
				"payload": map[string]interface{}{
					"success":    true,
					"session_id": client.SessionID,
					"config":     r.Config,
				},
			}
			client.SendJSON(loginMsg)

			// IF Game already started, send Map info immediately
			if r.GameLoop.GameState.Phase >= logic.PhaseSearch {
				startMsg := map[string]interface{}{
					"type": 3001,
					"payload": map[string]interface{}{
						"map_width":  r.GameLoop.GameState.Map.Width,
						"map_height": r.GameLoop.GameState.Map.Height,
						"spawn_pos":  p.Pos,
						"map_tiles":  r.GameLoop.GameState.Map.Tiles,
						"inventory":  p.Inventory,
					},
				}
				client.SendJSON(startMsg)
			}
			r.Mutex.Unlock()

		case client := <-r.Unregister:
			r.Mutex.Lock()
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				// Only mark disconnected if no other connection for this session_id exists.
				stillConnected := false
				for other := range r.Clients {
					if other != nil && other.SessionID == client.SessionID {
						stillConnected = true
						break
					}
				}
				if !stillConnected {
					r.GameLoop.GameState.MarkPlayerDisconnected(client.SessionID)
				}
				close(client.Send)
			}
			r.Mutex.Unlock()

		case snapshots := <-r.GameLoop.SnapshotChan:
			var currentPhase int = -1
			for _, s := range snapshots {
				if m, ok := s.(map[string]interface{}); ok {
					if p, ok := m["phase"].(int); ok {
						currentPhase = p
						break
					}
				}
			}

			if currentPhase != -1 {
				if lastPhase == logic.PhaseInit && currentPhase == logic.PhaseSearch {
					log.Println("Game Started! Sending Map Info...")
					r.Mutex.Lock()
					for client := range r.Clients {
						if p, ok := r.GameLoop.GameState.Players[client.SessionID]; ok {
							startMsg := map[string]interface{}{
								"type": 3001,
								"payload": map[string]interface{}{
									"map_width":  r.GameLoop.GameState.Map.Width,
									"map_height": r.GameLoop.GameState.Map.Height,
									"spawn_pos":  p.Pos,
									"map_tiles":  r.GameLoop.GameState.Map.Tiles,
									"inventory":  p.Inventory,
								},
							}
							client.SendJSON(startMsg)
						}
					}
					r.Mutex.Unlock()
					lastPhase = logic.PhaseSearch
				}
			}

			// Broadcast State
			r.Mutex.RLock()
			for client := range r.Clients {
				if snap, ok := snapshots[client.SessionID]; ok {
					msg := map[string]interface{}{
						"type":    3002,
						"payload": snap,
					}

					select {
					case client.Send <- toJSON(msg):
					default:
					}
				}
			}
			r.Mutex.RUnlock()
		}
	}
}
