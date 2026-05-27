package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestSurvival_Registered verifies the survival planner is wired via init().
func TestSurvival_Registered(t *testing.T) {
	if LookupPlanner("survival") == nil {
		t.Fatalf("survival not registered")
	}
}

// TestSurvival_HighHP_NotInCombat_Success verifies that a mob at full HP
// and not in combat satisfies the goal (StatusSuccess).
func TestSurvival_HighHP_NotInCombat_Success(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 100
	mob.Character.HealthMax.Value = 100
	// Aggro is nil (not in combat).
	res := fn(mob, &goals.Goal{Type: "survival"})
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want StatusSuccess (HP 100/100, not in combat)", res.Status)
	}
}

// TestSurvival_InCombat_FleesOrFails verifies that a mob in combat
// attempts to flee. pickRandomExit is a TODO-ADAPT stub returning ""
// (no live room state), so the expected result is StatusFailure until
// the helper is fully wired.
func TestSurvival_InCombat_FleesOrFails(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 30
	mob.Character.HealthMax.Value = 100
	mob.Character.Aggro = &characters.Aggro{}
	res := fn(mob, &goals.Goal{Type: "survival"})
	// pickRandomExit returns "" (no live room available in unit tests) →
	// StatusFailure. When the helper is fully wired, StatusRunning + a
	// "flee <exit>" command is expected; update this test at that time.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (pickRandomExit stubbed, no room)", res.Status)
	}
}

// TestSurvival_LowHP_NotInCombat_RestsByDefault verifies that a mob at
// low HP and not in combat (no potion available) falls back to "rest".
func TestSurvival_LowHP_NotInCombat_RestsByDefault(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 10
	mob.Character.HealthMax.Value = 100
	// Items is empty — pickHealingPotion returns "".
	res := fn(mob, &goals.Goal{Type: "survival"})
	if res.Command != "rest" {
		t.Errorf("command=%q, want rest", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}

// TestSurvival_CustomParams_Respected verifies that safe_threshold_pct and
// flee_threshold_pct params override the defaults. A mob at 70% HP with
// safe_threshold_pct=80 should not be treated as recovered.
func TestSurvival_CustomParams_Respected(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 70
	mob.Character.HealthMax.Value = 100
	// safe_threshold_pct = 80: 70% < 80% → not yet recovered.
	g := &goals.Goal{
		Type: "survival",
		Params: map[string]any{
			"safe_threshold_pct": 80,
			"flee_threshold_pct": 20,
		},
	}
	res := fn(mob, g)
	// HP 70% < safe 80%, not in combat, HP 70% >= flee 20% → rest.
	if res.Command != "rest" {
		t.Errorf("command=%q, want rest (70%% HP below custom safe_threshold 80%%)", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}
