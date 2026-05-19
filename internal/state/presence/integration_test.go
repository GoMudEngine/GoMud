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
