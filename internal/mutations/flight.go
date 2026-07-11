package mutations

// IsFlying reports whether any owned mutation grants the "flying" flag (the
// Winged Flight transformation). Read by the combat, flee, and movement hooks
// that give a flyer its edge over the earthbound.
func IsFlying(owned map[string]int) bool {
	return HasMutationFlag(owned, "flying")
}
