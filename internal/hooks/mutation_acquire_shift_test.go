package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// TestApplyAcquiredMutationTriggersShift: acquiring a pull-mutation
// re-archetypes an eligible mob. End-to-end over the extracted
// deterministic path (no RNG).
func TestApplyAcquiredMutationTriggersShift(t *testing.T) {
	cleanupDir := setGoalsTempDir(t)
	defer cleanupDir()
	goals.ClearCache()

	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brawn": {MutationId: "brawn", Rarity: 9, Clusters: []string{"colossus"}, Name: "Brawn", Visual: "muscles ripple"},
	})
	defer cleanup()

	mob := &mobs.Mob{MobId: 99950, InstanceId: 88801, BehaviorArchetype: "prey"}
	mob.Character.Name = "shiftprobe"
	mob.Character.RoomId = 0 // no room — the visual sends no-op safely
	cleanupMob := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{99950: mob},
		map[int]*mobs.Mob{88801: mob},
	)
	defer cleanupMob()

	applyAcquiredMutation(mob, "brawn")

	if mob.Character.Mutations["brawn"] != 1 {
		t.Fatal("mutation was not recorded")
	}
	if mob.BehaviorArchetype != "tank_taunter" {
		t.Fatalf("BehaviorArchetype = %q, want tank_taunter", mob.BehaviorArchetype)
	}
}
