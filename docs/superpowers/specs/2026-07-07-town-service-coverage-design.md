# Town Service Coverage — banks + full baseline craft shops

**Date:** 2026-07-07
**Status:** Design approved (brainstorm), pending plan.
**Scope:** Give every *proper town* a gold bank + item storage and a full baseline
set of craft shops {smith, tailor, alchemist, enchanter, jeweler, cook, general},
adding only what each town lacks. Each new shop is a vendor NPC in a new dedicated
room that also carries the discipline's crafting station, so players can buy
materials and craft on the spot. Fixes the enchanting (1 world-wide) and
jewelcrafting (2) starvation and the total absence of banking outside Stillwater
/ Thornwall / Pothole.

## Motivation

The economy dashboard shows starved disciplines — 1 enchanting shop, 2
jewelcrafting, vs. 7 smiths / 7 tailors / many cooks — and banking exists in only
3 rooms world-wide (item storage in only 2). Players in New Plymouth, the
Confluence, Greenford, and Hartcharn have no way to bank gold or store items, and
enchanting/jewelcrafting materials are effectively unbuyable across most of the
map. This is a world-content pass: rooms + mobs + shop inventories, no engine
changes.

## Current state (verified 2026-07-07)

**Banks (`isbank`): 3** — `stillwater/5100`, `thornwall_city/510`,
`pothole_coulee/5208`. **Storage (`isstorage`): 2** — `stillwater/5100`,
`thornwall_city/510` (Pothole's bank is gold-only).

**Craft-shop counts by `craft_support:` tag:** general 35 · cooking 20 ·
tailoring 7 · blacksmithing 7 · alchemy 6 · **jewelcrafting 2 · enchanting 1**.

**Mechanics confirmed:**
- **Shop** = mob with `shop:` item list + `craft_support:` (one of
  `blacksmithing / alchemy / tailoring / cooking / jewelcrafting / enchanting /
  general`), `hostile: false`, `non_combatant: true`. Pricing/restock is the
  living-economy system driven off these tags — no per-shop tuning needed.
- **Station** = room field `station: <key>`. Craft-shop rooms conventionally
  carry the matching station (verified in Thornwall: enchanter room 483 →
  `enchanting_circle`, jeweler 482 → `jeweler_bench`, smith 470 → `forge`,
  alchemist 471 → `alchemy_bench`). Station keys: `forge`, `loom`,
  `alchemy_bench`, `enchanting_circle`, `jeweler_bench`, `cooking_fire`.
- **Bank** = room with `isbank: true` + `isstorage: true` + `storagecapacity:
  1000` (the Counting House `stillwater/5100` pattern). Banking/storage commands
  gate on the room flags; a clerk NPC is flavor only.
- Room placement is validated by `ValidateZoneConsistency` at boot
  (`GamePlay.MapConsistencyEnforce`) and the `cartcheck` admin command:
  new rooms must have non-overlapping coords + reciprocal exits.
- IDs are **global, not per-zone**; a room can carry any global-unique ID while
  living in its zone folder. `python tools/id_inventory.py` reports next-free
  (rooms `6443`, mobs `9588` as of this survey) and verified-empty bands.

## Scope — the gap matrix

Baseline = {smith · tailor · alchemist · enchanter · jeweler · cook · general} +
bank/storage. Add **only what is missing**:

| Town | Bank/storage to add | Craft shops to add |
|---|---|---|
| Stillwater | — (has both) | enchanter |
| Thornwall | — | — (already complete) |
| Greenford | **bank+storage** | smith, tailor, alchemist, enchanter, jeweler |
| The Confluence | **bank+storage** | smith, enchanter, jeweler |
| Hartcharn | **bank+storage** | tailor, alchemist, enchanter, jeweler, cook |
| Pothole Coulee | **storage** (upgrade existing bank room 5208) | tailor, enchanter, jeweler, cook |
| New Plymouth (city) | **bank+storage** (central, Merchant district) | enchanter, jeweler (into Crafting district) |

Totals: **~20 new shop mobs + rooms**, **4 new bank rooms + clerks**, **1 storage
upgrade** → **~24 new rooms, ~24 new mobs**, each with a light dialogue file and a
simple schedule.

**Interpretation calls (locked with the user):**
- **New Plymouth is one city, not per-district.** Its districts already cover
  smith/tailor/alchemist/cook/general between them; the city only lacks enchanter
  + jeweler (→ Crafting district) and a bank (→ Merchant district). NOT a 7-shop
  set per district.
- Enchanter + jeweler are **spread to every proper town** — this is the fix for
  the 1-enchanter / 2-jeweler starvation.

## Design

### Bank building block
A new bank is one room:
```yaml
isbank: true
isstorage: true
storagecapacity: 1000
```
themed to the town (e.g. "The Greenford Counting House", "Confluence Exchange"),
hung off the town's commercial hub, plus a `non_combatant: true` clerk NPC
(flavor, light greeting dialogue). Pothole Coulee's existing bank room `5208`
just gains `isstorage: true` + `storagecapacity: 1000` (no new room).

### Craft-shop building block
A new craft shop is:
- **A vendor mob**: `craft_support: <discipline>`, `hostile: false`,
  `non_combatant: true`, a `shop:` item list cloned from the discipline exemplar
  (below), `schedule_id:` → its simple schedule, and light greeting dialogue.
- **A new room** themed to the trade, hung off a hub with a spare exit, carrying
  the matching `station:` (smith→`forge`, tailor→`loom`, alchemist→`alchemy_bench`,
  enchanter→`enchanting_circle`, jeweler→`jeweler_bench`, cook→`cooking_fire`;
  general → no station).

### Inventories — clone the discipline exemplar
Each new shop clones its exemplar `shop:` list, with 1–2 regional-flavor item
swaps allowed. Exemplars (verified):

| Discipline | Exemplar mob | Representative stock |
|---|---|---|
| enchanter (`enchanting`) | Thornwall Enchanter Vael (109) | chrysalis core/shard, binding paste, mutation catalyst, chrysalis setting, hive fragment, black pearl |
| jeweler (`jewelcrafting`) | Thornwall Jeweler Tess (108) | copper/silver/gold wire, polished stone, raw gem, gem dust, black pearl, chain link |
| smith (`blacksmithing`) | Thornwall Blacksmith Kerra (97) | steel/iron ingot, lake-iron nodule, pine pitch, wooden plank, chain link, coal dust |
| tailor (`tailoring`) | Thornwall Weaver Maren (113) | thread spool, bone needle, beeswax, cloth |
| alchemist (`alchemy`) | Thornwall Apothecary Voss (98) | herbs (willow bark, lake mint, oak bark, shadowcap, blood-moss, healer's root, thistle, dustwalk herb), glass vial, clay flask |
| cook (`cooking`) | Thornwall Tavern Cook Brynn (248) | raw meat, wild hare, freshwater clam, wild vegetables, water flask, salt pouch |
| general (`general`) | Stillwater Storekeeper Wulf (341) | oil lantern, torch, wild vegetables, water/salt, wooden plank, lake-iron nodule |

### Placement & cartesian consistency
Each new room attaches to an existing hub via one spare exit direction, with fresh
non-overlapping `mapx/mapy` coords and a reciprocal exit back. Attach hubs per
town are in the appendix. After the build, `cartcheck <zone>` must be clean for
every touched zone and the boot validator must not warn.

### IDs & collision strategy
Per-town contiguous ID blocks, pulled from the verified-free near-district bands
where they exist (`5725–5799` above NP Crafting, `5825–5899` above NP Merchant,
`5482–5499`, `5530–5599`, `5625–5699`, `5925–5999`, `511–999` above Thornwall) or
the global tail (rooms `6443+`, mobs `9588+`). The plan pins exact IDs per room/
mob via `id_inventory.py --alloc`; a final `id_inventory.py` pass detects any
collision. Build town-by-town (sequential) so blocks never overlap.

### Naming
Every new merchant/clerk name follows its town's convention and is checked against
the existing 77-name list. Conventions: Stillwater/Thornwall `[Trade-title]
[Name]`; Greenford `[Name] the [Trade]`; Confluence `[Name] the [Trade]` (named)
or `The [Trade]` (civic); Hartcharn `[Given] [Surname]`; NP by district (Crafting
= name+trade/material-surname, Merchant = honorific, e.g. Dame/Madam); Pothole
`[Trade] [Name]`, harsh short names. **Avoid** compounding existing dup names
Maret, Edda, Voss.

### Dialogue (light)
Each NPC gets one small dialogue file: a first-person greeting + 2–3 keyword
responses (trade/town flavor), following the NPC-voice SOP (NPC text 1st person,
hints are narrator). No quests, no grants.

### Schedules (simple)
Each NPC gets a minimal 2-segment schedule covering all 24h **within its own shop
room** — work by day, sleep at night — so target rooms are trivially reachable and
no extra rooms are needed:
```yaml
segments:
  - {start: 6,  end: 22, target_room: <shop room>, activity: craft, idlecommands: [...]}
  - {start: 22, end: 6,  target_room: <shop room>, activity: sleeping, idlecommands: [...]}
```
(Bank clerks use `activity: ""` by day instead of `craft`.) Wandering/tavern
routines are explicitly deferred — a later schedule-polish pass can enrich them.

## Per-town appendix (attach hubs + build list)

- **Stillwater** — +enchanter. Attach off Coalsmoke Alley `4107` (spare exit) or
  the temple lane near `4126`. Naming: `Enchanter <Name>`.
- **Greenford** — +bank, +{smith, tailor, alchemist, enchanter, jeweler}. Attach
  off **Guild Lane `6294`** (themed for crafts) and Market Cross `6289`. Naming:
  `<Name> the <Trade>`.
- **The Confluence** — +bank, +{smith, enchanter, jeweler}. Attach off the
  craft row near `6234–6240` (Cooper/Weaver/Potter block). Naming: `<Name> the
  <Trade>` or civic `The <Trade>`.
- **Hartcharn** — +bank, +{tailor, alchemist, enchanter, jeweler, cook}. Attach
  off the town core `5402–5421`. Naming: `<Given> <Surname>`, rustic.
- **Pothole Coulee** — upgrade `5208` with storage; +{tailor, enchanter, jeweler,
  cook}. Attach off the trader/bank cluster `5207–5208` and town core. Naming:
  `<Trade> <Name>`, harsh.
- **New Plymouth** — +bank in Merchant district (attach off the auction/market
  around `5808–5814`, band `5825–5899`); +enchanter, +jeweler in Crafting
  district (attach off `5709–5717`, band `5725–5799`). Naming per district.

## Out of scope
- No engine changes (shop/bank/station/schedule systems all exist).
- No new items — shops sell existing item IDs; if a discipline exemplar references
  an item, it already exists.
- No quests, no trainers (progression is use-based; no XP trainers exist).
- No per-district full-baseline for New Plymouth; no banks in the tiny 1-shop
  roadside stops (Kilnreach/Greywater/Watchers/Amber Valley/Ashwick).
- No wandering schedules (simple work/sleep only this pass).

## Testing / validation
- **Boot test** (pre-push SOP): nuke instance saves, `go run .` (or the smoke
  exe), confirm `rooms.LoadDataFiles`, `mobs.LoadDataFiles`, schedule validators,
  and `ValidateZoneConsistency` all pass with no panics/warnings for touched zones.
- **`cartcheck <zone>`** clean for every touched zone.
- **`id_inventory.py`** final pass: no collisions.
- **Buy/sell smoke** at 2–3 new shops (e.g. a new enchanter + a new bank):
  `list`, `buy`, `sell`, `deposit`/`withdraw`, `storage add/`unstore`, and
  `craft` at the station in a shop room.

## Open items for the plan
- Exact room IDs, coords, and exit wiring per new room (pin via `id_inventory
  --alloc`, one block per town).
- Final chosen names per NPC (draft in the plan, checked against the 77-name list).
- Exact per-shop inventory (exemplar clone + the 1–2 regional swaps).
- Which existing hub exit direction each new room consumes (must stay
  cartesian-consistent).
