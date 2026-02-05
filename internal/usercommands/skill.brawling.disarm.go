package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Brawling Skill
Level 4 - Attempt to disarm an opponent.
*/
func Disarm(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Brawling)

	// If they don't have a skill, act like it's not a valid command
	if skillLevel < 4 {
		return false, nil
	}

	if user.Character.Aggro == nil {
		user.SendText("Disarm is only used while in combat!")
		return true, nil
	}

	attackMobInstanceId := user.Character.Aggro.MobInstanceId
	attackPlayerId := user.Character.Aggro.UserId

	if attackMobInstanceId > 0 || attackPlayerId > 0 {
		if !user.Character.TryCooldown(skills.Brawling.String(`disarm`), "1 real minute") {
			user.SendText(fmt.Sprintf("You can try disarming again in %d rounds.", user.Character.GetCooldown(skills.Brawling.String(`disarm`))))
			return true, nil
		}
	}

	// Fire an event that a skill has been used
	events.AddToQueue(events.SkillUsed{user.UserId, skills.Brawling, `disarm`})

	if attackMobInstanceId > 0 {

		m := mobs.GetInstance(attackMobInstanceId)

		if m != nil {

			if m.Character.HasBuffFlag(buffs.PermaGear) {
				user.SendText(fmt.Sprintf(`Some force prevents you from disarming <ansi fg="mobname">%s</ansi>!`, m.Character.Name))
				return true, nil
			}

			if m.Character.Equipment.Weapon.ItemId == 0 {
				user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> has no weapon to disarm!`, m.Character.Name))
				return true, nil
			}

			chanceIn100 := (user.Character.Stats.Dexterity.ValueAdj + user.Character.Stats.Perception.ValueAdj) - (m.Character.Stats.Strength.ValueAdj + m.Character.Stats.Charisma.ValueAdj)
			if chanceIn100 < 0 {
				chanceIn100 = 0
			}
			chanceIn100 += 5
			roll := util.Rand(100)

			util.LogRoll(`Disarm`, roll, chanceIn100)

			if roll < chanceIn100 {

				user.SendText(
					fmt.Sprintf(`You disarm <ansi fg="mobname">%s</ansi>!`, m.Character.Name),
				)

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> disarms <ansi fg="mobname">%s</ansi>!`, user.Character.Name, m.Character.Name),
					user.UserId,
				)

				removedItem := m.Character.Equipment.Weapon
				m.Character.RemoveFromBody(removedItem)
				m.Character.StoreItem(removedItem)

			} else {
				user.SendText(
					fmt.Sprintf(`You try to disarm <ansi fg="mobname">%s</ansi> and fail!`, m.Character.Name),
				)

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to disarm <ansi fg="mobname">%s</ansi> and fails!`, user.Character.Name, m.Character.Name),
					user.UserId,
				)

			}
		}
	} else if attackPlayerId > 0 {

		u := users.GetByUserId(attackPlayerId)

		if u != nil {

			if u.Character.HasBuffFlag(buffs.PermaGear) {
				user.SendText(fmt.Sprintf(`Some force prevents you from disarming <ansi fg="username">%s</ansi>!`, u.Character.Name))
				return true, nil
			}

			chanceIn100 := (user.Character.Stats.Dexterity.ValueAdj + user.Character.Stats.Perception.ValueAdj) - (u.Character.Stats.Strength.ValueAdj + u.Character.Stats.Charisma.ValueAdj)
			if chanceIn100 < 0 {
				chanceIn100 = 0
			}
			chanceIn100 += 5
			roll := util.Rand(100)

			util.LogRoll(`Disarm`, roll, chanceIn100)

			if roll < chanceIn100 {

				user.SendText(
					fmt.Sprintf(`You disarm <ansi fg="username">%s</ansi>!`, u.Character.Name),
				)

				if atkUser := users.GetByUserId(attackPlayerId); atkUser != nil {
					atkUser.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> disarms you!`, user.Character.Name),
					)
				}

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> disarms <ansi fg="username">%s</ansi>!`, user.Character.Name, u.Character.Name),
					user.UserId,
					attackPlayerId,
				)

				removedItem := u.Character.Equipment.Weapon
				u.Character.RemoveFromBody(removedItem)
				u.Character.StoreItem(removedItem)

			} else {
				user.SendText(
					fmt.Sprintf(`You try to disarm <ansi fg="username">%s</ansi> and miss!`, u.Character.Name),
				)

				if atkUser := users.GetByUserId(attackPlayerId); atkUser != nil {
					atkUser.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to disarm you and misses!`, user.Character.Name),
					)
				}

				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to disarm <ansi fg="username">%s</ansi> and misses!`, user.Character.Name, u.Character.Name),
					user.UserId,
					attackPlayerId,
				)

			}
		}
	}

	return true, nil
}
