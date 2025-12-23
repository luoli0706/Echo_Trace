package logic

import (
	"log"
	"time"
)

const (
	ItemTypeOffense  = "OFFENSE"
	ItemTypeSurvival = "SURVIVAL"
	ItemTypeRecon    = "RECON"
)

var ItemDB = map[string]Item{
	"WPN_SHOCK":   {ID: "WPN_SHOCK", Type: ItemTypeOffense, Name: "Stun Gun", MaxUses: 1, Weight: 2.0, Tier: 1},
	"WPN_SHOCK_T2": {ID: "WPN_SHOCK_T2", Type: ItemTypeOffense, Name: "Taser X2", MaxUses: 2, Weight: 2.0, Tier: 2},
	"WPN_SHOCK_T3": {ID: "WPN_SHOCK_T3", Type: ItemTypeOffense, Name: "Volt Rifle", MaxUses: 3, Weight: 3.5, Tier: 3},

	"SURV_MEDKIT": {ID: "SURV_MEDKIT", Type: ItemTypeSurvival, Name: "MedKit", MaxUses: 1, Weight: 1.5, Tier: 1},
	"SURV_MEDKIT_T2": {ID: "SURV_MEDKIT_T2", Type: ItemTypeSurvival, Name: "MedKit+", MaxUses: 2, Weight: 1.5, Tier: 2},

	"RECON_RADAR": {ID: "RECON_RADAR", Type: ItemTypeRecon, Name: "Scanner", MaxUses: 3, Weight: 3.0, Tier: 1},
	"RECON_RADAR_T2": {ID: "RECON_RADAR_T2", Type: ItemTypeRecon, Name: "Scanner Pro", MaxUses: 5, Weight: 2.5, Tier: 2},
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
				p.Funds += 50 // Pickups give small funds
				delete(gs.Entities, targetUID)
				gs.RecalculateStats(p)
				log.Printf("Player %s picked up %s", playerID, item.ID)
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
	case "SURV_MEDKIT":
		if p.HP < p.MaxHP {
			p.HP += 50
			if p.HP > p.MaxHP { p.HP = p.MaxHP }
			used = true
		}
	case "WPN_SHOCK":
		target := gs.findNearestEnemy(p, 3.0)
		if target != nil {
			target.HP -= 40
			if target.HP <= 0 {
				target.HP = 0
				target.IsAlive = false
				gs.handleDeath(target)
			}
			used = true
		}
	case "RECON_RADAR":
		p.ViewRadius += 5.0
		go func() {
			time.Sleep(10 * time.Second)
			gs.Mutex.Lock()
			if p.IsAlive { p.ViewRadius -= 5.0 }
			gs.Mutex.Unlock()
		}()
		used = true
	}

	if used {
		p.Inventory = append(p.Inventory[:slotIndex], p.Inventory[slotIndex+1:]...)
		gs.RecalculateStats(p)
	}
}

func (gs *GameState) HandleAttack(attackerID string, targetID string) bool {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	attacker, ok := gs.Players[attackerID]
	if !ok || !attacker.IsAlive { return false }

	var target *Player
	if targetID != "" {
		target = gs.Players[targetID]
	} else {
		target = gs.findNearestEnemy(attacker, 2.0)
	}

	if target != nil {
		dmg := 25.0
		target.HP -= dmg
		log.Printf("Player %s hit %s. HP: %.1f", attackerID, target.SessionID, target.HP)
		
		if target.HP <= 0 {
			target.HP = 0
			target.IsAlive = false
			gs.handleDeath(target)
		}
		return true
	}
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

func NewUID() string {
	return  "ent_" + string(time.Now().UnixNano())
}