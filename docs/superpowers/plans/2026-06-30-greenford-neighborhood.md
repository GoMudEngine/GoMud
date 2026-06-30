# Greenford — District 4: Brennan's & Reth's Neighborhood — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** Build the residential quarter behind the college and COMPLETE Quest 75 — with Brennan's introduction, the retired surveyor Reth gives his testimony (directions east + "it's not natural") + the marked map (40160) + margin rep, closing the split quest. 8 rooms, 6 NPCs, the Q75 back half. No supporting quest (user's call: keep D4 focused on the payoff).

**Architecture:** Pure YAML in the `greenford` zone, z=2, south of the college (off D3's 6308 back gate). Wires Quest 75's forward-declared `testimony`/`end` steps (declared in D3's `75-the_surveyors_report.yaml`). The only existing-content edit is the 6308 seam (south→6309). Reth completes Q75 via a `room_interact` on his field-notes (Q74 reveal+completion pattern).

**Tech Stack:** YAML; built-binary boot; `cartcheck`; mudagent harness + `questtoken`.

**Spec:** `docs/superpowers/specs/2026-06-30-greenford-neighborhood-design.md` (city-wide: `…greenford-citywide-design.md`). Quest pattern: `_datafiles/world/dogmud/quests/74-the_undercroft.yaml` (room_interact reveal grants + give_item + bump_rep). Q75 YAML: `_datafiles/world/dogmud/quests/75-the_surveyors_report.yaml` (steps declared; D4 wires testimony/end triggers).

**Reserved IDs:** rooms **6309–6316**, mobs/dialogue **9524–9529**, items **40160** (Reth's map) + optional **40161** (tea good). Quest **75** (back half).

**Branch:** `feature/greenford-neighborhood` off `master`.

## CRITICAL continuity — read Brennan's actual directions FIRST
Brennan's D3 dialogue (`dialogue/greenford/9516.yaml`, the intro/turn-in node that grants `75-intro`) gives the player SPECIFIC directions to Reth — observed in feel-testing as "first left, third house with the overgrown yew." **Before authoring rooms, READ that node's exact text** and lay out D4 (the route from the back gate to Reth's cottage) + Reth's cottage (6315) to MATCH it — the overgrown yew, the "first left," the "third house." If Brennan's text and the layout can't be reconciled, lightly reword Brennan's directions to match the built layout (it's the same merged zone). The player must be able to follow Brennan's words to Reth.

## Conventions (Greenford set + quest SOPs — same as D3)
- Folder `greenford`; no second zone-config. Title-Case; mob filename = `ConvertForFilename(name)`; ambient `noncombat_passive`.
- `idlemessages`/idle colon-space single-quoted; description/noun prose-colons in `>` blocks; **`|` literal block scalars for ALL long NPC `text:`** (bit D1/D2).
- Exits `{roomid}` only — no `kind:`. Vendors `craft_support`+`shop:`; items never `general`; reward item `not_salable`.
- Dialogue: gated nodes FIRST; no short trigger substring-shadowing a later topic; **every hint word routes**.
- **Mutation uniqueness vs the WHOLE Greenford roster** (D1+D2+D3 = 21 mutations).
- **Quest SOPs:** `room_interact` noun = ansi-highlighted HYPHENATED token + matching key; a trigger may only `grant` a DECLARED step (75-testimony, 75-end declared in D3); Reth's gated dialogue nodes FIRST; `questRequired` is a LIST; **rewards fire on `end`** (the onward playermessage already in the Q75 YAML).
- **Stage explicit git pathspecs, NEVER `git add -A`.** Pre-smoke: wipe instances.

---

## Task 1: 8 rooms (6309–6316) + the 6308 seam

**Files:** Create `rooms/greenford/6309.yaml`…`6316.yaml`; Modify `rooms/greenford/6308.yaml`.

- [ ] **Step 0:** READ `dialogue/greenford/9516.yaml` (Brennan) — extract his EXACT directions to Reth (the yew, "first left," "third house"). Lay out D4 to honor them.

- [ ] **Step 1: Seam.** In `6308.yaml` (College Lane, `{22,-83,2}`, exits `north→6307, west→6306`, `back gate` noun), add `south: {roomid: 6309}` and revise the prose so the back gate now leads onward into the residential streets (open/walkable — undo the D3 "coming-soon" softening).

- [ ] **Step 2: Author 6309–6316** on **z=2**, south of 6308 (y ≤ −84; collision-free vs all prior — D3 ends y=−83). Suggested layout (builder finalizes a clean reciprocal, collision-free graph honoring Brennan's directions):

| roomid | title | x | y | z | role |
|--------|-------|---|---|---|------|
| 6309 | Back Gate Lane | 22 | -84 | 2 | from 6308 (north); the streets begin |
| 6310 | College Row | 22 | -85 | 2 | residential street; the "first left" toward Reth branches here |
| 6311 | The Blue Door | 21 | -85 | 2 | **Brennan's house** — lore (no NPC) |
| 6312 | The Tea House | 23 | -85 | 2 | tea-house keeper (cooking vendor) |
| 6313 | The Walled Garden | 23 | -86 | 2 | green/quiet court |
| 6314 | The Quiet End | 21 | -86 | 2 | the lane toward Reth's (residents thin) |
| 6315 | Reth's Cottage | 21 | -87 | 2 | **Q75 PAYOFF — Reth + the field-notes room_interact**; the overgrown yew |
| 6316 | The Garden Wall | 22 | -86 | 2 | a quiet terminus (neighborhood's far edge) |

**Builder finalizes** a reciprocal graph that makes Brennan's "first left, third house with the yew" walkable to 6315. INVARIANTS: 6308 south↔6309 north (seam); every exit reciprocal; coords collision-free + direction-consistent; **6316 a soft terminus (no stub to D5 — D5 attaches elsewhere); no broken bumps.** Adjust coords if needed (keep 6309 the entry, 6315 Reth's).

**Quest-target noun (6315 Reth's Cottage):** author an ansi-highlighted HYPHENATED noun **`field-notes`** (`<ansi fg="itemname">field-notes</ansi>` + `field-notes:` key). BASE (ungated) text = a retired surveyor's old papers/maps, nothing remarkable to a stranger. (The quest trigger in Task 4 adds the gated testimony.) Also give 6315 the **overgrown yew** (per Brennan's directions) and a sparse, orderly, watchful character.

**Brennan's House (6311):** the **blue door**; examinable lore (maps through the window, a scholar's clutter) — NO NPC. A quiet character beat; keep mundane.

**Quality bar:** three-layer descriptions, ≤80 cols, ≥2 nouns, ~20% container nouns. Vary the leading sense. Signature: a quiet, lived-in residential quarter — warm and gentle, the weight all at Reth's end. NO symbol content here (the symbol is the college's, D3); NO crash-site explanation.

- [ ] **Step 3: Spawninfo** (`respawnrate: "20 real minutes"`): 6315→9524 (Reth); 6312→9525 (tea-house keeper); 6310→9526 (neighbor)+9529 (resident); 6313→9527 (gardener); 6314→9528 (retired functionary).
- [ ] **Step 4: Self-check** — reciprocity incl. seam; no collision; no `kind:`; 6316 clean terminus; `field-notes` noun hyphenated+highlighted in 6315; the route honors Brennan's directions; titles Title-Case; colons handled; ≤80 cols; no symbol/crash content.
- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/rooms/greenford/63*.yaml
git commit -m "feat(greenford): Neighborhood 8 rooms (6309-6316) + back-gate seam"
```

---

## Task 2: Items (40160 Reth's map + optional tea good)

- [ ] **40160** — `not_salable` reward item, Reth's marked field-map / survey notes (the crash-site directions; a tangible takeaway). Model on a `not_salable` lore item (e.g. an East Road/Confluence survey/map item). Name e.g. "Reth's Marked Map" (filename keeps no article issue) or "A Surveyor's Marked Map".
- [ ] OPTIONAL **40161** — a tea-house cooking good (`vendor_categories: [cooking]`) if the tea house vendors; else reuse existing.
- [ ] Commit:
```bash
git add _datafiles/world/dogmud/items/materials-40000/4016*.yaml
git commit -m "feat(greenford): Reth's marked map (40160) + tea good"
```

---

## Task 3: Mobs + dialogue (9524–9529) incl. Reth's gated testimony

**Files:** `mobs/greenford/9524.yaml`…`9529.yaml`; `dialogue/greenford/9524.yaml`… (Reth + tea-keeper + neighbor + functionary talk; gardener/resident may be light/ambient).

Common shape: copy a Greenford mob. `non_combatant`/`charm_immune`/`hostile:false`/`maxwander:0`/`noncombat_passive`; Title-Case; `groups: [humanoid]` (Reth `[humanoid]` — retired, apart, NOT margin). Each NPC: ≥3 topics, idle behaviors, a UNIQUE mutation (vs the 21-mutation roster). Voice rules; `|` long text; hint-routing cross-check; NO undeclared quest fields.

| mob | name | room | notes |
|-----|------|------|-------|
| 9524 | (named) Reth | 6315 | **Q75 PAYOFF.** Reluctant/unsettled; **gated dialogue node FIRST: `questRequired: ['75-intro']`** — with Brennan's word he relents, gives the verbal testimony (directions east, landmarks — the lightning-split cairn, the route into the highlands — and **"it's not natural"**: he saw the exposed thing, metal, no seam, not landscape) and tells the player to take his **field-notes** off the table ("there — take the map, it'll get you to the cairn"). NEVER what it is/inside/the symbol (threshold-only). A separate UNGATED node (cold caller, no 75-intro) = he turns them away politely. `|` blocks for the long testimony. |
| 9525 | (named) The Tea-House Keeper | 6312 | **cooking vendor**; neighborhood gossip — Reth "keeps to himself since he came back," Brennan "always at his maps" (reinforces the thread, points toward Reth) |
| 9526 | A Neighbor | 6310 | dialogue; the streets, the college, who lives where (can echo Brennan's "third house with the yew") |
| 9527 | A Gardener | 6313 | ambient; the walled garden, the season |
| 9528 | A Retired Functionary | 6314 | dialogue; an old resident near Reth's end — knew Reth "before he went quiet" (light lore, NO answers) |
| 9529 | A Resident | 6310 | ambient; daily-life color |

Reth is the one heavy NPC. The rest is warm residential texture. NO symbol/crash content from anyone.

- [ ] Commit:
```bash
git add _datafiles/world/dogmud/mobs/greenford/952*.yaml _datafiles/world/dogmud/dialogue/greenford/952*.yaml
git commit -m "feat(greenford): Neighborhood NPCs + dialogue incl. Reth Q75 (9524-9529)"
```

---

## Task 4: Quest 75 — COMPLETION (wire the back-half triggers)

**File:** Modify `_datafiles/world/dogmud/quests/75-the_surveyors_report.yaml` — ADD the back-half triggers to the existing `triggers:` list (the steps + reward + front-half survey trigger are already there from D3). Do NOT change the declared steps or the rewards block.

- [ ] **Step 1: The testimony + completion trigger** (6315 `field-notes` room_interact). Model on Q74's reveal trigger (grant + give_item + bump_rep + send_text):
```
- event: room_interact
  room: 6315
  noun: field-notes
  conditions:
    has: ["75-intro"]
    missing: ["75-end"]
  actions:
    - grant: "75-testimony"
    - grant: "75-end"
    - give_item: 40160
    - bump_rep: {faction: margin, delta: 20}
    - send_text: >-
        (the directions east + landmarks + "it's not natural" — threshold-only;
        you take Reth's marked map. NEVER what it is/inside/the symbol.)
    - room_text: "takes the surveyor's marked map from the table."
- event: room_interact
  room: 6315
  noun: field-notes
  conditions:
    missing: ["75-intro"]
  actions:
    - send_text: >-
        (ungated fallback: a retired man's old survey papers; without his leave,
        not yours to take, and they say little to a stranger.)
```
  **NOTE the multi-grant:** two `grant:` actions (75-testimony then 75-end) in one trigger. **VERIFY at boot + feel-test that BOTH fire** (the quest must reach `end` so rewards fire). IF the engine only honors the first grant (boot/feel shows the quest stuck at testimony, not complete), FALL BACK to: grant only `75-end` in this trigger (drop the `75-testimony` grant — it's a declared step that simply won't display as a discrete beat; the quest jumps intro→end), keeping give_item + bump_rep + send_text. The CRITICAL requirement: **the quest reaches `75-end` (complete) and the player gets 40160 + margin rep.**

- [ ] **Step 2: Self-check** — the only NEW triggers are the 6315 `field-notes` pair; `noun:` matches the room token exactly; grants reference DECLARED steps (75-testimony/75-end); `give_item: 40160` (exists); `bump_rep` faction margin; reward playermessage fires on `end`. No change to steps/rewards. **Commit**
```bash
git add _datafiles/world/dogmud/quests/75-the_surveyors_report.yaml
git commit -m "feat(greenford): Quest 75 completion — Reth's testimony triggers (D4 back half)"
```

---

## Task 5: Schedules (light)

- [ ] 1 anchor schedule `gf_reth` @6315 (Reth reliably home — he's the payoff; day at home/garden, night sleeping) modeled on D1-D3. Optionally `gf_teahouse` @6312. 24h gap-free, in-zone, `schedule_id` wired. Commit:
```bash
git add _datafiles/world/dogmud/schedules/greenford/ _datafiles/world/dogmud/mobs/greenford/9524-*.yaml _datafiles/world/dogmud/mobs/greenford/9525-*.yaml
git commit -m "feat(greenford): Neighborhood anchor schedules"
```

---

## Task 6: Boot + cartcheck

- [ ] Wipe instances. Build + boot; confirm room +8, mobs +6, items +1-2; **`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; **no "trigger grants unknown step"** (75-testimony/75-end are declared); ValidateAllFlags OK; no greenford/63xx/952x warnings.
- [ ] `cartcheck greenford` clean (the seam + D4). Kill. Commit fixes.

---

## Task 7: World-critic + feel polish (MANDATORY — Q75 END-TO-END)

- [ ] **World-critic pass** — directions/geography (Brennan's "first left, third house with the yew" leads to Reth; the college is north/up; the streets quiet); Title-Case; colons; node-shadowing + hint routing; `|` blocks; mutation uniqueness vs whole roster; **NO symbol/crash content in D4**; Reth's threshold (directions + "it's not natural" only); 6316 clean terminus; the Q75 completion trigger correct (give_item + bump_rep + reaches end).
- [ ] **Feel-test (harness + `questtoken`) — Q75 FULL END-TO-END across D3→D4:** from a clean-ish state: Brennan (D3) grant → archive `filed-survey` (75-survey) → Brennan intro (75-intro) → follow his directions through the **back gate (6308 south)** to **Reth (6315)** → Reth's gated dialogue (with intro) → `examine field-notes` → **get the map (40160) + margin rep + Q75 COMPLETES** (75-end; the onward reward beat fires; `quests` shows it DONE). Also confirm: a cold caller (no 75-intro) → Reth turns away + the ungated field-notes fallback (no map). Use `questtoken`/`questtoken flags` for reliable checks. Verify every NPC's hints route; no symbol leak. Report `tools/playtest/reports/2026-06-30-local-feel-tester-greenford-neighborhood.md`.
- [ ] Fix; re-boot; commit.

---

## Task 8: Finish + docs + merge

- [ ] Final clean boot.
- [ ] **Update `docs/ZONE_EXPANSION.md`** row 19: District 4 ✅ built, **Building (4/5)**; note **Q75 The Surveyor's Report COMPLETE** (the crash-site directions now obtainable end-to-end). Commit.
- [ ] `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch.
- [ ] **Update memory** ([[project-zone-expansion-redesign]] + MEMORY.md index): D4 built (UNPUSHED); **Q75 complete end-to-end** (the Eastern Arc's crash-site directions are in the player's hands); next = District 5 West Outskirts (the last Greenford district; the West Road / NP-loop stub).
- [ ] Report to the user.

---

## Self-Review

**Spec coverage:** rooms + seam + Brennan-directions continuity (T1) ✓; `field-notes` noun (T1) ✓; Brennan's-house lore (T1) ✓; 40160 map item (T2) ✓; 6 NPCs incl. Reth's gated testimony (T3) ✓; Q75 completion triggers (T4) ✓; schedule (T5) ✓; boot incl. quest-load (T6) ✓; world-critic + Q75 end-to-end feel (T7) ✓; docs + memory + merge (T8) ✓. No supporting quest (per user) ✓. Threshold-only (Reth gives directions + "it's not natural", never the answer) ✓.

**Placeholder scan:** room/dialogue prose authored by implementer subagents from briefs; the send_text contents in T4 are specified by intent (the build writes the actual testimony prose, threshold-only). "(named)" = builder-named NPCs. The multi-grant fallback in T4 is an explicit, documented branch (verify-then-fallback), not a gap.

**Id/token consistency:** rooms 6309–6316 (8), mobs 9524–9529 (6), item 40160 (+opt 40161) — consistent T1/T2/T3. Q75 back-half tokens 75-testimony/75-end are DECLARED in D3's quest YAML; T4 grants them (+ give_item 40160 + bump_rep margin). The 6308 seam is the only existing-content edit. Carries D1-D3 lessons + the quest SOPs. Closes the split quest.
