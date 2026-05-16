package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestValidateGrapplePair_ValidMount(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	_ = a.Position.TransitionToClinch(
		position.GrappleData{Partner: refB},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = a.Position.TransitionToMount(
		position.GrappleData{Partner: refB, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	_ = b.Position.TransitionToClinch(
		position.GrappleData{Partner: refA},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = b.Position.TransitionToMount(
		position.GrappleData{Partner: refA, ControlLevel: position.Controlled},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)

	if err := position.ValidateGrapplePair(a, b); err != nil {
		t.Errorf("expected valid pair; got %v", err)
	}
}

func TestValidateGrapplePair_BothInControlRejected(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	// Force both into Mount with InControl (invariant violation).
	_ = a.Position.TransitionToClinch(
		position.GrappleData{Partner: refB},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = a.Position.TransitionToMount(
		position.GrappleData{Partner: refB, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	_ = b.Position.TransitionToClinch(
		position.GrappleData{Partner: refA},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = b.Position.TransitionToMount(
		position.GrappleData{Partner: refA, ControlLevel: position.InControl}, // BUG: both InControl
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected role-exclusivity violation")
	}
	violation, ok := err.(position.PairInvariantViolation)
	if !ok {
		t.Fatalf("expected PairInvariantViolation; got %T", err)
	}
	if violation.Invariant != "role-exclusivity" {
		t.Errorf("expected role-exclusivity; got %q", violation.Invariant)
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
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, ControlLevel: position.InControl}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refA}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToSideControl(position.GrappleData{Partner: refA, ControlLevel: position.Controlled}, state.TransitionReason{Trigger: position.TriggerTakedownSide})

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
	_ = a.Position.TransitionToMount(position.GrappleData{Partner: refB, ControlLevel: position.InControl}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	_ = b.Position.TransitionToClinch(position.GrappleData{Partner: refStale}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = b.Position.TransitionToMount(position.GrappleData{Partner: refStale, ControlLevel: position.Controlled}, state.TransitionReason{Trigger: position.TriggerTakedownMount})

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected single-partner violation")
	}
}
