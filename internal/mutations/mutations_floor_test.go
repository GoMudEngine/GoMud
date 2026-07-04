package mutations

import "testing"

func TestGetWeightedPoolWithFloor(t *testing.T) {
	defer SeedMutationsForTest(map[string]*MutationSpec{
		"common-1": {MutationId: "common-1", Name: "C1", Rarity: 2},
		"rare-1":   {MutationId: "rare-1", Name: "R1", Rarity: 7},
		"rare-2":   {MutationId: "rare-2", Name: "R2", Rarity: 8},
	})()

	pool := GetWeightedPoolWithFloor(map[string]int{}, nil, 5)
	for _, id := range pool {
		if id == "common-1" {
			t.Fatal("rarity floor 5 should exclude rarity-2 mutations")
		}
	}
	if len(pool) == 0 {
		t.Fatal("rare mutations should remain in the floored pool")
	}

	full := GetWeightedPoolWithFloor(map[string]int{}, nil, 0)
	foundCommon := false
	for _, id := range full {
		if id == "common-1" {
			foundCommon = true
		}
	}
	if !foundCommon {
		t.Fatal("floor 0 should include commons")
	}
}
