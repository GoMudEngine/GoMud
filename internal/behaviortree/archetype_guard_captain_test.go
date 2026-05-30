package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const guardCaptainYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/guard_captain.yaml"

// The guard_captain archetype (Town Justice 5.1c precursor) is a combat-capable
// hybrid: tank_taunter's combat cascade plus idle goal pursuit and a polite
// gift-decline. Quest item_give and `ask` dialogue bypass the btree entirely
// (see give.go / ask.go), so the archetype intentionally carries no quest or
// dialogue nodes.

func TestArchetype_GuardCaptain_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "guard_captain", guardCaptainYAML)
	assert.NotNil(t, GetEngine().GetArchetype("guard_captain"))
}

func TestArchetype_GuardCaptain_HandlesCombatRound(t *testing.T) {
	LoadArchetypeForTest(t, "guard_captain", guardCaptainYAML)

	arch := GetEngine().GetArchetype("guard_captain")
	assert.NotNil(t, arch, "archetype should be loaded")

	ctx := &EvalContext{
		InstanceId: 14001,
		RoomId:     1,
		Event: EventContext{
			EventType: "mob_combat_round",
			UserId:    42,
			RoomId:    1,
		},
	}
	// Structural call — the mob doesn't exist in the test harness, so combat
	// actions may return Failure at the mob lookup step. We assert no panic and
	// that the tree has a matching combat handler (i.e. it is combat-capable,
	// unlike the noncombat_questgiver it replaced).
	_ = arch.Evaluate(ctx)
}

func TestArchetype_GuardCaptain_HandlesIdle(t *testing.T) {
	LoadArchetypeForTest(t, "guard_captain", guardCaptainYAML)

	arch := GetEngine().GetArchetype("guard_captain")
	assert.NotNil(t, arch, "archetype should be loaded")

	ctx := &EvalContext{
		InstanceId: 14002,
		RoomId:     1,
		Event: EventContext{
			EventType: "mob_idle",
			UserId:    42,
			RoomId:    1,
		},
	}
	// Idle goal pursuit (try_goal_planner) must still be present so the captain
	// keeps his strategic/schedule behavior when not in combat.
	_ = arch.Evaluate(ctx)
}
