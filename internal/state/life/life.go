// Package life defines the Life state machine — the third
// consumer of internal/state, after combatphase and awareness.
// It replaces scattered death-cleanup logic in suicide.go and
// MobDeath_*.go hooks with a cascade-driven model.
package life

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Life state enum.
type State int

const (
	Alive State = iota
	Dead
	Respawning
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Alive:
		return "Alive"
	case Dead:
		return "Dead"
	case Respawning:
		return "Respawning"
	}
	return "Unknown"
}

// AliveData is empty — default state.
type AliveData struct{}

// DeadData carries killer + damage attribution. Observers
// consume this for kill credit, faction rep, party-share,
// kill stats, etc.
type DeadData struct {
	Reason    state.TransitionReason
	Killer    state.ActorRef
	DamageMap map[int]int // userId → damage; for kill-credit and party-share
}

// RespawningData captures the in-flight respawn cycle.
// Player-only (mobs don't reach Respawning).
type RespawningData struct {
	Reason     state.TransitionReason
	DestRoomId int // graveyard or home room id
}

// Machine wraps state.Machine[State] with Life-specific API.
type Machine struct {
	inner      *state.Machine[State]
	dead       *DeadData
	respawning *RespawningData
	self       state.ActorRef
}

// NewMachine returns a Life machine in Alive.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Alive, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// IsAlive returns true when state is Alive.
func (m *Machine) IsAlive() bool { return m.State() == Alive }

// IsDead returns true when state is Dead.
func (m *Machine) IsDead() bool { return m.State() == Dead }

// IsRespawning returns true when state is Respawning.
func (m *Machine) IsRespawning() bool { return m.State() == Respawning }

// DeadData returns the death context if currently Dead.
func (m *Machine) DeadData() (DeadData, bool) {
	if m.State() != Dead || m.dead == nil {
		return DeadData{}, false
	}
	return *m.dead, true
}

// RespawningData returns the respawn context if currently Respawning.
func (m *Machine) RespawningData() (RespawningData, bool) {
	if m.State() != Respawning || m.respawning == nil {
		return RespawningData{}, false
	}
	return *m.respawning, true
}

// Inner returns the underlying state.Machine — used by rules.go
// (Task 3) and hooks (Task 5+). Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef. Called from
// the registry during character creation.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// === Machine registry ===
// Cross-character lookups for death cascades, party notifications, etc.

var (
	registryMu      sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

// RegisterMachine binds an ActorRef to its Machine for cross-
// character notifications.
func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
	m.SetSelf(ref)
}

// UnregisterMachine removes a binding (player logout, mob despawn).
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

