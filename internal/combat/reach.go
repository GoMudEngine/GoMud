package combat

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// PositionReachRadius returns the effective grapple radius (meters)
// for a Position state. Returns 0 for non-grapple states (Standing,
// Prone, Supine, Turtle); ReachUtility treats 0 as "no penalty,
// full damage."
//
// Standing-grapple states (Clinch, BackStanding) and ground-grapple
// states (Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
// BackGround, HalfGuard, Guard) get the configured radii.
func PositionReachRadius(s position.State) float64 {
	cfg := configs.GetBalanceConfig()
	switch s {
	case position.Clinch, position.BackStanding:
		return float64(cfg.ReachStandingGrappleRadius)
	case position.Mount, position.SideControl, position.KneeOnBelly,
		position.NorthSouth, position.Crucifix, position.BackGround,
		position.HalfGuard, position.Guard:
		return float64(cfg.ReachGroundGrappleRadius)
	default:
		// Standing, Prone, Supine, Turtle — no grapple-radius penalty.
		return 0
	}
}

// ReachUtility returns the damage multiplier from the reach curve.
// Returns 1.0 (no penalty) when the position has no grapple radius
// (zero sentinel) or when the weapon's reach fits inside the
// radius. Otherwise returns radius/reach, floored at
// Balance.ReachUtilityFloor so even maximally long weapons can
// still poke.
func ReachUtility(weaponReach, posRadius float64) float64 {
	if posRadius == 0 {
		return 1.0
	}
	if weaponReach <= posRadius {
		return 1.0
	}
	cfg := configs.GetBalanceConfig()
	util := posRadius / weaponReach
	floor := float64(cfg.ReachUtilityFloor)
	if util < floor {
		return floor
	}
	return util
}

// ShouldBludgeon reports whether the weapon's reach exceeds the
// position's grapple radius — i.e., the swing degraded to a
// pommel/hilt strike. Used by the attack-message selection site to
// swap bladed-weapon vocabulary to Crushing so the fiction tracks
// the math. Returns false for non-grapple positions (radius == 0).
func ShouldBludgeon(weaponReach, posRadius float64) bool {
	return posRadius > 0 && weaponReach > posRadius
}
