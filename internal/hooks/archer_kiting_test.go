package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const archerArchetypeYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/archer.yaml"

// TestHandleMobCombat_ArcherKitesAndFiresAfterAggroLoss is the end-to-end
// regression guard for the kiting-AI defect: an archer that has retreated one
// exit (so the round driver's ValidateAggro already ENDED its aggro) must NOT
// be skipped on the following round — its behavior tree must still fire and
// land a directional `shoot` in the input queue.
//
// What this test actually drives: the REAL unexported handleMobCombat round
// loop (NOT arch.Evaluate directly). It constructs the faithful post-retreat
// state — the archer sitting in room B with NIL aggro and a FRESH CombatMemory
// pointing at the player who is still in the adjacent room A — then runs one
// round and asserts the shoot command was enqueued. This exercises every gate
// the production round loop applies (health/non-combatant/zone-active/aggro-nil
// branch + the new archerReengageable exemption + TryMobBehavior archetype
// dispatch). The two-round melee→retreat prelude is covered separately by the
// behaviortree keep_distance unit test (which asserts CombatMemory refresh).
func TestHandleMobCombat_ArcherKitesAndFiresAfterAggroLoss(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// seedAllRegistries places players 1 (Aliceia) and 2 in room 1 of TestZone
	// (making the zone active) and wires room 1 <-> room 2 (north/south).
	behaviortree.LoadArchetypeForTest(t, "archer", archerArchetypeYAML)

	const roomA, roomB = 1, 2 // archer fled A -> B; target still stands in A

	weapon := items.Item{
		ItemId: 1,
		Loaded: true,
		Spec: &items.ItemSpec{
			ItemId:  1,
			Type:    items.Weapon,
			Subtype: items.Shooting,
			AmmoTag: "arrows",
		},
	}

	archer := &mobs.Mob{
		MobId:             3,
		InstanceId:        200,
		Zone:              "TestZone",
		HomeRoomId:        roomB,
		BehaviorArchetype: "archer",
		Character: characters.Character{
			Name:      "Bandit Archer",
			RoomId:    roomB,
			Health:    100,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	archer.Character.HealthMax.Value = 100
	archer.Character.Equipment.Weapon = weapon
	archer.Character.Position = position.NewMachine()
	// Post-retreat state: aggro already ended by ValidateAggro; only the
	// CombatMemory survives, pointing at player 1 last seen in room A.
	archer.Character.Aggro = nil
	archer.CombatMemory = mobs.SetCombatMemory(1, 0, roomA, 5)

	mobs.SetInstanceForTest(archer.InstanceId, archer)
	defer mobs.SetInstanceForTest(archer.InstanceId, nil)

	// Sanity: the target player really is one exit away from the archer.
	require.Equal(t, 1, users.GetByUserId(1).Character.RoomId, "target must be in room A")
	events.DrainQueuedInputsForTest(archer.InstanceId) // clear

	// Drive ONE real combat round.
	evt := events.NewRound{RoundNumber: 5}
	handleMobCombat(evt, map[int]bool{}, map[int]bool{})

	cmds := events.DrainQueuedInputsForTest(archer.InstanceId)
	assert.Contains(t, cmds, "shoot Aliceia south",
		"archer with nil aggro + fresh CombatMemory must fire a directional shot, got %v", cmds)
}
