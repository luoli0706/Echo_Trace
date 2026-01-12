package network

import (
	"log"
	"sync"
	"time"

	"echo_trace_server/logic"
)

type Room struct {
	ID         string
	Name       string
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	GameLoop   *logic.GameLoop
	Config     *logic.GameConfig
	Mutex      sync.RWMutex
	closedOnce sync.Once
	ClosedChan chan struct{}
}

func NewRoom(id string, name string, cfg *logic.GameConfig) *Room {
	r := &Room{
		ID:         id,
		Name:       name,
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		GameLoop:   logic.NewGameLoop(cfg),
		Config:     cfg,
		ClosedChan: make(chan struct{}),
	}
	return r
}

func (r *Room) Close(reason string) {
	r.closedOnce.Do(func() {
		log.Printf("Room %s closing: %s", r.ID, reason)
		close(r.ClosedChan)
		if r.GameLoop != nil {
			r.GameLoop.Stop()
		}

		// Detach clients but keep their WS connections alive.
		clients := make([]*Client, 0)
		r.Mutex.Lock()
		for c := range r.Clients {
			clients = append(clients, c)
			delete(r.Clients, c)
		}
		r.Mutex.Unlock()

		for _, c := range clients {
			if c == nil {
				continue
			}
			if c.CurrentRoom == r {
				c.CurrentRoom = nil
			}
			c.SendJSON(map[string]interface{}{
				"type": 1017, // ROOM_CLOSED
				"payload": map[string]interface{}{
					"reason": reason,
				},
			})
		}

		if GlobalManager != nil {
			GlobalManager.RemoveRoom(r.ID)
		}
	})
}

func (r *Room) RemoveClient(client *Client) {
	if client == nil {
		return
	}
	shouldClose := false
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
		if !stillConnected && r.GameLoop != nil && r.GameLoop.GameState != nil {
			r.GameLoop.GameState.MarkPlayerDisconnected(client.SessionID)
		}
	}
	if len(r.Clients) == 0 {
		shouldClose = true
	}
	r.Mutex.Unlock()

	if client.CurrentRoom == r {
		client.CurrentRoom = nil
	}

	// Auto-release when no players remain.
	if shouldClose {
		r.Close("empty room")
	}
}

func (r *Room) Run() {
	// Start Game Loop
	go r.GameLoop.Run()
	log.Printf("Room %s started. Tick: %dms", r.ID, r.Config.Server.TickRateMs)

	lastPhase := logic.PhaseInit
	var endedAt time.Time

	for {
		select {
		case <-r.ClosedChan:
			return
		case client := <-r.Register:
			r.Mutex.Lock()
			// If the same session_id is already connected, drop the old connection.
			for other := range r.Clients {
				if other != nil && other.SessionID == client.SessionID {
					delete(r.Clients, other)
					// Keep WS alive; the old client connection will be closed by its own pumps.
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
				if currentPhase == logic.PhaseEnded {
					if endedAt.IsZero() {
						endedAt = time.Now()
					}
					if time.Since(endedAt) >= 5*time.Second {
						r.Close("game ended")
						return
					}
				} else {
					endedAt = time.Time{}
				}
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
