package combat

// GetDamageDescription converts numeric damage into a descriptive phrase based on
// the damage as a percentage of the target's maximum HP.
//
// This creates immersive combat text instead of showing raw numbers:
//   - "Your slash causes moderate injuries to the guard"
// instead of:
//   - "Your slash causes 12 damage to the guard"
//
// The description is scaled relative to the target's max HP, so the same
// numeric damage feels different against weak vs. strong opponents.
func GetDamageDescription(damageAmount int, targetMaxHP int) string {
	// Fallback for edge cases (no target HP data)
	if targetMaxHP <= 0 {
		return "moderate injuries"
	}

	// Calculate damage as percentage of target's max HP
	pct := float64(damageAmount) / float64(targetMaxHP) * 100

	// Return descriptive text based on percentage thresholds
	// These phrases work grammatically with existing YAML messages:
	//   "causing {damage}" → "causing moderate injuries"
	//   "dealing {damage}" → "dealing serious wounds"
	//   "inflicting {damage}" → "inflicting critical injuries"
	switch {
	case pct < 5:
		return "negligible damage"
	case pct < 15:
		return "light wounds"
	case pct < 30:
		return "moderate injuries"
	case pct < 50:
		return "serious wounds"
	case pct < 75:
		return "critical injuries"
	default:
		return "devastating wounds"
	}
}

// GetHealDescription converts a numeric healing amount into a descriptive phrase,
// scaled relative to the target's maximum HP. Mirrors GetDamageDescription so that
// neither function leaks raw numbers into player-facing messages.
func GetHealDescription(healAmount int, targetMaxHP int) string {
	if targetMaxHP <= 0 {
		return "light mending"
	}

	pct := float64(healAmount) / float64(targetMaxHP) * 100

	switch {
	case pct < 5:
		return "negligible mending"
	case pct < 15:
		return "light mending"
	case pct < 30:
		return "moderate restoration"
	case pct < 50:
		return "substantial healing"
	case pct < 75:
		return "significant healing"
	default:
		return "extraordinary healing"
	}
}
