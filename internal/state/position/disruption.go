// Per-position spell-disruption damage%-equivalents (chunk 4f). Fed into
// the existing characters.CalcConcentrationChance(Wil, dmgPctEquiv) curve
// in internal/hooks/combat_shared_helpers.go:processFoldRound to produce a
// per-round, Willpower-mediated disruption chance. Replaces the three
// deterministic 100% gates that chunks 4e and earlier shipped.
//
// Returns 0 for Standing (no check). Higher = more disruption. Controlled-role
// values are generally >= controller-role values because the controlled side
// has hands and movement suppressed. Guard inverts: bottom (Controlling) has
// free hands and lower disruption than top (Controlled).
//
// See docs/superpowers/specs/2026-05-19-state-chunk-4f-balance-smoke-design.md
// §3.1 for the table + rationale.
package position

import (
	"github.com/GoMudEngine/GoMud/internal/state/control"
)

// PositionDisruptionDmgEquiv returns the damage%-equivalent for a caster in
// the given position + control role. Returns 0 for Standing (skip the
// check). Returned value is fed to CalcConcentrationChance(Wil, dmgPctEquiv).
func PositionDisruptionDmgEquiv(pos State, role control.State) int {
	switch pos {
	case Standing:
		return 0
	case Prone:
		return 30
	case Supine:
		return 25
	case Clinch:
		return 40
	case BackStanding:
		if role == control.Controlling {
			return 35
		}
		return 50
	case Mount:
		if role == control.Controlling {
			return 35
		}
		return 60
	case SideControl:
		if role == control.Controlling {
			return 30
		}
		return 55
	case KneeOnBelly:
		if role == control.Controlling {
			return 30
		}
		return 50
	case NorthSouth:
		if role == control.Controlling {
			return 30
		}
		return 45
	case Crucifix:
		if role == control.Controlling {
			return 35
		}
		return 70
	case BackGround:
		if role == control.Controlling {
			return 35
		}
		return 65
	case HalfGuard:
		if role == control.Controlling {
			return 30
		}
		return 40
	case Guard:
		// Inverted: bottom (Controlling) has free hands; top (Controlled)
		// is stuck on someone's hips.
		if role == control.Controlling {
			return 25
		}
		return 40
	case Turtle:
		if role == control.Controlling {
			return 35
		}
		return 45
	}
	return 0
}
