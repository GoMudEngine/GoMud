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
- `GetEngine().EvaluateEvent(...)` — lower-level call used by hooks.
- `GetEngine().DrainQueue()` — called once per round tick to execute
  pending delayed actions.
