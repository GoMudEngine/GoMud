# Newbie Rework — Chunk 7 Sub-Spec: Spoke F (Lore & Folk Tradition)

> Parent spec: `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
> (§4.2 the sci-fi tease / Orbital Stone, §6.3 spoke roster — "the soft spoke, no
> boss, all social and discovery", §7.2 reward table row F, §5 Tier-1 `help`
> reinforcement + Tier-2 dialogue/faction/social beats, §10 chunk 7 = ~25 rooms).
> Hub (C1) + Spokes A Martial / B Forge / C Alchemy / D Wilderness / E Folding
> all built, verified, committed. This is the **sixth spoke** and the FIRST
> non-combat one. Same phase gates: **rooms+nouns → REVIEW → mobs+items → REVIEW
> → dialogue+quests (inline) → REVIEW**.

## 0. What makes this spoke different (read first)

Every spoke A–E ended in a **boss fight**. Spoke F deliberately does **not** —
it is the parent spec's "soft" spoke (§6.3): "no boss, all social and discovery
... so the player who only plays combat-y content still sees the world's
texture." The capstone is a **quiet discovery** — the **Orbital Stone** (§4.2),
a half-buried relic of the world's sci-fi underpinning that the player finds,
examines, and does *not* understand (the payoff is wider-world, much later).

That inverts the verification model: quest progress comes from **`ask`/dialogue,
`search` (find the hidden stone), `look`/examine, room-entry, and `command`
triggers** rather than kills. The one exception is a single low-stakes scuffle —
a belligerent farmhand the player **`taunt`s** down (decision §1.1) — and even
that beat is `command: taunt`, not a required `mob_death`. The "challenge" is
social and exploratory, not combat. This spoke also carries the **Tier-1 `help`
reinforcement** beat (§5) and is where the **schedule system** becomes visible
(farmstead NPCs on daily routines).

## 1. Decisions for you to confirm (flagged — see §10 for detail)

These are the real forks; my recommended default is **bolded**. I can build on
the defaults if you'd rather not micro-decide — they're all reversible content.

1. **Rhetoric — exercised via a light taunt encounter (USER DECISION
   2026-06-15).** Rhetoric is the engine's Conviction/taunt skill. Rather than
   grant it purely socially, the spoke includes **one low-stakes scuffle**: a
   belligerent farmhand at the standing stones squares up, and the player
   **`taunt`s him down** — genuinely exercising rhetoric — to defuse the dispute.
   The Q48 rhetoric beat is `command: taunt` against him (trains rhetoric via
   `OnSkillUse` + fires the quest step); he then yields (dialogue/btree). He is
   weak and the encounter reads as non-lethal. This is the ONE combat touch in an
   otherwise social/discovery spoke; the `skillinfo rhetoric:N` reward now lands
   on something the player actually did.
2. **The "faction nod" reward (§7.2 outer).** Recommended: **a narrative
   recognition + a wearable folk token, NOT a mechanical faction-rep grant**
   (quest rewards have no faction-rep field today; a real grant would be engine
   work). The elder formally names you a friend of the coulee folk; you get a
   charm that marks it. A deeper faction tease (a wider-world order) is delivered
   as *lore in dialogue*, not membership.
3. **Orbital Stone discovery mechanic.** Recommended: **a hidden object revealed
   by `search` in the shrine (reusing the hidden-object discovery system), then
   `look`/examine it for the lore reveal that completes the spoke.** Confirm at
   build time that finding a hidden object can fire a quest trigger (§6).
4. **Almost combat-free (USER DECISION 2026-06-15).** Per the chosen path the
   spoke has exactly ONE low-stakes scuffle (the belligerent farmhand, decision
   #1) so rhetoric is genuinely exercised; every other mob is `non_combatant` and
   everything else is social/discovery. A deliberate, minimal softening of the
   parent spec's strict "no boss / no combat" — not a return to a combat spoke.

## 2. Scope

The lore / folk-tradition spoke. It runs **north / north-west** from the
already-built hub stub **5225 (Old Field Track, (46,-2,0))** — a cart-track
leading out of town — up through an **outlying farmstead** (a working Opened-folk
homestead, the social inner ring), out to a ring of **standing stones** (folk
tradition + the faction/lore tease), and finally to an **old shrine** on a rise
where the **Orbital Stone** lies half-buried (the discovery capstone). Three
concentric rings:

| Ring | Risk | Teaches | Cert reward (§7.2) |
|---|---|---|---|
| **Inner** (the farmstead) | None — sanctuary | `ask <npc> <keyword>` dialogue depth, the in-game **`help`** system (Tier-1 reinforcement — "reading is your long-term reference"), the **schedule** system made visible (farm folk on daily routines) | Charisma seeds (`stat_info`/seed) + a **folk charm** item |
| **Middle** (the standing stones) | None — no hostiles | folk tradition + the **faction/lore tease** (who the coulee folk are, hints of a wider order), unlocking deeper dialogue keywords by asking the right things; a light **persuasion/social** beat (defuse a small dispute via dialogue choice) | Rhetoric rank bump (`skillinfo`) + **dialogue keywords unlocked** + a **lore item** |
| **Outer** (the old shrine) | None — quiet | `search` for hidden things, `look`/examine for lore, the **Orbital Stone** discovery (the §4.2 sci-fi tease — found, read, not understood) | Charisma bump (`stat_info`) + a **faction nod** (narrative + token) + the **discovery moment** |

**Theme:** A–E armed the body and the mind; F opens the *world* — who these
people are, what they believe, and the one strange thing buried under it all
that none of them can explain. It is the breather spoke, and the seed of the
deepest mystery in the game.

Out of scope: real faction quest chains (wider world), any combat, deep
persuasion mechanics (DOGMud has no skill-checked dialogue beyond keyword/quest
gating — we use quest flags + keyword unlocks to *simulate* "you said the right
thing", inventing no system), understanding the Orbital Stone (that is the point).

## 3. ID allocations (DRAFT — verify with `id_inventory.py` at Phase R/M)

> LESSON (C4/C5/C6): run `id_inventory.py` BEFORE picking ids; its per-slot "next
> free" is NOT a global check — verify against the global max. C6 consumed rooms
> 5302–5321, mobs 9144–9152, quests 44–46.

| Type | Spoke F block |
|---|---|
| Rooms | **5322–5346** (~25) |
| Mobs (all non-combat NPCs + maybe 1–2 ambient non-hostile animals) | **9153–9165** |
| Quests | **47–49** (inner social cert / middle tradition+rhetoric / outer discovery) |
| Dialogue | files by mobid |
| Items | **2 new:** a folk charm (inner reward) + a lore item / readable relic (middle reward). Verify the next-free consumable/misc id with `id_inventory` at Phase M (the C4 30060→30065 lesson). The Orbital Stone is a **room feature / hidden object**, not an inventory item. |
| Engine | **Target: NONE.** `ask`/dialogue, `help`, `search`/hidden-object discovery, `look`/examine, schedules, `stat_info`/`skillinfo`/`itemid` rewards all ship. **Build-time checks (§6):** (a) can a hidden-object `search` find fire a quest `command`/discovery trigger? (b) is there a `read` command or is examine = `look <noun>`? If a hidden-object find can't notify the quest engine, add the 1-line Notify (the forage/drink/throw/cast precedent). |
| **Reused, NOT new** | schedules plumbing, dialogue engine, hidden-object discovery, faction `IsPeacefulToward`/groups, charm/trinket item patterns. |

**Coordinate reserve.** Hub stub **5225 (Old Field Track) sits at (46,-2,0)**
(chunk-1 manifest), currently exiting only `south→5217`. This chunk wires its
**north** exit to the farmstead. Spoke F lays out across the **N / NW ground at
z0–1**, lane **x[33..48], y[-4..-15], z[0..1]** — north of Spoke A (A is
y[-3..3]) and at ground level *under* Spoke E's elevated observatory (E occupies
x40–44 y[-2..-11] at **z1–3**, so z0 there is free except E's single stub room
5224 at (44,-2,0), which F must avoid). Clear of Spoke G's reserved NE-bluff arm
(x51–59). `tools/coord_inventory.py` gates 0 collisions; `cartcheck
pothole_coulee` verifies intra-zone geometry. The other free hub stub, **5226
(Bluff Steps), is Spoke G's** (bluff-top terraces) — not touched here.

## 4. Ring sketch (rooms 5322–5346, ~25) — detailed manifest at Phase R

> Detailed room table (titles/coords/exits/nouns) is authored at Phase R and
> hand-verified cartesian-clean, then run through `coord_inventory.py` +
> `cartcheck` + the extended `newbie_manifest_check.py` (same as A–E). Conventions:
> north=y-1, south=y+1, east=x+1, west=x-1, up=z+1.

- **Inner — the farmstead (sanctuary, ~7–8):** a working homestead of Opened folk
  — a longhouse, a barn, kitchen garden, a well. The **elder/folk-keeper NPC** +
  the social cert quest 47; the `help` reinforcement beat; 2–3 farm NPCs on
  **schedules** (the player sees them move dawn→field→dusk→hearth). `ask`-depth
  lesson. Sanctuary ends at the farmstead's gate.
- **Middle — the standing stones (no sanctuary, ~8–9):** a windswept ring of
  carved megaliths on the open scrub; a **tradition-keeper NPC** + quest 48; the
  faction/lore tease (the carvings, the folk's origins, a hint of a wider order);
  a small **dispute** with a belligerent farmhand the player **`taunt`s** down
  (the one low-stakes scuffle — the rhetoric beat — quest-flag branch); the
  **dialogue-keyword unlock** beat.
- **Outer — the old shrine (no sanctuary, quiet, ~8–9):** a tumbledown shrine on a
  low rise (z+1), older than the folk tradition that half-remembers it; quest 49;
  `search` reveals the half-buried **Orbital Stone**; `look`/examine delivers the
  §4.2 sci-fi tease; the discovery completes the spoke. A reward vista + the
  lateral-connector stub toward Spoke A's outer ring (the actual connector is
  authored in chunk 8 per the amended plan).

Biomes: farmstead interiors `house` (indoor — weather note), farmyard/track
`farmland` or `land`; standing stones `land`/`scrub`; shrine `land` + a `fort`/
`house` interior if walled, on a z+1 rise. Every painted feature look-able
(noun-token rule enforced by `newbie_manifest_check.py`). No `mapsymbol` surprises.

## 5. Mob manifest (9153–9165) — DRAFT, all NON-COMBAT

| Id | Role | Ring | Notes |
|---|---|---|---|
| 9153 | **Elder / folk-keeper** (quest 47 giver) | farmstead | Opened, `noncombat_questgiver`; teaches `ask`/`help`; grants the folk charm |
| 9154–9156 | farm folk (scheduled) | farmstead | Opened `noncombat_passive` w/ `schedule_id` daily routines (the schedule teaching beat); each asks/answers flavor + folk lore |
| 9157 | **Tradition-keeper** (quest 48 giver) | standing stones | Opened `noncombat_questgiver`; faction/lore tease, keyword unlocks, sends you to defuse the dispute |
| 9158 | **Belligerent farmhand** (the taunt target) | standing stones | Opened, **ATTACKABLE** (hostile-lite, weak statpool ~25, NOT `non_combatant`); the ONE combat touch — player `taunt`s him down (Q48 rhetoric beat), he then yields via btree/dialogue. Defeatable by force too, but taunt is the intended path |
| 9159 | aggrieved disputant (the other party) | standing stones | Opened `noncombat_questgiver`/`noncombat`; the quieter half of the feud; thanks the player after the defusal |
| 9160 | **Shrine-tender / hermit** (quest 49 giver) | old shrine | Opened `noncombat_questgiver`; points the player to search the shrine; reacts to the Orbital Stone find |
| 9161–9165 | spare / ambient (e.g. a non-hostile herd animal, a wandering pilgrim) | — | reserve; flavor only, non-hostile |

All NPCs are **Opened** (each gets one understated visible mutation in its
description, per canon) and **non-combatant** (`non_combatant: true` — cannot be
attacked) **EXCEPT the belligerent farmhand 9158**, who is attackable so the
taunt beat works (the single, deliberate combat touch — decision §1.1). Names
cleared against the novel
`STORY_BIBLE.md` + the live mob roster at Phase M (the C6 method; "Grieve" is
spent, so all-new names). No archetypes beyond the existing `noncombat_*`.

## 6. Engine touches — target NONE, with build-time checks

The whole loop should already ship. Confirm at build time (Phase D, the
forage/drink/throw/cast Notify precedent):

- **`ask`/dialogue depth, keyword unlocks** — dialogue engine + `questRequired`/
  `questFlagRequired` gating (ships; used heavily in A–E).
- **`help`** — ships (the hub cleric already walks the player through it in C1).
  F reinforces it via the elder's dialogue + a `command: help` quest beat — **verify
  `help` fires `questengine.Notify("command", help)`**; if not, add the 1-liner.
- **`taunt`** (the rhetoric beat, Q48) — the player taunts the belligerent
  farmhand. `taunt` ships and trains rhetoric (`OnSkillUse`). **Verify `taunt`
  fires `questengine.Notify("command", taunt)`** so the Q48 step registers; if
  not, add the 1-liner (the consider/kick/trip/set precedent from Spoke A, and
  cast from Spoke E). The farmhand yields after being taunted via a small btree
  (`player_taunt`/low-HP) or a flag-gated dialogue node.
- **`search` → hidden Orbital Stone** — the hidden-object discovery system ships
  (`docs/.../2026-03-17-hidden-object-discovery`). **Verify a hidden-object find
  can advance a quest** (a `command: search` beat in the shrine room, or a
  discovery hook). If neither path notifies the quest engine, add a Notify or
  gate the capstone on a `room_enter`/`look`-based token instead.
- **`look`/examine for the lore reveal** — confirm whether a `read` command
  exists or examine is `look <noun>`; author the reveal on whichever ships.
- **Schedules** — `schedule_id` plumbing ships (townsfolk use it). Farm NPCs get
  schedules purely for ambient teaching; no quest dependency.
- **Rewards** — `stat_info` (charisma), `skillinfo` (rhetoric), `itemid` (charm +
  lore item) all ship and are verified. No new reward type needed (decision #2
  keeps the faction nod narrative, so no faction-rep reward field).

If a genuine gap appears (e.g. hidden-object finds truly can't touch quests),
the fix is a 1-line Notify mirroring the established pattern — flagged, not
expected to be large.

## 7. Lesson coverage (Tier-1 help reinforcement + Tier-2 social/lore beats)

| Lesson | Where |
|---|---|
| `ask <npc> <keyword>` dialogue depth | elder + all F NPCs (Q47) |
| in-game `help` system (Tier-1 reinforcement) | elder (Q47) |
| schedules made visible (NPC daily routines) | farm folk (ambient, Q47 framing) |
| dialogue keyword unlocks ("you said the right thing") | tradition-keeper (Q48, quest-flag gated) |
| faction / wider-world lore tease | standing stones carvings + tradition-keeper (Q48) |
| `taunt` (rhetoric in action) — talk down the belligerent farmhand | standing stones (Q48) — the one combat touch |
| a light dispute defused (via the taunt + a dialogue choice) | dispute pair (Q48) |
| `search` for hidden things | old shrine (Q49) |
| `look`/examine for lore | Orbital Stone (Q49) |
| the §4.2 sci-fi discovery tease | Orbital Stone (Q49 capstone) |
| Charisma via certs | Q47 seed + Q49 bump (`stat_info`) |
| Rhetoric via the taunt beat | Q48 (`skillinfo`, earned by taunting 9158) |

## 8. Acceptance criteria

1. Boots clean (rooms/mobs/quests up by the new counts; flags validate; 0 panics).
2. `coord_inventory.py` 0 collisions; `cartcheck pothole_coulee` clean.
3. `newbie_manifest_check.py` extended for Spoke F rooms + mobs (noun-token rule;
   NPCs asserted non_combatant).
4. A character can: enter from the hub → talk the farmstead, use `help`, see the
   folk on schedules, get the charm (Q47) → reach the standing stones, learn the
   folk/faction lore, unlock deeper keywords, **`taunt` the belligerent farmhand
   to defuse the dispute** (the rhetoric beat), get the rhetoric bump + lore item
   (Q48) → reach the shrine, `search` out the Orbital Stone, examine it for the
   tease, claim the Charisma bump + faction nod (Q49). Verified by a **scripted
   mechanics walkthrough** (the A–E method): confirm each
   `ask`/`help`/`taunt`/`search`/`look` beat advances its quest and **every
   reward lands on the sheet**.
5. **Almost-no-combat invariant:** every F mob is `non_combatant: true` EXCEPT
   the belligerent farmhand (9158); the walkthrough confirms the `taunt` beat
   fires + the player completes all three quests with at most that one optional
   scuffle (and never needs to KILL anything — the farmhand yields to taunt).

## 9. Build-phase task breakdown (per the phase gates)

- **Phase R — rooms + nouns:** ~25 rooms (5322–5346), wire 5225 N→5322, sanctuary
  on the farmstead only, the z+1 shrine rise, biomes, nouns, coords (hand-verified
  → authoritative `coord_inventory`/`cartcheck`/manifest in a popup window). The
  Orbital Stone room gets its hidden-object noun. REVIEW.
- **Phase M — mobs + items:** the elder + tradition-keeper + shrine-tender
  questgivers, scheduled farm folk, the dispute pair (all Opened, non_combatant,
  names cleared); the folk charm + lore item (ids verified via `id_inventory`);
  schedule files for the farm NPCs; spawninfo. Boot + manifest. REVIEW.
- **Phase D — dialogue + quests (INLINE):** all NPC dialogue (ask-depth, help,
  faction/lore tease, keyword unlocks, the dispute branch, the shrine), quests
  47–49 + repeatables, the engine build-time checks (§6), the Orbital Stone
  discovery wiring. Scripted mechanics walkthrough verifying all rewards + the
  no-combat invariant. REVIEW (= chunk complete).

## 10. Open items for you (flagged, not blockers)

1. **The four decisions in §1** — rhetoric handling, faction-nod nature, Orbital
   Stone mechanic, and the combat-free invariant. Defaults are sensible and
   buildable; confirm or redirect.
2. **NPC names** — picked at Phase M against the novel + roster (all-new; Grieve
   is spent). Folk-tradition flavor (homestead elders, a tradition-keeper, a
   shrine hermit).
3. **The folk charm + lore item identities** — proposed: a minor charisma/luck
   trinket (charm) and a readable folk relic (lore item, e.g. a carved bone token
   or a wax-sealed account). Open to specifics.
4. **Difficulty is N/A here** (no combat) — but pacing/length of a 25-room
   no-fight spoke is a feel question for the evening playtest
   ([[feedback-defer-tuning-to-post-build-playtest]]); if it drags, trim rooms.
5. **The Orbital Stone's text** — the one place the game's sci-fi underpinning
   peeks through. Worth getting the voice exactly right (cryptic, technical,
   wrong-for-the-setting); I'll draft and you can tune at the D gate.
