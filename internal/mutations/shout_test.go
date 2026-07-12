package mutations

import "testing"

func TestGetShoutAmp(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"booming-lungs": {MutationId: "booming-lungs", Name: "Booming Lungs", Rarity: 5,
			Pros: []MutationEffect{{Type: "shout_amp", Value: 0.30}}},
		"plain": {MutationId: "plain", Name: "Plain", Rarity: 2,
			Pros: []MutationEffect{{Type: "stat_flat", Target: "charisma", Value: 5}}},
	})
	defer cleanup()

	if got := GetShoutAmp(map[string]int{"booming-lungs": 1, "plain": 1}); got != 0.30 {
		t.Fatalf("GetShoutAmp = %v, want 0.30", got)
	}
	if got := GetShoutAmp(map[string]int{"plain": 1}); got != 0 {
		t.Fatalf("GetShoutAmp(no amp) = %v, want 0", got)
	}
}

func TestDescribeEffect_ShoutMechanics(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "shout_amp", Value: 0.30}) == "" {
		t.Fatal("shout_amp must have a non-empty description")
	}
	if DescribeEffect(MutationEffect{Type: "flag", Target: "shout-stacking"}) == "" {
		t.Fatal("shout-stacking flag must have a non-empty description")
	}
}
