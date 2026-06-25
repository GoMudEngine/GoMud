# New Plymouth Capital Questlines — Design Spec

**Date:** 2026-06-24
**Status:** Design approved (scope/style/branching/cohesion + optional Q70 fight confirmed)
**Author:** brainstorming session

## Goal

Give the five quest-less districts of the New Plymouth capital their first
standalone quests by wiring the pre-built lore hooks into the quest engine.
Four quests cover all five quest-less districts (Crafting, Merchant, Temple,
Noble, Old Quarter). The capital's other districts already have quests (Docks:
63 Dock Rat + 66/67 Bloom Trail; Common: 65 Street Sweeper's Secret + the
Bloom arc).

## Design decisions (locked)

- **Scope:** all four quests, covering all five quest-less districts.
- **Play-style:** investigation/social, capital-appropriate. Combat only where
  it earns its place — here that is exactly one optional beat (the Q70 canal
  descent). The other three quests are non-combat by design, fitting a capital
  of watched, untouchable antagonists.
- **Branching:** Q68 (Cooperage Circle) is the branching marquee (join vs.
  report). The other three are linear with branch-flavored / arc-aware dialogue.
- **Cohesion: soft arc.** Each quest is independently startable and completable
  (own giver, own steps, own reward). They share one meta-story, reference each
  other's discoveries, and have a recommended order surfaced through NPC hints —
  never hard `requires`. Q68's allegiance flag *lightly colors* the other three
  (a line or two of warmth/coldness) without gating them.

## The meta-arc

One buried truth underlies the capital: the bloodline's official founding date
is false, and their charter, courts, and 4% tribute all derive from it. The
Cooperage Circle (Orin and, secretly, Keeper Lysha) preserves the proof; the
Bloodline Domestic (Horst, Clerk Vell, Guide Ferrol) is the suppression and
enforcement apparatus. The four quests approach the truth from four angles, each
with a distinct role so the player never re-discovers the same fact:

| Quest | District(s) | Role | Shape |
|---|---|---|---|
| **68 The Cooperage Circle** | Crafting | THE PEOPLE — allegiance | Branching |
| **69 The Gallery Cipher** | Temple → Noble | THE CIPHER — scholarly proof | Linear |
| **70 The Pre-Founding Web** | Old Quarter | THE RELIC — physical proof | Linear (+ optional fight) |
| **71 Horst / The Tribute** | Merchant → Noble | THE POWER — present consequence | Linear |

Recommended play order 68 → 69 → 70 → 71 (allegiance → cipher → relic → power),
hinted but not enforced.

## Shared mechanics

### Allegiance flag (cross-quest coloring)

Quest 68 declares a flag `68-allegiance` with values `[circle, bloodline]`.
- `setsQuestFlag` on each Q68 branch node sets it.
- Q69/Q70/Q71 dialogue may use `questFlagRequired`/`questFlagExcluded` on
  *optional flavor variant nodes only* — a warmer line for `circle`, a cooler
  line for `bloodline`. These are additive root/greeting variants or extra
  hint lines; the core quest-advancing nodes are NOT flag-gated, so a player who
  never did Q68 (no flag) completes every quest normally.

### Factions & reputation

Existing factions used: `cooperage_circle`, `bloodline_domestic` (mutual
enemies), `np_commonfolk`, `np_dockfolk`, `temple_np`. No new factions.

Reward rep philosophy (no XP/levels — removed from the game; rewards are gold +
faction rep + items):
- **Q68 circle branch:** `cooperage_circle` +20, `bloodline_domestic` −15.
- **Q68 bloodline branch:** `bloodline_domestic` +20, `cooperage_circle` −15,
  and a larger gold purse (the bloodline pays well).
- **Q69:** `temple_np` +10 (Dross is temple_np), small `cooperage_circle` +5
  (you helped Lysha's hidden work).
- **Q70:** `np_commonfolk` +10 (Coll), small `cooperage_circle` +5.
- **Q71:** `np_dockfolk` +10 (the wharf-merchant economy), small
  `cooperage_circle` +5 if the player leveraged the founding-proof, else none.

### IDs

- **Quests:** 68, 69, 70, 71 (next free; 67 was the last used).
- **Quest items:** 40112–40119 (eight items; next free, 40111 was last used):
  Q68 → 40112–40114, Q69 → 40115–40116, Q70 → 40117–40118, Q71 → 40119. Exact
  assignment in the per-quest sections below.
- **New mob:** 9393 — the Q70 canal lurker (next free; 9392 was last used). The
  only new mob; every other NPC already exists.

### Reward-block YAML gotcha (reference)

Quest reward blocks use tag-less keys: `gold`, `rep_faction`, `rep_amount`,
`playermessage`, `roommessage`, `itemid`, `skillinfo`. snake_case silently
no-ops in the reward block. Trigger actions/conditions and dialogue ARE
snake_case. (See memory `reference_quest_reward_yaml_key_gotcha`.) A quest with
two rep awards or two item awards needs the secondary ones applied via a
`quest_granted`/`item_give` trigger action (`bump_rep`, `give_item`) rather than
the single-value reward block — same pattern the Bloom Trail used.

### Dialogue SOPs (apply to every quest-granting node)

- Every `grantsQuest` node lists `"quest"` and `"task"` in `triggers`; quest
  patterns list them in `keywords`.
- Every `grantsQuest` includes the quest's **end token** in `questExcluded`
  (e.g. `grantsQuest: "68-start"` → `questExcluded: ["68-start", "68-end"]`).
- **Gated grant nodes go FIRST under `tree.nodes`** (the substring/node-order
  shadow lesson from the Bloom Trail — short lore triggers substring-match
  topics and shadow later grant nodes). Each quest's new grant node is placed at
  the top of its giver's `tree.nodes`.
- NPC `text` is first person; `hints` are narrator second person; every trigger
  word is discoverable in a hint, NPC line, or quest log.
- `prefer questRequired over requires`; avoid `expiryPeriod`.

---

## Q68 — The Cooperage Circle (Crafting; branching)

**Giver:** Orin the Bookseller (9332, Orin's Stall 5711), secret circle member.
**Antagonist contact (report branch):** Clerk Vell (9349, Civic Permit Office
5814, Merchant).
**Supporting NPC:** Toby the Cooper's Lad (9338, Abandoned Cooperage 5719).

**Flag:** `68-allegiance` = `[circle, bloodline]`.

**Items:**
- 40112 **Bench-Mark Rubbing** — proof Toby still tends the cooperage (trust
  token; given by examining/asking Toby, returned to Orin).
- 40113 **Edvar's Map-Fragment** — the hidden pre-founding survey page Toby
  entrusts. On the circle branch it is preserved; on the report branch it is
  evidence handed to Vell.
- 40114 **Copy of Edvar's Map** — circle-branch reward keepsake (a lore item
  that name-drops the eight-pointed symbol and the inland site, hinting Q69/Q70).

**Steps:**
1. `start` — Orin, sensing the "old questions," sends the player to the shuttered
   cooperage (5719) to confirm the circle survives: see whether Toby still tends
   the tools, and bring back the bench-mark rubbing.
2. `toby` — At the cooperage, Toby (wary) confirms the circle, reveals the
   bloodline (Clerk Vell) has filed to seize the cooperage as derelict and that
   Edvar's last survey is hidden there. He hands over the Bench-Mark Rubbing and
   Edvar's Map-Fragment.
3. `report_back` — Return both to Orin. Orin reveals the circle's purpose
   (preserving pre-founding truth) and the crux: the seizure inspection is
   imminent. He presents the choice.
4. **Branch:**
   - `branch_circle` — Help re-register the cooperage "in use": carry a
     cooper's-guild token (Orin provides) and confirm the archive is moved to
     safety (to Coll/Gritta's network). `setsQuestFlag {68-allegiance: circle}`.
   - `branch_bloodline` — Carry what you learned (and Edvar's Map-Fragment as
     evidence) to Clerk Vell. `setsQuestFlag {68-allegiance: bloodline}`. The
     cooperage is seized; Toby is displaced.
5. `end` — Branch-flavored resolution.

**Rewards:**
- Circle: gold (modest, ~60), `cooperage_circle` +20, `bloodline_domestic` −15
  (−15 via `bump_rep` trigger action), item 40114 (Copy of Edvar's Map).
- Bloodline: gold (larger, ~100), `bloodline_domestic` +20, `cooperage_circle`
  −15 (via `bump_rep`), a "permit favor" flavor line (no item required).

**Branching Quest SOP scaffolding:**
1. Flag declared in quest YAML with both values.
2. `setsQuestFlag` on each branch node.
3. **Dismissal nodes at the TOP of Vell's tree** for wrong-path / non-quest
   players (so keyword patterns don't imply a hidden quest).
4. Branch-flavored `end` text.
5. Orin root variant acknowledging an in-progress allegiance choice.

**Combat:** none.

---

## Q69 — The Gallery Cipher (Temple → Noble; linear)

**Giver:** Scholar Dross (9360, Cloister Walk 5904, Temple).
**Supporting NPCs:** Guide Ferrol (9370, Tour Steps 6004, Noble — the steer/
warn-off), Keeper Lysha (9371, The Gallery — Upper 6008, Noble — the cipher
keeper).
**Key rooms:** Grand Temple nave 5901 (processional reliefs), Art Gallery 6007
(third panel), Gallery Upper 6008 (Lysha).

**Items:**
- 40115 **Cipher Rubbing** — taken from the processional reliefs in 5901.
- 40116 **Lysha's Annotated Reading** — Lysha's sketch matching the third panel
  to the rubbing; carried back to Dross to complete the cipher.

**Steps:**
1. `start` — Dross asks for a rubbing of the hand-position cipher on the
   processional reliefs in the Grand Temple nave (5901).
2. `rubbing` — Player examines/interacts with the reliefs in 5901 → receives the
   Cipher Rubbing (room_interact grants the item + advances). Return to Dross; he
   reads it partway, needs the gallery's key.
3. `gallery` — At the Noble gallery (6007), Ferrol steers the player to the
   portraits and warns against asking Lysha about "the third panel from the
   left" — the discoverable hint. (Ferrol gets a new quest-aware node; he does
   NOT grant the quest, just routes.)
4. `lysha` — Upstairs (6008), Lysha reads the third panel against the rubbing:
   the eight-pointed symbol marks an inland pre-founding settlement. She gives
   Lysha's Annotated Reading. (Warmer variant line if `68-allegiance: circle`;
   cooler but still helpful if `bloodline`.)
5. `complete` — Return to Dross with the reading. He completes the cipher: the
   real founding date is roughly a century earlier. Bittersweet close — provable
   truth, immovable powers.
6. `end`.

**Rewards:** gold (~70), `temple_np` +10, `cooperage_circle` +5 (via `bump_rep`),
a scholar's keepsake flavor line. Dross's end text hints the physical original
lies below, in the Old Quarter (→ Q70).

**Combat:** none.

---

## Q70 — The Pre-Founding Web (Old Quarter; linear + optional fight)

**Giver:** Coll the Sweeper (9320, Carter's Rise 5602, Common) — the accessible
end of the underground network.
**Supporting NPC:** Gritta (9381, The Buried Lintel 6037, Old Quarter z−2).
**Key rooms:** the Old Quarter canal descent (6020 → 6030s → z−2), The Buried
Lintel 6037, The Deep Canal 6038.

**Items:**
- 40117 **Sealed Grey Fragment** — Coll entrusts it to carry down to Gritta.
- 40118 **Lintel Rubbing** — Gritta's rubbing of the eight-pointed symbol on the
  Buried Lintel; carried back up as the physical proof.

**New mob:**
- 9393 **Canal Lurker** — a hostile creature in the flooded canal descent (a
  blind, pale, eel-/crab-like scavenger adapted to the lightless water). Low-mid
  difficulty, fits a capital-newcomer power level. Spawns in one of the descent
  rooms (e.g. the Deep Canal approach). This is the single combat beat. It is
  *not* required to talk to Gritta if avoidable, but it blocks/contests the most
  direct path, so most players will fight it. `hostile: true`, modest stats,
  archetype `fighting`, drops a minor canal-salvage material (reuse an existing
  grey-material / salvage item — no new drop item required).

**Steps:**
1. `start` — Coll asks the player to carry the Sealed Grey Fragment down to
   Gritta in the deep cellar and see the lintel themselves — "someone outside
   the network should know it's real."
2. `descend` — Descend the Old Quarter canals to z−2 (the Canal Lurker contests
   the way). Reach Gritta at the Buried Lintel (6037).
3. `lintel` — Examine the Buried Lintel (room_interact in 6037) and the Deep
   Canal grey-material seam (6038): "you are standing beneath the something."
   Gritta gives the Lintel Rubbing.
4. `return` — Bring the Lintel Rubbing back up to Coll (or Orin) → routed
   onward; the player is quietly acknowledged as part of the network.
5. `end` — The web is complete: discovery + cipher + relic align. Knowledge
   stays underground; the bloodline's charter still stands.

**Rewards:** gold (~70), `np_commonfolk` +10, `cooperage_circle` +5 (via
`bump_rep`), a lintel-symbol keepsake flavor line. If Q69 is complete, extra
dialogue acknowledging the painting copied this very stone.

**Combat:** one fight — the Canal Lurker (9393).

---

## Q71 — Horst / The Tribute (Merchant → Noble; linear)

**Giver:** Dame Ostry (9347, Dame Ostry's Blades 5804, Merchant) — a tributepayer
squeezed by an audit and a held surety deposit.
**Supporting NPCs:** Clerk Vell (9349, Civic Permit Office 5814), Horst (9344,
Gilt Threshold — Private Parlor 5815). Both untouchable (non_combatant).

**Items:**
- 40119 **Tribute Ledger Page** — a scrap obtained near/through Horst's parlor
  showing the tribute funnels to the bloodline.

**Steps:**
1. `start` — Ostry's grievance: Vell's 4% + a held surety deposit are bleeding
   her trade. She asks the player to learn what the tribute is *for* and whether
   there is any appeal.
2. `vell` — Investigate Clerk Vell (5814): "the rate is set by charter." The
   charter = the founding authority. If the player has done Q69/Q70 (truth
   known), a flavor branch lets them privately note the charter's false
   foundation — leverage, even if unusable in court.
3. `horst` — Reach Horst at the Gilt Threshold (5815). The player learns the
   tribute funnels up to the bloodline and that discretion is everything; obtain
   the Tribute Ledger Page (via a room_interact on a scrap in the parlor, or an
   item_give exchange — Horst himself stays untouchable).
4. `report` — Bring the ledger page to Ostry. Realistic outcome: she cannot
   overturn the bloodline, but with evidence + the player's discretion she gets
   her surety released / permit cleared. A small, human win.
5. `end` — One merchant breathes; the bloodline endures.

**Rewards:** gold (Ostry pays well, ~110), `np_dockfolk` +10 (the wharf-merchant
economy), a quality item from Ostry (reuse an existing blade item, e.g. 40001,
or a shop-discount flavor line — no new item required for the reward), and
`cooperage_circle` +5 (via `bump_rep`) only if the founding-proof was leveraged
(i.e. player had Q69/Q70 done — gated via questFlagRequired/has-quest condition).

**Combat:** none.

---

## Architecture / file inventory

For each quest (68–71):
- `_datafiles/world/dogmud/quests/<id>-<name>.yaml` — steps, flags (Q68 only),
  rewards, triggers (room_interact, item_give, quest_granted, bump_rep,
  give_item, set_flag).
- Dialogue node additions on existing givers/supporting NPCs:
  - Q68: Orin 9332 (grant + branch + report-back), Toby 9338 (hand-off), Vell
    9349 (report branch + dismissal nodes).
  - Q69: Dross 9360 (grant + complete), Ferrol 9370 (steer/warn node), Lysha
    9371 (reading node + allegiance flavor variants).
  - Q70: Coll 9320 (grant + return), Gritta 9381 (lintel node).
  - Q71: Ostry 9347 (grant + report), Vell 9349 (charter node), Horst 9344
    (apparatus node + ledger hand-off).
- New items 40112–40119 under `_datafiles/world/dogmud/items/<category>/`.
- New mob 9393 (Canal Lurker) under
  `_datafiles/world/dogmud/mobs/new_plymouth_old_quarter/` + dialogue not
  required (hostile) + spawn in a descent room's spawn list + (optional) a
  schedule not needed (always-hostile ambient).
- Room edits: spawn list for 9393; room_interact hooks on 5901 (reliefs), 6007
  (third panel), 6037 (lintel), 6038 (grey seam), 5815 (ledger scrap). Confirm
  the quest engine's `room_interact` event wiring against an existing example
  (the Bloom Trail / Street Sweeper quest) before authoring.

## Testing

- Boot test (ValidateZoneConsistency errors=0 mode=panic; quests load;
  flag-declaration validator passes — undeclared flag refs panic at startup).
- Harness playtest each quest end-to-end (both Q68 branches), verifying:
  grants fire on natural words (put grant nodes first), items hand over, steps
  advance, rewards/rep apply, the Canal Lurker fight is survivable at
  capital-newcomer power, branch flags set and color later quests.
- Minimal-prompting discoverability pass on at least the marquee (Q68), per the
  Bloom Trail precedent.
- Re-grant prevention: each grant node's `questExcluded` includes the end token.

## Out of scope (deferred)

- The economy money-sink / legendary BIS craft items (separate spec).
- The Bloom mechanic deepening (shipped separately).
- Hard-chaining the four quests, additional branches beyond Q68, or per-branch
  combat variants.
- The prod push (held by user policy).
