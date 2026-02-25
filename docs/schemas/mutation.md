# Mutation YAML Schema

Mutations are loaded from `_datafiles/world/dogmud/mutations/` at startup.
Filename must match `{mutationid}.yaml`.

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mutationid` | string | yes | Unique ID, used as filename base |
| `name` | string | yes | Display name |
| `description` | string | yes | Flavor text shown on acquisition |
| `rarity` | int (1-10) | yes | 1=common, 10=very rare. Also determines acquisition cost (load = rarity x level) |
| `visual` | string | no | Appended to character look description |
| `pro` | MutationEffect | no | Legacy single pro effect (migrated into `pros` at load) |
| `con` | MutationEffect | no | Legacy single con effect (migrated into `cons` at load) |
| `pros` | []MutationEffect | no | List of beneficial effects (Phase 24+) |
| `cons` | []MutationEffect | no | List of detrimental effects (Phase 24+) |
| `conflicts` | []string | no | Mutation IDs that cannot coexist with this one |

## MutationEffect

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Effect type identifier (see table below) |
| `target` | string | Stat name or flag name, or "" if not applicable |
| `value` | float64 | Magnitude (positive = bonus, negative = penalty) |

## Effect Types

| Type | Target | Description |
|------|--------|-------------|
| `stat_multiplier` | stat name | Multiply stat adjusted value (e.g., 0.15 = +15%) |
| `stat_flat` | stat name | Add flat bonus to stat |
| `health_multiplier` | — | Multiply max HP |
| `stamina_regen_multiplier` | — | Multiply stamina regen rate |
| `natural_armor` | — | Flat physical damage reduction |
| `natural_weapon` | — | Flat unarmed damage bonus |
| `magical_damage_reduction` | — | Fraction of magical damage reduced (0.0–1.0) |
| `conviction_cost_multiplier` | — | Multiply spell conviction cost |
| `aggro_magnet` | — | Multiplier for mob target weighting |
| `conditional_damage_low_hp` | — | Damage bonus when HP < 25% |
| `dodge_modifier` | — | Flat dodge score bonus/penalty |
| `damage_multiplier` | — | Multiply all outgoing physical damage |
| `movement_speed` | — | Modify movement stamina cost (negative = faster) |
| `health_regen` | — | Passive HP regen per tick |
| `skill_progression_multiplier` | — | Scale skill gain chance |
| `stat_progression_multiplier` | — | Scale stat gain chance |
| `flag` | flag name | Grant a permanent flag (nightvision, lightsource, hidden, see-hidden) |
| `health_regen_if_lit` | — | HP regen only in lit rooms |

## Conflicts

Conflicts are bidirectional by convention but checked in both directions by code.
If mutation A lists B in its `conflicts`, then a character owning A cannot acquire B
(and vice versa, since the code also checks owned mutations' conflict lists).

## Example (simple, legacy format)

```yaml
mutationid: dense-muscles
name: Dense Muscles
description: Your muscles condense into cables of extraordinary power.
rarity: 4
visual: Their musculature is visibly denser and more compact than normal.
conflicts:
  - hollow-bones
pro:
  type: stat_multiplier
  target: strength
  value: 0.15
con:
  type: stamina_regen_multiplier
  target: ""
  value: -0.10
```

## Example (multi-effect, Phase 24+ format)

```yaml
mutationid: large
name: Large
description: Your frame expands beyond normal proportions.
rarity: 7
visual: They tower over most others, broad and imposing.
conflicts:
  - small
pros:
  - type: health_multiplier
    value: 0.20
  - type: damage_multiplier
    value: 0.15
cons:
  - type: dodge_modifier
    value: -15
  - type: stat_flat
    target: dexterity
    value: -10
```
