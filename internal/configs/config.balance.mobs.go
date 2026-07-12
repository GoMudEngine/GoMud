package configs

// validateMobs sets defaults for mob AI, regen, damage, progression,
// mutations, pack scaling, gossip, moon phases, and manifestation fields.
func (b *Balance) validateMobs() {
	// ── MOB AI ───────────────────────────────────────────────────────────────
	if b.CombatMemoryDuration < 1 {
		b.CombatMemoryDuration = 300
	}
	if !bool(b.MobAIEnabled) {
		b.MobAIEnabled = true
	}
	if b.MobReactionDelayMin <= 0 {
		b.MobReactionDelayMin = 0.25
	}
	if b.MobReactionDelayMax <= 0 {
		b.MobReactionDelayMax = 2.0
	}
	if b.MobBTreeReactionBase <= 0 {
		b.MobBTreeReactionBase = 2.0
	}
	if b.MobBTreeReactionPerceptionScale < 1 {
		b.MobBTreeReactionPerceptionScale = 100
	}

	// ── MOB SCHEDULES ────────────────────────────────────────────────────────
	if b.ScheduleMaxPathRetries < 1 {
		b.ScheduleMaxPathRetries = 20
	}

	// ── MOB SLEEP & WAKE ─────────────────────────────────────────────────────
	if b.SleepRegenMultiplier <= 0 {
		b.SleepRegenMultiplier = 5.0
	}
	if b.ScheduleWakeGraceRounds < 1 {
		b.ScheduleWakeGraceRounds = 50
	}

	// ── NPC-NPC IDLE CONVERSATIONS ───────────────────────────────────────────
	if b.ConversationBaseChancePct <= 0 {
		b.ConversationBaseChancePct = 1.0
	}
	if b.ConversationPlayerArrivalBoostPct <= 0 {
		b.ConversationPlayerArrivalBoostPct = 25
	}
	if b.ConversationCooldownRounds < 1 {
		b.ConversationCooldownRounds = 50
	}

	// ── MOB REGEN ────────────────────────────────────────────────────────────
	clampPct := func(v *ConfigFloat, def ConfigFloat) {
		if *v <= 0 {
			*v = def
		}
		if *v > 1.0 {
			*v = 1.0
		}
	}
	clampPct(&b.MobHealthRegenPct, 0.01)
	clampPct(&b.MobStaminaRegenPct, 0.02)
	clampPct(&b.MobConvictionRegenPct, 0.02)

	// ── MOB DAMAGE ───────────────────────────────────────────────────────────
	if b.MobDamageMultiplier <= 0 {
		b.MobDamageMultiplier = 1.0
	}

	// ── MOB PROGRESSION ──────────────────────────────────────────────────────
	if b.MobProgressionRate <= 0 || b.MobProgressionRate > 1.0 {
		b.MobProgressionRate = 0.5
	}
	if b.MobStatCap < 1 {
		b.MobStatCap = 200
	}
	if b.MobSkillCap < 1 {
		b.MobSkillCap = 3
	}
	if b.MobSaveIntervalRounds < 1 {
		b.MobSaveIntervalRounds = 100
	}
	if b.MobInstanceMaxAgeDays < 1 {
		b.MobInstanceMaxAgeDays = 7
	}

	// ── MOB MUTATIONS ────────────────────────────────────────────────────────
	if b.MobMutationRate <= 0 || b.MobMutationRate > 1.0 {
		b.MobMutationRate = 0.3
	}

	// ── PACK SCALING ─────────────────────────────────────────────────────────
	if b.PackSurvivalRounds < 1 {
		b.PackSurvivalRounds = 10
	}
	if b.PackBonusTrainingPts < 1 {
		b.PackBonusTrainingPts = 1
	}
	if b.PackMaxBonus < 1 {
		b.PackMaxBonus = 5
	}
	if b.PackMaxSize == 0 {
		b.PackMaxSize = -1
	}
	if b.PackScatterRounds < 0 {
		b.PackScatterRounds = 2
	}

	// ── PACK ROAMING ────────────────────────────────────────────────────────
	// (PackMaxSize and PackScatterRounds covered in PACK SCALING above)

	// ── GOSSIP ───────────────────────────────────────────────────────────────
	if b.GossipIntervalRounds < 20 {
		b.GossipIntervalRounds = 75
	}

	// ── MOON PHASES ───────────────────────────────────────────────────────────
	if b.MoonStatModMax <= 0 {
		b.MoonStatModMax = 0.05
	}

	// ── MANIFESTATION ────────────────────────────────────────────────────────
	if b.ManifestStatScaleChaFactor < 1 {
		b.ManifestStatScaleChaFactor = 150
	}
	if b.ManifestStatScaleSkillFactor <= 0 {
		b.ManifestStatScaleSkillFactor = 0.02
	}

	// ── COMPANION CONVICTION ECONOMY ──────────────────────────────────────────
	if b.CompanionReserveSkillPct <= 0 {
		b.CompanionReserveSkillPct = 0.01
	}
	if b.CompanionReserveSkillCap <= 0 {
		b.CompanionReserveSkillCap = 0.55
	}
	if b.CompanionReserveMutPctPerRank <= 0 {
		b.CompanionReserveMutPctPerRank = 0.06
	}
	if b.CompanionReserveMutCap <= 0 {
		b.CompanionReserveMutCap = 0.24
	}
	if b.CompanionReserveTotalCap <= 0 {
		b.CompanionReserveTotalCap = 0.79
	}
	if b.CompanionSoftCap < 1 {
		b.CompanionSoftCap = 5
	}
	if b.CompanionSoftCapApex < 1 {
		b.CompanionSoftCapApex = 7
	}
	if b.CompanionReserveDefault < 1 {
		b.CompanionReserveDefault = 350
	}
	// CompanionCastingFloorPct intentionally defaults to 0.0 (costs self-limit).
	if b.HomunculusCraftScale <= 0 {
		b.HomunculusCraftScale = 4.0
	}
	if b.HomunculusConvictionReserve < 1 {
		b.HomunculusConvictionReserve = 1000
	}

	// ── GOAL SELECTION ───────────────────────────────────────────────────────
	if b.GoalSelectSwitchMargin <= 0 {
		b.GoalSelectSwitchMargin = 5.0
	}
	if b.GoalSelectMinHoldRounds < 1 {
		b.GoalSelectMinHoldRounds = 100
	}
	if !bool(b.GoalSelectTickEnabled) {
		// Default true: most installs want the tick path on.
		b.GoalSelectTickEnabled = true
	}
	if b.GoalPruneIntervalRounds < 1 {
		b.GoalPruneIntervalRounds = 50
	}
	if b.GoalAbandonDormantRounds < 1 {
		b.GoalAbandonDormantRounds = 600
	}
	if b.GuardWarnGraceRounds < 1 {
		b.GuardWarnGraceRounds = 50
	}
	if b.JusticeCrimeLookbackRounds < 1 {
		b.JusticeCrimeLookbackRounds = 1000
	}
	if b.JusticeBountyExpiryRounds < 1 {
		b.JusticeBountyExpiryRounds = 5000
	}
	if b.JusticeBountyMurderMult < 1 {
		b.JusticeBountyMurderMult = 2.0
	}
	if b.JusticeBountyRepMultMax < 1 {
		b.JusticeBountyRepMultMax = 2.0
	}
	if b.ArrestResistGraceRounds < 1 {
		b.ArrestResistGraceRounds = 3
	}
	if b.JusticeFineDecayPerRound < 1 {
		b.JusticeFineDecayPerRound = 5
	}
	if b.JusticeArrestRepReset == 0 {
		b.JusticeArrestRepReset = -10
	}
}
