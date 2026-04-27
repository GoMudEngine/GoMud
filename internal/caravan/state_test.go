package caravan

import "testing"

func TestCaravanState_NameRoundTrip(t *testing.T) {
	// Every state must have a unique name; ParseState(NameOf(s)) == s.
	for s := StateThornwallDwell; s <= StateThornwallRoute; s++ {
		name := s.Name()
		if name == "" {
			t.Errorf("state %d has empty name", s)
		}
		parsed, ok := ParseState(name)
		if !ok {
			t.Errorf("ParseState(%q) returned ok=false", name)
		}
		if parsed != s {
			t.Errorf("ParseState(%q) = %v, want %v", name, parsed, s)
		}
	}
}

func TestCaravanState_ParseUnknownReturnsFalse(t *testing.T) {
	if _, ok := ParseState("not_a_state"); ok {
		t.Error("ParseState(\"not_a_state\") should return ok=false")
	}
	if _, ok := ParseState(""); ok {
		t.Error("ParseState(\"\") should return ok=false")
	}
}

func TestCaravanState_AdvanceCycles(t *testing.T) {
	// Six states form a cycle that loops back to the first.
	expected := []CaravanState{
		StateOutboundTransit,
		StateStillwaterRoute,
		StateStillwaterDwell,
		StateInboundTransit,
		StateThornwallRoute,
		StateThornwallDwell, // wraps
	}
	cur := StateThornwallDwell
	for i, want := range expected {
		cur = AdvanceState(cur)
		if cur != want {
			t.Errorf("step %d: AdvanceState gave %v, want %v", i, cur, want)
		}
	}
}

func TestCaravanState_Classifiers(t *testing.T) {
	cases := []struct {
		s         CaravanState
		isDwell   bool
		isTransit bool
		isRoute   bool
	}{
		{StateThornwallDwell, true, false, false},
		{StateOutboundTransit, false, true, false},
		{StateStillwaterRoute, false, false, true},
		{StateStillwaterDwell, true, false, false},
		{StateInboundTransit, false, true, false},
		{StateThornwallRoute, false, false, true},
	}
	for _, c := range cases {
		if got := IsDwellState(c.s); got != c.isDwell {
			t.Errorf("IsDwellState(%v) = %v, want %v", c.s, got, c.isDwell)
		}
		if got := IsTransitState(c.s); got != c.isTransit {
			t.Errorf("IsTransitState(%v) = %v, want %v", c.s, got, c.isTransit)
		}
		if got := IsRouteState(c.s); got != c.isRoute {
			t.Errorf("IsRouteState(%v) = %v, want %v", c.s, got, c.isRoute)
		}
	}
}

func TestCaravanState_RouteFor(t *testing.T) {
	if RouteForState(StateOutboundTransit) != &OutboundRoute {
		t.Error("OutboundTransit should map to OutboundRoute")
	}
	if RouteForState(StateStillwaterRoute) != &OutboundRoute {
		t.Error("StillwaterRoute should map to OutboundRoute (continues outbound leg)")
	}
	if RouteForState(StateInboundTransit) != &InboundRoute {
		t.Error("InboundTransit should map to InboundRoute")
	}
	if RouteForState(StateThornwallRoute) != &InboundRoute {
		t.Error("ThornwallRoute should map to InboundRoute")
	}
	if RouteForState(StateThornwallDwell) != nil {
		t.Error("ThornwallDwell should map to no route (nil)")
	}
}
