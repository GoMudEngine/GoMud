package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestFindMostWoundedPackmate_PicksLowestRatio(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9800,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	self.Character.HealthMax.Value = 100

	lightlyWounded := &mobs.Mob{
		InstanceId: 9801,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 60},
	}
	lightlyWounded.Character.HealthMax.Value = 100

	mostWounded := &mobs.Mob{
		InstanceId: 9802,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 10},
	}
	mostWounded.Character.HealthMax.Value = 100

	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		9800: self, 9801: lightlyWounded, 9802: mostWounded,
	})
	defer cleanup()

	got := findMostWoundedPackmate(self)
	assert.NotNil(t, got)
	assert.Equal(t, 9802, got.InstanceId, "should pick the packmate with the lowest HP ratio")
}

func TestFindMostWoundedPackmate_NoPackmatesReturnsNil(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9803,
		Routine:    "wraith_haunt",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9803: self})
	defer cleanup()

	got := findMostWoundedPackmate(self)
	assert.Nil(t, got)
}

func TestFindTankingPackmate_ReturnsPackmateWithAggro(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9804,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	idle := &mobs.Mob{
		InstanceId: 9805,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50}, // no Aggro
	}
	tanking := &mobs.Mob{
		InstanceId: 9806,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
			Aggro:  &characters.Aggro{UserId: 42, Type: characters.DefaultAttack},
		},
	}

	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		9804: self, 9805: idle, 9806: tanking,
	})
	defer cleanup()

	got := findTankingPackmate(self)
	assert.NotNil(t, got)
	assert.Equal(t, 9806, got.InstanceId)
}

func TestFindTankingPackmate_NoneInCombatReturnsNil(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9807,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	idle := &mobs.Mob{
		InstanceId: 9808,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50}, // no Aggro
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9807: self, 9808: idle})
	defer cleanup()

	got := findTankingPackmate(self)
	assert.Nil(t, got)
}
