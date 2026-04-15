package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func SayTo(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) < 2 {
		return true, nil
	}

	playerId, mobInstanceId := room.FindByName(args[0])
	if playerId > 0 {

		toUser := users.GetByUserId(playerId)

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

		if isSneaking {
			toUser.SendText(fmt.Sprintf(`someone says to you, "<ansi fg="saytext-mob">%s</ansi>"`, rest))

			events.AddToQueue(events.Communication{
				SourceMobInstanceId: mob.InstanceId,
				TargetUserId:        toUser.UserId,
				CommType:            `say`,
				Name:                mob.Character.Name,
				Message:             rest,
			})

		} else {
			toUser.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to you, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to <ansi fg="username">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toUser.Character.Name, rest), toUser.UserId)

			events.AddToQueue(events.Communication{
				SourceMobInstanceId: mob.InstanceId,
				CommType:            `say`,
				Name:                mob.Character.Name,
				Message:             rest,
			})
		}
	} else if mobInstanceId > 0 {

		toMob := mobs.GetInstance(mobInstanceId)

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

		if !isSneaking {
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to <ansi fg="mobname">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toMob.Character.Name, rest))

			events.AddToQueue(events.Communication{
				SourceMobInstanceId: mob.InstanceId,
				CommType:            `say`,
				Name:                mob.Character.Name,
				Message:             rest,
			})

		}
	}

	return true, nil
}

func SayToOnly(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) < 2 {
		return true, nil
	}

	playerId, _ := room.FindByName(args[0])
	if playerId > 0 {

		toUser := users.GetByUserId(playerId)

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

		if isSneaking {
			toUser.SendText(fmt.Sprintf(`someone says to you, "<ansi fg="saytext-mob">%s</ansi>"`, rest))
		} else {
			toUser.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to you, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
		}

		events.AddToQueue(events.Communication{
			SourceMobInstanceId: mob.InstanceId,
			TargetUserId:        toUser.UserId,
			CommType:            `say`,
			Name:                mob.Character.Name,
			Message:             rest,
		})
	}

	return true, nil
}

func ReplyTo(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) < 2 {
		return true, nil
	}

	playerId, mobInstanceId := room.FindByName(args[0])
	if playerId > 0 {

		toUser := users.GetByUserId(playerId)

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

		if isSneaking {
			toUser.SendText(fmt.Sprintf(`someone replies to you, "<ansi fg="saytext-mob">%s</ansi>"`, rest))
		} else {
			toUser.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> replies to you, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> replies to <ansi fg="username">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toUser.Character.Name, rest), toUser.UserId)
		}
	} else if mobInstanceId > 0 {

		toMob := mobs.GetInstance(mobInstanceId)

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

		if !isSneaking {
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> replies to <ansi fg="mobname">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toMob.Character.Name, rest))
		}
	}

	return true, nil
}
