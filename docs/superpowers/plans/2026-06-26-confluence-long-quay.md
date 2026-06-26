# The Confluence — District 2: The Long Quay — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Confluence's second district — the 16-room commercial waterfront (rooms 6122–6137): river market, guild halls, counting houses, trade wharves; extend the Quayfolk, debut the `margin` faction via one whisper, plant one mystery seed. No quest.

**Architecture:** Pure content build (room/mob/item/dialogue/faction YAML) in the existing `the_confluence` zone. New `margin` faction (one YAML; the Scrivener joins via `groups:`). Reuses the just-built Landings patterns verbatim. Validated by boot smoke-test + `cartcheck`.

**Tech Stack:** GoMud engine; YAML under `_datafiles/world/dogmud/`; `cartcheck` admin command; boot `ValidateZoneConsistency`.

**Spec:** `docs/superpowers/specs/2026-06-26-confluence-long-quay-design.md`
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md`

---

## Reference patterns (read before authoring) — all from the built Landings (district 1)

- **Rooms (water biome, block-scalar nouns, spawninfo):** `_datafiles/world/dogmud/rooms/the_confluence/6109.yaml` (a waterfront room w/ NPC), `6118.yaml` (the Old Mole — **model the Old Customs House mystery seed on this**), `6112.yaml` (overlook/feature room).
- **Seam edit:** `_datafiles/world/dogmud/rooms/the_confluence/6121.yaml` (the file Task 1 edits) and how the Landings opened a stub — `6105.yaml`.
- **Ambient + vendor mobs:** `_datafiles/world/dogmud/mobs/the_confluence/9419-holt_the_gate_warden.yaml` (ambient), `9421-pella_the_fish_trader.yaml` (cooking vendor), `9423-ferrick_the_chandler.yaml` (general vendor).
- **Dialogue:** `_datafiles/world/dogmud/dialogue/the_confluence/9413.yaml`-style (first-person text, player-perspective hints) — e.g. `9420.yaml` (the barge master, for a lore-aside model).
- **Faction definition:** `_datafiles/world/dogmud/factions/quayfolk.yaml` (the exact pattern for `margin`).
- **Item:** `_datafiles/world/dogmud/items/materials-40000/40125-smoked_river_fish.yaml`.

### Cross-cutting GOTCHAS (apply to every task)

1. Zone folder `the_confluence/` exists; each new file's `zone:` = `The Confluence`. `biome: water` for all 16 rooms.
2. **`nouns` values + multi-sentence `description` use `>` block scalars.**
   **`idlemessages` with a colon-space MUST be double-quoted** (the Landings 6106
   YAML map-vs-string panic — idlemessages can't use `>`, so quote them).
3. **Coords: the spec grid is proposed — the build assigns final coords and MUST
   `cartcheck the_confluence` / boot `ValidateZoneConsistency` (mode=panic),
   resolving any collision** against the Landings (6106–6121) and all prior zones.
   Reciprocal exits.
4. **Mob filename = `{mobid}-{ConvertForFilename(name)}.yaml`** — mismatch panics.
5. **Vendor items need a real `vendor_categories`** from `ValidCraftSupports`
   minus `general` — never `general` on items. **Shops validate item category vs
   `craft_support`** (`buyrules.go`): a `general` vendor accepts everything; a
   `cooking`/`tailoring` vendor only accepts matching categories.
6. **Faction:** create only `factions/margin.yaml`; the member joins via a
   `groups:` entry `margin`. The `factions.rep/` file is runtime-generated and
   gitignored — do NOT create it.
7. **Instance saves shadow templates** — wipe `rooms.instances/*` + `mobs.instances/*`
   (NOT `shops/`) before any smoke test.
8. **Dispatch subagent paths with the FULL `_datafiles/world/dogmud/...` prefix.**

---

## Task 1: Branch + groundwork (margin faction, open the seam)

**Files:**
- Create: `_datafiles/world/dogmud/factions/margin.yaml`
- Modify: `_datafiles/world/dogmud/rooms/the_confluence/6121.yaml`

- [ ] **Step 1: Branch**

```bash
git checkout master && git pull --ff-only 2>/dev/null; git checkout -b feature/confluence-long-quay
```
Expected: `Switched to a new branch 'feature/confluence-long-quay'`.

- [ ] **Step 2: Create the margin faction** (model on `factions/quayfolk.yaml`)

`_datafiles/world/dogmud/factions/margin.yaml`:
```yaml
faction_id: margin
display_name: "The Margin"
description: |
  A quiet community of Confluence scholars, scriveners, and chart-keepers who
  study the old records and notice where they disagree — particularly the
  pre-Founding maps and marks the temple reads one way and the evidence reads
  another. They hold no office and make no noise; they simply keep asking the
  questions the official account would rather were settled. Most of the city
  takes them for harmless antiquarians. A few know better.
default_rep: 10
allies: []
enemies: []
```

- [ ] **Step 3: Open the Landings 6121 seam**

Edit `rooms/the_confluence/6121.yaml`:
- Add `south: {roomid: 6122}` to `exits:` (keep `north: 6120`).
- Revise the `description` and the `the way south` noun so the road now **runs
  on** into the Long Quay — flip "That way is a walk for another day" / "today's
  walk ends at the Landings" to the road being open south into the trade
  district. Keep the voice and the accurate Long-Quay description already present.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/factions/margin.yaml _datafiles/world/dogmud/rooms/the_confluence/6121.yaml
git commit -m "feat(confluence): margin faction + open the Long Quay seam (Landings 6121)"
```

---

## Task 2: The Long Quay rooms (6122–6137)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_confluence/6122.yaml` … `6137.yaml`

`zone: The Confluence`, `biome: water`, hard-wrap ≤80, block-scalar nouns,
quoted colon idlemessages, 3 idlemessages each (commercial-waterfront flavor:
market cries, the chink of coin/scales, cart-wheels, the river under the wharves,
gulls). Add `spawninfo` for NPC rooms.

```yaml
spawninfo:
  - mobid: 9427
    respawnrate: "20 real minutes"
```

| Room | Title | Coord | Mob | Content brief |
|------|-------|-------|-----|---------------|
| 6122 | South Quay Road | {-5,-59,0} | — | Seam from Landings (north); the working dock gives way to commerce |
| 6123 | The Market Cross | {-5,-60,0} | 9433 | Central junction; the market bell; a porter/crier; the busy heart |
| 6124 | The Long Quay | {-4,-60,0} | 9427 | The great trade wharf (riverside); barges of bulk cargo; the River-Trade Factor; a `the wharf` noun |
| 6125 | The Import Wharf | {-3,-60,0} | — | Cargo cranes, bonded import goods, the smell of far places (riverside) |
| 6126 | Quay Road South | {-5,-61,0} | — | Spine; shopfronts, foot traffic |
| 6128 | The Spice Quay | {-4,-61,0} | 9428 | River-import market (riverside); the Importer; sacks of spice, strange smells |
| 6127 | The River Market | {-6,-60,0} | 9429 | Market stalls (west); the Provisioner; the everyday trade |
| 6130 | The Market Lane | {-6,-61,0} | — | Market row continues; awnings, hawkers |
| 6133 | The Rivermen's Guildhall | {-7,-60,0} | 9432 | The river-trade guild; the Guild Steward; ledgers of charters and tolls; a `the guildhall` noun |
| 6134 | The Counting House | {-7,-61,0} | — | A quiet trade counting-house; abacuses, strongboxes, the smell of ink |
| 6129 | The Weighhouse Yard | {-5,-62,0} | — | The customs yard; the great scales, queued carts |
| 6132 | The Weighhouse | {-4,-62,0} | 9430 | Customs scales; the Weighmaster; tallies and tariffs |
| 6131 | The Scriveners' Stall | {-6,-62,0} | 9431 | The Scrivener's stall (the Margin whisper); old charts and copy-work; a `the charts` noun (old harbor maps, some showing a fourth channel — incidental here; the lore is in the Scrivener's dialogue) |
| 6135 | The Old Customs House | {-7,-62,0} | — | **THE MYSTERY SEED** — a disused customs house whose **foundation-stone** carries the weathered **orbital mark** (nested rings). `the foundation-stone` + `the mark` nouns, model on 6118 (incidental, no trigger, understated) |
| 6136 | The South Market | {-5,-63,0} | — | The market thinning toward the south gate |
| 6137 | The Quay's South Gate | {-5,-64,0} | — | South edge; a `the way south` noun framing a stub toward Tri-Cross Square (the civic center) — NO south exit yet (intentional) |

Exit skeleton (finalize + cartcheck):
```
6122 n->6121(Landings) s->6123
6123 n->6122 s->6126 e->6124 w->6127
6124 w->6123 e->6125 s->6128
6125 w->6124
6126 n->6123 s->6129 e->6128 w->6130
6128 w->6126 n->6124
6127 e->6123 s->6130 w->6133
6133 e->6127
6130 n->6127 s->6131 e->6126 w->6134
6134 e->6130
6129 n->6126 s->6136 e->6132 w->6131
6132 w->6129
6131 n->6130 e->6129 w->6135
6135 e->6131
6136 n->6129 s->6137
6137 n->6136   (south = stub to Tri-Cross Square, NO exit)
```

- [ ] **Step 1: Author 6122–6128** (spine + riverside wharves: seam, market cross, the great quay, import wharf, spice quay).
- [ ] **Step 2: Author 6127, 6130, 6133, 6134** (the market + guild row west).
- [ ] **Step 3: Author 6129, 6132, 6131, 6135** (weighhouse yard/house, the Scrivener's stall, and the Old Customs House — the seed nouns are load-bearing).
- [ ] **Step 4: Author 6136, 6137** (south market + the barred south gate stub).
- [ ] **Step 5: Cross-check** every exit's reciprocal; coords match the table; 6121↔6122 seam.
- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/
git commit -m "feat(confluence): The Long Quay rooms 6122-6137 (market, guilds, wharves, Old Customs House)"
```

---

## Task 3: Import & trade goods (items 40130–40132)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40130-river_spice.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40131-salt_cured_roe.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40132-bolt_of_river_cloth.yaml`

Model on `40125-smoked_river_fish.yaml`. All in `materials-40000/`. Block-scalar
descriptions.

- [ ] **Step 1: Author the three items**

- **40130 River-Spice** — Importer good. A warm-smelling ground spice traded up
  the Solt from the southwest. `is_component: true`, `component_tag: river-spice`,
  `type: object`, `subtype: mundane`, value ~8, `vendor_categories: [cooking]`.
- **40131 Salt-Cured Roe** — Importer delicacy. River-fish roe, salt-cured in
  clay, a Confluence luxury. `component_tag: cured-roe`, value ~10,
  `vendor_categories: [cooking]`.
- **40132 Bolt of River-Cloth** — Provisioner trade good. A bolt of sturdy
  river-traded cloth, dyed in the valley's madder-red. `component_tag:
  river-cloth`, value ~9, `vendor_categories: [tailoring]`.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/4013{0,1,2}-*.yaml
git commit -m "feat(confluence): Long Quay import/trade goods 40130-40132 (spice, roe, river-cloth)"
```

---

## Task 4: Long Quay NPCs + dialogue (9427–9433)

**Files:**
- Create: `mobs/the_confluence/9427-...yaml` … `9433-...yaml`
- Create: `dialogue/the_confluence/9427.yaml` … (for the speakers)

All 7: `zone: The Confluence`, `non_combatant: true`, `hostile: false`,
`charm_immune: true`, `speciesid: 1`, `level: 1`, `maxwander: 0`, `statpool: 30`,
`activitylevel: 10`. **Unique names** (no recycling; check roster + novel cast).
Dialogue named by mobid; first-person `text`, player-perspective `hints`,
discoverable triggers.

**`groups`:** all `[humanoid, quayfolk]` **EXCEPT** the Scrivener 9431 →
`[humanoid, margin]`.

| Mob | Role | Room | Dialogue focus |
|-----|------|------|----------------|
| 9427 | River-Trade Factor | 6124 | The three-rivers trade, the guild, the quay's prosperity; the prosperous Quayfolk voice (may name Aldren grain / Brenn wool from the east / Solt ore from the southwest — canon directions) |
| 9428 | The Importer | 6128 | **Vendor** (`craft_support: cooking`, `noncombat_shopkeeper`, `gold: 40`); exotic river-imports |
| 9429 | The Provisioner | 6127 | **Vendor** (`craft_support: general`, `noncombat_shopkeeper`, `gold: 40`); everyday trade goods |
| 9430 | The Weighmaster | 6132 | Customs, scales, tariffs; the regulatory Quayfolk voice |
| 9431 | The Scrivener | 6131 | **THE MARGIN WHISPER** — `groups: [humanoid, margin]`. Keeps old harbor charts; in ONE node notes offhand that the **old charts disagree on the count of waters** — a few show a **fourth channel** long gone — and that "the scholars up in their quarter" have argued it for years. Points curious players toward the Scholars' Quarter. **Understated; no quest token, no givesItem.** |
| 9432 | The Guild Steward | 6133 | The Rivermen's Guildhall — charters, tolls, river-trade matters |
| 9433 | A Quay Porter | 6123 | Ambient (idlecommands hauling/crying wares; minimal or short tree) |

- [ ] **Step 1: Author the Importer (9428)** — cooking vendor, shop list:
```yaml
  shop:
    - itemid: 40130   # river-spice
    - itemid: 40131   # salt-cured roe
    - itemid: 40123   # watercress (reuse)
    - itemid: 40125   # smoked river-fish (reuse)
```
- [ ] **Step 2: Author the Provisioner (9429)** — general vendor (accepts any
  category), shop list:
```yaml
  craft_support: general
  shop:
    - itemid: 40132   # bolt of river-cloth (new)
    - itemid: 40102   # rope (reuse NP good)
    - itemid: 40103   # waterskin (reuse)
    - itemid: 40105   # tinderbox (reuse)
```
- [ ] **Step 3: Author 9427/9430/9431/9432** with dialogue trees (factor,
  weighmaster, the Scrivener + the four-waters aside, guild steward).
- [ ] **Step 4: Author 9433** as light ambient (idlecommands; minimal dialogue).
- [ ] **Step 5: Verify** mob filenames match `ConvertForFilename(name)`; the
  Scrivener carries `margin` (not quayfolk); all others carry `quayfolk`;
  placement rooms (Task 2) have matching `spawninfo`.
- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/ _datafiles/world/dogmud/dialogue/the_confluence/
git commit -m "feat(confluence): Long Quay NPCs 9427-9433 + dialogue + vendors + the Margin scrivener"
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

Run: `go build -o C:/tmp/lq.exe . && C:/tmp/lq.exe` (boot, then stop).
Expected: no panics; `rooms.loadAllRoomZones` loadedCount=**1054**; `mobs` up by 7;
**`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; `factions.LoadAllDefinitions`
count up by 1 (`margin` loads). A coord collision, filename mismatch, or unquoted
colon idlemessage panics here — fix and re-boot.

- [ ] **Step 3: `cartcheck the_confluence`** (running server, admin) — expect clean.

- [ ] **Step 4: Walk it** — Landings 6121 → south → 6122 → through the district;
  `list`/`buy` from the Importer (6128) and the Provisioner (6127); `ask
  <scrivener> charts` / `ask <scrivener> waters` (the four-waters aside); `look
  mark` at the Old Customs House (6135); confirm 6137 south is a dead stub.

- [ ] **Step 5: Update `docs/ZONE_EXPANSION.md`** — the Confluence row (#17):
  district 2 (The Long Quay) built, rooms 6122–6137, "Building (2/10)", world now
  40 zones / 1054 rooms.

```bash
git add docs/ZONE_EXPANSION.md
git commit -m "docs(zones): The Confluence district 2 (The Long Quay) built"
```

- [ ] **Step 6: Merge `--no-ff`**

```bash
git checkout master
git merge --no-ff feature/confluence-long-quay -m "Merge: The Confluence district 2 — The Long Quay"
git branch -d feature/confluence-long-quay
git tag -d master 2>/dev/null || true
```

---

## Self-review checklist (run before merge)

- [ ] All 16 rooms 6122–6137 exist, `zone: The Confluence`, coords cartcheck-clean.
- [ ] `margin.yaml` faction present (no `.rep` file committed); the Scrivener 9431 carries `margin`, all other NPCs `quayfolk`.
- [ ] 6121 south exit added; the "walk for another day" framing flipped to open.
- [ ] No item uses `vendor_categories: general`; Importer goods `cooking`, cloth `tailoring`.
- [ ] All mob filenames match `ConvertForFilename`; vendors stock valid categories.
- [ ] The four-waters aside appears in the Scrivener only; Old Customs House mark the single seed.
- [ ] Boot clean (no panics), `cartcheck the_confluence` clean.
