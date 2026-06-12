package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const archerYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/archer.yaml"

// TestArchetype_Archer_FiresRangedShot is the end-to-end regression guard for
// the archer archetype: a mob with a LOADED ranged weapon and an aggro target
// one exit away, evaluating a mob_combat_round, must walk the real archer.yaml
// cascade (keep_distance fails because no one has closed into melee) and reach
// try_fire, issuing a directional ranged shot — i.e. return Success and enqueue
// `shoot <target> <dir>`.
func TestArchetype_Archer_FiresRangedShot(t *testing.T) {
	LoadArchetypeForTest(t, "archer", archerYAML)
	arch := GetEngine().GetArchetype("archer")
	if arch == nil {
		t.Fatal("archer archetype not loaded")
	}

	const here, there = 4500, 4501
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{
			"north": {RoomId: there},
		}},
		there: {RoomId: there, Zone: "test", Exits: map[string]exit.RoomExit{
			"south": {RoomId: here},
		}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	// Aggro target one exit (north) away.
	target := seedTargetMob(t, 4510, there)

	// Archer with a loaded bow, aggro on the cross-room target, in `here`.
	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.AIProfile = "archer"
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 100
	mob.Character.Equipment.Weapon = rangedBow(true)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
	queuedCmds(mob.InstanceId) // clear

	ctx := &EvalContext{
		InstanceId: mob.InstanceId,
		RoomId:     here,
		Event:      EventContext{EventType: "mob_combat_round"},
	}
	if result := arch.Evaluate(ctx); result != Success {
		t.Fatalf("archer mob_combat_round should fire a ranged shot via try_fire (Success), got %v", result)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "shoot Target north") {
		t.Errorf("expected 'shoot Target north' queued, got %v", cmds)
	}
}
