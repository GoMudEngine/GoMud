# Combat State Machines — Chunk 0: Framework + Combat Phase

> **First chunk of the combat-state-machines redesign**
> (master spec: `2026-05-13-combat-state-machines-design.md`).
> Co-develops the `internal/state/` framework package with
> the first consumer (Combat Phase). Replaces
> `Character.Aggro` field, the Stage 2a unused
> `NewRound_DoCombat_unified.go`, and ~30 ad-hoc `Aggro != nil`
> checks. Marquee bug fix for the chunk 2.7 thief-archetype
> issue. Aliveness paused for the duration.

## Goal

Build the state-machine framework born co-developed with
Combat Phase — the machine that replaces `Character.Aggro`.
After this chunk:

- `internal/state/` package exists with reusable infrastructure
  (Machine, transitions, vetoes, cascades, observers, scheduled
  transitions, tick events).
- `Character.CombatPhase *state.Machine[CombatState]` is the
  source of truth for "who am I attacking?" and "am I in
  combat?".
- `Character.Aggro` field is **deleted** at chunk end. Every
  caller migrated. Old `SetAggro` / `EndAggro` / `Aggro != nil`
  references are compile errors after this chunk.
- The chunk-2.7 thief-archetype bug is structurally impossible
  — "set a target without entering combat" is no longer
  expressible because target lives on the `Engaged` state, and
  entering Engaged requires the full transition with all
  vetoes/cascades firing.
- Multi-attacker scenarios have first-class
  `cp.Attackers() []ActorRef` queries; companion auto-assist
  and retarget-on-death become trivial.

## Non-goals

- Other machines (Activity, Position, Awareness, Life,
  Presence). They consume the framework in chunks 1-5.
- Btree event vocabulary changes beyond adding transition
  events for Combat Phase (`mob_engaging`, `mob_disengaging`,
  `mob_combat_ended`). Existing events
  (`mob_combat_round`, `mob_idle`, `mob_hurt`, `player_enter`)
  keep their names and semantics.
- Combat math redesign.
- Aliveness substrate redesign (faction/opinion/knowledge
  hooks keep their current shape; they just subscribe to new
  events instead of reading `Aggro != nil`).

## Architectural musts

Confirmed during brainstorm:

1. **Generics-based framework** (`internal/state/Machine[S comparable]`).
   Type-safe transitions; typos become compile errors. Modern
   Go, idiomatic for this codebase.

2. **Parameterized states carry data.** `Engaged` carries the
   target; `Idle` is empty. Target is queried as
   `cp.Engaged().Target()` returning typed ActorRef. There is
   no "I have a target but not engaged" representation — non-
   combat target picking uses a different mechanism entirely
   (action parameters, not state).

3. **Veto + Cascade hooks, synchronous.** Two hook types:
   `BeforeTransition` (vetoes; return error to block) and
   `AfterTransition` (cascades; fire transitions on other
   machines or btree events). Hooks run in one stack frame.
   Cascade chains are visible in stack traces. Plus
   `Subscribe(handler)` for pure-observer subscribers (quest
   engine, aliveness substrate).

4. **Bidirectional multi-attacker tracking.** When A enters
   `Engaged{target: B}`, B's Combat Phase auto-tracks A in an
   `Attackers []ActorRef` list. Framework-maintained. Single
   `cp.Attackers()` query replaces every "scan room for
   attackers" loop currently in the codebase.

5. **Btree events: tick + transition.** Round driver fires
   `mob_combat_round` for Engaged, `mob_idle` for Idle (tick
   events; vocabulary preserved). Transition cascades fire
   `mob_engaging`, `mob_disengaging`, `mob_combat_ended` on
   state entry (new transition events; later chunks add their
   own).

6. **Hard cutover within chunk.** `Character.Aggro` field
   deleted at chunk end. No coexistence beyond the migration
   window. The Stage 2a graveyard exists because coexistence
   was indefinite — don't repeat.

7. **Persistence: always boot to Idle.** Combat Phase state
   does not persist across server restart. The character
   awakens uncommitted. (Behavior matrix CP-31 documents this
   explicitly.) Inbound attackers list is reconstructed
   lazily as transitions land post-boot.

8. **Combatant becomes a flag now, mob `Hostile` field
   replaced.** Even though Combatant is technically a "later"
   piece, Combat Phase's entry vetoes need it. Chunk 0
   introduces `Character.Combatant bool` and migrates
   `mob.Hostile` references onto it. The `IsNonCombatant()`
   method on mob becomes a query on the new field.

## Framework design (`internal/state/`)

### Core types

```go
package state

// Machine is a finite state machine instance bound to a
// Character.
type Machine[S comparable] struct {
    // private internals
}

// State returns the current state.
func (m *Machine[S]) State() S

// CanTransition asks whether a transition to `to` is allowed
// given current state, valid-transition table, and registered
// vetoes. Does not mutate.
func (m *Machine[S]) CanTransition(to S, reason TransitionReason) error

// TransitionTo attempts the transition. On success the
// state changes, cascades fire, observers fire. On veto,
// returns the veto error and no state change occurs.
func (m *Machine[S]) TransitionTo(to S, reason TransitionReason) error

// BeforeTransition registers a veto handler. Veto handlers
// are called in registration order; first non-nil return
// halts the chain and that error is returned to the caller.
func (m *Machine[S]) BeforeTransition(handler VetoHandler[S])

// AfterTransition registers a cascade handler. Cascade
// handlers are called in registration order, all of them,
// in the same stack frame as the originating TransitionTo.
// Cascades may call TransitionTo on this or other machines.
func (m *Machine[S]) AfterTransition(handler CascadeHandler[S])

// Subscribe registers a pure observer (no veto, no cascade).
// Used by aliveness substrate, quest engine, telemetry.
func (m *Machine[S]) Subscribe(handler ObserverHandler[S])

// ScheduleTransition registers a deferred transition. When
// `at` is reached (round count or wall time), the framework
// calls TransitionTo. Used for Engaging→Engaged auto-advance
// after RoundsWaiting, Awareness Revealing→Visible delay,
// etc. (Chunk 0 uses for the Engaging→Engaged auto-advance.)
func (m *Machine[S]) ScheduleTransition(to S, at ScheduleAt, reason TransitionReason)

// CancelScheduled cancels any pending scheduled transition.
// Used when a state change preempts a scheduled outcome
// (e.g., interrupted Engaging because target died).
func (m *Machine[S]) CancelScheduled()
```

### Transition handler signatures

```go
type VetoHandler[S comparable] func(from, to S, reason TransitionReason) error

type CascadeHandler[S comparable] func(from, to S, reason TransitionReason)

type ObserverHandler[S comparable] func(from, to S, reason TransitionReason)
```

### TransitionReason

```go
type TransitionReason struct {
    Trigger  string   // "attack_command", "flee_success", "target_died", etc.
    Actor    ActorRef // who initiated (may be self)
    Target   ActorRef // optional (combat target)
    Metadata map[string]any
}
```

Reason propagates through veto, cascade, and observer hooks.
Cascade handlers can branch on reason (e.g., "if reason ==
attack_command AND from == Idle, don't fire mob_engaging for
SurpriseAttack").

### Valid-transitions table

Each machine declares its allowed transitions in a table that
`Machine[S]` validates. Example for Combat Phase:

```go
var combatPhaseTransitions = state.TransitionTable[CombatState]{
    Idle:        {Engaging},
    Engaging:    {Engaged, Idle},          // Idle on cancel/target-died
    Engaged:     {Disengaging, Idle},      // Idle direct on death/despawn
    Disengaging: {Idle, Engaged},          // Engaged on flee failure
}
```

`Machine[S]` rejects transitions not in the table with a
typed `ErrInvalidTransition` (no veto handler needed — it's
a framework-level invariant).

### Framework files

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/machine.go` | NEW | `Machine[S]` core, transition logic |
| `internal/state/transition.go` | NEW | `TransitionReason`, `TransitionTable`, error types |
| `internal/state/handlers.go` | NEW | VetoHandler/CascadeHandler/ObserverHandler types |
| `internal/state/scheduled.go` | NEW | Scheduled-transition machinery (round-tick advance + wall-time variant) |
| `internal/state/actor_ref.go` | NEW | `ActorRef` shared type (user vs mob discriminator with id) |
| `internal/state/machine_test.go` | NEW | Generic framework tests |
| `internal/state/scheduled_test.go` | NEW | Scheduled-transition tests |
| `internal/state/context.md` | NEW | Package documentation |

## Combat Phase design

### States

```go
package combatphase  // OR within internal/state/combatphase/

type State int
const (
    Idle State = iota
    Engaging
    Engaged
    Disengaging
)
```

Each state has an associated data type (some empty):

```go
type IdleData struct{}        // no data
type EngagingData struct {
    Target      ActorRef
    Reason      TransitionReason // captured for SurpriseAttack tracking, etc.
    RoundsUntil int              // typically from weapon WaitRounds
}
type EngagedData struct {
    Target ActorRef
    // RoundsWaiting for next swing — replaces Aggro.RoundsWaiting
    NextSwingAt int
}
type DisengagingData struct {
    Target    ActorRef // last target (for "flee failure → back to Engaged")
    FleeRound int
}
```

The `Machine[State]` instance carries the current state's data
via parallel storage. Access pattern:

```go
if d, ok := cp.EngagedData(); ok {
    target := d.Target
    // ...
}
```

### Transitions

| From | To | Trigger source | Conditions | Cascades |
|------|----|----|-----------|---------|
| Idle | Engaging | attack command (player or mob), btree attack action, surprise-from-hidden | Activity = Free; Combatant = Combatant; Life = Alive; target valid; target.Combatant = Combatant; target.Presence ∈ {Active for player; Active or Dormant for mob} | Awareness Hidden → Revealing (unless reason=SurpriseAttack); B.Attackers.append(self); btree mob_engaging |
| Engaging | Engaged | scheduled (after RoundsUntil); RoundsUntil=0 case is immediate | (no veto — internal state advance) | btree mob_engaged (fires once); first combat round dispatches mob_combat_round on next tick |
| Engaging | Idle | target died / despawned; cancel command; self died | (no veto — these are forced) | B.Attackers.remove(self); CancelScheduled; btree mob_combat_ended |
| Engaged | Disengaging | flee command; self moved (player movement-rejected case clears); target moved out (cross-room policy) | Position ∈ {Standing} (vetoes from Clinched/Grounded) | flee skill check scheduled; on success → Idle, on fail → back to Engaged |
| Engaged | Idle | target died; despawn; self died; Life→Dead | (forced) | B.Attackers.remove(self); btree mob_combat_ended |
| Disengaging | Idle | flee succeeded; cross-room transition completed | (forced — scheduled outcome) | B.Attackers.remove(self); btree mob_combat_ended |
| Disengaging | Engaged | flee failed | (forced — scheduled outcome) | btree mob_re_engaged |

### `Character.Attackers()` query

Framework-maintained list. Read with:

```go
func (cp *Machine[State]) Attackers() []ActorRef
```

Mutated only by the framework on transitions:
- `A.cp.TransitionTo(Engaging{Target:B}, ...)` → framework pushes A onto B.cp.Attackers
- `A.cp.TransitionTo(Idle, ...)` → framework removes A from B.cp.Attackers (if A was attacking B)
- `A.cp.TransitionTo(Engaging{Target:C}, ...)` where A was previously Engaged with B → framework removes A from B.cp.Attackers, pushes A onto C.cp.Attackers

No caller bookkeeping. The list is always coherent with the
state.

## Behavior Matrix

The intent rules. Each row becomes a RED-phase test before
implementation begins. ID prefix `CP-` = Combat Phase.

### Standard entry / exit

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-001 | A.Idle, B.Idle | A: attack command targeting B | A.Activity=Free, A.Combatant=Combatant, A.Life=Alive, B.Combatant=Combatant, B.Life=Alive, B.Presence valid | A → Engaging{B}; B unchanged (B's Combat Phase doesn't auto-engage; B chooses via btree/player command). B.Attackers gains A. | Attack initiates engagement on attacker side only; defender independently decides whether to engage back. |
| CP-002 | A.Engaging{B}, R rounds remaining | next round tick, R>0 | none | R decrement; remain Engaging | Pre-engagement wait absorbs weapon WaitRounds |
| CP-003 | A.Engaging{B}, R=0 | scheduled transition fires | (forced) | A → Engaged{B}; fire `mob_engaged` btree event; first `mob_combat_round` on next tick | Engagement completes |
| CP-004 | A.Engaged{B}, B dies | B.Life → Dead cascade | (forced) | A → Idle; B.Attackers no longer contains A; fire `mob_combat_ended` | Target gone, combat ends |
| CP-005 | A.Engaged{B}, A dies | A.Life → Dead cascade | (forced) | A → Idle; B.Attackers loses A; all inbound attackers of A see A.Attackers cleared (symmetric to current player death) | Self-death ends own combat outbound + inbound |
| CP-006 | A.Engaged{B} | A: flee command | A.Position=Standing | A → Disengaging{B}; schedule flee-outcome transition after 1 round | Flee starts a disengage attempt |
| CP-007 | A.Disengaging{B} | scheduled fire, flee roll succeeded | (forced) | A → Idle; B.Attackers loses A; A leaves room (movement happens during Disengaging) | Successful flee disengages and moves |
| CP-008 | A.Disengaging{B} | scheduled fire, flee roll failed | (forced) | A → Engaged{B}; remain in room; A.Attackers list intact | Failed flee stays engaged |
| CP-009 | A.Idle, B.Idle | A: attack command, A.Awareness=Hidden | base + Hidden | A → Engaging{B} with reason=SurpriseAttack; Awareness NOT cascaded to Revealing on this transition (preserves stealth for first swing); B.Attackers gains A | Stealth attack gets one round of surprise |

### Vetoes (entry blocked)

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-010 | A.Idle | A: attack command | A.Combatant = NonCombatant | Veto, error returned, no transition | Pacifist can't attack |
| CP-011 | A.Idle | A: attack command targeting B | B.Combatant = NonCombatant | Veto, error returned | Pacifist target can't be attacked |
| CP-012 | A.Idle | A: attack command | A.Activity ≠ Free | Veto, error returned | Can't start fight while casting/crafting |
| CP-013 | A.Idle | A: attack command | A.Life ≠ Alive | Veto, error returned | Dead/respawning can't attack |
| CP-014 | A.Idle | A: attack command targeting B | B.Life ≠ Alive | Veto, error returned | Can't attack the dead |
| CP-015 | A.Idle | A: attack command targeting B (player) | B.Presence ∈ {AFK, Disconnected} | Veto, error returned with grace-protect message | AFK/disconnected players are not targetable |
| CP-016 | A.Engaged{B} | A: flee command | A.Position ∈ {Clinched, Grounded} | Veto, error returned with "you're grappled" message | Must break grapple before fleeing |

### Multi-attacker scenarios

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-017 | M1.Idle, M2.Idle, M3.Idle, P.Idle | M1 attacks P, then M2, then M3 (sequentially) | each passes veto checks | P.Attackers = [M1, M2, M3] (order preserved); P.Combat Phase remains Idle (defenders don't auto-engage) | Multi-attacker inbound list maintained framework-side |
| CP-018 | M1.Engaged{P}, M2.Engaged{P}, P.Engaged{M1} | M1 dies | M1.Life → Dead | M1 → Idle, P.Attackers = [M2]. P.Combat Phase: target M1 invalidated, P → Idle (no auto-retarget to M2); P retains aggression intent via attack command if desired | Death of attacker cleans inbound; defender doesn't auto-retarget (preserves current behavior — players choose new target manually) |
| CP-019 | M1.Engaged{P}, M2.Engaged{P} | M2 has companion C of P; C's btree subscribes to P.Attackers change | P.Attackers grew | C's btree fires its assist branch, transitions C → Engaging{M2} | Companion auto-assist becomes a btree subscriber to inbound list change |

### Cross-room scenarios

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-020 | P.Engaged{M}, both in room R1 | P moves north to room R2 | P.Position = Standing, exit valid | P transitions Engaged → Disengaging (because movement during Engaged requires exit) → after 0-round flee → Idle; M.Attackers loses P; M.Combat Phase unaffected | Player movement during combat goes through Disengaging |
| CP-021 | M.Engaged{P}, P moves to room R2 | P moves out | M has cross-room policy enabled | M.Engaged data persists; M.cp tracks target across rooms; next M tick triggers btree branch "target in adjacent room" which can decide to pursue (via `go <direction>`) or stay (transition Engaged → Idle) | Mobs CAN follow players across rooms (framework supports; archetypes decide); ends the "legitimate asymmetry" |
| CP-022 | M.Engaged{P}, P moves to room R5 (non-adjacent) | P moves multiple rooms | M.Engaged data references target | M.Combat Phase auto-transitions Engaged → Idle on next tick (target unreachable); fire `mob_combat_ended` | Target out of pursuit range ends combat |

### Stealth & surprise

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-023 | A.Idle, A.Awareness=Hidden | A: attack command | base | A → Engaging{B, reason=SurpriseAttack}; A.Awareness stays Hidden until end of first combat round | Surprise bonus rewards the setup |
| CP-024 | A.Engaging{B, reason=SurpriseAttack} → Engaged | scheduled fire | (forced) | A → Engaged{B}; A.Awareness Hidden → Revealing → Visible (cascade on Engaged entry, NOT Engaging) | Stealth breaks on first swing landed, not on engage start |
| CP-025 | A.Engaged{B}, A.Awareness=Hidden | round tick fires combat round | A took an attack action | A.Awareness Hidden → Revealing → Visible after the attack | Active combat reveals; passive observation does not |

### Non-combat target picking (the chunk 2.7 bug case)

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-026 | T.Idle (thief mob), P.Idle (player) | T's btree picks P as steal target | T.Activity=Free, T.Combatant=Combatant | T.Combat Phase **does NOT transition**; T's btree calls `actions.Steal(T, opts{TargetUserId: P.UserId})` directly; the action takes target as argument | This is the chunk 2.7 bug fixed structurally. Target lives on the Engaged state ONLY; non-combat picks pass target via action params. |
| CP-027 | T.Engaged{P} after a failed steal that triggered combat | hypothetical: steal action sets aggro on detection-fail | (no veto — explicit transition) | Goes through normal Engaging path with all vetoes/cascades; legacy "auto-aggro on failed steal" must be explicit | Action authors who want failed-steal-to-combat must call TransitionTo explicitly with all vetoes firing. No silent "you're in combat now" side effect. |

### Life cascade (Death)

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-028 | A.Engaged{B} | A.Life → Dead transition | (cascade) | A.Combat Phase → Idle (forced, no veto); B.Attackers loses A; for each X ∈ A.Attackers, X.Combat Phase → Idle (their target is dead). | Death cascades to clean both outbound + inbound cleanly. Replaces scattered cleanup in suicide.go. |
| CP-029 | A.Idle, A is attacked by N mobs (A.Attackers populated) | A.Life → Dead transition | (cascade) | A.Attackers cleared; each X → Idle. | Inbound attackers reset on death even when outbound was Idle (e.g., player downed without attacking back). |

### Persistence

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-030 | A.Engaged{B} at shutdown | server restart | (boot) | A.Combat Phase reset to Idle; A.Attackers reset to empty; the world boots clean of combat state. | No combat-mid-restart confusion; players rejoin uncommitted. |
| CP-031 | A.Disengaging at shutdown | server restart | (boot) | A.Combat Phase = Idle | Scheduled transitions don't survive restart (chunk 0). Later chunks may add wall-time scheduled transitions; round-tick scheduled transitions explicitly do not persist. |

### Combatant flag

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-032 | A.Engaged{B} | A.Combatant → NonCombatant (flag toggle) | (flag mutation, NOT machine transition) | A.Combat Phase forced → Idle; B.Attackers loses A; future Engaging vetoed | Pacifism is immediate. Replaces today's per-call IsNonCombatant() checks scattered through commands. |

### Round driver and tick events

| ID | Start state | Trigger | Conditions | Outcome | Rationale |
|----|-------------|---------|-----------|---------|-----------|
| CP-033 | A.Engaged{B} | round tick | (per round) | fire `mob_combat_round` btree event for A; trigger combat resolution | Engaged state drives combat round dispatch |
| CP-034 | A.Idle | round tick | (per round) | fire `mob_idle` btree event for A | Idle state drives idle dispatch |
| CP-035 | A.Engaging | round tick | (per round) | RoundsUntil decrement; NO `mob_combat_round` fired; NO `mob_idle` fired | Pre-engagement is silent — neither idle nor combat |
| CP-036 | A.Disengaging | round tick | (per round) | flee resolution scheduled; NO tick btree event fires | Mid-disengage is silent until resolution |

## Smoke scenarios

In-game validation after the chunk lands. Each maps to one or
more Behavior Matrix rows.

1. **Basic combat (CP-001, CP-003, CP-004, CP-033):** Player
   attacks a hostile mob in Thornwall Outskirts. Combat
   engages, swings resolve, mob dies, player returns to Idle.

2. **Player flee — success (CP-006, CP-007):** Player engages,
   then flees. Verify Engaged → Disengaging → Idle transition;
   player moves to adjacent room; mob's Attackers list updates.

3. **Player flee — failure (CP-006, CP-008):** Player engages
   with a faster mob, flees, flee roll fails. Verify
   Disengaging → Engaged; player stays in room.

4. **Grapple blocks flee (CP-016):** Mob grapples player;
   player attempts flee; veto fires with grappled message.

5. **Multi-attacker (CP-017):** Spawn 3 mobs against a single
   player. Verify `player.Attackers().Len() == 3`. Kill one
   mob; verify list shrinks.

6. **Companion auto-assist symmetry (CP-019):** Player has a
   companion. Mob attacks companion. Player's btree (or
   command) sees companion's Attackers list grew; player can
   transition to Engaging on the attacker. Both directions of
   the assist channel work.

7. **Cross-room chase (CP-021):** Player engages mob, then
   walks to adjacent room. Verify the mob (with pursuit
   archetype branch) follows; player remains targetable.

8. **Chunk 2.7 thief regression (CP-026):** Thornwall
   highwayman with thief archetype, hidden, picks player as
   steal target. Verify **highwayman remains in Idle Combat
   Phase** while attempting the steal. No grapple. Steal
   succeeds or fails per skill check; either way, no combat
   side effects unless explicitly triggered.

9. **Death cascade (CP-028):** Player kills a mob mid-combat.
   Verify mob's Combat Phase → Idle, its Attackers cleared,
   and any other mobs attacking the dying mob also see their
   target invalidated and transition to Idle.

10. **Persistence (CP-030):** Mid-combat, kill the server.
    Restart. Verify all characters boot to Combat Phase = Idle.

## Migration approach

**Hard cutover within chunk 0**. Sequencing within the chunk
(spec'd here; concrete tasks land in the plan):

1. **Framework foundation.** `internal/state/` package created
   with `Machine`, transitions, vetoes, cascades, observers,
   scheduled transitions. Generic framework tests pass.

2. **Combat Phase machine defined.** `CombatState` enum, state
   data types, transition table, valid-transition matrix.
   `Character.CombatPhase *state.Machine[CombatState]` field
   added (alongside `Character.Aggro` temporarily).

3. **RED phase.** Tests written from Behavior Matrix.
   ~35 tests in `combat_phase_test.go`. All fail.

4. **GREEN phase.** Behavior Matrix implemented. All tests
   pass.

5. **Reader migration.** Every `Aggro != nil` / `Aggro.UserId`
   / `Aggro.MobInstanceId` site migrated to `CombatPhase`
   predicates and queries. Old Aggro field still readable
   (becomes derived: getter proxies to CombatPhase). Build
   green, tests green.

6. **Writer migration.** Every `SetAggro` / `EndAggro` site
   migrated to `CombatPhase.TransitionTo(...)`. The derived
   Aggro field now has no writers. Tests still green.

7. **Btree event integration.** Round driver updated to
   dispatch tick events from Combat Phase state. Combat Phase
   cascades wired to fire `mob_engaging` / `mob_engaged` /
   `mob_disengaging` / `mob_combat_ended` btree events.

8. **Aliveness substrate migration.** Faction rep, opinion
   bumps, crime witnessing, knowledge recording — all migrated
   from direct `Aggro` reads to Combat Phase observer
   subscriptions. Behaviors verified per existing chunk
   smoke tests.

9. **Combatant flag.** `Character.Combatant bool` field added.
   `mob.Hostile` references migrated. `IsNonCombatant()` becomes
   a query on the flag. Veto rules CP-010/CP-011 wired.

10. **Sunset.** `Character.Aggro` field deleted. All compile
    errors fixed (should be zero — all callers migrated). Old
    `NewRound_DoCombat_unified.go` deleted (it was the Stage
    2a graveyard for this exact work).

11. **In-game smoke.** Walk through smoke scenarios 1-10.

12. **Roadmap + chunk 2.7 re-smoke.** Mark chunk 0 done. Re-run
    chunk 2.7 thief-archetype smoke; verify the bug is gone
    structurally. Mark chunk 2.7 Task 19 done.

## Sunset list

- `Character.Aggro *Aggro` field
- `internal/characters/aggro.go` (entire file — Aggro struct,
  AggroType enum, SetAggro, EndAggro, ValidateAggro logic moves
  to framework / cascade handlers)
- `internal/hooks/aggro_helpers.go` (ValidateAggro,
  RetargetOrEnd, CompanionAutoTarget — replaced by framework's
  inbound-attacker tracking + cascade handlers)
- `internal/hooks/NewRound_DoCombat_unified.go` (Stage 2a
  graveyard — replaced by Combat-Phase-driven round driver)
- `mob.Hostile bool` field — replaced by `Character.Combatant`
- `mob.IsNonCombatant()` method — folded into Combatant flag
  query
- Ad-hoc `Aggro != nil` checks (~30 sites across
  usercommands, mobcommands, hooks, behaviortree) — replaced
  by `IsEngaged()` predicate

## Risks / known limitations

- **Framework v1 may need refinement during chunks 1-5.**
  Co-developing with one machine means later machines might
  expose API gaps. Expected; each later chunk gets a small
  refinement budget for the framework.

- **Cascade chain depth.** Combat Phase → Awareness →
  Buffs.cancel → ... can be deep. Synchronous cascades make
  this debuggable but a pathological cycle would stack-
  overflow. Framework adds a cycle-detection assertion (dev
  mode only).

- **Observer ordering.** Multiple subscribers on the same
  transition fire in registration order. Aliveness substrate
  + quest engine + telemetry all subscribe; their order is
  effectively fixed by init code. Document this in framework
  context.md.

- **Btree event vocabulary expansion.** Adding
  `mob_engaging` / `mob_disengaging` / `mob_combat_ended` is
  three new events. Archetype YAMLs don't need to use them
  (they default to not subscribing); but archetype authors
  should know they exist. Update btree context.md.

## Open questions

- **Surprise attack timing (CP-009 / CP-024).** The matrix
  says stealth persists through Engaging→Engaged, breaks on
  the first swing landing during Engaged. Edge case: what if
  the surprise attack misses? Does stealth still break, or
  does the attacker get another stealth-bonus swing? Spec
  says it breaks (the swing was made, regardless of outcome).
  Worth user confirmation during spec review.

- **Cross-room mob pursuit policy.** CP-021 says mobs CAN
  follow but the archetype decides whether to. Default
  behavior: do not pursue. Pursuit is an opt-in archetype
  feature for later (bounty hunter chunk 5.2). Confirm.

## Roadmap impact

- Master spec `2026-05-13-combat-state-machines-design.md`
  references this chunk as Chunk 0.
- On completion: chunk 0 marked Done; chunk 1 (Awareness)
  brainstorm + spec + plan + execute begins.
- Chunk 2.7 (mob skullduggery suite) Task 19 unblocked after
  this chunk's smoke scenario 8 passes.
