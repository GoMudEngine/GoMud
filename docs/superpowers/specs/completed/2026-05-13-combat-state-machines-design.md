# Combat State Machines — Systemic Redesign

> **Side quest from mob aliveness chunk 2.7.** A multi-chunk
> systemic redesign of character state in the DOGMud engine.
> Replaces the overloaded `Character.Aggro` field, the scattered
> activity flags (`CastingState`, `CraftingState`, `Charmed`),
> the half-migrated `CombatPosition` enum, the unused Stage 2a
> `NewRound_DoCombat_unified.go` graveyard, and the ad-hoc
> "is this character available?" checks scattered across the
> codebase with six explicit orthogonal state machines plus
> one flag, sharing a small reusable framework.
>
> **Aliveness is paused for the duration.** Chunk 2.7 (mob
> skullduggery suite) Task 19 (roadmap mark-done) is held
> until this work lands. Aliveness resumes after.

## Goal

The combat-state surface of the engine has been "fixed" multiple
times (targeting ≥ 4x, flee ≥ 3x, aggro lifecycle ≥ 3x) without
the root cause being addressed. The root cause: `Character.Aggro`
is overloaded to mean three things at once (current target, in-
combat flag, fleeing sentinel), and combat-adjacent state is
scattered across at least five stores (`Aggro`, `CombatPosition`,
`Conditions` array, `Buffs` system, pointer flags like
`CastingState`) with no unified model and no central predicates.

This redesign:
- **Replaces overloaded fields with explicit, orthogonal state
  machines.** Six machines + one flag, each with documented
  states, transitions, and behavior matrices.
- **Achieves mob/player parity by construction.** Both actors
  run through the same machines with the same transitions.
  Legitimate asymmetries (loot drop on mob death only) become
  documented per-state policies rather than scattered if-
  `IsPlayer` branches.
- **Provides single predicates** for every query that's
  currently ad-hoc (`IsEngaged()`, `IsActing()`, `IsAvailable()`,
  `IsHidden()`, `IsAlive()`).
- **Formalizes interrupt and cancellation rules** as cross-
  machine transitions, replacing the current mix of
  cancel-on-combat buff flags, concentration-break casting
  logic, and scattered Aggro-nil checks.
- **Ships staged via TDD with intent-driven tests.** Each
  chunk has a Behavior Matrix that drives RED-phase tests
  before any implementation begins.

## Non-goals

- **Combat math redesign.** Damage formulas, dice rolls,
  mitigation curves, skill multipliers — all unchanged. This is
  a state-of-character redesign, not a balance redesign.
- **Combat tactics redesign.** Mob archetypes, btree primitives,
  combat command surfaces — all unchanged in behavior. They
  just consume different predicates and call different
  transition methods.
- **New game features.** No new commands, no new mechanics, no
  new buffs. Pure structural work.
- **Eliminating legitimate asymmetries.** Player death →
  graveyard teleport vs mob death → corpse drop stays
  asymmetric (different game-design intents).
- **Cross-cutting features that benefit from the framework.**
  Town justice, bounty hunting, NPC schedules — they're
  *enabled* by this work but they're not part of this work.

## Architectural musts

Brainstorming refined the framing:

1. **Seven orthogonal state machines, one flag.** Each machine
   models a single concern with explicit states and transitions.
   Concerns are orthogonal — a hidden character can be in
   combat or not; a casting character can be prone or standing.
   The state list (final, after pushback; Perception added
   2026-05-13 as chunk 6 after recurring blind/dark-room
   broadcast bugs surfaced during chunks 1-2):

   | # | Type | Name | States |
   |---|------|------|--------|
   | 1 | Machine | Combat Phase | Idle / Engaging / Engaged / Disengaging |
   | 2 | Machine | Activity | Free / Casting / Crafting / Foraging / Salvaging / Tracking / … |
   | 3 | Machine | Position | Standing / Prone / Clinched / Grounded |
   | 4 | Machine | Awareness | Visible / Concealing / Hidden / Revealing |
   | 5 | Machine | Life | Alive / Dead / Respawning |
   | 6 | Machine | Presence | Player: Connecting / Active / Idle / AFK / Disconnected • Mob: Spawning / Active / Dormant / Despawning |
   | 7 | Machine | Perception | Sighted / Blinded |
   | 8 | Flag | Combatant | Combatant / NonCombatant |

   Charm stays a struct field on Character (uses the
   framework's scheduled-transition machinery for rebellion
   checks, but isn't its own machine).

2. **Mob/player parity by construction.** The same machines
   apply to both. Asymmetries become explicit per-actor
   transition policies, never scattered `IsPlayer()` branches.
   Presence is the one machine with different state lists per
   actor (player has AFK/Disconnected; mob has Dormant/
   Despawning) — even there the framework supports the
   polymorphism cleanly without bifurcating other concerns.

3. **Framework-with-Combat-Phase as a single chunk.** The
   framework is born co-developed with the highest-pain
   machine so its API is shaped by real needs, not invented
   abstractly. The `NewRound_DoCombat_unified.go` graveyard is
   direct evidence that big-bang abstract designs for combat
   code stall before landing.

4. **Per-machine staged migration.** Each machine ships in its
   own chunk, end-to-end (spec → tests → implementation →
   caller migration → sunset of old fields). Bounded blast
   radius, independent smoke validation, regression bisection
   stays cheap. Six chunks total.

5. **Aliveness paused for the duration.** Chunk 2.7 Task 19
   held; chunks 2.8+ not started. The aliveness substrate
   depends on stable combat-state semantics — building on
   shifting ground produced the chunk 2.7 thief-archetype bug.
   Power through.

6. **Intent-driven TDD, not parity TDD.** Every chunk's spec
   includes a Behavior Matrix enumerating intended
   `(state, trigger, conditions) → outcome (+ rationale)`
   rules. The matrix IS the test set — every row becomes a
   RED-phase test before implementation. Parity tests would
   re-encode current buggy behavior; intent tests catch design
   drift. This is the discipline that breaks the "fixed 4x"
   pattern.

7. **Machine coordination is local with documented cross-
   machine effects.** Each machine's transitions are local;
   cross-machine effects are documented as `OnEnter` / `OnExit`
   callbacks. Combat Phase entering `Engaging` triggers
   Awareness `Hidden → Revealing`; Activity entering `Casting`
   blocks Combat Phase `Idle → Engaging`. The interaction
   matrix is a documented set of cross-machine rules attached
   to specific transitions.

8. **Scheduled transitions are a framework primitive.**
   Awareness's `Revealing → Visible` delay, Charm's rebellion
   check, future Activity `Casting → Free` on completion — all
   share the same scheduling infrastructure. No per-machine
   reinvention.

9. **Charm field uses framework scheduled transitions but
   doesn't become its own machine.** Pragmatic compromise:
   charm has real internal state (duration, intensity,
   rebellion curve) but a 3-state machine for one purpose
   (modeling Rebelling) bloats the design. Framework supports
   the scheduling generically; charm consumes it as a field.

10. **Sunset is part of each chunk.** Old fields (e.g.,
    `Aggro`, `CastingState`) are deleted at the end of their
    machine's chunk, not deferred. No coexistence beyond the
    migration window. The Stage 2a graveyard exists because
    coexistence was indefinite — don't repeat.

## Framework design

A small package `internal/state/` provides the reusable
infrastructure all six machines build on. Pre-spec sketch (will
firm up during chunk 0 implementation):

### Core types

```go
package state

// Machine is a finite state machine instance bound to a
// Character. Each Character has one Machine per concern
// (CombatPhase, Activity, Position, Awareness, Life, Presence).
type Machine[S comparable] interface {
    State() S
    CanTransition(to S) bool
    TransitionTo(to S, reason TransitionReason) error
    On(event Event, handler Handler[S])
    ScheduleTransition(to S, after time.Duration, reason TransitionReason)
}

// TransitionReason captures why a transition fired —
// debuggable, observable, used by quest engine + aliveness
// substrate subscribers.
type TransitionReason struct {
    Trigger string  // "attack_command", "flee_success", "buff_expired", etc.
    Actor   ActorRef
    Target  ActorRef // optional
    Metadata map[string]any
}

// Event is fired before/after every transition. Subscribers
// can veto (BeforeTransition) or react (AfterTransition).
type Event struct {
    Machine string
    From    string
    To      string
    Reason  TransitionReason
}
```

### Key framework capabilities

- **Local transition logic.** Each machine declares its valid
  transitions; the framework enforces them.
- **Cross-machine rules.** A machine can subscribe to other
  machines' transitions via `On(event, handler)`. Combat Phase
  entering `Engaging` triggers Awareness's
  `Visible/Concealing/Hidden → Revealing` handler.
- **Veto pattern.** `BeforeTransition` handlers can return
  veto. Example: Activity vetoes `Combat Phase → Engaging`
  while in `Casting`.
- **Scheduled transitions.** `ScheduleTransition(to, after,
  reason)` registers a deferred transition (Awareness's
  Revealing→Visible delay, Charm rebellion check). Persists
  across server restarts where applicable.
- **Observable.** Quest engine, faction system, opinion store,
  knowledge system all subscribe to transitions via `On(event,
  handler)`. Replaces the current ad-hoc `events.AddToQueue`
  + manual dispatch scattered across hooks.
- **Persistence.** Each machine declares which states persist
  across server restart. Combat Phase = always Idle on restart;
  Life = preserved (a Dead character stays Dead pending
  Respawning); Position = always Standing on restart.

## The six machines + one flag

### 1. Combat Phase

**States:** `Idle | Engaging | Engaged | Disengaging`

**Concerns:** target tracking, "am I in combat?" predicate
(`IsEngaged() = State() == Engaged`), entering and leaving
combat, the marquee bug fix.

**Replaces:** `Character.Aggro` field entirely. Target becomes a
data field on the Engaged state (e.g.,
`combatPhase.Target() ActorRef`). Aggro's `Type` enum (DefaultAttack,
SurpriseAttack, Shooting, SpellCast, Flee) becomes a transition
reason metadata field, not a state field.

**Key transitions:**
- `Idle → Engaging` (attack command, surprise ambush, btree
  engage action). Conditions: Combatant flag = Combatant,
  Activity = Free, Life = Alive, target is valid.
- `Engaging → Engaged` (first swing resolves, round tick
  consumes the wait).
- `Engaged → Disengaging` (flee command, target lost, target
  dead, room change). Per-actor: player movement requires
  Disengaging-then-Idle; mob movement allowed during
  Engaged with cross-room target retained.
- `Disengaging → Idle` (flee succeeded, target re-acquired
  → back to Engaged).
- `Engaging/Engaged → Idle` direct (target died, Life=Dead
  on self, despawn).

**Behavior Matrix:** Authored during Chunk 0 spec/plan with
~30-50 intent rows covering every meaningful (start state,
trigger, condition) combination.

**Interaction with other machines:**
- Entering Engaging triggers Awareness `Hidden → Revealing`
  (unless trigger is SurpriseAttack — preserves stealth for one
  round).
- Blocked by Activity ∈ {Casting, Crafting, Foraging} (those
  must transition to Free first).
- Blocked by Life ∈ {Dead, Respawning}.
- Blocked by Combatant=NonCombatant (the flag).
- Entering Engaged forces Buffs with `cancel-on-combat` flag to
  cancel (replaces current `CancelCombatBuffs()` call site).
- Disengaging via flee requires Position=Standing (can't flee
  while Clinched/Grounded — break grapple first).

### 2. Activity

**States:** `Free | Casting | Crafting | Foraging | Salvaging | Tracking | …`

**Concerns:** "is this character locked into a multi-round
action?", interrupt handling, the "you can't do X while you're
doing Y" rules.

**Replaces:** `CastingState`, `CraftingState` pointer fields.
Per-activity data (cast progress, recipe being crafted, forage
target tag) becomes data attached to the state, not separate
fields.

**Key transitions:**
- `Free → Casting/Crafting/Foraging/Salvaging/...` on activity
  start command. Each activity has its own start gate.
- `Casting/Crafting/... → Free` on activity completion.
- `Casting → Free` on interrupt (damage > concentration
  threshold, movement, cancel command). **Interrupt rules
  formalized**: each activity declares its interrupt triggers
  in the framework.
- `Crafting → Free` on movement (current behavior; preserved).

**Interaction:**
- Activity ≠ Free vetoes Combat Phase `Idle → Engaging`
  (entering combat from a non-free activity not allowed
  except via interrupt).
- Activity transitions to Free can be triggered by Combat
  Phase `Idle → Engaging` interrupt-promotion (the
  combat-causes-interrupt path).
- Life=Dead forces Activity=Free on next tick.

### 3. Position

**States (expanded 2026-05-16 to a full BJJ/MMA position taxonomy
— see "Position rich-grapple expansion" below for rationale and
sub-chunk breakdown):**

```
Standing | Prone | Clinch | BackStanding |
Mount | SideControl | KneeOnBelly | NorthSouth | Crucifix | BackGround |
HalfGuard | Guard | Turtle
```

13 states total — 2 non-grapple (Standing, Prone), 2 standing-pair
(Clinch, BackStanding), 6 ground-top-dominant (Mount, SideControl,
KneeOnBelly, NorthSouth, Crucifix, BackGround), 1 ground-transitional
(HalfGuard), 1 ground-bottom-active (Guard), 1 ground-defensive
(Turtle).

**Concerns:** body position in combat, grapple state (full BJJ/MMA
position taxonomy), control gradient between paired grapplers,
weapon utility per position, submission setup, recovery mechanics,
third-party-interaction asymmetries.

**Replaces:** the half-migrated `CombatPosition` enum + the
deprecated `Prone` bool + `PositionRoundsMin` recovery counter +
`GrappleControllerId` field + `ConditionGrappleController` side-
channel. Recovery rules become transition cooldowns on Standing-
from-Prone transitions; controller role becomes a derived property
of the per-state control level instead of an explicit flag.

**Key axes (orthogonal to FSM state):**

- **Control axis** — per-grappler 5-level scale: `InControl |
  LosingControl | Neutral | BecomingControlled | Controlled`.
  Stored as per-state data on grapple states. Each round, opposed
  rolls shift control; when control crosses thresholds, position
  transitions trigger.
- **Weapon utility** — `(Position × WeaponType) → modifier`
  content table. Drives damage / hit math.
- **Submission availability** — `(Position, ControlLevel)` tuple
  gates which submissions can be attempted. Auto/opportunistic
  rather than explicit command (rework of current submission
  special-attack command in 4d).
- **Third-party interaction** — both grapplers have degraded
  defense vs outside attackers (controlled is severely degraded,
  controller is moderately degraded — asymmetry maps to control
  axis). Outside damage to controller degrades control level.

**Key transitions (geometric only; control-axis transitions live
in 4b):**

| From | To | Trigger |
|---|---|---|
| Standing | Prone | knockdown (bash / trip / spell knockdown) |
| Prone | Standing | recovery roll / explicit `stand` (stamina cost) |
| Standing | Clinch | grapple entry roll |
| Clinch | Standing | grapple break |
| Clinch | BackStanding | back-take from clinch |
| Clinch | Mount / SideControl / NorthSouth / Guard / BackGround | takedown variants |
| BackStanding | Standing | break |
| BackStanding | BackGround | back-controller pulls down |
| BackStanding | Clinch | controlled turns to face |
| Mount ↔ SideControl ↔ KneeOnBelly ↔ NorthSouth | each other | controller transitions (control-roll-gated) |
| Mount / SideControl / NorthSouth | Crucifix | controller isolates arms |
| Mount / SideControl / NorthSouth / Crucifix / BackGround | HalfGuard / Guard | controlled escapes (control-roll-gated) |
| Mount / SideControl / NorthSouth | BackGround | controlled rolls, controller takes back |
| HalfGuard ↔ Guard | each other | top passes / bottom recovers |
| Guard / HalfGuard / Mount-bottom | Standing | bottom escapes + stands |
| any-ground | Turtle | controlled curls defensively (failed escape) |
| Turtle | Standing | controlled stands |
| Turtle | BackGround | controller hooks in |
| Prone | Mount / SideControl / NorthSouth / BackGround | another character mounts the prone target |

Full transition graph + Behavior Matrix authored in chunk-4a spec.

**Interaction with other machines:**

- Position ∈ {grapple states} blocks Combat Phase
  `Engaged → Disengaging` via flee (existing rule, preserved).
- Position = Prone gates certain commands (stomp variant
  selector — already wired).
- Position ∈ {grapple top-dominant} unlocks position-gated
  attacks (knee / cross-face / ground-and-pound — new in 4c).
- Life=Dead forces Position=Standing on next tick (chunk-2
  pre-wire repointed onto Position-machine observer in 4a).
- Combat Phase Idle→Engaging veto: Position must be Standing
  OR a grapple top-dominant state (Mount/SC/KOB/etc.) where the
  controller can choose to enter combat with a third party.

### 3a. Position rich-grapple expansion (added 2026-05-16)

Chunk 4 was originally specced for 4 positions (Standing / Prone /
Clinched / Grounded). On 2026-05-16 the scope expanded to a full
BJJ/MMA position taxonomy (13 states) plus an orthogonal control
axis, weapon-utility table, submission system, and third-party
interaction mechanics. Rationale: the original 4-state model
gives correct structure but loses the tactical richness of real
grappling — different ground positions have meaningfully different
submission opportunities, weapon utility, escape difficulty, and
defense-asymmetry vs third parties. The richer system enables the
"very real tactical decisions" the user identified as the design
intent.

Because the expanded scope is roughly 4× the size of the prior
chunks, it ships as 6 sub-chunks each sized like a chunk-3 sibling:

| Sub-chunk | Scope | Size |
|---|---|---|
| **4a** | Position FSM: 13 geometric states, transitions, basic per-state data (incl. `ControlLevel` field default Neutral), migration from `CombatPosition` enum, btree primitives for position queries. No per-round control rolls yet — control axis exists as a data slot only. | M-L |
| **4b** | Control-axis mechanics: per-round opposed rolls, threshold-triggered position transitions, gradient messaging ("you feel your mount slipping"). | M |
| **4c** | Weapon-utility-by-position table: (Position × WeaponType) → damage/hit modifier YAML, combat resolution reads it. | S-M |
| **4d** | Submission system: rework or sunset the existing `submission` special-attack command in favor of automatic / opportunistic submissions gated on (Position, ControlLevel). Submission outcomes (choked out, damaged limb, tap-out, continue) authored. | M-L |
| **4e** | Third-party interaction: symmetric defense degradation (controller moderately, controlled severely), grappler offense restrictions vs third party, outside-damage→control-axis degradation, mob AI bias toward attacking grappled enemies, submission-interrupt risk. | M |
| **4f** | Balance pass + position flavor text + smoke against full-stack combat scenarios. | M |

**Out of scope across all of chunk 4 (logged as future followups
if pulled):**
- N-vs-1 grappling (mob B physically joins an existing grapple
  to make it 2-on-1). Doesn't exist in current code; real but
  rare; significant complexity in the per-state data shape.
- Cardio / fatigue effects specific to grapple duration. Existing
  stamina system handles general tiredness; layering grapple-
  specific cardio is bonus content.
- Clinch-grip granularity as distinct FSM states (Thai plum,
  50/50, over-under, double-underhooks, single-collar). Modeled
  as per-state `ClinchGrip` data field instead — drives modifiers
  but isn't a separate FSM state. Promotes to distinct states
  only if grip transitions reveal the data-driven model is too
  loose.

### 4. Awareness

**States:** `Visible | Concealing | Hidden | Revealing`

**Concerns:** stealth, the cancel-on-combat rule for Hidden
buff, the "ambusher mid-attack" representation.

**Replaces:** the current Hidden buff (#9) with `cancel-on-combat`
flag as the sole source of "I am hidden." Buff system continues
to provide the *effect* (sneak bonuses, visual concealment) but
the *state* lives in the Awareness machine.

**Key transitions:**
- `Visible → Concealing` (sneak command initiated).
- `Concealing → Hidden` (sneak roll succeeded — synchronous
  today, room to model multi-round concealment later).
- `Concealing → Visible` (sneak roll failed; observer
  detected).
- `Hidden → Revealing` (Combat Phase entered, except
  SurpriseAttack reason; movement detected; explicit
  reveal).
- `Revealing → Visible` (scheduled transition after one round
  for atomicity — "lifts gold from pocket" message + buff
  cancel resolve in the same frame).

**Interaction:**
- Combat Phase → Engaging cascades Awareness → Revealing
  (the canonical "stealth breaks on attack" rule, but with
  the SurpriseAttack exemption).
- Activity ≠ Free blocks Visible → Concealing (can't sneak
  while crafting).

### 5. Life

**States:** `Alive | Dead | Respawning`

**Concerns:** death/respawn flow, state cleanup on death,
respawn grace period, the 2026-04-21 respawn-loop bug class.

**Replaces:** the scattered death-cleanup logic in `suicide.go`
(player) and `MobDeath_*` hooks (mob). Cleanup becomes a single
`OnEnterDead` handler that cascades to every other machine.

**Key transitions:**
- `Alive → Dead` (health drops to 0, killed command,
  permadeath). Triggers:
  - Combat Phase → Idle (clear all aggro, all inbound)
  - Activity → Free
  - Position → Standing
  - Awareness → Visible
  - All buffs with `cancel-on-combat` flag canceled
  - Loot drop (mob only — legitimate asymmetry)
  - Aliveness substrate notifications (MobDeath event,
    faction rep, crime resolution, etc.)
- `Dead → Respawning` (after respawn delay; player teleport
  to graveyard begins; mob instance scheduled for cleanup).
- `Respawning → Alive` (player at graveyard with grace buff;
  mob new instance spawned via Presence).

### 6. Presence

**States** (per-actor polymorphism):
- Player: `Connecting | Active | Idle | AFK | Disconnected`
- Mob: `Spawning | Active | Dormant | Despawning`

**Concerns:** "is this character meaningfully present in the
game world?", AFK / disconnect handling, mob hibernation/idle-
zone optimization.

**Replaces:** scattered AFK checks, `BoredomCounter`,
`WanderCount`, the idle-zone shortcut in `handleMobCombat`.

**Key transitions:** activity-based timeouts (Player: command in
last N rounds → Active; otherwise Idle, AFK, Disconnected).
Mob equivalents tied to nearby-player presence and zone
activity.

**Interaction:**
- Presence ∈ {Idle, AFK, Disconnected, Dormant} should
  generally prevent Combat Phase transitions to Engaging
  (the survey's "AFK can take aggro" bug fix). Exception: if
  attacked, the character transitions back to Active first,
  then accepts the aggro.
- Presence = Despawning / Disconnected cancels all pending
  scheduled transitions.

### 7. Perception

**States:** `Sighted | Blinded`

**Concerns:** "can this observer see room events?", the recurring
whack-a-mole bug class where messages leak through to blind or
dark-room players. Latest exemplar (2026-05-13): a player's own
companion casts a spell and the message renders with the
companion's name, despite the owner being in a dark room. Same
class of bug has surfaced repeatedly: combat broadcasts ignoring
observer state, mob actions visible despite blindness, room-
broadcast text not gated by observer night-vision.

**Replaces:** the scattered ad-hoc visibility/blindness checks at
each room-broadcast site. The current `room.SendTextVisual(text,
excludeUserId)` family assumes every non-excluded recipient can
"see" the broadcast; Perception centralizes the observer-side
gate so the check lives in one helper instead of N broadcast
sites.

**Key transitions:**
- `Sighted → Blinded` (blindness condition / debuff buff /
  spell effect applied)
- `Blinded → Sighted` (effect expires, cure cast, condition
  cleared)

Two states minimum. Dimsighted / partial-vision deferred (YAGNI —
no current game mechanic pulls for an intermediate state). Add
later if a "blurred vision" or "partial blindness" effect needs
distinct semantics.

**Observer + actor + environment composition:**

Perception is the *observer*-side complement to Awareness (which
is *actor*-side). The composite "does this observer perceive this
actor's action" relationship is:

```
SEEING(observer, actor, room) :=
    observer.IsSighted()                              // Perception (this chunk)
      && actor.IsVisibleTo(observer)                  // Awareness (chunk 1)
      && (room.IsLit() || observer.HasNightVision())  // environment + buff
```

Room-broadcast helpers funnel through a single perceiver-aware
dispatcher. Each potential recipient's Perception is consulted;
Blinded observers receive either fallback text ("you hear
someone casting" / "you sense movement nearby") or full
suppression, gated by broadcast tag. Default policy by tag:

| Broadcast tag | Sighted | Blinded |
|---|---|---|
| Speech (say / shout) | full text | full text (audible) |
| Combat hit/miss | full text | fallback ("you hear blows landing nearby") |
| Spell cast | full text | fallback ("you hear an incantation") |
| Movement (enter/leave) | full text | fallback ("you hear footsteps") |
| Item drop/pickup | full text | fallback ("you hear something clatter") |
| Atmospheric description | full text | suppress |

Final per-tag matrix authored during chunk 6 spec.

**Cross-machine rules:**
- `Life: → Dead` does NOT transition Perception. Dead characters
  route through the Shadow Realm room separation, not via
  blindness suppression.
- `Presence: Spawning/Respawning → Active` starts Sighted (fresh
  state on entry).
- No machine vetoes Perception transitions — blindness is an
  externally-applied effect, not a player-elected state.
- Awareness queries (`IsVisibleTo`) and Perception queries
  (`IsSighted`) compose at broadcast time, not at transition time.
  Neither machine vetoes the other.

**Sunset:**
- Ad-hoc `c.HasCondition(ConditionBlinded)` checks at broadcast sites
- Any `IsBlind()` predicate scattered across hooks / usercommands
- `room.SendTextVisual` callers audited and routed through the
  perceiver-aware dispatcher (speech / combat / spell / movement
  / item / atmosphere — one tag per call site)

**Behavior Matrix:** ~20-30 rows authored during chunk 6 spec.
Coverage:
- Sighted → Blinded transitions on each effect source
- Blinded → Sighted on each cure / expiry path
- Per-tag broadcast contract (suppress vs fallback)
- Companion-by-name leak regression: blind observer + companion
  cast in same room → fallback text, not "<companion> casts X"
- Multi-observer room broadcast with mixed Perception states →
  each observer sees the appropriate variant
- Interaction with NightVision and dark-room: blind observer
  with NightVision is still blind (Perception trumps light)
- Self-perception: the actor of an action always perceives their
  own action regardless of blind state (UI consistency)

**Why chunk 6 (rationale for the addition):** Awareness (chunk 1)
established actor-side visibility; the operational bugs that
remain (and keep recurring after each spot-fix) are observer-
side gaps. Centralization is the unblock. Slots cleanly after
Presence because no earlier chunk depends on it, and the
recurring nature of the bug class argues for a structural fix
rather than continued whack-a-mole.

### 8. Combatant (Flag)

**Values:** `Combatant | NonCombatant`

**Concerns:** "is this character available as a combat target
right now?", the passivity-aura use case, the mob-non-combatant
auto-clear pattern.

**Replaces:** `mob.IsNonCombatant()` (currently mob-only flag in
YAML). Promotes to a runtime-toggleable flag on Character,
making the future player passivity spell trivial.

**Effects:** NonCombatant flag vetoes:
- Inbound Combat Phase Idle → Engaging from other actors
- Outbound Combat Phase Idle → Engaging from self
- (i.e., NonCombatants can neither attack nor be attacked
  without an explicit override path)

## Cross-machine interaction matrix

A reference table of the most important cross-machine rules.
Authoritative version lives in each chunk's per-machine spec;
this is the bird's-eye view.

| Trigger (machine: transition) | Effect (other machine) | Rationale |
|---|---|---|
| Combat Phase: → Engaging | Awareness: Hidden → Revealing (unless SurpriseAttack) | Canonical "attack breaks stealth" |
| Combat Phase: → Engaging | Activity must = Free (veto otherwise) | "Can't start a fight while casting" |
| Combat Phase: → Engaging | Combatant must = Combatant (veto NonCombatant) | Passivity respected |
| Combat Phase: → Engaging | Life must = Alive (veto Dead/Respawning) | Obvious |
| Combat Phase: → Engaging | Presence must = Active (veto AFK/Idle for target) | "Don't attack AFK players" |
| Combat Phase: Engaged → Disengaging via flee | Position must = Standing (veto Clinched/Grounded) | Break grapple first |
| Activity: → Casting | Position should allow (Clinched/Grounded penalize?) | Combat-flavor choice |
| Activity: → Free (interrupt) | Triggered by Combat Phase entering Engaging *if* Activity was Casting | Concentration-break rule formalized |
| Life: → Dead | Cascade clear: Combat Phase → Idle, Activity → Free, Position → Standing, Awareness → Visible, all inbound aggro cleared, loot drop (mob), graveyard teleport (player) | Death is a hard reset of all state |
| Position: → Clinched | Other actor's Position: → Clinched (grapple is symmetric) | Grapple involves two |
| Awareness: → Hidden | Combat Phase next Engaging transition uses SurpriseAttack reason | Hidden attack = surprise crit setup |
| Presence: → Disconnected/Dormant | Scheduled transitions paused, not canceled | Resume on reconnect |
| Combatant: → NonCombatant | Combat Phase: any → Idle (forced); all inbound aggro cleared | Pacifism is immediate |
| Any: room broadcast (visual tags) | Perception consulted per observer; Blinded → suppress or fallback per tag | Centralized observer-side visibility gating (chunk 6) |
| Awareness: actor.IsVisibleTo(observer) | Composed with observer.IsSighted() and room.IsLit() to produce SEEING | Actor + observer + environment combine at broadcast time |
| Presence: Spawning/Respawning → Active | Perception → Sighted (fresh start) | Initial state on entry |

## Migration strategy

**Per chunk, formal red/green/refactor, intent-driven TDD:**

1. **Behavior Matrix in spec.** Spec PR includes the per-machine
   Behavior Matrix with every intended rule. User reviews.
2. **RED phase.** Tests written from Behavior Matrix rows. All
   tests fail (machine doesn't exist yet, or transitions return
   default).
3. **GREEN phase.** Minimum implementation to pass tests.
   Framework primitives added/extended as needed for the
   machine's requirements. Caller migration begins:
   readers updated to query new predicates (`IsEngaged()`,
   `IsHidden()`, etc.), writers updated to call transition
   methods.
4. **REFACTOR phase.** Code cleanup while tests stay green.
   Cross-machine interaction handlers wired up.
5. **In-game smoke.** Boot server, run scenarios from the
   Behavior Matrix that can't be unit-tested (multi-character
   interactions, btree-driven flows).
6. **Sunset.** Old field deleted (e.g., `Character.Aggro`
   removed after Combat Phase migration). Any caller still
   referencing the old field is a compile error.

**Per-chunk smoke scenarios** are part of the spec, not
discovered during smoke.

## Per-chunk breakdown

| Chunk | Title | Size | Depends on | Status |
|-------|-------|------|-----------|--------|
| 0 | State machine framework + Combat Phase | XL | — | Done (2026-05-13) |
| 1 | Awareness machine | M | 0 | Done (2026-05-15) |
| 2 | Life machine | M | 0 | Done (2026-05-13) |
| 3 | Activity machine | L | 0 | Done (2026-05-15) |
| 4a | Position machine — FSM + 13 states + migration | M-L | 0, 1 | Not started |
| 4b | Position — control-axis mechanics | M | 4a | Not started |
| 4c | Position — weapon-utility-by-position table | S-M | 4a | Not started |
| 4d | Position — submission system (rework/sunset existing command) | M-L | 4b | Not started |
| 4e | Position — third-party interaction asymmetries | M | 4a, 4b | Not started |
| 4f | Position — balance pass + flavor text + full-stack smoke | M | 4a-4e | Not started |
| 5 | Presence machine | M | 0 | Not started |
| 6 | Perception machine | S-M | 0, 1 (composes with Awareness) | Not started |

Total estimated effort: 12-15 weeks (chunk 4 expanded 2026-05-16
into 6 sub-chunks for the rich-grapple system — see section "3a.
Position rich-grapple expansion" for rationale). Aliveness pauses
for the duration; resumes after chunk 6 lands.

**Note on chunk 6 ordering:** Chunk 6 has no hard dependency on
chunks 3-5 and could ship earlier if the blind/dark-room broadcast
bugs become blocking. Current plan keeps chronological numbering;
revisit if operational pain escalates.

**Note on chunk 4 sub-chunk ordering:** 4a is the architectural
prerequisite for everything else (it ships the FSM that 4b-4e
build on). After 4a lands, 4b-4e have weaker ordering — 4c
(weapon utility) and 4e (third party) could potentially ship
in parallel with 4b (control axis), but for simplicity the
canonical order is sequential. 4f is always last (depends on
all prior sub-chunks for balance / smoke surface).

## Behavior Matrix template

Each chunk's spec includes a matrix with these columns:

| ID | Start state | Trigger | Conditions | Outcome | Rationale | Test name |
|----|-------------|---------|-----------|---------|-----------|-----------|
| CP-001 | Idle | `attack` command | target valid, Activity=Free, Combatant=Combatant, Life=Alive | → Engaging | Standard combat entry | TestCombatPhase_AttackFromIdle |
| CP-002 | Engaged | target dies | target.Life → Dead | → Idle (Disengaging skipped) | Target dying ends combat instantly | TestCombatPhase_TargetDies |
| ... | ... | ... | ... | ... | ... | ... |

The `Rationale` column is the discipline-enforcer: future
contributors changing a rule must justify the change against
the rationale, not just "make the test pass."

## Sunset list (post-chunk-5)

- `Character.Aggro *Aggro` field
- `Character.CastingState *CastingState` field (folded into Activity)
- `Character.CraftingState *CraftingState` field (folded into Activity)
- `Character.CombatPosition` enum (folded into Position)
- `Character.PositionRoundsMin int` field (folded into Position transition cooldowns)
- `Character.GrappleControllerId int` field (folded into Position)
- `Character.Conditions []CombatCondition` array — partially folded (per-effect; some stay as buffs, some become Position/Activity states)
- `Character.NoCombat` / similar flag-buffs — folded into Combatant flag where appropriate
- `mob.Hostile` field — folded into Combatant flag + Combat Phase entry rules
- `mob.IsNonCombatant()` method — replaced by Combatant flag query
- `internal/hooks/NewRound_DoCombat_unified.go` (Stage 2a graveyard) — deleted; the unified handler is now the production path via Combat Phase
- `internal/usercommands/stealth_detection.go` (already deleted in chunk 2.7 task 16)
- All ad-hoc `Aggro != nil` checks — replaced by `IsEngaged()`
- All ad-hoc `CastingState != nil` / `CraftingState != nil` checks — replaced by `Activity() == Free` or similar

## Open questions / known risks

- **Framework API generality.** Born co-developed with Combat
  Phase; will need adjustment as Activity (which has many
  states) and Presence (which has actor-polymorphic states)
  land. Chunk 0's framework is a v1 — chunks 1-5 may refine
  it. Each refinement is a small change to a stable substrate,
  not a redesign.

- **Aliveness substrate consumers.** Quest engine, opinion
  store, faction rep, crime log, knowledge model all listen
  for combat events today. Migration must preserve every event
  they currently consume. Audit during chunk 0 spec.

- **Btree event integration.** Behavior trees fire
  `mob_combat_round`, `mob_idle`, `mob_hurt`, `player_enter`
  events that are loosely tied to Aggro state. Chunk 0 must
  define how state transitions fire btree events (or vice
  versa) consistently.

- **Persistence across restarts.** Which machine states
  survive? Combat Phase: always Idle on boot (sanest reset).
  Life: persists (Dead/Respawning characters stay Dead). Other
  machines: TBD per machine, declared in their spec.

- **Performance.** Each transition fires events; subscribers
  run. A combat round with many participants fires many
  transitions. Profile during chunk 0; cap subscriber count if
  needed.

- **Migration window bugs.** Each chunk's GREEN-to-sunset
  window has old fields and new machines coexisting briefly.
  Tests must explicitly cover the coexistence period (a
  smoke-mode option that runs both old + new paths in
  parallel and asserts they agree).

## Roadmap impact

- **Chunk 2.7 (mob skullduggery suite) Task 19 held.** Roadmap
  marks 2.7 as "In progress" pending state-machines work +
  re-smoke.
- **Aliveness chunks 2.8+ deferred.** Roadmap explicitly notes
  these are paused until state machines complete.
- **New top-level work item** added to MEMORY.md / roadmap:
  "Combat State Machines — systemic redesign (6 chunks)."

## Followup candidates (post-chunk-6)

- **Shared ability cooldown.** rally / warcry / taunt / special
  attacks / casts all share one cooldown timer. Two states
  (Ready / OnCooldown) fits the framework cleanly, but no cross-
  machine consumer (no quest/faction/AI subscriber wants to react
  to "cooldown started"). Lean toward a `Character.AbilityCooldown`
  helper rather than a full machine. Revisit if cross-cutting
  consumers surface during chunks 3-6.
- **Deafened state on Perception.** Adds an audio-broadcast gate
  parallel to the visual gate. YAGNI for now (no current
  mechanic pulls for it; the audio-broadcast surface is much
  smaller than visual). Add as a Perception sub-state or split
  into an Audition machine if a "deafen" effect lands.

## Resumption criteria

State machines work is complete when:
1. All seven machine chunks landed with their Behavior Matrices
   green.
2. Chunk 2.7 smoke re-runs cleanly against the new substrate
   (thief archetype behaves correctly: hides, attempts steal,
   flees or re-stealths; never enters combat unless attacked
   or power-overmatched).
3. Sunset list complete: every old field deleted, every
   ad-hoc predicate replaced.
4. Aliveness chunk 2.7 Task 19 (roadmap mark-done) lands.

Aliveness resumes with chunk 2.8 (mob scout/track/scan) on a
known-stable substrate.
