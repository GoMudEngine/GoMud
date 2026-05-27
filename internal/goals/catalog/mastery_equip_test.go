package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestMasteryEquip_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("mastery-equip"); !ok {
		t.Fatalf("mastery-equip not registered")
	}
}

func TestMasteryEquip_DedupKey_BySlot(t *testing.T) {
	meta, _ := goals.LookupGoalType("mastery-equip")
	g1 := &goals.Goal{Type: "mastery-equip", Params: map[string]any{"slot": "weapon", "min_rarity_tier": 60}}
	g2 := &goals.Goal{Type: "mastery-equip", Params: map[string]any{"slot": "head", "min_rarity_tier": 60}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different slots")
	}
}
