# Tank Stat Archetype + Rhetoric Bump — Design

**Date:** 2026-04-21
**Status:** Approved
**Related memory:** `project_tank_taunt_viability.md`
**Surfaced by:** T7 smoke of the tank_taunter behavior-tree archetype (earlier today). Golem's taunts almost never landed because its `fighting` stat archetype allocated ~7% of the stat pool to charisma.

## Problem

The `tank_taunter` behavior-tree archetype fires taunt as its signature move. Taunt is a conviction attack resolved as:

```
attackScore  = Charisma    + (rhetoric × SkillWeight)
defenseScore = target.Wil  + (target.rhetoric × SkillWeight)
```

Opposed roll picks winner.

All three current tank mobs (305 flesh_golem, 311 earth_elemental, 314 magma_elemental) use `archetype: fighting` which distributes the stat pool 80% physical (Str/Dex/Vit) / 20% mental (Per/Wil/Cha). Charisma ends up ~7% of total training (~1/3 of the 20% mental share). A mob with statpool=100 has ~7 charisma training, ~50-ish total Charisma after rolls. Against a player with typical willpower ~40-60, the tank loses the opposed roll more often than it wins.

## Design rule

> **A `tank` stat-distribution archetype allocates the stat pool so Charisma (for taunt) and Vitality (for HP buffer) are the highest-weighted stats. Combined with a modest Rhetoric skill bump, tank mobs land taunts reliably without being un-missable.**

## Changes

### Change 1 — New `tank` stat archetype

**File:** `internal/mobs/mobs.go`

Add a new case to the archetype switch (currently at line 402) alongside `fighting` / `casting`:

```go
case "tank":
    // Tank/taunter distribution: 25% Cha (core taunt mechanic),
    // 20% Vit (HP buffer), 15% each Str/Dex/Wil, 10% Per.
    r := util.Rand(100)
    switch {
    case r < 25:
        statIdx = 5 // Charisma
    case r < 45:
        statIdx = 2 // Vitality
    case r < 60:
        statIdx = 0 // Strength
    case r < 75:
        statIdx = 1 // Dexterity
    case r < 90:
        statIdx = 4 // Willpower
    default:
        statIdx = 3 // Perception
    }
```

Stat index mapping matches the existing switch at lines 421-432:
- 0 = Str, 1 = Dex, 2 = Vit, 3 = Per, 4 = Wil, 5 = Cha.

### Change 2 — Wire three tank mobs to `archetype: tank`

Flip the existing `archetype:` field from `fighting` to `tank` in:

- `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml`
- `_datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml`
- `_datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml`

Single-token change per file.

### Change 3 — Rhetoric skill bump

Add (or extend) a `skills:` block in each of the three tank mob YAMLs with `rhetoric: 10`.

Example for `305-flesh_golem.yaml`:

```yaml
  skills:
    rhetoric: 10
```

If an existing `skills:` block is present, extend it; don't duplicate. Pattern matches `304-vampire.yaml` which uses the same `skills:` sub-map shape.

**Value choice rationale:** skill 10 adds `10 × SkillWeight` to the taunt attack score. Combined with the ~25% Cha share (~25 Cha training at statpool 100), a tank mob's taunt attack score should hit ~75-90 against a player's ~50-70 willpower + low-to-mid rhetoric. Opposed roll probability: tank wins ~65-75% of taunts against typical players.

## Expected balance outcome

At `statpool: 100`:
- ~25 Charisma training → ~75 total Charisma after validation rolls
- ~20 Vitality training → noticeably more HP than a "fighting" counterpart at the same pool
- ~15 each Str/Dex/Wil, ~10 Per → respectable damage output and defense, weaker perception (detection/search)

At `statpool: 85` (earth elemental's default):
- ~21 Cha, ~17 Vit, ~13 each Str/Dex/Wil, ~8 Per — same ratios, lower absolute values.

## Out of scope

- **Balancing other non-existent tank mobs.** The three wired mobs are the current tank set; future tank content can opt in via `archetype: tank`.
- **Adjusting the `fighting` / `casting` distributions.** Existing distributions stay as-is.
- **New `command_best_of` gate or btree changes.** Purely stat/skill balance; no engine changes.
- **Rhetoric skill bumps for non-tank mobs.** Per-mob tuning stays per-mob.
- **Taunt formula changes** (e.g., reducing willpower weight on defense). Would affect all taunters including players; keep the formula stable.
- **Dynamic `stat_boost:` ItemSpec field** for fine-grained per-mob tuning. Not needed when archetype + `skills:` block cover the use cases.

## Testing

### Unit tests

**New test in `internal/mobs/mobs_test.go`:** `TestNewMobById_TankArchetypeDistributesStats`. Build a mob with `archetype: tank, statpool: 1000` (large pool shrinks random variance to stable ratios). Sample the resulting training distribution. Assert:

- Charisma training ≥ 200 (~25% of 1000 with slack)
- Vitality training ≥ 150 (~20% of 1000 with slack)
- Perception training ≤ 150 (~10% of 1000 with ceiling slack)
- Sum of all 6 stats' training = 1000 (no stat pool leaks)

### Smoke tests

1. **Stats on summon** — summon flesh golem, `consider`/`status`, verify Charisma is the highest stat and Vitality is the second-highest. Repeat for earth elemental and magma elemental.
2. **Taunt landing rate** — engage a tank mob against a single target in combat. Count taunt outcomes across ~10 taunts. ≥6 should land (aggro actually switches, "pulls" the target). If landing rate is consistently <50%, re-tune.
3. **No regression on other archetypes** — summon any non-tank mob that still uses `fighting` or `casting`. Confirm stats look normal (no overflow or underflow).
