# Combat State — Chunk 4a: Position FSM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Position state machine (`Standing | Prone | Supine | Clinch | BackStanding | Mount | SideControl | KneeOnBelly | NorthSouth | Crucifix | BackGround | HalfGuard | Guard | Turtle`) on the chunk-0 framework. 14 geometric states, per-state data, ~75-edge transition graph, 19 Character predicates, 10 btree primitives, Life-Dead cascade observer. Ships DORMANT — zero behavior change. All existing position-driven code paths (CombatPosition enum, command writers, recovery, kick variant selector, flee veto, defense degradation) remain unchanged.

**Architecture:** Same generics-based pattern as Combat Phase, Awareness, Life, Activity. Per-state data (StandingData, ProneData, SupineData, shared GrappleData across all 11 grapple states). ControlLevel enum exists as data slot but is not driven in 4a (4b adds the rolls). Cross-machine: a single Life Dead → Position Standing observer coexists with the chunk-2 Life pre-wire (both reach Standing; no drift because the new FSM defaults to Standing and 4a has no writers). The legacy `CombatPosition` enum and all its consumers are untouched — 4a is pure scaffold.

**Tech Stack:** Go 1.21+ with generics, existing `internal/state/` framework, existing `internal/state/combatphase/`, `internal/state/awareness/`, `internal/state/life/`, `internal/state/activity/` machines.

**Spec:** `docs/superpowers/specs/2026-05-16-state-chunk-4a-position-fsm-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

**Doc-scope flag for future sub-chunks:** 4b's writer cutover and 4b/4c/4d/4e's mechanic additions will require updates to many existing `context.md` files (combat, hooks, characters, mobs, behaviortree, actions packages) AND likely helpfiles (player-facing combat help, grapple help, position help). **All of that doc work is OUT OF SCOPE for 4a.** 4a only ships:
- `internal/state/position/context.md` (new package — full docs)
- Thin "Position Machine Integration (chunk 4a — scaffold)" section appended to `internal/characters/context.md`
- Thin "Position Cascade (chunk 4a — scaffold)" section appended to `internal/hooks/context.md`

No audit task in 4a — it's purely additive scaffold with no existing-code touch.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/position/position.go` | NEW | State enum, ControlLevel enum, per-state data structs, Machine wrapper, registry |
| `internal/state/position/transitions.go` | NEW | Valid-transition table (~75 edges), trigger constants |
| `internal/state/position/rules.go` | NEW | Transition method implementations (14 TransitionTo* methods + ForceStanding) |
| `internal/state/position/position_test.go` | NEW | Behavior Matrix tests (PO-001 through PO-045) |
| `internal/state/position/context.md` | NEW | Package documentation |
| `internal/characters/character.go` | MODIFY | Add `Position *position.Machine` field; init in `New()` |
| `internal/characters/validate.go` | MODIFY | Nil-guard init for YAML-loaded chars |
| `internal/characters/position_predicates.go` | NEW | 14 per-state + 5 rollup predicates |
| `internal/characters/context.md` | MODIFY | Append "Position Machine Integration (chunk 4a — scaffold)" section |
| `internal/hooks/Position_Cascades.go` | NEW | Life Dead → Position Standing observer |
| `internal/hooks/Position_Cascades_test.go` | NEW | Integration tests for the cascade |
| `internal/hooks/context.md` | MODIFY | Append "Position Cascade (chunk 4a — scaffold)" section |
| `internal/behaviortree/conditions_position.go` | NEW | 10 btree primitives for position queries |
| `internal/behaviortree/conditions_position_test.go` | NEW | Smoke tests for the primitives |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 4a Done |

---

## Task 1: Position types + Character field

**Files:**
- Create: `internal/state/position/position.go`
- Create: `internal/state/position/transitions.go`
- Modify: `internal/characters/character.go` (Position field, init in `New()`)
- Modify: `internal/characters/validate.go` (nil-guard init)

Foundation. State enum, ControlLevel enum, per-state data structs, Machine wrapper, transition table, trigger constants, Character field bootstrap.

- [ ] **Step 1: Create `internal/state/position/position.go`**

```go
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

const (
	InControl ControlLevel = iota
	LosingControl
	Neutral
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
	inner    *state.Machine[State]
	prone    *ProneData
	supine   *SupineData
	grapple  *GrappleData // shared across all 11 grapple states
	self     state.ActorRef
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
```

- [ ] **Step 2: Create `internal/state/position/transitions.go`**

```go
package position

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enumerates every valid Position transition.
// Star-ish topology around Standing. ~75 edges across 14×14.
// Each row lists the valid SUCCESSOR states from the source state.
var validTransitions = state.TransitionTable[State]{
	Standing: {
		Prone, Supine, Clinch,
	},
	Prone: {
		Standing,
		// Someone mounts the prone target:
		Mount, SideControl, NorthSouth, Crucifix, BackGround,
	},
	Supine: {
		Standing,
		Guard, // pull guard when attacker engages
		// Someone mounts the supine target:
		Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
	},
	Clinch: {
		Standing,
		BackStanding,
		Mount, SideControl, Guard, HalfGuard, BackGround,
		// Clinch → KOB / NorthSouth / Crucifix are NOT direct;
		// reach via SideControl first.
	},
	BackStanding: {
		Standing,
		BackGround, // back-controller pulls down
		Clinch,     // controlled turns to face
	},
	Mount: {
		Standing,
		SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround,
		HalfGuard, Guard, // controlled escapes
	},
	SideControl: {
		Standing,
		Mount, KneeOnBelly, NorthSouth, Crucifix,
		BackGround,
		HalfGuard, Guard, Turtle, // controlled escapes
	},
	KneeOnBelly: {
		Standing,
		Mount, SideControl, NorthSouth,
		HalfGuard, Guard, Turtle, // controlled escapes
	},
	NorthSouth: {
		Standing,
		Mount, SideControl, Crucifix,
		Turtle, // controlled escapes
	},
	Crucifix: {
		Standing,
		SideControl, Mount,
	},
	BackGround: {
		Standing,
		Mount, SideControl,
		Turtle,
	},
	HalfGuard: {
		Standing,
		Guard,
		Mount, SideControl, // top passes
		BackGround, // bottom takes back via sweep
	},
	Guard: {
		Standing,
		HalfGuard,
		Mount, SideControl, NorthSouth, // top passes
		BackGround, // bottom takes back
	},
	Turtle: {
		Standing,
		BackGround, // attacker hooks in
		SideControl, Mount,
	},
}

// Trigger reason constants for Position transitions. 4a names
// every trigger that 4b+ will fire from production code.
const (
	// Knockdowns / falls
	TriggerKnockdownFaceForward  = "knockdown_face_forward"  // → Prone
	TriggerKnockdownFaceBackward = "knockdown_face_backward" // → Supine
	TriggerKnockdownSpell        = "knockdown_spell"         // → Prone or Supine (caller picks)

	// Recovery
	TriggerRecoveryRoll = "recovery_roll" // → Standing (auto, gated by MinRecoveryRounds)
	TriggerStandCommand = "stand_command" // → Standing (explicit, stamina cost, bypasses min)

	// Grapple entry / break
	TriggerGrappleEntry = "grapple_entry" // Standing → Clinch
	TriggerGrappleBreak = "grapple_break" // any grapple → Standing

	// Takedowns from Clinch
	TriggerTakedownMount      = "takedown_mount"
	TriggerTakedownSide       = "takedown_side"
	TriggerTakedownGuardPull  = "takedown_guard_pull"
	TriggerTakedownHalfGuard  = "takedown_half_guard"
	TriggerTakedownBackGround = "takedown_back_ground"

	// Back-takes
	TriggerBackTakeStanding = "back_take_standing" // Clinch → BackStanding
	TriggerBackTakeGround   = "back_take_ground"   // various → BackGround
	TriggerBackPullDown     = "back_pull_down"     // BackStanding → BackGround

	// Controller-initiated transitions within ground subgraph
	TriggerPositionAdvance = "position_advance" // Mount ↔ SC ↔ KOB ↔ NS, etc.

	// Controlled-initiated escapes
	TriggerPositionEscape = "position_escape" // → Standing or up the chain (HalfGuard, Guard)

	// Defensive
	TriggerTurtleDefend = "turtle_defend" // ground state → Turtle
	TriggerGuardPull    = "guard_pull"    // Supine → Guard

	// Opportunistic top-side
	TriggerMountProneTarget = "mount_prone_target" // attacker takes Prone target

	// Arm-isolation
	TriggerArmIsolation = "arm_isolation" // → Crucifix

	// Cascades
	TriggerDeath = "death" // any → Standing (Life cascade)

	// 4b-placeholder (named here so 4a tests can reference; 4a
	// code paths never fire this — 4b implements rolls + thresholds)
	TriggerControlThresholdCrossed = "control_threshold_crossed"
)
```

- [ ] **Step 3: Verify the package builds**

Run:
```bash
go build ./internal/state/position/
```
Expected: clean (rules.go doesn't exist yet, but transitions.go is referenced by validTransitions only and the package can compile without TransitionTo* methods).

- [ ] **Step 4: Modify `internal/characters/character.go` — add the Position field**

Find the existing state-machine fields (look for `Activity *activity.Machine` from chunk 3, alongside `CombatPhase`, `Awareness`, `Life`). Add `Position *position.Machine` next to them. Add the import.

```go
import (
    // ... existing imports
    "github.com/GoMudEngine/GoMud/internal/state/position"
)

// Inside Character struct, alongside other machines:
Position *position.Machine `yaml:"-"`
```

Then find `func New()` (the Character constructor) and initialize the Position machine:

```go
func New() *Character {
    c := &Character{
        // ... existing initializations
        CombatPhase: combatphase.NewMachine(),
        Awareness:   awareness.NewMachine(),
        Life:        life.NewMachine(),
        Activity:    activity.NewMachine(),
        Position:    position.NewMachine(),
        // ... rest
    }
    // ... existing post-init
    return c
}
```

Do NOT touch the legacy `CombatPosition` field or any other existing position-related state. Both coexist throughout chunk 4a.

- [ ] **Step 5: Modify `internal/characters/validate.go` — nil-guard init for YAML-loaded chars**

Find the `Validate()` method. Locate the chunk-3 Activity nil-guard:
```go
if c.Activity == nil {
    c.Activity = activity.NewMachine()
}
```

Add the parallel Position guard after it:
```go
if c.Position == nil {
    c.Position = position.NewMachine()
}
```

Add the import:
```go
"github.com/GoMudEngine/GoMud/internal/state/position"
```

- [ ] **Step 6: Build verify**

```bash
go build ./...
```
Expected: clean. Fix any compile errors (typically missing imports or typos in the new files).

- [ ] **Step 7: Run the existing test suite to confirm no regressions**

```bash
go test ./internal/state/... ./internal/characters/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS. The new field starts at Standing; no production code reads it; behavior is unchanged.

- [ ] **Step 8: Commit**

```bash
git add internal/state/position/position.go \
        internal/state/position/transitions.go \
        internal/characters/character.go \
        internal/characters/validate.go
git commit -m "$(cat <<'EOF'
feat(position): state types, transition table, Character field

Chunk 4a foundation. 14-state enum (Standing/Prone/Supine/Clinch/
BackStanding/Mount/SideControl/KneeOnBelly/NorthSouth/Crucifix/
BackGround/HalfGuard/Guard/Turtle), per-state data structs
(StandingData/ProneData/SupineData/GrappleData), star-ish valid-
transitions table (~75 edges), trigger constants (22 named), Machine
wrapper with per-state + rollup predicates + data accessors, sync-
mutex registry mirroring life/activity packages.

ControlLevel enum stored as GrappleData field; default Neutral. 4a
does not drive control transitions — 4b adds the per-round rolls.

Character.Position field added (initialized in New() + nil-guarded
in Validate()). Coexists with legacy CombatPosition enum and all
existing position state — chunk 4a is pure scaffold, no behavior
change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Behavior Matrix RED tests (PO-001 through PO-045)

**Files:**
- Create: `internal/state/position/position_test.go`

All 45 Behavior Matrix tests as failing skeletons. Unit-level tests (basic transitions, predicates, data accessors, invalid-rejection) compile-fail because TransitionTo* methods don't exist yet (Task 3). Integration-level tests (cascade, btree primitive) are `t.Skip` stubs that turn on in Tasks 5/6.

- [ ] **Step 1: Create `internal/state/position/position_test.go`**

```go
package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// --- PO-001 through PO-004: Default + nil-safety ---

func TestPO_001_NewMachineStartsStanding(t *testing.T) {
	m := position.NewMachine()
	if m.State() != position.Standing {
		t.Errorf("expected Standing, got %v", m.State())
	}
	if !m.IsStanding() {
		t.Errorf("IsStanding() = false, want true")
	}
	if m.IsGrappling() {
		t.Errorf("IsGrappling() = true, want false")
	}
}

func TestPO_002_StandingHasNoData(t *testing.T) {
	m := position.NewMachine()
	if _, ok := m.ProneData(); ok {
		t.Errorf("ProneData() should return ok=false in Standing")
	}
	if _, ok := m.SupineData(); ok {
		t.Errorf("SupineData() should return ok=false in Standing")
	}
	if _, ok := m.GrappleData(); ok {
		t.Errorf("GrappleData() should return ok=false in Standing")
	}
}

func TestPO_003_StateStringFormatted(t *testing.T) {
	// Sanity: each enum value has a non-empty, non-"Unknown" String().
	for s := position.Standing; s <= position.Turtle; s++ {
		got := s.String()
		if got == "" || got == "Unknown" {
			t.Errorf("State(%d).String() = %q, want non-empty + non-Unknown", s, got)
		}
	}
}

func TestPO_004_ControlLevelStringFormatted(t *testing.T) {
	for c := position.InControl; c <= position.Controlled; c++ {
		got := c.String()
		if got == "" || got == "Unknown" {
			t.Errorf("ControlLevel(%d).String() = %q, want non-empty + non-Unknown", c, got)
		}
	}
}

// --- PO-005 through PO-018: Basic transitions (14 representative samples) ---

func TestPO_005_StandingToProne(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Prone {
		t.Errorf("expected Prone, got %v", m.State())
	}
}

func TestPO_006_StandingToSupine(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToSupine(
		position.SupineData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Supine {
		t.Errorf("expected Supine, got %v", m.State())
	}
}

func TestPO_007_StandingToClinch(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 1}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Clinch {
		t.Errorf("expected Clinch, got %v", m.State())
	}
}

func TestPO_008_ProneToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
	err := m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerStandCommand})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Standing {
		t.Errorf("expected Standing, got %v", m.State())
	}
}

func TestPO_009_SupineToGuard(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	err := m.TransitionToGuard(
		position.GrappleData{Partner: state.ActorRef{UserId: 2}},
		state.TransitionReason{Trigger: position.TriggerGuardPull},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Guard {
		t.Errorf("expected Guard, got %v", m.State())
	}
}

func TestPO_010_ClinchToMount(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 3}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 3}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Mount {
		t.Errorf("expected Mount, got %v", m.State())
	}
}

func TestPO_011_MountToSideControl(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 4}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 4}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	err := m.TransitionToSideControl(
		position.GrappleData{Partner: state.ActorRef{UserId: 4}},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.SideControl {
		t.Errorf("expected SideControl, got %v", m.State())
	}
}

func TestPO_012_SideControlToMount(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 5}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 5}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 5}},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Mount {
		t.Errorf("expected Mount, got %v", m.State())
	}
}

func TestPO_013_MountToBackGround(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 6}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 6}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	err := m.TransitionToBackGround(
		position.GrappleData{Partner: state.ActorRef{UserId: 6}},
		state.TransitionReason{Trigger: position.TriggerBackTakeGround},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.BackGround {
		t.Errorf("expected BackGround, got %v", m.State())
	}
}

func TestPO_014_BackGroundToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 7}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToBackGround(position.GrappleData{Partner: state.ActorRef{UserId: 7}}, state.TransitionReason{Trigger: position.TriggerTakedownBackGround})
	err := m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerPositionEscape})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Standing {
		t.Errorf("expected Standing, got %v", m.State())
	}
}

func TestPO_015_GuardToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	_ = m.TransitionToGuard(position.GrappleData{Partner: state.ActorRef{UserId: 8}}, state.TransitionReason{Trigger: position.TriggerGuardPull})
	err := m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerGrappleBreak})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestPO_016_TurtleAllowsZeroPartner(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 9}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 9}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToTurtle(
		position.GrappleData{Partner: state.ActorRef{}}, // zero — solo defensive curl
		state.TransitionReason{Trigger: position.TriggerTurtleDefend},
	)
	if err != nil {
		t.Fatalf("Turtle should allow zero Partner; got %v", err)
	}
}

func TestPO_017_CrucifixViaSideControl(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 10}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 10}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToCrucifix(
		position.GrappleData{Partner: state.ActorRef{UserId: 10}},
		state.TransitionReason{Trigger: position.TriggerArmIsolation},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.Crucifix {
		t.Errorf("expected Crucifix, got %v", m.State())
	}
}

func TestPO_018_BackStandingViaClinch(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 11}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToBackStanding(
		position.GrappleData{Partner: state.ActorRef{UserId: 11}},
		state.TransitionReason{Trigger: position.TriggerBackTakeStanding},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if m.State() != position.BackStanding {
		t.Errorf("expected BackStanding, got %v", m.State())
	}
}

// --- PO-019 through PO-024: Invalid-transition rejection ---

func TestPO_019_StandingToMountFails(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 12}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if err == nil {
		t.Fatal("Standing → Mount should fail (must go via Clinch or Prone/Supine)")
	}
	if m.State() != position.Standing {
		t.Errorf("state should remain Standing on failed transition, got %v", m.State())
	}
}

func TestPO_020_StandingToBackStandingFails(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToBackStanding(
		position.GrappleData{Partner: state.ActorRef{UserId: 13}},
		state.TransitionReason{Trigger: position.TriggerBackTakeStanding},
	)
	if err == nil {
		t.Fatal("Standing → BackStanding should fail (must go via Clinch)")
	}
}

func TestPO_021_ClinchToKneeOnBellyFails(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 14}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToKneeOnBelly(
		position.GrappleData{Partner: state.ActorRef{UserId: 14}},
		state.TransitionReason{Trigger: position.TriggerPositionAdvance},
	)
	if err == nil {
		t.Fatal("Clinch → KneeOnBelly should fail (KOB requires ground first; go via SC)")
	}
}

func TestPO_022_SupineToBackGroundFails(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	err := m.TransitionToBackGround(
		position.GrappleData{Partner: state.ActorRef{UserId: 15}},
		state.TransitionReason{Trigger: position.TriggerBackTakeGround},
	)
	if err == nil {
		t.Fatal("Supine → BackGround should fail (attacker would need to flip target first)")
	}
}

func TestPO_023_MountToProneFails(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 16}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 16}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
	err := m.TransitionToProne(
		position.ProneData{},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	if err == nil {
		t.Fatal("Mount → Prone should fail (controller can't drop into a non-grapple knockdown directly)")
	}
}

func TestPO_024_GrappleRequiresNonZeroPartner(t *testing.T) {
	m := position.NewMachine()
	err := m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{}}, // zero Partner
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	if err == nil {
		t.Fatal("Clinch should reject zero Partner (only Turtle allows it)")
	}
}

// --- PO-025 through PO-028: GrappleData carries data ---

func TestPO_025_ClinchDataPreserved(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 17}, ControlLevel: position.LosingControl},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	d, ok := m.GrappleData()
	if !ok {
		t.Fatal("expected GrappleData to be available")
	}
	if d.Partner.UserId != 17 {
		t.Errorf("Partner.UserId = %d, want 17", d.Partner.UserId)
	}
	if d.ControlLevel != position.LosingControl {
		t.Errorf("ControlLevel = %v, want LosingControl", d.ControlLevel)
	}
}

func TestPO_026_GrappleDataDefaultsNeutral(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 18}}, // ControlLevel omitted
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	d, _ := m.GrappleData()
	if d.ControlLevel != position.Neutral {
		t.Errorf("default ControlLevel = %v, want Neutral", d.ControlLevel)
	}
}

func TestPO_027_ProneDataPreserved(t *testing.T) {
	m := position.NewMachine()
	src := state.ActorRef{UserId: 19}
	_ = m.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 3, KnockdownSource: src},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	d, ok := m.ProneData()
	if !ok {
		t.Fatal("expected ProneData to be available")
	}
	if d.MinRecoveryRounds != 3 {
		t.Errorf("MinRecoveryRounds = %d, want 3", d.MinRecoveryRounds)
	}
	if d.KnockdownSource.UserId != 19 {
		t.Errorf("KnockdownSource.UserId = %d, want 19", d.KnockdownSource.UserId)
	}
}

func TestPO_028_DataClearedOnReturnToStanding(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 20}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToStanding(state.TransitionReason{Trigger: position.TriggerGrappleBreak})
	if _, ok := m.GrappleData(); ok {
		t.Errorf("GrappleData() should return ok=false after returning to Standing")
	}
}

// --- PO-029 through PO-036: Predicate correctness (Machine-level) ---

func TestPO_029_IsGrapplingMatchesGrappleStates(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*position.Machine)
		want  bool
	}{
		{"Standing", func(m *position.Machine) {}, false},
		{"Prone", func(m *position.Machine) {
			_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
		}, false},
		{"Supine", func(m *position.Machine) {
			_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
		}, false},
		{"Clinch", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
		}, true},
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if got := m.IsGrappling(); got != tc.want {
				t.Errorf("IsGrappling() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPO_030_IsTopDominantMatchesTopStates(t *testing.T) {
	tops := []struct {
		name  string
		setup func(*position.Machine)
	}{
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}},
		{"SideControl", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
		}},
		{"BackGround", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToBackGround(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownBackGround})
		}},
	}
	for _, tc := range tops {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if !m.IsTopDominant() {
				t.Errorf("IsTopDominant() = false for %s, want true", tc.name)
			}
		})
	}
}

func TestPO_031_IsGuardNotTopDominant(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	_ = m.TransitionToGuard(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGuardPull})
	if m.IsTopDominant() {
		t.Errorf("Guard should NOT be IsTopDominant (it's bottom-active)")
	}
	if !m.IsGrappling() || !m.IsGroundGrapple() {
		t.Errorf("Guard should be IsGrappling AND IsGroundGrapple")
	}
}

func TestPO_032_IsStandingGrappleMatchesClinchAndBackStanding(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*position.Machine)
		want  bool
	}{
		{"Clinch", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
		}, true},
		{"BackStanding", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToBackStanding(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerBackTakeStanding})
		}, true},
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if got := m.IsStandingGrapple(); got != tc.want {
				t.Errorf("IsStandingGrapple() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPO_033_IsOnFloorMatchesProneSupineAndGroundGrapples(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*position.Machine)
		want  bool
	}{
		{"Standing", func(m *position.Machine) {}, false},
		{"Prone", func(m *position.Machine) {
			_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
		}, true},
		{"Supine", func(m *position.Machine) {
			_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
		}, true},
		{"Clinch", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
		}, false},
		{"Mount", func(m *position.Machine) {
			_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
			_ = m.TransitionToMount(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownMount})
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := position.NewMachine()
			tc.setup(m)
			if got := m.IsOnFloor(); got != tc.want {
				t.Errorf("IsOnFloor() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPO_034_ProneIsNotSupine(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToProne(position.ProneData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})
	if !m.IsProne() {
		t.Errorf("IsProne() should be true")
	}
	if m.IsSupine() {
		t.Errorf("IsSupine() should be false when in Prone")
	}
}

func TestPO_035_SupineIsNotProne(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToSupine(position.SupineData{}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward})
	if !m.IsSupine() {
		t.Errorf("IsSupine() should be true")
	}
	if m.IsProne() {
		t.Errorf("IsProne() should be false when in Supine")
	}
}

func TestPO_036_RegistryRoundTrip(t *testing.T) {
	m := position.NewMachine()
	ref := state.ActorRef{UserId: 42}
	position.RegisterMachine(ref, m)
	defer position.UnregisterMachine(ref)
	if m.Self() != ref {
		t.Errorf("Self() = %v, want %v", m.Self(), ref)
	}
}

// --- PO-037 through PO-040: Cascade verification (integration — Task 5) ---

func TestPO_037_LifeDeadCascadesPositionStanding(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests (internal/hooks/Position_Cascades_test.go)")
}

func TestPO_038_CascadeFiresFromMount(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestPO_039_CascadeFiresFromGuard(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

func TestPO_040_CascadeFiresFromBackGround(t *testing.T) {
	t.Skip("integration test — verified in Task 5 cascade tests")
}

// --- PO-041 through PO-043: Btree primitive smoke (integration — Task 6) ---

func TestPO_041_BtreeMobInMountFires(t *testing.T) {
	t.Skip("integration test — verified in Task 6 btree primitive tests (internal/behaviortree/conditions_position_test.go)")
}

func TestPO_042_BtreeTargetIsGrappledFires(t *testing.T) {
	t.Skip("integration test — verified in Task 6 btree primitive tests")
}

func TestPO_043_BtreeMobIsProneFires(t *testing.T) {
	t.Skip("integration test — verified in Task 6 btree primitive tests")
}

// --- PO-044, PO-045: Turtle Partner edge case ---

func TestPO_044_TurtleZeroPartnerAccepted(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	_ = m.TransitionToSideControl(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerTakedownSide})
	err := m.TransitionToTurtle(
		position.GrappleData{Partner: state.ActorRef{}},
		state.TransitionReason{Trigger: position.TriggerTurtleDefend},
	)
	if err != nil {
		t.Errorf("TransitionToTurtle should accept zero Partner; got %v", err)
	}
}

func TestPO_045_MountZeroPartnerRejected(t *testing.T) {
	m := position.NewMachine()
	_ = m.TransitionToClinch(position.GrappleData{Partner: state.ActorRef{UserId: 1}}, state.TransitionReason{Trigger: position.TriggerGrappleEntry})
	err := m.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{}}, // zero
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if err == nil {
		t.Errorf("TransitionToMount should reject zero Partner")
	}
}
```

- [ ] **Step 2: Run tests — expect compile-RED**

```bash
go test ./internal/state/position/ 2>&1 | head -20
```

Expected: compile errors mentioning `undefined: m.TransitionToProne / TransitionToSupine / TransitionToClinch / ...`. The 14 TransitionTo* methods don't exist yet (Task 3 adds them).

If the test file doesn't compile but the production code does, that's the RED phase — proceed.

- [ ] **Step 3: Commit RED phase**

```bash
git add internal/state/position/position_test.go
git commit -m "$(cat <<'EOF'
test(position): Behavior Matrix RED — PO-001 through PO-045

45 Behavior Matrix rows authored as failing skeletons:
  - PO-001 through PO-004: Default + nil-safety (4 unit)
  - PO-005 through PO-018: Basic transitions (14 unit — compile-RED until Task 3 implements TransitionTo* methods)
  - PO-019 through PO-024: Invalid-transition rejection (6 unit)
  - PO-025 through PO-028: GrappleData carries data (4 unit)
  - PO-029 through PO-036: Predicate correctness (8 unit including a table-driven IsGrappling sweep)
  - PO-037 through PO-040: Cascade verification (4 SKIP — verified in Task 5 integration tests)
  - PO-041 through PO-043: Btree primitive smoke (3 SKIP — verified in Task 6 integration tests)
  - PO-044, PO-045: Turtle Partner edge case (2 unit)

Compile-RED expected — TransitionTo* methods land in Task 3 (rules.go).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Basic transitions (rules.go) — PO-005 through PO-028 + PO-044/045

**Files:**
- Create: `internal/state/position/rules.go`

The 14 `TransitionTo*` methods plus a `ForceStanding` admin/emergency helper. After this, the unit-level Behavior Matrix rows go GREEN. Pattern mirrors chunks 2/3 `rules.go` files — set per-state data BEFORE calling `m.inner.TransitionTo` (because AfterTransition observers fire during that call), with rollback on error. `TransitionToStanding` clears all per-state data slots (star topology: Standing is the only state with no payload).

- [ ] **Step 1: Create `internal/state/position/rules.go`**

```go
package position

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/state"
)

// ErrPartnerRequired is returned when a grapple transition (other
// than Turtle) is called with a zero ActorRef Partner.
var ErrPartnerRequired = errors.New("grapple transition requires non-zero Partner (only Turtle accepts zero)")

// TransitionToStanding moves to Standing and clears all per-state
// data slots. Used for grapple-break, recovery, escape, and the
// Life Dead cascade.
func (m *Machine) TransitionToStanding(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Standing, r); err != nil {
		return err
	}
	m.prone = nil
	m.supine = nil
	m.grapple = nil
	return nil
}

// TransitionToProne moves to Prone (face-down knockdown, alone).
func (m *Machine) TransitionToProne(d ProneData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.prone
	m.prone = &d
	if err := m.inner.TransitionTo(Prone, r); err != nil {
		m.prone = prev
		return err
	}
	m.grapple = nil
	m.supine = nil
	return nil
}

// TransitionToSupine moves to Supine (face-up knockdown, alone).
func (m *Machine) TransitionToSupine(d SupineData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.supine
	m.supine = &d
	if err := m.inner.TransitionTo(Supine, r); err != nil {
		m.supine = prev
		return err
	}
	m.grapple = nil
	m.prone = nil
	return nil
}

// transitionGrapple is the shared body for all 11 grapple TransitionTo* methods.
// Validates Partner (non-zero except for Turtle), stores data with
// the Reason field populated, fires the inner transition with rollback
// on error, and clears non-grapple data slots on success.
func (m *Machine) transitionGrapple(target State, d GrappleData, r state.TransitionReason) error {
	if target != Turtle && d.Partner.IsZero() {
		return ErrPartnerRequired
	}
	d.Reason = r
	prev := m.grapple
	m.grapple = &d
	if err := m.inner.TransitionTo(target, r); err != nil {
		m.grapple = prev
		return err
	}
	m.prone = nil
	m.supine = nil
	return nil
}

func (m *Machine) TransitionToClinch(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Clinch, d, r)
}

func (m *Machine) TransitionToBackStanding(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(BackStanding, d, r)
}

func (m *Machine) TransitionToMount(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Mount, d, r)
}

func (m *Machine) TransitionToSideControl(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(SideControl, d, r)
}

func (m *Machine) TransitionToKneeOnBelly(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(KneeOnBelly, d, r)
}

func (m *Machine) TransitionToNorthSouth(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(NorthSouth, d, r)
}

func (m *Machine) TransitionToCrucifix(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Crucifix, d, r)
}

func (m *Machine) TransitionToBackGround(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(BackGround, d, r)
}

func (m *Machine) TransitionToHalfGuard(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(HalfGuard, d, r)
}

func (m *Machine) TransitionToGuard(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Guard, d, r)
}

func (m *Machine) TransitionToTurtle(d GrappleData, r state.TransitionReason) error {
	return m.transitionGrapple(Turtle, d, r)
}

// ForceStanding transitions to Standing from any state, bypassing
// the validTransitions table. Used by admin commands and emergency
// cleanup. Idempotent if already Standing.
func (m *Machine) ForceStanding(r state.TransitionReason) {
	if m.State() == Standing {
		return
	}
	_ = m.inner.TransitionTo(Standing, r)
	m.prone = nil
	m.supine = nil
	m.grapple = nil
}
```

`d.Partner.IsZero()` assumes `state.ActorRef` has an `IsZero()` method. If it doesn't, replace with `d.Partner.UserId == 0 && d.Partner.MobInstanceId == 0` — verify with a quick grep:
```bash
grep -n "func (.*ActorRef.*) IsZero" internal/state/
```
If absent, inline the zero-check above.

- [ ] **Step 2: Run the unit-level tests**

```bash
go test ./internal/state/position/ -count=1 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: `ok` for the package. The 10 SKIP rows (PO-037 through PO-043) skip; the rest PASS.

Sanity check the tally:
```bash
go test ./internal/state/position/ -v -count=1 2>&1 | grep -E "^--- (PASS|FAIL|SKIP)" | awk '{print $2}' | sort | uniq -c
```
Expected: ~32 PASS, ~10 SKIP, 0 FAIL. (The IsGrappling table-driven test contains subtests; if the count differs from spec preview, that's because subtests count as individual rows.)

- [ ] **Step 3: Commit**

```bash
git add internal/state/position/rules.go
git commit -m "$(cat <<'EOF'
feat(position): basic transitions PO-005 through PO-028, PO-044/045

14 TransitionTo* methods + ForceStanding admin helper. Pattern
mirrors Life machine's TransitionToDead: per-state data stored
BEFORE inner.TransitionTo (so AfterTransition observers see
populated data), with rollback on error. Shared
transitionGrapple() body for all 11 grapple states factors out
the Partner-zero validation + data storage + cleanup pattern.

TransitionToStanding clears all per-state data slots (star
topology: Standing is the only state with no payload, and it's
the convergent point for all grapple-break / recovery / escape /
cascade transitions).

ErrPartnerRequired returned from grapple transitions called with
zero Partner ActorRef. Turtle exempt (allows solo defensive curl
before any attacker follows).

Unit-level Behavior Matrix rows now GREEN. Integration rows
(PO-037-043) remain SKIP pending Tasks 5/6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Character predicates

**Files:**
- Create: `internal/characters/position_predicates.go`

19 predicates on Character (14 per-state + 5 rollup). Each is a thin nil-guarded delegator to the underlying `c.Position.IsXxx()` method.

- [ ] **Step 1: Create `internal/characters/position_predicates.go`**

```go
// Position predicates on Character — chunk 4a additions.
// Each method delegates to c.Position.IsXxx() with a nil guard
// (a Character constructed outside New() and not run through
// Validate() may have c.Position == nil).
//
// These methods coexist with the legacy CombatPosition enum +
// CombatPosition.IsGroundPosition() / IsGrapplePosition() helpers.
// 4b/4c sunset the enum helpers once command sites cut over.
package characters

// --- Per-state predicates (14) ---

// IsStanding returns true when the character is in Standing position.
func (c *Character) IsStanding() bool {
	if c.Position == nil {
		return true // defensive default; matches NewMachine() initial state
	}
	return c.Position.IsStanding()
}

// IsProne returns true when the character is face-down on the floor, alone.
func (c *Character) IsProne() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsProne()
}

// IsSupine returns true when the character is face-up on the floor, alone.
func (c *Character) IsSupine() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsSupine()
}

// IsClinch returns true when the character is in a standing grapple (clinch).
func (c *Character) IsClinch() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsClinch()
}

// IsBackStanding returns true when one grappler has the back of another, standing.
func (c *Character) IsBackStanding() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsBackStanding()
}

// IsMount returns true when the character is in Mount.
func (c *Character) IsMount() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsMount()
}

// IsSideControl returns true when the character is in Side Control.
func (c *Character) IsSideControl() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsSideControl()
}

// IsKneeOnBelly returns true when the character is in Knee-on-Belly.
func (c *Character) IsKneeOnBelly() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsKneeOnBelly()
}

// IsNorthSouth returns true when the character is in North-South.
func (c *Character) IsNorthSouth() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsNorthSouth()
}

// IsCrucifix returns true when the character is in Crucifix.
func (c *Character) IsCrucifix() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsCrucifix()
}

// IsBackGround returns true when the character is in Back-Ground (rear mount on ground).
func (c *Character) IsBackGround() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsBackGround()
}

// IsHalfGuard returns true when the character is in Half Guard.
func (c *Character) IsHalfGuard() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsHalfGuard()
}

// IsGuard returns true when the character is in Guard.
func (c *Character) IsGuard() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsGuard()
}

// IsTurtle returns true when the character is in Turtle.
func (c *Character) IsTurtle() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsTurtle()
}

// --- Rollup predicates (5) ---

// IsGrappling returns true for any grapple state (any of the 11).
func (c *Character) IsGrappling() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsGrappling()
}

// IsStandingGrapple returns true for Clinch or BackStanding.
func (c *Character) IsStandingGrapple() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsStandingGrapple()
}

// IsGroundGrapple returns true for any ground grapple state (9 states).
func (c *Character) IsGroundGrapple() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsGroundGrapple()
}

// IsTopDominant returns true when the character is in a controller-dominant
// ground position (Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix,
// BackGround). Does NOT take ControlLevel into account — that's a 4b
// refinement.
func (c *Character) IsTopDominant() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsTopDominant()
}

// IsOnFloor returns true for Prone, Supine, or any ground grapple.
func (c *Character) IsOnFloor() bool {
	if c.Position == nil {
		return false
	}
	return c.Position.IsOnFloor()
}
```

- [ ] **Step 2: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 3: Run tests to confirm no regressions**

```bash
go test ./internal/state/... ./internal/characters/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS. New predicates return defaults (true for IsStanding, false for the rest) on a freshly-constructed Character — exactly what existing tests expect since nothing transitions the Position machine.

- [ ] **Step 4: Commit**

```bash
git add internal/characters/position_predicates.go
git commit -m "$(cat <<'EOF'
feat(position): IsStanding/IsProne/IsSupine/.../IsOnFloor predicates on Character

19 predicates (14 per-state + 5 rollup) delegating to the underlying
Position machine with nil guards. Net-new API surface — coexists with
the existing CombatPosition enum + its IsGroundPosition() /
IsGrapplePosition() helpers. 4b/4c sunset the enum helpers once
command sites cut over to write the new FSM.

Defaults on machine-less Character: IsStanding=true, everything else
false. Matches NewMachine() initial-state behavior.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Cross-machine cascade — Life Dead → Position Standing

**Files:**
- Create: `internal/hooks/Position_Cascades.go`
- Create: `internal/hooks/Position_Cascades_test.go`

Single observer subscribing to Life Alive→Dead transitions. Coexists with the chunk-2 Life pre-wire (both reach Standing; no drift because the new FSM defaults to Standing). The 4 SKIP rows from Task 2's matrix (PO-037 through PO-040) become live integration tests here.

- [ ] **Step 1: Create `internal/hooks/Position_Cascades.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// wirePositionCrossMachineCascades registers Position-side observers
// for cross-machine transitions. 4a wires only the Life Dead →
// Standing cascade; 4b adds the per-round control-roll subscriber,
// and 4d may add Activity-side hooks if submissions touch Position.
//
// The Life Dead cascade COEXISTS with chunk-2 Life_Cascades.go
// pre-wire (which still resets c.CombatPosition = PositionStanding
// directly + clears GrappleControllerId). Both observers fire on
// death. Both reach Standing (the chunk-2 pre-wire on the legacy
// field; this observer on the new FSM). No drift is possible
// because the new FSM defaults to Standing and 4a has no writers.
// 4b removes the chunk-2 pre-wire once command sites cut over.
func wirePositionCrossMachineCascades(c *characters.Character) {
	c.Life.Inner().AfterTransition("position_life_dead",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.Position == nil || c.Position.IsStanding() {
				return
			}
			_ = c.Position.TransitionToStanding(state.TransitionReason{
				Trigger: position.TriggerDeath,
				Actor:   c.Position.Self(),
			})
		})
}

func init() {
	characters.OnCharacterCreated(wirePositionCrossMachineCascades)
}
```

- [ ] **Step 2: Create `internal/hooks/Position_Cascades_test.go`**

```go
package hooks_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	_ "github.com/GoMudEngine/GoMud/internal/hooks" // wire init() observers
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// TestPositionCascadeFromMount covers PO-038: Life Dead from Mount
// cascades Position → Standing.
func TestPositionCascadeFromMount(t *testing.T) {
	c := characters.New()
	_ = c.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 100}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = c.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 100}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	if !c.IsMount() {
		t.Fatal("setup: expected Mount")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Position should cascade to Standing on Life Dead; got %v", c.Position.State())
	}
}

// TestPositionCascadeFromGuard covers PO-039.
func TestPositionCascadeFromGuard(t *testing.T) {
	c := characters.New()
	_ = c.Position.TransitionToSupine(
		position.SupineData{},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceBackward},
	)
	_ = c.Position.TransitionToGuard(
		position.GrappleData{Partner: state.ActorRef{UserId: 101}},
		state.TransitionReason{Trigger: position.TriggerGuardPull},
	)
	if !c.IsGuard() {
		t.Fatal("setup: expected Guard")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Position should cascade to Standing on Life Dead from Guard; got %v", c.Position.State())
	}
}

// TestPositionCascadeFromBackGround covers PO-040.
func TestPositionCascadeFromBackGround(t *testing.T) {
	c := characters.New()
	_ = c.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 102}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = c.Position.TransitionToBackGround(
		position.GrappleData{Partner: state.ActorRef{UserId: 102}},
		state.TransitionReason{Trigger: position.TriggerTakedownBackGround},
	)
	if !c.IsBackGround() {
		t.Fatal("setup: expected BackGround")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Position should cascade to Standing on Life Dead from BackGround; got %v", c.Position.State())
	}
}

// TestPositionCascadeNoOpFromStanding covers PO-037 — a character
// who dies while Standing already remains Standing. The observer
// should early-return without firing a redundant transition.
func TestPositionCascadeNoOpFromStanding(t *testing.T) {
	c := characters.New()
	if !c.IsStanding() {
		t.Fatal("setup: expected Standing default")
	}

	_ = c.Life.TransitionToDead(
		life.DeadData{},
		state.TransitionReason{Trigger: life.TriggerSuicide},
	)

	if !c.IsStanding() {
		t.Errorf("Standing should remain Standing on Life Dead; got %v", c.Position.State())
	}
}
```

- [ ] **Step 3: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/state/... ./internal/characters/ ./internal/hooks/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS.

- [ ] **Step 5: Verify the cascade tests specifically**

```bash
go test ./internal/hooks/ -v -count=1 -run 'TestPositionCascade' 2>&1 | grep -E "^--- (PASS|FAIL)"
```
Expected: 4 PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Position_Cascades.go \
        internal/hooks/Position_Cascades_test.go
git commit -m "$(cat <<'EOF'
feat(position): cross-machine cascade — Life Dead → Position Standing

Single AfterTransition observer subscribed to Life Alive→Dead.
Coexists with the chunk-2 Life_Cascades.go pre-wire that still
resets c.CombatPosition = PositionStanding directly + clears
GrappleControllerId. Both observers fire on death; both reach
Standing (chunk-2 pre-wire on the legacy field; this observer on
the new FSM). No drift possible because the new FSM defaults to
Standing and 4a has no writers. 4b removes the pre-wire once
command sites cut over.

Four integration tests cover PO-037 (Standing → no-op), PO-038
(Mount → Standing), PO-039 (Guard → Standing), PO-040 (BackGround
→ Standing). 4a Behavior Matrix integration rows now GREEN.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Btree primitives

**Files:**
- Create: `internal/behaviortree/conditions_position.go`
- Create: `internal/behaviortree/conditions_position_test.go`

10 primitives — 7 self-position + 3 target-position. Registered in the existing conditions registry; dormant in 4a (always return Failure because no mob's Position machine has been transitioned). The 3 SKIP rows from Task 2's matrix (PO-041 through PO-043) become live integration tests here.

- [ ] **Step 1: Read the existing btree conditions pattern**

```bash
cat internal/behaviortree/conditions_mob.go | head -100
```

Look at how `condMobHealthBelow` and `condTargetIsCasting` are structured. The new file mirrors that shape — function takes `(params map[string]any, ctx *EvalContext) Result`, looks up the mob via `mobs.GetInstance(ctx.InstanceId)`, returns Success or Failure.

Also verify the registry pattern:
```bash
grep -n "RegisterCondition\|conditionsRegistry" internal/behaviortree/conditions.go | head -10
```

Conditions are typically registered in `init()` of their file or in a central registry call.

- [ ] **Step 2: Create `internal/behaviortree/conditions_position.go`**

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// --- Self-position queries (7) ---

func condMobIsStanding(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsStanding() {
		return Success
	}
	return Failure
}

func condMobIsProne(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsProne() {
		return Success
	}
	return Failure
}

func condMobIsGrappling(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsGrappling() {
		return Success
	}
	return Failure
}

func condMobInMount(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsMount() {
		return Success
	}
	return Failure
}

func condMobInGuard(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsGuard() {
		return Success
	}
	return Failure
}

func condMobInClinch(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsClinch() {
		return Success
	}
	return Failure
}

func condMobInTopDominant(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsTopDominant() {
		return Success
	}
	return Failure
}

// --- Target-position queries (3) ---

func condTargetIsStanding(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsStanding() {
		return Success
	}
	return Failure
}

func condTargetIsProne(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsProne() {
		return Success
	}
	return Failure
}

func condTargetIsGrappled(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsGrappling() {
		return Success
	}
	return Failure
}

func init() {
	RegisterCondition("mob_is_standing", condMobIsStanding)
	RegisterCondition("mob_is_prone", condMobIsProne)
	RegisterCondition("mob_is_grappling", condMobIsGrappling)
	RegisterCondition("mob_in_mount", condMobInMount)
	RegisterCondition("mob_in_guard", condMobInGuard)
	RegisterCondition("mob_in_clinch", condMobInClinch)
	RegisterCondition("mob_in_top_dominant", condMobInTopDominant)
	RegisterCondition("target_is_standing", condTargetIsStanding)
	RegisterCondition("target_is_prone", condTargetIsProne)
	RegisterCondition("target_is_grappled", condTargetIsGrappled)
}
```

Verify the `RegisterCondition` function signature matches existing usage. If the registry uses a different API (e.g., `conditionsRegistry["name"] = fn` directly), conform to whatever pattern exists. Adjust the `init()` block as needed.

- [ ] **Step 3: Create `internal/behaviortree/conditions_position_test.go`**

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// --- PO-041: mob_in_mount fires only when mob in Mount ---

func TestCondMobInMount_NotInMount_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine() // already Standing
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobInMount(nil, ctx))
}

func TestCondMobInMount_InMount_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 200}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	_ = mob.Character.Position.TransitionToMount(
		position.GrappleData{Partner: state.ActorRef{UserId: 200}},
		state.TransitionReason{Trigger: position.TriggerTakedownMount},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobInMount(nil, ctx))
}

// --- PO-042: target_is_grappled fires when target is in any grapple state ---

func TestCondTargetIsGrappled_TargetGrappling_Success(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 201}
	target.Character.Name = "Target"
	target.Character.Position = position.NewMachine()
	_ = target.Character.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{MobInstanceId: mob.InstanceId}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	)
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condTargetIsGrappled(nil, ctx))
}

func TestCondTargetIsGrappled_TargetStanding_Failure(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 202}
	target.Character.Name = "Target"
	target.Character.Position = position.NewMachine() // Standing default
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condTargetIsGrappled(nil, ctx))
}

// --- PO-043: mob_is_prone fires only when mob is in Prone ---

func TestCondMobIsProne_InProne_Success(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine()
	_ = mob.Character.Position.TransitionToProne(
		position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
	)
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Success, condMobIsProne(nil, ctx))
}

func TestCondMobIsProne_InStanding_Failure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.Position = position.NewMachine() // Standing default
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	assert.Equal(t, Failure, condMobIsProne(nil, ctx))
}

// --- Lookup verification: all 10 primitives registered ---

func TestPositionPrimitives_AllRegistered(t *testing.T) {
	names := []string{
		"mob_is_standing", "mob_is_prone", "mob_is_grappling",
		"mob_in_mount", "mob_in_guard", "mob_in_clinch", "mob_in_top_dominant",
		"target_is_standing", "target_is_prone", "target_is_grappled",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if LookupCondition(name) == nil {
				t.Errorf("condition %q not registered", name)
			}
		})
	}
}
```

`newTestMob(t)` already exists in `internal/behaviortree/conditions_test.go` (chunk-3 added the helper). Reuse it.

If `LookupCondition` doesn't exist, use the actual registry name from chunk 3's pattern (cross-check `internal/behaviortree/conditions.go` — chunk 3 added `LookupCondition` for the cancel_activity primitive).

- [ ] **Step 4: Build verify + run tests**

```bash
go build ./...
go test ./internal/behaviortree/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: PASS.

```bash
go test ./internal/behaviortree/ -v -count=1 -run 'TestCondMob|TestCondTarget|TestPositionPrimitives' 2>&1 | grep -E "^--- (PASS|FAIL)" | head -20
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/conditions_position.go \
        internal/behaviortree/conditions_position_test.go
git commit -m "$(cat <<'EOF'
feat(position): 10 btree primitives for position queries

7 self-position: mob_is_standing, mob_is_prone, mob_is_grappling,
mob_in_mount, mob_in_guard, mob_in_clinch, mob_in_top_dominant.

3 target-position: target_is_standing, target_is_prone, target_is_grappled.

All registered in the conditions registry; dormant in 4a (no
mob's Position machine has been transitioned, so all return
Failure in real combat). Available for content authors to
reference in btree YAML; will start firing correctly once 4b
drives transitions from production code.

Control-axis primitives (mob_is_in_control / target_is_being_controlled)
deferred to 4b along with the rolls. `mob_is_supine`,
`mob_in_side_control`, etc. deferred to as-needed expansion —
the rollup mob_in_top_dominant covers the broad cases for 4a.

Six smoke tests cover PO-041 / PO-042 / PO-043 + lookup
verification of all 10 primitives.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Documentation (light touch — 4b's broader sweep flagged)

**Files:**
- Create: `internal/state/position/context.md`
- Modify: `internal/characters/context.md` (append integration section)
- Modify: `internal/hooks/context.md` (append cascade section)

4a docs scope is intentionally narrow: the new package gets full context.md; the two integration points (characters, hooks) get short scaffold-note sections. **The broader doc sweep — combat package, mobs package, behaviortree package, behaviortree archetype YAMLs, helpfiles for stand/grapple/trip/etc., player-facing combat docs — is deferred to 4b's cutover** (and beyond, as control axis / weapons / submissions / third-party each add their own doc surface).

- [ ] **Step 1: Create `internal/state/position/context.md`**

Use `internal/state/life/context.md` (~250 lines, chunk-2) as the structural template. Required sections:

- **Overview** — package purpose, replaces what (eventually — explicit note that 4a is scaffold and 4b is the cutover; legacy `CombatPosition` enum + `PositionRoundsMin` + `GrappleControllerId` + `ConditionGrappleController` are untouched in 4a).
- **Key Components** (file map): position.go / transitions.go / rules.go / position_test.go.
- **State diagram** (ASCII or just listed): 14 states grouped by category (Standing, knockdown alone, standing pair, ground top-dominant, ground transitional, ground bottom-active, ground defensive).
- **Per-state data**: StandingData (empty), ProneData (Reason + MinRecoveryRounds + KnockdownSource), SupineData (same shape — split reasons documented), shared GrappleData (Reason + Partner + ControlLevel).
- **ControlLevel enum** (5 values; 4b adds the rolls).
- **Transition graph summary** (point at transitions.go for the full table; document the intentional non-edges Standing→BackStanding, Supine→BackGround, Clinch→KOB/NS/Crucifix).
- **Trigger constants** — 22 named.
- **Key Functions / Machine API** — TransitionTo* + ForceStanding + data accessors + Inner.
- **Character API** — 19 predicates (14 per-state + 5 rollup), each with one-line purpose.
- **Btree primitives** — list the 10 with one-line purpose each.
- **Cascade integration** — the Life Dead → Standing observer (`Position_Cascades.go` `activity_life_dead` handler). Note that the chunk-2 Life pre-wire coexists during 4a and is removed in 4b.
- **Intentional simplifications** (REQUIRED — chunk 2/3 pattern):
  1. No Prone/Supine to Prone consolidation — split intentional (submissions, recovery, vulnerability diverge).
  2. No per-grapple-state extras on GrappleData (ClinchGrip, ArmsIsolated, HooksIn, TrappedLeg, GuardVariant) — placeholders for 4b/4c.
  3. No control-axis rolls — ControlLevel field exists, default Neutral; 4b adds the rolls.
  4. No btree primitives for control axis — `mob_is_in_control` etc. are 4b.
  5. Cascade observers coexist with chunk-2 pre-wire — no drift possible because the new FSM defaults to Standing and 4a has no writers.
  6. Standing → BackStanding NOT a direct edge — must go via Clinch.
  7. Supine → BackGround NOT a direct edge — would require flipping the supine target face-down first.
  8. Clinch → KOB / NorthSouth / Crucifix NOT direct edges — those positions require the target already on the ground.
  9. No `mob_is_supine`, `mob_in_side_control`, etc. btree primitives in 4a — only commonly-queried positions get individual primitives; rollup `mob_in_top_dominant` covers the broad cases.
- **Persistence** — non-persistent (`yaml:"-"`); character logs in at Standing regardless of prior state.
- **Testing notes** — PO-001 through PO-045, unit-level vs integration split (~32 PASS + ~10 SKIP at unit level; SKIPs become live in Tasks 5/6 integration tests).
- **Sunset notes** — what 4a deletes (nothing in production code). Catalog of legacy targets for FUTURE sub-chunks: CombatPosition enum + PositionRoundsMin + GrappleControllerId + ConditionGrappleController + chunk-2 Life pre-wire + chunk-0 CombatPhase RegisterPositionCheck.
- **What 4b/4c/4d/4e/4f bring** — short forward-reference: 4b cuts over writers + control rolls + sunset legacy; 4c weapon-utility table; 4d submission rework; 4e third-party asymmetries; 4f balance + flavor + smoke.

Target ~250-300 lines, match `life/context.md` style.

- [ ] **Step 2: Append section to `internal/characters/context.md`**

Read the file first to find the chunk-3 Activity Integration section as the insertion-point reference:
```bash
grep -n "Activity Machine Integration\|^## " internal/characters/context.md | head -20
```

Append a new section after the chunk-3 Activity Machine Integration section:

```markdown
## Position Machine Integration (chunk 4a — scaffold)

Chunk 4a adds `Character.Position *position.Machine` as a 14-state
FSM scaffold. **Dormant in 4a** — no production code transitions
the machine; 4b's command-site cutover makes it live.

- Field initialized in `New()` alongside other state machines
  (`CombatPhase`, `Awareness`, `Life`, `Activity`).
- Nil-guarded in `Validate()` for YAML-loaded characters.
- 19 predicates on Character (14 per-state — IsStanding, IsProne,
  IsSupine, IsClinch, IsBackStanding, IsMount, IsSideControl,
  IsKneeOnBelly, IsNorthSouth, IsCrucifix, IsBackGround,
  IsHalfGuard, IsGuard, IsTurtle — plus 5 rollups — IsGrappling,
  IsStandingGrapple, IsGroundGrapple, IsTopDominant, IsOnFloor) in
  `internal/characters/position_predicates.go`. Default behavior on
  a machine-less Character: IsStanding returns true, everything
  else false.

**Coexists with the legacy `CombatPosition` enum** during 4a. The
legacy field + `PositionRoundsMin` + `GrappleControllerId` +
`ConditionGrappleController` + all command writers + all readers
remain unchanged. 4b's cutover swaps the command-site writers to
the new FSM and incrementally sunsets the legacy state.

See `internal/state/position/context.md` for the full Position
package documentation (state list, transitions, intentional
simplifications, what 4b-4f bring).
```

- [ ] **Step 3: Append section to `internal/hooks/context.md`**

Same pattern — find the chunk-3 Activity cascade section as the insertion-point reference, append after it:

```markdown
## Position Cascade (chunk 4a — scaffold)

Chunk 4a adds `Position_Cascades.go` — a single AfterTransition
observer that subscribes to Life Alive→Dead and transitions
`Position` to Standing when fired.

**Coexists with the chunk-2 Life pre-wire.** Chunk 2's
`Life_Cascades.go` still resets `c.CombatPosition = PositionStanding`
directly + clears `GrappleControllerId`. The 4a observer fires
alongside it. No drift possible because the new Position FSM
defaults to Standing and 4a has no writers (nothing has transitioned
the new FSM during normal play). 4b removes the chunk-2 Life
pre-wire once command sites cut over to write the new FSM.

The observer handler key is `position_life_dead`. Self-position
predicates remain stable across the cascade because death always
reaches Standing regardless of starting position.

See `internal/state/position/context.md` for the broader Position
package documentation.
```

- [ ] **Step 4: Build verify (defensive)**

```bash
go build ./...
```
Docs shouldn't break anything, but verify nothing accidentally broke.

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/context.md \
        internal/characters/context.md \
        internal/hooks/context.md
git commit -m "$(cat <<'EOF'
docs(position): chunk-4a documentation scaffold

NEW: internal/state/position/context.md (~250 lines) — full
package docs for the Position state machine. 14-state taxonomy,
per-state data (StandingData / ProneData / SupineData /
GrappleData shared across 11 grapple states), ControlLevel enum
(default Neutral, rolls land in 4b), trigger constants, Machine
API, 19 Character predicates, 10 btree primitives, Life-Dead
cascade observer, intentional simplifications (9 documented:
Prone/Supine split, shared GrappleData, no control rolls, no
control btree primitives, cascade coexistence, intentional non-
edges, btree primitive subset), persistence, testing notes,
sunset catalog for 4b+, forward-reference to 4b-4f.

APPENDED: thin integration sections to internal/characters/context.md
("Position Machine Integration — scaffold") and
internal/hooks/context.md ("Position Cascade — scaffold").

The broader doc sweep — combat package, mobs package, btree
archetype YAMLs, helpfiles for stand/grapple/trip/etc., player-
facing combat docs — is deferred to 4b's cutover. 4a only touches
the new package's docs + thin integration notes; nothing
production-facing changes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Build / test / smoke validation

**Files:** (verification only)

- [ ] **Step 1: Full build**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 2: Full test suite**

```bash
go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)" | head -40
```
Expected: every package PASS. No FAILs.

- [ ] **Step 3: Position Behavior Matrix status**

```bash
go test ./internal/state/position/ -v -count=1 2>&1 | grep -E "^--- (PASS|FAIL|SKIP)" | awk '{print $2}' | sort | uniq -c
```
Expected tally: ~32 PASS, ~10 SKIP, 0 FAIL. (Subtests under TestPO_029 / TestPO_030 / TestPO_032 / TestPO_033 push the PASS count above 32; the SKIP count stays at 10.)

- [ ] **Step 4: Integration tests (cascade + btree)**

```bash
go test ./internal/hooks/ -v -count=1 -run 'TestPositionCascade' 2>&1 | grep -E "^--- (PASS|FAIL)"
go test ./internal/behaviortree/ -v -count=1 -run 'TestCondMob|TestCondTarget|TestPositionPrimitives' 2>&1 | grep -E "^--- (PASS|FAIL)"
```
Expected: all PASS.

- [ ] **Step 5: Chunk 0/1/2/3 regression check**

```bash
go test ./internal/state/combatphase/ ./internal/state/awareness/ ./internal/state/life/ ./internal/state/activity/ -count=1 2>&1 | grep -E "^(ok|FAIL)"
```
Expected: all PASS. No regressions in prior state machines.

- [ ] **Step 6: Server boot**

```bash
go build -o /tmp/dogmud-chunk4a-validate.exe . && /tmp/dogmud-chunk4a-validate.exe > /tmp/dogmud-chunk4a-validate.log 2>&1 &
SERVER_PID=$!
until grep -qE "Server Ready|panic|FATAL" /tmp/dogmud-chunk4a-validate.log; do sleep 3; done
grep -E "Server Ready|panic|FATAL|loadedCount" /tmp/dogmud-chunk4a-validate.log | head -25
kill -9 $SERVER_PID 2>/dev/null
rm -f /tmp/dogmud-chunk4a-validate.exe /tmp/dogmud-chunk4a-validate.log
```
Expected: `Server Ready`, no panic, all data files load.

- [ ] **Step 7: Note in-game smoke deferred**

Per the spec's "open questions" section, 4a has a known reachability test gap: no production code transitions the new FSM, so we can't smoke-test "does combat-induced position change work" until 4b lands. 4a smoke is "legacy combat still works + new FSM tests pass + server boots cleanly." Full in-game smoke is 4b/4f territory.

DO NOT commit; just verify and report.

---

## Task 9: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 4a Done**

Find the chunk 4a row in the progress table (added by the 2026-05-16 master spec amendment):
```bash
grep -n "^| 4a " COMBAT_STATE_ROADMAP.md
```

Update the status from `Not started` to `Done (2026-05-16)` (or actual completion date). Keep the description but extend with the as-shipped specifics:

```markdown
| 4a | Position — FSM | Done (2026-05-16) | 14 geometric states (Standing / Prone / Supine / Clinch / BackStanding / Mount / SideControl / KneeOnBelly / NorthSouth / Crucifix / BackGround / HalfGuard / Guard / Turtle). Per-state data (StandingData / ProneData / SupineData / shared GrappleData), 22 trigger constants, ~75-edge transition graph, 19 Character predicates, 10 btree primitives, Life-Dead cascade observer. Ships DORMANT — zero behavior change; legacy CombatPosition enum + all command writers untouched. 4b cuts over writers + control rolls + sunsets legacy. |
```

Add a "Chunk 4a — Shipped" section parallel to chunks 0/1/2/3, after the existing Chunk 3 — Shipped section. Cover:

- New package `internal/state/position/` with 14-state taxonomy (Standing / Prone / Supine / Clinch / BackStanding / Mount / SideControl / KneeOnBelly / NorthSouth / Crucifix / BackGround / HalfGuard / Guard / Turtle).
- Per-state data: StandingData (empty), ProneData (Reason + MinRecoveryRounds + KnockdownSource), SupineData (same shape — split because submission paths, recovery difficulty, and back-take vulnerability diverge between face-down and face-up knockdown), shared GrappleData (Reason + Partner + ControlLevel) across all 11 grapple states. Per-state extras (ClinchGrip, ArmsIsolated, HooksIn, TrappedLeg, GuardVariant) deferred to 4b/4c as wrapping structs.
- ControlLevel enum (InControl / LosingControl / Neutral / BecomingControlled / Controlled) stored as GrappleData field; defaults to Neutral; nothing in 4a drives transitions (4b adds the per-round opposed rolls).
- Star-ish transition graph: ~75 valid edges across the 14×14 matrix. Intentional non-edges documented (Standing → BackStanding requires Clinch first; Supine → BackGround requires intermediate state; Clinch → KOB/NS/Crucifix requires ground-pin first).
- 22 trigger constants covering knockdowns (face-forward / face-backward / spell), recovery (roll + stand-command), grapple entry/break, 5 takedown variants from Clinch, 3 back-take paths, controller-advance + controlled-escape, defensive (turtle-defend + guard-pull), opportunistic (mount-prone-target), arm-isolation (Crucifix), and cascade (death).
- 19 Character predicates: 14 per-state + 5 rollup (IsGrappling / IsStandingGrapple / IsGroundGrapple / IsTopDominant / IsOnFloor).
- 10 btree primitives: 7 self-position (mob_is_standing, mob_is_prone, mob_is_grappling, mob_in_mount, mob_in_guard, mob_in_clinch, mob_in_top_dominant) + 3 target-position (target_is_standing, target_is_prone, target_is_grappled). Dormant in 4a (always Failure because no mob's Position machine is transitioned in production); become live when 4b drives transitions.
- Cross-machine cascade: Life Dead → Position Standing observer coexists with the chunk-2 Life pre-wire. Both fire on death; both reach Standing; no drift possible because the new FSM defaults to Standing and 4a has no writers. Pre-wire removed in 4b once command sites cut over.
- 4a ships DORMANT: legacy `CombatPosition` enum + `PositionRoundsMin` + `GrappleControllerId` + `ConditionGrappleController` + all command writers (trip / bash / grapple / stand / kick / spell knockdown / AttemptRecovery) + all readers (kick variant selector / flee veto / defense degradation / RegisterPositionCheck) remain unchanged. Zero behavior change.
- Behavior Matrix: 45 intent-driven tests (PO-001 through PO-045) authored in `position_test.go`. ~32 PASS directly at the unit layer; ~10 SKIP because they require cross-machine wiring (covered by `Position_Cascades_test.go` + `conditions_position_test.go` integration tests). Chunks 0/1/2/3 regression tests pass; package tests across the affected boundary (state/..., characters, hooks, behaviortree) all green.
- Intentional simplifications documented in `internal/state/position/context.md` (9 documented: Prone/Supine split, shared GrappleData, no control rolls, no control btree primitives, cascade coexistence, intentional non-edges Standing→BackStanding / Supine→BackGround / Clinch→KOB/NS/Crucifix, btree primitive subset).
- Deferred to 4b: command-site writer cutover, per-round control rolls, legacy field sunset, broader doc sweep (combat / mobs / behaviortree / archetype YAMLs / helpfiles).

Update "Next" pointer to chunk 4b (Position — control axis).

- [ ] **Step 2: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 4a (Position FSM) Done

14-state Position state machine ships as pure-architecture
scaffold. Zero behavior change in 4a; legacy CombatPosition enum
and all command writers / readers remain unchanged. 4b cuts over
writers + adds per-round control rolls + sunsets legacy. Behavior
Matrix ~32 PASS + ~10 SKIP at unit level; integration verified via
4 cascade tests + 6 btree primitive tests. Chunks 0-3 regression
clean.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Task(s) |
|---|---|
| State enumeration (14 states) | T1 |
| ControlLevel enum (5 values, default Neutral) | T1 |
| Per-state data (StandingData / ProneData / SupineData / shared GrappleData) | T1 |
| Transition table (~75 edges, star-ish topology) | T1 (transitions.go) |
| Trigger constants (22 named) | T1 (transitions.go) |
| Machine API (TransitionTo* x14, ForceStanding, *Data accessors, Inner, registry) | T1 (declarations) + T3 (implementations) |
| Character predicates (14 per-state + 5 rollup) | T4 |
| Btree primitives (10 — 7 self + 3 target) | T6 |
| Cross-machine cascade (Life Dead → Position Standing) | T5 |
| Cascade coexistence with chunk-2 Life pre-wire (no drift, no removal) | T5 (observer body + test docs) |
| Mob/player parity (Position field on all Characters; no asymmetry) | T1 (by construction) |
| Turtle Partner zero-value allowance, other grapple Partner required | T3 (transitionGrapple helper + ErrPartnerRequired) |
| Behavior Matrix authored (~45 rows) | T2 |
| Intentional simplifications documented (9 items) | T7 (context.md) |
| Build / test / smoke validation | T8 |
| Roadmap closeout | T9 |

All spec sections covered.

## Known followups (out of chunk 4a)

- **4b — command-site writer cutover + control axis.** All
  `trip` / `bash` / `grapple` / `stand` / `combat_kick` / spell
  knockdown / `AttemptRecovery` writers migrate to the new FSM.
  Per-round opposed rolls drive ControlLevel. Threshold-triggered
  position transitions. Legacy `CombatPosition` enum +
  `PositionRoundsMin` + `GrappleControllerId` +
  `ConditionGrappleController` + chunk-2 Life pre-wire +
  chunk-0 `RegisterPositionCheck` all sunset incrementally.
  Broader doc sweep + helpfile updates land here.
- **4c — weapon-utility-by-position table.** YAML content table:
  `(Position × WeaponType) → modifier`. Combat resolution reads
  it. Lets long weapons fail in mount, knives stay useful.
- **4d — submission rework.** Sunset or repurpose the existing
  submission special-attack command. Automatic/opportunistic
  submissions gated on (Position, ControlLevel). Submission
  outcomes (choked-out, damaged-limb, tap-out, continue) authored.
- **4e — third-party interaction.** Symmetric defense degradation
  (controller moderately, controlled severely), grappler offense
  restrictions vs third party, outside-damage → control-axis
  degradation, mob AI bias toward attacking grappled enemies,
  submission-interrupt risk.
- **4f — balance pass + flavor text + full-stack combat smoke.**
- **Prone/Supine in-grapple orientation extras** (e.g., "mount
  controller is sitting upright vs lying on chest"). If 4c/4d/4e
  pull for it, add per-state Orientation field.
- **Per-grapple-state grip / variant details** (ClinchGrip,
  ArmsIsolated, HooksIn, TrappedLeg, GuardVariant) — added in
  4b/4c via wrapping structs around GrappleData.
- **N-vs-1 grappling, cardio effects, clinch-grip granularity as
  FSM states** — explicitly logged in master spec as out-of-scope
  followups for the whole chunk 4 series.
