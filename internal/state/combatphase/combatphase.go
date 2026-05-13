// Package combatphase defines the Combat Phase state machine,
// the first consumer of internal/state. It replaces the
// Character.Aggro field as the source of truth for "who am I
// attacking?" and "am I in combat?".
package combatphase

import (
	"errors"
	"sync"

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

// === Machine registry ===
// Cross-character lookups for inbound attacker tracking, target-death
// cascades, etc. Real engine integration (Task 10) wires this from
// Character setup.

var (
	registryMu      sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

// RegisterMachine binds an ActorRef to its Machine.
func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
}

// UnregisterMachine removes a binding (e.g. on logout or despawn).
func UnregisterMachine(ref state.ActorRef) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(machineRegistry, ref)
}

// lookupMachine returns the registered Machine for ref, or nil.
func lookupMachine(ref state.ActorRef) *Machine {
	registryMu.Lock()
	defer registryMu.Unlock()
	return machineRegistry[ref]
}

// === STUBS — Implementations land in Tasks 5-8. ===

// errNotImplemented is returned by stubs so tests get an error rather than
// a panic when the real logic hasn't been written yet.
var errNotImplemented = errors.New("not implemented")

// TransitionToEngaging initiates combat against a target.
func (m *Machine) TransitionToEngaging(d EngagingData, r state.TransitionReason) error {
	return errNotImplemented
}

// TransitionToDisengaging initiates a flee attempt from Engaged state.
func (m *Machine) TransitionToDisengaging(r state.TransitionReason) error {
	return errNotImplemented
}

// ResolveFlee completes a flee attempt: success=true → Idle, false → Engaged.
func (m *Machine) ResolveFlee(success bool) {}

// OnRoundTick advances Engaging countdown or fires scheduled transitions.
func (m *Machine) OnRoundTick() {}

// NotifyTargetDied is called when this machine's outbound target has died.
func (m *Machine) NotifyTargetDied(target state.ActorRef) {}

// NotifySelfDied is called when this character has died; clears all combat state.
func (m *Machine) NotifySelfDied() {}

// ForceIdle unconditionally transitions the machine to Idle (e.g. on
// combatant-flag toggle, charm, or despawn).
func (m *Machine) ForceIdle(r state.TransitionReason) {}

// OnEndOfRoundIfSurprise registers a callback that fires once at end of the
// first combat round when this engagement was initiated with TriggerSurpriseAttack.
func (m *Machine) OnEndOfRoundIfSurprise(fn func(state.TransitionReason)) {}

// OnCombatRoundEnd is called at the end of each combat round; advances
// surprise tracking and fires registered end-of-round callbacks.
func (m *Machine) OnCombatRoundEnd() {}

// RegisterCombatantVeto adds a veto that blocks Engaging when the attacker
// is a NonCombatant. check() returns true when combat IS allowed.
func (m *Machine) RegisterCombatantVeto(check func() bool) {}

// RegisterActivityCheck adds a veto that blocks Engaging when the character
// is busy with an activity. check() returns true when free.
func (m *Machine) RegisterActivityCheck(check func() bool) {}

// RegisterLifeCheck adds a veto that blocks Engaging when the attacker
// is dead. check() returns true when alive.
func (m *Machine) RegisterLifeCheck(check func() bool) {}

// RegisterPositionCheck adds a veto that blocks Disengaging when the
// character is grappled. check() returns true when movement is possible.
func (m *Machine) RegisterPositionCheck(check func() bool) {}

// RegisterTargetCombatantCheck adds a veto that blocks Engaging when the
// target is a NonCombatant. check(target) returns true when target can be attacked.
func (m *Machine) RegisterTargetCombatantCheck(check func(state.ActorRef) bool) {}

// RegisterTargetLifeCheck adds a veto that blocks Engaging when the target
// is dead. check(target) returns true when alive.
func (m *Machine) RegisterTargetLifeCheck(check func(state.ActorRef) bool) {}

// RegisterTargetPresenceCheck adds a veto that blocks Engaging when the
// target is AFK or disconnected. check(target) returns true when present.
func (m *Machine) RegisterTargetPresenceCheck(check func(state.ActorRef) bool) {}

// SubscribeAttackersChange registers a callback that fires whenever the
// inbound Attackers list changes (add or remove).
func (m *Machine) SubscribeAttackersChange(fn func([]state.ActorRef)) {}

// OnTickEvent registers a callback that DispatchTickEvent will invoke with
// the state-appropriate event name.
func (m *Machine) OnTickEvent(fn func(name string, r state.TransitionReason)) {}

// DispatchTickEvent fires the registered tick-event callback with the event
// name appropriate to the current state (mob_combat_round, mob_idle, or
// silent for Engaging/Disengaging).
func (m *Machine) DispatchTickEvent() {}
