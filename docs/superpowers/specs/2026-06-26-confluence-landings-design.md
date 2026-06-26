# The Confluence — District 1: The Landings — Design

**Date:** 2026-06-26
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (district 1 of 10)
**Seam:** River Road **6105 "The Confluence Gates"** — the barred south stub this district opens.

## Purpose

The player's **arrival in the Confluence**. The Landings is the working
river-ward waterfront — the **Quayfolk's** territory — and the city's first
impression: open the gate, see the tri-city and the temple island across the
channel, meet the river-trade folk, find the barge that runs to New Plymouth.
**No quest** (per the umbrella, quest budget is for Q73/Q74); this district
establishes the city's feel, debuts the Quayfolk faction, lands the Davan
continuity touch, and plants **one** subtle mystery seed.

## Scope & IDs

- **16 rooms, 6106–6121** (the umbrella's reserved block for district 1).
- **Mobs/dialogue 9419–9426** (8; pull the block via `id_inventory --alloc` at build).
- **Items 40127–40129** (2–3 Quayfolk vendor goods).
- **Quests:** none.
- **Zone:** `The Confluence` → folder `_datafiles/world/dogmud/rooms/the_confluence/`
  and `mobs/the_confluence/` (verify via `ConvertForFilename` — underscores).
- **Creates** `rooms/the_confluence/zone-config.yaml`
  (`name: The Confluence`, `roomid: 6106`, `defaultbiome: water`,
  `region: The Tri-Rivers` — confirm the region name is new at build).
- **Biome:** `water` throughout (riverfront).
- **New faction:** `quayfolk` (Confluence river-trade) — the district debuts it
  (mob `groups:` entry; mirror how NP `np_dockfolk` is defined — confirm the
  faction-definition mechanism at build).

## Seam — open the river-ward gate (River Road 6105)

Edit `_datafiles/world/dogmud/rooms/river_road/6105.yaml`:
- Add `south: {roomid: 6106, zone: The Confluence}` to `exits:` (keep `northwest`).
- Revise the `description` so the gate is **open**: the river-warden has returned
  to post, the season turned, the bar lifted. Keep the existing voice and the
  pre-Founding "stones older than the bar" detail (a latent seed).
- Update the `the gates` and `the road south` nouns to reflect the open gate (the
  bar raised, the warden back, the city reachable now). The little "ring of
  stones at the gatepost" idlemessage can stay.

## Layout — 16 rooms

River on the **east edge** (the Aldren, the player's river, running down the
bank); the docks and streets on the **west bank**. Proposed coordinate grid
below — **the build assigns final coords and `cartcheck`-verifies** against River
Road and all prior work (umbrella §7). Exits reciprocal; the build finalizes
wiring from this plan.

| Room | Title | Coord (proposed) | Role |
|------|-------|------------------|------|
| 6106 | The River Gate | {-5,-52,0} | Entry from 6105 (north); the just-opened gate, customs plaza; **Gate-Warden (9419)** |
| 6107 | Gate Street | {-5,-53,0} | The way in from the gate |
| 6117 | Gate Plaza | {-6,-53,0} | West side of the gate, a small square |
| 6108 | The Quayhead | {-5,-54,0} | Customs/quay office; **Dockmaster (9422)** |
| 6113 | Warehouse Row | {-6,-54,0} | Bonded warehouses, cargo |
| 6109 | The Barge Dock | {-4,-54,0} | The moored Confluence↔NP barge; **Barge Master (9420)**; Davan's departure point |
| 6119 | Market Lane | {-5,-55,0} | The street tying inland to waterfront |
| 6114 | The Chandlery | {-6,-55,0} | Ship/dock goods; **Chandler (9423)** vendor |
| 6110 | The Fish Quay | {-4,-55,0} | River-fish market; **Fish-Trader (9421)** vendor |
| 6120 | Quay Crossing | {-5,-56,0} | Central quay junction |
| 6115 | Cooper's Yard | {-6,-56,0} | Dock crafts; ambient stevedore |
| 6111 | The Long Wharf | {-4,-56,0} | Cargo wharf, stevedores; ambient dockhand |
| 6118 | The Old Mole | {-3,-56,0} | An old stone jetty — **the mystery seed** (pre-Founding mooring-stone, weathered orbital mark) |
| 6116 | The Quayside Tavern | {-6,-57,0} | Dockside tavern; **tavern-keeper (9424)** |
| 6112 | Three-Rivers Overlook | {-4,-57,0} | The view: all three rivers join, the **temple island** across the channel |
| 6121 | South Quay Gate | {-5,-58,0} | South edge; **stub** toward District 2 (The Long Quay) |

Proposed exit skeleton (build finalizes + cartchecks):
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
6121 n->6120 (south = stub to The Long Quay, NO exit yet)
```

## NPCs (Quayfolk debut) — mobs 9419–9426

All `non_combatant: true`, `hostile: false`, `charm_immune: true`,
`behavior_archetype: noncombat_passive` (vendors use `noncombat_shopkeeper`).
`groups:` include `humanoid` + `quayfolk`. Unique names (no recycling; check
roster + novel cast).

| Mob | Role | Room | Notes |
|-----|------|------|-------|
| 9419 | Gate-Warden | 6106 | Just back at post; explains the opened gate, welcomes travelers, points to the docks/market |
| 9420 | Barge Master | 6109 | Confluence↔NP passage **lore** (transit deferred); the **Davan tie-back** — recalls a young man from up the valley (Amber Valley) who took passage north a while back |
| 9421 | Fish-Trader | 6110 | **Vendor** (`craft_support: cooking`); river-fish market |
| 9422 | Dockmaster | 6108 | Runs the quay/customs; Quayfolk anchor; the city's pragmatic working voice |
| 9423 | Chandler | 6114 | **Vendor** (ship/dock goods; pick a valid discipline category, see Economy) |
| 9424 | Tavern-keeper | 6116 | The Quayside tavern; gossip, ambient |
| 9425 | Stevedore | 6111 | Ambient dockhand |
| 9426 | Net-mender / cooper | 6115 | Ambient dock craft |

## Economy — items 40127–40129

- **Fish-Trader (9421):** a `cooking` vendor like Birrel, stocking river food.
  Reuse the River Road goods (40123–40126) plus **1–2 new Confluence items**
  (e.g. 40127 smoked river-eel, 40128 a Confluence delicacy). `vendor_categories:
  [cooking]`; never `general`.
- **Chandler (9423):** ship/dock goods. **GOTCHA:** items need a real discipline
  category. Reuse existing general-trade goods where possible (e.g. NP's rope
  40102, waterskin 40103, tinderbox 40105) via an explicit `shop:` list, and add
  at most **1 new** dock item (e.g. 40129 a coil of tarred line) with a valid
  category (`tailoring` for cordage, or `blacksmithing` for fittings — confirm
  against `ValidCraftSupports` at build).

## Lore touches (keep light)

- **Davan continuity:** the Barge Master's dialogue carries one node recalling the
  young Amber Valley man who shipped north — no quest, just a warm tie-back to the
  Southern Road the player came down. (Davan = the woodworker's son, novel canon.)
- **The mystery seed (6118 The Old Mole):** one weathered **orbital mark** on a
  pre-Founding mooring-stone, `look mark` / `look stone` noun — incidental, no
  emphasis, no trigger. The Quayfolk tie up to it daily without a second thought.
  Echoes the River Road waystone; the city's deeper marks come later (Tri-Cross
  Square, the temple).
- **The forgotten fourth / Tri-Cities flavor:** the Overlook (6112) names the
  three rivers (Aldren, Brenn, Solt) and the "Tri-Cities" — without invoking the
  fourth (that thread belongs to the Margin/temple). Civic pride, plainly stated.

## Build approach

Standard content pipeline, on branch `feature/confluence-landings`:
1. Create `zone-config.yaml`; open the 6105 seam.
2. Rooms 6106–6121 (final coords + exits, cartchecked).
3. Mobs 9419–9426 + dialogue; define/attach the `quayfolk` faction.
4. Items 40127–40129; vendor `shop:` lists.
5. Smoke test: wipe instances, boot clean (no panics, `ValidateZoneConsistency
   errors=0 mode=panic`), `cartcheck the_confluence` clean, walk 6105→6106→…,
   buy from both vendors, check the Davan node + the Old Mole mark.
6. Merge `--no-ff` to master; update `ZONE_EXPANSION.md` (Confluence district 1 built).

World after this district: **39 zones / 1022 rooms → 40 zones / 1038 rooms**
(the Confluence zone debuts; +16).

## Out of scope

- Districts 2–10 (south/east/temple-island stubs left unbuilt).
- The working barge transit (deferred per umbrella §9).
- Any mystery exposition beyond the single Old Mole seed (the spine is Q73/Q74).
