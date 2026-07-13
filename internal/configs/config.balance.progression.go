package configs

// validateProgression sets defaults for skill/stat progression curves,
// multipliers, regen progression, and mutation progression fields.
func (b *Balance) validateProgression() {
	// ── PROGRESSION ───────────────────────────────────────────────────────────
	if b.SkillSoftCap < 1 {
		b.SkillSoftCap = 50
	}
	if b.StatSoftCap < 1 {
		b.StatSoftCap = 150
	}
	if b.BaseProgressionChance <= 0 || b.BaseProgressionChance > 1.0 {
		b.BaseProgressionChance = 0.30
	}
	if b.ProgressionDecayBelowCap <= 0 {
		b.ProgressionDecayBelowCap = 3.0
	}
	if b.ProgressionDecayAboveCap <= 0 {
		b.ProgressionDecayAboveCap = 2.0
	}
	if b.UsesPerRank < 1 {
		b.UsesPerRank = 25
	}
	if b.SkillWeight <= 0 {
		b.SkillWeight = 2.0
	}

	// ── PROGRESSION MULTIPLIERS ──────────────────────────────────────────────
	if b.StatProgressionMultipliers == nil {
		b.StatProgressionMultipliers = map[string]float64{}
	}
	for k, v := range b.StatProgressionMultipliers {
		if v <= 0 {
			delete(b.StatProgressionMultipliers, k)
		}
	}
	if b.SkillProgressionMultipliers == nil {
		b.SkillProgressionMultipliers = map[string]float64{}
	}
	for k, v := range b.SkillProgressionMultipliers {
		if v <= 0 {
			delete(b.SkillProgressionMultipliers, k)
		}
	}

	// ── REGEN-BASED STAT PROGRESSION ─────────────────────────────────────────
	if b.RegenProgressionBase <= 0 {
		b.RegenProgressionBase = 0.005
	}
	if b.RegenProgressionBase > 1.0 {
		b.RegenProgressionBase = 1.0
	}
	if b.RegenProgressionCurve <= 0 {
		b.RegenProgressionCurve = 3.0
	}

	// ── MUTATIONS ─────────────────────────────────────────────────────────────
	if b.MutationBaseProgress <= 0 {
		b.MutationBaseProgress = 50.0
	}
	if b.MutationProgressScale <= 0 {
		b.MutationProgressScale = 1.5
	}
	if b.MutationMaxCount < 1 {
		b.MutationMaxCount = 5
	}
	if b.MutationMaxLevel < 1 {
		b.MutationMaxLevel = 3
	}
	if b.MutationDeepenChance <= 0 || b.MutationDeepenChance > 1.0 {
		b.MutationDeepenChance = 0.70
	}
	if b.MutationProgressGainPerRound <= 0 {
		b.MutationProgressGainPerRound = 1.0
	}
	if b.MutationAffinityPerSkillUse <= 0 {
		b.MutationAffinityPerSkillUse = 1.0
	}
	if b.MutationAffinityPerCombatEvent <= 0 {
		b.MutationAffinityPerCombatEvent = 1.0
	}
	if b.StatProgressionRate <= 0 {
		b.StatProgressionRate = 1.0
	}
	if b.MutationAffinityPerRarity <= 0 {
		b.MutationAffinityPerRarity = 6.0
	}
	if b.MutationAffinityDecay <= 0 || b.MutationAffinityDecay > 1.0 {
		b.MutationAffinityDecay = 0.98
	}
	if b.MutationBodyConvictionDecayMax <= 0 || b.MutationBodyConvictionDecayMax > 1.0 {
		b.MutationBodyConvictionDecayMax = 0.9
	}
	if b.MutationBeliefGearDecayMax <= 0 || b.MutationBeliefGearDecayMax > 1.0 {
		b.MutationBeliefGearDecayMax = 0.9
	}
	if b.MutationPoleDecayRef <= 0 {
		b.MutationPoleDecayRef = 60.0
	}
	if b.MutationLevel2Multiplier <= 0 {
		b.MutationLevel2Multiplier = 1.5
	}
	if b.MutationLevel3Multiplier <= 0 {
		b.MutationLevel3Multiplier = 2.0
	}
	if b.MutationLevel4Multiplier <= 0 {
		b.MutationLevel4Multiplier = 2.5
	}

	// ── STAT SOFT CAP ─────────────────────────────────────────────────────────
	if b.StatSoftCapThreshold < 100 {
		b.StatSoftCapThreshold = 105
	}
	if b.StatSoftCapMultiplier <= 0 {
		b.StatSoftCapMultiplier = 2.0
	}
}
