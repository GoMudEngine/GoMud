// Round ticks for players
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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

		// Instance AFK: removed. The portal timer handles instance lifecycle.
		// Players in instances are not ejected for being idle — the instance
		// dies naturally when the portal expires and all players leave.

		if li-cutoffRound == 5 {
			user.SendText(messaging.CategoryWarning, `<ansi fg="203">WARNING:</ansi> <ansi fg="208">You are about to be kicked for inactivity!</ansi>`)
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
