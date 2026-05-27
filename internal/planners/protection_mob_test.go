package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionMob_Registered(t *testing.T) {
	if LookupPlanner("protection-mob") == nil {
		t.Fatalf("protection-mob not registered")
	}
}

func TestProtectionMob_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("protection-mob")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "protection-mob"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestProtectionMob_TargetGone_Failure(t *testing.T) {
	fn := LookupPlanner("protection-mob")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "protection-mob", Params: map[string]any{
		"target_kind": "mob", "target_id": 99999,
	}}
	res := fn(mob, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (target nonexistent → 4.6 prunes)", res.Status)
	}
}
