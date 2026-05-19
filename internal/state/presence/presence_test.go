package presence

import (
	"errors"
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

// PR-003: Active → Active on input is a no-op (idempotent re-enter).
func TestPlayerPresence_ActiveToActiveOnInput(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
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
	if err := m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerInputReceived}); err == nil {
		t.Errorf("TransitionTo(Active) from Disconnected: got nil err; want ErrInvalidTransition")
	}
}

// PR-012: Observer fires after a successful transition.
func TestPlayerPresence_ObserverFiresOnTransition(t *testing.T) {
	m := NewPlayerPresence()
	var observed []State
	m.RegisterObserver("test_obs", func(from, to State, r state.TransitionReason) {
		observed = append(observed, to)
	})
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	if len(observed) != 1 || observed[0] != Active {
		t.Errorf("observer got %v, want [Active]", observed)
	}
}

// PR-013: Observer does not fire when transition is rejected.
func TestPlayerPresence_ObserverNotFiredOnRejection(t *testing.T) {
	m := NewPlayerPresence()
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerEnteredRoom})
	_ = m.TransitionTo(Disconnected, state.TransitionReason{Trigger: TriggerTCPClosed})
	var observed []State
	m.RegisterObserver("test_obs_rejected", func(from, to State, r state.TransitionReason) {
		observed = append(observed, to)
	})
	// Attempt invalid transition from Disconnected (terminal)
	_ = m.TransitionTo(Active, state.TransitionReason{Trigger: TriggerInputReceived})
	if len(observed) != 0 {
		t.Errorf("observer fired on rejected transition; want no fires, got %v", observed)
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
	m.RegisterVeto(Active, Dormant, func(r state.TransitionReason) error {
		return errors.New("essential mob")
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
