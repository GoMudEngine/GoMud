# Quest Port Batch 4 (Final) — Complex Multi-Step + Dungeon Design Spec

## Overview

Port Quests 9, 10, and 14 to the quest engine. Convert Quest 17 from
a quest to pure lore discovery. Wire up the `room_interact` event
type so quest engine triggers can handle interactive room objects
(strongbox, shelves, etc.) without JS scripts.

## Go Code Change: Wire Up room_interact

The `room_interact` event type is defined in the quest engine but has
no call site. Wire it up so that noun interactions (get, push, open,
search, etc.) fire the event. This is a one-time investment that
enables all future quests to use interactive room objects via YAML
triggers.

**Where to add:** The noun/container interaction handling in the
`get` command (`internal/usercommands/get.go` or equivalent). When a
player interacts with a room noun (container, furniture, object), fire:

```go
questengine.GetEngine().Notify("room_interact", questengine.EventDetails{
    UserId: user.UserId,
    RoomId: room.RoomId,
    Noun:   nounName,
    Verb:   cmd,    // "get", "push", "open", "search", etc.
}, bridge, bridge)
```

Also add to `push`, `open`, `search` handlers if they exist as
separate commands, or to the general noun interaction path.

**Important:** The quest engine trigger should fire BEFORE the
default noun interaction (so it can intercept and provide custom
text). If the trigger's `Handled` result is true, skip the default
interaction. This lets triggers replace JS `onCommand` handlers.

## Quests

### Quest 9: The Temple's Tithe Audit

**NPCs:** Temple Priest Olen (mob 95, room 468)

**Steps:** Collapse to `[start, end]`

**Flow:**
1. Player asks Olen → `9-start`
2. Player goes to Records Office (room 476), picks up tithe ledger
   (item 29, spawninfo)
3. Player gives ledger to Olen → `9-end`

**Room 476:** Convert ledger from script-given to spawninfo item.
Add spawninfo entry for item 29, 5 min respawn. The room description
already mentions shelves — the ledger spawns on the floor or in a
shelf container if one exists.

**Triggers:**
```yaml
- event: item_give
  mob: 95
  item: 29
  conditions:
    has: [9-start]
    missing: [9-end]
  actions:
    - grant: 9-end
    - npc_say:
        mob: 95
        lines:
          - {delay: 1, text: "Let me see that."}
          - {delay: 3, text: "Both sides. The merchants are
              under-paying, and our own expense records are
              generous. This will not be pleasant, but it needed
              to come to light."}
```

**Dialogue changes:** Collapse dead step refs (9-investigate,
9-evidence → 9-start/9-end). Verify SOP.

**JS disabled:** Room script 476.js, mob script 95.

### Quest 10: The Drowning Post's Debt

**NPCs:** Tavern Keeper Marek (mob 96, room 472), Guard Captain
Velk (mob 94, room 473)

**Steps:** Collapse to `[start, report, end]`
- start: Marek gives protection notice (item 30) + quest
- report: Player delivers notice to Velk
- end: Player returns to Marek to confirm

**Flow:**
1. Player asks Marek → `10-start` + receives item 30 via `givesItem`
2. Player gives notice to Velk → `10-report`
3. Player returns to Marek → dialogue grants `10-end`

**Triggers:**
```yaml
# Notice delivery to Velk
- event: item_give
  mob: 94
  item: 30
  conditions:
    has: [10-start]
    missing: [10-report]
  actions:
    - grant: 10-report
    - npc_say:
        mob: 94
        lines:
          - {delay: 1, text: "A protection racket operating out of
              the back alleys. This is enough to act on."}
          - {delay: 3, text: "I will put a patrol on it. Tell the
              tavern keeper he can stop paying."}
```

**Dialogue changes:** Collapse dead step refs. Marek's completion
node checks `questRequired: ["10-report"]` and grants `10-end`.
This stays in dialogue — no trigger needed for end.

**JS disabled:** Velk's mob script handles Q10 AND Q14 item
delivery. Disable AFTER Q14 is also ported (they share the script).

### Quest 14: The Undertow

**NPCs:** Tavern Keeper Marek (mob 96), Guard Captain Velk (mob 94),
Torvan Cresk (mob 249)

**Steps:** Keep all 6: `[start, explore, confront, evidence, report, end]`

**Flow:**
1. Marek grants `14-start` + gives lantern (item 40038) via dialogue
2. Player descends cellar (room 485 — gated by quest check)
3. Player finds tally stick (item 40035) in room 492 stash →
   `item_gain` trigger grants `14-explore`
4. Player enters Operations Room (room 498) → `room_enter` trigger
   grants `14-confront`
5. Player defeats Torvan (mob 249) → gets strongbox key (item 40034)
   from mob drops
6. Player interacts with strongbox → `room_interact` trigger
   consumes key, gives bribe ledger (item 40036), grants `14-evidence`
7. Player gives ledger to Velk → `item_give` trigger grants
   `14-report`
8. Velk dialogue grants `14-end` (same conversation)

**Triggers:**
```yaml
# Tally stick pickup
- event: item_gain
  item: 40035
  conditions:
    has: [14-start]
    missing: [14-explore]
  actions:
    - grant: 14-explore
    - send_text: "A tally stick with hash marks and crude notations.
        Someone is counting shipments through these tunnels."

# Enter operations room
- event: room_enter
  room: 498
  conditions:
    has: [14-explore]
    missing: [14-confront]
  actions:
    - grant: 14-confront
    - send_text: "You step into a well-lit chamber. A man at the
        table looks up sharply. A heavy iron strongbox sits beneath
        his table, locked tight."

# Strongbox interaction (requires room_interact wiring)
- event: room_interact
  room: 498
  noun: strongbox
  verb: open
  conditions:
    has: [14-confront]
    missing: [14-evidence]
    has_item: 40034
  actions:
    - consume_item: 40034
    - give_item: 40036
    - grant: 14-evidence
    - send_text: "You fit the brass key into the padlock and turn.
        Inside the strongbox you find a bribe ledger filled with
        names, dates, and amounts paid to corrupt city officials."
    - room_text: "unlocks the strongbox and pulls out a leather
        journal."

# Strongbox without key — feedback
- event: room_interact
  room: 498
  noun: strongbox
  verb: open
  conditions:
    has: [14-confront]
    missing: [14-evidence]
    missing_item: 40034
  actions:
    - send_text: "The strongbox is locked with a heavy brass
        padlock. You need a key."

# Bribe ledger delivery to Velk
- event: item_give
  mob: 94
  item: 40036
  conditions:
    has: [14-evidence]
    missing: [14-report]
  actions:
    - grant: 14-report
    - npc_say:
        mob: 94
        lines:
          - {delay: 1, text: "Smuggling tunnels under the
              Craftsmen's Quarter? And a bribe ledger naming
              officials?"}
          - {delay: 3, text: "I am ordering arrests tonight. You
              have done this city a real service."}

# Velk completes quest in same conversation
- event: quest_granted
  quest_token: "14-report"
  conditions:
    missing: [14-end]
  actions:
    - grant: 14-end
```

**Cellar gate (room 485):** Replace JS gate script with a locked
exit that unlocks via quest trigger. Room 485's `down` exit should
have `lock: {difficulty: 255}`. Add a `quest_granted` trigger on
`14-start` that unlocks it. OR use a `room_enter` trigger that sends
the blocking message and prevents movement. The simplest approach:
keep the exit always open but add a `room_enter` trigger on the
destination room (486) that teleports the player back if they don't
have `14-start`.

Actually, the cleanest: just check in the `room_enter` trigger on
room 486 — if missing `14-start`, send text and teleport back to 485.

**Room 492:** Convert tally stick from static container to spawninfo.
Delete stale instance save.

**JS disabled:** Room scripts 485.js, 498.js. Velk's mob script
(after BOTH Q10 and Q14 triggers are in place).

### Quest 17: Convert to Lore Discovery (No Quest)

**Remove from quest system.** Keep room interactions as flavor.

**Option A:** Keep the room script (4023.js) for push stone / drawer
interactions. Items are lore objects, not quest items. Remove quest
17 YAML entirely.

**Option B:** Convert to `room_interact` triggers that just give
items with flavor text — no quest tokens. This removes the JS but
requires room_interact to be wired up first.

**Recommended:** Option B if room_interact is wired up. The triggers
would just give items + send text, no quest grants.

**Quest 17 YAML:** Either delete or mark `secret: true` with no
steps/rewards. If keeping the quest for discovery tracking, use
`secret: true` with minimal steps.

## Container → Spawninfo Conversions

| Room | Item | Container | Respawn |
|------|------|-----------|---------|
| 476 (Records Office) | 29 (tithe ledger) | shelf or floor | 5 min |
| 492 (Stash Room) | 40035 (tally stick) | crates | 5 min |

## SOP Compliance Checklist

Same as previous batches — all the standard checks for voice, hints,
triggers, quest/task keywords, questExcluded, line width.

## Files Changed

| Action | File | Quest |
|--------|------|-------|
| MODIFY | Go code: get.go or noun handlers | room_interact |
| MODIFY | `quests/9-*.yaml` | Q9 |
| MODIFY | `quests/10-*.yaml` | Q10 |
| MODIFY | `quests/14-*.yaml` | Q14 |
| DELETE or MODIFY | `quests/17-*.yaml` | Q17 |
| MODIFY | `dialogue/thornwall_city/95.yaml` | Q9 |
| MODIFY | `dialogue/thornwall_city/96.yaml` | Q10/Q14 |
| VERIFY | `dialogue/thornwall_city/94.yaml` | Q10/Q14 |
| VERIFY | `dialogue/thornwall_city/249.yaml` | Q14 |
| MODIFY | `rooms/thornwall_city/476.yaml` | Q9 spawninfo |
| MODIFY | `rooms/thornwall_city/492.yaml` | Q14 spawninfo |
| DISABLE | `rooms/thornwall_city/476.js` | Q9 |
| DISABLE | `rooms/thornwall_city/485.js` | Q14 |
| DISABLE | `rooms/thornwall_city/498.js` | Q14 |
| DISABLE | `rooms/ashwick/4023.js` | Q17 |
| DISABLE | `mobs/.../scripts/95-*.js` | Q9 |
| DISABLE | `mobs/.../scripts/94-*.js` | Q10/Q14 |

## Testing

**Q9:** Ask Olen → get ledger from room 476 → give to Olen → verify
rewards.

**Q10:** Ask Marek → receive notice → give to Velk → return to Marek.

**Q14:** Full dungeon crawl: Marek gives lantern → descend cellar →
navigate tunnels → find tally stick → enter ops room → defeat Torvan
→ get key → open strongbox → get ledger → give to Velk.

**Q14 gate:** Try to descend cellar without Q14 — should be blocked.

**Q17:** Visit cottage → push stone → find letter → open drawer →
find recipe. Verify no quest tokens granted.

**room_interact:** Test strongbox with key, without key, after
already opened. Test push stone. Verify triggers fire correctly.
