# River Road to the Confluence (Zone 5.3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a 16-room river-valley connector (rooms 6090–6105) linking Amber Valley to the unbuilt Confluence, with ambient NPCs, river fauna, a fishmonger vendor, river forageables, and subtle pre-Founding lore — no quest.

**Architecture:** Pure content build (room/mob/item/dialogue YAML) plus one small Go change (append river forageables to `ForageYields["water"]`). New zone folder `River Road` → `river_road/`. North seam opens an already-barred exit in Amber Valley 6071; south seam stubs toward the unbuilt Confluence. Validated by boot smoke-test + `cartcheck`.

**Tech Stack:** GoMud engine; YAML data files under `_datafiles/world/dogmud/`; Go (`internal/forager`); `go test`; `cartcheck` admin command.

**Spec:** `docs/superpowers/specs/2026-06-26-river-road-to-confluence-design.md`

---

## Reference patterns (read before authoring)

- **Room YAML w/ block-scalar nouns:** `_datafiles/world/dogmud/rooms/amber_valley/6071.yaml` (the seam room — also the file Task 2 edits).
- **Vendor mob (explicit `shop:` list):** `_datafiles/world/dogmud/mobs/new_plymouth_common/9390-mardle_the_sundries_seller.yaml`.
- **Zone fauna mobs:** `_datafiles/world/dogmud/mobs/amber_valley/9407*.yaml`–`9409*.yaml`.
- **Forage data + test:** `internal/forager/forage_core.go` (`ForageYields`), `internal/forager/forage_core_test.go` (`TestForageYields_ForestHasCookingFlora` pattern).
- **Craft/vendor categories:** `internal/shops/shopinventory.go` — `cooking` is the discipline for fishmonger food goods (`ValidVendorCategories` = blacksmithing/alchemy/tailoring/cooking/jewelcrafting/enchanting; **`general` is INVALID on items**).

### Cross-cutting GOTCHAS (Amber Valley leg-1 lessons — apply to every task)

1. **Zone folder = underscores.** Display name `River Road` → folders
   `rooms/river_road/` and `mobs/river_road/`. A mismatch panics at boot
   (`filesystem path ... did not end in Filepath() ...`).
2. **Room `nouns` values that contain prose colons MUST use `>` block
   scalars** (e.g. `the south road: >`). An unquoted scalar with an interior
   `:` breaks YAML parsing. Same for multi-sentence `description:`.
3. **Coordmap must be collision-checked.** A duplicate coord panics at boot via
   `ValidateZoneConsistency`. Verify the proposed coords against Amber Valley's
   southern rooms before committing (6071 is `{-8,-39,0}`).
4. **Quest reward/vendor YAML keys differ from trigger keys** — N/A here (no
   quest), but vendor items use `vendor_categories` (snake) on ItemSpec.
5. **Dispatch paths with the FULL `_datafiles/world/dogmud/...` prefix** when
   tasking subagents, so nothing lands at repo root.
6. **Instance saves shadow template edits.** Before any smoke test, wipe
   `rooms.instances/*` and `mobs.instances/*` (NOT `shops/`).

---

## Task 1: Feature branch

**Files:** none (git only)

- [ ] **Step 1: Create the branch off master**

Run:
```bash
git checkout master && git pull --ff-only && git checkout -b feature/southern-road-river-road
```
Expected: `Switched to a new branch 'feature/southern-road-river-road'`

- [ ] **Step 2: Confirm clean tree**

Run: `git status --short`
Expected: only the pre-existing untracked/modified files from the session start (no new staged changes).

---

## Task 2: Open the north seam (Amber Valley 6071)

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/amber_valley/6071.yaml`

The road is currently barred ("WAY WASHED OUT — NO PASSAGE / SOON"). Open it: add the south exit and revise the description + the `the south road` / `the open country` nouns so the wash-out reads **freshly mended**.

- [ ] **Step 1: Add the south exit**

In the `exits:` block, add the line (keep `north`/`west`):
```yaml
exits:
  north: {roomid: 6070}
  west: {roomid: 6075}
  south: {roomid: 6090, zone: River Road}
```

- [ ] **Step 2: Revise the description** so the barrier is gone

Replace the `description:` body so the lane no longer "gives out" — the road is mended and open south. Keep the dry-valley-lip voice; ~5–7 lines, hard-wrapped ≤80. The `<ansi fg="itemname">south road</ansi>` highlight stays. Example closing beat: "...a fresh-cut roadbed runs on south now toward the river country, the flood-gully bridged and the warning board painted over."

- [ ] **Step 3: Rewrite the `the south road` and `the open country` nouns**

`the south road` noun should now describe the repaired road (the old "NO PASSAGE" board replaced/painted over; the work-gang's fresh stonework; the word that was "SOON" now struck through and "OPEN" beside it). `the open country` noun should read as a reachable destination now ("the way to the Confluence, the road mended at last"). Use `>` block scalars (GOTCHA #2).

- [ ] **Step 4: Verify YAML parses**

Run:
```bash
go run . -h >nul 2>&1 || true
```
(Compilation only; full boot validation happens in Task 10.) Visually confirm indentation. Expected: no syntax error in the file.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/amber_valley/6071.yaml
git commit -m "feat(river-road): open the mended south road seam at Amber Valley 6071"
```

---

## Task 3: Stage 5.3a rooms (6090–6100, 11 rooms)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/river_road/6090.yaml` … `6100.yaml`

Author each room per the table. **All** rooms: `zone: River Road`; hard-wrap prose ≤80; block-scalar nouns (GOTCHA #2); add 3 `idlemessages` (river birds / water over stones / wet earth, dry-wind for the `land` rooms). Exits are reciprocal — author both sides.

| Room | Title | Biome | Coord | Exits | Content brief |
|------|-------|-------|-------|-------|---------------|
| 6090 | The Mended Road | land | {-8,-40,0} | n→6071 (zone Amber Valley), s→6091 | The fresh-cut roadbed where the wash-out was; the **Road Warden (9410)** and a work-camp; raw stone, new ruts. North noun nods back to the valley. |
| 6091 | Descending Track | land | {-8,-41,0} | n→6090, s→6092 | Dry road dropping off the valley lip; scrub thinning; the first far-off glint of water below. |
| 6092 | Where the River Begins | water | {-8,-42,0} | n→6091, s→6093 | A thin stream appears beside the road and joins it; reeds; the air turns damp. The dry/lush transition lands here. |
| 6093 | Riverside Road | water | {-8,-43,0} | n→6092, s→6094, e→6097 | The water broadening; a fork east toward the fishing village (signed). |
| 6094 | The Barge Landing | water | {-8,-44,0} | n→6093, s→6095 | A timber dock; a moored barge; **Dock-hand (9411)**; river traffic visible mid-channel (lore: passage to the Confluence/NP — not a working transit). |
| 6095 | Broadwater Road | water | {-8,-45,0} | n→6094, s→6096 | The river wide and slow; willows; the road hugging the bank. |
| 6096 | The Confluence Bluff | water | {-8,-46,0} | n→6095, s→6101, w→6100 | An overlook: the **second river joining** the first is visible below — the meeting of waters. Bluff noun describes the convergence. (Single subtle echo of the fisher's four-waters line is permitted here but NOT required.) |
| 6097 | Fishers' Landing | water | {-7,-43,0} | w→6093, e→6098 | Village edge: drying nets, beached coracles, fish-smell. |
| 6098 | Netmender's Row | water | {-6,-43,0} | w→6097, e→6099 | Village center; the **Netmender / fishmonger (9412)** and stall; mended nets strung between posts. |
| 6099 | The Smokehouse | water | {-5,-43,0} | w→6098 | Smoke racks, split river-fish; the **Old Fisher (9413)** who carries the one four-waters lore line (Task 7). |
| 6100 | Old Waystone Rise | land | {-9,-46,0} | e→6096 | A low rise off the bluff with a **pre-Founding waystone**; a weathered **inner-orbit symbol** worked into it, now read by pilgrims as a Chrysalis motif. `look symbol` / `look waystone` nouns describe it (NO interaction trigger). |

- [ ] **Step 1: Author 6090–6096** (the spine) following the briefs and gotchas.
- [ ] **Step 2: Author 6097–6099** (the fishing-village spur).
- [ ] **Step 3: Author 6100** (the waystone rise) — the orbital-symbol nouns are lore-load-bearing; weathered, almost flat, no explanation.
- [ ] **Step 4: Cross-check exits** — every exit has its reciprocal in the partner file; 6071 (Task 2) ↔ 6090; coords match the table exactly.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/river_road/
git commit -m "feat(river-road): stage 5.3a rooms 6090-6100 (road, dock, fishing village, bluff)"
```

---

## Task 4: Stage 5.3b rooms (6101–6105, 5 rooms)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/river_road/6101.yaml` … `6105.yaml`

| Room | Title | Biome | Coord | Exits | Content brief |
|------|-------|-------|-------|-------|---------------|
| 6101 | The River Road South | water | {-8,-47,0} | n→6096, s→6102 | Pilgrim traffic begins; foot-worn road; way-markers. |
| 6102 | Pilgrims' Way | water | {-8,-48,0} | n→6101, se→6103 | Road bends **south-east**; small way-shrines; more pilgrims. |
| 6103 | The Pilgrim Camp | water | {-7,-49,0} | nw→6102, se→6104 | A camp outside the city; the two **Pilgrims (9414/9415)**; cookfires, talk of the temple. |
| 6104 | Sight of the Spires | water | {-6,-50,0} | nw→6103, se→6105 | The three rivers and the **city's spires** emerging in haze ahead. |
| 6105 | The Confluence Gates | water | {-5,-51,0} | nw→6104, (NO south exit) | The barred approach to the unbuilt Confluence. A `nouns` entry (`the gates` / `the road south`) describes the city beyond and frames the bar intentionally ("the river ward isn't passing travelers yet / the last span of road is still being cut"). |

- [ ] **Step 1: Author 6101–6105** per briefs; mind the `se`/`nw` diagonal reciprocity and coords.
- [ ] **Step 2: Confirm 6105 has NO `south` exit** (intentional stub) and 6096↔6101 reciprocity.
- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/river_road/
git commit -m "feat(river-road): stage 5.3b rooms 6101-6105 (pilgrim approach + Confluence stub)"
```

---

## Task 5: River goods & forageable items (40123–40126)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40123-watercress.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40124-freshwater_mussels.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40125-smoked_river_fish.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40126-fresh_river_catch.yaml`

> **GOTCHA:** all 40xxx item files live in `items/materials-40000/` regardless of
> type (Filepath() routes by id-range). Salable items need a real
> `vendor_categories` — use `cooking` for all four (food/provisions). Use `>`
> block scalars for multi-sentence descriptions.

- [ ] **Step 1: Author the four item YAMLs**

Model on an existing forageable in `items/materials-40000/` (e.g. `40121-*.yaml`). Each needs at minimum: `itemid`, `name`, `description` (block scalar), a `type`/subtype consistent with edible provisions, a low `value`/`cost`, a `component_tag` if it should be craftable (e.g. `watercress`, `river_fish`), and `vendor_categories: [cooking]`. Watercress (40123) and mussels (40124) are the **forageables**; smoked river-fish (40125) and fresh catch (40126) are **vendor stock** (40126 may also be a fauna drop from 9418).

Example (40125):
```yaml
itemid: 40125
name: smoked river-fish
description: >
  A split river-fish, brined and hung over slow alder-smoke until the flesh
  goes amber and keeps for the road. The river towns trade them by the string.
type: food
value: 6
component_tag: river_fish
vendor_categories:
  - cooking
```
(Adjust `type`/fields to match the existing forageable schema you opened.)

- [ ] **Step 2: Verify the files load**

Run: `go build ./... ` then a quick boot is deferred to Task 10. Confirm `vendor_categories` is `cooking` (NOT `general`) in all four.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/4012{3,4,5,6}-*.yaml
git commit -m "feat(river-road): river food goods 40123-40126 (watercress, mussels, smoked/fresh fish)"
```

---

## Task 6: Wire river forageables into the water biome (Go, TDD)

**Files:**
- Modify: `internal/forager/forage_core.go:35`
- Test: `internal/forager/forage_core_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/forager/forage_core_test.go` (mirror `TestForageYields_ForestHasCookingFlora`):
```go
func TestForageYields_WaterHasRiverForageables(t *testing.T) {
	water := ForageYields["water"]
	has := func(id int) bool {
		for _, x := range water {
			if x == id {
				return true
			}
		}
		return false
	}
	if !has(40123) {
		t.Error("water forage should include watercress (40123)")
	}
	if !has(40124) {
		t.Error("water forage should include freshwater mussels (40124)")
	}
}
```

- [ ] **Step 2: Run it — expect FAIL**

Run: `go test ./internal/forager/ -run TestForageYields_WaterHasRiverForageables -v`
Expected: FAIL ("water forage should include watercress (40123)").

- [ ] **Step 3: Append the IDs to the water yield slice**

In `internal/forager/forage_core.go`, change line 35:
```go
	"water":     {40058, 40058, 40058, 40058, 40058, 40059, 40123, 40124}, // +40123 watercress, +40124 freshwater mussels (river country, e.g. River Road)
```

- [ ] **Step 4: Run it — expect PASS**

Run: `go test ./internal/forager/ -run TestForageYields_WaterHasRiverForageables -v`
Expected: PASS.

- [ ] **Step 5: Run the whole forager package** to confirm no regressions

Run: `go test ./internal/forager/`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/forager/forage_core.go internal/forager/forage_core_test.go
git commit -m "feat(forage): add watercress + freshwater mussels to water-biome yields"
```

---

## Task 7: Ambient NPCs + fishmonger vendor + dialogue (9410–9415)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/river_road/9410-road_warden.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9411-dock_hand.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9412-netmender.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9413-old_fisher.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9414-pilgrim.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9415-pilgrim_companion.yaml`
- Create dialogue files for the speaking NPCs under `_datafiles/world/dogmud/dialogue/river_road/` (full prefix, GOTCHA #5).

> Pick **unique names** (check against existing NPC roster + the novel cast —
> no recycling). All six are `non_combatant: true`, `hostile: false`,
> `charm_immune: true`. Place each via the room's spawn list (or the mob's home
> room) matching the Task 3/4 assignments: 9410→6090, 9411→6094, 9412→6098,
> 9413→6099, 9414/9415→6103.

- [ ] **Step 1: Author the fishmonger (9412)** modeled on Mardle (9390)

`craft_support: cooking`, `non_combatant: true`, an explicit `shop:` list, and a Chrysalis-flavored description. Stock the food goods:
```yaml
  shop:
    - itemid: 40125   # smoked river-fish
    - itemid: 40126   # fresh river catch
    - itemid: 40123   # watercress
    - itemid: 40124   # freshwater mussels
```

- [ ] **Step 2: Author 9410/9411/9413/9414/9415** (ambient, low statpool) with
  `idlecommands` flavor. The **Old Fisher (9413)** carries the **single**
  four-waters breadcrumb — one offhand dialogue line naming a fourth channel
  "that went dry before my grandfather's time." Do NOT repeat it elsewhere.

- [ ] **Step 3: Author dialogue** for the speakers (warden: the reopening;
  dock-hand: river traffic / the Confluence & NP; netmender: village life;
  old fisher: the rivers + the one four-waters line; pilgrims: the temple, the
  symbol above its door, the pull south). NPC `text` first-person; `hints`
  player-perspective; every trigger discoverable (CLAUDE.md dialogue SOP).

- [ ] **Step 4: Verify mob filenames** match `ConvertForFilename(name)` (a
  mismatch panics at boot). Confirm placement rooms exist.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/river_road/ _datafiles/world/dogmud/dialogue/river_road/
git commit -m "feat(river-road): ambient NPCs 9410-9415 + fishmonger vendor + dialogue"
```

---

## Task 8: River fauna (9416–9418)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/river_road/9416-grey_heron.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9417-river_otter.yaml`
- Create: `_datafiles/world/dogmud/mobs/river_road/9418-<shallows-creature>.yaml`

Model stats on Amber Valley cave fauna (9407–9409) — same low-level power band.

- [ ] **Step 1: Author the three fauna.** 9416 heron (weak, evasive — high dodge,
  low HP); 9417 otter (fast, low HP); 9418 a mutated eel/gar in the shallows
  (the zone's tougher target — `archetype: fighting`, may drop 40126 fresh
  catch). Spawn them across the `water`-biome rooms (6092–6096, 6101–6104).
- [ ] **Step 2: Confirm filenames** match `ConvertForFilename(name)`.
- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mobs/river_road/941{6,7,8}-*.yaml
git commit -m "feat(river-road): river fauna 9416-9418 (heron, otter, shallows creature)"
```

---

## Task 9 (OPTIONAL): Fisherfolk schedules

**Files:**
- Create: `_datafiles/world/dogmud/schedules/river_road/<id>.yaml` (+ `schedule_id:` on the mob)

Nice-to-have, **skip if it risks the smoke test**. 1–2 simple day-post /
night-sleep schedules (netmender works the row by day, sleeps at the
smokehouse). Must be **findability-preserving** (NPC stays in its advertised
room during play hours) and cover all 24h (validator panics on gaps — see
`docs/schemas/schedule.md`).

- [ ] **Step 1: Author 1–2 schedules** covering 24h, target rooms reachable.
- [ ] **Step 2: Commit** (only if added):
```bash
git add _datafiles/world/dogmud/schedules/river_road/ _datafiles/world/dogmud/mobs/river_road/
git commit -m "feat(river-road): day-post/night-sleep schedules for fisherfolk"
```

---

## Task 10: Smoke test, cartcheck, docs, merge

**Files:**
- Modify: `docs/ZONE_EXPANSION.md` (mark 5.3 built)

- [ ] **Step 1: Wipe instance saves** (GOTCHA #6 — NOT `shops/`)

Run:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot the server**, watch for clean load

Run: `go build ./... && go run .` (let it boot, then stop it).
Expected: no panics; `rooms.LoadDataFiles()` / `mobs.LoadDataFiles()` /
`quests.LoadDataFiles()` log non-zero counts; **`ValidateZoneConsistency` errors=0**
(GamePlay.MapConsistencyEnforce). A coord collision or filename mismatch will
panic here — fix and re-boot.

- [ ] **Step 3: `cartcheck` the new zone**

In the running server (admin), run `cartcheck river_road`.
Expected: clean — no coordinate collisions, no non-reciprocal exits, no stray
wrap exits. (The 6105 south stub is fine — it has no exit, not a one-way.)

- [ ] **Step 4: Walk the seam** Amber Valley 6071 → south → 6090 → … → 6105;
  east into the fishing village (6097–6099); `buy` from the fishmonger;
  `forage` in a `water` room and confirm watercress/mussels can appear;
  `look symbol` at 6100; confirm the old fisher's four-waters line; confirm
  6105 south is barred with the intentional message.

- [ ] **Step 5: Update `docs/ZONE_EXPANSION.md`** — mark Zone 5.3 built (leg 2),
  note rooms 6090–6105, world total 39 zones / 1022 rooms.

```bash
git add docs/ZONE_EXPANSION.md
git commit -m "docs(zones): mark River Road to the Confluence built (Southern Road leg 2)"
```

- [ ] **Step 6: Merge to master** `--no-ff`

```bash
git checkout master
git merge --no-ff feature/southern-road-river-road -m "Merge: River Road to the Confluence (Southern Road leg 2)"
```

(PATCH_NOTES.md + the actual prod push happen later under the Pre-Push SOP, not in this plan.)

---

## Self-review checklist (run before merge)

- [ ] Every room 6090–6105 exists, `zone: River Road`, coords match the tables, exits reciprocal.
- [ ] 6071 south exit added; barrier prose removed.
- [ ] No item carries `vendor_categories: general`; all four use `cooking`.
- [ ] `go test ./internal/forager/` green; water yields include 40123/40124.
- [ ] Mob filenames match `ConvertForFilename(name)`; all ambient NPCs `non_combatant`.
- [ ] Four-waters lore appears in exactly ONE place (old fisher).
- [ ] Boot clean (no panics), `cartcheck river_road` clean.
