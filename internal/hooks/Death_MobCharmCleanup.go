package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// wireMobCharmCleanup subscribes to Life Alive→Dead transitions on
// mob characters and:
//  1. Records the mob's instance ID in the recent-deaths registry
//     (mobs.TrackRecentDeath) for spawn-throttle checks.
//  2. Removes any active charm from the mob (RemoveCharm) and
//     reverse-tracks the owning player's charmed-mob list.
//
// Migrated from internal/mobcommands/suicide.go lines 35 and 40-44
// as part of chunk-2 Task 10.
func wireMobCharmCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_charm_cleanup",
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

			// Track in the recent-deaths ring so spawn throttles work.
			mobs.TrackRecentDeath(m.InstanceId)

			// Clean up charm: remove the charm from the mob and
			// reverse-track the player's charmed-mob list.
			if charmedUserId := m.Character.RemoveCharm(); charmedUserId > 0 {
				if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
					charmedUser.Character.TrackCharmed(m.InstanceId, false)
				}
			}
		})
}

func init() {
	characters.OnCharacterCreated(wireMobCharmCleanup)
}
