# State Chunk 6 — Perception Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the Perception state machine (Sighted / Blinded) as a dormant primitive — the FSM transitions correctly via existing buff/condition lifecycle hooks, but no consumer reads the state yet. The future Messaging Framework chunk will consume it.

**Architecture:** New `internal/state/perception` package with a two-state union enum + `Machine` wrapper. `Character.Perception *perception.Machine` field initialized at construction. Transitions fire INLINE from `Character.AddBuff` / `Character.RemoveBuff` (for buff IDs 3 and 77) and `Character.AddCondition` / `Character.RemoveCondition` (for `ConditionBlinded`). No new buff-flag constants required — detect by buff ID. A `HasAnyBlindSource()` helper guards expire-paths so overlapping sources don't flicker.

**Tech Stack:** Go, `internal/state` framework, `internal/characters` + `internal/buffs` packages. NO YAML edits, NO broadcast cutover, NO smoke pass.

**Spec reference:** `docs/superpowers/specs/2026-05-19-state-chunk-6-perception-design.md`

---

## File Structure

### New files

| Path | Responsibility |
|---|---|
| `internal/state/perception/perception.go` | `State` enum, `String()`, `Machine` wrapper, `NewMachine()`, `TransitionTo()`, observer/veto wrappers (mirroring chunk-5 presence). |
| `internal/state/perception/transitions.go` | `transitions TransitionTable[State]` + trigger string constants + blind-source buff-ID constants (`BuffIdBlinded = 3`, `BuffIdFlashbangBlindness = 77`). |
| `internal/state/perception/perception_test.go` | Behavior Matrix unit tests PE-001 through PE-011. |
| `internal/state/perception/integration_test.go` | Real-Character integration tests for overlap + single-source paths. |
| `internal/state/perception/context.md` | Package documentation. |

### Modified files

| Path | Change |
|---|---|
| `internal/characters/character.go` | Add `Perception *perception.Machine` field; init in `New()`; reset in `ResetForMobInstance()`. |
| `internal/characters/validate.go` | Nil-guard installing default machine for YAML-loaded characters. |
| `internal/mobs/mobs.go` | Unconditional overwrite in `Validate()` after `Character.Validate()` (consistency with Presence pattern). |
| `internal/characters/sight.go` (new) | `HasAnyBlindSource()` helper method on Character. |
| `internal/characters/buffs.go` | Fire Perception transitions from `AddBuff` and `RemoveBuff` for buff IDs 3 + 77. |
| `internal/characters/conditions.go` | Fire Perception transitions from `AddCondition` and `RemoveCondition` for `ConditionBlinded`. |
| `internal/state/context.md` | Add Perception as the sixth consumer. |
| `internal/characters/context.md` | Note the new `Perception` field. |
| `COMBAT_STATE_ROADMAP.md` | Mark chunk 6 as Done (dormant). |
| `PATCH_NOTES.md` | Brief entry noting the FSM ships dormant. |

---

## Tasks

### Task 1: Perception package + Behavior Matrix tests

**Files:**
- Create: `internal/state/perception/perception.go`
- Create: `internal/state/perception/transitions.go`
- Create: `internal/state/perception/perception_test.go`

- [ ] **Step 1: Write `perception.go`**

```go
// Package perception defines the Perception state machine — the sixth
// consumer of internal/state, after combatphase, awareness, life,
// activity, position, and presence. Two-state FSM (Sighted / Blinded)
// gating broadcast visibility and active-inspection.
//
// SHIPS DORMANT in chunk 6. Transitions fire correctly via buff /
// condition observers, but no consumer reads the state yet. The future
// centralized messaging framework chunk will consume this primitive
// (broadcast cutover, infrared rendering, look-cmd gating, color
// coding, line wrapping). See
// docs/superpowers/specs/2026-05-19-state-chunk-6-perception-design.md
// for the full design and the messaging-framework memory for the
// successor work.
package perception

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Perception state enum.
type State int

const (
	Sighted State = iota // default — eyes work
	Blinded              // any active blind source
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Sighted:
		return "Sighted"
	case Blinded:
		return "Blinded"
	}
	return "Unknown"
}

// Machine wraps state.Machine[State] with Perception-specific API.
// Mirrors the chunk-5 presence.Machine wrapper.
type Machine struct {
	inner *state.Machine[State]
	self  state.ActorRef
}

// NewMachine returns a Machine in Sighted. Same constructor for both
// players and mobs — no per-actor polymorphism (unlike chunk 5
// Presence).
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Sighted, transitions),
	}
}

// State returns the current state. Safe from any goroutine; nil-safe
// (returns Sighted on a nil receiver).
func (m *Machine) State() State {
	if m == nil || m.inner == nil {
		return Sighted
	}
	return m.inner.State()
}

// SetSelf binds the machine to its owning ActorRef.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// TransitionTo moves the machine to the target state. Returns
// ErrInvalidTransition if the transition isn't allowed by the table
// (e.g., Blinded→Blinded). Errors are non-fatal and discardable at
// most call sites — re-applying a buff while already Blinded is a
// no-op, not an error.
func (m *Machine) TransitionTo(to State, r state.TransitionReason) error {
	if m == nil || m.inner == nil {
		return nil
	}
	return m.inner.TransitionTo(to, r)
}

// RegisterVeto installs a veto on transitions from `from` to `to`.
// Mirrors the chunk-5 presence.Machine wrapper. Unused in chunk 6
// (no vetoes registered); future messaging framework may use this.
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
// Mirrors the chunk-5 presence.Machine wrapper. Unused in chunk 6;
// future messaging framework may use this for broadcast routing.
func (m *Machine) RegisterObserver(name string, obs func(from, to State, r state.TransitionReason)) {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.Subscribe(name, obs)
}

// Inner returns the underlying state.Machine. Available for any
// consumer that needs direct access to the raw framework machine.
// Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// CancelScheduled cancels all pending scheduled transitions on this
// Perception machine. Called by Character.CancelAllScheduled on
// terminal-state entry (per chunk 5 T8 cascade pattern).
func (m *Machine) CancelScheduled() {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.CancelScheduled()
}
```

- [ ] **Step 2: Write `transitions.go`**

```go
package perception

import "github.com/GoMudEngine/GoMud/internal/state"

// transitions enforces the Perception invariant matrix. Two states,
// two edges. Re-entry (Sighted→Sighted or Blinded→Blinded) is not in
// the table — callers should check current state before firing the
// transition to avoid ErrInvalidTransition.
var transitions = state.TransitionTable[State]{
	Sighted: {Blinded},
	Blinded: {Sighted},
}

// Trigger reason constants. Used in state.TransitionReason.Trigger.
const (
	TriggerBuffApplied      = "buff_applied"
	TriggerBuffExpired      = "buff_expired"
	TriggerConditionAdded   = "condition_added"
	TriggerConditionRemoved = "condition_removed"
)

// Blind-source buff IDs. Detected by ID rather than by flag because
// the existing buff YAMLs don't carry a "blinded" flag — they only
// have stat mods. Adding flags to the YAML would touch data; detecting
// by ID keeps chunk 6 dormant on the data side.
const (
	BuffIdBlinded            = 3  // _datafiles/world/dogmud/buffs/3-blinded.yaml
	BuffIdFlashbangBlindness = 77 // _datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml
)
```

- [ ] **Step 3: Write the failing Behavior Matrix tests**

Create `internal/state/perception/perception_test.go`:

```go
package perception

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestStateString(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{Sighted, "Sighted"},
		{Blinded, "Blinded"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("State(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}

// PE-001: NewMachine starts in Sighted.
func TestNewMachine_InitialSighted(t *testing.T) {
	m := NewMachine()
	if m.State() != Sighted {
		t.Errorf("NewMachine() state = %v, want Sighted", m.State())
	}
}

// PE-002: Sighted → Blinded via buff applied.
func TestSightedToBlindedOnBuffApplied(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied}); err != nil {
		t.Fatalf("TransitionTo(Blinded): %v", err)
	}
	if m.State() != Blinded {
		t.Errorf("state = %v, want Blinded", m.State())
	}
}

// PE-003: Same path with flashbang trigger (no semantic difference in the FSM —
// the trigger string distinguishes the source for logs).
func TestSightedToBlindedOnFlashbang(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied, Metadata: map[string]any{"buffId": BuffIdFlashbangBlindness}}); err != nil {
		t.Fatalf("TransitionTo(Blinded): %v", err)
	}
	if m.State() != Blinded {
		t.Errorf("state = %v, want Blinded", m.State())
	}
}

// PE-004: Sighted → Blinded via condition added.
func TestSightedToBlindedOnConditionAdded(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerConditionAdded}); err != nil {
		t.Fatalf("TransitionTo(Blinded): %v", err)
	}
	if m.State() != Blinded {
		t.Errorf("state = %v, want Blinded", m.State())
	}
}

// PE-005: Blinded → Sighted via buff expired.
func TestBlindedToSightedOnBuffExpired(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied})
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerBuffExpired}); err != nil {
		t.Fatalf("TransitionTo(Sighted): %v", err)
	}
	if m.State() != Sighted {
		t.Errorf("state = %v, want Sighted", m.State())
	}
}

// PE-006: Blinded → Sighted via flashbang expired.
func TestBlindedToSightedOnFlashbangExpired(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied})
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerBuffExpired, Metadata: map[string]any{"buffId": BuffIdFlashbangBlindness}}); err != nil {
		t.Fatalf("TransitionTo(Sighted): %v", err)
	}
	if m.State() != Sighted {
		t.Errorf("state = %v, want Sighted", m.State())
	}
}

// PE-007: Blinded → Sighted via condition removed.
func TestBlindedToSightedOnConditionRemoved(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerConditionAdded})
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerConditionRemoved}); err != nil {
		t.Fatalf("TransitionTo(Sighted): %v", err)
	}
	if m.State() != Sighted {
		t.Errorf("state = %v, want Sighted", m.State())
	}
}

// PE-008: Pure-FSM check that Blinded→Blinded is not in the table.
// (Caller guards against this via HasAnyBlindSource; this test just
// confirms the table rejects re-entry.)
func TestBlindedSelfTransitionRejected(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied})
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied}); err == nil {
		t.Errorf("TransitionTo(Blinded) from Blinded: got nil err; want ErrInvalidTransition")
	}
	// State unchanged.
	if m.State() != Blinded {
		t.Errorf("state after rejected self-transition = %v, want Blinded", m.State())
	}
}

// PE-009: Sighted→Sighted is also not in the table (symmetric check).
func TestSightedSelfTransitionRejected(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerBuffExpired}); err == nil {
		t.Errorf("TransitionTo(Sighted) from Sighted: got nil err; want ErrInvalidTransition")
	}
	if m.State() != Sighted {
		t.Errorf("state after rejected self-transition = %v, want Sighted", m.State())
	}
}
```

Note: PE-008/PE-009 cover the table-rejects-re-entry cases at the pure-FSM level. PE-010/PE-011 from the spec's Behavior Matrix are about the CALLER's behavior (only call TransitionTo when state would actually change) — those land in `integration_test.go` in Task 6, not here, because they exercise the inline guard pattern in `AddBuff`.

- [ ] **Step 4: Run the tests and verify they pass**

Run: `go test ./internal/state/perception/... -v`
Expected: PASS — 10 unit tests (1 String test + 9 PE-NNN tests).

- [ ] **Step 5: Run the full state-machine test suite**

Run: `go test ./internal/state/...`
Expected: PASS — no regressions in other state packages.

- [ ] **Step 6: Commit**

```bash
git add internal/state/perception/
git commit -m "$(cat <<'EOF'
feat(state): T1 — Perception package with Behavior Matrix tests

New internal/state/perception package: two-state union enum (Sighted /
Blinded) with one transition table. NewMachine constructor (same for
players and mobs), Machine wrapper mirroring chunk-5 presence shape
(State, TransitionTo, RegisterVeto, RegisterObserver, Inner,
CancelScheduled).

Behavior Matrix unit tests PE-001 through PE-009: initial state,
transitions in both directions, re-entry rejection by the table.
PE-010/PE-011 (caller-guard semantics) land in T6's integration tests.

Detects blind sources by buff ID (3, 77) rather than by flag — buff
YAMLs don't currently carry a blindness flag and adding one would
require data file edits. ID-based detection keeps chunk 6 dormant on
the data side.

Ships DORMANT per the chunk 4a precedent. No consumer reads the state
yet; the future messaging framework chunk will wire it into broadcast
gating, infrared rendering, look-cmd blocking.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Wire `Character.Perception` field

**Files:**
- Modify: `internal/characters/character.go` (struct field + `New()` init + `ResetForMobInstance` clear)
- Modify: `internal/characters/validate.go` (nil-guard installing default machine)
- Modify: `internal/mobs/mobs.go` (unconditional overwrite in `Validate()`)

- [ ] **Step 1: Add `Perception` field to Character struct**

Open `internal/characters/character.go`. Find the state-machine field group (`CombatPhase`, `Life`, `Position`, `Awareness`, `Activity`, `Control`, `Presence`). Add right after `Presence`:

```go
	// Perception is the canonical state machine for "do this character's
	// eyes work?" — Sighted / Blinded. Ships DORMANT in chunk 6: the
	// machine transitions correctly via buff/condition observers but no
	// consumer reads the state yet. The future centralized messaging
	// framework chunk will wire it into broadcast gating, infrared
	// rendering, look-command blocking. See
	// internal/state/perception/context.md.
	Perception               *perception.Machine            `yaml:"-"`
```

Add the import at the top of the file:

```go
	"github.com/GoMudEngine/GoMud/internal/state/perception"
```

- [ ] **Step 2: Initialize Perception in `characters.New()`**

Find the `New()` function. After the other state machines are initialized (look for `Presence:` in the struct literal or `c.Presence = ...`), add:

```go
		Perception: perception.NewMachine(),
```

inside the struct literal, OR add `c.Perception = perception.NewMachine()` right after the existing `c.Presence = presence.NewPlayerPresence()` line — whichever matches the surrounding pattern.

- [ ] **Step 3: Reset Perception in `ResetForMobInstance`**

Find `ResetForMobInstance()` (around line 73). It clears `Life`, `CombatPhase`, `Position`, `Awareness`, `Activity`, `Control`, `Presence`. Add:

```go
	c.Perception = nil
```

- [ ] **Step 4: Add nil-guard in `Character.Validate()`**

Open `internal/characters/validate.go`. Find the existing nil-guard block where `Presence` is installed (around line 545 from the chunk-5 work — search `c.Presence == nil`). Add an analogous block immediately after:

```go
	if c.Perception == nil {
		// Player default — mob.Validate() overwrites unconditionally
		// after this runs. Same constructor for both actor types but
		// the unconditional overwrite matches the Presence pattern.
		c.Perception = perception.NewMachine()
	}
```

Add the import to `validate.go` if not already present:

```go
	"github.com/GoMudEngine/GoMud/internal/state/perception"
```

- [ ] **Step 5: Unconditional overwrite in mob `Validate()`**

Open `internal/mobs/mobs.go`. Find the spot in `Validate()` where `Presence` is set unconditionally (per chunk 5 T2 — search `r.Character.Presence = presence.NewMobPresence()`). Add right after the Presence assignment + observer block:

```go
	r.Character.Perception = perception.NewMachine()
```

The unconditional overwrite is defensive — it ensures the mob path always installs a fresh machine even though the constructor is the same as the player path.

Add the import:

```go
	"github.com/GoMudEngine/GoMud/internal/state/perception"
```

- [ ] **Step 6: Build and verify compilation**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7: Run the package tests**

Run: `go test ./internal/characters/... ./internal/mobs/...`
Expected: PASS — existing tests not affected.

- [ ] **Step 8: Commit**

```bash
git add internal/characters/character.go internal/characters/validate.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(characters): T2 — wire Character.Perception field

Add Perception *perception.Machine field to Character (yaml:"-"). Init
in characters.New(); reset in ResetForMobInstance(); nil-guarded in
Character.Validate() (player default); unconditional overwrite in
mob.Validate() after Character.Validate() runs (consistency with
chunk-5 Presence pattern, even though both actor types use the same
constructor).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: `HasAnyBlindSource()` helper

**Files:**
- Create: `internal/characters/sight.go`

- [ ] **Step 1: Create `sight.go`**

```go
// Vision-related helpers on Character. Currently houses HasAnyBlindSource
// for chunk 6's Perception machine. Future messaging framework will add
// CanSee / CanSeeClearly / CanSeeShapes here.
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// HasAnyBlindSource returns true if any active blind source is currently
// affecting this character. Used by Perception expire-paths in AddBuff,
// RemoveBuff, AddCondition, RemoveCondition to decide whether to fire
// the Blinded→Sighted transition when one of multiple overlapping
// sources clears.
//
// Sources checked:
//   - Buff 3 (Blinded) — _datafiles/world/dogmud/buffs/3-blinded.yaml
//   - Buff 77 (Flashbang Blindness) — _datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml
//   - ConditionBlinded — currently applied by blinding-flash and
//     blinding-spit mutations (see usercommands/mutation_blinding_*.go).
func (c *Character) HasAnyBlindSource() bool {
	if c == nil {
		return false
	}
	if c.Buffs.HasBuff(perception.BuffIdBlinded) {
		return true
	}
	if c.Buffs.HasBuff(perception.BuffIdFlashbangBlindness) {
		return true
	}
	if c.HasCondition(ConditionBlinded) {
		return true
	}
	return false
}
```

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: clean build. No import cycle (characters → state/perception is one-way).

- [ ] **Step 3: Commit**

```bash
git add internal/characters/sight.go
git commit -m "$(cat <<'EOF'
feat(characters): T3 — HasAnyBlindSource() helper for Perception

New sight.go houses vision-related Character helpers. First helper:
HasAnyBlindSource returns true if Buff 3, Buff 77, or ConditionBlinded
is active. Used by Perception expire-paths in T4+T5 to avoid flicker
when one of multiple overlapping blind sources clears.

Future messaging framework will add CanSee / CanSeeClearly /
CanSeeShapes predicates here.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: `ConditionBlinded` add/remove fires transitions

**Files:**
- Modify: `internal/characters/conditions.go` (extend `AddCondition` and `RemoveCondition`)

- [ ] **Step 1: Extend `AddCondition` to fire Perception transition**

Open `internal/characters/conditions.go`. Find `AddCondition()` (around line 84). Right before the closing brace of the function (after the new-or-overwrite logic), add:

```go
	// Chunk 6 (Perception): ConditionBlinded triggers Sighted → Blinded.
	// Guard against re-entry: only fire if state is currently Sighted.
	if typ == ConditionBlinded && c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerConditionAdded})
	}
```

Add the imports at the top of the file:

```go
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
```

- [ ] **Step 2: Extend `RemoveCondition` to fire Perception transition**

Find `RemoveCondition()` (around line 115). After the existing remove logic (right before the closing brace), add:

```go
	// Chunk 6 (Perception): ConditionBlinded clear may flip Blinded →
	// Sighted, but only if no other blind source is still active.
	if typ == ConditionBlinded && c.Perception != nil && c.Perception.State() == perception.Blinded && !c.HasAnyBlindSource() {
		_ = c.Perception.TransitionTo(perception.Sighted,
			state.TransitionReason{Trigger: perception.TriggerConditionRemoved})
	}
```

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Run the characters tests**

Run: `go test ./internal/characters/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/conditions.go
git commit -m "$(cat <<'EOF'
feat(characters): T4 — ConditionBlinded fires Perception transitions

AddCondition(ConditionBlinded, ...) fires Sighted→Blinded if current
state is Sighted (re-entry guard). RemoveCondition(ConditionBlinded)
fires Blinded→Sighted if no other blind source remains active
(HasAnyBlindSource guard).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: `AddBuff` / `RemoveBuff` fire transitions for buffs 3 + 77

**Files:**
- Modify: `internal/characters/buffs.go` (extend `AddBuff` and `RemoveBuff`)

- [ ] **Step 1: Extend `AddBuff` to fire Perception transition**

Open `internal/characters/buffs.go`. Find `AddBuff()` (around line 77). The function currently:

```go
func (c *Character) AddBuff(buffId int, isPermanent bool) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuff(buffId, isPermanent) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	c.Validate()
	return nil
}
```

Insert the Perception transition between `c.Buffs.AddBuff(...)` success and `c.Validate()`:

```go
func (c *Character) AddBuff(buffId int, isPermanent bool) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuff(buffId, isPermanent) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	// Chunk 6 (Perception): blind-source buffs trigger Sighted → Blinded.
	// Guard against re-entry: only fire if state is currently Sighted.
	if (buffId == perception.BuffIdBlinded || buffId == perception.BuffIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerBuffApplied, Metadata: map[string]any{"buffId": buffId}})
	}
	c.Validate()
	return nil
}
```

Add the imports:

```go
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
```

- [ ] **Step 2: Apply the same to `AddBuffScaled`**

Find `AddBuffScaled()` (around line 87). Same insertion pattern:

```go
func (c *Character) AddBuffScaled(buffId int, durationMult float64) error {
	buffId = int(math.Abs(float64(buffId)))
	if !c.Buffs.AddBuffScaled(buffId, durationMult) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, buffId)
	}
	// Chunk 6 (Perception): see AddBuff above.
	if (buffId == perception.BuffIdBlinded || buffId == perception.BuffIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerBuffApplied, Metadata: map[string]any{"buffId": buffId}})
	}
	c.Validate()
	return nil
}
```

- [ ] **Step 3: Extend `RemoveBuff` to fire Perception transition**

Find `RemoveBuff()` (around line 104):

```go
func (c *Character) RemoveBuff(buffId int) {
	buffId = int(math.Abs(float64(buffId)))
	c.Buffs.RemoveBuff(buffId)
	c.Validate()
}
```

Insert the transition between `RemoveBuff` and `Validate`:

```go
func (c *Character) RemoveBuff(buffId int) {
	buffId = int(math.Abs(float64(buffId)))
	c.Buffs.RemoveBuff(buffId)
	// Chunk 6 (Perception): clearing a blind-source buff may flip
	// Blinded → Sighted, but only if no other blind source remains.
	if (buffId == perception.BuffIdBlinded || buffId == perception.BuffIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Blinded && !c.HasAnyBlindSource() {
		_ = c.Perception.TransitionTo(perception.Sighted,
			state.TransitionReason{Trigger: perception.TriggerBuffExpired, Metadata: map[string]any{"buffId": buffId}})
	}
	c.Validate()
}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Run the characters tests**

Run: `go test ./internal/characters/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/characters/buffs.go
git commit -m "$(cat <<'EOF'
feat(characters): T5 — buff 3 / buff 77 fire Perception transitions

AddBuff and AddBuffScaled fire Sighted→Blinded for buff IDs 3 (Blinded)
and 77 (Flashbang Blindness), guarded by current-state check to avoid
re-entry. RemoveBuff fires Blinded→Sighted for the same IDs, guarded
by HasAnyBlindSource() so overlapping sources don't flicker.

Detected by buff ID rather than flag because the existing buff YAMLs
don't carry a blindness-specific flag. ID-based detection keeps chunk 6
data-file-free.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Integration tests for overlap + single-source paths

**Files:**
- Create: `internal/state/perception/integration_test.go`

- [ ] **Step 1: Write the integration tests**

Create `internal/state/perception/integration_test.go`:

```go
package perception_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// PE-INT-001: AddBuff(3) → Blinded.
func TestIntegration_BuffBlindedAppliesBlinded(t *testing.T) {
	c := characters.New()
	if c.Perception.State() != perception.Sighted {
		t.Fatalf("initial state = %v, want Sighted", c.Perception.State())
	}
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(3), state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-002: AddBuff(3) + AddCondition(Blinded) + RemoveBuff(3) → still Blinded.
func TestIntegration_OverlapKeepsBlinded(t *testing.T) {
	c := characters.New()
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	c.AddCondition(characters.ConditionBlinded, 5, 0.7, "test-overlap")
	c.RemoveBuff(perception.BuffIdBlinded)
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after removing buff but condition still active, state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-003: Add both, remove both → Sighted.
func TestIntegration_AllSourcesClearedReturnsSighted(t *testing.T) {
	c := characters.New()
	_ = c.AddBuff(perception.BuffIdBlinded, false)
	c.AddCondition(characters.ConditionBlinded, 5, 0.7, "test-clear")
	c.RemoveBuff(perception.BuffIdBlinded)
	c.RemoveCondition(characters.ConditionBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after clearing all sources, state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-004 (PE-010 from spec matrix): re-applying buff while already
// Blinded is a no-op (no ErrInvalidTransition propagated, no log spam).
func TestIntegration_ReapplyBuffNoOp(t *testing.T) {
	c := characters.New()
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("first AddBuff: %v", err)
	}
	// Re-add the same buff (the buff system stacks duration, but the
	// blind-source state is the same). The current-state guard in
	// AddBuff prevents the transition from firing twice.
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("second AddBuff: %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after duplicate AddBuff, state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-005: Flashbang (buff 77) drives the same Perception transitions.
func TestIntegration_FlashbangBlindness(t *testing.T) {
	c := characters.New()
	if err := c.AddBuff(perception.BuffIdFlashbangBlindness, false); err != nil {
		t.Fatalf("AddBuff(77): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(77), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveBuff(perception.BuffIdFlashbangBlindness)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveBuff(77), state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-006: ConditionBlinded alone drives the transitions.
func TestIntegration_ConditionBlindedSolo(t *testing.T) {
	c := characters.New()
	c.AddCondition(characters.ConditionBlinded, 3, 0.5, "test-solo-cond")
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddCondition(Blinded), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(characters.ConditionBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveCondition(Blinded), state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-007: Mixed source order — condition added first, then buff,
// then condition removed → still Blinded (buff still active).
func TestIntegration_MixedSourceOrder(t *testing.T) {
	c := characters.New()
	c.AddCondition(characters.ConditionBlinded, 5, 0.7, "test-order")
	if c.Perception.State() != perception.Blinded {
		t.Fatalf("after AddCondition, state = %v, want Blinded", c.Perception.State())
	}
	if err := c.AddBuff(perception.BuffIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	// Re-adding-while-already-Blinded path; state must remain Blinded.
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff while condition active, state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(characters.ConditionBlinded)
	// Buff still active → still Blinded.
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after RemoveCondition (buff still active), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveBuff(perception.BuffIdBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveBuff (no sources left), state = %v, want Sighted", c.Perception.State())
	}
}
```

- [ ] **Step 2: Run the integration tests**

Run: `go test ./internal/state/perception/... -v -run TestIntegration`
Expected: PASS — all 7 integration tests green.

- [ ] **Step 3: Run the full state-machine test suite**

Run: `go test ./internal/state/... ./internal/characters/... ./internal/mobs/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/state/perception/integration_test.go
git commit -m "$(cat <<'EOF'
test(perception): T6 — integration tests for overlap + single-source paths

Seven integration tests using real Character + buff + condition state:
- PE-INT-001: AddBuff(3) → Blinded
- PE-INT-002: Overlap keeps Blinded when one source clears
- PE-INT-003: All sources cleared → Sighted
- PE-INT-004: Re-applying buff while Blinded is a no-op
- PE-INT-005: Flashbang (buff 77) drives same transitions
- PE-INT-006: ConditionBlinded alone drives transitions
- PE-INT-007: Mixed source order doesn't break the state machine

Covers the spec's PE-010 + PE-011 (caller-guard semantics) which sit
at the integration layer rather than the pure-FSM layer.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Context.md sweep + roadmap + patch notes close-out

**Files:**
- Create: `internal/state/perception/context.md`
- Modify: `internal/state/context.md` (cross-link as sixth consumer)
- Modify: `internal/characters/context.md` (note the Perception field)
- Modify: `COMBAT_STATE_ROADMAP.md` (mark chunk 6 as Done, dormant)
- Modify: `PATCH_NOTES.md` (brief entry)

- [ ] **Step 1: Write `internal/state/perception/context.md`**

Follow the chunk-5 presence/context.md style. Cover:

```markdown
# perception — Package Documentation

## Overview

The `internal/state/perception` package is the sixth and final consumer
of the `internal/state` framework — completing the combat-state-machines
arc (chunks 0-6). It defines a two-state FSM (`Sighted | Blinded`) that
gates "does this character's eyes work?" semantics.

**Status (shipped DORMANT, 2026-05-19):**

The machine transitions correctly via existing buff/condition lifecycle
hooks (Buff 3 Blinded, Buff 77 Flashbang Blindness, ConditionBlinded).
NO CONSUMER reads the state yet. The future centralized messaging
framework chunk wires this primitive into broadcast gating, infrared
"red shapes" rendering, look-command blocking, color coding by event
category, and centralized line wrapping. See the
`[[messaging-framework-chunk]]` memory for the full scope of that
successor work.

The dormant ship follows the chunk-4a precedent (Position FSM shipped
DORMANT before chunk 4b wired writers + readers).

---

## State enum

| State | Semantics |
|---|---|
| Sighted | Default — eyes work |
| Blinded | Any active blind source (Buff 3, Buff 77, or ConditionBlinded) |

Two states. No transient states. No state-data structs.

---

## Transition table

```
Sighted → {Blinded}
Blinded → {Sighted}
```

Re-entry (Sighted→Sighted, Blinded→Blinded) is NOT in the table.
Callers must check current state before firing transitions — the
inline guards in `Character.AddBuff` / `AddCondition` / etc. handle
this.

---

## Trigger sources

| Source | File | Hook |
|---|---|---|
| Buff 3 (Blinded) | `_datafiles/world/dogmud/buffs/3-blinded.yaml` | `Character.AddBuff` / `RemoveBuff` |
| Buff 77 (Flashbang Blindness) | `_datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml` | `Character.AddBuff` / `RemoveBuff` |
| ConditionBlinded | `internal/characters/conditions.go` | `Character.AddCondition` / `RemoveCondition` |

Detection is by buff ID (not by flag) because the existing buff YAMLs
don't carry a blindness-specific flag — adding one would require data
file edits. Buff IDs `BuffIdBlinded = 3` and `BuffIdFlashbangBlindness
= 77` are constants in `transitions.go`.

---

## Helper: `HasAnyBlindSource`

`Character.HasAnyBlindSource()` (in `internal/characters/sight.go`)
returns true if any of the three sources is currently active. Used
by expire-paths to decide whether to fire Blinded→Sighted when one
of multiple overlapping sources clears.

---

## Construction

`NewMachine()` returns a Machine in Sighted state. Same constructor
for both player and mob — no per-actor polymorphism (unlike chunk 5
Presence).

`Character.Perception` field initialized at four sites:

1. `characters.New()` — player default.
2. `Character.Validate()` — nil-guard for YAML-loaded characters.
3. `mobs.Mob.Validate()` — unconditional overwrite.
4. `Character.ResetForMobInstance()` — reset to nil so fresh mob
   instances get their own machine.

---

## Integration points (current and future)

| When | Where |
|---|---|
| **Now (dormant)** | Transitions fire correctly; no consumer reads state. |
| **Future (messaging framework chunk)** | `room.SendTextVisual` consults `Perception.State()`; `look` command blocked when Blinded; infrared anonymizer transforms visual broadcasts. See `[[messaging-framework-chunk]]` memory. |

---

## Testing

- `perception_test.go` — Behavior Matrix unit tests (PE-001 through PE-009): pure-FSM coverage.
- `integration_test.go` — real-Character integration (PE-INT-001 through PE-INT-007): overlap + single-source paths via `AddBuff` / `AddCondition` / etc.

No smoke pass — chunk 6 ships dormant; no player-visible behavior to verify in-game.
```

- [ ] **Step 2: Update `internal/state/context.md`**

Open it. Find the introductory paragraph that lists the six consumers. Edit to confirm Perception is the sixth and that the arc is complete:

Look for text like "currently only `internal/state/combatphase`; future chunks 1-5 add..." and update it to reflect that all six consumers are now present.

- [ ] **Step 3: Update `internal/characters/context.md`**

Find the state-machine field listing for Character (it already includes Presence per chunk 5 T13). Add Perception with a 2-3 sentence description, noting it ships dormant in chunk 6 and the future messaging framework chunk wires it up.

- [ ] **Step 4: Update `COMBAT_STATE_ROADMAP.md`**

Find the chunk 6 row:

```markdown
| 6 | Perception | Not started | Sighted / Blinded. Observer-side dual to Awareness — centralizes room-broadcast visibility gating. Added 2026-05-13 after recurring blind/dark-room broadcast bugs (latest: companion-name leak through blindness). No hard dependency on chunks 3-5; could ship earlier if pain escalates. |
```

Replace with:

```markdown
| 6 | Perception | Done DORMANT (2026-05-19) | Two-state FSM (Sighted / Blinded) shipped DORMANT per the chunk-4a precedent. Transitions fire correctly via existing buff/condition lifecycle hooks (Buff 3 Blinded, Buff 77 Flashbang Blindness, ConditionBlinded — detected by buff ID, no YAML changes needed). HasAnyBlindSource() helper guards expire-paths against flicker when overlapping sources clear. Behavior Matrix unit tests (PE-001 through PE-009) + integration tests (PE-INT-001 through PE-INT-007) exercise overlap, mixed-order, and re-entry-no-op semantics. NO CONSUMER reads the state yet — the future centralized messaging framework chunk (captured as the `messaging-framework-chunk` project memory) wires this primitive into broadcast gating, infrared anonymized rendering, look-command blocking, color coding by event category, line wrapping, and the headline companion-name-leak bug fix. Original chunk-6 scope was found too narrow during brainstorm — the broader messaging problem deserves its own chunk. **Combat-state-machines arc complete** (chunks 0-6 all shipped). Aliveness substrate work can resume. |
```

- [ ] **Step 5: Update `PATCH_NOTES.md`**

Open it. Find the top entry (chunk 5 from earlier today). Add a new section ABOVE it:

```markdown
## 2026-05-19 — Chunk 6: Perception state machine (dormant)

**Internal-only change.** The engine now tracks every character's
visual state — Sighted or Blinded — as a proper state machine. Today
no gameplay surface uses this; existing dark-room and blindness
handling behaves exactly as before. The primitive is in place for a
future broader messaging-framework upgrade that ties together
color-coded combat text, infrared "red shapes" rendering, line
wrapping, and the long-standing bug where messages can leak through
blindness or darkness with character names visible.

The combat-state-machines arc that began with chunk 0 (2026-05-13)
is now complete — six FSMs (Combat Phase, Awareness, Life, Activity,
Position, Presence, Perception) all shipped. Mob aliveness substrate
work can resume.

## 2026-05-19 — Chunk 5: Presence state machine
```

- [ ] **Step 6: Final build + test**

```bash
go build ./...
go test ./...
```

Expected: PASS on both.

- [ ] **Step 7: Final boot check**

Per CLAUDE.md Pre-Push SOP:

```bash
go build -o dogmud-c6-final.exe .
./dogmud-c6-final.exe > /tmp/c6-final-boot.log 2>&1 &
sleep 10
grep -E "Server Ready|panic|LoadDataFiles.*loadedCount" /tmp/c6-final-boot.log
taskkill //F //IM dogmud-c6-final.exe
rm dogmud-c6-final.exe /tmp/c6-final-boot.log
```

Expected: "Server Ready" present, all `LoadDataFiles` lines show normal counts, no panics.

- [ ] **Step 8: Commit**

```bash
git add internal/state/perception/context.md internal/state/context.md \
        internal/characters/context.md COMBAT_STATE_ROADMAP.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(roadmap): T7 — chunk 6 done (dormant), combat-state-machines arc complete

New internal/state/perception/context.md package doc. Cross-link from
internal/state/context.md (sixth consumer) and internal/characters/context.md
(Perception field). COMBAT_STATE_ROADMAP chunk 6 row marked Done DORMANT
with deliverables summary and the link to messaging-framework-chunk
memory for the successor work. Dated PATCH_NOTES entry noting the FSM
ships internal-only with no player-visible change.

The combat-state-machines arc that started 2026-05-13 (chunk 0) is now
complete — six FSMs across the engine. Mob aliveness substrate work can
resume.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Verification Checklist

Before declaring chunk 6 complete:

- [ ] `go build ./...` clean.
- [ ] `go test ./...` clean.
- [ ] `internal/state/perception/perception.go` exists with the two-state enum + Machine wrapper.
- [ ] `internal/state/perception/transitions.go` has the table + trigger constants + buff-ID constants.
- [ ] `internal/state/perception/perception_test.go` covers PE-001 through PE-009.
- [ ] `internal/state/perception/integration_test.go` covers PE-INT-001 through PE-INT-007.
- [ ] `Character.Perception` field initialized for both player (constructor + Validate) and mob (Validate) paths.
- [ ] `Character.HasAnyBlindSource()` helper present in `internal/characters/sight.go`.
- [ ] `Character.AddBuff` / `AddBuffScaled` / `RemoveBuff` fire Perception transitions for buff IDs 3 + 77 with current-state guards.
- [ ] `Character.AddCondition` / `RemoveCondition` fire Perception transitions for `ConditionBlinded` with current-state guards.
- [ ] No buff YAML changes (no flag additions, no field edits).
- [ ] No broadcast call-site changes (`SendTextVisual` and friends untouched).
- [ ] No `canSeeInRoom` or `sendRoomTextDarknessAware` changes.
- [ ] Context.md updates land (perception package + state + characters + roadmap + patch notes).
- [ ] Server boots cleanly past data-file loading.

---

## Estimated effort

- T1 (perception package + tests): 45-60 min
- T2 (Character.Perception wiring): 20-30 min
- T3 (HasAnyBlindSource helper): 15 min
- T4 (Condition transitions): 15-20 min
- T5 (Buff transitions): 20-25 min
- T6 (integration tests): 30-45 min
- T7 (context.md + roadmap + patch notes): 30 min

Total: ~3-4 hours. Smallest chunk since 4a.

---

## Out of Scope (Reminders)

From spec §11 — all of the following moves to the future messaging-framework chunk:

- Broadcast cutover (228 sites; visual/audio classification).
- Composite predicates (`CanSee`, `CanSeeClearly`, `CanSeeShapes`).
- InfraredVision flag + anonymized "red shapes" rendering.
- Magical darkness as a room-level effect or observer debuff.
- `SendTextVisual` extension to consult Perception.
- `look` / `look <target>` blindness gating.
- Color coding by event category.
- Centralized line wrapping.
- Companion-name-leak bug fix.

If a smoke or implementation finding falls into one of the above, it becomes a memory addition to `project_messaging_framework_chunk.md`, NOT a chunk 6 task.
