package network

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
		if typeCode == 1013 { // LIST_ROOMS
			c.handleListRooms()
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
		case 2009: // SHOP_REFRESH_REQ
			input.Type = logic.InputShopRefresh
			c.CurrentRoom.GameLoop.InputChan <- input
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

	roomName := ""
	if payload != nil {
		if rn, ok := payload["room_name"].(string); ok {
			roomName = rn
		}
	}
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		c.SendJSON(map[string]interface{}{
			"type":    4001,
			"payload": map[string]interface{}{"msg": "创建房间失败：必须填写房间名。"},
		})
		return
	}

	// Start from server defaults loaded from game_config.json.
	cfg := getDefaultConfigClone()

	// Override from payload (legacy flat fields).
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

		// Optional: allow a full nested config overlay.
		// If provided, it should be a partial GameConfig-shaped object.
		if rawCfg, ok := payload["config"].(map[string]interface{}); ok {
			if b, err := json.Marshal(rawCfg); err == nil {
				_ = json.Unmarshal(b, cfg)
			}
		}
	}

	// Final safety clamp (server-authoritative).
	logic.ClampGameConfig(cfg)

	// Hard safety bounds: clamp user-provided values to allowed maximums.
	logic.ClampGameConfig(cfg)

	room, roomID, ok := GlobalManager.CreateRoom(roomName, cfg)
	if !ok || room == nil {
		c.SendJSON(map[string]interface{}{
			"type":    4001,
			"payload": map[string]interface{}{"msg": "创建房间失败：房间名已存在，请重新命名。"},
		})
		return
	}
	c.CurrentRoom = room
	room.Register <- c

	c.SendJSON(map[string]interface{}{
		"type": 1012, // ROOM_JOINED
		"payload": map[string]interface{}{
			"success":   true,
			"room_id":   roomID,
			"room_name": roomName,
			"config":    cfg,
		},
	})
}

func (c *Client) handleJoinRoom(payload map[string]interface{}) {
	if c.CurrentRoom != nil {
		return
	}

	roomID := ""
	if payload != nil {
		if rid, ok := payload["room_id"].(string); ok {
			roomID = rid
		}
	}
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		c.SendJSON(map[string]interface{}{
			"type":    4001,
			"payload": map[string]interface{}{"msg": "加入房间失败：缺少 room_id。"},
		})
		return
	}

	room := GlobalManager.GetRoom(roomID)
	if room != nil {
		c.CurrentRoom = room
		room.Register <- c
		c.SendJSON(map[string]interface{}{
			"type": 1012, // ROOM_JOINED
			"payload": map[string]interface{}{
				"success":   true,
				"room_id":   room.ID,
				"room_name": room.Name,
				"config":    room.Config,
			},
		})
	} else {
		// Error
		c.SendJSON(map[string]interface{}{
			"type":    4001,
			"payload": map[string]interface{}{"msg": "加入房间失败：房间不存在或已关闭。"},
		})
	}
}

func (c *Client) handleListRooms() {
	rooms := GlobalManager.ListRoomSummaries()
	c.SendJSON(map[string]interface{}{
		"type": 1014, // ROOMS_LIST
		"payload": map[string]interface{}{
			"rooms": rooms,
		},
	})
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
