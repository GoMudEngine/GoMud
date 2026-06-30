# Greenford — District 3: University District — Design

*Spec date: 2026-06-30. District 3 of 5 (city-wide layer
`2026-06-30-greenford-citywide-design.md`; D1 River merged `6b162857`, D2 Town
Center merged `b8b14bcb`). **The scholarly + symbol-web heart** — the Q75
hub (Brennan + the archive), the Margin's open Greenford node, and the FIRST
orbital-symbol beat in Greenford (threshold-only). The most involved district:
it wires the FRONT HALF of Q75, the back half (Reth's testimony) lands in
District 4.*

## Role

Up the University Stair from the town center: the university grounds — a
quadrangle, a lecture hall, the **library** (public stacks / reading room /
the **archive**), **Brennan's office**, faculty, and a back gate toward the
residential streets (District 4). Quiet, bookish, a "backwater that has escaped
the attention of the people who manage what gets remembered." This is where the
Surveyor's Report becomes a real investigation, where the Margin operates
openly, and where the orbital symbol resurfaces on old maps — unexplained.

- **Folder:** `greenford`. **Rooms:** 6298–6308 (11). **Mobs/dialogue:**
  9516–9523 (8). **Items:** 40160 (reserved; mostly D4). **Quest:** **75** (front
  half declared+granted here; completed in D4).
- **Faction:** extends the existing **`margin`** (Brennan + truth-seeking
  scholars join via `groups: [humanoid, margin]`; no new faction file). The
  university-as-institution / ambient students stay `[humanoid]`.

## Geography & Seam

- **Seam:** D2's **6297 "The University Stair"** `{x:22,y:-80,z:1}` (exit
  `north→6296`; has an iron-`gate` noun). Add **`6297 up → 6298`** (through the
  gate, up into the grounds) and revise 6297's prose so the way up is now
  open/walkable (the gate admits; the grounds climb). 6298 on **z=2** (the
  university sits highest — above the town z=1, above the river z=0; the climb
  reads clean).
- **Back seam to District 4:** a room (the **back gate / college lane**) carries
  a described **stub** toward the residential streets (Brennan's home, Reth's
  cottage — District 4). NOT wired (D4 attaches there).
- **Suggested layout** (builder finalizes a clean reciprocal, collision-free
  graph on z=2; cartcheck/boot-verify):

| Room | Title | role |
|------|-------|------|
| 6298 | The University Gate | up from 6297; the porter; the threshold |
| 6299 | The Quadrangle | hub; scholars; the debate; faculty cross here |
| 6300 | The Lecture Hall | one accessible hall (a lecturer/students) |
| 6301 | The Library | public stacks (the librarian) |
| 6302 | The Reading Room | quiet study; a scholar or two |
| 6303 | The Archive | **Q75 investigation** — Reth's filed survey + the symbol on old maps; the archivist |
| 6304 | The Restricted Collection | gated/stub (deeper records — future; a locked door, lore) |
| 6305 | Brennan's Office | **Q75 giver (Brennan)** + the symbol on his maps |
| 6306 | The Senior Common Room | faculty (the skeptical scholar; debate) |
| 6307 | The Cloister Walk | a quiet connecting walk / garden |
| 6308 | The College Lane | **stub** toward the Neighborhood (District 4) |

(6298 hub-ward to the quad; the library suite 6301→6302→6303→(6304 gated);
Brennan's office + common room off the quad/cloister; 6308 the back-gate stub.
Use a vertical/`up` for the gate climb; keep coords clean.)

## NPCs (mobs 9516–9523: 8)

Canonical Title-Case names, `ConvertForFilename` filenames, ambient archetype
`noncombat_passive`, unique mutations (cross-check vs the WHOLE Greenford roster,
not just this batch — D2 shipped a clone), ≥3 dialogue topics, idle behaviors,
voice rules (**every hint word routes to its node**; **`|` literal block scalars
for ALL long NPC `text:`** — both bit prior districts), NO undeclared quest
fields.

| mob | name | room | role |
|-----|------|------|------|
| 9516 | **Brennan** (named) | 6305 | **Q75 GIVER**, `groups:[humanoid, margin]`. Studies pre-Chrysalis history; knows Reth holds something but can't get it himself. Grants Q75 (find Reth's survey in the archive); on turn-in, gives the **introduction to Reth** (→ go to the north end / D4). Discusses the symbol on his old maps — threshold-only, NO explanation, NO crash-site answer. |
| 9517 | **The Archivist** (named) | 6303 | `[humanoid, margin]`; tends the archive; helps the player locate Reth's filed survey; the restricted collection (gated); lore-light symbol talk |
| 9518 | **The Librarian** (named) | 6301 | `[humanoid]`; the stacks; reading rules; gentle |
| 9519 | **A Skeptical Scholar** | 6306 | `[humanoid]` (or margin); the doubting-scholar debate — questions the official account WITHOUT the answer (a thinking NPC, not a mystery-dump) |
| 9520 | A Lecturer | 6300 | ambient/dialogue-light; the day's lecture, the university |
| 9521 | A Student | 6299/6302 | ambient; scholarly daily life |
| 9522 | The Porter | 6298 | gate-keeper; welcomes/directs; the grounds, the colleges |
| 9523 | A Scholar | 6307/6299 | ambient; cloister/quad color |

Brennan (9516) + the Archivist (9517) are the Q75 + symbol carriers (lore-light,
threshold-only). The Skeptical Scholar (9519) gives the "doubting" texture
WITHOUT leaking the answer. No one explains the symbol or the crash site.

## Quest 75 — The Surveyor's Report (FRONT HALF — declare full skeleton, grant through the intro)

**Pattern: the Confluence Q74 split** — declare the WHOLE skeleton here; grant
only through the **intro** step in D3; the **testimony** + **end** steps' triggers
(and Reth) are wired in D4; **rewards fire ONLY on the `end` step** (D4).

**Skeleton (declare all 5 steps in the Q75 YAML now):**
- `start` — Brennan has set you to find the survey Reth filed on the eastern
  country (the one that "says nothing").
- `survey` — you found Reth's filed survey in the archive: an anomalous
  "mineral deposit" entry, deliberately empty of detail — a man who saw
  something and chose not to write it down. *(room_interact in 6303.)*
- `intro` — Brennan, convinced you're serious, gives you his introduction to
  Reth (retired, the north end of the town). *(Brennan turn-in node → grant
  `75-intro`; quest now in-progress "go to Reth" — D4.)* **← D3 grants to here.**
- `testimony` — *(D4: Reth gives his directions + "it's not natural"; grants the
  marked map/notes item 40160.)*
- `end` — *(D4: you have the directions; the thread runs east to the highlands.
  Rewards + margin rep fire here.)*

**D3 wires:**
- **Brennan grant node** (dialogue, `grantsQuest: '75-start'`, `questExcluded:
  ['75-start','75-end']`, triggers incl. `quest`/`task`) — the breadcrumb rule:
  Brennan's "difficult source" + (D2's bookseller "retired early") + the archive
  survey are the 3 breadcrumbs.
- **The archive survey** (6303 `room_interact`, hyphenated noun e.g.
  `filed-survey`, gated `has:['75-start'] missing:['75-survey']` → `grant
  75-survey` + flavor; ungated fallback = a dry survey, nothing). Model on Q74's
  record triggers exactly.
- **Brennan turn-in node** (`questRequired: ['75-survey']` → grants `75-intro`,
  gives the introduction; voice: go to Reth, north end). This is the
  "Brennan's-introduction" resolution path; the archive-research + approach-Reth
  paths play out at Reth in D4 (Reth responds to whether you carry the intro).
- **The symbol beat** (lore, threshold-only): a noun / room_interact on
  **Brennan's old maps** (6305) and/or the archive (6303) — the orbital symbol
  on the oldest survey maps, recurring, **unexplained** ("it's on the oldest
  maps; the scholars have argued for years; no one agrees"). NO numerology, NO
  "fourth"-count lecture, NO crash-site. Brennan has a dialogue topic on it that
  stays a question, not an answer.

**Mystery boundary (LOCKED):** the symbol APPEARS (first Greenford beat) but is
never explained; Reth's testimony (D4) is "it's not natural" + directions; the
revelation stays for the crash-site interior (#22).

## Items

- **40160** (Reth's marked map / field notes) — the Q75 payoff item, `not_salable`.
  **Granted in D4** (the testimony step). Declare/create it in D4, NOT here
  (reserve the id). D3 may need **no new items** (the archive survey is a
  room_interact record; the intro is a quest token, not an item — avoids the
  give.go gotcha). A flavor book/map item is optional.

## Restricted Collection (6304)

A **gated stub** — a locked door / "by the Keeper of the Archive's leave only,"
deeper records the player can't enter yet (future content / a later quest hook).
Described, lore-rich, not enterable. Models the NP Temple Restricted Collection
stub. NOT a dead-end-bump: frame it as locked-and-known.

## Build conventions & validation

Carry the full Greenford convention/gotcha list PLUS the quest SOPs:
- **Quest dialogue SOP:** grant nodes need `grantsQuest` + `questExcluded`
  (incl. the `75-end` token); `questRequired`/`questExcluded` are LISTS; quest
  nodes include `"quest"`/`"task"` triggers; put **gated grant nodes FIRST** under
  `tree.nodes` (the substring-shadow lesson).
- **`room_interact` nouns are ansi-highlighted HYPHENATED tokens** with matching
  hyphenated noun keys; the quest YAML `noun:` matches that token (model 6303's
  `filed-survey` on Q74's `building-ledger`).
- **A quest trigger may only `grant` a DECLARED step token** (undeclared →
  panic). Declare the full 5-step skeleton; grant only start/survey/intro in D3.
- **Completion/rewards fire ONLY on the step named `end`** (D4) — D3's grants
  leave Q75 in-progress.
- **Declared quest `flags`** (if any) must use the BARE key, referenced
  prefixed (`75-x`) — but Q75 may need NO flag (it's linear, not branching;
  unlike Q74's allegiance). Confirm: no flag unless a branch is added.
- Folder `greenford`; Title-Case; `>`-blocks/quoted colons; no `kind:`; vendor
  categories never `general`; `|` blocks for long text; stage explicit git
  pathspecs never `-A`.

Per-district SOP: `id_inventory` → author → wipe instances → clean boot
(`ValidateZoneConsistency errors=0 mode=panic`, **no "trigger grants unknown
step"**) → `cartcheck greenford` → **world-critic + harness feel-test** (the seam
up, the library/archive, **Q75 end-to-end as far as D3 goes**: Brennan grant →
archive survey room_interact → Brennan intro → in-progress "go to Reth"; the
symbol beat reads threshold-only; the restricted stub; no mystery over-reach) →
update `docs/ZONE_EXPANSION.md` row 19 (3/5) + memory → merge `--no-ff`.

**Quest-testing note (from prior builds):** `questtoken <tok>` GRANTS+persists
(doesn't query) — test the natural grant from a clean char, or grant the chain
to test gated nodes; the harness adapter is flaky for multi-step quests — use
`questtoken`/`questtoken flags` admin for reliable mechanic checks.

## Out of scope (this district)

Reth + his cottage + Q75's testimony/end (District 4); the supporting district
quest (D4 or skip); the West Outskirts (D5); any crash-site/symbol EXPLANATION
(crash-site zone #22).
