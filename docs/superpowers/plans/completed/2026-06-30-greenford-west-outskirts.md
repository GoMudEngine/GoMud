# Greenford — District 5: West Outskirts — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** Build the last Greenford district — the road leaving town toward New Plymouth (the West Gate, the West Road, a coaching stable, a farewell shrine, the NP-road terminus stub). 6 rooms, 5 NPCs, no quest. On merge, **Greenford is complete (5/5)**.

**Architecture:** Pure YAML in the `greenford` zone, z=1, west of the town center. The only existing-content edit is opening a west exit on the town's west edge. Lore-and-ambient terminus; no quest, no symbol content.

**Tech Stack:** YAML; built-binary boot; `cartcheck`; mudagent harness.

**Spec:** `docs/superpowers/specs/2026-06-30-greenford-west-outskirts-design.md` (city-wide: `…greenford-citywide-design.md`).

**Reserved IDs:** rooms **6317–6322**, mobs/dialogue **9530–9534**, items **40162** (optional). No quest, no faction, no buffs.

**Branch:** `feature/greenford-west-outskirts` off `master` (AFTER D4 merges).

## Conventions (Greenford set — same as D1-D4)
- Folder `greenford`; no second zone-config. Title-Case; mob filename = `ConvertForFilename(name)`; ambient `noncombat_passive`.
- `idlemessages`/idle colon-space single-quoted; description/noun prose-colons in `>` blocks; **`|` literal block scalars for ALL long NPC `text:`**.
- Exits `{roomid}` only — no `kind:`. Vendors `craft_support`+`shop:`; items never `general`.
- Dialogue: gated/specific nodes first; **every hint word routes** (cross-check); no short-trigger shadowing.
- **Mutation uniqueness vs the WHOLE Greenford roster** (D1+D2+D3+D4 = 27 mutations).
- **No symbol/crash/mystery content anywhere** (the symbol is D3's; this is the mundane close).
- **Stage explicit git pathspecs, NEVER `git add -A`.** Pre-smoke: wipe instances.

---

## Task 1: 6 rooms (6317–6322) + the west-edge seam

**Files:** Create `rooms/greenford/6317.yaml`…`6322.yaml`; Modify ONE town-center room (the chosen west-edge attach).

- [ ] **Step 1: Pick + open the seam.** READ the D2 town-center rooms' exits to pick a clean FREE WEST exit on the town's west side. Candidates: **6295 "The Town Hall Steps"** (`{22,-78,1}`, west free — a civic steps onto a street leaving town, reads well) or **6293 "The General Store"** (`{21,-77,1}`, west free). Pick the most natural (prefer 6295 unless 6293 reads better). Add a `west: {roomid: 6317}` exit to that room and revise its prose lightly so a street/lane now leads WEST out of town toward the outskirts. (Note which room you used.)

- [ ] **Step 2: Author 6317–6322** on **z=1**, WEST of the attach (x ≤ 20; clear of D2 [x 21–24, z=1] and D1 [z=0]). Suggested layout (builder finalizes a clean reciprocal, collision-free graph; adjust coords to fit the chosen attach room):

| roomid | title | role |
|--------|-------|------|
| 6317 | The West Gate | from the attach (east back into town); a road-warden |
| 6318 | The West Road | leaving town, country opening; a departing traveler |
| 6319 | The Coaching Stable | the ostler's yard (optional vendor) |
| 6320 | The Wayfarer's Shrine | a farewell waypoint shrine (MUNDANE — NOT an orbital marker) |
| 6321 | The Milepost | the open road; a milestone naming New Plymouth (days west/NW) |
| 6322 | The Plymouth Road | **terminus stub** — road bends toward NP, NOT passable; described "another journey, another day"; the closing beat for all of Greenford |

**INVARIANTS:** the seam (attach-room west ↔ 6317 east) reciprocal; every exit reciprocal; coords collision-free + direction-consistent (west=−x); **6322 has NO onward exit** (the NP road is prose-only — model the East Road barred-bridge / River Road barred-gate terminus, a clean "another journey," not a broken bump); the stable/shrine hang off the West Road sensibly. Adjust coords if needed; note changes.

**Quality bar:** three-layer descriptions, ≤80 cols, ≥2 nouns, ~20% container nouns. Vary the leading sense. Signature: the quiet outward edge — the town thinning behind, the country opening west, the road-dust and stable-smell, a shrine for travelers, the milestone to NP. **A satisfying CLOSE to the whole city** (the player has walked all of Greenford). **NO symbol/crash content** — the shrine is a mundane traveler's blessing-post.

- [ ] **Step 3: Spawninfo** (`respawnrate: "20 real minutes"`): 6317→9530 (road-warden)+9534 (resident); 6318→9532 (departing traveler); 6319→9531 (ostler); 6320→9533 (shrine-keeper).

- [ ] **Step 4: Self-check** — reciprocity incl. seam; no collision; no `kind:`; 6322 no onward exit (clean stub); titles Title-Case; colons handled; ≤80 cols; NO symbol/crash content; the shrine is mundane.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/rooms/greenford/631*.yaml _datafiles/world/dogmud/rooms/greenford/632*.yaml _datafiles/world/dogmud/rooms/greenford/<attach-room>.yaml
git commit -m "feat(greenford): West Outskirts 6 rooms (6317-6322) + west-edge seam"
```

---

## Task 2: Items (optional 40162)

- [ ] OPTIONAL: one travel good at the stable (trail rations / feed), `vendor_categories: [cooking]` (never `general`), modeled on a D2/D4 cooking good. If the ostler reuses an existing good, SKIP (note it).
- [ ] If created, commit:
```bash
git add _datafiles/world/dogmud/items/materials-40000/40162-*.yaml
git commit -m "feat(greenford): West Outskirts travel good (40162)"
```

---

## Task 3: Mobs + dialogue (9530–9534)

**Files:** `mobs/greenford/9530.yaml`…`9534.yaml`; `dialogue/greenford/9530.yaml`… (road-warden/ostler/traveler/shrine-keeper talk; resident ambient).

Common shape: copy a Greenford mob. `non_combatant`/`charm_immune`/`hostile:false`/`maxwander:0`/`noncombat_passive` (ostler = shopkeeper archetype if vendor); Title-Case; `groups: [humanoid]`. Each NPC: ≥3 topics, idle behaviors, a UNIQUE mutation (vs the 27-roster). Voice rules; `|` long text; hint-routing cross-check; NO quest fields; NO symbol/crash content.

| mob | name | room | notes |
|-----|------|------|-------|
| 9530 | (named) The Road-Warden | 6317 | town's west edge; directs travelers; the road, the long way to NP (soft outward gesture, NO mystery) |
| 9531 | (named) The Ostler | 6319 | the coaching stable; coach/travel talk; **optional `cooking` vendor** (40162 + reuse) |
| 9532 | A Departing Traveler | 6318 | bound west/NP — the outward gesture; the long road, the wider world (lore-light; NO crash/symbol) |
| 9533 | (named) The Shrine-Keeper | 6320 | the farewell shrine; the traveler's road-blessing; gentle MUNDANE faith (NO orbital symbol) |
| 9534 | A Resident | 6317 | ambient; edge-of-town daily life |

The Road-Warden + Departing Traveler carry the "world goes on west toward NP" gesture (Aldric's route; the loop the world closes later). Everything mundane.

- [ ] Commit:
```bash
git add _datafiles/world/dogmud/mobs/greenford/953*.yaml _datafiles/world/dogmud/dialogue/greenford/953*.yaml
git commit -m "feat(greenford): West Outskirts NPCs + dialogue (9530-9534)"
```

---

## Task 4: Schedules (light)

- [ ] 1 anchor schedule (e.g. `gf_ostler` @6319 or `gf_roadwarden` @6317) modeled on D1-D4 — 24h gap-free, in-zone target room, `schedule_id` wired. Commit:
```bash
git add _datafiles/world/dogmud/schedules/greenford/ _datafiles/world/dogmud/mobs/greenford/<wired-mob>.yaml
git commit -m "feat(greenford): West Outskirts anchor schedule"
```

---

## Task 5: Boot + cartcheck

- [ ] Wipe instances. Build + boot; confirm room +6, mobs +5; **`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; no panics; no greenford/63xx/953x warnings.
- [ ] `cartcheck greenford` clean (the west seam + D5). Kill. Commit fixes.

---

## Task 6: World-critic + feel polish (MANDATORY)

- [ ] **World-critic pass** — directions/geography (the town is EAST/back; the road runs west toward NP; the river/university not misplaced); Title-Case; colons; node-shadowing + hint routing; `|` blocks; mutation uniqueness vs whole roster; **NO symbol/crash content** (the shrine stays mundane); 6322 reads as "another journey" not broken.
- [ ] **Feel-test (harness)** — from the town center, take the new WEST exit into the outskirts; the road, the stable (`list`/`buy` if vendor), the shrine, the milepost, the NP-road terminus stub; all NPC hints route; confirm NO symbol leak + a satisfying close. Report `tools/playtest/reports/2026-06-30-local-feel-tester-greenford-westoutskirts.md`.
- [ ] Fix; re-boot; commit.

---

## Task 7: Finish + docs + merge — GREENFORD COMPLETE

- [ ] Final clean boot (confirm full Greenford: ~45 rooms 6278-6322, Q75 end-to-end intact).
- [ ] **Update `docs/ZONE_EXPANSION.md`** row 19: status **✅ Built (5/5)** — Greenford COMPLETE (the Eastern Arc's approach city done; Q75 The Surveyor's Report end-to-end). Refresh the TOTAL row (zone/room counts; next = Cascade Pass → Eastern Highlands → Crash Site). Commit.
- [ ] `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch.
- [ ] **Update memory** ([[project-zone-expansion-redesign]] + MEMORY.md index): **GREENFORD ✅ COMPLETE (5/5, 45rm 6278-6322; Q75 done end-to-end)**; the Eastern Arc's full approach city is built; next Eastern Arc = Cascade Pass Road (#20) → Eastern Highlands (#21) → Crash Site (#22, the moon-crash payoff). NP Sewers (#13.5) standalone fill-in remains.
- [ ] Report to the user — **Greenford done.**

---

## Self-Review

**Spec coverage:** rooms + west seam (T1) ✓; mundane shrine + NP-road terminus stub (T1) ✓; optional travel item (T2) ✓; 5 NPCs (T3) ✓; schedule (T4) ✓; boot/cartcheck (T5) ✓; world-critic + feel (T6) ✓; finish + **Greenford-complete docs/memory** + merge (T7) ✓. No quest/faction/symbol — matches spec (the mundane close) ✓.

**Placeholder scan:** room/dialogue prose authored by implementer subagents from briefs; "(named)" = builder-named NPCs; the attach-room choice is an explicit build decision (read D2, pick the cleanest west exit). No TBD logic.

**Id consistency:** rooms 6317–6322 (6), mobs 9530–9534 (5), item 40162 (optional) — consistent T1/T2/T3. z=1, x≤20, west of D2 (collision-free). The west-edge seam is the only existing-content edit. Carries all D1-D4 lessons. On merge: Greenford 5/5, 45 rooms, complete.
