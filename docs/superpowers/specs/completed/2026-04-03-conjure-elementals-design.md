# Conjure Elementals — Design Spec

**Date:** 2026-04-03
**Goal:** 5 elemental conjure spells (water, earth, air, fire, magma)
with unique combat identities. High conviction cost, no components.

## Elemental Types

| Type | Skill Gate | Archetype | Base Pool | Conviction Cost | Base Folds |
|------|-----------|-----------|-----------|----------------|------------|
| Water | 3 | fighting | 80 | 150 | 4 |
| Earth | 10 | fighting | 90 | 200 | 6 |
| Air | 18 | fighting | 70 | 280 | 8 |
| Fire | 25 | fighting | 85 | 350 | 10 |
| Magma | 60 | fighting | 130 | 450 | 14 |

Conviction costs are punishing — conjuring drains most of a
caster's conviction pool. A typical endgame caster with ~450-500
CP will be nearly empty after conjuring a magma elemental. Even
water at 150 is a significant commitment. This is the trade-off
for not needing components or corpses.

All are `school: manifestation`, `type: neutral` (no target needed).
No components — pure conviction cost. Scaling uses the standard
companion formula: `CalcCompanionStatPool(basePool, charisma, manifestation)`.

## Species Base Stats (100 = average human)

**Water Elemental** (species 36):
- Str: 100, Dex: 50, Per: 60, Vit: 200, Wil: 40, Cha: 5
- Slow, massive HP pool, absorbs punishment
- basedamage: 4

**Earth Elemental** (species 37):
- Str: 160, Dex: 45, Per: 50, Vit: 220, Wil: 30, Cha: 5
- Slowest but hardest hitting tank, high physical mitigation
- Special: bash (has shield-equivalent for bash check, or skip
  shield requirement for elementals)
- basedamage: 6

**Air Elemental** (species 38):
- Str: 40, Dex: 200, Per: 160, Vit: 30, Wil: 60, Cha: 5
- Extremely hard to hit (Dex 200), glass cannon, low HP
- basedamage: 2

**Fire Elemental** (species 39):
- Str: 60, Dex: 180, Per: 140, Vit: 35, Wil: 80, Cha: 5
- Hard to hit, return damage to melee attackers
- Return damage: when hit by melee, attacker takes fire damage
  (percentage of damage dealt back)
- basedamage: 3

**Magma Elemental** (species 40):
- Str: 180, Dex: 70, Per: 80, Vit: 180, Wil: 60, Cha: 5
- Total beast: high Str AND Vit, functional Dex
- Has both bash AND return damage
- Aspirational — skill gate 60 means this is endgame
- basedamage: 7

## Return Damage System (General — not elemental-specific)

Return damage is a general combat mechanic: when a melee attacker
hits a defender that has return damage, the attacker takes a
percentage of their own damage back.

```
returnDamage = attackDamage × defender's return_damage_pct
```

This is implemented as a **stat mod** on the Character, not a
species-specific hack. Sources of return damage:
- Equipment: `return_damage` stat mod on armor (battlerager armor)
- Buffs: `return_damage` stat mod from buff effects
- Species: base return damage on species definition
- Mutations: `return_damage` mutation effect (future)

Fire elemental gets return damage via its species definition.
Magma elemental gets it the same way. Players can get it from
equipment or buffs.

### Implementation

In the combat resolution (after damage is applied to the defender),
check `defender.StatMod("return_damage")`. If > 0, apply:

```go
returnPct := float64(defender.StatMod("return_damage")) / 100.0
returnDmg := int(float64(damageDealt) * returnPct)
if returnDmg > 0 {
    attacker.Health -= returnDmg
    // Message: "[defender]'s [source] lashes back at [attacker]!"
}
```

**Anti-loop:** Return damage must NOT trigger return damage back.
Add a flag to the combat context (`IsReturnDamage bool`) that
skips the return damage check when processing return damage.

### Species Return Damage

Add a `return_damage` field to species definitions:

```yaml
# In species/39-fire_elemental.yaml
return_damage: 25  # 25% of melee damage returned to attacker
```

This gets picked up by `StatMod("return_damage")` via the species
stat mod system, or added as a separate field on the species struct.

### Messaging

The return damage message should be flavor-appropriate to the source:
- Fire/magma species: "Flames lash back at [attacker]!"
- Equipment (armor): "[attacker] is cut by the barbed armor!"
- Buff: "[attacker] recoils from the retaliatory magic!"

A generic fallback: "[attacker] takes damage from striking [defender]!"

## Earth/Magma Bash

Earth and magma elementals use bash in combat. The bash check
normally requires `HasShield()`. For elementals, bash should work
without a shield — they bash with their massive rocky fists.

Options:
- **A)** Give elementals a shield-slot item (invisible rock shield)
- **B)** Modify bash to skip the shield check for certain species
- **C)** Add a species flag `natural_shield` that satisfies HasShield

Option A is simplest — create a non-droppable "rocky hide" shield
item and equip it on the elemental mob templates. It works with
the existing bash system with no code changes.

## Mob Templates

All in `_datafiles/world/dogmud/mobs/summons/`:

| Mob ID | Name | Species ID |
|--------|------|-----------|
| 310 | Water Elemental | 36 |
| 311 | Earth Elemental | 37 |
| 312 | Air Elemental | 38 |
| 313 | Fire Elemental | 39 |
| 314 | Magma Elemental | 40 |

Combat commands:
- Water: standard melee emotes
- Earth: `bash` + crushing emotes
- Air: fast striking emotes
- Fire: burning/searing emotes
- Magma: `bash` + molten/crushing emotes

Idle commands: elemental-themed ambient emotes.

## Spells

| Spell ID | Name |
|----------|------|
| conjure-water | Conjure Water Elemental |
| conjure-earth | Conjure Earth Elemental |
| conjure-air | Conjure Air Elemental |
| conjure-fire | Conjure Fire Elemental |
| conjure-magma | Conjure Magma Elemental |

All follow the same JS pattern as summon spells:
1. Check companion cap
2. Calculate scaled statpool
3. SpawnMobScaled
4. CharmSet + AddCompanion with "conjured" source type
5. Flavor text

No component check needed (unlike summon spells).

## New Files Needed

- 5 species YAML (36-40)
- 5 mob templates (310-314)
- 5 spell YAML + 5 spell JS (10 files)
- 5 help templates
- 1 shield item for earth/magma (or modify bash)
- Config: ElementalReturnDamagePercent
- Go: return damage hook in combat

## What This Does NOT Include

- Charm spell rework (separate spec)
- Elemental spell discovery for companion caster types
- Environmental interactions (fire elemental in water, etc.)
