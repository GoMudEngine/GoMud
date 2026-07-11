# The Confluence — District 3: Tri-Cross Square — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Confluence's third district — the 16-room civic heart (rooms 6138–6153, `city` biome): the central square, The Three Waters inn (with upstairs lodging), the Municipal Hall; seed the Q73 Margin-Notation thread as lore and escalate the orbital-symbol motif. No quest.

**Architecture:** Pure content build (room/mob/item/dialogue YAML) in the existing `the_confluence` zone. No new faction (`margin` already exists; civic NPCs are unfactioned `humanoid`, the one scholar is `margin`). Reuses the built Landings/Long-Quay patterns. Validated by boot smoke-test + `cartcheck`.

**Tech Stack:** GoMud engine; YAML under `_datafiles/world/dogmud/`; `cartcheck`; boot `ValidateZoneConsistency`.

**Spec:** `docs/superpowers/specs/2026-06-26-confluence-tri-cross-square-design.md`
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md`

---

## Reference patterns (read before authoring)

- **`city`-biome civic rooms (for tone/biome):** `_datafiles/world/dogmud/rooms/new_plymouth_merchant/5800.yaml` (a square), and the **vertical-interior pattern** `new_plymouth_merchant/5814.yaml` + `5816.yaml` (the Gilt Threshold inn / Lodging — an inn with `up` to lodging, the model for 6146↔6148).
- **Confluence rooms (block-scalar nouns, mystery-seed nouns, spawninfo):** `_datafiles/world/dogmud/rooms/the_confluence/6135.yaml` (the Old Customs House mark — model the Bell-Tower/Records symbol nouns), `6131.yaml` (a stall/feature room).
- **Seam edit:** `_datafiles/world/dogmud/rooms/the_confluence/6137.yaml` (the file Task 1 edits).
- **Mobs:** `_datafiles/world/dogmud/mobs/the_confluence/9419-holt_the_gate_warden.yaml` (ambient), `9429-lenne_the_provisioner.yaml` (general vendor), `9431-tallis_the_scrivener.yaml` (a `margin` NPC with a lore-aside node — model the Scholar on this).
- **Dialogue:** `_datafiles/world/dogmud/dialogue/the_confluence/9431.yaml` (the four-waters aside model).
- **Item:** `_datafiles/world/dogmud/items/materials-40000/40125-smoked_river_fish.yaml`.

### Cross-cutting GOTCHAS

1. Zone folder `the_confluence/` exists; each file's `zone:` = `The Confluence`. **`biome: city`** for all 16 rooms (NOT water — the waterfront ended at the Long Quay).
2. **`nouns` + multi-sentence `description` use `>` block scalars. `idlemessages` with a colon-space MUST be double-quoted** (the 6106 YAML map-vs-string panic).
3. **Mob `character.name` MUST be canonical Title Case** (the Long Quay porter panic: "a quay porter" → "A Quay Porter"). Filename stays lowercase via `ConvertForFilename`.
4. **Coords: the spec grid is proposed — assign final coords and `cartcheck the_confluence` / boot `ValidateZoneConsistency` (mode=panic), resolving collisions** against the Long Quay (6122–6137) and all prior. Reciprocal exits.
5. **The inn lodging (6148) is a PROPER vertical:** coord `{-4,-67,1}` (same x,y as 6146, z+1); 6146 `up`→6148, 6148 `down`→6146. (Apply the foreman-office fix: vertical fiction = vertical exit + stacked z.)
6. **Vendor items need a real `vendor_categories`** (never `general` on items). The shopkeeper is a `general` vendor → accepts any category, so it can stock mixed reused goods.
7. **Mobs join factions via `groups:`** — civic NPCs `[humanoid]`, the Scholar `[humanoid, margin]` (faction already exists; do NOT recreate it).
8. **Instance saves** — wipe `rooms.instances/*` + `mobs.instances/*` (NOT `shops/`) before smoke test.
9. Dispatch subagent paths with the FULL `_datafiles/world/dogmud/...` prefix.

---

## Task 1: Branch + open the seam (Long Quay 6137)

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/the_confluence/6137.yaml`

- [ ] **Step 1: Branch**

```bash
git checkout master && git pull --ff-only 2>/dev/null; git checkout -b feature/confluence-tri-cross-square
```
Expected: `Switched to a new branch 'feature/confluence-tri-cross-square'`.

- [ ] **Step 2: Open the 6137 seam**

Edit `rooms/the_confluence/6137.yaml`:
- Add `south: {roomid: 6138}` to `exits:` (keep `north: 6136`).
- Revise the `description` and the `the way south` noun so the road now **runs
  on** south into Tri-Cross Square — flip both "a walk for another day" lines to
  the road being open into the square. Keep the voice and the accurate square
  description already present.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6137.yaml
git commit -m "feat(confluence): open the Tri-Cross Square seam (Long Quay 6137)"
```

---

## Task 2: Tri-Cross Square rooms (6138–6153)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_confluence/6138.yaml` … `6153.yaml`

`zone: The Confluence`, **`biome: city`**, hard-wrap ≤80, block-scalar nouns,
quoted colon idlemessages, 3 idlemessages each (civic flavor: the bell, cart-
wheels on the square, a crier's call, pigeons, the murmur of a hall, feast-day
echoes). Add `spawninfo` for NPC rooms.

| Room | Title | Coord | Mob | Content brief |
|------|-------|-------|-----|---------------|
| 6138 | Cross Street | {-5,-65,0} | — | Seam from Long Quay (north); the road leaving the waterfront for the civic district |
| 6139 | The North Cross | {-5,-66,0} | 9439 | North edge of the square; a Town Crier (9439) calling notices |
| 6140 | Tri-Cross Square | {-5,-67,0} | 9440 | **The central plaza** — wide, old paving, a civic monument; a Citizen (9440); the Tri-Cross where the roads (and the city's three rivers) meet. Name the three rivers in flavor (Aldren N / Brenn E / Solt SW). A `the monument` noun |
| 6141 | The South Cross | {-5,-68,0} | — | South edge of the square; the road on toward the temple |
| 6153 | Processional Gate | {-5,-69,0} | — | South edge; a `the way south` noun framing a stub toward the Processional/temple approach — NO south exit yet (intentional) |
| 6145 | The Inn-Yard | {-4,-66,0} | — | The Three Waters' yard (east); stables, arriving coaches and travelers |
| 6146 | The Three Waters | {-4,-67,0} | 9436 | **The great inn** common room; the Innkeeper (9436); warmth, food, gossip. `up` to the lodging |
| 6148 | The Three Waters — Lodging | {-4,-67,1} | — | Inn rooms upstairs; `down` to 6146 ONLY. Prose makes clear it's the upper floor (a stair, the square seen from a window) |
| 6147 | Inn Lane | {-4,-68,0} | — | The lane along the inn's south side |
| 6150 | The Civic Stores | {-3,-66,0} | 9437 | A general-goods shop (east); the Shopkeeper (9437) vendor |
| 6152 | The Notary's Office | {-3,-68,0} | 9438 | A notary/scrivener of civic record (east); deeds, seals, wax |
| 6142 | The Hall Steps | {-6,-66,0} | — | The approach steps to the Municipal Hall (west) |
| 6143 | The Municipal Hall | {-6,-67,0} | 9434,9435 | **The civic hall** (west); the Hall-Clerk (9434) + a Margin Scholar (9435). **THE NOTATION MAP** — give a `the map` noun: a large civic river-chart whose **margin carries an older annotation disagreeing on the count of waters** (a fourth channel noted in a different hand / noted and struck through). Plainly displayed, official, on the wall. (This is the central Q73 breadcrumb — lore only, no trigger.) |
| 6144 | Hall Lane | {-6,-68,0} | — | The lane along the hall's south side |
| 6151 | The Bell-Tower | {-7,-66,0} | — | The civic bell-tower (west). **THE OLD-STONE SYMBOL** — its **pre-Founding base** carries the weathered nested-rings/orbital mark; give `the base` + `the symbol` nouns (model on 6135; plain, no explanation). Note in the room that the **newer civic facades nearby show the same shape reinterpreted as a Chrysalis motif** — observed, not lectured |
| 6149 | The Hall of Records | {-7,-67,0} | — | The records room (west of the hall); old charters, ledgers, more old maps; another sheltered orbital mark on the old stonework (a brief `look` noun) — the "it's everywhere" beat |

Exit skeleton (finalize + cartcheck):
```
6138 n->6137(Long Quay) s->6139
6139 n->6138 s->6140 e->6145 w->6142
6140 n->6139 s->6141 e->6146 w->6143
6141 n->6140 s->6153 e->6147 w->6144
6153 n->6141   (south = stub to the Processional, NO exit)
6145 w->6139 s->6146 e->6150
6150 w->6145
6146 w->6140 n->6145 s->6147 up->6148
6148 down->6146
6147 w->6141 n->6146 e->6152
6152 w->6147
6142 e->6139 s->6143 w->6151
6151 e->6142
6143 e->6140 n->6142 s->6144 w->6149
6149 e->6143
6144 e->6141 n->6143
```

- [ ] **Step 1: Author 6138–6141, 6153** (the civic spine + the square + the south stub).
- [ ] **Step 2: Author 6145, 6146, 6148, 6147, 6150, 6152** (the east side: inn-yard, the inn + upstairs lodging, inn lane, the shop, the notary). Mind the `up`/`down` z+1 vertical for 6146↔6148.
- [ ] **Step 3: Author 6142, 6143, 6144, 6151, 6149** (the west side: hall steps/lane, **the Municipal Hall + the notation map**, **the Bell-Tower + the old-stone symbol**, the Hall of Records). The map (6143) and symbol (6151/6149) nouns are load-bearing.
- [ ] **Step 4: Cross-check** every exit's reciprocal; coords match; 6137↔6138 seam; 6146 up↔6148 down with 6148 at z+1.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/
git commit -m "feat(confluence): Tri-Cross Square rooms 6138-6153 (square, inn, hall, notation map, bell-tower)"
```

---

## Task 3: Civic good (item 40133)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40133-tri_cities_broadsheet.yaml`

Reuse existing goods for most of the shop (40077 Tallow Candle, 40102 rope,
40103 waterskin, 40105 tinderbox, 40123 watercress). Add ONE new civic-flavor
good. Model on `40125-smoked_river_fish.yaml`.

- [ ] **Step 1: Author the item**

**40133 The Tri-Cities Broadsheet** — `40133-tri_cities_broadsheet.yaml`. A
single printed sheet of civic news, notices, and river-trade prices, sold cheap
at the square. `is_component: true`, `component_tag: broadsheet`, `type: object`,
`subtype: mundane`, weight ~0.1, value ~2, rarity_tier ~40, `vendor_categories:
[tailoring]` (paper/fiber).

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40133-tri_cities_broadsheet.yaml
git commit -m "feat(confluence): the Tri-Cities Broadsheet (40133) — civic good"
```

---

## Task 4: Tri-Cross Square NPCs + dialogue (9434–9440)

**Files:**
- Create: `mobs/the_confluence/9434-...yaml` … `9440-...yaml`
- Create: `dialogue/the_confluence/9434.yaml` … (for speakers)

All 7: `zone: The Confluence`, `non_combatant: true`, `hostile: false`,
`charm_immune: true`, `speciesid: 1`, `level: 1`, `maxwander: 0`, `statpool: 30`,
`activitylevel: 10`. Non-vendors `behavior_archetype: noncombat_passive`.
**Unique names** (check roster + novel cast). **Title-Case the `name:` field**
(ambient = `A Town Crier`, `A Confluence Citizen` — capital A). `groups:
[humanoid]` (civic) EXCEPT the Scholar 9435 → `[humanoid, margin]`.

| Mob | Role | Room | Dialogue focus |
|-----|------|------|----------------|
| 9434 | The Hall-Clerk | 6143 | Civic official; the Municipal Hall's business, civic record, the map as municipal property (matter-of-fact about it — to him it's just an old chart) |
| 9435 | A Margin Scholar | 6143 | `groups: [humanoid, margin]`. Studying the notation map; **the four-waters thread** — the maps disagree on the count, a fourth channel keeps appearing on the oldest ones, and **the answer (if any) is up in the Scholars' Quarter** where the Margin keeps at it. Dry, scholarly; **points the player to the Scholars' Quarter**. No quest token. (Model tone on Tallis 9431.) |
| 9436 | The Three Waters Innkeeper | 6146 | The great inn; lodging, food, the city. **Q73 inn-remark breadcrumb (one node):** an old traveler's tale / a regular's claim about a "fourth water" the old maps show — told as inn folklore, an offhand "you hear all sorts in here," NOT analysis |
| 9437 | The Shopkeeper | 6150 | **Vendor** (`craft_support: general`, `noncombat_shopkeeper`, `gold: 40`); civic everyday goods. Shop list below |
| 9438 | The Notary | 6152 | Civic scrivener — deeds, seals, the paperwork of city life (short tree or ambient) |
| 9439 | A Town Crier | 6139 | Ambient (idlecommands crying the day's notices) |
| 9440 | A Confluence Citizen | 6140 | Ambient (idlecommands; the life of the square) |

- [ ] **Step 1: Author the Shopkeeper (9437)** — general vendor:
```yaml
  craft_support: general
  behavior_archetype: noncombat_shopkeeper
  gold: 40
  ...
  shop:
    - itemid: 40133   # tri-cities broadsheet (new)
    - itemid: 40077   # tallow candle (reuse)
    - itemid: 40102   # rope (reuse)
    - itemid: 40103   # waterskin (reuse)
    - itemid: 40105   # tinderbox (reuse)
```
- [ ] **Step 2: Author 9434/9435/9436 with dialogue** (the clerk, the Margin
  scholar + the four-waters thread pointing to the Scholars' Quarter, the
  innkeeper + the inn-remark breadcrumb).
- [ ] **Step 3: Author 9438** (notary, short tree) and **9439/9440** (ambient,
  idlecommands; minimal dialogue).
- [ ] **Step 4: Verify** mob filenames match `ConvertForFilename(name)`; names are
  Title Case; the Scholar carries `margin`, all others `humanoid` only; placement
  rooms (Task 2) have matching `spawninfo`.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/ _datafiles/world/dogmud/dialogue/the_confluence/
git commit -m "feat(confluence): Tri-Cross Square NPCs 9434-9440 + dialogue + civic vendor + the Margin scholar"
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

Run: `go build -o C:/tmp/tcs.exe . && C:/tmp/tcs.exe` (boot, then stop).
Expected: no panics; `rooms.loadAllRoomZones` loadedCount=**1070**; `mobs` up by 7;
**`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**. A coord collision,
filename/casing mismatch, unquoted colon idlemessage, or a bad vertical panics
here — fix and re-boot.

- [ ] **Step 3: `cartcheck the_confluence`** (running server, admin) — expect clean.

- [ ] **Step 4: Walk it** — Long Quay 6137 → south → 6138 → through the square;
  `up` into the inn lodging (6148) and `down` back; `list`/`buy` from the
  Shopkeeper; `ask <innkeeper>` for the inn-remark; `ask <scholar> waters` for the
  thread; `look map` at the Hall (6143); `look symbol` at the Bell-Tower (6151);
  confirm 6153 south is a dead stub.

- [ ] **Step 5: Update `docs/ZONE_EXPANSION.md`** — the Confluence row (#17):
  district 3 (Tri-Cross Square) built, "Building (3/10)", rooms 6138–6153, world
  now 40 zones / 1070 rooms; Next = Scholars' Quarter (Q73).

```bash
git add docs/ZONE_EXPANSION.md
git commit -m "docs(zones): The Confluence district 3 (Tri-Cross Square) built"
```

- [ ] **Step 6: Merge `--no-ff`**

```bash
git checkout master
git merge --no-ff feature/confluence-tri-cross-square -m "Merge: The Confluence district 3 — Tri-Cross Square"
git branch -d feature/confluence-tri-cross-square
git tag -d master 2>/dev/null || true
```

---

## Self-review checklist (run before merge)

- [ ] All 16 rooms 6138–6153 exist, `zone: The Confluence`, `biome: city`, coords cartcheck-clean.
- [ ] The inn vertical works: 6148 at z+1, 6146 `up`↔6148 `down`; `up`/`down` traversal confirmed in-game.
- [ ] 6137 south exit added; "walk for another day" flipped to open.
- [ ] The notation map (6143) + the bell-tower symbol (6151) + records mark (6149) nouns present; Chrysalis-reinterpretation noted on a new facade.
- [ ] The Scholar 9435 carries `margin`; all other NPCs `humanoid` only; all mob names Title Case.
- [ ] The four-waters thread is in the Scholar + the innkeeper's remark only; no quest token anywhere.
- [ ] No item uses `vendor_categories: general`; the broadsheet is `tailoring`.
- [ ] Boot clean (no panics), `cartcheck the_confluence` clean.
