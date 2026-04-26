# QUEST: The Lake-Caves Bounty

**Quest ID:** 19
**Zone(s):** Stillwater (single-zone)
**Type:** Combat / escalating bounty (multi-tier completion)
**Quest giver:** Constable Drunn (mob 335, room 4110 Constabulary)
**Alt redirect NPC:** Dock master Arn (mob 342, room 4116 Fishing Docks) — points players at Drunn but does NOT grant
**Completion NPC(s):** Drunn (4110) OR Arn (4116) — either accepts the leviathan tooth

---

## CONCEPT

Drunn's constabulary has posted a bounty on the cave creatures spilling into the lake — the shrimp swarms are eating the fishing nets, the drowned hunters have killed two fishermen this season, and one of the more imaginative dock workers swears something far worse lives in the deep sump. The bounty escalates in tiers: clear the shallows, clear the hunters, then (if you survive the boss) bring back proof.

The quest is the WARMUP for Stillwater's quest content — mechanically straightforward (combat objective, item handover completion), no branching dialogue mysteries, validates all the quest-engine plumbing before we tackle the more delicate Voss family thread next.

---

## STEP CHAIN

```
Step 1: "start" — granted by dialogue
  Trigger: `ask drunn quest` (or `task`) → tree node `accept_bounty`
           with grantsQuest: "19-start"
  Token: 19-start
  Description: "Constable Drunn has posted a bounty on the lake-cave
                creatures. Clear the shallows of the shrimp swarms,
                deal with at least one drowned hunter, then come back
                to the constabulary for further instructions."
  Hint: "Take a light source — your chrysalis-glow spell or a lantern.
         The caves are dark. The cave mouth is east of the fishing
         docks. Type `ask drunn quest` again here for a reminder."

Step 2: "shrimp" — auto-granted on mob_kill
  Trigger: quest engine event mob_killed for mob 330 (skitter shrimp swarm)
  Token: 19-shrimp
  Description: "You have cleared a swarm of skitter-shrimp from the
                cave shallows. The fishing nets will hold longer."
  Hint: "A drowned hunter still lurks deeper in the cave system. Clear
         at least one, then walk back to Constable Drunn at the
         constabulary and `ask drunn quest` to report."

Step 3: "hunter" — auto-granted on mob_kill
  Trigger: quest engine event mob_killed for mob 331 (drowned hunter)
  Token: 19-hunter
  Description: "You have killed a drowned hunter. The shallows are
                meaningfully safer."
  Hint: "Walk back to Constable Drunn at the constabulary and
         `ask drunn quest` to report. (If you haven't cleared a
         shrimp swarm yet, do that first.)"

Step 4: "signed" — granted by dialogue (at Drunn) once shrimp + hunter held
  Trigger: `ask drunn quest` (or `task`) with both 19-shrimp AND 19-hunter
           → tree node `sign_bounty` with grantsQuest: "19-signed"
  Token: 19-signed
  Description: "Drunn has signed the upgraded bounty. The constabulary
                will pay handsomely for proof that the deepest threat
                in the sump has been put down — but the warrant is now
                yours to pursue or to walk away from. Either way you've
                earned the partial payment."
  Hint: "Hand Drunn a leviathan tooth (`give tooth drunn`) from the
         sump dweller for the full reward — or `ask drunn quest` again
         here to close the bounty and take the partial payment now.
         You can also turn in the tooth to dock master Arn at the
         fishing docks if you'd rather."

Step 5: "end" — three completion paths, all flag-tracked
  Path A (FULL): `give tooth drunn` (item 40054 to mob 335)
                 → quest engine item_give trigger requires 19-signed,
                 grants 19-end + sets flag completion: full
  Path B (FULL alt): `give tooth arn` (item 40054 to mob 342)
                 → same: requires 19-signed, grants 19-end + sets flag
                 completion: full
  Path C (PARTIAL): `ask drunn quest` at Drunn with 19-signed (and no
                 tooth handover) → tree node `take_partial` grants
                 19-end + sets flag completion: partial
  Path D (PARTIAL alt): `ask arn quest` at Arn with 19-signed →
                 same: tree node `take_partial` grants 19-end + sets
                 flag completion: partial
  (NB: item_give triggers gate on `has: ["19-signed"]`. If a player
   gives the tooth before having 19-signed, the trigger fails — and
   per the give.go gotcha, the tooth would be consumed silently. So
   Drunn AND Arn need behavior tree player_give handlers with
   return_item action for item 40054 when 19-signed is NOT held — see
   Files Needed.)
  Token: 19-end
  Description: "The lake-caves bounty is closed. The shallows are
                safer. The lake-folk owe you a quiet round at the inn."
```

---

## FLAG DECLARATION

```yaml
flags:
  - key: completion
    values: [partial, full]
    description: "Whether the player took the partial bounty after signed,
                  or pushed for the boss kill and brought back the tooth"
```

The flag does NOT change which followup quests unlock (no Stillwater followups gate on this — yet). Its purpose is:
1. Drunn's post-quest dialogue varies by path (acknowledgment of full vs partial)
2. Brindle the smith offers a one-time free lake-iron nodule to full-completion players (gated dialogue node, see below)
3. Future quests CAN reference it if we want a "proven heroes" recognition system

---

## ALTERNATIVE PATHS

```
Step 1 alternative — Arn redirect:
  Trigger: `ask arn quest` (or `task`) → tree node `redirect_to_drunn`
           with NO grantsQuest. Just a hint pointing at the constabulary.
  Mechanism: dialogue tree node, no quest token granted.
  Text: Arn explains the cave problem, says Constable Drunn handles
        the official bounties at the constabulary, points the player
        north to town. Hint: "You could ask Constable Drunn at the
        constabulary about the bounty."

Step 5 alternative A — give tooth to Arn instead of Drunn:
  Mechanism: quest engine item_give trigger on quest YAML targets BOTH
             mob 335 AND mob 342. Same grant, same flag.
  Text: Arn dialogue thanks the player; says he'll pass word to Drunn
        and the docks will rest easier.

Step 5 alternative B/D — partial completion at Drunn or Arn:
  Trigger: tree node `take_partial` (on each NPC) keyed on triggers
           including "quest", "task", "close", "pay", "payment",
           "done", "bounty"
  Mechanism: dialogue node grants 19-end with flag completion: partial.
             No tooth required, no item consumed.
  Note: At state `19-signed`, `ask drunn quest` matches `take_partial`
        (since `sign_bounty` is now `questExcluded` by 19-signed).
        This means asking for the quest at signed-state explicitly
        takes the partial path — players who want full must `give tooth
        drunn` BEFORE asking quest. The hint at step 4 makes this
        choice explicit and discoverable.

Step 4 alternative — visit Arn at signed step:
  Mechanism: Arn dialogue root variant for
             `questRequired: ["19-signed"]` + `questExcluded: ["19-end"]`
             greets player as a known agent, mentions the boss is still
             down there if they want the full reward, and hints at the
             give-tooth or ask-quest options at Arn too.
```

---

## QUEST GATING DIAGRAM

```
[no quest] ──ask drunn quest──▶ [19-start]
[no quest] ──ask arn quest──▶ (redirect, no token)

[19-start] ──kill mob 330──▶ [19-shrimp]
[19-start] ──kill mob 331──▶ [19-hunter]
   (these can be earned in either order; quest engine grants whichever
    fires first; the second auto-fires when its mob is killed)

[19-shrimp] + [19-hunter] ──ask drunn quest──▶ [19-signed]
   (sign_bounty fires; take_partial doesn't match yet because it
    requires 19-signed)

[19-signed] ──give tooth (40054) to Drunn (335) or Arn (342)──▶
   [19-end] + flag completion: full
   (item_give trigger; tooth consumed)

[19-signed] ──ask drunn quest  OR  ask arn quest──▶
   [19-end] + flag completion: partial
   (take_partial fires; sign_bounty is questExcluded by 19-signed.
    Player who wants full path must give tooth BEFORE asking quest —
    step-4 hint makes this explicit.)
```

Verification:
- Step 2 and 3 can be earned in either order (no dependency between them) ✓
- Step 4 explicitly requires BOTH 2 and 3 (dialogue node has both in `questRequired`) ✓
- Step 5 has three valid paths, all granting `19-end` ✓
- `questExcluded: ["19-end"]` on `accept_bounty` prevents re-grant ✓
- `questExcluded: ["19-signed", "19-end"]` on `sign_bounty` prevents re-grant ✓
- `questExcluded: ["19-end"]` on `take_partial` prevents re-grant ✓

---

## FILES NEEDED

| Action | File | Purpose |
|--------|------|---------|
| CREATE | `quests/19-the_lake_caves_bounty.yaml` | Quest definition with 5 steps, flag declaration, mob_killed triggers (330, 331), item_give triggers (Drunn, Arn), rewards block |
| CREATE | `dialogue/stillwater/335.yaml` | Constable Drunn dialogue — quest grant, signed step, partial completion path, post-completion variants |
| CREATE | `dialogue/stillwater/342.yaml` | Dock master Arn dialogue — Arn-redirect at start, signed-step acknowledgment variant, partial completion path, post-full-completion variant |
| MODIFY | `dialogue/stillwater/337.yaml` | Smith Brindle — add post-completion node giving free lake-iron nodule to flag completion: full players (file does not yet exist; CREATE) |
| MODIFY | `mobs/stillwater/337-smith_brindle.yaml` | Add behavior tree `player_give` handler with `return_item` action for item 40054 (leviathan tooth) — Brindle politely refuses and points player at the constabulary |
| MODIFY | `mobs/stillwater/335-constable_drunn.yaml` | Add behavior tree `player_give` handler with `return_item` for item 40054 when `19-signed` is NOT held (safety against give.go gotcha) |
| MODIFY | `mobs/stillwater/342-dock_master_arn.yaml` | Same handler as Drunn — `player_give` returns 40054 if 19-signed missing |

**No new items needed.** The leviathan tooth (40054) already exists with `dropchance: 100` on the sump dweller. The optional commendation item could be a flavor reuse — recommend item 40037 (guard-captain's-commendation, already used in quest 14) repurposed as a generic constabulary token, OR leave it gold-only (cleaner). **Recommend gold-only with a flavor message; no commendation item.**

**No new room edits needed.** The notice-board, Crier's notice-frame, and Cave Mouth notice are already in place from zone build — they reinforce discoverability passively.

**Instance saves to delete after work:**
- `_datafiles/world/dogmud/mobs.instances/stillwater/337-smith_brindle.yaml` (if exists) — Brindle behavior change won't apply otherwise

---

## GOTCHAS CHECKLIST

- [x] Every `grantsQuest` dialogue node has `"quest"` and `"task"` in triggers
  - `accept_bounty` (Drunn): triggers `"quest", "task", "bounty", "caves", "cave", "fishermen", "nets"`
  - `sign_bounty` (Drunn): triggers `"quest", "task", "bounty", "report", "tooth", "boss", "sump", "dweller"`
  - `take_partial` (Drunn): triggers `"quest", "task", "bounty", "close", "pay", "payment", "done"`
  - `take_partial` (Arn): same triggers as Drunn's `take_partial`
- [x] Patterns introducing the quest also include `"quest"` and `"task"` in keywords (Drunn's pattern entry for "bounty/cave/notice")
- [x] **Narrative voice:** Drunn first-person ("I"), Arn first-person ("I"), Brindle first-person. Hints in narrator voice ("You could ask Drunn about the bounty terms.") No 3rd-person self-references.
- [x] **Trigger discoverability** — every trigger word above is reachable
      without esoteric guesswork:
  - `"bounty"` — appears in 4 room nouns (gate notice-board 4100,
    Crier's notice-frame 4108, constabulary postings 4110, Cave Mouth
    notice 4121); also in Drunn's posted notice text
  - `"caves"` / `"cave"` / `"sump"` — in all 4 bounty notices, in
    Arn's dock dialogue, in cave room titles (Cave Mouth, Hollow Sump)
  - `"fishermen"` / `"nets"` — in bounty notice text ("nets being
    raided"), in Arn's dock dialogue
  - `"tooth"` / `"leviathan"` — in the leviathan-tooth-trophy item
    description AND in Drunn's signed-step dialogue ("bring back a
    leviathan tooth")
  - `"sump"` / `"dweller"` — appears in Hollow Sump room title and
    description (room 4131)
  - `"boss"` — in Drunn's signed-step dialogue and Arn's signed-step
    variant
  - `"close"` / `"pay"` / `"payment"` — in the step-4 hint (player is
    explicitly told they can ask to close the bounty for partial)
  - `"report"` — in step-3 hint and in Drunn's variant text
  - `"done"` — universal informal closure word; included as backup
  - **DROPPED from earlier draft:** `"help"`, `"work"`, `"back"`,
    `"finish"`, `"partial"` — these are either too generic or not
    discoverable from any in-game source. Replaced with words that
    appear in notices, item descriptions, room titles, hints, or NPC
    dialogue.
- [x] **SOP — every step advances via `ask <npc> quest`:**
  - Step 1 grant: `ask drunn quest` at no-quest state → grants 19-start
  - Step 4 grant: `ask drunn quest` at 19-shrimp + 19-hunter state →
    grants 19-signed (sign_bounty wins because take_partial requires
    19-signed in `questRequired`, so it doesn't match yet)
  - Step 5 partial: `ask drunn quest` at 19-signed state → grants
    19-end with flag partial (take_partial wins because sign_bounty is
    `questExcluded` by 19-signed)
  - Step 5 full: `give tooth drunn` (item_give trigger; not via ask)
    — but `ask drunn quest` STILL works, it just takes the partial
    path. Player must choose deliberately by giving tooth before
    asking. Step-4 hint communicates this clearly.
  - Steps 2 and 3 advance via `mob_killed` events, not dialogue —
    `ask drunn quest` between steps still gives helpful status (via
    Drunn's root variants) but does not auto-advance, since the
    advancement criterion is "kill the mob" not "talk to NPC"
- [x] **Prefer `questRequired` over `requires`** — used throughout. No `requires:` fields in the quest dialogue.
- [x] **`expiryPeriod` not set** — Drunn and Arn dialogue files use `memory: { expiryPeriod: "" }` per SOP. Quest is non-urgent.
- [x] Item delivery has BOTH dialogue path AND quest YAML `item_give` trigger:
  - Quest YAML triggers fire for mob 335 AND mob 342 with item 40054
  - Drunn and Arn dialogue files have variant root text acknowledging the handover
- [x] **give.go gotcha:** Brindle the smith is most likely tempt-target for "I'll give my tooth to the smith." Brindle gets a behavior tree `player_give` handler with `return_item` action for item 40054 — returns it with a polite refusal. Other shopkeepers (apothecary, weaver, jewelcrafter, fishmonger, storekeeper) are unlikely to be given a leviathan tooth, but if testing reveals a problem we can extend the handler. Pearl-carver Kess is the second-most-likely target — flag for follow-up.
- [x] **Lost item recovery:** Drunn does NOT hand the player a physical item. The leviathan tooth comes from killing the boss; if the player loses it (drops it, sells it), they need to wait for the boss to respawn. Acceptable risk; no recovery dialogue needed. Drunn's dialogue at `19-signed` reminds the player to keep the tooth safe.
- [x] **NPC item handoff via dialogue:** No `givesItem` needed for this quest (no items handed to player). Optional Brindle "free nodule" gift uses `givesItem: 40059` on a flag-gated node.
- [x] `requiresItem`: not used (delivery is via item_give triggers, not requiresItem-on-dialogue).
- [x] Room behavior trees that give items: not used.
- [x] Mob groups: Drunn (`humanoid, guard`), Arn (`humanoid, fisher`), Brindle (`humanoid, merchant, blacksmith`). None hostile, all charm-immune.
- [x] No physical item handed to player by quest giver — quest is verbal contract + bounty-board posting (already in room descriptions).
- [x] Multi-zone: single zone (Stillwater). Skip.
- [x] `questExcluded` on completion nodes: `take_partial` and the item_give dialogue acknowledgments all `questExcluded: ["19-end"]`.
- [x] **End-token exclusion:** `accept_bounty` excludes `["19-start", "19-end"]`. `sign_bounty` excludes `["19-signed", "19-end"]`. `take_partial` excludes `["19-end"]`.
- [x] Quest YAML `rewards` section: gold 50, no item, generic player message. Flag-conditional flavor in dialogue.
- [x] Instance saves: only Brindle (337) for the player_give handler change.
- [x] Line width: all dialogue and descriptions wrap at ≤80 chars.
- [x] No raw numbers in player-facing text: bounty payment described as "a heavy purse" / "a fair purse" depending on full vs partial; no "you receive 50 gold" verbiage.
- [x] **Branching quests:** Flag declared in quest YAML (`completion: [partial, full]`). The "branch" here is end-state, not a fork mid-quest, so dismissal nodes for wrong-path are NOT needed. Both NPCs honor both paths. Root variants on Drunn and Arn use `questFlagRequired` to reflect chosen path post-completion.
- [x] **Flag-gated nodes:** Brindle's "free nodule" gift uses `questFlagRequired: {"19-completion": "full"}`. Drunn's post-quest greeting variant uses `questFlagRequired: {"19-completion": "full"}` for the "you're a hero of the docks" line.
- [x] **Dismissal nodes:** Not needed — neither NPC has a competing path that excludes the other.
- [x] **Mid-quest variants:** Drunn root variants for `19-start` (in progress), `19-shrimp+19-hunter` (ready to sign), `19-signed` (carry on or take partial), `19-end + full` (post-completion full), `19-end + partial` (post-completion partial). Arn variants for the same milestones, framed from his side.
- [x] **Quest items not components:** Leviathan tooth (40054) — confirm `is_component: false` (unset/missing). VERIFIED in current YAML — has no `is_component` field, defaults false. ✓

---

## REWARDS

```
Gold: 50 (baseline, both paths)
Item: none from quest YAML reward block
Skill: none

Player message (full path, granted via Drunn dialogue node):
  "Constable Drunn weighs the leviathan tooth in his hand and lets out
  a low whistle. 'I'd half-decided that thing was a story the dock-
  drunks made up. Stillwater owes you for this — the docks will sleep
  easy for a long while.' He counts out a heavy purse and adds a
  smaller one on top. 'Drinks at the Pike & Lantern are on the
  constabulary tonight. Tell Sigrid I sent you.'"

Player message (full path, alt via Arn dialogue):
  "Dock master Arn turns the leviathan tooth over in his hands twice
  and goes very still. 'I'll be damned. There really WAS one.' He
  looks up at you with an expression that's part awe, part guilt for
  the men he sent down. 'I haven't got Drunn's purse, but the dockmen
  have a fund for things like this. Take it. And — thank you.'"

Player message (partial path, via Drunn or Arn dialogue):
  "{Drunn|Arn} nods grimly. 'The shallows are quieter — that's worth
  paying for. The bigger thing in the sump can wait for someone with
  more steel. Take the bounty; it's earned.' A fair purse changes
  hands."

Room message:
  "{Drunn|Arn} hands over a purse with a nod of respect."

(Quest YAML rewards block fires on 19-end regardless of path; the
 dialogue node carrying the grant supplies the path-specific flavor.)
```

**Brindle bonus (full path only, post-completion):**
  Brindle dialogue node `hero_thanks` gated on `19-completion: full`,
  triggers `["thanks", "lake-iron", "nodule", "discount"]`,
  `givesItem: 40059` (one free lake-iron nodule),
  text describes Brindle's quiet appreciation and the gift. Single use
  via additional `questExcluded` on a sub-flag if we want to enforce
  one-time, OR just use `requires` on the dialogue node visited list.
  RECOMMEND: skip the free-nodule mechanic in v1 to keep the quest
  shippable; add as v2 polish if it tests well.

---

## VERIFICATION PLAN

1. Restart server with new files in place.
2. Walk to 4110 Constabulary → `look` → `look postings` → `ask drunn quest`. Confirm quest granted (token 19-start in `questtoken list`).
3. Walk to 4116 Fishing Docks → `ask arn quest` BEFORE having 19-start. Confirm Arn redirects to Drunn, no token granted (regression test).
4. Cast `chrysalis-glow`, descend 4121 → 4127 → 4128. Kill skitter shrimp swarm (mob 330). Confirm `questtoken list` shows 19-shrimp.
5. Continue to 4129/4130. Kill drowned hunter (mob 331). Confirm 19-hunter granted.
6. Return to Drunn at 4110. `ask drunn quest`. Confirm `sign_bounty` node fires, 19-signed granted, dialogue mentions the boss path option.
7. PARTIAL PATH TEST: at 19-signed state, immediately type
   `ask drunn quest`. Confirm `take_partial` fires (NOT `sign_bounty`,
   which is now questExcluded). Confirm 19-end granted with flag
   completion: partial. Quest YAML rewards fire (gold + message).
   This is the critical SOP test — `ask quest` must be the universal
   advance command, even for the partial-completion path.
8. (NEW CHARACTER OR RESET) Repeat through 19-signed. This time, descend to 4131 Hollow Sump, kill sump dweller (mob 332), loot leviathan tooth (40054).
9. `give tooth drunn` at 4110. Confirm 19-end granted with flag completion: full. Confirm tooth consumed. Confirm reward fires + dialogue flavor reflects full path.
10. Test alternative full path: kill the boss, then `give tooth arn` at 4116. Confirm same completion via Arn.
11. Test give-tooth-to-Brindle (the player_give safety net): `give tooth brindle`. Confirm tooth is returned with refusal text. Item still in player inventory.
12. After full completion, return to Drunn. Confirm post-quest variant greeting reflects "hero of the docks" framing. Same for Arn.
13. After partial completion, return. Confirm post-quest variant is more subdued.
14. (FUTURE-PROOF) Test that a player who completes the quest does NOT see the bounty notice phrasing on `look notice-board` shift inappropriately — the postings stay static (this is a flavor check, not a regression).
15. Confirm `recipes` and shop interactions at smith Brindle (337) still work normally for full-path players (the new `hero_thanks` node should not break the existing shop/talk loop).

---

## DESIGN NOTES (not part of file generation)

1. **Why no commendation item:** Quest 14 uses item 40037 (guard-captain's-commendation). For Stillwater, adding another commendation item risks inventory clutter. Gold-only baseline + flag-tracked dialogue flavor is cleaner. If we later want a Stillwater Constabulary Token to function as a faction-style item, that can be added.

2. **Why no separate "drove off the boss but didn't kill it" path:** Considered, but the sump dweller's mutations (regenerative-tissue, dense-muscles) make "drive off" mechanically tricky to detect — there's no flee state we track. Boss is binary: kill (drops tooth) or don't engage. Acceptable.

3. **The Bone Shoals (4130) hidden cache (E.V. → Brindle note):** Do NOT consume in this quest. That cache is seeded for the Voss family quest (questid 20, next sketch). If the player finds it during quest 19, the note is still readable but no quest token fires. Quest 20 will use it.

4. **Mob respawn vs quest progress:** If a player kills the shrimp/hunter, gets stuck on `signed` (forgets to return to Drunn), and the mobs respawn — re-killing them is harmless. The mob_killed event fires regardless, but quest engine triggers ignore the event if the token is already held.

5. **What if the player kills the boss BEFORE accepting the quest?** They'll have a leviathan tooth in inventory but no quest. `give tooth drunn` should then... not advance the quest, because no `19-signed` is held. They need to ask Drunn for the quest first, then go through the chain. The tooth stays in inventory. They can continue the quest from `19-start` and skip directly to `19-signed` if they have BOTH 19-shrimp AND 19-hunter — which only auto-grant on kill events. So a pre-quest boss-killer who returns later still has to clear shrimp + hunter to advance. **This is acceptable and arguably good** — the bounty escalation is the design intent.

6. **The chrysalis-glow hint in Drunn's dialogue:** important atmospheric detail AND practical gameplay info. Drunn's quest-grant dialogue should mention "the caves are dark — bring a light or know how to make one." This sets up the cave navigation friction without spoiling the boss difficulty.

---

This is a planning document only — no game files have been written.

Review and annotate the plan, then run:
`/new-quest 19-the_lake_caves_bounty.md`
to generate all files.
