# QUEST: Ulla's Silence

**Quest ID:** 20
**Zone(s):** Stillwater (single-zone)
**Type:** Investigation / lore (with item-delivery completion)
**Quest giver:** Ulla (mob 347, room 4137 Ulla's Parlor)
**Completion NPC:** Ulla (terminal); intermediate handover at Vella Thorne (NEW mob 355, room 4136 Healer's Cottage)
**Other NPCs involved:** Smith Brindle (existing 337, dialogue extension), Temple Priest Seren (existing 344, dialogue extension)

---

## CONCEPT

Ulla has been quietly grieving for years. The town accepts that her husband Elgar Voss disappeared on a fishing trip and was lost to the lake. Ulla has always sensed there was more — but the workshop above her parlor, where Elgar's tools and unfinished carvings still lie, has gone untouched.

The quest is investigative. With Ulla's blessing, the player explores the workshop, finds a spiral-motif carving Elgar scratched into his bench-vise, and follows the same motif across five other Stillwater locations (the temple's pillars, the temple garden, the old chapel ruin, the wardstone circle, and the lake-cave bone shoals). The investigation reveals that Elgar was researching pre-Chrysalis lore — specifically, that the spiral motif marked alignment points connecting the lake caves to the wardstone circle, and that something ancient (the sump dweller from quest 19) lived in the deep. Elgar went down alone to investigate and never came back. Smith Brindle was supposed to go with him and didn't.

The cache at Elgar's empty grave contains his sister Vella's confession — "He was your brother." — and his carved kingfisher. The player carries the kingfisher to Vella, who finally allows herself to be unburdened and gives the player Elgar's last journal entry. The player brings the journal to Ulla; she reads it, finally able to grieve openly. Quest closes with a flag-tracked choice — does the player tell Ulla the full truth (including Vella's guilt), or just give her the journal?

This is the centerpiece lore quest of Stillwater. Mechanical reward is modest; emotional resolution is the actual payoff.

---

## CANON CLEANUP — INCLUDED IN THIS QUEST'S IMPLEMENTATION

The Voss death timeline is **inconsistent across three existing canonical sources**:

| Source | Stated date | Action |
|--------|-------------|--------|
| Ulla's mob description (347) | "four years ago" | **UPDATE → "twelve years ago"** |
| Uncle's Workshop room (4138) | "twelve years ago, on a fishing trip east" | Keep as-is (canonical) |
| Stillwater Cemetery (4140) | "perhaps fifteen years ago" | Keep as-is (the dates suggest the town has rounded UP slightly — Vella's note in dialogue can acknowledge "near twelve years now, though the marker says fifteen because Brindle wanted it kinder") |

**Canonical date:** Elgar disappeared twelve years ago. Ulla's mob description is the only outlier and gets updated as part of this quest's implementation (NOT a prerequisite step). Folded into the file list below.

---

## STEP CHAIN

```
Step 1: "start" — granted by dialogue at Ulla
  Trigger: `ask ulla quest` (or `task`) → tree node `accept_inquiry`
           with grantsQuest: "20-start"
  Token: 20-start
  Description: "Ulla has asked you to look through her late husband
                Elgar's things in the workshop above her parlor. She
                cannot bring herself to do it. She wants to know what
                was on his mind in the months before he disappeared."
  Hint: "Climb the stair from Ulla's parlor up to the workshop and
         search what you find there. The dust has not been disturbed
         in years. Look at the bench, the tools, the unfinished carving
         in the vise."

Step 2: "workshop_carving" — granted by room_interact
  Trigger: `search bench-vise carving` in room 4138 (the hidden_noun
           that already exists from zone build). Quest engine
           room_interact trigger: noun "bench-vise carving",
           room 4138, has 20-start, missing 20-workshop_carving →
           grant.
  Token: 20-workshop_carving
  Description: "Beneath the cast-iron handle of Elgar's bench-vise,
                a long tight spiral inside an outer circle has been
                scratched into the metal. Old work. Older than the
                cottage. Whoever scratched it knew the symbol meant
                something."
  Hint: "The spiral motif is older than Elgar's cottage. The temple's
         pillars carry the same design — speak to the temple priest,
         or examine the pillars yourself."

Step 3: "temple_pillars" — granted by room_interact OR dialogue
  Triggers (either):
    A) `look pillars` (or `examine pillars`) in room 4123 (Temple of
       Stillwater) — quest engine room_interact trigger on noun
       "pillars" with has 20-workshop_carving, missing
       20-temple_pillars → grant.
    B) `ask seren spiral` (or `quest`/`task`) at Temple Priest Seren
       — dialogue tree node `acknowledge_spiral` granting same token.
  Token: 20-temple_pillars
  Description: "The four pillars at the corners of the Temple of
                Stillwater each carry the same spiral motif as Elgar's
                workshop. Priest Seren confirms the symbol predates
                the present Chrysalis order. There is a buried marker
                in the temple garden that bears the same design."
  Hint: "Search the temple garden for the buried marker."

Step 4: "garden_marker" — granted by room_interact
  Trigger: `search buried marker` (or noun variants) in room 4124
           (Temple Garden, hidden_noun already exists from zone build).
           Quest engine room_interact trigger: noun "buried marker",
           room 4124, has 20-temple_pillars, missing 20-garden_marker.
  Token: 20-garden_marker
  Description: "The marker beneath the crab-apple tree carries the
                same spiral. Older than the temple. The carving
                points westward — toward the old chapel ruin and the
                wardstone circle beyond."
  Hint: "Walk west to the old chapel ruin and the wardstone circle.
         The spiral motif appears at both."

Step 5: "chapel_altar" — granted by room_interact
  Trigger: `look altar stone` in room 4144 (Old Chapel Ruin) — quest
           engine room_interact trigger on the altar-stone noun.
  Token: 20-chapel_altar
  Description: "The chapel altar stone bears the same spiral, framed
                by the lake-side compass directions. The old chapel
                stood on this spot before the temple was built in
                town."
  Hint: "Continue west to the wardstone circle. The spiral connects
         the sites in a line."

Step 6: "wardstone_altar" — granted by room_interact
  Trigger: `look altar slab` in room 4145 (Wardstone Circle) — quest
           engine room_interact trigger on the altar-slab noun.
  Token: 20-wardstone_altar
  Description: "The wardstone circle's central altar slab carries
                the spiral too — and a worn line, almost a channel,
                pointing east toward the lake. The four sites form a
                line across the geography of Stillwater. The line
                ends in the lake."
  Hint: "The line ends in the deep lake — somewhere only the cave
         system can reach. Search the bone shoals in the lake caves
         for what Elgar found."

Step 7: "bone_shoals_cache" — granted by room_interact
  Trigger: `search cache` (the hidden_noun already exists from zone
           build) in room 4130 (Bone Shoals). Quest engine
           room_interact trigger on noun "cache".
  Token: 20-bone_shoals_cache
  Description: "Elgar's cache lay where he hid it before going
                deeper: an oilskin pouch with a folded note to
                Brindle the smith, a small Stillwater Black Pearl,
                and a curl of waxed line. The note explains the
                spiral — the sites form alignment markers; the lake
                caves connect to the wardstone via the deep sump.
                Elgar wrote that he was going down alone because he
                could not wait for Brindle to finish the work he was
                doing. The note is dated the day he disappeared."
  Hint: "Speak to Smith Brindle. He needs to know you found this."

Step 8: "brindle_confession" — granted by dialogue at Brindle
  Trigger: `ask brindle quest` (or `task`/`elgar`/`spiral`/`note`) at
           Brindle (mob 337). Dialogue tree node `confession` requires
           20-bone_shoals_cache, grants 20-brindle_confession.
  Token: 20-brindle_confession
  Description: "Brindle was Elgar's friend and was supposed to descend
                with him. He didn't go that day. He has been finishing
                Elgar's hook-spear ever since, never quite able to
                give it to anyone. He directs you to Vella the healer
                — Elgar's sister — who has been carrying her own
                version of the same silence."
  Hint: "Visit Vella Thorne at the healer's cottage west of Ulla's
         parlor. Look first at Elgar's grave at the cemetery — there
         is something buried there for whoever knows to look."

Step 9: "kingfisher_found" — granted by room_interact
  Trigger: `search beneath elgar's marker` (or noun variants — the
           hidden_noun already exists from zone build) in room 4140
           (Stillwater Cemetery). Quest engine room_interact trigger
           with has 20-brindle_confession, missing 20-kingfisher_found.
  Token: 20-kingfisher_found
  Actions: grant + give_item: <kingfisher item id, e.g. 40060>
  Description: "Beneath Elgar's empty marker, Vella buried a small
                wooden box with a single carved kingfisher and a
                note in her own hand reading 'He was your brother.'
                The kingfisher is unmistakably Elgar's work."
  Hint: "Bring the carved kingfisher to Vella Thorne at the
         healer's cottage. She is the one the note is for."

Step 10: "vella_journal" — granted by item_give to Vella
  Trigger: `give kingfisher vella` (item_give of 40060 to mob 355).
           Quest engine item_give trigger: mob 355, item 40060,
           has 20-kingfisher_found, missing 20-vella_journal →
           grant + consume_item kingfisher + give_item journal +
           npc_say sequence (Vella's confession).
  Token: 20-vella_journal
  Description: "Vella has accepted the kingfisher and entrusted you
                with Elgar's last journal entry — the page he wrote
                the morning before he descended. The page records
                what he expected to find, and what he wanted Ulla
                to know if he didn't come back."
  Hint: "Bring Elgar's journal entry to Ulla at her parlor."

Step 11: "end" — granted by item_give to Ulla, with dialogue choice
  Trigger: `give journal ulla` (item_give of 40061 to mob 347).
           Quest engine item_give trigger: mob 347, item 40061,
           has 20-vella_journal, missing 20-end → grant + consume_item
           journal + ulla's dialogue auto-fires.

           Then a dialogue choice on Ulla determines the flag:
             - `ask ulla truth` (or `vella`/`whole`) at state has 20-end
               with no flag set → setsQuestFlag truth: whole + Vella's
               role explained.
             - `ask ulla journal` (or `quiet`/`partial`) → setsQuestFlag
               truth: partial + Vella's role kept private.
             - If player walks away without choosing, default to truth:
               partial after some idle period (or set partial as default
               on initial item_give and let the truth-telling node
               OVERWRITE to whole if asked). Recommend: default partial,
               player must explicitly ask about Vella to set whole.

           **Quest 19 cross-reference (FOLD IN):** Ulla's terminal
           dialogue has a root variant gated on `questRequired:
           ["19-end"]` + `questFlagRequired: {"19-completion": "full"}`
           that adds an acknowledgment of the sump dweller kill
           ("Drunn told me you went down into the deep and killed the
           thing in the sump. Elgar would have wanted to be the one.
           He'd be glad it was you instead.") This variant fires once
           when the player visits Ulla after both quests are
           complete. Players who did 19 partial or skipped it get
           the standard terminal greeting.
  Token: 20-end
  Description: "Ulla has read the journal and finally allowed herself
                to grieve openly. The town's accepted version of
                Elgar's death is gone, replaced by what really
                happened. The hook-spear Brindle never finished is
                yours now — Ulla brought it down from the workshop
                herself."
```

---

## BRANCHING / OPPOSED QUEST

This is **NOT a branching quest in the structural sense** — there is one quest path, one ending. But the ending has a small flag-tracked variant for the truth-telling choice:

```yaml
flags:
  - key: truth
    values: [whole, partial]
    description: "Whether the player told Ulla the full truth about
                  Vella's role, or just gave her the journal"
```

**Mechanism:** The flag is set during the `end` step's dialogue interaction. There is no branch NPC; both flag values are reached at Ulla. No followup quests currently gate on this flag — it's saved for future quests (e.g., a hypothetical Vella-redemption quest could require `truth: whole`).

**Default behavior:** When the player gives the journal to Ulla, set flag to `truth: partial` immediately. If the player then `ask ulla truth` (or related keywords at the same room), the dialogue node overwrites the flag to `truth: whole`. This means a player who just hands over the journal and walks away gets the partial path automatically — no missed flag, no broken state.

**Dismissal nodes:** Not needed (no path-mutual exclusion).

---

## ALTERNATIVE PATHS

```
Step 3 alternative — discover spiral via Seren OR via room_interact:
  Both paths grant 20-temple_pillars. Player can find it by either
  asking Seren (dialogue) or examining the pillars (room_interact).
  Quest engine handles both via separate triggers/nodes; whichever
  fires first is fine, the other becomes a no-op (questExcluded).

Step 7 alternative — give E.V. note directly to Brindle:
  The cache contains a folded note from "E.V." to Brindle. A player
  who gathers the note as a separate item (if it exists as such; if
  it's just descriptive text in the cache hidden_noun, not applicable)
  could hand it to Brindle for the same advancement. RECOMMEND: the
  cache contents are descriptive only (no separate inventory item for
  the note); Brindle's confession is gated by 20-bone_shoals_cache
  alone. Keeps inventory clean.

Step 8 alternative — players who haven't visited the cache:
  A player who has 20-workshop_carving but NOT 20-bone_shoals_cache
  who asks Brindle about the spiral or about Elgar gets a guarded
  response — Brindle has things to say but won't open up until the
  player has done the work. Brindle dialogue node: requires has
  20-workshop_carving, gives a redirect ("the temple priest knows
  more about the spiral than I do — start there"). Does NOT grant.

Step 9 alternative — bring kingfisher to Vella before searching grave:
  Not possible — kingfisher is only obtainable via room_interact at
  4140, gated on 20-brindle_confession. Player who arrives at Vella
  with no kingfisher gets a "you've not yet found what was buried"
  redirect dialogue node.

Step 11 alternative — give journal to wrong NPC:
  Quest engine item_give trigger for journal (40061) is ONLY on Ulla
  (347). Giving the journal to Vella, Brindle, Seren, or any other
  NPC should bounce back via player_give safety nets on those NPCs
  (return_item with refusal text). RECOMMEND: add player_give
  handlers for 40061 on Vella and Brindle; everyone else is unlikely
  to be tried.

Step 11 alternative — drop journal on the ground:
  If player drops 40061, it sits on the floor like any other item.
  They can pick it back up. No special handling.
```

---

## QUEST GATING DIAGRAM

```
[no quest] ──ask ulla quest──▶ [20-start]

[20-start] ──search bench-vise carving (4138)──▶ [20-workshop_carving]

[20-workshop_carving] ──either:
   look pillars (4123)  OR  ask seren spiral
   ──▶ [20-temple_pillars]

[20-temple_pillars] ──search buried marker (4124)──▶ [20-garden_marker]

[20-garden_marker] ──look altar stone (4144)──▶ [20-chapel_altar]

[20-chapel_altar] ──look altar slab (4145)──▶ [20-wardstone_altar]

[20-wardstone_altar] ──search cache (4130)──▶ [20-bone_shoals_cache]

[20-bone_shoals_cache] ──ask brindle quest──▶ [20-brindle_confession]

[20-brindle_confession] ──search beneath elgar's marker (4140)──▶
   [20-kingfisher_found] + give_item kingfisher (40060)

[20-kingfisher_found] ──give kingfisher vella──▶
   [20-vella_journal] + consume kingfisher + give_item journal (40061)

[20-vella_journal] ──give journal ulla──▶
   [20-end] + consume journal + flag truth: partial (default)

[20-end] ──ask ulla truth/vella/whole──▶ flag truth: whole
   (overwrites partial; both paths still 20-end)
```

**Order flexibility:** Steps 3-7 are presented in geographic order (temple → garden → chapel → wardstone → bone shoals) but the quest engine could allow them in any order with chained `questRequired` checks. RECOMMEND: enforce strict order (each step requires the previous one). It mirrors the breadcrumb intent and keeps the lore reveal coherent. Player who tries to look at Wardstone first sees the spiral but doesn't recognize it without the Elgar-context — quest doesn't advance.

**Verification:**
- Every step has exactly one trigger (or two equivalent triggers for step 3) ✓
- No step is unreachable (each gates on the previous) ✓
- `questExcluded` prevents re-triggering: every grant node excludes the granted token AND `20-end` ✓
- The `end` step fires rewards via the quest YAML ✓

---

## FILES NEEDED

| Action | File | Purpose |
|--------|------|---------|
| MODIFY | `mobs/stillwater/347-ulla.yaml` | (1) Canon cleanup: description "four years ago" → "twelve years ago". (2) Optional: small description softening if needed for tone consistency with the dialogue file. |
| CREATE | `quests/20-ullas_silence.yaml` | Quest definition: 11 steps, flag declaration, 7 room_interact triggers, 2 item_give triggers, rewards block (gold 100 + Elgar's hook-spear 10032). |
| CREATE | `mobs/stillwater/355-mistress_vella_thorne.yaml` | NEW NPC at room 4136. Mirror Apothecary Ilsa (338) for archetype/stats. `noncombat_questgiver` archetype. Older woman, dignified, folk-medicine healer, sister of Elgar Voss |
| CREATE | `dialogue/stillwater/347.yaml` | Ulla full dialogue: accept_inquiry (start), tell_truth (whole flag), keep_quiet (partial flag default), root variants for every quest state |
| CREATE | `dialogue/stillwater/355.yaml` | Vella full dialogue: redirect-no-quest, take_kingfisher (item_give acknowledgment, gives journal), root variants for quest states |
| CREATE | `dialogue/stillwater/337.yaml` | Smith Brindle dialogue (currently only has behavior tree, no dialogue): basic shop chat + confession node (gated 20-bone_shoals_cache) |
| CREATE | `dialogue/stillwater/344.yaml` | Temple Priest Seren dialogue: basic temple chat + acknowledge_spiral node |
| CREATE | `behaviors/stillwater/355-mistress_vella_thorne.yaml` | player_give safety net: kingfisher (40060) before quest = return; journal (40061) any time = return ("Ulla needs to read this, not me") |
| MODIFY | `behaviors/stillwater/337-smith_brindle.yaml` | Extend existing tree: add player_give for journal (40061) → return ("Ulla is the one who needs to read this") |
| CREATE | `items/materials-40000/40060-elgars_carved_kingfisher.yaml` | Quest item, NOT component (`is_component: false`). Small wooden figurine, dark wood, Elgar's distinctive carving |
| CREATE | `items/materials-40000/40061-elgars_journal_entry.yaml` | Quest item, NOT component. Folded paper, Elgar's handwriting |
| CREATE | `items/weapons-10000/10032-elgars_hook_spear.yaml` | Quest reward weapon. Base spec mirrors 10031 (lake-iron hook-spear), with affix-budget-equivalent stat bonuses baked in directly (~99 budget points spent — same as `arena 200` instance loot drops). Spread is hand-picked rather than per-player random; see "Reward Stat Spread" section below for the full numbers and reasoning. |
| MODIFY | `rooms/stillwater/4136.yaml` | Add `spawninfo` block for Vella (mob 355, cooldown 600 rounds) |

**No room edits needed for hidden_nouns** — they already exist from zone build:
- 4124 `buried marker` ✓ verified
- 4130 `cache` ✓ verified
- 4138 `bench-vise carving` ✓ verified
- 4140 `beneath Elgar's marker` ✓ verified
- 4144 altar stone — need to verify; if not present, ADD as a regular `nouns:` entry (not hidden)
- 4145 altar slab — same as 4144
- 4123 pillars — already a regular noun, no change needed

**Verify before implementation:** glob 4144 and 4145 for `nouns:` to confirm "altar stone" / "altar slab" are present. If they're only in the room description but not in the `nouns:` section, the quest engine `room_interact` trigger won't fire on `look altar stone` — need to add the noun to make the trigger work. Quest YAML triggers can also fire on `room_command: look <noun>` regardless of the nouns section, but having the noun makes the response cleaner.

**Instance saves to delete after work:**
- `mobs.instances/stillwater/347-ulla-room4137.yaml` (if exists) — Ulla template change
- `mobs.instances/stillwater/337-smith_brindle-room4106.yaml` — Brindle dialogue addition
- `mobs.instances/stillwater/344-temple_priest_seren-room4123.yaml` — Seren dialogue addition

---

## GOTCHAS CHECKLIST

- [x] Every `grantsQuest` dialogue node has `"quest"` and `"task"` in triggers
  - `accept_inquiry` (Ulla): triggers `"quest", "task", "elgar", "husband", "workshop", "help"`
  - `acknowledge_spiral` (Seren): triggers `"quest", "task", "spiral", "pillars", "elgar"`
  - `confession` (Brindle): triggers `"quest", "task", "elgar", "spiral", "note", "cache", "spear"`
  - `tell_truth` (Ulla, post-end flag-set): triggers `"truth", "vella", "whole", "everything"`
- [x] Patterns introducing the quest also include `"quest"` and `"task"` in keywords (Ulla's pattern entry for "elgar"/"workshop")
- [x] **Narrative voice:** All NPCs first-person ("I", "my"). Hints describe player options ("You could ask Ulla about her husband"). No 3rd-person self-references.
- [x] **Trigger discoverability** — every trigger word reachable:
  - `"elgar"` — Ulla mentions her husband by first name in her description and dialogue; the room 4138 description references him
  - `"spiral"` — appears in the workshop hidden_noun description ("a long tight spiral set inside an outer circle"), in temple pillar nouns, in temple garden hidden_noun, in chapel altar, in wardstone altar
  - `"workshop"` — room 4138 title and Ulla's parlor description reference the workshop
  - `"vella"` — appears in Vella's mob name AND in room 4136 description ("Mistress Vella Thorne")
  - `"kingfisher"` — appears in 4137 noun "birds" (kingfisher in mid-dive), 4138 noun "bird" (would have been a kingfisher), 4140 hidden_noun (carved kingfisher)
  - `"note"` / `"cache"` — both in 4130 hidden_noun description
  - `"journal"` — will appear in Vella's dialogue and the journal item description
  - `"truth"` / `"whole"` / `"vella"` (for end-flag choice) — will be in Ulla's post-end hint
  - `"spear"` (for Brindle context) — appears in Brindle's mob description ("half-finished hook-spear on his bench")
  - **DROPPED candidates:** `"investigate"`, `"mystery"`, `"secret"` — too generic and not sourced from in-game text
- [x] **SOP — every step advances via `ask <npc> quest`:**
  - Step 1: `ask ulla quest` at no-quest state grants 20-start ✓
  - Step 2 (room_interact): not a dialogue advance, but `ask ulla quest` at 20-start state shows hint via root variant ("did you find anything in the workshop?")
  - Step 3 alt: `ask seren quest` at 20-workshop_carving grants 20-temple_pillars ✓
  - Steps 4-7 (room_interact): not dialogue advances, but Seren/Ulla/Brindle root variants give hints
  - Step 8: `ask brindle quest` at 20-bone_shoals_cache grants 20-brindle_confession ✓
  - Step 11 (item_give): `give journal ulla`. Player CAN'T advance to end via `ask ulla quest` because end requires the journal. Ulla's hint at 20-vella_journal state explicitly tells player to give the journal ("Bring it to me when you're ready").
- [x] **Prefer `questRequired` over `requires`** — used throughout
- [x] **`expiryPeriod` not set** — investigative quest, no urgency
- [x] Item delivery has BOTH dialogue path AND quest YAML `item_give` trigger:
  - Vella's dialogue node `take_kingfisher` (questRequired 20-kingfisher_found) acknowledges via text + hint; quest engine item_give trigger does the actual grant + journal handover
  - Ulla's dialogue node responds to journal handover; quest engine item_give trigger does the grant + consume + flag default
- [x] **give.go gotcha:** Vella, Ulla, and Brindle each get player_give safety nets:
  - Vella: kingfisher BEFORE 20-kingfisher_found = return ("you've not yet found what was buried"); journal ANY TIME = return ("Ulla is the one who should read this, not me")
  - Ulla: kingfisher = return ("the kingfisher is for someone who knew Elgar from before. Take it to Vella."); journal BEFORE 20-vella_journal = return ("I've nothing to read until you bring it from Vella")
  - Brindle: journal = return ("Ulla is the one who should read this")
  - Other shopkeepers: not expected to be give-targets for quest items; if testing reveals problems, extend handlers
- [x] **Lost item recovery:** Two physical items handed out by NPCs:
  - Kingfisher (40060): given by room_interact at cemetery. If player loses it, they can re-search the cemetery — the hidden_noun's `room_interact` trigger should be re-fireable IF player has 20-kingfisher_found and missing 40060 in inventory. Add a CONDITIONAL re-grant in the quest engine: room_interact on "beneath Elgar's marker" with has 20-kingfisher_found + missing_item 40060 → give_item 40060 (no token grant; replacement only).
  - Journal (40061): given by Vella's dialogue. Add Vella dialogue node `lost_journal` triggers `["journal", "lost", "another", "again"]`, requires 20-vella_journal + missing_item 40061, gives replacement.
- [x] **NPC item handoff via dialogue:** Vella uses `givesItem: 40061` on `take_kingfisher` node OR the quest engine item_give trigger uses `give_item: 40061` action. RECOMMEND: quest engine item_give action for the primary handoff (consistent with quest 19's pattern); dialogue lost_journal as the recovery path.
- [x] `requiresItem`: not used (item handovers go through item_give triggers, not requiresItem).
- [x] Room behavior trees that give items: NOT NEEDED. The hidden_nouns are already discoverable via standard `search` mechanics; quest engine room_interact triggers fire on the noun discovery to advance the quest. Only the cemetery hidden_noun GIVES an item (kingfisher), and that's done via quest engine `give_item` action on the room_interact trigger, not via room behavior tree.
- [x] Mob groups: Ulla (humanoid, widow), Vella (humanoid, healer), Brindle (humanoid, merchant, blacksmith), Seren (humanoid, priest). None hostile, all charm-immune for the questgivers.
- [x] No physical item handed to player by Ulla at start — quest is verbal contract. The workshop is freely climbable; quest just frames the discovery.
- [x] Multi-zone: single zone (Stillwater).
- [x] `questExcluded` on completion nodes: every grant node excludes the granted token AND `20-end`.
- [x] **End-token exclusion:** `accept_inquiry` excludes `["20-start", "20-end"]`. Every other grant node excludes its own token + `20-end`.
- [x] Quest YAML `rewards` section: gold (modest, 75g — this is a story quest, not a bounty), itemid 10032 (Elgar's hook-spear), generic player message (specific flavor in dialogue).
- [x] Instance saves: list 3 (Ulla, Brindle, Seren) for the dialogue additions.
- [x] Line width: all dialogue and descriptions wrap at ≤80 chars.
- [x] No raw numbers in player-facing text: no "+2 perception" in spear description; describe as "the spear feels alive in your hand."
- [x] **Branching quests:** Flag declared (`truth: [whole, partial]`). The "branch" is end-state only; no mid-quest fork, no dismissal nodes needed.
- [x] **Flag-gated nodes:** Ulla's post-end root variant uses `questFlagRequired: {"20-truth": "whole"}` for the "you told me everything" greeting; default variant for partial.
- [x] **Dismissal nodes:** Not needed.
- [x] **Mid-quest variants:** Ulla, Brindle, Seren, and Vella all need root variants for every quest state they're relevant to. Substantial dialogue work — outlined in dialogue files.
- [x] **Quest items not components:** Verify 40060 and 40061 spec files have NO `is_component: true`. They are plain narrative items.

---

## REWARD STAT SPREAD — Elgar's Hook-Spear (item 10032)

Base spec mirrors lake-iron hook-spear (10031): one-handed piercing, ash shaft, hook on the socket. The Elgar variant has a **hand-picked stat spread totaling ~99 affix-budget points** — the same budget the engine spends on a baseline `arena 200` instance loot drop (`floor(7.0 * sqrt(200)) = 99`).

Per-player random rolls would require a new `give_affixed_item` quest engine action (~50 lines of engine work). Deferred to v2 polish; see "Design Notes" section. For v1, the spread is deterministic — the same item every player receives.

**Affix budget:** 99 points
**Spread (snowball-weighted, mirroring how the engine concentrates picks):**

| Affix | Cost | Ranks | Total Spend | Effect |
|-------|------|-------|-------------|--------|
| `damage_mult_phys` | 8/rank | +3 | 24 pts | `damage_multiplier: 1.10` (base 0.95 → 1.10) |
| `stat_perception` | 3/rank | +9 | 27 pts | `statmods.perception: 9` |
| `skill_weapon-combat` | 12/rank | +4 | 48 pts | `statmods.weapon-combat: 4` |
| **Total** | | | **99 pts** | |

**Final spec for 10032:**
```yaml
itemid: 10032
name: Elgar's hook-spear
namesimple: spear
description: >-
  A long-shafted hook-spear of refined lake-iron, finished by hands
  that knew exactly what the original carver had wanted. The leaf-
  blade is keener than common stock, the back-curving hook polished
  smooth from waiting on a smith's bench too long. A small fish-sigil
  is burned into the underside of the leather grip — Stillwater-born,
  Stillwater-finished. The spear feels alive in the hand, like it has
  been waiting to be carried.
type: weapon
hands: 1
subtype: piercing
weight: 5.0
speedmultiplier: 0.70
staminacost: 7
grapplemodifier: -3.0
damage:
  basedamage: 7
  variance: 2
damage_multiplier: 1.10
parryrating: 10
value: 250
statmods:
  perception: 9
  weapon-combat: 4
```

Comparable to a `Masterwork` or `Empowered` arena loot drop — concentrated in three affixes (the snowball pattern) rather than thinly spread.

---

## REWARDS

```
Gold: 100 (story quest, modest but meaningful — bumped from 75 per
      design feedback; the spear is the bigger reward)
Item: Elgar's hook-spear (item 10032) — see "Reward Stat Spread"
      section above for the full statline
Skill: none

Player message (default — fires on item_give of journal):
  "Ulla unfolds the journal page slowly. She reads it twice. The
   second time she sets it on her knee and puts both hands flat
   over it for a long moment, as if to keep it from blowing away.
   When she looks up, her eyes are wet but steady for the first
   time in years.
   'I knew it was something.' she says quietly. 'I knew it was
   something he could not say.'
   She rises with effort, walks to the workshop stair, and brings
   down a long wrapped bundle that has been sitting up there for
   years -- the hook-spear Brindle finished and never delivered.
   She places it in your hands without ceremony.
   'It was meant for someone who would understand. He'd want it
   to be you.'"

Player message (whole-truth path, fires after the truth-telling node):
  "Ulla is quiet for a long time. Then: 'Vella has been carrying
   that for as long as I have, then. Tell her -- tell her she did
   not cause it. Tell her her brother went down because that is
   who he was. Tell her I would like to see her at the loom one
   afternoon.'
   The shawl on her shoulders settles a little differently as she
   straightens. The grief has a shape now."

Room message:
  "is given a long wrapped bundle by Ulla, who closes her hand
   over hers for a moment before letting go."
```

**Brindle post-quest acknowledgment:** Add a Brindle dialogue node `quest_complete_thanks` requires 20-end, triggers `["thanks", "spear", "elgar", "ulla"]`, gives a brief text expressing relief — "Ulla brought it down? Good. I could not bring myself to take it to her, and she could not bear to ask. The spear should not have sat that long. Wear it well." No item handoff (the spear is already with the player from the quest reward).

---

## VERIFICATION PLAN

1. Restart server with all new files in place.
2. Walk to Ulla's Parlor (4137). `look ulla` — confirm description renders. `ask ulla quest` → 20-start granted.
3. Climb `up` to Uncle's Workshop (4138). `search bench-vise carving` (or `look bench-vise carving`) → 20-workshop_carving granted, hidden_noun reveals.
4. Walk to Temple of Stillwater (4123). `look pillars` → 20-temple_pillars granted via room_interact. (Alt test: skip this, then visit Seren and `ask seren spiral` → confirm grant via dialogue path.)
5. Walk to Temple Garden (4124). `search buried marker` → 20-garden_marker granted.
6. Walk to Old Chapel Ruin (4144). `look altar stone` → 20-chapel_altar granted. **If altar stone is not in the room's `nouns:` section, this step needs the noun added — verify before implementation.**
7. Walk to Wardstone Circle (4145). `look altar slab` → 20-wardstone_altar granted. **Same noun-presence check.**
8. Cast chrysalis-glow, descend to Bone Shoals (4130). `search cache` → 20-bone_shoals_cache granted, descriptive text reveals the contents (no item gain — note is descriptive only).
9. Return to Brindle's Smithy (4106). `ask brindle quest` (or `ask brindle elgar`) → 20-brindle_confession granted with confession dialogue.
10. Walk to Stillwater Cemetery (4140). `search beneath elgar's marker` → 20-kingfisher_found granted + kingfisher (40060) in inventory.
11. Walk to Healer's Cottage (4136). Verify Vella (mob 355) is spawned. `look vella` — confirm description. `give kingfisher vella` → 20-vella_journal granted + kingfisher consumed + journal (40061) in inventory + Vella's npc_say sequence fires.
12. Walk to Ulla's Parlor (4137). `give journal ulla` → 20-end granted + journal consumed + flag truth: partial set (default) + Elgar's hook-spear (10032) given via rewards block + 75g paid.
13. **Truth-telling test:** still at Ulla's parlor, `ask ulla truth` (or `ask ulla vella`) → flag overwritten to truth: whole + Ulla's whole-truth response fires.
14. **Lost item recovery tests:**
    a. Drop the kingfisher between cemetery and Vella; pick up; confirm normal.
    b. Drop the kingfisher and walk away. Return to cemetery and `search beneath elgar's marker` again → conditional re-grant should give a new kingfisher (or polite "you've already taken this" if no replacement implemented).
    c. After Vella gives journal, drop journal; walk away; return to Vella; `ask vella journal` → lost_journal node gives replacement.
15. **Safety net tests:**
    a. `give kingfisher ulla` → returned with redirect ("take it to Vella")
    b. `give kingfisher brindle` → returned with redirect (Brindle behavior tree)
    c. `give journal vella` → returned with redirect ("take it to Ulla")
    d. `give journal brindle` → returned with redirect
16. **Wrong-state tests:**
    a. `ask brindle elgar` BEFORE 20-bone_shoals_cache → guarded redirect, no grant
    b. `ask vella quest` BEFORE 20-brindle_confession → redirect-no-quest variant
    c. `give kingfisher vella` BEFORE 20-kingfisher_found (impossible without item) — should not be reachable
17. **Canon cleanup check:** `look ulla` → confirm description now says "twelve years ago" (changed from "four years ago" as part of this quest's implementation). `look workshop` (room 4138) → confirm matches "twelve years". `look elgar's grave` (room 4140) → confirm "fifteen years" wording (kept as-is, see Vella dialogue cover for the slight discrepancy).
18. After full completion, return to Brindle. `ask brindle thanks` (or `spear`) → quest_complete_thanks node fires.
19. **Reward check:** `look spear` (or `look elgar's hook-spear`) → confirm description renders. `equip spear` → confirm statmods apply (perception +9, weapon-combat +4). `status` → confirm bonuses visible. Confirm damage_multiplier 1.10 in combat feels meaningfully better than the standard hook-spear's 0.95.
20. **Quest 19 cross-reference test (requires both quests done):** with a character who has completed quest 19 with `19-completion: full`, complete quest 20. After 20-end, return to Ulla → confirm the "you killed the thing in the sump, Elgar would have wanted to be the one" root variant fires. Then test the negative case: with a character who did 19 partial OR didn't do 19 at all, confirm the standard terminal greeting fires (no acknowledgment).

---

## DESIGN NOTES (not part of file generation)

1. **Why no workshop key item:** Considered making a key (item) that Ulla hands over. Decided against — the workshop is already physically accessible (the stairs are visible from Ulla's parlor and the up exit is open). Quest grant frames the discovery as meaningful; the lock is emotional, not physical. Saves an item slot.

2. **Why two spear items (10031 and 10032):** Elgar's hook-spear (10032) is a flavor variant of the standard lake-iron hook-spear (10031), with arena-tier affix-equivalent stats baked in (~99 budget points: damage_mult +0.15, perception +9, weapon-combat +4). Players who haven't done quest 19 might encounter Elgar's spear first (good — it teaches them the hook-spear weapon archetype); players who have done both own a standard one and a meaningfully better story one. The Elgar spear is the upgrade path.

   **v2 polish (deferred):** add a `give_affixed_item` quest engine action that takes `item_id` + `gold_paid` and calls `items.GenerateAffixedItem` per-player. Would let Elgar's hook-spear roll random affixes per drop instead of being deterministic. ~50 lines of engine work + tests. For now, the deterministic spread is fine — a single named item with a known statline arguably suits a memento better than a random roll anyway.

3. **Why default flag is `partial`:** A player who hands over the journal and walks away has chosen the path of least involvement. The whole-truth path requires explicit follow-up (`ask ulla truth`). This rewards engagement without punishing the disengaged.

4. **Vella's dialogue is the second-heaviest writing in the zone:** After Ulla's. Vella is older, dignified, has been carrying a sister's grief without the public sympathy a widow gets. Her acceptance of the kingfisher should land — the mob YAML and dialogue need to set this up before the climactic moment.

5. **Brindle's character arc is a quiet B-plot:** He doesn't need a huge dialogue tree. The confession node is the one moment he opens up. His existing description's "half-finished hook-spear on his bench" already foreshadows it; the quest just pays it off.

6. **Date canon cleanup is GENUINELY important:** A player walking the zone and reading "four years" / "twelve years" / "fifteen years" for the same death will be confused. Pick one, fix the others. This is a 5-minute prep task before quest implementation.

7. **The temple priest Seren as alternate Step 3 path:** Including her gives players two ways to discover the temple connection — random examination OR direct inquiry. Both feel natural for an investigation quest. Two paths = two ways to fail = we should test both.

8. **The pre-Chrysalis lore is deliberately understated:** The quest reveals that the spiral predates the modern Chrysalis order without explaining what the spiral MEANT. That mystery is preserved for future content. Vella, Seren, and Elgar's journal hint that the spiral marks alignment points — but the WHY (what was being aligned, by whom, for what purpose) is left open. This is intentional. Don't over-explain.

9. **Sump dweller continuity with quest 19 (FOLDED IN):** Elgar's journal explicitly identifies the sump dweller as "what was waiting in the deep." Per design feedback, Ulla's terminal dialogue includes a root variant gated on `19-end + 19-completion: full` that acknowledges the avenging — see Step 11 description for the exact phrasing. The variant fires once when the player visits Ulla after both quests are complete with the full path on 19. Players who did 19 partial or skipped it get the standard terminal greeting (no penalty, no acknowledgment).

---

This is a planning document only — no game files have been written.

Review and annotate the plan, then run:
`/new-quest 20-ullas_silence.md`
to generate all files.
