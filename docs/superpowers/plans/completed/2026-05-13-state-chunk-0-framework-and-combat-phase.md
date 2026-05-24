# Combat State — Chunk 0: Framework + Combat Phase Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Co-develop the `internal/state/` state-machine framework with the first consumer (Combat Phase). Replace `Character.Aggro` field, delete the Stage 2a unused unified handler, introduce the `Combatant` flag, and structurally fix the chunk 2.7 thief-archetype bug — all via intent-driven TDD against the ~38-row Behavior Matrix.

**Architecture:** Generic `state.Machine[S]` parameterized over each machine's state enum. Parameterized state types carry data (Engaged carries target). Synchronous veto-then-cascade hooks; framework-maintained `Attackers()` inbound list. Round driver consults Combat Phase for tick dispatch (`mob_combat_round`/`mob_idle`). Hard cutover within the chunk — `Character.Aggro` deleted at end.

**Tech Stack:** Go 1.21+ with generics. Existing engine: `internal/characters/`, `internal/hooks/NewRound_DoCombat*.go`, `internal/behaviortree/`, `internal/events/`, `internal/users/`, `internal/mobs/`.

**Spec:** `docs/superpowers/specs/2026-05-13-state-chunk-0-framework-and-combat-phase-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` (the umbrella branch for the side quest; aliveness 2.7 Task 19 remains pending).

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/machine.go` | NEW | `Machine[S comparable]` generic struct, transition logic |
| `internal/state/transition.go` | NEW | `TransitionReason`, `TransitionTable`, error types, `ActorRef` |
| `internal/state/handlers.go` | NEW | Veto/Cascade/Observer handler types |
| `internal/state/scheduled.go` | NEW | Scheduled-transition machinery (round-tick driven) |
| `internal/state/machine_test.go` | NEW | Framework unit tests |
| `internal/state/scheduled_test.go` | NEW | Scheduled-transition tests |
| `internal/state/context.md` | NEW | Framework package documentation |
| `internal/state/combatphase/combatphase.go` | NEW | Combat Phase state enum + data types |
| `internal/state/combatphase/transitions.go` | NEW | Valid-transition table for Combat Phase |
| `internal/state/combatphase/rules.go` | NEW | Veto and cascade handlers per matrix |
| `internal/state/combatphase/combatphase_test.go` | NEW | Behavior Matrix tests (CP-001 through CP-036) |
| `internal/characters/character.go` | MODIFY | Add `CombatPhase` field, `Combatant` bool field; remove `Aggro` field at sunset |
| `internal/characters/character.go` | MODIFY | Add predicate methods: `IsEngaged()`, `EngagedTarget()`, `Attackers()` |
| `internal/characters/aggro.go` | DELETE | Old Aggro struct + methods (at sunset) |
| `internal/characters/aggro_grace_test.go` | DELETE | Replaced by CP-015 + new grace test in combatphase_test.go |
| `internal/hooks/NewRound_DoCombat.go` | MODIFY | Round driver consults Combat Phase for tick dispatch |
| `internal/hooks/NewRound_DoCombat_unified.go` | DELETE | Stage 2a graveyard (replaced by Combat Phase) |
| `internal/hooks/aggro_helpers.go` | DELETE | ValidateAggro/RetargetOrEnd/CompanionAutoTarget (replaced by framework + cascades) |
| `internal/usercommands/attack.go` | MODIFY | `SetAggro` → `combatPhase.TransitionTo` |
| `internal/usercommands/flee.go` | MODIFY | Use Disengaging transition; remove Aggro{Type:Flee} sentinel |
| `internal/usercommands/go.go` | MODIFY | `Aggro != nil` blocking → `IsEngaged()` check; movement triggers Disengaging |
| `internal/usercommands/suicide.go` | MODIFY | Death cleanup via Combat Phase Idle cascade |
| `internal/usercommands/bash.go` | MODIFY | `Aggro != nil` check → `IsEngaged()` |
| `internal/usercommands/kick.go` | MODIFY | `Aggro != nil` → `IsEngaged()` |
| `internal/usercommands/grapple.go` | MODIFY | `Aggro != nil` → `IsEngaged()` |
| `internal/usercommands/taunt.go` | MODIFY | `Aggro != nil` → `IsEngaged()` |
| `internal/usercommands/trip.go` | MODIFY | `Aggro != nil` → `IsEngaged()` |
| `internal/usercommands/submit.go` | MODIFY | `Aggro != nil` → `IsEngaged()` |
| `internal/usercommands/mutation_*.go` | MODIFY | All `Aggro != nil` checks → `IsEngaged()` |
| `internal/mobcommands/attack.go` | MODIFY | `SetAggro` → `combatPhase.TransitionTo` |
| `internal/mobcommands/flee.go` | MODIFY | `EndAggro` → Disengaging transition |
| `internal/behaviortree/actions_combat.go` | MODIFY | `actAttack` uses TransitionTo; `actTargetWeakestMobInRoom` uses TransitionTo; `actFlee` uses Disengaging; `actTargetRandomPlayerInRoom` — **see Task 13 for the chunk-2.7 specific fix** |
| `internal/behaviortree/conditions_combat.go` | MODIFY | `resolveTargetPower` reads Combat Phase target |
| `internal/behaviortree/conditions_skullduggery.go` | MODIFY | `resolveSkullduggeryTarget` reads Combat Phase target (or context-passed target) |
| `internal/behaviortree/actions_skullduggery.go` | MODIFY | `try_steal`/`try_plant`/`try_shadow` pass target via action params, not via aggro |
| `internal/hooks/NewRound_AutoHeal.go` | MODIFY | `inCombat := user.Character.Aggro != nil` → `IsEngaged()` |
| `internal/hooks/MobDeath_FactionRep.go` | MODIFY | Subscribe to Combat Phase observer instead of reading Aggro |
| `internal/hooks/aggro_helpers_companion.go` | NEW (replaces deleted aggro_helpers.go portion) | Companion auto-assist via Attackers() subscription |
| `internal/mobs/mobs.go` | MODIFY | `Hostile` field → `Combatant` query (with backward-compat YAML read) |
| `internal/combat/combat.go` | MODIFY | Grapple transition aggro fixes → Combat Phase transitions |
| `internal/characters/charminfo.go` | MODIFY | Charm clears aggro → Combat Phase Idle |
| `_datafiles/world/dogmud/mobs/**` | MODIFY | YAML `hostile:` field → `combatant:` (with engine accepting both for one chunk's grace window before YAML migration) |
| `internal/characters/context.md` | MODIFY | Document Combat Phase + Combatant + sunset of Aggro |
| `internal/behaviortree/context.md` | MODIFY | Document new transition events |
| `internal/hooks/context.md` | MODIFY | Document round driver dispatch via Combat Phase |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Note chunk 2.7 unblocked; aliveness still paused until chunks 1-5 complete |
| `COMBAT_STATE_ROADMAP.md` | NEW | Master roadmap tracker (analogous to MOB_ALIVENESS_ROADMAP.md) |

---

## Task 1: Framework — types and machine struct

**Files:**
- Create: `internal/state/transition.go`
- Create: `internal/state/handlers.go`
- Create: `internal/state/machine.go`
- Create: `internal/state/machine_test.go`

Foundation. The generic `Machine[S]` type with `State()`, basic constructor, and the supporting types `TransitionReason`, `ActorRef`, error types, and handler signatures.

- [ ] **Step 1: Create `internal/state/transition.go`**

```go
package state

import "errors"

// ActorRef is a discriminated reference to either a user or mob.
// Zero value means "no actor."
type ActorRef struct {
	UserId        int
	MobInstanceId int
}

// IsZero returns true if the ActorRef points to nobody.
func (a ActorRef) IsZero() bool {
	return a.UserId == 0 && a.MobInstanceId == 0
}

// IsPlayer returns true if the ActorRef is a player target.
func (a ActorRef) IsPlayer() bool {
	return a.UserId > 0
}

// IsMob returns true if the ActorRef is a mob target.
func (a ActorRef) IsMob() bool {
	return a.MobInstanceId > 0
}

// TransitionReason captures why a state transition occurred.
// Used by veto, cascade, and observer handlers to decide
// whether to react and how. Propagated through the entire
// transition chain.
type TransitionReason struct {
	// Trigger is a stable string identifier ("attack_command",
	// "flee_success", "target_died", "surprise_attack", etc.)
	Trigger string

	// Actor is who initiated the transition (may be the same
	// character whose state is changing, or another character).
	Actor ActorRef

	// Target is an optional companion ActorRef (the combat
	// target for Engaging transitions, the killer for death
	// transitions, etc.).
	Target ActorRef

	// Metadata is open key-value storage for transition-specific
	// data (e.g., damage amount on a hurt transition).
	Metadata map[string]any
}

// TransitionTable maps "from" states to the set of valid
// "to" states. Used by Machine to reject framework-invariant
// violations before any veto handler runs.
type TransitionTable[S comparable] map[S][]S

// IsAllowed returns true if from→to is in the table.
func (t TransitionTable[S]) IsAllowed(from, to S) bool {
	for _, allowed := range t[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Framework error types.
var (
	ErrInvalidTransition = errors.New("state: transition not allowed by table")
	ErrVetoed            = errors.New("state: transition vetoed")
)

// VetoError wraps a veto reason with the vetoing handler's
// identifier for debugging.
type VetoError struct {
	HandlerName string
	Reason      string
}

func (e *VetoError) Error() string {
	return "state: vetoed by " + e.HandlerName + ": " + e.Reason
}

func (e *VetoError) Unwrap() error { return ErrVetoed }
```

- [ ] **Step 2: Create `internal/state/handlers.go`**

```go
package state

// VetoHandler may return a non-nil error to block a pending
// transition. The framework runs vetoes in registration order;
// the first non-nil return halts the chain and is returned to
// the TransitionTo caller. No state change occurs.
type VetoHandler[S comparable] func(from, to S, reason TransitionReason) error

// CascadeHandler runs after a successful state change in the
// same stack frame. May call TransitionTo on this or other
// machines. Fires in registration order; all cascade handlers
// run (none short-circuit).
type CascadeHandler[S comparable] func(from, to S, reason TransitionReason)

// ObserverHandler is a pure observer. Cannot veto, does not
// participate in cascade ordering. Used by quest engine,
// aliveness substrate, telemetry. Fires after all cascade
// handlers complete.
type ObserverHandler[S comparable] func(from, to S, reason TransitionReason)
```

- [ ] **Step 3: Create `internal/state/machine.go`**

```go
package state

import "sync"

// Machine is a generic finite state machine instance.
// Type parameter S is the state enum (typically an int-based
// type alias defined in the consumer package).
//
// Machines are intended to live as fields on Character; one
// Machine per concern (CombatPhase, Activity, Position,
// Awareness, Life, Presence).
//
// Machines are NOT safe for concurrent use by themselves —
// the engine's per-character lock should guard transitions
// (consistent with existing Character mutation patterns).
type Machine[S comparable] struct {
	current S
	table   TransitionTable[S]

	vetoes    []vetoEntry[S]
	cascades  []cascadeEntry[S]
	observers []observerEntry[S]

	mu sync.Mutex // serializes transitions within a single character
}

type vetoEntry[S comparable] struct {
	name    string
	handler VetoHandler[S]
}

type cascadeEntry[S comparable] struct {
	name    string
	handler CascadeHandler[S]
}

type observerEntry[S comparable] struct {
	name    string
	handler ObserverHandler[S]
}

// NewMachine constructs a Machine with the given initial state
// and valid-transitions table.
func NewMachine[S comparable](initial S, table TransitionTable[S]) *Machine[S] {
	return &Machine[S]{current: initial, table: table}
}

// State returns the current state.
func (m *Machine[S]) State() S {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// BeforeTransition registers a veto handler. name is used
// for debugging — the returned VetoError carries it.
func (m *Machine[S]) BeforeTransition(name string, handler VetoHandler[S]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vetoes = append(m.vetoes, vetoEntry[S]{name: name, handler: handler})
}

// AfterTransition registers a cascade handler.
func (m *Machine[S]) AfterTransition(name string, handler CascadeHandler[S]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cascades = append(m.cascades, cascadeEntry[S]{name: name, handler: handler})
}

// Subscribe registers an observer.
func (m *Machine[S]) Subscribe(name string, handler ObserverHandler[S]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers = append(m.observers, observerEntry[S]{name: name, handler: handler})
}

// CanTransition returns nil if a transition from current state
// to `to` would be allowed: the transition table accepts it and
// no veto handler returns an error. Does not mutate.
func (m *Machine[S]) CanTransition(to S, reason TransitionReason) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.canTransitionLocked(to, reason)
}

func (m *Machine[S]) canTransitionLocked(to S, reason TransitionReason) error {
	if !m.table.IsAllowed(m.current, to) {
		return ErrInvalidTransition
	}
	for _, v := range m.vetoes {
		if err := v.handler(m.current, to, reason); err != nil {
			return &VetoError{HandlerName: v.name, Reason: err.Error()}
		}
	}
	return nil
}

// TransitionTo attempts the transition. On success:
//  1. State changes.
//  2. All cascade handlers fire in registration order.
//  3. All observer handlers fire in registration order.
//
// On veto: returns the veto error; no state change, no cascades.
// On invalid transition (not in table): returns ErrInvalidTransition.
func (m *Machine[S]) TransitionTo(to S, reason TransitionReason) error {
	m.mu.Lock()

	if err := m.canTransitionLocked(to, reason); err != nil {
		m.mu.Unlock()
		return err
	}

	from := m.current
	m.current = to

	// Snapshot handlers under lock, then run without it.
	// Cascades may call TransitionTo on this or other machines;
	// holding the lock would deadlock.
	cascades := append([]cascadeEntry[S]{}, m.cascades...)
	observers := append([]observerEntry[S]{}, m.observers...)
	m.mu.Unlock()

	for _, c := range cascades {
		c.handler(from, to, reason)
	}
	for _, o := range observers {
		o.handler(from, to, reason)
	}
	return nil
}
```

- [ ] **Step 4: Create `internal/state/machine_test.go` with basic tests**

```go
package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type testState int

const (
	stateIdle testState = iota
	stateRunning
	stateDone
)

func makeTestMachine() *Machine[testState] {
	return NewMachine(stateIdle, TransitionTable[testState]{
		stateIdle:    {stateRunning},
		stateRunning: {stateDone, stateIdle},
	})
}

func TestMachine_InitialState(t *testing.T) {
	m := makeTestMachine()
	require.Equal(t, stateIdle, m.State())
}

func TestMachine_ValidTransition(t *testing.T) {
	m := makeTestMachine()
	err := m.TransitionTo(stateRunning, TransitionReason{Trigger: "test"})
	require.NoError(t, err)
	require.Equal(t, stateRunning, m.State())
}

func TestMachine_InvalidTransition(t *testing.T) {
	m := makeTestMachine()
	// stateIdle → stateDone is not in the table.
	err := m.TransitionTo(stateDone, TransitionReason{Trigger: "test"})
	require.ErrorIs(t, err, ErrInvalidTransition)
	require.Equal(t, stateIdle, m.State(), "state unchanged on invalid transition")
}

func TestMachine_VetoBlocks(t *testing.T) {
	m := makeTestMachine()
	m.BeforeTransition("blocker", func(from, to testState, _ TransitionReason) error {
		return errors.New("no")
	})
	err := m.TransitionTo(stateRunning, TransitionReason{Trigger: "test"})
	require.ErrorIs(t, err, ErrVetoed)
	require.Equal(t, stateIdle, m.State(), "state unchanged when vetoed")
}

func TestMachine_CascadeFires(t *testing.T) {
	m := makeTestMachine()
	var fired bool
	m.AfterTransition("cascade", func(from, to testState, _ TransitionReason) {
		require.Equal(t, stateIdle, from)
		require.Equal(t, stateRunning, to)
		fired = true
	})
	require.NoError(t, m.TransitionTo(stateRunning, TransitionReason{}))
	require.True(t, fired, "cascade should fire after successful transition")
}

func TestMachine_CascadeRunsAllInOrder(t *testing.T) {
	m := makeTestMachine()
	var order []string
	m.AfterTransition("first", func(_, _ testState, _ TransitionReason) {
		order = append(order, "first")
	})
	m.AfterTransition("second", func(_, _ testState, _ TransitionReason) {
		order = append(order, "second")
	})
	require.NoError(t, m.TransitionTo(stateRunning, TransitionReason{}))
	require.Equal(t, []string{"first", "second"}, order)
}

func TestMachine_ObserverFiresAfterCascade(t *testing.T) {
	m := makeTestMachine()
	var order []string
	m.AfterTransition("cascade", func(_, _ testState, _ TransitionReason) {
		order = append(order, "cascade")
	})
	m.Subscribe("observer", func(_, _ testState, _ TransitionReason) {
		order = append(order, "observer")
	})
	require.NoError(t, m.TransitionTo(stateRunning, TransitionReason{}))
	require.Equal(t, []string{"cascade", "observer"}, order)
}

func TestMachine_CanTransitionDoesNotMutate(t *testing.T) {
	m := makeTestMachine()
	require.NoError(t, m.CanTransition(stateRunning, TransitionReason{}))
	require.Equal(t, stateIdle, m.State())
}
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/state/ -v -run "TestMachine_"
```
Expected: all PASS (we implemented the framework with the tests).

- [ ] **Step 6: Build verification**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/state/
git commit -m "$(cat <<'EOF'
feat(state): generic Machine[S] framework with vetoes/cascades/observers

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Framework — scheduled transitions

**Files:**
- Create: `internal/state/scheduled.go`
- Create: `internal/state/scheduled_test.go`

Round-tick-driven scheduled transitions. Used for Combat Phase's `Engaging → Engaged` auto-advance after RoundsWaiting, end-of-round Awareness Hidden→Revealing (future chunk 1), Charm rebellion checks (later), etc.

This task adds a `Scheduler` interface plus a `RoundScheduler` implementation that fires scheduled transitions at round-tick boundaries.

- [ ] **Step 1: Write failing tests in `internal/state/scheduled_test.go`**

```go
package state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduler_TransitionFiresOnRoundTick(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()

	// Schedule transition to Running after 2 rounds.
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(2),
		TransitionReason{Trigger: "scheduled_test"})

	require.Equal(t, stateIdle, m.State(), "should not fire before round elapses")

	sched.Tick()
	require.Equal(t, stateIdle, m.State(), "still 1 round remaining")

	sched.Tick()
	require.Equal(t, stateRunning, m.State(), "should fire at round 2")
}

func TestScheduler_TransitionFiresImmediatelyAtZero(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(0),
		TransitionReason{Trigger: "immediate"})
	sched.Tick()
	require.Equal(t, stateRunning, m.State())
}

func TestScheduler_CancelScheduled(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(2),
		TransitionReason{Trigger: "to_cancel"})
	m.CancelScheduled()
	sched.Tick()
	sched.Tick()
	require.Equal(t, stateIdle, m.State(), "canceled transition should not fire")
}

func TestScheduler_VetoedScheduledTransitionLeavesStateAlone(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()
	m.BeforeTransition("blocker", func(from, to testState, r TransitionReason) error {
		if r.Trigger == "scheduled_vetoed" {
			return errVetoTest
		}
		return nil
	})
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(1),
		TransitionReason{Trigger: "scheduled_vetoed"})
	sched.Tick()
	require.Equal(t, stateIdle, m.State(), "veto blocks scheduled transition")
}

var errVetoTest = errors.New("test veto")
```

(Add `"errors"` to imports.)

- [ ] **Step 2: Run tests, expect FAIL**

```bash
go test ./internal/state/ -v -run "TestScheduler_"
```
Expected: FAIL (Scheduler/ScheduleTransition/CancelScheduled don't exist).

- [ ] **Step 3: Implement `internal/state/scheduled.go`**

```go
package state

// ScheduleAt is an opaque scheduled-time handle. RoundScheduler
// produces these; Machine consumes them.
type ScheduleAt struct {
	scheduler *RoundScheduler
	round     int
}

// RoundScheduler fires scheduled transitions when its round
// counter advances past their target rounds.
//
// The engine wires the master round tick to RoundScheduler.Tick().
// Per-character RoundSchedulers do not exist — there's one
// global scheduler that all machines register against.
type RoundScheduler struct {
	currentRound int
	pending      []scheduledTransition
}

type scheduledTransition struct {
	round     int
	fire      func()
	cancelled *bool // pointer because copies of the entry share cancel state
}

// NewRoundScheduler returns a fresh scheduler at round 0.
func NewRoundScheduler() *RoundScheduler {
	return &RoundScheduler{}
}

// RoundsFromNow returns a ScheduleAt that fires after n round
// ticks (where 0 means "the next tick fires it").
func (s *RoundScheduler) RoundsFromNow(n int) ScheduleAt {
	return ScheduleAt{scheduler: s, round: s.currentRound + n}
}

// Tick advances the scheduler by one round and fires any
// scheduled transitions whose target round <= currentRound.
func (s *RoundScheduler) Tick() {
	s.currentRound++
	keep := s.pending[:0]
	for _, p := range s.pending {
		if *p.cancelled {
			continue // dropped
		}
		if p.round <= s.currentRound {
			p.fire()
			continue
		}
		keep = append(keep, p)
	}
	s.pending = keep
}

// ScheduleTransition registers a deferred TransitionTo.
// When the scheduler's round counter reaches at.round, the
// transition fires. Reuse of ScheduleTransition without
// CancelScheduled adds a second pending transition — callers
// who want at-most-one-pending must call CancelScheduled first.
//
// On the configured firing round, the framework calls
// TransitionTo with the supplied target and reason. If the
// transition is vetoed at fire time, the state remains
// unchanged; no error is returned (no caller to receive it).
// Cascades and observers fire normally on success.
func (m *Machine[S]) ScheduleTransition(to S, at ScheduleAt, reason TransitionReason) {
	if at.scheduler == nil {
		return
	}
	cancelled := false
	m.mu.Lock()
	m.cancelTokens = append(m.cancelTokens, &cancelled)
	m.mu.Unlock()
	at.scheduler.pending = append(at.scheduler.pending, scheduledTransition{
		round:     at.round,
		cancelled: &cancelled,
		fire: func() {
			_ = m.TransitionTo(to, reason)
		},
	})
}

// CancelScheduled marks all pending scheduled transitions on
// this machine as cancelled. Tick() will skip them.
func (m *Machine[S]) CancelScheduled() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.cancelTokens {
		*c = true
	}
	m.cancelTokens = nil
}
```

Add `cancelTokens []*bool` field to `Machine[S]` struct in `machine.go`:

```go
type Machine[S comparable] struct {
	// ... existing fields ...
	cancelTokens []*bool
}
```

- [ ] **Step 4: Run tests, expect PASS**

```bash
go test ./internal/state/ -v -run "TestScheduler_"
```

- [ ] **Step 5: Commit**

```bash
git add internal/state/
git commit -m "$(cat <<'EOF'
feat(state): round-tick scheduled transitions

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Combat Phase — state types and machine bootstrap

**Files:**
- Create: `internal/state/combatphase/combatphase.go`
- Create: `internal/state/combatphase/transitions.go`
- Modify: `internal/characters/character.go` (add `CombatPhase` field alongside existing `Aggro`)

The Combat Phase state enum, per-state data types, transition table, and a `NewCombatPhase()` constructor. Character gets the new field but old Aggro stays until later tasks.

- [ ] **Step 1: Create `internal/state/combatphase/combatphase.go`**

```go
// Package combatphase defines the Combat Phase state machine,
// the first consumer of internal/state. It replaces the
// Character.Aggro field as the source of truth for "who am I
// attacking?" and "am I in combat?".
package combatphase

import (
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
	NextSwingAt  int // round number for next swing
	SurpriseLeft bool // true during the first round of a SurpriseAttack engagement
}

// DisengagingData is the state-data type for the Disengaging state.
type DisengagingData struct {
	LastTarget state.ActorRef // target at time of flee
	FleeRound  int            // round flee was initiated
}

// Machine wraps state.Machine[State] with Combat-Phase-specific
// API including per-state data storage and Attackers tracking.
type Machine struct {
	inner     *state.Machine[State]
	engaging  *EngagingData
	engaged   *EngagedData
	disengaging *DisengagingData
	attackers []state.ActorRef // inbound attacker list
}

// NewMachine returns a Combat Phase machine in Idle.
func NewMachine() *Machine {
	m := &Machine{
		inner: state.NewMachine(Idle, validTransitions),
	}
	return m
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

// DisengagingData returns the Disengaging state's data if currently Disengaging.
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
// Renamed in code reviews if a clearer term emerges.
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

// Inner returns the underlying state.Machine — used by
// rules.go to register vetoes/cascades. Not part of the
// stable API; do not depend on it from outside this package.
func (m *Machine) Inner() *state.Machine[State] {
	return m.inner
}
```

- [ ] **Step 2: Create `internal/state/combatphase/transitions.go`**

```go
package combatphase

import "github.com/GoMudEngine/GoMud/internal/state"

// validTransitions enforces the Combat Phase invariant matrix.
// Vetoes layer additional rules on top.
var validTransitions = state.TransitionTable[State]{
	Idle:        {Engaging},
	Engaging:    {Engaged, Idle},          // Idle on cancel/target-died
	Engaged:     {Disengaging, Idle},      // Idle direct on death/despawn
	Disengaging: {Idle, Engaged},          // Engaged on flee failure
}

// Trigger reason constants. Use these in TransitionReason.Trigger
// to ensure stable strings across the codebase.
const (
	TriggerAttackCommand    = "attack_command"
	TriggerSurpriseAttack   = "surprise_attack"
	TriggerEngagementReady  = "engagement_ready"
	TriggerFleeCommand      = "flee_command"
	TriggerFleeSuccess      = "flee_success"
	TriggerFleeFailure      = "flee_failure"
	TriggerTargetDied       = "target_died"
	TriggerSelfDied         = "self_died"
	TriggerTargetOutOfRoom  = "target_out_of_room"
	TriggerCharm            = "charm_acquired"
	TriggerCombatantToggle  = "combatant_toggle"
	TriggerForceIdle        = "force_idle"
)
```

- [ ] **Step 3: Add Combat Phase field to Character**

Open `internal/characters/character.go`. Find the struct definition. Add the field near the existing `Aggro` field:

```go
// Existing field (KEEP for now — sunset in Task 16):
Aggro *Aggro `yaml:"-"`

// NEW (Task 3): Combat Phase state machine.
CombatPhase *combatphase.Machine `yaml:"-"`
```

Import the package at top of file:

```go
import (
    // ... existing imports ...
    "github.com/GoMudEngine/GoMud/internal/state/combatphase"
)
```

Find the Character constructor (`New()` or similar — grep for `func New(` in characters package). Initialize the CombatPhase field:

```go
func New() Character {
    return Character{
        // ... existing init ...
        CombatPhase: combatphase.NewMachine(),
    }
}
```

If there are multiple Character constructors / init paths, ensure all initialize CombatPhase. Grep for `Character{` in `internal/characters/` to find them.

- [ ] **Step 4: Build verification**

```bash
go build ./...
```
Expected: clean (no callers using CombatPhase yet; just adding the field is non-breaking).

- [ ] **Step 5: Commit**

```bash
git add internal/state/combatphase/ internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat(combatphase): state types, transition table, Character field

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Combat Phase — Behavior Matrix tests (RED phase)

**Files:**
- Create: `internal/state/combatphase/combatphase_test.go`

Author the test suite that encodes the Behavior Matrix. Tests are all failing initially — `TransitionTo` returns ErrInvalidTransition or panics because no rules are wired yet. This is the RED phase: tests as design documentation.

This is a large task — ~35 tests. We write them all in one task so the entire matrix is encoded before any implementation begins. The implementation tasks (5-15) will turn these tests green incrementally.

- [ ] **Step 1: Create `internal/state/combatphase/combatphase_test.go` with all matrix tests**

```go
package combatphase

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/require"
)

// Test helpers. Each test constructs Machine instances directly
// (no Character — those come in integration tests in later tasks).

func makePair() (attacker, defender *Machine) {
	return NewMachine(), NewMachine()
}

func actor(userId int) state.ActorRef {
	return state.ActorRef{UserId: userId}
}

func attackReason(attacker, target state.ActorRef) state.TransitionReason {
	return state.TransitionReason{
		Trigger: TriggerAttackCommand,
		Actor:   attacker,
		Target:  target,
	}
}

// --- Standard entry / exit (CP-001 through CP-008) ---

// CP-001: Basic attack initiation. Attacker → Engaging; defender
// gains attacker in its Attackers list; defender's Combat Phase
// unchanged (defenders don't auto-engage).
func TestCP_001_BasicAttackInitiation(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)

	err := A.TransitionToEngaging(EngagingData{Target: b, RoundsUntil: 0},
		attackReason(a, b))
	require.NoError(t, err)

	require.Equal(t, Engaging, A.State())
	require.Equal(t, Idle, B.State(), "defender does not auto-engage")
	require.Len(t, B.Attackers(), 1, "defender's Attackers list gains attacker")
	require.Equal(t, a, B.Attackers()[0])
}

// CP-002: Engaging counts down each round until ready.
func TestCP_002_EngagingCountdown(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(
		EngagingData{Target: actor(2), RoundsUntil: 2}, state.TransitionReason{}))

	A.OnRoundTick()
	d, _ := A.EngagingData()
	require.Equal(t, 1, d.RoundsUntil, "decrement after one round")

	A.OnRoundTick()
	require.Equal(t, Engaged, A.State(), "transition to Engaged when RoundsUntil reaches 0")
}

// CP-003: Engaging → Engaged fires mob_engaged event (verified via cascade).
func TestCP_003_EngagingToEngagedFiresCascade(t *testing.T) {
	A, _ := makePair()
	var firedTransitions []State
	A.Inner().AfterTransition("test_cascade", func(from, to State, _ state.TransitionReason) {
		firedTransitions = append(firedTransitions, to)
	})
	require.NoError(t, A.TransitionToEngaging(
		EngagingData{Target: actor(2), RoundsUntil: 0}, state.TransitionReason{}))
	A.OnRoundTick()
	require.Contains(t, firedTransitions, Engaged)
}

// CP-004: Target dies → attacker auto-transitions Engaged → Idle.
func TestCP_004_TargetDiesEndsEngagement(t *testing.T) {
	A, B := makePair()
	require.NoError(t, A.TransitionToEngaging(
		EngagingData{Target: actor(2), RoundsUntil: 0}, state.TransitionReason{}))
	A.OnRoundTick() // Engaging → Engaged

	// Simulate B dying — for the test, call A's NotifyTargetDied().
	A.NotifyTargetDied(actor(2))

	require.Equal(t, Idle, A.State(), "attacker returns to Idle when target dies")
	require.Empty(t, B.Attackers(), "B's Attackers cleared")
}

// CP-005: Self dies → outbound combat ends + inbound attackers cleared.
func TestCP_005_SelfDeathClearsBoth(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: b}, attackReason(a, b)))
	require.NoError(t, B.TransitionToEngaging(EngagingData{Target: a}, attackReason(b, a)))

	A.NotifySelfDied()
	require.Equal(t, Idle, A.State(), "self died → Idle")
	require.Empty(t, A.Attackers(), "inbound attackers cleared")
	require.Empty(t, B.Attackers(), "outbound cleared from B's perspective")
}

// CP-006: Flee command initiates Disengaging.
func TestCP_006_FleeInitiatesDisengaging(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // Engaging → Engaged
	require.NoError(t, A.TransitionToDisengaging(
		state.TransitionReason{Trigger: TriggerFleeCommand}))
	require.Equal(t, Disengaging, A.State())
}

// CP-007: Flee success → Idle.
func TestCP_007_FleeSuccess(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: b}, attackReason(a, b)))
	A.OnRoundTick()
	require.NoError(t, A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand}))
	A.ResolveFlee(true) // success
	require.Equal(t, Idle, A.State())
	require.Empty(t, B.Attackers())
}

// CP-008: Flee failure → back to Engaged.
func TestCP_008_FleeFailureReturnsEngaged(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick()
	require.NoError(t, A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand}))
	A.ResolveFlee(false) // failure
	require.Equal(t, Engaged, A.State())
}

// --- Surprise attack semantics (CP-009, CP-023 through CP-025b) ---

// CP-009/CP-023: Hidden attacker enters Engaging with surprise marker; Awareness stays Hidden.
func TestCP_023_SurpriseEngagingPreservesStealth(t *testing.T) {
	A, _ := makePair()
	reason := state.TransitionReason{
		Trigger: TriggerSurpriseAttack,
		Actor:   actor(1),
		Target:  actor(2),
	}
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), Reason: reason}, reason))

	d, _ := A.EngagingData()
	require.Equal(t, TriggerSurpriseAttack, d.Reason.Trigger,
		"reason persisted on Engaging data for surprise carry-through")
}

// CP-024: Engaging → Engaged with surprise marker preserves stealth (no Awareness cascade fires here).
func TestCP_024_SurpriseEngagedStillPreservesStealth(t *testing.T) {
	A, _ := makePair()
	reason := state.TransitionReason{Trigger: TriggerSurpriseAttack}
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), Reason: reason}, reason))
	A.OnRoundTick()
	d, _ := A.EngagedData()
	require.True(t, d.SurpriseLeft, "Engaged data flags surprise as still available")
}

// CP-025: At end of first combat round, surprise consumed; cascade for stealth-break can fire.
func TestCP_025_SurpriseConsumedAtEndOfFirstRound(t *testing.T) {
	A, _ := makePair()
	reason := state.TransitionReason{Trigger: TriggerSurpriseAttack}
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), Reason: reason}, reason))
	A.OnRoundTick() // Engaging → Engaged

	var stealthBreakFired bool
	A.OnEndOfRoundIfSurprise(func(_ state.TransitionReason) {
		stealthBreakFired = true
	})

	A.OnCombatRoundEnd() // round resolves
	require.True(t, stealthBreakFired, "end-of-round cascade fires once surprise round ends")

	d, _ := A.EngagedData()
	require.False(t, d.SurpriseLeft, "surprise flag consumed after first round end")
}

// --- Vetoes (CP-010 through CP-016) ---

// CP-010: NonCombatant cannot attack.
func TestCP_010_NonCombatantCannotInitiate(t *testing.T) {
	A, _ := makePair()
	A.RegisterCombatantVeto(func() bool { return false /* NonCombatant */ })
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)}, state.TransitionReason{})
	require.ErrorIs(t, err, state.ErrVetoed)
	require.Equal(t, Idle, A.State())
}

// CP-011: NonCombatant target cannot be attacked.
func TestCP_011_NonCombatantTargetVetoed(t *testing.T) {
	A, _ := makePair()
	A.RegisterTargetCombatantCheck(func(target state.ActorRef) bool {
		return false // target is NonCombatant
	})
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{Target: actor(2)})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-012: Casting/Crafting/Foraging blocks Engaging (Activity != Free).
func TestCP_012_ActivityBlocksEngaging(t *testing.T) {
	A, _ := makePair()
	A.RegisterActivityCheck(func() bool { return false /* not Free */ })
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)}, state.TransitionReason{})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-013: Dead character cannot attack.
func TestCP_013_DeadCannotAttack(t *testing.T) {
	A, _ := makePair()
	A.RegisterLifeCheck(func() bool { return false /* not Alive */ })
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)}, state.TransitionReason{})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-014: Cannot attack a dead target.
func TestCP_014_CannotAttackDeadTarget(t *testing.T) {
	A, _ := makePair()
	A.RegisterTargetLifeCheck(func(_ state.ActorRef) bool {
		return false // target dead
	})
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{Target: actor(2)})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-015: AFK/Disconnected player target vetoed.
func TestCP_015_AFKTargetVetoed(t *testing.T) {
	A, _ := makePair()
	A.RegisterTargetPresenceCheck(func(_ state.ActorRef) bool {
		return false // target AFK/Disconnected
	})
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{Target: actor(2)})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-016: Cannot flee while grappled.
func TestCP_016_GrappledCannotFlee(t *testing.T) {
	A, _ := makePair()
	A.RegisterPositionCheck(func() bool { return false /* Clinched/Grounded */ })
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick()
	err := A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// --- Multi-attacker (CP-017 through CP-019) ---

// CP-017: Three attackers track inbound on victim.
func TestCP_017_MultipleAttackersTracked(t *testing.T) {
	M1, M2, M3, P := NewMachine(), NewMachine(), NewMachine(), NewMachine()
	p := actor(1)
	for i, m := range []*Machine{M1, M2, M3} {
		require.NoError(t, m.TransitionToEngaging(EngagingData{Target: p},
			state.TransitionReason{Actor: actor(100 + i), Target: p}))
	}
	require.Equal(t, Idle, P.State(), "defender stays Idle")
	require.Len(t, P.Attackers(), 3)
}

// CP-018: When attacker dies, inbound list shrinks.
func TestCP_018_DeadAttackerRemovedFromInbound(t *testing.T) {
	M1, M2 := NewMachine(), NewMachine()
	P := NewMachine()
	p := actor(1)
	require.NoError(t, M1.TransitionToEngaging(EngagingData{Target: p},
		state.TransitionReason{Actor: actor(101), Target: p}))
	require.NoError(t, M2.TransitionToEngaging(EngagingData{Target: p},
		state.TransitionReason{Actor: actor(102), Target: p}))

	require.Len(t, P.Attackers(), 2)
	M1.NotifySelfDied()
	require.Len(t, P.Attackers(), 1)
	require.Equal(t, actor(102), P.Attackers()[0])
}

// CP-019: Combat Phase exposes Attackers as observable change.
func TestCP_019_AttackersChangeIsObservable(t *testing.T) {
	M1 := NewMachine()
	P := NewMachine()
	var observedAttackerCount int
	P.SubscribeAttackersChange(func(_ []state.ActorRef) {
		observedAttackerCount++
	})
	require.NoError(t, M1.TransitionToEngaging(EngagingData{Target: actor(1)},
		state.TransitionReason{Actor: actor(101), Target: actor(1)}))
	require.Equal(t, 1, observedAttackerCount, "Attackers-change observer fires on inbound add")
}

// --- Non-combat target picking (CP-026, CP-027) ---

// CP-026: Soft-target (skullduggery) does NOT transition Combat Phase.
// This is the chunk-2.7 regression test.
func TestCP_026_SoftTargetDoesNotTransition(t *testing.T) {
	// Thief picks player as steal target. Stealing is called
	// directly via actions.Steal(actor, opts{TargetUserId: ...}).
	// No Combat Phase transition occurs.
	T, _ := makePair()
	require.Equal(t, Idle, T.State(), "thief stays Idle when picking a soft target")
	// (We don't even call any TransitionTo here — that's the point.
	// Compare to CP-001 where attack DOES transition.)
}

// CP-027: A failed steal does NOT auto-transition the mob to combat.
// The implementer chooses whether to escalate; framework doesn't force it.
func TestCP_027_FailedStealNoAutoCombat(t *testing.T) {
	T, _ := makePair()
	// Hypothetical: T failed a steal. Some old code paths might
	// have flipped aggro here. New design: only an explicit
	// TransitionToEngaging puts you in combat.
	require.Equal(t, Idle, T.State(),
		"failed non-combat action does not silently engage combat")
}

// --- Life cascade (CP-028, CP-029) ---

// CP-028: Death cascades to Idle outbound + clears inbound list.
func TestCP_028_DeathCascadeClearsAll(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: b}, attackReason(a, b)))
	require.NoError(t, B.TransitionToEngaging(EngagingData{Target: a}, attackReason(b, a)))

	A.NotifySelfDied()
	require.Equal(t, Idle, A.State())
	require.Empty(t, A.Attackers())
	require.Empty(t, B.Attackers())
}

// CP-029: Inbound-only death still cleans both sides.
func TestCP_029_InboundOnlyDeathCleansAttackers(t *testing.T) {
	A, B := makePair() // A is defender, never attacked back
	require.NoError(t, B.TransitionToEngaging(EngagingData{Target: actor(1)},
		state.TransitionReason{Actor: actor(2), Target: actor(1)}))
	require.Len(t, A.Attackers(), 1)

	A.NotifySelfDied()
	require.Empty(t, A.Attackers())
	// B's outbound also clears (target gone).
}

// --- Persistence (CP-030, CP-031) ---

// CP-030: Fresh Machine instance is in Idle.
func TestCP_030_FreshMachineIsIdle(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Idle, m.State())
	require.Empty(t, m.Attackers())
}

// CP-031: Scheduled transitions do not survive instance recreation.
func TestCP_031_ScheduledTransitionsDoNotPersist(t *testing.T) {
	// Implicit in CP-030 — the constructor returns a fresh machine
	// with no pending scheduled transitions. Documented for clarity.
	m := NewMachine()
	require.Equal(t, Idle, m.State())
}

// --- Combatant flag (CP-032) ---

// CP-032: Combatant flag toggle forces Idle.
func TestCP_032_CombatantOffForcesIdle(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // → Engaged
	require.Equal(t, Engaged, A.State())

	A.ForceIdle(state.TransitionReason{Trigger: TriggerCombatantToggle})
	require.Equal(t, Idle, A.State())
}

// --- Tick events (CP-033 through CP-036) ---

// CP-033: Engaged state dispatches mob_combat_round.
func TestCP_033_EngagedDispatchesCombatRoundEvent(t *testing.T) {
	A, _ := makePair()
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // → Engaged
	A.DispatchTickEvent()
	require.Equal(t, "mob_combat_round", event)
}

// CP-034: Idle state dispatches mob_idle.
func TestCP_034_IdleDispatchesIdleEvent(t *testing.T) {
	A, _ := makePair()
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	A.DispatchTickEvent() // currently Idle
	require.Equal(t, "mob_idle", event)
}

// CP-035: Engaging state dispatches no tick event.
func TestCP_035_EngagingDispatchesNoTickEvent(t *testing.T) {
	A, _ := makePair()
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), RoundsUntil: 5},
		state.TransitionReason{}))
	A.DispatchTickEvent()
	require.Empty(t, event, "Engaging is silent")
}

// CP-036: Disengaging state dispatches no tick event.
func TestCP_036_DisengagingDispatchesNoTickEvent(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // → Engaged
	require.NoError(t, A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand}))
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	A.DispatchTickEvent()
	require.Empty(t, event, "Disengaging is silent")
}
```

- [ ] **Step 2: Run tests — expect compile errors (many helper methods don't exist yet)**

```bash
go test ./internal/state/combatphase/ -v
```
Expected: compile errors for missing methods like `TransitionToEngaging`, `OnRoundTick`, `NotifyTargetDied`, `RegisterCombatantVeto`, `ForceIdle`, etc. This is the RED phase.

- [ ] **Step 3: Add method stubs to `internal/state/combatphase/combatphase.go` so tests compile but fail**

Add stub implementations (returning errors / no-ops) for every method referenced in the tests. The full implementations land in tasks 5-9. Example stubs:

```go
func (m *Machine) TransitionToEngaging(d EngagingData, r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) TransitionToDisengaging(r state.TransitionReason) error {
	return errors.New("not implemented")
}
func (m *Machine) ResolveFlee(success bool)             { /* not implemented */ }
func (m *Machine) OnRoundTick()                         { /* not implemented */ }
func (m *Machine) NotifyTargetDied(target state.ActorRef) { /* not implemented */ }
func (m *Machine) NotifySelfDied()                      { /* not implemented */ }
func (m *Machine) ForceIdle(r state.TransitionReason)   { /* not implemented */ }
func (m *Machine) OnEndOfRoundIfSurprise(fn func(state.TransitionReason)) { /* not implemented */ }
func (m *Machine) OnCombatRoundEnd()                    { /* not implemented */ }
func (m *Machine) RegisterCombatantVeto(check func() bool) { /* not implemented */ }
func (m *Machine) RegisterActivityCheck(check func() bool) { /* not implemented */ }
func (m *Machine) RegisterLifeCheck(check func() bool)     { /* not implemented */ }
func (m *Machine) RegisterPositionCheck(check func() bool) { /* not implemented */ }
func (m *Machine) RegisterTargetCombatantCheck(check func(state.ActorRef) bool) { /* not implemented */ }
func (m *Machine) RegisterTargetLifeCheck(check func(state.ActorRef) bool)      { /* not implemented */ }
func (m *Machine) RegisterTargetPresenceCheck(check func(state.ActorRef) bool)  { /* not implemented */ }
func (m *Machine) SubscribeAttackersChange(fn func([]state.ActorRef)) { /* not implemented */ }
func (m *Machine) OnTickEvent(fn func(name string, r state.TransitionReason)) { /* not implemented */ }
func (m *Machine) DispatchTickEvent()                   { /* not implemented */ }
```

Import `"errors"`.

- [ ] **Step 4: Run tests, expect FAIL (not compile errors)**

```bash
go test ./internal/state/combatphase/ -v
```
Expected: all 30+ tests FAIL (not skip, not compile error). RED phase complete.

- [ ] **Step 5: Commit**

```bash
git add internal/state/combatphase/
git commit -m "$(cat <<'EOF'
test(combatphase): Behavior Matrix RED — all matrix rows as failing tests

CP-001 through CP-036 encoded. Method stubs make tests compile
but fail. Implementation lands in tasks 5-15.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Combat Phase — Basic transitions (CP-001 through CP-008)

**Files:**
- Modify: `internal/state/combatphase/combatphase.go`
- Create: `internal/state/combatphase/rules.go`

Implement `TransitionToEngaging`, `TransitionToDisengaging`, `OnRoundTick`, `ResolveFlee`, `NotifyTargetDied`, `NotifySelfDied` — enough to pass CP-001 through CP-008.

- [ ] **Step 1: Implement Engaging transition + countdown + Engaged advance**

In `combatphase.go`, replace the stub `TransitionToEngaging` and related methods. Wire up the `attackers` slice maintenance via a package-level registry (covered in detail; the full code is below).

Author the full implementation. After this step:

```go
func (m *Machine) TransitionToEngaging(d EngagingData, r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Engaging, r); err != nil {
		return err
	}
	m.engaging = &d
	registerInbound(d.Target, r.Actor)
	return nil
}

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
		// Round tick during Engaged is handled by combat round driver,
		// not internal — but we may update NextSwingAt here in later tasks.
	case Disengaging:
		// Flee resolution scheduled separately (see ResolveFlee).
	}
}

func (m *Machine) advanceToEngaged() {
	prevEngaging := m.engaging
	if err := m.inner.TransitionTo(Engaged, state.TransitionReason{
		Trigger: TriggerEngagementReady,
		Target:  prevEngaging.Target,
	}); err != nil {
		// Invalid transition shouldn't happen; if it does, swallow.
		return
	}
	m.engaged = &EngagedData{
		Target:       prevEngaging.Target,
		SurpriseLeft: prevEngaging.Reason.Trigger == TriggerSurpriseAttack,
	}
	m.engaging = nil
}
```

Plus the Disengaging transition, ResolveFlee, NotifyTargetDied, NotifySelfDied, and the inbound-attackers registry.

The inbound registry is a package-level map keyed by ActorRef. Implementation:

```go
// At package level in rules.go:
var (
	inboundMu    sync.Mutex
	inboundIndex = map[state.ActorRef][]inboundEntry{} // target → list of attackers
)

type inboundEntry struct {
	attacker state.ActorRef
	machine  *Machine
}

func registerInbound(target, attacker state.ActorRef) {
	if target.IsZero() || attacker.IsZero() {
		return
	}
	inboundMu.Lock()
	defer inboundMu.Unlock()
	// Look up target's Machine via a separate machineByActor map maintained
	// elsewhere — for now, this is a stub. Real linkage lands in Task 7 when
	// we wire Character.CombatPhase to actor identity.
	_ = target
	_ = attacker
}
```

(For testing purposes, the test helpers will bind the machines directly. Real Character integration in Task 7.)

For the test setup, the registry is initialized per test. The tests as written use the `Attackers()` query on the Machine directly. The Machine stores its own inbound list (`attackers` field on Machine), populated when other machines call into it.

Simplest approach: have Machine expose a `RecordInboundAttacker(a state.ActorRef)` and `RemoveInboundAttacker(a state.ActorRef)` method. The framework calls them on transitions via cascade. For unit tests, test helpers either:
- Construct two Machines and have one call `B.RecordInboundAttacker(a)` when `A.TransitionToEngaging(target=b)`
- Use a registry that links by ActorRef

For these tests to pass without real Character objects, the test helpers in Task 4 use direct method calls. The Machine's `Attackers()` returns its `attackers` field. So:

```go
func (m *Machine) RecordInboundAttacker(a state.ActorRef) {
	if a.IsZero() {
		return
	}
	for _, existing := range m.attackers {
		if existing == a {
			return // already present
		}
	}
	m.attackers = append(m.attackers, a)
	m.notifyAttackersChange()
}

func (m *Machine) RemoveInboundAttacker(a state.ActorRef) {
	for i, existing := range m.attackers {
		if existing == a {
			m.attackers = append(m.attackers[:i], m.attackers[i+1:]...)
			m.notifyAttackersChange()
			return
		}
	}
}

func (m *Machine) notifyAttackersChange() {
	for _, fn := range m.attackersChangeListeners {
		fn(m.Attackers())
	}
}
```

To wire this between two test Machines, the test helpers need to look up the target Machine by ActorRef. The unit tests in Task 4 use the `makePair` helper which returns the pair directly — but tests reference targets by `actor(2)` etc., not by Machine. We need a registry within the test or a different test pattern.

**Decision for Task 4 tests:** Adjust the test helpers to use a `MachineRegistry` that maps ActorRef → *Machine. Tests register their Machines via `RegisterMachine(actor(1), A)` etc. The framework's transition logic looks up target via the registry to call RecordInboundAttacker.

Add to `combatphase.go`:

```go
// Registry maps ActorRef to Machine instances for cross-machine
// notification (inbound tracking, target death, etc.).
var (
	registryMu sync.Mutex
	machineRegistry = map[state.ActorRef]*Machine{}
)

// RegisterMachine binds an ActorRef to its Machine for cross-character
// lookups. Real engine integration (Task 7) wires this from Character
// setup. Tests register directly.
func RegisterMachine(ref state.ActorRef, m *Machine) {
	registryMu.Lock()
	defer registryMu.Unlock()
	machineRegistry[ref] = m
}

// UnregisterMachine removes a binding (e.g., on player logout or mob
// despawn).
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

Test helpers in `combatphase_test.go` register pairs:

```go
func makePair() (attacker, defender *Machine) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()
	A := NewMachine()
	B := NewMachine()
	RegisterMachine(actor(1), A)
	RegisterMachine(actor(2), B)
	return A, B
}
```

Update `TransitionToEngaging` to use lookup:

```go
func (m *Machine) TransitionToEngaging(d EngagingData, r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Engaging, r); err != nil {
		return err
	}
	m.engaging = &d
	if target := lookupMachine(d.Target); target != nil {
		target.RecordInboundAttacker(r.Actor)
	}
	return nil
}
```

**Refine the Task 4 test helpers accordingly** — `makePair` already registers actor(1)=A and actor(2)=B, so the tests as written should pass once the implementation is in.

- [ ] **Step 2: Run CP-001 through CP-008 tests, expect PASS**

```bash
go test ./internal/state/combatphase/ -v -run "TestCP_00"
```
Expected: tests CP-001 through CP-008 pass. Veto and surprise tests still fail (not implemented yet).

- [ ] **Step 3: Commit**

```bash
git add internal/state/combatphase/
git commit -m "$(cat <<'EOF'
feat(combatphase): basic transitions (CP-001 through CP-008)

Engaging entry, countdown to Engaged, flee/disengage cycle,
target/self death cleanup, inbound attacker tracking. Veto rules
and surprise semantics still pending.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Combat Phase — Veto rules (CP-010 through CP-016)

**Files:**
- Modify: `internal/state/combatphase/combatphase.go`
- Modify: `internal/state/combatphase/rules.go`

Implement all veto registration methods and wire them into the transition path. The Machine reads vetoes registered via `RegisterCombatantVeto`, `RegisterActivityCheck`, `RegisterLifeCheck`, `RegisterPositionCheck`, plus target-side check registrations.

- [ ] **Step 1: Implement veto registration + chain**

In `rules.go`:

```go
package combatphase

import "github.com/GoMudEngine/GoMud/internal/state"

// vetoChain holds the registered veto functions. Each
// function returns true if the transition is OK, false if
// it should be vetoed.
type vetoChain struct {
	combatantSelf      func() bool                          // self.Combatant
	activitySelf       func() bool                          // self.Activity == Free
	lifeSelf           func() bool                          // self.Life == Alive
	positionSelf       func() bool                          // self.Position == Standing (for flee)
	targetCombatant    func(state.ActorRef) bool            // target.Combatant
	targetLife         func(state.ActorRef) bool            // target.Life == Alive
	targetPresence     func(state.ActorRef) bool            // target.Presence != AFK/Disconnected
}

func (m *Machine) RegisterCombatantVeto(check func() bool) { m.vetoes.combatantSelf = check }
func (m *Machine) RegisterActivityCheck(check func() bool) { m.vetoes.activitySelf = check }
func (m *Machine) RegisterLifeCheck(check func() bool)     { m.vetoes.lifeSelf = check }
func (m *Machine) RegisterPositionCheck(check func() bool) { m.vetoes.positionSelf = check }
func (m *Machine) RegisterTargetCombatantCheck(c func(state.ActorRef) bool) { m.vetoes.targetCombatant = c }
func (m *Machine) RegisterTargetLifeCheck(c func(state.ActorRef) bool)      { m.vetoes.targetLife = c }
func (m *Machine) RegisterTargetPresenceCheck(c func(state.ActorRef) bool)  { m.vetoes.targetPresence = c }
```

Add `vetoes vetoChain` field to Machine in `combatphase.go`.

Update `TransitionToEngaging` to run vetoes:

```go
func (m *Machine) TransitionToEngaging(d EngagingData, r state.TransitionReason) error {
	if m.vetoes.combatantSelf != nil && !m.vetoes.combatantSelf() {
		return &state.VetoError{HandlerName: "combatant_self", Reason: "non-combatant"}
	}
	if m.vetoes.activitySelf != nil && !m.vetoes.activitySelf() {
		return &state.VetoError{HandlerName: "activity_self", Reason: "busy with activity"}
	}
	if m.vetoes.lifeSelf != nil && !m.vetoes.lifeSelf() {
		return &state.VetoError{HandlerName: "life_self", Reason: "not alive"}
	}
	if m.vetoes.targetCombatant != nil && !m.vetoes.targetCombatant(d.Target) {
		return &state.VetoError{HandlerName: "target_combatant", Reason: "target is non-combatant"}
	}
	if m.vetoes.targetLife != nil && !m.vetoes.targetLife(d.Target) {
		return &state.VetoError{HandlerName: "target_life", Reason: "target not alive"}
	}
	if m.vetoes.targetPresence != nil && !m.vetoes.targetPresence(d.Target) {
		return &state.VetoError{HandlerName: "target_presence", Reason: "target unavailable"}
	}

	if err := m.inner.TransitionTo(Engaging, r); err != nil {
		return err
	}
	m.engaging = &d
	if target := lookupMachine(d.Target); target != nil {
		target.RecordInboundAttacker(r.Actor)
	}
	return nil
}
```

Update `TransitionToDisengaging` for the flee+position veto:

```go
func (m *Machine) TransitionToDisengaging(r state.TransitionReason) error {
	if r.Trigger == TriggerFleeCommand &&
		m.vetoes.positionSelf != nil && !m.vetoes.positionSelf() {
		return &state.VetoError{HandlerName: "position_self", Reason: "grappled"}
	}
	if err := m.inner.TransitionTo(Disengaging, r); err != nil {
		return err
	}
	// data setup
	target := state.ActorRef{}
	if m.engaged != nil {
		target = m.engaged.Target
	}
	m.disengaging = &DisengagingData{LastTarget: target}
	return nil
}
```

- [ ] **Step 2: Run veto tests, expect PASS**

```bash
go test ./internal/state/combatphase/ -v -run "TestCP_01[0-6]"
```
Expected: CP-010 through CP-016 pass.

- [ ] **Step 3: Commit**

```bash
git add internal/state/combatphase/
git commit -m "$(cat <<'EOF'
feat(combatphase): veto rules CP-010 through CP-016

Self Combatant/Activity/Life/Position checks; target Combatant/
Life/Presence checks. Vetoed transitions return state.VetoError.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Combat Phase — Surprise attack + end-of-round cascade (CP-023 through CP-025b)

**Files:**
- Modify: `internal/state/combatphase/combatphase.go`

Implement `OnEndOfRoundIfSurprise`, `OnCombatRoundEnd`. The surprise-attack flag carries through Engaging → Engaged, and is consumed at end of first combat round (after all weapon swings). This is where dual/triple-weapon characters get surprise on EVERY swing before stealth breaks.

- [ ] **Step 1: Implement surprise tracking**

Add fields to Machine:

```go
type Machine struct {
	// ... existing fields ...
	endOfRoundIfSurprise []func(state.TransitionReason)
}

func (m *Machine) OnEndOfRoundIfSurprise(fn func(state.TransitionReason)) {
	m.endOfRoundIfSurprise = append(m.endOfRoundIfSurprise, fn)
}

func (m *Machine) OnCombatRoundEnd() {
	if m.State() != Engaged || m.engaged == nil {
		return
	}
	if !m.engaged.SurpriseLeft {
		return // not a surprise round
	}
	// Fire surprise-consumed cascades.
	reason := state.TransitionReason{Trigger: "surprise_consumed"}
	for _, fn := range m.endOfRoundIfSurprise {
		fn(reason)
	}
	m.engaged.SurpriseLeft = false
}
```

Verify `advanceToEngaged` (from Task 5) carries SurpriseLeft properly:

```go
func (m *Machine) advanceToEngaged() {
	prevEngaging := m.engaging
	// ... existing ...
	m.engaged = &EngagedData{
		Target:       prevEngaging.Target,
		SurpriseLeft: prevEngaging.Reason.Trigger == TriggerSurpriseAttack,
	}
}
```

- [ ] **Step 2: Run surprise tests, expect PASS**

```bash
go test ./internal/state/combatphase/ -v -run "TestCP_02[3-5]"
```
Expected: CP-023, CP-024, CP-025 pass.

- [ ] **Step 3: Commit**

```bash
git add internal/state/combatphase/
git commit -m "$(cat <<'EOF'
feat(combatphase): surprise attack carry-through and end-of-round consume

Hidden persists through Engaging→Engaged; stealth-break cascade
fires at end of first combat round, after all weapon swings.
Supports dual/triple/Extra-Arms weapon configurations.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Combat Phase — Tick event dispatch + remaining matrix rows

**Files:**
- Modify: `internal/state/combatphase/combatphase.go`

Implement `OnTickEvent`, `DispatchTickEvent`, `ForceIdle`, `SubscribeAttackersChange`, and the remaining test methods. After this task, every matrix row test passes.

- [ ] **Step 1: Implement tick event dispatch**

Add fields and methods:

```go
type Machine struct {
	// ... existing fields ...
	tickEventListeners       []func(name string, r state.TransitionReason)
	attackersChangeListeners []func([]state.ActorRef)
}

func (m *Machine) OnTickEvent(fn func(name string, r state.TransitionReason)) {
	m.tickEventListeners = append(m.tickEventListeners, fn)
}

func (m *Machine) DispatchTickEvent() {
	var name string
	switch m.State() {
	case Engaged:
		name = "mob_combat_round"
	case Idle:
		name = "mob_idle"
	default:
		return // Engaging and Disengaging are silent
	}
	r := state.TransitionReason{Trigger: "tick"}
	for _, fn := range m.tickEventListeners {
		fn(name, r)
	}
}

func (m *Machine) SubscribeAttackersChange(fn func([]state.ActorRef)) {
	m.attackersChangeListeners = append(m.attackersChangeListeners, fn)
}

func (m *Machine) ForceIdle(r state.TransitionReason) {
	switch m.State() {
	case Engaging, Engaged, Disengaging:
		_ = m.inner.TransitionTo(Idle, r)
		// Clear all state data.
		target := m.CurrentTarget()
		m.engaging = nil
		m.engaged = nil
		m.disengaging = nil
		// Remove self from target's inbound list.
		if t := lookupMachine(target); t != nil {
			t.RemoveInboundAttacker(r.Actor)
		}
	}
}

func (m *Machine) NotifyTargetDied(target state.ActorRef) {
	current := m.CurrentTarget()
	if current != target {
		return
	}
	m.ForceIdle(state.TransitionReason{
		Trigger: TriggerTargetDied,
		Target:  target,
	})
}

func (m *Machine) NotifySelfDied() {
	// Clear outbound.
	target := m.CurrentTarget()
	if !target.IsZero() {
		if t := lookupMachine(target); t != nil {
			// Conservative: remove all attackers entries for this machine's actor.
			// Actor identity lookup is via a reverse map (TBD: maintain in Task 9).
		}
	}
	// Force own state to Idle.
	if m.State() != Idle {
		_ = m.inner.TransitionTo(Idle, state.TransitionReason{Trigger: TriggerSelfDied})
		m.engaging = nil
		m.engaged = nil
		m.disengaging = nil
	}
	// Clear inbound attackers: notify each one they have no target.
	attackers := append([]state.ActorRef{}, m.attackers...)
	m.attackers = nil
	m.notifyAttackersChange()
	// For each inbound attacker, force them to Idle.
	for _, a := range attackers {
		if am := lookupMachine(a); am != nil {
			am.ForceIdle(state.TransitionReason{Trigger: TriggerTargetDied})
		}
	}
}

func (m *Machine) ResolveFlee(success bool) {
	if m.State() != Disengaging {
		return
	}
	if success {
		m.ForceIdle(state.TransitionReason{Trigger: TriggerFleeSuccess})
	} else {
		// Return to Engaged.
		target := state.ActorRef{}
		if m.disengaging != nil {
			target = m.disengaging.LastTarget
		}
		_ = m.inner.TransitionTo(Engaged, state.TransitionReason{Trigger: TriggerFleeFailure})
		m.engaged = &EngagedData{Target: target}
		m.disengaging = nil
	}
}
```

- [ ] **Step 2: Run all combat phase tests, expect ALL PASS**

```bash
go test ./internal/state/combatphase/ -v
```
Expected: every TestCP_* test passes.

- [ ] **Step 3: Build verify**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/state/combatphase/
git commit -m "$(cat <<'EOF'
feat(combatphase): tick events, ForceIdle, death cascade

CP-028 through CP-036 implemented. Behavior Matrix complete;
every test passing. Ready for caller migration.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Combatant flag introduction

**Files:**
- Modify: `internal/characters/character.go`
- Modify: `internal/mobs/mobs.go`
- Modify: YAML files via Engine YAML loader (no per-file edits — backward-compat read)

Add `Character.Combatant bool` field. Migrate `mob.Hostile` semantics. The YAML loader currently reads `hostile: true/false`; preserve that read but expose it as `Combatant` (inverted-ish: `hostile=true` → `Combatant=true` AND aggressive default; `hostile=false` → `Combatant=true` BUT no auto-aggression).

**Important nuance:** Combatant flag is "can be in combat at all" (true by default for almost everything). Aggression behavior (whether to auto-attack on player entry) is separate and lives in archetype/btree logic. The old `mob.Hostile` field conflated these two.

For chunk 0: introduce `Combatant` defaulting to `true`. Add a new `mob.AutoAggro bool` field replacing the auto-attack semantics of `Hostile`. Update YAML to support both: prefer `auto_aggro:` if present, fall back to legacy `hostile:`. The deprecation warning logs once at load.

- [ ] **Step 1: Add Combatant + AutoAggro fields**

In `character.go`:

```go
// Combatant indicates whether this character can participate in
// combat at all. Default true; toggled false by passivity spells,
// non-combatant NPC YAML, etc. Replaces the conflated mob.Hostile.
Combatant bool `yaml:"combatant"`
```

Initialize to `true` in `New()`.

In `mobs.go`, add `AutoAggro bool` field. Keep `Hostile` for backward compat YAML read. In the loader (probably `mobs.LoadDataFiles`), after parsing:

```go
// Backward-compatibility: hostile: true → AutoAggro: true.
// New YAML should use auto_aggro: directly.
if m.Hostile && !m.AutoAggro {
    m.AutoAggro = true
    // Log deprecation warning once per mob:
    mudlog.Info("MobYAML", "mob", m.MobId, "deprecation", "hostile→auto_aggro")
}
```

`Combatant` defaults true for all mobs unless YAML explicitly says `combatant: false` (the new non-combatant flag).

For mobs that have `IsNonCombatant() == true` today (via `non_combatant: true` in YAML), set `Combatant = false` on load.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 3: Verify mob load — boot server briefly**

```bash
timeout 15 go run main.go 2>&1 | grep -E "mobs.LoadDataFiles|panic" | head -5
```
Expected: `loadedCount=225` (or current count), no panics.

- [ ] **Step 4: Commit**

```bash
git add internal/characters/character.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(state): introduce Combatant flag + AutoAggro on mobs

Combatant defaults true. Mob YAML backward-compat: hostile→AutoAggro.
Non-combatant mobs (non_combatant: true) load with Combatant=false.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Wire Combat Phase vetoes to Character state

**Files:**
- Modify: `internal/characters/character.go` (or create `internal/characters/combatphase_wiring.go`)

Register the Combatant/Activity/Life/Position/Target* checks on each Character's CombatPhase machine when it's created. The checks consult the actual Character fields.

For chunk 0:
- `Combatant` check reads `Character.Combatant`.
- `Activity` check reads `Character.IsCrafting() / IsCasting()` (Activity machine arrives in chunk 3; for now use the existing field nil-checks).
- `Life` check reads `Character.Health > 0` (Life machine arrives in chunk 2; for now use health).
- `Position` check reads `Character.CombatPosition == PositionStanding` (Position machine arrives in chunk 4).
- Target* checks look up the target character via `lookupMachine` and check the target's fields.

- [ ] **Step 1: Wire vetoes at Character creation**

Add a method `wireCombatPhaseVetoes()` called from `New()`:

```go
func (c *Character) wireCombatPhaseVetoes() {
	c.CombatPhase.RegisterCombatantVeto(func() bool { return c.Combatant })
	c.CombatPhase.RegisterActivityCheck(func() bool {
		return c.CastingState == nil && c.CraftingState == nil
	})
	c.CombatPhase.RegisterLifeCheck(func() bool { return c.Health > 0 })
	c.CombatPhase.RegisterPositionCheck(func() bool {
		return c.CombatPosition == PositionStanding
	})
	c.CombatPhase.RegisterTargetCombatantCheck(func(t state.ActorRef) bool {
		// Look up target character; check Combatant.
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				return u.Character.Combatant
			}
		}
		if t.IsMob() {
			if m := mobs.GetInstance(t.MobInstanceId); m != nil {
				return m.Character.Combatant
			}
		}
		return true // unknown target = allow (don't veto on stale ref)
	})
	c.CombatPhase.RegisterTargetLifeCheck(func(t state.ActorRef) bool {
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				return u.Character.Health > 0
			}
		}
		if t.IsMob() {
			if m := mobs.GetInstance(t.MobInstanceId); m != nil {
				return m.Character.Health > 0
			}
		}
		return true
	})
	c.CombatPhase.RegisterTargetPresenceCheck(func(t state.ActorRef) bool {
		if t.IsPlayer() {
			if u := users.GetByUserId(t.UserId); u != nil {
				// AFK/Disconnected gate — Presence machine in chunk 5 will
				// replace this. For now check NoAggroTarget grace buff.
				return !u.Character.HasBuffFlag(buffs.NoAggroTarget)
			}
		}
		return true
	})
}
```

This will cause an import cycle (characters → users / mobs). Resolve by:
- Making `wireCombatPhaseVetoes` take callbacks as parameters
- OR defining the check functions externally and having the engine register them after Character creation

Cleaner approach: don't wire inside Character. Move wiring to a new file `internal/hooks/CombatPhase_Vetoes.go` that registers itself in init.

```go
package hooks

func init() {
	characters.OnCharacterCreated(func(c *characters.Character) {
		wireVetoes(c)
	})
}

func wireVetoes(c *characters.Character) {
	c.CombatPhase.RegisterCombatantVeto(/* ... */)
	// etc.
}
```

`characters.OnCharacterCreated` is a new registration hook — add it to `character.go` along with a call site in `New()`.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/characters/ internal/hooks/CombatPhase_Vetoes.go
git commit -m "$(cat <<'EOF'
feat(combatphase): wire vetoes to Character fields

Combat Phase reads Combatant/Activity/Life/Position from existing
Character state. Pending chunks 1-5 will replace these as their
machines land.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Predicate methods on Character

**Files:**
- Modify: `internal/characters/character.go`

Add convenience methods: `IsEngaged()`, `IsInCombat()`, `EngagedTarget()`, `Attackers()`. These are the public API every old `Aggro != nil` check migrates to.

- [ ] **Step 1: Add predicate methods**

```go
// IsEngaged returns true if Combat Phase is Engaged.
// Replacement for `c.Aggro != nil && c.Aggro.RoundsWaiting == 0` (closest equivalent).
func (c *Character) IsEngaged() bool {
	return c.CombatPhase.IsEngaged()
}

// IsInCombat returns true if Combat Phase is anything but Idle.
// Replacement for `c.Aggro != nil`.
func (c *Character) IsInCombat() bool {
	return c.CombatPhase.IsInCombat()
}

// EngagedTarget returns the current Engaged target as an ActorRef.
// Zero value if not Engaged.
func (c *Character) EngagedTarget() state.ActorRef {
	if d, ok := c.CombatPhase.EngagedData(); ok {
		return d.Target
	}
	return state.ActorRef{}
}

// Attackers returns the framework-maintained inbound attacker list.
func (c *Character) Attackers() []state.ActorRef {
	return c.CombatPhase.Attackers()
}
```

- [ ] **Step 2: Build verify**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat(combatphase): predicate methods on Character

IsEngaged(), IsInCombat(), EngagedTarget(), Attackers() — the
public API replacing every Aggro nil-check.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Migrate Aggro readers — usercommands package

**Files:**
- Modify: `internal/usercommands/go.go`
- Modify: `internal/usercommands/bash.go`
- Modify: `internal/usercommands/kick.go`
- Modify: `internal/usercommands/grapple.go`
- Modify: `internal/usercommands/taunt.go`
- Modify: `internal/usercommands/trip.go`
- Modify: `internal/usercommands/submit.go`
- Modify: any other usercommand grepped to read `Aggro`

Every `user.Character.Aggro != nil` becomes `user.Character.IsInCombat()`. Every `user.Character.Aggro.UserId` / `.MobInstanceId` becomes `user.Character.EngagedTarget().UserId` etc.

- [ ] **Step 1: Grep + migrate**

```bash
grep -rn "\.Aggro\b" --include="*.go" internal/usercommands/
```

For each hit, evaluate:
- `if .Aggro != nil` → `if .IsInCombat()` (or `.IsEngaged()` if the intent is "actively in combat-round-dispatching state")
- `.Aggro.UserId` → `.EngagedTarget().UserId`
- `.Aggro.MobInstanceId` → `.EngagedTarget().MobInstanceId`
- `.Aggro.Type == Flee` → `c.CombatPhase.State() == combatphase.Disengaging`

Take notes on any callsite that's doing something subtler — those need design judgment, not mechanical migration.

- [ ] **Step 2: Build + test**

```bash
go build ./...
go test ./internal/usercommands/ -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/
git commit -m "$(cat <<'EOF'
refactor(usercommands): migrate Aggro reads to IsInCombat/EngagedTarget

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Migrate Aggro readers — hooks, behaviortree, combat, mobcommands

**Files:**
- Modify: `internal/hooks/NewRound_AutoHeal.go`
- Modify: `internal/hooks/aggro_helpers.go` (or its replacement file)
- Modify: `internal/behaviortree/conditions_combat.go`
- Modify: `internal/behaviortree/conditions_skullduggery.go`
- Modify: `internal/behaviortree/actions_combat.go`
- Modify: `internal/behaviortree/actions_skullduggery.go` (THIS IS THE CHUNK 2.7 FIX)
- Modify: `internal/combat/*.go`
- Modify: `internal/mobcommands/*.go`

Continue grep-and-replace for `Aggro` reads. Special attention to:

- **`actions_combat.go`**: `actTargetRandomPlayerInRoom` and `actTargetWeakestMobInRoom` currently call `SetAggro`. For `actTargetWeakestMobInRoom` (predator), keep the SetAggro semantics — change to `TransitionToEngaging`. For `actTargetRandomPlayerInRoom`, **remove the SetAggro entirely** — see Task 14.

- **`actions_skullduggery.go`**: This is where chunk 2.7's bug lives. The `try_steal` action currently reads target from `mob.Character.Aggro`. Change it to read target from a per-action target slot or from `ctx.Event.UserId`. The thief archetype's `target_random_player_in_room` action will be reworked in Task 14 to set this slot without going through Combat Phase.

- [ ] **Step 1: Migrate readers in hooks, behaviortree (non-skullduggery), combat, mobcommands**

Grep + replace as in Task 12.

- [ ] **Step 2: Build + test**

```bash
go build ./...
go test ./internal/hooks/ ./internal/behaviortree/ ./internal/combat/ ./internal/mobcommands/ -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/ internal/behaviortree/ internal/combat/ internal/mobcommands/
git commit -m "$(cat <<'EOF'
refactor: migrate Aggro reads across hooks/btree/combat/mobcommands

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Fix the chunk 2.7 bug — non-aggro target picking for skullduggery

**Files:**
- Modify: `internal/behaviortree/actions_combat.go` (target_random_player_in_room)
- Modify: `internal/behaviortree/actions_skullduggery.go` (try_steal/try_plant/try_shadow)

The chunk 2.7 root cause: `target_random_player_in_room` sets aggro. New design: it picks a target and stashes it in an EvalContext slot. `try_steal` etc. read from that slot, not from aggro.

- [ ] **Step 1: Add SoftTarget to EvalContext**

In `internal/behaviortree/conditions.go` (or wherever EvalContext is defined), add:

```go
type EvalContext struct {
	// ... existing fields ...
	SoftTarget state.ActorRef // set by target-picker actions; consumed by skullduggery actions
}
```

- [ ] **Step 2: Rewrite `actTargetRandomPlayerInRoom`**

```go
func actTargetRandomPlayerInRoom(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	playerIds := room.GetPlayers()
	if len(playerIds) == 0 {
		return Failure
	}
	idx := util.Rand(len(playerIds))
	// CRITICAL: do NOT call SetAggro/TransitionToEngaging.
	// Stash in ctx for the next action to consume.
	ctx.SoftTarget = state.ActorRef{UserId: playerIds[idx]}
	return Success
}
```

- [ ] **Step 3: Update `try_steal`/`try_plant`/`try_shadow` to read SoftTarget first**

```go
func actTrySteal(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	// Prefer SoftTarget (set by target_random_player_in_room).
	// Fall back to Combat Phase target if Engaged.
	// Fall back to Event.UserId if neither.
	var target state.ActorRef
	switch {
	case !ctx.SoftTarget.IsZero():
		target = ctx.SoftTarget
	case mob.Character.CombatPhase.IsEngaged():
		target = mob.Character.EngagedTarget()
	case ctx.Event.UserId > 0:
		target = state.ActorRef{UserId: ctx.Event.UserId}
	default:
		return Failure
	}

	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := actions.NewMobActorInRoom(mob, room)
	result := actions.Steal(actor, actions.StealOptions{
		TargetUserId:        target.UserId,
		TargetMobInstanceId: target.MobInstanceId,
	})
	if result.Succeeded {
		return Success
	}
	return Failure
}
```

Same pattern for `actTryPlant`, `actTryShadow`.

- [ ] **Step 4: Build + run combatphase tests**

```bash
go build ./...
go test ./internal/state/combatphase/ ./internal/behaviortree/ -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/
git commit -m "$(cat <<'EOF'
fix(btree): skullduggery target picking uses SoftTarget, not aggro

target_random_player_in_room stashes target in EvalContext;
try_steal/try_plant/try_shadow read from there. No Combat Phase
transition for non-combat target picking.

This is the structural fix for the chunk 2.7 thief-archetype bug.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Migrate Aggro writers + round driver

**Files:**
- Modify: `internal/usercommands/attack.go`
- Modify: `internal/usercommands/flee.go`
- Modify: `internal/mobcommands/attack.go`
- Modify: `internal/mobcommands/flee.go`
- Modify: `internal/behaviortree/actions_combat.go` (actAttack)
- Modify: `internal/hooks/NewRound_DoCombat.go`

Every `SetAggro` becomes `TransitionToEngaging`. Every `EndAggro` becomes `ForceIdle`. The round driver in `NewRound_DoCombat.go` dispatches based on Combat Phase state.

- [ ] **Step 1: Grep all SetAggro / EndAggro sites**

```bash
grep -rn "SetAggro\|EndAggro" --include="*.go" internal/
```

- [ ] **Step 2: Migrate each — pattern**

```go
// OLD:
user.Character.Aggro = &Aggro{UserId: targetId}

// NEW:
user.Character.CombatPhase.TransitionToEngaging(
    combatphase.EngagingData{
        Target: state.ActorRef{UserId: targetId},
        RoundsUntil: 0,
    },
    state.TransitionReason{
        Trigger: combatphase.TriggerAttackCommand,
        Actor:   state.ActorRef{UserId: user.UserId},
        Target:  state.ActorRef{UserId: targetId},
    },
)
```

- [ ] **Step 3: Update round driver to dispatch via Combat Phase**

In `NewRound_DoCombat.go`, replace `Aggro != nil` iteration with Combat Phase state queries. Wire `OnRoundTick()` on every character per round. The driver becomes:

```go
for each user {
    user.Character.CombatPhase.OnRoundTick()
    switch user.Character.CombatPhase.State() {
        case combatphase.Engaged:
            // existing combat resolution
        case combatphase.Idle:
            // skip
    }
    user.Character.CombatPhase.DispatchTickEvent()
}
// Same for mobs.
```

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./... 2>&1 | grep -E "FAIL|ok" | head -30
```

- [ ] **Step 5: Commit**

```bash
git add internal/
git commit -m "$(cat <<'EOF'
refactor: migrate Aggro writers to Combat Phase transitions

SetAggro→TransitionToEngaging; EndAggro→ForceIdle. Round driver
dispatches based on Combat Phase state.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: Wire btree transition events (mob_engaging, mob_disengaging, mob_combat_ended)

**Files:**
- Modify: `internal/behaviortree/events.go` (or wherever btree events are registered)
- Modify: `internal/hooks/CombatPhase_BtreeEvents.go` (NEW)

Register cascade handlers on every mob's CombatPhase that fire the corresponding btree event.

- [ ] **Step 1: Create `internal/hooks/CombatPhase_BtreeEvents.go`**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state"
)

func init() {
	characters.OnCharacterCreated(func(c *characters.Character) {
		// Wire transition events.
		c.CombatPhase.Inner().AfterTransition("btree_events",
			func(from, to combatphase.State, r state.TransitionReason) {
				switch {
				case from == combatphase.Idle && to == combatphase.Engaging:
					behaviortree.TryMobBehavior(/* ... */ "mob_engaging" /* ... */)
				case to == combatphase.Engaged && from == combatphase.Engaging:
					behaviortree.TryMobBehavior(/* ... */ "mob_engaged" /* ... */)
				case to == combatphase.Disengaging:
					behaviortree.TryMobBehavior(/* ... */ "mob_disengaging" /* ... */)
				case to == combatphase.Idle && (from == combatphase.Engaged || from == combatphase.Disengaging):
					behaviortree.TryMobBehavior(/* ... */ "mob_combat_ended" /* ... */)
				}
			})
	})
}
```

- [ ] **Step 2: Document in btree context.md**

Add to `internal/behaviortree/context.md` Events section:

```markdown
- `mob_engaging` — fires once on Combat Phase Idle→Engaging.
- `mob_engaged` — fires once on Combat Phase Engaging→Engaged.
- `mob_disengaging` — fires once on Combat Phase Engaged→Disengaging.
- `mob_combat_ended` — fires once on any →Idle from a combat state.
```

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/behaviortree/ ./internal/hooks/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/CombatPhase_BtreeEvents.go internal/behaviortree/context.md
git commit -m "$(cat <<'EOF'
feat(combatphase): wire btree transition events from cascades

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 17: Migrate aliveness substrate consumers

**Files:**
- Modify: `internal/hooks/MobDeath_FactionRep.go`
- Modify: `internal/hooks/aggro_helpers.go` → split into companion-assist via `CombatPhase_CompanionAssist.go`
- Modify: any other aliveness consumer that reads `Aggro`

Subscribe to Combat Phase observers instead of reading `Aggro`. The faction-rep / opinion / crime / knowledge bumps fire as observers on transitions or as observers on death.

- [ ] **Step 1: Refactor MobDeath_FactionRep to listen to Life→Dead transitions**

(Currently it fires on `events.MobDeath` event. That stays. The change is to use the new `EngagedTarget()` API in any reads.)

- [ ] **Step 2: Companion auto-assist via Attackers() observer**

Replace `CompanionAutoTarget` with a new file that subscribes to Attackers-change events on owned-companion characters. When a companion's Attackers list grows, the owner gets a notification (or auto-engages the new attacker depending on policy).

- [ ] **Step 3: Build + test**

```bash
go build ./...
go test ./internal/hooks/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/
git commit -m "$(cat <<'EOF'
refactor(hooks): aliveness substrate migrates to Combat Phase observers

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 18: SUNSET — delete Aggro field, aggro.go, aggro_helpers.go, NewRound_DoCombat_unified.go

**Files:**
- Delete: `internal/characters/aggro.go`
- Delete: `internal/characters/aggro_grace_test.go`
- Delete: `internal/hooks/aggro_helpers.go`
- Delete: `internal/hooks/NewRound_DoCombat_unified.go`
- Modify: `internal/characters/character.go` (remove `Aggro *Aggro` field)
- Modify: `internal/mobs/mobs.go` (remove `Hostile bool` field if all migration done)

The cutover. Every Aggro consumer should be migrated by Task 17. Delete the old code; CI should stay green.

- [ ] **Step 1: Pre-flight grep**

```bash
grep -rn "\.Aggro\b\|SetAggro\|EndAggro\|\.Hostile\b" --include="*.go" internal/
```
Expected: zero hits (all migrated).

If there are residual hits, fix them before proceeding.

- [ ] **Step 2: Delete files**

```bash
git rm internal/characters/aggro.go internal/characters/aggro_grace_test.go
git rm internal/hooks/aggro_helpers.go
git rm internal/hooks/NewRound_DoCombat_unified.go
```

- [ ] **Step 3: Remove Aggro field**

In `character.go`:

```go
// DELETE this line:
Aggro *Aggro `yaml:"-"`
```

In `mobs.go`:

```go
// DELETE this line:
Hostile bool `yaml:"hostile"`
```

(Keep `AutoAggro bool` field — that's the replacement.)

- [ ] **Step 4: Full build + test**

```bash
go build ./...
go test ./...
```
Expected: clean build, all tests pass.

- [ ] **Step 5: Boot server for data file validation**

```bash
timeout 15 go run main.go 2>&1 | grep -E "LoadDataFiles|panic" | head -20
```
Expected: all loadedCount lines, no panics.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(state): sunset Aggro field + aggro_helpers + Stage 2a unified handler

Combat Phase is now the source of truth. Aggro struct deleted;
old helpers replaced by framework. Stage 2a graveyard cleared.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 19: Documentation updates

**Files:**
- Modify: `internal/characters/context.md`
- Modify: `internal/behaviortree/context.md`
- Modify: `internal/hooks/context.md`
- Modify: `internal/state/context.md`
- Create: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Document Combat Phase API on characters/context.md**

Add a new section "Combat Phase (chunk 0)" describing:
- `CombatPhase *combatphase.Machine` field
- `IsEngaged()`, `IsInCombat()`, `EngagedTarget()`, `Attackers()` predicates
- How transitions work
- Combatant flag

- [ ] **Step 2: Document state framework on `internal/state/context.md`**

Author the framework's package context.md per the existing DOGMud style.

- [ ] **Step 3: Document btree transition events on btree context.md**

(Already added in Task 16; verify and expand if needed.)

- [ ] **Step 4: Create master COMBAT_STATE_ROADMAP.md**

```markdown
# DOGMud — Combat State Machines Roadmap

> Living document. Tracks the 6-chunk state-machines effort.
> Master spec: docs/superpowers/specs/2026-05-13-combat-state-machines-design.md

## Progress

| Chunk | Title | Status |
|-------|-------|--------|
| 0 | Framework + Combat Phase | Done (2026-05-XX) |
| 1 | Awareness | Not started |
| 2 | Life | Not started |
| 3 | Activity | Not started |
| 4 | Position | Not started |
| 5 | Presence | Not started |

(Aliveness chunks 2.7-onward paused for the duration.)
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
docs(state): chunk 0 documentation + COMBAT_STATE_ROADMAP

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 20: Build, full test, in-game smoke

**Files:** (verification only — no commits unless smoke surfaces a fix)

- [ ] **Step 1: Full build**

```bash
go build ./...
```

- [ ] **Step 2: Full test**

```bash
go test ./...
```

- [ ] **Step 3: Boot server**

```bash
go run main.go
```

Watch for clean load: all `LoadDataFiles loadedCount=` lines fire, no panics, "Server Ready" appears.

- [ ] **Step 4: In-game smoke (per spec section "Smoke scenarios")**

Walk through scenarios 1-10 from the spec. Especially scenario 8 (chunk 2.7 thief-archetype regression test).

For scenario 8: log in as quester0, walk to Thornwall Outskirts highwayman. Verify:
- Highwayman idle: picks up sword (gearup), then hides (sneak)
- Highwayman in same room as quester0: does NOT attack
- Highwayman attempts steal — if detection roll succeeds, quester0 sees "lifts gold" message; otherwise silent
- Highwayman never grapples / never enters combat unless quester0 attacks first
- If quester0 attacks first, highwayman engages and combat resolves normally; on quester0 outclassing the highwayman, the highwayman's panic-flee branch fires when HP < 25%

If any scenario fails, debug + fix + re-smoke. Each fix gets its own commit.

- [ ] **Step 5: Kill all running test servers**

Per SOP.

---

## Task 21: Roadmap close-out — mark chunk 0 done, re-validate chunk 2.7

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `COMBAT_STATE_ROADMAP.md`

- [ ] **Step 1: Mark chunk 0 done in COMBAT_STATE_ROADMAP.md**

Replace status to `Done (YYYY-MM-DD)` with a Shipped paragraph summarizing what landed.

- [ ] **Step 2: Reopen chunk 2.7 Task 19 in MOB_ALIVENESS_ROADMAP.md**

The chunk 2.7 thief-archetype smoke scenario 8 should pass after chunk 0. If so, mark chunk 2.7 Done.

- [ ] **Step 3: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md COMBAT_STATE_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): chunk 0 done; chunk 2.7 thief regression validated

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Spec coverage check

| Spec section | Tasks |
|--------------|-------|
| Framework package | 1, 2 |
| Combat Phase state types | 3 |
| Behavior Matrix tests (RED) | 4 |
| Transitions CP-001 through CP-008 | 5 |
| Vetoes CP-010 through CP-016 | 6 |
| Surprise attack CP-023 through CP-025b | 7 |
| Tick/death/Combatant CP-028 through CP-036 | 8 |
| Combatant flag introduction | 9 |
| Veto wiring to Character fields | 10 |
| Predicate methods on Character | 11 |
| Reader migration | 12, 13 |
| Chunk 2.7 fix (target picker) | 14 |
| Writer migration + round driver | 15 |
| Btree transition events | 16 |
| Aliveness substrate migration | 17 |
| Sunset | 18 |
| Documentation | 19 |
| Smoke validation | 20 |
| Roadmap closeout | 21 |

All spec sections covered. The Behavior Matrix in the spec maps row-by-row to tests in Task 4, with implementation across Tasks 5-8.

## Known followups (not in chunk 0)

- Awareness machine (chunk 1) will subscribe to Combat Phase's surprise end-of-round cascade.
- Activity machine (chunk 3) will replace the Activity veto callback with a proper machine query.
- Life machine (chunk 2) will replace the Life veto callback.
- Position machine (chunk 4) will replace the Position veto callback.
- Presence machine (chunk 5) will replace the target-presence veto.
- Charm scheduled-transitions will land alongside or after chunk 5.
