package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestAttack_NonCombatantFiresEventOnce_ThenDedupes(t *testing.T) {
	originalTry := attackTryMobBehavior
	var fireCount int
	attackTryMobBehavior = func(instanceId int, ctx behaviortree.EventContext) bool {
		if ctx.EventType == "player_attack_rejected" {
			fireCount++
		}
		return true
	}
	defer func() { attackTryMobBehavior = originalTry }()

	mob := &mobs.Mob{
		InstanceId: 12001,
		Character: characters.Character{
			RoomId: 1,
			Name:   "Test NPC",
		},
		NonCombatant: true,
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{12001: mob})
	defer cleanup()

	util.SetRoundCountForTest(100)
	defer util.ResetRoundCountForTest()

	fireAttackRejected(mob, 42)
	fireAttackRejected(mob, 42)
	assert.Equal(t, 1, fireCount, "second call in same round should be deduped")

	util.SetRoundCountForTest(101)
	fireAttackRejected(mob, 42)
	assert.Equal(t, 2, fireCount, "call in a new round should fire again")
}
