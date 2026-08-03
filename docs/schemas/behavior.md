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
| `mob_hurt` | Mob takes damage in combat | `UserId` = attacker |
| `mob_die` | Mob's health reaches zero | `UserId` = killing player |
| `mob_flee` | Mob successfully flees combat | No player context |
| `player_enter` | A player enters the mob's room | `UserId` = entering player |

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
| `player_has_spell` | `spell` (string) | Player knows the named spell. |
| `player_has_misc_data` | `key`, `value` (strings) | Misc data key equals value. |

### Mob State

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_in_combat` | none | Mob is currently in combat. |
| `mob_health_below` | `percent` (int) | Mob health is below N% of max. |
| `mob_at_home` | none | Mob is in its home room. |
| `mob_has_buff` | `buff_id` (int) | Mob currently has the buff active. |
| `state_equals` | `key`, `value` (strings) | BehaviorState string equals. |
| `state_greater_than` | `key` (string), `value` (int) | BehaviorState int > value. |

### Environment

| Condition | Params | Description |
|-----------|--------|-------------|
| `time_of_day` | `period` ("day" or "night") | Checks in-game time of day. |
| `round_mod` | `n` (int) | Succeeds when current round % n == 0. |
| `random_chance` | `percent` (int) | Succeeds with N% probability. |
| `players_in_room` | none | At least one player is in the mob's room. |
| `item_matches` | `item_id` (int) | Event ItemId matches. For `player_give`. |
| `multiple_enemies` | none | More than one player + charmed mob in room. |

### Forager (Stage 3.1)

| Condition | Params | Description |
|-----------|--------|-------------|
| `mob_can_safely_engage` | none | True if the mob's current Aggro target is in its `ForagerProfile.PreyWhitelist`, the target's stat-pool ≤ 60% of the mob's, and the mob's HP ≥ 75%. Used by foragers to gate opportunistic prey engagement. |
| `mob_inventory_at_threshold` | none | True if mob's carry weight / carry capacity ≥ `Balance.ForagerCarryThresholdPct` (default 0.75). Foragers head home when this fires. |
| `mob_hp_below_recall_threshold` | none | True if mob's HP ratio ≤ `Balance.ForagerHPRecallThresholdPct` (default 0.50). Foragers cast fold-recall when this fires. |

### Party (NPC Party System — Stage 1)

| Condition | Params | Description |
|-----------|--------|-------------|
| `party_member_below_pct` | `pool` ("hp"\|"sp"\|"cp"), `percent` (int) | True if any party member's pool is below the threshold percent. |
| `party_in_combat` | none | True if any party member is currently in combat (Aggro != nil). |
| `party_leader_in_combat` | none | True if specifically the leader is in combat. |
| `party_in_room` | none | True if all party members are in the same room. |
| `party_at_home` | none | True if all party members are in the party's HomeRoomId. Returns false if HomeRoomId is 0 (no home designated). |

All party conditions return false (not panic) when the caller isn't in a party, so behavior tree selectors fall through gracefully.

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
| `give_item` | `item_id` (int) | Gives one item copy to the triggering player. |
| `give_item_multiple` | `item_id` (int), `count` (int, default 1) | Gives N copies of an item. |
| `return_item` | none | Returns the event's item to the player. For `player_give` rejection — no params needed. Hands back the **real** item from the mob's inventory (located via `ctx.Event.ItemUUID`, falling back to the newest `ctx.Event.ItemId` match), preserving enchant tier and remaining uses. Fails if the mob is not holding it; never mints a copy. |
| `take_item` | `item_id` (int) | Removes first matching item from the player. |
| `give_gold` | `amount` (int) | Gives gold to the player. |
| `take_gold` | `amount` (int) | Takes gold from the player (floor 0). |

### Movement & Combat

Actions in this category are subject to perception-scaled reaction delays.
See the Reaction Delay System section below.

| Action | Params | Description |
|--------|--------|-------------|
| `move` | `direction` (string) | Mob moves in the specified direction. |
| `attack` | none | Mob attacks the triggering player; if no player context, picks a random player in the room. |
| `flee` | none | Mob flees from combat. |
| `cast` | `spell` (string) | Mob casts the named spell. |

### Boss & Companion Control

Also subject to reaction delays.

| Action | Params | Description |
|--------|--------|-------------|
| `add_buff` | `buff_id` (int) | Applies a buff to the acting mob. |
| `command_mob` | `mob_id` (int), `cmd` (string) | Issues a command string to the first mob in the room matching `mob_id`. |

### Spawning & Environment

| Action | Params | Description |
|--------|--------|-------------|
| `spawn_mob` | `mob_id` (int), `room_id` (int, optional) | Spawns a mob. Defaults to current room. |
| `summon_companion` | `mob_id` (int), `count` (int, default 1), `base_pool` (int, default 50) | Spawns mob(s) as charmed companions of the acting mob, stat pool scaled by charisma + manifestation skill. |
| `spawn_item_in_room` | `item_id` (int), `room_id` (int, optional), `chance` (int 1-100, default 100) | Places an item on the floor of a room. |
| `add_temp_exit` | `exit_name`, `room_id` (int), `title` (string), `expires` (string) | Adds a temporary exit to the current room. |
| `set_room_locked` | `direction` (string), `locked` ("true"/"false", default "true") | Locks or unlocks a named exit in the current room. |

### State Management

| Action | Params | Description |
|--------|--------|-------------|
| `set_state` | `key` (string), `value` (any) | Sets a BehaviorState key. Persists for mob's lifetime. |
| `increment_state` | `key` (string), `amount` (int, default 1) | Adds N to a numeric BehaviorState key. |
| `decrement_state` | `key` (string), `amount` (int, default 1) | Subtracts N from a numeric BehaviorState key. |
| `set_misc_data` | `key` (string), `value` (string) | Sets arbitrary misc data on the triggering player. |
| `command` | `cmd` (string) | Executes an arbitrary mob command string. Escape hatch. |

### Party (NPC Party System — Stage 1)

NPC parties coordinate group behavior across multiple mobs (movement, combat targeting, retreat). The actions below operate on the calling mob's party (looked up via `parties.GetByMobInstanceId(ctx.InstanceId)`). All return Failure when the caller isn't in a party.

| Action | Params | Description |
|--------|--------|-------------|
| `party_ensure_npc_party` | `leader_mob_id` (int), `home_room_id` (int) | Idempotent party creation/join. Place at the start of NPC party-member behavior trees so the party exists by the time later party_* actions run. If the designated leader mob is loaded, joins its party; otherwise the caller forms a new party as interim leader. |
| `party_call_help` | none | Marks the party's HelpRoomId to the caller's current room and fires `PartyHelpRequested` event. Used by lookouts on intruder spot AND by any member needing reinforcements. |
| `party_respond_to_help` | none | Navigates the caller one step toward `party.HelpRoomId`. Returns Success if already there. |
| `party_follow_leader` | none | Default movement: navigates one step toward the leader's room. Returns Success if already in same room. |
| `party_assist_target` | none | Sets the caller's combat aggro to match the leader's current target. Returns Failure if leader isn't in combat. |
| `party_flee_to_room` | `room_id` (int) | All party members navigate one step toward `room_id` (typically `party.HomeRoomId`). Triggered by leader-side btree on group-pressure threshold. |
| `party_at_home_stand` | none | If caller is at `party.HomeRoomId`, sets a `party_standing` BehaviorState flag to suppress flee branches in subsequent btree evaluation. Used at the camp/home for last-stand behavior. |
| `caravan_step` | none | Drives the Stage 2 caravan state machine. Reads the caller's `MobState["caravan_state"]`, dispatches based on state category (dwell / transit / route / fernway-pickup), advances state on the right environmental conditions (timer expired / arrival / all stops visited). Stage 3.1 added the fernway-pickup substates inside each transit leg, plus `caravan_load` MobState tracking that gates `RestockBuckets` calls at vendors. Used only on the caravan leader; follower btrees use `party_follow_leader` + `party_assist_target`. State persistence keys: `caravan_state`, `caravan_state_started_round`, `caravan_route_index`, `caravan_load`. |
| `forager_step` | none | Drives the Stage 3.1 forager state machine. Reads the caller's `MobState["forager_state"]`, dispatches per-state (resting / traveling-to-territory / foraging / traveling-to-dropoff / delivering / recalling), advances state on environmental conditions. HP-emergency short-circuit forces `recalling` from any state when HP drops below `Balance.ForagerHPRecallThresholdPct`. Stage 3.4 added rest-extension: forager stays home when carry > `ForagerRestCarryThreshold` (default 0.5). Used by the three foragers registered in `internal/forager.profiles` (Vella 371, Halix 372, Kessa 373). State persistence keys: `forager_state`, `forager_state_started_round`, `forager_forage_timer`, `forager_fatigue_timer`, `forager_visit_index`, `forager_wait_timer`. |
| `distribute_cargo_to_hostiles` | none | (Stage 3.4) Wagon-death handler. Walks all mobs in the wagon's current room, identifies hostiles (those that `HatesMob(wagon)`), distributes wagon `Character.Items` round-robin into their inventories until wagon is empty or all hostiles are at `CarryCapacity`. Items that don't fit drop as standard wagon-corpse loot via the engine's normal mob-death path. Returns Failure when no hostiles present (lets standard corpse-drop run unmodified). Used by the caravan wagon (mob 374). |

**Party events** (fired by party actions or by the `MobDeath_PackFlee` hook):

| Event | Fired by | Payload |
|-------|----------|---------|
| `PartyHelpRequested` | `party_call_help` action | `PartyId`, `CallerActorId`, `CallerIsPlayer`, `RallyRoomId` |
| `PartyDissolved` | `Party.Dissolve(reason)` (called by `MobDeath_PackFlee` when a party leader dies, or by explicit disband) | `PartyId`, `Reason` ("leader_died"\|"disbanded"\|"all_dead"), `MemberActorIds` |

**Leader death model:** Stage 1 uses dissolution ("cut the head off the snake") — when a leader dies, `PartyDissolved` fires for all members and the party is removed from the registry. Members revert to solo behavior via their individual btrees. Promotion-style succession is explicitly out of scope; future opt-in via a `party_succession_chain` field on the party if needed.

---

## Reaction Delay System

Communication and combat actions are executed after a perception-scaled
delay rather than immediately. This makes high-Perception mobs feel faster.

### Formula

```
delay = MobBTreeReactionBase - (Perception / MobBTreeReactionPerceptionScale)
delay = clamp(delay, MobReactionDelayMin, MobReactionDelayMax)
```

### Config Knobs (Balance section of config.yaml)

| Key | Default | Description |
|-----|---------|-------------|
| `MobBTreeReactionBase` | 3.0 s | Base delay before Perception adjustment. |
| `MobBTreeReactionPerceptionScale` | 100 | Divides Perception to get reduction. |
| `MobReactionDelayMin` | 0.25 s | Minimum possible delay. |
| `MobReactionDelayMax` | 4.0 s | Maximum possible delay. |

**Delayed actions:** `respond`, `say`, `emote`, `attack`, `flee`, `cast`,
`move`, `add_buff`, `command_mob`.

All other actions execute immediately (quest grants, item transfers, state
writes, spawn actions, etc.).

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

---

## Room Behavior Trees

Rooms support their own behavior trees evaluated on room-level events.

### File Path

```
_datafiles/world/dogmud/behaviors/rooms/{zone}/{roomId}.yaml
```

**Example:** Zone `sanctum_basin`, Room ID 113 →
`behaviors/rooms/sanctum_basin/113.yaml`

Room trees use the same node types, conditions, and actions as mob trees.
The `BehaviorState` for a room persists for the lifetime of the process.

### Room Events

| Event | Trigger | Context |
|-------|---------|---------|
| `room_enter` | Player enters the room | `UserId` = entering player |
| `room_exit` | Player leaves the room | `UserId` = leaving player |
| `room_command` | Player types a command in the room | `Command`, `Rest` |
| `room_idle` | Room idle tick (every round) | No player context |
| `room_load` | Room first loads from disk | No player context |

### Command Interception

Use the `intercept` action inside a `room_command` branch to prevent the
default command handler from running:

```yaml
- type: sequence
  event: room_command
  children:
    - type: condition
      check: command_matches
      commands: [north, south, east, west]
    - type: action
      do: send_user_text
      text: A barrier blocks the way.
    - type: action
      do: intercept
```

---

## Static Delay

Any action node may include a `delay: <seconds>` field (float64). The action
is scheduled for the future rather than firing immediately. Useful for timed
NPC dialogue sequences.

```yaml
- type: action
  do: mob_say
  mob_id: 50
  text: You will feel the change over time.
  delay: 14.5
```

---

## Additional Conditions

| Condition | Params | Description |
|-----------|--------|-------------|
| `command_matches` | `commands` (list) | Matches `Command` of a `room_command` event. Case-insensitive. |
| `command_rest_contains` | `keywords` (list) | Matches any keyword against `Rest` of a `room_command` event. Case-insensitive. |
| `mob_in_room` | `mob_id` (int) | At least one mob with the given template ID is in the room. |

---

## Additional Actions

### NPC Targeting

| Action | Params | Description |
|--------|--------|-------------|
| `mob_say` | `mob_id` (int), `text` (string) | Finds the first mob with the given template ID in the room and makes it say text. |
| `mob_emote` | `mob_id` (int), `text` (string) | Like `mob_say` but uses `emote`. |

### Player & Room Text

| Action | Params | Description |
|--------|--------|-------------|
| `send_user_text` | `text` (string) | Sends raw text to the triggering player (no mob prefix). |
| `send_room_text` | `text` (string) | Sends raw text to all players in the room. |
| `grant_mutation` | none | Rolls and grants a random mutation to the triggering player. |
| `remove_buff` | `buff_id` (int) | Removes a buff from the triggering player. |
| `move_player` | `room_id` (int) | Teleports the triggering player to a target room. |
| `intercept` | none | Prevents the default command handler from running. `room_command` only. |

### Instance Portals

| Action | Params | Description |
|--------|--------|-------------|
| `open_instance_portal` | `zones` (map), `min_gold` (int), `exit_expires` (string) | Full portal-vendor flow. Parses `"<zone> <gold>"` from ask text, validates, charges gold, clones instance, adds temp exit. Returns Failure if text doesn't match; otherwise Success with inline mob dialogue. |
| `create_instance` | `zone_name` (string), `gold_amount` (int), `state_key` (string) | Clones an instance and stores entry room ID in state. Follow with `add_temp_exit` (omit `room_id` to read from state). |

`open_instance_portal` `zones` param maps short names to template zone names:
```yaml
zones:
  arena: "Instance Arena"
  oasis: "Instance Planar Oasis"
```

---

## Instant vs Delayed Action Table (updated)

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
| `mob_say` | No (use `delay:` param for timing) |
| `mob_emote` | No (use `delay:` param for timing) |
| `grant_mutation` | No |
| `send_user_text` | No |
| `send_room_text` | No |
| `intercept` | No |
| `remove_buff` | No |
| `move_player` | No |
| `create_instance` | No |
| `open_instance_portal` | No |
