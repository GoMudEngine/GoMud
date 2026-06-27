# The Confluence — District 6b: The Undercroft (Q74 climax) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Undercroft — the three-level descent beneath the temple island (18 rooms, one guardian) — and complete Q74: the orbital-"face" threshold reveal + the soft allegiance choice (carry the truth to the Margin vs keep the Keepers' confidence).

**Architecture:** Pure YAML content. 18 rooms across z−1…z−3 beneath the island, reached by wiring the 6a descent stairhead (6199) `down`. Q74's back half completes via quest triggers modeled on the shipped **Q68** branching quest (`quests/68-the_cooperage_circle.yaml`): a `room_interact` reveal at the orbital-face grants `74-reveal` and gives a "rubbing" item; two `item_give` branches (give the rubbing to Aldric = keepers, or to the Margin scholar = margin) each `set_flag` + `bump_rep` (chosen faction +, other −) + `grant: 74-end`. The full Q74 step skeleton + the `allegiance` flag are already declared (6a); 6b adds the triggers + finalizes rewards. **Lore-boundary is paramount: threshold only — the reveal proves a fourth was lost and the temple suppressed it, never the why (reserved for the crash-site zone).**

**Tech Stack:** YAML data files; `go run .` boot; `cartcheck`/`ValidateZoneConsistency`; mudagent harness + `questtoken` admin.

**Spec:** `docs/superpowers/specs/2026-06-27-confluence-undercroft-design.md`

**Reserved IDs (verify with `python tools/id_inventory.py` at build):** rooms **6200–6217**, mobs/dialogue **9471+**, item **40142**, quest **74** (extend), buffs **94+** (only if the guardian debuffs).

**Branch:** create `feature/confluence-undercroft` off `master` before Task 1.

---

## Authoring conventions (read once)

Same load-fatal rules as 5a/5b/6a, plus:
1. Mob names + room titles canonical Title-Case (em-dash `—` not `--`); idlemessages with colon-space single-quoted; description/noun colons in `>` blocks.
2. Dialogue/quest gates: `questRequired`/`questExcluded` LISTS; gated nodes first; quest triggers may only grant **declared** steps.
3. **Inter-level stairs** = `up`/`down` exits with **stacked coords** (same x,y, z±1); within a level, cardinal exits. No `kind:` field on exits.
4. **`room_interact` nouns** = ansi-highlighted hyphenated token in prose + matching `nouns:` key; quest `noun:` matches exactly.
5. **Quest flag key is BARE** in the declaration (`allegiance`, already declared in 6a) and referenced `74-allegiance` everywhere (verified vs shipped Q11/Q68).
6. **Completion/rewards fire only on the step named `end`** — granting `74-end` (in the item_give branches) completes Q74 + fires the rewards block.
7. **40xxx items** live in `items/materials-40000/`; `not_salable: true` for non-vendor items.
8. Pre-smoke SOP: wipe `mobs.instances/*` + `rooms.instances/*` before every boot.
9. **THRESHOLD-ONLY LORE** everywhere in this district — the single most important content rule. No crash, gray material, mutation link, numerology.

Prose delegated to build agents against the worked examples + the voice of the 5b/6a Confluence files. Plan fixes all IDs/coords/exits/nouns/quest wiring.

---

## Task 1: Wire the descent + fix the 6a stairhead stub

**Files:** Modify `rooms/the_confluence/6199.yaml`.

The 6a feel-review flagged 6199 as reading "broken" (the quest says "go down" but no exit). This task wires it.

- [ ] **Step 1: Add the `down` exit** to 6200. Current `exits:` is `west: {roomid: 6198}`. Final:

```yaml
exits:
  west:
    roomid: 6198
  down:
    roomid: 6200
```

- [ ] **Step 2: Add a `stair` noun** so `look stair` reads (and the room no longer feels like a dead-end): a hyphenated `descending-stair` noun describing the steps going down into the dark, now passable. Update the description so the stair is clearly the way down. Keep ≤80 cols, threshold-only (cool air, deep water, old stone — no reveal).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6199.yaml
git commit -m "feat(confluence): wire the undercroft descent (6199 down->6200) + stair noun"
```

---

## Task 2: The 18 undercroft rooms (6200–6217, z−1…−3)

**Files (Create, in `rooms/the_confluence/`):** 6200–6217.

**Coordinate + exit table (authoritative; cartcheck-verified; every exit reciprocal; no `kind:`). Inter-level stairs are stacked (same x,y, z±1):**

| roomid | title | x | y | z | exits |
|--------|-------|---|---|---|-------|
| **z−1 — Upper Undercroft** ||||||
| 6200 | The Undercroft Stair-Foot | 10 | -78 | -1 | up→6199, west→6201 |
| 6201 | The Foundation Vault | 9 | -78 | -1 | east→6200, west→6202, south→6204 |
| 6202 | The Early Crypt | 8 | -78 | -1 | east→6201, west→6203 |
| 6203 | The Seeping Passage | 7 | -78 | -1 | east→6202, down→6206, south→6205 |
| 6204 | A Storeroom | 9 | -79 | -1 | north→6201 |
| 6205 | The Old Cistern | 7 | -79 | -1 | north→6203 |
| **z−2 — The Old Halls** ||||||
| 6206 | The Old Hall Landing | 7 | -78 | -2 | up→6203, west→6207 |
| 6207 | The Pillared Hall | 6 | -78 | -2 | east→6206, west→6208, south→6209 |
| 6208 | The Water Gallery | 5 | -78 | -2 | east→6207, down→6212, west→6211 |
| 6209 | The Wardroom | 6 | -79 | -2 | north→6207 |
| 6210 | A Flooded Cell | 5 | -79 | -2 | north→6208 |
| 6211 | The Echoing Walk | 4 | -78 | -2 | east→6208 |
| **z−3 — The Deep / Sealed Chamber** ||||||
| 6212 | The Deep Landing | 5 | -78 | -3 | up→6208, west→6213 |
| 6213 | The Antechamber | 4 | -78 | -3 | east→6212, west→6214, south→6216 |
| 6214 | The Sealed Door | 3 | -78 | -3 | east→6213, west→6215, south→6217 |
| 6215 | The Chamber of the Old Sky | 2 | -78 | -3 | east→6214 |
| 6216 | A Drowned Vault | 4 | -79 | -3 | north→6213 |
| 6217 | The Still Water | 3 | -79 | -3 | north→6214 |

All exits reciprocal (walk the table); stairs 6199↔6200, 6203↔6206, 6208↔6212 are stacked verticals. Biome `city` (cut stone) throughout — or a darker tint if one reads better; keep consistent. No coord collisions (all z<0; distinct within level).

**Descent arc (threshold-only):**
- **z−1** — the temple's own foundations meeting older work; an early-Keepers crypt; seeping water. The masonry shifts from temple-cut to older-cut as you go west/down.
- **z−2** — pre-Founding halls; the geometry is *not* the temple's; the joined waters run close in the Water Gallery (6208). **The guardian (9471) is in 6208**, by the down-stair.
- **z−3** — the deep: antechamber → the sealed door (6214) → **the Chamber of the Old Sky (6215)**, the reveal.

**Required nouns:**
- **6215 Chamber of the Old Sky** → `orbital-face` (the climax): a great pre-Founding carving/mechanism of the **old sky with FOUR rings** — the fourth that was lost; older than the temple, the thing the whole faith was laid over. The base `nouns:` lore is suggestive; the Q74 `room_interact` trigger (Task 4) adds the gated reveal text. Also a supporting `inscription` or `ring-carving` noun (threshold-only).
- **6214 The Sealed Door** → `sealed-door` (the threshold into the chamber; pre-Founding seals, opened by the Keepers long ago and never closed).
- **6208 Water Gallery** → a `dark-water` / `joined-channel` noun (the three rivers running as one, close beneath — payoff of "the waters join below the floor," now you're *at* them).

**Spawninfo:** 6208 → 9471 (the guardian). (Optional threshold-voice Keeper on z−1 per §3 — builder's call; if added, allocate 9472 and place at 6200/6201. Lean spare.)

- [ ] **Step 1: Author 6200–6217.** Worked example (the reveal chamber — base noun lore; the quest layers the gated reveal in Task 4):

```yaml
roomid: 6215
zone: The Confluence
title: The Chamber of the Old Sky
description: >
  The deepest room the stair reaches: a domed chamber of
  pre-Founding stone, dry where everything above it wept
  water. The whole of the far wall is given to a single
  great carving — the
  <ansi fg="itemname">orbital-face</ansi>
  — a map of the sky in concentric rings, cut by a hand
  that worked before the temple was a thought. The Keepers
  built three hundred years of doctrine in the rooms above
  this one. Down here, the wall keeps its own older count.
  The way back is east.
biome: city
coord:
  x: 2
  y: -78
  z: -3
exits:
  east:
    roomid: 6214
nouns:
  orbital-face: >
    A carving that fills the wall: the night sky rendered
    as a set of nested rings about a central mark, precise
    as an instrument. Whoever cut it counted the rings with
    care -- there are four. The temple above keeps to three
    in everything it teaches. The stone does not argue; it
    simply shows what it was cut to show, and lets the
    difference sit there in the dark.
idlemessages:
- The chamber is silent in a way the rooms above are not --
  no water, no draft, only stone and the carving.
- 'A faint line of old pigment survives in one ring of the
  carving: it was painted, once, and meant to be seen.'
```
(Threshold-only — the carving shows *four where the temple teaches three*; it never says why, or what the fourth was beyond "a ring / a water / a light that was lost.")

- [ ] **Step 2: Self-check** — every exit reciprocal; the 3 inter-level stairs stacked; no coord collision; no `kind:`; `orbital-face` (6215), `sealed-door` (6214), water noun (6208) present (hyphenated token + key); 6208 spawninfo 9471; titles Title-Case; colons handled; ≤80 cols; lore threshold-only (re-read every room for any "why" leak).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/620*.yaml _datafiles/world/dogmud/rooms/the_confluence/621*.yaml
git commit -m "feat(confluence): the Undercroft, 18 rooms across z-1..-3 (6200-6217)"
```

---

## Task 3: The guardian (+ optional threshold-voice Keeper)

**Files (Create):** `mobs/the_confluence/9471-the_ward_construct.yaml` (+ dialogue optional/none — a construct need not talk). If a threshold-voice Keeper is added, `9472-...` + dialogue.

The guardian — locked strength (tune from playtest): **the toughest single mob in the Confluence, beatable by a Q73→Q74 character, avoidable by a careful one.**

- [ ] **Step 1: Author the guardian (9471).** Model the combat shape on an existing fighting mob (e.g. read `mobs/ironwind_steppe/215-alpha_wolf.yaml` or `9393-a_canal_lurker.yaml` for the field set), then:

```yaml
mobid: 9471
zone: The Confluence
behavior_archetype: defender
archetype: fighting
hostile: false
statpool: 100
itemdropchance: 0
maxwander: 0
groups:
  - construct
activitylevel: 8
idlecommands:
  - 'emote stands motionless in the gallery, a figure of fitted pre-Founding
    stone, and does not acknowledge you.'
  - ''
  - 'emote shifts its weight a single degree, stone grinding on stone, tracking
    movement it was set here to track.'
  - ''
character:
  name: The Ward of the Deep
  description: |
    A guardian of fitted grey stone in the rough shape of a robed figure,
    set into the gallery where the stair goes down. It is older than the
    temple and answers to nothing the temple teaches -- a ward left by the
    builders of the deep to keep their threshold. It does not pursue. It
    does not speak. It permits the quiet and rouses against the forceful,
    and it has stood here long enough to have forgotten it was ever told
    why.
  speciesid: 1
  level: 1
  gold: 0
  stats:
    vitality:
      training: 12
    strength:
      training: 9
    dexterity:
      training: 3
```
Rationale baked into the values: **`statpool: 100`** (top of the Confluence, below the 120–130 raid elites); **high vitality** (durability/HP) + **high strength** (real damage) + **low dexterity** (slow, inaccurate, few swings → beatable by patience); **`hostile: false` + `maxwander: 0`** (guards, doesn't hunt — the player can fight it OR walk past it down the stair, since mobs don't block exits, satisfying "optional/avoidable"). `non_combatant` is NOT set (it is attackable). `groups: [construct]` (NOT keepers — the old site's own ward). No ranged, no spells. (If you want it to bite back harder, give it a "chill of the deep" debuff buff 94 on hit — OPTIONAL; only add if it earns its keep.)

- [ ] **Step 2 (optional): threshold-voice Keeper (9472).** If included per spec §3, a Keeper on z−1 (6200/6201) who descended this far and will go no lower — a human anchor at the threshold; `groups: [humanoid, keepers]`, non_combatant, short dialogue (awe/dread, points the player down, threshold-only). Skip if keeping it spare.

- [ ] **Step 3: Self-check** — name Title-Case; the guardian is attackable (no non_combatant); not in the keepers faction; no ranged/spell; threshold-only description.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/9471-the_ward_construct.yaml
git commit -m "feat(confluence): the Ward of the Deep guardian (9471)"
```

---

## Task 4: Item 40142 + Q74 completion wiring (the climax)

**Files:** Create `items/materials-40000/40142-a_rubbing_of_the_old_sky.yaml`; Modify `quests/74-the_undercroft.yaml` (add the reveal + two completion triggers; finalize rewards).

- [ ] **Step 1: The reward/proof item (40142).** The rubbing the player takes from the orbital-face — the carried proof + the thing they choose what to do with. Model on Q73's `not_salable` survey item (40138).

```yaml
itemid: 40142
name: A Rubbing of the Old Sky
namesimple: rubbing
description: >
  A charcoal rubbing taken from the great carving in the
  undercroft -- the old sky in four nested rings, where the
  temple above teaches three. Proof, in soft black line,
  of a count the Confluence stopped keeping. What you do
  with it is the only question left.
type: object
subtype: mundane
component_tag: sky-rubbing
weight: 0.2
value: 0
rarity_tier: 60
not_salable: true
```

- [ ] **Step 2: Add the Q74 triggers** to `quests/74-the_undercroft.yaml` (append to the existing `triggers:` list). Modeled on Q68. **(a) the reveal** (room_interact at the orbital-face, grants `74-reveal` + gives the rubbing); **(b) keepers branch** (give the rubbing to Aldric 9464); **(c) margin branch** (give the rubbing to the Margin scholar 9454). Each branch: `set_flag` + `give_gold` + two `bump_rep` + `grant: 74-end` + flavor:

```yaml
  # ── THE REVEAL: the orbital-face (6215, z-3 sealed chamber) ──
  - event: room_interact
    room: 6215
    noun: orbital-face
    conditions:
      has: ["74-descent"]
      missing: ["74-reveal"]
    actions:
      - grant: "74-reveal"
      - give_item: 40142
      - send_text: >-
          You read the carving the way the mason's survey taught you to read
          stone. Four rings, cut with an instrument's care, where every wall
          above keeps to three. The temple did not miscount -- it chose a
          count, and built three hundred years on the choice, and set its
          floor over the wall that remembers otherwise. There was a fourth.
          The sky the Founders found was not the sky we are taught. You take a
          rubbing of it -- proof in soft black line -- and the dark keeps the
          rest of its counsel. Whatever unmade the fourth, the stone does not
          say. That answer is somewhere else.
      - room_text: "takes a careful rubbing from the great carving on the chamber wall."
  - event: room_interact
    room: 6215
    noun: orbital-face
    conditions:
      missing: ["74-descent"]
    actions:
      - send_text: >-
          A vast carving of the night sky in nested rings, cut into the
          chamber's far wall by a very old hand. Without a reason to count
          them closely, it is only a beautiful, weathered thing in the dark.
  # ── COMPLETION, KEEPERS BRANCH: give the rubbing to Aldric (9464) ──
  - event: item_give
    mob: 9464
    item: 40142
    conditions:
      has: ["74-reveal"]
      missing: ["74-end"]
    actions:
      - set_flag: {key: "74-allegiance", value: "keepers"}
      - give_gold: 80
      - bump_rep: {faction: keepers, delta: 20}
      - bump_rep: {faction: margin, delta: -10}
      - npc_say:
          mob: 9464
          lines:
            - {delay: 1, text: "You took a rubbing. Of course you did."}
            - {delay: 4, text: "Then you understand what we have carried, and why some of us decided to carry it quietly. I will keep this with the others. The order will hold it -- and you, I think, will keep our confidence."}
      - send_text: >-
          You hand Aldric the rubbing. He looks at it a long moment, then folds
          it away with the care of a man filing something he has decided not to
          act on. You have kept the Keepers' confidence. What that costs, and
          what it is worth, will be yours to weigh.
      - grant: "74-end"
  # ── COMPLETION, MARGIN BRANCH: give the rubbing to the Margin scholar (9454) ──
  - event: item_give
    mob: 9454
    item: 40142
    conditions:
      has: ["74-reveal"]
      missing: ["74-end"]
    actions:
      - set_flag: {key: "74-allegiance", value: "margin"}
      - give_gold: 40
      - bump_rep: {faction: margin, delta: 20}
      - bump_rep: {faction: keepers, delta: -10}
      - npc_say:
          mob: 9454
          lines:
            - {delay: 1, text: "Is that -- it is. From below. They let you down, and you brought this up."}
            - {delay: 5, text: "Then it is real, and it is out, and it cannot be un-known now. The Margin will keep it where the temple cannot reorganize it. Thank you. You have no idea what this is worth -- or perhaps you do."}
      - send_text: >-
          You give the Margin scholar the rubbing. She holds it like something
          that might break, then like something that cannot. The truth is out
          of the temple's keeping now, in hands that will not let it go quiet.
          You have carried it into the light. Where it leads next is another
          road.
      - grant: "74-end"
```
Note both branches gate `has: [74-reveal], missing: [74-end]` and grant `74-end` — whichever the player does first completes Q74; the other can't fire (74-end now present). The flag + rep differ by branch. `give.go` transfers the rubbing to the NPC before the trigger; both NPCs keep it (no return) — correct (the player chose who gets the truth).

- [ ] **Step 3: Finalize the rewards block** (fires on `74-end`, both branches). Replace the 6a placeholder `rewards:` with the onward-pointing completion message (faction-neutral; the gold/rep are in the branch triggers):

```yaml
rewards:
  playermessage: >-
    The Undercroft kept its threshold and gave up its secret: there was a
    fourth, struck from the sky and the count and the record, and the temple
    was raised on the place that remembered. What unmade it is not here. The
    rubbing in your memory points away from the Confluence entirely -- north
    and older, to wherever the sky was changed. That road is for another day.
  roommessage: ""
  gold: 0
```
(The crash is NEVER named — "wherever the sky was changed" is the onward thread, threshold-only.)

- [ ] **Step 4: Self-check** — the reveal grants the declared `74-reveal` step; both branches grant the declared `74-end`; `74-allegiance` referenced with the questid prefix; `bump_rep` factions are `keepers` + `margin` (both faction files exist); item 40142 exists + `not_salable`; the orbital-face `room:`/`noun:` match 6215's noun; lore threshold-only (no crash/why) in every send_text/npc_say/reward.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40142-a_rubbing_of_the_old_sky.yaml _datafiles/world/dogmud/quests/74-the_undercroft.yaml
git commit -m "feat(confluence): Q74 completion - the reveal + allegiance branches + rewards"
```

---

## Task 5: Boot test + cartcheck + quest-load

- [ ] **Step 1: Wipe instances** (`rm -rf .../mobs.instances/* .../rooms.instances/*`).
- [ ] **Step 2: Build + boot** (`MapConsistencyEnforce: panic`). Wait for load; scan: clean load lines + `ValidateZoneConsistency errors=0 warnings=0 mode=panic`, `quests.LoadDataFiles` + `ValidateAllFlags` clean. Watch for: duplicate-coord panic; non-reciprocal/vertical-exit error across z-levels; room-title/mob-name casing panic; **quest panics** (unknown step — the reveal/end grants must be declared steps; bad bump_rep faction; bad room/noun); the guardian loading as a valid combat mob. Confirm no WARN/ERROR references a 6b id (6200–6217 / 9471 / quest 74 / item 40142).
- [ ] **Step 3: `cartcheck the_confluence`** — zero collisions; the three z-level stairs render as vertical. Kill the server.
- [ ] **Step 4:** Fix any failure; re-run; commit.

```bash
git add -A && git commit -m "fix(confluence): Undercroft boot/cartcheck/quest-load fixes"
```

---

## Task 6: World-critic + feel + FULL Q74 harness-verify (MANDATORY)

- [ ] **Step 1: World-critic (data review)** over 6199–6217 + the guardian + the Q74 triggers: **lore-boundary is the #1 check** (every room/noun/send_text/npc_say/reward must be threshold-only — no crash, gray material, mutation, numerology; "wherever the sky was changed" is the furthest the onward thread may go); direction/level canon (down is deeper; waters at z−2; reveal at z−3); vertical-exit reciprocity; the guardian reads as beatable+avoidable; casing; node/trigger correctness (both branches gate + grant 74-end; reveal grants 74-reveal).
- [ ] **Step 2: Feel + FULL Q74 harness-verify** (per `tools/playtest/`, local; kill all GoMud/go procs first). Fresh char: grant the Q73 chain + Q74 through descent (`questtoken 73-start/73-map/73-end`, then `74-start/74-ledger/74-record/74-survey/74-descent` — or play 6a naturally), then: descend 6199→...→6215; **fight the guardian** (confirm beatable for a developed char; confirm you can also walk past it down the stair without fighting); `look orbital-face` → `74-reveal` + receive the rubbing; then test **BOTH branches on separate runs** — `give rubbing to aldric` → `74-allegiance=keepers`, keeper rep +, `74-end`, quest 100% + onward message; and (fresh run) `give rubbing to scholar` (9454 at the portico) → `74-allegiance=margin`, margin rep +, `74-end`, 100%. Use `questtoken flags` to confirm the flag value each way. Save the report to `tools/playtest/reports/2026-06-27-local-feel-tester-confluence-undercroft.md`.
- [ ] **Step 3: Fix everything found**, re-boot, commit.

```bash
git add -A && git commit -m "fix(confluence): Undercroft feel/world-critic polish"
```

---

## Task 7: Finish the branch

- [ ] **Step 1:** Working tree clean + one final clean boot.
- [ ] **Step 2:** `superpowers:finishing-a-development-branch` — merge `feature/confluence-undercroft` into `master` `--no-ff`; delete branch. Do NOT push to prod.
- [ ] **Step 3:** Update memory: append the 6b outcome to `project_confluence_build.md` + the MEMORY.md index — **the Confluence spine (island + Q74) is COMPLETE**; note the verified Q68 branching-completion pattern (item_give + set_flag + bump_rep + grant end). Flag the remaining work: the outer quarters (Craftsmen's/Residential 6232–6245 + East Gate 6246–6257) to finish the city; the `74-allegiance` flag is recorded for any future use.
- [ ] **Step 4:** Fire the 6b background reviewers (adversarial + feel) per the standing SOP, then proceed to the outer quarters.

---

## Self-review (completed by plan author)

- **Spec coverage:** §2 geography/descent → T1+T2; §3 NPCs → T3; §4 guardian → T3; §5 Q74 completion (reveal + allegiance branches + rewards) → T4; §6 items/buffs → T4 (item; buff optional in T3); §7 schedules → none (undercroft unscheduled, noted); §8 verification → T5/T6. All covered.
- **Placeholder scan:** none — the reveal + both completion branches + the reward item + the guardian are given in full; ambient room prose is delegated with briefs.
- **ID/type consistency:** room ids/coords/exits cross-checked (T2 table reciprocal, 3 stacked verticals); the reveal `room:`/`noun:` (6215/`orbital-face`) match T2; guardian 9471 in 6208 spawninfo (T2) = T3; item 40142 (T4) referenced by the reveal give_item + both branch item_give; `74-reveal`/`74-end` are declared steps (6a); `74-allegiance` flag declared (6a), set by both branches; `bump_rep` factions `keepers`/`margin` both exist; completion fires on `74-end` (verified). Q68 is the structural model for the branch triggers.
