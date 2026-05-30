package bountyhunter

import "testing"

func TestScaledStatpool(t *testing.T) {
	// clamp(round(base + gold*perGold), min, max)
	// math.Round uses half-away-from-zero: Round(462.5) = 463
	cases := []struct{ gold, want int }{
		{300, 325},  // round(250 + 75.0)   = 325
		{600, 400},  // round(250 + 150.0)  = 400
		{850, 463},  // round(250 + 212.5)  = 463 (half-away-from-zero)
		{1000, 500}, // round(250 + 250.0)  = 500, at max cap
		{100, 300},  // round(250 + 25.0)   = 275, clamped to min 300
	}
	for _, c := range cases {
		got := scaledStatpool(c.gold, 250, 0.25, 300, 500)
		if got != c.want {
			t.Fatalf("scaledStatpool(%d) = %d, want %d", c.gold, got, c.want)
		}
	}
}

func TestGearGold(t *testing.T) {
	if g := gearGold(500, 5); g != 100 {
		t.Fatalf("gearGold(500,5) = %d, want 100", g)
	}
	if g := gearGold(500, 0); g != 500 {
		t.Fatalf("gearGold with 0 divisor should guard to /1, got %d", g)
	}
}
