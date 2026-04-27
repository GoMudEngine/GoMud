package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestAttack_PlayerAttackImmune_RebuffsAttack verifies that a mob with
// PlayerAttackImmune: true cannot be attacked by players — aggro must not be
// set on the user after attacking such a mob.
func TestAttack_PlayerAttackImmune_RebuffsAttack(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	// Seed a PlayerAttackImmune mob instance (NonCombatant: false so it can fight).
	immuneMob := &mobs.Mob{
		MobId:              5,
		InstanceId:         200,
		HomeRoomId:         1,
		Hostile:            true,
		NonCombatant:       false,
		PlayerAttackImmune: true,
		Character: characters.Character{
			Name:   "Caravan Guard",
			RoomId: 1,
			Health: 100,
			Buffs:  buffs.New(),
		},
	}
	immuneMob.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(200, immuneMob)
	defer mobs.SetInstanceForTest(200, nil)
	room.AddMob(200)
	defer room.RemoveMob(200)

	// Ensure no aggro before the attack.
	assert.Nil(t, user.Character.Aggro, "user should start with no aggro")

	handled, err := Attack("caravan guard", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)

	// The attack must be rebuffed — aggro must NOT be set.
	assert.Nil(t, user.Character.Aggro, "attack on PlayerAttackImmune mob must not set aggro")
}

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
