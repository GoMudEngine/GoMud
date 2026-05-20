// Package activity defines the Activity state machine — the fourth
// consumer of internal/state, after combatphase, awareness, life.
// It replaces Character.CastingState and Character.CraftingState
// pointer fields with a single state machine carrying per-state
// data, and cleans up the salvage-hijacks-crafting-slot pattern.
package activity

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Activity state enum.
type State int

const (
	Free State = iota
	Casting
	Crafting
	Salvaging
)

// String for logging / debugging.
func (s State) String() string {
	switch s {
	case Free:
		return "Free"
	case Casting:
		return "Casting"
	case Crafting:
		return "Crafting"
	case Salvaging:
		return "Salvaging"
	}
	return "Unknown"
}

// FreeData is empty — the default state.
type FreeData struct{}

// CastingData replaces internal/characters/CastingState. Field
// shape preserved verbatim so existing consumers (per-tick spell
// resolution, prompt rendering) only need to swap accessor.
type CastingData struct {
	Reason               state.TransitionReason
	SpellId              string
	FoldsNeeded          int
	FoldsAccumulated     int
	FoldsPerRound        int
	TotalConvictionCost  int
	ConvictionSpent      int
	TargetUserIds        []int
	TargetMobInstanceIds []int
	SpellRest            string
}

// CraftingData replaces internal/characters/CraftingState.
type CraftingData struct {
	Reason         state.TransitionReason
	RecipeId       string
	RoundsTotal    int
	RoundsComplete int
	TargetSlot     string
}

// SalvagingData is new — gives salvage its own state with its
// own data instead of hijacking CraftingState + MiscData.
type SalvagingData struct {
	Reason         state.TransitionReason
	ItemUuid       string
	RoundsTotal    int
	RoundsComplete int
	SpoiledPotion  bool
}

// Machine wraps state.Machine[State] with Activity-specific API.
type Machine struct {
	inner     *state.Machine[State]
	casting   *CastingData
	crafting  *CraftingData
	salvaging *SalvagingData
	self      state.ActorRef
}

// NewMachine returns an Activity machine in Free.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Free, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// IsFree returns true when state is Free.
func (m *Machine) IsFree() bool { return m.State() == Free }

// IsCasting returns true when state is Casting.
func (m *Machine) IsCasting() bool { return m.State() == Casting }

// IsCrafting returns true when state is Crafting.
func (m *Machine) IsCrafting() bool { return m.State() == Crafting }

// IsSalvaging returns true when state is Salvaging.
func (m *Machine) IsSalvaging() bool { return m.State() == Salvaging }

// IsActing returns true for any non-Free state.
func (m *Machine) IsActing() bool { return m.State() != Free }

// CastingData returns the casting context if currently Casting.
func (m *Machine) CastingData() (CastingData, bool) {
	if m.State() != Casting || m.casting == nil {
		return CastingData{}, false
	}
	return *m.casting, true
}

// CraftingData returns the crafting context if currently Crafting.
func (m *Machine) CraftingData() (CraftingData, bool) {
	if m.State() != Crafting || m.crafting == nil {
		return CraftingData{}, false
	}
	return *m.crafting, true
}

// SalvagingData returns the salvaging context if currently Salvaging.
func (m *Machine) SalvagingData() (SalvagingData, bool) {
	if m.State() != Salvaging || m.salvaging == nil {
		return SalvagingData{}, false
	}
	return *m.salvaging, true
}

// Inner returns the underlying state.Machine. Used by rules.go
// (Task 3) and hooks (Task 5+). Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// === Machine registry ===
// Cross-character lookups (parity with combatphase/awareness/life).

var (
	registryMu      sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

// RegisterMachine binds an ActorRef to its Machine.
func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
	m.SetSelf(ref)
}

// UnregisterMachine removes a binding.
func UnregisterMachine(ref state.ActorRef) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(machineRegistry, ref)
}
