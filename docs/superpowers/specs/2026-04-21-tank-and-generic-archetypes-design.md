# Tank/Taunter + Generic Fighter Archetypes — Design

**Date:** 2026-04-21
**Status:** Approved
**Related:** Companion Phase 4 shipped 2026-04-20 — introduced behavior-tree archetypes
(`melee_self_buff`, `pure_caster`) via `mob_combat_round` event. This spec adds two
new archetypes on the same framework.

## Goal

Two new behavior-tree archetypes for companion-type mobs:

- **tank_taunter** — signature taunt + knockdown control + self/ally buffs. Adopted by
  flesh golem (305), earth elemental (311), magma elemental (314).
- **generic_fighter** — straightforward control rotation (no taunt/buffs). Adopted by
  steppe spirit wolf (243), zombie (301).

Skeleton (300) and water elemental (310) are intentionally dumb trash-tier summons
and stay un-archetyped.

All moves + spells share one cooldown, so each archetype fires at most one "special"
per cooldown cycle and falls through to the legacy attack loop on cooldown rounds.

## Constraint surfaced during brainstorming

`mob.Command(cmd)` is fire-and-forget async — the btree can't know at dispatch time
whether the command will actually execute (cooldown / resource / state gates fire at
execution time). To get selector fall-through like `cast_best_in_category`, we need
a new btree action that checks readiness synchronously before issuing.

## Engine additions

### 1. New btree action `command_best_of`

Mirrors `cast_best_in_category`'s self-gating. Takes an ordered list of command names,
issues the first one that passes a readiness check, returns Success. If none ready,
returns Failure and the selector falls through.

```go
func actCommandBestOf(params map[string]any, ctx *EvalContext) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil {
        return Failure
    }
    cmds := getStringListParam(params, "cmds")
    for _, cmd := range cmds {
        if actions.CommandIsReady(mob, cmd) {
            mob.Command(cmd)
            return Success
        }
    }
    return Failure
}
```

### 2. Shared readiness helper `actions.CommandIsReady(mob, cmd string) bool`

Lives in `internal/actions/` (the same package that hosts `ExecuteTaunt`, `ExecuteBash`,
etc.). Switch by command name; returns true iff the command would fire right now.
Readiness factors, by command:

| Command | Readiness checks |
|---------|------------------|
| `taunt` | has aggro target, off shared-move cooldown, enough conviction |
| `rally` | off shared-move cooldown, enough conviction |
| `warcry` | off shared-move cooldown, enough stamina/conviction |
| `trip` | has aggro target, target not already prone/grounded, off cooldown, enough stamina |
| `bash` | has aggro target, off cooldown, enough stamina, AND (mob has shield OR species has `NaturalBash`) |
| `grapple` | has aggro target, target not already clinched/grounded, off cooldown, enough stamina |
| `kick` | has aggro target, off cooldown, enough stamina |

The readiness logic mirrors the command's execution-time gates, kept in sync by having
both paths call the same check. Current command implementations each duplicate a
version of these gates internally; when time permits, the command's own code can be
refactored to call `CommandIsReady` as a pre-check and early-return. For this spec,
we only need the btree to have a reliable yes/no signal — we do **not** refactor the
existing commands to call the helper (too much scope churn in one spec).

Unsupported command names in `CommandIsReady` → return false. Keeps it safe for
future btree writers who typo a command name; they see the branch never fire, fix
the name.

### 3. Three new btree conditions

| Condition | What it checks |
|-----------|---------------|
| `target_is_casting` | `mob.Character.Aggro.Target.Character.IsCasting()` |
| `target_aggro_not_on_me` | The target exists AND either has no aggro, or its aggro target is not the mob |
| `target_not_standing` | Target's `CombatPosition != PositionStanding` |

All three share the pattern of reading from the mob's aggro target via
`actions.ResolveTargetActor` semantics (actors.Actor interface, player or mob).
Missing target → condition returns Failure.

### 4. Shared rally + warcry, mob command wrappers

Per the mobs/players parity policy, extract the user-command bodies into shared
action helpers:

- `actions.ExecuteRally(actor Actor) ...Result`
- `actions.ExecuteWarcry(actor Actor) ...Result`

Pattern already established by `actions.ExecuteTaunt`, `ExecuteBash`, `ExecuteTrip`,
`ExecuteGrapple`, `ExecuteKick`. The shared function handles: cost/cooldown gating,
dice roll, buff application, message formatting (source-side; target-side text stays
in the command file since it's per-player).

User commands `usercommands/rally.go` and `usercommands/warcry.go` become thin
wrappers that call the shared action and handle display.

New mob command files `mobcommands/rally.go` and `mobcommands/warcry.go` are thin
wrappers mirroring `mobcommands/howl.go` (which wraps `ExecuteTaunt`). Register in
`mobcommands/mobcommands.go`.

## Content additions

### tank_taunter archetype

File: `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`

```yaml
# tank_taunter archetype
#
# Sticky front-liner that holds aggro, cycles self/ally buffs, and prefers
# knockdown/interrupt control over grappling. All moves share the special-
# move cooldown, so priorities determine what fires on any given ready-round.
#
# Decision order per mob_combat_round:
#   1. Interrupt — bash/trip a casting target
#   2. Taunt — if I'm not already the target's aggro
#   3. Bonus-damage kick — if target is prone (stomp) or clinched (knee)
#   4. Rally — mitigation buff, skip if active
#   5. Warcry — damage buff, skip if active
#   6-8. Knockdown cascade: bash (first if qualified) → grapple (single-enemy
#        only) → trip (final fallback)
#   fall through to legacy attack

tree:
  type: selector
  event: mob_combat_round
  children:
    - type: sequence
      children:
        - type: condition
          check: target_is_casting
        - type: action
          do: command_best_of
          cmds: [bash, trip]

    - type: sequence
      children:
        - type: condition
          check: target_aggro_not_on_me
        - type: action
          do: command_best_of
          cmds: [taunt]

    - type: sequence
      children:
        - type: condition
          check: target_not_standing
        - type: action
          do: command_best_of
          cmds: [kick]

    - type: sequence
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: mob_has_buff
            buff_id: 80
        - type: action
          do: command_best_of
          cmds: [rally]

    - type: sequence
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: mob_has_buff
            buff_id: 79
        - type: action
          do: command_best_of
          cmds: [warcry]

    - type: action
      do: command_best_of
      cmds: [bash]

    - type: sequence
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: multiple_enemies
        - type: action
          do: command_best_of
          cmds: [grapple]

    - type: action
      do: command_best_of
      cmds: [trip]
```

### generic_fighter archetype

File: `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`

```yaml
# generic_fighter archetype
#
# Melee mob with the same control toolkit as tank_taunter but no signature
# taunt / self-buffs. Good default for competent non-tank fighter companions.
#
# Decision order per mob_combat_round:
#   1. Interrupt — bash/trip a casting target
#   2. Bonus-damage kick — if target is prone (stomp) or clinched (knee)
#   3-5. Knockdown cascade: bash → grapple (single-enemy only) → trip
#   fall through to legacy attack

tree:
  type: selector
  event: mob_combat_round
  children:
    - type: sequence
      children:
        - type: condition
          check: target_is_casting
        - type: action
          do: command_best_of
          cmds: [bash, trip]

    - type: sequence
      children:
        - type: condition
          check: target_not_standing
        - type: action
          do: command_best_of
          cmds: [kick]

    - type: action
      do: command_best_of
      cmds: [bash]

    - type: sequence
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: multiple_enemies
        - type: action
          do: command_best_of
          cmds: [grapple]

    - type: action
      do: command_best_of
      cmds: [trip]
```

### Mob YAML wiring

Add `behavior_archetype: <name>` field to five mob templates under
`_datafiles/world/dogmud/mobs/summons/`:

| File | Archetype |
|------|-----------|
| `305-flesh_golem.yaml` | `tank_taunter` |
| `311-earth_elemental.yaml` | `tank_taunter` |
| `314-magma_elemental.yaml` | `tank_taunter` |
| `243-steppe_spirit_wolf.yaml` | `generic_fighter` |
| `301-zombie.yaml` | `generic_fighter` |

Skeleton (300) and water elemental (310) intentionally stay un-archetyped.

## Behavior under shared cooldown

Since all special moves + spells share one cooldown, the effective cadence on a
tank with all buffs inactive and in combat with a solo caster is:

- Round 1 (cooldown ready): target is casting → **interrupt** (bash or trip) fires,
  cooldown set.
- Rounds 2-N: cooldown not ready → every branch returns Failure → legacy attack loop.
- Cooldown clears, next ready round: target is casting again? re-interrupt. Not
  casting? check next priorities — not-holding-aggro → taunt, target prone → kick,
  rally/warcry buffs needed, etc.

Over a full fight, the tank naturally cycles through maintenance (rally, warcry) at
the start, then alternates taunt-refresh / interrupt / knockdown as the fight evolves.
Generic fighter is the same without the maintenance layer.

## Testing

### Unit tests (Go)

1. **`TestCommandIsReady_AllCommands`** — table-driven, one row per supported command.
   Each row sets up a mob state that should yield ready/not-ready and asserts the
   helper's response. Covers: cooldown, aggro, resources, target position, species
   flag for bash.
2. **`TestActCommandBestOf_FiresFirstReady`** — build a mob with specific readiness
   per command (e.g., taunt blocked, bash ready); dispatch action with
   `cmds: [taunt, bash, trip]`; assert bash was the one issued.
3. **`TestActCommandBestOf_AllFailReturnsFailure`** — all commands not-ready → action
   returns Failure so selector can fall through.
4. **`TestCondition_TargetIsCasting`** — target in IsCasting=true → Success; no target
   → Failure.
5. **`TestCondition_TargetAggroNotOnMe`** — target aggro on me → Failure; target aggro
   on someone else → Success; target has no aggro → Success.
6. **`TestCondition_TargetNotStanding`** — target standing → Failure; prone/clinched/
   grounded → Success.

### Smoke tests (server)

1. Summon an earth elemental, force combat with a caster mob (wraith), confirm the
   elemental bashes the caster during cast attempts (in-game look + watch the
   combat log for "knocks to the ground" during a cast).
2. Summon a flesh golem without shield, force combat with a caster, confirm the
   golem **trips** (not bashes — no naturalbash, no shield).
3. Summon a zombie (generic_fighter), force fight with a prone target (trip them
   first), confirm next cooldown round fires a stomp-variant kick (higher damage
   description than plain kick).
4. Summon a magma elemental (tank_taunter), let it fight solo against a single enemy,
   confirm it cycles through rally → warcry → taunt → bash/trip/grapple across
   cooldown cycles. Taunt should dominate once buffs are up.
5. Summon a magma elemental into a fight with multiple enemies, confirm grapple does
   NOT fire (multi-enemy gate).

## Out of scope

- **Refactoring existing command implementations** to call `CommandIsReady`
  themselves for execution-time gating. The helper mirrors the commands' existing
  gates; the commands keep their own internal checks. If the two drift, the btree
  might issue a command that the command itself then silently no-ops. Acceptable
  for v1 — flagged as a follow-up memory `project_command_readiness_drift.md` to
  merge the two paths at a future convenience.
- **Separate tank+mitigation vs tank+damage archetypes.** A single tank_taunter that
  maintains both rally and warcry is sufficient for now; if players later want
  specialization, spawn new archetypes without touching this code.
- **Individual tuning knobs per mob.** Archetype is assigned at the template level;
  if a specific tank mob (e.g., a unique boss) wants a custom order, they can ship
  a per-mob btree file (resolution order per-mob → archetype → none).
- **Assigning archetype to skeleton / water elemental.** Intentionally left dumb per
  user direction.
