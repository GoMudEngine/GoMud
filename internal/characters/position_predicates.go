// Position predicates on Character — chunk 4a additions.
// Each method delegates to c.Position.IsXxx() with a nil guard
// (a Character constructed outside New() and not run through
// Validate() may have c.Position == nil).
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// --- Per-state predicates (14) ---

// IsStanding returns true when the character is in Standing position.
func (c *Character) IsStanding() bool {
	if c.Position == nil {
		return true // defensive default; matches NewMachine() initial state
	}
	return c.Position.IsStanding()
}

// IsProne returns true when the character is face-down on the floor, alone.
func (c *Character) IsProne() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsProne()
}

// IsSupine returns true when the character is face-up on the floor, alone.
func (c *Character) IsSupine() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsSupine()
}

// IsClinch returns true when the character is in a standing grapple (clinch).
func (c *Character) IsClinch() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsClinch()
}

// IsBackStanding returns true when one grappler has the back of another, standing.
func (c *Character) IsBackStanding() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsBackStanding()
}

// IsMount returns true when the character is in Mount.
func (c *Character) IsMount() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsMount()
}

// IsSideControl returns true when the character is in Side Control.
func (c *Character) IsSideControl() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsSideControl()
}

// IsKneeOnBelly returns true when the character is in Knee-on-Belly.
func (c *Character) IsKneeOnBelly() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsKneeOnBelly()
}

// IsNorthSouth returns true when the character is in North-South.
func (c *Character) IsNorthSouth() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsNorthSouth()
}

// IsCrucifix returns true when the character is in Crucifix.
func (c *Character) IsCrucifix() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsCrucifix()
}

// IsBackGround returns true when the character is in Back-Ground (rear mount on ground).
func (c *Character) IsBackGround() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsBackGround()
}

// IsHalfGuard returns true when the character is in Half Guard.
func (c *Character) IsHalfGuard() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsHalfGuard()
}

// IsGuard returns true when the character is in Guard.
func (c *Character) IsGuard() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsGuard()
}

// IsTurtle returns true when the character is in Turtle.
func (c *Character) IsTurtle() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsTurtle()
}

// --- Rollup predicates (5) ---

// IsGrappling returns true for any grapple state (any of the 11).
func (c *Character) IsGrappling() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsGrappling()
}

// IsStandingGrapple returns true for Clinch or BackStanding.
func (c *Character) IsStandingGrapple() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsStandingGrapple()
}

// IsGroundGrapple returns true for any ground grapple state (9 states).
func (c *Character) IsGroundGrapple() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsGroundGrapple()
}

// IsTopDominant returns true when the character is in a controller-dominant
// ground position (Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
// BackGround). Does NOT take ControlLevel into account — that's a 4b
// refinement.
func (c *Character) IsTopDominant() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsTopDominant()
}

// IsOnFloor returns true for Prone, Supine, or any ground grapple.
func (c *Character) IsOnFloor() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsOnFloor()
}

// --- Control-axis predicates (chunk 4b) ---

// IsController returns true when the character is the controller
// side of a grapple pair. False outside of grapples.
func (c *Character) IsController() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsController()
}

// IsBeingControlled returns true when the character is the
// controlled side of a grapple pair.
func (c *Character) IsBeingControlled() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsBeingControlled()
}

// GetPositionSpeedMultiplier returns the combat attack-speed multiplier for
// the character's current Position state. Replaces the legacy
// CombatPosition.GetSpeedMultiplier helper (sunset in S5 of chunk 4b).
//
// Mapping (matches the legacy enum's intent; Turtle joins Prone/Supine
// because a balled-up defender swings just as poorly):
//
//	Standing                  → 1.0
//	Prone / Supine / Turtle   → 0.5
//	Clinch / BackStanding     → 0.6
//	Ground grapples (9 states) → 0.3
func (c *Character) GetPositionSpeedMultiplier() float64 {
	if c.Position == nil {
		return 1.0
	}
	switch c.Position.State() {
	case position.Standing:
		return 1.0
	case position.Prone, position.Supine, position.Turtle:
		return 0.5
	case position.Clinch, position.BackStanding:
		return 0.6
	default:
		return 0.3
	}
}

// IsLowGrappleStamina returns true when stamina fraction is below
// GrappleStaminaLowThreshold (config). Used by btree primitive
// mob_low_grapple_stamina and by Position_Messaging (T7) for
// stamina warnings.
func (c *Character) IsLowGrappleStamina() bool {
	cfg := configs.GetBalanceConfig()
	threshold := float64(cfg.GrappleStaminaLowThreshold)
	if threshold <= 0 {
		threshold = 0.25 // fallback if config not loaded
	}
	if c.StaminaMax.Value <= 0 {
		return false
	}
	return float64(c.Stamina)/float64(c.StaminaMax.Value) < threshold
}
