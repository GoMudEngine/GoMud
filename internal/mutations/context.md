# internal/mutations — Package Context

## Purpose

Implements the Chrysalis mutation system introduced in **Stage 12.1**.
Mutations are the primary character-differentiation mechanic in DOGMud.
The Chrysalis (a world-spanning plague) reshapes characters who survive sustained
combat, granting both a benefit and a drawback each time it acts.

Adding a new mutation requires **only a YAML file** in
`_datafiles/world/dogmud/mutations/` — no Go code changes needed.

---

## Key Concepts

### MutationSpec (mutations.go)
Each mutation is a `MutationSpec` loaded from a YAML file.  Fields:
- `mutationid` — filesystem key and registry key (`"tough-skin"`)
- `name` — display name (`"Tough Skin"`)
- `description` — player-facing flavor text
- `rarity` — 1 (common) … 10 (very rare); drives weighted acquisition pool
- `visual` — appended to character look description (Stage 12.2 hook)
- `pro` / `con` — `MutationEffect{Type, Target, Value}`

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

### Acquisition System

Progress accumulates at `MutationProgressGain` (1.0) per combat round
while `character.Aggro != nil`.

Threshold for mutation N (0-indexed):
```
threshold = MutationBaseProgress (50) × MutationProgressScale (1.5) ^ N
```

So: 1st mutation at 50 progress, 2nd at 75, 3rd at ~113, etc.

When threshold is reached, `GetWeightedPool()` builds a slice where each
mutation appears `(11 - Rarity)` times.  `RollAcquisition()` picks uniformly
at random from that slice.  Already-owned mutations are excluded.

Maximum mutations per character: `MutationMaxCount` = 5.

### Registry

```go
mutations.LoadMutationFiles()   // called once from main.go at startup
mutations.GetMutation(id)       // look up a spec by id
mutations.GetAll()              // the full map
```

---

## Character Fields

```go
// in internal/characters/character.go
Mutations        map[string]int  `yaml:"mutations,omitempty"`        // id → level
MutationProgress float64         `yaml:"mutationprogress,omitempty"` // combat pressure
```

`Validate()` ensures `Mutations` is never nil.
Levels are always 1 for Stage 12.1; Stage 12.2 will use L2/L3 for upgrades.

---

## Hook Integration Points

| Hook File | What It Does |
|-----------|-------------|
| `internal/hooks/NewRound_UserRoundTick.go` | Progress tick + acquisition trigger |
| `internal/hooks/NewRound_DoCombat.go` | Adrenaline Surge bonus damage |
| `internal/hooks/spell_resolution.go` (`resolveMobSpellAgainstPlayer`) | Magical Resistance |
| `internal/usercommands/skill.cast.go` | Conviction cost multiplier |
| `internal/mobcommands/lookfortrouble.go` | Pheromone Glands aggro weight |

---

## Adding a New Mutation

1. Create `_datafiles/world/dogmud/mutations/<mutationid>.yaml`
2. Fill in all fields (see any existing file for reference)
3. If the effect type is **new**, add a helper function in `mutations.go`
   and a hook call in the appropriate game system file
4. Restart the server (or `reload` if hot-reload is available)

No Go code changes are required for existing effect types.

---

## Files in This Package

| File | Purpose |
|------|---------|
| `mutations.go` | Structs, registry, loader, all effect helpers |
| `mutations_test.go` | Unit tests for helpers, weighted pool, acquisition |
| `context.md` | This file — package overview for Claude Code |

---

## Stage Roadmap

- **12.1** (complete) — framework, 10 mutations, all hooks
- **12.2** (next) — mutation level upgrades (L2/L3), visual integration in `look`,
  possible mutation-specific skill/ability unlocks
