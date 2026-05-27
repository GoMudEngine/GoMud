package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestBefriendFaction_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("befriend-faction"); !ok {
		t.Fatalf("befriend-faction not registered")
	}
}

func TestBefriendFaction_DedupKey_ByFaction(t *testing.T) {
	meta, _ := goals.LookupGoalType("befriend-faction")
	g1 := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "merchants"}}
	g2 := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "watch"}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different factions")
	}
}

func TestBefriendFaction_DedupKey_SameFaction_Collides(t *testing.T) {
	meta, _ := goals.LookupGoalType("befriend-faction")
	g1 := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "merchants", "rep_threshold": 50}}
	g2 := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "merchants", "rep_threshold": 80}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same faction (threshold is not part of key)")
	}
}
