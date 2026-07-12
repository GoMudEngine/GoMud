package mutations

// ironhide.go — effect readers for the Ironhide (retaliation-tank) cluster.

// GetReflectDamage returns the net percent of incoming damage the owner lashes
// back at melee attackers (Reflect Skin / Living Carapace). It feeds the combat
// return-damage path alongside the species/equipment return_damage sources.
func GetReflectDamage(owned map[string]int) float64 {
	return sumEffects(owned, "reflect_damage", "")
}

// IsControlImmune reports whether an owned mutation grants the "control-immune"
// flag — immovable: cannot be knocked down (trip/bash) or grappled. Granted by
// Ironhide's Living Carapace and Colossus's Ossified Frame.
func IsControlImmune(owned map[string]int) bool {
	return HasMutationFlag(owned, "control-immune")
}
