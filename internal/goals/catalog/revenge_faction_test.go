package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestRevengeFaction_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("revenge-faction"); !ok {
		t.Fatalf("revenge-faction not registered")
	}
}

func TestRevengeFaction_DedupKey_ByFactionId(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-faction")
	g1 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "bandits", "target_kill_count": 5}}
	g2 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "guards", "target_kill_count": 5}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for different faction_ids")
	}
}

func TestRevengeFaction_DedupKey_SameFactionDifferentCount_Collides(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-faction")
	g1 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "bandits", "target_kill_count": 5}}
	g2 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "bandits", "target_kill_count": 10}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same faction (count is not part of key)")
	}
}
