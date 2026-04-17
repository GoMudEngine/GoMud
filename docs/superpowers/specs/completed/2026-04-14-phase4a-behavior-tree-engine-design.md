# Phase 4a: Behavior Tree Engine + Proof of Concept

**Date:** 2026-04-14
**Status:** Draft
**Phase:** JS Audit Phase 4a — Build behavior tree engine, migrate 3
representative mobs, validate architecture

## Goal

Replace the imperative JS scripting system for mobs with a declarative
YAML-driven behavior tree engine evaluated by Go. This eliminates JS
scripts, the JS↔Go scripting bridge (for mob hooks), and the separate
dialogue YAML system by unifying all mob behavior — AI, routines, quest
interactions, and dialogue — into one composable tree structure.

Phase 4a builds the engine and validates it on 3 representative mobs.
Phase 4b bulk-migrates the remaining mobs and dialogue files.

## Problem Statement

The current system has five pain points:

1. **Three systems for one job** — mob behavior is split across JS
   scripts, dialogue YAML, and Go hooks. Debugging requires checking
   all three.
2. **Tick-based sluggishness** — quest advancement and NPC reactions
   wait for the next round tick (4 seconds) instead of firing
   immediately.
3. **Timing fragility** — `mob.Command()` delays, idle conflicts, and
   pathing issues cause sequencing bugs.
4. **Undiscoverable triggers** — keyword-based dialogue that players
   can't find without trial and error.
5. **Syntax fragility** — JS code is easy to get wrong (YAML colons,
   wrong method names, missing bridge functions). 90% of zone
   debugging time is spent on mob/quest scripts.

## Design

### Architecture Overview

A new Go package `internal/behaviortree/` implements a tree evaluator.
Each mob can have a behavior tree defined in a YAML file discovered by
filename convention (same pattern as the former JS scripts). The engine
evaluates trees in two modes:

- **Event-driven (immediate)** — when a game event fires (player
  enters, player asks, player gives item, combat hit, etc.), the tree
  evaluates immediately in the same server frame. Quest interactions
  and dialogue feel instant.
- **Idle tick (round-based)** — once per game round, mobs get an idle
  tick for ambient behavior (patrol, emotes, scanning for threats).

### Timing Model

Actions have two timing categories:

**Instant** (no delay):
- Quest actions: grant_quest, set_quest_flag
- Commerce: give_gold, take_gold, give_item, take_item
- Quest advancement triggers

**Reaction-delayed** (perception-scaled):
- Dialogue: respond, say
- Combat: attack, flee, cast
- Movement: move
- Emotes

Reaction delay formula:
```
delay = base_reaction - (perception / scaling_factor)
clamped to [min, max]
```

Config in `_datafiles/config.yaml` under Balance:
```yaml
MobReactionBase: 3.0
MobReactionPerceptionScale: 100
MobReactionMin: 0.25
MobReactionMax: 3.5
```

A perception-100 mob reacts in ~2s. A perception-150 mob in ~1.5s.
A perception-50 mob in ~2.5s. Hyper-perceptive scouts react in 0.25s.

### Node Types

**Return values:** Each node returns `Success`, `Failure`, or `Running`
(for multi-round actions like moving to a waypoint).

**Core structural nodes:**

| Node | Behavior |
|------|----------|
| `selector` | Try children in order, return on first success |
| `sequence` | Run children in order, stop on first failure |
| `condition` | Check a predicate, return success/failure |
| `action` | Perform a game action |
| `decorator` | Modify a child node's behavior |

**Condition nodes:**

| Condition | Parameters | Purpose |
|-----------|-----------|---------|
| `keyword_match` | keywords: [] | Player ask/say text matches |
| `player_has_quest` | quest: "" | Player has quest token |
| `player_missing_quest` | quest: "" | Player lacks quest token |
| `player_has_item` | item_id: 0 | Player carries item |
| `player_has_gold` | amount: 0 | Player has N+ gold |
| `player_has_flag` | flag: "", value: "" | Quest flag check |
| `mob_in_combat` | | Mob is fighting |
| `mob_health_below` | percent: 0 | HP below threshold |
| `mob_at_home` | | Mob in home room |
| `time_of_day` | time: "day"/"night" | Day/night check |
| `round_mod` | mod: N | Round number % N == 0 |
| `random_chance` | percent: 0 | Percentage roll |
| `state_equals` | key: "", value: "" | BehaviorState check |
| `players_in_room` | min: 0 | Room has N+ players |

**Action nodes:**

| Action | Parameters | Purpose |
|--------|-----------|---------|
| `respond` | user_text, room_text, hints | Send text + hints to player |
| `say` | text | Mob speaks (room broadcast) |
| `emote` | text | Mob emotes |
| `grant_quest` | quest | Give quest token |
| `set_quest_flag` | key, value | Set quest flag |
| `give_item` | item_id | Give item to player |
| `take_item` | item_id | Take item from player |
| `give_gold` | amount | Give gold to player |
| `take_gold` | amount | Take gold from player |
| `move` | room_id / direction | Move mob |
| `attack` | target | Start combat |
| `flee` | | Flee from combat |
| `cast` | spell, target | Cast a spell |
| `spawn_mob` | mob_id | Spawn mob in room |
| `add_temp_exit` | exit config | Add temporary room exit |
| `set_state` | key, value | Set BehaviorState |
| `command` | cmd | Execute arbitrary mob command (escape hatch) |

**Decorator nodes:**

| Decorator | Parameters | Purpose |
|-----------|-----------|---------|
| `cooldown` | rounds: N | Skip child if fired within N rounds |
| `repeat` | times: N | Run child N times |
| `invert` | | Flip success/failure |
| `random` | percent: N | Run child with N% probability |
| `delay` | rounds: N | Wait N rounds (scripted pauses) |

### Event Types

Events injected into tree evaluation context:

| Event | Trigger | Context |
|-------|---------|---------|
| `player_enter` | Player enters mob's room | userId |
| `player_exit` | Player leaves mob's room | userId |
| `player_ask` | Player uses ask command | userId, text |
| `player_say` | Player says something | userId, text |
| `player_give` | Player gives item to mob | userId, itemId |
| `player_show` | Player shows item to mob | userId, itemId |
| `player_attack` | Player attacks mob | userId |
| `mob_hurt` | Mob takes damage | attackerId, damage |
| `mob_idle` | Round tick (no event) | |
| `mob_death` | Mob is killed | killerId |
| `combat_round` | Each combat round | |

Future events (Phase 4b):
| `mob_enter_room` | Another mob enters | mobInstanceId |
| `mob_emote_nearby` | Another mob emotes | mobInstanceId, text |
| `mob_say_nearby` | Another mob speaks | mobInstanceId, text |

Nodes filter by event using the `event` field. A branch with
`event: player_ask` only evaluates when that event fires. Branches
without an `event` field evaluate on every tick/event.

### YAML Format

Behavior files live alongside mob YAML:
```
_datafiles/world/dogmud/mobs/{zone}/behaviors/{mobid}-{name}.yaml
```

Discovery by filename convention — engine looks for
`behaviors/{mobid}-{ConvertForFilename(name)}.yaml` in the mob's zone
folder. No explicit reference in mob YAML needed.

Example — quest NPC:
```yaml
tree:
  type: selector
  children:

    # Quest dialogue — player asks about tasks
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [quest, task, tithe, audit]
        - type: condition
          check: player_missing_quest
          quest: "9-start"
        - type: action
          do: respond
          user_text: "The tithe records are a mess. I need someone
                      trustworthy to investigate."
          room_text: "{source} leans forward, speaking quietly
                      to {target}."
          hints:
            - "You could ask about the tithe details."
            - "You could ask what he needs you to do."
        - type: action
          do: grant_quest
          quest: "9-start"

    # Receive quest item via give
    - type: sequence
      event: player_give
      children:
        - type: condition
          check: player_has_quest
          quest: "9-investigate"
        - type: condition
          check: item_matches
          item_id: 21
        - type: action
          do: respond
          user_text: "Olen examines the ledger closely, his frown
                      deepening with each page."
        - type: action
          do: grant_quest
          quest: "9-end"

    # Idle behavior
    - type: sequence
      event: mob_idle
      children:
        - type: decorator
          mod: cooldown
          rounds: 10
          child:
            type: selector
            children:
              - type: action
                do: emote
                text: "flips through a ledger, muttering about numbers"
              - type: action
                do: emote
                text: "dips a quill in ink and scratches a note"
```

### Behavior State

Each mob instance carries a `BehaviorState map[string]any` that
persists for the mob's lifetime. Reset on respawn. Used for:

- Tracking patrol waypoint index
- Remembering last interaction round (for cooldowns)
- Boss phase tracking (engage → flee → hide → re-engage)
- Counting repeated interactions

Accessed via `state_equals` condition and `set_state` action.

### Token Substitution

Reuses the `textutil.SubstituteTokens` system from Phase 1:
- `{source}` — mob's tagged name
- `{target}` — triggering player's tagged name
- `{source_plain}` / `{target_plain}` — plain names

### Content Creation Support

- `context.md` in each mob zone folder with style guide, ordering
  conventions, syntax reference, and examples
- `/new-mob` slash command updated to generate behavior tree templates
- Schema doc: `docs/schemas/behavior.md`

### Discoverability SOP

Every `respond` node with quest-related content MUST include:
- `keywords` that contain "quest" and "task"
- `hints` that tell the player what to ask about next
- All trigger keywords must appear in a hint, NPC text, room
  description, or quest log

This is enforced by load-time validation — warn if a respond node
with `grant_quest` lacks "quest"/"task" in its keywords.

## Scope

**Phase 4a (this spec):**
- `internal/behaviortree/` package — engine, node types, YAML loading
- Hook integration — wire tree evaluation into mob event dispatch
- 3 proof-of-concept mobs: Barmaid Dal, Bandit Leader, Temple Priest Olen
- Migrate corresponding dialogue YAML into behavior trees
- Delete 3 JS scripts + 3 dialogue files
- `context.md` style guides
- Schema doc + `/new-mob` updates

**Phase 4b (future):**
- Bulk migrate remaining ~23 mob scripts
- Bulk migrate remaining ~27 dialogue files
- Mob-to-mob events (mob_enter_room, mob_emote_nearby)
- NPC-NPC interactions (barmaid + old men)
- Death recovery buff migration
- Bridge cleanup

**Phase 5 (future):**
- Room script behavior trees
- Goja removal

## Files Created/Modified

**New:**
- `internal/behaviortree/` — engine package (multiple files)
  - `engine.go` — tree evaluator, tick dispatch
  - `nodes.go` — node type definitions and registry
  - `conditions.go` — all condition node implementations
  - `actions.go` — all action node implementations
  - `decorators.go` — decorator implementations
  - `loader.go` — YAML parsing and tree construction
  - `state.go` — BehaviorState management
  - `engine_test.go` — unit tests
- `_datafiles/world/dogmud/mobs/thornwall_city/behaviors/117-barmaid_dal.yaml`
- `_datafiles/world/dogmud/mobs/marches_spur_road/behaviors/254-bandit_leader.yaml`
- `_datafiles/world/dogmud/mobs/thornwall_city/behaviors/95-temple_priest_olen.yaml`
- `docs/schemas/behavior.md`
- Zone `context.md` files with style guides

**Modified:**
- `internal/hooks/` — wire behavior tree evaluation into mob events
- `internal/mobs/mobs.go` — add BehaviorState to mob instance
- `.claude/commands/new-mob.md` — generate behavior tree templates
- Config: add reaction delay balance knobs

**Deleted:**
- `_datafiles/world/dogmud/mobs/thornwall_city/scripts/117-barmaid_dal.js`
- `_datafiles/world/dogmud/mobs/marches_spur_road/scripts/254-bandit_leader.js`
- `_datafiles/world/dogmud/mobs/thornwall_city/scripts/95-temple_priest_olen.js`
- Corresponding dialogue YAML files for the 3 mobs

## Testing

**Unit tests:**
- Tree evaluation: selector/sequence/condition/action/decorator
- Event filtering: branches only fire on matching events
- State management: set/get/reset
- Condition evaluation: all condition types
- Token substitution in action text

**Integration tests:**
- Mob with behavior tree responds to player_ask event
- Quest advancement fires immediately (no tick delay)
- Idle emotes fire on round tick with cooldown
- Reaction delay applied to combat actions

**AI smoke test:**
- `/test-mud local feature-tester phase4a-behavior.yaml`
- Test Olen's quest dialogue
- Test Bandit Leader's combat AI
- Test Dal's routine

## Future Extensions

- **Mob-to-mob events** (Phase 4b) — mobs react to other mobs
- **Memory** — persistent cross-respawn memory for recurring NPCs
- **Motivations** — goal-driven behavior (hungry → seek food → go to
  tavern → eat)
- **Economic behavior** — NPCs that buy/sell/trade based on needs
- **Shared/reusable trees** — named behavior templates that multiple
  mobs reference
- **Visual tree editor** — browser-based tool for designing trees
  (far future)
