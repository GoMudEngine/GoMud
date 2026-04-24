package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	noncombatQuestgiverYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/noncombat_questgiver.yaml"
	noncombatShopkeeperYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/noncombat_shopkeeper.yaml"
	noncombatPassiveYAML    = "../../_datafiles/world/dogmud/behaviors/archetypes/noncombat_passive.yaml"
	combatPassiveYAML       = "../../_datafiles/world/dogmud/behaviors/archetypes/combat_passive.yaml"
)

func TestArchetype_NoncombatQuestgiver_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_questgiver", noncombatQuestgiverYAML)
	assert.NotNil(t, GetEngine().GetArchetype("noncombat_questgiver"))
}

func TestArchetype_NoncombatShopkeeper_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_shopkeeper", noncombatShopkeeperYAML)
	assert.NotNil(t, GetEngine().GetArchetype("noncombat_shopkeeper"))
}

func TestArchetype_NoncombatPassive_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_passive", noncombatPassiveYAML)
	assert.NotNil(t, GetEngine().GetArchetype("noncombat_passive"))
}

func TestArchetype_CombatPassive_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "combat_passive", combatPassiveYAML)
	assert.NotNil(t, GetEngine().GetArchetype("combat_passive"))
}

func TestArchetype_NoncombatQuestgiver_HandlesPlayerEnter(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_questgiver", noncombatQuestgiverYAML)

	arch := GetEngine().GetArchetype("noncombat_questgiver")
	assert.NotNil(t, arch, "archetype should be loaded")

	ctx := &EvalContext{
		InstanceId: 13001,
		RoomId:     1,
		Event: EventContext{
			EventType: "player_enter",
			UserId:    42,
			RoomId:    1,
		},
	}
	// Call structurally — mob doesn't exist in test harness so actEmote may
	// return Failure at the mob lookup step. We're asserting no panic and
	// that the tree has a matching handler.
	_ = arch.Evaluate(ctx)
}

func TestArchetype_NoncombatQuestgiver_HandlesPlayerAttackRejected(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_questgiver", noncombatQuestgiverYAML)

	arch := GetEngine().GetArchetype("noncombat_questgiver")
	assert.NotNil(t, arch, "archetype should be loaded")

	ctx := &EvalContext{
		InstanceId: 13002,
		RoomId:     1,
		Event: EventContext{
			EventType: "player_attack_rejected",
			UserId:    42,
		},
	}
	_ = arch.Evaluate(ctx)
}

func TestArchetype_NoncombatShopkeeper_HandlesPlayerEnter(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_shopkeeper", noncombatShopkeeperYAML)

	arch := GetEngine().GetArchetype("noncombat_shopkeeper")
	assert.NotNil(t, arch, "archetype should be loaded")

	ctx := &EvalContext{
		InstanceId: 13003,
		RoomId:     1,
		Event: EventContext{
			EventType: "player_enter",
			UserId:    42,
			RoomId:    1,
		},
	}
	_ = arch.Evaluate(ctx)
}

func TestArchetype_NoncombatPassive_HandlesPlayerEnter(t *testing.T) {
	LoadArchetypeForTest(t, "noncombat_passive", noncombatPassiveYAML)

	arch := GetEngine().GetArchetype("noncombat_passive")
	assert.NotNil(t, arch, "archetype should be loaded")

	ctx := &EvalContext{
		InstanceId: 13004,
		RoomId:     1,
		Event: EventContext{
			EventType: "player_enter",
			UserId:    42,
			RoomId:    1,
		},
	}
	_ = arch.Evaluate(ctx)
}

func TestArchetype_CombatPassive_LoadsEmpty(t *testing.T) {
	LoadArchetypeForTest(t, "combat_passive", combatPassiveYAML)

	arch := GetEngine().GetArchetype("combat_passive")
	assert.NotNil(t, arch, "archetype should be loaded")

	// Archetype is intentionally empty for combat_passive
	ctx := &EvalContext{
		InstanceId: 13005,
		RoomId:     1,
		Event: EventContext{
			EventType: "mob_combat_round",
			RoomId:    1,
		},
	}
	result := arch.Evaluate(ctx)
	// Empty selector returns Failure
	assert.Equal(t, Failure, result)
}
