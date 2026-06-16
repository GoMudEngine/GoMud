# Newbie Rework — Chunk 8 Sub-Spec: Spoke G (Ranged & Marksmanship) + Connectors

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§5 Tier-2 ranged beats, §6.1 topology + lateral connectors, §6.3 roster row G,
> §7.2 reward table row G, §10 chunk 8). Hub + Spokes A/B/C/D/E/F built, verified,
> committed. This is the **seventh and final spoke**, and it also authors the
> **lateral outer-ring connectors** that close the exploration ring (moved here
> from chunk 7 per the amended plan — they belong with the last spoke). Same
> gates: **rooms+nouns → REVIEW → mobs+items → REVIEW → dialogue+quests (inline)
> → REVIEW**.

## 0. Decisions for you to confirm (flagged; my default in **bold**)

1. **Weapon-per-ring (reward ladder).** Per §7.2: inner = **sling (10038) + Pouch
   of Shot (30064)**; middle = **hand crossbow (10039) + Case of Bolts (30063)**;
   outer = **hunting bow (10041) + a generous ammo stockpile**. All EXIST — pure
   reuse, no new items. Confirm or reshuffle.
2. **The boss = a kiting raider marksman** (parent §6.3) on the overlook — uses
   the existing **`archer` archetype** (keeps distance, shoots, retreats as you
   close). The lesson it embodies: you must close the gap or out-shoot a kiter.
   Default: **archer-archetype raider, statpool ~200 ("Velk-lite" ranged)**.
3. **Cross-room-shot lesson.** The middle ring teaches shooting into the *next*
   room (and that a cross-room shot provokes the target to come for you). Default:
   **a two-room setup — a firing step with a foe visible in the adjacent room;
   `shoot <foe>` across the exit, then it charges in.**
4. **Lateral connectors (the extra deliverable).** Default: **author F↔A and
   C↔D (the §6.1 connectors) + A↔G (the new NE spoke links to A's outer area).**
   These are short exit-links between existing outer-ring rooms (plus at most a
   couple of connector rooms where geography needs them). E (far west) and B (far
   east) stay spurs — the ring is two arcs joined at top (F-A) and bottom (C-D),
   not a full loop. Confirm the connector set.
5. **Engine touch.** `shoot` (and likely `reload`) do **not** notify the quest
   engine — add the 1-line `Notify("command", …)` (the cast/taunt/search
   precedent) so Q50/Q51 shoot/reload beats register. The `item_info` multi-item
   reward (from C4) delivers the outer ammo stockpile. `stat_info`/`skillinfo`/
   `itemid`/`givesItem` all exist. NO new items/recipes.

## 1. Scope

The ranged-combat tutorial spoke, and the capstone of the whole newbie area. It
climbs **north-east** from hub stub **5226 (Bluff Steps, (47,-1,0))** up onto a
range of wind-scoured **bluffs**: a tier of **shooting terraces** (inner), down
into an **echoing box canyon** (middle, where shots carry between rooms), and out
to a **raider overlook** (outer) where a kiting marksman holds the high ground.
Three rings teach the ranged loop:

| Ring | Risk | Teaches | Cert reward (§7.2) |
|---|---|---|---|
| **Inner** (shooting terraces) | None — sanctuary | `shoot`, `reload`, the ammo economy (a ranged weapon needs ammo in inventory), ranged-combat as a **Perception** skill | Ranged seeds (`skillinfo ranged-combat:1`) + a **sling** + a **shot pouch** |
| **Middle** (echoing box canyon) | Real — can KO | **cross-room shots** (`shoot` a foe in the next room) + the **retaliation rule** (the target comes for you) + kiting awareness | Ranged rank bump + a **hand crossbow** + **bolts** |
| **Outer** (raider overlook) | A kiting boss | sustaining a ranged fight, closing on or out-shooting a kiter | **Perception bump** + a **hunting bow** + a **generous ammo stockpile** (`item_info`) |

**Theme:** the other spokes taught you to fight up close, brew, cast, track, and
talk. G teaches you to reach out and touch something from across the canyon — the
last tool, and the one that rewards a keen eye (Perception). The capstone is the
first foe that fights the way *you* now can: at range, on the move.

Out of scope: the deep ammo-crafting economy, firearm (pistol/arbalest) tiers
(teased, stocked in the wider world), advanced kiting tactics. Every command used
(`shoot`, `reload`, `equip`, `get`) already ships — see §6 for the one engine
1-liner.

## 2. ID allocations (DRAFT — verify with `id_inventory.py` at Phase R/M)

> LESSON (C4/C5/C6/C7): run `id_inventory.py` BEFORE picking ids; its per-slot
> "next free" is NOT global. Spoke F consumed rooms 5322–5346, mobs 9153–9160,
> quests 47–49, items 20089 + 30066.

| Type | Spoke G block |
|---|---|
| Rooms | **5347–5366** (~20) + a couple of connector rooms if needed (next free past 5366) |
| Mobs | **9161–9176** |
| Quests | **50–52** |
| Dialogue | files by mobid |
| Items / recipes | **NONE NEW** — sling 10038, hand crossbow 10039, hunting bow 10041; ammo Pouch of Shot 30064, Case of Bolts 30063, Quiver of Arrows 30062 (all exist). Leanest-tier spoke alongside E. |
| Engine | **1 touch:** `shoot` (+ maybe `reload`) → `questengine.Notify("command", …)`. Everything else ships. |

**Coordinate reserve.** Stub **5226 (47,-1,0)** exits south to hub 5209-area
(per the chunk-1 manifest it currently has `south:5209`). Spoke G ASCENDS NE onto
the bluffs — lane **x[51..59], y[-15..-7], z[1..3]** (the coord-budget G
allotment). It threads **above and north of Spoke A** (A occupies x49–57 at
y0..-3, z0–2, with its watchtower at x56–57 z2) — G stays at y ≤ -7 and z ≥ 1,
clear of A's cells; Phase R hand-verifies every coord against A's watchtower in
particular. `coord_inventory.py` gates 0 collisions; `cartcheck` verifies
intra-zone geometry (watch the z-axis vertical exits — the bluffs climb).

## 3. Ring sketch (rooms 5347–5366, ~20) — detailed manifest at Phase R

- **Inner — the shooting terraces (sanctuary, ~6):** wind-scoured stone shelves
  the bluff-folk use as a range; straw butts and distance-marks. The
  marksman-instructor NPC + cert quest 50; the `shoot`/`reload`/ammo lesson
  against a fixed practice butt; a vertical climb up the terraces (z1→z2).
  Sanctuary ends at the canyon lip.
- **Middle — the echoing box canyon (no sanctuary, ~7):** a slot canyon where
  sound and shot carry between chambers; raider scouts at range. The second NPC +
  quest 51; the **cross-room shot** beat (a foe in the adjacent chamber; shoot it
  through the exit, it charges you); the ammo-economy + kiting-awareness lesson.
- **Outer — the raider overlook (no sanctuary, ~7):** the bluff-top where a
  raider band has made a shooting-perch; escalating raider marksmen; the **kiting
  raider boss** on the highest overlook (z3). Reward vista + the A↔G lateral
  connector stub.

Biomes: bluffs/terraces `cliffs`; canyon `cliffs` (or `cave` for the slot); a
shooter's hut interior `house`/`fort`. Vertical exits for the climb. Noun-token
rule enforced. Cross-room shooting needs clean adjacent-room exits (no portals).

## 4. Mob sketch (9161–9176) — detailed manifest at Phase M

- **2 NPCs (Opened):** a **marksman-instructor** (inner, quest 50 — teaches
  shoot/reload/ammo) and a **range-warden / scout** (middle, quests 51 + sends
  you at the overlook). Names cleared vs novel + roster at Phase M.
- **Practice butt** (inner): a fixed, durable, harmless `combat_passive` target
  to shoot at (mirrors Spoke A's training dummy / Spoke E's practice mote).
- **Foes:** raider scouts/marksmen (some using the **`archer` archetype** so they
  shoot + kite — teaching the player what ranged feels like from the other side);
  the **kiting raider boss** (`archer`, statpool ~200) on the overlook. Statpools
  mirror the curve (weak ~20 → boss ~200), **untuned**
  ([[feedback-defer-tuning-to-post-build-playtest]]).
- Reuse existing combat + `archer` archetypes; no new btrees unless the boss
  wants a scripted kite beat beyond what `archer` provides.

## 5. Quest sketch

- **Q50 — inner cert (instructor):** learn to shoot. The instructor hands you a
  **sling + a pouch of shot** (`givesItem`), teaches `equip` it, `shoot` the
  practice butt, and `reload` when the pouch runs low. Reward: `skillinfo
  ranged-combat:1` + keep the sling/pouch + gold. (`shoot`/`reload` command
  beats — see §6 Notify.)
- **Q51 — middle rank (scout):** a foe waits in the next chamber. `shoot` it
  across the exit (the cross-room beat) — it charges in; finish it, and mind your
  ammo. Reward: `skillinfo ranged-combat:1` + **hand crossbow (10039)** + **Case
  of Bolts (30063)**.
- **Q52 — outer cert (scout):** climb to the overlook and bring down the **kiting
  raider marksman** — the first foe that shoots and runs the way you now can.
  Boss `mob_death` → reward: `stat_info perception:3` + **hunting bow (10041)** +
  a **generous ammo stockpile** (`item_info`, e.g. `30062:3,30063:2,30064:2`).
  Turn-in automatic on the kill (the A/B/E/D capstone pattern).

Repeatable: re-shoot the butt (inner, skill-use); a raider cull (middle, gold).
SOP: every `grantsQuest` node carries `{id}-end` in `questExcluded` + quest/task
triggers; reward keys no-underscore for tag-less fields (`itemid`/`skillinfo`),
tagged for `stat_info`/`item_info`.

## 6. Engine touches — ONE 1-liner + checks

- **`shoot` Notify (required).** `internal/usercommands/shoot.go` does NOT call
  `questengine.Notify("command", "shoot")`. Add it (the cast/taunt/search
  precedent) so Q50/Q51 shoot beats register. **Confirm `reload` too** if Q50
  uses a reload beat — `reload.go` also lacks it; add if the beat needs it.
- **Cross-room shoot** ships (the MOTD: "fire instantly with shoot, even into the
  next room"). The retaliation ("the target comes for you") is existing ranged
  behavior — surface it in quest text. Build-time: confirm a cross-room `shoot`
  fires the `command:shoot` Notify with the SHOOTER's room id (the quest gates on
  the room the player shoots *from*).
- **`item_info`** (multi-item ammo stockpile) shipped in C4; **`stat_info`**
  (perception) + **`skillinfo`** (ranged-combat) + **`itemid`/`givesItem`** all
  exist and are verified. NO new reward type.
- **Practice butt + boss** reuse `combat_passive` / `archer` archetypes.

## 7. Lesson coverage (Tier-2 ranged beats owned by Spoke G)

| Lesson | Where |
|---|---|
| `shoot` a ranged weapon | terraces (Q50) |
| `reload` + the ammo economy | terraces (Q50) |
| ranged-combat as a Perception skill | instructor dialogue (Q50) |
| **cross-room shots** (`shoot` into the next room) | box canyon (Q51) |
| retaliation — a shot target comes for you | box canyon (Q51) |
| kiting awareness (foes that shoot + retreat) | canyon raiders + the boss |
| Perception via a cert | Q52 (`stat_info`) |
| a hunting bow + ammo stockpile | Q52 rewards |

## 8. The lateral outer-ring connectors (the extra chunk-8 deliverable)

Per §6.1, the spokes' outer rings link via `══` connectors that close the
exploration ring. With all seven spokes now built, author them:

- **F↔A** (north): a connector between Spoke F's outer (the shrine rise, ~5346)
  and Spoke A's outer (the watchtower area). 
- **C↔D** (south): between Spoke C's outer (Clearwater Spring 5281) and Spoke D's
  outer (Wolfwater Spring 5299 — both are "spring" reward vistas, a natural join).
- **A↔G** (NE): between Spoke A's outer and Spoke G's overlook (both NE/bluff).

Each connector is the **lateral-stub exit** the prior spokes' reward-vista rooms
already earmarked, wired to its neighbor (plus at most one short connector room
where the geography needs a bridge). They must stay cartesian-clean —
`coord_inventory` + `cartcheck` gate them exactly like spoke rooms. They create
non-obvious shortcuts + the "I never noticed that" replay layer (parent §6.1
tenet). Detailed connector geometry is hand-drafted at Phase R alongside the G
rooms. **Open item:** confirm the exact connector set (§0.4) before wiring.

## 9. Acceptance criteria

1. Boots clean (rooms/mobs/quests up; flags validate; 0 panic).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean (incl. the
   vertical bluff exits AND the new lateral connectors).
3. `newbie_manifest_check.py` extended for Spoke G rooms + mobs + the connector
   exit-pairs.
4. A character can: enter from the hub → get a sling, `shoot` + `reload` at the
   butt (Q50) → in the canyon, `shoot` a foe in the next room and survive its
   charge, get the crossbow (Q51) → climb the overlook, out-shoot/close on the
   kiting raider, claim the Perception bump + hunting bow + ammo stockpile (Q52).
   Verified by a scripted walkthrough: `shoot`/`reload` fire the quest beats, the
   cross-room shot works, and **every reward lands** (incl. the `item_info`
   stockpile). The lateral connectors are walkable both directions.
5. Difficulty NOT a gate — reasonable statpools, defer to the evening playtest.

## 10. Build-phase breakdown

- **Phase R — rooms+nouns:** ~20 rooms 5347-5366, wire 5226, sanctuary inner
  only, the vertical bluff climb (z1→z3), the cross-room-shot adjacency, biomes,
  nouns, coords (hand-verified, then authoritative audits in a popup window).
  **Author the lateral connectors here** (F↔A, C↔D, A↔G) + extend the manifest
  checker for them. REVIEW.
- **Phase M — mobs+items:** instructor + scout NPCs + the practice butt + canyon
  raiders (archer) + the kiting raider boss; spawninfo; statpools (untuned). NO
  new items — reuse the ranged gear. Boot + manifest. REVIEW.
- **Phase D — dialogue+quests (INLINE):** instructor + scout dialogue (shoot/
  reload/ammo/cross-room/kiting teaching), quests 50-52, the `shoot`(+`reload`)
  Notify engine touch + checks, the `item_info` stockpile reward. Scripted
  walkthrough verifying all beats + rewards. REVIEW (= chunk complete).

## 11. Open items for you (flagged, not blockers)

1. **The five §0 decisions** — weapon ladder, archer boss, cross-room setup,
   connector set, the shoot/reload Notify. Defaults are buildable; confirm or
   redirect.
2. **NPC names** — Phase M, vs novel + roster (all-new).
3. **Connector geometry** — which exact outer-ring rooms join, and whether any
   need a bridge room. Hand-drafted at Phase R; flagged because it touches
   already-committed spoke rooms (A/C/D/F outer reward-vista rooms gain an exit).
4. **Firearms tease** — the primitive pistol (10040) + arbalest (10042) exist;
   Spoke G can *mention* them (wider-world gear) without granting them. Optional
   flavor.
