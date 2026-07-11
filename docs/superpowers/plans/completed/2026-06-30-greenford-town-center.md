# Greenford — District 2: Town Center — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** Build Greenford's civic hub — the market square, bookshop (Q75 breadcrumb), Cartographer's Rest inn, general store, and a stub up to the University (10 rooms, 7 NPCs, no quest grant).

**Architecture:** Pure YAML in the existing `greenford` zone, on the z=1 plane above the riverfront. The only edit to existing content is wiring the District-1 Town Stair (6287 up→6288). Reuses existing goods where possible.

**Tech Stack:** YAML; built-binary boot; `cartcheck`; mudagent harness.

**Spec:** `docs/superpowers/specs/2026-06-30-greenford-town-center-design.md` (city-wide: `2026-06-30-greenford-citywide-design.md`).

**Reserved IDs:** rooms **6288–6297**, mobs/dialogue **9509–9515**, items **40155–40158**. No quest, no faction, no buffs.

**Branch:** `feature/greenford-town-center` off `master`.

## Conventions (load-fatal — same as District 1 / East Road)
- Zone folder `greenford` (already exists). `zone-config.yaml` already present — do NOT create a second.
- Mob `name` + room `title` canonical Title-Case; mob filename = `ConvertForFilename(name)`; ambient archetype `noncombat_passive`.
- `idlemessages`/idle lines with colon-space single-quoted; description/noun prose-colons in `>` block scalars.
- **Long NPC `text:` fields use `|` LITERAL block scalars** (District 1 shipped a truncation bug where a long double-quoted flow scalar was cut ~140 chars in-game).
- Exits `{roomid}` only — NO `kind:` field.
- Vendors: `craft_support` + `shop:`; salable ITEMS carry a real discipline, never `general` (the general STORE uses `craft_support: general` as the catch-all vendor, but its stocked items each carry a real discipline).
- **Dialogue node-shadowing:** specific nodes first; no short trigger that substring-matches another node's topic; **cross-check EVERY word used in a `hints` line — it must route to the node it advertises** (District 1's feel-test caught dead hint words `road`/`learn`/`trade`/`catch`/`work`/`grain`).
- **Stage explicit git pathspecs, NEVER `git add -A`.**
- Pre-smoke: wipe `rooms.instances/*` + `mobs.instances/*`.

---

## Task 1: 10 rooms (6288–6297) on z=1 + the 6287 seam

**Files:**
- Create: `_datafiles/world/dogmud/rooms/greenford/6288.yaml`…`6297.yaml`
- Modify: `_datafiles/world/dogmud/rooms/greenford/6287.yaml` (wire the stair up)

- [ ] **Step 1: Seam.** In `6287.yaml` (The Town Stair, `{22,-76,0}`, exits `north→6284, west→6286`, no onward; has a `the stair` noun "rising into the town"), add:
```yaml
  up:
    roomid: 6288
```
and lightly revise its prose so the stair now CLIMBS to the market square above (the way up is open/walkable), not a "coming soon" stub. Keep its other exits/nouns.

- [ ] **Step 2: Author 6288–6297** on **z=1** (the upper town). Suggested coords (builder finalizes a clean, fully-reciprocal, collision-free graph; the z=1 plane around {22,−76} is clear):

| roomid | title | x | y | z | role |
|--------|-------|---|---|---|------|
| 6288 | The Market Square | 22 | -76 | 1 | down→6287; the hub |
| 6289 | The Market Cross | 22 | -77 | 1 | stalls / notice board / crier |
| 6290 | The Bookshop | 21 | -76 | 1 | bookseller + Q75 breadcrumb |
| 6291 | The Cartographer's Rest | 23 | -76 | 1 | inn common room (vendor) |
| 6292 | The Cartographer's Rest, Upstairs | 23 | -76 | 2 | inn lodging (up from 6291) |
| 6293 | The General Store | 21 | -77 | 1 | general-store vendor |
| 6294 | The Guild Lane | 23 | -77 | 1 | scribe/chandler texture |
| 6295 | The Town Hall Steps | 22 | -78 | 1 | civic building exterior |
| 6296 | The Upper Street | 22 | -79 | 1 | climbs toward the university |
| 6297 | The University Stair | 22 | -80 | 1 | STUB up to District 3 |

**Builder finalizes** a sensible reciprocal graph: 6288 is the hub (down→6287, plus exits to the square/shops); the bookshop/inn/store open off the square; 6295→6296→6297 climbs south toward the university. REQUIRED invariants: 6287 up↔6288 down (the seam); every exit reciprocal; **6297 "The University Stair" has NO onward (up/south-beyond) exit** (prose only — District 3 attaches later); coords collision-free + direction-consistent (north=+y, east=+x, up=+z). If suggested coords don't yield a clean graph, adjust (keep 6288 the hub, 6297 the top stub) and note changes.

**Each room:** three-layer description (atmospheric lead → grounded detail → interaction hint), ≤80 cols, ≥2 nouns, ~20% with a container noun (a notice board, a book-crate, a sack of post, an inn lost-and-found). Vary the leading sense. Signature: a SMALL, quiet, bookish civic town (contrast the Confluence plaza) — comfortable, lived-in; the university tower close and clear uphill (named at 6296/6297). **NO orbital-symbol content anywhere** (mundane only; the mystery is District 3).

- [ ] **Step 3: Spawninfo.** `respawnrate: "20 real minutes"`: 6290→9509 (bookseller); 6291→9510 (innkeeper); 6293→9511 (store-keeper); 6289→9512 (produce-seller)+9513 (crier); 6296→9514 (student); 6294→9515 (townsperson).

- [ ] **Step 4: Self-check** — reciprocity incl. the 6287↔6288 seam; no coord collision; no `kind:`; 6297 no onward exit; titles Title-Case; colons handled; ≤80 cols; no symbol content.

- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/rooms/greenford/62*.yaml
git commit -m "feat(greenford): Town Center 10 rooms (6288-6297) + Town Stair seam"
```

---

## Task 2: Items (40155–40158)

- [ ] Create up to 4 goods, each `vendor_categories: [<real discipline>]` (never `general`); model on East Road `40147` / Greenford `40152`. Suggested: an inn cooking good (`cooking`), 1-2 general-store mixed goods carrying real disciplines (e.g. a `tailoring` cloth, a `general`-store-stocked `cooking` item), a market produce (`cooking`). Reuse existing goods where they fit (fewer new items is fine). A book/map flavor item is optional — if non-vendor, `not_salable: true`.
- [ ] Commit:
```bash
git add _datafiles/world/dogmud/items/materials-40000/4015*.yaml
git commit -m "feat(greenford): Town Center vendor goods (40155-40158)"
```

---

## Task 3: Mobs + dialogue (9509–9515)

**Files:** `mobs/greenford/9509.yaml`…`9515.yaml`; `dialogue/greenford/9509.yaml`…`9515.yaml` (all 7 talk; none are fauna).

Common shape: copy a Greenford D1 mob (`mobs/greenford/9501-*` warden, `9502-*` vendor) or river_road equivalents. `non_combatant: true`, `charm_immune: true`, `hostile: false`, `archetype: noncombat_passive` (vendors use the shopkeeper archetype like 9502), `maxwander: 0`, Title-Case name, `groups: [humanoid]`, idlecommands. Each NPC: ≥3 topics, idle behaviors, a UNIQUE mutation. Voice rules; **long `text:` fields = `|` literal block scalars**; cross-check every hint word routes to its node; NO quest fields.

| mob | name | room | type |
|-----|------|------|------|
| 9509 | (named) The Bookseller | 6290 | dialogue; books/local lore; **plants the Q75 breadcrumb** — asked about surveyors / the eastern country / "the hill", names "the surveyor who retired early" (Reth) + that Brennan at the university "asks the same sorts of questions." NO crash-site answer, NO orbital symbol |
| 9510 | (named) The Innkeeper | 6291 | **cooking vendor** (shop: an inn food + reuse); the road/town/travelers |
| 9511 | (named) The Store-Keeper | 6293 | **`craft_support: general` vendor** (shop: the mixed goods — each item a real discipline); daily trade |
| 9512 | A Produce-Seller | 6289 | dialogue-light or small `cooking` vendor; market color |
| 9513 | A Town Crier | 6289 | dialogue-light; civic news, no mystery |
| 9514 | A Student | 6296 | dialogue; lore-light forward-gesture to the university (what's studied, why people come); NEVER mystery/crash-site |
| 9515 | A Townsperson | 6294 | ambient; civic daily-life |

The bookseller (9509) is the Q75 seed; the student (9514) the university gesture. No Margin here (District 3).

- [ ] Commit:
```bash
git add _datafiles/world/dogmud/mobs/greenford/95*.yaml _datafiles/world/dogmud/dialogue/greenford/95*.yaml
git commit -m "feat(greenford): Town Center NPCs + dialogue (9509-9515)"
```

---

## Task 4: Schedules (light)

- [ ] 1–2 anchor schedules (e.g. `gf_innkeeper` @6291, `gf_bookseller` @6290) modeled on D1's `gf_warden`/`gf_miller` — full 24h coverage, in-zone target rooms, `schedule_id` wired on those mobs. Vendors stay reliably at post by day. Commit:
```bash
git add _datafiles/world/dogmud/schedules/greenford/ _datafiles/world/dogmud/mobs/greenford/9509-*.yaml _datafiles/world/dogmud/mobs/greenford/9510-*.yaml
git commit -m "feat(greenford): Town Center anchor schedules"
```

---

## Task 5: Boot + cartcheck

- [ ] Wipe instances. Build + boot; confirm room +10, mobs +7, items +N; **`ValidateZoneConsistency errors=0 warnings=0 mode=panic`**; no panics; no greenford/629x/95xx warnings.
- [ ] `cartcheck greenford` clean (incl. the z=1 town + the 6287↔6288 seam). Kill. Commit fixes.

---

## Task 6: World-critic + feel polish (MANDATORY)

- [ ] **World-critic pass** over the new district — directions/geography (the river/riverfront is DOWN via the Town Stair; the university is UP/further south; the Confluence is far west); Title-Case; colons; vendor categories; **node-shadowing + every hint word routes**; 6297 reads as "university just up, coming soon" not broken; **NO orbital-symbol/mystery bleed** (the bookseller stays lore-light, breadcrumb-only); the general-store vendor wiring is correct.
- [ ] **Feel-test** (harness) — `teleport 6287`, `up` into the square; the shops + inn (`list`/`buy` at innkeeper + store-keeper), the bookseller breadcrumb (ask about surveyors/Reth/the hill → routes + lore-light), the student's university gesture, every hint word routes, the University Stair stub. Zero-bug bar. Report `tools/playtest/reports/2026-06-30-local-feel-tester-greenford-towncenter.md`.
- [ ] Fix; re-boot; commit.

---

## Task 7: Finish + docs + merge

- [ ] Final clean boot.
- [ ] **Update `docs/ZONE_EXPANSION.md`** row 19: District 2 ✅ built, status **Building (2/5)**; note the Town Center + Q75 breadcrumb seeded. Commit.
- [ ] `superpowers:finishing-a-development-branch` → merge `--no-ff` into master; delete branch.
- [ ] **Update memory** ([[project-zone-expansion-redesign]] + MEMORY.md index): Greenford District 2 built (UNPUSHED); next = District 3 University (Q75 hub + Margin + the symbol content).
- [ ] Report to the user.

---

## Self-Review

**Spec coverage:** rooms + seam (T1) ✓; bookshop breadcrumb (T1/T3) ✓; University-stair stub (T1) ✓; items (T2) ✓; 7 NPCs incl. bookseller + student + 3 vendors (T3) ✓; schedules (T4) ✓; boot/cartcheck (T5) ✓; world-critic + feel (T6) ✓; ZONE_EXPANSION + memory + merge (T7) ✓. No quest grant / Margin / symbol content — matches spec (deferred to D3) ✓.

**Placeholder scan:** room/dialogue prose authored by implementer subagents from the per-room/per-NPC briefs (content, not logic); "(named)" markers for the bookseller/innkeeper/store-keeper are intentional builder-named NPCs. No TBD logic.

**Id consistency:** rooms 6288–6297 (10), mobs 9509–9515 (7, all talking), items 40155–40158 (≤4) — consistent across T1/T2/T3. z=1 plane (6292 at z=2 for the inn upstairs). The 6287 seam is the only existing-content edit. Carries the District-1 lessons: `|` blocks for long text, hint-word routing cross-check, general-store vendor pattern.
