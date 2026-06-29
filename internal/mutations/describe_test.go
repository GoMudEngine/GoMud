package mutations

import "testing"

func TestDescribeEffect(t *testing.T) {
	cases := []struct {
		name string
		e    MutationEffect
		want string
	}{
		{"stat_flat down", MutationEffect{Type: "stat_flat", Target: "charisma", Value: -10}, "Dulls your Charisma."},
		{"stat_flat up", MutationEffect{Type: "stat_flat", Target: "strength", Value: 8}, "Heightens your Strength."},
		{"stat_multiplier up", MutationEffect{Type: "stat_multiplier", Target: "dexterity", Value: 1.2}, "Heightens your Dexterity."},
		{"stat_multiplier down", MutationEffect{Type: "stat_multiplier", Target: "perception", Value: 0.8}, "Dulls your Perception."},
		{"lightsource", MutationEffect{Type: "flag", Target: "lightsource", Value: 1}, "You shed light -- a beacon in the dark, easy to spot."},
		{"nightvision", MutationEffect{Type: "flag", Target: "nightvision", Value: 1}, "You see clearly in the dark."},
		{"see-hidden", MutationEffect{Type: "flag", Target: "see-hidden", Value: 1}, "You notice hidden creatures and things others miss."},
		{"natural_armor", MutationEffect{Type: "natural_armor", Value: 5}, "Hardens your hide against physical blows."},
		{"aggro up", MutationEffect{Type: "aggro_magnet", Target: "aggro", Value: 1}, "Draws hostile attention toward you."},
		{"unknown type", MutationEffect{Type: "no_such_type"}, ""},
	}
	for _, c := range cases {
		if got := DescribeEffect(c.e); got != c.want {
			t.Errorf("%s: DescribeEffect = %q, want %q", c.name, got, c.want)
		}
	}
}
