package mutations

import "testing"

func TestIsFlying(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"winged-flight": {MutationId: "winged-flight", Name: "Winged Flight", Rarity: 8,
			Pros: []MutationEffect{{Type: "flag", Target: "flying"}}},
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3},
	})
	defer cleanup()

	if IsFlying(map[string]int{"claws": 2}) {
		t.Fatal("no flight mutation -> not flying")
	}
	if !IsFlying(map[string]int{"winged-flight": 1}) {
		t.Fatal("winged-flight -> flying")
	}
}
