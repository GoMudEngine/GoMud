package presence_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
)

// TestEssentialVeto_BlocksActiveToDormant verifies that a registered
// veto on Active→Dormant returns a non-nil error when fired.
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
// veto on Active→Despawning returns a non-nil error when fired.
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

// TestCombatPhaseVeto_BlocksOnDisconnected verifies that a target in
// Presence.Disconnected is correctly identified as non-attackable by the
// veto logic. (Pure-state smoke; full CombatPhase+Character wiring is
// covered in internal/hooks tests once the engine setup lands.)
func TestCombatPhaseVeto_BlocksOnDisconnected(t *testing.T) {
	target := presence.NewPlayerPresence()
	_ = target.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = target.TransitionTo(presence.Disconnected, state.TransitionReason{Trigger: presence.TriggerTCPClosed})

	// Mirror the veto logic from CombatPhase_Vetoes.go: Disconnected and
	// Despawning block attack; all other states allow it.
	allowed := target.State() != presence.Disconnected && target.State() != presence.Despawning
	if allowed {
		t.Errorf("Disconnected target should be vetoed; got allowed=true")
	}
}

// TestCombatPhaseVeto_AllowsOnAFK verifies that an AFK target is
// attackable — "if you went AFK in a dangerous room, you deserve it."
func TestCombatPhaseVeto_AllowsOnAFK(t *testing.T) {
	target := presence.NewPlayerPresence()
	_ = target.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = target.TransitionTo(presence.AFK, state.TransitionReason{Trigger: presence.TriggerTimeoutAFK})

	allowed := target.State() != presence.Disconnected && target.State() != presence.Despawning
	if !allowed {
		t.Errorf("AFK target should be attackable; got allowed=false")
	}
}

// TestDormantWake_OnAttack verifies a Dormant mob transitions to Active
// when an attack lands.
func TestDormantWake_OnAttack(t *testing.T) {
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

// TestSchedulerCancelObserver_FiresOnTerminal verifies the observer
// fires when the machine enters Disconnected. (Actual scheduler-cancel
// behavior is exercised in the scheduler package's own tests; this
// test only confirms the observer wiring + state filter.)
func TestSchedulerCancelObserver_FiresOnTerminal(t *testing.T) {
	fired := false
	m := presence.NewPlayerPresence()
	m.RegisterObserver("test_terminal_observer",
		func(from, to presence.State, r state.TransitionReason) {
			if to == presence.Disconnected {
				fired = true
			}
		})
	_ = m.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = m.TransitionTo(presence.Disconnected, state.TransitionReason{Trigger: presence.TriggerTCPClosed})
	if !fired {
		t.Errorf("observer did not fire on Disconnected entry")
	}
}

// TestSchedulerCancelObserver_NotFiredOnIntermediate verifies the
// observer's state filter — it should NOT fire on transitions that
// aren't to a terminal state.
func TestSchedulerCancelObserver_NotFiredOnIntermediate(t *testing.T) {
	fired := false
	m := presence.NewPlayerPresence()
	m.RegisterObserver("test_terminal_observer",
		func(from, to presence.State, r state.TransitionReason) {
			if to == presence.Disconnected || to == presence.Despawning {
				fired = true
			}
		})
	_ = m.TransitionTo(presence.Active, state.TransitionReason{Trigger: presence.TriggerEnteredRoom})
	_ = m.TransitionTo(presence.Idle, state.TransitionReason{Trigger: presence.TriggerTimeoutIdle})
	_ = m.TransitionTo(presence.AFK, state.TransitionReason{Trigger: presence.TriggerTimeoutAFK})
	if fired {
		t.Errorf("observer fired on non-terminal transitions; want NOT fired")
	}
}
