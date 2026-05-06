package opinions

import "testing"

func TestTierOf(t *testing.T) {
	cases := []struct {
		score int
		want  Tier
	}{
		{-100, TierHostile},
		{-50, TierHostile},
		{-49, TierCold},
		{-15, TierCold},
		{-14, TierNeutral},
		{0, TierNeutral},
		{14, TierNeutral},
		{15, TierWarm},
		{49, TierWarm},
		{50, TierFriendly},
		{100, TierFriendly},
	}
	for _, c := range cases {
		if got := TierOf(c.score); got != c.want {
			t.Errorf("TierOf(%d) = %v, want %v", c.score, got, c.want)
		}
	}
}

func TestPullTowardDefault(t *testing.T) {
	cases := []struct {
		name  string
		score int
		def   int
		steps int
		want  int
	}{
		{"no steps", -50, 0, 0, -50},
		{"one step toward 0 from negative", -50, 0, 1, -49},
		{"five steps toward 0 from negative", -50, 0, 5, -45},
		{"steps cannot overshoot zero default", -3, 0, 10, 0},
		{"toward positive default", -10, 5, 3, -7},
		{"steps cannot overshoot positive default", -3, 5, 100, 5},
		{"already at default", 5, 5, 100, 5},
		{"positive score decays toward 0", 50, 0, 5, 45},
		{"positive score decays toward negative default", 50, -10, 100, -10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pull(c.score, c.def, c.steps); got != c.want {
				t.Errorf("pull(%d, %d, %d) = %d, want %d", c.score, c.def, c.steps, got, c.want)
			}
		})
	}
}

func TestDecayedScore(t *testing.T) {
	const halfLife uint64 = 100
	cases := []struct {
		name     string
		score    int
		def      int
		anchor   uint64
		now      uint64
		halfLife uint64
		want     int
	}{
		{"no time elapsed", -50, 0, 1000, 1000, halfLife, -50},
		{"half a half-life elapsed (no integer step)", -50, 0, 1000, 1049, halfLife, -50},
		{"one half-life elapsed", -50, 0, 1000, 1100, halfLife, -49},
		{"ten half-lives elapsed", -50, 0, 1000, 2000, halfLife, -40},
		{"halfLife=0 means no decay", -50, 0, 1000, 99999, 0, -50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := decayedScore(c.score, c.def, c.anchor, c.now, c.halfLife); got != c.want {
				t.Errorf("decayedScore(%d, %d, anchor=%d, now=%d, hl=%d) = %d, want %d",
					c.score, c.def, c.anchor, c.now, c.halfLife, got, c.want)
			}
		})
	}
}

func TestClampScore(t *testing.T) {
	cases := []struct{ in, want int }{
		{-200, -100},
		{-100, -100},
		{-50, -50},
		{0, 0},
		{50, 50},
		{100, 100},
		{200, 100},
	}
	for _, c := range cases {
		if got := clampScore(c.in); got != c.want {
			t.Errorf("clampScore(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
