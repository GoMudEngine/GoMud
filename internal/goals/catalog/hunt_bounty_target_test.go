package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestHuntBountyTarget_Registered(t *testing.T) {
	meta, ok := goals.LookupGoalType("hunt_bounty_target")
	if !ok {
		t.Fatalf("hunt_bounty_target not registered")
	}
	if meta.Predicate == nil {
		t.Fatalf("hunt_bounty_target needs a predicate")
	}
	if meta.ContextScore == nil {
		t.Fatalf("hunt_bounty_target needs a context score")
	}
}

func TestHuntBountyTarget_PredicateAlwaysActive(t *testing.T) {
	// The predicate must always return false (goal is never self-satisfied).
	// Per-hunter lifecycle is owned by the dispatch manager; the goal must
	// not self-prune based on params (there are none).
	meta, ok := goals.LookupGoalType("hunt_bounty_target")
	if !ok {
		t.Fatalf("hunt_bounty_target not registered")
	}
	g := &goals.Goal{Type: "hunt_bounty_target"}
	if meta.Predicate(g, nil) {
		t.Fatalf("predicate should be always-active (return false), got true")
	}
}
