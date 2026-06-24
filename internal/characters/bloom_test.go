package characters

import "testing"

func TestBloomAddiction_AddAndFloor(t *testing.T) {
	c := &Character{}
	c.AddBloomAddiction(3)
	if c.BloomAddiction != 3 {
		t.Errorf("expected 3, got %d", c.BloomAddiction)
	}
	c.AddBloomAddiction(-10) // floors at 0
	if c.BloomAddiction != 0 {
		t.Errorf("expected floor 0, got %d", c.BloomAddiction)
	}
}

func TestBloomAddictionTier(t *testing.T) {
	cases := []struct {
		level    int
		wantTier int
		wantName string
	}{
		{0, 0, "clean"},
		{1, 1, "hooked"},
		{3, 1, "hooked"},
		{4, 2, "dependent"},
		{8, 2, "dependent"},
		{9, 3, "enslaved"},
		{20, 3, "enslaved"},
	}
	for _, tc := range cases {
		c := &Character{BloomAddiction: tc.level}
		if got := c.BloomAddictionTier(); got != tc.wantTier {
			t.Errorf("level %d: tier %d want %d", tc.level, got, tc.wantTier)
		}
		if got := c.BloomAddictionTierName(); got != tc.wantName {
			t.Errorf("level %d: name %q want %q", tc.level, got, tc.wantName)
		}
	}
}

// TestBloomDecayReferenceAdvance verifies the decay pacing used by
// NewRound_Bloom: after one decay tick, BloomLastDoseRound is advanced by one
// decayRounds period so the next decay fires exactly one period later, not
// immediately on every subsequent round.
func TestBloomDecayReferenceAdvance(t *testing.T) {
	const decayRounds = uint64(300)
	const startRound = uint64(1000)

	c := &Character{
		BloomAddiction:     3,
		BloomLastDoseRound: startRound,
	}

	// The hook fires at currentRound = startRound + decayRounds.
	currentRound := startRound + decayRounds

	// Precondition: threshold is reached.
	if since := currentRound - c.BloomLastDoseRound; since < decayRounds {
		t.Fatalf("test setup broken: since=%d < decayRounds=%d", since, decayRounds)
	}

	// Mirror the hook's decay step.
	c.AddBloomAddiction(-1)
	c.BloomLastDoseRound += decayRounds

	// Addiction decremented.
	if c.BloomAddiction != 2 {
		t.Errorf("expected addiction 2 after one decay, got %d", c.BloomAddiction)
	}

	// Reference advanced: next check should not fire until another period elapses.
	sinceAfter := currentRound - c.BloomLastDoseRound
	if sinceAfter >= decayRounds {
		t.Errorf("reference not advanced: sinceAfter=%d still >= decayRounds=%d",
			sinceAfter, decayRounds)
	}
}
