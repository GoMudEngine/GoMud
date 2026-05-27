package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestVisitZone_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("visit-zone"); !ok {
		t.Fatalf("visit-zone not registered")
	}
}

func TestVisitZone_DedupKey_ByTargetZone(t *testing.T) {
	meta, _ := goals.LookupGoalType("visit-zone")
	g1 := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	g2 := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "thornwall_city"}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different zones")
	}
}

func TestVisitZone_Predicate_Visited_True(t *testing.T) {
	meta, _ := goals.LookupGoalType("visit-zone")
	mob := &mobs.Mob{}
	mob.VisitedZones = map[string]bool{"stillwater": true}
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	if !meta.Predicate(g, mob) {
		t.Errorf("predicate when visited: got false, want true")
	}
}

func TestVisitZone_Predicate_Unvisited_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("visit-zone")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate when unvisited: got true, want false")
	}
}
