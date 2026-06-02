package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestLookupCorpseSalvage_Animal(t *testing.T) {
	got := LookupCorpseSalvage([]string{"animal", "canine", "predator"})
	want := []items.SalvageReturn{
		{ItemTag: "raw-meat", Quantity: 1},
		{ItemTag: "leather-strip", Quantity: 2},
		{ItemTag: "sinew", Quantity: 1},
	}
	if !equalReturns(got, want) {
		t.Errorf("animal: got %+v, want %+v", got, want)
	}
}

func TestLookupCorpseSalvage_Humanoid(t *testing.T) {
	got := LookupCorpseSalvage([]string{"bandit", "humanoid"})
	want := []items.SalvageReturn{
		{ItemTag: "cloth-strip", Quantity: 2},
		{ItemTag: "leather-strip", Quantity: 1},
	}
	if !equalReturns(got, want) {
		t.Errorf("humanoid: got %+v, want %+v", got, want)
	}
}

func TestLookupCorpseSalvage_NoMatch(t *testing.T) {
	got := LookupCorpseSalvage([]string{"chrysalis", "elemental"})
	if got != nil {
		t.Errorf("no-match: got %+v, want nil", got)
	}
}

func TestLookupCorpseSalvage_EmptyGroups(t *testing.T) {
	got := LookupCorpseSalvage(nil)
	if got != nil {
		t.Errorf("nil groups: got %+v, want nil", got)
	}
	got = LookupCorpseSalvage([]string{})
	if got != nil {
		t.Errorf("empty groups: got %+v, want nil", got)
	}
}

func TestLookupCorpseSalvage_FirstTableEntryWins(t *testing.T) {
	// table order: rodent, animal, humanoid — if a mob has both
	// animal and humanoid groups, the animal entry wins.
	got := LookupCorpseSalvage([]string{"humanoid", "animal"})
	want := []items.SalvageReturn{
		{ItemTag: "raw-meat", Quantity: 1},
		{ItemTag: "leather-strip", Quantity: 2},
		{ItemTag: "sinew", Quantity: 1},
	}
	if !equalReturns(got, want) {
		t.Errorf("multi-group: got %+v, want %+v", got, want)
	}
}

func TestLookupCorpseSalvage_AnimalYieldsRawMeat(t *testing.T) {
	got := LookupCorpseSalvage([]string{"animal", "predator"})
	tags := map[string]int{}
	for _, r := range got {
		tags[r.ItemTag] = r.Quantity
	}
	if tags["raw-meat"] < 1 {
		t.Errorf("animal corpse should yield raw-meat, got %v", got)
	}
}

func TestLookupCorpseSalvage_SmallGameYieldsHareMeat(t *testing.T) {
	got := LookupCorpseSalvage([]string{"animal", "rodent", "prey"})
	tags := map[string]int{}
	for _, r := range got {
		tags[r.ItemTag] = r.Quantity
	}
	if tags["wild-hare-meat"] < 1 {
		t.Errorf("small-game corpse should yield wild-hare-meat, got %v", got)
	}
}

func equalReturns(a, b []items.SalvageReturn) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ItemTag != b[i].ItemTag || a[i].Quantity != b[i].Quantity {
			return false
		}
	}
	return true
}
