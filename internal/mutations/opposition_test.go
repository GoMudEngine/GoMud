package mutations

import "testing"

func TestPoleDepth(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Pole: "body"},
		"ghost": {MutationId: "ghost", Name: "Ghost", Rarity: 5, Pole: "belief"},
	})
	defer cleanup()
	owned := map[string]int{"claws": 2, "ghost": 1}
	if d := PoleDepth(owned, "body"); d != 6 { // 3*2
		t.Fatalf("body depth = %v, want 6", d)
	}
	if d := PoleDepth(owned, "belief"); d != 5 {
		t.Fatalf("belief depth = %v, want 5", d)
	}
}

func TestBodyConvictionScale_MonotonicallyShrinks(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 10, Pole: "body"},
	})
	defer cleanup()
	none := BodyConvictionScale(map[string]int{})
	shallow := BodyConvictionScale(map[string]int{"claws": 1})
	deep := BodyConvictionScale(map[string]int{"claws": 4})
	if none != 1.0 {
		t.Fatalf("no body mutations -> scale 1.0, got %v", none)
	}
	if !(shallow < 1.0 && deep < shallow) {
		t.Fatalf("scale must shrink with depth: none=%v shallow=%v deep=%v", none, shallow, deep)
	}
	if deep < 0 {
		t.Fatalf("scale must stay >= 0, got %v", deep)
	}
}

func TestGearEffectivenessMultiplier_FoldsBeliefDecay(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"ghost": {MutationId: "ghost", Name: "Ghost", Rarity: 10, Pole: "belief"},
	})
	defer cleanup()
	none := GearEffectivenessMultiplier(map[string]int{})
	deep := GearEffectivenessMultiplier(map[string]int{"ghost": 4})
	if none != 1.0 {
		t.Fatalf("no belief mutations -> full gear effectiveness, got %v", none)
	}
	if !(deep < none) {
		t.Fatalf("deep Belief should reduce gear effectiveness: none=%v deep=%v", none, deep)
	}
}
