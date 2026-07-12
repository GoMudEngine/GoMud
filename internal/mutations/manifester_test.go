package mutations

import "testing"

func TestGetCompanionEmpowerment(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"symbiotic": {MutationId: "symbiotic", Name: "Symbiotic", Rarity: 5,
			Pros: []MutationEffect{{Type: "companion_empowerment", Value: 0.15}}},
		"other": {MutationId: "other", Name: "Other", Rarity: 3},
	})
	defer cleanup()

	if got := GetCompanionEmpowerment(map[string]int{"other": 1}); got != 0 {
		t.Fatalf("no empowerment mutation -> 0, got %v", got)
	}
	if got := GetCompanionEmpowerment(map[string]int{"symbiotic": 1}); got != 0.15 {
		t.Fatalf("symbiotic -> 0.15, got %v", got)
	}
}
