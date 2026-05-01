# Stillwater Marsh — Zone Sketch (Stage 3.0a of Caravan/Economy Effort)

**Date:** 2026-04-28
**Status:** Draft — awaiting review

## Goal

Build a 20-room wetland zone west of Stillwater so the Stillwater-anchored
forager NPC (Stage 3.1) has territory to roam. Theme: marsh giving way
to bog and finally to upland steppe at the southern terminus. Mirrors
Fernway South in structure (entry + branching layout + 1-2 hostile mobs +
emergent wildlife dynamic + biome-shift terminal) but with a wetland
character instead of forest.

## Multi-stage context

Stage 3.0a of the multi-stage caravan/economy effort. 3.0c (Fernway South)
just shipped to dev (commits 7e4e71f4 → dd76eb7e); 3.0a is the parallel
zone for the Stillwater-side forager. Per user direction, nothing ships
to prod (`master`) until the entire economy stack lands.

## Architecture

Pure data zone build — no Go code changes. 20 room YAMLs in a new
`stillwater_marsh/` folder, 5 mob YAMLs (4 reuse existing species, 1
reuses the mustelid species shipped in 3.0c T1), 1 zone-config YAML, 1
boundary edit on Stillwater 4133 Mill Creek Footbridge (add a west exit),
coord_map update, PATCH_NOTES entry. Same 9-task subagent-driven shape
as 3.0c.

---

## Zone metadata

- **Display name:** `Stillwater Marsh`
- **Folder:** `_datafiles/world/dogmud/rooms/stillwater_marsh/`
- **Default biome:** `water` (matches existing Stillwater wetland rooms like 4141 Sluice Pond)
- **Region:** Windward Marches
- **Roomid range:** 4197–4216 (20 rooms; the 4177–4196 range from the original draft was reserved for Fernway South — wait, no, Fernway South used 4157–4176. So 4177–4196 is free for this zone.)
- **CORRECTION:** Use roomid range **4177–4196** (the next 20 IDs after Fernway South's 4157–4176).

---

## Boundary connection

| Existing room | Direction | Target | Action |
|---|---|---|---|
| 4133 Mill Creek Footbridge (`stillwater/4133.yaml`) | west | 4177 Marsh Track | Add a `west:` exit. Today 4133 has no west exit. Description already implies "creek upstream" — the new exit makes that real. |

Single boundary edit. Same shape as Fox Den 4156 → Briar Tangle 4157 in 3.0c.

---

## Room list (20 rooms)

```
        x=-24    x=-23    x=-22    x=-21    x=-20
y=3                                [4180]                       <- Spring Pool (terminal N)
y=2                                [4179]                       <- Mill Creek Source
y=1     [4182]──[4181]──[4178]──[4177]──{4133 Footbridge}      <- entry trio
y=0     [4186]──[4185]──[4184]──[4183]                          <- Heron Marsh hub at 4184
y=-1    [4190]──[4189]──[4188]──[4187]                          <- forage row
y=-2             [4193]──[4192]──[4191]                         <- Adder Den at 4193 (terminal W)
y=-3                     [4194]   [4196]                        <- 4196 = Hidden Spring (terminal SE pocket)
y=-4                     [4195]                                 <- Far Bog Heart (terminal S, biome plains)
```

| ID | Title | Coord | Biome | Exits |
|---|---|---|---|---|
| 4177 | Marsh Track | (-21,1,0) | water | n→4179, s→4183, e→4133 (Stillwater), w→4178 |
| 4178 | Cattail Verge | (-22,1,0) | water | e→4177, w→4181 |
| 4179 | Mill Creek Source | (-21,2,0) | water | s→4177, n→4180 |
| 4180 | Spring Pool | (-21,3,0) | water | s→4179 (terminal N) |
| 4181 | Reed Beds | (-23,1,0) | water | e→4178, w→4182 |
| 4182 | Willow Grove | (-24,1,0) | water | e→4181 (terminal W) |
| 4183 | Cattail Bend | (-21,0,0) | water | n→4177, s→4187, w→4184 |
| 4184 | Heron Marsh | (-22,0,0) | water | central hub: e→4183, w→4185, s→4188 |
| 4185 | Otter Slide | (-23,0,0) | water | e→4184, w→4186, s→4189 |
| 4186 | Clam Beds | (-24,0,0) | water | e→4185, s→4190 (terminal W) |
| 4187 | Iron Seep | (-21,-1,0) | water | n→4183, s→4191 |
| 4188 | Shrimp Shallows | (-22,-1,0) | water | n→4184, s→4192, e→4187, w→4189 |
| 4189 | Sundew Hollow | (-23,-1,0) | water | n→4185, e→4188, w→4190, s→4193 |
| 4190 | Black Pool | (-24,-1,0) | water | n→4186, e→4189 (terminal W — rare pearl) |
| 4191 | Mossy Hummock | (-21,-2,0) | water | n→4187, s→4196, w→4192 |
| 4192 | Dragonfly Glade | (-22,-2,0) | water | n→4188, e→4191, w→4193, s→4194 |
| 4193 | Adder Den | (-23,-2,0) | water | n→4189, e→4192 (terminal W — **HOSTILE bog adder**) |
| 4194 | Bog Edge | (-22,-3,0) | water | n→4192, s→4195 |
| 4195 | Far Bog Heart | (-22,-4,0) | **plains** | n→4194 (terminal S — biome shift to upland) |
| 4196 | Hidden Spring | (-21,-3,0) | water | n→4191 (terminal SE pocket) |

**Coord audit:** All coords verified unclaimed against current Stillwater rooms (Stillwater occupies x=-21 to -14, y=1 to 8, with one outlier at 4140 Cemetery (-21,5,0)). My zone uses x=-21 to -24, y=-4 to 3. The (-21,1,0) entry point is free; no conflict.

---

## Forage themes (existing Stillwater mats)

The 6 existing Stillwater mats from the 3.0b audit get fresh marsh territory. Each gets a thematic home room. Per the brief, NO new mats — the forager (Stage 3.1) gathers from these via room-type association.

| Mat | Room | Notes |
|---|---|---|
| 40055 cattail down | 4178 Cattail Verge | Already in audit; deferred to forager. Cattails grow at 4178. |
| 40056 marsh willow bark | 4182 Willow Grove | Existing Stillwater mat; willow stand at 4182. |
| 40057 lake mint | 4178 Cattail Verge (or scattered) | Existing Stillwater mat. |
| 40058 freshwater clam | 4186 Clam Beds | Existing Stillwater mat. Clam beds at 4186. |
| 40059 lake-iron nodule | 4187 Iron Seep | Same mineral-seep theme as 4141 Sluice Pond. |
| 40051 skitter-shrimp shell | 4188 Shrimp Shallows | Existing Stillwater mat. Shallow water, crustaceans. |
| 40053 Stillwater black pearl | 4190 Black Pool | Rare drop. The "deep reward" room. |

Forager wiring is Stage 3.1's job — these are just narrative associations baked into room descriptions and `nouns:` blocks.

---

## Mob list (5 mobs)

Mirrors the Fernway South wildlife pattern. **1 hostile mob (bog adder)**; emergent dynamic = adder hates rats and hunts them (parallel to wolf-hates-boar).

```
river otter — species: mustelid (24, NEW from 3.0c T1!), archetype: prey
  Playful, skittish, flees on sight. REUSES the mustelid species shipped
  in Stage 3.0c — first non-badger consumer of that species. Spawns at
  4185 Otter Slide. Drops freshwater clam (40058) — otters fish for
  clams, fits naturally.

marsh rat — species: rodent (10), archetype: prey
  Smaller cousin of the wild hare. Flees on sight. Drops nothing
  intentional (or generic meat). Spawns at 4189 Sundew Hollow and
  4192 Dragonfly Glade.

dragonfly swarm — species: insectoid (12), archetype: combat_passive
  Atmospheric. Won't attack first; if attacked, mild bite damage.
  Spawns at 4192 Dragonfly Glade.

snapping turtle — species: reptile (21), archetype: combat_passive
  The "boar equivalent" — passive but fights HARD if engaged. Slow
  movement; drops nothing. Spawns at 4188 Shrimp Shallows.

bog adder — species: serpent (8), archetype: ambusher, HOSTILE
  hostile: true + hates: [rodent] — attacks players on sight AND
  hunts marsh rats. The "wolf equivalent" (predator with intra-zone
  hate target) AND "badger equivalent" (only true hostile to players)
  rolled into one. The emergent wildlife dynamic is bog-adder-vs-
  marsh-rat: walk through Sundew Hollow when both are present and
  watch the adder strike.
```

5 mobs total. All use existing archetypes. **mustelid (24)** is the only "new" species and it just shipped in 3.0c — this zone reuses it for otter, validating the species investment.

**Aggro summary:** only the bog adder (4193, but it wanders to 4189/4192 to hunt) is hostile to players. Otter, rat, dragonfly, turtle are all safe to walk past.

---

## Item suggestions

None new. The smoke checklist's "identify one zone-specific item" requirement is satisfied by foraged mats from the existing Stillwater pool (lake-iron, marsh willow bark, etc.). No curio loot per the same call we made on 3.0c.

---

## Tone notes

The Stillwater Marsh is the wet-and-wild edge of the lake basin. Where Fernway South tapers from forest to steppe, Stillwater Marsh tapers from cultivated waterway (Sluice Pond, the watermill, the creek) to wild bog. The theme is *water everywhere, footing uncertain* — board-trail in the early rooms, dry hummocks in the south, then suddenly upland at the Far Bog Heart terminus. The bog adder is the "stay alert" reminder. For foragers (Stage 3.1) this is home territory: enough roaming room (20 rooms) for the Stillwater-anchored forager, with a real if rare threat (the adder + occasional snapping turtle miscalculation) that justifies fold-recall.

The single biome-shift at 4195 Far Bog Heart (water → plains) parallels Fernway South's 4175 Steppe Edge (forest → plains) — a one-way view of the Dustwalk steppe beyond, hinting at unmapped territory south.

---

## Out of scope (explicitly)

- **Quests in this zone** — Phase 2 work, deferred. Aligns with Stage 3.1 forager wiring.
- **New foragable mats** — the existing 6 Stillwater mats are the supply pool.
- **New zone-specific items** — no curio loot.
- **Heron mob** — no avian species exists; heron stays as atmospheric noun in 4184 Heron Marsh.
- **Frog/bullfrog mob** — amphibians don't fit any current species cleanly; stay as nouns.
- **Beaver mob / dam interactivity** — could be a future Stage 3.0a follow-up but skipped for v1 to keep the species investment minimal.
- **Stillwater coord-map catch-up** — the coord map is missing all Stillwater rooms (it's stale). The 3.0a docs task should add the new zone AND the existing Stillwater rooms (~48 rows) similar to how 3.0c added Fernway catch-up.

---

## Implementation order (preview for the plan stage)

Follows the 9-task 3.0c shape exactly:

1. (No new species needed — mustelid already exists from 3.0c T1.) **T1: zone-config + entry trio (4177-4179)** + Mill Creek Source (4180)? Actually let me re-shape to mirror 3.0c task counts.

Better shape:

1. **T1: Zone-config + entry trio** (4177, 4178, 4179) [3 rooms]
2. **T2: Northern source** (4180, 4181, 4182) [3 rooms — Spring Pool + W trail]
3. **T3: Heron Marsh hub + W branch** (4183, 4184, 4185, 4186) [4 rooms]
4. **T4: Forage row** (4187, 4188, 4189, 4190) [4 rooms]
5. **T5: Adder pocket + south spine + steppe** (4191, 4192, 4193, 4194, 4195, 4196) [6 rooms; includes 4195 biome:plains and 4193 hostile-mob terminal]
6. **T6: 5 mob YAMLs** (otter, rat, dragonflies, turtle, adder) — mobids 366-370
7. **T7: Spawninfo wiring + Mill Creek Footbridge west exit**
8. **T8: Coord map (Stillwater catch-up + new zone) + PATCH_NOTES**

8 tasks (one fewer than 3.0c since no new species needed).

---

## Done = ?

All 8 tasks complete, all commits landed on `development` branch, manual smoke verification green. Per the multi-stage caravan/economy effort: this lands on `development` only. Nothing ships to `master` until Stage 3.4 lands.
