# East Road to Greenford (zone 18) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the East Road to Greenford — a 15-room lore-and-ambient connector (no quest) from the Confluence East Gate (6250) to the unbuilt Greenford, across dry wheat plateau with a waypoint village, an orbital-waystone symbol seed, and a barred Greenford-bridge terminus stub.

**Architecture:** Pure YAML in a new `east_road` zone, plus one small TDD Go change (dry-country forageables in `forager.ForageYields`). Matches the River Road connector pattern. The only edit to existing content is opening Confluence 6250's east exit.

**Tech Stack:** YAML data files; Go (`internal/forager` + tests); `go run .` boot; `cartcheck`; mudagent harness (walkability + the cooking vendor + forageables; no quest).

**Spec:** `docs/superpowers/specs/2026-06-30-east-road-to-greenford-design.md`

**Reserved IDs:** rooms **6263–6277**, mobs/dialogue **9492–9500**, items **40147–40151**. No quest, no faction, no buffs.

**Branch:** `feature/east-road-greenford` off `master`.

## Conventions (same load-fatal rules as every recent zone build)
- **Every zone folder needs `zone-config.yaml`** (name/roomid/defaultbiome/region) — missing → boot panic.
- Mob `character.name` + room `title` canonical Title-Case (em-dash `—` not `--`); "of/the/a" stay lowercase, other words capitalize. **Mob filename = `ConvertForFilename(name)`** exactly.
- `idlemessages` with a colon-space MUST be single-quoted; `description`/`noun` values with prose colons use `>` block scalars.
- Exits are just `roomid` (no `kind:` field — kind is mapper-derived from coord delta; keep spine rooms 1 cell apart so nothing auto-classifies `long`).
- Vendor mobs need `craft_support` + a `shop:` list; salable items carry a real discipline category, **never `general`**.
- Ambient NPC archetype is **`noncombat_passive`** (there is no `noncombat_ambient`).
- Items 40xxx live in `items/materials-40000/` by id-range regardless of type; filenames keep any leading article.
- Dispatch dialogue authoring with the FULL path prefix `_datafiles/world/dogmud/dialogue/east_road/...`.
- Pre-smoke SOP: wipe `rooms.instances/*` + `mobs.instances/*` before every boot.

---

## Task 1: Zone scaffold + the 15 rooms (6263–6277) + the 6250 seam

**Files:**
- Create: `_datafiles/world/dogmud/rooms/east_road/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/east_road/6263.yaml` … `6277.yaml`
- Modify: `_datafiles/world/dogmud/rooms/the_confluence/6250.yaml` (add east exit)

- [ ] **Step 1: zone-config.** Create `rooms/east_road/zone-config.yaml`:
```yaml
name: East Road to Greenford
roomid: 6263
defaultbiome: farmland
region: The Tri-Rivers
```

- [ ] **Step 2: Seam.** In `the_confluence/6250.yaml` (The Greenford Road, `{10,-67,0}`, currently only `west: {roomid: 6249}`), add under `exits:`:
```yaml
  east:
    roomid: 6263
```
Lightly extend its prose so the paved road "running east into open country" now actually leads onward (keep the milestone noun; the road is no longer just described — it goes there).

- [ ] **Step 3: Author 6263–6277** per this authoritative table (reciprocal exits; no `kind:`; coords collision-free, all x≥11):

| roomid | title | biome | x | y | z | exits |
|--------|-------|-------|---|---|---|-------|
| 6263 | The Road Out of the River Country | farmland | 11 | -67 | 0 | west→6250, east→6264 |
| 6264 | The Wheat Country | farmland | 12 | -67 | 0 | west→6263, east→6265 |
| 6265 | The Drovers' Stretch | farmland | 13 | -67 | 0 | west→6264, east→6266 |
| 6266 | The Old Waystone | farmland | 14 | -67 | 0 | west→6265, east→6267 |
| 6267 | The Village Approach | farmland | 15 | -67 | 0 | west→6266, east→6268 |
| 6268 | The Wheatside Hamlet | farmland | 16 | -67 | 0 | west→6267, east→6270, north→6269 |
| 6269 | The Well-Yard | farmland | 16 | -66 | 0 | south→6268 |
| 6270 | The Open Plateau | land | 17 | -67 | 0 | west→6268, east→6271 |
| 6271 | The Lonely Milepost | land | 18 | -67 | 0 | west→6270, east→6272 |
| 6272 | The Falling Ground | land | 19 | -67 | 0 | west→6271, east→6273 |
| 6273 | The Greenford Overlook | land | 20 | -67 | 0 | west→6272, east→6274 |
| 6274 | The River Valley | farmland | 21 | -67 | 0 | west→6273, south→6275 |
| 6275 | The Riverside Road | farmland | 21 | -68 | 0 | north→6274, south→6276 |
| 6276 | The Bridge Approach | farmland | 21 | -69 | 0 | north→6275, south→6277 |
| 6277 | The Greenford Bridge | farmland | 21 | -70 | 0 | north→6276 |

(Reciprocals: 6250E↔6263W, 6263E↔6264W, 6264E↔6265W, 6265E↔6266W, 6266E↔6267W, 6267E↔6268W, 6268N↔6269S, 6268E↔6270W, 6270E↔6271W, 6271E↔6272W, 6272E↔6273W, 6273E↔6274W, 6274S↔6275N, 6275S↔6276N, 6276S↔6277N.)

**Each room:** three-layer description (atmospheric lead → grounded detail → an interaction hint), hard-wrapped ≤80 cols, ≥2 examinable nouns, ~20% with a container noun (a hollow fencepost, a drover's left sack, loose cairn stones). **Vary the leading sense** room to room (the dry wind, the smell of cut grain and dust, the heat off the road, the river-cool returning near 6274). Signature arc = the Confluence river-smell fading west, wheat plateau dryness peaking at 6270–6273, then the Greenford river bringing green back.

- [ ] **Step 4: The symbol seed (6266 The Old Waystone).** A weathered **pre-Founding waystone** standing at the field's edge among the wheat, far older than the farm walls. Highlight the noun in prose as `<ansi fg="itemname">waystone</ansi>` (single word — no hyphenation needed) with a matching `waystone:` noun key (use a `>` block scalar). `look waystone` reveals **nested rings cut into the stone, softened by weather, no readable inscription** — the eastern echo of River Road's Old Waystone Rise. **Understated. No NPC tie. No numerology, no "fourth" talk.**

- [ ] **Step 5: The terminus (6277 The Greenford Bridge).** Greenford is visible **across the river** (the university up on its hill, town roofs, a different water from the Confluence's three). A stone bridge runs out over the water but the way on is **not yet passable** — a closed toll-gate / warden's-leave gate at the far end, described as a clean narrative gate (NOT a broken/dangling exit). Onward direction toward Greenford is in prose only, **no wired exit**. Models River Road's barred Confluence Gates (6105). 6273 "The Greenford Overlook" is where the town's profile **first** appears, far off.

- [ ] **Step 6: Spawninfo.** Add per Task 4's room column. Use `respawnrate: "20 real minutes"`.

- [ ] **Step 7: Self-check** — exits reciprocal (incl. the 6250 seam); no coord collision; no `kind:`; 6277 has no onward exit; titles Title-Case; colons handled (`>` blocks / quoted idlemessages); ≤80 cols; the waystone is the only mystery beat and it's understated.

- [ ] **Step 8: Commit**
```bash
git add _datafiles/world/dogmud/rooms/east_road/ _datafiles/world/dogmud/rooms/the_confluence/6250.yaml
git commit -m "feat(east-road): zone scaffold + 15 rooms (6263-6277) + Confluence 6250 seam"
```

---

## Task 2: Items (40147–40151)

**Files:** Create under `_datafiles/world/dogmud/items/materials-40000/`.

- [ ] **Step 1: Vendor goods (40147–40149)** — 2–3 wheat-country foods for the victualler's cooking shop. Model on an existing `cooking` good (e.g. `40130-*` or `40146-a_bowl_of_travelers_stew.yaml`). Each MUST carry `vendor_categories: [cooking]` (never `general`). Suggested:
  - `40147-a_barley_loaf.yaml`
  - `40148-a_wheel_of_hard_cheese.yaml`
  - `40149-a_handful_of_dried_plums.yaml`
- [ ] **Step 2: Forage items (40150–40151)** — the two dry-country forageables (consumed by Task 3's yield table). Model on `40121-*`/`40122-*` (Amber Valley produce). Give each a real category so they're salable (`vendor_categories: [cooking]`), OR `not_salable: true` if intended non-vendor — pick salable to match 40121/40122:
  - `40150-a_cluster_of_wild_plums.yaml`
  - `40151-a_sheaf_of_gleaned_grain.yaml`
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/items/materials-40000/4014*.yaml _datafiles/world/dogmud/items/materials-40000/4015*.yaml
git commit -m "feat(east-road): wheat-country vendor goods + forageables (40147-40151)"
```

---

## Task 3: Dry-country forageables in ForageYields (TDD)

**Files:**
- Modify: `internal/forager/forage_core.go:31-32` (the `land` + `farmland` yield lists)
- Test: `internal/forager/forage_core_test.go`

- [ ] **Step 1: Write the failing test.** Append to `forage_core_test.go` (mirrors `TestForageYields_WaterHasRiverForageables`):
```go
func TestForageYields_FarmlandHasWheatCountryForageables(t *testing.T) {
	has := func(list []int, id int) bool {
		for _, x := range list {
			if x == id {
				return true
			}
		}
		return false
	}
	farmland := ForageYields["farmland"]
	if !has(farmland, 40150) {
		t.Error("farmland forage should include wild plums (40150)")
	}
	if !has(farmland, 40151) {
		t.Error("farmland forage should include gleaned grain (40151)")
	}
	land := ForageYields["land"]
	if !has(land, 40151) {
		t.Error("dry land forage should include gleaned grain (40151)")
	}
}
```

- [ ] **Step 2: Run it; verify it fails.**
Run: `go test ./internal/forager/ -run TestForageYields_FarmlandHasWheatCountryForageables -v`
Expected: FAIL ("farmland forage should include wild plums (40150)").

- [ ] **Step 3: Implement.** In `forage_core.go`, append the new ids to the existing lists (keep the inline comments):
```go
	"land":      {40004, 40005, 40049, 40047, 40121, 40122, 40151},                       // +40151 gleaned grain (dry wheat plateau, e.g. East Road)
	"farmland":  {40004, 40004, 40005, 40007, 40121, 40121, 40122, 40122, 40150, 40151},  // +40150 wild plums, +40151 gleaned grain (East Road wheat country)
```

- [ ] **Step 4: Run the test; verify it passes.**
Run: `go test ./internal/forager/ -run TestForageYields_FarmlandHasWheatCountryForageables -v`
Expected: PASS.

- [ ] **Step 5: Run the full forager package** to confirm no regressions.
Run: `go test ./internal/forager/`
Expected: ok.

- [ ] **Step 6: Commit**
```bash
git add internal/forager/forage_core.go internal/forager/forage_core_test.go
git commit -m "feat(east-road): wheat-country forageables in farmland/land yields"
```

---

## Task 4: Mobs + dialogue (9492–9500)

**Files:** Create `mobs/east_road/9492.yaml`…`9500.yaml` and `dialogue/east_road/9492.yaml`… for the talking NPCs.

Common ambient shape (copy a River Road ambient like `mobs/river_road/9410-*` or the Confluence `9440-a_confluence_citizen.yaml`): `non_combatant: true`, `charm_immune: true`, `hostile: false`, `archetype: noncombat_passive`, `statpool: 30`, `maxwander: 0`, `activitylevel: 10`, Title-Case name, `groups: [humanoid]`, idlecommands. Each NPC: ≥3 dialogue topics, idle behaviors, a **unique visible mutation** in its appearance. Voice: NPC `text` first-person; `hints` second-person (no 3rd-person self-reference); every trigger discoverable.

| mobid | name | filename | room | type |
|-------|------|----------|------|------|
| 9492 | (builder names) The Victualler | 9492-….yaml | 6268 | **cooking vendor** (`shop:` 40147/40148/40149); rest-stop anchor; road/weather/Greenford talk |
| 9493 | A Drover | 9493-a_drover.yaml | 6265 | dialogue-light (stock, the dry year, the road east) |
| 9494 | A Wheat Farmer | 9494-a_wheat_farmer.yaml | 6264 | dialogue (harvest; the waystone — "older than the walls, nobody minds it") |
| 9495 | A Greenford-Bound Scholar | 9495-a_greenford_bound_scholar.yaml | 6267 | dialogue (what Greenford is, why people study there — the gentle forward nod; **NO crash-site/Reth content**) |
| 9496 | A Carter | 9496-a_carter.yaml | 6271 | dialogue-light (distances, the long haul, resting at the milepost) |
| 9497 | A Well-Woman | 9497-a_well_woman.yaml | 6269 | ambient (hamlet daily life, local color) |
| 9498 | A Wheeling Hawk | 9498-a_wheeling_hawk.yaml | 6264 | fauna (ambient; optional low `maxwander`) |
| 9499 | A Jackrabbit | 9499-a_jackrabbit.yaml | 6270 | fauna |
| 9500 | A Basking Snake | 9500-a_basking_snake.yaml | 6271 | fauna |

Fauna: low-stakes ambient scaled to a traveling player (not a dungeon). The scholar is the one outward gesture — never names any mystery answer; never mentions the waystone's meaning. **No quest fields anywhere.**

- [ ] **Step 1: Author mobs + dialogue** per the table.
- [ ] **Step 2: Commit**
```bash
git add _datafiles/world/dogmud/mobs/east_road/ _datafiles/world/dogmud/dialogue/east_road/
git commit -m "feat(east-road): NPCs + dialogue (9492-9500)"
```

---

## Task 5: Schedules (light)

**Files:** Create `_datafiles/world/dogmud/schedules/east_road/<id>.yaml`; wire `schedule_id` on the mob.

- [ ] 1–2 anchor schedules so the village feels alive without hurting findability — e.g. `er_victualler` @6268 (day shop-tend, evening tavern, night sleep) modeled on `cf_innkeeper`/`cf_historian`. Keep the victualler reliably at/near 6268 by day (it's the only vendor). Travelers/fauna stay put (no schedule) for findability. Validators panic on coverage gaps / unreachable rooms — keep targets in-zone.
- [ ] Commit:
```bash
git add _datafiles/world/dogmud/schedules/east_road/ _datafiles/world/dogmud/mobs/east_road/
git commit -m "feat(east-road): victualler anchor schedule"
```

---

## Task 6: Boot + cartcheck

- [ ] **Step 1:** Wipe instances:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
- [ ] **Step 2:** Build + boot (`go run .`), watch the load log. Confirm: room count +15, mobs +9, items +5; **`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; no casing/colon/category/Filepath panics; no `east_road`/62xx/949x warnings.
- [ ] **Step 3:** `cartcheck east_road` clean (no collisions, no non-reciprocal exits, the 6250 seam consistent). Kill the server.
- [ ] **Step 4:** Commit any fixes.

---

## Task 7: World-critic + feel polish (MANDATORY)

- [ ] **World-critic pass** (data-review mode) over `east_road` — it reliably catches (a) **river/compass-direction botches** (check every direction word against the room canon: the river is *west/behind* leaving the Confluence; Greenford's river is *south/ahead* at the bridge) and (b) **dialogue node-shadowing** (`strings.Contains(topic,trigger)` substring shadow — specific/gated nodes first, or drop colliding short triggers). Also: Title-Case, colons, vendor categories, the waystone stays understated, 6277 reads as "barred crossing" not broken.
- [ ] **Feel-test** — a full west→east walk: the 6250 seam, the dry-plateau progression + sensory variety, the orbital waystone, all NPC dialogues + the cooking-vendor buy, foraging on the plateau (wild plums / gleaned grain), Greenford's profile appearing at 6273, the barred bridge terminus. Zero-bug bar. Report to `tools/playtest/reports/2026-06-30-local-feel-tester-east-road.md`.
- [ ] Fix; re-boot; commit.

---

## Task 8: Finish + docs + merge

- [ ] **Step 1:** Final clean boot.
- [ ] **Step 2: Update `docs/ZONE_EXPANSION.md`** — (a) fix the stale **Confluence row 17** to ✅ Built (all 10 districts / 153 rooms 6106–6257, pushed `524357df`); (b) mark **row 18 East Road to Greenford** ✅ Built with roomid range 6263–6277 + mob/item ranges; (c) refresh the **TOTAL** row (zone + room counts, and "next on-plan" → Greenford). Commit.
- [ ] **Step 3:** `superpowers:finishing-a-development-branch` → merge `feature/east-road-greenford` `--no-ff` into master; delete the branch.
- [ ] **Step 4: Update memory** — `project_zone_expansion_redesign.md` (East Road built, UNPUSHED, the Eastern Arc has begun) + the MEMORY.md index line. Note the next on-plan zone is **Greenford** (45rm — the crash-site-directions hub, Reth/Brennan).
- [ ] **Step 5:** Report to the user: East Road built + merged (unpushed), what's verified, and that Greenford is next.

---

## Self-Review

**Spec coverage:** scaffold+rooms+seam (T1) ✓; symbol waystone (T1 S4) ✓; barred terminus (T1 S5) ✓; vendor goods + forage items (T2) ✓; forageables Go/TDD (T3) ✓; 6 ambient + 3 fauna NPCs (T4) ✓; light schedule (T5) ✓; boot/cartcheck (T6) ✓; world-critic + feel-test (T7) ✓; ZONE_EXPANSION table + memory (T8) ✓. No quest/faction/buff — none in plan ✓.

**Placeholder scan:** room descriptions/dialogue prose are authored by the executing subagents from the per-room/per-NPC briefs in T1/T4 (content, not logic — the brief specifies role, nouns, voice, and constraints per the tables); the only "builder names" placeholder is the victualler's proper name, intentional. No TBD logic steps.

**Type/id consistency:** rooms 6263–6277 (15) match the spec; mobs 9492–9500 (9) match; items 40147–40151 (5) match; forage ids 40150/40151 appear identically in T2 (item files), T3 (yield lists + test), and T7 (feel-test). Coords all x≥11, collision-free vs the Confluence (stops at x:10). The 6250 seam is the only existing-content edit.
