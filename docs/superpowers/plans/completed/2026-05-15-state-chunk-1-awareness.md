# Combat State — Chunk 1: Awareness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Awareness state machine (`Visible / Concealing / Hidden / Revealing`) on the chunk-0 framework. Bundle the Hidden mechanic refresh: no duration, stamina cost for movement, light-conditional sneak score, logout safety valve. Migrate ~30 reader callsites + writer sites. Subscribe to Combat Phase's `OnEndOfRoundIfSurprise` callback to close the chunk-0 surprise handshake. Sunset dead `very_hidden` buff. Replace buff #9-as-state-of-truth with Awareness state; keep buff #9 as the effect carrier.

**Architecture:** Same generics-based pattern as Combat Phase. State machine carries per-state data (Concealing for sneak roll, Revealing for cascade context). Buff #9 stays as side-effect-only (no duration); Awareness transitions cascade-add and cascade-remove it. Hard cutover within chunk — no compat wrappers (the surface is small enough).

**Tech Stack:** Go 1.21+ with generics, existing `internal/state/` framework, existing `internal/state/combatphase/` machine (subscribed to).

**Spec:** `docs/superpowers/specs/2026-05-15-state-chunk-1-awareness-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/awareness/awareness.go` | NEW | State enum, data types, Machine wrapper, accessors |
| `internal/state/awareness/transitions.go` | NEW | Valid-transition table, trigger constants |
| `internal/state/awareness/rules.go` | NEW | Transition method implementations, registry, internal helpers |
| `internal/state/awareness/awareness_test.go` | NEW | Behavior Matrix tests (AW-001 through AW-033) |
| `internal/state/awareness/context.md` | NEW | Package documentation |
| `internal/characters/character.go` | MODIFY | Add `Awareness *awareness.Machine` field; `IsHidden()` predicate |
| `internal/characters/validate.go` | MODIFY | Nil-guard init of `Awareness` for YAML-loaded characters |
| `internal/hooks/Awareness_Vetoes.go` | NEW | Activity pre-wire veto (Casting/Crafting blocks Concealing) |
| `internal/hooks/Awareness_Cascades.go` | NEW | Combat Phase subscriptions; buff #9 add/remove on Awareness transitions |
| `internal/hooks/Awareness_LightChange.go` | NEW | Light-state-change event handler; re-roll fires |
| `internal/hooks/Logout_AwarenessCleanup.go` | NEW | Logout safety valve |
| `internal/actions/skill_helpers.go` | MODIFY | `CalcSneakScore(char, effectiveLit bool)` + `CalcSneakScoreVsObserver(...)` helper |
| `internal/actions/sneak.go` | MODIFY | Use `Awareness.TransitionToConcealing` instead of direct `AddBuff(9, ...)` |
| `internal/actions/steal.go` | MODIFY | Replace `CancelBuffsWithFlag(buffs.Hidden)` with `Awareness.TransitionToRevealing` |
| `internal/actions/plant.go` | MODIFY | Same |
| `internal/actions/remove_equip.go` | MODIFY | Same |
| `internal/actions/shadow.go` | MODIFY | `IsHidden()` check via Awareness |
| `internal/actions/say.go` | MODIFY | Hidden check via Awareness; noisy-action reveal trigger (AW-032) |
| `internal/usercommands/skill.skullduggery.sneak.go` | MODIFY | Reader migration; transition via Awareness |
| `internal/usercommands/go.go` | MODIFY | Stamina cost multiplier; light-change detection wiring |
| `internal/usercommands/shout.go` | MODIFY | Noisy-action reveal (AW-032) |
| `internal/usercommands/whisper.go` | MODIFY | Room-broadcast variant reveals; targeted whisper stays quiet |
| `internal/mobcommands/sneak.go` | MODIFY | Reader migration |
| ~30 reader sites | MODIFY | `HasBuffFlag(buffs.Hidden)` → `IsHidden()` |
| `internal/actions/combat_rally.go` | MODIFY | Noisy-action reveal (AW-033) |
| `internal/actions/combat_warcry.go` | MODIFY | Noisy-action reveal (AW-033) |
| `internal/actions/combat_taunt.go` | MODIFY | Noisy-action reveal (AW-033; idempotent with combat cascade) |
| `internal/configs/balance.go` | MODIFY | New fields: `HiddenMoveStaminaMultiplier`, `SneakModEmitsLight*` |
| `_datafiles/config.yaml` | MODIFY | Defaults for new balance fields |
| `_datafiles/world/dogmud/buffs/9-hidden.yaml` | MODIFY | Drop `triggerrate` and `triggercount` |
| `_datafiles/world/default/buffs/20-very_hidden.yaml` | DELETE | Dead content |
| `internal/characters/context.md` | MODIFY | Document Awareness field, IsHidden() predicate |
| `internal/hooks/context.md` | MODIFY | Document new Awareness hook files |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 1 Done; update progress |

---

## Task 1: Awareness types + transitions + Character field

**Files:**
- Create: `internal/state/awareness/awareness.go`
- Create: `internal/state/awareness/transitions.go`
- Modify: `internal/characters/character.go` (add field, init in `New()`)
- Modify: `internal/characters/validate.go` (init for YAML-loaded chars)

Foundation. State enum, per-state data types, transition table, Machine wrapper, Character field bootstrap.

- [ ] **Step 1: Create `internal/state/awareness/awareness.go`**

```go
// Package awareness defines the Awareness state machine —
// the second consumer of internal/state, after combatphase.
// It replaces the buff-#9 "Hidden flag" as the canonical
// source of "is this character hidden?" Buff #9 stays as the
// side-effect carrier (stat mods, room broadcast text); the
// Awareness machine drives its addition and removal via
// cascade handlers.
package awareness

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Awareness state enum.
type State int

const (
	Visible State = iota
	Concealing
	Hidden
	Revealing
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Visible:
		return "Visible"
	case Concealing:
		return "Concealing"
	case Hidden:
		return "Hidden"
	case Revealing:
		return "Revealing"
	}
	return "Unknown"
}

// VisibleData is empty — default state has no per-state data.
type VisibleData struct{}

// ConcealingData captures an in-flight sneak attempt.
// Today synchronous; chunk-1 sets and clears in one call.
// Future multi-round concealment could populate RoundsUntil.
type ConcealingData struct {
	RoundsUntil int
}

// HiddenData carries hidden-state metadata.
// Today empty; reserved for future light-source tracking or
// per-observer awareness lists.
type HiddenData struct{}

// RevealingData captures the in-flight reveal cascade.
// Reason carries context for subscribers ("why is this character
// being revealed?"). Lifetime is one cascade cycle.
type RevealingData struct {
	Reason state.TransitionReason
}

// Machine wraps state.Machine[State] with awareness-specific
// API including per-state data storage.
type Machine struct {
	inner      *state.Machine[State]
	concealing *ConcealingData
	hidden     *HiddenData
	revealing  *RevealingData
	self       state.ActorRef
}

// NewMachine returns an Awareness machine in Visible.
func NewMachine() *Machine {
	return &Machine{
		inner: state.NewMachine(Visible, validTransitions),
	}
}

// State returns the current state.
func (m *Machine) State() State { return m.inner.State() }

// IsHidden returns true when state is Hidden.
func (m *Machine) IsHidden() bool { return m.State() == Hidden }

// ConcealingData returns the in-flight sneak data if Concealing.
func (m *Machine) ConcealingData() (ConcealingData, bool) {
	if m.State() != Concealing || m.concealing == nil {
		return ConcealingData{}, false
	}
	return *m.concealing, true
}

// RevealingData returns the cascade context if Revealing.
func (m *Machine) RevealingData() (RevealingData, bool) {
	if m.State() != Revealing || m.revealing == nil {
		return RevealingData{}, false
	}
	return *m.revealing, true
}

// Inner returns the underlying state.Machine — used by rules.go
// for hooks. Not part of the stable API.
func (m *Machine) Inner() *state.Machine[State] { return m.inner }

// SetSelf binds the machine to its owning ActorRef. Called from
// the registry during character creation.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }
```

- [ ] **Step 2: Create `internal/state/awareness/transitions.go`**

```go
package awareness

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the Awareness invariant matrix.
// Vetoes layer additional rules on top.
var validTransitions = state.TransitionTable[State]{
	Visible:    {Concealing},
	Concealing: {Hidden, Visible}, // success vs failure
	Hidden:     {Revealing, Visible}, // Revealing for cascade; Visible for force (logout/death)
	Revealing:  {Visible},
}

// Trigger reason constants.
const (
	TriggerSneakCommand       = "sneak_command"
	TriggerSneakSuccess       = "sneak_success"
	TriggerSneakFailed        = "sneak_failed"
	TriggerCombatEntered      = "combat_entered"
	TriggerSurpriseRoundEnd   = "surprise_round_end"
	TriggerMovementDetected   = "movement_detected"
	TriggerObserverSearch     = "observer_search"
	TriggerLightChange        = "light_change"
	TriggerSkullduggeryFailed = "skullduggery_failed"
	TriggerNoisyAction        = "noisy_action"
	TriggerLogout             = "logout_safety_valve"
	TriggerDeath              = "death_cascade"
	TriggerForceVisible       = "force_visible"
)
```

- [ ] **Step 3: Add Awareness field to Character struct**

Open `internal/characters/character.go`. Add field near `CombatPhase`:

```go
// Awareness state machine (chunk 1). Source of truth for
// "is this character hidden?" Buff #9 still exists as effect
// carrier; this machine drives its add/remove via cascade.
Awareness *awareness.Machine `yaml:"-"`
```

Import:

```go
"github.com/GoMudEngine/GoMud/internal/state/awareness"
```

In `New()`, after the existing `CombatPhase: combatphase.NewMachine()` init, add:

```go
Awareness: awareness.NewMachine(),
```

- [ ] **Step 4: Nil-guard init in `Validate()`**

Open `internal/characters/validate.go`. After the existing `CombatPhase == nil` guard, add:

```go
if c.Awareness == nil {
    c.Awareness = awareness.NewMachine()
}
```

- [ ] **Step 5: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 6: Server boot smoke**

```bash
timeout 15 go run main.go 2>&1 | grep -E "LoadDataFiles|panic" | head -10
```
Expected: clean load, all `loadedCount=...` lines, no panics.

- [ ] **Step 7: Commit**

```bash
git add internal/state/awareness/ internal/characters/character.go internal/characters/validate.go
git commit -m "$(cat <<'EOF'
feat(awareness): state types, transition table, Character field

State enum (Visible/Concealing/Hidden/Revealing) with per-state
data types. Machine wrapper. Character.Awareness field
initialized in New() and Validate() (alongside chunk-0
CombatPhase).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Behavior Matrix RED tests

**Files:**
- Create: `internal/state/awareness/awareness_test.go`
- Modify: `internal/state/awareness/awareness.go` (add method stubs so tests compile)

Author the test suite encoding the Behavior Matrix. Tests fail (stubs return errors). Implementation lands in tasks 3-11.

- [ ] **Step 1: Add method stubs + registry to `awareness.go`**

Append to `awareness.go`:

```go
import (
	"errors"
	"sync"
)

// Machine registry for cross-character lookups (parallel to
// combatphase's registry).
var (
	registryMu      sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
	m.SetSelf(ref)
}

func UnregisterMachine(ref state.ActorRef) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(machineRegistry, ref)
}

func lookupMachine(ref state.ActorRef) *Machine {
	registryMu.Lock()
	defer registryMu.Unlock()
	return machineRegistry[ref]
}

// Transition method stubs — implementations in Tasks 3-11.

func (m *Machine) TransitionToConcealing(d ConcealingData, r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) ResolveConcealment(success bool, r state.TransitionReason) {}
func (m *Machine) TransitionToRevealing(r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) ForceVisible(r state.TransitionReason)                   {}
func (m *Machine) RegisterActivityCheck(check func() bool)                 {}
func (m *Machine) RegisterRoomBroadcast(fn func(text, leaveText string))   {}
func (m *Machine) SubscribeToTransitions(fn func(from, to State, r state.TransitionReason)) {}
```

- [ ] **Step 2: Create `internal/state/awareness/awareness_test.go`**

Author the full Behavior Matrix as ~33 tests (CP-style). The full code block is long; outline by category, each test using `require.NoError` / `require.ErrorIs` / `require.Equal`. Tests should:

- **AW-001 through AW-003 (basic sneak resolve)**: `TestAW_001_SneakAloneSucceeds` constructs a Machine, calls `TransitionToConcealing` + `ResolveConcealment(true, ...)`, asserts state == Hidden.
- **AW-004 through AW-011 (detection rolls)**: tests register two Machines (sneaker + observer); tests use a stub-roll mechanism where the test pre-sets a roll outcome flag.
- **AW-012 through AW-014 (light changes)**: tests stub `EmitsLight` and room `IsLit` and call `LightStateChanged()` method (added in Task 6).
- **AW-015 through AW-018 (Combat Phase cascade)**: tests construct a Machine, simulate a Combat Phase transition via `Inner().TransitionTo`, assert Awareness cascade fired.
- **AW-019, AW-020 (stamina cost)**: framework-level tests just verify the `Character.IsHidden()` predicate; the actual movement-cost integration is tested via integration tests in Task 7 or smoke.
- **AW-021, AW-022 (logout)**: tests verify the explicit `ForceVisible` method on a Hidden machine transitions through Revealing → Visible.
- **AW-023 (Activity veto)**: tests register an activity-check callback returning false; assert `TransitionToConcealing` returns `state.ErrVetoed`.
- **AW-028, AW-029 (persistence)**: tests verify `NewMachine().State() == Visible`.
- **AW-030, AW-031 (Revealing semantics)**: tests verify Revealing fires cascade subscribers, transitions to Visible same-tick.
- **AW-032, AW-033 (noisy actions)**: framework-level tests assert that explicit `TransitionToRevealing(TriggerNoisyAction)` works; per-command integration is verified via integration in Task 11.

Each test function follows the pattern from `internal/state/combatphase/combatphase_test.go`. Example:

```go
package awareness

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/require"
)

func makePair() (sneaker, observer *Machine) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()
	A := NewMachine()
	B := NewMachine()
	RegisterMachine(actor(1), A)
	RegisterMachine(actor(2), B)
	return A, B
}

func actor(userId int) state.ActorRef { return state.ActorRef{UserId: userId} }

// AW-001: Sneak alone succeeds.
func TestAW_001_SneakAloneSucceeds(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand}))
	A.ResolveConcealment(true, state.TransitionReason{Trigger: TriggerSneakSuccess})
	require.Equal(t, Hidden, A.State())
}

// AW-002: Sneak fails when observer wins.
func TestAW_002_SneakObserverDetects(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand}))
	A.ResolveConcealment(false, state.TransitionReason{Trigger: TriggerSneakFailed})
	require.Equal(t, Visible, A.State())
}

// AW-023: Activity blocks Concealing.
func TestAW_023_ActivityBlocksConcealing(t *testing.T) {
	A, _ := makePair()
	A.RegisterActivityCheck(func() bool { return false /* not Free */ })
	err := A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand})
	require.ErrorIs(t, err, state.ErrVetoed)
	require.Equal(t, Visible, A.State())
}

// AW-028: Fresh machine is Visible.
func TestAW_028_FreshMachineIsVisible(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Visible, m.State())
}

// ... ~28 more tests covering the rest of the matrix
```

Author all 33 tests. Many follow this skeleton. See `awareness-design.md` Behavior Matrix section for the full row text.

- [ ] **Step 3: Run tests, expect FAIL**

```bash
go test ./internal/state/awareness/ -v
```
Expected: tests compile, most FAIL with "not implemented" or invariant violations. A few coincidentally pass (e.g., TestAW_028 which just checks initial state) — fine.

- [ ] **Step 4: Build verify**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/state/awareness/
git commit -m "$(cat <<'EOF'
test(awareness): Behavior Matrix RED — AW-001 through AW-033

33 intent-driven tests encoding the full matrix. Method stubs
make tests compile. Implementation lands in chunk-1 Tasks 3-11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Basic transitions (AW-001 through AW-003, AW-028, AW-031)

**Files:**
- Modify: `internal/state/awareness/awareness.go` (transition methods)
- Create: `internal/state/awareness/rules.go` (vetos + cascade helper plumbing)

Implement `TransitionToConcealing`, `ResolveConcealment`, `TransitionToRevealing`, `ForceVisible`, plus the registry plumbing. After this task, the basic-sneak-resolve matrix rows pass.

- [ ] **Step 1: Create `internal/state/awareness/rules.go`**

```go
package awareness

import "github.com/GoMudEngine/GoMud/internal/state"

// vetoChain holds registered veto functions per concern.
// Activity check is the chunk-1 pre-wire for the future
// Activity machine; chunk 3 will repoint to the real machine.
type vetoChain struct {
	activitySelf func() bool // Activity == Free returns true
}

func (m *Machine) RegisterActivityCheck(check func() bool) {
	m.vetoes.activitySelf = check
}
```

Add `vetoes vetoChain` field to the `Machine` struct in `awareness.go`.

- [ ] **Step 2: Implement TransitionToConcealing in rules.go**

```go
// TransitionToConcealing initiates a sneak attempt. Runs the
// Activity veto first (can't sneak while casting/crafting).
// Stores ConcealingData; caller is responsible for calling
// ResolveConcealment with the roll outcome.
func (m *Machine) TransitionToConcealing(d ConcealingData, r state.TransitionReason) error {
	if m.vetoes.activitySelf != nil && !m.vetoes.activitySelf() {
		return &state.VetoError{HandlerName: "activity_self",
			Reason: "busy with activity"}
	}
	if err := m.inner.TransitionTo(Concealing, r); err != nil {
		return err
	}
	m.concealing = &d
	return nil
}
```

- [ ] **Step 3: Implement ResolveConcealment**

```go
// ResolveConcealment finalizes the sneak attempt.
// success=true → Hidden; success=false → Visible.
func (m *Machine) ResolveConcealment(success bool, r state.TransitionReason) {
	if m.State() != Concealing {
		return
	}
	target := Visible
	if success {
		target = Hidden
	}
	_ = m.inner.TransitionTo(target, r)
	m.concealing = nil
	if success {
		m.hidden = &HiddenData{}
	} else {
		m.hidden = nil
	}
}
```

- [ ] **Step 4: Implement TransitionToRevealing**

```go
// TransitionToRevealing transitions Hidden → Revealing →
// Visible same-tick. Cascade subscribers fire during Revealing;
// then Visible is set immediately.
//
// If the actor is NOT Hidden, this is a no-op (idempotent).
func (m *Machine) TransitionToRevealing(r state.TransitionReason) error {
	if m.State() != Hidden {
		return nil // idempotent; not an error
	}
	if err := m.inner.TransitionTo(Revealing, r); err != nil {
		return err
	}
	m.hidden = nil
	m.revealing = &RevealingData{Reason: r}

	// Fire cascade subscribers via inner.AfterTransition (already
	// wired by state.Machine framework). Then immediately drop
	// to Visible same-tick.
	_ = m.inner.TransitionTo(Visible, state.TransitionReason{
		Trigger: r.Trigger, // preserve cause for downstream observers
		Actor:   r.Actor,
		Target:  r.Target,
	})
	m.revealing = nil
	return nil
}
```

- [ ] **Step 5: Implement ForceVisible**

```go
// ForceVisible drops to Visible from any state, used by logout
// safety valve, death cascade, charm changes, etc. If currently
// Hidden, routes through Revealing (so cascade subscribers
// fire). Otherwise transitions directly.
func (m *Machine) ForceVisible(r state.TransitionReason) {
	switch m.State() {
	case Hidden:
		_ = m.TransitionToRevealing(r)
	case Concealing:
		_ = m.inner.TransitionTo(Visible, r)
		m.concealing = nil
	case Revealing:
		// already in flight; nothing to do
	}
}
```

- [ ] **Step 6: Verify AW-001, AW-002, AW-003, AW-028, AW-031 pass**

```bash
go test ./internal/state/awareness/ -v -run "TestAW_00[1-3]|TestAW_028|TestAW_031"
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/state/awareness/
git commit -m "$(cat <<'EOF'
feat(awareness): basic transitions AW-001 through AW-003

TransitionToConcealing + ResolveConcealment + TransitionToRevealing
+ ForceVisible. Activity-veto registration. Behavior Matrix
rows for basic sneak resolve + persistence + Revealing
semantics now pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Detection rolls + room entry (AW-004 through AW-011)

**Files:**
- Modify: `internal/state/awareness/rules.go`
- Modify: `internal/state/awareness/awareness_test.go` (potentially adjust tests if helpers changed)

Implement detection-roll wiring. The Awareness machine itself doesn't roll dice; the detection caller does. Awareness exposes a clean API for "I rolled, here's the outcome." This task wires up the test patterns and adds any helpers needed.

The actual detection logic stays in `internal/hooks/go.go` (room entry) and observer-command handlers; this task ensures the Awareness API can receive their results.

- [ ] **Step 1: Verify the `TransitionToRevealing` is sufficient for detection outcomes**

The four trigger reasons cover all detection events:
- `TriggerMovementDetected` — failed movement roll
- `TriggerObserverSearch` — failed observer-initiated search
- `TriggerLightChange` — failed re-roll after light change
- `TriggerSkullduggeryFailed` — failed steal/plant detection

All are handled by the same `TransitionToRevealing(reason)` API. No new methods needed.

- [ ] **Step 2: Add a convenience helper for the room-entry case**

In `rules.go`, add:

```go
// NotifyRoomChanged is called when the actor changes room.
// Caller (typically go.go) has already run per-observer
// detection rolls; this method ensures any failure transitions
// the machine through Revealing. If success, Hidden persists.
//
// Per the spec, the detection ROLLS happen at the call site
// (which has room/observer/sneaker all in scope); this method
// just receives the outcome.
func (m *Machine) NotifyRoomChanged(detected bool, r state.TransitionReason) {
	if detected && m.State() == Hidden {
		_ = m.TransitionToRevealing(r)
	}
}
```

- [ ] **Step 3: Verify detection-related matrix rows pass**

```bash
go test ./internal/state/awareness/ -v -run "TestAW_00[4-9]|TestAW_01[0-1]"
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/state/awareness/
git commit -m "$(cat <<'EOF'
feat(awareness): detection roll outcomes AW-004 through AW-011

NotifyRoomChanged helper plus the existing TransitionToRevealing
handle all four detection trigger reasons.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Combat Phase cascade subscription (AW-015 through AW-018)

**Files:**
- Create: `internal/hooks/Awareness_Cascades.go`

The biggest cross-machine integration. The Awareness machine subscribes to Combat Phase transitions and the `OnEndOfRoundIfSurprise` callback. Non-surprise combat-entry triggers Awareness Hidden → Revealing; surprise combat-entry preserves Hidden through the engagement; end-of-first-round consumes surprise via the callback hook.

- [ ] **Step 1: Create `internal/hooks/Awareness_Cascades.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
)

// wireAwarenessFromCombatPhase registers a cascade handler on
// each character's CombatPhase that fires Awareness Hidden →
// Revealing on combat entry, EXCEPT when the trigger reason is
// TriggerSurpriseAttack (which preserves Hidden through the
// surprise round). Also subscribes to OnEndOfRoundIfSurprise to
// consume surprise at end of the first combat round.
//
// Also wires the cascade that maintains buff #9 alongside the
// Awareness state — when Awareness enters Hidden, apply buff #9;
// when Awareness leaves Hidden, remove it.
func wireAwarenessFromCombatPhase(c *characters.Character) {
	// Combat Phase Idle → Engaging cascade.
	c.CombatPhase.Inner().AfterTransition("awareness_combat_cascade",
		func(from, to combatphase.State, r state.TransitionReason) {
			if from == combatphase.Idle && to == combatphase.Engaging {
				if r.Trigger == combatphase.TriggerSurpriseAttack {
					return // preserve Hidden through Engaging
				}
				if c.Awareness.State() == awareness.Hidden {
					_ = c.Awareness.TransitionToRevealing(state.TransitionReason{
						Trigger: awareness.TriggerCombatEntered,
					})
				}
			}
		})

	// Surprise-round-end callback subscription.
	c.CombatPhase.OnEndOfRoundIfSurprise(func(r state.TransitionReason) {
		if c.Awareness.State() == awareness.Hidden {
			_ = c.Awareness.TransitionToRevealing(state.TransitionReason{
				Trigger: awareness.TriggerSurpriseRoundEnd,
			})
		}
	})

	// Awareness state → buff #9 mirror.
	c.Awareness.Inner().AfterTransition("awareness_buff_mirror",
		func(from, to awareness.State, r state.TransitionReason) {
			switch {
			case to == awareness.Hidden:
				// Apply buff #9 (no duration, just stat-mod
				// and flag carrier).
				c.AddBuff(9, "awareness")
			case from == awareness.Hidden && (to == awareness.Revealing || to == awareness.Visible):
				// Remove buff #9 via flag-based cancel.
				c.CancelBuffsWithFlag(buffs.Hidden)
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireAwarenessFromCombatPhase)
}
```

- [ ] **Step 2: Build + test**

```bash
go build ./...
go test ./internal/state/awareness/ -v -run "TestAW_01[5-8]"
go test ./internal/state/combatphase/ -v 2>&1 | tail -3
```
Expected: chunk-1 surprise-cascade tests pass; chunk-0 Combat Phase tests still 32/32 green.

- [ ] **Step 3: Boot server smoke**

```bash
timeout 15 go run main.go 2>&1 | grep -E "LoadDataFiles|panic" | head -10
```
Expected: clean load, no panics.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Awareness_Cascades.go
git commit -m "$(cat <<'EOF'
feat(awareness): Combat Phase cascade subscription + buff #9 mirror

Hidden → Revealing on non-surprise combat-entry; preserved
through Engaging with SurpriseAttack reason; consumed at
end-of-first-round via OnEndOfRoundIfSurprise callback.
Awareness state mirrors to buff #9 (apply on Hidden enter,
remove on Hidden exit).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Light state change handler + CalcSneakScore refactor (AW-012 through AW-014, AW-024 through AW-027)

**Files:**
- Modify: `internal/actions/skill_helpers.go` (CalcSneakScore signature + 4-way conditional)
- Modify: every caller of `CalcSneakScore` (update to new signature)
- Modify: `internal/configs/balance.go` (new fields)
- Modify: `_datafiles/config.yaml` (defaults)
- Create: `internal/hooks/Awareness_LightChange.go`

The biggest substantive change in chunk 1. CalcSneakScore signature change cascades to all callers; the light-change hook is new infrastructure.

- [ ] **Step 1: Add balance config fields**

In `internal/configs/balance.go`, add to the Balance struct:

```go
HiddenMoveStaminaMultiplier float64 `yaml:"HiddenMoveStaminaMultiplier"`
SneakModEmitsLightDarkRoom  float64 `yaml:"SneakModEmitsLightDarkRoom"`
SneakModEmitsLightLitRoom   float64 `yaml:"SneakModEmitsLightLitRoom"`
SneakModNoLightLitRoom      float64 `yaml:"SneakModNoLightLitRoom"`
```

In the `DefaultBalance()` function (or wherever defaults live):

```go
HiddenMoveStaminaMultiplier: 3.0,
SneakModEmitsLightDarkRoom:  0.5,
SneakModEmitsLightLitRoom:   0.85,
SneakModNoLightLitRoom:      0.9,
```

In `_datafiles/config.yaml`, add under the Balance section:

```yaml
HiddenMoveStaminaMultiplier: 3.0
SneakModEmitsLightDarkRoom: 0.5
SneakModEmitsLightLitRoom: 0.85
SneakModNoLightLitRoom: 0.9
```

- [ ] **Step 2: Refactor CalcSneakScore**

Open `internal/actions/skill_helpers.go`. Replace the existing signature:

```go
// CalcSneakScore computes a sneak score with light-conditional
// modifier. effectiveLit reflects the room visibility from the
// observer's POV — true if room is lit OR observer has NightVision.
//
// Caller is responsible for computing effectiveLit per observer.
// Convenience helper CalcSneakScoreVsObserver wraps the common case.
func CalcSneakScore(char *characters.Character, effectiveLit bool) float64 {
	base := float64(char.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(char.GetSkillLevel(skills.Skullduggery))*25.0 +
		mutationStealthBonus(char.Mutations)

	cfg := configs.GetBalanceConfig()
	emits := char.EmitsLight()

	switch {
	case emits && !effectiveLit:
		base *= cfg.SneakModEmitsLightDarkRoom // default 0.5
	case emits && effectiveLit:
		base *= cfg.SneakModEmitsLightLitRoom // default 0.85
	case !emits && effectiveLit:
		base *= cfg.SneakModNoLightLitRoom // default 0.9
		// else baseline (no mod applied)
	}
	return base
}

// CalcSneakScoreVsObserver is a convenience for the common
// detection-roll case where the caller has sneaker + observer +
// room in scope.
func CalcSneakScoreVsObserver(sneaker, observer *characters.Character, room *rooms.Room) float64 {
	effectiveLit := room.GetVisibility() >= 1 ||
		observer.HasFlagFromAnySource(buffs.NightVision)
	return CalcSneakScore(sneaker, effectiveLit)
}
```

Verify imports for `configs`, `buffs`, `rooms` are present.

- [ ] **Step 3: Migrate every caller of CalcSneakScore**

```bash
grep -rn "CalcSneakScore" --include="*.go" internal/
```

Likely callers:
- `internal/hooks/go.go` (room-entry detection)
- `internal/actions/sneak.go`
- `internal/actions/shadow.go`
- `internal/actions/steal.go` / `plant.go` (skullduggery detection)
- Existing `internal/usercommands/stealth_detection.go` was deleted in chunk 2.7 Task 16; check no stale references

For each call site, replace:
- `CalcSneakScore(char)` → `CalcSneakScoreVsObserver(char, observer, room)` when an observer is in scope
- `CalcSneakScore(char)` → `CalcSneakScore(char, false /* dark */)` for callers without an observer (e.g., self-only sneak validation — rare)

The bulk of detection rolls already have observer in scope. Audit each site individually.

- [ ] **Step 4: Create `internal/hooks/Awareness_LightChange.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// LightChangeHandler fires re-rolls for hidden actors in a room
// when the room's effective light state changes. Triggers:
//   1. A new actor with EmitsLight enters the room.
//   2. The last EmitsLight actor leaves the room.
//   3. A hidden actor's own emission state changes (equips/
//      removes torch, casts/cancels light spell, etc.)
//
// For each hidden actor still in the room, re-roll detection
// against every observer using the new (effectiveLit, emits)
// combination. On detection failure, transition Hidden → Revealing.
type lightChangeListener struct{}

func (lightChangeListener) RoomChanged(evt events.RoomChange) {
	room := rooms.LoadRoom(evt.ToRoomId)
	if room == nil {
		return
	}
	reRollDetectionForHiddenActors(room, state.TransitionReason{Trigger: awareness.TriggerLightChange})
}

func (lightChangeListener) EquipmentChanged(evt events.EquipmentChange) {
	if evt.UserId == 0 && evt.MobInstanceId == 0 {
		return
	}
	// If the actor is currently hidden, re-roll all observers.
	var c *characters.Character
	var roomId int
	if evt.UserId > 0 {
		if u := users.GetByUserId(evt.UserId); u != nil {
			c = &u.Character
			roomId = u.Character.RoomId
		}
	}
	if evt.MobInstanceId > 0 {
		if m := mobs.GetInstance(evt.MobInstanceId); m != nil {
			c = &m.Character
			roomId = m.Character.RoomId
		}
	}
	if c == nil || c.Awareness.State() != awareness.Hidden {
		return
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	reRollDetectionForHiddenActor(c, room, state.TransitionReason{Trigger: awareness.TriggerLightChange})
}

// reRollDetectionForHiddenActors re-rolls every hidden actor
// in the room against every observer.
func reRollDetectionForHiddenActors(room *rooms.Room, reason state.TransitionReason) {
	// Implementation: walk room.GetPlayers() and room.GetMobs(),
	// for each with Awareness=Hidden, run reRollDetectionForHiddenActor.
	// (Full implementation needs current room iteration — port from
	// existing detection code in internal/hooks/go.go.)
}

// reRollDetectionForHiddenActor re-rolls a single hidden actor
// against every observer in the room. On detection failure,
// transitions Hidden → Revealing.
func reRollDetectionForHiddenActor(sneaker *characters.Character, room *rooms.Room, reason state.TransitionReason) {
	// Implementation: walk observers, call CalcSneakScoreVsObserver,
	// call CalcSearchScore on observer, dice.OpposedRollStat.
	// If detected → sneaker.Awareness.TransitionToRevealing(reason).
}

func init() {
	listener := lightChangeListener{}
	events.Subscribe(listener.RoomChanged)
	events.Subscribe(listener.EquipmentChanged)
}
```

Note: this is a scaffolding file. The actual `reRollDetectionForHiddenActor` body needs to port logic from `internal/hooks/go.go`'s existing detection-roll code. Implementer will need to read `go.go` carefully and adapt.

- [ ] **Step 5: Build + verify tests**

```bash
go build ./...
go test ./internal/actions/ ./internal/hooks/ ./internal/state/awareness/ -v 2>&1 | grep -E "^ok|FAIL" | head -10
```
Expected: clean, all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/skill_helpers.go internal/hooks/Awareness_LightChange.go internal/configs/balance.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(awareness): light-conditional sneak score + light-change re-roll

CalcSneakScore takes effectiveLit bool; CalcSneakScoreVsObserver
helper computes effectiveLit (room lit OR observer NightVision)
per-observer. Four-way conditional applies sneaker-side modifier.
LightChange hook fires re-rolls when room or actor emission
state changes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Stamina cost for hidden movement (AW-019, AW-020)

**Files:**
- Modify: `internal/usercommands/go.go` (movement stamina multiplier)

- [ ] **Step 1: Locate the movement stamina-cost site**

```bash
grep -n "Stamina\.\|StaminaCost\|stamina cost\|MoveStamina\|encumbranceMod" internal/usercommands/go.go | head -10
```

- [ ] **Step 2: Add the Hidden multiplier**

After the encumbrance multiplier is applied (find the `moveCost *= encumbranceMod` or equivalent line), add:

```go
if user.Character.IsHidden() {
    moveCost *= configs.GetBalanceConfig().HiddenMoveStaminaMultiplier
}
```

Place AFTER the encumbrance multiplier so they stack multiplicatively.

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/usercommands/ -v 2>&1 | grep -E "^ok|FAIL" | head -5
```

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/go.go
git commit -m "$(cat <<'EOF'
feat(awareness): hidden-movement stamina multiplier (default 3.0×)

Stacks multiplicatively with encumbrance. Hidden+over-capacity
characters pay base × encumbranceMod × hiddenMod (e.g., 5×3 =
15× cost), reinforcing that stealth wants you light.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Logout safety valve (AW-021, AW-022)

**Files:**
- Create: `internal/hooks/Logout_AwarenessCleanup.go`

- [ ] **Step 1: Find the logout/disconnect cleanup site**

```bash
grep -rn "func.*Logout\|func.*Disconnect" --include="*.go" internal/users/ internal/connections/ 2>/dev/null | head -10
```

- [ ] **Step 2: Create the hook**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Logout_AwarenessCleanup forces Awareness through Revealing →
// Visible synchronously before session cleanup. Ensures hidden
// players don't leave the world Hidden; on reconnect, they're
// Visible.
func init() {
	events.Subscribe(func(evt events.PlayerLogout) {
		u := users.GetByUserId(evt.UserId)
		if u == nil {
			return
		}
		u.Character.Awareness.ForceVisible(state.TransitionReason{
			Trigger: awareness.TriggerLogout,
			Actor:   state.ActorRef{UserId: evt.UserId},
		})
	})
}
```

Verify `events.PlayerLogout` exists; if not, use the actual event type name (grep for logout-emit sites).

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/hooks/ ./internal/state/awareness/ -v 2>&1 | grep -E "^ok|FAIL" | head -5
```

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/Logout_AwarenessCleanup.go
git commit -m "$(cat <<'EOF'
feat(awareness): logout safety valve — force Visible before session cleanup

Hidden players who quit see room broadcast fire before disconnect;
reconnecting they're Visible. Mob despawn handled by instance
destruction (no separate cascade needed).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Activity veto pre-wire (AW-023)

**Files:**
- Create: `internal/hooks/Awareness_Vetoes.go`

- [ ] **Step 1: Create the veto wiring hook**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// wireAwarenessVetoes registers the Activity-check callback that
// vetoes Visible → Concealing when the character is busy with a
// multi-round activity. Chunk-3 (Activity machine) will repoint
// this callback to the real Activity machine query; for chunk 1
// it consults the existing CastingState/CraftingState pointers.
func wireAwarenessVetoes(c *characters.Character) {
	c.Awareness.RegisterActivityCheck(func() bool {
		return c.CastingState == nil && c.CraftingState == nil
	})
}

func init() {
	characters.OnCharacterCreated(wireAwarenessVetoes)
}
```

- [ ] **Step 2: Build + test AW-023**

```bash
go build ./...
go test ./internal/state/awareness/ -v -run "TestAW_023"
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/Awareness_Vetoes.go
git commit -m "$(cat <<'EOF'
feat(awareness): Activity veto pre-wire (chunk-3 will repoint)

Visible → Concealing blocked when CastingState or CraftingState
is non-nil. Chunk-3 Activity machine will replace this with the
proper machine query.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: IsHidden() predicate + reader migration

**Files:**
- Modify: `internal/characters/character.go` (predicate)
- Migrate: ~30 callsites of `HasBuffFlag(buffs.Hidden)` → `IsHidden()`

- [ ] **Step 1: Add IsHidden() predicate**

In `internal/characters/character.go`:

```go
// IsHidden returns true when Awareness state is Hidden.
// Replacement for HasBuffFlag(buffs.Hidden) reads.
func (c *Character) IsHidden() bool {
	if c.Awareness == nil {
		return false
	}
	return c.Awareness.IsHidden()
}
```

- [ ] **Step 2: Grep + migrate**

```bash
grep -rn "HasBuffFlag(buffs.Hidden)\|HasFlagFromAnySource(buffs.Hidden)" --include="*.go" internal/
```

For each hit, replace:
- `c.HasBuffFlag(buffs.Hidden)` → `c.IsHidden()`
- `c.HasFlagFromAnySource(buffs.Hidden)` → `c.IsHidden()`

Note: `HasFlagFromAnySource` may have semantics around permabuffs/temp buffs that `IsHidden()` doesn't capture. Per the spec, Awareness state is the source of truth post-chunk-1, so the migration is correct — but read each callsite carefully to ensure no permabuff-specific logic is lost.

Expected file count: ~20-25 files modified, ~30 callsites.

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./... 2>&1 | grep -E "^ok|FAIL" | head -30
```
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
refactor: migrate HasBuffFlag(buffs.Hidden) → Character.IsHidden()

~30 callsites across actions/, behaviortree/, characters/,
hooks/, usercommands/. Awareness state is now the canonical
source of truth for "is this character hidden?"

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Writer migration (sneak/steal/plant/remove_equip + noisy actions)

**Files:**
- Modify: `internal/actions/sneak.go` (use Awareness transitions instead of AddBuff)
- Modify: `internal/actions/steal.go`, `plant.go`, `remove_equip.go` (use TransitionToRevealing instead of CancelBuffsWithFlag)
- Modify: `internal/actions/say.go` / `internal/usercommands/shout.go` / `whisper.go` / `combat_rally.go` / `combat_warcry.go` / `combat_taunt.go` (noisy-action reveal calls)

- [ ] **Step 1: Migrate sneak action**

In `internal/actions/sneak.go`, find the success path (currently `actor.AddBuff(9, "skill")` per the survey at line ~116). Replace:

```go
// OLD:
actor.AddBuff(9, "skill")
char.SetMiscData("sneaking", true)

// NEW:
char := actor.GetCharacter()
if err := char.Awareness.TransitionToConcealing(
    awareness.ConcealingData{},
    state.TransitionReason{Trigger: awareness.TriggerSneakCommand},
); err != nil {
    // Veto fired (e.g., busy with activity); error path
    actor.SendText(/* veto-specific message */)
    return SneakResult{Reason: "vetoed"}
}
char.Awareness.ResolveConcealment(true, state.TransitionReason{
    Trigger: awareness.TriggerSneakSuccess,
})
// (Existing misc-data sneaking flag preserved for pre-tick edge case)
char.SetMiscData("sneaking", true)
```

For the failure path (sneak roll lost to observer), call:

```go
char.Awareness.ResolveConcealment(false, state.TransitionReason{
    Trigger: awareness.TriggerSneakFailed,
})
```

The cascade in `Awareness_Cascades.go` (Task 5) handles buff #9 add/remove automatically.

- [ ] **Step 2: Migrate steal/plant/remove_equip explicit cancels**

```bash
grep -rn "CancelBuffsWithFlag(buffs.Hidden)" --include="*.go" internal/actions/
```

For each hit, replace:

```go
// OLD:
actor.GetCharacter().CancelBuffsWithFlag(buffs.Hidden)

// NEW:
actor.GetCharacter().Awareness.TransitionToRevealing(
    state.TransitionReason{Trigger: awareness.TriggerSkullduggeryFailed},
)
```

Reason trigger should match the context (TriggerSkullduggeryFailed for steal/plant failures; TriggerNoisyAction for movement-induced reveals; etc.). Cascade handles buff removal.

- [ ] **Step 3: Add noisy-action reveal triggers (AW-032, AW-033)**

For each command handler that should trigger noisy reveal:
- `internal/actions/say.go` — but only for `say` (room-broadcast), not whisper
- `internal/usercommands/shout.go`
- `internal/usercommands/whisper.go` — only for the room-broadcast variant (no target). The targeted form stays quiet.
- `internal/actions/combat_rally.go`
- `internal/actions/combat_warcry.go`
- `internal/actions/combat_taunt.go`

Add at the start of each handler:

```go
if char := actor.GetCharacter(); char.IsHidden() {
    char.Awareness.TransitionToRevealing(state.TransitionReason{
        Trigger: awareness.TriggerNoisyAction,
        Metadata: map[string]any{"command": "say" /* or whatever */},
    })
}
```

For `taunt` and `warcry` (which also enter Combat Phase), the noisy-action reveal is idempotent with the cascade from combat-entry — first call transitions to Revealing/Visible, second call no-ops.

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./... 2>&1 | grep -E "^ok|FAIL" | head -20
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/ internal/usercommands/
git commit -m "$(cat <<'EOF'
refactor: migrate sneak/steal/plant/remove_equip + noisy-action stealth

Sneak success path uses Awareness.TransitionToConcealing +
ResolveConcealment(true). Steal/plant/remove_equip explicit
cancels use Awareness.TransitionToRevealing. say/shout/whisper
(broadcast)/rally/warcry/taunt break stealth via
TriggerNoisyAction. Cascade handles buff #9 add/remove.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: YAML changes (buff 9 no duration + delete very_hidden)

**Files:**
- Modify: `_datafiles/world/dogmud/buffs/9-hidden.yaml`
- Delete: `_datafiles/world/default/buffs/20-very_hidden.yaml`

- [ ] **Step 1: Drop duration from buff 9**

Edit `_datafiles/world/dogmud/buffs/9-hidden.yaml` — remove the `triggerrate` and `triggercount` lines:

```yaml
buffid: 9
name: Hidden
description: You're very sneaky.
secret: false
flags:
  - hidden
  - cancel-on-combat
start_user_text: "You feel sneaky."
start_room_text: "{source_plain} disappears into the shadows."
end_user_text: "You no longer feel sneaky."
end_room_text: "{source_plain} emerges from the shadows."
```

(No `triggerrate`, no `triggercount`. Buff persists until explicit cancellation by Awareness machine cascade.)

- [ ] **Step 2: Delete very_hidden**

```bash
git rm _datafiles/world/default/buffs/20-very_hidden.yaml
```

- [ ] **Step 3: Boot server to verify YAML changes load cleanly**

```bash
timeout 15 go run main.go 2>&1 | grep -E "LoadDataFiles|panic" | head -10
```
Expected: `buffSpec.LoadDataFiles() loadedCount=...` (decremented by 1 since we deleted #20), no panics.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(buffs): drop Hidden duration; delete dead very_hidden buff

Buff #9 (Hidden) no longer has triggerrate/triggercount —
persists until Awareness machine cascade removes it. Buff #20
(very_hidden) deleted: dead content with no production
consumers (the chunk-2.7 survey found only test-fixture
references).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Documentation

**Files:**
- Create: `internal/state/awareness/context.md`
- Modify: `internal/characters/context.md` (document Awareness field, IsHidden() predicate)
- Modify: `internal/hooks/context.md` (document new Awareness hook files)
- Modify: `internal/behaviortree/context.md` (note IsHidden() migration)

Follow the chunk-0 documentation style. Each context.md gets a "Chunk 1 — Awareness integration" section. The new `internal/state/awareness/context.md` is the package-level doc.

- [ ] **Step 1: Author `internal/state/awareness/context.md`** (~150-200 lines, follow `internal/state/combatphase/context.md` as template)
- [ ] **Step 2: Append "Awareness Integration (chunk 1)" to `internal/characters/context.md`**
- [ ] **Step 3: Append to `internal/hooks/context.md`** documenting the three new files (Awareness_Vetoes, Awareness_Cascades, Awareness_LightChange) + Logout_AwarenessCleanup
- [ ] **Step 4: Note in `internal/behaviortree/context.md`** that conditions/actions reading hidden state now use `IsHidden()` predicate

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
docs(awareness): chunk 1 context.md updates

New state/awareness/context.md; integration sections in
characters/, hooks/, and behaviortree/ context.mds.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Build/test/smoke validation

**Files:** (verification only)

- [ ] **Step 1: Full build**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 2: Full test suite**

```bash
go test ./...
```
Expected: every package PASS. Note any failures.

- [ ] **Step 3: Behavior Matrix green**

```bash
go test ./internal/state/awareness/ -v 2>&1 | grep -E "^--- (PASS|FAIL):" | head -40
```
Expected: 33 PASS, 0 FAIL.

- [ ] **Step 4: Combat Phase regression check**

```bash
go test ./internal/state/combatphase/ -v 2>&1 | tail -3
```
Expected: still 32/32 PASS — chunk-0 surprise-attack hand-off still works correctly post-cascade-subscription.

- [ ] **Step 5: Server boot**

```bash
go run main.go
```
Use `run_in_background: true`. Wait ~15s. Check all `LoadDataFiles()` markers, no panics, "Server Ready" appears. Kill server.

- [ ] **Step 6: In-game smoke (per spec section "Smoke scenarios")**

10 scenarios from the spec. Likely test-mud session — author goals file
`tools/testing/goals/chunk-1-awareness-state-machine.yaml` and run.

- [ ] **Step 7: Kill any test servers**

Per project SOP.

---

## Task 15: Roadmap closeout

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 1 Done** with shipped paragraph summarizing what landed (Behavior Matrix complete, mechanic refresh, light-conditional sneak, etc.).

- [ ] **Step 2: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 1 (Awareness machine) Done

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Tasks |
|--------------|-------|
| Awareness types | Task 1 |
| Behavior Matrix tests (RED) | Task 2 |
| Basic sneak resolve (AW-001 to AW-003, AW-028, AW-031) | Task 3 |
| Detection rolls (AW-004 to AW-011) | Task 4 |
| Combat Phase cascade (AW-015 to AW-018) | Task 5 |
| Light state + CalcSneakScore (AW-012-014, AW-024-027) | Task 6 |
| Stamina cost (AW-019, AW-020) | Task 7 |
| Logout safety valve (AW-021, AW-022) | Task 8 |
| Activity veto pre-wire (AW-023) | Task 9 |
| Reader migration | Task 10 |
| Writer migration + noisy actions (AW-032, AW-033) | Task 11 |
| YAML sunset | Task 12 |
| Documentation | Task 13 |
| Build/smoke validation | Task 14 |
| Roadmap closeout | Task 15 |

All spec sections covered. Behavior Matrix rows AW-001 through AW-033 each map to a Task 2 RED test, with implementation distributed across Tasks 3-11.

## Known followups (out of chunk 1)

- Activity machine (chunk 3) will repoint the Activity veto callback from `CastingState/CraftingState` nil-check to the proper machine query.
- Life machine (chunk 2) will provide a cleaner Life→Dead trigger for the Awareness force-Visible cascade than the current cancel-on-combat-flag indirection.
- Per-observer detection state (if ever wanted) is an out-of-scope future expansion; today's Awareness is global per actor.
- Sneak skill mechanics rework (skill rolls, mutation bonuses, etc.) stays out of scope.
