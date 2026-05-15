// Package awareness defines the Awareness state machine —
// the second consumer of internal/state, after combatphase.
// It replaces the buff-#9 "Hidden flag" as the canonical
// source of "is this character hidden?" Buff #9 stays as the
// side-effect carrier (stat mods, room broadcast text); the
// Awareness machine drives its addition and removal via
// cascade handlers.
package awareness

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Awareness state enum.
type State int

const (
	Visible State = iota
	Concealing
	Hidden
	Revealing
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Visible:
		return "Visible"
	case Concealing:
		return "Concealing"
	case Hidden:
		return "Hidden"
	case Revealing:
		return "Revealing"
	}
	return "Unknown"
}

// VisibleData is empty — default state has no per-state data.
type VisibleData struct{}

// ConcealingData captures an in-flight sneak attempt.
// Today synchronous; chunk-1 sets and clears in one call.
// Future multi-round concealment could populate RoundsUntil.
type ConcealingData struct {
	RoundsUntil int
}

// HiddenData carries hidden-state metadata.
// Today empty; reserved for future light-source tracking or
// per-observer awareness lists.
type HiddenData struct{}

// RevealingData captures the in-flight reveal cascade.
// Reason carries context for subscribers ("why is this character
// being revealed?"). Lifetime is one cascade cycle.
type RevealingData struct {
	Reason state.TransitionReason
}

// Machine wraps state.Machine[State] with awareness-specific
// API including per-state data storage.
type Machine struct {
	inner      *state.Machine[State]
	concealing *ConcealingData
	hidden     *HiddenData
	revealing  *RevealingData
	self       state.ActorRef
}

// NewMachine returns an Awareness machine in Visible.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Visible, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// IsHidden returns true when state is Hidden.
func (m *Machine) IsHidden() bool { return m.State() == Hidden }

// ConcealingData returns the in-flight sneak data if Concealing.
func (m *Machine) ConcealingData() (ConcealingData, bool) {
	if m.State() != Concealing || m.concealing == nil {
		return ConcealingData{}, false
	}
	return *m.concealing, true
}

// RevealingData returns the cascade context if Revealing.
func (m *Machine) RevealingData() (RevealingData, bool) {
	if m.State() != Revealing || m.revealing == nil {
		return RevealingData{}, false
	}
	return *m.revealing, true
}

// Inner returns the underlying state.Machine — used by rules.go
// (Task 3+) and hooks (Task 5+). Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef. Called from
// the registry during character creation.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }
