# Seeders Package Context

## Overview

The `internal/seeders` package is the reactive goal-generation layer for
chunk 4.5 of the mob aliveness roadmap. It observes world events (mob
deaths, attacks, gifts, quest completions, skill rank-ups) and seeds
goals into the goals substrate so NPCs react to things that happen
around them — a friend's death triggers a revenge goal, a gift boosts
the giver's standing, faction kills increment counters that drive
existing `revenge-faction` and `befriend-faction` predicates.

Seeders are complementary to planners (chunk 4.4). Planners translate
an already-selected goal into per-tick commands. Seeders *create* the
goals that planners later act on. The flow is:

```
world event → seeders.Dispatch → rule fn → goals.Add / opinions.Bump
                                                    ↓
                               next tick: goals.Recompute picks highest goal
                                                    ↓
                               try_goal_planner btree action executes it
```

## Activation

`main.go` imports the package explicitly (not blank — exported entry
points `SeedMaterialsForRecipe` and `OnTheft` are called from planners
and actions respectively). Per-rule `init()` registrations fire at
import time. `main.go` then wires `seeders.Dispatch` as the listener
for every event type the package subscribes to:

```go
import "github.com/GoMudEngine/GoMud/internal/seeders"

// chunk-4.5 event listeners — one line per subscribed event type
events.AddListener(events.MobDeath{},         seeders.Dispatch)
events.AddListener(events.Communication{},    seeders.Dispatch)
events.AddListener(events.Quest{},            seeders.Dispatch)
events.AddListener(events.PlayerAttackedMob{}, seeders.Dispatch)
events.AddListener(events.GiftAccepted{},     seeders.Dispatch)
events.AddListener(events.SkillUsed{},        seeders.Dispatch)
events.AddListener(events.ItemOwnership{},    seeders.Dispatch)
```

`SkillUsed` and `ItemOwnership` are listed here for completeness; see
rule 10 and rule 5 notes below — both rules have architectural reasons
why the listener alone is insufficient.

## File Layout

| File | Contents |
|------|----------|
| `seeders.go` | Framework: `RuleFn` type, `Register`, `Dispatch`, `invokeRuleSafely` (panic recovery), `resetRegistryForTest` |
| `state.go` | Shared helpers: `applyCooldown`, `seedRevengeGoalIfAbsent`, `readMiscUint64`, `readMiscInt`, `bumpMiscInt`, `paramAsInt`, `resolveKillerFromMobDeath`, `instanceIdAsKey`, `userIdAsKey` |
| `faction_kill_counter.go` | Rule 1 — MobDeath subscriber, writes faction kill counters |
| `faction_rep_counter.go` | Rule 2 — Communication subscriber (stub) |
| `craft_materials_to_wealth_item.go` | Rule 3 — exported `SeedMaterialsForRecipe`, planner-invoked |
| `friend_killed_to_revenge.go` | Rule 4 — MobDeath subscriber, relationships walk |
| `witness_of_theft_to_revenge.go` | Rule 5 — exported `OnTheft`, steal-action-invoked |
| `aggressive_action_to_revenge.go` | Rule 6 — PlayerAttackedMob subscriber |
| `gift_to_opinion_boost.go` | Rule 7 — GiftAccepted subscriber |
| `quest_completion_to_opinion_boost.go` | Rule 8 — Quest subscriber (stub) |
| `combat_assist_to_opinion_boost.go` | Rule 9 — PlayerAttackedMob subscriber |
| `mastery_milestone_to_priority_bump.go` | Rule 10 — SkillUsed subscriber (deferred) |
| `*_test.go` | One test file per source file; `testmain_test.go` provides shared fixtures |

## The 10 Rules

| # | Name | Trigger | Effect | Status |
|---|------|---------|--------|--------|
| 1 | `faction_kill_counter` | `MobDeath` (mob killer) | Bumps `faction_kills_inflicted:<fid>` on killer for each of victim's factions | **LIVE** |
| 2 | `faction_rep_counter` | `Communication` | Would bump `faction_rep_built_with:<fid>` on receiver for player→mob positive interaction | **STUB** — `Communication` carries no positive-interaction subtype; early-returns on every event |
| 3 | `craft_materials_to_wealth_item` | Planner direct-invoke | Seeds `wealth-item` (priority 60) for each missing recipe ingredient when craft-item planner hits a missing-materials Failure | **LIVE** (architectural exception — `SeedMaterialsForRecipe`) |
| 4 | `friend_killed_to_revenge` | `MobDeath` | Walks victim's chunk-1.6 relationship edges; seeds `revenge-mob` (priority 85) on each friend/family/lover targeting the killer | **LIVE** |
| 5 | `witness_of_theft_to_revenge` | Steal-action direct-invoke | Victim (pri 90) + room witnesses (pri 60) on successful steal, **routed through the witness-response classifier** (see below) | **LIVE** (architectural exception — `OnTheft`) |
| 6 | `aggressive_action_to_revenge` | `PlayerAttackedMob` | Attacked mob (pri 75) + non-`AutoAggro` room witnesses (pri 50), **routed through the witness-response classifier** | **LIVE** |

### Witness-response classifier (`witness_response.go`, 2026-06-05)

Rules 5 (theft) and 6 (assault) no longer blanket-seed `revenge-mob` into every
room witness. Both route victim + witnesses through `seedWitnessResponse`, which
calls the pure `classifyWitnessResponse(mob) WitnessResponse`:
- **guard** (`mobs.IsGuardMob(groups)`) → `ResponseReportOnly` — seed nothing; the
  5.1 crime record + `RunGuardEnforcement` handle it (a personal revenge goal would
  derail proper enforcement).
- **noncombatant** (`mob.IsNonCombatant()`) → `ResponseAlarm` — `alarmReaction`: a
  fright emote + one step toward a random exit. No persistent goal (avoids the
  survival-goal-pruned-at-full-HP behavior). The 5.1 crime record is the report.
- **combat-capable non-guard** → `ResponseRevenge` — `seedRevengeGoalIfAbsent`
  (unchanged).

The victim is never a noncombatant (you cannot steal from / attack a
`non_combatant`), so it only hits guard or revenge. Rule 4 (`friend_killed_to_revenge`)
is **unchanged** — it is relationship/kin-scoped, not indiscriminate room-witness,
so it already targets the right mobs. (Filename `witness_of_theft_to_revenge.go` is
now slightly broad — it also reports/alarms — but kept to avoid churn.)
| 7 | `gift_to_opinion_boost` | `GiftAccepted` | Value-tiered opinion bump (+1/+3/+5/+8) with per-(giver, receiver) cooldown of 100 rounds | **LIVE** |
| 8 | `quest_completion_to_opinion_boost` | `Quest` (`-end` token) | Would bump quest-giver opinion on completion | **STUB** — `quests.Quest` struct carries no `GiverMobTemplateId`; returns 0 and logs at debug |
| 9 | `combat_assist_to_opinion_boost` | `PlayerAttackedMob` | If the attacked mob was already in combat with a different mob, bumps that mob's opinion of the assisting player (cooldown 150 rounds) | **LIVE** (shared listener with rule 6) |
| 10 | `mastery_milestone_to_priority_bump` | `SkillUsed` | Would seed `mastery-skill` goal at each rank-10 milestone for the progressing mob | **DEFERRED** — `events.SkillUsed` is player-only and carries no `MobInstanceId` or `NewRank` field; `Register` call is commented out |

## Framework (seeders.go)

### RuleFn
```go
type RuleFn func(event events.Event)
```
A rule receives the raw event, performs its own type assertion, and
returns. Rules must return early on unexpected payload shapes rather
than panicking — the framework provides panic recovery, but clean
early-returns are preferred.

### Register
```go
func Register(ruleName string, fn RuleFn, types ...string)
```
Called from each per-rule `init()`. `types` contains the event type
name strings (e.g., `"MobDeath"`, `"GiftAccepted"`) as returned by
the corresponding `events.Event.Type()` method. One registration maps
the rule to all listed types simultaneously — useful when two rules
share a listener (rules 6 + 9 both subscribe to `PlayerAttackedMob`).

### Dispatch
```go
func Dispatch(event events.Event) events.ListenerReturn
```
The single wiring point in `main.go`. Looks up all rules registered
for `event.Type()`, invokes each under `invokeRuleSafely`. Always
returns `events.Continue` — seeders observe events, never suppress
them. Panic in one rule does not prevent subsequent rules from firing.

## Shared Helpers (state.go)

### applyCooldown
```go
func applyCooldown(mob *mobs.Mob, ruleName, key string, windowRounds uint64) bool
```
Throttles repeated firings on the same (mob, rule, key) triple.
Returns `true` and writes an expiry round if no active cooldown exists;
returns `false` without writing if the cooldown is still live. Cooldown
markers live in `mob.Character.MiscData` under:

```
seed_cooldown:<rule_name>:<key>
```

The `seed_cooldown:` prefix is intentionally distinct from the chunk-4.4
`plan:` prefix — `ClearPlanState` (fired on goal switch) does NOT wipe
seeder cooldowns. Cooldowns survive goal switches and persist until
their expiry round is past.

### seedRevengeGoalIfAbsent
Walks `goals.GoalsOf` for an existing `revenge-mob` goal targeting the
same `(target_kind, target_id)`. If absent, calls `goals.Add` with the
standard revenge-mob shape (priority passed by caller). Dedup prevents
priority escalation on repeat offenses within 4.5 — a mob that is
attacked twice still has one revenge goal at the original priority.

### MiscData helpers
`readMiscUint64`, `readMiscInt`, `bumpMiscInt` — type-coercing wrappers
for `mob.Character.GetMiscData` / `SetMiscData`. Handle `int`/`int64`
widening from YAML load. Pattern matches the 4.3 catalog and 4.4
planners packages.

### resolveKillerFromMobDeath
Extracts `(kind, id)` from a `MobDeath` event. Returns `("player", userId)`
when `KillerUserId > 0`, `("mob", instanceId)` when
`KillerMobInstanceId > 0`, `("", 0)` when killer is unresolvable.

## Architectural Exceptions

Two rules are invoked directly by other packages rather than via the
event dispatcher. This is the "clean call beats noisy event" pattern:

**Rule 3 — `SeedMaterialsForRecipe(mob, recipeId)`**
Called from `internal/planners/craft_item.go`'s Failure branch when
materials are missing. No clean "planner failed on materials" world
event exists. Import direction: `planners → seeders` (seeders does not
import planners, so no cycle).

**Rule 5 — `OnTheft(thiefUserId, victimMob, item)`**
Called from `internal/actions/steal.go` at the steal-success point.
`events.ItemOwnership` lacks a theft subtype — any filter over
ItemOwnership would over-fire on normal drops, shop purchases, etc.
The direct call avoids a false-positive-heavy event path.

Neither function calls `Register`. Neither appears in the dispatcher's
registry. Both are exported, so callers resolve them without a blank
import.

## MiscData Key Conventions

Seeder-written keys follow two naming patterns:

| Pattern | Used by | Purpose |
|---------|---------|---------|
| `seed_cooldown:<rule>:<discriminant>` | `applyCooldown` | Per-(rule, target) throttle; survives goal switches |
| `faction_kills_inflicted:<factionId>` | Rule 1 | Kill counter read by `revenge-faction` Predicate |
| `faction_rep_built_with:<factionId>` | Rule 2 (stub) | Rep counter read by `befriend-faction` Predicate |

These keys live on the mob that acts or accumulates (the killer for
rule 1, the receiver for rule 7's cooldown). They are not scoped to a
goal instance — they accumulate over the mob's lifetime and survive
`goals.Clear`.

The chunk-4.4 `plan:<type>:<key>` namespace is strictly separate.

## How to Add a New Rule

1. Create `internal/seeders/<rule_name>.go`. Declare a `const ruleName<Rule>`
   constant matching the registration string.

2. If the rule subscribes to an event, add an `init()` that calls
   `Register(ruleNameFoo, fooRule, "EventTypeName")`. If the rule is an
   architectural exception (no suitable event), export a named function
   and have the caller invoke it directly — do NOT add a Register call.

3. Write the `RuleFn`. Guard the type assertion at the top and return
   early on failure. Access subsystems only after nil-checking every
   pointer from those systems.

4. If the rule needs per-(mob, target) throttling, call `applyCooldown`
   before the effect. Choose a stable discriminant string (`userIdAsKey`,
   `instanceIdAsKey`) as the cooldown key.

5. If the rule is a new event subscription, add the corresponding
   `events.AddListener` line in `main.go` (next to the existing chunk-4.5
   block — keep the list hand-maintained and annotated).

6. Add `<rule_name>_test.go` alongside. Minimum coverage: registration
   check, nil/unmatched-type no-panic, one branch test.

7. Run `go test ./internal/seeders/ -v` and `go build ./...`.

## Out of Scope

- **Prune sweep (4.6)**: periodic removal of satisfied/expired goals.
  Seeder goals accumulate on disk until 4.6 sweeps them.
- **Cross-type conflict mechanism**: `ConflictsWith` on `GoalTypeMeta`
  is available but no seeder-created goal declares cross-type conflicts
  in 4.5. Deferred to a future balance pass.
- **Per-archetype gating**: all 4.5 rules fire for any mob that satisfies
  the event conditions. Guards like "only hostile-archetype mobs get
  revenge goals" are not applied. This is intentional for 4.5 —
  over-seeding is observable and tunable; under-seeding produces silent
  failures.

## Known Follow-Ups

- **Rule 2 (faction_rep_counter)**: needs a clean positive-interaction
  event signal. Candidates: `GiftAccepted` (already exists), a future
  `TradeCompleted`, or a `QuestTurnedIn` event carrying giver mob id.
  Track as a follow-up to whichever chunk introduces the richer event.

- **Rule 5 witness-of-theft archetype split**: guards and civilians
  should react differently to witnessed theft (guards: report to faction
  authority; civilians: personal revenge). Deferred to chunk 5.1 Town
  Justice. See `project_revenge_witness_report_vs_react_followup.md`.

- **Rule 8 (quest_completion_to_opinion_boost)**: needs a
  `GiverMobTemplateId int` field on `quests.Quest` (the YAML struct).
  Also needs a per-quest `complete_opinion_bump int` override field.
  Once added, `resolveQuestGiverMobId` in the rule file is the only
  site to update.

- **Rule 10 (mastery_milestone_to_priority_bump)**: needs
  `events.SkillUsed` (or a new `events.MobSkillProgressed`) extended
  with `MobInstanceId int` and `NewRank int` fields, and the event
  fired from the mob skill-progression path. The rule's full logic is
  already implemented — only the `Register` call is commented out.
  Uncomment it plus adapt the two `TODO-ADAPT` field reads when the
  event surface is ready.

## Spec Reference

`docs/superpowers/specs/2026-05-27-mob-aliveness-4.5-reactive-goal-generation-design.md`
