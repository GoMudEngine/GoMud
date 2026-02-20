package characters

// ConditionType identifies a temporary combat condition.
type ConditionType int

const (
	ConditionRecoveryPenalty  ConditionType = iota // Limits attacks to 1 this round (prone recovery)
	ConditionDefensePenalty                        // Reduces defense this round (failed grapple exposure)
	ConditionGrappleController                     // Is the active grapple controller
	ConditionShield                                // Magical armor barrier (+physical armor, Stage 11.4)
)

// CombatCondition represents a single active combat state on a character.
type CombatCondition struct {
	Type      ConditionType
	Duration  int     // rounds remaining; 0 = permanent (cleared explicitly only)
	Magnitude float64 // effect strength (e.g., 0.85 for defense multiplier)
	Source    string  // for debugging / display
}

// DisplayName returns a human-readable name for this condition type.
func (c ConditionType) DisplayName() string {
	switch c {
	case ConditionRecoveryPenalty:
		return "Recovery Penalty"
	case ConditionDefensePenalty:
		return "Defense Penalty"
	case ConditionGrappleController:
		return "Grapple Control"
	case ConditionShield:
		return "Minor Shield"
	default:
		return "Unknown Condition"
	}
}

// Description returns a short description of this condition type's effect.
func (c ConditionType) Description() string {
	switch c {
	case ConditionRecoveryPenalty:
		return "Attacks reduced to 1 (prone recovery)"
	case ConditionDefensePenalty:
		return "Defense reduced 15% (off-balance, exposed)"
	case ConditionGrappleController:
		return "Active grapple controller"
	case ConditionShield:
		return "Magical armor barrier (+physical armor)"
	default:
		return ""
	}
}

// AddCondition adds or overwrites a combat condition of the given type.
func (c *Character) AddCondition(typ ConditionType, duration int, magnitude float64, source string) {
	for i, cond := range c.Conditions {
		if cond.Type == typ {
			c.Conditions[i] = CombatCondition{Type: typ, Duration: duration, Magnitude: magnitude, Source: source}
			return
		}
	}
	c.Conditions = append(c.Conditions, CombatCondition{Type: typ, Duration: duration, Magnitude: magnitude, Source: source})
}

// HasCondition returns true if the character currently has the given condition.
func (c *Character) HasCondition(typ ConditionType) bool {
	for _, cond := range c.Conditions {
		if cond.Type == typ {
			return true
		}
	}
	return false
}

// GetConditionMagnitude returns the magnitude of the given condition, or 0 if absent.
func (c *Character) GetConditionMagnitude(typ ConditionType) float64 {
	for _, cond := range c.Conditions {
		if cond.Type == typ {
			return cond.Magnitude
		}
	}
	return 0
}

// RemoveCondition removes the given condition type if present.
func (c *Character) RemoveCondition(typ ConditionType) {
	for i, cond := range c.Conditions {
		if cond.Type == typ {
			c.Conditions = append(c.Conditions[:i], c.Conditions[i+1:]...)
			return
		}
	}
}

// DecrementCondition subtracts 1 from the Duration of the matching condition.
// Does nothing if the condition is absent or permanent (Duration == 0).
func (c *Character) DecrementCondition(typ ConditionType) {
	for i, cond := range c.Conditions {
		if cond.Type == typ {
			if cond.Duration > 0 {
				c.Conditions[i].Duration--
			}
			return
		}
	}
}

// GetConditionDuration returns the current Duration for the given condition type,
// or 0 if the condition is absent.
func (c *Character) GetConditionDuration(typ ConditionType) int {
	for _, cond := range c.Conditions {
		if cond.Type == typ {
			return cond.Duration
		}
	}
	return 0
}

// TickConditions decrements Duration on all timed conditions and removes any that reach 0.
// Called once per round end. Permanent conditions (Duration == 0) are never decremented.
func (c *Character) TickConditions() {
	remaining := c.Conditions[:0]
	for _, cond := range c.Conditions {
		if cond.Duration == 0 {
			// Permanent — keep as-is
			remaining = append(remaining, cond)
			continue
		}
		cond.Duration--
		if cond.Duration > 0 {
			remaining = append(remaining, cond)
		}
		// Duration hit 0 → condition expires, do not append
	}
	c.Conditions = remaining
}
