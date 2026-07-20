package characters

import (
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// ConditionType identifies a temporary combat condition.
type ConditionType int

const (
	ConditionRecoveryPenalty   ConditionType = iota // Limits attacks to 1 this round (prone recovery)
	ConditionDefensePenalty                         // Reduces defense this round (failed grapple exposure)
	ConditionShield                                 // Magical armor barrier (+physical armor, Stage 11.4)
	ConditionRegen                                  // Regenerates HP each AutoHeal tick (heal spell, Stage 11.5)
	ConditionBlinded                                // Reduces perception/dodge/accuracy (Phase 24.5)
	ConditionPoisoned                               // DoT damage over time (Phase 24.5)
	ConditionEnchantWithdrawal                      // Pool max penalty after disenchanting (Stage 31.6)
	ConditionBleeding                               // Wounds seeping blood, taking damage over time (Stage 42.7)
	ConditionWarcry                                 // Physical damage multiplier from warcry shout
	ConditionRally                                  // Defense score multiplier from rally shout
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
	case ConditionShield:
		return "Minor Shield"
	case ConditionRegen:
		return "Regenerating"
	case ConditionBlinded:
		return "Blinded"
	case ConditionPoisoned:
		return "Poisoned"
	case ConditionEnchantWithdrawal:
		return "Enchant Withdrawal"
	case ConditionBleeding:
		return "Bleeding"
	case ConditionWarcry:
		return "Warcry"
	case ConditionRally:
		return "Rally"
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
	case ConditionShield:
		return "Magical armor barrier (+physical armor)"
	case ConditionRegen:
		return "Healing magic mending wounds over time"
	case ConditionBlinded:
		return "Vision impaired — dodge and accuracy reduced"
	case ConditionPoisoned:
		return "Toxins coursing through your body, dealing damage over time"
	case ConditionEnchantWithdrawal:
		return "Weakened from severing a Chrysalis bond"
	case ConditionBleeding:
		return "Wounds seeping blood, taking damage over time"
	case ConditionWarcry:
		return "A rallying battle cry bolsters your fighting spirit"
	case ConditionRally:
		return "An inspiring shout steadies your defenses"
	default:
		return ""
	}
}

// AddCondition adds or overwrites a combat condition of the given type.
func (c *Character) AddCondition(typ ConditionType, duration int, magnitude float64, source string) {
	for i, cond := range c.Conditions {
		if cond.Type == typ {
			c.Conditions[i] = CombatCondition{Type: typ, Duration: duration, Magnitude: magnitude, Source: source}
			// Chunk 6 (Perception): ConditionBlinded triggers Sighted → Blinded.
			// Guard against re-entry: only fire if state is currently Sighted.
			if typ == ConditionBlinded && c.Perception != nil && c.Perception.State() == perception.Sighted {
				_ = c.Perception.TransitionTo(perception.Blinded,
					state.TransitionReason{Trigger: perception.TriggerConditionAdded})
			}
			return
		}
	}
	c.Conditions = append(c.Conditions, CombatCondition{Type: typ, Duration: duration, Magnitude: magnitude, Source: source})
	// Chunk 6 (Perception): ConditionBlinded triggers Sighted → Blinded.
	// Guard against re-entry: only fire if state is currently Sighted.
	if typ == ConditionBlinded && c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerConditionAdded})
	}
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
	removed := false
	for i, cond := range c.Conditions {
		if cond.Type == typ {
			c.Conditions = append(c.Conditions[:i], c.Conditions[i+1:]...)
			removed = true
			break
		}
	}
	// Chunk 6 (Perception): ConditionBlinded clear may flip Blinded →
	// Sighted, but only if no other blind source is still active.
	if removed && typ == ConditionBlinded && c.Perception != nil && c.Perception.State() == perception.Blinded && !c.HasAnyBlindSource() {
		_ = c.Perception.TransitionTo(perception.Sighted,
			state.TransitionReason{Trigger: perception.TriggerConditionRemoved})
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
