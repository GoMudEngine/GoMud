# The Confluence — District 7: Craftsmen's Row & Residential — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build the Craftsmen's Row & Residential quarter — the lived-in working SW of the Confluence (14 rooms, ~11 texture NPCs, no quest).

**Architecture:** Pure YAML content in the existing `the_confluence` zone, attached SW off the Scholars' Quarter (6231 west → 6232). A craft row + daily market + residential lanes. No quest, no mystery — everyday city texture, with two named-character anchors (the riverman's wife, the retired functionary).

**Tech Stack:** YAML data files; `go run .` boot; `cartcheck`/`ValidateZoneConsistency`; mudagent harness (no quest to verify — walkability + vendors + the two anchors).

**Spec:** `docs/superpowers/specs/2026-06-27-confluence-craftsmens-row-design.md`

**Reserved IDs (verified clean):** rooms **6232–6245**, mobs/dialogue **9472–9483**, items **40143+**, no quest/buffs.

**Branch:** create `feature/confluence-craftsmens-row` off `master` before Task 1.

## Authoring conventions (load-fatal rules — same as every Confluence district)
- Mob `character.name` AND room `title` canonical Title-Case (em-dash `—` not `--`); **mob filename must match `ConvertForFilename(name)`** (the 6b ward-construct panic).
- `idlemessages` with a colon-space MUST be single-quoted (the 6b 6211 panic); description/noun colons in `>` block scalars.
- Exits are just `roomid` — NO `kind:` field.
- Highlighted nouns: `<ansi fg="itemname">token</ansi>` + matching `nouns:` key; multi-word keys hyphenated.
- Shops: vendor `craft_support` (may be `general` for a mixed stall); **items themselves carry a real discipline, never `general`**; vendor lists items in `shop:`.
- 40xxx items live in `items/materials-40000/`.
- Pre-smoke SOP: wipe `mobs.instances/*` + `rooms.instances/*` before every boot.
- Lore: **NONE of the pre-Founding mystery** — everyday city only.

---

## Task 1: The 14 rooms (6232–6245) + seam

**Files:** Create `rooms/the_confluence/6232.yaml` … `6245.yaml`; Modify `6231.yaml` (add the west exit).

- [ ] **Step 1: Wire the seam.** In `6231.yaml` (The Garden Walk, `{-12,-68,0}`, currently `east→6229`), add `west: {roomid: 6232}`. Lightly extend its prose so the outer wall now has a gate to the working quarter (keep the Scholars'-Quarter voice).

- [ ] **Step 2: Author the 14 rooms** per this coord/exit table (authoritative; cartcheck-verified; every exit reciprocal; no `kind:`; biome `city`):

| roomid | title | x | y | z | exits |
|--------|-------|---|---|---|-------|
| 6232 | The Artisans' Gate | -13 | -68 | 0 | east→6231, west→6233 |
| 6233 | Upper Craft Row | -14 | -68 | 0 | east→6232, west→6234, south→6236 |
| 6234 | The Cooperage | -15 | -68 | 0 | east→6233, south→6235 |
| 6235 | The Weaver's | -15 | -69 | 0 | north→6234, east→6236 |
| 6236 | The Potter's | -14 | -69 | 0 | north→6233, west→6235, south→6237 |
| 6237 | The Daily Market | -14 | -70 | 0 | north→6236, west→6238, south→6239 |
| 6238 | The Baker's | -15 | -70 | 0 | east→6237 |
| 6239 | The Smithy | -14 | -71 | 0 | north→6237, east→6240, south→6241 |
| 6240 | The Craft-Supply Stall | -13 | -71 | 0 | west→6239 |
| 6241 | Craftsmen's Lane | -14 | -72 | 0 | north→6239, west→6243, south→6242 |
| 6243 | The Riverman's House | -15 | -72 | 0 | east→6241 |
| 6242 | A Tenement Court | -14 | -73 | 0 | north→6241, east→6244, south→6245 |
| 6244 | The Functionary's Door | -13 | -73 | 0 | west→6242 |
| 6245 | The Well-Court | -14 | -74 | 0 | north→6242 |

**Spawninfo:** 6234→9473 (cooper), 6235→9475 (weaver), 6236→9474 (potter), 6237→9482 (journeyman/ambient), 6238→9472 (baker), 6239→9476 (smith), 6240→9477 (craft-supply seller), 6242→9480 (washerwoman), 6243→9478 (riverman's wife), 6244→9479 (retired functionary), 6245→9481 (old man at the well). `respawnrate: "20 real minutes"`.

**Flavor:** craft shops smell/sound of their trade; the market is the lively hub; the baker's is warm; the residential lanes have washing-lines, children, doorstep gossip. 2–3 idlemessages each (quote colons). A flavor noun or two where natural (e.g. the well, a craft display) — no quest nouns.

- [ ] **Step 3: Self-check** — every exit reciprocal (walk the table); no coord collision; no `kind:`; titles Title-Case; colons handled; ≤80 cols; no pre-Founding lore.

- [ ] **Step 4: Commit**
```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6231.yaml _datafiles/world/dogmud/rooms/the_confluence/623*.yaml _datafiles/world/dogmud/rooms/the_confluence/624*.yaml
git commit -m "feat(confluence): Craftsmen's Row & Residential, 14 rooms (6232-6245)"
```

---

## Task 2: Items (40143+)

**Files:** Create up to ~4 flavor goods in `items/materials-40000/`. Reuse first (40135 broth, 40136 black bread, NP goods) — only create what vendors genuinely need.

- [ ] **Step 1:** Create the new goods, e.g.:
  - `40143-a_fruit_pastry.yaml` (baker; `vendor_categories: [cooking]`)
  - `40144-a_fired_clay_pot.yaml` (potter/craft-supply; a real discipline, e.g. `[cooking]` or `[alchemy]` as a vessel)
  - `40145-a_bolt_of_homespun.yaml` (weaver; `vendor_categories: [tailoring]`)
  Model on `items/materials-40000/40130-river_spice.yaml`. Filenames keep leading article. `is_component: true`, small value/weight.
- [ ] **Step 2: Commit**
```bash
git add _datafiles/world/dogmud/items/materials-40000/4014*.yaml
git commit -m "feat(confluence): Craftsmen's Row flavor goods (40143+)"
```

---

## Task 3: Mobs + dialogue (9472–9483)

**Files:** Create paired mob+dialogue for the vendors + the two named anchors; mob-only (idlecommands) for pure ambient.

Common mob shape (copy from `mobs/the_confluence/9437-corliss_the_shopkeeper.yaml` for a vendor, `9440-a_confluence_citizen.yaml` for ambient): `non_combatant: true`, `charm_immune: true`, `hostile: false`, `statpool: 30`, `maxwander: 0`, `activitylevel: 10`, `groups: [humanoid]` (residents unfactioned), Title-Case name, 4–6 idlecommands.

| mobid | name | room | vendor? |
|-------|------|------|---------|
| 9472 | The Baker | 6238 | YES (`cooking`) |
| 9473 | The Cooper | 6234 | YES |
| 9474 | The Potter | 6236 | YES |
| 9475 | The Weaver | 6235 | ambient (or vendor) |
| 9476 | The Smith | 6239 | ambient |
| 9477 | The Craft-Supply Seller | 6240 | YES (`general` craft_support) |
| 9478 | The Riverman's Wife | 6243 | no (named anchor, full dialogue) |
| 9479 | The Retired Functionary | 6244 | no (named anchor, full dialogue) |
| 9480 | A Washerwoman | 6242 | ambient |
| 9481 | An Old Man | 6245 | ambient |
| 9482 | A Journeyman | 6237 | ambient |
| 9483 | A Confluence Child | 6237/6242 | ambient (Title-Case "A Confluence Child") |

- [ ] **Step 1: Vendors (9472/9473/9474/9477, + 9475 if vendor).** `craft_support` + `shop:` list of their goods (new 40143-45 + reused 40136 etc.). Each item a real discipline.
- [ ] **Step 2: The Riverman's Wife (9478) dialogue.** A real character: her husband poles the river (Quayfolk tie); sharp, a little lonely, funny; daily-life voice. May have a wry word about the temple/scholars — NO pre-Founding lore. Give her a Chrysalis-change touch if natural.
- [ ] **Step 3: The Retired Functionary (9479) dialogue.** An old pensioned civic clerk; remembers the city's *ordinary* history and civic gossip (NOT the mystery); opinionated, comic-poignant.
- [ ] **Step 4: Ambient (9475/9476/9480/9481/9482/9483).** Mob-only or short dialogue; atmospheric idlecommands of their trade/life. "A Confluence Child" Title-Case.
- [ ] **Step 5: Self-check** — names Title-Case; filenames match ConvertForFilename; no quest fields anywhere; vendor shop itemids exist; ≤80 cols; no pre-Founding lore.
- [ ] **Step 6: Commit**
```bash
git add _datafiles/world/dogmud/mobs/the_confluence/947*.yaml _datafiles/world/dogmud/mobs/the_confluence/948*.yaml _datafiles/world/dogmud/dialogue/the_confluence/947*.yaml _datafiles/world/dogmud/dialogue/the_confluence/948*.yaml
git commit -m "feat(confluence): Craftsmen's Row NPCs + dialogue (9472-9483)"
```

---

## Task 4: Anchor schedules

**Files:** Create 2–3 schedules in `schedules/the_confluence/` + wire `schedule_id` into those mobs. Model on `cf_historian.yaml`. Findability-preserving, 24h coverage.
- [ ] **Step 1:** e.g. `cf_baker` (9472 — up before dawn baking, shop by day, sleeps), `cf_riverman_wife` (9478 — home/market by day), `cf_smith` (9476 — at the forge by day). Wire `schedule_id`.
- [ ] **Step 2: Commit**
```bash
git add _datafiles/world/dogmud/schedules/the_confluence/cf_*.yaml _datafiles/world/dogmud/mobs/the_confluence/9472*.yaml _datafiles/world/dogmud/mobs/the_confluence/9476*.yaml _datafiles/world/dogmud/mobs/the_confluence/9478*.yaml
git commit -m "feat(confluence): Craftsmen's Row anchor schedules"
```

---

## Task 5: Boot + cartcheck
- [ ] Wipe instances; `go build ./... && go run .`; confirm `ValidateZoneConsistency errors=0 warnings=0 mode=panic`, clean loads, no casing/colon/category panics, no 62xx/947x warnings. `cartcheck the_confluence` clean. Kill server. Commit any fixes.

---

## Task 6: World-critic + feel polish (MANDATORY)
- [ ] **World-critic** over the 14 rooms + dialogue: directions vs the seam/coord layout; Title-Case (rooms+mobs); colon quoting; hyphenated nouns; no `kind:`; vendor category validity; **confirm zero pre-Founding lore leaked in** (this quarter is the mundane city).
- [ ] **Feel-tester harness walk:** the seam (6231↔6232), the craft row + market + baker (buy something), the two named anchors' dialogue, residential walkability. Report to `tools/playtest/reports/2026-06-27-local-feel-tester-confluence-craftsmens-row.md`.
- [ ] Fix everything; re-boot; commit.

---

## Task 7: Finish + reviewers
- [ ] Final clean boot; `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch (no prod push).
- [ ] Update memory (district 7 done; only East Gate & Brennside remains).
- [ ] Fire the district-7 background reviewers (adversarial + feel), then build the **last district: East Gate & Brennside** (6246–6257, Brenn bridge off 6146 The Three Waters east → east bank x+12).

## Self-review (plan author)
- Spec coverage: §2 seam → T1; §3 rooms → T1; §4 NPCs → T3; §5 items → T2; §7 schedules → T4; §8 verification → T5/T6. Covered.
- Placeholders: none (room table exact; NPC roster + vendor flags fixed; prose delegated with briefs).
- Consistency: room ids/coords/exits reciprocal in the T1 table; spawninfo mob ids (9472–9483) match T3; vendor shop items (40143–45 + reuse) match T2; no quest anywhere (texture district).
