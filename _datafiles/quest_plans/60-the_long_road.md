# QUEST PLAN — The Long Road (questid 60)

**Quest ID:** 60
**Zone(s):** North Road North → Hartcharn → Kingsbarrow Vale → New Plymouth Outskirts
**Type:** Multi-zone delivery relay (LINEAR — no flags, no branching, no choice nodes)
**Quest giver:** Toller Garrick (mob 9170, room 5375, North Road North)
**Completion NPC:** An East Gate Guard (mob 9208, room 5471, New Plymouth Outskirts)
**Relay NPCs:** Severin Pell (mob 9182, room 5409, Hartcharn) · Tithe-Clerk Verrold (mob 9198, room 5445, Kingsbarrow Vale)

**Design intent:** the FIRST corridor quest and the corridor's narrative
spine. A capital-bound dispatch travels up the road leg by leg, pulling the
player through all five new zones. Every advancement uses the two highest-
discoverability mechanics only — `ask <npc> quest/task` (★★★★) and
`give <item> <npc>` (★★★★). No hidden nouns, no magic words, no branches.
This is deliberately the safe pattern; the smuggler choice-quest (9215) stays
parked until linear quests are proven solid in-game.

---

## STEP 4A — PLAYER POV WALKTHROUGH (mandatory)

```
Step start: accept the dispatch
  Player thinks: "A toller on the road — let me see if he has work."
  Player types:  "ask garrick quest"   (then "give"/auto-receive dispatch)
  Discovers via: Universal MUD SOP — players try `ask <npc> quest` on every
                 NPC. Garrick's root hint also names "a delivery."

Step waybill: hand the dispatch to the ferry agent at Hartcharn
  Player thinks: "Garrick told me to take this dispatch to Severin Pell, the
                  ferry agent up the road at Hartcharn, to start it north."
  Player types:  "give dispatch to pell"   (or "give dispatch to severin")
  Discovers via: Garrick's grant text + the dispatch item description both
                 name Severin Pell at Hartcharn. `ask pell quest` also tells
                 the player to hand it over. Universal give-what-you-were-
                 given intuition.

Step manifest: carry the counter-signed manifest to the tithe-clerk
  Player thinks: "Pell counter-signed it and gave me a manifest for the
                  tithe-clerk, Verrold, at Kingsbarrow Vale."
  Player types:  "give manifest to verrold"
  Discovers via: Pell's hand-off text + the manifest item description name
                 Verrold at Kingsbarrow Vale. `ask verrold quest` confirms.

Step end: deliver the sealed packet to the East Gate guard
  Player thinks: "Verrold stamped it into the ledger and gave me the sealed
                  packet for the East Gate guard at New Plymouth."
  Player types:  "give packet to guard"
  Discovers via: Verrold's hand-off text + the packet item description name
                 the East Gate guard at New Plymouth. `ask guard quest`
                 confirms.
```

**Thousand-mudder test:** every step is `ask <npc> quest` or
`give <named-item> <named-npc>`, and the next NPC + location is named in the
prior NPC's text AND on the carried item's description. Estimated pass rate
>950/1000. No step relies on out-of-band knowledge.

---

## STEP CHAIN

```
Step 1: "start" — granted by dialogue node on Garrick (9170)
  Trigger: ask garrick quest/task → node `quest_start`
           grantsQuest: "60-start", givesItem: 40071 (sealed road-dispatch)
  Token: 60-start
  Description: "Toller Garrick has entrusted you with a sealed dispatch bound
    for New Plymouth. Carry it up the road to Severin Pell, the ferry agent
    at Hartcharn, who handles the official post relay north."
  Hint: "Take the sealed dispatch north to Hartcharn and give it to Severin
    Pell, the ferry agent."

Step 2: "waybill" — granted by GIVING item 40071 to Severin Pell (9182)
  Trigger: give dispatch to pell → quest engine item_give trigger (mob 9182,
           item 40071): consume_item 40071, give_item 40072, grant 60-waybill
  Token: 60-waybill
  Description: "Severin Pell counter-signed the dispatch and gave you a sealed
    manifest. Carry it onward to Tithe-Clerk Verrold at Kingsbarrow Vale, who
    must enter it in the capital's ledger."
  Hint: "Continue north to Kingsbarrow Vale and give the manifest to
    Tithe-Clerk Verrold."

Step 3: "manifest" — granted by GIVING item 40072 to Tithe-Clerk Verrold (9198)
  Trigger: give manifest to verrold → quest engine item_give trigger (mob 9198,
           item 40072): consume_item 40072, give_item 40073, grant 60-manifest
  Token: 60-manifest
  Description: "Verrold stamped the manifest into the tithe ledger and sealed
    it into a capital packet. Deliver it to the East Gate guard at New
    Plymouth — the last leg of the road."
  Hint: "Carry the sealed packet north to the East Gate of New Plymouth and
    give it to the gate guard."

Step 4: "end" — granted by GIVING item 40073 to An East Gate Guard (9208)
  Trigger: give packet to guard → quest engine item_give trigger (mob 9208,
           item 40073): consume_item 40073, give_item 40074 (courier's
           road-token), grant 60-end. Rewards block fires on 60-end.
  Token: 60-end
  Description: "You delivered the dispatch the length of the road, from the
    northern toll to the gates of the capital."
```

**No pre-discovery / out-of-order triggers needed:** items 40072/40073 ONLY
come from the relay NPCs, so a player can never hold a later item without
having completed the prior leg. The chain is strictly gated by construction.

---

## ALTERNATIVE PATHS

```
Each leg — `ask <npc> <topic>` instead of `give`:
  Mechanism: each relay NPC has a quest_active dialogue node (questRequired
    the leg's prerequisite token, questExcluded the leg's grant token) that,
    when the player asks, instructs them to HAND OVER the carried item
    ("give it here"). It does NOT itself complete the handoff — the `give`
    path (quest engine item_give trigger) is the single completion mechanic.
    This keeps item consume/give atomic and avoids double-issue.

Give item to the WRONG relay NPC (skip-ahead):
  e.g. give dispatch (40071) to Verrold, or give manifest (40072) to the guard.
  Mechanism: no matching item_give trigger on that NPC → give.go falls through
    to the NPC's archetype default player_give handler, which declines and
    returns the item. Safe no-op. (Severin = noncombat_shopkeeper and Verrold
    = noncombat_passive both decline+return by archetype; confirm guard_captain
    returns — add a return_item player_give handler to 9208 if it does not.)

Return to Garrick mid-quest:
  Mechanism: Garrick `quest_active` node (questRequired 60-start, questExcluded
    60-end) reminds the player where the dispatch is bound. No re-grant.
```

---

## QUEST GATING DIAGRAM

```
[no quest] --ask garrick quest--> [60-start] (+item 40071 dispatch)
[60-start] --give 40071 to Pell 9182--> [60-waybill] (-40071, +40072 manifest)
[60-waybill] --give 40072 to Verrold 9198--> [60-manifest] (-40072, +40073 packet)
[60-manifest] --give 40073 to Guard 9208--> [60-end] (-40073, +40074 token, REWARDS)
```

Verify:
- Each step has exactly one completion mechanic (the named `give`, or the
  `ask garrick quest` grant for step 1). No dead steps.
- `questExcluded` on every grant node prevents re-trigger.
- `60-end` fires the rewards block.
- Strictly linear: token N requires token N-1; no token is reachable early.

---

## NEW QUEST ITEMS (item ids 40071–40074, all NON-component, flag nodrop if supported)

| ID | Name | Type | Role | Notes |
|----|------|------|------|-------|
| 40071 | Sealed Road-Dispatch | quest/object | Garrick→Pell | Desc names "Severin Pell, ferry agent, Hartcharn" |
| 40072 | Counter-Signed Manifest | quest/object | Pell→Verrold | Desc names "Tithe-Clerk Verrold, Kingsbarrow Vale" |
| 40073 | Sealed Capital Packet | quest/object | Verrold→Guard | Desc names "the East Gate guard, New Plymouth" |
| 40074 | Courier's Road-Token | object (keepsake) | reward | A stamped brass token; flavor/no stats (or tiny charisma flavor) |

**Critical:** none may have `is_component: true` (component bag bypasses
give/requiresItem). Each delivery item's `description` must name the NEXT NPC
and zone — that is the primary discoverability surface, so write it carefully.
Recommend `nodrop`/quest flag on 40071–40073 so they can't be lost mid-relay
(simplifies recovery — see below).

---

## FILES NEEDED

| Action | File | Purpose |
|--------|------|---------|
| CREATE | `quests/60-the_long_road.yaml` | Quest def: steps, rewards, item_give triggers for mobs 9182/9198/9208 |
| CREATE | `items/quest/40071-sealed_road_dispatch.yaml` | Leg-1 carry item |
| CREATE | `items/quest/40072-counter_signed_manifest.yaml` | Leg-2 carry item |
| CREATE | `items/quest/40073-sealed_capital_packet.yaml` | Leg-3 carry item |
| CREATE | `items/quest/40074-couriers_road_token.yaml` | Reward keepsake |
| CREATE | `dialogue/north_road_north/9170.yaml` | Garrick: quest_start (grants + givesItem 40071), quest_active, quest_done, lost_report, flavor |
| CREATE | `dialogue/hartcharn/9182.yaml` | Severin Pell: quest_active ("give it here"), post-handoff node, lost_report (re-give 40072), flavor — NOTE: Pell already has a `shop:`; dialogue file is separate and coexists |
| CREATE | `dialogue/kingsbarrow_vale/9198.yaml` | Verrold: quest_active, post-handoff node, lost_report (re-give 40073), flavor |
| CREATE | `dialogue/new_plymouth_outskirts/9208.yaml` | East Gate guard: quest_active, quest_done, flavor |
| MAYBE | `mobs/new_plymouth_outskirts/9208-an_east_gate_guard.yaml` | Add `player_give` behavior tree `return_item` handler IF guard_captain archetype does not already decline+return wrong items |

**Instance saves to clear before testing:** `mobs.instances/*`,
`rooms.instances/*` (standard pre-smoke wipe). Shops dir untouched.

**Lost-item recovery (per SOP — every NPC that hands out a physical item):**
- Garrick `lost_report` node: questRequired [60-start], questExcluded
  [60-waybill, 60-end] → givesItem 40071 (re-issue dispatch).
- Pell `lost_report` node: questRequired [60-waybill], questExcluded
  [60-manifest, 60-end] → givesItem 40072.
- Verrold `lost_report` node: questRequired [60-manifest], questExcluded
  [60-end] → givesItem 40073.
- If 40071–40073 are flagged `nodrop`, these nodes are belt-and-suspenders
  (items can't be lost) but harmless — keep them for robustness.

---

## GOTCHAS CHECKLIST

- [x] Every `grantsQuest` node has `"quest"` and `"task"` in triggers
      (Garrick `quest_start`).
- [x] No grantsQuest patterns used (tree nodes only) — N/A for patterns, but
      Garrick's pattern block will also include quest/task keywords for
      discovery parity.
- [x] **Voice:** all NPC `text` first person ("I", "my"). Hints in narrator
      voice ("Take the dispatch north…", "You could ask…"). No 3rd-person
      self-reference.
- [x] **Trigger discoverability:** every trigger word (give targets, ask
      topics) appears in prior NPC text and/or the carried item's description.
- [x] **POV walkthrough complete (4a):** all four steps filled, each
      `give`/`ask` sourced to named NPC text + item description.
- [x] **Trigger mechanic tier:** every step is ★★★★ (`ask quest` /
      `give item npc`). No hidden nouns, no room_interact, no magic words.
- [x] **Thousand-mudder:** >950/1000 estimated. Linear named hand-offs.
- [x] **Narrator never overreaches:** send_text/item descriptions describe
      only the physical packet and the named next recipient — no mind-reading
      of absent characters. Forward hints ("the tithe-clerk will enter it")
      are narrator guidance, allowed.
- [x] **`questRequired` over `requires`** everywhere (no `requires` memory).
- [x] **No `expiryPeriod`** — omitted (no urgency design intent). memory
      expiryPeriod: "".
- [x] **Item delivery has BOTH paths:** `give` (quest engine item_give
      trigger, primary, consumes+gives) AND `ask <npc> quest` dialogue
      discovery on each relay NPC pointing the player to `give`.
- [x] **give.go + consume_item:** EVERY item_give trigger (mobs 9182, 9198,
      9208) includes `consume_item: <itemId>` so give.go marks Handled and
      does NOT bounce the item via the archetype decline branch. This is
      mandatory — without it Pell (shopkeeper) and the guard would decline
      AND keep/return the item after the quest already accepted it.
- [x] **return_item on wrong-item paths:** relay NPCs decline+return wrong
      items via archetype default (shopkeeper/passive). Guard (guard_captain)
      to be confirmed — add player_give return_item handler if needed.
- [x] **Lost-item recovery:** lost_report nodes on Garrick, Pell, Verrold
      (above). Guard is terminal (no re-issue needed past it).
- [x] **NPC handoff via `givesItem`:** Garrick hands 40071 via dialogue
      `givesItem`. Relay outputs (40072/40073) given via quest engine
      `give_item` action in the item_give trigger (atomic with consume).
- [x] **`requiresItem` sanity:** the `give` path is item-gated by the trigger
      `item:` field; no dialogue requiresItem needed for completion (avoids
      the give-but-not-consumed dialogue pitfall).
- [x] **Room nouns:** carried items are given by NPCs, not picked up from
      rooms — no `get <noun>` discoverability needed.
- [x] **Mob groups:** all four NPCs are non-hostile (noncombat_* /
      guard_captain), not in hostile groups. Confirmed from mob YAMLs.
- [x] **Physical items used** (not narrative-only) — the relay IS the quest.
- [x] **Multi-zone:** all four NPCs confirmed spawned (9170@5375, 9182@5409,
      9198@5445, 9208@5471) with spawninfo in their rooms.
- [x] **`questExcluded` on completion:** guard end node + every grant node
      excludes its own token AND `60-end`.
- [x] **End-token exclusion:** Garrick `quest_start` questExcluded
      ["60-start", "60-end"]; relay nodes exclude their grant token + "60-end".
- [x] **Rewards filled:** see below.
- [x] **Line width:** all text wraps ≤80 chars when authored.
- [x] **No raw numbers** in player-facing text (gold reward is the mechanical
      exception, shown by the engine).
- [x] **No flags / no branching** — N/A (deliberately linear).
- [x] **Quest items not components** — 40071–40074 all `is_component: false`.

---

## REWARDS (on 60-end, fired by guard item_give trigger)

```
Gold: 150       # longest quest in-game (5 zones); sized for the trek, tune post-playtest
Item: Courier's Road-Token (item 40074) — given via give_item in end trigger
Skill: none
Player message: "The guard breaks the seal, scans the packet, and gives a
  short nod. 'Came the whole road, did it. And you with it.' He presses a
  stamped brass token into your hand. 'Couriers' token. The post-houses from
  here to the toll will know it — a bed and a meal on the road, and a name
  that travels ahead of you. The capital remembers who carries its word.'"
Room message: "The East Gate guard takes the sealed packet, checks its seal
  against the gate ledger, and waves the courier through."
```

(Reward gold/itemized via the proven quest-17 pattern: `give_item` + `grant`
+ `consume_item` in the end trigger; `gold` + `playermessage` + `roommessage`
in the rewards block. Reward YAML keys are NO-underscore: `itemid`,
`playermessage`, `roommessage`, `gold`.)

---

## VERIFICATION PLAN (in-game, harness or manual)

1. Wipe instance saves; restart server. Confirm clean boot (quests
   `loadedCount` up by 1, no flag-panic, ValidateZoneConsistency errors=0).
2. Teleport to 5375. `ask garrick quest` → confirm 60-start granted +
   "You receive a Sealed Road-Dispatch." `look dispatch` → names Pell/Hartcharn.
3. Teleport to 5409. `give dispatch to pell` → confirm 40071 consumed, 40072
   received, 60-waybill granted, npc_say flavor. `ask pell quest` BEFORE
   giving → confirm it instructs the hand-off (alt path).
4. Teleport to 5445. `give manifest to verrold` → confirm 40072 consumed,
   40073 received, 60-manifest granted.
5. Teleport to 5471. `give packet to guard` → confirm 40073 consumed, 60-end
   granted, 40074 received, 150 gold added, player+room messages fire.
6. **Skip-ahead test:** new char, `give dispatch to verrold` → confirm
   declined+returned, no token. `give manifest to guard` → declined+returned.
7. **Lost-item test:** after step 2, drop dispatch, `ask garrick lost` →
   confirm re-issue (or confirm nodrop prevents the drop entirely).
8. **Re-grant test:** completed char, `ask garrick quest` → confirm quest_done
   node (no re-offer).

---

> This is a planning document only — no game files have been written.
>
> Review and annotate, then run:
> `/new-quest 60-the_long_road.md`
> to generate all files.
