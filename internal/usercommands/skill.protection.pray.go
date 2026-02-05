package usercommands

import (
	"errors"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
Protection Skill
Level 4 - Pray to gods for a blessing
*/
func Pray(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Protection)

	if skillLevel < 4 {
		user.SendText("You don't know how to pray.")
		return true, fmt.Errorf("you don't know how to pray")
	}

	if !user.Character.TryCooldown(skills.Protection.String(), "5 real minutes") {
		user.SendText(
			`You can only pray once every 5 minutes.`,
		)
		return true, errors.New(`you can only pray once every 5 minutes`)
	}

	prayPlayerId, prayMobId := 0, 0

	if rest == `` {
		prayPlayerId = user.UserId
	} else {
		prayPlayerId, prayMobId = room.FindByName(rest)
	}

	if prayPlayerId == 0 && prayMobId == 0 {
		user.SendText("Aid whom?")
		return true, nil
	}

	possibleBuffIds := []int{4, 11, 14, 16, 17, 18}
	totalBuffCount := 1 + int(float64(user.Character.Stats.Willpower.ValueAdj)/15) + util.Rand(2)

	if totalBuffCount > len(possibleBuffIds) {
		totalBuffCount = len(possibleBuffIds)
	}

	if prayPlayerId > 0 {

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Protection, `pray`})

		if prayPlayerId == user.UserId {
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> begins to pray.`, user.Character.Name), user.UserId)
		} else {
			targetUser := users.GetByUserId(prayPlayerId)
			if targetUser != nil {
				room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> puts his hand over <ansi fg="username">%s</ansi> and begins to pray.`, user.Character.Name, targetUser.Character.Name), user.UserId, targetUser.UserId)
				targetUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> puts his hand over you and begins to pray.`, user.Character.Name))
			}
		}

		for i := 0; i < totalBuffCount; i++ {
			randBuffIndex := util.Rand(len(possibleBuffIds))

			events.AddToQueue(events.Buff{
				UserId:        prayPlayerId,
				MobInstanceId: 0,
				BuffId:        possibleBuffIds[randBuffIndex],
				Source:        `skill`,
			})

			possibleBuffIds = append(possibleBuffIds[:randBuffIndex], possibleBuffIds[randBuffIndex+1:]...)
			room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> glows for a moment.`, user.Character.Name))
		}

	} else if prayMobId > 0 {

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Protection, `pray`})

		if mob := mobs.GetInstance(prayMobId); mob != nil {
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> puts his hand over <ansi fg="mobname">%s</ansi> and begins to pray.`, user.Character.Name, mob.Character.Name), user.UserId)

			for i := 0; i < totalBuffCount; i++ {
				randBuffIndex := util.Rand(len(possibleBuffIds))

				events.AddToQueue(events.Buff{
					UserId:        0,
					MobInstanceId: prayMobId,
					BuffId:        possibleBuffIds[randBuffIndex],
					Source:        `skill`,
				})

				possibleBuffIds = append(possibleBuffIds[:randBuffIndex], possibleBuffIds[randBuffIndex+1:]...)
				room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> glows for a moment.`, mob.Character.Name))
			}
		}

	} else {
		user.SendText("Pray for whom?")
	}

	return true, nil
}
