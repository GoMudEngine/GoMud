# Newbie Rework — Chunk 6 Sub-Spec: Spoke E (The Folding / Magic)

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§6.3 roster, §6.4 rings, §7.2 rewards, §5 Tier-2 cast/channels/concentration/
> willpower lessons + §4.4 Folding lore). Hub + Spokes A/B/C/D built, verified,
> committed. This is the **fifth spoke**. Same gates: **rooms+nouns → REVIEW →
> mobs+items → REVIEW → dialogue+quests (inline) → REVIEW**.

## 0. Decisions locked (user, 2026-06-14)

1. **Spell grants:** mid-ring "basic spell" = **`sparks` (Conviction Sparks)** —
   the weakest AoE spell in the game (mag 120, diff 25), so a newbie's first area
   attack. Outer "notable spell" = **`heal` (Mend Flesh)** — a self-heal, the big
   take-away.
2. **Focus item (inner reward) = Oak Staff (10018)** — 2-handed, defensive (high
   parry), 1.60x spell-damage multiplier (the strongest caster focus). REUSE, no
   new item.
3. **Boss = solo caster-duel:** the "Unfolded" — a long-dead caster's escaped
   folds — fights as a single reality-warping aberrant that casts at you. A clean
   spell-vs-spell capstone (no spawned adds). Difficulty deferred
   ([[feedback-defer-tuning-to-post-build-playtest]]).

## 1. Scope

The magic / Folding tutorial spoke. It runs from hub stub **5224 (Stargazer
Cut)** at (44,-2,0) up into a ruined **observatory**, through a **meditation
grove**, out to a **reality-thin scabland** where the Unfolded has bled through.
Three rings teach the casting loop:

| Ring | Risk | Teaches | Cert reward (§7.2) |
|---|---|---|---|
| **Inner** (observatory ruin) | None — sanctuary | `cast <spell>` syntax (no memorization), the Folding lore (§4.4), willpower as the casting stat, the three channels (physical/magical/conviction, descriptive) | spellcasting seeds (rank) + the **Oak Staff** focus |
| **Middle** (meditation grove) | Real — can KO you | casting in a real fight, **concentration** (damage / position breaks a cast), willpower scaling | spellcasting rank + the **`sparks`** AoE spell granted |
| **Outer** (reality-thin scabland) | A caster boss | sustaining a spell-fight, healing mid-combat | **Willpower bump** + **`heal` (Mend Flesh)** granted |

**Theme:** the other spokes armed the body; E opens the mind. New characters
already carry `conviction-spike` (a conviction attack) and `chrysalis-glow`, so
this spoke leans into the **magical/willpower** side of the Folding and gives the
player their first **area** attack and their first **heal** — the tools that make
a caster self-sufficient. Out of scope: deep spell theory, spell memorization
(there is none — `cast` is direct), the summoning/necromancy trees (wider world).

## 2. ID allocations (DRAFT — verify with `id_inventory.py` at Phase R)

> LESSON (C4/C5): run `id_inventory.py` BEFORE picking ids, and its per-slot
> "next free" is NOT a global check — verify against the global max.

| Type | Spoke E block (after Spoke D's 5282-5301 / 9136-9143 / 41-43) |
|---|---|
| Rooms | **5302–5321** (~20) |
| Mobs | **9144–9159** |
| Quests | **44–46** |
| Dialogue | files by mobid |
| Items / recipes | **NONE NEW** — focus = Oak Staff 10018 (reuse); rewards are spells (`spellid`) + stat/skill. Leanest spoke. |
| Engine | **NONE** — `cast` ships; `spellid` reward grants spells (verified path); stat_info/skillinfo/itemid all exist. |

**Coordinate reserve.** Stub **5224 (44,-2,0)** exits north to hub 5218. The NW
ground is crowded (Spoke A + the Chrysalis School at x44-45 y-1..-3; Spoke F is
the reserved NW arm), so **the Folding spoke ASCENDS** — the observatory ruin
climbs the z-axis (z0→z3, empty above the whole zone) with a modest NW/W
footprint, then the grove and reality-thin scabland spread at the upper levels.
This uses the vertical reserve the coord-budget earmarked for E ("observatory ↑,
z[0..3]"). Exact coords hand-verified cartesian-clean at Phase R; the spatial
lane is roughly **x[37..44], y[-8..-2], z[0..3]** (threading west/up of A and the
school). `coord_inventory.py` gates 0 collisions.

## 3. Ring sketch (rooms 5302–5321, ~20) — detailed manifest at Phase R

- **Inner — observatory ruin (sanctuary, ~6):** a toppled stargazing tower the
  Folding-adepts once used; the teacher NPC + cert quest 44; `cast` lesson (cast
  a spell you already know at a practice mark); Folding-lore + channels exposition;
  vertical climb up the ruin (z0→z1/2). Sanctuary ends at the grove's edge.
- **Middle — meditation grove (no sanctuary, ~6-7):** a grove where the veil
  thins; reality-warped lesser creatures (casting/erratic foes); the second NPC +
  quest 45; the **concentration** beat (cast while being hit — damage/movement
  can break the cast); grant the `sparks` AoE.
- **Outer — reality-thin scabland (no sanctuary, ~7-8):** the land comes apart —
  floating stone, folded distance; escalating aberrant casters; the **Unfolded**
  boss in the thinnest place. Reward vista + lateral-connector stub.

Biomes: observatory interior `house`/`fort` (indoor — watch the house-biome
weather note); grove `forest`; reality-thin scabland `land`/`cliffs` (or a
suitably eerie existing biome). Vertical exits (up/down) for the tower climb.
Noun-token rule enforced.

## 4. Mob sketch (9144–9159) — detailed manifest at Phase M

- **2 NPCs (Opened):** a **Folding-adept** teacher (inner, quest 44 — teaches
  cast/channels/willpower) and a **grove warden / meditation-keeper** (middle,
  quests 45+46 — concentration + sends you at the Unfolded). Names cleared vs
  novel+roster at Phase M (Grieve is reserved-and-cleared; second name TBD).
- **Foes:** reality-warped grove creatures (erratic, some cast) for the middle;
  aberrant "fold-echoes" for the outer; the **Unfolded** boss (aberration,
  `tank_taunter` or a caster archetype, statpool ~200, casts at the player —
  solo). Species: aberration (23) + maybe elemental (36+) for warped things.
- Reuse existing archetypes; statpools mirror the curve (weak ~18 → boss ~200),
  **untuned**. The boss should actually CAST (so the player feels a spell-duel) —
  check for a caster/`casting` archetype or give it combatcommands that cast.

## 5. Quest sketch

- **Quest 44 — inner cert (Folding-adept):** learn to cast — `cast
  conviction-spike` (a spell the player already has) at a practice mark; hear the
  Folding lore + the three channels. Reward: `skillinfo spellcasting:1` + the
  **Oak Staff** (`itemid 10018`) + gold.
- **Quest 45 — middle rank (grove warden):** cast in a real fight against a
  veil-warped creature, and learn concentration (a hit or a move can break your
  cast — push through it). Reward: `skillinfo spellcasting:1` + **`spellid:
  sparks`** (the AoE) + gold.
- **Quest 46 — outer cert (grove warden):** cross the reality-thin scabland and
  defeat the **Unfolded** in a spell-duel. Boss `mob_death` → reward: `stat_info
  willpower:3` + **`spellid: heal`** (Mend Flesh) + gold. (Turn-in automatic on
  the kill, like the other capstones.)

Repeatable: the inherent cast-to-train loop + respawning grove foes (matches
prior spokes). SOP: every `grantsQuest` node carries the `{id}-end` token in
`questExcluded` + quest/task triggers; reward keys no-underscore for tag-less
fields, `stat_info` tagged. (Note: `spellid` is a tag-less field → use the
no-underscore key `spellid`.)

## 6. Engine touches — NONE (all verified machinery)

- `cast <spell>` ships and works. The **`spellid` quest-reward field already
  grants a spell** (QuestReward.SpellId → Character.LearnSpell, fired in
  HandleQuestUpdate — the same block that handles itemid/skillinfo/stat_info).
  So both spell grants (`sparks`, `heal`) need NO new code.
- `stat_info` (willpower bump), `skillinfo` (spellcasting rank), `itemid` (Oak
  Staff) all exist + verified.
- **Build-time checks:** confirm a fresh char can `cast conviction-spike` (they
  start with it) for the Q44 beat; confirm the concentration-break mechanic fires
  on damage/move mid-cast (for the Q45 lesson — it's existing combat behavior,
  just surfaced); confirm the boss can be made to actually cast (caster archetype
  or combatcommands).

## 7. Lesson coverage (Tier-2 magic beats owned by Spoke E)

| Lesson | Where |
|---|---|
| `cast <spell>` syntax (no memorization) | adept, Q44 |
| Folding lore (§4.4) + willpower as casting stat | adept, Q44 |
| three channels (physical/magical/conviction, descriptive) | adept dialogue |
| casting in combat | grove, Q45 |
| concentration (damage/position breaks a cast) | grove, Q45 |
| a first AoE spell | Q45 reward (`sparks`) |
| Willpower via a cert | Q46 (`stat_info`) |
| a granted heal | Q46 reward (`heal` / Mend Flesh) |

## 8. Acceptance criteria

1. Boots clean (rooms/mobs/quests up; flags validate; 0 panic).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean.
3. `newbie_manifest_check.py` extended for Spoke E rooms + mobs.
4. A character can: enter → `cast` at the practice mark + hear the lore (Q44, get
   the staff) → fight + cast in the grove, feel concentration, get `sparks` (Q45)
   → cross the thin scabland, out-cast the Unfolded → claim Willpower + `heal`
   (Q46). Verified by a scripted walkthrough confirming `cast` fires the quest
   step and every reward (incl. both granted spells) lands (save = ground truth).
5. Difficulty NOT a gate — reasonable statpools, defer to playtest.

## 9. Build-phase breakdown

- **Phase R — rooms+nouns:** ~20 rooms 5302-5321, wire 5224, sanctuary inner
  only, the vertical observatory climb (z0→z3), biomes, nouns, coords (hand-
  verified, then authoritative audits in a popup window). REVIEW.
- **Phase M — mobs:** 2 NPCs + grove/scabland foes + the Unfolded (make it cast);
  spawninfo; statpools (untuned). NO new items/recipes. Boot + manifest. REVIEW.
- **Phase D — dialogue+quests (INLINE):** adept + warden dialogue (cast/channels/
  concentration teaching), quests 44-46 (spell grants via `spellid`), scripted
  walkthrough. REVIEW (= chunk complete).

## 10. Open items for you (flagged, not blockers)

1. **NPC names** — Phase M (Grieve reserved; second adept/warden name cleared vs
   novel+roster then).
2. **Boss casting** — the Unfolded should visibly cast (spell-duel feel). If no
   clean "casting" mob archetype exists, give it combatcommands that `cast` harm
   spells. Confirm approach at Phase M.
3. **`sparks` is conviction-channel** (Conviction Sparks) — it's the weakest AoE
   as you asked, but it shares the player's existing conviction-spike channel
   rather than teaching the magical channel. Flagging only; your call stands.
