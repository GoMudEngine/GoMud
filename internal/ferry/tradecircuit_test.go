package ferry

import "testing"

func validCircuit() TradeCircuit {
	return TradeCircuit{
		RouteId:     "test_route",
		FactorMobId: 9577,
		HomePortIdx: 0,
		PortExports: [2][]int{{40057, 40058}, {40018, 40006}},
		PortStops:   [2][]int{{4105, 4106}, {5505, 5508}},
		PortDeliveryBuckets: [2][]string{
			{"thornwall", "base"}, // delivered AT port 0 (= port 1's export buckets)
			{"stillwater"},        // delivered AT port 1
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	}
}

func TestTradeCircuitValidate_Valid(t *testing.T) {
	if err := validCircuit().Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

// TestIsFactorMobId pins the membership check the idle hook uses to exempt
// trade factors from the displaced-home recovery pull (2026-07-03 playtest,
// BUG-1: factors were dying aboard exitless vessel decks because pathto-home
// can never resolve from there).
func TestIsFactorMobId(t *testing.T) {
	cases := []struct {
		name  string
		mobId int
		want  bool
	}{
		{"lakeway factor", 9577, true},
		{"riverway factor", 9578, true},
		{"broadwater factor", 9579, true},
		{"non-factor mob id", 9576, false},
		{"zero", 0, false},
	}
	for _, tc := range cases {
		if got := IsFactorMobId(tc.mobId); got != tc.want {
			t.Errorf("%s: IsFactorMobId(%d) = %v, want %v", tc.name, tc.mobId, got, tc.want)
		}
	}
}

func TestTradeCircuitValidate_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*TradeCircuit)
	}{
		{"missing route", func(c *TradeCircuit) { c.RouteId = "" }},
		{"missing factor", func(c *TradeCircuit) { c.FactorMobId = 0 }},
		{"bad home port", func(c *TradeCircuit) { c.HomePortIdx = 2 }},
		{"empty exports port0", func(c *TradeCircuit) { c.PortExports[0] = nil }},
		{"empty stops port1", func(c *TradeCircuit) { c.PortStops[1] = nil }},
		{"zero loadcap", func(c *TradeCircuit) { c.LoadCap = 0 }},
		{"zero slot max", func(c *TradeCircuit) { c.NewSlotMaxStock = 0 }},
	}
	for _, tc := range cases {
		c := validCircuit()
		tc.mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
		}
	}
}
