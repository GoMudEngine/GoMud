# Go-Native Quest Engine Design Spec

## Overview

Replace the JavaScript-based quest scripting system with a Go-native quest
engine. Quest definitions expand from simple step lists into complete
self-contained YAML files that describe triggers, conditions, actions, and
NPC dialogue. The quest engine evaluates triggers synchronously (state
changes are immediate), with multiple layers of safety guards, configurable
verbose logging, and comprehensive test infrastructure.

**Goals (in priority order):**
1. Safety — don't crash the MUD or cause performance issues
2. Feature parity — handle everything the JS scripts do today
3. Testing — data integrity tests, unit tests, walkthrough tests, runtime logging
4. Debuggability — configurable verbose logging, per-player debug mode

## Current System Problems

- Quest logic is split across three places: quest YAML (steps/rewards),
  dialogue YAML (gating/triggers), and JS scripts (custom logic). Debugging
  requires cross-referencing all three.
- JS scripts have no type safety — token typos fail silently.
- The `give.go` item transfer happens BEFORE `onGive` fires — no rollback
  if the script fails, items can be lost.
- `requires` + `expiryPeriod` on dialogue memory can silently brick quests.
- Event-driven quest grants are async — you can't check a token you just
  granted in the same interaction, which is a subtle authoring footgun.
- No automated testing — quest bugs are only found by players.
- 90% of hard-to-solve bugs are quest-related.

## Architecture

### New Package: `internal/questengine/`

Core files:
- `engine.go` — trigger registration, event routing, synchronous evaluation
- `types.go` — QuestDef, Trigger, Condition, Action structs
- `actions.go` — action execution (grant, give_item, npc_say, etc.)
- `conditions.go` — condition evaluation (has, missing, in_room, etc.)
- `guards.go` — safety guards (depth limit, duplicate detection, visit tracking)
- `loader.go` — YAML loading and validation
- `logging.go` — configurable quest logging
- `engine_test.go` — unit tests
- `walkthrough_test.go` — per-quest integration tests
- `validation_test.go` — data integrity tests

### Integration Points

The quest engine exposes one primary function:

```go
func Notify(eventType string, details EventDetails)
```

Called from existing Go code at these hook points:
- `OnSkillUse()` in `progression.go` — covers cast, craft, forage, combat, salvage
- `give.go` — item delivery to NPC (BEFORE transfer, not after)
- `go.go` — room entry
- Combat death handler — mob death
- `ask.go` / `talk.go` — dialogue topic
- `ItemOwnership_CheckItemQuests.go` — item gain

### Event Flow

```
Player action (give item, enter room, use skill, etc.)
  │
  ├─ Go code calls questengine.Notify(eventType, details)
  │
  ├─ Quest engine finds matching triggers:
  │   └─ Filter by event type → filter by specific fields (mob, room, item)
  │     └─ Evaluate conditions (has/missing quest tokens, room check, etc.)
  │
  ├─ For each matched trigger, execute actions IN ORDER:
  │   ├─ grant → immediately update player quest state
  │   ├─ consume_item → remove from player inventory
  │   ├─ give_item → add to player inventory
  │   ├─ npc_say → queue NPC dialogue lines
  │   └─ etc.
  │
  ├─ After each `grant` action:
  │   └─ Check for `quest_granted` triggers that match the new token
  │     └─ Evaluate and execute those too (chained progression)
  │
  └─ Log everything at configured verbosity
```

## Quest YAML Format (Expanded)

```yaml
questid: 1
name: The Sanctum Trials
description: >-
  Complete the Awakening Rite and prove yourself in the
  Sanctum Basin's training grounds.
secret: false

steps:
  - id: start
    description: "Begin the Awakening Rite"
  - id: mutation
    description: "Survive the chrysalis transformation"
  - id: shopping_buy
    description: "Purchase equipment from a merchant"
  - id: shopping_equip
    description: "Equip your new gear"
  - id: alchemy_craft
    description: "Brew a potion at the alchemy bench"
  - id: crafting_craft
    description: "Forge an item at the smithy"
  - id: wilderness_forage
    description: "Forage for materials in the wild"
  - id: wilderness_track
    description: "Track a creature"
  - id: magic_cast
    description: "Cast a spell"
  - id: combat_defeat
    description: "Defeat the training dummy"
  - id: cave
    description: "Defeat the cave goblin"
  - id: warden
    description: "Report to the Basin Warden"
  - id: end
    description: "Leave the Sanctum Basin"

rewards:
  gold: 25
  playerMessage: >-
    You have completed the Sanctum Trials. The world awaits.

triggers:
  # ── AWAKENING RITE (Room 113) ─────────────────────────
  - event: room_enter
    room: 113
    conditions:
      missing: [1-start]
    actions:
      - lock_exits: {room: 113, player_scoped: true}
      - grant: 1-start
      - sequence:
          delay_between: 3
          lines:
            - {speaker: 50, text: "The chrysalis stirs..."}
            - {speaker: 50, text: "You feel the change begin."}
            - {text: "A mutation takes hold within you."}
          on_complete:
            - apply_buff: {buff: 1, source: "awakening"}
            - grant: 1-mutation
            - give_gold: 5
            - unlock_exits: {room: 113, player_scoped: true}

  # ── SHOPPING TUTORIAL (Room 108) ──────────────────────
  - event: command
    command: buy
    room: 108
    conditions:
      has: [1-mutation]
      missing: [1-shopping_buy]
    actions:
      - grant: 1-shopping_buy
      - npc_say:
          mob: 63
          lines:
            - "Good purchase! Now try equipping it."

  - event: command
    command: equip
    room: 108
    conditions:
      has: [1-shopping_buy]
      missing: [1-shopping_equip]
    actions:
      - grant: 1-shopping_equip

  # ── CRAFTING TUTORIALS ────────────────────────────────
  - event: skill_use
    skill: alchemy
    room: 111
    conditions:
      has: [1-shopping_equip]
      missing: [1-alchemy_craft]
    actions:
      - grant: 1-alchemy_craft
      - npc_say:
          mob: 53
          lines:
            - "Well done! The art of alchemy is yours to explore."

  - event: skill_use
    skill: blacksmithing
    room: 109
    conditions:
      has: [1-alchemy_craft]
      missing: [1-crafting_craft]
    actions:
      - grant: 1-crafting_craft

  # ── WILDERNESS TUTORIAL ───────────────────────────────
  - event: skill_use
    skill: search
    room: 106
    conditions:
      has: [1-crafting_craft]
      missing: [1-wilderness_forage]
    actions:
      - grant: 1-wilderness_forage
      - npc_say:
          mob: 54
          lines:
            - "Sharp eyes! Now try tracking a creature."

  - event: command
    command: track
    room: 106
    conditions:
      has: [1-wilderness_forage]
      missing: [1-wilderness_track]
    actions:
      - grant: 1-wilderness_track

  # ── MAGIC TUTORIAL ────────────────────────────────────
  - event: skill_use
    skill: spellcasting
    room: 116
    conditions:
      has: [1-wilderness_track]
      missing: [1-magic_cast]
    actions:
      - grant: 1-magic_cast

  # ── COMBAT TUTORIAL ───────────────────────────────────
  - event: mob_death
    mob: 65
    room: 114
    conditions:
      has: [1-magic_cast]
      missing: [1-combat_defeat]
    actions:
      - grant: 1-combat_defeat
      - npc_say:
          mob: 51
          lines:
            - "Well struck! You've proven yourself capable."

  # ── BOSS CAVE ────────────────────────────────────────
  - event: mob_death
    mob: 68
    room: 120
    conditions:
      has: [1-combat_defeat]
      missing: [1-cave]
    actions:
      - grant: 1-cave

  # ── WARDEN GATE (Room 102) ───────────────────────────
  - event: room_enter
    room: 102
    conditions:
      has: [1-cave]
      missing: [1-warden]
    actions:
      - grant: 1-warden
      - grant: 1-end
      - npc_say:
          mob: 56
          lines:
            - "You have passed every trial. The Basin gate opens."
            - {delay: 2, text: "Go forth. The world needs you."}
```

## Trigger Types

| Event | Fires When | Key Fields |
|-------|-----------|------------|
| `room_enter` | Player enters a room | `room` |
| `item_give` | Player gives item to NPC | `item`, `mob` |
| `skill_use` | `OnSkillUse` fires | `skill`, `room` (opt) |
| `mob_death` | Mob dies, player in room | `mob`, `room` (opt) |
| `command` | Player executes command | `command`, `room` (opt) |
| `item_gain` | Player receives/picks up item | `item` |
| `dialogue` | Player asks NPC about topic | `mob`, `topic` |
| `quest_granted` | Quest token just granted | `quest_token` |
| `room_interact` | Player interacts with noun | `room`, `noun`, `verb` |

## Condition Fields

| Field | Type | Meaning |
|-------|------|---------|
| `has` | `[]string` | Player must have ALL these quest tokens |
| `missing` | `[]string` | Player must NOT have ANY of these tokens |
| `in_room` | `int` | Player must be in this room (redundant with trigger room, but useful for `quest_granted` chains) |
| `has_item` | `int` | Player must have this item in backpack |
| `missing_item` | `int` | Player must NOT have this item |

## Action Types

| Action | Fields | What It Does |
|--------|--------|-------------|
| `grant` | `string` | Grant quest token (synchronous, immediate) |
| `consume_item` | `int` | Remove item from player backpack |
| `give_item` | `int` | Create and give item to player |
| `give_gold` | `int` | Give gold to player |
| `npc_say` | `mob`, `lines[]` | NPC speaks lines with optional `{delay, text}` |
| `send_text` | `string` | Send message to player |
| `room_text` | `string` | Send message to all in room |
| `spawn_mob` | `mob`, `room` | Spawn mob instance in room |
| `spawn_item` | `item`, `room` | Spawn item in room |
| `lock_exits` | `room`, `player_scoped` | Lock room exits (for this player only if scoped) |
| `unlock_exits` | `room`, `player_scoped` | Unlock room exits |
| `teach_spell` | `string` | Teach player a spell |
| `train_skill` | `skill`, `level` | Set skill to level |
| `apply_buff` | `buff`, `source` | Apply buff to player |
| `teleport` | `int` | Move player to room |
| `sequence` | `delay_between`, `lines[]`, `on_complete[]` | Multi-round timed sequence with actions on completion |

## Safety Guards (Defense in Depth)

### Layer 1: Depth Limit
Max 10 chained `grant` → `quest_granted` evaluations per original event.
If exceeded: log error with full chain trace, halt further evaluation,
do NOT crash. Configurable via `QuestChainDepthLimit` in Balance config.

### Layer 2: Duplicate Detection
Granting a token the player already has is a no-op. Logged at medium
verbosity as a warning. Prevents infinite loops where A grants B grants A.

### Layer 3: Visit Tracking
Track which trigger IDs fired during this evaluation chain. Same trigger
cannot fire twice in one chain. Prevents re-entrant loops even if
conditions would otherwise match.

### Layer 4: Step Validation
`grant` validates the token is a valid step for the quest using existing
`IsTokenAfter` logic. However, the engine should also support non-linear
quests (like Scholar's Collection where totem and sac can arrive in any
order). Config field `linear: false` on quest definition relaxes step
ordering to allow any step as long as it hasn't been granted yet.

### Layer 5: Panic Recovery
Each trigger evaluation is wrapped in `recover()`. A panic in one trigger
logs the full stack trace but does NOT crash the server or block other
triggers from evaluating.

### Layer 6: Performance Guard
If trigger evaluation for a single event exceeds 50ms (configurable),
log a warning with the trigger chain that caused it. This catches
pathological quest definitions before they impact game performance.

## Item Delivery Rework

**Critical change:** For `item_give` triggers, the quest engine intercepts
the give flow BEFORE the item transfer. The sequence becomes:

1. Player types `give bone-totem scholar`
2. `give.go` calls `questengine.Notify("item_give", details)` BEFORE
   transferring the item
3. Quest engine evaluates triggers:
   - If a trigger matches and has `consume_item`: engine tells `give.go`
     to consume the item (remove from player, do NOT give to mob)
   - If no trigger matches: `give.go` proceeds with normal transfer
     (item goes to mob, onGive JS fires for non-quest NPCs)
4. This eliminates the "item transferred but quest didn't advance" bug

The `Notify` function returns a `NotifyResult` struct:
```go
type NotifyResult struct {
    Handled     bool  // A quest trigger matched and handled this event
    ConsumeItem bool  // The item should be consumed (not transferred to mob)
}
```

## Logging System

### Log Levels

**Verbose** (default during development):
```
[QUEST] Player 5 entered room 461
[QUEST]   Evaluating trigger quest-3-totem (event: room_enter, room: 461)
[QUEST]     Condition has:[3-start] → PASS
[QUEST]     Condition missing:[3-totem] → PASS (player missing 3-totem)
[QUEST]     MATCHED → executing 2 actions
[QUEST]     Action grant:3-totem → OK
[QUEST]     Action npc_say:79 → queued 1 line
[QUEST]   Evaluating trigger quest-3-end (event: quest_granted, token: 3-totem)
[QUEST]     Condition has:[3-totem,3-sac] → FAIL (player missing 3-sac)
[QUEST]   No more triggers to evaluate
```

**Medium:**
```
[QUEST] Player 5: trigger quest-3-totem MATCHED → granted 3-totem
```

**Minimal:**
```
[QUEST] Player 5: granted 3-totem
```

### Per-Player Debug Mode
Admin command: `questdebug <player>` — sets that player to verbose logging
regardless of global setting. `questdebug <player> off` to disable.

### Config
```yaml
QuestLogLevel: verbose  # verbose, medium, minimal
QuestChainDepthLimit: 10
QuestPerformanceWarnMs: 50
```

## Testing Infrastructure

### Layer 1: Data Integrity Tests (build-time)

In `internal/questengine/validation_test.go`:

- Every quest token in trigger conditions (`has`, `missing`) references
  a valid step ID in a quest definition
- Every mob/room/item ID in triggers exists in data files
- Every `event: skill_use` references a registered skill tag
- Every `event: command` references a command that emits quest notifications
  (cross-referenced against a registry of notification-enabled commands)
- Every quest step has at least one trigger that can grant it (no
  unreachable steps)
- No circular quest chains (A rewards B which rewards A)
- Every `consume_item` references an item that appears in a trigger's
  `item` field or a condition's `has_item` (can't consume what you
  don't have)
- Quest files parse without error

### Layer 2: Engine Unit Tests

In `internal/questengine/engine_test.go`:

- Trigger matching: correct triggers fire for each event type
- Condition evaluation: has/missing/in_room/has_item all work correctly
- Action execution: each action type produces expected state changes
- Synchronous state: granting a token is immediately visible to
  subsequent condition checks in the same chain
- Guard rails: depth limit fires, duplicate detection works, visit
  tracking prevents re-entry, panic recovery catches panics
- Performance: large trigger sets evaluate within time budget
- NotifyResult: item_give returns correct Handled/ConsumeItem values

### Layer 3: Quest Walkthrough Tests

In `internal/questengine/walkthrough_test.go`:

One test per quest. Each test:
1. Sets up a mock player with empty quest state
2. Fires events in the order a player would experience them
3. Asserts each step is granted at the right time
4. Asserts items are consumed/given correctly
5. Asserts wrong-order attempts are rejected
6. Asserts completion fires rewards

Example:
```go
func TestWalkthrough_Quest3_ScholarsCollection(t *testing.T) {
    engine, player := setupTestEngine(t)

    // Start quest via dialogue
    engine.Notify("dialogue", EventDetails{
        UserId: player.Id, MobId: 79, Topic: "specimens"})
    assert.True(t, player.HasQuest("3-start"))

    // Deliver bone totem
    result := engine.Notify("item_give", EventDetails{
        UserId: player.Id, MobId: 79, ItemId: 14})
    assert.True(t, result.ConsumeItem)
    assert.True(t, player.HasQuest("3-totem"))
    assert.False(t, player.HasQuest("3-end")) // not yet

    // Deliver spore sac — should chain to completion
    result = engine.Notify("item_give", EventDetails{
        UserId: player.Id, MobId: 79, ItemId: 40008})
    assert.True(t, result.ConsumeItem)
    assert.True(t, player.HasQuest("3-sac"))
    assert.True(t, player.HasQuest("3-end")) // chained!
}
```

### Layer 4: Runtime Logging

Covered in the Logging System section above. Configurable verbosity,
per-player debug mode, performance warnings.

## Dialogue System Changes

### Keeps
- `questRequired` / `questExcluded` on TreeNode, Pattern, QuestGreeting —
  these control NPC conversation presentation based on quest state. This
  is display logic, not quest logic.
- `requires` / `unlocks` on TreeNode — conversation tree progression,
  not quest progression.

### Removes (moves to quest YAML)
- `grantsQuest` — quest token granting moves to quest triggers
- `requiresItem` — item consumption moves to quest trigger conditions/actions
- `givesItem` — item giving moves to quest trigger actions

### Migration
Fields are removed from dialogue engine processing but kept in the YAML
struct for backward compatibility during migration. A deprecation warning
is logged if a loaded dialogue file uses the removed fields.

## give.go Rework

The current flow:
```
1. Transfer item from player to mob
2. Fire onGive JS script
```

The new flow:
```
1. Call questengine.Notify("item_give", details)
2. If result.Handled:
   a. If result.ConsumeItem: remove item from player (don't give to mob)
   b. Skip onGive JS script (quest engine handled it)
3. If !result.Handled:
   a. Transfer item to mob (current behavior)
   b. Fire onGive JS script (backward compat for non-quest NPCs)
```

This is the critical safety improvement — the quest engine decides what
happens to the item BEFORE any transfer occurs. No more lost items.

## Migration Path

### Phase 1: Build Engine
- Create `internal/questengine/` package
- Implement trigger/condition/action evaluation
- Implement safety guards
- Implement logging
- Write unit tests

### Phase 2: Port Simple Quests
- Start with Quest 3 (Scholar's Collection) — simple multi-item delivery
- Then Quest 4 (Warden's Report) — fetch quest with room interactions
- Verify with walkthrough tests
- Delete corresponding JS scripts

### Phase 3: Port Tutorial
- Port Quest 1 (Sanctum Trials) — the complex tutorial
- Implement sequence actions for the ceremony
- Verify every step with walkthrough test
- Delete room JS scripts for Sanctum Basin

### Phase 4: Port Remaining Quests
- Port all 14 remaining quests
- Delete all quest-related JS scripts
- Remove JS quest hook fallbacks from Go code

### Phase 5: Documentation & Tooling
- Update CLAUDE.md with new quest engine docs
- Update `/sketch-quest` and `/new-quest` slash commands
- Create quest expert skill
- Update memory files

## Config Changes

New Balance fields:
```yaml
QuestLogLevel: verbose        # verbose, medium, minimal
QuestChainDepthLimit: 10      # max chained grant evaluations
QuestPerformanceWarnMs: 50    # warn if evaluation exceeds this
```

## Files Touched (estimated)

### New files:
- `internal/questengine/engine.go`
- `internal/questengine/types.go`
- `internal/questengine/actions.go`
- `internal/questengine/conditions.go`
- `internal/questengine/guards.go`
- `internal/questengine/loader.go`
- `internal/questengine/logging.go`
- `internal/questengine/engine_test.go`
- `internal/questengine/walkthrough_test.go`
- `internal/questengine/validation_test.go`
- Expanded quest YAML files (17 quests)

### Modified files:
- `internal/characters/progression.go` — add quest notification to OnSkillUse
- `internal/usercommands/give.go` — intercept item_give before transfer
- `internal/usercommands/go.go` — add room_enter notification
- `internal/usercommands/ask.go` / `talk.go` — add dialogue notification
- `internal/hooks/NewRound_DoCombat_helpers.go` — add mob_death notification
- `internal/hooks/ItemOwnership_CheckItemQuests.go` — add item_gain notification
- `internal/configs/config.balance.go` — add quest config fields
- `_datafiles/config.yaml` — add quest config values
- `internal/dialogue/types.go` — deprecate grantsQuest/requiresItem/givesItem
- `CLAUDE.md` — update quest system docs
- Various command files — add quest notifications for `buy`, `equip`, `track`, etc.

### Deleted files (after migration):
- ~26 quest-related JS scripts
- Old quest YAML files (replaced by expanded versions)
