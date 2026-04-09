# Necromancy System — Phase 3 Design

**Date:** 2026-04-03
**Goal:** Six undead companion types raised from corpses via
manifestation-school spells. Tiered progression with caster and
corpse contributing 50/50 to raised companion power. Corpse
assessment command. Caster undead types with fixed starting spells
that can discover more. Flavorful failure on insufficient
corpse/skill.

## Undead Types

| Type | Skill Gate | Archetype | Base Pool | Caster? | Fantasy |
|------|-----------|-----------|-----------|---------|---------|
| Skeleton | 1 | fighting | 60 | No | Fast, fragile, dual-wield |
| Zombie | 5 | fighting | 80 | No | Tanky shambler, high Vit |
| Wraith | 15 | casting | 70 | Yes | Ethereal, spell damage |
| Spectre | 25 | casting | 90 | Yes | Conviction damage, fear |
| Vampire | 40 | fighting | 100 | Hybrid | Life drain, self-buff |
| Flesh Golem | 50 | fighting | 120 | No | Siege unit, absorbs corpses |

Each type gets its own species definition with appropriate base
stats, its own mob YAML template, and a corresponding raise spell.

### Species Base Stats

All stats use 100 = average human as baseline.

| Species | Str | Dex | Per | Vit | Wil | Cha | Notes |
|---------|-----|-----|-----|-----|-----|-----|-------|
| Skeleton | 100 | 150 | 80 | 60 | 15 | 5 | Fast striker, fragile |
| Zombie | 130 | 50 | 40 | 200 | 15 | 5 | Hits hard, absorbs everything |
| Wraith | 40 | 160 | 150 | 35 | 170 | 30 | Ethereal — hard to hit, strong caster |
| Spectre | 30 | 150 | 130 | 30 | 150 | 170 | Ethereal — hard to hit, conviction powerhouse |
| Vampire | 120 | 140 | 120 | 110 | 110 | 160 | All-rounder, nothing below average |
| Flesh Golem | 220 | 65 | 40 | 240 | 15 | 5 | Slow but functional, extreme Str/Vit |

**Wraith/Spectre defense philosophy:** Low Vit = tiny HP pool, but
extremely high Dex = nearly impossible to hit with melee. Players
fight them with spells. This is thematic — ethereal beings phase
through physical attacks.

### Species Details

**Skeleton:**
- No special abilities
- Fast attacks from high Dex, average human strength
- Fragile (Vit 60) — a glass cannon melee fighter

**Zombie:**
- No special abilities
- Enormous HP pool from Vit 200, strong hits (Str 130)
- Very slow (Dex 50), poor awareness — a wall of meat

**Wraith:**
- Starting spells: chill touch (harm), minor shield (buff)
- Can discover more spells via manifestation discovery
- Spawns with nightvision/see-hidden buffs (detects hidden)

**Spectre:**
- Starting spells: conviction spike, conviction ward, fear
- Slightly better starting spells than wraith
- Can discover more spells

**Vampire:**
- Starting spells: ward (self-buff), conviction surge (buff)
- Special attack: bite (melee + life drain, heals self)
- Hybrid: primarily fights, self-buffs between engagements
- Can discover more spells (self-buff focused)

**Flesh Golem:**
- No spells
- Special: absorbs corpses in room for temporary stat buff /
  healing (extends existing consume mechanic)
- Flavor: "rips a piece from the fallen and grafts it onto itself"

## Raise Spells

Six separate spells, one per undead type:

| Spell ID | Name | Manifestation Gate | Base Folds | Cost |
|----------|------|-------------------|------------|------|
| raise-skeleton | Raise Skeleton | 1 | 4 | 20 |
| raise-zombie | Raise Zombie | 5 | 6 | 30 |
| raise-wraith | Raise Wraith | 15 | 8 | 45 |
| raise-spectre | Raise Spectre | 25 | 10 | 60 |
| raise-vampire | Raise Vampire | 40 | 12 | 80 |
| raise-golem | Raise Flesh Golem | 50 | 16 | 100 |

All spells:
- School: manifestation
- Type: neutral (targets a corpse in the room, not a player/mob)
- No quest_required — discovered via manifestation skill
- Requires a corpse in the room (spell script checks)
- Corpse is consumed on success

### Corpse Statpool Requirement

Each undead type requires the corpse to have a minimum original
statpool (sum of all stat training values). If the corpse is too
weak, the raise fails with flavor text.

| Type | Min Corpse Statpool | What qualifies currently |
|------|-------------------|------------------------|
| Skeleton | 30 | Most basic mobs (rats, pests) |
| Zombie | 60 | Standard combat mobs (bandits, scouts) |
| Wraith | 120 | Mid-tier mobs (guard captains, wardens) |
| Spectre | 200 | Strong mobs (pale lurkers, deep gnawers) |
| Vampire | 300 | Future endgame content only |
| Flesh Golem | 500 | Future raid/boss content only |

## Scaling Formula

```
raisedPool = (companionScale × 0.5) + (corpsePool × 0.5)

where:
  companionScale = CalcCompanionStatPool(typeBasePool, charisma, manifestation)
  corpsePool = sum of corpse's training values across all 6 stats
```

This means:
- 50% of the raised companion's power comes from the caster
  (same scaling as summon/conjure)
- 50% comes from the quality of the corpse
- A necro companion from a weak corpse ≈ summoned companion
- A necro companion from a strong corpse > summoned, but not
  dramatically (the corpse contribution is capped by the 50% split)

## Corpse Assessment Command

`assess <corpse>` or `assess corpse`

Uses manifestation skill. Informational only — not required
before raising. Shows:

```
You study the remains of Guard Captain Velk...
The residual essence feels substantial — enough to sustain
most forms of undeath. A wraith or even a spectre could be
bound from these remains. The flesh is still fresh enough
for corporeal raising.
```

Information conveyed (descriptively, no numbers):
- How strong the corpse is (relative to undead type thresholds)
- Which undead types it could support
- Corpse freshness (affects nothing mechanically, just flavor)

Skill progression: fires OnSkillUse for manifestation.

## Failure Text

### Corpse too weak for the type:
- "The remains shudder and twitch, but the spark of undeath
  finds nothing to cling to."
- "Dark energy seeps into the corpse... and leaks out through
  a thousand tiny fractures. The remains are too frail."
- "Bones rattle briefly, then collapse into dust. There wasn't
  enough left to work with."
- "The corpse stirs, a hollow mockery of life flickers in its
  eyes — then fades. Not enough essence remains."
- "Your magic courses through the remains, but finds only
  emptiness. This one died too thoroughly."

### Caster too weak / skill too low:
- "You reach for the corpse with your will, but the threads
  of undeath slip through your grasp."
- "The ritual begins, but the weight of what you're attempting
  overwhelms your focus. The magic dissipates."
- "Dark power gathers... and scatters. You lack the mastery
  to bind something this complex."

Both failure types consume conviction (partial cost) but not
the corpse. The corpse remains for another attempt.

## Companion Movement: Cast Interruption

**Companions interrupt casting to follow the owner.** If the
owner moves rooms, any companion mid-cast abandons the spell
and follows immediately. The owner bond takes priority.

This prevents the annoying scenario where a caster companion
gets left behind because it was mid-spell when the player moved.

Implementation: in the `go` command's charmed-mob-follow logic,
check if the companion has `CastingState != nil` and clear it
before issuing the follow command.

## Vampire Bite Special Attack

New mob combat command: `bite`

- Melee attack using Strength + unarmed-combat
- On hit: deals physical damage AND heals the vampire for a
  percentage of damage dealt (like life drain)
- Heal amount: `damage × BiteDrainPercent` (config, default 0.50)
- Can only be used by mobs with the vampire species
- Shares the special-move cooldown with bash/kick/trip
- Added to vampire mob's combatcommands

## Flesh Golem Corpse Absorption

Extends the existing mob `consume` command:

- When a flesh golem consumes a corpse, instead of the standard
  regen buff, it gets a temporary stat buff based on the corpse's
  strongest stat
- Flavor: "[golem] rips a piece from the fallen [name] and
  grafts it onto itself!"
- Duration: ~20 rounds
- Can stack from multiple corpses (up to a cap)

## New Files Needed

### Species (6 new species YAML files):
- `_datafiles/world/dogmud/species/` — skeleton, zombie, wraith,
  spectre, vampire, flesh_golem

### Mob Templates (6 new mob YAML files):
- `_datafiles/world/dogmud/mobs/summons/` — one per undead type
  with appropriate idle/combat commands, starting spells

### Spells (6 new spell YAML + JS files):
- `_datafiles/world/dogmud/spells/raise-skeleton.*`
- `_datafiles/world/dogmud/spells/raise-zombie.*`
- etc.

### Commands:
- `internal/usercommands/assess.go` — corpse assessment
- Possibly: `internal/mobcommands/bite.go` — vampire bite

### Help Files:
- `_datafiles/world/dogmud/templates/help/assess.template`
- Update manifestation help to mention necromancy

## What This Phase Does NOT Include

- Companion progression (stats/skills advancing) — Phase 5
- Companion mutations — Phase 5
- Corpse spell inheritance — future refinement
- Player necromancy (raising player corpses) — deliberately
  excluded, mob corpses only
