package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Watches the rounds go by
// Applies autohealing where appropriate
//

func AutoHeal(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	// Every 3 rounds. Else, pass it along.
	if evt.RoundNumber%3 != 0 {
		return events.Continue
	}

	deathRecoveryRoomId := int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom)

	onlineIds := users.GetOnlineUserIds()
	for _, userId := range onlineIds {
		user := users.GetByUserId(userId)

		if user.Character.RoomId == deathRecoveryRoomId {
			continue
		}

		inCombat := user.Character.Aggro != nil
		healthStart := user.Character.Health

		if user.Character.Health < 1 {

			if user.Character.Health <= -10 {

				user.Command(`suicide`) // suicide drops all money/items and transports to land of the dead.

			} else {
				user.Character.Health--
				user.SendText(`<ansi fg="red">you are bleeding out!</ansi>`)
				if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
					room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is <ansi fg="red">bleeding out</ansi>! Somebody needs to provide aid!`, user.Character.Name), user.UserId)
				}
			}

		}

		// Regeneration (only heal health if health > 0)
		if user.Character.Health > 0 {
			// Only heal health when NOT in combat
			if !inCombat {
				user.Character.Heal(
					user.Character.HealthPerRound(),
				)
			}
		}

		// Regenerate Stamina FIRST - slower during combat (always regenerate, even if health is low)
		var staminaRegen int
		if inCombat {
			// Combat: 1/4 normal regen rate (much slower)
			staminaRegen = user.Character.StaminaPerRound() / 4
			if staminaRegen < 1 {
				staminaRegen = 1 // Minimum 1 stamina per 3 rounds even in combat
			}
		} else {
			// Out of combat: full regen
			staminaRegen = user.Character.StaminaPerRound()
		}
		user.Character.Stamina += staminaRegen
		if user.Character.Stamina > user.Character.StaminaMax.Value {
			user.Character.Stamina = user.Character.StaminaMax.Value
		}

		// THEN check stamina exhaustion (after regen, so recovery is possible)
		if user.Character.Stamina < 1 {
			if user.Character.Stamina <= -10 {
				// Death from exhaustion
				user.Command(`suicide`)
			} else {
				// Exhausted - stamina continues to decrease
				user.Character.Stamina--
				user.SendText(`<ansi fg="yellow">you are exhausted!</ansi>`)
				if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
					room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is <ansi fg="yellow">exhausted</ansi>!`, user.Character.Name), user.UserId)
				}
			}
		}

		// Regenerate Conviction (not affected by combat state)
		user.Character.Conviction += user.Character.ConvictionPerRound()
		if user.Character.Conviction > user.Character.ConvictionMax.Value {
			user.Character.Conviction = user.Character.ConvictionMax.Value
		}

		// If it has changed, send an update
		if user.Character.Health-healthStart != 0 {

			// Trigger a redraw, but only if the users prompt has changed.
			events.AddToQueue(events.RedrawPrompt{UserId: user.UserId, OnlyIfChanged: true}, 100)

			events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

		}

	}

	return events.Continue
}
