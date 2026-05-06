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
