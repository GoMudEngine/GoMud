# Mob AI Framework — Design Spec

**Date**: 2026-04-07
**Status**: Approved

## Problem

Mobs only make decisions on the 4-second round tick. Players can
act 2-4 times in that window, making even "dangerous" mobs feel
sluggish. The current workaround is inflating mob stats, which
makes them hit hard but still feel dumb. JS scripting is unreliable
for combat AI (50ms timeout, synchronous execution, flaky state
management) and we want to avoid adding more of it.

Mobs need four tunable danger levers:
1. **Smarter reactions** — better tactical decisions
2. **Faster reactions** — sub-round response time
3. **Better stats/skills/spells** — already exists
4. **Better gear** — already exists

This spec covers levers 1 and 2: a pure-Go, event-driven tactical
AI framework with YAML-configurable behavior and per-mob reaction
speed.

## Solution

An event-driven reaction system where mobs respond to combat events
(player casting, taking damage, target fleeing, etc.) within a
configurable delay, rather than waiting for the next round tick.
Behavior is defined as priority-ordered tactic rules in mob YAML.
A discipline factor controls how reliably a mob follows its tactics.

## Architecture

### Event-Driven Reactions

The engine already has an event system. The framework adds a new
listener that fires on combat-relevant events and evaluates mob
tactics. When a trigger matches, the mob's reaction is queued with
a per-mob delay.

**Event sources** (things that cause mob AI to evaluate):
- `events.RoomAction` — player enters room, starts casting
- Damage dealt to mob — mob takes a hit
- Combat state changes — aggro set/cleared, target fled
- Mob's own action completes — for combo chains
- Round tick — fallback for periodic re-evaluation

The round tick still handles auto-attacks. Smart decisions happen
reactively between rounds.

### Reaction Timing

Each mob has a `reaction_delay` field (float64, seconds). When a
trigger matches and a tactic fires, the resulting command is queued
with this delay. A fast rogue reacts in 0.5s; a slow brute in 2.0s.

```yaml
reaction_delay: 0.5  # seconds before executing a reactive tactic
```

Default: 1.5 seconds (comparable to an average player). Valid range
0.25 to 4.0 seconds. Values outside this range are clamped.

A mob should not be able to queue multiple reactions within a short
window. A cooldown equal to `reaction_delay` prevents spamming —
after a reaction fires, the mob ignores triggers until the cooldown
expires.

### Tactical Discipline

Each mob has a `tactical_discipline` field (float64, 0.0 to 1.0).
When a tactic matches, the mob rolls against this value:
- Roll succeeds → execute the tactic
- Roll fails → skip (do nothing, fall through to default combat)

```yaml
tactical_discipline: 0.95  # almost always follows the playbook
```

Default: 0.5. The Phantom at 0.95 almost always executes hit-and-run.
A generic bandit at 0.4 sometimes does something smart, often just
swings. A training dummy at 0.0 never does anything tactical.

### Combat Memory

Mobs get a `CombatMemory` struct that persists across flee/re-engage
cycles:

```go
type CombatMemory struct {
    TargetUserId    int     // Who they were fighting
    TargetMobId     int     // Or which mob (for mob-vs-mob)
    LastSeenRoomId  int     // Where the target was last seen
    LastSeenRound   uint64  // When they last saw the target
    Grudge          bool    // Should they pursue?
}
```

Memory is set when combat starts and cleared after a configurable
timeout (`CombatMemoryDuration`, default "5 minutes" game time).
When a mob flees, memory persists so they can track the target
back.

## Tactic Rules (YAML)

### Format

```yaml
tactics:
  - trigger: target_casting
    action: trip
    priority: 10
    
  - trigger: health_below:30
    action: flee
    priority: 9
    
  - trigger: after_action:surprise-strike
    action: flee
    priority: 8

  - trigger: multiple_targets
    action: cast conviction-barrage
    priority: 7
```

### Evaluation

When an event fires:
1. Collect all tactics whose trigger matches the current state
2. Sort by priority (highest first)
3. For the highest-priority match, roll against `tactical_discipline`
4. If roll succeeds, queue the action with `reaction_delay`
5. If roll fails, skip (mob does nothing special this reaction)

Only one reaction fires per event — no chaining within a single
evaluation. The `after_action` trigger handles chaining across
sequential actions (trip → stomp is two separate reaction cycles).

### Trigger Vocabulary

**Combat state triggers:**

| Trigger | Fires when |
|---------|-----------|
| `combat_start` | Mob enters combat (aggro set) |
| `target_casting` | Current aggro target is casting a spell |
| `target_prone` | Current target is prone |
| `target_grappled` | Current target is grappled (and mob is controller) |
| `health_below:<pct>` | Mob's HP drops below this percentage |
| `multiple_targets` | More than one enemy in the room |
| `single_target` | Exactly one enemy in the room |
| `target_fled` | Aggro target left the room |

**Mob state triggers:**

| Trigger | Fires when |
|---------|-----------|
| `not_hidden` | Mob is not hidden and has stealth capability |
| `has_buff:<buffid>` | Mob has a specific buff active |
| `missing_buff:<buffid>` | Mob does not have a specific buff |
| `no_aggro` | Not in combat but has combat memory |
| `after_action:<action>` | Just successfully completed a specific action |

**Pre-combat triggers:**

| Trigger | Fires when |
|---------|-----------|
| `player_entered` | A player entered the mob's room |

### Action Vocabulary

**Direct commands** — any mob command string:
- `trip`, `bash`, `kick`, `grapple`, `submit`, `flee`
- `cast <spellid>`, `cast <spellid> <target>`
- `attack <target>`, `shoot <target>`
- `hide`, `sneak`

**Special actions** — framework-provided:
- `retarget_strongest` — switch aggro to the highest-power
  target in the room
- `call_for_help` — alert allied mobs in adjacent rooms;
  alerted mobs path to this room and engage the mob's current
  target
- `track_memory` — use the combat memory to path toward the
  last known location of the remembered target
- `recall:<roomid>` — path to a specific room (for retreat-
  heal-return patterns)

### Standard Tactic Sets

For convenience, common tactic patterns can be defined as named
presets and referenced in mob YAML:

```yaml
# On the mob:
tactic_preset: aggressive_melee
```

Presets are defined in a config file (or hardcoded initially).
Examples:

**`aggressive_melee`** (bandits, warriors):
```yaml
- trigger: target_prone
  action: kick
  priority: 10
- trigger: target_casting
  action: bash
  priority: 9
```

**`defensive_caster`** (wizards, priests):
```yaml
- trigger: combat_start
  action: cast chrysalis-cocoon
  priority: 10
- trigger: multiple_targets
  action: cast conviction-barrage
  priority: 9
- trigger: health_below:30
  action: flee
  priority: 8
```

**`ambusher`** (rogues, phantoms):
```yaml
- trigger: after_action:surprise-strike
  action: flee
  priority: 10
- trigger: no_aggro
  action: track_memory
  priority: 9
- trigger: not_hidden
  action: hide
  priority: 8
```

Mob-specific tactics in the YAML override or extend the preset.

## Implementation Approach

### New Package: `internal/mobai/`

Keeps AI logic separate from combat calculations and mob data.

- `reactor.go` — event listener, trigger matching, reaction queuing
- `tactics.go` — tactic rule parsing, evaluation, preset loading
- `memory.go` — combat memory struct and management
- `triggers.go` — trigger implementations (each trigger is a
  function that checks current state)
- `actions.go` — special action implementations (retarget, call
  for help, track memory, recall)

### Event Integration

Register listeners on existing events:
- `events.NewRound` — periodic re-evaluation + memory decay
- `events.RoomAction` — player movement, casting
- Custom events emitted from combat hooks when damage is dealt,
  aggro changes, or actions complete

The `after_action` trigger requires combat hooks to emit a new
event when a mob successfully completes a special move. This is a
small addition to the existing special move handlers.

### Mob YAML Fields

```yaml
# AI reaction tuning
reaction_delay: 0.5        # seconds before executing reaction
tactical_discipline: 0.95  # how reliably mob follows tactics
tactic_preset: ambusher    # named preset (optional)
tactics:                   # per-mob tactic overrides (optional)
  - trigger: target_casting
    action: trip
    priority: 10
```

### Config Knobs

| Key | Default | Description |
|-----|---------|-------------|
| `CombatMemoryDuration` | 300 | Rounds before combat memory expires |
| `MobAIEnabled` | true | Global toggle for the reactive AI system |
| `MobReactionDelayMin` | 0.25 | Minimum reaction delay (seconds) |
| `MobReactionDelayMax` | 4.0 | Maximum reaction delay (seconds) |

## Scope

### In Scope (this spec)
- Event-driven reaction framework
- Trigger/action vocabulary listed above
- Combat memory system
- Per-mob reaction delay and tactical discipline
- Tactic presets
- Integration with existing combat hooks

### Out of Scope (future work)
- Specific archetype implementations (Phantom, Edrin, Velk) —
  these are YAML configuration using the framework, not code changes
- Group tactics / mob coordination beyond `call_for_help`
- Learning AI / adaptive behavior
- Pathfinding improvements
- JS script replacement for quest/dialogue (that system stays)

## Migration

The existing `combat/ai.go` system (`ChooseSpecialMove`,
`ChooseCastAction`, `AIProfile`) continues to work as the fallback.
Mobs without `tactics` in their YAML behave exactly as they do
today. The new system only activates for mobs that have tactic rules
defined.

The `CombatCommands` list on mob YAML also continues to work as a
lower-priority fallback when no tactic matches.

## Files to Create

- `internal/mobai/reactor.go` — event listener and reaction dispatch
- `internal/mobai/tactics.go` — rule parsing, evaluation, presets
- `internal/mobai/memory.go` — combat memory management
- `internal/mobai/triggers.go` — trigger implementations
- `internal/mobai/actions.go` — special action implementations

## Files to Modify

- `internal/mobs/mobs.go` — add reaction_delay, tactical_discipline,
  tactics, tactic_preset, CombatMemory fields
- `internal/hooks/hooks.go` — register AI reactor listeners
- `internal/hooks/NewRound_DoCombat.go` — emit action-complete events
- `internal/hooks/NewRound_DoCombat_helpers.go` — integrate reactive
  AI with existing combat flow
- `internal/configs/config.balance.go` — add config knobs
- `_datafiles/config.yaml` — add config entries
