package control

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
)

func TestStateString(t *testing.T) {
	cases := []struct {
		state State
		want  string
	}{
		{Controlling, "Controlling"},
		{LosingControl, "LosingControl"},
		{Neutral, "Neutral"},
		{BecomingControlled, "BecomingControlled"},
		{Controlled, "Controlled"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := c.state.String(); got != c.want {
				t.Errorf("State(%d).String() = %q, want %q", c.state, got, c.want)
			}
		})
	}
}

func TestNewMachineDefaultsToNeutral(t *testing.T) {
	m := NewMachine()
	if m.State() != Neutral {
		t.Errorf("NewMachine() state = %v, want Neutral", m.State())
	}
}

func TestTransitionToControllingFromNeutral(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Fatalf("TransitionToControlling: %v", err)
	}
	if m.State() != Controlling {
		t.Errorf("after TransitionToControlling, state = %v, want Controlling", m.State())
	}
}

func TestTransitionToControlledFromNeutral(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss}); err != nil {
		t.Fatalf("TransitionToControlled: %v", err)
	}
	if m.State() != Controlled {
		t.Errorf("after TransitionToControlled, state = %v, want Controlled", m.State())
	}
}

func TestTransitionToNeutralFromControlling(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	if err := m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftLoss}); err != nil {
		t.Fatalf("TransitionToNeutral: %v", err)
	}
	if m.State() != Neutral {
		t.Errorf("after TransitionToNeutral, state = %v, want Neutral", m.State())
	}
}

func TestTransitionToNeutralFromControlled(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss})
	if err := m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Fatalf("TransitionToNeutral: %v", err)
	}
	if m.State() != Neutral {
		t.Errorf("after TransitionToNeutral, state = %v, want Neutral", m.State())
	}
}

func TestTransitionCrossingBothBoundariesControllingToControlled(t *testing.T) {
	// Direct jump from Controlling to Controlled traverses both
	// boundaries. End state must be Controlled.
	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	if err := m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss}); err != nil {
		t.Fatalf("TransitionToControlled: %v", err)
	}
	if m.State() != Controlled {
		t.Errorf("after Controlling→Controlled, state = %v, want Controlled", m.State())
	}
}

func TestTransitionCrossingBothBoundariesControlledToControlling(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss})
	if err := m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Fatalf("TransitionToControlling: %v", err)
	}
	if m.State() != Controlling {
		t.Errorf("after Controlled→Controlling, state = %v, want Controlling", m.State())
	}
}

func TestTransitionIdempotency(t *testing.T) {
	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	// Transitioning to current state should be a no-op
	if err := m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Errorf("idempotent transition: %v", err)
	}
	if m.State() != Controlling {
		t.Errorf("idempotent transition changed state to %v", m.State())
	}
}

func TestTransitionToNeutralWhenAlreadyNeutralIsNoOp(t *testing.T) {
	m := NewMachine()
	if err := m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftWin}); err != nil {
		t.Errorf("Neutral→Neutral: %v", err)
	}
	if m.State() != Neutral {
		t.Errorf("state changed to %v", m.State())
	}
}

func TestBoundaryCallbackFiresOnceForSingleBoundary(t *testing.T) {
	var fires []struct {
		transient State
		from      State
		to        State
	}
	RegisterBoundaryCrossCallback(func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
		fires = append(fires, struct {
			transient State
			from      State
			to        State
		}{transient, from, to})
	})
	defer RegisterBoundaryCrossCallback(nil)

	m := NewMachine()
	// Neutral → Controlling crosses upper boundary once
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})

	if len(fires) != 1 {
		t.Fatalf("expected 1 callback fire, got %d: %+v", len(fires), fires)
	}
	if fires[0].transient != LosingControl {
		t.Errorf("transient = %v, want LosingControl", fires[0].transient)
	}
	if fires[0].from != Neutral || fires[0].to != Controlling {
		t.Errorf("from→to = %v→%v, want Neutral→Controlling", fires[0].from, fires[0].to)
	}
}

func TestBoundaryCallbackFiresTwiceForFullSpan(t *testing.T) {
	var fires []State
	RegisterBoundaryCrossCallback(func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
		fires = append(fires, transient)
	})
	defer RegisterBoundaryCrossCallback(nil)

	m := NewMachine()
	_ = m.TransitionToControlling(state.TransitionReason{Trigger: TriggerDriftWin})
	fires = nil
	// Controlling → Controlled crosses upper AND lower boundaries
	_ = m.TransitionToControlled(state.TransitionReason{Trigger: TriggerDriftLoss})

	if len(fires) != 2 {
		t.Fatalf("expected 2 callback fires, got %d: %v", len(fires), fires)
	}
	if fires[0] != LosingControl {
		t.Errorf("first fire = %v, want LosingControl (upper boundary first)", fires[0])
	}
	if fires[1] != BecomingControlled {
		t.Errorf("second fire = %v, want BecomingControlled (lower boundary second)", fires[1])
	}
}

func TestBoundaryCallbackNotFiredOnNoOpTransition(t *testing.T) {
	var fireCount int
	RegisterBoundaryCrossCallback(func(self state.ActorRef, transient State, from State, to State, r state.TransitionReason) {
		fireCount++
	})
	defer RegisterBoundaryCrossCallback(nil)

	m := NewMachine()
	// Neutral → Neutral (no-op, no boundary crossing)
	_ = m.TransitionToNeutral(state.TransitionReason{Trigger: TriggerDriftWin})

	if fireCount != 0 {
		t.Errorf("expected 0 callback fires for no-op, got %d", fireCount)
	}
}
