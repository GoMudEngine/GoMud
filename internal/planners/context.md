# Planners Package Context

## Overview

The `internal/planners` package is the tactical-translation layer for chunk
4.4 of the mob aliveness roadmap. It bridges the strategic goal selected by
chunk 4.2 (`goals.CurrentGoalOf`) with the per-round mob command that
actually executes in-world.

Each goal type has exactly one registered planner — a `PlanFn` that receives
the current mob and its active `*goals.Goal` and returns a `PlanResult`
(command string + btree status) to execute this tick. Planners are called by
the `try_goal_planner` btree action (chunk 4.4, Task 5); if no planner is
registered for the active goal type, the btree falls through as if the node
returned Failure.

The package is stateless from the framework's perspective: planners write any
intermediate progress to `mob.Character.MiscData` under the `plan:<type>:`
prefix and recover it on the next tick. On goal switch, `ClearPlanState`
wipes all `"plan:"` prefixed keys automatically.

Spec: `docs/superpowers/specs/2026-05-27-mob-aliveness-4.4-strategic-
tactical-translation-design.md`

## File Layout

| File | Contents |
|------|----------|
| `planners.go` | Framework: `BTreeStatus`, `PlanResult`, `PlanFn`, registry (`RegisterPlanner`, `LookupPlanner`) |
| `state.go` | `ClearPlanState`, `PlanKeyPrefix` constant |
| `helpers.go` | Subsystem adapters: shop finders, faction/hostile finders, station finder, zone-adjacency cache, gift picker, random exit, social emote rotation, recipe picker, MiscData read/write helpers, goal-param read helpers |
| `skill_training_table.go` | `SkillTrainingContext` type + constants, `skillTrainingTable` map, `SkillTrainingContextOf` |
| `survival.go` | `survival` planner |
| `wealth_gold.go` | `wealth-gold` planner |
| `wealth_item.go` | `wealth-item` planner |
| `craft_item.go` | `craft-item` planner |
| `revenge_mob.go` | `revenge-mob` planner + `resolveTargetRoomId`, `targetCommandName` helpers |
| `revenge_faction.go` | `revenge-faction` planner |
| `protection_mob.go` | `protection-mob` planner |
| `protection_faction.go` | `protection-faction` planner |
| `befriend.go` | `befriend` planner |
| `befriend_faction.go` | `befriend-faction` planner |
| `mastery_skill.go` | `mastery-skill` planner |
| `mastery_equip.go` | `mastery-equip` planner |
| `visit_zone.go` | `visit-zone` planner + `exitRoomToward` helper |
| `*_test.go` | One test file per source file |

## Activation

`main.go` pulls the package via a blank import:

```go
import _ "github.com/GoMudEngine/GoMud/internal/planners"
```

This fires every per-planner `init()` registration before the server
loop starts. The same boot call also registers `ClearPlanState` into the
goals layer:

```go
// main.go — boot wiring (Task 4)
goals.SetPlanStateClear(planners.ClearPlanState)
```

After this wiring, `goals.Recompute` invokes `ClearPlanState` on every goal
switch, ensuring stale plan state never bleeds across goal types.

## The 13 Planners

| Goal type | File | Description |
|-----------|------|-------------|
| `survival` | `survival.go` | Flee combat, drink heal potion, or rest until HP meets safe threshold |
| `wealth-gold` | `wealth_gold.go` | Sell loot at a vendor in zone until gold target met |
| `wealth-item` | `wealth_item.go` | Buy or forage for a specific item (by tag or id) |
| `craft-item` | `craft_item.go` | Navigate to crafting station and execute a known recipe |
| `revenge-mob` | `revenge_mob.go` | Pursue and attack a specific mob template until kill count met |
| `revenge-faction` | `revenge_faction.go` | Pursue and attack faction members in zone |
| `protection-mob` | `protection_mob.go` | Defend a named ally; attack their aggressor or close distance |
| `protection-faction` | `protection_faction.go` | Defend faction members in zone; attack hostiles threatening them |
| `befriend` | `befriend.go` | Raise opinion with named target via social emotes + pathfinding |
| `befriend-faction` | `befriend_faction.go` | Emit social actions toward faction members in zone |
| `mastery-skill` | `mastery_skill.go` | Train a skill via context-appropriate actions (combat/craft/forage/social) |
| `mastery-equip` | `mastery_equip.go` | Upgrade an equipment slot to a target rarity tier via vendor shopping |
| `visit-zone` | `visit_zone.go` | Walk toward a target zone using zone-adjacency graph hops |

## Framework Types (planners.go)

### BTreeStatus
```go
type BTreeStatus int
const (
    StatusFailure BTreeStatus = iota
    StatusSuccess
    StatusRunning
)
```
Mirrors the behaviour-tree status enum. Re-exported here to avoid forcing
planner files to import `internal/behaviortree` (which would create an
import cycle since btree imports mobs).

### PlanResult
```go
type PlanResult struct {
    Command string       // mob command to execute this tick; "" = no action
    Status  BTreeStatus  // propagated as try_goal_planner node result
}
```
`Command` is executed via `mob.Command(cmd)` inside the `try_goal_planner`
btree action. An empty `Command` with `StatusRunning` is a valid no-op tick
(e.g., waiting on a cooldown).

### PlanFn
```go
type PlanFn func(mob *mobs.Mob, goal *goals.Goal) PlanResult
```
Stateless from the framework's perspective. Implementations must be safe to
call from multiple goroutines simultaneously (the mob tick loop may fan out
across goroutines). Do not store state in closures — use `mob.Character
.MiscData` under the `plan:<type>:` key prefix instead.

## MiscData Convention

All plan state written to MiscData uses the `plan:<goal_type>:<key>` prefix:

```
plan:wealth-gold:target_shop_room   int    — cached vendor room id
plan:wealth-item:target_shop_room   int    — cached vendor room id
plan:craft-item:target_station_room int    — cached station room id
plan:mastery-equip:target_shop_room int    — cached vendor room id
plan:befriend:cooldown_round        int    — round after which next interaction fires
plan:visit-zone:next_hop_zone       string — intermediate zone hop target
```

`ClearPlanState` (`state.go`) deletes every key whose name starts with
`"plan:"` from `mob.Character.MiscData`. It is nil-safe on both mob and
MiscData. Wired via `goals.SetPlanStateClear` at boot (Task 4); fires
automatically on every goal switch inside `goals.Recompute`.

## Helpers (helpers.go)

File-private adapter functions decouple planners from subsystem APIs.
The adapter is the single blast-radius point when a subsystem API changes.

| Helper | Purpose |
|--------|---------|
| `findShopInZoneSelling(mob, tag, itemId)` | First shop in mob's zone stocking the target item or component tag |
| `findShopInZoneBuying(mob)` | First shop in mob's zone with gold reserves (can buy from mob) |
| `findFactionMemberInZone(mob, factionId, mustBeInCombat)` | First faction member in zone; optional combat filter |
| `findHostileInZone(mob)` | First auto-aggro mob in zone (excluding self) |
| `findCraftingStationInZone(mob, stationName)` | First room in zone whose `Station` field matches |
| `zoneAdjacentTo(zone)` | Zones sharing a room-exit border; lazy-computed + cached at first call |
| `pickGiftItemFromInventory(mob)` | Highest-value non-quest item from backpack for gifting |
| `pickRandomExit(mob)` | Random exit direction name from mob's current room |
| `pickSocialEmote()` | Random social emote command string (nod/bow/smile/wave/grin) |
| `pickKnownRecipeForSkill(mob, skillName)` | Lowest-SkillMinimum known recipe that trains the given skill |
| `mobMiscIntOr(mob, key, def)` | Read int from MiscData with default |
| `mobMiscStringOr(mob, key, def)` | Read string from MiscData with default |
| `mobSetMisc(mob, key, val)` | Write value to MiscData; nil-safe |
| `goalParamIntOr(goal, key, def)` | Read int from goal.Params with default |
| `goalParamStringOr(goal, key, def)` | Read string from goal.Params with default |

`resolveTargetRoomId(kind, id)` and `targetCommandName(kind, id)` live in
`revenge_mob.go` (they were created there and used by several planners in
that file). They follow the same adapter pattern.

## Skill Training Table (skill_training_table.go)

`SkillTrainingContextOf(skillName)` maps a `skills.SkillTag` string to one
of five context constants:

| Context | Constant | Skills |
|---------|----------|--------|
| `"combat"` | `TrainingCombat` | weapon-combat, unarmed-combat, spellcasting, manifestation |
| `"crafting"` | `TrainingCrafting` | blacksmithing, alchemy, tailoring, cooking, jewelcrafting, enchanting, salvage |
| `"foraging"` | `TrainingForaging` | search |
| `"social"` | `TrainingSocial` | rhetoric, bartering |
| `"skullduggery"` | `TrainingSkullduggery` | skullduggery |
| `"unknown"` | `TrainingUnknown` | (any unrecognized name) |

The `mastery-skill` planner switches on `SkillTrainingContextOf` to pick the
appropriate action (attack for combat, craft for crafting, wander for
foraging, emote for social, etc.). When a new skill is added to the codebase,
append a row to `skillTrainingTable` in this file.

## How to Add a New Planner

1. Create `internal/planners/<type_name>.go` (underscores for hyphens,
   e.g. `my_type.go` for `"my-type"`).
2. Declare any MiscData keys as package-level `const` strings following the
   `"plan:<type>:<key>"` naming convention.
3. Write the `PlanFn` implementation (`<type>Planner`). Always guard
   `mob == nil` at the top and return `StatusFailure`.
4. Register in `init()`:
   ```go
   func init() {
       RegisterPlanner("my-type", myTypePlanner)
   }
   ```
5. Add subsystem adapters to `helpers.go` if the planner needs new
   cross-package queries (do not call subsystem APIs inline in the
   planner function).
6. Add `<type>_test.go` alongside (minimum: nil-mob guard returns Failure,
   success branch, one running branch).
7. There is no registry count assertion to update (unlike the goals/catalog
   package). The `try_goal_planner` btree action falls through gracefully for
   unregistered types, so adding a new planner is non-breaking.

## chunk 5.3 — upgrade-gear (equipment-aware shopping)

`upgrade_gear.go` — the survey-worst-slot equipment-shopping planner. It does
not target a pre-chosen slot; `shop_upgrade.go`'s `scanZoneUpgrades` scores
every in-stock item across the mob's zone shops via `itemvalue.ItemValueDelta`
(so the highest positive delta naturally targets whichever slot benefits most),
prices each with `shops.CalcSellPrice` (the buyer-side dynamic price — NOT
`CalcBuyPrice`), and returns the best affordable positive-delta candidate
(tie-break lower price).

Per-tick state machine: (1) `pending_equip` flag set last tick → `gearup`;
(2) affordable upgrade → `buy <name>` at the shop (set the equip flag for next
tick) or `pathto` it; (3) upgrade exists but unaffordable → compose the
wealth-gold sell loop to save up (`sell all` / `pathto` a buying vendor /
`wander`); (4) nothing in stock → idle. The buy target is rescanned each tick
(zone shops are few), so only the save-up sell vendor is sticky
(`plan:upgrade-gear:sell_shop_room`); `plan:upgrade-gear:pending_equip` is the
one-shot equip flag. Because the goal's predicate is always false (perpetual
drive), `ClearPlanState` only fires on a goal SWITCH — so the buy branch
explicitly clears the sell-shop sticky to prevent a stale vendor persisting
across buy/sell cycles within one continuous run. Config knobs:
`MobUpgradeGoldReserve`, `MobUpgradeMinDelta`. Seeded as a low-priority
`default_goals` entry on `thief` and `guard_captain`. Same-zone only;
cross-respawn persistence of bought gear is out of scope (instance saves are
wiped in prod / smoke).

## Out of Scope

- **Reactive goal seeding (4.5)**: hooks that call `goals.Add` when an ally
  dies, a theft is witnessed, or a faction kill occurs. Planners consume
  goals already in the store; they do not create new ones.
- **Satisfaction sweep (4.6)**: periodic removal of completed or expired
  goals from disk.
- **Planner visualization**: no admin command reads or displays per-planner
  MiscData state. The existing `goal current <mobId>` command shows the
  selected goal; per-step plan progress is opaque.
- **General-purpose planner**: no fallback planner for unregistered goal
  types. Unregistered types silently fall through in the btree.
- **Cross-zone pursuit**: planners that require a target in a different zone
  return `StatusFailure` rather than issuing cross-zone `pathto`. Zone-hop
  travel logic (chunked as chunk 3.7) is out of scope here.
