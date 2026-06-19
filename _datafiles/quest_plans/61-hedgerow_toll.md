# QUEST PLAN — Hedgerow Toll (questid 61)

**Quest ID:** 61
**Zone(s):** North Road North, The Empty Reach
**Type:** Combat / kill (LINEAR — no flags, no branching)
**Quest giver:** Goodwife Pemberton (mob 9171, room 5378, North Road North — the Lake & Ladle common room)
**Completion NPC:** Goodwife Pemberton (same — accept and turn in at the inn)
**Targets:** A Hedgerow Bandit (mob 9173, hostile ambusher, North Road North room 5376) · The Reach-Touched (mob 9178, hostile ambusher, The Empty Reach room 5395)

**Design intent:** the second corridor quest after The Long Road (quest 60), and
the simplest robust pattern — a kill-clear modelled exactly on quest 33 "Hold
the Wash" (kill scout + kill squatter → report to Garve). One named target in
each of the first two corridor zones; order-independent; completion is a `report`
dialogue node gated on holding BOTH kill tokens. No items change hands, so none
of the give.go/consume_item machinery applies. Exercises the combat mobs we
placed and gives the innkeeper a reason to care about the road.

> **Kill-COUNT note:** the engine's proven pattern (quests 32–34) grants a token
> on the death of a *specific named mob* once (`event: mob_death`, `mob: <id>`,
> `missing: <token>`). There is no proven "kill N of the same mob" counter, and
> building one is exactly the kind of fragile custom mechanic we're avoiding.
> So v1 is **one Hedgerow Bandit + one Reach-Touched** (two distinct targets),
> not "kill 3 each." The targets respawn, so the road stays dangerous after.

---

## STEP 4A — PLAYER POV WALKTHROUGH (mandatory)

```
Step start: take the job from the innkeeper
  Player thinks: "An innkeeper — let me see if she has work."
  Player types:  "ask pemberton quest"
  Discovers via: Universal MUD SOP. Pemberton's root hint also flags trouble
                 on the road; her grant text names both targets + where.

Step bandit: kill the hedgerow bandit on the road
  Player thinks: "She said a bandit works the hedgerows just up the road north."
  Player types:  "attack bandit"   (or the mob engages first — it's a hostile
                                     ambusher)
  Discovers via: Pemberton's text names "A Hedgerow Bandit" in the hedgerows
                 north; the mob is hostile and in room 5376 on the way. Players
                 fight what attacks them.

Step reach: kill the Reach-Touched in the Empty Reach
  Player thinks: "And another — a Reach-touched thing — out in the empty
                  stretch beyond."
  Player types:  "attack reach-touched"   (or it ambushes first)
  Discovers via: Pemberton's text names "The Reach-Touched" in The Empty Reach
                 (the next zone north); hostile ambusher in room 5395.

Step end: report back to Pemberton
  Player thinks: "Both are down — back to the inn to tell her."
  Player types:  "ask pemberton report"   (also: ask pemberton quest)
  Discovers via: Pemberton's hint says to come back and tell her when both are
                 down; `ask <npc> report/quest` is SOP. Turn-in node also
                 triggers on quest/task/cleared/done.
```

**Thousand-mudder test:** every step is `ask <npc> quest`, `attack <hostile
mob the NPC just named>`, or `ask <npc> report`. No items, no nouns, no magic
words. Estimated pass rate >950/1000. (Only risk: a player kills a target
*before* accepting — handled below, they simply kill the respawn after
accepting; the targets respawn on their normal timers.)

---

## STEP CHAIN

```
Step 1: "start" — granted by dialogue node on Pemberton (9171)
  Trigger: ask pemberton quest/task → node `quest_start`, grantsQuest "61-start"
  Token: 61-start
  Description: "Goodwife Pemberton's trade is bleeding — her lodgers and the
    road's travellers are being robbed on the stretch north. She asked you to
    put down the two that work it: a hedgerow bandit on the road, and a
    Reach-touched thing out in the Empty Reach beyond."
  Hint: "Head north up the road. Kill the hedgerow bandit that works the
    hedges, and the Reach-Touched out in the Empty Reach beyond. Come back and
    tell Pemberton when both are down."

Step 2: "bandit" — granted by KILLING A Hedgerow Bandit (mob 9173)
  Trigger: mob_death 9173 → quest engine trigger, has [61-start], missing
           [61-bandit] → grant 61-bandit + send_text
  Token: 61-bandit
  Description: "The hedgerow bandit that worked the road north is dead. The
    Reach-Touched still haunts the Empty Reach beyond."

Step 3: "reach" — granted by KILLING The Reach-Touched (mob 9178)
  Trigger: mob_death 9178 → quest engine trigger, has [61-start], missing
           [61-reach] → grant 61-reach + send_text
  Token: 61-reach
  Description: "The Reach-Touched is down. Both road-preyers are dead — report
    back to Pemberton at the Lake & Ladle."

Step 4: "end" — granted by dialogue turn-in on Pemberton (9171)
  Trigger: ask pemberton report/quest → node `quest_turnin`, questRequired
           [61-start, 61-bandit, 61-reach] (AND), grantsQuest "61-end".
           Rewards block fires on 61-end.
  Token: 61-end
  Description: "Pemberton can promise her lodgers a safe road again. The toll
    the hedgerows took is paid off."
```

Steps 2 and 3 are **order-independent** (two separate mob_death triggers, each
gated only on `61-start` + its own `missing`). The turn-in requires both.

---

## ALTERNATIVE PATHS

```
Turn in via `ask pemberton quest` (not "report"):
  Mechanism: quest_turnin node triggers include "quest" and "task" (SOP), so
    asking about the quest with both kills held completes it. A quest_active
    node (questRequired [61-start], listed AFTER quest_turnin) handles the
    partial state and reminds which targets remain.

Kill a target BEFORE accepting the quest:
  Mechanism: the mob_death trigger requires has [61-start]; with no quest, the
    kill grants nothing. The target respawns; the player kills the respawn
    after accepting. No soft-lock. (Matches quest 33 behaviour — no pre-credit.)

Return to Pemberton with only one kill:
  Mechanism: quest_turnin's questRequired (needs BOTH kill tokens) fails, so
    dialogue falls through to quest_active, which names the remaining target.
```

---

## QUEST GATING DIAGRAM

```
[no quest] --ask pemberton quest--> [61-start]
[61-start] --kill Hedgerow Bandit 9173--> [61-bandit]   (order-independent)
[61-start] --kill Reach-Touched 9178--> [61-reach]      (order-independent)
[61-start + 61-bandit + 61-reach] --ask pemberton report--> [61-end] (REWARDS)
```

Verify: turn-in gated on all three tokens (AND); every grant node excludes
61-end; mob_death triggers can't re-fire (missing-own-token) and can't fire
without 61-start. No dead steps.

---

## FILES NEEDED

| Action | File | Purpose |
|--------|------|---------|
| CREATE | `quests/61-hedgerow_toll.yaml` | Quest def: 4 steps, 2 mob_death triggers, rewards. `linear: true` |
| CREATE | `dialogue/north_road_north/9171.yaml` | Pemberton: quest_start (grant), quest_turnin (gated on both kills), quest_active (partial), quest_done, inn/road flavor. (She has a `shop:` already; dialogue is a separate file and coexists — noncombat_shopkeeper falls through to dialogue on `ask`.) |

**No new items** — reward reuses Healing Salve (30036).
**No mob edits** — targets 9173/9178 are already hostile + spawned (5376 / 5395).
**No room edits.**
**Instance saves to clear before testing:** `mobs.instances/*`, `rooms.instances/*` (standard pre-smoke wipe).

---

## GOTCHAS CHECKLIST

- [x] grant node (`quest_start`) and turn-in node (`quest_turnin`) both include
      `"quest"` and `"task"` in triggers (SOP).
- [x] **Voice:** Pemberton `text` first person; hints narrator voice
      ("Head north…", "Come back and tell Pemberton…").
- [x] **Discoverability:** both target names appear in Pemberton's grant text;
      `attack`/`ask report` are universal. No magic words, no nouns.
- [x] **POV walkthrough (4a) complete:** all four steps, each ★★★★.
- [x] **Trigger tiers:** ask-quest (★★★★), mob_death of named hostiles
      (★★★★), ask-report (★★★★). Highest tier throughout.
- [x] **Thousand-mudder:** >950/1000.
- [x] **Narrator never overreaches:** send_text on each kill describes only the
      kill just made and names the remaining target — no mind-reading.
- [x] **`questRequired` over `requires`** (turn-in/active use quest tokens).
- [x] **No `expiryPeriod`** — omitted (no urgency design intent).
- [x] **Item delivery N/A** — no items change hands, so no item_give /
      consume_item / return_item / lost_report needed. (Documented: kill quest,
      narrative + token gating only.)
- [x] **Mob groups:** Pemberton (giver) is noncombat_shopkeeper, NOT in a
      hostile group. Targets 9173/9178 ARE hostile — that is the point.
- [x] **Multi-zone:** both targets confirmed spawned (9173@5376 NRN,
      9178@5395 The Empty Reach); Pemberton confirmed @5378.
- [x] **`questExcluded` on completion:** quest_turnin excludes 61-end;
      quest_start excludes [61-start, 61-end].
- [x] **End-token exclusion:** every grant node excludes 61-end.
- [x] **Node order:** quest_turnin listed BEFORE quest_active so the
      all-tokens-held case matches first; quest_active is the partial fallback.
- [x] **Rewards filled** (below).
- [x] **Line width** ≤80; **no raw numbers** in player-facing text (gold is the
      engine-shown exception).
- [x] **No flags / no branching** — N/A.

---

## REWARDS (on 61-end, fired by the quest_turnin dialogue node + rewards block)

```
Gold: 40
Item: Healing Salve (item 30036) — a road provision from the innkeeper
Skill: weapon-combat:1   (they earned it clearing the road; matches quest-33 precedent)
Player message: "Pemberton listens to the whole of it, then lets out a breath
  she seems to have been holding for a week. 'Both of them. Then I can tell my
  lodgers the road north is walkable again and not be lying.' She counts coin
  from the strongbox without haggling for once, and presses a salve into your
  hand on top of it. 'For the next set of bruises. You'll have them, doing work
  like this. Bed and board here are yours at a friend's rate from now on.'"
Room message: "Goodwife Pemberton settles a purse into the traveller's hand
  with the air of a woman whose trade just started breathing again."
```

(Reward keys are NO-underscore: `gold`, `itemid`, `skillinfo`, `playermessage`,
`roommessage` — matches quest 33.)

---

## VERIFICATION PLAN (in-game, harness)

1. Wipe instance saves; restart server. Confirm clean boot (quests
   loadedCount +1, no panic, ValidateZoneConsistency errors=0).
2. Teleport to 5378. `ask pemberton quest` → confirm 61-start granted, grant
   text names both targets.
3. Teleport to 5376. Kill A Hedgerow Bandit (9173) → confirm 61-bandit granted
   + send_text. (Confirm the mob engages / is attackable.)
4. Teleport to 5395. Kill The Reach-Touched (9178) → confirm 61-reach granted
   + send_text.
5. **Partial-turn-in test:** before the second kill, `ask pemberton report` →
   confirm quest_active fallback (names remaining target), NOT completion.
6. After both kills, teleport to 5378. `ask pemberton report` → confirm 61-end,
   +40 gold, Healing Salve received, weapon-combat skill bump, player+room
   messages fire.
7. **Re-grant test:** `ask pemberton quest` again → confirm quest_done node
   (not re-offered).
8. **Order-independence:** (optional second char) kill Reach-Touched first,
   then bandit → confirm both tokens still grant and turn-in works.

---

> This is a planning document only — no game files have been written.
>
> Review and annotate, then run:
> `/new-quest 61-hedgerow_toll.md`
> to generate all files.
