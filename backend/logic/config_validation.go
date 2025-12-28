package logic

import "math"

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func clampFloat(v, minV, maxV float64) float64 {
	if math.IsNaN(v) {
		return minV
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// ClampGameConfig enforces hard safety bounds for room configs.
// It mutates cfg in-place so callers can accept user-provided values while guaranteeing sane limits.
func ClampGameConfig(cfg *GameConfig) {
	if cfg == nil {
		return
	}

	// --- server ---
	cfg.Server.TickRateMs = clampInt(cfg.Server.TickRateMs, 10, 200)
	cfg.Server.MaxPlayers = clampInt(cfg.Server.MaxPlayers, 1, 16)
	cfg.Server.WaitForPlayersTimeoutSec = clampInt(cfg.Server.WaitForPlayersTimeoutSec, 5, 300)
	cfg.Server.DisconnectGraceSec = clampInt(cfg.Server.DisconnectGraceSec, 0, 600)

	// --- map ---
	cfg.Map.Width = clampInt(cfg.Map.Width, 16, 256)
	cfg.Map.Height = clampInt(cfg.Map.Height, 16, 256)
	cfg.Map.AOIGridSize = clampInt(cfg.Map.AOIGridSize, 4, 64)
	cfg.Map.WallDensity = clampFloat(cfg.Map.WallDensity, 0.0, 0.6)

	// --- gameplay ---
	cfg.Gameplay.InventorySize = clampInt(cfg.Gameplay.InventorySize, 1, 12)
	cfg.Gameplay.SafeSlotCount = clampInt(cfg.Gameplay.SafeSlotCount, 0, 4)
	if cfg.Gameplay.SafeSlotCount > cfg.Gameplay.InventorySize {
		cfg.Gameplay.SafeSlotCount = cfg.Gameplay.InventorySize
	}
	cfg.Gameplay.BaseMoveSpeed = clampFloat(cfg.Gameplay.BaseMoveSpeed, 0.5, 10.0)
	cfg.Gameplay.BaseViewRadius = clampFloat(cfg.Gameplay.BaseViewRadius, 1.0, 20.0)
	cfg.Gameplay.HearRadius = clampFloat(cfg.Gameplay.HearRadius, 1.0, 30.0)
	cfg.Gameplay.BaseMaxHP = clampFloat(cfg.Gameplay.BaseMaxHP, 10.0, 300.0)
	cfg.Gameplay.BaseMaxWeight = clampFloat(cfg.Gameplay.BaseMaxWeight, 1.0, 50.0)
	cfg.Gameplay.WeightThresholdNoiseDouble = clampFloat(cfg.Gameplay.WeightThresholdNoiseDouble, 0.0, 1.0)
	cfg.Gameplay.WeightThresholdViewReduce = clampFloat(cfg.Gameplay.WeightThresholdViewReduce, 0.0, 1.0)
	cfg.Gameplay.WeightThresholdImmobilize = clampFloat(cfg.Gameplay.WeightThresholdImmobilize, 0.0, 1.0)

	// --- items ---
	cfg.Items.InitialWorldItemCount = clampInt(cfg.Items.InitialWorldItemCount, 0, 500)
	cfg.Items.RespawnIntervalSec = clampFloat(cfg.Items.RespawnIntervalSec, 0.5, 30.0)
	cfg.Items.MerchantStockSize = clampInt(cfg.Items.MerchantStockSize, 1, 6)
	cfg.Items.MerchantRefreshCost = clampInt(cfg.Items.MerchantRefreshCost, 0, 10000)
	cfg.Items.MaxWorldItemCount.Phase1 = clampInt(cfg.Items.MaxWorldItemCount.Phase1, 0, 1000)
	cfg.Items.MaxWorldItemCount.Phase2 = clampInt(cfg.Items.MaxWorldItemCount.Phase2, 0, 1000)
	cfg.Items.MaxWorldItemCount.Phase3 = clampInt(cfg.Items.MaxWorldItemCount.Phase3, 0, 1000)

	cfg.Items.TierWeightsByPhase.Phase1.T1 = clampFloat(cfg.Items.TierWeightsByPhase.Phase1.T1, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase1.T2 = clampFloat(cfg.Items.TierWeightsByPhase.Phase1.T2, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase1.T3 = clampFloat(cfg.Items.TierWeightsByPhase.Phase1.T3, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase2.T1 = clampFloat(cfg.Items.TierWeightsByPhase.Phase2.T1, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase2.T2 = clampFloat(cfg.Items.TierWeightsByPhase.Phase2.T2, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase2.T3 = clampFloat(cfg.Items.TierWeightsByPhase.Phase2.T3, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase3.T1 = clampFloat(cfg.Items.TierWeightsByPhase.Phase3.T1, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase3.T2 = clampFloat(cfg.Items.TierWeightsByPhase.Phase3.T2, 0.0, 1.0)
	cfg.Items.TierWeightsByPhase.Phase3.T3 = clampFloat(cfg.Items.TierWeightsByPhase.Phase3.T3, 0.0, 1.0)

	cfg.Items.ScavengeShareByPhase.Phase1 = clampFloat(cfg.Items.ScavengeShareByPhase.Phase1, 0.0, 1.0)
	cfg.Items.ScavengeShareByPhase.Phase2 = clampFloat(cfg.Items.ScavengeShareByPhase.Phase2, 0.0, 1.0)
	cfg.Items.ScavengeShareByPhase.Phase3 = clampFloat(cfg.Items.ScavengeShareByPhase.Phase3, 0.0, 1.0)
	cfg.Items.TacticFocusShare = clampFloat(cfg.Items.TacticFocusShare, 0.0, 1.0)

	// --- tactics multipliers ---
	cfg.Tactics.Recon.MaxHPMult = clampFloat(cfg.Tactics.Recon.MaxHPMult, 0.5, 2.0)
	cfg.Tactics.Recon.MoveSpeedMult = clampFloat(cfg.Tactics.Recon.MoveSpeedMult, 0.5, 2.0)
	cfg.Tactics.Recon.ViewRadiusMult = clampFloat(cfg.Tactics.Recon.ViewRadiusMult, 0.5, 2.0)
	cfg.Tactics.Recon.HearRadiusMult = clampFloat(cfg.Tactics.Recon.HearRadiusMult, 0.5, 2.0)
	cfg.Tactics.Recon.HealEffectMult = clampFloat(cfg.Tactics.Recon.HealEffectMult, 0.5, 2.0)
	cfg.Tactics.Recon.DamageEffectMult = clampFloat(cfg.Tactics.Recon.DamageEffectMult, 0.5, 2.0)
	cfg.Tactics.Recon.ReconEffectMult = clampFloat(cfg.Tactics.Recon.ReconEffectMult, 0.5, 2.0)

	cfg.Tactics.Defense.MaxHPMult = clampFloat(cfg.Tactics.Defense.MaxHPMult, 0.5, 2.0)
	cfg.Tactics.Defense.MoveSpeedMult = clampFloat(cfg.Tactics.Defense.MoveSpeedMult, 0.5, 2.0)
	cfg.Tactics.Defense.ViewRadiusMult = clampFloat(cfg.Tactics.Defense.ViewRadiusMult, 0.5, 2.0)
	cfg.Tactics.Defense.HearRadiusMult = clampFloat(cfg.Tactics.Defense.HearRadiusMult, 0.5, 2.0)
	cfg.Tactics.Defense.HealEffectMult = clampFloat(cfg.Tactics.Defense.HealEffectMult, 0.5, 2.0)
	cfg.Tactics.Defense.DamageEffectMult = clampFloat(cfg.Tactics.Defense.DamageEffectMult, 0.5, 2.0)
	cfg.Tactics.Defense.ReconEffectMult = clampFloat(cfg.Tactics.Defense.ReconEffectMult, 0.5, 2.0)

	cfg.Tactics.Trap.MaxHPMult = clampFloat(cfg.Tactics.Trap.MaxHPMult, 0.5, 2.0)
	cfg.Tactics.Trap.MoveSpeedMult = clampFloat(cfg.Tactics.Trap.MoveSpeedMult, 0.5, 2.0)
	cfg.Tactics.Trap.ViewRadiusMult = clampFloat(cfg.Tactics.Trap.ViewRadiusMult, 0.5, 2.0)
	cfg.Tactics.Trap.HearRadiusMult = clampFloat(cfg.Tactics.Trap.HearRadiusMult, 0.5, 2.0)
	cfg.Tactics.Trap.HealEffectMult = clampFloat(cfg.Tactics.Trap.HealEffectMult, 0.5, 2.0)
	cfg.Tactics.Trap.DamageEffectMult = clampFloat(cfg.Tactics.Trap.DamageEffectMult, 0.5, 2.0)
	cfg.Tactics.Trap.ReconEffectMult = clampFloat(cfg.Tactics.Trap.ReconEffectMult, 0.5, 2.0)

	// --- combat ---
	cfg.Combat.BaseAttackDamage = clampFloat(cfg.Combat.BaseAttackDamage, 1.0, 200.0)
	cfg.Combat.AdvancedReconDurationSec = clampFloat(cfg.Combat.AdvancedReconDurationSec, 1.0, 120.0)

	// --- phases ---
	cfg.Phases.Phase1.Duration = clampInt(cfg.Phases.Phase1.Duration, 10, 3600)
	cfg.Phases.Phase2.Duration = clampInt(cfg.Phases.Phase2.Duration, 10, 3600)
	cfg.Phases.Phase2.MotorsSpawnCount = clampInt(cfg.Phases.Phase2.MotorsSpawnCount, 0, 50)
	cfg.Phases.Phase2.MotorsRequiredToOpenExit = clampInt(cfg.Phases.Phase2.MotorsRequiredToOpenExit, 0, 50)
	if cfg.Phases.Phase2.MotorsRequiredToOpenExit > cfg.Phases.Phase2.MotorsSpawnCount {
		cfg.Phases.Phase2.MotorsRequiredToOpenExit = cfg.Phases.Phase2.MotorsSpawnCount
	}
	cfg.Phases.Phase2.MotorDecipherTimeSec = clampInt(cfg.Phases.Phase2.MotorDecipherTimeSec, 1, 120)

	cfg.Phases.Phase3.ExtractionSlotsTotal = clampInt(cfg.Phases.Phase3.ExtractionSlotsTotal, 0, 8)
	cfg.Phases.Phase3.ExtractionCooldownSec = clampInt(cfg.Phases.Phase3.ExtractionCooldownSec, 0, 120)
	cfg.Phases.Phase3.GlobalPulseIntervalSec = clampInt(cfg.Phases.Phase3.GlobalPulseIntervalSec, 1, 60)
	cfg.Phases.Phase3.ViewRadiusDecayRatePerSec = clampFloat(cfg.Phases.Phase3.ViewRadiusDecayRatePerSec, 0.0, 2.0)
}
