# Combat State — Chunk 3: Activity Machine Design

> **Side quest from mob aliveness chunk 2.7.** Chunk 3 of the
> combat-state-machines redesign (master spec:
> `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`).
> Builds the Activity state machine on the chunk-0 framework.
> Consolidates `CastingState` + `CraftingState` pointer fields,
> formalizes per-activity interrupt rules, normalizes mob/player
> parity, and cleans up the salvage-hijacks-crafting-slot pattern.
>
> **Aliveness paused** for the duration of chunks 1-6.

## Goal

The current "is this character locked into a multi-round action"
surface is two scattered nullable pointer fields (`CastingState`,
`CraftingState`), one hijack (salvage shoving item UUID into
`CraftingState.MiscData`), and three asymmetries between mob and
player behavior. Interrupt rules are scattered across hooks
(`combat_shared_helpers.go` concentration break, `go.go` movement
cancel, `NewRound_MobRoundTick.go` mob-only combat cancel) with no
documented per-activity policy.

This chunk:
- **Replaces the two pointer fields with a single Activity state
  machine** on the framework chunk 0 built. Per-state data
  (CastingData, CraftingData, SalvagingData) follows the Life
  machine's per-state-data pattern.
- **Formalizes per-activity interrupt policy** in a documented
  table — combat entry, damage, movement, death, cancel command,
  and completion each get an explicit row per activity.
- **Normalizes mob/player parity** for the three current
  asymmetries (mob crafting cancels on combat / player doesn't;
  damage breaks cast / not craft; movement breaks player craft /
  not mob craft).
- **Cleans up the salvage hijack** by giving Salvaging its own
  state with its own data struct.
- **Adds mob `cancel` command parity + a `cancel_activity` btree
  action primitive** so behavior trees can tactically abort
  activities (e.g., flee instead of finishing offensive cast at
  low HP).
- **Ships staged via intent-driven TDD** — Behavior Matrix drives
  the RED phase as in chunks 0-2.

## Non-goals

- **New crafting / casting mechanics.** No new spells, recipes,
  craft stations, concentration knobs, or interrupt thresholds.
  Pure structural work.
- **Behavior tree content authoring.** The `cancel_activity`
  primitive ships in chunk 3; actual btree authoring that uses
  it (panic-flee on low HP, swap to heal mid-cast, etc.) is
  content / aliveness work that resumes after chunk 6.
- **Foraging / Tracking as states.** Both are one-shot commands
  today. Not adding states for ceremony's sake; documented as
  an intentional asymmetry.
- **Mob forager FSM unification.** The btree-driven
  `forager.ForagerState` (Foraging / Delivering / Recalling /
  Resting) stays in `internal/forager/` and
  `internal/behaviortree/actions_forager.go`. Different
  abstraction layer from `Character.Activity`. Documented as an
  intentional asymmetry.
- **Persistence across login.** Like the current `CastingState` /
  `CraftingState` (both `yaml:"-"`), Activity resets to `Free` on
  login. Documented invariant.
- **Cooldown system.** All special abilities share one cooldown
  timer today; modeling it as a state machine adds ceremony
  without earning the framework's veto/cascade/observer surface.
  Logged in the master spec as a Phase-7 candidate (helper, not
  machine).

## Architecture

`internal/state/activity/` package, mirroring `internal/state/life/`,
`internal/state/combatphase/`, `internal/state/awareness/`. Same
generics framework (`state.Machine[S]`), same per-state-data
pattern, same `OnCharacterCreated` init wiring.

### Files

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/activity/activity.go` | NEW | State enum, per-state data structs, Machine wrapper |
| `internal/state/activity/transitions.go` | NEW | Valid-transition table, trigger constants |
| `internal/state/activity/rules.go` | NEW | Transition method implementations |
| `internal/state/activity/activity_test.go` | NEW | Behavior Matrix tests (AC-001 through AC-040ish) |
| `internal/state/activity/context.md` | NEW | Package documentation, including the intentional-asymmetry rationale |
| `internal/characters/character.go` | MODIFY | Add `Activity *activity.Machine` field; delete `CastingState` + `CraftingState` pointer fields; update `IsCasting()` / `IsCrafting()` to query the machine; add `IsFree()` / `IsSalvaging()` / `IsActing()` predicates |
| `internal/characters/casting.go` | DELETE | `CastingState` struct moves to `activity.CastingData` |
| `internal/characters/crafting.go` | DELETE | `CraftingState` struct moves to `activity.CraftingData` |
| `internal/characters/validate.go` | MODIFY | Nil-guard init of `Activity` for YAML-loaded characters |
| `internal/hooks/Activity_Cascades.go` | NEW | Cross-machine cascade wiring (Life Dead → Free, Combat Phase Engaging → Free for craft/salvage, damage → cancel per policy, movement → cancel for craft/salvage) |
| `internal/hooks/Life_Cascades.go` | MODIFY | Remove the Activity pre-wire (lines 43-44 — direct `CastingState = nil` / `CraftingState = nil`); the new Activity cascade observer subscribes to Life Dead directly |
| `internal/usercommands/skill.cast.go` | MODIFY | Use `Activity.TransitionToCasting` + `CastingData`; replace `c.IsCrafting()` early-exit with `c.IsActing()` |
| `internal/usercommands/craft.go` | MODIFY | Use `Activity.TransitionToCrafting` + `CraftingData` |
| `internal/usercommands/salvage.go` | MODIFY | Use `Activity.TransitionToSalvaging` + `SalvagingData`; remove `CraftingState.RecipeId = "salvage:<itemid>"` hijack and `MiscData["salvage_item_uuid"]` storage |
| `internal/usercommands/cancel.go` | MODIFY | Generalize: handle any non-Free activity, dispatch on current state for refund logic |
| `internal/usercommands/go.go` | MODIFY | Replace direct `c.CraftingState = nil` with `Activity.TransitionToFree(TriggerMovementInterrupt)` |
| `internal/mobcommands/cancel.go` | NEW | Mob parity for `cancel` (mirrors player) |
| `internal/mobcommands/cast.go` | MODIFY | Use `Activity.TransitionToCasting` |
| `internal/mobcommands/craft.go` | MODIFY | Use `Activity.TransitionToCrafting` |
| `internal/mobcommands/salvage.go` | MODIFY | Use `Activity.TransitionToSalvaging` with a 1-round `SalvagingData` for parity of data shape; keep the single-tick resolution path (no per-round progress display for mobs). Multi-round mob salvage with per-round messaging is deferred to a future chunk if a use case arises. |
| `internal/behaviortree/actions.go` | MODIFY | Add `cancel_activity` action primitive |
| `internal/hooks/combat_shared_helpers.go` | MODIFY | `checkConcentrationBreak` rewires to call `Activity.TransitionToFree(TriggerConcentrationBreak)` on roll failure |
| `internal/hooks/NewRound_DoCombat_*.go` | MODIFY | Damage application path extended to fire activity-cancel cascade for Crafting + Salvaging (hard cancel, no roll) |
| `internal/hooks/NewRound_UserRoundTick.go` | MODIFY | `resolveSalvage` consumes `SalvagingData` directly; round-tick increments use `Activity.CraftingData()` / `Activity.SalvagingData()` |
| `internal/hooks/NewRound_MobRoundTick.go` | MODIFY | Same as user round-tick; remove the mob-only combat-cancel block (lines ~404) since the cascade handles it now |
| `internal/characters/context.md` | MODIFY | Document `Activity` field + predicates |
| `internal/hooks/context.md` | MODIFY | Document Activity cascade observers + interrupt policy wiring |
| `internal/forager/context.md` | MODIFY (if exists) | Document why forager FSM stays separate from Activity |
| `internal/actions/command_readiness.go` | MODIFY | Special-moves gate switches from `IsCrafting()` to `IsActing()` — verify each call site's intent |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 3 Done |

## States + per-state data

```go
type State int

const (
    Free State = iota
    Casting
    Crafting
    Salvaging
)

type FreeData struct{} // empty

// Replaces internal/characters/CastingState
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

// Replaces internal/characters/CraftingState
type CraftingData struct {
    Reason         state.TransitionReason
    RecipeId       string
    RoundsTotal    int
    RoundsComplete int
    TargetSlot     string
}

// New — cleans up salvage's current crafting-slot hijack
type SalvagingData struct {
    Reason         state.TransitionReason
    ItemUuid       string  // item being broken down
    RoundsTotal    int
    RoundsComplete int
    SpoiledPotion  bool    // current MiscData flag
}
```

Each data struct field shape preserves the existing data the
current `CastingState` / `CraftingState` carry, so per-tick
consumers (round ticks, spell resolution, recipe completion) see
the same data through a new accessor.

## Transition table

```
Free       ──▶ Casting, Crafting, Salvaging
Casting    ──▶ Free
Crafting   ──▶ Free
Salvaging  ──▶ Free
```

Star topology. No direct transitions between active states
(cancel-then-start enforces serialization). Cross-activity-start
veto is implicit in the table: trying to go `Crafting → Casting`
fails because the table doesn't allow it; the cast command's
pre-flight `if !c.IsFree() { send "you can't X while Y"; return }`
catches it before the transition attempt.

## Trigger constants

```go
const (
    // Free → active
    TriggerCastBegin    = "cast_begin"
    TriggerCraftBegin   = "craft_begin"
    TriggerSalvageBegin = "salvage_begin"

    // active → Free, success
    TriggerCastComplete    = "cast_complete"
    TriggerCraftComplete   = "craft_complete"
    TriggerSalvageComplete = "salvage_complete"

    // active → Free, user-initiated
    TriggerCastCancel    = "cast_cancel"
    TriggerCraftCancel   = "craft_cancel"
    TriggerSalvageCancel = "salvage_cancel"

    // active → Free, externally induced
    TriggerConcentrationBreak = "concentration_break"   // Casting only
    TriggerCombatInterrupt    = "combat_interrupt"      // Crafting / Salvaging
    TriggerMovementInterrupt  = "movement_interrupt"    // Crafting / Salvaging
    TriggerDamageInterrupt    = "damage_interrupt"      // Crafting / Salvaging (hard cancel, no roll)
    TriggerDeath              = "death"                 // cascade from Life
)
```

## Machine API

```go
func (m *Machine) TransitionToCasting(d CastingData, r state.TransitionReason) error
func (m *Machine) TransitionToCrafting(d CraftingData, r state.TransitionReason) error
func (m *Machine) TransitionToSalvaging(d SalvagingData, r state.TransitionReason) error
func (m *Machine) TransitionToFree(r state.TransitionReason) error
func (m *Machine) CastingData() (CastingData, bool)
func (m *Machine) CraftingData() (CraftingData, bool)
func (m *Machine) SalvagingData() (SalvagingData, bool)
func (m *Machine) State() State
func (m *Machine) Inner() *state.Machine[State]
```

## Character predicates

```go
func (c *Character) IsFree() bool       // new unified "no activity locked in"
func (c *Character) IsCasting() bool    // was c.CastingState != nil
func (c *Character) IsCrafting() bool   // was c.CraftingState != nil
func (c *Character) IsSalvaging() bool  // was c.CraftingState != nil && c.MiscData["salvage_item_uuid"] != ""
func (c *Character) IsActing() bool     // new canonical "any non-Free activity" gate
```

`IsActing()` replaces `IsCrafting()` as the special-moves gate in
`actions/command_readiness.go`. The 30+ call sites that read
`IsCrafting()` are audited during chunk 3 plan — some genuinely
want "is crafting specifically" (the craft command's own
re-entrancy check), most want "is busy with anything."

## Per-activity interrupt policy

| Interrupt source | Casting | Crafting | Salvaging |
|---|---|---|---|
| Combat Phase Idle → Engaging (self attacks) | no effect | → Free (cancel) | → Free (cancel) |
| Combat Phase Idle → Engaging (mob attacks self) | no effect | no effect — being targeted ≠ entering combat | no effect |
| Damage taken | concentration break check (willpower roll, existing rule) | → Free (hard cancel, no roll) | → Free (hard cancel, no roll) |
| Movement (room change) | no effect | → Free (cancel; existing for player, new for mob) | → Free (cancel) |
| `cancel` command | → Free (refund 50% unspent conviction, existing) | → Free (no refund) | → Free (no refund) |
| Life Alive → Dead | → Free (via Life cascade) | → Free | → Free |
| Activity completion | → Free + spell resolves | → Free + recipe yields | → Free + materials roll |

## Cross-machine interactions

**Activity-side observers added in chunk 3** (subscribe to other
machines' transitions, fire Activity transitions in response):

- `Life: Alive → Dead` → cascade `Activity → Free`. Repoints the
  chunk-2 pre-wire in `Life_Cascades.go` (currently does
  `c.CastingState = nil` / `c.CraftingState = nil` directly)
  onto a proper Activity-side observer. The chunk-2 pre-wire
  block is deleted as part of this chunk's sunset.
- `Combat Phase: Idle → Engaging` (self-initiated, not "being
  attacked") → cascade Activity → Free if current state is
  Crafting or Salvaging. Casting is exempted per the policy
  table.
- Damage application (a hook event, not a machine transition) →
  the existing `checkConcentrationBreak` in
  `combat_shared_helpers.go` is rewired to call
  `Activity.TransitionToFree(TriggerConcentrationBreak)` on roll
  failure. Extended to also fire
  `Activity.TransitionToFree(TriggerDamageInterrupt)` for
  Crafting + Salvaging unconditionally (no roll — hard cancel).
- Movement (room change) → existing player-side code in `go.go`
  already cancels crafting; rewired to call
  `Activity.TransitionToFree(TriggerMovementInterrupt)`. Mob
  movement gets the same hook for parity.

**Activity → other machine cascades:** intentionally none in
chunk 3. Activity is a leaf consumer. If Position later wants to
forbid Crafting while Clinched, that's a Position-side veto, not
an Activity cascade.

## Mob / player parity normalization

Three current asymmetries resolved by routing both actors through
the same observers:

| Asymmetry today | Parity after chunk 3 |
|---|---|
| Mob crafting auto-cancels on combat entry; player crafting doesn't | Both cancel via the same observer |
| Player craft cancels on movement; mob craft doesn't (mobs don't `go`) | Movement hook covers both actors |
| Damage breaks cast only; craft/salvage damage-resilient | All three respond to damage per the policy table (cast = roll, craft/salvage = hard cancel) |

The mob-only combat-cancel block in
`NewRound_MobRoundTick.go:tickMobCrafting()` is deleted; the
cascade observer covers it generically.

## `cancel` command unification

Today: `internal/usercommands/cancel.go` clears `CastingState` only.

After chunk 3: `cancel` works for any non-Free Activity. The
handler dispatches on `c.Activity.State()`:

- `Casting → Free` with `TriggerCastCancel`. Refund 50% of unspent
  conviction (existing behavior preserved via the data struct).
- `Crafting → Free` with `TriggerCraftCancel`. No refund (no
  materials consumed until completion — existing).
- `Salvaging → Free` with `TriggerSalvageCancel`. No refund (item
  not yet consumed — existing).
- `Free` — "You aren't doing anything to cancel." (existing
  error message preserved.)

`internal/mobcommands/cancel.go` is new — mirrors player. Same
dispatch logic, same refund rules. The btree action
`cancel_activity` wraps this command.

## Mob `cancel` + btree primitive

`internal/behaviortree/actions.go` gains a new action:

```go
// cancel_activity: aborts the mob's current Activity, if any.
// Returns success if anything was canceled, failure if Activity
// was already Free.
func actionCancelActivity(mob *mobs.Mob, ctx EvalContext) Status {
    if mob.Character.Activity.State() == activity.Free {
        return Failure
    }
    // Dispatch to the right trigger based on current state.
    switch mob.Character.Activity.State() {
    case activity.Casting:
        mob.Character.Activity.TransitionToFree(
            state.TransitionReason{Trigger: activity.TriggerCastCancel})
    case activity.Crafting:
        mob.Character.Activity.TransitionToFree(
            state.TransitionReason{Trigger: activity.TriggerCraftCancel})
    case activity.Salvaging:
        mob.Character.Activity.TransitionToFree(
            state.TransitionReason{Trigger: activity.TriggerSalvageCancel})
    }
    return Success
}
```

Behavior tree YAMLs gain access to:
```yaml
- selector:
    children:
      - sequence:
          children:
            - is_low_hp: 0.10
            - is_casting:
            - cancel_activity:
            - flee:
      - default_combat
```

(Actual behavior tree authoring that uses this is out of chunk-3
scope — content / aliveness work after chunk 6 lands.)

## Notes on intentional asymmetries

These become prose in `internal/state/activity/context.md` (and
referenced from `internal/forager/context.md`) so future
contributors understand the negative space rather than treating
gaps as oversights.

1. **No `Foraging` or `Tracking` state.** Both are one-shot
   commands for players (no multi-round progress, no interrupt
   surface, no observer would subscribe). Adding states for
   them would be ceremony without payoff. If either becomes
   multi-round in the future, adding the state then is cheap
   (one transition table entry + per-state data struct + a
   handful of test rows).

2. **Mob forager `forager.ForagerState` left in btree, not
   folded into Activity.** The two FSMs answer different
   questions:
   - `Character.Activity` = "what game-mechanic-level action is
     this character locked into right now (interruptible by the
     rules in the policy table)"
   - `forager.ForagerState` = "what AI behavior phase is this
     mob in (Gathering / Delivering / Recalling / Resting loop)"
   - The forager FSM is an AI orchestration tool; Activity is a
     character-state primitive. Mixing them would muddy both.
   - The forager FSM stays in `internal/forager/` and
     `internal/behaviortree/actions_forager.go`. Mob foragers
     remain `Activity = Free` throughout the forage loop.

3. **No `IsForaging()` / `IsTracking()` predicates on Character.**
   Direct consequence of (1). If a caller needs to know "is
   this mob currently in the foraging AI phase," it should query
   the btree's `forager.ForagerState`, not `Character.Activity`.

4. **Salvage gets its own state despite being structurally
   similar to crafting.** Cleans up the current hijack (where
   salvage shoves item UUID into `CraftingState.MiscData`).
   Future divergence is likely (salvage stations, salvage-only
   skill scaling, different damage-interrupt rules).

## Behavior Matrix preview (drafted in plan, completed in tests)

~30-40 rows total, grouped:

- **Basic transitions** (~12 rows): each Free → active and
  active → Free path with its trigger. AC-001 through AC-012.
- **Per-activity interrupt policy** (~12 rows): one row per cell
  of the policy table — Cast + damage = concentration roll, Cast
  + completion = resolve spell, Craft + combat entry = cancel,
  etc. AC-013 through AC-024.
- **Cross-activity start veto** (~4 rows): `cast` while crafting
  fails with the existing error message, etc. AC-025 through
  AC-028.
- **Mob/player parity** (~6 rows): mob craft + combat entry now
  cancels (was already true — regression check), mob damage
  during craft cancels (new), player craft on damage cancels
  (new), etc. AC-029 through AC-034.
- **Cascade verification** (~4 rows): Life Dead cascades
  Activity Free; Movement cascades Crafting/Salvaging Free;
  Combat Phase Engaging cascades Crafting/Salvaging Free.
  AC-035 through AC-038.

Authored in full in the chunk-3 plan PR.

## Sunset list (deleted at end of chunk 3)

- `Character.CastingState *characters.CastingState` field
- `Character.CraftingState *characters.CraftingState` field
- `internal/characters/casting.go` `CastingState` struct
  (replaced by `activity.CastingData`)
- `internal/characters/crafting.go` `CraftingState` struct
  (replaced by `activity.CraftingData`)
- `Character.MiscData["salvage_item_uuid"]` + spoiled-potion
  flag (replaced by `activity.SalvagingData` fields)
- `usercommands/salvage.go` recipe-id hijack pattern
  (`"salvage:<itemid>"` in `CraftingState.RecipeId`)
- All ad-hoc `c.CastingState != nil` / `c.CraftingState != nil`
  checks — replaced by `IsCasting()` / `IsCrafting()` /
  `IsSalvaging()` / `IsFree()` / `IsActing()` predicates
- `Character.IsCasting()` and `Character.IsCrafting()` method
  bodies — reimplemented as Activity-machine queries (signatures
  preserved for backward compatibility within the codebase)
- `Life_Cascades.go` Activity pre-wire (direct `CastingState =
  nil` / `CraftingState = nil` at lines 43-44) — replaced by
  Activity-side observer subscribed to Life Dead
- `NewRound_MobRoundTick.go:tickMobCrafting` mob-only
  combat-cancel block — covered by generic cascade observer

## Persistence

Activity is fully ephemeral. Like the current `CastingState` /
`CraftingState` (both `yaml:"-"`), the Activity machine resets
to `Free` on every login. Documented invariant in
`internal/state/activity/context.md`.

## Open questions / risks

- **Damage-during-craft cancel is a behavior change for players.**
  Today a player crafting in a contested zone can complete a
  5-round craft while getting hit. Per the new policy, the
  first damage cancels the craft. Could be a UX regression for
  some workflows. Mitigation: a future "focused crafting" buff
  or station bonus could grant damage resistance for crafters,
  but that's out of scope. Document in patch notes.
- **Combat-entry cancel for player craft is also new.** Today a
  player can attack mid-craft (basic `attack`, not special
  moves). Per the new policy, attacking cancels the craft. UX
  regression mitigated by "what did you expect to happen"
  reasoning. Document in patch notes.
- **Btree primitive `cancel_activity` is one of the first
  behavior primitives that operates on the new state machines
  as a write** (existing `is_casting` / `is_crafting` btree
  primitives are queries). Sets a precedent for chunks 4-6's
  btree primitives.
- **`IsActing()` becomes the new canonical busy gate** (was
  `IsCrafting()`). The 30+ call sites that read `IsCrafting()`
  need to be audited — some want "is crafting specifically"
  (the craft command's own re-entrancy check), most want "is
  busy with anything." Audit during chunk-3 plan.
- **No mob `cancel` command in pre-existing mobcommands tests.**
  Adding `mobcommands/cancel.go` requires fresh test coverage;
  no prior tests to regress against.

## Resumption criteria (chunk 3 done when)

1. `internal/state/activity/` package exists with state enum,
   per-state data, transition table, machine API, all
   Behavior Matrix tests green (AC-001 through AC-040ish).
2. `Character.CastingState` and `Character.CraftingState`
   fields deleted. All call sites migrated to Activity machine
   queries.
3. `internal/characters/casting.go` and
   `internal/characters/crafting.go` files deleted.
4. Per-activity interrupt policy live: damage cancels craft +
   salvage, combat entry cancels craft + salvage, movement
   cancels craft + salvage (both actors).
5. Salvage hijack cleaned up — no `MiscData` salvage keys, no
   `"salvage:<itemid>"` recipe ID convention.
6. `mobcommands/cancel.go` + `cancel_activity` btree action
   land with smoke test ("btree calling `cancel_activity`
   actually transitions Activity to Free").
7. Chunks 0/1/2 regression tests pass.
8. Context.md docs (activity package + characters + hooks +
   forager) updated, including the intentional-asymmetry
   rationale.
9. `COMBAT_STATE_ROADMAP.md` chunk 3 row marked Done.
