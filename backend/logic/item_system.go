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
	"WPN_SHOCK":   {ID: "WPN_SHOCK", Type: ItemTypeOffense, Name: "Stun Gun", MaxUses: 1},
	"SURV_MEDKIT": {ID: "SURV_MEDKIT", Type: ItemTypeSurvival, Name: "MedKit", MaxUses: 1},
	"RECON_RADAR": {ID: "RECON_RADAR", Type: ItemTypeRecon, Name: "Scanner", MaxUses: 3},
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

	p, ok := gs.Players[playerID]
	if !ok || !p.IsAlive { return }

	pickupRange := 1.5
	var targetUID = ""

	for uid, e := range gs.Entities {
		if e.Type == EntityTypeItemDrop {
			if Distance(p.Pos, e.Pos) <= pickupRange {
				targetUID = uid
				break
			}
		}
	}

	if targetUID != "" {
		ent := gs.Entities[targetUID]
		if len(p.Inventory) < 6 {
			item := ent.Extra.(Item)
			p.Inventory = append(p.Inventory, item)
			delete(gs.Entities, targetUID)
			log.Printf("Player %s picked up %s", playerID, item.ID)
		}
	}
}

func (gs *GameState) HandleUseItem(playerID string, slotIndex int) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

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