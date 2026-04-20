# Companion AI Phase 4 — Melee Self-Buff Archetype

**Date:** 2026-04-20
**Scope:** First slice of Companion Phase 4 ("smarter casting, self-buff"). Introduces behavior-tree archetypes as a first-class concept and delivers the first archetype: a melee specialist who maintains self-buffs.
**Status:** Design — awaiting user review before plan.

---

## Summary

DOGMud has two parallel mob AI systems today: the older reactive-tactics system in `internal/mobai/` (~46 mobs) and the behavior-tree framework in `internal/behaviortree/` (12 mobs + 11 rooms). Companions currently run the same AI as wild mobs — there is no companion-specific behavior, no self-buff logic, and no cross-mob archetype concept.

This spec delivers **the first reusable btree archetype** and the framework bits needed to support it. The archetype is a melee specialist who maintains self-buffs in priority order and otherwise attacks. Three summons adopt it in this phase: vampire (304), air elemental (312), fire elemental (313).

Archetypes are introduced as a first-class concept (not as shared includes) so that future work — NPCs making life-altering archetype switches at runtime — has the data model it needs without a refactor.

## Goals

- Give summoned/raised/conjured/charmed companions a path to AI that's smarter than "random pick from combatcommands"
- Make archetypes a reusable unit of AI design that scales across many mobs
- Keep the framework work tight: one new event, one new action, two new YAML fields, resolution-order change in `helpers.go`
- Prove the pattern end-to-end with three concrete mobs before designing more archetypes

## Non-goals

- **Not** extending archetypes beyond companions in this phase (they happen to be companion-only today because that's who we picked to adopt first)
- **Not** building a caster or tank archetype (follow-up phases)
- **Not** converging the old `mobai` tactics system with behavior trees (tactics keep working for mobs not on btrees)
- **Not** adding mid-run archetype swap commands, admin tooling, or world-economy NPC reasoning — the data model supports those, but they are future work
- **Not** building cross-room pathfinding or "complete-cast-then-teleport" companion follow (Option B2 from brainstorm; future)

---

## 1. Architecture Overview

Three new concepts joining the btree framework:

1. **Spell categories** — each spell YAML gains an optional `categories: [...]` field. Free-form strings, no enum. One spell may be in multiple categories. Purely descriptive metadata; the engine doesn't know what they mean — archetypes do.

2. **Archetypes** — a new class of btree file at `_datafiles/world/dogmud/behaviors/archetypes/<name>.yaml`. Structurally identical to existing mob btrees (same node types, same loader, same engine). They live in a separate cache keyed by archetype name.

3. **Mob-level archetype binding** — one new optional field on mob YAML: `behavior_archetype: <archetype-name>`.

**Resolution order when a mob fires a btree event** (new logic in `internal/behaviortree/helpers.go`):

```
per-mob btree file (behaviors/<zone>/<mobid>-<name>.yaml)
  ↓ if not present
archetype tree (if mob YAML has behavior_archetype field)
  ↓ if not present
no tree → caller runs legacy path
```

Negative cache gets two namespaces: `noTree[mobId]` (as today) and `noArchetype[archetypeName]` (new). The Stage 1.8 `TODO(hot-reload)` caveat applies to both.

**No changes to:** the old `mobai` system, the spell casting pipeline, the buff system, the combat loop's core logic (we add one event-firing call site — see §5).

## 2. Spell Categorization

### New field on spell YAML

```yaml
categories:
  - self_defense
```

- Optional. String list. Missing field = untagged = invisible to archetype lookup (normal combatcommands/cast still work).
- Validated only as "list of strings" at load time. Unknown category names are **not** rejected (no registry).

### Category vocabulary for this archetype

| Category | Meaning | Example spells |
|---|---|---|
| `self_defense` | Defensive buff/shield castable on self | `iron-will`, `conviction-ward`, `conviction-armor` |
| `self_offense` | Attack-boosting self-buff | `conviction-surge` |

### Ranking

Spells in a category are ranked at cast time by:

```
score = base_folds × cost
```

Higher score wins. Ties resolved alphabetically by `spellid` for determinism. Both signals are already authored — no new numeric tier field.

### Spells tagged in this phase

- `spells/iron-will.yaml` → `categories: [self_defense]`
- `spells/conviction-ward.yaml` → `categories: [self_defense]`
- `spells/conviction-armor.yaml` → `categories: [self_defense]`
- `spells/conviction-surge.yaml` → `categories: [self_offense]`

## 3. Archetype File: `melee_self_buff`

**Location:** `_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml`

**Format:** structurally identical to existing mob btrees — same YAML schema, same loader, same node types. Only difference is path and cache lookup.

### Tree content

```yaml
name: melee_self_buff
description: |
  Melee specialist who maintains self-buffs first, attacks otherwise.
  Reused by vampire, air elemental, fire elemental, plus any future
  melee-specialist mobs that opt in.

events:
  - mob_combat_round

root:
  type: selector
  children:
    - type: cast_best_in_category
      params:
        category: self_offense
        target: self

    - type: cast_best_in_category
      params:
        category: self_defense
        target: self
```

### Decision flow per round

1. `mob_combat_round` fires for a combatant mob on rounds where it's free to act (not mid-cast, not stunned).
2. Selector tries child 1: `cast_best_in_category(self_offense)`.
   - Filters mob's spellbook to `self_offense` spells
   - Skips candidates whose buff is already active, whose CP cost the mob can't pay, that require a component, that summon/raise/conjure/charm, or when the mob is on the shared cast/special-move cooldown
   - Of remaining candidates, picks highest `base_folds × cost`
   - Success → initiates cast on self; Failure → no viable candidate
3. On Success → selector Success → engine reports `handled=true` → legacy combat swing is skipped this round (the cast's wind-up consumes the round).
4. On Failure → selector tries child 2 (`self_defense`), same semantics.
5. If both children Fail → selector Fails → engine reports `handled=false` → legacy combat swing runs.

### Offense-first rationale

Best defense is a dead enemy. In short fights, defensive buffs never pay off; in long fights they still land by round 9 of the cadence below. Child order is trivially swappable per archetype if a future archetype wants defense-first.

### Realistic fresh-vampire cadence (with 4-round shared cooldown)

| Round | Action | Cooldown remaining |
|---|---|---|
| 1 | Cast conviction-surge (wind-up 1 round) | 4 |
| 2 | Surge resolves; tree fires, cooldown blocks cast → fallthrough → attack | 3 |
| 3 | Attack | 2 |
| 4 | Attack | 1 |
| 5 | Tree fires, cooldown clear → cast iron-will | 4 |
| 6–8 | Wind-up resolves → attacks | counting down |
| 9 | Tree fires, cooldown clear → cast conviction-ward | 4 |
| 10+ | All buffs active, attack loop | — |

Three rounds of "silent" cast wind-ups across 10 rounds to max-buff. Self-balancing: long fights benefit, short fights didn't need the vampire anyway.

## 4. Mob YAML Changes

**Naming clarification:** Existing mob YAMLs have an `archetype:` field (e.g., `archetype: fighting`) controlling **statpool distribution**. The new field is `behavior_archetype:` — deliberately distinct.

### Vampire (`304-vampire.yaml`)

```yaml
behavior_archetype: melee_self_buff   # NEW

spellbook:
  conviction-ward: 6                  # self_defense
  iron-will: 4                        # self_defense
  conviction-surge: 3                 # self_offense (NEW in spellbook)

combatcommands:
  - 'bite'                            # kept — flavor finisher, falls through from archetype
  - ''
  - 'emote moves with impossible speed, closing distance in a single step'
  - ''
  # removed: 'cast conviction-ward' — archetype handles it now
```

### Air Elemental (`312-air_elemental.yaml`)

```yaml
behavior_archetype: melee_self_buff   # NEW

spellbook:                            # NEW (was empty)
  iron-will: 3                        # self_defense
  conviction-ward: 4                  # self_defense

combatcommands:
  - ''
  - 'emote crackles and spins faster, striking from unexpected angles'
  - ''
```

### Fire Elemental (`313-fire_elemental.yaml`)

```yaml
behavior_archetype: melee_self_buff   # NEW

spellbook:                            # NEW (was empty)
  conviction-armor: 3                 # self_defense
  conviction-ward: 3                  # self_defense

combatcommands:
  - ''
  - 'emote flares brightly, flames licking in all directions at once'
  - ''
```

### Coverage matrix

| Mob | self_offense | self_defense | Exercises... |
|---|---|---|---|
| vampire | conviction-surge | ward + iron-will | both branches, multiple candidates per category |
| air elemental | — | ward + iron-will | empty-category fallthrough, multi-candidate defense |
| fire elemental | — | ward + armor | same |

## 5. Engine Changes

All in `internal/behaviortree/` and `internal/hooks/`.

### New event: `mob_combat_round`

- Fires from top of `handleCombatRound` in `internal/hooks/NewRound_DoCombat_unified.go`, once per combatant mob on rounds where it's free to act
- Tree returns Success → engine `handled=true` → legacy combat swing skipped
- Tree returns Failure → engine `handled=false` → legacy combat swing runs as today

### New action: `cast_best_in_category`

Self-gating smart-cast. Pseudocode:

```
1. If mob is on shared cast/special-move cooldown → return Failure
2. Walk mob.spellbook → filter spells where spell.categories contains params.category
3. For each remaining candidate:
     - Skip if spell requires a component (component_tag or summon_component_id set)
     - Skip if spell is a summon/raise/conjure/charm effect
     - Skip if the effect it would grant is already active on target (see below)
     - Skip if mob.conviction < spell.cost
4. If no candidates survive → return Failure
5. Sort surviving candidates by (base_folds × cost) desc, spellid asc for ties
6. InitiateCast(mob, target=<resolved>, spell=top candidate)
7. Return Success
```

**Params:** `category: <string>`, `target: self|owner|enemy` (only `self` wired in this phase; others reserved for later archetypes).

**"Already active" check** has two branches:
- If `spell.buff_ids` non-empty → `mob.HasBuff(id)` for any id
- If `spell.effect_type == "shield"` → check mob's active shield state (Character struct tracks shield since Stage 11.x)

If neither applies (unusual spell shape), log at `mudlog.Warn` and treat "already active" as false (conservative: may recast, won't silently stall).

### New YAML fields

- Mob YAML: `behavior_archetype: <string>` — optional
- Spell YAML: `categories: [<string>, ...]` — optional

### Resolution order implementation (`internal/behaviortree/helpers.go`)

Per §1:
1. `behaviors/<zone>/<mobid>-<name>.yaml` if present
2. Else `behaviors/archetypes/<behavior_archetype>.yaml` if field set
3. Else no tree

Archetype negative cache is a separate map from the per-mob file cache.

### Explicitly not added in this phase

- `mob_on_cast_cooldown` condition — covered internally by the action
- `mob_has_buff_for_category` condition — covered internally by the action
- Archetype inheritance / composition
- Mid-run archetype swap commands

Add when a future archetype needs them.

## 6. Companion Follow & Cast Interrupt

Companion-wide, not archetype-specific. Applies to Summoned, Raised, Conjured, Charmed equally.

### Trigger point

New hook fires after every successful owner movement:
- `go <dir>`
- `recall`
- `fold-recall`
- `portal` / instance portals
- `sable` / admin / scripted teleport

Single call site (post-movement), dedupe-by-design. Iterates the user's `Companions` list, transports each live companion.

### Transport behavior per companion

1. If companion is mid-cast → **abort the cast**, forfeit already-spent conviction (consistent with player self-interrupt)
2. Remove companion from its current room
3. Add to owner's new room
4. Fire normal `mob_enters_room` side-effects (hidden detection roll, room broadcasts)
5. If companion had active aggro on a target not in the new room → `EndAggro`. Otherwise keep aggro live.

### Edge cases

- Companion in same room as destination → no-op
- Companion dead / instance reaped → skip silently
- Destination room doesn't exist → skip, log error
- Owner in an instance the companion wasn't in (e.g., companion left at gate) → transport anyway; access control is owner's problem
- Multiple companions → each transports independently; one failure doesn't cascade

### Broadcast text

- Owner: `Your vampire rejoins you.`
- Departure room: `<name> the vampire follows their summoner.`
- Arrival room: standard mob-arrival text

### Deliberately not built

- Pathfinding / cross-room tracking
- "Complete cast, then teleport" (Option B2 — revisit if playtesting shows wind-up-loss pain)
- Per-companion-type differentiation (charmed wild wolf teleports same as summoned vampire)

## 7. Error Handling & Edge Cases

### Load-time errors

| Failure | Behavior |
|---|---|
| Archetype file missing | Negative-cache name, log at `mudlog.Warn` first time, mobs using it fall through to legacy |
| Archetype file malformed YAML / tree compile error | Log at `mudlog.Error`, negative-cache, legacy fallthrough |
| `behavior_archetype` field not a string | YAML unmarshaller panics at startup (existing pattern) |
| Spell `categories` field not list-of-strings | YAML unmarshaller panics at startup |
| Category name not referenced by any archetype | Silent no-op |

### Runtime errors in `cast_best_in_category`

| Failure | Behavior |
|---|---|
| Spellbook references deleted spellid | Skip silently, log `mudlog.Warn` once per spellid |
| Shield-active detection inconclusive | Log `mudlog.Warn`, treat "already active" as false |
| `InitiateCast` returns error | Return Failure → legacy swing runs this round |
| Panic inside action | Stage 1.8 engine panic recovery handles it, treats as Failure |

### Companion transport errors

| Failure | Behavior |
|---|---|
| Destination room doesn't exist | Skip that companion, log error |
| Companion instance reaped between event + hook | Skip silently (race is expected) |
| Companion in inaccessible instance | Transport anyway — companion follows owner |
| Multiple companions, one transport fails | Others still transport |

### Invariants the code must hold

- `cast_best_in_category` must never return Success without initiating a cast (else legacy swing is suppressed with nothing replacing it → silent round)
- Transport hook fires exactly once per owner movement regardless of path
- Archetype negative cache is separate from per-mob-file negative cache

### Deliberately not added

- Startup cross-validation that every `behavior_archetype` points at an existing archetype file (future CI lint)
- Category registry / typo detection (cost of typo is "spell becomes archetype-invisible" — caught in playtest or tests)

## 8. Testing Strategy

### Unit tests — `internal/behaviortree/`

**New: `action_cast_best_in_category_test.go`**
- Empty category → Failure
- Category populated, no spellbook match → Failure
- Single valid candidate → Success + correct `InitiateCast`
- Multiple candidates → picks highest `base_folds × cost`, ties broken by `spellid` asc
- Candidate buff already active → skipped
- Candidate shield already active → skipped
- Mob lacks CP → skipped
- Mob on cooldown → Failure before spellbook walk
- Candidate has `component_tag` → skipped
- Candidate is summon → skipped
- Deleted spellid → skipped, warn logged once

**New: `engine_archetype_test.go`**
- Per-mob file + archetype both declared → per-mob wins
- Per-mob absent + archetype declared → archetype loads
- Both absent → no tree, `handled=false`
- Archetype file missing → negative cache + second call skips disk
- Archetype file malformed → logged, negative-cached, `handled=false`

**Extend `helpers_test.go`** — archetype and per-mob negative caches are independent.

### Unit tests — companion follow

**Extend `go_test.go`** (or `movement_test.go` — whichever matches the established pattern):
- Normal `go` → companion moved
- Companion mid-cast → cast aborted, transport completes
- Companion in same room → no-op
- Companion dead → skipped
- Destination doesn't exist → graceful skip, error logged
- Two companions, one fails → other still transports

### Integration-shape tests

Using `behaviortree.SetMobTreeForTest` + `MinimalCombatMessageFixture` (Stage 1.6 / 1.2a harnesses):

**New: `melee_self_buff_archetype_integration_test.go`**
- Fresh vampire in combat → round 1 initiates `conviction-surge`
- Vampire with surge active → initiates `iron-will`
- Vampire with surge + iron-will active → initiates `conviction-ward`
- Vampire fully buffed → falls through to legacy combat
- Air elemental → initiates `conviction-ward` (skips empty self_offense)

### Regression guarantees

- Mob with no `behavior_archetype` and no per-mob btree → behaves identically to today
- Existing btree mobs (Edrin, Sable, Sylara, Rhett, bandit_leader, etc.) continue to work — new `mob_combat_round` event fires but their trees don't listen
- Legacy `tactics:` / `tactic_preset:` / mobai system untouched

### Manual smoke (pre-merge)

1. Summon vampire, attack a dummy, verify ~10-round buff cadence
2. Summon air elemental → casts `conviction-ward` on first engagement
3. Summon vampire, move rooms mid-cast → cast aborts, vampire rejoins
4. Charm a spell-less wild mob → no crashes, attacks normally
5. Instance portal with vampire → follows into and out of instance

### Out of scope for automated testing

- Playtest tuning of the 10-round cadence (feedback loop, not a test)
- Load-testing extra per-round event dispatch (cheap — one map lookup per combatant)

---

## Out of Scope / Future Work

- Caster archetype (wraith, spectre)
- Tank/taunter archetype (golem, earth elemental, magma elemental)
- "Cheap rent" no-buff melee archetype for skeleton, zombie, wolf, hive swarm, water elemental (or leave them on legacy path — decide when we tackle them)
- Owner-awareness conditions (`owner_health_below`, `owner_in_room`, etc.) — deferred until needed
- Archetype inheritance / composition
- Mid-run archetype swap command (admin tool)
- World-economy NPC reasoning that switches archetypes mid-play
- "Complete cast, then teleport" companion follow (Option B2 from brainstorm)
- CI lint validating `behavior_archetype` references resolve
- Convergence of legacy `mobai` tactics and behavior-tree archetypes
- Player-facing UI additions for companions (Phase 5)

## Summary of Deliverables

### New files
- `_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml`
- `internal/behaviortree/action_cast_best_in_category.go` (+ test)
- `internal/behaviortree/engine_archetype_test.go`
- `internal/behaviortree/melee_self_buff_archetype_integration_test.go`

### Modified files
- `internal/behaviortree/helpers.go` — resolution order + archetype negative cache
- `internal/behaviortree/engine.go` — archetype tree load/cache (may also touch `loader.go`)
- `internal/behaviortree/actions_combat.go` — register `cast_best_in_category`
- `internal/hooks/NewRound_DoCombat_unified.go` — fire `mob_combat_round`
- `internal/usercommands/go.go` (+ other movement paths) — companion follow hook
- `internal/characters/` — add transport helper used by the follow hook
- `internal/mobs/mobs.go` — new `BehaviorArchetype` field
- `internal/spells/` — new `Categories` field on spell struct
- `_datafiles/world/dogmud/mobs/summons/304-vampire.yaml`
- `_datafiles/world/dogmud/mobs/summons/312-air_elemental.yaml`
- `_datafiles/world/dogmud/mobs/summons/313-fire_elemental.yaml`
- `_datafiles/world/dogmud/spells/iron-will.yaml`
- `_datafiles/world/dogmud/spells/conviction-ward.yaml`
- `_datafiles/world/dogmud/spells/conviction-armor.yaml`
- `_datafiles/world/dogmud/spells/conviction-surge.yaml`
