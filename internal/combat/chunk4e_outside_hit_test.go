package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// enableOutsideHitForTest forces ControlDegradeOnOutsideHit to true in the
// test environment. Tests run without a loaded config file so ConfigBool
// defaults to false; this override ensures the guard doesn't short-circuit
// the helper logic under test.
func enableOutsideHitForTest(t *testing.T) {
	t.Helper()
	_ = configs.AddOverlayOverrides(map[string]any{
		"Balance.ControlDegradeOnOutsideHit": true,
	})
}

// buildControllerAndPartner constructs two characters in a Clinch
// grapple where 'a' is the Controlling side and 'b' is Controlled.
// 'c' (if non-nil) is a standing third party.
func buildControllerAndPartner(t *testing.T) (a, b, c *characters.Character) {
	t.Helper()
	newChar := func(uid int) *characters.Character {
		ch := characters.New()
		ch.SetUserId(uid)
		return ch
	}
	a = newChar(1)
	b = newChar(2)
	c = newChar(3)

	// Put both into a Clinch with each other as partner.
	err := a.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: b.GetUserId()}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	assert.NoError(t, err, "a.TransitionToClinch")

	err = b.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: a.GetUserId()}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	assert.NoError(t, err, "b.TransitionToClinch")

	// Elevate 'a' to Controlling via the control FSM.
	err = a.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerDriftWin})
	assert.NoError(t, err, "a.TransitionToControlling")

	err = b.Control.TransitionToControlled(state.TransitionReason{Trigger: control.TriggerDriftLoss})
	assert.NoError(t, err, "b.TransitionToControlled")

	return a, b, c
}

// TestChunk4e_OutsideHitDegradesController verifies §5 of the chunk 4e
// spec: when a third party damages a grapple controller, their ControlLevel
// shifts one step toward Neutral.
func TestChunk4e_OutsideHitDegradesController(t *testing.T) {
	enableOutsideHitForTest(t)
	a, _, c := buildControllerAndPartner(t)

	// Pre-condition: 'a' is Controlling.
	assert.Equal(t, control.Controlling, a.Control.State(), "pre: a should be Controlling")

	// 'c' (third party) hits 'a'.
	chunk4eApplyOutsideHitDisruption(c, a)

	// Post-condition: 'a' should now be Neutral (one step toward Neutral).
	assert.Equal(t, control.Neutral, a.Control.State(), "post: a should be Neutral after outside hit")
}

// TestChunk4e_OutsideHitOncePerRound verifies that the disruption fires
// at most once per round (dedupe via OutsideHitDisruptedRound).
func TestChunk4e_OutsideHitOncePerRound(t *testing.T) {
	enableOutsideHitForTest(t)
	a, _, c := buildControllerAndPartner(t)

	// First hit: shifts from Controlling → Neutral.
	chunk4eApplyOutsideHitDisruption(c, a)
	assert.Equal(t, control.Neutral, a.Control.State(), "first hit: a → Neutral")

	// Manually push 'a' back to Controlling to simulate regaining control.
	err := a.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerDriftWin})
	assert.NoError(t, err)
	assert.Equal(t, control.Controlling, a.Control.State(), "re-setup: a back to Controlling")

	// Second hit in the SAME round: OutsideHitDisruptedRound should block it.
	// (Round counter is 0 in tests; OutsideHitDisruptedRound was set to 0
	// by the first hit, so both use the same round number — second should no-op.)
	chunk4eApplyOutsideHitDisruption(c, a)
	assert.Equal(t, control.Controlling, a.Control.State(), "second hit same round: a should stay Controlling")
}

// TestChunk4e_PartnerHitDoesNotDisrupt verifies that damage from the
// grapple partner does NOT trigger the outside-hit disruption.
func TestChunk4e_PartnerHitDoesNotDisrupt(t *testing.T) {
	a, b, _ := buildControllerAndPartner(t)

	// 'b' is the grapple partner of 'a', not a third party.
	chunk4eApplyOutsideHitDisruption(b, a)

	// 'a' should remain Controlling — partner hits don't degrade.
	assert.Equal(t, control.Controlling, a.Control.State(), "partner hit: a should remain Controlling")
}

// TestChunk4e_NonControllerNotDisrupted verifies that the disruption
// only fires when the target is the Controller (not Neutral/Controlled).
func TestChunk4e_NonControllerNotDisrupted(t *testing.T) {
	a, b, c := buildControllerAndPartner(t)

	// 'b' is Controlled, not Controlling.
	chunk4eApplyOutsideHitDisruption(c, b)

	// 'a' is the Controller; its state should be unchanged.
	assert.Equal(t, control.Controlling, a.Control.State(), "controller unchanged")

	// 'b' is Controlled; disruption should not affect them.
	assert.Equal(t, control.Controlled, b.Control.State(), "b remains Controlled")
}

// TestChunk4e_NilSafetyGuards verifies that nil attacker or target
// does not panic — these guards protect against edge cases at call sites.
func TestChunk4e_NilSafetyGuards(t *testing.T) {
	a, _, c := buildControllerAndPartner(t)

	assert.NotPanics(t, func() {
		chunk4eApplyOutsideHitDisruption(nil, a)
	}, "nil attacker should not panic")

	assert.NotPanics(t, func() {
		chunk4eApplyOutsideHitDisruption(c, nil)
	}, "nil target should not panic")
}
