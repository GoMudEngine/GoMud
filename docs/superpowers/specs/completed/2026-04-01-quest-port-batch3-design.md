# Quest Port Batch 3 — Item Fetch + Delivery Quests Design Spec

## Overview

Port Quests 3, 4, 5, 6, and 16 to the quest engine. These are item
fetch + delivery quests involving room pickups, container items, and
cross-zone delivery. JS scripts replaced by quest engine triggers.
Static container items converted to spawninfo respawn.

## Container → Spawninfo Conversions

Three rooms have quest items in static containers that don't respawn
once taken. Convert all to spawninfo:

| Room | Item | Container | Respawn |
|------|------|-----------|---------|
| 312 (Carved Niche) | 14 (bone totem) | shelves | 5 real minutes |
| 421 (Tollhouse) | 21 (crossing ledger) | nouns/floor | 5 real minutes |
| 4033 (Hidden Grove) | 40040 (forest herbs) | herbs | 5 real minutes |

Delete any stale room instance saves for these rooms.

## Quests

### Quest 3: The Scholar's Collection

**NPCs:** Basin Scholar (mob 79, room 117)

**Steps:** `[start, partial, end]` with flag tracking

**Flag declaration:**
```yaml
flags:
  - key: delivered
    values: [totem, sac]
    description: "Which item was delivered first"
```

**Flow:**
1. Player asks Scholar → `3-start`
2. Player obtains bone totem (room 312 shelves) and/or spore sac
   (drops from spore crawler mob 78)
3. First item delivered → grants `3-partial` + sets flag
   `3-delivered: totem` or `3-delivered: sac`
4. Second item delivered → checks flag confirms the OTHER item was
   already delivered, grants `3-end`

**Triggers:**
```yaml
# Totem delivered first (no prior delivery)
- event: item_give
  mob: 79
  item: 14
  conditions:
    has: [3-start]
    missing: [3-partial]
  actions:
    - grant: 3-partial
    - set_flag: {key: "3-delivered", value: "totem"}
    - npc_say:
        mob: 79
        lines:
          - {delay: 1, text: "A bone totem from the warren shelves.
              Remarkable preservation. But I still need a spore sac
              from the crawlers deeper in the tunnels."}

# Sac delivered first (no prior delivery)
- event: item_give
  mob: 79
  item: 40008
  conditions:
    has: [3-start]
    missing: [3-partial]
  actions:
    - grant: 3-partial
    - set_flag: {key: "3-delivered", value: "sac"}
    - npc_say:
        mob: 79
        lines:
          - {delay: 1, text: "A spore sac. The membrane structure
              is extraordinary. But I still need a bone totem from
              the carved niche in the warren."}

# Totem completes (sac was first)
- event: item_give
  mob: 79
  item: 14
  conditions:
    has: [3-partial]
    missing: [3-end]
    has_flag: {"3-delivered": "sac"}
  actions:
    - grant: 3-end
    - npc_say:
        mob: 79
        lines:
          - {delay: 1, text: "The totem. Now I have both specimens.
              This will keep me occupied for months."}

# Sac completes (totem was first)
- event: item_give
  mob: 79
  item: 40008
  conditions:
    has: [3-partial]
    missing: [3-end]
    has_flag: {"3-delivered": "totem"}
  actions:
    - grant: 3-end
    - npc_say:
        mob: 79
        lines:
          - {delay: 1, text: "The spore sac. Now I have both
              specimens. This will keep me occupied for months."}
```

**Dialogue changes:**
- Update Scholar dialogue (mob 79) to match collapsed steps
- Replace `grantsQuest` refs from old step names to new
- Ensure `ask scholar quest/task` works

**JS disabled:** `79-basin_scholar.js` → `.js.bak`

**Room 312:** Convert bone totem from static shelves container to
spawninfo. Delete instance save.

### Quest 4: The Warden's Report

**NPCs:** Road Warden Tessara (mob 83)

**Steps:** Keep `[start, investigate, evidence, end]` — all working

**Flow:**
1. Player asks Tessara → `4-start`
2. Player enters Bandit Hollow (room 408) → room script grants
   `4-investigate` (or quest engine room_enter trigger)
3. Player picks up bandit's purse (item 16, spawninfo) → room
   script grants `4-evidence` (or quest engine item_gain trigger)
4. Player gives purse to Tessara → `4-end`

**Triggers:**
```yaml
# Room enter — player reaches bandit camp
- event: room_enter
  room: 408
  conditions:
    has: [4-start]
    missing: [4-investigate]
  actions:
    - grant: 4-investigate

# Item pickup — player finds the purse
- event: item_gain
  item: 16
  conditions:
    has: [4-investigate]
    missing: [4-evidence]
  actions:
    - grant: 4-evidence

# Item delivery via give command
- event: item_give
  mob: 83
  item: 16
  conditions:
    has: [4-evidence]
    missing: [4-end]
  actions:
    - grant: 4-end
    - npc_say:
        mob: 83
        lines:
          - {delay: 1, text: "Marked coins. Proving the bandits are
              spending in Thornwall. Someone inside those walls is
              fencing for them."}
          - {delay: 3, text: "This is enough to act on. You have
              done the road a service."}
```

**Dialogue changes:** Verify SOP compliance. Update any dead step
references in Tessara's dialogue.

**JS disabled:** Room script `408.js` → `.js.bak`. Tessara's onGive
script (if any) → `.js.bak`.

**Room 408:** Already uses spawninfo ✅. Check instance save.

### Quest 5: The Innkeeper's Complaint

**NPCs:** Innkeeper Tolva (mob 84, room 423)

**Steps:** Collapse to `[start, end]`

**Flow:**
1. Player asks Tolva → `5-start`
2. Player goes to Tollhouse (room 421), picks up crossing ledger
   (item 21)
3. Player gives ledger to Tolva → `5-end`

**Triggers:**
```yaml
# Item delivery via give command
- event: item_give
  mob: 84
  item: 21
  conditions:
    has: [5-start]
    missing: [5-end]
  actions:
    - grant: 5-end
    - npc_say:
        mob: 84
        lines:
          - {delay: 1, text: "The crossing ledger. Let me see those
              numbers."}
          - {delay: 3, text: "Just as I suspected. The toll rates
              do not match what Harn is collecting. Someone is
              skimming."}
```

**Dialogue changes:**
- Tolva dialogue: update dead step refs (5-ledger, 5-evidence → 5-end)
- Ensure `requiresItem: 21` and `grantsQuest: "5-end"` on dialogue
  delivery node

**JS disabled:** Room script `421.js` → `.js.bak`. Tolva mob script
`84-innkeeper_tolva.js` → `.js.bak`.

**Room 421:** Add spawninfo for crossing ledger (item 21) with
5 minute respawn. Currently no spawn mechanism — the room script
was the only way the item appeared. Delete instance save.

### Quest 6: The Collector's Burden

**NPCs:** Toll Collector Harn (mob 86, room 421), Records Clerk
Pell (mob 99, Thornwall City)

**Steps:** Keep `[start, report, end]`

**Flow:**
1. Player asks Harn → `6-start` + receives maintenance report
   (item 31 via `givesItem`)
2. Player travels to Thornwall, gives report to Clerk Pell →
   `6-report`
3. Player returns to Harn → `6-end`

**Triggers:**
```yaml
# Report delivery to Clerk Pell
- event: item_give
  mob: 99
  item: 31
  conditions:
    has: [6-start]
    missing: [6-report]
  actions:
    - grant: 6-report
    - npc_say:
        mob: 99
        lines:
          - {delay: 1, text: "A maintenance report from the
              crossing. Let me file this properly."}
          - {delay: 3, text: "Done. Tell Harn the report is on
              record. The Council will review it in due course."}
```

**Dialogue changes:** Verify SOP. Harn's dialogue already uses
`givesItem: 31` for handoff — keep as-is. Pell's dialogue already
has `requiresItem: 31` — verify it works alongside the trigger.

**JS disabled:** Harn script `86-toll_collector_harn.js` → `.js.bak`.
Pell script (if exists) → `.js.bak`.

**No container issues.** Item 31 is given via dialogue, not picked up.

### Quest 16: The Herbalist's Shortage

**NPCs:** Delia (mob 259), The Forager (mob 262)

**Steps:** Keep `[start, forager, end]` — two paths

**Flow — Path A (Forager):**
1. Player asks Delia → `16-start`
2. Player finds Forager in forest → negotiates arrangement →
   `16-forager`
3. Player returns to Delia → `16-end`

**Flow — Path B (Herb bypass):**
1. Player finds herbs (item 40040) in Hidden Grove before or after
   talking to Delia
2. Player gives herbs to Delia → grants `16-start` (if missing) +
   `16-end` via chain

**Triggers:**
```yaml
# Forager negotiation complete — return to Delia
# (forager dialogue handles 16-forager grant via grantsQuest)

# Herb delivery — normal path (has quest)
- event: item_give
  mob: 259
  item: 40040
  conditions:
    has: [16-start]
    missing: [16-end]
  actions:
    - grant: 16-end
    - npc_say:
        mob: 259
        lines:
          - {delay: 1, text: "These are exactly what I needed.
              Fresh and potent. Where did you find them?"}
          - {delay: 3, text: "The grove. I should have looked
              there myself. Thank you."}

# Herb delivery — bypass (no quest yet)
- event: item_give
  mob: 259
  item: 40040
  conditions:
    missing: [16-start]
  actions:
    - grant: 16-start
    - npc_say:
        mob: 259
        lines:
          - {delay: 1, text: "Forest herbs! Where did you find
              these? I have been running low for weeks."}
          - {delay: 3, text: "You have no idea how much this
              helps. Thank you, truly."}

# Chain: bypass grants 16-start, auto-complete
- event: quest_granted
  quest_token: "16-start"
  conditions:
    missing: [16-end]
  actions:
    - grant: 16-end
```

**Dialogue changes:** Verify SOP. Ensure forager dialogue grants
`16-forager` correctly.

**JS disabled:** Delia script `259-delia.js` → `.js.bak`.

**Room 4033:** Convert herbs from static container to spawninfo.
Delete instance save.

**Missing rewards:** Add to quest YAML (suggest 12 gold to match
zone level).

## SOP Compliance Checklist

For each quest:
- [ ] `ask <npc> quest/task` works on all quest-granting nodes
- [ ] `questExcluded` prevents double-completion
- [ ] All `text` fields are NPC speech (first person, no narration)
- [ ] All `hints` fields are narrator voice (bracketed)
- [ ] Quest step `hint:` fields give explicit cardinal directions
- [ ] Trigger words discoverable from NPC text or hints
- [ ] Quest items NOT `is_component: true`
- [ ] Container quest items use spawninfo respawn
- [ ] `item_give` triggers handle `give <item> <npc>` elephant path
- [ ] Dialogue `requiresItem` handles `ask <npc> <topic>` path
- [ ] Both paths grant same token
- [ ] Text wraps at 78-80 chars

## Files Changed

| Action | File | Quest |
|--------|------|-------|
| MODIFY | `quests/3-the_scholars_collection.yaml` | Q3 |
| MODIFY | `quests/4-the_wardens_report.yaml` | Q4 |
| MODIFY | `quests/5-the_innkeepers_complaint.yaml` | Q5 |
| MODIFY | `quests/6-the_collectors_burden.yaml` | Q6 |
| MODIFY | `quests/16-the_herbalists_shortage.yaml` | Q16 |
| MODIFY | `dialogue/sanctum_basin/79.yaml` | Q3 |
| VERIFY | `dialogue/dustwalk_road/83.yaml` | Q4 |
| MODIFY | `dialogue/watchers_crossing/84.yaml` | Q5 |
| VERIFY | `dialogue/watchers_crossing/86.yaml` | Q6 |
| VERIFY | `dialogue/thornwall_city/99.yaml` | Q6 |
| VERIFY | `dialogue/ashwick/259.yaml` | Q16 |
| VERIFY | `dialogue/ashwick/262.yaml` | Q16 |
| MODIFY | `rooms/labyrinth_of_low_tunnels/312.yaml` | Q3 |
| MODIFY | `rooms/watchers_crossing/421.yaml` | Q5 |
| MODIFY | `rooms/ashwick/4033.yaml` | Q16 |
| DISABLE | `mobs/.../scripts/79-basin_scholar.js` | Q3 |
| DISABLE | `rooms/dustwalk_road/408.js` | Q4 |
| DISABLE | `rooms/watchers_crossing/421.js` | Q5 |
| DISABLE | `mobs/.../scripts/84-innkeeper_tolva.js` | Q5 |
| DISABLE | `mobs/.../scripts/86-toll_collector_harn.js` | Q6 |
| DISABLE | `mobs/.../scripts/259-delia.js` | Q16 |

## Testing

For each quest: start → obtain items → deliver → verify completion.

**Q3:** Deliver totem first, then sac. Verify flag tracks delivery
order. Then test reverse order with a fresh character.

**Q4:** Walk to bandit hollow → verify investigate step fires on
entry → pick up purse → verify evidence step → give to Tessara.

**Q5:** Get ledger from tollhouse → give to Tolva. Verify ledger
respawns after 5 minutes.

**Q6:** Get report from Harn (dialogue) → travel to Thornwall →
give to Clerk Pell → return to Harn.

**Q16 Path A:** Ask Delia → find forager → negotiate → return to
Delia. **Path B:** Find herbs first → give to Delia without prior
conversation → verify both start+end fire.

**Respawn test:** Take container items, wait 5 minutes, verify they
reappear.
