package control

import "testing"

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
