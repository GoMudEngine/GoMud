package usercommands

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// goalFixtureMobIdent returns the (mobId, namesimple) of the test fixture
// mob seeded by usercommands_test.go (Skeleton, id 1).
func goalFixtureMobIdent(t *testing.T) (int, string) {
	t.Helper()
	spec := mobs.GetMobSpec(mobs.MobId(1))
	if spec == nil {
		t.Fatal("test fixture mob id=1 missing — usercommands_test.go did not seed registry")
	}
	return 1, util.ConvertForFilename(spec.Character.Name)
}

// goalTestSetup arranges per-test isolation: temp goals dir, clean cache,
// clean registry, seeded usercommands registries. Returns the cleanup
// to defer.
func goalTestSetup(t *testing.T) func() {
	t.Helper()
	cleanup := seedAllRegistries()
	t.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", t.TempDir())
	goals.ClearCache()
	return cleanup
}

func TestGoalCmd_NoArgsRunsWithoutError(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	if _, err := Goal("", admin, room, 0); err != nil {
		t.Fatalf("Goal usage path returned error: %v", err)
	}
	// State assertion: usage print should not have mutated any goals.
	mobId, ns := goalFixtureMobIdent(t)
	if len(goals.GoalsOf(mobId, ns)) != 0 {
		t.Error("usage print should not have created goals")
	}
}

func TestGoalCmd_AddCreatesGoal(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50 reason=because", admin, room, 0); err != nil {
		t.Fatalf("Goal add: %v", err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(got))
	}
	if got[0].Type != "alpha" || got[0].Priority != 50 || got[0].Id != "g1" {
		t.Errorf("goal fields wrong: %+v", got[0])
	}
	if got[0].Params["reason"] != "because" {
		t.Errorf("param lost: %v", got[0].Params["reason"])
	}
}

func TestGoalCmd_AddBlockedDoesNotAppend(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// Lower priority same-type Add should be blocked.
	if _, err := Goal("add 1 alpha 30", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal (block path), got %d", len(got))
	}
	if got[0].Priority != 50 {
		t.Errorf("existing goal should be untouched, prio=%d", got[0].Priority)
	}
}

func TestGoalCmd_AddDisplacesLowerPriority(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 30", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("add 1 alpha 70", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal after displacement, got %d", len(got))
	}
	if got[0].Id != "g2" || got[0].Priority != 70 {
		t.Errorf("expected only g2 prio=70, got %+v", got[0])
	}
}

func TestGoalCmd_KeyValueParamTypes(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	cmd := "add 1 alpha 50 name=tova count=42 ratio=1.5 active=true"
	if _, err := Goal(cmd, admin, room, 0); err != nil {
		t.Fatal(err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(got))
	}
	p := got[0].Params
	if p["name"] != "tova" {
		t.Errorf("string param: %v (%T)", p["name"], p["name"])
	}
	if p["count"] != 42 {
		t.Errorf("int param: %v (%T)", p["count"], p["count"])
	}
	if p["ratio"] != 1.5 {
		t.Errorf("float param: %v (%T)", p["ratio"], p["ratio"])
	}
	if p["active"] != true {
		t.Errorf("bool param: %v (%T)", p["active"], p["active"])
	}
}

func TestGoalCmd_RemoveDeletesGoal(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("remove 1 g1", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if len(goals.GoalsOf(mobId, ns)) != 0 {
		t.Error("expected empty after remove")
	}
}

func TestGoalCmd_ClearWipesGoals(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("add 1 beta 30", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("clear 1", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if len(goals.GoalsOf(mobId, ns)) != 0 {
		t.Error("expected empty after clear")
	}
}

func TestGoalCmd_BadMobIdentRunsWithoutError(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	// The command should print "Unknown mob" and return (true, nil) —
	// no Go-level error. State should be untouched.
	if _, err := Goal("list nosuchmob_abcxyz", admin, room, 0); err != nil {
		t.Fatalf("expected nil error for bad ident, got: %v", err)
	}
}

func TestGoalCmd_RemoveMissingGoalNoError(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	// Underlying goals.Remove returns ErrGoalNotFound, but the admin
	// command swallows that into a user-facing message and returns nil.
	if _, err := Goal("remove 1 g99", admin, room, 0); err != nil {
		// Sanity: make sure the wrapped error class isn't being leaked.
		if errors.Is(err, goals.ErrGoalNotFound) {
			t.Fatal("admin command leaked ErrGoalNotFound to caller")
		}
		t.Fatalf("unexpected error: %v", err)
	}
}
