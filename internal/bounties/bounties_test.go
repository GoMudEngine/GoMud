package bounties

import (
	"testing"
)

func TestComputeDefaultGold(t *testing.T) {
	defer func() { goldMultiplierForTest = nil; goldFloorForTest = nil }()
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }

	cases := []struct {
		statpool int
		want     int
	}{
		{600, 300},  // 600 * 0.5 = 300
		{1000, 500}, // 1000 * 0.5 = 500
		{50, 50},    // floor wins (25 -> 50)
		{0, 50},     // floor wins
		{200, 100},  // 200 * 0.5 = 100
	}
	for _, c := range cases {
		if got := computeDefaultGold(c.statpool); got != c.want {
			t.Errorf("statpool=%d: got %d, want %d", c.statpool, got, c.want)
		}
	}
}

func TestComputeDefaultRep(t *testing.T) {
	cases := []struct {
		statpool int
		want     int
	}{
		{600, 6},    // 600 / 100 = 6
		{100, 1},    // 100 / 100 = 1
		{50, 1},     // floor of 1 (50 / 100 = 0 -> 1)
		{0, 1},      // floor of 1
		{1000, 10},  // 1000 / 100 = 10
	}
	for _, c := range cases {
		if got := computeDefaultRep(c.statpool); got != c.want {
			t.Errorf("statpool=%d: got %d, want %d", c.statpool, got, c.want)
		}
	}
}
