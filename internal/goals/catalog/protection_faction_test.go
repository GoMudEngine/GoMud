package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionFaction_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("protection-faction"); !ok {
		t.Fatalf("protection-faction not registered")
	}
}

func TestProtectionFaction_DedupKey_ByFactionId(t *testing.T) {
	meta, _ := goals.LookupGoalType("protection-faction")
	g1 := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "watch"}}
	g2 := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "watch"}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same faction_id")
	}
}

func TestProtectionFaction_Predicate_NeverSatisfied(t *testing.T) {
	meta, _ := goals.LookupGoalType("protection-faction")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "watch"}}
	if meta.Predicate(g, mob) {
		t.Errorf("protection-faction predicate should never satisfy")
	}
}
