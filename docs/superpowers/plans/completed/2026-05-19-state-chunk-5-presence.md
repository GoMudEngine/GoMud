# State Chunk 5 — Presence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Presence state machine (the sixth and last of the combat-state-machines arc except for Perception) that centralizes "is this character meaningfully present?" into one canonical FSM with per-actor state lists encoded in a union enum.

**Architecture:** New `internal/state/presence` package with a single `State` enum spanning both actor types (Active shared; Connecting/Idle/AFK/Disconnected player-only; Spawning/Dormant/Despawning mob-only). Two transition tables, one per actor, on the same `Machine[State]` type. `Character.Presence *presence.Machine` field on `characters.Character`. A new `NewRound_PresenceTick` hook drives timeout transitions. Veto handlers gate Active→Dormant/Despawning on `IsEssential() || IsCharmed()` and gate `CombatPhase.Idle→Engaging` on `Disconnected/Despawning`. Scheduled-transition cancellation observer wired on terminal states. Sunset list: `ManualAFK`, `AFKMessage`, `BoredomCounter`, `PreventIdle`, `MaxMobBoredom`.

**Tech Stack:** Go, existing `internal/state` framework, `internal/state/combatphase` (veto integration), `internal/users`, `internal/mobs`, `internal/hooks` (round-driver hook), `internal/usercommands/afk.go`, `internal/configs`.

**Spec reference:** `docs/superpowers/specs/2026-05-19-state-chunk-5-presence-design.md`

---

## File Structure

### New files

| Path | Responsibility |
|---|---|
| `internal/state/presence/presence.go` | `State` enum, `String()` method, `AFKData` struct, constructor functions `NewPlayerPresence()` + `NewMobPresence()`, `Machine` wrapper. |
| `internal/state/presence/transitions.go` | Two `TransitionTable[State]` constants — `playerTransitions` + `mobTransitions` — plus trigger string constants. |
| `internal/state/presence/presence_test.go` | Behavior Matrix unit tests (rows PR-001 through PR-031). |
| `internal/state/presence/integration_test.go` | Veto + observer integration covering `CombatPhase.Idle→Engaging` block list and scheduler cleanup. |
| `internal/state/presence/context.md` | Package documentation. |
| `internal/hooks/NewRound_PresenceTick.go` | Per-round timeout transitions: player Active↔Idle↔AFK↔Disconnected, mob Active↔Dormant↔Despawning. |

### Modified files

| Path | Change |
|---|---|
| `internal/characters/character.go` | Add `Presence *presence.Machine` field; reset in `ResetForMobInstance`. |
| `internal/characters/character.go:New()` | Initialize `Presence = NewPlayerPresence()` (player default; mobs override in `Validate`). |
| `internal/mobs/mobs.go:Validate()` | Initialize `Presence = NewMobPresence()` for mob instances (after `ResetForMobInstance`). |
| `internal/state/combatphase/combatphase.go` | Already has a `targetPresence func(state.ActorRef) bool` veto slot (line 68). Populate it during machine setup. |
| `internal/configs/config.server.go` | Add `PresenceIdleAfterRounds`, `PresenceAFKAfterRounds`, `PresenceDisconnectAfterRounds`, `PresenceMobDormantAfterRounds`, `PresenceMobDespawnAfterRounds` knobs. |
| `internal/configs/config.memory.go` | Remove `MaxMobBoredom` (replaced by `PresenceMobDormantAfterRounds`). |
| `internal/users/users.go` | Login flow fires `Presence.TransitionTo(Active, ...)` after character enters room. TCP-close path fires `Presence.TransitionTo(Disconnected, ...)`. |
| `internal/users/userrecord.go` | Drop `ManualAFK` + `AFKMessage` fields. `getOnlineInfo()` reads `Presence.State() == AFK` for the `IsAFK` shim. |
| `internal/usercommands/afk.go` | Rewrite to transition the Presence machine. |
| `internal/usercommands/usercommands.go` | Drop the `ManualAFK` clear-on-next-cmd shim (lines 294-297). Any non-`afk` command fires `Presence.TransitionTo(Active)`. |
| `internal/usercommands/online.go` | Unchanged — reads `OnlineInfo.IsAFK` (still populated correctly). |
| `internal/rooms/roomdetails.go` | Read manual-AFK indicator via `AFKData` instead of `user.ManualAFK`. |
| `internal/mobs/mobs.go` | Drop `BoredomCounter` and `PreventIdle` fields. |
| `internal/hooks/NewRound_IdleMobs.go` | Replace `mob.BoredomCounter >= maxBoredom` despawn path with `mob.Character.Presence.State() == Despawning` terminal-tick removal. |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | Drop any `BoredomCounter` reset patterns; rely on Presence transitions. |
| `internal/mobcommands/lookfortrouble.go` | Drop `mob.BoredomCounter++` (lines 186-187). Track "rounds since last target found" via a new per-mob field `LastTargetFoundRound uint64`. |
| `internal/rooms/rooms.go:2144` | Drop `mob.BoredomCounter = 0`; rely on the Presence player-entry observer (registered via room-change event). |
| `internal/hooks/hooks.go` | Add `events.RegisterListener(events.NewRound{}, PresenceTick)` and `events.RegisterListener(events.RoomChange{}, PresencePlayerEntry)` listener registrations. |
| `internal/state/presence/context.md` (new) | Package documentation. |
| `internal/state/combatphase/context.md` | Note the Presence veto on Idle→Engaging. |
| `internal/characters/context.md` | Note the `Presence` field and its initialization. |
| `internal/hooks/context.md` | Document `NewRound_PresenceTick.go` in the NewRound listener chain. |
| `internal/users/context.md` | Document the Presence machine on UserRecord (via Character). |
| `internal/mobs/context.md` | Note the sunset of `BoredomCounter` and `PreventIdle`. |
| `COMBAT_STATE_ROADMAP.md` | Mark chunk 5 as Done. |
| `PATCH_NOTES.md` | Dated entry for chunk 5. |

---

## Tasks

### Task 1: Presence package + Behavior Matrix tests

**Files:**
- Create: `internal/state/presence/presence.go`
- Create: `internal/state/presence/transitions.go`
- Create: `internal/state/presence/presence_test.go`

- [ ] **Step 1: Write the transition table + state enum (RED-phase, tests first below)**

Create `internal/state/presence/presence.go`:

```go
// Package presence defines the Presence state machine — the sixth
// consumer of internal/state, after combatphase, awareness, life,
// activity, and position. It centralizes "is this character meaningfully
// present?" into one canonical FSM with per-actor state lists encoded
// in a single union enum.
//
// Player states: Connecting / Active / Idle / AFK / Disconnected.
// Mob states:    Spawning / Active / Dormant / Despawning.
// Active is shared between actors; transitions are per-actor (two
// transition tables, one constructor per actor type).
//
// Sunsets: ManualAFK, AFKMessage (UserRecord); BoredomCounter, PreventIdle
// (Mob); MaxMobBoredom (config).
package presence

import (
	"github.com/GoMudEngine/GoMud/internal/state"
)

// State is the Presence state enum. Per-actor polymorphism is enforced
// by the transition tables, not by the type system — both actor types
// share the same enum so Character only carries one *Machine pointer.
type State int

const (
	Active       State = iota // both actors — normal "in world, ticking"
	Connecting                // player-only — logged in but not yet in a room
	Idle                      // player-only — no input for N rounds
	AFK                       // player-only — no input for M rounds OR manual cmd
	Disconnected              // player-only — TCP gone, character in graveyard
	Spawning                  // mob-only — freshly created, auto → Active next tick
	Dormant                   // mob-only — bored OR zone has no nearby players
	Despawning                // mob-only — scheduled for removal next tick
)

// String for logging/debugging.
func (s State) String() string {
	switch s {
	case Active:
		return "Active"
	case Connecting:
		return "Connecting"
	case Idle:
		return "Idle"
	case AFK:
		return "AFK"
	case Disconnected:
		return "Disconnected"
	case Spawning:
		return "Spawning"
	case Dormant:
		return "Dormant"
	case Despawning:
		return "Despawning"
	}
	return "Unknown"
}

// AFKData is the only state-data struct. Carried on AFK-state entry
// from either the manual `afk <message>` command (Manual=true) or
// the timeout transition (Manual=false, Message="").
type AFKData struct {
	Message string // set by manual `afk <message>`; empty for auto-AFK
	Manual  bool   // distinguishes manual vs timeout
}

// Machine wraps state.Machine[State] with Presence-specific API.
// Per-state data is stored alongside the inner machine. Only AFK
// state carries data; other states are stateless.
type Machine struct {
	inner   *state.Machine[State]
	afkData *AFKData
	self    state.ActorRef
}

// NewPlayerPresence returns a Machine in Connecting state with the
// player transition table. Used by characters.New() for player
// characters.
func NewPlayerPresence() *Machine {
	return &Machine{
		inner: state.NewMachine(Connecting, playerTransitions),
	}
}

// NewMobPresence returns a Machine in Spawning state with the mob
// transition table. Used by mobs.Mob.Validate() after a fresh shallow
// copy via newMobByIdInternal.
func NewMobPresence() *Machine {
	return &Machine{
		inner: state.NewMachine(Spawning, mobTransitions),
	}
}

// State returns the current state. Safe from any goroutine.
func (m *Machine) State() State {
	if m == nil || m.inner == nil {
		return Active
	}
	return m.inner.State()
}

// AFKData returns the AFK state-data (Message + Manual flag) if the
// machine is currently in AFK. Returns (zero, false) otherwise.
func (m *Machine) AFKData() (AFKData, bool) {
	if m == nil || m.State() != AFK || m.afkData == nil {
		return AFKData{}, false
	}
	return *m.afkData, true
}

// SetSelf binds the machine to its owning ActorRef. Called from
// engine integration when the machine is attached to a Character.
func (m *Machine) SetSelf(ref state.ActorRef) { m.self = ref }

// Self returns the bound ActorRef.
func (m *Machine) Self() state.ActorRef { return m.self }

// TransitionTo moves the machine to the target state. For AFK
// transitions that should carry data, use TransitionToAFK instead.
func (m *Machine) TransitionTo(to State, r state.TransitionReason) error {
	if m == nil || m.inner == nil {
		return nil
	}
	// Clear AFK data when leaving AFK.
	if m.State() == AFK && to != AFK {
		m.afkData = nil
	}
	return m.inner.TransitionTo(to, r)
}

// TransitionToAFK is a convenience for AFK with data.
func (m *Machine) TransitionToAFK(d AFKData, r state.TransitionReason) error {
	if m == nil || m.inner == nil {
		return nil
	}
	if err := m.inner.TransitionTo(AFK, r); err != nil {
		return err
	}
	m.afkData = &d
	return nil
}

// RegisterVeto installs a veto on transitions from `from` to `to`.
// Used by the engine to gate Active→Dormant/Despawning on
// IsEssential() + IsCharmed().
func (m *Machine) RegisterVeto(from, to State, veto func(state.TransitionReason) error) {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.RegisterVeto(from, to, veto)
}

// RegisterObserver installs an observer fired after every transition.
// Used for terminal-state scheduler cleanup.
func (m *Machine) RegisterObserver(obs func(state.Event)) {
	if m == nil || m.inner == nil {
		return
	}
	m.inner.RegisterObserver(obs)
}
```

- [ ] **Step 2: Write the transition tables + trigger constants**

Create `internal/state/presence/transitions.go`:

```go
package presence

import "github.com/GoMudEngine/GoMud/internal/state"

// playerTransitions enforces the Presence invariant matrix for player actors.
//
//	Connecting --in-world (character entered room)--> Active
//	Active     --N rounds no input------------------> Idle
//	Idle       --M rounds no input------------------> AFK
//	AFK        --hours OR TCP timeout---------------> Disconnected
//
//	[any non-Disconnected] --input received---------> Active
//	[any non-Connecting]   --manual `afk` cmd-------> AFK
//	[any]                  --TCP closed-------------> Disconnected
var playerTransitions = state.TransitionTable[State]{
	Connecting:   {Active, Disconnected},
	Active:       {Idle, AFK, Disconnected},
	Idle:         {Active, AFK, Disconnected},
	AFK:          {Active, Disconnected},
	Disconnected: {}, // terminal until reconnect (new Machine instance)
}

// mobTransitions enforces the Presence invariant matrix for mob actors.
//
//	Spawning   --auto next tick-------------------> Active
//	Active     --bored N rounds && !essential-----> Dormant
//	Active     --zone has no nearby players-------> Dormant
//	Dormant    --player nearby OR attacked--------> Active
//	Dormant    --too long alone && !essential-----> Despawning
//	Despawning --next tick------------------------> (removed)
var mobTransitions = state.TransitionTable[State]{
	Spawning:   {Active},
	Active:     {Dormant, Despawning}, // Despawning direct from Active for force-despawn
	Dormant:    {Active, Despawning},
	Despawning: {}, // terminal — mob is removed on next tick
}

// Trigger reason constants. Used in state.TransitionReason.Trigger.
const (
	TriggerInputReceived     = "input_received"
	TriggerManualAFK         = "manual_afk"
	TriggerTimeoutIdle       = "timeout_idle"
	TriggerTimeoutAFK        = "timeout_afk"
	TriggerTimeoutDisconnect = "timeout_disconnect"
	TriggerTCPClosed         = "tcp_closed"
	TriggerEnteredRoom       = "entered_room"
	TriggerPlayerEntry       = "player_entry"
	TriggerAttacked          = "attacked"
	TriggerBored             = "bored"
	TriggerZoneEmpty         = "zone_empty"
	TriggerDormantTooLong    = "dormant_too_long"
	TriggerSpawnTickResolve  = "spawn_tick_resolve"
)
```

- [ ] **Step 3: Write the failing Behavior Matrix tests**

Create `internal/state/presence/presence_test.go`:

```go
package presence

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestStateString(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{Active, "Active"},
		{Connecting, "Connecting"},
		{Idle, "Idle"},
		{AFK, "AFK"},
		{Disconnected, "Disconnected"},
		{Spawning, "Spawning"},
		{Dormant, "Dormant"},
		{Despawning, "Despawning"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("State(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}

// PR-001: Player constructor → Connecting.
func TestPlayerPresence_InitialConnecting(t *testing.T) {
	m := NewPlayerPresence()
	if m.State() != Connecting {
		t.Errorf("NewPlayerPresence() = %v, want Connecting", m.State())
	}
}

// PR-002: Connecting → Active on entered_room.
func TestPlayerPresence_ConnectingToActive(t *testing.T) {
	m := NewPlayerPresence()
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom}); err != nil {
		t.Fatalf("TransitionTo(Active): %v", err)
	}
	if m.State() != Active {
		t.Errorf("after entered_room, state = %v, want Active", m.State())
	}
}

// PR-003: Active → Active on input is a no-op (or self-transition).
// The machine treats it as a valid transition (idempotent re-enter).
func TestPlayerPresence_ActiveToActiveOnInput(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	// Re-firing Active on input shouldn't error; consumer code may skip the
	// call if already Active, but the machine doesn't require it.
	if m.State() != Active {
		t.Errorf("Active after re-entry, state = %v, want Active", m.State())
	}
}

// PR-004: Active → Idle on timeout.
func TestPlayerPresence_ActiveToIdle(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	if err := m.TransitionTo(Idle, state.TransitionReason{Trigger: TriggerTimeoutIdle}); err != nil {
		t.Fatalf("TransitionTo(Idle): %v", err)
	}
	if m.State() != Idle {
		t.Errorf("after timeout_idle, state = %v, want Idle", m.State())
	}
}

// PR-005: Idle → Active on input.
func TestPlayerPresence_IdleToActiveOnInput(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	_ = m.TransitionTo(Idle, state.TransitionReason{Trigger: TriggerTimeoutIdle})
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerInputReceived}); err != nil {
		t.Fatalf("TransitionTo(Active): %v", err)
	}
	if m.State() != Active {
		t.Errorf("after input, state = %v, want Active", m.State())
	}
}

// PR-006: Idle → AFK on timeout.
func TestPlayerPresence_IdleToAFK(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	_ = m.TransitionTo(Idle, state.TransitionReason{Trigger: TriggerTimeoutIdle})
	if err := m.TransitionTo(AFK, state.TransitionReason{Trigger: TriggerTimeoutAFK}); err != nil {
		t.Fatalf("TransitionTo(AFK): %v", err)
	}
	if m.State() != AFK {
		t.Errorf("after timeout_afk, state = %v, want AFK", m.State())
	}
}

// PR-007: Manual `afk <message>` transitions to AFK with data.
func TestPlayerPresence_ManualAFKWithMessage(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	d := AFKData{Message: "lunch break", Manual: true}
	if err := m.TransitionToAFK(d, state.TransitionReason{Trigger: TriggerManualAFK}); err != nil {
		t.Fatalf("TransitionToAFK: %v", err)
	}
	if m.State() != AFK {
		t.Errorf("after manual afk, state = %v, want AFK", m.State())
	}
	got, ok := m.AFKData()
	if !ok {
		t.Fatalf("AFKData() ok = false, want true")
	}
	if got.Message != "lunch break" || !got.Manual {
		t.Errorf("AFKData() = %+v, want {lunch break, true}", got)
	}
}

// PR-008: AFK → Active on input clears AFKData.
func TestPlayerPresence_AFKToActiveClearsData(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	_ = m.TransitionToAFK(AFKData{Message: "brb", Manual: true},
		state.TransitionReason{Trigger: TriggerManualAFK})
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerInputReceived}); err != nil {
		t.Fatalf("TransitionTo(Active): %v", err)
	}
	if m.State() != Active {
		t.Errorf("state = %v, want Active", m.State())
	}
	if _, ok := m.AFKData(); ok {
		t.Errorf("AFKData() still present after Active transition; want cleared")
	}
}

// PR-009: AFK → Disconnected on timeout.
func TestPlayerPresence_AFKToDisconnected(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	_ = m.TransitionTo(AFK, state.TransitionReason{Trigger: TriggerTimeoutAFK})
	if err := m.TransitionTo(Disconnected, state.TransitionReason{Trigger: TriggerTimeoutDisconnect}); err != nil {
		t.Fatalf("TransitionTo(Disconnected): %v", err)
	}
	if m.State() != Disconnected {
		t.Errorf("state = %v, want Disconnected", m.State())
	}
}

// PR-010: Any state → Disconnected on tcp_closed.
func TestPlayerPresence_ActiveToDisconnectedOnTCPClose(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	if err := m.TransitionTo(Disconnected, state.TransitionReason{Trigger: TriggerTCPClosed}); err != nil {
		t.Fatalf("TransitionTo(Disconnected): %v", err)
	}
	if m.State() != Disconnected {
		t.Errorf("state = %v, want Disconnected", m.State())
	}
}

// PR-011: Disconnected is terminal — no transitions out.
func TestPlayerPresence_DisconnectedIsTerminal(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	_ = m.TransitionTo(Disconnected, state.TransitionReason{Trigger: TriggerTCPClosed})
	// Try to transition out — should error.
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerInputReceived}); err == nil {
		t.Errorf("TransitionTo(Active) from Disconnected: got nil err; want ErrInvalidTransition")
	}
}

// PR-020: Mob constructor → Spawning.
func TestMobPresence_InitialSpawning(t *testing.T) {
	m := NewMobPresence()
	if m.State() != Spawning {
		t.Errorf("NewMobPresence() = %v, want Spawning", m.State())
	}
}

// PR-021: Spawning → Active on next tick.
func TestMobPresence_SpawningToActive(t *testing.T) {
	m := NewMobPresence()
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve}); err != nil {
		t.Fatalf("TransitionTo(Active): %v", err)
	}
	if m.State() != Active {
		t.Errorf("after spawn_tick_resolve, state = %v, want Active", m.State())
	}
}

// PR-022 (no-essential path): Active → Dormant on bored.
func TestMobPresence_ActiveToDormantOnBored(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	if err := m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerBored}); err != nil {
		t.Fatalf("TransitionTo(Dormant): %v", err)
	}
	if m.State() != Dormant {
		t.Errorf("after bored, state = %v, want Dormant", m.State())
	}
}

// PR-023 (no-essential path): Active → Dormant on zone_empty.
func TestMobPresence_ActiveToDormantOnZoneEmpty(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	if err := m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerZoneEmpty}); err != nil {
		t.Fatalf("TransitionTo(Dormant): %v", err)
	}
	if m.State() != Dormant {
		t.Errorf("state = %v, want Dormant", m.State())
	}
}

// PR-024: Essential-mob veto on Active→Dormant.
func TestMobPresence_EssentialVetoBlocksDormant(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	// Simulate essential-mob veto.
	m.RegisterVeto(Active, Dormant, func(r state.TransitionReason) error {
		return &state.VetoError{HandlerName: "essential", Reason: "essential mob"}
	})
	err := m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerBored})
	if err == nil {
		t.Errorf("TransitionTo(Dormant) with veto: got nil err; want VetoError")
	}
	if m.State() != Active {
		t.Errorf("after vetoed transition, state = %v, want Active (unchanged)", m.State())
	}
}

// PR-025: Dormant → Active on player_entry.
func TestMobPresence_DormantToActiveOnPlayerEntry(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	_ = m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerBored})
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerPlayerEntry}); err != nil {
		t.Fatalf("TransitionTo(Active): %v", err)
	}
	if m.State() != Active {
		t.Errorf("after player_entry, state = %v, want Active", m.State())
	}
}

// PR-026: Dormant → Active on attacked.
func TestMobPresence_DormantToActiveOnAttacked(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	_ = m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerBored})
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerAttacked}); err != nil {
		t.Fatalf("TransitionTo(Active): %v", err)
	}
	if m.State() != Active {
		t.Errorf("after attacked, state = %v, want Active", m.State())
	}
}

// PR-027 (no-essential path): Dormant → Despawning after too long.
func TestMobPresence_DormantToDespawning(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	_ = m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerBored})
	if err := m.TransitionTo(Despawning, state.TransitionReason{Trigger: TriggerDormantTooLong}); err != nil {
		t.Fatalf("TransitionTo(Despawning): %v", err)
	}
	if m.State() != Despawning {
		t.Errorf("state = %v, want Despawning", m.State())
	}
}

// PR-028: Despawning is terminal.
func TestMobPresence_DespawningIsTerminal(t *testing.T) {
	m := NewMobPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerSpawnTickResolve})
	_ = m.TransitionTo(Dormant, state.TransitionReason{Trigger: TriggerBored})
	_ = m.TransitionTo(Despawning, state.TransitionReason{Trigger: TriggerDormantTooLong})
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerAttacked}); err == nil {
		t.Errorf("TransitionTo(Active) from Despawning: got nil err; want ErrInvalidTransition")
	}
}
```

- [ ] **Step 4: Run the tests and verify they fail**

Run: `go test ./internal/state/presence/... -v`
Expected: FAIL — `presence` package doesn't exist yet.

(After Step 1+2 implementations are in place, this step is the "is the package compiling now?" gate.)

- [ ] **Step 5: Run the tests and verify they pass**

Run: `go test ./internal/state/presence/... -v`
Expected: PASS — all 18+ unit tests green.

- [ ] **Step 6: Run the full state-machine test suite**

Run: `go test ./internal/state/...`
Expected: PASS — no regressions in other state packages.

- [ ] **Step 7: Commit**

```bash
git add internal/state/presence/
git commit -m "$(cat <<'EOF'
feat(state): T1 — Presence package with Behavior Matrix tests

New internal/state/presence package: single union enum (Active,
Connecting, Idle, AFK, Disconnected, Spawning, Dormant, Despawning)
with per-actor transition tables (NewPlayerPresence + NewMobPresence).
AFKData state-data struct for manual `afk <message>`. Behavior Matrix
unit tests for PR-001 through PR-028 (player + mob life cycles, essential
veto, terminal states).

Integration with Character, hooks, and call-site cutover comes in
subsequent tasks.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Wire `Character.Presence` field

**Files:**
- Modify: `internal/characters/character.go` (struct field + `New()` + `ResetForMobInstance()`)
- Modify: `internal/mobs/mobs.go` (`Validate()` to set mob presence)
- Test: `internal/characters/character_test.go` (extend if exists; otherwise create)

- [ ] **Step 1: Add `Presence` field to Character struct**

Open `internal/characters/character.go`. Find the struct definition (around line 95). Locate the existing state-machine fields (`CombatPhase`, `Life`, `Position`, `Awareness`, `Activity`, `Control`). Add a new field right after `Control` or in alphabetical order with the others:

```go
	// Presence is the canonical state machine for "is this character
	// meaningfully present?". Per-actor states (Player: Connecting /
	// Active / Idle / AFK / Disconnected; Mob: Spawning / Active /
	// Dormant / Despawning). See internal/state/presence/context.md.
	Presence                 *presence.Machine              `yaml:"-"`
```

Add the import at the top of the file:

```go
	"github.com/GoMudEngine/GoMud/internal/state/presence"
```

- [ ] **Step 2: Initialize Presence in `characters.New()`**

Find the `New()` function in `internal/characters/character.go`. It returns a player-default Character. After the other state machines are initialized (look for `c.CombatPhase = ...`, `c.Life = ...`), add:

```go
	c.Presence = presence.NewPlayerPresence()
```

- [ ] **Step 3: Reset Presence in `ResetForMobInstance`**

Find `ResetForMobInstance()` (around line 73). It currently clears `Life`, `CombatPhase`, `Position`, `Awareness`, `Activity`, `Control`. Add:

```go
	c.Presence = nil
```

This ensures a freshly shallow-copied mob instance gets its own machine (set in mob `Validate()` below).

- [ ] **Step 4: Initialize Presence in mob `Validate()`**

Open `internal/mobs/mobs.go`. Find the `Validate()` method on `Mob`. After the existing state-machine initialization (look for the place where `c.Position`, `c.CombatPhase`, etc. get initialized for the mob — search for `NewMachine` or `combatphase.NewMachine`), add:

```go
	if m.Character.Presence == nil {
		m.Character.Presence = presence.NewMobPresence()
	}
```

Add the import:

```go
	"github.com/GoMudEngine/GoMud/internal/state/presence"
```

- [ ] **Step 5: Build and verify compilation**

Run: `go build ./...`
Expected: clean build. No unused-import errors. No undeclared identifiers.

- [ ] **Step 6: Run the package tests**

Run: `go test ./internal/characters/... ./internal/mobs/...`
Expected: PASS — existing tests not affected. (No new test added in this task; coverage is via Task 1's package tests plus the integration tests in Task 4+.)

- [ ] **Step 7: Commit**

```bash
git add internal/characters/character.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(characters): T2 — wire Character.Presence field

Add Presence *presence.Machine field to Character. Initialize in
characters.New() with NewPlayerPresence(); reset in ResetForMobInstance;
initialize in mobs.Mob.Validate() with NewMobPresence().

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Config knobs

**Files:**
- Modify: `internal/configs/config.server.go` (add 5 new knobs)
- Modify: `internal/configs/config.memory.go` (remove `MaxMobBoredom` — but keep it as deprecated for compat in this task; final deletion in T12)
- Modify: `_datafiles/config.yaml` (add defaults so the validator picks them up; old `MaxMobBoredom` stays but unused after T11)

- [ ] **Step 1: Read existing config structure**

Read `internal/configs/config.server.go` to understand the existing `Server` struct shape. Look for the `Network.MaxIdleSeconds` knob — that's the closest analog. New knobs go in the `Server` block (or a new `Presence` sub-block; check existing structure to decide).

- [ ] **Step 2: Add the five new knobs**

In `internal/configs/config.server.go`, in the appropriate sub-struct (likely `Server` or a new `Presence` block — pick whichever matches the existing convention; if knobs are flat under `Server`, follow that), add:

```go
	PresenceIdleAfterRounds       ConfigInt `yaml:"PresenceIdleAfterRounds"`
	PresenceAFKAfterRounds        ConfigInt `yaml:"PresenceAFKAfterRounds"`
	PresenceDisconnectAfterRounds ConfigInt `yaml:"PresenceDisconnectAfterRounds"`
	PresenceMobDormantAfterRounds ConfigInt `yaml:"PresenceMobDormantAfterRounds"`
	PresenceMobDespawnAfterRounds ConfigInt `yaml:"PresenceMobDespawnAfterRounds"`
```

Find the `Validate()` method on the same struct and append default-setting clauses:

```go
	if s.PresenceIdleAfterRounds < 1 {
		s.PresenceIdleAfterRounds = 8 // ~30s at 4s/round
	}
	if s.PresenceAFKAfterRounds < 1 {
		s.PresenceAFKAfterRounds = 75 // ~5min
	}
	if s.PresenceDisconnectAfterRounds < 1 {
		s.PresenceDisconnectAfterRounds = 900 // ~1h
	}
	if s.PresenceMobDormantAfterRounds < 1 {
		s.PresenceMobDormantAfterRounds = 30
	}
	if s.PresenceMobDespawnAfterRounds < 1 {
		s.PresenceMobDespawnAfterRounds = 60
	}
```

- [ ] **Step 3: Add a `GetConfig()` accessor if not present**

The existing pattern (per `config.memory.go:31`) is:

```go
func GetServerConfig() Server {
	configDataLock.RLock()
	defer configDataLock.RUnlock()
	if !configData.validated {
		configData.Validate()
	}
	return configData.Server
}
```

If the accessor already exists, skip this step.

- [ ] **Step 4: Add the new knobs to `_datafiles/config.yaml`**

Open `_datafiles/config.yaml`. Find the `Server:` section. Add the five new knobs with their defaults under the appropriate location (or under a new `Presence:` sub-section if you created one in Step 2):

```yaml
  # Presence machine (chunk 5 — 2026-05-19) — replaces MaxMobBoredom +
  # ad-hoc AFK computation. See internal/state/presence/context.md.
  PresenceIdleAfterRounds: 8         # Active → Idle (~30s)
  PresenceAFKAfterRounds: 75         # Idle → AFK (~5min)
  PresenceDisconnectAfterRounds: 900 # AFK → Disconnected (~1h)
  PresenceMobDormantAfterRounds: 30  # mob Active → Dormant
  PresenceMobDespawnAfterRounds: 60  # mob Dormant → Despawning
```

(Note: `MaxMobBoredom` stays untouched here. T12 removes it after the call-site cutover lands.)

- [ ] **Step 5: Build and verify**

Run: `go build ./...`
Expected: clean build.

Run: `go test ./internal/configs/...`
Expected: PASS — Validate() defaults kick in.

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.server.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(configs): T3 — Presence threshold knobs

Five new server-config knobs gating the Presence machine's timeout
transitions: PresenceIdleAfterRounds (8), PresenceAFKAfterRounds (75),
PresenceDisconnectAfterRounds (900), PresenceMobDormantAfterRounds (30),
PresenceMobDespawnAfterRounds (60). Defaults applied in Validate(); YAML
seeded with same. MaxMobBoredom stays for compat; final deletion in T12.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: `NewRound_PresenceTick` round-driver hook

**Files:**
- Create: `internal/hooks/NewRound_PresenceTick.go`
- Modify: `internal/hooks/hooks.go` (register the listener)

- [ ] **Step 1: Create the hook file**

Create `internal/hooks/NewRound_PresenceTick.go`:

```go
// Round ticks for Presence transitions.
//
// Chunk 5 (2026-05-19): centralized timeout-driven transitions for both
// players (Active → Idle → AFK → Disconnected) and mobs (Active →
// Dormant → Despawning). Replaces scattered AFK checks in userrecord.go
// and the BoredomCounter increment path in lookfortrouble.go.
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PresenceTick walks all active characters once per round and fires
// timeout-driven Presence transitions. Ordering: must run AFTER
// DoCombat (so attack-driven Dormant→Active fires before this checks
// timeouts) and BEFORE IdleMobs (so Despawning mobs get their
// terminal-tick removal in the same round).
func PresenceTick(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)
	srvCfg := configs.GetServerConfig()
	roundNow := evt.RoundNumber

	// === Player presence transitions ===
	idleAfter := uint64(srvCfg.PresenceIdleAfterRounds)
	afkAfter := uint64(srvCfg.PresenceAFKAfterRounds)
	disconnectAfter := uint64(srvCfg.PresenceDisconnectAfterRounds)

	for _, user := range users.GetAllActiveUsers() {
		if user.Character == nil || user.Character.Presence == nil {
			continue
		}
		li := user.GetLastInputRound()
		if li == 0 || roundNow < li {
			continue
		}
		elapsed := roundNow - li

		switch user.Character.Presence.State() {
		case presence.Active:
			if elapsed >= idleAfter {
				_ = user.Character.Presence.TransitionTo(presence.Idle,
					state.TransitionReason{Trigger: presence.TriggerTimeoutIdle})
			}
		case presence.Idle:
			if elapsed >= afkAfter {
				_ = user.Character.Presence.TransitionTo(presence.AFK,
					state.TransitionReason{Trigger: presence.TriggerTimeoutAFK})
			}
		case presence.AFK:
			if elapsed >= disconnectAfter {
				_ = user.Character.Presence.TransitionTo(presence.Disconnected,
					state.TransitionReason{Trigger: presence.TriggerTimeoutDisconnect})
			}
		}
	}

	// === Mob presence transitions ===
	dormantAfter := uint64(srvCfg.PresenceMobDormantAfterRounds)
	despawnAfter := uint64(srvCfg.PresenceMobDespawnAfterRounds)

	for _, mobId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(mobId)
		if mob == nil || mob.Character.Presence == nil {
			continue
		}
		switch mob.Character.Presence.State() {
		case presence.Spawning:
			// Same-tick resolution → Active.
			_ = mob.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerSpawnTickResolve})

		case presence.Active:
			// Bored check: rounds since last target found.
			ltf := mob.Character.LastTargetFoundRound
			if ltf > 0 && roundNow-ltf >= dormantAfter {
				_ = mob.Character.Presence.TransitionTo(presence.Dormant,
					state.TransitionReason{Trigger: presence.TriggerBored})
			}

		case presence.Dormant:
			// Despawn check: rounds since entering Dormant.
			// Use LastDormantEntryRound on Character; if 0, set it now.
			if mob.Character.LastDormantEntryRound == 0 {
				mob.Character.LastDormantEntryRound = roundNow
				continue
			}
			if roundNow-mob.Character.LastDormantEntryRound >= despawnAfter {
				_ = mob.Character.Presence.TransitionTo(presence.Despawning,
					state.TransitionReason{Trigger: presence.TriggerDormantTooLong})
			}
		}
	}

	return events.Continue
}
```

- [ ] **Step 2: Add the two new Character fields needed by the hook**

Open `internal/characters/character.go`. Add two fields next to the other `yaml:"-"` runtime fields:

```go
	// LastTargetFoundRound tracks the round number when this character
	// last found a combat target. Used by Presence.PresenceTick to
	// determine when a mob is "bored". Replaces Mob.BoredomCounter.
	LastTargetFoundRound uint64 `yaml:"-"`

	// LastDormantEntryRound tracks when this character entered
	// Presence.Dormant. Used by Presence.PresenceTick to determine
	// when to transition to Despawning.
	LastDormantEntryRound uint64 `yaml:"-"`
```

Reset both in `ResetForMobInstance()`:

```go
	c.LastTargetFoundRound = 0
	c.LastDormantEntryRound = 0
```

- [ ] **Step 3: Register the listener in `hooks.go`**

Open `internal/hooks/hooks.go`. Find the NewRound listener block (around line 19-38). Add the new listener BETWEEN `DoCombat` (line 32) and `IdleMobs` (line 38):

```go
	events.RegisterListener(events.NewRound{}, DoCombat)
	//
	// Done with combat
	//
	events.RegisterListener(events.NewRound{}, PresenceTick) // NEW: chunk 5
	events.RegisterListener(events.NewRound{}, AutoHeal)
```

Ordering rationale (from spec §5):
- After `DoCombat`: an attack that landed this round already wake-transitioned Dormant→Active in the attack path; PresenceTick won't re-bounce it.
- Before `IdleMobs`: Despawning mobs get terminal-tick removal in the same round (T11 wires that).

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean build.

Run: `go test ./internal/hooks/...`
Expected: PASS — no regressions.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_PresenceTick.go internal/hooks/hooks.go \
        internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat(hooks): T4 — NewRound_PresenceTick driver hook

New round-driver hook walks all active players + mobs once per round,
fires timeout-driven Presence transitions. Player: Active→Idle→AFK→
Disconnected. Mob: Spawning→Active and Active→Dormant→Despawning. Two
new Character fields (LastTargetFoundRound, LastDormantEntryRound)
replace Mob.BoredomCounter timing.

Hook registered between DoCombat and IdleMobs so attack-driven wakes
fire first and Despawning mobs get terminal-tick removal next.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Essential-mob veto on Dormant/Despawning

**Files:**
- Modify: `internal/mobs/mobs.go` (`Validate()` — register vetoes after `NewMobPresence()`)
- Test: `internal/state/presence/integration_test.go` (new — exercises veto with a real Mob)

- [ ] **Step 1: Write the failing veto integration test**

Create `internal/state/presence/integration_test.go`:

```go
package presence_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
)

// TestEssentialVeto_BlocksActiveToDormant verifies that a registered
// veto on Active→Dormant returns ErrVetoed when fired.
func TestEssentialVeto_BlocksActiveToDormant(t *testing.T) {
	m := presence.NewMobPresence()
	_ = m.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerSpawnTickResolve})

	isEssential := true
	m.RegisterVeto(presence.Active, presence.Dormant, func(r state.TransitionReason) error {
		if isEssential {
			return &state.VetoError{HandlerName: "essential", Reason: "essential mob"}
		}
		return nil
	})

	if err := m.TransitionTo(presence.Dormant,
		state.TransitionReason{Trigger: presence.TriggerBored}); err == nil {
		t.Errorf("TransitionTo(Dormant) with essential veto: got nil; want VetoError")
	}
	if m.State() != presence.Active {
		t.Errorf("state after vetoed transition = %v, want Active", m.State())
	}
}

// TestEssentialVeto_BlocksActiveToDespawning verifies that a registered
// veto on Active→Despawning returns ErrVetoed when fired.
func TestEssentialVeto_BlocksActiveToDespawning(t *testing.T) {
	m := presence.NewMobPresence()
	_ = m.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerSpawnTickResolve})

	m.RegisterVeto(presence.Active, presence.Despawning, func(r state.TransitionReason) error {
		return &state.VetoError{HandlerName: "essential", Reason: "essential mob"}
	})

	if err := m.TransitionTo(presence.Despawning,
		state.TransitionReason{Trigger: presence.TriggerDormantTooLong}); err == nil {
		t.Errorf("TransitionTo(Despawning) with essential veto: got nil; want VetoError")
	}
}
```

- [ ] **Step 2: Run the test to verify it passes (with the veto registration in place)**

Run: `go test ./internal/state/presence/... -run TestEssentialVeto -v`
Expected: PASS — the framework's veto mechanism is already in place from T1.

- [ ] **Step 3: Register the vetoes in mob `Validate()`**

Open `internal/mobs/mobs.go`. After the `Presence = NewMobPresence()` line from T2, register the vetoes:

```go
	if m.Character.Presence == nil {
		m.Character.Presence = presence.NewMobPresence()
		// Essential-mob veto: shopkeepers, foragers, caravan crew, and
		// charmed companions must never transition out of Active. Wraps
		// the existing Despawns() + IsCharmed() + IsEssential() policy.
		essentialVeto := func(r state.TransitionReason) error {
			if !m.Despawns() || m.IsEssential() || m.Character.IsCharmed() {
				return &state.VetoError{
					HandlerName: "essential_mob",
					Reason:      "essential mob (shop/forager/caravan/charmed)",
				}
			}
			return nil
		}
		m.Character.Presence.RegisterVeto(presence.Active, presence.Dormant, essentialVeto)
		m.Character.Presence.RegisterVeto(presence.Active, presence.Despawning, essentialVeto)
		m.Character.Presence.RegisterVeto(presence.Dormant, presence.Despawning, essentialVeto)
	}
```

Add the `state` import if not present:

```go
	"github.com/GoMudEngine/GoMud/internal/state"
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean build.

Run: `go test ./internal/mobs/... ./internal/state/presence/...`
Expected: PASS — essential mobs are now veto-protected.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go internal/state/presence/integration_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): T5 — essential-mob veto on Presence.Dormant/Despawning

Register vetoes on Active→Dormant, Active→Despawning, and Dormant→
Despawning in mob Validate(). Veto policy wraps the existing
Despawns() + IsEssential() + IsCharmed() predicates so shopkeepers,
foragers, caravan crew, and charmed companions stay Active permanently.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: CombatPhase veto on `Idle→Engaging`

**Files:**
- Modify: `internal/state/combatphase/combatphase.go` (the existing `targetPresence` veto slot needs to be populated by engine setup)
- Modify: `internal/characters/character.go` (the place where CombatPhase machine is constructed — wire the `targetPresence` veto to check the target's `Presence.State()`)
- Test: extend `internal/state/presence/integration_test.go`

- [ ] **Step 1: Locate the CombatPhase setup site**

Open `internal/characters/character.go` and find where `c.CombatPhase` is initialized. There's an existing setup that wires the veto chain — find it and study which vetoes are populated today. (`grep -n "vetoes\." internal/state/combatphase/combatphase.go` shows the field list around line 60-69, and `grep -n "WireCombatPhaseVetoes\|combatPhaseWired" internal/characters/*.go` finds the wiring site.)

- [ ] **Step 2: Wire the targetPresence veto**

In whatever helper sets up `c.CombatPhase`'s veto chain (likely a method like `c.wireCombatPhaseVetoes()`), add the Presence-checking veto:

```go
	// Chunk 5 (Presence): block Engaging if target is Disconnected or
	// Despawning. AFK / Idle / Dormant targets ARE attackable (Dormant
	// auto-wakes to Active on attack via the wake-on-attack hook in T7).
	machine.SetTargetPresenceVeto(func(target state.ActorRef) bool {
		targetChar := lookupCharacterByRef(target) // existing helper
		if targetChar == nil || targetChar.Presence == nil {
			return true
		}
		switch targetChar.Presence.State() {
		case presence.Disconnected, presence.Despawning:
			return false
		}
		return true
	})
```

Note: if the existing CombatPhase machine API doesn't yet have a `SetTargetPresenceVeto` setter, add a minimal one in `internal/state/combatphase/combatphase.go`:

```go
// SetTargetPresenceVeto installs the Presence veto for target validity.
// Wired by character setup once Presence machines are attached.
func (m *Machine) SetTargetPresenceVeto(fn func(state.ActorRef) bool) {
	m.vetoes.targetPresence = fn
}
```

- [ ] **Step 3: Add an integration test verifying the veto blocks**

In `internal/state/presence/integration_test.go`, append:

```go
// TestCombatPhaseVeto_BlocksOnDisconnected verifies a target in
// Presence.Disconnected blocks the attacker's CombatPhase Idle→Engaging.
// (This is an integration smoke; the full CombatPhase+Presence wiring
// is tested in internal/characters/... after the engine setup lands.)
func TestCombatPhaseVeto_BlocksOnDisconnected(t *testing.T) {
	// Simulate the veto-eligibility check: a Disconnected target
	// should produce a veto.
	target := presence.NewPlayerPresence()
	_ = target.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = target.TransitionTo(presence.Disconnected, state.TransitionReason{Trigger: presence.TriggerTCPClosed})

	allowed := target.State() != presence.Disconnected && target.State() != presence.Despawning
	if allowed {
		t.Errorf("Disconnected target should be vetoed; got allowed=true")
	}
}

// TestCombatPhaseVeto_AllowsOnAFK verifies AFK target is ATTACKABLE.
func TestCombatPhaseVeto_AllowsOnAFK(t *testing.T) {
	target := presence.NewPlayerPresence()
	_ = target.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = target.TransitionTo(presence.AFK, state.TransitionReason{Trigger: presence.TriggerTimeoutAFK})

	allowed := target.State() != presence.Disconnected && target.State() != presence.Despawning
	if !allowed {
		t.Errorf("AFK target should be attackable; got allowed=false")
	}
}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean build.

Run: `go test ./internal/state/presence/... ./internal/state/combatphase/... ./internal/characters/...`
Expected: PASS — veto wiring in place, existing CombatPhase tests still green.

- [ ] **Step 5: Commit**

```bash
git add internal/state/combatphase/combatphase.go internal/characters/character.go \
        internal/state/presence/integration_test.go
git commit -m "$(cat <<'EOF'
feat(combatphase): T6 — Presence veto on Idle→Engaging

Wire the existing targetPresence veto slot on CombatPhase.Machine. Block
list: target.Presence.State() ∈ {Disconnected, Despawning}. AFK / Idle /
Dormant targets ARE attackable (Dormant wakes on attack via T7's wake
hook). Per design "if you went AFK in a dangerous room, you deserve
it."

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Auto-wake Dormant mobs on attack

**Files:**
- Modify: target-resolution path (most likely `internal/actions/actor.go` or wherever `ResolveTargetActor` lives — verify before editing)
- OR (alternative): `internal/hooks/NewRound_DoCombat_helpers.go` — wherever the per-swing target reads `mob` and applies damage

- [ ] **Step 1: Find the lowest-common-ancestor attack site**

Run: `grep -rn "ResolveTargetActor\|TargetActorRef\|func.*resolveAttack" internal/actions internal/combat internal/hooks 2>&1 | head -20`

Look for the single point in the codebase where:
1. An attacker's target has been resolved to a concrete `*Mob` or `*UserRecord`
2. Damage is about to be applied
3. Both player→mob and mob→mob paths go through this point

The most likely candidates (verify by reading):
- `internal/actions/actor.go` (if it has the resolution helper)
- `internal/hooks/NewRound_DoCombat.go` (if combat resolves there)
- `internal/combat/combat.go` (if damage application lives there)

- [ ] **Step 2: Add the wake transition at the call site**

Once the LCA is identified, insert (before damage application):

```go
// Chunk 5 (Presence): auto-wake Dormant mobs on incoming attack.
// The mob's per-round tick was being skipped; combat receivability
// stays intact. Wake fires BEFORE damage so the target is Active when
// per-round logic runs.
if targetChar != nil && targetChar.Presence != nil &&
	targetChar.Presence.State() == presence.Dormant {
	_ = targetChar.Presence.TransitionTo(presence.Active,
		state.TransitionReason{Trigger: presence.TriggerAttacked, Actor: attackerRef})
	targetChar.LastDormantEntryRound = 0 // reset so the next Active→Dormant timer restarts fresh
}
```

Imports as needed:
```go
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
```

- [ ] **Step 3: Add an integration test for the wake**

Either extend `integration_test.go` or add a hook-level test in `internal/hooks/NewRound_PresenceTick_test.go` (if the wake site is in the hooks package). The minimal coverage is: target starts Dormant, attack arrives, target ends Active.

```go
// TestDormantWake verifies a Dormant mob transitions to Active when
// it would be considered for an attack.
func TestDormantWake(t *testing.T) {
	m := presence.NewMobPresence()
	_ = m.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerSpawnTickResolve})
	_ = m.TransitionTo(presence.Dormant, state.TransitionReason{Trigger: presence.TriggerBored})

	// Simulate the wake check that the attack-resolution path will do:
	if m.State() == presence.Dormant {
		_ = m.TransitionTo(presence.Active,
			state.TransitionReason{Trigger: presence.TriggerAttacked})
	}
	if m.State() != presence.Active {
		t.Errorf("after attack wake, state = %v, want Active", m.State())
	}
}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean.

Run: `go test ./...`
Expected: PASS — no regressions.

- [ ] **Step 5: Commit**

```bash
git add <path-to-attack-resolution-file> internal/state/presence/integration_test.go
git commit -m "$(cat <<'EOF'
feat(combat): T7 — auto-wake Dormant mobs on incoming attack

In the target-resolution path (lowest-common-ancestor of player→mob and
mob→mob attacks), transition any Dormant target back to Active before
damage applies. Resets LastDormantEntryRound so the next Active→Dormant
timer starts fresh. Receivability was never skipped — only the mob's
own per-round tick was.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Scheduler observer — cancel on terminal-state entry

**Files:**
- Modify: `internal/mobs/mobs.go` and/or `internal/users/userrecord.go` (wherever the Presence machine is constructed) — register an observer that calls `RoundScheduler.CancelAllFor(self)` on entry to Disconnected or Despawning
- Test: extend `internal/state/presence/integration_test.go`

- [ ] **Step 1: Read the framework's scheduler API**

Open `internal/state/scheduled.go` and find the `RoundScheduler` type. Look for the method that cancels all scheduled transitions for a given character — most likely `CancelAllFor(ref state.ActorRef)`.

If the framework's API doesn't yet have `CancelAllFor`, that's a small framework addition needed first; add the method to `RoundScheduler` with a test in `scheduled_test.go`. (This is a one-line method using existing internal maps.)

- [ ] **Step 2: Register the observer when the Presence machine is created**

In `internal/mobs/mobs.go` (where `NewMobPresence()` is called) and `internal/characters/character.go` (where `NewPlayerPresence()` is called), register the observer immediately after construction:

```go
	m.Character.Presence.RegisterObserver(func(evt state.Event) {
		if evt.To == int(presence.Disconnected) || evt.To == int(presence.Despawning) {
			state.GlobalScheduler.CancelAllFor(m.Character.Presence.Self())
		}
	})
```

The exact API call shape depends on the existing framework — adjust based on whatever `RoundScheduler` exposes today.

(Equivalent block in `characters.New()` for player characters.)

- [ ] **Step 3: Add a test verifying scheduled transitions cancel on terminal entry**

In `internal/state/presence/integration_test.go`, append:

```go
// TestSchedulerCancelOnTerminal verifies that entry to Disconnected
// or Despawning triggers RoundScheduler.CancelAllFor.
func TestSchedulerCancelOnTerminal(t *testing.T) {
	// (This is mostly a smoke test for the observer being registered;
	// the actual scheduler-cancel behavior lives in the scheduler
	// package's own tests.)
	cancelled := false
	m := presence.NewPlayerPresence()
	m.RegisterObserver(func(evt state.Event) {
		// In production this calls scheduler.CancelAllFor; the test
		// just verifies the observer fires on terminal entry.
		if evt.To == int(presence.Disconnected) {
			cancelled = true
		}
	})
	_ = m.TransitionTo(presence.Active,
		state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = m.TransitionTo(presence.Disconnected,
		state.TransitionReason{Trigger: presence.TriggerTCPClosed})
	if !cancelled {
		t.Errorf("observer did not fire on Disconnected entry")
	}
}
```

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: clean.

Run: `go test ./internal/state/... ./internal/mobs/... ./internal/users/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go internal/characters/character.go \
        internal/state/scheduled.go internal/state/presence/integration_test.go
git commit -m "$(cat <<'EOF'
feat(state): T8 — scheduler cancel on Presence terminal-state entry

Register a Presence observer at machine-construction time. On entry to
Disconnected (player) or Despawning (mob), the observer calls
state.GlobalScheduler.CancelAllFor(self) to wipe pending scheduled
transitions across ALL of the character's machines (Activity casting
timers, Position recovery timers, etc.).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Connection lifecycle — login → Connecting → Active, TCP close → Disconnected

**Files:**
- Modify: `internal/users/users.go` (login + logout paths)
- Modify: connection-management code (wherever the TCP-close event is detected)

- [ ] **Step 1: Locate the login completion site**

Find where a player's character is placed in their starting room after authentication. The natural transition site for `Connecting → Active`. Likely candidates:

```
grep -n "MoveToRoom\|loginCharacterTo\|enterRoom" internal/users/users.go internal/inputhandlers/*.go
```

- [ ] **Step 2: Fire `Active` transition at login completion**

Once located, add (after the character is placed in the room):

```go
	// Chunk 5 (Presence): newly-joined character transitions Connecting → Active.
	if user.Character != nil && user.Character.Presence != nil {
		_ = user.Character.Presence.TransitionTo(presence.Active,
			state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	}
```

- [ ] **Step 3: Locate the TCP-close path**

```
grep -rn "ConnectionClosed\|LogOutUserByConnectionId\|disconnectUser" internal/users internal/connections 2>&1 | head -10
```

- [ ] **Step 4: Fire `Disconnected` transition on TCP close**

In the disconnect handler, before the user is removed from the active list:

```go
	if user.Character != nil && user.Character.Presence != nil {
		_ = user.Character.Presence.TransitionTo(presence.Disconnected,
			state.TransitionReason{Trigger: presence.TriggerTCPClosed})
	}
```

- [ ] **Step 5: Update `SetLastInputRound` to fire `Input` transition**

Open `internal/users/userrecord.go`. Find `SetLastInputRound()` (around line 157). After setting the round, fire a transition if the user is currently in Idle or AFK:

```go
func (u *UserRecord) SetLastInputRound(rdNum uint64) {
	u.lastInputRound = rdNum

	// Chunk 5 (Presence): any input resets to Active.
	if u.Character != nil && u.Character.Presence != nil {
		switch u.Character.Presence.State() {
		case presence.Idle, presence.AFK:
			_ = u.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerInputReceived})
		}
	}
}
```

- [ ] **Step 6: Build and verify**

Run: `go build ./...`
Expected: clean.

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/users/users.go internal/users/userrecord.go <any other touched files>
git commit -m "$(cat <<'EOF'
feat(users): T9 — Presence connection lifecycle integration

Login completion fires Presence.TransitionTo(Active) on character
entry to starting room. TCP close fires Presence.TransitionTo(Disconnected).
SetLastInputRound transitions Idle/AFK→Active on any input.

The Connecting state is set by the Machine constructor (T1); this task
just wires the transitions out of it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Rewrite `afk` command

**Files:**
- Modify: `internal/usercommands/afk.go` (rewrite to use Presence machine)
- Modify: `internal/usercommands/usercommands.go:294-297` (drop the ManualAFK clear-on-next-cmd shim)

- [ ] **Step 1: Rewrite `afk` command**

Replace `internal/usercommands/afk.go` with:

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func AFK(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character == nil || user.Character.Presence == nil {
		return true, nil // safety
	}

	currentState := user.Character.Presence.State()

	// Toggle off if already AFK (manual) and no message argument.
	if currentState == presence.AFK && rest == "" {
		if d, ok := user.Character.Presence.AFKData(); ok && d.Manual {
			_ = user.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerInputReceived})
			user.SendText(`You are no longer AFK.`)
			room.SendTextVisual(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> is back.`,
				user.Character.Name), user.UserId)
			return true, nil
		}
	}

	// Set AFK with optional message.
	msg := strings.TrimSpace(rest)
	_ = user.Character.Presence.TransitionToAFK(
		presence.AFKData{Message: msg, Manual: true},
		state.TransitionReason{Trigger: presence.TriggerManualAFK})

	if msg != "" {
		user.SendText(fmt.Sprintf(`You are now AFK: %s`, msg))
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> goes AFK: %s`,
			user.Character.Name, msg), user.UserId)
	} else {
		user.SendText(`You are now AFK. Type <ansi fg="command">afk</ansi> again to return.`)
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> goes AFK.`,
			user.Character.Name), user.UserId)
	}

	return true, nil
}
```

- [ ] **Step 2: Drop the ManualAFK clear-on-next-cmd shim**

Open `internal/usercommands/usercommands.go`. Find lines 294-297:

```go
	if user.ManualAFK && cmd != "afk" {
		user.ManualAFK = false
		user.AFKMessage = ""
		user.SendText(`<ansi fg="8">You are no longer AFK.</ansi>`)
```

Delete these four lines. The `SetLastInputRound` transition installed in T9 already covers the AFK→Active path on any command via the Presence machine's observer.

Verify the surrounding control flow still makes sense — if there's an `}` closing the deleted block, drop that too.

- [ ] **Step 3: Build and verify**

Run: `go build ./...`
Expected: clean.

Run: `go test ./internal/usercommands/...`
Expected: PASS — existing AFK tests may need updates; if they reference `user.ManualAFK`, they should now read `user.Character.Presence.State() == presence.AFK`.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/afk.go internal/usercommands/usercommands.go
git commit -m "$(cat <<'EOF'
feat(usercommands): T10 — `afk` command uses Presence machine

Rewrite afk.go to transition the Presence machine to AFK with AFKData
(Manual=true, Message=text). Re-entering the command transitions back
to Active. Drops the legacy ManualAFK clear-on-next-cmd shim from
usercommands.go:294-297 (subsumed by SetLastInputRound's Idle/AFK→Active
transition installed in T9).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Cutover call sites

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs.go` (replace BoredomCounter check with Despawning state check)
- Modify: `internal/mobcommands/lookfortrouble.go` (replace BoredomCounter++ with LastTargetFoundRound tracking)
- Modify: `internal/rooms/rooms.go:2144` (replace BoredomCounter reset with Presence-based logic)
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go` (drop BoredomCounter resets; rely on Presence transitions)
- Modify: `internal/rooms/roomdetails.go:267-271` (replace user.ManualAFK with Presence AFKData lookup)
- Modify: `internal/users/userrecord.go:543-546` (replace ad-hoc isAfk with Presence.State() == AFK)
- Create: `internal/hooks/RoomChange_PresencePlayerEntry.go` (new observer firing Dormant→Active when a player enters a room with Dormant mobs)
- Modify: `internal/hooks/hooks.go` (register the new RoomChange listener)

This task is the bulk of the existing-call-site cutover. Treat each sub-step as its own logical unit; commit after each rather than at the end.

- [ ] **Step 1: Replace BoredomCounter check in `NewRound_IdleMobs.go`**

Open `internal/hooks/NewRound_IdleMobs.go`. Find lines 44-55:

```go
		if allowedUnloadCt > 0 && mob.BoredomCounter >= maxBoredom {
			if mob.Despawns() {
				mob.Command(`despawn` + fmt.Sprintf(` depression %d/%d`, mob.BoredomCounter, maxBoredom))
				allowedUnloadCt--
			} else {
				mob.BoredomCounter = 0
			}
			continue
		}
```

Replace with:

```go
		// Chunk 5 (Presence): despawn driven by Presence state, not BoredomCounter.
		// PresenceTick has already transitioned the mob to Despawning; this hook
		// fires the actual removal on the next tick.
		if mob.Character.Presence != nil && mob.Character.Presence.State() == presence.Despawning {
			if allowedUnloadCt > 0 {
				mob.Command(`despawn presence_despawning`)
				allowedUnloadCt--
			}
			continue
		}
```

Drop the no-longer-needed `maxBoredom` and `mc := configs.GetMemoryConfig()` if unused. Add the presence import.

- [ ] **Step 2: Drop `PreventIdle` block in `NewRound_IdleMobs.go:58-61`**

```go
		// If idle prevented, it's a one round interrupt (until another comes along)
		if mob.PreventIdle {
			mob.PreventIdle = false
			continue
		}
```

Delete this entire block. PreventIdle is subsumed by the Presence Active state.

Commit this sub-step before continuing.

- [ ] **Step 3: Replace BoredomCounter increment in `lookfortrouble.go`**

Open `internal/mobcommands/lookfortrouble.go`. Find lines 185-187:

```go
		if mob.Despawns() {
			if mob.BoredomCounter < 255 {
				mob.BoredomCounter++
			}
		}
```

Replace with a no-op (the timing is now driven by `LastTargetFoundRound`, set elsewhere). Delete the block entirely — the Presence machine derives "bored" from "rounds since LastTargetFoundRound was updated", which happens when a target IS found (not when one ISN'T):

```go
		// Chunk 5 (Presence): "bored" timer is now driven by
		// LastTargetFoundRound, updated below when a target IS found.
		// No counter to increment here.
```

Then, in the SAME function, find the place where `mob.Command(...)` is called after a target is selected (i.e. when the look-for-trouble loop finds something to attack). Add:

```go
		mob.Character.LastTargetFoundRound = util.GetRoundCount()
```

Commit this sub-step.

- [ ] **Step 4: Replace BoredomCounter reset in `rooms.go:2144`**

Open `internal/rooms/rooms.go`. Find the line `mob.BoredomCounter = 0` (around 2144). Replace with:

```go
		// Chunk 5 (Presence): player entering a room wakes Dormant mobs.
		if mob.Character.Presence != nil && mob.Character.Presence.State() == presence.Dormant {
			_ = mob.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerPlayerEntry})
		}
```

Add the imports.

Commit.

- [ ] **Step 5: Drop BoredomCounter resets in `MobIdle_HandleIdleMobs.go`**

Open `internal/hooks/MobIdle_HandleIdleMobs.go`. Grep for `BoredomCounter` references and drop any remaining ones (any `BoredomCounter = 0` reset patterns are now subsumed by the Presence transitions).

Commit.

- [ ] **Step 6: Replace `user.ManualAFK` reads in `roomdetails.go`**

Open `internal/rooms/roomdetails.go`. Find lines 267-271:

```go
		if player.ManualAFK {
			if player.AFKMessage != "" {
				playerEntry += ` <ansi fg="8">(AFK: ` + player.AFKMessage + `)</ansi>`
			} else {
				playerEntry += ` <ansi fg="8">(AFK)</ansi>`
			}
		}
```

Replace with:

```go
		if player.Character != nil && player.Character.Presence != nil {
			if d, ok := player.Character.Presence.AFKData(); ok && d.Manual {
				if d.Message != "" {
					playerEntry += ` <ansi fg="8">(AFK: ` + d.Message + `)</ansi>`
				} else {
					playerEntry += ` <ansi fg="8">(AFK)</ansi>`
				}
			}
		}
```

Commit.

- [ ] **Step 7: Replace ad-hoc isAfk in `userrecord.go:543`**

Open `internal/users/userrecord.go`. Find lines 543-546:

```go
	isAfk := u.ManualAFK
	if !isAfk && afkRounds > 0 && roundNow-u.GetLastInputRound() >= afkRounds {
		isAfk = true
	}
```

Replace with:

```go
	// Chunk 5 (Presence): IsAFK shim reads the canonical Presence state.
	isAfk := false
	if u.Character != nil && u.Character.Presence != nil {
		isAfk = u.Character.Presence.State() == presence.AFK
	}
	_ = afkRounds // legacy var; remove if unused elsewhere
```

If `afkRounds` is unused after this change, drop its declaration.

Commit.

- [ ] **Step 8: New `RoomChange_PresencePlayerEntry.go` observer**

Some mobs in the destination room may be Dormant and need to wake when a player enters. The room-change event is the natural trigger.

Create `internal/hooks/RoomChange_PresencePlayerEntry.go`:

```go
// Wake any Dormant mobs in the destination room when a player enters.
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func PresencePlayerEntry(e events.Event) events.ListenerReturn {
	evt := e.(events.RoomChange)

	// Only fire on PLAYER entry (mobs entering rooms don't wake other mobs
	// for free — only nearby players do).
	if evt.UserId == 0 {
		return events.Continue
	}
	u := users.GetByUserId(evt.UserId)
	if u == nil {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.ToRoomId)
	if room == nil {
		return events.Continue
	}

	for _, mobInstId := range room.GetMobs() {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Presence == nil {
			continue
		}
		if mob.Character.Presence.State() == presence.Dormant {
			_ = mob.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerPlayerEntry})
			mob.Character.LastDormantEntryRound = 0
		}
	}

	return events.Continue
}
```

In `internal/hooks/hooks.go`, register the listener:

```go
	events.RegisterListener(events.RoomChange{}, PresencePlayerEntry)
```

Commit.

- [ ] **Step 9: Build + run tests**

Run: `go build ./...`
Expected: clean (one or two unused-var warnings may need a quick pass; address inline).

Run: `go test ./...`
Expected: PASS. Some hooks tests may need updates; review and adjust.

- [ ] **Step 10: Final task-11 verification commit**

If any small cleanups remain (formatting, stray imports), make a final commit:

```bash
git commit -am "$(cat <<'EOF'
chore(presence): T11 — final cutover cleanups

Any leftover bits from the call-site migration.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Sunset deletions

**Files:**
- Modify: `internal/users/userrecord.go` (delete `ManualAFK` + `AFKMessage` fields)
- Modify: `internal/mobs/mobs.go` (delete `BoredomCounter` + `PreventIdle` fields)
- Modify: `internal/configs/config.memory.go` (delete `MaxMobBoredom`)
- Modify: `_datafiles/config.yaml` (drop `MaxMobBoredom`)

- [ ] **Step 1: Delete UserRecord fields**

Open `internal/users/userrecord.go`. Delete lines 53-54:

```go
	ManualAFK       bool                  `yaml:"-"` // Manually set AFK status (don't save)
	AFKMessage      string                `yaml:"-"` // Optional AFK message (don't save)
```

Build to find any remaining references: `go build ./... 2>&1 | grep -i 'ManualAFK\|AFKMessage'`. Should be zero — all sites were migrated in T11.

- [ ] **Step 2: Delete Mob fields**

Open `internal/mobs/mobs.go`. Delete the lines:

```go
	BoredomCounter  uint8    `yaml:"-"`                          // how many rounds have passed since this mob has seen a player
	PreventIdle     bool     // (line will vary — find via grep)
```

Build to find any remaining references.

- [ ] **Step 3: Delete `MaxMobBoredom` from config struct**

Open `internal/configs/config.memory.go`. Delete:

```go
	MaxMobBoredom       ConfigInt `yaml:"MaxMobBoredom"`
```

And the corresponding `Validate()` clause:

```go
	if m.MaxMobBoredom < 1 {
		m.MaxMobBoredom = 150
	}
```

- [ ] **Step 4: Drop `MaxMobBoredom` from `_datafiles/config.yaml`**

Open `_datafiles/config.yaml`. Search for `MaxMobBoredom` and delete the line.

- [ ] **Step 5: Build and verify**

Run: `go build ./...`
Expected: clean.

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Boot the server locally and confirm clean startup past data-file loading**

Per CLAUDE.md SOP, build a binary and boot:

```bash
go build -o dogmud-chunk5.exe .
./dogmud-chunk5.exe > /tmp/chunk5-boot.log 2>&1 &
sleep 10
grep -E "Server Ready|panic" /tmp/chunk5-boot.log
```

Expected: "Server Ready" present; no panics.

Kill the server. Delete the binary:

```bash
taskkill //F //IM dogmud-chunk5.exe   # on Windows
rm dogmud-chunk5.exe
```

- [ ] **Step 7: Commit**

```bash
git add internal/users/userrecord.go internal/mobs/mobs.go \
        internal/configs/config.memory.go _datafiles/config.yaml
git commit -m "$(cat <<'EOF'
feat(state): T12 — sunset legacy AFK + boredom fields

Drop UserRecord.ManualAFK, UserRecord.AFKMessage, Mob.BoredomCounter,
Mob.PreventIdle fields and the MaxMobBoredom config knob. All call
sites were migrated to the Presence machine in T11.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Context.md sweep

**Files:**
- Create: `internal/state/presence/context.md` (new)
- Modify: `internal/state/combatphase/context.md` (note Presence veto)
- Modify: `internal/characters/context.md` (note Presence field)
- Modify: `internal/hooks/context.md` (document NewRound_PresenceTick + RoomChange_PresencePlayerEntry)
- Modify: `internal/users/context.md` (note Presence on UserRecord via Character)
- Modify: `internal/mobs/context.md` (note BoredomCounter + PreventIdle sunset)

- [ ] **Step 1: Write `internal/state/presence/context.md`**

Author a package-doc context.md following the pattern of `internal/state/control/context.md` and `internal/state/position/context.md`. Cover:

- Overview (single Machine, two transition tables, union enum, per-actor polymorphism)
- State list with semantics
- Key Components (presence.go, transitions.go, etc.)
- Construction (NewPlayerPresence + NewMobPresence)
- Integration points (CombatPhase veto, scheduler observer, NewRound_PresenceTick hook, RoomChange observer)
- Sunset list

Keep it concise — 80-120 lines typical for this package.

- [ ] **Step 2: Update remaining context.md files**

For each of:
- `internal/state/combatphase/context.md` — add a paragraph about the Presence veto on `Idle→Engaging` (block list: Disconnected, Despawning).
- `internal/characters/context.md` — add `Presence *presence.Machine` to the state-machine field list with a 2-3 sentence summary.
- `internal/hooks/context.md` — document `PresenceTick` and `PresencePlayerEntry` in the listener-chain section. Add ordering note: PresenceTick runs between DoCombat and IdleMobs.
- `internal/users/context.md` — note Presence lives on Character but the connection lifecycle is wired here.
- `internal/mobs/context.md` — note `BoredomCounter` and `PreventIdle` are sunset; Presence machine replaces them.

Use Edit on each file for targeted edits (do NOT rewrite whole files).

- [ ] **Step 3: Build to catch broken doc refs**

Run: `go build ./...`
Expected: clean. (Docs don't affect Go compilation, but this is the cheapest broken-rename check.)

- [ ] **Step 4: Commit**

```bash
git add internal/state/presence/context.md internal/state/combatphase/context.md \
        internal/characters/context.md internal/hooks/context.md \
        internal/users/context.md internal/mobs/context.md
git commit -m "$(cat <<'EOF'
docs(context): T13 — chunk 5 context.md sweep

New internal/state/presence/context.md. Cross-link from combatphase
(Idle→Engaging veto), characters (Presence field), hooks (PresenceTick
+ PresencePlayerEntry listeners), users (connection lifecycle), and mobs
(BoredomCounter + PreventIdle sunset).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 14: AI smoke pass (feature-tester)

**Files:**
- Create: `tools/testing/goals/chunk-5-presence-smoke.yaml`
- Output: `tools/testing/reports/YYYY-MM-DD-local-feature-tester-chunk-5-presence-smoke.md`

- [ ] **Step 1: Write the smoke goals file**

Create `tools/testing/goals/chunk-5-presence-smoke.yaml`:

```yaml
goals:
  - Type `status` and verify the command works normally (baseline —
    Presence machine should be in Active).

  - Wait 30+ rounds without sending any commands. Then send `look`.
    Verify the room reads normally. The character should have
    transitioned Active → Idle → Active across this window
    (transparent to the player).

  - Send `afk testing chunk 5`. Verify response says "You are now
    AFK: testing chunk 5". Have another character (or admin via
    `online`) verify the AFK marker shows up next to your name.

  - Send `look`. Verify "You are no longer AFK." Reading the room
    should also work. The Presence machine should have transitioned
    AFK → Active.

  - Spawn a humanoid grappling-archetype mob via `mob spawn 105`
    (Thornwall Thug). Wait several rounds. Mob should remain
    Active (you're in the room with it).

  - Move to a different room far from the spawned mob (or `goto`
    elsewhere). Wait ~30 rounds. The mob should eventually go
    Dormant (PresenceMobDormantAfterRounds = 30).

  - Return to the room with the mob. Verify the mob is visible
    and present. Attack it. Verify the attack lands (Dormant → Active
    wake fires before damage applies).

  - Spawn a shopkeeper mob (any mob with HasShop()) or a forager
    mob (any mob in the "forager" group). Wait 60+ rounds in a
    different room. Return. Verify the essential mob is STILL
    present (essential-mob veto blocked Active → Dormant).

  - Try `afk` again (no message). Verify it toggles. Try `afk`
    twice in a row (no message, then a different message).

  - Read `help afk`. Verify wording is not stale.

  - Report no panics, no missing-template debug strings, no
    "unexpected presence state" log spam, no double-message bugs.
```

- [ ] **Step 2: Boot server locally and run feature-tester smoke**

Run:

```bash
go build -o dogmud-chunk5.exe .
./dogmud-chunk5.exe > /tmp/chunk5-smoke.log 2>&1 &
sleep 10
grep -E "Server Ready|panic" /tmp/chunk5-smoke.log
```

Then invoke `/test-mud local feature-tester chunk-5-presence-smoke.yaml`.

Hard caps: 60 commands, 15 minutes.

- [ ] **Step 3: Read the report and classify findings**

Apply spec §13 success criteria. Critical regressions (chunk 5 introduced) → fix as a hotfix task in this chunk. Tuning-wants / polish / pre-existing → log as followup memories in the same shape as chunk 4f's react step.

- [ ] **Step 4: Kill the server + bridge**

```bash
echo "quit" > tools/mud_cmd.txt
sleep 2
taskkill //F //IM dogmud-chunk5.exe
rm dogmud-chunk5.exe /tmp/chunk5-smoke.log
```

- [ ] **Step 5: (If critical regressions surface) hotfix + retest**

Follow the same TDD pattern as the main tasks: write a failing test for the regression, fix the underlying code, verify green, re-run the smoke.

- [ ] **Step 6: Commit any followup memories**

If memories are created, they live OUTSIDE the repo at `~/.claude/projects/.../memory/` — no git add needed. Just update MEMORY.md's Loose Followups table per the standard pattern.

---

### Task 15: Roadmap + patch notes close-out

**Files:**
- Modify: `COMBAT_STATE_ROADMAP.md` (mark chunk 5 as Done)
- Modify: `PATCH_NOTES.md` (dated entry)

- [ ] **Step 1: Update `COMBAT_STATE_ROADMAP.md`**

Find the chunk 5 row:

```markdown
| 5 | Presence | Not started | Player and mob variants |
```

Replace with:

```markdown
| 5 | Presence | Done (2026-05-19) | Single union-enum Presence machine on every Character. Player states: Connecting / Active / Idle / AFK / Disconnected. Mob states: Spawning / Active / Dormant / Despawning. Active is shared between actors. Per-actor transition tables. Combat Phase veto on Idle→Engaging blocks ONLY Disconnected + Despawning (AFK / Idle / Dormant targets stay attackable; Dormant auto-wakes via the attack-resolution path). Essential-mob veto (shopkeepers, foragers, caravan crew, charmed companions) prevents Active→Dormant/Despawning transitions — those mobs stay Active permanently. Scheduled-transition cleanup observer wipes pending timers on Disconnected/Despawning entry. Sunsets: ManualAFK + AFKMessage (UserRecord); BoredomCounter + PreventIdle (Mob); MaxMobBoredom (config). UI compat shim preserves OnlineInfo.IsAFK. AI feature-tester smoke verified end-to-end. |
```

(Adjust the text if the smoke surfaced critical regressions or notable findings.)

- [ ] **Step 2: Add `PATCH_NOTES.md` entry**

Open `PATCH_NOTES.md`. Add a new section above the most recent dated entry:

```markdown
## 2026-05-19 — Chunk 5: Presence state machine

**Cleaner AFK and idle handling.** The engine now tracks every
character's "presence" — whether they're actively in the world, idle,
AFK, or disconnected — through a dedicated state machine instead of
scattered checks. Functionally identical for most cases. The one
visible change: an AFK player in a dangerous room can STILL be
attacked (intentional — going AFK in a dangerous place was always a
risk).

**Mob hibernation.** Mobs that have been bored for a while (no players
nearby for a stretch) now go Dormant — they skip their per-round tick
to save engine work. The moment a player enters their room or attacks
them, they wake up to normal Active behavior. Shopkeepers, foragers,
caravan crew, and charmed companions never go Dormant — they're
exempt from idle-out so the living-economy systems keep running
smoothly.

**Quieter sunset.** Legacy ManualAFK, AFKMessage, BoredomCounter, and
PreventIdle fields are gone, along with the MaxMobBoredom config knob.

Chunk 5 closes another step of the combat-state-machines arc; chunk 6
(Perception) is what remains.
```

- [ ] **Step 3: Final build + tests**

Run:

```bash
go build ./...
go test ./...
```

Expected: PASS on both.

- [ ] **Step 4: Final boot check**

Per CLAUDE.md Pre-Push SOP:

```bash
go build -o dogmud-chunk5-final.exe .
./dogmud-chunk5-final.exe > /tmp/chunk5-final-boot.log 2>&1 &
sleep 10
grep -E "Server Ready|panic|LoadDataFiles.*loadedCount" /tmp/chunk5-final-boot.log
taskkill //F //IM dogmud-chunk5-final.exe
rm dogmud-chunk5-final.exe /tmp/chunk5-final-boot.log
```

Expected: "Server Ready" present, all `LoadDataFiles` lines show normal counts, no panics.

- [ ] **Step 5: Commit**

```bash
git add COMBAT_STATE_ROADMAP.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(roadmap): T15 — chunk 5 done, Presence machine closed

Update COMBAT_STATE_ROADMAP to mark chunk 5 as Done with the full
deliverables summary. PATCH_NOTES dated entry describing the Presence
machine + mob hibernation. Combat-state-machines arc is now down to
chunk 6 (Perception) only.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Verification Checklist

Before declaring chunk 5 complete:

- [ ] `go build ./...` clean.
- [ ] `go test ./...` clean.
- [ ] `internal/state/presence/presence.go` exists with the union enum + `Machine` wrapper.
- [ ] `internal/state/presence/transitions.go` has both player + mob tables.
- [ ] `internal/state/presence/presence_test.go` covers Behavior Matrix rows PR-001 through PR-028.
- [ ] `Character.Presence` field initialized for both player (constructor) and mob (`Validate()`) paths.
- [ ] `NewRound_PresenceTick` registered between `DoCombat` and `IdleMobs`.
- [ ] Essential-mob veto registered on Active→Dormant + Active→Despawning + Dormant→Despawning.
- [ ] CombatPhase veto blocks Engaging on target Disconnected/Despawning; AFK / Idle / Dormant targets still attackable.
- [ ] Dormant→Active wake fires in the attack-resolution path.
- [ ] Scheduler observer wipes pending timers on Disconnected/Despawning entry.
- [ ] Login → Connecting → Active wired; TCP close → Disconnected wired; SetLastInputRound wakes Idle/AFK.
- [ ] `afk` command rewritten to use Presence; ManualAFK clear-on-next-cmd shim deleted.
- [ ] All call sites cut over (NewRound_IdleMobs, lookfortrouble, rooms.go, MobIdle_HandleIdleMobs, roomdetails.go, userrecord.go).
- [ ] Sunset deletions: ManualAFK, AFKMessage, BoredomCounter, PreventIdle, MaxMobBoredom.
- [ ] Six context.md files updated.
- [ ] AI feature-tester smoke report exists in `tools/testing/reports/`.
- [ ] Critical-regression fixes (if any) committed.
- [ ] Followup memories logged.
- [ ] `COMBAT_STATE_ROADMAP.md` chunk 5 row marked Done.
- [ ] `PATCH_NOTES.md` dated entry added.
- [ ] Server boots cleanly past data-file loading.

---

## Estimated effort

- T1 (package + tests): 60-90 min (most tests; substantial setup)
- T2 (Character wiring): 20-30 min
- T3 (config knobs): 15-20 min
- T4 (presence-tick hook): 30-45 min
- T5 (essential veto): 20-30 min
- T6 (CombatPhase veto): 30-45 min (depends on how clean the existing veto-chain wiring is)
- T7 (auto-wake): 30 min plus the LCA hunt
- T8 (scheduler observer): 20-30 min
- T9 (connection lifecycle): 30-45 min
- T10 (afk command): 20-30 min
- T11 (call-site cutover): 60-90 min (largest non-package task)
- T12 (sunset deletions): 30 min
- T13 (context.md sweep): 30 min
- T14 (AI smoke): 30-45 min wall (most is waiting on the AI run)
- T15 (roadmap + patch notes): 15-20 min

Total: ~7-10 hours, dominated by T1 (package + tests) and T11 (cutover).

---

## Out of Scope (Reminders)

From spec §11:

- `Mob.WanderCount` / `MaxWander` migration (orthogonal wander-budget; stays).
- Helpfile rewrites for `afk` (single-line touchup OK if it's wrong; no broader sweep).
- Idle-mob behavior templates (wandering messages, etc.).
- `mob_spawned` BTree cascade re-routing.
- Per-player AFK thresholds (single set of knobs in v1).
- Charmed-companion auto-Dormant when owner Disconnects.
- Idle-aware faction / quest hooks.

If a smoke finding falls into one of the above categories, it becomes a followup memory in Task 14, NOT a chunk 5 fix task.
