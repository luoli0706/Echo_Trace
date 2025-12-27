package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"echo_trace_server/logic"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	CurrentRoom *Room
	Conn        *websocket.Conn
	Send        chan []byte
	SessionID   string
}

func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	sessID := fmt.Sprintf("u_%d", time.Now().UnixNano())

	client := &Client{CurrentRoom: nil, Conn: conn, Send: make(chan []byte, 256), SessionID: sessID}

	// Don't register yet. Wait for Join/Create.

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		if c.CurrentRoom != nil {
			c.CurrentRoom.Unregister <- c
		}
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var req map[string]interface{}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		typeCodeFloat, ok := req["type"].(float64)
		if !ok {
			continue
		}
		typeCode := int(typeCodeFloat)

		// Room Management Packets
		if typeCode == 1010 { // CREATE_ROOM
			payload, _ := req["payload"].(map[string]interface{})
			c.handleCreateRoom(payload)
			continue
		}
		if typeCode == 1011 { // JOIN_ROOM
			payload, _ := req["payload"].(map[string]interface{})
			c.handleJoinRoom(payload)
			continue
		}

		// Game Packets (Require Room)
		if c.CurrentRoom == nil {
			continue
		}

		input := logic.PlayerInput{SessionID: c.SessionID}

		switch typeCode {
		case 1001: // LOGIN_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if name, ok := payload["name"].(string); ok {
					input.Type = logic.InputLogin
					input.Name = name
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 2001: // MOVE_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if dirMap, ok := payload["dir"].(map[string]interface{}); ok {
					input.Type = logic.InputMove
					input.Dir = logic.Vector2{
						X: dirMap["x"].(float64),
						Y: dirMap["y"].(float64),
					}
					if lookMap, ok := payload["look_dir"].(map[string]interface{}); ok {
						// look_dir is optional for backward compatibility.
						lx, lxOk := lookMap["x"].(float64)
						ly, lyOk := lookMap["y"].(float64)
						if lxOk && lyOk {
							input.LookDir = logic.Vector2{X: lx, Y: ly}
							input.HasLookDir = true
						}
					}
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 2002: // USE_ITEM_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				input.Type = logic.InputUseItem
				if slot, ok := payload["slot_index"].(float64); ok {
					input.SlotIndex = int(slot)
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 2003: // INTERACT_REQ
			input.Type = logic.InputInteract
			c.CurrentRoom.GameLoop.InputChan <- input
		case 2004: // PICKUP_REQ
			input.Type = logic.InputPickup
			c.CurrentRoom.GameLoop.InputChan <- input
		case 2005: // DROP_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if slot, ok := payload["slot_index"].(float64); ok {
					input.Type = logic.InputDrop
					input.SlotIndex = int(slot)
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 2006: // CHOOSE_TACTIC_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if tactic, ok := payload["tactic"].(string); ok {
					input.Type = logic.InputTactic
					input.Tactic = tactic
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 2007: // BUY_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if itemID, ok := payload["item_id"].(string); ok {
					input.Type = logic.InputBuy
					input.ItemID = itemID
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 2008: // SELL_REQ
			if payload, ok := req["payload"].(map[string]interface{}); ok {
				if slot, ok := payload["slot_index"].(float64); ok {
					input.Type = logic.InputSell
					input.SlotIndex = int(slot)
					c.CurrentRoom.GameLoop.InputChan <- input
				}
			}
		case 9001: // DEV_SKIP_PHASE
			input.Type = logic.InputDevSkip
			c.CurrentRoom.GameLoop.InputChan <- input
		}
	}
}

func (c *Client) handleCreateRoom(payload map[string]interface{}) {
	if c.CurrentRoom != nil {
		return
	} // Already in room

	// Parse Config from payload or use Default
	// For now, let's just use default config passed from main (we need access to it?)
	// Or parse parts.

	// Minimal: Generate Room ID
	roomID := fmt.Sprintf("room_%d", time.Now().Unix()%1000)

	// Deep Copy Logic Config? Or create new.
	// We need logic.GameConfig struct.
	// Since we are inside network package, we need to import logic.

	cfg := &logic.GameConfig{}

	// Basic default
	cfg.Server.TickRateMs = 50
	cfg.Server.MaxPlayers = 6
	cfg.Map.Width = 32
	cfg.Map.Height = 32
	cfg.Map.WallDensity = 0.2
	cfg.Gameplay.BaseMoveSpeed = 4.0
	cfg.Gameplay.BaseViewRadius = 5.0
	cfg.Phases.Phase1.Duration = 120
	cfg.Phases.Phase2.Duration = 180

	// Override from payload
	if payload != nil {
		if mp, ok := payload["max_players"].(float64); ok {
			cfg.Server.MaxPlayers = int(mp)
		}
		if p1, ok := payload["phase1_dur"].(float64); ok {
			cfg.Phases.Phase1.Duration = int(p1)
		}
		if p2, ok := payload["phase2_dur"].(float64); ok {
			cfg.Phases.Phase2.Duration = int(p2)
		}
		if m, ok := payload["motors"].(float64); ok {
			cfg.Phases.Phase2.MotorsSpawnCount = int(m)
		}
	}

	room := GlobalManager.CreateRoom(roomID, cfg)
	c.CurrentRoom = room
	room.Register <- c

	c.SendJSON(map[string]interface{}{
		"type": 1012, // ROOM_JOINED
		"payload": map[string]interface{}{
			"success": true,
			"room_id": roomID,
			"config":  cfg,
		},
	})
}

func (c *Client) handleJoinRoom(payload map[string]interface{}) {
	if c.CurrentRoom != nil {
		return
	}

	// Auto join first available or by ID
	var room *Room

	// For Alpha: Join "room_id" if provided, else any
	if payload != nil {
		if rid, ok := payload["room_id"].(string); ok {
			room = GlobalManager.GetRoom(rid)
		}
	}

	if room == nil {
		// Pick first
		rooms := GlobalManager.ListRooms()
		if len(rooms) > 0 {
			room = GlobalManager.GetRoom(rooms[0])
		}
	}

	if room != nil {
		c.CurrentRoom = room
		room.Register <- c
		c.SendJSON(map[string]interface{}{
			"type": 1012, // ROOM_JOINED
			"payload": map[string]interface{}{
				"success": true,
				"room_id": room.ID,
				"config":  room.Config,
			},
		})
	} else {
		// Error
		c.SendJSON(map[string]interface{}{
			"type":    4001,
			"payload": map[string]interface{}{"msg": "No rooms available. Create one!"},
		})
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (c *Client) SendJSON(v interface{}) {
	b, _ := json.Marshal(v)
	c.Send <- b
}

func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
