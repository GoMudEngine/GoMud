# Newbie Rework — Chunk 3 Sub-Spec: Spoke B (Forge)

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§6.3 spoke roster, §6.4 ring structure, §7.2 reward table). Hub (chunk 1)
> and Spoke A Martial (chunk 2) are built + verified. This is the second spoke.
> Same phase-gate workflow as Spoke A: **rooms+nouns → REVIEW → mobs+items →
> REVIEW → dialogue+quests (inline) → REVIEW**.

## 1. Scope

The smithing / crafting tutorial spoke. It climbs **west** from the already-built
hub stub **5221 (Talus Gap)** — a talus-strewn draw that smells of quenched coal
— up to a working **smithy**, out across a **talus slope**, and down into a
**basalt mine shaft** where something stone-blooded has denned. Three concentric
rings teach the crafting loop:

| Ring | Risk | Teaches | Cert reward (§7.2) |
|---|---|---|---|
| **Inner** (the smithy) | None — sanctuary | `recipes` / `craft list` / `craft <recipe>`, the forge-station requirement, the material loop (buy/own ingots → craft) | smithing seeds + an ingot starter (+ first recipes) |
| **Middle** (the talus slope) | Real — can KO you → wake at the Mending Hut | the component bag (auto-route + `sort`), `salvage <item>`, gathering local ore | smithing rank bump + a **component bag** + a recipe |
| **Outer** (the mine shaft) | Genuine challenge — dark, descending, a boss | crafting your own gear to win, then using it | **Strength bump** + a forged weapon + an advanced recipe |

**Theme:** Spoke A taught you to *fight* with what you're handed; Spoke B teaches
you to *make* what you fight with. The capstone deliberately pays off the loop —
you smith a weapon at the inner forge and carry your own work down into the mine
to beat the boss.

Out of scope: deep crafting economy, enchanting (own spoke later / city), multi-
station chains. Every command used here already ships (`craft`, `recipes`,
`salvage`, `sort`) — but see §6 for the one real engine gap (granting recipes).

## 2. ID allocations (verified `tools/id_inventory.py`, 2026-06-13)

| Type | Spoke B block |
|---|---|
| Rooms | **5244–5263** (~18 used; rest spare) |
| Mobs | **9116–9127** |
| Weapons | **10045+** |
| Armor | **20088+** (by slot sub-dir) |
| Materials (new ore) | **40069+** |
| Quests | **35–37** (inner cert / middle rank / outer cert) |
| Dialogue | files by mobid (9116.yaml +) |

**Coordinate reserve — IMPORTANT discrepancy resolved here.** The coord-budget
doc earmarked Forge to the *east* (x[51..59]), but the chunk-1 hub physically
wired the Forge stub **5221 at (42,0,0) heading WEST** (through 5210 Wickerwork
Cottage), and Spoke A already occupies x[49..57]. **Forge therefore lays out
WEST: x[30..41], y[-6..6], z[-3..0]** — inside the Pothole reserve x[30..59],
clear of the hub (x42-48) and Spoke A (x49-57). The mine shaft descends to
**z-1/z-2** in the outer ring. (Action: the coord-budget sidecar gets corrected
to "Forge = west" as part of Phase R.) `tools/coord_inventory.py` gates every
chunk at 0 collisions.

## 3. Room manifest (18 rooms, 3 rings) — coords hand-verified cartesian-clean

> 5221 (Talus Gap, **already built**, sanctuary stub at (42,0,0)) is the
> threshold; this chunk wires its WEST exit to 5244. Conventions: north=y-1,
> south=y+1, east=x+1, west=x-1, up=z+1, down=z-1. Every coord below is unique
> and every exit's delta matches its direction (hand-checked; the authoritative
> `coord_inventory.py`/`cartcheck` run is batched to a popup-tolerant window
> before the Phase-R gate closes). Existing Pothole rooms occupy x42-57; Spoke B
> sits at x31-41 (no overlap). Inner z0; mine descends z-1 then z-2.

### Inner ring — the smithy (sanctuary; trivial) — 5 rooms [5244-5248]
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5244 | Forge Path | (41,0,0) | E→5221, W→5245 | the climb up from the gap; coal-smell intensifies |
| 5245 | The Coulee Smithy | (40,0,0) | E→5244, W→5246, N→5247 | **`station: forge`**; smith NPC + cert quest 35 start; anvil/forge/bellows nouns |
| 5246 | Ore Stall | (39,0,0) | E→5245 | the smith's material stock (ingots, leather, coal); buy-materials beat |
| 5247 | Quench Shed | (40,-1,0) | S→5245, W→5248 | **`station: forge`** (second workbench) for the repeatable craft beat |
| 5248 | The Last Worked Stone | (39,-1,0) | E→5247, W→5249 | sanctuary ENDS here; smith warns + teaches sethome |

### Middle ring — the talus slope (NO sanctuary; can KO) — 6 rooms [5249-5254]
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5249 | Lower Talus | (38,-1,0) | E→5248, W→5250, S→5251 | first live foe (scree scavenger) |
| 5250 | Scree Field | (37,-1,0) | E→5249, W→5253, S→5252 | loose-rock hazard; ore-vein noun (forageable basalt-iron) |
| 5251 | Collapsed Cut | (38,0,0) | N→5249, W→5252 | middle-ring quest NPC (cave-in **survivor**) |
| 5252 | Ore Pocket | (37,0,0) | E→5251, N→5250 | salvage/gather lesson room; armor-drop foe |
| 5253 | Upper Talus | (36,-1,0) | E→5250, W→5254 | repeatable ore/scavenger hub |
| 5254 | Mine Mouth | (35,-1,0) | E→5253, D→5255 | the descent begins; last middle room; vertical down |

### Outer ring — the mine shaft (challenge + boss) — 7 rooms [5255-5261]
| Id | Title | Coord | Exits | Notes |
|---|---|---|---|---|
| 5255 | Mine Head | (35,-1,-1) | U→5254, W→5256 | first dark room; a denizen; dark/footing note |
| 5256 | Timbered Drift | (34,-1,-1) | E→5255, W→5257, D→5258 | crew fight (2); side-branch down |
| 5257 | Flooded Sump | (33,-1,-1) | E→5256 | loot + a tougher foe; dead-end pocket |
| 5258 | Lower Drift | (34,-1,-2) | U→5256, W→5259 | descent; a lieutenant-tier denizen |
| 5259 | The Stone Gallery | (33,-1,-2) | E→5258, W→5260 | approach to the boss; tension beat |
| 5260 | The Den | (32,-1,-2) | E→5259, W→5261 | the **boss**: a stone-blooded mine beast |
| 5261 | Deep Vein | (31,-1,-2) | E→5260 | post-boss reward vista (rich basalt-iron vein); lateral-connector stub (future) |

Coord set (all unique): inner (41,0,0)(40,0,0)(39,0,0)(40,-1,0)(39,-1,0);
middle (38,-1,0)(37,-1,0)(36,-1,0)(35,-1,0)(38,0,0)(37,0,0); outer
(35,-1,-1)(34,-1,-1)(33,-1,-1)(34,-1,-2)(33,-1,-2)(32,-1,-2)(31,-1,-2). Lateral
connector stub on 5261 (parent §6.1). Biomes: smithy interior `house`/`fort`,
slope `mountains`, mine `cave`; every painted feature look-able (noun-token rule
enforced by `newbie_manifest_check.py`).

## 4. Mob manifest (9116–9127) — DRAFT

| Id | Name (clears novel + roster — VERIFY at Phase M) | Room | Archetype | Hostile | Role |
|---|---|---|---|---|---|
| 9116 | Smith (TBD: Tarn/Delk/Rusk…) | 5245 | noncombat_questgiver | no | teaches craft loop; grants quest 35 + starter recipes + ingots |
| 9117 | Cave-in Survivor (TBD name) | 5251 | noncombat_questgiver | no | middle-ring quest giver (clear the slope / gather ore) |
| 9118 | Scree Scavenger | 5249/5253 | (weak fighter) | yes | first live foe; low statpool |
| 9119 | Talus Lurker | 5252/5253 | (weak fighter) | yes | armor-drop foe |
| 9120 | Mine Crawler | 5255/5256 | (fighter) | yes | dark-ring denizen |
| 9121 | Tunnel Brute | 5256/5257 | (fighter, tankier) | yes | outer crew |
| 9122 | Stone-Crusted Lurker | 5258 | (fighter) | yes | mini-boss before the beast |
| 9123 | **Stone-Blooded Beast** | 5260 | (boss fighter) | yes | the spoke boss (mutated cave beast) |
| 9124–9127 | spare (extra scavengers / flavor) | — | — | — | reserve |

Hostile mobs reuse existing combat archetypes (`behaviors/archetypes/`) — no new
btrees unless the boss wants a scripted beat. **Difficulty NOT tuned here per
[[feedback-defer-tuning-to-post-build-playtest]]** — reasonable starting
statpools (scavenger ~18, lurker ~40, brute ~90, mini ~130, beast ~200), flagged
for the user's post-build evening-playtest tuning pass.

The two NPCs (smith + survivor) are Opened — each gets one understated visible
mutation in its description (per canon; Smith Brindle in Stillwater is NOT a
model — he predates the Opened-everywhere rule).

## 5. Quest manifest

**Quest 35 — "First Heat" (inner, cert).** The smith grants on `ask <smith>
craft`/`learn`/`quest`. Beats (the new craft hooks — see §6): see your recipes
(`recipes`/`craft list`), get/own the ingredients, `craft iron-dagger` at the
forge (craft already notifies the quest engine — verified). Reward: smithing
seeds (`skillinfo blacksmithing:1`) + an ingot starter (a few `iron-ingot`s and a
`leather-strip`) + **the starting recipes** (iron-dagger, iron-buckler — see §6
engine gap). Closes teaching `sethome coulee` + pointing west to the slope.

**Quest 36 — "Hold the Slope / Pick the Vein" (middle, rank).** The cave-in
survivor grants. Defeat N scavengers/lurkers AND gather/`salvage` for ore; the
quest TEXT teaches that falling wakes you at the Mending Hut. Reward: blacksmithing
rank bump + a **component bag** + a recipe (iron-short-sword). + a repeatable
ore-bounty. Teaches the component bag (auto-route + `sort`) and `salvage`.

**Quest 37 — "Into the Vein" (outer, cert).** Granted after 36 / on entering the
mine. Beats: descend, beat the crew, and defeat the **Stone-Blooded Beast (9123)**
in the Den. Reward (§7.2): a **Strength bump** (the engine stat-grant from Spoke A
— `train_stat` already exists!) + a forged weapon (a notable smithed blade) + an
advanced recipe (a scabland-ore weapon unique to the spoke). The design beat: you
should descend carrying gear *you smithed upstairs*.

Inner repeatable: re-craft at the quench shed (skill-use, no gold). Middle
repeatable: ore-bounty. No re-grant bugs — every `grantsQuest` node carries the
`{id}-end` token in `questExcluded` (SOP). Reward-block keys use the no-underscore
spelling ([[reference-quest-reward-yaml-key-gotcha]]).

## 6. Engine touches — ONE real gap (flagged for your decision)

Most of the loop already ships and is verified:
- `craft <recipe>` / `craft list` / `recipes`, the `station: forge` room check,
  `salvage`, `sort`, the component bag — all exist.
- `craft` ALREADY emits `questengine.Notify("command", ...)` (craft.go) so a quest
  step can gate on crafting. **Build-time check:** confirm `salvage` does too; if
  not, add the Notify (same 1-line pattern we added to consider/kick/trip/set in
  Spoke A).
- The **Strength bump** for Q37 uses the `train_stat` action we built for Spoke A
  — no new engine work.

**THE GAP — granting a recipe.** Players carry a `knownrecipes:` set, and the
`craft` command gates unknown recipes (`RecipeNotKnown`). But there is **no quest-
reward or dialogue field to GRANT a recipe** (QuestReward has gold/item/buff/
skill/stat/spell/… but no `recipe`). This is the Spoke-B analog of Spoke-A's
missing stat-grant. Options (DECISION NEEDED):
  1. **Build it in the engine** (like we did `train_stat` for stats): add a
     `recipe_info` quest-reward field + an `IncreaseRecipe`/`LearnRecipe`
     Character method, and a `learn_recipe` trigger action. Reusable for all
     future crafting content. (Recommended — consistent with the stat-grant
     precedent, and the rest of the game will want it.)
  2. **Recipe-scroll items**: make "recipe: iron dagger" a consumable item that
     teaches the recipe on `use`/read, handed out via `givesItem`. Needs a
     use-handler that adds to knownrecipes. More content, less clean.
  3. **Auto-known by skill**: drop recipe-gating for the starter recipes (let
     blacksmithing:0 know iron-dagger/iron-buckler automatically). Simplest, but
     changes a global crafting rule and removes the "learn a recipe" lesson.

(Build-time also: confirm there's an existing **component bag** item to grant for
Q36, or create one; and create the new **basalt-iron ore** material (40069+) +
its forageable node + the advanced scabland-ore recipe + its output weapon.)

## 7. Lesson coverage (Tier-2 crafting touch points owned by Spoke B)

| Lesson | Where |
|---|---|
| `recipes` / `craft list` | smith (Q35) |
| `craft <recipe>` + the forge-station requirement | smithy (Q35) |
| the material loop (own/buy ingredients) | ore stall (Q35) |
| component bag (auto-route on pickup + `sort`) | middle ring (Q36 reward + lesson) |
| `salvage <item>` | talus slope (Q36) |
| gathering a local resource (basalt-iron) | scree field / ore pocket |
| crafting gear then USING it | descend the mine carrying your smithed weapon (Q37) |
| Strength via a cert | Q37 reward (train_stat) |

## 8. Acceptance criteria

1. Boots clean (mobs/quests/recipes loadedCount up by the new counts; 0 panics;
   flag refs validate).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean.
3. `newbie_manifest_check.py` extended for Spoke B rooms+mobs (noun-token rule).
4. A character can: enter from the hub → craft an iron dagger at the forge (Q35)
   → gear up, gather ore, salvage, get the component bag (Q36) → descend → beat
   the beast with smithed gear → claim the Strength bump + forged weapon +
   advanced recipe (Q37). Verified by a **scripted mechanics walkthrough** (the
   Spoke-A method): confirm recipes are granted, `craft` completes, the component
   bag auto-routes, salvage yields, and all rewards land on the sheet.
5. **Difficulty is NOT an acceptance gate** — set reasonable values and defer to
   the user's post-build playtest tuning pass.

## 9. Build-phase task breakdown (per the phase gates)

- **Phase R — rooms + nouns:** ~18 rooms (5244–5263), wire 5221 W→5244, sanctuary
  on inner ring only, `station: forge` on 5245 (+5247), the mine `cave` biome +
  z-descent, nouns, coords, correct the coord-budget sidecar (Forge = west).
  Audits + walkthrough → **REVIEW gate.**
- **Phase M — mobs + items:** smith + survivor NPCs + slope/mine foes + the beast;
  the basalt-iron ore material + node; the component bag (reuse or create); the
  forged-weapon + advanced-recipe outputs; spawninfo; statpools (untuned).
  Boot + manifest → **REVIEW gate.**
- **Phase D — dialogue + quests (INLINE):** smith + survivor dialogue (crafting-
  command teaching, voice-matched, Opened), quests 35–37 + repeatables, **the
  recipe-granting engine work (per §6 decision)**, the new recipe/material files.
  Scripted mechanics walkthrough → **REVIEW gate (= chunk complete).**

## 10. Open decisions for you

1. **Recipe-granting mechanism (§6 gap)** — build it in the engine (`recipe_info`
   reward + `learn_recipe` action, recommended), recipe-scroll items, or
   auto-known-by-skill? This shapes Phase D.
2. **Boss flavor** — "stone-blooded beast" (a mutated cave creature, combat boss
   that drops/justifies the forged-weapon + Str reward) vs the parent spec's
   alternative "cave-in survivor" framing. Draft assumes the **beast** (fits the
   combat + Str-bump reward). OK?
3. **Coord direction** — confirmed Forge goes **west** (x30-41), correcting the
   coord-budget doc. Flagging since the budget doc said east.
4. **Smith + survivor names** — picked at Phase M against the novel + roster;
   candidates clearing both: Tarn, Delk, Rusk, Ovell, Birna, Falv, Grieve.
