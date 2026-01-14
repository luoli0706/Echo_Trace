package main

import (
	"bytes"
	"echo_trace_server/network"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type AdminRequest struct {
	SessionID string `json:"session_id"`
}

type ModifyHealthRequest struct {
	AdminRequest
	HP float64 `json:"hp"`
}

type ModifyArmorRequest struct {
	AdminRequest
	Armor float64 `json:"armor"`
}

type ModifyCapacityRequest struct {
	AdminRequest
	Capacity int `json:"capacity"`
}

type AddItemRequest struct {
	AdminRequest
	ItemIDs []string `json:"item_ids"`
}

type ModifySpeedRequest struct {
	AdminRequest
	Multiplier float64 `json:"multiplier"`
	Duration   float64 `json:"duration"`
}

type SetThreatRequest struct {
	AdminRequest
	IsThreat bool `json:"is_threat"`
}

type CommandAIRequest struct {
	TargetX float64 `json:"target_x"`
	TargetY float64 `json:"target_y"`
}

type MovePlayerRequest struct {
	SessionID string  `json:"session_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
}

type WishRequest struct {
	AdminRequest
	Wish string `json:"wish"`
}

type mcpWishResponse struct {
	Status           string        `json:"status"`
	Results          []interface{} `json:"results"`
	PlayerVisibleMsg string        `json:"玩家可见回应"`
	ActionSummary    string        `json:"action_summary"`
}

func setPlayerClientMsg(sessionID string, msg string) {
	if sessionID == "" {
		return
	}
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			gs := room.GameLoop.GameState
			gs.Mutex.Lock()
			if p, ok := gs.Players[sessionID]; ok {
				p.ClientMsg = msg
				gs.Mutex.Unlock()
				return
			}
			gs.Mutex.Unlock()
		}
	}
}

// Helper to set LastAIAction specifically
func setPlayerLastAIAction(sessionID string, action string) {
	if sessionID == "" {
		return
	}
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			gs := room.GameLoop.GameState
			gs.Mutex.Lock()
			if p, ok := gs.Players[sessionID]; ok {
				p.LastAIAction = action
				gs.Mutex.Unlock()
				return
			}
			gs.Mutex.Unlock()
		}
	}
}

// ... (Legacy code, kept for reference but reusing logic)

type ModifyGlobalHealthRequest struct {
	AdminRequest
	HP float64 `json:"hp"`
}

// ... (Existing types)

func handleAdminModifyGlobalHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ModifyGlobalHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.AdminModifyGlobalHealth(req.HP) {
				found = true
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "No active game states", http.StatusNotFound)
	}
}

func handleAdminModifyHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ModifyHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.AdminModifyHealth(req.SessionID, req.HP) {
				found = true
				break
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleAdminModifyArmor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ModifyArmorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.AdminModifyArmor(req.SessionID, req.Armor) {
				found = true
				break
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleAdminModifyCapacity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ModifyCapacityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.AdminModifyInventoryCap(req.SessionID, req.Capacity) {
				found = true
				break
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleAdminAddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.AdminAddItems(req.SessionID, req.ItemIDs) {
				found = true
				break
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleAdminModifySpeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ModifySpeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.AdminModifySpeed(req.SessionID, req.Multiplier, req.Duration) {
				found = true
				break
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleAdminSetThreat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req SetThreatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			if room.GameLoop.GameState.SetPlayerThreat(req.SessionID, req.IsThreat) {
				found = true
				break
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleAdminCommandAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CommandAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			// Iterate all rooms, if any has AI, command it.
			// In multi-room setup, this is ambiguous (which room?), but for this demo assuming single active room or command all.
			if room.GameLoop.GameState.CommandAI(req.TargetX, req.TargetY) {
				found = true
				// Continue to command all AIs in all rooms? Or break?
				// Let's command all.
			}
		}
	}

	if found {
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "AI not found in any room", http.StatusNotFound)
	}
}

func handleAdminMovePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req MovePlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	resultMsg := ""
	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			var ok bool
			ok, resultMsg = room.GameLoop.GameState.AdminMovePlayer(req.SessionID, req.X, req.Y)
			if ok {
				found = true
				break
			}
		}
	}

	if found {
		if resultMsg == "OK" {
			w.Write([]byte("OK"))
		} else {
			// Return 200 but with error message in body so MCP can see it,
			// OR return 400/409. 
			// MCP tool helper `_call_backend` checks for 200.
			// If I return 400, MCP says "Failed".
			// "Move blocked" is a failure condition.
			http.Error(w, resultMsg, http.StatusConflict)
		}
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
}

func handleWish(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req WishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	consumed := false
	consumedItemID := ""
	var playerInfoSnapshot map[string]interface{}

	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			gs := room.GameLoop.GameState
			gs.Mutex.Lock()
			if p, ok := gs.Players[req.SessionID]; ok {
				found = true
				// Check for Item
				slotIdx := -1
				for i, item := range p.Inventory {
					if item.ID == "UTIL_DEV_FORGOTTEN_CLI" {
						slotIdx = i
						consumedItemID = "UTIL_DEV_FORGOTTEN_CLI"
						break
					}
					if item.ID == "UTIL_WISH_MACHINE" {
						slotIdx = i
						consumedItemID = "UTIL_WISH_MACHINE"
						break
					}
				}

				if slotIdx >= 0 {
					// Consume Item
					p.Inventory = append(p.Inventory[:slotIdx], p.Inventory[slotIdx+1:]...)
					gs.RecalculateStats(p)
					consumed = true
					
					// Snapshot player data for MCP
					playerInfoSnapshot = map[string]interface{}{
						"pos_x":   p.Pos.X,
						"pos_y":   p.Pos.Y,
						"hp":      p.HP,
						"max_hp":  p.MaxHP,
						"armor":   p.Armor,
						"funds":   p.Funds,
						"name":    p.Name,
						"is_dead": p.IsDead,
					}
				}
			}
			gs.Mutex.Unlock()

			// Call AdminWish outside the lock to avoid deadlock
			if consumed {
				gs.AdminWish(req.SessionID, req.Wish)
			}

			if found {
				break
			}
		}
	}

	if !found {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	if !consumed {
		http.Error(w, "You do not have a Wish Device!", http.StatusForbidden)
		return
	}

	// Async call to MCP
	go func(sid, wText string, itemID string, info map[string]interface{}) {
		setPlayerClientMsg(sid, "Wish received. Processing...")
		mcpURL := "http://localhost:9091/wish"
		
		bodyData := map[string]interface{}{
			"session_id":  sid,
			"wish":        wText,
			"item_id":     itemID,
			"player_info": info,
		}
		
		jsonData, _ := json.Marshal(bodyData)
		client := &http.Client{Timeout: 12 * time.Second}
		resp, err := client.Post(mcpURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Failed to call MCP: %v", err)
			setPlayerClientMsg(sid, "Wish failed: MCP unreachable.")
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		log.Printf("[WISH] MCP response: %s", string(b))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			setPlayerClientMsg(sid, "Wish failed: MCP error.")
			return
		}
		var parsed mcpWishResponse
		if err := json.Unmarshal(b, &parsed); err != nil {
			setPlayerClientMsg(sid, "Wish failed: invalid MCP response.")
			return
		}
		
		// Log Action Summary to Console
		if parsed.ActionSummary != "" {
			log.Printf("[MCP ACTION] Player %s: %s", sid, parsed.ActionSummary)
			setPlayerLastAIAction(sid, parsed.ActionSummary)
		}

		msg := parsed.PlayerVisibleMsg
		if msg == "" {
			msg = "Wish processed."
		}
		setPlayerClientMsg(sid, msg)
	}(req.SessionID, req.Wish, consumedItemID, playerInfoSnapshot)

	w.Write([]byte("Wish granted (Item Consumed)."))
}

type RoomPlayersRequest struct {
	SessionID string `json:"session_id"`
}

type PlayerSummary struct {
	SessionID string  `json:"session_id"`
	Name      string  `json:"name"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	IsAlive   bool    `json:"is_alive"`
}

func handleAdminGetRoomPlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RoomPlayersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	foundRoom := false
	var summaries []PlayerSummary

	roomIDs := network.GlobalManager.ListRooms()
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room != nil && room.GameLoop != nil && room.GameLoop.GameState != nil {
			gs := room.GameLoop.GameState
			gs.Mutex.Lock()
			// Check if requester is in this room
			if _, ok := gs.Players[req.SessionID]; ok {
				foundRoom = true
				summaries = make([]PlayerSummary, 0, len(gs.Players))
				for sid, p := range gs.Players {
					summaries = append(summaries, PlayerSummary{
						SessionID: sid,
						Name:      p.Name,
						X:         p.Pos.X,
						Y:         p.Pos.Y,
						IsAlive:   p.IsAlive,
					})
				}
			}
			gs.Mutex.Unlock()
			if foundRoom {
				break
			}
		}
	}

	if !foundRoom {
		http.Error(w, "Player room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"players": summaries,
	})
}
