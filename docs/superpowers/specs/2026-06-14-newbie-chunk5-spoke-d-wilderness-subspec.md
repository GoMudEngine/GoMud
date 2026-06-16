# Newbie Rework — Chunk 5 Sub-Spec: Spoke D (Wilderness & Tracking)

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§6.3 spoke roster, §6.4 rings, §7.2 reward table, §5 Tier-2 forage/track/hunt/
> sleep lessons). Hub (C1), Spoke A Martial (C2), Spoke B Forge (C3), Spoke C
> Alchemy (C4) are built + verified + committed. This is the **fourth spoke**.
> Same phase-gate workflow: **rooms+nouns → REVIEW → mobs+items → REVIEW →
> dialogue+quests (inline) → REVIEW**.

## 0. Decisions locked (user, 2026-06-14)

1. **Boss = canine pack-leader** — an alpha at the head of a predator pack
   (dire/scab-hound type). The arc: track the pack, thin it, take the alpha.
2. **Crafting nod = BOTH** — hunting yields hides (→ a new **tailoring** garment
   recipe = the "wilderness garment" reward) AND meat (→ a **cooking** meal
   recipe). Two recipe rewards across the spoke.
3. **Combat signature = pack fights** — the outer ring throws multiple predators
   at once (teaches positioning, `flee`, AoE via thrown items, and why you rest/
   heal between fights). Difficulty untuned per
   [[feedback-defer-tuning-to-post-build-playtest]].

## 1. Scope

The wilderness / fieldcraft tutorial spoke. It runs **south / south-west** from
the already-built hub stub **5223 (Scrub Draw)** at (43,1,0) — out of the coulee
proper onto open **scrub steppe**, into **predator fringe**, and finally into the
pack's **predator territory / den**. Three concentric rings teach the field loop:

| Ring | Risk | Teaches | Cert reward (§7.2) |
|---|---|---|---|
| **Inner** (scrub steppe) | None — sanctuary | `track <quarry>` (follow a trail to its maker), biome `forage` on the steppe, reading sign | tracking seeds (Search rank) + scout gear |
| **Middle** (predator fringe) | Real — can KO you | hunting (track → kill prey → gather **hide + meat**), **`sleep` in the field** (rest for fast regen between fights; wake triggers), cooking the meat | Search rank + slot-filler stamina gear + a **cooking** recipe |
| **Outer** (predator territory / the den) | Pack fights + a boss | fighting **multiple predators at once** (flee, throw, positioning), then the alpha | **Perception bump** + a **wilderness garment** (crafted) + a **hunting kit** + the tailoring recipe |

**Theme:** A taught you to fight, B to make, C to sustain; D teaches you to
*live off the land* — read sign, run quarry down, rest rough, and turn the kill
into gear and food. The capstone pays the loop off: you track the pack to its
den, break it, and walk out wearing what you hunted.

Both `track` and `forage` advance the **Search** skill (Perception-primary), so
the spoke's skill rewards are `search:N` and its stat capstone is Perception —
one coherent progression line. Out of scope: a dedicated "hunt" command (there
is none — hunting is track→find→kill), deep survival sim, weather survival
(Tier-3 tease only).

## 2. ID allocations (DRAFT — verify with `tools/id_inventory.py` at Phase R)

> **LESSON from C4:** run `id_inventory.py` BEFORE picking item ids — the C4
> capstone potion collided with an existing 30060. Pin every new item/mob/room/
> quest id against the scanner at Phase R, in a popup-tolerant window.

| Type | Spoke D block (after Spoke C's 5264-5281 / 9128-9135 / 38-40 / 30065) |
|---|---|
| Rooms | **5282–5301** (~20) |
| Mobs | **9136–9151** (pack needs several hounds + alpha + 2 questgivers) |
| Items | new: scout-gear piece, stamina-gear piece, the wilderness garment, a meal, + a pack-hide material — **type-range correct** (armor by slot sub-dir, consumables 30xxx, materials 40070+), ids verified at Phase M |
| Quests | **41–43** (inner cert / middle rank / outer cert) |
| Dialogue | files by mobid |
| Recipes | 1 new **tailoring** (pack-hide pelt + sinew → the garment item) + GRANT existing **cooking** `hearty-stew` (no new cooking file) |
| New materials | 1 **pack-hide pelt** (pack drop, 40070+) — meat reuses raw-meat 40014 / wild-hare-meat 40064 |

**Coordinate reserve.** Hub stub **5223 (Scrub Draw) at (43,1,0)** exits east to
hub 5216; the spoke radiates **south/south-west**. It must thread between the
already-built neighbours: Spoke B Forge (x31-41, y-6..6), Spoke A (x42-57,
y-3..3), the hub (y≤~3), and Spoke C (x46-48, y3-14). The clean lane is **the
southern band: roughly x[37..45], y[7..14], z0** (south of B's y-reach and A,
west of C). Inner ring sits just south of the stub; outer den can dip the
southern/western corner. `coord_inventory.py` gates 0 collisions; layout
hand-verified cartesian-clean at Phase R (the C-budget doc said D = south/SW —
consistent).

## 3. Ring sketch (rooms 5282–5301, ~20) — detailed manifest at Phase R

- **Inner — scrub steppe (sanctuary, ~5):** the draw opens onto dry steppe; a
  scout/tracker NPC + cert quest 41; track-a-critter lesson (a harmless steppe
  hare leaves a trail to follow); steppe `forage`; sanctuary ends at the fringe.
- **Middle — predator fringe (no sanctuary, ~6-7):** scattered prey + first lone
  predators; the hunter NPC + quest 42; the **sleep-in-field** beat (a safe-ish
  hollow to rest); gather hide+meat from kills; a campfire/cook spot.
- **Outer — predator territory / the den (no sanctuary, ~7-8):** escalating
  **pack** encounters (2-3 hounds per room), a mini-boss (a scarred outrider),
  then the **alpha pack-leader** in the den; post-boss reward vista + lateral-
  connector stub (toward Spoke C's 5281 / future ring).

Biomes: steppe = `plains` or `land`; fringe/territory = `land`/`forest`/`hills`
as fits (all outdoor — weather visible, Tier-3 tease). Every painted feature
look-able (noun-token rule, `newbie_manifest_check.py`).

## 4. Mob sketch (9136–9151) — detailed manifest at Phase M

- **2 NPCs (Opened):** **Scout Tarn** (inner, quest 41 — teaches track/forage)
  and **Hunter Delk** (middle, quests 42+43 — teaches hunt/sleep/cook). Names
  cleared vs roster (0 hits) + novel list; Grieve reserved. Re-verify at Phase M.
- **Prey (non-aggressive or weak):** a steppe hare (the track-lesson quarry;
  harmless) + a forageable prey animal that drops meat/hide.
- **Predators (the pack):** scab-hound (weak, comes in 2-3s), a tougher
  fringe-stalker, a scarred **outrider** mini-boss, and the **alpha pack-leader**
  boss (`tank_taunter`, statpool ~200, "Velk-lite"). Several hound spawns so
  outer rooms field genuine pack fights.
- Reuse existing combat archetypes (`behaviors/archetypes/`); statpools mirror
  prior spokes' curve (weak ~18 → boss ~200), **untuned** (defer to playtest).
  Species: canine (2) for hounds/alpha; rodent/deer/boar for prey.

## 5. Quest sketch

- **Quest 41 — inner cert (Scout Tarn):** learn to read sign — `track` the steppe
  hare to where it's hiding, `forage` the steppe. Reward: `skillinfo search:1`
  (tracking seeds) + a scout-gear item (`itemid`) + gold.
- **Quest 42 — middle rank (Hunter Delk):** hunt — `track` then kill prey, gather
  its **hide + meat** (raw-meat 40014 / wild-hare-meat 40064), learn to **`sleep`**
  in a safe hollow to recover, and `cook` the meat. Reward: `skillinfo search:1` +
  a slot-filler stamina garment + `recipe_info: hearty-stew` (the cooking meal —
  skill 5, raw-meat + wild-hare-meat + wild-vegetables) + gold. (Engine touch §6:
  `sleep` command Notify so the rest beat can be a quest step.)
- **Quest 43 — outer cert (Hunter Delk):** track the pack to its den, thin the pack,
  defeat the **alpha pack-leader**. Reward (§7.2): `stat_info perception:3` + the
  **wilderness garment** (`itemid` the crafted-tier garment, or grant via the new
  tailoring recipe) + a **hunting kit** (`item_info` multi — e.g. trail rations,
  bandages, a spare throwable) + `recipe_info` (the new **tailoring** pack-hide
  garment recipe) + gold.

Repeatable: the inherent hunt→gather→cook/sell loop + respawning pack (matches
prior spokes — no separate repeatable quest file). No re-grant bugs (every
`grantsQuest` carries the `{id}-end` token in `questExcluded`); reward keys use
the no-underscore spelling for tag-less fields and the exact tagged keys
(`stat_info`/`recipe_info`/`item_info`) ([[reference-quest-reward-yaml-key-gotcha]]).

## 6. Engine touches — ONE tiny one + checks

Almost everything reuses existing, verified machinery:
- `track` (Search skill) ALREADY emits `questengine.Notify("command")`. `forage`
  does too. The reward fields `skillinfo`/`stat_info`/`itemid`/`recipe_info`/
  **`item_info`** (the multi-item reward built in C4) all exist and are verified.
- **THE ONE TOUCH: `sleep` does NOT emit the command Notify** (verified — 0 calls
  in `sleep.go`). To make the Quest-42 "rest in the field" beat a real step, add
  the same 1-line `questengine.Notify("command", … Command:"sleep")` we added to
  drink/throw. Reusable; trivial. (Alternative: teach sleep via dialogue only and
  skip the quest step — but the parent spec calls sleep-in-field a Tier-2 lesson,
  so a beat is worth the 1-liner.)
- **Build-time checks:** confirm `cook`/`craft` notify (craft does); confirm the
  prey actually drop hide+meat (carried-item or loot) so Q42 gathering works;
  create the new tailoring garment recipe (+ its output item) and the cooking
  meal (new, or grant an existing like `grilled-meat`/`trail-rations`); add a
  pack-hide material if no existing hide fits the garment recipe.

## 7. Lesson coverage (Tier-2 wilderness beats owned by Spoke D)

| Lesson | Where |
|---|---|
| `track <quarry>` (follow sign) | scout, Q41 |
| biome `forage` on the steppe | inner/middle |
| hunting: track → kill → gather hide+meat | hunter, Q42 |
| `sleep` in the field (fast regen, wake triggers) | Q42 (the §6 engine touch) |
| `cook` the kill (meal buff) | Q42 reward recipe |
| pack combat (multiple foes; flee/throw/positioning) | outer ring, Q43 |
| Perception via a cert | Q43 (`stat_info`) |
| a hunting kit (multi-item) | Q43 (`item_info`) |
| crafting a wilderness garment from hides | Q43 reward (tailoring recipe) |

## 8. Acceptance criteria

1. Boots clean (rooms/mobs/quests/recipes/items counts up; flags validate; 0 panic).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean.
3. `newbie_manifest_check.py` extended for Spoke D rooms + mobs.
4. A character can: enter from the hub → track the hare + forage (Q41) → hunt
   prey, gather hide+meat, sleep to recover, cook a meal (Q42) → track the pack,
   win the pack fights, beat the alpha → claim Perception + the crafted garment +
   the hunting kit + the recipes (Q43). Verified by a **scripted mechanics
   walkthrough** confirming `track`/`sleep`/`cook` fire their quest steps and
   every reward lands (save = ground truth, the C4 method).
5. Difficulty NOT an acceptance gate — reasonable statpools, defer to playtest.

## 9. Build-phase breakdown

- **Phase R — rooms+nouns:** ~20 rooms 5282-5301, wire 5223 south, sanctuary on
  inner only, biomes, nouns, coords (hand-verified clean, then authoritative
  `coord_inventory`/`cartcheck`/boot in a popup window). REVIEW.
- **Phase M — mobs+items:** 2 NPCs + prey + the pack + alpha; the garment item +
  meal item + any pack-hide material; spawninfo (pack density in outer rooms);
  statpools (untuned). Boot + manifest. REVIEW.
- **Phase D — dialogue+quests (INLINE):** scout + hunter dialogue, quests 41-43 +
  the `sleep` Notify engine touch + the tailoring & cooking recipes, scripted
  walkthrough. REVIEW (= chunk complete).

## 10. Open items — RESOLVED (user, 2026-06-14)

1. **NPC names — DECIDED:** **Scout Tarn** (inner, Q41 — teaches track/forage)
   and **Hunter Delk** (middle+outer, Q42/Q43 — teaches hunt/sleep/cook + the
   pack). Both cleared vs roster (0 mob-file hits) and on the pre-vetted novel
   list; **Grieve** held in reserve. Re-verify at Phase M as SOP.
2. **Cooking reward recipe — DECIDED: grant the existing `hearty-stew` recipe**
   (skill_minimum 5, so NOT born-known — a real reward; ingredients raw-meat +
   wild-hare-meat + wild-vegetables, all huntable/forageable). NO new cooking
   item/recipe. (`grilled-meat`/`trail-rations` are skill_minimum 0 = already
   known → granting them would silently no-op, so they're out.) Meat reuses
   raw-meat (40014) + wild-hare-meat (40064) as prey drops.
3. **Garment delivery — DECIDED: BOTH.** Q43 grants the finished garment as an
   `itemid` AND the new tailoring recipe via `recipe_info`. New content for this:
   one **pack-hide pelt** material (pack drop), one **tailoring garment recipe**
   (pelt + sinew → garment), one **garment item** (the wilderness garment).
