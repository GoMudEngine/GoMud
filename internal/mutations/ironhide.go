package mutations

// ironhide.go — effect readers for the Ironhide (retaliation-tank) cluster.

// GetReflectDamage returns the net percent of incoming damage the owner lashes
// back at melee attackers (Reflect Skin / Living Carapace). It feeds the combat
// return-damage path alongside the species/equipment return_damage sources.
func GetReflectDamage(owned map[string]int) float64 {
	return sumEffects(owned, "reflect_damage", "")
}
