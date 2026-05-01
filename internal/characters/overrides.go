package characters

// ApplyMobOverrides applies the Stage 3.4 spawn-time overrides for
// special mobs (wagons, statues, etc.). Each parameter is gated by
// > 0; zero values leave the existing stat-derived value alone.
//
// Called from the mob spawn path AFTER Validate() has computed the
// stat-derived defaults but BEFORE Health = HealthMax.Value etc. so
// the new max propagates into the current pool.
func ApplyMobOverrides(c *Character, healthMax, staminaMax int, carryCapacity float64) {
	if c == nil {
		return
	}
	if healthMax > 0 {
		c.HealthMax.Value = healthMax
		c.Health = healthMax
	}
	if staminaMax > 0 {
		c.StaminaMax.Value = staminaMax
		c.Stamina = staminaMax
	}
	if carryCapacity > 0 {
		c.carryCapacityOverride = carryCapacity
	}
}
