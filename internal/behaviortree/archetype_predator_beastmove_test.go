package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
)

const predatorBeastYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/predator.yaml"

// TestArchetype_Predator_FiresBeastMove is the end-to-end regression guard for
// the btree-delegation fix: a fanged beast on the predator archetype, evaluating
// a mob_combat_round, must reach the `try_special_move` node (the first child of
// the combat cascade) and issue a beast special move — i.e. return Success.
// Before the fix the cascade only offered bash/trip/kick/grapple and beast moves
// never fired.
func TestArchetype_Predator_FiresBeastMove(t *testing.T) {
	LoadArchetypeForTest(t, "predator", predatorBeastYAML)
	arch := GetEngine().GetArchetype("predator")
	if arch == nil {
		t.Fatal("predator archetype not loaded")
	}

	cleanupSp := species.SeedSpeciesForTest(map[int]*species.Species{
		9912: {SpeciesId: 9912, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
	})
	defer cleanupSp()

	// Standing prey at InstanceId 301.
	target := &mobs.Mob{MobId: 2, InstanceId: 301}
	target.Character.Name = "Prey"
	target.Character.HealthMax.Value = 100
	target.Character.Health = 100
	mobs.SetInstanceForTest(301, target)
	t.Cleanup(func() { mobs.SetInstanceForTest(301, nil) })

	// Fanged beast (predator profile), aggro on the prey, no cooldown.
	mob := newTestMob(t)
	mob.Character.SpeciesId = 9912
	mob.AIProfile = "predator"
	mob.SpecialMoveChance = 100
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 100
	mob.Character.Cooldowns = characters.Cooldowns{}
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{
		InstanceId: mob.InstanceId,
		Event:      EventContext{EventType: "mob_combat_round"},
	}
	if result := arch.Evaluate(ctx); result != Success {
		t.Errorf("predator mob_combat_round should fire a beast move via try_special_move (Success), got %v", result)
	}
}
