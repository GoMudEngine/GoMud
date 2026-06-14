# Newbie Rework — Chunk 4 Sub-Spec: Spoke C (Herbalism & Alchemy)

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§6.3 spoke roster, §6.4 ring structure, §7.2 reward table, §5 Tier-2
> "shared cooldown" + "consumables" + "skullduggery/throw" lessons). Hub
> (chunk 1), Spoke A Martial (chunk 2), and Spoke B Forge (chunk 3) are built +
> verified. This is the **third spoke**. Same phase-gate workflow as A/B:
> **rooms+nouns → REVIEW → mobs+items → REVIEW → dialogue+quests (inline) →
> REVIEW**.

## 0. Decisions locked (user, 2026-06-14)

1. **Stockpile reward = BUILD a reusable multi-item reward in the engine**
   (mirrors the `stat_info` / `recipe_info` precedent — "build it in the engine,
   we'll need it later"). New tagged `item_info` reward field + a `parseItemGrants`
   hook. See §6.
2. **Boss = flavor-only straight combat** (like Spokes A/B). Alchemy is strong
   *prep* (brew healing salves, buffs, grenades and carry them in) but is **not
   mechanically required** — no forced toxicity/antidote puzzle. Simpler to tune,
   consistent with the other spokes' boss feel.
3. **Grenade + shared-cooldown beat = mid-ring side lesson** (NOT the boss
   finisher). Taught earlier, in the middle ring against a lesser foe, so the
   capstone stays a clean combat/consumable test. Spoke C still *owns* this
   parent-spec teaching beat (§5 Tier-2) — it just lands in the marsh, not the
   mire.

## 1. Scope

The herbalism / alchemy tutorial spoke. It descends **south-east** from the
already-built hub stub **5222 (Reedwash Mouth)** — a reed-choked seep where the
hub's spring water drains away — down through a **sheltered plunge pool**, out
into a **reedy marsh**, and finally into a sunken **poison swamp** where a
swamp-spirit aberrant has gathered. Three concentric rings teach the brewing
loop:

| Ring | Risk | Teaches | Cert reward (§7.2) |
|---|---|---|---|
| **Inner** (the plunge pool) | None — sanctuary | `recipes` / `craft list` / `craft <recipe>`, the **`alchemy_bench`** station requirement, the material loop (buy/forage herbs + a bottle → brew), `drink` | herbalism/alchemy seeds + starter bottles (+ first recipes) |
| **Middle** (the reedy marsh) | Real — can KO you → wake at the Mending Hut | biome-gated `forage` (swamp/water herbs), **`throw <grenade>` + the shared special-move cooldown**, the **potion bandolier** (auto-route + drink-oldest-first) | alchemy rank bump + a **potion bandolier** + a recipe |
| **Outer** (the poison swamp) | Genuine challenge — sunken, descending, a boss | brewing/bottling your own consumables, filling the bandolier, then USING them | **Vitality bump** + a **granted potion stockpile** + an advanced recipe |

**Theme:** Spoke A taught you to *fight* with what you're handed; Spoke B taught
you to *make* what you fight with; Spoke C teaches you to *sustain* yourself —
brew the salves, antidotes, buffs and grenades that keep you alive in the field.
The capstone pays off the loop: you bottle your own kit at the pool and carry it
into the mire.

Out of scope: aging-optimization play (Tier 4), the deep toxicity economy
(mentioned/flavored only — Tier 2 "your gut churns"), bottle-tier min-maxing
(teased as glass-craft quality), the wider potion catalogue. Every command used
here already ships (`craft`, `recipes`, `forage`, `drink`, `throw`, `sort`) —
see §6 for the one real engine build (the multi-item stockpile reward).

## 2. ID allocations (DRAFT — verify with `tools/id_inventory.py` at Phase R)

> Spoke B consumed rooms 5244–5261, mobs 9116–9123, quests 35–37, weapon 10045,
> material 40069, armor 20088. Spoke C blocks pick up after those. The
> authoritative `id_inventory.py` run is batched to a popup-tolerant window
> (no-console-popup constraint) before the Phase-R gate closes; these are the
> reserved blocks.

| Type | Spoke C block |
|---|---|
| Rooms | **5264–5283** (~18 used; rest spare) |
| Mobs | **9128–9139** |
| Consumables (1 new capstone potion) | **30060+** |
| Quests | **38–40** (inner cert / middle rank+grenade / outer cert) |
| Dialogue | files by mobid (9128.yaml, 9129.yaml) |
| Recipes | 1 new advanced recipe (`swamp-renewal-elixir.yaml` or similar) |
| **Reused, NOT new** | bottles (40006/40043-45), herbs (40004/40005/40009/40055-57), bandolier (20059), grenades (30057-59), potions (30028/30036…), recipes (healing-salve, minor-antidote, firebomb, …) |

**Coordinate reserve.** Hub stub **5222 (Reedwash Mouth) sits at (47,2,0)**
(chunk-1 manifest), attached to hub room 5207. Spoke C lays out **SOUTH /
south-east: x[45..48], y[3..14], z[-1..0]** — inside the Pothole reserve
x[30..59], south of Spoke A (x42-57 **y-3..3**) and the hub (y≤2), east of Spoke
D's reserved arm (x33-45) and clear of Spoke B (x31-41). The poison-swamp outer
ring **descends to z-1** (a sunken mire). `tools/coord_inventory.py` gates the
chunk at 0 collisions; `cartcheck pothole_coulee` verifies intra-zone geometry.

## 3. Room manifest (18 rooms, 3 rings) — coords hand-drafted cartesian-clean

> 5222 (Reedwash Mouth, **already built**, sanctuary stub at (47,2,0)) is the
> threshold; this chunk wires its **SOUTH** exit to 5264. Conventions: north=y-1,
> south=y+1, east=x+1, west=x-1, up=z+1, down=z-1. Every coord is unique and
> every exit's delta matches its direction (hand-checked; authoritative
> `coord_inventory.py`/`cartcheck` run batched to the popup-tolerant window
> before the Phase-R gate). Existing Pothole rooms occupy x42-57 (A+hub) and
> x31-41 (B); Spoke C sits at x45-48 / y3-14 (no overlap — different y-band from
> A). Inner z0; outer mire descends z-1.

### Inner ring — the sheltered plunge pool (sanctuary; trivial) — 5 rooms [5264–5268]
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5264 | Reedwash Descent | (47,3,0) | N→5222, S→5265 | the path down from the stub; reeds, draining spring water |
| 5265 | The Sheltered Pool | (47,4,0) | N→5264, S→5266, W→5267, E→5268 | **`station: alchemy_bench`**; herbalist NPC + cert quest 38; pool/water/reeds nouns |
| 5266 | Mossy Verge | (47,5,0) | N→5265, S→5269 | sanctuary ENDS here; herbalist warns + teaches `sethome coulee` |
| 5267 | Drying Racks | (46,4,0) | E→5265 | the herbalist's stall: bottles, healers-root, dustwalk-herb (buy-materials beat); **`station: alchemy_bench`** #2 for the repeatable brew |
| 5268 | Still Shallows | (48,4,0) | W→5265 | water-biome `forage` intro (clams/lake-mint); look-able shallows |

### Middle ring — the reedy marsh (NO sanctuary; can KO) — 6 rooms [5269–5274]
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5269 | Reed Beds | (47,6,0) | N→5266, S→5270, E→5271 | first live foe (a weak marsh creature) |
| 5270 | Sunken Path | (47,7,0) | N→5269, S→5273 | hazard footing; swamp-biome `forage` (cattail-down) |
| 5271 | Cattail Stand | (48,6,0) | W→5269, S→5272 | middle-ring quest NPC (a marsh-dweller, Opened); the **grenade lesson** is granted here |
| 5272 | Mudflat | (48,7,0) | N→5271, W→5273 | the **grenade-lesson foe** (reed lurker); armor/loot drop |
| 5273 | Black Pool | (47,8,0) | N→5270, E→5272, S→5274 | repeatable forage/cull hub (marsh-willow-bark, lake-mint) |
| 5274 | The Miasma Edge | (47,9,0) | N→5273, S→5275 | last middle room; the water turns foul; descent into the mire begins |

### Outer ring — the poison swamp (challenge + boss; descends z-1) — 7 rooms [5275–5281]
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5275 | Fenrot Approach | (47,10,0) | N→5274, S→5276 | first poison-swamp denizen; toxicity-flavored fumes (no mechanic — flavor) |
| 5276 | Sulphur Flats | (47,11,0) | N→5275, S→5277, D→5278 | crew fight (2); side-branch down into the sunken pocket |
| 5277 | Drowned Thicket | (47,12,0) | N→5276 | loot + a tougher foe; dead-end pocket |
| 5278 | Sunken Hollow | (47,11,-1) | U→5276, S→5279 | sunken (z-1); a lieutenant-tier denizen |
| 5279 | The Black Mere | (47,12,-1) | N→5278, S→5280 | approach to the boss; tension beat |
| 5280 | Heart of the Mire | (47,13,-1) | N→5279, S→5281 | the **boss**: a spirit-of-the-swamp aberrant |
| 5281 | Clearwater Spring | (47,14,-1) | N→5280 | post-boss reward vista (the mire clears to a clean spring); lateral-connector stub (future → Spoke D) |

Coord set (all unique): inner (47,3,0)(47,4,0)(47,5,0)(46,4,0)(48,4,0); middle
(47,6,0)(47,7,0)(48,6,0)(48,7,0)(47,8,0)(47,9,0); outer (47,10,0)(47,11,0)
(47,12,0)(47,11,-1)(47,12,-1)(47,13,-1)(47,14,-1). Lateral connector stub on
5281 (parent §6.1). Biomes: plunge pool `water` (inner) / pool-edge rooms may use
`swamp`; marsh `swamp`; mire `swamp` (z-1 sunken). The stall/herbalist interior
can be `house` if walled, else open-air `swamp`. Every painted feature
look-able (noun-token rule enforced by `newbie_manifest_check.py`).

## 4. Mob manifest (9128–9139) — DRAFT

| Id | Name (clears novel + roster — VERIFY at Phase M) | Room | Archetype | Hostile | Role |
|---|---|---|---|---|---|
| 9128 | Herbalist (TBD name) | 5265 | noncombat_questgiver | no | teaches brew loop; grants quest 38 + starter recipes + bottles |
| 9129 | Marsh-dweller (TBD name) | 5271 | noncombat_questgiver | no | middle-ring quest giver; grants the grenade + bandolier lesson (Q39) |
| 9130 | Bog Crawler / Marsh Leech | 5269/5273 | (weak fighter) | yes | first live foe; low statpool |
| 9131 | Reed Lurker | 5272 | (weak fighter) | yes | the grenade-lesson foe; armor drop |
| 9132 | Bog Stalker | 5275/5276 | (fighter) | yes | outer denizen |
| 9133 | Mire Brute | 5276/5277 | (fighter, tankier) | yes | outer crew |
| 9134 | Fen-Touched Lurker | 5278 | (fighter) | yes | mini-boss before the spirit |
| 9135 | **Spirit of the Swamp** | 5280 | (boss fighter, `tank_taunter`) | yes | the spoke boss (a swamp aberrant) |
| 9136–9139 | spare (extra crawlers / flavor) | — | — | — | reserve |

Hostile mobs reuse existing combat archetypes (`behaviors/archetypes/`) — no new
btrees unless the boss wants a scripted beat. **Difficulty NOT tuned here per
[[feedback-defer-tuning-to-post-build-playtest]]** — reasonable starting
statpools (crawler ~18, lurker ~40, stalker/brute ~90, mini ~130, spirit ~200,
`tank_taunter` "Velk-lite"), flagged for the user's post-build evening playtest.

The two NPCs (herbalist + marsh-dweller) are **Opened** — each gets one
understated visible mutation in its description (per canon). Names picked at
Phase M against the novel + roster (candidates clearing both, TBD).

## 5. Quest manifest

**Quest 38 — "First Brew" (inner, cert).** The herbalist grants on `ask
<herbalist> brew`/`learn`/`quest`/`task`. Beats: see your recipes (`recipes` /
`craft list`), get the ingredients (buy a **bottle** + **healers-root** from the
stall, or `forage` them at the pool), `craft healing salve` at the
`alchemy_bench`, then `drink` one. Reward: alchemy seeds (`skillinfo alchemy:1`)
+ a few starter **bottles** (`item_info`, see §6) + the **starter recipes**
(healing-salve, minor-antidote via `recipe_info` — built in Spoke B). Closes
teaching `sethome coulee` + pointing south to the marsh.

> **`craft <recipe>` name gotcha (carried from Spoke B):** `FindRecipeByName`
> matches the **display name with spaces** ("healing salve"), not the hyphenated
> id. All dialogue hints + quest beats say `craft healing salve`, never
> `craft healing-salve`. And `craft list`, not `recipes`-as-a-command (the
> command is `craft list`; `recipes` exists but B standardized on `craft list`).

**Quest 39 — "Wade the Marsh" (middle, rank + the grenade/cooldown lesson).**
The marsh-dweller grants. Beats: defeat N marsh foes (crawler/lurker) AND
`forage` swamp herbs (cattail-down / lake-mint); **the grenade beat lands here**
— the NPC hands you one grenade (`givesItem` a firebomb 30058), teaches
`throw <foe>`, and teaches that **throwing a grenade, casting, and special moves
all share one cooldown** ("you can only loose one trick per few breaths"), used
against the reed lurker (9131). Reward: alchemy rank bump (`skillinfo`) + a
**potion bandolier** (`itemid: 20059`) + a recipe (the **firebomb** recipe via
`recipe_info` — you threw one, now learn to brew them). Teaches the bandolier
(auto-route on pickup + `drink` consumes oldest-first) + `throw` + the shared
cooldown (Spoke C's mandated §5 beat).

**Quest 40 — "Heart of the Mire" (outer, cert).** Granted after 39 / on entering
the poison swamp. Beats: descend into the mire, beat the denizens + the
fen-touched lurker, and defeat the **Spirit of the Swamp (9135)** in the Heart of
the Mire. Reward (§7.2): a **Vitality bump** (`stat_info: vitality:3` — the
engine stat-grant from Spoke A, already exists) + a **granted potion stockpile**
(the NEW `item_info` multi-item reward, §6 — e.g. 3× healing salve + 2× minor
antidote + 2× firebomb) + an **advanced recipe** (a new spoke-unique swamp
elixir via `recipe_info`). Design beat: brew + bottle your own kit at the pool,
fill the bandolier, and carry it down into the mire.

Inner repeatable: re-brew at the Drying Racks bench (skill-use, no gold). Middle
repeatable: a marsh herb-bounty / cull (gold). No re-grant bugs — every
`grantsQuest` node carries the `{id}-end` token in `questExcluded` (SOP).
**Reward-block keys use the no-underscore spelling** for the tag-less fields
(`itemid`, `skillinfo`, `playermessage`) and the **exact tagged keys**
(`stat_info`, `recipe_info`, and the new `item_info`) for the tagged ones
([[reference-quest-reward-yaml-key-gotcha]]).

## 6. Engine touches — ONE real build (your decision #1) + checks

Most of the loop already ships and is verified:
- `craft <recipe>` / `craft list`, the `station:` room check (proved with
  `station: forge` in Spoke B — `alchemy_bench` is just the recipe's declared
  station string; **build-time check: confirm the craft station check is generic
  and honours `alchemy_bench`**), `forage` (swamp/water biomes already yield all
  needed herbs — **NO forage-table change**, unlike Spoke B), `drink`, `throw`,
  `sort`, the potion bandolier (item 20059) — all exist.
- The **Vitality bump** for Q40 uses the `stat_info`/`train_stat` machinery from
  Spoke A. The **starter + advanced recipe grants** use `recipe_info` from Spoke
  B. No new engine work for those.
- **Quest `command:` Notify check (carried from Spoke A):** quest `command:`
  triggers only fire when the handler calls `questengine.Notify("command", …)`.
  `craft` does. **Confirm `forage`, `drink`, and `throw` do too**; if any
  doesn't, add the 1-line Notify (same pattern we added to consider/kick/trip/set
  in Spoke A). Q38 (`craft`/`drink`), Q39 (`forage`/`throw`) depend on this.

**THE BUILD — multi-item stockpile reward (decision #1: build it in the
engine).** `QuestReward` grants exactly one item (`ItemId int`, no quantity).
Spoke C's outer reward (§7.2) is a *potion stockpile*. Mirror the `stat_info` /
`recipe_info` precedent exactly:
  1. Add a tagged field `ItemInfo string `yaml:"item_info,omitempty"`` to
     `QuestReward` (`internal/quests/quests.go`) — format
     `"itemid:qty[,itemid:qty,…]"`, e.g. `"30036:3,30028:2,30058:2"`.
  2. Add a `parseItemGrants` hook alongside `parseStatGrants` / `parseRecipeGrants`
     (`internal/hooks/Quest_HandleQuestUpdate.go`) that, on quest completion,
     creates N of each item and stores them (respecting bandolier/component-bag
     auto-routing via the normal `StoreItem` path), with a floor-guard on bad
     input like the skill/stat parsers.
  3. Player-facing: one descriptive line (no raw counts beyond the item names —
     "You receive a parcel of brewed supplies." is fine; itemizing names is OK,
     it's not a balance number).
  4. Tests mirroring the stat/recipe grant tests (parse + apply + a malformed
     input case). Reusable for every future "starter kit" reward.
  - *Optional / defer:* a `give_items` **trigger action** (mid-quest multi-give)
    — NOT needed for Spoke C (the stockpile is an end-reward; the single mid-ring
    grenade uses `givesItem`). Skip unless a later spoke wants it.

**New content files (Phase D/M):** 1 capstone potion item (consumables 30060+)
as the advanced recipe's output, and 1 advanced recipe
(`recipes/alchemy/<name>.yaml`, `station: alchemy_bench`, built from existing
swamp herbs — lake-mint / marsh-willow-bark — so no new material). Everything
else is reuse.

## 7. Lesson coverage (Tier-2 alchemy + consumable + throw beats owned by Spoke C)

| Lesson | Where |
|---|---|
| `recipes` / `craft list` | herbalist (Q38) |
| `craft <recipe>` + the `alchemy_bench` station requirement | the pool (Q38) |
| the material loop (buy bottles/herbs at the stall, or forage them) | Drying Racks + pool (Q38) |
| `drink` a potion | the pool (Q38) |
| biome-gated `forage` (swamp + water herbs) | marsh / shallows (Q39) |
| **`throw <grenade>` + the shared special-move cooldown** | reed lurker, middle ring (Q39) — **Spoke C's mandated §5 beat** |
| potion bandolier (auto-route on pickup + drink-oldest-first) | Q39 reward + lesson |
| toxicity ("your gut churns") | flavored in the poison swamp + when stacking potions (tease, no forced mechanic) |
| bottle tiers (clay→glass→sealed→crystalline as glass-craft quality) | herbalist tease (Q38) |
| Vitality via a cert | Q40 reward (`stat_info`) |
| a granted potion stockpile | Q40 reward (the new `item_info` multi-item grant) |

## 8. Acceptance criteria

1. Boots clean (mobs/quests/recipes/items loadedCount up by the new counts; 0
   panics; flag refs validate).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean.
3. `newbie_manifest_check.py` extended for Spoke C rooms + mobs (noun-token rule).
4. A character can: enter from the hub → buy/forage herbs + a bottle and `craft
   healing salve` at the pool, then `drink` it (Q38) → wade the marsh: forage
   swamp herbs, `throw` a granted grenade and feel the shared cooldown, get the
   bandolier (Q39) → bottle a kit, descend the mire, beat the Spirit of the Swamp
   → claim the Vitality bump + potion stockpile + advanced recipe (Q40). Verified
   by a **scripted mechanics walkthrough** (the A/B method): confirm recipes are
   granted, `craft`/`forage`/`throw`/`drink` complete and notify the quest engine,
   the bandolier auto-routes, and **every reward lands on the sheet — including
   the new multi-item stockpile.**
5. **Difficulty is NOT an acceptance gate** — set reasonable statpools and defer
   to the user's post-build playtest tuning pass
   [[feedback-defer-tuning-to-post-build-playtest]].

## 9. Build-phase task breakdown (per the phase gates)

- **Phase R — rooms + nouns:** ~18 rooms (5264–5283), wire 5222 S→5264,
  sanctuary on the inner ring only, `station: alchemy_bench` on 5265 (+5267), the
  mire `swamp` biome + z-descent (z-1) in the outer ring, water-biome plunge
  pool, nouns, coords. Audits (`coord_inventory`, `cartcheck`, manifest checker
  extended) + walkthrough → **REVIEW gate.**
- **Phase M — mobs + items:** herbalist + marsh-dweller NPCs (Opened, names
  cleared) + marsh/mire foes + the spirit boss; spawninfo; the 1 new capstone
  potion item; statpools (untuned). Reuse bandolier 20059, grenades 30057-59,
  existing herbs + bottles. Boot + manifest → **REVIEW gate.**
- **Phase D — dialogue + quests (INLINE):** herbalist + marsh-dweller dialogue
  (brew/forage/throw/bandolier teaching, voice-matched, Opened), quests 38–40 +
  repeatables, **the multi-item-reward engine build (§6) + the Notify checks +
  the new advanced recipe**. Scripted mechanics walkthrough verifying all rewards
  (esp. the stockpile) → **REVIEW gate (= chunk complete).**

## 10. Open items for you (flagged, not blockers)

1. **NPC names** — picked at Phase M against the novel + roster. (Spoke A/B
   precedent: Vorn, Garve, Rusk, Ovell; candidates here TBD, cleared against the
   novel.)
2. **Advanced recipe identity** — proposed: a spoke-unique swamp regen elixir
   built from lake-mint + marsh-willow-bark (output the new 30060 potion). Open
   to a different capstone if you have one in mind.
3. **Which grenade for the mid lesson** — draft uses **firebomb (30058)** (a
   damage throwable, so the lesson works against the reed lurker). Flashbang
   (30057) is the cleaner pure-cooldown demo but does no damage; toxic-flask
   (30059) fits the swamp flavor. Easy to swap.
