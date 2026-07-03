package ferry

import "testing"

func testRoute() Route {
	r := validRoute() // from route_test.go
	return r
}

func TestStateAt_CycleBoundaries(t *testing.T) {
	r := testRoute() // crossing 2h=75r, layover 1h=37r, cycle=224r
	cases := []struct {
		round       uint64
		docked      bool
		portIdx     int
		roundsUntil int
	}{
		{0, true, 0, 37},
		{36, true, 0, 1},
		{37, false, 1, 75},
		{111, false, 1, 1},
		{112, true, 1, 37},
		{148, true, 1, 1},
		{149, false, 0, 75},
		{223, false, 0, 1},
		{224, true, 0, 37},
	}
	for _, tc := range cases {
		s := StateAt(r, tc.round, 900)
		if s.Docked != tc.docked || s.PortIdx != tc.portIdx || s.RoundsUntilTransition != tc.roundsUntil {
			t.Errorf("round %d: got %+v, want docked=%v port=%d until=%d",
				tc.round, s, tc.docked, tc.portIdx, tc.roundsUntil)
		}
	}
}

func TestStateAt_PhaseOffset(t *testing.T) {
	r := testRoute()
	r.PhaseOffsetRounds = 37
	s := StateAt(r, 0, 900)
	if s.Docked || s.PortIdx != 1 {
		t.Fatalf("with offset 37, round 0 should be at sea toward port 1, got %+v", s)
	}
}

func TestNextDockedRound(t *testing.T) {
	r := testRoute()
	if got := NextDockedRound(r, 0, 0, 900); got != 0 {
		t.Errorf("port 0 from round 0: got %d, want 0", got)
	}
	if got := NextDockedRound(r, 1, 0, 900); got != 112 {
		t.Errorf("port 1 from round 0: got %d, want 112", got)
	}
	if got := NextDockedRound(r, 1, 113, 900); got != 113 {
		t.Errorf("port 1 from round 113: got %d, want 113", got)
	}
	if got := NextDockedRound(r, 1, 150, 900); got != 336 {
		t.Errorf("port 1 from round 150: got %d, want 336", got)
	}
}
