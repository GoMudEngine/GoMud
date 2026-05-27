package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestAllCatalogTypes_Registered(t *testing.T) {
	want := []string{
		"survival",
		"wealth-gold", "wealth-item", "craft-item",
		"revenge-mob", "revenge-faction",
		"protection-mob", "protection-faction",
		"befriend", "befriend-faction",
		"mastery-skill", "mastery-equip",
		"visit-zone",
	}
	for _, name := range want {
		if _, ok := goals.LookupGoalType(name); !ok {
			t.Errorf("type %q not registered", name)
		}
	}
}
