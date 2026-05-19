# activity — Package Documentation

## Overview

The `internal/state/activity` package is the fourth consumer of the
`internal/state` framework, after `combatphase`, `awareness`, and `life`.
It defines the **Activity state machine**, replacing:

- `Character.CastingState *characters.CastingState` — nullable pointer field
- `Character.CraftingState *characters.CraftingState` — nullable pointer field
- The salvage hijack: `CraftingState.RecipeId = "salvage:<itemid>"` and
  `CraftingState.MiscData["salvage_item_uuid"]` shoved into the crafting slot

Before chunk 3, "is this character locked into a multi-round action" was
expressed as two scattered nullable pointer fields, with interrupt rules
scattered across hooks (`combat_shared_helpers.go` concentration break,
`go.go` movement cancel, `NewRound_MobRoundTick.go` mob-only combat cancel)
and no documented per-activity policy. Chunk 3 consolidates all of this into
one machine with explicit per-state data, a documented interrupt table, and
normalized mob/player parity.

Activity has four states:

| State | Meaning |
|-------|---------|
| `Free` | No multi-round activity in flight; character can take any action. |
| `Casting` | Fold-casting in progress; accumulating folds toward a spell. |
| `Crafting` | Crafting recipe in progress; accumulating rounds toward completion. |
| `Salvaging` | Salvage breakdown in progress; rolling for material recovery. |

The machine is mob/player symmetric: both player and mob `Character`
instances carry a `*activity.Machine` field. All interrupt rules (damage,
movement, combat entry, death, `cancel` command) route through the same
machine API for both actors.

---

## Key Components

### Core Files

- **activity.go** — `State` enum, `FreeData` / `CastingData` /
  `CraftingData` / `SalvagingData` structs, `Machine` wrapper with
  predicate and data accessor methods, machine registry.
- **transitions.go** — `validTransitions` star-topology table and all
  Trigger string constants.
- **rules.go** — `TransitionToCasting`, `TransitionToCrafting`,
  `TransitionToSalvaging`, `TransitionToFree`, `ForceFree`, and
  `Advance*` helpers.
- **activity_test.go** — Behavior Matrix tests AC-001 through AC-038.

---

## State Diagram

```
        cast_begin / craft_begin / salvage_begin
              +------+------+
              |      |      |
           Casting Crafting Salvaging
              |      |      |
              +------+------+
                     |
                   Free
```

Star topology: every active state returns to `Free`; `Free` can become
any active state. There are no direct active-to-active transitions.
Cross-activity starts are blocked at the call-site layer — cast/craft/
salvage commands check `c.IsFree()` before attempting a transition.

---

## Per-State Data

### FreeData

```go
type FreeData struct{}
```

Empty — the default state has no metadata.

### CastingData

Replaces `internal/characters/CastingState`. Field shape preserved so
existing per-tick consumers swap the accessor without other changes.

```go
type CastingData struct {
    Reason               state.TransitionReason
    SpellId              string
    FoldsNeeded          int
    FoldsAccumulated     int
    FoldsPerRound        int
    TotalConvictionCost  int
    ConvictionSpent      int
    TargetUserIds        []int
    TargetMobInstanceIds []int
    SpellRest            string
}
```

### CraftingData

Replaces `internal/characters/CraftingState`.

```go
type CraftingData struct {
    Reason         state.TransitionReason
    RecipeId       string
    RoundsTotal    int
    RoundsComplete int
    TargetSlot     string
}
```

### SalvagingData

New — gives salvage its own state instead of hijacking the crafting slot.

```go
type SalvagingData struct {
    Reason         state.TransitionReason
    ItemUuid       string  // item being broken down
    RoundsTotal    int
    RoundsComplete int
    SpoiledPotion  bool    // replaces the old MiscData flag
}
```

---

## Trigger Constants

Defined in `transitions.go`. Always use these constants instead of
inline string literals for stable identifiers across the codebase.

| Constant | Value | Purpose |
|----------|-------|---------|
| `TriggerCastBegin` | `"cast_begin"` | Free → Casting (player or mob starts a spell) |
| `TriggerCraftBegin` | `"craft_begin"` | Free → Crafting (player or mob starts a recipe) |
| `TriggerSalvageBegin` | `"salvage_begin"` | Free → Salvaging (player or mob starts a breakdown) |
| `TriggerCastComplete` | `"cast_complete"` | Casting → Free, spell resolved successfully |
| `TriggerCraftComplete` | `"craft_complete"` | Crafting → Free, recipe yields |
| `TriggerSalvageComplete` | `"salvage_complete"` | Salvaging → Free, materials roll fired |
| `TriggerCastCancel` | `"cast_cancel"` | Casting → Free, player/mob typed `cancel` |
| `TriggerCraftCancel` | `"craft_cancel"` | Crafting → Free, player/mob typed `cancel` |
| `TriggerSalvageCancel` | `"salvage_cancel"` | Salvaging → Free, player/mob typed `cancel` |
| `TriggerConcentrationBreak` | `"concentration_break"` | Casting → Free, failed Willpower roll on damage |
| `TriggerCombatInterrupt` | `"combat_interrupt"` | Crafting/Salvaging → Free, character entered combat |
| `TriggerMovementInterrupt` | `"movement_interrupt"` | Crafting/Salvaging → Free, character moved rooms |
| `TriggerDamageInterrupt` | `"damage_interrupt"` | Crafting/Salvaging → Free, damage received (hard cancel) |
| `TriggerDeath` | `"death"` | Any active → Free, Life Alive→Dead cascade |

---

## Key Functions / Machine API

### TransitionToCasting

```go
func (m *Machine) TransitionToCasting(d CastingData,
    r state.TransitionReason) error
```

Moves `Free → Casting` and stores the casting context. Caller is
responsible for pre-checks (character is free, has enough conviction,
not prone/disabled).

### TransitionToCrafting

```go
func (m *Machine) TransitionToCrafting(d CraftingData,
    r state.TransitionReason) error
```

Moves `Free → Crafting` and stores the crafting context.

### TransitionToSalvaging

```go
func (m *Machine) TransitionToSalvaging(d SalvagingData,
    r state.TransitionReason) error
```

Moves `Free → Salvaging` and stores the salvaging context.

### TransitionToFree

```go
func (m *Machine) TransitionToFree(r state.TransitionReason) error
```

Returns the machine to `Free`, clearing all per-state data. All cancel,
complete, and interrupt paths route through here.

### ForceFree

```go
func (m *Machine) ForceFree(r state.TransitionReason)
```

Idempotent transition to `Free` from any state. Used by admin commands
and emergency cleanup. Does not return an error.

### Advance helpers

```go
func (m *Machine) AdvanceCastingFolds(folds int, convictionCost int) (CastingData, bool)
func (m *Machine) AdvanceCraftingRound() (CraftingData, bool)
func (m *Machine) AdvanceSalvagingRound() (SalvagingData, bool)
```

Increment progress counters on the current activity and return the
updated data plus a `complete` flag. Caller is responsible for calling
`TransitionToFree` and resolving the activity when `complete` is true.

### Data accessors

```go
func (m *Machine) CastingData() (CastingData, bool)
func (m *Machine) CraftingData() (CraftingData, bool)
func (m *Machine) SalvagingData() (SalvagingData, bool)
```

Return the activity context for the current state. Second return value
is `false` if the machine is not in the matching state.

### Inner

```go
func (m *Machine) Inner() *state.Machine[State]
```

Returns the underlying `state.Machine[State]`. Used by `rules.go` and
hooks to register `AfterTransition` observers. Not part of the stable
caller API.

---

## Character API

Defined in `internal/characters/character.go`. All predicates delegate
to the Activity machine; none read legacy pointer fields.

```go
func (c *Character) IsFree() bool
    // true when Activity == Free (no activity in flight)

func (c *Character) IsCasting() bool
    // true when Activity == Casting

func (c *Character) IsCrafting() bool
    // true when Activity == Crafting

func (c *Character) IsSalvaging() bool
    // true when Activity == Salvaging

func (c *Character) IsActing() bool
    // true when Activity != Free (any non-Free state)
    // canonical "busy" gate — replaces the old IsCrafting() gate
    // at special-moves and command-readiness call sites
```

`IsActing()` is the preferred gate for "should this action be blocked
because the character is busy?" checks. Use the specific predicates
only when you genuinely need to distinguish which activity is running
(e.g., the craft command's own re-entrancy check, or the btree
`is_casting` condition).

---

## Cascade Integration

`Activity_Cascades.go` in `internal/hooks/` wires the Activity machine
into cross-machine cascades via `characters.OnCharacterCreated`.

### activity_life_dead observer

Subscribes to the Life machine's `AfterTransition`. When Life transitions
`Alive → Dead`, the observer calls
`Activity.TransitionToFree(TriggerDeath)` if any activity is in flight.
This repoints the chunk-2 pre-wire in `Life_Cascades.go` (which used to
nil `CastingState` / `CraftingState` directly) onto a proper
Activity-side observer subscribed to the Life machine.

**Casting exempt note:** Casting is canceled by the `activity_life_dead`
observer the same as Crafting and Salvaging — in the dead-state cascade,
all activities clear regardless of type.

### Combat-entry cancellation

Crafting and Salvaging are interrupted when the character enters combat
(self-initiated `Idle → Engaging`). This is implemented as a veto in
`CombatPhase_Vetoes.go` (`RegisterActivityCheck` reads `c.IsActing()`),
which prevents `TransitionToEngaging` from succeeding while any activity
is active. Casting is exempt from this veto per the interrupt policy
table (see below).

A separate `AfterTransition` cascade for combat-entry is unreachable and
was removed; the veto approach is the correct integration point here.

### Movement interrupts (call-site wiring)

`internal/usercommands/go.go` calls `c.Activity.TransitionToFree(
TriggerMovementInterrupt)` when the character moves between rooms and is
currently Crafting or Salvaging. Mobs receive the same call for parity.
Casting is not interrupted by movement.

### Damage interrupts (call-site wiring)

`cancelCraftOrSalvageOnDamage` in `internal/hooks/combat_shared_helpers.go`
fires `TriggerDamageInterrupt` unconditionally (hard cancel, no roll) at
every damage application site when the target is Crafting or Salvaging.
Casting takes a different path: `checkConcentrationBreak` rolls Willpower
and fires `TriggerConcentrationBreak` only on failure.

### Completion wiring (call-site, per-tick)

Per-tick consumers fire the completion triggers after a successful
`Advance*` call returns `complete = true`:

- Casting: `processFoldRound` in `NewRound_UserRoundTick.go` fires
  `TriggerCastComplete` and calls `resolveSpell()`.
- Crafting: inline craft-tick block in `NewRound_UserRoundTick.go`
  (player) / `NewRound_MobRoundTick.go` (mob) fires `TriggerCraftComplete`
  and delivers recipe output.
- Salvaging: inline salvage-tick block fires `TriggerSalvageComplete`
  and runs the material recovery roll.

---

## Per-Activity Interrupt Policy

| Interrupt source | Casting | Crafting | Salvaging |
|---|---|---|---|
| Combat Phase Idle → Engaging (self attacks) | no effect (casting exempt) | → Free (cancel) | → Free (cancel) |
| Combat Phase Idle → Engaging (mob attacks self) | no effect | no effect | no effect |
| Damage taken | concentration break check (Willpower roll) | → Free (hard cancel, no roll) | → Free (hard cancel, no roll) |
| Movement (room change) | no effect | → Free (cancel) | → Free (cancel) |
| `cancel` command | → Free (50% conviction refund) | → Free (no refund) | → Free (no refund) |
| Life Alive → Dead | → Free | → Free | → Free |
| Activity completion | → Free + spell resolves | → Free + recipe yields | → Free + materials roll |

---

## Notes on Intentional Asymmetries

These decisions define the negative space of the Activity machine.
Future contributors should read these before adding states or predicates.

1. **No `Foraging` or `Tracking` state.** Both are one-shot commands for
   players today — no multi-round progress accumulation, no interrupt
   surface, no observer would subscribe to them. Adding states purely
   for structural ceremony would yield no payoff. If either becomes
   multi-round in the future, adding the state is cheap: one transition
   table entry, one per-state data struct, a handful of test rows.

2. **Mob forager `forager.ForagerState` left in `internal/forager/`.**
   The two machines answer different questions. `Character.Activity`
   answers "what game-mechanic-level action is this character locked into
   right now, interruptible by the rules in the policy table?"
   `forager.ForagerState` answers "what AI behavior phase is this mob
   in (Gathering / Delivering / Recalling / Resting loop)?" The forager
   FSM is an AI orchestration tool; Activity is a character-state
   primitive. Mixing them would muddy both. Mob foragers remain
   `Activity = Free` throughout the entire forage loop. Cross-reference
   `internal/forager/context.md` (if it exists) for the forager-side
   boundary note.

3. **No `IsForaging()` / `IsTracking()` predicates on Character.**
   Direct consequence of (1). If a caller needs to know whether a mob
   is in the foraging AI phase, it should query `forager.ForagerState`
   via the btree, not `Character.Activity`.

4. **Salvage gets its own state despite structural similarity to
   crafting.** This cleans up the hijack pattern where salvage shoved
   item UUID into `CraftingState.MiscData` and set
   `RecipeId = "salvage:<itemid>"`. Future divergence between crafting
   and salvaging is likely (salvage stations, salvage-only skill scaling,
   different damage-interrupt rules in future chunks).

---

## Persistence

Activity is fully ephemeral. Like the pre-chunk-3 `CastingState` and
`CraftingState` fields (both tagged `yaml:"-"`), the Activity machine
resets to `Free` on every login. There is no mid-session save of
in-flight activities. `Character.Activity` is initialized in `New()` and
nil-guarded in `Validate()` for characters loaded from YAML.

---

## Testing Notes

### activity_test.go — Behavior Matrix

Tests follow the AC-NNN naming scheme from the chunk-3 spec. Each test
exercises one cell of the state × trigger × policy matrix.

| Range | Area |
|-------|------|
| AC-001 – AC-012 | Basic transitions (Free ↔ each active state, data accessors) |
| AC-013 – AC-024 | Per-activity interrupt policy (damage, combat entry, movement, cancel, completion) |
| AC-025 – AC-028 | Cross-activity start veto (active-to-active attempts fail) |
| AC-029 – AC-033 | Mob/player parity (mob craft cancels on combat entry + damage, etc.) |
| AC-034 | Btree `cancel_activity` fires `TransitionToFree` |
| AC-035 – AC-038 | Cascade verification (Life Dead, movement, combat phase engaging) |

At the unit level (no full server / hook wiring): **16 PASS, 22 SKIP**.
The 22 skipped tests require full `Character` + hook wiring
(interrupt policy, mob parity, cascade verification); they are
integration-level tests that pass in the full server smoke test.

---

## Sunset Notes

## Position-based casting disruption (chunks 4e + 4f)

The Activity machine's `Casting` state is disrupted by two independent paths:

1. **Damage-path** (`checkConcentrationBreak` in
   `internal/hooks/combat_shared_helpers.go`): fires when the caster takes
   damage. Rolls `characters.CalcConcentrationChance(Wil, damagePct)`;
   on failure fires `TriggerConcentrationBreak`. Unchanged since chunk 3.

2. **Position-path** (`processFoldRound`, chunk 4f): at the start of every
   fold round, if the caster is not `Standing`, looks up
   `position.PositionDisruptionDmgEquiv(pos, role)` from
   `internal/state/position/disruption.go`, feeds it through the same
   `CalcConcentrationChance` curve, and rolls. Standing returns 0 (check
   skipped). Both paths can break a single cast in the same round
   (layered disruption). See `internal/hooks/context.md` for the full
   walkthrough and `internal/state/position/context.md` for the per-position
   table.

The Activity machine itself has no position awareness — `processFoldRound`
drives the break by calling `clearCastingActivity(char,
TriggerConcentrationBreak)`, which calls `Activity.TransitionToFree`.

---

## Sunset Notes

Chunk 3 deleted the following. Do not re-add these patterns:

- `Character.CastingState *characters.CastingState` field
- `Character.CraftingState *characters.CraftingState` field
- `internal/characters/casting.go` — `CastingState` struct
  (data moved to `activity.CastingData`)
- `internal/characters/crafting.go` — `CraftingState` struct
  (data moved to `activity.CraftingData`)
- `CraftingState.MiscData["salvage_item_uuid"]` key
- `CraftingState.RecipeId = "salvage:<itemid>"` convention
- All ad-hoc `c.CastingState != nil` / `c.CraftingState != nil` checks
  — replaced by `IsCasting()` / `IsCrafting()` / `IsSalvaging()` /
  `IsFree()` / `IsActing()` predicates
- `Life_Cascades.go` Activity pre-wire (direct `CastingState = nil` /
  `CraftingState = nil`) — replaced by `Activity_Cascades.go`
  `activity_life_dead` observer subscribed to the Life machine
- `NewRound_MobRoundTick.go:tickMobCrafting` mob-only combat-cancel
  block — covered by the activity veto in `CombatPhase_Vetoes.go`
  (`RegisterActivityCheck` rejects combat entry when `IsCrafting()` or
  `IsSalvaging()`; casting is exempt per the per-activity policy)
