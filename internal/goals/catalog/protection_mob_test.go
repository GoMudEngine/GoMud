package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionMob_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("protection-mob"); !ok {
		t.Fatalf("protection-mob not registered")
	}
}

func TestProtectionMob_DedupKey_PerTarget(t *testing.T) {
	meta, _ := goals.LookupGoalType("protection-mob")
	g1 := &goals.Goal{Type: "protection-mob", Params: map[string]any{"target_kind": "mob", "target_id": 100}}
	g2 := &goals.Goal{Type: "protection-mob", Params: map[string]any{"target_kind": "mob", "target_id": 200}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for different targets")
	}
}

func TestProtectionMob_Predicate_NeverSatisfied(t *testing.T) {
	// Protection is ongoing — never satisfied. 4.6 will remove the goal
	// via expiry/dead-target logic.
	meta, _ := goals.LookupGoalType("protection-mob")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "protection-mob", Params: map[string]any{"target_kind": "mob", "target_id": 100}}
	if meta.Predicate(g, mob) {
		t.Errorf("protection-mob predicate should never satisfy at 4.3 (got true)")
	}
}
