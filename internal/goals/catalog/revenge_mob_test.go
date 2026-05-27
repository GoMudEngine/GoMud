package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestRevengeMob_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("revenge-mob"); !ok {
		t.Fatalf("revenge-mob not registered")
	}
}

func TestRevengeMob_DedupKey_DifferentTargets_Distinct(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-mob")
	g1 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "mob", "target_id": 5}}
	g2 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "mob", "target_id": 7}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for different targets")
	}
}

func TestRevengeMob_DedupKey_DifferentKinds_Distinct(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-mob")
	g1 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "mob", "target_id": 5}}
	g2 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "player", "target_id": 5}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for mob:5 vs player:5")
	}
}

func TestRevengeMob_DedupKey_SameTarget_Match(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-mob")
	g1 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "player", "target_id": 3}}
	g2 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "player", "target_id": 3}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same target")
	}
}

// Predicate + ContextScore behavior depend on live mobs/users + combat-memory
// state. Integration coverage lands at Task 23 smoke. Unit-test only the
// registration + DedupKey shape here.
