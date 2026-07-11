# The Confluence — District 1: The Landings — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Confluence's first district — the 16-room river-ward waterfront (rooms 6106–6121), opening the River Road 6105 seam, debuting the Quayfolk faction with a barge master + two vendors + ambient crew, and planting one mystery seed. No quest.

**Architecture:** Pure content build (room/mob/item/dialogue/faction YAML). New zone `The Confluence` (folder `the_confluence/`) with its own `zone-config.yaml`. New `quayfolk` faction (one YAML; mobs join via `groups:`). Reuses the River Road content patterns verbatim. Validated by boot smoke-test + `cartcheck`.

**Tech Stack:** GoMud engine; YAML under `_datafiles/world/dogmud/`; `cartcheck` admin command; boot `ValidateZoneConsistency`.

**Spec:** `docs/superpowers/specs/2026-06-26-confluence-landings-design.md`
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md`

---

## Reference patterns (read before authoring)

- **Rooms (water biome, block-scalar nouns, spawninfo):** `_datafiles/world/dogmud/rooms/river_road/6093.yaml`, `6096.yaml` (bluff/overlook), `6100.yaml` (the waystone — model the Old Mole mystery seed on this).
- **Seam edit pattern:** `_datafiles/world/dogmud/rooms/river_road/6105.yaml` (the file Task 1 edits) and how River Road's north seam opened — `rooms/amber_valley/6071.yaml`.
- **Ambient NPC mob:** `_datafiles/world/dogmud/mobs/river_road/9410-carew_the_road_warden.yaml`. **Vendor mob:** `_datafiles/world/dogmud/mobs/river_road/9412-birrel_the_netmender.yaml` (cooking vendor + `shop:` list).
- **Dialogue:** `_datafiles/world/dogmud/dialogue/river_road/9413.yaml` (schema; first-person text, player-perspective hints).
- **Item (forageable/vendor good):** `_datafiles/world/dogmud/items/materials-40000/40125-smoked_river_fish.yaml`.
- **Faction definition:** `_datafiles/world/dogmud/factions/np_dockfolk.yaml` (the exact pattern for `quayfolk`).
- **zone-config:** `_datafiles/world/dogmud/rooms/river_road/zone-config.yaml`.

### Cross-cutting GOTCHAS (apply to every task)

1. **Zone folder = underscores:** `rooms/the_confluence/`, `mobs/the_confluence/`, `dialogue/the_confluence/`. Each file's `zone:` = `The Confluence` (with space).
2. **Every zone needs `zone-config.yaml`** — missing → boot panic "No zone-config.yaml was loaded for roomId" (the River Road lesson). Task 1 creates it.
3. **`nouns` values + multi-sentence `description` use `>` block scalars** (prose colons break unquoted scalars).
4. **Coords: the spec's grid is proposed — the build assigns final coords and MUST `cartcheck the_confluence` / boot `ValidateZoneConsistency` (mode=panic) and resolve any collision** against River Road and all prior zones before committing. Reciprocal exits.
5. **Mob filename = `{mobid}-{ConvertForFilename(name)}.yaml`** (lowercase, a-z/0-9, drop apostrophes, else `_`) — mismatch panics at boot.
6. **Vendor items need a real `vendor_categories`** from `ValidCraftSupports` minus `general` — never `general` on items.
7. **Faction:** create only `factions/quayfolk.yaml`; mobs join via a `groups:` entry `quayfolk`. The `factions.rep/` file is runtime-generated and gitignored — do NOT create it.
8. **Instance saves shadow templates** — wipe `rooms.instances/*` + `mobs.instances/*` (NOT `shops/`) before any smoke test.
9. **Dispatch subagent paths with the FULL `_datafiles/world/dogmud/...` prefix.**

---

## Task 1: Branch + zone groundwork (zone-config, faction, open the seam)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_confluence/zone-config.yaml`
- Create: `_datafiles/world/dogmud/factions/quayfolk.yaml`
- Modify: `_datafiles/world/dogmud/rooms/river_road/6105.yaml`

- [ ] **Step 1: Branch**

Run:
```bash
git checkout master && git pull --ff-only 2>/dev/null; git checkout -b feature/confluence-landings
```
Expected: `Switched to a new branch 'feature/confluence-landings'`.

- [ ] **Step 2: Create the zone-config**

`_datafiles/world/dogmud/rooms/the_confluence/zone-config.yaml`:
```yaml
name: The Confluence
roomid: 6106
defaultbiome: water
region: The Tri-Rivers
```
(Verify `The Tri-Rivers` is not an existing region name — grep `region:` across `rooms/*/zone-config.yaml`; if it collides, the design intends a NEW region, so pick an unused variant and note it.)

- [ ] **Step 3: Create the quayfolk faction**

`_datafiles/world/dogmud/factions/quayfolk.yaml` (model on `np_dockfolk.yaml`):
```yaml
faction_id: quayfolk
display_name: "The Confluence Quayfolk"
description: |
  The river-trade community of the Confluence waterfront — barge masters,
  dockmasters, fish-traders, chandlers, and the stevedores who keep the
  three rivers' cargo moving. Pragmatic and outward-facing, they care more
  about tides and freight than temple politics, and they measure a stranger
  by whether they're good for the work. They run the passage downriver to
  New Plymouth and the markets that feed the city.
default_rep: 10
allies: []
enemies: []
```

- [ ] **Step 4: Open the River Road 6105 seam**

Edit `rooms/river_road/6105.yaml`:
- Add to `exits:` → `  south: {roomid: 6106, zone: The Confluence}` (keep `northwest`).
- Revise `description`: the gate is now **open** — the river-warden returned to
  post, the season turned, the bar lifted; the road runs on under the city's
  first buildings. Keep the voice and the "stones older than the bar" detail.
- Update `the gates` noun (the bar raised, the warden back) and `the road south`
  noun (the city reachable now, not "come back when it opens").

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/zone-config.yaml _datafiles/world/dogmud/factions/quayfolk.yaml _datafiles/world/dogmud/rooms/river_road/6105.yaml
git commit -m "feat(confluence): zone groundwork — the_confluence zone-config, quayfolk faction, open 6105 seam"
```

---

## Task 2: The Landings rooms (6106–6121)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_confluence/6106.yaml` … `6121.yaml`

Author all 16 per the spec table. `zone: The Confluence`, `biome: water`,
hard-wrap ≤80, block-scalar nouns, 3 `idlemessages` each (river-trade waterfront
flavor: gulls, the slap of water on pilings, cargo sounds, river-mud smell). Add
`spawninfo` for the rooms hosting NPCs (Task 4 IDs are fixed):

```yaml
spawninfo:
  - mobid: 9419
    respawnrate: "20 real minutes"
```

| Room | Title | Coord | Mob | Content brief |
|------|-------|-------|-----|---------------|
| 6106 | The River Gate | {-5,-52,0} | 9419 | Entry from 6105 (north); the just-opened gate, customs plaza, the Gate-Warden back at post |
| 6107 | Gate Street | {-5,-53,0} | — | The way in; first city buildings, dock noise ahead |
| 6117 | Gate Plaza | {-6,-53,0} | — | Small square west of the gate; notice-board, a fountain or well |
| 6108 | The Quayhead | {-5,-54,0} | 9422 | Customs/quay office; the Dockmaster; ledgers, the working heart |
| 6113 | Warehouse Row | {-6,-54,0} | — | Bonded warehouses, stacked freight, big doors |
| 6109 | The Barge Dock | {-4,-54,0} | 9420 | The moored Confluence↔NP barge; the Barge Master; Davan's departure point; a `the barge` noun |
| 6119 | Market Lane | {-5,-55,0} | — | The street tying inland to waterfront; stalls |
| 6114 | The Chandlery | {-6,-55,0} | 9423 | Ship/dock goods; the Chandler; rope, tar, lamps |
| 6110 | The Fish Quay | {-4,-55,0} | 9421 | River-fish market; the Fish-Trader; baskets on river-weed |
| 6120 | Quay Crossing | {-5,-56,0} | — | Central quay junction |
| 6115 | Cooper's Yard | {-6,-56,0} | 9426 | Dock crafts; barrels, the cooper/net-mender |
| 6111 | The Long Wharf | {-4,-56,0} | 9425 | Cargo wharf; stevedores at work |
| 6118 | The Old Mole | {-3,-56,0} | — | An old stone jetty; **the mystery seed** — a pre-Founding mooring-stone with a weathered orbital mark. `the mooring-stone` + `the mark` nouns (model on river_road 6100; incidental, no trigger, no emphasis) |
| 6116 | The Quayside Tavern | {-6,-57,0} | 9424 | Dockside tavern; the tavern-keeper; warmth, gossip |
| 6112 | Three-Rivers Overlook | {-4,-57,0} | — | The view: all three rivers (Aldren, Brenn, Solt) join; the **temple island** across the channel, spires beyond. A `the temple` / `the rivers` noun (name the three + "Tri-Cities"; do NOT mention a fourth) |
| 6121 | South Quay Gate | {-5,-58,0} | — | South edge; a `the way south` noun framing a stub toward the Long Quay (NO south exit yet — intentional) |

Exit skeleton (finalize + cartcheck):
```
6106 n->6105(River Road) s->6107
6107 n->6106 s->6108 w->6117
6117 e->6107 s->6113
6108 n->6107 s->6119 w->6113 e->6109
6113 n->6117 e->6108 s->6114
6109 w->6108 s->6110
6119 n->6108 s->6120 w->6114 e->6110
6114 n->6113 e->6119 s->6115
6110 n->6109 w->6119 s->6111
6120 n->6119 s->6121 w->6115 e->6111
6115 n->6114 e->6120 s->6116
6111 n->6110 w->6120 s->6112 e->6118
6118 w->6111
6116 n->6115
6112 n->6111
6121 n->6120   (south = stub to The Long Quay, NO exit)
```

- [ ] **Step 1: Author 6106–6113** (gate + quayhead + warehouses + barge dock).
- [ ] **Step 2: Author 6119–6116** (market lane, chandlery, fish quay, crossing, cooper's yard, long wharf, tavern).
- [ ] **Step 3: Author 6118, 6112, 6121** — the Old Mole seed (load-bearing lore noun), the Overlook (the tri-city + temple-island reveal), the south stub.
- [ ] **Step 4: Cross-check** every exit's reciprocal; coords match the table; 6105↔6106 seam.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/
git commit -m "feat(confluence): The Landings rooms 6106-6121 (waterfront, barge dock, Old Mole, overlook)"
```

---

## Task 3: Quayfolk vendor goods (items 40127–40129)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40127-smoked_river_eel.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40128-confluence_trout.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40129-coil_of_tarred_line.yaml`

Model on `items/materials-40000/40125-smoked_river_fish.yaml`. All 40xxx items
live in `materials-40000/`. Block-scalar descriptions.

- [ ] **Step 1: Author the three items**

- **40127 Smoked River-Eel** — fish-trader good. `is_component: true`,
  `component_tag: river-eel`, `type: object`, `subtype: mundane`, value ~7,
  `vendor_categories: [cooking]`.
- **40128 Confluence Trout** — fish-trader good. `component_tag: confluence-trout`,
  value ~5, `vendor_categories: [cooking]`.
- **40129 Coil of Tarred Line** — chandler good (cordage). `component_tag:
  tarred-line`, value ~6, `vendor_categories: [tailoring]` (cordage = the
  tailoring discipline; confirm `tailoring` is in `ValidCraftSupports` —
  `internal/shops/shopinventory.go` — it is).

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/4012{7,8,9}-*.yaml
git commit -m "feat(confluence): Quayfolk vendor goods 40127-40129 (eel, trout, tarred line)"
```

---

## Task 4: Quayfolk NPCs + dialogue + vendors (9419–9426)

**Files:**
- Create: `mobs/the_confluence/9419-...yaml` … `9426-...yaml`
- Create: `dialogue/the_confluence/9419.yaml` … (for the speaking NPCs)

All 8: `zone: The Confluence`, `non_combatant: true`, `hostile: false`,
`charm_immune: true`, `speciesid: 1`, `level: 1`, `groups: [humanoid, quayfolk]`,
`maxwander: 0`, `statpool ~30`, `activitylevel ~10`. Non-vendors:
`behavior_archetype: noncombat_passive`. **Unique names** (no recycling; check
roster + novel cast Maren/Davan/Aldric/Brennan). Dialogue files named by mobid;
first-person `text`, player-perspective `hints`, discoverable triggers.

| Mob | Role | Room | Dialogue focus |
|-----|------|------|----------------|
| 9419 | Gate-Warden | 6106 | The opened gate (warden just back); welcome; directions to docks/market/overlook |
| 9420 | Barge Master | 6109 | Confluence↔NP passage **lore** (no working transit here); **Davan tie-back node** — recalls a young man from up the Amber Valley who took passage north a while back (carving tools, quiet, headed for the capital) |
| 9421 | Fish-Trader | 6110 | **Vendor** (`craft_support: cooking`, `behavior_archetype: noncombat_shopkeeper`, `gold: 40`); the catch, the market |
| 9422 | Dockmaster | 6108 | The working quay, customs, the Quayfolk; the pragmatic city voice |
| 9423 | Chandler | 6114 | **Vendor**; ship/dock goods |
| 9424 | Tavern-keeper | 6116 | Gossip, the city, the rivers; ambient color |
| 9425 | Stevedore | 6111 | Ambient dockhand (idlecommands; light/no tree, or a short one) |
| 9426 | Cooper / Net-mender | 6115 | Ambient dock craft (idlecommands; light/no tree) |

- [ ] **Step 1: Author the Fish-Trader (9421)** — vendor like Birrel:
```yaml
  shop:
    - itemid: 40127   # smoked river-eel
    - itemid: 40128   # confluence trout
    - itemid: 40125   # smoked river-fish (reuse River Road good)
    - itemid: 40123   # watercress
    - itemid: 40124   # freshwater mussels
```
- [ ] **Step 2: Author the Chandler (9423)** — vendor. A ship's chandler is a
  general store for ships, so use `craft_support: general` (like Mardle 9390) —
  the shop's category check (`buyrules.go`: `vendorAcceptsAny`) treats `general`
  as accepting every item category, so the mixed reused goods all stock cleanly
  (verified: 40102 rope = tailoring, 40103 waterskin = tailoring, 40105 tinderbox
  = blacksmithing; a `tailoring` vendor would REJECT the tinderbox, a `general`
  one does not):
```yaml
  craft_support: general
  behavior_archetype: noncombat_shopkeeper
  gold: 40
  ...
  shop:
    - itemid: 40129   # coil of tarred line (new)
    - itemid: 40102   # rope (reuse NP good)
    - itemid: 40103   # waterskin
    - itemid: 40105   # tinderbox
```
- [ ] **Step 3: Author 9419/9420/9422/9424** with dialogue trees (warden, barge master + Davan node, dockmaster, tavern-keeper).
- [ ] **Step 4: Author 9425/9426** as light ambient (idlecommands; minimal or no dialogue tree).
- [ ] **Step 5: Verify** mob filenames match `ConvertForFilename(name)`; all carry `quayfolk` in `groups`; placement rooms (Task 2) have matching `spawninfo`.
- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/ _datafiles/world/dogmud/dialogue/the_confluence/
git commit -m "feat(confluence): Quayfolk NPCs 9419-9426 + dialogue + fish/chandler vendors"
```

---

## Task 5: Smoke test, cartcheck, docs, merge

**Files:**
- Modify: `docs/ZONE_EXPANSION.md`

- [ ] **Step 1: Wipe instance saves** (NOT `shops/`)

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot**, watch for clean load

Run: `go build -o C:/tmp/confl.exe . && C:/tmp/confl.exe` (boot, then stop).
Expected: no panics; `rooms.loadAllRoomZones` zoneCount=**40**, loadedCount=**1038**;
`mobs.LoadDataFiles` count up by 8; **`ValidateZoneConsistency errors=0 warnings=0
mode=panic`**; faction loads without error. A coord collision, filename mismatch,
or missing zone-config panics here — fix and re-boot.

- [ ] **Step 3: `cartcheck the_confluence`** (in the running server, admin) — expect clean.

- [ ] **Step 4: Walk it** — River Road 6105 → south → 6106 → through the district;
  `list`/`buy` from the Fish-Trader and Chandler; `ask <barge-master> davan` (the
  tie-back); `look mark` at the Old Mole; `look temple` at the Overlook; confirm
  6121 south is a dead stub.

- [ ] **Step 5: Update `docs/ZONE_EXPANSION.md`** — mark Zone 5.4 The Confluence
  **building**, district 1 (The Landings) ✅ built, rooms 6106–6121; world now
  40 zones / 1038 rooms.

```bash
git add docs/ZONE_EXPANSION.md
git commit -m "docs(zones): mark The Confluence district 1 (The Landings) built"
```

- [ ] **Step 6: Merge `--no-ff`**

```bash
git checkout master
git merge --no-ff feature/confluence-landings -m "Merge: The Confluence district 1 — The Landings"
```

---

## Self-review checklist (run before merge)

- [ ] All 16 rooms 6106–6121 exist, `zone: The Confluence`, coords cartcheck-clean.
- [ ] `zone-config.yaml` present; `quayfolk.yaml` faction present (no `.rep` file committed).
- [ ] 6105 south exit added; gate prose now reads OPEN.
- [ ] No item uses `vendor_categories: general`; fish goods `cooking`, line `tailoring`.
- [ ] All 8 mobs carry `quayfolk` in `groups`; filenames match `ConvertForFilename`.
- [ ] Davan tie-back present in the Barge Master only; Old Mole mark the single seed.
- [ ] Boot clean (no panics), `cartcheck the_confluence` clean.
