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

Both zones need more rooms and mobs to cover equipment slots and
provide a proper progression experience.

### Arena Layout (ejected, no recall)
- Entry Chamber (safe, return portal)
- Gauntlet rooms (2-3 rooms, trash mobs, wave-like respawn)
- Tough room (1 room, tough mobs with gear)
- Boss pit (1 room, arena champion boss)

### Oasis Layout (rejoin, recall allowed)
- Oasis Threshold (safe, return portal)
- Elemental wastes (2-3 rooms, trash elemental mobs)
- Elemental nexus (1 room, tough elementals with gear)
- Oasis Heart (boss room, king/queen/prince)

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
| Crystal Golem (tough) | tough | stone ring | ring |
| Magma Hound (tough) | tough | ember boots | feet |
| Void Wisp (tough) | tough | shadow necklace | neck |
| Gale Dancer (tough) | tough | storm bracer | wrist |

**Coverage:** weapon, offhand, head, neck, shoulders, body, wrist,
gloves, ring, legs, feet = 11 slots. Missing (intentional): back,
belt, component bag (utility slots).

### New Mobs Needed

**Arena:**
- Arena Veteran (tough, statpool 2) — human fighter with gear
- Arena Champion (boss, statpool 4) — arena boss

**Oasis:**
- Crystal Golem (tough, statpool 2) — defensive elemental
- Magma Hound (tough, statpool 2) — aggressive fire creature
- Void Wisp (tough, statpool 2) — evasive magical creature
- Gale Dancer (tough, statpool 2) — fast wind elemental

### New Base Items Needed

All the items in the loot table above need base item YAMLs with
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
7. Create new mob templates (arena veteran/champion, oasis tough mobs)
8. Expand zone layouts with new rooms
9. Update spawn info with new mobs and loot pools
10. Test at various gold levels
