# New Plymouth — Temple District Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the New Plymouth Temple Quarter (`new_plymouth_temple`, 25 rooms
5900–5924): the Grand Temple + 7 anchors + ambient, dialogue, the `temple_np`
faction (+ `bloodline_domestic` ally backfill), an opt-in Grand Temple respawn
anchor, anchor schedules, Dobb's visible transit to the temple gate, and the
Archive deep-lore stub + Ept/Dross lore breadcrumbs — boot-clean, harness-tested.

**Architecture:** Content (YAML) authored by content subagents within a
pre-allocated ID block, plus one small TDD Go change (the respawn `HomeLocations`
map entry) and a patrol-YAML extension. Validated by server boot + `cartcheck` +
schedule/patrol validators + `go test` (respawn) + a harness playtest. The
controller (main loop) drives all shell — subagents are shell-denied in this env.

**Tech Stack:** GoMud engine (run locally), DOGMud world YAML (`docs/schemas/`),
`tools/id_inventory.py`, `cartcheck`, `go test ./internal/characters/...`, the
`/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-06-22-np-temple-district-design.md`.

---

## Conventions for every task (READ FIRST — identical discipline to prior districts)

- **Branch:** `feature/np-temple-district` (from `master` in Task 0). Commit per
  stage; trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Controller drives all shell** (subagents Write/Edit YAML only; controller runs
  boot/cartcheck/test/commit).
- **Boot test = verification** (YAML errors panic at startup):
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . > /tmp/t_boot.log 2>&1 &
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath" /tmp/t_boot.log   # expect none
  grep -nE "ValidateZoneConsistency|loadAllRoomZones|mobs.LoadDataFiles|factions.LoadAllDefiniti|LoadSchedules|LoadPatrols|Server Ready" /tmp/t_boot.log
  ```
  NOT bare `panic` (gotcha #8). Kill after (`taskkill //F //IM go.exe //T`).
- **Coordinates — solve against the LIVE frame.** Controller dumps occupied cells
  before Stage A/B and hands exact collision-free coords to the room subagent.
  Temple goes EAST of the Merchant zone (x[−8..+6], y[85..95]); entry seam Merchant
  5823 → east. `cartcheck`/`ValidateZoneConsistency errors=0` is the gate.
- **Authoring gotchas:** (1) `": "` in a prose value → `" — "`. (2) faction refs:
  a MUTUAL ref between two factions both built (temple_np↔bloodline_domestic) is
  fine; a ref to a not-built faction PANICS. (3) Title-Case mob names. (4) faction
  via `groups:`. (5) no exits to unbuilt rooms (prose-stub). (6) shop mobs need
  `craft_support:`. (7) noun ansi `<ansi fg="itemname">noun</ansi>` only; post-grep
  `fg="[^"]* [^"]*"` → zero. (8) interiors compass/`up`/`down` only. (9)
  **sentence-leading noun tags lowercase** (match the lowercase noun key — a real
  prior defect). (10) avoid short trigger stems that prefix-shadow longer
  discoverable words (the `buy`/`buyer` lesson).
- **Reference files:** rooms `…/rooms/new_plymouth_merchant/5802.yaml` (hub) +
  `…/5820.yaml` (impassable gate, for Holt's Archive); mob+dialogue
  `…/mobs/new_plymouth_merchant/9346-goss_the_moneylender.yaml` (a no-shop anchor)
  + `…/dialogue/new_plymouth_merchant/9346.yaml`; faction `…/factions/temple_np`…
  use `np_dockfolk.yaml` shape; schedule `…/schedules/new_plymouth_merchant/np_merchant_vell.yaml`;
  patrol `…/patrols/new_plymouth/np_docks_runner_circuit.yaml`; respawn
  `internal/characters/respawn_home.go` + `respawn_home_test.go`.

---

## ID allocation (pre-assigned)

| Kind | Range | Assignment |
|------|-------|------------|
| Rooms | **5900–5924** | see Room Manifest; **5901 = the Grand Temple sanctuary = the respawn room** |
| Mobs | **9356–9367** | 7 anchors (9356–9362) + 5 ambient (9363–9367) |
| Dialogue | by mobid | one per speaking mob |
| Faction | `temple_np` (create) + `bloodline_domestic` (ally backfill) | |
| Zone folder | `new_plymouth_temple` | `ConvertForFilename("New Plymouth Temple")` |
| Items / Quests | none (reuse existing if a prop is needed) | |

---

## Room Manifest (25 — clusters + purpose; coords solved at Stage A dispatch)

Entry: Merchant **5823 (Porters' Dock) → east** → the temple processional road
(5900). The Grand Temple sanctuary is **5901** (the respawn room).

**Cluster 1 — the Grand Temple + entry (6):** 5900 The Processional Road (entry
from Merchant 5823; the road climbs to the precinct), 5901 **The Grand Temple —
Sanctuary** (the hub + respawn room; the great worship hall), 5902 The High Altar
(off the sanctuary; the respawn/rest focus), 5903 The Temple Gate Plaza (pilgrim
arrival; Dobb's visible transit stop; a beggar), 5904 The Cloister Walk (connector
ringing the precinct), 5905 The Bell Tower Base (vertical flavor / ambient).

**Cluster 2 — the Keeper's House (4):** 5906 The Keeper's House — Hall (**Yelin
9356**, warden), 5907 The Keeper's House — Desk (**Father Thane 9357**, intake),
5908 The Keeper's House — Lodging (up; keepers' rooms), 5909 The Warden's Cell
(Yelin's room).

**Cluster 3 — chapel + canon (4):** 5910 The Healer's Chapel (**Sister Alms 9361**),
5911 The Chapel — Recovery Room (up/adjacent), 5912 The Canon's Cell (**Canon Merid
9358**; blessings), 5913 The Canon's Oratory (Merid's private prayer room).

**Cluster 4 — seminary + courtyard + Archive (6):** 5914 The Seminary — Dormitory
(**Novice Ept 9359**; the orbital-symbol lore), 5915 The Seminary — Study Hall,
5916 The Temple Courtyard (**Scholar Dross 9360**; debates; the gallery-cipher
lore), 5917 The Archive — Reading Room (public scholarship), 5918 The Restricted
Collection — Door (**Archivist Holt 9362**; a LOCKED deep-lore gate — prose-only,
impassable, NO interior exit, like Merchant 5820), 5919 The Archive Stacks
(public stacks, off the reading room).

**Cluster 5 — connective / civic / ambient (5):** 5920 The Pilgrims' Rest (a
hostel room; ambient pilgrim), 5921 The Almonry (alms distribution; a lay sister),
5922 The Censer Court (incense + an acolyte), 5923 The Garden of Repose (a quiet
contemplative garden), 5924 The Sexton's Walk (a maintenance lane; ambient).

(The subagent wires intra-cluster + cluster-to-sanctuary exits per the controller's
exact coords; live-in rooms use `up`/`down` or adjacent cells.)

---

## Mob Manifest (12 — 9356–9367)

| Mob | Name | Work room | Mutation | Shop / `craft_support` | `groups` | Notes |
|-----|------|-----------|----------|------------------------|----------|-------|
| 9356 | Yelin | 5906 | hands worn smooth like prayer-stones | none | humanoid, temple_np | canon; warden; gatekeeps keepers |
| 9357 | Father Thane | 5907 | a voice that soothes the anxious | none | humanoid, temple_np | canon; desk intake; orderly |
| 9358 | Canon Merid | 5912 | a faint dawn-prayer halo-glow | optional minor `general` blessing-goods shop (reuse existing items, verify) — or none | humanoid, temple_np | blessings/regen; Archive access; bloodline-aligned |
| 9359 | Novice Ept | 5914 | sees the old orbital symbol everywhere | none | humanoid, temple_np | doubting; the orbital-symbol lore; **cross-refs Orin (Crafting)** |
| 9360 | Scholar Dross | 5916 | an overlarge, veined cranium | none | humanoid, temple_np | argues inscriptions' age; the gallery-cipher lore |
| 9361 | Sister Alms | 5910 | warm hands that mend | optional minor `general` healing-goods shop (reuse existing, verify) — or none | humanoid, temple_np | the Ysolde mirror |
| 9362 | Archivist Holt | 5918 | eyes that catalog at a glance | none | humanoid, temple_np, **bloodline_domestic** | guards the Restricted Collection; institutional intertwine |
| 9363 | A Pilgrim | 5920 | — | — | humanoid | ambient |
| 9364 | A Lay Sister | 5921 | — | — | humanoid, temple_np | ambient |
| 9365 | An Acolyte | 5922 | — | — | humanoid, temple_np | ambient |
| 9366 | A Beggar at the Gate | 5903 | — | — | humanoid | ambient |
| 9367 | A Visiting Keeper | 5908 | — | — | humanoid | ambient |

---

## Task 0: Branch + ID sanity + coordinate groundwork

- [ ] **Step 1: Branch** — `git checkout master && git checkout -b feature/np-temple-district`.
- [ ] **Step 2: Confirm IDs free** — `python tools/id_inventory.py --type rooms | grep -E "temple|next free"` (expect next-free 5825 → 5900–5924 clear); `--type mobs` next-free ≥ 9356.
- [ ] **Step 3: Dump the live coordinate frame** — run the per-zone occupied-cell dump (as used in the Merchant build) over `new_plymouth_common/crafting/merchant`, confirm the eastern frontier (Merchant max x = −9), and solve 5900–5924 into free cells east of x −9 (Grand Temple sanctuary 5901 as the hub), keeping Merchant 5823 → east → 5900 a 1-step exit. Record the assignment.
- [ ] **Step 4: Baseline boot** — boot-test recipe; expect `ValidateZoneConsistency errors=0`. Kill server.

## Task 1: Stage A rooms — Grand Temple + entry seam

**Files:** Create `rooms/new_plymouth_temple/5900.yaml`…`5905.yaml` + `zone-config.yaml`; Modify `rooms/new_plymouth_merchant/5823.yaml` (add `east`→5900).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Cluster 1 manifest + the controller's exact coords + Conventions + reference files. Author 5900–5905 + `zone-config.yaml` (name `New Plymouth Temple`, region `Windward Marches`, roomid 5901). **5901 (the Sanctuary)** is the richest room (the great worship hall; the respawn focus — make it feel like the place the dead return to). 5903 (Gate Plaza) is the pilgrim-arrival + Dobb-transit stop. Prose-stub the Long-Market terminus (no further east exit beyond the precinct).
- [ ] **Step 2:** Edit `new_plymouth_merchant/5823.yaml` — add `east:` → roomid 5900 (reciprocal of 5900's `west`). Keep 5823's existing exits + prose intact.
- [ ] **Step 3: ansi leak check** — `grep -rnE 'fg="[^"]* [^"]*"' _datafiles/world/dogmud/rooms/new_plymouth_temple/` → none.
- [ ] **Step 4: Boot-verify + cartcheck** — boot-test recipe; `errors=0`. Re-solve coords on collision.
- [ ] **Step 5: Commit** — `feat(np-temple): Stage A rooms (5900-5905) — the Grand Temple, gate plaza, Merchant entry`.

## Task 2: Stage B rooms — the cluster precincts

**Files:** Create `rooms/new_plymouth_temple/5906.yaml`…`5924.yaml` (19 rooms).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with Clusters 2–5 manifest + exact coords + Conventions. The **Restricted Collection Door (5918)** is a LOCKED deep-lore gate: prose-only, impassable, NO interior exit (copy the technique from Merchant 5820 — the door does not open). The seminary dormitory (5914) and courtyard (5916) carry the **orbital-symbol** and **gallery-cipher** lore nouns respectively (evocative, unresolved). Live-in cells (warden's 5909, canon's 5912/5913) per the manifest.
- [ ] **Step 2: ansi leak check** (same) — none.
- [ ] **Step 3: Boot-verify + cartcheck** — clean.
- [ ] **Step 4: Commit** — `feat(np-temple): Stage B rooms (5906-5924) — Keeper's House, chapel, seminary, courtyard, Archive`.

## Task 3: The `temple_np` faction + `bloodline_domestic` ally backfill

**Files:** Create `factions/temple_np.yaml`; Modify `factions/bloodline_domestic.yaml`.

- [ ] **Step 1: Author `temple_np.yaml`** (match `np_dockfolk.yaml` shape):
```yaml
faction_id: temple_np
display_name: "The Temple of New Plymouth"
description: |
  The faith that keeps New Plymouth's calendar, its altars, and its Archive — lay
  brothers and wardens, canons and healers, scholars and doubting novices. It
  shelters pilgrims, returns the dead to the world, and keeps the city's longer
  memory. It is also, at its institutional summit, intertwined with the bloodline
  that rules: the canons answer to interests beyond the altar, and the Restricted
  Collection answers to no one a supplicant can name.
default_rep: 0
allies:
  - bloodline_domestic
enemies: []
```
- [ ] **Step 2: Backfill `bloodline_domestic.yaml`** — change `allies: []` to:
```yaml
allies:
  - temple_np
```
(keep `enemies: [cooperage_circle]`). Both factions load this build → the mutual ally ref resolves.
- [ ] **Step 3: Boot-verify** — `factions.LoadAllDefinitions loadedCount` increments to 19; no faction panic.
- [ ] **Step 4: Commit** — `feat(np-temple): temple_np faction + bloodline_domestic ally backfill`.

## Task 4: Stage C — the 7 anchors (9356–9362) + dialogue

**Files:** Create `mobs/new_plymouth_temple/9356-*.yaml`…`9362-*.yaml` + `dialogue/new_plymouth_temple/9356.yaml`…`9362.yaml`.

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Mob Manifest rows 9356–9362 + dialogue SOPs + reference files. Each: Title-Case name, `ConvertForFilename` filename, mutation woven into the description, `non_combatant:true`/`hostile:false`/`maxwander:0`, `groups:` per manifest (temple_np on all 7; **Holt 9362 in BOTH temple_np + bloodline_domestic**), ≥3 discoverable first-person dialogue topics, **no quests**. No-shop anchors omit `craft_support:`; if Merid/Alms get a minor shop, reuse VERIFIED-existing item ids + a valid `craft_support:`. Key dialogue intents: **Ept** — his doubt + the orbital symbol + "ask Orin in the Crafting Quarter" cross-reference (lore, unresolved); **Dross** — the inscriptions' age + the gallery cipher (points at the Noble gallery, later); **Holt** — cold gatekeeper of the Restricted Collection, reveals nothing, the bloodline-temple intertwine felt not stated; **Merid** — blessings + bloodline-aligned (serenely evasive about the Archive); **Alms** — the Ysolde mirror (she heals openly where Ysolde heals in shadow); **Yelin/Thane** — the canon-keeper intake, orderly, 19-years-of-service warmth.
- [ ] **Step 2: Verify filenames** match `<id>-<ConvertForFilename(name)>.yaml`.
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +7; `ValidateShopMobTags` + `AssertCanonical` pass.
- [ ] **Step 4: Commit** — `feat(np-temple): the 7 anchors + dialogue (9356-9362)`.

## Task 5: Stage C — ambient mobs (9363–9367) + room spawns

**Files:** Create `mobs/new_plymouth_temple/9363-*.yaml`…`9367-*.yaml`; Modify room YAMLs to add `spawninfo:`.

- [ ] **Step 1:** Author 5 ambient `non_combatant` mobs (manifest 9363–9367; `behavior_archetype: noncombat_passive`, no shop/craft_support/dialogue; temple_np on the lay sister 9364 + acolyte 9365). Per the Merchant ambient pattern.
- [ ] **Step 2: Add `spawninfo:`** placing anchors (9356→5906, 9357→5907, 9358→5912, 9359→5914, 9360→5916, 9361→5910, 9362→5918) + ambient (9363→5920, 9364→5921, 9365→5922, 9366→5903, 9367→5908), `respawnrate: "10 real minutes"`.
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +5; spawns load.
- [ ] **Step 4: Verify spawn coverage** — `grep "mobid: 93XX"` per room.
- [ ] **Step 5: Commit** — `feat(np-temple): ambient residents (9363-9367) + room spawns`.

## Task 6: The Grand Temple respawn anchor (TDD Go change)

**Files:** Modify `internal/characters/respawn_home.go`; Test `internal/characters/respawn_home_test.go`.

- [ ] **Step 1: Write the failing test** — append to `respawn_home_test.go`:
```go
func TestResolveRespawnRoom_NewPlymouthHome(t *testing.T) {
	c := &Character{}
	c.SetSetting("home", "newplymouth")
	if got := c.ResolveRespawnRoom(); got != 5901 {
		t.Errorf("newplymouth home respawn = %d, want 5901 (Grand Temple sanctuary)", got)
	}
}
```
(Verify `SetSetting`/`GetSetting` exist on Character — they're used by `ResolveRespawnRoom`; if the test harness needs a different setter, match the existing tests in this file.)
- [ ] **Step 2: Run — expect FAIL** — `go test ./internal/characters/ -run TestResolveRespawnRoom_NewPlymouth -v` → fails (falls back to default 5209).
- [ ] **Step 3: Implement** — in `respawn_home.go`, add to both maps:
```go
// HomeLocations:
	"newplymouth": 5901,
// HomeLocationNames:
	"newplymouth": "New Plymouth (The Grand Temple)",
```
- [ ] **Step 4: Run — expect PASS** — same command → PASS; then `go test ./internal/characters/...` (no regression).
- [ ] **Step 5: Boot-verify** — full build + boot clean (5901 exists as a room from Stage A).
- [ ] **Step 6: Commit** — `feat(np-temple): Grand Temple respawn/home anchor (opt-in, room 5901)`.

## Task 7: Stage D — anchor schedules

**Files:** Create `schedules/new_plymouth_temple/np_temple_*.yaml`; Modify each anchor mob to add `schedule_id:`.

- [ ] **Step 1:** Author 24h-contiguous schedules (validators panic on gaps/unreachable; compass/`up`/`down` routing). Beats: **dawn matins** — the anchors converge on the Sanctuary 5901 for morning office; daytime at their work rooms (Alms's chapel 5910, Dross's courtyard 5916, Yelin's warden rounds about the Keeper's House, Merid's blessings 5912); evening vespers back at 5901; night rest in their cells (Holt keeps long Archive hours). Every `target_room` within the built Temple rooms. Reference `np_merchant_vell.yaml`.
- [ ] **Step 2:** Add `schedule_id:` to each anchor mob (9356–9362).
- [ ] **Step 3: Boot-verify** — `LoadSchedules` +7; no coverage-gap/unreachable panic.
- [ ] **Step 4: Commit** — `feat(np-temple): anchor schedules — matins, offices, the warden's rounds`.

## Task 8: Stage D — Ept/Dross/Holt lore breadcrumbs + Dobb transit to the gate

**Files:** Modify Ept (9359) + Dross (9360) + Holt (9362) dialogue / room nouns as needed; Modify `patrols/new_plymouth/np_docks_runner_circuit.yaml`.

- [ ] **Step 1: Lore breadcrumbs** — ensure Ept's dialogue surfaces the **orbital-symbol** doubt + the discoverable cross-reference "ask Orin in the Crafting Quarter" (avoid prefix-shadowing the trigger); Dross's dialogue surfaces the **gallery-cipher** pointer (toward the Noble gallery, later); Holt's dialogue/the Restricted-Collection door (5918) make the sealed deep-lore tantalizing + impassable. All UNRESOLVED. (Most of this is authored in Task 4 dialogue; this step confirms/tops up the cross-district pointers + room nouns.)
- [ ] **Step 2: Dobb transit** — append ONE waypoint to `np_docks_runner_circuit.yaml`: from Brun (5812, the current last vendor) east to the **temple gate plaza (5903)** as a visible transit stop (`dwell_rounds: 4`, `arrival_event: np_runner_vendor` — harmless, no shop there so it delivers nothing; or `""`). The new waypoint must be pathto-reachable across the Merchant→Temple seam. Keep depot wp0 zero-dwell. NO `CaravanServedZones` change (Temple isn't a craft vendor).
- [ ] **Step 3: Boot-verify** — patrol loads with the new waypoint; no patrol-validator panic; dialogue loads.
- [ ] **Step 4: Commit** — `feat(np-temple): Ept/Dross/Holt lore breadcrumbs + Dobb's transit to the temple gate`.

## Task 9: District harness playtest

- [ ] **Step 1:** `/playtest local feature-tester` — drive from the Merchant→Temple seam: visit each anchor, `ask <npc> about <topic>` (incl. Ept's orbital symbol + Orin cross-ref, Dross's cipher), try (and fail) to enter the Restricted Collection, **test the respawn anchor** (`set home`, confirm it reports the Grand Temple; if practical, take damage to death and confirm respawn at 5901), observe an anchor at matins/on schedule.
- [ ] **Step 2:** Triage; fix blocking/cosmetic inline (`fix(np-temple): …`); log deferred polish.
- [ ] **Step 3: Final boot test** — clean.

## Task 10: Merge to master (hold push)

- [ ] **Step 1:** `git checkout master && git merge --no-ff feature/np-temple-district -m "Merge: New Plymouth Temple Quarter (district 5/7)"`.
- [ ] **Step 2:** Update `project_new_plymouth_build.md` + `MEMORY.md`: Temple done + merged; `temple_np` now exists (its bloodline ally edge live); the Grand Temple is an opt-in respawn anchor; NEXT = District 6 Noble (the gallery cipher resolves there; Wenna/Bloom-Trail Noble beat). **Do NOT push.**

---

## Self-Review (completed during planning)

- **Spec coverage:** §1 zone/IDs/coords → Task 0 + manifests; §2 layout/entry/stubs
  → Tasks 1–2; §3 anchors (incl. Holt dual-faction) → Task 4; §4 factions (create +
  ally backfill) → Task 3; §5 respawn anchor → Task 6; §6 Dobb transit + Archive
  stub + lore → Tasks 2 (Archive door) / 4 (dialogue) / 8 (breadcrumbs + transit);
  §7 schedules → Task 7; §8 staging A–E → Tasks 1–9; §9 DoD → Tasks 9–10.
- **Placeholder scan:** room/mob bodies are subagent-authored from manifests +
  controller-solved coords (the established content pattern); the faction YAMLs,
  the respawn Go diff, and the patrol waypoint are literal. Merid/Alms optional
  shops are gated on "verify item ids exist". No TBD/TODO.
- **Consistency:** rooms 5900–5924, mobs 9356–9367, dialogue-by-mobid, faction
  `temple_np`/`bloodline_domestic`, respawn room **5901**, and the Merchant 5823→east
  seam are used identically across manifests and tasks. The respawn test (5901)
  matches the room manifest (5901 = sanctuary).
