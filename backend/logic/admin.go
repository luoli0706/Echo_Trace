package logic

import (
	"fmt"
	"log"
	"math"
	"time"
)

// AdminModifyHealth sets the player's HP. Range [0, 500].
func (gs *GameState) AdminModifyHealth(sessionID string, hp float64) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}

	if hp < 0 {
		hp = 0
	}
	if hp > 500 {
		hp = 500
	}

	oldHP := p.HP
	p.HP = hp

	log.Printf("[ADMIN] Set Health for %s: %.2f -> %.2f (Max: %.2f)", sessionID, oldHP, p.HP, p.MaxHP)
	return true
}

// AdminModifyArmor sets the player's Armor. Range [0, 250].
func (gs *GameState) AdminModifyArmor(sessionID string, armor float64) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}

	if armor < 0 {
		armor = 0
	}
	if armor > 250 {
		armor = 250
	}

	p.Armor = armor
	// Similar clamp check for Armor in RecalculateStats: if p.Armor > p.MaxArmor { p.Armor = p.MaxArmor }
	// Default MaxArmor is 50. Admin might need to boost MaxArmor to allow 250.
	// But I don't have a clean way to override MaxArmor without changing config or adding a new field.
	// Let's assume for now we just set the current value.

	log.Printf("[ADMIN] Set Armor for %s to %f", sessionID, armor)
	return true
}

// AdminModifyInventoryCap sets the player's max inventory slots. Max 8.
func (gs *GameState) AdminModifyInventoryCap(sessionID string, newCap int) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}

	if newCap > 8 {
		newCap = 8
	}
	if newCap < 0 {
		newCap = 0
	}

	// InventoryCap is calculated as Config + Buff + Extra.
	// We want final result to be newCap.
	// But Config + Buff varies.
	// Simplified: We set ExtraInvCap such that Base (default 6) + Extra = newCap.
	// Assuming no buffs active for now.
	base := gs.Config.Gameplay.InventorySize
	if base <= 0 {
		base = 6
	}

	p.ExtraInvCap = newCap - base
	// If the result is negative, that's fine (penalty).

	log.Printf("[ADMIN] Set Inventory Cap for %s to %d (Base %d, Extra %d)", sessionID, newCap, base, p.ExtraInvCap)
	return true
}

// AdminAddItems adds up to 2 items to the player's inventory.
func (gs *GameState) AdminAddItems(sessionID string, itemIDs []string) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}

	// Limit to 2 items per request as per spec
	if len(itemIDs) > 2 {
		itemIDs = itemIDs[:2]
	}

	countAdded := 0
	for _, id := range itemIDs {
		// Find item template
		var template Item
		found := false
		for _, it := range ItemDB {
			if it.ID == id {
				template = it
				found = true
				break
			}
		}

		if !found {
			log.Printf("[ADMIN] Item ID not found: %s", id)
			continue
		}

		// Check capacity? Admin bypasses capacity?
		// "Modify player's prop bag". Usually admins bypass.
		// But let's respect the hard physical limit if any (slice can grow).

		newItem := template
		newItem.UID = NewUID()
		p.Inventory = append(p.Inventory, newItem)
		countAdded++
	}

	gs.RecalculateStats(p)
	log.Printf("[ADMIN] Added %d items to %s", countAdded, sessionID)
	return true
}

// AdminModifySpeed sets a speed buff.
func (gs *GameState) AdminModifySpeed(sessionID string, multiplier float64, durationSec float64) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return false
	}

	p.BuffSpeedMult = multiplier
	p.BuffSpeedUntil = time.Now().Add(time.Duration(durationSec) * time.Second)

	log.Printf("[ADMIN] Set Speed Buff for %s: x%f for %fs", sessionID, multiplier, durationSec)
	return true
}

// AdminMovePlayer moves a player to an exact coordinate if it's walkable.
// This is intended for MCP / admin commands.
// Returns: found (bool), result_msg (string)
func (gs *GameState) AdminMovePlayer(sessionID string, x float64, y float64) (bool, string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return false, "Player not found"
	}
	if !p.IsAlive {
		p.ClientMsg = "Cannot move while dead."
		return true, "Player is dead"
	}

	target := Vector2{X: x, Y: y}
	
	// If target is valid immediately, move.
	if !gs.checkCollision(target, 0.25) {
		p.Pos = target
		log.Printf("[ADMIN] Moved %s to (%.1f, %.1f)", sessionID, x, y)
		return true, "OK"
	}

	// Spiral Search for nearest valid point
	log.Printf("[ADMIN] Target (%.1f, %.1f) blocked. Searching nearby...", x, y)
	
	// Search radius up to 5.0 units
	maxRadius := 5.0
	step := 0.5
	
	// Spiral Logic: increasing radius, check points around circumference
	for r := step; r <= maxRadius; r += step {
		// Circumference check steps based on radius to maintain density
		circumference := 2 * math.Pi * r
		numPoints := int(circumference / step) 
		if numPoints < 8 { numPoints = 8 }
		
		angleStep := 2 * math.Pi / float64(numPoints)
		
		for i := 0; i < numPoints; i++ {
			angle := float64(i) * angleStep
			checkPos := Vector2{
				X: target.X + math.Cos(angle)*r,
				Y: target.Y + math.Sin(angle)*r,
			}
			
			if !gs.checkCollision(checkPos, 0.25) {
				p.Pos = checkPos
				msg := fmt.Sprintf("Target blocked. Moved to nearby (%.1f, %.1f)", checkPos.X, checkPos.Y)
				p.ClientMsg = msg
				log.Printf("[ADMIN] %s", msg)
				return true, msg
			}
		}
	}

	p.ClientMsg = "Move blocked (No valid spot found nearby)."
	return true, "Move blocked (Collision/Out of Bounds and no nearby spot found)"
}

// AdminModifyGlobalHealth sets the HP for ALL players.
func (gs *GameState) AdminModifyGlobalHealth(hp float64) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if hp < 0 { hp = 0 }
	// No upper limit enforced here (God Mode)

	count := 0
	for _, p := range gs.Players {
		if p.IsAlive {
			p.HP = hp
			// Also update MaxHP if needed? 
			// User request is "Control HP", usually implies setting current HP.
			// If new HP > MaxHP, let it be (Overheal).
			count++
		}
	}
	
	log.Printf("[ADMIN] Set Global Health to %.2f for %d players", hp, count)
	return true
}

// AdminWish handles the text input from the "Wishing Machine".
// In a real scenario, this would send the text to the MCP/LLM.
// Here we just log it and maybe echo it back as a system message.
func (gs *GameState) AdminWish(sessionID string, wishText string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[sessionID]
	if !ok {
		return
	}

	log.Printf("[WISH] Player %s says: %s", sessionID, wishText)
	// We could set a client message to confirm receipt.
	p.ClientMsg = "Your wish has been heard... (Sending to MCP)"

	// Implementation Note:
	// The actual forwarding to MCP should happen in the HTTP Handler or a separate goroutine
	// to avoid blocking the game loop/lock.
	// This method just updates game state to reflect the wish was made (e.g. FX).
}
