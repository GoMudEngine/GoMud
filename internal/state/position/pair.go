// Pair-aware transitions for the Position FSM. TransitionPair is
// the canonical way to put two characters into the same grapple
// state with role-appropriate GrappleData. Direct calls to
// TransitionToXxx remain available for tests + edge cases but
// don't enforce pair semantics.
//
// Chunk 4b-fixup: Role type + InitialControlForPair removed.
// GrappleData.IsControllerRole replaces ControlLevel as the
// minimal role discriminator.
package position

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// GrappleActor is the minimal interface TransitionPair requires from
// a character. Defined here so that internal/state/position does not
// import internal/characters (which imports position — cycle).
// *characters.Character satisfies this interface via GetUserId(),
// the public MobInstanceId field (exposed via GetMobInstanceId()),
// and the public Position field.
//
// Note: if Character does not yet have GetMobInstanceId(), callers
// can wrap with a thin shim or we add the method in T3.
type GrappleActor interface {
	GetUserId() int
	GetMobInstanceId() int
	GetPosition() *Machine
}

// ErrPairInvalidSourceStates is returned when TransitionPair is
// called with characters that aren't in compatible source states
// for the target transition.
var ErrPairInvalidSourceStates = errors.New("TransitionPair: source states incompatible with target")

// ErrPairRollbackFailed indicates the second-side transition
// failed and the rollback of the first side also failed —
// extremely unlikely but the consistency checker will catch it.
var ErrPairRollbackFailed = errors.New("TransitionPair: rollback of first side failed; pair may be desynced")

// DefaultEscapeTarget returns the position state to transition to
// when the controlled fighter escapes the given grapple position.
// Represents the most common BJJ outcome per state.
func DefaultEscapeTarget(current State) State {
	switch current {
	case Clinch:
		return Standing
	case BackStanding:
		return Standing
	case Mount:
		return HalfGuard
	case SideControl, KneeOnBelly, NorthSouth:
		return Guard
	case Crucifix:
		return SideControl
	case BackGround:
		return Mount
	case HalfGuard:
		return Guard
	case Guard:
		return Standing
	case Turtle:
		return Standing
	}
	return Standing
}

// TransitionPair atomically moves a controller + controlled pair
// into the same grapple state, stamping IsControllerRole=true on
// the controller side and false on the controlled side. Validates
// source states; rolls back the first side if the second fails.
// Standing target is a special case: clears both sides' GrappleData
// (the grapple has ended).
//
// Caller is responsible for identifying the controller and
// controlled (the existing CheckClinchProgression / grapple
// command code already knows which is which; new code can use
// IsController()).
func TransitionPair(
	controller, controlled GrappleActor,
	target State,
	r state.TransitionReason,
) error {
	if controller == nil || controlled == nil ||
		controller.GetPosition() == nil || controlled.GetPosition() == nil {
		return ErrPairInvalidSourceStates
	}

	// Standing target: break the grapple. Both sides return to
	// Standing; no GrappleData required.
	if target == Standing {
		prev := snapshotPosition(controller)
		if err := controller.GetPosition().TransitionToStanding(r); err != nil {
			return err
		}
		if err := controlled.GetPosition().TransitionToStanding(r); err != nil {
			if rbErr := restorePosition(controller, prev); rbErr != nil {
				return ErrPairRollbackFailed
			}
			return err
		}
		return nil
	}

	// Non-Standing target: build pair data + fire both transitions.
	// For symmetric states (Clinch, HalfGuard, Turtle), both sides
	// get IsControllerRole=false by convention.
	ctrlRef := state.ActorRef{
		UserId:        controller.GetUserId(),
		MobInstanceId: controller.GetMobInstanceId(),
	}
	cdRef := state.ActorRef{
		UserId:        controlled.GetUserId(),
		MobInstanceId: controlled.GetMobInstanceId(),
	}

	ctrlData := GrappleData{
		Partner:          cdRef,
		IsControllerRole: !isSymmetricGrapple(target),
	}
	cdData := GrappleData{
		Partner:          ctrlRef,
		IsControllerRole: false,
	}

	prev := snapshotPosition(controller)
	if err := transitionTo(controller.GetPosition(), target, ctrlData, r); err != nil {
		return err
	}
	if err := transitionTo(controlled.GetPosition(), target, cdData, r); err != nil {
		if rbErr := restorePosition(controller, prev); rbErr != nil {
			return ErrPairRollbackFailed
		}
		return err
	}
	return nil
}

// transitionTo dispatches to the right TransitionToXxx method per
// target state. Go doesn't have generic dispatch over state values;
// switch is the cleanest way to keep TransitionPair as a single
// helper.
func transitionTo(m *Machine, target State, d GrappleData, r state.TransitionReason) error {
	switch target {
	case Clinch:
		return m.TransitionToClinch(d, r)
	case BackStanding:
		return m.TransitionToBackStanding(d, r)
	case Mount:
		return m.TransitionToMount(d, r)
	case SideControl:
		return m.TransitionToSideControl(d, r)
	case KneeOnBelly:
		return m.TransitionToKneeOnBelly(d, r)
	case NorthSouth:
		return m.TransitionToNorthSouth(d, r)
	case Crucifix:
		return m.TransitionToCrucifix(d, r)
	case BackGround:
		return m.TransitionToBackGround(d, r)
	case HalfGuard:
		return m.TransitionToHalfGuard(d, r)
	case Guard:
		return m.TransitionToGuard(d, r)
	case Turtle:
		return m.TransitionToTurtle(d, r)
	}
	return ErrPairInvalidSourceStates
}

// positionSnapshot captures enough state to roll back a transition.
type positionSnapshot struct {
	state   State
	grapple *GrappleData
	prone   *ProneData
	supine  *SupineData
}

func snapshotPosition(c GrappleActor) positionSnapshot {
	m := c.GetPosition()
	snap := positionSnapshot{state: m.State()}
	if d, ok := m.GrappleData(); ok {
		snap.grapple = &d
	}
	if d, ok := m.ProneData(); ok {
		snap.prone = &d
	}
	if d, ok := m.SupineData(); ok {
		snap.supine = &d
	}
	return snap
}

// restorePosition reverses a transition. Uses ForceStanding then
// re-applies the snapshotted state. If the snapshot was Standing,
// just ForceStanding is enough.
func restorePosition(c GrappleActor, snap positionSnapshot) error {
	m := c.GetPosition()
	m.ForceStanding(state.TransitionReason{Trigger: "pair_rollback"})
	if snap.state == Standing {
		return nil
	}
	r := state.TransitionReason{Trigger: "pair_rollback"}
	if snap.grapple != nil {
		return transitionTo(m, snap.state, *snap.grapple, r)
	}
	if snap.prone != nil {
		return m.TransitionToProne(*snap.prone, r)
	}
	if snap.supine != nil {
		return m.TransitionToSupine(*snap.supine, r)
	}
	return nil
}
