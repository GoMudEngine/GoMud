# combatphase — Package Documentation

## Overview

The `internal/state/combatphase` package is the first consumer of the
`internal/state` framework. It defines the **Combat Phase state machine**,
replacing `Character.Aggro` as the source of truth for "who am I attacking?"
and "am I in combat?"

Combat Phase has four states:

| State | Meaning |
|-------|---------|
| `Idle` | Not engaged in combat. |
| `Engaging` | Attack initiated; weapon wait-rounds countdown in progress. |
| `Engaged` | Actively fighting; round driver processes swings. |
| `Disengaging` | Flee in progress; waiting for flee resolution. |

The machine is mob/player symmetric: both player and mob `Character` instances
carry a `*combatphase.Machine` field. The btree event wiring fires only for
mob instances (players have no behavior tree); combat round dispatch reads the
same predicates for both.

---

## Key Components

### Core Files

- **combatphase.go** — `Machine` wrapper, state-data structs, all public API
  (transition methods, predicate methods, registration methods, Attackers
  tracking, tick-event dispatch).
- **transitions.go** — `validTransitions` table constant and Trigger string
  constants.
- **combatphase_test.go** — Behavior Matrix tests CP-001 through CP-036.

---

## Key Functions

### Construction

```go
func NewMachine() *Machine
```

Returns a `Machine` in `Idle`. Always call `RegisterMachine` immediately
after creating a character so inbound-attacker tracking and
cross-character target-death notifications work correctly.

### Entry into combat

```go
func (m *Machine) TransitionToEngaging(d EngagingData, r TransitionReason) error
```

Primary entry point into combat. Runs the seven veto chain (Combatant,
Activity, Life, Position, TargetCombatant, TargetLife, TargetPresence)
before delegating to the inner framework transition. On success:
- Stores `EngagingData` on the machine.
- Calls `target.RecordInboundAttacker(r.Actor)` to register self on the
  target's inbound list (enabling companion auto-assist).

### Disengaging

```go
func (m *Machine) TransitionToDisengaging(r TransitionReason) error
```

Starts a flee attempt. Vetoes via `positionSelf` when trigger is
`TriggerFleeCommand` and the character is grappled/clinched/grounded.

### Round tick

```go
func (m *Machine) OnRoundTick()
```

Called once per round per character by the round driver. In `Engaging`
state: decrements `RoundsUntil` and advances to `Engaged` when it hits
zero. In `Engaged` and `Disengaging`: no-op (those states are driven
externally by the combat resolution and flee logic).

### Flee resolution

```go
func (m *Machine) ResolveFlee(success bool)
```

Finalizes a `Disengaging` state. Success → `ForceIdle`
(`TriggerFleeSuccess`). Failure → back to `Engaged` with the last known
target restored in `EngagedData`.

### Force-idle

```go
func (m *Machine) ForceIdle(r TransitionReason)
```

Transitions to `Idle` from any state, clearing all state-data and
removing self from the target's inbound list. Used for death cascade,
`Combatant`-toggle, target-died, and `EndAggro`.

### Target-died notification

```go
func (m *Machine) NotifyTargetDied(target ActorRef)
```

Called by the dying target's `NotifySelfDied`. If my current target
matches, transitions to `Idle`.

### Self-died notification

```go
func (m *Machine) NotifySelfDied()
```

Called when this character dies. Clears own outbound state, clears all
inbound attackers, and notifies each inbound attacker that their target
is gone (causing them to force-idle too).

### Tick-event dispatch

```go
func (m *Machine) DispatchTickEvent()
```

Fires the per-state tick event to all registered listeners:
- `Engaged` → fires `"mob_combat_round"`
- `Idle` → fires `"mob_idle"`
- `Engaging` / `Disengaging` → silent (no event)

Called once per round per character by the round driver before combat
resolution. Drives btree idle and combat-round events without polling.

### Predicate methods

```go
func (m *Machine) IsEngaged() bool       // State == Engaged
func (m *Machine) IsInCombat() bool      // State != Idle
func (m *Machine) CurrentTarget() ActorRef  // target across all non-Idle states
```

### Attackers

```go
func (m *Machine) Attackers() []state.ActorRef
func (m *Machine) RecordInboundAttacker(a state.ActorRef)
func (m *Machine) RemoveInboundAttacker(a state.ActorRef)
func (m *Machine) SubscribeAttackersChange(fn func([]state.ActorRef))
```

Framework-maintained inbound attacker list. `TransitionToEngaging` calls
`RecordInboundAttacker` on the target; `ForceIdle` calls
`RemoveInboundAttacker`. `SubscribeAttackersChange` is used by
`CombatPhase_CompanionAssist.go` to trigger reactive companion auto-assist
the moment a new attacker is recorded.

### Veto registration

```go
func (m *Machine) RegisterCombatantVeto(check func() bool)
func (m *Machine) RegisterActivityCheck(check func() bool)
func (m *Machine) RegisterLifeCheck(check func() bool)
func (m *Machine) RegisterPositionCheck(check func() bool)
func (m *Machine) RegisterTargetCombatantCheck(c func(ActorRef) bool)
func (m *Machine) RegisterTargetLifeCheck(c func(ActorRef) bool)
func (m *Machine) RegisterTargetPresenceCheck(c func(ActorRef) bool)
```

Wired at character-creation time by `CombatPhase_Vetoes.go` via the
`OnCharacterCreated` callback. Each registration accepts a closure over
the character's own fields so the checks always read current values.

### Surprise round

```go
func (m *Machine) OnEndOfRoundIfSurprise(fn func(TransitionReason))
func (m *Machine) OnCombatRoundEnd()
```

`OnEndOfRoundIfSurprise` is reserved for the Awareness machine (chunk 1)
to transition `Hidden → Revealing → Visible` after all weapon swings for
a surprise round complete. `OnCombatRoundEnd` is called by the round
driver; it fires the registered callbacks and clears `SurpriseLeft` on
the first Engaged round of a surprise engagement.

---

## Global State

### machineRegistry

```go
var machineRegistry = map[state.ActorRef]*Machine{}
```

Guarded by `registryMu`. Populated by `RegisterMachine` at character
creation; cleared by `UnregisterMachine` on logout / despawn.

The registry is the bridge between `ActorRef` (the identity type used
in `TransitionReason.Target`) and the live `Machine` pointer. It allows
the framework to call `NotifyTargetDied` and `RecordInboundAttacker` on
a target's machine without the caller knowing its memory address.

---

## Data Structure Design

### State enum

```go
type State int
const (
    Idle State = iota
    Engaging
    Engaged
    Disengaging
)
```

### Per-state data structs

```go
type EngagingData struct {
    Target      state.ActorRef
    Reason      state.TransitionReason
    RoundsUntil int  // weapon WaitRounds before first swing
}

type EngagedData struct {
    Target       state.ActorRef
    NextSwingAt  int   // round number for next swing
    SurpriseLeft bool  // true during the first round of a surprise engagement
}

type DisengagingData struct {
    LastTarget state.ActorRef  // target at time of flee initiation
    FleeRound  int             // round flee was initiated
}
```

Stored as nullable pointer fields on `Machine`. Only the pointer for the
current state is non-nil; transitions nil out the previous state's data.

### validTransitions

```go
var validTransitions = state.TransitionTable[State]{
    Idle:        {Engaging},
    Engaging:    {Engaged, Idle},     // Idle on cancel / target-died
    Engaged:     {Disengaging, Idle}, // Idle direct on death / despawn
    Disengaging: {Idle, Engaged},     // Engaged on flee failure
}
```

### Trigger constants

Defined in `transitions.go`. Use these string constants instead of
inline literals to ensure stable identifiers across the codebase:

| Constant | Value |
|----------|-------|
| `TriggerAttackCommand` | `"attack_command"` |
| `TriggerSurpriseAttack` | `"surprise_attack"` |
| `TriggerEngagementReady` | `"engagement_ready"` |
| `TriggerFleeCommand` | `"flee_command"` |
| `TriggerFleeSuccess` | `"flee_success"` |
| `TriggerFleeFailure` | `"flee_failure"` |
| `TriggerTargetDied` | `"target_died"` |
| `TriggerSelfDied` | `"self_died"` |
| `TriggerTargetOutOfRoom` | `"target_out_of_room"` |
| `TriggerCharm` | `"charm_acquired"` |
| `TriggerCombatantToggle` | `"combatant_toggle"` |
| `TriggerForceIdle` | `"force_idle"` |

---

## Integration Notes

### Consumes

- `internal/state` — framework (`Machine[State]`, `TransitionTable`,
  `TransitionReason`, `ActorRef`).

### Consumed by

- **`internal/characters`** — `Character.CombatPhase *combatphase.Machine`
  field. Initialized in `New()` and (lazily) in `Validate()`. Predicate
  methods `IsEngaged()`, `IsInCombat()`, `IsDisengaging()`,
  `EngagedTarget()`, `CurrentCombatTarget()`, `Attackers()` delegate to
  the machine.
- **`internal/hooks/CombatPhase_Vetoes.go`** — wires the seven veto
  callbacks at character-creation time via `OnCharacterCreated`.
- **`internal/hooks/CombatPhase_BtreeEvents.go`** — registers an
  `AfterTransition` cascade that fires `mob_engaging`, `mob_engaged`,
  `mob_disengaging`, `mob_combat_ended` btree events on state changes.
- **`internal/hooks/CombatPhase_CompanionAssist.go`** — registers
  `SubscribeAttackersChange` to reactively trigger companion auto-assist
  when a new attacker is recorded on the companion's inbound list.

### Replaces

`Character.Aggro` field (compat wrappers in
`internal/characters/combat_state_compat.go`). The `Aggro` struct and
`SetAggro` / `EndAggro` methods are preserved as a compat surface for
the ~200 direct field reads across the codebase that were not migrated
in chunk 0. All writes dual-write to `CombatPhase`. Full field removal
is scheduled for a cleanup chunk after chunks 1-5 land.

---

## Testing Notes

### combatphase_test.go — Behavior Matrix

Tests follow the CP-NNN naming scheme from the chunk 0 spec. Each test
exercises one cell of the state × trigger × veto matrix.

| Range | Area |
|-------|------|
| CP-001 – CP-008 | Basic state transitions (happy path) |
| CP-010 – CP-016 | Veto rules (non-combatant, dead, busy, grappled) |
| CP-017 – CP-022 | ForceIdle + NotifySelfDied cascade |
| CP-023 – CP-025b | Surprise attack — first-round bonus + SurpriseLeft |
| CP-026 – CP-036 | Tick events, attacker list, flee, retarget, edge cases |

All tests use local `Machine` instances with injected veto closures;
no server / database setup required.
