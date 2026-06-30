# Greenford — District 3: University District — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** Build the university grounds — quad, library suite, Brennan's office, faculty — and wire the FRONT HALF of Q75 The Surveyor's Report (Brennan grant → archive survey → Brennan's intro → in-progress "go to Reth"), plus the first orbital-symbol beat (threshold-only). 11 rooms, 8 NPCs, 1 quest (half).

**Architecture:** Pure YAML in the `greenford` zone, on z=2 (the grounds). The only existing-content edit is the D2 University Stair seam (6297 up→6298). Q75 declares its full skeleton but grants only through `75-intro`; District 4 completes it (Reth + testimony + end). Extends the existing `margin` faction.

**Tech Stack:** YAML; built-binary boot; `cartcheck`; mudagent harness + `questtoken` admin.

**Spec:** `docs/superpowers/specs/2026-06-30-greenford-university-design.md` (city-wide: `2026-06-30-greenford-citywide-design.md`). Quest pattern reference: `_datafiles/world/dogmud/quests/74-the_undercroft.yaml` (split-quest: declare skeleton, grant partway, complete on `end`).

**Reserved IDs:** rooms **6298–6308**, mobs/dialogue **9516–9523**, items **40160 reserved (created in D4)**, quest **75**.

**Branch:** `feature/greenford-university` off `master`.

## Conventions (load-fatal — Greenford set + quest SOPs)
- Folder `greenford` (exists); no second zone-config.
- Title-Case names/titles; mob filename = `ConvertForFilename(name)`; ambient `noncombat_passive`.
- `idlemessages`/idle colon-space single-quoted; description/noun prose-colons in `>` blocks; **`|` literal block scalars for ALL long NPC `text:`** (≥~120 chars — bit D1 + D2).
- Exits `{roomid}` only — no `kind:`.
- Vendors: `craft_support` + `shop:`; items never `general`. (D3 may have no vendor.)
- **Dialogue node-shadowing:** gated/specific nodes FIRST; no short trigger substring-shadowing a later topic; **cross-check every hint word routes to its node**.
- **Mutation uniqueness vs the WHOLE Greenford roster** (D1 9501-9508 + D2 9509-9515), not just this batch.
- **Quest SOPs:** grant nodes need `grantsQuest` + `questExcluded` (incl. `75-end`); `questRequired`/`questExcluded` are LISTS; quest-giving nodes include `"quest"`/`"task"` triggers; **gated grant nodes FIRST** under `tree.nodes`; `room_interact` nouns are ansi-highlighted HYPHENATED tokens with matching hyphenated keys; **a trigger may only `grant` a DECLARED step** (declare all 5 steps; grant only start/survey/intro in D3); **rewards fire only on `end`** (D4). Non-vendor reward items `not_salable`.
- **Stage explicit git pathspecs, NEVER `git add -A`.** Pre-smoke: wipe instances.

---

## Task 1: 11 rooms (6298–6308) on z=2 + the 6297 seam

**Files:** Create `rooms/greenford/6298.yaml`…`6308.yaml`; Modify `rooms/greenford/6297.yaml`.

- [ ] **Step 1: Seam.** In `6297.yaml` (University Stair, `{22,-80,1}`, exit `north→6296`, has an iron-`gate` noun), add `up: {roomid: 6298}` and revise prose so the gate now admits / the way up into the grounds is open. Keep north exit + nouns.

- [ ] **Step 2: Author 6298–6308** on **z=2**. Suggested coords (builder finalizes a clean reciprocal, collision-free graph; z=2 around {22,−80} is clear):

| roomid | title | x | y | z | role |
|--------|-------|---|---|---|------|
| 6298 | The University Gate | 22 | -80 | 2 | up→6297 (down); the porter |
| 6299 | The Quadrangle | 22 | -81 | 2 | hub; scholars/debate |
| 6300 | The Lecture Hall | 21 | -81 | 2 | a lecturer/students |
| 6301 | The Library | 23 | -81 | 2 | public stacks (librarian) |
| 6302 | The Reading Room | 23 | -82 | 2 | quiet study |
| 6303 | The Archive | 23 | -83 | 2 | **Q75 investigation** (archivist; `filed-survey` + symbol nouns) |
| 6304 | The Restricted Collection | 24 | -83 | 2 | **gated stub** (locked door, lore) |
| 6305 | Brennan's Office | 21 | -82 | 2 | **Q75 giver Brennan**; the symbol on his maps |
| 6306 | The Senior Common Room | 21 | -83 | 2 | faculty; the skeptical scholar |
| 6307 | The Cloister Walk | 22 | -82 | 2 | quiet walk/garden |
| 6308 | The College Lane | 22 | -83 | 2 | **stub** to District 4 (Neighborhood) |

**Builder finalizes** a sensible reciprocal graph: 6298 gate (up↔6297) → 6299 quad (hub); the library suite 6301→6302→6303, with 6304 off 6303 (a `west`/`east` to the locked door); Brennan's office 6305 + common room 6306 off the quad/cloister; 6307 cloister connecting; 6308 the back-gate stub. INVARIANTS: 6297↔6298 reciprocal seam; every exit mirrored; coords collision-free + direction-consistent (north=+y, up=+z); **6304 Restricted Collection is a gated stub** (no onward exit beyond the locked door — described locked, not a bump); **6308 College Lane has NO onward exit** to D4 (prose only — the residential streets "just beyond," coming with D4). If coords don't yield a clean graph, adjust + note.

**Quest-relevant nouns to author now (the room_interact targets — T4 wires the triggers):**
- **6303 The Archive:** an ansi-highlighted hyphenated **`filed-survey`** noun (Reth's filed survey) — its base/ungated text reads as a dry, deliberately-empty "mineral deposit" survey; the quest trigger (T4) adds the gated reveal. Also a **symbol** beat here or in 6305 (the orbital symbol on old maps — a noun reading it as recurring + unexplained, NO numerology/explanation).
- **6305 Brennan's Office:** Brennan's **`old-maps`** (or `survey-maps`) noun carrying the orbital symbol — threshold-only, unexplained.

**Quality bar:** three-layer descriptions, ≤80 cols, ≥2 nouns, container nouns ~20%. Vary the leading sense. Signature: a quiet, old, bookish university — ink, vellum, stone, the hush of the stacks; scholarly but unhurried; the symbol present but unexplained (a question the place lives with). **THRESHOLD-ONLY on the symbol — it appears, nothing explains it.**

- [ ] **Step 3: Spawninfo** (`respawnrate: "20 real minutes"`): 6298→9522 (porter); 6299→9521 (student)+9523 (scholar); 6300→9520 (lecturer); 6301→9518 (librarian); 6303→9517 (archivist); 6305→9516 (Brennan); 6306→9519 (skeptical scholar).

- [ ] **Step 4: Self-check** — reciprocity incl. seam; no collision; no `kind:`; 6304/6308 stubs (no broken bumps — locked/described); `filed-survey` + symbol nouns present + hyphenated; titles Title-Case; colons handled; ≤80 cols; symbol threshold-only.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/rooms/greenford/629*.yaml _datafiles/world/dogmud/rooms/greenford/630*.yaml
git commit -m "feat(greenford): University District 11 rooms (6298-6308) + University Stair seam"
```

---

## Task 2: Items (minimal)

- [ ] D3 likely needs **no new items** (the archive survey is a room_interact record; the intro is a quest token; Reth's map 40160 is created in D4). OPTIONAL: one flavor book/map item — if non-vendor, `not_salable: true`. If none, skip this task (note it).
- [ ] If created, commit:
```bash
git add _datafiles/world/dogmud/items/materials-40000/4016*.yaml
git commit -m "feat(greenford): University flavor item"
```

---

## Task 3: Mobs + dialogue (9516–9523) incl. Brennan's Q75 nodes

**Files:** `mobs/greenford/9516.yaml`…`9523.yaml`; `dialogue/greenford/9516.yaml`… (all talk except pure-ambient students — give the porter/lecturer/scholars light dialogue or idle-only as fits).

Common shape: copy a Greenford vendor/dialogue mob; `non_combatant: true`, `charm_immune: true`, `hostile: false`, `archetype: noncombat_passive`, `maxwander: 0`, Title-Case name, idlecommands. Brennan + Archivist: `groups: [humanoid, margin]`; others `[humanoid]`. Each NPC: ≥3 topics, idle behaviors, a UNIQUE mutation (vs whole roster). Voice rules; `|` long text; hint-routing cross-check; NO undeclared quest fields.

| mob | name | room | notes |
|-----|------|------|-------|
| 9516 | (named) Brennan | 6305 | **margin; Q75 GIVER.** GRANT node (gated FIRST): `grantsQuest: '75-start'`, `questExcluded: ['75-start','75-end']`, triggers incl. `quest`/`task`/`surveyor`/`reth`/`hill`/`east` — sets you to find Reth's filed survey in the archive. TURN-IN node: `questRequired: ['75-survey']` → `grantsQuest: '75-intro'` (gives the introduction to Reth, "north end of the town" → D4). A `symbol`/`maps` topic = threshold-only (it's on the oldest maps, no one agrees, NO explanation, NO crash-site). Root variants gated by quest progress as needed. |
| 9517 | (named) The Archivist | 6303 | margin; helps locate Reth's filed survey; the restricted collection (gated, "by leave only"); lore-light symbol talk |
| 9518 | (named) The Librarian | 6301 | `[humanoid]`; the stacks, reading rules |
| 9519 | A Skeptical Scholar | 6306 | the doubting-scholar debate — questions the official account WITHOUT the answer (thinking, not mystery-dump) |
| 9520 | A Lecturer | 6300 | dialogue-light; the lecture, the university |
| 9521 | A Student | 6299 | ambient; scholarly life |
| 9522 | The Porter | 6298 | gate-keeper; welcomes/directs to the colleges/library |
| 9523 | A Scholar | 6307 | ambient; cloister color |

NO ONE explains the symbol or the crash site. The skeptical scholar gives doubt, not answers. Brennan's grant/turn-in must follow the Quest Dialogue SOP exactly (gated nodes first; `quest`/`task` triggers; `questExcluded` incl. `75-end`).

- [ ] **Step 1: Author.** **Step 2: Commit**
```bash
git add _datafiles/world/dogmud/mobs/greenford/951*.yaml _datafiles/world/dogmud/mobs/greenford/952*.yaml _datafiles/world/dogmud/dialogue/greenford/951*.yaml _datafiles/world/dogmud/dialogue/greenford/952*.yaml
git commit -m "feat(greenford): University NPCs + dialogue incl. Brennan Q75 (9516-9523)"
```

---

## Task 4: Quest 75 — The Surveyor's Report (declare skeleton; wire front-half triggers)

**File:** Create `_datafiles/world/dogmud/quests/75-the_surveyors_report.yaml`. **Model structure on `74-the_undercroft.yaml`.**

- [ ] **Step 1: Declare the full 5-step skeleton** (`start`, `survey`, `intro`, `testimony`, `end`) with descriptions + hints. testimony/end describe the D4 payoff (forward-declared; their TRIGGERS are wired in D4). NO `flags` block (Q75 is linear, not branching — unlike Q74). `rewards.playermessage` = the faction-neutral onward beat (fires on `end`, in D4).

- [ ] **Step 2: Wire the front-half triggers (D3 only):**
  - `room_interact` on **6303 `filed-survey`**: gated `has:['75-start'] missing:['75-survey']` → `grant: '75-survey'` + `send_text` (Reth's anomalous, deliberately-empty survey — a man who saw something and didn't write it down) + `room_text`. Plus the **ungated fallback** trigger (`missing:['75-start']`) → a dry "just a survey" text. (Mirror Q74's building-ledger pair exactly.)
  - (The `start` grant + the `intro` grant live on **Brennan's dialogue nodes** in Task 3 — NOT here. Task 4 only declares the steps + the room_interact survey trigger. testimony/end triggers + Reth are D4.)
  - OPTIONAL: a `room_interact` on the **symbol** (6305 `old-maps` / 6303) — ungated lore `send_text` (threshold-only), no grant, OR leave the symbol as a plain room noun (no quest trigger). Keep it lore, not a quest step.

- [ ] **Step 3: Self-check** — all granted tokens (`75-start` by Brennan, `75-survey` by the room_interact, `75-intro` by Brennan) are DECLARED steps; testimony/end declared but NOT granted in D3; no `flags` referenced; reward fires on `end` only. **Commit**
```bash
git add _datafiles/world/dogmud/quests/75-the_surveyors_report.yaml
git commit -m "feat(greenford): Quest 75 The Surveyor's Report (skeleton + D3 front-half triggers)"
```

---

## Task 5: Schedules (light)

- [ ] 1–2 anchor schedules (e.g. `gf_brennan` @6305, `gf_archivist` @6303) modeled on D1/D2 — 24h gap-free, in-zone target rooms, `schedule_id` wired. Brennan must stay reliably findable @6305 by day (the Q75 giver). Commit:
```bash
git add _datafiles/world/dogmud/schedules/greenford/ _datafiles/world/dogmud/mobs/greenford/9516-*.yaml _datafiles/world/dogmud/mobs/greenford/9517-*.yaml
git commit -m "feat(greenford): University anchor schedules"
```

---

## Task 6: Boot + cartcheck

- [ ] Wipe instances. Build + boot; confirm room +11, mobs +8, quests +1; **`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; **no "trigger grants unknown step" / quest-load panic**; no greenford/63xx/95xx warnings; ValidateAllFlags OK.
- [ ] `cartcheck greenford` clean (z=2 grounds + the 6297↔6298 seam). Kill. Commit fixes.

---

## Task 7: World-critic + feel polish (MANDATORY)

- [ ] **World-critic pass** — directions/geography (the town is DOWN via the stair; the university is the top; the river/Confluence far below/west); Title-Case; colons; **node-shadowing + every hint word routes**; `|` blocks for long text; **mutation uniqueness vs whole roster**; the symbol stays threshold-only (Brennan/archivist NEVER explain it; no numerology/crash-site); 6304/6308 stubs read locked/coming-soon not broken; **Quest Dialogue SOP** on Brennan (gated nodes first, `quest`/`task` triggers, `questExcluded` incl. `75-end`).
- [ ] **Feel-test (harness + `questtoken`)** — `teleport 6297`, up into the grounds; the library/archive; **Q75 front-half end-to-end**: `ask brennan quest` → grant `75-start`; examine the archive `filed-survey` (room_interact → `75-survey`); back to Brennan → `75-intro` ("go to Reth"); confirm Q75 sits **in-progress** (NOT complete — `end` is D4). Verify the symbol beat reads threshold-only; the restricted stub; every NPC's hints route. Use `questtoken`/`questtoken flags` for reliable mechanic checks (harness flaky on multi-step). Report `tools/playtest/reports/2026-06-30-local-feel-tester-greenford-university.md`.
- [ ] Fix; re-boot; commit.

---

## Task 8: Finish + docs + merge

- [ ] Final clean boot.
- [ ] **Update `docs/ZONE_EXPANSION.md`** row 19: District 3 ✅ built, **Building (3/5)**; note Q75 front-half + the symbol beat; next = District 4 (Reth + Q75 completion).
- [ ] `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch.
- [ ] **Update memory** ([[project-zone-expansion-redesign]] + MEMORY.md index): D3 built (UNPUSHED); Q75 front half live (D4 completes at Reth); the first Greenford symbol beat; next = District 4 Neighborhood (Reth + Q75 testimony/end + 40160 map item).
- [ ] Report to the user.

---

## Self-Review

**Spec coverage:** rooms + seam (T1) ✓; archive `filed-survey` + symbol nouns (T1) ✓; restricted + college-lane stubs (T1) ✓; items minimal (T2) ✓; 8 NPCs incl. Brennan giver + archivist (margin) (T3) ✓; Q75 skeleton + front-half triggers (T4) ✓; schedules (T5) ✓; boot incl. quest-load (T6) ✓; world-critic + feel incl. Q75 front-half end-to-end (T7) ✓; docs + memory + merge (T8) ✓. The split-quest design (declare skeleton, grant through intro, D4 completes) is in T4 ✓. Symbol threshold-only ✓.

**Placeholder scan:** room/dialogue/quest prose authored by implementer subagents from the per-room/per-NPC/per-step briefs (content, not logic); "(named)" markers for Brennan/Archivist/Librarian are intentional. No TBD logic; the quest skeleton's testimony/end are intentionally forward-declared (D4 wires their triggers) — documented, not a gap.

**Id/token consistency:** rooms 6298–6308 (11), mobs 9516–9523 (8), quest 75 — consistent T1/T3/T4. Q75 tokens `75-start`(Brennan,T3)/`75-survey`(6303 room_interact,T4)/`75-intro`(Brennan,T3) all DECLARED in T4's skeleton; `75-testimony`/`75-end` declared-not-granted (D4). 40160 reserved (D4). The 6297 seam is the only existing-content edit. Carries D1/D2 lessons (`|` blocks, hint routing, roster-wide mutations, stub-not-bump) + the quest SOPs.
