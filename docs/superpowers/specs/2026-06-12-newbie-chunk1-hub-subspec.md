# Newbie Rework — Chunk 1 Sub-Spec: The Hub Town

**Status:** Draft for user review
**Date:** 2026-06-12
**Parent spec:** `2026-05-27-newbie-area-rework-design.md` (amended 2026-06-12)
**Branch:** `worktree-feature+newbie-area`
**Build order within this chunk (mandated):** rooms + room nouns → mobs +
items → dialogue + quests, with user review between phases.

## 1. Scope

The hub town of Pothole Coulee, authored end-to-end: ~19 hub rooms + 7
spoke-mouth stubs, the Awakening Rite, the portal escape hatch, 8 NPCs, 2
quests, and one small engine feature (zone death-recovery override). After
this chunk, the veteran skip path works start-to-finish and every spoke
mouth is walkable to an "under construction" stub.

NOT in this chunk: spoke content (chunks 2–8), NPC schedules and NPC↔NPC
conversations (chunk 9 polish), wider-world connection (chunk 10 — the
portal TARGETS Thornwall Temple 468 but that room already exists; no new
connection rooms are built until cutover), new items (the store stocks
existing starter goods).

## 2. ID allocations (verified against `tools/id_inventory.py`, 2026-06-12)

| Type | Global next-free | Pothole Coulee block |
|---|---|---|
| Rooms | 5108 | **5200–5499** (whole zone; chunk 1 uses 5200–5226) |
| Mobs | 9006 | **9100–9299** (chunk 1: 9100–9107) |
| Items | 40069 | **41000–41199** (chunk 1: none) |
| Quests | 21 | **30–69** (chunk 1: 30–31) |
| Dialogue | 374 | **400–499** (chunk 1: 400–407) |
| Buffs | 90 | **95–109** (chunk 1: none) |

Zone folder: `pothole_coulee` (display name `Pothole Coulee` —
ConvertForFilename compliant). Coordinate budget: x[30..59], y[-15..14],
z[-3..3] per `newbie-area-coord-budget.md`; hub anchor (45,0,0).

## 3. Room manifest (19 hub + 7 stubs)

All hub rooms carry the `sanctuary` mutator (no combat, 5× regen). Biomes
marked `house` are indoor (weather-sheltered). Coordinates assume the
standard mapper deltas — **the builder's first task is to verify the
sign convention against `internal/mapper` posDeltas and the coord-budget
doc before mass-authoring**; the topology below is normative, exact signs
may flip uniformly.

| Id | Name | Biome | Coord | Exits (→ id) | Notes |
|---|---|---|---|---|---|
| 5200 | The Awakening Pool | water | (45,0,0) | W→5201, E→5202, S→5203, N→5211 | Zone root. The plunge pool. Rite site. Cleric Maren stationed here. |
| 5201 | West Shore Path | shore | (44,0,0) | E→5200, W→5210, S→5216, N→5218 | |
| 5202 | East Shore Path | shore | (46,0,0) | W→5200, E→5209, S→5208, N→5217 | |
| 5203 | Hub Square | city | (45,-1,0) | N→5200, S→5204, W→5216, E→5208 | Crier here. Town heart. |
| 5204 | Market Row | city | (45,-2,0) | N→5203, W→5205, E→5207, S→5215 | |
| 5205 | The Drowned Lantern (inn) | house | (44,-2,0) | E→5204, U→5206 | Innkeep Tally. |
| 5206 | Lantern Sleeping Loft | house | (44,-2,1) | D→5205 | Beds; sleep lesson site. |
| 5207 | Coulee Provisions (store) | house | (46,-2,0) | W→5204 | Trader Onna; shop. |
| 5208 | Strongbox House (bank) | house | (46,-1,0) | W→5203, N→5202 | Ledger-Keeper Croup. |
| 5209 | The Mending Hut | house | (47,0,0) | W→5202 | Healer Sala. |
| 5210 | Wickerwork Cottage | house | (43,0,0) | E→5201 | Granny Wicker (folk tradition). |
| 5211 | Basalt Stair | cliffs | (45,1,0) | S→5200, N→5212 | Climb to the school shelf. |
| 5212 | School Shelf | cliffs | (45,2,0) | S→5211, N→5213 | Open shelf over the pool. |
| 5213 | Chrysalis School Hall | house | (45,3,0) | S→5212, W→5214 | Lessons, notice board. |
| 5214 | Cleric's Study | house | (44,3,0) | E→5213 | Maren's books; quiet lore. |
| 5215 | The Threshold House | house | (45,-3,0) | N→5204 | Portal room. Warden Esk. |
| 5216 | Stilt-House Walk | shore | (44,-1,0) | N→5201, E→5203 | Texture; stilted homes. |
| 5217 | North Shore Overlook | cliffs | (46,1,0) | S→5202 | View over the pool; lore flavor. |
| 5218 | Reed Jetty | shore | (44,1,0) | S→5201 | Fishing flavor; texture. |

Spoke-mouth stubs (one room each; flavor text says the way is being
cleared / a guide will open it soon; the only exit is back):

| Id | Stub name | Coord | Attaches to | Future spoke |
|---|---|---|---|---|
| 5220 | Dry Coulee Mouth — East (stub) | (48,0,0) | 5209 E | A — Martial |
| 5221 | Talus Gap — West (stub) | (42,0,0) | 5210 W | B — Forge |
| 5222 | Reedwash Mouth — Southeast (stub) | (47,-2,0) | 5207 E | C — Alchemy |
| 5223 | Scrub Draw — Southwest (stub) | (43,-2,0) | 5216 S* | D — Wilderness |
| 5224 | Stargazer Cut — Northwest (stub) | (44,2,0) | 5218 N | E — Folding |
| 5225 | Old Field Track — North (stub) | (46,2,0) | 5217 N | F — Lore |
| 5226 | Bluff Steps — Northeast (stub) | (47,1,0) | 5209 N* | G — Ranged |

(*) Two attachments marked with an asterisk need a small exit addition on
the host room; the builder balances exits so no room exceeds ~5 exits.
Spoke H (reserved) gets **no stub and no exit** per the parent spec.

**Room nouns:** every hub room ships with 2–4 `nouns:` entries supporting
`look <noun>` (e.g. the pool's water, the basalt, the stilts, the notice
board, the strongboxes, the portal arch). Multi-word nouns use the
hyphenation convention (e.g. `notice-board`) per the known parser
limitation. The noun pass is part of the rooms phase, not an afterthought.

## 4. Mob manifest (8 NPCs, all Opened, all non_combatant)

Tenet 5: every NPC is visibly mutated; each description names its
mutation naturally. All `non_combatant: true` (hub is sanctuary anyway).
No schedules in this chunk (chunk 9 adds them); each NPC gets 2–3 idle
commands for texture.

| Id | Name | Role / room | Visible mutation (flavor) | Dialogue |
|---|---|---|---|---|
| 9100 | Cleric Maren | Awakening Rite + orientation; 5200 | Pale chitin plates along the jaw | d-400 |
| 9101 | Innkeep Tally | Inn; 5205 | Gill-frills at the collar | d-401 |
| 9102 | Sala the Mender | Healer; 5209 | A third, slow-blinking eye | d-402 |
| 9103 | Ledger-Keeper Croup | Bank; 5208 | Knuckles of polished horn | d-403 |
| 9104 | Trader Onna | Store + shop; 5207 | Skin that shifts like reed-shadow | d-404 |
| 9105 | Granny Wicker | Folk tradition; 5210 | Willow-green hair that moves on its own | d-405 |
| 9106 | Crier Pell | Hub square; 5203 | A second mouth at the throat (the loud one) | d-406 |
| 9107 | Warden Esk | Portal; 5215 | Eyes like settled embers | d-407 |

Trader Onna's shop stocks **existing** starter goods only (builder selects
from current items: torch/rations/waterskin-tier plus a couple of the
cheapest existing weapons; the sling + pouch of shot from the ranged
content are included as a quiet Spoke-G preview). No new item specs.

## 5. Quest manifest

### Quest 30 — "The Awakening" (the rite; mandatory beat)

Flow (mechanisms verified against the engine 2026-06-12):

1. New arrival is pointed to the pool by every NPC's root dialogue.
2. `ask maren rite` (triggers incl. `quest`/`task` per SOP) → dialogue
   grants `30-start` (`questExcluded: ["30-start", "30-end"]` per the
   re-grant SOP) and Maren describes the rite.
3. The rite beat: Maren's behavior tree responds to the player's
   follow-up (`ask maren begin` or speech keyword) — gated on quest token
   `30-start` and NOT `30-end` — and fires the existing
   **`grant_mutation` btree action** (`internal/behaviortree/actions_quest.go:47`,
   rolls one random mutation from the player's pool) plus `grant_quest`
   `30-end`. Messaging: an in-character rite sequence (pool light, the
   Chrysalis dream, waking changed) — multi-line, no numbers.
4. Post-rite, Maren's `30-end`-gated root points the player at the
   `mutations` command (in-character), mentions `help`, `renameself`,
   and the portal ("Warden Esk will open the way whenever you wish").
5. Re-asking for the rite after `30-end` → gentle refusal node
   ("The pool opens a person once.").

**Builder verification step:** confirm the btree event/trigger that lets
a mob dialogue/keyword fire `grant_mutation` for the *triggering player*
(the action exists and was built for this; the exact trigger wiring —
dialogue-tree-fired btree event vs keyword respond — is verified at build
time, with a fallback of a room script on 5200 handling `enter pool`).
This is the chunk's highest-risk integration point; it gets a dedicated
smoke check.

### Quest 31 — "Find Your Footing" (optional orientation)

- Granted by Crier Pell (`ask pell quest`), requires `30-end`.
- Steps: visit the school hall (5213), buy anything at Coulee Provisions
  (or just `list`), and look in at the Drowned Lantern — three beats that
  walk the player past most of the hub.
- Mechanism: dialogue-gated progression via tokens on the three NPCs'
  trees (visit-room quest triggers if the quest engine supports them —
  builder verifies; dialogue-token fallback works regardless).
- Reward: a small purse of gold + Pell announces the seven coulee mouths
  (the spoke teaser). No items.

### Portal (not a quest — a permanent mechanism)

The Threshold House (5215) portal is a **room script** (JS, onCommand):
`enter portal` / `portal` → if the player has quest token `30-end`,
teleport to **Thornwall Temple Interior (room 468)** with an in-character
transit line; otherwise Warden Esk explains the pool comes first. Script
is repeatable forever; this is the veteran exit. (Quest-engine `Teleport`
exists at `internal/questengine/bridge.go:205`; the room-script route is
chosen because the portal must be repeatable and not quest-step-bound —
builder confirms the room-script API offers teleport or routes via the
bridge.)

## 6. Engine touch (one): zone death-recovery override

Parent spec §11.2 default: deaths inside the newbie zone recover at the
hub, not the global shrine. Today `DeathRecoveryRoom` is global-only
(`configs.GetSpecialRoomsConfig().DeathRecoveryRoom`, consumed at
`internal/hooks/NewRound_AutoHeal.go:37` and `internal/rooms/roommanager.go:280`).

Add an optional per-zone override: `death_recovery_room:` in
zone-config.yaml; the two consumption sites prefer the override when the
DYING character's zone declares one. Pothole Coulee sets it to **5209
(The Mending Hut)** — waking under Sala's care. TDD: unit tests on the
resolution helper (zone with override, zone without, character outside
any zone). This lands in this chunk because middle-ring knockouts (chunks
2+) depend on it for testing.

## 7. Lesson coverage (Tier 1 touch points owned by chunk 1)

| Lesson | Where |
|---|---|
| Awakening + `mutations` command | Maren, post-rite (Q30) |
| `help` / `help <topic>` literacy | Maren orientation line + school notice board noun |
| `ask <npc> <topic>` dialogue | every NPC; hints per SOP |
| `say`/`tell`/`emote` mention | Crier Pell flavor |
| `list`/`buy` (+ multi-buy mention) | Trader Onna (Q31 beat) |
| Bank deposit/withdraw | Ledger-Keeper Croup |
| `sleep`/`stand` + sleep tradeoff | Innkeep Tally + the loft (5206) |
| `renameself` touch | Maren post-rite |
| `set linewidth` QoL surface | school notice board noun text |
| `quest`/`hint` flow | Q31 via Pell |
| Portal / leaving | Warden Esk |
| Sanctuary concept | hub flavor text (felt, not explained) |

Spoke-owned lessons (combat, crafting, magic, ranged, etc.) are NOT
taught in the hub — the hub points outward.

## 8. Acceptance criteria (gate for chunk completion)

1. Boot clean (full pre-smoke instance wipe; loaders show +26 rooms, +8
   mobs, +2 quests, +8 dialogue trees; zero panics).
2. `python tools/coord_inventory.py` — zero collisions; all new coords
   inside the reserved block. `cartcheck Pothole Coulee` clean.
3. Veteran path live-verified: fresh-ish character (admin-teleported in,
   no mutation) → rite → exactly ONE mutation granted (re-ask refused) →
   portal refuses before `30-end`, teleports to 468 after. Under 5
   minutes.
4. Q31 completes; reward pays once.
5. Death inside the zone recovers at 5209 (engine test + live check);
   death outside the zone still uses the global room.
6. No-numbers audit: grep sweep of all new player-facing strings.
7. Every dialogue trigger discoverable (hint/text/noun coverage per SOP);
   quest-granting nodes carry `quest`/`task` triggers and correct
   questExcluded.
8. All 7 stubs walkable and return-only; no exit to the reserved H slot.
9. Phase reviews: rooms+nouns reviewed before mobs/items phase; mobs
   reviewed before dialogue/quests phase (user-mandated cadence).

## 9. Build-phase task breakdown (for the writing-plans doc)

- **Phase R (rooms + nouns):** zone-config + 26 room YAMLs + sanctuary
  flags + coords + nouns; scanner + cartcheck + boot smoke; REVIEW GATE.
- **Phase M (mobs + items):** 8 NPC YAMLs (mutated descriptions, idle
  commands, non_combatant) + Onna's shop stock; boot smoke; REVIEW GATE.
- **Phase D (dialogue + quests):** 8 dialogue trees, quests 30/31, the
  rite btree wiring, the portal room script, the death-recovery engine
  touch (can land earlier if convenient), live smoke per §8; REVIEW GATE.

## 10. Open questions for THIS chunk (answer before/at build)

1. The rite's exact btree trigger wiring (verified affordance; exact
   event TBD at build — fallback `enter pool` room script documented).
2. Whether visit-room quest triggers exist for Q31 (dialogue-token
   fallback specified).
3. Stub flavor: one shared "the way is being cleared" voice vs per-spoke
   teaser lines (recommend per-spoke one-liners — cheap and characterful).
