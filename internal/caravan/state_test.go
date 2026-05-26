package caravan

import "testing"

func TestCaravanState_Name(t *testing.T) {
	// Every state must have a unique non-empty name.
	seen := map[string]bool{}
	for s := StateThornwallDwell; s < 8; s++ {
		name := s.Name()
		if name == "" {
			t.Errorf("state %d has empty name", s)
		}
		if seen[name] {
			t.Errorf("duplicate state name %q", name)
		}
		seen[name] = true
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

func TestIsFernwayPickupState(t *testing.T) {
	if !IsFernwayPickupState(StateOutboundFernwayPickup) {
		t.Error("OutboundFernwayPickup should be a pickup state")
	}
	if !IsFernwayPickupState(StateInboundFernwayPickup) {
		t.Error("InboundFernwayPickup should be a pickup state")
	}
	if IsFernwayPickupState(StateOutboundTransit) {
		t.Error("OutboundTransit should NOT be a pickup state")
	}
	if IsFernwayPickupState(StateThornwallDwell) {
		t.Error("ThornwallDwell should NOT be a pickup state")
	}
}

func TestIsTransitState_IncludesPickup(t *testing.T) {
	transit := []CaravanState{
		StateOutboundTransit, StateInboundTransit,
		StateOutboundFernwayPickup, StateInboundFernwayPickup,
	}
	for _, s := range transit {
		if !IsTransitState(s) {
			t.Errorf("IsTransitState(%v) = false, want true", s)
		}
	}
}
