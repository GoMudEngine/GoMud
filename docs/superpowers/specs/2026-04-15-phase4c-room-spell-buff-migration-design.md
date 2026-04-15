# Phase 4c: Room, Spell & Buff Script Migration — Design Spec

## Goal

Migrate all remaining dogmud JS scripts to Go/YAML/behavior trees,
introduce room behavior trees as a first-class system, simplify the
death flow, and delete vestigial spells/buffs. After this phase, the
only JS files remaining in `_datafiles/world/dogmud/` should be zero.

## Scope

**In scope:**
- 1 mob JS (Sable portal vendor) → mob behavior tree
- 14 room JS scripts → room behavior trees + YAML changes
- 6 spell JS → delete 3 stubs/vestigial, move 3 to Go hooks
- 9 buff JS → delete 8 stubs/vestigial, move 1 to Go hook
- Death flow simplification (eliminate shadow realm detour)
- Room behavior tree engine (new system, parallel to mob trees)

**Out of scope (deferred to Phase 5):**
- `default` and `empty` world JS files
- JS/Go scripting bridge removal
- Validator changes for world folder structure

## Architecture

### Room Behavior Trees

Room behavior trees are a first-class system parallel to mob behavior
trees. They share the same YAML format, node types, conditions, actions,
and decorators. The key differences:

**File convention:**
`_datafiles/world/dogmud/behaviors/rooms/{zone}/{roomId}.yaml`

**State:** `RoomBehaviorState` — same struct as mob `BehaviorState`,
stored on the room instance. Persists across events within a server
session. Reset on `room_load`.

**Events:**

| Event          | Fired from          | Context fields              |
|----------------|---------------------|-----------------------------|
| `room_enter`   | go.go after arrival | UserId, RoomId              |
| `room_exit`    | go.go before leave  | UserId, RoomId, Direction   |
| `room_command`  | command dispatch    | UserId, RoomId, Command, Rest |
| `room_idle`    | room idle tick      | RoomId                      |
| `room_load`    | room first loaded   | RoomId                      |

**Command interception:** `room_command` events evaluate the room
behavior tree before the engine processes the command. After evaluation,
the hook checks `ctx.Intercepted`. Only actions that explicitly call
`do: intercept` set this flag. If set, the engine skips normal command
processing for that input.

The `intercept` action sets `ctx.Intercepted = true` on the
`EvalContext`. This is a boolean flag on the context struct, not a new
Result type — tree evaluation semantics remain binary
(Success/Failure). Any branch can observe a command without blocking
it; only branches that include `do: intercept` will prevent default
engine processing.

**Entry points:**
- `TryRoomBehavior(roomId int, event EventContext) bool`
- Returns true if the tree intercepted the event
- Parallel to `TryMobBehavior`

**Negative caching:** Same `noTree` pattern as mobs — room IDs without
behavior files get cached to avoid repeated `os.Stat`.

**Shared infrastructure:** Room trees reuse the existing condition
registry, action registry, decorator registry, and YAML parser. New
room-specific conditions and actions are registered in the same
registries with appropriate nil-checks for mob vs room context.

### New Engine Features

**Static delay on action nodes:**

Action nodes gain an optional `delay` field (float64, seconds) that
queues the action for execution after a fixed delay. This is distinct
from the perception-scaled reaction delay — it is a scripted timeline
value specified in the YAML.

```yaml
- type: action
  do: mob_say
  mob_id: 50
  text: "Be still for a moment."
  delay: 1.0
```

When `delay` is set, it takes precedence over the perception-scaled
delay. When `delay` is absent, the existing perception-scaled delay
applies for actions in the `delayedActions` set. When neither applies,
the action fires immediately.

**New conditions:**

| Condition          | Params              | Purpose                    |
|--------------------|---------------------|----------------------------|
| `command_matches`  | `commands: [list]`  | Check command string       |
| `command_rest_contains` | `keywords: [list]` | Check rest string for keywords |
| `mob_in_room`      | `mob_id: int`       | Check if mob template present |

**New actions:**

| Action            | Params                    | Purpose                   |
|-------------------|---------------------------|---------------------------|
| `mob_say`         | `mob_id`, `text`          | Make a room mob speak     |
| `mob_emote`       | `mob_id`, `text`          | Make a room mob emote     |
| `grant_mutation`  | (none)                    | Roll + give random mutation |
| `give_gold`       | `amount: int`             | Grant gold to player      |
| `send_user_text`  | `text: string`            | Send text to triggering player |
| `send_room_text`  | `text: string`            | Send text to entire room  |
| `intercept`       | (none)                    | Set ctx.Intercepted=true  |
| `remove_buff`     | `buff_id: int`            | Remove a buff from player |
| `move_player`     | `room_id: int`            | Teleport player to room   |

`mob_say` and `mob_emote` differ from `command_mob` in that they are
purpose-built for room trees — they find a mob by template ID in the
current room and issue a say/emote command. `command_mob` already
exists and could be used, but dedicated actions are clearer in YAML
and allow the `delay` field to pace scripted NPC dialogue sequences.

### Ceremony Room as Reference Example

The Academy Hall (room 113) ceremony demonstrates the full room
behavior tree pattern. This example should be documented in the
behavior tree `context.md` as a reference.

**Current JS:** Room-level state (`ceremonyUnderway`, `ceremonyTicks`),
18 timed priest commands, mutation grant, exit locking, command
interception during the rite, multi-player collision handling.

**Room behavior tree design:**

```yaml
tree:
  type: selector
  children:

    # Clean state on load
    - type: sequence
      event: room_load
      children:
        - type: action
          do: set_state
          key: ceremony_active
          value: "false"
        - type: action
          do: set_room_locked
          direction: north
          locked: "false"
        - type: action
          do: set_room_locked
          direction: south
          locked: "false"
        - type: action
          do: set_room_locked
          direction: east
          locked: "false"
        - type: action
          do: set_room_locked
          direction: west
          locked: "false"

    # Block movement during ceremony
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: state_equals
          key: ceremony_active
          value: "true"
        - type: condition
          check: command_matches
          commands: [north, south, east, west, n, s, e, w, go]
        - type: action
          do: mob_say
          mob_id: 50
          text: >
            The Rite is not yet complete. A moment more.
        - type: action
          do: intercept

    # Prevent picking up the mosaic
    - type: sequence
      event: room_command
      children:
        - type: condition
          check: command_matches
          commands: [get, take, grab]
        - type: condition
          check: command_rest_contains
          keywords: [map, mosaic]
        - type: action
          do: send_user_text
          text: >
            The mosaic is set into the floor. It is not
            going anywhere.
        - type: action
          do: intercept

    # Ceremony: full rite for first eligible player
    - type: sequence
      event: room_enter
      children:
        - type: condition
          check: player_missing_quest
          quest: "1-mutation"
        - type: condition
          check: state_equals
          key: ceremony_active
          value: "false"
        - type: condition
          check: mob_in_room
          mob_id: 50
        - type: action
          do: set_state
          key: ceremony_active
          value: "true"
        - type: action
          do: set_state
          key: ceremony_ticks
          value: "0"
        - type: action
          do: set_room_locked
          direction: north
          locked: "true"
        - type: action
          do: set_room_locked
          direction: south
          locked: "true"
        - type: action
          do: set_room_locked
          direction: east
          locked: "true"
        - type: action
          do: set_room_locked
          direction: west
          locked: "true"
        # 18 timed priest lines (abbreviated here)
        - type: action
          do: mob_say
          mob_id: 50
          text: "Be still for a moment."
          delay: 1.0
        - type: action
          do: mob_say
          mob_id: 50
          text: >
            Type <ansi fg="command">look</ansi> at any
            time to examine your surroundings.
          delay: 2.5
        # ... remaining lines with increasing delays ...
        - type: action
          do: mob_emote
          mob_id: 50
          text: >
            steps forward and places two fingers lightly
            on your forehead. The air around you hums very
            quietly.
          delay: 20.5
        - type: action
          do: mob_emote
          mob_id: 50
          text: "withdraws their hand. The hum fades."
          delay: 22.5
        # Immediate grants (no delay)
        - type: action
          do: grant_mutation
        - type: action
          do: give_gold
          amount: 10
        - type: action
          do: grant_quest
          quest: "1-start"
        - type: action
          do: grant_quest
          quest: "1-mutation"

    # Ceremony: abbreviated path for late arrivals
    - type: sequence
      event: room_enter
      children:
        - type: condition
          check: player_missing_quest
          quest: "1-mutation"
        - type: condition
          check: state_equals
          key: ceremony_active
          value: "true"
        - type: action
          do: set_room_locked
          direction: north
          locked: "true"
        - type: action
          do: set_room_locked
          direction: south
          locked: "true"
        - type: action
          do: set_room_locked
          direction: east
          locked: "true"
        - type: action
          do: set_room_locked
          direction: west
          locked: "true"
        - type: action
          do: grant_mutation
        - type: action
          do: give_gold
          amount: 10
        - type: action
          do: grant_quest
          quest: "1-start"
        - type: action
          do: grant_quest
          quest: "1-mutation"

    # Unlock exits after ceremony duration
    - type: sequence
      event: room_idle
      children:
        - type: condition
          check: state_equals
          key: ceremony_active
          value: "true"
        - type: action
          do: increment_state
          key: ceremony_ticks
        - type: condition
          check: state_greater_than
          key: ceremony_ticks
          value: 5
        - type: action
          do: set_state
          key: ceremony_active
          value: "false"
        - type: action
          do: set_room_locked
          direction: north
          locked: "false"
        - type: action
          do: set_room_locked
          direction: south
          locked: "false"
        - type: action
          do: set_room_locked
          direction: east
          locked: "false"
        - type: action
          do: set_room_locked
          direction: west
          locked: "false"
```

Key patterns demonstrated:
- Room state (`ceremony_active`, `ceremony_ticks`) for cross-event
  persistence
- `intercept` action to block movement and item pickup during ceremony
- Static `delay` field for scripted NPC dialogue timing
- `mob_say`/`mob_emote` to puppet the priest NPC from the room tree
- Abbreviated path for multi-player collision
- `room_idle` tick counter for timed exit unlock
- `room_load` for clean state on server restart

## Room Script Migrations

### Sanctum Basin Tutorial (9 rooms)

All tutorial rooms follow a pattern: NPC teaches a skill, room detects
player commands, NPC responds with flavor, quest step granted.

| Room | Name | Migration approach |
|------|------|--------------------|
| 102 | Basin Gate | `room_enter`: warden dialogue across 4 quest states, gate lock/unlock via `set_room_locked` |
| 106 | West Meadow | `room_command`: match `forage`/`track` with quest gates, grant quest + NPC flavor with static delay |
| 108 | Market Street | `room_command`: match `buy`/`equip` with quest gates, same pattern as 106 |
| 109 | The Forge | `room_command`: match `craft` + keyword `dagger`/`iron`, grant quest. No delay needed — detecting the attempt is sufficient |
| 111 | Alchemist | `room_command`: match `craft` + keyword `salve`/`healing`, same pattern as 109 |
| 113 | Academy Hall | Full ceremony tree (see reference example above) |
| 114 | Training Yard | `room_idle`: check `mob_in_room` inverted (dummy died), grant quest step |
| 116 | Observatory | `room_command`: match `cast`, grant quest step with NPC flavor |
| 120 | Boss Cave | `room_idle`: check `mob_in_room` inverted (boss died), grant quest step |

### Non-Tutorial Rooms (5 rooms)

| Room | Name | Migration approach |
|------|------|--------------------|
| 407 | Abandoned Campsite | `room_enter`: quest gate → `grant_quest("4-investigate")`. Trivial one-branch tree |
| 4023 | Maren's Cottage | `room_command`: match `push` + keyword `stone`, check room state, spawn letter item. Intercepts command |
| 1 | Startland | Delete JS. Add room noun `map`/`sign` with static ASCII map description in room YAML |
| 75 | Shadow Realm Startland | Delete JS. Add room noun `map`/`sign` with static map description in room YAML |
| -1 | Shadow Realm | Delete JS entirely (shadow realm eliminated by death flow change) |

## Death Flow Simplification

**Current flow:**
1. Penalties applied (stat decay, skill rust) — Go, unchanged
2. Pools set to 5% of max — Go
3. Player teleported to `DeathRecoveryRoom` (shadow realm room 75)
4. `death_recovery` buff ticks, gradually healing the player
5. Shadow realm idle spawns a portal exit
6. Player exits portal → buff removed → redirected to home room

**New flow:**
1. Penalties applied (stat decay, skill rust) — unchanged
2. Pools restored to full
3. Flavor text: death description
4. Player teleported directly to home room
5. Brief disorientation buff (optional — cosmetic cooldown)

**Changes to `internal/usercommands/suicide.go`:**
- Replace pool-to-5% logic with pool-to-full
- Replace `MoveToRoom(userId, DeathRecoveryRoom)` with
  `MoveToRoom(userId, homeRoomId)` where homeRoomId is resolved
  from the player's `home` setting
- Add flavor text for the transition
- Remove death_recovery buff application if present

**Files deleted:**
- `_datafiles/world/dogmud/rooms/shadow_realm/-1.js`
- `_datafiles/world/dogmud/rooms/shadow_realm/75.js`
- `_datafiles/world/dogmud/buffs/24-death_recovery.js`

**Note:** The shadow realm rooms themselves (YAML) can remain for now
as vestigial content. Permadeath still sends players to room -1
(line 159 of suicide.go). Only the JS scripts and the non-permadeath
detour are removed.

## Spell Migrations

| Spell | Action | Details |
|-------|--------|---------|
| chrysalis-aid.js | **Delete spell** | Vestigial — death system handles respawn. Add player migration to prune from spellbooks. Remove from any NPC spell lists |
| heal.js | **Delete JS** | Stub — all logic in Go engine already |
| identify.js | **Delete JS** | Stub — one-line comment, logic in Go |
| fold-anchor.js | **Go hook** | New function in `internal/hooks/`: stores `room.RoomId` in `MiscData("fold-anchor-room")`, sends flavor text. Called from spell resolution |
| fold-recall.js | **Go hook** | New function in `internal/hooks/`: reads anchor from MiscData, validates (exists, not current room, not recall-blocked), clears combat, teleports. Called from spell resolution |
| purge-affliction.js | **Go hook** | New function in `internal/hooks/`: iterates target's active buffs, removes any with poison flag. Branches flavor text for self vs other |

**Spell hook integration:** The three Go hooks replace the JS
`onMagic` handler. Check how existing Go-native spells wire their
resolution (grep for `onMagic` or spell resolution dispatch) and
follow the same pattern.

**chrysalis-aid migration:** Add a `MiscData` migration check (same
pattern as existing migrations like `migration-alchemy-potions-done`)
that removes `chrysalis-aid` from `user.Character.Spellbook` on first
login after the update.

## Buff Migrations

| Buff | Action | Details |
|------|--------|---------|
| 0-meditating.js | **Go hook** | Graceful logoff: displays countdown messages per trigger, removes player on natural expiry. Move to buff tick handler in Go |
| 1-illumination.js | **Delete JS** | Stub — start/end messages. Move messages to YAML `onstart_text`/`onend_text` fields if they exist, otherwise add to buff YAML as flavor fields |
| 2-stunned.js | **Delete JS** | Stub — same pattern |
| 3-blinded.js | **Delete JS** | Stub — same pattern |
| 9-hidden.js | **Delete JS** | Stub — same pattern |
| 24-death_recovery.js | **Delete JS** | Eliminated by death flow change |
| 48-clarity_tonic.js | **Delete JS** | Stub — one-line comment |
| 49-fire_resistance.js | **Delete JS** | Stub — one-line comment |
| 51-berserker_elixir.js | **Delete JS** | Stub — one-line comment |

**Buff text fields:** Check whether buff YAML already supports
`onstart_text` / `onend_text` or similar. If not, adding these fields
to the buff spec is a small engine change that handles the 4 flavor-
only buff scripts without any Go hook code.

## Mob Migration

**315-sable.js (Portal Vendor):**

Sable is a portal vendor NPC who opens rifts to instanced zones
(Arena, Planar Oasis) for a gold fee. The JS handles `onCommand_talk`
and `onAsk` events with complex instance creation, difficulty scaling,
temporary exits, and refund logic.

This migrates to a mob behavior tree using `player_ask` events. New
actions needed:
- `create_instance` — creates an instanced zone with parameters
  (zone template, difficulty, gold cost)
- `take_gold` already exists

Alternatively, if instance creation is too complex for a behavior tree
action (it involves zone templates, access control, exit wiring), a
dedicated Go hook for Sable may be simpler. Read the JS carefully
during implementation to decide.

## Summary of New Engine Work

### Room Behavior Tree System
- `RoomBehaviorState` struct (reuse `BehaviorState`)
- `TryRoomBehavior()` entry point
- `GetRoomBehaviorPath()` helper
- Room event hooks in `go.go`, command dispatch, room idle, room load
- `ctx.Intercepted` flag on `EvalContext`
- Negative caching for rooms (parallel to mob `noTree`)

### New Conditions (3)
- `command_matches` — check command string against list
- `command_rest_contains` — check rest string for keywords
- `mob_in_room` — check if mob template ID present in room

### New Actions (9)
- `mob_say` — make a room mob speak
- `mob_emote` — make a room mob emote
- `grant_mutation` — roll + give random mutation
- `give_gold` — grant gold to player
- `send_user_text` — send text to triggering player
- `send_room_text` — send text to entire room
- `intercept` — set ctx.Intercepted flag
- `remove_buff` — remove a buff from player
- `move_player` — teleport player to room

### Static Delay Field
- `delay` float64 on action nodes (seconds)
- Takes precedence over perception-scaled delay
- Used for scripted NPC dialogue sequences

### Spell Go Hooks (3)
- fold-anchor resolution
- fold-recall resolution
- purge-affliction resolution

### Buff Go Hook (1)
- meditating tick/expiry handler

### Death Flow Change
- Simplify suicide.go: skip shadow realm, go directly home
- Restore pools to full instead of 5%
- Add flavor text

### Buff Text Fields
- `onstart_text` / `onend_text` on buff YAML (if not already present)

### Player Migration
- Remove chrysalis-aid from spellbooks

## File Counts

| Category | Create | Modify | Delete |
|----------|--------|--------|--------|
| Behavior trees (room) | ~15 | 0 | 0 |
| Behavior trees (mob) | 1 | 0 | 0 |
| Room YAML | 0 | 2 | 0 |
| Go (behaviortree/) | 2-3 | 3-4 | 0 |
| Go (hooks/) | 3-4 | 1-2 | 0 |
| Go (usercommands/) | 0 | 1 | 0 |
| JS files | 0 | 0 | 30 |
| Buff YAML | 0 | 4-5 | 0 |
| Docs | 1 | 1 | 0 |
