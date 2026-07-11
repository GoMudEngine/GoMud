package mutations

import "testing"

func TestGetCompanionReserveRank(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"broodmaster": {
			MutationId: "broodmaster", Name: "Broodmaster", Rarity: 5, Pole: "belief",
			Pros: []MutationEffect{{Type: "companion_reserve_reduction"}},
		},
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Pole: "body"},
	})
	defer cleanup()

	if r := GetCompanionReserveRank(map[string]int{}); r != 0 {
		t.Fatalf("no mutations -> rank 0, got %d", r)
	}
	if r := GetCompanionReserveRank(map[string]int{"claws": 4}); r != 0 {
		t.Fatalf("no reducer mutation -> rank 0, got %d", r)
	}
	if r := GetCompanionReserveRank(map[string]int{"broodmaster": 3}); r != 3 {
		t.Fatalf("reducer at rank 3 -> 3, got %d", r)
	}
}
