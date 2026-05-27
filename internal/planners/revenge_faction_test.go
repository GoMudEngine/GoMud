package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestRevengeFaction_Registered(t *testing.T) {
	if LookupPlanner("revenge-faction") == nil {
		t.Fatalf("revenge-faction not registered")
	}
}

func TestRevengeFaction_NoFactionParam_Failure(t *testing.T) {
	fn := LookupPlanner("revenge-faction")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "revenge-faction"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestRevengeFaction_NoMembersInZone_Wanders(t *testing.T) {
	fn := LookupPlanner("revenge-faction")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "revenge-faction", Params: map[string]any{
		"faction_id": "nonexistent-faction", "target_kill_count": 5,
	}}
	res := fn(mob, g)
	if res.Command != "wander" {
		t.Errorf("command=%q, want wander", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want Running", res.Status)
	}
}
