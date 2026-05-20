// Position-tiered hit modifiers for chunk 4e (third-party + defense
// degradation). Two lookup functions compose to produce the
// attacker's net hit modifier:
//
//	net = AttackerSelfHitModifier(attacker.Position.State(), attacker.Control.State())
//	    × TargetSideHitModifier(target.Position.State(), target.Control.State())
//
// Both default to 1.0 for Standing — no behavior change outside grapples.
// Symmetric — first-party in the grapple and third-party intruders both
// pick up the target-side bonus when hitting a grappled target.
//
// See docs/superpowers/specs/2026-05-19-state-chunk-4e-third-party-design.md
// §3 for the full tables + rationale + composition examples.
package position

import (
	"github.com/GoMudEngine/GoMud/internal/state/control"
)

// TargetSideHitModifier returns the multiplier applied to ANY attacker's
// hit roll when the target is in the given position + role.
// Higher = easier to hit a target in that position.
func TargetSideHitModifier(pos State, role control.State) float64 {
	switch pos {
	case Standing:
		return 1.0
	case Prone:
		return 1.10
	case Supine:
		return 1.15
	case Clinch:
		return 1.08
	case BackStanding:
		if role == control.Controlling {
			return 1.10
		}
		return 1.15
	case Mount:
		if role == control.Controlling {
			return 1.12
		}
		return 1.20
	case SideControl:
		if role == control.Controlling {
			return 1.10
		}
		return 1.15
	case KneeOnBelly:
		if role == control.Controlling {
			return 1.10
		}
		return 1.13
	case NorthSouth:
		if role == control.Controlling {
			return 1.10
		}
		return 1.12
	case Crucifix:
		if role == control.Controlling {
			return 1.10
		}
		return 1.25
	case BackGround:
		if role == control.Controlling {
			return 1.12
		}
		return 1.22
	case HalfGuard:
		if role == control.Controlling {
			return 1.08
		}
		return 1.10
	case Guard:
		// Inverted: bottom (Controlling) less exposed; top (Controlled) more exposed to legs.
		if role == control.Controlling {
			return 1.08
		}
		return 1.10
	case Turtle:
		if role == control.Controlling {
			return 1.10
		}
		return 1.12
	}
	return 1.0
}

// AttackerSelfHitModifier returns the multiplier applied to the attacker's
// own hit roll based on their OWN position + role. Higher = your position
// helps you swing; lower = your position hurts.
func AttackerSelfHitModifier(pos State, role control.State) float64 {
	switch pos {
	case Standing:
		return 1.0
	case Prone:
		return 0.75
	case Supine:
		return 0.70
	case Clinch:
		return 1.0
	case BackStanding:
		if role == control.Controlling {
			return 1.05
		}
		return 0.85
	case Mount:
		if role == control.Controlling {
			return 1.10
		}
		return 0.70
	case SideControl:
		if role == control.Controlling {
			return 1.05
		}
		return 0.75
	case KneeOnBelly:
		if role == control.Controlling {
			return 1.05
		}
		return 0.80
	case NorthSouth:
		if role == control.Controlling {
			return 1.0
		}
		return 0.80
	case Crucifix:
		if role == control.Controlling {
			return 1.10
		}
		return 0.50
	case BackGround:
		if role == control.Controlling {
			return 1.10
		}
		return 0.65
	case HalfGuard:
		if role == control.Controlling {
			return 1.05
		}
		return 0.85
	case Guard:
		// Inverted: bottom (Controlling) attacks via legs; top (Controlled) has frames.
		if role == control.Controlling {
			return 1.05
		}
		return 0.90
	case Turtle:
		if role == control.Controlling {
			return 1.0
		}
		return 0.85
	}
	return 1.0
}
