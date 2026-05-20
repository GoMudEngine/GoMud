package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	_ "github.com/GoMudEngine/GoMud/internal/hooks" // wire init() observers
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestPositionCascadeFromMount covers PO-038: Life Dead from Mount
// cascades Position → Standing.
func TestPositionCascadeFromMount(t *testing.T) {
	c := characters.New()
	_ = c.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 100}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = c.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 100}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if !c.IsMount() {
		t.Fatal("setup: expected Mount")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Position should cascade to Standing on Life Dead; got %v", c.Position.State())
	}
}

// TestPositionCascadeFromGuard covers PO-039.
func TestPositionCascadeFromGuard(t *testing.T) {
	c := characters.New()
	_ = c.Position.TransitionToSupine(
		position.SupineData{},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward},
	)
	_ = c.Position.TransitionToGuard(
		position.GrappleData{Partner: state.ActorRef{UserId: 101}},
		state.TransitionReason{Trigger: position.TriggerGuardPull},
	)
	if !c.IsGuard() {
		t.Fatal("setup: expected Guard")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Position should cascade to Standing on Life Dead from Guard; got %v", c.Position.State())
	}
}

// TestPositionCascadeFromBackGround covers PO-040.
func TestPositionCascadeFromBackGround(t *testing.T) {
	c := characters.New()
	_ = c.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 102}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = c.Position.TransitionToBackGround(
		position.GrappleData{Partner: state.ActorRef{UserId: 102}},
		state.TransitionReason{Trigger: position.TriggerTakedownBackGround},
	)
	if !c.IsBackGround() {
		t.Fatal("setup: expected BackGround")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Position should cascade to Standing on Life Dead from BackGround; got %v", c.Position.State())
	}
}

// TestPositionCascadeNoOpFromStanding covers PO-037 — a character
// who dies while Standing already remains Standing. The observer
// should early-return without firing a redundant transition.
func TestPositionCascadeNoOpFromStanding(t *testing.T) {
	c := characters.New()
	if !c.IsStanding() {
		t.Fatal("setup: expected Standing default")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Standing should remain Standing on Life Dead; got %v", c.Position.State())
	}
}
