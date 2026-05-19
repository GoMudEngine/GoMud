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

// boundaryCrossCallback is invoked when a transition crosses one of
// the two boundary states (LosingControl or BecomingControlled). The
// `transient` argument is the transient state crossed; `from` and `to`
// are the stable states on either side of the transition (so messaging
// can pick the right flavor by direction). Wired to grapplemessaging
// in chunk 4b-fixup-2 T13.
type boundaryCrossCallback func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason)

var (
	registeredCallback boundaryCrossCallback
)

// RegisterBoundaryCrossCallback sets the global callback invoked
// during same-tick boundary crossings. Set to nil to disable.
// Caller is responsible for ensuring the callback is safe to call
// from any goroutine that runs a transition (typically called once
// from grapplemessaging package init).
func RegisterBoundaryCrossCallback(cb boundaryCrossCallback) {
	registeredCallback = cb
}

func fireBoundaryCross(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
	if registeredCallback != nil {
		registeredCallback(self, transient, from, to, r)
	}
}

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

// TransitionToControlling moves the machine to Controlling, traversing
// any necessary transient states same-tick.
//   - From Neutral: Neutral → LosingControl → Controlling (1 boundary)
//   - From Controlled: Controlled → BecomingControlled → Neutral →
//     LosingControl → Controlling (2 boundaries)
//   - From Controlling: no-op
func (m *Machine) TransitionToControlling(r state.TransitionReason) error {
	current := m.State()
	if current == Controlling {
		return nil
	}
	return m.traverse(current, Controlling, r)
}

// TransitionToNeutral moves the machine to Neutral, traversing any
// necessary transient states same-tick.
//   - From Controlling: Controlling → LosingControl → Neutral
//   - From Controlled: Controlled → BecomingControlled → Neutral
//   - From Neutral: no-op
func (m *Machine) TransitionToNeutral(r state.TransitionReason) error {
	current := m.State()
	if current == Neutral {
		return nil
	}
	return m.traverse(current, Neutral, r)
}

// TransitionToControlled moves the machine to Controlled, traversing
// any necessary transient states same-tick.
//   - From Neutral: Neutral → BecomingControlled → Controlled
//   - From Controlling: Controlling → LosingControl → Neutral →
//     BecomingControlled → Controlled
//   - From Controlled: no-op
func (m *Machine) TransitionToControlled(r state.TransitionReason) error {
	current := m.State()
	if current == Controlled {
		return nil
	}
	return m.traverse(current, Controlled, r)
}

// traverse executes the same-tick chain of transitions from current
// to target. Crosses the upper boundary via LosingControl and the
// lower boundary via BecomingControlled, firing the callback for each.
func (m *Machine) traverse(current, target State, r state.TransitionReason) error {
	// Determine direction.
	currentRank := stableRank(current)
	targetRank := stableRank(target)
	if currentRank == targetRank {
		return nil
	}

	// Step up or down through stable states, crossing transient
	// states between them.
	step := 1
	if targetRank < currentRank {
		step = -1
	}

	for currentRank != targetRank {
		nextRank := currentRank + step
		nextStable := stableFromRank(nextRank)

		// Determine which boundary to cross.
		var transient State
		if (currentRank == 0 && nextRank == 1) || (currentRank == 1 && nextRank == 0) {
			// Crossing upper boundary (Controlling ↔ Neutral)
			transient = LosingControl
		} else if (currentRank == 1 && nextRank == 2) || (currentRank == 2 && nextRank == 1) {
			// Crossing lower boundary (Neutral ↔ Controlled)
			transient = BecomingControlled
		}

		// Transition: stable → transient → next stable, all same-tick.
		if err := m.inner.TransitionTo(transient, r); err != nil {
			return err
		}
		fromState := stableFromRank(currentRank)
		fireBoundaryCross(m.self, transient, fromState, nextStable, r)
		if err := m.inner.TransitionTo(nextStable, r); err != nil {
			return err
		}

		currentRank = nextRank
	}
	return nil
}

// stableRank maps the 3 stable states to a linear gradient rank:
// 0 = Controlling (best for self), 1 = Neutral, 2 = Controlled (worst).
// Returns -1 for transient states (shouldn't be called for them).
func stableRank(s State) int {
	switch s {
	case Controlling:
		return 0
	case Neutral:
		return 1
	case Controlled:
		return 2
	}
	return -1
}

// stableFromRank is the inverse of stableRank.
func stableFromRank(rank int) State {
	switch rank {
	case 0:
		return Controlling
	case 1:
		return Neutral
	case 2:
		return Controlled
	}
	return Neutral
}
