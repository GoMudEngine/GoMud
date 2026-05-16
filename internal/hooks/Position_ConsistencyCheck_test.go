package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestConsistencyCheck_NoFalsePositives_ValidMount verifies that a
// pair set up correctly via TransitionPair passes ValidateGrapplePair.
// Standing → Mount isn't a direct edge, so route through Clinch first.
func TestConsistencyCheck_NoFalsePositives_ValidMount(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	if err := position.TransitionPair(a, b, position.Clinch,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("TransitionPair → Clinch failed: %v", err)
	}
	if err := position.TransitionPair(a, b, position.Mount,
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("TransitionPair → Mount failed: %v", err)
	}

	if err := position.ValidateGrapplePair(a, b); err != nil {
		t.Errorf("ValidateGrapplePair on valid Mount should return nil; got %v", err)
	}
}

// TestForceBreakPair_BothReturnToStanding verifies that the
// force-break path unwinds both sides to Standing regardless of the
// invariant violation that triggered it. Builds a deliberately broken
// pair (both sides InControl in Mount — role-exclusivity violation)
// and confirms the recovery.
func TestForceBreakPair_BothReturnToStanding(t *testing.T) {
	a := characters.New()
	a.SetUserId(1)
	b := characters.New()
	b.SetUserId(2)

	refA := state.ActorRef{UserId: 1}
	refB := state.ActorRef{UserId: 2}

	// Walk both through Standing → Clinch → Mount, then deliberately
	// stamp both ControlLevels to InControl to violate invariant 4.
	if err := a.Position.TransitionToClinch(
		position.GrappleData{Partner: refB, ControlLevel: position.Neutral},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("a → Clinch failed: %v", err)
	}
	if err := a.Position.TransitionToMount(
		position.GrappleData{Partner: refB, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("a → Mount failed: %v", err)
	}
	if err := b.Position.TransitionToClinch(
		position.GrappleData{Partner: refA, ControlLevel: position.Neutral},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("b → Clinch failed: %v", err)
	}
	if err := b.Position.TransitionToMount(
		position.GrappleData{Partner: refA, ControlLevel: position.InControl},
		state.TransitionReason{Trigger: position.TriggerTakedownMount}); err != nil {
		t.Fatalf("b → Mount failed: %v", err)
	}

	err := position.ValidateGrapplePair(a, b)
	if err == nil {
		t.Fatal("expected invariant violation in setup (both InControl)")
	}

	forceBreakPair(a, b, err)

	if !a.Position.IsStanding() {
		t.Errorf("a should be Standing after force-break; got %v", a.Position.State())
	}
	if !b.Position.IsStanding() {
		t.Errorf("b should be Standing after force-break; got %v", b.Position.State())
	}
}

// TestForceBreakSolo_ReturnsToStanding verifies the solo recovery
// path (used when a character is grappling but its partner can't be
// resolved — a partner-disappeared edge case). Uses Clinch as the
// stand-in grapple state; the function only cares that the FSM is
// in some grapple state and can be force-recovered to Standing.
func TestForceBreakSolo_ReturnsToStanding(t *testing.T) {
	c := characters.New()
	c.SetUserId(1)
	dangling := state.ActorRef{UserId: 999}

	if err := c.Position.TransitionToClinch(
		position.GrappleData{Partner: dangling, ControlLevel: position.Neutral},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry}); err != nil {
		t.Fatalf("→ Clinch failed: %v", err)
	}

	forceBreakSolo(c, "test")

	if !c.Position.IsStanding() {
		t.Errorf("c should be Standing after solo force-break; got %v", c.Position.State())
	}
}
