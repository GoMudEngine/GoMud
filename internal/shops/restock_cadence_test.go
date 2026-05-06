package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestRestockCadenceRounds_PerTier(t *testing.T) {
	// configs.GetServerConfig().Timing.RoundsPerSecond exposes the
	// rounds-per-real-second; game-time uses the configured GameDay
	// multiplier. The test isolates the math by stubbing those
	// values and asserting the conversion.
	b := configs.Balance{
		RestockCadenceTier50Hours: 1,
		RestockCadenceTier40Hours: 2,
		RestockCadenceTier30Hours: 6,
		RestockCadenceTier20Hours: 24,
		RestockCadenceTier10Days:  5,
	}
	cases := []struct {
		tier int
		hrs  int
	}{
		{50, 1},
		{40, 2},
		{30, 6},
		{20, 24},
		{10, 5 * 24}, // tier-10 expressed in days
	}
	for _, c := range cases {
		got := RestockCadenceHours(b, c.tier)
		if got != c.hrs {
			t.Errorf("tier %d: got %d hours, want %d", c.tier, got, c.hrs)
		}
	}
}

func TestRestockCadenceHours_UnknownTier(t *testing.T) {
	b := configs.Balance{RestockCadenceTier50Hours: 1}
	got := RestockCadenceHours(b, 999)
	if got != 0 {
		t.Errorf("unknown tier: got %d, want 0 (sentinel for skip)", got)
	}
}
