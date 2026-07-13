package mutations

import "testing"

func TestEffectiveMaxAndDeepen(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"keystone": {MutationId: "keystone", Rarity: 3},         // no MaxRank -> global (4)
		"apex":     {MutationId: "apex", Rarity: 9, MaxRank: 1}, // binary
	})
	defer cleanup()

	// A keystone at level 1 can deepen; an apex at level 1 cannot.
	if !CanDeepen(map[string]int{"keystone": 1}) {
		t.Error("keystone at 1 should be deepenable")
	}
	if CanDeepen(map[string]int{"apex": 1}) {
		t.Error("apex at 1 (MaxRank 1) must NOT be deepenable")
	}
	// RollDeepening never returns an at-cap mutation.
	for i := 0; i < 20; i++ {
		if RollDeepening(map[string]int{"apex": 1}) != "" {
			t.Fatal("RollDeepening returned a capped apex")
		}
	}
	if RollDeepening(map[string]int{"keystone": 1, "apex": 1}) != "keystone" {
		t.Error("RollDeepening should pick the deepenable keystone, not the capped apex")
	}
}
