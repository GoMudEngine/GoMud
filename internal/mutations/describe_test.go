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
		// Multiplier effects use ADDITIVE deltas centered on 0 (the engine
		// applies them as base * (1.0 + value); see mutations.go). A positive
		// value is a bonus, a negative value a penalty -- NOT thresholds at 1.0.
		{"stat_multiplier up", MutationEffect{Type: "stat_multiplier", Target: "dexterity", Value: 0.2}, "Heightens your Dexterity."},
		{"stat_multiplier down", MutationEffect{Type: "stat_multiplier", Target: "perception", Value: -0.2}, "Dulls your Perception."},
		{"health_multiplier up", MutationEffect{Type: "health_multiplier", Value: 0.2}, "Toughens your body, deepening your reserves of health."},
		{"health_multiplier down", MutationEffect{Type: "health_multiplier", Value: -0.1}, "Thins your body's reserves of health."},
		{"stamina_regen up", MutationEffect{Type: "stamina_regen_multiplier", Value: 0.2}, "Quickens how fast your stamina returns."},
		{"stamina_regen down", MutationEffect{Type: "stamina_regen_multiplier", Value: -0.15}, "Slows how fast your stamina returns."},
		{"conviction_cost down", MutationEffect{Type: "conviction_cost_multiplier", Value: -0.2}, "Lessens the conviction your abilities cost."},
		{"conviction_cost up", MutationEffect{Type: "conviction_cost_multiplier", Value: 0.2}, "Raises the conviction your abilities cost."},
		{"lightsource", MutationEffect{Type: "flag", Target: "lightsource", Value: 1}, "You shed light -- a beacon in the dark, easy to spot."},
		{"nightvision", MutationEffect{Type: "flag", Target: "nightvision", Value: 1}, "You see clearly in the dark."},
		{"see-hidden", MutationEffect{Type: "flag", Target: "see-hidden", Value: 1}, "You notice hidden creatures and things others miss."},
		{"natural_armor", MutationEffect{Type: "natural_armor", Value: 5}, "Hardens your hide against physical blows."},
		{"aggro up", MutationEffect{Type: "aggro_magnet", Target: "aggro", Value: 1}, "Draws hostile attention toward you."},
		{"stealth_bonus", MutationEffect{Type: "stealth_bonus", Value: 20}, "Sharpens your ability to move unseen and unheard."},
		{"movement_speed faster", MutationEffect{Type: "movement_speed", Value: -0.15}, "Lightens your step -- you move faster and more quietly."},
		{"unknown type", MutationEffect{Type: "no_such_type"}, ""},
	}
	for _, c := range cases {
		if got := DescribeEffect(c.e); got != c.want {
			t.Errorf("%s: DescribeEffect = %q, want %q", c.name, got, c.want)
		}
	}
}
