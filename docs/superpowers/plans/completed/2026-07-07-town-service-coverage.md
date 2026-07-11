# Town Service Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a gold+storage bank and the missing baseline craft shops
(smith/tailor/alchemist/enchanter/jeweler/cook/general) to the proper towns, each
new shop being a vendor NPC + matching crafting station in a new dedicated room,
with light dialogue and a simple work/sleep schedule.

**Architecture:** Pure world-content (YAML). Each town is an independent task
producing rooms + mobs + dialogue + schedules in that town's zone folders, wired
via room `spawninfo`. No engine changes. Correctness is gated by the boot-time
validators, `cartcheck`, and `id_inventory.py`.

**Tech Stack:** GoMud YAML data files under `_datafiles/world/dogmud/`; validation
via `go run .` boot test, the `cartcheck` admin command, and
`python tools/id_inventory.py`.

---

## Reference spec
`docs/superpowers/specs/2026-07-07-town-service-coverage-design.md` — read for the
gap matrix rationale and scope boundaries.

## Global ID allocation (PINNED — do not deviate)

Rooms `6443–6466` (24), mobs `9588–9611` (24). One block per town:

| Town | Room IDs | Mob IDs | Zone folder | Zone display name |
|---|---|---|---|---|
| Stillwater | 6443 | 9588 | `stillwater` | `Stillwater` |
| Greenford | 6444–6449 | 9589–9594 | `greenford` | `Greenford` |
| The Confluence | 6450–6453 | 9595–9598 | `the_confluence` | `The Confluence` |
| Hartcharn | 6454–6459 | 9599–9604 | `hartcharn` | `Hartcharn` |
| Pothole Coulee | 6460–6463 | 9605–9608 | `pothole_coulee` | `Pothole Coulee` |
| New Plymouth (bank) | 6464 | 9609 | `new_plymouth_merchant` | `New Plymouth Merchant` |
| New Plymouth (shops) | 6465–6466 | 9610–9611 | `new_plymouth_crafting` | `New Plymouth Crafting` |

The `zone:` field inside every room/mob YAML MUST be the **display name** above;
the containing **folder** MUST be `ConvertForFilename(zone)` (already the folder
names above), or the loader panics at boot.

---

## Task 0: Canonical templates & conventions (read-only, no files created)

**Files:** none created. This task establishes the exact shapes every later task
clones. Exemplar files to open and mirror:
- Shop mob: `_datafiles/world/dogmud/mobs/thornwall_city/109-enchanter_vael.yaml`
- Bank room: `_datafiles/world/dogmud/rooms/stillwater/5100.yaml`
- Bank clerk mob: `_datafiles/world/dogmud/mobs/stillwater/356-counting_house_clerk.yaml`
- Shop room w/ station + spawninfo: `_datafiles/world/dogmud/rooms/stillwater/4106.yaml` (station `forge`, `spawninfo: [{mobid: 337, cooldown: 600 rounds}]`)
- Dialogue: `_datafiles/world/dogmud/dialogue/stillwater/341.yaml`
- Simple schedule: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_smith.yaml`

- [ ] **Step 1: Internalize the SHOP MOB shape.** Every new vendor mob file is
`mobs/<zone_folder>/<mobid>-<ConvertForFilename(name)>.yaml` and contains at
minimum:
```yaml
mobid: <ID>
zone: <Zone Display Name>
craft_support: <blacksmithing|tailoring|alchemy|enchanting|jewelcrafting|cooking|general>
hostile: false
non_combatant: true
schedule_id: <zone_trade>
character:
  name: <Pinned Name>
  description: >-
    <2-4 lines, 78-col wrapped, in-world flavor for the merchant.>
shop:
  - itemid: <id>   # clone the exemplar list for this discipline (table per task)
  - itemid: <id>
```
(Match the exact key nesting of the exemplar `109-enchanter_vael.yaml` — e.g.
`character:` block for name/description. Copy its structure verbatim, change
values.)

- [ ] **Step 2: Internalize the SHOP ROOM shape.** Every new shop room is
`rooms/<zone_folder>/<roomid>.yaml`:
```yaml
roomid: <ID>
zone: <Zone Display Name>
title: <Room Title>
description: >
  <3-6 lines, 78-col wrapped.>
biome: city
coord: {x: <hx±1>, y: <hy±1>, z: <hz>}   # hub coord + one unit in a FREE direction
station: <forge|loom|alchemy_bench|enchanting_circle|jeweler_bench|cooking_fire>  # omit for general/bank
exits:
  <dir>: {roomid: <hub roomid>}          # reciprocal to the hub's new exit
spawninfo:
- mobid: <shop mobid>
  cooldown: 600 rounds
```

- [ ] **Step 3: Internalize the BANK ROOM shape** (clone `5100.yaml`): same as a
room but with `isbank: true`, `isstorage: true`, `storagecapacity: 1000`, **no
`station:`**, and `spawninfo` pointing at the clerk mob.

- [ ] **Step 4: Internalize the DIALOGUE shape** (clone `341.yaml`):
`dialogue/<zone_folder>/<mobid>.yaml`:
```yaml
mobid: <ID>
zone: <Zone Display Name>
defaultMood: friendly
patterns:
  - keywords: ["hello", "hi", "greet"]
    responses:
      - "<first-person greeting naming the shop/trade>"
  - keywords: ["<trade>", "buy", "wares", "stock"]
    responses:
      - "<first-person line pointing at the wares>"
```
Voice SOP: NPC text is FIRST PERSON. No quests, no grants, no semicolons in text.

- [ ] **Step 5: Internalize the SCHEDULE shape** (clone `thornwall_smith.yaml`),
kept entirely within the shop room so target rooms are trivially reachable:
```yaml
id: <zone_trade>
description: "<Name>'s day: open by day, rest by night."
segments:
  - start: 6
    end: 22
    target_room: <shop roomid>
    activity: ""
    idlecommands:
      - emote <trade-flavor action>.
      - say <trade-flavor line>.
  - start: 22
    end: 6
    target_room: <shop roomid>
    activity: sleeping
    idlecommands:
      - emote settles onto a cot in the back, asleep.
```
Bank clerks use schedule id `<zone>_clerk`, same shape. Schedule ids MUST be
globally unique — the `<zone>_<trade>` scheme guarantees it (no town has two of
the same trade).

- [ ] **Step 6: Internalize the ROOM-PLACEMENT RECIPE (cartesian-safe).** For each
new room:
  1. Open the assigned **attach hub** room; read its `coord` and existing `exits`.
  2. Infer the zone's direction→delta convention from the hub and one neighbor
     (e.g. hub exit `north` → neighbor coord differs by +1 or −1 in y; match it).
  3. Pick a **free** compass direction on the hub (or, when a hub runs out of free
     exits, chain the new room off a previously-created new room in this task).
  4. Set the new room's `coord` = hub coord + one unit in that direction. **Verify
     no existing room in the zone already occupies that coord** (grep the zone for
     `coord: {x: <val>, y: <val>`).
  5. Add the exit on the hub (`<dir>: {roomid: <new>}`) AND the reciprocal on the
     new room (`<opposite dir>: {roomid: <hub>}`).
  6. After the town's rooms are written, run `cartcheck <zone>` (see Task 7) — it
     MUST report no collisions / non-reciprocal exits for the zone.

**Naming filename rule:** `ConvertForFilename(name)` = lowercase, keep a–z/0–9,
drop apostrophes, all else → underscore. Pinned filenames are given per task.

---

## Task 1: Stillwater — enchanter

**Files:**
- Create: `_datafiles/world/dogmud/rooms/stillwater/6443.yaml`
- Create: `_datafiles/world/dogmud/mobs/stillwater/9588-enchanter_rane.yaml`
- Create: `_datafiles/world/dogmud/dialogue/stillwater/9588.yaml`
- Create: `_datafiles/world/dogmud/schedules/stillwater/stillwater_enchanter.yaml`

Attach hub: **Coalsmoke Alley `4107`** (has a free exit). Clone shop list from mob
`109` (Enchanter Vael).

| mobid | name | craft_support | roomid | room title | station | schedule_id |
|---|---|---|---|---|---|---|
| 9588 | Enchanter Rane | enchanting | 6443 | The Chrysalis Workshop | enchanting_circle | stillwater_enchanter |

- [ ] **Step 1:** Create room `6443` per the shop-room shape (Task 0 Step 2), placed
off hub `4107` using the placement recipe (Task 0 Step 6), `station:
enchanting_circle`, `spawninfo` → mobid 9588.
- [ ] **Step 2:** Create mob `9588-enchanter_rane.yaml` (shop-mob shape), `shop:`
list cloned from `mobs/thornwall_city/109-enchanter_vael.yaml`,
`schedule_id: stillwater_enchanter`.
- [ ] **Step 3:** Create dialogue `dialogue/stillwater/9588.yaml` (light greeting).
- [ ] **Step 4:** Create schedule `schedules/stillwater/stillwater_enchanter.yaml`
(work/sleep in room 6443).
- [ ] **Step 5:** Add the reciprocal exit on hub `rooms/stillwater/4107.yaml`.
- [ ] **Step 6: Verify** — `python tools/id_inventory.py` shows no collision;
boot-test (Task 7 recipe) loads Stillwater with no panic.
- [ ] **Step 7: Commit** — `git add` the 4 new files + the edited hub;
`git commit -m "content(stillwater): add Enchanter Rane + Chrysalis Workshop"`.

---

## Task 2: Greenford — bank + smith, tailor, alchemist, enchanter, jeweler

**Files (create each; dialogue = `dialogue/greenford/<mobid>.yaml`, schedule =
`schedules/greenford/<schedule_id>.yaml`):**
- Rooms `rooms/greenford/6444..6449.yaml`
- Mobs `mobs/greenford/9589..9594-<name>.yaml`

Attach hubs: **Guild Lane `6294`** (coord `{x:23,y:-77,z:1}`, exits west→6289,
north→6291; free: east/south/up/down) and **Market Cross `6289`** (4-way; check
free dirs). Distribute the 6 new rooms across these two hubs; chain if needed.

| mobid | name | craft_support | roomid | room title | station | schedule_id | clone shop from |
|---|---|---|---|---|---|---|---|
| 9589 | Prewitt the Reeve (bank clerk) | — | 6444 | The Greenford Counting House | *(bank: isbank+isstorage, no station)* | greenford_clerk | *(clerk mob 356; no shop)* |
| 9590 | Aldo the Smith | blacksmithing | 6445 | The Guild Forge | forge | greenford_smith | mob 97 |
| 9591 | Wenna the Weaver | tailoring | 6446 | The Weaver's Loft | loom | greenford_tailor | mob 113 |
| 9592 | Hebb the Herbalist | alchemy | 6447 | The Herbalist's Stall | alchemy_bench | greenford_alchemist | mob 98 |
| 9593 | Cade the Enchanter | enchanting | 6448 | The Sigil Room | enchanting_circle | greenford_enchanter | mob 109 |
| 9594 | Tibb the Jeweler | jewelcrafting | 6449 | The Jeweler's Bench | jeweler_bench | greenford_jeweler | mob 108 |

Filenames: `9589-prewitt_the_reeve.yaml`, `9590-aldo_the_smith.yaml`,
`9591-wenna_the_weaver.yaml`, `9592-hebb_the_herbalist.yaml`,
`9593-cade_the_enchanter.yaml`, `9594-tibb_the_jeweler.yaml`.

- [ ] **Step 1:** Create bank room `6444` (bank-room shape, Task 0 Step 3),
`spawninfo` → 9589, placed off Guild Lane `6294`.
- [ ] **Step 2:** Create the 5 shop rooms `6445–6449` (shop-room shape), each with
its station, `spawninfo` → its mobid, placed per the placement recipe across the
hubs (verify coords non-overlapping; add reciprocal exits).
- [ ] **Step 3:** Create the 6 mobs `9589–9594` (clerk = clone `356`; shops = clone
the discipline exemplar `shop:` lists), each with its `schedule_id`.
- [ ] **Step 4:** Create 6 dialogue files `dialogue/greenford/9589..9594.yaml`
(Greenford voice; `<Name> the <Trade>` folks are warm/rural).
- [ ] **Step 5:** Create 6 schedules `schedules/greenford/greenford_{clerk,smith,
tailor,alchemist,enchanter,jeweler}.yaml`.
- [ ] **Step 6:** Add reciprocal exits on the hub room(s) touched (`6294` and/or
`6289`).
- [ ] **Step 7: Verify** — `cartcheck greenford` clean; `id_inventory.py` no
collision; boot loads Greenford clean.
- [ ] **Step 8: Commit** — `content(greenford): add Counting House + smith/tailor/
alchemist/enchanter/jeweler`.

---

## Task 3: The Confluence — bank + smith, enchanter, jeweler

**Files:** rooms `rooms/the_confluence/6450..6453.yaml`; mobs
`mobs/the_confluence/9595..9598-<name>.yaml`; dialogue
`dialogue/the_confluence/<mobid>.yaml`; schedules
`schedules/the_confluence/<id>.yaml`.

Attach hub: the craft row around **The Cooperage `6234`** (exits to 6233/6235) and
the 6234–6240 artisan block. Distribute/chain the 4 rooms there.

| mobid | name | craft_support | roomid | room title | station | schedule_id | clone shop from |
|---|---|---|---|---|---|---|---|
| 9595 | The Exchange-Keeper (bank clerk) | — | 6450 | The Confluence Exchange | *(bank)* | the_confluence_clerk | clerk 356 |
| 9596 | Roan the Smith | blacksmithing | 6451 | The Riverside Forge | forge | the_confluence_smith | mob 97 |
| 9597 | Merrow the Enchanter | enchanting | 6452 | The Warded Alcove | enchanting_circle | the_confluence_enchanter | mob 109 |
| 9598 | Bevan the Jeweler | jewelcrafting | 6453 | The Gemcutter's Nook | jeweler_bench | the_confluence_jeweler | mob 108 |

Filenames: `9595-the_exchange_keeper.yaml`, `9596-roan_the_smith.yaml`,
`9597-merrow_the_enchanter.yaml`, `9598-bevan_the_jeweler.yaml`.
(Confluence civic NPCs use `The <Trade>` bare form — the Exchange-Keeper fits.)

- [ ] **Step 1:** Bank room `6450` + `spawninfo` → 9595, off the artisan block.
- [ ] **Step 2:** Shop rooms `6451–6453` with stations + spawninfo, placed per
recipe.
- [ ] **Step 3:** Mobs `9595–9598` (clerk clone 356; shops clone exemplars).
- [ ] **Step 4:** Dialogue `9595–9598` (Confluence voice).
- [ ] **Step 5:** Schedules `the_confluence_{clerk,smith,enchanter,jeweler}`.
- [ ] **Step 6:** Reciprocal hub exits.
- [ ] **Step 7: Verify** — `cartcheck the_confluence` clean; id_inventory clean;
boot loads The Confluence clean.
- [ ] **Step 8: Commit** — `content(confluence): add Exchange + smith/enchanter/jeweler`.

---

## Task 4: Hartcharn — bank + tailor, alchemist, enchanter, jeweler, cook

**Files:** rooms `rooms/hartcharn/6454..6459.yaml`; mobs
`mobs/hartcharn/9599..9604-<name>.yaml`; dialogue `dialogue/hartcharn/<mobid>.yaml`;
schedules `schedules/hartcharn/<id>.yaml`.

Attach hubs: the town core `5402–5421` (e.g. off The Smithy `5410` and its
neighbors). Distribute/chain the 6 rooms.

| mobid | name | craft_support | roomid | room title | station | schedule_id | clone shop from |
|---|---|---|---|---|---|---|---|
| 9599 | Osric Vane (bank clerk) | — | 6454 | The Hartcharn Strongbox | *(bank)* | hartcharn_clerk | clerk 356 |
| 9600 | Bryd Harrow | tailoring | 6455 | The Weaving Shed | loom | hartcharn_tailor | mob 113 |
| 9601 | Alard Fen | alchemy | 6456 | The Apothecary's Hut | alchemy_bench | hartcharn_alchemist | mob 98 |
| 9602 | Sefa Crane | enchanting | 6457 | The Rune-Carver's Hut | enchanting_circle | hartcharn_enchanter | mob 109 |
| 9603 | Til Marsh | jewelcrafting | 6458 | The Gem Hut | jeweler_bench | hartcharn_jeweler | mob 108 |
| 9604 | Gerta Crook | cooking | 6459 | The Cookhouse | cooking_fire | hartcharn_cook | mob 248 |

Filenames: `9599-osric_vane.yaml`, `9600-bryd_harrow.yaml`, `9601-alard_fen.yaml`,
`9602-sefa_crane.yaml`, `9603-til_marsh.yaml`, `9604-gerta_crook.yaml`.
(Hartcharn convention: `Given Surname`, rustic, no trade epithet.)

- [ ] **Step 1:** Bank room `6454` + spawninfo → 9599.
- [ ] **Step 2:** Shop rooms `6455–6459` with stations + spawninfo, placed per recipe.
- [ ] **Step 3:** Mobs `9599–9604`.
- [ ] **Step 4:** Dialogue `9599–9604` (Hartcharn rustic voice).
- [ ] **Step 5:** Schedules `hartcharn_{clerk,tailor,alchemist,enchanter,jeweler,cook}`.
- [ ] **Step 6:** Reciprocal hub exits.
- [ ] **Step 7: Verify** — `cartcheck hartcharn` clean; id_inventory clean; boot clean.
- [ ] **Step 8: Commit** — `content(hartcharn): add Strongbox + tailor/alchemist/enchanter/jeweler/cook`.

---

## Task 5: Pothole Coulee — storage upgrade + tailor, enchanter, jeweler, cook

**Files:**
- **Modify:** `_datafiles/world/dogmud/rooms/pothole_coulee/5208.yaml` — add
`isstorage: true` and `storagecapacity: 1000` to the existing bank room (no new
clerk).
- Create rooms `rooms/pothole_coulee/6460..6463.yaml`; mobs
`mobs/pothole_coulee/9605..9608-<name>.yaml`; dialogue
`dialogue/pothole_coulee/<mobid>.yaml`; schedules `schedules/pothole_coulee/<id>.yaml`.

Attach hub: the trader/bank cluster around **Coulee Provisions `5207`** and the
town core. Distribute/chain the 4 rooms.

| mobid | name | craft_support | roomid | room title | station | schedule_id | clone shop from |
|---|---|---|---|---|---|---|---|
| 9605 | Stitcher Bex | tailoring | 6460 | The Patch-Tent | loom | pothole_coulee_tailor | mob 113 |
| 9606 | Enchanter Skell | enchanting | 6461 | The Warder's Dugout | enchanting_circle | pothole_coulee_enchanter | mob 109 |
| 9607 | Jeweler Vurl | jewelcrafting | 6462 | The Gem Stall | jeweler_bench | pothole_coulee_jeweler | mob 108 |
| 9608 | Cook Hesk | cooking | 6463 | The Cook-Fire | cooking_fire | pothole_coulee_cook | mob 248 |

Filenames: `9605-stitcher_bex.yaml`, `9606-enchanter_skell.yaml`,
`9607-jeweler_vurl.yaml`, `9608-cook_hesk.yaml`.
(Pothole convention: `Trade Name`, harsh short scablands names.)

- [ ] **Step 1:** Edit `rooms/pothole_coulee/5208.yaml` — add `isstorage: true`
and `storagecapacity: 1000` (leave `isbank: true` as-is).
- [ ] **Step 2:** Create shop rooms `6460–6463` with stations + spawninfo, placed
per recipe off the `5207` cluster.
- [ ] **Step 3:** Mobs `9605–9608` (clone exemplars).
- [ ] **Step 4:** Dialogue `9605–9608` (harsh scablands voice).
- [ ] **Step 5:** Schedules `pothole_coulee_{tailor,enchanter,jeweler,cook}`.
- [ ] **Step 6:** Reciprocal hub exits.
- [ ] **Step 7: Verify** — `cartcheck pothole_coulee` clean; id_inventory clean;
boot clean; confirm `5208` now has storage (grep `isstorage` in the file).
- [ ] **Step 8: Commit** — `content(pothole): storage at the bank + tailor/enchanter/jeweler/cook`.

---

## Task 6: New Plymouth — central bank + enchanter, jeweler

**Files:**
- Bank (Merchant district): room `rooms/new_plymouth_merchant/6464.yaml`; mob
`mobs/new_plymouth_merchant/9609-master_coyne.yaml`; dialogue
`dialogue/new_plymouth_merchant/9609.yaml`; schedule
`schedules/new_plymouth_merchant/new_plymouth_merchant_clerk.yaml`.
- Shops (Crafting district): rooms `rooms/new_plymouth_crafting/6465..6466.yaml`;
mobs `mobs/new_plymouth_crafting/9610..9611-<name>.yaml`; dialogue
`dialogue/new_plymouth_crafting/<mobid>.yaml`; schedules
`schedules/new_plymouth_crafting/<id>.yaml`.

Attach hubs: bank off **Falk's Auction House `5808`** (Merchant); shops off **The
Tannery `5717`** / the crafting row (Crafting).

| mobid | name | craft_support | roomid | room title | station | schedule_id | zone | clone shop from |
|---|---|---|---|---|---|---|---|---|
| 9609 | Master Coyne (bank clerk) | — | 6464 | The Plymouth Exchange | *(bank)* | new_plymouth_merchant_clerk | New Plymouth Merchant | clerk 356 |
| 9610 | Ansel Rune | enchanting | 6465 | The Enchanter's Atelier | enchanting_circle | new_plymouth_crafting_enchanter | New Plymouth Crafting | mob 109 |
| 9611 | Perla Gilt | jewelcrafting | 6466 | The Jeweler's Atelier | jeweler_bench | new_plymouth_crafting_jeweler | New Plymouth Crafting | mob 108 |

Filenames: `9609-master_coyne.yaml`, `9610-ansel_rune.yaml`, `9611-perla_gilt.yaml`.
(NP Merchant = honorific `Master/Dame`; NP Crafting = name + material surname like
`Edda Glass` → `Ansel Rune`, `Perla Gilt`.)

- [ ] **Step 1:** Bank room `6464` in `new_plymouth_merchant/` (`zone: New Plymouth
Merchant`), spawninfo → 9609, off `5808`.
- [ ] **Step 2:** Shop rooms `6465`, `6466` in `new_plymouth_crafting/` (`zone: New
Plymouth Crafting`), stations + spawninfo, off `5717`/crafting row.
- [ ] **Step 3:** Mobs 9609 (clerk, Merchant folder), 9610–9611 (shops, Crafting
folder) — each with its zone-matching `zone:` field.
- [ ] **Step 4:** Dialogue 9609 (merchant folder), 9610–9611 (crafting folder).
- [ ] **Step 5:** Schedules in the matching zone folders.
- [ ] **Step 6:** Reciprocal hub exits on `5808` and `5717` (or chained new room).
- [ ] **Step 7: Verify** — `cartcheck new_plymouth_merchant` and `cartcheck
new_plymouth_crafting` clean; id_inventory clean; boot clean.
- [ ] **Step 8: Commit** — `content(new_plymouth): Plymouth Exchange + enchanter/jeweler`.

---

## Task 7: Integration verification (whole build)

**Files:** none (verification only).

- [ ] **Step 1: ID collision sweep.** Run `python tools/id_inventory.py` — confirm
no duplicate room/mob IDs reported and that 6443–6466 / 9588–9611 are now owned.
- [ ] **Step 2: Cartesian sweep.** Nuke instance saves, then in-game (or via a
one-off boot) run `cartcheck` for every touched zone: `stillwater`, `greenford`,
`the_confluence`, `hartcharn`, `pothole_coulee`, `new_plymouth_merchant`,
`new_plymouth_crafting`. All must report clean (no collisions, all exits
reciprocal).
- [ ] **Step 3: Boot test (pre-push SOP).** From repo root:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -iE "panic|LoadDataFiles|schedule|ValidateZone|error" | head -40
```
Expected: `rooms.LoadDataFiles`/`mobs.LoadDataFiles`/schedule validators all report
loaded counts with **no panic** and no consistency warnings for the touched zones.
- [ ] **Step 4: Live smoke** (harness or manual) at a sample of new sites:
  - New enchanter (e.g. Greenford Cade / room 6448): `list`, `buy <a mat>`, `sell`.
  - A new bank (e.g. Greenford Counting House 6444): `deposit 10`, `withdraw 10`,
    `storage add <item>`, `unstore 1`.
  - `craft` at a shop room's station (e.g. a jeweler recipe at 6449) resolves the
    station correctly.
  - `talk`/greet a new NPC → light dialogue fires.
- [ ] **Step 5: Final dispatch review + finish the branch** — use
`superpowers:requesting-code-review` on the whole branch, then
`superpowers:finishing-a-development-branch` (merge `--no-ff` to master; do NOT
push — prod push is a separate EOD step).

---

## Notes for the implementer
- **Instance saves shadow template edits.** Always `rm -rf mobs.instances/* rooms.instances/*` before any boot/cartcheck test, or new spawninfo/exits won't take effect.
- **Do NOT reuse names** Maret, Edda, Voss (existing dups) — all pinned names above are already checked clean against the 77-name world list.
- **YAML gotchas:** no colons inside noun/dialogue text (use `>-`/em-dash); no semicolons in NPC `text:`; `biome: city` for town rooms (not `wilderness`).
- **Filename must match** `<id>-<ConvertForFilename(name)>.yaml` for mobs / `<id>.yaml` for rooms / `<mobid>.yaml` for dialogue, or the loader panics.
- Shop item-ids come straight from the exemplar clone — every id already exists; do not invent items.
