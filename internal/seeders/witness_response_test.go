package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestClassifyWitnessResponse(t *testing.T) {
	guard := &mobs.Mob{Groups: []string{"humanoid", "guard"}}
	if got := classifyWitnessResponse(guard); got != ResponseReportOnly {
		t.Fatalf("guard: want ResponseReportOnly, got %v", got)
	}

	civilian := &mobs.Mob{Groups: []string{"humanoid"}}
	civilian.Character.NonCombatant = true
	if got := classifyWitnessResponse(civilian); got != ResponseAlarm {
		t.Fatalf("noncombatant: want ResponseAlarm, got %v", got)
	}

	fighter := &mobs.Mob{Groups: []string{"humanoid"}}
	if got := classifyWitnessResponse(fighter); got != ResponseRevenge {
		t.Fatalf("combat mob: want ResponseRevenge, got %v", got)
	}

	if got := classifyWitnessResponse(nil); got != ResponseReportOnly {
		t.Fatalf("nil: want ResponseReportOnly (safe), got %v", got)
	}

	ncGuard := &mobs.Mob{Groups: []string{"guard"}}
	ncGuard.Character.NonCombatant = true
	if got := classifyWitnessResponse(ncGuard); got != ResponseReportOnly {
		t.Fatalf("noncombat guard: want ResponseReportOnly, got %v", got)
	}
}
