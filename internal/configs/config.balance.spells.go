package configs

// validateSpells sets defaults for spell costs, spellcasting parameters,
// and enchantment fields.
func (b *Balance) validateSpells() {
	// ── SPELL COSTS ──────────────────────────────────────────────────────────
	if b.SpellConvictionCostMultiplier <= 0 {
		b.SpellConvictionCostMultiplier = 1.0
	}
	if b.SpellHealthCostMultiplier <= 0 {
		b.SpellHealthCostMultiplier = 1.0
	}
	if b.SelfCastProgressionMultiplier <= 0 {
		b.SelfCastProgressionMultiplier = 0.5
	}

	// ── SPELLCASTING ─────────────────────────────────────────────────────────
	if b.SpellConcentrationBase <= 0 {
		b.SpellConcentrationBase = 50
	}
	if b.SpellInitiationBase <= 0 {
		b.SpellInitiationBase = 60
	}
	if b.SpellInitiationWillpowerDivisor < 1 {
		b.SpellInitiationWillpowerDivisor = 4
	}
	if b.SpellInitiationSkillFactor < 1 {
		b.SpellInitiationSkillFactor = 5
	}
	if b.SpellFoldsSkillFactor < 1 {
		b.SpellFoldsSkillFactor = 25
	}
	if b.SpellDamageScale <= 0 {
		b.SpellDamageScale = 1.0
	}
	if b.SpellAttackSkillFactor < 1 {
		b.SpellAttackSkillFactor = 3
	}
	if b.SpellAvoidanceDamageMultiplier <= 0 || b.SpellAvoidanceDamageMultiplier > 1.0 {
		b.SpellAvoidanceDamageMultiplier = 0.50
	}
	if b.SpellDifficultyProgressionScale <= 0 {
		b.SpellDifficultyProgressionScale = 0.01
	}
	if b.SpellDiscoveryBaseChance <= 0 {
		b.SpellDiscoveryBaseChance = 5.0
	}
	if b.SpellDiscoveryDecayRate <= 0 {
		b.SpellDiscoveryDecayRate = 0.1
	}
	if b.SpellProficiencyCastsPerPoint < 1 {
		b.SpellProficiencyCastsPerPoint = 50
	}

	// ── ENCHANTMENTS ─────────────────────────────────────────────────────────
	if b.EnchantTierUpBaseChance <= 0 {
		b.EnchantTierUpBaseChance = 0.02
	}
	if b.EnchantTierUsesBase < 1 {
		b.EnchantTierUsesBase = 25
	}
	if b.EnchantTierUsesScale <= 0 {
		b.EnchantTierUsesScale = 2.5
	}
	if b.EnchantRemovalPenaltyRounds < 1 {
		b.EnchantRemovalPenaltyRounds = 50
	}
	if b.EnchantMaxTier < 1 {
		b.EnchantMaxTier = 4
	}
}
