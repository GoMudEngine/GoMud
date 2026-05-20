package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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

	target, err := actions.ResolveTargetActor(room, args[0])
	if err != nil {
		return true, nil
	}
	if target.IsPlayer() {

		toUser := target.(*actions.UserActor).User

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.IsHidden()

		if isSneaking {
			toUser.SendTextLegacy(fmt.Sprintf(`someone says to you, "<ansi fg="saytext-mob">%s</ansi>"`, rest))

			events.AddToQueue(events.Communication{
				SourceMobInstanceId: mob.InstanceId,
				TargetUserId:        toUser.UserId,
				CommType:            `say`,
				Name:                mob.Character.Name,
				Message:             rest,
			})

		} else {
			toUser.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to you, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
			room.SendTextVisualLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to <ansi fg="username">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toUser.Character.Name, rest), toUser.UserId)

			events.AddToQueue(events.Communication{
				SourceMobInstanceId: mob.InstanceId,
				CommType:            `say`,
				Name:                mob.Character.Name,
				Message:             rest,
			})
		}
	} else {

		toMob := target.(*actions.MobActor).Mob

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.IsHidden()

		if !isSneaking {
			room.SendTextVisualLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to <ansi fg="mobname">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toMob.Character.Name, rest))

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

	target, err := actions.ResolveTargetActor(room, args[0])
	if err != nil || !target.IsPlayer() {
		return true, nil
	}
	toUser := target.(*actions.UserActor).User

	rest = strings.TrimSpace(rest[len(args[0]):])
	isSneaking := mob.Character.IsHidden()

	if isSneaking {
		toUser.SendTextLegacy(fmt.Sprintf(`someone says to you, "<ansi fg="saytext-mob">%s</ansi>"`, rest))
	} else {
		toUser.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says to you, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
	}

	events.AddToQueue(events.Communication{
		SourceMobInstanceId: mob.InstanceId,
		TargetUserId:        toUser.UserId,
		CommType:            `say`,
		Name:                mob.Character.Name,
		Message:             rest,
	})

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

	target, err := actions.ResolveTargetActor(room, args[0])
	if err != nil {
		return true, nil
	}
	if target.IsPlayer() {

		toUser := target.(*actions.UserActor).User

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.IsHidden()

		if isSneaking {
			toUser.SendTextLegacy(fmt.Sprintf(`someone replies to you, "<ansi fg="saytext-mob">%s</ansi>"`, rest))
		} else {
			toUser.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> replies to you, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
			room.SendTextVisualLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> replies to <ansi fg="username">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toUser.Character.Name, rest), toUser.UserId)
		}
	} else {

		toMob := target.(*actions.MobActor).Mob

		rest = strings.TrimSpace(rest[len(args[0]):])
		isSneaking := mob.Character.IsHidden()

		if !isSneaking {
			room.SendTextVisualLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> replies to <ansi fg="mobname">%s</ansi>, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, toMob.Character.Name, rest))
		}
	}

	return true, nil
}
