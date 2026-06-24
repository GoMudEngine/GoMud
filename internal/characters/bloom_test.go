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
