# position — Package Documentation

## Overview

The `internal/state/position` package is the fifth consumer of the
`internal/state` framework, after `combatphase`, `awareness`, `life`,
and `activity`. It defines the **Position state machine** — 14 geometric
states drawn from the full BJJ/MMA position taxonomy, covering everything
from standing upright to ground-dominant control positions to defensive
curls.

**Status (fully shipped, 2026-05-16):**

- Writer cutover **W1-W8 shipped**: every production writer
  (`ApplyGrappleResult`, submission outcomes, grapple crit-fail, trip,
  bash, spell knockdown, `AttemptRecovery`, `stand`) writes the
  Position FSM directly.
- Reader cutover **R1-R6 all shipped**: combat math
  (`combat_helpers.go`), third-party defense filter
  (`IsThirdPartyAttack`), flee blockers, CombatPhase position check,
  the `{pos}` prompt token, and the Life-cascade pre-wire deletion
  (R4) all read or removed the legacy enum.
- **Legacy sunset S1-S5 complete**: `CombatPosition` enum,
  `PositionRoundsMin` field, `GrappleControllerId` field,
  `ConditionGrappleController` constant, and
  `internal/characters/combatposition.go` are all deleted. The
  Position FSM is the sole source of truth.
- Test fixtures **F1 shipped**: every test that previously wrote
  `Character.CombatPosition` now sets the FSM directly.
- **Control axis live**: per-round opposed-roll drift mechanics
  (`Position_GrappleTick.go`), gradient/transition/stamina messaging
  (`Position_Messaging.go`), and the periodic pair-invariant checker
  (`Position_ConsistencyCheck.go`) all fire in production.

**4c shipped:** Weapon reach utility. `internal/combat/reach.go` reads
`State()` to compute a damage multiplier (position-radius curve:
standing-grapple 0.5 m, ground-grapple 0.3 m). Long weapons degrade in
grapples; short weapons stay effective. Bladed weapons narrate as
Bludgeoning when `ShouldBludgeon` fires. See
`internal/combat/context.md` for the integration.

**Next chunks:** 4d — submissions engine (chokes / joint-locks gated by
ControlLevel thresholds). 4e — player command parity (player-facing
`grapple`, `escape`, `submit`, `position`). 4f — helpfile + full doc
sweep (player-facing help content for the 14-state model, Supine
distinction, per-round drift narrative).

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

Chunk 4b added the per-round opposed Strength/Dexterity rolls that drive
state shifts via `Position_GrappleTick.go`.

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

`internal/behaviortree/conditions_position.go` registers **16 primitives**
in the behaviortree conditions registry — 10 chunk-4a per-state/rollup
primitives + 6 chunk-4b control-axis primitives. Post-4b they're live
(chunk-4b writers transition mob Position FSMs in the same situations
players transition theirs). Canonical mapping table:
`internal/behaviortree/context.md` "Position & Grapple (chunks 4a + 4b)".

### Self-position (chunk 4a — 7)

| Condition key | Fires Success when mob is in |
|---------------|------------------------------|
| `mob_is_standing` | Standing |
| `mob_is_prone` | Prone |
| `mob_is_grappling` | any grapple state |
| `mob_in_mount` | Mount |
| `mob_in_guard` | Guard |
| `mob_in_clinch` | Clinch |
| `mob_in_top_dominant` | any top-dominant ground state |

### Target-position (chunk 4a — 3)

| Condition key | Fires Success when target is in |
|---------------|----------------------------------|
| `target_is_standing` | Standing |
| `target_is_prone` | Prone |
| `target_is_grappled` | any grapple state |

### Control-axis (chunk 4b — 6)

| Condition key | Fires Success when |
|---------------|--------------------|
| `mob_is_in_control` | self `IsController()` (controller side of a grapple pair) |
| `mob_is_being_controlled` | self `IsBeingControlled()` |
| `mob_control_at_least` | self `ControlLevel ≥` the `level` parameter (string-named) |
| `mob_low_grapple_stamina` | self `IsLowGrappleStamina()` — stamina < `GrappleStaminaLowThreshold` |
| `target_is_in_control` | target `IsController()` |
| `target_is_being_controlled` | target `IsBeingControlled()` |

---

## Control-Axis API (chunk 4b)

Five entry points that act on the per-grappler `ControlLevel` axis
without (in most cases) firing FSM transitions:

- `Machine.MutateGrappleControlLevel(newLevel ControlLevel)` —
  sets `ControlLevel` on the current `GrappleData` in place. The FSM
  transition table forbids `Mount→Mount` etc., so per-round drift
  can't go through `TransitionTo*`. Called from
  `Position_GrappleTick.go` only.
- `Machine.ConsumeRecoveryRound()` — decrements `MinRecoveryRounds`
  on the current `ProneData` or `SupineData` in place. Mirrors
  `MutateGrappleControlLevel`. Called from `AttemptRecovery` once per
  round during the minimum-recovery window.
- `Machine.IsController() bool` — true when the character is on the
  controller side of a grapple pair (reads `GrappleData.ControlLevel`
  via `IsControllerLevel`).
- `Machine.IsBeingControlled() bool` — symmetric to `IsController`.
- Free functions in `pair.go`: `IsControllerLevel(ControlLevel) bool`,
  `IsControlledLevel(ControlLevel) bool`,
  `InitialControlForPair(target State, role Role) ControlLevel`,
  `DefaultEscapeTarget(s State) State`,
  `TransitionPair(controller, controlled, target, r)`,
  `ValidateGrapplePair(a, b) error`.

`TransitionPair` is the canonical entry for grapple-entry and
controller-initiated position changes: it transitions both sides
atomically, rolling back if either fails. `ValidateGrapplePair` is
the invariant check used by `Position_ConsistencyCheck.go`.

---

## Per-Round Messaging Contract (chunk 4b)

`internal/hooks/Position_Messaging.go` generates three message
classes with per-grapple cooldowns. Cooldown state lives on
`Character.PerGrappleMessageCooldowns map[string]int` and is cleared
on any `TransitionToStanding` (escape / break / death).

| Class | Trigger | Cooldown | Variants |
|-------|---------|----------|----------|
| **Gradient** | `ControlLevel` crossing (InControl → LosingControl → Neutral → BecomingControlled → Controlled) | Once per direction per grapple | controller / controlled / room |
| **Transition** | Position FSM state change while grappling | Per transition (no cooldown) | controller / controlled / room |
| **Stamina warning** | `c.IsLowGrappleStamina()` (stamina < `GrappleStaminaLowThreshold`) | Once per grapple | self only |

Transition messages have controller/controlled/room variants to
keep prose accurate to perspective ("you scramble out of mount and
into guard" vs "they scramble out of your mount and pull guard").
Gradient messages tighten as the spread widens — `LosingControl` is
"you feel your grip slipping", `Controlled` is "you can barely
move".

---

## Three new hooks observers (chunk 4b)

Wired via `OnCharacterCreated` in `internal/hooks/`:

- **`Position_GrappleTick.go`** — per-round drift (opposed roll,
  stamina cost, threshold-triggered transitions). Drives the
  control-axis evolution.
- **`Position_Messaging.go`** — gradient/transition/stamina-warning
  text generation with per-grapple cooldowns (see above).
- **`Position_ConsistencyCheck.go`** — periodic invariant checker
  (`ValidateGrapplePair`). Logs WARN on partner-ref mismatches,
  asymmetric `ControlLevel`, or orphan grapples.

See `internal/hooks/context.md` "Position Cascade + Observers
(chunks 4a + 4b)" for the full operational walkthrough.

---

## Cascade Integration

`internal/hooks/Position_Cascades.go` registers a single
`AfterTransition` observer on the Life machine via
`characters.OnCharacterCreated(wirePositionCrossMachineCascades)`.

**Handler key:** `position_life_dead`

**Trigger:** Life `Alive → Dead`

**Effect:** Calls `c.Position.TransitionToStanding(TriggerDeath)` if the
machine is non-nil and not already `Standing`.

This observer is now the sole death-cascade for position. Chunk 4b R4
deleted the chunk-2 `Life_Cascades.go` pre-wire that previously reset
`c.CombatPosition = PositionStanding` and `c.GrappleControllerId = 0`
directly — those legacy fields no longer exist (T21 sunset).

---

## Intentional Simplifications

Items 1, 3, 4, 5 below were 4b targets and **shipped** (status updated
post-cutover). Items 2, 6, 7, 8, 9 remain deferred to their named
sub-chunks.

1. ~~No per-round control rolls.~~ **Shipped in 4b** —
   `Position_GrappleTick.go` fires the opposed Strength + Unarmed-combat
   roll every round, scaled by stamina + encumbrance curves, and
   shifts `ControlLevel` via `MutateGrappleControlLevel`.
2. **No per-state extra structs.** `GrappleData` is still shared
   across all 11 grapple states. Chunk 4c adds `ClinchGrip`,
   `ArmsIsolated`, `HooksIn`, `TrappedLeg`, `GuardVariant` when
   consumers (weapon integration, submission engine) materialize.
3. ~~No command-site writers.~~ **Shipped in 4b** — every writer
   (`trip`, `bash`, `grapple`, `stand`, spell knockdown,
   `AttemptRecovery`, submission outcomes, grapple crit-fail)
   parallel-writes the FSM.
4. ~~No combat-modifier reads.~~ **Shipped in 4b R1/R2** —
   `combat_helpers.go` reads `c.IsX()` predicates;
   `IsThirdPartyAttack` reads `GrappleData.Partner`. Broader reader
   sweep (combat/ai.go, combat/grapple.go, etc.) still in progress.
5. ~~No flee veto.~~ **Shipped in 4b R3/R5** — `mobcommands/flee.go`,
   `handlePlayerFlee`, and `RegisterPositionCheck` all read the FSM.
6. ~~No weapon interaction.~~ **Shipped in 4c** — `internal/combat/reach.go`
   reads `State()` to penalise long weapons in grapples. Attack-variant
   selection and weapon-availability gating remain for 4d.
7. **No submission system.** Submissions (chokes, joint locks) require
   `ControlLevel` to exceed thresholds; chunk 4d adds the submission engine.
8. ~~`CombatPosition` enum removal in progress.~~ **Shipped in 4b** —
   all readers migrated (R-sweep + R4), all legacy fields and
   `internal/characters/combatposition.go` deleted (S1-S5).
9. **No persistence migration.** The Position machine is `yaml:"-"`. Characters
   log in at `Standing` via `Validate()` initialization. No save-file changes.

---

## Persistence

The `Position *position.Machine` field on `Character` is tagged `yaml:"-"`.
The machine is not serialized. Characters always log in at `Standing`:
`Validate()` calls `position.NewMachine()` if the field is nil (matching
the `New()` constructor path). This is correct behavior: position is
transient combat state that resets on disconnect.

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

### Legacy targets — status (fully shipped, 2026-05-16)

| Legacy item | Location | Status |
|-------------|----------|--------|
| `CombatPosition` enum | `internal/characters/` | **Deleted** — T21 S5. All readers migrated in R-sweep (4738b26e). |
| `PositionRoundsMin` field | `Character` struct | **Deleted** — T21 S2. Replaced by `ProneData.MinRecoveryRounds` / `SupineData.MinRecoveryRounds`. |
| `GrappleControllerId` field | `Character` struct | **Deleted** — T21 S3. Controller identity now derives from `Position.ControlLevel` + `GrappleData.Partner`. |
| `ConditionGrappleController` constant | combat / hooks | **Deleted** — T21 S4. All readers replaced by `c.IsController()`. |
| `Life_Cascades.go` CombatPosition reset | hooks | **Deleted** — R4 (`a481797f`). `position_life_dead` observer is now the sole death cascade for position. |
| `internal/characters/combatposition.go` | `characters/` | **Deleted** — T21. File removed; all enum helpers gone. |
| `AttemptRecovery()` path | `characters/` | **Shipped W6** — gates on `IsProne() \|\| IsSupine()`, reads `ProneData/SupineData.MinRecoveryRounds`. |

---

## What 4b / 4c / 4d / 4e / 4f Bring

- **4b — Writer cutover + control axis (fully shipped 2026-05-16):**
  Every command-site writer (`trip`, `bash`, `grapple`, `stand`, spell
  knockdown, `AttemptRecovery`, submission outcomes, grapple crit-fail)
  writes the FSM directly. All readers migrated (R1-R6 including R4
  pre-wire deletion). Legacy fields `CombatPosition`, `PositionRoundsMin`,
  `GrappleControllerId`, `ConditionGrappleController`, and
  `internal/characters/combatposition.go` deleted (S1-S5). Per-round
  `ControlLevel` drift, gradient/transition/stamina messaging, and the
  periodic pair-invariant checker all fire.
- **4c — Weapon reach utility (fully shipped 2026-05-16):** `State()` is
  read by `internal/combat/reach.go` to compute a damage multiplier.
  Standing-grapple radius 0.5 m, ground-grapple radius 0.3 m. Bladed
  weapons swap to Bludgeoning narration when `ShouldBludgeon` fires.
- **4d — Submissions engine:** Choke and joint-lock submissions gated by
  `ControlLevel` thresholds, tap/break resolution, injury consequences.
- **4e — Player command parity:** Player-facing `grapple`, `escape`,
  `submit`, `position` commands matching mob capabilities.
- **4f — Helpfile + full doc sweep:** Player-facing help content for all
  grapple/position commands; full doc updates to combat, mobs, btree,
  actions packages deferred until cutover is complete.
