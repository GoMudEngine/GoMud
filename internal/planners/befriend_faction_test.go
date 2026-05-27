package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestBefriendFaction_Registered(t *testing.T) {
	if LookupPlanner("befriend-faction") == nil {
		t.Fatalf("befriend-faction not registered")
	}
}

func TestBefriendFaction_NoFactionId_Failure(t *testing.T) {
	fn := LookupPlanner("befriend-faction")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "befriend-faction"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestBefriendFaction_NoMembersInZone_Failure(t *testing.T) {
	fn := LookupPlanner("befriend-faction")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "nonexistent"}}
	res := fn(mob, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no members)", res.Status)
	}
}
