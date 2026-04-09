// Round ticks for players
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Handle mobs that are bored
//

func InactivePlayers(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	c := configs.GetConfig()

	maxIdleRounds := c.Timing.SecondsToRounds(int(c.Network.MaxIdleSeconds))
	if maxIdleRounds == 0 {
		return events.Continue
	}

	if evt.RoundNumber < uint64(maxIdleRounds) {
		return events.Continue
	}

	kickMods := bool(c.Network.TimeoutMods)

	cutoffRound := evt.RoundNumber - uint64(maxIdleRounds)

	for _, user := range users.GetAllActiveUsers() {

		// If don't kick mods and they aren't a regular user, skip
		if !kickMods && user.Role != users.RoleUser {
			continue
		}

		li := user.GetLastInputRound()

		if li == 0 {
			continue
		}

		// Check if player is AFK in an instanced zone — eject them to overworld
		if rooms.IsEphemeralRoomId(user.Character.RoomId) {
			afkRounds := uint64(c.Timing.SecondsToRounds(int(c.Network.AfkSeconds)))
			if afkRounds > 0 && evt.RoundNumber >= afkRounds && evt.RoundNumber-li >= afkRounds {
				if inst := rooms.GetInstanceRegistry().FindByRoomId(user.Character.RoomId); inst != nil {
					user.SendText(`<ansi fg="yellow">You've been idle too long. The unstable magic of this place expels you.</ansi>`)
					rooms.MoveToRoom(user.UserId, inst.OverworldRoomId)
					continue
				}
			}
		}

		if li-cutoffRound == 5 {
			user.SendText(`<ansi fg="203">WARNING:</ansi> <ansi fg="208">You are about to be kicked for inactivity!</ansi>`)
		}

		if li < cutoffRound {
			events.AddToQueue(events.System{
				Command:     `kick`,
				Data:        user.UserId,
				Description: `Inactivity`,
			})
		}

	}

	return events.Continue

}
