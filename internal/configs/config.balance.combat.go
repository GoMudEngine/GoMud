package configs

// validateCombat sets defaults for combat rolls, defense, prone/grapple,
// special moves, skullduggery, darkness, damage channels, mitigation caps,
// and toxicity fields.
func (b *Balance) validateCombat() {
	// ── ROLL SPREAD ──────────────────────────────────────────────────────────
	if b.RollSpread < 0.05 || b.RollSpread > 0.50 {
		b.RollSpread = 0.15
	}

	// ── COMBAT: DEFENSE COSTS ────────────────────────────────────────────────
	if b.DodgeMultiplier <= 0 {
		b.DodgeMultiplier = 0.9
	}
	if b.ParryMultiplier <= 0 {
		b.ParryMultiplier = 0.9
	}
	if b.BlockMultiplier <= 0 {
		b.BlockMultiplier = 0.9
	}

	// ── COMBAT: DEFENSE EFFECTIVENESS ────────────────────────────────────────
	if b.DodgeEffectiveness <= 0 {
		b.DodgeEffectiveness = 1.0
	}
	if b.ParryEffectiveness <= 0 {
		b.ParryEffectiveness = 1.0
	}
	if b.BlockEffectiveness <= 0 {
		b.BlockEffectiveness = 1.0
	}

	// ── COMBAT: PRONE & GRAPPLE ──────────────────────────────────────────────
	if b.ProneAttackMultiplier <= 0 {
		b.ProneAttackMultiplier = 0.80
	}
	if b.ProneDodgePenalty <= 0 || b.ProneDodgePenalty > 1.0 {
		b.ProneDodgePenalty = 0.70
	}
	if b.ProneParryPenalty <= 0 || b.ProneParryPenalty > 1.0 {
		b.ProneParryPenalty = 0.80
	}
	if b.ProneBlockPenalty <= 0 || b.ProneBlockPenalty > 1.0 {
		b.ProneBlockPenalty = 0.90
	}
	if b.ProneDamagePenalty <= 0 || b.ProneDamagePenalty > 1.0 {
		b.ProneDamagePenalty = 0.80
	}
	if b.ProneVulnerabilityMultiplier <= 0 {
		b.ProneVulnerabilityMultiplier = 1.15
	}

	// ── SPECIAL MOVES ────────────────────────────────────────────────────────
	if b.SpecialMoveCooldown < 1 {
		b.SpecialMoveCooldown = 5
	}
	if b.TauntHoldRounds < 1 {
		b.TauntHoldRounds = 4
	}

	// ── MUTATION: WINGED FLIGHT ──
	if b.FlightOpposedEdge < 1 {
		b.FlightOpposedEdge = 25
	}
	if b.FlightMoveStaminaMult <= 0 {
		b.FlightMoveStaminaMult = 0.5
	}
	if b.FlightFleeStaminaMult <= 0 {
		b.FlightFleeStaminaMult = 0.5
	}

	// ── RANGED ───────────────────────────────────────────────────────────────
	if b.RangedShotScale <= 0 {
		b.RangedShotScale = 1.0
	}
	// <= 0 mirrors SpecialMoveCooldown/GrappleStaminaCostPerRound convention:
	// treat 0 as absent (a shield granting zero bonus makes no design sense).
	if b.RangedShieldDefenseBonus <= 0 {
		b.RangedShieldDefenseBonus = 15
	}

	// ── SKULLDUGGERY ─────────────────────────────────────────────────────────
	if b.SneakFailCooldown < 0 {
		b.SneakFailCooldown = 3
	}
	if b.StealSkillMultiplier <= 0 {
		b.StealSkillMultiplier = 1.0
	}
	if b.StealHiddenBonus < 0 {
		b.StealHiddenBonus = 25
	}
	if b.StealCooldown < 0 {
		b.StealCooldown = 60
	}
	if b.ShadowCooldown < 0 {
		b.ShadowCooldown = 5
	}
	if b.HiddenMoveStaminaMultiplier <= 0 {
		b.HiddenMoveStaminaMultiplier = 3.0
	}
	if b.SneakModEmitsLightDarkRoom <= 0 {
		b.SneakModEmitsLightDarkRoom = 0.5
	}
	if b.SneakModEmitsLightLitRoom <= 0 {
		b.SneakModEmitsLightLitRoom = 0.85
	}
	if b.SneakModNoLightLitRoom <= 0 {
		b.SneakModNoLightLitRoom = 0.9
	}

	// ── REACH UTILITY CURVE (chunk 4c) ───────────────────────────────────────
	if b.ReachStandingGrappleRadius <= 0 {
		b.ReachStandingGrappleRadius = 0.5
	}
	if b.ReachGroundGrappleRadius <= 0 {
		b.ReachGroundGrappleRadius = 0.3
	}
	if b.ReachUtilityFloor <= 0 {
		b.ReachUtilityFloor = 0.15
	}

	// ── CHUNK 4D: SUBMISSION TICK ────────────────────────────────────────────
	if b.SubmissionAttemptAlpha <= 0 {
		b.SubmissionAttemptAlpha = 1.0
	}
	if b.SubmissionAttemptCritZ <= 0 {
		b.SubmissionAttemptCritZ = 2.0
	}
	if b.SubSkillWeight <= 0 {
		b.SubSkillWeight = 1.5
	}
	if b.SubBadZThreshold == 0 {
		b.SubBadZThreshold = -1.0
	}
	if b.SubCritZThreshold <= 0 {
		b.SubCritZThreshold = 2.0
	}
	if b.SubGoldLossFraction < 0 || b.SubGoldLossFraction > 1.0 {
		b.SubGoldLossFraction = 0.20
	}
	if b.BrokenLimbBuffDuration <= 0 {
		b.BrokenLimbBuffDuration = 900
	}

	// ── CHUNK 4E: THIRD-PARTY INTERFERENCE ──────────────────────────────────
	// ControlDegradeOnOutsideHit is a ConfigBool — no zero-check needed.
	// SubInterruptDamageThresholdPct: valid range [0.0, 1.0]. 0.0 means
	// crit-only path; negative values are nonsensical, clamp to 0.
	if b.SubInterruptDamageThresholdPct < 0 {
		b.SubInterruptDamageThresholdPct = 0
	}
	if b.SubInterruptDamageThresholdPct > 1.0 {
		b.SubInterruptDamageThresholdPct = 0.10
	}

	// ── GRAPPLE THRESHOLDS ───────────────────────────────────────────────────
	if b.GrappleStaminaLowThreshold <= 0 || b.GrappleStaminaLowThreshold >= 1.0 {
		b.GrappleStaminaLowThreshold = 0.25
	}

	// ── GRAPPLE CONTROL AXIS (chunk 4b) ──────────────────────────────────────
	if b.GrappleStaminaPenaltyMax <= 0 || b.GrappleStaminaPenaltyMax > 1.0 {
		b.GrappleStaminaPenaltyMax = 0.60
	}
	if b.GrappleStaminaPenaltyCurve <= 0 {
		b.GrappleStaminaPenaltyCurve = 1.5
	}
	if b.GrappleEncumbrancePenaltyMax <= 0 || b.GrappleEncumbrancePenaltyMax > 1.0 {
		b.GrappleEncumbrancePenaltyMax = 0.80
	}
	if b.GrappleEncumbrancePenaltyCurve <= 0 {
		b.GrappleEncumbrancePenaltyCurve = 1.5
	}
	if b.GrappleStaminaCostPerRound <= 0 {
		b.GrappleStaminaCostPerRound = 5
	}
	if b.GrappleControllerCostMultiplier <= 0 {
		b.GrappleControllerCostMultiplier = 1.0
	}
	if b.GrappleControlledCostMultiplier <= 0 {
		b.GrappleControlledCostMultiplier = 2.0
	}
	if b.PositionConsistencyCheckRounds <= 0 {
		b.PositionConsistencyCheckRounds = 10
	}

	// ── DARKNESS ─────────────────────────────────────────────────────────────
	if b.DarknessCombatPenalty <= 0 || b.DarknessCombatPenalty > 1.0 {
		b.DarknessCombatPenalty = 0.80
	}

	// ── MESSAGES ─────────────────────────────────────────────────────────────
	// ConsistentAttackMessages is a ConfigBool — no zero-check needed

	// ── DAMAGE ───────────────────────────────────────────────────────────────
	if b.MeleeDamageScale <= 0 {
		b.MeleeDamageScale = 0.30
	}
	if b.RhetoricDamageScale <= 0 {
		b.RhetoricDamageScale = 1.0
	}
	if b.RhetoricAvoidanceDamageMultiplier <= 0 || b.RhetoricAvoidanceDamageMultiplier > 1.0 {
		b.RhetoricAvoidanceDamageMultiplier = 0.50
	}
	if b.PhysicalMitigationCap <= 0 || b.PhysicalMitigationCap > 1.0 {
		b.PhysicalMitigationCap = 0.75
	}
	if b.MagicalMitigationCap <= 0 || b.MagicalMitigationCap > 1.0 {
		b.MagicalMitigationCap = 0.75
	}
	if b.ConvictionMitigationCap <= 0 || b.ConvictionMitigationCap > 1.0 {
		b.ConvictionMitigationCap = 0.75
	}

	// ── TOXICITY ────────────────────────────────────────────────────────────
	if b.ToxicityDecayPerTick <= 0 {
		b.ToxicityDecayPerTick = 1.0
	}
	if b.ToxicityBaseMax <= 0 {
		b.ToxicityBaseMax = 100
	}
	if b.ToxicityVitalityScale <= 0 {
		b.ToxicityVitalityScale = 5
	}
	if b.ToxicitySicknessDamagePct <= 0 {
		b.ToxicitySicknessDamagePct = 0.02
	}
	if b.ToxicityHighDecaySlowMult <= 0 {
		b.ToxicityHighDecaySlowMult = 0.5
	}

	// ── BLOOM ───────────────────────────────────────────────────────────────
	if b.BloomAddictionPerDose <= 0 {
		b.BloomAddictionPerDose = 1
	}
	if b.BloomAddictionDecayRounds <= 0 {
		b.BloomAddictionDecayRounds = 300
	}
	if b.BloomWithdrawalOnsetRounds <= 0 {
		b.BloomWithdrawalOnsetRounds = 60
	}
	if b.BloomMutationAdvanceChance <= 0 {
		b.BloomMutationAdvanceChance = 0.50
	}
	if b.BloomNewMutationChance <= 0 {
		b.BloomNewMutationChance = 0.10
	}
	if b.BloomCommunionRounds <= 0 {
		b.BloomCommunionRounds = 30
	}
	if b.BloomCrashRoundsMult <= 0 {
		b.BloomCrashRoundsMult = 2.5
	}
}
