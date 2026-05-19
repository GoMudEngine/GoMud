// Package control defines the ControlLevel state machine —
// per-character dominance tracking within a grapple.
//
// Chunk 4b-fixup-2: restores ControlLevel as a proper FSM
// (replacing the broken `IsControllerRole bool` from chunk
// 4b-fixup) with 5 states — 3 stable (Controlling, Neutral,
// Controlled) + 2 transient (LosingControl, BecomingControlled).
// Transient states are entered same-tick during boundary crossings
// and immediately resolve to the target stable state, mirroring
// the awareness package's Hidden → Revealing → Visible pattern.
//
// Boundary-cross events fire registered callbacks for gradient
// flavor messaging (wired in chunk 4b-fixup-2 T13).
package control

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the ControlLevel state enum.
type State int

const (
	Neutral            State = iota // Stable
	Controlling                     // Stable
	LosingControl                   // Transient (Controlling ↔ Neutral boundary)
	BecomingControlled              // Transient (Neutral ↔ Controlled boundary)
	Controlled                      // Stable
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Neutral:
		return "Neutral"
	case Controlling:
		return "Controlling"
	case LosingControl:
		return "LosingControl"
	case BecomingControlled:
		return "BecomingControlled"
	case Controlled:
		return "Controlled"
	}
	return "Unknown"
}

// Machine wraps state.Machine[State] with ControlLevel-specific API.
type Machine struct {
	inner *state.Machine[State]
	self  state.ActorRef
}

// NewMachine returns a ControlLevel machine in Neutral.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Neutral, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// SetSelf binds the machine to its owning ActorRef.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }
