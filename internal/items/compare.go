package items

// ItemPower computes a rough numeric power score for an item based on
// its stat modifiers, damage multiplier, and mitigation values.
// This is intentionally simple — it exists to let NPCs make reasonable
// self-gearing decisions, not to be a precise mechanical balance tool.
func ItemPower(spec ItemSpec) float64 {
	score := 0.0

	// Sum all positive stat modifier contributions (1 point per stat point)
	for _, v := range spec.StatMods {
		if v > 0 {
			score += float64(v)
		}
	}

	// Weapon damage: normalize to a comparable scale as stat mods
	if spec.DamageMultiplier > 0 {
		score += spec.DamageMultiplier * 100
	}

	// Caster weapon spell damage boost
	if spec.SpellDamageMultiplier > 0 {
		score += spec.SpellDamageMultiplier * 50
	}

	// Armor mitigations: 1 point per percentage point
	if spec.PhysicalMitigation > 0 {
		score += float64(spec.PhysicalMitigation)
	}
	if spec.MagicalMitigation > 0 {
		score += float64(spec.MagicalMitigation)
	}
	if spec.ConvictionMitigation > 0 {
		score += float64(spec.ConvictionMitigation)
	}

	return score
}

// IsUpgrade returns true if candidate is strictly better than current
// for the same equipment slot. Returns false if the types differ,
// or if candidate has no ItemId (i.e. is an empty/placeholder spec).
func IsUpgrade(current ItemSpec, candidate ItemSpec) bool {
	if candidate.ItemId == 0 {
		return false
	}
	if current.Type != candidate.Type {
		return false
	}
	return ItemPower(candidate) > ItemPower(current)
}
