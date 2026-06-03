# Planar Oasis: All Bosses + Elemental Princess + Larger Grid

**Date:** 2026-05-29
**Zone:** Instance Planar Oasis (`instance_planar_oasis`)
**Type:** Content + small engine change (cube generator)

---

## Overview

Three changes to the procedurally-generated Planar Oasis instance:

1. **Spawn all bosses** instead of one random boss per run.
2. **Add a fourth boss**, the Elemental Princess (water), who drops two
   instance-scaled items — a set of unarmed **claws** and a **neck** piece.
3. **Grow the cube grid** from 4×4×4 (64 rooms) to 5×5×5 (125 rooms).

The live oasis is built at instance-creation time by
`rooms.GenerateOasisCube` in `internal/rooms/cubegen.go`; the authored
rooms 5003–5005 are just the entry threshold + legacy. All three changes
are small edits to that generator plus three new data files.

"Scaling gear" is the existing instance **affix system**
(`internal/items/affixgen.go`): base item YAMLs listed in a mob's
`loot_pool` receive random affixes scaled by gold paid
(`budget = floor(7·√goldPaid)`). The princess's two items are plain base
templates; the affix engine does the scaling. No new scaling mechanic.

---

## Locked Decisions

| Decision | Value |
|----------|-------|
| Boss spawning | All bosses spawn per run, each in its own distinct room |
| Bosses | King (320), Queen (321), Prince (322), **Princess (377, new)** |
| Grid | `cubeSize` 4 → 5 (64 → 125 rooms) |
| Princess element | Water (species 36), agile melee "huntress" |
| Princess loot | Claws (weapon) + Neck piece, both instance-scaled |
| Claws skill | Unarmed-combat (via `subtype: claws`) — applies to any wielder |
| Claws base stats | `damage_multiplier: 0.45`, `speedmultiplier: 1.20`, basedamage 4/var2 |

### New IDs (verified free)
- Princess mob: **377** (`instance_planar_oasis/377-elemental_princess.yaml`)
- Claws weapon: **10036** (`weapons-10000/`)
- Neck armor: **20079** (`armor-20000/neck/`)

---

## Change 1 — `internal/rooms/cubegen.go`

### Grid size
- `cubeSize = 4` → `cubeSize = 5` (so `cubeTotal` = 125). `cubeIndex`,
  wrapping exits, and the `(2,2,0)` entry all derive from `cubeSize`, so
  no other math changes.
- `cubeTitles` currently has 4 entries (z 0–3). With z now reaching 4,
  add a 5th title (e.g. `"Elemental Apex"`) so the top layer doesn't fall
  back to the generic `"Planar Wastes"`.

### Spawn all bosses, each in its own room
- `cubeBossMobs = []int{320, 321, 322}` → `{320, 321, 322, 377}`.
- Replace the single-boss placement:

  ```go
  // before: one random boss in one room
  bossIndices := pickUniqueIndices(1, cubeTotal, toughIndices)
  for _, idx := range bossIndices {
      rooms[idx].SpawnInfo = []SpawnInfo{{ MobId: cubeBossMobs[util.Rand(len(cubeBossMobs))], ... }}
  }
  ```

  ```go
  // after: every boss, each in its own distinct room
  bossIndices := pickUniqueIndices(len(cubeBossMobs), cubeTotal, toughIndices)
  for i, idx := range bossIndices {
      rooms[idx].SpawnInfo = []SpawnInfo{{
          MobId:        cubeBossMobs[i],
          StatPool:     4,
          RespawnRate:  "2 real hours",
          ForceHostile: true,
          MaxWander:    5,
          IdleCommands: wanderIdle,
      }}
  }
  ```

- `pickUniqueIndices` already excludes the tough-mob rooms; with 125 rooms
  and 2 tough + 4 boss = 6 special rooms there is ample space. Each boss
  wanders independently (existing `MaxWander: 5` + wander idle).

---

## Change 2 — Elemental Princess (mob 377)

`_datafiles/world/dogmud/mobs/instance_planar_oasis/377-elemental_princess.yaml`

- `zone: Instance Planar Oasis`, `speciesid: 36` (water elemental),
  `statpool: 4`, `hostile: true`, `groups: [elemental]`,
  `routine: planar_oasis_pack` (shares pack-tactics with the other bosses).
- `archetype: fighting`, `behavior_archetype: leader` (matches king/prince).
- Stats weighted for an agile skirmisher: high dexterity, moderate
  perception (mirrors the prince's profile).
- `skills: { unarmed-combat: 5 }` — she fights with claws.
- `itemdropchance: 75` (matches the other bosses).
- `loot_pool: [10036, 20079]` — claws + neck, both affix-scaled on spawn.
- Water/claw-themed `combatcommands` (strike-and-retreat flavor).
- Species 36 already has all equipment slots enabled (from the 2026-05-29
  companion-gear fix), so she can equip the claws and neck piece.

---

## Change 3 — Claws weapon (item 10036)

`_datafiles/world/dogmud/items/weapons-10000/10036-drowned_claws.yaml`

```yaml
itemid: 10036
name: drowned claws
namesimple: claws
vendor_categories: [blacksmithing]
type: weapon
subtype: claws          # -> routes to unarmed-combat for ANY wielder
hands: 1
damage_multiplier: 0.45
speedmultiplier: 1.20
damage: { basedamage: 4, variance: 2 }
parryrating: 2
staminacost: 3
grapplemodifier: 0.5
weight: 0.8
value: 70
description: >-
  Four curved talons of solidified oasis water, clear as glass and
  cold to the touch. They flex like living tendrils between strikes,
  then snap rigid at the moment of impact. Water beads and runs along
  the edges without ever dripping free.
```

`subtype: claws` is what makes the engine resolve attacks with this weapon
to `unarmed-combat` (`characters.CombatSkillTagForItem`). This is the
core "behaves as intended" requirement — verified in code, no engine
change needed.

### Scaling vs. Chrysalis Knuckles (the unarmed benchmark)

Knuckles are a fixed craftable: `damage_multiplier 0.55`, speed 1.15, no
stats/skills. The claws scale via the affix engine. Simulated (40k trials,
real `affixgen.go` algorithm, ±15% budget variance):

| | 100g (budget ~70) | 300g (budget ~121) | Knuckles |
|---|---|---|---|
| Final damage_mult (avg) | 0.57 | 0.65 | 0.55 |
| Final damage_mult (10–90%) | 0.45–0.65 | 0.50–0.80 | 0.55 |
| + Stat points | ~6.8 | ~11 | none |
| + Skill ranks | ~2.5 | ~4.5 | none |
| Speed | 1.20 | 1.20 | 1.15 |

Intent: a **sidegrade** at cheap runs (knuckles stay relevant) that scales
into a clear prize at higher buy-in, with the affix stats/skills (and
faster swing) as the added reward.

---

## Change 4 — Neck piece (item 20079)

`_datafiles/world/dogmud/items/armor-20000/neck/20079-tidal_torc.yaml`

```yaml
itemid: 20079
name: tidal torc
namesimple: torc
vendor_categories: [jewelcrafting]
type: neck
subtype: wearable
magical_mitigation: 6
physical_mitigation: 3
weight: 1.0
value: 85
description: >-
  A circlet of ever-flowing oasis water held in a perfect ring by
  some quiet planar will. It turns and ripples against the wearer's
  throat without spilling, and the air around it carries the cool,
  mineral smell of deep water.
```

As a non-weapon (armor) item, its affixes draw from the
mitigation/stat/skill pool. Base mitigations kept modest; the affix engine
adds the scaled power. Fills the previously-uncovered **neck** slot.

---

## Files Touched

| File | Change |
|------|--------|
| `internal/rooms/cubegen.go` | `cubeSize` 4→5, 5th title, add 377 to bosses, place all bosses |
| `internal/rooms/cubegen_test.go` | New/updated assertions (see Testing) |
| `mobs/instance_planar_oasis/377-elemental_princess.yaml` | New boss |
| `items/weapons-10000/10036-drowned_claws.yaml` | New claws |
| `items/armor-20000/neck/20079-tidal_torc.yaml` | New neck piece |

---

## Testing

1. **Unit (`cubegen_test.go`):** after `GenerateOasisCube`, assert exactly
   125 rooms; exactly 4 boss spawns present (mob ids {320,321,322,377}),
   each in a distinct room, none colliding with the 2 tough-mob rooms.
2. **`id_inventory.py`** collision pass after authoring (377 / 10036 / 20079).
3. **Boot smoke** (instance saves wiped per SOP): server boots clean past
   data-file loading — validates the new mob + item YAMLs and the
   `subtype: claws` weapon load without panics.
4. **In-instance check (manual/optional):** enter an instance, confirm all
   four bosses present and the princess wields claws that train
   unarmed-combat; loot drops the affixed claws + torc.

---

## Out of Scope

- Rebalancing the existing three bosses or trash/tough mobs.
- Changes to the affix engine itself (point costs, weights).
- Difficulty retuning for the larger grid / 4-boss load beyond the
  natural scaling already in place.
