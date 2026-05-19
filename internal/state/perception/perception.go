// Package perception defines the Perception state machine — the sixth
// consumer of internal/state, after combatphase, awareness, life,
// activity, position, and presence. Two-state FSM (Sighted / Blinded)
// gating broadcast visibility and active-inspection.
//
// SHIPS DORMANT in chunk 6. Transitions fire correctly via buff /
// condition observers, but no consumer reads the state yet. The future
// centralized messaging framework chunk will consume this primitive
// (broadcast cutover, infrared rendering, look-cmd gating, color
// coding, line wrapping). See
// docs/superpowers/specs/2026-05-19-state-chunk-6-perception-design.md
// for the full design and the messaging-framework memory for the
// successor work.
package perception

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Perception state enum.
type State int

const (
	Sighted State = iota // default — eyes work
	Blinded              // any active blind source
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Sighted:
		return "Sighted"
	case Blinded:
		return "Blinded"
	}
	return "Unknown"
}

// Machine wraps state.Machine[State] with Perception-specific API.
// Mirrors the chunk-5 presence.Machine wrapper.
type Machine struct {
	inner *state.Machine[State]
	self  state.ActorRef
}

// NewMachine returns a Machine in Sighted. Same constructor for both
// players and mobs — no per-actor polymorphism (unlike chunk 5
// Presence).
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Sighted, transitions),
	}
}

// State returns the current state. Safe from any goroutine; nil-safe
// (returns Sighted on a nil receiver).
func (m *Machine) State() State {
	if m == nil || m.inner == nil {
		return Sighted
	}
	return m.inner.State()
}

// SetSelf binds the machine to its owning ActorRef.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// TransitionTo moves the machine to the target state. Returns
// ErrInvalidTransition if the transition isn't allowed by the table
// (e.g., Blinded→Blinded). Errors are non-fatal and discardable at
// most call sites — re-applying a buff while already Blinded is a
// no-op, not an error.
func (m *Machine) TransitionTo(to State, r state.TransitionReason) error {
	if m == nil || m.inner == nil {
		return nil
	}
	return m.inner.TransitionTo(to, r)
}

// RegisterVeto installs a veto on transitions from `from` to `to`.
// Mirrors the chunk-5 presence.Machine wrapper. Unused in chunk 6
// (no vetoes registered); future messaging framework may use this.
func (m *Machine) RegisterVeto(from, to State, veto func(state.TransitionReason) error) {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.BeforeTransition(
		"veto_"+from.String()+"_to_"+to.String(),
		func(f, t State, r state.TransitionReason) error {
			if f == from && t == to {
				return veto(r)
			}
			return nil
		},
	)
}

// RegisterObserver installs an observer fired after every transition.
// Mirrors the chunk-5 presence.Machine wrapper. Unused in chunk 6;
// future messaging framework may use this for broadcast routing.
func (m *Machine) RegisterObserver(name string, obs func(from, to State, r state.TransitionReason)) {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.Subscribe(name, obs)
}

// Inner returns the underlying state.Machine. Available for any
// consumer that needs direct access to the raw framework machine.
// Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// CancelScheduled cancels all pending scheduled transitions on this
// Perception machine. Called by Character.CancelAllScheduled on
// terminal-state entry (per chunk 5 T8 cascade pattern).
func (m *Machine) CancelScheduled() {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.CancelScheduled()
}
