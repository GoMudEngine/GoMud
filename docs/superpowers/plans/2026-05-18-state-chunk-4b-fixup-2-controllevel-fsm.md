# State Chunk 4b-fixup-2 — ControlLevel FSM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-05-18-state-chunk-4b-fixup-2-controllevel-fsm-design.md`

**Goal:** Replace chunk 4b-fixup's `IsControllerRole bool` (which broke per-round drift in symmetric Clinch grapples) with a proper 5-state FSM in `internal/state/control/`, mirroring the Awareness package pattern, plus pair iteration in `processGrappleTick` to fix the iteration-layer bug, plus ~36 gradient messaging templates fired on boundary-crossing transient states.

**Architecture:** Three layers — (1) new `internal/state/control/` package containing a 5-state machine (3 stable + 2 transient, mirroring Awareness `Hidden → Revealing → Visible` same-tick pattern), per-character `Character.Control` field; (2) `processGrappleTick` refactored to iterate pairs (deduped) instead of per-character + bool filter, with ControlLevel shift logic running parallel to the unchanged chunk-4b-fixup outcome resolver; (3) gradient messaging library extension with new `GradientTriad` struct + `gradients:` YAML section + transient-state boundary-cross callbacks for flavor emission.

**Tech Stack:** Go 1.21+, existing `internal/state/` framework (`state.Machine[S]`, `TransitionTable[S]`, `TransitionReason`), existing `dice.OpposedRollStat`, existing `position.TransitionPair`, existing `internal/grapplemessaging/` loader + render. No new dependencies.

---

## File Structure

### Created
| Path | Responsibility |
|---|---|
| `internal/state/control/control.go` | State enum + Machine struct + transition methods + boundary-cross callback registration |
| `internal/state/control/control_test.go` | Unit tests (transitions, transient resolution, callbacks, idempotency) |
| `internal/state/control/transitions.go` | Trigger constants + validTransitions table |
| `internal/state/control/context.md` | Package docs |

### Modified
| Path | Change |
|---|---|
| `internal/characters/character.go` | Add `Control *control.Machine` field; init in `NewCharacter`; reset in `ResetForMobInstance` |
| `internal/state/position/position.go` | Delete `GrappleData.IsControllerRole`; refactor `IsController()` / `IsBeingControlled()` to delegate to Character.Control |
| `internal/state/position/pair.go` | Add `IsAggressor` field; refactor `TransitionPair` for ControlLevel init via callback |
| `internal/state/position/validation.go` | Replace invariant 4 (IsControllerRole exclusivity) with ControlLevel-state invariants |
| `internal/state/position/submissions.go` | Refactor `IsTopSubEligible` / `IsBottomSubEligible` to take `control.State` instead of `bool` |
| `internal/state/position/context.md` | Document ControlLevel FSM integration |
| `internal/combat/grapple.go` | Set `IsAggressor=true` on attacker when calling TransitionPair via ApplyGrappleResult |
| `internal/hooks/Position_GrappleTick.go` | Pair iteration + ControlLevel shift logic in processGrapplePair |
| `internal/hooks/Position_SubmissionTick.go` | Update sub eligibility callers to pass `control.State` instead of bool |
| `internal/hooks/context.md` | Update for pair iteration + ControlLevel shift |
| `internal/behaviortree/conditions_submission.go` | Update sub eligibility callers |
| `internal/grapplemessaging/loader.go` | Add `GradientTriad` struct + `Library.Gradients` field + `RequiredGradientKeys` + extend `ValidateCompleteness` |
| `internal/grapplemessaging/loader_test.go` | Gradient validator tests + production-library guard tests |
| `internal/grapplemessaging/render.go` | Add `RenderGradient` helper (mirrors RenderTemplate but for the Self/Partner/Observers triad) |
| `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml` | New `gradients:` section with ~36 templates |
| `COMBAT_STATE_ROADMAP.md` | Add chunk 4b-fixup-2 row |

### Deleted
(No standalone file deletions — `IsControllerRole` field removal is in-place.)

---

## Task 1: Scaffold internal/state/control/ package — State enum + transition table + trigger constants

**Files:**
- Create: `internal/state/control/control.go`
- Create: `internal/state/control/transitions.go`
- Test: `internal/state/control/control_test.go`

- [ ] **Step 1: Create `internal/state/control/transitions.go` with valid transitions + trigger constants**

```go
// Package-level transition table + trigger constants for the
// ControlLevel state machine. Mirrors the awareness package shape.
package control

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the ControlLevel invariant matrix.
//
// Stable states (Controlling, Neutral, Controlled) can transition to
// any of the other stables; transient states (LosingControl,
// BecomingControlled) are entered same-tick during boundary crossings
// and immediately transition to their resolution state.
//
// The machine itself enforces only that transitions follow the
// gradient — the same-tick chaining is implemented in the
// TransitionTo* methods (see control.go).
var validTransitions = state.TransitionTable[State]{
	Controlling:        {LosingControl},
	LosingControl:      {Neutral, Controlling}, // Same-tick resolution to whichever stable state is the target
	Neutral:            {LosingControl, BecomingControlled},
	BecomingControlled: {Neutral, Controlled}, // Same-tick resolution
	Controlled:         {BecomingControlled},
}

// Trigger reason constants.
const (
	// TriggerGrappleEnter fires when a grapple pair enters its
	// initial position. Sets per-side ControlLevel based on the
	// position's symmetry class (see internal/state/position/pair.go).
	TriggerGrappleEnter = "grapple_enter"

	// TriggerDriftWin fires when a side won the per-round drift
	// roll by enough margin to shift state toward Controlling.
	TriggerDriftWin = "drift_win"

	// TriggerDriftLoss fires when a side lost the per-round drift
	// roll by enough margin to shift state toward Controlled.
	TriggerDriftLoss = "drift_loss"

	// TriggerGrappleExit fires when a grapple breaks (escape, death,
	// etc.). Resets ControlLevel to Neutral.
	TriggerGrappleExit = "grapple_exit"
)
```

- [ ] **Step 2: Write the failing test for the State enum, String(), and NewMachine()**

Create `internal/state/control/control_test.go`:

```go
package control

import "testing"

func TestStateString(t *testing.T) {
	cases := []struct {
		state State
		want  string
	}{
		{Controlling, "Controlling"},
		{LosingControl, "LosingControl"},
		{Neutral, "Neutral"},
		{BecomingControlled, "BecomingControlled"},
		{Controlled, "Controlled"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := c.state.String(); got != c.want {
				t.Errorf("State(%d).String() = %q, want %q", c.state, got, c.want)
			}
		})
	}
}

func TestNewMachineDefaultsToNeutral(t *testing.T) {
	m := NewMachine()
	if m.State() != Neutral {
		t.Errorf("NewMachine() state = %v, want Neutral", m.State())
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/state/control/ -count=1`
Expected: FAIL with no package or undefined types.

- [ ] **Step 4: Create `internal/state/control/control.go` with State enum + Machine scaffolding**

```go
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
```

- [ ] **Step 5: Run tests to verify pass**

Run: `go test ./internal/state/control/ -count=1 -v`
Expected: PASS.

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Commit**

```bash
git add internal/state/control/control.go internal/state/control/transitions.go internal/state/control/control_test.go
git commit -m "feat(control): chunk 4b-fixup-2 T1 — scaffold control package

Adds internal/state/control/ — State enum (5 states: 3 stable +
2 transient), validTransitions table, 4 trigger constants
(GrappleEnter/DriftWin/DriftLoss/GrappleExit), Machine wrapper,
NewMachine. Mirrors the awareness package shape.

Transition methods + boundary-cross callbacks land in T2.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: ControlLevel transition methods with same-tick transient handling

**Files:**
- Modify: `internal/state/control/control.go`
- Modify: `internal/state/control/control_test.go`

- [ ] **Step 1: Write failing tests for transition methods**

Append to `internal/state/control/control_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestTransitionToControllingFromNeutral(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Fatalf("TransitionToControlling: %v", err)
	}
	if m.State() != Controlling {
		t.Errorf("after TransitionToControlling, state = %v, want Controlling", m.State())
	}
}

func TestTransitionToControlledFromNeutral(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss}); err != nil {
		t.Fatalf("TransitionToControlled: %v", err)
	}
	if m.State() != Controlled {
		t.Errorf("after TransitionToControlled, state = %v, want Controlled", m.State())
	}
}

func TestTransitionToNeutralFromControlling(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	if err := m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftLoss}); err != nil {
		t.Fatalf("TransitionToNeutral: %v", err)
	}
	if m.State() != Neutral {
		t.Errorf("after TransitionToNeutral, state = %v, want Neutral", m.State())
	}
}

func TestTransitionToNeutralFromControlled(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss})
	if err := m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Fatalf("TransitionToNeutral: %v", err)
	}
	if m.State() != Neutral {
		t.Errorf("after TransitionToNeutral, state = %v, want Neutral", m.State())
	}
}

func TestTransitionCrossingBothBoundariesControllingToControlled(t *testing.T) {
	// Direct jump from Controlling to Controlled traverses both
	// boundaries. End state must be Controlled.
	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	if err := m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss}); err != nil {
		t.Fatalf("TransitionToControlled: %v", err)
	}
	if m.State() != Controlled {
		t.Errorf("after Controlling→Controlled, state = %v, want Controlled", m.State())
	}
}

func TestTransitionCrossingBothBoundariesControlledToControlling(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss})
	if err := m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Fatalf("TransitionToControlling: %v", err)
	}
	if m.State() != Controlling {
		t.Errorf("after Controlled→Controlling, state = %v, want Controlling", m.State())
	}
}

func TestTransitionIdempotency(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	// Transitioning to current state should be a no-op
	if err := m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Errorf("idempotent transition: %v", err)
	}
	if m.State() != Controlling {
		t.Errorf("idempotent transition changed state to %v", m.State())
	}
}

func TestTransitionToNeutralWhenAlreadyNeutralIsNoOp(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Errorf("Neutral→Neutral: %v", err)
	}
	if m.State() != Neutral {
		t.Errorf("state changed to %v", m.State())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/control/ -run "TestTransition" -count=1`
Expected: FAIL with undefined methods.

- [ ] **Step 3: Add transition methods to `internal/state/control/control.go`**

Append to `control.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/state/control/ -count=1 -v`
Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/state/control/control.go internal/state/control/control_test.go
git commit -m "feat(control): chunk 4b-fixup-2 T2 — transition methods + boundary callbacks

TransitionToControlling/Neutral/Controlled handle direct jumps
across both boundaries same-tick. traverse() walks the stable
rank gradient (Controlling=0, Neutral=1, Controlled=2),
crossing transient states (LosingControl for upper boundary,
BecomingControlled for lower) and firing a registered callback
for each crossing.

Callback wiring to grapplemessaging lands in T13.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Boundary-cross callback test coverage

**Files:**
- Modify: `internal/state/control/control_test.go`

- [ ] **Step 1: Add failing test for boundary callback firing**

Append to `control_test.go`:

```go
func TestBoundaryCallbackFiresOnceForSingleBoundary(t *testing.T) {
	var fires []struct {
		transient State
		from      State
		to        State
	}
	RegisterBoundaryCrossCallback(func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
		fires = append(fires, struct {
			transient State
			from      State
			to        State
		}{transient, from, to})
	})
	defer RegisterBoundaryCrossCallback(nil)

	m := NewMachine()
	// Neutral → Controlling crosses upper boundary once
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})

	if len(fires) != 1 {
		t.Fatalf("expected 1 callback fire, got %d: %+v", len(fires), fires)
	}
	if fires[0].transient != LosingControl {
		t.Errorf("transient = %v, want LosingControl", fires[0].transient)
	}
	if fires[0].from != Neutral || fires[0].to != Controlling {
		t.Errorf("from→to = %v→%v, want Neutral→Controlling", fires[0].from, fires[0].to)
	}
}

func TestBoundaryCallbackFiresTwiceForFullSpan(t *testing.T) {
	var fires []State
	RegisterBoundaryCrossCallback(func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
		fires = append(fires, transient)
	})
	defer RegisterBoundaryCrossCallback(nil)

	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	fires = nil
	// Controlling → Controlled crosses upper AND lower boundaries
	_ = m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss})

	if len(fires) != 2 {
		t.Fatalf("expected 2 callback fires, got %d: %v", len(fires), fires)
	}
	if fires[0] != LosingControl {
		t.Errorf("first fire = %v, want LosingControl (upper boundary first)", fires[0])
	}
	if fires[1] != BecomingControlled {
		t.Errorf("second fire = %v, want BecomingControlled (lower boundary second)", fires[1])
	}
}

func TestBoundaryCallbackNotFiredOnNoOpTransition(t *testing.T) {
	var fireCount int
	RegisterBoundaryCrossCallback(func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
		fireCount++
	})
	defer RegisterBoundaryCrossCallback(nil)

	m := NewMachine()
	// Neutral → Neutral (no-op, no boundary crossing)
	_ = m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftWin})

	if fireCount != 0 {
		t.Errorf("expected 0 callback fires for no-op, got %d", fireCount)
	}
}
```

- [ ] **Step 2: Run tests to verify they pass (T2 implementation should already support this)**

Run: `go test ./internal/state/control/ -run "TestBoundaryCallback" -count=1 -v`
Expected: PASS. The T2 implementation already wired the callback firing; these are pure test coverage for that behavior.

- [ ] **Step 3: Commit**

```bash
git add internal/state/control/control_test.go
git commit -m "test(control): chunk 4b-fixup-2 T3 — boundary callback coverage

Tests verify the same-tick boundary-cross callback fires exactly
once per boundary crossed: upper-only for single-step transitions
adjacent to Neutral, both boundaries (upper first, lower second)
for full-span Controlling→Controlled transitions, zero for no-op
transitions.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Character.Control field + init in NewCharacter

**Files:**
- Modify: `internal/characters/character.go`

- [ ] **Step 1: Audit Character struct for the field insertion point**

Read `internal/characters/character.go` and find the field definitions block. Locate where other state-machine pointers live (likely near `Position`, `Activity`, `Awareness`).

Also locate `NewCharacter` or equivalent constructor — that's where the machine gets initialized.

Also locate `ResetForMobInstance` (used in T16 of chunk 4b-fixup; sets Mob-instance fields to default).

- [ ] **Step 2: Add Control field to Character struct**

After the existing state-machine pointer fields (Position, Activity, Awareness), add:

```go
// Control is the per-character ControlLevel state machine
// (chunk 4b-fixup-2). Tracks dominance within a grapple — 5 states:
// 3 stable (Controlling/Neutral/Controlled) + 2 transient
// (LosingControl/BecomingControlled) entered same-tick during
// boundary crossings. Resets to Neutral on grapple exit.
Control *control.Machine `yaml:"-"` // not persisted; recomputed at boot
```

Add the import:

```go
import (
    // ...existing imports...
    "github.com/GoMudEngine/GoMud/internal/state/control"
)
```

- [ ] **Step 3: Initialize Control in NewCharacter**

In `NewCharacter` (or equivalent constructor), after the other machine inits:

```go
c.Control = control.NewMachine()
```

- [ ] **Step 4: Reset Control in ResetForMobInstance**

In `ResetForMobInstance`, alongside existing per-instance resets:

```go
c.Control = control.NewMachine()
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: clean build. (No tests to fail at this point; integration is later.)

- [ ] **Step 6: Run existing characters package tests to ensure no regression**

Run: `go test ./internal/characters/ -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/characters/character.go
git commit -m "feat(characters): chunk 4b-fixup-2 T4 — Character.Control field

Adds per-character Control *control.Machine field, initialized in
NewCharacter and reset in ResetForMobInstance. Lives alongside
Position/Activity/Awareness machine pointers. Not persisted to
YAML (state is recomputed at boot per the existing per-character
state-machine convention).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: GrappleData.IsAggressor field + ApplyGrappleResult passes attacker info

**Files:**
- Modify: `internal/state/position/position.go` (add IsAggressor field)
- Modify: `internal/combat/grapple.go` (set IsAggressor in ApplyGrappleResult)

- [ ] **Step 1: Add IsAggressor field to GrappleData**

In `internal/state/position/position.go`, find the `GrappleData` struct definition and add:

```go
type GrappleData struct {
    Reason  string
    Partner state.ActorRef
    // IsControllerRole bool  // ← this field is sunset in T16; for now, leave it
    IsAggressor bool // true if this side initiated the grapple; used as drift-roll tiebreaker in symmetric positions
}
```

- [ ] **Step 2: Update ApplyGrappleResult to pass attacker info via TransitionPair**

In `internal/combat/grapple.go` around line 105 (the `ApplyGrappleResult` function), modify the TransitionPair call site to also stamp `IsAggressor`. This requires a new TransitionPair signature change (T6 implements it); for T5, leave a `// TODO: T6 wire IsAggressor` comment and pass the attacker → controller arg consistently.

Replace any local GrappleData construction (currently in TransitionPair) with a pattern that takes the aggressor info from the caller. The actual wiring happens in T6; T5 just lays the field + sets the source-of-truth for who is the attacker.

The simplest approach for T5: add a new helper:

```go
// markAggressor sets IsAggressor=true on the attacker side's
// GrappleData after a successful TransitionPair. Called from
// ApplyGrappleResult immediately after TransitionPair returns.
// Chunk 4b-fixup-2 T5: pre-wire for T6's full integration.
func markAggressor(attacker *characters.Character) {
    if attacker == nil || attacker.Position == nil {
        return
    }
    d, ok := attacker.Position.GrappleData()
    if !ok {
        return
    }
    d.IsAggressor = true
    // Re-store the modified GrappleData (GrappleData() returned a
    // value type; we need to push it back through a setter).
    attacker.Position.SetGrappleData(d)
}
```

Verify `SetGrappleData` exists on the Position machine — if not, add it as a setter in `internal/state/position/position.go`.

Call `markAggressor(attacker)` immediately after the TransitionPair call inside `ApplyGrappleResult`.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Quick test that ApplyGrappleResult preserves attacker info**

If `internal/combat/grapple_test.go` exists, add a smoke test verifying that after `ApplyGrappleResult`, the attacker's GrappleData has `IsAggressor=true` and the defender has `IsAggressor=false`. If no test file exists, skip (the integration smoke in T19 catches this).

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/position.go internal/combat/grapple.go
git commit -m "feat(position+combat): chunk 4b-fixup-2 T5 — IsAggressor field + ApplyGrappleResult wire

Adds GrappleData.IsAggressor bool. ApplyGrappleResult marks the
attacker side as aggressor immediately after TransitionPair returns.

Used as a tiebreaker for the drift roll's attacker-arg in symmetric
positions (Clinch round 1, etc.) where both sides start at the same
ControlLevel state. Orthogonal to the controller/controlled role
which is determined by position semantics.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: TransitionPair refactor — drop IsControllerRole, init ControlLevel state

**Files:**
- Modify: `internal/state/position/pair.go`

- [ ] **Step 1: Audit current TransitionPair behavior**

Read `internal/state/position/pair.go` end-to-end. Note:
- The `IsControllerRole: !isSymmetricGrapple(target)` line at ~121 (the bug source).
- How GrappleData is constructed for controller and controlled.
- The `isSymmetricGrapple` helper.

- [ ] **Step 2: Replace IsControllerRole stamping with ControlLevel state initialization**

Find the GrappleData literals in TransitionPair (around line 119-126). Update:

```go
// Before:
ctrlData := GrappleData{
    Partner:          cdRef,
    IsControllerRole: !isSymmetricGrapple(target),
}
cdData := GrappleData{
    Partner:          ctrlRef,
    IsControllerRole: false,
}

// After:
ctrlData := GrappleData{
    Partner: cdRef,
    // IsAggressor set by ApplyGrappleResult after TransitionPair returns
    // (chunk 4b-fixup-2 T5).
}
cdData := GrappleData{
    Partner: ctrlRef,
}
```

After both sides successfully transition (post the existing rollback logic), initialize ControlLevel for each side:

```go
// Chunk 4b-fixup-2 T6: initialize ControlLevel state per position
// symmetry class.
if isSymmetricGrapple(target) {
    _ = controller.Control.TransitionToNeutral(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
    _ = controlled.Control.TransitionToNeutral(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
} else {
    _ = controller.Control.TransitionToControlling(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
    _ = controlled.Control.TransitionToControlled(state.TransitionReason{Trigger: control.TriggerGrappleEnter})
}
```

Add the import:

```go
import (
    // ...existing imports...
    "github.com/GoMudEngine/GoMud/internal/state/control"
)
```

Note: `controller` and `controlled` are the GrappleActor interface, not *characters.Character directly. You may need to add a `GetControl()` method to the GrappleActor interface and have Character implement it. Add the interface method:

```go
// In pair.go GrappleActor interface:
type GrappleActor interface {
    // ...existing methods...
    GetControl() *control.Machine
}
```

And in `internal/characters/character.go`:

```go
func (c *Character) GetControl() *control.Machine { return c.Control }
```

- [ ] **Step 3: Handle Standing target — reset ControlLevel to Neutral on grapple exit**

When `target == Standing`, the grapple ends. Both sides' ControlLevel should reset to Neutral. Update the Standing-target branch of TransitionPair:

```go
if target == Standing {
    // ...existing transition logic...
    // Chunk 4b-fixup-2 T6: reset ControlLevel on grapple exit.
    if controller.GetControl() != nil {
        _ = controller.GetControl().TransitionToNeutral(state.TransitionReason{Trigger: control.TriggerGrappleExit})
    }
    if controlled.GetControl() != nil {
        _ = controlled.GetControl().TransitionToNeutral(state.TransitionReason{Trigger: control.TriggerGrappleExit})
    }
    return nil
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build. (Don't worry about all tests passing yet — IsControllerRole readers will fail until T7 refactors them.)

If you get a compile error like "GetControl undefined on GrappleActor mock" in tests, that's expected — add the method to the mock or leave the build broken and fix in T7.

- [ ] **Step 5: Run position package tests; expect some to fail**

Run: `go test ./internal/state/position/ -count=1`
Expected: SOME PASS, some may FAIL — tests asserting IsControllerRole will fail until T7. Note which fail in the commit message.

- [ ] **Step 6: Commit**

```bash
git add internal/state/position/pair.go internal/characters/character.go
git commit -m "feat(position): chunk 4b-fixup-2 T6 — TransitionPair inits ControlLevel state

GrappleData no longer stamps IsControllerRole (still present as
field for T7 to read during refactor; deleted in T16).

After TransitionPair successfully moves both sides to the target
position, both sides' Character.Control machine is initialized via
TransitionToNeutral (symmetric) or TransitionToControlling/Controlled
(asymmetric). On Standing target (grapple exit), both reset to Neutral.

GrappleActor interface gains GetControl(); Character implements it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Refactor Machine.IsController() / IsBeingControlled() to delegate to Character.Control

**Files:**
- Modify: `internal/state/position/position.go`

- [ ] **Step 1: Find and rewrite IsController / IsBeingControlled**

In `internal/state/position/position.go`, locate `Machine.IsController()` and `Machine.IsBeingControlled()`. They currently read `GrappleData.IsControllerRole`. The refactor: they should delegate to the owning character's Control machine.

The Position.Machine doesn't have a back-pointer to the Character. Two options:
- **(A)** Add a `Control() *control.Machine` accessor on Position.Machine that's wired up at NewCharacter time.
- **(B)** Move IsController/IsBeingControlled OFF the Position machine and ONTO the Character struct.

Option B is cleaner — the question "is this character a grapple controller?" is naturally a Character concern, and Character has access to both its Position and Control.

Refactor: delete IsController / IsBeingControlled from Position.Machine. Add equivalents on Character:

In `internal/characters/character.go`:

```go
// IsController returns true if this character is the "controller"
// side of a grapple — i.e., Control state is Controlling. Replaces
// chunk-4b-fixup's IsControllerRole bool read.
func (c *Character) IsController() bool {
    if c.Control == nil {
        return false
    }
    return c.Control.State() == control.Controlling
}

// IsBeingControlled returns true if this character is being
// dominated in a grapple — i.e., Control state is Controlled.
func (c *Character) IsBeingControlled() bool {
    if c.Control == nil {
        return false
    }
    return c.Control.State() == control.Controlled
}
```

- [ ] **Step 2: Update all callers of the old Position.IsController() / IsBeingControlled()**

Run:

```
grep -rn "Position.IsController\|Position.IsBeingControlled\|\.Position\.\(Machine\)\.\(IsController\|IsBeingControlled\)" internal/
```

For each match, change to call the new Character method:

```
// Before:
if c.Position.IsController() { ... }
// After:
if c.IsController() { ... }
```

If a caller has only the Position pointer (not the Character), this is a sign the caller has the wrong type — likely should be holding a Character pointer instead. Address those individually.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Run all tests**

Run: `go test ./... -count=1 2>&1 | grep -E "^(FAIL|ok)" | tail -30`
Expected: ALL PASS (or only failures unrelated to this work — e.g., IsControllerRole field still exists and is read in some tests; that's T16's cleanup).

If `IsControllerRole`-reading tests fail, leave them — T16 deletes the field and updates them.

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/position.go internal/characters/character.go
git commit -m "refactor(position): chunk 4b-fixup-2 T7 — IsController moves to Character

IsController() / IsBeingControlled() deleted from Position.Machine
and reimplemented on Character. They now read Character.Control.State()
instead of GrappleData.IsControllerRole.

Conceptually: the question 'is this character a grapple controller?'
is a Character concern, not a Position concern. Position is geometric;
role is mechanical.

All callers updated. IsControllerRole bool field still present in
GrappleData (deleted in T16 after final cleanup).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Pair iteration fix in processGrappleTick (the shippable bug fix)

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`

This task is the load-bearing bug fix from the spec §1. It can ship standalone if the ControlLevel FSM work is delayed.

- [ ] **Step 1: Audit current processGrappleTick implementation**

Read `internal/hooks/Position_GrappleTick.go`. Current shape:

```go
func processGrappleTick(e events.Event) events.ListenerReturn {
    for _, u := range users.GetAllActiveUsers() {
        if u.Character == nil || !u.Character.IsController() { continue }
        partner := resolvePartner(u.Character)
        if partner == nil { continue }
        processGrapplePair(u.Character, partner)
    }
    for _, mobInstId := range mobs.GetAllMobInstanceIds() {
        m := mobs.GetInstance(mobInstId)
        if m == nil || !m.Character.IsController() { continue }
        // ...same pattern...
    }
    return events.Continue
}
```

The bug: `IsController()` filter skips both sides in symmetric Clinch positions.

- [ ] **Step 2: Rewrite to iterate pairs with deduplication**

Replace processGrappleTick with:

```go
func processGrappleTick(e events.Event) events.ListenerReturn {
    seen := map[state.ActorRef]bool{}

    for _, u := range users.GetAllActiveUsers() {
        if u == nil || u.Character == nil {
            continue
        }
        myRef := state.ActorRef{UserId: u.UserId}
        if seen[myRef] {
            continue
        }
        if u.Character.Position == nil {
            continue
        }
        if !u.Character.Position.IsGrappling() {
            continue
        }
        partner := resolvePartner(u.Character)
        if partner == nil {
            continue
        }
        if err := position.ValidateGrapplePair(u.Character, partner); err != nil {
            mudlog.Warn("Position_GrappleTick: invalid pair", "user", u.UserId, "err", err)
            continue
        }
        // Mark both sides as seen so we don't double-process.
        partnerRef := state.ActorRef{
            UserId:        partner.GetUserId(),
            MobInstanceId: partner.GetMobInstanceId(),
        }
        seen[myRef] = true
        seen[partnerRef] = true

        processGrapplePair(u.Character, partner)
    }

    for _, mobInstId := range mobs.GetAllMobInstanceIds() {
        m := mobs.GetInstance(mobInstId)
        if m == nil {
            continue
        }
        myRef := state.ActorRef{MobInstanceId: m.InstanceId}
        if seen[myRef] {
            continue
        }
        if m.Character.Position == nil {
            continue
        }
        if !m.Character.Position.IsGrappling() {
            continue
        }
        partner := resolvePartner(&m.Character)
        if partner == nil {
            continue
        }
        if err := position.ValidateGrapplePair(&m.Character, partner); err != nil {
            mudlog.Warn("Position_GrappleTick: invalid pair", "mob", m.InstanceId, "err", err)
            continue
        }
        partnerRef := state.ActorRef{
            UserId:        partner.GetUserId(),
            MobInstanceId: partner.GetMobInstanceId(),
        }
        seen[myRef] = true
        seen[partnerRef] = true

        processGrapplePair(&m.Character, partner)
    }

    return events.Continue
}
```

Key changes:
- No more `IsController()` filter.
- New `seen` map tracks processed pairs; BOTH sides marked seen.
- Pair iteration runs `processGrapplePair` exactly once per pair per round.
- Replaces the `IsController()` filter with `IsGrappling()` (already defined in Position) — the only filter is "in a grapple at all."

- [ ] **Step 3: Inside processGrapplePair, determine which side is the "attacker arg" for the drift roll**

The drift roll's `dice.OpposedRollStat(attackerScore, defenderScore)` needs to know which side is attacker. Currently this was implicit (the loop always passed the IsController side as attacker). Now we need to pick.

Add a helper at the top of processGrapplePair:

```go
// determineDriftAttacker picks which side is the drift-roll's
// attacker-arg. Priority:
//   1. Whoever has the more controller-leaning Control state.
//   2. Tiebreaker: whoever has IsAggressor=true.
//   3. Final fallback: lhs (the iteration order side).
func determineDriftAttacker(lhs, rhs *characters.Character) (attacker, defender *characters.Character) {
    if lhs.Control == nil || rhs.Control == nil {
        return lhs, rhs
    }
    lhsRank := controlRank(lhs.Control.State())
    rhsRank := controlRank(rhs.Control.State())
    if lhsRank < rhsRank {
        // lhs is more controller-leaning
        return lhs, rhs
    }
    if rhsRank < lhsRank {
        return rhs, lhs
    }
    // Tie: prefer aggressor
    if lhs.Position != nil {
        if d, ok := lhs.Position.GrappleData(); ok && d.IsAggressor {
            return lhs, rhs
        }
    }
    if rhs.Position != nil {
        if d, ok := rhs.Position.GrappleData(); ok && d.IsAggressor {
            return rhs, lhs
        }
    }
    // Last resort: caller order
    return lhs, rhs
}

// controlRank: 0 = Controlling (most attacker-favored), 1 = Neutral,
// 2 = Controlled. Transient states get the rank of their "from" stable
// state (defensive — transients shouldn't be observed between ticks).
func controlRank(s control.State) int {
    switch s {
    case control.Controlling:
        return 0
    case control.LosingControl:
        return 0 // closest to Controlling
    case control.Neutral:
        return 1
    case control.BecomingControlled:
        return 1 // closest to Neutral
    case control.Controlled:
        return 2
    }
    return 1
}
```

Modify the top of `processGrapplePair` to invoke this:

```go
func processGrapplePair(lhs, rhs *characters.Character) {
    controller, controlled := determineDriftAttacker(lhs, rhs)
    // ...rest unchanged: compute scores using controller/controlled as attacker/defender args...
}
```

(Note: the existing `processGrapplePair` signature is `(controller, controlled)`. Keep it; just compute these via the new helper before the existing logic runs.)

Actually — the existing signature is already `processGrapplePair(controller, controlled *characters.Character)`. Refactor to take generic `(lhs, rhs)` and pick inside, OR add a wrapper that the new iteration calls. The wrapper approach is less invasive:

```go
// processGrapplePairFromIteration is called by processGrappleTick;
// determines which side is the controller arg, then delegates to
// processGrapplePair (existing).
func processGrapplePairFromIteration(lhs, rhs *characters.Character) {
    controller, controlled := determineDriftAttacker(lhs, rhs)
    processGrapplePair(controller, controlled)
}
```

Update the two iteration call sites in `processGrappleTick` to call `processGrapplePairFromIteration(...)` instead of `processGrapplePair(...)`.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5: Run tests for hooks + position packages**

Run: `go test ./internal/hooks/ ./internal/state/position/ ./internal/state/control/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go
git commit -m "fix(hooks): chunk 4b-fixup-2 T8 — pair iteration in processGrappleTick

Fixes the chunk-4b-fixup regression where symmetric Clinch grapples
never had their drift roll fire because both sides had
IsControllerRole=false and the IsController() filter skipped both.

processGrappleTick now iterates pairs (deduped via a seen map),
running processGrapplePair exactly once per pair per round. No
filter — every character in a grapple gets processed.

Inside processGrapplePair, the drift-roll attacker-arg is determined
by determineDriftAttacker: most controller-leaning Control state
wins; tiebreaker is IsAggressor; final fallback is iteration order.

This task alone resolves the Clinch tick-skipping bug surfaced by
the 2026-05-18 AI smoke test. Subsequent tasks add the new
ControlLevel shift logic + gradient messaging on top.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: ControlLevel shift logic in processGrapplePair

**Files:**
- Modify: `internal/hooks/Position_GrappleTick.go`

- [ ] **Step 1: Add shift function**

After `processGrapplePair`'s existing drift roll + stamina cost logic, add a call to apply ControlLevel shifts based on the z magnitude:

```go
// applyControlShift updates both sides' ControlLevel state based on
// drift roll outcome. Chunk 4b-fixup-2 T9 spec §5:
//   |z| < 0.5      → no shift
//   0.5 ≤ |z| < 1.5 → 1 step
//   |z| ≥ 1.5      → 2 steps
// Winner shifts toward Controlling; loser shifts toward Controlled.
func applyControlShift(controller, controlled *characters.Character, z float64) {
    absZ := z
    if absZ < 0 {
        absZ = -absZ
    }
    var steps int
    switch {
    case absZ < 0.5:
        return // no shift
    case absZ < 1.5:
        steps = 1
    default:
        steps = 2
    }

    if z > 0 {
        // Controller wins — shifts toward Controlling
        shiftControl(controller, +steps)
        shiftControl(controlled, -steps)
    } else {
        // Controlled wins — shifts toward Controlling
        shiftControl(controlled, +steps)
        shiftControl(controller, -steps)
    }
}

// shiftControl moves the given character's Control state by `steps`
// stable-state ranks. Positive steps = toward Controlling; negative
// = toward Controlled. Caps at the endpoints.
func shiftControl(c *characters.Character, steps int) {
    if c == nil || c.Control == nil {
        return
    }
    currentRank := stableRankFromControl(c.Control.State())
    newRank := currentRank - steps // toward Controlling means lower rank
    if newRank < 0 {
        newRank = 0
    }
    if newRank > 2 {
        newRank = 2
    }
    if newRank == currentRank {
        return
    }
    reason := state.TransitionReason{Trigger: control.TriggerDriftLoss}
    if newRank < currentRank {
        reason.Trigger = control.TriggerDriftWin
    }
    switch newRank {
    case 0:
        _ = c.Control.TransitionToControlling(reason)
    case 1:
        _ = c.Control.TransitionToNeutral(reason)
    case 2:
        _ = c.Control.TransitionToControlled(reason)
    }
}

func stableRankFromControl(s control.State) int {
    switch s {
    case control.Controlling, control.LosingControl:
        return 0
    case control.Neutral:
        return 1
    case control.BecomingControlled, control.Controlled:
        return 2
    }
    return 1
}
```

- [ ] **Step 2: Wire applyControlShift into processGrapplePair**

In `processGrapplePair`, after the existing stamina cost call and BEFORE the messaging emit (which fires last), add:

```go
applyControlShift(controller, controlled, z)
```

The boundary-cross callbacks fire automatically during these transitions; gradient messaging is wired up in T13 to receive them.

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/ -count=1`
Expected: clean build + tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Position_GrappleTick.go
git commit -m "feat(hooks): chunk 4b-fixup-2 T9 — ControlLevel shift logic

processGrapplePair now applies per-round ControlLevel shifts based
on drift z magnitude:
  |z| < 0.5      → no shift
  0.5 ≤ |z| < 1.5 → 1 step
  |z| ≥ 1.5      → 2 steps
Winner shifts toward Controlling; loser shifts toward Controlled.
Capped at the gradient endpoints.

Boundary-cross callbacks fire automatically during the same-tick
chain (TransitionTo*); gradient messaging consumers land in T13.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: GradientTriad type + Library.Gradients field + validator extension

**Files:**
- Modify: `internal/grapplemessaging/loader.go`
- Modify: `internal/grapplemessaging/loader_test.go`

- [ ] **Step 1: Write failing tests for GradientTriad parsing + validation**

Append to `internal/grapplemessaging/loader_test.go`:

```go
func TestGradientTriadParse(t *testing.T) {
	path := writeTempYAML(t, `
advancements: {}
degradations: {}
reversals: {}
escapes: {}
holds: {}
striking_apex: {}
gradients:
  upper_boundary_down:
    self:
      - "Your grip slips."
    partner:
      - "{controllerName}'s grip slips."
    observers:
      - "{controllerName}'s grip on {controlledName} slips."
`)
	lib, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	g, ok := lib.Gradients["upper_boundary_down"]
	if !ok {
		t.Fatal("missing upper_boundary_down key")
	}
	if len(g.Self) != 1 || len(g.Partner) != 1 || len(g.Observers) != 1 {
		t.Errorf("triad lengths: self=%d partner=%d obs=%d, want 1/1/1",
			len(g.Self), len(g.Partner), len(g.Observers))
	}
}

func TestValidateCompletenessGradientsMissing(t *testing.T) {
	lib := &Library{
		Advancements: map[string]TemplateTriad{},
		Degradations: map[string]TemplateTriad{},
		Reversals:    map[string]TemplateTriad{},
		Escapes:      map[string]TemplateTriad{},
		Holds:        map[string]TemplateTriad{},
		StrikingApex: map[string][]string{},
		Gradients:    map[string]GradientTriad{},
	}
	errs := ValidateCompleteness(lib)
	foundGradientErr := false
	for _, e := range errs {
		if e.Error() != "" && contains(e.Error(), "gradients:") {
			foundGradientErr = true
			break
		}
	}
	if !foundGradientErr {
		t.Error("empty Gradients should produce a 'gradients: missing key' error")
	}
}

// contains is a tiny strings.Contains alias to avoid an import dependency
// shuffle in this file.
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/grapplemessaging/ -run "TestGradient|TestValidateCompletenessGradients" -count=1`
Expected: FAIL with undefined Gradients / GradientTriad.

- [ ] **Step 3: Add GradientTriad + Library.Gradients to loader.go**

In `internal/grapplemessaging/loader.go`:

```go
// GradientTriad holds the three speaker-variant template lists for
// a gradient (ControlLevel boundary-crossing) event. Self is shown
// to the character whose state changed; partner is shown to the
// other side of the grapple; observers is broadcast to the room.
//
// Different from TemplateTriad's controller/controlled/observers
// semantics — gradients fire per-character (self), not per-role.
type GradientTriad struct {
    Self      []string `yaml:"self"`
    Partner   []string `yaml:"partner"`
    Observers []string `yaml:"observers"`
}

// Library extended:
type Library struct {
    Advancements map[string]TemplateTriad  `yaml:"advancements"`
    Degradations map[string]TemplateTriad  `yaml:"degradations"`
    Reversals    map[string]TemplateTriad  `yaml:"reversals"`
    Escapes      map[string]TemplateTriad  `yaml:"escapes"`
    Holds        map[string]TemplateTriad  `yaml:"holds"`
    StrikingApex map[string][]string       `yaml:"striking_apex"`
    Gradients    map[string]GradientTriad  `yaml:"gradients"` // NEW
}
```

In `Load`, initialize the nil map:

```go
if lib.Gradients == nil {
    lib.Gradients = map[string]GradientTriad{}
}
```

Add the required-keys list:

```go
// RequiredGradientKeys: 4 boundary-direction keys.
var RequiredGradientKeys = []string{
    "upper_boundary_down", // Controlling → Neutral
    "upper_boundary_up",   // Neutral → Controlling
    "lower_boundary_down", // Neutral → Controlled
    "lower_boundary_up",   // Controlled → Neutral
}
```

Extend `ValidateCompleteness`:

```go
// (add this block to ValidateCompleteness, alongside the existing
// checks for advancements/degradations/etc.)

for _, key := range RequiredGradientKeys {
    triad, ok := lib.Gradients[key]
    if !ok {
        errs = append(errs, fmt.Errorf("gradients: missing key %q", key))
        continue
    }
    if len(triad.Self) < MinTemplatesPerSpeaker {
        errs = append(errs, fmt.Errorf("gradients.%s.self: %d templates, need >= %d",
            key, len(triad.Self), MinTemplatesPerSpeaker))
    }
    if len(triad.Partner) < MinTemplatesPerSpeaker {
        errs = append(errs, fmt.Errorf("gradients.%s.partner: %d templates, need >= %d",
            key, len(triad.Partner), MinTemplatesPerSpeaker))
    }
    if len(triad.Observers) < MinTemplatesPerSpeaker {
        errs = append(errs, fmt.Errorf("gradients.%s.observers: %d templates, need >= %d",
            key, len(triad.Observers), MinTemplatesPerSpeaker))
    }
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/grapplemessaging/ -count=1 -v`
Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/grapplemessaging/loader.go internal/grapplemessaging/loader_test.go
git commit -m "feat(grapplemessaging): chunk 4b-fixup-2 T10 — GradientTriad + validator

New GradientTriad struct (Self/Partner/Observers) parallel to
TemplateTriad. Library gains Gradients map. ValidateCompleteness
extended with RequiredGradientKeys (4 boundary-direction keys:
upper_boundary_down/up, lower_boundary_down/up).

Templates land in T11; observer wiring in T13.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Author ~36 gradient templates

**Files:**
- Modify: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`
- Modify: `internal/grapplemessaging/loader_test.go`

This is a PROSE AUTHORING task. 4 keys × 3 speakers × 3 templates min = ~36 templates minimum.

- [ ] **Step 1: Add the `gradients:` block to the YAML**

Open `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`. Append (at the bottom, after `striking_apex:`):

```yaml
gradients:
  # upper_boundary_down: Controlling → Neutral (your dominance is slipping)
  upper_boundary_down:
    self:
      - "Your control slips — {controlledName} squirms free of the lock you had."
      - "You feel your grip loosen; {controlledName}'s frantic movements pry leverage back from you."
      - "The dominant position shifts under you — you're no longer in control of this exchange."
    partner:
      - "{controllerName}'s dominance slips. You feel space opening where there was none."
      - "{controllerName} can't hold the pressure — your bridges and frames are working."
      - "The lock {controllerName} had on you starts to give. You're scrambling back to even."
    observers:
      - "{controllerName} loses their controlling grip on {controlledName}; the exchange evens out."
      - "{controlledName} squirms and frames; {controllerName}'s dominance slips."
      - "The pressure {controllerName} had on {controlledName} fades — neutral footing returns."

  # upper_boundary_up: Neutral → Controlling (you're asserting dominance)
  upper_boundary_up:
    self:
      - "You feel the position click — your weight settles right and you assert full control."
      - "Your leverage finally takes; you're now in the dominant position."
      - "A surge of pressure and you've got it — you're the controlling side now."
    partner:
      - "{controllerName} settles their weight and the position locks against you — they're in control."
      - "You feel {controllerName} establish the dominant grip. The fight tightens."
      - "{controllerName} times the pressure right and asserts control over you."
    observers:
      - "{controllerName} establishes dominance over {controlledName} — control settles in their favor."
      - "{controllerName}'s pressure finally takes; they assume the controlling position."
      - "{controllerName} locks in dominance over {controlledName}."

  # lower_boundary_down: Neutral → Controlled (you're being overwhelmed)
  lower_boundary_down:
    self:
      - "Pressure crashes down — {controllerName} sinks the hooks and you're pinned flat."
      - "Your space evaporates. {controllerName}'s weight settles heavy and you can't generate leverage."
      - "The position turns on you — {controllerName} establishes full control and you're stuck."
    partner:
      - "Your weight settles and {controlledName} flattens out under you — full control."
      - "You drive the pressure home; {controlledName} can't find leverage anymore."
      - "{controlledName}'s frame collapses; you've got them fully pinned now."
    observers:
      - "{controllerName} drives pressure home and {controlledName} is pinned flat — full control established."
      - "{controlledName}'s leverage gives way; {controllerName} settles into dominant control."
      - "{controllerName} locks {controlledName} down — the position is fully controlled now."

  # lower_boundary_up: Controlled → Neutral (you create space)
  lower_boundary_up:
    self:
      - "You create space — a bridge and a hip-escape pries {controllerName}'s weight off your sternum."
      - "Your frame catches and you wrench free of the worst of the pressure. Breathing room."
      - "You scramble back to a fighting position; {controllerName} no longer has you fully pinned."
    partner:
      - "{controlledName} bridges and creates space — your dominant control slips into a scramble."
      - "{controlledName}'s frame pries your weight off; the pin starts to break."
      - "You feel {controlledName} earn back leverage — they're no longer fully pinned."
    observers:
      - "{controlledName} bridges and scrambles back from {controllerName}'s control — the position evens out."
      - "{controlledName} earns space against {controllerName}; the pin breaks back to neutral."
      - "{controllerName}'s control on {controlledName} fails; they create breathing room."
```

That's 4 × 9 = 36 templates exactly.

- [ ] **Step 2: Add a production-library guard test**

Append to `internal/grapplemessaging/loader_test.go`:

```go
func TestProductionLibraryGradientsComplete(t *testing.T) {
    lib, err := Load("../../_datafiles/world/dogmud/messaging/grapple_outcomes.yaml")
    if err != nil {
        t.Fatalf("Load prod library: %v", err)
    }
    for _, key := range RequiredGradientKeys {
        triad, ok := lib.Gradients[key]
        if !ok {
            t.Errorf("missing gradient key: %s", key)
            continue
        }
        if len(triad.Self) < MinTemplatesPerSpeaker {
            t.Errorf("%s.self: %d < %d", key, len(triad.Self), MinTemplatesPerSpeaker)
        }
        if len(triad.Partner) < MinTemplatesPerSpeaker {
            t.Errorf("%s.partner: %d < %d", key, len(triad.Partner), MinTemplatesPerSpeaker)
        }
        if len(triad.Observers) < MinTemplatesPerSpeaker {
            t.Errorf("%s.observers: %d < %d", key, len(triad.Observers), MinTemplatesPerSpeaker)
        }
    }
}
```

- [ ] **Step 3: Run the guard test**

Run: `go test ./internal/grapplemessaging/ -run "TestProductionLibraryGradientsComplete" -count=1 -v`
Expected: PASS.

Also re-run the full library validation to ensure nothing else broke:

Run: `go test ./internal/grapplemessaging/ -count=1`
Expected: ALL PASS.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml internal/grapplemessaging/loader_test.go
git commit -m "content(messaging): chunk 4b-fixup-2 T11 — gradient templates (36)

4 boundary-direction keys × 9 templates each = 36 gradient templates.
Self/Partner/Observers speaker variants. MMA/BJJ vocabulary per the
chunk-4b-fixup §7.2 rubric. Realism review pass in T12.

Boundaries:
- upper_boundary_down: Controlling → Neutral (dominance slips)
- upper_boundary_up: Neutral → Controlling (you assert control)
- lower_boundary_down: Neutral → Controlled (you're overwhelmed)
- lower_boundary_up: Controlled → Neutral (you create space)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Realism sanity-check pass on gradient templates

**Files (review-only, possibly small revisions):**
- `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml` (gradients section only)

- [ ] **Step 1: Dispatch a fresh subagent for the realism review (same pattern as chunk 4b-fixup T14)**

Dispatch prompt:

```
You are a fresh reviewer with no prior context.

Review every template in the `gradients:` section of
_datafiles/world/dogmud/messaging/grapple_outcomes.yaml against
this question: "Does this read like a real description of
grappling state changes?"

A reader familiar with MMA/BJJ should be able to picture each
line without cringing at:
- Anatomical impossibilities or made-up moves
- Wrong direction for the boundary (e.g., upper_boundary_down
  templates should describe LOSING dominance — if a template
  describes asserting/gaining control, it's in the wrong key)
- Position-of-the-bodies errors (the boundary state changes
  apply to ALL positions — Clinch, Mount, SC, etc. — so templates
  shouldn't lock in a specific position's vocabulary unless it's
  generic enough to fit any)
- Wrong subject for the speaker variant (self templates should
  always describe what THIS character experiences; partner
  templates should always be about the OTHER side)
- Tone inconsistency

For each issue found:
- Location (gradients.{key}.{speaker}[{index}])
- Quote the line
- Describe the problem
- Suggest a revised line

Approve templates that pass with no comment.

Do NOT modify the YAML yourself. Report only.
```

- [ ] **Step 2: Apply any reviewer revisions**

Edit the YAML for each flagged issue. Use Edit tool for surgical replacements.

- [ ] **Step 3: Re-run validator + guard tests**

Run: `go test ./internal/grapplemessaging/ -count=1`
Expected: ALL PASS.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/messaging/grapple_outcomes.yaml
git commit -m "content(messaging): chunk 4b-fixup-2 T12 — gradient realism revisions

Fresh-subagent review per spec §7 (mirrors chunk-4b-fixup §7.6
pattern). N templates revised for [list specific issues]. No
template counts change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

(If no revisions needed, commit only a note: "Review passed with no revisions needed; commit is empty marker for plan tracking.")

---

## Task 13: Gradient messaging observer wiring

**Files:**
- Modify: `internal/grapplemessaging/render.go` (add gradient render helper)
- Modify: `internal/hooks/Position_GrappleTick.go` (register the boundary-cross callback)

- [ ] **Step 1: Add gradient render helper in grapplemessaging/render.go**

Append to `internal/grapplemessaging/render.go`:

```go
// PickGradientTemplate picks a template from a gradient triad's
// speaker list with cooldown-aware variety. Same algorithm as
// PickTemplate but for the GradientTriad type.
func PickGradientTemplate(pool []string, cooldowns map[string]bool, keyPrefix string) string {
    return PickTemplate(pool, cooldowns, keyPrefix)
}
```

(The pool slicing is identical; PickTemplate is reusable.)

- [ ] **Step 2: Register the boundary-cross callback in Position_GrappleTick.go init()**

Open `internal/hooks/Position_GrappleTick.go`. In `init()` (or in `loadGrappleLib` per the existing sync.Once pattern), add:

```go
// Chunk 4b-fixup-2 T13: register the ControlLevel boundary-cross
// callback to fire gradient messaging when characters cross state
// boundaries.
control.RegisterBoundaryCrossCallback(func(self state.ActorRef, transient control.State, from control.State, to control.State, r state.TransitionReason) {
    emitGradientMessage(self, transient, from, to)
})
```

- [ ] **Step 3: Implement emitGradientMessage**

Add at the end of `internal/hooks/Position_GrappleTick.go`:

```go
// emitGradientMessage fires the appropriate gradient flavor when
// a character's ControlLevel state crosses a boundary same-tick.
// Resolves the gradient key from (transient, from, to) and dispatches
// the Self/Partner/Observers triad with name substitution + cooldown.
func emitGradientMessage(self state.ActorRef, transient control.State, from control.State, to control.State) {
    if grappleOutcomesLib == nil {
        return
    }
    key := gradientKeyForCrossing(transient, from, to)
    if key == "" {
        return
    }
    triad, ok := grappleOutcomesLib.Gradients[key]
    if !ok {
        mudlog.Warn("Position_GrappleTick: missing gradient key", "key", key)
        return
    }

    // Resolve self + partner characters.
    selfChar := characterFromRef(self)
    if selfChar == nil {
        return
    }
    partner := resolvePartner(selfChar)
    if partner == nil {
        return
    }

    if selfChar.PerGrappleMessageCooldowns == nil {
        selfChar.PerGrappleMessageCooldowns = map[string]bool{}
    }
    if partner.PerGrappleMessageCooldowns == nil {
        partner.PerGrappleMessageCooldowns = map[string]bool{}
    }

    selfName := characterDisplayName(selfChar)
    partnerName := characterDisplayName(partner)

    // The gradient is about `selfChar` — they crossed the boundary.
    // Templates use {controllerName} and {controlledName} substitution
    // for compatibility with existing render code, but the semantic
    // is self = the one who crossed.
    if msg := PickGradientTemplate(triad.Self, selfChar.PerGrappleMessageCooldowns, "gradient:"+key+":self"); msg != "" {
        sendToCharacter(selfChar, grapplemessaging.RenderTemplate(msg, selfName, partnerName))
    }
    if msg := PickGradientTemplate(triad.Partner, partner.PerGrappleMessageCooldowns, "gradient:"+key+":partner"); msg != "" {
        sendToCharacter(partner, grapplemessaging.RenderTemplate(msg, selfName, partnerName))
    }
    if msg := PickGradientTemplate(triad.Observers, selfChar.PerGrappleMessageCooldowns, "gradient:"+key+":obs"); msg != "" {
        broadcastToRoomExcluding(selfChar, partner, grapplemessaging.RenderTemplate(msg, selfName, partnerName))
    }
}

// PickGradientTemplate is a small alias for PickTemplate, kept as
// a separate name for grep-ability (it's the gradient path, not
// the outcome path).
func PickGradientTemplate(pool []string, cooldowns map[string]bool, keyPrefix string) string {
    return grapplemessaging.PickTemplate(pool, cooldowns, keyPrefix)
}

// gradientKeyForCrossing returns the YAML key for a boundary crossing.
// Spec §7:
//   upper_boundary_down: Controlling → Neutral
//   upper_boundary_up:   Neutral → Controlling
//   lower_boundary_down: Neutral → Controlled
//   lower_boundary_up:   Controlled → Neutral
func gradientKeyForCrossing(transient control.State, from control.State, to control.State) string {
    if transient == control.LosingControl {
        if from == control.Controlling && to == control.Neutral {
            return "upper_boundary_down"
        }
        if from == control.Neutral && to == control.Controlling {
            return "upper_boundary_up"
        }
    }
    if transient == control.BecomingControlled {
        if from == control.Neutral && to == control.Controlled {
            return "lower_boundary_down"
        }
        if from == control.Controlled && to == control.Neutral {
            return "lower_boundary_up"
        }
    }
    return ""
}

// characterFromRef resolves a state.ActorRef to its Character pointer.
// Helper for emitGradientMessage; mirrors the resolvePartner pattern.
func characterFromRef(ref state.ActorRef) *characters.Character {
    if ref.UserId > 0 {
        if u := users.GetByUserId(ref.UserId); u != nil {
            return u.Character
        }
    }
    if ref.MobInstanceId > 0 {
        if m := mobs.GetInstance(ref.MobInstanceId); m != nil {
            return &m.Character
        }
    }
    return nil
}
```

(Note: PickGradientTemplate is defined twice in the patch — once as a small alias in `render.go`, once locally in `Position_GrappleTick.go`. Pick ONE location only; delete the duplicate. Recommended: keep it in `render.go` and `import` from there in hooks. Adjust the hook code's `PickGradientTemplate` calls to use `grapplemessaging.PickGradientTemplate` instead.)

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/ ./internal/grapplemessaging/ ./internal/state/control/ -count=1`
Expected: clean build + tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/grapplemessaging/render.go internal/hooks/Position_GrappleTick.go
git commit -m "feat(hooks): chunk 4b-fixup-2 T13 — gradient messaging observer wiring

Position_GrappleTick init registers the ControlLevel
boundary-cross callback. emitGradientMessage resolves the
(transient, from, to) tuple to a gradient YAML key
(upper_boundary_down/up, lower_boundary_down/up) and dispatches
the Self/Partner/Observers triad with name substitution +
cooldown.

Gradient flavor now fires automatically when a character's
Control state crosses a boundary during a drift roll (T9).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: Refactor sub eligibility predicates + update callers

**Files:**
- Modify: `internal/state/position/submissions.go`
- Modify: `internal/hooks/Position_SubmissionTick.go`
- Modify: `internal/behaviortree/conditions_submission.go`

- [ ] **Step 1: Refactor IsTopSubEligible and IsBottomSubEligible signatures**

In `internal/state/position/submissions.go`:

```go
// Add import:
import (
    // ...
    "github.com/GoMudEngine/GoMud/internal/state/control"
)

// IsTopSubEligible returns true if the side at posState with
// ctrlState can attempt a top-position submission. Chunk 4b-fixup-2:
// requires Controlling state (not just "is in top position") — the
// side must have earned the dominant control to set up a submission.
func IsTopSubEligible(posState State, ctrlState control.State) bool {
    if ctrlState != control.Controlling {
        return false
    }
    switch posState {
    case Mount, SideControl, KneeOnBelly, NorthSouth, BackGround, Crucifix:
        return true
    }
    return false
}

// IsBottomSubEligible returns true if the side at posState with
// ctrlState can attempt a bottom-position submission. Requires
// Controlled state — the side must be on the receiving end of
// dominant control for bottom subs to make sense (e.g., guard
// triangle when being passed).
func IsBottomSubEligible(posState State, ctrlState control.State) bool {
    if ctrlState != control.Controlled {
        return false
    }
    switch posState {
    case Guard, HalfGuard:
        return true
    }
    return false
}
```

- [ ] **Step 2: Update caller in Position_SubmissionTick.go**

Find the calls to `IsTopSubEligible` and `IsBottomSubEligible` in `internal/hooks/Position_SubmissionTick.go`. Update from `gd.IsControllerRole` to `controller.Control.State()` (resp. `controlled.Control.State()`):

```go
// Before:
if position.IsTopSubEligible(posState, ctrlData.IsControllerRole) &&
    position.IsBottomSubEligible(posState, cdData.IsControllerRole) {
    ...
}

// After:
if position.IsTopSubEligible(posState, controller.Control.State()) &&
    position.IsBottomSubEligible(posState, controlled.Control.State()) {
    ...
}
```

(Adapt to exact variable names in the existing code; "controller" and "controlled" are typical.)

- [ ] **Step 3: Update caller in behaviortree/conditions_submission.go**

Same pattern. Find:

```go
if position.IsTopSubEligible(mob.Character.Position.State(), gd.IsControllerRole) { ... }
```

Replace with:

```go
if position.IsTopSubEligible(mob.Character.Position.State(), mob.Character.Control.State()) { ... }
```

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./... -count=1 2>&1 | grep -E "^(FAIL|ok)" | tail -20`
Expected: clean build, tests pass (or only IsControllerRole-field reads in tests still failing — T16 cleans those).

- [ ] **Step 5: Commit**

```bash
git add internal/state/position/submissions.go internal/hooks/Position_SubmissionTick.go internal/behaviortree/conditions_submission.go
git commit -m "refactor(position+hooks+btree): chunk 4b-fixup-2 T14 — sub eligibility uses control.State

IsTopSubEligible / IsBottomSubEligible now take control.State
instead of a bool. Top subs require Controlling; bottom subs
require Controlled. Position alone no longer gates sub windows
— the side must have earned the appropriate dominance state.

Callers updated in Position_SubmissionTick and
conditions_submission btree.

Behavior change: a controller in Mount whose Control has drifted
to Neutral can no longer initiate a top sub until they drift back
to Controlling.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: Update validation invariants

**Files:**
- Modify: `internal/state/position/validation.go`

- [ ] **Step 1: Replace invariant 4 with ControlLevel-state invariants**

In `internal/state/position/validation.go`, find the current "role-exclusivity" invariant (around line 104) and replace:

```go
// Chunk 4b-fixup-2 T15: ControlLevel state must be consistent with
// position type.
// - For asymmetric positions: not both at Controlling, not both at
//   Controlled (one must be more controller-leaning than the other,
//   OR one must be at Neutral as a transitional state during drift).
// - For symmetric positions: same invariant — drift roll outcomes
//   may shift sides asymmetrically, but a configuration where both
//   are at the same non-Neutral state is impossible.

if a == nil || b == nil || a.Control == nil || b.Control == nil {
    // Defensive: if Control wasn't initialized (shouldn't happen post
    // T4), skip this invariant rather than panicking.
    return nil
}
aState := a.Control.State()
bState := b.Control.State()

if aState == control.Controlling && bState == control.Controlling {
    return PairInvariantViolation{
        Invariant: "control-exclusivity",
        Description: fmt.Sprintf(
            "both sides at Controlling state in position %v (impossible: only one side can dominate)",
            stateA),
    }
}
if aState == control.Controlled && bState == control.Controlled {
    return PairInvariantViolation{
        Invariant: "control-exclusivity",
        Description: fmt.Sprintf(
            "both sides at Controlled state in position %v (impossible: only one side can be dominated)",
            stateA),
    }
}

return nil
```

The signature of `ConsistencyCheck` may need adjustment if it doesn't currently accept Character pointers. Add the parameter or change the call site as needed. This may require updating the caller in `Position_ConsistencyCheck.go`.

- [ ] **Step 2: Verify build + tests**

Run: `go build ./... && go test ./internal/state/position/ -count=1`
Expected: clean build + tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/state/position/validation.go
git commit -m "refactor(position): chunk 4b-fixup-2 T15 — ControlLevel invariants

Invariant 4 replaced with ControlLevel-state consistency check:
- Both sides at Controlling is impossible (only one dominates)
- Both sides at Controlled is impossible (only one is dominated)

Applies to both symmetric and asymmetric positions. Transient
states are not expected at consistency-check time (between ticks)
so they don't need explicit handling.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 16: Delete IsControllerRole bool field + final cleanup

**Files:**
- Modify: `internal/state/position/position.go` (delete IsControllerRole field)
- Audit: every test/file that referenced IsControllerRole — should be none after T7+T14

- [ ] **Step 1: Audit for remaining IsControllerRole references**

Run:

```
grep -rn "IsControllerRole" internal/ --include="*.go"
```

Should be zero matches in production code; possibly a few in test files. Inspect each.

- [ ] **Step 2: Delete the field from GrappleData**

In `internal/state/position/position.go`, find the GrappleData struct:

```go
type GrappleData struct {
    Reason       string
    Partner      state.ActorRef
    IsAggressor  bool
    IsControllerRole bool // ← DELETE THIS LINE
}
```

Delete the IsControllerRole field.

- [ ] **Step 3: Delete or update any remaining test references**

For each remaining test reference, either delete the test assertion (if it was specifically testing the bool) or update to use the new Control state model.

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./... -count=1 2>&1 | grep -E "^(FAIL|ok)" | tail -30`
Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore(position): chunk 4b-fixup-2 T16 — delete IsControllerRole field

The bool that started this chunk is gone. All readers have moved
to Character.IsController() / IsBeingControlled() (T7) or to
control.State directly (T14). GrappleData now contains only what
makes sense for the pair (Reason, Partner, IsAggressor).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 17: Update context.md files (control + position + hooks)

**Files:**
- Create: `internal/state/control/context.md`
- Modify: `internal/state/position/context.md`
- Modify: `internal/hooks/context.md`

- [ ] **Step 1: Create internal/state/control/context.md**

```markdown
# Package: internal/state/control

Per-character ControlLevel state machine for grapple dominance tracking.
Chunk 4b-fixup-2 (2026-05-18).

## States

- **Controlling** (stable): you have positional dominance.
- **LosingControl** (transient, auto-resolves same-tick): traversing
  Controlling↔Neutral boundary; fires gradient flavor messaging.
- **Neutral** (stable): neither side dominant; symmetric or in-flux.
- **BecomingControlled** (transient, auto-resolves same-tick): traversing
  Neutral↔Controlled boundary; fires gradient flavor messaging.
- **Controlled** (stable): you're dominated.

## Pattern

Mirrors `internal/state/awareness/` (the `Hidden → Revealing → Visible`
same-tick traversal). Transient states are real FSM states but
auto-resolve to the target stable state in the same call to
`TransitionTo*`. Boundary-cross callbacks fire during the brief in-state
moment.

## Triggers

- `TriggerGrappleEnter`: initial setup at pair transition.
- `TriggerDriftWin`: drift roll favored this side; shift toward Controlling.
- `TriggerDriftLoss`: drift roll opposed this side; shift toward Controlled.
- `TriggerGrappleExit`: grapple breaks; reset to Neutral.

## Boundary-cross callbacks

Registered via `RegisterBoundaryCrossCallback`. Receives
`(self, transient, from, to, reason)`. Fires once per boundary crossed
during a same-tick transition. Used by `internal/hooks/Position_GrappleTick.go`
to emit gradient messaging via `internal/grapplemessaging`.

## Initial states per position (set by `internal/state/position/pair.go`)

- Symmetric (Clinch, HalfGuard, Turtle): both sides Neutral.
- Asymmetric: controller arg → Controlling; controlled arg → Controlled.

## Validation

Per-pair invariant (in `internal/state/position/validation.go`):
- Not both at Controlling.
- Not both at Controlled.

## Not currently used for

- Position-change outcome gating. Position changes still come from the
  z-bucket outcome resolver (`internal/state/position/outcomes.go`,
  chunk 4b-fixup).
- Sub gate magnitude. Sub windows still gate on `|z| >= 1.5` (chunk 4d).
  ControlLevel state DOES gate sub eligibility per-position (must be
  Controlling for top subs; Controlled for bottom subs).
```

- [ ] **Step 2: Update internal/state/position/context.md**

Replace any chunk-4b-fixup references to `IsControllerRole bool` with the new model. Add a section on Character.Control machine; describe TransitionPair's ControlLevel initialization; document the new `IsAggressor` field's purpose.

- [ ] **Step 3: Update internal/hooks/context.md**

Update the Position_GrappleTick section:
- Now iterates pairs (not characters with bool filter).
- Calls `determineDriftAttacker` for attacker-arg.
- Calls `applyControlShift` after stamina cost.
- Boundary-cross callback registered at init for gradient messaging via grapplemessaging.

- [ ] **Step 4: Commit**

```bash
git add internal/state/control/context.md internal/state/position/context.md internal/hooks/context.md
git commit -m "docs: chunk 4b-fixup-2 T17 — context.md sweep

New context.md for internal/state/control/. position + hooks
context.md updated for ControlLevel FSM integration, pair
iteration, gradient messaging wiring.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 18: Update COMBAT_STATE_ROADMAP.md

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Add chunk 4b-fixup-2 row**

Find the chunk table. After the chunk 4b-fixup row, insert:

```markdown
| 4b-fixup-2 | Position — ControlLevel FSM | Done (2026-MM-DD) | Restores ControlLevel as a proper FSM in `internal/state/control/` (5 states: 3 stable + 2 transient mirroring Awareness Revealing) after chunk 4b-fixup's `IsControllerRole bool` collapsed Neutral to "both false" and broke per-round drift in symmetric Clinch grapples. `processGrappleTick` refactored to iterate pairs (deduped) instead of per-character with bool filter — fixes the iteration-layer bug independent of ControlLevel. Two parallel consumers of drift z: outcome resolver (chunk 4b-fixup, unchanged) for position changes, ControlLevel shift for state transitions + gradient messaging. ~36 new gradient templates across 4 boundary-direction keys. Sub eligibility tightens: top subs require Controlling state, bottom subs require Controlled. `IsAggressor` field on GrappleData as drift-roll tiebreaker for symmetric positions. |
```

(Replace `2026-MM-DD` with the actual ship date.)

- [ ] **Step 2: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "docs: chunk 4b-fixup-2 T18 — roadmap entry as Done

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 19: Boot smoke + re-run AI feature-tester smoke

**Files (no code change — verification only):**
- Smoke output

- [ ] **Step 1: Local boot smoke**

```bash
go build -o dogmud-test.exe . && ./dogmud-test.exe > test-server.log 2>&1 &
```

Wait until "Server Ready" appears in the log (~6 seconds). Look for:
- No panics, no nil-pointer crashes.
- `Config name=Balance.GlobalDamageMultiplier value=0.5` (from prior test config).
- grapple_outcomes.yaml loads cleanly (no `ValidateCompleteness` warnings).
- No "missing gradient key" warnings.

- [ ] **Step 2: Re-dispatch the AI feature-tester smoke**

Use the existing goal file from chunk 4b-fixup:

```
tools/testing/goals/chunk-4b-fixup-position-advancement-smoke.yaml
```

Dispatch via `/test-mud local feature-tester chunk-4b-fixup-position-advancement-smoke.yaml`.

Expected outcomes (this time):
- Goal 1 (engage humanoid mob, 5+ rounds): PASS
- Goal 2 (position advances within 10 rounds): PASS (Clinch → Mount within reasonable time)
- Goal 3 (Hold flavor fires sparsely): PASS
- Goal 4 (Mount strike apex flavor): PASS once Mount is reached
- Goal 5 (degradation / reversal / escape messaging): PASS for at least one outcome
- Goal 6 (no panics / no missing template debug strings): PASS

Plus new observable behaviors:
- Gradient messages fire when ControlLevel crosses boundaries (e.g., "your control slips" during Clinch).
- `combatstats position` should now show nonzero `Grapple Controller %` instead of 0.0%.

- [ ] **Step 3: Address any findings**

If the smoke surfaces new issues, address them. If the smoke passes, commit the goal-file-or-report-only changes (if any) and proceed.

- [ ] **Step 4: Cleanup**

```bash
taskkill //F //IM dogmud-test.exe
rm -f dogmud-test.exe test-server.log
```

- [ ] **Step 5: Commit (if any fixes were needed)**

If the smoke ran clean, no commit is strictly needed — but optionally:

```bash
git commit --allow-empty -m "smoke(chunk-4b-fixup-2): T19 boot + AI tester pass clean

Chunk 4b-fixup-2 ships. Clinch grapples now have per-round drift
firing, position advancement, gradient flavor on ControlLevel
boundary crossings, and all chunk-4b-fixup outcome messaging
intact.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage (cross-referencing spec sections to plan tasks):**

| Spec section | Covered by |
|---|---|
| §1 Problem Statement | (rationale only — no task needed) |
| §2 Design Goals | (rationale only) |
| §3 Concept Model | T1 (state enum), T4 (Character.Control field), T5 (IsAggressor) |
| §3.1 ControlLevel state semantics | T1 (enum + transitions table), T2 (transition methods) |
| §3.2 Role definitions | T5 (IsAggressor), T8 (determineDriftAttacker) |
| §4 Per-Round Resolution Flow | T8 (pair iteration), T9 (shift), T13 (gradient messaging fires) |
| §5 Drift z → ControlLevel Shift Mapping | T9 (applyControlShift) |
| §6 Per-Position Initial ControlLevel State | T6 (TransitionPair) |
| §7 Gradient Messaging Library Extension | T10 (GradientTriad + validator), T11 (templates), T12 (review), T13 (wiring) |
| §8 Sub Eligibility Refactor | T14 |
| §9 Pair Iteration in processGrappleTick | T8 (the bug fix) |
| §10 Validation Invariants | T15 |
| §11 What Survives Unchanged | (no task — verified by passing tests) |
| §12 What Gets Replaced | T6-T8 + T14 + T16 |
| §13 New Artifacts | T1-T5 + T10-T13 |
| §14 Migration Order | Plan task ordering matches |
| §15 Testing Strategy — unit | T1-T13 each include unit tests |
| §15 Testing Strategy — integration | T9 (z + shift integration), T13 (gradient + shift integration) |
| §15 Testing Strategy — smoke | T19 |
| §16 Out of Scope | (no tasks — confirmed) |
| §17 Risks | Addressed in task plan (T8 standalone-shippable, T12 realism review) |
| §18 Files Touched | All listed files have tasks |
| §19 Success Criteria | T19 verifies |

**Placeholder scan:** clean — every step has explicit code or commands. T15 has one approximation ("the signature of `ConsistencyCheck` may need adjustment") — implementer adjusts based on actual current signature. Acceptable.

**Type consistency:** `control.State`, `control.Machine`, `control.NewMachine()`, `control.TransitionToControlling/Neutral/Controlled`, `RegisterBoundaryCrossCallback`, `GradientTriad`, `Library.Gradients`, `RequiredGradientKeys`, `IsAggressor`, `determineDriftAttacker`, `applyControlShift`, `emitGradientMessage`, `gradientKeyForCrossing`, `PickGradientTemplate` — all used consistently across tasks.

Plan complete.

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-18-state-chunk-4b-fixup-2-controllevel-fsm.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
