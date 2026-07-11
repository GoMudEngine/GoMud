# New Plymouth — Docks District (Design Spec)

*Date: 2026-06-20 · District 1 of 7 (built first). Carries the supply-engine
wiring + the Bloom Trail opening.*

> **Parent designs:** master plan `docs/superpowers/specs/2026-06-20-new-plymouth-design.md`;
> city-wide layer `docs/superpowers/specs/2026-06-20-new-plymouth-citywide-design.md`
> (pull anchors §6, civic §4, supply §5, geometry §2); canon
> `docs/new_plymouth_canon.md`. Pricing-fix wiring recipe:
> `docs/superpowers/specs/2026-06-20-np-supply-pricing-fix-design.md` §2.
>
> **Scope:** ~30 rooms, the 7 Docks anchors (full life-sheets), the ambient/
> transient dock crowd, the supply-engine wiring, two questlines (Dock Rat +
> Bloom Trail *opening*), expansion stubs. **Staged A→B→C with a boot-smoke
> after each.** Bloom = breadcrumbs only (no item/mechanic — deferred per
> [[project-bloom-mechanic]]).

---

## 0. Decisions locked

- Build **staged A (waterfront) → B (dock streets) → C (outer edge/shadows)**,
  boot-smoke between stages.
- **Bloom Trail = narrative breadcrumbs only** (Marn's back room, an addled
  mutterer, a constable hint pointing to the Old Quarter). No Bloom item, no
  mechanic.
- Supply wiring = **Option A** (Dunmar a Dock-Master merchant; runner drains
  overstock and delivers).

---

## 1. The living rhythm (what the rooms must host)

Dawn: ships/barges land goods at the wharves → hauled to **Warehouse Row** →
**Dobb** loads the caravan at the **Depot** and sets out east on the Long Market.
Workers hired off **Jesset's** board haul cargo, break for a bowl at the
**cookshop**, drink at the **Salt Cellar** at night; the poorest bunk in the
**flophouse**, the rooted live in the dock streets. **Greer** keeps the manifests
just legal. Beneath the pilings and in **Marn's** back room, the Bloom moves.
Every room exists to host that day.

---

## 2. Geometry & room list

**Region:** x −42…−22, y 82…102, z 0 (+ underdock transitions). **Entry:**
outskirts **River Road 5480** (−17,78) gets a new **north** exit → a thin
waterfront road threading **west** along y≈79–81 (south of the Common floor) to
the Docks SE corner. **Exits out:** NE **bridge to Crafting** (→ `new_plymouth_crafting`,
the Long Market continues); **Underdock Stair** down → `new_plymouth_oldquarter`
(z−1). Rooms **5500–5529**.

> The coords AND the exit directions in the tables below are an **indicative
> layout guide** — exact coords and reciprocal exits are assigned and
> **`cartcheck`-validated during the build** (where any A↔B direction conflicts
> are resolved), and `docs/coordinate_map.md` updated. The Long Market is the
> E–W spine through the district (waterfront → Dock Street → Crafting bridge).

### Stage A — The Waterfront (5500–5509)
| Room | Name | Role / key nouns | Occupants | Exits |
|------|------|------------------|-----------|-------|
| 5500 | River Road Landing | entry from 5480; the road bends west; *milepost*, *tide-line* | — | s→5480(outskirts), w→5501, n→5510(Dock St) |
| 5501 | The Lower Wharf | barge landing (Davan arrived here); *gangplank*, *mooring rings* | ambient stevedores | e→5500, w→5502, n→5503 |
| 5502 | The North Quay | the quiet dock end (atmospheric); *coiled rope*, *lone fisherman's bucket* | a lone fisherman (ambient) | e→5501; **stub:** chained shipyard (n, blocked) |
| 5503 | The Long Quay | Jesset's wharves; *hiring board* (noun→jobs), *cargo cranes* | **Jesset** (day), ambient haulers | s→5501, w→5504, e→5505 |
| 5504 | The Fish Market | brine + smoke smell; *dried-fish racks*, *fishmonger's slab* | a fishmonger (ambient) | e→5503 |
| 5505 | Warehouse Row | import storehouses; *stacked crates*, *manifest chalkboard* | warehouse hands (ambient) | w→5503, e→5506, s→5507 |
| 5506 | The Caravan Depot | the supply origin; *parked wagon*, *loading dock* | **Dunmar Wells**, **Dobb** (pre-circuit) | w→5505, e→5510 |
| 5507 | The Harbormaster's Office | manifests/licensing; *ledger*, *bloodline seal* | **Harbormaster Greer** | n→5505 |
| 5508 | The Chandlery | ship-supply vendor; *tar barrels*, *spare canvas* | a chandler (minor vendor) | (off Warehouse Row) e→5505 |
| 5509 | The Underdock Stair | pilings, dark, damp; *slick steps*, *waterline* | — | up→5501; **down→Old Quarter** (z−1, stub-gated until that build) |

### Stage B — The Dock Streets (5510–5519)
| Room | Name | Role / key nouns | Occupants | Exits |
|------|------|------------------|-----------|-------|
| 5510 | Dock Street | the Long Market continues E; *gutter*, *ballad-seller's post* | crowd (ambient) | w→5506, n→5500, e→5511, s→5515 |
| 5511 | The Salt Cellar | inn/tavern exterior+door; *weathered sign*, *lantern* | — | w→5510, in→5512, e→5514 |
| 5512 | Salt Cellar Taproom | dockfolk evenings; *long bar*, *hearth* | **Bressa Toll**, drinkers (ambient) | out→5511, up→5513 |
| 5513 | Salt Cellar Rooms | transient lodging | transients | down→5512 |
| 5514 | Aldric's Public House | rougher lodging, horse yard; *trough*, *plain table* | a scarred-lip landlord (minor) | w→5511 |
| 5515 | Cutter's Lane | the alley; *dead-end refuse*, *loose grate* (→underdock) | a lurker (ambient) | n→5510, e→5516, down→5520(underdocks) |
| 5516 | The Cookshop | dock midday meals; *cookpot*, *bread tray* | a cookshop keeper (minor) | w→5515, e→5517 |
| 5517 | Jesset's Hiring Office | Jesset's home above; *job ledger*, *strongbox* | **Jesset** (off-hours) | w→5516, e→5518 |
| 5518 | Old Sable's Pawnshop | gossip/fence node; *cluttered shelves*, *brass scales* | **Old Sable** | w→5517, e→5519 |
| 5519 | Dockworker Tenements | where dockfolk sleep; *washing lines*, *stoop* | ambient dock families | w→5518; **stub:** sealed warehouse (e, machinery sounds) |

### Stage C — The Outer Edge & Shadows (5520–5529)
| Room | Name | Role / key nouns | Occupants | Exits |
|------|------|------------------|-----------|-------|
| 5520 | The Underdocks | below street, smuggler level; *low pilings*, *bilge stink* | thugs (ambient, human) | up→5515, e→5521, w→5509 |
| 5521 | The Pilings Haunt | a Bloom-addled corner (breadcrumb); *scratched wall*, *spent wrapper* | a Bloom-addled wanderer (mutters of "the low room") | w→5520, up→5522 |
| 5522 | The Outer Dock Lane | toward the Crafting edge; *handcart ruts* | crowd (ambient) | down→5521, e→5523, n→5525 |
| 5523 | Marn's Fabric-Remnants Shop | Bloom front (legit half); *remnant bolts*, *cupboard door* (→back) | **Marn the Draper** | w→5522, in→5524, **ne→Crafting bridge** |
| 5524 | Marn's Back Room | overhear breadcrumb (thick wall); *crates*, *muffled voices* | (timed overheard line) | out→5523 |
| 5525 | The Dock Constable's Post | Dock Rat hub; *notice board* (accusation), *cell* | **Dock Constable** (quest) | s→5522, e→5526 |
| 5526 | The Foreman's Office | Dock Rat: the real thief; *skewed ledger*, *hidden lockbox* | **Foreman Hewd** (quest, the skimmer) | w→5525 |
| 5527 | The Tar Yard | texture/atmosphere; *tar vats*, *drying nets* | a tar-boiler (ambient) | (off Outer Dock Lane) w→5522 |
| 5528 | The Customs Shed | Dock Rat: the "stolen" cargo (inconsistencies noun) | a customs clerk (minor) | (off Warehouse Row area) link→5505/5525 |
| 5529 | The Wharf Steps | quiet spot, moons over the water (atmosphere) | — | (off North Quay) link→5502 |

---

## 3. The 7 anchors (full life-sheets to author)

Each: home room · workplace · 24h `schedule_id` outline · `activity` gates ·
faction · relationships (1.6) · supply link · mutation (unique) · ≥3 dialogue
topics. Mobs **9300–9306**.

1. **Jesset (9300)** *(canon)* — mutation: extra knuckle-joint per finger.
   Home 5517 (above the office) · work 5503 (Long Quay, `activity: work`) · sched:
   sleep 5517 → day at the quay hiring/overseeing → evening Salt Cellar (5512) →
   home. Faction `np_dockfolk`. Rel: employs ambient haulers; friend of Bressa.
   Topics: dock work / the hiring board, the bloodline's harbor licensing
   (resentment), "the group that left" (heard rumors). Quest: gives **Dock Rat**.
2. **Bressa Toll (9301)** — amber laugh-flush. Home 5513 · work 5512
   (`activity: work` tending bar). Sched: tavern most of day/evening → late sleep.
   `np_dockfolk`. Rel: buys ale from **Renn Bowl** (Common — cross-district
   supply); friend of Jesset. Topics: gossip/rumors, the docks at night, who's
   behind on their tab.
3. **Harbormaster Greer (9302)** — salt-crystal knuckles. Home (a back room of
   5507) · work 5507. `np_dockfolk` but bloodline-licensed (tension). Rel: oversees
   Jesset; deals with Dunmar. Topics: the manifests/licensing, what ships brought
   in, the bloodline's cut (guarded).
4. **Dunmar Wells (9303)** — a third eye that never closes. Home (above 5505) ·
   work 5505/5506 (`activity: work`). **Supply hub** — the Dock-Master merchant
   (§4 wiring). Rel: dispatches Dobb; supplies every district crafter. Topics:
   imports/stock, the caravan schedule, "everything comes through here."
5. **Dobb (9304)** — tireless legs. Home (depot bunk 5506) · **runs the
   `np_docks_runner_circuit`** by day (the moving thread; leaves the district).
   `np_dockfolk`. Rel: works for Dunmar; greeted by every vendor on his route.
   Topics: the route, what each quarter's short on, road gossip.
6. **Marn the Draper (9305)** — unnaturally clean, scentless. Home (above 5523) ·
   work 5523 (front) + 5524 (back). **`bloom_trade`** (hidden). Rel: supplied by
   **Deren** (Old Quarter). Topics: fabric remnants (cover), deflection on
   anything past the cupboard door, nervous about "new competition." **Bloom
   breadcrumb:** timed overheard line in 5524.
7. **Old Sable (9306)** — silver coin-scales on her palms. Home (above 5518) ·
   work 5518. `np_dockfolk`. Rel: fences for the underdocks; the gossip node.
   Topics: rumors (sells info), the pawn trade, what's been "found" lately
   (Dock Rat + Bloom hints).

**Ambient/transient (pooled schedules, ~6–8 mobs 9307–9312):** stevedores/haulers
(work the wharves by day, Salt Cellar/cookshop, sleep tenements), a fishmonger, a
chandler, a lone fisherman (North Quay), transient sailors (cycle through the
flophouse/Salt Cellar rooms), underdock thugs/lurkers (predator/thief archetypes,
gated). These fill the streets per cluster templates.

---

## 4. Supply-engine wiring (the Docks build owns it)

Per the pricing-fix spec §2, Option A. Engine code (~6–10 lines) + config + YAML:
1. **Config:** add `"New Plymouth Docks"` (and the other NP zone display names as
   they're built) to `Balance.CaravanServedZones` in `_datafiles/config.yaml`.
2. **Register circuit:** `internal/caravan/runner_completion_listener.go` —
   add `"np_docks_runner_circuit": {}` to `runnerCircuitPatrols`.
3. **Buckets:** `internal/caravan/arrival_listener.go` `bucketsForRunnerPatrol` —
   add a case returning NP delivery buckets (`[]string{"base", "overlap", "np_imported"}`),
   pickup `[]string{}`; add NP imported items to `internal/economy/buckets.go`.
4. **Dunmar = Dock-Master merchant:** mob 9303 in the Docks zone, **not** in
   `CaravanServedZones` exemption logic for *his own* restock — he self-restocks
   imported goods via the tier ticker; the runner's pickup pass drains his
   overstock.
5. **Runner circuit patrol YAML:** `_datafiles/world/dogmud/patrols/new_plymouth_docks/np_docks_runner_circuit.yaml`
   — `loop_shape: oneshot`, depot (5506) start, `caravan_vendor` waypoints at
   each district's vendor rooms (added as districts land; Docks-internal vendors
   first), return to depot.
6. **Vendor StockEntry declarations:** Docks vendors (Chandlery, etc.) pre-declare
   deliverable items (`RestockQty: 0` now prices sanely after the pricing fix).

Smoke: boot, confirm the runner walks its circuit and stock moves; no panic.

---

## 5. Questlines

### Dock Rat (quest **63**)
A dockworker (ambient-named, e.g. "Teo") is accused of theft he didn't commit.
- **3 breadcrumbs:** his friend pleads at the Salt Cellar; the constable's
  **notice board** (5525) names him; examining the **"stolen" cargo** at the
  Customs Shed (5528) shows inconsistencies.
- **2+ resolutions:** (a) investigate → the real thief is **Foreman Hewd** (5526)
  skimming — search his office (hidden lockbox noun) + confront; (b) gather
  witness testimony around the docks → convince the **Dock Constable**; (c)
  **Old Sable** knows the truth but wants a favor first.
- NPCs: Teo (accused, 9313), Foreman Hewd (9314, the skimmer), Dock Constable
  (9315). Reward: small item + Teo becomes a reliable dock contact.
- SOPs: `grantsQuest` node has `questExcluded` w/ end token; `quest`/`task`
  triggers; give.go recovery if an item is handed over.

### The Bloom Trail — opening (quest **64**, multi-zone spine; *not completable in Docks*)
Discovery-driven, no quest-giver. **Breadcrumbs seeded here only:**
- **Marn's back room (5524):** a timed overheard line ("…if she survives the
  first month…") — canon echo.
- **The Pilings Haunt (5521):** a Bloom-addled wanderer mutters about "the room
  with the low ceiling" (points to the Old Quarter / Lintel St).
- **The Dock Constable (5525):** mentions investigating "something in the
  pilings."
The quest's middle/climax (Cade→Deren chain, the captive) is authored in the
Crafting + Old Quarter builds. Here it only *exists to be noticed*.

---

## 6. Expansion stubs (described, gated, not broken)

- **Chained shipyard** (off North Quay 5502) — harbormaster: "authorized
  personnel only." (Future: sea travel / coastal zones.)
- **Sealed warehouse** (off Tenements 5519) — machinery sounds, a guard who won't
  discuss it. (Future: Bloom refinery / bloodline logistics.)
- **Collapsed underdock tunnel** (off the Underdocks 5520) — warning signs.
  (Future: deeper smuggler network / coastal caves.)

---

## 7. ID allocation

| Type | Block | Use |
|------|-------|-----|
| Rooms | 5500–5529 | 30 (A 5500–5509, B 5510–5519, C 5520–5529) |
| Mobs | 9300–9316 | anchors 9300–9306, ambient/transient 9307–9312, quest NPCs 9313–9316 |
| Items | 40100–40109 | imported goods (sea salt, exotic cloth, etc.), Dock Rat reward/cargo |
| Quests | 63 (Dock Rat), 64 (Bloom Trail spine — opened here) | |
| Dialogue | from 9209 | the 7 anchor trees + quest nodes |
| Factions | `np_dockfolk`, `bloom_trade` (defs authored in this build) | |

Run `python tools/id_inventory.py` before authoring; confirm zone folder name
`new_plymouth_docks` round-trips `ConvertForFilename("New Plymouth Docks")`.

---

## 8. Build sequence (staged, smoke between)

1. **Stage A — Waterfront:** rooms 5500–5509 (cartcheck-clean, River Road 5480
   north exit), the supply anchors (Jesset, Dunmar, Dobb, Greer) + supply-engine
   wiring (§4), the Chandlery vendor. **Boot-smoke.**
2. **Stage B — Dock Streets:** rooms 5510–5519, Bressa + Salt Cellar + cookshop +
   Old Sable + Jesset's home, ambient dock crowd + pooled schedules. **Boot-smoke.**
3. **Stage C — Outer Edge/Shadows:** rooms 5520–5529, Marn + back room, the
   underdocks, the Dock Rat questline (constable/foreman/Teo) + the Bloom Trail
   breadcrumbs, the Crafting bridge + Old Quarter stair stubs. **Boot-smoke.**
4. **District smoke:** nuke instance saves; boot; harness-verify Dock Rat
   end-to-end + the runner circuit moving stock + the breadcrumbs discoverable.

Each stage = its own writing-plans pass + commit; the district is one cohesive
branch (`feature/np-docks-district`).

---

## 9. Quality bar (per `docs/ZONE_EXPANSION.md`)

3-layer room descriptions (atmosphere → detail → interaction hint), ≥2 examinable
nouns/room, sensory variety (the docks lead with *smell* — brine/tar/rotting
wood/livestock per canon), 80-col wrap, idle NPC behaviors, ≥3 dialogue topics,
discoverable quests (3 breadcrumbs / 2 resolutions / no dead ends), `cartcheck`
clean. Canon atmosphere: *"brine and tar and the sweetness of rotting wood…and
under everything a warm animal smell from the livestock pens two streets over."*

---

## 10. Next step

writing-plans → **Stage A** implementation plan (rooms 5500–5509 + the supply-
engine wiring + the 4 waterfront anchors), then B, then C.
