# Instance Randomized Loot Design

**Date:** 2026-04-10
**Scope:** Sub-project 3 of 3 — Randomized loot for instanced zones.
Builds on framework (sub-project 1) and difficulty scaling (sub-project
2). Also includes fleshing out zone layouts.

---

## Overview

Instance mobs spawn wearing randomly-generated gear that makes them
tougher and drops when they die. A point-budget system determines item
power based on the gold paid for the instance. Points are randomly
distributed across eligible bonus types, creating natural variety —
some items are damage-focused, some defensive, some hybrid.

---

## Point Budget

```
budget = floor(7 * sqrt(goldPaid))
```

Gaussian variance applied via `dice.RollStat(budget)` for ±15% spread.

| Gold | Base Budget |
|------|-------------|
| 200  | 99          |
| 500  | 156         |
| 1000 | 221         |
| 2000 | 313         |
| 5000 | 495         |
| 10000| 700         |

Config: `LootBudgetScalar` (default 7.0) in balance config for tuning.

---

## Point Costs

| Bonus Type | Per-Unit | Cost | Notes |
|-----------|----------|------|-------|
| Damage mult (phys+spell) | +0.01 | 12 pts | Caster weapons only |
| Damage mult (phys only) | +0.01 | 8 pts | Non-caster weapons |
| Physical mitigation | +1% | 5 pts | Armor only |
| Magical mitigation | +1% | 5 pts | Armor only |
| Conviction mitigation | +1% | 5 pts | Armor only |
| Stat bonus | +1 | 3 pts | All items |
| Skill bonus | +1 rank | 12 pts | All items |

---

## Eligibility by Item Type

**Weapons (non-caster):** phys-only damage mult, stats, skills
**Weapons (caster — wand/sceptre/staff):** phys+spell damage mult,
stats, skills
**Armor/wearables:** all three mitigations, stats, skills
**All items:** stats and skills always eligible

### Eligible Stats
All 6: strength, dexterity, perception, vitality, willpower, charisma

### Eligible Skills (combat/progression only)
weapon-combat, unarmed-combat, skullduggery, spellcasting, rhetoric,
manifestation

Excluded: alchemy, blacksmithing, cooking, enchanting, jewelcrafting,
tailoring, bartering, salvage, search

---

## Affix Generation Algorithm

When a mob spawns in an instance room:

1. Check if mob template has a `loot_pool` (list of base item IDs)
2. Pick a random item ID from the pool
3. Create a new item instance from the template
4. Read `gold_paid` from room temp data
5. Calculate budget: `floor(7 * sqrt(goldPaid))`
6. Roll variance: `budget = dice.RollStat(budget)`
7. Build eligible bonus list based on item type/subtype
8. Loop:
   a. Pick a random eligible bonus type
   b. If budget >= cost, spend cost and apply +1 unit of that bonus
   c. Repeat until budget < cheapest remaining bonus cost
9. Equip the item on the mob

The random selection in step 8a means points spread naturally across
multiple bonus types. Items will typically have 2-5 different bonuses
depending on budget size and luck.

---

## Item Display

Affixed items need to be visually distinct from base items. Options:

- Prefix the item name based on highest bonus type: "Reinforced
  Shield", "Keen Scimitar", "Warding Crown"
- Or suffix: "Shield of the Elements", "Scimitar of Precision"
- The item description stays as the base template description

Naming can be a simple lookup: if highest point spend was on damage
mult → "Keen", if mitigation → "Warding", if stats → "Empowered",
if skills → "Masterwork". Multiple prefixes if budget was large
enough.

---

## Mob Loot Tiers

- **Trash mobs** (statpool 1): no loot pool, no item drops
- **Tough mobs** (statpool 2): 1 item in loot pool, equipped
- **Bosses** (statpool 4): 2 items in loot pool, both equipped

### Mob Template Field

New field on mob YAML:

```yaml
loot_pool:
  - 10100
  - 10101
```

When the mob spawns in an instance, one item is randomly selected from
the pool (for tough mobs) or two items are selected (for bosses, one
per pool entry — both equipped).

For bosses with 2 items: both items are rolled independently with
separate budgets and equipped in appropriate slots.

---

## When Affixes Are Generated

**At mob spawn time** — NOT at instance creation. Each individual mob
gets its own unique rolls. Two of the same mob type in the same room
will have different gear. This means:

- Hook into `Room.Prepare()` mob spawn path
- After mob is created with scaled stat pool
- Check room temp data for `gold_paid` (only applies in instances)
- If `gold_paid` > 0 and mob has `loot_pool`, generate and equip

---

## Drop Mechanic

Uses existing `itemdropchance` on mob templates. When the mob dies,
the engine's existing item drop system handles whether equipped items
drop. The affixed item drops as-is — the player gets exactly what the
mob was wearing.

---

## Zone Layout Expansion

Two completely different gameplay experiences:

### Arena Layout (ejected, no recall) — Linear Gauntlet
- Entry Chamber (safe, return portal)
- Gauntlet rooms (2-3 rooms, trash mobs, fast 30m gametime respawn)
- Tough room (1 room, tough mobs with gear)
- Boss pit (1 room, arena champion boss)

The arena is a short, brutal gauntlet with wave-like fast respawns.
Players push through as far as they can before dying or retreating.
Linear room progression, no navigation challenge.

### Oasis Layout (rejoin, recall allowed) — 3D Wrapping Cube

The oasis is a **5x5x5 cube** of 125 rooms with wrapping exits.
Going north from (x,4,z) takes you to (x,0,z). Going up from
(x,y,4) takes you to (x,y,0). All 6 directions wrap. This creates
a disorienting 3D space where navigation is the secondary challenge.

**Structure:**
- **Oasis Threshold** — safe entry room, outside the cube, with
  return portal
- **Cube entrance** — Oasis Threshold connects north to room
  (2,2,0), the bottom-center of the cube. That room's south exit
  goes back to the entrance. This is the ONLY non-wrapping exit.
- **125 cube rooms** — each gets 1 trash elemental mob (randomly
  selected from water, earth, air, fire, magma at instance creation)
- **2 tough mobs** (sand elemental, storm elemental) — placed in
  random rooms within the cube
- **1 boss** (king, queen, or prince — randomly selected) — placed
  in a random room within the cube

**Respawn:** longer than the portal duration (e.g., 2 real hours).
Once a room is cleared, it stays cleared for the run. The oasis is
a cube to be methodically cleared, not farmed.

**Wandering:** All elementals wander through the cube. The boss and
tough mobs also wander. Players may encounter the boss unexpectedly
while clearing trash, or the boss may wander into them.

**Room generation:** The 125 cube rooms are generated programmatically
at instance creation time by a cube generator function. Room
descriptions pull from a pool of elemental biome templates (crystal
wastes, scorched sand, frozen dunes, storm-torn flats, etc.) assigned
randomly per room.

**Cube Generator Function:**
Added to `CreateZoneInstance` (or called from it). For the oasis zone:
1. Generate 125 ephemeral rooms with coords (0-4, 0-4, 0-4)
2. Connect all 6 exits per room with wrapping:
   - north: (x, y+1 mod 5, z)
   - south: (x, y-1 mod 5, z)
   - east: (x+1 mod 5, y, z)
   - west: (x-1 mod 5, y, z)
   - up: (x, y, z+1 mod 5)
   - down: (x, y, z-1 mod 5)
3. Override room (2,2,0) south exit to point to Oasis Threshold
   (non-wrapping)
4. Assign each room a random description from the biome pool
5. Assign each room 1 random trash elemental spawn
6. Pick 2 random rooms for tough mob spawns
7. Pick 1 random room for the boss spawn (randomly select king,
   queen, or prince)

The cube rooms are NOT pre-authored YAML files. They are created
entirely in Go code during instance creation using the ephemeral
room system. The Oasis Threshold entry room IS a pre-authored
template (existing room 5003).

### Loot Slot Coverage

Each tough/boss mob drops a specific equipment slot. Between both
zones, all combat-relevant slots are covered:

**Arena (humanoid gear):**

| Mob | Type | Item | Slot |
|-----|------|------|------|
| Arena Champion (boss) | boss | warhammer | weapon |
| Arena Champion (boss) | boss | tower shield | offhand |
| Arena Veteran | tough | chain gloves | gloves |
| Arena Veteran | tough | iron greaves | legs |

**Oasis (elemental gear):**

| Mob | Type | Item | Slot |
|-----|------|------|------|
| Elemental King (boss) | boss | obsidian mace | weapon |
| Elemental King (boss) | boss | volcanic plate | body |
| Elemental Queen (boss) | boss | crystal sceptre | weapon |
| Elemental Queen (boss) | boss | ice crown | head |
| Elemental Prince (boss) | boss | wind scimitar | weapon |
| Elemental Prince (boss) | boss | mist pauldrons | shoulders |
| Sand Elemental (tough) | tough | stone ring | ring |
| Storm Elemental (tough) | tough | storm bracer | wrist |

Note: fewer tough mobs in oasis (2 per instance) but the boss always
has 2 items. Remaining slots (feet, neck) can be added to future
instance zones or as additional tough mob variants.

**Coverage:** weapon, offhand, head, shoulders, body, wrist, gloves,
ring, legs = 9 slots. Partial coverage with room for expansion.

### New Mobs Needed

**Arena:**
- Arena Veteran (tough, statpool 2) — human fighter with gear
- Arena Champion (boss, statpool 4) — arena boss

**Oasis (additions to existing):**
Existing trash elementals (water 313, earth 311, air 312, fire 310,
magma 314) are used as cube trash. Existing sand (318) and storm
(319) are the tough mobs. Existing king/queen/prince (320-322) are
the boss variants. No new oasis mobs needed — the cube generator
assigns them.

### New Base Items Needed

All the items in the loot tables above need base item YAMLs with
appropriate stats for their slot. These are the "templates" before
affixes are applied — they should have reasonable base stats that
make sense for the slot but nothing extraordinary. The affix system
adds the magic.

---

## Config

Add to balance config:

```yaml
LootBudgetScalar: 7.0  # Multiplier for sqrt(goldPaid) budget calc
```

Point costs could also be config-driven but YAGNI — hardcode them
initially and extract to config only if tuning becomes frequent.

---

## Implementation Summary

1. Add `loot_pool` field to mob template struct
2. Add `LootBudgetScalar` to balance config
3. Create affix generation engine (budget calc, point distribution,
   item creation)
4. Hook into mob spawn path to generate and equip affixed items
5. Add item naming system for affixed items
6. Create new base item YAMLs for all loot table entries
7. Create new mob templates (arena veteran/champion)
8. Expand arena zone layout with new rooms
9. Build oasis cube generator function (125 rooms, wrapping exits,
   random mob placement, random room descriptions)
10. Create biome description pool for cube rooms
11. Update Riftkeeper NPC script for new oasis behavior
12. Update spawn info and loot pools on all instance mobs
13. Test at various gold levels — both zones
