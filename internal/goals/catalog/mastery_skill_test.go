package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestMasterySkill_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("mastery-skill"); !ok {
		t.Fatalf("mastery-skill not registered")
	}
}

func TestMasterySkill_DedupKey_BySkillName(t *testing.T) {
	meta, _ := goals.LookupGoalType("mastery-skill")
	g1 := &goals.Goal{Type: "mastery-skill", Params: map[string]any{"skill_name": "weapon-combat", "target_rank": 30}}
	g2 := &goals.Goal{Type: "mastery-skill", Params: map[string]any{"skill_name": "spellcasting", "target_rank": 30}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different skills")
	}
}
