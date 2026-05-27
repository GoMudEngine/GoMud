package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestBefriend_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("befriend"); !ok {
		t.Fatalf("befriend not registered")
	}
}

func TestBefriend_DedupKey_PerTarget(t *testing.T) {
	meta, _ := goals.LookupGoalType("befriend")
	g1 := &goals.Goal{Type: "befriend", Params: map[string]any{"target_kind": "player", "target_id": 5}}
	g2 := &goals.Goal{Type: "befriend", Params: map[string]any{"target_kind": "player", "target_id": 6}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different targets")
	}
}
