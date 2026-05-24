# Combat State — Chunk 4a: Position FSM Design

> **Side quest from mob aliveness chunk 2.7.** Sub-chunk 4a of the
> combat-state-machines redesign (master spec:
> `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`).
> First of six sub-chunks in the rich-grapple expansion of chunk 4
> (see master-spec section "3a. Position rich-grapple expansion"):
>
>   - **4a (this spec)** — Position FSM: 14 geometric states,
>     transitions, basic per-state data (incl. `ControlLevel` field
>     default Neutral), migration scaffolding from `CombatPosition`
>     enum, btree primitives for position queries. No per-round
>     control rolls yet.
>   - 4b — Control-axis mechanics (per-round opposed rolls +
>     threshold-triggered position transitions + gradient messaging)
>   - 4c — Weapon-utility-by-position table
>   - 4d — Submission system rework
>   - 4e — Third-party interaction asymmetries
>   - 4f — Balance pass + flavor text + full-stack smoke
>
> **Aliveness paused for the duration** of chunks 1-6.

## Goal

Today's `Character.CombatPosition` enum captures only 4 positions
(Standing / Prone / Clinched / Grounded). Real grappling — the
"very real tactical decisions" target identified during brainstorm —
needs the full BJJ/MMA position taxonomy: face-down vs face-up
knockdown, the entire ground-top-dominant family (Mount, Side
Control, Knee-on-Belly, North-South, Crucifix, Back-Ground),
asymmetric transitional states (Half Guard, Guard, Turtle), and
explicit standing back-control (Back-Standing). 14 states total,
plus an orthogonal control gradient.

This sub-chunk:

- **Builds the Position state machine** on the chunk-0 framework
  with 14 geometric states, full transition table, per-state data,
  and the `ControlLevel` field as a data slot (rolls land in 4b).
- **Adds Character predicates** (`IsStanding()`, `IsMount()`,
  `IsGrappling()`, `IsTopDominant()`, etc.) — 19 in total (14
  per-state + 5 rollup).
- **Registers btree primitives** for position queries
  (`mob_is_prone`, `target_is_grappled`, `mob_in_mount`, etc.) —
  ~10 conditions.
- **Wires a Life Dead → Position Standing cascade observer** that
  coexists with the chunk-2 Life pre-wire (no drift possible
  because the new FSM defaults to Standing and 4a has no writers).
- **Ships dormant** — zero behavior change. The existing
  `CombatPosition` enum, `PositionRoundsMin`, `GrappleControllerId`,
  `ConditionGrappleController`, all `combat_*.go` action writers,
  spell handlers, kick variant selector, flee veto, defense
  degradation, and the chunk-0 `RegisterPositionCheck` veto pre-wire
  all remain unchanged. The new FSM is plumbing without consumers.
- **Sets the stage for 4b**, which cuts over command sites to write
  the new FSM, removes the chunk-2 Life pre-wire, repoints the
  chunk-0 veto, and introduces per-round control rolls. 4b is the
  cutover sub-chunk; 4a is the scaffold.

Pure-architecture work. The largest 4a payoff is shape, not
behavior: chunk 4b becomes a writer-migration task rather than a
combined design-and-migration task.

## Non-goals (4a)

- **Per-round control rolls.** `ControlLevel` exists as a data slot
  default Neutral; nothing transitions it. 4b's job.
- **Command-site writer migration.** `trip` / `bash` / `grapple` /
  `stand` / `combat_kick` / spell knockdown / `AttemptRecovery` all
  continue to write `Character.CombatPosition` directly. 4b cuts
  them over.
- **Removal of legacy fields** (`CombatPosition`, `PositionRoundsMin`,
  `GrappleControllerId`, `ConditionGrappleController`). Sunset
  happens incrementally across 4b-4d.
- **Weapon-utility-by-position table.** 4c.
- **Submission rework.** The existing `submission` special-attack
  command stays unchanged. 4d reworks toward opportunistic
  submissions.
- **Third-party defense-degradation changes.** Current "parry/dodge
  removed when grappled" logic in `combat/combat_helpers.go` stays
  exactly as-is. 4e extends.
- **State-specific extras on grapple data** (ClinchGrip,
  ArmsIsolated, HooksIn, TrappedLeg, GuardVariant). 4b/4c add them
  as wrapping structs around `GrappleData` when consumers
  materialize.
- **Control-axis btree primitives** (`mob_is_in_control`,
  `target_is_being_controlled`). 4b.
- **Mob behavior tree authoring** that uses the new position
  primitives. Content / aliveness work post-chunk-6.

## Architecture

`internal/state/position/` package, mirroring `life/`,
`combatphase/`, `awareness/`, `activity/`. Same generics framework
(`state.Machine[S]`), same per-state-data pattern, same
`OnCharacterCreated` init wiring.

### Files

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/position/position.go` | NEW | State enum, per-state data structs, Machine wrapper, registry |
| `internal/state/position/transitions.go` | NEW | Valid-transition table, trigger constants |
| `internal/state/position/rules.go` | NEW | Transition method implementations |
| `internal/state/position/position_test.go` | NEW | Behavior Matrix tests (PO-001 through PO-040ish) |
| `internal/state/position/context.md` | NEW | Package documentation, intentional simplifications |
| `internal/characters/character.go` | MODIFY | Add `Position *position.Machine` field; init in `New()` |
| `internal/characters/validate.go` | MODIFY | Nil-guard init for YAML-loaded chars |
| `internal/characters/position_predicates.go` | NEW | IsStanding / IsProne / IsSupine / IsMount / ... / IsTopDominant predicates |
| `internal/hooks/Position_Cascades.go` | NEW | Life Dead → Position Standing observer (coexists with chunk-2 pre-wire) |
| `internal/hooks/Position_Cascades_test.go` | NEW | Integration test for the cascade |
| `internal/behaviortree/conditions_position.go` | NEW | mob_is_standing / mob_is_prone / mob_is_grappling / ... — 10 primitives |
| `COMBAT_STATE_ROADMAP.md` | MODIFY (at end) | Mark chunk 4a Done |

## States + per-state data

### State enum (14)

```go
type State int

const (
    Standing State = iota
    Prone               // face-down knockdown, alone
    Supine              // face-up knockdown, alone
    Clinch              // standing grapple, both upright
    BackStanding        // standing grapple, one has back of other
    Mount               // ground top-dominant, controller on opponent's chest
    SideControl         // ground top-dominant, perpendicular pin
    KneeOnBelly         // ground top-dominant, knee crushing pin
    NorthSouth          // ground top-dominant, head-to-toe
    Crucifix            // ground top-dominant, opponent's arms isolated
    BackGround          // ground top-dominant, rear mount on ground
    HalfGuard           // ground transitional, one leg trapped
    Guard               // ground bottom-active, legs around opponent
    Turtle              // ground defensive, curled exposing back
)
```

String representation, IsValid(), etc. follow the chunk-2/3 pattern.

### Control axis enum

```go
type ControlLevel int

const (
    InControl ControlLevel = iota
    LosingControl
    Neutral
    BecomingControlled
    Controlled
)
```

Per-grappler. Default Neutral on transition entry. **4a does not
drive control transitions** — the field exists so 4b can fill in
the rolls without an API change.

### Per-state data

```go
// StandingData — empty (default state).
type StandingData struct{}

// ProneData — face-down knockdown, alone.
type ProneData struct {
    Reason            state.TransitionReason
    MinRecoveryRounds int            // replaces PositionRoundsMin; 0 = can stand immediately
    KnockdownSource   state.ActorRef // who knocked us down (for revenge / kill credit)
}

// SupineData — face-up knockdown, alone. Same shape as ProneData
// today; split because recovery and vulnerability mechanics diverge
// (Prone is back-take vulnerable; Supine can pull guard).
type SupineData struct {
    Reason            state.TransitionReason
    MinRecoveryRounds int
    KnockdownSource   state.ActorRef
}

// GrappleData — shared across all 11 grapple states (Clinch,
// BackStanding, Mount, SideControl, KneeOnBelly, NorthSouth,
// Crucifix, BackGround, HalfGuard, Guard, Turtle). 4a does not
// introduce per-state extras. 4b/4c add state-specific wrapping
// structs (ClinchData embedding GrappleData + Grip, etc.) when
// consumers materialize.
type GrappleData struct {
    Reason       state.TransitionReason
    Partner      state.ActorRef    // zero only for solo Turtle (failed-takedown scramble before anyone follows)
    ControlLevel ControlLevel      // default Neutral; 4b drives changes
}
```

The shared `GrappleData` is intentional: 4a doesn't need state-
specific extras. All 11 grapple states answer the same two questions
("who am I grappling, who's winning"). When 4b/4c need extras
(ClinchGrip on Clinch, ArmsIsolated on Crucifix, HooksIn on
BackGround, TrappedLeg on HalfGuard, GuardVariant on Guard), each
gets its own struct that embeds `GrappleData`. Until then,
embedding/wrapping isn't worth the boilerplate.

## Transition graph

Star-ish topology around Standing. Every grapple state can return
to Standing (via break, escape, or cascade); Standing is the entry
point for all combat positions. The 11 grapple states form a
connected sub-graph; some edges are direct, some are mediated.

### From Standing

| To | Trigger |
|---|---|
| Prone | knockdown (face-forward — trip, leg sweep, push from behind, face-plant) |
| Supine | knockdown (face-backward — bash, backward shove, fall on back, knockback spell) |
| Clinch | grapple entry (mutual lockup, body-lock attempt) |

### From Prone (face-down knockdown, alone)

| To | Trigger |
|---|---|
| Standing | recovery roll (auto, gated by MinRecoveryRounds) or explicit `stand` command (stamina cost, bypasses min) |
| BackGround | another character takes back (face-down target — natural back-take) |
| Mount | another character flips + mounts (rare; usually goes via BackGround first) |
| SideControl | another character takes side |
| NorthSouth | another character takes NS |
| Crucifix | another character isolates arms (rare from Prone — usually via SC first) |

### From Supine (face-up knockdown, alone)

| To | Trigger |
|---|---|
| Standing | recovery (easier than Prone — defender can see threats) |
| Guard | attacker engages, defender pulls guard (one-step transition) |
| Mount | attacker mounts the supine target |
| SideControl | attacker takes side |
| KneeOnBelly | attacker pins with KOB |
| NorthSouth | attacker takes NS |
| Crucifix | attacker isolates arms (typical Supine → SC → Crucifix path, but allow direct) |

Note: Supine does NOT directly transition to BackGround (would
require flipping the supine target face-down first, which usually
involves an intermediate Mount or SC).

### From Clinch (standing grapple, both upright)

| To | Trigger |
|---|---|
| Standing | grapple break |
| BackStanding | back-take from standing |
| Mount | takedown into mount (slam, high-amp throw) |
| SideControl | takedown into side |
| Guard | takedown caught in guard (bottom pulls guard during takedown) |
| HalfGuard | takedown caught in half guard |
| BackGround | takedown with back-take (rare combo) |

Clinch → KOB / NorthSouth / Crucifix are NOT direct edges — those
positions require the target already on the ground; reach them via
SideControl first.

### From BackStanding (standing back-grapple)

| To | Trigger |
|---|---|
| Standing | break (back-controller releases, controlled escapes) |
| BackGround | back-controller pulls down (drag to ground while maintaining back) |
| Clinch | controlled turns to face the controller |

### From Mount (ground top-dominant, controller on chest)

| To | Trigger |
|---|---|
| Standing | controller dismounts, or controlled escapes + stands |
| SideControl | controller transitions to side |
| KneeOnBelly | controller transitions to KOB |
| NorthSouth | controller transitions to NS |
| Crucifix | controller isolates arms |
| BackGround | controlled rolls, controller takes back |
| HalfGuard | controlled escapes partially |
| Guard | controlled fully recovers guard |

### From SideControl

| To | Trigger |
|---|---|
| Standing | controller stands off / controlled escapes + stands |
| Mount | controller transitions to mount |
| KneeOnBelly | controller transitions to KOB |
| NorthSouth | controller transitions to NS |
| Crucifix | controller isolates arms |
| BackGround | controlled rolls away, controller takes back |
| HalfGuard | controlled recovers half guard |
| Guard | controlled fully recovers guard |
| Turtle | controlled curls defensively |

### From KneeOnBelly

| To | Trigger |
|---|---|
| Standing | controller steps off |
| Mount | controller transitions to mount |
| SideControl | controller transitions back to SC |
| NorthSouth | controller transitions to NS |
| HalfGuard | controlled escapes partially |
| Guard | controlled fully recovers guard |
| Turtle | controlled curls defensively |

### From NorthSouth

| To | Trigger |
|---|---|
| Standing | controller stands off / controlled escapes |
| Mount | controller transitions to mount |
| SideControl | controller transitions back to SC |
| Crucifix | controller isolates arms |
| Turtle | controlled curls defensively |

### From Crucifix

| To | Trigger |
|---|---|
| Standing | rare full escape |
| SideControl | controller backs off arm isolation |
| Mount | controller transitions to mount |

### From BackGround (rear mount, ground)

| To | Trigger |
|---|---|
| Standing | controlled escapes back to feet |
| Mount | controller transitions when controlled rolls face-up |
| SideControl | controller transitions (rare) |
| Turtle | controlled curls (already face-down — natural defensive) |

### From HalfGuard

| To | Trigger |
|---|---|
| Standing | bottom kicks off + stands, or top stands |
| Guard | bottom recovers full guard |
| Mount | top passes to mount |
| SideControl | top passes to side |
| BackGround | bottom takes back via sweep (rare) |

### From Guard

| To | Trigger |
|---|---|
| Standing | top stands up (stand-in-base) / bottom kicks off and rises |
| HalfGuard | top passes partially |
| Mount | top fully passes to mount |
| SideControl | top fully passes to side |
| NorthSouth | top fully passes to NS |
| BackGround | bottom takes back via sweep |

### From Turtle

| To | Trigger |
|---|---|
| Standing | turtled person stands |
| BackGround | attacker hooks in |
| SideControl | attacker spreads turtled person |
| Mount | attacker rolls turtled person over (rare) |

### Implicit edge: cascade-driven

Every state can transition to Standing via the Life Dead cascade
observer (see "Cross-machine cascade" below).

### Total

Approximately **75 valid transition edges** across the 14×14
matrix. Authoritative validation lives in `transitions.go`.

## Trigger constants

```go
const (
    // Knockdown / falls
    TriggerKnockdownFaceForward = "knockdown_face_forward"  // → Prone
    TriggerKnockdownFaceBackward = "knockdown_face_backward" // → Supine
    TriggerKnockdownSpell        = "knockdown_spell"         // → Prone or Supine (caller decides)

    // Recovery
    TriggerRecoveryRoll = "recovery_roll" // → Standing (auto, gated by MinRecoveryRounds)
    TriggerStandCommand = "stand_command" // → Standing (explicit, stamina cost, bypasses min)

    // Grapple entry / break
    TriggerGrappleEntry = "grapple_entry"  // Standing → Clinch
    TriggerGrappleBreak = "grapple_break"  // any grapple → Standing

    // Takedowns (Clinch → various ground states)
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
    TriggerTurtleDefend = "turtle_defend" // ground states → Turtle
    TriggerGuardPull    = "guard_pull"    // Supine → Guard

    // Top-side opportunistic
    TriggerMountProneTarget = "mount_prone_target" // attacker takes Prone target

    // Crucifix arm-isolation
    TriggerArmIsolation = "arm_isolation" // → Crucifix

    // Cascades
    TriggerDeath = "death" // any → Standing (Life cascade)

    // 4b-placeholder (named here so 4a tests can reference, but no
    // 4a code path fires this; 4b implements the rolls + threshold
    // logic):
    TriggerControlThresholdCrossed = "control_threshold_crossed"
)
```

## Machine API

```go
package position

// Constructor
func NewMachine() *Machine

// State access
func (m *Machine) State() State
func (m *Machine) Inner() *state.Machine[State]   // used by hooks/tests
func (m *Machine) SetSelf(ref state.ActorRef)
func (m *Machine) Self() state.ActorRef

// Transition methods (one per state)
func (m *Machine) TransitionToStanding(r state.TransitionReason) error
func (m *Machine) TransitionToProne(d ProneData, r state.TransitionReason) error
func (m *Machine) TransitionToSupine(d SupineData, r state.TransitionReason) error
func (m *Machine) TransitionToClinch(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToBackStanding(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToMount(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToSideControl(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToKneeOnBelly(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToNorthSouth(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToCrucifix(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToBackGround(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToHalfGuard(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToGuard(d GrappleData, r state.TransitionReason) error
func (m *Machine) TransitionToTurtle(d GrappleData, r state.TransitionReason) error

// Data accessors
func (m *Machine) ProneData() (ProneData, bool)
func (m *Machine) SupineData() (SupineData, bool)
func (m *Machine) GrappleData() (GrappleData, bool)  // returns data for ANY grapple state

// State predicates (one per state) — also exposed on Character
func (m *Machine) IsStanding() bool
func (m *Machine) IsProne() bool
func (m *Machine) IsSupine() bool
func (m *Machine) IsClinch() bool
func (m *Machine) IsBackStanding() bool
func (m *Machine) IsMount() bool
func (m *Machine) IsSideControl() bool
func (m *Machine) IsKneeOnBelly() bool
func (m *Machine) IsNorthSouth() bool
func (m *Machine) IsCrucifix() bool
func (m *Machine) IsBackGround() bool
func (m *Machine) IsHalfGuard() bool
func (m *Machine) IsGuard() bool
func (m *Machine) IsTurtle() bool

// Rollup predicates
func (m *Machine) IsGrappling() bool         // any of 11 grapple states
func (m *Machine) IsStandingGrapple() bool   // Clinch | BackStanding
func (m *Machine) IsGroundGrapple() bool     // Mount/SC/KOB/NS/Crucifix/BackGround/HalfGuard/Guard/Turtle
func (m *Machine) IsTopDominant() bool       // Mount/SC/KOB/NS/Crucifix/BackGround
func (m *Machine) IsOnFloor() bool           // Prone | Supine | any ground grapple

// Registry (cross-character lookups for grapple-pair writes in 4b+)
func RegisterMachine(ref state.ActorRef, m *Machine)
func UnregisterMachine(ref state.ActorRef)
```

`TransitionToStanding` takes no data (default state has no payload).
All grapple transitions take `GrappleData` — `Partner` is required
non-zero EXCEPT for `TransitionToTurtle` (allow zero Partner for
solo-defensive-curl scenarios).

`GrappleData()` returns the same struct regardless of which grapple
state is active. Callers that want to distinguish "am I mounted"
vs "am I in mount" check the state predicates separately.

## Character predicates

```go
// In internal/characters/position_predicates.go (NEW file)

func (c *Character) IsStanding() bool
func (c *Character) IsProne() bool
func (c *Character) IsSupine() bool
func (c *Character) IsClinch() bool
func (c *Character) IsBackStanding() bool
func (c *Character) IsMount() bool
func (c *Character) IsSideControl() bool
func (c *Character) IsKneeOnBelly() bool
func (c *Character) IsNorthSouth() bool
func (c *Character) IsCrucifix() bool
func (c *Character) IsBackGround() bool
func (c *Character) IsHalfGuard() bool
func (c *Character) IsGuard() bool
func (c *Character) IsTurtle() bool

func (c *Character) IsGrappling() bool
func (c *Character) IsStandingGrapple() bool
func (c *Character) IsGroundGrapple() bool
func (c *Character) IsTopDominant() bool
func (c *Character) IsOnFloor() bool
```

Each delegates to `c.Position.IsXxx()` with a nil guard
(`if c.Position == nil { return defaultFor(state) }`).

**Important:** these are net-new methods. They do NOT replace the
existing `c.CombatPosition == characters.PositionXxx` checks
scattered across the codebase. Both APIs coexist during 4a; 4b
swaps call sites incrementally.

The existing helpers on the `CombatPosition` enum
(`IsGroundPosition()`, `IsGrapplePosition()` on
`internal/characters/combatposition.go`) stay unchanged in 4a.
They continue to read the legacy enum. 4b/4c sunset them.

## Cross-machine cascade (4a — single observer)

```go
// internal/hooks/Position_Cascades.go (NEW)

package hooks

import (
    "github.com/GoMudEngine/GoMud/internal/characters"
    "github.com/GoMudEngine/GoMud/internal/state"
    "github.com/GoMudEngine/GoMud/internal/state/life"
    "github.com/GoMudEngine/GoMud/internal/state/position"
)

// wirePositionCrossMachineCascades registers Position-side observers
// for cross-machine transitions. 4a wires only the Life Dead → Standing
// cascade; 4b adds the per-round control-roll subscriber, and 4d
// may add Activity-side hooks if submissions touch Position.
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

That's the entire 4a cross-machine surface. Other cross-machine
work deferred:

- `RegisterPositionCheck` in `CombatPhase_Vetoes.go` continues to
  read the legacy enum. 4b repoints to `c.IsStanding()` or
  similar.
- Flee veto continues to read the legacy enum. 4b/4e repoint.
- Defense degradation when grappled continues to read
  `c.CombatPosition.IsGrapplePosition()`. 4e revisits.

## Btree primitives (10)

```go
// internal/behaviortree/conditions_position.go (NEW)

// Self-position queries
"mob_is_standing"     // mob in Standing
"mob_is_prone"        // mob in Prone (face-down)
"mob_is_grappling"    // mob in any grapple state
"mob_in_mount"        // mob in Mount specifically (controller side, by historical convention)
"mob_in_guard"        // mob in Guard specifically (could be top or bottom — disambiguation comes in 4b)
"mob_in_clinch"       // mob in Clinch
"mob_in_top_dominant" // mob in Mount/SC/KOB/NS/Crucifix/BackGround

// Target-position queries (resolved via ResolveAggroTarget — same pattern as existing
// target_is_casting condition)
"target_is_standing"
"target_is_prone"
"target_is_grappled"  // target in any grapple state
```

All 10 registered in 4a, **dormant**: they always return Failure
because no mob's `Position` machine has been transitioned (the FSM
defaults to Standing and 4a has no writers). They become useful
once 4b drives transitions from real combat. Content authors can
reference them in btree YAML; the conditions will start firing
correctly once 4b lands.

Control-axis btree primitives (`mob_is_in_control`,
`target_is_being_controlled`) are NOT included in 4a — they require
the control rolls that 4b adds.

`mob_is_supine`, `target_is_supine`, `mob_in_side_control`,
`mob_in_back_ground`, etc. are also intentionally omitted from 4a
to keep the primitive list small. Add them in 4b/content-time as
needed; the rollup `mob_in_top_dominant` covers the broad cases.

## Mob/player parity

Full parity by construction — `Character.Position` is a single field
that applies identically to players and mobs. The two existing
asymmetries to preserve:

- **Player-only commands**: `stand` is a usercommand only; mobs
  recover via the auto recovery roll path (`AttemptRecovery` —
  legacy code path, unchanged in 4a). 4b/4d might surface a mob
  `stand` equivalent via btree, but not needed in 4a.
- **Action-layer parity already done**: `trip`, `bash`, `grapple`,
  `kick` all have mob equivalents. They write the legacy enum
  identically. 4b's command-site cutover handles both sides.

No new mob behaviors authored in 4a. The 10 new btree primitives
are available for content authors but no archetype YAML modifies
to use them in 4a — that's content work after the machine drives
real transitions in 4b.

## Behavior Matrix preview (drafted in plan, completed in tests)

~35-45 rows total, all geometric-transition focused. No control-roll
rows (4b). Grouped:

- **Default + nil-safety** (~4 rows). Character starts Standing.
  Machine-less Character returns Standing-equivalent defaults.
  Validate() nil-guard initializes for YAML-loaded chars.
  Persistence: re-load produces Standing regardless of pre-save
  state.
- **Basic transitions** (~14 rows). Each transition KIND exercised
  with a representative source/target pair: Standing → Prone,
  Standing → Supine, Standing → Clinch, Clinch → Mount, Mount →
  SideControl, SideControl → Mount, Mount → BackGround, BackGround
  → Mount, Mount → Standing, etc.
- **Invalid-transition rejection** (~6 rows). Standing → Mount
  fails (must go via Clinch or Prone/Supine). Clinch → KOB fails.
  Mount → Prone fails (controller can't transition to a non-grapple
  state without going via Standing first). BackStanding → Mount
  is invalid (rare grapple geometry; force via BackGround first).
- **GrappleData carries Partner + ControlLevel** (~4 rows). Set
  Mount → read GrappleData → Partner + ControlLevel preserved.
  Default ControlLevel is Neutral when not specified.
- **Predicates correctness** (~8 rows). IsStanding / IsProne /
  IsSupine / IsGrappling / IsTopDominant / IsOnFloor / IsBackGround
  / IsGuard return correct booleans for each state, with sampled
  inputs.
- **Cascade verification** (~4 rows). Life Dead cascades Position
  → Standing from each grapple state (Mount, Guard, BackGround,
  Turtle as representative samples). Chunk-2 pre-wire continues
  to fire alongside (test verifies both legacy enum and new FSM
  reach Standing).
- **Btree primitive smoke** (~3 rows). `mob_in_mount` returns
  Success only when mob in Mount. `target_is_grappled` returns
  Success only when target in any grapple state.
- **Turtle Partner edge case** (~2 rows). `TransitionToTurtle(d)`
  with zero Partner succeeds. Other grapple transitions with zero
  Partner fail with a clear error.

Full matrix authored in chunk-4a plan PR.

## Sunset list (4a — almost nothing)

4a deletes nothing from production code. The "sunset" surface for
4a:

- **Tests**: existing tests that fake `c.CombatPosition = PositionXxx`
  to set up scenarios stay valid. The new FSM is dormant. No
  fixture migration needed in 4a.

Sunset list cataloged for FUTURE sub-chunks (so 4b knows the
targets):

- `Character.CombatPosition CombatPosition` field → 4b-or-later
  (after all command sites cut over to write the new FSM).
- `Character.PositionRoundsMin int` field → 4b (folds into
  ProneData.MinRecoveryRounds + SupineData.MinRecoveryRounds).
- `Character.GrappleControllerId int` field → 4b/4d (folds into
  GrappleData.Partner + ControlLevel; "I am the controller" is
  derived from `ControlLevel ∈ {InControl, LosingControl}` rather
  than a stored bool or condition).
- `Character.HasCondition(ConditionGrappleController)` checks →
  4b/4d (same derivation).
- `internal/characters/combatposition.go` enum + helpers → 4b-or-
  later (last to go; many callsites).
- Chunk-2 Life cascade pre-wire (raw `c.CombatPosition =
  PositionStanding` + `c.GrappleControllerId = 0` lines 55-57 of
  `Life_Cascades.go`) → 4b removes once command sites cut over.
- Chunk-0 `CombatPhase_Vetoes.go:RegisterPositionCheck` raw-field
  read → 4b rewires to `c.IsStanding()`.

## Notes on intentional simplifications

These become prose in `internal/state/position/context.md`:

1. **No Prone/Supine to Prone consolidation. Two states intentionally.**
   Earlier brainstorm versions used a single Prone state. The split
   is structural: submission paths diverge (Prone → back-take → RNC
   only; Supine → guard pull → all BJJ submissions). Recovery
   difficulty differs. Vulnerability asymmetries differ. Conflation
   would lose mechanically meaningful information.
2. **No per-grapple-state extras on GrappleData (yet).** ClinchGrip,
   ArmsIsolated, HooksIn, TrappedLeg, GuardVariant are all
   placeholders for 4b/4c. The shared `GrappleData` type covers
   4a's needs (Partner + ControlLevel + Reason). Per-state extras
   land as wrapping structs (e.g., `type ClinchData struct {
   GrappleData; Grip ClinchGrip }`) when consumers materialize.
3. **No control-axis rolls.** `ControlLevel` field exists in
   GrappleData (default Neutral) but nothing in 4a transitions it.
   4b implements the per-round opposed rolls.
4. **No btree primitives for control axis.** `mob_is_in_control` /
   `target_is_being_controlled` are 4b additions.
5. **Cascade observers coexist with chunk-2 pre-wire.** Both fire
   on Life Dead; both reach Standing; impossible to drift because
   the new FSM defaults to Standing and 4a has no writers. Pre-wire
   removed in 4b after command sites cut over.
6. **Standing → BackStanding is NOT a direct edge.** Must go via
   Clinch. Real-life "running back-take" is rare enough to model
   as a two-step transition (sprint behind = Clinch with high
   back-take roll, then immediate back-take). Simplifies the
   graph.
7. **Supine → BackGround is NOT a direct edge.** Would require the
   attacker to flip a supine target face-down, which involves an
   intermediate position. Model via Supine → Mount → BackGround or
   Supine → SideControl → BackGround.
8. **Clinch → KOB / NorthSouth / Crucifix are NOT direct edges.**
   Those positions require the target already on the ground. Reach
   them via Clinch → SideControl → KOB/NS/Crucifix.
9. **No `mob_is_supine`, `mob_in_side_control`, etc. btree
   primitives.** Only the most commonly-queried positions get
   individual primitives in 4a; the rollup `mob_in_top_dominant`
   covers most cases. Add more in 4b/content-time as needed.

## Persistence

Non-persistent (`yaml:"-"`). Matches existing `CombatPosition`
behavior and chunks 1-3. Character logs in at Standing regardless
of prior in-memory state. Combat doesn't survive logout anyway.

## Open questions / risks

- **Reachability test gap.** 4a ships the FSM with valid transitions
  defined but no production-code writers. Unit tests verify the
  FSM via direct API calls; integration smoke just verifies
  "nothing broke." We can't smoke-test "does combat-induced
  position change work" because combat isn't wired to write the
  FSM yet. Mitigation: 4b's smoke is the real integration check.
  4a smoke = legacy combat still works + new FSM tests pass +
  server boots cleanly.
- **Predicate naming collision avoidance.** `c.IsGroundPosition()`
  currently exists on the `CombatPosition` enum
  (`c.CombatPosition.IsGroundPosition()`). The new Character API
  uses `c.IsOnFloor()` and `c.IsGroundGrapple()` to avoid name
  collision. 4b/4c sunset the enum's helper when call sites
  migrate.
- **GrappleData.Partner zero-value semantics.** All grapple
  transitions EXCEPT `TransitionToTurtle` require non-zero
  Partner. Spec-time decision: zero Partner returns an error from
  the transition method (does not panic). Tests cover the error
  case.
- **Symmetric grapple writes (4b+).** When 4b starts driving the
  FSM, grapple-entry needs to transition BOTH characters' Position
  machines simultaneously (mount-side and mounted-side). Each
  character has its own machine + Partner field; caller fires two
  transitions, one per side. No design change needed in 4a; just
  flag for 4b that the pair-write pattern is the convention.
- **Btree primitive subset risk.** The 10 primitives chosen for 4a
  may be insufficient for the content authors who arrive after 4b.
  Mitigation: easy to add more primitives in 4b/4c without
  breaking 4a. The 10 are a reasonable starter set covering the
  highest-frequency queries.
- **Transition table size.** ~75 valid edges across 14×14 matrix.
  Easy to mis-encode. Mitigation: 4a Behavior Matrix tests
  exercise each transition KIND (not every edge) — combined with
  invalid-transition tests, this catches common encoding errors.
  4b's writer migration will surface any specific edges that were
  missed (caller can't reach an expected state).

## Out-of-scope / future-followup candidates

- **Prone/Supine orientation extras.** If 4c/4d/4e want
  orientation-aware mechanics on the in-grapple states (e.g.,
  "mount controller is sitting upright vs lying on chest"), add
  per-state `Orientation` data field. Not needed in 4a.
- **Per-grapple-state grip / variant details** (ClinchGrip,
  ArmsIsolated, HooksIn, TrappedLeg, GuardVariant). Authored in
  4b/4c as wrapping structs around GrappleData.
- **N-vs-1 grappling** (mob B physically joins to make 2-on-1
  mount). YAGNI for now; significant complexity in the per-state
  data shape.
- **Cardio / fatigue effects** on control rolls. Existing stamina
  system handles tiredness; layering grapple-specific cardio is
  bonus.

## Resumption criteria (chunk 4a done when)

1. `internal/state/position/` package exists with state enum, per-
   state data, transition table, machine API, all Behavior Matrix
   tests green (PO-001 through PO-040ish).
2. `Character.Position` field initialized in `New()` + nil-guarded
   in `Validate()`. Tests verify both.
3. 18 Character predicates (state + rollup) work correctly.
4. 10 btree primitives registered and pass smoke (return Failure
   for non-matching states, Success when state matches — verified
   via direct FSM transitions in test setup).
5. `Position_Cascades.go` Life Dead observer wires up; integration
   test verifies cascade fires on Life Dead from each grapple
   state without breaking the chunk-2 pre-wire.
6. Existing combat flows unchanged: trip, bash, grapple, kick,
   stand, recovery, defense degradation, kick variant selector,
   spell knockdown — all behave identically to pre-4a.
7. Server boots cleanly. Full test suite green. Chunks 0-3
   regression tests pass.
8. Context.md updates: new `internal/state/position/context.md`
   (~200 lines, mirrors `life/` / `activity/` shape) + integration
   notes in `characters/context.md` and `hooks/context.md`.
9. `COMBAT_STATE_ROADMAP.md` chunk 4a row marked Done.
