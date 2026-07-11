# Greenford — District 1: River District & Bridge Landing — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build Greenford's entrance district — open the East Road's barred bridge and establish the riverfront (10 rooms, 8 riverfolk/fauna, 3 vendor foods, no quest).

**Architecture:** Pure YAML in a NEW `greenford` zone. The only edit to existing content is opening the East Road's 6277 bridge (south→6278). Reuses River Road's water forageables (no Go change) and fauna patterns.

**Tech Stack:** YAML; `go run`/built binary boot; `cartcheck`; mudagent harness.

**Spec:** `docs/superpowers/specs/2026-06-30-greenford-river-district-design.md` (city-wide: `2026-06-30-greenford-citywide-design.md`).

**Reserved IDs:** rooms **6278–6287**, mobs/dialogue **9501–9508**, items **40152–40154**. No quest, no faction, no buffs.

**Branch:** `feature/greenford-river-district` off `master`.

## Conventions (load-fatal — same as East Road / Confluence)
- **Zone folder = `ConvertForFilename("Greenford")` = `greenford`** (East Road cost a boot cycle on a folder-name mismatch — verify rooms/mobs/dialogue/schedules all under `greenford`).
- `zone-config.yaml` required (name/roomid/defaultbiome/region) or boot panic.
- Mob `name` + room `title` canonical Title-Case; mob filename = `ConvertForFilename(name)`; ambient archetype `noncombat_passive`.
- `idlemessages`/idle lines with colon-space single-quoted; description/noun prose-colons in `>` block scalars.
- Exits are `{roomid}` only — NO `kind:` field.
- Vendors: `craft_support` + `shop:`; items carry a real discipline, never `general`.
- Dialogue voice: NPC text 1st-person, hints 2nd-person, every trigger discoverable; **node-shadowing**: put specific nodes first, avoid short triggers that substring-match other nodes' topics (bit East Road repeatedly — and the feel-tester still caught a hinted "road"/"learn" word not reaching its node, so cross-check EVERY hint word against the node it should hit).
- **Stage explicit git pathspecs, NEVER `git add -A`** (repo carries dirty economy-snapshot noise).
- Pre-smoke: wipe `rooms.instances/*` + `mobs.instances/*`.

---

## Task 1: Zone scaffold + 10 rooms (6278–6287) + the 6277 seam

**Files:**
- Create: `_datafiles/world/dogmud/rooms/greenford/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/greenford/6278.yaml`…`6287.yaml`
- Modify: `_datafiles/world/dogmud/rooms/east_road_to_greenford/6277.yaml` (open the bridge)

- [ ] **Step 1: zone-config.** Create `rooms/greenford/zone-config.yaml`:
```yaml
name: Greenford
roomid: 6278
defaultbiome: city
region: The Tri-Rivers
```

- [ ] **Step 2: Seam.** In `east_road_to_greenford/6277.yaml` (The Greenford Bridge, `{22,-70,0}`, currently exit `north→6276` only; prose has a BARRED toll-gate "NO PASSAGE WITHOUT THE WARDEN'S LEAVE"), add:
```yaml
  south:
    roomid: 6278
```
and **revise the prose** so the gate now reads OPEN/passable — the warden is on duty and admits travelers into Greenford (keep the bridge/river description; just flip the barred toll-gate to an open, warden-staffed gate). Keep the existing nouns; update any that describe the gate as sealed.

- [ ] **Step 3: Author 6278–6287** per this authoritative table (reciprocal exits; NO `kind:`; collision-free vs the East Road frame which uses x≤22, y −66…−70 — these are all y≤-72):

| roomid | title | biome | x | y | z | exits |
|--------|-------|-------|---|---|---|-------|
| 6278 | The Greenford Gate | city | 22 | -72 | 0 | north→6277, south→6279, east→6282 |
| 6279 | The River Landing | water | 22 | -73 | 0 | north→6278, west→6280, south→6284 |
| 6280 | The Watermill | water | 21 | -73 | 0 | east→6279 |
| 6281 | The Fishing Steps | water | 23 | -74 | 0 | north→6282, west→6283 |
| 6282 | Riverside Row | city | 23 | -73 | 0 | west→6278, south→6281 |
| 6283 | The Boathouse | water | 22 | -74 | 0 | east→6281, north→6279(?) — see note |
| 6284 | The Climbing Lane | city | 22 | -75 | 0 | north→6279, south→6285 |
| 6285 | The Riverside Green | city | 21 | -75 | 0 | east→6284, south→6286 |
| 6286 | The Upper Bank | city | 21 | -76 | 0 | north→6285, east→6287 |
| 6287 | The Town Stair | city | 22 | -76 | 0 | west→6286 |

**NOTE on 6283:** the table above is a STARTING layout — the implementer should finalize a clean, fully-reciprocal graph (every exit has its mirror) and a collision-free coordinate set, adjusting 6283's links so it sits sensibly off the landing/fishing cluster. The REQUIRED invariants: 6278 is the bridgehead (north→6277 seam); the water-biome rooms are 6279/6280/6281/6283; the lane climbs from the river up to 6287; **6287 "The Town Stair" is the stub up to District 2 — it has NO onward (up/north) exit** (described in prose only). Re-verify all reciprocals + run boot `ValidateZoneConsistency` in Task 5.

**Each room:** three-layer description (atmospheric lead → grounded detail → interaction hint), ≤80 cols, ≥2 examinable nouns, ~20% with a container noun (nets, a moored skiff, a grain sack, crab pots). Vary the leading sense (river-sound, wet-stone smell, the mill's rhythm, fish-and-tar). Signature: the warm, practical riverfront at the foot of the hill; the university tower glimpsed uphill from the climbing lane (6284+) and named at the Town Stair (6287). Water rooms feel of the river; city rooms of a working town edge.

- [ ] **Step 4: 6287 The Town Stair (stub).** The lane reaches a stair/gate up into the Town Center — the market, a bookshop, the university tower higher up, all described in prose. NO wired up/onward exit (District 2 attaches later). Softer than a barred gate: an open town "just up the steps," a place you'll climb soon.

- [ ] **Step 5: Spawninfo.** Add per Task 3's room column, `respawnrate: "20 real minutes"`.

- [ ] **Step 6: Self-check** — all exits reciprocal (incl. the 6277 seam); no coord collision; no `kind:`; 6287 no onward exit; water biome on 6279/6280/6281/6283; titles Title-Case; colons handled; ≤80 cols.

- [ ] **Step 7: Commit**
```bash
git add _datafiles/world/dogmud/rooms/greenford/ _datafiles/world/dogmud/rooms/east_road_to_greenford/6277.yaml
git commit -m "feat(greenford): zone scaffold + River District 10 rooms (6278-6287) + East Road 6277 seam"
```

---

## Task 2: Items (40152–40154)

**Files:** Create under `_datafiles/world/dogmud/items/materials-40000/`.

- [ ] 3 river/mill vendor foods, each `vendor_categories: [cooking]` (never `general`); model on an existing cooking good (e.g. River Road `40125`/`40126` or East Road `40147`). Suggested:
  - `40152-a_river_trout.yaml`
  - `40153-a_sack_of_milled_flour.yaml`
  - `40154-a_river_loaf.yaml`
  (Reuse River Road fish goods 40125/40126 in the fishmonger's stock where they fit, to avoid surplus items.)
- [ ] Commit:
```bash
git add _datafiles/world/dogmud/items/materials-40000/4015*.yaml
git commit -m "feat(greenford): river-district vendor foods (40152-40154)"
```

---

## Task 3: Mobs + dialogue (9501–9508)

**Files:** `mobs/greenford/9501.yaml`…`9508.yaml`; `dialogue/greenford/9501.yaml`…`9506.yaml` (the 2 fauna 9507/9508 get NO dialogue).

Common ambient shape (copy a River Road mob, e.g. `mobs/river_road/9410-*` for a dialogue NPC, `9412-birrel_the_netmender` for a vendor, `9416/9417` for fauna): `non_combatant: true`, `charm_immune: true`, `hostile: false`, `archetype: noncombat_passive`, `maxwander: 0`, Title-Case name, `groups: [humanoid]` (fauna: the river-fauna shape), idlecommands. Each NPC: ≥3 dialogue topics, idle behaviors, a UNIQUE visible mutation; voice rules; **every hint word must reach a real node** (cross-check triggers vs hints — the East Road feel-test caught hinted words that didn't route); NO quest fields.

| mob | name | room | type |
|-----|------|------|------|
| 9501 | (name) The Bridge-Warden | 6278 | dialogue; grants leave; town/river-trade talk; soft forward-gesture to the university uphill (NO mystery) |
| 9502 | (name) The Miller | 6280 | **cooking vendor** (shop: 40153 flour, 40154 loaf, + a fish); mill/dry-year talk |
| 9503 | A Fisherman | 6281 | dialogue; the river, the catch, the bridge |
| 9504 | A Barge-Hand | 6279 | dialogue; river trade up/down to the Confluence |
| 9505 | (name) The Fishmonger | 6282 | **cooking vendor** (shop: 40152 trout, + 40125/40126 reuse); market patter |
| 9506 | A Washerwoman (or child) | 6285 | ambient; daily-life color |
| 9507 | A Grey Heron | 6281 | fauna (model River Road 9416) |
| 9508 | A River Otter | 6283 | fauna (model River Road 9417) |

The Bridge-Warden (9501) is the welcoming face — gives the player their first read on Greenford as a quiet scholarly town uphill; NEVER any crash-site/mystery content. Riverfolk = no Margin.

- [ ] Commit:
```bash
git add _datafiles/world/dogmud/mobs/greenford/ _datafiles/world/dogmud/dialogue/greenford/
git commit -m "feat(greenford): river-district NPCs + dialogue (9501-9508)"
```

---

## Task 4: Schedule (light)

- [ ] 1–2 anchor schedules (e.g. `gf_warden` @6278 day-post + night; `gf_miller` @6280 day-mill + night-sleep), modeled on `cf_innkeeper`/East Road `er_victualler`. Full 24h coverage, in-zone target rooms. Wire `schedule_id` on the mobs. Vendors stay reliably at their posts by day (findability). Commit:
```bash
git add _datafiles/world/dogmud/schedules/greenford/ _datafiles/world/dogmud/mobs/greenford/
git commit -m "feat(greenford): river-district anchor schedules"
```

---

## Task 5: Boot + cartcheck

- [ ] Wipe instances (`rm -rf .../mobs.instances/* .../rooms.instances/*`).
- [ ] Build + boot; confirm room +10, mobs +8, items +3; **`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; no casing/colon/category/Filepath panics; no `greenford`/62xx/950x warnings.
- [ ] `cartcheck greenford` clean (incl. the 6277 seam). Kill server. Commit any fixes.

---

## Task 6: World-critic + feel polish (MANDATORY)

- [ ] **World-critic pass** over `greenford` — directions vs the river/bridge/hill layout (the Confluence is WEST across the bridge; the town climbs UP; the river runs through); Title-Case; colons; vendor categories; **node-shadowing + every hint word reaches its node**; 6287 reads as "town just up the steps, not yet built" not broken; the warden leaks no mystery.
- [ ] **Feel-test** (harness) — walk the seam from East Road 6277 across the now-open bridge into 6278, the riverfront, both vendors (`list`/`buy`), forage on the water rooms, all NPC dialogue (cross-check hinted topics actually route), the Town Stair stub. Zero-bug bar. Report `tools/playtest/reports/2026-06-30-local-feel-tester-greenford-river.md`.
- [ ] Fix; re-boot; commit.

---

## Task 7: Finish + docs + merge

- [ ] Final clean boot.
- [ ] **Update `docs/ZONE_EXPANSION.md`** row 19 (Greenford): mark **District 1 ✅ built** (rooms 6278–6287) with a "Building (1/5)" status; note the bridge now opens. Commit.
- [ ] `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch.
- [ ] **Update memory** ([[project-zone-expansion-redesign]] + the MEMORY.md index): Greenford District 1 built (UNPUSHED), Eastern Arc progressing; next = District 2 Town Center.
- [ ] Report to the user.

---

## Self-Review

**Spec coverage:** scaffold + rooms + seam (T1) ✓; bridge-opening (T1 S2) ✓; Town Stair stub (T1 S4) ✓; vendor foods (T2) ✓; 6 riverfolk + 2 fauna (T3) ✓; schedules (T4) ✓; boot/cartcheck (T5) ✓; world-critic + feel (T6) ✓; ZONE_EXPANSION + memory + merge (T7) ✓. No quest/faction/forage-code — none in plan, matches spec ✓.

**Placeholder scan:** room/dialogue prose authored by implementer subagents from the per-room/per-NPC briefs in T1/T3 (content, not logic). The "(name)" markers for the warden/miller/fishmonger are intentional builder-named NPCs. No TBD logic.

**Id consistency:** rooms 6278–6287 (10), mobs 9501–9508 (8, with 9507/9508 fauna having no dialogue), items 40152–40154 (3) — consistent across T1/T2/T3. Water biome on 6279/6280/6281/6283 reuses existing River Road forageables (no Go task). The 6277 seam is the only existing-content edit.
