# Mob Aliveness 4.3 — Goal Types Catalog Design

**Date:** 2026-05-27
**Status:** Approved (design)
**Roadmap position:** Phase 4 (strategic layer). After 4.3: 25/42.
**Depends on:** 4.1 (goal substrate — shipped), 4.2 (goal selection — shipped).
**Next chunks:** 4.4 Strategic→tactical translation → 4.5 Reactive goal generation → 4.6 Goal satisfaction & pruning.

---

## Summary

Populates the chunk-4.1 goal substrate + chunk-4.2 selection engine with a
concrete catalog of 13 goal types covering survival, wealth, revenge,
protection, social, mastery, and exploration. Each type ships a Predicate
(when satisfied), a ContextScore (current relevance multiplier), a
declarative ParamSchema (validated at `Add` time), and — where multi-instance
makes sense — an AllowMultiple flag plus DedupKey func.

The chunk also ships three engine deltas (ParamSchema validation,
AllowMultiple + DedupKey, archetype lazy-seed sentinel) and sparse archetype
defaults so the system produces observable selection state out of the box.

**No btree integration yet** — 4.4 wires the selected goal into tactical
execution. Observable change in 4.3: most loaded mobs will have a current
goal (usually `survival`), visible via `goal current <mob>`, and the
`goals.switch` debug log fires when survival kicks in during combat.

---

## Goals

- Author 13 narrow goal types per the brainstorming decision (broad-types
  rejected — narrow keeps per-type tunability + clean conflict encoding).
- Declarative ParamSchema validates type-specific params at `Add` time, so
  Predicates can assume well-typed params at read time.
- AllowMultiple + DedupKey enable multi-instance types (multiple revenge
  targets, multiple training goals, etc.) without bespoke per-Predicate
  dedup logic.
- Archetype YAML `default_goals:` block + lazy seeding on first MobGoals
  access. Sentinel field on the file prevents re-seeding after admin
  `Clear`.
- Sparse defaults: `survival` for every combat-capable archetype + generic
  `wealth-gold` for thief/shopkeeper. Mob-specific param defaults defer to
  4.5 reactive seeding or a future per-mob YAML chunk.
- Stay decoupled: catalog lives in `internal/goals/catalog/` subpackage and
  imports factions/opinions/skills/etc. without forcing those into the
  `internal/goals/` substrate.
- Cross-type conflict mechanism deferred (decision in §7). Selector + sensible
  seeders cover it; revisit if 4.5 testing surfaces real issues.

## Non-goals

- Behavior-tree integration / tactical execution — 4.4.
- Reactive goal generation from events — 4.5.
- Goal satisfaction sweep / pruning — 4.6.
- Per-mob YAML defaults (override or supplement archetype) — deferred.
- Cross-type conflict mechanism (`ConflictsWithFn` hook) — deferred.
- Param templating in archetype defaults (`{{mob.faction_id}}` etc.) —
  deferred.
- Goal-hierarchy / sub-goals — Phase 5+ concept; emergent
  "craft-item triggers mastery-skill" behavior comes from 4.5 reactive
  seeding rules, not engine hierarchy.
- Bespoke boss-mob defaults — defer with per-mob YAML.
- `gossip` goal type — dropped per §8 (existing working system).
- `goal reseed` admin command — workaround is delete the YAML file.

---

## 1. Architecture & data flow

```
internal/goals/                    (4.1 substrate + 4.2 selection)
    context.md                       NEW — close documentation gap from 4.1/4.2
internal/goals/catalog/            (NEW — 4.3)
    context.md                       package overview (per project convention)
    catalog.go                       package-level helpers
    survival.go                      one file per type, init() registers
    wealth_gold.go
    wealth_item.go
    craft_item.go
    revenge_mob.go
    revenge_faction.go
    protection_mob.go
    protection_faction.go
    befriend.go
    befriend_faction.go
    mastery_skill.go
    mastery_equip.go
    visit_zone.go
```

Each `<type>.go` file:
- Defines its Predicate + ContextScore + (if AllowMultiple) DedupKey funcs.
- Calls `goals.RegisterGoalType("name", GoalTypeMeta{...})` in `init()`.
- Carries its own unit tests (per-type test file alongside).

`main.go` adds a blank import to fire all the inits:
```go
import _ "github.com/GoMudEngine/GoMud/internal/goals/catalog"
```

**Lazy archetype seeding** is the only new runtime path. When
`loadOrLazyInit(mobId, namesimple)` creates a fresh in-memory MobGoals
(no file on disk):

1. Resolve the mob template's `BehaviorArchetype` field.
2. Look up the archetype's `default_goals:` block (new field in archetype
   YAML, parsed alongside `tree:` + `goal_weights:`).
3. For each default, build a `*Goal` and route through `goals.Add` (so
   ParamSchema validation + dedup fire normally).
4. Flip `MobGoals.SeededFromArchetype = true`.
5. `saveToDisk` once.

Subsequent calls observe `SeededFromArchetype == true` and skip the seed
path. `Add`/`Remove`/`Clear` never touch the sentinel — once seeded, the
file is admin-/event-owned.

---

## 2. API surface

```go
package goals

// GoalTypeMeta gains three fields (existing fields kept).
type GoalTypeMeta struct {
    Predicate     PredicateFn       // 4.1
    ConflictsWith []string          // 4.1 (type-name-based; see §7)
    ContextScore  ContextScoreFn    // 4.2

    // chunk 4.3:
    Params        []ParamSchema     // declarative per-type param schema
    AllowMultiple bool              // true → coexist; same DedupKey → conflict
    DedupKey      func(g *Goal) string // when AllowMultiple, returns dedup key
}

// ParamSchema describes one expected key on Goal.Params.
type ParamSchema struct {
    Key      string
    Required bool
    GoType   string // "int" | "string" | "[]string" | "float64" | "bool"
}

// ValidateParams checks g.Params against the registered type's schema.
// Returns ErrBadParams (with key + expected type) on failure.
// Called by Add before conflict check.
func ValidateParams(g *Goal, schema []ParamSchema) error

// MobGoals gains one field.
type MobGoals struct {
    // ...4.1 + 4.2 fields...
    SeededFromArchetype bool `yaml:"seeded_from_archetype,omitempty"`
    Goals               []*Goal `yaml:"goals"`
}
```

Behavior-tree side gains nothing new — 4.2's `LoadArchetypeYAMLFromFile`
extends to read `default_goals:` and expose via a new accessor:

```go
package behaviortree

// GetArchetypeDefaultGoals returns the parsed default_goals list for an
// archetype (or nil if absent). Wired through main.go via a registered
// callback into goals.SetArchetypeDefaultsLookup, mirroring the 4.2
// SetWeightsLookup pattern.
func (e *Engine) GetArchetypeDefaultGoals(name string) []GoalDefault

type GoalDefault struct {
    Type     string         `yaml:"type"`
    Priority int            `yaml:"priority"`
    Params   map[string]any `yaml:"params,omitempty"`
}
```

Goals package gains the symmetric registration:

```go
package goals

// ArchetypeDefaultsLookupFn returns the archetype's default goal list for
// the given mob. Registered once at boot from main.go to avoid the goals →
// behaviortree import cycle.
type ArchetypeDefaultsLookupFn func(mob *mobs.Mob) []GoalDefault

// GoalDefault is the goals-package mirror of behaviortree.GoalDefault
// (kept separate to avoid the cycle).
type GoalDefault struct {
    Type     string
    Priority int
    Params   map[string]any
}

func SetArchetypeDefaultsLookup(fn ArchetypeDefaultsLookupFn)
```

`Add`'s conflict check changes:
```
For each existing goal e on the mob:
  if e.Type == g.Type:
    if meta.AllowMultiple:
      if meta.DedupKey != nil && meta.DedupKey(e) == meta.DedupKey(g):
        → conflict (priority compare)
      else:
        coexist
    else:
      → conflict (priority compare)
  else if e.Type in newMeta.ConflictsWith or g.Type in eMeta.ConflictsWith:
    → conflict (priority compare)
```

---

## 3. Engine deltas

### 3.1 ParamSchema validation

`ValidateParams(g, schema)` walks the schema:
- For each `Required: true` entry, `g.Params[key]` must exist.
- The value must satisfy the declared `GoType` via a type-switch
  (`int` → `int` or `int64`; `string` → `string`; `[]string` →
  `[]interface{}` of strings or `[]string`; `float64` → `float64` or
  convertible from int; `bool` → `bool`).

On failure, returns `ErrBadParams{Key, ExpectedType, GotType}`. Admin sees
a readable error. 4.5 reactive seeders log and skip.

Schema-less types (current 4.1 behavior) pass through unchanged — types
without a Params block on their GoalTypeMeta skip validation. New types
should always declare a schema.

### 3.2 AllowMultiple + DedupKey

Strict-default: `AllowMultiple: false` keeps 4.1's "same type always
conflicts" semantics. Types opt in by setting `AllowMultiple: true` and
providing a `DedupKey` func.

DedupKey is `func(g *Goal) string`. Convention: return a stable string
derived from the goal's identifying params (e.g.,
`"mob:5"` for revenge-mob targeting mobInstanceId=5).
Two goals of the same type with equal DedupKey strings collide and
priority-compare; with different keys they coexist.

If `AllowMultiple: true` is set but `DedupKey: nil`, multiple goals coexist
freely with no dedup — useful for goals that should never collapse (rare,
but available). The catalog uses DedupKey for every multi-instance type.

### 3.3 Archetype lazy seeding

`loadOrLazyInit(mobId, namesimple)`:

```go
if cached, ok := cache[mobId]; ok { return cached }
if mg := loadFromDisk(mobId, namesimple); mg != nil {
    cache[mobId] = mg
    return mg
}
// Fresh — create and seed.
mg := &MobGoals{MobId: mobId, NextGoalId: 1}
cache[mobId] = mg
seedFromArchetype(mobId, namesimple)
return mg
```

`seedFromArchetype`:

1. Look up the mob instance via `mobs.GetMobSpec(MobId(mobId))` — read
   template's `BehaviorArchetype`.
2. Call the registered `archetypeDefaultsLookup` (set by `main.go`).
3. For each `GoalDefault`, call `Add(mobId, namesimple, &Goal{Type:..., Priority:..., Params:...})`.
   Any `Add` failure (ErrBadParams, conflict) logs a warning and continues.
4. Mark `mg.SeededFromArchetype = true` regardless of seed result (don't
   retry on subsequent calls).
5. `saveToDisk` once.

The eager Recompute that Add fires (from chunk 4.2) selects the seeded
top goal naturally — `current_goal_id` lands in the file in the same
write.

**Backward compat:** existing 4.1/4.2 files load with
`SeededFromArchetype = false`. First post-deploy `GoalsOf` on such a file
triggers seeding — for most mobs this means `survival` gets prepended to
their existing goal list. This is intentional. Operators who want to
opt-out per-file can hand-edit `seeded_from_archetype: true` into the
file before deploy.

---

## 4. The 13-type catalog

### 4.1 survival

- **Purpose:** Flee/heal when HP drops near critical, until safe.
- **Params:**
  - `safe_threshold_pct: int` (optional, default 60)
  - `flee_threshold_pct: int` (optional, default 25)
- **AllowMultiple:** no. **DedupKey:** —.
- **Predicate:** `hp_pct ≥ safe_threshold_pct AND not in combat`.
- **ContextScore:**
  - `hp_pct > safe_threshold_pct` → 0 (filtered)
  - `flee_threshold_pct ≤ hp_pct ≤ safe_threshold_pct` → linear from
    1.0 (at safe) to 3.0 (at flee)
  - `hp_pct < flee_threshold_pct` → 5.0
- **ConflictsWith:** — (coexists; selector arbitrates).
- **State read:** `mob.Character.Health`, `mob.Character.HealthMax`,
  in-combat status.

### 4.2 wealth-gold

- **Purpose:** Accumulate gold to a target amount.
- **Params:**
  - `target: int` (required, > 0)
- **AllowMultiple:** no. **DedupKey:** —.
- **Predicate:** `mob.Character.Gold ≥ target`.
- **ContextScore:** baseline 1.0; scales with `(target - gold) / target`
  to a max of 2.0. Drops to 0 when predicate satisfied.
- **ConflictsWith:** —.
- **State read:** `mob.Character.Gold`.

### 4.3 wealth-item

- **Purpose:** Acquire a specific item.
- **Params:** exactly one of:
  - `item_tag: string` (matches `ItemSpec.ComponentTag` or similar tag fields)
  - `item_id: int`
- **AllowMultiple:** yes. **DedupKey:**
  `"tag:"+item_tag` or `"id:"+strconv.Itoa(item_id)`.
- **Predicate:** item with matching tag/id is in `mob.Character.Items` or
  any equipped slot.
- **ContextScore:** 0 if present. 1.0 if absent. +0.5 bump if the item is
  sold in any shop in the mob's current zone (scan via `shops.InZone`).
- **ConflictsWith:** —.
- **State read:** inventory + equipment + zone shop registry.

### 4.4 craft-item

- **Purpose:** Produce a specific recipe.
- **Params:**
  - `recipe_id: string` (required)
- **AllowMultiple:** yes. **DedupKey:** `recipe_id`.
- **Predicate:** output item of the recipe is in inventory.
- **ContextScore:**
  - Recipe not in `mob.Character.KnownRecipes` → 0 (filtered)
  - Skill rank below recipe's required minimum → 0.3 (very low; a coexisting
    `mastery-skill` goal wins; the planner trains first)
  - Known + skilled + materials missing → 1.0
  - Known + skilled + materials on hand → 2.0
- **ConflictsWith:** —.
- **State read:** `KnownRecipes`, `Skills`, inventory, recipe registry.

### 4.5 revenge-mob

- **Purpose:** Kill or witness death of a specific tormentor.
- **Params:**
  - `target_kind: string` (required, "mob" | "player")
  - `target_id: int` (required, > 0 — mob template id or user id)
- **AllowMultiple:** yes. **DedupKey:**
  `target_kind + ":" + strconv.Itoa(target_id)`.
- **Predicate:** target is dead (mob: any instance gone for ≥ N rounds, or
  player: in death-flag state).
- **ContextScore:**
  - Target not seen by this mob's CombatMemory in last 1000 rounds → 0
  - Target currently in same room → 2.0
  - Target in adjacent room → 1.5
  - Target elsewhere in zone → 0.5
  - Target out of zone → 0.1
- **ConflictsWith:** — (cross-type with befriend deferred per §7).
- **State read:** mobs/users target lookup, `mob.CombatMemory`.

### 4.6 revenge-faction

- **Purpose:** Inflict N kills against a faction's members.
- **Params:**
  - `faction_id: string` (required)
  - `target_kill_count: int` (required, > 0)
- **AllowMultiple:** yes. **DedupKey:** `faction_id`.
- **Predicate:** per-mob `mob.MiscData["faction_kills_inflicted:"+faction_id] ≥ target_kill_count`.
  Counter is incremented by a 4.5 reactive hook on kill events (4.3 only
  defines the read-path; 4.5 ships the writer).
- **ContextScore:**
  - No faction members in current zone → 0
  - 1+ faction members in zone → 1.0 + 0.1 × member_count, capped at 2.0
- **ConflictsWith:** — (cross-type with befriend-faction / protection-faction
  deferred per §7).
- **State read:** `factions.IsMember`, mob MiscData, zone mob scan.

### 4.7 protection-mob

- **Purpose:** Keep a named ally alive; intervene when they're attacked.
- **Params:**
  - `target_kind: string` (required, "mob" | "player")
  - `target_id: int` (required, > 0)
- **AllowMultiple:** yes. **DedupKey:**
  `target_kind + ":" + strconv.Itoa(target_id)`.
- **Predicate:** never satisfied (ongoing). 4.6 will remove the goal if
  the target has been dead for ≥ N rounds via its pruning sweep.
- **ContextScore:**
  - Target dead → 0
  - Target in combat anywhere → 2.5
  - Target in same room (not in combat) → 1.5
  - Target in same zone → 0.8
  - Target in different zone → 0.2
- **ConflictsWith:** —.
- **State read:** target combat status, room, zone.

### 4.8 protection-faction

- **Purpose:** Defend faction members in the current zone from hostile mobs.
- **Params:**
  - `faction_id: string` (required)
- **AllowMultiple:** yes. **DedupKey:** `faction_id`.
- **Predicate:** never satisfied (ongoing).
- **ContextScore:**
  - No faction members in current zone → 0
  - Any faction member in combat in zone → 2.0
  - Hostile mobs present in zone but no member in combat → 1.0
  - Zone calm → 0.3
- **ConflictsWith:** — (cross-type with revenge-faction deferred per §7).
- **State read:** `factions.Members`, zone mob scan with combat-state check.

### 4.9 befriend

- **Purpose:** Raise opinion with a specific target above threshold.
- **Params:**
  - `target_kind: string` (required, "mob" | "player")
  - `target_id: int` (required, > 0)
  - `opinion_threshold: int` (optional, default 60)
- **AllowMultiple:** yes. **DedupKey:**
  `target_kind + ":" + strconv.Itoa(target_id)`.
- **Predicate:** `opinions.Of(mobId, target_kind, target_id) ≥ opinion_threshold`.
- **ContextScore:**
  - Target not in same zone → 0
  - Target in same room → 1.5
  - Target in same zone (different room) → 0.8
- **ConflictsWith:** — (cross-type with revenge-mob deferred per §7).
- **State read:** opinions (1.1), target location.

### 4.10 befriend-faction

- **Purpose:** Raise rep with a faction above threshold.
- **Params:**
  - `faction_id: string` (required)
  - `rep_threshold: int` (optional, default 60)
- **AllowMultiple:** yes. **DedupKey:** `faction_id`.
- **Predicate:** `factions.GetRep(mobId, faction_id) ≥ rep_threshold`.
- **ContextScore:**
  - No faction members in zone → 0
  - Members in zone → 1.0 + 0.1 × (rep_threshold - current_rep) / threshold,
    capped at 1.8 (scales mildly with rep gap)
- **ConflictsWith:** — (cross-type deferred per §7).
- **State read:** factions.

### 4.11 mastery-skill

- **Purpose:** Train a named skill to a target rank.
- **Params:**
  - `skill_name: string` (required — must match a registered skill)
  - `target_rank: int` (required, > 0)
- **AllowMultiple:** yes. **DedupKey:** `skill_name`.
- **Predicate:** `mob.Character.Skills[skill_name].Rank ≥ target_rank`.
- **ContextScore:**
  - Rank already ≥ target → 0 (satisfied; predicate fires next tick)
  - No training opportunity in zone (no shops/stations/spawns matching the
    skill's training context) → 0.2 (low — planner can wander to find one)
  - Opportunity in zone → 1.0
  - Opportunity in current room → 2.0
  - Scaling: multiply by `(1.0 - rank/target_rank) × 0.5 + 0.5` so closer
    to target = less urgent (range 0.5 at target down to 1.0 at rank=0)
- **ConflictsWith:** —.
- **State read:** `mob.Character.Skills`, skill training-context heuristics
  (per-skill table mapping skill name → training context kind).

### 4.12 mastery-equip

- **Purpose:** Upgrade a specific equipment slot to rarity tier ≥ N.
- **Params:**
  - `slot: string` (required — one of weapon/offhand/head/body/legs/feet/etc.)
  - `min_rarity_tier: int` (required)
- **AllowMultiple:** yes. **DedupKey:** `slot`.
- **Predicate:** equipped item in slot has `rarity_tier ≥ min_rarity_tier`.
  Items lacking a `rarity_tier` field treated as tier 50 (per engine
  fallback) — Predicate uses the fallback.
- **ContextScore:**
  - Predicate satisfied → 0
  - Slot empty → 1.5 (very motivated)
  - No shop sells items for this slot in zone → 0.3 (low — planner can
    wander)
  - Shop in current room → 1.5
  - Shop in zone → 1.0
- **ConflictsWith:** —.
- **State read:** equipment slot inspection, shop inventory in zone.

  *Note:* Live tuning will be patchy until the deferred
  `project_rarity_tier_audit` lands (145/213 items lack rarity tags;
  engine treats them as tier 50). Predicate is well-defined; ContextScore
  behavior is correct against tagged items. Recommend authors avoid this
  type on mobs in zones with many untagged items until the audit ships.

### 4.13 visit-zone

- **Purpose:** Visit a named zone for the first time.
- **Params:**
  - `target_zone: string` (required — must match a loaded zone)
- **AllowMultiple:** yes. **DedupKey:** `target_zone`.
- **Predicate:** `mob.VisitedZones[target_zone] == true` (new mob-instance
  state, set in the room-change hook).
- **ContextScore:**
  - Predicate satisfied → 0
  - Mob currently in target_zone → 0 (predicate fires next tick on the
    room-change hook firing)
  - No path known from current zone → 0 (defensive)
  - Target zone is adjacent (1 hop via zone-graph) → 1.5
  - 2–3 hops → 0.8
  - 4+ hops → 0.3
- **ConflictsWith:** —.
- **State read:** `mob.VisitedZones` (new field on `*mobs.Mob` instance,
  persisted via `mobs.instances/`), zone-graph distance via existing
  `rooms.GetZoneConfig` / adjacency helpers.

---

## 5. Archetype defaults

Per the brainstorming decision: sparse defaults limited to param-free or
generic-param goals. Mob-specific param defaults defer to 4.5 reactive
seeding and a future per-mob YAML chunk.

### 5.1 Default goals by archetype

| Archetype | Default goals |
|-----------|---------------|
| `ambusher` | `survival` |
| `combat_passive` | `survival` |
| `defensive_caster` | `survival` |
| `forager` | `survival` |
| `generic_fighter` | `survival` |
| `leader` | `survival` |
| `lookout` | `survival` |
| `melee_self_buff` | `survival` |
| `predator` | `survival` |
| `prey` | `survival` |
| `pure_caster` | `survival` |
| `scout` | `survival` |
| `support_caster` | `survival` |
| `tank_taunter` | `survival` |
| `thief` | `survival`, `wealth-gold (target=500, priority=40)` |
| `noncombat_shopkeeper` | `survival`, `wealth-gold (target=1000, priority=30)` |
| `noncombat_passive` | — |
| `noncombat_questgiver` | — |
| `boss_chrysalis_phantom` | — |
| `boss_edrin` | — |
| `boss_rhett` | — |
| `boss_soren` | — |
| `boss_sylara` | — |

`survival` is authored with default thresholds (no params block needed —
`safe_threshold_pct: 60`, `flee_threshold_pct: 25` baked into the type).

### 5.2 Archetype YAML schema

```yaml
tree:
  type: ...
goal_weights:        # chunk 4.2
  ...
default_goals:       # chunk 4.3
  - type: survival
    priority: 80
  - type: wealth-gold
    priority: 40
    params:
      target: 500
```

Parsing: `behaviortree.LoadArchetypeYAMLFromFile` extends to read this
block and exposes it via `GetArchetypeDefaultGoals(name)`. `main.go`
registers the goals-side lookup adapter mirroring the 4.2
`SetWeightsLookup` pattern.

**Validation:** each entry's `type` must match a registered goal type;
unknown types log a single-line warning at archetype-load time and skip.
Params are validated against the type's ParamSchema at seed time (in `Add`).

---

## 6. Persistence schema delta

One new field on `MobGoals`:

```yaml
mob_id: 371
next_goal_id: 4
current_goal_id: g2
current_since_round: 12450
last_switch_round: 12450
seeded_from_archetype: true   # NEW — 4.3 sentinel
goals:
  - id: g1
    type: survival
    priority: 80
    ...
```

```go
type MobGoals struct {
    MobId               int     `yaml:"mob_id"`
    NextGoalId          int     `yaml:"next_goal_id"`
    CurrentGoalId       string  `yaml:"current_goal_id,omitempty"`
    CurrentSinceRound   uint64  `yaml:"current_since_round,omitempty"`
    LastSwitchRound     uint64  `yaml:"last_switch_round,omitempty"`
    SeededFromArchetype bool    `yaml:"seeded_from_archetype,omitempty"` // NEW 4.3
    Goals               []*Goal `yaml:"goals"`
}
```

**Backward compat:** existing 4.1/4.2 files load with sentinel `false`.
First post-deploy `GoalsOf` triggers the seed path. For most mobs this
appends `survival` to their existing goal list. Operators who want to
opt-out per-file can hand-edit `seeded_from_archetype: true` before
deploy.

**No `goal reseed` admin command.** Workaround: delete the YAML file and
the engine treats the mob as fresh on next access. Marginal utility,
extra surface area; skip for 4.3.

---

## 7. Cross-type conflict policy

4.3 ships **no cross-type conflict mechanism**. Three pairs look like they
should conflict by param:
- `befriend` ↔ `revenge-mob` (same target)
- `befriend-faction` ↔ `revenge-faction` (same faction)
- `protection-faction` ↔ `revenge-faction` (same faction)

4.1's `ConflictsWith []string` is type-name-based and can't see params.
Adding a `ConflictsWithFn(a, b *Goal) bool` hook is doable (~10 lines)
but rejected for 4.3 because:

1. **The selector handles it.** If both ever coexist on a mob, scoring
   picks one and the other waits. They never both fire planners
   simultaneously (4.4 reads only the *current* goal).
2. **Sensible 4.5 reactive seeders won't seed the bad pairs.** A
   "befriend on save" hook checks for an existing revenge goal first.
3. **Admin manual nonsense is the admin's problem.** Admin testing
   workflows shouldn't drive engine design.

If real issues surface during 4.5 testing, add `ConflictsWithFn` then.
Cheaper than over-building 4.3.

---

## 8. Note on dropped `gossip` type

The brainstorming pass initially proposed a 14th type `gossip` (deliver a
topic to a relationship partner). It was dropped because DOGMud already
has a working gossip/news/rumors system:

- **Hook:** `internal/hooks/MobIdle_HandleIdleMobs.go` —
  `buildGossipLine(mob)` fires from `MobIdle` for gossiper-tagged mobs.
- **Content sources:** `worldevents.GetRecentWorldEvents` (filtered by
  zone/region/significance) blended with `facts.KnownFactsOf(mobId)` (the
  chunk 1.7 world-facts substrate), 70/30 split.
- **Templates:** `_datafiles/world/dogmud/gossip_templates.yaml`, keyed
  by `EventType-Significance-Distance` with `fact-{id}`/`fact-{tag}`/
  `fact-default` fallback chains.
- **Dedup:** `facts.HeardEvent` / `facts.RecordHeardEvent` — per-mob,
  per-event, persisted via the 1.7 substrate.
- **Mob authoring:** ~7 mobs use it today (old_wrex, old_fen, old_gobb,
  old_cottager_gyda, old_fisherman_hodder, barmaid_neva, constable_drunn).
- **Direction:** broadcast (room-level utterance, anyone in earshot hears).

A goal-driven `gossip` type would have been *conceptually different*
(directed-to-partner vs broadcast-in-room) but practically muddy:

- Both produce "did you hear about X?" lines. Player can't distinguish.
- Both use the same English word — admin output `goal current
  barmaid_neva: gossip` is ambiguous to anyone who knows the existing
  system.
- A new MiscData dedup mechanism would run in parallel with the existing
  `facts.HeardEvent` substrate — exactly the duplication that ages
  badly.

If a goal-driven directed-gossip mechanism turns out to be genuinely
useful, it lands as part of the (separately tracked) gossip-system
refinement chunk, designed coherently with the existing mechanism rather
than alongside it. 4.3 leaves the existing system untouched and ships
`befriend` + `befriend-faction` for the "social" category.

---

## 9. Edge cases

| # | Case | Behavior |
|---|---|---|
| 1 | ParamSchema validation fails at Add | `Add` returns `ErrBadParams{Key, ExpectedType, GotType}`. Admin sees readable error. 4.5 reactive seeders log + skip. |
| 2 | Type referenced in archetype YAML's `default_goals:` is not registered | Loader logs single-line warning at archetype load, skips that entry. Other defaults seed normally. Server boots. |
| 3 | DedupKey panics (author bug) | Wrapped in panic-recovery (mirrors 4.2 ContextScore wrapper). Logs warning + returns empty string (collapses to "no key" — same-type dedup falls through to AllowMultiple semantics). One bad type does not crash Add. |
| 4 | Archetype seed encounters a Predicate that's already satisfied (e.g., survival on a mob at full HP) | Goal still gets added — Predicate is checked at *selection* time, not Add time. Selector filters it out via ContextScore=0 until HP drops. Correct behavior. |
| 5 | Existing 4.1/4.2 file with no SeededFromArchetype field, mob has 0 existing goals | First GoalsOf seeds archetype defaults, sentinel flips, persists. Behaves like a fresh mob. |
| 6 | Existing file with no SeededFromArchetype field, mob has existing admin-added goals | First GoalsOf appends defaults to existing goals. This is the deploy-day behavior — operators get a one-line warning in the spec deploy notes. Workaround: hand-edit sentinel before deploy. |
| 7 | Admin Clears a mob's goals | Goals list zeroed, sentinel preserved. No re-seeding on subsequent access. Cleared file stays empty until admin manually re-adds or deletes the file. |
| 8 | `revenge-faction` Predicate fires for the very first time | `mob.MiscData["faction_kills_inflicted:..."]` defaults to 0 on first read. Predicate returns false. ContextScore fires zone scan normally. 4.5 hook will start incrementing on kill events. |
| 9 | `visit-zone` target_zone doesn't exist | At Add time, ParamSchema validates string type only — not zone-existence. If `target_zone` is bogus, Predicate never fires (zone never visited), ContextScore returns 0 (no path), goal is dead weight until removed. 4.5 / admin authoring should validate zone names against `rooms.GetAllZones()` at creation time. (4.3 could add this validation; deferred — failure mode is silent goal-rot, not a crash.) |
| 10 | `craft-item` recipe_id refers to a recipe the mob's species can't ever learn | Predicate never fires (recipe not in KnownRecipes). ContextScore returns 0 (filtered). Dead weight. Authoring problem. |
| 11 | `mastery-equip` slot string is invalid (typo'd "head" as "hed") | Predicate inspects `mob.Character.Equipment[slot]` which returns nil → predicate false. ContextScore returns 1.5 (slot "empty"). Goal pursues forever. Authoring problem; consider validating slot names against an enum at Add time. (4.3 could add this; deferred.) |
| 12 | Mob template loaded with `BehaviorArchetype: ""` (no archetype) | `seedFromArchetype` short-circuits — no archetype lookup. Sentinel still flips to `true`. File persists with empty goals. Mob has no current goal until admin or 4.5 adds one. |
| 13 | Two mobs share the same template (rare; per chunk 4.2's shared-template note) | Both seed once from same archetype defaults (the first to call GoalsOf seeds; the second sees the sentinel and reads the persisted file). Shared-template selection state limitation from 4.2 still applies. |

---

## 10. Testing strategy & rollout

### 10.1 Per-type unit tests

Each `internal/goals/catalog/<type>_test.go` covers:

- **Registration:** type appears in registry after init.
- **ParamSchema:** valid params pass; missing required key fails; wrong
  Go-type fails; default-value omission preserves zero (or default).
- **Predicate truth table:** at least one case per branch — satisfied,
  not satisfied, edge cases per the type's semantics.
- **ContextScore boundary curves:** at least 3 values along the curve
  per type (e.g., for `survival`: hp=80 returns 0, hp=40 returns ~1.5,
  hp=10 returns 5.0).
- **DedupKey (if AllowMultiple):** two goals with same key collide; two
  goals with different keys coexist; goal without required dedup-params
  produces stable error or empty key.

Estimated 10–20 tests per type × 13 types = ~150–250 tests.

### 10.2 Engine integration tests

`internal/goals/store_test.go` (extend):

- `Add` with AllowMultiple=true + matching DedupKey → conflict error
  (priority compare).
- `Add` with AllowMultiple=true + non-matching DedupKey → both coexist
  in `Goals` slice.
- `Add` with ParamSchema validation failure → `ErrBadParams` returned;
  goal not added.
- `loadOrLazyInit` for fresh mob with archetype defaults → seeds, sets
  sentinel, persists.
- `loadOrLazyInit` for fresh mob with no archetype → sentinel still
  flips, no seeds.
- `loadOrLazyInit` for existing file with sentinel=true → no seeding
  (idempotent).
- `Clear` preserves sentinel; subsequent `GoalsOf` does NOT re-seed.

### 10.3 Archetype loader tests

`internal/behaviortree/engine_default_goals_test.go` (new):

- `default_goals:` parses to expected struct list.
- Missing `default_goals:` field → empty list, no error.
- Malformed entry (e.g., missing `type:`) → loader logs warning,
  drops entry, other entries load.
- `GetArchetypeDefaultGoals("nonexistent")` → empty list.

### 10.4 Admin command exercise

Existing `goal current` / `goal scores` commands work against multi-instance
goals. Tests should confirm:

- `goal scores` displays multiple instances of the same type with distinct
  ids when AllowMultiple=true.
- `goal current` returns the highest-scoring instance among multiple.

### 10.5 Boot smoke (per CLAUDE.md pre-push SOP)

- Server starts; no panics from any catalog `init()`.
- All 23 archetype YAMLs parse with the new `default_goals:` field.
- The 16 archetypes that get defaults (Section 5.1) successfully seed
  their first mob instance.
- Log a `goals.switch` line when a freshly-spawned mob's `survival` goal
  is selected (HP=100 = filtered; spawn a damaged test mob to verify).
- `goal current <mob-with-survival-default>` returns the survival goal.

### 10.6 Out of scope for 4.3 tests

- Behavior-tree integration / planner-consumes-goal tests — 4.4.
- Reactive event-driven goal generation tests — 4.5.
- Bulk satisfaction sweep tests — 4.6.
- Cross-type conflict tests — deferred per §7.

### 10.7 Rollout — feature branch decomposition

Single chunk on `feature/aliveness-4.3-goal-types-catalog`. Suggested task
ordering for the plan:

1. Engine deltas: `ParamSchema`, `ValidateParams`, `Add` wiring (no types
   yet, just the engine surface + tests).
2. Engine deltas: `AllowMultiple` + `DedupKey` fields, `Add` conflict
   logic, tests.
3. Engine deltas: `SeededFromArchetype` field, lazy-seed branch in
   `loadOrLazyInit`, tests.
4. Behaviortree side: `TreeDef.DefaultGoals`, `LoadArchetypeYAMLFromFile`
   extension, `GetArchetypeDefaultGoals` accessor, tests.
5. Goals side: `ArchetypeDefaultsLookupFn` + `SetArchetypeDefaultsLookup`
   (parallel to 4.2's SetWeightsLookup), tests.
6. `main.go` wiring: register the defaults lookup adapter.
7–19. One task per type — Predicate + ContextScore + DedupKey + tests +
   `init()` registration. 13 tasks.
20. Catalog package blank-import in `main.go`.
21. Archetype YAML edits: add `default_goals:` block to the 16
   defaulted-archetypes per Section 5.1.
22. Author `internal/goals/context.md` (closes 4.1/4.2 gap) +
   `internal/goals/catalog/context.md` (new package).
23. Smoke checklist + roadmap rollup (24/42 → 25/42) + PATCH_NOTES entry.

**Push to prod is safe** — selection runs, archetype-seeded survival goals
exist, but nothing tactical reads them yet (4.4 wires btree). Observable
change: most mobs have a current goal visible via `goal current <mob>`,
and the `goals.switch` debug log fires when survival kicks in during combat.

### 10.8 Roadmap rollup after 4.3 ships

25/42. Next: 4.4 Strategic→tactical translation (where btree finally reads
`CurrentGoalOf` and dispatches to per-goal-type planners) → 4.5 Reactive
goal generation (event hooks seed new goals from world state) → 4.6 Goal
satisfaction & pruning.

---

## File touch list

**New:**
- `internal/goals/context.md` — close 4.1/4.2 documentation gap (per
  `internal/relationships/`, `internal/knowledge/`, `internal/conversations/`
  package convention).
- `internal/goals/catalog/` subpackage — 13 type files + tests + a
  `catalog.go` with package doc + shared helpers + `context.md`.
- `internal/behaviortree/engine_default_goals_test.go`.

**Modified (engine — goals package):**
- `internal/goals/lookup.go` — extend with `ArchetypeDefaultsLookupFn`,
  `SetArchetypeDefaultsLookup`, internal `resolveArchetypeDefaults`.

**Modified:**

- `internal/goals/types.go` — `ParamSchema`, extend `GoalTypeMeta` with
  `Params`, `AllowMultiple`, `DedupKey`; extend `MobGoals` with
  `SeededFromArchetype`.
- `internal/goals/store.go` — `Add` runs `ValidateParams` and applies
  AllowMultiple/DedupKey conflict logic; `loadOrLazyInit` calls
  `seedFromArchetype` on fresh mobs.
- `internal/goals/persistence.go` — no behavior change; backward-compat
  test for files without sentinel.
- `internal/goals/persistence_test.go` — sentinel round-trip + legacy
  load tests.
- `internal/goals/store_test.go` — extend with AllowMultiple/DedupKey/
  ParamSchema/seeding integration tests.
- `internal/behaviortree/types.go` — extend `TreeDef` with
  `DefaultGoals []GoalDefault`; declare `GoalDefault` struct.
- `internal/behaviortree/loader.go` — `LoadArchetypeYAMLFromFile` reads
  the new field.
- `internal/behaviortree/engine.go` — add `archetypeDefaultGoals` cache
  + `GetArchetypeDefaultGoals` accessor.
- `main.go` — register `goals.SetArchetypeDefaultsLookup` adapter; add
  blank import of `internal/goals/catalog`.
- `_datafiles/world/dogmud/behaviors/archetypes/*.yaml` — add
  `default_goals:` block to the 16 defaulted archetypes per §5.1.
- `MOB_ALIVENESS_ROADMAP.md` — flip 4.3 to Done, bump rollup to 25/42.
- `PATCH_NOTES.md` — chunk 4.3 entry.

**Not touched in 4.3:** `internal/hooks/MobIdle_HandleIdleMobs.go` (existing
gossip system stays as-is); `internal/goals/select.go` (4.2 selection
mechanics unchanged); behavior-tree actions/conditions; schedules; patrols.
