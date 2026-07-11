# The Confluence — District 6a: Cloisters & Archive (Q74 part 1)

**Date:** 2026-06-27
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (§3 row 6, §6 Q74)
**Predecessor:** District 5b — The Temple of Confluence (public), merged `cb628f47`. Its
warded inner door (6183) is the seam this district opens.
**Successor:** District 6b — The Undercroft. Q74 **grants and plays its first half here**;
the descent, the orbital "face" reveal, and the allegiance-flag choice land in 6b.

## 1. Concept

The inner temple beyond the public nave's warded door — the senior Keepers' domain,
where the official account is *administered* and quietly guarded. This is where **Q74
grants** and the player gathers the **construction-history thread**: the temple was
raised *on* older foundations the Founders found and consecrated, not built. The
institutional conscience of the faith, in three figures:

- **Aldric the Keeper** (temple head) — the **lid-keeper**. He *knows*, and has chosen
  to keep it quiet; a man carrying the burden, not a villain. The trust-gate who
  reluctantly conditions the descent.
- **Brother Cael** (archivist) — the **sympathizer**. Quietly helps; gives the player
  the construction records. The Q74 enabler.
- **Prioress Crane** — the **believer**. Genuine, devout, and troubled by the
  questions; the human face of conviction.

**Lore boundary (threshold only):** the records show *there was older work / a
suppressed building history / the faith is laid over something earlier.* They NEVER
state the why (crash, gray material, mutation link, numerology). The reveal itself is
6b; 6a only earns the descent.

## 2. Entry / seam

District 5b ends at the warded inner door (6183 "The Inner Threshold", `{5,-77,0}`,
currently `west→6175` only). **6a wires `6183 → east → 6184`** (the cloister gate).
The door admits a player who has come this far on the thread; the Threshold Warden
(9459, 5b) turns back the idle but lets the vouched pass. **Gating is narrative +
quest, not a hard exit lock** — mobs don't block exits in this engine (consistent
with the rest of the world), so the cloister is physically walkable; the **descent**
(6199 down) and the **senior Keepers' confidence** (Q-gated dialogue) are the real
gates.

**Pay off the 5b Officiant hook:** the 5b Officiant (9457) `beyond` node and Warden
(9459) both promise "the senior Keepers, if your questions warrant it." 6a adds a
**`73-end`-gated line** to the Officiant (and/or Warden) that, for a Q73-completed
player, explicitly admits them through to the cloisters — closing the loop the 5b
feel-review flagged (do this as a dialogue edit in the build).

## 3. Coordinate frame & collision discipline

- Island **east half**, x ≈ +6 → +10, y −75 → −79, z 0. Attaches at 6183 `{5,-77,0}`.
- The **descent stairhead (6199)** has a `down` exit that is a **described/gated stub
  this build** (the 6b Undercroft, z<0 beneath, is not wired until 6b).
- **Exact coords below are the build target and MUST be `cartcheck`-verified
  (mode=panic)** against all built districts. The cloister block (x +6→+10) sits east
  of the public temple (x +2→+5) — no overlap; no rooms exist east of 6183 yet.

## 4. Rooms (6184–6199, 16) — a monastic cloister

| roomid | title | coord (x,y,z) | exits |
|--------|-------|---------------|-------|
| 6184 | The Cloister Gate | 6,-77,0 | W→6183, E→6185 |
| 6185 | The Cloister Garth | 7,-77,0 | W→6184, N→6186, S→6187, E→6192 |
| 6186 | The North Walk | 7,-76,0 | S→6185, N→6190, W→6188, E→6189 |
| 6188 | The Chapter House | 6,-76,0 | E→6186 |
| 6190 | The Scriptorium | 7,-75,0 | S→6186 |
| 6189 | The Archive | 8,-76,0 | W→6186, E→6195 |
| 6195 | Aldric's Study | 9,-76,0 | W→6189 |
| 6192 | The East Walk | 8,-77,0 | W→6185, E→6196, S→6193 |
| 6193 | The Infirmary | 8,-78,0 | N→6192 |
| 6196 | The Prioress's Oratory | 9,-77,0 | W→6192, S→6198 |
| 6198 | The Older East Corridor | 9,-78,0 | N→6196, E→6199 |
| 6199 | The Descent Stairhead | 10,-78,0 | W→6198, **D→stub (6b Undercroft)** |
| 6187 | The South Walk | 7,-78,0 | N→6185, W→6191, S→6194 |
| 6191 | The Refectory | 6,-78,0 | E→6187, S→6197 |
| 6197 | The Kitchen Court | 6,-79,0 | N→6191 |
| 6194 | The Cells | 7,-79,0 | N→6187 |

Room count: 16. All exits reciprocal (verify each pair at build); all unit-adjacent
(no long exits); no coord collisions. Biome `city` throughout (temple stone),
consistent with 5a/5b. **6199's `down` is the only stub** — described in prose (a
stair descending into the dark beneath the island), not wired east/down to a real
room until 6b.

**Room roles / required nouns:**
- 6184 Cloister Gate — the inner face of the warded door; quiet, the boundary crossed.
- 6185 Cloister Garth — the central quadrangle/garden; the hub; daily monastic life.
- 6186 North Walk / 6187 South Walk / 6192 East Walk — covered cloister walks.
- 6188 Chapter House — where the senior Keepers meet; Aldric presides. Required noun
  **`consecration-record`** (the founding record: the Founders "consecrated what they
  found" — a Q74 clue; quest-gated `look`).
- 6189 The Archive — Cael's domain; the temple's records. Required noun
  **`building-ledger`** (construction accounts showing the foundations predate the
  recorded build — a Q74 clue; quest-gated).
- 6190 Scriptorium — copying rooms; a scribe.
- 6195 Aldric's Study — the lid-keeper's private room; a guarded, lived-in space; a
  hint of a man who has made his peace with silence (threshold-only).
- 6196 Prioress's Oratory — Crane's chapel/quarters; devotion, and quiet unease.
- 6191 Refectory / 6197 Kitchen Court / 6194 Cells / 6193 Infirmary — the lived-in
  monastery (meals, sleep, care).
- 6198 The Older East Corridor — visibly **pre-Founding stonework** (the same older
  work as the crossing/relic stone); leads to the descent. Required noun
  **`masons-survey`** (a surveyor's note left in the corridor that the lower courses
  here are older work, cut by hands the temple's records don't reach — a Q74 clue;
  quest-gated). Threshold-only.
- 6199 The Descent Stairhead — the stair down beneath the island; gated (the Warden's
  counterpart / Aldric's permission); the Q74 descent point; `down` stub to 6b.

The three Q74 record-nouns use the **quest-gated `room_interact` `look <hyphenated-
noun>` pattern** (the Q73 model): an ansi-highlighted hyphenated token in the prose
+ matching noun key + the quest YAML `room_interact` trigger. Without the quest token
the `look` shows only the lore description; with it, the trigger fires and advances
the quest.

## 5. NPCs (mobs/dialogue 9464–9470, ~7)

All `groups: [..., keepers]`, `non_combatant: true`.

| mobid | name | where | role |
|-------|------|-------|------|
| 9464 | Aldric the Keeper | 6188 Chapter House (schedule may visit 6195) | **lid-keeper, anchor**; conditions the descent; the burdened head. Q74: the reluctant permission. |
| 9465 | Brother Cael | 6189 Archive | **sympathizer**; **grants Q74**; gives the construction-history thread; points to the records. |
| 9466 | Prioress Crane | 6196 Oratory | **believer**; devout, troubled; the human face; lore on the faith, not the descent. |
| 9467 | A Scribe | 6190 Scriptorium | ambient; copying work. |
| 9468 | The Cellarer | 6191 Refectory | ambient; the monastery's daily provisioning. |
| 9469 | The Infirmarian | 6193 Infirmary | ambient; care; ties to the Almoner's referenced infirmary. |
| 9470 | A Cloister Keeper | 6185 Garth | ambient; tends the garth. |

Senior Keepers Aldric/Cael/Crane get full dialogue (with Q73-gated asides where
fitting and Q74 nodes on Cael/Aldric); ambient keepers are idlecommand-flavor + light
dialogue.

## 6. Q74 — The Undercroft (the 6a portion)

IDs from **74**. Declare the quest YAML now (steps + the allegiance flag), even though
the flag is **set** in 6b.

1. **Grant (Cael, the Archive 6189):** a **`73-end`-gated, `74-start`/`74-end`-excluded**
   dialogue node — Cael recognizes a Q73-completed seeker and gives the thread:
   the temple's own building records disagree with its founding story; examine them
   yourself. `grantsQuest: "74-start"`. Framed as "Aldric permits a supervised look"
   (the lid-keeper's conditional, the sympathizer's enabling — the two flavored ways
   in, one mechanical path).
2. **Investigate (room_interact, quest-gated):** examine the three construction-history
   records — `building-ledger` (6189 Archive), `masons-survey` (6198 Older East
   Corridor), `consecration-record` (6188 Chapter House) — each a quest-gated `look`. Model the step mechanics on the shipped
   **Q73** quest YAML (`73-the_margin_notation`): declare each granted step token in
   the quest YAML (an undeclared intermediate token panics at load), gate the
   `room_interact` triggers on the prior token, and require all three before the
   turn-in.
3. **Earn the descent (report to Cael/Aldric):** a turn-in node → establishes the
   descent is warranted → grants an in-progress token (e.g. `74-descent`) and Aldric's
   conditional permission. **Q74 stays in-progress** (not `74-end`) — the descent, the
   sealed-chamber reveal, and the `74-allegiance` flag choice are **6b**. The 6199
   stairhead's `down` is a stub until then.
4. **Allegiance flag:** declare in the Q74 YAML — `key: 74-allegiance`, `values:
   [margin, keepers]`, description "whether the player carries the truth to the Margin
   or keeps the Keepers' confidence." **Set in 6b** at the climax (not in 6a).

**SOPs to honor (from the Q73 build + the dialogue lessons):**
- Every `grantsQuest` node includes the quest end token in `questExcluded`
  (`74-start` node excludes `["74-start","74-end"]`); quest-grant nodes include
  `"quest"`+`"task"` in triggers.
- `questRequired`/`questExcluded` are **lists**; gated/grant nodes placed **FIRST**.
- `room_interact` nouns are ansi-highlighted **hyphenated** tokens, noun key matches.
- A quest trigger may only **grant a declared step token**; final completion (in 6b)
  grants `74-end` directly.

## 7. Items

**0 new items** in 6a (the records are room_interact lore nouns, not items). Any Q74
reward lands in 6b at completion.

## 8. Schedules (anchor NPCs)

Day-at-post + night, findability-preserving (the established pattern). Likely: Aldric
(Chapter House by day, his study evenings), Cael (the Archive), Crane (the Oratory).
Keep them findable for Q74. Author so the world-critic/feel checks pass.

## 9. Process & verification

Same proven cycle:
1. Subagent-driven build: wire the 6183→6184 seam + the 5b Officiant/Warden `73-end`
   admit line; rooms; mobs/dialogue; the Q74 quest YAML; schedules.
2. **Boot** `ValidateZoneConsistency` mode=panic + `cartcheck the_confluence` clean
   (reciprocal exits, no collisions, the 6199 down-stub, the quest loads without the
   "trigger grants unknown step" / flag-undeclared panics). Instance-wipe per SOP.
3. **Polish pass (mandatory):** world-critic + feel-tester over 6184–6199. Recurring
   traps: river/compass directions; dialogue node-shadowing (gated/grant nodes FIRST);
   room **and** mob **and** room-title Title-Case; colon quoting; hyphenated nouns;
   no `kind:` on exits; **Q74 grant/room_interact/turn-in actually fires** (test with
   a clean char or the `questtoken` chain — note `questtoken` grants don't survive a
   force-killed server).
4. **Harness-verify Q74's 6a half end-to-end:** grant (Cael), the three record
   room_interacts, the turn-in / descent-earned token, and confirm Q74 sits
   in-progress with the 6199 stub described correctly.

## 10. Out of scope

- The descent, the Undercroft rooms, the orbital "face" reveal, the `74-allegiance`
  flag *set*, and `74-end` — all **district 6b**.
- The senior Keepers' deeper knowledge stated outright (threshold-only until 6b's
  reveal, and even there only the threshold).
- Any combat/wards (the Undercroft's guardians are 6b).
- New items (none in 6a).

## 11. World impact

World grows by 16 rooms and ~7 mobs. The Confluence reaches 7 of ~10 districts (the
whole island — public temple + inner cloisters — plus the river/civic west bank),
with Q74 live through its first half and the descent staged for the climax build.
