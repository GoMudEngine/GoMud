package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

func TestCondition_PackmateBelowHpRatio_MatchesWoundedPackmate(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9700,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	wounded := &mobs.Mob{
		InstanceId: 9701,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 20},
	}
	wounded.Character.HealthMax.Value = 100
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9700: self, 9701: wounded})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 9700}
	assert.Equal(t, Success, condPackmateBelowHpRatio(map[string]any{"threshold": 0.40}, ctx))
}

func TestCondition_PackmateBelowHpRatio_NoMatchWhenHealthy(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9702,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	healthy := &mobs.Mob{
		InstanceId: 9703,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 95},
	}
	healthy.Character.HealthMax.Value = 100
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9702: self, 9703: healthy})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 9702}
	assert.Equal(t, Failure, condPackmateBelowHpRatio(map[string]any{"threshold": 0.40}, ctx))
}

func TestCondition_PackmateBelowHpRatio_DefaultThreshold(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9704,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	wounded := &mobs.Mob{
		InstanceId: 9705,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 35},
	}
	wounded.Character.HealthMax.Value = 100 // 35% — below default 0.40
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9704: self, 9705: wounded})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 9704}
	// No threshold param — should use default 0.40.
	assert.Equal(t, Success, condPackmateBelowHpRatio(map[string]any{}, ctx))
}

func TestCondition_PackmateIsTanking_PackmateWithAggroReturnsSuccess(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9706,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	tanking := &mobs.Mob{
		InstanceId: 9707,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: 1,
			Health: 50,
			Aggro:  &characters.Aggro{UserId: 42, Type: characters.DefaultAttack},
		},
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9706: self, 9707: tanking})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 9706}
	assert.Equal(t, Success, condPackmateIsTanking(nil, ctx))
}

func TestCondition_PackmateIsTanking_NoPackmatesInCombatReturnsFailure(t *testing.T) {
	self := &mobs.Mob{
		InstanceId: 9708,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50},
	}
	idle := &mobs.Mob{
		InstanceId: 9709,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 1, Health: 50}, // no Aggro
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9708: self, 9709: idle})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 9708}
	assert.Equal(t, Failure, condPackmateIsTanking(nil, ctx))
}
