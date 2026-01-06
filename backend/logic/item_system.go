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
	ItemTypeScavenge = "SCAVENGE"
)

var ItemDB = map[string]Item{
	// Offense
	"WPN_AP_AMMO":     {ID: "WPN_AP_AMMO", Type: ItemTypeOffense, Name: "AP Ammo", MaxUses: 1, Weight: 2.0, Tier: 1, Value: 150},
	"WPN_BOUNCE_AMMO": {ID: "WPN_BOUNCE_AMMO", Type: ItemTypeOffense, Name: "Reflect Ammo", MaxUses: 1, Weight: 2.0, Tier: 2, Value: 300},
	"WPN_RAILGUN":     {ID: "WPN_RAILGUN", Type: ItemTypeOffense, Name: "Railgun", MaxUses: 1, Weight: 4.0, Tier: 3, Value: 600},

	// Survival
	"SURV_REPAIR":      {ID: "SURV_REPAIR", Type: ItemTypeSurvival, Name: "Repair Kit", MaxUses: 1, Weight: 1.0, Tier: 1, Value: 100},
	"SURV_PHASE_SHIFT": {ID: "SURV_PHASE_SHIFT", Type: ItemTypeSurvival, Name: "Phase Shift", MaxUses: 1, Weight: 2.0, Tier: 2, Value: 350},
	"SURV_PURGE":       {ID: "SURV_PURGE", Type: ItemTypeSurvival, Name: "Purge System", MaxUses: 1, Weight: 2.5, Tier: 3, Value: 500},

	// Recon
	"RECON_SCOPE":  {ID: "RECON_SCOPE", Type: ItemTypeRecon, Name: "Scope", MaxUses: 1, Weight: 1.0, Tier: 1, Value: 120},
	"RECON_SENSOR": {ID: "RECON_SENSOR", Type: ItemTypeRecon, Name: "Rear Sensor", MaxUses: 1, Weight: 1.5, Tier: 2, Value: 250},
	"RECON_SCANNER": {ID: "RECON_SCANNER", Type: ItemTypeRecon, Name: "Global Scan", MaxUses: 1, Weight: 3.0, Tier: 3, Value: 550},

	// Utility (Scavenge/Util renamed in internal logic but keep type consistent)
	"UTIL_BLINK":   {ID: "UTIL_BLINK", Type: ItemTypeScavenge, Name: "Blink", MaxUses: 1, Weight: 1.5, Tier: 1, Value: 150},
	"UTIL_RADAR":   {ID: "UTIL_RADAR", Type: ItemTypeScavenge, Name: "Pulse Radar", MaxUses: 1, Weight: 1.5, Tier: 2, Value: 280},
	"UTIL_STEALTH": {ID: "UTIL_STEALTH", Type: ItemTypeScavenge, Name: "Stealth Cloak", MaxUses: 1, Weight: 2.0, Tier: 3, Value: 600},
}

type weightedChoice[T any] struct {
	Val    T
	Weight float64
}

func pickWeighted[T any](choices []weightedChoice[T]) (T, bool) {
	var zero T
	total := 0.0
	for _, c := range choices {
		if c.Weight > 0 {
			total += c.Weight
		}
	}
	if total <= 0 {
		return zero, false
	}
	r := rand.Float64() * total
	acc := 0.0
	for _, c := range choices {
		w := c.Weight
		if w <= 0 {
			continue
		}
		acc += w
		if r <= acc {
			return c.Val, true
		}
	}
	return choices[len(choices)-1].Val, true
}

func (gs *GameState) currentLootPhaseIndex() int {
	// Map game phases to 1..3 for loot balancing.
	if gs.Phase <= PhaseInit {
		return 1
	}
	if gs.Phase == PhaseSearch {
		return 1
	}
	if gs.Phase == PhaseConflict {
		return 2
	}
	if gs.Phase == PhaseEscape {
		return 3
	}
	return 3
}

func (gs *GameState) getTierWeightsForPhase(phaseIdx int) (t1, t2, t3 float64) {
	// Defaults from Items.md matrix.
	if phaseIdx == 1 {
		t1, t2, t3 = 0.70, 0.25, 0.05
	} else if phaseIdx == 2 {
		t1, t2, t3 = 0.30, 0.50, 0.20
	} else {
		t1, t2, t3 = 0.10, 0.40, 0.50
	}

	if gs.Config != nil {
		cfg := gs.Config.Items.TierWeightsByPhase
		switch phaseIdx {
		case 1:
			if cfg.Phase1.T1+cfg.Phase1.T2+cfg.Phase1.T3 > 0 {
				t1, t2, t3 = cfg.Phase1.T1, cfg.Phase1.T2, cfg.Phase1.T3
			}
		case 2:
			if cfg.Phase2.T1+cfg.Phase2.T2+cfg.Phase2.T3 > 0 {
				t1, t2, t3 = cfg.Phase2.T1, cfg.Phase2.T2, cfg.Phase2.T3
			}
		case 3:
			if cfg.Phase3.T1+cfg.Phase3.T2+cfg.Phase3.T3 > 0 {
				t1, t2, t3 = cfg.Phase3.T1, cfg.Phase3.T2, cfg.Phase3.T3
			}
		}
	}
	return
}

func (gs *GameState) getScavengeShareForPhase(phaseIdx int) float64 {
	// Defaults: P1 high, later lower.
	share := 0.20
	if phaseIdx == 2 {
		share = 0.15
	} else if phaseIdx >= 3 {
		share = 0.10
	}
	if gs.Config != nil {
		cfg := gs.Config.Items.ScavengeShareByPhase
		switch phaseIdx {
		case 1:
			if cfg.Phase1 > 0 {
				share = cfg.Phase1
			}
		case 2:
			if cfg.Phase2 > 0 {
				share = cfg.Phase2
			}
		case 3:
			if cfg.Phase3 > 0 {
				share = cfg.Phase3
			}
		}
	}
	if share < 0 {
		share = 0
	}
	if share > 0.9 {
		share = 0.9
	}
	return share
}

func (gs *GameState) pickTacticForLoot() string {
	// Use a random alive player's tactic so the room's loot reflects team choices.
	players := make([]*Player, 0, len(gs.Players))
	for _, p := range gs.Players {
		if p != nil && p.IsAlive {
			players = append(players, p)
		}
	}
	if len(players) == 0 {
		return "RECON"
	}
	p := players[rand.Intn(len(players))]
	if p.Tactic == "" {
		return "RECON"
	}
	return p.Tactic
}

func (gs *GameState) pickLootCategory(phaseIdx int, tactic string) string {
	scavShare := gs.getScavengeShareForPhase(phaseIdx)
	remaining := 1.0 - scavShare
	if remaining < 0 {
		remaining = 0
	}

	focusShare := 0.50
	if gs.Config != nil && gs.Config.Items.TacticFocusShare > 0 {
		focusShare = gs.Config.Items.TacticFocusShare
	}
	if focusShare < 0.34 {
		focusShare = 0.34
	}
	if focusShare > 0.80 {
		focusShare = 0.80
	}
	otherShare := (1.0 - focusShare) / 2.0

	// Existing tactics map onto the 3 non-scavenge categories:
	// RECON -> Recon focus, DEFENSE -> Survival focus, TRAP -> Offense focus.
	focus := ItemTypeRecon
	if tactic == "DEFENSE" {
		focus = ItemTypeSurvival
	} else if tactic == "TRAP" {
		focus = ItemTypeOffense
	}

	weights := []weightedChoice[string]{
		{Val: ItemTypeScavenge, Weight: scavShare},
		{Val: ItemTypeOffense, Weight: remaining * otherShare},
		{Val: ItemTypeSurvival, Weight: remaining * otherShare},
		{Val: ItemTypeRecon, Weight: remaining * otherShare},
	}
	for i := range weights {
		if weights[i].Val == focus {
			weights[i].Weight = remaining * focusShare
		}
	}

	cat, ok := pickWeighted(weights)
	if !ok {
		return ItemTypeSurvival
	}
	return cat
}

func (gs *GameState) pickLootTier(phaseIdx int) int {
	t1, t2, t3 := gs.getTierWeightsForPhase(phaseIdx)
	tier, ok := pickWeighted([]weightedChoice[int]{
		{Val: 1, Weight: t1},
		{Val: 2, Weight: t2},
		{Val: 3, Weight: t3},
	})
	if !ok {
		return 1
	}
	return tier
}

func (gs *GameState) pickItemID(category string, tier int) (string, bool) {
	ids := make([]string, 0)
	for id, it := range ItemDB {
		if it.Type == category && it.Tier == tier {
			ids = append(ids, id)
		}
	}
	// Fallback: allow <= tier
	if len(ids) == 0 {
		for id, it := range ItemDB {
			if it.Type == category && it.Tier <= tier {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return "", false
	}
	return ids[rand.Intn(len(ids))], true
}

func (gs *GameState) spawnWeightedRandomItemAt(pos Vector2) bool {
	phaseIdx := gs.currentLootPhaseIndex()
	tactic := gs.pickTacticForLoot()
	cat := gs.pickLootCategory(phaseIdx, tactic)
	tier := gs.pickLootTier(phaseIdx)
	itemID, ok := gs.pickItemID(cat, tier)
	if !ok {
		return false
	}
	item := ItemDB[itemID]
	item.UID = NewUID()
	gs.Entities[item.UID] = Entity{
		UID:   item.UID,
		Type:  EntityTypeItemDrop,
		Pos:   pos,
		State: 1,
		Extra: item,
	}
	return true
}

func (gs *GameState) generateShopStock(phaseIdx int, tactic string) []string {
	count := 6
	if gs.Config != nil && gs.Config.Items.MerchantStockSize > 0 {
		count = gs.Config.Items.MerchantStockSize
	}
	if count < 3 {
		count = 3
	}
	if count > 6 {
		count = 6
	}

	stock := make([]string, 0, count)
	seen := map[string]bool{}
	// Try to build a diverse stock; cap attempts to avoid infinite loops.
	for attempts := 0; len(stock) < count && attempts < 200; attempts++ {
		cat := gs.pickLootCategory(phaseIdx, tactic)
		tier := gs.pickLootTier(phaseIdx)
		id, ok := gs.pickItemID(cat, tier)
		if !ok {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		stock = append(stock, id)
	}

	// Fallback: fill anything.
	if len(stock) < count {
		for id := range ItemDB {
			if len(stock) >= count {
				break
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			stock = append(stock, id)
		}
	}

	return stock
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
	// Supply drop tier roughly follows phase.
	targetTier := phase + 1
	if targetTier > 3 {
		targetTier = 3
	}

	// Generate 1-3 items. Phase 3 likely gives 3.
	count := 1 + (time.Now().UnixNano() % 3) // 1-3
	if phase >= 3 {
		count = 3
	}

	items := []Item{}
	for i := 0; i < int(count); i++ {
		// Bias by current room tactic/phase, but cap to drop tier.
		phaseIdx := gs.currentLootPhaseIndex()
		tactic := gs.pickTacticForLoot()
		cat := gs.pickLootCategory(phaseIdx, tactic)
		tier := gs.pickLootTier(phaseIdx)
		if tier > targetTier {
			tier = targetTier
		}
		itemID, ok := gs.pickItemID(cat, tier)
		if !ok {
			continue
		}
		it := ItemDB[itemID]
		it.UID = NewUID()
		items = append(items, it)
	}
	if len(items) == 0 {
		return
	}

	drop := SupplyDropData{Funds: 500 * targetTier, Items: items}

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
	if gs.spawnWeightedRandomItemAt(pos) {
		log.Printf("Spawned Item at %v", pos)
	}
}

func (gs *GameState) HandlePickup(playerID string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.Phase == PhaseInit {
		return
	}

	p, ok := gs.Players[playerID]
	if !ok || !p.IsAlive {
		return
	}

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
			if len(p.Inventory) < p.InventoryCap {
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
				if len(p.Inventory) < p.InventoryCap {
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

	if gs.Phase == PhaseInit {
		return
	}

	p, ok := gs.Players[playerID]
	if !ok || !p.IsAlive {
		return
	}

	if slotIndex < 0 || slotIndex >= len(p.Inventory) {
		return
	}

	item := p.Inventory[slotIndex]
	used := false

	// Tactic effect multipliers (defaults 1.0).
	healMult := 1.0
	// dmgMult := 1.0 // Unused in new logic
	reconMult := 1.0
	if p.Tactic != "" {
		switch p.Tactic {
		case "RECON":
			if gs.Config.Tactics.Recon.HealEffectMult > 0 {
				healMult = gs.Config.Tactics.Recon.HealEffectMult
			}
			// if gs.Config.Tactics.Recon.DamageEffectMult > 0 {
			// 	dmgMult = gs.Config.Tactics.Recon.DamageEffectMult
			// }
			if gs.Config.Tactics.Recon.ReconEffectMult > 0 {
				reconMult = gs.Config.Tactics.Recon.ReconEffectMult
			}
		case "DEFENSE":
			if gs.Config.Tactics.Defense.HealEffectMult > 0 {
				healMult = gs.Config.Tactics.Defense.HealEffectMult
			}
			// if gs.Config.Tactics.Defense.DamageEffectMult > 0 {
			// 	dmgMult = gs.Config.Tactics.Defense.DamageEffectMult
			// }
			if gs.Config.Tactics.Defense.ReconEffectMult > 0 {
				reconMult = gs.Config.Tactics.Defense.ReconEffectMult
			}
		case "TRAP":
			if gs.Config.Tactics.Trap.HealEffectMult > 0 {
				healMult = gs.Config.Tactics.Trap.HealEffectMult
			}
			// if gs.Config.Tactics.Trap.DamageEffectMult > 0 {
			// 	dmgMult = gs.Config.Tactics.Trap.DamageEffectMult
			// }
			if gs.Config.Tactics.Trap.ReconEffectMult > 0 {
				reconMult = gs.Config.Tactics.Trap.ReconEffectMult
			}
		}
	}

	switch item.ID {
	// --- Offense ---
	case "WPN_AP_AMMO":
		p.AmmoType = "AP"
		p.AmmoCount = 3
		used = true
	case "WPN_BOUNCE_AMMO":
		p.AmmoType = "BOUNCE"
		p.AmmoCount = 3
		used = true
	case "WPN_RAILGUN":
		p.AmmoType = "RAILGUN"
		p.AmmoCount = 1
		used = true

	// --- Survival ---
	case "SURV_REPAIR":
		// Restore 30% HP, then 20 Armor
		maxHP := p.MaxHP
		if maxHP <= 0 { maxHP = 100 }
		heal := maxHP * 0.30 * healMult
		
		p.HP += heal
		if p.HP > p.MaxHP {
			overflow := p.HP - p.MaxHP
			p.HP = p.MaxHP
			// Overflow converts to armor (up to max + temp?)
			// For simplicity: just add 20 flat armor + overflow
			p.Armor += 20 + overflow
			if p.Armor > p.MaxArmor + 50 { // Cap temp armor
				p.Armor = p.MaxArmor + 50
			}
		} else {
			p.Armor += 20
			if p.Armor > p.MaxArmor {
				p.Armor = p.MaxArmor
			}
		}
		used = true

	case "SURV_PHASE_SHIFT":
		// 3.0s Invincible, cannot shoot (Handled in HandleFire check)
		p.BuffInvincibleUntil = time.Now().Add(3 * time.Second)
		used = true
	
	case "SURV_PURGE":
		// 2.0s Invincible + Clear Debuffs
		p.BuffInvincibleUntil = time.Now().Add(2 * time.Second)
		// Clear debuffs (if any implemented later, e.g. Track Dart)
		used = true

	// --- Recon ---
	case "RECON_SCOPE":
		// +20% View Radius for 15s
		p.BuffViewBonus = p.ViewRadius * 0.20 * reconMult
		p.BuffViewUntil = time.Now().Add(15 * time.Second)
		used = true

	case "RECON_SENSOR":
		// Rear Vision (Client side rendering trick)
		p.BuffVisionInvertUntil = time.Now().Add(10 * time.Second)
		used = true

	case "RECON_SCANNER":
		// Global Scan: Reveal nearest enemy to EVERYONE
		target := gs.findNearestEnemy(p, 9999.0)
		if target != nil {
			gs.addEvent("SCAN", fmt.Sprintf("SCAN DETECTED: %s at [%d, %d]", target.Name, int(target.Pos.X), int(target.Pos.Y)))
			// Also set a buff on self to show UI indicator? 
			// Or just the global event is enough for Alpha.
			p.BuffScanUntil = time.Now().Add(30 * time.Second)
			used = true
		} else {
			p.ClientMsg = "No targets found."
		}

	// --- Utility ---
	case "UTIL_BLINK":
		// Teleport 2.0m forward
		dist := 2.0
		dx := p.LookDir.X * dist
		dy := p.LookDir.Y * dist
		dest := Vector2{X: p.Pos.X + dx, Y: p.Pos.Y + dy}
		if !gs.checkCollision(dest, 0.25) {
			p.Pos = dest
			used = true
		} else {
			p.ClientMsg = "Blink blocked!"
		}

	case "UTIL_RADAR":
		// Pulse Radar (Client side blips)
		// We'll handle this by sending a specific packet or just let client render blips if it has item?
		// Item description says "active use". So maybe it reveals blips for 10s?
		// For Alpha, let's say it gives a "Radar Buff"
		// But types.go doesn't have it. Let's reuse Vision Bonus for now or just skip complex logic.
		// Actually, let's just mark it used and send a message.
		p.ClientMsg = "Radar Active (Not Implemented visually yet)"
		used = true

	case "UTIL_STEALTH":
		// Invisible for 15s
		p.BuffInvisibleUntil = time.Now().Add(15 * time.Second)
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

func (gs *GameState) applyDamage(target *Player, dmg float64) {
	if target == nil || dmg <= 0 {
		return
	}
	// Damage Reduction Buff removed. Armor is the new mitigation.
	
	eff := dmg 
	target.HP -= eff
	if target.HP <= 0 {
		target.HP = 0
		target.IsAlive = false
		gs.handleDeath(target)
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

var globalUIDCounter int64

func NewUID() string {
	val := atomic.AddInt64(&globalUIDCounter, 1)
	return fmt.Sprintf("ent_%d_%d", time.Now().UnixNano(), val)
}
