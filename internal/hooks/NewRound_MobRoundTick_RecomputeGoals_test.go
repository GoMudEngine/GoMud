package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// setGoalsTempDir points DOGMUD_GOALS_DIR_OVERRIDE at a per-test temp
// directory so that goals.Add / goals.Recompute can persist to disk
// without needing a real datafiles tree. Returns a cleanup func that
// removes the temp dir and restores the env var.
func setGoalsTempDir(t *testing.T) func() {
	t.Helper()
	dir, err := os.MkdirTemp("", "goals-hook-test-*")
	if err != nil {
		t.Fatalf("setGoalsTempDir: mkdirtemp: %v", err)
	}
	prev := os.Getenv("DOGMUD_GOALS_DIR_OVERRIDE")
	os.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", filepath.Join(dir, "goals"))
	return func() {
		os.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", prev)
		os.RemoveAll(dir)
	}
}

func TestTickMobRecomputeGoals_EmptyGoalList_NoOp(t *testing.T) {
	cleanupDir := setGoalsTempDir(t)
	defer cleanupDir()

	goals.ClearCache()
	mob := &mobs.Mob{MobId: mobs.MobId(99201)}
	mob.Character.Name = "tick_empty_test"
	// No goals seeded — should return early without panic or write.
	tickMobRecomputeGoals(mob, 1000)
	// Nothing to assert beyond "doesn't panic"; the cheap-path early
	// return is the whole behavior.
}

func TestTickMobRecomputeGoals_WithGoals_RecomputesCurrent(t *testing.T) {
	cleanupDir := setGoalsTempDir(t)
	defer cleanupDir()

	goals.ClearCache()
	mob := &mobs.Mob{MobId: mobs.MobId(99202)}
	mob.Character.Name = "tick_with_goals_test"
	if _, err := goals.Add(int(mob.MobId), "tick_with_goals_test",
		&goals.Goal{Type: "wealth", Priority: 50}); err != nil {
		t.Fatalf("seed Add: %v", err)
	}
	// Re-clear cache to force GoalsOf to reload from disk on the next
	// access, confirming the full disk-round-trip path works. Because
	// setGoalsTempDir pointed the override at a real temp dir, Add's
	// saveToDisk succeeded and the file is readable.
	goals.ClearCache()
	tickMobRecomputeGoals(mob, 1000)
	cur := goals.CurrentGoalOf(int(mob.MobId), "tick_with_goals_test")
	if cur == nil {
		t.Errorf("CurrentGoalOf nil after tick — hook did not Recompute")
	}
}
