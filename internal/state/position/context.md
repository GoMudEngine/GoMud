# position — Package Documentation

## Overview

The `internal/state/position` package is the fifth consumer of the
`internal/state` framework, after `combatphase`, `awareness`, `life`,
and `activity`. It defines the **Position state machine** — 14 geometric
states drawn from the full BJJ/MMA position taxonomy, covering everything
from standing upright to ground-dominant control positions to defensive
curls.

**Status (fully shipped, 2026-05-18):**

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
- **Chunk 4b-fixup shipped (2026-05-18):** `ControlLevel` drift-needle
  sunset entirely. Per-round drift roll now resolves directly to one
  of five outcome kinds (Hold / Advance / Degrade / Reversal / Escape)
  via `position.ResolveOutcome`. `GrappleData.IsAggressor` bool is the
  sole remaining role discriminator in the Position package; per-side
  dominance tracking moved to the dedicated `internal/state/control`
  FSM on `Character.Control`. ~280 flavor templates in
  `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`.

- **Chunk 4b-fixup-2 shipped (2026-05-18):** Restored a proper
  ControlLevel FSM (`internal/state/control`) to replace the brittle
  `IsControllerRole bool`. `Character.Control *control.Machine` is
  initialized to Neutral at grapple start (symmetric positions) or
  Controlling/Controlled (asymmetric positions) by `TransitionPair`.
  `IsController()` / `IsBeingControlled()` on Character read the
  Control FSM state instead of the deleted bool. Pair iteration in
  `processGrappleTick` uses a seen-map (no more IsController filter),
  fixing Clinch where both sides previously returned false. Per-round
  `applyControlShift` drives ControlLevel shifts via the FSM.

**4c shipped:** Weapon reach utility. `internal/combat/reach.go` reads
`State()` to compute a damage multiplier (position-radius curve:
standing-grapple 0.5 m, ground-grapple 0.3 m). Long weapons degrade in
grapples; short weapons stay effective. Bladed weapons narrate as
Bludgeoning when `ShouldBludgeon` fires. See
`internal/combat/context.md` for the integration.

**4d shipped:** Automatic submission system. Fires from
`internal/hooks/Position_SubmissionTick.go` once per grapple round when
the drift-roll margin snapshot exceeds `SubmissionAttemptAlpha`. Four-tier
resolution (bad / neutral / success / crit) consumes the attempter's
`SubmissionPolicy` and may consult the defender's `SurrenderPolicy` (mercy
policy only). Life cascade extended with `NoDeprogression` +
`GoldLossFraction` DeadData flags. New buffs: broken-limb (id 83) for
cripple outcomes, submission-stunned (id 84) for crit-tier mercy. Role-split
submission mapping lives in `internal/state/position/submissions.go`; the
consumer side is documented in `internal/combat/context.md`.

**Next chunks:** 4e — player command parity (player-facing `grapple`,
`escape`, `position` commands matching mob capabilities). 4f — helpfile +
full doc sweep (player-facing help content for all grapple/position
commands, per-round drift narrative).

---

## Key Components

### Core Files

- **position.go** — `State` enum (14 values), per-state data structs
  (`StandingData`, `ProneData`, `SupineData`, `GrappleData`), `Machine`
  wrapper with predicate methods, machine registry.
- **outcomes.go** — `OutcomeTier` enum, `ResolveOutcome` dispatcher,
  `AdvancementTarget`, `DegradeTarget`, `ReversalTarget` tables.
  **Authoritative source for all per-position transition tables.**
- **transitions.go** — `validTransitions` table (~75 edges), 26 Trigger
  string constants.
- **pair.go** — `TransitionPair`, `ValidateGrapplePair`,
  `DefaultEscapeTarget`, `isSymmetricGrapple`.
- **rules.go** — `TransitionToStanding`, 11 `TransitionToXxx(GrappleData)`
  methods, `TransitionToProne`, `TransitionToSupine`, `ForceStanding`
  admin helper.
- **submissions.go** — `SubmissionType` enum (8 values), role-split pools
  (`TopSubmissionsForPosition`, `BottomSubmissionsForPosition`),
  `IsTopSubEligible`, `IsBottomSubEligible`.
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
| `Clinch` | Both grapplers upright, engaged but neither has dominant
control. |
| `BackStanding` | One grappler has taken the back of the other, standing. |

### Top-Dominant Ground (6)

| State | Description |
|-------|-------------|
| `Mount` | Controller sits on opponent's chest. Most dominant position. |
| `SideControl` | Controller perpendicular, pinning opponent's torso. |
| `KneeOnBelly` | Controller's knee drives into opponent's midsection. |
| `NorthSouth` | Controller's weight on opponent's head-to-toe; heads
opposite. |
| `Crucifix` | Controller isolates both of opponent's arms. |
| `BackGround` | Rear mount on the ground; hooks in or near-hooks. |

### Transitional / Bottom-Active Ground (3)

| State | Description |
|-------|-------------|
| `HalfGuard` | Bottom fighter has one leg trapped; contested control. |
| `Guard` | Bottom fighter's legs wrap opponent's waist. Active defense. |
| `Turtle` | Curled defensive position, exposing back. Solo or with
partner. |

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

Face-down knockdown, alone. Distinct from Supine because submission
paths, recovery difficulty, and back-take vulnerability differ.
`MinRecoveryRounds` gates auto-recovery attempts and replaces the legacy
`PositionRoundsMin` field on `Character`.

### SupineData

```go
type SupineData struct {
    Reason            state.TransitionReason
    MinRecoveryRounds int
    KnockdownSource   state.ActorRef
}
```

Face-up knockdown, alone. Same shape as ProneData today. Split into a
distinct type because Supine mechanics diverge from Prone: Supine can
pull Guard, recovery is easier, and different submission entries apply.

### GrappleData

```go
type GrappleData struct {
    Reason      state.TransitionReason
    Partner     state.ActorRef // zero only for solo Turtle
    IsAggressor bool           // true if this side initiated the grapple;
                               // used as drift-roll tiebreaker in symmetric
                               // positions
}
```

Shared across all 11 grapple states. `IsAggressor` is the sole remaining
role discriminator inside the Position package — it is a tiebreaker for
`determineDriftAttacker` when both sides have identical ControlLevel ranks.
`TransitionPair` no longer stamps an `IsControllerRole` field; per-side
dominance is instead tracked by `Character.Control *control.Machine`
(see `internal/state/control/`).

`IsController()` / `IsBeingControlled()` predicates on `Character` read
the `Control` FSM state (`Controlling` or `Controlled`) rather than a
deleted bool field.

---

## Per-Round Outcome Resolution (chunk 4b-fixup)

Each round, `processGrapplePair` in
`internal/hooks/Position_GrappleTick.go` runs an opposed
Str+WeaponCombat roll modified by stamina and encumbrance multipliers.
The z-score result dispatches through `position.ResolveOutcome` (in
`outcomes.go`) to one of five outcome kinds:

| Outcome | Z condition | Effect |
|---------|-------------|--------|
| **Hold** | \|z\| < 0.5 | No position change |
| **Advance** | z > 0, \|z\| ≥ 0.5 | Controller wins; position changes per
`AdvancementTarget` |
| **Degrade** | z < 0, \|z\| ∈ [0.5, 1.0) | Defender wins moderately;
position regresses per `DegradeTarget` |
| **Reversal** | z < 0, \|z\| ∈ [1.0, 2.0) | Roles swap; position may
change per `ReversalTarget` |
| **Escape** | z ≤ -2.0 | `TransitionPair` to Standing |

The **sub-window gate** is independent: `|z| >= 1.5` on the controller
side opens a sub attempt for `Position_SubmissionTick` (chunk 4d) to
resolve against the post-advance position.

### Why ControlLevel was sunset

The chunk-4b drift needle (`InControl → LosingControl → Neutral →
BecomingControlled → Controlled`) was coherent for symmetric positions
(Clinch, HalfGuard, Turtle) but inverted for asymmetric established
positions (Mount, BackGround, etc.): `InitialControlForPair` started the
defender at `Controlled` by design, so winning the drift fired the escape
gate faster — the opposite of intended behavior. The 9 ground-grapple
states were also effectively unreachable in normal play because the drift
produced only ControlLevel shifts or threshold-escape, never position
advances. The outcome model replaces the needle with direct position
changes, making every ground state reachable and meaningful.

### Per-position transition tables

See `outcomes.go` for the authoritative tables. Summary:

**Advancement** (`AdvancementTarget`):
- Clinch → Mount / SideControl / BackGround by defender posture and tier
- Mount is the striking apex: 1/2-step Advance is a Hold-strike round;
  only TierThreeStep moves to BackGround
- BackGround + Crucifix are terminal apexes (sub-only; Hold on all tiers)
- Most ground states advance toward Mount → BackGround along the
  dominance ladder

**Degrade** (`DegradeTarget`):
- Clinch / Guard / Turtle are terminal degrade positions (Hold round;
  defender must escape or reverse from here)
- BackGround degrades to Mount; Crucifix degrades to BackGround

**Reversal** (`ReversalTarget`):
- Mount → Guard (former defender bridges up, former controller is now
  the Guard-top)
- BackGround → Mount (former defender turns into attacker)
- All other positions: same state, roles swapped

**Escape**: `TransitionPair(controller, controlled, Standing,
TriggerControlledEscape)` — hard exit, both return to Standing.

---

## Role Tracking

Per-side dominance tracking now lives in `Character.Control
*control.Machine` (see `internal/state/control/`). `IsController()` /
`IsBeingControlled()` predicates on `Character` read this FSM.
`GrappleData.IsAggressor` is a separate field used only as a
drift-roll tiebreaker when both sides share equal ControlLevel rank.

`TransitionPair` initializes `Character.Control` at grapple entry:

| Position | Initial ControlLevel |
|---|---|
| Clinch, HalfGuard, Turtle | Both sides → `TransitionToNeutral` |
| BackStanding | Back-taker → `TransitionToControlling`; other side
→ `TransitionToControlled` |
| Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround |
Top / back fighter → Controlling; bottom fighter → Controlled |
| Guard | Bottom fighter → Controlling; top fighter → Controlled |

---

## Transition Graph Summary

The full transition table lives in `transitions.go`. High-level topology:

- **Star center:** `Standing` is reachable from every state. Every state
  can return to `Standing` (grapple break, recovery, escape, or Life
  cascade).
- **Entry path:** All grapple states are reached via `Clinch` or
  `Prone`/`Supine`. You cannot jump `Standing → Mount` directly.
- **Intentional non-edges (design gates):**
  - `Standing → BackStanding` — must go via `Clinch` first
  - `Supine → BackGround` — attacker must flip target into Prone first
  - `Clinch → KneeOnBelly` / `NorthSouth` / `Crucifix` — require ground
    entry via `SideControl` or `Mount`

Within the ground subgraph, controller-initiated advances
(`position_advance` trigger) move between top-dominant states;
controlled-initiated escapes (`position_escape` trigger) move bottom-up
toward `Standing`.

---

## Trigger Constants

26 named trigger constants in `transitions.go`. Use these constants
instead of inline string literals for stable identifiers.

| Constant | Value | Purpose |
|----------|-------|---------|
| `TriggerKnockdownFaceForward` | `"knockdown_face_forward"` | Standing
→ Prone (trip, bash) |
| `TriggerKnockdownFaceBackward` | `"knockdown_face_backward"` |
Standing → Supine |
| `TriggerKnockdownSpell` | `"knockdown_spell"` | Standing → Prone or
Supine (caller picks) |
| `TriggerRecoveryRoll` | `"recovery_roll"` | auto-recovery → Standing |
| `TriggerStandCommand` | `"stand_command"` | explicit stand → Standing |
| `TriggerGrappleEntry` | `"grapple_entry"` | Standing → Clinch |
| `TriggerGrappleBreak` | `"grapple_break"` | any grapple → Standing |
| `TriggerTakedownMount` | `"takedown_mount"` | Clinch → Mount |
| `TriggerTakedownSide` | `"takedown_side"` | Clinch → SideControl |
| `TriggerTakedownGuardPull` | `"takedown_guard_pull"` | Clinch → Guard |
| `TriggerTakedownHalfGuard` | `"takedown_half_guard"` | Clinch →
HalfGuard |
| `TriggerTakedownBackGround` | `"takedown_back_ground"` | Clinch →
BackGround |
| `TriggerBackTakeStanding` | `"back_take_standing"` | Clinch →
BackStanding |
| `TriggerBackTakeGround` | `"back_take_ground"` | various → BackGround |
| `TriggerBackPullDown` | `"back_pull_down"` | BackStanding → BackGround |
| `TriggerPositionAdvance` | `"position_advance"` | controller-side
progression |
| `TriggerPositionDegrade` | `"position_degrade"` | defender wins
moderate; position regresses (chunk 4b-fixup) |
| `TriggerReversal` | `"reversal"` | defender wins big; roles swap
(chunk 4b-fixup) |
| `TriggerControlledEscape` | `"controlled_escape"` | defender wins
decisive; both → Standing (chunk 4b-fixup) |
| `TriggerPositionEscape` | `"position_escape"` | controlled-side escape |
| `TriggerTurtleDefend` | `"turtle_defend"` | ground state → Turtle |
| `TriggerGuardPull` | `"guard_pull"` | Supine → Guard |
| `TriggerMountProneTarget` | `"mount_prone_target"` | attacker mounts
Prone target |
| `TriggerArmIsolation` | `"arm_isolation"` | → Crucifix |
| `TriggerDeath` | `"death"` | Life cascade → Standing |
| `TriggerControlThresholdCrossed` | `"control_threshold_crossed"` |
chunk-4a/4b placeholder (retained for backward compat; not fired in
production) |

---

## Key Functions / Machine API

### TransitionToStanding

```go
func (m *Machine) TransitionToStanding(r state.TransitionReason) error
```

Moves to `Standing` and clears all per-state data slots (`prone`,
`supine`, `grapple`). Used for grapple-break, recovery, escape, and the
Life Dead cascade. The "star center" of the topology.

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
`validTransitions` table. Used by admin commands and emergency cleanup
(e.g., the Life Dead cascade path). No-op if already `Standing`.

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
| `IsTopDominant()` | Mount, SideControl, KneeOnBelly, NorthSouth,
Crucifix, or BackGround |
| `IsOnFloor()` | Prone, Supine, or any ground grapple |

Nil-guard convention: `IsStanding()` returns `true` on a nil machine
(matches `NewMachine()` default). All others return `false` on a nil
machine.

---

## Btree Primitives

`internal/behaviortree/conditions_position.go` registers primitives
in the behaviortree conditions registry. Chunk 4b-fixup removed the
6 ControlLevel control-axis primitives (`mob_control_at_least` etc.);
the two role predicates (`mob_is_in_control` /
`mob_is_being_controlled`) are retained as they read `IsController()`
and `IsBeingControlled()` which are still valid.

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

### Role predicates (chunk 4b, surviving 4b-fixup — 2)

| Condition key | Fires Success when |
|---------------|--------------------|
| `mob_is_in_control` | self `IsController()` |
| `mob_is_being_controlled` | self `IsBeingControlled()` |

The 4 additional chunk-4b ControlLevel primitives
(`mob_control_at_least`, `mob_low_grapple_stamina`,
`target_is_in_control`, `target_is_being_controlled`) were sunset in
chunk 4b-fixup T19.

---

## Submissions (chunk 4d)

The submission-type mapping and eligibility predicates live in
`internal/state/position/submissions.go`. The consumer-side roll,
tier classification, and policy dispatch are in `internal/combat/`
(see `internal/combat/context.md` "Submission System").

### Role-split submission pools

Each grapple position has two submission pools: top-attack subs
(controller side) and bottom-attack subs (controlled side, sparser
by design). The split is defined in `TopSubmissionsForPosition` and
`BottomSubmissionsForPosition`.

### SubmissionType enum (8 values)

| Value | Class | Body part |
|-------|-------|-----------|
| `SubNone` | — | sentinel / empty pool |
| `SubRNC` | Choke | — (no limb) |
| `SubTriangle` | Choke | — (no limb) |
| `SubArmbar` | Joint lock | arm |
| `SubAmericana` | Joint lock | shoulder |
| `SubKimura` | Joint lock | shoulder |
| `SubOmoplata` | Joint lock | shoulder |
| `SubAnaconda` | Choke | — (no limb) |

`CrippleBodyPart(subType) string` returns the limb name for joint-lock
subs and `""` for choke subs. The cripple → subdue degradation in
`submission_outcome.go` uses this to avoid applying a broken-limb buff
for chokes.

### Eligibility predicates

- `IsTopSubEligible(state State, controlState control.State) bool` —
  true when the position has top-attack subs available and
  `controlState == control.Controlling`.
- `IsBottomSubEligible(state State, controlState control.State) bool` —
  true when the position has bottom-attack subs available and
  `controlState == control.Controlled`.

Both predicates take a `control.State` argument (chunk 4b-fixup-2 T14
updated callers from the deleted `isControllerRole bool`).

Both predicates are called from `EvaluateSubAttempt` in
`internal/hooks/Position_SubmissionTick.go` and from the btree
conditions `mob_can_submit_top` / `mob_can_submit_bottom`.

---

## Pair Invariants

`TransitionPair` is the canonical entry for grapple-entry and
controller-initiated position changes: it transitions both sides
atomically and initializes each side's `Character.Control` FSM state
(Neutral or Controlling/Controlled depending on position symmetry),
rolling back if either position transition fails. `ValidateGrapplePair`
is the invariant check used by `Position_ConsistencyCheck.go`.

Key invariants enforced by the pair system:
- Both sides of a pair must be in the same grapple state.
- `Partner` references must be symmetric (A's Partner == B, B's Partner
  == A).
- Symmetric states (Clinch, HalfGuard, Turtle): both sides have
  `Character.Control` at Neutral.
- Asymmetric states (ControlLevel exclusivity): not both at Controlling;
  not both at Controlled. Exactly one side is Controlling and one is
  Controlled after grapple entry.

---

## ControlLevel FSM

Per-side dominance within a grapple is tracked by
`Character.Control *control.Machine` in `internal/state/control/`.
See `internal/state/control/context.md` for the full state diagram,
trigger constants, boundary-cross callback protocol, and invariants.

The Position package is the **source of truth for what position two
characters are in**; the Control package is the **source of truth for
who has dominance**. They are orthogonal: a character can be in Mount
(asymmetric — typically Controlling) or Clinch (symmetric — both Neutral)
independently of the position FSM.

---

## Per-Round Messaging Contract (chunk 4b-fixup)

`internal/hooks/Position_Messaging.go` generates messages from flavor
template YAML loaded by `internal/grapplemessaging/`. The chunk-4b
gradient messages ("losing control of the clinch", "becoming
controlled") were sunset along with ControlLevel. Current message
classes:

| Class | Trigger | Source |
|-------|---------|--------|
| **Outcome** | Per-round `ResolveOutcome` result (Advance / Degrade /
Reversal / Escape / Hold) | `grapple_outcomes.yaml` templates |
| **Transition** | Position FSM state change while grappling | Per
transition (no cooldown) |
| **Stamina warning** | `c.IsLowGrappleStamina()` | Once per grapple |

Outcome templates have `controller` / `controlled` / `observers`
variants and are drawn randomly with per-grapple cooldowns (via
`PerGrappleMessageCooldowns`) to prevent repetition within a single
fight.

---

## Hook Observers (chunks 4b / 4b-fixup)

Wired via `OnCharacterCreated` in `internal/hooks/`:

- **`Position_GrappleTick.go`** — per-round drift (opposed roll,
  stamina cost, `ResolveOutcome` dispatch, `TransitionPair` apply).
  Drives position advancement / degrade / reversal / escape each round.
- **`Position_Messaging.go`** — outcome/transition/stamina-warning
  text generation with per-grapple cooldowns.
- **`Position_ConsistencyCheck.go`** — periodic invariant checker
  (`ValidateGrapplePair`). Logs WARN on partner-ref mismatches,
  ControlLevel exclusivity violations (both Controlling or both
  Controlled), or orphan grapples.

See `internal/hooks/context.md` for the full operational walkthrough.

---

## Cascade Integration

`internal/hooks/Position_Cascades.go` registers a single
`AfterTransition` observer on the Life machine via
`characters.OnCharacterCreated(wirePositionCrossMachineCascades)`.

**Handler key:** `position_life_dead`

**Trigger:** Life `Alive → Dead`

**Effect:** Calls `c.Position.TransitionToStanding(TriggerDeath)` if the
machine is non-nil and not already `Standing`.

This observer is the sole death-cascade for position. Chunk 4b R4
deleted the chunk-2 `Life_Cascades.go` pre-wire that previously reset
`c.CombatPosition = PositionStanding` and `c.GrappleControllerId = 0`
directly — those legacy fields no longer exist.

---

## Intentional Simplifications

1. **Shipped in 4b / 4b-fixup** — Per-round drift rolls fire every
   round and resolve directly to position changes (not ControlLevel
   needle shifts).
2. **No per-state extra structs.** `GrappleData` is still shared across
   all 11 grapple states. Chunk 4c adds `ClinchGrip`, `ArmsIsolated`,
   `HooksIn`, `TrappedLeg`, `GuardVariant` when consumers (weapon
   integration, submission engine) materialize.
3. **Shipped in 4b** — every writer (`trip`, `bash`, `grapple`, `stand`,
   spell knockdown, `AttemptRecovery`, submission outcomes, grapple
   crit-fail) parallel-writes the FSM.
4. **Shipped in 4b R1/R2** — `combat_helpers.go` reads `c.IsX()`
   predicates; `IsThirdPartyAttack` reads `GrappleData.Partner`. Broader
   reader sweep still in progress.
5. **Shipped in 4b R3/R5** — `mobcommands/flee.go`,
   `handlePlayerFlee`, and `RegisterPositionCheck` all read the FSM.
6. **Shipped in 4c** — `internal/combat/reach.go` reads `State()` to
   penalise long weapons in grapples.
7. **Shipped in 4d** — `Position_SubmissionTick.go` fires the opposed
   submission roll each grapple round, gated by `SubmissionAttemptAlpha`.
8. **Shipped in 4b** — all legacy fields and
   `internal/characters/combatposition.go` deleted (S1-S5).
9. **No persistence migration.** The Position machine is `yaml:"-"`.
   Characters log in at `Standing` via `Validate()` initialization. No
   save-file changes.

---

## Persistence

The `Position *position.Machine` field on `Character` is tagged
`yaml:"-"`. The machine is not serialized. Characters always log in at
`Standing`: `Validate()` calls `position.NewMachine()` if the field is
nil (matching the `New()` constructor path). This is correct behavior:
position is transient combat state that resets on disconnect.

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
| PO-025 – PO-028 | Per-state data carries through / clears on
Standing |
| PO-029 – PO-036 | Predicate correctness (including table-driven
sweeps) |
| PO-037 – PO-040 | Cascade verification (SKIP in position_test.go;
live in hooks) |
| PO-041 – PO-043 | Btree primitive smoke (SKIP in position_test.go;
live in btree) |
| PO-044 – PO-045 | Turtle zero-Partner edge case |

Integration tests for cascade (PO-037 through PO-040) live in
`internal/hooks/Position_Cascades_test.go`.

Integration tests for btree primitives (PO-041 through PO-043) live in
`internal/behaviortree/conditions_position_test.go`.

---

## Sunset Notes

### What 4a deletes

Nothing. 4a is purely additive.

### What chunk 4b-fixup deletes

| Deleted artifact | Notes |
|---|---|
| `internal/state/position/control.go` | Entire file removed |
| `ControlLevel` enum + 5 constants | `Neutral`, `InControl`,
`LosingControl`, `BecomingControlled`, `Controlled` — gone from
position package (restored as proper FSM in `internal/state/control/`
by 4b-fixup-2) |
| `GrappleData.ControlLevel` field | Replaced by `IsControllerRole
bool` (itself deleted by 4b-fixup-2 T16; dominance now in
`Character.Control`) |
| `InitialControlForPair` | Removed from `pair.go` — replaced by
`TransitionPair` initializing `Character.Control` via the new FSM |
| `IsControllerLevel`, `IsControlledLevel` | Removed from `pair.go` |
| `ControlRankExported`, `MarginToDelta` | Removed (replaced by §5
outcome buckets in spec) |
| `Machine.MutateGrappleControlLevel` | Removed |
| Chunk-4b gradient message functions | `Position_Messaging.go`
gradient class removed; outcome class added |
| Chunk-4b control-axis btree primitives | 4 of 6 removed in T19;
`mob_is_in_control` + `mob_is_being_controlled` retained as role
checks (now read `Character.Control` FSM) |

### What chunk 4b-fixup-2 additionally deletes

| Deleted artifact | Notes |
|---|---|
| `GrappleData.IsControllerRole bool` | Removed (T16). Dominance
tracking moved to `Character.Control *control.Machine`. |

### Legacy targets — status (fully shipped, 2026-05-18)

| Legacy item | Location | Status |
|-------------|----------|--------|
| `CombatPosition` enum | `internal/characters/` | **Deleted** — T21
S5. All readers migrated in R-sweep. |
| `PositionRoundsMin` field | `Character` struct | **Deleted** — T21
S2. Replaced by `ProneData.MinRecoveryRounds` /
`SupineData.MinRecoveryRounds`. |
| `GrappleControllerId` field | `Character` struct | **Deleted** — T21
S3. Controller identity now derives from
`GrappleData.IsControllerRole`. |
| `ConditionGrappleController` constant | combat / hooks | **Deleted**
— T21 S4. All readers replaced by `c.IsController()`. |
| `Life_Cascades.go` CombatPosition reset | hooks | **Deleted** — R4.
`position_life_dead` observer is now the sole death cascade for
position. |
| `internal/characters/combatposition.go` | `characters/` |
**Deleted** — T21. File removed; all enum helpers gone. |
| `AttemptRecovery()` path | `characters/` | **Shipped W6** — gates on
`IsProne() \|\| IsSupine()`, reads
`ProneData/SupineData.MinRecoveryRounds`. |

---

## What 4e / 4f Bring

- **4e — Player command parity:** Player-facing `grapple`, `escape`,
  `position` commands matching mob capabilities.
- **4f — Helpfile + full doc sweep:** Player-facing help content for all
  grapple/position commands; full doc updates to combat, mobs, btree,
  actions packages; per-round drift narrative for players.
