package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestRevengeMob_Registered(t *testing.T) {
	if LookupPlanner("revenge-mob") == nil {
		t.Fatalf("revenge-mob not registered")
	}
}

func TestRevengeMob_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("revenge-mob")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "revenge-mob"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestRevengeMob_TargetGone_Success(t *testing.T) {
	fn := LookupPlanner("revenge-mob")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "revenge-mob", Params: map[string]any{
		"target_kind": "mob", "target_id": 99999, // doesn't exist
	}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success (target nonexistent)", res.Status)
	}
}
