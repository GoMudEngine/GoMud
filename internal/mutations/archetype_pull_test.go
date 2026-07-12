package mutations

import "testing"

func TestAllSpecsReturnsSeeded(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Rarity: 3},
		"b": {MutationId: "b", Rarity: 7},
	})
	defer cleanup()

	specs := AllSpecs()
	if len(specs) != 2 {
		t.Fatalf("AllSpecs len = %d, want 2", len(specs))
	}
}
