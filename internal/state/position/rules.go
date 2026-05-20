package position

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// ErrPartnerRequired is returned when a grapple transition (other
// than Turtle) is called with a zero ActorRef Partner.
var ErrPartnerRequired = errors.New("grapple transition requires non-zero Partner (only Turtle accepts zero)")

// TransitionToStanding moves to Standing and clears all per-state
// data slots. Used for grapple-break, recovery, escape, and the
// Life Dead cascade.
func (m *Machine) TransitionToStanding(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Standing, r); err != nil {
		return err
	}
	m.prone = nil
	m.supine = nil
	m.grapple = nil
	return nil
}

// TransitionToProne moves to Prone (face-down knockdown, alone).
func (m *Machine) TransitionToProne(d ProneData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.prone
	m.prone = &d
	if err := m.inner.TransitionTo(Prone, r); err != nil {
		m.prone = prev
		return err
	}
	m.grapple = nil
	m.supine = nil
	return nil
}

// TransitionToSupine moves to Supine (face-up knockdown, alone).
func (m *Machine) TransitionToSupine(d SupineData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.supine
	m.supine = &d
	if err := m.inner.TransitionTo(Supine, r); err != nil {
		m.supine = prev
		return err
	}
	m.grapple = nil
	m.prone = nil
	return nil
}

// transitionGrapple is the shared body for all 11 grapple TransitionTo* methods.
// Validates Partner (non-zero except for Turtle), stores data with
// the Reason field populated, fires the inner transition with rollback
// on error, and clears non-grapple data slots on success.
func (m *Machine) transitionGrapple(target State, d GrappleData, r state.TransitionReason) error {
	if target != Turtle && d.Partner.IsZero() {
		return ErrPartnerRequired
	}
	d.Reason = r
	prev := m.grapple
	m.grapple = &d
	if err := m.inner.TransitionTo(target, r); err != nil {
		m.grapple = prev
		return err
	}
	m.prone = nil
	m.supine = nil
	return nil
}

func (m *Machine) TransitionToClinch(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Clinch, d, r)
}

func (m *Machine) TransitionToBackStanding(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(BackStanding, d, r)
}

func (m *Machine) TransitionToMount(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Mount, d, r)
}

func (m *Machine) TransitionToSideControl(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(SideControl, d, r)
}

func (m *Machine) TransitionToKneeOnBelly(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(KneeOnBelly, d, r)
}

func (m *Machine) TransitionToNorthSouth(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(NorthSouth, d, r)
}

func (m *Machine) TransitionToCrucifix(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Crucifix, d, r)
}

func (m *Machine) TransitionToBackGround(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(BackGround, d, r)
}

func (m *Machine) TransitionToHalfGuard(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(HalfGuard, d, r)
}

func (m *Machine) TransitionToGuard(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Guard, d, r)
}

func (m *Machine) TransitionToTurtle(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Turtle, d, r)
}

// ForceStanding transitions to Standing from any state, bypassing
// the validTransitions table. Used by admin commands and emergency
// cleanup. Idempotent if already Standing.
func (m *Machine) ForceStanding(r state.TransitionReason) {
	if m.State() == Standing {
		return
	}
	_ = m.inner.TransitionTo(Standing, r)
	m.prone = nil
	m.supine = nil
	m.grapple = nil
}
