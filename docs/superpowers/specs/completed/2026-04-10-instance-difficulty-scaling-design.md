# Instance Difficulty Scaling Design

**Date:** 2026-04-10
**Scope:** Sub-project 2 of 3 — Difficulty scaling for instanced zones.
Builds on the instance framework (sub-project 1). Loot/affixes are
sub-project 3.

---

## Overview

The gold amount paid to the Riftkeeper directly determines mob power
inside the instance. The scaling is aggressive and roughly linear —
pay 500g, mobs have ~500 stat pool. Pay 5000g, mobs have ~5000. The
economy self-regulates: players can only spend what they've earned,
so the difficulty ceiling is naturally gated by progression.

---

## Scaling Formula

Template stat pools act as **multipliers** that preserve relative
difficulty between mobs within a zone:

```
effectiveStatPool = GoldPaid * templateStatPool
```

A trash mob template with `statpool: 1` gets exactly `GoldPaid` as
its stat pool. A tough mob with `statpool: 2` gets double. A boss
with `statpool: 3` gets triple.

This means:
- At 500g: trash = 500, tough = 1000, boss = 1500
- At 2000g: trash = 2000, tough = 4000, boss = 6000

### Stat Pool Cap

Optional safety cap at 50000 to prevent absurd edge cases from
exploited gold. Configurable in balance config as
`InstanceStatPoolCap`.

---

## Where It's Applied

In `CreateZoneInstance()` in `internal/rooms/instances.go`, after
cloning the ephemeral rooms but before registering the instance.
Iterate every room's `SpawnInfo` and multiply `StatPool` by
`GoldPaid`:

```go
for _, ephId := range roomIdMap {
    if room := LoadRoom(ephId); room != nil {
        for i := range room.SpawnInfo {
            if room.SpawnInfo[i].StatPool > 0 {
                room.SpawnInfo[i].StatPool *= goldPaid
            } else {
                room.SpawnInfo[i].StatPool = goldPaid
            }
            if cap > 0 && room.SpawnInfo[i].StatPool > cap {
                room.SpawnInfo[i].StatPool = cap
            }
        }
    }
}
```

Mobs spawn normally via `Room.Prepare()` using the overridden stat
pool. The existing mob stat generation distributes the pool across
stats based on archetype (fighting = 80% physical / 20% mental,
casting = inverse, balanced = uniform).

---

## Template Stat Pool Conventions

Instance zone mob templates use stat pool as a **relative power
multiplier**, not an absolute stat value:

| Template StatPool | Role | Example |
|------------------|------|---------|
| 1                | Trash mob | arena grunt, sand elemental |
| 2                | Tough mob | elite fighter, armored golem |
| 3                | Boss | elemental king/queen/prince |

These values are intentionally small. The `GoldPaid` multiplication
makes them meaningful.

Mobs with `statpool: 0` (or unset) default to `GoldPaid * 1`.

---

## What the Scaling Does NOT Touch

- **Mob equipment**: gear is defined on the mob template and stays
  as-is. Better-geared mobs are authored directly. (Sub-project 3
  will add randomized loot drops, not mob equipment scaling.)
- **Mob AI**: tactical presets, discipline, combat commands are all
  per-template. Bosses are authored with sophisticated AI; trash
  mobs with simple patterns.
- **Number of mobs**: spawn count is defined in room `SpawnInfo`
  and not modified by scaling. More mobs = author more spawn
  entries in the template.
- **Mob skills**: skill ranks come from the template. Higher stat
  pools make skills more effective naturally (via the
  `SkillMultiplier` curve).

---

## Config

Add to balance config:

```yaml
InstanceStatPoolCap: 50000  # Maximum effective stat pool per mob
```

Default 50000. Set to 0 to disable the cap.

---

## Content: New Instance Mobs

Not part of the scaling engine, but planned content for the two
test zones. Created as normal mob YAML/JS files.

### Arena Mobs (new, unique to instances)
- Trash mob(s) — fighters the player hasn't seen in the overworld
- At least one distinct mob type per arena room

### Planar Oasis Mobs
- Existing elementals (earth, water, fire, air) as trash
- New tough variants for higher-tier encounters
- Three boss variants (randomized per instance or per room):
  - **Elemental King** — fighting archetype, heavy melee
  - **Elemental Queen** — casting archetype, spell-heavy
  - **Elemental Prince** — skullduggery/rogue archetype, fast
    and evasive
- Boss spawns use `statpool: 3` in templates

Boss variant selection (which boss spawns) can be randomized via
JS room scripts or by having multiple spawn entries with low spawn
rates. Content design detail, not engine work.

---

## Implementation Summary

The engine change is minimal — roughly 10-15 lines added to
`CreateZoneInstance()`. The rest is content: new mob templates,
zone room updates, and balance tuning.

1. Add `InstanceStatPoolCap` to balance config
2. Add stat pool scaling loop to `CreateZoneInstance()`
3. Update instance zone templates with multiplier-based stat pools
4. Create new mob templates for arena and oasis zones
5. Test at various gold levels to verify scaling feels right
