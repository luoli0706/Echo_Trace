package logic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"sync/atomic"
	"time"
)

const (
	ItemTypeOffense  = "OFFENSE"
	ItemTypeSurvival = "SURVIVAL"
	ItemTypeRecon    = "RECON"
)

var ItemDB = map[string]Item{
	"WPN_SHOCK":      {ID: "WPN_SHOCK", Type: ItemTypeOffense, Name: "Stun Gun", MaxUses: 1, Weight: 2.0, Tier: 1, Value: 100},
	"WPN_SHOCK_T2":   {ID: "WPN_SHOCK_T2", Type: ItemTypeOffense, Name: "Taser X2", MaxUses: 2, Weight: 2.0, Tier: 2, Value: 200},
	"WPN_SHOCK_T3":   {ID: "WPN_SHOCK_T3", Type: ItemTypeOffense, Name: "Volt Rifle", MaxUses: 3, Weight: 3.5, Tier: 3, Value: 350},

	"SURV_MEDKIT":    {ID: "SURV_MEDKIT", Type: ItemTypeSurvival, Name: "MedKit", MaxUses: 1, Weight: 1.5, Tier: 1, Value: 50},
	"SURV_MEDKIT_T2": {ID: "SURV_MEDKIT_T2", Type: ItemTypeSurvival, Name: "MedKit+", MaxUses: 2, Weight: 1.5, Tier: 2, Value: 100},

	"RECON_RADAR":    {ID: "RECON_RADAR", Type: ItemTypeRecon, Name: "Scanner", MaxUses: 3, Weight: 3.0, Tier: 1, Value: 150},
	"RECON_RADAR_T2": {ID: "RECON_RADAR_T2", Type: ItemTypeRecon, Name: "Scanner Pro", MaxUses: 5, Weight: 2.5, Tier: 2, Value: 300},
	"RECON_RADAR_T3": {ID: "RECON_RADAR_T3", Type: ItemTypeRecon, Name: "Global Scanner", MaxUses: 1, Weight: 4.0, Tier: 3, Value: 500},
}

func LoadItemValues() {
	absPath, _ := filepath.Abs("../item_values.json")
	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		log.Printf("Warning: Could not load item_values.json: %v", err)
		return
	}
	
	var values map[string]int
	if err := json.Unmarshal(data, &values); err != nil {
		log.Printf("Warning: Parse item_values error: %v", err)
		return
	}
	
	for id, val := range values {
		if item, ok := ItemDB[id]; ok {
			item.Value = val
			ItemDB[id] = item
		}
	}
	log.Println("Item Values Loaded.")
}

func (gs *GameState) SpawnSupplyDrop(pos Vector2, phase int) {
	// Drop Tier = Phase + 1
	targetTier := phase + 1
	if targetTier > 3 { targetTier = 3 }

	// Collect items matching tier (fallback to lower tier if none)
	candidates := []Item{}
	for _, it := range ItemDB {
		if it.Tier <= targetTier {
			candidates = append(candidates, it)
		}
	}

	if len(candidates) == 0 { return }

	// Generate 1-3 items. Phase 3 likely gives 3.
	count := 1 + (time.Now().UnixNano() % 3) // 1-3
	if phase >= 3 { count = 3 }
	
	items := []Item{}
	for i := 0; i < int(count); i++ {
		idx := time.Now().UnixNano() % int64(len(candidates))
		it := candidates[idx]
		it.UID = NewUID()
		items = append(items, it)
	}
	
	drop := SupplyDropData{
		Funds: 500 * targetTier,
		Items: items,
	}
	
	uid := NewUID()
	gs.Entities[uid] = Entity{
		UID:   uid,
		Type:  EntityTypeSupplyDrop,
		Pos:   pos,
		State: 1,
		Extra: drop,
	}
	log.Printf("Spawned Supply Drop at %v (Tier %d)", pos, targetTier)
}

func (gs *GameState) SpawnRandomItem(pos Vector2) {
	keys := []string{"WPN_SHOCK", "SURV_MEDKIT", "RECON_RADAR"}
	choice := keys[time.Now().UnixNano()%3]
	
	item := ItemDB[choice]
	item.UID = NewUID()
	
	entity := Entity{
		UID:   item.UID,
		Type:  EntityTypeItemDrop,
		Pos:   pos,
		State: 1, 
		Extra: item,
	}
	// Use Map
	gs.Entities[entity.UID] = entity
	log.Printf("Spawned Item %s at %v", choice, pos)
}

func (gs *GameState) HandlePickup(playerID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	
	if gs.Phase == PhaseInit { return }

	p, ok := gs.Players[playerID]
	if !ok || !p.IsAlive { return }

	pickupRange := 1.5
	var targetUID = ""

	for uid, e := range gs.Entities {
		if e.Type == EntityTypeItemDrop || e.Type == EntityTypeSupplyDrop {
			if Distance(p.Pos, e.Pos) <= pickupRange {
				targetUID = uid
				break
			}
		}
	}

	if targetUID != "" {
		ent := gs.Entities[targetUID]
		
		if ent.Type == EntityTypeItemDrop {
			if len(p.Inventory) < 6 {
				item := ent.Extra.(Item)
				p.Inventory = append(p.Inventory, item)
				
				// Random Funds 20-80
				gain := 20 + rand.Intn(61)
				p.Funds += gain 
				
				delete(gs.Entities, targetUID)
				gs.RecalculateStats(p)
				log.Printf("Player %s picked up %s (+$%d)", playerID, item.ID, gain)
			}
		} else if ent.Type == EntityTypeSupplyDrop {
			data := ent.Extra.(SupplyDropData)
			p.Funds += data.Funds
			
			// Try add all items
			addedCount := 0
			for _, item := range data.Items {
				if len(p.Inventory) < 6 {
					p.Inventory = append(p.Inventory, item)
					addedCount++
				}
			}
			delete(gs.Entities, targetUID)
			gs.RecalculateStats(p)
			gs.addEvent("SUPPLY_CLAIMED", "A Supply Drop has been claimed!")
			log.Printf("Player %s claimed Supply Drop (+%d funds, %d items)", playerID, data.Funds, addedCount)
		}
	}
}

func (gs *GameState) HandleBuyItem(playerID string, itemID string) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	p, ok := gs.Players[playerID]
	if !ok || !p.IsAlive { return false }
	
	// Check Phase/Tier Restriction
	currentTier := gs.Phase
	if currentTier < 1 { currentTier = 1 }
	if currentTier > 3 { currentTier = 3 }

	itemTemplate, valid := ItemDB[itemID]
	if !valid { return false }
	
	if itemTemplate.Tier > currentTier {
		return false // Cannot buy higher tier
	}
	
	cost := itemTemplate.Value
	if p.Funds >= cost {
		if len(p.Inventory) < 6 {
			p.Funds -= cost
			newItem := itemTemplate
			newItem.UID = NewUID()
			p.Inventory = append(p.Inventory, newItem)
			gs.RecalculateStats(p)
			log.Printf("Player %s bought %s for $%d", playerID, itemID, cost)
			return true
		}
	}
	return false
}

func (gs *GameState) HandleUseItem(playerID string, slotIndex int) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	
	if gs.Phase == PhaseInit { return }

	p, ok := gs.Players[playerID]
	if !ok || !p.IsAlive { return }

	if slotIndex < 0 || slotIndex >= len(p.Inventory) { return }

	item := p.Inventory[slotIndex]
	used := false

	switch item.ID {
	case "SURV_MEDKIT", "SURV_MEDKIT_T2":
		if p.HP < p.MaxHP {
			heal := 50.0
			if item.ID == "SURV_MEDKIT_T2" { heal = 100.0 }
			
			p.HP += heal
			if p.HP > p.MaxHP { p.HP = p.MaxHP }
			used = true
		}
	case "WPN_SHOCK", "WPN_SHOCK_T2", "WPN_SHOCK_T3":
		dmg := 40.0
		rangeVal := 3.0
		if item.ID == "WPN_SHOCK_T2" { dmg = 60.0; rangeVal = 4.0 }
		if item.ID == "WPN_SHOCK_T3" { dmg = 80.0; rangeVal = 6.0 }

		target := gs.findNearestEnemy(p, rangeVal)
		if target != nil {
			target.HP -= dmg
			if target.HP <= 0 {
				target.HP = 0
				target.IsAlive = false
				gs.handleDeath(target)
			}
			used = true
		}
	case "RECON_RADAR", "RECON_RADAR_T2", "RECON_RADAR_T3":
		bonus := 5.0
		duration := 10 * time.Second
		if item.ID == "RECON_RADAR_T2" { bonus = 10.0; duration = 15 * time.Second }
		if item.ID == "RECON_RADAR_T3" { bonus = 20.0; duration = 30 * time.Second }

		p.ViewRadius += bonus
		go func(player *Player, b float64, d time.Duration) {
			time.Sleep(d)
			gs.Mutex.Lock()
			if player.IsAlive { player.ViewRadius -= b }
			gs.Mutex.Unlock()
		}(p, bonus, duration)
		used = true
	}

	if used {
		// Reduce Uses or Remove
		if item.MaxUses > 1 {
			// Not implemented uses tracking yet, just consume for now for Alpha
			// Ideally update MaxUses-- and check if 0
		}
		p.Inventory = append(p.Inventory[:slotIndex], p.Inventory[slotIndex+1:]...)
		gs.RecalculateStats(p)
	}
}

func (gs *GameState) HandleAttack(attackerID string, targetID string) bool {
	// Basic Attack logic if we want default melee, currently disabled/unused by frontend
	return false
}

func (gs *GameState) findNearestEnemy(attacker *Player, rng float64) *Player {
	var target *Player
	minDist := rng
	for _, p := range gs.Players {
		if p.SessionID != attacker.SessionID && p.IsAlive {
			d := Distance(attacker.Pos, p.Pos)
			if d < minDist && d <= attacker.ViewRadius {
				target = p
				minDist = d
			}
		}
	}
	return target
}

func (gs *GameState) handleDeath(p *Player) {
	for _, item := range p.Inventory {
		uid := NewUID()
		ent := Entity{
			UID:   uid,
			Type:  EntityTypeItemDrop,
			Pos:   p.Pos,
			State: 1,
			Extra: item,
		}
		gs.Entities[uid] = ent
	}
	p.Inventory = []Item{}
	log.Printf("Player %s died.", p.SessionID)
}

var globalUIDCounter int64

func NewUID() string {
	val := atomic.AddInt64(&globalUIDCounter, 1)
	return fmt.Sprintf("ent_%d_%d", time.Now().UnixNano(), val)
}
