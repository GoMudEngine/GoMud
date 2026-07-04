package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestGrantRandomMutationRare(t *testing.T) {
	defer seedScourTestMutations()() // existing helper in mutation_scour_test.go: 3 commons rarity 2, 3 rares rarity 7
	c := New()
	c.SpeciesId = 1
	c.Mutations = map[string]int{}

	id := c.GrantRandomMutationRare(5)
	if id == "" {
		t.Fatal("expected a mutation granted")
	}
	if spec := mutations.GetMutation(id); spec == nil || spec.Rarity < 5 {
		t.Fatalf("granted %q below rarity floor", id)
	}
	if c.Mutations[id] != 1 {
		t.Fatal("mutation not recorded at level 1")
	}
}
