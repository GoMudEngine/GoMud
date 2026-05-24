# Mob Aliveness 2.7 — Mob Skullduggery Suite

> **Phase 2 tactical (seventh chunk).** Lift the four remaining
> skullduggery verbs (`steal`, `plant`, `defuse`, `shadow`) into
> the `internal/actions/` package alongside the existing
> `actions.ExecuteSneak`. Player commands become thin wrappers.
> Mob wrappers added for all five verbs. Five new btree action
> primitives + three new state-query conditions. One new mob
> archetype (`thief`) and one wired test mob (Thornwall
> highwayman). Sunset three pieces of legacy code that the
> consolidation makes redundant.

## Goal

Bandit-class mobs become bandits. A Thornwall highwayman that
sees a passing traveler should fade into cover, lift a coin
purse, and slip away — not stand in the road waving a sword.
This is the marquee aliveness lever for the chunk.

Two unlocked behaviors from one consolidated set of primitives:

- **Stealth-only opportunism (thief pattern):** an idle mob
  sneaks, waits for a player, attempts theft, flees — without
  ever entering combat unless attacked first.
- **Power-overmatch escape hatch:** the same mob, faced with a
  much weaker target, drops the stealth fantasy and engages
  directly. Re-uses chunk 2.4's `target_power_ratio_above`
  condition.

Defuse and shadow get lifted alongside steal/plant/sneak for
completeness — every player skullduggery verb has a mob-callable
twin and a btree primitive after this chunk. Active btree
consumers for defuse/shadow are deferred (defuse: trap-dungeon
mobs; shadow: chunk 5.2 bounty hunters) but the plumbing is in
place.

## Architectural musts

Brainstorming refined the framing:

1. **Five verbs lift to `actions/` via the actor pattern**,
   mirroring chunk 2.1 (`actions.Buy`) and 2.4
   (`actions.Consider`). Each exposes a single entry point
   `<Verb>(actor Actor, opts <Verb>Options) <Verb>Result`. Both
   player wrappers and mob wrappers thin to ~25 lines.

2. **Existing `actions.ExecuteSneak` renames to `actions.Sneak`**
   for naming parity with `Buy`/`Consider`. Same signature, same
   behavior. Existing `mobcommands/sneak.go` and
   `usercommands/skill.skullduggery.sneak.go` repointed.

3. **Player-side detection of mob theft is opposed-roll
   gated**, symmetric with the existing player-on-mob pattern.
   The mob's skill check decides whether the theft succeeds;
   an independent Per+Search vs Dex+Skullduggery roll on the
   victim decides whether they *notice*. High-Per players will
   nearly always see a thief; low-Per players may lose coin
   without ever knowing. Inventory diff is the only authoritative
   evidence — by design.

4. **Crime/opinion bookkeeping stays asymmetric.** Player→mob
   theft already records crimes (chunk 1.3) and bumps opinion
   (chunk 1.1). Mob→player theft records *nothing* in the
   substrate. Rationale: crimes are faction-scoped against a
   victim faction, and "the warren bandits as a faction
   committed a crime against player X" has no useful consumer
   yet — the player isn't a faction member, and the warren's
   rep with the player is already represented through 1.1.
   When the victim notices, the gameplay loop (chase, attack,
   call guard) is the response surface, not a ledger entry.

5. **Btree primitives are verb-shaped, one per action**, plus
   minimal state-query conditions. No composite "steal and
   flee" primitives — the archetype YAML composes existing
   target-selection, condition, and action primitives. Matches
   the 2.4 pattern where `target_weakest_mob_in_room` and
   `target_power_ratio_above` are independently useful.

6. **Target resolution follows the standard chain.** For
   `try_steal` / `try_plant` / `try_shadow`: prefer
   `ctx.Event.UserId` → `mob.Character.Aggro.UserId` →
   `mob.Character.Aggro.MobInstanceId`. Idle predation (no
   existing aggro) uses an existing target-picker primitive
   (e.g., a new `target_random_player_in_room` if one doesn't
   already exist — see Open Questions).

7. **One new archetype (`thief`), one wired test mob.**
   Following the chunk 2.4 precedent of minimal demo wiring.
   `thief.yaml` lives alongside `predator.yaml` in
   `_datafiles/world/dogmud/behaviors/archetypes/`. The
   Thornwall highwayman (mob 90) is the single test target for
   v1. Additional mobs flip to `thief` in a follow-up content
   pass.

8. **Legacy sunset is targeted, not opportunistic.** Three
   pieces of code become redundant once the actions-package
   consolidation lands: the `usercommands/stealth_detection.go`
   shim, an empty `pickpocket` test placeholder, and a
   triple-removal pattern on mob-hidden-bust in
   `hooks/go.go`. The dual-path sneak detection
   (buff-flag-AND-misc-data) in `go.go:81-86` is *not* touched
   — refactoring stealth state in the same chunk as adding new
   consumers is risk we don't need. Logged as followup.

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/actions/sneak.go` | RENAME | `ExecuteSneak` → `Sneak`; signature unchanged |
| `internal/actions/steal.go` | NEW | `Steal(actor, opts) StealResult` — lifted from usercommands |
| `internal/actions/plant.go` | NEW | `Plant(actor, opts) PlantResult` |
| `internal/actions/defuse.go` | NEW | `Defuse(actor, opts) DefuseResult` |
| `internal/actions/shadow.go` | NEW | `Shadow(actor, opts) ShadowResult` |
| `internal/actions/steal_test.go` | NEW | Unit tests covering success/fail/detection/cooldown/skill-progression |
| `internal/actions/plant_test.go` | NEW | Same shape |
| `internal/actions/defuse_test.go` | NEW | Same shape |
| `internal/actions/shadow_test.go` | NEW | Same shape |
| `internal/usercommands/skill.skullduggery.steal.go` | REWRITE | Thin wrapper (~25 LoC) |
| `internal/usercommands/skill.skullduggery.plant.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.defuse.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.shadow.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.sneak.go` | TOUCH | Repoint to `actions.Sneak` (already mostly wrapped) |
| `internal/usercommands/stealth_detection.go` | DELETE | Redundant cross-package shim |
| `internal/usercommands/usercommands_test.go` | MODIFY | Remove empty pickpocket placeholder at line 7142 |
| `internal/mobcommands/steal.go` | NEW | Thin wrapper |
| `internal/mobcommands/plant.go` | NEW | Thin wrapper |
| `internal/mobcommands/defuse.go` | NEW | Thin wrapper |
| `internal/mobcommands/shadow.go` | NEW | Thin wrapper |
| `internal/mobcommands/sneak.go` | TOUCH | Repoint to `actions.Sneak` |
| `internal/mobcommands/mobcommands.go` | MODIFY | Register `steal`, `plant`, `defuse`, `shadow` in `mobCommands` map |
| `internal/behaviortree/actions_skullduggery.go` | NEW | Five btree actions: `try_steal`, `try_sneak`, `try_plant`, `try_defuse`, `try_shadow` |
| `internal/behaviortree/conditions_skullduggery.go` | NEW | Three btree conditions: `mob_is_hidden`, `target_is_hidden`, `target_has_gold` |
| `internal/behaviortree/actions.go` | MODIFY | Register the five new actions in `init()` |
| `internal/behaviortree/conditions.go` | MODIFY | Register the three new conditions in `init()` |
| `internal/behaviortree/context.md` | MODIFY | Document new actions + conditions |
| `internal/hooks/go.go` | MODIFY | Collapse triple-removal at line 445-447 to single removal |
| `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml` | NEW | New archetype tree |
| `_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml` | MODIFY | `behavior_archetype: thief`, skullduggery rank bumped to 2 |
| `internal/actions/context.md` | MODIFY | Document the new actions |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.7 Done, roll-up 15/41 |

## Public API

### `actions.Steal`

```go
package actions

// StealOptions parameterizes a theft attempt. Source narrows the
// containing entity (mob inventory vs room container); if zero,
// the action resolves a default target from actor.Aggro.
type StealOptions struct {
    TargetMobInstanceId int    // pickpocket a mob
    TargetUserId        int    // pickpocket a player (mob-on-player path)
    ContainerNoun       string // steal from a named container in the room
    ItemNoun            string // if set, target this specific item (not gold)
}

// StealResult is the structured outcome.
type StealResult struct {
    Succeeded       bool   // skill check passed
    Detected        bool   // detection roll fired and the defender noticed
    StoleGold       int    // gold transferred (0 if item-only or failed)
    StoleItemId     int    // item id transferred (0 if gold-only or failed)
    StoleItemName   string // for messaging
    DefenderName    string // who/what was robbed
    OnCooldown      bool   // attempt was blocked by skullduggery cooldown
    Reason          string // when Succeeded==false, why
}

// Steal runs a skullduggery theft attempt from actor against
// the resolved target. Both UserActor and MobActor are supported.
// On success: gold/item transfer happens before return. On
// detection: actor and defender (and observers) get appropriate
// SendText calls. Cooldown is set on the actor's command map
// regardless of outcome (when a roll happened).
func Steal(actor Actor, opts StealOptions) StealResult
```

### `actions.Plant`

```go
type PlantOptions struct {
    TargetMobInstanceId int
    TargetUserId        int
    ContainerNoun       string
    ItemNoun            string // required — must be in actor's backpack
}

type PlantResult struct {
    Succeeded     bool
    Detected      bool
    PlantedItemId int
    DefenderName  string
    OnCooldown    bool
    Reason        string
}

func Plant(actor Actor, opts PlantOptions) PlantResult
```

`Plant` shares the skullduggery cooldown key with `Steal` (current
intentional design — preserve).

### `actions.Defuse`

```go
type DefuseOptions struct {
    TrapNoun string // names a trap in the current room
}

type DefuseResult struct {
    Succeeded     bool
    TrapName      string
    KitConsumed   bool   // disarm kit consumed (success only)
    KitBonusUsed  int    // bonus from disarm kit, for messaging
    TriggeredTraps []int // on failure, traps that fired
    Reason        string
}

func Defuse(actor Actor, opts DefuseOptions) DefuseResult
```

No cooldown — current behavior, preserved.

### `actions.Shadow`

```go
type ShadowOptions struct {
    TargetMobInstanceId int
    TargetUserId        int
}

type ShadowResult struct {
    Succeeded      bool
    Detected       bool   // target won the detection roll and knows they're shadowed
    TargetName     string
    OnCooldown     bool
    Reason         string
}

func Shadow(actor Actor, opts ShadowOptions) ShadowResult
```

Caller must already be hidden (buff 9) — preserved precondition.
Cooldown: separate `cfg.ShadowCooldown`.

### `actions.Sneak` (rename of `ExecuteSneak`)

```go
type SneakResult struct {
    Succeeded     bool
    Detected      bool
    RollHappened  bool // false when no observers — free sneak
    Reason        string
}

func Sneak(actor Actor) SneakResult
```

### Player wrapper shape (example — `steal`)

```go
func Steal(rest string, user *users.UserRecord, room *rooms.Room,
           flags events.EventFlag) (bool, error) {
    args := util.SplitButRespectQuotes(rest)
    if len(args) == 0 {
        user.SendText("Steal what from whom?")
        return true, nil
    }
    opts := parseStealArgs(args, room)
    actor := &actions.UserActor{User: user, Room: room}
    result := actions.Steal(actor, opts)
    formatStealResultForPlayer(user, result)
    return true, nil
}
```

`parseStealArgs` and `formatStealResultForPlayer` are
package-local helpers; the action itself stays I/O-clean except
for `actor.SendText` calls for required side-effect messaging
(observer text, defender text, room broadcast). Player-only
flair (colored help text, suggestion of alternates) lives in
the wrapper.

### Mob wrapper shape (example — `steal`)

```go
func Steal(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    opts := parseMobStealArgs(rest, mob, room)
    actor := &actions.MobActor{Mob: mob, Room: room}
    result := actions.Steal(actor, opts)
    return true, nil
}
```

`MobActor.SendText` is a no-op (existing convention) — observer
messaging still fires via room broadcast calls inside the
action, which the action emits via `room.SendText(...)`
directly (not through actor.SendText).

## Btree primitive shapes

### Actions

```yaml
# Attempts to steal from the resolved target. Target chain:
# ctx.Event.UserId → Aggro.UserId → Aggro.MobInstanceId.
# Returns Success on transfer (regardless of detection),
# Failure on skill-check fail or cooldown.
- type: action
  do: try_steal
  prefer_gold: true   # optional; default true

# Attempts to plant a named item on the resolved target. Item
# must be in mob inventory; if absent, returns Failure.
- type: action
  do: try_plant
  item_tag: "incriminating_letter"

# Attempts a sneak roll on self. Success applies hidden buff.
- type: action
  do: try_sneak

# Shadows the resolved target. Requires mob already hidden.
- type: action
  do: try_shadow

# Disarms a trap in the current room. Picks first trap if
# multiple. Returns Failure if no traps present.
- type: action
  do: try_defuse
```

### Conditions

```yaml
# True when self carries buff 9 (Hidden).
- type: condition
  check: mob_is_hidden

# True when resolved target carries buff 9.
- type: condition
  check: target_is_hidden

# True when resolved target has at least N gold.
- type: condition
  check: target_has_gold
  min: 10
```

`target_has_gold` resolves a player target only (mob gold is
internal merchant currency, not a steal target). Returns Failure
when no player resolvable.

## `thief` archetype YAML sketch

```yaml
# _datafiles/world/dogmud/behaviors/archetypes/thief.yaml
name: thief
description: |
  Sneak-first opportunist. Hides on idle, pickpockets passing
  players, flees on detection or aggression. Engages directly
  only when target is vastly outclassed.

tree:
  - type: selector
    children:

      # 1. Shared panic-flee branch (from chunk 2.6)
      - { type: include, archetype: "_panic_flee" }

      # 2. Power-overmatch combat opportunism
      - type: sequence
        on: player_enter
        children:
          - { type: condition, check: target_power_ratio_above, value: 1.5 }
          - { type: include, archetype: "generic_fighter" }

      # 3. Self-defense (if attacked, fight back)
      - type: sequence
        on: mob_attacked
        children:
          - { type: include, archetype: "generic_fighter" }

      # 4. Steal-and-flee loop
      - type: sequence
        on: mob_idle
        children:
          - { type: condition, check: mob_is_hidden }
          - { type: action, do: target_random_player_in_room }
          - { type: condition, check: target_has_gold, min: 5 }
          - { type: action, do: try_steal }
          - { type: action, do: flee_random_exit }

      # 5. Re-stealth when uncovered
      - type: sequence
        on: mob_idle
        children:
          - { type: condition, check: mob_is_hidden, invert: true }
          - { type: action, do: try_sneak }

      # 6. Fallback wander
      - { type: action, do: idle_wander }
```

(Exact include/inversion syntax to match existing archetype
conventions discovered during implementation; this sketch is
intent-level.)

## Test mob — Thornwall highwayman

`_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml`
changes:

- Set `behavior_archetype: thief`.
- Confirm skullduggery skill ≥ rank 2 (bump if not).
- All other fields preserved.

Single mob for v1. Additional thief-archetype mobs follow in a
content pass.

## Legacy sunset

1. **`internal/usercommands/stealth_detection.go`** — the
   `calcSneakScore` package shim that delegates to
   `actions.CalcSneakScore`. Redundant after wrappers import
   `actions` directly. **Delete entire file.**
2. **`internal/usercommands/usercommands_test.go:7142`** — empty
   `pickpocket` placeholder. Misleading; we are not implementing
   pickpocket as a separate command. **Remove.**
3. **`internal/hooks/go.go:445-447`** — triple-removal pattern
   on mob-hidden-bust: `CancelBuffsWithFlag`, `RemovePermaBuff(9)`,
   `Buffs.RemoveBuff(9)`. Defensive over-coding from earlier
   debugging session. **Collapse to single `CancelBuffsWithFlag`
   call** (the canonical removal path for `cancel-on-*` flags).
   Verify no regression on the mob-hidden-bust path during smoke.

Not touched (logged as followup):

- Dual-path sneak detection at `go.go:81-86` (buff flag AND
  misc-data `sneaking`). Refactoring stealth state in this
  chunk would risk regressing the new consumers. Followup
  ticket: stealth state consolidation.

## Testing & smoke validation

### Unit tests

`internal/actions/{steal,plant,defuse,shadow}_test.go` — for each
action:

- Success path (skill check passes, transfer happens).
- Failure path (skill check fails, no transfer).
- Detection path (mob steals from player, player wins detection
  roll, message fires).
- Cooldown path (second invocation within window returns
  `OnCooldown: true`, no roll).
- Skill progression (`OnSkillUse("skullduggery")` called on
  every roll that happened).
- Pre-existing sneak tests stay; rename `ExecuteSneak` →
  `Sneak` in test calls.

### No new btree primitive unit tests

Consistent with chunks 2.4 and 2.6 — primitives validated via
in-game smoke. The unit-test surface for btree primitives is
mostly registration + param parsing, not per-condition math.

### Smoke test plan

Test character at low Per/Dex, mid Per/Dex, and very high
power-level. For each:

1. **Stealth-only loop** (mid character). Walk to Thornwall
   Outskirts, enter the highwayman's room, wait 2–3 rounds,
   leave. Re-enter. Verify:
   - Inventory diff reveals lost gold (silent theft path).
   - Or detection message fires on a Per-favorable roll.
   - Mob remains hidden between visits.

2. **Detection path** (high-Per character). Same flow, but
   ensure the detection roll succeeds. Verify the message text
   matches design ("The Thornwall highwayman lifts N gold from
   your pocket and dashes off!") and the mob exits the room.

3. **Power-overmatch override** (high-level character). Walk
   in vastly stronger than the mob. Verify the
   `target_power_ratio_above: 1.5` branch fires — mob drops
   stealth, engages in combat per `generic_fighter` subtree.

4. **Self-defense** (any character). Attack the highwayman.
   Verify it fights back via `mob_attacked` branch; does not
   flee on the first hit.

5. **Build/data smoke:** `go build ./...` clean, all package
   tests pass, server boots past `mobs.LoadDataFiles()` and
   `behaviors.LoadDataFiles()` without panic.

## Risks / known limitations

- **`target_random_player_in_room` may not exist as a btree
  primitive yet.** If absent, the chunk adds it (small —
  walks `room.GetPlayers()`, picks at random, sets aggro).
  Confirmed during implementation.
- **Hidden-buff regen interaction** — buff 9 has a 15-round
  duration. A thief that sneaks, fails to find a target,
  loses hidden, and re-sneaks should be fine, but rapid
  cycling could be wasteful. Acceptable for v1; tune if smoke
  reveals jank.
- **Mob-on-player theft has no crime ledger entry** — by
  design (architectural must #4). If later content demands
  "find the bandit who stole my purse" gameplay, we add a
  separate substrate (or extend 1.4 knowledge) rather than
  retrofitting crime logs.
- **The highwayman's existing combat presence drops.** As a
  thief, the mob becomes much harder to engage casually —
  it'll be hidden most of the time. This is the intended
  behavior change; flag for the smoke test.

## Open questions

- **Does `target_random_player_in_room` already exist?** Best
  guess: there's some equivalent in
  `internal/behaviortree/actions_target.go` or similar.
  Verified during implementation; spec assumes either reuse
  or trivial-add.

## Out of scope

- **Pickpocket as a separate verb.** Steal handles it. An
  alias could be added later as a 3-line addition.
- **Hide as a separate verb.** Sneak applies the hidden buff.
- **Additional thief-archetype mob flips.** Just the
  highwayman for v1. Content pass follows.
- **Btree consumers for `try_defuse` and `try_shadow`.** The
  primitives ship, but no archetype uses them in v1 —
  defuse waits for trap-aware dungeon mobs; shadow waits for
  chunk 5.2 bounty hunting.
- **Stealth-state dual-path consolidation** at
  `hooks/go.go:81-86`. Followup.
- **Crime/opinion bookkeeping for mob-on-player theft.**
  Asymmetric by design (architectural must #4).

## Roadmap impact

- Chunk 2.7 marked Done.
- Roll-up: 15 / 41 done • 0 in progress • 26 not started.
- Unblocks: chunks 2.8 (mob scout/track/scan — sibling
  pattern), 5.2 (bounty hunting — consumes `try_shadow`
  primitive), 6.1 (Stillwater town-flavor pass — thief
  archetype now available for street toughs).
