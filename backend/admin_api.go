package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"echo_trace_server/network"
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

type WishRequest struct {
	AdminRequest
	Wish string `json:"wish"`
}

// ... (Legacy code, kept for reference but reusing logic)

func handleAdminModifyHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req ModifyHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	
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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "Player not found", http.StatusNotFound) }
}

func handleAdminModifyArmor(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req ModifyArmorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "Player not found", http.StatusNotFound) }
}

func handleAdminModifyCapacity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req ModifyCapacityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "Player not found", http.StatusNotFound) }
}

func handleAdminAddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "Player not found", http.StatusNotFound) }
}

func handleAdminModifySpeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req ModifySpeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "Player not found", http.StatusNotFound) }
}

func handleAdminSetThreat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req SetThreatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "Player not found", http.StatusNotFound) }
}

func handleAdminCommandAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req CommandAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

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

	if found { w.Write([]byte("OK")) } else { http.Error(w, "AI not found in any room", http.StatusNotFound) }
}

func handleWish(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
	var req WishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }

	found := false
	consumed := false
	
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
					if item.ID == "UTIL_WISH_MACHINE" {
						slotIdx = i
						break
					}
				}
				
				if slotIdx >= 0 {
					// Consume Item
					p.Inventory = append(p.Inventory[:slotIdx], p.Inventory[slotIdx+1:]...)
					gs.RecalculateStats(p)
					consumed = true
				}
			}
			gs.Mutex.Unlock()
			
			// Call AdminWish outside the lock to avoid deadlock
			if consumed {
				gs.AdminWish(req.SessionID, req.Wish)
			}
			
			if found { break }
		}
	}
	
	if !found {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	
	if !consumed {
		http.Error(w, "You do not have a Wish Machine!", http.StatusForbidden)
		return
	}
	
	// Async call to MCP
	go func(sid, wText string) {
		mcpURL := "http://localhost:9091/wish"
		bodyData := map[string]string{
			"session_id": sid,
			"wish":       wText,
		}
		jsonData, _ := json.Marshal(bodyData)
		resp, err := http.Post(mcpURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("Failed to call MCP: %v", err)
			return
		}
		defer resp.Body.Close()
	}(req.SessionID, req.Wish)
	
	w.Write([]byte("Wish granted (Item Consumed)."))
}