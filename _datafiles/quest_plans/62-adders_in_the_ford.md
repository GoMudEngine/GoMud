# QUEST PLAN — Adders in the Ford (questid 62)

**Quest ID:** 62
**Zone(s):** Greywater Flats
**Type:** Combat / kill (LINEAR, single-target bounty — no flags, no branching)
**Quest giver:** Tamsin Reed (mob 9191, room 5425 Ford Approach, Greywater Flats — the ferryman/ferrywoman; has a `shop:`, no dialogue yet)
**Completion NPC:** Tamsin Reed (same — accept and turn in at the ford landing)
**Target:** A Marsh Adder (mob 9195, hostile ambusher, room 5423, south through the reeds before the ford)

**Design intent:** the third corridor quest, deliberately the simplest robust
pattern — a single-target kill bounty (a focused variant of the quest-33/61
kill pattern). Tamsin's ferry passengers keep getting bitten in the reeds on
the south approach; she wants the adder that lairs there put down. One named
hostile, one report, a venom-themed reward. No items change hands on the quest
path, so none of the give.go machinery applies. Shorter than Hedgerow Toll (one
kill vs two) by design — the variety here is theme + reward, not length.

> **Why single-target:** Greywater Flats has exactly one combat mob (9195); the
> rest are passive (ferryman, warden, fisher) or prey (heron). The engine grants
> a token per *named* mob death once — no kill-counter — so "clear the adders"
> resolves to killing the one named adder (which respawns, keeping the ford
> dangerous after). This is honest and robust; we are not faking a counter.

---

## STEP 4A — PLAYER POV WALKTHROUGH (mandatory)

```
Step start: take the bounty from the ferryman
  Player thinks: "A ferryman at the crossing — let me see if there's work."
  Player types:  "ask tamsin quest"
  Discovers via: Universal MUD SOP. Tamsin's root hint flags the adder; her
                 grant text names the target and where (the reeds south, before
                 the ford).

Step adder: kill the marsh adder in the reeds
  Player thinks: "She said it lairs in the reeds just south, before the ford."
  Player types:  "south" (toward 5423), then "attack adder"  (the mob is a
                 hostile ambusher and may strike first)
  Discovers via: Tamsin's text names "A Marsh Adder" in the reeds to the south;
                 the mob is hostile and lairs in room 5423 on that approach.
                 The feel-test confirmed it is hostile-on-look and aggressive.

Step end: report back to Tamsin
  Player thinks: "It's dead — back to the ferryman to collect."
  Player types:  "ask tamsin report"   (also: ask tamsin quest)
  Discovers via: Tamsin's hint says to come back and tell her when it's done;
                 `ask <npc> report/quest` is SOP. Turn-in node also triggers on
                 quest/task/done/dead/cleared.
```

**Thousand-mudder test:** `ask quest` → `attack <hostile the NPC just named, two
rooms away>` → `ask report`. No items, nouns, or magic words. Estimated pass
rate >950/1000. (Only edge: killing the adder before accepting just no-credits;
it respawns and the player kills the respawn after accepting.)

---

## STEP CHAIN

```
Step 1: "start" — granted by dialogue node on Tamsin Reed (9191)
  Trigger: ask tamsin quest/task → node `quest_start`, grantsQuest "62-start"
  Token: 62-start
  Description: "Tamsin Reed's ferry passengers keep getting struck in the reeds
    on the south approach to the ford. She asked you to put down the marsh adder
    that lairs there."
  Hint: "Head south into the reeds before the ford and kill the marsh adder.
    Come back and tell Tamsin at the ford landing when it's dead."

Step 2: "adder" — granted by KILLING A Marsh Adder (mob 9195)
  Trigger: mob_death 9195 → quest engine trigger, has [62-start], missing
           [62-adder] → grant 62-adder + send_text
  Token: 62-adder
  Description: "The marsh adder that lairs in the reeds is dead. Report back to
    Tamsin at the ford landing."

Step 3: "end" — granted by dialogue turn-in on Tamsin Reed (9191)
  Trigger: ask tamsin report/quest → node `quest_turnin`, questRequired
           [62-start, 62-adder] (AND), grantsQuest "62-end". Rewards block fires
           on 62-end.
  Token: 62-end
  Description: "Tamsin can pole her passengers across without watching the reeds
    for a strike. The ford is the ferryman's again."
```

---

## ALTERNATIVE PATHS

```
Turn in via `ask tamsin quest` (not "report"):
  Mechanism: quest_turnin triggers include "quest"/"task" (SOP). With 62-adder
    held it completes; a quest_active node (questRequired [62-start], listed
    AFTER quest_turnin) handles the not-yet-killed state.

Kill the adder BEFORE accepting:
  Mechanism: mob_death trigger requires has [62-start]; with no quest the kill
    grants nothing. The adder respawns; the player kills the respawn after
    accepting. No soft-lock.

Report with the adder not yet dead:
  Mechanism: quest_turnin's questRequired (needs 62-adder) fails → falls through
    to quest_active, which re-points the player south to the reeds.
```

---

## QUEST GATING DIAGRAM

```
[no quest] --ask tamsin quest--> [62-start]
[62-start] --kill Marsh Adder 9195--> [62-adder]
[62-start + 62-adder] --ask tamsin report--> [62-end] (REWARDS)
```

Verify: turn-in AND-gated on both tokens; every grant node excludes 62-end;
the mob_death trigger can't re-fire (missing-own-token) or fire without
62-start; no dead steps; quest_turnin listed before quest_active.

---

## FILES NEEDED

| Action | File | Purpose |
|--------|------|---------|
| CREATE | `quests/62-adders_in_the_ford.yaml` | Quest def: 3 steps, 1 mob_death trigger, rewards. `linear: true` |
| CREATE | `dialogue/greywater_flats/9191.yaml` | Tamsin: quest_start (grant), quest_turnin (gated on 62-adder), quest_active (partial), quest_done, ferry/ford flavor. (She has a `shop:` already; dialogue coexists — noncombat_shopkeeper falls through to dialogue on `ask`.) |

**No new items** — reward reuses Minor Antidote (30028).
**No mob edits** — target 9195 already hostile + spawned (5423).
**No room edits.**
**Instance saves to clear before testing:** `mobs.instances/*`, `rooms.instances/*` (standard pre-smoke wipe).

---

## GOTCHAS CHECKLIST

- [x] grant node (`quest_start`) and turn-in node (`quest_turnin`) both include
      `"quest"` and `"task"` in triggers (SOP).
- [x] **Voice:** Tamsin `text` first person; hints narrator voice.
- [x] **Discoverability:** target name + location appear in Tamsin's grant text;
      `attack`/`ask report` universal. No magic words, no nouns.
- [x] **POV walkthrough (4a) complete:** all three steps, each ★★★★.
- [x] **Trigger tiers:** ask-quest (★★★★), mob_death of named hostile (★★★★),
      ask-report (★★★★).
- [x] **Thousand-mudder:** >950/1000.
- [x] **Narrator never overreaches:** kill send_text describes only the kill.
- [x] **`questRequired` over `requires`**; **no `expiryPeriod`**.
- [x] **Item delivery N/A** — no items change hands (kill quest, token gating).
- [x] **Mob groups:** Tamsin (giver) noncombat_shopkeeper, not hostile-grouped;
      target 9195 IS hostile (the point).
- [x] **NPC spawned:** 9191@5425, 9195@5423 confirmed.
- [x] **`questExcluded`:** quest_turnin excludes 62-end; quest_start excludes
      [62-start, 62-end]; every grant node excludes 62-end.
- [x] **Node order:** quest_turnin BEFORE quest_active.
- [x] **Rewards filled** (below); **line width** ≤80; **no raw numbers**.
- [x] **No flags / no branching** — N/A.

---

## REWARDS (on 62-end, fired by the quest_turnin dialogue node + rewards block)

```
Gold: 30
Item: Minor Antidote (item 30028) — venom-themed, fitting the adder
Skill: none   (varies it from Hedgerow Toll's weapon-combat bump)
Player message: "Tamsin works the pole-callus on her palm with a thumb and
  looks south toward the reeds, where nothing is moving for once. 'That thing
  put two of my passengers in the mud this season and near cost me a third.
  It's a kinder crossing without it.' She counts out coin and tucks a small
  stoppered vial in on top. 'Antivenom. Keep it -- the reeds breed more than
  one of those, and the next one might find you before you find it.'"
Room message: "Tamsin Reed presses coin and a small vial into the traveller's
  hand, glancing south toward the quiet reeds."
```

(Reward keys NO-underscore: `gold`, `itemid`, `playermessage`, `roommessage`.)

---

## VERIFICATION PLAN (in-game, harness — WIELD A MELEE WEAPON FIRST)

> Lesson from quest 61: the admin smoketester defaults to a Hunting Bow, which
> does little melee damage and lets ambushers flee. Before fighting:
> `remove bow; wield steel longsword`.

1. Wipe instance saves; restart server. Confirm clean boot (quests loadedCount
   +1, no panic, ValidateZoneConsistency errors=0).
2. Teleport to 5425. `ask tamsin quest` → confirm 62-start granted, grant text
   names the adder + the reeds south.
3. Teleport to 5423. Kill A Marsh Adder (9195) → confirm 62-adder granted +
   send_text. (Confirm the mob engages.)
4. **Partial-turn-in test:** (optional) before the kill, `ask tamsin report` →
   confirm quest_active fallback, NOT completion.
5. After the kill, teleport to 5425. `ask tamsin report` → confirm 62-end,
   +30 gold, Minor Antidote received, player+room messages fire.
6. **Re-grant test:** `ask tamsin quest` again → confirm quest_done node
   (not re-offered).

---

> This is a planning document only — no game files have been written.
>
> Review and annotate, then run:
> `/new-quest 62-adders_in_the_ford.md`
> to generate all files.
