# awareness — Package Documentation

## Overview

The `internal/state/awareness` package is the second consumer of the
`internal/state` framework. It defines the **Awareness state machine**,
replacing the buff-#9 "Hidden flag" as the canonical source of truth for
"is this character hidden?" and handling state transitions for concealment,
detection, and revealing.

Awareness has four states:

| State | Meaning |
|-------|---------|
| `Visible` | Not hidden; detectable by normal observation. |
| `Concealing` | Sneak attempt in flight; transitioning to Hidden. |
| `Hidden` | Successfully concealed; undetectable by unprepared observers. |
| `Revealing` | Detection occurred or surprise round ending; transitioning to Visible. |

The machine is mob/player symmetric: both player and mob `Character` instances
carry a `*awareness.Machine` field. The btree event wiring does not fire for
Awareness transitions (there is no btree transition event); instead, the
framework uses internal state-change cascades and hook subscribers to
integrate with the world (buff #9 mirroring, light-source re-rolls, movement
stamina scaling).

---

## Key Components

### Core Files

- **awareness.go** — `Machine` wrapper, state-data structs, all public API
  (transition methods, predicate methods, registration methods, self tracking).
- **transitions.go** — `validTransitions` table constant and Trigger string
  constants.
- **rules.go** — `vetoChain` and veto registration (activity check, detection
  roll).
- **awareness_test.go** — Behavior Matrix tests AW-001 through AW-033.

---

## Key Functions

### Construction

```go
func NewMachine() *Machine
```

Returns a `Machine` in `Visible`. The machine must be paired with a
`Character` via `SetSelf()` so that detection rolls and cascading updates
can reference the character's stats and inventory.

### Entry into concealment

```go
func (m *Machine) TransitionToConcealing(r TransitionReason) error
```

Initiates a sneak attempt. Runs the activity veto before delegating to
the inner framework. On success:
- Stores `ConcealingData` on the machine.
- Calls `ResolveConcealment()` to immediately compute detection outcome
  (today synchronous; future multi-round concealment will change this).

### Detection resolution

```go
func (m *Machine) ResolveConcealment() error
```

Executes the sneak check against all observers in the room. Runs the
detection-roll veto (which validates the character is actually Concealing)
and performs opposed rolls (Perception + Search vs. Dexterity + Skullduggery).
On success, transitions to `Hidden`. On failure, transitions back to `Visible`.

### Revealing after detection or surprise

```go
func (m *Machine) TransitionToRevealing(r TransitionReason) error
```

Initiates the reveal cascade, typically triggered by detection or end-of-surprise
logic in Combat Phase. Stores the `TransitionReason` for subscribers to query
why the reveal is happening. Immediately cascades to `Visible` in a single call
(today); future multi-round revealing will defer this.

### Room change notification

```go
func (m *Machine) NotifyRoomChanged(oldRoomId int, newRoomId int)
```

Called when the character moves between rooms. Resets to `Visible` and schedules
a light-source awareness re-roll in the new room (via the `Awareness_LightChange`
hook). Prevents hiding from "following" a character across zone boundaries.

### Force visible

```go
func (m *Machine) ForceVisible(r TransitionReason)
```

Transitions to `Visible` from any state, clearing all state-data. Used for
logout, character cleanup, and edge cases. Functionally equivalent to
`TransitionToRevealing` but doesn't queue a cascade.

### Activity check registration

```go
func (m *Machine) RegisterActivityCheck(check func() bool)
```

Wired at character-creation time by `Awareness_Vetoes.go` to ensure the
character is not engaged in crafting or casting before allowing the sneak
transition. The closure reads the character's `CastingState` and
`CraftingState` fields.

### Detection check registration

```go
func (m *Machine) RegisterDetectionCheck(check func() bool)
```

Wired at character-creation time by `Awareness_Vetoes.go` to validate the
sneak attempt is proceeding (character is actually in Concealing state).
Unused pending implementation; included for API symmetry.

### Self reference

```go
func (m *Machine) SetSelf(actorRef state.ActorRef)
func (m *Machine) Self() state.ActorRef
```

`SetSelf()` stores the character's `ActorRef` on the machine so detection
rolls and cascading updates can reference the character by identity without
creating a dependency cycle. Called at character-creation time by the
`RegisterMachine` flow in hooks.

### Predicate methods

```go
func (m *Machine) IsHidden() bool       // State == Hidden
func (m *Machine) IsConcealing() bool   // State == Concealing
func (m *Machine) IsRevealing() bool    // State == Revealing
```

### Registry helpers

```go
func RegisterMachine(c state.ActorRef, m *Machine)
func UnregisterMachine(c state.ActorRef)
func GetMachine(c state.ActorRef) *Machine
```

Framework-maintained registry of active machines, keyed by `ActorRef`.
Populated at character creation; cleared on logout / despawn. Allows the
framework to look up a character's awareness machine by identity without
the caller holding a pointer.

---

## Global State

### machineRegistry

```go
var machineRegistry = map[state.ActorRef]*Machine{}
```

Guarded by `registryMu`. Populated by `RegisterMachine` at character
creation; cleared by `UnregisterMachine` on logout / despawn.

The registry is the bridge between `ActorRef` (the identity type used
in `TransitionReason`) and the live `Machine` pointer. It allows the
framework to call `NotifyRoomChanged` and other notifications on a
character's awareness machine without the caller knowing its memory address.

---

## Data Structure Design

### State enum

```go
type State int
const (
    Visible State = iota
    Concealing
    Hidden
    Revealing
)
```

### Per-state data structs

```go
type VisibleData struct{}
    // Empty — default state has no metadata.

type ConcealingData struct {
    RoundsUntil int  // Reserved for multi-round sneak in future chunks.
}

type HiddenData struct{}
    // Empty — reserved for future light-source tracking or
    // per-observer awareness lists.

type RevealingData struct {
    Reason state.TransitionReason  // Why is the character being revealed?
}
```

Stored as nullable pointer fields on `Machine`. Only the pointer for the
current state is non-nil; transitions nil out the previous state's data.

### validTransitions

```go
var validTransitions = state.TransitionTable[State]{
    Visible:    {Concealing},
    Concealing: {Hidden, Visible},     // Visible on detection failure
    Hidden:     {Revealing},           // Revealing on detection or surprise-end
    Revealing:  {Visible},             // Visible after cascade
}
```

### Trigger constants

Defined in `transitions.go`. Use these string constants instead of
inline literals to ensure stable identifiers across the codebase:

| Constant | Value |
|----------|-------|
| `TriggerSneakCommand` | `"sneak_command"` |
| `TriggerDetectionSuccess` | `"detection_success"` |
| `TriggerDetectionFailure` | `"detection_failure"` |
| `TriggerSurpriseRoundEnd` | `"surprise_round_end"` |
| `TriggerRoomChange` | `"room_change"` |
| `TriggerLightSourceChange` | `"light_source_change"` |
| `TriggerForceVisible` | `"force_visible"` |

---

## Integration Notes

### Consumes

- `internal/state` — framework (`Machine[State]`, `TransitionTable`,
  `TransitionReason`, `ActorRef`).
- `internal/characters` — reads `Dexterity`, `Perception`, `CastingState`,
  `CraftingState` for veto and detection checks.
- `internal/rooms` — room entry/exit for detecting observers.

### Consumed by

- **`internal/characters`** — `Character.Awareness *awareness.Machine`
  field. Initialized in `New()` and (lazily) in `Validate()`. Predicate
  method `IsHidden()` delegates to the machine.
- **`internal/hooks/Awareness_Vetoes.go`** — wires the activity and
  detection-check callbacks at character-creation time via
  `OnCharacterCreated`.
- **`internal/hooks/Awareness_Cascades.go`** — subscribes to `AfterTransition`
  and `OnCombatRoundEnd` events. Mirrors `Hidden` state to buff #9
  (apply on Hidden entry, remove on Hidden exit). Subscribes to Combat Phase
  `OnEndOfRoundIfSurprise` to trigger reveal cascade at end-of-first-round in
  surprise engagements.
- **`internal/hooks/Awareness_LightChange.go`** — subscribes to light-state-change
  events and re-rolls detection when the room's light state changes (scaffolding
  for future-chunk light-source mechanics; today a no-op pending full design).
- **`internal/hooks/Logout_AwarenessCleanup.go`** — calls `ForceVisible` when a
  character logs out, ensuring the awareness machine doesn't leak state.

### Replaces

No direct replacement, but removes the canonical use of the buff-#9 "Hidden" flag
for detection logic. Buff #9 is now a **side-effect carrier only** — the Awareness
machine is the source of truth; the Cascades hook ensures buff #9 stays in sync.

---

## Testing Notes

### awareness_test.go — Behavior Matrix

Tests follow the AW-NNN naming scheme from the chunk 1 spec. Each test
exercises one cell of the state × trigger × veto matrix.

| Range | Area |
|-------|------|
| AW-001 – AW-003 | Basic state transitions (happy path) |
| AW-004 – AW-011 | Detection rolls (observer presence, opposed checks) |
| AW-012 – AW-014 | Light-state interaction scaffold |
| AW-015 – AW-018 | Combat Phase subscription + surprise reveal |
| AW-019 – AW-020 | Hidden movement stamina cost |
| AW-021 – AW-022 | Logout cascade |
| AW-023 | Activity veto (crafting/casting block) |
| AW-024 – AW-027 | CalcSneakScore refactor tests (in actions/skill_helpers_test.go) |
| AW-028 | ForceVisible from any state |
| AW-029 – AW-030 | Registry helpers |
| AW-031 | IsHidden() predicate |
| AW-032 – AW-033 | Writer migration (noisy actions) |

All tests use local `Machine` instances with injected veto closures;
no server / database setup required. Light-state tests (AW-012 – AW-014)
exercise the scaffold scaffolding but do not yet test light-source re-roll logic
(pending full light-system design in later chunks).

---

## Veto Chain

The awareness machine maintains a veto chain (`vetoChain`) with two registered
checks:

1. **Activity Veto** — blocks `Visible → Concealing` if the character is
   crafting or casting. Prevents hidden state during resource-consuming
   activities.
2. **Detection Veto** — blocks `Concealing → Hidden` if detection succeeds.
   (Today integrated into `ResolveConcealment`; kept for API symmetry.)

---

## Detection Roll Mechanics

### Formula

```
sneaker_roll = Dexterity × SkillMultiplier(Skullduggery) + equipment_bonus
observer_roll = Perception × SkillMultiplier(Search) + equipment_bonus
```

Calculation is performed in `actions/skill_helpers.go:CalcSneakScore()` and
exposed via the `ResolveConcealment()` method. Each observer in the room
performs an independent opposed roll.

### Example

A character with Dex 120 and Skullduggery rank 25 sneaking past an observer
with Perception 110 and Search rank 10:
- Sneaker roll: 120 × 1.44 + mods ≈ 173
- Observer roll: 110 × 1.10 + mods ≈ 121
- Success: sneaker wins the opposed check
