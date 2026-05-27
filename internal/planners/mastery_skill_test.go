package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestMasterySkill_Registered(t *testing.T) {
	if LookupPlanner("mastery-skill") == nil {
		t.Fatalf("mastery-skill not registered")
	}
}

func TestMasterySkill_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-skill")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "mastery-skill"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestMasterySkill_UnknownSkill_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-skill")
	g := &goals.Goal{Type: "mastery-skill", Params: map[string]any{
		"skill_name": "made-up-skill", "target_rank": 30,
	}}
	res := fn(&mobs.Mob{}, g)
	// Unknown skill → TrainingUnknown → Failure.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

// TestMasterySkill_ForagingSkill_ForageCommand verifies the search skill
// (the canonical foraging/tracking skill in this codebase) dispatches
// the "forage" command.
func TestMasterySkill_ForagingSkill_ForageCommand(t *testing.T) {
	fn := LookupPlanner("mastery-skill")
	g := &goals.Goal{Type: "mastery-skill", Params: map[string]any{
		"skill_name": "search", "target_rank": 30,
	}}
	res := fn(&mobs.Mob{}, g)
	if res.Command != "forage" {
		t.Errorf("command=%q, want forage", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want Running", res.Status)
	}
}
