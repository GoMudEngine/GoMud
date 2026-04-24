package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestAttack_NonCombatantFiresEventOnce_ThenDedupes(t *testing.T) {
	originalTry := mobs.AttackRejectedTryMobBehavior
	var fireCount int
	mobs.AttackRejectedTryMobBehavior = func(instanceId int, ctx mobs.EventContext) bool {
		if ctx.EventType == "player_attack_rejected" {
			fireCount++
		}
		return true
	}
	defer func() { mobs.AttackRejectedTryMobBehavior = originalTry }()

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

	mobs.FireAttackRejected(mob, 42)
	mobs.FireAttackRejected(mob, 42)
	assert.Equal(t, 1, fireCount, "second call in same round should be deduped")

	util.SetRoundCountForTest(101)
	mobs.FireAttackRejected(mob, 42)
	assert.Equal(t, 2, fireCount, "call in a new round should fire again")
}
