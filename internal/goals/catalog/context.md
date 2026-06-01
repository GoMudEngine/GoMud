# Goals Catalog Subpackage Context

## Overview

The `internal/goals/catalog` package registers all concrete goal types
with the goals substrate. Each file's `init()` calls
`goals.RegisterGoalType(name, goals.GoalTypeMeta{...})`. The package
has no exported symbols — it exists only for its registration side-
effects. `main.go` fires those registrations via a blank import:

```go
import _ "github.com/GoMudEngine/GoMud/internal/goals/catalog"
```

The 13 types cover the full strategic vocabulary for chunk 4.3 of the
mob aliveness roadmap (survival through zone exploration).

## File Layout

| File | Contents |
|------|----------|
| `catalog.go` | Package doc + type list (no runnable code) |
| `helpers.go` | Shared adapters: `resolveTargetRoomId`, `targetAlive`, `targetProximityHops`, `targetInCombat`, `mobMiscInt`, `factionMembersInZone` |
| `survival.go` | `survival` type + `paramIntOr` helper |
| `wealth_gold.go` | `wealth-gold` type |
| `wealth_item.go` | `wealth-item` type |
| `craft_item.go` | `craft-item` type |
| `revenge_mob.go` | `revenge-mob` type |
| `revenge_faction.go` | `revenge-faction` type + `mobMiscInt`, `factionMembersInZone` |
| `protection_mob.go` | `protection-mob` type |
| `protection_faction.go` | `protection-faction` type |
| `befriend.go` | `befriend` type |
| `befriend_faction.go` | `befriend-faction` type |
| `mastery_skill.go` | `mastery-skill` type + `mobSkillRank`, `skillTrainingProximity` stubs |
| `mastery_equip.go` | `mastery-equip` type + `shopProximity` stub |
| `visit_zone.go` | `visit-zone` type + `zoneGraphDistance` stub |
| `*_test.go` | One test file per type; `catalog_test.go` covers the full 13-type registry |

## The 13 Types

| Name | AllowMultiple | Purpose |
|------|---------------|---------|
| `survival` | false | Suppress other activity when HP is low; flee threshold drives ContextScore to 5× |
| `wealth-gold` | false | Accumulate gold to a target amount; ContextScore scales with shortfall |
| `wealth-item` | true | Acquire one specific item (by ItemId); dedup by item id |
| `craft-item` | true | Craft one specific recipe (by recipe id); dedup by recipe id |
| `revenge-mob` | true | Kill N instances of a specific mob template; dedup by target mob id |
| `revenge-faction` | true | Kill N members of a named faction; dedup by faction id |
| `protection-mob` | true | Keep a specific mob/player alive; ContextScore peaks when target is in combat |
| `protection-faction` | true | Keep faction members in zone alive; ContextScore scales with members under threat |
| `befriend` | true | Build rep with a specific mob/player; dedup by target id + kind |
| `befriend-faction` | true | Build rep with a named faction; dedup by faction id |
| `mastery-skill` | true | Reach a target rank in a named skill; dedup by skill name |
| `mastery-equip` | true | Acquire and equip an item of a given rarity tier + slot; dedup by slot |
| `visit-zone` | true | Travel to and explore a target zone; dedup by zone name |

## Registration Pattern

Each type file contains exactly one `init()` function:

```go
func init() {
    goals.RegisterGoalType("wealth-gold", goals.GoalTypeMeta{
        Predicate:    wealthGoldPredicate,
        ContextScore: wealthGoldContextScore,
        Params: []goals.ParamSchema{
            {Key: "target", Required: true, GoType: "int"},
        },
        // AllowMultiple: false  ← omit when false
    })
}
```

`Predicate` and `ContextScore` are file-private functions. They receive
a `*goals.Goal` and a `*mobs.Mob`; both may be nil. Panics inside either
are caught by wrapper functions in `goals.store.go` (logs warn, returns
safe default: `false` for Predicate, `0.0` for ContextScore).

## How to Add a New Type

1. Create `internal/goals/catalog/<type_name>.go` (underscores for
   hyphens, e.g. `my_type.go` for `"my-type"`).
2. Declare a `ParamSchema` slice covering every expected key.
3. Write `<type>Predicate` (satisfied = goal complete) and
   `<type>ContextScore` (0 = suppress; higher = more urgent).
4. If `AllowMultiple: true`, declare `<type>DedupKey` — it must return
   a stable string that uniquely identifies the target for this type.
   Same DedupKey on two goals of the same type = conflict (same target).
5. Wire in `init()` with `goals.RegisterGoalType`.
6. Add `<type>_test.go` alongside (minimum: predicate + contextScore
   with nil mob + one satisfied/unsatisfied pair).
7. Run `go test ./internal/goals/...` — `catalog_test.go` asserts that
   exactly 13 types are registered; update that count.

## Adapter Pattern (TODO-ADAPT Helpers)

Catalog files access subsystems (factions, mobs, rooms, skills, items)
only through file-private or `helpers.go` adapter functions. The adapter
is the single blast-radius point when a subsystem API changes.

Adapters as of chunk 4.3:

| Adapter | Location | Wraps |
|---------|----------|-------|
| `resolveTargetRoomId(kind, id)` | helpers.go | `mobs.GetAllMobInstanceIds`, `users.GetByUserId` |
| `targetAlive(kind, id)` | helpers.go | mob instance scan + user health check |
| `targetProximityHops(mob, kind, id)` | helpers.go | `resolveTargetRoomId` + `rooms.LoadRoom` zone check |
| `targetInCombat(kind, id)` | helpers.go | `mob.Character.Aggro` / `user.Character.Aggro` |
| `mobMiscInt(mob, key)` | revenge_faction.go | `mob.Character.GetMiscData` with int64 widening |
| `factionMembersInZone(mob, fid)` | revenge_faction.go | `mobs.GetAllMobInstanceIds` + `factions.FactionsForMob` |
| `mobSkillRank(mob, name)` | mastery_skill.go | `mob.Character.GetSkillLevel` |
| `skillTrainingProximity(mob, name)` | mastery_skill.go | TODO-ADAPT stub; always returns 1 |
| `shopProximity(mob)` | mastery_equip.go | TODO-ADAPT stub; always returns -1 |
| `zoneGraphDistance(from, to)` | visit_zone.go | TODO-ADAPT stub; always returns 1 |

The three TODO-ADAPT stubs are deliberate 4.3 heuristic shortcuts:

- **`skillTrainingProximity`**: always returns 1 (in zone, not room).
  The per-skill training-context table (combat trains via fighting,
  crafting trains at stations) is deferred to chunk 4.4.
- **`shopProximity`**: always returns -1 (no shop known). The shop
  lookup that would return a vendor in the current zone stocking the
  target rarity tier is deferred to chunk 4.4.
- **`zoneGraphDistance`**: always returns 1. The BFS over zone
  adjacency (derived from inter-zone room exits) is deferred to
  chunk 4.4, where the planner integrates zone-aware pathfinding.

## MiscData Counters (Write Path Deferred to 4.5)

Several types read mob MiscData counters that chunk 4.5 will write:

| Counter key pattern | Read by | Written by |
|---------------------|---------|------------|
| `faction_kills_inflicted:<factionId>` | `revenge-faction` Predicate | 4.5 on-kill hook |
| `faction_rep_built_with:<factionId>` | `befriend-faction` Predicate | 4.5 on-interact hook |

Until 4.5 ships, both Predicates always return false (counter absent =
0 < any positive target). Goals of these types are held correctly by
the selection engine; they just never complete.

## Known Limitations

- **Rarity-tier tuning** in `mastery-equip` uses `rarity_tier`
  comparisons, but many items lack that field (see
  `project_rarity_tier_audit.md`). The fallback treats unlabeled
  items as tier 50 (mid-range). Scores will be rough until the audit
  lands.
- **`befriend` / `protection-mob` ContextScore** uses
  `targetProximityHops` which collapses to a two-bucket answer
  (same room = 1.5, same zone = 0.8, other = 0). No full BFS.
- **`factionMembersInZone`** counts all live instances with the faction
  tag, including friendly faction members who were already allies.
  In 4.3 there is no "enemy faction member" filter — correction
  belongs in 4.5 when opinion scores are available.

## chunk 5.3 — upgrade-gear goal type

`upgrade_gear.go` — a perpetual "want better gear" drive. `Predicate` always
returns false (no terminal state); activation is governed entirely by
`ContextScore`, which returns a positive floor (`1.0`) in all cases so the 4.6
dormancy sweep never abandons this standing default, rising to `2.5` when the
mob is idle/out-of-combat AND has a plausible path to a purchase (spendable gold
above the reserve, or sellable loot to fund saving). Deliberately cheap and
self-contained — no shop scan in scoring (the chunk-5.3 planner owns stock
decisions, mirroring the `mastery-equip` precedent). Optional `reserve` int
param overrides the `MobUpgradeGoldReserve` config default. `AllowMultiple:
false`. The matching planner lives in `internal/planners/upgrade_gear.go`.

## Out of Scope

- **Planner (4.4)**: btree actions that read `CurrentGoalOf` and
  translate goal type + params into a concrete action sequence.
- **Reactive seeding (4.5)**: hooks that call `goals.Add` in response
  to in-world events (ally death, theft witnessed, faction kill).
- **Satisfaction sweep (4.6)**: periodic background pass that removes
  satisfied/expired goals from disk.
