package forager

import "testing"

func TestStateNameRoundTrip(t *testing.T) {
	// Iterate all defined states including the new StateStoring.
	allStates := []ForagerState{
		StateResting,
		StateTravelingToTerritory,
		StateForaging,
		StateTravelingToDropoff,
		StateDelivering,
		StateStoring,
		StateRecalling,
	}
	for _, s := range allStates {
		name := s.Name()
		got, ok := ParseState(name)
		if !ok || got != s {
			t.Errorf("roundtrip %v -> %q -> %v,%v", s, name, got, ok)
		}
	}
}

func TestAdvanceStateLinearCycle(t *testing.T) {
	cases := []struct {
		from, to ForagerState
	}{
		{StateResting, StateTravelingToTerritory},
		{StateTravelingToTerritory, StateForaging},
		{StateForaging, StateTravelingToDropoff},
		{StateTravelingToDropoff, StateDelivering},
		{StateDelivering, StateStoring},
		{StateStoring, StateRecalling},
		{StateRecalling, StateResting}, // wraps
	}
	for _, c := range cases {
		if got := AdvanceState(c.from); got != c.to {
			t.Errorf("AdvanceState(%v) = %v, want %v", c.from, got, c.to)
		}
	}
}

func TestStateStoring_Name(t *testing.T) {
	if got := StateStoring.Name(); got != "storing" {
		t.Errorf("StateStoring.Name() = %q, want \"storing\"", got)
	}
}

func TestStateStoring_ParseRoundTrip(t *testing.T) {
	got, ok := ParseState("storing")
	if !ok {
		t.Fatal("ParseState(\"storing\") returned ok=false")
	}
	if got != StateStoring {
		t.Errorf("ParseState(\"storing\") = %v, want StateStoring", got)
	}
}

func TestParseStateUnknownReturnsZeroFalse(t *testing.T) {
	s, ok := ParseState("not_a_state")
	if ok {
		t.Errorf("expected ok=false for unknown state, got true")
	}
	if s != StateResting {
		t.Errorf("expected zero-value StateResting, got %v", s)
	}
}

func TestParseStateEmptyStringReturnsZeroFalse(t *testing.T) {
	s, ok := ParseState("")
	if ok || s != StateResting {
		t.Errorf("expected (StateResting, false), got (%v, %v)", s, ok)
	}
}
