package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionFaction_Registered(t *testing.T) {
	if LookupPlanner("protection-faction") == nil {
		t.Fatalf("protection-faction not registered")
	}
}

func TestProtectionFaction_NoFactionId_Failure(t *testing.T) {
	fn := LookupPlanner("protection-faction")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "protection-faction"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestProtectionFaction_NoMembersInZone_Failure(t *testing.T) {
	fn := LookupPlanner("protection-faction")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "nonexistent"}}
	res := fn(mob, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no members in zone)", res.Status)
	}
}

func TestProtectionFaction_NilMob_Failure(t *testing.T) {
	fn := LookupPlanner("protection-faction")
	res := fn(nil, &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "bandits"}})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (nil mob)", res.Status)
	}
}
