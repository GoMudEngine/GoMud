package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireMobDeathBehaviorTree subscribes to Life Alive→Dead transitions
// on mob characters and fires the "mob_die" behavior-tree event with
// the primary killer's UserId from DeadData.DamageMap.
//
// Migrated from internal/mobcommands/suicide.go lines 119-131 as
// part of chunk-2 Task 10.
func wireMobDeathBehaviorTree(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_death_behavior_tree",
		func(from, to life.State, r state.TransitionReason) {
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for mob characters.
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}

			d, _ := c.Life.DeadData()
			if len(d.DamageMap) == 0 {
				return
			}

			// Pick the first killer as the primary for the event.
			var killerUserId int
			for uid := range d.DamageMap {
				killerUserId = uid
				break
			}

			behaviortree.TryMobBehavior(m.InstanceId, behaviortree.EventContext{
				EventType: "mob_die",
				UserId:    killerUserId,
				RoomId:    m.Character.RoomId,
			})
		})
}

func init() {
	characters.OnCharacterCreated(wireMobDeathBehaviorTree)
}
