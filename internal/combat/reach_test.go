package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// TestPositionReachRadius_AllStates verifies the radius lookup for all 14
// Position states. Standing/Prone/Supine/Turtle return 0 (no penalty);
// Clinch/BackStanding return the standing-grapple radius; all ground
// grapple states return the ground-grapple radius.
func TestPositionReachRadius_AllStates(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	standR := float64(cfg.ReachStandingGrappleRadius)
	groundR := float64(cfg.ReachGroundGrappleRadius)

	cases := map[position.State]float64{
		position.Standing:     0,
		position.Prone:        0,
		position.Supine:       0,
		position.Turtle:       0,
		position.Clinch:       standR,
		position.BackStanding: standR,
		position.Mount:        groundR,
		position.SideControl:  groundR,
		position.KneeOnBelly:  groundR,
		position.NorthSouth:   groundR,
		position.Crucifix:     groundR,
		position.BackGround:   groundR,
		position.HalfGuard:    groundR,
		position.Guard:        groundR,
	}
	for s, want := range cases {
		assert.Equal(t, want, PositionReachRadius(s), "state %s", s)
	}
}

// TestReachUtility_ZeroRadius_NoPenalty ensures a pike in a standing
// (non-grapple) position receives no penalty.
func TestReachUtility_ZeroRadius_NoPenalty(t *testing.T) {
	assert.Equal(t, 1.0, ReachUtility(2.0, 0)) // pike in standing
}

// TestReachUtility_WeaponFits_NoPenalty ensures weapons at or below the
// position radius take no penalty.
func TestReachUtility_WeaponFits_NoPenalty(t *testing.T) {
	assert.Equal(t, 1.0, ReachUtility(0.3, 0.3)) // dagger in mount, exact
	assert.Equal(t, 1.0, ReachUtility(0.1, 0.3)) // fist in mount, under
}

// TestReachUtility_WeaponExceedsRadius_PenaltyApplies verifies the
// radius/reach calculation for standard weapon-in-grapple cases.
func TestReachUtility_WeaponExceedsRadius_PenaltyApplies(t *testing.T) {
	// Sword (1.0m) in mount (0.3m) → 0.3/1.0 = 0.3
	assert.InDelta(t, 0.3, ReachUtility(1.0, 0.3), 0.001)
	// Sword in clinch (0.5m) → 0.5/1.0 = 0.5
	assert.InDelta(t, 0.5, ReachUtility(1.0, 0.5), 0.001)
}

// TestReachUtility_FlooredAtConfigMin ensures the floor prevents
// extreme penalties — a pike in mount raw would be 0.1, floored to 0.15.
func TestReachUtility_FlooredAtConfigMin(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	// Pike (3.0m) in mount (0.3m) raw would be 0.1 → floored to cfg floor
	assert.Equal(t, float64(cfg.ReachUtilityFloor), ReachUtility(3.0, 0.3))
}

// TestShouldBludgeon_PositiveAndNegative covers the four key cases:
// weapon exceeds radius (true), weapon fits exactly (false), weapon
// under radius (false), and non-grapple position (false).
func TestShouldBludgeon_PositiveAndNegative(t *testing.T) {
	assert.True(t, ShouldBludgeon(1.0, 0.3))  // sword in mount
	assert.False(t, ShouldBludgeon(0.3, 0.3)) // dagger fits exactly
	assert.False(t, ShouldBludgeon(0.1, 0.3)) // fist under
	assert.False(t, ShouldBludgeon(1.0, 0))   // not grappling
}

// TestBehaviorMatrix_Reach is the PB-201..PB-220 behavior matrix from the
// chunk 4c design spec. Each case exercises PositionReachRadius +
// ReachUtility + ShouldBludgeon in combination so the composite behaviour
// (radius × reach → multiplier, bludgeon flag) is covered by a single
// named scenario.
func TestBehaviorMatrix_Reach(t *testing.T) {
	cfg := configs.GetBalanceConfig()
	floor := float64(cfg.ReachUtilityFloor)

	tests := []struct {
		id           string
		pos          position.State
		reach        float64
		wantMult     float64
		wantBludgeon bool
	}{
		{"PB-201 fist in mount", position.Mount, 0.1, 1.00, false},
		{"PB-202 dagger in mount", position.Mount, 0.3, 1.00, false},
		{"PB-203 sword in mount", position.Mount, 1.0, 0.30, true},
		{"PB-204 spear in mount (floored)", position.Mount, 2.0, floor, true},
		{"PB-205 sword standing", position.Standing, 1.0, 1.00, false},
		{"PB-206 sword vs prone (attacker standing)", position.Standing, 1.0, 1.00, false},
		{"PB-207 sword in clinch", position.Clinch, 1.0, 0.50, true},
		{"PB-208 wand in clinch", position.Clinch, 0.4, 1.00, false},
		{"PB-209 wand in mount", position.Mount, 0.4, 0.75, true},
		{"PB-210 staff in mount", position.Mount, 1.5, 0.20, true},
		{"PB-211 greatsword halfguard", position.HalfGuard, 1.5, 0.20, true},
		// PB-212: 0.5/3.0 = 0.1667 — above the 0.15 floor, so no floor applied.
		{"PB-212 pike in clinch", position.Clinch, 3.0, 0.5 / 3.0, true},
		{"PB-219 pike in mount floor", position.Mount, 3.0, floor, true},
		{"PB-220 turtle no penalty", position.Turtle, 1.5, 1.00, false},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			radius := PositionReachRadius(tc.pos)
			mult := ReachUtility(tc.reach, radius)
			bld := ShouldBludgeon(tc.reach, radius)
			assert.InDelta(t, tc.wantMult, mult, 0.005, "mult")
			assert.Equal(t, tc.wantBludgeon, bld, "bludgeon")
		})
	}
}
