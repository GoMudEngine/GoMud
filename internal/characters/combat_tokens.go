package characters

// combat_tokens.go - Stage 9.4: Contextual Combat Token Calculations
//
// This file provides methods for calculating dynamic combat tokens used in
// combat messaging. These tokens reflect the current state of combat in
// real-time, making combat messages more contextual and immersive.

// CalculateStanceString returns the character's combat stance based on their
// recent attack/defense ratio. This reflects their overall combat approach.
//
// Returns:
//   - "defensive" - more defenses than attacks (ratio > 1.5)
//   - "balanced" - roughly equal attacks and defenses (ratio 1.0-1.5)
//   - "reckless" - moderately aggressive (ratio 0.66-1.0)
//   - "aggressive" - primarily offensive (ratio < 0.66)
func (c *Character) CalculateStanceString() string {
	if c.AttacksThisRound == 0 && c.DefensesThisRound == 0 {
		return "balanced"
	}
	if c.AttacksThisRound == 0 {
		return "defensive"
	}

	ratio := float64(c.DefensesThisRound) / float64(c.AttacksThisRound)
	switch {
	case ratio > 1.5:
		return "defensive"
	case ratio >= 1.0:
		return "balanced"
	case ratio >= 0.66:
		return "reckless"
	default:
		return "aggressive"
	}
}

// CalculatePositionString converts the CombatPosition enum to a human-readable
// string suitable for use in combat messages.
//
// Returns:
//   - "standing" - PositionStanding
//   - "prone" - PositionProne
//   - "locked in close combat" - PositionClinched
//   - "grappling on the ground" - PositionGrounded
func (c *Character) CalculatePositionString() string {
	switch c.CombatPosition {
	case PositionStanding:
		return "standing"
	case PositionProne:
		return "prone"
	case PositionClinched:
		return "locked in close combat"
	case PositionGrounded:
		return "grappling on the ground"
	default:
		return "standing"
	}
}

// CalculateMomentumString returns the character's combat momentum based on
// their recent hit/miss streak. This reflects the current flow of combat.
//
// Returns:
//   - "on the offensive" - 3+ consecutive hits
//   - "on the defensive" - 2+ consecutive misses
//   - "in control" - has landed at least one recent hit
//   - "pressured" - default state or mixed results
func (c *Character) CalculateMomentumString() string {
	if c.ConsecutiveHits >= 3 {
		return "on the offensive"
	}
	if c.ConsecutiveMisses >= 2 {
		return "on the defensive"
	}
	if c.ConsecutiveHits > 0 {
		return "in control"
	}
	return "pressured"
}

// IncrementAttackCount tracks an attack action for stance calculation.
// Call this whenever a character initiates an attack.
func (c *Character) IncrementAttackCount() {
	c.AttacksThisRound++
}

// IncrementDefenseCount tracks a defensive action for stance calculation.
// Call this whenever a character attempts a defense (dodge, parry, block).
func (c *Character) IncrementDefenseCount() {
	c.DefensesThisRound++
}

// UpdateMomentum updates the consecutive hit/miss tracking based on the
// outcome of an attack. This maintains the momentum state.
//
// Parameters:
//   - hit: true if the attack landed, false if it missed
func (c *Character) UpdateMomentum(hit bool) {
	if hit {
		c.ConsecutiveHits++
		c.ConsecutiveMisses = 0
	} else {
		c.ConsecutiveMisses++
		c.ConsecutiveHits = 0
	}
}
