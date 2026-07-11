# The Confluence — District 8: East Gate & the Brennside — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** Build the East Gate & Brennside — the east bank + the public Brenn bridge + the road-stub to Greenford (12 rooms, ~6–7 NPCs, no quest) — the **final district that completes the Confluence.**

**Architecture:** Pure YAML in the `the_confluence` zone. A bridge over the Brenn from 6146 (The Three Waters) east to the east bank; a travelers' gate-quarter (gate, inn, stable, wharf, drying flats). No quest, no mystery. On merge, the Confluence is done.

**Tech Stack:** YAML; `go run .` boot; `cartcheck`; mudagent harness (walkability + the inn vendor + the gate captain; no quest).

**Spec:** `docs/superpowers/specs/2026-06-27-confluence-east-gate-design.md`

**Reserved IDs:** rooms **6246–6257**, mobs/dialogue **9484–9490**, items **40146+**, no quest.

**Branch:** `feature/confluence-east-gate` off `master`.

## Conventions (same load-fatal rules as every Confluence district)
- Mob name + room title canonical Title-Case (em-dash `—` not `--`); **mob filename = ConvertForFilename(name)**.
- idlemessages colon-space single-quoted; description/noun colons in `>` blocks.
- Exits just `roomid` (no `kind:`); bridge long-exits are auto-derived from the coord gap (no rooms in the span → cartcheck-clean).
- Vendor `craft_support` (+ `shop:`); items carry a real discipline, never `general`.
- No quest fields anywhere; no pre-Founding mystery lore.
- Pre-smoke SOP: wipe instances before boot.

---

## Task 1: The 12 rooms (6246–6257) + seam

**Files:** Modify `rooms/the_confluence/6146.yaml` (add east exit); Create 6246–6257.

- [ ] **Step 1: Seam.** In `6146.yaml` (The Three Waters, `{-4,-67,0}`, free EAST), add `east: {roomid: 6246}`; extend its prose so a door/lane east leads to the Brenn bridge and the east bank (keep the inn's voice).

- [ ] **Step 2: Author 6246–6257** per this table (authoritative; reciprocal; no `kind:`; biome `city`):

| roomid | title | x | y | z | exits |
|--------|-------|---|---|---|-------|
| 6246 | The Brenn Bridge, West End | -3 | -67 | 0 | west→6146, east→6247 |
| 6247 | The Brenn Bridge, Crown | 2 | -67 | 0 | west→6246, east→6248 |
| 6248 | The Brenn Bridge, East End | 8 | -67 | 0 | west→6247, east→6249 |
| 6249 | The East Gate | 9 | -67 | 0 | west→6248, east→6250, north→6251, south→6252 |
| 6250 | The Greenford Road | 10 | -67 | 0 | west→6249 |
| 6251 | The Brennside Wharf | 9 | -66 | 0 | south→6249, east→6253 |
| 6253 | The Eastern Landing | 10 | -66 | 0 | west→6251 |
| 6252 | The Travelers' Inn | 9 | -68 | 0 | north→6249, south→6254, east→6255 |
| 6255 | The Stable | 10 | -68 | 0 | west→6252 |
| 6254 | The Inn Common Room | 9 | -69 | 0 | north→6252, east→6256 |
| 6256 | The Drying Flats | 10 | -69 | 0 | west→6254, south→6257 |
| 6257 | The Eastern Edge | 10 | -70 | 0 | north→6256 |

(Reciprocals: 6146E↔6246W, 6246E↔6247W, 6247E↔6248W, 6248E↔6249W, 6249E↔6250W, 6249N↔6251S, 6249S↔6252N, 6251E↔6253W, 6252S↔6254N, 6252E↔6255W, 6254E↔6256W, 6256S↔6257N.) The bridge exits 6246E (5-cell), 6247W+E (5/6-cell), 6248W auto-classify `long` over open water.

**6250 Greenford stub:** do NOT wire an east exit. Its PROSE describes the paved road running east out of the city toward Greenford — a journey for another day (a described stub, like River Road 6105). A `milestone`/`gate-arch` noun noting the road/distance is good (the outward gesture).

**6257 The Eastern Edge:** the city's outward terminus — paving giving way to scrub, the river-country opening east; a quiet closing beat for the whole tri-city. North back to the flats only.

**Spawninfo:** 6249→9484 (Gate Captain) + 9485 (Gate Guard); 6252→9486 (Innkeeper) + 9488 (Wayfarer); 6255→9487 (Stablehand); 6251→9489 (Brennside Fisher); (6254 or 6256 → 9490 ambient if used). `respawnrate: "20 real minutes"`.

**Bridge feel:** the crossing over the Brenn is a real moment (the eastern river broad below; the city behind, the country ahead). 2–3 idlemessages per room (quote colons).

- [ ] **Step 3: Self-check** — exits reciprocal (incl. seam); no coord collision; no `kind:`; 6250 no east exit; titles Title-Case; colons handled; ≤80 cols; no mystery lore.
- [ ] **Step 4: Commit**
```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6146.yaml _datafiles/world/dogmud/rooms/the_confluence/624*.yaml _datafiles/world/dogmud/rooms/the_confluence/625*.yaml
git commit -m "feat(confluence): East Gate & the Brennside, 12 rooms (6246-6257)"
```

---

## Task 2: Items (40146+)

- [ ] Create ≤2 flavor goods (reuse first). E.g. `40146-a_bowl_of_travelers_stew.yaml` (`vendor_categories: [cooking]`), model on 40130. The innkeeper also reuses 40135/40136. Commit:
```bash
git add _datafiles/world/dogmud/items/materials-40000/4014*.yaml
git commit -m "feat(confluence): East Gate traveler goods (40146+)"
```

---

## Task 3: Mobs + dialogue (9484–9490)

**Files:** mob+dialogue per NPC (vendors + the gate captain get dialogue; ambient idlecommands-only or short).

Common shape (copy `9436-bremm_the_innkeeper.yaml` for the innkeeper, `9419-holt_the_gate_warden.yaml` for the gate captain, `9440-a_confluence_citizen.yaml` ambient): `non_combatant: true`, `charm_immune: true`, `hostile: false`, `statpool: 30`, `maxwander: 0`, `activitylevel: 10`, Title-Case name, groups, idlecommands.

| mobid | name | filename | room | type |
|-------|------|----------|------|------|
| 9484 | The Gate Captain | 9484-the_gate_captain.yaml | 6249 | dialogue (outward voice; the Greenford road; the city's edge) |
| 9485 | A Gate Guard | 9485-a_gate_guard.yaml | 6249 | ambient |
| 9486 | The Innkeeper | 9486-the_innkeeper.yaml | 6252 | vendor `cooking` (40146 + 40135/40136) |
| 9487 | The Stablehand | 9487-the_stablehand.yaml | 6255 | ambient |
| 9488 | A Wayfarer | 9488-a_wayfarer.yaml | 6252 | dialogue (bound to/from Greenford — the outward nod, lore-light, NO mystery) |
| 9489 | A Brennside Fisher | 9489-a_brennside_fisher.yaml | 6251 | ambient (groups: humanoid, quayfolk) |

(9490 optional drover/child if a 7th adds life.) Gate Captain/guards may carry `road_wardens` group if it reads right; otherwise `[humanoid]`. **No quest fields. No pre-Founding mystery.** The Wayfarer is the one outward gesture (the world is bigger; Greenford east) — never names any mystery answer.

- [ ] Commit:
```bash
git add _datafiles/world/dogmud/mobs/the_confluence/948*.yaml _datafiles/world/dogmud/mobs/the_confluence/949*.yaml _datafiles/world/dogmud/dialogue/the_confluence/948*.yaml _datafiles/world/dogmud/dialogue/the_confluence/949*.yaml
git commit -m "feat(confluence): East Gate NPCs + dialogue (9484-9490)"
```

---

## Task 4: Schedules
- [ ] 2–3 anchor schedules (cf_innkeeper @6252 day+evening; cf_gate_captain @6249 day; cf_stablehand @6255) + wire `schedule_id`. Model on cf_historian. Commit.

## Task 5: Boot + cartcheck
- [ ] Wipe instances; build + boot; confirm `ValidateZoneConsistency errors=0 warnings=0 mode=panic` (the bridge long-exits clean), clean loads, no casing/colon/category panics, no 62xx/948x warnings; `cartcheck the_confluence` clean. Kill. Commit fixes.

## Task 6: World-critic + feel polish (MANDATORY)
- [ ] World-critic: directions vs the bridge/east-bank layout; Title-Case; colons; vendor categories; **no mystery lore**; 6250 stub reads as "road out, not yet" not broken; the bridge reads as a real crossing.
- [ ] Feel-walk: the seam (6146→6246), the bridge crossing, the gate + Greenford stub, the inn (buy), the gate captain + wayfarer dialogue, the Eastern Edge closing beat. Report to `tools/playtest/reports/2026-06-27-local-feel-tester-confluence-east-gate.md`.
- [ ] Fix; re-boot; commit.

## Task 7: Finish + reviewers + CITY COMPLETE
- [ ] Final clean boot; `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch.
- [ ] Update memory: **THE CONFLUENCE IS COMPLETE** (all 8 districts + 5a/5b/6a/6b; ~150 rooms; Q74 climax; the whole bundle ready for the user's prod push). Update the MEMORY.md index + `project_confluence_build.md`.
- [ ] Fire the district-8 background reviewers (adversarial + feel). Report to the user that the Confluence is done and the bundle awaits their droplet push.

## Self-review
- Spec coverage: §2 seam+bridge → T1; §3 rooms → T1; §4 NPCs → T3; §5 items → T2; §7 schedules → T4; §8 verify → T5/T6. Covered.
- Placeholders: none (room table exact; NPC roster fixed; prose delegated with briefs).
- Consistency: room ids/coords/exits reciprocal (T1); spawninfo ids (9484–9490) match T3; vendor item 40146 matches T2; 6250 no-east-exit consistent (table + stub note); no quest anywhere.
