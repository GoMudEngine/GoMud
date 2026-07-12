package mutations

import "testing"

func TestGetReflectDamage(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"barbed": {MutationId: "barbed", Name: "Barbed", Rarity: 5,
			Pros: []MutationEffect{{Type: "reflect_damage", Value: 20}}},
		"plain": {MutationId: "plain", Name: "Plain", Rarity: 3},
	})
	defer cleanup()

	if got := GetReflectDamage(map[string]int{"plain": 1}); got != 0 {
		t.Fatalf("no reflect mutation -> 0, got %v", got)
	}
	if got := GetReflectDamage(map[string]int{"barbed": 1}); got != 20 {
		t.Fatalf("barbed -> 20, got %v", got)
	}
}
