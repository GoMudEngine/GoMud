# Mob Aliveness 2.9 — Mob Forage + Salvage

> **Phase 2 tactical (ninth chunk).** Lift forage and salvage into
> the `internal/actions/` actor pattern. Add btree primitives
> `try_forage` and `try_salvage`. Refactor the existing `forager_step`
> state machine to dissolve only the Foraging-state per-tick loop
> into YAML, keeping the higher-level state transitions in Go. New
> shared `forager` archetype replaces the three per-mob behavior
> YAMLs for Tova (371), Halix (372), Kessa (373).
>
> Originally scoped Size S as "promote forage from routine-only
> behavior to a callable verb." Expanded to Size M during
> brainstorming when the user requested parallel salvage handling
> plus full archetype migration of the three existing foragers.

## Goal

Routine foragers stop using a parallel Go-state-machine code path
for their per-tick forage roll. The shared `actions.Forage` and
`actions.Salvage` are now the unified entry points — same code
path whether a player types `forage`, a mob runs the btree
primitive, or the forager state machine fires it indirectly.
Future strategic NPCs (Phase 4) can compose `try_forage` and
`try_salvage` in archetype trees without re-implementing the
gathering logic.

The three existing forager mobs (Tova, Halix, Kessa) migrate from
per-mob behavior YAMLs to a shared `forager` archetype that
combines the dissolved per-tick loop in YAML with the preserved
multi-state daily cycle in Go.

## Architectural musts

Brainstorming refined the framing:

1. **Verb lifts follow the chunk-2.7/2.8 actor pattern.** Each
   exposes a single entry point `<Verb>(actor Actor, opts) <Verb>Result`.
   Player wrappers thin to ~25 lines; mob wrappers thin similarly.

2. **Mob-actor side effects are silent.** `MobActor.SendText` is
   a no-op (existing convention). Player wrappers retain all
   existing flavor text.

3. **Salvage lift covers the single-tick core only.** Player
   `usercommands/salvage.go` keeps its multi-round `CraftingState`
   scheduling; each tick of the scheduler calls `actions.Salvage`
   for the actual material-recovery roll. Mob path is direct
   single-tick. Both unified on the same per-tick logic.

4. **Forager state machine stays in Go for non-Foraging states.**
   The five other states (Resting, TravelingToTerritory,
   TravelingToDropoff, Delivering, Recalling) remain in
   `actForagerStep` and its helpers. Only the Foraging-state
   per-tick loop dissolves into YAML.

5. **Foraging-state transition guards move to the top of
   `forager_step`.** Even though the per-tick loop runs in YAML,
   the state machine still owns transitions OUT of Foraging
   (fatigue limit → TravelingToDropoff, carry threshold →
   TravelingToDropoff, HP emergency → Recalling). These checks
   fire at the top of `forager_step` whenever the current state
   is Foraging, then `forager_step` returns Success without
   dispatching to a foraging tick handler.

6. **Territory-aware wander preserved.** The existing
   `tickForagerForaging` called `npcWanderTerritoryNeighbor`
   on every tick to keep the forager moving inside their
   profile-defined territory. The new YAML branch uses a new
   `wander_territory` btree primitive that delegates to the
   same helper, preserving territory awareness instead of
   falling back to a generic wander.

7. **One shared archetype replaces three per-mob behavior YAMLs.**
   The three foragers (371-tova, 372-halix, 373-kessa) currently
   have per-mob `behaviors/<zone>/<mobId>-<name>.yaml` files. The
   new `behaviors/archetypes/forager.yaml` replaces all three;
   the per-mob YAMLs are deleted.

8. **No new player-facing behavior.** This is a refactor + plumbing
   chunk. Player commands `forage` and `salvage` behave identically
   from the player's POV. Routine foragers' observable behavior
   stays the same (forage in territory, deliver to vendors, recall,
   rest — same cycle, same cadence).

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/actions/forage.go` | NEW | `Forage(actor, opts) ForageResult` |
| `internal/actions/forage_test.go` | NEW | Unit tests, local fake-actor pattern (scan/track/search precedent) |
| `internal/actions/salvage.go` | NEW | `Salvage(actor, opts) SalvageResult` — single-tick core |
| `internal/actions/salvage_test.go` | NEW | Unit tests |
| `internal/usercommands/skill.forage.go` | REWRITE | Thin wrapper (~25 LoC) |
| `internal/usercommands/salvage.go` | REFACTOR | Retains CraftingState scheduling; per-tick logic calls `actions.Salvage` |
| `internal/mobcommands/forage.go` | NEW | Thin wrapper |
| `internal/mobcommands/salvage.go` | REWRITE | Thin wrapper calling `actions.Salvage` |
| `internal/mobcommands/mobcommands.go` | MODIFY | Register `forage` in `mobCommands` map (salvage already registered) |
| `internal/behaviortree/actions_forager_verbs.go` | NEW | `try_forage`, `try_salvage`, `wander_territory` btree primitives |
| `internal/behaviortree/conditions_forager.go` | NEW | `forager_state_is_foraging` |
| `internal/behaviortree/actions.go` | MODIFY | Register `try_forage`, `try_salvage`, `wander_territory` in `init()` |
| `internal/behaviortree/conditions.go` | MODIFY | Register `forager_state_is_foraging` in `init()` |
| `internal/behaviortree/actions_forager.go` | MODIFY | `forager_step` early-returns on Foraging state after running transition guards; `tickForagerForaging` deleted (transition logic moves to top of `forager_step`); `npcAttemptForage` deleted if unused after refactor |
| `internal/behaviortree/context.md` | MODIFY | Document new primitives + condition |
| `internal/actions/context.md` | MODIFY | Document new Forage + Salvage actions |
| `_datafiles/world/dogmud/behaviors/archetypes/forager.yaml` | NEW | Shared archetype |
| `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml` | MODIFY | `behavior_archetype: forager` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml` | MODIFY | Same |
| `_datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml` | MODIFY | Same |
| `_datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml` | DELETE | Obsolete (replaced by archetype) |
| `_datafiles/world/dogmud/behaviors/ironwind_steppe/372-halix.yaml` | DELETE | Same |
| `_datafiles/world/dogmud/behaviors/the_fernway_south/373-kessa.yaml` | DELETE | Same |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.9 Done; size S → M; roll-up 17/41 |

## Public API

### `actions.Forage`

```go
package actions

// ForageOptions parameterizes a forage attempt.
// Biome is derived from the actor's room — no caller override v1.
type ForageOptions struct{}

// ForageResult is the structured outcome.
type ForageResult struct {
    Found        bool   // skill check passed AND biome had yields
    ItemId       int    // 0 if not found
    ItemName     string // for caller messaging
    Reason       string // human-readable when !Found
    OnCooldown   bool   // 6-round cooldown collision on the "forage" key
    RollHappened bool   // for skill progression gating
}

// Forage runs a Perception+Search forage attempt scoped to the
// actor's current room biome. Cooldown key "forage" shared with
// the player path (6 rounds). UserActor emits the existing
// snooping emote + "you find X" message; MobActor SendText is a
// no-op (silent). On success the item is created via items.New,
// added to the actor's inventory, and ItemOwnership event fired.
// Skill progression on every roll-happened path via
// actor.OnSkillUse("search").
func Forage(actor Actor, opts ForageOptions) ForageResult
```

### `actions.Salvage`

```go
package actions

// SalvageOptions identifies the salvage target.
//
//   - TargetCorpse: salvage the first eligible corpse in the room.
//     Used by mob/btree paths for opportunistic gathering.
//   - TargetItemId: salvage a specific item from actor inventory.
//     0 means "no item target". Used by the player path.
//
// Exactly one of TargetCorpse or TargetItemId>0 should be set.
type SalvageOptions struct {
    TargetCorpse bool
    TargetItemId int
}

// SalvageResult is the structured outcome of one salvage tick.
type SalvageResult struct {
    Succeeded    bool
    MaterialIds  []int  // item ids of materials produced
    Reason       string // human-readable on failure
    OnCooldown   bool   // 2-round cooldown on the "salvage" key
    RollHappened bool
}

// Salvage runs one tick of the salvage roll on the named target.
// Single-tick by design — player-side multi-round UX wraps this
// via CraftingState in usercommands/salvage.go.
// UserActor emits per-tick progress text; MobActor is silent.
// Skill progression via actor.OnSkillUse("salvage").
func Salvage(actor Actor, opts SalvageOptions) SalvageResult
```

### Player wrapper shape (forage)

```go
func Forage(rest string, user *users.UserRecord, room *rooms.Room,
            flags events.EventFlag) (bool, error) {
    actor := &actions.UserActor{User: user, Room: room}
    _ = actions.Forage(actor, actions.ForageOptions{})

    // Quest engine notification preserved.
    bridge := questengine.NewGameBridge(user, room.RoomId)
    questengine.GetEngine().Notify("command", questengine.EventDetails{
        UserId:  user.UserId,
        RoomId:  room.RoomId,
        Command: "forage",
    }, bridge, bridge)
    return true, nil
}
```

### Mob wrapper shape (forage)

```go
func Forage(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    actor := actions.NewMobActorInRoom(mob, room)
    _ = actions.Forage(actor, actions.ForageOptions{})
    return true, nil
}
```

### Player salvage wrapper

Player salvage retains the CraftingState multi-round scheduler.
Each tick of the scheduler calls `actions.Salvage(actor, opts)`
with `TargetItemId` set to the in-progress item. The wrapper
manages the activity-cascade integration, message rendering for
multi-round progress, and final yield announcement.

The single-tick `actions.Salvage` returns the materials produced
by ONE tick of work; the player wrapper accumulates them across
the activity's full duration.

## Btree primitive shapes

### `actions_forager_verbs.go`

```yaml
# Forage in the actor's current room. Returns Success on item
# found; Failure on miss, cooldown, or non-foragable biome.
- type: action
  do: try_forage

# Salvage in the current room or against a specific item.
# Default mode (no params): salvage first eligible corpse in room.
# With item_id: salvage that specific inventory item.
# Returns Success on materials returned; Failure on no target /
# no materials / cooldown.
- type: action
  do: try_salvage
  # item_id: 12345  # optional; defaults to corpse-target

# Wander to a random adjacent room within the actor's forager
# territory. Requires the actor to have a forager profile
# registered via forager.ProfileFor(mobId). Returns Success on
# move dispatched; Failure if no profile / no neighbors.
- type: action
  do: wander_territory
```

### `conditions_forager.go`

```yaml
# True when the actor's forager state machine is currently in
# the Foraging state. Cheap pre-check before per-tick foraging
# work.
- type: condition
  check: forager_state_is_foraging
```

## `forager_step` refactor

Top of the existing `actForagerStep` function gains a Foraging-state
guard block. When state is Foraging:

1. **Increment fatigue counter** (was inside `tickForagerForaging`):
   ```go
   fatigue := getIntFromState(ctx.MobState, keyFatigueTimer) + 1
   ctx.MobState.Set(keyFatigueTimer, strconv.Itoa(fatigue))
   ```

2. **Check transition triggers** — fatigue or carry full → TravelingToDropoff:
   ```go
   if fatigue >= fatigueLimit ||
       carryRatio(mob) >= float64(cfg.ForagerCarryThresholdPct) {
       transitionForager(ctx.MobState, forager.StateTravelingToDropoff)
       return Success
   }
   ```
   (HP-emergency transition to Recalling is already at the top of
   `actForagerStep` for ALL states — unchanged.)

3. **Return Success without dispatching to a foraging tick**:
   ```go
   return Success
   ```

The existing switch dispatch loses its `case forager.StateForaging`
arm — that's handled above. The other arms (Resting,
TravelingToTerritory, TravelingToDropoff, Delivering, Recalling)
stay unchanged.

`tickForagerForaging` is deleted. `npcAttemptForage` is deleted
if no other code calls it (verify during implementation).

`npcWanderTerritoryNeighbor` stays — it's exported as the helper
that `wander_territory` btree primitive calls.

## `forager` archetype YAML

```yaml
# forager archetype
#
# Routine NPC foragers (Tova 371, Halix 372, Kessa 373). The
# high-level daily cycle (Resting → Traveling → Foraging →
# Delivering → Recalling) stays in the Go state machine via
# forager_step. The per-tick Foraging loop dissolves into this
# YAML so the verb calls (try_forage, try_salvage) flow through
# the unified actions.Forage / actions.Salvage pipeline.
#
# Spec: docs/superpowers/specs/
#       2026-05-22-mob-aliveness-2.9-mob-forage-salvage-design.md

tree:
  type: selector
  children:

    # 1. Self-defense — fight back if attacked.
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: attack

    # 2. Foraging-state per-tick loop:
    #    salvage opportunistically → try a forage roll → wander
    #    within territory. Each step returns Failure cleanly when
    #    nothing's available, so the selector falls through.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: forager_state_is_foraging
        - type: selector
          children:
            - type: action
              do: try_salvage
            - type: action
              do: try_forage
            - type: action
              do: wander_territory

    # 3. All non-Foraging states (Resting, Traveling, Delivering,
    # Recalling) flow through the Go state machine.
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: forager_step
```

## Forager mob migration

For each of the three forager mob YAMLs
(`371-tova.yaml`, `372-halix.yaml`, `373-kessa.yaml`):

- Set `behavior_archetype: forager`.
- All other fields preserved.

Delete the three per-mob behavior YAMLs:
- `_datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml`
- `_datafiles/world/dogmud/behaviors/ironwind_steppe/372-halix.yaml`
- `_datafiles/world/dogmud/behaviors/the_fernway_south/373-kessa.yaml`

The loader should resolve the new archetype reference cleanly. No
field needs to change on the forager profile YAMLs
(`_datafiles/world/dogmud/foragers/<zone>/<mobid>.yaml`) — those
are profile data, not behavior data, and remain untouched.

## Testing & smoke

### Unit tests

`internal/actions/forage_test.go` (4 tests):
- Non-foragable biome → `Found=false`, `Reason="wrong biome"`.
- Cooldown collision → second call within 6 rounds returns
  `OnCooldown=true` and skips the roll.
- High score → eventually finds (multiple iterations against the
  loaded yield table — same probabilistic shape as the existing
  `internal/forager/forage_core_test.go`).
- MobActor silent — no SendText calls captured by the fake.

`internal/actions/salvage_test.go` (4 tests):
- Single-tick core works on a mob actor with a corpse in the room.
- No-target (no corpse, no item id) → `Reason="no target"`.
- Item-targeted mode → correct item consumed from inventory.
- MobActor silent.

Existing `internal/forager/forage_core_test.go` continues to pass —
the core math is unchanged; only its callers move.

Existing `internal/behaviortree/actions_forager_test.go` continues
to pass — covers state transitions, not the per-tick foraging
behavior (which moves to YAML). Verify; if any test references
`tickForagerForaging` directly, update or delete.

### No new btree primitive unit tests

Consistent with chunks 2.4/2.6/2.7/2.8. Primitives validated via
smoke.

### Smoke test plan

1. **Tova daily cycle.** Watch Tova (371-tova) in Stillwater
   Marsh. Verify she enters Foraging in her territory, the
   YAML branch fires `try_salvage` (no corpses → Failure → fall
   through) → `try_forage` (rolls; some succeed) → `wander_territory`
   (territory-aware adjacent move). When carry threshold hits,
   the Go state machine transitions her to TravelingToDropoff.
   Daily cycle completes; she returns to spawn.

2. **Halix on Ironwind Steppe (372).** Same expected behavior,
   different biome and territory.

3. **Kessa in Fernway South (373).** Same.

4. **Player forage unchanged.** Run `forage` as a player in a
   foragable biome. Should produce the same flavor text and
   yield as before — proves the player-wrapper thin didn't
   regress.

5. **Player salvage unchanged.** Salvage a crafted item. The
   multi-round CraftingState UX fires correctly, progress
   messages render, materials land in inventory at completion.

6. **Mob salvage on corpse.** Kill a critter near Tova's
   territory and verify she salvages the corpse during her
   Foraging state.

7. **`try_forage` / `try_salvage` callable from admin console.**
   Admin-issue `forage` to a non-forager mob (e.g., goblin scout
   217 from chunk 2.8) and confirm it works generically — proves
   the primitives are mob-agnostic.

8. **Build/data smoke.** `go run .` clean boot. The three deleted
   per-mob behavior YAMLs cause no loader errors; the three
   forager mob YAMLs resolve to the new `forager` archetype.
   `behaviorSpec.LoadDataFiles()` loadedCount up by 1 (new
   archetype) minus 3 (deleted per-mob YAMLs) — net depends on
   how the loader counts archetypes vs per-mob.

9. **Roadmap update.** Mark 2.9 Done. Size: S → M. Roll-up:
   17 / 41 done.

Kill test mud servers after smoke per the standing SOP.

## Risks / known limitations

- **`tickForagerForaging` deletion.** Any code outside
  `actions_forager.go` that called `tickForagerForaging` (unlikely
  but verify) breaks. Grep before deleting.
- **`npcAttemptForage` deletion.** Same caveat — verify no other
  callers via grep.
- **`wander_territory` requires forager profile.** Returns
  Failure on mobs without a profile. The forager archetype is
  the only consumer for v1; if other archetypes ever use it,
  they'd need profile registration. Acceptable.
- **Cooldown semantics on `try_forage` / `try_salvage`.** Both
  share their respective cooldown keys with the player path
  (forage: 6 rounds, salvage: 2 rounds). A mob using the
  primitive too fast hits the cooldown and gets Failure. Fine
  for v1.
- **Multi-tick player salvage refactor.** The activity-cascade
  + CraftingState integration in `usercommands/salvage.go` is
  fragile. The per-tick lift to `actions.Salvage` must preserve
  the full progress-message sequence and final-yield behavior.
  Smoke scenario #5 catches regressions.
- **Existing forager state-machine tests.** Some tests may
  reference internal helpers being deleted. Verify by running
  `go test ./internal/behaviortree/...` after the refactor.

## Open questions

- **Should `wander_territory` move to a more general location**
  (e.g., shared with other "territory-bound" archetypes)?
  Currently only foragers have a territory concept. Defer to
  when a second consumer needs it.

## Out of scope

- **Re-implementing the multi-state forager daily cycle in YAML.**
  Hybrid approach keeps the Go state machine.
- **Generalizing the territory concept beyond foragers.** YAGNI
  until a second use case appears.
- **Player-facing changes to `forage` or `salvage` UX.** This is
  a refactor + plumbing chunk.
- **Strategic-NPC consumers of `try_forage` / `try_salvage`.**
  Those are Phase 4. The primitives ship as available
  infrastructure.

## Roadmap impact

- Chunk 2.9 marked Done.
- Roll-up: 17 / 41 done • 0 in progress • 24 not started.
- Size: S → M (scope expanded during plan-writing — salvage parallel
  + archetype migration + state machine refactor).
- Unblocks: chunks 4.1+ (strategic-layer goals that need
  composable `try_forage` / `try_salvage` primitives), 6.1
  (Stillwater town-flavor pass — could give NPCs opportunistic
  foraging idle behavior).
