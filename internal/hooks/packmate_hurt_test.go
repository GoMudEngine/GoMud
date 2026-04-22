package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// TestDispatchPackmateHurt_FiresOnEachPackmate verifies the dispatcher
// calls the event hook with packmate_hurt on every mob that
// FindPackmatesInRoom returns, and no others.
func TestDispatchPackmateHurt_FiresOnEachPackmate(t *testing.T) {
	victim := &mobs.Mob{
		InstanceId: 1,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	packmateA := &mobs.Mob{
		InstanceId: 2,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	packmateB := &mobs.Mob{
		InstanceId: 3,
		Routine:    "bandit_camp_guard",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	unrelated := &mobs.Mob{
		InstanceId: 4,
		Routine:    "temple_service",
		Character:  characters.Character{RoomId: 100, Health: 50},
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		1: victim, 2: packmateA, 3: packmateB, 4: unrelated,
	})
	t.Cleanup(cleanup)

	type firing struct {
		instId int
		evt    string
		userId int
		mobId  int
	}
	recorded := []firing{}
	orig := dispatchEventFn
	t.Cleanup(func() { dispatchEventFn = orig })
	dispatchEventFn = func(mobInstanceId int, evt behaviortree.EventContext) bool {
		recorded = append(recorded, firing{
			instId: mobInstanceId,
			evt:    evt.EventType,
			userId: evt.UserId,
			mobId:  evt.MobId,
		})
		return true
	}

	dispatchPackmateHurt(victim, 42 /*attackerUserId*/, 99 /*attackerMobInstanceId*/)

	assert.Len(t, recorded, 2)
	ids := []int{recorded[0].instId, recorded[1].instId}
	assert.Contains(t, ids, 2)
	assert.Contains(t, ids, 3)
	assert.NotContains(t, ids, 4, "unrelated mob must not receive packmate_hurt")
	for _, r := range recorded {
		assert.Equal(t, "packmate_hurt", r.evt)
		assert.Equal(t, 42, r.userId)
		assert.Equal(t, 99, r.mobId)
	}
}
