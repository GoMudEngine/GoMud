package mutations

// chrysifier.go — effect-type readers for the Chrysifier (crafter) cluster.
// Numeric effects sum through sumEffects (rank-scaled); the two apex-adjacent
// abilities are flags.

// GetForageYieldMult returns the net forage-yield multiplier (Provident Hands).
// Apply as: score *= (1 + GetForageYieldMult(m)).
func GetForageYieldMult(owned map[string]int) float64 {
	return sumEffects(owned, "forage_yield_multiplier", "")
}

// GetSalvageYieldBonus returns the net salvage-chance bonus fraction (Provident Hands).
// Apply as: chance = min(1, chance*(1+GetSalvageYieldBonus(m))).
func GetSalvageYieldBonus(owned map[string]int) float64 {
	return sumEffects(owned, "salvage_yield_bonus", "")
}

// GetCraftMaterialDiscount returns the net per-ingredient refund chance (Provident Hands).
func GetCraftMaterialDiscount(owned map[string]int) float64 {
	return sumEffects(owned, "craft_material_discount", "")
}

// GetCraftQualityBonus returns the net crafted-quality bonus fraction (Faithwrought).
// Apply as: craftSkill += round(craftSkill * GetCraftQualityBonus(m)).
func GetCraftQualityBonus(owned map[string]int) float64 {
	return sumEffects(owned, "craft_quality_bonus", "")
}

// GetCarryCapacityMultiplier returns the net carry-capacity multiplier (Faithwrought).
// Apply as: capacity *= (1 + GetCarryCapacityMultiplier(m)).
func GetCarryCapacityMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "carry_capacity_multiplier", "")
}

// HasPortableWorkshop reports whether an owned mutation grants the Walking
// Chrysalis "portable-workshop" flag (craft anywhere, no station/tools).
func HasPortableWorkshop(owned map[string]int) bool {
	return HasMutationFlag(owned, "portable-workshop")
}

// HasHomunculus reports whether an owned mutation grants the Homunculus apex flag.
func HasHomunculus(owned map[string]int) bool {
	return HasMutationFlag(owned, "homunculus")
}
