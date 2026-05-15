package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireAlivenessSubstrate fires the events.MobDeath event when a
// mob's Life transitions Alive→Dead. Existing aliveness-substrate
// subscribers (faction rep, opinion, crime, knowledge, bounty,
// quest notify, companion cleanup, pack flee) react downstream
// without any changes on their side.
//
// NOTE (Chunk-2): This observer is wired but dormant until
// Task 10 routes mob death paths through Life.TransitionToDead.
// The existing inline events.AddToQueue call in
// internal/mobcommands/suicide.go (lines 81-87) is the live
// path today; it will be removed when Task 10 lands.
func wireAlivenessSubstrate(c *characters.Character) {
	c.Life.Inner().AfterTransition("aliveness_substrate",
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
			mobRoom := rooms.LoadRoom(m.Character.RoomId)
			roomId := 0
			if mobRoom != nil {
				roomId = mobRoom.RoomId
			}

			events.AddToQueue(events.MobDeath{
				MobId:         int(m.MobId),
				InstanceId:    m.InstanceId,
				RoomId:        roomId,
				CharacterName: m.Character.Name,
				PlayerDamage:  m.Character.PlayerDamage,
			})
		})
}

func init() {
	characters.OnCharacterCreated(wireAlivenessSubstrate)
}
