package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestValidateGrapplePair_ValidMount verifies a correctly set-up Mount
// pair passes all invariants. Uses TransitionPair (the canonical path)
// so Control.State() is properly initialized (Controlling / Controlled).
func TestValidateGrapplePair_ValidMount(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	// Walk through Clinch → Mount via TransitionPair so ControlLevel
	// state is initialized alongside IsControllerRole.
	if err := position.TransitionPair(a, b, position.Clinch,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("TransitionPair → Clinch failed: %v", err)
	}
	if err := position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("TransitionPair → Mount failed: %v", err)
	}

	if err := position.ValidateGrapplePair(a, b); err != nil {
		t.Errorf("expected valid pair; got %v", err)
	}
}

// TestValidateGrapplePair_BothInControlRejected verifies that a pair
// where both sides are in the Controlling state fails with a
// role-exclusivity violation.
func TestValidateGrapplePair_BothInControlRejected(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	// Build pair via direct transitions. Then manually force both
	// Control machines to Controlling to create the role-exclusivity
	// violation (both sides claiming controller).
	_ = a.Position.TransitionToClinch(
		position.GrappleData{Partner: refB},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = a.Position.TransitionToMount(
		position.GrappleData{Partner: refB, IsControllerRole: true},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	_ = b.Position.TransitionToClinch(
		position.GrappleData{Partner: refA},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = b.Position.TransitionToMount(
		position.GrappleData{Partner: refA, IsControllerRole: true}, // BUG: both controller
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	// Mirror the broken IsControllerRole in Control.State() so invariant
	// 4 fires. TransitionToControlling is idempotent if already there.
	_ = a.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
	_ = b.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerGrappleEnter})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected role-exclusivity violation")
	}
	violation, ok := err.(position.PairInvariantViolation)
	if !ok {
		t.Fatalf("expected PairInvariantViolation; got %T", err)
	}
	if violation.Invariant != "control-exclusivity" {
		t.Errorf("expected control-exclusivity; got %q", violation.Invariant)
	}
}

func TestValidateGrapplePair_StateMismatchRejected(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	// a in Mount, b in SideControl — state mismatch.
	_ = a.Position.TransitionToClinch(position.GrappleData{Partner: refB}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, IsControllerRole: true}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refA}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToSideControl(position.GrappleData{Partner: refA, IsControllerRole: false}, state.TransitionReason{Trigger: position.TriggerTakedownSide})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected matching-state violation")
	}
	violation, _ := err.(position.PairInvariantViolation)
	if violation.Invariant != "matching-state" {
		t.Errorf("expected matching-state; got %q", violation.Invariant)
	}
}

func TestValidateGrapplePair_BrokenPartnerRefRejected(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	refB := state.ActorRef{UserId: 2}
	refStale := state.ActorRef{UserId: 99}

	_ = a.Position.TransitionToClinch(position.GrappleData{Partner: refB}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, IsControllerRole: true}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refStale}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToMount(position.GrappleData{Partner: refStale, IsControllerRole: false}, state.TransitionReason{Trigger: position.TriggerTakedownMount})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected single-partner violation")
	}
}
