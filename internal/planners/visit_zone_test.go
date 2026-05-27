package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestVisitZone_Registered(t *testing.T) {
	if LookupPlanner("visit-zone") == nil {
		t.Fatalf("visit-zone not registered")
	}
}

func TestVisitZone_NoTargetZone_Failure(t *testing.T) {
	fn := LookupPlanner("visit-zone")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "visit-zone"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestVisitZone_AlreadyVisited_Success(t *testing.T) {
	fn := LookupPlanner("visit-zone")
	mob := &mobs.Mob{}
	mob.VisitedZones = map[string]bool{"stillwater": true}
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success", res.Status)
	}
}

func TestVisitZone_InTargetZone_Success(t *testing.T) {
	fn := LookupPlanner("visit-zone")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success", res.Status)
	}
}
