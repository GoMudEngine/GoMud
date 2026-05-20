package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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

		target, err := actions.ResolveTargetActor(room, rest)
		if err == actions.ErrTargetVanished {
			user.SendText(messaging.CategorySystem, "Zap target not found.")
			return true, nil
		}
		if err == nil {
			if !target.IsPlayer() {
				mob := target.(*actions.MobActor).Mob
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You zap <ansi fg="mobname">%s</ansi> with a %s!`, mob.Character.Name, boltOfLightning))
				room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="mobname">%s</ansi> with a %s!`, user.Character.Name, mob.Character.Name, boltOfLightning), user.UserId)

				mob.Character.Health = 1
				mob.Character.Conviction = 1

				return true, nil
			}

			u := target.(*actions.UserActor).User
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You zap <ansi fg="username">%s</ansi> with a %s!`, u.Character.Name, boltOfLightning))
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="username">%s</ansi> with a %s!`, user.Character.Name, u.Character.Name, boltOfLightning), user.UserId, u.UserId)
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps you with a %s!`, user.Character.Name, boltOfLightning))

			u.Character.Health = 1
			u.Character.Conviction = 1

			events.AddToQueue(events.CharacterVitalsChanged{UserId: u.UserId})

			return true, nil
		}
		// err == ErrTargetNotFound → fall through to no-arg path

	}

	if !user.Character.IsInCombat() {
		user.SendText(messaging.CategorySystem, "You are not in combat.")
		return true, nil
	}

	if user.Character.EngagedTarget().MobInstanceId > 0 {
		mob := mobs.GetInstance(user.Character.EngagedTarget().MobInstanceId)
		if mob == nil {
			user.SendText(messaging.CategorySystem, "Zap Mob not found.")
			return true, nil
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You zap <ansi fg="mobname">%s</ansi> with a %s!`, mob.Character.Name, boltOfLightning))
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="mobname">%s</ansi> with a %s!`, user.Character.Name, mob.Character.Name, boltOfLightning), user.UserId)

			mob.Character.Health = 1
			mob.Character.Conviction = 1
		}
	} else if user.Character.EngagedTarget().UserId > 0 {
		u := users.GetByUserId(user.Character.EngagedTarget().UserId)
		if u == nil {
			user.SendText(messaging.CategorySystem, "Zap User not found.")
			return true, nil
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You zap <ansi fg="username">%s</ansi> with a %s!`, u.Character.Name, boltOfLightning))
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> zaps <ansi fg="username">%s</ansi> with a %s!`, user.Character.Name, u.Character.Name, boltOfLightning), user.UserId)

			u.Character.Health = 1
			u.Character.Conviction = 1

			events.AddToQueue(events.CharacterVitalsChanged{UserId: u.UserId})
		}
	}

	return true, nil
}
