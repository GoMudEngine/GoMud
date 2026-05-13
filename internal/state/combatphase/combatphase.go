// Package combatphase defines the Combat Phase state machine,
// the first consumer of internal/state. It replaces the
// Character.Aggro field as the source of truth for "who am I
// attacking?" and "am I in combat?".
package combatphase

import (
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
	inner                    *state.Machine[State]
	self                     state.ActorRef // own identity, set by RegisterMachine
	engaging                 *EngagingData
	engaged                  *EngagedData
	disengaging              *DisengagingData
	attackers                []state.ActorRef // inbound attacker list
	attackersChangeListeners []func([]state.ActorRef)
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
	m.self = ref
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

// === Task 5 implementations ===

// TransitionToEngaging is the primary entry point into combat.
// Vetoes (Task 6) run inside via state.Machine's TransitionTo.
// On success, stores EngagingData and notifies target's inbound list.
func (m *Machine) TransitionToEngaging(d EngagingData, r state.TransitionReason) error {
	// (Vetoes registered via Register*Veto/Check land in Task 6;
	// they wire into the inner Machine's BeforeTransition hook.)
	if err := m.inner.TransitionTo(Engaging, r); err != nil {
		return err
	}
	m.engaging = &d
	if target := lookupMachine(d.Target); target != nil {
		target.RecordInboundAttacker(r.Actor)
	}
	return nil
}

// TransitionToDisengaging starts a flee/disengage attempt.
// Veto for grappled state (Task 6) runs inside.
func (m *Machine) TransitionToDisengaging(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Disengaging, r); err != nil {
		return err
	}
	// Capture last target from EngagedData if present.
	target := state.ActorRef{}
	if m.engaged != nil {
		target = m.engaged.Target
	}
	m.disengaging = &DisengagingData{LastTarget: target}
	return nil
}

// OnRoundTick advances per-state round counters and fires any
// state transitions whose round has come. Called once per round
// per character by the round driver.
func (m *Machine) OnRoundTick() {
	switch m.State() {
	case Engaging:
		if m.engaging == nil {
			return
		}
		if m.engaging.RoundsUntil <= 0 {
			m.advanceToEngaged()
			return
		}
		m.engaging.RoundsUntil--
		if m.engaging.RoundsUntil == 0 {
			m.advanceToEngaged()
		}
	case Engaged:
		// Combat resolution is driven by the round driver, not here.
	case Disengaging:
		// Flee resolution is driven externally via ResolveFlee.
	}
}

// advanceToEngaged transitions Engaging → Engaged, carrying
// EngagingData into EngagedData. Surprise marker persists.
func (m *Machine) advanceToEngaged() {
	prevEngaging := m.engaging
	if prevEngaging == nil {
		return
	}
	if err := m.inner.TransitionTo(Engaged, state.TransitionReason{
		Trigger: TriggerEngagementReady,
		Target:  prevEngaging.Target,
	}); err != nil {
		return // shouldn't happen — invariant violation
	}
	m.engaged = &EngagedData{
		Target:       prevEngaging.Target,
		SurpriseLeft: prevEngaging.Reason.Trigger == TriggerSurpriseAttack,
	}
	m.engaging = nil
}

// ResolveFlee finalizes a Disengaging state. Success → Idle.
// Failure → back to Engaged.
func (m *Machine) ResolveFlee(success bool) {
	if m.State() != Disengaging {
		return
	}
	if success {
		m.ForceIdle(state.TransitionReason{Trigger: TriggerFleeSuccess})
		return
	}
	// Failure: restore Engaged with the last target.
	target := state.ActorRef{}
	if m.disengaging != nil {
		target = m.disengaging.LastTarget
	}
	if err := m.inner.TransitionTo(Engaged, state.TransitionReason{
		Trigger: TriggerFleeFailure,
	}); err != nil {
		return
	}
	m.engaged = &EngagedData{Target: target}
	m.disengaging = nil
}

// NotifyTargetDied is invoked by the dying target's Machine
// during its own ForceIdle/death cascade. If my current target
// matches, I transition to Idle.
func (m *Machine) NotifyTargetDied(target state.ActorRef) {
	if m.CurrentTarget() != target {
		return
	}
	m.ForceIdle(state.TransitionReason{
		Trigger: TriggerTargetDied,
		Target:  target,
	})
}

// NotifySelfDied is invoked when this character dies. Clears
// own outbound combat state, clears all inbound attackers
// (notifying them that their target is gone).
func (m *Machine) NotifySelfDied() {
	// Capture inbound attackers before clearing.
	attackers := append([]state.ActorRef{}, m.attackers...)

	// Force self to Idle (also clears state data and notifies
	// my own outbound target if any).
	if m.State() != Idle {
		m.ForceIdle(state.TransitionReason{Trigger: TriggerSelfDied})
	}
	// Explicitly clear inbound list (ForceIdle handles outbound).
	m.attackers = nil
	m.notifyAttackersChange()

	// Notify each inbound attacker that their target died.
	for _, a := range attackers {
		if am := lookupMachine(a); am != nil {
			am.ForceIdle(state.TransitionReason{Trigger: TriggerTargetDied})
		}
	}
}

// ForceIdle transitions to Idle from any state, clearing all
// state-data and removing self from target's inbound list.
// Used for death cascade, Combatant-toggle, target-died, etc.
func (m *Machine) ForceIdle(r state.TransitionReason) {
	switch m.State() {
	case Idle:
		return
	}
	// Capture target before we clear data so we can remove
	// our entry from target's inbound list.
	target := m.CurrentTarget()

	if err := m.inner.TransitionTo(Idle, r); err != nil {
		return
	}
	m.engaging = nil
	m.engaged = nil
	m.disengaging = nil

	// Remove self from target's inbound list. Use r.Actor if set,
	// otherwise fall back to the machine's own registered identity.
	selfRef := r.Actor
	if selfRef.IsZero() {
		selfRef = m.self
	}
	if t := lookupMachine(target); t != nil && !selfRef.IsZero() {
		t.RemoveInboundAttacker(selfRef)
	}
}

// RecordInboundAttacker appends to the inbound attacker list.
// Called via the framework when another character transitions
// to Engaging with us as target. Idempotent — duplicate inserts
// are skipped.
func (m *Machine) RecordInboundAttacker(a state.ActorRef) {
	if a.IsZero() {
		return
	}
	for _, existing := range m.attackers {
		if existing == a {
			return
		}
	}
	m.attackers = append(m.attackers, a)
	m.notifyAttackersChange()
}

// RemoveInboundAttacker drops an attacker from the inbound list.
// Idempotent — removing a non-existent entry is a no-op.
func (m *Machine) RemoveInboundAttacker(a state.ActorRef) {
	for i, existing := range m.attackers {
		if existing == a {
			m.attackers = append(m.attackers[:i], m.attackers[i+1:]...)
			m.notifyAttackersChange()
			return
		}
	}
}

// notifyAttackersChange fires registered SubscribeAttackersChange
// callbacks (the callbacks themselves are registered via a stub
// today; Task 8 implements registration).
func (m *Machine) notifyAttackersChange() {
	for _, fn := range m.attackersChangeListeners {
		fn(m.Attackers())
	}
}

// SubscribeAttackersChange registers a callback that fires whenever the
// inbound Attackers list changes (add or remove).
func (m *Machine) SubscribeAttackersChange(fn func([]state.ActorRef)) {
	m.attackersChangeListeners = append(m.attackersChangeListeners, fn)
}

// === STUBS — Implementations land in Tasks 6-8. ===

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

// OnTickEvent registers a callback that DispatchTickEvent will invoke with
// the state-appropriate event name.
func (m *Machine) OnTickEvent(fn func(name string, r state.TransitionReason)) {}

// DispatchTickEvent fires the registered tick-event callback with the event
// name appropriate to the current state (mob_combat_round, mob_idle, or
// silent for Engaging/Disengaging).
func (m *Machine) DispatchTickEvent() {}
