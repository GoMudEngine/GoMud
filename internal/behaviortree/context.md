# behaviortree — Package Documentation

## Overview

The `behaviortree` package implements an event-driven AI engine for mob
behavior. Behavior trees are declarative YAML files that define how a mob
responds to game events. They are evaluated before JS scripts and dialogue,
giving them first-priority on all handled events.

When an event fires for a mob (player asks a question, gives an item, idle
tick, etc.), the engine:

1. Looks up the mob's compiled tree by template mob ID.
2. If no tree is cached, resolves the file path and loads it on demand.
3. If the file does not exist, records a **negative cache** entry to skip
   future file-system checks for that mob ID.
4. Creates an `EvalContext` and calls `tree.Evaluate(ctx)`.
5. Returns `true` if the tree returned `Success`.

---

## File Path Convention

```
_datafiles/world/dogmud/behaviors/{zone}/{mobId}-{convertedName}.yaml
```

- `zone` is the mob's zone folder name (sanitized with `ZoneNameSanitize`).
- `mobId` is the integer mob template ID.
- `convertedName` is the mob's display name passed through
  `ConvertForFilename` (lowercase, keep a-z/0-9, drop apostrophes, all
  other characters become underscores).

**Example:**
- Zone: `Startland`, Mob ID: `14`, Name: `Barmaid Dal`
- Path: `behaviors/startland/14-barmaid_dal.yaml`

Behavior trees live parallel to `mobs/`, **not inside it**. The mob loader
panics if it encounters unknown YAML keys in the mobs directory.

---

## YAML Format

Every file has a single top-level `tree:` key whose value is a node
definition. A node definition has a `type` field plus type-specific fields.
Any node may include an `event:` field to restrict evaluation to one event
type (see Event Types below).

**Node types:** `selector`, `sequence`, `condition`, `action`, `decorator`

**Condition nodes** use `check: <name>` and place parameters as siblings:

```yaml
type: condition
check: keyword_match
keywords: [quest, task, help]
```

**Action nodes** use `do: <name>` and place parameters as siblings:

```yaml
type: action
do: respond
user_text: "I could use your help."
hints: "Ask about the missing shipment."
```

**Decorator nodes** use `mod: <name>` and require a single `child:` block:

```yaml
type: decorator
mod: cooldown
rounds: 10
child:
  type: action
  do: emote
  text: adjusts her apron.
```

### Full Example Tree

```yaml
tree:
  type: selector
  children:

    # Player asks — quest offer
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [quest, task, help]
        - type: condition
          check: player_missing_quest
          quest: "10-start"
        - type: action
          do: respond
          user_text: "I need someone to clear those bandits out."
          hints: "Ask about the bandits."
        - type: action
          do: grant_quest
          quest: "10-start"

    # Player gives a quest item
    - type: sequence
      event: player_give
      children:
        - type: condition
          check: item_matches
          item_id: 30001
        - type: condition
          check: player_has_quest
          quest: "10-start"
        - type: action
          do: respond
          user_text: "This is exactly what I needed, thank you!"
        - type: action
          do: grant_quest
          quest: "10-end"

    # Idle emote with cooldown
    - type: decorator
      event: mob_idle
      mod: cooldown
      rounds: 15
      child:
        type: action
        do: emote
        text: wipes down the counter absently.

    # Combat — flee at low health
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 20
        - type: action
          do: flee
```

---

## Event Types

| Event | Trigger | Notable Context |
|-------|---------|-----------------|
| `player_ask` | Player sends a `say` or `ask` to the mob | `Text` = spoken text |
| `player_give` | Player gives an item to the mob | `ItemId` = item given |
| `mob_idle` | Mob's periodic idle tick fires | No player context |
| `mob_hurt` | Mob takes damage in combat | `UserId` = attacker |
| `mob_die` | Mob's health reaches zero | `UserId` = killing player |
| `mob_flee` | Mob successfully flees combat | No player context |
| `player_enter` | A player enters the mob's room | `UserId` = player |

Any node may include `event: <type>` to skip that branch when the event
does not match. Nodes without an `event` field evaluate on all events.

### Combat Phase transition events (chunk 0)

These events fire ONCE per state transition (not per round):

- `mob_engaging` — fires when Combat Phase transitions Idle → Engaging
  (e.g., on attack command or attack btree action).
- `mob_engaged` — fires when Engaging → Engaged completes (after the
  RoundsUntil weapon-wait countdown).
- `mob_disengaging` — fires when Engaged → Disengaging (flee initiated).
- `mob_combat_ended` — fires when any state transitions to Idle (combat
  ends for any reason: target died, flee succeeded, force-idle, etc.).

Tick events (`mob_combat_round`, `mob_idle`) fire per-round-while-in-state.
Transition events fire at the moment of state change.

Future chunks (1-5) will add transition events for their machines.

---

## Condition Reference

Condition nodes use `type: condition` with `check: <name>`.

### Keyword & Text Matching

| Condition | Params | Description |
|-----------|--------|-------------|
| `keyword_match` | `keywords` (list) | Matches any word in event Text. |

### Player State

| Condition | Params | Description |
|-----------|--------|-------------|
| `player_has_quest` | `quest` (string) | Player holds the quest token. |
| `player_missing_quest` | `quest` (string) | Player does not hold the token. |
| `player_has_item` | `item_id` (int) | Player has item in inventory. |
| `player_has_gold` | `amount` (int) | Player has at least N gold. |
| `player_has_flag` | `flag_key`, `flag_value` (strings) | Quest flag matches. |
| `player_has_spell` | `spell` (string) | Player knows the named spell. |
| `player_has_misc_data` | `key`, `value` (strings) | Misc data key equals value. |

### Mob State

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_in_combat` | none | Mob's Aggro field is non-nil. |
| `mob_health_below` | `percent` (int) | Health < N% of max HP. |
| `mob_at_home` | none | Mob is in its home room. |
| `mob_has_buff` | `buff_id` (int) | Mob currently has the buff. |
| `state_equals` | `key`, `value` (strings) | BehaviorState string equals. |
| `state_greater_than` | `key` (string), `value` (int) | BehaviorState int > value. |

### Environment

| Condition | Params | Description |
|-----------|--------|-------------|
| `time_of_day` | `period` ("day" or "night") | In-game time of day. |
| `round_mod` | `n` (int) | `round % n == 0`. |
| `random_chance` | `percent` (int) | N% probability. |
| `players_in_room` | none | At least one player in the room. |
| `item_matches` | `item_id` (int) | Event ItemId matches. `player_give` only. |
| `multiple_enemies` | none | More than one player + charmed mob in room. |

### Combat Assessment

| Condition | Params | Description |
|-----------|--------|-------------|
| `target_power_ratio_above` | `value` (float) | True when self_power / target_power > value. Target resolution: `Event.UserId` → `Aggro.MobInstanceId` → `Aggro.UserId`. Returns Failure if no target resolvable or value missing/zero. |
| `target_power_ratio_below` | `value` (float) | Mirror of `_above`: true when ratio < value. |

### Stealth & Visibility (chunk 1)

Post-chunk-1, hidden-state conditions read via the `Character.IsHidden()`
predicate, which consults the Awareness machine instead of the buff #9 flag.
Conditions now reflect the canonical awareness state, not the side-effect
buff.

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_is_hidden` | none | True when self.Awareness.IsHidden() (previously checked buff #9). |
| `target_is_hidden` | none | True when the resolved target.Awareness.IsHidden() (previously checked buff #9). |
| `target_has_gold` | `min` (int) | True when the resolved PLAYER target has at least N gold. Mob targets always return Failure. |

### Position & Grapple (chunks 4a + 4b)

Registered in `conditions_position.go`. All conditions read the new
Position FSM via the `Character.IsXxx()` predicate family — never the
legacy `CombatPosition` enum.

**Chunk 4a — per-state and rollup predicates (10):**

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_is_standing` | none | True when `self.IsStanding()` (Position FSM Standing). |
| `mob_is_prone` | none | True when `self.IsProne()` (face-down knockdown). |
| `mob_is_grappling` | none | True when `self.IsGrappling()` (any of the 11 grapple states). |
| `mob_in_mount` | none | True when `self.IsMount()`. |
| `mob_in_guard` | none | True when `self.IsGuard()`. |
| `mob_in_clinch` | none | True when `self.IsClinch()`. |
| `mob_in_top_dominant` | none | True when `self.IsTopDominant()` (Mount / SideControl / KneeOnBelly / NorthSouth / Crucifix / BackGround). |
| `target_is_standing` | none | True when aggro target `IsStanding()`. |
| `target_is_prone` | none | True when aggro target `IsProne()`. |
| `target_is_grappled` | none | True when aggro target `IsGrappling()`. |

The chunk-4a predicates intentionally cover the commonly-queried
positions only; the full 14-state predicate API is available on
Character (`IsSupine`, `IsBackStanding`, `IsSideControl`,
`IsKneeOnBelly`, `IsNorthSouth`, `IsCrucifix`, `IsBackGround`,
`IsHalfGuard`, `IsTurtle`) for future primitives if archetype YAMLs
need finer-grained checks.

**Chunk 4b — control-axis predicates (6):**

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_is_in_control` | none | True when `self.IsController()` — controller side of a grapple pair. Replaces the legacy `HasCondition(ConditionGrappleController)` check (sunset target S4). |
| `mob_is_being_controlled` | none | True when `self.IsBeingControlled()` — controlled side. |
| `mob_control_at_least` | `level` (string: "InControl", "LosingControl", "Neutral", "BecomingControlled", "Controlled") | True when self's `ControlLevel` ≥ the named level on the controller→controlled axis. Used by archetypes that want graduated reactions (e.g. "if I'm only `LosingControl`, try to scramble; if I'm `Controlled`, turtle"). |
| `mob_low_grapple_stamina` | none | True when `self.IsLowGrappleStamina()` — stamina fraction below `GrappleStaminaLowThreshold` (config, default 0.25). Drives the "I'm gassed" archetype reactions. |
| `target_is_in_control` | none | True when aggro target `IsController()`. |
| `target_is_being_controlled` | none | True when aggro target `IsBeingControlled()`. |

The control-axis primitives let mob archetypes react to the per-round
drift mechanics introduced in chunk 4b — for example, a wrestler
archetype can press the advantage when `target_is_being_controlled`,
or a brawler archetype can disengage when `mob_low_grapple_stamina`
fires.

---

## Action Reference

Action nodes use `type: action` with `do: <name>`. Actions marked **delayed**
are subject to perception-scaled reaction delays (see below).

### Communication — delayed

| Action | Params | Description |
|--------|--------|-------------|
| `respond` | `user_text` (string), `room_text` (optional), `hints` (optional) | Sends text to triggering player; `room_text` to others; `hints` shown as a hint line. |
| `say` | `text` (string) | Mob says text to the whole room. |
| `emote` | `text` (string) | Mob emotes (no "says" prefix). |

### Quest & Flags — instant

| Action | Params | Description |
|--------|--------|-------------|
| `grant_quest` | `quest` (string) | Grants quest token to player. |
| `grant_quest_to_user` | `quest` (string) | Alias for `grant_quest`. |
| `set_quest_flag` | `flag_key`, `flag_value` (strings) | Sets quest flag on player. |

### Item & Gold — instant

| Action | Params | Description |
|--------|--------|-------------|
| `give_item` | `item_id` (int) | Gives one item copy to player. |
| `give_item_multiple` | `item_id` (int), `count` (int, default 1) | Gives N copies to player. |
| `return_item` | none | Returns the event's item to the player (for `player_give` rejection). Uses `ctx.Event.ItemId`. |
| `take_item` | `item_id` (int) | Removes first matching item from player. |
| `give_gold` | `amount` (int) | Adds N gold to player. |
| `take_gold` | `amount` (int) | Subtracts N gold (floor 0). |

### Movement & Combat — delayed

| Action | Params | Description |
|--------|--------|-------------|
| `move` | `direction` (string) | Mob moves in direction. |
| `attack` | none | Mob attacks the triggering player; if none, picks random player in room. |
| `flee` | none | Mob flees combat. |
| `cast` | `spell` (string) | Mob casts the named spell. |

### Combat Targeting — instant

| Action | Params | Description |
|--------|--------|-------------|
| `target_weakest_mob_in_room` | `ratio_below` (float, default 1.0) | Scans `room.GetMobs()`, picks the mob with the lowest power ratio relative to self that the caller's `HatesMob` returns true for, sets it as Aggro. Skips self, dead, non-combatant, same-owner companions, and mobs the caller doesn't hate. Players are NOT scanned. Returns Success on a pick, Failure otherwise. |
| `target_random_player_in_room` | none | Picks a random player in the current room and sets them as Aggro. Returns Success on a pick, Failure if no players. |

### Skullduggery — varied delays (chunk 2.7)

| Action | Params | Description |
|--------|--------|-------------|
| `try_sneak` | none | Invoke `actions.Sneak` (delayed). Success when self enters or is already in the hidden state. |
| `try_steal` | none | Invoke `actions.Steal` against the resolved target (delayed). Target resolution uses Event.UserId or Aggro fallback. |
| `try_plant` | `item_tag` (string, e.g. "copper coin") | Invoke `actions.Plant` with the named item from backpack (delayed). Failure if item not found or not in backpack. |
| `try_shadow` | none | Invoke `actions.Shadow` against the resolved target (delayed). Requires self already hidden. |
| `try_defuse` | none | Invoke `actions.Defuse` on the first trap found in room (delayed). Scans containers then exits. |

### Boss & Companion Control — delayed

| Action | Params | Description |
|--------|--------|-------------|
| `add_buff` | `buff_id` (int) | Applies buff to the acting mob. |
| `command_mob` | `mob_id` (int), `cmd` (string) | Issues a command to the first matching mob in room. |

### Spawning & Environment — instant

| Action | Params | Description |
|--------|--------|-------------|
| `spawn_mob` | `mob_id` (int), `room_id` (int, optional) | Spawns a mob. Defaults to current room. |
| `summon_companion` | `mob_id` (int), `count` (int, default 1), `base_pool` (int, default 50) | Spawns mob(s) as charmed companions of the acting mob, scaled by charisma + manifestation skill. |
| `spawn_item_in_room` | `item_id` (int), `room_id` (int, optional), `chance` (int 1-100, default 100) | Places an item on the floor of a room. |
| `add_temp_exit` | `exit_name` (string), `room_id` (int), `title` (string), `expires` (string) | Adds a temporary exit to the current room. |

### State Management — instant

| Action | Params | Description |
|--------|--------|-------------|
| `set_state` | `key` (string), `value` (any) | Sets a BehaviorState key. |
| `increment_state` | `key` (string), `amount` (int, default 1) | Adds N to a numeric BehaviorState key. |
| `decrement_state` | `key` (string), `amount` (int, default 1) | Subtracts N from a numeric BehaviorState key. |
| `set_misc_data` | `key` (string), `value` (string) | Sets misc data on the triggering player. |
| `command` | `cmd` (string) | Issues an arbitrary command to the mob. Escape hatch. |
| `set_room_locked` | `direction` (string), `locked` ("true"/"false", default "true") | Locks or unlocks a named exit in the current room. |

---

## Decorator Reference

Decorator nodes use `type: decorator` with `mod: <name>` and a single
`child:` node.

| Decorator | Params | Description |
|-----------|--------|-------------|
| `cooldown` | `rounds` (int) | Skips child if it last fired within N rounds. Uses BehaviorState for tracking. |
| `repeat` | `times` (int) | Runs child N times. Stops early on Failure. |
| `invert` | none | Flips Success/Failure. Running passes through. |
| `random` | `percent` (int) | Runs child with N% probability. |
| `delay` | `rounds` (int) | Waits N rounds before executing child. Returns Running until elapsed. |

---

## Reaction Delay System

Actions marked **delayed** (see Action Reference) do not fire immediately —
they are scheduled after a perception-scaled delay. This makes high-Perception
mobs feel quicker and more dangerous.

### Formula

```
delay = MobBTreeReactionBase - (Perception / MobBTreeReactionPerceptionScale)
delay = clamp(delay, MobReactionDelayMin, MobReactionDelayMax)
```

### Config Knobs (Balance section of config.yaml)

| Key | Default | Description |
|-----|---------|-------------|
| `MobBTreeReactionBase` | 3.0 s | Base delay before Perception adjustment. |
| `MobBTreeReactionPerceptionScale` | 100 | Divides Perception value to get reduction. |
| `MobReactionDelayMin` | 0.25 s | Minimum possible delay. |
| `MobReactionDelayMax` | 4.0 s | Maximum possible delay. |

### Example Values

At default config, a mob with Perception 100 has:
`3.0 - (100 / 100) = 2.0 seconds`

A mob with Perception 200 has:
`3.0 - (200 / 100) = 1.0 second`

A mob with Perception 50 has:
`3.0 - (50 / 100) = 2.5 seconds`

### Instant vs. Delayed Action Table

| Action | Delayed? |
|--------|----------|
| `respond` | Yes |
| `say` | Yes |
| `emote` | Yes |
| `attack` | Yes |
| `flee` | Yes |
| `cast` | Yes |
| `move` | Yes |
| `add_buff` | Yes |
| `command_mob` | Yes |
| `grant_quest` | No |
| `grant_quest_to_user` | No |
| `set_quest_flag` | No |
| `give_item` | No |
| `give_item_multiple` | No |
| `return_item` | No |
| `take_item` | No |
| `give_gold` | No |
| `take_gold` | No |
| `spawn_mob` | No |
| `summon_companion` | No |
| `spawn_item_in_room` | No |
| `add_temp_exit` | No |
| `set_state` | No |
| `increment_state` | No |
| `decrement_state` | No |
| `set_misc_data` | No |
| `set_room_locked` | No |
| `command` | No |
| `target_weakest_mob_in_room` | No |
| `target_random_player_in_room` | No |
| `try_sneak` | Yes |
| `try_steal` | Yes |
| `try_plant` | Yes |
| `try_shadow` | Yes |
| `try_defuse` | Yes |

---

## BehaviorState Patterns

`BehaviorState` is a per-mob-instance key/value store. It persists for the
mob's lifetime and is reset on respawn. Keys are strings; values may be
strings or integers.

Use `set_state` / `state_equals` / `state_greater_than` for string or
numeric flags. Use `increment_state` / `decrement_state` for counters.

The `cooldown` and `delay` decorators automatically write internal state
keys derived from the node's YAML path. You do not need to manage these.

### Counter: phase tracking

```yaml
# On mob_hurt, advance to phase 2 after taking 3 hits
- type: sequence
  event: mob_hurt
  children:
    - type: condition
      check: state_greater_than
      key: hit_count
      value: 2
    - type: action
      do: say
      text: "Enough! Now you face my true power!"
    - type: action
      do: add_buff
      buff_id: 5
    - type: action
      do: set_state
      key: phase
      value: "two"
- type: sequence
  event: mob_hurt
  children:
    - type: condition
      check: state_equals
      key: phase
      value: ""          # empty = default state
    - type: action
      do: increment_state
      key: hit_count
```

### Flag: one-shot dialogue

```yaml
- type: sequence
  event: player_enter
  children:
    - type: condition
      check: state_equals
      key: greeted
      value: ""
    - type: action
      do: say
      text: "Welcome, adventurer. I've been expecting you."
    - type: action
      do: set_state
      key: greeted
      value: "true"
```

### Timer: use `cooldown` decorator

```yaml
- type: decorator
  event: mob_idle
  mod: cooldown
  rounds: 20
  child:
    type: action
    do: cast
    spell: fireball
```

---

## Negative Caching

On the first event for any mob, the engine checks whether a behavior tree
file exists on disk. If no file is found, the mob's ID is recorded in the
`noTree` map. Subsequent events for that mob skip the `os.Stat` call
entirely — this prevents filesystem overhead on every event for mobs that
have no behavior tree.

The negative cache is cleared when `LoadTree` successfully loads a tree for
the same mob ID, so adding a behavior file at runtime is picked up after the
first successful load without a full engine restart.

---

## Structural Node Types

| Type | Description |
|------|-------------|
| `selector` | Tries children in order. Returns Success on first child success (OR gate). Returns Failure only if all children fail. |
| `sequence` | Runs children in order. Returns Failure on first child failure (AND gate). Returns Success only if all children succeed. |
| `condition` | Checks a condition. Returns Success or Failure. Uses `check:` field. |
| `action` | Performs a side effect. Returns Success or Failure. Uses `do:` field. |
| `decorator` | Wraps one `child:` node with modifying behavior. Uses `mod:` field. |

The `event:` field may be placed on any node type. It wraps the compiled
node in an `EventFilterNode` that returns Failure immediately when the
current event type does not match, short-circuiting the entire subtree.

---

## Entry Points

- `TryMobBehavior(instanceId int, event EventContext) bool` — main entry
  point. Resolves tree, builds context, evaluates, returns true on Success.
- `TryRoomBehavior(roomId int, event EventContext) bool` — room entry point.
  For `room_command` events returns `ctx.Intercepted`; for all others returns
  `true` on Success.
- `GetEngine().EvaluateEvent(...)` — lower-level call used by hooks.
- `GetEngine().DrainQueue()` — called once per round tick to execute
  pending delayed actions.

---

## Room Behavior Tree System

Rooms can have their own behavior trees evaluated on room-level events. Room
trees share all the same node types, conditions, and actions as mob trees, but
they run in a room context rather than a mob context.

### File Path

```
_datafiles/world/dogmud/behaviors/rooms/{zone}/{roomId}.yaml
```

- `zone` is the room's zone folder name (sanitized with `ZoneNameSanitize`).
- `roomId` is the integer room ID (no name suffix needed — room IDs are unique).

**Example:**
- Zone: `sanctum_basin`, Room ID: `113`
- Path: `behaviors/rooms/sanctum_basin/113.yaml`

### Room Events

| Event | Trigger | Notable Context |
|-------|---------|-----------------|
| `room_enter` | A player enters the room | `UserId` = entering player |
| `room_exit` | A player leaves the room | `UserId` = leaving player |
| `room_command` | A player types a command in the room | `Command` = command name, `Rest` = arguments |
| `room_idle` | Room's idle tick fires (every server round) | No player context |
| `room_load` | Room first loads from disk | No player context |

### Command Interception

Room behavior trees can intercept commands typed by players in the room,
preventing the default handler from processing them. Use the `intercept` action
to suppress the normal command handling after the behavior tree responds.

`ctx.Intercepted` is set to `true` by the `intercept` action. `TryRoomBehavior`
returns `ctx.Intercepted` for `room_command` events so the command dispatcher
knows whether to proceed.

### RoomBehaviorState

Each room has its own `BehaviorState` (stored in `EnsureRoomBTreeState`). It
persists for the lifetime of the process — rooms do not respawn, so the state
survives indefinitely. This enables persistent room-level state like ceremony
phases, NPC dialogue counters, and flag tracking.

The state is passed into `EvalContext.MobState` so all existing `set_state`,
`state_equals`, `state_greater_than`, etc. conditions and actions work
identically in room trees.

### Negative Caching (rooms)

Room trees use the same negative-cache pattern as mob trees. If no file is
found on first lookup, the room ID is recorded in `noRoomTree`. Subsequent
events skip `os.Stat`. Adding a behavior file at runtime is picked up after
the next successful load.

---

## Static Delay on Action Nodes

Any action node may carry a `delay: <seconds>` field (float64). When present,
the action is scheduled for execution that many seconds in the future rather
than firing immediately. All pending delayed actions are drained once per round
by `GetEngine().DrainQueue()`.

This is distinct from the perception-scaled reaction delay system: static
delays let content authors script precise timing for scripted NPC sequences
(e.g., the Awakening Rite ceremony). Perception-scaled delays apply only to
the communication and combat actions when triggered by mob behavior trees.

```yaml
- type: action
  do: mob_say
  mob_id: 50
  text: Step forward. This will not hurt.
  delay: 2.5        # fires 2.5 seconds from now
```

---

## New Conditions

These conditions were added alongside the room behavior tree system:

| Condition | Params | Description |
|-----------|--------|-------------|
| `command_matches` | `commands` (list of strings) | Matches the `Command` field of a `room_command` event. Case-insensitive. |
| `command_rest_contains` | `keywords` (list of strings) | Matches any keyword against the `Rest` field of a `room_command` event. Case-insensitive. |
| `mob_in_room` | `mob_id` (int) | At least one mob with the given template ID is present in the room. |

---

## New Actions

These actions were added alongside the room behavior tree system:

### NPC Targeting (room context)

| Action | Params | Description |
|--------|--------|-------------|
| `mob_say` | `mob_id` (int), `text` (string) | Finds the first mob in the room with the given template ID and makes it say text. |
| `mob_emote` | `mob_id` (int), `text` (string) | Same as `mob_say` but uses `emote` instead of `say`. |

### Player Effects

| Action | Params | Description |
|--------|--------|-------------|
| `grant_mutation` | none | Rolls and grants a random mutation to the triggering player from the weighted acquisition pool. |
| `send_user_text` | `text` (string) | Sends raw text to the triggering player (no mob prefix). |
| `send_room_text` | `text` (string) | Sends raw text to all players in the room. |
| `remove_buff` | `buff_id` (int) | Removes the specified buff from the triggering player. |
| `move_player` | `room_id` (int) | Teleports the triggering player to the target room. |

### Command Control

| Action | Params | Description |
|--------|--------|-------------|
| `intercept` | none | Marks the event as intercepted, preventing the default command handler from processing it. Only meaningful on `room_command` events. |

### Instance Portals

| Action | Params | Description |
|--------|--------|-------------|
| `create_instance` | `zone_name` (string), `gold_amount` (int), `state_key` (string, default `"instance_entry_room_id"`) | Clones an instanced zone and stores the entry room ID in BehaviorState. Use `add_temp_exit` with no `room_id` param to consume it. |
| `open_instance_portal` | `zones` (map), `min_gold` (int, default 100), `exit_expires` (string, default `"30 real minutes"`) | Full portal-vendor flow: parses `"<zone> <gold>"` from ask text, validates, charges gold, creates instance, adds temp exit. Returns Failure if text doesn't match pattern; returns Success with mob dialogue on all other outcomes (success or error). |

The `zones` param is a YAML mapping of short names to template zone names:
```yaml
zones:
  arena: "Instance Arena"
  oasis: "Instance Planar Oasis"
```

### Updated: `add_temp_exit`

When `room_id` is 0 or absent, `add_temp_exit` reads the entry room ID from
`BehaviorState` using the `state_key` param (default `"instance_entry_room_id"`).
This allows `create_instance` and `add_temp_exit` to work as a two-step pair:

```yaml
- type: action
  do: create_instance
  zone_name: "Instance Arena"
  gold_amount: 200
- type: action
  do: add_temp_exit
  exit_name: arena-portal
  title: arena-portal rift
  expires: "30 real minutes"
  # room_id omitted — read from "instance_entry_room_id" state key
```

---

## Ceremony Room Example (Room 113 — Academy Hall)

Room 113 (`behaviors/rooms/sanctum_basin/113.yaml`) is the reference example
for the room behavior tree system. It demonstrates:

1. **Exit locking via `set_room_locked`** — all four cardinal exits are locked
   when a new player enters and the Awakening Rite begins.
2. **Timed NPC dialogue via static delays** — the Chrysalis Priest (mob 50)
   delivers a 32-second monologue using `mob_say` with `delay:` fields.
3. **Multi-player collision handling via room state** — a `ceremony_active`
   flag prevents re-triggering if a second player enters mid-ceremony; they
   receive the mutation silently without restarting the sequence.
4. **Idle tick counter for timed unlock** — `room_idle` increments
   `ceremony_ticks`; after 5 ticks the exits unlock and state resets.
5. **Command interception** — `room_command` with `command_matches` intercepts
   movement commands during the ceremony and blocks item pickup of the mosaic.
6. **Inline text display** — `look mosaic` is handled by a sequence of
   `send_user_text` actions rendering an ASCII art world map.

Key takeaway: room state (`ceremony_active`, `ceremony_ticks`) coordinates
between the `room_enter` (which starts the ceremony) and `room_idle` (which
ends it), demonstrating how to build timed multi-phase room events without any
JavaScript.

---

## EvalContext.SoftTarget (chunk 2.7 fix)

`SoftTarget state.ActorRef` is a non-combat target slot on `EvalContext`.
It exists to solve the chunk-2.7 class of bugs where thief-archetype
behavior trees used `target_random_player_in_room` to pick a target and
then the steal/plant/shadow primitives read the wrong actor because there
was no safe place to stash "player I want to pickpocket" that is separate
from "player I am currently fighting."

### Design contract

- `target_random_player_in_room` stashes its pick in `ctx.SoftTarget`
  and does **not** call `SetAggro` or `TransitionToEngaging`. The mob's
  Combat Phase stays `Idle`; only the evaluation-local `SoftTarget` is set.
- Skullduggery conditions and actions (`target_is_hidden`, `target_has_gold`,
  `try_steal`, `try_plant`, `try_shadow`) read `resolveSkullduggeryTarget`,
  whose priority chain is:
  1. `ctx.SoftTarget` — non-combat pick from this evaluation round
  2. `ctx.Event.UserId` — player who triggered the btree event
  3. `CombatPhase` current target — attacker in combat

`SoftTarget` always wins if set, so `target_random_player_in_room → try_steal`
sequences work correctly even when the mob is simultaneously in combat with
a different player.

### What it prevents

Before this fix, `target_random_player_in_room` called `SetAggro`, which
set `CombatPhase` to `Engaging` for the picked player. This caused:
1. A non-combat thief mob unexpectedly entering combat with a bystander.
2. The steal action targeting the combat target (potentially a different
   player) instead of the intended pickpocket target.

The `SoftTarget` slot severs the non-combat targeting path from the
combat state machine entirely.

---

## Panic-Flee Pattern (chunk 2.6)

A shared `mob_hurt + mob_health_below:N → flee` branch is the FIRST
child of the top-level selector in five core archetypes:
`generic_fighter`, `predator`, `leader`, `lookout`, `tank_taunter`.
Threshold defaults to 25% HP. Emergency flee outranks any combat
action because it's the first matching branch in the selector
evaluation order.

Mobs that need a different threshold (e.g., Chrysalis Phantom at
20%, Edrin's heal-at-50% sequence) author a per-boss archetype that
overrides the default.

---

## New Archetypes (chunk 2.6)

Added in the legacy tactics-engine sunset migration:

- **`defensive_caster`** — Caster pattern with self-preservation:
  panic-flee at HP<30, panic-buff (chrysalis-cocoon when
  Chrysalis Shell buff 52 is missing), AoE on multiple targets
  (conviction-barrage), single-target spike (conviction-spike).
  Used by goblin_shaman (219), tunnel_shaman (74),
  bandit_caster (285), elemental_queen (321). Absorbed the
  legacy `defensive_caster` and `caster_backline` tactic presets.
- **`boss_edrin`** — Old Edrin's fragile-caster rotation with
  fold-recall at HP<30, panic-flee at HP<25, heal at HP<50,
  opening conviction-ward (shield spell — no buff gate),
  mind-spike on casters, hemorrhagic-burst on multi,
  pyretic-surge single-target.
- **`boss_sylara`** — Windwarden Sylara's heal-at-30 + panic
  chrysalis-cocoon (buff 52) + conviction-ward opener
  (shield spell — no buff gate) + bash interrupt.
- **`boss_rhett`** — Geomancer Rhett's defense-only opener
  (conviction-armor when buff 38 missing) + panic-flee.
- **`boss_soren`** — Soren's leader-archetype combat plus a
  call_for_help at HP<30 branch.
- **`boss_chrysalis_phantom`** — Tight panic-flee (HP<20) +
  target_casting → trip interrupt.

Spec: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`
