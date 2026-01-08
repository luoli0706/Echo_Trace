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

type WishRequest struct {
	AdminRequest
	Wish string `json:"wish"`
}

func findPlayerAndExecute(sessionID string, action func(gs interface{}) bool) bool {
	// Access GlobalManager
	if network.GlobalManager == nil {
		return false
	}
	
	// We need to iterate rooms safely.
	// Since RoomManager.Rooms access needs lock, we use ListRooms then GetRoom
	roomIDs := network.GlobalManager.ListRooms()
	
	for _, rid := range roomIDs {
		room := network.GlobalManager.GetRoom(rid)
		if room == nil { continue }
		
		// Access GameState
		// Room struct has GameLoop, GameLoop has GameState.
		// We need to check if player exists there first to avoid iterating all rooms if possible,
		// but GameState locks are internal.
		// The Admin methods in GameState handle the check "if player exists".
		// But they return false if not found.
		// So we just try every room until one returns true.
		
		if room.GameLoop != nil && room.GameLoop.GameState != nil {
			// Because logic package is imported as "echo_trace_server/logic",
			// and GameState is in that package.
			// But here 'action' takes interface{} because we can't import "logic" inside main easily if main imports "logic"?
			// Wait, main ALREADY imports logic.
			
			// We can cast gs to *logic.GameState in the closure.
			// Actually, let's just pass the room's GameState directly.
			
			// Note: We are accessing GameState field of GameLoop.
			// Is it thread safe? The pointer is constant, the content is mutex-protected.
			// Yes.
			
			// We need to reflect or use type assertion in the caller, but here we are in 'main' package.
			// We should pass the concrete type if possible, but let's see.
			// main.go imports logic.
			
			success := action(room.GameLoop.GameState)
			if success {
				return true
			}
		}
	}
	return false
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
	
	found := findPlayerAndExecute(req.SessionID, func(gs interface{}) bool {
		// Use type assertion to access methods
		// We know it is *logic.GameState but Go doesn't like circular deps if we were in logic.
		// We are in main, so it is fine.
		// But wait, the methods AdminModifyHealth are defined on *GameState.
		// We need to import "echo_trace_server/logic".
		
		// The issue is: how to cast interface{} to *logic.GameState inside main?
		// We need to make sure main.go imports logic. It does.
		
		// We can't use type assertion easily if we don't declare the type here.
		// Let's refactor `findPlayerAndExecute` to take `*logic.GameState`.
		return false // Placeholder, see below
	})
	
	// Since I can't easily write the closure with proper types in this generic helper string,
	// I will just write the loops in the handlers. It's safer.
	
	if !found {
		// Try again with explicit loop
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
	}

	if found {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		http.Error(w, "Player not found", http.StatusNotFound)
	}
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
			
			// Call AdminWish outside the lock to avoid deadlock (AdminWish acquires lock)
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
		// We could read response here if needed
	}(req.SessionID, req.Wish)
	
	w.Write([]byte("Wish granted (Item Consumed)."))
}
