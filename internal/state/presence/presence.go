// Package presence defines the Presence state machine — the sixth
// consumer of internal/state, after combatphase, awareness, life,
// activity, and position. It centralizes "is this character meaningfully
// present?" into one canonical FSM with per-actor state lists encoded
// in a single union enum.
//
// Player states: Connecting / Active / Idle / AFK / Disconnected.
// Mob states:    Spawning / Active / Dormant / Despawning.
// Active is shared between actors; transitions are per-actor (two
// transition tables, one constructor per actor type).
//
// Sunsets: ManualAFK, AFKMessage (UserRecord); BoredomCounter, PreventIdle
// (Mob); MaxMobBoredom (config).
package presence

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Presence state enum. Per-actor polymorphism is enforced
// by the transition tables, not by the type system — both actor types
// share the same enum so Character only carries one *Machine pointer.
type State int

const (
	Active       State = iota // both actors — normal "in world, ticking"
	Connecting                // player-only — logged in but not yet in a room
	Idle                      // player-only — no input for N rounds
	AFK                       // player-only — no input for M rounds OR manual cmd
	Disconnected              // player-only — TCP gone, character in graveyard
	Spawning                  // mob-only — freshly created, auto → Active next tick
	Dormant                   // mob-only — bored OR zone has no nearby players
	Despawning                // mob-only — scheduled for removal next tick
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Active:
		return "Active"
	case Connecting:
		return "Connecting"
	case Idle:
		return "Idle"
	case AFK:
		return "AFK"
	case Disconnected:
		return "Disconnected"
	case Spawning:
		return "Spawning"
	case Dormant:
		return "Dormant"
	case Despawning:
		return "Despawning"
	}
	return "Unknown"
}

// AFKData is the only state-data struct. Carried on AFK-state entry
// from either the manual `afk <message>` command (Manual=true) or
// the timeout transition (Manual=false, Message="").
type AFKData struct {
	Message string // set by manual `afk <message>`; empty for auto-AFK
	Manual  bool   // distinguishes manual vs timeout
}

// Machine wraps state.Machine[State] with Presence-specific API.
// Per-state data is stored alongside the inner machine. Only AFK
// state carries data; other states are stateless.
type Machine struct {
	inner   *state.Machine[State]
	afkData *AFKData
	self    state.ActorRef
}

// NewPlayerPresence returns a Machine in Connecting state with the
// player transition table. Used by characters.New() for player
// characters.
func NewPlayerPresence() *Machine {
	return &Machine{
		inner: state.NewMachine(Connecting, playerTransitions),
	}
}

// NewMobPresence returns a Machine in Spawning state with the mob
// transition table. Used by mobs.Mob.Validate() after a fresh shallow
// copy via newMobByIdInternal.
func NewMobPresence() *Machine {
	return &Machine{
		inner: state.NewMachine(Spawning, mobTransitions),
	}
}

// State returns the current state. Safe from any goroutine.
func (m *Machine) State() State {
	if m == nil || m.inner == nil {
		return Active
	}
	return m.inner.State()
}

// AFKData returns the AFK state-data (Message + Manual flag) if the
// machine is currently in AFK. Returns (zero, false) otherwise.
func (m *Machine) AFKData() (AFKData, bool) {
	if m == nil || m.State() != AFK || m.afkData == nil {
		return AFKData{}, false
	}
	return *m.afkData, true
}

// SetSelf binds the machine to its owning ActorRef. Called from
// engine integration when the machine is attached to a Character.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// TransitionTo moves the machine to the target state. For AFK
// transitions that should carry data, use TransitionToAFK instead.
func (m *Machine) TransitionTo(to State, r state.TransitionReason) error {
	if m == nil || m.inner == nil {
		return nil
	}
	// Clear AFK data when leaving AFK.
	if m.State() == AFK && to != AFK {
		m.afkData = nil
	}
	return m.inner.TransitionTo(to, r)
}

// TransitionToAFK is a convenience for AFK with data.
//
// Note: AFK→AFK is intentionally excluded from the player transition
// table. If a future change adds it as a self-transition, the data-swap
// semantics in this method (set m.afkData AFTER inner.TransitionTo)
// will need to clear the previous AFKData explicitly.
func (m *Machine) TransitionToAFK(d AFKData, r state.TransitionReason) error {
	if m == nil || m.inner == nil {
		return nil
	}
	if err := m.inner.TransitionTo(AFK, r); err != nil {
		return err
	}
	m.afkData = &d
	return nil
}

// RegisterVeto installs a veto on transitions from `from` to `to`.
// The veto function receives the TransitionReason and returns a non-nil
// error to block the transition. Used by the engine to gate
// Active→Dormant/Despawning on IsEssential() + IsCharmed().
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
// Used for terminal-state scheduler cleanup.
func (m *Machine) RegisterObserver(name string, obs func(from, to State, r state.TransitionReason)) {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.Subscribe(name, obs)
}

// Inner returns the underlying state.Machine. Used by
// Character.CancelAllScheduled (T8) and any hook that needs direct
// access to the raw machine. Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// CancelScheduled cancels all pending scheduled transitions on this
// Presence machine. Called by Character.CancelAllScheduled on
// terminal-state entry (Disconnected for players, Despawning for mobs).
func (m *Machine) CancelScheduled() {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.CancelScheduled()
}
