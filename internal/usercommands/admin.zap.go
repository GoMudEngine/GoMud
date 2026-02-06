package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* zap 				(All)
 */
func Zap(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	boltOfLightning := colorpatterns.ApplyColorPattern(`bolt of lightning`, `glowing`)

	if rest != `` {

		playerId, mobId := room.FindByName(rest)

		if mobId > 0 {

			mob := mobs.GetInstance(mobId)
			if mob == nil {
				user.SendText("Zap Mob not found.")
				return true, nil
			}

			user.SendText(fmt.Sprintf(`You zap <ansi fg="mobname">%s</ansi> with a %s!`, mob.Character.Name, boltOfLightning))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="mobname">%s</ansi> with a %s!`, user.Character.Name, mob.Character.Name, boltOfLightning), user.UserId)

			mob.Character.Health = 1
			mob.Character.Conviction = 1

			return true, nil
		}

		if playerId > 0 {
			if u := users.GetByUserId(playerId); u != nil {
				user.SendText(fmt.Sprintf(`You zap <ansi fg="username">%s</ansi> with a %s!`, u.Character.Name, boltOfLightning))
				room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="username">%s</ansi> with a %s!`, user.Character.Name, u.Character.Name, boltOfLightning), user.UserId, u.UserId)
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps you with a %s!`, user.Character.Name, boltOfLightning))

				u.Character.Health = 1
				u.Character.Conviction = 1

				events.AddToQueue(events.CharacterVitalsChanged{UserId: u.UserId})

				return true, nil
			}
		}

	}

	if user.Character.Aggro == nil {
		user.SendText("You are not in combat.")
		return true, nil
	}

	if user.Character.Aggro.MobInstanceId > 0 {
		mob := mobs.GetInstance(user.Character.Aggro.MobInstanceId)
		if mob == nil {
			user.SendText("Zap Mob not found.")
			return true, nil
		} else {
			user.SendText(fmt.Sprintf(`You zap <ansi fg="mobname">%s</ansi> with a %s!`, mob.Character.Name, boltOfLightning))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="mobname">%s</ansi> with a %s!`, user.Character.Name, mob.Character.Name, boltOfLightning), user.UserId)

			mob.Character.Health = 1
			mob.Character.Conviction = 1
		}
	} else if user.Character.Aggro.UserId > 0 {
		u := users.GetByUserId(user.Character.Aggro.UserId)
		if u == nil {
			user.SendText("Zap User not found.")
			return true, nil
		} else {
			user.SendText(fmt.Sprintf(`You zap <ansi fg="username">%s</ansi> with a %s!`, u.Character.Name, boltOfLightning))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="username">%s</ansi> with a %s!`, user.Character.Name, u.Character.Name, boltOfLightning), user.UserId)

			u.Character.Health = 1
			u.Character.Conviction = 1

			events.AddToQueue(events.CharacterVitalsChanged{UserId: u.UserId})
		}
	}

	return true, nil
}
