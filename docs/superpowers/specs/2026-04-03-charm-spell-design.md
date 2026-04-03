# Charm Spell — Design Spec

**Date:** 2026-04-03
**Goal:** Single charm spell that converts hostile mobs into
companions via opposed roll. Duration-based with diminishing
re-rolls. Charmed mobs keep original stats and gear.

## Spell Definition

| Field | Value |
|-------|-------|
| Spell ID | charm |
| Name | Charm |
| School | manifestation |
| Type | harmsingle (targets one mob) |
| Base Folds | 36 |
| Cost | 120 conviction |
| Manifestation Gate | 12 |

## Opposed Roll

```
attacker = Charisma + (manifestation × SkillMultiplier × 25)
defender = Willpower + (statpool × 0.10)
```

The statpool component means stronger mobs are harder to charm
regardless of their willpower. A 200-statpool mob with low
willpower is still harder than a 50-statpool mob with the same
willpower.

### Aggro Penalty

If the target is in combat, the caster's roll is penalized:

| Target State | Caster Effectiveness |
|-------------|---------------------|
| Idle (not in combat) | 100% |
| Aggroed on someone else | 85% |
| Aggroed on the caster | 75% |

Effectiveness multiplies the caster's roll score.

### Charm-Immune Mobs

New mob YAML field: `charm_immune: true`

Applied to:
- All mobs in the `merchant` group
- Quest-critical NPCs (dialogue NPCs, trainers, etc.)
- Future boss mobs

The spell fails immediately with: "Your will washes over [name]
but finds no purchase. This creature cannot be charmed."

## Duration

```
baseDuration = 50 + Charisma/2 + manifestation × 3
```

At Cha 100, Manifest 10: `50 + 50 + 30 = 130 rounds`
At Cha 150, Manifest 30: `50 + 75 + 90 = 215 rounds`

Duration counts down each round. When it hits 0, re-roll.

## Re-Roll Mechanic

When duration expires, an opposed re-roll fires automatically.
Same formula as the initial charm, but with diminishing
effectiveness on the caster's side:

- 1st re-roll: 99% effectiveness
- 2nd re-roll: 97%
- 3rd re-roll: 94%
- 4th re-roll: 90%
- 5th re-roll: 85%
- Pattern: `effectiveness = 1.0 - (rerollCount × 0.01) × rerollCount`
  (quadratic decay)

If the caster WINS the re-roll:
- Duration resets to full (recalculated from current stats)
- Re-roll counter increments
- Player sees: "Your hold on [name] wavers... but you reassert
  your will."

If the caster LOSES:
- Charm breaks immediately
- Mob turns hostile (full betrayal, same as dismiss)
- CompanionInfo removed
- Player sees: "[name] breaks free of your control!"
- Room sees: "[name] snarls and turns on [player]!"

## On Charm Success

1. Remove target from combat (EndAggro on target + anyone
   fighting the target)
2. Charm the mob permanently (duration managed separately)
3. Create CompanionInfo with source type "charmed"
4. Track in CharmedMobs
5. Strip companion chains (anti-recursion)
6. Mob keeps all original stats, gear, spells, skills

## On Failure (initial cast)

"You attempt to bend [name]'s will, but it resists!"
Conviction is consumed. Target is NOT consumed (stays hostile).
If the target was attacking the caster, combat continues.

## Companion Cap Check

The `onCast` handler checks `GetCompanionCount() >= GetMaxCompanionCount()`
before allowing the cast. If at cap: "You cannot maintain another
companion bond." and the cast is cancelled (no conviction spent).

## Messaging

### During cast (onWait folds)
- "You lock eyes with [name], your will pressing against theirs..."
- "The air between you and [name] crackles with psychic tension..."
- "You feel [name]'s resistance wavering..."

### On charm success
- To caster: "[name]'s eyes glaze as your will takes hold. It is yours."
- To room: "[player] bends [name] to their will!"
- To target (if player): N/A (mob targets only)

### On charm failure (initial)
- To caster: "You reach for [name]'s mind, but its will is iron. The spell shatters."
- To room: "[player]'s charm spell breaks against [name]'s resolve!"

### On charm-immune target
- To caster: "Your will washes over [name] but finds no purchase. This creature cannot be charmed."

### On re-roll success (charm holds)
- To caster: "Your hold on [name] wavers... but you reassert your will."
- No room message (internal struggle, not visible)

### On re-roll failure (charm breaks)
- To caster: "[name] breaks free of your control!"
- To room: "[name] snarls and turns on [player]!"
- Same betrayal mechanic as dismiss (mob turns hostile)

### Periodic warning as re-rolls accumulate
When `rerollCount >= 3`:
- "You sense [name]'s will straining against your bond..."
When `rerollCount >= 5`:
- "[name]'s eyes flash with defiance. Your control is slipping..."

## Duration Tracking

Store on CompanionInfo:
- `CharmDuration int` — rounds remaining
- `CharmRerolls int` — number of re-rolls so far

The round tick decrements `CharmDuration`. When it hits 0,
the re-roll fires.

## New Files

- `_datafiles/world/dogmud/spells/charm.yaml` + `.js`
- `_datafiles/world/dogmud/templates/help/charm.template`
- Modify: `internal/mobs/mobs.go` — add `CharmImmune` field
- Modify: `internal/characters/companions.go` — add duration fields
- Modify: `internal/hooks/NewRound_MobRoundTick.go` — charm duration tick + re-roll
- Add `charm_immune: true` to all merchant mobs and quest NPCs
