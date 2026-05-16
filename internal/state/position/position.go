// Package position defines the Position state machine — the fifth
// consumer of internal/state, after combatphase, awareness, life,
// and activity. Models body position + grapple geometry using the
// full BJJ/MMA position taxonomy (14 states) plus a per-grappler
// control axis (ControlLevel) stored as data on grapple states.
//
// Chunk 4a ships this scaffold DORMANT — no production code
// transitions the machine. 4b cuts over command-site writers.
package position

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Position state enum.
type State int

const (
	Standing State = iota
	Prone        // face-down knockdown, alone
	Supine       // face-up knockdown, alone
	Clinch       // standing grapple, both upright
	BackStanding // standing grapple, one has back of other
	Mount        // ground top-dominant, controller on opponent's chest
	SideControl  // ground top-dominant, perpendicular pin
	KneeOnBelly  // ground top-dominant, knee crushing pin
	NorthSouth   // ground top-dominant, head-to-toe
	Crucifix     // ground top-dominant, opponent's arms isolated
	BackGround   // ground top-dominant, rear mount on ground
	HalfGuard    // ground transitional, one leg trapped
	Guard        // ground bottom-active, legs around opponent
	Turtle       // ground defensive, curled exposing back
)

// String for logging / debugging.
func (s State) String() string {
	switch s {
	case Standing:
		return "Standing"
	case Prone:
		return "Prone"
	case Supine:
		return "Supine"
	case Clinch:
		return "Clinch"
	case BackStanding:
		return "BackStanding"
	case Mount:
		return "Mount"
	case SideControl:
		return "SideControl"
	case KneeOnBelly:
		return "KneeOnBelly"
	case NorthSouth:
		return "NorthSouth"
	case Crucifix:
		return "Crucifix"
	case BackGround:
		return "BackGround"
	case HalfGuard:
		return "HalfGuard"
	case Guard:
		return "Guard"
	case Turtle:
		return "Turtle"
	}
	return "Unknown"
}

// ControlLevel is the per-grappler control axis. Stored on
// GrappleData. 4a defaults to Neutral on transition entry; 4b
// adds per-round opposed rolls that shift it.
type ControlLevel int

// Neutral is iota=0 so the zero value of a GrappleData{} literal
// defaults to Neutral (matches the spec — control rolls drive
// shifts away from Neutral in 4b). Display ordering in narrative
// text still follows the "in control → losing → neutral →
// becoming → controlled" gradient; this enum order is purely a
// zero-value convenience.
const (
	Neutral ControlLevel = iota
	InControl
	LosingControl
	BecomingControlled
	Controlled
)

func (c ControlLevel) String() string {
	switch c {
	case InControl:
		return "InControl"
	case LosingControl:
		return "LosingControl"
	case Neutral:
		return "Neutral"
	case BecomingControlled:
		return "BecomingControlled"
	case Controlled:
		return "Controlled"
	}
	return "Unknown"
}

// StandingData — empty (default state, no payload).
type StandingData struct{}

// ProneData — face-down knockdown, alone. Distinct from Supine
// because submission paths, recovery difficulty, and back-take
// vulnerability all differ.
type ProneData struct {
	Reason            state.TransitionReason
	MinRecoveryRounds int            // replaces legacy PositionRoundsMin; 0 = can stand immediately
	KnockdownSource   state.ActorRef // who knocked us down
}

// SupineData — face-up knockdown, alone. Same shape as ProneData
// today; split because mechanics diverge (Supine can pull guard,
// recovery is easier).
type SupineData struct {
	Reason            state.TransitionReason
	MinRecoveryRounds int
	KnockdownSource   state.ActorRef
}

// GrappleData — shared across all 11 grapple states. 4a does not
// introduce per-state extras (ClinchGrip, ArmsIsolated, HooksIn,
// TrappedLeg, GuardVariant). 4b/4c add state-specific wrapping
// structs when consumers materialize.
type GrappleData struct {
	Reason       state.TransitionReason
	Partner      state.ActorRef // zero only for solo Turtle
	ControlLevel ControlLevel   // default Neutral; 4b drives changes
}

// Machine wraps state.Machine[State] with Position-specific API.
type Machine struct {
	inner   *state.Machine[State]
	prone   *ProneData
	supine  *SupineData
	grapple *GrappleData // shared across all 11 grapple states
	self    state.ActorRef
}

// NewMachine returns a Position machine in Standing.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Standing, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// === Per-state predicates ===

func (m *Machine) IsStanding() bool     { return m.State() == Standing }
func (m *Machine) IsProne() bool        { return m.State() == Prone }
func (m *Machine) IsSupine() bool       { return m.State() == Supine }
func (m *Machine) IsClinch() bool       { return m.State() == Clinch }
func (m *Machine) IsBackStanding() bool { return m.State() == BackStanding }
func (m *Machine) IsMount() bool        { return m.State() == Mount }
func (m *Machine) IsSideControl() bool  { return m.State() == SideControl }
func (m *Machine) IsKneeOnBelly() bool  { return m.State() == KneeOnBelly }
func (m *Machine) IsNorthSouth() bool   { return m.State() == NorthSouth }
func (m *Machine) IsCrucifix() bool     { return m.State() == Crucifix }
func (m *Machine) IsBackGround() bool   { return m.State() == BackGround }
func (m *Machine) IsHalfGuard() bool    { return m.State() == HalfGuard }
func (m *Machine) IsGuard() bool        { return m.State() == Guard }
func (m *Machine) IsTurtle() bool       { return m.State() == Turtle }

// === Rollup predicates ===

// IsGrappling returns true for any of the 11 grapple states
// (Clinch through Turtle).
func (m *Machine) IsGrappling() bool {
	s := m.State()
	return s == Clinch || s == BackStanding || s == Mount || s == SideControl ||
		s == KneeOnBelly || s == NorthSouth || s == Crucifix || s == BackGround ||
		s == HalfGuard || s == Guard || s == Turtle
}

// IsStandingGrapple returns true for Clinch or BackStanding.
func (m *Machine) IsStandingGrapple() bool {
	s := m.State()
	return s == Clinch || s == BackStanding
}

// IsGroundGrapple returns true for any of the 9 ground grapple
// states (Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
// BackGround, HalfGuard, Guard, Turtle).
func (m *Machine) IsGroundGrapple() bool {
	s := m.State()
	return s == Mount || s == SideControl || s == KneeOnBelly || s == NorthSouth ||
		s == Crucifix || s == BackGround || s == HalfGuard || s == Guard || s == Turtle
}

// IsTopDominant returns true for the 6 controller-dominant ground
// states (Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
// BackGround).
func (m *Machine) IsTopDominant() bool {
	s := m.State()
	return s == Mount || s == SideControl || s == KneeOnBelly || s == NorthSouth ||
		s == Crucifix || s == BackGround
}

// IsOnFloor returns true for Prone, Supine, or any ground grapple.
func (m *Machine) IsOnFloor() bool {
	return m.IsProne() || m.IsSupine() || m.IsGroundGrapple()
}

// IsController returns true when the character is the controller
// side of a grapple pair. False for Standing, Prone, Supine, and
// for grapple states where the character's ControlLevel is on the
// controlled side or neutral (symmetric positions).
//
// Used by the per-round tick to identify which side of a pair to
// iterate from; used by btree primitives for AI decisions.
func (m *Machine) IsController() bool {
	if !m.IsGrappling() {
		return false
	}
	d, ok := m.GrappleData()
	if !ok {
		return false
	}
	return IsControllerLevel(d.ControlLevel)
}

// IsBeingControlled returns true when the character is the
// controlled side of a grapple pair.
func (m *Machine) IsBeingControlled() bool {
	if !m.IsGrappling() {
		return false
	}
	d, ok := m.GrappleData()
	if !ok {
		return false
	}
	return IsControlledLevel(d.ControlLevel)
}

// === Data accessors ===

// ProneData returns the prone context if currently Prone.
func (m *Machine) ProneData() (ProneData, bool) {
	if m.State() != Prone || m.prone == nil {
		return ProneData{}, false
	}
	return *m.prone, true
}

// SupineData returns the supine context if currently Supine.
func (m *Machine) SupineData() (SupineData, bool) {
	if m.State() != Supine || m.supine == nil {
		return SupineData{}, false
	}
	return *m.supine, true
}

// GrappleData returns the grapple context if currently in any
// grapple state.
func (m *Machine) GrappleData() (GrappleData, bool) {
	if !m.IsGrappling() || m.grapple == nil {
		return GrappleData{}, false
	}
	return *m.grapple, true
}

// MutateGrappleControlLevel updates the ControlLevel on the
// current GrappleData WITHOUT firing a transition. Used by the
// per-round drift hook in Position_GrappleTick.go — the FSM
// transition table forbids Mount→Mount, so per-round
// re-transitions aren't viable. This is the ONE place outside
// transition methods that mutates per-state data.
//
// No-op if the machine is not in a grapple state.
func (m *Machine) MutateGrappleControlLevel(newLevel ControlLevel) {
	if !m.IsGrappling() || m.grapple == nil {
		return
	}
	m.grapple.ControlLevel = newLevel
}

// ConsumeRecoveryRound decrements MinRecoveryRounds on the current
// ProneData or SupineData WITHOUT firing a transition. Called by
// AttemptRecovery once per round while the character is still
// gated by the minimum-recovery window. Mirrors
// MutateGrappleControlLevel — per-state data mutation outside the
// transition methods.
//
// No-op if not in Prone or Supine or if MinRecoveryRounds is
// already zero.
func (m *Machine) ConsumeRecoveryRound() {
	if m.IsProne() && m.prone != nil && m.prone.MinRecoveryRounds > 0 {
		m.prone.MinRecoveryRounds--
		return
	}
	if m.IsSupine() && m.supine != nil && m.supine.MinRecoveryRounds > 0 {
		m.supine.MinRecoveryRounds--
	}
}

// === Framework escape hatches ===

// Inner returns the underlying state.Machine — used by rules.go
// (Task 3) and hooks (Task 5+). Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// === Machine registry ===
// Cross-character lookups for grapple-pair writes (4b+).

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

// lookupMachine returns the registered Machine for ref, or nil.
func lookupMachine(ref state.ActorRef) *Machine {
	registryMu.Lock()
	defer registryMu.Unlock()
	return machineRegistry[ref]
}
