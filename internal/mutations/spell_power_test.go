package mutations

import "testing"

func TestGetSpellPowerMultiplier(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"ether-gland": {MutationId: "ether-gland", Name: "Ether Gland", Rarity: 4,
			Pros: []MutationEffect{{Type: "spell_power", Value: 0.5}}},
	})
	defer cleanup()
	// level 1 → 1.0× → 0.5
	if got := GetSpellPowerMultiplier(map[string]int{"ether-gland": 1}); got != 0.5 {
		t.Fatalf("spell_power L1 = %v, want 0.5", got)
	}
	if got := GetSpellPowerMultiplier(map[string]int{}); got != 0 {
		t.Fatalf("no mutation → 0, got %v", got)
	}
}
