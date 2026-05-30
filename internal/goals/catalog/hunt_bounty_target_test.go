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
