package configs

// validateMisc sets defaults for miscellaneous balance fields.
func (b *Balance) validateMisc() {
	// ── REGEN RATES ──────────────────────────────────────────────────────────
	clampPct := func(v *ConfigFloat, def ConfigFloat) {
		if *v <= 0 {
			*v = def
		}
		if *v > 1.0 {
			*v = 1.0
		}
	}
	clampPct(&b.PlayerHealthRegenPct, 0.01)
	clampPct(&b.PlayerStaminaRegenPct, 0.01)
	clampPct(&b.PlayerConvictionRegenPct, 0.01)

	// ── STAMINA & CONVICTION ──────────────────────────────────────────────────
	if b.MovementBaseStaminaCost <= 0 {
		b.MovementBaseStaminaCost = 2.0
	}
	if b.MovementMaxStaminaCost <= 0 {
		b.MovementMaxStaminaCost = 20.0
	}
	if b.UnarmedAttackStaminaCost < 1 {
		b.UnarmedAttackStaminaCost = 4
	}

	// ── RESOURCE MAXIMUMS ─────────────────────────────────────────────────────
	if b.HealthBase < 0 {
		b.HealthBase = 5
	}
	if b.HealthPerStrength < 0 {
		b.HealthPerStrength = 1
	}
	if b.HealthPerVitality < 0 {
		b.HealthPerVitality = 4
	}
	if b.StaminaBase < 0 {
		b.StaminaBase = 5
	}
	if b.StaminaPerStrength < 0 {
		b.StaminaPerStrength = 1
	}
	if b.StaminaPerWillpower < 0 {
		b.StaminaPerWillpower = 1
	}
	if b.StaminaPerVitality < 0 {
		b.StaminaPerVitality = 3
	}
	if b.ConvictionBase < 0 {
		b.ConvictionBase = 5
	}
	if b.ConvictionPerWilCha < 0 {
		b.ConvictionPerWilCha = 2
	}

	// ── RESOURCE DEPLETION PENALTIES ─────────────────────────────────────────
	if b.ResourcePenaltyCurve <= 0 {
		b.ResourcePenaltyCurve = 2.0
	}
	if b.HealthPenaltyMax <= 0 || b.HealthPenaltyMax > 1.0 {
		b.HealthPenaltyMax = 0.28
	}
	if b.StaminaPenaltyMax <= 0 || b.StaminaPenaltyMax > 1.0 {
		b.StaminaPenaltyMax = 0.28
	}
	if b.ConvictionPenaltyMax <= 0 || b.ConvictionPenaltyMax > 1.0 {
		b.ConvictionPenaltyMax = 0.28
	}

	// ── CHARACTER CREATION ────────────────────────────────────────────────────
	if b.StatRollMean <= 0 {
		b.StatRollMean = 100.0
	}
	if b.StatRollStdDev <= 0 {
		b.StatRollStdDev = 15.0
	}
	if b.StatRollMin <= 0 {
		b.StatRollMin = 70.0
	}
	if b.StatRollMax <= 0 {
		b.StatRollMax = 130.0
	}
	if b.StartingHealth < 1 {
		b.StartingHealth = 10
	}

	// ── SALVAGE ──────────────────────────────────────────────────────────────
	if b.SalvageMinChance <= 0 {
		b.SalvageMinChance = 0.15
	}
	if b.SalvageMaxChance <= 0 {
		b.SalvageMaxChance = 0.85
	}
	if b.SalvageSoftCap < 1 {
		b.SalvageSoftCap = 50
	}
	if b.SalvageGoldPerRound < 1 {
		b.SalvageGoldPerRound = 10
	}
	if b.SalvageMaxRounds < 1 {
		b.SalvageMaxRounds = 5
	}

	// ── QUEST ENGINE ─────────────────────────────────────────────────────────
	if b.QuestLogLevel == "" {
		b.QuestLogLevel = "verbose"
	}
	if b.QuestChainDepthLimit < 1 {
		b.QuestChainDepthLimit = 10
	}
	if b.QuestPerformanceWarnMs < 1 {
		b.QuestPerformanceWarnMs = 50
	}

	// ── WORLD EVENTS ─────────────────────────────────────────────────────────
	if b.WorldEventBufferSize < 10 {
		b.WorldEventBufferSize = 200
	}

	// ── CHARACTER MANAGEMENT ────────────────────────────────────────────────
	if b.CharacterRenameCooldownHours < 0 {
		b.CharacterRenameCooldownHours = 0
	}

	// ── CARRY CAPACITY ──────────────────────────────────────────────────────
	if b.CarryCapacityMultiplier < 0.1 || b.CarryCapacityMultiplier > 10.0 {
		b.CarryCapacityMultiplier = 0.65
	}

	// ── MIN DEFENSE/ATTACK CHANCE ────────────────────────────────────────────
	if b.MinDefenseChance < 0 || b.MinDefenseChance > 0.50 {
		b.MinDefenseChance = 0.15
	}
	if b.MinAttackHitChance < 0 || b.MinAttackHitChance > 0.50 {
		b.MinAttackHitChance = 0.15
	}

	// ── MOVEMENT ─────────────────────────────────────────────────────────────
	// (MovementBaseStaminaCost and MovementMaxStaminaCost handled in STAMINA & CONVICTION above)

	// ── STAND ────────────────────────────────────────────────────────────────
	if b.StandStaminaCost <= 0 || b.StandStaminaCost > 1.0 {
		b.StandStaminaCost = 0.15
	}
	if b.StandMinStamina <= 0 || b.StandMinStamina > 1.0 {
		b.StandMinStamina = 0.15
	}

	// ── GLOBAL DAMAGE ────────────────────────────────────────────────────────
	if b.GlobalDamageMultiplier <= 0 {
		b.GlobalDamageMultiplier = 1.0
	}
	if b.HasteSwingMultiplier <= 0 {
		b.HasteSwingMultiplier = 1.50
	}

	// ── SKILL MULTIPLIER ─────────────────────────────────────────────────────
	if b.SkillMultiplierBase <= 0 {
		b.SkillMultiplierBase = 1.0
	}
	if b.SkillMultiplierMax <= 0 {
		b.SkillMultiplierMax = 3.0
	}

	// ── THIRD-PARTY GRAPPLE ──────────────────────────────────────────────────
	if b.ThirdPartyGrapplePenalty <= 0 || b.ThirdPartyGrapplePenalty > 1.0 {
		b.ThirdPartyGrapplePenalty = 0.70
	}

	// ── UNARMED ──────────────────────────────────────────────────────────────
	if b.UnarmedBaseDamage <= 0 {
		b.UnarmedBaseDamage = 2.0
	}
	if b.UnarmedStrengthDivisor <= 0 {
		b.UnarmedStrengthDivisor = 25.0
	}
	if b.UnarmedSkillDivisor <= 0 {
		b.UnarmedSkillDivisor = 10.0
	}
	if b.UnarmedBaseVariance <= 0 {
		b.UnarmedBaseVariance = 3.0
	}
	if b.UnarmedDamageMultiplier <= 0 {
		b.UnarmedDamageMultiplier = 0.30
	}
	if b.UnarmedSpeedMultiplier <= 0 {
		b.UnarmedSpeedMultiplier = 1.8
	}

	// ── SPECIAL MOVE DAMAGE % ────────────────────────────────────────────────
	if b.BashDamagePercent <= 0 || b.BashDamagePercent > 1.0 {
		b.BashDamagePercent = 0.50
	}
	if b.BashKnockdownChance < 0 || b.BashKnockdownChance > 100 {
		b.BashKnockdownChance = 40
	}
	if b.TripDamagePercent <= 0 || b.TripDamagePercent > 1.0 {
		b.TripDamagePercent = 0.25
	}
	if b.TripKnockdownChance < 0 || b.TripKnockdownChance > 100 {
		b.TripKnockdownChance = 60
	}
	if b.KickDamagePercent <= 0 || b.KickDamagePercent > 2.0 {
		b.KickDamagePercent = 0.80
	}
	if b.KickKnockdownChance < 0 || b.KickKnockdownChance > 100 {
		b.KickKnockdownChance = 35
	}
	if b.StompDamagePercent <= 0 || b.StompDamagePercent > 3.0 {
		b.StompDamagePercent = 1.20
	}
	if b.KneeDamagePercent <= 0 || b.KneeDamagePercent > 2.0 {
		b.KneeDamagePercent = 1.00
	}

	// ── SURPRISE ATTACK ──────────────────────────────────────────────────────
	if b.SurpriseAttackOffhandPenalty < 0 || b.SurpriseAttackOffhandPenalty > 1.0 {
		b.SurpriseAttackOffhandPenalty = 0.10
	}
	if b.SurpriseAttackExtraArm1Penalty < 0 || b.SurpriseAttackExtraArm1Penalty > 1.0 {
		b.SurpriseAttackExtraArm1Penalty = 0.25
	}
	if b.SurpriseAttackExtraArm2Penalty < 0 || b.SurpriseAttackExtraArm2Penalty > 1.0 {
		b.SurpriseAttackExtraArm2Penalty = 0.40
	}
	if b.SurpriseAttackExtraArm3Penalty < 0 || b.SurpriseAttackExtraArm3Penalty > 1.0 {
		b.SurpriseAttackExtraArm3Penalty = 0.55
	}
	if b.SurpriseAttackExtraArm4Penalty < 0 || b.SurpriseAttackExtraArm4Penalty > 1.0 {
		b.SurpriseAttackExtraArm4Penalty = 0.70
	}

	// ── COUP DE GRACE ────────────────────────────────────────────────────────
	if b.CoupDeGraceRounds < 0 {
		b.CoupDeGraceRounds = 1
	}

	// ── CLINCH ───────────────────────────────────────────────────────────────
	if b.ClinchDodgePenalty <= 0 || b.ClinchDodgePenalty > 1.0 {
		b.ClinchDodgePenalty = 0.80
	}
	if b.ClinchParryPenalty <= 0 || b.ClinchParryPenalty > 1.0 {
		b.ClinchParryPenalty = 0.83
	}
	if b.ClinchBlockPenalty <= 0 || b.ClinchBlockPenalty > 1.0 {
		b.ClinchBlockPenalty = 0.85
	}

	// ── GROUNDED ─────────────────────────────────────────────────────────────
	if b.GroundedDodgePenalty <= 0 || b.GroundedDodgePenalty > 1.0 {
		b.GroundedDodgePenalty = 0.75
	}
	if b.GroundedParryPenalty <= 0 || b.GroundedParryPenalty > 1.0 {
		b.GroundedParryPenalty = 0.77
	}
	if b.GroundedBlockPenalty <= 0 || b.GroundedBlockPenalty > 1.0 {
		b.GroundedBlockPenalty = 0.80
	}

	// ── LOOT ──────────────────────────────────────────────────────────────────
	if b.LootBudgetScalar <= 0 {
		b.LootBudgetScalar = 7.0
	}

	// ── INSTANCES ────────────────────────────────────────────────────────────
	if b.InstanceStatPoolCap < 1 {
		b.InstanceStatPoolCap = 50000
	}

	// ── SKILL WEIGHT ─────────────────────────────────────────────────────────
	if b.SkillWeight <= 0 {
		b.SkillWeight = 2.0
	}

	// ── CARAVAN SYSTEM ───────────────────────────────────────────────────────
	if len(b.CaravanServedZones) == 0 {
		b.CaravanServedZones = []string{"Stillwater", "Thornwall City"}
	}
	if b.CaravanDepotDwellRounds == 0 {
		b.CaravanDepotDwellRounds = 720
	}
	if b.FernwayPickupDwellRounds == 0 {
		b.FernwayPickupDwellRounds = 6
	}

	// ── FORAGER SYSTEM (Stage 3.1) ───────────────────────────────────────────
	if b.ForagerForageDwellRounds == 0 {
		b.ForagerForageDwellRounds = 8
	}
	if b.ForagerCarryThresholdPct == 0 {
		b.ForagerCarryThresholdPct = 0.75
	}
	if b.ForagerHPRecallThresholdPct == 0 {
		b.ForagerHPRecallThresholdPct = 0.50
	}
	if b.ForagerHealPotionThresholdPct == 0 {
		b.ForagerHealPotionThresholdPct = 0.75
	}
	if b.ForagerWaitTimeoutRounds == 0 {
		b.ForagerWaitTimeoutRounds = 150
	}
}
