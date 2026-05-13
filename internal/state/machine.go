package state

import "sync"

// Machine is a generic finite state machine instance.
// Type parameter S is the state enum (typically an int-based
// type alias defined in the consumer package).
//
// Machines are intended to live as fields on Character; one
// Machine per concern (CombatPhase, Activity, Position,
// Awareness, Life, Presence).
//
// Machines are NOT safe for concurrent use by themselves —
// the engine's per-character lock should guard transitions
// (consistent with existing Character mutation patterns).
type Machine[S comparable] struct {
	current S
	table   TransitionTable[S]

	vetoes    []vetoEntry[S]
	cascades  []cascadeEntry[S]
	observers []observerEntry[S]

	cancelTokens []*bool // populated by Task 2 (scheduled.go); declared here

	mu sync.Mutex // serializes transitions within a single character
}

type vetoEntry[S comparable] struct {
	name    string
	handler VetoHandler[S]
}

type cascadeEntry[S comparable] struct {
	name    string
	handler CascadeHandler[S]
}

type observerEntry[S comparable] struct {
	name    string
	handler ObserverHandler[S]
}

// NewMachine constructs a Machine with the given initial state
// and valid-transitions table.
func NewMachine[S comparable](initial S, table TransitionTable[S]) *Machine[S] {
	return &Machine[S]{current: initial, table: table}
}

// State returns the current state.
func (m *Machine[S]) State() S {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// BeforeTransition registers a veto handler. name is used
// for debugging — the returned VetoError carries it.
func (m *Machine[S]) BeforeTransition(name string, handler VetoHandler[S]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vetoes = append(m.vetoes, vetoEntry[S]{name: name, handler: handler})
}

// AfterTransition registers a cascade handler.
func (m *Machine[S]) AfterTransition(name string, handler CascadeHandler[S]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cascades = append(m.cascades, cascadeEntry[S]{name: name, handler: handler})
}

// Subscribe registers an observer.
func (m *Machine[S]) Subscribe(name string, handler ObserverHandler[S]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers = append(m.observers, observerEntry[S]{name: name, handler: handler})
}

// CanTransition returns nil if a transition from current state
// to `to` would be allowed: the transition table accepts it and
// no veto handler returns an error. Does not mutate.
func (m *Machine[S]) CanTransition(to S, reason TransitionReason) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.canTransitionLocked(to, reason)
}

func (m *Machine[S]) canTransitionLocked(to S, reason TransitionReason) error {
	if !m.table.IsAllowed(m.current, to) {
		return ErrInvalidTransition
	}
	for _, v := range m.vetoes {
		if err := v.handler(m.current, to, reason); err != nil {
			return &VetoError{HandlerName: v.name, Reason: err.Error()}
		}
	}
	return nil
}

// TransitionTo attempts the transition. On success:
//  1. State changes.
//  2. All cascade handlers fire in registration order.
//  3. All observer handlers fire in registration order.
//
// On veto: returns the veto error; no state change, no cascades.
// On invalid transition (not in table): returns ErrInvalidTransition.
func (m *Machine[S]) TransitionTo(to S, reason TransitionReason) error {
	m.mu.Lock()

	if err := m.canTransitionLocked(to, reason); err != nil {
		m.mu.Unlock()
		return err
	}

	from := m.current
	m.current = to

	// Snapshot handlers under lock, then run without it.
	// Cascades may call TransitionTo on this or other machines;
	// holding the lock would deadlock.
	cascades := append([]cascadeEntry[S]{}, m.cascades...)
	observers := append([]observerEntry[S]{}, m.observers...)
	m.mu.Unlock()

	for _, c := range cascades {
		c.handler(from, to, reason)
	}
	for _, o := range observers {
		o.handler(from, to, reason)
	}
	return nil
}
