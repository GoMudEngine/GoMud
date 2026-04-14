# Behavior Tree Schema Reference

## Overview

Behavior trees are YAML files that define event-driven mob AI. They replace
JS scripts with declarative, composable logic. Trees evaluate on events
(`player_ask`, `player_give`, `mob_idle`, `mob_hurt`) **before** JS scripts
and dialogue, giving behavior trees first priority.

**Path formula:**
```
_datafiles/world/dogmud/behaviors/{zone_folder}/{mobId}-{ConvertForFilename(name)}.yaml
```

Behavior trees live in a top-level `behaviors/` directory parallel to `mobs/`,
NOT inside `mobs/` (the mob loader panics on unknown YAML in its tree).

**Naming convention:** `{mobId}-{convertedName}.yaml` where `convertedName`
follows `ConvertForFilename` rules (lowercase, keep a-z/0-9, drop
apostrophes, all other chars become underscore).

**Worked example:**
- Zone: `Startland`, Mob ID: `14`, Name: `"Barmaid Dal"`
- Path: `_datafiles/world/dogmud/behaviors/startland/14-barmaid_dal.yaml`

---

## Node Types

Every node in a behavior tree has a `type` field that determines how it
evaluates. Nodes return one of three statuses: **Success**, **Failure**,
or **Running**.

| Type | Description |
|------|-------------|
| `selector` | Tries children in order. Returns Success on first child success (OR gate). Returns Failure only if all children fail. |
| `sequence` | Runs children in order. Returns Failure on first child failure (AND gate). Returns Success only if all children succeed. |
| `condition` | Checks a condition. Returns Success if true, Failure if false. Uses `check` field. |
| `action` | Performs a side effect. Returns Success or Failure. Uses `do` field. |
| `decorator` | Wraps a single `child` node with modifying behavior. Uses `mod` field. |

---

## Event Types

Trees are evaluated in response to events. Any node can include an
`event: <type>` field to restrict evaluation to that event type only.
Nodes without an `event` field evaluate on all events.

| Event | Trigger | Context Available |
|-------|---------|-------------------|
| `player_ask` | Player asks the mob something | `Text` = the ask text |
| `player_give` | Player gives an item to the mob | `ItemId` = item given |
| `mob_idle` | Mob's idle tick (periodic) | No player context |
| `mob_hurt` | Mob takes damage in combat | Player context from attacker |

---

## Conditions

Condition nodes use `type: condition` with a `check` field specifying
which condition to evaluate. Parameters are sibling fields on the same
node.

### Keyword & Text Matching

| Condition | Params | Description |
|-----------|--------|-------------|
| `keyword_match` | `keywords` (list of strings) | Matches any keyword against event Text. Case-insensitive. |

### Player State

| Condition | Params | Description |
|-----------|--------|-------------|
| `player_has_quest` | `quest` (string token) | Player has the specified quest token. |
| `player_missing_quest` | `quest` (string token) | Player does NOT have the specified quest token. |
| `player_has_item` | `item_id` (int) | Player has the item in inventory. |
| `player_has_gold` | `amount` (int) | Player has at least N gold. |
| `player_has_flag` | `flag_key`, `flag_value` (strings) | Player's quest flag matches. |

### Mob State

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_in_combat` | none | Mob is currently in combat. |
| `mob_health_below` | `percent` (int) | Mob health is below N% of max. |
| `mob_at_home` | none | Mob is in its home room. |
| `state_equals` | `key`, `value` (strings) | Checks mob's BehaviorState map. |

### Environment

| Condition | Params | Description |
|-----------|--------|-------------|
| `time_of_day` | `period` ("day" or "night") | Checks in-game time of day. |
| `round_mod` | `n` (int) | Succeeds when current round % n == 0. |
| `random_chance` | `percent` (int) | Succeeds with N% probability. |
| `players_in_room` | none | At least one player is in the mob's room. |
| `item_matches` | `item_id` (int) | Event ItemId matches. For `player_give` events. |

---

## Actions

Action nodes use `type: action` with a `do` field specifying which action
to perform. Parameters are sibling fields on the same node.

### Communication

| Action | Params | Description |
|--------|--------|-------------|
| `respond` | `user_text` (string), `room_text` (optional), `hints` (optional) | Sends text to the triggering player. `room_text` is shown to others in the room. |
| `say` | `text` (string) | Mob speaks to the entire room. |
| `emote` | `text` (string) | Mob emotes to the room (no "says" prefix). |

### Quest & Flags

| Action | Params | Description |
|--------|--------|-------------|
| `grant_quest` | `quest` (string token) | Grants a quest token to the player. |
| `set_quest_flag` | `flag_key`, `flag_value` (strings) | Sets a quest flag on the player. |

### Item & Gold

| Action | Params | Description |
|--------|--------|-------------|
| `give_item` | `item_id` (int) | Gives an item to the triggering player. |
| `take_item` | `item_id` (int) | Removes an item from the player's inventory. |
| `give_gold` | `amount` (int) | Gives gold to the player. |
| `take_gold` | `amount` (int) | Takes gold from the player. |

### Movement & Combat

| Action | Params | Description |
|--------|--------|-------------|
| `move` | `direction` (string) | Mob moves in the specified direction. |
| `attack` | none | Mob attacks the triggering player. |
| `flee` | none | Mob flees from combat. |
| `cast` | `spell` (string) | Mob casts the named spell. |

### Spawning & Environment

| Action | Params | Description |
|--------|--------|-------------|
| `spawn_mob` | `mob_id` (int), `room_id` (int, optional) | Spawns a mob. Defaults to current room if `room_id` omitted. |
| `add_temp_exit` | `exit_name`, `room_id` (int), `title` (string), `expires` (int) | Adds a temporary exit to the current room. `expires` is rounds until removal. |

### State Management

| Action | Params | Description |
|--------|--------|-------------|
| `set_state` | `key`, `value` (strings) | Sets a key in the mob's BehaviorState map. Persists for the mob's lifetime. |
| `command` | `cmd` (string) | Executes an arbitrary mob command string. Escape hatch for anything not covered by dedicated actions. |

---

## Decorators

Decorator nodes use `type: decorator` with a `mod` field and a single
`child` node.

| Decorator | Params | Description |
|-----------|--------|-------------|
| `cooldown` | `rounds` (int) | Skips child execution if it last fired within N rounds. |
| `repeat` | `times` (int) | Runs the child node N times. |
| `invert` | none | Flips child result: Success becomes Failure, Failure becomes Success. |
| `random` | `percent` (int) | Runs child with N% probability. Otherwise returns Failure. |
| `delay` | `rounds` (int) | Waits N rounds before executing child. Returns Running until delay elapses. |

---

## Event Filtering

Any node in the tree can include an `event: <type>` field. When present,
the node (and its entire subtree) is **skipped** unless the current event
matches. This lets you organize a single tree file into event-specific
branches:

```yaml
tree:
  type: selector
  children:
    - type: sequence
      event: player_ask     # Only evaluates on ask events
      children: [...]
    - type: selector
      event: mob_idle        # Only evaluates on idle ticks
      children: [...]
    - type: sequence
      event: player_give     # Only evaluates on give events
      children: [...]
```

---

## Complete Example

```yaml
tree:
  type: selector
  children:
    # --- Player asks about quests ---
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
          user_text: "I could use some help around here."
          hints: "Ask about the trouble in the mines."

    # --- Player gives an item ---
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
          user_text: "Ah, you found it! Thank you kindly."
        - type: action
          do: grant_quest
          quest: "10-end"

    # --- Idle behavior ---
    - type: selector
      event: mob_idle
      children:
        - type: decorator
          mod: cooldown
          rounds: 10
          child:
            type: action
            do: emote
            text: scratches behind one ear thoughtfully.
        - type: sequence
          children:
            - type: condition
              check: random_chance
              percent: 20
            - type: action
              do: say
              text: Another quiet day...
```

---

## BehaviorState

Each mob instance maintains a `BehaviorState` map (string keys to string
values) that persists for the mob's lifetime. Use `set_state` actions and
`state_equals` conditions to build stateful behaviors like patrol routes,
conversation phases, or cooldown flags.

Example patrol pattern:
```yaml
- type: sequence
  event: mob_idle
  children:
    - type: condition
      check: state_equals
      key: patrol
      value: outbound
    - type: action
      do: move
      direction: north
    - type: action
      do: set_state
      key: patrol
      value: return
```
