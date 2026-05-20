# Combat State — Chunk 4b: Position Control Axis Design

> **Side quest from mob aliveness chunk 2.7.** Sub-chunk 4b of the
> combat-state-machines redesign (master spec:
> `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`).
> Second of six sub-chunks in the rich-grapple expansion (master-spec
> section "3a. Position rich-grapple expansion"):
>
>   - 4a (shipped) — Position FSM scaffold: 14 states + per-state data,
>     dormant.
>   - **4b (this spec)** — Control-axis mechanics + writer/reader
>     cutover. Per-round opposed rolls with stamina/encumbrance
>     penalties, threshold-triggered position transitions, gradient
>     messaging, six new btree primitives. Migrates ~11 command-site
>     writers + ~10 reader sites from legacy `CombatPosition` enum to
>     the new FSM. Sunsets legacy fields. **End of 4b: the 14-state
>     position machine is the sole source of truth; control rolls fire
>     every round; the rich-grapple system is live.**
>   - 4c — Weapon-utility-by-position table.
>   - 4d — Submission system rework.
>   - 4e — Third-party interaction asymmetries.
>   - 4f — Balance pass + flavor text + full-stack smoke.
>
> **Aliveness paused for the duration** of chunks 1-6.

## Goal

Chunk 4a shipped the Position FSM as a dormant scaffold — 14 geometric
states, transitions, predicates, btree primitives — but no production
code transitions the machine. Chunk 4b lights it up: cuts over every
writer + reader from the legacy `CombatPosition` enum, adds the
per-round control-axis mechanics that make grappling tactically deep
(graduated drift toward thresholds, not single-round binary outcomes),
and sunsets the legacy fields.

This sub-chunk:

- **Migrates all 11 command-site writers** (`grapple` entry +
  progression, `trip`, `bash`, kick variant selector, spell knockdowns,
  recovery, `stand`, submission outcomes, grapple crit-fail, death
  cascade) to write the Position machine via `TransitionTo*` /
  `TransitionPair`.
- **Migrates all ~10 reader sites** (damage / speed / crit multipliers,
  defense filtering, kick variant selector, flee blockers, grapple roll
  modifiers, player `{pos}` prompt token, chunk-0 `RegisterPositionCheck`,
  chunk-2 Life pre-wire).
- **Adds per-round control-axis mechanics**: opposed Str+CombatSkill
  (+Dex×0.5 for controlled) rolls with stamina + encumbrance penalty
  curves, margin-based magnitude (1-3 levels of ControlLevel shift per
  round), gradient messaging at level transitions.
- **Adds threshold-triggered position transitions**: when a fighter's
  ControlLevel reaches `Controlled`, the position transitions per the
  default-escape-path table (Mount → HalfGuard, etc.). Pair-coordinated
  via `TransitionPair`.
- **Adds six new control-axis btree primitives** (`mob_is_in_control`,
  `mob_is_being_controlled`, `mob_control_at_least`, `mob_low_grapple_stamina`,
  `target_is_in_control`, `target_is_being_controlled`).
- **Adds four formal pair-state invariants + a periodic consistency
  checker** that force-breaks invalid pair state if it ever forms
  (defensive against design bugs).
- **Sunsets**: `Character.CombatPosition` enum field +
  `PositionRoundsMin` + `GrappleControllerId` + `ConditionGrappleController`
  + `internal/characters/combatposition.go` (legacy enum + helpers).
- **End-state**: the rich-grapple system is LIVE. Players in grapples
  feel control shifting round-by-round; the AI can query control state
  for tactical decisions; legacy enum is gone.

Largest sub-chunk in the chunk-4 series (~22-25 tasks). Single landing
point chosen over splitting because the cutover and control mechanics
are tightly coupled — cutover-without-mechanics ships "rewire for no
user-visible reason" with no positive smoke signal.

## Non-goals (4b)

- **Weapon-utility-by-position table.** 4c.
- **Submission system rework.** Today's `submission` command stays. 4d
  reworks toward opportunistic submissions.
- **Third-party interaction refinements.** Current "parry/dodge removed
  when grappled" logic in `combat/combat_helpers.go:400-429` stays
  unchanged in 4b (just rewired to read from the new FSM). 4e extends.
- **Voluntary controller-advancement** (Mount → BackGround via explicit
  player choice). 4b's automatic transitions only fire in the
  controlled-escape direction. Voluntary advancement is content / AI
  work; can land in 4d alongside submission attempts.
- **Per-state asymmetric stamina cost / penalty curves.** 4b uses
  uniform multipliers for controller vs controlled. Per-state
  differentiation (Mount-bottom drains worse than HalfGuard-bottom) is
  4f balance work.
- **Per-position roll formula variants** (Mount Str-heavy, Guard
  Dex-heavy). 4b uses the unified formula. Per-position formulas can
  add later via per-state extras on GrappleData wrapping structs.
- **State-specific extras on GrappleData** (ClinchGrip, ArmsIsolated,
  HooksIn, TrappedLeg, GuardVariant). 4c/4d adds them as wrapping
  structs as consumers materialize.
- **Multi-character (N-vs-1) grappling.** Master-spec out-of-scope.
- **Knockdown immunity** flags / mechanics. Future work.

## Architecture

`internal/state/position/` package gets the control-axis API
(`TransitionPair`, `InitialControlForPair`, `DefaultEscapeTarget`,
`ValidateGrapplePair`). The per-round tick lives in `internal/hooks/`
(new file). Cutover work touches `internal/combat/`,
`internal/hooks/`, `internal/usercommands/`, `internal/mobcommands/`,
`internal/characters/` per the cutover plan below.

### Files

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/position/pair.go` | NEW | `TransitionPair`, `InitialControlForPair`, `DefaultEscapeTarget`, `RoleController`/`RoleControlled` constants |
| `internal/state/position/control.go` | NEW | `IsController` derivation, `ControlLevel` arithmetic (Shift / Clamp), per-round control-roll formula helpers |
| `internal/state/position/control_test.go` | NEW | Unit tests for control arithmetic + initial-state table + escape-target table |
| `internal/state/position/validation.go` | NEW | `ValidateGrapplePair` — the four invariants encoded |
| `internal/state/position/validation_test.go` | NEW | Tests for each invariant violation case |
| `internal/state/position/position.go` | MODIFY | Add `IsController()` predicate; per-state predicates already in chunk-4a stay |
| `internal/hooks/Position_GrappleTick.go` | NEW | Per-round opposed-roll observer; finds controller side of each pair, calls `ProcessGrappleControlRound` |
| `internal/hooks/Position_GrappleTick_test.go` | NEW | Integration tests for per-round drift + threshold transitions |
| `internal/hooks/Position_ConsistencyCheck.go` | NEW | Periodic checker; runs every `PositionConsistencyCheckRounds`; force-breaks invalid pairs |
| `internal/hooks/Position_ConsistencyCheck_test.go` | NEW | Tests for detect + heal + no-false-positive |
| `internal/hooks/Position_Messaging.go` | NEW | Per-level gradient messages, transition messages, stamina-warning message; per-grapple cooldown tracking |
| `internal/characters/character.go` | MODIFY | Add `IsController()` predicate to Character (delegates to Position machine); add `PerGrappleMessageCooldowns` map field (yaml:"-") for the messaging cooldown tracking |
| `internal/characters/position_predicates.go` | MODIFY | Add `IsController()`, `IsBeingControlled()`, `IsLowGrappleStamina()` predicates |
| `internal/characters/combatposition.go` | DELETE (sunset) | Legacy enum + helpers; deleted after all readers migrated |
| `internal/combat/grapple.go` | MODIFY | `ApplyGrappleResult` / `ApplyPositionProgression` / `ApplySubmissionFailure` / `ApplySubmissionSuccess` / crit-failure block all migrate to `TransitionPair`; existing `CheckClinchProgression` / `CheckGroundedEscape` removed (replaced by per-round tick) |
| `internal/combat/skill_moves.go` | MODIFY | Trip / bash knockdown writes migrate; direction (Prone vs Supine) selected per attack type |
| `internal/combat/combat_helpers.go` | MODIFY | Damage / speed / crit / kick-variant / third-party-defense readers all repointed to Position predicates |
| `internal/hooks/spell_resolution.go` | MODIFY | Spell knockdown writers migrate (lines 408, 1121) |
| `internal/characters/skills.go` | MODIFY | `AttemptRecovery` migrates to `Position.TransitionToStanding` |
| `internal/usercommands/stand.go` | MODIFY | Explicit stand command migrates |
| `internal/usercommands/grapple.go` + `internal/mobcommands/grapple.go` | MODIFY | Entry command uses `TransitionPair` |
| `internal/usercommands/trip.go` + `bash.go` + `kick.go` (and mob equivalents) | MODIFY | Migrate writes / variant-selector reads |
| `internal/mobcommands/flee.go` + `internal/hooks/NewRound_DoCombat_helpers.go` | MODIFY | Flee blocker reads repointed |
| `internal/hooks/Life_Cascades.go` | MODIFY | DELETE lines 55-57 (the chunk-2 pre-wire `c.CombatPosition = PositionStanding` + `c.GrappleControllerId = 0`); the chunk-4a `Position_Cascades.go` observer already handles the equivalent |
| `internal/hooks/CombatPhase_Vetoes.go` | MODIFY | `RegisterPositionCheck` rewired from `c.CombatPosition == PositionStanding` to `c.IsStanding()` |
| `internal/users/userrecord.prompt.go` | MODIFY | `{pos}` token repointed to new Position state for display |
| `internal/hooks/NewRound_DoCombat.go` | MODIFY | DELETE existing `ProcessGrappleProgression` block (lines 327-378); per-round tick is the replacement |
| Various test files | MODIFY | Test fixtures parallel-write both fields during migration window; final sunset deletes legacy field writes |
| `_datafiles/messages/position_control.yaml` | NEW | YAML config for control-level gradient messages, transition messages, stamina warning |
| `internal/state/position/context.md` | MODIFY | Document control-axis API + messaging contract + invariants |
| `internal/hooks/context.md` | MODIFY | Document GrappleTick + ConsistencyCheck + Messaging observers |
| `internal/characters/context.md` | MODIFY | Document new control-axis Character predicates + sunset notes for legacy fields |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 4b Done |

## Per-round control roll mechanics

### Tick wiring

A new observer `Position_GrappleTick.go` registers on `NewRound`
events. Per round:

1. Iterate active players + mobs in rooms with active combat.
2. For each character with `c.Position.IsGrappling()` AND
   `c.IsController()` (one side of each pair):
   - Look up partner via `c.Position.GrappleData().Partner` → resolve to
     `controlled` Character via `actions.ResolveActorRef`.
   - Validate the pair via `position.ValidateGrapplePair(controller,
     controlled)`. On violation, force-break the pair (Section 4) and
     skip further processing for this round.
   - Call `ProcessGrappleControlRound(controller, controlled)`.
3. Skip characters where `c.IsController()` is false — the controlled
   side gets processed via their controller's iteration.

This ensures one roll per pair per round.

`ProcessGrappleControlRound` lives in `Position_GrappleTick.go`. It
performs the roll, computes the magnitude, shifts both characters'
ControlLevels via `position.ShiftControl`, fires gradient messages via
`Position_Messaging`, applies stamina costs, and checks for
threshold-triggered transitions.

The existing `ProcessGrappleProgression` block in
`NewRound_DoCombat.go:327-378` (which calls today's binary
`CheckClinchProgression` / `CheckGroundedEscape`) is deleted as part of
the cutover.

### Roll formula

```
controller_score = (Str + CombatSkill)
                   × stamina_multiplier(controller.staminaFraction)
                   × encumbrance_multiplier(controller.carryFraction)

controlled_score = (Str + CombatSkill + Dex × 0.5 + body_armor.EscapeModifier)
                   × stamina_multiplier(controlled.staminaFraction)
                   × encumbrance_multiplier(controlled.carryFraction)

result = dice.OpposedRollStat(controller_score, controlled_score)
```

`body_armor.EscapeModifier` reads from the controlled character's body
armor slot (existing field on ItemSpec; today consumed only by
`CheckGroundedEscape`). Stays as-is in 4b.

### Margin → control delta

Margin = `result.ZScore` (the opposed roll's z-score; positive favors
controller).

| `|ZScore|` range | Shift magnitude | Direction toward winner |
|---|---|---|
| 0.0 – 0.5 | 0 levels (no shift) | — |
| 0.5 – 1.0 | 1 level | toward winner |
| 1.0 – 2.0 | 2 levels | toward winner |
| ≥ 2.0 (crit) | 3 levels | toward winner |

"Toward winner" means: if `ZScore > 0` (controller wins), the
controlled fighter's ControlLevel shifts toward `Controlled`, and the
controller's shifts toward `InControl` (mirrored). If `ZScore < 0`, the
reverse.

Helper: `position.ShiftControl(currentLevel, delta) ControlLevel`
clamps to the [InControl, Controlled] range.

### Stamina penalty curve (new, more punishing than combat default)

```
stamina_multiplier(s) = 1.0 - GrappleStaminaPenaltyMax × (1 - s)^GrappleStaminaPenaltyCurve
```

Defaults (in `_datafiles/config.yaml` under `Balance`):
- `GrappleStaminaPenaltyMax = 0.60`
- `GrappleStaminaPenaltyCurve = 1.5`

Sample values:

| Stamina fraction | Multiplier |
|---|---|
| 1.0 (full) | 1.00 |
| 0.5 | ~0.79 |
| 0.25 | ~0.61 |
| 0.05 | ~0.45 |
| 0.0 | 0.40 |

Significantly steeper than the existing combat stamina curve (caps at
28% penalty). Applies symmetrically to both sides.

### Encumbrance penalty curve (new)

```
encumbrance_multiplier(e) = 1.0 - GrappleEncumbrancePenaltyMax × (max(0, e - 0.5) / 1.5)^GrappleEncumbrancePenaltyCurve
```

Where `e` = controller's or controlled's carry fraction (current load
/ carry capacity). Defaults:
- `GrappleEncumbrancePenaltyMax = 0.80`
- `GrappleEncumbrancePenaltyCurve = 1.5`

Sample values:

| Carry fraction | Encumbrance tier | Multiplier |
|---|---|---|
| ≤ 0.5 | Light | 1.00 (no penalty) |
| 0.8 | Moderate | ~0.88 |
| 1.0 | Heavy | ~0.75 |
| 1.5 | Overburdened | ~0.40 |
| 2.0+ | Crushed | ~0.20 |

The body-armor `EscapeModifier` (already on ItemSpec; controlled-side
only) is independent of the encumbrance penalty — they represent
different concepts (armor design for escape vs total carried weight)
and both apply.

### Stamina cost per round (asymmetric by role)

```
controller_cost = GrappleStaminaCostPerRound × ControllerCostMultiplier
controlled_cost = GrappleStaminaCostPerRound × ControlledCostMultiplier
```

Defaults:
- `GrappleStaminaCostPerRound = 5`
- `ControllerCostMultiplier = 1.0`
- `ControlledCostMultiplier = 2.0`

Per round: controller pays 5; controlled pays 10. Over 10 rounds: 50
vs 100 stamina. Pairs with the steeper stamina penalty curve to
produce the natural "I'm getting smothered" feedback loop — controlled
fighter drains faster, hits low stamina sooner, suffers the penalty
curve earlier, roll quality collapses, stays controlled longer, drains
more stamina, etc.

Both characters keep grappling at 0 stamina (no auto-break). Their
penalty curve maxes out (~0.40-0.60x roll multiplier). Future
enhancement: low-stamina threshold could trigger auto-tap — deferred
to 4d submission system.

### Per-state cost asymmetry — deferred

4b uses uniform multipliers per role. Per-state asymmetry (e.g.,
Mount-bottom drains worse than HalfGuard-bottom because it's
mechanically more compressing) is 4f balance work. The
`GrappleStaminaCostPerRound` const becomes a config-data lookup table
in 4f.

## Position transitions + pair coordination

### Initial ControlLevel per state on transition entry

When `TransitionPair(controller, controlled, target, r)` fires, the
controller and controlled sides start with role-appropriate
ControlLevels. Table:

| Position | Controller starts at | Controlled starts at | Notes |
|---|---|---|---|
| Clinch | Neutral | Neutral | Symmetric standing grapple |
| BackStanding | InControl | Controlled | Back-controller has full advantage |
| Mount | InControl | Controlled | Top has full advantage |
| SideControl | InControl | Controlled | Top has full advantage |
| KneeOnBelly | LosingControl | BecomingControlled | KOB is offensively powerful but UNSTABLE — top starts slightly losing |
| NorthSouth | LosingControl | BecomingControlled | NS is harder to maintain than Mount/SC |
| Crucifix | InControl | Controlled | Arms isolated = max control |
| BackGround | InControl | Controlled | Rear mount = most dominant |
| HalfGuard | Neutral | Neutral | True 50/50 transitional position |
| Guard | Controlled | InControl | **Inverted!** Bottom controls top (active legs). Bottom is the "controller" per our naming convention; top is "controlled" by the legs |
| Turtle | BecomingControlled | LosingControl | Defensive; curling fighter has some control over their own back |

The Guard inversion is the design subtlety — in BJJ, the bottom of
guard is *active* and the top is *contained*. Our `IsController`
derivation (`ControlLevel ∈ {InControl, LosingControl}`) handles this
naturally: bottom is the controller in Guard, top is the controller in
Mount.

Helper: `position.InitialControlForPair(state State, role Role)
ControlLevel`. Single source of truth for the table; callers pass
`RoleController` or `RoleControlled`.

### Threshold-triggered position transitions

After each per-round drift, if either character's ControlLevel reaches
`Controlled`:

1. **Either side's `Controlled` triggers escape.** The controlled
   fighter has escaped (their controller's grip collapsed, or they
   themselves overpowered).
2. **Default escape path lookup**: `position.DefaultEscapeTarget(currentState)
   State`. Per state:

| From | Default escape target |
|---|---|
| Clinch | Standing (break apart) |
| BackStanding | Standing (controlled spins out) |
| Mount | HalfGuard (most common BJJ outcome) |
| SideControl | Guard (controlled recovers full guard) |
| KneeOnBelly | Guard |
| NorthSouth | Guard |
| Crucifix | SideControl (controller backs off arm isolation) |
| BackGround | Mount (controlled rolls face-up, controller follows) |
| HalfGuard | Guard (bottom recovers full guard) |
| Guard | Standing (top stands up out of guard) |
| Turtle | Standing (controlled stands up) |

3. **Fire `TransitionPair(controller, controlled, escapeTarget, r)`.**
   New state initializes per the per-state initial-control table
   above. Cycle continues from the new state.

**No automatic controller-advancement in 4b.** When the controller
hits `InControl` (their best), the position does NOT automatically
transition to a more dominant state (Mount → BackGround). Advancement
requires an explicit player/AI choice — deferred to 4c/4d/content
work.

### `TransitionPair` API

```go
// package position

type Role int
const (
    RoleController Role = iota
    RoleControlled
)

// TransitionPair atomically moves a controller + controlled pair into
// the same grapple state with role-appropriate initial ControlLevels.
// Validates the input (both must be in compatible source states);
// returns error and refuses if invalid. On second-side transition
// failure, rolls back the first via snapshot to keep pair consistent.
func TransitionPair(
    controller, controlled *characters.Character,
    target State,
    r state.TransitionReason,
) error
```

Direct calls to `c.Position.TransitionToMount(...)` etc. remain
available for tests + edge cases but are considered low-level — they
don't enforce pair semantics. All command-site writers (grapple,
trip, bash, recovery, etc.) use `TransitionPair` where pair semantics
apply.

`Standing` is a special case: transitioning to Standing breaks the
grapple entirely. `TransitionPair(_, _, Standing, _)` transitions
both sides to Standing and clears both GrappleData slots.

### Knockdown direction logic for trip/bash/spell migration

Trip and bash currently set Prone uniformly. With Prone/Supine
distinction live, callers choose direction:

| Attack source | Default direction |
|---|---|
| Trip / leg sweep / sucker punch from behind | Prone (face-forward) |
| Bash / backward kick / push from front | Supine (fall on back) |
| Spell knockback (direct-hit blast) | Supine |
| Spell shockwave / area knockdown | Prone |

Hardcoded in W4 / W5 cutover tasks per attack type. Per-attack
config field for tuning lands in 4f.

## Messaging contract

### Three message classes

1. **Control-level gradient.** Fires when ControlLevel changes level
   (one step), gated by per-grapple-session cooldown (each level
   fires its message at most once per grapple session, to avoid spam
   when control oscillates around a boundary).

   Per-character, second-person to players, third-person to room
   observers. Templates per role + level. Initial table (polished in
   4f):

| New level | Controller msg ("you") | Controlled msg ("you") |
|---|---|---|
| InControl | "You settle into a dominating {position}." | (controlled doesn't reach InControl from a normal start) |
| LosingControl | "You feel your {position} slipping." | "You feel an opening — they're losing the position." |
| Neutral | "The position is even — it's all up for grabs." | (same — both sides see the shift) |
| BecomingControlled | "You're about to lose the {position}." | "You're starting to control the position!" |
| Controlled | "You've lost the {position}!" (next round will transition) | "You're about to escape!" (next round will transition) |

   Templates in `_datafiles/messages/position_control.yaml`. {position}
   substitution renders the position's display name (e.g., "mount",
   "side control", "back-ground"). Room observers see a generic
   third-person form: "{Controller} struggles to maintain the
   {position}."

2. **Position transitions.** Fire when a threshold-triggered
   transition lands. Two per-character + one room msg:

   - Controller: "You lose the {old position} as they recover to
     {new position}."
   - Controlled: "You escape the {old position} to {new position}!"
   - Room: "{Controller} loses the {old position} to {Controlled},
     who recovers to {new position}."

   No cooldown — transitions are rare events and each should be
   announced.

3. **Stamina exhaustion warning.** Fires when either fighter first
   drops below `GrappleStaminaLowThreshold` (default 25%) during the
   current grapple. "You're getting gassed — your {position} is hard
   to maintain." Fires once per grapple per character, not per round.

### Per-grapple cooldown tracking

Character carries a non-persisted map field:
```go
PerGrappleMessageCooldowns map[string]bool `yaml:"-"`
```

Keys: `"gradient_in_control"`, `"gradient_losing_control"`, ...,
`"stamina_low"`. Set to true on first fire; checked before subsequent
fires; cleared when the grapple ends (Standing transition).

### YAML config for templates

`_datafiles/messages/position_control.yaml` ships with starter copy.
Schema:

```yaml
gradient_messages:
  controller:
    in_control:
      self: "You settle into a dominating {position}."
      room: "{Controller} settles into a dominating {position}."
    losing_control:
      self: "You feel your {position} slipping."
      room: "{Controller} struggles to maintain {position}."
    # ... etc.
  controlled:
    losing_control:
      self: "You feel an opening — they're losing the position."
      # ... etc.

transition_messages:
  controller:
    self: "You lose the {old_position} as they recover to {new_position}."
    room: "{Controller} loses the {old_position} to {Controlled}, who recovers to {new_position}."
  controlled:
    self: "You escape the {old_position} to {new_position}!"

stamina_warning:
  self: "You're getting gassed — your {position} is hard to maintain."
  room: "{Character} looks exhausted in the {position}."
```

YAML loader caches templates at boot. 4f balance pass polishes copy.

## Btree primitives (control axis)

6 new primitives added to `internal/behaviortree/conditions_position.go`:

```go
"mob_is_in_control"           // mob.IsController() — ControlLevel ∈ {InControl, LosingControl}
"mob_is_being_controlled"     // mob.IsBeingControlled() — ControlLevel ∈ {BecomingControlled, Controlled}
"mob_control_at_least"        // parameterized — takes "level" (string) param; true if mob's ControlLevel is at-or-better-than the param for the mob's role perspective
"mob_low_grapple_stamina"     // mob.IsLowGrappleStamina() — stamina fraction < GrappleStaminaLowThreshold
"target_is_in_control"        // resolve target; check IsController()
"target_is_being_controlled"  // resolve target; check IsBeingControlled()
```

Combined with chunk-4a's 10 primitives, btree authors have 16 total
position-related queries by end of 4b. Enables tactical patterns like:

```yaml
- selector:
    children:
      - sequence:
          # If I'm in mount AND in control AND target is gassed,
          # try the (eventual 4d) submission.
          children:
            - mob_in_mount:
            - mob_is_in_control:
            - target_low_grapple_stamina:
            - try_submission:  # 4d work
      - sequence:
          # If I'm being controlled, prioritize escape.
          children:
            - mob_is_being_controlled:
            - try_escape:  # could be cancel_activity + flee, etc.
      - default_combat
```

Actual btree authoring is content / aliveness work after chunk 6 —
4b ships only the primitives, content uses them later.

## Cutover plan

Following chunk-3's parallel-write strategy. Each writer site is
migrated in a separate task; readers migrate after writers; legacy
fields delete after all readers migrated.

### Writers (8 task slots)

| Task | Writer site | Notes |
|---|---|---|
| W1 | `internal/combat/grapple.go:ApplyGrappleResult` | Grapple entry → Clinch or Mount/SC/etc. Use `TransitionPair`. Parallel-write `CombatPosition` legacy fields. |
| W2 | `internal/combat/grapple.go:ApplyPositionProgression` | Replaced — per-round tick now drives progression. Existing `CheckClinchProgression` / `CheckGroundedEscape` deleted. |
| W3 | `internal/combat/grapple.go:ApplySubmissionFailure` / `ApplySubmissionSuccess` | Submission outcomes (controller → Prone, controlled → Prone). Use `Position.TransitionToProne` (not pair — they're solo knockdowns). |
| W4 | `internal/combat/skill_moves.go` (trip / bash) | Knockdowns with direction logic. Use `Position.TransitionToProne` or `TransitionToSupine` based on attack type. |
| W5 | `internal/hooks/spell_resolution.go:408,1121` | Spell knockdowns. Direction per spell type. |
| W6 | `internal/characters/skills.go:AttemptRecovery` | Auto-recovery from Prone/Supine → Standing. `Position.TransitionToStanding`. |
| W7 | `internal/usercommands/stand.go` | Explicit stand command. `Position.TransitionToStanding`. |
| W8 | `internal/combat/grapple.go` crit-failure block | Grapple crit-fail → attacker Prone. `Position.TransitionToProne`. |

Each Wn task: parallel-write both legacy `CombatPosition` field AND
new Position machine; commit; verify regression tests + smoke.

### Readers (6 task slots)

| Task | Reader site | Migration |
|---|---|---|
| R1 | `internal/combat/combat_helpers.go` damage / speed / crit / kick-variant blocks | Replace `c.CombatPosition == X` with `c.IsX()` predicates; `c.CombatPosition.GetSpeedMultiplier()` → new helper `c.GetPositionSpeedMultiplier()` on Character |
| R2 | `internal/combat/combat_helpers.go:400-429` third-party defense filtering | Replace `target.CombatPosition.IsGrapplePosition()` with `target.IsGrappling()`. Behavior preserved (4e refines) |
| R3 | `internal/mobcommands/flee.go` + `internal/hooks/NewRound_DoCombat_helpers.go:504-510` flee blockers | Replace position-enum checks with `c.IsStandingGrapple() || c.IsGroundGrapple()` |
| R4 | `internal/hooks/Life_Cascades.go:55-57` chunk-2 pre-wire | DELETE. Chunk-4a `Position_Cascades.go` observer is now sole owner. |
| R5 | `internal/hooks/CombatPhase_Vetoes.go:32-33` chunk-0 `RegisterPositionCheck` | Rewire to `c.IsStanding()`. |
| R6 | `internal/users/userrecord.prompt.go` `{pos}` token | Repoint to new Position state display. Format: read `c.Position.State().String()` (returns "Standing" / "Prone" / "Mount" / etc.); display non-Standing positions with color. Need to add `GetPositionColor()` helper on Character or `position.State` (preserve existing chunk-4a color helpers if any). |

### Test fixtures + sunset (6 task slots)

| Task | Notes |
|---|---|
| F1 | Migrate ~4 test files that set `CombatPosition` directly to also call `Position.TransitionTo*`. Parallel-write pattern. |
| S1 | DELETE `Character.CombatPosition CombatPosition` field. Verify zero remaining readers. |
| S2 | DELETE `Character.PositionRoundsMin int` field. Folded into `ProneData.MinRecoveryRounds` / `SupineData.MinRecoveryRounds` (4a already has these fields). |
| S3 | DELETE `Character.GrappleControllerId int` field. Controller role derived from ControlLevel; partner ref in GrappleData. |
| S4 | DELETE `ConditionGrappleController` constant + all usages. Replaced by `c.IsController()`. |
| S5 | DELETE `internal/characters/combatposition.go` (legacy enum + `IsGroundPosition` / `IsGrapplePosition` / `GetSpeedMultiplier` / `GetPositionColor` / `GetWorstPosition` helpers). All callers migrated. |

Cutover sequence — strict order:
- W1-W8 land first (writers parallel-write both fields).
- F1 migrates test fixtures (parallel-write).
- R1-R6 migrate readers (now read from new FSM; legacy field unused except for parallel writes).
- S1-S5 sunset (deletion; verify zero references).

## Consistency invariants + safeguards

### The four invariants

1. **Single-partner invariant.** For any character `c` in a grapple
   state, `c.Position.GrappleData().Partner` is non-zero AND refers to
   exactly one other character — and that character is also in a
   grapple state. (Turtle is the exception — solo Turtle has zero
   Partner.)

2. **Bidirectional invariant.** If `c1.Partner = c2.Self` then
   `c2.Partner = c1.Self`. Always reciprocal. No one-way "I'm
   grappling them but they don't know it."

3. **Matching-state invariant.** If `c1` and `c2` are grappling each
   other, they're in the SAME grapple state.

4. **Role-exclusivity invariant.** For asymmetric grapple states (any
   except Clinch / HalfGuard / Turtle), exactly one side is the
   controller (`ControlLevel ∈ {InControl, LosingControl}`) and the
   other is the controlled (`ControlLevel ∈ {BecomingControlled,
   Controlled}`). Both at Neutral is allowed only for the three
   symmetric states. Both at InControl is never valid for an
   asymmetric state.

The "control chain" failure (A→B→C→A) is impossible if invariants 1+2
both hold: every grapple link is bidirectional and exclusive, so cycles
can't form.

### `position.ValidateGrapplePair(a, b *Character) error`

Checks all four invariants for a pair. Returns descriptive error on
violation. Used in tests (every pair-state test asserts validation
after every transition), defensive assertions in transition methods
(debug-build panic; production log + self-heal), and the periodic
consistency checker.

### Pair-aware enforcement at write points

- `TransitionPair(controller, controlled, target, r)` is the ONLY
  supported way to put two characters into the same grapple state.
- Direct `c.Position.TransitionToMount(...)` calls remain available for
  tests but considered low-level (don't enforce pair semantics).
- `TransitionPair` validates input before firing: confirms both
  characters are in compatible source states; rejects on mismatch.
- Per-round tick processes PAIRS (controller's iteration looks up
  partner via GrappleData; single roll updates both sides atomically).
  No code path updates one side without the other.

### Periodic consistency checker

`internal/hooks/Position_ConsistencyCheck.go` runs every
`PositionConsistencyCheckRounds` rounds (default 10 — config knob).
Walks all characters with `IsGrappling()`; for each, looks up partner
and calls `ValidateGrapplePair`. On violation:

1. Force-break BOTH characters to Standing via direct
   `Position.TransitionToStanding` calls (bypassing `TransitionPair`
   since the pair is already invalid).
2. Clear both `GrappleData` slots.
3. Log the violation with character IDs + room + state details + the
   specific invariant violated, at WARN level.
4. Send a generic "the grapple suddenly breaks apart" room message.

Fail-safe code: if design has a bug that creates invalid pair state,
the system recovers rather than getting stuck. The log entry IS the
bug report — production failures surface as engineering tickets, not
stuck players.

### Required tests (added to 4b's Behavior Matrix)

- `TestPairInvariants_FreshGrapple` — `TransitionPair` produces valid
  pair state for each grapple state.
- `TestPairInvariants_PerRoundTick` — repeated per-round ticks preserve
  all four invariants across many drift steps (table-driven over many
  rounds).
- `TestPairInvariants_ThresholdTransition` — when a threshold-triggered
  transition fires, the new pair state is also valid.
- `TestPairInvariants_BothInControlRejected` — explicitly construct a
  "both InControl" pair via direct `TransitionToMount` calls; verify
  validation catches the invariant violation.
- `TestPairInvariants_ChainImpossible` — try to set up A→B→C→A; verify
  the third transition fails because B is already in a grapple state
  (chunk-4a transition table already prevents this; test confirms).
- `TestConsistencyChecker_DetectsAndHeals` — manually construct an
  invalid pair (via test-only direct field manipulation, bypassing
  `TransitionPair`); run the checker; verify both characters end up
  in Standing and the violation is logged.
- `TestConsistencyChecker_NoFalsePositives` — run the checker against
  legitimate pair states (each grapple position); verify no spurious
  force-breaks.

## Behavior Matrix preview (drafted in plan, completed in tests)

~60-80 rows total. Major groups:

- **Per-round drift mechanics (PB-001-015):** roll formula correctness,
  margin → magnitude mapping, stamina penalty curve, encumbrance
  penalty curve, controller-cost vs controlled-cost asymmetry.
- **Initial-state table (PB-016-027):** `InitialControlForPair` returns
  expected values for each state + role.
- **Threshold transitions (PB-028-042):** each `DefaultEscapeTarget`
  triggers correctly when Controlled threshold hit; new state has
  fresh ControlLevels per the initial-state table.
- **Pair invariants (PB-043-052):** the 7 tests listed in Section 4 +
  3 stress tests (many-rounds drift, oscillating control, near-miss
  thresholds).
- **Cutover smoke (PB-053-070):** trip/bash/grapple/stand/recovery/
  spell-knockdown all produce expected Position state changes; flee
  blocker fires on grapple; defense degradation fires on grapple;
  speed multipliers apply.
- **Messaging contract (PB-071-080):** gradient messages fire on
  level transitions; cooldown prevents spam; transition messages fire;
  stamina warning fires once.

## Sunset list (deleted at end of chunk 4b)

- `Character.CombatPosition CombatPosition` field
- `Character.PositionRoundsMin int` field (folded into ProneData /
  SupineData)
- `Character.GrappleControllerId int` field (derived from ControlLevel
  + GrappleData.Partner)
- `ConditionGrappleController` constant + all usages
- `internal/characters/combatposition.go` — legacy enum + helpers
  (`IsGroundPosition`, `IsGrapplePosition`, `GetSpeedMultiplier`,
  `GetPositionColor`, `GetWorstPosition`)
- All ad-hoc `c.CombatPosition == characters.PositionXxx` checks —
  replaced by `c.IsXxx()` predicates from chunk 4a
- All ad-hoc `c.CombatPosition.IsGroundPosition()` /
  `.IsGrapplePosition()` calls — replaced by `c.IsGroundGrapple()` /
  `c.IsGrappling()`
- `internal/combat/grapple.go` `CheckClinchProgression()` and
  `CheckGroundedEscape()` functions (replaced by per-round control
  tick)
- `internal/hooks/NewRound_DoCombat.go:327-378` `ProcessGrappleProgression`
  block (replaced by per-round tick)
- `internal/hooks/Life_Cascades.go:55-57` chunk-2 pre-wire (chunk-4a
  Position cascade observer is now sole owner)

## Persistence

`Character.PerGrappleMessageCooldowns` field is non-persistent
(`yaml:"-"`). Resets to empty on login. Matches existing
position-state non-persistence.

## Open questions / risks

- **Behavior drift during migration window.** Parallel-write means both
  systems are live at once. If a writer is migrated but a reader is
  still on legacy, drift. Mitigation: chunk-3 precedent — strict
  sequence (W → F → R → S); per-step build + test verification; tests
  cover both legacy and new code paths during the window.
- **Per-round drift tuning.** Initial defaults (`StaminaPenaltyMax =
  0.6`, `EncumbrancePenaltyMax = 0.8`, controller/controlled cost
  multipliers 1.0/2.0) are design intuition. Smoke testing in 4f will
  reveal whether grapples last too long / too short. Mitigation: all
  config knobs; 4f balance pass tunes.
- **Threshold-triggered transitions overlap with explicit commands.**
  Currently, the `grapple` command attempts a single roll → state
  change. After 4b, that command attempts an entry transition; the
  per-round tick takes over after. If the grapple command's input
  becomes more ambiguous (e.g., player typing `grapple` mid-grapple
  to "advance" position), we need a clear contract. Mitigation: 4b
  cuts over the existing `grapple` command to mean ONLY entry; the
  per-round tick handles all post-entry mechanics; explicit advance
  is a separate future command (4c/4d).
- **`{pos}` prompt token semantics.** Today shows position name when
  non-Standing. With 14 states, the prompt may need width tuning (e.g.,
  "BackStanding" is longer than "Clinched"). Mitigation: prompt
  rendering already handles variable-width tokens; if visual
  inspection during smoke shows issues, abbreviate display names in
  the prompt context (e.g., "BG" instead of "BackGround").
- **AttemptRecovery being a per-round side-effect AND a transition.**
  Today `AttemptRecovery` is called from `NewRound_UserRoundTick`/
  `NewRound_MobRoundTick` and may set CombatPosition → Standing. With
  the new FSM, the recovery hook fires `Position.TransitionToStanding`
  AND the Position pre-validation (must currently be Prone/Supine).
  Mitigation: hook order — recovery fires first, then per-round
  control-tick processes any remaining grapples. Both can't
  legitimately fire for the same character in the same round
  (Prone/Supine and grappling are mutually exclusive states).
- **Pair-write rollback failure modes.** `TransitionPair` rollback
  uses snapshot/restore. If the snapshot save itself fails or the
  rollback fails, pair is desynced. The periodic consistency checker
  is the final backstop. Mitigation: rollback is a simple field
  assignment (low failure surface); checker covers anything that
  slips through.

## Resumption criteria (chunk 4b done when)

1. `internal/state/position/pair.go` + `control.go` + `validation.go`
   exist; Behavior Matrix tests PB-001 through PB-080ish green.
2. `Position_GrappleTick.go` runs per round; processes grapple pairs;
   updates ControlLevels; fires threshold-triggered transitions.
3. `Position_ConsistencyCheck.go` runs per N rounds; tests verify
   detect + heal + no-false-positives.
4. `Position_Messaging.go` fires gradient + transition + stamina
   messages with per-grapple cooldowns; YAML config loads cleanly.
5. All 8 writer sites cut over (W1-W8). All 6 reader sites cut over
   (R1-R6). Test fixtures migrated (F1).
6. Legacy fields deleted (S1-S5): `CombatPosition` enum + field,
   `PositionRoundsMin`, `GrappleControllerId`, `ConditionGrappleController`,
   `internal/characters/combatposition.go`.
7. 6 new btree primitives registered.
8. End-state combat behavior validated via smoke: grapples now have
   per-round texture (control shifts, messages fire); positions
   transition gradually rather than via single-round binary outcomes;
   stamina drains visibly for the controlled side; encumbrance
   penalties show up; pair invariants hold across many rounds; the
   consistency checker doesn't fire under normal play.
9. Chunks 0/1/2/3/4a regression tests pass; no FAILs across the
   broader test suite.
10. Context.md updates: `internal/state/position/context.md` extended
    with control-axis API + messaging contract + invariants;
    `internal/hooks/context.md` documents the three new observers;
    `internal/characters/context.md` documents the new control-axis
    predicates + sunset notes.
11. `COMBAT_STATE_ROADMAP.md` chunk 4b row marked Done.

## Out-of-scope / future followup candidates

- **Voluntary controller-advancement** (Mount → BackGround via player
  command). 4c/4d/content. Probably ships with 4d submissions since
  they share the "explicit player action against a grappled opponent"
  surface.
- **Per-state asymmetric stamina cost / penalty curves** (Mount-bottom
  drains worse than HalfGuard-bottom). 4f balance pass.
- **Per-position roll formula variants** (Mount Str-heavy, Guard
  Dex-heavy). 4c/4d/4f.
- **State-specific extras on GrappleData** (ClinchGrip, ArmsIsolated,
  HooksIn, TrappedLeg, GuardVariant). 4c/4d.
- **Low-stamina auto-tap** (controlled fighter at 0 stamina
  auto-submits). 4d submission system.
- **N-vs-1 grappling.** Master-spec out-of-scope.
- **Cardio-specific mechanics** (long-fight fatigue beyond per-round
  stamina cost). Master-spec out-of-scope.
- **Knockdown immunity / resistance** flags. Future feature.
- **Player-facing position-info command** ("look grapple" to see
  current pair state, control levels, etc.). Not strictly needed; the
  gradient messages already convey state.
