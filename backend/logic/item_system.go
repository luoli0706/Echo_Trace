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
	// Offense (Items.md)
	"WPN_SHOCK_T1":     {ID: "WPN_SHOCK_T1", Type: ItemTypeOffense, Name: "Shock", MaxUses: 1, Weight: 2.0, Tier: 1, Value: 100},
	"WPN_STONE":        {ID: "WPN_STONE", Type: ItemTypeOffense, Name: "Stone", MaxUses: 3, Weight: 0.5, Tier: 1, Value: 30},
	"WPN_KNIFE_T2":     {ID: "WPN_KNIFE_T2", Type: ItemTypeOffense, Name: "Knife", MaxUses: 1, Weight: 1.0, Tier: 2, Value: 200},
	"WPN_STUN_GRENADE": {ID: "WPN_STUN_GRENADE", Type: ItemTypeOffense, Name: "Stun Grenade", MaxUses: 1, Weight: 1.5, Tier: 2, Value: 220},
	"WPN_TRACK_DART":   {ID: "WPN_TRACK_DART", Type: ItemTypeOffense, Name: "Track Dart", MaxUses: 1, Weight: 1.0, Tier: 3, Value: 350},
	"WPN_EMP_MINE":     {ID: "WPN_EMP_MINE", Type: ItemTypeOffense, Name: "EMP Mine", MaxUses: 1, Weight: 2.5, Tier: 3, Value: 500},

	// Survival (Items.md)
	"SURV_BANDAGE":     {ID: "SURV_BANDAGE", Type: ItemTypeSurvival, Name: "Bandage", MaxUses: 1, Weight: 0.8, Tier: 1, Value: 60},
	"SURV_ENERGY_BAR":  {ID: "SURV_ENERGY_BAR", Type: ItemTypeSurvival, Name: "Energy Bar", MaxUses: 1, Weight: 0.3, Tier: 1, Value: 40},
	"SURV_ADRENALINE":  {ID: "SURV_ADRENALINE", Type: ItemTypeSurvival, Name: "Adrenaline", MaxUses: 1, Weight: 0.6, Tier: 2, Value: 180},
	"SURV_SILENT_PAD":  {ID: "SURV_SILENT_PAD", Type: ItemTypeSurvival, Name: "Silent Pad", MaxUses: 1, Weight: 1.0, Tier: 2, Value: 160},
	"SURV_JAMMER":      {ID: "SURV_JAMMER", Type: ItemTypeSurvival, Name: "Jammer", MaxUses: 1, Weight: 1.8, Tier: 3, Value: 400},
	"SURV_ARMOR_LIGHT": {ID: "SURV_ARMOR_LIGHT", Type: ItemTypeSurvival, Name: "Armor", MaxUses: 1, Weight: 2.0, Tier: 3, Value: 450},

	// Recon (Items.md)
	"RECON_AMP_T1":      {ID: "RECON_AMP_T1", Type: ItemTypeRecon, Name: "Amp", MaxUses: 1, Weight: 0.8, Tier: 1, Value: 120},
	"RECON_FLASHLIGHT":  {ID: "RECON_FLASHLIGHT", Type: ItemTypeRecon, Name: "Flashlight", MaxUses: 1, Weight: 1.0, Tier: 1, Value: 100},
	"RECON_HEARTBEAT":   {ID: "RECON_HEARTBEAT", Type: ItemTypeRecon, Name: "Heartbeat", MaxUses: 3, Weight: 1.5, Tier: 2, Value: 260},
	"RECON_DRONE_TAG":   {ID: "RECON_DRONE_TAG", Type: ItemTypeRecon, Name: "Drone Tag", MaxUses: 1, Weight: 1.8, Tier: 2, Value: 300},
	"RECON_GLOBAL_SCAN": {ID: "RECON_GLOBAL_SCAN", Type: ItemTypeRecon, Name: "Global Scan", MaxUses: 1, Weight: 3.0, Tier: 3, Value: 600},
	"RECON_XRAY":        {ID: "RECON_XRAY", Type: ItemTypeRecon, Name: "X-Ray", MaxUses: 1, Weight: 2.2, Tier: 3, Value: 650},

	// Scavenge (Items.md)
	"SCAV_BACKPACK_M": {ID: "SCAV_BACKPACK_M", Type: ItemTypeScavenge, Name: "Backpack+", MaxUses: 1, Weight: 1.2, Tier: 1, Value: 150},
	"SCAV_DETECTOR":   {ID: "SCAV_DETECTOR", Type: ItemTypeScavenge, Name: "Detector", MaxUses: 1, Weight: 1.4, Tier: 1, Value: 140},
	"SCAV_DECODER":    {ID: "SCAV_DECODER", Type: ItemTypeScavenge, Name: "Decoder", MaxUses: 1, Weight: 0.8, Tier: 2, Value: 300},
	"SCAV_MASTER_KEY": {ID: "SCAV_MASTER_KEY", Type: ItemTypeScavenge, Name: "Master Key", MaxUses: 1, Weight: 1.0, Tier: 3, Value: 500},
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
	dmgMult := 1.0
	reconMult := 1.0
	if p.Tactic != "" {
		switch p.Tactic {
		case "RECON":
			if gs.Config.Tactics.Recon.HealEffectMult > 0 {
				healMult = gs.Config.Tactics.Recon.HealEffectMult
			}
			if gs.Config.Tactics.Recon.DamageEffectMult > 0 {
				dmgMult = gs.Config.Tactics.Recon.DamageEffectMult
			}
			if gs.Config.Tactics.Recon.ReconEffectMult > 0 {
				reconMult = gs.Config.Tactics.Recon.ReconEffectMult
			}
		case "DEFENSE":
			if gs.Config.Tactics.Defense.HealEffectMult > 0 {
				healMult = gs.Config.Tactics.Defense.HealEffectMult
			}
			if gs.Config.Tactics.Defense.DamageEffectMult > 0 {
				dmgMult = gs.Config.Tactics.Defense.DamageEffectMult
			}
			if gs.Config.Tactics.Defense.ReconEffectMult > 0 {
				reconMult = gs.Config.Tactics.Defense.ReconEffectMult
			}
		case "TRAP":
			if gs.Config.Tactics.Trap.HealEffectMult > 0 {
				healMult = gs.Config.Tactics.Trap.HealEffectMult
			}
			if gs.Config.Tactics.Trap.DamageEffectMult > 0 {
				dmgMult = gs.Config.Tactics.Trap.DamageEffectMult
			}
			if gs.Config.Tactics.Trap.ReconEffectMult > 0 {
				reconMult = gs.Config.Tactics.Trap.ReconEffectMult
			}
		}
	}

	switch item.ID {
	// Survival
	case "SURV_BANDAGE":
		if p.HP < p.MaxHP {
			heal := 30.0 * healMult
			p.HP += heal
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
			used = true
		}
	case "SURV_ENERGY_BAR":
		if p.HP < p.MaxHP {
			heal := 10.0 * healMult
			p.HP += heal
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
			used = true
		}
	case "SURV_ADRENALINE":
		p.BuffSpeedMult = 1.20
		p.BuffSpeedUntil = time.Now().Add(8 * time.Second)
		used = true
	case "SURV_SILENT_PAD":
		p.BuffSilentUntil = time.Now().Add(20 * time.Second)
		used = true
	case "SURV_JAMMER":
		p.BuffJammerUntil = time.Now().Add(12 * time.Second)
		used = true
	case "SURV_ARMOR_LIGHT":
		p.BuffDamageReduction = 0.35
		p.BuffDamageReductionUntil = time.Now().Add(20 * time.Second)
		used = true

	// Offense
	case "WPN_SHOCK_T1", "WPN_KNIFE_T2", "WPN_STUN_GRENADE", "WPN_TRACK_DART", "WPN_EMP_MINE", "WPN_STONE":
		dmg := 5.0
		rangeVal := 3.0
		switch item.ID {
		case "WPN_SHOCK_T1":
			dmg = 20
			rangeVal = 3.0
		case "WPN_KNIFE_T2":
			dmg = 40
			rangeVal = 6.0
		case "WPN_STUN_GRENADE":
			dmg = 10
			rangeVal = 5.0
		case "WPN_TRACK_DART":
			dmg = 30
			rangeVal = 6.0
		case "WPN_EMP_MINE":
			dmg = 50
			rangeVal = 8.0
		case "WPN_STONE":
			dmg = 5
			rangeVal = 6.0
		}
		dmg *= dmgMult

		target := gs.findNearestEnemy(p, rangeVal)
		if target != nil {
			gs.applyDamage(target, dmg)
			used = true
		}

	// Recon
	case "RECON_AMP_T1":
		p.BuffHearBonus = 6.0 * reconMult // 50% of default 12 ~= 6
		p.BuffHearUntil = time.Now().Add(10 * time.Second)
		used = true
	case "RECON_FLASHLIGHT", "RECON_HEARTBEAT", "RECON_DRONE_TAG", "RECON_GLOBAL_SCAN", "RECON_XRAY":
		// Minimal implementation: temporary view radius boost (placeholder for richer intel behaviors).
		bonus := 5.0
		duration := 10 * time.Second
		switch item.ID {
		case "RECON_FLASHLIGHT":
			bonus, duration = 5.0, 12*time.Second
		case "RECON_HEARTBEAT":
			bonus, duration = 6.0, 10*time.Second
		case "RECON_DRONE_TAG":
			bonus, duration = 6.0, 12*time.Second
		case "RECON_GLOBAL_SCAN":
			bonus, duration = 8.0, 15*time.Second
		case "RECON_XRAY":
			bonus, duration = 8.0, 5*time.Second
		}
		p.BuffViewBonus = bonus * reconMult
		p.BuffViewUntil = time.Now().Add(duration)
		used = true

	// Scavenge
	case "SCAV_BACKPACK_M":
		// Convert passive into a timed consumable effect.
		p.BuffInvCapBonus = 2
		p.BuffInvCapUntil = time.Now().Add(30 * time.Second)
		p.BuffMaxWeightBonus = 3.0
		p.BuffMaxWeightUntil = time.Now().Add(30 * time.Second)
		used = true
	case "SCAV_DETECTOR", "SCAV_MASTER_KEY":
		// Minimal implementation: small funds bonus (placeholder for richer behaviors).
		p.Funds += int(50 * float64(item.Tier))
		used = true
	case "SCAV_DECODER":
		// Minimal implementation: if channeling a motor, boost its progress.
		if p.ChannelingTargetUID != "" {
			if ent, ok := gs.Entities[p.ChannelingTargetUID]; ok && ent.Type == EntityTypeMotor {
				if md, ok2 := ent.Extra.(MotorData); ok2 {
					md.Progress += 25
					if md.Progress > md.MaxProgress {
						md.Progress = md.MaxProgress
					}
					ent.Extra = md
					gs.Entities[p.ChannelingTargetUID] = ent
					used = true
				}
			}
		}
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
	dr := 0.0
	now := time.Now()
	if now.Before(target.BuffDamageReductionUntil) && target.BuffDamageReduction > 0 {
		dr = target.BuffDamageReduction
		if dr < 0 {
			dr = 0
		}
		if dr > 0.90 {
			dr = 0.90
		}
	}
	eff := dmg * (1.0 - dr)
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
