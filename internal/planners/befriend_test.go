package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestBefriend_Registered(t *testing.T) {
	if LookupPlanner("befriend") == nil {
		t.Fatalf("befriend not registered")
	}
}

func TestBefriend_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("befriend")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "befriend"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestBefriend_TargetOutOfZone_Failure(t *testing.T) {
	fn := LookupPlanner("befriend")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "befriend", Params: map[string]any{
		"target_kind": "player", "target_id": 99999,
	}}
	res := fn(mob, g)
	// User doesn't exist → resolveTargetRoomId returns 0 → Failure.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}
