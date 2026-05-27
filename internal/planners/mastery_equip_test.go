package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestMasteryEquip_Registered(t *testing.T) {
	if LookupPlanner("mastery-equip") == nil {
		t.Fatalf("mastery-equip not registered")
	}
}

func TestMasteryEquip_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-equip")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "mastery-equip"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestMasteryEquip_NoShopInZone_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-equip")
	g := &goals.Goal{Type: "mastery-equip", Params: map[string]any{
		"slot": "weapon", "min_rarity_tier": 60,
	}}
	// findShopInZoneSelling returns (0, false) for an empty zone — Failure branch.
	res := fn(&mobs.Mob{}, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no shop in zone)", res.Status)
	}
}
