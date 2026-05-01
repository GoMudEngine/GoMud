package forager

import "testing"

func TestStateNameRoundTrip(t *testing.T) {
	for s := StateResting; s <= StateRecalling; s++ {
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
		{StateDelivering, StateRecalling},
		{StateRecalling, StateResting}, // wraps
	}
	for _, c := range cases {
		if got := AdvanceState(c.from); got != c.to {
			t.Errorf("AdvanceState(%v) = %v, want %v", c.from, got, c.to)
		}
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
