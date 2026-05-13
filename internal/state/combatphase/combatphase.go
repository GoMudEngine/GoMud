// Package combatphase defines the Combat Phase state machine,
// the first consumer of internal/state. It replaces the
// Character.Aggro field as the source of truth for "who am I
// attacking?" and "am I in combat?".
package combatphase

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Combat Phase state enum.
type State int

const (
	Idle State = iota
	Engaging
	Engaged
	Disengaging
)

// String for logging / debugging.
func (s State) String() string {
	switch s {
	case Idle:
		return "Idle"
	case Engaging:
		return "Engaging"
	case Engaged:
		return "Engaged"
	case Disengaging:
		return "Disengaging"
	}
	return "Unknown"
}

// EngagingData is the state-data type for the Engaging state.
type EngagingData struct {
	Target      state.ActorRef
	Reason      state.TransitionReason
	RoundsUntil int // weapon WaitRounds before swing
}

// EngagedData is the state-data type for the Engaged state.
type EngagedData struct {
	Target       state.ActorRef
	NextSwingAt  int  // round number for next swing
	SurpriseLeft bool // true during the first round of a SurpriseAttack engagement
}

// DisengagingData is the state-data type for the Disengaging state.
type DisengagingData struct {
	LastTarget state.ActorRef // target at time of flee
	FleeRound  int            // round flee was initiated
}

// Machine wraps state.Machine[State] with Combat-Phase-specific
// API including per-state data storage and Attackers tracking.
//
// Per-state data and inbound attacker tracking are populated by
// the transition methods that arrive in Tasks 5-8. This Task 3
// only establishes the type with empty data slots and the basic
// State() / Inner() accessors.
type Machine struct {
	inner       *state.Machine[State]
	engaging    *EngagingData
	engaged     *EngagedData
	disengaging *DisengagingData
	attackers   []state.ActorRef // inbound attacker list
}

// NewMachine returns a Combat Phase machine in Idle.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Idle, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// EngagingData returns the Engaging state's data if currently Engaging.
func (m *Machine) EngagingData() (EngagingData, bool) {
	if m.State() != Engaging || m.engaging == nil {
		return EngagingData{}, false
	}
	return *m.engaging, true
}

// EngagedData returns the Engaged state's data if currently Engaged.
func (m *Machine) EngagedData() (EngagedData, bool) {
	if m.State() != Engaged || m.engaged == nil {
		return EngagedData{}, false
	}
	return *m.engaged, true
}

// DisengagingData returns the Disengaging state's data if currently
// Disengaging.
func (m *Machine) DisengagingData() (DisengagingData, bool) {
	if m.State() != Disengaging || m.disengaging == nil {
		return DisengagingData{}, false
	}
	return *m.disengaging, true
}

// Attackers returns the inbound attacker list — characters
// currently Engaging or Engaged with this character as their
// target. Framework-maintained; do not mutate directly.
func (m *Machine) Attackers() []state.ActorRef {
	out := make([]state.ActorRef, len(m.attackers))
	copy(out, m.attackers)
	return out
}

// IsEngaged returns true if Combat Phase is Engaged.
func (m *Machine) IsEngaged() bool {
	return m.State() == Engaged
}

// IsInCombat returns true if Combat Phase is anything other
// than Idle. (Engaging, Engaged, and Disengaging all count.)
func (m *Machine) IsInCombat() bool {
	return m.State() != Idle
}

// CurrentTarget returns the ActorRef of the current target if
// any state has one (Engaging, Engaged, Disengaging), else zero.
func (m *Machine) CurrentTarget() state.ActorRef {
	switch m.State() {
	case Engaging:
		if m.engaging != nil {
			return m.engaging.Target
		}
	case Engaged:
		if m.engaged != nil {
			return m.engaged.Target
		}
	case Disengaging:
		if m.disengaging != nil {
			return m.disengaging.LastTarget
		}
	}
	return state.ActorRef{}
}

// Inner returns the underlying state.Machine — used by rules.go
// (Task 5+) to register vetoes/cascades. Not part of the stable
// API; do not depend on it from outside this package.
func (m *Machine) Inner() *state.Machine[State] {
	return m.inner
}
