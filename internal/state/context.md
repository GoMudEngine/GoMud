# state — Package Documentation

## Overview

The `internal/state` package is the generic finite-state-machine framework
for DOGMud. It provides a type-parameterized `Machine[S]` that handles the
common mechanics of all state machines in the engine: transition-table
enforcement, veto/cascade/observer hooks, and scheduled deferred transitions.

The package is intentionally domain-free. It knows nothing about combat,
characters, or game events. All game-specific logic lives in the consumer
packages: `internal/state/combatphase` (chunk 0), `internal/state/awareness`
(chunk 1), `internal/state/life` (chunk 2), `internal/state/activity` (chunk 3),
`internal/state/position` + `internal/state/control` (chunks 4a-4f),
`internal/state/presence` (chunk 5), and `internal/state/perception` (chunk 6).

The motivation for a shared framework is uniformity across all seven machines:
every machine gets veto + cascade + observer infrastructure for free, and all
machines share the same global `RoundScheduler` so deferred transitions
are driven by a single authoritative tick.

**Arc status (2026-05-19):** The combat-state-machines arc is complete — all
six consumer packages (chunks 0-6) have shipped. Mob aliveness substrate
work can resume.

---

## Key Components

### Core Files

- **machine.go** — `Machine[S]` struct, `NewMachine`, `TransitionTo`,
  `CanTransition`, `BeforeTransition`, `AfterTransition`, `Subscribe`.
- **transition.go** — `ActorRef`, `TransitionReason`, `TransitionTable[S]`,
  `VetoError`, `ErrInvalidTransition`, `ErrVetoed`.
- **handlers.go** — `VetoHandler[S]`, `CascadeHandler[S]`, `ObserverHandler[S]`
  type definitions.
- **scheduled.go** — `RoundScheduler`, `ScheduleAt`, `ScheduleTransition`,
  `CancelScheduled`.
- **machine_test.go** — table-driven unit tests for `Machine[S]`.
- **scheduled_test.go** — unit tests for `RoundScheduler`.

---

## Key Functions

### Machine construction

```go
func NewMachine[S comparable](initial S, table TransitionTable[S]) *Machine[S]
```

Returns a machine in `initial` state with the supplied transition table.
Caller is responsible for registering vetoes, cascades, and observers
before any `TransitionTo` call.

### State query

```go
func (m *Machine[S]) State() S
```

Returns the current state under the machine's own mutex. Safe to call
from any goroutine, though the engine's per-character lock should
still be held for multi-step read-then-act patterns.

### Transition

```go
func (m *Machine[S]) TransitionTo(to S, reason TransitionReason) error
```

Attempts the transition. Steps in order:

1. Checks the `TransitionTable` — returns `ErrInvalidTransition` on miss.
2. Runs veto handlers in registration order — returns `*VetoError` on
   first veto.
3. Updates `current` state atomically under `mu`.
4. Snapshots cascade + observer lists, releases `mu`.
5. Runs all cascade handlers (may call `TransitionTo` on this or other
   machines without deadlock — lock is released before cascades fire).
6. Runs all observer handlers.

### Registration

```go
func (m *Machine[S]) BeforeTransition(name string, h VetoHandler[S])
func (m *Machine[S]) AfterTransition(name string, h CascadeHandler[S])
func (m *Machine[S]) Subscribe(name string, h ObserverHandler[S])
```

All three accept a `name` string used in debug output and veto error
messages. Registration order is preserved; later additions run last.

### Scheduled transitions

```go
func (m *Machine[S]) ScheduleTransition(to S, at ScheduleAt, r TransitionReason)
func (m *Machine[S]) CancelScheduled()
```

`ScheduleTransition` registers a deferred `TransitionTo` that fires when
`RoundScheduler.Tick()` advances past `at.round`. If the transition is
vetoed at fire time the state remains unchanged silently (no caller to
receive the error). `CancelScheduled` cancels all pending entries on this
machine — used by `ForceIdle` and death cascade to prevent stale
transitions from a superseded state.

---

## Global State

### Scheduler singleton

`RoundScheduler` is created once at boot and ticked by the round driver
in `NewRound_DoCombat.go`. All machines in the engine register against
the same scheduler so the tick budget is a single loop, not one-per-machine.

There is no per-character scheduler; all deferred transitions from all
characters flow through one pending list, sorted by target round.

### Cancel tokens

Each `ScheduleTransition` call appends a `*bool` cancel token to both
the `Machine.cancelTokens` slice and the pending entry in the scheduler.
`CancelScheduled` sets all of the machine's tokens to `true`; the
scheduler's `Tick()` skips cancelled entries.

---

## Data Structure Design

### Machine[S]

```go
type Machine[S comparable] struct {
    current      S
    table        TransitionTable[S]
    vetoes        []vetoEntry[S]
    cascades      []cascadeEntry[S]
    observers     []observerEntry[S]
    cancelTokens  []*bool       // for scheduled transitions
    mu            sync.Mutex    // serializes transitions per character
}
```

Type parameter `S` is typically an `int`-based alias defined in the
consumer package (e.g., `combatphase.State`).

### TransitionTable[S]

```go
type TransitionTable[S comparable] map[S][]S
```

Maps each "from" state to a slice of valid "to" states. Checked before
any veto handler runs, providing an invariant-level hard stop.

### TransitionReason

```go
type TransitionReason struct {
    Trigger  string             // stable identifier ("attack_command", etc.)
    Actor    ActorRef           // who initiated the transition
    Target   ActorRef           // optional companion reference
    Metadata map[string]any     // open key-value for transition-specific data
}
```

Propagated through the entire veto/cascade/observer chain so handlers
can make context-aware decisions without knowing the upstream caller.

### ActorRef

```go
type ActorRef struct {
    UserId        int
    MobInstanceId int
}
```

Discriminated reference to a player or mob. Zero value means "no actor."
`IsZero()`, `IsPlayer()`, `IsMob()` helpers avoid raw integer comparisons.

### Handler types

```go
type VetoHandler[S]   func(from, to S, reason TransitionReason) error
type CascadeHandler[S] func(from, to S, reason TransitionReason)
type ObserverHandler[S] func(from, to S, reason TransitionReason)
```

- **Veto** — returns non-nil to block; first non-nil halts the chain.
- **Cascade** — runs after the state change; may chain into further
  transitions. All cascade handlers run (no short-circuit).
- **Observer** — read-only view fired after all cascades complete. Used
  by quest engine, aliveness substrate, telemetry.

---

## Integration Notes

### All consumers (arc complete, 2026-05-19)

| Chunk | Package | States |
|-------|---------|--------|
| 0 | `internal/state/combatphase` | Idle / Engaging / Engaged / Disengaging |
| 1 | `internal/state/awareness` | Visible / Concealing / Hidden / Revealing |
| 2 | `internal/state/life` | Alive / Dead / Respawning |
| 3 | `internal/state/activity` | Free / Casting / Crafting / Salvaging |
| 4a-4f | `internal/state/position` + `internal/state/control` | 14 position states; 5 control-level states |
| 5 | `internal/state/presence` | Active / Connecting / Idle / AFK / Disconnected / Spawning / Dormant / Despawning |
| 6 | `internal/state/perception` | Sighted / Blinded (ships DORMANT) |

Each consumer adds a `Character.<MachineName> *<package>.Machine` field
and registers wire callbacks via `OnCharacterCreated` in the hooks package.
The Perception machine (chunk 6) ships DORMANT — transitions fire but no
consumer reads the state yet. The future messaging framework chunk wires
it into broadcast gating and look-command blocking.

### Import rules

`internal/state` must not import any game-specific package. Its only
standard-library dependency is `sync` and `errors`.

---

## Testing Notes

### machine_test.go

Table-driven tests covering:
- Initial state correctness
- Valid and invalid single-hop transitions
- Veto short-circuit (first veto blocks; second veto not consulted)
- Cascade ordering and mutation side effects
- Observer fire-after-cascade ordering
- `CanTransition` does not mutate state

### scheduled_test.go

Tests covering:
- `RoundsFromNow(n)` fires on the correct tick
- Multiple pending entries fire in round order
- `CancelScheduled` prevents firing
- Stale (cancelled) entries are pruned from the pending list
- Re-registration after cancel adds a fresh non-cancelled entry
