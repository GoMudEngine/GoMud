package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestLookupPlanner_Unregistered_ReturnsNil(t *testing.T) {
	if fn := LookupPlanner("nonexistent-type"); fn != nil {
		t.Errorf("got non-nil for unregistered type")
	}
}

func TestRegisterPlanner_RoundTrip(t *testing.T) {
	called := false
	RegisterPlanner("test-roundtrip", func(mob *mobs.Mob, goal *goals.Goal) PlanResult {
		called = true
		return PlanResult{Command: "rest", Status: StatusRunning}
	})
	defer RegisterPlanner("test-roundtrip", nil)

	fn := LookupPlanner("test-roundtrip")
	if fn == nil {
		t.Fatalf("LookupPlanner returned nil after RegisterPlanner")
	}
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "test-roundtrip"})
	if !called {
		t.Errorf("registered fn not invoked")
	}
	if res.Command != "rest" {
		t.Errorf("Command: got %q, want rest", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("Status: got %v, want StatusRunning", res.Status)
	}
}

func TestRegisterPlanner_OverwritesPrevious(t *testing.T) {
	RegisterPlanner("test-overwrite", func(*mobs.Mob, *goals.Goal) PlanResult {
		return PlanResult{Command: "first"}
	})
	RegisterPlanner("test-overwrite", func(*mobs.Mob, *goals.Goal) PlanResult {
		return PlanResult{Command: "second"}
	})
	defer RegisterPlanner("test-overwrite", nil)

	fn := LookupPlanner("test-overwrite")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "test-overwrite"})
	if res.Command != "second" {
		t.Errorf("Command: got %q, want second (last-write-wins)", res.Command)
	}
}

func TestRegisterPlanner_NilUnregisters(t *testing.T) {
	RegisterPlanner("test-nil-unreg", func(*mobs.Mob, *goals.Goal) PlanResult {
		return PlanResult{Command: "x"}
	})
	RegisterPlanner("test-nil-unreg", nil)
	// Map still has the key but value is nil — LookupPlanner returns nil.
	if fn := LookupPlanner("test-nil-unreg"); fn != nil {
		t.Errorf("expected nil after Register(nil), got non-nil")
	}
}
