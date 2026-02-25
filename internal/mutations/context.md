# internal/mutations — Package Context

## Purpose

Implements the Chrysalis mutation system introduced in **Stage 12.1** and
expanded in **Phase 24** with multi-effect mutations, conflicts, and load-based
acquisition.

Mutations are the primary character-differentiation mechanic in DOGMud.
The Chrysalis (a world-spanning plague) reshapes characters who survive sustained
combat, granting both benefits and drawbacks each time it acts.

Adding a new mutation requires **only a YAML file** in
`_datafiles/world/dogmud/mutations/` — no Go code changes needed for existing
effect types.

---

## Key Concepts

### MutationSpec (mutations.go)
Each mutation is a `MutationSpec` loaded from a YAML file.  Fields:
- `mutationid` — filesystem key and registry key (`"tough-skin"`)
- `name` — display name (`"Tough Skin"`)
- `description` — player-facing flavor text
- `rarity` — 1 (common) … 10 (very rare); drives weighted acquisition pool AND load cost
- `visual` — appended to character look description
- `pro` / `con` — legacy single `MutationEffect{Type, Target, Value}` (migrated into lists)
- `pros` / `cons` — lists of `MutationEffect` (Phase 24+, supports multiple effects)
- `conflicts` — list of mutation IDs that cannot coexist with this mutation

### Backward Compatibility
Legacy `pro`/`con` single-effect fields are automatically migrated into the
`pros`/`cons` slices during `Validate()`. Existing YAML files work without changes.

### MutationEffect Types

| Type | Applied In | Notes |
|------|-----------|-------|
| `stat_multiplier` | `character.RecalculateStats()` | Multiplies `ValueAdj` after `Recalculate()`; Target = stat name |
| `stat_flat` | `character.RecalculateStats()` | Added to `Mods` before `Recalculate()`; Target = stat name |
| `natural_armor` | `character.GetDefense()` | Flat addition to physical damage reduction |
| `health_multiplier` | `character.RecalculateStats()` | Multiplies `HealthMax.Value` post-Recalculate; positive = more HP |
| `stamina_regen_multiplier` | `character.StaminaPerRound()` | Multiplies regen; negative = slower recovery |
| `natural_weapon` | `character.CalculateUnarmedDamage()` | Flat bonus to baseDamage |
| `magical_damage_reduction` | `hooks/spell_resolution.go` | Reduces incoming mob spell damage (0.0–1.0 fraction) |
| `conviction_cost_multiplier` | `usercommands/skill.cast.go` | Multiplies total conviction cost at cast initiation |
| `conditional_damage_low_hp` | `hooks/NewRound_DoCombat.go` | +20% physical damage when HP < 25% (Adrenaline Surge) |
| `aggro_magnet` | `mobcommands/lookfortrouble.go` | Multiplies target-selection weight for predatory mobs |
| `dodge_modifier` | `combat.calculateCombat()` | Flat dodge score bonus/penalty |
| `damage_multiplier` | `combat.calculateCombat()` | Multiply all outgoing physical damage |
| `movement_speed` | `character.GetMovementStaminaCost()` | Modify movement stamina cost (negative = faster) |
| `health_regen` | `hooks/UserRoundTick` | Passive HP regen per tick |
| `skill_progression_multiplier` | `character.CheckSkillProgression()` | Scale skill gain chance |
| `stat_progression_multiplier` | `character.CheckStatProgression()` | Scale stat gain chance |
| `flag` | various | Grant a permanent flag (nightvision, lightsource, hidden, see-hidden) |
| `health_regen_if_lit` | `hooks/UserRoundTick` | HP regen only in lit rooms |

### Acquisition System

Progress accumulates at `MutationProgressGain` (1.0) per combat round
while `character.Aggro != nil`. The Eye moon modulates rate (0.5x–1.5x).

Threshold formula (Phase 24.1 — load-based):
```
load = sum(rarity × level) for each owned mutation
threshold = MutationBaseProgress (50) × MutationProgressScale (1.5) ^ load
```

Higher rarity mutations contribute more to load, making subsequent acquisitions
progressively harder. This replaced the flat event-count formula.

When threshold is reached, `GetWeightedPool()` builds a slice where each
mutation appears `(11 - Rarity)` times.  Conflicting and already-owned mutations
are excluded. `RollAcquisition()` picks uniformly at random from that slice.

Maximum mutations per character: `MutationMaxCount` = 5.

### Conflict System (Phase 24.1)

Mutations can declare `conflicts: [id1, id2]` in their YAML. `HasConflict()`
checks both directions:
- Candidate's conflicts list vs owned mutations
- Each owned mutation's conflicts list vs candidate

Conflicting mutations are excluded from `GetWeightedPool()` during acquisition.

### Registry

```go
mutations.LoadMutationFiles()   // called once from main.go at startup
mutations.GetMutation(id)       // look up a spec by id
mutations.GetAll()              // the full map
mutations.GetMutationLoad(m)    // rarity-weighted load for acquisition scaling
mutations.HasConflict(m, id)    // check if candidate conflicts with owned
```

---

## Character Fields

```go
// in internal/characters/character.go
Mutations        map[string]int  `yaml:"mutations,omitempty"`        // id → level
MutationProgress float64         `yaml:"mutationprogress,omitempty"` // combat pressure
```

`Validate()` ensures `Mutations` is never nil.

---

## Hook Integration Points

| Hook File | What It Does |
|-----------|-------------|
| `internal/hooks/NewRound_UserRoundTick.go` | Progress tick + acquisition trigger (load-based) |
| `internal/hooks/NewRound_DoCombat.go` | Adrenaline Surge bonus damage |
| `internal/hooks/spell_resolution.go` (`resolveMobSpellAgainstPlayer`) | Magical Resistance |
| `internal/usercommands/skill.cast.go` | Conviction cost multiplier |
| `internal/mobcommands/lookfortrouble.go` | Pheromone Glands aggro weight |

---

## Adding a New Mutation

1. Create `_datafiles/world/dogmud/mutations/<mutationid>.yaml`
2. Fill in all fields (see `docs/schemas/mutation.md` for the full schema)
3. If using multi-effect, use `pros:` / `cons:` lists instead of singular `pro:` / `con:`
4. Add `conflicts:` if the mutation is incompatible with another
5. If the effect type is **new**, add a helper function in `mutations.go`
   and a hook call in the appropriate game system file
6. Restart the server

No Go code changes are required for existing effect types.

---

## Files in This Package

| File | Purpose |
|------|---------|
| `mutations.go` | Structs, registry, loader, conflict/load system, all effect helpers |
| `mutations_test.go` | Unit tests for helpers, weighted pool, acquisition |
| `context.md` | This file — package overview for Claude Code |

---

## Stage Roadmap

- **12.1** (complete) — framework, 10 mutations, all hooks
- **12.2** (complete) — mutation level upgrades (L2/L3), visual integration in `look`
- **24.1** (complete) — multi-effect schema, conflicts, load-based acquisition
- **24.2** — 12 new passive mutations with new effect types
- **24.3** — NPC/mob mutations
- **24.4** — environmental/conditional mutations with flags
- **24.5** — active ability mutations (combat commands)
- **24.6** — extra limbs + quad wielding
