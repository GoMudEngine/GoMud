package itemvalue

import "testing"

func TestProfileFor_BehaviorPrimary(t *testing.T) {
	cases := []struct {
		behavior string
		want     string // profile Name
	}{
		{"tank_taunter", "PhysicalTank"},
		{"pure_caster", "MagicalPure"},
		{"support_caster", "MagicalSupport"},
		{"ambusher", "Stealth"},
		{"lookout", "Stealth"},
		{"generic_fighter", "PhysicalBruiser"},
		{"melee_self_buff", "PhysicalBruiser"},
		{"leader", "PhysicalBruiser"},
		{"combat_passive", "Neutral"},
		{"prey", "Neutral"},
		{"noncombat_passive", "Neutral"},
		{"noncombat_questgiver", "Neutral"},
		{"noncombat_shopkeeper", "Neutral"},
	}
	for _, c := range cases {
		// Pass a non-empty stat to verify behavior takes precedence.
		got := ProfileFor("fighting", c.behavior)
		if got.Name != c.want {
			t.Errorf("ProfileFor(\"fighting\", %q).Name = %q, want %q",
				c.behavior, got.Name, c.want)
		}
	}
}

func TestProfileFor_StatFallback(t *testing.T) {
	cases := []struct {
		stat string
		want string
	}{
		{"fighting", "PhysicalBruiser"},
		{"casting", "MagicalSupport"},
		{"tank", "PhysicalTank"},
		{"", "Neutral"},
	}
	for _, c := range cases {
		got := ProfileFor(c.stat, "")
		if got.Name != c.want {
			t.Errorf("ProfileFor(%q, \"\").Name = %q, want %q",
				c.stat, got.Name, c.want)
		}
	}
}

func TestProfileFor_UnknownBehaviorFallsThrough(t *testing.T) {
	// An unknown behavior archetype should NOT match anything
	// and fall through to the stat archetype.
	got := ProfileFor("tank", "spelunker_unknown")
	if got.Name != "PhysicalTank" {
		t.Errorf("ProfileFor(\"tank\", unknown) = %q, want PhysicalTank", got.Name)
	}

	got = ProfileFor("", "spelunker_unknown")
	if got.Name != "Neutral" {
		t.Errorf("ProfileFor(\"\", unknown) = %q, want Neutral", got.Name)
	}
}
