# position — Package Documentation

## Overview

The `internal/state/position` package is the fifth consumer of the
`internal/state` framework, after `combatphase`, `awareness`, `life`,
and `activity`. It defines the **Position state machine** — 14 geometric
states drawn from the full BJJ/MMA position taxonomy, covering everything
from standing upright to ground-dominant control positions to defensive
curls.

**Chunk 4a ships this package DORMANT.** No production code writes to the
machine in 4a; all existing position-driven code paths remain on the
legacy system:

- `CombatPosition` enum (in `internal/characters`) — untouched
- `PositionRoundsMin` field on `Character` — untouched
- `GrappleControllerId` field on `Character` — untouched
- `ConditionGrappleController` condition check — untouched
- Recovery rolls in `AttemptRecovery()` — untouched
- Kick variant selector (`kick` / `stomp` / `knee`) — untouched
- Flee veto, defense degradation, prone multipliers — untouched

**Chunk 4b** is the cutover chunk that wires command-site writers
(`trip`, `bash`, `grapple`, `stand`, etc.) to the new FSM and removes the
legacy `CombatPosition` enum. All behavioral changes belong to 4b or later.

---

## Key Components

### Core Files

- **position.go** — `State` enum (14 values), `ControlLevel` enum (5
  values), per-state data structs (`StandingData`, `ProneData`,
  `SupineData`, `GrappleData`), `Machine` wrapper with predicate methods,
  machine registry.
- **transitions.go** — `validTransitions` table (~75 edges), 22 Trigger
  string constants.
- **rules.go** — `TransitionToStanding`, 11 `TransitionToXxx(GrappleData)`
  methods, `TransitionToProne`, `TransitionToSupine`, `ForceStanding`
  admin helper.
- **position_test.go** — Behavior Matrix tests PO-001 through PO-045.

---

## States

14 states grouped by category:

### Upright (3)

| State | Description |
|-------|-------------|
| `Standing` | Default. No contact. Upright, free to move. |
| `Prone` | Face-down knockdown, alone. Trip, bash, or knockdown spell. |
| `Supine` | Face-up knockdown, alone. Same shape, different mechanics. |

### Standing Grapple (2)

| State | Description |
|-------|-------------|
| `Clinch` | Both grapplers upright, engaged but neither has dominant control. |
| `BackStanding` | One grappler has taken the back of the other, standing. |

### Top-Dominant Ground (6)

| State | Description |
|-------|-------------|
| `Mount` | Controller sits on opponent's chest. Most dominant position. |
| `SideControl` | Controller perpendicular, pinning opponent's torso. |
| `KneeOnBelly` | Controller's knee drives into opponent's midsection. |
| `NorthSouth` | Controller's weight on opponent's head-to-toe; heads opposite. |
| `Crucifix` | Controller isolates both of opponent's arms. |
| `BackGround` | Rear mount on the ground; hooks in or near-hooks. |

### Transitional / Bottom-Active Ground (3)

| State | Description |
|-------|-------------|
| `HalfGuard` | Bottom fighter has one leg trapped; contested control. |
| `Guard` | Bottom fighter's legs wrap opponent's waist. Active defense. |
| `Turtle` | Curled defensive position, exposing back. Solo or with partner. |

---

## Per-State Data

### StandingData

```go
type StandingData struct{}
```

Empty — Standing has no payload. It is the convergent "reset" state that
all other states transition through when grapples break or recovery
succeeds.

### ProneData

```go
type ProneData struct {
    Reason            state.TransitionReason
    MinRecoveryRounds int            // replaces legacy PositionRoundsMin; 0 = can stand immediately
    KnockdownSource   state.ActorRef // who knocked us down
}
```

Face-down knockdown, alone. Distinct from Supine because submission paths,
recovery difficulty, and back-take vulnerability differ. `MinRecoveryRounds`
gates auto-recovery attempts and replaces the legacy `PositionRoundsMin`
field on `Character` (the legacy field is retained in 4a; 4b migrates
consumers).

### SupineData

```go
type SupineData struct {
    Reason            state.TransitionReason
    MinRecoveryRounds int
    KnockdownSource   state.ActorRef
}
```

Face-up knockdown, alone. Same shape as ProneData today. Split into a
distinct type because Supine mechanics diverge from Prone: Supine can pull
Guard, recovery is easier, and different submission entries apply.

### GrappleData

```go
type GrappleData struct {
    Reason       state.TransitionReason
    Partner      state.ActorRef // zero only for solo Turtle
    ControlLevel ControlLevel   // default Neutral; 4b drives changes via opposed rolls
}
```

Shared across all 11 grapple states. In 4a the `ControlLevel` field defaults
to `Neutral` and is never modified by production code. Chunk 4b adds per-round
opposed rolls that shift `ControlLevel` away from `Neutral`. No per-state
extra structs (e.g., `ClinchGrip`, `ArmsIsolated`, `HooksIn`, `TrappedLeg`,
`GuardVariant`) exist yet; those emerge in 4b/4c when consumers materialize.

---

## ControlLevel Enum

```go
const (
    Neutral            ControlLevel = iota // zero value — default for all new GrappleData
    InControl
    LosingControl
    BecomingControlled
    Controlled
)
```

Five values representing the grapple control axis. **`Neutral` is iota=0**
(zero value), so `GrappleData{}` literals always default to `Neutral`
naturally without explicit initialization. This ordering was finalized in
commit `17cf7c3a`; earlier drafts of the plan showed `InControl` as iota=0.
Display ordering in narrative text still follows the "in control → losing →
neutral → becoming → controlled" gradient; the enum order is a zero-value
convenience, not a ranked scale.

In 4a, `ControlLevel` exists only as a data slot. Chunk 4b adds the per-round
opposed Strength/Dexterity rolls that drive state shifts.

---

## Transition Graph Summary

The full transition table lives in `transitions.go`. High-level topology:

- **Star center:** `Standing` is reachable from every state. Every state can
  return to `Standing` (grapple break, recovery, escape, or Life cascade).
- **Entry path:** All grapple states are reached via `Clinch` or `Prone`/`Supine`.
  You cannot jump `Standing → Mount` directly.
- **Intentional non-edges (design gates):**
  - `Standing → BackStanding` — must go via `Clinch` first
  - `Supine → BackGround` — attacker must flip target into Prone first
  - `Clinch → KneeOnBelly` / `NorthSouth` / `Crucifix` — require ground
    entry via `SideControl` or `Mount`

Within the ground subgraph, controller-initiated advances (`position_advance`
trigger) move between top-dominant states; controlled-initiated escapes
(`position_escape` trigger) move bottom-up toward `Standing`.

---

## Trigger Constants

22 named trigger constants in `transitions.go`. Use these constants instead
of inline string literals for stable identifiers.

| Constant | Value | Purpose |
|----------|-------|---------|
| `TriggerKnockdownFaceForward` | `"knockdown_face_forward"` | Standing → Prone (trip, bash) |
| `TriggerKnockdownFaceBackward` | `"knockdown_face_backward"` | Standing → Supine |
| `TriggerKnockdownSpell` | `"knockdown_spell"` | Standing → Prone or Supine (caller picks) |
| `TriggerRecoveryRoll` | `"recovery_roll"` | auto-recovery → Standing |
| `TriggerStandCommand` | `"stand_command"` | explicit stand → Standing |
| `TriggerGrappleEntry` | `"grapple_entry"` | Standing → Clinch |
| `TriggerGrappleBreak` | `"grapple_break"` | any grapple → Standing |
| `TriggerTakedownMount` | `"takedown_mount"` | Clinch → Mount |
| `TriggerTakedownSide` | `"takedown_side"` | Clinch → SideControl |
| `TriggerTakedownGuardPull` | `"takedown_guard_pull"` | Clinch → Guard |
| `TriggerTakedownHalfGuard` | `"takedown_half_guard"` | Clinch → HalfGuard |
| `TriggerTakedownBackGround` | `"takedown_back_ground"` | Clinch → BackGround |
| `TriggerBackTakeStanding` | `"back_take_standing"` | Clinch → BackStanding |
| `TriggerBackTakeGround` | `"back_take_ground"` | various → BackGround |
| `TriggerBackPullDown` | `"back_pull_down"` | BackStanding → BackGround |
| `TriggerPositionAdvance` | `"position_advance"` | controller-side progression |
| `TriggerPositionEscape` | `"position_escape"` | controlled-side escape |
| `TriggerTurtleDefend` | `"turtle_defend"` | ground state → Turtle |
| `TriggerGuardPull` | `"guard_pull"` | Supine → Guard |
| `TriggerMountProneTarget` | `"mount_prone_target"` | attacker mounts Prone target |
| `TriggerArmIsolation` | `"arm_isolation"` | → Crucifix |
| `TriggerDeath` | `"death"` | Life cascade → Standing |
| `TriggerControlThresholdCrossed` | `"control_threshold_crossed"` | 4b placeholder |

---

## Key Functions / Machine API

### TransitionToStanding

```go
func (m *Machine) TransitionToStanding(r state.TransitionReason) error
```

Moves to `Standing` and clears all per-state data slots (`prone`, `supine`,
`grapple`). Used for grapple-break, recovery, escape, and the Life Dead
cascade. The "star center" of the topology.

### TransitionToProne / TransitionToSupine

```go
func (m *Machine) TransitionToProne(d ProneData, r state.TransitionReason) error
func (m *Machine) TransitionToSupine(d SupineData, r state.TransitionReason) error
```

Knockdown transitions. Store per-state data BEFORE calling the inner
framework (so `AfterTransition` observers can read it via `ProneData()` /
`SupineData()`). Rollback on error.

### TransitionToXxx (11 grapple methods)

All 11 grapple states share the same signature:

```go
func (m *Machine) TransitionToXxx(d GrappleData, r state.TransitionReason) error
```

Implemented via `transitionGrapple()` — validates `Partner` is non-zero
(except `Turtle`, which allows solo defensive curl), sets `d.Reason = r`,
stores data before the inner transition, clears non-grapple slots on
success. Returns `ErrPartnerRequired` for zero-Partner violations.

### ForceStanding

```go
func (m *Machine) ForceStanding(r state.TransitionReason)
```

Idempotent transition to `Standing` from any state, bypassing the
`validTransitions` table. Used by admin commands and emergency cleanup (e.g.,
the Life Dead cascade path). No-op if already `Standing`.

### Data accessors

```go
func (m *Machine) ProneData() (ProneData, bool)
func (m *Machine) SupineData() (SupineData, bool)
func (m *Machine) GrappleData() (GrappleData, bool)
```

Return the per-state context while the machine is in the matching state.
Return zero value + `false` otherwise.

### Inner

```go
func (m *Machine) Inner() *state.Machine[State]
```

Returns the underlying `state.Machine[State]`. Used by `rules.go` and
hooks to register `AfterTransition` observers. Not part of the stable
caller API.

---

## Character API — Predicates

`internal/characters/position_predicates.go` exposes 19 nil-guarded
predicates on `Character`. Each delegates to the underlying
`c.Position.IsXxx()` method.

### Per-state (14)

| Method | Returns true when |
|--------|-------------------|
| `IsStanding()` | Position == Standing (default true if machine is nil) |
| `IsProne()` | Position == Prone |
| `IsSupine()` | Position == Supine |
| `IsClinch()` | Position == Clinch |
| `IsBackStanding()` | Position == BackStanding |
| `IsMount()` | Position == Mount |
| `IsSideControl()` | Position == SideControl |
| `IsKneeOnBelly()` | Position == KneeOnBelly |
| `IsNorthSouth()` | Position == NorthSouth |
| `IsCrucifix()` | Position == Crucifix |
| `IsBackGround()` | Position == BackGround |
| `IsHalfGuard()` | Position == HalfGuard |
| `IsGuard()` | Position == Guard |
| `IsTurtle()` | Position == Turtle |

### Rollup (5)

| Method | Returns true when |
|--------|-------------------|
| `IsGrappling()` | any of the 11 grapple states |
| `IsStandingGrapple()` | Clinch or BackStanding |
| `IsGroundGrapple()` | any of the 9 ground grapple states |
| `IsTopDominant()` | Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix, or BackGround |
| `IsOnFloor()` | Prone, Supine, or any ground grapple |

Nil-guard convention: `IsStanding()` returns `true` on a nil machine
(matches `NewMachine()` default). All others return `false` on a nil machine.

---

## Btree Primitives

`internal/behaviortree/conditions_position.go` registers 10 primitives in
the behaviortree conditions registry. All are dormant in 4a (mobs never
transition their Position machine yet); they always return `Failure` unless
the mob's Position was manually set in a test.

### Self-position (7)

| Condition key | Fires Success when mob is in |
|---------------|------------------------------|
| `mob_is_standing` | Standing |
| `mob_is_prone` | Prone |
| `mob_is_grappling` | any grapple state |
| `mob_in_mount` | Mount |
| `mob_in_guard` | Guard |
| `mob_in_clinch` | Clinch |
| `mob_in_top_dominant` | any top-dominant ground state |

### Target-position (3)

| Condition key | Fires Success when target is in |
|---------------|----------------------------------|
| `target_is_standing` | Standing |
| `target_is_prone` | Prone |
| `target_is_grappled` | any grapple state |

---

## Cascade Integration

`internal/hooks/Position_Cascades.go` registers a single
`AfterTransition` observer on the Life machine via
`characters.OnCharacterCreated(wirePositionCrossMachineCascades)`.

**Handler key:** `position_life_dead`

**Trigger:** Life `Alive → Dead`

**Effect:** Calls `c.Position.TransitionToStanding(TriggerDeath)` if the
machine is non-nil and not already `Standing`.

This observer **coexists** with the chunk-2 `Life_Cascades.go` pre-wire
that still resets `c.CombatPosition = PositionStanding` directly and clears
`GrappleControllerId`. Both observers fire on every death. No drift is
possible because the new FSM defaults to `Standing` and 4a has no writers.
The chunk-2 pre-wire is removed in 4b once command sites cut over to write
the new FSM.

---

## Intentional Simplifications

These were left out of 4a deliberately. Each is a named 4b/4c/4d target:

1. **No per-round control rolls.** `ControlLevel` is stored but never driven.
   Chunk 4b adds the opposed Strength/Dexterity roll loop.
2. **No per-state extra structs.** `GrappleData` is shared across all 11
   grapple states. Chunk 4b/4c adds `ClinchGrip`, `ArmsIsolated`, `HooksIn`,
   `TrappedLeg`, `GuardVariant` when consumers materialize.
3. **No command-site writers.** `trip`, `bash`, `grapple`, `stand`, and
   related commands are not wired to the FSM. Chunk 4b handles cutover.
4. **No combat-modifier reads.** Position-based attack/defense modifiers do
   not read the new FSM yet. Chunk 4b wires the reads after command-site
   cutover.
5. **No flee veto.** The position-check veto in `CombatPhase_Vetoes.go` still
   reads `c.CombatPosition`. Chunk 4b migrates it.
6. **No weapon interaction.** Ground positions affect weapon availability and
   attack variants; chunk 4c/4d adds those rules.
7. **No submission system.** Submissions (chokes, joint locks) require
   `ControlLevel` to exceed thresholds; chunk 4d adds the submission engine.
8. **No `CombatPosition` enum removal.** The legacy `CombatPosition` enum and
   all ~50 read sites coexist with the new FSM throughout 4a. Chunk 4b
   removes them after cutover.
9. **No persistence migration.** The Position machine is `yaml:"-"`. Characters
   log in at `Standing` via `Validate()` initialization. No save-file changes.

---

## Persistence

The `Position *position.Machine` field on `Character` is tagged `yaml:"-"`.
The machine is not serialized. Characters always log in at `Standing`:
`Validate()` calls `position.NewMachine()` if the field is nil (matching
the `New()` constructor path). This is correct behavior: position is
transient combat state that resets on disconnect, which is identical to how
`CombatPosition` behaves today.

---

## Testing Notes

### position_test.go — Behavior Matrix

Tests follow the PO-NNN naming scheme. Unit tests use local `Machine`
instances; no server or database setup required.

| Range | Area |
|-------|------|
| PO-001 – PO-004 | Default state + nil-safety |
| PO-005 – PO-018 | Basic valid transitions (14 cases) |
| PO-019 – PO-024 | Invalid-transition rejection (6 cases) |
| PO-025 – PO-028 | Per-state data carries through / clears on Standing |
| PO-029 – PO-036 | Predicate correctness (including table-driven sweeps) |
| PO-037 – PO-040 | Cascade verification (SKIP in position_test.go; live in hooks) |
| PO-041 – PO-043 | Btree primitive smoke (SKIP in position_test.go; live in btree) |
| PO-044 – PO-045 | Turtle zero-Partner edge case |

Integration tests for cascade (PO-037 through PO-040) live in
`internal/hooks/Position_Cascades_test.go`.

Integration tests for btree primitives (PO-041 through PO-043) live in
`internal/behaviortree/conditions_position_test.go`.

---

## Sunset Notes

### What 4a deletes

Nothing. 4a is purely additive.

### Legacy targets catalogued for future sub-chunks

| Legacy item | Location | When removed |
|-------------|----------|--------------|
| `CombatPosition` enum | `internal/characters/` | 4b (after command-site cutover) |
| `PositionRoundsMin` field | `Character` struct | 4b (replaced by `ProneData.MinRecoveryRounds`) |
| `GrappleControllerId` field | `Character` struct | 4b (replaced by `GrappleData.Partner`) |
| `ConditionGrappleController` check | combat / hooks | 4b |
| `Life_Cascades.go` CombatPosition reset | hooks | 4b (pre-wire removed when new FSM is written) |
| Legacy `AttemptRecovery()` path | `characters/` | 4b (migrated to FSM recovery roll) |
| Prone/prone check in kick variant selector | `usercommands/kick.go` | 4b |
| ~50 `c.CombatPosition` / `IsProne()` read sites | various packages | 4b/4c |

---

## What 4b / 4c / 4d / 4e / 4f Bring

- **4b — Writer cutover:** Wires `trip`, `bash`, `grapple`, `stand`,
  `flee` to the new FSM. Removes `CombatPosition` enum and its ~50
  read sites. Cuts over combat-modifier reads. Adds `ControlLevel`
  per-round opposed rolls.
- **4c — Weapon/combat integration:** Position-based weapon availability
  rules, attack variant selection by position, ground-striking modifiers.
- **4d — Submissions engine:** Choke and joint-lock submissions gated by
  `ControlLevel` thresholds, tap/break resolution, injury consequences.
- **4e — Player command parity:** Player-facing `grapple`, `escape`,
  `submit`, `position` commands matching mob capabilities.
- **4f — Helpfile + full doc sweep:** Player-facing help content for all
  grapple/position commands; full doc updates to combat, mobs, btree,
  actions packages deferred until cutover is complete.
