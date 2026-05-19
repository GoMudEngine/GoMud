package perception

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestStateString(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{Sighted, "Sighted"},
		{Blinded, "Blinded"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("State(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}

// PE-001: NewMachine starts in Sighted.
func TestNewMachine_InitialSighted(t *testing.T) {
	m := NewMachine()
	if m.State() != Sighted {
		t.Errorf("NewMachine() state = %v, want Sighted", m.State())
	}
}

// PE-002: Sighted → Blinded via buff applied.
func TestSightedToBlindedOnBuffApplied(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied}); err != nil {
		t.Fatalf("TransitionTo(Blinded): %v", err)
	}
	if m.State() != Blinded {
		t.Errorf("state = %v, want Blinded", m.State())
	}
}

// PE-003: Same Sighted→Blinded path, this time carrying a buffId metadata
// payload identifying flashbang as the source. The FSM itself ignores
// metadata — it's a pass-through payload for observers/consumers to use
// during the messaging-framework wiring. Trigger string is identical to
// PE-002 (both use TriggerBuffApplied).
func TestSightedToBlindedOnFlashbang(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied, Metadata: map[string]any{"buffId": BuffIdFlashbangBlindness}}); err != nil {
		t.Fatalf("TransitionTo(Blinded): %v", err)
	}
	if m.State() != Blinded {
		t.Errorf("state = %v, want Blinded", m.State())
	}
}

// PE-004: Sighted → Blinded via condition added.
func TestSightedToBlindedOnConditionAdded(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerConditionAdded}); err != nil {
		t.Fatalf("TransitionTo(Blinded): %v", err)
	}
	if m.State() != Blinded {
		t.Errorf("state = %v, want Blinded", m.State())
	}
}

// PE-005: Blinded → Sighted via buff expired.
func TestBlindedToSightedOnBuffExpired(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied})
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerBuffExpired}); err != nil {
		t.Fatalf("TransitionTo(Sighted): %v", err)
	}
	if m.State() != Sighted {
		t.Errorf("state = %v, want Sighted", m.State())
	}
}

// PE-006: Blinded → Sighted via flashbang expired.
func TestBlindedToSightedOnFlashbangExpired(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied})
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerBuffExpired, Metadata: map[string]any{"buffId": BuffIdFlashbangBlindness}}); err != nil {
		t.Fatalf("TransitionTo(Sighted): %v", err)
	}
	if m.State() != Sighted {
		t.Errorf("state = %v, want Sighted", m.State())
	}
}

// PE-007: Blinded → Sighted via condition removed.
func TestBlindedToSightedOnConditionRemoved(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerConditionAdded})
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerConditionRemoved}); err != nil {
		t.Fatalf("TransitionTo(Sighted): %v", err)
	}
	if m.State() != Sighted {
		t.Errorf("state = %v, want Sighted", m.State())
	}
}

// PE-008: Pure-FSM check that Blinded→Blinded is not in the table.
// (Caller guards against this via HasAnyBlindSource; this test just
// confirms the table rejects re-entry.)
func TestBlindedSelfTransitionRejected(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied})
	if err := m.TransitionTo(Blinded, state.TransitionReason{Trigger: TriggerBuffApplied}); err == nil {
		t.Errorf("TransitionTo(Blinded) from Blinded: got nil err; want ErrInvalidTransition")
	}
	// State unchanged.
	if m.State() != Blinded {
		t.Errorf("state after rejected self-transition = %v, want Blinded", m.State())
	}
}

// PE-009: Sighted→Sighted is also not in the table (symmetric check).
func TestSightedSelfTransitionRejected(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionTo(Sighted, state.TransitionReason{Trigger: TriggerBuffExpired}); err == nil {
		t.Errorf("TransitionTo(Sighted) from Sighted: got nil err; want ErrInvalidTransition")
	}
	if m.State() != Sighted {
		t.Errorf("state after rejected self-transition = %v, want Sighted", m.State())
	}
}
