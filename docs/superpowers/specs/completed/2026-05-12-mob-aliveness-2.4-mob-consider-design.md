# Mob Aliveness 2.4 — Mob `consider` + Threat-Aware Behaviors

> **Phase 2 tactical (fourth chunk).** Consolidate `consider`
> via the actor pattern so mobs and players use the same code
> path. Add two btree primitives (a condition for power-ratio
> checks, an action for opportunistic predation) plus demo
> wiring on two real archetypes (lookout + a new predator). No
> changes to `combat.PowerScore` math — gear is already
> reflected through the existing `ValueAdj` / `Get*Mitigation`
> pipes; the audit deliverable is documentation.

## Goal

Mobs can size up combat threats the same way players can.
Two unlocked behaviors from one set of primitives:

- **Reactive (lookout pattern):** when a player enters, a
  hidden lookout decides ambush vs. stay-hidden based on
  power ratio. If the player outclasses the lookout, the
  ambush silently fails and the mob remains hidden.
- **Opportunistic (predator pattern):** a wolf on an idle
  tick scans the room for a weaker mob it would normally
  consider prey and engages.

The chunk's *why* from the roadmap: "Lets NPCs decide to
flee a strong player or covet a player's weapon." This spec
drops the covet half entirely (players don't drop gear, so
there's nothing to covet) and replaces it with the
predator-on-prey case, which exercises the same primitives
and is more universally useful.

## Architectural musts

Brainstorming refined the framing:

1. **`consider` consolidates via the actor pattern**, mirroring
   chunk 2.1's `actions.Buy` consolidation. `actions.Consider(
   actor, target) ConsiderResult` does the math and (via
   `actor.SendText`) the optional text emission. `MobActor.
   SendText` is a no-op (existing convention) so the math runs
   for mobs but no text fires. Player wrapper and mob wrapper
   both collapse to ~15 lines.

2. **No changes to `combat.PowerScore`.** Audit established
   that equipment is already reflected through the existing
   math: weapon `DamageMultiplier` / `SpeedMultiplier` drive
   per-swing offense; equipment `StatMods` flow through
   `Stats.X.ValueAdj` into every stat-derived term;
   `char.GetPhysicalMitigation()` / `GetMagicalMitigation()` /
   `GetConvictionMitigation()` sum equipment mitigation;
   `char.GetDefenseScore(...)` includes equipment-driven
   dodge/parry/block. The audit deliverable is a new section
   in `internal/combat/context.md` documenting the gear
   contribution path — no math change.

3. **Btree condition: `target_power_ratio_above` /
   `target_power_ratio_below`.** Paired naming matches the
   existing `mob_health_below` pattern. Single `value`
   parameter (the ratio threshold). Returns
   `Success`/`Failure`. Stateless — the math runs each time
   the condition evaluates, so mid-combat gear changes are
   reflected automatically.

4. **Target resolution: `Event.UserId` → `Aggro` fallback.**
   The condition prefers `EvalContext.Event.UserId` (the
   triggering player from events like `player_enter`,
   `mob_hurt`). If the event has no user, it falls back to
   `mob.Character.Aggro` — which carries either `UserId` or
   `MobInstanceId`, transparently supporting mob-vs-mob
   assessment. Returns `Failure` if no target resolvable.

5. **Btree action: `target_weakest_mob_in_room`** for
   predation. Walks `room.GetMobs()`, computes
   `PowerScore(target) / PowerScore(self)` for each
   candidate, picks the lowest ratio, sets it as Aggro,
   returns Success. Skips: self, dead mobs, non-combatant
   mobs, mobs the caller's `mob.HatesMob(...)` returns false
   for, charmed mobs sharing the caller's owner (companions),
   and — when caller is itself charmed — its owner's other
   companions. Optional `ratio_below` param gates further
   (default `1.0` = engage anyone strictly weaker).

6. **Faction-awareness via `mob.HatesMob`.** The existing
   `mobs.Mob.HatesMob(other)` predicate handles same-species
   skip + the YAML `hates:` list. Wolves with
   `hates: [boar, rodent]` automatically prey on boars/rats
   and skip fellow canines. This is the canonical "should I
   attack this other mob" predicate in the codebase
   (`mobcommands/lookfortrouble.go` already uses it for the
   adjacent question), and it covers the faction-awareness
   concern without coupling 2.4 to the 1.2 `factions`
   substrate. If a mob's existing `hates:` list is empty,
   predation simply never fires for that mob — explicit
   author opt-in.

7. **Players are NOT predation targets** in
   `target_weakest_mob_in_room`. The action's job is the
   wolves-vs-wandering-mob path; player aggression flows
   through `Aggro` + the standard hostile-mob attack chain
   (which still fires alongside this action's selector
   branches). Including players would risk PvE side-effects
   we haven't designed for. Documented limitation; can be
   revisited.

8. **Demo wiring is minimal — no full archetype pass.**
   `lookout` archetype gains a `player_enter` branch gated
   on `target_power_ratio_above: 1.0` (self > target = I'm
   stronger = ambush). New `predator` archetype copies
   `generic_fighter`'s tree verbatim and adds a leading
   `mob_idle` branch that calls `target_weakest_mob_in_room
   ratio_below: 0.85`. Four wolf YAMLs flip
   `behavior_archetype: generic_fighter → predator`.
   No changes to non-target archetypes/mobs.

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/actions/consider.go` | NEW | `Consider(actor, target) ConsiderResult` |
| `internal/actions/consider_test.go` | NEW | Unit tests for `Consider` |
| `internal/usercommands/consider.go` | REWRITE | Thin wrapper: resolve target, call `actions.Consider` |
| `internal/mobcommands/consider.go` | NEW | Thin wrapper: parse rest, resolve target, call `actions.Consider` |
| `internal/mobcommands/mobcommands.go` | MODIFY | Register `consider` in the `mobCommands` map |
| `internal/behaviortree/conditions_combat.go` | NEW | `condTargetPowerRatioAbove`, `condTargetPowerRatioBelow` |
| `internal/behaviortree/conditions.go` | MODIFY | Register both conditions in `init()` |
| `internal/behaviortree/actions_combat.go` | MODIFY | Add `actTargetWeakestMobInRoom` |
| `internal/behaviortree/actions.go` | MODIFY | Register the action in `init()` |
| `internal/behaviortree/conditions_combat_test.go` | NEW | Unit tests for the new conditions |
| `internal/behaviortree/actions_combat_test.go` | MODIFY | Unit tests for the new action |
| `internal/behaviortree/context.md` | MODIFY | Document the new condition + action |
| `internal/combat/context.md` | MODIFY | NEW section: "Power Scoring & Gear Contribution" |
| `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml` | MODIFY | Add `player_enter` branch |
| `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml` | NEW | New archetype tree |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/205-steppe_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/206-young_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/223-scarred_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.4 Done, roll-up 12/41 |

## Public API

### `actions.Consider`

```go
package actions

// ConsiderResult is the structured output of an actor's consider
// action. Ratio is self power divided by target power; values > 1
// mean the actor outclasses the target.
type ConsiderResult struct {
    Ratio          float64 // 0 if target power is 0 (degenerate)
    SelfPower      float64
    TargetPower    float64
    TargetName     string
    TargetIsPlayer bool
}

// Consider computes a power-ratio assessment of target from
// actor's POV. For UserActor: also formats a colored prediction
// string and calls actor.SendText(...). For MobActor: SendText
// is a no-op (existing actor abstraction), so the math runs
// silently. Triggers OnStatUse("perception") on the actor.
func Consider(actor Actor, target Actor) ConsiderResult
```

The prediction-text emission path is preserved verbatim from
the pre-refactor `usercommands.Consider`:

| Ratio range | Prediction |
|---|---|
| `> 4` | "They pose no threat to you" (blue-bold) |
| `> 3` | "You hold a clear advantage" (green) |
| `> 2` | "The odds favor you" (green) |
| `> 1` | "An even contest — tread carefully" (yellow) |
| `> 0.5` | "They have the upper hand" (red-bold) |
| `> 0` | "You are severely outmatched" (red-bold) |
| `0` (zero target power) | "You will not survive this fight" (red-bold) |

Plus the leading "You consider <targetname>..." line, formatted
the same as today.

### Player wrapper (`internal/usercommands/consider.go`)

```go
func Consider(rest string, user *users.UserRecord, room *rooms.Room,
              flags events.EventFlag) (bool, error) {
    args := util.SplitButRespectQuotes(rest)
    if len(args) == 0 {
        return true, nil
    }
    target, err := actions.ResolveTargetActor(room, args[0],
        actions.ResolveTargetOptions{ExcludeUserId: user.UserId})
    if err != nil {
        if err == actions.ErrTargetVanished {
            user.SendText("You don't see them here.")
        }
        return true, nil
    }
    actor := &actions.UserActor{User: user, Room: room}
    actions.Consider(actor, target)
    return true, nil
}
```

### Mob wrapper (`internal/mobcommands/consider.go`)

```go
func Consider(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    if rest == "" {
        return true, nil
    }
    target, err := actions.ResolveTargetActor(room, rest,
        actions.ResolveTargetOptions{ExcludeMobInstanceId: mob.InstanceId})
    if err != nil {
        return true, nil
    }
    actor := &actions.MobActor{Mob: mob, Room: room}
    actions.Consider(actor, target)
    return true, nil
}
```

### Btree condition shape

```yaml
# True when self_power / target_power > value.
- type: condition
  check: target_power_ratio_above
  value: 1.5

# True when self_power / target_power < value.
- type: condition
  check: target_power_ratio_below
  value: 0.7
```

**Implementation outline** (`internal/behaviortree/conditions_combat.go`):

```go
func condTargetPowerRatioAbove(params map[string]any, ctx *EvalContext) Result {
    return targetPowerRatioCompare(params, ctx, true)
}
func condTargetPowerRatioBelow(params map[string]any, ctx *EvalContext) Result {
    return targetPowerRatioCompare(params, ctx, false)
}

// targetPowerRatioCompare resolves the mob, picks a target via
// Event.UserId → Aggro fallback, computes the power ratio, and
// compares to the value param. above=true returns Success when
// ratio > value; above=false returns Success when ratio < value.
func targetPowerRatioCompare(params map[string]any, ctx *EvalContext, above bool) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil {
        return Failure
    }
    threshold := getFloatParam(params, "value", 0)
    if threshold == 0 {
        return Failure // missing/zero value is treated as a config error
    }

    // Target resolution: Event.UserId → mob.Character.Aggro.
    selfChar := mob.Character
    targetPower, ok := resolveTargetPower(mob, ctx)
    if !ok {
        return Failure
    }

    selfPower := combat.PowerScore(selfChar)
    if targetPower <= 0 {
        // Degenerate target — caller treats as "infinitely
        // weaker"; above=true succeeds, below=true fails.
        if above {
            return Success
        }
        return Failure
    }
    ratio := selfPower / targetPower
    if above && ratio > threshold {
        return Success
    }
    if !above && ratio < threshold {
        return Success
    }
    return Failure
}

// resolveTargetPower returns the PowerScore of the contextual
// target, with fallback chain:
//   1. ctx.Event.UserId  → player
//   2. mob.Character.Aggro.UserId → player
//   3. mob.Character.Aggro.MobInstanceId → mob
// Returns (0, false) when no target resolvable.
func resolveTargetPower(mob *mobs.Mob, ctx *EvalContext) (float64, bool) {
    if ctx.Event.UserId > 0 {
        if u := users.GetByUserId(ctx.Event.UserId); u != nil {
            return combat.PowerScore(*u.Character), true
        }
    }
    if mob.Character.Aggro != nil {
        if mob.Character.Aggro.UserId > 0 {
            if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
                return combat.PowerScore(*u.Character), true
            }
        }
        if mob.Character.Aggro.MobInstanceId > 0 {
            if m := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); m != nil {
                return combat.PowerScore(m.Character), true
            }
        }
    }
    return 0, false
}
```

### Btree action shape

```yaml
# Pick the weakest mob in the room (by power ratio) and set
# Aggro on it. Optional ratio_below caps engagement to mobs
# whose power ratio (target/self) is below the threshold.
- type: action
  do: target_weakest_mob_in_room
  ratio_below: 0.7
```

**Implementation outline** (`internal/behaviortree/actions_combat.go`):

```go
func actTargetWeakestMobInRoom(params map[string]any, ctx *EvalContext) Result {
    mob := mobs.GetInstance(ctx.InstanceId)
    if mob == nil || mob.IsNonCombatant() {
        return Failure
    }
    room := rooms.LoadRoom(ctx.RoomId)
    if room == nil {
        return Failure
    }

    selfPower := combat.PowerScore(mob.Character)
    if selfPower <= 0 {
        return Failure
    }
    // ratio_below defaults to 1.0 (engage anyone strictly weaker).
    ceiling := getFloatParam(params, "ratio_below", 1.0)

    callerCharmedBy := mob.Character.GetCharmedUserId()
    var bestId int
    bestRatio := ceiling
    for _, otherId := range room.GetMobs() {
        if otherId == mob.InstanceId {
            continue
        }
        other := mobs.GetInstance(otherId)
        if other == nil || other.IsNonCombatant() {
            continue
        }
        if other.Character.Health <= 0 {
            continue
        }
        // Companion-allegiance skip: if caller is itself charmed,
        // skip fellow companions of the same owner. A wild caller
        // can still prey on a player's companion if HatesMob says
        // so — companions of an enemy are still enemies.
        if callerCharmedBy > 0 && other.Character.IsCharmed(callerCharmedBy) {
            continue
        }
        if !mob.HatesMob(other) {
            continue
        }
        targetPower := combat.PowerScore(other.Character)
        if targetPower <= 0 {
            continue
        }
        ratio := targetPower / selfPower
        if ratio < bestRatio {
            bestRatio = ratio
            bestId = otherId
        }
    }
    if bestId == 0 {
        return Failure
    }
    mob.Character.SetAggro(0, bestId, characters.DefaultAttack)
    return Success
}
```

**Notes on the action:**

- "Weakest" is defined as the lowest `target_power / self_power`
  ratio among eligible candidates. Lower is weaker.
- The optional `ratio_below` defaults to `1.0`. Without an
  override, the action engages any mob weaker than self that
  passes the `HatesMob` filter. Tighter thresholds (e.g., `0.7`)
  make the mob pickier — only engages clearly weaker prey.
- Aggro is set via `mob.Character.SetAggro(0, mobInstanceId,
  characters.DefaultAttack)` — the same path the existing
  `lookfortrouble` mobcommand uses. The next combat tick handles
  the actual swing through existing pipelines.
- The action does not move the mob between rooms — it only
  targets a candidate present in `room.GetMobs()`.

## PowerScore audit — `internal/combat/context.md`

A new section titled "Power Scoring & Gear Contribution" is
appended to the existing combat context doc. The section
documents the gear-flow table below and states explicitly that
PowerScore reflects gear via the standard `ValueAdj` /
mitigation pipes rather than as a separate axis. Outline:

```
## Power Scoring & Gear Contribution

`combat.PowerScore(char)` combines six terms: Offense, Defense,
Durability, Skills, Mutations, and KD ratio. Equipment
contribution flows through the standard pipes; there is no
separate "gear quality" axis.

| PowerScore term | Equipment field(s) that feed it |
|---|---|
| Offense (physAtk per-swing) | weapon `DamageMultiplier`, `SpeedMultiplier`; offhand + ExtraArm weapons |
| Offense (magAtk caster) | equipped weapon `SpellDamageMultiplier` |
| Offense (any stat-derived) | equipment `StatMods` → `Stats.X.ValueAdj` |
| Defense (mitigation) | equipment `PhysicalMitigation` / `MagicalMitigation` / `ConvictionMitigation` summed by `char.Get*Mitigation()` |
| Defense (avoidance) | equipment-driven dodge/parry/block via `char.GetDefenseScore(...)` |
| Durability | `char.HealthMax.Value` / `StaminaMax.Value` / `ConvictionMax.Value` — all reflect equipment stat boosts |
| Skills | not gear-driven |
| Mutations | not gear-driven |
| KD ratio | not gear-driven |

A player swapping a steel sword for an iron one will see
PowerScore drop because (a) the weapon's DamageMultiplier
changes (physAtk) and (b) any stat-mod difference flows through
ValueAdj into multiple terms. The Incorporeal mutation (chunk
2.2a) further scales gear contributions via
`mutations.GearEffectivenessMultiplier` — an ethereal wraith's
PowerScore reflects gear at the rank-determined fraction.

Consumers: `actions.Consider` (player + mob `consider`),
behavior tree conditions `target_power_ratio_above` and
`target_power_ratio_below`, behavior tree action
`target_weakest_mob_in_room`.
```

## Demo wiring

### Lookout archetype — add `player_enter` branch

Currently `lookout.yaml` only reacts to `packmate_hurt`,
`mob_hurt`, and `heard_callforhelp`. The new branch
ambushes weaker players that walk into the room; stronger
players pass through unmolested.

```yaml
tree:
  type: selector
  children:
    # NEW: ambush only if I outmatch the entering player.
    - type: sequence
      event: player_enter
      children:
        - type: condition
          check: target_power_ratio_above
          value: 1.0
        - type: action
          do: attack

    # Existing branches — preserved verbatim.
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: attack

    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: attack

    - type: action
      event: heard_callforhelp
      do: go_to_caller_room
```

Threshold `1.0` means: ambush anyone weaker than me. A more
conservative `1.2` would mean: only ambush clearly weaker
targets (require at least a 20% advantage). v1 ships `1.0` —
the lookout's `tactical_discipline: 0.80` and existing
`flee` tactic at `health_below:25` handle the "I picked a
fight I'm losing" recovery path.

### Predator archetype (new)

`_datafiles/world/dogmud/behaviors/archetypes/predator.yaml`:

```yaml
# predator archetype
#
# Opportunistic hunter. On idle ticks, scans the room for a
# weaker mob it would normally prey on (via mob.HatesMob —
# uses the YAML `hates:` list and same-species skip) and
# engages. Otherwise inherits the full generic_fighter
# combat behavior — packmate response, callforhelp
# navigation, and the per-round combat cascade (interrupt
# casts, kick prone/clinched, bash, grapple lone targets,
# trip).
#
# Faction/pack awareness comes from `hates:`; wolves with
# `hates: [boar, rodent]` will prey on boars and rats but
# never fellow canines. A mob with an empty `hates:` list
# effectively never opportunistically engages — the
# `target_weakest_mob_in_room` action will Failure-out.
#
# Example users: steppe wolves, alpha wolf, scarred wolf,
# young wolf. Distinguished from `generic_fighter` by the
# leading mob_idle predation branch.

tree:
  type: selector
  children:
    # NEW: opportunistic predation on idle ticks. Selector
    # short-circuits on Success; if a weaker hated mob is
    # in the room, Aggro is set and the next tick's
    # mob_combat_round fires the cascade below.
    - type: action
      event: mob_idle
      do: target_weakest_mob_in_room
      ratio_below: 0.85

    # ── Below: verbatim copy of generic_fighter.yaml ─────

    # packmate_hurt: engage the attacker.
    - type: action
      event: packmate_hurt
      do: attack

    # heard_callforhelp: navigate toward an adjacent caller.
    - type: action
      event: heard_callforhelp
      do: go_to_caller_room

    # mob_combat_round: original combat cascade.
    - type: selector
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

Threshold `ratio_below: 0.85` means: only engage prey
clearly weaker than self (target's power < 85% of mine).
Conservative; avoids near-peer engagements that the wolf
might lose.

### Wolf YAML changes

Four files flip `behavior_archetype: generic_fighter` to
`behavior_archetype: predator`:

- `205-steppe_wolf.yaml` — `groups: [steppe-wolf, canine]`, `hates: [boar, rodent]`
- `206-young_wolf.yaml` — same family
- `215-alpha_wolf.yaml` — same family
- `223-scarred_wolf.yaml` — same family

No changes to stats, items, or other fields. The existing
`hates:` lists drive predation targeting automatically.

## Testing

Per the chunks 2.1/2.2/2.3 pattern, tests use synthetic
`Mob`/`Character` values inline. Fixture-dependent integration
cases SKIP cleanly with documented reasons; full coverage relies
on the smoke section.

| Test file | Cases |
|---|---|
| `internal/actions/consider_test.go` | `ConsiderResult` math correctness — ratio = self/target; degenerate target_power=0 → ratio=0 + prediction "will not survive"; player text emitted via SendText; mob no-op (MobActor SendText is a no-op so we just verify the result struct); `OnStatUse("perception")` called once per invocation. |
| `internal/behaviortree/conditions_combat_test.go` | `target_power_ratio_above` with strong target → Failure; with weak target → Success; `_below` mirrored. Target resolution: `Event.UserId` populated → uses player; `Event.UserId == 0` + `Aggro.UserId` set → uses aggro player; `Event.UserId == 0` + `Aggro.MobInstanceId` set → uses aggro mob; both absent → Failure. Missing/zero `value` param → Failure. |
| `internal/behaviortree/actions_combat_test.go` | `target_weakest_mob_in_room`: empty room → Failure; non-combatant self → Failure; no hated mobs → Failure; weakest hated mob picked + Aggro set; companion-of-same-owner skipped; dead mob skipped; respects `ratio_below` param. |
| `internal/mobcommands/mobcommands_test.go` (parity) | Add `consider` to expected-commands list. |

## Smoke test

After unit tests pass:

1. `go build ./...` clean
2. `go test ./...` no FAILs
3. Boot the server, watch for clean data load — `behaviors.
   LoadDataFiles() loadedCount=...` increments by 1 (the new
   predator archetype).
4. Spot-check via admin:
   - **Player consider parity:** as a low-stat character, run
     `consider` on a high-stat NPC → see "outmatched" line.
     Run on a weak NPC → see "advantage" line. Confirm text
     identical to pre-refactor.
   - **Mob consider (no text):** as an admin, run
     `mob <mobid> consider <targetname>` — verify no panic
     and no text leaks to the room (MobActor.SendText no-op).
   - **Lookout ambush (weak player):** spawn the bandit
     lookout (mob 283). Walk into its room as a low-stat
     throwaway character → lookout attacks (room broadcast
     fires through the existing `attack` action).
   - **Lookout stays hidden (strong player):** as an
     admin-buffed character with elevated stats, walk into
     the lookout's room → no aggression; lookout remains
     hidden.
   - **Wolf predation:** spawn a steppe wolf (mob 205) and a
     rat (or other mob in its `hates:` list) in the same
     room with weaker stats → on next idle tick, the wolf
     sets Aggro on the rat. Verify via `mob <wolfid> show
     aggro` or by observing the combat that follows.
   - **Wolf skips same-species:** spawn two steppe wolves
     together with no prey present → neither sets Aggro on
     the other; both stay idle.
   - **Wolf skips companion:** charm a wolf, drop a charmed
     rat (companion of the same owner) into the room → wolf
     does NOT engage the rat. Same-owner companion skip.
5. Event-ordering sanity check: `MobIdle_HandleIdleMobs.go`
   already calls `behaviortree.TryMobBehavior(..., mob_idle)`
   BEFORE its legacy wander/lookfortrouble logic, and short-
   circuits via `events.Continue` when the btree returns
   true. So a successful predator branch (Aggro set) skips
   the wander step naturally — no call-site change needed.
   Confirm in smoke that a wolf which acquires a target does
   not also wander in the same tick.

## Out of scope / deferred

- **PowerScore tuning.** No math changes. If post-smoke data
  shows mobs flee too eagerly or commit to bad fights,
  re-tune thresholds in archetype YAMLs (cheap) before
  touching the formula.
- **Faction-substrate-based predation gating.** The 1.2
  `factions` package has `IsPeacefulToward(mob, userId)` but
  no `IsPeacefulTowardMob(self, other)`. Adding one would
  couple this chunk to 1.2 for one query; `HatesMob` covers
  the wolves-vs-pack-mates concern using existing
  infrastructure. Revisit if a real faction-aware predation
  scenario emerges that `HatesMob` can't express.
- **Per-mob predation cooldown.** Every idle tick is cheap
  (PowerScore for a handful of room mobs); if anti-fun
  emerges (wolves chain-engaging everything in sight), add a
  `ratio_recheck_cooldown` knob to the archetype. v1 ships
  uncooled.
- **Players as predation targets.** `target_weakest_mob_in_
  room` deliberately scopes to mob targets. Player
  aggression continues through the existing hostile-mob
  attack chain.
- **Scan radius beyond current room.** Predators don't peek
  into adjacent rooms.
- **Memoizing PowerScore.** `combat.PowerScore` is recomputed
  per condition/action evaluation. Cheap, but if hot-pathed
  the `actions_party` package's caching pattern would be
  the model. Profile after the chunk ships.
- **Full archetype audit / pass.** Only the lookout archetype
  is modified, and only the four ironwind wolves are flipped
  to predator. A broader pass (every fighter archetype,
  every aggressive mob) is content-pass work; not in this
  chunk.
- **Player-vs-player power-ratio gating.** Players don't run
  through the new btree primitives. PvP encounters use the
  same `actions.Consider` text-only path as today.
- **`appraise` deprecation.** Player command is obsoleted by
  the identify spell; could be retired separately. Out of
  scope here.

## Roadmap touchpoints

This chunk:

- Closes chunk **2.4** on `MOB_ALIVENESS_ROADMAP.md`. Roll-up
  moves from 11/41 → 12/41.
- Consumes nothing new beyond chunk 1.2 (faction/groups
  semantics via `mob.HatesMob`, which predates 1.2 but
  conceptually aligns) and existing actor infrastructure
  from chunk 2.1.
- Unblocks future chunks that need "is this fight worth it?"
  decisioning: chunk 2.6 (tactics-cast preemption — the
  power-ratio condition can gate offensive vs. defensive
  cast selection) and chunk 5.2 (bounty hunting — bounty
  hunters need to assess wanted targets before engaging).
